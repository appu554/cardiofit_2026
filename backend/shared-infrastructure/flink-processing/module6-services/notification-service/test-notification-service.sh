#!/bin/bash

# CardioFit Notification Service - Integration Test Script

set -e

BASE_URL="http://localhost:8070"
USER_ID="test-user-123"

echo "========================================="
echo "Notification Service Integration Tests"
echo "========================================="
echo ""

# Color codes for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
TESTS_PASSED=0
TESTS_FAILED=0

# Function to test endpoint
test_endpoint() {
    local name=$1
    local method=$2
    local endpoint=$3
    local data=$4

    echo -n "Testing $name... "

    if [ -z "$data" ]; then
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint")
    else
        response=$(curl -s -w "\n%{http_code}" -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    fi

    http_code=$(echo "$response" | tail -n 1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" -ge 200 ] && [ "$http_code" -lt 300 ]; then
        echo -e "${GREEN}PASSED${NC} (HTTP $http_code)"
        TESTS_PASSED=$((TESTS_PASSED + 1))
        return 0
    else
        echo -e "${RED}FAILED${NC} (HTTP $http_code)"
        echo "Response: $body"
        TESTS_FAILED=$((TESTS_FAILED + 1))
        return 1
    fi
}

# Wait for service to be ready
echo "Waiting for service to be ready..."
max_attempts=30
attempt=0
while [ $attempt -lt $max_attempts ]; do
    if curl -s "$BASE_URL/api/v1/notifications/health" > /dev/null 2>&1; then
        echo -e "${GREEN}Service is ready!${NC}"
        echo ""
        break
    fi
    attempt=$((attempt + 1))
    echo -n "."
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    echo -e "${RED}Service failed to start${NC}"
    exit 1
fi

# Run tests
echo "Running integration tests..."
echo ""

# Test 1: Health check
test_endpoint "Health Check" "GET" "/api/v1/notifications/health"

# Test 2: Statistics
test_endpoint "Get Statistics" "GET" "/api/v1/notifications/stats"

# Test 3: Get user preferences
test_endpoint "Get User Preferences" "GET" "/api/v1/notifications/preferences/$USER_ID"

# Test 4: Update user preferences
PREFS_JSON='{
  "userId": "'$USER_ID'",
  "email": "test@example.com",
  "phoneNumber": "+1234567890",
  "enabledChannels": ["EMAIL", "SMS", "PUSH"],
  "severityThresholds": {
    "CRITICAL": ["SMS", "PUSH", "EMAIL"],
    "HIGH": ["PUSH", "EMAIL"]
  },
  "quietHours": {
    "enabled": true,
    "startHour": 22,
    "endHour": 7,
    "overrideCritical": true
  },
  "onCallSchedule": {
    "onCall": true
  },
  "alertBundlingEnabled": true,
  "bundlingWindowMinutes": 10
}'

test_endpoint "Update User Preferences" "PUT" "/api/v1/notifications/preferences/$USER_ID" "$PREFS_JSON"

# Test 5: Get rate limit status
test_endpoint "Get Rate Limit Status" "GET" "/api/v1/notifications/rate-limit/$USER_ID"

# Test 6: Check on-call status
test_endpoint "Check On-Call Status" "GET" "/api/v1/notifications/on-call/$USER_ID"

# Test 7: Prometheus metrics
echo -n "Testing Prometheus Metrics... "
response=$(curl -s -w "\n%{http_code}" "$BASE_URL/actuator/prometheus")
http_code=$(echo "$response" | tail -n 1)
body=$(echo "$response" | sed '$d')

if [ "$http_code" -eq 200 ] && echo "$body" | grep -q "alerts_received"; then
    echo -e "${GREEN}PASSED${NC} (HTTP $http_code)"
    TESTS_PASSED=$((TESTS_PASSED + 1))
else
    echo -e "${RED}FAILED${NC} (HTTP $http_code)"
    TESTS_FAILED=$((TESTS_FAILED + 1))
fi

# Test 8: Actuator health
test_endpoint "Actuator Health" "GET" "/actuator/health"

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="
echo -e "Total Tests: $((TESTS_PASSED + TESTS_FAILED))"
echo -e "${GREEN}Passed: $TESTS_PASSED${NC}"
echo -e "${RED}Failed: $TESTS_FAILED${NC}"
echo ""

if [ $TESTS_FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed.${NC}"
    exit 1
fi
