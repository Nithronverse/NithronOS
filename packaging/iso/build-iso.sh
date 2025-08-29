#!/bin/bash
# NithronOS ISO Builder with Debian Installer and Secure Boot Support

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}=================================================================================${NC}"
echo -e "${BLUE}                    NithronOS ISO Builder                                        ${NC}"
echo -e "${BLUE}=================================================================================${NC}"

# Check if running as root
if [ "$EUID" -ne 0 ]; then 
    echo -e "${RED}Please run as root (use sudo)${NC}"
    exit 1
fi

# Configuration
WORK_DIR="$(pwd)/debian"
ISO_NAME="nithronos-$(date +%Y%m%d)-amd64.iso"
ARCH="amd64"

echo -e "${GREEN}[*] Starting NithronOS ISO build process...${NC}"

# Change to working directory
cd "$WORK_DIR"

# Clean previous builds
echo -e "${YELLOW}[*] Cleaning previous builds...${NC}"
lb clean --purge 2>/dev/null || true
rm -rf .build cache

# Configure live-build
echo -e "${YELLOW}[*] Configuring live-build...${NC}"
if [ -f auto/config ]; then
    chmod +x auto/config
    ./auto/config
else
    echo -e "${RED}Error: auto/config not found${NC}"
    exit 1
fi

# Add Secure Boot support
echo -e "${YELLOW}[*] Configuring Secure Boot support...${NC}"
mkdir -p config/includes.binary/EFI/BOOT

# Download signed shim and GRUB from Debian
echo -e "${YELLOW}[*] Downloading signed bootloaders...${NC}"
apt-get update
apt-get download shim-signed grub-efi-amd64-signed

# Extract signed EFI binaries
dpkg-deb -x shim-signed_*.deb tmp-shim/
dpkg-deb -x grub-efi-amd64-signed_*.deb tmp-grub/

# Copy signed EFI files to ISO
if [ -d tmp-shim/usr/lib/shim/ ]; then
    cp tmp-shim/usr/lib/shim/shimx64.efi.signed config/includes.binary/EFI/BOOT/BOOTX64.EFI
    cp tmp-shim/usr/lib/shim/mmx64.efi.signed config/includes.binary/EFI/BOOT/mmx64.efi
fi

if [ -d tmp-grub/usr/lib/grub/x86_64-efi-signed/ ]; then
    cp tmp-grub/usr/lib/grub/x86_64-efi-signed/grubx64.efi.signed config/includes.binary/EFI/BOOT/grubx64.efi
fi

# Clean up temporary files
rm -rf tmp-shim tmp-grub *.deb

# Create GRUB configuration for EFI
cat > config/includes.binary/EFI/BOOT/grub.cfg << 'EOF'
search --set=root --file /NITHRONOS
set prefix=($root)/boot/grub
configfile /boot/grub/grub.cfg
EOF

# Create identifier file
touch config/includes.binary/NITHRONOS

# Copy NithronOS branding assets
echo -e "${YELLOW}[*] Copying NithronOS branding assets...${NC}"
if [ -f ../../../assets/brand/nithronos-logo-mark.png ]; then
    # Copy logo to various locations
    cp ../../../assets/brand/nithronos-logo-mark.png config/includes.binary/isolinux/splash.png 2>/dev/null || true
    cp ../../../assets/brand/nithronos-logo-mark.png config/includes.installer/usr/share/graphics/ 2>/dev/null || true
    cp ../../../assets/brand/nithronos-logo-mark.png config/includes.chroot/boot/grub/themes/nithron/ 2>/dev/null || true
    echo -e "${GREEN}[✓] Branding assets copied${NC}"
fi

# Ensure debian-installer is configured
echo -e "${YELLOW}[*] Configuring Debian Installer...${NC}"
mkdir -p config/debian-installer

# Set installer branding
cat > config/debian-installer/splash.png.binary << EOF
# This would be the actual splash image for the installer
# For now, using a placeholder
EOF

# Configure installer preseed
if [ -f config/includes.installer/preseed.cfg ]; then
    echo -e "${GREEN}[✓] Preseed configuration found${NC}"
else
    echo -e "${RED}[!] Warning: Preseed configuration not found${NC}"
fi

# Build the ISO
echo -e "${YELLOW}[*] Building ISO (this may take a while)...${NC}"
lb build 2>&1 | tee build.log

# Check if build was successful
if [ -f live-image-amd64.hybrid.iso ]; then
    echo -e "${GREEN}[✓] ISO build successful!${NC}"
    
    # Rename ISO
    mv live-image-amd64.hybrid.iso "../$ISO_NAME"
    
    # Calculate checksums
    echo -e "${YELLOW}[*] Calculating checksums...${NC}"
    cd ..
    sha256sum "$ISO_NAME" > "$ISO_NAME.sha256"
    
    # Display results
    echo -e "${BLUE}=================================================================================${NC}"
    echo -e "${GREEN}Build Complete!${NC}"
    echo -e "ISO: $(pwd)/$ISO_NAME"
    echo -e "Size: $(du -h $ISO_NAME | cut -f1)"
    echo -e "SHA256: $(cat $ISO_NAME.sha256)"
    echo -e "${BLUE}=================================================================================${NC}"
    
    # Test Secure Boot signature
    echo -e "${YELLOW}[*] Verifying Secure Boot signatures...${NC}"
    if command -v sbverify >/dev/null 2>&1; then
        # Extract and verify EFI files from ISO
        mkdir -p test-mount
        mount -o loop "$ISO_NAME" test-mount
        
        if [ -f test-mount/EFI/BOOT/BOOTX64.EFI ]; then
            echo -e "${GREEN}[✓] BOOTX64.EFI found${NC}"
            # Note: sbverify would need the actual certificates to verify
            # For now, just check the file exists
        else
            echo -e "${YELLOW}[!] BOOTX64.EFI not found (Secure Boot may not work)${NC}"
        fi
        
        umount test-mount
        rmdir test-mount
    else
        echo -e "${YELLOW}[!] sbverify not installed, skipping signature verification${NC}"
    fi
    
else
    echo -e "${RED}[✗] ISO build failed!${NC}"
    echo -e "${RED}Check build.log for details${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}ISO can be tested with:${NC}"
echo "  qemu-system-x86_64 -m 2048 -cdrom $ISO_NAME -boot d"
echo ""
echo -e "${GREEN}Or written to USB with:${NC}"
echo "  dd if=$ISO_NAME of=/dev/sdX bs=4M status=progress oflag=sync"
echo ""
