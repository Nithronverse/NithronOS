#!/bin/sh
set -e

echo "[*] Checking Caddy config"

# Create temporary TLS certificates if they don't exist
if [ ! -f /etc/nithronos/tls/cert.pem ] || [ ! -f /etc/nithronos/tls/key.pem ]; then
    echo "[*] Creating temporary TLS certificates for validation..."
    mkdir -p /etc/nithronos/tls
    openssl req -x509 -nodes -days 1 -newkey rsa:2048 \
        -keyout /etc/nithronos/tls/key.pem \
        -out /etc/nithronos/tls/cert.pem \
        -subj "/CN=localhost" 2>/dev/null || true
fi

caddy validate --config /etc/caddy/Caddyfile

echo "[*] Sockets"
ss -ltnp | egrep '(:80|:443|:9000)' || true

echo "[*] HTTP -> HTTPS"
code=$(curl -s -o /dev/null -w "%{http_code}" http://127.0.0.1/)
echo "HTTP status: $code (expect 308)"

echo "[*] API JSON"
curl -sk https://127.0.0.1/api/setup/state | head -c 200; echo
