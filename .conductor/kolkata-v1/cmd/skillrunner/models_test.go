package main

import (
	"bytes"
	"context"
	"io"
	"os"
	"strings"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/models"
	"github.com/spf13/cobra"
)

// mockProvider is a test provider implementation
type mockProvider struct {
	name         string
	providerType models.ProviderType
	modelList    []models.ModelRef
	modelData    map[string]models.ModelMetadata
	available    bool
	shouldError  bool
}

func (m *mockProvider) Info() models.ProviderInfo {
	return models.ProviderInfo{
		Name: m.name,
		Type: m.providerType,
	}
}

func (m *mockProvider) Models(ctx context.Context) ([]models.ModelRef, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}
	return m.modelList, nil
}

func (m *mockProvider) SupportsModel(ctx context.Context, model string) (bool, error) {
	if m.shouldError {
		return false, context.DeadlineExceeded
	}
	for _, ref := range m.modelList {
		if strings.EqualFold(ref.Name, model) {
			return true, nil
		}
	}
	return false, nil
}

func (m *mockProvider) IsModelAvailable(ctx context.Context, model string) (bool, error) {
	if m.shouldError {
		return false, context.DeadlineExceeded
	}
	for _, ref := range m.modelList {
		if strings.EqualFold(ref.Name, model) {
			return m.available, nil
		}
	}
	return false, nil
}

func (m *mockProvider) ModelMetadata(ctx context.Context, model string) (models.ModelMetadata, error) {
	if m.shouldError {
		return models.ModelMetadata{}, context.DeadlineExceeded
	}
	for _, ref := range m.modelList {
		if strings.EqualFold(ref.Name, model) {
			if metadata, ok := m.modelData[strings.ToLower(model)]; ok {
				return metadata, nil
			}
			// Return default metadata if not in map
			return models.ModelMetadata{
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     ref.Description,
			}, nil
		}
	}
	return models.ModelMetadata{}, models.ErrModelNotFound
}

func (m *mockProvider) ResolveModel(ctx context.Context, model string) (*models.ResolvedModel, error) {
	if m.shouldError {
		return nil, context.DeadlineExceeded
	}
	for _, ref := range m.modelList {
		if strings.EqualFold(ref.Name, model) && m.available {
			metadata, err := m.ModelMetadata(ctx, model)
			if err != nil {
				return nil, err
			}
			return &models.ResolvedModel{
				Name:            model,
				Provider:        m.Info(),
				Route:           "mock://route",
				Tier:            metadata.Tier,
				CostPer1KTokens: metadata.CostPer1KTokens,
			}, nil
		}
	}
	return nil, models.ErrModelNotFound
}

func (m *mockProvider) CheckModelHealth(ctx context.Context, modelID string) (*models.HealthStatus, error) {
	if m.shouldError {
		return &models.HealthStatus{
			Healthy: false,
			Message: "Provider unavailable",
			Suggestions: []string{
				"Check provider connection",
				"Verify configuration",
			},
		}, nil
	}

	// Check if model exists
	for _, ref := range m.modelList {
		if strings.EqualFold(ref.Name, modelID) {
			if m.available {
				metadata, _ := m.ModelMetadata(ctx, modelID)
				return &models.HealthStatus{
					Healthy: true,
					Message: "Model is available",
					Details: map[string]interface{}{
						"description": metadata.Description,
						"tier":        metadata.Tier,
						"cost":        metadata.CostPer1KTokens,
					},
				}, nil
			}
			return &models.HealthStatus{
				Healthy:     false,
				Message:     "Model exists but is not available",
				Suggestions: []string{"Check provider status"},
			}, nil
		}
	}

	return &models.HealthStatus{
		Healthy: false,
		Message: "Model not found",
		Suggestions: []string{
			"Check model name spelling",
			"List available models",
		},
	}, nil
}

