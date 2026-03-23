#!/bin/bash
# Verify Clinical Signal Capture Layer — Phase 2
# Runs build + test for all modified services

set -e
PASS=0
FAIL=0
SERVICES=(
    "kb-20-patient-profile"
    "kb-21-behavioral-intelligence"
    "kb-22-hpi-engine"
    "kb-23-decision-cards"
    "kb-25-lifestyle-knowledge-graph"
    "kb-26-metabolic-digital-twin"
)
BASE="$(cd "$(dirname "$0")/.." && pwd)"

for svc in "${SERVICES[@]}"; do
    echo "=== Building $svc ==="
    if (cd "$BASE/$svc" && go build ./...); then
        echo "  BUILD: OK"
    else
        echo "  BUILD: FAIL"
        FAIL=$((FAIL+1))
        continue
    fi

    echo "=== Testing $svc ==="
    if (cd "$BASE/$svc" && go test ./... 2>&1); then
        echo "  TEST: OK"
        PASS=$((PASS+1))
    else
        echo "  TEST: FAIL (check output above)"
        FAIL=$((FAIL+1))
    fi
    echo ""
done

echo "=== Summary ==="
echo "PASS: $PASS  FAIL: $FAIL  TOTAL: ${#SERVICES[@]}"
if [ $FAIL -gt 0 ]; then
    exit 1
fi
echo "All services verified."
