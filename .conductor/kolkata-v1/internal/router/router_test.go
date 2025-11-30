package router

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/jbctechsolutions/skillrunner/internal/types"
)

// createTestConfig creates a temporary test config file
func createTestConfig(t *testing.T) string {
	t.Helper()

	content := `models:
  ollama/test-model:
    provider: ollama
    endpoint: http://localhost:11434
    context_window: 16384
    cost_per_1k_input_tokens: 0.0
    cost_per_1k_output_tokens: 0.0
    capabilities:
      - code_understanding
      - summarization
    task_types:
      - summarization
      - extraction
    performance:
      tokens_per_second: 50
      typical_latency_ms: 200

  anthropic/test-claude:
    provider: anthropic
    api_key_env: TEST_ANTHROPIC_KEY
    context_window: 200000
    cost_per_1k_input_tokens: 0.003
    cost_per_1k_output_tokens: 0.015
    capabilities:
      - verification
      - nuanced_reasoning
    task_types:
      - verification
      - review
    performance:
      tokens_per_second: 100
      typical_latency_ms: 1500

routing:
  prefer_local: true
  by_task_type:
    summarization:
      preferred:
        - ollama/test-model
      fallback:
        - anthropic/test-claude
    verification:
      preferred:
        - anthropic/test-claude
      fallback:
        - ollama/test-model
`

	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "models.yaml")
	if err := os.WriteFile(configPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test config: %v", err)
	}

	return configPath
}

func TestNewModelRouter(t *testing.T) {
	configPath := createTestConfig(t)

	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	if router == nil {
		t.Fatal("Router should not be nil")
	}

	if len(router.models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(router.models))
	}
}

func TestNewModelRouter_InvalidPath(t *testing.T) {
	_, err := NewModelRouter("/nonexistent/path.yaml")
	if err == nil {
		t.Error("Expected error for invalid config path")
	}
}

func TestSelectModel_LocalFirst(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	router.SetPreferLocal(true)

	// Pre-populate health cache so models are considered healthy
	// This avoids needing to actually connect to Ollama in tests
	router.healthCache["ollama/test-model"] = time.Now()
	router.healthCache["anthropic/test-claude"] = time.Now()

	ctx := context.Background()
	model, err := router.SelectModel(ctx, types.TaskTypeSummarization)
	if err != nil {
		t.Fatalf("Failed to select model: %v", err)
	}

	if model != "ollama/test-model" {
		t.Errorf("Expected ollama/test-model for local-first, got %s", model)
	}
}

func TestSelectModel_CloudPreferred(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Set API key so the Anthropic model is healthy
	os.Setenv("TEST_ANTHROPIC_KEY", "test-key")
	defer os.Unsetenv("TEST_ANTHROPIC_KEY")

	router.SetPreferLocal(false)

	ctx := context.Background()
	model, err := router.SelectModel(ctx, types.TaskTypeVerification)
	if err != nil {
		t.Fatalf("Failed to select model: %v", err)
	}

	if model != "anthropic/test-claude" {
		t.Errorf("Expected anthropic/test-claude for verification, got %s", model)
	}
}

func TestSelectModel_UnknownTaskType(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	_, err = router.SelectModel(ctx, types.TaskType("unknown"))
	if err == nil {
		t.Error("Expected error for unknown task type")
	}
}

func TestSelectModel_NoHealthyModels(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Force all models to be unhealthy by using a context that's already cancelled
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err = router.SelectModel(ctx, types.TaskTypeSummarization)
	if err == nil {
		t.Error("Expected error when no healthy models available")
	}
}

func TestEstimateCost_LocalModel(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	cost := router.EstimateCost("ollama/test-model", 1000, 1000)
	if cost != 0.0 {
		t.Errorf("Expected cost 0.0 for local model, got %f", cost)
	}
}

func TestEstimateCost_CloudModel(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Input: 1000 tokens * $0.003/1K = $0.003
	// Output: 1000 tokens * $0.015/1K = $0.015
	// Total: $0.018
	cost := router.EstimateCost("anthropic/test-claude", 1000, 1000)
	expected := 0.018
	if cost != expected {
		t.Errorf("Expected cost %f for cloud model, got %f", expected, cost)
	}
}

func TestEstimateCost_UnknownModel(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	cost := router.EstimateCost("unknown/model", 1000, 1000)
	if cost != 0.0 {
		t.Errorf("Expected cost 0.0 for unknown model, got %f", cost)
	}
}

