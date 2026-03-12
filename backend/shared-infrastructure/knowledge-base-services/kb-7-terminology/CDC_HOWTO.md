# KB-7 CDC Pipeline - HOW-TO Guide

## Overview

The CDC (Change Data Capture) pipeline synchronizes SNOMED CT terminology changes from GraphDB (master) to Neo4j (read replica) via Kafka streaming.

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   GraphDB   │────▶│    Kafka    │────▶│   Neo4j     │
│   (Brain)   │     │  (Stream)   │     │ (AU Replica)│
│  Port 7200  │     │  Port 9093  │     │  Port 7688  │
└─────────────┘     └─────────────┘     └─────────────┘
      │                    │                   │
   Master DB          CDC Topic           Read Replica
   (SPARQL)      kb7.graphdb.changes      (Cypher)
```

## Prerequisites

Before starting the CDC pipeline, ensure these services are running:

| Service | Port | Check Command |
|---------|------|---------------|
| Kafka | 9093 | `nc -z localhost 9093` |
| GraphDB | 7200 | `curl http://localhost:7200/rest/repositories` |
| Neo4j AU | 7688 | `nc -z localhost 7688` |

### Quick Health Check

```bash
# Check all prerequisites
nc -z localhost 9093 && echo "✅ Kafka OK" || echo "❌ Kafka not running"
curl -s http://localhost:7200/rest/repositories | grep -q kb7 && echo "✅ GraphDB OK" || echo "❌ GraphDB not running"
nc -z localhost 7688 && echo "✅ Neo4j AU OK" || echo "❌ Neo4j AU not running"
```

## Quick Start

### Option 1: Using the Start Script (Recommended)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# Start both producer and consumer
./scripts/start-cdc-pipeline.sh --mode=both

# Or start producer only
./scripts/start-cdc-pipeline.sh --mode=producer

# Or start consumer only
./scripts/start-cdc-pipeline.sh --mode=consumer
```

### Option 2: Manual Start

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# 1. Build the CDC pipeline binary
go build -o cdc-pipeline ./cmd/cdc

# 2. Set environment variables
export KAFKA_BROKERS="localhost:9093"
export KAFKA_TOPIC="kb7.graphdb.changes"
export GRAPHDB_URL="http://localhost:7200"
export GRAPHDB_REPOSITORY="kb7-terminology"
export NEO4J_URL="bolt://localhost:7688"
export NEO4J_USERNAME="neo4j"
export NEO4J_PASSWORD="kb7aupassword"
export NEO4J_DATABASE="neo4j"

# 3. Start producer (GraphDB → Kafka)
./cdc-pipeline --mode=producer --poll-interval=5s &

# 4. Start consumer (Kafka → Neo4j AU)
./cdc-pipeline --mode=consumer &
```

### Option 3: Using .env File

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/knowledge-base-services/kb-7-terminology

# Load environment from .env file
source .env

# Build and start
go build -o cdc-pipeline ./cmd/cdc
./cdc-pipeline --mode=both
```

## Configuration

### Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `KAFKA_BROKERS` | `localhost:9093` | Kafka broker address |
| `KAFKA_TOPIC` | `kb7.graphdb.changes` | CDC topic name |
| `GRAPHDB_URL` | `http://localhost:7200` | GraphDB endpoint |
| `GRAPHDB_REPOSITORY` | `kb7-terminology` | GraphDB repository name |
| `NEO4J_URL` | `bolt://localhost:7688` | Neo4j AU bolt URL |
| `NEO4J_USERNAME` | `neo4j` | Neo4j username |
| `NEO4J_PASSWORD` | `kb7aupassword` | Neo4j password |
| `NEO4J_DATABASE` | `neo4j` | Neo4j database name |

### Command Line Options

```bash
./cdc-pipeline [options]

Options:
  --mode=<mode>           Mode: producer, consumer, or both (default: both)
  --poll-interval=<dur>   Producer poll interval (default: 5s)
  --batch-size=<n>        Batch size for processing (default: 100)
  --workers=<n>           Number of consumer workers (default: 4)
```

## Monitoring

### Check Pipeline Status

```bash
# Check if producer is running
ps aux | grep "cdc-pipeline.*producer" | grep -v grep

# Check if consumer is running
ps aux | grep "cdc-pipeline.*consumer" | grep -v grep

# Get PIDs from saved files
cat /tmp/kb7-cdc-producer.pid
cat /tmp/kb7-cdc-consumer-au.pid
```

### View Logs

The CDC pipeline outputs JSON logs with statistics every 30 seconds:

```json
{
  "component": "consumer",
  "messages_received": 13200,
  "messages_processed": 13211,
  "batches_committed": 133,
  "messages_failed": 0
}
```

### Check Kafka Topic

```bash
# View messages in the CDC topic
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic kb7.graphdb.changes \
  --from-beginning \
  --max-messages 5

# Check topic info
docker exec kafka kafka-topics --describe \
  --bootstrap-server localhost:9092 \
  --topic kb7.graphdb.changes
```

### Check Neo4j AU Data

