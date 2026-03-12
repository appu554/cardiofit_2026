#!/bin/bash

# ============================================================================
# Module 4 Sepsis Pattern Test - Module 1 Input Format
# ============================================================================
# This script sends a 3-event sequence to Module 1 input topics to trigger
# Module 4's sepsis CEP pattern: infection → SIRS → organ dysfunction
# ============================================================================

KAFKA_BROKER="localhost:9092"
PATIENT_ID="PAT-ROHAN-001"
BASE_TIME=$(date +%s)000  # Current time in milliseconds

echo "🔬 Module 4 Sepsis Pattern Test (Module 1 Input)"
echo "================================================="
echo "Patient: $PATIENT_ID"
echo "Base Time: $BASE_TIME"
echo ""

# ============================================================================
# EVENT 1: Infection Markers (T+0)
# Lab results showing leukocytosis and elevated procalcitonin
# ============================================================================

echo "📤 Event 1: Infection Markers (T+0)"
echo "   WBC: 15,000 (elevated - leukocytosis)"
echo "   Procalcitonin: 1.2 ng/mL (elevated - bacterial infection)"
echo ""

EVENT1=$(cat <<EOF
{
  "patient_id": "$PATIENT_ID",
  "event_time": $BASE_TIME,
  "type": "lab_result",
  "source": "central_lab_system",
  "payload": {
    "wbc": 15000,
    "procalcitonin": 1.2,
    "lactate": 1.5,
    "creatinine": 1.0,
    "bilirubin": 0.8,
    "platelet": 200000
  },
  "metadata": {
    "lab_order_id": "LAB-001-INFECTION",
    "test_type": "infection_panel",
    "urgency": "stat"
  }
}
EOF
)

echo "$EVENT1" | docker exec -i kafka kafka-console-producer \
  --broker-list $KAFKA_BROKER \
  --topic lab-result-events.v1

echo "✅ Event 1 sent to lab-result-events.v1"
echo ""

# Wait 30 minutes (simulated)
sleep 3

# ============================================================================
# EVENT 2: SIRS Criteria (T+30min)
# Vital signs meeting 3/4 SIRS criteria
# ============================================================================

SIRS_TIME=$((BASE_TIME + 1800000))  # +30 minutes

echo "📤 Event 2: SIRS Criteria Met (T+30min)"
echo "   Temperature: 39.2°C (fever)"
echo "   Heart Rate: 105 bpm (tachycardia)"
echo "   Respiratory Rate: 24 breaths/min (tachypnea)"
echo "   SIRS Score: 3/4"
echo ""

EVENT2=$(cat <<EOF
{
  "patient_id": "$PATIENT_ID",
  "event_time": $SIRS_TIME,
  "type": "vital_signs",
  "source": "bedside_monitor_001",
  "payload": {
    "temperature": 39.2,
    "heartRate": 105,
    "respiratoryRate": 24,
    "systolicBP": 115,
    "diastolicBP": 75,
    "oxygenSaturation": 94,
    "consciousness": "Alert",
    "supplementalOxygen": false
  },
  "metadata": {
    "device_id": "MONITOR-ICU-BED-12",
    "measurement_method": "continuous",
    "alert_triggered": true
  }
}
EOF
)

echo "$EVENT2" | docker exec -i kafka kafka-console-producer \
  --broker-list $KAFKA_BROKER \
  --topic vital-signs-events.v1

echo "✅ Event 2 sent to vital-signs-events.v1"
echo ""

# Wait 60 minutes (simulated)
sleep 3

# ============================================================================
# EVENT 3: Organ Dysfunction (T+90min)
# Hypotension + elevated lactate = septic shock
# ============================================================================

SHOCK_TIME=$((BASE_TIME + 5400000))  # +90 minutes

echo "📤 Event 3: Organ Dysfunction (T+90min) - SEPTIC SHOCK"
echo "   Systolic BP: 85 mmHg (hypotension)"
echo "   Lactate: 2.8 mmol/L (elevated - tissue hypoperfusion)"
echo "   Oxygen Saturation: 88% (hypoxia)"
echo ""

EVENT3_VITALS=$(cat <<EOF
{
  "patient_id": "$PATIENT_ID",
  "event_time": $SHOCK_TIME,
  "type": "vital_signs",
  "source": "bedside_monitor_001",
  "payload": {
    "temperature": 38.8,
    "heartRate": 118,
    "respiratoryRate": 28,
    "systolicBP": 85,
    "diastolicBP": 55,
    "oxygenSaturation": 88,
    "consciousness": "Confused",
    "supplementalOxygen": true
  },
  "metadata": {
    "device_id": "MONITOR-ICU-BED-12",
    "measurement_method": "continuous",
    "alert_triggered": true,
    "critical_alert": "hypotension"
  }
}
EOF
)

echo "$EVENT3_VITALS" | docker exec -i kafka kafka-console-producer \
  --broker-list $KAFKA_BROKER \
  --topic vital-signs-events.v1

echo "✅ Event 3a sent to vital-signs-events.v1"
echo ""

sleep 1

EVENT3_LABS=$(cat <<EOF
{
  "patient_id": "$PATIENT_ID",
  "event_time": $SHOCK_TIME,
  "type": "lab_result",
  "source": "point_of_care_lab",
  "payload": {
    "lactate": 2.8,
    "wbc": 16500,
    "procalcitonin": 2.5,
    "creatinine": 1.4,
    "bilirubin": 1.2,
    "platelet": 150000
  },
  "metadata": {
    "lab_order_id": "LAB-002-SEPSIS-PANEL",
    "test_type": "sepsis_panel",
    "urgency": "critical"
  }
}
EOF
)

echo "$EVENT3_LABS" | docker exec -i kafka kafka-console-producer \
  --broker-list $KAFKA_BROKER \
  --topic lab-result-events.v1

echo "✅ Event 3b sent to lab-result-events.v1"
echo ""

# ============================================================================
# Summary
# ============================================================================

echo "🎯 Sepsis Sequence Complete!"
echo "=============================="
echo ""
echo "📊 Expected Pipeline Flow:"
echo "  1. Module 1 (Ingestion)    → Validates and canonicalizes events"
echo "  2. Module 2_Enhanced       → Enriches with patient context"
echo "  3. Module 3 (CDS)          → Generates comprehensive CDS events"
echo "  4. Module 4 (CEP)          → Detects sepsis pattern!"
echo ""
echo "🔍 Expected Module 4 Output:"
echo "  Topic: alert-management.v1"
echo "  Pattern: SEPSIS_DETERIORATION"
echo "  Severity: CRITICAL"
echo "  Sequence: infection(T+0) → SIRS(T+30m) → shock(T+90m)"
echo ""
echo "📈 Monitor Output:"
echo "  docker exec kafka kafka-console-consumer \\"
echo "    --bootstrap-server localhost:9092 \\"
echo "    --topic alert-management.v1 \\"
echo "    --from-beginning"
echo ""
echo "🌐 Flink Web UI: http://localhost:8081"
echo "   Check Module 4 metrics for pattern matches"
echo ""
