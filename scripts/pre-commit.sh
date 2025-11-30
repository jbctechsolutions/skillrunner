#!/bin/bash
# Pre-commit hook for Skillrunner
# Automatically runs format, lint, and test checks before each commit

set -e

echo "Running pre-commit checks..."
echo "=============================="

# 1. Format code
echo ""
echo "1. Formatting code..."
if ! make fmt > /dev/null 2>&1; then
    echo "Error: Code formatting failed"
    exit 1
fi
echo "✓ Code formatted"

# 2. Run linter
echo ""
echo "2. Running linter..."
if ! make lint > /dev/null 2>&1; then
    echo "Error: Linting failed. Run 'make lint' to see details."
    exit 1
fi
echo "✓ Linter passed"

# 3. Run tests (short mode for speed)
echo ""
echo "3. Running tests..."
if ! go test ./... -short > /dev/null 2>&1; then
    echo "Error: Tests failed. Run 'make test' to see details."
    exit 1
fi
echo "✓ Tests passed"

echo ""
echo "=============================="
echo "✓ All pre-commit checks passed!"
echo "Proceeding with commit..."
