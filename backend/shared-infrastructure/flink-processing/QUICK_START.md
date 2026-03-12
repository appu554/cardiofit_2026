# Flink Pipeline - Quick Start Guide

## 🚨 You're experiencing a JAR deployment error?

**Most likely causes** (in order of probability):
1. **Kafka connectivity** - Flink can't reach Kafka from Docker
2. **Missing Kafka topics** - Required topics don't exist yet
3. **Network misconfiguration** - Docker networks not properly set up
4. **Dependency issues** - Missing Kafka connector in classpath

---

## 🚀 Quick Fix (One-Command Solution)

If your JAR deployment is failing, run this first:

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
./quick-fix-kafka-connectivity.sh
```

This script will automatically:
- ✅ Detect your Kafka configuration
- ✅ Create required Docker networks
- ✅ Update Kafka bootstrap server settings
- ✅ Create all required Kafka topics
- ✅ Restart Flink cluster with correct config

**Then submit your job:**
```bash
./submit-job.sh full-pipeline development
```

---

## 🔍 Detailed Diagnostics

If the quick fix doesn't work, run comprehensive diagnostics:

```bash
./diagnose-flink-kafka.sh
```

This will check:
- Docker container status
- Flink cluster health
- Kafka connectivity
- Network configuration
- JAR file integrity
- Required Kafka topics
- Recent error logs

---

## 📋 Manual Troubleshooting Steps

### Step 1: Verify Basic Setup

```bash
# Check if containers are running
docker ps | grep -E "flink|kafka"

# Check Flink Web UI
curl http://localhost:8081/overview

# Check TaskManager registration
curl http://localhost:8081/taskmanagers
```

### Step 2: Test Kafka Connectivity

```bash
# Test from Flink container
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka 9092"

# If that fails, try multi-node setup
docker exec cardiofit-flink-jobmanager bash -c "nc -zv kafka1 29092"
```

### Step 3: Check Kafka Topics

```bash
# Find Kafka container
KAFKA_CONTAINER=$(docker ps --format "{{.Names}}" | grep kafka | head -1)

# List topics
docker exec $KAFKA_CONTAINER kafka-topics --list --bootstrap-server localhost:9092

# Create missing topics (if needed)
docker exec $KAFKA_CONTAINER kafka-topics --create \
  --topic patient-events.v1 \
  --bootstrap-server localhost:9092 \
  --partitions 4 \
  --replication-factor 1
```

### Step 4: Review Logs

```bash
# JobManager logs
docker logs cardiofit-flink-jobmanager | tail -100

# TaskManager logs
docker logs cardiofit-flink-taskmanager-1 | tail -100

# Look for specific errors
docker logs cardiofit-flink-jobmanager 2>&1 | grep -i "exception\|error" | tail -20
```

---

## 🔧 Common Fixes

### Fix 1: Update Kafka Bootstrap Servers

Edit `flink-datastores.env`:
```bash
# For single-node Kafka
KAFKA_BOOTSTRAP_SERVERS=kafka:9092

# For 3-node cluster
KAFKA_BOOTSTRAP_SERVERS=kafka1:29092,kafka2:29093,kafka3:29094
```

Then restart:
```bash
docker-compose down && docker-compose up -d
```

### Fix 2: Ensure Docker Networks Exist

```bash
# Create networks if missing
docker network create cardiofit-network
docker network create kafka_cardiofit-network

# Connect Flink to Kafka network
docker network connect kafka_cardiofit-network cardiofit-flink-jobmanager
docker network connect kafka_cardiofit-network cardiofit-flink-taskmanager-1
docker network connect kafka_cardiofit-network cardiofit-flink-taskmanager-2
docker network connect kafka_cardiofit-network cardiofit-flink-taskmanager-3

# Restart containers
docker-compose restart
```

### Fix 3: Add Kafka Connector to Classpath (if missing from JAR)

```bash
# Download connector
wget https://repo1.maven.org/maven2/org/apache/flink/flink-connector-kafka/1.17.1/flink-connector-kafka-1.17.1.jar

# Copy to all Flink containers
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-jobmanager:/opt/flink/lib/
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-taskmanager-1:/opt/flink/lib/
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-taskmanager-2:/opt/flink/lib/
docker cp flink-connector-kafka-1.17.1.jar cardiofit-flink-taskmanager-3:/opt/flink/lib/

# Restart cluster
docker-compose restart
```

---

## 📊 Testing Your Pipeline

### 1. Submit the Job

```bash
# Full pipeline
./submit-job.sh full-pipeline development

# Or test individual modules
./submit-job.sh ingestion-only development
```

### 2. Monitor Job Status

```bash
# Web UI (recommended)
open http://localhost:8081

# CLI
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list

# Real-time logs
docker logs -f cardiofit-flink-jobmanager
```

### 3. Send Test Event

```bash
# Find Kafka container
KAFKA_CONTAINER=$(docker ps --format "{{.Names}}" | grep kafka | head -1)

