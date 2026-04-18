# Gap 19: Time-to-Response Tracking — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build T0→T4 lifecycle tracking for every detection event — measuring delivery latency, clinician response time, action latency, and outcome improvement — providing the evidence chain that proves the pilot changes outcomes.

**Architecture:** The tracking engine lives in KB-23 (where cards + escalations + worklist already exist) as inline function calls, not Kafka events. Each detection auto-creates a `DetectionLifecycle` record at T0. Existing escalation acknowledgment (T1/T2), worklist resolution (T3), and Gap 16 acute event resolution (T4) update the lifecycle. A metrics aggregator computes per-clinician, per-patient, and system-level rolling statistics. The reporting API exposes pilot KPIs.

**Tech Stack:** Go 1.21 (Gin, GORM) for KB-23 extensions. PostgreSQL for lifecycle persistence. Existing escalation + worklist infrastructure for T1-T3 signal capture. Existing Gap 16 resolution detection for T4.

---

## Existing Infrastructure

| What exists | Where | Relevance |
|---|---|---|
| EscalationEvent with CreatedAt/DeliveredAt/AcknowledgedAt/ActedAt | KB-23 `internal/models/escalation_event.go` | Already captures T0-T3 timestamps — lifecycle wraps and extends this |
| Worklist resolution handler (T2/T3 loop closure) | KB-23 `internal/api/worklist_handlers.go` | Already records acknowledgment + action via Gap 15 loop closure |
| Acute event resolution detection | KB-26 `internal/services/acute_event_handler.go` | Detects when deviation returns to baseline (T4 signal) |
| Card persistence with `notifyFHIR` + `escalationManager` | KB-23 9 persist sites | T0 signal source — every card creation is a detection |
| PAI change event trigger | KB-26 `internal/services/pai_event_trigger.go` | T0 signal source for PAI tier changes |

## File Inventory

### KB-23 — Tracking Engine
| Action | File | Responsibility |
|---|---|---|
| Create | `internal/models/detection_lifecycle.go` | DetectionLifecycle GORM model, LifecycleEvent, LifecycleState constants |
| Create | `internal/services/lifecycle_tracker.go` | RecordT0, RecordT1, RecordT2, RecordT3, RecordT4, state machine transitions |
| Create | `internal/services/lifecycle_tracker_test.go` | 7 tests |
| Create | `internal/services/response_metrics.go` | ComputeClinicianMetrics, ComputeSystemMetrics, rolling windows |
| Create | `internal/services/response_metrics_test.go` | 5 tests |
| Create | `internal/api/tracking_handlers.go` | GET /tracking/detection/:id, GET /tracking/patient/:id, GET /metrics/clinician/:id, GET /metrics/system, GET /metrics/pilot |
| Modify | `internal/api/routes.go` | Add tracking + metrics route groups |
| Modify | `internal/api/server.go` | Add LifecycleTracker dependency |
| Modify | `internal/database/connection.go` | AutoMigrate DetectionLifecycle |
| Modify | `internal/services/escalation_manager.go` | Call tracker.RecordT0 on escalation create |
| Modify | `internal/api/worklist_handlers.go` | Call tracker.RecordT2/T3 on worklist actions |

### Market Configs
| Action | File | Responsibility |
|---|---|---|
| Create | `market-configs/shared/response_tracking_parameters.yaml` | Expected response windows per tier, timeout thresholds, attribution windows |

**Total: 12 files (7 create, 5 modify), ~12 tests**

---

### Task 1: Detection lifecycle models + config YAML

**Files:**
- Create: `kb-23-decision-cards/internal/models/detection_lifecycle.go`
- Create: `market-configs/shared/response_tracking_parameters.yaml`

- [ ] **Step 1:** Create `detection_lifecycle.go` with:

