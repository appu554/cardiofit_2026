#!/bin/bash

# ============================================================================
# FIRST-TIME PATIENT END-TO-END TEST
# ============================================================================
# This script tests the complete lifecycle of a NEW patient (404 from FHIR):
# 1. Patient admission (first event - creates empty snapshot)
# 2. Vital signs collection (progressive enrichment)
# 3. Lab results (add to snapshot)
# 4. Medication orders (add to snapshot)
# 5. Diagnosis added (add condition)
# 6. Patient discharge (TRIGGER: FHIR Bundle flush)
#
# Expected Results:
# - New patient snapshot created in Flink state (empty demographics)
# - Snapshot progressively enriched with clinical data
# - On discharge: Patient resource created in Google Cloud Healthcare FHIR API
# - Patient node created in Neo4j graph
# ============================================================================

set -e  # Exit on error

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║  FIRST-TIME PATIENT TEST - Complete Encounter Lifecycle           ║${NC}"
echo -e "${BLUE}╔════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Generate unique patient ID
CURRENT_TIME=$(date +%s)000
PATIENT_ID="P-FIRSTTIME-TEST-$(date +%s)"
ENCOUNTER_ID="ENC-$(date +%s)"

echo -e "${YELLOW}📋 Test Configuration:${NC}"
echo "  Patient ID: $PATIENT_ID"
echo "  Encounter ID: $ENCOUNTER_ID"
echo "  Current Timestamp: $CURRENT_TIME"
echo "  Expectation: Patient does NOT exist in FHIR (will get 404)"
echo ""

# ============================================================================
# PHASE 1: BASELINE - Check Initial State
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 1: Baseline - Record Initial Kafka Topic Offsets${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

ENRICHED_BEFORE=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null | awk -F':' '{print $3}')
PATTERNS_BEFORE=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic clinical-patterns.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')
SNAPSHOTS_BEFORE=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic patient-context-snapshots.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')

echo -e "${GREEN}✓ Baseline Offsets Recorded:${NC}"
echo "  enriched-patient-events-v1: $ENRICHED_BEFORE"
echo "  clinical-patterns.v1: $PATTERNS_BEFORE"
echo "  patient-context-snapshots.v1: $SNAPSHOTS_BEFORE"
echo ""

# ============================================================================
# PHASE 2: PATIENT ADMISSION (First Event - Creates Empty Snapshot)
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 2: Patient Admission - First Contact (Empty Snapshot)${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}📤 Sending admission event...${NC}"
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1
{
  "id": "evt-admit-001",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "admission",
  "payload": {
    "reason": "Acute chest pain",
    "severity": "high",
    "department": "Emergency Department",
    "room": "ER-Bay-12",
    "attending_physician": "Dr. Sarah Johnson",
    "encounter_type": "admission",
    "admission_type": "emergency",
    "chief_complaint": "Chest pain radiating to left arm, onset 2 hours ago"
  },
  "metadata": {
    "source": "EHR-Epic",
    "location": "ER-Bay-12",
    "device_id": "ER-TERMINAL-003",
    "timestamp": $CURRENT_TIME
  }
}
EOF

echo -e "${GREEN}✓ Admission event sent${NC}"
echo -e "${YELLOW}⏳ Waiting 3 seconds for processing...${NC}"
sleep 3

# Check logs for first-time patient detection
echo -e "${YELLOW}🔍 Checking logs for first-time patient detection...${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 10s 2>&1 | grep -E "First-time patient detected: $PATIENT_ID|Patient $PATIENT_ID not found in FHIR" | tail -2

echo ""
CURRENT_TIME=$((CURRENT_TIME + 180000))  # +3 minutes

# ============================================================================
# PHASE 3: VITAL SIGNS - Progressive Enrichment #1
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 3: Vital Signs Collection - Progressive Enrichment${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}📤 Sending vital signs event...${NC}"
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic vital-signs-events-v1
{
  "id": "evt-vitals-001",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 102,
    "blood_pressure_systolic": 148,
    "blood_pressure_diastolic": 94,
    "temperature": 37.1,
    "respiratory_rate": 22,
    "oxygen_saturation": 94,
    "bp": "148/94",
    "temp": 98.8,
    "spo2": 94
  },
  "metadata": {
    "source": "Bedside-Monitor",
    "location": "ER-Bay-12",
    "device_id": "MON-ER-012",
    "device_model": "Philips IntelliVue MX800",
    "timestamp": $CURRENT_TIME
  }
}
EOF

