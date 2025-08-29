#!/bin/bash
set -euo pipefail

# Wait for test environment to be ready
# Used by E2E tests to ensure all services are up

TIMEOUT=${TIMEOUT:-120}
INTERVAL=2
ELAPSED=0

echo "Waiting for test environment to be ready..."

# Function to check if a service is ready
check_service() {
    local service=$1
    local check_cmd=$2
    
    if eval "$check_cmd" > /dev/null 2>&1; then
        echo "  ✓ $service is ready"
        return 0
    else
        return 1
    fi
}

# Wait loop
while [ $ELAPSED -lt $TIMEOUT ]; do
    ALL_READY=true
    
    # Check nosd API
    if ! check_service "nosd API" "curl -sf http://localhost:9000/api/v1/health"; then
        ALL_READY=false
    fi
    
    # Check Caddy
    if ! check_service "Caddy" "curl -sf http://localhost:8080/"; then
        ALL_READY=false
    fi
    
    # Check PostgreSQL (if using external port)
    # if ! check_service "PostgreSQL" "nc -z localhost 5432"; then
    #     ALL_READY=false
    # fi
    
    # Check Redis (if using external port)
    # if ! check_service "Redis" "nc -z localhost 6379"; then
    #     ALL_READY=false
    # fi
    
    # Check MailHog SMTP
    if ! check_service "MailHog SMTP" "nc -z localhost 1025"; then
        ALL_READY=false
    fi
    
    # Check webhook mock
    if ! check_service "Webhook Mock" "curl -sf http://localhost:8084/"; then
        ALL_READY=false
    fi
    
    if [ "$ALL_READY" = true ]; then
        echo ""
        echo "✅ All services are ready!"
        echo ""
        
        # Additional setup if needed
        if [ "${SETUP_TEST_DATA:-false}" = "true" ]; then
            echo "Setting up test data..."
            
            # Get OTP for setup
            OTP=$(docker-compose exec -T nosd cat /var/lib/nos/setup-otp.txt 2>/dev/null || echo "123456")
            echo "  OTP: $OTP"
            
            # Complete setup
            curl -X POST http://localhost:8080/api/setup/verify-otp \
                -H "Content-Type: application/json" \
                -d "{\"otp\": \"$OTP\"}" \
                -o /tmp/setup-token.json
            
            TOKEN=$(jq -r '.token' /tmp/setup-token.json)
            
            curl -X POST http://localhost:8080/api/setup/create-admin \
                -H "Content-Type: application/json" \
                -H "Authorization: Bearer $TOKEN" \
                -d '{
                    "username": "admin",
                    "password": "TestAdmin123!",
                    "email": "admin@test.local"
                }' \
                -o /tmp/admin-token.json
            
            echo "  ✓ Test admin created"
        fi
        
        exit 0
    fi
    
    sleep $INTERVAL
    ELAPSED=$((ELAPSED + INTERVAL))
    
    # Show progress
    echo -n "."
done

echo ""
echo "❌ Timeout waiting for services to be ready"
echo ""
echo "Service status:"
docker-compose ps

echo ""
echo "Recent logs:"
docker-compose logs --tail=50

exit 1
