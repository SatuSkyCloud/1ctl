#!/bin/bash
set -euo pipefail

# SatuSky CLI dev build (1ctl-dev) installer — INTERNAL TESTING ONLY.
# Installs a separate `1ctl-dev` binary that targets the dev backend
# (https://dev-core-api.satusky.com). Safe to install alongside the prod `1ctl`.
#
# Usage: curl -sSL https://raw.githubusercontent.com/SatuSkyCloud/1ctl/main/install-dev.sh | bash

REPO="SatuSkyCloud/1ctl"
INSTALL_DIR="/usr/local/bin"
BINARY="1ctl-dev"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  mingw*|msys*|cygwin*)
    echo "Windows detected. Please download the 1ctl-dev archive from:"
    echo "  https://github.com/$REPO/releases/latest"
    exit 1
    ;;
  *)
    echo "Unsupported OS: $OS"
    exit 1
    ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)       ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *)
    echo "Unsupported architecture: $ARCH"
    exit 1
    ;;
esac

echo "Fetching latest version..."
VERSION=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep '"tag_name"' | sed -E 's/.*"([^"]+)".*/\1/')
if [ -z "$VERSION" ] || [ "$VERSION" = "null" ]; then
  echo "Failed to fetch latest version. Check your internet connection."
  exit 1
fi
CLEAN_VERSION="${VERSION#v}"

echo "Installing $BINARY $VERSION ($OS/$ARCH) — points to dev-core-api.satusky.com"

URL="https://github.com/$REPO/releases/download/$VERSION/1ctl-dev-$CLEAN_VERSION-$OS-$ARCH.tar.gz"
TMPDIR=$(mktemp -d)
trap 'rm -rf "$TMPDIR"' EXIT

curl -fsSL "$URL" -o "$TMPDIR/$BINARY.tar.gz"
tar -xzf "$TMPDIR/$BINARY.tar.gz" -C "$TMPDIR"
chmod +x "$TMPDIR/$BINARY"

if [ -w "$INSTALL_DIR" ]; then
  mv "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
else
  echo "Installing to $INSTALL_DIR (requires sudo)..."
  sudo mv "$TMPDIR/$BINARY" "$INSTALL_DIR/$BINARY"
fi

echo "$BINARY $VERSION installed successfully!"
echo "Run '$BINARY --version' to verify (should show [development])."
echo "Note: the dev binary respects SATUSKY_API_URL like prod — env var overrides the baked-in dev default."
