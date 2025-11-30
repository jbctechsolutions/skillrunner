package engine

import (
	"context"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/jbctechsolutions/skillrunner/internal/config"
	"github.com/jbctechsolutions/skillrunner/internal/models"
	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// testdataDir returns the absolute path to the testdata directory
func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "testdata")
}

// setupTestProviders registers test providers with the engine for testing
func setupTestProviders(eng *Skillrunner) {
	// Register a local (Ollama-like) provider - this will be preferred for auto-detect
	localProvider := models.NewStaticProvider(models.ProviderInfo{
		Name: "ollama",
		Type: models.ProviderTypeLocal,
	}, []models.StaticModel{
		{
			Name:            "local-model",
			Route:           "ollama/local-model",
			Description:     "Test local model",
			Available:       true,
			Tier:            models.AgentTierFast,
			CostPer1KTokens: 0,
		},
	})

	// Register OpenAI-like provider with gpt-4
	openaiProvider := models.NewStaticProvider(models.ProviderInfo{
		Name: "openai",
		Type: models.ProviderTypeCloud,
	}, []models.StaticModel{
		{
			Name:            "gpt-4",
			Route:           "openai/gpt-4",
			Description:     "Test GPT-4 model",
			Available:       true,
			Tier:            models.AgentTierPowerful,
			CostPer1KTokens: 0.03,
		},
	})

	// Register Anthropic-like provider with claude-3
	anthropicProvider := models.NewStaticProvider(models.ProviderInfo{
		Name: "anthropic",
		Type: models.ProviderTypeCloud,
	}, []models.StaticModel{
		{
			Name:            "claude-3",
			Route:           "anthropic/claude-3",
			Description:     "Test Claude 3 model",
			Available:       true,
			Tier:            models.AgentTierPowerful,
			CostPer1KTokens: 0.028,
		},
	})

	eng.RegisterTestProvider(localProvider)
	eng.RegisterTestProvider(openaiProvider)
	eng.RegisterTestProvider(anthropicProvider)
}

func TestNewSkillrunner(t *testing.T) {
	sr := NewSkillrunner("/test/workspace", models.ResolvePolicyAuto)

	if sr.workspacePath != "/test/workspace" {
		t.Errorf("workspacePath = %s; want /test/workspace", sr.workspacePath)
	}

	if sr.skillsCache == nil {
		t.Error("skillsCache should be initialized")
	}

	if sr.models == nil {
		t.Error("model orchestrator should be initialized")
	}
}

func TestNewSkillrunnerEmptyWorkspace(t *testing.T) {
	sr := NewSkillrunner("", models.ResolvePolicyAuto)

	if sr.workspacePath != "." {
		t.Errorf("workspacePath = %s; want .", sr.workspacePath)
	}
}

func TestRunTestSkill(t *testing.T) {
	sr := NewSkillrunner(testdataDir(), models.ResolvePolicyAuto)
	setupTestProviders(sr)

	envelope, err := sr.Run("hello-orchestration", "hello world", "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if envelope == nil {
		t.Fatal("Run returned nil envelope")
	}

	if envelope.Skill != "hello-orchestration" {
		t.Errorf("Skill = %s; want hello-orchestration", envelope.Skill)
	}

	if envelope.Request != "hello world" {
		t.Errorf("Request = %s; want hello world", envelope.Request)
	}

	if len(envelope.Steps) != 1 {
		t.Errorf("Steps count = %d; want 1", len(envelope.Steps))
	}

	step := envelope.Steps[0]
	if step.Intent != types.IntentPlan {
		t.Errorf("Step intent = %s; want plan", step.Intent)
	}

	// With auto-detect (no default model), should pick local-model (local provider preferred)
	if step.Model != "local-model" {
		t.Errorf("Step model = %s; want local-model (auto-detected local)", step.Model)
	}

	// Local models are fast tier with zero cost
	if step.Metadata["model_provider_tier"] != "fast" {
		t.Errorf("Expected model_provider_tier fast, got %v", step.Metadata["model_provider_tier"])
	}
	if cost, ok := step.Metadata["model_cost_per_1k_tokens"].(float64); !ok || cost != 0 {
		t.Errorf("Expected model cost 0, got %v", step.Metadata["model_cost_per_1k_tokens"])
	}
}

// TestRunBackendArchitect removed - backend-architect is no longer a built-in skill
// It should be imported from marketplace or created as an orchestrated skill

