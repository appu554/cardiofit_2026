# HTN Integration — Remaining Gaps Implementation Plan

**Sources**:
- `Vaidshala_HTN_Integration_Proposal.docx` (foundational proposal — §1-10)
- `Vaidshala_HTN_Integration_Amendments.docx` (13 amendments — cross-checked)
- `Vaidshala_HTN_EarlyWarning_Deprescribing.docx` (EW-01..EW-09, AD-01..AD-10)

**Date**: 2026-03-10
**Scope**: Close all remaining gaps identified in the 3-document cross-check. Amendments doc is 100% complete. EarlyWarning doc is 100% complete. This plan covers gaps from the original Proposal + 1 gap from Deprescribing (AD-09).

---

## Priority Legend

| Priority | Meaning | Gate |
|----------|---------|------|
| **P0** | Safety-critical — blocks pilot | Patient harm possible without this |
| **P1** | Clinical completeness — blocks GA | Feature gap, not safety gap |

---

## Wave A — P0 Safety Gaps (2 items)

### A.1 AD-09: CKD Stage 4 Antihypertensive Deprescribing Hard Block

**Why P0**: A near-dialysis patient (eGFR 15-29) has a critically narrow BP window. Proposing antihypertensive deprescribing risks dropping BP below the J-curve floor, causing renal perfusion collapse.

**Source**: Deprescribing doc §2.6, AD-09

#### Step 1: Add Stage 4 exclusion to deprescribing eligibility

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/patient_service.go`

In `EvaluateHTNDeprescribingEligibility()`:
- Before all other checks, add: if current eGFR < 30 (Stage 4/5) → return `eligible: false, reason: "CKD_STAGE_4_BLOCK"`
- This is a hard block — no override, no lifestyle bonus, no exceptions
- Log: "HTN deprescribing blocked: CKD Stage 4 (eGFR < 30)"

#### Step 2: Add Channel C protocol rule AD-09

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/protocol_rules.yaml`

```yaml
- rule_id: AD-09
  description: "CKD Stage 4/5 hard block on antihypertensive deprescribing"
  guideline_ref: "KDIGO_CKD_2021_STAGE4_SAFETY"
  condition:
    field: ckd_stage4_deprescribing_requested
    operator: eq
    value: true
  gate: HALT
  status: active
```

---

### A.2 B-11: Beta-Blocker Glucose HALT Threshold (4.5 instead of 3.9)

**Why P0**: "The single most dangerous drug interaction in the DM+HTN population" (Proposal §1.2). Beta-blockers mask hypoglycaemia warning symptoms. By the time glucose reaches 3.9 in a beta-blocked patient, the autonomic warning window has already closed. Patient can progress to unconsciousness without any warning.

**Source**: Proposal §1.2, Channel B rule B-11

#### Step 1: Add beta-blocker glucose threshold override to Channel B

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/monitor.go`

Add new rule B-11 (insert between existing glucose rules):
```go
// B-11: Beta-blocker + glucose < 4.5 → HALT (raised threshold)
// Beta-blockers mask adrenergic warning symptoms (tachycardia, tremor, palpitations).
// Standard 3.9 threshold is too late — patient may already be neuroglycopaenic.
if raw.BetaBlockerActive && glucose != nil && *glucose < 4.5 && *glucose >= 3.9 {
    signals = append(signals, GateSignal{
        Rule:   "B-11",
        Gate:   HALT,
        Reason: "Beta-blocker active: glucose below 4.5 mmol/L — raised threshold due to suppressed hypoglycaemia warning symptoms",
    })
}
```

**Note**: This fires in the range [3.9, 4.5). Below 3.9, the existing B-01 HALT already fires for all patients.

#### Step 2: Ensure BetaBlockerActive is in RawPatientData

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/channel_b/raw_inputs.go`

Verify `BetaBlockerActive bool` exists (should already be present from Amendment 3 work). If not, add it.

---

## Wave B — P1 KB-20 Extensions (3 items)

### B.1 Add URGENCY and HYPOTENSIVE to BPStatus enum

**Source**: Proposal §3.2

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

Add to `BPStatus` constants:
```go
BPStatusUrgency     BPStatus = "URGENCY"      // Any single clinic reading >= 180 + symptoms
BPStatusHypotensive BPStatus = "HYPOTENSIVE"   // sbp_7d_mean < 100 OR orthostatic_drop < -20
```

### B.2 Add orthostatic_drop field + computation

