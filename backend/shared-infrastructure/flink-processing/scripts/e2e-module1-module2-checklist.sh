#!/usr/bin/env bash
# =============================================================================
# E2E Deployment Verification Checklist — Module 1/1b + Module 2
# =============================================================================
#
# Prerequisites:
#   1. Kafka running (docker-compose.hpi-lite.yml or Confluent Cloud)
#   2. Module 1 Flink job running (flink-module1-ingestion consumer group)
#   3. Module 1b Flink job running (module1b-ingestion-canonicalizer consumer group)
#   4. Module 2 Flink job running (module2-enhanced-consumer consumer group)
#
# Usage:
#   KAFKA_BOOTSTRAP=localhost:9092 ./e2e-module1-module2-checklist.sh [test_number]
#
#   Run all tests:   ./e2e-module1-module2-checklist.sh
#   Run test 3 only: ./e2e-module1-module2-checklist.sh 3
#
# Each test produces a message, then consumes from the expected output topic
# with a timeout. Verdict: PASS if expected fields are found, FAIL otherwise.
# =============================================================================

set -euo pipefail

KAFKA_BOOTSTRAP="${KAFKA_BOOTSTRAP:-localhost:9092}"
CONSUME_TIMEOUT="${CONSUME_TIMEOUT:-30}"  # seconds to wait for output
PATIENT_ID="e2e-test-$(date +%s)"

# Colors
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

# Kafka CLI tool detection
KAFKA_BIN=""
if command -v kafka-console-producer &>/dev/null; then
    KAFKA_BIN="native"
elif docker exec cardiofit-kafka-lite kafka-topics --version &>/dev/null 2>&1; then
    KAFKA_BIN="docker"
else
    echo -e "${RED}ERROR: No Kafka CLI found. Install kafka-tools or use Docker.${NC}"
    echo "  Option 1: brew install kafka"
    echo "  Option 2: Ensure cardiofit-kafka-lite container is running"
    exit 1
fi

produce() {
    local topic="$1"
    local message="$2"
    if [[ "$KAFKA_BIN" == "native" ]]; then
        echo "$message" | kafka-console-producer --bootstrap-server "$KAFKA_BOOTSTRAP" --topic "$topic" 2>/dev/null
    else
        echo "$message" | docker exec -i cardiofit-kafka-lite kafka-console-producer \
            --bootstrap-server kafka-lite:29092 --topic "$topic" 2>/dev/null
    fi
}

consume_and_check() {
    local topic="$1"
    local pattern="$2"
    local timeout="${3:-$CONSUME_TIMEOUT}"
    local tmpfile
    tmpfile=$(mktemp)

    if [[ "$KAFKA_BIN" == "native" ]]; then
        timeout "$timeout" kafka-console-consumer \
            --bootstrap-server "$KAFKA_BOOTSTRAP" \
            --topic "$topic" \
            --from-beginning \
            --group "e2e-check-$(date +%s%N)" \
            --max-messages 50 \
            2>/dev/null > "$tmpfile" || true
    else
        timeout "$timeout" docker exec cardiofit-kafka-lite kafka-console-consumer \
            --bootstrap-server kafka-lite:29092 \
            --topic "$topic" \
            --from-beginning \
            --group "e2e-check-$(date +%s%N)" \
            --max-messages 50 \
            2>/dev/null > "$tmpfile" || true
    fi

    if grep -q "$pattern" "$tmpfile" 2>/dev/null; then
        rm -f "$tmpfile"
        return 0
    else
        cat "$tmpfile"
        rm -f "$tmpfile"
        return 1
    fi
}

header() {
    echo ""
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${CYAN}  TEST $1: $2${NC}"
    echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
}

verdict() {
    if [[ "$1" == "PASS" ]]; then
        echo -e "  ${GREEN}✅ PASS${NC} — $2"
    else
        echo -e "  ${RED}❌ FAIL${NC} — $2"
    fi
}

SELECTED_TEST="${1:-all}"
PASS_COUNT=0
FAIL_COUNT=0

