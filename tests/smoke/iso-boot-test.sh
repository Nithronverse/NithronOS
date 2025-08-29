#!/bin/bash
set -euo pipefail

# ISO Boot Smoke Test
# Tests that the ISO boots correctly and services are running

ISO_PATH="${1:-}"
if [ -z "$ISO_PATH" ]; then
    echo "Usage: $0 <iso-path>"
    exit 1
fi

if [ ! -f "$ISO_PATH" ]; then
    echo "Error: ISO not found at $ISO_PATH"
    exit 1
fi

# Configuration
VM_NAME="nithronos-smoke-$$"
VM_DISK="/tmp/${VM_NAME}.qcow2"
VM_PORT_SSH=2222
VM_PORT_HTTP=8080
VM_PORT_API=9000
TIMEOUT=90
LOG_DIR="tests/smoke/logs"

# Cleanup function
cleanup() {
    echo "Cleaning up..."
    if [ -n "${QEMU_PID:-}" ]; then
        kill "$QEMU_PID" 2>/dev/null || true
    fi
    rm -f "$VM_DISK"
}
trap cleanup EXIT

# Create log directory
mkdir -p "$LOG_DIR"

# Create a temporary disk
echo "Creating temporary disk..."
qemu-img create -f qcow2 "$VM_DISK" 20G

# Start QEMU VM
echo "Starting VM..."
qemu-system-x86_64 \
    -name "$VM_NAME" \
    -machine pc,accel=kvm \
    -cpu host \
    -m 2048 \
    -smp 2 \
    -drive file="$ISO_PATH",media=cdrom,readonly=on \
    -drive file="$VM_DISK",if=virtio \
    -netdev user,id=net0,hostfwd=tcp::${VM_PORT_SSH}-:22,hostfwd=tcp::${VM_PORT_HTTP}-:80,hostfwd=tcp::${VM_PORT_API}-:9000 \
    -device e1000,netdev=net0 \
    -display none \
    -serial stdio \
    -monitor tcp:127.0.0.1:4444,server,nowait \
    > "$LOG_DIR/qemu.log" 2>&1 &

QEMU_PID=$!

echo "VM PID: $QEMU_PID"
echo "Waiting for boot..."

# Function to check if service is ready
check_ready() {
    local start_time=$(date +%s)
    local current_time
    local elapsed
    
    while true; do
        current_time=$(date +%s)
        elapsed=$((current_time - start_time))
        
        if [ $elapsed -gt $TIMEOUT ]; then
            echo "Timeout waiting for system to be ready"
            return 1
        fi
        
        # Check for boot banner in serial output
        if grep -q "NithronOS ready" "$LOG_DIR/qemu.log" 2>/dev/null; then
            echo "✓ Boot banner detected"
            break
        fi
        
        sleep 2
    done
    
    # Wait a bit more for services to fully start
    sleep 5
    return 0
}

# Function to extract OTP from logs
extract_otp() {
    # Try to get OTP from serial console
    local otp=$(grep -oE 'OTP: [A-Z0-9]{6}' "$LOG_DIR/qemu.log" | tail -1 | cut -d' ' -f2)
    
    if [ -z "$otp" ]; then
        # Try via SSH if available
        echo "Attempting to get OTP via monitor..."
        echo "sendkey ctrl-alt-f2" | nc -N 127.0.0.1 4444
        sleep 2
        echo "sendkey ret" | nc -N 127.0.0.1 4444
        sleep 1
        
        # Send journalctl command
        for char in j o u r n a l c t l space minus u space n o s d space bar g r e p space O T P; do
            echo "sendkey $char" | nc -N 127.0.0.1 4444
            sleep 0.1
        done
        echo "sendkey ret" | nc -N 127.0.0.1 4444
        sleep 2
        
        otp=$(grep -oE 'OTP: [A-Z0-9]{6}' "$LOG_DIR/qemu.log" | tail -1 | cut -d' ' -f2)
    fi
    
    echo "$otp"
}

# Wait for system to be ready
if ! check_ready; then
    echo "System failed to boot within timeout"
    cat "$LOG_DIR/qemu.log"
    exit 1
fi

echo "System booted successfully"

# Test 1: Check nosd is listening
echo -n "Checking nosd API..."
if curl -s -f "http://localhost:${VM_PORT_API}/api/v1/health" > /dev/null 2>&1; then
    echo " ✓"
else
    echo " ✗"
    echo "nosd API not responding"
    exit 1
fi

# Test 2: Check Caddy is serving
echo -n "Checking Caddy web server..."
if curl -s -f "http://localhost:${VM_PORT_HTTP}/" > /dev/null 2>&1; then
    echo " ✓"
