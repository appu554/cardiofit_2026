#!/bin/bash

# Manual Kafka & Zookeeper Startup Script
# Uses already-pulled Docker images

set -e

GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

echo -e "${BLUE}🚀 Starting Kafka Infrastructure Manually${NC}"
echo ""

# Step 1: Start Zookeeper
echo -e "${YELLOW}📦 Starting Zookeeper...${NC}"
docker run -d \
  --name manual-zookeeper \
  --network bridge \
  -p 2181:2181 \
  -e ZOOKEEPER_CLIENT_PORT=2181 \
  -e ZOOKEEPER_TICK_TIME=2000 \
  confluentinc/cp-zookeeper:7.4.0

echo -e "${GREEN}✅ Zookeeper started on port 2181${NC}"

# Wait for Zookeeper to be ready
echo -e "${YELLOW}⏳ Waiting for Zookeeper to be ready...${NC}"
sleep 10

# Step 2: Start Kafka
echo -e "${YELLOW}📦 Starting Kafka...${NC}"
docker run -d \
  --name manual-kafka \
  --network bridge \
  -p 9092:9092 \
  -p 29092:29092 \
  -e KAFKA_ZOOKEEPER_CONNECT=host.docker.internal:2181 \
  -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092,PLAINTEXT_HOST://localhost:29092 \
  -e KAFKA_LISTENER_SECURITY_PROTOCOL_MAP=PLAINTEXT:PLAINTEXT,PLAINTEXT_HOST:PLAINTEXT \
  -e KAFKA_INTER_BROKER_LISTENER_NAME=PLAINTEXT \
  -e KAFKA_OFFSETS_TOPIC_REPLICATION_FACTOR=1 \
  -e KAFKA_AUTO_CREATE_TOPICS_ENABLE=false \
  -e KAFKA_TRANSACTION_STATE_LOG_REPLICATION_FACTOR=1 \
  -e KAFKA_TRANSACTION_STATE_LOG_MIN_ISR=1 \
  confluentinc/cp-kafka:7.4.0

echo -e "${GREEN}✅ Kafka started on ports 9092 and 29092${NC}"

# Wait for Kafka to be ready
echo -e "${YELLOW}⏳ Waiting for Kafka to be ready...${NC}"
sleep 15

# Test connection
echo -e "${YELLOW}🔍 Testing Kafka connection...${NC}"
docker exec manual-kafka kafka-topics --bootstrap-server localhost:9092 --list

echo ""
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ Kafka Infrastructure Ready!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}Connection Details:${NC}"
echo "  • Zookeeper: localhost:2181"
echo "  • Kafka:     localhost:9092"
echo "  • Kafka Alt: localhost:29092"
echo ""
echo -e "${BLUE}Container Names:${NC}"
echo "  • manual-zookeeper"
echo "  • manual-kafka"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "  1. Create topics: ./create-hybrid-topics-manual.sh"
echo "  2. Test topics:   docker exec manual-kafka kafka-topics --bootstrap-server localhost:9092 --list"
echo ""
echo -e "${YELLOW}To stop:${NC}"
echo "  docker stop manual-kafka manual-zookeeper"
echo "  docker rm manual-kafka manual-zookeeper"