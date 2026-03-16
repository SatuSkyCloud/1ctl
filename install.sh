#!/bin/bash
set -euo pipefail

# SatuSky CLI (1ctl) installer
# Usage: curl -sSL https://raw.githubusercontent.com/SatuSkyCloud/1ctl/main/install.sh | bash

REPO="SatuSkyCloud/1ctl"
INSTALL_DIR="/usr/local/bin"

# Detect OS
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*)
    echo "Windows detected. Please download from:"
    echo "  https://github.com/$REPO/releases/latest"
    exit 1
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

# Detect architecture
ARCH=$(uname -m)
case "$ARCH" in
  x86_64)       ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

# Get latest version
echo "Fetching latest version..."
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ] || [ "$VERSION" = "null" ]; then
  echo "Failed to fetch latest version. Check your internet connection."
  exit 1
fi
CLEAN_VERSION="${VERSION#v}"

echo "Installing 1ctl $VERSION ($OS/$ARCH)..."

# Download and extract
URL="https://github.com/$REPO/releases/download/$VERSION/1ctl-$CLEAN_VERSION-$OS-$ARCH.tar.gz"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -sL "$URL" -o "$TMPDIR/1ctl.tar.gz"
tar -xzf "$TMPDIR/1ctl.tar.gz" -C "$TMPDIR"
chmod +x "$TMPDIR/1ctl"

# Install
if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR/1ctl" "$INSTALL_DIR/1ctl"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMPDIR/1ctl" "$INSTALL_DIR/1ctl"
fi

echo "1ctl $VERSION installed successfully!"
echo "Run '1ctl --help' to get started."
