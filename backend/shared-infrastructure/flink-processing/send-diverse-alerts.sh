#!/bin/bash

# Diverse Alert Generator for Module 6 Analytics Testing
# Sends alerts with multiple patients and alert types to trigger analytics

echo "=================================================="
echo "Diverse Alert Generator for Module 6 Analytics"
echo "=================================================="
echo "Generating alerts with diverse patients and types"
echo ""

TIMESTAMP=$(date +%s)000
BATCH_ID=$(date +%s)

# Patient IDs for diversity
PATIENTS=("PAT-001" "PAT-002" "PAT-003" "PAT-004" "PAT-005")

# Alert types for diversity
ALERT_TYPES=("VITAL_THRESHOLD_BREACH" "LAB_CRITICAL_VALUE" "MEDICATION_MISSED" "CLINICAL_DETERIORATION" "SEPSIS_RISK")

# Severities
SEVERITIES=("CRITICAL" "HIGH" "WARNING" "INFO")

echo "🚨 Sending 20 diverse alerts..."

for i in {1..20}; do
    # Random selections for diversity
    PATIENT=${PATIENTS[$((RANDOM % 5))]}
    ALERT_TYPE=${ALERT_TYPES[$((RANDOM % 5))]}
    SEVERITY=${SEVERITIES[$((RANDOM % 4))]}

    # Timestamp with slight offset
    EVENT_TIME=$((TIMESTAMP + i * 1000))

    # Create SimpleAlert JSON
    ALERT_JSON=$(cat <<EOF
{
  "alert_id": "ALT-${BATCH_ID}-${i}",
  "patient_id": "${PATIENT}",
  "alert_type": "${ALERT_TYPE}",
  "severity": "${SEVERITY}",
  "message": "Test alert ${i} - ${ALERT_TYPE} for patient ${PATIENT}",
  "timestamp": ${EVENT_TIME},
  "context": {
    "test": "diverse_alert_test",
    "batch": "${BATCH_ID}",
    "alert_number": ${i}
  }
}
EOF
)

    # Send to alert-management.v1 topic
    echo "$ALERT_JSON" | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic alert-management.v1 2>/dev/null

    echo "  ✓ Sent: ${PATIENT} - ${ALERT_TYPE} - ${SEVERITY}"

    # Small delay to avoid overwhelming
    sleep 0.1
done

echo ""
echo "✅ Sent 20 diverse alerts with:"
echo "   - 5 different patients"
echo "   - 5 different alert types"
echo "   - 4 severity levels"
echo ""
echo "⏳ Waiting 5 seconds for processing..."
sleep 5

echo ""
echo "📊 Checking Module 6 metrics..."

# Check composed alerts count
COMPOSED_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic composed-alerts.v1 --time -1 2>/dev/null | awk -F: '{sum += $3} END {print sum}')
echo "   composed-alerts.v1: ${COMPOSED_COUNT} total messages"

# Check analytics output count
ANALYTICS_COUNT=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic analytics-alert-metrics --time -1 2>/dev/null | awk -F: '{sum += $3} END {print sum}')
echo "   analytics-alert-metrics: ${ANALYTICS_COUNT} total messages"

echo ""
echo "💡 Note: Wait 1-2 minutes for window aggregation to complete"
echo "   Then check analytics-alert-metrics topic for updated metrics with patient_ids"
