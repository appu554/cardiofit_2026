# Replacing Hardcoded Clinical Constants: Architecture Proposal

## The Problem

Modules 7–13 contain **~85 hardcoded clinical constants** scattered across Java source files. These constants encode clinical guideline thresholds (ESH 2023 BP targets, KDIGO 2024 eGFR stages, ADA 2024 glucose targets) that:

1. **Cannot be updated without redeployment.** When KDIGO 2025 changes the SGLT2i eGFR contraindication from 20 to 15, you need a code change, a build, a Flink job restart, and state migration. For a clinical system, this is unacceptable turnaround.

2. **Are population-level when they should be patient-level.** A 45-year-old with isolated HTN and a 78-year-old with CKD G4 + heart failure should not share the same SBP target (130 mmHg). ESH 2023 explicitly recommends age-stratified targets: <130 for 18-69, <140 for ≥70, and even higher floors for CKD G4+ due to J-curve risk.

3. **Are disconnected from the guideline extraction pipeline.** You've built an 8-channel extraction pipeline that produces L3/L5 facts into KB-1, KB-4, KB-16. You've built KB-20's stratum engine that computes patient-specific targets. But the Flink modules don't consume any of this — they use their own copy-pasted numbers.

4. **Create silent clinical drift.** When two guidelines disagree (ADA says FBG target 80-130, RSSDI says <100 for young patients), the hardcoded value picks one silently. There's no audit trail showing which guideline a threshold came from, and no mechanism to apply the Indian guideline for Indian patients.

---

## The Architecture You Already Have (But Don't Use)

Your v4 architecture already solves this in principle. The data flow is:

```
Clinical Guideline PDFs
    ↓ (8-channel extraction pipeline)
KB-1 (Dosing), KB-4 (Safety), KB-16 (Monitoring)
    ↓ (stratum engine at enrollment)
KB-20 (Patient Profile — personalized targets per stratum)
    ↓ (Module 2 enrichment via FHIR/RichAsyncFunction)
Enriched Patient Events (with patient context attached)
    ↓
Flink Modules 7-13 (should read from context, not hardcodes)
```

The gap is the last arrow. Module 2 enriches events with patient context, but Modules 7-13 ignore most of it and use hardcoded constants instead. The M13 spec even calls this out explicitly: `DEFAULT_SBP_TARGET`, `DEFAULT_FBG_TARGET`, `DEFAULT_EGFR_THRESHOLD` are documented as "hardcoded fallbacks used when KB-20 personalised targets aren't available."

The fix is to formalize that pattern — hardcoded fallback with dynamic override — and extend it to every module.

---

## Proposed Design: Three-Layer Threshold Resolution

Every clinical threshold in the pipeline resolves through three layers, checked in order:

### Layer 1: Patient-Specific (KB-20 Stratum Targets)

Computed at enrollment by KB-20's stratum engine based on age, CKD stage, comorbidity profile, guideline region. Attached to the event by Module 2's enrichment phase. Stored in Flink keyed state per patient.

**Example:** Rajesh, 58M, CKD G3b, DM 12yr:
- SBP target: 130 mmHg (ACC/AHA for CKD+DM)
- SBP floor: 105 mmHg (J-curve for G3b, per your KB-20 Amendment 8)
- eGFR decline threshold: 25%/14d becomes 20%/14d (CKD G3b is more sensitive)
- Metformin dose cap: 1000mg (G3b threshold from KDIGO)
- FBG target: 130 mg/dL (ADA for DM with CKD)

### Layer 2: Guideline-Derived Population Defaults (KB-16 Facts)

Extracted by the pipeline from specific guideline PDFs. Versioned and tagged with source (`KDIGO-2024-§4.2`, `ESH-2023-Table-7`). Loaded into Flink as a broadcast stream or lookup table. Updated when a guideline is re-extracted — no Flink restart needed.

**Example:** When no patient-specific stratum is available:
- ARV HIGH threshold: 15.0 mmHg (ESH 2023 — Mena et al. reference)
- Hypertensive crisis SBP: ≥180 mmHg (ACC/AHA/ESH consensus)
- Potassium concern with ACEi: ≥5.3 mEq/L (KDIGO 2024)

### Layer 3: Hardcoded Fallback (Current Constants)

The existing hardcoded values remain as compile-time defaults. They are used only when both Layer 1 and Layer 2 are unavailable (cold start, degraded mode, testing). They carry a `source: HARDCODED_FALLBACK` tag in the audit trail so any clinical decision based on a fallback is traceable.

### Resolution Logic

