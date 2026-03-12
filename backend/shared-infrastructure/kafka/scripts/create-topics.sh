#!/bin/bash

# CardioFit Platform - Kafka Topic Creation Script
# Creates all 68 topics with proper configuration
# Reference: KAFKA_TOPICS_REFERENCE.md

set -e

echo "==========================================="
echo "CardioFit Kafka Topic Initialization"
echo "Creating 68 topics across 11 categories"
echo "==========================================="

# Kafka connection settings
KAFKA_BROKERS="kafka1:29092,kafka2:29093,kafka3:29094"
REPLICATION_FACTOR=3
MIN_ISR=2

# Wait for Kafka to be ready
echo "⏳ Waiting for Kafka cluster to be ready..."
sleep 10

# Function to create a topic
create_topic() {
    local TOPIC_NAME=$1
    local PARTITIONS=$2
    local RETENTION_MS=$3
    local CLEANUP_POLICY=$4
    local COMPRESSION=$5
    local MAX_MESSAGE_BYTES=$6
    local MIN_ISR_OVERRIDE=$7

    echo "📝 Creating topic: $TOPIC_NAME (partitions: $PARTITIONS, retention: $RETENTION_MS ms)"

    CONFIG_OPTIONS="--config retention.ms=$RETENTION_MS"
    CONFIG_OPTIONS="$CONFIG_OPTIONS --config cleanup.policy=$CLEANUP_POLICY"
    CONFIG_OPTIONS="$CONFIG_OPTIONS --config compression.type=$COMPRESSION"
    CONFIG_OPTIONS="$CONFIG_OPTIONS --config min.insync.replicas=${MIN_ISR_OVERRIDE:-$MIN_ISR}"

    if [ ! -z "$MAX_MESSAGE_BYTES" ]; then
        CONFIG_OPTIONS="$CONFIG_OPTIONS --config max.message.bytes=$MAX_MESSAGE_BYTES"
    fi

    kafka-topics --bootstrap-server $KAFKA_BROKERS \
        --create \
        --if-not-exists \
        --topic $TOPIC_NAME \
        --partitions $PARTITIONS \
        --replication-factor $REPLICATION_FACTOR \
        $CONFIG_OPTIONS || echo "⚠️  Topic $TOPIC_NAME already exists or error occurred"
}

echo ""
echo "============================================"
echo "1. CLINICAL EVENTS TOPICS (9 topics)"
echo "============================================"

create_topic "patient-events.v1" 12 259200000 "delete" "snappy" "" 2
create_topic "medication-events.v1" 12 259200000 "delete" "snappy" "" 2
create_topic "observation-events.v1" 12 259200000 "delete" "snappy" "" 2
create_topic "safety-events.v1" 12 604800000 "delete" "snappy" "10485760" 2
create_topic "vital-signs-events.v1" 12 259200000 "delete" "snappy" "" 2
create_topic "lab-result-events.v1" 12 604800000 "delete" "snappy" "" 2
create_topic "encounter-events.v1" 8 259200000 "delete" "snappy" "" 2
create_topic "diagnostic-events.v1" 8 604800000 "delete" "snappy" "" 2
create_topic "procedure-events.v1" 8 604800000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "2. DEVICE DATA TOPICS (4 topics)"
echo "============================================"

create_topic "raw-device-data.v1" 12 259200000 "delete" "lz4" "" 2
create_topic "validated-device-data.v1" 12 259200000 "delete" "snappy" "" 2
create_topic "waveform-data.v1" 24 86400000 "delete" "lz4" "" 2
create_topic "device-telemetry.v1" 4 604800000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "3. RUNTIME LAYER TOPICS (5 topics)"
echo "============================================"

create_topic "enriched-patient-events.v1" 12 604800000 "delete" "snappy" "" 2
create_topic "clinical-patterns.v1" 8 2592000000 "delete" "snappy" "" 2
create_topic "pathway-adherence-events.v1" 8 2592000000 "delete" "snappy" "" 2
create_topic "semantic-mesh-updates.v1" 4 604800000 "compact" "snappy" "" 2
create_topic "patient-context-snapshots.v1" 12 604800000 "compact" "snappy" "" 2

echo ""
echo "============================================"
echo "4. KNOWLEDGE BASE CDC TOPICS (8 topics)"
echo "============================================"

create_topic "kb3.clinical_protocols.changes" 4 604800000 "compact" "snappy" "" 2
create_topic "kb4.drug_calculations.changes" 4 604800000 "compact" "snappy" "" 2
create_topic "kb4.dosing_rules.changes" 4 604800000 "compact" "snappy" "" 2
create_topic "kb4.weight_adjustments.changes" 4 604800000 "compact" "snappy" "" 2
create_topic "kb5.drug_interactions.changes" 4 604800000 "compact" "snappy" "" 2
create_topic "kb6.validation_rules.changes" 4 604800000 "compact" "snappy" "" 2
create_topic "kb7.terminology.changes" 4 604800000 "compact" "snappy" "" 2
create_topic "semantic-mesh.changes" 4 604800000 "compact" "snappy" "" 2

echo ""
echo "============================================"
echo "5. EVIDENCE MANAGEMENT TOPICS (6 topics)"
echo "============================================"

