#!/bin/bash

# tfplan-commenter installation script
set -e

# Configuration
REPO="akomic/go-tfplan-commenter"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="tfplan-commenter"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $OS in
    linux)
        case $ARCH in
            x86_64)
                PLATFORM="linux-amd64"
                ;;
            *)
                echo "Unsupported architecture: $ARCH"
                exit 1
                ;;
        esac
        ;;
    darwin)
        case $ARCH in
            x86_64)
                PLATFORM="darwin-amd64"
                ;;
            arm64)
                PLATFORM="darwin-arm64"
                ;;
            *)
                echo "Unsupported architecture: $ARCH"
                exit 1
                ;;
        esac
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Detected platform: $PLATFORM"

# Get latest release version
echo "Fetching latest release information..."
LATEST_VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name":' | sed -E 's/.*"([^"]+)".*/\1/')

if [ -z "$LATEST_VERSION" ]; then
    echo "Failed to get latest version"
    exit 1
fi

echo "Latest version: $LATEST_VERSION"

# Download binary
DOWNLOAD_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$BINARY_NAME-$PLATFORM"
TEMP_FILE="/tmp/$BINARY_NAME-$PLATFORM"

echo "Downloading $BINARY_NAME from $DOWNLOAD_URL..."
curl -L -o "$TEMP_FILE" "$DOWNLOAD_URL"

# Make executable
chmod +x "$TEMP_FILE"

# Install binary
echo "Installing $BINARY_NAME to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
    sudo mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

# Verify installation
if command -v "$BINARY_NAME" >/dev/null 2>&1; then
    echo "✅ Installation successful!"
    echo "Version: $($BINARY_NAME -version)"
    echo ""
    echo "Usage:"
    echo "  $BINARY_NAME plan.json"
    echo "  $BINARY_NAME -help"
else
    echo "❌ Installation failed. Please check that $INSTALL_DIR is in your PATH."
    exit 1
fi