func TestSetPreferLocal(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	router.SetPreferLocal(true)
	if !router.preferLocal {
		t.Error("Expected preferLocal to be true")
	}

	router.SetPreferLocal(false)
	if router.preferLocal {
		t.Error("Expected preferLocal to be false")
	}
}

func TestHealthChecking_Ollama(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()

	// This will likely fail since we don't have Ollama running in tests
	// but it tests the code path
	healthy := router.isHealthy(ctx, "ollama/test-model")

	// We don't assert the result since it depends on environment
	// Just verify it doesn't panic
	t.Logf("Ollama health check result: %v", healthy)
}

func TestHealthChecking_Anthropic(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()

	// Set a test API key
	os.Setenv("TEST_ANTHROPIC_KEY", "test-key")
	defer os.Unsetenv("TEST_ANTHROPIC_KEY")

	healthy := router.isHealthy(ctx, "anthropic/test-claude")

	// Should be healthy since API key is set
	if !healthy {
		t.Error("Expected Anthropic model to be healthy when API key is set")
	}
}

func TestHealthChecking_Cache(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()

	// Set API key
	os.Setenv("TEST_ANTHROPIC_KEY", "test-key")
	defer os.Unsetenv("TEST_ANTHROPIC_KEY")

	// First check
	router.isHealthy(ctx, "anthropic/test-claude")

	// Cache should be populated
	if len(router.healthCache) == 0 {
		t.Error("Expected health cache to be populated")
	}

	// Second check should use cache
	router.isHealthy(ctx, "anthropic/test-claude")
}

func TestModelInfo_LoadedCorrectly(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	// Check Ollama model
	ollamaModel, exists := router.models["ollama/test-model"]
	if !exists {
		t.Fatal("Ollama model not found in registry")
	}

	if ollamaModel.Provider != types.ModelProviderOllama {
		t.Errorf("Expected provider ollama, got %s", ollamaModel.Provider)
	}

	if ollamaModel.ContextWindow != 16384 {
		t.Errorf("Expected context window 16384, got %d", ollamaModel.ContextWindow)
	}

	// Check Anthropic model
	anthropicModel, exists := router.models["anthropic/test-claude"]
	if !exists {
		t.Fatal("Anthropic model not found in registry")
	}

	if anthropicModel.Provider != types.ModelProviderAnthropic {
		t.Errorf("Expected provider anthropic, got %s", anthropicModel.Provider)
	}

	if anthropicModel.CostPer1KInput != 0.003 {
		t.Errorf("Expected input cost 0.003, got %f", anthropicModel.CostPer1KInput)
	}

	if anthropicModel.CostPer1KOutput != 0.015 {
		t.Errorf("Expected output cost 0.015, got %f", anthropicModel.CostPer1KOutput)
	}
}

func TestRoutingConfig_LoadedCorrectly(t *testing.T) {
	configPath := createTestConfig(t)
	router, err := NewModelRouter(configPath)
	if err != nil {
		t.Fatalf("Failed to create router: %v", err)
	}

	if !router.routing.PreferLocal {
		t.Error("Expected PreferLocal to be true")
	}

	// Check summarization routing
	summaryRouting, exists := router.routing.ByTaskType[types.TaskTypeSummarization]
	if !exists {
		t.Fatal("Summarization routing not found")
	}

	if len(summaryRouting.Preferred) != 1 || summaryRouting.Preferred[0] != "ollama/test-model" {
		t.Error("Summarization preferred model not set correctly")
	}

	if len(summaryRouting.Fallback) != 1 || summaryRouting.Fallback[0] != "anthropic/test-claude" {
		t.Error("Summarization fallback model not set correctly")
	}
}

// Benchmark tests
func BenchmarkSelectModel(b *testing.B) {
	configPath := createTestConfig(&testing.T{})
	router, err := NewModelRouter(configPath)
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}

	ctx := context.Background()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = router.SelectModel(ctx, types.TaskTypeSummarization)
	}
}

func BenchmarkEstimateCost(b *testing.B) {
	configPath := createTestConfig(&testing.T{})
	router, err := NewModelRouter(configPath)
	if err != nil {
		b.Fatalf("Failed to create router: %v", err)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = router.EstimateCost("anthropic/test-claude", 1000, 1000)
	}
}
