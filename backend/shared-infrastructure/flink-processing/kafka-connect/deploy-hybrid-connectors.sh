#!/bin/bash

# Deploy Updated Kafka Connect Connectors for Hybrid Topic Architecture
# This script updates all connectors to use the new hybrid Kafka topics

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Kafka Connect configuration
KAFKA_CONNECT_URL=${KAFKA_CONNECT_URL:-"http://localhost:8083"}
CONNECTORS_DIR="$(dirname "$0")/connectors"

echo -e "${BLUE}🔄 Deploying Updated Kafka Connect Connectors for Hybrid Architecture${NC}"
echo "========================================================================"
echo -e "${YELLOW}Kafka Connect URL: ${KAFKA_CONNECT_URL}${NC}"
echo ""

# Function to deploy a connector
deploy_connector() {
    local connector_file=$1
    local connector_name=$(basename "$connector_file" .json)

    echo -e "${BLUE}📊 Deploying connector: ${connector_name}${NC}"

    # Check if connector already exists
    if curl -s "${KAFKA_CONNECT_URL}/connectors/${connector_name}" | grep -q "error_code"; then
        echo -e "${YELLOW}⚠️  Connector '${connector_name}' doesn't exist. Creating new...${NC}"

        # Create new connector
        curl -X POST \
            -H "Content-Type: application/json" \
            -d @"${connector_file}" \
            "${KAFKA_CONNECT_URL}/connectors" \
            --fail --silent --show-error

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ Created connector: ${connector_name}${NC}"
        else
            echo -e "${RED}❌ Failed to create connector: ${connector_name}${NC}"
            exit 1
        fi
    else
        echo -e "${YELLOW}🔄 Connector '${connector_name}' exists. Updating configuration...${NC}"

        # Extract config from JSON file (remove the outer wrapper)
        config_only=$(jq '.config' "${connector_file}")

        # Update existing connector
        curl -X PUT \
            -H "Content-Type: application/json" \
            -d "${config_only}" \
            "${KAFKA_CONNECT_URL}/connectors/${connector_name}/config" \
            --fail --silent --show-error

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ Updated connector: ${connector_name}${NC}"
        else
            echo -e "${RED}❌ Failed to update connector: ${connector_name}${NC}"
            exit 1
        fi
    fi

    # Wait a moment for connector to initialize
    sleep 2

    # Check connector status
    status=$(curl -s "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status" | jq -r '.connector.state')
    if [ "$status" = "RUNNING" ]; then
        echo -e "${GREEN}✅ Connector ${connector_name} is RUNNING${NC}"
    else
        echo -e "${RED}⚠️  Connector ${connector_name} status: ${status}${NC}"
    fi

    echo ""
}

echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}DEPLOYING HYBRID ARCHITECTURE CONNECTORS${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Deploy FHIR Store connector (consumes from fhir.upsert + events.enriched)
deploy_connector "${CONNECTORS_DIR}/fhir-store-sink.json"

# Deploy ClickHouse connector (consumes from analytics.events only)
deploy_connector "${CONNECTORS_DIR}/clickhouse-sink.json"

# Deploy Redis connector (consumes from alerts.critical + fhir.upsert)
deploy_connector "${CONNECTORS_DIR}/redis-sink.json"

# Deploy Neo4j connector (consumes from graph.mutations + events.enriched)
deploy_connector "${CONNECTORS_DIR}/neo4j-sink.json"

echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${GREEN}🎉 ALL CONNECTORS DEPLOYED SUCCESSFULLY!${NC}"
echo -e "${GREEN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# List all connector statuses
echo ""
echo -e "${BLUE}📋 Connector Status Summary:${NC}"
echo ""

for connector_file in "${CONNECTORS_DIR}"/*.json; do
    connector_name=$(jq -r '.name' "$connector_file")
    status=$(curl -s "${KAFKA_CONNECT_URL}/connectors/${connector_name}/status" | jq -r '.connector.state')

    if [ "$status" = "RUNNING" ]; then
        echo -e "${GREEN}✅ ${connector_name}: ${status}${NC}"
    else
        echo -e "${RED}❌ ${connector_name}: ${status}${NC}"
    fi
done

echo ""
echo -e "${BLUE}🚀 Next Steps:${NC}"
echo "1. Verify that hybrid topics are created: bash create-hybrid-architecture-topics.sh"
echo "2. Start the TransactionalMultiSinkRouter in Flink"
echo "3. Monitor connector consumption lag: kafka-consumer-groups --bootstrap-server localhost:9092 --describe --all-groups"
echo "4. Test end-to-end data flow through each data store"
echo ""
echo -e "${BLUE}📊 Monitor connectors: ${KAFKA_CONNECT_URL}/connectors${NC}"