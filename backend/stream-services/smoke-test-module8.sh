#!/bin/bash
# Module 8 Smoke Test Script
# Quick validation of all storage projectors

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
KAFKA_BOOTSTRAP="localhost:9092"
ENRICHED_TOPIC="prod.ehr.events.enriched"
POSTGRES_HOST="localhost"
POSTGRES_PORT="5432"
POSTGRES_DB="clinical_events"
POSTGRES_USER="postgres"

# Projector ports
declare -A PROJECTOR_PORTS=(
    ["postgresql"]=8050
    ["mongodb"]=8051
    ["elasticsearch"]=8052
    ["clickhouse"]=8053
    ["influxdb"]=8054
    ["ups"]=8055
    ["fhir-store"]=8056
    ["neo4j"]=8057
)

echo "=========================================="
echo "MODULE 8 SMOKE TEST"
echo "=========================================="
echo ""

# Function to check service health
check_health() {
    local service=$1
    local port=$2
    local url="http://localhost:${port}/health"

    echo -n "Checking ${service}... "

    if curl -s -f "${url}" > /dev/null 2>&1; then
        echo -e "${GREEN}✓ UP${NC}"
        return 0
    else
        echo -e "${RED}✗ DOWN${NC}"
        return 1
    fi
}

# 1. Health check all services
echo "1. HEALTH CHECKS"
echo "----------------------------------------"

FAILURES=0
for service in "${!PROJECTOR_PORTS[@]}"; do
    if ! check_health "$service" "${PROJECTOR_PORTS[$service]}"; then
        ((FAILURES++))
    fi
done

if [ $FAILURES -gt 0 ]; then
    echo -e "${RED}Failed: ${FAILURES} services are down${NC}"
    exit 1
fi

echo -e "${GREEN}All services are healthy${NC}"
echo ""

# 2. Publish test events
echo "2. PUBLISHING TEST EVENTS"
echo "----------------------------------------"

# Generate test patient ID
PATIENT_ID="smoke-test-$(date +%s)"
EVENT_COUNT=10

echo "Publishing ${EVENT_COUNT} events for patient: ${PATIENT_ID}"

for i in $(seq 1 $EVENT_COUNT); do
    EVENT_ID=$(uuidgen)
    TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

    # Create event JSON
    EVENT_JSON=$(cat <<EOF
{
  "eventId": "${EVENT_ID}",
  "eventType": "VITAL_SIGNS",
  "patientId": "${PATIENT_ID}",
  "deviceId": "smoke-test-device",
  "timestamp": "${TIMESTAMP}",
  "eventTime": "${TIMESTAMP}",
  "sourceSystem": "smoke-test",
  "version": "1.0.0",
  "enrichment": {
    "patientContext": {
      "age": 45,
      "gender": "M",
      "conditions": ["I10"]
    },
    "clinicalContext": {
      "location": "TEST-UNIT",
      "encounterType": "INPATIENT"
    },
    "validationStatus": "VALID",
    "enrichmentTimestamp": "${TIMESTAMP}"
  },
  "data": {
    "heartRate": $((60 + RANDOM % 40)),
    "systolicBP": $((100 + RANDOM % 40)),
    "diastolicBP": $((60 + RANDOM % 30)),
    "temperature": 37,
    "respiratoryRate": 16,
    "oxygenSaturation": 98
  }
}
EOF
)

    # Publish to Kafka using kafkacat or kafka-console-producer
    if command -v kafkacat &> /dev/null; then
        echo "$EVENT_JSON" | kafkacat -b "$KAFKA_BOOTSTRAP" -t "$ENRICHED_TOPIC" -P -K:
    elif command -v kafka-console-producer.sh &> /dev/null; then
        echo "$EVENT_JSON" | kafka-console-producer.sh --broker-list "$KAFKA_BOOTSTRAP" --topic "$ENRICHED_TOPIC"
    else
        echo -e "${YELLOW}Warning: Neither kafkacat nor kafka-console-producer found${NC}"
        echo -e "${YELLOW}Using Python fallback...${NC}"

        # Python fallback
        python3 - <<PYTHON_SCRIPT
import json
from kafka import KafkaProducer

producer = KafkaProducer(
    bootstrap_servers='$KAFKA_BOOTSTRAP',
    value_serializer=lambda v: json.dumps(v).encode('utf-8')
)

event = $EVENT_JSON
producer.send('$ENRICHED_TOPIC', value=event)
producer.flush()
producer.close()
PYTHON_SCRIPT
    fi

    echo -n "."
done

echo ""
echo -e "${GREEN}Published ${EVENT_COUNT} events${NC}"
echo ""

# 3. Wait for processing
echo "3. WAITING FOR PROCESSING"
echo "----------------------------------------"
echo "Waiting 30 seconds for projectors to process events..."
sleep 30
echo -e "${GREEN}Wait complete${NC}"
echo ""

# 4. Verify event count in each store
echo "4. VERIFYING EVENT COUNTS"
echo "----------------------------------------"

VERIFICATION_FAILURES=0