// TestRunWithModelOverride removed - test skill is no longer a built-in skill
// Model override functionality should be tested with imported or orchestrated skills

func TestRunInvalidSkill(t *testing.T) {
	sr := NewSkillrunner("/test", models.ResolvePolicyAuto)

	_, err := sr.Run("nonexistent", "test", "")
	if err == nil {
		t.Error("Expected error for nonexistent skill, got nil")
	}
}

func TestListSkills(t *testing.T) {
	sr := NewSkillrunner("/test", models.ResolvePolicyAuto)

	skills, err := sr.ListSkills()
	if err != nil {
		t.Fatalf("ListSkills failed: %v", err)
	}

	// No built-in skills anymore - skills must be imported or orchestrated
	// Just verify the function works and returns a list (may be empty)
	_ = skills // Use the variable to avoid unused variable warning
}

func TestGetStatus(t *testing.T) {
	// Use test engine to avoid needing live providers
	sr := NewSkillrunnerForTesting("/workspace", models.ResolvePolicyAuto)
	setupTestProviders(sr)

	status, err := sr.GetStatus()
	if err != nil {
		t.Fatalf("GetStatus failed: %v", err)
	}

	if status.Version != Version {
		t.Errorf("Version = %s; want %s", status.Version, Version)
	}

	if status.Workspace != "/workspace" {
		t.Errorf("Workspace = %s; want /workspace", status.Workspace)
	}

	if !status.Ready {
		t.Error("Status should be ready")
	}

	// No built-in skills anymore - skill count may be 0 or more depending on imports
	// Just verify the count is non-negative
	if status.SkillCount < 0 {
		t.Errorf("SkillCount = %d; want non-negative", status.SkillCount)
	}

	if len(status.ConfiguredModels) == 0 {
		t.Error("ConfiguredModels should not be empty")
	}
}

func TestLoadSkillCaching(t *testing.T) {
	sr := NewSkillrunner(testdataDir(), models.ResolvePolicyAuto)

	// Load skill first time
	skill1, err := sr.loadSkill("hello-orchestration")
	if err != nil {
		t.Fatalf("First load failed: %v", err)
	}

	// Load same skill again (should come from cache)
	skill2, err := sr.loadSkill("hello-orchestration")
	if err != nil {
		t.Fatalf("Second load failed: %v", err)
	}

	// Should be the same pointer (from cache)
	if skill1 != skill2 {
		t.Error("Skills should be the same instance from cache")
	}
}

func TestGenerateStepsDefaultModel(t *testing.T) {
	sr := NewSkillrunner(testdataDir(), models.ResolvePolicyAuto)
	setupTestProviders(sr)

	skillConfig := &types.SkillConfig{
		Name:         "test",
		DefaultModel: "gpt-4",
	}

	steps, err := sr.generateSteps(skillConfig, "test request", "")
	if err != nil {
		t.Fatalf("generateSteps returned error: %v", err)
	}

	if len(steps) != 1 {
		t.Errorf("Steps count = %d; want 1", len(steps))
	}

	if steps[0].Model != "gpt-4" {
		t.Errorf("Model = %s; want gpt-4", steps[0].Model)
	}

	if steps[0].Metadata["model_provider"] != "openai" {
		t.Errorf("Expected model_provider openai, got %v", steps[0].Metadata["model_provider"])
	}
	if steps[0].Metadata["model_provider_tier"] != "powerful" {
		t.Errorf("Expected model_provider_tier powerful, got %v", steps[0].Metadata["model_provider_tier"])
	}
	if cost, ok := steps[0].Metadata["model_cost_per_1k_tokens"].(float64); !ok || cost != 0.03 {
		t.Errorf("Expected model cost 0.03, got %v", steps[0].Metadata["model_cost_per_1k_tokens"])
	}
}

