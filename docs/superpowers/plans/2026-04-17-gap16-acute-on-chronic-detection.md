# Gap 16: Acute-on-Chronic Deterioration Detection — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build per-patient baseline deviation detection in KB-26 that catches acute deterioration (AKI, fluid overload, BP crisis, compound cardiorenal syndrome) within minutes of data arrival — using individualized 7-day rolling baselines, not population thresholds.

**Architecture:** Three components in KB-26: (1) PatientBaseline engine computes per-patient 7-day rolling median + MAD for each vital sign, with 14-day fallback for sparse data; (2) DeviationScorer computes deviation from baseline with directional rules, gap amplification, and confounder dampening; (3) CompoundPatternDetector matches multi-vital-sign deviation templates (cardiorenal syndrome, infection cascade, medication crisis). An AcuteEventHandler orchestrates the pipeline on each new reading: baseline → deviation → compound → persist → PAI update → escalation publish. KB-23 gets 6 acute-specific card templates. Flink Tier 1 (CGM streaming) is deferred to Sprint 2.

**Tech Stack:** Go 1.21 (Gin, GORM) for KB-26. PostgreSQL for baselines + acute events. Kafka for event publishing. YAML market configs for thresholds + compound patterns. Existing PAI + escalation engine for downstream action.

---

## Scope — Sprint 1 vs Sprint 2

| Component | Sprint | Rationale |
|-----------|--------|-----------|
| Per-patient baseline engine | **Sprint 1** | Foundation for all detection |
| Deviation scorer (6 vital signs) | **Sprint 1** | Core detection logic |
| Compound pattern detector (5 patterns) | **Sprint 1** | Cross-domain differentiator |
| KB-26 acute event handler | **Sprint 1** | Orchestrates pipeline |
| KB-23 acute card templates (6) | **Sprint 1** | Clinician-facing output |
| Market config YAML (4 files) | **Sprint 1** | Threshold definitions |
| API handlers + wiring | **Sprint 1** | Query + trigger endpoints |
| Flink Module14 CGM streaming | **Sprint 2** | Requires Flink operator work, independent of Tier 2 |

## File Inventory

### KB-26 — Acute Detection Engine
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/acute_event.go` | AcuteEvent, PatientBaselineSnapshot, DeviationResult, CompoundPatternMatch |
| Create | `internal/services/patient_baseline.go` | ComputeBaseline (7d median + MAD), RefreshBaseline, BaselineConfidence |
| Create | `internal/services/patient_baseline_test.go` | 6 tests |
| Create | `internal/services/deviation_scorer.go` | ComputeDeviation (directional, gap amplification, confounder dampening) |
| Create | `internal/services/deviation_scorer_test.go` | 8 tests |
| Create | `internal/services/compound_pattern_detector.go` | DetectCompoundPatterns (5 syndromes from YAML) |
| Create | `internal/services/compound_pattern_detector_test.go` | 7 tests |
| Create | `internal/services/acute_event_handler.go` | HandleNewReading pipeline + resolution tracking |
| Create | `internal/services/acute_event_handler_test.go` | 6 tests |
| Create | `internal/services/acute_repository.go` | Persist + query acute events + baselines |
| Create | `internal/api/acute_handlers.go` | GET /acute/:patientId, POST /acute/:patientId/reading |
| Modify | `internal/api/routes.go` | Add acute route group |
| Modify | `main.go` | AutoMigrate + wire services |

### KB-23 — Acute Card Templates
| Action | File | Responsibility |
|---|---|---|
| Create | `templates/acute/acute_kidney_injury.yaml` | SAFETY card for eGFR acute drop |
| Create | `templates/acute/fluid_overload.yaml` | SAFETY/URGENT card for HF weight gain |
| Create | `templates/acute/hypertensive_emergency.yaml` | SAFETY card for SBP crisis |
| Create | `templates/acute/severe_hypoglycaemia.yaml` | SAFETY card for sustained low glucose |
| Create | `templates/acute/compound_cardiorenal.yaml` | SAFETY card for multi-organ deterioration |
| Create | `templates/acute/medication_induced_crisis.yaml` | IMMEDIATE card for drug-induced adverse effect |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/acute_detection_thresholds.yaml` | Per-vital-sign deviation thresholds, baseline params, gap amplification |
| Create | `market-configs/shared/compound_patterns.yaml` | 5 compound syndrome definitions |
| Create | `market-configs/india/acute_overrides.yaml` | Heat-season eGFR sensitivity, festival glucose, rural gaps |
| Create | `market-configs/australia/acute_overrides.yaml` | Indigenous eGFR sensitivity, aged care weight, disaster mode |

