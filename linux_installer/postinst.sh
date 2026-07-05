#!/bin/bash
# Post-install script for Linux .deb/.rpm packages

set -e

# Copy desktop file
cp /usr/share/applications/zoop-one.desktop /usr/share/applications/ 2>/dev/null || true
xdg-desktop-menu forceupdate 2>/dev/null || true

echo "Zoop One installed successfully."
echo "Run 'zoop-one' from terminal or launch from your application menu."
