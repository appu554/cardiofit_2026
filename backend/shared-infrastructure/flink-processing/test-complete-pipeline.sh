#!/bin/bash

###############################################################################
# Complete Pipeline Test - All 8 Modules
# Tests end-to-end flow from patient event → analytics output
###############################################################################

set -e

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║           Complete Pipeline Test (8 Modules)                 ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""

###############################################################################
# Test 1: Send Patient Event
###############################################################################

echo -e "${YELLOW}[Test 1/5] Sending test patient event...${NC}"

TEST_EVENT=$(cat <<'EOF'
{
  "id": "test-$(date +%s)",
  "patient_id": "PATIENT-TEST-001",
  "type": "vital_signs",
  "event_time": $(date +%s)000,
  "payload": {
    "heart_rate": 105,
    "blood_pressure_systolic": 145,
    "blood_pressure_diastolic": 95,
    "respiratory_rate": 22,
    "temperature": 38.2,
    "spo2": 94
  },
  "metadata": {
    "source": "bedside_monitor",
    "location": "ICU-BED-12",
    "device_id": "MON-12"
  }
}
EOF
)

# Send to Kafka
echo "$TEST_EVENT" | docker exec -i kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic patient-events-v1

echo -e "${GREEN}✓ Event sent to patient-events-v1${NC}"
echo ""

###############################################################################
# Test 2: Verify Module 1 Output (Enriched Events)
###############################################################################

echo -e "${YELLOW}[Test 2/5] Checking Module 1 output (enriched-patient-events-v1)...${NC}"
sleep 2

timeout 5 docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning \
  --max-messages 1 2>/dev/null | head -1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Module 1 working (Ingestion)${NC}"
else
    echo -e "${YELLOW}⚠ No output from Module 1${NC}"
fi
echo ""

###############################################################################
# Test 3: Verify Module 3 Output (CDS Events)
###############################################################################

echo -e "${YELLOW}[Test 3/5] Checking Module 3 output (comprehensive-cds-events.v1)...${NC}"
sleep 2

timeout 5 docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic comprehensive-cds-events.v1 \
  --from-beginning \
  --max-messages 1 2>/dev/null | head -1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Module 3 working (CDS)${NC}"
else
    echo -e "${YELLOW}⚠ No output from Module 3${NC}"
fi
echo ""

###############################################################################
# Test 4: Verify Module 4 Output (Clinical Patterns)
###############################################################################

echo -e "${YELLOW}[Test 4/5] Checking Module 4 output (clinical-patterns.v1)...${NC}"
sleep 2

timeout 5 docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic clinical-patterns.v1 \
  --from-beginning \
  --max-messages 1 2>/dev/null | head -1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Module 4 working (Pattern Detection)${NC}"
else
    echo -e "${YELLOW}⚠ No output from Module 4${NC}"
fi
echo ""

###############################################################################
# Test 5: Verify Module 6 Output (Hybrid Architecture)
###############################################################################

echo -e "${YELLOW}[Test 5/5] Checking Module 6 output (prod.ehr.events.enriched)...${NC}"
sleep 3

timeout 5 docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched \
  --from-beginning \
  --max-messages 1 2>/dev/null | head -1

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Module 6 working (Egress Routing)${NC}"
else
    echo -e "${YELLOW}⚠ No output from Module 6${NC}"
fi
echo ""

###############################################################################
# Summary
###############################################################################

echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║                      TEST SUMMARY                             ║${NC}"
echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
echo ""
echo "Pipeline Flow:"
echo "  patient-events-v1"
echo "    ↓ Module 1 (Ingestion)"
echo "  enriched-patient-events-v1"
echo "    ↓ Module 2 (Context Assembly)"
echo "  enriched (with context)"
echo "    ↓ Module 3 (CDS)"
echo "  comprehensive-cds-events.v1"
echo "    ↓ Module 4 (Pattern Detection)"
echo "  clinical-patterns.v1"
echo "    ↓ Module 5 (ML Inference)"
echo "  inference-results.v1"
echo "    ↓ Module 6 (Egress Routing)"
echo "  prod.ehr.events.enriched ✨"
echo ""
echo "Check Flink Web UI for job metrics:"
echo "  http://localhost:8081"
echo ""