create_topic "audit-events.v1" 6 31536000000 "delete" "gzip" "" 3
create_topic "envelope-events.v1" 6 7776000000 "delete" "snappy" "" 2
create_topic "evidence-requests.v1" 4 604800000 "delete" "snappy" "" 2
create_topic "evidence-validations.v1" 4 2592000000 "delete" "snappy" "" 2
create_topic "clinical-reasoning-events.v1" 8 2592000000 "delete" "snappy" "" 2
create_topic "inference-results.v1" 8 2592000000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "6. WORKFLOW & ORCHESTRATION TOPICS (6 topics)"
echo "============================================"

create_topic "workflow-events.v1" 8 604800000 "delete" "snappy" "" 2
create_topic "workflow-ui-interactions.v1" 8 259200000 "delete" "snappy" "" 2
create_topic "clinical-overrides.v1" 4 7776000000 "delete" "snappy" "" 2
create_topic "task-assignments.v1" 8 604800000 "delete" "snappy" "" 2
create_topic "decision-support-events.v1" 8 2592000000 "delete" "snappy" "" 2
create_topic "orchestration-commands.v1" 4 259200000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "7. SLA & MONITORING TOPICS (6 topics)"
echo "============================================"

create_topic "sla-measurements.v1" 8 604800000 "delete" "snappy" "" 2
create_topic "sla-violations.v1" 4 2592000000 "delete" "snappy" "" 2
create_topic "performance-metrics.v1" 8 604800000 "delete" "lz4" "" 2
create_topic "clinical-metrics.v1" 6 7776000000 "delete" "snappy" "" 2
create_topic "usage-analytics.v1" 4 2592000000 "delete" "snappy" "" 2
create_topic "alert-notifications.v1" 6 604800000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "8. CACHE & OPTIMIZATION TOPICS (4 topics)"
echo "============================================"

create_topic "cache-invalidation.v1" 8 86400000 "delete" "snappy" "" 2
create_topic "prefetch-predictions.v1" 4 259200000 "delete" "snappy" "" 2
create_topic "cache-warmup.v1" 4 86400000 "delete" "snappy" "" 2
create_topic "query-patterns.v1" 4 604800000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "9. DEAD LETTER QUEUE TOPICS (9 topics)"
echo "============================================"

create_topic "failed-validation.v1" 4 2592000000 "delete" "snappy" "" 2
create_topic "critical-data-dlq.v1" 4 7776000000 "delete" "snappy" "" 3
create_topic "poison-messages.v1" 2 31536000000 "delete" "snappy" "" 2
create_topic "sink-write-failures.v1" 6 1209600000 "delete" "snappy" "" 2
create_topic "critical-sink-failures.v1" 4 7776000000 "delete" "snappy" "" 3
create_topic "poison-messages-stage2.v1" 2 31536000000 "delete" "snappy" "" 2
create_topic "processing-errors.v1" 4 604800000 "delete" "snappy" "" 2
create_topic "integration-failures.v1" 4 2592000000 "delete" "snappy" "" 2
create_topic "flink-checkpoint-failures.v1" 2 604800000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "10. REAL-TIME COLLABORATION TOPICS (5 topics)"
echo "============================================"

create_topic "clinical-chat.v1" 8 604800000 "delete" "snappy" "" 2
create_topic "notification-push.v1" 8 259200000 "delete" "snappy" "" 2
create_topic "presence-updates.v1" 4 86400000 "compact" "snappy" "" 2
create_topic "collaboration-events.v1" 6 259200000 "delete" "snappy" "" 2
create_topic "graphql-subscriptions.v1" 8 86400000 "delete" "snappy" "" 2

echo ""
echo "============================================"
echo "11. EXTERNAL INTEGRATION TOPICS (6 topics)"
echo "============================================"

create_topic "hl7-messages.v1" 8 604800000 "delete" "snappy" "1048576" 2
create_topic "fhir-bundles.v1" 8 604800000 "delete" "snappy" "10485760" 2
create_topic "external-lab-results.v1" 6 2592000000 "delete" "snappy" "1048576" 2
create_topic "pharmacy-orders.v1" 6 604800000 "delete" "snappy" "1048576" 2
create_topic "billing-events.v1" 4 7776000000 "delete" "snappy" "1048576" 2
create_topic "google-healthcare-sync.v1" 6 259200000 "delete" "snappy" "1048576" 2

echo ""
echo "============================================"
echo "TOPIC CREATION COMPLETE"
echo "============================================"

# List all created topics
echo ""
echo "📊 Verifying created topics..."
echo ""
kafka-topics --bootstrap-server $KAFKA_BROKERS --list | sort | while read topic; do
    if [[ $topic == *".v1" ]] || [[ $topic == *".changes" ]]; then
        echo "✅ $topic"
    fi
done

# Count topics
TOPIC_COUNT=$(kafka-topics --bootstrap-server $KAFKA_BROKERS --list | grep -E '\.(v1|changes)$' | wc -l)
echo ""
echo "============================================"
echo "✅ Successfully created $TOPIC_COUNT topics"
echo "============================================"
echo ""
echo "Access Kafka UI at:"
echo "  - Kafka UI: http://localhost:8080"
echo "  - Kafdrop: http://localhost:9000"
echo "  - Schema Registry: http://localhost:8081"
echo "  - KSQL DB: http://localhost:8088"
echo ""
echo "Kafka brokers available at:"
echo "  - Broker 1: localhost:9092"
echo "  - Broker 2: localhost:9093"
echo "  - Broker 3: localhost:9094"
echo ""
echo "============================================"