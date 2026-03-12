# Module 6 → Module 8 Integration Guide

**Issue Found**: Module 8 storage projectors are configured to consume from hybrid Kafka topics, but those topics don't exist yet and Module 6 may not be deployed.

---

## 🔍 Root Cause Analysis

### What Module 8 Expects (Implemented)
Module 8 storage projectors are configured to consume from:
```
prod.ehr.events.enriched (24 partitions, 90d) → 6 core projectors
prod.ehr.fhir.upsert (12 partitions, 365d, compacted) → FHIR Store projector
prod.ehr.graph.mutations (16 partitions, 30d) → Neo4j Graph projector
```

### What Module 6 Produces (Configured but NOT Running)
Module 6 (`TransactionalMultiSinkRouter.java`) IS configured to write to these topics:
- ✅ Code ready: Lines 517-570 in TransactionalMultiSinkRouter.java
- ✅ Topic constants defined: Lines 118-133 in KafkaTopics.java
- ❌ Topics NOT created in Kafka
- ❌ Module 6 NOT deployed/running

### Current State
```
Module 1-5 → Old Topics → ❌ (Module 6 not running) → ❌ (Hybrid topics missing) → Module 8
             (enriched-
              patient-
              events-v1)
```

---

## ✅ Solution: 3-Step Integration

### Step 1: Create Hybrid Kafka Topics (5 minutes)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Create the hybrid topics
./create-hybrid-kafka-topics.sh
```

**This creates**:
```
✓ prod.ehr.events.enriched (24 partitions, 90d)
✓ prod.ehr.fhir.upsert (12 partitions, 365d, compacted)
✓ prod.ehr.graph.mutations (16 partitions, 30d)
✓ prod.ehr.alerts.critical (16 partitions, 7d)
✓ prod.ehr.analytics.events (32 partitions, 180d)
✓ prod.ehr.semantic.mesh (4 partitions, 365d, compacted)
✓ prod.ehr.audit.logs (8 partitions, 7 years)
```

**Verify topics created**:
```bash
kafka-topics --list --bootstrap-server localhost:9092 | grep "^prod\.ehr\."
```

---

### Step 2: Deploy Module 6 (10 minutes)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Deploy Module 6 Egress Routing
./deploy-module6.sh
```

**What this does**:
1. Compiles Module 6 Java code
2. Uploads JAR to Flink JobManager
3. Starts the `Module6_EgressRouting` job
4. Activates `TransactionalMultiSinkRouter` which writes to hybrid topics

**Verify Module 6 is running**:
```bash
# Check Flink Web UI
open http://localhost:8081

# Or via CLI
curl http://localhost:8081/jobs
```

**Verify Module 6 is producing events**:
```bash
# Check topic offsets (should be > 0)
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic prod.ehr.events.enriched \
  --time -1

# Consume a sample message
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched \
  --from-beginning \
  --max-messages 1
```

---

### Step 3: Start Module 8 Storage Projectors (5 minutes)

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/stream-services

# Configure Kafka credentials
cp .env.module8.example .env.module8
nano .env.module8  # Add KAFKA_API_KEY and KAFKA_API_SECRET

# Configure network and detect existing containers
./configure-network-module8.sh

# Start infrastructure (if not already running)
docker-compose -f docker-compose.module8-infrastructure.yml up -d

# Wait for infrastructure to be healthy
sleep 30

# Start all 8 Module 8 projectors
./start-module8-projectors.sh
```

**Verify Module 8 is consuming**:
```bash
# Check all 8 projector health endpoints
./health-check-module8.sh

# Check consumer lag (should decrease over time)
for port in 8050 8051 8052 8053 8054 8055 8056 8057; do
  echo "Projector on port $port:"
  curl -s http://localhost:$port/metrics | grep consumer_lag
