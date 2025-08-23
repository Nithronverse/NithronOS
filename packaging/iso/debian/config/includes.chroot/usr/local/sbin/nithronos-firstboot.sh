#!/bin/bash
set -euo pipefail

useradd -r -s /usr/sbin/nologin -d /var/lib/nos nos || true

systemctl enable --now caddy || true
systemctl enable --now nftables || true
systemctl enable --now fail2ban || true
systemctl enable --now nosd || true
systemctl enable --now nos-agent || true

IP=$(hostname -I | awk '{print $1}')
OTP=$(head -c 12 /dev/urandom | base64 | tr -dc A-Za-z0-9 | head -c 8)
echo "$OTP" > /etc/nos/first-run-otp
cat >/etc/issue <<EOF
Welcome to NithronOS
Access the web UI at: https://$IP/
One-time setup OTP: $OTP
EOF
echo "NithronOS ready. UI: https://$IP/ OTP: $OTP"

