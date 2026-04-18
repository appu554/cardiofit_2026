# Gap 17: Care Transition Bridge — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a 30-day post-discharge surveillance module that coordinates medication reconciliation, baseline adjustment, and engagement monitoring — converting the platform's steady-state detection into a dynamic transition support system that prevents the 20-25% HF readmission rate.

**Architecture:** The bridge is a temporal overlay, not a parallel system. It modifies existing detection parameters during the 30-day window: tightens Gap 16 deviation thresholds by 25%, boosts PAI context score by +15, shortens engagement gap alerting from 7d to 72h, and amplifies escalation tiers. Three services participate: KB-20 (discharge detection + medication reconciliation), KB-26 (4-state baseline machine + heightened surveillance), KB-23 (milestone scheduler + 7 card templates + outcome tracking). Milestone-triggered reviews at 48h/7d/14d/30d provide structured assessment without continuous surveillance noise.

**Tech Stack:** Go 1.21 (Gin, GORM) across KB-20, KB-23, KB-26. PostgreSQL for transition state + milestone tracking + outcome registry. YAML market configs for transition parameters + reconciliation rules. Existing PAI + Gap 15 + Gap 16 as dependencies.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| PatientProfile with medications | KB-20 | Pre-admission medication list source |
| FHIR sync worker | KB-20 `internal/fhir/fhir_sync_worker.go` | Discharge event detection from FHIR Encounters |
| FormularyChecker (Gap 13) | KB-23 | Drug availability cross-check for discharge meds |
| PatientBaselineSnapshot (Gap 16) | KB-26 | Needs BaselineStage field for post-discharge state machine |
| DeviationScorer (Gap 16) | KB-26 | Needs threshold multiplier for heightened surveillance |
| PAI context dimension | KB-26 `internal/services/pai_context.go` | Needs post-discharge boost (+15 points) |
| EscalationRouter (Gap 15) | KB-23 | Needs POST_DISCHARGE amplification rule |
| AuditService (Gap 11) | KB-23 | Transition lifecycle events logged to audit trail |

## File Inventory

### KB-20 — Discharge Detection + Reconciliation
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/care_transition.go` | CareTransition, DischargeMedication, TransitionMilestone, TransitionOutcome, MedicationReconciliationReport |
| Create | `internal/services/discharge_detector.go` | Multi-source discharge detection (FHIR, manual, patient-reported) |
| Create | `internal/services/discharge_detector_test.go` | 5 tests |
| Create | `internal/services/medication_reconciliation.go` | Compare pre-admission vs discharge regimens |
| Create | `internal/services/medication_reconciliation_test.go` | 6 tests |
| Create | `internal/api/care_transition_handlers.go` | POST /patient/:id/discharge, GET /patient/:id/transition |
| Modify | `internal/api/routes.go` | Add transition routes |
| Modify | `internal/database/connection.go` | AutoMigrate transition models |

### KB-26 — Baseline Adjustment + Heightened Surveillance
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/services/baseline_adjustment.go` | 4-state machine: HOSPITAL_INFLUENCED → BUILDING → EVOLVING → STEADY_STATE |
| Create | `internal/services/baseline_adjustment_test.go` | 5 tests |
| Create | `internal/services/heightened_surveillance.go` | Threshold tightening, PAI boost, engagement gap sensitivity, escalation amplification |
| Create | `internal/services/heightened_surveillance_test.go` | 5 tests |
| Modify | `internal/models/acute_event.go` | Add BaselineStage field to PatientBaselineSnapshot |
| Modify | `internal/services/pai_context.go` | Add post-discharge boost |