**Source**: Proposal §3.1

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/lab_tracker.go`

Add to `BPTrajectory`:
```go
OrthostaticDrop     *float64 `json:"orthostatic_drop,omitempty"`       // SBP standing - SBP seated (negative = drop)
LastClinicReading   *float64 `json:"last_clinic_reading_sbp,omitempty"` // Most recent clinic SBP
BPTargetSBP         *float64 `json:"bp_target_sbp,omitempty"`          // Patient-stratum target
BPTargetDBP         *float64 `json:"bp_target_dbp,omitempty"`
```

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/services/lab_service.go`

Add `ComputeOrthostaticDrop(seated, standing float64) float64` — returns standing minus seated.

### B.3 Add missing BP events

**Source**: Proposal §3.3

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-20-patient-profile/internal/models/events.go`

Add event constants:
```go
EventBPAlert            = "BP_ALERT"             // ABOVE_TARGET or DECLINING
EventBPSevereAlert      = "BP_SEVERE_ALERT"      // SEVERE
EventBPUrgencyAlert     = "BP_URGENCY_ALERT"     // URGENCY — immediate notification
EventBPControlled       = "BP_CONTROLLED"         // Transition to AT_TARGET
EventOrthostaticAlert   = "ORTHOSTATIC_ALERT"     // orthostatic_drop < -20
EventMaskedHTNDetected  = "MASKED_HTN_DETECTED"   // bp_pattern = MASKED confirmed
```

Add corresponding payload structs.

---

## Wave C — P1 KB-22 Extensions (2 items)

### C.1 HTN Symptom Differential Nodes (6 nodes)

**Source**: Proposal §4.1

**Directory**: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/nodes/`

Create 6 new YAML nodes:

1. **`htn_headache.yaml`** — Occipital headache (morning, on waking)
   - Differentials: HTN urgency (prior 0.35 if SEVERE, 0.08 if AT_TARGET), migraine, tension headache, medication side effect
   - Bilingual EN/HI, 4-5 discriminating questions

2. **`htn_visual_disturbance.yaml`** — Visual disturbance (blurring, flashing)
   - Differentials: Hypertensive retinopathy, hypoglycaemic visual change, diabetic retinopathy, migraine aura
   - High discriminating value — visual + severe headache = urgency

3. **`htn_ankle_oedema.yaml`** — Bilateral pitting ankle oedema
   - Differentials: CCB side effect (prior 0.65 if on amlodipine), heart failure (0.20 if eGFR declining), renal fluid retention
   - Impacts adherence if CCB-related

4. **`htn_orthostatic_dizziness.yaml`** — Dizziness on standing
   - Differentials: Orthostatic hypotension, hypoglycaemia, dehydration (SGLT2i), medication timing
   - Enriched with KB-20 orthostatic_drop value

5. **`htn_epistaxis.yaml`** — Nosebleed
   - Differentials: HTN-related (prior 0.45 if ABOVE_TARGET), anticoagulant, dry climate
   - Cross-reference with BP status

6. **`htn_facial_flushing.yaml`** — Facial flushing
   - Differentials: CCB side effect (amlodipine), anxiety, menopausal
   - Not dangerous but affects adherence — flag for physician

### C.2 Beta-Blocker Symptom Question Modifier

**Source**: Proposal §4.2

**File**: `backend/shared-infrastructure/knowledge-base-services/kb-22-hpi-engine/nodes/beta_blocker_hypo_modifier.yaml` (NEW)

Create a modifier node that:
- Checks `beta_blocker_active` in session context
- Suppresses palpitation/tremor question weights (near-zero info gain for BB patients)
- Weights higher: diaphoresis, cognitive symptoms, hunger, perioral numbness
- Applied at runtime — same question graph, different information gain values

```yaml
modifier_id: BB_HYPO_SYMPTOM_MODIFIER
version: "1.0"
trigger: beta_blocker_active == true
category: HYPOGLYCAEMIA_SYMPTOM
modifications:
  - question_maps_to: palpitations
    weight_multiplier: 0.05    # suppress — blocked by beta-blocker
  - question_maps_to: tremor
    weight_multiplier: 0.10    # suppress — attenuated by β2 blockade
  - question_maps_to: diaphoresis
    weight_multiplier: 2.5     # most preserved adrenergic symptom
  - question_maps_to: cognitive_symptoms
    weight_multiplier: 3.0     # first presentation in BB patients
  - question_maps_to: hunger
    weight_multiplier: 2.0     # preserved in beta-blockade
  - question_maps_to: perioral_numbness
    weight_multiplier: 2.5     # reliable neuroglycopaenic symptom
reasoning_note: "Beta-blocker active — hypoglycaemia question weights modified per Proposal §4.2"
```

---

## Wave D — P1 V-MCU Fixes (3 items)

### D.1 SGLT2i Deprescribing Block (eGFR <60 OR ACR >30)

