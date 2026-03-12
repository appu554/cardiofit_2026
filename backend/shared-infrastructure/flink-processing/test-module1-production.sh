#!/bin/bash
set -e

echo "================================================"
echo "Module 1 Production-Like Testing"
echo "Patient ID: 905a60cb-8241-418f-b29b-5b020e851392"
echo "================================================"

# Get current timestamp
CURRENT_TIME=$(python3 -c "import time; print(int(time.time() * 1000))")

echo ""
echo "📊 Step 1: Sending VALID events across different topics..."
echo ""

# 1. Vital Signs Event (to patient-events-v1)
echo "1️⃣  Sending Vital Signs..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":${CURRENT_TIME},\"type\":\"vital_signs\",\"source\":\"bedside-monitor\",\"payload\":{\"heart_rate\":125,\"blood_pressure_systolic\":165,\"blood_pressure_diastolic\":100,\"oxygen_saturation\":91,\"temperature\":39.0,\"respiratory_rate\":30},\"metadata\":{\"unit\":\"ICU-3B\",\"encounter_id\":\"ENC-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

sleep 1

# 2. Medication Event
echo "2️⃣  Sending Medication Administration..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":${CURRENT_TIME},\"type\":\"medication_administration\",\"source\":\"pharmacy-system\",\"payload\":{\"medication_name\":\"Lisinopril\",\"dosage\":\"10mg\",\"route\":\"oral\",\"frequency\":\"daily\"},\"metadata\":{\"prescriber_id\":\"DR-123\",\"encounter_id\":\"ENC-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1

sleep 1

# 3. Lab Result Event
echo "3️⃣  Sending Lab Results..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":${CURRENT_TIME},\"type\":\"lab_result\",\"source\":\"lab-system\",\"payload\":{\"test_name\":\"Troponin\",\"value\":0.8,\"unit\":\"ng/mL\",\"reference_range\":\"<0.04\",\"status\":\"CRITICAL\"},\"metadata\":{\"lab_id\":\"LAB-456\",\"encounter_id\":\"ENC-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1

sleep 1

# 4. Observation Event
echo "4️⃣  Sending Clinical Observation..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":${CURRENT_TIME},\"type\":\"observation\",\"source\":\"clinician-notes\",\"payload\":{\"observation_type\":\"pain_assessment\",\"pain_score\":8,\"location\":\"chest\",\"description\":\"Sharp chest pain radiating to left arm\"},\"metadata\":{\"clinician_id\":\"DR-789\",\"encounter_id\":\"ENC-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1

sleep 1

# 5. Device Data Event
echo "5️⃣  Sending Validated Device Data..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":${CURRENT_TIME},\"type\":\"device_reading\",\"source\":\"ecg-monitor\",\"payload\":{\"device_type\":\"ECG\",\"rhythm\":\"sinus_tachycardia\",\"heart_rate\":125,\"qt_interval\":420},\"metadata\":{\"device_id\":\"ECG-001\",\"encounter_id\":\"ENC-001\"}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic validated-device-data-v1

echo ""
echo "✅ All valid events sent successfully"
echo ""

echo "================================================"
echo "📊 Step 2: Sending INVALID events (should go to DLQ)..."
echo "================================================"
echo ""

# Invalid Event 1: Missing patient_id
echo "❌ 1. Missing patient_id..."
echo "{\"event_time\":${CURRENT_TIME},\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":120}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

sleep 1

# Invalid Event 2: Zero timestamp
echo "❌ 2. Zero/invalid timestamp..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":0,\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":120}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

sleep 1

# Invalid Event 3: Empty payload
echo "❌ 3. Empty payload..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":${CURRENT_TIME},\"type\":\"vital_signs\",\"payload\":{}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

sleep 1

# Invalid Event 4: Future timestamp (>1 hour)
FUTURE_TIME=$((CURRENT_TIME + 7200000))  # 2 hours in future
echo "❌ 4. Timestamp too far in future..."
echo "{\"patient_id\":\"905a60cb-8241-418f-b29b-5b020e851392\",\"event_time\":${FUTURE_TIME},\"type\":\"vital_signs\",\"payload\":{\"heart_rate\":120}}" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1

echo ""
echo "✅ All invalid events sent (should be in DLQ)"
echo ""

echo "================================================"
echo "⏳ Step 3: Waiting for processing (10 seconds)..."
echo "================================================"
sleep 10

echo ""
echo "================================================"
echo "📊 Step 4: Verifying Results"
echo "================================================"
echo ""

# Check message counts
echo "📈 Message Counts:"
INPUT_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic patient-events-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
ENRICHED_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
DLQ_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic dlq.processing-errors.v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
MED_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic medication-events-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
LAB_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic lab-result-events-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
OBS_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic observation-events-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')
DEVICE_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic validated-device-data-v1 --time -1 2>&1 | awk -F: '{sum+=$NF} END {print sum}')

echo "  📥 Input Topics:"
echo "    - patient-events-v1: ${INPUT_COUNT} messages"
echo "    - medication-events-v1: ${MED_COUNT} messages"
echo "    - lab-result-events-v1: ${LAB_COUNT} messages"
echo "    - observation-events-v1: ${OBS_COUNT} messages"
echo "    - validated-device-data-v1: ${DEVICE_COUNT} messages"
echo ""
echo "  📤 Output Topics:"
echo "    - enriched-patient-events-v1: ${ENRICHED_COUNT} messages (✅ Valid events)"
echo "    - dlq.processing-errors.v1: ${DLQ_COUNT} messages (❌ Invalid events)"
echo ""

TOTAL_INPUT=$((INPUT_COUNT + MED_COUNT + LAB_COUNT + OBS_COUNT + DEVICE_COUNT))
TOTAL_OUTPUT=$((ENRICHED_COUNT + DLQ_COUNT))

echo "  📊 Processing Summary:"
echo "    Total Input: ${TOTAL_INPUT}"
echo "    Total Processed: ${TOTAL_OUTPUT}"
echo "    Valid Events: ${ENRICHED_COUNT}"
echo "    Invalid Events (DLQ): ${DLQ_COUNT}"
echo ""

echo "================================================"
echo "📋 Step 5: Sample Enriched Events"
echo "================================================"
echo ""
echo "🔍 Latest Enriched Event (from any input topic):"
docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic enriched-patient-events-v1 --from-beginning --max-messages 1 --timeout-ms 5000 2>&1 | grep -v "ERROR\|Processed" | python3 -m json.tool 2>/dev/null || echo "No output yet"

echo ""
echo "================================================"
echo "🚨 Step 6: Sample DLQ Events (Validation Failures)"
echo "================================================"
echo ""
echo "❌ Latest DLQ Event:"
docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic dlq.processing-errors.v1 --from-beginning --max-messages 1 --timeout-ms 5000 2>&1 | grep -v "ERROR\|Processed" | head -20 || echo "No DLQ messages (all events valid)"

echo ""
echo "================================================"
echo "✅ Production Test Complete!"
echo "================================================"
echo ""
echo "Expected Results:"
echo "  ✅ 5 valid events → enriched-patient-events-v1"
echo "  ❌ 4 invalid events → dlq.processing-errors.v1"
echo ""
echo "🌐 View in Kafka UI: http://localhost:8080"
echo "🎯 View in Flink UI: http://localhost:8081"
echo ""