### KB-23 — Milestones + Cards + Outcomes
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/services/milestone_scheduler.go` | Schedule 48h/7d/14d/30d milestones + market-specific extras |
| Create | `internal/services/milestone_scheduler_test.go` | 5 tests |
| Create | `internal/services/transition_exit_controller.go` | 4-outcome assessment (SUCCESSFUL/READMITTED/DETERIORATED/DISENGAGED) |
| Create | `internal/services/transition_exit_controller_test.go` | 4 tests |
| Create | `templates/transition/medication_reconciliation.yaml` | 48h reconciliation card |
| Create | `templates/transition/day7_followup.yaml` | 7-day trajectory summary |
| Create | `templates/transition/day14_midpoint.yaml` | 14-day workstream review |
| Create | `templates/transition/day30_exit.yaml` | Transition exit assessment |
| Create | `templates/transition/engagement_cliff.yaml` | 72h measurement gap alert |
| Create | `templates/transition/medication_supply_gap.yaml` | Drug availability concern |
| Create | `templates/transition/missed_milestone.yaml` | Missed milestone alert |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/care_transition_parameters.yaml` | Windows, baseline adjustment, surveillance params, milestone schedule, reconciliation rules |
| Create | `market-configs/india/care_transition_overrides.yaml` | ASHA integration, paper discharge, supply sensitivity |
| Create | `market-configs/australia/care_transition_overrides.yaml` | MHR updates, HITH, aged care, MBS billing |

**Total: 28 files (24 create, 4 modify), ~40 tests**

---

### Task 1: Care transition models + YAML configs

**Files:**
- Create: `kb-20-patient-profile/internal/models/care_transition.go`
- Create: `market-configs/shared/care_transition_parameters.yaml`
- Create: `market-configs/india/care_transition_overrides.yaml`
- Create: `market-configs/australia/care_transition_overrides.yaml`

- [ ] **Step 1:** Create `care_transition.go` with 5 types:

`CareTransition` GORM model: ID, PatientID (indexed), DischargeDate, DetectedAt, DischargeSource (FHIR_ENCOUNTER/MANUAL/PATIENT_REPORTED/ASHA_REPORTED), FacilityName, FacilityType (ACUTE_HOSPITAL/REHAB/AGED_CARE/HITH), PrimaryDiagnosis, LengthOfStayDays, DischargeDisposition (HOME/AGED_CARE_FACILITY/HITH/REHAB), TransitionState (ACTIVE/COMPLETED_SUCCESSFUL/COMPLETED_READMITTED/COMPLETED_DETERIORATED/COMPLETED_DISENGAGED), HeightenedSurveillanceActive bool, ReconciliationStatus (PENDING/RECONCILED/DISCREPANCIES_FOUND), TransitionEndDate, SourceConfidence (HIGH/MODERATE/LOW), CreatedAt, UpdatedAt.

`DischargeMedication` GORM model: ID, TransitionID (foreign key), DrugName, DrugClass, DoseMg, Frequency, ReconciliationStatus (NEW/CONTINUED/CHANGED_DOSE/STOPPED/UNCLEAR), PreAdmissionDrugName (nullable — what it replaced), ChangeReason, ClinicalRiskLevel (LOW/MEDIUM/HIGH/CRITICAL), FormularyStatus (AVAILABLE/UNAVAILABLE/SUBSTITUTE_AVAILABLE), SupplyGapRisk (LOW/MEDIUM/HIGH).

`TransitionMilestone` GORM model: ID, TransitionID, MilestoneType (MEDICATION_RECONCILIATION_48H/FIRST_FOLLOWUP_7D/MIDPOINT_REVIEW_14D/EXIT_ASSESSMENT_30D/ENGAGEMENT_CHECK_72H/MEDICATION_SUPPLY_CHECK), ScheduledFor, CompletedAt, CompletionStatus (SCHEDULED/TRIGGERED/COMPLETED/MISSED), CardsGenerated int, Notes.

`TransitionOutcome` GORM model: ID, TransitionID, OutcomeCategory (SUCCESSFUL/READMITTED/DETERIORATED/DISENGAGED), ReadmissionDate, ReadmissionReason, FinalPAITier, MedicationReconciliationOutcome, EngagementMetric float64, EscalationsTriggeredCount int, ComputedAt.

