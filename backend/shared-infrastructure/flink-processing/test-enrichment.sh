#!/bin/bash

# Test script to verify FHIR and Neo4j enrichment in Module 2 output

echo "========================================="
echo "Testing Module 2 FHIR/Neo4j Enrichment"
echo "========================================="
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Kafka is running
echo "1. Checking Kafka availability..."
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092 > /dev/null 2>&1
if [ $? -ne 0 ]; then
    echo -e "${RED}❌ Kafka is not running. Please start Kafka first.${NC}"
    exit 1
fi
echo -e "${GREEN}✅ Kafka is running${NC}"

# Check if Flink is deployed
echo ""
echo "2. Checking Flink jobs..."
curl -s http://localhost:8081/jobs | grep -q "Module 2"
if [ $? -ne 0 ]; then
    echo -e "${YELLOW}⚠️  Module 2 not deployed. Deploying now...${NC}"
    # Deploy Module 2 (you would need to adjust this based on your deployment method)
    echo "Please deploy Module 2 manually or run your deployment script"
fi

# Send a test event to trigger enrichment
echo ""
echo "3. Sending test patient event..."
TEST_EVENT='{
  "id": "test-'$(date +%s)'",
  "patient_id": "PAT-TEST-001",
  "encounter_id": "ENC-001",
  "event_type": "VITAL_SIGN",
  "event_time": '$(($(date +%s) * 1000))',
  "source_system": "test-script",
  "payload": {
    "heart_rate": 75,
    "blood_pressure": "120/80",
    "temperature": 37.0,
    "oxygen_saturation": 98,
    "respiratory_rate": 16
  }
}'

echo "$TEST_EVENT" | docker exec -i kafka kafka-console-producer \
    --broker-list localhost:9092 \
    --topic enriched-patient-events-v1

echo -e "${GREEN}✅ Test event sent to enriched-patient-events-v1 topic${NC}"

# Wait for processing
echo ""
echo "4. Waiting for enrichment processing..."
sleep 5

# Check the output in clinical-patterns topic
echo ""
echo "5. Checking enriched output in clinical-patterns-v1 topic..."
echo "Looking for enrichment_data field with FHIR and Neo4j data..."
echo ""

ENRICHED_OUTPUT=$(timeout 3 docker exec kafka kafka-console-consumer \
    --bootstrap-server localhost:9092 \
    --topic clinical-patterns-v1 \
    --from-beginning \
    --max-messages 1 \
    2>/dev/null | tail -1)

if [ -z "$ENRICHED_OUTPUT" ]; then
    echo -e "${RED}❌ No enriched output found in clinical-patterns-v1${NC}"
    echo "This could mean Module 2 is not running or not processing events."
    exit 1
fi

# Parse and check for enrichment fields
echo "Enriched Event Output:"
echo "----------------------"
echo "$ENRICHED_OUTPUT" | python3 -m json.tool 2>/dev/null || echo "$ENRICHED_OUTPUT"
echo ""

# Check for key enrichment fields
echo "6. Verifying enrichment data..."
echo ""

# Check for enrichment_data field
if echo "$ENRICHED_OUTPUT" | grep -q '"enrichment_data"'; then
    echo -e "${GREEN}✅ enrichment_data field present${NC}"

    # Check for FHIR demographics
    if echo "$ENRICHED_OUTPUT" | grep -q '"fhir_demographics"'; then
        echo -e "${GREEN}✅ FHIR demographics included${NC}"
    else
        echo -e "${YELLOW}⚠️  FHIR demographics not found${NC}"
    fi

    # Check for FHIR medications
    if echo "$ENRICHED_OUTPUT" | grep -q '"fhir_medications"'; then
        echo -e "${GREEN}✅ FHIR medications included${NC}"
    else
        echo -e "${YELLOW}⚠️  FHIR medications not found (may be empty for new patient)${NC}"
    fi

    # Check for FHIR conditions
    if echo "$ENRICHED_OUTPUT" | grep -q '"fhir_conditions"'; then
        echo -e "${GREEN}✅ FHIR conditions included${NC}"
    else
        echo -e "${YELLOW}⚠️  FHIR conditions not found (may be empty for new patient)${NC}"
    fi

    # Check for Neo4j care team
    if echo "$ENRICHED_OUTPUT" | grep -q '"neo4j_care_team"'; then
        echo -e "${GREEN}✅ Neo4j care team included${NC}"
    else
        echo -e "${YELLOW}⚠️  Neo4j care team not found (may be empty for new patient)${NC}"
    fi

    # Check for Neo4j risk cohorts
    if echo "$ENRICHED_OUTPUT" | grep -q '"neo4j_risk_cohorts"'; then
        echo -e "${GREEN}✅ Neo4j risk cohorts included${NC}"
    else
        echo -e "${YELLOW}⚠️  Neo4j risk cohorts not found (may be empty for new patient)${NC}"
    fi

    # Check for risk scores
    if echo "$ENRICHED_OUTPUT" | grep -q '"sepsis_score"\|"deterioration_score"\|"readmission_risk"'; then
        echo -e "${GREEN}✅ Risk scores included${NC}"
    else
        echo -e "${YELLOW}⚠️  Risk scores not found${NC}"
    fi

    # Check for latest vitals
    if echo "$ENRICHED_OUTPUT" | grep -q '"latest_vitals"'; then
        echo -e "${GREEN}✅ Latest vitals included${NC}"
    else
        echo -e "${YELLOW}⚠️  Latest vitals not found${NC}"
    fi

else
    echo -e "${RED}❌ enrichment_data field NOT present - this is the issue!${NC}"
    echo ""
    echo "The enriched event should contain an 'enrichment_data' field with:"
    echo "  - fhir_demographics"
    echo "  - fhir_medications"
    echo "  - fhir_conditions"
    echo "  - fhir_allergies"
    echo "  - neo4j_care_team"
    echo "  - neo4j_risk_cohorts"
    echo "  - risk scores"
    echo "  - latest vitals/labs"
fi

echo ""
echo "========================================="
echo "Test Summary"
echo "========================================="

# Check patient_context for completeness
if echo "$ENRICHED_OUTPUT" | grep -q '"patient_context"'; then
    echo -e "${GREEN}✅ patient_context field present${NC}"

    if echo "$ENRICHED_OUTPUT" | grep -q '"careTeam"'; then
        echo -e "${GREEN}✅ careTeam in patient_context${NC}"
    else
        echo -e "${YELLOW}⚠️  careTeam not in patient_context${NC}"
    fi

    if echo "$ENRICHED_OUTPUT" | grep -q '"riskCohorts"'; then
        echo -e "${GREEN}✅ riskCohorts in patient_context${NC}"
    else
        echo -e "${YELLOW}⚠️  riskCohorts not in patient_context${NC}"
    fi
else
    echo -e "${RED}❌ patient_context field missing${NC}"
fi

echo ""
echo "Note: Empty fields for new patients are expected on first encounter."
echo "The key is that the enrichment_data structure should be present with all expected fields."
echo ""
echo "To see full enriched output, run:"
echo "docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic clinical-patterns-v1 --from-beginning --max-messages 5"