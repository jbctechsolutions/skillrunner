#!/bin/bash
# scripts/start-services.sh
# Start Ollama and LiteLLM services via Docker Compose

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
PROJECT_ROOT="$( cd "$SCRIPT_DIR/.." && pwd )"

cd "$PROJECT_ROOT"

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}Error: Docker is not running. Please start Docker and try again.${NC}"
    exit 1
fi

# Check if docker-compose is available
if ! command -v docker-compose > /dev/null 2>&1 && ! docker compose version > /dev/null 2>&1; then
    echo -e "${RED}Error: docker-compose is not installed.${NC}"
    exit 1
fi

# Use 'docker compose' if available, otherwise 'docker-compose'
if docker compose version > /dev/null 2>&1; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

# Check if services are already running
if $DOCKER_COMPOSE ps | grep -q "Up"; then
    echo -e "${GREEN}Services are already running${NC}"
    $DOCKER_COMPOSE ps
    exit 0
fi

# Check for .env file
if [ ! -f .env ]; then
    echo -e "${YELLOW}Warning: .env file not found.${NC}"
    echo "Creating .env from .env.example..."
    if [ -f .env.example ]; then
        cp .env.example .env
        echo -e "${YELLOW}Please edit .env and add your API keys, then run this script again.${NC}"
        exit 1
    else
        echo -e "${RED}Error: .env.example not found.${NC}"
        exit 1
    fi
fi

# Start services
echo "Starting Ollama and LiteLLM services..."
$DOCKER_COMPOSE up -d

# Wait for services to be healthy
echo "Waiting for services to be ready..."
sleep 5

# Check Ollama health
echo -n "Checking Ollama... "
if curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Ready${NC}"
else
    echo -e "${YELLOW}⚠️  May still be starting${NC}"
    echo "   Check logs with: $DOCKER_COMPOSE logs ollama"
fi

# Check LiteLLM health
echo -n "Checking LiteLLM... "
if curl -s http://localhost:4000/health > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Ready${NC}"
else
    echo -e "${YELLOW}⚠️  May still be starting${NC}"
    echo "   Check logs with: $DOCKER_COMPOSE logs litellm"
fi

echo ""
echo -e "${GREEN}Services started!${NC}"
echo ""
echo "Useful commands:"
echo "  View logs:        $DOCKER_COMPOSE logs -f"
echo "  Stop services:    $DOCKER_COMPOSE down"
echo "  Restart services: $DOCKER_COMPOSE restart"
echo "  Status:            $DOCKER_COMPOSE ps"
