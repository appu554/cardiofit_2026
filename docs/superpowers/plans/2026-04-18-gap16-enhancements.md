# Gap 16 Enhancements — Weight Validation + Clinician Labels + Temporal State Machine

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add three clinical-safety enhancements to the acute-on-chronic detection engine: (1) weight validation gate preventing false-positive CHF alerts, (2) clinician-friendly compound pattern labels for nurse-readable cards, (3) explicit Spike→Trend→Persistence temporal classification for every deviation.

**Architecture:** All three enhancements extend existing Gap 16 files in KB-26 and card templates in KB-23. No new services — modifications to existing models, deviation scorer, compound pattern detector, event handler, and card templates. A new `ValidationManager` handles pending weight confirmations.

**Tech Stack:** Go 1.21 (GORM) for KB-26 extensions. PostgreSQL for pending_validations table. Existing YAML configs extended with new fields.

---

## File Inventory

| Action | File | Enhancement |
|---|---|---|
| Modify | `kb-26/internal/models/acute_event.go` | ValidationState enum, TemporalClassification enum, UsualMeasurementHour, RawSeverity/EffectiveSeverity, ClinicianLabel/PatientLabel on CompoundPatternMatch |
| Modify | `kb-26/internal/services/patient_baseline.go` | Compute UsualMeasurementHour from reading timestamps |
| Modify | `kb-26/internal/services/deviation_scorer.go` | Weight validation rules + temporal classification + severity modulation |
| Create | `kb-26/internal/services/validation_manager.go` | PendingValidation model, create/confirm/refute/expire logic, background expiry checker |
| Create | `kb-26/internal/services/validation_manager_test.go` | 6 tests |
| Modify | `kb-26/internal/services/compound_pattern_detector.go` | Add ClinicianLabel/PatientLabel to matches |
| Modify | `market-configs/shared/compound_patterns.yaml` | Add clinician_label + patient_label to all 5 patterns |
| Modify | `kb-26/internal/services/acute_event_handler.go` | Pass temporal + validation state through pipeline |
| Create | `kb-26/internal/services/temporal_classifier.go` | ClassifyTemporal + ModulateSeverity pure functions |
| Create | `kb-26/internal/services/temporal_classifier_test.go` | 4 tests |
| Modify | 6 KB-23 templates in `templates/acute/` | Add {{.ClinicianLabel}}, {{.TemporalContext}}, {{.ValidationNote}} |

**Total: ~15 files (3 create, 12 modify), ~15 tests**

---

### Task 1: Weight validation gate

**Files:**
- Modify: `kb-26-metabolic-digital-twin/internal/models/acute_event.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/patient_baseline.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/deviation_scorer.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/validation_manager.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/validation_manager_test.go`

- [ ] **Step 1:** Add to `acute_event.go`: `ValidationState` enum (CONFIRMED, UNCONFIRMED, AWAITING_CONFIRMATION, NOT_APPLICABLE), `ValidationReason` field on DeviationResult, `PendingValidation` GORM model (ID, PatientID, VitalSignType, OriginalValue, OriginalDeviation, OriginalReadingTime, ExpiresAt, ConfirmationValue, ValidationOutcome, CreatedAt).

- [ ] **Step 2:** Add `UsualMeasurementHour int` field to `PatientBaselineSnapshot`. In `patient_baseline.go`, compute median hour from reading timestamps during `ComputeBaseline`.

- [ ] **Step 3:** In `deviation_scorer.go`, add weight-specific validation rules to `ComputeDeviation`: if vitalType is "WEIGHT" and (a) measurement hour differs >2h from UsualMeasurementHour → ValidationState=UNCONFIRMED reason TIME_OF_DAY_INCONSISTENT, or (b) deviation is MODERATE-HIGH and no prior deviating reading in 48h → ValidationState=AWAITING_CONFIRMATION, or (c) deviation exceeds CRITICAL for CKM 4c → ValidationState=UNCONFIRMED_CRITICAL (bypasses waiting but notes uncertainty).

- [ ] **Step 4:** Write 6 tests for ValidationManager:
1. `TestValidation_TimeOfDay_Flags` — weight at 3pm (usual 7am) → UNCONFIRMED
2. `TestValidation_FirstDeviation_AwaitingConfirmation` — 2.2kg gain, no prior → AWAITING_CONFIRMATION
3. `TestValidation_Confirmation_Within20Pct_Confirmed` — original 2.2kg, confirmation 2.0kg → CONFIRMED
4. `TestValidation_Confirmation_Over50Pct_Refuted` — original 2.2kg, confirmation 0.5kg → REFUTED
5. `TestValidation_Expired_NoConfirmation` — 24h passes, no confirmation → EXPIRED_UNCONFIRMED
6. `TestValidation_Critical_BypassesWaiting` — >3kg CKM 4c → fires immediately as UNCONFIRMED_CRITICAL

