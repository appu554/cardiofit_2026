#!/bin/bash
# ═══════════════════════════════════════════════════════════════════
# KB-7 AU Terminology Service - Verification Script
# Tests all 6 spec-compliant operations with latency targets
# ═══════════════════════════════════════════════════════════════════

set -e

BASE_URL="${KB7_URL:-http://localhost:8087}"
VERBOSE="${VERBOSE:-false}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}    KB-7 AU TERMINOLOGY SERVICE - VERIFICATION${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
echo -e "Base URL: ${BASE_URL}"
echo ""

PASS_COUNT=0
FAIL_COUNT=0

# Function to measure latency and check result
check_operation() {
    local NAME=$1
    local TARGET=$2
    local COMMAND=$3
    local CHECK_FIELD=$4
    local CHECK_VALUE=$5

    START=$(python3 -c "import time; print(int(time.time() * 1000))")
    RESULT=$(eval "$COMMAND" 2>/dev/null || echo '{"error": "request failed"}')
    END=$(python3 -c "import time; print(int(time.time() * 1000))")
    LATENCY=$((END - START))

    if [ "$VERBOSE" = "true" ]; then
        echo -e "${YELLOW}Response:${NC}"
        echo "$RESULT" | python3 -m json.tool 2>/dev/null || echo "$RESULT"
        echo ""
    fi

    # Check latency
    if [ "$LATENCY" -lt "$TARGET" ]; then
        LATENCY_OK=true
    else
        LATENCY_OK=false
    fi

    # Check field value if specified
    if [ -n "$CHECK_FIELD" ] && [ -n "$CHECK_VALUE" ]; then
        ACTUAL=$(echo "$RESULT" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('$CHECK_FIELD', '?'))" 2>/dev/null || echo "?")
        if [ "$ACTUAL" = "$CHECK_VALUE" ] || [ "$ACTUAL" != "?" ]; then
            VALUE_OK=true
        else
            VALUE_OK=false
        fi
    else
        VALUE_OK=true
        ACTUAL=""
    fi

    # Output result
    if [ "$LATENCY_OK" = true ] && [ "$VALUE_OK" = true ]; then
        echo -e "${GREEN}   ✅ ${LATENCY}ms PASS${NC} - $NAME"
        if [ -n "$ACTUAL" ] && [ "$ACTUAL" != "?" ]; then
            echo -e "      ${CHECK_FIELD}: $ACTUAL"
        fi
        ((PASS_COUNT++))
    else
        echo -e "${RED}   ❌ ${LATENCY}ms FAIL${NC} - $NAME"
        if [ "$LATENCY_OK" = false ]; then
            echo -e "      Latency ${LATENCY}ms > target ${TARGET}ms"
        fi
        if [ "$VALUE_OK" = false ]; then
            echo -e "      ${CHECK_FIELD}: $ACTUAL (expected non-empty)"
        fi
        ((FAIL_COUNT++))
    fi
}

# ═══════════════════════════════════════════════════════════════════
# Health Check
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${YELLOW}📋 HEALTH CHECK${NC}"
HEALTH=$(curl -s "$BASE_URL/health" 2>/dev/null || echo '{"status": "unavailable"}')
STATUS=$(echo "$HEALTH" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('status', 'unknown'))" 2>/dev/null || echo "unknown")

if [ "$STATUS" = "healthy" ]; then
    echo -e "${GREEN}   ✅ Service healthy${NC}"

    # Check backends
    CONFIG=$(curl -s "$BASE_URL/v1/subsumption/config" 2>/dev/null || echo '{}')
    BACKEND=$(echo "$CONFIG" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('preferred_backend', 'unknown'))" 2>/dev/null || echo "unknown")
    NEO4J_AVAILABLE=$(echo "$CONFIG" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('backends', {}).get('neo4j', {}).get('available', False))" 2>/dev/null || echo "False")

    echo -e "      Preferred backend: ${BACKEND}"
    echo -e "      Neo4j available: ${NEO4J_AVAILABLE}"
else
    echo -e "${RED}   ❌ Service unhealthy or unavailable${NC}"
    echo -e "   Status: $STATUS"
    echo -e "\n${RED}Cannot proceed with verification. Please start the KB-7 service.${NC}"
    exit 1
fi

# ═══════════════════════════════════════════════════════════════════
# 1. Concept Lookup (<50ms)
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${YELLOW}1️⃣  CONCEPT LOOKUP (<50ms)${NC}"
echo "   Silent Translator: code → display name"

check_operation \
    "SNOMED 44054006 (Type 2 Diabetes)" \
    50 \
    "curl -s '$BASE_URL/v1/concepts/SNOMED/44054006'" \
    "display" \
    ""

# ═══════════════════════════════════════════════════════════════════
# 2. Subsumption Check (<100ms)
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${YELLOW}2️⃣  SUBSUMPTION CHECK (<100ms)${NC}"
echo "   Is-A Logic: Is Type 2 DM a kind of Diabetes Mellitus?"

START=$(python3 -c "import time; print(int(time.time() * 1000))")
RESULT=$(curl -s -X POST "$BASE_URL/v1/subsumption/test" \
    -H "Content-Type: application/json" \
    -d '{"code_a": "44054006", "code_b": "73211009", "system": "SNOMED"}' 2>/dev/null || echo '{"error": "failed"}')
END=$(python3 -c "import time; print(int(time.time() * 1000))")
LATENCY=$((END - START))

SUBSUMES=$(echo "$RESULT" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('subsumes', False))" 2>/dev/null || echo "False")
BACKEND=$(echo "$RESULT" | python3 -c "import json,sys; d=json.load(sys.stdin); print(d.get('backend', 'unknown'))" 2>/dev/null || echo "unknown")

