#!/bin/bash

# Start Kafka Cluster for CardioFit Hybrid Architecture
# This script starts Zookeeper, Kafka brokers, and verifies the cluster is operational

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Kafka installation directory
KAFKA_HOME=${KAFKA_HOME:-"/usr/local/kafka"}
KAFKA_LOGS=${KAFKA_LOGS:-"/tmp/kafka-logs"}
ZOOKEEPER_DATA=${ZOOKEEPER_DATA:-"/tmp/zookeeper"}

echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}    CardioFit Kafka Cluster Deployment - Hybrid Architecture${NC}"
echo -e "${BLUE}═══════════════════════════════════════════════════════════${NC}"
echo ""

# Function to check if a service is running
check_service() {
    local service=$1
    local port=$2

    if nc -z localhost $port 2>/dev/null; then
        echo -e "${GREEN}✅ $service is running on port $port${NC}"
        return 0
    else
        echo -e "${YELLOW}⚠️  $service is not running on port $port${NC}"
        return 1
    fi
}

# Function to wait for service to start
wait_for_service() {
    local service=$1
    local port=$2
    local max_attempts=30
    local attempt=1

    echo -e "${YELLOW}⏳ Waiting for $service to start on port $port...${NC}"

    while [ $attempt -le $max_attempts ]; do
        if nc -z localhost $port 2>/dev/null; then
            echo -e "${GREEN}✅ $service started successfully${NC}"
            return 0
        fi

        echo -n "."
        sleep 2
        attempt=$((attempt + 1))
    done

    echo -e "${RED}❌ $service failed to start after $max_attempts attempts${NC}"
    return 1
}