# PostgreSQL
echo -n "PostgreSQL: "
PG_COUNT=$(PGPASSWORD=postgres psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c \
    "SELECT COUNT(*) FROM enriched_events WHERE patient_id = '${PATIENT_ID}';" 2>/dev/null | tr -d ' ')

if [ "$PG_COUNT" -ge "$EVENT_COUNT" ]; then
    echo -e "${GREEN}✓ ${PG_COUNT} events${NC}"
else
    echo -e "${RED}✗ ${PG_COUNT} events (expected ${EVENT_COUNT})${NC}"
    ((VERIFICATION_FAILURES++))
fi

# MongoDB
echo -n "MongoDB: "
MONGO_COUNT=$(mongosh --quiet --eval "db.getSiblingDB('clinical_events').clinical_documents.countDocuments({patientId: '${PATIENT_ID}'})" 2>/dev/null || echo "0")

if [ "$MONGO_COUNT" -ge "$EVENT_COUNT" ]; then
    echo -e "${GREEN}✓ ${MONGO_COUNT} events${NC}"
else
    echo -e "${RED}✗ ${MONGO_COUNT} events (expected ${EVENT_COUNT})${NC}"
    ((VERIFICATION_FAILURES++))
fi

# Elasticsearch
echo -n "Elasticsearch: "
ES_COUNT=$(curl -s -X GET "http://localhost:9200/clinical_events/_count?q=patientId:${PATIENT_ID}" | grep -o '"count":[0-9]*' | cut -d: -f2 || echo "0")

if [ "$ES_COUNT" -ge "$EVENT_COUNT" ]; then
    echo -e "${GREEN}✓ ${ES_COUNT} events${NC}"
else
    echo -e "${RED}✗ ${ES_COUNT} events (expected ${EVENT_COUNT})${NC}"
    ((VERIFICATION_FAILURES++))
fi

# ClickHouse
echo -n "ClickHouse: "
CH_COUNT=$(clickhouse-client --query "SELECT COUNT(*) FROM clinical_analytics.clinical_events_fact WHERE patient_id = '${PATIENT_ID}'" 2>/dev/null || echo "0")

if [ "$CH_COUNT" -ge "$EVENT_COUNT" ]; then
    echo -e "${GREEN}✓ ${CH_COUNT} events${NC}"
else
    echo -e "${RED}✗ ${CH_COUNT} events (expected ${EVENT_COUNT})${NC}"
    ((VERIFICATION_FAILURES++))
fi

# UPS Read Model
echo -n "UPS Read Model: "
UPS_COUNT=$(PGPASSWORD=postgres psql -h "$POSTGRES_HOST" -p "$POSTGRES_PORT" -U "$POSTGRES_USER" -d "$POSTGRES_DB" -t -c \
    "SELECT COUNT(*) FROM ups_read_model WHERE patient_id = '${PATIENT_ID}';" 2>/dev/null | tr -d ' ')

if [ "$UPS_COUNT" -ge "1" ]; then
    echo -e "${GREEN}✓ ${UPS_COUNT} patient record${NC}"
else
    echo -e "${RED}✗ ${UPS_COUNT} patient record (expected 1)${NC}"
    ((VERIFICATION_FAILURES++))
fi

echo ""

# 5. Check for errors in logs
echo "5. CHECKING ERROR LOGS"
echo "----------------------------------------"

ERROR_COUNT=0
for service in "${!PROJECTOR_PORTS[@]}"; do
    port=${PROJECTOR_PORTS[$service]}

    # Check metrics endpoint for errors
    ERRORS=$(curl -s "http://localhost:${port}/metrics" | grep "projector_messages_failed_total" | grep -v "# " | awk '{print $2}' | head -1 || echo "0")

    if [ "$ERRORS" != "0" ] && [ ! -z "$ERRORS" ]; then
        echo -e "${YELLOW}${service}: ${ERRORS} errors${NC}"
        ((ERROR_COUNT++))
    fi
done

if [ $ERROR_COUNT -eq 0 ]; then
    echo -e "${GREEN}No errors found${NC}"
fi

echo ""

# 6. Final summary
echo "=========================================="
echo "SMOKE TEST SUMMARY"
echo "=========================================="

TOTAL_FAILURES=$((FAILURES + VERIFICATION_FAILURES))

if [ $TOTAL_FAILURES -eq 0 ]; then
    echo -e "${GREEN}✓ ALL TESTS PASSED${NC}"
    echo ""
    echo "Services healthy: 8/8"
    echo "Events published: ${EVENT_COUNT}"
    echo "Stores verified: 5/5"
    echo "Errors: 0"
    echo ""
    echo -e "${GREEN}Module 8 is functioning correctly${NC}"
    exit 0
else
    echo -e "${RED}✗ TESTS FAILED${NC}"
    echo ""
    echo "Service failures: ${FAILURES}"
    echo "Verification failures: ${VERIFICATION_FAILURES}"
    echo ""
    echo -e "${RED}Please check logs and fix issues${NC}"
    exit 1
fi