func TestModelsListCommand(t *testing.T) {
	// Save original stdout
	oldStdout := os.Stdout
	defer func() { os.Stdout = oldStdout }()

	// Create test orchestrator
	orchestrator := models.NewOrchestrator()

	// Add mock providers
	mockOllama := &mockProvider{
		name:         "ollama",
		providerType: models.ProviderTypeLocal,
		modelList: []models.ModelRef{
			{Name: "qwen2.5:14b", Description: "Qwen 2.5 14B"},
			{Name: "llama3.2:3b", Description: "Llama 3.2 3B"},
		},
		available: true,
		modelData: map[string]models.ModelMetadata{
			"qwen2.5:14b": {
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     "Qwen 2.5 14B",
			},
			"llama3.2:3b": {
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     "Llama 3.2 3B",
			},
		},
	}

	mockAnthropic := &mockProvider{
		name:         "anthropic",
		providerType: models.ProviderTypeCloud,
		modelList: []models.ModelRef{
			{Name: "claude-3-5-sonnet-20241022", Description: "Claude 3.5 Sonnet"},
		},
		available: true,
		modelData: map[string]models.ModelMetadata{
			"claude-3-5-sonnet-20241022": {
				Tier:            models.AgentTierPowerful,
				CostPer1KTokens: 0.015,
				Description:     "Claude 3.5 Sonnet",
			},
		},
	}

	orchestrator.RegisterProvider(mockOllama)
	orchestrator.RegisterProvider(mockAnthropic)

	// Test listing all models
	ctx := context.Background()
	allModels, err := orchestrator.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	if len(allModels) != 3 {
		t.Errorf("Expected 3 models, got %d", len(allModels))
	}

	// Verify model data
	foundOllama := false
	foundAnthropic := false
	for _, model := range allModels {
		if model.Provider.Name == "ollama" {
			foundOllama = true
			if model.Provider.Type != models.ProviderTypeLocal {
				t.Errorf("Expected ollama to be local provider")
			}
		}
		if model.Provider.Name == "anthropic" {
			foundAnthropic = true
			if model.Provider.Type != models.ProviderTypeCloud {
				t.Errorf("Expected anthropic to be cloud provider")
			}
		}
	}

	if !foundOllama {
		t.Error("Expected to find ollama provider")
	}
	if !foundAnthropic {
		t.Error("Expected to find anthropic provider")
	}
}

func TestModelsCheckCommand(t *testing.T) {
	orchestrator := models.NewOrchestrator()

	mockOllama := &mockProvider{
		name:         "ollama",
		providerType: models.ProviderTypeLocal,
		modelList: []models.ModelRef{
			{Name: "qwen2.5:14b", Description: "Qwen 2.5 14B"},
		},
		available: true,
		modelData: map[string]models.ModelMetadata{
			"qwen2.5:14b": {
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     "Qwen 2.5 14B",
			},
		},
	}

	orchestrator.RegisterProvider(mockOllama)

	// Test checking available model
	ctx := context.Background()
	available, err := orchestrator.IsAvailable(ctx, "qwen2.5:14b")
	if err != nil {
		t.Fatalf("IsAvailable failed: %v", err)
	}
	if !available {
		t.Error("Expected model to be available")
	}

	// Test checking unavailable model
	available, err = orchestrator.IsAvailable(ctx, "nonexistent:model")
	if err != nil {
		t.Fatalf("IsAvailable failed: %v", err)
	}
	if available {
		t.Error("Expected model to be unavailable")
	}
}

func TestModelsValidateCommand(t *testing.T) {
	orchestrator := models.NewOrchestrator()

	mockOllama := &mockProvider{
		name:         "ollama",
		providerType: models.ProviderTypeLocal,
		modelList: []models.ModelRef{
			{Name: "qwen2.5:14b", Description: "Qwen 2.5 14B"},
			{Name: "llama3.2:3b", Description: "Llama 3.2 3B"},
		},
		available: true,
		modelData: map[string]models.ModelMetadata{
			"qwen2.5:14b": {
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     "Qwen 2.5 14B",
			},
			"llama3.2:3b": {
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     "Llama 3.2 3B",
			},
		},
	}

	orchestrator.RegisterProvider(mockOllama)

	// Test validate with preferred models
	preferredModels := []string{
		"qwen2.5-coder:32b", // Not available
		"qwen2.5:14b",       // Available
		"llama3.2:3b",       // Not in supportedModel
	}

	ctx := context.Background()
	var availableCount int
	for _, modelName := range preferredModels {
		available, err := orchestrator.IsAvailable(ctx, modelName)
		if err != nil {
			continue
		}
		if available {
			availableCount++
		}
	}

	if availableCount == 0 {
		t.Error("Expected at least one available model")
	}
}

func TestModelsRefreshCommand(t *testing.T) {
	orchestrator := models.NewOrchestrator()

	mockOllama := &mockProvider{
		name:         "ollama",
		providerType: models.ProviderTypeLocal,
		modelList: []models.ModelRef{
			{Name: "qwen2.5:14b", Description: "Qwen 2.5 14B"},
		},
		available: true,
		modelData: map[string]models.ModelMetadata{
			"qwen2.5:14b": {
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     "Qwen 2.5 14B",
			},
		},
	}

	orchestrator.RegisterProvider(mockOllama)

	// First fetch
	ctx := context.Background()
	models1, err := orchestrator.ListModels(ctx)
	if err != nil {
		t.Fatalf("First ListModels failed: %v", err)
	}

	// Second fetch (should use cache or refresh)
	models2, err := orchestrator.ListModels(ctx)
	if err != nil {
		t.Fatalf("Second ListModels failed: %v", err)
	}

	if len(models1) != len(models2) {
		t.Errorf("Model counts differ: %d vs %d", len(models1), len(models2))
	}
}

