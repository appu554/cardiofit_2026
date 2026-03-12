# HTN Integration — Implementation Plan

**Sources**:
- `Vaidshala_HTN_Integration_Amendments.docx` (13 amendments)
- `Vaidshala_HTN_EarlyWarning_Deprescribing.docx` (9 EW items + 10 AD items)

**Date**: 2026-03-09
**Scope**: Transform Vaidshala from "diabetes correction loop safe for HTN patients" → "DM+HTN co-management loop with renal protection, early warning, and safe deprescribing"

---

## Priority Legend

| Priority | Meaning | Gate |
|----------|---------|------|
| **P0** | Must ship before any HTN-aware pilot | Blocks clinical deployment |
| **P1** | Must ship before general availability | Blocks GA |
| **P2** | Ship in second wave | Post-GA enhancement |

---

## Wave 1 — P0 (Blocks Pilot)

### 1.1 Amendment 1: RAAS Creatinine Tolerance Rule

**Why P0**: Without this, every ACEi/ARB uptitration fires false AKI HALT and removes the most renal-protective drug.

#### Step 1: Extend Channel B inputs with RAAS context

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/raw_inputs.go`

Add to `RawPatientData`:
```go
RAASChangeWithin14d     bool       // true if ACEi or ARB dose changed in last 14 days
CreatininePreRAASBaseline *float64 // creatinine value before the RAAS change
RAASChangeDateUTC       *time.Time // when the RAAS change occurred
```

#### Step 2: Add PG-14 to Channel C protocol rules

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/protocol_rules.yaml`

```yaml
- id: PG-14
  name: "RAAS creatinine tolerance"
  description: "Expected creatinine rise after ACEi/ARB intensification — not AKI"
  condition_field: raas_creatinine_tolerance
  operator: eq
  threshold: true
  gate: PAUSE
  guideline_ref: KDIGO_CKD_2021_RAAS_TOLERANCE
  status: active
```

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_c/context.go`

Add to `TitrationContext`:
```go
RAASChangeWithin14d        bool    // ACEi/ARB dose changed within 14 days
CreatinineRisePctFromRAAS  float64 // % rise from pre-RAAS baseline
PotassiumCurrent           float64 // K+ level (PG-14 requires <5.5)
OliguriaReported           bool    // from KB-22 or Tier-1
CreatinineRiseExplained    bool    // computed: true if RAAS change + rise <30% + K<5.5 + no oliguria
```

#### Step 3: Modify Channel B B-03 to respect causal suppression

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor.go`

In `checkB03` (creatinine 48h delta > 26 µmol/L):
- If `CreatinineRiseExplained == true` → downgrade from HALT to PAUSE
- SafetyTrace records both the raw B-03 evaluation AND the causal suppression

#### Step 4: Orchestrator computes `CreatinineRiseExplained` before Channel B

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go`

In `RunCycle()`, insert between step 2 (Channel A) and step 3 (Channel B):
```
if raw.RAASChangeWithin14d && raw.CreatininePreRAASBaseline != nil {
    risePct = (creatinineCurrent - baseline) / baseline * 100
    if risePct < 30 && potassium < 5.5 && !oliguria {
        raw.CreatinineRiseExplained = true
    }
}
```

#### Step 5: RAAS monitoring protocol card in KB-23

**New file**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/templates/raas_creatinine_monitoring.yaml`

Template for MEDICATION_REVIEW card with embedded monitoring protocol:
- Day 7: Repeat creatinine + K+. If rise >30% OR K+>5.5 → escalate to HALT
- Day 14: If stabilised within 30% → resume SAFE, set new baseline
- Day 30: eGFR recalculation. If >5% below pre-change AND declining → HALT + nephrology referral

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_builder.go`

Add RAAS monitoring card generation when PG-14 fires PAUSE.

#### Step 5b: RAAS monitoring re-evaluation trigger

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

When a creatinine lab result arrives during an active RAAS monitoring window:
- Day 7 creatinine arrives → evaluate against 30% threshold → emit `RAAS_MONITORING_ESCALATE` event if rise >30% OR K+>5.5
- Day 14 creatinine arrives → evaluate stabilisation → emit `RAAS_MONITORING_RESOLVED` if within 30%
- Day 30 eGFR recalculated → emit `RAAS_MONITORING_ESCALATE` if eGFR >5% below pre-change AND declining

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/events.go`

Add event types:
```go
EventRAASMonitoringEscalate = "RAAS_MONITORING_ESCALATE"  // lab result triggers safety re-check
EventRAASMonitoringResolved = "RAAS_MONITORING_RESOLVED"  // creatinine stabilised within window
```

The `RAAS_MONITORING_ESCALATE` event is consumed by the V-MCU orchestrator to upgrade the existing PAUSE to HALT and notify via KB-23 card.

#### Step 6: KB-20 medication service exposes RAAS change recency

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/medication_service.go`

Add method: `GetRAASChangeRecency(patientID) → (changed bool, changeDate time.Time, preChangeCr float64)`

#### Step 7: Tests

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor_test.go`
- `TestB03_WithRAASToleranceSuppression` — creatinine rise 20%, RAAS change 5 days ago → PAUSE not HALT
- `TestB03_RAASToleranceExceeded` — creatinine rise 35% → HALT despite RAAS change
- `TestB03_RAASToleranceWithHighPotassium` — K+ 5.8 → HALT despite <30% rise

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_c/guard_test.go`
- `TestPG14_RAASCreatinineTolerance`

**New scenario**: `vaidshala/clinical-runtime-platform/engines/vmcu/simulation/scenario_9_raas_tolerance_test.go`
- Scenario A: Genuine AKI (no RAAS change) → B-03 HALT fires normally
- Scenario B: Expected RAAS response (ACEi increased 5 days ago, rise 18%) → PAUSE with monitoring card

---

### 1.2 Amendment 8: J-Curve eGFR-Stratified BP Floor

**Why P0**: Prevents renal perfusion harm in CKD Stage 3-4 patients from aggressive BP lowering.

#### Step 1: Add `SBPLowerLimit` to Channel B inputs

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/raw_inputs.go`

Add to `RawPatientData`:
```go
SBPLowerLimit *float64 // eGFR-stratified lower BP limit (computed by orchestrator)
```

#### Step 2: Add B-12 rule to Channel B

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor.go`

New rule `checkB12`:
```
if SBP < SBPLowerLimit → PAUSE
if SBP < 90 → HALT (B-05 unchanged — absolute floor)
```

Add thresholds to `PhysioConfig`:
```go
SBPLowerLimitStage3a float64 // default 100
SBPLowerLimitStage3b float64 // default 105
SBPLowerLimitStage4  float64 // default 110
SBPHaltFloorStage4   float64 // default 100 (below this → HALT for Stage 4)
```

#### Step 3: Orchestrator computes `sbp_lower_limit` from eGFR

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go`

In `RunCycle()`, before Channel B evaluation:
```
eGFR → sbp_lower_limit mapping:
  eGFR >= 60       → 90   (no modification, B-05 applies)
  eGFR 45-59 (3a)  → 100
  eGFR 30-44 (3b)  → 105
  eGFR 15-29 (4)   → 110  (HALT if <100)
  eGFR < 15  (5)   → B-08 already fires HALT
```

SafetyTrace records: applied threshold + source eGFR value.

#### Step 4: Tests

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor_test.go`
- `TestB12_JCurve_Stage3a` — SBP 98 + eGFR 50 → PAUSE
- `TestB12_JCurve_Stage3b` — SBP 103 + eGFR 35 → PAUSE
- `TestB12_JCurve_Stage4` — SBP 95 + eGFR 20 → HALT (below 100 for Stage 4)
- `TestB12_NormalEGFR_NoModification` — SBP 92 + eGFR 70 → only B-05 fires if SBP<90

---

### 1.3 AD-09: CKD Stage 4 Antihypertensive Deprescribing Hard Block

**Why P0**: CKD Stage 4 patients have J-curve floor at 110 mmHg — no BP headroom to remove any drug. System must never propose antihypertensive step-down for these patients.

#### Step 1: Add Stage 4 exclusion to deprescribing eligibility

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`

In the DEPRESCRIBING_MODE evaluator (or wherever `htn_deprescribing_eligible` is computed):
```go
if patient.CKDStage == "STAGE_4" || patient.CKDStage == "STAGE_5" {
    htnDeprescribingEligible = false
    exclusionReason = "CKD_STAGE_4_5_BP_FLOOR_CONSTRAINT"
}
```

#### Step 2: Guard in KB-23 DEPRESCRIBING_PROPOSAL generation

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_builder.go`

Before generating any antihypertensive DEPRESCRIBING_PROPOSAL card, check CKD stage from KB-20 patient context. If Stage 4+, do not generate card.

#### Step 3: Tests

- Unit test: patient with eGFR=22 (Stage 4) + BP at target 16 weeks → `htn_deprescribing_eligible = false`
- Unit test: patient with eGFR=55 (Stage 3a) + BP at target 16 weeks → `htn_deprescribing_eligible = true`

---

### 1.4 VitalFact BP Extension Fields + bp_pattern + Measurement Uncertainty

**Why P0**: Without contextual BP metadata, Channel B cannot distinguish genuine hypertension from white-coat artefact or postural changes. Without `measurement_uncertainty`, safety rules treat noisy readings with the same confidence as clean data.

#### Step 1: Extend VitalFact with BP metadata

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go`

Add to `VitalFact` struct:
```go
// BP measurement context fields
MeasurementContext       string   // CLINIC | HOME | AMBULATORY | PHARMACY
Posture                  string   // SEATED | STANDING | SUPINE | UNKNOWN
Arm                      string   // LEFT | RIGHT | BOTH | UNKNOWN
TimeOfDay                string   // MORNING_FASTING | MORNING_POST_MED | AFTERNOON | EVENING | NOCTURNAL
ConsecutiveReadingIndex  int      // 0-based; 0 = first reading, 1 = second (use for averaging)
MinutesSinceLastActivity *int     // nil = unknown; used for resting state validation
WhiteCoatFlag            bool     // true if historical pattern suggests white-coat effect
MeasurementUncertainty   string   // LOW | MODERATE | HIGH (HIGH when irregular HR, postural change, single reading)
```

