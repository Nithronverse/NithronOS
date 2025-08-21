#!/usr/bin/env bash
set -euo pipefail
if [ "${1:-}" = "" ]; then
  echo "::error::No ISO path passed to qemu-smoke.sh"; exit 1
fi
ISO="$1"

LOG="/tmp/nos-serial.log"
DISK="/tmp/nos-smoke.qcow2"

# Small test disk
qemu-img create -f qcow2 "$DISK" 8G >/dev/null

# Headless boot, capture serial via stdio. TCG only (no KVM on GitHub runners).
# -nographic routes serial + monitor to stdio; we tee it into $LOG.
stdbuf -oL -eL timeout 480s qemu-system-x86_64 \
  -accel tcg \
  -m 2048 -smp 2 \
  -no-reboot -no-shutdown \
  -nographic \
  -drive file="$DISK",if=virtio,format=qcow2 \
  -cdrom "$ISO" \
  -boot d 2>&1 | tee "$LOG" >/dev/null || true

# Did it actually boot? Look for a broad set of common markers on serial.
if grep -Ei "Linux version|systemd|Debian GNU/Linux|Welcome to|Starting systemd|initramfs| init:|Kernel command line" "$LOG" >/dev/null 2>&1; then
  echo "[smoke] Boot output detected on serial console:"
  sed -n '1,200p' "$LOG" | sed -e 's/^/[serial] /'
  exit 0
else
  echo "::error::No boot output detected on serial console. First 200 lines:"
  sed -n '1,200p' "$LOG" || true
  echo "[smoke] Last 100 lines (tail):"
  tail -n 100 "$LOG" || true
  exit 1
fi