**Total: 25 files (22 create, 3 modify), ~50 tests**

---

### Task 1: Acute models + threshold YAML configs

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/models/acute_event.go`
- Create: `market-configs/shared/acute_detection_thresholds.yaml`
- Create: `market-configs/shared/compound_patterns.yaml`
- Create: `market-configs/india/acute_overrides.yaml`
- Create: `market-configs/australia/acute_overrides.yaml`

- [ ] **Step 1:** Create `acute_event.go` with 4 types: `AcuteEvent` (GORM model: ID, PatientID, DetectedAt, EventType, Severity, DeviationDetails JSON, BaselineValues JSON, CurrentValues JSON, DeviationPercent, CompoundPattern, MedicationContext, ConfounderContext, EscalationTier, SuggestedAction, ResolvedAt, ResolutionType), `PatientBaselineSnapshot` (GORM: PatientID, VitalSignType, BaselineMedian, BaselineMAD, ReadingCount, Confidence, ComputedAt, LookbackDays), `DeviationResult` (value type: VitalSignType, CurrentValue, BaselineMedian, BaselineMAD, DeviationAbsolute, DeviationPercent, Direction, ClinicalSignificance, GapAmplified, ConfounderDampened), `CompoundPatternMatch` (value type: PatternName, MatchedDeviations, PatternConfidence, ClinicalSyndrome, RecommendedResponse).

- [ ] **Step 2:** Create `acute_detection_thresholds.yaml` with: baseline params (primary_lookback_days: 7, fallback_lookback_days: 14, min_readings_high_confidence: 7, min_readings_any: 3), per-vital-sign thresholds (eGFR: CRITICAL >30% drop, HIGH >25%, MODERATE >20%; SBP: CRITICAL >180 absolute OR >40 above baseline; weight: CRITICAL >3kg/72h CKM 4c; potassium: CRITICAL >6.0; glucose CGM: CRITICAL sustained >300 for >6h), gap amplification (threshold_hours: 48, amplify_levels: 1), confounder dampening (enabled: true, dampen_levels: 1, min_floor: MODERATE).

- [ ] **Step 3:** Create `compound_patterns.yaml` with 5 patterns: CARDIORENAL_SYNDROME (eGFR drop ≥15% + SBP drop ≥15 OR weight gain ≥1.5kg), INFECTION_CASCADE (glucose rise ≥30% + SBP drop ≥15% + measurement freq drop ≥50%), MEDICATION_CRISIS (any deviation + new med within 14d), FLUID_OVERLOAD_TRIAD (weight gain ≥1.5kg + SBP rise ≥10 + CKM 4c), POST_DISCHARGE_DETERIORATION (any MODERATE deviation within 30d of discharge).

- [ ] **Step 4:** Create India + Australia override YAMLs.

- [ ] **Step 5:** Verify compile + YAML parse. Commit: `feat(kb26): acute-on-chronic models + threshold configs (Gap 16 Task 1)`

---

### Task 2: Per-patient baseline engine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/patient_baseline.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/patient_baseline_test.go`

- [ ] **Step 1:** Write 6 tests:
1. `TestBaseline_SufficientData_7DayMedian` — 10 readings over 7 days → median computed, confidence HIGH
2. `TestBaseline_SparseData_14DayFallback` — 4 readings in 7 days, 8 in 14 days → uses 14-day window, confidence MODERATE
3. `TestBaseline_InsufficientData_LowConfidence` — 2 readings total → confidence LOW
4. `TestBaseline_MAD_Computation` — known values [100, 102, 98, 105, 97] → verify MAD matches expected
5. `TestBaseline_Refresh_NewReading` — existing baseline + new reading → median shifts appropriately
6. `TestBaseline_IdenticalReadings_ZeroMAD` — all readings same value → MAD=0, handled gracefully