#### Step 2: Add `bp_pattern` enum to BPTrajectory

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

Add to `BPTrajectory`:
```go
BPPattern string // NORMAL | WHITE_COAT | MASKED | MORNING_SURGE | NOCTURNAL_NONDIP
```

Detection logic (in `lab_service.go`):
- `WHITE_COAT`: clinic SBP consistently >10 mmHg above home SBP (≥3 paired readings)
- `MASKED`: home SBP consistently >10 mmHg above clinic SBP (≥3 paired readings)
- `MORNING_SURGE`: morning_fasting SBP − evening SBP > 20 mmHg (≥5 day pairs)
- `NOCTURNAL_NONDIP`: nocturnal SBP / daytime SBP > 0.90 (ambulatory data required)
- `NORMAL`: none of the above patterns detected

#### Step 3: Propagate `measurement_uncertainty` to Channel B

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/raw_inputs.go`

Add to `RawPatientData`:
```go
MeasurementUncertainty string // LOW | MODERATE | HIGH — from VitalFact
BPPattern              string // from BPTrajectory
```

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor.go`

Modify all SBP/DBP-dependent rules (B-05, B-12, B-17–B-19):
- When `MeasurementUncertainty == "HIGH"`: downgrade HALT → PAUSE, append `"uncertainty_dampened": true` to SafetyTrace
- When `BPPattern == "WHITE_COAT"`: do not fire PAUSE/HALT on elevated clinic readings alone; require home confirmation

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go`

Populate `raw.MeasurementUncertainty` from KB-20 VitalFact and `raw.BPPattern` from BPTrajectory during cache resolution.

#### Step 4: Tests

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor_test.go`
- `TestB05_HighUncertainty_Dampened` — SBP=88 + uncertainty HIGH → PAUSE (not HALT)
- `TestB12_WhiteCoat_ClinicReadingIgnored` — clinic SBP=165 + BPPattern=WHITE_COAT + home SBP=128 → no PAUSE
- `TestB12_NormalPattern_ClinicReadingApplied` — clinic SBP=165 + BPPattern=NORMAL → PAUSE

---

### 1.5 Channel C HTN Protocol Rules PG-08 through PG-13

**Why P0**: These 6 rules protect against dangerous HTN drug interactions that Channel B physiology rules alone cannot catch. Without them, the system cannot safely manage HTN co-prescriptions.

#### Step 1: Add 6 new rules to Channel C protocol YAML

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/protocol_rules.yaml`

```yaml
- id: PG-08
  name: "ACEi/ARB + hyperkalaemia + declining eGFR"
  description: "ACEi or ARB active AND K+ > 5.5 AND eGFR slope negative → immediate safety halt"
  condition_field: acei_arb_hyperK_declining_egfr
  operator: eq
  threshold: true
  gate: HALT
  guideline_ref: KDIGO_CKD_2021_RAAS_SAFETY
  status: active

- id: PG-09
  name: "Beta-blocker + insulin interaction"
  description: "Beta-blocker masks hypoglycaemia symptoms in insulin-treated patients"
  condition_field: beta_blocker_insulin_active
  operator: eq
  threshold: true
  gate: MODIFY
  safety_instruction: "Warn patient: beta-blocker may mask hypo symptoms. Consider cardioselective agent."
  guideline_ref: ESC_HTN_2023_BETA_BLOCKER_DM
  status: active

- id: PG-10
  name: "SGLT2i + low SBP"
  description: "SGLT2i active AND SBP 7-day mean < 105 mmHg → pause titration"
  condition_field: sglt2i_low_sbp
  operator: eq
  threshold: true
  gate: PAUSE
  guideline_ref: KDIGO_CKD_2024_SGLT2I_BP
  status: active

- id: PG-11
  name: "Thiazide + hypokalaemia"
  description: "Thiazide active AND K+ < 3.5 mmol/L → pause and correct K+"
  condition_field: thiazide_hypoK
  operator: eq
  threshold: true
  gate: PAUSE
  guideline_ref: ESH_HTN_2023_THIAZIDE_MONITORING
  status: active

- id: PG-12
  name: "Dual RAAS blockade"
  description: "ACEi AND ARB both active simultaneously → halt (contraindicated)"
  condition_field: dual_raas_active
  operator: eq
  threshold: true
  gate: HALT
  safety_instruction: "Dual RAAS blockade contraindicated. Discontinue one agent."
  guideline_ref: ONTARGET_DUAL_RAAS_CONTRAINDICATION
  status: active

- id: PG-13
  name: "High-dose HCTZ"
  description: "HCTZ dose > 25mg → modify to evidence-based maximum"
  condition_field: hctz_high_dose
  operator: eq
  threshold: true
  gate: MODIFY
  safety_instruction: "HCTZ >25mg increases metabolic side effects without additional BP benefit. Reduce to 12.5-25mg or switch to chlorthalidone."
  guideline_ref: NICE_HTN_2023_THIAZIDE_DOSE
  status: active
```

#### Step 2: Extend Channel C context with required fields

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_c/context.go`

Add to `TitrationContext`:
```go
// PG-08 fields
ACEiOrARBActive       bool
PotassiumAbove55      bool    // K+ > 5.5 mmol/L
EGFRSlopeNegative     bool    // eGFR declining

// PG-09 fields
BetaBlockerActive     bool
InsulinActive         bool

// PG-10 fields
SGLT2iActive          bool
SBP7dMean             float64

// PG-11 fields
ThiazideActive        bool
PotassiumBelow35      bool    // K+ < 3.5 mmol/L

// PG-12 fields
DualRAASActive        bool    // both ACEi AND ARB active

// PG-13 fields
HCTZDoseMg            float64 // current HCTZ dose in mg
```

#### Step 3: Orchestrator populates new context fields

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go`

In `RunCycle()`, before Channel C evaluation, compute composite boolean fields:
```
ctx.ACEiOrARBActive = hasActiveMed("ACEi") || hasActiveMed("ARB")
ctx.PotassiumAbove55 = raw.PotassiumCurrent != nil && *raw.PotassiumCurrent > 5.5
ctx.EGFRSlopeNegative = raw.EGFRSlope != nil && *raw.EGFRSlope < 0
ctx.BetaBlockerActive = hasActiveMed("BETA_BLOCKER")
ctx.InsulinActive = hasActiveMed("INSULIN")
ctx.SGLT2iActive = hasActiveMed("SGLT2I")
ctx.SBP7dMean = raw.SBP7dMean
ctx.ThiazideActive = hasActiveMed("THIAZIDE")
ctx.PotassiumBelow35 = raw.PotassiumCurrent != nil && *raw.PotassiumCurrent < 3.5
ctx.DualRAASActive = hasActiveMed("ACEi") && hasActiveMed("ARB")
ctx.HCTZDoseMg = getActiveDoseMg("HCTZ")
```

Also add `EGFRSlope *float64` to `raw_inputs.go` → `RawPatientData` if not already present.

#### Step 4: Tests

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_c/guard_test.go`
- `TestPG08_ACEiHyperKDecliningEGFR` — ACEi active + K+=5.8 + eGFR slope −2.1 → HALT
- `TestPG09_BetaBlockerInsulin` — beta-blocker + insulin both active → MODIFY + safety instruction
- `TestPG10_SGLT2iLowSBP` — SGLT2i active + SBP 7d mean 102 → PAUSE
- `TestPG11_ThiazideHypoK` — thiazide active + K+=3.2 → PAUSE
- `TestPG12_DualRAAS` — ACEi + ARB both active → HALT + safety instruction
- `TestPG13_HighDoseHCTZ` — HCTZ 50mg → MODIFY + safety instruction

---

### 1.6 Tier-1 Question Extensions for HTN

**Why P0**: Several safety rules (PG-14 oliguria check, B-16 AF investigation, adherence gating) depend on patient-reported data that must be collected via Tier-1 questions. Without these questions, the rules have no input signal.

#### KB-22 Tier-1 Question Additions

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/nodes/tier1_htn_questions.yaml`

```yaml
questions:
  - id: T1_HTN_01
    text: "Have you noticed reduced urine output in the last 24 hours?"
    maps_to: oliguria_reported
    trigger: "RAAS_MONITORING active"
    response_type: BOOLEAN

  - id: T1_HTN_02
    text: "Have you been taking your blood pressure tablets regularly?"
    maps_to: antihypertensive_adherence_self_report
    trigger: "HYPERTENSION_REVIEW pending"
    response_type: LIKERT_5  # Always / Usually / Sometimes / Rarely / Never

  - id: T1_HTN_03
    text: "Have you noticed any dry cough that started after your BP medication?"
    maps_to: acei_cough_symptom
    trigger: "ACEi active"
    response_type: BOOLEAN

  - id: T1_HTN_04
    text: "Do you feel dizzy when you stand up from sitting or lying down?"
    maps_to: orthostatic_symptoms
    trigger: "antihypertensive_count >= 2"
    response_type: FREQUENCY  # Never / Occasionally / Often / Always

  - id: T1_HTN_05
    text: "Have you experienced any palpitations or irregular heartbeat?"
    maps_to: irregular_hr_symptom
    trigger: "BP_VARIABILITY_ALERT OR HR_IRREGULAR"
    response_type: BOOLEAN
```

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/internal/services/question_router.go`

Add HTN question trigger evaluation: when V-MCU gate signals or card events match question triggers, enqueue the corresponding Tier-1 question for the patient's next interaction.

---

## Wave 2 — P1 (Blocks GA)

