#!/bin/bash

# Test Module 2 enrichment with Rohan Sharma synthetic data
# This script:
# 1. Sends a patient event to Kafka (patient-events-v1)
# 2. Module 1 validates and enriches to canonical format
# 3. Module 2 enriches with FHIR + Neo4j data
# 4. Verifies enriched output in clinical-patterns.v1 topic

set -e

echo "========================================================================"
echo "Testing Module 2 Enrichment with Rohan Sharma Synthetic Data"
echo "========================================================================"
echo ""

# Check if Kafka is running
if ! docker ps | grep -q kafka; then
    echo "❌ Kafka container not running. Please start Kafka first."
    echo "   Run: docker-compose up -d kafka"
    exit 1
fi

# Check if Flink is running
if ! curl -s http://localhost:8081/overview > /dev/null; then
    echo "❌ Flink not running. Please start Flink cluster first."
    echo "   Run: ./start-flink.sh"
    exit 1
fi

echo "✅ Kafka and Flink are running"
echo ""

# Patient event for Rohan Sharma
PATIENT_EVENT=$(cat <<'EOF'
{
  "id": "evt-rohan-20251010-001",
  "patientId": "PAT-ROHAN-001",
  "encounterId": "ENC-ROHAN-20251010",
  "eventType": "VITAL_SIGN",
  "eventTime": 1728547500000,
  "sourceSystem": "clinic-system",
  "payload": {
    "heart_rate": 88,
    "blood_pressure_systolic": 150,
    "blood_pressure_diastolic": 96,
    "temperature": 36.8,
    "oxygen_saturation": 97,
    "respiratory_rate": 16,
    "measurement_location": "Cardiology Clinic - JP Nagar"
  }
}
EOF
)

echo "📤 Sending patient event to patient-events-v1..."
echo "$PATIENT_EVENT" | docker exec -i kafka kafka-console-producer \
    --bootstrap-server localhost:9092 \
    --topic patient-events-v1

echo "✅ Event sent successfully"
echo ""

# Wait for processing
echo "⏳ Waiting 5 seconds for Module 1 & 2 processing..."
sleep 5

# Check enriched output
echo ""
echo "========================================================================"
echo "📊 Checking Enriched Output in clinical-patterns.v1"
echo "========================================================================"
echo ""

# Consume from clinical-patterns topic (Module 2 output)
echo "🔍 Fetching enriched event (timeout 10s)..."
ENRICHED_OUTPUT=$(timeout 10 docker exec kafka kafka-console-consumer \
    --bootstrap-server localhost:9092 \
    --topic clinical-patterns.v1 \
    --from-beginning \
    --max-messages 1 \
    2>/dev/null || echo "")

if [ -z "$ENRICHED_OUTPUT" ]; then
    echo "⚠️  No enriched output found in clinical-patterns.v1"
    echo ""
    echo "🔍 Troubleshooting steps:"
    echo "  1. Check Module 1 output: docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic enriched-patient-events-v1 --from-beginning --max-messages 1"
    echo "  2. Check Flink job status: http://localhost:8081"
    echo "  3. Check Flink logs: docker logs flink-taskmanager"
    echo "  4. Verify FHIR data exists: curl -H \"Authorization: Bearer \$(gcloud auth print-access-token)\" https://healthcare.googleapis.com/v1/projects/cardiofit-ehr/locations/us-central1/datasets/cardiofit-fhir-dataset/fhirStores/cardiofit-fhir-store/fhir/Patient/PAT-ROHAN-001"
    exit 1
fi

echo "✅ Enriched event received!"
echo ""
echo "========================================================================"
echo "📋 Enriched Event Details"
echo "========================================================================"

# Parse and display enriched event
echo "$ENRICHED_OUTPUT" | python3 -m json.tool 2>/dev/null || echo "$ENRICHED_OUTPUT"

echo ""
echo "========================================================================"
echo "🔍 Verification Checklist"
echo "========================================================================"
echo ""

# Check for FHIR enrichment
if echo "$ENRICHED_OUTPUT" | grep -q "PAT-ROHAN-001"; then
    echo "✅ Patient ID present"
else
    echo "❌ Patient ID missing"
fi

# Check for patient context
if echo "$ENRICHED_OUTPUT" | grep -q "patientContext"; then
    echo "✅ Patient context enriched"

    # Check for specific FHIR data
    if echo "$ENRICHED_OUTPUT" | grep -q "demographics\|Sharma"; then
        echo "  ✅ Demographics from FHIR"
    fi

    if echo "$ENRICHED_OUTPUT" | grep -q "activeMedications\|Telmisartan"; then
        echo "  ✅ Medications from FHIR"
    fi

    if echo "$ENRICHED_OUTPUT" | grep -q "chronicConditions\|hypertension\|prediabetes"; then
        echo "  ✅ Conditions from FHIR"
    fi
else
    echo "❌ Patient context missing"
fi

# Check for Neo4j enrichment
if echo "$ENRICHED_OUTPUT" | grep -q "careTeam\|Priya"; then
    echo "✅ Care team from Neo4j"
fi

if echo "$ENRICHED_OUTPUT" | grep -q "Metabolic\|cohort"; then
    echo "✅ Risk cohort from Neo4j"
fi

# Check for enrichment metadata
if echo "$ENRICHED_OUTPUT" | grep -q "enrichmentData"; then
    echo "✅ Enrichment metadata present"

    if echo "$ENRICHED_OUTPUT" | grep -q "state_version"; then
        echo "  ✅ State version tracked"
    fi

    if echo "$ENRICHED_OUTPUT" | grep -q "was_new_patient"; then
        echo "  ✅ First-time patient detection"
    fi
fi

# Check for clinical scores
if echo "$ENRICHED_OUTPUT" | grep -q "clinicalScores"; then
    echo "✅ Clinical scores calculated"

    if echo "$ENRICHED_OUTPUT" | grep -q "mews_score"; then
        echo "  ✅ MEWS score present"
    fi

    if echo "$ENRICHED_OUTPUT" | grep -q "qsofa_score"; then
        echo "  ✅ qSOFA score present"
    fi
fi

# Check for risk indicators
if echo "$ENRICHED_OUTPUT" | grep -q "riskIndicators"; then
    echo "✅ Risk indicators generated"

    if echo "$ENRICHED_OUTPUT" | grep -q "hypertension\|tachycardia"; then
        echo "  ✅ Vital sign risk flags"
    fi
fi

# Check for immediate alerts
if echo "$ENRICHED_OUTPUT" | grep -q "immediateAlerts"; then
    echo "✅ Immediate alerts checked"

    ALERT_COUNT=$(echo "$ENRICHED_OUTPUT" | grep -o "alertId" | wc -l)
    if [ "$ALERT_COUNT" -gt 0 ]; then
        echo "  ⚠️  $ALERT_COUNT alerts generated (BP threshold breach expected)"
    else
        echo "  ✅ No critical alerts"
    fi
fi

echo ""
echo "========================================================================"
echo "✅ Module 2 Enrichment Test Complete!"
echo "========================================================================"
echo ""
echo "🔍 Additional Verification:"
echo "  - Flink Web UI: http://localhost:8081"
echo "  - Kafka UI (if running): http://localhost:8080"
echo "  - Neo4j Browser: http://localhost:7474"
echo ""
echo "📊 View full enrichment data:"
echo "  timeout 10 docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic clinical-patterns.v1 --from-beginning --max-messages 5 | jq ."
echo ""
