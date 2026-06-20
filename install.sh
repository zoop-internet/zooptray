#!/bin/bash
set -e

BIN_NAME="zoop-one"
SOURCE_BIN="./build/bin/$BIN_NAME"
ICON_SOURCE="./zoopicon.png"

TARGET_BIN_DIR="$HOME/.local/bin"
TARGET_BIN="$TARGET_BIN_DIR/$BIN_NAME"
TARGET_ICON_DIR="$HOME/.local/share/icons"
TARGET_ICON="$TARGET_ICON_DIR/zoop-one.png"
TARGET_DESKTOP_DIR="$HOME/.local/share/applications"
TARGET_DESKTOP="$TARGET_DESKTOP_DIR/zoop-one.desktop"

echo "=== Zoop One Linux Installer ==="

# 1. Build the binary
echo "Checking for system tray dependency (libayatana-appindicator3-dev)..."
BUILD_TAGS="webkit2_41"
if pkg-config --exists ayatana-appindicator3-0.1 2>/dev/null || pkg-config --exists appindicator3-0.1 2>/dev/null; then
    echo "Found appindicator headers! Enabling System Tray support."
    BUILD_TAGS="webkit2_41 tray"
else
    echo "Appindicator headers NOT found. System tray will run in window-only mode."
    echo "To enable the system tray, please run: sudo apt install libayatana-appindicator3-dev"
    echo "Then re-run this script."
    echo ""
fi

echo "Building Zoop One binary with tags: $BUILD_TAGS..."
wails build -tags "$BUILD_TAGS" -clean

# 2. Create directories
mkdir -p "$TARGET_BIN_DIR"
mkdir -p "$TARGET_ICON_DIR"
mkdir -p "$TARGET_DESKTOP_DIR"

# 3. Copy binary and icon
echo "Installing binary to $TARGET_BIN..."
cp "$SOURCE_BIN" "$TARGET_BIN"
echo "Installing icon to $TARGET_ICON..."
cp "$ICON_SOURCE" "$TARGET_ICON"

# 4. Generate .desktop file
echo "Generating desktop launcher at $TARGET_DESKTOP..."
cat > "$TARGET_DESKTOP" <<EOF
[Desktop Entry]
Name=Zoop One
Comment=Zoop Network Desktop Node Daemon
Exec=$TARGET_BIN
Icon=$TARGET_ICON
Terminal=false
Type=Application
Categories=Network;Utility;
StartupWMClass=zoop-one
EOF

chmod +x "$TARGET_DESKTOP"

echo "=== Installation complete ==="
echo "You can now launch 'Zoop One' directly from your application launcher or menu with its icon!"
