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

# Prefer active health check if nosd binds a known port in smoke images
if command -v curl >/dev/null 2>&1; then
  if curl -fsS --retry 10 --retry-connrefused --retry-delay 2 http://127.0.0.1:18080/health >/dev/null 2>&1; then
    echo "[smoke] nosd health endpoint reachable"
    exit 0
  fi
fi

# Fallback: inspect serial output for startup marker
if grep -Ei "nosd listening|nosd started|Linux version|systemd|Debian GNU/Linux|Welcome to|Starting systemd|initramfs| init:|Kernel command line" "$LOG" >/dev/null 2>&1; then
  echo "[smoke] Boot output detected on serial console:"
  sed -n '1,200p' "$LOG" | sed -e 's/^/[serial] /'
  exit 0
fi

echo "::error::Smoke failed. First 200 lines of serial:"
sed -n '1,200p' "$LOG" || true
echo "[smoke] Last 100 lines (tail):"
tail -n 100 "$LOG" || true
exit 1


