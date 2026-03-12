# 🚀 Action Plan: Fix Alert Pipeline & Get patient_ids Working

## ✅ Completed
1. ✅ Added `patient_ids` field to Module 6.3 Analytics alert metrics
2. ✅ Fixed Module 5 to output to `ml-risk-alerts.v1` instead of `alert-management.v1`

## 🔧 Next Steps to Complete

### Step 1: Fix Module 6 Alert Composition (Remove Broken SimpleAlert Source)

**Problem**: Module 6 tries to read SimpleAlerts from alert-management.v1 but it contains PatternEvents

**Solution**: Make Module 6 read PatternEvents from BOTH Module 4 topics

**File to Edit**: `src/main/java/com/cardiofit/flink/operators/Module6_AlertComposition.java`

**Changes**:
```java
// CURRENT (Lines 94-106) - BROKEN:
DataStream<SimpleAlert> simpleAlerts = createSimpleAlertSource(env);  // ❌ FAILS
DataStream<PatternEvent> patternEvents = createPatternEventSource(env);  // ✅ WORKS
DataStream<SimpleAlert> convertedPatternAlerts = patternEvents.map(new PatternToAlertConverter());
DataStream<SimpleAlert> allAlerts = simpleAlerts.union(convertedPatternAlerts);  // ❌ Fails!

// FIXED - Read PatternEvents from BOTH Module 4 topics:
// Source 1: General patterns from pattern-events.v1
DataStream<PatternEvent> patternEventsGeneral = KafkaSource.<PatternEvent>builder()
    .setBootstrapServers(getBootstrapServers())
    .setTopics("pattern-events.v1")
    .setGroupId("alert-composition")
    .setStartingOffsets(OffsetsInitializer.latest())
    .setValueOnlyDeserializer(new PatternEventDeserializer())
    .build();

// Source 2: Deterioration patterns from alert-management.v1
DataStream<PatternEvent> patternEventsDeterior = KafkaSource.<PatternEvent>builder()
    .setBootstrapServers(getBootstrapServers())
    .setTopics("alert-management.v1")
    .setGroupId("alert-composition")
    .setStartingOffsets(OffsetsInitializer.latest())
    .setValueOnlyDeserializer(new PatternEventDeserializer())
    .build();

// Source 3: Clinical patterns from Module 2 (keep existing)
DataStream<PatternEvent> patternEventsClinical = createPatternEventSource(env);

// Union all three PatternEvent streams
DataStream<PatternEvent> allPatternEvents = patternEventsGeneral
    .union(patternEventsDeterior)
    .union(patternEventsClinical);

// Convert to SimpleAlert for processing
DataStream<SimpleAlert> allAlerts = allPatternEvents
    .map(new PatternToAlertConverter());
```

---

### Step 2: Rebuild JAR with All Fixes

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing

# Clean and rebuild
mvn clean package

# Verify JAR was built
ls -lh target/flink-ehr-intelligence-1.0.0.jar
```

**Expected output**: JAR file ~50-100 MB

---

### Step 3: Deploy Module 6 Alert Composition to Flink

```bash
# Upload JAR to Flink
curl -X POST -H "Content-Type: multipart/form-data" \
  -F "jarfile=@target/flink-ehr-intelligence-1.0.0.jar" \
  http://localhost:8081/jars/upload

# Note the JAR ID from response, then submit Module 6 Alert Composition job
curl -X POST http://localhost:8081/jars/<JAR_ID>/run \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.operators.Module6_AlertComposition",
    "parallelism": 2
  }'
```

---

### Step 4: Deploy Module 6.3 Analytics to Flink

```bash
# Submit Module 6.3 Analytics job (with patient_ids field)
curl -X POST http://localhost:8081/jars/<JAR_ID>/run \
  -H "Content-Type: application/json" \
  -d '{
    "entryClass": "com.cardiofit.flink.analytics.Module6_AnalyticsEngine",
    "parallelism": 1
  }'
```

---

### Step 5: Create Missing Kafka Topics (if needed)

```bash
# Create ml-risk-alerts.v1 for Module 5 high-risk predictions
docker exec kafka kafka-topics --create \
  --topic ml-risk-alerts.v1 \
  --bootstrap-server localhost:9092 \
  --partitions 4 \
  --replication-factor 1

# Create pattern-events.v1 if not exists
docker exec kafka kafka-topics --create \
  --topic pattern-events.v1 \
  --bootstrap-server localhost:9092 \
  --partitions 4 \
  --replication-factor 1

