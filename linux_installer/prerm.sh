#!/bin/bash
# Pre-remove script for Linux .deb/.rpm packages

set -e

# Remove desktop file
rm -f /usr/share/applications/zoop-one.desktop
xdg-desktop-menu forceupdate 2>/dev/null || true

echo "Zoop One removed."