### 2.1 Amendment 2: Antihypertensive TreatmentPerturbation Windows

#### Files to modify:

1. **`backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/treatment_perturbation.go`**
   - Extend `TreatmentPerturbation` struct:
     ```go
     ExpectedDirection    string   // "UP" or "DOWN"
     ExpectedMagnitudeMin float64  // lower bound of expected change
     ExpectedMagnitudeMax float64  // upper bound
     CausalNote           string   // human-readable for CTL Panel 3
     ```

2. **`backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/perturbation_service.go`**
   - Add perturbation window definitions for 6 drug classes:
     | Drug Class | Signal | Window | Direction |
     |------------|--------|--------|-----------|
     | SGLT2i added | SBP | 4 weeks | DOWN (−3 to −5 mmHg) |
     | SGLT2i stopped | SBP | 4 weeks | UP (+3 to +5) |
     | ACEi/ARB started/increased | Creatinine | 14 days | UP (+10-30%) |
     | ACEi/ARB stopped | Creatinine | 7 days | DOWN |
     | Thiazide added | K+ | 3 weeks | DOWN (−0.3 to −0.5) |
     | Beta-blocker added/increased | HR, glucose | 2 weeks | DOWN (HR), UP (glucose) |

3. **`vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go`**
   - `resolveFromCache` passes `affected_observables` to Channel B and C contexts

4. **New migration**: `kb-23-decision-cards/migrations/004_perturbation_extensions.sql`
   - Add columns: `expected_direction`, `expected_magnitude_min`, `expected_magnitude_max`, `causal_note`

---

### 2.2 Amendment 3: Heart Rate VitalFact + Channel B Rules

#### Files to modify:

1. **`vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/raw_inputs.go`**
   ```go
   HeartRateCurrent  *float64          // bpm
   HRRegularity      string            // REGULAR | IRREGULAR | UNKNOWN
   HRContext         string            // RESTING | POST_ACTIVITY | STANDING | SUPINE
   HeartRateConfirmed bool             // true if 2 consecutive readings
   BetaBlockerDoseChangeWithin7d bool  // for B-14
   ```

2. **`vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor.go`**
   - Add rules B-13 through B-16:
     | Rule | Condition | Gate |
     |------|-----------|------|
     | B-13 | HR < 45 + RESTING + confirmed | HALT |
     | B-14 | HR < 55 + beta_blocker_active + dose_change_7d | PAUSE |
     | B-15 | HR > 120 + RESTING + confirmed | PAUSE |
     | B-16 | IRREGULAR + confirmed | PAUSE + KB22_TRIGGER |
   - Add thresholds to `PhysioConfig` and `DefaultPhysioConfig()`

3. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go`**
   - Add HR fields to VitalFact/PatientProfile
   - Add `MeasurementUncertainty` flag (HIGH when HR irregular)

4. **New file**: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/nodes/p_irregular_hr.yaml`
   - AF differential node triggered by B-16

5. **`vaidshala/clinical-runtime-platform/engines/vmcu/events/kb19_handler.go`**
   - Publish `KB22_HPI_TRIGGER` event when B-16 fires
   - **KB22_TRIGGER mechanism**: B-16 fires → arbiter records PAUSE → vmcu_engine.go step 12 checks for `KB22_TRIGGER` annotation on gate signals → publishes `KB22_HPI_TRIGGER` event to KB-19 event bus → KB-19 routes to KB-22 with node ID `p_irregular_hr` → KB-22 opens differential investigation

6. **PG-16: Atrial Fibrillation anticoagulation safety rule**

   **File**: `vaidshala/clinical-runtime-platform/engines/vmcu/protocol_rules.yaml`
   ```yaml
   - id: PG-16
     name: "AF detected — anticoagulation check"
     description: "Irregular HR confirmed as AF → check anticoagulation status"
     condition_field: af_confirmed_no_anticoagulation
     operator: eq
     threshold: true
     gate: PAUSE
     safety_instruction: "AF confirmed. Assess CHA2DS2-VASc and initiate anticoagulation if indicated."
     guideline_ref: ESC_AF_2020_ANTICOAGULATION
     status: active
   ```

   **File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_c/context.go`
   Add: `AFConfirmedNoAnticoagulation bool` to `TitrationContext`

   **Dependency**: Requires KB-22 `p_irregular_hr` node to confirm AF diagnosis (posterior >0.80) AND KB-20 MedicationFact to confirm no active anticoagulant.

7. **Beta-blocker perturbation: HR threshold adjustment (−5 bpm)**

   **File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/perturbation_service.go`

   When beta-blocker perturbation is active (dose change within 2 weeks):
   - Adjust Channel B HR thresholds: B-13 bradycardia threshold shifts from 45 → 50 bpm (more conservative)
   - B-14 threshold shifts from 55 → 60 bpm
   - Pass `BetaBlockerHRThresholdAdjust int` (default −5) to Channel B via `RawPatientData`
   - SafetyTrace records: `"hr_threshold_adjusted": true, "adjustment_reason": "beta_blocker_perturbation"`

8. **Thiazide perturbation: K+ causal context card annotation**

   **File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_builder.go`

   When thiazide perturbation is active AND K+ drops within expected range (−0.3 to −0.5 mmol/L):
   - Add `causal_context` annotation to any HYPERTENSION_REVIEW or MEDICATION_REVIEW card:
     `"K+ decline is within expected range for recent thiazide initiation. Monitor but do not discontinue unless K+ < 3.5."`
   - If K+ < 3.5: annotation changes to: `"K+ below safe threshold despite expected thiazide effect. Consider K+ supplementation or dose reduction."`

---

### 2.3 Amendment 4: Antihypertensive Adherence in KB-21

#### Files to modify:

1. **`backend/shared-infrastructure/knowledge-base-services/kb-21-behavioral-intelligence/internal/models/models.go`**
   ```go
   // Add to AdherenceState or new parallel struct:
   AntihypertensiveAdherence map[string]float64 // drug_class → 0.0-1.0
   AdherenceReason           string             // COST | SIDE_EFFECT | FORGOT | SUPPLY | UNKNOWN
   ```

2. **`backend/shared-infrastructure/knowledge-base-services/kb-21-behavioral-intelligence/internal/services/adherence_service.go`**
   - Add antihypertensive adherence computation (parallel to diabetes drug adherence)
   - FDC-aware: telmisartan+amlodipine treated as single adherence unit

3. **`backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/kb21_client.go`**
   - Fetch `antihypertensive_adherence` alongside existing adherence data

4. **`backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_builder.go`**
   - HYPERTENSION_REVIEW adherence gate:
     | Adherence | Card Behaviour |
     |-----------|---------------|
     | >= 0.85 | Standard escalation |
     | 0.60-0.84 | Lead with adherence finding |
     | < 0.60 | Adherence intervention, not dose card |
     | SIDE_EFFECT | Route to KB-22 HPI |

5. **New migration**: `kb-21-behavioral-intelligence/migrations/004_antihypertensive_adherence.sql`

---

### 2.4 Amendment 5: ACEi Cough + ARB Switch Logic

#### Files to modify:

1. **New file**: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/nodes/p_acei_cough.yaml`
   - COUGH_ON_ACEI node with priors: ACEi-induced 0.60, infection 0.20, cardiac 0.15, other 0.05
   - Discriminating questions: dry cough? started after BP tablet? day and night?

2. **`vaidshala/clinical-runtime-platform/engines/vmcu/protocol_rules.yaml`**
   ```yaml
   - id: PG-15
     name: "ACEi cough safety flag"
     condition_field: acei_induced_cough_probability
     operator: gt
     threshold: 0.70
     gate: MODIFY
     guideline_ref: ACEI_COUGH_ARB_SWITCH
     status: active
   ```

3. **`vaidshala/clinical-runtime-platform/engines/vmcu/channel_c/context.go`**
   - Add `ACEiInducedCoughProbability float64` to `TitrationContext`

4. **New file**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/templates/acei_cough_arb_switch.yaml`
   - MEDICATION_REVIEW: "Likely ACEi-induced cough. Recommend ARB switch."

---

### 2.5 Amendment 6: Resistant Hypertension Classification

#### Files to modify:

1. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`**
   - Add resistant HTN detection: BP above target + 3+ drug classes at optimised doses + 1 diuretic + adherence >= 0.85 + sustained 12+ weeks
   - Emit `RESISTANT_HTN_DETECTED` event

2. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/events.go`**
   - Add `ResistantHTNDetectedPayload`

3. **New KB-22 nodes** (secondary cause investigation):
   - `nodes/p_hyperaldosteronism.yaml` — spontaneous hypoK, muscle weakness
   - `nodes/p_osa_screening.yaml` — STOP-BANG items, morning headache, daytime sleepiness
   - `nodes/p_renal_artery_stenosis.yaml` — flash pulmonary oedema, sudden BP instability

4. **New KB-23 template**: `templates/resistant_htn_review.yaml`
   - HYPERTENSION_REVIEW with RESISTANT flag + specialist referral recommendation

---

### 2.6 Amendment 9: ACR Longitudinal Tracking

#### Files to modify:

1. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`**
   ```go
   type ACRReading struct {
       ValueMgMmol   float64
       CollectedAt   time.Time
       UrineCollection string // SPOT | 24H
   }
   type ACRTracking struct {
       Readings    []ACRReading
       Trend       string // IMPROVING | STABLE | WORSENING
       Category    string // A1 | A2 | A3
       OnRAAS      bool
   }
   ```

2. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`**
   - ACR processing for LOINC 2951-2
   - Trend computation across consecutive readings

3. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/events.go`**
   - `ACR_WORSENING` and `ACR_TARGET_MET` events

4. **`vaidshala/clinical-runtime-platform/engines/vmcu/titration/deprescribing.go`**
   - **Deprescribing veto**: ACEi/ARB must never be deprescribed if ACR = A2 or A3
   - Override check in `StartDeprescribing()` and `StepDown()`

