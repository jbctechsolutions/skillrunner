package docker

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// ServiceManager manages Docker Compose services for Skillrunner
type ServiceManager struct {
	composeFile string
	projectDir  string
	envVars     map[string]string // Environment variables for docker-compose
}

// NewServiceManager creates a new Docker service manager
func NewServiceManager(projectDir string, envVars map[string]string) (*ServiceManager, error) {
	if projectDir == "" {
		// Try to find project root by looking for docker-compose.yml
		wd, err := os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("get working directory: %w", err)
		}

		// Walk up directory tree to find docker-compose.yml
		dir := wd
		for {
			composePath := filepath.Join(dir, "docker-compose.yml")
			if _, err := os.Stat(composePath); err == nil {
				projectDir = dir
				break
			}

			parent := filepath.Dir(dir)
			if parent == dir {
				// Reached root
				return nil, fmt.Errorf("docker-compose.yml not found")
			}
			dir = parent
		}
	}

	composeFile := filepath.Join(projectDir, "docker-compose.yml")
	if _, err := os.Stat(composeFile); err != nil {
		return nil, fmt.Errorf("docker-compose.yml not found at %s", composeFile)
	}

	return &ServiceManager{
		composeFile: composeFile,
		projectDir:  projectDir,
		envVars:     envVars,
	}, nil
}

// IsDockerRunning checks if Docker daemon is running
func (sm *ServiceManager) IsDockerRunning() bool {
	cmd := exec.Command("docker", "info")
	cmd.Stdout = nil
	cmd.Stderr = nil
	return cmd.Run() == nil
}

// IsServiceRunning checks if a specific service is running
func (sm *ServiceManager) IsServiceRunning(serviceName string) bool {
	cmd := sm.dockerComposeCmd("ps", "--format", "json", serviceName)
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	// Check if output contains "Up" status
	return strings.Contains(string(output), `"State":"running"`) || strings.Contains(string(output), `"State":"Up"`)
}

// AreServicesRunning checks if both Ollama and LiteLLM are running
func (sm *ServiceManager) AreServicesRunning() (bool, []string) {
	var notRunning []string

	if !sm.IsServiceRunning("ollama") {
		notRunning = append(notRunning, "ollama")
	}
	if !sm.IsServiceRunning("litellm") {
		notRunning = append(notRunning, "litellm")
	}

	return len(notRunning) == 0, notRunning
}

// StartServices starts Docker Compose services
func (sm *ServiceManager) StartServices() error {
	if !sm.IsDockerRunning() {
		return fmt.Errorf("Docker is not running. Please start Docker and try again")
	}

	fmt.Fprintf(os.Stderr, "Starting Docker services (Ollama and LiteLLM)...\n")

	cmd := sm.dockerComposeCmd("up", "-d")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	// Wait a bit for services to start
	time.Sleep(3 * time.Second)

	// Check if services are now running
	running, notRunning := sm.AreServicesRunning()
	if !running {
		return fmt.Errorf("services started but some are not running: %v. Check logs with: docker-compose logs", notRunning)
	}

	fmt.Fprintf(os.Stderr, "✅ Services started successfully\n")
	return nil
}

// StopServices stops Docker Compose services
func (sm *ServiceManager) StopServices() error {
	cmd := sm.dockerComposeCmd("down")
	cmd.Stdout = os.Stderr
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// RestartServices restarts Docker Compose services
func (sm *ServiceManager) RestartServices() error {
	if err := sm.StopServices(); err != nil {
		return err
	}
	time.Sleep(2 * time.Second)
	return sm.StartServices()
}

// HealthCheck checks if LiteLLM is responding to health checks
func (sm *ServiceManager) HealthCheck(liteLLMURL string) error {
	if liteLLMURL == "" {
		liteLLMURL = "http://localhost:18432" // Default custom port
	}

	// Check if LiteLLM container is running
	if !sm.IsServiceRunning("litellm") {
		return fmt.Errorf("LiteLLM service is not running")
	}

	// Try to hit the health endpoint
	cmd := exec.Command("curl", "-f", "-s", liteLLMURL+"/health")
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("LiteLLM health check failed: %w", err)
	}

	return nil
}