```go
package models

import (
    "time"
    "github.com/google/uuid"
)

// LifecycleState tracks progression of a detection through the response pipeline.
type LifecycleState string

const (
    LifecyclePendingNotification LifecycleState = "PENDING_NOTIFICATION"
    LifecycleNotified            LifecycleState = "NOTIFIED"
    LifecycleAcknowledged        LifecycleState = "ACKNOWLEDGED"
    LifecycleActioned            LifecycleState = "ACTIONED"
    LifecycleResolved            LifecycleState = "RESOLVED"
    LifecycleTimedOut            LifecycleState = "TIMED_OUT"
    LifecycleCancelled           LifecycleState = "CANCELLED"
)

// DetectionLifecycle tracks the T0→T4 lifecycle of a single detection event.
type DetectionLifecycle struct {
    ID                    uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
    DetectionType         string     `gorm:"size:40;index;not null" json:"detection_type"`
    DetectionSubtype      string     `gorm:"size:60" json:"detection_subtype,omitempty"`
    PatientID             string     `gorm:"size:100;index;not null" json:"patient_id"`
    AssignedClinicianID   string     `gorm:"size:100;index" json:"assigned_clinician_id,omitempty"`
    CurrentState          string     `gorm:"size:30;not null;default:'PENDING_NOTIFICATION'" json:"current_state"`
    TierAtDetection       string     `gorm:"size:20" json:"tier_at_detection"`

    // T0-T4 timestamps
    DetectedAt            time.Time  `gorm:"not null" json:"detected_at"`
    DeliveredAt           *time.Time `json:"delivered_at,omitempty"`
    AcknowledgedAt        *time.Time `json:"acknowledged_at,omitempty"`
    ActionedAt            *time.Time `json:"actioned_at,omitempty"`
    ResolvedAt            *time.Time `json:"resolved_at,omitempty"`

    // Computed latencies (milliseconds)
    DeliveryLatencyMs     *int64     `json:"delivery_latency_ms,omitempty"`
    AcknowledgmentLatencyMs *int64   `json:"acknowledgment_latency_ms,omitempty"`
    ActionLatencyMs       *int64     `json:"action_latency_ms,omitempty"`
    OutcomeLatencyMs      *int64     `json:"outcome_latency_ms,omitempty"`
    TotalLatencyMs        *int64     `json:"total_latency_ms,omitempty"`

    // Action details
    ActionType            string     `gorm:"size:60" json:"action_type,omitempty"`
    ActionDetail          string     `gorm:"type:text" json:"action_detail,omitempty"`
    OutcomeDescription    string     `gorm:"type:text" json:"outcome_description,omitempty"`

    // Source references
    CardID                *uuid.UUID `gorm:"type:uuid" json:"card_id,omitempty"`
    EscalationID          *uuid.UUID `gorm:"type:uuid" json:"escalation_id,omitempty"`
    SourceService         string     `gorm:"size:30" json:"source_service"`

    CreatedAt             time.Time  `gorm:"autoCreateTime" json:"created_at"`
    UpdatedAt             time.Time  `gorm:"autoUpdateTime" json:"updated_at"`
}

func (DetectionLifecycle) TableName() string { return "detection_lifecycles" }

// ClinicianResponseMetrics holds computed metrics for one clinician.
type ClinicianResponseMetrics struct {
    ClinicianID             string  `json:"clinician_id"`
    WindowDays              int     `json:"window_days"`
    TotalDetections         int     `json:"total_detections"`
    MedianDeliveryMs        *int64  `json:"median_delivery_ms,omitempty"`
    MedianAcknowledgmentMs  *int64  `json:"median_acknowledgment_ms,omitempty"`
    MedianActionMs          *int64  `json:"median_action_ms,omitempty"`
    ActionCompletionRate    float64 `json:"action_completion_rate"`
    OutcomeRate             float64 `json:"outcome_rate"`
    TeamMedianAckMs         *int64  `json:"team_median_ack_ms,omitempty"`
}

// SystemResponseMetrics holds aggregate metrics.
type SystemResponseMetrics struct {
    WindowDays            int     `json:"window_days"`
    TotalDetections       int     `json:"total_detections"`
    MedianT0toT2Ms        *int64  `json:"median_t0_to_t2_ms,omitempty"`
    MedianT0toT3Ms        *int64  `json:"median_t0_to_t3_ms,omitempty"`
    ActionCompletionRate  float64 `json:"action_completion_rate"`
    OutcomeRate           float64 `json:"outcome_rate"`
    TimeoutRate           float64 `json:"timeout_rate"`
    ByTier                map[string]TierMetrics `json:"by_tier,omitempty"`
}

// TierMetrics holds metrics for a specific escalation tier.
type TierMetrics struct {
    Count                int     `json:"count"`
    MedianAckMs          *int64  `json:"median_ack_ms,omitempty"`
    MedianActionMs       *int64  `json:"median_action_ms,omitempty"`
    ActionCompletionRate float64 `json:"action_completion_rate"`
}

// PilotMetrics holds HCF CHF pilot-specific KPIs.
type PilotMetrics struct {
    TotalDetections              int     `json:"total_detections"`
    DetectionsAcknowledgedInTime int     `json:"detections_acknowledged_in_time"`
    DetectionsWithAction         int     `json:"detections_with_action"`
    MedicationChanges            int     `json:"medication_changes"`
    OutreachCalls                int     `json:"outreach_calls"`
    AppointmentsScheduled        int     `json:"appointments_scheduled"`
    MedianDetectionToActionHrs   float64 `json:"median_detection_to_action_hrs"`
    PatientsWithTimelyAction     int     `json:"patients_with_timely_action"`
    PatientsWithoutTimelyAction  int     `json:"patients_without_timely_action"`
}
```