```java
public class ThresholdResolver {

    /**
     * Resolves a clinical threshold through three layers.
     * Returns the most specific available value with its provenance.
     */
    public ResolvedThreshold resolve(
            String thresholdKey,          // e.g., "sbp_crisis"
            PatientContext patientCtx,    // From Module 2 enrichment (may be null)
            GuidelineDefaults guidelines, // From broadcast stream (may be empty)
            double hardcodedFallback      // Compile-time constant
    ) {
        // Layer 1: Patient-specific override from KB-20 stratum
        if (patientCtx != null) {
            Double patientValue = patientCtx.getThreshold(thresholdKey);
            if (patientValue != null) {
                return new ResolvedThreshold(
                    patientValue,
                    ResolutionLayer.PATIENT_SPECIFIC,
                    patientCtx.getStratumId(),
                    patientCtx.getGuidelineSource()  // e.g., "KDIGO-2024 via KB-20 stratum DM_HTN_CKD_G3b"
                );
            }
        }

        // Layer 2: Guideline-derived population default
        GuidelineFact fact = guidelines.get(thresholdKey);
        if (fact != null) {
            return new ResolvedThreshold(
                fact.getValue(),
                ResolutionLayer.GUIDELINE_DEFAULT,
                null,
                fact.getSource()  // e.g., "ESH-2023-Table-7 extracted 2026-03-15"
            );
        }

        // Layer 3: Hardcoded fallback
        return new ResolvedThreshold(
            hardcodedFallback,
            ResolutionLayer.HARDCODED_FALLBACK,
            null,
            "HARDCODED_V4_INITIAL"
        );
    }
}
```

---

## Module-by-Module Migration Plan

### Module 7: BP Variability Engine (12 constants)

| Current Constant | Value | Layer 1 (Patient-Specific) | Layer 2 (Guideline) | Notes |
|---|---|---|---|---|
| `ARV_THRESHOLD_LOW` | 8.0 | Not personalized — ARV thresholds are population-level | ESH-2023 reference. Extract as KB-16 fact | Keep population-level |
| `ARV_THRESHOLD_HIGH` | 15.0 | Not personalized | ESH-2023 reference | Keep population-level |
| `SBP_CRISIS` | ≥180 | **Yes** — elderly frail patients may crisis at lower SBP. KB-20 stratum `crisis_sbp_threshold` | ACC/AHA consensus | CKD G4+ patients: consider ≥170 |
| `DBP_CRISIS` | ≥120 | Rarely personalized | ACC/AHA | Keep population-level |
| `ACUTE_SURGE_DELTA` | >30 mmHg | Not personalized | ESH-2023 | Keep population-level |
| `MAX_SURGE_LOOKBACK_DAYS` | 1 day | Not personalized | Operational parameter | Keep hardcoded — not clinical |
| `MIN_SURGE_PAIRS` | 3 | Not personalized | Operational parameter | Keep hardcoded |
| `MIN_ARV_READINGS` | 3 | Not personalized | Operational parameter | Keep hardcoded |
| `WHITE_COAT_THRESHOLD` | 15 mmHg | Not personalized | ESH-2023 Table 4 | Extract as KB-16 fact |
| `STATE_TTL` | 31 days | Not personalized | Operational parameter | Keep hardcoded |
| `MIN_DIP_DAYS` | 3 | Not personalized | Operational parameter | Keep hardcoded |

**Migration effort: LOW.** Only `SBP_CRISIS` truly benefits from patient-specific resolution. ARV thresholds and white coat threshold move to Layer 2 (guideline-derived). Operational parameters (min readings, TTL, lookback) stay hardcoded — they're engineering constraints, not clinical thresholds.

**Implementation:** Add `ClinicalThresholds` field to the M7 operator's keyed state. On first event per patient, resolve thresholds. Cache in state. Invalidate when Module 2 sends an updated patient context event (lab result changes CKD stage → new stratum → new thresholds).

### Module 8: Comorbidity Interaction Engine (25+ constants)

This module has the most constants and the highest clinical stakes. Many of its thresholds are directly extractable from guidelines you're already processing.

