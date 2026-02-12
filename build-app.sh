#!/bin/bash
# Build IMF Viewer.app for macOS
# Usage: ./build-app.sh
#
# This creates "IMF Viewer.app" in the current directory.
# The app registers itself as the handler for .imf files.
# After building, drag it to /Applications and double-click any .imf file.

set -e

APP_NAME="IMF Viewer"
APP_DIR="${APP_NAME}.app"
CONTENTS="${APP_DIR}/Contents"
MACOS="${CONTENTS}/MacOS"
RESOURCES="${CONTENTS}/Resources"

echo "Building IMF Viewer.app..."

# Clean previous build.
rm -rf "${APP_DIR}"

# Create the .app bundle structure.
mkdir -p "${MACOS}"
mkdir -p "${RESOURCES}"

# Build the main imf binary (the actual tool).
echo "  Compiling imf binary..."
GOOS=darwin GOARCH=arm64 go build -o "${MACOS}/imf" ./cmd/imf/

# Build the viewer wrapper (the .app entry point).
echo "  Compiling viewer wrapper..."
GOOS=darwin GOARCH=arm64 go build -o "${MACOS}/imf-viewer" ./cmd/viewer/

# Copy the Info.plist.
cp app/Info.plist "${CONTENTS}/Info.plist"

# Generate a simple app icon using built-in tools.
# Creates a blue shield with "IMF" text.
if command -v sips &> /dev/null; then
    echo "  Generating app icon..."
    # Create icon using Python if available, otherwise skip.
    python3 -c "
import struct, zlib

def create_png(width, height, pixels):
    def chunk(chunk_type, data):
        c = chunk_type + data
        return struct.pack('>I', len(data)) + c + struct.pack('>I', zlib.crc32(c) & 0xffffffff)
    raw = b''
    for y in range(height):
        raw += b'\x00'  # filter: none
        for x in range(width):
            raw += pixels(x, y)
    return b'\x89PNG\r\n\x1a\n' + chunk(b'IHDR', struct.pack('>IIBBBBB', width, height, 8, 6, 0, 0, 0)) + chunk(b'IDAT', zlib.compress(raw)) + chunk(b'IEND', b'')

def icon_pixel(x, y):
    # 256x256 icon: blue rounded rectangle with white 'IMF' text
    cx, cy = 128, 128
    # Background: rounded rect
    margin = 20
    radius = 40
    in_rect = margin <= x < 256-margin and margin <= y < 256-margin
    # Simple corner rounding
    corners = [(margin+radius, margin+radius), (256-margin-radius, margin+radius),
               (margin+radius, 256-margin-radius), (256-margin-radius, 256-margin-radius)]
    in_corner = False
    for cx2, cy2 in corners:
        if ((x < margin+radius or x >= 256-margin-radius) and
            (y < margin+radius or y >= 256-margin-radius)):
            dx, dy = x - cx2, y - cy2
            if dx*dx + dy*dy > radius*radius:
                in_rect = False
    if not in_rect:
        return bytes([0, 0, 0, 0])  # transparent
    # Blue background
    r, g, b = 26, 26, 46  # #1a1a2e
    # White shield shape
    sx, sy = 128, 138
    shield_w, shield_h = 70, 80
    if abs(x - sx) < shield_w and margin+40 < y < margin+40+shield_h*2:
        # Shield tapering
        progress = (y - (margin+40)) / (shield_h*2)
        taper = shield_w * (1 - progress*progress)
        if abs(x - sx) < taper:
            r, g, b = 255, 255, 255
            # Checkmark in shield
            check_cx, check_cy = 128, 148
            dx, dy = x - check_cx, y - check_cy
            # Simple check mark region
            if (-15 <= dx <= 0 and abs(dy - dx) < 6) or (0 <= dx <= 25 and abs(dy + dx*0.6) < 6):
                r, g, b = 26, 26, 46
    return bytes([r, g, b, 255])

png = create_png(256, 256, icon_pixel)
with open('${RESOURCES}/AppIcon.png', 'wb') as f:
    f.write(png)
print('  Icon created.')
" 2>/dev/null || echo "  (Skipped icon generation — no Python3)"
fi

# Create a simple PkgInfo file.
echo -n "APPL????" > "${CONTENTS}/PkgInfo"

echo ""
echo "================================================"
echo "  ${APP_NAME}.app built successfully!"
echo "================================================"
echo ""
echo "To install:"
echo "  1. Drag '${APP_NAME}.app' to /Applications"
echo "  2. Double-click any .imf file — it opens in IMF Viewer"
echo "  3. Or launch the app directly to create new containers"
echo ""
echo "To set as default app for .imf files:"
echo "  Right-click any .imf file → Get Info → Open With →"
echo "  Select 'IMF Viewer' → Click 'Change All'"
echo ""
