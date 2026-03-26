#!/bin/sh
# Install script for clean-sql
# Usage: curl -sSL https://raw.githubusercontent.com/jimmyalcala/clean-sql/main/install.sh | sh
set -e

REPO="jimmyalcala/clean-sql"
BINARY="clean-sql"
INSTALL_DIR="/usr/local/bin"

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case "$ARCH" in
    x86_64|amd64) ARCH="amd64" ;;
    arm64|aarch64) ARCH="arm64" ;;
    *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

case "$OS" in
    darwin|linux) ;;
    *) echo "Unsupported OS: $OS"; exit 1 ;;
esac

ASSET="${BINARY}-${OS}-${ARCH}"

# Get latest release URL
LATEST=$(curl -sSL "https://api.github.com/repos/${REPO}/releases/latest" | grep "browser_download_url.*${ASSET}" | cut -d '"' -f 4)

if [ -z "$LATEST" ]; then
    echo "Could not find release for ${ASSET}"
    echo "Check https://github.com/${REPO}/releases"
    exit 1
fi

echo "Downloading ${BINARY} for ${OS}/${ARCH}..."
curl -sSL "$LATEST" -o "/tmp/${BINARY}"
chmod +x "/tmp/${BINARY}"

echo "Installing to ${INSTALL_DIR}/${BINARY}..."
if [ -w "$INSTALL_DIR" ]; then
    mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
else
    sudo mv "/tmp/${BINARY}" "${INSTALL_DIR}/${BINARY}"
fi

echo "Done! Run: ${BINARY} --help"
