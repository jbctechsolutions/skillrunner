# Contributing to Skillrunner

Thank you for your interest in contributing to Skillrunner! This guide will help you get started.

## Quick Start for Contributors

### Prerequisites

1. **Go 1.21+** - [Install Go](https://go.dev/doc/install)
2. **Ollama** - [Install Ollama](https://ollama.ai)
3. **Git** - For version control
4. **golangci-lint** - For code quality (optional but recommended)

```bash
# Install golangci-lint
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
```

### Development Setup (< 5 minutes)

```bash
# 1. Clone the repository
git clone https://github.com/jbctechsolutions/skillrunner
cd skillrunner

# 2. Install dependencies
make deps

# 3. Build the project
make build

# 4. Run tests to verify setup
make test

# 5. Start Ollama and pull required model
ollama serve &
ollama pull qwen2.5:14b
```

That's it! You're ready to contribute.

## Development Workflow

### 1. Create a Branch

```bash
git checkout -b feature/your-feature-name
# or
git checkout -b fix/issue-description
```

### 2. Make Your Changes

Follow our coding standards:

```bash
# Format code before committing
make fmt

# Run linter
make lint

# Run tests
make test

# Check coverage
make test-coverage
```

### 3. Write Tests

We follow Test-Driven Development (TDD):
- Write tests before implementation
- Aim for >80% code coverage
- Include unit tests for all new functions
- Add integration tests for new features

See [docs/TESTING_STRATEGY.md](docs/TESTING_STRATEGY.md) for details.

### 4. Commit Your Changes

We follow conventional commits:

```bash
git commit -m "feat: add new routing strategy"
git commit -m "fix: resolve context overflow issue"
git commit -m "docs: update quick start guide"
git commit -m "test: add coverage for chunker"
git commit -m "refactor: simplify router logic"
```

**Commit types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `test`: Test additions or fixes
- `refactor`: Code refactoring
- `perf`: Performance improvements
- `chore`: Build/tooling changes

### 5. Push and Create PR

```bash
# Push your branch
git push origin feature/your-feature-name

# Create PR via GitHub UI or gh CLI
gh pr create --title "feat: your feature" --body "Description"
```

## Code Quality Standards

### Formatting

All code must be formatted with `gofmt` and `goimports`:

```bash
make fmt
```

### Linting

Code must pass `golangci-lint`:

```bash
make lint
```

### Testing

All tests must pass with race detection:

```bash
make test
```

### Coverage

Maintain >80% test coverage:

```bash
make test-coverage
# Opens coverage.html in browser
```

## Project Structure

```
skillrunner/
├── cmd/skillrunner/              # CLI entry point (binary: sr)
├── internal/
│   ├── context/           # Context management
│   ├── orchestration/     # Multi-phase execution
│   ├── router/            # Model routing
│   ├── llm/               # LLM client
│   └── ...
├── docs/                  # Documentation
├── skills/                # Example skills
└── config/                # Configuration files
```

## Common Tasks

### Adding a New Feature

1. Create feature branch
2. Write tests first (TDD)
3. Implement feature
4. Update documentation
5. Run full test suite
6. Create PR

### Fixing a Bug

1. Create bug fix branch
2. Write test that reproduces bug
3. Fix the bug
4. Verify test passes
5. Add regression test if needed
6. Create PR

### Improving Documentation

1. Create docs branch
2. Make changes
3. Build and verify locally
4. Create PR

## Running Locally

### Quick Build and Run

```bash
make build
./bin/sr --help
```

### Run Specific Tests

```bash
# Test single package
go test ./internal/router/... -v

# Test specific function
go test ./internal/router/... -run TestRouter_RouteTask -v

# With coverage
go test ./internal/router/... -cover
```

### Debug Mode

```bash
# Run with verbose output
./bin/sr run hello-orchestration "test" --verbose

# Check config
./bin/sr config show
```

## Pull Request Guidelines

### PR Checklist

- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Comments added for complex logic
- [ ] Documentation updated
- [ ] Tests added/updated
- [ ] All tests pass locally
- [ ] No new warnings from linter
- [ ] `make fmt` run
- [ ] `make lint` passes
- [ ] `make test` passes

### PR Template

When creating a PR, fill out the template completely:
- Description of changes
- Type of change
- Testing completed
- Related issues

### Review Process

1. Automated checks must pass (CI/CD)
2. Code review by maintainer
3. Address feedback
4. Final approval and merge

## Code Style Guide

### Go Best Practices

1. **Error Handling**
   ```go
   // Good
   result, err := doSomething()
   if err != nil {
       return fmt.Errorf("failed to do something: %w", err)
   }

   // Bad
   result, _ := doSomething()
   ```

2. **Naming Conventions**
   ```go
   // Good
   type RouterConfig struct {}
   func NewRouter() *Router {}

   // Bad
   type router_config struct {}
   func newrouter() *Router {}
   ```

3. **Package Organization**
   - Keep packages focused
   - Avoid circular dependencies
   - Use internal/ for private packages

4. **Comments**
   ```go
   // Good - explains why, not what
   // Use batch processing to reduce memory allocation
   func processBatch() {}

   // Bad - states the obvious
   // Process batch
   func processBatch() {}
   ```

### Testing Best Practices

1. **Test Names**
   ```go
   // Good
   func TestRouter_RouteTask_WithValidInput_ReturnsSuccess(t *testing.T) {}

   // Bad
   func TestRouter1(t *testing.T) {}
   ```

2. **Table-Driven Tests**
   ```go
   func TestRouter_RouteTask(t *testing.T) {
       tests := []struct {
           name    string
           input   string
           want    string
           wantErr bool
       }{
           {"valid input", "test", "result", false},
           {"empty input", "", "", true},
       }

       for _, tt := range tests {
           t.Run(tt.name, func(t *testing.T) {
               got, err := RouteTask(tt.input)
               if tt.wantErr {
                   assert.Error(t, err)
                   return
               }
               assert.Equal(t, tt.want, got)
           })
       }
   }
   ```

## Getting Help

- **Documentation**: Check `docs/` directory
- **Issues**: Search existing [GitHub issues](https://github.com/jbctechsolutions/skillrunner/issues)
- **Discussions**: Join [GitHub Discussions](https://github.com/jbctechsolutions/skillrunner/discussions)
- **Chat**: Coming soon

## Release Process

Maintainers handle releases following [docs/RELEASE_GUIDE.md](docs/RELEASE_GUIDE.md).

## Code of Conduct

Be respectful, inclusive, and professional. We're all here to build something great together.

## License

By contributing, you agree that your contributions will be licensed under the project's MIT License.

---

**Thank you for contributing to Skillrunner!**

Questions? Open an issue or discussion.