# Step 1: Check if Kafka is already running
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 1: Checking existing services${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if check_service "Zookeeper" 2181 && check_service "Kafka" 9092; then
    echo -e "${GREEN}✅ Kafka cluster is already running${NC}"
    echo ""
    echo -e "${BLUE}To restart the cluster, first run:${NC}"
    echo "  $0 stop"
    echo ""
    exit 0
fi

# Step 2: Start Zookeeper
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 2: Starting Zookeeper${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if ! check_service "Zookeeper" 2181; then
    echo -e "${YELLOW}🚀 Starting Zookeeper...${NC}"

    # Create Zookeeper data directory
    mkdir -p $ZOOKEEPER_DATA

    # Start Zookeeper in background
    nohup $KAFKA_HOME/bin/zookeeper-server-start.sh \
        $KAFKA_HOME/config/zookeeper.properties \
        > /tmp/zookeeper.log 2>&1 &

    echo $! > /tmp/zookeeper.pid

    # Wait for Zookeeper to start
    wait_for_service "Zookeeper" 2181
fi

# Step 3: Configure Kafka for Hybrid Architecture
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 3: Configuring Kafka for Hybrid Architecture${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Create custom Kafka configuration for hybrid architecture
cat > /tmp/server-hybrid.properties << EOF
# Kafka Server Configuration for CardioFit Hybrid Architecture
broker.id=0
listeners=PLAINTEXT://localhost:9092
advertised.listeners=PLAINTEXT://localhost:9092
log.dirs=$KAFKA_LOGS
num.network.threads=8
num.io.threads=8
socket.send.buffer.bytes=102400
socket.receive.buffer.bytes=102400
socket.request.max.bytes=104857600

# Log retention for hybrid topics (default 7 days, topics override this)
log.retention.hours=168
log.segment.bytes=1073741824
log.retention.check.interval.ms=300000

# Zookeeper connection
zookeeper.connect=localhost:2181
zookeeper.connection.timeout.ms=18000

# Group coordinator configuration
group.initial.rebalance.delay.ms=0

# Transaction support for exactly-once semantics
transaction.state.log.replication.factor=1
transaction.state.log.min.isr=1

# Compression for efficiency
compression.type=snappy

# Auto-create topics disabled (we create explicitly)
auto.create.topics.enable=false

# Increase max message size for clinical events
message.max.bytes=10485760
replica.fetch.max.bytes=10485760
EOF

echo -e "${GREEN}✅ Kafka configuration created${NC}"

# Step 4: Start Kafka Broker
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 4: Starting Kafka Broker${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if ! check_service "Kafka" 9092; then
    echo -e "${YELLOW}🚀 Starting Kafka broker...${NC}"

    # Create Kafka logs directory
    mkdir -p $KAFKA_LOGS

    # Start Kafka broker in background
    nohup $KAFKA_HOME/bin/kafka-server-start.sh \
        /tmp/server-hybrid.properties \
        > /tmp/kafka.log 2>&1 &

    echo $! > /tmp/kafka.pid

    # Wait for Kafka to start
    wait_for_service "Kafka" 9092
fi

# Step 5: Verify Kafka Cluster
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 5: Verifying Kafka Cluster${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

# Check cluster ID
echo -e "${YELLOW}📊 Fetching cluster information...${NC}"
CLUSTER_ID=$($KAFKA_HOME/bin/kafka-metadata.sh --snapshot $KAFKA_LOGS/meta.properties 2>/dev/null | grep cluster.id | cut -d'=' -f2)

if [ ! -z "$CLUSTER_ID" ]; then
    echo -e "${GREEN}✅ Cluster ID: $CLUSTER_ID${NC}"
else
    # Try alternative method
    $KAFKA_HOME/bin/kafka-broker-api-versions.sh --bootstrap-server localhost:9092 > /dev/null 2>&1
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✅ Kafka broker is responding${NC}"
    fi
fi

# List existing topics
echo ""
echo -e "${YELLOW}📋 Existing topics:${NC}"
$KAFKA_HOME/bin/kafka-topics.sh --bootstrap-server localhost:9092 --list 2>/dev/null | head -10

# Step 6: Start Kafka Connect (if available)
echo ""
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "${BLUE}Step 6: Starting Kafka Connect${NC}"
echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [ -f "$KAFKA_HOME/bin/connect-distributed.sh" ]; then
    if ! check_service "Kafka Connect" 8083; then
        echo -e "${YELLOW}🚀 Starting Kafka Connect in distributed mode...${NC}"

        # Create Connect configuration
        cat > /tmp/connect-distributed.properties << EOF
# Kafka Connect Distributed Configuration
bootstrap.servers=localhost:9092
group.id=cardiofit-connect-cluster

# Converters
key.converter=org.apache.kafka.connect.storage.StringConverter
value.converter=org.apache.kafka.connect.json.JsonConverter
value.converter.schemas.enable=false

# Internal topics for Connect
offset.storage.topic=connect-offsets
offset.storage.replication.factor=1
offset.storage.partitions=25

config.storage.topic=connect-configs
config.storage.replication.factor=1

status.storage.topic=connect-status
status.storage.replication.factor=1
status.storage.partitions=5

# REST API
rest.host.name=localhost
rest.port=8083

# Plugin path (update based on your installation)
plugin.path=/usr/local/share/kafka/plugins
EOF

        # Start Kafka Connect
        nohup $KAFKA_HOME/bin/connect-distributed.sh \
            /tmp/connect-distributed.properties \
            > /tmp/kafka-connect.log 2>&1 &

        echo $! > /tmp/kafka-connect.pid

        # Wait for Kafka Connect to start
        wait_for_service "Kafka Connect" 8083
    fi
else
    echo -e "${YELLOW}⚠️  Kafka Connect not found. Install connectors separately if needed.${NC}"
fi

# Final Summary
echo ""
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}    ✅ Kafka Cluster Started Successfully!${NC}"
echo -e "${GREEN}═══════════════════════════════════════════════════════════${NC}"
echo ""
echo -e "${BLUE}Service Status:${NC}"
check_service "Zookeeper" 2181
check_service "Kafka Broker" 9092
check_service "Kafka Connect" 8083
echo ""
echo -e "${BLUE}Log Files:${NC}"
echo "  • Zookeeper: /tmp/zookeeper.log"
echo "  • Kafka: /tmp/kafka.log"
echo "  • Connect: /tmp/kafka-connect.log"
echo ""
echo -e "${BLUE}Next Steps:${NC}"
echo "  1. Create hybrid topics: ./create-hybrid-architecture-topics.sh"
echo "  2. Deploy connectors: cd ../flink-processing/kafka-connect && ./deploy-hybrid-connectors.sh"
echo "  3. Start Flink job: See FLINK_HYBRID_DEPLOYMENT_GUIDE.md"
echo ""
echo -e "${YELLOW}To stop the cluster:${NC}"
echo "  $0 stop"
echo ""

# Create stop script
cat > /tmp/stop-kafka-cluster.sh << 'EOF'
#!/bin/bash

echo "Stopping Kafka services..."

# Stop Kafka Connect
if [ -f /tmp/kafka-connect.pid ]; then
    kill $(cat /tmp/kafka-connect.pid) 2>/dev/null
    rm /tmp/kafka-connect.pid
    echo "✓ Kafka Connect stopped"
fi

# Stop Kafka
if [ -f /tmp/kafka.pid ]; then
    kill $(cat /tmp/kafka.pid) 2>/dev/null
    rm /tmp/kafka.pid
    echo "✓ Kafka stopped"
fi

# Stop Zookeeper
if [ -f /tmp/zookeeper.pid ]; then
    kill $(cat /tmp/zookeeper.pid) 2>/dev/null
    rm /tmp/zookeeper.pid
    echo "✓ Zookeeper stopped"
fi

echo "All services stopped."
EOF

chmod +x /tmp/stop-kafka-cluster.sh

# Handle stop command
if [ "$1" == "stop" ]; then
    /tmp/stop-kafka-cluster.sh
fi