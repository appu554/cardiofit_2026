#!/bin/bash

echo "🧪 Sepsis Pattern Detection Test - Baseline → Early Warning → Deterioration"
echo "=============================================================================="

# Get timestamps for sequential events (5 minutes apart to allow watermarks to advance)
NOW=$(date +%s)000
T0=$NOW                          # Baseline: Normal vitals
T1=$((NOW + 300000))            # +5 min: Early warning (SIRS developing)
T2=$((NOW + 600000))            # +10 min: Deterioration (sepsis shock)

KAFKA_BROKER="kafka:29092"
TOPIC="patient-events-v1"
PATIENT="PAT-ROHAN-001"

echo ""
echo "📋 Test Sequence:"
echo "  Event 1 (T+0):   BASELINE - Normal vitals (temp 37.0°C, HR 75, SBP 120)"
echo "  Event 2 (T+5m):  EARLY WARNING - qSOFA developing (temp 38.5°C, HR 105, RR 24, SBP 95)"
echo "  Event 3 (T+10m): DETERIORATION - Sepsis shock (temp 39.5°C, HR 125, RR 28, SBP 85)"
echo ""

# Event 1: BASELINE - Patient with normal vitals (matches baseline criteria)
echo "📤 Event 1/3: BASELINE - Normal vitals, patient stable"
docker exec kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic $TOPIC <<EOF
{"patient_id":"$PATIENT","event_time":$T0,"type":"vital_signs","source":"icu_monitor","payload":{"temperature":37.0,"heartRate":75,"respiratoryRate":16,"systolicBP":120,"diastolicBP":80,"oxygenSaturation":98,"consciousness":"Alert"}}
EOF

if [ $? -eq 0 ]; then
    echo "   ✅ Baseline event sent (Temp: 37.0°C, HR: 75, RR: 16, SBP: 120)"
else
    echo "   ❌ Failed to send baseline event"
    exit 1
fi

sleep 6

# Event 2: EARLY WARNING - qSOFA criteria developing (tachypnea + borderline hypotension + fever)
echo "📤 Event 2/3: EARLY WARNING - SIRS criteria developing"
docker exec kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic $TOPIC <<EOF
{"patient_id":"$PATIENT","event_time":$T1,"type":"vital_signs","source":"icu_monitor","payload":{"temperature":38.5,"heartRate":105,"respiratoryRate":24,"systolicBP":95,"diastolicBP":60,"oxygenSaturation":94,"consciousness":"Alert"}}
EOF

if [ $? -eq 0 ]; then
    echo "   ✅ Early warning event sent (Temp: 38.5°C, HR: 105, RR: 24, SBP: 95)"
else
    echo "   ❌ Failed to send early warning event"
    exit 1
fi

sleep 6

# Event 3: DETERIORATION - Sepsis shock (hypotension, hypoxia, altered mental status)
echo "📤 Event 3/3: DETERIORATION - Sepsis shock with organ dysfunction"
docker exec kafka kafka-console-producer --broker-list $KAFKA_BROKER --topic $TOPIC <<EOF
{"patient_id":"$PATIENT","event_time":$T2,"type":"vital_signs","source":"icu_monitor","payload":{"temperature":39.5,"heartRate":125,"respiratoryRate":28,"systolicBP":85,"diastolicBP":50,"oxygenSaturation":89,"consciousness":"Confused"}}
EOF

if [ $? -eq 0 ]; then
    echo "   ✅ Deterioration event sent (Temp: 39.5°C, HR: 125, RR: 28, SBP: 85)"
else
    echo "   ❌ Failed to send deterioration event"
    exit 1
fi

echo ""
echo "✅ Test Complete - 3 sequential events sent"
echo ""
echo "📊 Expected CEP Pattern Match:"
echo "   🔍 BASELINE matched:      Temp 37.0°C (36-38°C ✓), HR 75 (60-110 ✓), SBP 120 (≥90 ✓)"
echo "   🔍 EARLY_WARNING matched: RR 24 (≥22 ✓), SBP 95 (≤100 ✓), Fever 38.5°C ✓"
echo "   🔍 DETERIORATION matched: SBP 85 (≤90 ✓), SpO2 89% (<92% ✓), Altered mental status ✓"
echo ""
echo "🕐 Wait 30 seconds for CEP pattern processing..."
echo "   (Watermarks need time to advance for pattern matching)"
echo ""
echo "📋 Then check Module 4 output:"
echo "   docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic clinical-patterns.v1 --from-beginning --max-messages 1"