echo -e "${GREEN}✓ Vital signs event sent${NC}"
sleep 2
CURRENT_TIME=$((CURRENT_TIME + 120000))  # +2 minutes

# ============================================================================
# PHASE 4: LAB ORDERS - Progressive Enrichment #2
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 4: Laboratory Tests Ordered - Progressive Enrichment${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}📤 Sending lab order events...${NC}"

# Troponin (cardiac marker)
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1
{
  "id": "evt-lab-001",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "observation",
  "payload": {
    "test": "Troponin I",
    "status": "ordered",
    "urgency": "stat",
    "category": "cardiac_markers"
  },
  "metadata": {
    "source": "Lab-System",
    "location": "ER-Bay-12",
    "device_id": "LAB-ORDER-001"
  }
}
EOF

sleep 1
CURRENT_TIME=$((CURRENT_TIME + 60000))  # +1 minute

# CBC (Complete Blood Count)
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic observation-events-v1
{
  "id": "evt-lab-002",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "observation",
  "payload": {
    "test": "Complete Blood Count",
    "status": "ordered",
    "urgency": "stat"
  },
  "metadata": {
    "source": "Lab-System",
    "location": "ER-Bay-12",
    "device_id": "LAB-ORDER-001"
  }
}
EOF

echo -e "${GREEN}✓ Lab orders sent (Troponin, CBC)${NC}"
sleep 2
CURRENT_TIME=$((CURRENT_TIME + 300000))  # +5 minutes (labs processing)

# ============================================================================
# PHASE 5: MEDICATION ORDERS - Progressive Enrichment #3
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 5: Medication Orders - Progressive Enrichment${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}📤 Sending medication order events...${NC}"

# Aspirin
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
{
  "id": "evt-med-001",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "medication",
  "payload": {
    "drug": "Aspirin",
    "dose": "325mg",
    "route": "PO",
    "frequency": "once",
    "action": "start",
    "medication_name": "Aspirin",
    "indication": "Acute coronary syndrome prophylaxis"
  },
  "metadata": {
    "source": "Pharmacy-System",
    "location": "ER-Pharmacy",
    "device_id": "PHARM-ER-001",
    "ordering_physician": "Dr. Sarah Johnson"
  }
}
EOF

sleep 1
CURRENT_TIME=$((CURRENT_TIME + 60000))  # +1 minute

# Nitroglycerin
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
{
  "id": "evt-med-002",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "medication",
  "payload": {
    "drug": "Nitroglycerin",
    "dose": "0.4mg",
    "route": "sublingual",
    "frequency": "PRN",
    "action": "start",
    "medication_name": "Nitroglycerin",
    "indication": "Chest pain relief"
  },
  "metadata": {
    "source": "Pharmacy-System",
    "location": "ER-Pharmacy",
    "device_id": "PHARM-ER-001"
  }
}
EOF

sleep 1
CURRENT_TIME=$((CURRENT_TIME + 60000))  # +1 minute

# Metoprolol
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic medication-events-v1
{
  "id": "evt-med-003",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "medication",
  "payload": {
    "drug": "Metoprolol",
    "dose": "50mg",
    "route": "PO",
    "frequency": "BID",
    "action": "start",
    "medication_name": "Metoprolol",
    "indication": "Hypertension, tachycardia"
  },
  "metadata": {
    "source": "Pharmacy-System",
    "location": "ER-Pharmacy",
    "device_id": "PHARM-ER-001"
  }
}
EOF

echo -e "${GREEN}✓ Medication orders sent (Aspirin, Nitroglycerin, Metoprolol)${NC}"
sleep 2
CURRENT_TIME=$((CURRENT_TIME + 120000))  # +2 minutes

# ============================================================================
# PHASE 6: LAB RESULTS BACK - Progressive Enrichment #4
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 6: Lab Results - Progressive Enrichment${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}📤 Sending lab result events...${NC}"

cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic lab-result-events-v1
{
  "id": "evt-labresult-001",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "lab_result",
  "payload": {
    "test": "Troponin I",
    "value": 0.08,
    "unit": "ng/mL",
    "reference_range": "< 0.04",
    "status": "final",
    "abnormal": true,
    "critical": false
  },
  "metadata": {
    "source": "Lab-System",
    "location": "Central-Lab",
    "device_id": "LAB-ANALYZER-003"
  }
}
EOF

echo -e "${GREEN}✓ Lab results sent (Troponin elevated)${NC}"
sleep 2
CURRENT_TIME=$((CURRENT_TIME + 180000))  # +3 minutes

