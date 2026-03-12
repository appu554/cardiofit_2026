#!/bin/bash

echo "=== Full Pipeline Test: Module 1 → Module 2 ==="
echo ""

CURRENT_TIME=$(date +%s)000
PATIENT_ID="P-PIPELINE-TEST-$(date +%s)"

echo "📅 Current timestamp: $CURRENT_TIME"
echo "👤 Test Patient ID: $PATIENT_ID"
echo ""

# Get baseline counts
echo "📊 Baseline Message Counts:"
ENRICHED_BEFORE=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null | awk -F':' '{print $3}')
PATTERNS_BEFORE=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic clinical-patterns.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')
SNAPSHOTS_BEFORE=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic patient-context-snapshots.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')

echo "  enriched-patient-events-v1: $ENRICHED_BEFORE"
echo "  clinical-patterns.v1: $PATTERNS_BEFORE"
echo "  patient-context-snapshots.v1: $SNAPSHOTS_BEFORE"
echo ""

echo "📤 Sending comprehensive test event set for patient..."

# Event 1: Admission
echo "{\"id\":\"evt-admit-001\",\"patient_id\":\"$PATIENT_ID\",\"event_time\":$CURRENT_TIME,\"type\":\"admission\",\"payload\":{\"reason\":\"Chest pain\",\"department\":\"ER\"},\"metadata\":{\"source\":\"EHR\",\"location\":\"ER-Bay-3\",\"device_id\":\"ER-003\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1
echo "  ✅ Sent: Patient admission"

sleep 1
CURRENT_TIME=$((CURRENT_TIME + 1000))

# Event 2: Vital signs
echo "{\"id\":\"evt-vital-001\",\"patient_id\":\"$PATIENT_ID\",\"event_time\":$CURRENT_TIME,\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":88,\"bp\":\"145/95\",\"temp\":37.2,\"spo2\":96},\"metadata\":{\"source\":\"Monitor\",\"location\":\"ER-Bay-3\",\"device_id\":\"MON-ER-003\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic vital-signs-events-v1
echo "  ✅ Sent: Vital signs reading"

sleep 1
CURRENT_TIME=$((CURRENT_TIME + 1000))

# Event 3: Lab order
echo "{\"id\":\"evt-lab-001\",\"patient_id\":\"$PATIENT_ID\",\"event_time\":$CURRENT_TIME,\"type\":\"observation\",\"payload\":{\"test\":\"Troponin\",\"status\":\"ordered\"},\"metadata\":{\"source\":\"Lab\",\"location\":\"ER-Bay-3\",\"device_id\":\"LAB-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1
echo "  ✅ Sent: Lab test order"

sleep 1
CURRENT_TIME=$((CURRENT_TIME + 1000))

# Event 4: Medication order
echo "{\"id\":\"evt-med-001\",\"patient_id\":\"$PATIENT_ID\",\"event_time\":$CURRENT_TIME,\"type\":\"medication\",\"payload\":{\"drug\":\"Aspirin\",\"dose\":\"325mg\",\"route\":\"PO\"},\"metadata\":{\"source\":\"Pharmacy\",\"location\":\"ER-Bay-3\",\"device_id\":\"PHARM-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
echo "  ✅ Sent: Medication order"

echo ""
echo "⏳ Waiting 10 seconds for pipeline processing..."
sleep 10

# Get final counts
echo ""
echo "📊 Final Message Counts:"
ENRICHED_AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null | awk -F':' '{print $3}')
PATTERNS_AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic clinical-patterns.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')
SNAPSHOTS_AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic patient-context-snapshots.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')

echo "  enriched-patient-events-v1: $ENRICHED_AFTER (+$((ENRICHED_AFTER - ENRICHED_BEFORE)))"
echo "  clinical-patterns.v1: $PATTERNS_AFTER (+$((PATTERNS_AFTER - PATTERNS_BEFORE)))"
echo "  patient-context-snapshots.v1: $SNAPSHOTS_AFTER (+$((SNAPSHOTS_AFTER - SNAPSHOTS_BEFORE)))"
echo ""

# Analysis
MODULE1_PROCESSED=$((ENRICHED_AFTER - ENRICHED_BEFORE))
MODULE2_PATTERNS=$((PATTERNS_AFTER - PATTERNS_BEFORE))
MODULE2_SNAPSHOTS=$((SNAPSHOTS_AFTER - SNAPSHOTS_BEFORE))

echo "=== Pipeline Analysis ==="
echo ""

if [ $MODULE1_PROCESSED -eq 4 ]; then
    echo "✅ Module 1: All 4 events processed correctly"
else
    echo "⚠️  Module 1: Expected 4 events, got $MODULE1_PROCESSED"
fi

if [ $MODULE2_PATTERNS -gt 0 ] || [ $MODULE2_SNAPSHOTS -gt 0 ]; then
    echo "✅ Module 2: Generating output ($MODULE2_PATTERNS patterns, $MODULE2_SNAPSHOTS snapshots)"
else
    echo "⚠️  Module 2: No output generated (may need more time or data)"
fi

echo ""
echo "=== Test Complete ==="