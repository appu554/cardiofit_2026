#!/bin/bash

# ============================================================================
# Create Kafka Topics for Ingestion Service + KB Threshold Hot-Swap
# ============================================================================
# Topics defined per Vaidshala_KafkaTopicArchitecture.docx (Layer 1: Ingestion Events)
# Partition counts target the Growth tier (10K-100K patients).
# Retention policies follow the three-tier model:
#   Hot  (7-30d)  — active clinical data consumed by Flink + KB services
#   Warm (90-180d) — regulatory retention for audit, insurance, ABDM compliance
#
# All topics use patient_id as the Kafka message key (partition key) to
# guarantee per-patient temporal ordering in every partition.
# ============================================================================

set -e

# Configuration
KAFKA_CONTAINER="${KAFKA_CONTAINER:-cardiofit-kafka-lite}"
KAFKA_BROKER="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
REPLICATION="${REPLICATION_FACTOR:-1}"

# Detect execution mode: Docker (default for dev) or direct CLI (for staging/prod)
if [ "${KAFKA_DIRECT:-false}" = "true" ]; then
    KAFKA_CMD="kafka-topics"
    KAFKA_CFG_CMD="kafka-configs"
else
    KAFKA_CMD="docker exec $KAFKA_CONTAINER kafka-topics"
    KAFKA_CFG_CMD="docker exec $KAFKA_CONTAINER kafka-configs"
fi

echo "=================================================="
echo "Creating Ingestion Service Kafka Topics"
echo "=================================================="
echo "Kafka Container: $KAFKA_CONTAINER"
echo "Kafka Broker: $KAFKA_BROKER"
echo "Replication Factor: $REPLICATION"
echo "Mode: $([ "${KAFKA_DIRECT:-false}" = "true" ] && echo "Direct CLI" || echo "Docker exec")"
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
RED='\033[0;31m'
NC='\033[0m'

# Function to create topic with specific configuration
create_topic() {
    local topic=$1
    local partitions=$2
    local retention_days=$3
    local description=$4
    local compacted=${5:-false}

    echo -e "${YELLOW}Creating topic: ${topic}${NC}"
    echo "  Description: $description"
    echo "  Partitions: $partitions"
    echo "  Retention: $retention_days days"
    echo "  Compacted: $compacted"

    # Calculate retention in milliseconds
    local retention_ms=$((retention_days * 24 * 60 * 60 * 1000))

    # Build cleanup policy
    local cleanup_policy="delete"
    if [ "$compacted" = "true" ]; then
        cleanup_policy="compact,delete"
    fi

    $KAFKA_CMD --create \
        --topic "$topic" \
        --bootstrap-server "$KAFKA_BROKER" \
        --partitions "$partitions" \
        --replication-factor "$REPLICATION" \
        --config retention.ms=$retention_ms \
        --config compression.type=snappy \
        --config cleanup.policy=$cleanup_policy \
        --config min.compaction.lag.ms=3600000 \
        --config max.compaction.lag.ms=86400000 \
        --if-not-exists 2>/dev/null

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}  ✓ Topic created successfully${NC}"
    else
        echo -e "${GREEN}  ✓ Topic already exists${NC}"
    fi
    echo ""
}