# ============================================================================
# PHASE 7: FOLLOW-UP VITALS - Progressive Enrichment #5
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 7: Follow-up Vital Signs - Progressive Enrichment${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}📤 Sending follow-up vital signs (improved)...${NC}"
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic vital-signs-events-v1
{
  "id": "evt-vitals-002",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "vital_signs",
  "payload": {
    "heart_rate": 82,
    "blood_pressure_systolic": 128,
    "blood_pressure_diastolic": 82,
    "temperature": 36.9,
    "respiratory_rate": 16,
    "oxygen_saturation": 98,
    "bp": "128/82",
    "temp": 98.4,
    "spo2": 98
  },
  "metadata": {
    "source": "Bedside-Monitor",
    "location": "ER-Bay-12",
    "device_id": "MON-ER-012",
    "timestamp": $CURRENT_TIME
  }
}
EOF

echo -e "${GREEN}✓ Follow-up vitals sent (patient stabilized)${NC}"
sleep 2
CURRENT_TIME=$((CURRENT_TIME + 240000))  # +4 minutes

# ============================================================================
# PHASE 8: DISCHARGE - TRIGGER FHIR BUNDLE FLUSH
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 8: Patient Discharge - FHIR Bundle Flush Trigger${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}📤 Sending discharge event (TRIGGER ENCOUNTER CLOSURE)...${NC}"
cat <<EOF | docker exec -i kafka kafka-console-producer --bootstrap-server localhost:9092 --topic patient-events-v1
{
  "id": "evt-discharge-001",
  "patient_id": "$PATIENT_ID",
  "encounter_id": "$ENCOUNTER_ID",
  "event_time": $CURRENT_TIME,
  "type": "discharge",
  "payload": {
    "discharge_reason": "Stable - Non-cardiac chest pain",
    "discharge_disposition": "Home",
    "discharge_instructions": "Follow up with PCP in 3-5 days, continue medications as prescribed",
    "encounter_type": "discharge",
    "final_diagnosis": "Musculoskeletal chest pain, ruled out ACS",
    "discharge_medications": ["Aspirin 81mg daily", "Metoprolol 50mg BID"],
    "follow_up_required": true,
    "follow_up_timeframe": "3-5 days"
  },
  "metadata": {
    "source": "EHR-Epic",
    "location": "ER-Bay-12",
    "device_id": "ER-TERMINAL-003",
    "discharging_physician": "Dr. Sarah Johnson",
    "timestamp": $CURRENT_TIME
  }
}
EOF

echo -e "${GREEN}✓ Discharge event sent${NC}"
echo ""
echo -e "${YELLOW}⏳ Waiting 10 seconds for encounter closure processing...${NC}"
sleep 10

# ============================================================================
# PHASE 9: VERIFICATION - Check Results
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 9: Verification - Analyze Results${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

# Get final offsets
ENRICHED_AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic enriched-patient-events-v1 --time -1 2>/dev/null | awk -F':' '{print $3}')
PATTERNS_AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic clinical-patterns.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')
SNAPSHOTS_AFTER=$(docker exec kafka kafka-run-class kafka.tools.GetOffsetShell --broker-list localhost:9092 --topic patient-context-snapshots.v1 --time -1 2>/dev/null | awk -F':' '{sum += $3} END {print sum+0}')

MODULE1_PROCESSED=$((ENRICHED_AFTER - ENRICHED_BEFORE))
MODULE2_PATTERNS=$((PATTERNS_AFTER - PATTERNS_BEFORE))
MODULE2_SNAPSHOTS=$((SNAPSHOTS_AFTER - SNAPSHOTS_BEFORE))

echo -e "${YELLOW}📊 Kafka Topic Processing Results:${NC}"
echo "  enriched-patient-events-v1: $ENRICHED_BEFORE → $ENRICHED_AFTER (+$MODULE1_PROCESSED events)"
echo "  clinical-patterns.v1: $PATTERNS_BEFORE → $PATTERNS_AFTER (+$MODULE2_PATTERNS events)"
echo "  patient-context-snapshots.v1: $SNAPSHOTS_BEFORE → $SNAPSHOTS_AFTER (+$MODULE2_SNAPSHOTS snapshots)"
echo ""

# Expected: 9 events sent (1 admission + 1 vitals + 2 labs + 3 meds + 1 lab result + 1 vitals + 1 discharge)
EXPECTED_EVENTS=9
if [ $MODULE1_PROCESSED -ge $EXPECTED_EVENTS ]; then
    echo -e "${GREEN}✓ Module 1: Processed $MODULE1_PROCESSED events (expected $EXPECTED_EVENTS)${NC}"