else
    echo " ✗"
    echo "Caddy not responding"
    exit 1
fi

# Test 3: Check setup endpoint
echo -n "Checking setup state..."
SETUP_STATE=$(curl -s "http://localhost:${VM_PORT_HTTP}/api/setup/state" | jq -r '.state' 2>/dev/null || echo "error")
if [ "$SETUP_STATE" = "pending" ] || [ "$SETUP_STATE" = "otp" ]; then
    echo " ✓ (state: $SETUP_STATE)"
else
    echo " ✗ (unexpected state: $SETUP_STATE)"
    exit 1
fi

# Test 4: Extract and verify OTP
echo "Extracting OTP..."
OTP=$(extract_otp)
if [ -z "$OTP" ]; then
    echo "Failed to extract OTP"
    exit 1
fi
echo "OTP found: $OTP"

# Test 5: Complete setup flow
echo "Completing setup..."

# Verify OTP
echo -n "  Verifying OTP..."
VERIFY_RESPONSE=$(curl -s -X POST "http://localhost:${VM_PORT_HTTP}/api/setup/verify-otp" \
    -H "Content-Type: application/json" \
    -d "{\"otp\": \"$OTP\"}")

TOKEN=$(echo "$VERIFY_RESPONSE" | jq -r '.token' 2>/dev/null)
if [ -z "$TOKEN" ] || [ "$TOKEN" = "null" ]; then
    echo " ✗"
    echo "Failed to verify OTP: $VERIFY_RESPONSE"
    exit 1
fi
echo " ✓"

# Create admin user
echo -n "  Creating admin user..."
ADMIN_RESPONSE=$(curl -s -X POST "http://localhost:${VM_PORT_HTTP}/api/setup/create-admin" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TOKEN" \
    -d '{
        "username": "admin",
        "password": "TestAdmin123!",
        "email": "admin@test.local"
    }')

JWT=$(echo "$ADMIN_RESPONSE" | jq -r '.token' 2>/dev/null)
if [ -z "$JWT" ] || [ "$JWT" = "null" ]; then
    echo " ✗"
    echo "Failed to create admin: $ADMIN_RESPONSE"
    exit 1
fi
echo " ✓"

# Test 6: Verify authenticated access
echo -n "Testing authenticated API access..."
ME_RESPONSE=$(curl -s "http://localhost:${VM_PORT_API}/api/v1/me" \
    -H "Authorization: Bearer $JWT")

USERNAME=$(echo "$ME_RESPONSE" | jq -r '.username' 2>/dev/null)
if [ "$USERNAME" = "admin" ]; then
    echo " ✓"
else
    echo " ✗"
    echo "Failed to get user info: $ME_RESPONSE"
    exit 1
fi

# Test 7: Check critical services via API
echo "Checking service health..."
SERVICES_RESPONSE=$(curl -s "http://localhost:${VM_PORT_API}/api/v1/system/services" \
    -H "Authorization: Bearer $JWT")

check_service() {
    local service="$1"
    local status=$(echo "$SERVICES_RESPONSE" | jq -r ".services[] | select(.name==\"$service\") | .active" 2>/dev/null)
    if [ "$status" = "true" ]; then
        echo "  ✓ $service is active"
    else
        echo "  ✗ $service is not active"
        return 1
    fi
}

check_service "nosd.service"
check_service "nos-agent.service"
check_service "caddy.service"

# Test 8: Performance check
echo "Performance metrics:"
BOOT_TIME=$(grep -oE 'Startup finished in [0-9.]+s' "$LOG_DIR/qemu.log" | grep -oE '[0-9.]+' | head -1)
if [ -n "$BOOT_TIME" ]; then
    echo "  Boot time: ${BOOT_TIME}s"
    
    # Check if under 90s threshold
    if (( $(echo "$BOOT_TIME < 90" | bc -l) )); then
        echo "  ✓ Boot time within target (< 90s)"
    else
        echo "  ✗ Boot time exceeds target (> 90s)"
        exit 1
    fi
fi

echo ""
echo "========================================="
echo "ISO Smoke Test: PASSED"
echo "========================================="
echo ""
echo "Summary:"
echo "  - System booted successfully"
echo "  - All services are running"
echo "  - API is accessible"
echo "  - Setup flow completed"
echo "  - Authentication working"
echo ""

# Save test results
cat > "$LOG_DIR/results.json" << EOF
{
    "test": "iso-smoke",
    "status": "passed",
    "boot_time": "${BOOT_TIME:-unknown}",
    "otp_extracted": true,
    "setup_completed": true,
    "services_healthy": true,
    "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF

exit 0
