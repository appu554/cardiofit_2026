#!/bin/bash

# CardioFit Sepsis Pattern Test - FIXED LAB FORMAT
# This version sends labs in the format Module 2 expects

KAFKA_BROKER="localhost:9092"
PATIENT_ID="PAT-ROHAN-001"

# Calculate timestamps
CURRENT_TIME=$(date +%s)000
BASE_TIME=$((CURRENT_TIME - 86400000))  # 24 hours ago

echo "🔬 24-Hour Sepsis Pattern Test - FIXED FORMAT"
echo "=============================================="
echo "Patient: $PATIENT_ID"
echo "Base Time: $BASE_TIME (24 hours ago)"
echo

# ============================================================================
# EVENT 1: Baseline vitals
# ============================================================================
echo "📤 Event 1/8: Baseline vitals (T+0h)"
T1=$BASE_TIME
EVENT1='{"patient_id":"'$PATIENT_ID'","event_time":'$T1',"type":"vital_signs","source":"bedside_monitor","payload":{"temperature":37.0,"heartRate":75,"respiratoryRate":16,"systolicBP":120,"diastolicBP":80,"oxygenSaturation":98,"consciousness":"Alert"}}'
echo "$EVENT1" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent"
sleep 1

# ============================================================================
# EVENT 2: WBC elevated (infection marker 1)
# ============================================================================
echo "📤 Event 2/8: WBC elevated - 15,000 (T+6h) [SEPSIS 1/3]"
T2=$((BASE_TIME + 21600000))
EVENT2='{"patient_id":"'$PATIENT_ID'","event_time":'$T2',"type":"lab_result","source":"central_lab","payload":{"labName":"WBC","value":15000,"unit":"cells/uL","loincCode":"6690-2","abnormal":true,"abnormalFlag":"H","referenceRangeLow":4000,"referenceRangeHigh":11000}}'
echo "$EVENT2" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent"
sleep 1

# ============================================================================
# EVENT 3: Procalcitonin elevated (infection marker 2)
# ============================================================================
echo "📤 Event 3/8: Procalcitonin 1.2 ng/mL (T+6h)"
EVENT3='{"patient_id":"'$PATIENT_ID'","event_time":'$T2',"type":"lab_result","source":"central_lab","payload":{"labName":"Procalcitonin","value":1.2,"unit":"ng/mL","loincCode":"33959-8","abnormal":true,"abnormalFlag":"H","referenceRangeLow":0,"referenceRangeHigh":0.5}}'
echo "$EVENT3" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent"
sleep 1

# ============================================================================
# EVENT 4: SIRS criteria met (temp, HR, RR abnormal)
# ============================================================================
echo "📤 Event 4/8: SIRS criteria - temp 39.2°C, HR 105, RR 24 (T+8h) [SEPSIS 2/3]"
T4=$((BASE_TIME + 28800000))
EVENT4='{"patient_id":"'$PATIENT_ID'","event_time":'$T4',"type":"vital_signs","source":"bedside_monitor","payload":{"temperature":39.2,"heartRate":105,"respiratoryRate":24,"systolicBP":115,"diastolicBP":75,"oxygenSaturation":94,"consciousness":"Alert"}}'
echo "$EVENT4" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent"
sleep 1

# ============================================================================
# EVENT 5: Lactate rising (organ dysfunction)
# ============================================================================
echo "📤 Event 5/8: Lactate 2.2 mmol/L (T+10h)"
T5=$((BASE_TIME + 36000000))
EVENT5='{"patient_id":"'$PATIENT_ID'","event_time":'$T5',"type":"lab_result","source":"poc_lab","payload":{"labName":"Lactate","value":2.2,"unit":"mmol/L","loincCode":"2524-7","abnormal":true,"abnormalFlag":"H","referenceRangeLow":0.5,"referenceRangeHigh":2.0}}'
echo "$EVENT5" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent"
sleep 1

# ============================================================================
# EVENT 6: Septic shock - hypotension + elevated lactate
# ============================================================================
echo "📤 Event 6/8: Septic shock - SBP 85 mmHg (T+12h) [SEPSIS 3/3 COMPLETE!]"
T6=$((BASE_TIME + 43200000))
EVENT6='{"patient_id":"'$PATIENT_ID'","event_time":'$T6',"type":"vital_signs","source":"bedside_monitor","payload":{"temperature":39.5,"heartRate":110,"respiratoryRate":26,"systolicBP":85,"diastolicBP":55,"oxygenSaturation":92,"consciousness":"Alert","supplementalOxygen":true}}'
echo "$EVENT6" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent - SEPSIS PATTERN COMPLETE!"
sleep 1

# ============================================================================
# EVENT 7: Lactate confirmation (still elevated)
# ============================================================================
echo "📤 Event 7/8: Lactate 2.8 mmol/L (T+12h)"
EVENT7='{"patient_id":"'$PATIENT_ID'","event_time":'$T6',"type":"lab_result","source":"poc_lab","payload":{"labName":"Lactate","value":2.8,"unit":"mmol/L","loincCode":"2524-7","abnormal":true,"abnormalFlag":"H","referenceRangeLow":0.5,"referenceRangeHigh":2.0}}'
echo "$EVENT7" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent"
sleep 1

# ============================================================================
# EVENT 8: WBC peak
# ============================================================================
echo "📤 Event 8/8: WBC 16,000 (T+12h)"
EVENT8='{"patient_id":"'$PATIENT_ID'","event_time":'$T6',"type":"lab_result","source":"central_lab","payload":{"labName":"WBC","value":16000,"unit":"cells/uL","loincCode":"6690-2","abnormal":true,"abnormalFlag":"H","referenceRangeLow":4000,"referenceRangeHigh":11000}}'
echo "$EVENT8" | docker exec -i kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic patient-events-v1
echo "✅ Sent"

echo
echo "🎉 All 8 Events Sent Successfully!"
echo "=================================="
echo
echo "📊 Sepsis Pattern Timeline:"
echo "  T+6h:  WBC 15,000 + PCT 1.2     [Infection]"
echo "  T+8h:  Temp 39.2 + HR 105 + RR 24  [SIRS]"
echo "  T+12h: SBP 85 + Lactate 2.8     [Septic Shock]"
echo
echo "🔍 Monitor Module 4 Output:"
echo "  docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic alert-management.v1 --from-beginning --max-messages 5"
