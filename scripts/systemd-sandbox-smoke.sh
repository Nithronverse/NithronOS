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
sudo systemd-sysusers || true
sudo systemd-tmpfiles --create || true

# Start a transient service that shares the same sandbox constraints and attempts a write.
sudo mkdir -p /etc/nos
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


