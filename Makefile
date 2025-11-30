.PHONY: build build-all test install clean lint fmt help

BINARY_NAME=sr
VERSION=$(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(shell date -u '+%Y-%m-%d_%H:%M:%S')
LDFLAGS=-ldflags "-X main.version=${VERSION} -X 'main.buildTime=${BUILD_TIME}'"

# Build for current platform
build:
	@echo "Building ${BINARY_NAME} for current platform..."
	go build ${LDFLAGS} -o bin/${BINARY_NAME} ./cmd/skillrunner

# Build for all platforms
build-all: clean
	@echo "Building for all platforms..."
	@mkdir -p dist

	@echo "Building for darwin/amd64..."
	GOOS=darwin GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-amd64 ./cmd/skillrunner

	@echo "Building for darwin/arm64..."
	GOOS=darwin GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-darwin-arm64 ./cmd/skillrunner

	@echo "Building for linux/amd64..."
	GOOS=linux GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-amd64 ./cmd/skillrunner

	@echo "Building for linux/arm64..."
	GOOS=linux GOARCH=arm64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-linux-arm64 ./cmd/skillrunner

	@echo "Building for windows/amd64..."
	GOOS=windows GOARCH=amd64 go build ${LDFLAGS} -o dist/${BINARY_NAME}-windows-amd64.exe ./cmd/skillrunner

	@echo "Build complete! Binaries in dist/"
	@ls -lh dist/

# Run tests
test:
	@echo "Running tests..."
	go test -v -race -cover ./...

# Run tests with coverage
test-coverage:
	@echo "Running tests with coverage..."
	go test -v -race -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report: coverage.html"

# Install to /usr/local/bin
install: build
	@echo "Installing ${BINARY_NAME} to /usr/local/bin..."
	@mkdir -p /usr/local/bin
	@cp bin/${BINARY_NAME} /usr/local/bin/${BINARY_NAME}
	@chmod +x /usr/local/bin/${BINARY_NAME}
	@echo "Installation complete! Run '${BINARY_NAME} --version' to verify."

# Clean build artifacts
clean:
	@echo "Cleaning build artifacts..."
	@rm -rf bin dist coverage.out coverage.html
	@echo "Clean complete!"

# Run linter
lint:
	@echo "Running linter..."
	@which golangci-lint > /dev/null || (echo "golangci-lint not found. Install from https://golangci-lint.run/usage/install/" && exit 1)
	golangci-lint run ./...

# Format code
fmt:
	@echo "Formatting code..."
	go fmt ./...
	@echo "Format complete!"

# Download dependencies
deps:
	@echo "Downloading dependencies..."
	go mod download
	go mod tidy
	@echo "Dependencies ready!"

# Run the binary
run: build
	@./bin/${BINARY_NAME}

# Verify code quality (quick pre-commit check)
check: fmt lint test
	@echo "All checks passed!"

# Watch for changes and run tests
watch:
	@which fswatch > /dev/null || (echo "fswatch not found. Install: brew install fswatch" && exit 1)
	@echo "Watching for changes..."
	@fswatch -o . -e ".*" -i "\\.go$$" | xargs -n1 -I{} make test

# Setup development environment
dev-setup:
	@echo "Setting up development environment..."
	@which golangci-lint > /dev/null || go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Installing git hooks..."
	@cp -n scripts/pre-commit.sh .git/hooks/pre-commit 2>/dev/null || true
	@chmod +x .git/hooks/pre-commit 2>/dev/null || true
	@echo "Development environment ready!"

# Benchmark tests
bench:
	@echo "Running benchmarks..."
	go test -bench=. -benchmem ./...

# Show help
help:
	@echo "Skillrunner Makefile"
	@echo ""
	@echo "Usage:"
	@echo "  make build          Build for current platform"
	@echo "  make build-all      Build for all platforms (darwin, linux, windows)"
	@echo "  make test           Run all tests"
	@echo "  make test-coverage  Run tests with coverage report"
	@echo "  make check          Run fmt, lint, and test (pre-commit check)"
	@echo "  make install        Install to /usr/local/bin"
	@echo "  make clean          Remove build artifacts"
	@echo "  make lint           Run golangci-lint"
	@echo "  make fmt            Format code with gofmt"
	@echo "  make deps           Download and tidy dependencies"
	@echo "  make run            Build and run binary"
	@echo "  make bench          Run benchmark tests"
	@echo "  make dev-setup      Setup development environment"
	@echo "  make watch          Watch for changes and run tests (requires fswatch)"
	@echo "  make help           Show this help message"

# Default target
.DEFAULT_GOAL := help