`MedicationReconciliationReport`: TransitionID, NewMedications []DischargeMedication, StoppedMedications []DischargeMedication, ChangedMedications []DischargeMedication, UnclearMedications []DischargeMedication, DiscrepanciesFound int, HighRiskChanges int, FormularyIssues int, ReconciliationOutcome (CLEAN/DISCREPANCIES_CLINICIAN_REVIEW/HIGH_RISK_URGENT/UNCLEAR_INSUFFICIENT_DATA).

- [ ] **Step 2:** Create `care_transition_parameters.yaml` with: standard window (30d), short window (14d for elective), extended window (60d for complex), baseline start delay (48h), minimum readings for new baseline (5), fallback to pre-admission (14d), deviation threshold multiplier (0.75 = 25% tighter), PAI context boost (+15), engagement gap alert (72h), milestone schedule offsets, reconciliation risk thresholds (high-risk drug classes list), reconciliation urgency mapping.

- [ ] **Step 3:** Create India + Australia override YAMLs with market-specific parameters.

- [ ] **Step 4:** Verify compile + YAML parse. Commit: `feat(kb20): care transition models + market config (Gap 17 Task 1)`

---

### Task 2: Medication reconciliation engine

**Files:**
- Create: `kb-20-patient-profile/internal/services/medication_reconciliation.go`
- Create: `kb-20-patient-profile/internal/services/medication_reconciliation_test.go`

- [ ] **Step 1:** Write 6 failing tests:
1. `TestReconciliation_NewAnticoagulant_HighRisk` — discharge list has warfarin not in pre-admission → NEW with CRITICAL risk
2. `TestReconciliation_StoppedBetaBlocker_PostMI_HighRisk` — pre-admission has metoprolol, discharge doesn't, diagnosis includes MI → STOPPED with HIGH risk (cardioprotective dropped)
3. `TestReconciliation_ContinuedStatin_NotFlagged` — atorvastatin in both lists at same dose → CONTINUED, no flag
4. `TestReconciliation_UnclearCardiacMeds_Clarification` — discharge says "continue cardiac medications" without specifics → UNCLEAR, generates clarification request
5. `TestReconciliation_NewMetforminLowEGFR_HighRisk` — new metformin added, patient eGFR 35 → HIGH risk (renal inappropriateness)
6. `TestReconciliation_CleanReconciliation_NoIssues` — identical pre/post lists → CLEAN outcome, zero discrepancies

- [ ] **Step 2:** Implement `ReconcileRegimens(preAdmission, discharge []MedicationEntry, patientEGFR *float64, diagnosis string) MedicationReconciliationReport` — pure function that compares two medication lists and classifies each drug as NEW/CONTINUED/CHANGED/STOPPED/UNCLEAR with risk assessment.

`MedicationEntry` struct: DrugName, DrugClass, DoseMg, Frequency. The function matches by drug name (case-insensitive, generic name normalization).

High-risk drug classes: ANTICOAGULANT, INSULIN, OPIOID, DIGOXIN, AMIODARONE. If a NEW drug is in this list → CRITICAL risk. If a STOPPED drug is cardioprotective (BETA_BLOCKER, ACEi, STATIN in post-MI) → HIGH risk. If a NEW drug is contraindicated at patient's eGFR → HIGH risk.

- [ ] **Step 3:** Run tests — all 6 pass. Commit: `feat(kb20): medication reconciliation engine — 5 change classifications (Gap 17 Task 2)`

---

### Task 3: Discharge event detector + API handlers

**Files:**
- Create: `kb-20-patient-profile/internal/services/discharge_detector.go`
- Create: `kb-20-patient-profile/internal/services/discharge_detector_test.go`
- Create: `kb-20-patient-profile/internal/api/care_transition_handlers.go`
- Modify: `kb-20-patient-profile/internal/api/routes.go`
- Modify: `kb-20-patient-profile/internal/database/connection.go`

