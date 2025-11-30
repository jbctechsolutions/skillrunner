# Testing Strategy

## Overview

This document outlines the testing strategy for Skillrunner, following Test-Driven Development (TDD) principles.

## Testing Philosophy

1. **TDD First**: Write tests before implementation
2. **Fast Tests**: Unit tests should run in milliseconds
3. **Isolated Tests**: No external dependencies in unit tests
4. **Comprehensive Coverage**: Aim for >80% code coverage
5. **Realistic Mocks**: Mocks should behave like real services

## Test Structure

```
internal/
  router/
    router.go
    router_test.go
    litellm/
      client.go
      client_test.go
  config/
    config.go
    config_test.go
  docker/
    docker.go
    docker_test.go
```

## Test Categories

### 1. Unit Tests

**Purpose**: Test individual functions and methods in isolation.

**Requirements**:
- No external dependencies
- Fast execution (<10ms per test)
- Deterministic results
- Mock all external calls

**Example**:
```go
func TestRouter_RouteTask(t *testing.T) {
    // Arrange
    mockProvider := &MockProvider{
        responses: []*Response{
            {Content: "Generated content", TotalTokens: 100},
        },
    }
    router := NewRouter(mockProvider)

    // Act
    result, err := router.RouteTask("test-skill", "test task", "")

    // Assert
    assert.NoError(t, err)
    assert.Equal(t, "Generated content", result.Generation)
}
```

### 2. Integration Tests

**Purpose**: Test interactions between components.

**Requirements**:
- May use test containers (Docker)
- Mock external APIs
- Test real HTTP calls to local services
- Slower than unit tests (<1s per test)

**Example**:
```go
func TestLiteLLMClient_Integration(t *testing.T) {
    // Start test LiteLLM container
    container := startTestLiteLLM(t)
    defer container.Stop()

    client := NewClient(container.URL())
    resp, err := client.ChatCompletion(ChatCompletionRequest{
        Model: "ollama/test-model",
        Messages: []Message{{Role: "user", Content: "test"}},
    })

    assert.NoError(t, err)
    assert.NotNil(t, resp)
}
```

### 3. End-to-End Tests

**Purpose**: Test complete workflows.

**Requirements**:
- Use real Docker containers
- May require API keys (optional, skip if not available)
- Slow execution (<30s per test)
- Mark with build tag: `//go:build e2e`

**Example**:
```go
//go:build e2e

func TestE2E_RouteTask(t *testing.T) {
    // Start full Docker Compose stack
    compose := startDockerCompose(t)
    defer compose.Stop()

    // Wait for services
    waitForServices(t, compose)

    // Execute router
    router := NewRouter(...)
    result, err := router.RouteTask("test-skill", "test", "")

    assert.NoError(t, err)
    assert.NotEmpty(t, result.Generation)
}
```

## Mocking Strategy

### HTTP Client Mock

```go
type MockHTTPClient struct {
    responses map[string]*http.Response
    errors    map[string]error
}

func (m *MockHTTPClient) Do(req *http.Request) (*http.Response, error) {
    key := req.URL.String()
    if err, ok := m.errors[key]; ok {
        return nil, err
    }
    if resp, ok := m.responses[key]; ok {
        return resp, nil
    }
    return nil, fmt.Errorf("unexpected request: %s", key)
}
```

### Provider Mock

```go
type MockProvider struct {
    GenerateFunc func(ctx context.Context, model string, messages []Message) (*Response, error)
}

func (m *MockProvider) Generate(ctx context.Context, model string, messages []Message) (*Response, error) {
    if m.GenerateFunc != nil {
        return m.GenerateFunc(ctx, model, messages)
    }
    return &Response{Content: "mock response"}, nil
}
```

### Docker Mock

```go
type MockDockerManager struct {
    IsRunningFunc func() bool
    StartFunc     func() error
}

func (m *MockDockerManager) IsServiceRunning(name string) bool {
    if m.IsRunningFunc != nil {
        return m.IsRunningFunc()
    }
    return true
}
```