- [ ] **Step 2:** Create `response_tracking_parameters.yaml`:
```yaml
expected_response_windows:
  SAFETY:
    t1_delivery_minutes: 2
    t2_acknowledgment_minutes: 30
    t3_action_hours: 4
    t4_outcome_hours: 24
  IMMEDIATE:
    t1_delivery_minutes: 5
    t2_acknowledgment_minutes: 120
    t3_action_hours: 24
    t4_outcome_hours: 72
  URGENT:
    t1_delivery_minutes: 30
    t2_acknowledgment_minutes: 1440
    t3_action_hours: 72
    t4_outcome_hours: 168
  ROUTINE:
    t1_delivery_minutes: 240
    t2_acknowledgment_minutes: 10080
    t3_action_hours: 336
    t4_outcome_hours: 720

metrics_windows_days: [7, 30, 90]
min_sample_size: 10

timely_action_definition:
  SAFETY: 240        # 4 hours
  IMMEDIATE: 1440    # 24 hours
  URGENT: 4320       # 72 hours
  ROUTINE: 20160     # 14 days
```

- [ ] **Step 3:** Verify compile + YAML parse. Commit: `feat(kb23): detection lifecycle models + response tracking config (Gap 19 Task 1)`

---

### Task 2: Lifecycle tracker — T0 through T4 state machine

**Files:**
- Create: `kb-23-decision-cards/internal/services/lifecycle_tracker.go`
- Create: `kb-23-decision-cards/internal/services/lifecycle_tracker_test.go`

- [ ] **Step 1:** Write 7 tests:
1. `TestTracker_RecordT0_CreatesLifecycle` — detection event → new DetectionLifecycle with state PENDING_NOTIFICATION, DetectedAt set
2. `TestTracker_RecordT1_SetsDelivered` — T1 event → DeliveredAt set, state NOTIFIED, DeliveryLatencyMs computed
3. `TestTracker_RecordT2_SetsAcknowledged` — T2 → AcknowledgedAt, state ACKNOWLEDGED, AcknowledgmentLatencyMs = T2-T1
4. `TestTracker_RecordT3_SetsActioned` — T3 → ActionedAt, state ACTIONED, ActionLatencyMs = T3-T2, ActionType set
5. `TestTracker_RecordT4_SetsResolved` — T4 → ResolvedAt, state RESOLVED, OutcomeLatencyMs = T4-T3, TotalLatencyMs = T4-T0
6. `TestTracker_FullLifecycle_AllLatencies` — T0→T1→T2→T3→T4 → all 5 latencies computed correctly
7. `TestTracker_OutOfOrder_T2BeforeT1` — T2 arrives before T1 → AcknowledgedAt set, AcknowledgmentLatencyMs nil until T1 arrives, then computed