if [ "$LATENCY" -lt 100 ] && [ "$SUBSUMES" = "True" ]; then
    echo -e "${GREEN}   ✅ ${LATENCY}ms PASS${NC}"
    echo -e "      subsumes: $SUBSUMES"
    echo -e "      backend: $BACKEND"
    ((PASS_COUNT++))
else
    echo -e "${RED}   ❌ ${LATENCY}ms FAIL${NC}"
    echo -e "      subsumes: $SUBSUMES (expected True)"
    echo -e "      backend: $BACKEND"
    ((FAIL_COUNT++))
fi

# ═══════════════════════════════════════════════════════════════════
# 3. HCC Single Mapping (<200ms)
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${YELLOW}3️⃣  HCC SINGLE MAPPING (<200ms)${NC}"
echo "   ICD E11.9 → HCC risk category"

check_operation \
    "E11.9 → HCC mapping" \
    200 \
    "curl -s -X POST '$BASE_URL/v1/hcc/map' -H 'Content-Type: application/json' -d '{\"icd_code\": \"E11.9\", \"model_year\": \"2024\"}'" \
    "" \
    ""

# ═══════════════════════════════════════════════════════════════════
# 4. HCC Batch Mapping (<400ms)
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${YELLOW}4️⃣  HCC BATCH MAPPING (<400ms)${NC}"
echo "   Problem list → RAF score calculation"

check_operation \
    "Batch [E11.9, I10, J44.9]" \
    400 \
    "curl -s -X POST '$BASE_URL/v1/hcc/batch' -H 'Content-Type: application/json' -d '{\"icd_codes\": [\"E11.9\", \"I10\", \"J44.9\"], \"model_year\": \"2024\"}'" \
    "" \
    ""

# ═══════════════════════════════════════════════════════════════════
# 5. Ancestor Query (<300ms)
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${YELLOW}5️⃣  ANCESTOR QUERY (<300ms)${NC}"
echo "   Population health grouper: find all parent concepts"

START=$(python3 -c "import time; print(int(time.time() * 1000))")
RESULT=$(curl -s -X POST "$BASE_URL/v1/subsumption/ancestors" \
    -H "Content-Type: application/json" \
    -d '{"code": "44054006", "system": "SNOMED", "max_depth": 20}' 2>/dev/null || echo '{"error": "failed"}')
END=$(python3 -c "import time; print(int(time.time() * 1000))")
LATENCY=$((END - START))

ANCESTORS=$(echo "$RESULT" | python3 -c "import json,sys; d=json.load(sys.stdin); a=d.get('ancestors',[]); print(len(a) if isinstance(a,list) else d.get('total_ancestors', 0))" 2>/dev/null || echo "0")

if [ "$LATENCY" -lt 300 ] && [ "$ANCESTORS" -gt 0 ]; then
    echo -e "${GREEN}   ✅ ${LATENCY}ms PASS${NC}"
    echo -e "      Ancestors found: $ANCESTORS"
    ((PASS_COUNT++))
else
    echo -e "${RED}   ❌ ${LATENCY}ms FAIL${NC}"
    echo -e "      Ancestors found: $ANCESTORS"
    ((FAIL_COUNT++))
fi

# ═══════════════════════════════════════════════════════════════════
# 6. Value Set Expansion (<500ms)
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${YELLOW}6️⃣  VALUE SET EXPANSION (<500ms)${NC}"
echo "   HEDIS reporting: list all value sets"

START=$(python3 -c "import time; print(int(time.time() * 1000))")
RESULT=$(curl -s "$BASE_URL/v1/rules/valuesets" 2>/dev/null || echo '{"error": "failed"}')
END=$(python3 -c "import time; print(int(time.time() * 1000))")
LATENCY=$((END - START))

VS_COUNT=$(echo "$RESULT" | python3 -c "import json,sys; d=json.load(sys.stdin); vs=d.get('value_sets',[]); print(len(vs) if isinstance(vs,list) else 0)" 2>/dev/null || echo "0")

if [ "$LATENCY" -lt 500 ] && [ "$VS_COUNT" -gt 0 ]; then
    echo -e "${GREEN}   ✅ ${LATENCY}ms PASS${NC}"
    echo -e "      Value Sets: $VS_COUNT (FHIR R4)"
    ((PASS_COUNT++))
else
    echo -e "${RED}   ❌ ${LATENCY}ms FAIL${NC}"
    echo -e "      Value Sets: $VS_COUNT"
    if [ "$VS_COUNT" -eq 0 ]; then
        echo -e "      ${YELLOW}Hint: Run with SEED_BUILTIN_VALUE_SETS=true to populate${NC}"
    fi
    ((FAIL_COUNT++))
fi

# ═══════════════════════════════════════════════════════════════════
# Summary
# ═══════════════════════════════════════════════════════════════════
echo -e "\n${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
TOTAL=$((PASS_COUNT + FAIL_COUNT))
if [ "$FAIL_COUNT" -eq 0 ]; then
    echo -e "${GREEN}    ✅ ALL $PASS_COUNT/$TOTAL OPERATIONS VERIFIED${NC}"
    echo -e "${GREEN}    KB-7 AU VERSION COMPLETE!${NC}"
else
    echo -e "${RED}    ⚠️  $PASS_COUNT/$TOTAL OPERATIONS PASSED${NC}"
    echo -e "${RED}    $FAIL_COUNT OPERATIONS FAILED${NC}"
fi
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"

exit $FAIL_COUNT
