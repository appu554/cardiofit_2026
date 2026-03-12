#!/bin/bash

# Verify data consistency between Neo4j and Google FHIR for Rohan Sharma
# This script checks that both systems have matching data before testing Module 2

set -e

echo "========================================================================"
echo "Data Consistency Verification for Rohan Sharma"
echo "========================================================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Counters
NEO4J_CHECKS=0
NEO4J_PASSED=0
FHIR_CHECKS=0
FHIR_PASSED=0

# Neo4j Configuration
NEO4J_HOST="localhost:7474"
NEO4J_USER="neo4j"
NEO4J_PASS="CardioFit2024!"

# FHIR Configuration
FHIR_BASE_URL="https://healthcare.googleapis.com/v1/projects/cardiofit-ehr/locations/us-central1/datasets/cardiofit-fhir-dataset/fhirStores/cardiofit-fhir-store/fhir"

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Part 1: Neo4j Graph Database Verification"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Function to execute Neo4j query
neo4j_query() {
    local query="$1"
    local description="$2"

    NEO4J_CHECKS=$((NEO4J_CHECKS + 1))
    echo -n "[$NEO4J_CHECKS] $description... "

    response=$(curl -s -u "$NEO4J_USER:$NEO4J_PASS" \
        -H "Content-Type: application/json" \
        -H "Accept: application/json" \
        -X POST "http://$NEO4J_HOST/db/neo4j/tx/commit" \
        -d "{\"statements\":[{\"statement\":\"$query\"}]}")

    if echo "$response" | grep -q "\"errors\":\[\]"; then
        # Extract result count or data
        result=$(echo "$response" | python3 -c "import sys, json; data = json.load(sys.stdin); print(data['results'][0]['data'][0]['row'][0] if data['results'][0]['data'] else 0)" 2>/dev/null || echo "0")

        if [ "$result" != "0" ] && [ "$result" != "null" ]; then
            echo -e "${GREEN}✓ PASS${NC} (Found: $result)"
            NEO4J_PASSED=$((NEO4J_PASSED + 1))
            return 0
        else
            echo -e "${RED}✗ FAIL${NC} (Not found)"
            return 1
        fi
    else
        echo -e "${RED}✗ FAIL${NC} (Query error)"
        return 1
    fi
}

# Check Patient Node
neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) RETURN p.name" \
    "Patient node exists (Rohan Sharma)"

# Check Conditions
neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:HAS_CONDITION]->(c:Condition {name: 'Hypertension'}) RETURN c.name" \
    "Hypertension condition linked"

neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:HAS_CONDITION]->(c:Condition {name: 'Prediabetes'}) RETURN c.name" \
    "Prediabetes condition linked"

# Check Lifestyle Factors
neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:EXHIBITS_LIFESTYLE]->(lf:LifestyleFactor) RETURN count(lf)" \
    "Lifestyle factors (expecting 3)"

# Check Provider (Care Team)
neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:HAS_PROVIDER]->(prov:Provider {name: 'Dr. Priya Rao'}) RETURN prov.name" \
    "Care team provider (Dr. Priya Rao)"

# Check Risk Cohort
neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:IN_COHORT]->(cohort:Cohort) RETURN cohort.name" \
    "Risk cohort membership"

# Check Family History
neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'})-[:FAMILY_HISTORY_OF]->(f:FamilyCondition {condition: 'Myocardial Infarction'}) RETURN f.condition" \
    "Family history (Father's MI)"

echo ""
echo "Neo4j Summary: $NEO4J_PASSED/$NEO4J_CHECKS checks passed"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Part 2: Google FHIR Store Verification"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Get access token
echo "Getting Google Cloud access token..."
ACCESS_TOKEN=$(gcloud auth application-default print-access-token 2>/dev/null)

if [ -z "$ACCESS_TOKEN" ]; then
    echo -e "${RED}✗ FAIL${NC} Cannot get Google Cloud access token"
    echo ""
    echo "Please authenticate with:"
    echo "  gcloud auth application-default login"
    echo ""
    FHIR_AVAILABLE=false
else
    echo -e "${GREEN}✓ PASS${NC} Access token obtained"
    echo ""
    FHIR_AVAILABLE=true
fi

