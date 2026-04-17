# V5 Plan D: Response Loop — Time-to-Response + Predictive Risk + Closed-Loop Learning (Gaps 19+20+21)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the measurement-and-learning infrastructure that tracks every card from detection to outcome (T0→T4), feeds outcome data into a predictive risk model, and closes the loop by adjusting detection thresholds based on what actually worked.

**Architecture:** Three layers built incrementally: (1) ResponseTracker in KB-23 captures 5 timestamps per card lifecycle (detection, delivery, acknowledgment, action, outcome) and computes interval metrics; (2) PredictiveRiskEngine in KB-26 uses the multi-domain feature set + outcome data to predict 30-day hospitalization probability; (3) OutcomeFeedbackLoop reads IOR outcomes + response tracker data and adjusts detection thresholds in market-config YAML. Layer 1 is immediately buildable. Layer 2 requires 3-6 months of outcome data. Layer 3 requires validated Layer 2 predictions.

**Tech Stack:** Go 1.21 for KB-23 response tracker + KB-26 predictive engine. PostgreSQL for response lifecycle tables. Python for ML model training (Layer 2, deferred). YAML market configs for threshold adjustment (Layer 3).

---

## Phased Approach

This plan covers 3 gaps with increasing maturity requirements:

| Phase | Gap | When to Build | Prerequisite |
|-------|-----|---------------|-------------|
| **Phase 1** | Gap 19: Time-to-Response | **Now** | KB-14 integration (Plan A) |
| **Phase 2** | Gap 20: Predictive Risk | **After 3-6 months of outcome data** | Phase 1 collecting data |
| **Phase 3** | Gap 21: Closed-Loop Learning | **After validated predictions** | Phase 2 model validated |

**This plan fully specifies Phase 1 and architecturally outlines Phases 2-3.**

---

## Phase 1: Time-to-Response Tracking (Gap 19)

### File Inventory

| Action | File | Responsibility |
|---|---|---|
| Create | `kb-23-decision-cards/internal/models/response_lifecycle.go` | ResponseLifecycle GORM model (5 timestamps + 4 interval metrics) |
| Create | `kb-23-decision-cards/internal/services/response_tracker.go` | RecordDetection, RecordDelivery, RecordAcknowledgment, RecordAction, RecordOutcome, ComputeMetrics |
| Create | `kb-23-decision-cards/internal/services/response_tracker_test.go` | 7 tests |
| Create | `kb-23-decision-cards/internal/api/response_handlers.go` | POST /cards/:id/acknowledge, POST /cards/:id/action, GET /cards/:id/lifecycle, GET /analytics/response-times |
| Modify | `kb-23-decision-cards/internal/api/routes.go` | Add response lifecycle routes |
| Modify | `kb-23-decision-cards/internal/database/connection.go` | AutoMigrate ResponseLifecycle |

**6 files (4 create, 2 modify)**

---

### Task 1: Response lifecycle model

