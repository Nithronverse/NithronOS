#!/bin/bash
# Fetch Debian Installer kernel and initrd for NithronOS ISO
# Uses Debian bookworm cdrom installer images (text frontend)

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ISO_DIR="$(dirname "$SCRIPT_DIR")"
WORK_DIR="${ISO_DIR}/debian/config/includes.binary"

# Debian mirror and paths
DEBIAN_MIRROR="${DEBIAN_MIRROR:-http://deb.debian.org/debian}"
DI_VERSION="${DI_VERSION:-bookworm}"
DI_ARCH="${DI_ARCH:-amd64}"
# Use cdrom images rather than netboot to avoid network dependency during install
# Example: https://deb.debian.org/debian/dists/bookworm/main/installer-amd64/current/images/cdrom/
DI_BASE_URL="${DEBIAN_MIRROR}/dists/${DI_VERSION}/main/installer-${DI_ARCH}/current/images/cdrom"

# Target paths
INSTALL_DIR="${WORK_DIR}/install.amd"
VMLINUZ_TARGET="${INSTALL_DIR}/vmlinuz"
INITRD_TARGET="${INSTALL_DIR}/initrd.gz"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log_info() {
    echo -e "${GREEN}[INFO]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $1"
}

# Create target directory
log_info "Creating installer directory: ${INSTALL_DIR}"
mkdir -p "${INSTALL_DIR}"

# Function to download with retry
download_file() {
    local url="$1"
    local output="$2"
    local max_retries=3
    local retry=0
    
    while [ $retry -lt $max_retries ]; do
        log_info "Downloading $(basename "$output") (attempt $((retry + 1))/${max_retries})..."
        if wget -q --show-progress -O "$output" "$url"; then
            log_info "Successfully downloaded $(basename "$output")"
            return 0
        else
            retry=$((retry + 1))
            if [ $retry -lt $max_retries ]; then
                log_warn "Download failed, retrying in 5 seconds..."
                sleep 5
            fi
        fi
    done
    
    log_error "Failed to download $url after ${max_retries} attempts"
    return 1
}

# Download kernel
if [ -f "${VMLINUZ_TARGET}" ]; then
    log_info "Kernel already exists at ${VMLINUZ_TARGET}, skipping download"
else
    download_file "${DI_BASE_URL}/vmlinuz" "${VMLINUZ_TARGET}" || exit 1
fi

# Download initrd
if [ -f "${INITRD_TARGET}" ]; then
    log_info "Initrd already exists at ${INITRD_TARGET}, skipping download"
else
    download_file "${DI_BASE_URL}/initrd.gz" "${INITRD_TARGET}" || exit 1
fi

# Verify files exist and are non-empty
for file in "${VMLINUZ_TARGET}" "${INITRD_TARGET}"; do
    if [ ! -f "$file" ]; then
        log_error "File not found: $file"
        exit 1
    fi
    
    size=$(stat -c%s "$file" 2>/dev/null || stat -f%z "$file" 2>/dev/null || echo 0)
    if [ "$size" -eq 0 ]; then
        log_error "File is empty: $file"
        exit 1
    fi
    
    log_info "Verified: $(basename "$file") ($(numfmt --to=iec-i --suffix=B $size))"
done

# Create symlink for UEFI compatibility if needed
if [ ! -e "${WORK_DIR}/install" ]; then
    ln -sf "install.amd" "${WORK_DIR}/install"
    log_info "Created symlink: install -> install.amd"
fi

log_info "Debian Installer files ready for ISO build"
log_info "  Kernel: ${VMLINUZ_TARGET}"
log_info "  Initrd: ${INITRD_TARGET}"

exit 0