- [ ] **Step 1:** Write 5 tests for discharge detector:
1. `TestDischarge_FHIREncounter_CreateTransition` — FHIR Encounter with status finished + class inpatient → CareTransition created, source HIGH confidence
2. `TestDischarge_ManualEntry_CreateTransition` — manual POST with discharge date + facility → created, source HIGH confidence
3. `TestDischarge_PatientReported_LowConfidence` — patient-reported discharge → created with LOW confidence
4. `TestDischarge_Duplicate_Deduplicated` — two sources for same patient + same date → single transition
5. `TestDischarge_TooOld_Rejected` — discharge >14 days ago → rejected

- [ ] **Step 2:** Implement `DischargeDetector` with:
- `DetectFromFHIR(encounter FHIREncounterData) (*CareTransition, error)`
- `DetectFromManual(patientID, facilityName string, dischargeDate time.Time, diagnosis string) (*CareTransition, error)`
- `DetectFromPatientReport(patientID string, dischargeDate time.Time, reportedBy string) (*CareTransition, error)`
- Deduplication: check existing transitions for same patient + discharge date within 24h
- Rejection: discharge >14 days old

- [ ] **Step 3:** Create API handlers:
- `POST /api/v1/patient/:id/discharge` — manual discharge registration (body: facility, date, diagnosis)
- `GET /api/v1/patient/:id/transition` — current active transition or most recent
- `GET /api/v1/patient/:id/transition/milestones` — milestone schedule + status

- [ ] **Step 4:** Add routes, AutoMigrate (CareTransition, DischargeMedication, TransitionMilestone, TransitionOutcome).

- [ ] **Step 5:** Run tests. Commit: `feat(kb20): discharge event detector + transition API (Gap 17 Task 3)`

---

