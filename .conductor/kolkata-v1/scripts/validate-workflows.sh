#!/bin/bash
#
# Validate GitHub Actions workflow YAML files
#
# This script checks that all workflow files are valid YAML
# and follows GitHub Actions best practices.
#

set -e

WORKFLOWS_DIR=".github/workflows"
ERRORS=0

echo "Validating GitHub Actions workflows..."
echo "======================================"
echo ""

# Check if yamllint is installed
if ! command -v yamllint &> /dev/null; then
    echo "Warning: yamllint not installed. Installing via pip..."
    pip3 install yamllint --quiet || {
        echo "Warning: Could not install yamllint. Skipping YAML validation."
        echo "Install manually with: pip3 install yamllint"
        exit 0
    }
fi

# Validate each workflow file
for workflow in "$WORKFLOWS_DIR"/*.yml "$WORKFLOWS_DIR"/*.yaml; do
    if [ ! -f "$workflow" ]; then
        continue
    fi

    echo "Validating: $workflow"

    # Run yamllint
    if yamllint -d "{extends: default, rules: {line-length: {max: 120}}}" "$workflow"; then
        echo "  ✓ Valid YAML syntax"
    else
        echo "  ✗ YAML syntax errors found"
        ERRORS=$((ERRORS + 1))
    fi

    # Check for common issues
    if grep -q "actions/checkout@v[0-2]" "$workflow"; then
        echo "  ⚠ Warning: Using old checkout action version (v1-v2). Consider upgrading to v4."
    fi

    if grep -q "actions/setup-go@v[0-3]" "$workflow"; then
        echo "  ⚠ Warning: Using old setup-go action version. Consider upgrading to v5."
    fi

    if ! grep -q "timeout-minutes:" "$workflow"; then
        echo "  ⚠ Warning: No timeout-minutes specified. Consider adding timeouts."
    fi

    echo ""
done

# Validate dependabot.yml
if [ -f ".github/dependabot.yml" ]; then
    echo "Validating: .github/dependabot.yml"
    if yamllint .github/dependabot.yml; then
        echo "  ✓ Valid YAML syntax"
    else
        echo "  ✗ YAML syntax errors found"
        ERRORS=$((ERRORS + 1))
    fi
    echo ""
fi

# Summary
if [ $ERRORS -eq 0 ]; then
    echo "======================================"
    echo "✓ All workflow files are valid!"
    exit 0
else
    echo "======================================"
    echo "✗ Found $ERRORS error(s) in workflow files"
    exit 1
fi
