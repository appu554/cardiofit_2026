# Manual Kafka Testing Guide - Flink Pipeline

This guide shows you how to manually push data to Kafka topics and view the enriched output processed by Flink.

## Prerequisites

✅ Flink cluster running (check: `docker ps | grep flink`)
✅ Kafka broker running (check: `docker ps | grep kafka`)
✅ Kafka UI running on port 8080 (check: http://localhost:8080)
✅ Flink job deployed and running

## Method 1: Using Shell Script (Easiest)

### Step 1: Use the Test Script

```bash
cd /Users/apoorvabk/Downloads/cardiofit/backend/shared-infrastructure/flink-processing
bash send-test-events.sh
```

This automatically sends 3 test events with current timestamps.

### Step 2: View Output in Kafka UI

1. Open browser: http://localhost:8080
2. Click on **Topics** in left sidebar
3. Find **enriched-patient-events-v1** topic
4. Click **Messages** tab
5. You'll see the enriched events with transformation applied

---

## Method 2: Manual Command Line (Full Control)

### Step 1: Prepare Your Event Data

Events must be **single-line JSON** with these required fields:
- `patient_id`: String (required)
- `event_time`: Number in milliseconds since epoch (required)
- `type`: String (required)
- `payload`: Object with data (required)
- `metadata`: Object with source info (optional)

### Step 2: Get Current Timestamp

```bash
CURRENT_TIME=$(date +%s)000
echo "Current timestamp: $CURRENT_TIME"
```

### Step 3: Send Events to Kafka

#### Example 1: Patient Vital Signs Event
```bash
echo '{"patient_id":"P12345","event_time":'$(date +%s)000',"type":"vital_signs","payload":{"heart_rate":78,"blood_pressure":"120/80","temperature":98.6,"oxygen_saturation":97},"metadata":{"source":"ICU Monitor","location":"Ward A","device_id":"MON-001"}}' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1
```

#### Example 2: Medication Administration Event
```bash
echo '{"patient_id":"P12345","event_time":'$(date +%s)000',"type":"medication","payload":{"medication_name":"Lisinopril","dosage":"10mg","route":"oral","frequency":"once daily"},"metadata":{"source":"Pharmacy System","administered_by":"Nurse Johnson","location":"Room 302"}}' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
```

#### Example 3: Lab Result Event
```bash
echo '{"patient_id":"P12345","event_time":'$(date +%s)000',"type":"lab_result","payload":{"test_name":"Complete Blood Count","hemoglobin":14.5,"wbc":7200,"platelets":250000},"metadata":{"source":"Lab System","lab_name":"Central Lab","technician":"Tech-456"}}' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1
```

#### Example 4: Vital Signs (Dedicated Topic)
```bash
echo '{"patient_id":"P12345","event_time":'$(date +%s)000',"type":"vital_signs","payload":{"heart_rate":72,"respiratory_rate":16,"temperature":98.2},"metadata":{"source":"Bedside Monitor"}}' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic vital-signs-events-v1
```

### Step 4: Verify Events Were Sent

Check message count in input topic:
```bash
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic patient-events-v1 \
  --time -1
```

Output shows partition:offset (e.g., `patient-events-v1:0:5` means 5 messages in partition 0)

---

## Method 3: View Output (Multiple Ways)

### Option A: Kafka UI (Recommended - Visual)

1. **Open Kafka UI**: http://localhost:8080
2. **Navigate**: Topics → enriched-patient-events-v1
3. **Click**: Messages tab
4. **View**: JSON formatted enriched events

**What you'll see**:
- Original patient data
- Added ingestion metadata
- Normalized payload fields
- Event IDs auto-generated if missing

### Option B: Command Line Consumer

```bash
# View all messages from beginning
docker exec kafka kafka-console-consumer \
  --bootstrap-server localhost:9092 \
  --topic enriched-patient-events-v1 \
  --from-beginning \
  --max-messages 10 \
  --timeout-ms 5000
```

### Option C: Check Message Count Only

```bash
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 \
  --topic enriched-patient-events-v1 \
  --time -1
```

### Option D: Monitor Flink Processing Metrics

```bash
# Check job is running
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r

# View Flink Web UI
open http://localhost:8081
```

In Flink UI:
- Click on your running job
- See **Records Sent** and **Records Received** metrics
- Check **Exceptions** tab (should be empty)

---

## Complete Example: Send and Verify

Here's a complete workflow:

```bash
# 1. Send a test event
TIMESTAMP=$(date +%s)000
echo '{"patient_id":"TEST-001","event_time":'$TIMESTAMP',"type":"vital_signs","payload":{"heart_rate":75,"bp":"118/76"},"metadata":{"source":"Manual Test"}}' | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

# 2. Wait a moment for Flink to process
sleep 3

# 3. Check input topic count
echo "Input topic messages:"
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic patient-events-v1 --time -1 2>/dev/null

# 4. Check output topic count
echo -e "\nOutput topic messages:"
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null

# 5. Open Kafka UI to view enriched event
echo -e "\n✅ Now open http://localhost:8080 to view enriched events!"
```

---

## Understanding the Data Transformation

### Input Event (What You Send):
```json
{
  "patient_id": "P12345",
  "event_time": 1759303966000,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 78,
    "blood-pressure": "120/80"
  },
  "metadata": {
    "source": "ICU Monitor"
  }
}
```

### Output Event (What Flink Creates):
```json
{
  "id": "auto-generated-uuid",
  "patient_id": "P12345",
  "event_type": "vital_signs",
  "event_time": 1759303966000,
  "processing_time": 1759303970123,
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80"    // Note: "-" normalized to "_"
  },
  "ingestion_metadata": {
    "source": "ICU Monitor",
    "ingestion_time": 1759303970123,
    "subtask_index": 2
  }
}
```

**Key Transformations**:
1. ✅ Auto-generates event `id` if missing
2. ✅ Normalizes payload keys (blood-pressure → blood_pressure)
3. ✅ Adds `processing_time` timestamp
4. ✅ Adds `ingestion_metadata` with source, time, and Flink subtask info
5. ✅ Validates required fields (patient_id, event_time, type, payload)

---

## Available Input Topics

You can send events to any of these topics:

| Topic Name | Event Type | Example Use Case |
|------------|------------|------------------|
| `patient-events-v1` | General patient events | Demographics, admissions |
| `medication-events-v1` | Medication events | Prescriptions, administrations |
| `observation-events-v1` | Clinical observations | Lab results, assessments |
| `vital-signs-events-v1` | Vital signs | Heart rate, BP, temperature |
| `lab-result-events-v1` | Laboratory results | Blood tests, cultures |
| `validated-device-data-v1` | Device data | IoT device readings |

**All events get enriched and sent to**: `enriched-patient-events-v1`

---

## Validation Rules

Your events must pass these validation checks:

✅ **Required Fields**:
- `patient_id` must exist and not be empty
- `event_time` must be > 0
- `type` must exist and not be empty
- `payload` must exist and not be empty

✅ **Time Validation**:
- Event time must not be > 1 hour in the future
- Event time must not be > 30 days in the past

❌ **Failed Events**: Sent to Dead Letter Queue (DLQ) topic: `dlq.processing-errors.v1`

---

## Troubleshooting

### Events Not Appearing in Output?

**Check 1**: Is Flink job running?
```bash
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r
```

**Check 2**: Are events in input topic?
```bash
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic patient-events-v1 --time -1
```

**Check 3**: Any validation failures in DLQ?
```bash
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic dlq.processing-errors.v1 --time -1
```

**Check 4**: Any Flink exceptions?
- Open http://localhost:8081
- Click on running job
- Check **Exceptions** tab

### Common Issues

**Issue**: "command not found: docker exec"
**Solution**: You're not in the right directory. Use full docker command path or check Docker is running.

**Issue**: Events sent but output count is 0
**Solution**: Check event timestamp - it must be current (not old test data). Use `$(date +%s)000` for current time.

**Issue**: Kafka UI not accessible
**Solution**: Start Kafka UI container:
```bash
docker run -d --rm -p 8080:8080 \
  --network kafka_cardiofit-network \
  -e KAFKA_CLUSTERS_0_NAME=cardiofit \
  -e KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092 \
  --name kafka-ui \
  provectuslabs/kafka-ui:latest
```

---

## Quick Reference Commands

```bash
# Send test event
bash send-test-events.sh

# Check input count
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic patient-events-v1 --time -1

# Check output count
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1

# View in Kafka UI
open http://localhost:8080

# View Flink metrics
open http://localhost:8081

# Check Flink job status
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r
```

---

## Summary

**Sending Data**: Use `kafka-console-producer` with single-line JSON
**Viewing Output**: Use Kafka UI at http://localhost:8080 (easiest) or command-line consumer
**Monitoring**: Flink Web UI at http://localhost:8081 for processing metrics
**Validation**: Events must have patient_id, event_time, type, and payload

Happy testing! 🚀
