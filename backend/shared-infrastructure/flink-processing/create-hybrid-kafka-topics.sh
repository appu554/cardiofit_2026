#!/bin/bash

# Create Hybrid Kafka Topic Architecture for Module 6 → Module 8
# These topics bridge the Flink processing layer (Module 6) with storage projectors (Module 8)

set -e

# Configuration
KAFKA_BROKER="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
REPLICATION="${REPLICATION_FACTOR:-1}"

echo "=================================================="
echo "Creating Hybrid Kafka Topic Architecture"
echo "=================================================="
echo "Kafka Broker: $KAFKA_BROKER"
echo "Replication Factor: $REPLICATION"
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

    kafka-topics --create \
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
        echo -e "${GREEN}✓ Topic created successfully${NC}"
    else
        echo -e "${GREEN}✓ Topic already exists${NC}"
    fi
    echo ""
}

# ============================================================================
# PHASE 1: Central System of Record
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}PHASE 1: Central System of Record${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "prod.ehr.events.enriched" \
    24 \
    90 \
    "Central enriched event stream - All clinical events with ML predictions, semantic annotations, and context" \
    false

echo -e "${GREEN}✓ Phase 1 Complete${NC}"
echo ""

# ============================================================================
# PHASE 2: Critical Action Topics
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}PHASE 2: Critical Action Topics${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "prod.ehr.alerts.critical" \
    16 \
    7 \
    "Critical clinical alerts requiring immediate action (HIGH + CRITICAL severity)" \
    false

create_topic \
    "prod.ehr.fhir.upsert" \
    12 \
    365 \
    "FHIR resource upserts for Google Healthcare API - Compacted for state management" \
    true

echo -e "${GREEN}✓ Phase 2 Complete${NC}"
echo ""

# ============================================================================
# PHASE 3: Supporting Systems
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}PHASE 3: Supporting Systems${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "prod.ehr.analytics.events" \
    32 \
    180 \
    "High-throughput analytics events for ClickHouse and business intelligence" \
    false

create_topic \
    "prod.ehr.graph.mutations" \
    16 \
    30 \
    "Neo4j graph database mutations for patient journey and clinical pathways" \
    false

echo -e "${GREEN}✓ Phase 3 Complete${NC}"
echo ""

# ============================================================================
# SUPPORTING INFRASTRUCTURE
# ============================================================================

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Supporting Infrastructure Topics${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo ""

create_topic \
    "prod.ehr.semantic.mesh" \
    4 \
    365 \
    "Semantic mesh updates - Knowledge graph changes (compacted)" \
    true

create_topic \
    "prod.ehr.audit.logs" \
    8 \
    2555 \
    "Compliance audit logs - 7-year retention for regulatory compliance" \
    false

echo -e "${GREEN}✓ Supporting Infrastructure Complete${NC}"
echo ""

# ============================================================================
# VERIFICATION
# ============================================================================

echo "=================================================="
echo "Verifying Hybrid Topics Created..."
echo "=================================================="

kafka-topics --list --bootstrap-server "$KAFKA_BROKER" | grep "^prod\.ehr\." | while read topic; do
    # Get topic details
    details=$(kafka-topics --describe --topic "$topic" --bootstrap-server "$KAFKA_BROKER" 2>/dev/null | grep "PartitionCount")
    echo -e "${GREEN}✓ $topic${NC}"
    echo "  $details"
done

echo ""
echo "=================================================="
echo "🎉 Hybrid Kafka Topic Architecture Complete!"
echo "=================================================="
echo ""
echo "📊 Topics Created:"
echo "  • prod.ehr.events.enriched (24 partitions, 90d) - Central stream"
echo "  • prod.ehr.fhir.upsert (12 partitions, 365d, compacted) - FHIR state"
echo "  • prod.ehr.graph.mutations (16 partitions, 30d) - Neo4j updates"
echo "  • prod.ehr.alerts.critical (16 partitions, 7d) - Urgent alerts"
echo "  • prod.ehr.analytics.events (32 partitions, 180d) - Analytics"
echo "  • prod.ehr.semantic.mesh (4 partitions, 365d, compacted) - Knowledge"
echo "  • prod.ehr.audit.logs (8 partitions, 2555d/7y) - Compliance"
echo ""
echo "🔗 Module 6 → Module 8 Data Flow:"
echo "  Flink (Module 6) → Hybrid Topics → Storage Projectors (Module 8)"
echo ""
echo "📋 Next Steps:"
echo "  1. Deploy Module 6: ./deploy-module6.sh"
echo "  2. Start Module 8 Projectors: cd backend/stream-services && ./start-module8-projectors.sh"
echo "  3. Monitor data flow: kafka-console-consumer --topic prod.ehr.events.enriched"
echo ""
echo "💡 To verify Module 6 is writing:"
echo "  docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \\"
echo "    --broker-list localhost:9092 \\"
echo "    --topic prod.ehr.events.enriched \\"
echo "    --time -1"
echo ""
