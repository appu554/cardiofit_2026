#!/bin/bash

CURRENT_TIME=$(date +%s)000
echo "Sending test events at timestamp: $CURRENT_TIME"

# Event 1: Patient vital signs
echo "{\"patient_id\":\"P999\",\"event_time\":$CURRENT_TIME,\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":72,\"bp\":\"118/76\"},\"metadata\":{\"source\":\"Test\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

# Event 2: Medication
echo "{\"patient_id\":\"P999\",\"event_time\":$CURRENT_TIME,\"type\":\"medication\",\"payload\":{\"drug\":\"Metformin\",\"dose\":\"500mg\"},\"metadata\":{\"source\":\"Test\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1

# Event 3: Observation
echo "{\"patient_id\":\"P999\",\"event_time\":$CURRENT_TIME,\"type\":\"observation\",\"payload\":{\"test\":\"Glucose\",\"value\":105},\"metadata\":{\"source\":\"Test\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1

echo "✓ Test events sent successfully"
