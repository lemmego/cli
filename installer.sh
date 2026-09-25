#!/bin/sh
# Installs the Lemmego CLI.
#
#   curl -fsSL https://raw.githubusercontent.com/lemmego/cli/refs/heads/main/installer.sh | sudo sh
#
# The version is resolved from the latest GitHub release, so this script does
# not need editing when a new version ships. Override it to pin:
#
#   LEMMEGO_VERSION=v0.1.45 ... | sudo sh
#   LEMMEGO_INSTALL_DIR=$HOME/.local/bin ... | sh
set -eu

REPO="lemmego/cli"
BIN_NAME="lemmego"
INSTALL_DIR="${LEMMEGO_INSTALL_DIR:-/usr/local/bin}"

OS=$(uname | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')

case "$OS" in
    darwin | linux) ;;
    *)
        echo "Unsupported OS: $OS" >&2
        exit 1
        ;;
esac

case "$ARCH" in
    amd64 | arm64) ;;
    *)
        echo "Unsupported architecture: $ARCH" >&2
        exit 1
        ;;
esac

# Follow the /releases/latest redirect rather than parsing the API, so this
# works without jq on a bare system.
VERSION="${LEMMEGO_VERSION:-}"
if [ -z "$VERSION" ]; then
    VERSION=$(curl -fsSLI -o /dev/null -w '%{url_effective}' \
        "https://github.com/$REPO/releases/latest" 2>/dev/null | sed 's#.*/tag/##')
fi

if [ -z "$VERSION" ]; then
    echo "Could not determine the latest version." >&2
    echo "Set LEMMEGO_VERSION to install a specific one, e.g. LEMMEGO_VERSION=v0.1.46" >&2
    exit 1
fi

ASSET="$BIN_NAME-$VERSION-$OS-$ARCH"
BASE_URL="https://github.com/$REPO/releases/download/$VERSION"

TMP_DIR=$(mktemp -d)
trap 'rm -rf "$TMP_DIR"' EXIT

echo "Downloading $ASSET"
if ! curl -fL --progress-bar "$BASE_URL/$ASSET" -o "$TMP_DIR/$BIN_NAME"; then
    echo "Download failed: $BASE_URL/$ASSET" >&2
    exit 1
fi

# Verify against the checksum published alongside the binary.
if curl -fsSL "$BASE_URL/$ASSET.md5" -o "$TMP_DIR/checksum" 2>/dev/null; then
    expected=$(tr -d ' \t\r\n' <"$TMP_DIR/checksum")
    if command -v md5sum >/dev/null 2>&1; then
        actual=$(md5sum "$TMP_DIR/$BIN_NAME" | awk '{print $1}')
    elif command -v md5 >/dev/null 2>&1; then
        actual=$(md5 -q "$TMP_DIR/$BIN_NAME")
    else
        actual=""
    fi
    if [ -n "$actual" ] && [ "$actual" != "$expected" ]; then
        echo "Checksum mismatch for $ASSET" >&2
        echo "  expected $expected" >&2
        echo "  actual   $actual" >&2
        exit 1
    fi
fi

chmod +x "$TMP_DIR/$BIN_NAME"

# Elevate only when needed, so the script also works run as root or into a
# directory the user already owns. A directory that does not exist yet is
# judged by its parent, since that is what mkdir has to write into.
writable_target="$INSTALL_DIR"
while [ ! -d "$writable_target" ] && [ "$writable_target" != "/" ] && [ "$writable_target" != "." ]; do
    writable_target=$(dirname "$writable_target")
done

SUDO=""
if [ "$(id -u)" -ne 0 ] && [ ! -w "$writable_target" ] && command -v sudo >/dev/null 2>&1; then
    SUDO="sudo"
fi

echo "Installing to $INSTALL_DIR/$BIN_NAME"
$SUDO mkdir -p "$INSTALL_DIR"
# mv replaces an existing binary, so re-running this upgrades in place.
$SUDO mv -f "$TMP_DIR/$BIN_NAME" "$INSTALL_DIR/$BIN_NAME"
$SUDO chmod +x "$INSTALL_DIR/$BIN_NAME"

echo "Installed $("$INSTALL_DIR/$BIN_NAME" --version 2>/dev/null || echo "$BIN_NAME $VERSION")"
