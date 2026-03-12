# Phase 1: Flink Modules Deployment Guide
## Complete Pipeline - All 8 Modules

**Status:** Ready to Deploy
**CDC Integration:** Phase 2 (not included in this deployment)
**Approach:** Static YAML Loading (Foundation)

---

## 📦 Modules to Deploy

| # | Module | Class | Purpose |
|---|--------|-------|---------|
| 1 | **Ingestion** | `Module1_Ingestion` | Consume & validate clinical events |
| 2 | **Context Assembly** | `Module2_Enhanced` | Enrich with patient context, Neo4j lookup |
| 3 | **Comprehensive CDS** | `Module3_ComprehensiveCDS` | Protocol matching, guidelines, medications |
| 4 | **Pattern Detection** | `Module4_PatternDetection` | CEP patterns, deterioration detection |
| 5 | **ML Inference** | `Module5_MLInference` | MIMIC-IV models, risk scoring |
| 6 | **Egress Routing** | `Module6_EgressRouting` | Multi-sink routing (FHIR, Analytics, Neo4j) |
| 7 | **Alert Composition** | `Module6_AlertComposition` | Alert aggregation & prioritization |
| 8 | **Analytics Engine** | `Module6_AnalyticsEngine` | Real-time analytics, dashboards |

---

## 🚀 Quick Start

### Prerequisites

✅ **Kafka Running:**
```bash
docker ps | grep kafka
# Should show: kafka container running on port 9092
```

✅ **Flink Cluster Running:**
```bash
curl -s http://localhost:8081 | head -1
# Should return HTML or "Apache Flink"
```

✅ **Topics Created:**
```bash
docker exec kafka kafka-topics --list --bootstrap-server localhost:9092
# Should show: patient-events-v1, enriched-patient-events-v1, etc.
```

### One-Command Deployment

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Deploy all 8 modules
./deploy-all-8-modules.sh
```

**What it does:**
1. ✅ Checks Flink and Kafka availability
2. ✅ Builds JAR if not exists (`mvn clean package`)
3. ✅ Cancels any existing jobs
4. ✅ Uploads JAR to Flink cluster
5. ✅ Deploys all 8 modules sequentially
6. ✅ Reports success/failure for each module

**Expected Output:**
```
╔═══════════════════════════════════════════════════════════════╗
║  Flink EHR Intelligence Platform - Complete Deployment        ║
║  Phase 1: Static YAML Loading (8 Modules)                    ║
╚═══════════════════════════════════════════════════════════════╝

[Step 1/4] Pre-Deployment Checks...
  ⏳ Checking Flink cluster... ✓ Running
  ⏳ Checking Kafka... ✓ Running
  ⏳ Checking JAR file... ✓ Found (152M)

[Step 2/4] Cleaning up existing jobs...
  ✓ No running jobs

[Step 3/4] Uploading JAR to Flink...
  ⏳ Uploading target/flink-ehr-intelligence-1.0.0.jar... ✓ Uploaded
  📦 JAR ID: 5a3b4c2d_flink-ehr-intelligence-1.0.0.jar

[Step 4/4] Deploying modules...

━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
Deploying Module 1: Ingestion & Gateway (1/8)
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
  📋 Class: com.cardiofit.flink.operators.Module1_Ingestion
  ⚙️  Parallelism: 2
  ⏳ Submitting job... ✓ Running
  🆔 Job ID: 7f8a9b0c1d2e3f4a5b6c7d8e9f0a1b2c

[... 7 more modules deploy successfully ...]

╔═══════════════════════════════════════════════════════════════╗
║                   DEPLOYMENT SUMMARY                          ║
╚═══════════════════════════════════════════════════════════════╝

✅ SUCCESS: All 8 modules deployed!
```

---

## 🧪 Testing the Pipeline

### Quick Test

```bash
./test-complete-pipeline.sh
```

**What it tests:**
1. Sends test patient event to `patient-events-v1`
2. Verifies Module 1 output → `enriched-patient-events-v1`
3. Verifies Module 3 output → `comprehensive-cds-events.v1`
4. Verifies Module 4 output → `clinical-patterns.v1`
5. Verifies Module 6 output → `prod.ehr.events.enriched`

### Manual Testing

#### Test 1: Send Patient Event
```bash
# Send high-risk vital signs
echo '{
  "id": "test-001",
  "patient_id": "PATIENT-123",
  "type": "vital_signs",
  "event_time": '$(date +%s)'000,
  "payload": {
    "heart_rate": 125,
    "blood_pressure_systolic": 160,
    "blood_pressure_diastolic": 100,
    "respiratory_rate": 28,
    "temperature": 39.5,
    "spo2": 88
  },
  "metadata": {
    "source": "bedside_monitor",
    "location": "ICU-BED-05",
    "device_id": "MON-05"
  }
}' | docker exec -i kafka kafka-console-producer \
  --bootstrap-server localhost:9092 \
  --topic patient-events-v1
```

#### Test 2: Verify Output Topics

**Module 1 Output (Validated Events):**
```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning --max-messages 1
```

**Module 3 Output (CDS Recommendations):**
```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic comprehensive-cds-events.v1 \
  --from-beginning --max-messages 1