func TestGenerateStepsModelOverride(t *testing.T) {
	sr := NewSkillrunner(testdataDir(), models.ResolvePolicyAuto)
	setupTestProviders(sr)

	skillConfig := &types.SkillConfig{
		Name:         "test",
		DefaultModel: "gpt-4",
	}

	steps, err := sr.generateSteps(skillConfig, "test request", "claude-3")
	if err != nil {
		t.Fatalf("generateSteps returned error: %v", err)
	}

	if steps[0].Model != "claude-3" {
		t.Errorf("Model = %s; want claude-3", steps[0].Model)
	}

	if steps[0].Metadata["model_provider"] != "anthropic" {
		t.Errorf("Expected model_provider anthropic, got %v", steps[0].Metadata["model_provider"])
	}
	if steps[0].Metadata["model_provider_tier"] != "powerful" {
		t.Errorf("Expected model_provider_tier powerful, got %v", steps[0].Metadata["model_provider_tier"])
	}
	if cost, ok := steps[0].Metadata["model_cost_per_1k_tokens"].(float64); !ok || cost != 0.028 {
		t.Errorf("Expected model cost 0.028, got %v", steps[0].Metadata["model_cost_per_1k_tokens"])
	}
}

func TestEnvelopeMetadata(t *testing.T) {
	workspace := testdataDir()
	sr := NewSkillrunner(workspace, models.ResolvePolicyAuto)
	setupTestProviders(sr)

	envelope, err := sr.Run("hello-orchestration", "hello", "")
	if err != nil {
		t.Fatalf("Run failed: %v", err)
	}

	if envelope.Metadata["workspace"] != workspace {
		t.Errorf("Envelope should contain workspace metadata, got %v", envelope.Metadata["workspace"])
	}

	if envelope.Metadata["created_at"] == nil {
		t.Error("Envelope should contain created_at metadata")
	}
}

// TestContextItemsInBackendArchitect removed - backend-architect is no longer a built-in skill
func TestContextItemsInBackendArchitect(t *testing.T) {
	t.Skip("backend-architect is no longer a built-in skill - test removed")
	// This test was for the hardcoded backend-architect skill which has been removed.
	// Context items should be tested with imported or orchestrated skills instead.
}

func TestTierToString(t *testing.T) {
	tests := []struct {
		name     string
		tier     models.AgentTier
		expected string
	}{
		{"Fast tier", models.AgentTierFast, "fast"},
		{"Balanced tier", models.AgentTierBalanced, "balanced"},
		{"Powerful tier", models.AgentTierPowerful, "powerful"},
		{"Unknown tier", models.AgentTier(999), "unknown"},
		{"Zero tier", models.AgentTier(0), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tierToString(tt.tier)
			if result != tt.expected {
				t.Errorf("tierToString(%v) = %q, want %q", tt.tier, result, tt.expected)
			}
		})
	}
}

func TestRegisterDefaultProviders(t *testing.T) {
	orchestrator := models.NewOrchestrator()

	// Test with empty config - should enable Ollama by default
	cfg := &config.Config{}
	registerDefaultProviders(orchestrator, cfg)

	ctx := context.Background()
	allModels, err := orchestrator.ListModels(ctx)
	if err != nil {
		// In CI environments without Ollama, this will fail. That's expected.
		t.Skipf("Skipping test: Ollama not available (%v)", err)
	}

	// With default config, we should have at least Ollama provider
	// The actual models will depend on what's running on localhost:11434
	// So we just check that the function doesn't error
	t.Logf("Found %d models from default providers", len(allModels))
}

func TestRegisterDefaultProvidersWithConfig(t *testing.T) {
	orchestrator := models.NewOrchestrator()

	// Test with explicit provider config
	cfg := &config.Config{
		Providers: &config.Providers{
			Ollama: &config.OllamaConfig{
				URL:     "http://localhost:11434",
				Enabled: true,
			},
		},
	}
	registerDefaultProviders(orchestrator, cfg)

	ctx := context.Background()
	allModels, err := orchestrator.ListModels(ctx)
	if err != nil {
		// In CI environments without Ollama, this will fail. That's expected.
		t.Skipf("Skipping test: Ollama not available (%v)", err)
	}

	// Should have registered the Ollama provider
	t.Logf("Found %d models with explicit Ollama config", len(allModels))
}

func TestNewSkillrunnerWithDifferentPolicies(t *testing.T) {
	policies := []models.ResolvePolicy{
		models.ResolvePolicyAuto,
		models.ResolvePolicyLocalFirst,
		models.ResolvePolicyPerformanceFirst,
		models.ResolvePolicyCostOptimized,
	}
	for _, policy := range policies {
		t.Run(string(policy), func(t *testing.T) {
			sr := NewSkillrunner(".", policy)
			if sr == nil {
				t.Fatal("expected non-nil Skillrunner")
			}
			if sr.workspacePath != "." {
				t.Errorf("expected workspace '.', got %q", sr.workspacePath)
			}
		})
	}
}
