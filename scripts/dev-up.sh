#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)

echo "Starting nosd and web dev servers..."

(
  cd "$ROOT/backend/nosd"
  if command -v air >/dev/null 2>&1; then
    air
  elif command -v reflex >/dev/null 2>&1; then
    reflex -r '\.go$' -- sh -c 'go run ./...'
  else
    go run ./...
  fi
) &

(
  cd "$ROOT/web"
  npm run dev
) &

wait


