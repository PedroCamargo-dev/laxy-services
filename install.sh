#!/bin/sh
set -e

REPO="PedroCamargo-dev/laxy-services"
BIN="laxy-services"
DEST="${INSTALL_DIR:-/usr/local/bin}"

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
  linux|darwin) ;;
  *) echo "unsupported OS: $OS" >&2; exit 1 ;;
esac

ARCH=$(uname -m)
case "$ARCH" in
  x86_64)  ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

LATEST=$(curl -sf "https://api.github.com/repos/$REPO/releases/latest" \
  | grep '"tag_name"' | cut -d'"' -f4)

if [ -z "$LATEST" ]; then
  echo "could not fetch latest release — check your internet connection" >&2
  exit 1
fi

FILE="${BIN}_${LATEST}_${OS}_${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$LATEST/$FILE"

echo "installing $BIN $LATEST ($OS/$ARCH) → $DEST/$BIN"
curl -fL "$URL" -o "/tmp/$FILE"
tar -xzf "/tmp/$FILE" -C /tmp "$BIN"
chmod +x "/tmp/$BIN"
rm "/tmp/$FILE"

if [ -w "$DEST" ]; then
  mv "/tmp/$BIN" "$DEST/$BIN"
else
  sudo mv "/tmp/$BIN" "$DEST/$BIN"
fi

echo "done — run: $BIN"
