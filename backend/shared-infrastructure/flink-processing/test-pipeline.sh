#!/bin/bash

# Flink EHR Intelligence Pipeline - Test Script
# Sends properly formatted test events and verifies processing

set -e

SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"
TEST_DATA_DIR="${SCRIPT_DIR}/test-data"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Functions
print_header() {
    echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║         Flink Pipeline Test - Event Processing Verification   ║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo
}

print_step() {
    echo -e "${BLUE}[$(date +%H:%M:%S)]${NC} $1"
}

print_success() {
    echo -e "${GREEN}✓${NC} $1"
}

print_error() {
    echo -e "${RED}✗${NC} $1"
}

print_warning() {
    echo -e "${YELLOW}⚠${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    print_step "Checking prerequisites..."

    # Check if Flink is running
    if ! curl -sf http://localhost:8081/overview > /dev/null 2>&1; then
        print_error "Flink cluster is not running at localhost:8081"
        echo "Please start the cluster first: cd ${SCRIPT_DIR} && docker-compose up -d"
        exit 1
    fi
    print_success "Flink cluster is running"

    # Check if Kafka is accessible
    if ! docker exec kafka kafka-broker-api-versions --bootstrap-server localhost:9092 > /dev/null 2>&1; then
        print_error "Kafka is not accessible"
        exit 1
    fi
    print_success "Kafka is accessible"

    # Check if job is running
    RUNNING_JOBS=$(curl -s http://localhost:8081/jobs | jq -r '.jobs[] | select(.status=="RUNNING") | .id' | wc -l)
    if [ "$RUNNING_JOBS" -eq 0 ]; then
        print_warning "No Flink jobs are currently running"
        echo "Submit a job first using: ./submit-job.sh ingestion-only development"
        read -p "Do you want to continue anyway? (y/N) " -n 1 -r
        echo
        if [[ ! $REPLY =~ ^[Yy]$ ]]; then
            exit 1
        fi
    else
        print_success "Found $RUNNING_JOBS running job(s)"
    fi
}

# Send individual test event
send_event() {
    local TOPIC=$1
    local EVENT=$2
    local EVENT_NAME=$3

    print_step "Sending: $EVENT_NAME"

    # Send event to Kafka
    echo "$EVENT" | docker exec -i kafka kafka-console-producer \
        --broker-list localhost:9092 \
        --topic "$TOPIC" 2>&1 | grep -v "WARN" || true

    print_success "Event sent to topic: $TOPIC"
}

# Test 1: Send vital signs event
test_vital_signs() {
    print_step "Test 1: Vital Signs - Normal Adult"

    EVENT=$(cat <<'EOF'
{"id":"evt-vital-001","source":"icu-monitor-bed-5","type":"vital-signs","patient_id":"PT-001","encounter_id":"ENC-2025-001","event_time":1727740800000,"received_time":1727740800000,"payload":{"heartRate":72,"bloodPressure":"118/78","temperature":98.4,"respiratoryRate":16,"oxygenSaturation":98},"metadata":{"device_id":"MON-ICU-005","location":"ICU-BED-5","facility":"CardioFit-Hospital-Main"},"correlation_id":"corr-vital-001","version":"1.0"}
EOF
    )

    send_event "patient-events-v1" "$EVENT" "Normal vital signs"
    sleep 2
}

# Test 2: Send deteriorating patient event
test_deteriorating_patient() {
    print_step "Test 2: Vital Signs - Deteriorating Patient (Sepsis Risk)"

    EVENT=$(cat <<'EOF'
{"id":"evt-vital-002","source":"icu-monitor-bed-3","type":"vital-signs","patient_id":"PT-002","encounter_id":"ENC-2025-002","event_time":1727741400000,"received_time":1727741400000,"payload":{"heartRate":125,"bloodPressure":"88/55","temperature":101.5,"respiratoryRate":28,"oxygenSaturation":89},"metadata":{"device_id":"MON-ICU-003","location":"ICU-BED-3","alert_level":"high"},"correlation_id":"corr-vital-002","version":"1.0"}
EOF
    )

    send_event "patient-events-v1" "$EVENT" "Deteriorating patient vitals"
    sleep 2
}

# Test 3: Send medication event
test_medication() {
    print_step "Test 3: Medication Administration - Antibiotics"

    EVENT=$(cat <<'EOF'
{"id":"evt-med-001","source":"pharmacy-system","type":"medication-administration","patient_id":"PT-002","encounter_id":"ENC-2025-002","event_time":1727741700000,"received_time":1727741700000,"payload":{"medicationName":"Vancomycin","dose":1000,"doseUnit":"mg","route":"IV","frequency":"Q12H","administeredBy":"RN-Johnson"},"metadata":{"order_id":"ORD-MED-5521","pharmacy_verified":"true","allergies_checked":"true"},"correlation_id":"corr-med-001","version":"1.0"}
EOF
    )

    send_event "medication-events-v1" "$EVENT" "Antibiotic administration"
    sleep 2
}

# Test 4: Send lab results
test_lab_results() {
    print_step "Test 4: Lab Results - Elevated White Blood Count"

    EVENT=$(cat <<'EOF'
{"id":"evt-lab-001","source":"lab-system","type":"lab-result","patient_id":"PT-002","encounter_id":"ENC-2025-002","event_time":1727742000000,"received_time":1727742000000,"payload":{"testName":"Complete Blood Count","results":{"wbc":18.5,"rbc":4.2,"hemoglobin":13.1,"hematocrit":39.2,"platelets":245},"units":{"wbc":"K/uL","rbc":"M/uL","hemoglobin":"g/dL","hematocrit":"%","platelets":"K/uL"},"criticalValues":["wbc"]},"metadata":{"lab_order_id":"LAB-2025-8821","performed_by":"LAB-TECH-042","verified_by":"MD-Pathologist-12"},"correlation_id":"corr-lab-001","version":"1.0"}
EOF
    )

    send_event "observation-events-v1" "$EVENT" "Lab results with critical values"
    sleep 2
}

# Test 5: Send device data
test_device_data() {
    print_step "Test 5: Device Data - Ventilator Settings"

    EVENT=$(cat <<'EOF'
{"id":"evt-device-001","source":"ventilator-vent-12","type":"ventilator-data","patient_id":"PT-003","encounter_id":"ENC-2025-003","event_time":1727742300000,"received_time":1727742300000,"payload":{"mode":"AC/VC","tidalVolume":450,"respiratoryRate":14,"peep":5,"fio2":40,"minuteVolume":6.3},"metadata":{"device_serial":"VENT-SN-9942","calibration_date":"2025-09-15","alarm_status":"none"},"correlation_id":"corr-device-001","version":"1.0"}
EOF
    )

    send_event "validated-device-data-v1" "$EVENT" "Ventilator data"
    sleep 2
}

# Test 6: Sepsis pattern detection (multiple events)
test_sepsis_pattern() {
    print_step "Test 6: Sepsis Pattern Detection (Sequential Events)"

    print_step "  → Sending vital signs (deteriorating)"
    EVENT1=$(cat <<'EOF'
{"id":"evt-sepsis-001","source":"icu-monitor-bed-3","type":"vital-signs","patient_id":"PT-002","encounter_id":"ENC-2025-002","event_time":1727744000000,"received_time":1727744000000,"payload":{"heartRate":128,"bloodPressure":"85/52","temperature":102.1,"respiratoryRate":30,"oxygenSaturation":87},"metadata":{"device_id":"MON-ICU-003","sequence":"1_of_3"},"correlation_id":"corr-sepsis-pattern","version":"1.0"}
EOF
    )
    send_event "vital-signs-events-v1" "$EVENT1" "Sepsis pattern 1/3"
    sleep 3

    print_step "  → Sending lab results (elevated lactate)"
    EVENT2=$(cat <<'EOF'
{"id":"evt-sepsis-002","source":"lab-system","type":"lab-result","patient_id":"PT-002","encounter_id":"ENC-2025-002","event_time":1727744300000,"received_time":1727744300000,"payload":{"testName":"Sepsis Panel","results":{"lactate":4.5,"procalcitonin":12.5,"wbc":22.0},"units":{"lactate":"mmol/L","procalcitonin":"ng/mL","wbc":"K/uL"},"criticalValues":["lactate","procalcitonin"]},"metadata":{"lab_order_id":"LAB-2025-8823","stat_priority":"true","sequence":"2_of_3"},"correlation_id":"corr-sepsis-pattern","version":"1.0"}
EOF
    )
    send_event "lab-result-events-v1" "$EVENT2" "Sepsis pattern 2/3"
    sleep 3

    print_step "  → Sending clinical assessment (SOFA score)"
    EVENT3=$(cat <<'EOF'
{"id":"evt-sepsis-003","source":"physician-assessment","type":"clinical-assessment","patient_id":"PT-002","encounter_id":"ENC-2025-002","event_time":1727744600000,"received_time":1727744600000,"payload":{"assessment":"Suspected septic shock","sofa_score":8,"qsofa_score":3,"mental_status":"altered","skin_perfusion":"mottled"},"metadata":{"physician_id":"MD-Critical-Care-05","assessment_type":"bedside","sequence":"3_of_3"},"correlation_id":"corr-sepsis-pattern","version":"1.0"}
EOF
    )
    send_event "observation-events-v1" "$EVENT3" "Sepsis pattern 3/3"
    sleep 2

    print_success "Sepsis pattern sequence completed"
}

# Verify output
verify_output() {
    print_step "Verifying pipeline output..."

    echo "Checking enriched-patient-events-v1 topic for processed events..."
    OUTPUT=$(timeout 10 docker exec kafka kafka-console-consumer \
        --bootstrap-server localhost:9092 \
        --topic enriched-patient-events-v1 \
        --from-beginning \
        --max-messages 5 2>/dev/null || echo "")

    if [ -n "$OUTPUT" ]; then
        EVENT_COUNT=$(echo "$OUTPUT" | wc -l)
        print_success "Found $EVENT_COUNT processed event(s) in output topic"
        echo
        echo "Sample output:"
        echo "$OUTPUT" | head -3 | jq -C '.' 2>/dev/null || echo "$OUTPUT" | head -3
    else
        print_warning "No events found in output topic yet"
        echo "This may be normal if:"
        echo "  - Job is still processing (check Flink UI)"
        echo "  - Events failed validation (check DLQ topic)"
        echo "  - Pipeline configuration needs adjustment"
    fi
}

# Check job health
check_job_health() {
    print_step "Checking Flink job health..."

    JOB_ID=$(curl -s http://localhost:8081/jobs | jq -r '.jobs[] | select(.status=="RUNNING") | .id' | head -1)

    if [ -z "$JOB_ID" ]; then
        print_error "No running jobs found"
        return 1
    fi

    JOB_STATE=$(curl -s "http://localhost:8081/jobs/$JOB_ID" | jq -r '.state')
    print_success "Job $JOB_ID is $JOB_STATE"

    # Check for exceptions
    EXCEPTIONS=$(curl -s "http://localhost:8081/jobs/$JOB_ID/exceptions" | jq -r '.["all-exceptions"] | length')
    if [ "$EXCEPTIONS" -gt 0 ]; then
        print_warning "Job has $EXCEPTIONS exception(s)"
        echo "View exceptions at: http://localhost:8081/#/job/$JOB_ID/exceptions"
    else
        print_success "No exceptions detected"
    fi
}

# Main execution
main() {
    print_header

    check_prerequisites
    echo

    print_step "Starting test event sequence..."
    echo

    test_vital_signs
    test_deteriorating_patient
    test_medication
    test_lab_results
    test_device_data
    test_sepsis_pattern

    echo
    print_step "All events sent successfully!"
    echo

    print_step "Waiting 10 seconds for processing..."
    sleep 10
    echo

    verify_output
    echo

    check_job_health
    echo

    echo -e "${BLUE}╔═══════════════════════════════════════════════════════════════╗${NC}"
    echo -e "${BLUE}║                     Test Complete                             ║${NC}"
    echo -e "${BLUE}╚═══════════════════════════════════════════════════════════════╝${NC}"
    echo

    echo "Next steps:"
    echo "  1. View Flink Web UI: http://localhost:8081"
    echo "  2. Check Grafana dashboards: http://localhost:3001"
    echo "  3. Monitor job metrics and throughput"
    echo "  4. Review DLQ topic for any failed events:"
    echo "     docker exec kafka kafka-console-consumer --bootstrap-server localhost:9092 --topic dlq.processing-errors.v1 --from-beginning"
    echo
}

# Run main function
main "$@"
