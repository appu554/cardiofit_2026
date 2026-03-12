#!/bin/bash
# KB-7 CDC Pipeline Startup Script
#
# Starts the complete CDC pipeline for real-time sync:
#   GraphDB (Brain) → Kafka → Neo4j (Read Replica) → Go API (Face)
#
# Usage:
#   ./scripts/start-cdc-pipeline.sh              # Start both producer and consumer
#   ./scripts/start-cdc-pipeline.sh --producer   # Start only producer
#   ./scripts/start-cdc-pipeline.sh --consumer   # Start only consumer
#   ./scripts/start-cdc-pipeline.sh --create-topic  # Create Kafka topic first

set -e

SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"
PROJECT_DIR="$(dirname "$SCRIPT_DIR")"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}    🔄 KB-7 CDC PIPELINE STARTUP${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════${NC}"

# Default configuration
MODE="both"
CREATE_TOPIC=false

# Parse arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --producer)
            MODE="producer"
            shift
            ;;
        --consumer)
            MODE="consumer"
            shift
            ;;
        --create-topic)
            CREATE_TOPIC=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Load environment
if [ -f "$PROJECT_DIR/.env" ]; then
    echo -e "${BLUE}ℹ️  Loading environment from .env${NC}"
    source "$PROJECT_DIR/.env"
fi

# Set defaults
# Note: Docker maps Kafka internal 9092 → external 9093
export KAFKA_BROKERS="${KAFKA_BROKERS:-localhost:9093}"
export KAFKA_TOPIC="${KAFKA_TOPIC:-kb7.graphdb.changes}"
export GRAPHDB_URL="${GRAPHDB_URL:-http://localhost:7200}"
export GRAPHDB_REPOSITORY="${GRAPHDB_REPOSITORY:-kb7-terminology}"
export NEO4J_URL="${NEO4J_AU_URL:-bolt://localhost:7687}"
export NEO4J_USERNAME="${NEO4J_AU_USERNAME:-neo4j}"
export NEO4J_PASSWORD="${NEO4J_AU_PASSWORD:-}"
export NEO4J_DATABASE="${NEO4J_AU_DATABASE:-kb7-au}"

echo ""
echo -e "${BLUE}📋 Configuration:${NC}"
echo "   Mode: $MODE"
echo "   Kafka: $KAFKA_BROKERS"
echo "   Topic: $KAFKA_TOPIC"
echo "   GraphDB: $GRAPHDB_URL/$GRAPHDB_REPOSITORY"
echo "   Neo4j: $NEO4J_URL/$NEO4J_DATABASE"
echo ""

# Pre-flight checks
echo -e "${YELLOW}0️⃣  Pre-flight checks...${NC}"

# Check Kafka (Docker exposes on 9093)
KAFKA_PORT="${KAFKA_BROKERS##*:}"
KAFKA_PORT="${KAFKA_PORT:-9093}"
if nc -z localhost $KAFKA_PORT 2>/dev/null; then
    echo -e "   ${GREEN}✅ Kafka is reachable on port $KAFKA_PORT${NC}"
else
    echo -e "   ${RED}❌ Kafka not reachable on localhost:$KAFKA_PORT${NC}"
    exit 1
fi

# Check GraphDB (if producer)
if [ "$MODE" = "producer" ] || [ "$MODE" = "both" ]; then
    if curl -s "$GRAPHDB_URL/rest/repositories/$GRAPHDB_REPOSITORY/size" > /dev/null 2>&1; then
        echo -e "   ${GREEN}✅ GraphDB is reachable${NC}"
    else
        echo -e "   ${RED}❌ GraphDB not reachable${NC}"
        exit 1
    fi
fi

# Check Neo4j (if consumer)
if [ "$MODE" = "consumer" ] || [ "$MODE" = "both" ]; then
    if [ -z "$NEO4J_PASSWORD" ]; then
        echo -e "   ${RED}❌ NEO4J_PASSWORD not set${NC}"
        exit 1
    fi
    echo -e "   ${GREEN}✅ Neo4j credentials configured${NC}"
fi

# Create Kafka topic if requested
if [ "$CREATE_TOPIC" = true ]; then
    echo ""
    echo -e "${YELLOW}1️⃣  Creating Kafka topic...${NC}"

    # Try to find a Kafka container to create topic
    KAFKA_CONTAINER=$(docker ps --filter "name=kafka" --format "{{.Names}}" 2>/dev/null | head -1)

    if [ -n "$KAFKA_CONTAINER" ]; then
        docker exec "$KAFKA_CONTAINER" kafka-topics --create \
            --bootstrap-server localhost:9092 \
            --topic "$KAFKA_TOPIC" \
            --partitions 3 \
            --replication-factor 1 \
            --if-not-exists 2>/dev/null || true
        echo -e "   ${GREEN}✅ Topic '$KAFKA_TOPIC' ready${NC}"
    else
        echo -e "   ${YELLOW}⚠️  No Kafka container found, assuming topic exists${NC}"
    fi
fi

# Build the CDC binary
echo ""
echo -e "${YELLOW}2️⃣  Building CDC pipeline...${NC}"
cd "$PROJECT_DIR"

if go build -o cdc-pipeline ./cmd/cdc 2>&1; then
    echo -e "   ${GREEN}✅ Build successful${NC}"
else
    echo -e "   ${RED}❌ Build failed${NC}"
    exit 1
fi

# Start the CDC pipeline
echo ""
echo -e "${YELLOW}3️⃣  Starting CDC pipeline (mode: $MODE)...${NC}"
echo ""

./cdc-pipeline --mode="$MODE"
