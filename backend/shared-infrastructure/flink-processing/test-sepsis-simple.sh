#!/bin/bash

KAFKA_BROKER="localhost:9092"
PATIENT_ID="PAT-SEPSIS-SIMPLE"
CURRENT_TIME=$(date +%s)000

echo "🔬 Simple 3-Event Sepsis Pattern Test"
echo "======================================"
echo "Patient: $PATIENT_ID"
echo "Current Time: $CURRENT_TIME"
echo ""

# Event 1: Baseline vitals (NOW)
T1=$CURRENT_TIME
echo "📤 Event 1/3: Baseline vitals (T+0h)"
EVENT1='{"patient_id":"'$PATIENT_ID'","event_time":'$T1',"type":"vital_signs","source":"test","payload":{"temperature":37.0,"heartRate":75,"respiratoryRate":16,"systolicBP":120,"diastolicBP":80,"oxygenSaturation":98}}'
echo "$EVENT1" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent (T=$T1)"
sleep 2

# Event 2: Early warning (T+2h = 7200 seconds)
T2=$((CURRENT_TIME + 7200000))
echo "📤 Event 2/3: Early warning - fever + tachycardia (T+2h)"
EVENT2='{"patient_id":"'$PATIENT_ID'","event_time":'$T2',"type":"vital_signs","source":"test","payload":{"temperature":39.2,"heartRate":105,"respiratoryRate":24,"systolicBP":100,"diastolicBP":70,"oxygenSaturation":94}}'
echo "$EVENT2" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent (T=$T2)"
sleep 2

# Event 3: Deterioration (T+4h = 14400 seconds)
T3=$((CURRENT_TIME + 14400000))
echo "📤 Event 3/3: Deterioration - septic shock (T+4h)"
EVENT3='{"patient_id":"'$PATIENT_ID'","event_time":'$T3',"type":"vital_signs","source":"test","payload":{"temperature":39.5,"heartRate":125,"respiratoryRate":28,"systolicBP":85,"diastolicBP":55,"oxygenSaturation":90}}'
echo "$EVENT3" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent (T=$T3)"

echo ""
echo "🎉 All 3 Events Sent!"
echo "Pattern: baseline(T+0h) → early_warning(T+2h) → deterioration(T+4h)"
echo "Window: 4 hours (within 6-hour sepsis window)"
