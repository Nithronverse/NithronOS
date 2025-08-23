#!/usr/bin/env bash
set -euo pipefail

# This smoke test runs on CI (Linux) and verifies that nosd unit allows writes
# to /etc/nos under ProtectSystem via ReadWritePaths.

if ! command -v systemctl >/dev/null 2>&1; then
  echo "systemctl not available; skipping smoke"
  exit 0
fi

sudo install -D -m0644 deploy/systemd/nosd.service /etc/systemd/system/nosd.service
sudo systemctl daemon-reload

# Ensure service user/group and writable dirs exist on CI hosts
if ! getent passwd nosd >/dev/null 2>&1; then
  if command -v useradd >/dev/null 2>&1; then
    sudo useradd -r -M -s /usr/sbin/nologin nosd || true
  elif command -v adduser >/dev/null 2>&1; then
    sudo adduser --system --no-create-home --shell /usr/sbin/nologin nosd || true
  fi
fi
if ! getent group nosd >/dev/null 2>&1; then
  sudo groupadd -r nosd || true
fi
sudo usermod -g nosd nosd || true

sudo install -d -m0750 -o nosd -g nosd /etc/nos
sudo install -d -m0750 -o nosd -g nosd /var/lib/nos

# Start a transient service that shares the same sandbox constraints and attempts a write.
TMP_FILE="/etc/nos/_ci_sandbox_test"
sudo systemd-run --unit=nosd-smoke --property=User=nosd --property=Group=nosd \
  --property=ProtectSystem=strict \
  --property=ReadWritePaths="/etc/nos /var/lib/nos /run" \
  --property=NoNewPrivileges=yes \
  bash -lc "echo ok | tee ${TMP_FILE} >/dev/null"

sleep 1
if ! sudo test -f "${TMP_FILE}"; then
  echo "::error::failed to write under /etc/nos with sandbox"
  exit 1
fi
sudo rm -f "${TMP_FILE}"
echo "sandbox smoke: ok"


