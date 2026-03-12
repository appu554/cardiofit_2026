#!/bin/bash

# Create Module 6 Analytics Kafka Topics
# This script creates all necessary topics for the Module 6 Analytics Engine

set -e

# Configuration
KAFKA_BROKER="${KAFKA_BOOTSTRAP_SERVERS:-localhost:9092}"
PARTITIONS=4
REPLICATION=1

echo "=================================================="
echo "Creating Module 6 Analytics Kafka Topics"
echo "=================================================="
echo "Kafka Broker: $KAFKA_BROKER"
echo "Partitions: $PARTITIONS"
echo "Replication Factor: $REPLICATION"
echo ""

# Color codes
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Function to create topic
create_topic() {
    local topic=$1
    local description=$2

    echo -e "${YELLOW}Creating topic: ${topic}${NC}"
    echo "  Description: $description"

    kafka-topics --create \
        --topic "$topic" \
        --bootstrap-server "$KAFKA_BROKER" \
        --partitions "$PARTITIONS" \
        --replication-factor "$REPLICATION" \
        --config retention.ms=604800000 \
        --config compression.type=snappy \
        --if-not-exists 2>/dev/null

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Topic created successfully${NC}"
    else
        echo -e "${GREEN}✓ Topic already exists${NC}"
    fi
    echo ""
}

# Create analytics output topics
echo "Creating analytics output topics..."
echo ""

create_topic "analytics-patient-census" \
    "Real-time patient census by department (1-min tumbling window)"

create_topic "analytics-alert-metrics" \
    "Alert performance metrics (1-min tumbling window)"

create_topic "analytics-ml-performance" \
    "ML model performance metrics (5-min tumbling window)"

create_topic "analytics-department-workload" \
    "Department workload trending (1-hour sliding window)"

create_topic "analytics-sepsis-surveillance" \
    "Real-time sepsis risk surveillance (streaming)"

# Verify topics were created
echo "=================================================="
echo "Verifying created topics..."
echo "=================================================="

kafka-topics --list --bootstrap-server "$KAFKA_BROKER" | grep "analytics-" | while read topic; do
    echo -e "${GREEN}✓ $topic${NC}"
done

echo ""
echo "=================================================="
echo "Topic creation complete!"
echo "=================================================="
echo ""
echo "Next steps:"
echo "1. Start the Flink Analytics Engine"
echo "2. Verify data flow with kafka-console-consumer"
echo "3. Start the Dashboard API service"
echo ""
echo "To monitor a topic:"
echo "  kafka-console-consumer --bootstrap-server $KAFKA_BROKER \\"
echo "    --topic analytics-patient-census --from-beginning"
