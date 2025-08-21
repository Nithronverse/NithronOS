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

# Configure build (Debian bookworm, amd64, ISO-hybrid)
DEBIAN_MIRROR="http://deb.debian.org/debian"

# Disable LB security mirrors to avoid legacy bookworm/updates
export LB_MIRROR_CHROOT_SECURITY=""
export LB_MIRROR_BINARY_SECURITY=""
export LB_SECURITY="false"

# Disable live-build's kernel autodetect/linux-image stage
export LB_LINUX_FLAVOURS=""
export LB_LINUX_PACKAGES=""

# --- begin: disable live-build linux-image stage ---
patch_skip_linux_stage() {
  local f="/usr/lib/live/build/lb_chroot_linux-image"
  if [ ! -f "$f" ]; then
    echo "[iso] skip: $f not found"
    return 0
  fi
  # Only patch once
  if grep -q 'nithronos-skip-linux-image' "$f"; then
    echo "[iso] linux-image stage already disabled"
    return 0
  fi
  echo "[iso] disabling live-build linux-image stage (we install kernel via package list)"
  cp -a "$f" "$f.orig" || true
  cat > "$f" <<'SH'
#!/bin/sh
# nithronos-skip-linux-image: live-build kernel autodetect disabled.
# The kernel is installed explicitly via package lists (linux-image-amd64).
echo "[iso] skipping lb_chroot_linux-image (kernel provided by package list)"
exit 0
SH
  chmod +x "$f"
}
patch_skip_linux_stage
# --- end: disable live-build linux-image stage ---

${SUDO_CMD} lb config \
  --mode debian \
  --distribution bookworm \
  --architectures amd64 \
  --binary-images iso-hybrid \
  --apt-recommends true \
  --apt-indices false \
  --debian-installer live \
  --archive-areas "main contrib non-free-firmware" \
  --mirror-bootstrap "$DEBIAN_MIRROR" \
  --mirror-chroot   "$DEBIAN_MIRROR" \
  --mirror-binary   "$DEBIAN_MIRROR"

# Persist kernel skip into profile so lb build picks it up even if envs are sanitized
printf '%s\n' 'LB_LINUX_FLAVOURS=""' 'LB_LINUX_PACKAGES=""' >> "$PROFILE_DIR/config/common"
printf '%s\n' 'LB_LINUX_FLAVOURS=""' 'LB_LINUX_PACKAGES=""' >> "$PROFILE_DIR/config/chroot"

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
echo "[iso] LB_LINUX_FLAVOURS='${LB_LINUX_FLAVOURS}' LB_LINUX_PACKAGES='${LB_LINUX_PACKAGES}' (kernel installed via package list)"
${SUDO_CMD} lb build

# Default output path from live-build
ISO_SRC="live-image-amd64.hybrid.iso"

# Name the ISO nicely
TAG="${GITHUB_REF_NAME:-dev}"
DATE="$(date +%Y%m%d)"
ISO_DST="$OUT_DIR/nithronos-${TAG}-${DATE}-amd64.iso"
[ -f "$ISO_SRC" ] || { echo "::error::ISO not found at $ISO_SRC"; exit 1; }
mv -v "$ISO_SRC" "$ISO_DST"

popd >/dev/null
echo "[iso] built $ISO_DST"


