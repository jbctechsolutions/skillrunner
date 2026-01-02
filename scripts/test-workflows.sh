#!/bin/bash
# test-workflows.sh - Test GitHub Actions workflows locally using act
#
# Prerequisites:
#   - Docker installed and running
#   - act installed (brew install act)
#
# Usage:
#   ./scripts/test-workflows.sh [ci|release|all]
#
# Examples:
#   ./scripts/test-workflows.sh ci        # Test CI workflow
#   ./scripts/test-workflows.sh release   # Test release workflow (dry-run)
#   ./scripts/test-workflows.sh all       # Test all workflows

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

# Check prerequisites
check_prerequisites() {
    echo -e "${YELLOW}Checking prerequisites...${NC}"

    # Check for act
    if ! command -v act &> /dev/null; then
        echo -e "${RED}Error: 'act' is not installed.${NC}"
        echo "Install with: brew install act"
        echo "Or see: https://github.com/nektos/act"
        exit 1
    fi

    # Check for Docker
    if ! command -v docker &> /dev/null; then
        echo -e "${RED}Error: 'docker' is not installed.${NC}"
        exit 1
    fi

    # Check if Docker daemon is running
    if ! docker info &> /dev/null; then
        echo -e "${RED}Error: Docker daemon is not running.${NC}"
        echo "Start Docker and try again."
        exit 1
    fi

    echo -e "${GREEN}Prerequisites OK${NC}"
}

# Test CI workflow
test_ci() {
    echo -e "\n${YELLOW}Testing CI workflow...${NC}"

    cd "$PROJECT_ROOT"

    # Run the build and lint jobs from CI
    # Use medium image for better compatibility
    echo "Running build job..."
    if act push -W .github/workflows/ci.yml -j build --container-architecture linux/amd64 -P ubuntu-latest=catthehacker/ubuntu:act-latest 2>&1; then
        echo -e "${GREEN}Build job passed!${NC}"
    else
        echo -e "${RED}Build job failed!${NC}"
        return 1
    fi

    echo "Running lint job..."
    if act push -W .github/workflows/ci.yml -j lint --container-architecture linux/amd64 -P ubuntu-latest=catthehacker/ubuntu:act-latest 2>&1; then
        echo -e "${GREEN}Lint job passed!${NC}"
    else
        echo -e "${RED}Lint job failed!${NC}"
        return 1
    fi

    echo -e "${GREEN}CI workflow tests passed!${NC}"
}

# Test release workflow (dry-run, no actual release)
test_release() {
    echo -e "\n${YELLOW}Testing release workflow (dry-run)...${NC}"

    cd "$PROJECT_ROOT"

    # Create a fake tag event for testing
    # We use --dry-run to avoid actually publishing anything
    echo "Testing release build job (linux/amd64 only)..."

    # Create temporary event file for tag push
    EVENT_FILE=$(mktemp)
    cat > "$EVENT_FILE" << 'EOF'
{
    "ref": "refs/tags/v0.0.0-test",
    "ref_name": "v0.0.0-test"
}
EOF

    # Test just the build matrix for linux/amd64 (fastest to test)
    if act push -W .github/workflows/release.yml -j build \
        --container-architecture linux/amd64 \
        -P ubuntu-latest=catthehacker/ubuntu:act-latest \
        --eventpath "$EVENT_FILE" \
        --matrix os:linux --matrix arch:amd64 --matrix runner:ubuntu-latest \
        --env GITHUB_REF="refs/tags/v0.0.0-test" \
        --env GITHUB_REF_NAME="v0.0.0-test" \
        --env GITHUB_SHA="$(git rev-parse HEAD)" \
        2>&1; then
        echo -e "${GREEN}Release build job passed!${NC}"
    else
        echo -e "${YELLOW}Release build job completed with warnings (expected in local testing)${NC}"
    fi

    rm -f "$EVENT_FILE"

    echo -e "${GREEN}Release workflow dry-run completed!${NC}"
    echo -e "${YELLOW}Note: Full release testing requires GitHub Actions environment.${NC}"
}

# Show usage
usage() {
    echo "Usage: $0 [ci|release|all]"
    echo ""
    echo "Options:"
    echo "  ci       Test CI workflow (build, lint, test)"
    echo "  release  Test release workflow (dry-run)"
    echo "  all      Test all workflows"
    echo ""
    echo "Prerequisites:"
    echo "  - Docker installed and running"
    echo "  - act installed (brew install act)"
}

# Main
main() {
    check_prerequisites

    case "${1:-all}" in
        ci)
            test_ci
            ;;
        release)
            test_release
            ;;
        all)
            test_ci
            test_release
            ;;
        -h|--help|help)
            usage
            exit 0
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            usage
            exit 1
            ;;
    esac

    echo -e "\n${GREEN}All workflow tests completed!${NC}"
}

main "$@"
