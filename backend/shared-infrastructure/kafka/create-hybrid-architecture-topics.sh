#!/bin/bash

# Create Kafka Topics for Hybrid Architecture
# This script creates the recommended hybrid topic architecture for the EHR Intelligence Engine

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Kafka configuration
KAFKA_BOOTSTRAP_SERVERS=${KAFKA_BOOTSTRAP_SERVERS:-"localhost:9092"}
REPLICATION_FACTOR=${REPLICATION_FACTOR:-3}

echo -e "${BLUE}🏗️ Creating Hybrid Kafka Topic Architecture${NC}"
echo "========================================================================"
echo -e "${YELLOW}Bootstrap Servers: ${KAFKA_BOOTSTRAP_SERVERS}${NC}"
echo -e "${YELLOW}Replication Factor: ${REPLICATION_FACTOR}${NC}"
echo ""

# Function to create a topic
create_topic() {
    local topic_name=$1
    local partitions=$2
    local retention_ms=$3
    local cleanup_policy=$4
    local description=$5

    echo -e "${BLUE}📊 Creating topic: ${topic_name}${NC}"
    echo -e "   Description: ${description}"
    echo -e "   Partitions: ${partitions}"
    echo -e "   Retention: $((retention_ms / 86400000)) days"
    echo -e "   Cleanup Policy: ${cleanup_policy}"

    # Check if topic already exists
    if kafka-topics --bootstrap-server ${KAFKA_BOOTSTRAP_SERVERS} --list | grep -q "^${topic_name}$"; then
        echo -e "${YELLOW}⚠️  Topic '${topic_name}' already exists. Skipping...${NC}"
    else
        # Create the topic
        kafka-topics --bootstrap-server ${KAFKA_BOOTSTRAP_SERVERS} \
            --create \
            --topic ${topic_name} \
            --partitions ${partitions} \
            --replication-factor ${REPLICATION_FACTOR} \
            --config retention.ms=${retention_ms} \
            --config cleanup.policy=${cleanup_policy} \
            --config compression.type=snappy \
            --config segment.ms=3600000 \
            --config min.insync.replicas=2

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ Created topic: ${topic_name}${NC}"
        else
            echo -e "${RED}❌ Failed to create topic: ${topic_name}${NC}"
            exit 1
        fi
    fi
    echo ""
}

# Function to create compacted topic
create_compacted_topic() {
    local topic_name=$1
    local partitions=$2
    local retention_ms=$3
    local description=$4

    echo -e "${BLUE}📊 Creating compacted topic: ${topic_name}${NC}"
    echo -e "   Description: ${description}"
    echo -e "   Partitions: ${partitions}"
    echo -e "   Retention: $((retention_ms / 86400000)) days"
    echo -e "   Cleanup Policy: compact"

    # Check if topic already exists
    if kafka-topics --bootstrap-server ${KAFKA_BOOTSTRAP_SERVERS} --list | grep -q "^${topic_name}$"; then
        echo -e "${YELLOW}⚠️  Topic '${topic_name}' already exists. Skipping...${NC}"
    else
        # Create the compacted topic
        kafka-topics --bootstrap-server ${KAFKA_BOOTSTRAP_SERVERS} \
            --create \
            --topic ${topic_name} \
            --partitions ${partitions} \
            --replication-factor ${REPLICATION_FACTOR} \
            --config cleanup.policy=compact \
            --config retention.ms=${retention_ms} \
            --config compression.type=snappy \
            --config min.compaction.lag.ms=3600000 \
            --config delete.retention.ms=86400000 \
            --config segment.ms=3600000 \
            --config min.insync.replicas=2

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ Created compacted topic: ${topic_name}${NC}"
        else
            echo -e "${RED}❌ Failed to create compacted topic: ${topic_name}${NC}"
            exit 1
        fi
    fi
    echo ""
}

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}PHASE 1: CENTRAL SYSTEM OF RECORD${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Central enriched events topic (90 days retention)
create_topic \
    "prod.ehr.events.enriched" \
    24 \
    7776000000 \
    "delete" \
    "Central system of record - Audit, replay, backfills, new consumer onboarding"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}PHASE 2: CRITICAL ACTION TOPICS${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Critical alerts topic (7 days retention)
create_topic \
    "prod.ehr.alerts.critical" \
    16 \
    604800000 \
    "delete" \
    "Actionable alerting - Latency-sensitive alerts requiring immediate action"

# FHIR upsert topic (365 days retention, compacted)
create_compacted_topic \
    "prod.ehr.fhir.upsert" \
    12 \
    31536000000 \
    "Stateful resource updates - Latest state of FHIR resources"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}PHASE 3: SUPPORTING SYSTEMS${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Analytics events topic (180 days retention)
create_topic \
    "prod.ehr.analytics.events" \
    32 \
    15552000000 \
    "delete" \
    "High-throughput analytics - Feeds OLAP systems like ClickHouse"

# Graph mutations topic (30 days retention)
create_topic \
    "prod.ehr.graph.mutations" \
    16 \
    2592000000 \
    "delete" \
    "Graph database ingestion - Batched commands to update Neo4j graph"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}SUPPORTING INFRASTRUCTURE${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Semantic mesh topic (365 days retention, compacted)
create_compacted_topic \
    "prod.ehr.semantic.mesh" \
    4 \
    31536000000 \
    "Reference data - Distributes clinical knowledge updates to Flink"

# Audit logs topic (7 years retention - 2555 days)
create_topic \
    "prod.ehr.audit.logs" \
    8 \
    220752000000 \
    "delete" \
    "Compliance & monitoring - Operational and regulatory audit trails"

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🎉 HYBRID ARCHITECTURE TOPICS CREATED SUCCESSFULLY!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# List all created topics
echo ""
echo -e "${BLUE}📋 Verifying created topics:${NC}"
kafka-topics --bootstrap-server ${KAFKA_BOOTSTRAP_SERVERS} --list | grep "^prod.ehr" | sort

echo ""
echo -e "${BLUE}📊 Topic Details:${NC}"
echo ""

# Show details for each topic
for topic in "prod.ehr.events.enriched" "prod.ehr.alerts.critical" "prod.ehr.fhir.upsert" \
             "prod.ehr.analytics.events" "prod.ehr.graph.mutations" \
             "prod.ehr.semantic.mesh" "prod.ehr.audit.logs"; do
    echo -e "${YELLOW}Topic: ${topic}${NC}"
    kafka-topics --bootstrap-server ${KAFKA_BOOTSTRAP_SERVERS} --describe --topic ${topic} 2>/dev/null | head -2
    echo ""
done

echo -e "${GREEN}✅ All topics created and verified!${NC}"
echo ""
echo -e "${BLUE}🚀 Next Steps:${NC}"
echo "1. Deploy the Flink EHR Intelligence Engine with TransactionalMultiSinkRouter"
echo "2. Configure Kafka Connect consumers for each downstream system"
echo "3. Set up monitoring for topic health and consumer lag"
echo "4. Test end-to-end data flow through the hybrid architecture"