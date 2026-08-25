#!/usr/bin/env bash
# loki.sh - Launch Loki Mode in a Docker container
#
# Usage:
#   ./loki.sh                          # Start interactive session (type prompt inside)
#   ./loki.sh prd docs/plans/prd.md    # Start with a specific PRD file
#   ./loki.sh build                    # Rebuild the Docker image
#   ./loki.sh shell                    # Open a shell instead of claude (for debugging)
#
set -euo pipefail

COMPOSE_FILE="docker-compose.loki.yaml"

# Verify ~/.claude exists (needed for Claude Code auth)
if [[ ! -d "$HOME/.claude" ]]; then
    echo "Error: ~/.claude directory not found."
    echo "Run 'claude' locally first to authenticate, then retry."
    exit 1
fi

# Create ~/.npm if it doesn't exist (bind mount requires the dir to exist)
mkdir -p "$HOME/.npm"

case "${1:-start}" in
    build)
        echo "Building Loki Mode image..."
        docker compose -f "$COMPOSE_FILE" build
        ;;
    shell)
        echo "Opening shell in Loki container..."
        docker compose -f "$COMPOSE_FILE" run --rm \
            --entrypoint bash \
            loki
        ;;
    prd)
        PRD_PATH="${2:?Usage: ./loki.sh prd <path/to/prd.md>}"
        echo "Starting Loki Mode with PRD: $PRD_PATH"
        echo ""
        docker compose -f "$COMPOSE_FILE" run --rm loki \
            "Loki Mode with PRD at $PRD_PATH"
        ;;
    start|*)
        echo "Starting Loki Mode..."
        echo "Inside the container, say: Loki Mode"
        echo "Or: Loki Mode with PRD at docs/plans/prd.md"
        echo ""
        docker compose -f "$COMPOSE_FILE" run --rm loki
        ;;
esac