5. **New KB-23 templates**:
   - `templates/acr_worsening_review.yaml` — MEDICATION_REVIEW with renal protection review
   - `templates/acr_milestone.yaml` — MILESTONE: "RAAS therapy is reducing proteinuria"

---

### 2.7 Amendment 11: Hyponatraemia + Seasonal Context

#### Files to modify:

1. **`vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/raw_inputs.go`**
   ```go
   SodiumCurrent   *float64 // Na+ mmol/L — sourced from KB-20 LabFact (NOT VitalFact)
   ThiazideActive  bool
   Season          string   // SUMMER | MONSOON | WINTER | AUTUMN
   ```
   **Note**: Sodium (Na+) is a **LabFact** (LOINC 2951-2), not a VitalFact. It comes from laboratory results via the KB-20 lab pipeline, not from patient-reported vitals.

2. **`vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor.go`**
   - B-17: Na+ < 132 AND thiazide → HALT
   - B-18: Na+ 132-135 AND thiazide → PAUSE
   - B-19: Na+ < 135 AND SUMMER AND thiazide → PAUSE (seasonal amplification)

3. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`**
   - Na+ as LabFact with LOINC 2951-2 in `LabEntry` pipeline
   - Add `SodiumLatest *float64` and `SodiumCollectedAt *time.Time` to lab tracking

4. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/patient_profile.go`**
   - Add `Season` enum to patient context (season is a profile attribute, not a lab value)

5. **`backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`**
   - Season derivation: system date + patient location → SUMMER (May-Jun India), MONSOON (Jul-Sep), etc.

---

### 2.8 Early Warning Loop (EW-01 through EW-08)

**Source**: `Vaidshala_HTN_EarlyWarning_Deprescribing.docx` Part 1

#### EW-01 + EW-02: Risk-Stratified DECLINING Thresholds

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

Add to BPTrajectory struct:
```go
BPRiskStratum          string  // LOW | MODERATE | HIGH | VERY_HIGH
SBPDecliningThreshold  float64 // per-patient, from risk stratum table
```

Risk stratum computation (eGFR × ACR):
| Patient Stratum | DECLINING Threshold | EARLY_WATCH Threshold |
|-----------------|--------------------|-----------------------|
| DM only, ACR A1 | +2.5 mmHg/week | +1.8 mmHg/week (6+ weeks) |
| DM + CKD 3a, ACR A1 | +2.0 | +1.4 (4+ weeks) |
| DM + CKD 3b, any ACR | +1.5 | +1.0 (3+ weeks) |
| DM + any CKD, ACR A2/A3 | +1.5 OR any upward slope 8+ weeks | Any positive slope 6+ weeks |

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

Replace hardcoded +2.5 mmHg/week threshold with patient-specific `SBPDecliningThreshold` from stratum table.

#### EW-03: EARLY_WATCH BP Status Tier

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

Add new bp_status value `EARLY_WATCH` between AT_TARGET and DECLINING:
```go
// bp_status enum extension
const (
    BPStatusAtTarget   = "AT_TARGET"
    BPStatusEarlyWatch = "EARLY_WATCH"    // NEW: slope positive, sustained, below DECLINING threshold
    BPStatusDeclining  = "DECLINING"
    BPStatusAboveTarget = "ABOVE_TARGET"
    BPStatusSevere     = "SEVERE"
)

// Add to BPTrajectory:
ConsecutiveEarlyWatchWeeks int    // counter, resets when slope reverses
EarlyWatchFloor            float64 // stratum-specific minimum slope for EARLY_WATCH
```

Behaviour: EARLY_WATCH does NOT generate a card immediately. It counts weeks. When `ConsecutiveEarlyWatchWeeks` exceeds stratum threshold (6/4/3) → emit `BP_TRAJECTORY_CONCERN` event.

#### EW-04: BP_TRAJECTORY_CONCERN Event + 72h Card

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/event_bus.go`

Add event type: `BP_TRAJECTORY_CONCERN`

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_builder.go`

On `BP_TRAJECTORY_CONCERN` event → generate HYPERTENSION_REVIEW at 72h SLA (lowest priority):
Card note: "BP trending upward slowly. No urgent action needed. Monitor antihypertensive adherence and dietary sodium."

#### EW-05 + EW-06: Time-to-Severe Projection

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

Add to BPTrajectory:
```go
WeeksToSevere    *float64 // (SEVERE_threshold - sbp_7d_mean) / sbp_4w_slope. Nil if slope ≤ 0.
SlopeConfidence  string   // HIGH (≥8 readings/4w) | MODERATE (5-7) | LOW (<5)
```

**New KB-23 template**: `templates/bp_declining.yaml`

Card body includes: "At current rate (+X.X mmHg/week), patient will reach severe range in approximately Y weeks without intervention."
If `SlopeConfidence == LOW`: "Projection based on limited readings — treat as indicative only."

#### EW-07 + EW-08: Compound Damage Concern Score

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

New struct:
```go
type DamageComposite struct {
    Score                int    // 0-8
    VariabilityContrib   int    // 0-2 (1 for MODERATE, 2 for HIGH)
    ACRTrendContrib      int    // 0-2 (1 for WORSENING, 2 for A2→A3)
    PulsePressureContrib int    // 0-2 (1 for >60+WIDENING, 2 for >80)
    BPStatusContrib      int    // 0-2 (2 for ABOVE_TARGET ≥8w + adherence ≥0.85)
    ComputedAt           time.Time
}
```

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

Compute after each BPTrajectory update. Integrate signals from Amendments 7, 9, 13.

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/event_bus.go`

Two new events:
- `BP_SUBCLINICAL_CONCERN` (score 3-4) → KB-23 HYPERTENSION_REVIEW at 72h SLA
- `DAMAGE_COMPOSITE_ALERT` (score ≥5) → KB-23 HYPERTENSION_REVIEW at 24h SLA, HIGH PRIORITY

---

### 2.9 Antihypertensive Deprescribing Operational Specification (AD-01 through AD-08, AD-10)

**Source**: `Vaidshala_HTN_EarlyWarning_Deprescribing.docx` Part 2
*Note: AD-09 (CKD Stage 4 block) is in Wave 1 as P0.*

#### AD-01: Adherence Pre-Condition Gate

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`

Add to DEPRESCRIBING_MODE entry evaluation:
```
if antihypertensive_adherence < 0.85 for the 16-week window:
    htn_deprescribing_eligible = false
    if adherence 0.70-0.84: generate MEDICATION_REVIEW card instead
    if adherence < 0.70: do not enter DEPRESCRIBING_MODE
```

#### AD-03: SGLT2i Buffer Check

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`

Add to DEPRESCRIBING_MODE entry evaluation:
```
if sglt2i_stopped_within_6_weeks:
    htn_deprescribing_eligible = false
    reason = "SGLT2I_RECENTLY_STOPPED_BP_BUFFER_LOST"
```

Check `MedicationFact` for SGLT2i status + stop date from KB-20.

#### AD-04: Dose-Halving State Machine (Two-Action Card)

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_lifecycle.go`

New state machine for antihypertensive deprescribing:
```
DOSE_REDUCTION_PROPOSED → (physician approves) →
DOSE_REDUCTION_APPROVED → (dose halved) →
MONITORING → (monitoring window expires) →
  if BP AT_TARGET → REMOVAL_PROPOSED → (physician approves) → REMOVED
  if BP ABOVE_TARGET → STEP_DOWN_FAILED → re-escalation card
```

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/models/decision_card.go`

Add deprescribing state fields:
```go
DeprescribingPhase    string  // DOSE_REDUCTION | MONITORING | REMOVAL | FAILED
DeprescribingDrugClass string
PreStepDownDose       float64
CurrentStepDownDose   float64
MonitoringWindowWeeks int     // 4 for thiazide/CCB, 6 for beta-blocker/ACEi-ARB
MonitoringStartDate   *time.Time
```

Per-class step-down sequences:
| Drug Class | Step 1 (Halve) | Monitor | Step 2 (Remove) |
|------------|---------------|---------|-----------------|
| Thiazide (HCTZ 25mg) | → 12.5mg | 4 weeks | → Remove |
| CCB (amlodipine 10mg) | → 5mg | 4 weeks | → Remove |
| Beta-blocker | → half dose | 6 weeks | → Remove |
| ACEi/ARB | → one dose step down | 6 weeks + ACR recheck | → NOT full removal (dose-reduce only) |

#### AD-05: Per-Class Monitoring Windows + Failure Thresholds

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_lifecycle.go`

Background job: at monitoring window expiry, check KB-20 `bp_status`:
- AT_TARGET → auto-generate Step 2 (REMOVAL_PROPOSED) card
- ABOVE_TARGET → close deprescribing card, generate HYPERTENSION_REVIEW

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

Step-down monitoring mode:
- Track `pre_step_down_baseline` separately from `bp_target`
- Weekly BP check cadence (Tier-1) during monitoring window

#### AD-06: Failure Threshold Correction

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

Correct interpretation: `failure_threshold = bp_target.sbp + 10` (NOT `baseline + 10`)

Example: patient with target 130, pre-step-down SBP 118:
- SBP rises to 129 after thiazide removal → still below target+10 (140) → NOT a failure
- SBP rises to 142 → exceeds 140 → STEP_DOWN_FAILED

#### AD-07: Re-Escalation Pathway

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-23-decision-cards/internal/services/card_builder.go`

Drug-class-specific re-escalation on STEP_DOWN_FAILED:
| Drug Class | Restart Dose | Card Note |
|------------|-------------|-----------|
| Thiazide (Step 1 failed at half-dose) | Restore full dose | "Not a candidate for reduction now. Re-attempt in 6 months." |
| Thiazide (Step 2 failed after removal) | Restart at 12.5mg (not full) | "Restart at lowest effective dose. Reassess in 90 days." |
| CCB | Restart at lowest effective dose | Surface oedema-vs-BP trade-off if applicable |
| Beta-blocker | Restart at half-dose, taper up | "Rebound tachycardia — restart at lower dose, taper upward." |
| ACEi/ARB | Restore full dose + recheck ACR | "ACR worsening overrides stable BP — restore full RAAS dose." |

#### AD-08: CKD Stage 3b Constraints on Thiazide Removal

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`

