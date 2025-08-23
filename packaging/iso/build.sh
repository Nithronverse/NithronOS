#!/usr/bin/env bash
set -euo pipefail

# run as root if available; otherwise prefix with sudo
SUDO_CMD=""
if [ "${EUID:-$(id -u)}" -ne 0 ]; then
  SUDO_CMD="sudo"
fi

DEB_DIR="${1:-packaging/iso/local-debs}"

SCRIPT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROFILE_DIR="$SCRIPT_DIR/debian"
OUT_DIR="$SCRIPT_DIR/../../dist/iso"
REPO_ROOT="$(cd "$SCRIPT_DIR/../.." && pwd)"

# --- begin logo wiring ---
LOGO_SRC=""
for p in \
  "$REPO_ROOT/assets/nithronos-logo-mark.png" \
  "$REPO_ROOT/assets/brand/nithronos-logo-mark.png"
do
  if [ -f "$p" ]; then LOGO_SRC="$p"; break; fi
done

if [ -n "$LOGO_SRC" ]; then
  echo "[iso] branding with logo: $LOGO_SRC"

  # Debian Installer graphics
  mkdir -p "$PROFILE_DIR/config/includes.installer/usr/share/graphics"
  # Logo (light + dark)
  cp "$LOGO_SRC" "$PROFILE_DIR/config/includes.installer/usr/share/graphics/logo_debian.png"
  cp "$LOGO_SRC" "$PROFILE_DIR/config/includes.installer/usr/share/graphics/logo_debian_dark.png" || true

  # Splash: try convert to 640x480, fallback to raw
  SPL_OUT="$PROFILE_DIR/config/includes.installer/usr/share/graphics/splash.png"
  if command -v convert >/dev/null 2>&1; then
    convert "$LOGO_SRC" -resize 640x480 -background black -gravity center -extent 640x480 "$SPL_OUT" || cp "$LOGO_SRC" "$SPL_OUT"
  else
    cp "$LOGO_SRC" "$SPL_OUT"
  fi

  # GRUB background (UEFI)
  mkdir -p "$PROFILE_DIR/config/includes.binary/boot/grub"
  cp "$SPL_OUT" "$PROFILE_DIR/config/includes.binary/boot/grub/splash.png" || cp "$LOGO_SRC" "$PROFILE_DIR/config/includes.binary/boot/grub/splash.png"

  # ISOLINUX background (BIOS)
  mkdir -p "$PROFILE_DIR/config/includes.binary/isolinux"
  cp "$SPL_OUT" "$PROFILE_DIR/config/includes.binary/isolinux/splash.png" || cp "$LOGO_SRC" "$PROFILE_DIR/config/includes.binary/isolinux/splash.png"
fi
# --- end logo wiring ---

# Ensure apt includes directories exist for pinned sources
mkdir -p "$PROFILE_DIR/config/includes.chroot/etc/apt" "$PROFILE_DIR/config/includes.binary/etc/apt"
mkdir -p "$PROFILE_DIR/config/archives"
mkdir -p "$PROFILE_DIR/config/hooks/early"

