#!/bin/bash

# Valid Alert Generator - Sends properly formatted single-line JSON alerts

echo "=================================================="
echo "Valid Alert Generator for Module 6 Analytics"
echo "=================================================="
echo "Sending 20 properly formatted alerts"
echo ""

TIMESTAMP=$(date +%s)000
BATCH_ID=$(date +%s)

# Patient IDs for diversity
PATIENTS=("PAT-001" "PAT-002" "PAT-003" "PAT-004" "PAT-005")

# Alert types (matching AlertType enum)
ALERT_TYPES=("VITAL_THRESHOLD_BREACH" "LAB_CRITICAL_VALUE" "MEDICATION_MISSED" "CLINICAL_SCORE_HIGH" "SEPSIS")

# Severities (matching AlertSeverity enum)
SEVERITIES=("CRITICAL" "HIGH" "WARNING" "INFO")

echo "🚨 Sending 20 valid alerts..."

for i in {1..20}; do
    # Random selections
    PATIENT=${PATIENTS[$((RANDOM % 5))]}
    ALERT_TYPE=${ALERT_TYPES[$((RANDOM % 5))]}
    SEVERITY=${SEVERITIES[$((RANDOM % 4))]}

    # Timestamp with offset
    EVENT_TIME=$((TIMESTAMP + i * 1000))

    # Create single-line JSON (no newlines, properly escaped)
    ALERT_JSON="{\"alert_id\":\"ALT-${BATCH_ID}-${i}\",\"patient_id\":\"${PATIENT}\",\"alert_type\":\"${ALERT_TYPE}\",\"severity\":\"${SEVERITY}\",\"message\":\"Test alert ${i} - ${ALERT_TYPE} for ${PATIENT}\",\"timestamp\":${EVENT_TIME},\"source_module\":\"MODULE_2_TEST\",\"context\":{\"test\":\"valid_alert_test\",\"batch\":\"${BATCH_ID}\",\"alert_number\":${i}}}"

    # Send to Kafka
    echo "$ALERT_JSON" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic alert-management.v1 2>/dev/null

    echo "  ✓ ${i}/20: ${PATIENT} - ${ALERT_TYPE} - ${SEVERITY}"

    sleep 0.1
done

echo ""
echo "✅ Sent 20 valid single-line JSON alerts"
echo ""
echo "⏳ Waiting 10 seconds for processing..."
sleep 10

echo ""
echo "📊 Checking results..."

# Check composed alerts
COMPOSED_BEFORE=10
COMPOSED_AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic composed-alerts.v1 --time -1 2>/dev/null | awk -F: '{sum += $3} END {print sum}')
echo "   composed-alerts.v1: ${COMPOSED_AFTER} messages (was ${COMPOSED_BEFORE})"

# Check analytics
ANALYTICS_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic analytics-alert-metrics --time -1 2>/dev/null | awk -F: '{sum += $3} END {print sum}')
echo "   analytics-alert-metrics: ${ANALYTICS_COUNT} messages"

echo ""
echo "💡 New alerts composed: $((COMPOSED_AFTER - COMPOSED_BEFORE))"
echo "   (Should be > 0 if Module 6 Alert Composition processed the new alerts)"