# Create composed-alerts.v1 if not exists
docker exec kafka kafka-topics --create \
  --topic composed-alerts.v1 \
  --bootstrap-server localhost:9092 \
  --partitions 4 \
  --replication-factor 1

# Create urgent-alerts.v1 if not exists
docker exec kafka kafka-topics --create \
  --topic urgent-alerts.v1 \
  --bootstrap-server localhost:9092 \
  --partitions 4 \
  --replication-factor 1
```

---

### Step 6: Verify Data Flow

#### 6.1 Check Module 4 Outputs PatternEvents
```bash
# Check pattern-events.v1
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic pattern-events.v1 \
  --from-beginning \
  --max-messages 1

# Check alert-management.v1 (should only have PatternEvents now)
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic alert-management.v1 \
  --from-beginning \
  --max-messages 1
```

#### 6.2 Check Module 6 Alert Composition Outputs
```bash
# Check composed-alerts.v1 (should have new ComposedAlerts)
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic composed-alerts.v1 \
  --from-beginning \
  --max-messages 5

# Get count
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic composed-alerts.v1 \
  --time -1 | awk -F: '{sum += $3} END {print "Total: " sum}'
```

#### 6.3 Check Module 6.3 Analytics Outputs
```bash
# Check analytics-alert-metrics (should have patient_ids field)
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic analytics-alert-metrics \
  --from-beginning \
  --max-messages 1 | jq '.'

# Should see output like:
# {
#   "window_start": "2025-01-08T...",
#   "pattern_type": "SEPSIS",
#   "severity": "CRITICAL",
#   "alert_count": 3,
#   "unique_patients": 2,
#   "patient_ids": "PAT-001,PAT-002",  <-- THIS FIELD!
#   "avg_confidence": 0.89
# }
```

---

### Step 7: Monitor Flink Jobs

```bash
# Check all running jobs
curl http://localhost:8081/jobs

# Check specific job status
curl http://localhost:8081/jobs/<JOB_ID>

# Check job metrics
curl http://localhost:8081/jobs/<JOB_ID>/metrics
```

---

## 🎯 Success Criteria

After completing these steps, you should see:

✅ **Module 4** outputs PatternEvents to:
   - pattern-events.v1
   - alert-management.v1

✅ **Module 5** outputs MLPredictions to:
   - inference-results.v1 (all)
   - ml-risk-alerts.v1 (high-risk only)

✅ **Module 6 Alert Composition** reads PatternEvents and outputs ComposedAlerts to:
   - composed-alerts.v1 (all)
   - urgent-alerts.v1 (critical only)

✅ **Module 6.3 Analytics** reads composed-alerts.v1 and outputs to:
   - analytics-alert-metrics (with patient_ids field!)
   - analytics-ml-performance (with patient_ids field!)

---

## 🔍 Troubleshooting

### If Module 6 Alert Composition Fails:
```bash
# Check Flink job logs
curl http://localhost:8081/jobs/<JOB_ID>/exceptions

# Check Kafka consumer group offsets
docker exec kafka kafka-consumer-groups --bootstrap-server localhost:9092 \
  --group alert-composition --describe
```

### If No Alerts in composed-alerts.v1:
```bash
# Check if Module 4 is running and producing alerts
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic pattern-events.v1 --time -1

# Check Module 6 Alert Composition job is running
curl http://localhost:8081/jobs | jq '.jobs[] | select(.name | contains("Alert"))'
```

### If patient_ids Missing in Analytics:
```bash
# Verify Module 6.3 Analytics is using the updated code
curl http://localhost:8081/jobs | jq '.jobs[] | select(.name | contains("Analytics"))'

# Check analytics-alert-metrics schema
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic analytics-alert-metrics \
  --from-beginning --max-messages 1 | jq 'keys'
```

---

## 📝 Quick Reference

**Key Topics**:
- `pattern-events.v1` - General patterns from Module 4
- `alert-management.v1` - Deterioration patterns from Module 4
- `composed-alerts.v1` - Composed alerts from Module 6
- `analytics-alert-metrics` - Alert metrics with patient_ids

**Key Files Modified**:
- `Module5_MLInference.java` - Fixed ML alerts output
- `Module6_AnalyticsEngine.java` - Added patient_ids field
- `Module6_AlertComposition.java` - Needs fix to remove SimpleAlert source

**Flink Web UI**: http://localhost:8081