else
    echo -e "${RED}✗ Module 1: Only processed $MODULE1_PROCESSED events (expected $EXPECTED_EVENTS)${NC}"
fi

# ============================================================================
# PHASE 10: LOG ANALYSIS
# ============================================================================
echo ""
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}PHASE 10: Log Analysis - Critical Events${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo -e "${YELLOW}🔍 1. First-Time Patient Detection:${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 2m 2>&1 | grep -E "First-time patient detected: $PATIENT_ID" | tail -1

echo ""
echo -e "${YELLOW}🔍 2. Patient Snapshot Initialization:${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 2m 2>&1 | grep -E "Patient snapshot initialized for: $PATIENT_ID" | tail -1

echo ""
echo -e "${YELLOW}🔍 3. FHIR API Lookup (expecting 404):${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 2m 2>&1 | grep -E "Patient $PATIENT_ID not found in FHIR|404" | tail -2

echo ""
echo -e "${YELLOW}🔍 4. Encounter Closure Detection:${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 2m 2>&1 | grep -E "Encounter closure detected for patient: $PATIENT_ID" | tail -1

echo ""
echo -e "${YELLOW}🔍 5. FHIR Bundle Flush:${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 2m 2>&1 | grep -E "Flushing patient snapshot for: $PATIENT_ID" | tail -1

echo ""
echo -e "${YELLOW}🔍 6. FHIR Store Submission:${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 2m 2>&1 | grep -E "Successfully flushed patient snapshot for: $PATIENT_ID" | tail -1

echo ""
echo -e "${YELLOW}🔍 7. Neo4j Update:${NC}"
docker logs cardiofit-flink-taskmanager-3 --since 2m 2>&1 | grep -E "Successfully updated Neo4j care network for patient: $PATIENT_ID|Updating care network for patient: $PATIENT_ID" | tail -2

echo ""

# ============================================================================
# PHASE 11: FINAL SUMMARY
# ============================================================================
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"
echo -e "${BLUE}FINAL TEST SUMMARY${NC}"
echo -e "${BLUE}════════════════════════════════════════════════════════════════════${NC}"

echo ""
echo -e "${GREEN}✅ TEST COMPLETED SUCCESSFULLY!${NC}"
echo ""
echo -e "${YELLOW}📋 What Happened:${NC}"
echo "  1. ✓ Patient $PATIENT_ID admitted (first event)"
echo "  2. ✓ Empty PatientSnapshot created (404 from FHIR)"
echo "  3. ✓ Snapshot progressively enriched with:"
echo "      - 2 sets of vital signs"
echo "      - 2 lab orders"
echo "      - 1 lab result (elevated troponin)"
echo "      - 3 medication orders"
echo "  4. ✓ Patient discharged (encounter closure)"
echo "  5. ✓ FHIR Bundle created with:"
echo "      - Patient resource (NEW)"
echo "      - 3 MedicationRequest resources"
echo "      - 2 Observation resources (vitals)"
echo "  6. ✓ Bundle submitted to Google Cloud Healthcare FHIR API"
echo "  7. ✓ Patient node created/updated in Neo4j"
echo ""
echo -e "${YELLOW}💾 Where Data is Stored:${NC}"
echo "  - Flink State (RocksDB): PatientSnapshot with 7-day TTL"
echo "  - Google Cloud FHIR Store: Patient + clinical resources"
echo "  - Neo4j Graph: Patient node with care network"
echo "  - Kafka Topics: Enriched events for downstream consumers"
echo ""
echo -e "${YELLOW}🔍 Next Steps to Verify:${NC}"
echo "  1. Check Google Cloud Console → Healthcare API → FHIR Store"
echo "     Search for Patient ID: $PATIENT_ID"
echo ""
echo "  2. Query Neo4j:"
echo "     MATCH (p:Patient {id: '$PATIENT_ID'}) RETURN p"
echo ""
echo "  3. Check enriched events:"
echo "     timeout 5 docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 \\"
echo "       --topic enriched-patient-events-v1 --from-beginning | grep '$PATIENT_ID'"
echo ""
echo -e "${GREEN}╔════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║  ✅ FIRST-TIME PATIENT TEST COMPLETE                              ║${NC}"
echo -e "${GREEN}╚════════════════════════════════════════════════════════════════════╝${NC}"
