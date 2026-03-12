#!/bin/bash

# Deploy Kafka Connect Connectors for Clinical Events Distribution
# Single topic (clinical-events-unified.v1) → Multiple data stores via SMTs

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Configuration
KAFKA_CONNECT_URL=${KAFKA_CONNECT_URL:-"http://localhost:8083"}
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CONNECTORS_DIR="$SCRIPT_DIR/connectors"

echo -e "${BLUE}🔌 Deploying Kafka Connect Connectors for Clinical Events Distribution${NC}"
echo "========================================================================"

# Function to check if Kafka Connect is running
check_kafka_connect() {
    echo -e "${BLUE}🔍 Checking Kafka Connect status...${NC}"

    if curl -s "$KAFKA_CONNECT_URL" > /dev/null 2>&1; then
        echo -e "${GREEN}✅ Kafka Connect is running at $KAFKA_CONNECT_URL${NC}"
        return 0
    else
        echo -e "${RED}❌ Kafka Connect is not accessible at $KAFKA_CONNECT_URL${NC}"
        echo -e "${YELLOW}💡 Please ensure Kafka Connect is running before deploying connectors${NC}"
        return 1
    fi
}

# Function to deploy a single connector
deploy_connector() {
    local connector_file="$1"
    local connector_name=$(basename "$connector_file" .json)

    echo -e "${YELLOW}📤 Deploying connector: $connector_name${NC}"

    # Check if connector already exists
    if curl -s "$KAFKA_CONNECT_URL/connectors/$connector_name" > /dev/null 2>&1; then
        echo -e "${YELLOW}⚠️  Connector '$connector_name' already exists. Updating...${NC}"

        # Update existing connector
        curl -X PUT \
            -H "Content-Type: application/json" \
            -d @"$connector_file" \
            "$KAFKA_CONNECT_URL/connectors/$connector_name/config" \
            > /dev/null 2>&1

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ Updated connector: $connector_name${NC}"
        else
            echo -e "${RED}❌ Failed to update connector: $connector_name${NC}"
            return 1
        fi
    else
        # Create new connector
        curl -X POST \
            -H "Content-Type: application/json" \
            -d @"$connector_file" \
            "$KAFKA_CONNECT_URL/connectors" \
            > /dev/null 2>&1

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✅ Created connector: $connector_name${NC}"
        else
            echo -e "${RED}❌ Failed to create connector: $connector_name${NC}"
            return 1
        fi
    fi
}

# Function to check connector status
check_connector_status() {
    local connector_name="$1"

    echo -e "${BLUE}🔍 Checking status of $connector_name...${NC}"

    local status=$(curl -s "$KAFKA_CONNECT_URL/connectors/$connector_name/status" | jq -r '.connector.state' 2>/dev/null)

    case "$status" in
        "RUNNING")
            echo -e "${GREEN}✅ $connector_name: RUNNING${NC}"
            ;;
        "FAILED")
            echo -e "${RED}❌ $connector_name: FAILED${NC}"
            ;;
        "PAUSED")
            echo -e "${YELLOW}⏸️  $connector_name: PAUSED${NC}"
            ;;
        *)
            echo -e "${YELLOW}⚠️  $connector_name: $status${NC}"
            ;;
    esac
}

# Function to list all deployed connectors
list_connectors() {
    echo -e "${BLUE}📋 Listing deployed connectors...${NC}"

    local connectors=$(curl -s "$KAFKA_CONNECT_URL/connectors" | jq -r '.[]' 2>/dev/null)

    if [ -n "$connectors" ]; then
        echo "$connectors" | while read -r connector; do
            check_connector_status "$connector"
        done
    else
        echo -e "${YELLOW}⚠️  No connectors found${NC}"
    fi
}

