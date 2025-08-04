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
                GORELEASER_PLATFORM="Linux_x86_64"
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
                GORELEASER_PLATFORM="Darwin_x86_64"
                ;;
            arm64)
                PLATFORM="darwin-arm64"
                GORELEASER_PLATFORM="Darwin_arm64"
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

# Try GoReleaser format first, then fallback to direct binary
TEMP_FILE="/tmp/$BINARY_NAME"
DOWNLOAD_SUCCESS=false

# Try GoReleaser archive format
GORELEASER_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/${BINARY_NAME}_${GORELEASER_PLATFORM}.tar.gz"
echo "Trying GoReleaser format: $GORELEASER_URL"

if curl -L --fail -o "/tmp/${BINARY_NAME}.tar.gz" "$GORELEASER_URL" 2>/dev/null; then
    echo "Downloaded GoReleaser archive, extracting..."
    cd /tmp
    tar -xzf "${BINARY_NAME}.tar.gz" "$BINARY_NAME" 2>/dev/null || {
        echo "Failed to extract from archive, trying direct binary..."
    }
    if [ -f "/tmp/$BINARY_NAME" ]; then
        DOWNLOAD_SUCCESS=true
    fi
fi

# Fallback to direct binary download
if [ "$DOWNLOAD_SUCCESS" = false ]; then
    DIRECT_URL="https://github.com/$REPO/releases/download/$LATEST_VERSION/$BINARY_NAME-$PLATFORM"
    echo "Trying direct binary: $DIRECT_URL"
    
    if curl -L --fail -o "$TEMP_FILE" "$DIRECT_URL" 2>/dev/null; then
        DOWNLOAD_SUCCESS=true
    else
        echo "❌ Failed to download $BINARY_NAME"
        echo "Please check the releases page: https://github.com/$REPO/releases"
        exit 1
    fi
fi

if [ "$DOWNLOAD_SUCCESS" = true ]; then
    echo "✅ Download successful"
else
    echo "❌ Download failed"
    exit 1
fi

# Make executable
chmod +x "$TEMP_FILE"

# Install binary
echo "Installing $BINARY_NAME to $INSTALL_DIR..."
if [ -w "$INSTALL_DIR" ]; then
    mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
else
    sudo mv "$TEMP_FILE" "$INSTALL_DIR/$BINARY_NAME"
fi

# Clean up
rm -f "/tmp/${BINARY_NAME}.tar.gz"

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
