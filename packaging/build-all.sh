#!/usr/bin/env bash
set -euo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
OUT="$ROOT/output"
mkdir -p "$OUT"

build_pkg() {
  local dir="$1"
  echo "Building $dir..."
  (cd "$dir" && dpkg-buildpackage -us -uc -b)
}

build_pkg "$ROOT/deb/nosd"
build_pkg "$ROOT/deb/nos-agent"
build_pkg "$ROOT/deb/nos-web"
build_pkg "$ROOT/deb/nithronos"

echo "Collecting .deb artifacts into $OUT"
find "$ROOT" -maxdepth 2 -type f -name "*.deb" -exec mv -v {} "$OUT" \;

echo "Done. Artifacts in $OUT"