# Send test patient event
docker exec $KAFKA_CONTAINER kafka-console-producer \
  --broker-list localhost:9092 \
  --topic patient-events.v1 << EOF
{
  "patientId": "test-patient-123",
  "eventType": "vital-signs",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)",
  "data": {
    "heartRate": 75,
    "bloodPressure": "120/80",
    "temperature": 98.6,
    "respiratoryRate": 16
  },
  "critical": false,
  "clinicalEvent": true
}
EOF
```

### 4. Verify Processing

```bash
# Check job metrics
curl http://localhost:8081/jobs/<job-id>/metrics

# Check if events are being processed
docker logs cardiofit-flink-taskmanager-1 | grep -i "processed\|enriched"

# Verify output topics
docker exec $KAFKA_CONTAINER kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched \
  --from-beginning \
  --max-messages 5
```

---

## 📖 Available Scripts

| Script | Purpose | Usage |
|--------|---------|-------|
| `diagnose-flink-kafka.sh` | Full system diagnostics | `./diagnose-flink-kafka.sh` |
| `quick-fix-kafka-connectivity.sh` | Auto-fix common issues | `./quick-fix-kafka-connectivity.sh` |
| `submit-job.sh` | Submit Flink job | `./submit-job.sh [job-type] [env]` |

---

## 🎯 Expected Architecture

```
┌────────────────────────────────────────────────────────┐
│                    Docker Network                       │
│                                                         │
│  ┌─────────────────┐         ┌─────────────────┐      │
│  │ Flink JobManager│◄───────►│  Kafka Cluster  │      │
│  │   Port: 8081    │         │   kafka:9092    │      │
│  └────────┬────────┘         └─────────────────┘      │
│           │                                             │
│           │ RPC: 6123                                   │
│           │                                             │
│  ┌────────▼────────┐                                   │
│  │  TaskManager 1  │                                   │
│  │    4 slots      │                                   │
│  └─────────────────┘                                   │
│  ┌─────────────────┐                                   │
│  │  TaskManager 2  │                                   │
│  │    4 slots      │                                   │
│  └─────────────────┘                                   │
│  ┌─────────────────┐                                   │
│  │  TaskManager 3  │                                   │
│  │    4 slots      │                                   │
│  └─────────────────┘                                   │
│                                                         │
│  JAR: flink-ehr-intelligence-1.0.0.jar (180MB)         │
│  Mounted at: /opt/flink/usrlib/                        │
│                                                         │
└────────────────────────────────────────────────────────┘

Networks Required:
- cardiofit-network (data stores)
- kafka_cardiofit-network (Kafka cluster)
```

---

## 🔗 Source Topics (Input)

The pipeline consumes from these Kafka topics:
- `patient-events.v1`
- `medication-events.v1`
- `observation-events.v1`
- `vital-signs-events.v1`
- `lab-result-events.v1`
- `safety-events.v1`

## 🔗 Destination Topics (Output)

The pipeline produces to these hybrid topics:
- `prod.ehr.events.enriched` - Central system of record
- `prod.ehr.fhir.upsert` - FHIR state updates
- `prod.ehr.alerts.critical` - Urgent notifications
- `prod.ehr.analytics.events` - Analytics stream
- `prod.ehr.graph.mutations` - Neo4j updates
- `prod.ehr.audit.logs` - Compliance logs

---

## ❓ Frequently Asked Questions

### Q: Job submits but fails immediately
**A:** Check JobManager logs for the full exception:
```bash
docker logs cardiofit-flink-jobmanager 2>&1 | grep -A 20 "Exception"
```
Most common: Kafka connectivity or missing topics.

### Q: "Job not found" errors
**A:** Job crashed during startup. Check:
1. Kafka is reachable from Flink
2. All required topics exist
3. No ClassNotFoundException in logs

### Q: TaskManagers won't connect
**A:** Network issue. Verify:
```bash
docker network inspect cardiofit-network
docker logs cardiofit-flink-taskmanager-1 | grep -i "connection"
```

### Q: How to restart everything cleanly?
**A:**
```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
docker-compose down
docker-compose up -d
sleep 30  # Wait for cluster to initialize
./submit-job.sh
```

---

## 📚 Additional Resources

- **Full Troubleshooting Guide**: [TROUBLESHOOTING_GUIDE.md](./TROUBLESHOOTING_GUIDE.md)
- **Project Documentation**: [CLAUDE.md](../../CLAUDE.md)
- **Flink Web UI**: http://localhost:8081
- **Prometheus Metrics**: http://localhost:9090
- **Grafana Dashboard**: http://localhost:3001

---

## 🆘 Still Having Issues?

If none of the above works, provide these details:

1. **Output of diagnostic script**:
   ```bash
   ./diagnose-flink-kafka.sh > diagnostics.log 2>&1
   ```

2. **Kafka setup details**:
   ```bash
   docker ps | grep kafka
   docker network inspect kafka_cardiofit-network > kafka-network.json
   ```

3. **Full error from job submission**:
   ```bash
   docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink run \
     --class com.cardiofit.flink.FlinkJobOrchestrator \
     /opt/flink/usrlib/flink-ehr-intelligence-1.0.0.jar \
     full-pipeline development 2>&1 | tee submit-error.log
   ```

4. **Recent JobManager logs**:
   ```bash
   docker logs cardiofit-flink-jobmanager --tail 200 > jobmanager.log
   ```

Share these files for detailed analysis!