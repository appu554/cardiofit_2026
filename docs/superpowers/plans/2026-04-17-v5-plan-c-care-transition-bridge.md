# V5 Plan C: Care Transition Bridge — Post-Discharge Surveillance (Gap 17)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build a 30-day post-discharge surveillance system that detects hospital discharge events, tightens monitoring thresholds, triggers medication reconciliation, and adjusts PAI weights — preventing the 20-25% HF readmission rate within the highest-risk window.

**Architecture:** A new CareTransition service in KB-20 detects FHIR Encounter discharge events and enters a per-patient "heightened surveillance" state stored on PatientProfile. During the 30-day window: PAI attention gap thresholds are halved (so 15 days without clinician contact triggers HIGH instead of 30), the confounder scorer flags post-discharge as a washout period (already exists — this connects it), and a medication reconciliation card is generated within 48 hours. A 7-day and 30-day summary card compares pre-admission vs post-discharge trajectories.

**Tech Stack:** Go 1.21 for KB-20 care transition service. FHIR R4 Encounter resources for discharge detection. KB-23 card templates for reconciliation + summary cards. Existing PAI infrastructure for threshold adjustment.

---

## File Inventory

### KB-20 (Patient Profile) — Care Transition Service
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/care_transition.go` | CareTransition GORM model (PatientID, AdmissionDate, DischargeDate, DischargeReason, SurveillanceEndDate, Status, TightenedThresholds) |
| Create | `internal/services/care_transition_service.go` | RecordDischarge, IsInSurveillanceWindow, GetTransitionContext, EndSurveillance |
| Create | `internal/services/care_transition_service_test.go` | 6 tests |
| Create | `internal/api/care_transition_handlers.go` | POST /patient/:id/discharge, GET /patient/:id/transition-status |
| Modify | `internal/api/routes.go` | Add transition routes |
| Modify | `internal/database/connection.go` | AutoMigrate CareTransition |

### KB-23 (Decision Cards) — Transition Cards
| Action | File | Responsibility |
|---|---|---|
| Create | `templates/transition/medication_reconciliation.yaml` | 48-hour post-discharge med review card |
| Create | `templates/transition/day7_summary.yaml` | 7-day trajectory comparison card |
| Create | `templates/transition/day30_summary.yaml` | 30-day surveillance end card |
| Create | `internal/services/care_transition_cards.go` | Generate transition-specific cards |
| Create | `internal/services/care_transition_cards_test.go` | 4 tests |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/care_transition.yaml` | Surveillance window days, threshold multipliers, reconciliation deadline hours |

**Total: 12 files (11 create, 1 modify)**

---

### Task 1: Care transition models + config

- [ ] **Step 1:** Create `CareTransition` GORM model with: PatientID, AdmissionDate, DischargeDate, DischargeReason (string: HF_DECOMPENSATION, AKI, DKA, HYPERTENSIVE_CRISIS, ELECTIVE, OTHER), SurveillanceEndDate (DischargeDate + 30 days), Status (ACTIVE, COMPLETED, CANCELLED), MedReconciliationDue (DischargeDate + 48h), MedReconciliationCompleted bool, PAIThresholdMultiplier float64 (0.5 = halved thresholds during surveillance), CreatedAt.

- [ ] **Step 2:** Create `care_transition.yaml`:
```yaml
surveillance_window_days: 30
reconciliation_deadline_hours: 48
pai_threshold_multiplier: 0.5
summary_days: [7, 30]
discharge_reasons:
  high_risk: [HF_DECOMPENSATION, AKI, DKA, HYPERTENSIVE_CRISIS]
  standard_risk: [ELECTIVE, OTHER]
high_risk_window_days: 45    # extended surveillance for high-risk discharges
```

- [ ] **Step 3:** Commit: `feat(kb20): care transition models + config (V5 Plan C Task 1)`

---

### Task 2: Care transition service

- [ ] **Step 1:** Write 6 tests:
1. `TestTransition_RecordDischarge_CreatesSurveillance` — discharge recorded → CareTransition with Status ACTIVE, SurveillanceEndDate = discharge + 30d
2. `TestTransition_HighRisk_ExtendedWindow` — HF_DECOMPENSATION discharge → window 45 days
3. `TestTransition_IsInSurveillance_WithinWindow` — 10 days post-discharge → true
4. `TestTransition_IsInSurveillance_OutsideWindow` — 35 days post-discharge → false
5. `TestTransition_GetContext_ReturnsThresholdMultiplier` — returns 0.5 multiplier for active surveillance
6. `TestTransition_EndSurveillance_MarksCompleted` — manual end → Status = COMPLETED

- [ ] **Step 2:** Implement CareTransitionService with RecordDischarge, IsInSurveillanceWindow, GetTransitionContext, EndSurveillance.

- [ ] **Step 3:** Commit: `feat(kb20): care transition service (V5 Plan C Task 2)`

---

### Task 3: Transition card templates + generator

- [ ] **Step 1:** Create 3 YAML templates (medication_reconciliation, day7_summary, day30_summary).

- [ ] **Step 2:** Write 4 tests for card generation:
1. `TestTransitionCards_MedReconciliation_48h` — discharge event → med reconciliation card within 48h deadline
2. `TestTransitionCards_Day7Summary_Generated` — 7 days post-discharge → trajectory comparison card
3. `TestTransitionCards_Day30Summary_SurveillanceEnd` — 30 days → final summary card with recommendation
4. `TestTransitionCards_HighRisk_UrgentReconciliation` — HF discharge → reconciliation card with IMMEDIATE urgency

- [ ] **Step 3:** Implement card generator.

- [ ] **Step 4:** Commit: `feat(kb23): care transition cards — reconciliation + trajectory summaries (V5 Plan C Task 3)`

---

### Task 4: API handlers + wiring + integration

- [ ] **Step 1:** Create handlers, add routes, wire service.
- [ ] **Step 2:** Full test sweep.
- [ ] **Step 3:** Commit: `feat: complete V5 care transition bridge`

---

## Effort Estimate

| Task | Expected |
|---|---|
| Task 1: Models + config | 1-2 hours |
| Task 2: Transition service + 6 tests | 2-3 hours |
| Task 3: Card templates + 4 tests | 2-3 hours |
| Task 4: API + wiring | 1-2 hours |
| **Total** | **~7-10 hours** |