- [ ] **Step 2:** Implement `LifecycleTracker`:

```go
type LifecycleTracker struct {
    db  *gorm.DB
    log *zap.Logger
}

func NewLifecycleTracker(db *gorm.DB, log *zap.Logger) *LifecycleTracker

// RecordT0 creates a new detection lifecycle.
func (t *LifecycleTracker) RecordT0(
    detectionType, detectionSubtype, patientID, tier, sourceService string,
    cardID, escalationID *uuid.UUID,
) (*models.DetectionLifecycle, error)

// RecordT1 records notification delivery.
func (t *LifecycleTracker) RecordT1(lifecycleID uuid.UUID, deliveredAt time.Time) error

// RecordT2 records clinician acknowledgment.
func (t *LifecycleTracker) RecordT2(lifecycleID uuid.UUID, clinicianID string, acknowledgedAt time.Time) error

// RecordT3 records clinician action.
func (t *LifecycleTracker) RecordT3(lifecycleID uuid.UUID, actionType, actionDetail string, actionedAt time.Time) error

// RecordT4 records outcome observation.
func (t *LifecycleTracker) RecordT4(lifecycleID uuid.UUID, outcomeDescription string, resolvedAt time.Time) error

// FindByPatient returns recent lifecycles for a patient.
func (t *LifecycleTracker) FindByPatient(patientID string, limit int) ([]models.DetectionLifecycle, error)

// FindByEscalation finds the lifecycle linked to an escalation.
func (t *LifecycleTracker) FindByEscalation(escalationID uuid.UUID) (*models.DetectionLifecycle, error)
```

Each Record method: fetch lifecycle by ID, update timestamp, compute latency (ms = new.Sub(previous).Milliseconds()), update state, save. Handle out-of-order: if T2 arrives and T1 is nil, set T2 but leave AcknowledgmentLatencyMs nil. When T1 later arrives, recompute.

For tests: the tracker is testable without DB by using nil db + operating on in-memory lifecycle structs passed by pointer. Alternatively, create a thin interface.

- [ ] **Step 3:** Run tests — all 7 pass. Commit: `feat(kb23): lifecycle tracker — T0→T4 state machine (Gap 19 Task 2)`

---

### Task 3: Wire T0 into card persist + escalation creation

**Files:**
- Modify: `kb-23-decision-cards/internal/services/escalation_manager.go`
- Modify: `kb-23-decision-cards/internal/api/server.go`
- Modify: `kb-23-decision-cards/internal/database/connection.go`

- [ ] **Step 1:** Add `lifecycleTracker *LifecycleTracker` field to EscalationManager. Add setter: `SetLifecycleTracker(t *LifecycleTracker)`.

- [ ] **Step 2:** In `HandleCardCreated`, after creating the EscalationEvent, call:
```go
if m.lifecycleTracker != nil {
    m.lifecycleTracker.RecordT0(
        card.PrimaryDifferentialID,
        string(card.MCUGate),
        card.PatientID.String(),
        result.Tier,
        "KB-23",
        &card.CardID,
        &event.ID,
    )
}
```

- [ ] **Step 3:** Add `DetectionLifecycle` to AutoMigrate in connection.go.

- [ ] **Step 4:** Add `lifecycleTracker` to Server struct with setter.

- [ ] **Step 5:** Build + test. Commit: `feat(kb23): wire T0 into escalation creation (Gap 19 Task 3)`

---

### Task 4: Wire T1/T2/T3 into worklist + escalation acknowledgment path

**Files:**
- Modify: `kb-23-decision-cards/internal/api/worklist_handlers.go`

- [ ] **Step 1:** In the worklist action handler, after the existing Gap 15 loop closure code, add lifecycle tracking:

For ACKNOWLEDGE (T2):
```go
if s.lifecycleTracker != nil {
    if lc, err := s.lifecycleTracker.FindByEscalation(pendingEsc.ID); err == nil && lc != nil {
        s.lifecycleTracker.RecordT2(lc.ID, req.ClinicianID, now)
    }
}
```

For clinical actions like CALL_PATIENT (T3):
```go
if s.lifecycleTracker != nil {
    if lc, err := s.lifecycleTracker.FindByEscalation(actedEsc.ID); err == nil && lc != nil {
        s.lifecycleTracker.RecordT3(lc.ID, req.ActionCode, req.Notes, now)
    }
}
```

- [ ] **Step 2:** The T1 (delivery) timestamp is already captured by the escalation's DeliveredAt field. Add a hook in the escalation manager's notification dispatch:

In `HandleCardCreated`, after successful channel dispatch + `tracker.RecordDelivery`:
```go
if m.lifecycleTracker != nil && lifecycle != nil {
    m.lifecycleTracker.RecordT1(lifecycle.ID, time.Now())
}
```

Store the lifecycle reference from RecordT0 and pass it to the dispatch section.

- [ ] **Step 3:** Build + test. Commit: `feat(kb23): wire T1/T2/T3 into worklist + escalation path (Gap 19 Task 4)`

---

### Task 5: Response metrics aggregation

**Files:**
- Create: `kb-23-decision-cards/internal/services/response_metrics.go`
- Create: `kb-23-decision-cards/internal/services/response_metrics_test.go`

- [ ] **Step 1:** Write 5 tests:
1. `TestMetrics_ClinicianMedians` — 10 lifecycles for clinician X with varied latencies → correct median T0→T2, T0→T3
2. `TestMetrics_ActionCompletionRate` — 8/10 reached T3 → rate 0.80
3. `TestMetrics_OutcomeRate` — 5/8 actioned reached T4 → rate 0.625
4. `TestMetrics_SystemLevel` — 30 lifecycles across 3 clinicians → correct system medians + timeout rate
5. `TestMetrics_PilotKPIs` — lifecycles with specific action types → MedicationChanges, OutreachCalls counted correctly

- [ ] **Step 2:** Implement:

```go
type ResponseMetricsService struct {
    db *gorm.DB
}

func NewResponseMetricsService(db *gorm.DB) *ResponseMetricsService

// ComputeClinicianMetrics returns rolling window metrics for a clinician.
func (s *ResponseMetricsService) ComputeClinicianMetrics(clinicianID string, windowDays int) (*models.ClinicianResponseMetrics, error)

// ComputeSystemMetrics returns system-level aggregate metrics.
func (s *ResponseMetricsService) ComputeSystemMetrics(windowDays int) (*models.SystemResponseMetrics, error)

// ComputePilotMetrics returns HCF CHF pilot-specific KPIs.
func (s *ResponseMetricsService) ComputePilotMetrics(windowDays int) (*models.PilotMetrics, error)
```

ComputeClinicianMetrics: query DetectionLifecycle WHERE assigned_clinician_id = ? AND detected_at > now - windowDays. Compute median of AcknowledgmentLatencyMs (for non-nil values), median ActionLatencyMs. ActionCompletionRate = count(actioned_at IS NOT NULL) / total. OutcomeRate = count(resolved_at IS NOT NULL) / count(actioned_at IS NOT NULL).

ComputePilotMetrics: query all lifecycles in window. Count by ActionType: MEDICATION_REVIEW/PRESCRIPTION_REVIEW → MedicationChanges, CALL_PATIENT/TELECONSULT → OutreachCalls, SCHEDULE_APPOINTMENT/SCHEDULE_CLINIC → AppointmentsScheduled. MedianDetectionToActionHrs = median(ActionLatencyMs) / 3600000.

Use `medianInt64(values)` helper that sorts and returns middle value.

- [ ] **Step 3:** Run tests — all 5 pass. Commit: `feat(kb23): response metrics — clinician + system + pilot KPIs (Gap 19 Task 5)`

---

### Task 6: Tracking + metrics API handlers