```bash
# Query node count
docker exec kb7-neo4j-au cypher-shell \
  -u neo4j -p kb7aupassword \
  "MATCH (n) RETURN count(n) as nodeCount"

# Query recent SNOMED concepts
docker exec kb7-neo4j-au cypher-shell \
  -u neo4j -p kb7aupassword \
  "MATCH (c:Class) WHERE c.uri CONTAINS 'snomed' RETURN c.code, c.rdfs__label LIMIT 5"
```

## Stopping the Pipeline

### Graceful Stop

```bash
# Stop producer
kill $(cat /tmp/kb7-cdc-producer.pid)

# Stop consumer
kill $(cat /tmp/kb7-cdc-consumer-au.pid)

# Or stop all CDC processes
pkill -f "cdc-pipeline"
```

### Force Stop

```bash
pkill -9 -f "cdc-pipeline"
```

## Troubleshooting

### Issue: "Unknown Topic Or Partition" Error

**Cause**: Kafka topic doesn't exist or wrong broker address.

**Solution**:
```bash
# Create the topic if it doesn't exist
docker exec kafka kafka-topics --create \
  --bootstrap-server localhost:9092 \
  --topic kb7.graphdb.changes \
  --partitions 3 \
  --replication-factor 1

# Verify the topic exists
docker exec kafka kafka-topics --list \
  --bootstrap-server localhost:9092 | grep kb7
```

### Issue: Connection Refused to Kafka

**Cause**: Using wrong port. Docker maps internal port 9092 to external port 9093.

**Solution**:
```bash
# Use port 9093 for external access
export KAFKA_BROKERS="localhost:9093"
```

### Issue: Neo4j Connection Failed

**Cause**: Wrong credentials or Neo4j AU not running.

**Solution**:
```bash
# Check Neo4j AU is running
docker ps | grep kb7-neo4j-au

# Start Neo4j AU if needed
docker start kb7-neo4j-au

# Verify connection
docker exec kb7-neo4j-au cypher-shell \
  -u neo4j -p kb7aupassword \
  "RETURN 1"
```

### Issue: GraphDB Not Responding

**Cause**: GraphDB service not running or wrong repository.

**Solution**:
```bash
# Check GraphDB status
curl http://localhost:7200/rest/repositories

# Verify repository exists
curl http://localhost:7200/rest/repositories/kb7-terminology/size
```

### Issue: Consumer Shows "Unknown operation type, skipping"

**Cause**: Messages in Kafka have empty or null operation field (legacy data).

**Solution**: This is informational - the consumer skips invalid messages but continues processing valid ones.

## Architecture Details

### Producer Flow

1. Polls GraphDB every N seconds (default: 5s)
2. Executes SPARQL query for recent changes (subClassOf, labels)
3. Converts triples to GraphDBChange events
4. Publishes to Kafka topic `kb7.graphdb.changes`

### Consumer Flow

1. Consumes from Kafka topic in consumer group `kb7-neo4j-sync`
2. Batches messages (default: 100 per batch)
3. Applies changes to Neo4j AU in transactions
4. Commits Kafka offsets after successful Neo4j commit

### Message Format

```json
{
  "operation": "INSERT",
  "subject": "http://snomed.info/id/44054006",
  "predicate": "http://www.w3.org/2000/01/rdf-schema#subClassOf",
  "object": "http://snomed.info/id/73211009",
  "object_type": "uri",
  "timestamp": "2025-12-09T14:30:00+05:30"
}
```

## Performance Tuning

### For High Throughput

```bash
./cdc-pipeline --mode=both \
  --poll-interval=1s \
  --batch-size=500 \
  --workers=8
```

### For Low Latency

```bash
./cdc-pipeline --mode=both \
  --poll-interval=1s \
  --batch-size=10 \
  --workers=2
```

## Docker Compose Integration

If running with Docker Compose, add the CDC pipeline service:

```yaml
services:
  kb7-cdc-pipeline:
    build:
      context: .
      dockerfile: Dockerfile.cdc
    environment:
      - KAFKA_BROKERS=kafka:29092
      - KAFKA_TOPIC=kb7.graphdb.changes
      - GRAPHDB_URL=http://graphdb:7200
      - GRAPHDB_REPOSITORY=kb7-terminology
      - NEO4J_URL=bolt://kb7-neo4j-au:7687
      - NEO4J_USERNAME=neo4j
      - NEO4J_PASSWORD=kb7aupassword
      - NEO4J_DATABASE=neo4j
    depends_on:
      - kafka
      - graphdb
      - kb7-neo4j-au
    command: ["./cdc-pipeline", "--mode=both", "--poll-interval=5s"]
```

## Related Files

- **Producer**: [internal/cdc/graphdb_producer.go](internal/cdc/graphdb_producer.go)
- **Consumer**: [internal/cdc/neo4j_consumer.go](internal/cdc/neo4j_consumer.go)
- **Start Script**: [scripts/start-cdc-pipeline.sh](scripts/start-cdc-pipeline.sh)
- **Main Entry**: [cmd/cdc/main.go](cmd/cdc/main.go)