# ─────────────────────────────────────────────────────────────
# TEST 1: Happy Path — Vital Sign through Module 1 → Module 2
# ─────────────────────────────────────────────────────────────
run_test_1() {
    header 1 "Happy Path: vital-signs-events-v1 → enriched-patient-events-v1 → clinical-patterns.v1"

    local event_json
    event_json=$(cat <<ENDJSON
{"patientId":"${PATIENT_ID}-t1","eventType":"VITAL_SIGN","eventTime":$(date +%s000),"sourceSystem":"legacy-ehr","payload":{"heartRate":85,"systolicBP":130,"diastolicBP":80,"oxygenSaturation":96,"respiratoryRate":16,"temperature":37.1}}
ENDJSON
)

    echo "  Producing vital sign event to vital-signs-events-v1..."
    produce "vital-signs-events-v1" "$event_json"

    echo "  Checking enriched-patient-events-v1 for sourceSystem=legacy-ehr..."
    if consume_and_check "enriched-patient-events-v1" "${PATIENT_ID}-t1"; then
        verdict "PASS" "Event arrived at enriched-patient-events-v1"
    else
        verdict "FAIL" "Event NOT found at enriched-patient-events-v1 within ${CONSUME_TIMEOUT}s"
        ((FAIL_COUNT++)) || true
        return
    fi

    echo "  Checking clinical-patterns.v1 for enrichment metadata..."
    if consume_and_check "clinical-patterns.v1" "enrichment_status"; then
        verdict "PASS" "Event arrived at clinical-patterns.v1 with enrichment metadata"
        ((PASS_COUNT++)) || true
    else
        verdict "FAIL" "Event NOT found at clinical-patterns.v1 within ${CONSUME_TIMEOUT}s"
        ((FAIL_COUNT++)) || true
    fi
}

# ─────────────────────────────────────────────────────────────
# TEST 2: Dual-Path Merge — Same event via Module 1 + Module 1b
# ─────────────────────────────────────────────────────────────
run_test_2() {
    header 2 "Dual-Path Merge: legacy EHR + ingestion outbox → enriched-patient-events-v1"

    local ts
    ts=$(date +%s000)

    # Module 1 path (legacy EHR)
    local ehr_json
    ehr_json=$(cat <<ENDJSON
{"patientId":"${PATIENT_ID}-t2","eventType":"VITAL_SIGN","eventTime":${ts},"sourceSystem":"legacy-ehr","payload":{"heartRate":72,"systolicBP":120,"diastolicBP":75,"oxygenSaturation":98}}
ENDJSON
)

    # Module 1b path (ingestion outbox envelope)
    local outbox_json
    outbox_json=$(cat <<ENDJSON
{"envelope_id":"env-${PATIENT_ID}-t2","event_type":"ingestion.vitals","aggregate_type":"patient","aggregate_id":"${PATIENT_ID}-t2","payload":{"patientId":"${PATIENT_ID}-t2","eventType":"VITAL_SIGN","eventTime":${ts},"sourceSystem":"ingestion-service","payload":{"heartRate":72,"systolicBP":120,"diastolicBP":75,"oxygenSaturation":98}},"created_at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)"}
ENDJSON
)

    echo "  Producing to vital-signs-events-v1 (Module 1 path)..."
    produce "vital-signs-events-v1" "$ehr_json"

    echo "  Producing to ingestion.vitals (Module 1b path)..."
    produce "ingestion.vitals" "$outbox_json"

    echo "  Checking enriched-patient-events-v1 for BOTH sourceSystem values..."
    local tmpfile
    tmpfile=$(mktemp)

    if [[ "$KAFKA_BIN" == "native" ]]; then
        timeout "$CONSUME_TIMEOUT" kafka-console-consumer \
            --bootstrap-server "$KAFKA_BOOTSTRAP" \
            --topic "enriched-patient-events-v1" \
            --from-beginning \
            --group "e2e-dual-$(date +%s%N)" \
            --max-messages 100 \
            2>/dev/null > "$tmpfile" || true
    else
        timeout "$CONSUME_TIMEOUT" docker exec cardiofit-kafka-lite kafka-console-consumer \
            --bootstrap-server kafka-lite:29092 \
            --topic "enriched-patient-events-v1" \
            --from-beginning \
            --group "e2e-dual-$(date +%s%N)" \
            --max-messages 100 \
            2>/dev/null > "$tmpfile" || true
    fi

    local found_ehr=false
    local found_ingestion=false
    if grep -q "legacy-ehr" "$tmpfile" 2>/dev/null; then found_ehr=true; fi
    if grep -q "ingestion-service" "$tmpfile" 2>/dev/null; then found_ingestion=true; fi
    rm -f "$tmpfile"

    if $found_ehr && $found_ingestion; then
        verdict "PASS" "Both sourceSystem values found (legacy-ehr + ingestion-service)"
        echo -e "  ${YELLOW}⚠️  WARNING: Deduplication strategy needed before production if both paths active${NC}"
        ((PASS_COUNT++)) || true
    elif $found_ehr; then
        verdict "FAIL" "Only legacy-ehr found — Module 1b path may not be running"
        ((FAIL_COUNT++)) || true
    elif $found_ingestion; then
        verdict "FAIL" "Only ingestion-service found — Module 1 path may not be running"
        ((FAIL_COUNT++)) || true
    else
        verdict "FAIL" "Neither sourceSystem found — both paths may be down"
        ((FAIL_COUNT++)) || true
    fi
}

