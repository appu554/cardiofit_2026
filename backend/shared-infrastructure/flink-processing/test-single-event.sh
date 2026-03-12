#!/bin/bash
CURRENT_TIME=$(date +%s)000
echo "{\"patient_id\":\"PAT-ROHAN-001\",\"event_time\":$CURRENT_TIME,\"type\":\"patient_admission\",\"payload\":{\"admission_type\":\"EMERGENCY\"},\"metadata\":{\"source\":\"Test\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1
