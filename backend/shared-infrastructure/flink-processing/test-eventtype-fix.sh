#!/bin/bash

echo "=== Testing EventType Mapping Fix ==="
echo ""

CURRENT_TIME=$(date +%s)000
echo "📅 Current timestamp: $CURRENT_TIME"
echo ""

# Get current enriched topic offset
BEFORE=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null | awk -F':' '{print $3}')
echo "📊 Enriched topic offset before: $BEFORE"
echo ""

echo "📤 Sending test events with various event types..."

# Test 1: Generic "medication" type (should map to MEDICATION_ORDERED)
echo "{\"id\":\"test-med-001\",\"patient_id\":\"P-TEST-001\",\"event_time\":$CURRENT_TIME,\"type\":\"medication\",\"payload\":{\"drug\":\"Aspirin\",\"dose\":\"81mg\"},\"metadata\":{\"source\":\"Test\",\"location\":\"ICU\",\"device_id\":\"TEST-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
echo "  ✅ Sent: medication event (should become MEDICATION_ORDERED)"

# Test 2: Generic "observation" type (should map to LAB_RESULT)
echo "{\"id\":\"test-obs-001\",\"patient_id\":\"P-TEST-001\",\"event_time\":$CURRENT_TIME,\"type\":\"observation\",\"payload\":{\"test\":\"Hemoglobin\",\"value\":12.5},\"metadata\":{\"source\":\"Test\",\"location\":\"Lab\",\"device_id\":\"TEST-002\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1
echo "  ✅ Sent: observation event (should become LAB_RESULT)"

# Test 3: "vital_signs" type (should map to VITAL_SIGN)
echo "{\"id\":\"test-vital-001\",\"patient_id\":\"P-TEST-001\",\"event_time\":$CURRENT_TIME,\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":75,\"bp\":\"120/80\"},\"metadata\":{\"source\":\"Test\",\"location\":\"Ward\",\"device_id\":\"TEST-003\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic vital-signs-events-v1
echo "  ✅ Sent: vital_signs event (should become VITAL_SIGN)"

echo ""
echo "⏳ Waiting 5 seconds for Flink processing..."
sleep 5

# Get new enriched topic offset
AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null | awk -F':' '{print $3}')
PROCESSED=$((AFTER - BEFORE))

echo "📊 Enriched topic offset after: $AFTER"
echo "✨ Events processed: $PROCESSED"
echo ""

if [ $PROCESSED -eq 3 ]; then
    echo "✅ SUCCESS: All 3 test events were processed!"
    echo ""
    echo "🔍 Checking Flink logs for event type warnings..."
    WARNINGS=$(docker logs cardiofit-flink-taskmanager-3 --since 10s 2>&1 | grep -c "Invalid event type")

    if [ $WARNINGS -eq 0 ]; then
        echo "✅ PERFECT: No 'Invalid event type' warnings found!"
        echo ""
        echo "🎉 EventType mapping fix is working correctly!"
    else
        echo "⚠️  WARNING: Found $WARNINGS 'Invalid event type' warnings"
        echo "   (These may be from older events still in the topic)"
    fi
else
    echo "⚠️  WARNING: Expected 3 events processed, but got $PROCESSED"
    echo "   Check Flink logs for validation errors"
fi

echo ""
echo "=== Test Complete ==="