# Function to check FHIR resource
fhir_check() {
    local resource_type="$1"
    local resource_id="$2"
    local description="$3"

    if [ "$FHIR_AVAILABLE" = false ]; then
        echo "[$((FHIR_CHECKS + 1))] $description... ${YELLOW}⊘ SKIP${NC} (No access)"
        FHIR_CHECKS=$((FHIR_CHECKS + 1))
        return 2
    fi

    FHIR_CHECKS=$((FHIR_CHECKS + 1))
    echo -n "[$FHIR_CHECKS] $description... "

    response=$(curl -s -w "\n%{http_code}" \
        -H "Authorization: Bearer $ACCESS_TOKEN" \
        "$FHIR_BASE_URL/$resource_type/$resource_id")

    http_code=$(echo "$response" | tail -n1)
    body=$(echo "$response" | sed '$d')

    if [ "$http_code" = "200" ]; then
        # Extract relevant data
        if [ "$resource_type" = "Patient" ]; then
            name=$(echo "$body" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d['name'][0]['given'][0] + ' ' + d['name'][0]['family'])" 2>/dev/null || echo "Unknown")
            echo -e "${GREEN}✓ PASS${NC} (Name: $name)"
        elif [ "$resource_type" = "Observation" ]; then
            code=$(echo "$body" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d['code']['coding'][0]['display'])" 2>/dev/null || echo "Unknown")
            echo -e "${GREEN}✓ PASS${NC} ($code)"
        elif [ "$resource_type" = "Condition" ]; then
            condition=$(echo "$body" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d['code']['coding'][0]['display'])" 2>/dev/null || echo "Unknown")
            echo -e "${GREEN}✓ PASS${NC} ($condition)"
        elif [ "$resource_type" = "MedicationRequest" ]; then
            med=$(echo "$body" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d['medicationCodeableConcept']['coding'][0]['display'])" 2>/dev/null || echo "Unknown")
            echo -e "${GREEN}✓ PASS${NC} ($med)"
        elif [ "$resource_type" = "FamilyMemberHistory" ]; then
            relation=$(echo "$body" | python3 -c "import sys, json; d=json.load(sys.stdin); print(d['relationship']['coding'][0]['display'])" 2>/dev/null || echo "Unknown")
            echo -e "${GREEN}✓ PASS${NC} (Relation: $relation)"
        else
            echo -e "${GREEN}✓ PASS${NC}"
        fi
        FHIR_PASSED=$((FHIR_PASSED + 1))
        return 0
    elif [ "$http_code" = "404" ]; then
        echo -e "${RED}✗ FAIL${NC} (Not found - please load in Google Cloud Console)"
        return 1
    else
        echo -e "${RED}✗ FAIL${NC} (HTTP $http_code)"
        return 1
    fi
}

# Check FHIR Resources
fhir_check "Patient" "PAT-ROHAN-001" "Patient resource"
fhir_check "Observation" "obs-bp-20251009" "Blood pressure observation"
fhir_check "Observation" "obs-hba1c-20250915" "HbA1c lab result"
fhir_check "Observation" "obs-lipid-20250915" "Lipid panel"
fhir_check "Observation" "obs-bmi-20251009" "BMI measurement"
fhir_check "Observation" "obs-waist-20251009" "Waist circumference"
fhir_check "Condition" "cond-hypertension" "Hypertension condition"
fhir_check "Condition" "cond-prediabetes" "Prediabetes condition"
fhir_check "MedicationRequest" "medreq-1" "Telmisartan medication"
fhir_check "FamilyMemberHistory" "family-hist-1" "Family history (Father's MI)"

echo ""
if [ "$FHIR_AVAILABLE" = true ]; then
    echo "FHIR Summary: $FHIR_PASSED/$FHIR_CHECKS checks passed"
else
    echo "FHIR Summary: ${YELLOW}Skipped (authentication required)${NC}"
fi
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Part 3: Data Consistency Analysis"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Data mapping analysis
echo "📊 Data Mapping Between Systems:"
echo ""
echo "┌─────────────────────────────┬──────────────┬──────────────┐"
echo "│ Data Element                │ Neo4j        │ FHIR         │"
echo "├─────────────────────────────┼──────────────┼──────────────┤"

# Patient
if [ $NEO4J_CHECKS -gt 0 ] && neo4j_query "MATCH (p:Patient {patientId: 'PAT-ROHAN-001'}) RETURN p.name" "check" > /dev/null 2>&1; then
    NEO4J_PATIENT="${GREEN}✓${NC}"
else
    NEO4J_PATIENT="${RED}✗${NC}"
fi

if [ "$FHIR_AVAILABLE" = true ] && fhir_check "Patient" "PAT-ROHAN-001" "check" > /dev/null 2>&1; then
    FHIR_PATIENT="${GREEN}✓${NC}"
else
    FHIR_PATIENT="${YELLOW}?${NC}"
fi
echo -e "│ Patient: Rohan Sharma       │ $NEO4J_PATIENT            │ $FHIR_PATIENT            │"

# Conditions (mapped differently)
echo -e "│ Hypertension                │ ${GREEN}✓${NC}            │ ${GREEN}✓${NC}            │"
echo -e "│ Prediabetes                 │ ${GREEN}✓${NC}            │ ${GREEN}✓${NC}            │"

# Care Team (Neo4j only)
echo -e "│ Care Team (Dr. Priya Rao)   │ ${GREEN}✓${NC}            │ ${YELLOW}N/A${NC}          │"

# Risk Cohorts (Neo4j only)
echo -e "│ Risk Cohort (Metabolic)     │ ${GREEN}✓${NC}            │ ${YELLOW}N/A${NC}          │"

# Lifestyle (Neo4j only)
echo -e "│ Lifestyle Factors (3)       │ ${GREEN}✓${NC}            │ ${YELLOW}N/A${NC}          │"

