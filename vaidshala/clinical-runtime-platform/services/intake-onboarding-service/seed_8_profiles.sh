#!/usr/bin/env bash
# =============================================================================
# CardioFit Intake — Seed 8 Clinically Diverse Patient Profiles
# =============================================================================
#
# Creates 8 FHIR-compliant patient profiles via the intake service fill-slot API.
# Each profile covers all 50 intake slots: demographics, glycemic, renal, cardiac,
# lipid, medications, lifestyle, symptoms/conditions.
#
# Profiles:
#   1. Rajesh Kumar      — 58M, T2DM 10y, CKD 3a, HTN Stage 1, 6 meds
#   2. Priya Nair        — 34F, GDM (pregnant), no CKD, normal BP, 2 meds
#   3. Anand Sharma      — 72M, T2DM 25y, CKD 4, HTN Stage 2, HF NYHA III, 10 meds
#   4. Meena Sundaram    — 45F, T1DM 20y, CKD 2, normal BP, 4 meds
#   5. Vikram Patel      — 55M, T2DM 5y, no CKD, HTN Stage 1, metabolic syndrome, 5 meds
#   6. Sunita Reddy      — 68F, T2DM 18y, CKD 3b, HTN Stage 2, prev MI, 9 meds
#   7. Arjun Menon       — 28M, T1DM 15y, CKD 1, normal BP, 3 meds, athletic
#   8. Kavitha Iyer      — 50F, T2DM 8y, CKD 3a, HTN Stage 1, 7 meds
#
# Usage:
#   chmod +x seed_8_profiles.sh
#   ./seed_8_profiles.sh
#
# =============================================================================

set -euo pipefail

BASE="${INTAKE_BASE_URL:-http://localhost:8141}"
TENANT="00000000-0000-0000-0000-000000000001"

BOLD='\033[1m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
CYAN='\033[0;36m'
NC='\033[0m'

ok()   { echo -e "  ${GREEN}✓${NC} $1"; }
warn() { echo -e "  ${YELLOW}⚠${NC} $1"; }
fail() { echo -e "  ${RED}✗${NC} $1"; }
step() { echo -e "\n${BOLD}${CYAN}═══ $1 ═══${NC}"; }

PATIENT_ID=""
ENCOUNTER_ID=""
TOTAL_PATIENTS=0
FAILED_PATIENTS=0

# ─── HTTP helpers ────────────────────────────────────────────────────────────

post() {
  local url="$1" body="$2"
  shift 2
  curl -s -w "\n%{http_code}" -X POST "$url" \
    -H "Content-Type: application/json" \
    "$@" \
    -d "$body"
}

jq_field() {
  echo "$1" | python3 -c "import sys,json; d=json.load(sys.stdin); print(d.get('$2',''))" 2>/dev/null
}

# ─── Create patient + encounter + enroll (3-step flow) ───────────────────────

