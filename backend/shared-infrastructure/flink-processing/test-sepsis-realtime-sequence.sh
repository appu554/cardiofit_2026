#!/bin/bash

echo "🧪 Real-Time Sepsis Pattern Detection Test"
echo "=========================================="
echo ""
echo "This test sends 3 events with realistic timing to trigger Module 4 CEP patterns:"
echo "  Event 1 (T+0):    Infection markers (Fever, Elevated WBC)"
echo "  Event 2 (T+5s):   SIRS progression (Tachycardia, Tachypnea)"
echo "  Event 3 (T+10s):  Organ dysfunction (Hypotension, Hypoxia)"
echo ""

# Get current timestamp in milliseconds
NOW=$(date +%s)000

# Calculate event timestamps (using seconds for testing, real-world would use 30min/90min intervals)
T0=$NOW
T1=$((NOW + 5000))   # 5 seconds later
T2=$((NOW + 10000))  # 10 seconds later

KAFKA_BROKER="kafka:29092"
TOPIC="vital-signs-events-v1"
PATIENT="PAT-SEPSIS-TEST"

echo "📤 Event 1/3: Baseline - Infection markers detected"
echo "   Timestamp: $T0 ($(date -r $((T0/1000)) '+%H:%M:%S'))"

docker exec kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic $TOPIC <<EOF
{"patient_id":"PAT-ROHAN-001","event_time":"1761888900000","type":"vital_signs","source":"icu_monitor","payload":{"temperature":38.9,"heartRate":95,"respiratoryRate":20,"systolicBP":120,"diastolicBP":75,"oxygenSaturation":96,"consciousness":"Alert"}}
EOF

echo "✅ Event 1 sent - Fever detected (38.9°C)"
echo ""

sleep 6  # Wait for event processing + 1 second buffer

echo "📤 Event 2/3: Early Warning - SIRS criteria developing"
echo "   Timestamp: $T1 ($(date -r $((T1/1000)) '+%H:%M:%S'))"

docker exec kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic $TOPIC <<EOF
{"patient_id":"PAT-ROHAN-001","event_time":"1761888905000","type":"vital_signs","source":"icu_monitor","payload":{"temperature":39.5,"heartRate":115,"respiratoryRate":26,"systolicBP":115,"diastolicBP":70,"oxygenSaturation":94,"consciousness":"Alert"}}
EOF

echo "✅ Event 2 sent - Worsening: HR 115, RR 26, Temp 39.5°C"
echo ""

sleep 6  # Wait for event processing + 1 second buffer

echo "📤 Event 3/3: Deterioration - Organ dysfunction"
echo "   Timestamp: $T2 ($(date -r $((T2/1000)) '+%H:%M:%S'))"

docker exec kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic $TOPIC <<EOF
{"patient_id":"PAT-ROHAN-001","event_time":"1761888910000","type":"vital_signs","source":"icu_monitor","payload":{"temperature":39.8,"heartRate":125,"respiratoryRate":28,"systolicBP":85,"diastolicBP":50,"oxygenSaturation":89,"consciousness":"Confused"}}
EOF

echo "✅ Event 3 sent - Critical: SBP 85, SpO2 89%, Confused"
echo ""
echo "=========================================="
echo "📊 Test Sequence Complete!"
echo ""
echo "Timeline:"
echo "  $(date -r $((T0/1000)) '+%H:%M:%S') - Baseline (Fever)"
echo "  $(date -r $((T1/1000)) '+%H:%M:%S') - SIRS criteria (Tachycardia + Tachypnea + Fever)"
echo "  $(date -r $((T2/1000)) '+%H:%M:%S') - Organ dysfunction (Hypotension + Hypoxia + AMS)"
echo ""
echo "🔍 Expected CEP Behavior:"
echo "  Module 4 should detect SEQUENTIAL deterioration pattern:"
echo "  1. Baseline → Early Warning → Deterioration = SEPSIS ALERT"
echo ""
echo "📡 Monitor Output (wait 15-20 seconds for CEP processing):"
echo ""
echo "# Check Module 4 processing logs:"
echo "docker logs --since 30s flink-taskmanager-1-2.1 2>&1 | grep -E 'Module4|Pattern|Sepsis|PAT-SEPSIS-TEST' | tail -20"
echo ""
echo "# Check Module 4 operator metrics:"
echo "curl -s http://localhost:8081/jobs/d3cfe7593ad707dda826b49c39219b76 | jq '.vertices[] | select(.name | contains(\"Cep\")) | {name, read: .metrics[\"read-records\"]}' | head -5"
echo ""
echo "# Check output topics:"
echo "docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic clinical-patterns.v1 --time -1 2>/dev/null"
echo ""
