# Quick Reference - Flink Kafka Pipeline

## 🚀 Send Event (Easiest Method)

```bash
python3 test_kafka_pipeline.py
# Choose option 1, 2, 3, or 4
```

## 📊 View Enriched Output

**Kafka UI (Recommended):**
```
http://localhost:8080
→ Topics → enriched-patient-events-v1 → Messages
```

## ✅ What Gets Enriched

| You Send | Flink Adds/Changes |
|----------|-------------------|
| `patient_id` | → `patientId` (renamed) |
| `type` | → `eventType` (renamed) |
| `event_time` | → `timestamp` (renamed) |
| (nothing) | → `eventId` (UUID generated) |
| (nothing) | → `encounterId` (null, for future) |
| `payload` | → Normalized (lowercase, underscores) |

## 📝 Event Template

```json
{
  "patient_id": "YOUR-PATIENT-ID",
  "event_time": CURRENT_TIMESTAMP_MS,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 78,
    "blood_pressure": "120/80"
  },
  "metadata": {
    "source": "Your Source"
  }
}
```

## 🔧 Useful Commands

```bash
# Check Flink job status
docker exec cardiofit-flink-jobmanager /opt/flink/bin/flink list -r

# Check input topic messages
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic patient-events-v1 --time -1

# Check output topic messages
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1

# Send test event
python3 test_kafka_pipeline.py send vital_signs PATIENT-123

# Check processing
python3 test_kafka_pipeline.py check
```

## 📍 Available Topics (Input)

- `patient-events-v1` ← General patient events
- `medication-events-v1` ← Medication data
- `observation-events-v1` ← Labs, observations
- `vital-signs-events-v1` ← Vital signs
- `lab-result-events-v1` ← Lab results
- `validated-device-data-v1` ← Device data

**All output to:** `enriched-patient-events-v1`

## 🎯 Validation Rules

✅ Must have:
- `patient_id` (not empty)
- `event_time` (> 0, not too old/future)
- `type` (not empty)
- `payload` (not empty)

❌ Invalid events → `dlq.processing-errors.v1`

## 📚 Documentation Files

1. **ENRICHMENT_EXPLANATION.md** - Detailed enrichment process
2. **MODULE_STATUS.md** - All 6 modules explained
3. **PYTHON_TESTING_README.md** - Python script usage
4. **MANUAL_TESTING_GUIDE.md** - Command-line usage
5. **PIPELINE_TEST_RESULTS.md** - Test results

## 🌐 Monitoring URLs

- Kafka UI: http://localhost:8080
- Flink Web UI: http://localhost:8081

## 🆘 Troubleshooting

**No enriched events?**
1. Check Flink job is running
2. Check event timestamp is current
3. Check Flink exceptions tab
4. Check DLQ for validation failures

**Python script error?**
- Make sure Docker is running
- Check you're in correct directory

**Kafka UI not working?**
```bash
docker run -d --rm -p 8080:8080 \
  --network kafka_cardiofit-network \
  -e KAFKA_CLUSTERS_0_NAME=cardiofit \
  -e KAFKA_CLUSTERS_0_BOOTSTRAPSERVERS=kafka:9092 \
  --name kafka-ui \
  provectuslabs/kafka-ui:latest
```
