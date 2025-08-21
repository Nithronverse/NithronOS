#!/usr/bin/env bash
set -euo pipefail

root_dir="$(cd "$(dirname "$0")/.." && pwd)"
cd "$root_dir"

if ! command -v npx >/dev/null 2>&1; then
  echo "npx is required (npm)." >&2
  exit 1
fi

npx openapi-typescript docs/api/openapi.yaml -o web/src/types/api.d.ts

echo "Generated web/src/types/api.d.ts from docs/api/openapi.yaml"