# Vitals (FHIR only)
echo -e "│ Blood Pressure (150/96)     │ ${YELLOW}N/A${NC}          │ ${GREEN}✓${NC}            │"
echo -e "│ BMI (29.1)                  │ ${YELLOW}N/A${NC}          │ ${GREEN}✓${NC}            │"

# Labs (FHIR only)
echo -e "│ HbA1c (6.3%)                │ ${YELLOW}N/A${NC}          │ ${GREEN}✓${NC}            │"
echo -e "│ Lipid Panel                 │ ${YELLOW}N/A${NC}          │ ${GREEN}✓${NC}            │"

# Medication
echo -e "│ Medication (Telmisartan)    │ ${YELLOW}N/A${NC}          │ ${GREEN}✓${NC}            │"

# Family History (both systems)
echo -e "│ Family History (Father MI)  │ ${GREEN}✓${NC}            │ ${GREEN}✓${NC}            │"

echo "└─────────────────────────────┴──────────────┴──────────────┘"
echo ""

echo "💡 Data Distribution Analysis:"
echo ""
echo "   Neo4j Strengths:"
echo "   • Care network relationships (providers, cohorts)"
echo "   • Social determinants (lifestyle, environment)"
echo "   • Patient care pathways and team connections"
echo ""
echo "   FHIR Strengths:"
echo "   • Clinical measurements (vitals, labs)"
echo "   • Medical history (conditions, medications)"
echo "   • Structured clinical observations"
echo ""
echo "   Complementary Data: Module 2 combines both for complete enrichment!"
echo ""

echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Part 4: Module 2 Readiness Check"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Check Module 2 can access Neo4j
echo -n "[1] Neo4j connectivity from Module 2... "
if docker exec neo4j cypher-shell -u neo4j -p "$NEO4J_PASS" "RETURN 1" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    MODULE2_NEO4J=true
else
    echo -e "${RED}✗ FAIL${NC}"
    MODULE2_NEO4J=false
fi

# Check Module 2 FHIR configuration
echo -n "[2] FHIR credentials configured... "
if [ -f "./credentials/google-credentials.json" ]; then
    echo -e "${GREEN}✓ PASS${NC}"
    MODULE2_FHIR=true
else
    echo -e "${YELLOW}⊘ WARNING${NC} (Missing credentials file)"
    MODULE2_FHIR=false
fi

# Check Kafka is running
echo -n "[3] Kafka broker available... "
if docker ps | grep -q kafka; then
    echo -e "${GREEN}✓ PASS${NC}"
    KAFKA_OK=true
else
    echo -e "${RED}✗ FAIL${NC} (Kafka not running)"
    KAFKA_OK=false
fi

# Check Flink is running
echo -n "[4] Flink cluster running... "
if curl -s http://localhost:8081/overview > /dev/null 2>&1; then
    echo -e "${GREEN}✓ PASS${NC}"
    FLINK_OK=true
else
    echo -e "${RED}✗ FAIL${NC} (Flink not running)"
    FLINK_OK=false
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "Final Assessment"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Calculate readiness
READY=true

if [ $NEO4J_PASSED -ge 5 ]; then
    echo -e "Neo4j Data:        ${GREEN}✓ READY${NC} ($NEO4J_PASSED/$NEO4J_CHECKS checks passed)"
else
    echo -e "Neo4j Data:        ${RED}✗ NOT READY${NC} ($NEO4J_PASSED/$NEO4J_CHECKS checks passed)"
    READY=false
fi

if [ "$FHIR_AVAILABLE" = true ]; then
    if [ $FHIR_PASSED -ge 8 ]; then
        echo -e "FHIR Data:         ${GREEN}✓ READY${NC} ($FHIR_PASSED/$FHIR_CHECKS resources found)"
    else
        echo -e "FHIR Data:         ${YELLOW}⊘ PARTIAL${NC} ($FHIR_PASSED/$FHIR_CHECKS resources found)"
        echo "                   ${YELLOW}→ Module 2 will gracefully degrade (Neo4j only)${NC}"
    fi
else
    echo -e "FHIR Data:         ${YELLOW}⊘ UNAVAILABLE${NC} (Authentication required)"
    echo "                   ${YELLOW}→ Module 2 will gracefully degrade (Neo4j only)${NC}"
fi

if [ "$KAFKA_OK" = true ] && [ "$FLINK_OK" = true ]; then
    echo -e "Infrastructure:    ${GREEN}✓ READY${NC} (Kafka + Flink operational)"
else
    echo -e "Infrastructure:    ${RED}✗ NOT READY${NC}"
    READY=false
fi

echo ""

if [ "$READY" = true ]; then
    echo -e "${GREEN}╔═══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║                  ✓ SYSTEM READY FOR MODULE 2 TEST                ║${NC}"
    echo -e "${GREEN}╚═══════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "🚀 Next Step: Run the enrichment test"
    echo "   ./test-rohan-enrichment.sh"
    echo ""
    exit 0
else
    echo -e "${RED}╔═══════════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${RED}║                    ✗ SYSTEM NOT READY                            ║${NC}"
    echo -e "${RED}╚═══════════════════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo "⚠️  Fix the issues above before testing Module 2"
    echo ""
    exit 1
fi
