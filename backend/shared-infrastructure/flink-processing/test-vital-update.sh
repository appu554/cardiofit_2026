#!/bin/bash

KAFKA_BROKER="localhost:9092"
PATIENT_ID="PAT-ROHAN-001"
CURRENT_TIME=$(date +%s)000

echo "Sending test vital update with CURRENT timestamp..."
echo "Timestamp: $CURRENT_TIME"

# Septic shock vitals
EVENT='{"patient_id":"'$PATIENT_ID'","event_time":'$CURRENT_TIME',"type":"vital_signs","source":"test_monitor","payload":{"temperature":39.5,"heartRate":110,"respiratoryRate":26,"systolicBP":85,"diastolicBP":55,"oxygenSaturation":92,"consciousness":"Alert"}}'

echo "$EVENT"
echo ""
echo "$EVENT" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1

echo "✅ Sent septic shock vitals with CURRENT timestamp!"