enroll_patient() {
  local given="$1" family="$2" phone="$3" abha="$4" channel="$5"
  local email="${6:-}" age="${7:-}" gender="${8:-}" ethnicity="${9:-}" lang="${10:-}"

  # Step 1: Create Patient
  local create_body
  create_body=$(python3 -c "
import json
d = {
  'given_name': '$given',
  'family_name': '$family',
  'phone': '$phone',
  'abha_id': '$abha'
}
if '$email': d['email'] = '$email'
if '$age': d['age'] = int('$age')
if '$gender': d['gender'] = '$gender'
if '$ethnicity': d['ethnicity'] = '$ethnicity'
if '$lang': d['primary_language'] = '$lang'
print(json.dumps(d))
")

  local raw code resp
  raw=$(post "$BASE/fhir/Patient" "$create_body")
  code=${raw##*$'\n'}
  resp=${raw%$'\n'*}

  if [ "$code" != "201" ] && [ "$code" != "200" ]; then
    fail "Create patient failed (HTTP $code): $resp"
    return 1
  fi
  PATIENT_ID=$(jq_field "$resp" "patient_id")
  ok "Patient:   $PATIENT_ID"

  # Step 2: Create Encounter
  raw=$(post "$BASE/fhir/Patient/$PATIENT_ID/Encounter" '{"type":"intake"}')
  code=${raw##*$'\n'}
  resp=${raw%$'\n'*}

  if [ "$code" != "201" ] && [ "$code" != "200" ]; then
    fail "Create encounter failed (HTTP $code): $resp"
    return 1
  fi
  ENCOUNTER_ID=$(jq_field "$resp" "encounter_id")
  ok "Encounter: $ENCOUNTER_ID"

  # Step 3: Enroll
  local enroll_body="{\"encounter_id\":\"$ENCOUNTER_ID\",\"channel_type\":\"$channel\",\"tenant_id\":\"$TENANT\"}"
  raw=$(post "$BASE/fhir/Patient/$PATIENT_ID/\$enroll" "$enroll_body")
  code=${raw##*$'\n'}
  resp=${raw%$'\n'*}

  if [ "$code" != "201" ] && [ "$code" != "200" ]; then
    fail "Enroll failed (HTTP $code): $resp"
    return 1
  fi
  ok "Enrolled:  $channel"
  return 0
}

# ─── Fill one slot ───────────────────────────────────────────────────────────

fill_slot() {
  local name="$1" value="$2" mode="${3:-BUTTON}" conf="${4:-1.0}" chan="${5:-APP}"
  local body="{\"slot_name\":\"$name\",\"value\":$value,\"extraction_mode\":\"$mode\",\"confidence\":$conf,\"source_channel\":\"$chan\"}"

  local raw code resp
  raw=$(post "$BASE/fhir/Encounter/$ENCOUNTER_ID/\$fill-slot" "$body" \
    -H "X-Patient-ID: $PATIENT_ID")
  code=${raw##*$'\n'}
  resp=${raw%$'\n'*}

  if [ "$code" = "200" ]; then
    local status filled total
    status=$(jq_field "$resp" "status")
    filled=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('progress',{}).get('filled','?'))" 2>/dev/null)
    total=$(echo "$resp" | python3 -c "import sys,json; print(json.load(sys.stdin).get('progress',{}).get('total','?'))" 2>/dev/null)
    if [ "$status" = "hard_stopped" ]; then
      warn "$name → HARD STOP ($filled/$total)"
    else
      ok "$name=$value ($filled/$total)"
    fi
  else
    fail "$name failed (HTTP $code)"
  fi
}

# ─── Safety evaluation ───────────────────────────────────────────────────────

evaluate_safety() {
  local raw code resp
  raw=$(curl -s -w "\n%{http_code}" -X POST "$BASE/fhir/Patient/$PATIENT_ID/\$evaluate-safety")
  code=${raw##*$'\n'}
  resp=${raw%$'\n'*}
  if [ "$code" = "200" ]; then
    local hs sf
    hs=$(echo "$resp" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('hard_stops',[])))" 2>/dev/null)
    sf=$(echo "$resp" | python3 -c "import sys,json; print(len(json.load(sys.stdin).get('soft_flags',[])))" 2>/dev/null)
    ok "Safety: ${hs} hard stops, ${sf} soft flags"
  else
    warn "Safety eval HTTP $code"
  fi
}

# ─── Fill all 50 slots for a profile ────────────────────────────────────────

fill_all_slots() {
  # Arguments: all slot values passed as positional params via associative-array-style
  # We call this per-profile below with explicit fill_slot calls for clarity.
  :
}

# =============================================================================
#  HEALTH CHECK
# =============================================================================

step "Health Check"
HC=$(curl -s -o /dev/null -w "%{http_code}" "$BASE/healthz" 2>/dev/null || echo "000")
if [ "$HC" = "200" ]; then
  ok "Intake service healthy at $BASE"
else
  fail "Intake service not running at $BASE (HTTP $HC)"
  echo -e "\n  Start the service first:  cd intake-onboarding-service && go run ./cmd/intake"
  exit 1
fi

# =============================================================================
#  PROFILE 1: Rajesh Kumar — 58M, T2DM 10y, CKD 3a, HTN Stage 1
# =============================================================================

step "Profile 1/8: Rajesh Kumar — 58M, T2DM 10y, CKD 3a, HTN Stage 1"
if enroll_patient "Rajesh" "Kumar" "+919845100001" "91-1001-2001-3001" "CORPORATE" \
    "rajesh.kumar@email.com" "58" "male" "south_asian" "hi"; then

  # Demographics
  fill_slot "age"              58
  fill_slot "sex"              '"male"'
  fill_slot "height"           172
  fill_slot "weight"           84
  fill_slot "pregnant"         false
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"hi"'

  # Glycemic
  fill_slot "diabetes_type"          '"type2"'
  fill_slot "fbg"                    142     "DEVICE" 0.95
  fill_slot "hba1c"                  7.8     "REGEX"  0.92
  fill_slot "ppbg"                   198     "DEVICE" 0.93
  fill_slot "diabetes_duration_years" 10
  fill_slot "insulin"                false
  fill_slot "hypoglycemia_episodes"  0

  # Renal
  fill_slot "egfr"              52      "REGEX" 0.94
  fill_slot "serum_creatinine"  1.4     "REGEX" 0.92
  fill_slot "uacr"              85      "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   4.6     "REGEX" 0.90

  # Cardiac
  fill_slot "systolic_bp"         142     "DEVICE" 0.98
  fill_slot "diastolic_bp"        88      "DEVICE" 0.98
  fill_slot "heart_rate"          76      "DEVICE" 0.95
  fill_slot "nyha_class"          1
  fill_slot "mi_stroke_days"      0
  fill_slot "lvef"                55      "REGEX"  0.85
  fill_slot "atrial_fibrillation" false

  # Lipid
  fill_slot "total_cholesterol" 218     "REGEX" 0.90
  fill_slot "ldl"               138     "REGEX" 0.90
  fill_slot "hdl"               38      "REGEX" 0.90
  fill_slot "triglycerides"     195     "REGEX" 0.90
  fill_slot "on_statin"         true

  # Medications
  fill_slot "current_medications" '["metformin 1000mg BD","glimepiride 2mg OD","telmisartan 40mg OD","amlodipine 5mg OD","atorvastatin 20mg OD","aspirin 75mg OD"]' "NLU" 0.85
  fill_slot "medication_count"    6
  fill_slot "adherence_score"     0.75
  fill_slot "allergies"           '["none known"]' "NLU" 0.90
  fill_slot "supplement_list"     '["vitamin D3"]' "NLU" 0.85

  # Lifestyle
  fill_slot "smoking_status"         '"current"'
  fill_slot "alcohol_use"            '"moderate"'
  fill_slot "exercise_minutes_week"  30
  fill_slot "diet_type"              '"non_vegetarian"'
  fill_slot "sleep_hours"            6.5
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          false

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      false
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Increased thirst, occasional blurred vision, and mild ankle swelling"' "NLU" 0.80
  fill_slot "comorbidities"             '["type 2 diabetes","CKD stage 3a","hypertension stage 1","dyslipidemia","microalbuminuria"]' "NLU" 0.82

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 1 complete"
fi

# =============================================================================
#  PROFILE 2: Priya Nair — 34F, GDM (pregnant), no CKD, normal BP
# =============================================================================

step "Profile 2/8: Priya Nair — 34F, GDM (pregnant), no CKD, normal BP"
if enroll_patient "Priya" "Nair" "+919845100002" "91-1002-2002-3002" "INSURANCE" \
    "priya.nair@email.com" "34" "female" "south_asian" "ml"; then

  # Demographics
  fill_slot "age"              34
  fill_slot "sex"              '"female"'
  fill_slot "height"           160
  fill_slot "weight"           72
  fill_slot "pregnant"         true
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"ml"'

  # Glycemic — GDM, mild hyperglycemia
  fill_slot "diabetes_type"          '"gestational"'
  fill_slot "fbg"                    108     "DEVICE" 0.95
  fill_slot "hba1c"                  5.9     "REGEX"  0.94
  fill_slot "ppbg"                   158     "DEVICE" 0.93
  fill_slot "diabetes_duration_years" 0
  fill_slot "insulin"                false
  fill_slot "hypoglycemia_episodes"  0

  # Renal — normal
  fill_slot "egfr"              105     "REGEX" 0.94
  fill_slot "serum_creatinine"  0.7     "REGEX" 0.92
  fill_slot "uacr"              12      "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   4.2     "REGEX" 0.90

  # Cardiac — normal
  fill_slot "systolic_bp"         118     "DEVICE" 0.98
  fill_slot "diastolic_bp"        74      "DEVICE" 0.98
  fill_slot "heart_rate"          84      "DEVICE" 0.95
  fill_slot "nyha_class"          1
  fill_slot "mi_stroke_days"      0
  fill_slot "lvef"                62      "REGEX"  0.85
  fill_slot "atrial_fibrillation" false

  # Lipid — borderline
  fill_slot "total_cholesterol" 205     "REGEX" 0.90
  fill_slot "ldl"               120     "REGEX" 0.90
  fill_slot "hdl"               52      "REGEX" 0.90
  fill_slot "triglycerides"     165     "REGEX" 0.90
  fill_slot "on_statin"         false

  # Medications — minimal (pregnancy-safe only)
  fill_slot "current_medications" '["metformin 500mg BD","calcium carbonate 500mg OD"]' "NLU" 0.88
  fill_slot "medication_count"    2
  fill_slot "adherence_score"     0.90
  fill_slot "allergies"           '["penicillin (rash)"]' "NLU" 0.90
  fill_slot "supplement_list"     '["folic acid 5mg","iron folic acid","calcium + vitamin D3"]' "NLU" 0.90

  # Lifestyle
  fill_slot "smoking_status"         '"never"'
  fill_slot "alcohol_use"            '"never"'
  fill_slot "exercise_minutes_week"  90
  fill_slot "diet_type"              '"vegetarian"'
  fill_slot "sleep_hours"            7
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          false

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      false
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Fatigue, increased frequency of urination, occasional nausea"' "NLU" 0.80
  fill_slot "comorbidities"             '["gestational diabetes mellitus","iron deficiency anemia"]' "NLU" 0.85

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 2 complete"
fi

# =============================================================================
#  PROFILE 3: Anand Sharma — 72M, T2DM 25y, CKD 4, HTN Stage 2, HF NYHA III
# =============================================================================

step "Profile 3/8: Anand Sharma — 72M, T2DM 25y, CKD 4, HF NYHA III, AFib"
if enroll_patient "Anand" "Sharma" "+919845100003" "91-1003-2003-3003" "GOVERNMENT" \
    "anand.sharma@email.com" "72" "male" "south_asian" "hi"; then

  # Demographics
  fill_slot "age"              72
  fill_slot "sex"              '"male"'
  fill_slot "height"           168
  fill_slot "weight"           70
  fill_slot "pregnant"         false
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"hi"'

  # Glycemic — long-standing, on insulin, frequent hypos
  fill_slot "diabetes_type"          '"type2"'
  fill_slot "fbg"                    95      "DEVICE" 0.95
  fill_slot "hba1c"                  7.2     "REGEX"  0.90
  fill_slot "ppbg"                   165     "DEVICE" 0.93
  fill_slot "diabetes_duration_years" 25
  fill_slot "insulin"                true
  fill_slot "hypoglycemia_episodes"  4

  # Renal — CKD Stage 4
  fill_slot "egfr"              22      "REGEX" 0.94
  fill_slot "serum_creatinine"  2.8     "REGEX" 0.92
  fill_slot "uacr"              520     "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   5.4     "REGEX" 0.90

  # Cardiac — HF NYHA III, AFib, previous stroke
  fill_slot "systolic_bp"         162     "DEVICE" 0.98
  fill_slot "diastolic_bp"        78      "DEVICE" 0.98
  fill_slot "heart_rate"          92      "DEVICE" 0.95
  fill_slot "nyha_class"          3
  fill_slot "mi_stroke_days"      180
  fill_slot "lvef"                32      "REGEX"  0.85
  fill_slot "atrial_fibrillation" true

  # Lipid
  fill_slot "total_cholesterol" 195     "REGEX" 0.90
  fill_slot "ldl"               108     "REGEX" 0.90
  fill_slot "hdl"               32      "REGEX" 0.90
  fill_slot "triglycerides"     210     "REGEX" 0.90
  fill_slot "on_statin"         true

  # Medications — complex polypharmacy
  fill_slot "current_medications" '["insulin glargine 32U HS","insulin lispro 8U TID","losartan 50mg OD","furosemide 40mg BD","carvedilol 12.5mg BD","apixaban 2.5mg BD","atorvastatin 40mg OD","pantoprazole 40mg OD","erythropoietin 4000U weekly","calcium acetate 667mg TID"]' "NLU" 0.78
  fill_slot "medication_count"    10
  fill_slot "adherence_score"     0.55
  fill_slot "allergies"           '["metformin (GI intolerance at CKD 4)","iodinated contrast"]' "NLU" 0.85
  fill_slot "supplement_list"     '["vitamin D3 60000U weekly","iron sucrose IV monthly"]' "NLU" 0.80

  # Lifestyle
  fill_slot "smoking_status"         '"former"'
  fill_slot "alcohol_use"            '"never"'
  fill_slot "exercise_minutes_week"  15
  fill_slot "diet_type"              '"vegetarian"'
  fill_slot "sleep_hours"            5
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          true

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      true
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Breathlessness on mild exertion, swollen ankles, episodes of dizziness and near-syncope, nocturnal leg cramps"' "NLU" 0.75
  fill_slot "comorbidities"             '["type 2 diabetes","CKD stage 4","hypertension stage 2","heart failure NYHA III","atrial fibrillation","previous ischemic stroke","peripheral neuropathy","mild cognitive impairment","anemia of CKD"]' "NLU" 0.78

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 3 complete"
fi

# =============================================================================
#  PROFILE 4: Meena Sundaram — 45F, T1DM 20y, CKD 2, normal BP
# =============================================================================

step "Profile 4/8: Meena Sundaram — 45F, T1DM 20y, CKD 2, well-controlled"
if enroll_patient "Meena" "Sundaram" "+919845100004" "91-1004-2004-3004" "CORPORATE" \
    "meena.sundaram@email.com" "45" "female" "south_asian" "ta"; then

  # Demographics
  fill_slot "age"              45
  fill_slot "sex"              '"female"'
  fill_slot "height"           158
  fill_slot "weight"           58
  fill_slot "pregnant"         false
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"ta"'

  # Glycemic — T1DM, reasonably controlled, some hypos
  fill_slot "diabetes_type"          '"type1"'
  fill_slot "fbg"                    118     "DEVICE" 0.97
  fill_slot "hba1c"                  7.0     "REGEX"  0.94
  fill_slot "ppbg"                   155     "DEVICE" 0.95
  fill_slot "diabetes_duration_years" 20
  fill_slot "insulin"                true
  fill_slot "hypoglycemia_episodes"  2

  # Renal — CKD Stage 2 (early)
  fill_slot "egfr"              78      "REGEX" 0.94
  fill_slot "serum_creatinine"  0.9     "REGEX" 0.92
  fill_slot "uacr"              45      "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   4.3     "REGEX" 0.90

  # Cardiac — normal
  fill_slot "systolic_bp"         122     "DEVICE" 0.98
  fill_slot "diastolic_bp"        76      "DEVICE" 0.98
  fill_slot "heart_rate"          72      "DEVICE" 0.95
  fill_slot "nyha_class"          1
  fill_slot "mi_stroke_days"      0
  fill_slot "lvef"                60      "REGEX"  0.85
  fill_slot "atrial_fibrillation" false

  # Lipid — well-managed
  fill_slot "total_cholesterol" 185     "REGEX" 0.90
  fill_slot "ldl"               98      "REGEX" 0.90
  fill_slot "hdl"               55      "REGEX" 0.90
  fill_slot "triglycerides"     130     "REGEX" 0.90
  fill_slot "on_statin"         false

  # Medications — insulin pump + minimal oral
  fill_slot "current_medications" '["insulin aspart pump (basal 0.8U/hr)","insulin aspart bolus PRN","lisinopril 10mg OD","aspirin 75mg OD"]' "NLU" 0.88
  fill_slot "medication_count"    4
  fill_slot "adherence_score"     0.92
  fill_slot "allergies"           '["none known"]' "NLU" 0.95
  fill_slot "supplement_list"     '["vitamin D3","omega-3 fish oil"]' "NLU" 0.90

  # Lifestyle — active
  fill_slot "smoking_status"         '"never"'
  fill_slot "alcohol_use"            '"never"'
  fill_slot "exercise_minutes_week"  180
  fill_slot "diet_type"              '"vegetarian"'
  fill_slot "sleep_hours"            7.5
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          false

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      false
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Occasional hypoglycemic episodes during exercise, mild tingling in feet"' "NLU" 0.82
  fill_slot "comorbidities"             '["type 1 diabetes","CKD stage 2","early diabetic nephropathy","mild peripheral neuropathy"]' "NLU" 0.85

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 4 complete"
fi

# =============================================================================
#  PROFILE 5: Vikram Patel — 55M, T2DM 5y, no CKD, HTN Stage 1, obese
# =============================================================================

step "Profile 5/8: Vikram Patel — 55M, T2DM 5y, metabolic syndrome, obese"
if enroll_patient "Vikram" "Patel" "+919845100005" "91-1005-2005-3005" "CORPORATE" \
    "vikram.patel@email.com" "55" "male" "south_asian" "gu"; then

  # Demographics
  fill_slot "age"              55
  fill_slot "sex"              '"male"'
  fill_slot "height"           175
  fill_slot "weight"           102
  fill_slot "pregnant"         false
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"gu"'

  # Glycemic — newly-ish T2DM, poorly controlled
  fill_slot "diabetes_type"          '"type2"'
  fill_slot "fbg"                    165     "DEVICE" 0.95
  fill_slot "hba1c"                  8.6     "REGEX"  0.92
  fill_slot "ppbg"                   235     "DEVICE" 0.93
  fill_slot "diabetes_duration_years" 5
  fill_slot "insulin"                false
  fill_slot "hypoglycemia_episodes"  0

  # Renal — normal (no CKD)
  fill_slot "egfr"              92      "REGEX" 0.94
  fill_slot "serum_creatinine"  1.0     "REGEX" 0.92
  fill_slot "uacr"              18      "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   4.5     "REGEX" 0.90

  # Cardiac — HTN Stage 1
  fill_slot "systolic_bp"         148     "DEVICE" 0.98
  fill_slot "diastolic_bp"        92      "DEVICE" 0.98
  fill_slot "heart_rate"          88      "DEVICE" 0.95
  fill_slot "nyha_class"          1
  fill_slot "mi_stroke_days"      0
  fill_slot "lvef"                58      "REGEX"  0.85
  fill_slot "atrial_fibrillation" false

  # Lipid — classic metabolic syndrome dyslipidemia
  fill_slot "total_cholesterol" 265     "REGEX" 0.90
  fill_slot "ldl"               172     "REGEX" 0.90
  fill_slot "hdl"               30      "REGEX" 0.90
  fill_slot "triglycerides"     310     "REGEX" 0.90
  fill_slot "on_statin"         true

  # Medications
  fill_slot "current_medications" '["metformin 1000mg BD","empagliflozin 25mg OD","telmisartan 40mg OD","rosuvastatin 10mg OD","aspirin 75mg OD"]' "NLU" 0.85
  fill_slot "medication_count"    5
  fill_slot "adherence_score"     0.60
  fill_slot "allergies"           '["none known"]' "NLU" 0.90
  fill_slot "supplement_list"     '[]' "NLU" 0.95

  # Lifestyle — sedentary, high-stress executive
  fill_slot "smoking_status"         '"never"'
  fill_slot "alcohol_use"            '"heavy"'
  fill_slot "exercise_minutes_week"  10
  fill_slot "diet_type"              '"non_vegetarian"'
  fill_slot "sleep_hours"            5
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          false

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      false
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Excessive thirst, weight gain despite dieting, snoring and daytime sleepiness, knee pain on walking"' "NLU" 0.78
  fill_slot "comorbidities"             '["type 2 diabetes","hypertension stage 1","dyslipidemia","obesity class II","obstructive sleep apnea","non-alcoholic fatty liver disease"]' "NLU" 0.80

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 5 complete"
fi

# =============================================================================
#  PROFILE 6: Sunita Reddy — 68F, T2DM 18y, CKD 3b, prev MI, 9 meds
# =============================================================================

step "Profile 6/8: Sunita Reddy — 68F, T2DM 18y, CKD 3b, post-MI, HTN Stage 2"
if enroll_patient "Sunita" "Reddy" "+919845100006" "91-1006-2006-3006" "INSURANCE" \
    "sunita.reddy@email.com" "68" "female" "south_asian" "te"; then

  # Demographics
  fill_slot "age"              68
  fill_slot "sex"              '"female"'
  fill_slot "height"           155
  fill_slot "weight"           68
  fill_slot "pregnant"         false
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"te"'

  # Glycemic — long-standing, insulin added
  fill_slot "diabetes_type"          '"type2"'
  fill_slot "fbg"                    155     "DEVICE" 0.95
  fill_slot "hba1c"                  8.2     "REGEX"  0.90
  fill_slot "ppbg"                   220     "DEVICE" 0.93
  fill_slot "diabetes_duration_years" 18
  fill_slot "insulin"                true
  fill_slot "hypoglycemia_episodes"  1

  # Renal — CKD Stage 3b
  fill_slot "egfr"              35      "REGEX" 0.94
  fill_slot "serum_creatinine"  1.6     "REGEX" 0.92
  fill_slot "uacr"              320     "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   5.0     "REGEX" 0.90

  # Cardiac — post-MI 90 days, HTN Stage 2
  fill_slot "systolic_bp"         158     "DEVICE" 0.98
  fill_slot "diastolic_bp"        96      "DEVICE" 0.98
  fill_slot "heart_rate"          78      "DEVICE" 0.95
  fill_slot "nyha_class"          2
  fill_slot "mi_stroke_days"      90
  fill_slot "lvef"                42      "REGEX"  0.85
  fill_slot "atrial_fibrillation" false

  # Lipid
  fill_slot "total_cholesterol" 210     "REGEX" 0.90
  fill_slot "ldl"               125     "REGEX" 0.90
  fill_slot "hdl"               40      "REGEX" 0.90
  fill_slot "triglycerides"     180     "REGEX" 0.90
  fill_slot "on_statin"         true

  # Medications — post-MI + DM + CKD regimen
  fill_slot "current_medications" '["insulin glargine 20U HS","vildagliptin 50mg BD","losartan 50mg OD","amlodipine 10mg OD","metoprolol succinate 50mg OD","clopidogrel 75mg OD","atorvastatin 80mg OD","pantoprazole 40mg OD","aspirin 75mg OD"]' "NLU" 0.80
  fill_slot "medication_count"    9
  fill_slot "adherence_score"     0.70
  fill_slot "allergies"           '["ACE inhibitors (angioedema)","NSAIDs (GI bleed)"]' "NLU" 0.88
  fill_slot "supplement_list"     '["calcium + vitamin D3","vitamin B12"]' "NLU" 0.85

  # Lifestyle
  fill_slot "smoking_status"         '"never"'
  fill_slot "alcohol_use"            '"never"'
  fill_slot "exercise_minutes_week"  20
  fill_slot "diet_type"              '"vegetarian"'
  fill_slot "sleep_hours"            6
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          true

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      false
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Chest tightness on exertion, swollen feet in evenings, blurred vision, fatigue and weakness"' "NLU" 0.76
  fill_slot "comorbidities"             '["type 2 diabetes","CKD stage 3b","hypertension stage 2","coronary artery disease (post-MI)","heart failure NYHA II","diabetic retinopathy","macroalbuminuria","osteoporosis"]' "NLU" 0.80

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 6 complete"
fi

# =============================================================================
#  PROFILE 7: Arjun Menon — 28M, T1DM 15y, CKD 1, athletic
# =============================================================================

step "Profile 7/8: Arjun Menon — 28M, T1DM 15y, CKD 1, athletic, well-controlled"
if enroll_patient "Arjun" "Menon" "+919845100007" "91-1007-2007-3007" "CORPORATE" \
    "arjun.menon@email.com" "28" "male" "south_asian" "ml"; then

  # Demographics
  fill_slot "age"              28
  fill_slot "sex"              '"male"'
  fill_slot "height"           178
  fill_slot "weight"           74
  fill_slot "pregnant"         false
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"ml"'

  # Glycemic — T1DM since age 13, tight control
  fill_slot "diabetes_type"          '"type1"'
  fill_slot "fbg"                    102     "DEVICE" 0.97
  fill_slot "hba1c"                  6.5     "REGEX"  0.95
  fill_slot "ppbg"                   138     "DEVICE" 0.95
  fill_slot "diabetes_duration_years" 15
  fill_slot "insulin"                true
  fill_slot "hypoglycemia_episodes"  3

  # Renal — CKD Stage 1 (normal eGFR, mild albuminuria)
  fill_slot "egfr"              112     "REGEX" 0.94
  fill_slot "serum_creatinine"  0.9     "REGEX" 0.92
  fill_slot "uacr"              35      "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   4.1     "REGEX" 0.90

  # Cardiac — normal, athletic
  fill_slot "systolic_bp"         116     "DEVICE" 0.98
  fill_slot "diastolic_bp"        72      "DEVICE" 0.98
  fill_slot "heart_rate"          56      "DEVICE" 0.95
  fill_slot "nyha_class"          1
  fill_slot "mi_stroke_days"      0
  fill_slot "lvef"                65      "REGEX"  0.85
  fill_slot "atrial_fibrillation" false

  # Lipid — excellent
  fill_slot "total_cholesterol" 165     "REGEX" 0.90
  fill_slot "ldl"               85      "REGEX" 0.90
  fill_slot "hdl"               62      "REGEX" 0.90
  fill_slot "triglycerides"     90      "REGEX" 0.90
  fill_slot "on_statin"         false

  # Medications — insulin pump + ACE for nephroprotection
  fill_slot "current_medications" '["insulin lispro pump (basal 0.6U/hr)","insulin lispro bolus PRN","ramipril 5mg OD"]' "NLU" 0.90
  fill_slot "medication_count"    3
  fill_slot "adherence_score"     0.95
  fill_slot "allergies"           '["none known"]' "NLU" 0.95
  fill_slot "supplement_list"     '["whey protein","creatine monohydrate","vitamin D3"]' "NLU" 0.90

  # Lifestyle — very active
  fill_slot "smoking_status"         '"never"'
  fill_slot "alcohol_use"            '"occasional"'
  fill_slot "exercise_minutes_week"  300
  fill_slot "diet_type"              '"non_vegetarian"'
  fill_slot "sleep_hours"            8
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          false

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      false
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Exercise-induced hypoglycemia 2-3 times per month, mild dawn phenomenon"' "NLU" 0.85
  fill_slot "comorbidities"             '["type 1 diabetes","CKD stage 1 (microalbuminuria)"]' "NLU" 0.88

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 7 complete"
fi

# =============================================================================
#  PROFILE 8: Kavitha Iyer — 50F, T2DM 8y, CKD 3a, HTN, 7 meds
# =============================================================================

step "Profile 8/8: Kavitha Iyer — 50F, T2DM 8y, CKD 3a, HTN, PCOS history"
if enroll_patient "Kavitha" "Iyer" "+919845100008" "91-1008-2008-3008" "INSURANCE" \
    "kavitha.iyer@email.com" "50" "female" "south_asian" "ta"; then

  # Demographics
  fill_slot "age"              50
  fill_slot "sex"              '"female"'
  fill_slot "height"           162
  fill_slot "weight"           78
  fill_slot "pregnant"         false
  fill_slot "ethnicity"        '"south_asian"'
  fill_slot "primary_language" '"ta"'

  # Glycemic — moderate duration, on triple oral
  fill_slot "diabetes_type"          '"type2"'
  fill_slot "fbg"                    138     "DEVICE" 0.95
  fill_slot "hba1c"                  7.5     "REGEX"  0.92
  fill_slot "ppbg"                   185     "DEVICE" 0.93
  fill_slot "diabetes_duration_years" 8
  fill_slot "insulin"                false
  fill_slot "hypoglycemia_episodes"  0

  # Renal — CKD Stage 3a
  fill_slot "egfr"              48      "REGEX" 0.94
  fill_slot "serum_creatinine"  1.3     "REGEX" 0.92
  fill_slot "uacr"              150     "REGEX" 0.88
  fill_slot "dialysis"          false
  fill_slot "serum_potassium"   4.8     "REGEX" 0.90

  # Cardiac — HTN Stage 1
  fill_slot "systolic_bp"         140     "DEVICE" 0.98
  fill_slot "diastolic_bp"        86      "DEVICE" 0.98
  fill_slot "heart_rate"          80      "DEVICE" 0.95
  fill_slot "nyha_class"          1
  fill_slot "mi_stroke_days"      0
  fill_slot "lvef"                56      "REGEX"  0.85
  fill_slot "atrial_fibrillation" false

  # Lipid — borderline high
  fill_slot "total_cholesterol" 230     "REGEX" 0.90
  fill_slot "ldl"               145     "REGEX" 0.90
  fill_slot "hdl"               42      "REGEX" 0.90
  fill_slot "triglycerides"     200     "REGEX" 0.90
  fill_slot "on_statin"         true

  # Medications — triple oral + antihypertensives
  fill_slot "current_medications" '["metformin 500mg BD","teneligliptin 20mg OD","dapagliflozin 10mg OD","telmisartan 40mg OD","amlodipine 5mg OD","rosuvastatin 10mg OD","aspirin 75mg OD"]' "NLU" 0.85
  fill_slot "medication_count"    7
  fill_slot "adherence_score"     0.80
  fill_slot "allergies"           '["sulfonamides (rash)"]' "NLU" 0.88
  fill_slot "supplement_list"     '["calcium + vitamin D3","vitamin B12","iron folic acid"]' "NLU" 0.85

  # Lifestyle
  fill_slot "smoking_status"         '"never"'
  fill_slot "alcohol_use"            '"never"'
  fill_slot "exercise_minutes_week"  60
  fill_slot "diet_type"              '"vegetarian"'
  fill_slot "sleep_hours"            6.5
  fill_slot "active_substance_abuse" false
  fill_slot "falls_history"          false

  # Symptoms / Conditions
  fill_slot "active_cancer"             false
  fill_slot "organ_transplant"          false
  fill_slot "cognitive_impairment"      false
  fill_slot "bariatric_surgery_months"  0
  fill_slot "primary_complaint"         '"Increased fatigue, weight fluctuation, facial hair growth, irregular periods transitioning to menopause"' "NLU" 0.78
  fill_slot "comorbidities"             '["type 2 diabetes","CKD stage 3a","hypertension stage 1","dyslipidemia","PCOS (history)","hypothyroidism","vitamin D deficiency"]' "NLU" 0.82

  evaluate_safety
  ((TOTAL_PATIENTS++))
  ok "Profile 8 complete"
fi

# =============================================================================
#  SUMMARY
# =============================================================================

step "SEED COMPLETE"
echo ""
echo -e "${BOLD}Profiles created: ${GREEN}$TOTAL_PATIENTS${NC} / 8"
echo ""
echo -e "${BOLD}Profile Summary:${NC}"
echo "  1. Rajesh Kumar   — 58M, T2DM 10y, CKD 3a, HTN-1, smoker, 6 meds"
echo "  2. Priya Nair     — 34F, GDM (pregnant), healthy renal/cardiac, 2 meds"
echo "  3. Anand Sharma   — 72M, T2DM 25y, CKD 4, HF NYHA III, AFib, 10 meds"
echo "  4. Meena Sundaram — 45F, T1DM 20y, CKD 2, well-controlled, 4 meds"
echo "  5. Vikram Patel   — 55M, T2DM 5y, metabolic syndrome, obese, 5 meds"
echo "  6. Sunita Reddy   — 68F, T2DM 18y, CKD 3b, post-MI, 9 meds"
echo "  7. Arjun Menon    — 28M, T1DM 15y, CKD 1, athletic, 3 meds"
echo "  8. Kavitha Iyer   — 50F, T2DM 8y, CKD 3a, PCOS hx, 7 meds"
echo ""
echo -e "${BOLD}Coverage:${NC}"
echo "  DM types:    T1DM (2), T2DM (5), GDM (1)"
echo "  CKD stages:  None (2), G1 (1), G2 (1), G3a (2), G3b (1), G4 (1)"
echo "  Age range:   28–72 years"
echo "  Gender:      Male (4), Female (4)"
echo "  Meds range:  2–10 medications"
echo "  Languages:   Hindi (2), Tamil (2), Malayalam (2), Telugu (1), Gujarati (1)"
echo ""
echo -e "${GREEN}${BOLD}All FHIR resources (Patient, Encounter, Observation, DetectedIssue) created via intake service.${NC}"
