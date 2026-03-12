#!/bin/bash

################################################################################
# Kafka Topic Creation Script for Clinical Recommendation Engine
#
# Creates 4 Kafka topics for routing clinical recommendations by urgency level:
# - clinical-recommendations-critical (CRITICAL priority)
# - clinical-recommendations-high (HIGH priority)
# - clinical-recommendations-medium (MEDIUM priority)
# - clinical-recommendations-routine (LOW/ROUTINE priority)
#
# Usage:
#   ./create-recommendation-topics.sh [kafka-bootstrap-servers]
#
# Example:
#   ./create-recommendation-topics.sh localhost:9092
#   ./create-recommendation-topics.sh kafka1:29092,kafka2:29093,kafka3:29094
#
# Requirements:
#   - Kafka CLI tools (kafka-topics.sh) must be in PATH
#   - Network access to Kafka cluster
#
# Author: Module 3 Clinical Recommendation Engine
# Version: 1.0
# Date: 2025-10-20
################################################################################

set -e  # Exit on any error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default Kafka bootstrap servers
KAFKA_BOOTSTRAP_SERVERS="${1:-localhost:9092}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Clinical Recommendation Topics Setup${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "Kafka Bootstrap Servers: ${YELLOW}${KAFKA_BOOTSTRAP_SERVERS}${NC}"
echo ""

# Function to create a Kafka topic
create_topic() {
    local topic_name=$1
    local partitions=$2
    local replication=$3
    local retention_days=$4
    local description=$5

    echo -e "${BLUE}Creating topic: ${YELLOW}${topic_name}${NC}"
    echo "  Partitions: ${partitions}"
    echo "  Replication Factor: ${replication}"
    echo "  Retention: ${retention_days} days"
    echo "  Description: ${description}"

    # Calculate retention in milliseconds
    local retention_ms=$((retention_days * 24 * 60 * 60 * 1000))

    # Create topic with configuration
    kafka-topics.sh --create \
        --bootstrap-server "${KAFKA_BOOTSTRAP_SERVERS}" \
        --topic "${topic_name}" \
        --partitions "${partitions}" \
        --replication-factor "${replication}" \
        --config compression.type=lz4 \
        --config cleanup.policy=delete \
        --config retention.ms="${retention_ms}" \
        --config min.insync.replicas=2 \
        --config segment.ms=86400000 \
        --if-not-exists

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Topic '${topic_name}' created successfully${NC}"
    else
        echo -e "${RED}✗ Failed to create topic '${topic_name}'${NC}"
        return 1
    fi
    echo ""
}

# Function to verify topic creation
verify_topic() {
    local topic_name=$1

    echo -e "${BLUE}Verifying topic: ${YELLOW}${topic_name}${NC}"

    kafka-topics.sh --describe \
        --bootstrap-server "${KAFKA_BOOTSTRAP_SERVERS}" \
        --topic "${topic_name}" 2>/dev/null

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Topic '${topic_name}' verified${NC}"
        return 0
    else
        echo -e "${RED}✗ Topic '${topic_name}' not found${NC}"
        return 1
    fi
    echo ""
}

# Check if Kafka is accessible
echo -e "${BLUE}Checking Kafka connectivity...${NC}"
kafka-topics.sh --list --bootstrap-server "${KAFKA_BOOTSTRAP_SERVERS}" >/dev/null 2>&1

if [ $? -ne 0 ]; then
    echo -e "${RED}✗ Cannot connect to Kafka at ${KAFKA_BOOTSTRAP_SERVERS}${NC}"
    echo -e "${YELLOW}Please verify:${NC}"
    echo "  1. Kafka is running"
    echo "  2. Bootstrap servers address is correct"
    echo "  3. Network connectivity to Kafka cluster"
    exit 1
fi

echo -e "${GREEN}✓ Kafka connection successful${NC}"
echo ""

# Create topics
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Creating Recommendation Topics${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Topic 1: CRITICAL recommendations
# High throughput, short retention, high replication for reliability
create_topic \
    "clinical-recommendations-critical" \
    3 \
    3 \
    7 \
    "CRITICAL priority recommendations (multi-channel: SMS, Pager, Push, Email, EHR, Dashboard)"

# Topic 2: HIGH recommendations
# Moderate throughput, short retention
create_topic \
    "clinical-recommendations-high" \
    3 \
    3 \
    7 \
    "HIGH priority recommendations (Push, Email, EHR, Dashboard with highlighting)"

# Topic 3: MEDIUM recommendations
# Moderate throughput, medium retention
create_topic \
    "clinical-recommendations-medium" \
    2 \
    3 \
    14 \
    "MEDIUM priority recommendations (Email, EHR, Dashboard)"

# Topic 4: ROUTINE recommendations
# Lower throughput, longer retention
create_topic \
    "clinical-recommendations-routine" \
    1 \
    3 \
    30 \
    "ROUTINE/LOW priority recommendations (EHR, Dashboard silent notification)"

# Verify all topics
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Verifying Topic Creation${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

verify_topic "clinical-recommendations-critical"
verify_topic "clinical-recommendations-high"
verify_topic "clinical-recommendations-medium"
verify_topic "clinical-recommendations-routine"

# Summary
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Topic Creation Summary${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""
echo -e "${GREEN}✓ All 4 recommendation topics created successfully${NC}"
echo ""
echo -e "${YELLOW}Topics:${NC}"
echo "  1. clinical-recommendations-critical (3 partitions, 7 days retention)"
echo "  2. clinical-recommendations-high (3 partitions, 7 days retention)"
echo "  3. clinical-recommendations-medium (2 partitions, 14 days retention)"
echo "  4. clinical-recommendations-routine (1 partition, 30 days retention)"
echo ""
echo -e "${YELLOW}Configuration:${NC}"
echo "  - Compression: lz4 (fast, moderate compression)"
echo "  - Cleanup Policy: delete (time-based retention)"
echo "  - Min In-Sync Replicas: 2 (high availability)"
echo "  - Replication Factor: 3 (data redundancy)"
echo ""
echo -e "${YELLOW}Next Steps:${NC}"
echo "  1. Start Flink Clinical Recommendation Engine"
echo "  2. Verify data flow: Module 2 → Filter → Processor → Router → Topics"
echo "  3. Monitor topic lag and throughput"
echo "  4. Set up downstream consumers (Notification Service, EMR, Dashboard)"
echo ""
echo -e "${GREEN}Setup complete!${NC}"