// WaitForServices waits for services to be ready with timeout
func (sm *ServiceManager) WaitForServices(timeout time.Duration, liteLLMURL string) error {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for time.Now().Before(deadline) {
		running, _ := sm.AreServicesRunning()
		if running {
			// Also check health
			if err := sm.HealthCheck(liteLLMURL); err == nil {
				return nil
			}
		}

		select {
		case <-ticker.C:
			continue
		case <-time.After(time.Until(deadline)):
			return fmt.Errorf("timeout waiting for services to be ready")
		}
	}

	return fmt.Errorf("timeout waiting for services")
}

// dockerComposeCmd creates a docker-compose command
func (sm *ServiceManager) dockerComposeCmd(args ...string) *exec.Cmd {
	var cmd *exec.Cmd

	// Try 'docker compose' first (newer Docker CLI)
	if _, err := exec.LookPath("docker"); err == nil {
		// Check if 'docker compose' works
		testCmd := exec.Command("docker", "compose", "version")
		if testCmd.Run() == nil {
			cmd = exec.Command("docker", append([]string{"compose", "-f", sm.composeFile}, args...)...)
		}
	}

	// Fall back to 'docker-compose'
	if cmd == nil {
		cmd = exec.Command("docker-compose", append([]string{"-f", sm.composeFile}, args...)...)
	}

	cmd.Dir = sm.projectDir

	// Set environment variables
	if sm.envVars != nil {
		cmd.Env = os.Environ()
		for k, v := range sm.envVars {
			cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
		}
	}

	return cmd
}

// GetServiceLogs returns logs for a service
func (sm *ServiceManager) GetServiceLogs(serviceName string, lines int) (string, error) {
	cmd := sm.dockerComposeCmd("logs", "--tail", fmt.Sprintf("%d", lines), serviceName)
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get logs: %w", err)
	}
	return string(output), nil
}

// EnsureServicesRunning ensures services are running, starting them if needed
func (sm *ServiceManager) EnsureServicesRunning(autoStart bool, liteLLMURL string) error {
	running, notRunning := sm.AreServicesRunning()
	if running {
		// Check health
		if err := sm.HealthCheck(liteLLMURL); err == nil {
			return nil
		}
		// Services are running but not healthy, try restart
		fmt.Fprintf(os.Stderr, "Services are running but not healthy, restarting...\n")
		return sm.RestartServices()
	}

	if !autoStart {
		return fmt.Errorf("services are not running: %v. Start them with: docker-compose up -d", notRunning)
	}

	fmt.Fprintf(os.Stderr, "Services not running (%v), starting them...\n", notRunning)
	if err := sm.StartServices(); err != nil {
		return fmt.Errorf("failed to start services: %w", err)
	}

	// Wait for services to be ready
	fmt.Fprintf(os.Stderr, "Waiting for services to be ready...\n")
	if err := sm.WaitForServices(30*time.Second, ""); err != nil {
		return fmt.Errorf("services did not become ready: %w", err)
	}

	return nil
}

// GetComposeVersion returns the docker-compose command variant being used
func (sm *ServiceManager) GetComposeVersion() string {
	// Try 'docker compose' first
	cmd := exec.Command("docker", "compose", "version")
	if err := cmd.Run(); err == nil {
		return "docker compose"
	}
	return "docker-compose"
}

// Status returns the status of all services
func (sm *ServiceManager) Status() (map[string]bool, error) {
	// Check service status directly without parsing docker-compose output
	status := make(map[string]bool)
	services := []string{"ollama", "litellm"}

	for _, service := range services {
		status[service] = sm.IsServiceRunning(service)
	}

	return status, nil
}
