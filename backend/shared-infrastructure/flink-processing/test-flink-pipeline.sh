#!/bin/bash

# Flink Pipeline End-to-End Test Script
# Tests the complete ingestion pipeline with properly formatted events

set -e

KAFKA_CONTAINER="kafka"
KAFKA_BOOTSTRAP="localhost:9092"
FLINK_API="http://localhost:8081"

echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║         Flink Pipeline End-to-End Test                        ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo

# Check job status
echo "📊 Checking Flink job status..."
JOB_ID=$(curl -s "$FLINK_API/jobs" | python3 -c "import json, sys; jobs = json.load(sys.stdin)['jobs']; print([j['id'] for j in jobs if j['status'] == 'RUNNING'][0] if any(j['status'] == 'RUNNING' for j in jobs) else '')")

if [ -z "$JOB_ID" ]; then
    echo "❌ No running Flink job found!"
    exit 1
fi

echo "✅ Found running job: $JOB_ID"
echo

# Send test events (single-line JSON to avoid stdin issues)
echo "📤 Sending test events..."

# Event 1: Normal vital signs
echo '{"id":"test-001","source":"test","type":"vital-signs","patient_id":"PT-TEST-001","encounter_id":"ENC-001","event_time":1727740900000,"received_time":1727740900000,"payload":{"heartRate":75,"bloodPressure":"120/80","temperature":98.6},"metadata":{"device":"monitor-1"},"correlation_id":"test-001","version":"1.0"}' | \
    docker exec -i $KAFKA_CONTAINER kafka-console-producer --bootstrap-server $KAFKA_BOOTSTRAP --topic patient-events-v1
echo "  ✅ Event 1: Vital signs sent"

# Event 2: Medication
echo '{"id":"test-002","source":"test","type":"medication","patient_id":"PT-TEST-002","encounter_id":"ENC-002","event_time":1727740920000,"received_time":1727740920000,"payload":{"medication":"Aspirin","dose":"100mg"},"metadata":{"nurse":"N001"},"correlation_id":"test-002","version":"1.0"}' | \
    docker exec -i $KAFKA_CONTAINER kafka-console-producer --bootstrap-server $KAFKA_BOOTSTRAP --topic medication-events-v1
echo "  ✅ Event 2: Medication sent"

# Event 3: Lab result
echo '{"id":"test-003","source":"test","type":"lab-result","patient_id":"PT-TEST-003","encounter_id":"ENC-003","event_time":1727740940000,"received_time":1727740940000,"payload":{"test":"WBC","value":8.5,"unit":"K/uL"},"metadata":{"lab":"LAB-001"},"correlation_id":"test-003","version":"1.0"}' | \
    docker exec -i $KAFKA_CONTAINER kafka-console-producer --bootstrap-server $KAFKA_BOOTSTRAP --topic observation-events-v1
echo "  ✅ Event 3: Lab result sent"

echo
echo "⏳ Waiting 10 seconds for processing..."
sleep 10

# Check metrics
echo
echo "📊 Checking processing metrics..."
curl -s "$FLINK_API/jobs/$JOB_ID" | python3 -c "
import json, sys
data = json.load(sys.stdin)
print('Job State:', data['state'])
total_read = sum(v['metrics'].get('read-records', 0) for v in data['vertices'])
total_write = sum(v['metrics'].get('write-records', 0) for v in data['vertices'])
print(f'Total Records: Read={total_read}, Write={total_write}')
print()
print('Per-Vertex Metrics:')
for v in data['vertices']:
    r, w = v['metrics'].get('read-records', 0), v['metrics'].get('write-records', 0)
    if r > 0 or w > 0:
        print(f'  {v[\"name\"][:50]:50s} R:{r:3d} W:{w:3d}')
"

# Check for exceptions
echo
echo "🔍 Checking for exceptions..."
EXCEPTIONS=$(curl -s "$FLINK_API/jobs/$JOB_ID/exceptions" | python3 -c "import json, sys; print(len(json.load(sys.stdin).get('all-exceptions', [])))")
if [ "$EXCEPTIONS" -eq "0" ]; then
    echo "✅ No exceptions - pipeline is healthy!"
else
    echo "❌ Found $EXCEPTIONS exceptions"
fi

echo
echo "╔═══════════════════════════════════════════════════════════════╗"
echo "║                     Test Complete!                            ║"
echo "╚═══════════════════════════════════════════════════════════════╝"
echo
echo "📌 Next Steps:"
echo "  1. View Flink UI: $FLINK_API"
echo "  2. Check output topic: enriched-patient-events-v1"
echo "  3. Check DLQ topic: dlq.processing-errors.v1"