For CKD Stage 3b patients:
- Step 1 (dose halving) → allowed
- Step 2 (full removal) → blocked unless `sbp_7d_mean >= 110` at half-dose
- Monitoring windows extended to 6 weeks (not 4) for all drug classes

For CKD Stage 3a:
- Standard hierarchy, but monitoring windows extended to 6 weeks

#### AD-10: ACR Recheck During ACEi/ARB Dose Reduction

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

When ACEi/ARB is in STEP_DOWN MONITORING state:
- Set `acr_check_due = true` at monitoring start
- At 6-week monitoring end: if ACR worsened (category step-up) → STEP_DOWN_FAILED, restore full RAAS dose regardless of BP status

---

## Wave 3 — P2 (Post-GA)

### 3.1 Amendment 7: BP Variability

**Files**:
- `kb-20-patient-profile/internal/models/lab_tracker.go` — add `SBPVisitVariability`, `DBPVisitVariability`, `VariabilityStatus`
- `kb-20-patient-profile/internal/services/lab_service.go` — SD computation over last 5 visits
- `kb-20-patient-profile/internal/models/events.go` — `BP_VARIABILITY_ALERT`
- `kb-23-decision-cards/internal/services/card_builder.go` — variability note in HYPERTENSION_REVIEW

### 3.2 Amendment 10: Chronotherapy

**Files**:
- `kb-20-patient-profile/internal/models/medication_state.go` — add `DoseTiming` enum (MORNING | EVENING | BEDTIME | TWICE_DAILY | WITH_MEALS | UNKNOWN)
- `kb-23-decision-cards/internal/services/card_builder.go` — timing-before-escalation logic: if `bp_pattern = MORNING_SURGE` AND `dose_timing = MORNING` → suggest bedtime switch before dose increase

### 3.3 Amendment 12: Salt Sensitivity in KB-21

**Files**:
- `kb-21-behavioral-intelligence/internal/models/models.go` — add `DietarySodiumEstimate` (LOW | MODERATE | HIGH), `SaltReductionPotential`
- `kb-21-behavioral-intelligence/internal/services/engagement_service.go` — Tier-1 dietary questions (pickles/papads frequency, post-cooking salt, processed food frequency)
- `kb-23-decision-cards/internal/services/card_builder.go` — lifestyle-first sequencing: if HIGH sodium + HIGH reduction potential → dietary intervention card before dose escalation

### 3.4 Amendment 13: Pulse Pressure

**Files**:
- `kb-20-patient-profile/internal/models/lab_tracker.go` — add `PulsePressureMean`, `PulsePressureTrend` (WIDENING | STABLE | NARROWING)
- `kb-20-patient-profile/internal/services/lab_service.go` — derive PP from SBP-DBP over last 5 readings
- `kb-23-decision-cards/internal/services/card_builder.go` — wide PP (>60) note in HYPERTENSION_REVIEW: "further lowering may reduce DBP below safe threshold"

### 3.5 EW-09: Non-Renal Early Damage Markers (v1 KB-22 hooks)

**Files**:
- `kb-22-hpi-engine/` — add `cardiac_strain_suspected` flag when exertional dyspnoea + SEVERE bp_status co-occur
- `kb-22-hpi-engine/` — add `ophthalmology_referral_needed` flag when visual disturbance + ABOVE_TARGET ≥12 weeks
- v2: reserve data fields in KB-20 for LVH (ECG/echo), retinopathy grade, cognitive change

### 3.6 AD-02: Lifestyle Attribution Bonus for Deprescribing Entry

**Files**:
- `kb-20-patient-profile/internal/services/patient_service.go` — if `salt_reduction_potential` changed HIGH→LOW during same period AND bp_status AT_TARGET, reduce 16-week entry window to 12 weeks

---

## Implementation Sequence (Execution Order)

```
WAVE 1 (P0) — Sequential, blocks pilot
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Phase 1A: KB-20 schema extensions
  ├─ VitalFact BP extension fields (measurement_context, posture, arm, time_of_day, etc.)
  ├─ MeasurementUncertainty field on VitalFact
  ├─ bp_pattern enum (NORMAL/WHITE_COAT/MASKED/MORNING_SURGE/NOCTURNAL_NONDIP) in BPTrajectory
  ├─ bp_pattern detection logic in lab_service.go
  ├─ RAAS change recency in medication_service.go
  ├─ Na+ as LabFact (LOINC 2951-2) in lab pipeline — NOT VitalFact
  ├─ Season derivation in patient_service.go
  ├─ Tier-1 question extensions for HTN in KB-22 (oliguria, adherence)
  └─ CKD Stage 4 deprescribing block (AD-09)

Phase 1B: Channel B extensions (can parallel with 1A)
  ├─ raw_inputs.go: RAAS fields + SBPLowerLimit + Na+ + Season + MeasurementUncertainty + BPPattern
  ├─ monitor.go: modify B-03 suppression + add B-12 (J-curve) + uncertainty dampening
  └─ monitor_test.go: all new test cases including uncertainty dampening + white-coat tests

Phase 1C: Channel C + Orchestrator
  ├─ context.go: CreatinineRiseExplained + PG-08–PG-13 fields
  ├─ protocol_rules.yaml: PG-08 through PG-14 (7 HTN rules)
  ├─ vmcu_engine.go: compute RAAS flag + J-curve limit + PG-08–PG-13 context before channels
  └─ guard_test.go: PG-08–PG-14 tests (7 test functions)

Phase 1D: KB-22 Tier-1 HTN questions
  ├─ tier1_htn_questions.yaml (5 questions: oliguria, adherence, cough, orthostatic, palpitations)
  └─ question_router.go: HTN question trigger evaluation

Phase 1E: KB-23 card templates + guards
  ├─ raas_creatinine_monitoring.yaml
  ├─ card_builder.go: RAAS monitoring card generation
  └─ card_builder.go: CKD Stage 4 deprescribing guard

Phase 1F: Integration tests
  ├─ scenario_9_raas_tolerance_test.go (with oliguria override test)
  ├─ scenario_10_jcurve_test.go
  ├─ scenario_27_dual_raas_blockade_halt_test.go
  └─ scenario_28_orthostatic_drop_posture_context_test.go

WAVE 2 (P1) — Can parallelise across services
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

Track A: V-MCU Channel B/C extensions
  ├─ Heart rate rules (B-13 to B-16)
  ├─ Sodium rules (B-17 to B-19)
  ├─ PG-15 (ACEi cough)
  ├─ PG-16 (AF anticoagulation check)
  ├─ KB22_TRIGGER event mechanism (B-16 → KB-19 → KB-22)
  ├─ Beta-blocker perturbation HR threshold adjustment (−5 bpm)
  ├─ V-MCU deprescribing mode (suppress escalation, enforce weekly BP)
  └─ Perturbation window dampening

Track B: KB-20 schema + early warning
  ├─ HR fields in VitalFact
  ├─ ACR longitudinal tracking
  ├─ Resistant HTN detection logic
  ├─ BP measurement uncertainty flag
  ├─ EW-01/02: bp_risk_stratum + per-patient DECLINING threshold
  ├─ EW-03: EARLY_WATCH tier + consecutive_weeks counter
  ├─ EW-05: weeks_to_severe projection + slope_confidence
  └─ EW-07: damage_concern_score composite

Track C: KB-21 behavioural extensions
  ├─ Antihypertensive adherence track
  ├─ FDC-aware adherence (requires KB-1 FDC component lookup)
  └─ Damage composite hysteresis (72h cooldown, all-clear reset)

Track D: KB-22 new HPI nodes
  ├─ p_acei_cough.yaml
  ├─ p_irregular_hr.yaml
  ├─ p_hyperaldosteronism.yaml
  ├─ p_osa_screening.yaml
  └─ p_renal_artery_stenosis.yaml

Track E: KB-23 templates + card logic
  ├─ HYPERTENSION_REVIEW adherence gate
  ├─ ACEi cough ARB switch template
  ├─ Resistant HTN review template (with optimised dose definition from KB-1)
  ├─ ACR worsening/milestone templates
  ├─ Deprescribing ACR veto (drug-class-specific: ACEi/ARB at A2+, SGLT2i at A3)
  ├─ Thiazide perturbation K+ causal context annotation
  ├─ EW-04/06: BP trajectory concern card + projection display + sbp_slope_acceleration note
  └─ EW-08: subclinical concern + damage composite alert cards

Track F: Antihypertensive deprescribing protocol (NEW)
  ├─ AD-01: Adherence pre-condition gate in KB-20
  ├─ AD-03: SGLT2i buffer check in KB-20 (with partial dose reduction handling)
  ├─ AD-04: Dose-halving state machine in KB-23 card_lifecycle.go
  ├─ AD-05: Per-class monitoring windows + background job
  ├─ AD-06: Failure threshold correction (target+10)
  ├─ AD-07: Re-escalation pathway (class-specific restart)
  ├─ AD-08: CKD Stage 3b thiazide removal constraint
  └─ AD-10: ACR recheck during ACEi/ARB dose reduction

Track G: KB-20 early warning extensions
  ├─ sbp_slope_acceleration (second derivative) computation
  ├─ Optimised dose lookup table (from KB-1)
  └─ FDC component mapping integration with KB-1

WAVE 3 (P2) — Post-GA enhancements
━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

  ├─ BP variability tracking + alerts (Amendment 7)
  ├─ Chronotherapy / dose timing (Amendment 10)
  ├─ Salt sensitivity in KB-21 (Amendment 12)
  ├─ Pulse pressure as arterial stiffness marker (Amendment 13)
  ├─ EW-09: Non-renal early damage markers (cardiac strain, ophthalmology referral)
  └─ AD-02: Lifestyle attribution bonus (12-week entry window)
```

