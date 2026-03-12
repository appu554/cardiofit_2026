#!/bin/bash
# Health check for all runtime services
# Usage: ./health-check.sh

set -e

FHIR_URL="${FHIR_URL:-http://localhost:8080/fhir}"
CQL_URL="${CQL_URL:-http://localhost:8081}"
REDIS_HOST="${REDIS_HOST:-localhost}"
REDIS_PORT="${REDIS_PORT:-6379}"

echo "=== Vaidshala Runtime Health Check ==="

# Check FHIR Server
echo -n "FHIR Server: "
if curl -sS "$FHIR_URL/metadata" > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

# Check CQL Executor
echo -n "CQL Executor: "
if curl -sS "$CQL_URL/health" > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

# Check Redis
echo -n "Redis Cache: "
if redis-cli -h $REDIS_HOST -p $REDIS_PORT ping > /dev/null 2>&1; then
    echo "✓ OK"
else
    echo "✗ FAILED"
fi

echo "=== Health Check Complete ==="