# ============================================================================
# LAYER 1: Ingestion Events (Source → Kafka via Transactional Outbox)
# ============================================================================
# Events published by the Ingestion Service via Global Outbox SDK.
# One topic per clinical data category. Each topic has its own retention,
# partition count, and consumer group configuration tuned to its clinical
# semantics and volume profile.
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}LAYER 1: Ingestion Events${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "ingestion.labs" \
    12 \
    90 \
    "Lab results: FBG, HbA1c, eGFR, lipids, UACR, K+, creatinine. LOINC-coded FHIR Observations. Critical values (eGFR<30, K+>6.0) ALSO published to ingestion.safety-critical." \
    false

create_topic \
    "ingestion.vitals" \
    8 \
    30 \
    "BP (systolic+diastolic), HR, SpO2. From devices and self-reports. SBP>180 or DBP>120 ALSO to ingestion.safety-critical." \
    false

create_topic \
    "ingestion.device-data" \
    8 \
    30 \
    "BLE device readings: glucometer, BP monitor, pulse oximeter, weight scale. IEEE 11073 coded. Glucometer >300 ALSO to safety-critical." \
    false

create_topic \
    "ingestion.patient-reported" \
    8 \
    30 \
    "FBG, PPBG, steps, meal quality, medication adherence, waist, weight from app checkins and WhatsApp Tier-1. FBG>300 ALSO to safety-critical." \
    false

create_topic \
    "ingestion.wearable-aggregates" \
    4 \
    14 \
    "Daily aggregated wearable data: steps, sleep, resting HR, CGM TIR/TAR/TBR/CV/MAG/Mean. Background priority — shed first under circuit breaker." \
    false

create_topic \
    "ingestion.cgm-raw" \
    4 \
    7 \
    "Raw 5-minute CGM glucose readings. 288 readings/day/patient. Compacted for dedup. Consumed ONLY by Flink CGM aggregation job." \
    true

create_topic \
    "ingestion.abdm-records" \
    4 \
    180 \
    "FHIR Bundles from ABDM HIE: OPConsultRecords, DischargeSummaries, DiagnosticReports, Prescriptions. 180d retention for ABDM consent audit." \
    false

create_topic \
    "ingestion.medications" \
    8 \
    90 \
    "Medication observations from all sources. 90d retention for medication reconciliation and insurance claims." \
    false

create_topic \
    "ingestion.observations" \
    8 \
    30 \
    "General/fallback observation topic for unmapped observation types. Acts as catch-all." \
    false

echo -e "${GREEN}✓ Ingestion source topics complete${NC}"
echo ""

# ============================================================================
# SAFETY-CRITICAL: The Red Phone
# ============================================================================
# Dedicated topic for critical values detected at source. KB-22 has a
# dedicated consumer thread with 10ms poll interval — processes ONLY this
# topic. Never mix with routine data.
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${RED}SAFETY-CRITICAL PATH${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "ingestion.safety-critical" \
    4 \
    90 \
    "Critical values: eGFR<30, FBG>300, K+>6.0, SBP>180, HARD_STOP triggers. Dedicated consumer with 10ms poll. THIS IS THE RED PHONE." \
    false

echo -e "${GREEN}✓ Safety-critical topic complete${NC}"
echo ""

# ============================================================================
# KB THRESHOLD HOT-SWAP
# ============================================================================
# Compacted topic for broadcasting threshold configuration changes from KB
# services to Flink operators via BroadcastState pattern.
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}KB THRESHOLD HOT-SWAP${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "kb.clinical-thresholds.changes" \
    1 \
    7 \
    "Clinical threshold config changes from KB-4/KB-20/KB-23. Compacted for latest-value semantics. Consumed by Flink BroadcastState." \
    true

echo -e "${GREEN}✓ KB threshold topic complete${NC}"
echo ""

# ============================================================================
# DEAD LETTER QUEUES (Ingestion Domain)
# ============================================================================
# DLQ topics for failed ingestion events. 90-day retention for investigation
# and replay. Partition count lower than source (4) since DLQ volume is
# expected to be <1% of source volume.
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}DEAD LETTER QUEUES${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "dlq.ingestion.labs.v1" \
    4 \
    90 \
    "DLQ for ingestion.labs — failed lab observations. 90d for investigation and replay." \
    false

create_topic \
    "dlq.ingestion.vitals.v1" \
    4 \
    90 \
    "DLQ for ingestion.vitals — failed vital sign observations." \
    false

create_topic \
    "dlq.ingestion.safety-critical.v1" \
    4 \
    90 \
    "DLQ for ingestion.safety-critical — MUST be investigated promptly. Safety-critical failures are clinical incidents." \
    false

echo -e "${GREEN}✓ DLQ topics complete${NC}"
echo ""

# ============================================================================
# VERIFICATION
# ============================================================================

echo "=================================================="
echo "Verifying Ingestion Topics..."
echo "=================================================="

EXPECTED_TOPICS=(
    "ingestion.labs"
    "ingestion.vitals"
    "ingestion.device-data"
    "ingestion.patient-reported"
    "ingestion.wearable-aggregates"
    "ingestion.cgm-raw"
    "ingestion.abdm-records"
    "ingestion.medications"
    "ingestion.observations"
    "ingestion.safety-critical"
    "kb.clinical-thresholds.changes"
    "dlq.ingestion.labs.v1"
    "dlq.ingestion.vitals.v1"
    "dlq.ingestion.safety-critical.v1"
)

MISSING=0
for topic in "${EXPECTED_TOPICS[@]}"; do
    if $KAFKA_CMD --describe --topic "$topic" --bootstrap-server "$KAFKA_BROKER" &>/dev/null; then
        details=$($KAFKA_CMD --describe --topic "$topic" --bootstrap-server "$KAFKA_BROKER" 2>/dev/null | head -1)
        echo -e "${GREEN}✓ $topic${NC}"
        echo "  $details"
    else
        echo -e "${RED}✗ MISSING: $topic${NC}"
        MISSING=$((MISSING + 1))
    fi
done

echo ""
if [ $MISSING -eq 0 ]; then
    echo -e "${GREEN}=================================================="
    echo "All 14 topics created successfully!"
    echo "==================================================${NC}"
else
    echo -e "${RED}=================================================="
    echo "WARNING: $MISSING topic(s) missing!"
    echo "==================================================${NC}"
    exit 1
fi

echo ""
echo "Topic Summary:"
echo "  Layer 1 (Ingestion Events):  10 topics"
echo "  KB Threshold Hot-Swap:        1 topic"
echo "  Dead Letter Queues:           3 topics"
echo "  Total:                       14 topics"
echo ""
echo "Consumer Group Assignments (per Kafka Architecture Doc §5):"
echo "  cg-kb20-state:     ingestion.labs, .vitals, .patient-reported, .device-data, .wearable-aggregates, .abdm-records"
echo "  cg-kb22-monitor:   ingestion.labs, .vitals, .safety-critical (10ms poll thread)"
echo "  cg-kb26-twin:      ingestion.labs, .vitals, .wearable-aggregates, .device-data"
echo "  cg-flink-module1b: ALL ingestion.* (except .safety-critical)"
echo "  cg-flink-cgm:      ingestion.cgm-raw"
echo "  cg-notification:   ingestion.safety-critical"
echo ""
echo "Next Steps:"
echo "  1. Verify consumer groups are configured in each service"
echo "  2. Enable OUTBOX_ENABLED=true in ingestion service"
echo "  3. Deploy Module1b Flink job: flink-module1b-ingestion"
echo ""