# Force-correct apt sources early inside chroot to avoid bookworm/updates and duplicates
FORCE_SOURCES_HOOK="$PROFILE_DIR/config/hooks/early/00-force-sources.chroot"
cat > "$FORCE_SOURCES_HOOK" <<'EOS'
#!/bin/sh
set -e
cat > /etc/apt/sources.list <<'EOF'
deb http://deb.debian.org/debian bookworm main contrib non-free-firmware
deb http://deb.debian.org/debian bookworm-updates main contrib non-free-firmware
deb http://security.debian.org/debian-security bookworm-security main contrib non-free-firmware
EOF
# Remove any additional lists to prevent duplicates
rm -f /etc/apt/sources.list.d/* 2>/dev/null || true
apt-get update || true
EOS
chmod +x "$FORCE_SOURCES_HOOK"

echo "[iso] using debs from: $DEB_DIR"
mkdir -p "$OUT_DIR"

# Prepare live-build working dir
mkdir -p "$PROFILE_DIR/config/includes.chroot/root/debs"
mkdir -p "$PROFILE_DIR/config/hooks/normal"

# Stage local debs (from CI artifact)
if ls "$DEB_DIR"/*.deb 1>/dev/null 2>&1; then
  cp -v "$DEB_DIR"/*.deb "$PROFILE_DIR/config/includes.chroot/root/debs/" || true
else
  echo "::error::No .deb files found in $DEB_DIR"
  exit 1
fi

# Hook to install staged debs inside chroot during build
HOOK="$PROFILE_DIR/config/hooks/normal/20-install-local-debs.chroot"
cat > "$HOOK" <<'EOS'
#!/bin/sh
set -e
if ls /root/debs/*.deb 1>/dev/null 2>&1; then
  dpkg -i /root/debs/*.deb || apt-get -y -f install
fi
EOS
chmod +x "$HOOK"

# No longer need sed-based fix hooks; sources are pinned via includes and archives config

# Run live-build
pushd "$PROFILE_DIR" >/dev/null

# Clean previous outputs
${SUDO_CMD} lb clean --purge || true
rm -rf chroot/ cache/* || true

# Configure build (Debian bookworm, ISO-hybrid)
DEBIAN_MIRROR="http://deb.debian.org/debian"
ARCH="${ISO_ARCH:-amd64}"

# Disable LB security mirrors to avoid legacy bookworm/updates
export LB_MIRROR_CHROOT_SECURITY=""
export LB_MIRROR_BINARY_SECURITY=""
export LB_SECURITY="false"

# Enable serial for syslinux (BIOS boot menu) so early bootloader output goes to serial
export LB_SYSLINUX_SERIAL="0 115200"

## No kernel stage overrides; rely on package lists (linux-image-amd64) and lb defaults

${SUDO_CMD} lb config \
  --mode debian \
  --distribution bookworm \
  --architectures "${ARCH}" \
  --binary-images iso-hybrid \
  --apt-recommends true \
  --apt-indices false \
  --debian-installer live \
  --archive-areas "main contrib non-free-firmware" \
  --mirror-bootstrap "$DEBIAN_MIRROR" \
  --mirror-chroot   "$DEBIAN_MIRROR" \
  --mirror-binary   "$DEBIAN_MIRROR"

## Do not persist LB_LINUX_* overrides; kernel handled via package lists

# Persist only LB_SYSLINUX_SERIAL into profile (clean any prior entries)
for f in "$PROFILE_DIR/config/common" "$PROFILE_DIR/config/chroot"; do
  [ -f "$f" ] && sed -i -E '/^LB_BOOTAPPEND_(LIVE|INSTALL)=/d; /^LB_SYSLINUX_SERIAL=/d' "$f"
done
printf '%s\n' 'LB_SYSLINUX_SERIAL="0 115200"' >> "$PROFILE_DIR/config/common"
printf '%s\n' 'LB_SYSLINUX_SERIAL="0 115200"' >> "$PROFILE_DIR/config/chroot"

# Remove any stale security lines live-build might inject
sed -i '/security\.debian\.org.*bookworm\/updates/d' "$PROFILE_DIR"/config/* 2>/dev/null || true

# Normalize LB_SECURITY in any persisted live-build config and remove legacy --security flags
sed -i -E 's/^LB_SECURITY=.*$/LB_SECURITY="false"/' "$PROFILE_DIR"/config/* 2>/dev/null || true
sed -i '/--security/d' "$PROFILE_DIR"/config/* 2>/dev/null || true

# Make auto/config executable if it exists to silence warnings
if [ -f "$PROFILE_DIR/auto/config" ]; then chmod +x "$PROFILE_DIR/auto/config"; fi

# Ensure previously persisted invalid LB_SECURITY is corrected in profile files
for f in "$PROFILE_DIR/config/common" "$PROFILE_DIR/config/chroot"; do
  [ -f "$f" ] && sed -i -E 's/^LB_SECURITY=.*$/LB_SECURITY="false"/' "$f"
done

# Build ISO (LB assumes non-interactive)
export DEBIAN_FRONTEND=noninteractive
${SUDO_CMD} lb build

# Default output path from live-build
ISO_SRC="live-image-${ARCH}.hybrid.iso"

# Name the ISO as: NithronOS - <arch> - <tag>.iso
TAG="${ISO_TAG:-${GITHUB_REF_NAME:-dev}}"
ISO_DST="$OUT_DIR/NithronOS - ${ARCH} - ${TAG}.iso"
[ -f "$ISO_SRC" ] || { echo "::error::ISO not found at $ISO_SRC"; exit 1; }
mv -v "$ISO_SRC" "$ISO_DST"

popd >/dev/null
echo "[iso] built $ISO_DST"


