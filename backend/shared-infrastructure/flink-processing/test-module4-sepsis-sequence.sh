#!/bin/bash

# Test Module 4 Sepsis Pattern Detection
# Sends a realistic sequence of events showing sepsis progression

echo "🧪 Module 4 Test: Sepsis Pattern Detection"
echo "=========================================="
echo ""

KAFKA_BROKER="localhost:9092"
TOPIC="clinical-patterns.v1"  # Module 3 input (Module 2 output)

# Base timestamp - use current time
BASE_TIME=$(date +%s)000

# Event 1: Initial presentation - infection markers (T=0)
echo "📤 Event 1: Infection markers detected"
EVENT1=$(cat <<EOF
{
  "patientId": "TEST-SEPSIS-001",
  "eventType": "LAB_RESULT",
  "eventTime": ${BASE_TIME},
  "processingTime": $(date +%s)000,
  "patientState": {
    "patientId": "TEST-SEPSIS-001",
    "recentLabs": {
      "6690-2": {"labCode": "6690-2", "labType": "WBC", "value": 15000, "unit": "cells/μL", "abnormal": true},
      "33959-8": {"labCode": "33959-8", "labType": "Procalcitonin", "value": 1.2, "unit": "ng/mL", "abnormal": true}
    },
    "latestVitals": {
      "temperature": 37.8,
      "heartrate": 88,
      "respiratoryrate": 18,
      "systolicbp": 120,
      "oxygensaturation": 97
    },
    "riskIndicators": {
      "leukocytosis": true,
      "sepsisRisk": false
    },
    "news2Score": 2,
    "qsofaScore": 0
  }
}
EOF
)

echo "$EVENT1" | docker exec -i kafka kafka-console-producer \
  --broker-list $KAFKA_BROKER \
  --topic $TOPIC 2>/dev/null

echo "✅ Event 1 sent (Infection: WBC 15,000, PCT 1.2)"
sleep 2

# Event 2: SIRS criteria developing (T+30 minutes)
echo "📤 Event 2: SIRS criteria developing"
SIRS_TIME=$((BASE_TIME + 1800000))  # +30 minutes
EVENT2=$(cat <<EOF
{
  "patientId": "TEST-SEPSIS-001",
  "eventType": "VITAL_SIGN",
  "eventTime": ${SIRS_TIME},
  "processingTime": $(date +%s)000,
  "patientState": {
    "patientId": "TEST-SEPSIS-001",
    "latestVitals": {
      "temperature": 38.9,
      "heartrate": 115,
      "respiratoryrate": 24,
      "systolicbp": 110,
      "diastolicbp": 70,
      "oxygensaturation": 94
    },
    "riskIndicators": {
      "fever": true,
      "tachycardia": true,
      "tachypnea": true,
      "hypoxia": true,
      "sepsisRisk": true
    },
    "news2Score": 6,
    "qsofaScore": 0,
    "activeAlerts": [
      {
        "alert_type": "SEPSIS_PATTERN",
        "severity": "WARNING",
        "message": "SIRS criteria met (3/4) - Consider sepsis workup"
      }
    ]
  }
}
EOF
)

echo "$EVENT2" | docker exec -i kafka kafka-console-producer \
  --broker-list $KAFKA_BROKER \
  --topic $TOPIC 2>/dev/null

echo "✅ Event 2 sent (SIRS: Fever 38.9°C, HR 115, RR 24)"
sleep 2

# Event 3: Organ dysfunction (T+90 minutes)
echo "📤 Event 3: Organ dysfunction developing"
ORGAN_TIME=$((BASE_TIME + 5400000))  # +90 minutes
EVENT3=$(cat <<EOF
{
  "patientId": "TEST-SEPSIS-001",
  "eventType": "VITAL_SIGN",
  "eventTime": ${ORGAN_TIME},
  "processingTime": $(date +%s)000,
  "patientState": {
    "patientId": "TEST-SEPSIS-001",
    "recentLabs": {
      "2524-7": {"labCode": "2524-7", "labType": "Lactate", "value": 3.2, "unit": "mmol/L", "abnormal": true}
    },
    "latestVitals": {
      "temperature": 39.2,
      "heartrate": 125,
      "respiratoryrate": 28,
      "systolicbp": 88,
      "diastolicbp": 55,
      "oxygensaturation": 90
    },
    "riskIndicators": {
      "fever": true,
      "tachycardia": true,
      "tachypnea": true,
      "hypoxia": true,
      "hypotension": true,
      "elevatedLactate": true,
      "severelyElevatedLactate": true,
      "sepsisRisk": true
    },
    "news2Score": 9,
    "qsofaScore": 2,
    "activeAlerts": [
      {
        "alert_type": "CLINICAL",
        "severity": "HIGH",
        "message": "SEPSIS LIKELY - SIRS criteria with elevated lactate and hypotension"
      }
    ]
  }
}
EOF
)

echo "$EVENT3" | docker exec -i kafka kafka-console-producer \
  --broker-list $KAFKA_BROKER \
  --topic $TOPIC 2>/dev/null

echo "✅ Event 3 sent (Organ Dysfunction: SBP 88, Lactate 3.2, SpO2 90%)"

echo ""
echo "📊 Test Sequence Complete!"
echo "=========================================="
echo ""
echo "Timeline:"
echo "  T+0min:  Infection markers (WBC 15k, PCT 1.2)"
echo "  T+30min: SIRS criteria (Fever, Tachycardia, Tachypnea)"
echo "  T+90min: Organ dysfunction (Hypotension, Elevated lactate)"
echo ""
echo "🔍 Expected Module 4 Behavior:"
echo "  1. Module 3 processes events → comprehensive-cds-events.v1"
echo "  2. Module 4 CEP detects sepsis SEQUENCE"
echo "  3. Output to alert-management.v1 within ~2-3 minutes"
echo ""
echo "📡 Monitor Output:"
echo "  # Watch Module 4 output topic for sepsis alert"
echo "  docker exec kafka kafka-console-consumer \\"
echo "    --bootstrap-server localhost:9092 \\"
echo "    --topic alert-management.v1 \\"
echo "    --from-beginning"
echo ""
echo "  # Check Module 4 metrics"
echo "  curl -s http://localhost:8081/jobs/0b67b802c3b2f6da1b18a8150826a4cb | \\"
echo "    jq '.vertices[0].metrics.\"read-records\"'"