**Source**: Proposal §7.1

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/titration/deprescribing.go`

Modify `ACRVetoCheck()` → rename to `DeprescribingVetoCheck()`:
```go
func DeprescribingVetoCheck(drugClass string, acrCategory string, eGFR float64) bool {
    // ACEi/ARB: never deprescribe if ACR A2/A3
    if (drugClass == "ACE_INHIBITOR" || drugClass == "ARB") &&
       (acrCategory == "A2" || acrCategory == "A3") {
        return true
    }
    // SGLT2i: never deprescribe if eGFR <60 OR ACR ≥A2
    // Renal protection persists even when glycaemic control adequate without it
    if drugClass == "SGLT2I" {
        if eGFR < 60 || acrCategory == "A2" || acrCategory == "A3" {
            return true
        }
    }
    return false
}
```

Update all callers of `ACRVetoCheck` to use `DeprescribingVetoCheck` with eGFR parameter.

### D.2 Beta-Blocker Gain Factor Floor 0.50

**Source**: Proposal §8.1

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/titration/engine.go`

In gain factor computation (after existing gain adjustments):
```go
// Beta-blocked patients cannot self-detect early hypoglycaemia.
// More conservative titration is mandatory regardless of adherence score.
if ctx.BetaBlockerActive && gainFactor < 0.50 {
    gainFactor = 0.50
}
```

### D.3 BP-Status Titration Velocity Mapping

**Source**: Proposal §8.1

**File**: `vaidshala/clinical-runtime-platform/engines/vmcu/vmcu_engine.go`

In `RunCycle()`, after Channel A result but before titration:
```go
switch bpStatus {
case "SEVERE":
    // Cardiovascular stress — reduce titration velocity by 30%
    titrationVelocityMultiplier = 0.70
case "URGENCY":
    // Clinical emergency — halt all titration
    arbiterResult.DominantGate = HALT
case "HYPOTENSIVE":
    // Over-treated or haemodynamic instability — pause dose increases
    arbiterResult.DominantGate = PAUSE
case "ABOVE_TARGET", "DECLINING":
    // Suboptimal BP — reduce velocity by 20%
    titrationVelocityMultiplier = 0.80
}
```

---

## Wave E — P1 KB-23 Templates (4 missing subtypes)

### E.1 HYPERTENSION_REVIEW — ABOVE_TARGET

**Source**: Proposal §5.1, §5.4

**New file**: `kb-23-decision-cards/templates/htn_safety/bp_above_target.yaml`

- card_type: HYPERTENSION_REVIEW
- trigger: bp_status == ABOVE_TARGET
- gate: MODIFY (reduce V-MCU titration velocity by 20%)
- SLA: 48h
- Recommendations: Review antihypertensive regimen, consider dose increase, salt restriction education, check K+ before ACEi/ARB increase

### E.2 HYPERTENSION_REVIEW — SEVERE

**New file**: `kb-23-decision-cards/templates/htn_safety/bp_severe.yaml`

- trigger: bp_status == SEVERE
- gate: PAUSE
- SLA: 4h
- Recommendations: Urgent review, check end-organ damage symptoms, secondary HTN screen if new onset, hold insulin dose increases

### E.3 HYPERTENSION_REVIEW — URGENCY

**New file**: `kb-23-decision-cards/templates/htn_safety/bp_urgency.yaml`

- trigger: bp_status == URGENCY
- gate: HALT
- SLA: Immediate (real-time notification)
- Recommendations: Immediate physician contact, no medication titration, same-day review if symptoms present

### E.4 HYPERTENSION_REVIEW — HYPOTENSIVE

**New file**: `kb-23-decision-cards/templates/htn_safety/bp_hypotensive.yaml`

- trigger: bp_status == HYPOTENSIVE
- gate: PAUSE
- SLA: 12h
- Recommendations: Consider antihypertensive dose reduction, educate on positional changes, check orthostatic_drop, review SGLT2i + diuretic combination

---

## Execution Strategy

| Wave | Agent | Services Modified | Estimated Effort |
|------|-------|-------------------|-----------------|
| A (P0) | Agent 1 | KB-20, V-MCU Channel B, Channel C | ~2h |
| B (P1) | Agent 2 | KB-20 models + services | ~3h |
| C (P1) | Agent 3 | KB-22 YAML nodes | ~4h |
| D (P1) | Agent 4 | V-MCU titration + engine | ~3h |
| E (P1) | Agent 5 | KB-23 templates | ~3h |

**Parallelization**: All 5 waves are independent — agents run simultaneously.
**Build verification**: Each agent runs `go build` / `go test` on modified packages.
