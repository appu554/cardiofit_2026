#!/bin/bash

# Create Hybrid Kafka Topics in Running Kafka Container
# This script adds the 7 hybrid topics to the existing Kafka setup

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🎯 Creating Hybrid Kafka Topics${NC}"
echo ""

# Function to create topic
create_topic() {
    local topic_name=$1
    local partitions=$2
    local retention_ms=$3
    local cleanup_policy=${4:-"delete"}
    local description=$5

    echo -e "${YELLOW}📦 Creating topic: ${topic_name}${NC}"
    echo "   Partitions: $partitions | Retention: $retention_ms ms | Policy: $cleanup_policy"
    echo "   Purpose: $description"

    if [ "$cleanup_policy" == "compact" ]; then
        docker exec manual-kafka kafka-topics \
            --bootstrap-server localhost:9092 \
            --create \
            --topic "$topic_name" \
            --partitions "$partitions" \
            --replication-factor 1 \
            --config retention.ms="$retention_ms" \
            --config cleanup.policy="$cleanup_policy" \
            --config min.compaction.lag.ms=3600000 \
            --if-not-exists
    else
        docker exec manual-kafka kafka-topics \
            --bootstrap-server localhost:9092 \
            --create \
            --topic "$topic_name" \
            --partitions "$partitions" \
            --replication-factor 1 \
            --config retention.ms="$retention_ms" \
            --config cleanup.policy="$cleanup_policy" \
            --if-not-exists
    fi

    echo -e "${GREEN}✅ Topic created: ${topic_name}${NC}"
    echo ""
}

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Phase 1: Central System of Record${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

create_topic "prod.ehr.events.enriched" 24 7776000000 "delete" \
    "Central system of record - Complete audit trail and replay capability"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Phase 2: Critical Action Topics${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

create_topic "prod.ehr.alerts.critical" 16 604800000 "delete" \
    "Critical alerts requiring immediate clinical attention"

create_topic "prod.ehr.fhir.upsert" 12 31536000000 "compact" \
    "FHIR resource state updates with latest patient data"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Phase 3: Supporting Systems${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

create_topic "prod.ehr.analytics.events" 32 15552000000 "delete" \
    "High-throughput analytics data for ClickHouse OLAP"

create_topic "prod.ehr.graph.mutations" 16 2592000000 "delete" \
    "Neo4j graph relationship updates and care pathways"

create_topic "prod.ehr.semantic.mesh" 8 7776000000 "compact" \
    "Knowledge base updates and semantic reasoning"

create_topic "prod.ehr.audit.logs" 8 220898664000 "delete" \
    "7-year audit retention for regulatory compliance"

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Source Topics for Flink Pipeline${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

create_topic "patient-events" 12 2592000000 "delete" \
    "Source: Patient demographic and encounter events"

create_topic "medication-events" 12 2592000000 "delete" \
    "Source: Medication orders, adherence, interactions"

create_topic "safety-events" 12 2592000000 "delete" \
    "Source: Safety violations and alert triggers"

create_topic "vital-signs-events" 12 2592000000 "delete" \
    "Source: Vital signs, observations, lab results"

create_topic "lab-result-events" 12 2592000000 "delete" \
    "Source: Laboratory results and diagnostic data"

echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}🎉 All Hybrid Topics Created Successfully!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo ""

# List all topics
echo -e "${BLUE}📋 Complete Topic List:${NC}"
docker exec manual-kafka kafka-topics --bootstrap-server localhost:9092 --list

echo ""
echo -e "${BLUE}🔍 Hybrid Topics Details:${NC}"
docker exec manual-kafka kafka-topics --bootstrap-server localhost:9092 --list | grep "prod.ehr" | while read topic; do
    echo -e "${YELLOW}Topic: $topic${NC}"
    docker exec manual-kafka kafka-topics --bootstrap-server localhost:9092 --describe --topic "$topic" | grep -E "(Topic:|PartitionCount:|ReplicationFactor:|Configs:)"
    echo ""
done

echo -e "${BLUE}🚀 Next Steps:${NC}"
echo "  1. Start Flink job: cd ../flink-processing && mvn clean package"
echo "  2. Deploy connectors: Update Kafka Connect configurations"
echo "  3. Test pipeline: Send test events to source topics"
echo ""
echo -e "${YELLOW}💡 Test Commands:${NC}"
echo "  • List topics:     docker exec manual-kafka kafka-topics --bootstrap-server localhost:9092 --list"
echo "  • Produce event:   docker exec -it manual-kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events"
echo "  • Consume events:  docker exec -it manual-kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic prod.ehr.events.enriched --from-beginning"