- [ ] **Step 2:** Implement `ComputeBaseline(readings []float64, timestamps []time.Time, lookbackDays int) PatientBaselineSnapshot` — pure function computing median + MAD. Implement `RefreshBaseline(existing PatientBaselineSnapshot, newReading float64, newTimestamp time.Time, allReadings []float64) PatientBaselineSnapshot`.

- [ ] **Step 3:** Run tests — all 6 pass. Commit: `feat(kb26): per-patient baseline engine — 7d median + MAD (Gap 16 Task 2)`

---

### Task 3: Deviation scorer

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/deviation_scorer.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/deviation_scorer_test.go`

- [ ] **Step 1:** Write 8 tests:
1. `TestDeviation_EGFR_25PercentDrop_High` — baseline 40, reading 30 (25% drop) → HIGH
2. `TestDeviation_EGFR_WithSteroidConfounder_Dampened` — same drop + steroid active → dampened to MODERATE
3. `TestDeviation_SBP_Spike40_Critical` — baseline 130, reading 172 (+42) → CRITICAL
4. `TestDeviation_SBP_AfterMeasurementGap_Amplified` — +32 mmHg spike after 72h gap → amplified HIGH→CRITICAL
5. `TestDeviation_Weight_2_5kg_CKM4c_Critical` — 2.5kg gain/72h in HF patient → CRITICAL
6. `TestDeviation_Weight_2_5kg_CKM2_Moderate` — same gain, no HF → MODERATE
7. `TestDeviation_LowConfidenceBaseline_WidenedThreshold` — LOW confidence baseline → 37.5% threshold instead of 25%
8. `TestDeviation_EGFR_Rise_NoAlert` — eGFR rises 20% above baseline → no alert (rises are good)

- [ ] **Step 2:** Implement `ComputeDeviation(currentValue float64, baseline PatientBaselineSnapshot, vitalType string, config *AcuteDetectionConfig, context DeviationContext) DeviationResult` where `DeviationContext` carries CKMStage, ActiveConfounders, HoursSinceLastReading, IsPostDischarge.

- [ ] **Step 3:** Run tests — all 8 pass. Commit: `feat(kb26): deviation scorer — directional + gap amplification + confounder dampening (Gap 16 Task 3)`

---

### Task 4: Compound pattern detector

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/compound_pattern_detector.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/compound_pattern_detector_test.go`

- [ ] **Step 1:** Write 7 tests:
1. `TestCompound_CardiorenalSyndrome_Matched` — eGFR drop 18% + SBP drop 20 mmHg → CARDIORENAL_SYNDROME
2. `TestCompound_CardiorenalSyndrome_SingleDeviation_NoMatch` — only eGFR drops → no compound
3. `TestCompound_InfectionCascade_Matched` — glucose +35% + SBP -18% + freq drop 60% → INFECTION_CASCADE
4. `TestCompound_MedicationCrisis_NSAIDPlusEGFR` — new NSAID 5d ago + eGFR drop 20% → MEDICATION_CRISIS
5. `TestCompound_FluidOverload_CKM4cOnly` — weight +2kg + SBP +15 in CKM 4c → match; CKM 2 → no match
6. `TestCompound_PostDischarge_Amplifies` — MODERATE eGFR drop within 30d of discharge → amplified to HIGH
7. `TestCompound_BelowMinThresholds_NoMatch` — deviations all below MODERATE → no pattern

- [ ] **Step 2:** Implement `DetectCompoundPatterns(deviations []DeviationResult, context CompoundContext) []CompoundPatternMatch` where CompoundContext carries CKMStage, DaysSinceDischarge, NewMedications, MeasurementFreqDrop. Load pattern definitions from config.

- [ ] **Step 3:** Run tests — all 7 pass. Commit: `feat(kb26): compound pattern detector — 5 multi-organ syndromes (Gap 16 Task 4)`

---

