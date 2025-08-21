#!/usr/bin/env bash
set -euo pipefail
ISO="${1:?Usage: $0 <path-to-iso>}"

LOG="/tmp/nos-serial.log"
DISK="/tmp/nos-smoke.qcow2"

# Create a small test disk
qemu-img create -f qcow2 "$DISK" 8G >/dev/null

# Boot headless with serial console captured to $LOG. TCG only (no KVM on GitHub-hosted runners).
# We rely on the ISO to pass console=ttyS0 to show logs here.
timeout 240s qemu-system-x86_64 \
  -accel tcg \
  -m 1024 -smp 2 \
  -no-reboot -no-shutdown \
  -display none \
  -serial file:"$LOG" \
  -drive file="$DISK",if=virtio,format=qcow2 \
  -cdrom "$ISO" \
  -boot d || true

# Basic heuristics that the kernel/systemd actually booted
if grep -E "Linux version|systemd|Debian GNU/Linux" "$LOG" >/dev/null 2>&1; then
  echo "[smoke] Boot output detected on serial console:"
  sed -n '1,120p' "$LOG" | sed -e 's/^/[serial] /'
  exit 0
else
  echo "::error::No boot output detected on serial console. Full log follows:"
  sed -n '1,200p' "$LOG" || true
  exit 1
fi

#!/usr/bin/env bash
set -euo pipefail

ISO_PATH=${1:-packaging/iso/debian/live-image-amd64.hybrid.iso}
QEMU_BIN=${QEMU_BIN:-qemu-system-x86_64}
RAM_MB=${RAM_MB:-2048}
SMP=${SMP:-2}
HOST_PORT=${HOST_PORT:-9443}
LOG=./qemu-smoke.log

if ! command -v "$QEMU_BIN" >/dev/null 2>&1; then
  echo "ERROR: $QEMU_BIN not found" >&2
  exit 2
fi

if [ ! -f "$ISO_PATH" ]; then
  echo "ERROR: ISO not found at $ISO_PATH" >&2
  exit 2
fi

KVM_FLAG=""
if [ -e /dev/kvm ]; then
  KVM_FLAG="-enable-kvm"
fi

echo "[smoke] Starting QEMU from $ISO_PATH (KVM: ${KVM_FLAG:-no})" | tee "$LOG"

# Clean up any previous instance
set +e
pkill -f "${QEMU_BIN}.*${ISO_PATH}" >/dev/null 2>&1
set -e

"$QEMU_BIN" $KVM_FLAG \
  -m "$RAM_MB" -smp "$SMP" -cpu host \
  -cdrom "$ISO_PATH" -boot d \
  -nographic -serial mon:stdio \
  -netdev user,id=n1,hostfwd=tcp::${HOST_PORT}-:443 \
  -device virtio-net-pci,netdev=n1 \
  -device virtio-blk-pci \
  -device virtio-rng \
  -no-reboot 2>&1 | tee -a "$LOG" &
QPID=$!

cleanup() { 
  set +e
  kill "$QPID" >/dev/null 2>&1
  sleep 2
  kill -9 "$QPID" >/dev/null 2>&1
}
trap cleanup EXIT

# Wait for readiness line up to 300s
echo "[smoke] Waiting for readiness signal..." | tee -a "$LOG"
for i in $(seq 1 300); do
  if grep -qE "NithronOS ready|nosd ready" "$LOG"; then
    echo "[smoke] Ready signal detected" | tee -a "$LOG"
    break
  fi
  sleep 1
  if ! kill -0 "$QPID" 2>/dev/null; then
    echo "[smoke] QEMU exited unexpectedly" | tee -a "$LOG"
    exit 1
  fi
  if [ "$i" -eq 300 ]; then
    echo "[smoke] Timeout waiting for readiness" | tee -a "$LOG"
    exit 1
  fi
done

# Curl health endpoint via forwarded port with retries
echo "[smoke] Checking health at https://127.0.0.1:${HOST_PORT}/api/health" | tee -a "$LOG"
ok=0
for i in $(seq 1 30); do
  body=$(curl -sk --max-time 5 https://127.0.0.1:${HOST_PORT}/api/health || true)
  echo "$body" | tee -a "$LOG"
  if echo "$body" | grep -q '"ok":\s*true'; then
    ok=1; break
  fi
  sleep 2
done

if [ "$ok" -ne 1 ]; then
  echo "[smoke] Health check failed after retries" | tee -a "$LOG"
  exit 1
fi

echo "[smoke] Success" | tee -a "$LOG"
exit 0