## Test Data

### Fixtures

Store test data in `testdata/` directory:

```
testdata/
  skills/
    test-skill.yaml
    invalid-skill.yaml
  responses/
    litellm-success.json
    litellm-error.json
```

### Loading Fixtures

```go
func loadFixture(t *testing.T, path string) []byte {
    data, err := os.ReadFile(filepath.Join("testdata", path))
    require.NoError(t, err)
    return data
}
```

## Test Utilities

### Assertions

Use `testify/assert` and `testify/require`:

```go
import (
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestSomething(t *testing.T) {
    result, err := doSomething()
    require.NoError(t, err)  // Stops test on error
    assert.Equal(t, expected, result)  // Continues on failure
}
```

### Test Helpers

```go
func setupTestRouter(t *testing.T) *Router {
    t.Helper()  // Marks as test helper
    mockProvider := &MockProvider{}
    return NewRouter(mockProvider)
}

func withTempConfig(t *testing.T, fn func(*config.Manager)) {
    t.Helper()
    tmpDir := t.TempDir()
    configPath := filepath.Join(tmpDir, "config.yaml")
    mgr, err := config.NewManager(configPath)
    require.NoError(t, err)
    fn(mgr)
}
```

## Coverage Goals

- **Unit Tests**: >90% coverage
- **Integration Tests**: >70% coverage
- **E2E Tests**: Critical paths only

## Running Tests

### All Tests
```bash
go test ./...
```

### With Coverage
```bash
go test -cover ./...
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Specific Package
```bash
go test ./internal/router/...
```

### E2E Tests Only
```bash
go test -tags=e2e ./...
```

### Verbose Output
```bash
go test -v ./...
```

### Race Detection
```bash
go test -race ./...
```

## CI/CD Integration

### GitHub Actions Example

```yaml
- name: Run tests
  run: |
    go test -v -race -coverprofile=coverage.out ./...

- name: Upload coverage
  uses: codecov/codecov-action@v3
  with:
    file: ./coverage.out
```

## Test Checklist

Before submitting code:

- [ ] All new functions have unit tests
- [ ] Error cases are tested
- [ ] Edge cases are tested
- [ ] Tests pass locally
- [ ] Tests pass in CI
- [ ] Coverage meets targets
- [ ] No race conditions (test with -race)
- [ ] Mocks are realistic
- [ ] Test names are descriptive

## TDD Workflow

1. **Red**: Write failing test
2. **Green**: Write minimal code to pass
3. **Refactor**: Improve code while keeping tests green
4. **Repeat**: Move to next test

### Example TDD Cycle

```go
// Step 1: Write failing test
func TestRouter_LoadSkill(t *testing.T) {
    router := NewRouter(nil)
    skill, err := router.LoadSkill("test-skill")
    assert.NoError(t, err)
    assert.Equal(t, "test-skill", skill.Name)
}

// Step 2: Implement minimal code
func (r *Router) LoadSkill(name string) (*Skill, error) {
    return &Skill{Name: name}, nil
}

// Step 3: Add more tests, refine implementation
func TestRouter_LoadSkill_NotFound(t *testing.T) {
    router := NewRouter(nil)
    _, err := router.LoadSkill("missing")
    assert.Error(t, err)
}
```

## Performance Testing

### Benchmark Tests

```go
func BenchmarkRouter_RouteTask(b *testing.B) {
    router := setupTestRouter(b)
    for i := 0; i < b.N; i++ {
        _, _ = router.RouteTask("test", "task", "")
    }
}
```

Run with:
```bash
go test -bench=. -benchmem
```

## Summary

- **TDD**: Write tests first
- **Isolation**: Mock external dependencies
- **Coverage**: Aim for >80%
- **Speed**: Fast unit tests, slower integration/E2E
- **Realistic**: Mocks should behave like real services
- **CI/CD**: All tests must pass
