#!/bin/bash

echo "📨 Sending demonstration event to Kafka..."
echo ""

TIMESTAMP=$(date +%s)000
echo "Event timestamp: $TIMESTAMP"
echo ""

# The event we're sending
cat << EOF
Event data:
{
  "patient_id": "DEMO-123",
  "event_time": $TIMESTAMP,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 82,
    "blood_pressure": "125/82",
    "temperature": 98.4,
    "spo2": 98
  },
  "metadata": {
    "source": "Demo Test",
    "location": "ICU-5"
  }
}
EOF

echo ""
echo "Sending to topic: patient-events-v1"

# Send the event (single-line JSON)
echo "{\"patient_id\":\"DEMO-123\",\"event_time\":$TIMESTAMP,\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":82,\"blood_pressure\":\"125/82\",\"temperature\":98.4,\"spo2\":98},\"metadata\":{\"source\":\"Demo Test\",\"location\":\"ICU-5\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

echo ""
echo "✅ Event sent successfully!"
echo ""
echo "⏳ Waiting 5 seconds for Flink to process..."
sleep 5

echo ""
echo "📊 Checking results..."
echo ""

# Check input topic
echo "Input topic (patient-events-v1) message count:"
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic patient-events-v1 --time -1 2>/dev/null | awk -F: '{sum+=$3} END {print "  Total messages: " sum}'

echo ""

# Check output topic
echo "Output topic (enriched-patient-events-v1) message count:"
docker exec kafka kafka-run-class kafka.tools.GetOffsetShell \
  --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null | awk -F: '{sum+=$3} END {print "  Total messages: " sum}'

echo ""
echo "🎯 How to view the enriched event:"
echo ""
echo "  Option 1 (Visual - Recommended):"
echo "    1. Open http://localhost:8080 in your browser"
echo "    2. Click 'Topics' in the sidebar"
echo "    3. Find and click 'enriched-patient-events-v1'"
echo "    4. Click 'Messages' tab"
echo "    5. See the enriched event with added metadata!"
echo ""
echo "  Option 2 (Command Line):"
echo "    docker exec kafka kafka-console-consumer \\"
echo "      --bootstrap-server localhost:9092 \\"
echo "      --topic enriched-patient-events-v1 \\"
echo "      --from-beginning --max-messages 1 --timeout-ms 5000"
echo ""
echo "  Option 3 (Flink Metrics):"
echo "    Open http://localhost:8081 to see processing stats"
echo ""
