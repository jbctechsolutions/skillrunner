# Contributing to Skillrunner

Thank you for your interest in contributing to Skillrunner! This document provides guidelines and information for contributors.

## Code of Conduct

By participating in this project, you agree to maintain a respectful, inclusive, and harassment-free environment for everyone. We expect all contributors to:

- Use welcoming and inclusive language
- Respect differing viewpoints and experiences
- Accept constructive criticism gracefully
- Focus on what is best for the community and project
- Show empathy towards other community members

Unacceptable behavior includes harassment, trolling, personal attacks, or publishing others' private information without permission. Report any issues to the project maintainers.

## Getting Started

### Prerequisites

- **Go 1.23** or later
- **golangci-lint** for code linting
- **Make** for build automation
- **Git** for version control

### Development Setup

1. **Fork and clone the repository**

   ```bash
   git clone https://github.com/YOUR_USERNAME/skillrunner.git
   cd skillrunner
   ```

2. **Install dependencies**

   ```bash
   make deps
   ```

3. **Verify your setup**

   ```bash
   make check
   ```

   This runs formatting, vetting, linting, and tests.

4. **Build the binary**

   ```bash
   make build
   ```

### Project Structure

```
skillrunner/
├── cmd/skillrunner/      # CLI entry point
├── internal/
│   ├── domain/           # Core business logic
│   ├── application/      # Ports and application services
│   ├── adapters/         # Provider implementations
│   ├── infrastructure/   # Config, skills loading, utilities
│   └── presentation/     # CLI commands and output formatting
├── skills/               # Built-in skill definitions
├── configs/              # Configuration examples
└── docs/                 # User documentation
```

## Development Workflow

### Running Tests

```bash
# Run all tests with coverage
make test

# Generate HTML coverage report
make coverage
```

### Code Style

We enforce consistent code style through automated tools:

```bash
# Format code
make fmt

# Run linter
make lint

# Run all checks (fmt, vet, lint, test)
make check
```

#### Style Guidelines

- **Formatting**: All code must be formatted with `go fmt`
- **Linting**: Code must pass `golangci-lint` checks
- **Imports**: Use `goimports` for organizing imports
- **Error handling**: Always handle errors explicitly
- **Documentation**: Export types and functions should have doc comments

### Making Changes

1. **Create a feature branch**

   ```bash
   git checkout -b feature/your-feature-name
   ```

2. **Make your changes** with atomic, focused commits

3. **Run tests and linting**

   ```bash
   make check
   ```

4. **Push your branch**

   ```bash
   git push origin feature/your-feature-name
   ```

## Pull Request Process

### Before Submitting

1. **Ensure all tests pass**: Run `make check` locally
2. **Update documentation**: If your change affects user-facing behavior, update relevant docs
3. **Add tests**: Include tests for new functionality
4. **Keep changes focused**: Each PR should address a single concern

### PR Guidelines

- **Title**: Use a clear, descriptive title summarizing the change
- **Description**: Explain what changed and why
- **Link issues**: Reference related issues using `Fixes #123` or `Relates to #456`
- **Small PRs**: Prefer smaller, reviewable PRs over large ones

### Review Process

1. Submit your PR against the `main` branch
2. CI must pass (build, tests, lint, security scan)
3. At least one maintainer approval is required
4. Address reviewer feedback with additional commits
5. Maintainer will merge once approved

## Reporting Issues

### Bug Reports

Use the [bug report template](.github/ISSUE_TEMPLATE/bug_report.md) and include:

- Skillrunner version (`sr version`)
- Operating system and Go version
- Steps to reproduce the issue
- Expected vs actual behavior
- Relevant logs or error messages

### Feature Requests

Use the [feature request template](.github/ISSUE_TEMPLATE/feature_request.md) and include:

- Clear description of the proposed feature
- Use case and motivation
- Potential implementation approach (optional)

### Security Issues

For security vulnerabilities, **do not open a public issue**. Instead, email the maintainers directly with details about the vulnerability.

## Development Tips

### Useful Make Targets

| Target | Description |
|--------|-------------|
| `make build` | Build the skillrunner binary |
| `make test` | Run tests with coverage |
| `make lint` | Run golangci-lint |
| `make fmt` | Format Go code |
| `make vet` | Run go vet |
| `make check` | Run fmt, vet, lint, and test |
| `make coverage` | Generate HTML coverage report |
| `make deps` | Download and tidy dependencies |
| `make clean` | Remove build artifacts |

### Testing Provider Integrations

For testing with LLM providers:

- **Ollama**: Install locally and run with default settings
- **Cloud providers**: Set API keys in your local configuration

### Architecture Notes

Skillrunner follows hexagonal architecture (ports and adapters):

- **Domain**: Pure business logic with no external dependencies
- **Application**: Defines ports (interfaces) for external services
- **Adapters**: Implements ports for specific providers
- **Infrastructure**: Cross-cutting concerns (config, logging)
- **Presentation**: CLI interface and output formatting

### Experimental Features Pattern

When implementing experimental or preview features, use the `ExperimentalError` pattern to provide clear feedback to users:

```go
// ExperimentalError indicates a feature is experimental and may not work correctly.
type ExperimentalError struct {
    Feature string
    Message string
    Err     error
}

func (e *ExperimentalError) Error() string {
    if e.Err != nil {
        return fmt.Sprintf("[experimental: %s] %s: %v", e.Feature, e.Message, e.Err)
    }
    return fmt.Sprintf("[experimental: %s] %s", e.Feature, e.Message)
}

// IsExperimental returns true to identify this as an experimental feature error.
func (e *ExperimentalError) IsExperimental() bool {
    return true
}

func (e *ExperimentalError) Unwrap() error {
    return e.Err
}
```

This pattern allows:
- Clear identification of experimental features in error messages
- Programmatic detection of experimental errors via `IsExperimental()`
- Proper error wrapping for debugging

See `internal/adapters/opencode/errors.go` for the reference implementation.

## License

By contributing to Skillrunner, you agree that your contributions will be licensed under the [MIT License](LICENSE).

---

Thank you for contributing to Skillrunner!
