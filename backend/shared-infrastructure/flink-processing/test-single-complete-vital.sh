#!/bin/bash

# Send a single vital signs event with ALL vitals and CURRENT timestamp
# This tests if Module 2 properly extracts and Module 4 can detect patterns

KAFKA_BROKER="localhost:9092"
PATIENT_ID="PAT-TEST-COMPLETE"
CURRENT_TIME=$(date +%s)000

echo "Sending complete vital signs event with ALL 6 vitals..."
echo "Timestamp: $CURRENT_TIME ($(date))"

EVENT='{"patient_id":"'$PATIENT_ID'","event_time":'$CURRENT_TIME',"type":"vital_signs","source":"test_monitor","payload":{"temperature":39.5,"heartRate":110,"respiratoryRate":26,"systolicBP":85,"diastolicBP":55,"oxygenSaturation":92,"consciousness":"Alert"}}'

echo "$EVENT"
echo ""
echo "$EVENT" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1

echo "✅ Event sent!"
echo ""
echo "Wait 10 seconds and check Module 4 debug logs:"
echo "docker logs flink-taskmanager-1-2.1 2>&1 | grep 'DEBUG - Patient $PATIENT_ID' | tail -3"
