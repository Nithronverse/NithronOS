#!/bin/bash
set -euo pipefail

echo "Waiting for :443..."
for i in {1..60}; do
  if ss -ltn 2>/dev/null | grep -q ":443"; then break; fi
  sleep 1
done

echo "Curl https://127.0.0.1/"
curl -skI https://127.0.0.1/ | head -n1

echo "GET /api/setup/state via https"
curl -sk https://127.0.0.1/api/setup/state || true


