#!/bin/bash
# flow-cli installation script
# Usage: curl -fsSL https://raw.githubusercontent.com/alvesdmateus/flow-cli/main/install.sh | bash

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
REPO="alvesdmateus/flow-cli"
BINARY_NAME="flow"
INSTALL_DIR="${INSTALL_DIR:-/usr/local/bin}"

# Detect OS and architecture
detect_platform() {
    OS=$(uname -s | tr '[:upper:]' '[:lower:]')
    ARCH=$(uname -m)

    case $ARCH in
        x86_64)
            ARCH="amd64"
            ;;
        aarch64|arm64)
            ARCH="arm64"
            ;;
        armv7l)
            ARCH="arm"
            ;;
        *)
            echo -e "${RED}Unsupported architecture: $ARCH${NC}"
            exit 1
            ;;
    esac

    case $OS in
        linux)
            OS="linux"
            ;;
        darwin)
            OS="darwin"
            ;;
        mingw*|msys*|cygwin*)
            OS="windows"
            BINARY_NAME="flow.exe"
            ;;
        *)
            echo -e "${RED}Unsupported operating system: $OS${NC}"
            exit 1
            ;;
    esac

    echo "${OS}_${ARCH}"
}

# Get latest release version
get_latest_version() {
    curl -fsSL "https://api.github.com/repos/${REPO}/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/'
}

# Download and install
install_flow() {
    PLATFORM=$(detect_platform)
    VERSION=${VERSION:-$(get_latest_version)}

    if [ -z "$VERSION" ]; then
        echo -e "${YELLOW}Could not detect latest version, using 'latest'${NC}"
        VERSION="latest"
    fi

    echo -e "${BLUE}Installing flow-cli ${VERSION} for ${PLATFORM}...${NC}"

    # Construct download URL
    DOWNLOAD_URL="https://github.com/${REPO}/releases/download/${VERSION}/flow_${VERSION#v}_${PLATFORM}.tar.gz"

    # Create temporary directory
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    # Download
    echo -e "${BLUE}Downloading from ${DOWNLOAD_URL}...${NC}"
    if ! curl -fsSL "$DOWNLOAD_URL" -o "$TMP_DIR/flow.tar.gz"; then
        echo -e "${RED}Failed to download flow-cli${NC}"
        echo -e "${YELLOW}You can build from source instead:${NC}"
        echo "  git clone https://github.com/${REPO}.git"
        echo "  cd flow-cli"
        echo "  go build -o flow ."
        exit 1
    fi

    # Extract
    echo -e "${BLUE}Extracting...${NC}"
    tar -xzf "$TMP_DIR/flow.tar.gz" -C "$TMP_DIR"

    # Install
    echo -e "${BLUE}Installing to ${INSTALL_DIR}...${NC}"
    if [ -w "$INSTALL_DIR" ]; then
        mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/"
        chmod +x "$INSTALL_DIR/$BINARY_NAME"
    else
        echo -e "${YELLOW}Need sudo to install to ${INSTALL_DIR}${NC}"
        sudo mv "$TMP_DIR/$BINARY_NAME" "$INSTALL_DIR/"
        sudo chmod +x "$INSTALL_DIR/$BINARY_NAME"
    fi

    echo -e "${GREEN}✓ flow-cli installed successfully!${NC}"
    echo ""
    echo "Run 'flow --help' to get started."
    echo ""
    echo "Quick start:"
    echo "  1. Make sure Ollama is running: ollama serve"
    echo "  2. Initialize config: flow config init"
    echo "  3. Start chatting: flow chat"
}

# Build from source
build_from_source() {
    echo -e "${BLUE}Building from source...${NC}"

    # Check for Go
    if ! command -v go &> /dev/null; then
        echo -e "${RED}Go is not installed. Please install Go 1.21+ first.${NC}"
        echo "Visit: https://golang.org/dl/"
        exit 1
    fi

    # Check Go version
    GO_VERSION=$(go version | grep -oE 'go[0-9]+\.[0-9]+' | sed 's/go//')
    MAJOR=$(echo $GO_VERSION | cut -d. -f1)
    MINOR=$(echo $GO_VERSION | cut -d. -f2)

    if [ "$MAJOR" -lt 1 ] || ([ "$MAJOR" -eq 1 ] && [ "$MINOR" -lt 21 ]); then
        echo -e "${RED}Go 1.21+ is required. Found: go${GO_VERSION}${NC}"
        exit 1
    fi

    # Clone and build
    TMP_DIR=$(mktemp -d)
    trap "rm -rf $TMP_DIR" EXIT

    echo -e "${BLUE}Cloning repository...${NC}"
    git clone "https://github.com/${REPO}.git" "$TMP_DIR/flow-cli"
    cd "$TMP_DIR/flow-cli"

    echo -e "${BLUE}Building...${NC}"
    go build -ldflags "-s -w -X github.com/alvesdmateus/flow-cli/cmd.Version=$(git describe --tags --always)" -o flow .

    # Install
    echo -e "${BLUE}Installing to ${INSTALL_DIR}...${NC}"
    if [ -w "$INSTALL_DIR" ]; then
        mv flow "$INSTALL_DIR/"
    else
        sudo mv flow "$INSTALL_DIR/"
    fi

    echo -e "${GREEN}✓ flow-cli built and installed successfully!${NC}"
}

# Main
main() {
    echo -e "${BLUE}"
    echo "╭─────────────────────────────────────────╮"
    echo "│       flow-cli Installer                │"
    echo "╰─────────────────────────────────────────╯"
    echo -e "${NC}"

    case "${1:-install}" in
        install)
            install_flow
            ;;
        build)
            build_from_source
            ;;
        *)
            echo "Usage: $0 [install|build]"
            echo ""
            echo "  install  - Download and install pre-built binary (default)"
            echo "  build    - Build from source"
            exit 1
            ;;
    esac
}

main "$@"