func TestTableFormatting(t *testing.T) {
	// Create a buffer to capture output
	var buf bytes.Buffer
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Simulate table output
	orchestrator := models.NewOrchestrator()
	mockOllama := &mockProvider{
		name:         "ollama",
		providerType: models.ProviderTypeLocal,
		modelList: []models.ModelRef{
			{Name: "qwen2.5:14b", Description: "Qwen 2.5 14B"},
		},
		available: true,
		modelData: map[string]models.ModelMetadata{
			"qwen2.5:14b": {
				Tier:            models.AgentTierFast,
				CostPer1KTokens: 0,
				Description:     "Qwen 2.5 14B",
			},
		},
	}
	orchestrator.RegisterProvider(mockOllama)

	ctx := context.Background()
	allModels, err := orchestrator.ListModels(ctx)
	if err != nil {
		t.Fatalf("ListModels failed: %v", err)
	}

	// Close writer and restore stdout
	w.Close()
	_, _ = io.Copy(&buf, r)
	os.Stdout = oldStdout

	// Verify we got models
	if len(allModels) == 0 {
		t.Error("Expected at least one model")
	}
}

func TestCommandParsing(t *testing.T) {
	tests := []struct {
		name       string
		subcommand string
		args       []string
		flags      []string
		wantErr    bool
	}{
		{
			name:       "list command",
			subcommand: "list",
			args:       []string{},
			wantErr:    false,
		},
		{
			name:       "list with provider filter",
			subcommand: "list",
			args:       []string{},
			flags:      []string{"--provider=ollama"},
			wantErr:    false,
		},
		{
			name:       "check command with model",
			subcommand: "check",
			args:       []string{"ollama/qwen2.5:14b"},
			wantErr:    false,
		},
		{
			name:       "check command without model",
			subcommand: "check",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "validate command with skill",
			subcommand: "validate",
			args:       []string{"golang-pro"},
			wantErr:    false,
		},
		{
			name:       "validate command without skill",
			subcommand: "validate",
			args:       []string{},
			wantErr:    true,
		},
		{
			name:       "refresh command",
			subcommand: "refresh",
			args:       []string{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Get the appropriate subcommand
			var subcmd *cobra.Command
			for _, cmd := range modelsCmd.Commands() {
				if cmd.Name() == tt.subcommand {
					subcmd = cmd
					break
				}
			}
			if subcmd == nil {
				t.Fatalf("Subcommand %s not found", tt.subcommand)
			}

			// Test flag parsing if flags are provided
			if len(tt.flags) > 0 {
				err := subcmd.ParseFlags(tt.flags)
				if err != nil {
					t.Errorf("ParseFlags() unexpected error = %v", err)
					return
				}
			}

			// Test argument validation
			var err error
			if subcmd.Args != nil {
				err = subcmd.Args(subcmd, tt.args)
			}

			if (err != nil) != tt.wantErr {
				t.Errorf("Args validation error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestErrorHandling(t *testing.T) {
	orchestrator := models.NewOrchestrator()

	// Test with error-prone provider
	mockProvider := &mockProvider{
		name:         "failing-provider",
		providerType: models.ProviderTypeCloud,
		shouldError:  true,
	}

	orchestrator.RegisterProvider(mockProvider)

	ctx := context.Background()
	_, err := orchestrator.ListModels(ctx)
	if err == nil {
		t.Error("Expected error from failing provider")
	}
}

func TestProviderTypeDetection(t *testing.T) {
	tests := []struct {
		name         string
		providerName string
		modelName    string
		wantType     models.ProviderType
	}{
		{
			name:         "ollama local provider",
			providerName: "ollama",
			modelName:    "qwen2.5:14b",
			wantType:     models.ProviderTypeLocal,
		},
		{
			name:         "anthropic cloud provider",
			providerName: "anthropic",
			modelName:    "claude-3-5-sonnet-20241022",
			wantType:     models.ProviderTypeCloud,
		},
		{
			name:         "openai cloud provider",
			providerName: "openai",
			modelName:    "gpt-4-turbo",
			wantType:     models.ProviderTypeCloud,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orchestrator := models.NewOrchestrator()

			mock := &mockProvider{
				name:         tt.providerName,
				providerType: tt.wantType,
				modelList: []models.ModelRef{
					{Name: tt.modelName, Description: tt.modelName},
				},
				available: true,
				modelData: map[string]models.ModelMetadata{
					strings.ToLower(tt.modelName): {
						Tier:            models.AgentTierFast,
						CostPer1KTokens: 0,
					},
				},
			}

			orchestrator.RegisterProvider(mock)

			ctx := context.Background()
			allModels, err := orchestrator.ListModels(ctx)
			if err != nil {
				t.Fatalf("ListModels failed: %v", err)
			}

			found := false
			for _, model := range allModels {
				if model.Provider.Name == tt.providerName {
					found = true
					if model.Provider.Type != tt.wantType {
						t.Errorf("Expected provider type %v, got %v", tt.wantType, model.Provider.Type)
					}
				}
			}

			if !found {
				t.Errorf("Provider %s not found in results", tt.providerName)
			}
		})
	}
}
