# Quick Start Guide: Neo4j Graph Projector

## Prerequisites

1. **Neo4j Running**: Container `e8b3df4d8a02` must be running
2. **Kafka Access**: Valid API key and secret for Confluent Cloud
3. **Python 3.11+**: With pip installed
4. **Module 8 Shared**: Located at `../module8-shared`

## Step 1: Verify Neo4j

```bash
# Check Neo4j container is running
docker ps | grep neo4j

# Should show: e8b3df4d8a02 (running)

# Test connection
docker exec e8b3df4d8a02 cypher-shell -u neo4j -p "CardioFit2024!" -d cardiofit "RETURN 1;"

# Expected output: 1
```

## Step 2: Install Dependencies

```bash
cd backend/stream-services/module8-neo4j-graph-projector

# Create virtual environment (optional but recommended)
python -m venv venv
source venv/bin/activate  # On Windows: venv\Scripts\activate

# Install dependencies
pip install -r requirements.txt
```

## Step 3: Configure Environment

```bash
# Copy example environment file
cp .env.example .env

# Edit .env with your Kafka credentials
nano .env  # or use your preferred editor

# Required variables:
# KAFKA_API_KEY=<your-confluent-api-key>
# KAFKA_API_SECRET=<your-confluent-api-secret>
```

## Step 4: Run Tests

```bash
# Set environment variables
export KAFKA_API_KEY="your-key"
export KAFKA_API_SECRET="your-secret"

# Run test suite
python test_projector.py
```

Expected output:
```
✅ Neo4j connection successful
✅ Found 7+ constraints
✅ Graph mutation successful - created patient, event, and relationship
✅ Patient journey query successful - found 1 events
✅ Test mutation sent to Kafka topic prod.ehr.graph.mutations
Total: 5/5 tests passed
```

## Step 5: Start Service

```bash
# Method 1: Direct Python
python -m uvicorn app.main:app --host 0.0.0.0 --port 8057 --reload

# Method 2: Using uvicorn with log level
uvicorn app.main:app --host 0.0.0.0 --port 8057 --log-level info

# Method 3: Production mode (no reload)
uvicorn app.main:app --host 0.0.0.0 --port 8057 --workers 2
```

## Step 6: Verify Service

### Check Health
```bash
curl http://localhost:8057/health

# Expected: {"status":"healthy","timestamp":"2024-11-15T..."}
```

### Check Status
```bash
curl http://localhost:8057/status | jq

# Expected:
# {
#   "status": "running",
#   "kafka_connected": true,
#   "neo4j_connected": true,
#   "consumer_group": "neo4j-graph-projector-group",
#   "topics": ["prod.ehr.graph.mutations"],
#   ...
# }
```

### Check Graph Statistics
```bash
curl http://localhost:8057/graph/stats | jq

# Expected:
# {
#   "node_counts": {
#     "Patient": 10,
#     "ClinicalEvent": 150,
#     "Condition": 25,
#     ...
#   },
#   "relationship_count": 200,
#   "total_nodes": 210
# }
```

### View Metrics
```bash
curl http://localhost:8057/metrics

# Prometheus-format metrics
```

## Step 7: Query Graph Data

### Via Service API
```bash
# Get patient journey
curl http://localhost:8057/graph/patient-journey/P12345 | jq
```

### Via Neo4j Browser

1. Open browser: http://localhost:7474
2. Login:
   - Username: `neo4j`
   - Password: `CardioFit2024!`
   - Database: `cardiofit`

3. Run queries:

```cypher
// See all patients
MATCH (p:Patient)
RETURN p
LIMIT 10;

// Patient journey
MATCH (p:Patient {nodeId: 'P12345'})-[:HAS_EVENT]->(e:ClinicalEvent)
RETURN p, e
ORDER BY e.timestamp;

// Graph statistics
MATCH (n)
RETURN labels(n) as NodeType, count(n) as Count
ORDER BY Count DESC;
```

## Troubleshooting

### Service won't start

**Problem**: Port 8057 already in use
```bash
# Find process using port 8057
lsof -i :8057

# Kill process
kill -9 <PID>
```

**Problem**: Neo4j connection failed
```bash
# Check Neo4j is running
docker ps | grep neo4j

# Restart Neo4j if needed
docker restart e8b3df4d8a02

# Check Neo4j logs
docker logs e8b3df4d8a02
```

**Problem**: Kafka authentication failed
```bash
# Verify credentials are set
echo $KAFKA_API_KEY
echo $KAFKA_API_SECRET

# Test Kafka connection (if kafka CLI tools installed)
kafka-console-consumer \
  --bootstrap-server pkc-p11xm.us-east-1.aws.confluent.cloud:9092 \
  --topic prod.ehr.graph.mutations \
  --consumer.config kafka.properties \
  --from-beginning \
  --max-messages 1
```

### No messages being processed

**Problem**: Consumer group has no lag
```bash
# Check if topic exists and has messages
# Use Confluent Cloud UI or CLI

# Reset consumer group offset (if needed)
# This will reprocess all messages from beginning
kafka-consumer-groups \
  --bootstrap-server <broker> \
  --group neo4j-graph-projector-group \
  --reset-offsets \
  --to-earliest \
  --topic prod.ehr.graph.mutations \
  --execute
```

**Problem**: Messages in DLQ
```bash
# Check dead letter queue for failed messages
curl http://localhost:8057/metrics | grep failed

# View DLQ messages (requires Kafka CLI)
kafka-console-consumer \
  --bootstrap-server <broker> \
  --topic prod.ehr.dlq.neo4j \
  --from-beginning
```

### Query performance issues

```cypher
// Check if constraints exist
SHOW CONSTRAINTS;

// Check if indexes exist
SHOW INDEXES;

// Profile slow queries
PROFILE MATCH (p:Patient {nodeId: 'P12345'})-[:HAS_EVENT]->(e:ClinicalEvent)
RETURN p, e;

// Manually create missing indexes
CREATE INDEX event_patient IF NOT EXISTS
FOR (e:ClinicalEvent) ON (e.patientId);
```

## Monitoring

### View Logs
```bash
# Service logs (if using systemd or docker)
journalctl -u neo4j-graph-projector -f

# Docker logs
docker logs -f neo4j-graph-projector

# Direct logs (if running in terminal)
# Logs are written to stdout in JSON format
```

### Prometheus Metrics

Add to Prometheus scrape config:
```yaml
scrape_configs:
  - job_name: 'neo4j-graph-projector'
    static_configs:
      - targets: ['localhost:8057']
    metrics_path: '/metrics'
```

### Grafana Dashboard

Import metrics:
- `projector_messages_consumed_total`: Rate of message consumption
- `projector_messages_processed_total`: Rate of successful processing
- `projector_consumer_lag`: Kafka consumer lag
- `projector_batch_flush_duration`: Batch processing time

## Next Steps

1. **Generate Test Data**: Use Module 6 semantic router to generate graph mutations
2. **Build Dashboards**: Create Grafana dashboards for monitoring
3. **Scale Consumers**: Run multiple instances for higher throughput
4. **Optimize Queries**: Add custom indexes based on query patterns
5. **Integrate Visualization**: Build patient journey visualization UI

## Support

- **Logs**: Check structured JSON logs for detailed error messages
- **Health Endpoint**: `/health` for quick status check
- **Status Endpoint**: `/status` for detailed diagnostics
- **Graph Stats**: `/graph/stats` for Neo4j statistics

For issues, check:
1. Service logs
2. Neo4j logs (`docker logs e8b3df4d8a02`)
3. Kafka consumer lag
4. DLQ messages
