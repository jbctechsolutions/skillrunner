# Skillrunner Makefile
# Build and development automation

.PHONY: all build test lint clean help

# Default target
all: clean lint test build

# Build the skillrunner binary
build:
	go build -o skillrunner ./cmd/skillrunner

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

# Show help
help:
	@echo "Available targets:"
	@echo "  all       - Clean, lint, test, and build (default)"
	@echo "  build     - Build the skillrunner binary"
	@echo "  test      - Run tests with coverage"
	@echo "  lint      - Run golangci-lint"
	@echo "  clean     - Remove build artifacts"
	@echo "  coverage  - Generate HTML coverage report"
	@echo "  deps      - Download and tidy dependencies"
	@echo "  fmt       - Format Go code"
	@echo "  vet       - Run go vet"
	@echo "  check     - Run fmt, vet, lint, and test"
	@echo "  help      - Show this help message"
