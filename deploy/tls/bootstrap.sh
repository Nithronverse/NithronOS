#!/usr/bin/env bash
set -euo pipefail

TLS_DIR=${TLS_DIR:-/etc/nos/tls}
CN=${CN:-nos.local}

mkdir -p "$TLS_DIR"

if ! command -v openssl >/dev/null 2>&1; then
  echo "openssl not found; please install it" >&2
  exit 1
fi

openssl req -x509 -nodes -days 365 -newkey rsa:2048 \
  -keyout "$TLS_DIR/key.pem" \
  -out "$TLS_DIR/cert.pem" \
  -subj "/CN=$CN" \
  -addext "subjectAltName=DNS:$CN,IP:127.0.0.1"

chmod 600 "$TLS_DIR/key.pem"
chmod 644 "$TLS_DIR/cert.pem"
echo "Wrote self-signed certs to $TLS_DIR (CN=$CN)"


