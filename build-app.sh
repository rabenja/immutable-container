#!/bin/bash
# Build IMF Viewer using Tauri
# Usage:
#   ./build-app.sh              # Production build (signed + notarized on macOS)
#   ./build-app.sh --dev        # Dev mode (hot reload)
#   ./build-app.sh --target <triple>  # Cross-compile
#
# Prerequisites:
#   - Go 1.22+
#   - Rust (via rustup)
#   - Tauri CLI: cargo install tauri-cli

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
SIDECAR_DIR="${SCRIPT_DIR}/src-tauri/sidecar"

# Detect platform
ARCH=$(uname -m)
case "$ARCH" in
    x86_64)  GOARCH="amd64" ;;
    arm64)   GOARCH="arm64" ;;
    aarch64) GOARCH="arm64" ;;
    *)       GOARCH="$ARCH" ;;
esac

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
case "$OS" in
    darwin)  GOOS="darwin" ;;
    mingw*|msys*|cygwin*) GOOS="windows" ;;
    linux)   GOOS="linux" ;;
    *)       GOOS="$OS" ;;
esac

echo "=== IMF Viewer Build ==="
echo "Platform: $GOOS/$GOARCH"
echo ""

# Step 0: Generate macOS .icns icon if missing (requires macOS tools)
if [ "$GOOS" = "darwin" ] && [ ! -f "$SCRIPT_DIR/src-tauri/icons/icon.icns" ]; then
    echo "Step 0: Generating macOS icon..."
    cd "$SCRIPT_DIR/src-tauri/icons"
    mkdir -p icon.iconset
    cp 32x32.png icon.iconset/icon_32x32.png
    cp 128x128.png icon.iconset/icon_128x128.png
    cp "128x128@2x.png" "icon.iconset/icon_128x128@2x.png"
    sips -z 16 16 32x32.png --out icon.iconset/icon_16x16.png 2>/dev/null
    sips -z 256 256 "128x128@2x.png" --out icon.iconset/icon_256x256.png 2>/dev/null
    sips -z 512 512 "128x128@2x.png" --out icon.iconset/icon_512x512.png 2>/dev/null
    iconutil -c icns icon.iconset -o icon.icns
    rm -rf icon.iconset
    echo "  Created icon.icns"
    cd "$SCRIPT_DIR"
fi

# Step 1: Build the Go sidecar binary
echo "Step 1: Building Go sidecar binary..."
mkdir -p "$SIDECAR_DIR"

BINARY_NAME="imf"
if [ "$GOOS" = "windows" ]; then
    BINARY_NAME="imf.exe"
fi

GOOS=$GOOS GOARCH=$GOARCH go build \
    -ldflags="-s -w" \
    -o "$SIDECAR_DIR/$BINARY_NAME" \
    "$SCRIPT_DIR/cmd/imf/"

echo "  Built: $SIDECAR_DIR/$BINARY_NAME"
echo "  Size: $(du -h "$SIDECAR_DIR/$BINARY_NAME" | cut -f1)"

# Step 2: Build with Tauri
echo ""
echo "Step 2: Building Tauri application..."

if [ "$1" = "--dev" ]; then
    echo "  Running in dev mode..."
    cd "$SCRIPT_DIR/src-tauri"
    cargo tauri dev
else
    cd "$SCRIPT_DIR/src-tauri"
    
    if [ "$1" = "--target" ] && [ -n "$2" ]; then
        echo "  Target: $2"
        cargo tauri build --target "$2"
    else
        cargo tauri build
    fi
fi

echo ""
echo "=== Build complete ==="

# Step 3: macOS signing and notarization info
if [ "$GOOS" = "darwin" ] && [ "$1" != "--dev" ]; then
    APP_PATH=$(find "$SCRIPT_DIR/src-tauri/target" -name "IMF Viewer.app" -type d 2>/dev/null | head -1)
    DMG_PATH=$(find "$SCRIPT_DIR/src-tauri/target" -name "*.dmg" 2>/dev/null | head -1)
    
    if [ -n "$APP_PATH" ]; then
        echo ""
        echo "=== macOS Distribution ==="
        echo "App: $APP_PATH"
        
        if [ -n "$DMG_PATH" ]; then
            echo "DMG: $DMG_PATH"
        fi
        
        # Check if signing identity exists
        if security find-identity -v -p codesigning 2>/dev/null | grep -q "3R489H66PC"; then
            echo ""
            echo "Signing identity found. Tauri should have signed automatically."
            echo ""
            echo "To notarize (required for distribution):"
            echo "  xcrun notarytool submit \"$DMG_PATH\" \\"
            echo "    --apple-id YOUR_APPLE_ID \\"
            echo "    --team-id 3R489H66PC \\"
            echo "    --password YOUR_APP_SPECIFIC_PASSWORD \\"
            echo "    --wait"
            echo ""
            echo "  xcrun stapler staple \"$DMG_PATH\""
        else
            echo ""
            echo "WARNING: No signing identity found for team 3R489H66PC."
            echo "Install your Developer ID certificate from developer.apple.com."
        fi
    fi
    
    echo ""
    echo "To install:"
    echo "  1. Open the DMG and drag 'IMF Viewer' to /Applications"
    echo "  2. Double-click any .imf file — it opens in IMF Viewer"
fi
