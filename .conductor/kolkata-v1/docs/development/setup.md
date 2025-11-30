# Development Guide

Complete guide for developing Skillrunner.

## Table of Contents

- [Quick Start](#quick-start)
- [Development Workflow](#development-workflow)
- [Testing](#testing)
- [Debugging](#debugging)
- [Performance](#performance)
- [Troubleshooting](#troubleshooting)

## Quick Start

### Initial Setup (< 2 minutes)

```bash
# Clone and setup
git clone https://github.com/jbctechsolutions/skillrunner
cd skillrunner
make dev-setup

# Build and test
make build
make test
```

### Verify Setup

```bash
# Check build works
./bin/sr --version

# Run example
./bin/sr run hello-orchestration "test"
```

## Development Workflow

### Daily Development Loop

```bash
# 1. Create feature branch
git checkout -b feature/my-feature

# 2. Make changes
# ... edit files ...

# 3. Quick check before commit
make check  # runs fmt + lint + test

# 4. Commit
git commit -m "feat: add my feature"

# 5. Push and create PR
git push origin feature/my-feature
```

### Makefile Commands

| Command | Purpose | Speed |
|---------|---------|-------|
| `make build` | Build binary | <1s |
| `make test` | Run all tests | ~30s |
| `make test-coverage` | Tests + coverage report | ~35s |
| `make fmt` | Format code | <1s |
| `make lint` | Run linter | ~10s |
| `make check` | fmt + lint + test | ~40s |
| `make bench` | Run benchmarks | varies |
| `make clean` | Remove build artifacts | <1s |

### Fast Iteration

For rapid development:

```bash
# Option 1: Auto-rebuild on changes (requires fswatch)
make watch

# Option 2: Quick manual loop
make build && ./bin/sr run hello-orchestration "test"

# Option 3: Test specific package
go test ./internal/router/... -v

# Option 4: Run single test
go test ./internal/router/... -run TestRouter_RouteTask -v
```

## Testing

### Test Organization

```
internal/
  router/
    router.go          # Implementation
    router_test.go     # Unit tests
    testdata/          # Test fixtures
```

### Running Tests

```bash
# All tests
make test

# Specific package
go test ./internal/context/... -v

# Single test
go test ./internal/router/... -run TestRouter_RouteTask -v

# With coverage
make test-coverage

# Short mode (skip slow tests)
go test ./... -short

# Race detection
go test ./... -race

# Benchmarks
make bench
```

### Writing Tests

Follow TDD approach:

```go
func TestRouter_RouteTask(t *testing.T) {
    // Arrange
    router := setupTestRouter(t)

    // Act
    result, err := router.RouteTask("skill", "task", "")

    // Assert
    assert.NoError(t, err)
    assert.NotNil(t, result)
}
```

See [Testing Guide](testing.md) for the complete guide.

### Coverage Goals

- Overall: >80%
- Core packages (router, orchestration, context): >90%
- New features: 100% of new code

Check coverage:

```bash
make test-coverage
# Opens coverage.html in browser
```

## Debugging

### VS Code Debugging

Launch configurations provided in `.vscode/launch.json`:

1. **Debug Skillrunner CLI** - Debug the main CLI
2. **Debug Current Test** - Debug selected test
3. **Debug All Tests** - Debug all tests in file
4. **Run Hello Orchestration** - Debug example workflow

### Command Line Debugging

```bash
# Run with delve
dlv debug ./cmd/skillrunner -- run hello-orchestration "test"

# Debug specific test
dlv test ./internal/router -- -test.run TestRouter_RouteTask
```

### Logging

Add debug output:

```go
import "log"

log.Printf("DEBUG: value=%v", value)
```

Run with verbose flags:

```bash
./bin/sr run hello-orchestration "test" --verbose
```

## Performance

### Build Time

- Clean build: <1s
- Incremental: <500ms

### Test Time

- Full suite: ~30s
- Single package: ~1-3s
- With race detection: ~45s

### Optimization Tips

```bash
# Cache dependencies
go mod download

# Parallel tests
go test ./... -parallel 8

# Skip slow tests during development
go test ./... -short

# Profile tests
go test -cpuprofile=cpu.prof -memprofile=mem.prof -bench=.
go tool pprof cpu.prof
```

## Code Quality

### Formatting

```bash
# Format all code
make fmt

# Check formatting (CI mode)
test -z $(go fmt ./...)
```

### Linting

Configuration in `.golangci.yml`:

```bash
# Run linter
make lint

# Run specific linter
golangci-lint run --enable=gocyclo

# Auto-fix issues
golangci-lint run --fix
```

### Common Issues

```bash
# Fix imports
goimports -w .

# Remove unused code
goimports -w -local github.com/jbctechsolutions/skillrunner .

# Check for common mistakes
go vet ./...
```

## Project Structure

```
skillrunner/
├── cmd/skillrunner/              # CLI entry point
│   ├── main.go              # Main function
│   ├── config_cmd.go        # Config commands
│   └── *_test.go            # CLI tests
├── internal/                 # Private packages
│   ├── router/              # Model routing
│   ├── context/             # Context management
│   ├── orchestration/       # Multi-phase execution
│   └── ...
├── docs/                     # Documentation
├── scripts/                  # Helper scripts
├── .vscode/                  # VS Code settings
├── .claude/                  # Claude commands
└── Makefile                  # Build automation
```

## Dependencies

Minimal dependency philosophy:

```go
// go.mod
require (
    github.com/spf13/cobra v1.10.1  // CLI framework
    gopkg.in/yaml.v3 v3.0.1         // YAML parsing
)
```

Adding new dependencies:

```bash
# Add dependency
go get github.com/example/package

# Update dependencies
make deps

# Verify no extra dependencies
go mod tidy
```

## Git Workflow

### Branching Strategy

- `main` - Stable releases
- `develop` - Development branch (if needed)
- `feature/*` - New features
- `fix/*` - Bug fixes
- `docs/*` - Documentation

### Commit Messages

Follow conventional commits:

```bash
git commit -m "feat: add semantic chunking"
git commit -m "fix: resolve context overflow"
git commit -m "docs: update quick start"
git commit -m "test: add router tests"
git commit -m "refactor: simplify chunker"
git commit -m "perf: optimize token counting"
```

### Pre-commit Checks

Install git hooks:

```bash
make dev-setup
```

This installs a pre-commit hook that runs:
1. Code formatting
2. Linting
3. Tests (short mode)

## Troubleshooting

### Build Issues

```bash
# Clear cache
go clean -cache -modcache -i -r

# Re-download dependencies
make deps

# Verify Go version
go version  # Should be 1.21+
```

### Test Failures

```bash
# Run single failing test
go test ./internal/router/... -run TestRouter_RouteTask -v

# Check for race conditions
go test ./... -race

# Verbose output
go test ./... -v
```

### Performance Issues

```bash
# Profile CPU
go test -cpuprofile=cpu.prof -bench=.
go tool pprof cpu.prof

# Profile memory
go test -memprofile=mem.prof -bench=.
go tool pprof mem.prof

# Check allocations
go test -benchmem -bench=.
```

### IDE Issues

```bash
# Regenerate Go module cache
go mod download

# Rebuild gopls
go install golang.org/x/tools/gopls@latest

# Clear VS Code Go cache
Cmd+Shift+P > "Go: Reset Go Cache"
```

## IDE Setup

### VS Code (Recommended)

Settings provided in `.vscode/`:

```bash
# Install recommended extensions
code --install-extension golang.go
code --install-extension redhat.vscode-yaml
```

Features:
- Auto-format on save
- Lint on save
- Test runner integration
- Debugging support
- Tasks for common operations

### GoLand / IntelliJ

1. Open project
2. Enable Go modules: `Settings > Go > Go Modules`
3. Install golangci-lint plugin
4. Configure file watchers for formatting

### Vim / Neovim

```vim
" Add to .vimrc
Plug 'fatih/vim-go'
let g:go_fmt_command = "goimports"
let g:go_metalinter_command = "golangci-lint"
```

## CI/CD

GitHub Actions workflows in `.github/workflows/`:

- `ci.yml` - Run on every push/PR
  - Test on multiple platforms
  - Check formatting
  - Run linter
  - Generate coverage

- `release.yml` - Run on tags
  - Build for all platforms
  - Create GitHub release
  - Upload binaries

Local CI simulation:

```bash
# Run what CI runs
make check
```

## Next Steps

- Read [CONTRIBUTING.md](../../CONTRIBUTING.md) for contribution guidelines
- Review [Testing Guide](testing.md) for testing best practices
- Check [ARCHITECTURE.md](../../ARCHITECTURE.md) for system design
- See [Quick Start Guide](../getting-started/quick-start.md) for usage examples

## Getting Help

- Documentation: `docs/` directory
- Issues: GitHub Issues
- Discussions: GitHub Discussions

---

**Happy developing!**
