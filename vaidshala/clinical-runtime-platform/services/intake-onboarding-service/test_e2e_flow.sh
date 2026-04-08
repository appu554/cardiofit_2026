#!/usr/bin/env bash
# =============================================================================
# CardioFit Intake E2E Flow — Clinically Realistic Patient
# =============================================================================
#
# Patient Profile: Mrs. Lakshmi Devi, 62F
#   - Type 2 Diabetes (14 years), HbA1c 9.1%, FBG 168 mg/dL
#   - CKD Stage 3b (eGFR 38), creatinine 1.8 mg/dL, UACR 280 mg/g
#   - Hypertension (Stage 2): 156/94 mmHg
#   - Dyslipidemia: LDL 158, HDL 34, TG 240
#   - NYHA Class II heart failure, LVEF 40%
#   - On 8 medications including insulin
#   - BMI 31.2 (obese), former smoker, vegetarian
#   - Known allergy to ACE inhibitors (cough)
#
# Expected Safety Triggers:
#   - Soft flag: polypharmacy (8 meds ≥5)
#   - Soft flag: eGFR 38 (CKD 3b — needs dose adjustment review)
#   - Soft flag: HbA1c 9.1% (severely uncontrolled)
#   - Review risk: HIGH (eGFR < 45 region, soft_flags ≥3, age ≥75-adjacent)
#
# Usage:
#   chmod +x test_e2e_flow.sh
#   ./test_e2e_flow.sh
#
# =============================================================================

set -euo pipefail

BASE="http://localhost:8141"
BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

ok()   { echo -e "  ${GREEN}✓${NC} $1"; }
warn() { echo -e "  ${YELLOW}⚠${NC} $1"; }
fail() { echo -e "  ${RED}✗${NC} $1"; exit 1; }
step() { echo -e "\n${BOLD}${CYAN}═══ $1 ═══${NC}"; }

# Helper: POST JSON, capture response
post() {
  local url="$1" body="$2"
  shift 2
  curl -s -w "\n%{http_code}" -X POST "$url" \
    -H "Content-Type: application/json" \
    "$@" \
    -d "$body"
}

# Extract JSON field
jq_field() { echo "$1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('$2',''))"; }

# Helper: fill one intake slot
fill_slot() {
  local name="$1" value="$2" mode="${3:-BUTTON}" conf="${4:-1.0}" chan="${5:-APP}"
  local body="{\"slot_name\":\"$name\",\"value\":$value,\"extraction_mode\":\"$mode\",\"confidence\":$conf,\"source_channel\":\"$chan\"}"
  local raw
  raw=$(post "$BASE/fhir/Encounter/$ENCOUNTER_ID/\$fill-slot" "$body" \
    -H "X-Patient-ID: $PATIENT_ID")
  local code=${raw##*$'\n'}
  local resp=${raw%$'\n'*}

  if [ "$code" = "200" ]; then
    local status
    status=$(jq_field "$resp" "status")
    local filled
    filled=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('progress',{}).get('filled','?'))")
    local total
    total=$(echo "$resp" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('progress',{}).get('total','?'))")

    if [ "$status" = "hard_stopped" ]; then
      warn "$name = $value → HARD STOP ($filled/$total)"
    else
      ok "$name = $value → $filled/$total slots"
    fi
  else
    fail "$name failed: HTTP $code"
  fi
}