```

**Module 4 Output (Clinical Patterns):**
```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic clinical-patterns.v1 \
  --from-beginning --max-messages 1
```

**Module 6 Output (Final Enriched Events):**
```bash
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic prod.ehr.events.enriched \
  --from-beginning --max-messages 1 | jq
```

---

## 📊 Monitoring

### Flink Web UI

**Access:** http://localhost:8081

**What to check:**
- ✅ All 8 jobs show `RUNNING` status
- ✅ No task failures
- ✅ Checkpoints completing successfully
- ✅ Backpressure indicators are LOW

### Job Status via CLI

```bash
# List all running jobs
curl -s http://localhost:8081/jobs/overview | jq '.jobs[] | {name, state, "start-time"}'

# Get specific job details
JOB_ID="<job-id-from-deployment>"
curl -s http://localhost:8081/jobs/$JOB_ID | jq
```

### Consumer Group Lag

```bash
# Check Module 1 consumer lag
docker exec kafka kafka-consumer-groups \
  --bootstrap-server localhost:9092 \
  --describe --group patient-ingestion-v2

# Should show LAG = 0 or very small number
```

---

## 🔍 Troubleshooting

### Issue: JAR Build Fails

**Error:**
```
[ERROR] Failed to execute goal org.apache.maven.plugins:maven-compiler-plugin
```

**Solution:**
```bash
# Check Java version (need Java 17+)
java -version

# Clean and rebuild
mvn clean
rm -rf target/
mvn package -DskipTests
```

### Issue: Module Fails to Deploy

**Error:**
```
{"errors":["Could not instantiate the program entry point"]}
```

**Solution:**
1. Check class name is correct (case-sensitive)
2. Verify JAR contains the class:
```bash
jar tf target/flink-ehr-intelligence-1.0.0.jar | grep Module1_Ingestion
```

### Issue: No Output in Topics

**Symptoms:**
- Consumer shows no messages
- Topics exist but are empty

**Diagnosis:**
```bash
# Check if job is actually running
curl -s http://localhost:8081/jobs/overview | jq '.jobs[] | select(.state=="RUNNING")'

# Check Flink task manager logs
docker logs flink-taskmanager

# Check for exceptions
curl -s http://localhost:8081/jobs/$JOB_ID/exceptions
```

**Common causes:**
1. ❌ Kafka topics don't exist → Run `./create-kafka-topics.sh`
2. ❌ Wrong Kafka bootstrap servers → Check `KafkaConfigLoader.java`
3. ❌ Serialization errors → Check TaskManager logs for Jackson errors

### Issue: High Backpressure

**Symptoms:**
- Flink UI shows RED backpressure indicators
- Processing lag increasing

**Solution:**
```bash
# Increase parallelism (edit deploy script)
PARALLELISM=4  # Change from 2 to 4

# Redeploy with higher resources
./deploy-all-8-modules.sh
```

---

## 📈 Performance Targets

| Metric | Target | How to Measure |
|--------|--------|----------------|
| **End-to-End Latency** | < 310ms | Check event timestamps (event_time → processing_time) |
| **Throughput** | 10,000 events/sec | Flink UI → Records Received/Sent |
| **Checkpoint Duration** | < 5 seconds | Flink UI → Checkpoints tab |
| **Consumer Lag** | < 100 messages | `kafka-consumer-groups --describe` |
| **CPU Usage** | < 70% | Docker stats or Prometheus metrics |

---

## ✅ Success Criteria

After deployment, verify:

- [x] **All 8 jobs RUNNING** in Flink UI
- [x] **Test event flows end-to-end** (patient-events-v1 → prod.ehr.events.enriched)
- [x] **Module 3 loads 17 protocols** (check logs: "Phase 1 SUCCESS: 17 clinical protocols loaded")
- [x] **Zero errors in TaskManager logs** (first 5 minutes)
- [x] **Consumer lag < 100** for all consumer groups
- [x] **Checkpoints completing** (green checkmarks in Flink UI)

---

## 🔄 Next Steps (Phase 2)

After Phase 1 is stable:

1. **Week 3-4: CDC Integration**
   - Add CDC event models (`ProtocolCDCEvent.java`)
   - Implement CDC deserializers
   - Add BroadcastStream to Module 3

2. **Week 5: Neo4j Synchronization**
   - Blue/Green Neo4j deployment
   - CDC consumer for semantic mesh

3. **Week 6: Production Hardening**
   - Chaos testing
   - Performance optimization
   - Monitoring dashboards (Grafana)

**Current Status:** Phase 1 ready for deployment ✅

---

## 📞 Support

**Flink Logs:**
```bash
# JobManager logs
docker logs flink-jobmanager

# TaskManager logs
docker logs flink-taskmanager
```

**Kafka Logs:**
```bash
docker logs kafka
```

**Check All Infrastructure:**
```bash
docker ps | grep -E "kafka|flink|neo4j"
```

**Restart Everything:**
```bash
docker-compose down
docker-compose up -d
# Wait 30 seconds for services to start
./deploy-all-8-modules.sh
```
