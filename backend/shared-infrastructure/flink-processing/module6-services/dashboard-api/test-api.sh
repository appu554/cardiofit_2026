#!/bin/bash

# Dashboard API Test Script
# Tests all major endpoints and functionality

set -e

API_URL="${API_URL:-http://localhost:4000}"
HOSPITAL_ID="${HOSPITAL_ID:-HOSP001}"

echo "=========================================="
echo "Dashboard API Test Suite"
echo "=========================================="
echo "API URL: $API_URL"
echo "Hospital ID: $HOSPITAL_ID"
echo ""

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test counter
PASSED=0
FAILED=0

# Function to test endpoint
test_endpoint() {
    local name=$1
    local url=$2
    local expected_code=${3:-200}

    echo -n "Testing $name... "
    response=$(curl -s -o /dev/null -w "%{http_code}" "$url")

    if [ "$response" -eq "$expected_code" ]; then
        echo -e "${GREEN}PASS${NC} (HTTP $response)"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC} (Expected $expected_code, got $response)"
        ((FAILED++))
    fi
}

# Function to test GraphQL query
test_graphql() {
    local name=$1
    local query=$2

    echo -n "Testing GraphQL: $name... "

    response=$(curl -s -X POST "$API_URL/graphql" \
        -H "Content-Type: application/json" \
        -d "{\"query\":\"$query\"}" \
        -w "\n%{http_code}")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | head -n-1)

    if [ "$http_code" -eq 200 ] && echo "$body" | grep -q "data"; then
        echo -e "${GREEN}PASS${NC}"
        ((PASSED++))
    else
        echo -e "${RED}FAIL${NC}"
        echo "Response: $body"
        ((FAILED++))
    fi
}

echo "1. Testing Health Endpoints"
echo "----------------------------"
test_endpoint "Health Check" "$API_URL/health" 200
test_endpoint "Readiness Probe" "$API_URL/ready" 200
test_endpoint "Liveness Probe" "$API_URL/live" 200
test_endpoint "Metrics" "$API_URL/metrics" 200
test_endpoint "Root Endpoint" "$API_URL/" 200
echo ""

echo "2. Testing GraphQL Endpoint"
echo "---------------------------"
test_endpoint "GraphQL Endpoint" "$API_URL/graphql" 400  # GET without query returns 400
echo ""

echo "3. Testing GraphQL Queries"
echo "--------------------------"

# Simple introspection query
test_graphql "Schema Introspection" "{__schema{types{name}}}"

# Hospital KPIs query
test_graphql "Hospital KPIs" "{hospitalKpis(hospitalId:\\\"$HOSPITAL_ID\\\"){hospitalId timestamp}}"

# Department metrics query
test_graphql "Department Metrics" "{departmentMetrics(hospitalId:\\\"$HOSPITAL_ID\\\"){departmentId departmentName}}"

# High risk patients query
test_graphql "High Risk Patients" "{highRiskPatients(hospitalId:\\\"$HOSPITAL_ID\\\" limit:10){patientId riskLevel}}"

# Sepsis surveillance query
test_graphql "Sepsis Surveillance" "{sepsisSurveillance(hospitalId:\\\"$HOSPITAL_ID\\\"){alertId patientId sepsisStage}}"

# Quality metrics query
test_graphql "Quality Metrics" "{qualityMetrics(hospitalId:\\\"$HOSPITAL_ID\\\"){metricId metricType metricValue}}"

# Dashboard summary query
test_graphql "Dashboard Summary" "{dashboardSummary(hospitalId:\\\"$HOSPITAL_ID\\\"){timestamp realtimeStats{activePatients}}}"

# Realtime stats query
test_graphql "Realtime Stats" "{realtimeStats(hospitalId:\\\"$HOSPITAL_ID\\\"){activePatients availableBeds lastUpdated}}"

echo ""

echo "4. Testing Error Handling"
echo "-------------------------"
test_graphql "Invalid Query" "{invalidQuery{field}}"  # Should return error but 200 status
echo ""

echo "5. Checking Service Health Details"
echo "-----------------------------------"
health_response=$(curl -s "$API_URL/health")
echo "$health_response" | jq '.' 2>/dev/null || echo "Health response: $health_response"

kafka_status=$(echo "$health_response" | jq -r '.services.kafka' 2>/dev/null || echo "unknown")
postgres_status=$(echo "$health_response" | jq -r '.services.postgres' 2>/dev/null || echo "unknown")
redis_status=$(echo "$health_response" | jq -r '.services.redis' 2>/dev/null || echo "unknown")
influxdb_status=$(echo "$health_response" | jq -r '.services.influxdb' 2>/dev/null || echo "unknown")

echo ""
echo "Service Status:"
echo "  Kafka:     $kafka_status"
echo "  PostgreSQL: $postgres_status"
echo "  Redis:     $redis_status"
echo "  InfluxDB:  $influxdb_status"
echo ""

echo "=========================================="
echo "Test Results"
echo "=========================================="
echo -e "Passed: ${GREEN}$PASSED${NC}"
echo -e "Failed: ${RED}$FAILED${NC}"
echo "Total:  $((PASSED + FAILED))"
echo ""

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed. Check logs with: docker-compose logs dashboard-api${NC}"
    exit 1
fi