| Current Constant | Value | Layer 1 | Layer 2 | Notes |
|---|---|---|---|---|
| `POTASSIUM_THRESHOLD` | 5.3 | **Yes** — CKD G4 patients tolerate higher K. Stratum: 5.5 for G4+, 5.3 for others | KDIGO-2024 | Already in your pipeline extraction targets |
| `GLUCOSE_HYPO_THRESHOLD` | 60 | **Yes** — elderly patients: 70 mg/dL. Patients on sulfonylureas: 70 mg/dL. KB-20 stratum | ADA-2024 | Age + medication-stratified |
| `SBP_HYPOTENSION_THRESHOLD` | 95 | **Yes** — CKD J-curve: 100-110 depending on stage. Already in KB-20 Amendment 8 | ACC/AHA + KDIGO | Already designed in KB-20 |
| `eGFR_CRITICAL` | <45 | Partial — SGLT2i threshold varies by specific drug (dapagliflozin <20, empagliflozin <20, canagliflozin <30) | KDIGO-2024 | Drug-specific from KB-1 |
| `EGFR_METFORMIN_LOW` | 30 | Not personalized | KDIGO-2024 / ADA-2024 | Guideline-derived |
| `EGFR_METFORMIN_HIGH` | 35 | Not personalized | KDIGO-2024 | Guideline-derived, dose-reduce zone |
| `EGFR_DROP_THRESHOLD_PCT` | 15% | **Yes** — CKD G3b: 10% drop is significant. G1-2: 15% is appropriate | KDIGO-2024 | Stage-stratified |
| `FBG_DELTA_THRESHOLD` | 15 mg/dL | **Yes** — tighter for young DM patients (10 mg/dL), looser for elderly (20 mg/dL) | ADA-2024 / RSSDI | Age-stratified |
| `WEIGHT_DROP_THRESHOLD` | 2.0 kg/7d | Partial — GLP-1RA patients expect weight loss. Stratum: 3.0 for GLP-1RA, 2.0 for others | Clinical practice | Medication-stratified |
| `ELDERLY_AGE_THRESHOLD` | 75 | Region-dependent — Indian guidelines (RSSDI) may use 70. KB-16 fact | RSSDI vs ADA | Guideline-region dependent |
| `SBP_TARGET_INTENSIVE` | 130 | **Yes** — Age-stratified: <130 for 18-69, <140 for ≥70. CKD G4+: <140. Already in KB-20 | ESH-2023 / ACC/AHA | Core personalization target |
| `HALT_DEDUP_WINDOW` | 4 hours | Not personalized | Operational | Keep hardcoded |

**Migration effort: HIGH.** This is the module that benefits most from personalization. The eGFR thresholds alone have 6+ constants that vary by drug (from KB-1) and CKD stage (from KB-20). The FBG and SBP targets are age-stratified. The potassium threshold is CKD-stage-stratified.

**Implementation:** M8 already processes enriched events from Module 2 which should carry patient context. The rule engine should resolve thresholds at rule evaluation time, not at compile time. Each CID rule becomes:

```java
// Before (hardcoded):
if (potassium >= 5.3 && hasACEiOrARB()) { fireAlert(CID_02); }

// After (resolved):
double kThreshold = resolver.resolve("potassium_concern_acei",
    patientCtx, guidelines, 5.3 /* fallback */);
if (potassium >= kThreshold.value() && hasACEiOrARB()) {
    ClinicalAlert alert = fireAlert(CID_02);
    alert.setThresholdProvenance(kThreshold.provenance());
    // Audit trail: "K=5.2 vs threshold 5.5 (KDIGO-2024 via KB-20 stratum CKD_G4)"
}
```

**Critical detail:** When a threshold resolution changes the clinical decision (e.g., K=5.2 would fire at the hardcoded 5.3 but not at the patient-specific 5.5 for CKD G4), the provenance must be surfaced to the physician. They need to see WHY the alert didn't fire, not just that it didn't.

### Module 9: Engagement Monitor (11 constants)

| Constant | Personalize? | Notes |
|---|---|---|
| `ZOMBIE_THRESHOLD_DAYS` | No | Operational |
| `SUSTAINED_LOW_THRESHOLD_DAYS` | Partial — new patients may need longer baseline (7 days) | Age/tech-literacy |
| `CLIFF_DROP_THRESHOLD` | No | Statistical threshold |
| `LEVEL_TRANSITION_PERSISTENCE_DAYS` | No | Operational |
| `W_STEPS, W_MEAL, W_LATENCY, W_CHECKIN, W_PROTEIN` | **Yes** — weights should depend on patient's data tier. TIER_1_CGM patients have richer signals, TIER_3_SMBG patients don't have steps/meal data | Data tier from Module 1b |
| `STEP_NORMALIZATION` | **Yes** — 10,000 steps unrealistic for elderly/CKD/wheelchair-bound. KB-20 stratum: mobility-adjusted target | Patient profile |
| `SESSION_NORMALIZATION` | No | Operational |

**Migration effort: MEDIUM.** The engagement weights are the key personalization: a patient without CGM shouldn't have their engagement score penalized for missing CGM-dependent signals. The step normalization needs patient-level adjustment (a 78-year-old's 3,000 steps is excellent engagement; a 35-year-old's 3,000 is below target).