done
```

---

## 📊 Complete Data Flow (After Integration)

```
┌─────────────┐
│  Module 1   │  Ingestion & Validation
│  (Running)  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Module 2   │  Context Assembly
│  (Running)  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Module 3   │  Clinical Decision Support
│  (Running)  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Module 4   │  Pattern Detection
│  (Running)  │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│  Module 5   │  ML Inference
│  (Running)  │
└──────┬──────┘
       │
       ▼
┌─────────────────────────────────────────────────────────┐
│  Module 6: Egress & Multi-Sink Routing                  │
│  (TransactionalMultiSinkRouter.java)                    │
│                                                         │
│  Reads: enriched-patient-events-v1, clinical-patterns  │
│  Writes: Hybrid Kafka Topics ↓                         │
└────────┬────────────────┬───────────────┬──────────────┘
         │                │               │
         ▼                ▼               ▼
┌────────────────┐ ┌──────────────┐ ┌─────────────────┐
│ prod.ehr.      │ │ prod.ehr.    │ │ prod.ehr.       │
│ events.        │ │ fhir.upsert  │ │ graph.mutations │
│ enriched       │ │ (12 parts,   │ │ (16 parts, 30d) │
│ (24 parts,     │ │  365d,       │ │                 │
│  90d)          │ │  COMPACTED)  │ │                 │
└────────┬───────┘ └──────┬───────┘ └────────┬────────┘
         │                │                  │
         │                │                  │
         └────────────────┼──────────────────┘
                          │
         ┌────────────────┴────────────────┐
         │                                 │
         ▼                                 ▼
┌────────────────────┐          ┌──────────────────────┐
│ 6 CORE PROJECTORS  │          │ 2 SPECIALIZED        │
│                    │          │ PROJECTORS           │
│ 1. PostgreSQL      │          │                      │
│ 2. MongoDB         │          │ 7. FHIR Store        │
│ 3. Elasticsearch   │          │ 8. Neo4j Graph       │
│ 4. ClickHouse      │          │                      │
│ 5. InfluxDB        │          └──────────────────────┘
│ 6. UPS Read Model  │
└────────────────────┘
         │
         ▼
┌───────────────────────────────────────────┐
│     8 SPECIALIZED STORAGE SYSTEMS         │
│                                           │
│  PostgreSQL  MongoDB  Elasticsearch       │
│  ClickHouse  InfluxDB  UPS (PostgreSQL)   │
│  Google FHIR Store    Neo4j Graph         │
└───────────────────────────────────────────┘
```

---

## 🔍 Troubleshooting

### Issue 1: Module 6 Not Writing to Topics

**Symptoms**:
```bash
# Topic offsets are 0
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic prod.ehr.events.enriched \
  --time -1
