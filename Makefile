# Skillrunner Makefile
# Build and development automation
#
# ============================================================================
# LDFLAGS for Version Information
# ============================================================================
# The following ldflags inject version info at build time into the binary:
#
#   github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands.Version
#   github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands.GitCommit
#   github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands.BuildDate
#
# Example manual build with version injection:
#   go build -ldflags "-X github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands.Version=1.0.0 \
#                      -X github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands.GitCommit=$(git rev-parse HEAD) \
#                      -X github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
#            -o skillrunner ./cmd/skillrunner
#
# For production releases, use 'make release' which runs goreleaser with
# proper ldflags injection automatically.
# ============================================================================

# Build variables
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
GIT_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")
BUILD_DATE ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS_PKG = github.com/jbctechsolutions/skillrunner/internal/presentation/cli/commands
LDFLAGS = -X $(LDFLAGS_PKG).Version=$(VERSION) \
          -X $(LDFLAGS_PKG).GitCommit=$(GIT_COMMIT) \
          -X $(LDFLAGS_PKG).BuildDate=$(BUILD_DATE)

.PHONY: all build build-release test lint clean help release release-snapshot test-ci test-release test-workflows

# Default target
all: clean lint test build

# Build the skillrunner binary (development build)
build:
	go build -o skillrunner ./cmd/skillrunner

# Build the skillrunner binary with version info injected via ldflags
# Use this for local release-style builds with proper versioning
build-release:
	go build -ldflags "$(LDFLAGS)" -o skillrunner ./cmd/skillrunner

# Run goreleaser to create a full release
# Requires GITHUB_TOKEN environment variable for publishing
# Creates binaries, checksums, and publishes to GitHub Releases
release:
	@if [ -z "$(GITHUB_TOKEN)" ]; then \
		echo "Error: GITHUB_TOKEN is required for release"; \
		echo "Set it with: export GITHUB_TOKEN=your_token"; \
		exit 1; \
	fi
	goreleaser release --clean

# Run goreleaser in snapshot mode (no publish, for testing)
# Useful for testing the release process locally without publishing
release-snapshot:
	goreleaser release --snapshot --clean

# Run tests with verbose output and coverage
test:
	go test -v -cover ./...

# Run linter
lint:
	golangci-lint run ./...

# Clean build artifacts
clean:
	rm -f skillrunner coverage.out

# Generate coverage report
coverage:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Install development dependencies
deps:
	go mod download
	go mod tidy

# Format code
fmt:
	go fmt ./...

# Vet code for issues
vet:
	go vet ./...

# Run all checks (format, vet, lint, test)
check: fmt vet lint test

# Test CI workflow locally using act
# Requires: docker, act (brew install act)
test-ci:
	./scripts/test-workflows.sh ci

# Test release workflow locally using act (dry-run)
# Requires: docker, act (brew install act)
test-release:
	./scripts/test-workflows.sh release

# Test all workflows locally using act
# Requires: docker, act (brew install act)
test-workflows:
	./scripts/test-workflows.sh all

# Show help
help:
	@echo "Available targets:"
	@echo ""
	@echo "  Development:"
	@echo "    all              - Clean, lint, test, and build (default)"
	@echo "    build            - Build the skillrunner binary (dev mode)"
	@echo "    build-release    - Build with ldflags version injection"
	@echo "    test             - Run tests with coverage"
	@echo "    lint             - Run golangci-lint"
	@echo "    clean            - Remove build artifacts"
	@echo "    coverage         - Generate HTML coverage report"
	@echo "    deps             - Download and tidy dependencies"
	@echo "    fmt              - Format Go code"
	@echo "    vet              - Run go vet"
	@echo "    check            - Run fmt, vet, lint, and test"
	@echo ""
	@echo "  Release:"
	@echo "    release          - Run goreleaser (requires GITHUB_TOKEN)"
	@echo "    release-snapshot - Run goreleaser snapshot (no publish)"
	@echo ""
	@echo "  Workflow Testing (requires docker and act):"
	@echo "    test-ci          - Test CI workflow locally"
	@echo "    test-release     - Test release workflow (dry-run)"
	@echo "    test-workflows   - Test all workflows locally"
	@echo ""
	@echo "  Other:"
	@echo "    help             - Show this help message"
	@echo ""
	@echo "Build variables (override with VAR=value):"
	@echo "  VERSION    - Version string (default: git describe)"
	@echo "  GIT_COMMIT - Git commit hash (default: git rev-parse)"
	@echo "  BUILD_DATE - Build timestamp (default: current UTC time)"
