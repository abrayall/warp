#!/bin/sh

# Warp Install Script
# Builds locally if in git repo, otherwise downloads from GitHub releases

set -e

# Colors
GREEN='\033[38;2;39;201;63m'
PURPLE='\033[38;2;157;78;221m'
BLUE='\033[38;2;59;130;246m'
WHITE='\033[1;37m'
RED='\033[0;31m'
NC='\033[0m'

# Use printf for POSIX compatibility
log() { printf "%b\n" "$1"; }
log_err() { printf "%b\n" "$1" >&2; }

REPO="abrayall/warp"
INSTALL_DIR="/usr/local/bin"

# Detect platform and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case "$OS" in
        darwin) OS="darwin" ;;
        linux) OS="linux" ;;
        *) log_err "${RED}Unsupported OS: $OS${NC}"; exit 1 ;;
    esac

    case "$ARCH" in
        x86_64) ARCH="amd64" ;;
        amd64) ARCH="amd64" ;;
        arm64) ARCH="arm64" ;;
        aarch64) ARCH="arm64" ;;
        *) log_err "${RED}Unsupported architecture: $ARCH${NC}"; exit 1 ;;
    esac
}

# Check if we're in the warp git repo
is_in_repo() {
    if [ -f "go.mod" ] && grep -q "warp" go.mod 2>/dev/null; then
        if [ -f "build.sh" ]; then
            return 0
        fi
    fi
    return 1
}

# Build from source
build_local() {
    ./build.sh >&2

    BINARY="warp"
    if [ ! -f "$BINARY" ]; then
        log_err "${RED}Build failed — no binary found${NC}"
        exit 1
    fi

    echo "$BINARY"
}

# Download from GitHub releases
download_release() {
    log_err "${BLUE}Fetching latest release...${NC}"

    # Get latest release tag
    if command -v curl > /dev/null 2>&1; then
        LATEST=$(curl -sL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    elif command -v wget > /dev/null 2>&1; then
        LATEST=$(wget -qO- "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
    else
        log_err "${RED}Error: curl or wget is required${NC}"
        exit 1
    fi

    if [ -z "$LATEST" ]; then
        log_err "${RED}Failed to fetch latest release${NC}"
        exit 1
    fi

    log_err "${BLUE}Latest version: ${WHITE}${LATEST}${NC}"
    echo "" >&2

    # Construct download URL
    # Strip leading 'v' from tag for filename
    VER="${LATEST#v}"
    FILENAME="warp-${VER}-${OS}-${ARCH}"
    URL="https://github.com/${REPO}/releases/download/${LATEST}/${FILENAME}"

    # Create temp directory
    TMP_DIR=$(mktemp -d)
    BINARY="${TMP_DIR}/warp"

    log_err "${BLUE}Downloading ${FILENAME}...${NC}"

    # Download binary
    if command -v curl > /dev/null 2>&1; then
        if ! curl -sL -f -o "$BINARY" "$URL"; then
            log_err "${RED}Failed to download from ${URL}${NC}"
            rm -rf "$TMP_DIR"
            exit 1
        fi
    else
        if ! wget -q -O "$BINARY" "$URL"; then
            log_err "${RED}Failed to download from ${URL}${NC}"
            rm -rf "$TMP_DIR"
            exit 1
        fi
    fi

    chmod +x "$BINARY"
    echo "$BINARY"
}

# Install binary
install_binary() {
    BINARY="$1"

    log "${BLUE}Installing to ${INSTALL_DIR}...${NC}"

    if [ -w "$INSTALL_DIR" ]; then
        cp "$BINARY" "$INSTALL_DIR/warp"
        chmod +x "$INSTALL_DIR/warp"
    else
        sudo cp "$BINARY" "$INSTALL_DIR/warp"
        sudo chmod +x "$INSTALL_DIR/warp"
    fi
}

# Main
detect_platform

if is_in_repo; then
    BINARY=$(build_local)
else
    BINARY=$(download_release)
fi

install_binary "$BINARY"

# Cleanup temp files if downloaded
case "$BINARY" in
    /tmp/*|*/tmp.*) rm -rf "$(dirname "$BINARY")" ;;
esac

echo ""
log "${GREEN}✓ Installed warp to ${INSTALL_DIR}/warp${NC}"
echo ""
echo "Run 'warp --help' to get started"
echo ""