# ─────────────────────────────────────────────────────────────
# TEST 3: Crash Resilience — Malformed JSON does NOT crash job
# ─────────────────────────────────────────────────────────────
run_test_3() {
    header 3 "Crash Resilience: malformed JSON → logged, not crash-looped"

    echo "  Producing malformed JSON to vital-signs-events-v1..."
    produce "vital-signs-events-v1" '{"this is not valid json'

    echo "  Waiting 5s for job stability..."
    sleep 5

    # Send a valid event AFTER the malformed one
    local valid_json
    valid_json=$(cat <<ENDJSON
{"patientId":"${PATIENT_ID}-t3-after","eventType":"VITAL_SIGN","eventTime":$(date +%s000),"sourceSystem":"legacy-ehr","payload":{"heartRate":70,"systolicBP":118,"diastolicBP":76,"oxygenSaturation":97}}
ENDJSON
)

    echo "  Producing valid event after malformed one..."
    produce "vital-signs-events-v1" "$valid_json"

    echo "  Checking that valid event was processed (job didn't crash)..."
    if consume_and_check "enriched-patient-events-v1" "${PATIENT_ID}-t3-after"; then
        verdict "PASS" "Job survived malformed JSON — valid event processed normally"
        ((PASS_COUNT++)) || true
    else
        verdict "FAIL" "Valid event NOT found — job may have crash-looped on malformed JSON"
        ((FAIL_COUNT++)) || true
    fi
}

# ─────────────────────────────────────────────────────────────
# TEST 4: DLQ Flow — null patientId → dlq.processing-errors.v1
# ─────────────────────────────────────────────────────────────
run_test_4() {
    header 4 "DLQ Flow: null patientId → dlq.processing-errors.v1"

    # Outbox envelope with null patientId in the inner payload
    local dlq_json
    dlq_json=$(cat <<ENDJSON
{"envelope_id":"env-dlq-${PATIENT_ID}","event_type":"ingestion.vitals","aggregate_type":"patient","aggregate_id":"null-patient","payload":{"patientId":null,"eventType":"VITAL_SIGN","eventTime":$(date +%s000),"sourceSystem":"ingestion-service","payload":{"heartRate":80}},"created_at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)"}
ENDJSON
)

    echo "  Producing event with null patientId to ingestion.vitals..."
    produce "ingestion.vitals" "$dlq_json"

    echo "  Checking dlq.processing-errors.v1 for the rejected event..."
    if consume_and_check "dlq.processing-errors.v1" "null-patient"; then
        verdict "PASS" "Event routed to DLQ with context for reprocessing"
        ((PASS_COUNT++)) || true
    else
        verdict "FAIL" "Event NOT found in DLQ — may have been silently dropped"
        echo -e "  ${YELLOW}NOTE: Also check enriched-patient-events-v1 — if found there, DLQ routing is broken${NC}"
        ((FAIL_COUNT++)) || true
    fi
}