### Module 10/10b: Meal Response + Patterns (8 constants)

| Constant | Personalize? | Notes |
|---|---|---|
| `FLAT_THRESHOLD` (20 mg/dL) | **Yes** — diabetic patients on insulin may have artificially flat curves. T1DM vs T2DM | KB-20 diabetes type |
| `TOP_FOODS` (5) | No | Operational |
| Curve shape thresholds | Partial | Population-level with possible age adjustment |

**Migration effort: LOW.** Most M10 constants are pattern classification parameters that work at population level. The flat response threshold is the main candidate for personalization.

### Module 11/11b: Activity Response + Fitness (8 constants)

| Constant | Personalize? | Notes |
|---|---|---|
| `HYPOGLYCEMIA_THRESHOLD` (70) | **Yes** — same as M8's glucose threshold. Elderly: 80 mg/dL | ADA-2024, age-stratified |
| `DEFAULT_RESTING_HR` (72) | **Yes** — beta-blocker patients have lower resting HR. KB-20 medication list | Medication-aware |
| `MIN_EFFORT_FRACTION` (0.60) | **Yes** — CKD/heart failure patients: 0.50. Elderly: 0.50 | KB-20 stratum |
| `WHO_MODERATE_MINIMUM` (150) | Partial — RSSDI recommends different targets for Indian diabetics | Guideline-region |
| HRR targets | No | Population-level exercise physiology |

**Migration effort: MEDIUM.** Resting HR and effort fraction benefit strongly from personalization. A beta-blocker patient's "resting HR" of 60 bpm is normal, not concerning.

### Module 12/12b: Intervention Window + Delta (5 constants)

| Constant | Personalize? | Notes |
|---|---|---|
| `MIN_READINGS` (3) | No | Operational |
| `MIN_SPAN_MS` (7 days) | Partial — some drug classes (fast-acting antihypertensives) show effect in 3-5 days | Drug-class from KB-1 |
| `OVERLAP_THRESHOLD_MS` (7 days) | No | Operational |
| `DEFAULT_ADHERENCE` (0.50) | **Yes** — patients with Module 9 engagement data should use actual adherence, not default | M9 output |
| Observation window | Already dynamic — comes from intervention event | Already solved |

**Migration effort: LOW.** `DEFAULT_ADHERENCE` is the key — this should consume actual Module 9 engagement data rather than assuming 50%.

### Module 13: Clinical State Synchroniser (18 constants)

This module already documents the "hardcoded fallback + KB-20 override" pattern. It just hasn't implemented it.

| Constant | Personalize? | Notes |
|---|---|---|
| `DEFAULT_FBG_TARGET` (110) | **Yes** — already documented as KB-20 override. ADA: 80-130 range. RSSDI: <100 for young | KB-20 stratum |
| `DEFAULT_SBP_TARGET` (130) | **Yes** — age-stratified. Already in KB-20 | KB-20 stratum |
| `DEFAULT_EGFR_THRESHOLD` (45) | **Yes** — CKD stage determines decline significance | KB-20 stratum |
| `COMPOSITE_DETERIORATING_THRESHOLD` (0.40) | No | Statistical threshold |
| `COMPOSITE_IMPROVING_THRESHOLD` (-0.30) | No | Statistical threshold |
| `AMPLIFICATION_FACTOR` (1.5) | No | Model parameter |
| `SNAPSHOT_ROTATION_INTERVAL_MS` (7d) | No | Operational |
| `CRITICAL_ABSENCE_MS` (14d) | Partial — elderly patients: 7d absence is critical | Patient profile |
| `ENGAGEMENT_COLLAPSE_DELTA` (0.35) | No | Statistical threshold |
| `HIGH_PRIORITY_CONFIDENCE_THRESHOLD` (0.50) | No | Quality gate |
| `DEDUP_WINDOW_MS` (24h) | No | Operational |

**Migration effort: MEDIUM.** The three `DEFAULT_*` targets are the priority — they directly affect the velocity computation that drives clinical decisions. M13's entire CKM velocity is distorted when using population-level targets for patients at the extremes.

---

## Implementation Strategy

### Phase 1: Infrastructure (1 week)

Build `ThresholdResolver`, the `GuidelineDefaults` broadcast stream, and the `ResolvedThreshold` record with provenance. These are shared components used by all modules.

**GuidelineDefaults broadcast stream:** A Kafka topic (`clinical.guideline-thresholds`) that carries versioned threshold facts extracted by the pipeline. Each fact has a key (e.g., `arv_threshold_high`), a value, a source reference, and a version. Flink consumes this as a broadcast stream and joins it to keyed patient streams. When a guideline is re-extracted, new facts are published and all Flink jobs pick them up without restart.