---

## Cross-Cutting Concerns

### Database Migrations Required

| Service | Migration | Wave | Content |
|---------|-----------|------|---------|
| KB-20 | `005_htn_extensions.sql` | 1+2 | **Wave 1 columns**: VitalFact BP context fields (measurement_context, posture, arm, time_of_day, consecutive_reading_index, minutes_since_last_activity, white_coat_flag, measurement_uncertainty), bp_pattern enum, Na+ LOINC type, season enum. **Wave 2 columns**: HR fields (heart_rate, hr_regularity, hr_context, hr_confirmed), ACR tracking (acr_readings JSONB, acr_trend, acr_category, acr_on_raas), BP risk stratum, EARLY_WATCH fields, weeks_to_severe, slope_confidence, damage_concern_score JSONB, BP variability, pulse pressure |
| KB-21 | `004_antihypertensive_adherence.sql` | 2 | antihypertensive adherence map (JSONB: drug_class → score), adherence_reason enum (COST, SIDE_EFFECT, FORGOT, SUPPLY, UNKNOWN), dietary sodium estimate, salt_reduction_potential |
| KB-23 | `004_perturbation_extensions.sql` | 2 | expected_direction, magnitude min/max, causal_note columns on treatment_perturbation table |
| KB-23 | `005_deprescribing_state.sql` | 2 | deprescribing_phase enum, deprescribing_drug_class, pre_step_down_dose, current_step_down_dose, monitoring_window_weeks, monitoring_start_date on decision_card table |

**Migration 005 (KB-20) DDL sketch**:
```sql
-- Wave 1 columns (run before pilot)
ALTER TABLE vital_facts ADD COLUMN measurement_context VARCHAR(20) DEFAULT 'UNKNOWN';
ALTER TABLE vital_facts ADD COLUMN posture VARCHAR(10) DEFAULT 'UNKNOWN';
ALTER TABLE vital_facts ADD COLUMN arm VARCHAR(10) DEFAULT 'UNKNOWN';
ALTER TABLE vital_facts ADD COLUMN time_of_day VARCHAR(30) DEFAULT 'UNKNOWN';
ALTER TABLE vital_facts ADD COLUMN consecutive_reading_index INT DEFAULT 0;
ALTER TABLE vital_facts ADD COLUMN minutes_since_last_activity INT;
ALTER TABLE vital_facts ADD COLUMN white_coat_flag BOOLEAN DEFAULT FALSE;
ALTER TABLE vital_facts ADD COLUMN measurement_uncertainty VARCHAR(10) DEFAULT 'LOW';

ALTER TABLE bp_trajectory ADD COLUMN bp_pattern VARCHAR(20) DEFAULT 'NORMAL';
ALTER TABLE bp_trajectory ADD COLUMN bp_risk_stratum VARCHAR(15);
ALTER TABLE bp_trajectory ADD COLUMN sbp_declining_threshold FLOAT;

CREATE TYPE season_enum AS ENUM ('SUMMER', 'MONSOON', 'WINTER', 'AUTUMN');
ALTER TABLE patient_profile ADD COLUMN season season_enum;

-- Wave 2 columns (run before GA)
ALTER TABLE vital_facts ADD COLUMN heart_rate FLOAT;
ALTER TABLE vital_facts ADD COLUMN hr_regularity VARCHAR(10) DEFAULT 'UNKNOWN';
ALTER TABLE vital_facts ADD COLUMN hr_context VARCHAR(15) DEFAULT 'UNKNOWN';
ALTER TABLE vital_facts ADD COLUMN hr_confirmed BOOLEAN DEFAULT FALSE;

ALTER TABLE lab_tracker ADD COLUMN acr_readings JSONB DEFAULT '[]';
ALTER TABLE lab_tracker ADD COLUMN acr_trend VARCHAR(12);
ALTER TABLE lab_tracker ADD COLUMN acr_category VARCHAR(5);
ALTER TABLE lab_tracker ADD COLUMN acr_on_raas BOOLEAN DEFAULT FALSE;

ALTER TABLE bp_trajectory ADD COLUMN early_watch_status VARCHAR(15);
ALTER TABLE bp_trajectory ADD COLUMN consecutive_early_watch_weeks INT DEFAULT 0;
ALTER TABLE bp_trajectory ADD COLUMN early_watch_floor FLOAT;
ALTER TABLE bp_trajectory ADD COLUMN weeks_to_severe FLOAT;
ALTER TABLE bp_trajectory ADD COLUMN slope_confidence VARCHAR(10);
ALTER TABLE bp_trajectory ADD COLUMN sbp_slope_acceleration FLOAT;
ALTER TABLE bp_trajectory ADD COLUMN damage_concern_score JSONB;
ALTER TABLE bp_trajectory ADD COLUMN sbp_visit_variability FLOAT;
ALTER TABLE bp_trajectory ADD COLUMN dbp_visit_variability FLOAT;
ALTER TABLE bp_trajectory ADD COLUMN variability_status VARCHAR(10);
ALTER TABLE bp_trajectory ADD COLUMN pulse_pressure_mean FLOAT;
ALTER TABLE bp_trajectory ADD COLUMN pulse_pressure_trend VARCHAR(12);
```

### TreatmentPerturbation Struct Ownership

The `TreatmentPerturbation` struct is **defined and persisted in KB-23** (`kb-23-decision-cards/internal/models/treatment_perturbation.go`) because perturbations are created as side-effects of decision card actions. However, the struct is **consumed by the V-MCU** (`vmcu_engine.go` → `resolveFromCache`) to dampen Channel B/C evaluations during active perturbation windows.

Ownership boundary:
- **KB-23 writes**: creates, updates, and closes perturbation records
- **V-MCU reads**: queries active perturbations via `PatientSafetyData.ActivePerturbations` in the cache
- **KB-20 does NOT own perturbations** — it only provides the underlying lab/vital data that perturbations reference

### Simulation File Naming Convention

All simulation/integration test files follow the pattern:
```
vaidshala/clinical-runtime-platform/engines/vmcu/simulation/scenario_{N}_{snake_case_description}_test.go
```

| Range | Domain |
|-------|--------|
| 1-8 | Pre-existing DM/CKD scenarios |
| 9-10 | Wave 1 P0 HTN safety (RAAS tolerance, J-curve) |
| 11-23 | Wave 2 P1 HTN scenarios |
| 24-28 | Wave 1/2 HTN additional scenarios (rising trend, urgency, SGLT2i, dual RAAS, orthostatic) |

File names must be descriptive enough to identify the clinical scenario without reading the code. Example: `scenario_27_dual_raas_blockade_halt_test.go`.

### SafetyTrace Extensions

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/trace/safety_trace.go`

Add to `SafetyTrace`:
```go
RAASCausalSuppression       bool    // true if PG-14 downgraded B-03
AppliedSBPLowerLimit        float64 // J-curve threshold used
SourceEGFRForThreshold      float64 // eGFR that produced the threshold
HeartRateEvaluation         string  // B-13/B-14/B-15/B-16 result
SodiumEvaluation            string  // B-17/B-18/B-19 result
MeasurementUncertainty      string  // LOW/MODERATE/HIGH — from VitalFact
UncertaintyDampened         bool    // true if any gate was dampened due to HIGH uncertainty
BPPatternApplied            string  // bp_pattern used for this cycle
HTNProtocolRulesEvaluated   []string // PG-08 through PG-16 rule IDs that fired
DeprescribingMonitoringActive bool  // true if patient in active deprescribing monitoring
DeprescribingDrugClass      string  // which drug class is being deprescribed
BetaBlockerHRThresholdAdjust int   // -5 if perturbation active, 0 otherwise
```

### V-MCU Cache Extensions

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/cache/safety_cache.go`

Add to `PatientSafetyData`:
```go
RAASChangeRecency     *RAASChangeInfo  // from KB-20 medication_service
HeartRateData         *HeartRateInfo   // from KB-20 VitalFact
SodiumCurrent         *float64         // from KB-20 LabFact (LOINC 2951-2)
Season                string           // from KB-20 patient_service
ACRCategory           string           // from KB-20 lab_service
BPPattern             string           // from KB-20 BPTrajectory
MeasurementUncertainty string          // from KB-20 VitalFact (latest)
ActivePerturbations   []TreatmentPerturbation // from KB-23 perturbation_service
DeprescribingActive   bool             // true if any drug in MONITORING state
DeprescribingDrugClass string          // which class is being deprescribed
AntihypertensiveAdherence map[string]float64 // from KB-21
DamageComposite       *DamageComposite // from KB-20 lab_service
SBPSlopeAcceleration  *float64         // from KB-20 BPTrajectory
```