# ─────────────────────────────────────────────────────────────
# TEST 5: TIER_1_CGM Propagation — data_tier survives Module 2
# ─────────────────────────────────────────────────────────────
run_test_5() {
    header 5 "TIER_1_CGM Propagation: data_tier preserved through Module 1b → Module 2"

    local cgm_json
    cgm_json=$(cat <<ENDJSON
{"envelope_id":"env-cgm-${PATIENT_ID}","event_type":"ingestion.cgm-raw","aggregate_type":"patient","aggregate_id":"${PATIENT_ID}-t5","payload":{"patientId":"${PATIENT_ID}-t5","eventType":"VITAL_SIGN","eventTime":$(date +%s000),"sourceSystem":"ingestion-service","payload":{"heartRate":75,"systolicBP":122,"diastolicBP":78,"oxygenSaturation":97,"data_tier":"TIER_1_CGM","glucose_mgdl":105,"device_type":"dexcom_g7"}},"created_at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)"}
ENDJSON
)

    echo "  Producing CGM event with data_tier=TIER_1_CGM to ingestion.cgm-raw..."
    produce "ingestion.cgm-raw" "$cgm_json"

    echo "  Checking clinical-patterns.v1 for data_tier propagation..."
    echo "  Expected paths: dataTier (first-class field) AND patientState.latestVitals.data_tier"

    local tmpfile
    tmpfile=$(mktemp)

    if [[ "$KAFKA_BIN" == "native" ]]; then
        timeout "$CONSUME_TIMEOUT" kafka-console-consumer \
            --bootstrap-server "$KAFKA_BOOTSTRAP" \
            --topic "clinical-patterns.v1" \
            --from-beginning \
            --group "e2e-cgm-$(date +%s%N)" \
            --max-messages 50 \
            2>/dev/null > "$tmpfile" || true
    else
        timeout "$CONSUME_TIMEOUT" docker exec cardiofit-kafka-lite kafka-console-consumer \
            --bootstrap-server kafka-lite:29092 \
            --topic "clinical-patterns.v1" \
            --from-beginning \
            --group "e2e-cgm-$(date +%s%N)" \
            --max-messages 50 \
            2>/dev/null > "$tmpfile" || true
    fi

    local found_first_class=false
    local found_nested=false

    # Check for first-class dataTier field (Task 16)
    if grep -q '"dataTier":"TIER_1_CGM"' "$tmpfile" 2>/dev/null; then
        found_first_class=true
    fi
    # Check for nested path in latestVitals
    if grep -q '"data_tier":"TIER_1_CGM"' "$tmpfile" 2>/dev/null; then
        found_nested=true
    fi
    rm -f "$tmpfile"

    if $found_first_class; then
        verdict "PASS" "dataTier=TIER_1_CGM found as first-class field on EnrichedPatientContext"
        if $found_nested; then
            echo -e "  ${GREEN}  Also present in latestVitals (expected dual presence)${NC}"
        fi
        ((PASS_COUNT++)) || true
    elif $found_nested; then
        verdict "PASS" "data_tier=TIER_1_CGM found in latestVitals (nested path)"
        echo -e "  ${YELLOW}⚠️  First-class dataTier field NOT found — Task 16 may not be deployed${NC}"
        ((PASS_COUNT++)) || true
    else
        verdict "FAIL" "data_tier=TIER_1_CGM NOT found in clinical-patterns.v1 output"
        echo -e "  ${RED}  MHRI in Module 3 will silently fall back to TIER_3_SMBG defaults${NC}"
        ((FAIL_COUNT++)) || true
    fi
}

