#!/bin/bash
# Prepare NithronOS installer files for ISO

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISO_DIR="$SCRIPT_DIR/debian"
WEB_DIR="$SCRIPT_DIR/../../web/dist"

echo "Preparing NithronOS installer files..."

# Copy web UI files to be included in ISO
if [ -d "$WEB_DIR" ]; then
    echo "Copying web UI files..."
    mkdir -p "$ISO_DIR/config/includes.binary/web"
    cp -r "$WEB_DIR"/* "$ISO_DIR/config/includes.binary/web/" 2>/dev/null || true
else
    echo "Warning: Web UI dist not found. Run 'npm run build' in web/ directory first."
fi

# Copy any built .deb packages
DEB_DIR="$SCRIPT_DIR/../deb"
if [ -d "$DEB_DIR" ]; then
    echo "Looking for .deb packages..."
    mkdir -p "$ISO_DIR/config/packages.binary"
    find "$DEB_DIR" -name "*.deb" -exec cp {} "$ISO_DIR/config/packages.binary/" \; 2>/dev/null || true
fi

# Make sure the installer kernel/initrd will be in the right place
# The debian-installer cdrom mode puts files in install.amd/
echo "Configuring installer paths..."
mkdir -p "$ISO_DIR/config/includes.installer"

echo "Installer preparation complete."
echo ""
echo "To build the ISO:"
echo "  cd $ISO_DIR"
echo "  sudo lb clean"
echo "  sudo lb config"
echo "  sudo lb build"