# Main execution
main() {
    # Parse command line arguments
    DEPLOY_ALL=true
    LIST_ONLY=false

    while [[ $# -gt 0 ]]; do
        case $1 in
            --list)
                LIST_ONLY=true
                shift
                ;;
            --connector)
                DEPLOY_ALL=false
                SPECIFIC_CONNECTOR="$2"
                shift 2
                ;;
            --help)
                echo "Usage: $0 [OPTIONS]"
                echo ""
                echo "Options:"
                echo "  --list              List all deployed connectors and their status"
                echo "  --connector NAME    Deploy specific connector only"
                echo "  --help              Show this help message"
                echo ""
                echo "Available connectors:"
                echo "  • neo4j-sink                 - Neo4j clinical knowledge graph"
                echo "  • clickhouse-sink            - ClickHouse time-series analytics"
                echo "  • elasticsearch-sink         - Elasticsearch search and indexing"
                echo "  • redis-sink                 - Redis real-time caching"
                echo "  • fhir-store-sink            - Google FHIR Store clinical persistence"
                exit 0
                ;;
            *)
                echo -e "${RED}❌ Unknown option: $1${NC}"
                echo "Use --help for usage information"
                exit 1
                ;;
        esac
    done

    # Check Kafka Connect availability
    if ! check_kafka_connect; then
        exit 1
    fi

    # Handle list-only mode
    if [ "$LIST_ONLY" = true ]; then
        list_connectors
        exit 0
    fi

    # Deploy connectors
    if [ "$DEPLOY_ALL" = true ]; then
        echo -e "${BLUE}🚀 Deploying all clinical event connectors...${NC}"

        # Deploy in dependency order
        connectors=(
            "neo4j-sink.json"
            "clickhouse-sink.json"
            "elasticsearch-sink.json"
            "redis-sink.json"
            "fhir-store-sink.json"
        )

        for connector in "${connectors[@]}"; do
            connector_path="$CONNECTORS_DIR/$connector"
            if [ -f "$connector_path" ]; then
                deploy_connector "$connector_path"
                sleep 2  # Brief pause between deployments
            else
                echo -e "${RED}❌ Connector file not found: $connector_path${NC}"
            fi
        done

    else
        # Deploy specific connector
        connector_path="$CONNECTORS_DIR/$SPECIFIC_CONNECTOR.json"
        if [ -f "$connector_path" ]; then
            deploy_connector "$connector_path"
        else
            echo -e "${RED}❌ Connector file not found: $connector_path${NC}"
            exit 1
        fi
    fi

    # Wait for connectors to initialize
    echo -e "${YELLOW}⏳ Waiting for connectors to initialize...${NC}"
    sleep 10

    # Check final status
    echo -e "${BLUE}📊 Final connector status:${NC}"
    list_connectors

    echo ""
    echo -e "${GREEN}🎉 Kafka Connect deployment complete!${NC}"
    echo "========================================================================"
    echo -e "${BLUE}🌐 Management URLs:${NC}"
    echo "• Kafka Connect REST API:   $KAFKA_CONNECT_URL"
    echo "• Connector Status:         $KAFKA_CONNECT_URL/connectors"
    echo "• Individual Status:        $KAFKA_CONNECT_URL/connectors/{name}/status"
    echo ""
    echo -e "${BLUE}🔄 Data Flow:${NC}"
    echo "Flink → clinical-events-unified.v1 → Kafka Connect → Data Stores"
    echo "• Neo4j:           Clinical knowledge graphs"
    echo "• ClickHouse:      Time-series analytics"
    echo "• Elasticsearch:   Search and indexing"
    echo "• Redis:           Real-time caching"
    echo "• Google FHIR:     Clinical persistence"
    echo ""
    echo -e "${YELLOW}💡 Next Steps:${NC}"
    echo "1. Monitor connector health via Kafka Connect REST API"
    echo "2. Verify data flow in target data stores"
    echo "3. Check for any failed messages in error logs"
    echo "4. Set up monitoring and alerting for connector failures"
}

# Run main function
main "$@"