- [ ] **Step 5:** Implement `ValidationManager` with CreatePending, ProcessConfirmation, ExpireStale methods.

- [ ] **Step 6:** Run tests. Commit: `feat(kb26): weight validation gate — pending confirmation + temporal consistency (Gap 16 Enhancement 1)`

---

### Task 2: Clinician-friendly compound pattern labels

**Files:**
- Modify: `market-configs/shared/compound_patterns.yaml`
- Modify: `kb-26-metabolic-digital-twin/internal/models/acute_event.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/compound_pattern_detector.go`
- Modify: 6 KB-23 templates in `templates/acute/`

- [ ] **Step 1:** Add `clinician_label` and `patient_label` to all 5 patterns in compound_patterns.yaml:
- CARDIORENAL_SYNDROME → "Heart-Kidney Strain" / "Change detected in heart and kidney measurements"
- INFECTION_CASCADE → "Possible Infection with Multi-System Impact" / "Your care team noticed changes that may indicate an infection"
- MEDICATION_CRISIS → "Possible Medication Reaction" / "A recent medication change may need adjustment"
- FLUID_OVERLOAD_TRIAD → "Fluid Overload" / "Your weight and blood pressure changes need attention"
- POST_DISCHARGE_DETERIORATION → "Post-Hospital Deterioration" / "Changes detected since your hospital discharge"

- [ ] **Step 2:** Add `ClinicianLabel string` and `PatientLabel string` fields to `CompoundPatternMatch` in acute_event.go.

- [ ] **Step 3:** Update `DetectCompoundPatterns` to populate ClinicianLabel and PatientLabel from pattern config.

- [ ] **Step 4:** Update 2 compound-relevant KB-23 templates (compound_cardiorenal.yaml, medication_induced_crisis.yaml) to use `{{.ClinicianLabel}}` in CLINICIAN fragment and `{{.PatientLabel}}` in PATIENT fragment.

- [ ] **Step 5:** Write 2 tests: compound match has ClinicianLabel populated, card template references ClinicianLabel.

- [ ] **Step 6:** Commit: `feat(kb26): clinician-friendly compound pattern labels (Gap 16 Enhancement 2)`

---

### Task 3: Spike → Trend → Persistence temporal state machine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/temporal_classifier.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/temporal_classifier_test.go`
- Modify: `kb-26-metabolic-digital-twin/internal/models/acute_event.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/acute_event_handler.go`

- [ ] **Step 1:** Add to `acute_event.go`: `TemporalClassification` enum (SPIKE, TREND, PERSISTENCE), `RawSeverity` and `EffectiveSeverity` fields on DeviationResult, `TemporalClassification` field on AcuteEvent.

- [ ] **Step 2:** Write 4 tests:
1. `TestTemporal_FirstDeviation_Spike` — no prior deviations → SPIKE
2. `TestTemporal_ThreeConsecutive_Trend` — 2+ prior deviations same direction within 24h → TREND
3. `TestTemporal_Sustained24h_Persistence` — 5+ deviations spanning >24h → PERSISTENCE
4. `TestTemporal_SeverityModulation` — SPIKE at CRITICAL → effective HIGH; PERSISTENCE at MODERATE → effective HIGH

- [ ] **Step 3:** Implement `ClassifyTemporal(currentDeviation DeviationResult, priorDeviations []DeviationResult) string` — pure function: counts matching-direction prior deviations, checks time span, returns SPIKE/TREND/PERSISTENCE.

- [ ] **Step 4:** Implement `ModulateSeverity(rawSeverity, temporalClass string) string` — applies the modulation table: SPIKE downgrades by 1 (CRITICAL→HIGH, HIGH→MODERATE), PERSISTENCE upgrades by 1 (MODERATE→HIGH, HIGH→CRITICAL), TREND preserves raw severity.

- [ ] **Step 5:** Wire into `acute_event_handler.go`: after computing deviation, call ClassifyTemporal + ModulateSeverity, store both RawSeverity and EffectiveSeverity on the event.

- [ ] **Step 6:** Run tests. Commit: `feat(kb26): temporal state machine — Spike/Trend/Persistence classification (Gap 16 Enhancement 3)`

---

### Task 4: Integration test + final commit

- [ ] **Step 1:** Full test sweep KB-26 + KB-23.
- [ ] **Step 2:** Verify YAML parses.
- [ ] **Step 3:** Commit: `feat: complete Gap 16 enhancements — validation + labels + temporal`
- [ ] **Step 4:** Push to origin.

---

## Effort Estimate

| Task | Expected |
|---|---|
| Task 1: Weight validation (6 tests) | 2 hours |
| Task 2: Clinician labels (2 tests) | 45 minutes |
| Task 3: Temporal classifier (4 tests) | 90 minutes |
| Task 4: Integration + commit | 30 minutes |
| **Total** | **~4.5 hours** |