# Shows: prod.ehr.events.enriched:0:0 (no messages)
```

**Solution**:
1. Check Module 6 is running:
   ```bash
   curl http://localhost:8081/jobs | jq '.jobs[] | select(.name | contains("Module6"))'
   ```

2. Check Module 6 logs for errors:
   ```bash
   docker logs flink-taskmanager | grep -A10 "Module6"
   ```

3. Ensure input topics have data:
   ```bash
   # Module 6 reads from these old topics:
   docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
     --broker-list localhost:9092 \
     --topic enriched-patient-events-v1 \
     --time -1
   ```

---

### Issue 2: Module 8 Projectors Not Consuming

**Symptoms**:
```bash
# Health check shows 0 messages processed
curl http://localhost:8050/metrics | grep messages_processed_total
# Shows: projector_messages_processed_total 0
```

**Solution**:
1. Verify topics exist:
   ```bash
   kafka-topics --list --bootstrap-server localhost:9092 | grep prod.ehr
   ```

2. Check Module 8 is connected to Kafka:
   ```bash
   docker logs postgresql-projector | grep "Kafka consumer started"
   ```

3. Check consumer group lag:
   ```bash
   kafka-consumer-groups --bootstrap-server localhost:9092 \
     --group module8-postgresql-projector \
     --describe
   ```

---

### Issue 3: Consumer Lag Increasing

**Symptoms**:
```bash
# Lag keeps increasing
curl http://localhost:8050/metrics | grep consumer_lag
# Shows: projector_consumer_lag 5000 (and growing)
```

**Solution**:
1. Increase batch size:
   ```bash
   # Edit docker-compose.module8-complete.yml
   # Change BATCH_SIZE from 100 to 500
   ```

2. Increase parallelism (add replicas):
   ```bash
   docker-compose -f docker-compose.module8-complete.yml \
     up -d --scale postgresql-projector=3
   ```

3. Check database performance:
   ```bash
   # PostgreSQL slow queries
   docker exec a2f55d83b1fa psql -U cardiofit -d cardiofit_analytics \
     -c "SELECT query, calls, mean_exec_time FROM pg_stat_statements ORDER BY mean_exec_time DESC LIMIT 10;"
   ```

---

## 📋 Verification Checklist

After completing all 3 steps, verify end-to-end flow:

### ✅ Kafka Topics
```bash
# Should show 7 prod.ehr.* topics
kafka-topics --list --bootstrap-server localhost:9092 | grep "^prod\.ehr\." | wc -l
# Expected: 7
```

### ✅ Module 6 Running
```bash
# Should show Module 6 job in RUNNING state
curl -s http://localhost:8081/jobs | jq '.jobs[] | select(.name | contains("Module6")) | .state'
# Expected: "RUNNING"
```

### ✅ Topics Have Data
```bash
# Should show offset > 0 for each partition
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic prod.ehr.events.enriched \
  --time -1
# Expected: prod.ehr.events.enriched:0:1523 (example with data)
```

### ✅ Module 8 Health
```bash
# Should show all 8 services healthy
./health-check-module8.sh | grep "✅" | wc -l
# Expected: 8
```

### ✅ Module 8 Processing
```bash
# Should show messages > 0
for port in 8050 8051 8052 8053 8054 8055 8056 8057; do
  curl -s http://localhost:$port/metrics | grep "projector_messages_processed_total" | grep -v " 0"
done
# Expected: Non-zero counts for all projectors
```

### ✅ Data in Databases
```bash
# PostgreSQL
docker exec a2f55d83b1fa psql -U cardiofit -d cardiofit_analytics \
  -c "SELECT COUNT(*) FROM enriched_events;"
# Expected: > 0

# MongoDB
docker exec mongodb mongosh --eval "db.clinical_documents.countDocuments()"
# Expected: > 0

# Elasticsearch
curl "http://localhost:9200/clinical_events-*/_count"
# Expected: {"count": N, ...} where N > 0
```

---

## 🎯 Success Criteria

✅ All 7 hybrid topics created in Kafka
✅ Module 6 deployed and RUNNING
✅ Module 6 writing to `prod.ehr.events.enriched` (offsets > 0)
✅ All 8 Module 8 projectors healthy
✅ Module 8 consuming events (processed count > 0)
✅ Consumer lag < 1000 messages
✅ Data visible in all 8 storage systems

---

## 📚 Additional Resources

- **Module 6 Code**: [TransactionalMultiSinkRouter.java](../shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/operators/TransactionalMultiSinkRouter.java)
- **Topic Definitions**: [KafkaTopics.java](../shared-infrastructure/flink-processing/src/main/java/com/cardiofit/flink/utils/KafkaTopics.java)
- **Module 8 Architecture**: [MODULE8_COMPLETE_IMPLEMENTATION_SUMMARY.md](MODULE8_COMPLETE_IMPLEMENTATION_SUMMARY.md)
- **Deployment Guide**: [MODULE8_ORCHESTRATION_COMPLETE.md](MODULE8_ORCHESTRATION_COMPLETE.md)

---

**Last Updated**: 2025-11-16
**Status**: Integration guide complete - ready for deployment
