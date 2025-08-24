#!/usr/bin/env bash
set -euo pipefail

# Validate Caddy configuration and show listeners on :80/:443

if ! command -v caddy >/dev/null 2>&1; then
  echo "caddy not found on PATH" >&2
  exit 1
fi

echo ">> caddy validate --config /etc/caddy/Caddyfile"
caddy validate --config /etc/caddy/Caddyfile || exit $?

echo ">> listening sockets (:80/:443)"
if command -v ss >/dev/null 2>&1; then
  ss -ltnp | egrep '(:80|:443)' || true
else
  echo "ss not available" >&2
fi