### Event Bus Extensions

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/event_bus.go`

New event types:
- `RESISTANT_HTN_DETECTED`
- `BP_VARIABILITY_ALERT`
- `ACR_WORSENING`
- `ACR_TARGET_MET`
- `RAAS_MONITORING_ESCALATE` (lab result triggers safety re-check during RAAS monitoring)
- `RAAS_MONITORING_RESOLVED` (creatinine stabilised within RAAS tolerance window)
- `BP_TRAJECTORY_CONCERN` (EW-04: EARLY_WATCH threshold exceeded)
- `BP_SUBCLINICAL_CONCERN` (EW-08: damage score 3-4)
- `DAMAGE_COMPOSITE_ALERT` (EW-08: damage score ≥5)
- `KB22_HPI_TRIGGER` (B-16 irregular HR → KB-22 AF differential)
- `DEPRESCRIBING_STEP_DOWN_FAILED` (monitoring window expiry + BP above threshold)
- `DEPRESCRIBING_STEP_COMPLETED` (monitoring window expiry + BP at target)

### Simulation Scenarios

| Scenario | Tests | Amendment |
|----------|-------|-----------|
| 9 | RAAS tolerance (genuine AKI vs expected response) | 1 |
| 10 | J-curve in CKD Stage 3b patient | 8 |
| 11 | Beta-blocker bradycardia during titration | 3 |
| 12 | ACEi cough → adherence drop → ARB switch | 5 |
| 13 | Resistant HTN detection after 12 weeks | 6 |
| 14 | Thiazide hyponatraemia in Indian summer | 11 |
| 15 | ACR worsening despite BP control | 9 |
| 16 | EARLY_WATCH → DECLINING escalation in CKD 3b patient | EW-01/03 |
| 17 | Compound damage score ≥5 fires before Channel B | EW-07/08 |
| 18 | Thiazide dose-halving → monitoring → removal success | AD-04/05 |
| 19 | Thiazide removal STEP_DOWN_FAILED → re-escalation at 12.5mg | AD-07 |
| 20 | Beta-blocker removal → rebound tachycardia → half-dose restart | AD-07 |
| 21 | ACEi/ARB dose reduction → ACR worsening → full dose restore | AD-10 |
| 22 | CKD Stage 4 patient → deprescribing card NOT generated | AD-09 |
| 23 | CKD Stage 3b → thiazide full removal blocked when SBP < 110 at half-dose | AD-08 |
| 24 | BP rising trend: AT_TARGET → EARLY_WATCH → DECLINING over 8 weeks with risk-stratified thresholds | EW-01/02/03 |
| 25 | HTN urgency: SBP > 180 + no organ damage → HYPERTENSION_REVIEW 4h SLA card | Amendment 8 |
| 26 | SGLT2i added → SBP drops 5 mmHg within perturbation window → not flagged as hypotension | 2 |
| 27 | Dual RAAS blockade (ACEi + ARB) → PG-12 HALT fires immediately | 1.5 (PG-12) |
| 28 | Orthostatic drop: standing SBP 25 mmHg below seated → B-05 variant with posture context | 1.4 (VitalFact) |

---

## Implementation Gap Resolutions

### Gap 1: Oliguria Test Case for RAAS Tolerance (B-03)

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor_test.go`

Add test: `TestB03_RAASToleranceWithOliguria`
- ACEi changed 5 days ago, creatinine rise 18% (within 30%), K+ 4.8 (within range), BUT `OliguriaReported = true`
- Expected: HALT (not PAUSE). Oliguria overrides the RAAS tolerance suppression because it signals genuine AKI.

### Gap 2: KB22_TRIGGER Mechanism (already addressed in Section 2.3 item 5)

Documented in Section 2.3 Amendment 3 (Heart Rate), item 5 — the full event flow from B-16 → arbiter → vmcu_engine.go → `KB22_HPI_TRIGGER` event → KB-19 → KB-22 `p_irregular_hr` node.

### Gap 3: FDC (Fixed-Dose Combination) KB-1 Mapping

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-21-behavioral-intelligence/internal/services/adherence_service.go`

FDC adherence logic requires mapping from FDC product to constituent drug classes. This depends on KB-1 Drug Rules:

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-1-drug-rules/` (existing service)

Required lookup: `GetFDCComponents(medicationCode string) → []DrugClass`

Examples:
| FDC Product | Constituent Classes |
|-------------|-------------------|
| Telmisartan/Amlodipine | ARB + CCB |
| Perindopril/Indapamide | ACEi + THIAZIDE |
| Losartan/HCTZ | ARB + THIAZIDE |

**Integration**: KB-21 adherence_service.go calls KB-1 to resolve FDC → classes before computing per-class adherence. Single FDC pill = single adherence unit for all constituent classes.

### Gap 4: "Optimised Dose" Definition for Resistant HTN

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`

The resistant HTN detection (Section 2.5 Amendment 6) requires "3+ drug classes at optimised doses". Define:

```go
// OptimisedDose returns true if the current dose is at or above the evidence-based
// therapeutic target for that drug class in HTN.
type DrugClassDoseTarget struct {
    DrugClass       string
    MinOptimisedDose float64  // mg
    MaxDose          float64  // mg (absolute ceiling)
    Source           string   // guideline reference
}

var HTNDoseTargets = []DrugClassDoseTarget{
    {"ACEi_RAMIPRIL",     10.0,  10.0, "NICE_NG136"},
    {"ARB_LOSARTAN",      100.0, 100.0, "ESH_2023"},
    {"CCB_AMLODIPINE",    10.0,  10.0, "NICE_NG136"},
    {"THIAZIDE_HCTZ",     25.0,  25.0, "NICE_NG136"},
    {"THIAZIDE_CHLORTHALIDONE", 25.0, 50.0, "AHA_2017"},
    {"BETA_BLOCKER_BISOPROLOL", 10.0, 10.0, "ESC_2023"},
    {"MRA_SPIRONOLACTONE", 25.0,  50.0, "PATHWAY2"},
}
```

A drug class is "at optimised dose" when `currentDose >= MinOptimisedDose`. This table is loaded from KB-1 at startup and cached.

### Gap 5: ACR Veto Drug-Class-Specific Granularity

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/titration/deprescribing.go`

Current spec says "ACR A2/A3 vetoes ACEi/ARB deprescribing" — refine:

```go
// ACR-based deprescribing veto rules (drug-class-specific)
func ACRVetoCheck(drugClass string, acrCategory string) (vetoed bool, reason string) {
    switch drugClass {
    case "ACEi", "ARB":
        if acrCategory == "A2" || acrCategory == "A3" {
            return true, "ACR_PROTEINURIA_RAAS_PROTECTIVE"
        }
    case "SGLT2I":
        if acrCategory == "A3" {
            return true, "ACR_A3_SGLT2I_RENOPROTECTIVE"
        }
    case "THIAZIDE", "CCB", "BETA_BLOCKER":
        // No ACR-specific veto for these classes
        return false, ""
    }
    return false, ""
}
```

Key refinement: SGLT2i also gets ACR veto at A3 (severely elevated albuminuria) because of its independent renoprotective effect beyond BP lowering.

### Gap 6: Damage Composite Score Hysteresis Specification

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

Prevent rapid oscillation of damage_concern_score alerts:

```go
type DamageHysteresis struct {
    LastAlertScore       int       // score at which last alert was emitted
    LastAlertTime        time.Time
    CooldownHours        int       // minimum hours between alert re-emissions (default: 72)
    ResetRequiresAllClear bool     // true: score must drop to 0 before re-alerting at same tier
}
```

Rules:
- Score rises from 2 → 5 → emit `DAMAGE_COMPOSITE_ALERT` (24h SLA)
- Score drops to 4 → do NOT emit `BP_SUBCLINICAL_CONCERN` (still above prior alert tier)
- Score drops to 1 → hysteresis resets (all contributing signals normalised)
- Score rises again to 4 → emit `BP_SUBCLINICAL_CONCERN` (new alert cycle)
- Minimum 72h between re-emissions of same alert tier

### Gap 7: V-MCU Behaviour During Active Deprescribing

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go`

When a patient is in active deprescribing (any drug in MONITORING state):

```go
// In RunCycle(), check deprescribing state before titration
if cache.DeprescribingActive {
    // 1. Do NOT propose any new titration UP for any antihypertensive
    // 2. Channel B still evaluates normally (safety never paused)
    // 3. Channel C: suppress escalation-type rules, keep safety rules active
    // 4. If Channel B fires HALT during monitoring → immediately fail deprescribing
    //    (emit STEP_DOWN_FAILED, restore previous dose)
    // 5. Weekly SBP re-check cadence enforced via Tier-1 question T1_HTN_DEPRESCRIBE_BP
}
```

**SafetyTrace addition**: `DeprescribingMonitoringActive bool`, `DeprescribingDrugClass string`

### Gap 8: SGLT2i Partial Dose Reduction Handling

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`

In SGLT2i buffer check (AD-03), distinguish full stop vs partial dose reduction:

```go
func SGLT2iBufferCheck(meds []MedicationFact) (eligible bool, reason string) {
    sglt2i := findActiveMed(meds, "SGLT2I")
    if sglt2i == nil {
        // SGLT2i never prescribed — no buffer issue
        return true, ""
    }
    if sglt2i.Status == "STOPPED" && sglt2i.StopDate.After(time.Now().Add(-6 * 7 * 24 * time.Hour)) {
        return false, "SGLT2I_RECENTLY_STOPPED_BP_BUFFER_LOST"
    }
    if sglt2i.Status == "ACTIVE" && sglt2i.DoseReductionPct > 50 {
        // Dose reduced >50% — treat as partial buffer loss
        // Extend deprescribing entry window by 4 weeks (16 → 20 weeks)
        return false, "SGLT2I_DOSE_REDUCED_GT50_PARTIAL_BUFFER_LOSS"
    }
    return true, ""
}
```

### Gap 9: `sbp_slope_acceleration` Field

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

Add to `BPTrajectory`:
```go
SBPSlopeAcceleration *float64 // second derivative: rate of change of slope (mmHg/week²)
                               // positive = accelerating rise, negative = decelerating
                               // nil = insufficient data (<6 weeks of readings)
```

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

Compute after each BPTrajectory slope update:
```
slope_week_N = sbp_7d_mean at week N - sbp_7d_mean at week N-1
slope_week_N-1 = ... (previous week)
acceleration = slope_week_N - slope_week_N-1
```

Used by EW-05/06 (weeks_to_severe projection):
- If `SBPSlopeAcceleration > 0`: card note amended to "BP rise is accelerating — projection is a minimum estimate"
- If `SBPSlopeAcceleration > 0.5`: promote weeks_to_severe card from 72h SLA to 24h SLA

---

## Risk Register