# ─────────────────────────────────────────────────────────────
# TEST 6: Timestamp Clamping — Future timestamp → SANITIZED
# ─────────────────────────────────────────────────────────────
run_test_6() {
    header 6 "Timestamp Clamping: event 2h in future → clamped to now+1h, validationStatus=SANITIZED"

    # 2 hours in the future (in milliseconds)
    local future_ts
    future_ts=$(( $(date +%s) * 1000 + 7200000 ))

    # 1 hour in the future (expected clamp ceiling)
    local clamp_ceiling
    clamp_ceiling=$(( $(date +%s) * 1000 + 3600000 ))

    local future_json
    future_json=$(cat <<ENDJSON
{"envelope_id":"env-future-${PATIENT_ID}","event_type":"ingestion.vitals","aggregate_type":"patient","aggregate_id":"${PATIENT_ID}-t6","payload":{"patientId":"${PATIENT_ID}-t6","eventType":"VITAL_SIGN","eventTime":${future_ts},"sourceSystem":"ingestion-service","payload":{"heartRate":82,"systolicBP":125,"diastolicBP":80,"oxygenSaturation":96}},"created_at":"$(date -u +%Y-%m-%dT%H:%M:%SZ)"}
ENDJSON
)

    echo "  Producing event with timestamp 2h in the future to ingestion.vitals..."
    echo "  Future timestamp: ${future_ts}"
    echo "  Expected clamp ceiling: ~${clamp_ceiling}"
    produce "ingestion.vitals" "$future_json"

    echo "  Checking enriched-patient-events-v1 for SANITIZED status..."
    local tmpfile
    tmpfile=$(mktemp)

    if [[ "$KAFKA_BIN" == "native" ]]; then
        timeout "$CONSUME_TIMEOUT" kafka-console-consumer \
            --bootstrap-server "$KAFKA_BOOTSTRAP" \
            --topic "enriched-patient-events-v1" \
            --from-beginning \
            --group "e2e-future-$(date +%s%N)" \
            --max-messages 50 \
            2>/dev/null > "$tmpfile" || true
    else
        timeout "$CONSUME_TIMEOUT" docker exec cardiofit-kafka-lite kafka-console-consumer \
            --bootstrap-server kafka-lite:29092 \
            --topic "enriched-patient-events-v1" \
            --from-beginning \
            --group "e2e-future-$(date +%s%N)" \
            --max-messages 50 \
            2>/dev/null > "$tmpfile" || true
    fi

    local found_patient=false
    local found_sanitized=false
    local found_clamped=false

    if grep -q "${PATIENT_ID}-t6" "$tmpfile" 2>/dev/null; then
        found_patient=true
    fi
    if grep -q "SANITIZED" "$tmpfile" 2>/dev/null; then
        found_sanitized=true
    fi
    # Check that the eventTime in output is NOT the original future timestamp
    if grep "${PATIENT_ID}-t6" "$tmpfile" 2>/dev/null | grep -qv "\"eventTime\":${future_ts}"; then
        found_clamped=true
    fi
    rm -f "$tmpfile"

    if $found_patient && $found_sanitized; then
        verdict "PASS" "Event arrived with validationStatus=SANITIZED"
        if $found_clamped; then
            echo -e "  ${GREEN}  Timestamp was clamped (not original ${future_ts})${NC}"
        else
            echo -e "  ${YELLOW}⚠️  Could not confirm timestamp clamping — check eventTime manually${NC}"
        fi
        ((PASS_COUNT++)) || true
    elif $found_patient; then
        verdict "FAIL" "Event arrived but validationStatus is NOT SANITIZED"
        echo -e "  ${RED}  Timestamp clamping may not be active in Module 1b${NC}"
        ((FAIL_COUNT++)) || true
    else
        verdict "FAIL" "Event NOT found — Module 1b may not be processing ingestion.vitals"
        ((FAIL_COUNT++)) || true
    fi
}

# ─────────────────────────────────────────────────────────────
# Main
# ─────────────────────────────────────────────────────────────
echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║  E2E Deployment Verification — Module 1/1b + Module 2      ║${NC}"
echo -e "${CYAN}║  Kafka: ${KAFKA_BOOTSTRAP}                                  ║${NC}"
echo -e "${CYAN}║  Patient ID prefix: ${PATIENT_ID}                ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"

if [[ "$SELECTED_TEST" == "all" ]]; then
    run_test_1
    run_test_2
    run_test_3
    run_test_4
    run_test_5
    run_test_6
else
    case "$SELECTED_TEST" in
        1) run_test_1 ;;
        2) run_test_2 ;;
        3) run_test_3 ;;
        4) run_test_4 ;;
        5) run_test_5 ;;
        6) run_test_6 ;;
        *) echo -e "${RED}Unknown test number: $SELECTED_TEST (valid: 1-6)${NC}"; exit 1 ;;
    esac
fi

echo ""
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
echo -e "  Results: ${GREEN}${PASS_COUNT} passed${NC}, ${RED}${FAIL_COUNT} failed${NC}"
echo -e "${CYAN}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

if [[ $FAIL_COUNT -gt 0 ]]; then
    exit 1
fi