- [ ] **Step 1:** Create `ResponseLifecycle` GORM model:
```go
type ResponseLifecycle struct {
    ID                string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()"`
    CardID            string    `gorm:"type:uuid;uniqueIndex;not null"`
    PatientID         string    `gorm:"size:100;index;not null"`
    TemplateID        string    `gorm:"size:60;not null"`
    ClinicalUrgency   string    `gorm:"size:20"`
    
    // T0-T4 timestamps
    DetectedAt        time.Time `gorm:"not null"`                    // T0: card generated
    DeliveredAt       *time.Time                                      // T1: push/notification sent
    AcknowledgedAt    *time.Time                                      // T2: clinician viewed/acked
    ActionTakenAt     *time.Time                                      // T3: downstream action detected
    OutcomeObservedAt *time.Time                                      // T4: target metric improved
    
    // Interval metrics (computed, in minutes)
    DeliveryLatency      *int   `json:"delivery_latency_min"`        // T1-T0
    ResponseTime         *int   `json:"response_time_min"`           // T2-T1
    ActionLatency        *int   `json:"action_latency_min"`          // T3-T2
    OutcomeLatency       *int   `json:"outcome_latency_min"`         // T4-T3
    TotalLoopTime        *int   `json:"total_loop_time_min"`         // T4-T0
    
    // Action details
    AcknowledgedBy    string    `gorm:"size:100"`
    ActionType        string    `gorm:"size:60"`                     // MED_CHANGE, LAB_ORDER, APPOINTMENT, REFERRAL, CALL_PATIENT
    ActionDetail      string    `gorm:"type:text"`
    OutcomeMetric     string    `gorm:"size:30"`                     // DELTA_HBA1C, DELTA_EGFR, etc.
    OutcomeDelta      *float64
    
    // Status
    Status            string    `gorm:"size:20;default:'DETECTED'"`  // DETECTED, DELIVERED, ACKNOWLEDGED, ACTED, OUTCOME_OBSERVED, EXPIRED
    ExpiredAt         *time.Time                                      // if SLA breached with no action
    
    CreatedAt         time.Time `gorm:"autoCreateTime"`
    UpdatedAt         time.Time `gorm:"autoUpdateTime"`
}
```

- [ ] **Step 2:** Commit: `feat(kb23): response lifecycle model (V5 Plan D Task 1)`

---

### Task 2: Response tracker service

- [ ] **Step 1:** Write 7 tests:
1. `TestTracker_RecordDetection` — creates lifecycle with T0 set, status DETECTED
2. `TestTracker_RecordDelivery` — updates T1, computes DeliveryLatency = T1-T0
3. `TestTracker_RecordAcknowledgment` — updates T2, computes ResponseTime = T2-T1
4. `TestTracker_RecordAction` — updates T3, computes ActionLatency = T3-T2, ActionType set
5. `TestTracker_RecordOutcome` — updates T4, computes OutcomeLatency + TotalLoopTime
6. `TestTracker_FullLoop_AllMetrics` — T0→T1→T2→T3→T4 → all 5 intervals computed correctly
7. `TestTracker_Expired_SLABreach` — no acknowledgment within SLA → status EXPIRED

- [ ] **Step 2:** Implement ResponseTracker with 5 Record methods. Each method: fetch lifecycle by CardID, update timestamp, compute interval, save. ComputeMetrics is called after each update.

- [ ] **Step 3:** Commit: `feat(kb23): response tracker — T0→T4 lifecycle with interval metrics (V5 Plan D Task 2)`

---

### Task 3: Response API handlers + analytics

- [ ] **Step 1:** Create handlers:
- `POST /api/v1/cards/:id/acknowledge` — clinician acknowledges card (sets T2)
- `POST /api/v1/cards/:id/action` — clinician records action taken (sets T3, ActionType, ActionDetail)
- `GET /api/v1/cards/:id/lifecycle` — returns full ResponseLifecycle for a card
- `GET /api/v1/analytics/response-times` — aggregate metrics: median T1-T0, T2-T1, T3-T2, T4-T0 by template type and urgency tier

- [ ] **Step 2:** Wire into KB-23 card persistence: when a card is created, auto-call `tracker.RecordDetection()`. When FHIRCardNotifier fires, call `tracker.RecordDelivery()`.

- [ ] **Step 3:** Commit: `feat(kb23): response lifecycle API + auto-tracking on card persist (V5 Plan D Task 3)`

---

### Task 4: Integration + final commit

- [ ] **Step 1:** Full test sweep KB-23.
- [ ] **Step 2:** Commit: `feat: complete V5 time-to-response tracking`

---

## Phase 2: Predictive Risk Layer (Gap 20) — Architectural Outline

**When:** After 3-6 months of response lifecycle data + IOR outcomes.

**What to build:**
- `kb-26-metabolic-digital-twin/internal/services/risk_predictor.go` — Go inference wrapper
- `kb-26-metabolic-digital-twin/ml/train_risk_model.py` — Python training script
- `kb-26-metabolic-digital-twin/ml/risk_model.onnx` — Exported model for Go inference
- Feature vector: PAI dimensions (5) + MHRI trajectory (4 domain slopes + composite) + phenotype cluster + CGM metrics (TIR, TBR, CV) + BP variability + medication count + engagement trajectory + confounder context + CKM stage + response history (median T2, action rate)
- Target: 30-day hospitalization (binary), 90-day eGFR decline >25% (binary)
- Model: Gradient boosted trees (XGBoost/LightGBM) — interpretable, works with mixed feature types, trains on small datasets (1000+ patients)
- Inference: ONNX Runtime in Go for real-time prediction alongside PAI computation
- Output: `RiskPrediction{Hospitalization30dProb, EGFRDecline90dProb, TopFeatures[]}`

**Not built now because:** Requires sufficient outcome data to train. Building the data collection infrastructure (Phase 1) is the prerequisite.

---

## Phase 3: Closed-Loop Outcome Learning (Gap 21) — Architectural Outline

**When:** After Phase 2 model is validated (AUC ≥ 0.75 on held-out test set).

**What to build:**
- `kb-26-metabolic-digital-twin/internal/services/outcome_feedback.go` — reads IOR outcomes + response tracker data, identifies subgroups where recommendations didn't work
- Feedback mechanism: for each detection template, compute action rate and outcome improvement rate. When a template's action rate is <30% or outcome improvement rate is <40%, flag for clinical review
- Threshold adjustment: when the confounder scorer consistently flags a subgroup (e.g., CKM 4c + eGFR <35 + polypharmacy) as having poor outcomes from intensification, propose adjusting the inertia detector's threshold for that subgroup (recommend deprescribing review instead)
- Output: quarterly "detection effectiveness report" showing per-template detection→action→outcome rates, subgroup analysis, and proposed threshold adjustments. Adjustments are proposed as market-config YAML patches, not auto-applied — clinical governance review required before activation.

**Not built now because:** Requires validated predictive model and sufficient longitudinal outcome data. The infrastructure (IOR outcomes, response tracker, confounder scoring) is in place to collect this data.

---

## Effort Estimate

| Phase | Scope | Expected |
|---|---|---|
| Phase 1: Response tracking | 6 files, 7 tests | 6-8 hours |
| Phase 2: Predictive risk (deferred) | ~8 files + ML pipeline | 2-3 weeks |
| Phase 3: Closed-loop (deferred) | ~4 files + governance | 1-2 weeks |
| **Phase 1 Total** | **6 files, 7 tests** | **6-8 hours** |
