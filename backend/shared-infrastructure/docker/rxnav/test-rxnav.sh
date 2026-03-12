#!/bin/bash
# Test RxNav-in-a-Box APIs for CardioFit Phase 3

set -e

BASE_URL="http://localhost:4000"

echo "=========================================="
echo "  RxNav-in-a-Box API Tests"
echo "=========================================="
echo ""

# Test 1: Version check
echo "1️⃣  Version Check"
curl -s "$BASE_URL/REST/version" | head -5
echo ""
echo ""

# Test 2: Get RxCUI by drug name
echo "2️⃣  Get RxCUI for 'metformin'"
curl -s "$BASE_URL/REST/rxcui.json?name=metformin" | python3 -m json.tool 2>/dev/null || cat
echo ""

# Test 3: Get RxCUI from NDC
echo "3️⃣  Get RxCUI from NDC (example: 00093-7212)"
curl -s "$BASE_URL/REST/rxcui.json?idtype=NDC&id=00093-7212" | python3 -m json.tool 2>/dev/null || cat
echo ""

# Test 4: Get related concepts
echo "4️⃣  Get related concepts for metformin (RxCUI: 6809)"
curl -s "$BASE_URL/REST/rxcui/6809/related.json?tty=SBD+SCD" | python3 -m json.tool 2>/dev/null | head -30
echo "..."
echo ""

# Test 5: Get SPL SetID (for DailyMed lookup)
echo "5️⃣  Get SPL_SET_ID property (DailyMed link)"
curl -s "$BASE_URL/REST/rxcui/6809/property.json?propName=SPL_SET_ID" | python3 -m json.tool 2>/dev/null || cat
echo ""

# Test 6: Drug interactions
echo "6️⃣  Check drug interactions (metformin + lisinopril)"
# metformin = 6809, lisinopril = 29046
curl -s "$BASE_URL/REST/interaction/list.json?rxcuis=6809+29046" | python3 -m json.tool 2>/dev/null | head -40
echo ""

# Test 7: Get drug classes (ATC)
echo "7️⃣  Get ATC drug class for metformin"
curl -s "$BASE_URL/REST/rxclass/class/byRxcui.json?rxcui=6809&relaSource=ATC" | python3 -m json.tool 2>/dev/null | head -30
echo ""

echo "=========================================="
echo "  ✅ All RxNav API tests completed"
echo "=========================================="
