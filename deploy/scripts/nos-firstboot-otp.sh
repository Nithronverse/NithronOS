#!/bin/bash
# NithronOS First-Boot OTP Display Script

set -e

OTP_FILE="/var/lib/nos/firstboot-otp"
FIRSTBOOT_FILE="/var/lib/nos/firstboot.json"
TTY_DEVICE="/dev/tty1"
CONSOLE_DEVICE="/dev/console"

# Check if first-boot is pending
if [ ! -f "$FIRSTBOOT_FILE" ]; then
    echo "First-boot already completed, skipping OTP display"
    exit 0
fi

# Check if OTP exists, regenerate if needed
if [ ! -f "$OTP_FILE" ] || [ ! -s "$OTP_FILE" ]; then
    echo "Generating new OTP..."
    # Call nosd to generate OTP
    if command -v nosd >/dev/null 2>&1; then
        nosd generate-otp > "$OTP_FILE" 2>/dev/null || true
    fi
    
    # Fallback: generate random OTP if nosd failed
    if [ ! -s "$OTP_FILE" ]; then
        tr -dc 'A-Za-z0-9' < /dev/urandom | head -c 8 > "$OTP_FILE"
    fi
fi

# Read the OTP
if [ -f "$OTP_FILE" ]; then
    OTP=$(cat "$OTP_FILE")
else
    echo "Failed to read OTP file"
    exit 1
fi

# Get IP addresses
IP_ADDRESSES=$(ip -4 addr show | grep -oP '(?<=inet\s)\d+(\.\d+){3}' | grep -v '127.0.0.1' | head -n 3)
if [ -z "$IP_ADDRESSES" ]; then
    IP_ADDRESSES="No network configured"
else
    IP_ADDRESSES=$(echo "$IP_ADDRESSES" | sed 's/^/  - /')
fi

# Prepare the message
MESSAGE="
╔═══════════════════════════════════════════════════════════════╗
║                  NithronOS First-Boot Setup                   ║
╠═══════════════════════════════════════════════════════════════╣
║                                                                ║
║  Welcome! Please complete the initial setup via web browser:  ║
║                                                                ║
║  Access URLs:                                                  ║
$IP_ADDRESSES
║                                                                ║
║  One-Time Password (OTP): $OTP                            ║
║                                                                ║
║  This OTP expires in 30 minutes and is required for setup.    ║
║                                                                ║
╚═══════════════════════════════════════════════════════════════╝
"

# Display to TTY1
if [ -w "$TTY_DEVICE" ]; then
    echo -e "\n$MESSAGE" > "$TTY_DEVICE"
fi

# Display to console
if [ -w "$CONSOLE_DEVICE" ]; then
    echo -e "\n$MESSAGE" > "$CONSOLE_DEVICE"
fi

# Also use wall to broadcast to all terminals
echo "$MESSAGE" | wall

# Log to journal
echo "First-boot OTP: $OTP"
echo "Access URLs: $IP_ADDRESSES"
logger -t nos-firstboot-otp "OTP displayed: $OTP"

# Create a marker file to indicate OTP was displayed
touch /var/lib/nos/otp-displayed

exit 0