```java
// Broadcast stream for guideline-derived thresholds
BroadcastStream<GuidelineFact> guidelineStream = env
    .addSource(new FlinkKafkaConsumer<>("clinical.guideline-thresholds", ...))
    .broadcast(guidelineStateDescriptor);

// Join with keyed patient stream
patientStream
    .connect(guidelineStream)
    .process(new ThresholdAwareBroadcastFunction());
```

**PatientContext propagation:** Ensure Module 2's enrichment attaches KB-20 stratum thresholds to the enriched event. Downstream modules read `event.getPatientContext().getThreshold("sbp_crisis")` instead of `SBP_CRISIS_CONSTANT`.

### Phase 2: Module 8 Migration (1 week)

Module 8 first because it has the highest clinical stakes and the most constants that benefit from personalization. The CID rule engine is refactored to resolve thresholds per-evaluation rather than per-compilation.

**Validation:** Run the E2E 14-day dataset with both old (hardcoded) and new (resolved) paths. Every clinical decision must match for the population-level case. For Rajesh (CKD G3b), some thresholds will change — document each difference as a clinical improvement.

### Phase 3: Module 13 Migration (1 week)

Module 13 is second because it already documents the pattern and its three `DEFAULT_*` targets are the most impactful. The CKM velocity computation starts using patient-specific targets from KB-20, which immediately fixes the "velocity computed against wrong baseline" problem.

### Phase 4: Remaining Modules (1 week)

M7, M9, M10, M11, M12 — each has fewer personalization-eligible constants. Most constants in these modules are operational parameters or population-level statistical thresholds that should stay hardcoded.

### Phase 5: Guideline Pipeline Integration (ongoing)

Populate `clinical.guideline-thresholds` topic from the extraction pipeline output. Map each extracted KB-16 fact to a threshold key. Build the CQL guideline registry cross-reference so extracted thresholds are validated against existing CQL defines before being published.

---

## What Should NOT Be Externalized

Not every constant should move to dynamic resolution. Constants that should stay hardcoded:

**Operational parameters:** State TTLs, minimum reading counts, window durations, dedup intervals, normalization factors. These are engineering decisions, not clinical thresholds. Changing them requires understanding Flink state management implications, not clinical guideline updates.

**Statistical model parameters:** Composite deteriorating threshold (0.40), amplification factor (1.5), cliff drop threshold (0.30). These are tuned against training data or clinical validation, not derived from guidelines. They change when the model is retrained, not when a guideline updates.

**Absolute safety floors:** SBP <90 for hypotension, glucose <54 for severe hypoglycemia. These are physiological constants that don't vary by guideline or patient. A systolic of 85 is dangerous for everyone.

**Rule:** If a constant comes from a clinical guideline and is cited with a guideline reference → externalize. If a constant is an engineering/model parameter → keep hardcoded. If a constant is a physiological absolute → keep hardcoded.

---

## The Audit Trail Contract

Every clinical decision produced by Modules 7-13 must carry provenance for every threshold that influenced it. The output schema gains:

```json
{
  "alert_id": "...",
  "rule_id": "CID_02",
  "fired": true,
  "thresholds_used": [
    {
      "key": "potassium_concern_acei",
      "value": 5.5,
      "layer": "PATIENT_SPECIFIC",
      "source": "KDIGO-2024 via KB-20 stratum CKD_G4",
      "stratum_id": "DM_HTN_CKD_G4",
      "patient_value": 5.2,
      "would_fire_at_default": false,
      "default_value": 5.3
    }
  ]
}
```

The `would_fire_at_default` field is critical — it lets clinical reviewers identify cases where personalization changed the outcome. This is both a safety check and a research dataset for validating that personalized thresholds improve clinical outcomes.

---

## Summary: What Moves Where

| Category | Count | Action |
|---|---|---|
| Patient-specific (KB-20 stratum) | ~18 | Layer 1 — resolve from patient context |
| Guideline-derived (KB-16 facts) | ~22 | Layer 2 — broadcast stream from pipeline |
| Operational parameters | ~30 | Stay hardcoded — not clinical |
| Statistical/model parameters | ~8 | Stay hardcoded — model-tuned |
| Absolute safety floors | ~7 | Stay hardcoded — physiological |
| **Total** | **~85** | **~40 externalized, ~45 stay hardcoded** |

The net result: roughly half the constants externalize into the two-layer resolution system. The other half stay where they are because they're engineering or physiological constants that shouldn't change with guideline updates. Every clinical decision carries provenance showing which layer supplied each threshold.
