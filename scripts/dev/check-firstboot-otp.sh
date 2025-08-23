#!/usr/bin/env bash
set -euo pipefail

unit=${1:-nosd}

echo "Checking journal for First-boot OTP messages (unit=$unit)"
journalctl -u "$unit" -b --no-pager | grep -E "First-boot OTP:" || {
  echo "No OTP message found in current boot for unit $unit" >&2
  exit 1
}


