#!/bin/bash

# Continuous Event Generator for Module 6 Analytics
# Sends test events every 10 seconds to trigger Module 6 windowed analytics

echo "=================================================="
echo "Continuous Event Generator for Module 6"
echo "=================================================="
echo "This script will send test events every 10 seconds"
echo "Press Ctrl+C to stop"
echo ""

# Counter for event batches
BATCH=0

while true; do
    BATCH=$((BATCH + 1))
    CURRENT_TIME=$(date +%s)000

    echo "[$BATCH] Sending event batch at $(date '+%H:%M:%S') (timestamp: $CURRENT_TIME)"

    # Event 1: Patient vital signs
    echo "{\"patient_id\":\"PAT-ROHAN-001\",\"event_time\":$CURRENT_TIME,\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":$((RANDOM % 40 + 60)),\"bp\":\"$((RANDOM % 40 + 100))/$((RANDOM % 30 + 60))\"},\"metadata\":{\"source\":\"ContinuousTest\",\"batch\":$BATCH}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1 2>/dev/null

    # Event 2: Medication
    MEDICATIONS=("Metformin" "Lisinopril" "Aspirin" "Atorvastatin" "Levothyroxine")
    MED_INDEX=$((RANDOM % 5))
    echo "{\"patient_id\":\"PAT-ROHAN-001\",\"event_time\":$CURRENT_TIME,\"type\":\"medication\",\"payload\":{\"drug\":\"${MEDICATIONS[$MED_INDEX]}\",\"dose\":\"$((RANDOM % 500 + 100))mg\"},\"metadata\":{\"source\":\"ContinuousTest\",\"batch\":$BATCH}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1 2>/dev/null

    # Event 3: Observation
    TESTS=("Glucose" "Hemoglobin" "Creatinine" "Sodium" "Potassium")
    TEST_INDEX=$((RANDOM % 5))
    echo "{\"patient_id\":\"PAT-ROHAN-001\",\"event_time\":$CURRENT_TIME,\"type\":\"observation\",\"payload\":{\"test\":\"${TESTS[$TEST_INDEX]}\",\"value\":$((RANDOM % 100 + 80))},\"metadata\":{\"source\":\"ContinuousTest\",\"batch\":$BATCH}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1 2>/dev/null

    echo "   ✓ Sent 3 events (vital signs, medication, observation)"

    # Check Module 6 output every 5 batches
    if [ $((BATCH % 5)) -eq 0 ]; then
        echo ""
        echo "   📊 Checking Module 6 output topics..."
        CENSUS_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic analytics-patient-census --time -1 2>/dev/null | awk -F: '{sum += $3} END {print sum}')
        ALERT_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic analytics-alert-metrics --time -1 2>/dev/null | awk -F: '{sum += $3} END {print sum}')
        ML_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic analytics-ml-performance --time -1 2>/dev/null | awk -F: '{sum += $3} END {print sum}')

        echo "   📈 analytics-patient-census: ${CENSUS_COUNT:-0} messages"
        echo "   📈 analytics-alert-metrics: ${ALERT_COUNT:-0} messages"
        echo "   📈 analytics-ml-performance: ${ML_COUNT:-0} messages"
        echo ""
    fi

    # Wait 10 seconds before next batch
    sleep 10
done
