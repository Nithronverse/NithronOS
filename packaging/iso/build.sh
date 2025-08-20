#!/usr/bin/env bash
set -euo pipefail

DEB_DIR="${1:-packaging/iso/local-debs}"

SCRIPT_DIR="$(cd -- "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROFILE_DIR="$SCRIPT_DIR/debian"
OUT_DIR="$SCRIPT_DIR/../../dist/iso"

# Ensure apt includes directories exist for pinned sources
mkdir -p "$PROFILE_DIR/config/includes.chroot/etc/apt" "$PROFILE_DIR/config/includes.binary/etc/apt"

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

# Hook to correct Debian security suite from bookworm/updates -> bookworm-security
FIXSEC_HOOK="$PROFILE_DIR/config/hooks/normal/05-fix-security-sources.chroot"
cat > "$FIXSEC_HOOK" <<'EOS'
#!/bin/sh
set -e
if command -v sed >/dev/null 2>&1; then
  [ -f /etc/apt/sources.list ] && sed -ri 's/bookworm\/updates/bookworm-security/g' /etc/apt/sources.list || true
  if [ -d /etc/apt/sources.list.d ]; then
    find /etc/apt/sources.list.d -maxdepth 1 -type f -print0 2>/dev/null | xargs -0 -r sed -ri 's/bookworm\/updates/bookworm-security/g' || true
  fi
fi
apt-get update || true
EOS
chmod +x "$FIXSEC_HOOK"

# Run live-build
pushd "$PROFILE_DIR" >/dev/null

# Clean previous outputs
lb clean || true
rm -rf cache/* || true

# Configure build (Debian bookworm, amd64, ISO-hybrid)
DEBIAN_MIRROR="http://deb.debian.org/debian"
DEBIAN_SECURITY_MIRROR="http://security.debian.org/debian-security"
lb config \
  --mode debian \
  --distribution bookworm \
  --architectures amd64 \
  --binary-images iso-hybrid \
  --apt-recommends true \
  --debian-installer live \
  --archive-areas "main contrib non-free-firmware" \
  --mirror-bootstrap "$DEBIAN_MIRROR" \
  --mirror-chroot   "$DEBIAN_MIRROR" \
  --mirror-binary   "$DEBIAN_MIRROR" \
  --mirror-chroot-security "$DEBIAN_SECURITY_MIRROR" \
  --mirror-binary-security "$DEBIAN_SECURITY_MIRROR"

# Build ISO (LB assumes non-interactive)
export DEBIAN_FRONTEND=noninteractive
lb build

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