### Task 5: Acute event handler — orchestrator pipeline

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/acute_event_handler.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/acute_event_handler_test.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/acute_repository.go`

- [ ] **Step 1:** Write 6 tests:
1. `TestHandler_EGFRBelowThreshold_CreatesEvent` — eGFR 25% drop → AcuteEvent with type ACUTE_KIDNEY_INJURY, severity HIGH
2. `TestHandler_NormalReading_NoEvent` — reading within baseline → no event
3. `TestHandler_CompoundTriggered` — eGFR drop + SBP drop within 72h → CARDIORENAL_SYNDROME event
4. `TestHandler_Resolution_MarksResolved` — recovery reading after acute event → previous event marked RESOLVED
5. `TestHandler_CriticalSeverity_SafetyEscalation` — CRITICAL event → EscalationTier = SAFETY
6. `TestHandler_ConfounderDampening_Applied` — deviation during seasonal window → severity dampened

- [ ] **Step 2:** Implement `AcuteEventHandler.HandleNewReading(patientID, vitalType string, value float64, timestamp time.Time) (*AcuteEvent, error)` — the full pipeline: fetch baseline → compute deviation → check recent deviations for compounds → persist event → update PAI → publish to Kafka/escalation.

- [ ] **Step 3:** Implement `AcuteRepository` — SaveEvent, FetchActive, FetchRecentDeviations (last 72h), MarkResolved, SaveBaseline, FetchBaseline.

- [ ] **Step 4:** Run tests — all 6 pass. Commit: `feat(kb26): acute event handler — full pipeline + resolution tracking (Gap 16 Task 5)`

---

### Task 6: KB-23 acute card templates

**Files:**
- Create 6 YAML templates in `kb-23-decision-cards/templates/acute/`

- [ ] **Step 1:** Create all 6 templates following the existing card template pattern: acute_kidney_injury.yaml (HALT gate, SAFETY), fluid_overload.yaml (HALT for >3kg CKM 4c, MODIFY otherwise), hypertensive_emergency.yaml (HALT gate), severe_hypoglycaemia.yaml (HALT gate), compound_cardiorenal.yaml (HALT gate), medication_induced_crisis.yaml (MODIFY gate). Each has CLINICIAN + PATIENT fragments with {{.CurrentValue}}, {{.BaselineValue}}, {{.DeviationPercent}} placeholders.

- [ ] **Step 2:** Verify YAML parses. Commit: `feat(kb23): acute-on-chronic card templates — 6 syndromes (Gap 16 Task 6)`

---

### Task 7: API handlers + wiring + integration test

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/api/acute_handlers.go`
- Modify: `kb-26-metabolic-digital-twin/internal/api/routes.go`
- Modify: `kb-26-metabolic-digital-twin/main.go`

- [ ] **Step 1:** Create handlers: GET `/acute/:patientId` (returns active acute events), GET `/acute/:patientId/baselines` (returns current baselines), POST `/acute/:patientId/reading` (triggers detection for a new reading — body: {vital_type, value, timestamp}).

- [ ] **Step 2:** Add routes, AutoMigrate AcuteEvent + PatientBaselineSnapshot, wire handler into server.

- [ ] **Step 3:** Full test sweep KB-26 + KB-23. Verify YAML. Commit: `feat: complete Gap 16 acute-on-chronic detection`

- [ ] **Step 4:** Push to origin.

---

## Verification Questions

1. Does a 25% eGFR drop from 7-day baseline trigger HIGH severity? (yes / test)
2. Does the same drop with active steroid confounder get dampened to MODERATE? (yes / test)
3. Does a SBP spike after 72h measurement gap get amplified? (yes / test)
4. Does weight gain only score CRITICAL for CKM 4c patients? (yes / test)
5. Does LOW confidence baseline widen thresholds by 50%? (yes / test)
6. Does eGFR drop + SBP drop trigger CARDIORENAL_SYNDROME compound? (yes / test)
7. Does a single deviation without compound partner NOT trigger compound? (yes / test)
8. Does a recovery reading resolve the previous acute event? (yes / test)
9. Are all KB-26 + KB-23 test suites green? (yes / sweep)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML configs | 5 files | 1-2 hours |
| Task 2: Baseline engine (6 tests) | 2 files | 2-3 hours |
| Task 3: Deviation scorer (8 tests) | 2 files | 2-3 hours |
| Task 4: Compound detector (7 tests) | 2 files | 2-3 hours |
| Task 5: Event handler (6 tests) | 3 files | 3-4 hours |
| Task 6: Card templates | 6 YAML | 1 hour |
| Task 7: API + wiring + integration | 3 files | 2-3 hours |
| **Total** | **~25 files, ~50 tests** | **~14-20 hours** |