### Task 4: Baseline adjustment controller — 4-state machine

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/baseline_adjustment.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/baseline_adjustment_test.go`
- Modify: `kb-26-metabolic-digital-twin/internal/models/acute_event.go`

- [ ] **Step 1:** Add `BaselineStage string` field to `PatientBaselineSnapshot` in acute_event.go. Constants: HOSPITAL_INFLUENCED, BUILDING_NEW_BASELINE, POST_DISCHARGE_EVOLVING, STEADY_STATE, PRE_ADMISSION_FALLBACK.

- [ ] **Step 2:** Write 5 tests:
1. `TestBaseline_Day1_HospitalInfluenced` — 1 day post-discharge → stage HOSPITAL_INFLUENCED, deviation detection suppressed (returns nil)
2. `TestBaseline_Day5_BuildingWithFallback` — 5 days post, 3 readings → stage BUILDING, uses pre-admission baseline as fallback
3. `TestBaseline_Day15_EvolvedBaseline` — 15 days post, 7+ readings → stage EVOLVING, uses post-discharge readings
4. `TestBaseline_Day31_SteadyState` — 31 days post → stage STEADY_STATE, normal detection
5. `TestBaseline_CriticalBypassesSuppression` — day 1 but SBP 190 → CRITICAL absolute threshold still fires despite hospital-influenced state

- [ ] **Step 3:** Implement `BaselineAdjustmentController` with:
- `DetermineBaselineStage(daysSinceDischarge int, postDischargeReadingCount int) string`
- `ShouldSuppressDeviation(stage string, severity string) bool` — suppress non-CRITICAL during HOSPITAL_INFLUENCED
- `GetEffectiveBaseline(stage string, postDischargeBaseline, preAdmissionBaseline *PatientBaselineSnapshot) *PatientBaselineSnapshot` — returns appropriate baseline for current stage
- `ThresholdMultiplier(stage string) float64` — 1.5 for BUILDING (wider thresholds), 1.0 for EVOLVING/STEADY_STATE

- [ ] **Step 4:** Run tests. Commit: `feat(kb26): baseline adjustment — 4-state post-discharge machine (Gap 17 Task 4)`

---

### Task 5: Heightened surveillance mode

**Files:**
- Create: `kb-26-metabolic-digital-twin/internal/services/heightened_surveillance.go`
- Create: `kb-26-metabolic-digital-twin/internal/services/heightened_surveillance_test.go`
- Modify: `kb-26-metabolic-digital-twin/internal/services/pai_context.go`

- [ ] **Step 1:** Write 5 tests:
1. `TestSurveillance_DeviationThreshold_Tightened` — 25% drop normally MODERATE, during transition becomes HIGH (0.75 multiplier makes 19% the threshold)
2. `TestSurveillance_PAIContextBoost` — PAI context score +15 during active transition
3. `TestSurveillance_EngagementGap72h` — 72h without measurement during transition → alert (vs 7d normally)
4. `TestSurveillance_EscalationAmplification` — ROUTINE card during transition → amplified to URGENT
5. `TestSurveillance_ExitRestoresNormal` — after transition ends, all parameters return to standard

- [ ] **Step 2:** Implement `HeightenedSurveillanceMode` with:
- `IsActive(patientID string) bool`
- `GetDeviationMultiplier() float64` — returns 0.75 during active transition
- `GetPAIContextBoost() float64` — returns 15.0 during active transition
- `GetEngagementGapHours() int` — returns 72 during transition, 168 (7d) normally
- `AmplifyEscalationTier(tier string) string` — ROUTINE→URGENT, URGENT→IMMEDIATE during transition
- `Deactivate(patientID string)` — restores all parameters to normal

- [ ] **Step 3:** Modify `pai_context.go`: in `ComputeContextScore`, check if patient has active transition and add boost.

- [ ] **Step 4:** Run tests. Commit: `feat(kb26): heightened surveillance — tightened thresholds + PAI boost (Gap 17 Task 5)`

---

### Task 6: Milestone scheduler

**Files:**
- Create: `kb-23-decision-cards/internal/services/milestone_scheduler.go`
- Create: `kb-23-decision-cards/internal/services/milestone_scheduler_test.go`

- [ ] **Step 1:** Write 5 tests:
1. `TestScheduler_StandardSchedule_4Milestones` — standard discharge → 48h, 7d, 14d, 30d milestones at correct offsets
2. `TestScheduler_EngagementCheck_Conditional` — patient with no measurement in 24h → additional 72h engagement check
3. `TestScheduler_SupplyCheck_HighRisk` — discharge with HIGH supply gap risk drug → 24h supply check
4. `TestScheduler_MissedMilestone_Alert` — milestone not completed within 48h grace → status MISSED
5. `TestScheduler_MilestoneAssessment_48h_Clean` — 48h assessment with CLEAN reconciliation → ROUTINE card

- [ ] **Step 2:** Implement `MilestoneScheduler` with:
- `ScheduleMilestones(transition CareTransition, config TransitionConfig) []TransitionMilestone`
- `AssessMilestone(milestone TransitionMilestone, transition CareTransition, reconciliation *MedicationReconciliationReport) MilestoneAssessment`
- `MilestoneAssessment` struct: MilestoneType, Findings, CardTier (ROUTINE/URGENT/IMMEDIATE), SuggestedActions

The scheduler creates TransitionMilestone records. A background check (or API-triggered) evaluates milestones when their ScheduledFor time arrives.

- [ ] **Step 3:** Run tests. Commit: `feat(kb23): milestone scheduler — 48h/7d/14d/30d structured reviews (Gap 17 Task 6)`

---

### Task 7: Transition exit controller + outcome registry

**Files:**
- Create: `kb-23-decision-cards/internal/services/transition_exit_controller.go`
- Create: `kb-23-decision-cards/internal/services/transition_exit_controller_test.go`

- [ ] **Step 1:** Write 4 tests:
1. `TestExit_30d_CleanTrajectory_Successful` — no readmission, PAI never CRITICAL, ≤2 escalations, reconciliation resolved → SUCCESSFUL
2. `TestExit_Readmission_Day14` — readmission detected at day 14 → READMITTED, correct timing
3. `TestExit_PAIHigh_Throughout_Deteriorated` — PAI HIGH/CRITICAL at day 30 → DETERIORATED
4. `TestExit_NoReadings14Days_Disengaged` — zero readings for 14+ consecutive days → DISENGAGED

- [ ] **Step 2:** Implement `TransitionExitController` with:
- `ComputeOutcome(transition CareTransition, paiHistory []PAIHistoryEntry, escalationCount int, readingCount int, readmitted bool) TransitionOutcome`
- `ExitTransition(transitionID string) error` — marks transition complete, resets baseline stage to STEADY_STATE, deactivates heightened surveillance

- [ ] **Step 3:** Run tests. Commit: `feat(kb23): transition exit controller — 4 outcome categories (Gap 17 Task 7)`

---

### Task 8: Card templates + wiring + integration test

**Files:**
- Create: 7 YAML templates in `kb-23-decision-cards/templates/transition/`
- Modify: `kb-23-decision-cards/internal/api/routes.go`
- Modify: KB-20 + KB-26 main.go for AutoMigrate + service wiring

- [ ] **Step 1:** Create 7 card templates: medication_reconciliation.yaml (HALT for HIGH_RISK, MODIFY for DISCREPANCIES, SAFE for CLEAN), day7_followup.yaml, day14_midpoint.yaml, day30_exit.yaml, engagement_cliff.yaml (MODIFY gate, 72h gap trigger), medication_supply_gap.yaml (MODIFY gate, formulary concern), missed_milestone.yaml (MODIFY gate, care pathway breakdown). Each follows existing template pattern with CLINICIAN + PATIENT fragments.

- [ ] **Step 2:** Wire services: KB-20 AutoMigrate transition models, KB-26 add BaselineStage to acute models AutoMigrate, KB-23 milestone scheduler available via setter injection.

- [ ] **Step 3:** Full test sweep KB-20 + KB-23 + KB-26. Verify YAML.

- [ ] **Step 4:** Commit: `feat: complete Gap 17 care transition bridge`

- [ ] **Step 5:** Push to origin.

---

## Verification Questions

1. Does a FHIR Encounter finished event create a CareTransition? (yes / test)
2. Does a duplicate discharge get deduplicated? (yes / test)
3. Does a new anticoagulant flag as CRITICAL risk? (yes / test)
4. Does a stopped beta-blocker in post-MI flag as HIGH risk? (yes / test)
5. Does day 1 post-discharge suppress non-CRITICAL deviations? (yes / test)
6. Does SBP 190 still fire on day 1 despite suppression? (yes / test)
7. Does day 15 with 7+ readings use post-discharge evolved baseline? (yes / test)
8. Does PAI context get +15 boost during active transition? (yes / test)
9. Does 72h measurement gap trigger alert during transition? (yes / test)
10. Does ROUTINE card amplify to URGENT during transition? (yes / test)
11. Does standard schedule create 4 milestones? (yes / test)
12. Does 30-day clean trajectory produce SUCCESSFUL outcome? (yes / test)
13. Does readmission produce READMITTED with correct timing? (yes / test)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML | 4 files | 1-2 hours |
| Task 2: Medication reconciliation (6 tests) | 2 files | 2-3 hours |
| Task 3: Discharge detector + API (5 tests) | 5 files | 2-3 hours |
| Task 4: Baseline adjustment (5 tests) | 3 files | 2-3 hours |
| Task 5: Heightened surveillance (5 tests) | 3 files | 2-3 hours |
| Task 6: Milestone scheduler (5 tests) | 2 files | 2-3 hours |
| Task 7: Exit controller (4 tests) | 2 files | 1-2 hours |
| Task 8: Templates + wiring | 10 files | 2-3 hours |
| **Total** | **~28 files, ~40 tests** | **~15-22 hours** |
