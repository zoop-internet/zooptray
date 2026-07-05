#!/bin/bash
# Cross-platform packaging script for Zoop One
# Usage: ./build/package.sh [version]
# Requires: wails, nsis (for Windows), dpkg-deb (for Linux)

set -e
VERSION="${1:-1.0.0}"
echo "Building Zoop One v${VERSION}..."

# Step 1: Build the binary for current platform
echo "==> Building binary..."
wails build -platform "$(go env GOOS)/$(go env GOARCH)" -tags webkit2_41

# Step 2: Platform-specific packaging
OS=$(uname -s)

case "$OS" in
  Darwin)
    echo "==> Creating macOS .app bundle..."
    bash build/darwin/create-dmg.sh build/bin/zoop-one
    mv "Zoop-One-macOS.dmg" "build/bin/Zoop-One-${VERSION}-macOS.dmg"
    ;;

  Linux)
    echo "==> Creating Linux .deb package..."
    PKG_ROOT="build/zoop-one_${VERSION}_amd64"
    rm -rf "$PKG_ROOT"
    mkdir -p "$PKG_ROOT/usr/bin"
    mkdir -p "$PKG_ROOT/usr/share/applications"
    mkdir -p "$PKG_ROOT/usr/share/icons/hicolor/256x256/apps"
    mkdir -p "$PKG_ROOT/DEBIAN"

    cp build/bin/zoop-one "$PKG_ROOT/usr/bin/"
    cp linux_installer/zoop-one.desktop "$PKG_ROOT/usr/share/applications/"
    cp build/appicon.png "$PKG_ROOT/usr/share/icons/hicolor/256x256/apps/zoop-one.png"

    cat > "$PKG_ROOT/DEBIAN/control" <<EOF
Package: zoop-one
Version: ${VERSION}
Section: net
Priority: optional
Architecture: amd64
Maintainer: Zoop Network <contact@zoopnetwork.com>
Depends: wireguard, wireguard-tools
Description: Decentralized network sharing node
 Zoop One turns your computer into a secure gateway
 for the Zoop decentralized network.
Homepage: https://zoopnetwork.com
EOF

    cp linux_installer/postinst.sh "$PKG_ROOT/DEBIAN/postinst"
    cp linux_installer/prerm.sh "$PKG_ROOT/DEBIAN/prerm"
    chmod 755 "$PKG_ROOT/DEBIAN/postinst" "$PKG_ROOT/DEBIAN/prerm"

    dpkg-deb --build "$PKG_ROOT"
    mv "build/zoop-one_${VERSION}_amd64.deb" "build/bin/"
    rm -rf "$PKG_ROOT"
    ;;

  MINGW*|MSYS*|CYGWIN*)
    echo "==> Creating Windows installer..."
    cd windows_installer
    makensis -DVERSION="$VERSION" zoop_installer.nsi
    mv "Zoop-One-Setup-${VERSION}.exe" "../build/bin/"
    cd ..
    ;;
esac

echo "==> Done! Package in build/bin/"
