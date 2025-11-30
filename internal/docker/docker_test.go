package docker

import (
	"os"
	"testing"
)

func TestNewServiceManager(t *testing.T) {
	// This test requires docker-compose.yml to exist
	// Skip if not in project root
	if _, err := os.Stat("docker-compose.yml"); os.IsNotExist(err) {
		t.Skip("docker-compose.yml not found, skipping test")
	}

	sm, err := NewServiceManager("", map[string]string{})
	if err != nil {
		t.Fatalf("NewServiceManager failed: %v", err)
	}

	if sm.composeFile == "" {
		t.Error("composeFile should be set")
	}
}

func TestIsDockerRunning(t *testing.T) {
	sm, err := NewServiceManager("", map[string]string{})
	if err != nil {
		t.Skip("docker-compose.yml not found, skipping test")
	}

	// Just check if the method runs without error
	_ = sm.IsDockerRunning()
}
