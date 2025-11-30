#!/bin/bash
#
# Skillrunner Installation Script
#
# Usage:
#   curl -sSL https://raw.githubusercontent.com/jbctechsolutions/skillrunner/main/install.sh | bash
#   or
#   ./install.sh
#

set -e

BINARY_NAME="sr"
INSTALL_DIR="/usr/local/bin"
REPO="jbctechsolutions/skillrunner"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin)
            OS="darwin"
            ;;
        linux)
            OS="linux"
            ;;
        mingw* | msys* | cygwin*)
            OS="windows"
            ;;
        *)
            echo -e "${RED}Unsupported OS: $OS${NC}"
            exit 1
            ;;
    esac

    case "$ARCH" in
        x86_64 | amd64)
            ARCH="amd64"
            ;;
        arm64 | aarch64)
            ARCH="arm64"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    PLATFORM="${OS}-${ARCH}"
    echo -e "${GREEN}Detected platform: $PLATFORM${NC}"
}

# Check if running from local build
install_local() {
    echo -e "${YELLOW}Installing from local build...${NC}"

    # Look for the binary in common locations
    if [ -f "bin/${BINARY_NAME}" ]; then
        BINARY_PATH="bin/${BINARY_NAME}"
    elif [ -f "dist/${BINARY_NAME}-${PLATFORM}" ]; then
        BINARY_PATH="dist/${BINARY_NAME}-${PLATFORM}"
    else
        echo -e "${RED}No local binary found. Run 'make build' or 'make build-all' first.${NC}"
        exit 1
    fi

    echo -e "${GREEN}Found binary: $BINARY_PATH${NC}"

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        echo -e "${YELLOW}Creating $INSTALL_DIR...${NC}"
        sudo mkdir -p "$INSTALL_DIR"
    fi

    # Copy binary
    echo -e "${YELLOW}Copying binary to $INSTALL_DIR...${NC}"
    sudo cp "$BINARY_PATH" "$INSTALL_DIR/${BINARY_NAME}"
    sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}"

    # Create symlink so both 'sr' and 'skillrunner' work
    echo -e "${YELLOW}Creating skillrunner symlink...${NC}"
    sudo ln -sf "$INSTALL_DIR/${BINARY_NAME}" "$INSTALL_DIR/skillrunner"

    echo -e "${GREEN}Installation complete!${NC}"
    echo -e "${GREEN}Run '${BINARY_NAME} --version' or 'skillrunner --version' to verify.${NC}"
}

# Download and install from GitHub releases
install_github() {
    echo -e "${YELLOW}Installing from GitHub releases...${NC}"

    # Determine binary extension
    BINARY_EXT=""
    if [ "$OS" = "windows" ]; then
        BINARY_EXT=".exe"
    fi

    BINARY_FILE="${BINARY_NAME}-${PLATFORM}${BINARY_EXT}"
    DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/${BINARY_FILE}"

    echo -e "${GREEN}Downloading from: $DOWNLOAD_URL${NC}"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    cd "$TMP_DIR"

    # Download binary
    if command -v curl &> /dev/null; then
        curl -fsSL -o "${BINARY_NAME}" "${DOWNLOAD_URL}"
    elif command -v wget &> /dev/null; then
        wget -q -O "${BINARY_NAME}" "${DOWNLOAD_URL}"
    else
        echo -e "${RED}Neither curl nor wget found. Please install one of them.${NC}"
        exit 1
    fi

    # Check if download was successful
    if [ ! -f "${BINARY_NAME}" ]; then
        echo -e "${RED}Failed to download binary from GitHub releases.${NC}"
        echo -e "${YELLOW}Please try installing from source:${NC}"
        echo -e "${YELLOW}  git clone https://github.com/${REPO}${NC}"
        echo -e "${YELLOW}  cd skillrunner && make install${NC}"
        exit 1
    fi

    # Verify it's a valid binary (basic check)
    if [ ! -s "${BINARY_NAME}" ]; then
        echo -e "${RED}Downloaded file is empty or invalid.${NC}"
        exit 1
    fi

    # Download checksums for verification (optional but recommended)
    if curl -fsSL -o checksums.txt "https://github.com/${REPO}/releases/latest/download/checksums.txt" 2>/dev/null; then
        echo -e "${YELLOW}Verifying checksum...${NC}"
        if command -v sha256sum &> /dev/null; then
            grep "${BINARY_FILE}" checksums.txt | sha256sum -c - || {
                echo -e "${YELLOW}Warning: Checksum verification failed${NC}"
            }
        fi
    fi

    # Create install directory if it doesn't exist
    if [ ! -d "$INSTALL_DIR" ]; then
        echo -e "${YELLOW}Creating $INSTALL_DIR...${NC}"
        sudo mkdir -p "$INSTALL_DIR"
    fi

    # Install binary
    echo -e "${YELLOW}Installing to $INSTALL_DIR...${NC}"
    sudo mv "${BINARY_NAME}" "$INSTALL_DIR/${BINARY_NAME}"
    sudo chmod +x "$INSTALL_DIR/${BINARY_NAME}"

    # Create symlink so both 'sr' and 'skillrunner' work
    echo -e "${YELLOW}Creating skillrunner symlink...${NC}"
    sudo ln -sf "$INSTALL_DIR/${BINARY_NAME}" "$INSTALL_DIR/skillrunner"

    # Cleanup
    cd -
    rm -rf "$TMP_DIR"

    echo -e "${GREEN}Installation complete!${NC}"
    echo -e "${GREEN}Run '${BINARY_NAME} --version' or 'skillrunner --version' to verify.${NC}"
}

# Main installation flow
main() {
    echo "Skillrunner Installation Script"
    echo "================================"
    echo ""

    detect_platform

    # Check if running from repo directory
    if [ -f "go.mod" ] && grep -q "skillrunner" go.mod 2>/dev/null; then
        install_local
    else
        install_github
    fi
}

main
