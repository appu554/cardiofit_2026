#!/bin/bash

# ============================================================================
# Create Kafka Topics for Modules 1 & 2
# ============================================================================

set -e

KAFKA_CONTAINER="kafka"
BOOTSTRAP_SERVER="localhost:9092"

echo "📋 Creating Kafka Topics for Modules 1 & 2"
echo "==========================================="

# Module 1 Input Topics (Ingestion)
TOPICS=(
    "patient-events-v1"
    "medication-events-v1"
    "observation-events-v1"
    "vital-signs-events-v1"
    "lab-result-events-v1"
    "validated-device-data-v1"
)

# Module 1 Output / Module 2 Input Topics
TOPICS+=(
    "enriched-patient-events-v1"
    "validation-errors-v1"
)

# Module 2 Output Topics
TOPICS+=(
    "patient-context-snapshots-v1"
    "context-assembly-errors-v1"
)

echo ""
echo "Creating topics with 3 partitions, replication factor 1..."
echo ""

for topic in "${TOPICS[@]}"; do
    echo "📌 Creating topic: $topic"

    docker exec $KAFKA_CONTAINER kafka-topics \
        --create \
        --if-not-exists \
        --bootstrap-server localhost:9092 \
        --topic "$topic" \
        --partitions 3 \
        --replication-factor 1 \
        --config retention.ms=604800000 \
        --config segment.ms=86400000 2>&1 | grep -v "already exists" || true
done

echo ""
echo "✅ Topic Creation Complete!"
echo ""
echo "📊 Listing all topics:"
docker exec $KAFKA_CONTAINER kafka-topics \
    --list \
    --bootstrap-server localhost:9092 | grep -E "patient|medication|observation|vital|lab|validated|enriched|context|validation|errors"

echo ""
echo "🔍 Topic Details:"
for topic in "patient-events-v1" "enriched-patient-events-v1" "patient-context-snapshots-v1"; do
    echo ""
    echo "Topic: $topic"
    docker exec $KAFKA_CONTAINER kafka-topics \
        --describe \
        --bootstrap-server localhost:9092 \
        --topic "$topic" 2>/dev/null || echo "  (not found)"
done

echo ""
echo "✅ Kafka topics ready for Modules 1 & 2!"