| Risk | Impact | Mitigation |
|------|--------|------------|
| PG-14 false negative (genuine AKI misclassified as RAAS tolerance) | Patient safety | Upgrade conditions: rise >30% OR K+>5.5 OR oliguria → HALT regardless |
| Perturbation window overlap (SGLT2i + ACEi started simultaneously) | Signal misinterpretation | Log warning when >2 perturbation windows active; do not suppress, only dampen |
| KB-21 adherence data staleness | Dose escalation in non-adherent patient | Define staleness threshold (>21 days → DATA_STALE warning on HYPERTENSION_REVIEW card) |
| Season derivation incorrect for diaspora patients | Wrong seasonal threshold | Use patient-reported location, not system locale; fallback to no seasonal adjustment |
| J-curve threshold too conservative | Undertreated HTN in CKD patients | Clinical team review of thresholds per CKD stage; configurable via PhysioConfig |
| Deprescribing dose-halving fails to detect rebound within monitoring window | Missed BP deterioration | Weekly Tier-1 BP checks + sbp_7d_mean recomputed at day 14 and day 28 |
| weeks_to_severe projection assumes linear trajectory | Underestimates urgency if BP accelerating | Card note: "projection is a minimum estimate" when slope is accelerating; slope_confidence field flags sparse data |
| Damage composite score alert fatigue if signals fluctuate | Physician ignores genuine damage signal | Score resets only when ALL contributing signals normalise; hysteresis prevents rapid oscillation |
| SGLT2i buffer accounting ignores partial SGLT2i contribution (dose reduction vs full stop) | Incorrect deprescribing entry decision | Check SGLT2i dose change, not just stop. If dose reduced >50%, treat as partial buffer loss |

---

## Acceptance Criteria Summary

### Wave 1 (P0)
- [ ] PG-08 fires HALT when ACEi active + K+=5.8 + eGFR declining
- [ ] PG-09 fires MODIFY + safety instruction when beta-blocker + insulin co-prescribed
- [ ] PG-10 fires PAUSE when SGLT2i active + SBP 7d mean < 105
- [ ] PG-11 fires PAUSE when thiazide active + K+ < 3.5
- [ ] PG-12 fires HALT + safety instruction when dual RAAS (ACEi + ARB both active)
- [ ] PG-13 fires MODIFY + safety instruction when HCTZ > 25mg
- [ ] PG-14 fires PAUSE (not HALT) when ACEi increased 5 days ago + creatinine rise 18% + K+ 4.8
- [ ] PG-14 does NOT suppress when creatinine rise is 35% (genuine AKI path)
- [ ] B-03 RAAS tolerance: oliguria present → HALT regardless of rise percentage
- [ ] B-12 fires PAUSE when SBP=98 + eGFR=50 (Stage 3a, limit=100)
- [ ] B-05 still fires HALT when SBP<90 regardless of eGFR stage
- [ ] SafetyTrace records both raw evaluation AND causal suppression
- [ ] RAAS monitoring card generated with Day 7/14/30 protocol
- [ ] RAAS monitoring re-evaluation: `RAAS_MONITORING_ESCALATE` event emitted by lab pipeline at Day 7/14/30
- [ ] VitalFact BP extension fields present: measurement_context, posture, arm, time_of_day, consecutive_reading_index, minutes_since_last_activity, white_coat_flag
- [ ] `bp_pattern` enum (NORMAL | WHITE_COAT | MASKED | MORNING_SURGE | NOCTURNAL_NONDIP) in BPTrajectory
- [ ] `measurement_uncertainty` propagated from VitalFact → Channel B `RawPatientData` → `monitor.go` evaluation
- [ ] CKD Stage 4 patient → `htn_deprescribing_eligible = false` (AD-09)
- [ ] KB-23 does NOT generate antihypertensive DEPRESCRIBING_PROPOSAL for eGFR < 30

### Wave 2 (P1)
- [ ] B-13 fires HALT when HR=42 + RESTING + confirmed
- [ ] B-16 fires PAUSE + KB22_TRIGGER when IRREGULAR + confirmed
- [ ] HYPERTENSION_REVIEW blocked when antihypertensive adherence < 0.60
- [ ] ACEi cough posterior >0.75 → MEDICATION_REVIEW with ARB switch
- [ ] Resistant HTN detected after 12 weeks + 3 drug classes + good adherence
- [ ] ACR A2/A3 vetoes ACEi/ARB deprescribing
- [ ] B-17 fires HALT when Na+ < 132 + thiazide active
- [ ] EARLY_WATCH fires for CKD 3b patient at +1.0 mmHg/week sustained 3 weeks (EW-03)
- [ ] Risk-stratified DECLINING threshold: +1.5 for CKD 3b vs +2.5 for DM-only (EW-02)
- [ ] weeks_to_severe projection shown in HYPERTENSION_REVIEW card body (EW-06)
- [ ] damage_concern_score ≥5 → 24h SLA HYPERTENSION_REVIEW card (EW-08)
- [ ] Deprescribing entry blocked when adherence < 0.85 for 16-week window (AD-01)
- [ ] Deprescribing entry blocked when SGLT2i stopped within 6 weeks (AD-03)
- [ ] Thiazide dose halved → 4-week monitoring → removal proposed (two-action card) (AD-04)
- [ ] Failure threshold at SBP > (bp_target + 10), not (baseline + 10) (AD-06)
- [ ] Beta-blocker STEP_DOWN_FAILED → restart at half-dose, not full (AD-07)
- [ ] ACEi/ARB dose reduction → ACR worsened → full dose restored (AD-10)
- [ ] CKD Stage 3b → thiazide full removal blocked when SBP < 110 at half-dose (AD-08)

### Wave 3 (P2)
- [ ] BP variability SD >15 mmHg → BP_VARIABILITY_ALERT
- [ ] Chronotherapy suggestion before dose escalation for morning surge
- [ ] High dietary sodium → lifestyle card before escalation
- [ ] Wide pulse pressure (>60) → arterial stiffness note in card
- [ ] Cardiac strain flag when exertional dyspnoea + SEVERE bp_status (EW-09)
- [ ] Ophthalmology referral when visual disturbance + ABOVE_TARGET ≥12 weeks (EW-09)
- [ ] Lifestyle attribution bonus: 12-week entry window when salt reduction confirmed (AD-02)

---

## Appendix A: Complete Rule Numbering Reference

### Channel B Physiology Rules (monitor.go)

| Rule | Name | Gate | Wave | Source |
|------|------|------|------|--------|
| B-01 | Glucose hypo | HALT | Existing | DM core |
| B-02 | Glucose hyper | PAUSE | Existing | DM core |
| B-03 | Creatinine 48h delta | HALT (→PAUSE if RAAS tolerance) | Existing + Wave 1 | Amendment 1 |
| B-04 | eGFR absolute floor | HALT | Existing | CKD core |
| B-05 | SBP absolute floor (<90) | HALT | Existing | CKD core |
| B-06 | Weight change | PAUSE | Existing | DM core |
| B-07 | A1C target exceeded | PAUSE | Existing | DM core |
| B-08 | eGFR Stage 5 | HALT | Existing | CKD core |
| B-09 | Potassium >6.0 | HALT | Existing | CKD core |
| B-10 | Potassium <3.0 | HALT | Existing | CKD core |
| DA-01–DA-05 | Diabetes-specific alerts | Various | Existing | DM core |
| B-12 | J-curve eGFR-stratified BP floor | PAUSE/HALT | Wave 1 | Amendment 8 |
| B-13 | Bradycardia HR<45 RESTING confirmed | HALT | Wave 2 | Amendment 3 |
| B-14 | Bradycardia HR<55 + beta-blocker + dose change | PAUSE | Wave 2 | Amendment 3 |
| B-15 | Tachycardia HR>120 RESTING confirmed | PAUSE | Wave 2 | Amendment 3 |
| B-16 | Irregular HR confirmed | PAUSE + KB22_TRIGGER | Wave 2 | Amendment 3 |
| B-17 | Na+ <132 + thiazide | HALT | Wave 2 | Amendment 11 |
| B-18 | Na+ 132-135 + thiazide | PAUSE | Wave 2 | Amendment 11 |
| B-19 | Na+ <135 + SUMMER + thiazide | PAUSE | Wave 2 | Amendment 11 |

### Channel C Protocol Guard Rules (protocol_rules.yaml)

| Rule | Name | Gate | Wave | Source |
|------|------|------|------|--------|
| PG-01 | eGFR rapid decline | HALT | Existing | CKD core |
| PG-02 | A1C target met pause | PAUSE | Existing | DM core |
| PG-03 | SGLT2i eGFR check | MODIFY | Existing | CKD core |
| PG-04 | Metformin eGFR floor | HALT | Existing | DM+CKD |
| PG-05 | Insulin dose ceiling | MODIFY | Existing | DM core |
| PG-06 | Sulphonylurea eGFR | MODIFY | Existing | DM+CKD |
| PG-07 | DPP4i dose adjust | MODIFY | Existing | DM+CKD |
| PG-08 | ACEi/ARB + hyperK + declining eGFR | HALT | Wave 1 | HTN safety |
| PG-09 | Beta-blocker + insulin | MODIFY | Wave 1 | HTN safety |
| PG-10 | SGLT2i + low SBP | PAUSE | Wave 1 | HTN safety |
| PG-11 | Thiazide + hypoK | PAUSE | Wave 1 | HTN safety |
| PG-12 | Dual RAAS blockade | HALT | Wave 1 | HTN safety |
| PG-13 | High-dose HCTZ | MODIFY | Wave 1 | HTN safety |
| PG-14 | RAAS creatinine tolerance | PAUSE | Wave 1 | Amendment 1 |
| PG-15 | ACEi cough safety flag | MODIFY | Wave 2 | Amendment 5 |
| PG-16 | AF anticoagulation check | PAUSE | Wave 2 | Amendment 3 |
