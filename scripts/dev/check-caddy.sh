#!/bin/sh
set -e

echo "[*] Checking Caddy config"
caddy validate --config /etc/caddy/Caddyfile

echo "[*] Sockets"
ss -ltnp | egrep '(:80|:443|:9000)' || true

echo "[*] HTTP -> HTTPS"
code=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1/)
echo "HTTP status: $code (expect 308)"

echo "[*] API JSON"
curl -sk https://127.0.0.1/api/setup/state | head -c 200; echo