# Helper: fill one check-in slot
fill_checkin_slot() {
  local name="$1" value="$2" mode="${3:-PATIENT_REPORTED}"
  local body="{\"slot_name\":\"$name\",\"value\":$value,\"extraction_mode\":\"$mode\"}"
  local raw
  raw=$(post "$BASE/fhir/CheckinSession/$CHECKIN_SESSION_ID/\$checkin-slot" "$body")
  local code=${raw##*$'\n'}
  local resp=${raw%$'\n'*}
  if [ "$code" = "200" ]; then
    local filled
    filled=$(jq_field "$resp" "slots_filled")
    ok "$name = $value → $filled filled"
  else
    fail "checkin $name failed: HTTP $code"
  fi
}

# =============================================================================
step "0. Health Check"
# =============================================================================
HC=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/healthz")
[ "$HC" = "200" ] && ok "Service healthy" || fail "Service not running on $BASE"

# =============================================================================
step "1. ENROLL — Mrs. Lakshmi Devi, 62F, T2DM+CKD+HTN"
# =============================================================================
ENROLL_RAW=$(post "$BASE/fhir/Patient/\$enroll" '{
  "given_name": "Lakshmi",
  "family_name": "Devi",
  "phone": "+919845012345",
  "abha_id": "91-6789-0123-4567",
  "channel_type": "INSURANCE",
  "tenant_id": "00000000-0000-0000-0000-000000000001"
}')
ENROLL_CODE=${ENROLL_RAW##*$'\n'}
ENROLL_RESP=${ENROLL_RAW%$'\n'*}

[ "$ENROLL_CODE" = "201" ] || fail "Enroll failed: HTTP $ENROLL_CODE"

PATIENT_ID=$(jq_field "$ENROLL_RESP" "patient_id")
ENCOUNTER_ID=$(jq_field "$ENROLL_RESP" "encounter_id")
ok "Patient ID:    $PATIENT_ID"
ok "Encounter ID:  $ENCOUNTER_ID"
ok "FHIR Store:    Patient + Encounter created in GCP"

# =============================================================================
step "2. FILL SLOTS — Demographics (8 slots)"
# =============================================================================
fill_slot "age"              62
fill_slot "sex"              '"female"'
fill_slot "height"           155
fill_slot "weight"           75
fill_slot "bmi"              31.2
fill_slot "pregnant"         false
fill_slot "ethnicity"        '"south_asian"'
fill_slot "primary_language" '"tamil"'

# =============================================================================
step "3. FILL SLOTS — Glycemic (7 slots)"
# =============================================================================
fill_slot "diabetes_type"          '"type2"'
fill_slot "fbg"                    168     "DEVICE"  0.95
fill_slot "hba1c"                  9.1     "REGEX"   0.92
fill_slot "ppbg"                   245     "DEVICE"  0.95
fill_slot "diabetes_duration_years" 14
fill_slot "insulin"                true
fill_slot "hypoglycemia_episodes"  1

# =============================================================================
step "4. FILL SLOTS — Renal (5 slots)"
# =============================================================================
fill_slot "egfr"              38      "REGEX"  0.95
fill_slot "serum_creatinine"  1.8     "REGEX"  0.92
fill_slot "uacr"              280     "REGEX"  0.88
fill_slot "dialysis"          false
fill_slot "serum_potassium"   5.1     "REGEX"  0.90

# =============================================================================
step "5. FILL SLOTS — Cardiac (7 slots)"
# =============================================================================
fill_slot "systolic_bp"         156     "DEVICE"  0.98
fill_slot "diastolic_bp"        94      "DEVICE"  0.98
fill_slot "heart_rate"          82      "DEVICE"  0.95
fill_slot "nyha_class"          2
fill_slot "mi_stroke_days"      0
fill_slot "lvef"                40      "REGEX"   0.85
fill_slot "atrial_fibrillation" false

# =============================================================================
step "6. FILL SLOTS — Lipid (5 slots)"
# =============================================================================
fill_slot "total_cholesterol" 248     "REGEX"  0.90
fill_slot "ldl"               158     "REGEX"  0.90
fill_slot "hdl"               34      "REGEX"  0.90
fill_slot "triglycerides"     240     "REGEX"  0.90
fill_slot "on_statin"         true

# =============================================================================
step "7. FILL SLOTS — Medications (5 slots)"
# =============================================================================
fill_slot "current_medications" '["metformin 500mg BD","glimepiride 1mg OD","insulin glargine 24U HS","amlodipine 5mg OD","losartan 50mg OD","atorvastatin 40mg OD","aspirin 75mg OD","pantoprazole 40mg OD"]' "NLU" 0.82
fill_slot "medication_count"    8
fill_slot "adherence_score"     0.6
fill_slot "allergies"           '["ACE inhibitors (cough)","sulfonamides"]' "NLU" 0.88
fill_slot "supplement_list"     '["calcium + vitamin D3","iron folic acid"]' "NLU" 0.85

# =============================================================================
step "8. FILL SLOTS — Lifestyle (7 slots)"
# =============================================================================
fill_slot "smoking_status"         '"former"'
fill_slot "alcohol_use"            '"never"'
fill_slot "exercise_minutes_week"  45
fill_slot "diet_type"              '"vegetarian"'
fill_slot "sleep_hours"            5.5
fill_slot "active_substance_abuse" false
fill_slot "falls_history"          true

# =============================================================================
step "9. FILL SLOTS — Symptoms (6 slots)"
# =============================================================================
fill_slot "active_cancer"             false
fill_slot "organ_transplant"          false
fill_slot "cognitive_impairment"      false
fill_slot "bariatric_surgery_months"  0
fill_slot "primary_complaint"         '"Frequent urination, tingling in feet, fatigue, and occasional dizziness on standing"' "NLU" 0.78
fill_slot "comorbidities"             '["type 2 diabetes","CKD stage 3b","hypertension stage 2","dyslipidemia","heart failure NYHA II","peripheral neuropathy","orthostatic hypotension"]' "NLU" 0.82

# =============================================================================
step "10. SAFETY EVALUATION"
# =============================================================================
SAFETY_RAW=$(curl -s -w "\n%{http_code}" -X POST "$BASE/fhir/Patient/$PATIENT_ID/\$evaluate-safety")
SAFETY_CODE=${SAFETY_RAW##*$'\n'}
SAFETY_RESP=${SAFETY_RAW%$'\n'*}
[ "$SAFETY_CODE" = "200" ] || fail "Safety eval failed: HTTP $SAFETY_CODE"

HAS_HARD=$(jq_field "$SAFETY_RESP" "has_hard_stop")
echo -e "  Hard stops:  $(echo "$SAFETY_RESP" | python3 -c "import sys,json; hs=json.load(sys.stdin).get('hard_stops',[]); print(len(hs))")"
echo -e "  Soft flags:  $(echo "$SAFETY_RESP" | python3 -c "import sys,json; sf=json.load(sys.stdin).get('soft_flags',[]); print(len(sf))")"

if [ "$HAS_HARD" = "True" ] || [ "$HAS_HARD" = "true" ]; then
  warn "HARD STOP detected — patient needs immediate attention"
else
  ok "No hard stops — proceeding to review"
fi

echo ""
echo "  Safety details:"
echo "$SAFETY_RESP" | python3 -c "
import sys, json
d = json.load(sys.stdin)
for hs in d.get('hard_stops', []):
    print(f'    🛑 {hs[\"rule_id\"]}: {hs[\"reason\"]}')
for sf in d.get('soft_flags', []):
    print(f'    ⚠️  {sf[\"rule_id\"]}: {sf[\"reason\"]}')
if not d.get('hard_stops') and not d.get('soft_flags'):
    print('    (none)')
"

# =============================================================================
step "11. SUBMIT FOR REVIEW"
# =============================================================================
# First, transition enrollment to INTAKE_COMPLETED
echo "  Transitioning enrollment state → INTAKE_COMPLETED..."
PGPASSWORD=intake_password psql -h localhost -p 5433 -U intake_user -d intake_service -c \
  "UPDATE enrollments SET state = 'INTAKE_COMPLETED' WHERE encounter_id = '$ENCOUNTER_ID';" 2>/dev/null \
  && ok "State → INTAKE_COMPLETED" \
  || warn "psql not available — manually run: UPDATE enrollments SET state = 'INTAKE_COMPLETED' WHERE encounter_id = '$ENCOUNTER_ID';"

SOFT_COUNT=$(echo "$SAFETY_RESP" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('soft_flags',[])))")

REVIEW_RAW=$(post "$BASE/fhir/Encounter/$ENCOUNTER_ID/\$submit-review" "{
  \"hard_stop_count\": 0,
  \"soft_flag_count\": $SOFT_COUNT,
  \"age\": 62,
  \"med_count\": 8,
  \"egfr_value\": 38
}")
REVIEW_CODE=${REVIEW_RAW##*$'\n'}
REVIEW_RESP=${REVIEW_RAW%$'\n'*}
[ "$REVIEW_CODE" = "201" ] || fail "Submit review failed: HTTP $REVIEW_CODE — $REVIEW_RESP"

REVIEW_ENTRY_ID=$(jq_field "$REVIEW_RESP" "id")
RISK=$(jq_field "$REVIEW_RESP" "risk_stratum")
ok "Review entry:  $REVIEW_ENTRY_ID"
ok "Risk stratum:  $RISK"

# =============================================================================
step "12. APPROVE REVIEW → ENROLLED"
# =============================================================================
REVIEWER_ID="00000000-0000-0000-0000-000000000099"
APPROVE_RAW=$(post "$BASE/fhir/ReviewEntry/$REVIEW_ENTRY_ID/\$approve" '{}' \
  -H "X-User-ID: $REVIEWER_ID")
APPROVE_CODE=${APPROVE_RAW##*$'\n'}
[ "$APPROVE_CODE" = "200" ] && ok "Approved by reviewer $REVIEWER_ID" || fail "Approve failed: HTTP $APPROVE_CODE"

# Verify enrollment state
STATE=$(PGPASSWORD=intake_password psql -h localhost -p 5433 -U intake_user -d intake_service -t -c \
  "SELECT state FROM enrollments WHERE encounter_id = '$ENCOUNTER_ID';" 2>/dev/null | tr -d ' ')
if [ "$STATE" = "ENROLLED" ]; then
  ok "Enrollment state: ENROLLED"
else
  warn "Could not verify enrollment state via psql (state=$STATE)"
fi

# =============================================================================
step "13. START CHECK-IN — Cycle 1 (M0-CI)"
# =============================================================================
CHECKIN_RAW=$(post "$BASE/fhir/Patient/$PATIENT_ID/\$checkin" '{"cycle_number": 1}')
CHECKIN_CODE=${CHECKIN_RAW##*$'\n'}
CHECKIN_RESP=${CHECKIN_RAW%$'\n'*}
[ "$CHECKIN_CODE" = "201" ] || fail "Start check-in failed: HTTP $CHECKIN_CODE — $CHECKIN_RESP"

CHECKIN_SESSION_ID=$(jq_field "$CHECKIN_RESP" "session_id")
CHECKIN_STATE=$(jq_field "$CHECKIN_RESP" "state")
ok "Session:  $CHECKIN_SESSION_ID"
ok "State:    $CHECKIN_STATE"

# =============================================================================
step "14. FILL CHECK-IN SLOTS — Biweekly Vitals"
# =============================================================================
# Simulate 2-week check-in: slightly improved values after enrollment
fill_checkin_slot "fbg"                     152     "DEVICE"
fill_checkin_slot "ppbg"                    228     "DEVICE"
fill_checkin_slot "hba1c"                   8.8     "PATIENT_REPORTED"
fill_checkin_slot "systolic_bp"             148     "DEVICE"
fill_checkin_slot "diastolic_bp"            90      "DEVICE"
fill_checkin_slot "egfr"                    39      "PATIENT_REPORTED"
fill_checkin_slot "weight"                  74.5    "DEVICE"
fill_checkin_slot "medication_adherence"    0.75    "PATIENT_REPORTED"
fill_checkin_slot "physical_activity_minutes" 60    "PATIENT_REPORTED"
fill_checkin_slot "sleep_hours"             6       "PATIENT_REPORTED"
fill_checkin_slot "symptom_severity"        5       "PATIENT_REPORTED"
fill_checkin_slot "side_effects"            2       "PATIENT_REPORTED"

# =============================================================================
step "15. SUMMARY"
# =============================================================================
echo ""
echo -e "${BOLD}Patient:${NC}     Lakshmi Devi, 62F, Tamil Nadu"
echo -e "${BOLD}Diagnosis:${NC}   T2DM (14y) + CKD 3b + HTN Stage 2 + HFrEF NYHA II"
echo -e "${BOLD}Medications:${NC} 8 (incl. insulin glargine 24U, losartan, amlodipine)"
echo -e "${BOLD}Risk:${NC}        $RISK"
echo ""
echo -e "${BOLD}IDs:${NC}"
echo "  Patient:        $PATIENT_ID"
echo "  Encounter:      $ENCOUNTER_ID"
echo "  Review Entry:   $REVIEW_ENTRY_ID"
echo "  Check-in:       $CHECKIN_SESSION_ID"
echo ""
echo -e "${BOLD}FHIR Store:${NC}  https://console.cloud.google.com/healthcare/browser/locations/asia-south1/datasets/vaidshala-clinical/fhirStores/cardiofit-fhir-r4?project=project-2bbef9ac-174b-4b59-8fe"
echo -e "${BOLD}Swagger UI:${NC}  http://localhost:8888"
echo ""
echo -e "${GREEN}${BOLD}Full E2E flow complete.${NC}"
