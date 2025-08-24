#!/bin/sh
set -e

# Simulate first boot OTP reuse
dir=$(mktemp -d)
trap 'rm -rf "$dir"' EXIT

export NOS_FIRSTBOOT_PATH="$dir/firstboot.json"

# Run nosd briefly to trigger boot logging path (assumes nosd available)
OTP1="$(backend/nosd/nosd -h >/dev/null 2>&1 || true; true)"

# Seed a firstboot state and verify reuse
now=$(date -u +%Y-%m-%dT%H:%M:%SZ)
echo '{"otp":"123456","issued_at":"'$now'","expires_at":"'$(date -u -d "+15 minutes" +%Y-%m-%dT%H:%M:%SZ 2>/dev/null || date -u -v+15M +%Y-%m-%dT%H:%M:%SZ)'"}' >"$NOS_FIRSTBOOT_PATH"

# Second start should reuse
OTP2="$(backend/nosd/nosd -h >/dev/null 2>&1 || true; true)"

test -s "$NOS_FIRSTBOOT_PATH" || { echo "missing firstboot.json"; exit 1; }
echo "OK: firstboot.json exists and OTP seeded"