**Files:**
- Create: `kb-23-decision-cards/internal/api/tracking_handlers.go`
- Modify: `kb-23-decision-cards/internal/api/routes.go`

- [ ] **Step 1:** Create 5 handlers:
```go
// GET /api/v1/tracking/detection/:id — full lifecycle for one detection
func (s *Server) getDetectionLifecycle(c *gin.Context)

// GET /api/v1/tracking/patient/:patientId — recent lifecycles for patient
func (s *Server) getPatientLifecycles(c *gin.Context)

// GET /api/v1/metrics/clinician/:clinicianId?window=30 — clinician response metrics
func (s *Server) getClinicianMetrics(c *gin.Context)

// GET /api/v1/metrics/system?window=30 — system-level metrics
func (s *Server) getSystemMetrics(c *gin.Context)

// GET /api/v1/metrics/pilot?window=90 — HCF CHF pilot KPIs
func (s *Server) getPilotMetrics(c *gin.Context)
```

- [ ] **Step 2:** Add routes:
```go
tracking := v1.Group("/tracking")
{
    tracking.GET("/detection/:id", s.getDetectionLifecycle)
    tracking.GET("/patient/:patientId", s.getPatientLifecycles)
}
metrics := v1.Group("/metrics")
{
    metrics.GET("/clinician/:clinicianId", s.getClinicianMetrics)
    metrics.GET("/system", s.getSystemMetrics)
    metrics.GET("/pilot", s.getPilotMetrics)
}
```

- [ ] **Step 3:** Build + test. Commit: `feat(kb23): tracking + metrics API endpoints (Gap 19 Task 6)`

---

### Task 7: Integration test + final commit

- [ ] **Step 1:** Full test sweep KB-23.
- [ ] **Step 2:** Verify YAML parses.
- [ ] **Step 3:** Commit: `feat: complete Gap 19 time-to-response tracking`
- [ ] **Step 4:** Push to origin.

---

## Verification Questions

1. Does card creation produce a T0 lifecycle record? (yes / wired in Task 3)
2. Does worklist ACKNOWLEDGE set T2 on the lifecycle? (yes / wired in Task 4)
3. Does worklist CALL_PATIENT set T3 on the lifecycle? (yes / wired in Task 4)
4. Does T0→T1→T2→T3→T4 produce all 5 latency values? (yes / test)
5. Does out-of-order T2-before-T1 handle gracefully? (yes / test)
6. Does clinician median computation use correct rolling window? (yes / test)
7. Does pilot KPIs count medication changes correctly? (yes / test)
8. Are all KB-23 test suites green? (yes / sweep)

## Effort Estimate

| Task | Scope | Expected |
|---|---|---|
| Task 1: Models + YAML | 2 files | 1-2 hours |
| Task 2: Lifecycle tracker (7 tests) | 2 files | 2-3 hours |
| Task 3: Wire T0 | 3 files modified | 1 hour |
| Task 4: Wire T1/T2/T3 | 1 file modified | 1-2 hours |
| Task 5: Metrics aggregation (5 tests) | 2 files | 2-3 hours |
| Task 6: API handlers | 2 files | 1-2 hours |
| Task 7: Integration + commit | sweep | 30 min |
| **Total** | **~12 files, ~12 tests** | **~9-13 hours** |

---

## Sprint 2 Deferred Items

| Component | Reason |
|-----------|--------|
| New KB-27 service | KB-23 inline is sufficient for 500-patient pilot |
| Kafka event consumer | Inline function calls work; Kafka adds latency not value for Sprint 1 |
| Outcome attribution engine (T4 probabilistic attribution) | Needs 3-6 months of outcome data to tune |
| TimescaleDB for time-series aggregation | PostgreSQL sufficient for pilot volume |
| Grafana dashboards (5 dashboards) | API endpoints support custom dashboards; Grafana is ops infrastructure |
| Timeout checker background goroutine | Gap 15 escalation timeout already covers the critical path |
| Signal source modifications (9 files across KB-20/KB-26) | T0 from escalation manager is the primary path; other sources add coverage later |
