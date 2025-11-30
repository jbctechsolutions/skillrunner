#!/bin/bash
#
# Post-installation script for Skillrunner
# Runs after package installation (deb/rpm)
#

set -e

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}Configuring Skillrunner...${NC}"

# 1. Create config directory for all users
CONFIG_DIR="/etc/skillrunner"
if [ ! -d "$CONFIG_DIR" ]; then
    mkdir -p "$CONFIG_DIR"
    chmod 755 "$CONFIG_DIR"
fi

# 2. Setup user config directory template
USER_CONFIG_TEMPLATE="/etc/skel/.skillrunner"
if [ ! -d "$USER_CONFIG_TEMPLATE" ]; then
    mkdir -p "$USER_CONFIG_TEMPLATE"
    chmod 755 "$USER_CONFIG_TEMPLATE"
fi

# 3. Create user directories for existing users (optional)
for home in /home/*; do
    if [ -d "$home" ]; then
        user=$(basename "$home")
        user_config="$home/.skillrunner"

        # Only create if doesn't exist and user can write
        if [ ! -d "$user_config" ] && [ -w "$home" ]; then
            mkdir -p "$user_config"
            chown "$user:$user" "$user_config"
            chmod 755 "$user_config"
        fi
    fi
done

# 4. Verify binary is executable
if [ -f "/usr/bin/sr" ]; then
    chmod +x /usr/bin/sr
    # Create symlink so both 'sr' and 'skillrunner' work
    ln -sf /usr/bin/sr /usr/bin/skillrunner
fi

# 5. Print post-install message
echo ""
echo -e "${GREEN}Skillrunner installed successfully!${NC}"
echo ""
echo "Next steps:"
echo "  1. Install Ollama: https://ollama.ai"
echo "  2. Pull models: ollama pull qwen2.5:14b"
echo "  3. Run: sr list"
echo ""
echo "Documentation: https://github.com/jbctechsolutions/skillrunner"
echo ""

# 6. Check if Ollama is installed
if command -v ollama &> /dev/null; then
    echo -e "${GREEN}✓ Ollama detected${NC}"
else
    echo -e "${YELLOW}⚠ Ollama not found. Install from https://ollama.ai${NC}"
fi

exit 0
