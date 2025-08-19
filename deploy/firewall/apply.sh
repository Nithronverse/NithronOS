#!/usr/bin/env bash
set -euo pipefail

RULES=${1:-$(dirname "$0")/nos.nft}

if ! command -v nft >/dev/null 2>&1; then
  echo "nft not found; please install nftables" >&2
  exit 1
fi

echo "Applying nftables rules from $RULES"
nft -f "$RULES"

echo "Saving active rules to /etc/nftables.conf"
mkdir -p /etc
nft list ruleset > /etc/nftables.conf

echo "Done. Enable persistence with: systemctl enable --now nftables.service"


