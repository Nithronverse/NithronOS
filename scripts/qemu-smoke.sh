#!/usr/bin/env bash
set -euo pipefail
ISO="${1:?Usage: $0 <path-to-iso>}"

LOG="/tmp/nos-serial.log"
DISK="/tmp/nos-smoke.qcow2"

# Small test disk
qemu-img create -f qcow2 "$DISK" 8G >/dev/null

# Headless boot, capture serial. TCG only (no KVM on GitHub runners).
# Requires the ISO to use console=ttyS0,115200n8 (we set this in build.sh).
timeout 300s qemu-system-x86_64 \
  -accel tcg \
  -m 1024 -smp 2 \
  -no-reboot -no-shutdown \
  -display none \
  -serial file:"$LOG" \
  -drive file="$DISK",if=virtio,format=qcow2 \
  -cdrom "$ISO" \
  -boot d || true

# Did it actually boot?
if grep -E "Linux version|systemd|Debian GNU/Linux" "$LOG" >/dev/null 2>&1; then
  echo "[smoke] Boot output detected on serial console:"
  sed -n '1,120p' "$LOG" | sed -e 's/^/[serial] /'
  exit 0
else
  echo "::error::No boot output detected on serial console. First 200 lines:"
  sed -n '1,200p' "$LOG" || true
  exit 1
fi


