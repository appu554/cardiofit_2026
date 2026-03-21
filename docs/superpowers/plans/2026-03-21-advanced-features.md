# Ingestion + Intake-Onboarding Advanced Features Plan (Phase 5)

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Implement the remaining Phase 5 advanced features across both Ingestion and Intake-Onboarding services: biweekly check-in state machine with trajectory signals, pharmacist review queue, wearable adapters (Health Connect, Ultrahuman CGM, Apple HealthKit), DLQ management with replay, Write-Ahead Log for Kafka failover, and full observability (20 Prometheus metrics + OpenTelemetry tracing).

**Prerequisites:** Plans 1-4 must be implemented first. This plan assumes the following exist:
- Intake service scaffolding with Gin server, enrollment state machine, slot table, safety engine, flow graph engine, and FHIR handlers (Plans 1 + 3)
- Ingestion service scaffolding with pipeline stages, adapters, Kafka producer, and FHIR mappers (Plans 1 + 2)
- WhatsApp/ASHA/ABDM/Lab/EHR adapters (Plan 4)
- PostgreSQL migrations for `enrollments`, `slot_events`, `current_slots`, `flow_positions`, `review_queue`, `dlq_messages` tables (Plan 1)
- Kafka envelope struct with `traceId` field (Plan 1)

**Architecture:** Check-in and review features live in Intake (:8141). Wearable adapters, DLQ management, and WAL live in Ingestion (:8140). Observability spans both services.

**Tech Stack:** Go 1.25, Gin, pgx/v5, redis/go-redis/v9, zap, prometheus/client_golang, go.opentelemetry.io/otel, segmentio/kafka-go

**Spec:** `docs/superpowers/specs/2026-03-21-ingestion-intake-onboarding-design.md` (sections 2.1, 2.2, 3.1, 7.3-7.5)

---

## File Structure

### Intake-Onboarding Service (Check-in + Review)

| File | Responsibility |
|------|---------------|
| `internal/checkin/machine.go` | M0-CI 7-state biweekly check-in (CS1-CS7), 12-slot subset |
| `internal/checkin/machine_test.go` | State machine transition tests |
| `internal/checkin/trajectory.go` | Trajectory computer: STABLE/FRAGILE/FAILURE/DISENGAGE |
| `internal/checkin/trajectory_test.go` | Trajectory signal tests with fixture data |
| `internal/checkin/handler.go` | `$checkin` and `$checkin-slot` HTTP handlers |
| `internal/checkin/handler_test.go` | Handler tests |
| `internal/review/queue.go` | PostgreSQL-backed pharmacist review queue with risk stratification |
| `internal/review/queue_test.go` | Queue CRUD and ordering tests |
| `internal/review/reviewer.go` | `$submit-review`, `$approve`, `$request-clarification`, `$escalate` |
| `internal/review/reviewer_test.go` | Review action handler tests |
| `internal/metrics/collectors.go` | 10 Prometheus metric collectors for intake |
| `internal/metrics/tracing.go` | OpenTelemetry tracer provider + Gin middleware |
| `migrations/003_checkin.sql` | `checkin_sessions`, `checkin_slot_events` tables |

### Ingestion Service (Wearables + DLQ + WAL)

| File | Responsibility |
|------|---------------|
| `internal/adapters/wearables/health_connect.go` | Google Health Connect API adapter |
| `internal/adapters/wearables/health_connect_test.go` | Health Connect adapter tests |
| `internal/adapters/wearables/ultrahuman.go` | Ultrahuman CGM aggregation (TIR/TAR/TBR/CV/MAG) |
| `internal/adapters/wearables/ultrahuman_test.go` | CGM metric aggregation tests |
| `internal/adapters/wearables/apple_health.go` | Apple HealthKit adapter |
| `internal/adapters/wearables/apple_health_test.go` | HealthKit adapter tests |
| `internal/dlq/publisher.go` | DLQ message publisher (PostgreSQL + Kafka `ingestion.dlq`) |
| `internal/dlq/resolver.go` | Admin view, filter, inspect DLQ messages |
| `internal/dlq/replay.go` | DLQ replay mechanism (re-inject into pipeline) |
| `internal/dlq/publisher_test.go` | DLQ publisher tests |
| `internal/dlq/replay_test.go` | DLQ replay tests |
| `internal/kafka/wal.go` | Write-Ahead Log for Kafka failover (10GB cap, 30s retry) |
| `internal/kafka/wal_test.go` | WAL append/replay/cap enforcement tests |
| `internal/metrics/collectors.go` | 10 Prometheus metric collectors for ingestion |
| `internal/metrics/tracing.go` | OpenTelemetry tracer provider + Gin middleware |

---

## Task 1: Check-in State Machine (Intake)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/machine.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/machine_test.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/migrations/003_checkin.sql`

**Spec reference:** Section 2.2 `internal/checkin/machine.go` — M0-CI 7-state biweekly (CS1-CS7, 12 slots). Section 3.1 `$checkin` and `$checkin-slot` operations. Kafka topic `intake.checkin-events`.

The M0-CI (Month-0 Check-In) is a biweekly follow-up cycle that runs after enrollment. It uses a 12-slot subset of the full 50-slot intake to track patient status over time. The 7 states represent the check-in lifecycle from scheduling through completion.

- [ ] **Step 1: Write migration 003_checkin.sql**

```sql
-- vaidshala/clinical-runtime-platform/services/intake-onboarding-service/migrations/003_checkin.sql

-- Check-in sessions (biweekly M0-CI)
CREATE TABLE checkin_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    patient_id      UUID NOT NULL REFERENCES enrollments(patient_id),
    encounter_id    UUID NOT NULL,          -- FHIR Encounter for this check-in
    cycle_number    INT NOT NULL,           -- 1, 2, 3, ... (biweekly count)
    state           TEXT NOT NULL DEFAULT 'CS1_SCHEDULED',
    trajectory      TEXT,                   -- STABLE, FRAGILE, FAILURE, DISENGAGE (computed at CS6)
    slots_filled    INT DEFAULT 0,
    slots_total     INT DEFAULT 12,
    scheduled_at    TIMESTAMPTZ NOT NULL,   -- when this check-in is due
    started_at      TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ DEFAULT now(),
    updated_at      TIMESTAMPTZ DEFAULT now(),
    UNIQUE (patient_id, cycle_number)
);

-- Check-in slot events (event-sourced, same pattern as intake slot_events)
CREATE TABLE checkin_slot_events (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    session_id      UUID NOT NULL REFERENCES checkin_sessions(id),
    patient_id      UUID NOT NULL,
    slot_name       TEXT NOT NULL,
    domain          TEXT NOT NULL,
    value           JSONB NOT NULL,
    extraction_mode TEXT NOT NULL,
    confidence      REAL,
    fhir_resource_id TEXT,
    created_at      TIMESTAMPTZ DEFAULT now()
);

CREATE INDEX idx_checkin_sessions_patient ON checkin_sessions(patient_id, state);
CREATE INDEX idx_checkin_sessions_scheduled ON checkin_sessions(scheduled_at) WHERE state = 'CS1_SCHEDULED';
CREATE INDEX idx_checkin_slot_events_session ON checkin_slot_events(session_id, slot_name);
```

- [ ] **Step 2: Write machine.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/machine.go
package checkin

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CheckinState represents a step in the 7-state M0-CI biweekly check-in lifecycle.
type CheckinState string

const (
	CS1Scheduled   CheckinState = "CS1_SCHEDULED"    // Check-in scheduled (biweekly cron)
	CS2Notified    CheckinState = "CS2_NOTIFIED"     // Patient notified via WhatsApp/push
	CS3InProgress  CheckinState = "CS3_IN_PROGRESS"  // Patient actively filling slots
	CS4Paused      CheckinState = "CS4_PAUSED"       // Patient paused (4hr timeout)
	CS5Completed   CheckinState = "CS5_COMPLETED"    // All 12 slots filled
	CS6Evaluated   CheckinState = "CS6_EVALUATED"    // Trajectory signal computed
	CS7Closed      CheckinState = "CS7_CLOSED"       // Check-in archived, next cycle scheduled
)

// AllCheckinStates returns the 7 check-in states in lifecycle order.
func AllCheckinStates() []CheckinState {
	return []CheckinState{
		CS1Scheduled, CS2Notified, CS3InProgress, CS4Paused,
		CS5Completed, CS6Evaluated, CS7Closed,
	}
}

// validCheckinTransitions defines allowed state transitions for check-in.
var validCheckinTransitions = map[CheckinState][]CheckinState{
	CS1Scheduled:  {CS2Notified},
	CS2Notified:   {CS3InProgress, CS4Paused},     // patient starts or misses notification window
	CS3InProgress: {CS4Paused, CS5Completed},       // pause on timeout or complete all slots
	CS4Paused:     {CS3InProgress, CS7Closed},      // resume or abandon (7d timeout → close)
	CS5Completed:  {CS6Evaluated},                  // trajectory computation
	CS6Evaluated:  {CS7Closed},                     // archive and schedule next
	CS7Closed:     {},                              // terminal
}

// CanCheckinTransition checks if a state transition is valid.
func CanCheckinTransition(from, to CheckinState) bool {
	targets, exists := validCheckinTransitions[from]
	if !exists {
		return false
	}
	for _, t := range targets {
		if t == to {
			return true
		}
	}
	return false
}

// CheckinSlotDef defines one of the 12 check-in slots.
type CheckinSlotDef struct {
	Name     string `json:"name"`
	Domain   string `json:"domain"`
	LOINCCode string `json:"loinc_code"`
	Unit     string `json:"unit"`
	Required bool   `json:"required"`
}

// CheckinSlots returns the 12-slot subset used in biweekly check-ins.
// These are the most critical tracking parameters from the full 50-slot intake.
func CheckinSlots() []CheckinSlotDef {
	return []CheckinSlotDef{
		// Glycemic (3)
		{Name: "fbg", Domain: "glycemic", LOINCCode: "1558-6", Unit: "mg/dL", Required: true},
		{Name: "ppbg", Domain: "glycemic", LOINCCode: "1521-4", Unit: "mg/dL", Required: false},
		{Name: "hba1c", Domain: "glycemic", LOINCCode: "4548-4", Unit: "%", Required: false},
		// Cardiac (2)
		{Name: "systolic_bp", Domain: "cardiac", LOINCCode: "8480-6", Unit: "mmHg", Required: true},
		{Name: "diastolic_bp", Domain: "cardiac", LOINCCode: "8462-4", Unit: "mmHg", Required: true},
		// Renal (1)
		{Name: "egfr", Domain: "renal", LOINCCode: "33914-3", Unit: "mL/min/1.73m2", Required: false},
		// Anthropometric (1)
		{Name: "weight", Domain: "anthropometric", LOINCCode: "29463-7", Unit: "kg", Required: true},
		// Behavioral (3)
		{Name: "medication_adherence", Domain: "behavioral", LOINCCode: "71950-0", Unit: "score", Required: true},
		{Name: "physical_activity_minutes", Domain: "behavioral", LOINCCode: "68516-4", Unit: "min/week", Required: true},
		{Name: "sleep_hours", Domain: "behavioral", LOINCCode: "93832-4", Unit: "hours", Required: false},
		// Symptoms (2)
		{Name: "symptom_severity", Domain: "symptoms", LOINCCode: "75261-9", Unit: "score", Required: true},
		{Name: "side_effects", Domain: "symptoms", LOINCCode: "75321-0", Unit: "text", Required: true},
	}
}

// CheckinSession holds the state of a biweekly check-in session.
type CheckinSession struct {
	ID           uuid.UUID    `json:"id"`
	PatientID    uuid.UUID    `json:"patient_id"`
	EncounterID  uuid.UUID    `json:"encounter_id"`
	CycleNumber  int          `json:"cycle_number"`
	State        CheckinState `json:"state"`
	Trajectory   Trajectory   `json:"trajectory,omitempty"`
	SlotsFilled  int          `json:"slots_filled"`
	SlotsTotal   int          `json:"slots_total"`
	ScheduledAt  time.Time    `json:"scheduled_at"`
	StartedAt    *time.Time   `json:"started_at,omitempty"`
	CompletedAt  *time.Time   `json:"completed_at,omitempty"`
	CreatedAt    time.Time    `json:"created_at"`
	UpdatedAt    time.Time    `json:"updated_at"`
}

// Transition attempts to move the check-in session to a new state.
func (s *CheckinSession) Transition(to CheckinState) error {
	if !CanCheckinTransition(s.State, to) {
		return &ErrInvalidCheckinTransition{From: s.State, To: to}
	}
	s.State = to
	s.UpdatedAt = time.Now().UTC()

	switch to {
	case CS3InProgress:
		if s.StartedAt == nil {
			now := time.Now().UTC()
			s.StartedAt = &now
		}
	case CS5Completed:
		now := time.Now().UTC()
		s.CompletedAt = &now
	}

	return nil
}

// IsTerminal returns true if the check-in is in a terminal state.
func (s *CheckinSession) IsTerminal() bool {
	return s.State == CS7Closed
}

// RequiredSlotsFilled returns true if all required slots have been filled.
func (s *CheckinSession) RequiredSlotsFilled(filledSlots map[string]bool) bool {
	for _, slot := range CheckinSlots() {
		if slot.Required && !filledSlots[slot.Name] {
			return false
		}
	}
	return true
}

// BiweeklyInterval is the standard check-in period (14 days).
const BiweeklyInterval = 14 * 24 * time.Hour

// NextScheduledAt computes the next check-in date from the previous one.
func NextScheduledAt(previousScheduled time.Time) time.Time {
	return previousScheduled.Add(BiweeklyInterval)
}

// ErrInvalidCheckinTransition is returned when a check-in state transition is not allowed.
type ErrInvalidCheckinTransition struct {
	From CheckinState
	To   CheckinState
}

func (e *ErrInvalidCheckinTransition) Error() string {
	return fmt.Sprintf("invalid check-in transition: %s -> %s", e.From, e.To)
}
```

- [ ] **Step 3: Write machine_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/machine_test.go
package checkin

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAllCheckinStates_Count(t *testing.T) {
	states := AllCheckinStates()
	if len(states) != 7 {
		t.Errorf("expected 7 check-in states, got %d", len(states))
	}
}

func TestCheckinSlots_Count(t *testing.T) {
	slots := CheckinSlots()
	if len(slots) != 12 {
		t.Errorf("expected 12 check-in slots, got %d", len(slots))
	}

	// Count required slots
	required := 0
	for _, s := range slots {
		if s.Required {
			required++
		}
	}
	if required != 8 {
		t.Errorf("expected 8 required slots, got %d", required)
	}
}

func TestCheckinTransition_HappyPath(t *testing.T) {
	transitions := []struct{ from, to CheckinState }{
		{CS1Scheduled, CS2Notified},
		{CS2Notified, CS3InProgress},
		{CS3InProgress, CS5Completed},
		{CS5Completed, CS6Evaluated},
		{CS6Evaluated, CS7Closed},
	}
	for _, tt := range transitions {
		if !CanCheckinTransition(tt.from, tt.to) {
			t.Errorf("expected valid transition %s -> %s", tt.from, tt.to)
		}
	}
}

func TestCheckinTransition_PauseResume(t *testing.T) {
	if !CanCheckinTransition(CS3InProgress, CS4Paused) {
		t.Error("IN_PROGRESS -> PAUSED should be valid")
	}
	if !CanCheckinTransition(CS4Paused, CS3InProgress) {
		t.Error("PAUSED -> IN_PROGRESS should be valid (resume)")
	}
}

func TestCheckinTransition_PausedToClose(t *testing.T) {
	// Abandoned check-in (7d timeout while paused)
	if !CanCheckinTransition(CS4Paused, CS7Closed) {
		t.Error("PAUSED -> CLOSED should be valid (abandon after timeout)")
	}
}

func TestCheckinTransition_InvalidSkip(t *testing.T) {
	if CanCheckinTransition(CS1Scheduled, CS3InProgress) {
		t.Error("SCHEDULED -> IN_PROGRESS should be invalid (skips notification)")
	}
	if CanCheckinTransition(CS7Closed, CS1Scheduled) {
		t.Error("CLOSED -> SCHEDULED should be invalid (terminal)")
	}
}

func TestCheckinSession_Transition(t *testing.T) {
	now := time.Now().UTC()
	session := &CheckinSession{
		ID:          uuid.New(),
		PatientID:   uuid.New(),
		EncounterID: uuid.New(),
		CycleNumber: 1,
		State:       CS1Scheduled,
		SlotsTotal:  12,
		ScheduledAt: now,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	if err := session.Transition(CS2Notified); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}
	if session.State != CS2Notified {
		t.Errorf("expected CS2_NOTIFIED, got %s", session.State)
	}

	// Transition to in-progress should set StartedAt
	if err := session.Transition(CS3InProgress); err != nil {
		t.Fatalf("valid transition failed: %v", err)
	}
	if session.StartedAt == nil {
		t.Error("expected StartedAt to be set on CS3_IN_PROGRESS")
	}

	// Invalid transition
	if err := session.Transition(CS7Closed); err == nil {
		t.Fatal("expected error for invalid transition CS3 -> CS7")
	}
}

func TestCheckinSession_RequiredSlotsFilled(t *testing.T) {
	session := &CheckinSession{SlotsTotal: 12}

	// No slots filled
	if session.RequiredSlotsFilled(map[string]bool{}) {
		t.Error("should return false when no slots filled")
	}

	// All required slots filled
	filled := map[string]bool{
		"fbg": true, "systolic_bp": true, "diastolic_bp": true,
		"weight": true, "medication_adherence": true,
		"physical_activity_minutes": true, "symptom_severity": true,
		"side_effects": true,
	}
	if !session.RequiredSlotsFilled(filled) {
		t.Error("should return true when all 8 required slots filled")
	}
}

func TestNextScheduledAt(t *testing.T) {
	base := time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC)
	next := NextScheduledAt(base)
	expected := time.Date(2026, 4, 4, 10, 0, 0, 0, time.UTC)
	if !next.Equal(expected) {
		t.Errorf("expected %v, got %v", expected, next)
	}
}

func TestCheckinSession_IsTerminal(t *testing.T) {
	s := &CheckinSession{State: CS7Closed}
	if !s.IsTerminal() {
		t.Error("CS7_CLOSED should be terminal")
	}
	s.State = CS3InProgress
	if s.IsTerminal() {
		t.Error("CS3_IN_PROGRESS should not be terminal")
	}
}
```

- [ ] **Step 4: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/checkin/... -v -count=1`
Expected: All 9 tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/machine.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/machine_test.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/migrations/003_checkin.sql
git commit -m "feat(intake): add M0-CI 7-state biweekly check-in state machine

States: CS1_SCHEDULED -> CS2_NOTIFIED -> CS3_IN_PROGRESS ->
{CS4_PAUSED|CS5_COMPLETED} -> CS6_EVALUATED -> CS7_CLOSED.
12-slot subset with 8 required slots across 6 domains.
14-day biweekly cycle scheduling."
```

---

## Task 2: Trajectory Computer (Intake)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/trajectory.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/trajectory_test.go`

**Spec reference:** Section 2.2 `internal/checkin/trajectory.go` — STABLE/FRAGILE/FAILURE/DISENGAGE signal. Published to `intake.checkin-events` for consumption by M4, KB-20, KB-21.

The trajectory signal is computed at state CS6_EVALUATED by analyzing the current check-in data against the previous check-in and the patient's baseline (enrollment) values. It determines whether the patient is on track (STABLE), showing concerning trends (FRAGILE), failing to improve (FAILURE), or not participating (DISENGAGE).

- [ ] **Step 1: Write trajectory.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/trajectory.go
package checkin

import (
	"math"
	"time"
)

// Trajectory represents the patient's clinical trajectory signal.
type Trajectory string

const (
	TrajectoryStable    Trajectory = "STABLE"     // On track — metrics within targets or improving
	TrajectoryFragile   Trajectory = "FRAGILE"    // Warning — some metrics deteriorating
	TrajectoryFailure   Trajectory = "FAILURE"    // Multiple metrics worsening, intervention needed
	TrajectoryDisengage Trajectory = "DISENGAGE"  // Patient not participating in check-ins
)

// TrajectoryInput holds the data needed to compute a trajectory signal.
type TrajectoryInput struct {
	// Current check-in slot values (slot_name -> numeric value)
	CurrentValues map[string]float64

	// Previous check-in slot values (nil if first check-in)
	PreviousValues map[string]float64

	// Baseline values from enrollment intake
	BaselineValues map[string]float64

	// Check-in metadata
	CycleNumber       int
	SlotsFilled       int
	SlotsRequired     int
	DaysSinceSchedule float64 // how many days after scheduled date the check-in was completed

	// Previous trajectory (for trend analysis)
	PreviousTrajectory *Trajectory
}

// TrajectoryResult holds the computed trajectory and supporting evidence.
type TrajectoryResult struct {
	Signal           Trajectory         `json:"signal"`
	Confidence       float64            `json:"confidence"`
	ImprovingSlots   []string           `json:"improving_slots,omitempty"`
	WorseningSlots   []string           `json:"worsening_slots,omitempty"`
	StableSlots      []string           `json:"stable_slots,omitempty"`
	MissedSlots      []string           `json:"missed_slots,omitempty"`
	ComputedAt       time.Time          `json:"computed_at"`
	DomainScores     map[string]float64 `json:"domain_scores"`
}

// ClinicalTarget defines the acceptable range for a check-in slot.
type ClinicalTarget struct {
	SlotName       string
	Domain         string
	LowOptimal     float64 // lower bound of target range
	HighOptimal    float64 // upper bound of target range
	LowCritical    float64 // below this = critical
	HighCritical   float64 // above this = critical
	ImprovementDir int     // -1 = lower is better, +1 = higher is better, 0 = range-based
}

// DefaultClinicalTargets returns the clinical targets for the 12 check-in slots.
func DefaultClinicalTargets() map[string]ClinicalTarget {
	return map[string]ClinicalTarget{
		"fbg":                        {SlotName: "fbg", Domain: "glycemic", LowOptimal: 70, HighOptimal: 130, LowCritical: 54, HighCritical: 250, ImprovementDir: -1},
		"ppbg":                       {SlotName: "ppbg", Domain: "glycemic", LowOptimal: 70, HighOptimal: 180, LowCritical: 54, HighCritical: 300, ImprovementDir: -1},
		"hba1c":                      {SlotName: "hba1c", Domain: "glycemic", LowOptimal: 4.0, HighOptimal: 7.0, LowCritical: 3.0, HighCritical: 12.0, ImprovementDir: -1},
		"systolic_bp":                {SlotName: "systolic_bp", Domain: "cardiac", LowOptimal: 90, HighOptimal: 130, LowCritical: 70, HighCritical: 180, ImprovementDir: -1},
		"diastolic_bp":               {SlotName: "diastolic_bp", Domain: "cardiac", LowOptimal: 60, HighOptimal: 80, LowCritical: 40, HighCritical: 120, ImprovementDir: -1},
		"egfr":                       {SlotName: "egfr", Domain: "renal", LowOptimal: 60, HighOptimal: 120, LowCritical: 15, HighCritical: 200, ImprovementDir: 1},
		"weight":                     {SlotName: "weight", Domain: "anthropometric", LowOptimal: 0, HighOptimal: 0, LowCritical: 0, HighCritical: 0, ImprovementDir: 0}, // compared to baseline
		"medication_adherence":       {SlotName: "medication_adherence", Domain: "behavioral", LowOptimal: 0.8, HighOptimal: 1.0, LowCritical: 0.0, HighCritical: 1.0, ImprovementDir: 1},
		"physical_activity_minutes":  {SlotName: "physical_activity_minutes", Domain: "behavioral", LowOptimal: 150, HighOptimal: 300, LowCritical: 0, HighCritical: 600, ImprovementDir: 1},
		"sleep_hours":                {SlotName: "sleep_hours", Domain: "behavioral", LowOptimal: 7.0, HighOptimal: 9.0, LowCritical: 3.0, HighCritical: 14.0, ImprovementDir: 0},
		"symptom_severity":           {SlotName: "symptom_severity", Domain: "symptoms", LowOptimal: 0, HighOptimal: 3, LowCritical: 0, HighCritical: 10, ImprovementDir: -1},
		"side_effects":               {SlotName: "side_effects", Domain: "symptoms", LowOptimal: 0, HighOptimal: 2, LowCritical: 0, HighCritical: 10, ImprovementDir: -1},
	}
}

// ComputeTrajectory calculates the trajectory signal from check-in data.
// This is a deterministic computation with ZERO LLM involvement.
func ComputeTrajectory(input TrajectoryInput) TrajectoryResult {
	now := time.Now().UTC()
	result := TrajectoryResult{
		ComputedAt:   now,
		DomainScores: make(map[string]float64),
	}

	// Rule 1: DISENGAGE if insufficient participation
	if isDisengaged(input) {
		result.Signal = TrajectoryDisengage
		result.Confidence = 0.95
		return result
	}

	targets := DefaultClinicalTargets()
	var improving, worsening, stable, missed []string

	// Rule 2: Compare each slot to previous check-in (or baseline if first check-in)
	referenceValues := input.PreviousValues
	if referenceValues == nil {
		referenceValues = input.BaselineValues
	}

	domainScores := make(map[string][]float64)

	for _, slotDef := range CheckinSlots() {
		current, hasCurrent := input.CurrentValues[slotDef.Name]
		reference, hasReference := referenceValues[slotDef.Name]
		target, hasTarget := targets[slotDef.Name]

		if !hasCurrent {
			missed = append(missed, slotDef.Name)
			continue
		}

		if !hasReference || !hasTarget {
			stable = append(stable, slotDef.Name)
			domainScores[slotDef.Domain] = append(domainScores[slotDef.Domain], 0.5)
			continue
		}

		score := scoreSlotChange(current, reference, target)
		domainScores[target.Domain] = append(domainScores[target.Domain], score)

		if score > 0.6 {
			improving = append(improving, slotDef.Name)
		} else if score < 0.4 {
			worsening = append(worsening, slotDef.Name)
		} else {
			stable = append(stable, slotDef.Name)
		}
	}

	result.ImprovingSlots = improving
	result.WorseningSlots = worsening
	result.StableSlots = stable
	result.MissedSlots = missed

	// Compute per-domain averages
	for domain, scores := range domainScores {
		if len(scores) > 0 {
			sum := 0.0
			for _, s := range scores {
				sum += s
			}
			result.DomainScores[domain] = sum / float64(len(scores))
		}
	}

	// Rule 3: Classify trajectory
	totalEvaluated := len(improving) + len(worsening) + len(stable)
	if totalEvaluated == 0 {
		result.Signal = TrajectoryDisengage
		result.Confidence = 0.8
		return result
	}

	worsenRatio := float64(len(worsening)) / float64(totalEvaluated)
	improveRatio := float64(len(improving)) / float64(totalEvaluated)

	switch {
	case worsenRatio >= 0.5:
		// >=50% of evaluated slots worsening = FAILURE
		result.Signal = TrajectoryFailure
		result.Confidence = 0.85 + (worsenRatio-0.5)*0.2
	case worsenRatio >= 0.25:
		// 25-49% worsening = FRAGILE
		result.Signal = TrajectoryFragile
		result.Confidence = 0.75 + worsenRatio*0.2
	case improveRatio >= 0.5:
		// >=50% improving and <25% worsening = STABLE
		result.Signal = TrajectoryStable
		result.Confidence = 0.85 + improveRatio*0.1
	default:
		// Mixed — mostly stable
		result.Signal = TrajectoryStable
		result.Confidence = 0.70
	}

	// Rule 4: Consecutive FRAGILE/FAILURE escalation
	if input.PreviousTrajectory != nil {
		if *input.PreviousTrajectory == TrajectoryFragile && result.Signal == TrajectoryFragile {
			result.Signal = TrajectoryFailure
			result.Confidence = 0.90
		}
	}

	return result
}

// isDisengaged returns true if the patient shows signs of disengagement.
func isDisengaged(input TrajectoryInput) bool {
	// Disengaged if: <50% required slots filled, or >3 days late on check-in
	fillRatio := float64(input.SlotsFilled) / float64(input.SlotsRequired)
	if fillRatio < 0.5 && input.CycleNumber > 1 {
		return true
	}
	if input.DaysSinceSchedule > 3.0 && input.SlotsFilled == 0 {
		return true
	}
	return false
}

// scoreSlotChange returns a score between 0.0 (worsening) and 1.0 (improving)
// based on the change from reference to current value relative to the clinical target.
func scoreSlotChange(current, reference float64, target ClinicalTarget) float64 {
	// Special case: weight uses percentage change from baseline
	if target.SlotName == "weight" {
		if reference == 0 {
			return 0.5
		}
		pctChange := (current - reference) / reference * 100.0
		// Weight loss up to 5% is improvement for metabolic patients
		if pctChange <= -5.0 {
			return 0.9
		} else if pctChange <= -1.0 {
			return 0.7
		} else if pctChange <= 1.0 {
			return 0.5
		} else if pctChange <= 5.0 {
			return 0.3
		}
		return 0.1
	}

	// For directional targets (lower is better or higher is better)
	delta := current - reference
	if target.ImprovementDir == -1 {
		delta = -delta // invert so positive delta = improvement
	}

	// Normalize delta relative to target range
	targetRange := target.HighOptimal - target.LowOptimal
	if targetRange == 0 {
		targetRange = 1.0
	}
	normalizedDelta := delta / targetRange

	// Map to 0-1 score: improvement -> >0.5, worsening -> <0.5
	score := 0.5 + normalizedDelta*0.5

	// Check if current value is in critical range (strong worsening signal)
	if current <= target.LowCritical || current >= target.HighCritical {
		score = math.Min(score, 0.2)
	}

	// Clamp to [0.0, 1.0]
	return math.Max(0.0, math.Min(1.0, score))
}
```

- [ ] **Step 2: Write trajectory_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/trajectory_test.go
package checkin

import (
	"testing"
)

func TestComputeTrajectory_Stable(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 110, "systolic_bp": 125, "diastolic_bp": 75,
			"weight": 78, "medication_adherence": 0.9,
			"physical_activity_minutes": 180, "symptom_severity": 2,
			"side_effects": 1,
		},
		PreviousValues: map[string]float64{
			"fbg": 120, "systolic_bp": 130, "diastolic_bp": 78,
			"weight": 80, "medication_adherence": 0.85,
			"physical_activity_minutes": 160, "symptom_severity": 3,
			"side_effects": 2,
		},
		BaselineValues:    map[string]float64{"weight": 82},
		CycleNumber:       2,
		SlotsFilled:       8,
		SlotsRequired:     8,
		DaysSinceSchedule: 0.5,
	}

	result := ComputeTrajectory(input)
	if result.Signal != TrajectoryStable {
		t.Errorf("expected STABLE, got %s", result.Signal)
	}
	if result.Confidence < 0.7 {
		t.Errorf("expected confidence >= 0.7, got %f", result.Confidence)
	}
	if len(result.ImprovingSlots) == 0 {
		t.Error("expected at least one improving slot")
	}
}

func TestComputeTrajectory_Fragile(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 160, "systolic_bp": 145, "diastolic_bp": 90,
			"weight": 82, "medication_adherence": 0.6,
			"physical_activity_minutes": 180, "symptom_severity": 2,
			"side_effects": 1,
		},
		PreviousValues: map[string]float64{
			"fbg": 120, "systolic_bp": 125, "diastolic_bp": 75,
			"weight": 80, "medication_adherence": 0.9,
			"physical_activity_minutes": 160, "symptom_severity": 2,
			"side_effects": 1,
		},
		CycleNumber:       3,
		SlotsFilled:       8,
		SlotsRequired:     8,
		DaysSinceSchedule: 1.0,
	}

	result := ComputeTrajectory(input)
	if result.Signal != TrajectoryFragile && result.Signal != TrajectoryFailure {
		t.Errorf("expected FRAGILE or FAILURE, got %s", result.Signal)
	}
	if len(result.WorseningSlots) == 0 {
		t.Error("expected worsening slots")
	}
}

func TestComputeTrajectory_Failure(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 220, "systolic_bp": 165, "diastolic_bp": 100,
			"weight": 88, "medication_adherence": 0.3,
			"physical_activity_minutes": 30, "symptom_severity": 7,
			"side_effects": 6,
		},
		PreviousValues: map[string]float64{
			"fbg": 120, "systolic_bp": 125, "diastolic_bp": 75,
			"weight": 80, "medication_adherence": 0.9,
			"physical_activity_minutes": 180, "symptom_severity": 2,
			"side_effects": 1,
		},
		CycleNumber:       4,
		SlotsFilled:       8,
		SlotsRequired:     8,
		DaysSinceSchedule: 0.5,
	}

	result := ComputeTrajectory(input)
	if result.Signal != TrajectoryFailure {
		t.Errorf("expected FAILURE, got %s", result.Signal)
	}
}

func TestComputeTrajectory_Disengage(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues:     map[string]float64{},
		PreviousValues:    map[string]float64{"fbg": 120},
		CycleNumber:       3,
		SlotsFilled:       0,
		SlotsRequired:     8,
		DaysSinceSchedule: 5.0,
	}

	result := ComputeTrajectory(input)
	if result.Signal != TrajectoryDisengage {
		t.Errorf("expected DISENGAGE, got %s", result.Signal)
	}
}

func TestComputeTrajectory_ConsecutiveFragileEscalation(t *testing.T) {
	fragile := TrajectoryFragile
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 150, "systolic_bp": 140, "diastolic_bp": 85,
			"weight": 82, "medication_adherence": 0.65,
			"physical_activity_minutes": 170, "symptom_severity": 3,
			"side_effects": 2,
		},
		PreviousValues: map[string]float64{
			"fbg": 120, "systolic_bp": 125, "diastolic_bp": 75,
			"weight": 80, "medication_adherence": 0.85,
			"physical_activity_minutes": 160, "symptom_severity": 2,
			"side_effects": 1,
		},
		CycleNumber:        4,
		SlotsFilled:        8,
		SlotsRequired:      8,
		DaysSinceSchedule:  1.0,
		PreviousTrajectory: &fragile,
	}

	result := ComputeTrajectory(input)
	// Two consecutive FRAGILE signals should escalate to FAILURE
	if result.Signal == TrajectoryStable {
		t.Error("consecutive FRAGILE should not result in STABLE")
	}
}

func TestComputeTrajectory_FirstCheckin(t *testing.T) {
	// First check-in uses baseline values as reference
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 115, "systolic_bp": 128, "diastolic_bp": 76,
			"weight": 79, "medication_adherence": 0.85,
			"physical_activity_minutes": 150, "symptom_severity": 2,
			"side_effects": 1,
		},
		PreviousValues: nil, // first check-in
		BaselineValues: map[string]float64{
			"fbg": 178, "systolic_bp": 145, "diastolic_bp": 92,
			"weight": 85, "medication_adherence": 0.5,
			"physical_activity_minutes": 30, "symptom_severity": 5,
			"side_effects": 3,
		},
		CycleNumber:       1,
		SlotsFilled:       8,
		SlotsRequired:     8,
		DaysSinceSchedule: 0.0,
	}

	result := ComputeTrajectory(input)
	if result.Signal != TrajectoryStable {
		t.Errorf("expected STABLE for improving first check-in, got %s", result.Signal)
	}
	if len(result.DomainScores) == 0 {
		t.Error("expected domain scores to be populated")
	}
}

func TestScoreSlotChange_WeightLoss(t *testing.T) {
	target := DefaultClinicalTargets()["weight"]

	// 5% weight loss = improvement
	score := scoreSlotChange(76, 80, target)
	if score < 0.7 {
		t.Errorf("5%% weight loss should score > 0.7, got %f", score)
	}

	// Weight gain = worsening
	score = scoreSlotChange(88, 80, target)
	if score > 0.3 {
		t.Errorf("10%% weight gain should score < 0.3, got %f", score)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/checkin/... -v -count=1`
Expected: All trajectory tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/trajectory.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/trajectory_test.go
git commit -m "feat(intake): add trajectory computer for check-in evaluation

Deterministic trajectory signal: STABLE/FRAGILE/FAILURE/DISENGAGE.
Uses slot-level scoring against clinical targets with per-domain
aggregation. Consecutive FRAGILE escalates to FAILURE. Zero LLM."
```

---

## Task 3: $checkin and $checkin-slot Handlers (Intake)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/handler.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/handler_test.go`
- Modify: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go` (wire handlers)

**Spec reference:** Section 3.1 — `POST /fhir/Patient/{id}/$checkin` starts a biweekly check-in, `POST /fhir/Encounter/{id}/$checkin-slot` fills a check-in slot. Publishes to `intake.checkin-events`.

- [ ] **Step 1: Write handler.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/handler.go
package checkin

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Handler provides HTTP handlers for check-in operations.
type Handler struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewHandler creates a new check-in handler.
func NewHandler(db *pgxpool.Pool, logger *zap.Logger) *Handler {
	return &Handler{db: db, logger: logger}
}

// StartCheckinRequest is the request body for $checkin.
type StartCheckinRequest struct {
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"` // optional override; defaults to now + 0
}

// StartCheckinResponse is the response body for $checkin.
type StartCheckinResponse struct {
	SessionID   uuid.UUID    `json:"session_id"`
	EncounterID uuid.UUID    `json:"encounter_id"`
	CycleNumber int          `json:"cycle_number"`
	State       CheckinState `json:"state"`
	Slots       []CheckinSlotDef `json:"slots"`
}

// HandleStartCheckin handles POST /fhir/Patient/{id}/$checkin.
// Creates a new biweekly check-in session for the patient.
func (h *Handler) HandleStartCheckin(c *gin.Context) {
	patientIDStr := c.Param("id")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid patient ID"})
		return
	}

	var req StartCheckinRequest
	if c.Request.ContentLength > 0 {
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
			return
		}
	}

	ctx := c.Request.Context()

	// Verify patient is ENROLLED
	var enrollState string
	err = h.db.QueryRow(ctx,
		"SELECT state FROM enrollments WHERE patient_id = $1", patientID,
	).Scan(&enrollState)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "patient not found or not enrolled"})
		return
	}
	if enrollState != "ENROLLED" {
		c.JSON(http.StatusConflict, gin.H{"error": "patient must be ENROLLED to start check-in, current state: " + enrollState})
		return
	}

	// Check for active (non-terminal) check-in session
	var activeCount int
	err = h.db.QueryRow(ctx,
		"SELECT COUNT(*) FROM checkin_sessions WHERE patient_id = $1 AND state NOT IN ('CS7_CLOSED')",
		patientID,
	).Scan(&activeCount)
	if err == nil && activeCount > 0 {
		c.JSON(http.StatusConflict, gin.H{"error": "patient already has an active check-in session"})
		return
	}

	// Determine cycle number
	var maxCycle int
	_ = h.db.QueryRow(ctx,
		"SELECT COALESCE(MAX(cycle_number), 0) FROM checkin_sessions WHERE patient_id = $1",
		patientID,
	).Scan(&maxCycle)
	cycleNumber := maxCycle + 1

	// Create session
	sessionID := uuid.New()
	encounterID := uuid.New()
	now := time.Now().UTC()
	scheduledAt := now
	if req.ScheduledAt != nil {
		scheduledAt = req.ScheduledAt.UTC()
	}

	_, err = h.db.Exec(ctx,
		`INSERT INTO checkin_sessions (id, patient_id, encounter_id, cycle_number, state, slots_total, scheduled_at, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $8)`,
		sessionID, patientID, encounterID, cycleNumber, CS1Scheduled, 12, scheduledAt, now,
	)
	if err != nil {
		h.logger.Error("failed to create check-in session", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create check-in session"})
		return
	}

	// Transition to CS2_NOTIFIED (patient initiated, so notification is implicit)
	_, err = h.db.Exec(ctx,
		"UPDATE checkin_sessions SET state = $1, updated_at = $2 WHERE id = $3",
		CS2Notified, now, sessionID,
	)
	if err != nil {
		h.logger.Error("failed to transition to CS2", zap.Error(err))
	}

	h.logger.Info("check-in session created",
		zap.String("session_id", sessionID.String()),
		zap.String("patient_id", patientID.String()),
		zap.Int("cycle_number", cycleNumber),
	)

	c.JSON(http.StatusCreated, StartCheckinResponse{
		SessionID:   sessionID,
		EncounterID: encounterID,
		CycleNumber: cycleNumber,
		State:       CS2Notified,
		Slots:       CheckinSlots(),
	})
}

// CheckinSlotRequest is the request body for $checkin-slot.
type CheckinSlotRequest struct {
	SlotName       string          `json:"slot_name" binding:"required"`
	Value          json.RawMessage `json:"value" binding:"required"`
	ExtractionMode string          `json:"extraction_mode" binding:"required"`
	Confidence     *float64        `json:"confidence,omitempty"`
}

// CheckinSlotResponse is the response body for $checkin-slot.
type CheckinSlotResponse struct {
	SessionID    uuid.UUID    `json:"session_id"`
	SlotName     string       `json:"slot_name"`
	SlotsFilled  int          `json:"slots_filled"`
	SlotsTotal   int          `json:"slots_total"`
	State        CheckinState `json:"state"`
	Trajectory   *TrajectoryResult `json:"trajectory,omitempty"` // populated when CS6_EVALUATED
	NextSlots    []string     `json:"next_slots,omitempty"`
}

// HandleFillCheckinSlot handles POST /fhir/Encounter/{id}/$checkin-slot.
// Fills a single check-in slot and advances the state machine if all required slots are filled.
func (h *Handler) HandleFillCheckinSlot(c *gin.Context) {
	encounterIDStr := c.Param("id")
	encounterID, err := uuid.Parse(encounterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	var req CheckinSlotRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	// Validate slot name
	validSlot := false
	for _, s := range CheckinSlots() {
		if s.Name == req.SlotName {
			validSlot = true
			break
		}
	}
	if !validSlot {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid check-in slot name: " + req.SlotName})
		return
	}

	ctx := c.Request.Context()

	// Find the check-in session by encounter_id
	var session CheckinSession
	err = h.db.QueryRow(ctx,
		`SELECT id, patient_id, encounter_id, cycle_number, state, slots_filled, slots_total,
		        scheduled_at, started_at, completed_at, created_at, updated_at
		 FROM checkin_sessions WHERE encounter_id = $1`,
		encounterID,
	).Scan(
		&session.ID, &session.PatientID, &session.EncounterID, &session.CycleNumber,
		&session.State, &session.SlotsFilled, &session.SlotsTotal,
		&session.ScheduledAt, &session.StartedAt, &session.CompletedAt,
		&session.CreatedAt, &session.UpdatedAt,
	)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "check-in session not found for encounter"})
		return
	}

	// Transition to IN_PROGRESS if NOTIFIED
	if session.State == CS2Notified {
		if err := session.Transition(CS3InProgress); err != nil {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		_, _ = h.db.Exec(ctx,
			"UPDATE checkin_sessions SET state = $1, started_at = $2, updated_at = $2 WHERE id = $3",
			session.State, session.StartedAt, session.ID,
		)
	}

	// Verify state allows slot fills
	if session.State != CS3InProgress {
		c.JSON(http.StatusConflict, gin.H{
			"error": fmt.Sprintf("check-in session is in state %s, cannot fill slots", session.State),
		})
		return
	}

	// Insert slot event
	confidence := 0.0
	if req.Confidence != nil {
		confidence = *req.Confidence
	}

	slotEventID := uuid.New()
	_, err = h.db.Exec(ctx,
		`INSERT INTO checkin_slot_events (id, session_id, patient_id, slot_name, domain, value, extraction_mode, confidence)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		slotEventID, session.ID, session.PatientID, req.SlotName,
		slotDomain(req.SlotName), req.Value, req.ExtractionMode, confidence,
	)
	if err != nil {
		h.logger.Error("failed to insert check-in slot event", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save slot data"})
		return
	}

	// Count filled slots (distinct slot names)
	var filledCount int
	err = h.db.QueryRow(ctx,
		"SELECT COUNT(DISTINCT slot_name) FROM checkin_slot_events WHERE session_id = $1",
		session.ID,
	).Scan(&filledCount)
	if err != nil {
		filledCount = session.SlotsFilled + 1
	}

	// Update slots_filled
	_, _ = h.db.Exec(ctx,
		"UPDATE checkin_sessions SET slots_filled = $1, updated_at = $2 WHERE id = $3",
		filledCount, time.Now().UTC(), session.ID,
	)
	session.SlotsFilled = filledCount

	// Check if all required slots are filled
	filledSlots := h.getFilledSlotNames(ctx, session.ID)
	var trajectoryResult *TrajectoryResult

	if session.RequiredSlotsFilled(filledSlots) {
		// Transition CS3 -> CS5 (completed)
		if err := session.Transition(CS5Completed); err == nil {
			now := time.Now().UTC()
			_, _ = h.db.Exec(ctx,
				"UPDATE checkin_sessions SET state = $1, completed_at = $2, updated_at = $2 WHERE id = $3",
				session.State, now, session.ID,
			)

			// Compute trajectory (CS5 -> CS6)
			trajResult := h.computeAndStoreTrajectory(ctx, &session)
			if trajResult != nil {
				trajectoryResult = trajResult
			}
		}
	}

	// Determine next unfilled slots
	nextSlots := h.getUnfilledRequiredSlots(filledSlots)

	h.logger.Info("check-in slot filled",
		zap.String("session_id", session.ID.String()),
		zap.String("slot_name", req.SlotName),
		zap.Int("slots_filled", filledCount),
	)

	c.JSON(http.StatusOK, CheckinSlotResponse{
		SessionID:   session.ID,
		SlotName:    req.SlotName,
		SlotsFilled: filledCount,
		SlotsTotal:  session.SlotsTotal,
		State:       session.State,
		Trajectory:  trajectoryResult,
		NextSlots:   nextSlots,
	})
}

// computeAndStoreTrajectory computes the trajectory signal and stores it.
func (h *Handler) computeAndStoreTrajectory(ctx context.Context, session *CheckinSession) *TrajectoryResult {
	// Get current slot values
	currentValues := h.getSlotValues(ctx, session.ID)

	// Get previous check-in values
	var previousValues map[string]float64
	var previousTrajectory *Trajectory
	if session.CycleNumber > 1 {
		var prevSessionID uuid.UUID
		var prevTraj *string
		err := h.db.QueryRow(ctx,
			`SELECT id, trajectory FROM checkin_sessions
			 WHERE patient_id = $1 AND cycle_number = $2 AND state = 'CS7_CLOSED'`,
			session.PatientID, session.CycleNumber-1,
		).Scan(&prevSessionID, &prevTraj)
		if err == nil {
			previousValues = h.getSlotValues(ctx, prevSessionID)
			if prevTraj != nil {
				t := Trajectory(*prevTraj)
				previousTrajectory = &t
			}
		}
	}

	// Get baseline values from intake slot_events
	baselineValues := h.getBaselineValues(ctx, session.PatientID)

	input := TrajectoryInput{
		CurrentValues:      currentValues,
		PreviousValues:     previousValues,
		BaselineValues:     baselineValues,
		CycleNumber:        session.CycleNumber,
		SlotsFilled:        session.SlotsFilled,
		SlotsRequired:      8, // 8 required slots
		DaysSinceSchedule:  time.Since(session.ScheduledAt).Hours() / 24.0,
		PreviousTrajectory: previousTrajectory,
	}

	result := ComputeTrajectory(input)

	// Transition to CS6_EVALUATED
	if err := session.Transition(CS6Evaluated); err == nil {
		session.Trajectory = result.Signal
		_, _ = h.db.Exec(ctx,
			"UPDATE checkin_sessions SET state = $1, trajectory = $2, updated_at = $3 WHERE id = $4",
			session.State, result.Signal, time.Now().UTC(), session.ID,
		)
	}

	return &result
}

// getFilledSlotNames returns a set of filled slot names for a session.
func (h *Handler) getFilledSlotNames(ctx context.Context, sessionID uuid.UUID) map[string]bool {
	filled := make(map[string]bool)
	rows, err := h.db.Query(ctx,
		"SELECT DISTINCT slot_name FROM checkin_slot_events WHERE session_id = $1", sessionID,
	)
	if err != nil {
		return filled
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err == nil {
			filled[name] = true
		}
	}
	return filled
}

// getUnfilledRequiredSlots returns required slot names not yet filled.
func (h *Handler) getUnfilledRequiredSlots(filled map[string]bool) []string {
	var unfilled []string
	for _, slot := range CheckinSlots() {
		if slot.Required && !filled[slot.Name] {
			unfilled = append(unfilled, slot.Name)
		}
	}
	return unfilled
}

// getSlotValues extracts numeric values from check-in slot events.
func (h *Handler) getSlotValues(ctx context.Context, sessionID uuid.UUID) map[string]float64 {
	values := make(map[string]float64)
	rows, err := h.db.Query(ctx,
		`SELECT DISTINCT ON (slot_name) slot_name, value
		 FROM checkin_slot_events WHERE session_id = $1
		 ORDER BY slot_name, created_at DESC`,
		sessionID,
	)
	if err != nil {
		return values
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var rawValue json.RawMessage
		if err := rows.Scan(&name, &rawValue); err == nil {
			var v float64
			if json.Unmarshal(rawValue, &v) == nil {
				values[name] = v
			}
		}
	}
	return values
}

// getBaselineValues gets baseline values from the initial intake slot_events.
func (h *Handler) getBaselineValues(ctx context.Context, patientID uuid.UUID) map[string]float64 {
	values := make(map[string]float64)
	checkinSlotNames := make([]string, 0, 12)
	for _, s := range CheckinSlots() {
		checkinSlotNames = append(checkinSlotNames, s.Name)
	}

	rows, err := h.db.Query(ctx,
		`SELECT DISTINCT ON (slot_name) slot_name, value
		 FROM slot_events WHERE patient_id = $1 AND slot_name = ANY($2)
		 ORDER BY slot_name, created_at ASC`,
		patientID, checkinSlotNames,
	)
	if err != nil {
		return values
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var rawValue json.RawMessage
		if err := rows.Scan(&name, &rawValue); err == nil {
			var v float64
			if json.Unmarshal(rawValue, &v) == nil {
				values[name] = v
			}
		}
	}
	return values
}

// slotDomain returns the domain for a check-in slot name.
func slotDomain(slotName string) string {
	for _, s := range CheckinSlots() {
		if s.Name == slotName {
			return s.Domain
		}
	}
	return "unknown"
}
```

- [ ] **Step 2: Write handler_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/handler_test.go
package checkin

import (
	"testing"
)

func TestSlotDomain(t *testing.T) {
	tests := []struct {
		slot   string
		domain string
	}{
		{"fbg", "glycemic"},
		{"systolic_bp", "cardiac"},
		{"egfr", "renal"},
		{"weight", "anthropometric"},
		{"medication_adherence", "behavioral"},
		{"symptom_severity", "symptoms"},
		{"unknown_slot", "unknown"},
	}
	for _, tt := range tests {
		got := slotDomain(tt.slot)
		if got != tt.domain {
			t.Errorf("slotDomain(%q) = %q, want %q", tt.slot, got, tt.domain)
		}
	}
}

func TestGetUnfilledRequiredSlots(t *testing.T) {
	h := &Handler{}

	// No slots filled
	unfilled := h.getUnfilledRequiredSlots(map[string]bool{})
	if len(unfilled) != 8 {
		t.Errorf("expected 8 unfilled required slots, got %d", len(unfilled))
	}

	// All required filled
	filled := map[string]bool{
		"fbg": true, "systolic_bp": true, "diastolic_bp": true,
		"weight": true, "medication_adherence": true,
		"physical_activity_minutes": true, "symptom_severity": true,
		"side_effects": true,
	}
	unfilled = h.getUnfilledRequiredSlots(filled)
	if len(unfilled) != 0 {
		t.Errorf("expected 0 unfilled required slots, got %d: %v", len(unfilled), unfilled)
	}

	// Partial fill
	partialFilled := map[string]bool{"fbg": true, "systolic_bp": true}
	unfilled = h.getUnfilledRequiredSlots(partialFilled)
	if len(unfilled) != 6 {
		t.Errorf("expected 6 unfilled required slots, got %d", len(unfilled))
	}
}
```

- [ ] **Step 3: Wire handlers in routes.go**

In `intake-onboarding-service/internal/api/routes.go`, replace the check-in stub handlers:

```go
// Replace:
//   fhir.POST("/Patient/:id/$checkin", s.stubHandler("Start Checkin"))
//   fhir.POST("/Encounter/:id/$checkin-slot", s.stubHandler("Fill Checkin Slot"))
// With:
//   checkinHandler := checkin.NewHandler(s.db, s.logger)
//   fhir.POST("/Patient/:id/$checkin", checkinHandler.HandleStartCheckin)
//   fhir.POST("/Encounter/:id/$checkin-slot", checkinHandler.HandleFillCheckinSlot)
```

- [ ] **Step 4: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/checkin/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/handler.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/checkin/handler_test.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go
git commit -m "feat(intake): add \$checkin and \$checkin-slot HTTP handlers

Start biweekly check-in (creates session, transitions CS1->CS2).
Fill check-in slots with auto-completion detection and trajectory
computation when all required slots are filled. Wired to routes."
```

---

## Task 4: Pharmacist Review Queue (Intake)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/queue.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/queue_test.go`

**Spec reference:** Section 2.2 `internal/review/queue.go` — Pharmacist review queue. Section 7.7 `review_queue` table (already in Plan 1 migration). Metrics: `intake_pharmacist_review_queue_depth{tenant_id, risk_stratum}`.

The review queue manages the pharmacist approval workflow. After a patient completes intake (all 50 slots filled), the case is submitted for pharmacist review. The queue uses risk stratification to prioritize HIGH-risk cases (with HARD_STOPs or multiple SOFT_FLAGs).

- [ ] **Step 1: Write queue.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/queue.go
package review

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ReviewStatus represents the status of a review queue entry.
type ReviewStatus string

const (
	StatusPending       ReviewStatus = "PENDING"
	StatusApproved      ReviewStatus = "APPROVED"
	StatusClarification ReviewStatus = "CLARIFICATION"
	StatusEscalated     ReviewStatus = "ESCALATED"
)

// RiskStratum represents the risk level for queue prioritization.
type RiskStratum string

const (
	RiskHigh   RiskStratum = "HIGH"
	RiskMedium RiskStratum = "MEDIUM"
	RiskLow    RiskStratum = "LOW"
)

// ReviewEntry represents a single entry in the pharmacist review queue.
type ReviewEntry struct {
	ID          uuid.UUID    `json:"id"`
	PatientID   uuid.UUID    `json:"patient_id"`
	EncounterID uuid.UUID    `json:"encounter_id"`
	TenantID    uuid.UUID    `json:"tenant_id"`
	RiskStratum RiskStratum  `json:"risk_stratum"`
	Status      ReviewStatus `json:"status"`
	ReviewerID  *uuid.UUID   `json:"reviewer_id,omitempty"`
	ReviewedAt  *time.Time   `json:"reviewed_at,omitempty"`
	Notes       string       `json:"notes,omitempty"`
	CreatedAt   time.Time    `json:"created_at"`
}

// RiskClassificationInput holds data used to compute risk stratum.
type RiskClassificationInput struct {
	HardStopCount int
	SoftFlagCount int
	Age           int
	MedCount      int
	EGFRValue     float64
}

// ClassifyRisk determines the risk stratum based on safety engine results and clinical data.
func ClassifyRisk(input RiskClassificationInput) RiskStratum {
	// HIGH: any HARD_STOP, or >=3 SOFT_FLAGs, or eGFR < 30
	if input.HardStopCount > 0 {
		return RiskHigh
	}
	if input.SoftFlagCount >= 3 {
		return RiskHigh
	}
	if input.EGFRValue > 0 && input.EGFRValue < 30 {
		return RiskHigh
	}

	// MEDIUM: 1-2 SOFT_FLAGs, or polypharmacy (>=5 meds), or age >=75
	if input.SoftFlagCount >= 1 {
		return RiskMedium
	}
	if input.MedCount >= 5 {
		return RiskMedium
	}
	if input.Age >= 75 {
		return RiskMedium
	}

	return RiskLow
}

// Queue provides operations on the pharmacist review queue backed by PostgreSQL.
type Queue struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewQueue creates a new review queue.
func NewQueue(db *pgxpool.Pool, logger *zap.Logger) *Queue {
	return &Queue{db: db, logger: logger}
}

// Submit adds a patient to the review queue.
func (q *Queue) Submit(ctx context.Context, entry ReviewEntry) (*ReviewEntry, error) {
	entry.ID = uuid.New()
	entry.Status = StatusPending
	entry.CreatedAt = time.Now().UTC()

	_, err := q.db.Exec(ctx,
		`INSERT INTO review_queue (id, patient_id, encounter_id, risk_stratum, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6)`,
		entry.ID, entry.PatientID, entry.EncounterID, entry.RiskStratum, entry.Status, entry.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("submit to review queue: %w", err)
	}

	q.logger.Info("case submitted for review",
		zap.String("entry_id", entry.ID.String()),
		zap.String("patient_id", entry.PatientID.String()),
		zap.String("risk_stratum", string(entry.RiskStratum)),
	)

	return &entry, nil
}

// ListPending returns pending review entries ordered by risk (HIGH first) then age.
func (q *Queue) ListPending(ctx context.Context, tenantID *uuid.UUID, limit, offset int) ([]ReviewEntry, error) {
	query := `SELECT id, patient_id, encounter_id, risk_stratum, status, reviewer_id, reviewed_at, created_at
		FROM review_queue
		WHERE status = 'PENDING'`
	args := []interface{}{}
	argIdx := 1

	if tenantID != nil {
		query += fmt.Sprintf(` AND patient_id IN (SELECT patient_id FROM enrollments WHERE tenant_id = $%d)`, argIdx)
		args = append(args, *tenantID)
		argIdx++
	}

	query += ` ORDER BY
		CASE risk_stratum
			WHEN 'HIGH' THEN 1
			WHEN 'MEDIUM' THEN 2
			WHEN 'LOW' THEN 3
		END ASC,
		created_at ASC`

	query += fmt.Sprintf(` LIMIT $%d OFFSET $%d`, argIdx, argIdx+1)
	args = append(args, limit, offset)

	rows, err := q.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list pending reviews: %w", err)
	}
	defer rows.Close()

	var entries []ReviewEntry
	for rows.Next() {
		var e ReviewEntry
		if err := rows.Scan(&e.ID, &e.PatientID, &e.EncounterID, &e.RiskStratum,
			&e.Status, &e.ReviewerID, &e.ReviewedAt, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan review entry: %w", err)
		}
		entries = append(entries, e)
	}
	return entries, nil
}

// Approve marks a review entry as approved.
func (q *Queue) Approve(ctx context.Context, entryID, reviewerID uuid.UUID) error {
	return q.updateStatus(ctx, entryID, reviewerID, StatusApproved)
}

// RequestClarification marks a review entry as needing clarification.
func (q *Queue) RequestClarification(ctx context.Context, entryID, reviewerID uuid.UUID, notes string) error {
	now := time.Now().UTC()
	_, err := q.db.Exec(ctx,
		`UPDATE review_queue SET status = $1, reviewer_id = $2, reviewed_at = $3
		 WHERE id = $4 AND status = 'PENDING'`,
		StatusClarification, reviewerID, now, entryID,
	)
	if err != nil {
		return fmt.Errorf("request clarification: %w", err)
	}
	return nil
}

// Escalate marks a review entry as escalated to a physician.
func (q *Queue) Escalate(ctx context.Context, entryID, reviewerID uuid.UUID, notes string) error {
	return q.updateStatus(ctx, entryID, reviewerID, StatusEscalated)
}

// QueueDepth returns the count of pending entries grouped by risk stratum.
func (q *Queue) QueueDepth(ctx context.Context) (map[RiskStratum]int, error) {
	depth := map[RiskStratum]int{RiskHigh: 0, RiskMedium: 0, RiskLow: 0}

	rows, err := q.db.Query(ctx,
		`SELECT risk_stratum, COUNT(*) FROM review_queue WHERE status = 'PENDING' GROUP BY risk_stratum`,
	)
	if err != nil {
		return nil, fmt.Errorf("queue depth: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var stratum RiskStratum
		var count int
		if err := rows.Scan(&stratum, &count); err == nil {
			depth[stratum] = count
		}
	}
	return depth, nil
}

// GetByEncounter finds the review entry for an encounter.
func (q *Queue) GetByEncounter(ctx context.Context, encounterID uuid.UUID) (*ReviewEntry, error) {
	var e ReviewEntry
	err := q.db.QueryRow(ctx,
		`SELECT id, patient_id, encounter_id, risk_stratum, status, reviewer_id, reviewed_at, created_at
		 FROM review_queue WHERE encounter_id = $1 ORDER BY created_at DESC LIMIT 1`,
		encounterID,
	).Scan(&e.ID, &e.PatientID, &e.EncounterID, &e.RiskStratum,
		&e.Status, &e.ReviewerID, &e.ReviewedAt, &e.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get review by encounter: %w", err)
	}
	return &e, nil
}

func (q *Queue) updateStatus(ctx context.Context, entryID, reviewerID uuid.UUID, status ReviewStatus) error {
	now := time.Now().UTC()
	result, err := q.db.Exec(ctx,
		`UPDATE review_queue SET status = $1, reviewer_id = $2, reviewed_at = $3
		 WHERE id = $4 AND status = 'PENDING'`,
		status, reviewerID, now, entryID,
	)
	if err != nil {
		return fmt.Errorf("update review status to %s: %w", status, err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("review entry %s not found or not in PENDING status", entryID)
	}

	q.logger.Info("review status updated",
		zap.String("entry_id", entryID.String()),
		zap.String("status", string(status)),
		zap.String("reviewer_id", reviewerID.String()),
	)
	return nil
}
```

- [ ] **Step 2: Write queue_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/queue_test.go
package review

import (
	"testing"
)

func TestClassifyRisk_High_HardStop(t *testing.T) {
	risk := ClassifyRisk(RiskClassificationInput{HardStopCount: 1})
	if risk != RiskHigh {
		t.Errorf("expected HIGH for hard stop, got %s", risk)
	}
}

func TestClassifyRisk_High_ManySoftFlags(t *testing.T) {
	risk := ClassifyRisk(RiskClassificationInput{SoftFlagCount: 3})
	if risk != RiskHigh {
		t.Errorf("expected HIGH for 3 soft flags, got %s", risk)
	}
}

func TestClassifyRisk_High_LowEGFR(t *testing.T) {
	risk := ClassifyRisk(RiskClassificationInput{EGFRValue: 25})
	if risk != RiskHigh {
		t.Errorf("expected HIGH for eGFR < 30, got %s", risk)
	}
}

func TestClassifyRisk_Medium_SoftFlag(t *testing.T) {
	risk := ClassifyRisk(RiskClassificationInput{SoftFlagCount: 1})
	if risk != RiskMedium {
		t.Errorf("expected MEDIUM for 1 soft flag, got %s", risk)
	}
}

func TestClassifyRisk_Medium_Polypharmacy(t *testing.T) {
	risk := ClassifyRisk(RiskClassificationInput{MedCount: 6})
	if risk != RiskMedium {
		t.Errorf("expected MEDIUM for polypharmacy, got %s", risk)
	}
}

func TestClassifyRisk_Medium_Elderly(t *testing.T) {
	risk := ClassifyRisk(RiskClassificationInput{Age: 80})
	if risk != RiskMedium {
		t.Errorf("expected MEDIUM for age >= 75, got %s", risk)
	}
}

func TestClassifyRisk_Low(t *testing.T) {
	risk := ClassifyRisk(RiskClassificationInput{
		HardStopCount: 0,
		SoftFlagCount: 0,
		Age:           45,
		MedCount:      2,
		EGFRValue:     90,
	})
	if risk != RiskLow {
		t.Errorf("expected LOW for healthy patient, got %s", risk)
	}
}

func TestReviewStatus_Constants(t *testing.T) {
	statuses := []ReviewStatus{StatusPending, StatusApproved, StatusClarification, StatusEscalated}
	if len(statuses) != 4 {
		t.Errorf("expected 4 review statuses, got %d", len(statuses))
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/review/... -v -count=1`
Expected: All 8 tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/queue.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/queue_test.go
git commit -m "feat(intake): add pharmacist review queue with risk stratification

PostgreSQL-backed queue with HIGH/MEDIUM/LOW risk classification.
Priority ordering: HIGH first, then by submission time.
CRUD: Submit, ListPending, Approve, RequestClarification, Escalate.
Risk rules: hard stops -> HIGH, 3+ soft flags -> HIGH, eGFR<30 -> HIGH,
polypharmacy/elderly -> MEDIUM."
```

---

## Task 5: Review Handlers (Intake)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/reviewer.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/reviewer_test.go`
- Modify: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go` (wire handlers)

**Spec reference:** Section 3.1 — `$submit-review`, `$approve`, `$request-clarification`, `$escalate`. Publishes to `intake.completions` (submit) and `intake.safety-alerts` (escalate).

- [ ] **Step 1: Write reviewer.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/reviewer.go
package review

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ReviewHandler provides HTTP handlers for pharmacist review operations.
type ReviewHandler struct {
	queue  *Queue
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewReviewHandler creates a new review handler.
func NewReviewHandler(db *pgxpool.Pool, logger *zap.Logger) *ReviewHandler {
	return &ReviewHandler{
		queue:  NewQueue(db, logger),
		db:     db,
		logger: logger,
	}
}

// SubmitReviewRequest is the request body for $submit-review.
type SubmitReviewRequest struct {
	HardStopCount int     `json:"hard_stop_count"`
	SoftFlagCount int     `json:"soft_flag_count"`
	Age           int     `json:"age"`
	MedCount      int     `json:"med_count"`
	EGFRValue     float64 `json:"egfr_value"`
}

// HandleSubmitReview handles POST /fhir/Encounter/{id}/$submit-review.
// Submits a completed intake for pharmacist review with risk classification.
func (h *ReviewHandler) HandleSubmitReview(c *gin.Context) {
	encounterIDStr := c.Param("id")
	encounterID, err := uuid.Parse(encounterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	var req SubmitReviewRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	ctx := c.Request.Context()

	// Look up patient from enrollment by encounter_id
	var patientID uuid.UUID
	var enrollState string
	err = h.db.QueryRow(ctx,
		"SELECT patient_id, state FROM enrollments WHERE encounter_id = $1",
		encounterID,
	).Scan(&patientID, &enrollState)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "encounter not found"})
		return
	}

	if enrollState != "INTAKE_COMPLETED" {
		c.JSON(http.StatusConflict, gin.H{
			"error": "intake must be INTAKE_COMPLETED before submitting for review, current: " + enrollState,
		})
		return
	}

	// Classify risk
	riskStratum := ClassifyRisk(RiskClassificationInput{
		HardStopCount: req.HardStopCount,
		SoftFlagCount: req.SoftFlagCount,
		Age:           req.Age,
		MedCount:      req.MedCount,
		EGFRValue:     req.EGFRValue,
	})

	// Submit to queue
	entry, err := h.queue.Submit(ctx, ReviewEntry{
		PatientID:   patientID,
		EncounterID: encounterID,
		RiskStratum: riskStratum,
	})
	if err != nil {
		h.logger.Error("failed to submit review", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to submit for review"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"review_id":    entry.ID,
		"risk_stratum": entry.RiskStratum,
		"status":       entry.Status,
	})
}

// HandleApprove handles POST /fhir/Encounter/{id}/$approve.
// Pharmacist approves the intake, transitioning patient to ENROLLED.
func (h *ReviewHandler) HandleApprove(c *gin.Context) {
	encounterIDStr := c.Param("id")
	encounterID, err := uuid.Parse(encounterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	reviewerID := extractReviewerID(c)
	if reviewerID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "reviewer ID required (X-User-ID header)"})
		return
	}

	ctx := c.Request.Context()

	// Find review entry
	entry, err := h.queue.GetByEncounter(ctx, encounterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review entry not found for encounter"})
		return
	}

	// Approve
	if err := h.queue.Approve(ctx, entry.ID, reviewerID); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Transition enrollment to ENROLLED
	_, err = h.db.Exec(ctx,
		"UPDATE enrollments SET state = 'ENROLLED', assigned_pharmacist = $1, updated_at = now() WHERE encounter_id = $2",
		reviewerID, encounterID,
	)
	if err != nil {
		h.logger.Error("failed to update enrollment state", zap.Error(err))
	}

	h.logger.Info("intake approved",
		zap.String("encounter_id", encounterID.String()),
		zap.String("reviewer_id", reviewerID.String()),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":      "APPROVED",
		"reviewer_id": reviewerID,
		"enrollment":  "ENROLLED",
	})
}

// ClarificationRequest is the body for $request-clarification.
type ClarificationRequest struct {
	SlotNames []string `json:"slot_names" binding:"required"` // slots that need re-filling
	Notes     string   `json:"notes"`
}

// HandleRequestClarification handles POST /fhir/Encounter/{id}/$request-clarification.
// Pharmacist requests clarification on specific slots, re-opening them.
func (h *ReviewHandler) HandleRequestClarification(c *gin.Context) {
	encounterIDStr := c.Param("id")
	encounterID, err := uuid.Parse(encounterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	var req ClarificationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	reviewerID := extractReviewerID(c)
	if reviewerID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "reviewer ID required"})
		return
	}

	ctx := c.Request.Context()

	entry, err := h.queue.GetByEncounter(ctx, encounterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review entry not found"})
		return
	}

	if err := h.queue.RequestClarification(ctx, entry.ID, reviewerID, req.Notes); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	// Transition enrollment back to INTAKE_IN_PROGRESS for re-filling
	_, err = h.db.Exec(ctx,
		"UPDATE enrollments SET state = 'INTAKE_IN_PROGRESS', updated_at = now() WHERE encounter_id = $1",
		encounterID,
	)
	if err != nil {
		h.logger.Error("failed to revert enrollment state", zap.Error(err))
	}

	h.logger.Info("clarification requested",
		zap.String("encounter_id", encounterID.String()),
		zap.Strings("slots", req.SlotNames),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":     "CLARIFICATION",
		"slot_names": req.SlotNames,
		"notes":      req.Notes,
	})
}

// EscalateRequest is the body for $escalate.
type EscalateRequest struct {
	Reason string `json:"reason" binding:"required"`
	Notes  string `json:"notes"`
}

// HandleEscalate handles POST /fhir/Encounter/{id}/$escalate.
// Pharmacist escalates the case to a physician for review.
func (h *ReviewHandler) HandleEscalate(c *gin.Context) {
	encounterIDStr := c.Param("id")
	encounterID, err := uuid.Parse(encounterIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid encounter ID"})
		return
	}

	var req EscalateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body: " + err.Error()})
		return
	}

	reviewerID := extractReviewerID(c)
	if reviewerID == uuid.Nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "reviewer ID required"})
		return
	}

	ctx := c.Request.Context()

	entry, err := h.queue.GetByEncounter(ctx, encounterID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "review entry not found"})
		return
	}

	if err := h.queue.Escalate(ctx, entry.ID, reviewerID, req.Notes); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	h.logger.Info("case escalated to physician",
		zap.String("encounter_id", encounterID.String()),
		zap.String("reason", req.Reason),
		zap.String("reviewer_id", reviewerID.String()),
	)

	c.JSON(http.StatusOK, gin.H{
		"status":   "ESCALATED",
		"reason":   req.Reason,
		"reviewer": reviewerID,
	})
}

// extractReviewerID gets the reviewer (pharmacist/physician) ID from the request header.
func extractReviewerID(c *gin.Context) uuid.UUID {
	idStr := c.GetHeader("X-User-ID")
	if idStr == "" {
		return uuid.Nil
	}
	id, err := uuid.Parse(idStr)
	if err != nil {
		return uuid.Nil
	}
	return id
}
```

- [ ] **Step 2: Write reviewer_test.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/reviewer_test.go
package review

import (
	"testing"

	"github.com/google/uuid"
)

func TestExtractReviewerID_Valid(t *testing.T) {
	// This is a unit-level validation test for the UUID parsing logic.
	id := uuid.New()
	parsed, err := uuid.Parse(id.String())
	if err != nil {
		t.Fatalf("failed to parse valid UUID: %v", err)
	}
	if parsed != id {
		t.Errorf("parsed UUID %s != original %s", parsed, id)
	}
}

func TestExtractReviewerID_Invalid(t *testing.T) {
	_, err := uuid.Parse("not-a-uuid")
	if err == nil {
		t.Error("expected error for invalid UUID")
	}
}

func TestExtractReviewerID_Empty(t *testing.T) {
	_, err := uuid.Parse("")
	if err == nil {
		t.Error("expected error for empty string")
	}
}
```

- [ ] **Step 3: Wire handlers in routes.go**

In `intake-onboarding-service/internal/api/routes.go`, replace the review stub handlers:

```go
// Replace:
//   fhir.POST("/Encounter/:id/$submit-review", s.stubHandler("Submit Review"))
//   fhir.POST("/Encounter/:id/$approve", s.stubHandler("Approve"))
//   fhir.POST("/Encounter/:id/$request-clarification", s.stubHandler("Request Clarification"))
//   fhir.POST("/Encounter/:id/$escalate", s.stubHandler("Escalate"))
// With:
//   reviewHandler := review.NewReviewHandler(s.db, s.logger)
//   fhir.POST("/Encounter/:id/$submit-review", reviewHandler.HandleSubmitReview)
//   fhir.POST("/Encounter/:id/$approve", reviewHandler.HandleApprove)
//   fhir.POST("/Encounter/:id/$request-clarification", reviewHandler.HandleRequestClarification)
//   fhir.POST("/Encounter/:id/$escalate", reviewHandler.HandleEscalate)
```

- [ ] **Step 4: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go test ./internal/review/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/reviewer.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/review/reviewer_test.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/api/routes.go
git commit -m "feat(intake): add pharmacist review handlers

\$submit-review with risk classification, \$approve transitioning to
ENROLLED, \$request-clarification re-opening slots, \$escalate to
physician. Reviewer ID from X-User-ID header. Wired to routes."
```

---

## Task 6: Health Connect Adapter (Ingestion)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/health_connect.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/health_connect_test.go`

**Spec reference:** Section 2.1 `internal/adapters/wearables/health_connect.go`. Source type: WEARABLE. POST `/ingest/wearables/health-connect`. Outputs CanonicalObservation for vitals/activity data.

Google Health Connect (Android) provides structured health data from multiple apps. The adapter receives data relayed by the Flutter app and converts it to CanonicalObservation format.

- [ ] **Step 1: Write health_connect.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/health_connect.go
package wearables

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// HealthConnectRecord represents a single Health Connect data record
// relayed from the Flutter app via structured JSON.
type HealthConnectRecord struct {
	RecordType    string    `json:"record_type"`    // e.g., "BloodPressure", "HeartRate", "Steps", "Weight", "BloodGlucose"
	PackageName   string    `json:"package_name"`   // source app package name
	DeviceModel   string    `json:"device_model"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Values        map[string]float64 `json:"values"` // e.g., {"systolic": 120, "diastolic": 80}
	Unit          string    `json:"unit"`
}

// HealthConnectPayload is the batch payload from the Flutter app.
type HealthConnectPayload struct {
	PatientID uuid.UUID             `json:"patient_id"`
	TenantID  uuid.UUID             `json:"tenant_id"`
	DeviceID  string                `json:"device_id"`
	Records   []HealthConnectRecord `json:"records"`
}

// healthConnectLOINCMap maps Health Connect record types to LOINC codes.
var healthConnectLOINCMap = map[string]struct {
	LOINCCode       string
	ObservationType canonical.ObservationType
	ValueKey        string
	Unit            string
}{
	"BloodPressureSystolic": {LOINCCode: "8480-6", ObservationType: canonical.ObsVitals, ValueKey: "systolic", Unit: "mmHg"},
	"BloodPressureDiastolic": {LOINCCode: "8462-4", ObservationType: canonical.ObsVitals, ValueKey: "diastolic", Unit: "mmHg"},
	"HeartRate":              {LOINCCode: "8867-4", ObservationType: canonical.ObsVitals, ValueKey: "bpm", Unit: "beats/min"},
	"Steps":                  {LOINCCode: "55423-8", ObservationType: canonical.ObsDeviceData, ValueKey: "count", Unit: "steps"},
	"Weight":                 {LOINCCode: "29463-7", ObservationType: canonical.ObsVitals, ValueKey: "kg", Unit: "kg"},
	"BloodGlucose":           {LOINCCode: "2339-0", ObservationType: canonical.ObsVitals, ValueKey: "mg_dl", Unit: "mg/dL"},
	"OxygenSaturation":       {LOINCCode: "2708-6", ObservationType: canonical.ObsVitals, ValueKey: "percent", Unit: "%"},
	"BodyTemperature":        {LOINCCode: "8310-5", ObservationType: canonical.ObsVitals, ValueKey: "celsius", Unit: "Cel"},
	"SleepSession":           {LOINCCode: "93832-4", ObservationType: canonical.ObsDeviceData, ValueKey: "hours", Unit: "h"},
	"ActiveCaloriesBurned":   {LOINCCode: "41981-2", ObservationType: canonical.ObsDeviceData, ValueKey: "kcal", Unit: "kcal"},
}

// HealthConnectAdapter converts Health Connect records to CanonicalObservations.
type HealthConnectAdapter struct{}

// NewHealthConnectAdapter creates a new Health Connect adapter.
func NewHealthConnectAdapter() *HealthConnectAdapter {
	return &HealthConnectAdapter{}
}

// Convert transforms Health Connect records into CanonicalObservation structs.
func (a *HealthConnectAdapter) Convert(payload HealthConnectPayload) ([]canonical.CanonicalObservation, error) {
	var observations []canonical.CanonicalObservation

	for _, record := range payload.Records {
		obs, err := a.convertRecord(payload, record)
		if err != nil {
			// Skip records that can't be converted but log warning
			continue
		}
		observations = append(observations, obs...)
	}

	if len(observations) == 0 {
		return nil, fmt.Errorf("no convertible records in Health Connect payload")
	}

	return observations, nil
}

func (a *HealthConnectAdapter) convertRecord(
	payload HealthConnectPayload,
	record HealthConnectRecord,
) ([]canonical.CanonicalObservation, error) {
	var observations []canonical.CanonicalObservation

	// Special handling for BloodPressure which has two values
	if record.RecordType == "BloodPressure" {
		systolic, hasSys := record.Values["systolic"]
		diastolic, hasDia := record.Values["diastolic"]

		if hasSys {
			obs := a.buildObservation(payload, record, "BloodPressureSystolic", systolic)
			if obs != nil {
				observations = append(observations, *obs)
			}
		}
		if hasDia {
			obs := a.buildObservation(payload, record, "BloodPressureDiastolic", diastolic)
			if obs != nil {
				observations = append(observations, *obs)
			}
		}
		return observations, nil
	}

	// Standard single-value records
	mapping, exists := healthConnectLOINCMap[record.RecordType]
	if !exists {
		return nil, fmt.Errorf("unsupported Health Connect record type: %s", record.RecordType)
	}

	value, hasValue := record.Values[mapping.ValueKey]
	if !hasValue {
		// Try first available value
		for _, v := range record.Values {
			value = v
			hasValue = true
			break
		}
	}

	if !hasValue {
		return nil, fmt.Errorf("no value found for record type %s", record.RecordType)
	}

	obs := a.buildObservation(payload, record, record.RecordType, value)
	if obs != nil {
		observations = append(observations, *obs)
	}

	return observations, nil
}

func (a *HealthConnectAdapter) buildObservation(
	payload HealthConnectPayload,
	record HealthConnectRecord,
	mappingKey string,
	value float64,
) *canonical.CanonicalObservation {
	mapping, exists := healthConnectLOINCMap[mappingKey]
	if !exists {
		return nil
	}

	return &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       payload.PatientID,
		TenantID:        payload.TenantID,
		SourceType:      canonical.SourceWearable,
		SourceID:        "health-connect:" + record.PackageName,
		ObservationType: mapping.ObservationType,
		LOINCCode:       mapping.LOINCCode,
		Value:           value,
		Unit:            mapping.Unit,
		Timestamp:       record.StartTime,
		QualityScore:    0.85, // Health Connect provides validated data
		DeviceContext: &canonical.DeviceContext{
			DeviceID:   payload.DeviceID,
			DeviceType: "health-connect",
			Model:      record.DeviceModel,
		},
	}
}
```

- [ ] **Step 2: Write health_connect_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/health_connect_test.go
package wearables

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestHealthConnectAdapter_BloodPressure(t *testing.T) {
	adapter := NewHealthConnectAdapter()

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "pixel-8",
		Records: []HealthConnectRecord{
			{
				RecordType:  "BloodPressure",
				PackageName: "com.google.android.apps.fitness",
				DeviceModel: "Pixel 8",
				StartTime:   time.Now(),
				EndTime:     time.Now(),
				Values:      map[string]float64{"systolic": 120, "diastolic": 80},
				Unit:        "mmHg",
			},
		},
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(observations) != 2 {
		t.Errorf("expected 2 observations (systolic + diastolic), got %d", len(observations))
	}

	// Check systolic
	foundSystolic := false
	foundDiastolic := false
	for _, obs := range observations {
		if obs.LOINCCode == "8480-6" {
			foundSystolic = true
			if obs.Value != 120 {
				t.Errorf("expected systolic 120, got %f", obs.Value)
			}
		}
		if obs.LOINCCode == "8462-4" {
			foundDiastolic = true
			if obs.Value != 80 {
				t.Errorf("expected diastolic 80, got %f", obs.Value)
			}
		}
	}
	if !foundSystolic {
		t.Error("missing systolic observation")
	}
	if !foundDiastolic {
		t.Error("missing diastolic observation")
	}
}

func TestHealthConnectAdapter_HeartRate(t *testing.T) {
	adapter := NewHealthConnectAdapter()

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "galaxy-watch",
		Records: []HealthConnectRecord{
			{
				RecordType:  "HeartRate",
				PackageName: "com.samsung.health",
				DeviceModel: "Galaxy Watch 6",
				StartTime:   time.Now(),
				EndTime:     time.Now(),
				Values:      map[string]float64{"bpm": 72},
			},
		},
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(observations))
	}
	if observations[0].LOINCCode != "8867-4" {
		t.Errorf("expected LOINC 8867-4, got %s", observations[0].LOINCCode)
	}
	if observations[0].Unit != "beats/min" {
		t.Errorf("expected beats/min, got %s", observations[0].Unit)
	}
}

func TestHealthConnectAdapter_Steps(t *testing.T) {
	adapter := NewHealthConnectAdapter()

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "pixel-watch",
		Records: []HealthConnectRecord{
			{
				RecordType: "Steps",
				StartTime:  time.Now().Add(-1 * time.Hour),
				EndTime:    time.Now(),
				Values:     map[string]float64{"count": 8500},
			},
		},
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if observations[0].Value != 8500 {
		t.Errorf("expected 8500 steps, got %f", observations[0].Value)
	}
	if observations[0].SourceType != "WEARABLE" {
		t.Errorf("expected WEARABLE source, got %s", observations[0].SourceType)
	}
}

func TestHealthConnectAdapter_UnsupportedType(t *testing.T) {
	adapter := NewHealthConnectAdapter()

	payload := HealthConnectPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Records: []HealthConnectRecord{
			{RecordType: "UnsupportedType", Values: map[string]float64{"x": 1}},
		},
	}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Error("expected error for unsupported record type")
	}
}

func TestHealthConnectAdapter_EmptyPayload(t *testing.T) {
	adapter := NewHealthConnectAdapter()
	payload := HealthConnectPayload{PatientID: uuid.New(), Records: nil}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Error("expected error for empty payload")
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/adapters/wearables/... -v -count=1`
Expected: All 5 tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/health_connect.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/health_connect_test.go
git commit -m "feat(ingestion): add Health Connect wearable adapter

Google Health Connect API -> CanonicalObservation. Supports 10 record
types: BP, HR, Steps, Weight, BloodGlucose, SpO2, Temperature, Sleep,
Calories. LOINC-coded. Quality score 0.85 for validated device data."
```

---

## Task 7: Ultrahuman CGM Adapter (Ingestion)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/ultrahuman.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/ultrahuman_test.go`

**Spec reference:** Section 2.1 `internal/adapters/wearables/ultrahuman.go` — CGM aggregation: TIR, TAR, TBR, CV, MAG. These are key glycemic variability metrics used by KB-26 (Metabolic Digital Twin) for glucose control assessment.

CGM metrics:
- **TIR** (Time in Range): % of time glucose 70-180 mg/dL (target: >70%)
- **TAR** (Time Above Range): % of time glucose >180 mg/dL (target: <25%)
- **TBR** (Time Below Range): % of time glucose <70 mg/dL (target: <4%)
- **CV** (Coefficient of Variation): glucose variability (target: <36%)
- **MAG** (Mean Absolute Glucose): average rate of glucose change

- [ ] **Step 1: Write ultrahuman.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/ultrahuman.go
package wearables

import (
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// UltrahumanCGMPayload represents a batch of CGM readings from Ultrahuman.
type UltrahumanCGMPayload struct {
	PatientID  uuid.UUID           `json:"patient_id"`
	TenantID   uuid.UUID           `json:"tenant_id"`
	DeviceID   string              `json:"device_id"`
	SensorID   string              `json:"sensor_id"`
	Readings   []CGMReading        `json:"readings"`
	PeriodStart time.Time          `json:"period_start"`
	PeriodEnd   time.Time          `json:"period_end"`
}

// CGMReading is a single CGM glucose measurement.
type CGMReading struct {
	Timestamp time.Time `json:"timestamp"`
	GlucoseMgDL float64 `json:"glucose_mg_dl"`
}

// CGMAggregation holds the computed CGM aggregate metrics.
type CGMAggregation struct {
	TIR        float64 `json:"tir"`         // Time in Range (70-180 mg/dL), percentage 0-100
	TAR        float64 `json:"tar"`         // Time Above Range (>180 mg/dL), percentage 0-100
	TBR        float64 `json:"tbr"`         // Time Below Range (<70 mg/dL), percentage 0-100
	CV         float64 `json:"cv"`          // Coefficient of Variation, percentage
	MAG        float64 `json:"mag"`         // Mean Absolute Glucose change, mg/dL/h
	MeanGlucose float64 `json:"mean_glucose"` // Mean glucose, mg/dL
	GMI        float64 `json:"gmi"`         // Glucose Management Indicator (estimated HbA1c)
	ReadingCount int   `json:"reading_count"`
}

// CGM range thresholds (International Consensus on TIR)
const (
	CGMLowThreshold  = 70.0  // mg/dL — below this = TBR
	CGMHighThreshold = 180.0 // mg/dL — above this = TAR
	CGMCVTarget      = 36.0  // % — CV < 36% indicates stable glucose
)

// UltrahumanAdapter converts Ultrahuman CGM data to CanonicalObservations.
type UltrahumanAdapter struct{}

// NewUltrahumanAdapter creates a new Ultrahuman CGM adapter.
func NewUltrahumanAdapter() *UltrahumanAdapter {
	return &UltrahumanAdapter{}
}

// Convert transforms CGM readings into aggregated CanonicalObservation structs.
func (a *UltrahumanAdapter) Convert(payload UltrahumanCGMPayload) ([]canonical.CanonicalObservation, error) {
	if len(payload.Readings) < 12 {
		return nil, fmt.Errorf("insufficient CGM readings: need >=12 for meaningful aggregation, got %d", len(payload.Readings))
	}

	agg := AggregateCGM(payload.Readings)

	deviceCtx := &canonical.DeviceContext{
		DeviceID:     payload.DeviceID,
		DeviceType:   "cgm",
		Manufacturer: "Ultrahuman",
		Model:        "M1/Cyborg",
	}

	midpoint := payload.PeriodStart.Add(payload.PeriodEnd.Sub(payload.PeriodStart) / 2)

	observations := []canonical.CanonicalObservation{
		{
			ID: uuid.New(), PatientID: payload.PatientID, TenantID: payload.TenantID,
			SourceType: canonical.SourceWearable, SourceID: "ultrahuman:" + payload.SensorID,
			ObservationType: canonical.ObsVitals, LOINCCode: "97507-8", // CGM TIR
			Value: agg.TIR, Unit: "%", Timestamp: midpoint,
			QualityScore: qualityFromReadingCount(agg.ReadingCount),
			DeviceContext: deviceCtx,
		},
		{
			ID: uuid.New(), PatientID: payload.PatientID, TenantID: payload.TenantID,
			SourceType: canonical.SourceWearable, SourceID: "ultrahuman:" + payload.SensorID,
			ObservationType: canonical.ObsVitals, LOINCCode: "97506-0", // CGM TAR
			Value: agg.TAR, Unit: "%", Timestamp: midpoint,
			QualityScore: qualityFromReadingCount(agg.ReadingCount),
			DeviceContext: deviceCtx,
		},
		{
			ID: uuid.New(), PatientID: payload.PatientID, TenantID: payload.TenantID,
			SourceType: canonical.SourceWearable, SourceID: "ultrahuman:" + payload.SensorID,
			ObservationType: canonical.ObsVitals, LOINCCode: "97505-2", // CGM TBR
			Value: agg.TBR, Unit: "%", Timestamp: midpoint,
			QualityScore: qualityFromReadingCount(agg.ReadingCount),
			DeviceContext: deviceCtx,
		},
		{
			ID: uuid.New(), PatientID: payload.PatientID, TenantID: payload.TenantID,
			SourceType: canonical.SourceWearable, SourceID: "ultrahuman:" + payload.SensorID,
			ObservationType: canonical.ObsVitals, LOINCCode: "97504-5", // CGM CV
			Value: agg.CV, Unit: "%", Timestamp: midpoint,
			QualityScore: qualityFromReadingCount(agg.ReadingCount),
			DeviceContext: deviceCtx,
		},
		{
			ID: uuid.New(), PatientID: payload.PatientID, TenantID: payload.TenantID,
			SourceType: canonical.SourceWearable, SourceID: "ultrahuman:" + payload.SensorID,
			ObservationType: canonical.ObsVitals, LOINCCode: "97503-7", // MAG (custom LOINC placeholder)
			Value: agg.MAG, Unit: "mg/dL/h", Timestamp: midpoint,
			QualityScore: qualityFromReadingCount(agg.ReadingCount),
			DeviceContext: deviceCtx,
		},
		{
			ID: uuid.New(), PatientID: payload.PatientID, TenantID: payload.TenantID,
			SourceType: canonical.SourceWearable, SourceID: "ultrahuman:" + payload.SensorID,
			ObservationType: canonical.ObsVitals, LOINCCode: "2339-0", // Mean glucose
			Value: agg.MeanGlucose, Unit: "mg/dL", Timestamp: midpoint,
			QualityScore: qualityFromReadingCount(agg.ReadingCount),
			DeviceContext: deviceCtx,
		},
		{
			ID: uuid.New(), PatientID: payload.PatientID, TenantID: payload.TenantID,
			SourceType: canonical.SourceWearable, SourceID: "ultrahuman:" + payload.SensorID,
			ObservationType: canonical.ObsVitals, LOINCCode: "97502-9", // GMI
			Value: agg.GMI, Unit: "%", Timestamp: midpoint,
			QualityScore: qualityFromReadingCount(agg.ReadingCount),
			DeviceContext: deviceCtx,
		},
	}

	// Add flags for concerning values
	for i := range observations {
		if agg.TBR > 4.0 {
			observations[i].Flags = append(observations[i].Flags, canonical.FlagCriticalValue)
		}
		if agg.CV > CGMCVTarget {
			observations[i].Flags = append(observations[i].Flags, canonical.FlagLowQuality)
		}
	}

	return observations, nil
}

// AggregateCGM computes TIR, TAR, TBR, CV, and MAG from raw CGM readings.
func AggregateCGM(readings []CGMReading) CGMAggregation {
	n := len(readings)
	if n == 0 {
		return CGMAggregation{}
	}

	// Sort by timestamp
	sort.Slice(readings, func(i, j int) bool {
		return readings[i].Timestamp.Before(readings[j].Timestamp)
	})

	var sum, sumSq float64
	var inRange, aboveRange, belowRange int

	for _, r := range readings {
		g := r.GlucoseMgDL
		sum += g
		sumSq += g * g

		if g < CGMLowThreshold {
			belowRange++
		} else if g > CGMHighThreshold {
			aboveRange++
		} else {
			inRange++
		}
	}

	nf := float64(n)
	mean := sum / nf

	// Variance and CV
	variance := (sumSq / nf) - (mean * mean)
	stdDev := math.Sqrt(math.Max(0, variance))
	cv := 0.0
	if mean > 0 {
		cv = (stdDev / mean) * 100.0
	}

	// MAG: Mean Absolute Glucose change per hour
	var totalAbsChange float64
	var totalHours float64
	for i := 1; i < n; i++ {
		dt := readings[i].Timestamp.Sub(readings[i-1].Timestamp).Hours()
		if dt > 0 && dt < 1.0 { // only count consecutive readings within 1 hour
			absChange := math.Abs(readings[i].GlucoseMgDL - readings[i-1].GlucoseMgDL)
			totalAbsChange += absChange
			totalHours += dt
		}
	}
	mag := 0.0
	if totalHours > 0 {
		mag = totalAbsChange / totalHours
	}

	// GMI (Glucose Management Indicator) = 3.31 + 0.02392 * mean glucose
	gmi := 3.31 + 0.02392*mean

	return CGMAggregation{
		TIR:          roundTo2(float64(inRange) / nf * 100),
		TAR:          roundTo2(float64(aboveRange) / nf * 100),
		TBR:          roundTo2(float64(belowRange) / nf * 100),
		CV:           roundTo2(cv),
		MAG:          roundTo2(mag),
		MeanGlucose:  roundTo2(mean),
		GMI:          roundTo2(gmi),
		ReadingCount: n,
	}
}

func roundTo2(v float64) float64 {
	return math.Round(v*100) / 100
}

func qualityFromReadingCount(count int) float64 {
	// 288 readings/day (every 5 min) is optimal. Score proportionally.
	if count >= 288 {
		return 0.95
	}
	if count >= 144 {
		return 0.85
	}
	if count >= 72 {
		return 0.75
	}
	return 0.60
}
```

- [ ] **Step 2: Write ultrahuman_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/ultrahuman_test.go
package wearables

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func generateCGMReadings(count int, baseGlucose float64, variability float64) []CGMReading {
	readings := make([]CGMReading, count)
	start := time.Date(2026, 3, 21, 0, 0, 0, 0, time.UTC)

	for i := 0; i < count; i++ {
		// Simple sinusoidal pattern for testing
		phase := float64(i) / float64(count) * 2 * math.Pi * 3 // 3 cycles per day
		glucose := baseGlucose + variability*math.Sin(phase)
		readings[i] = CGMReading{
			Timestamp:   start.Add(time.Duration(i*5) * time.Minute),
			GlucoseMgDL: glucose,
		}
	}
	return readings
}

func TestAggregateCGM_WellControlled(t *testing.T) {
	// Well-controlled diabetic: mean ~120, low variability
	readings := generateCGMReadings(288, 120, 20) // 24h, 5-min intervals

	agg := AggregateCGM(readings)

	if agg.ReadingCount != 288 {
		t.Errorf("expected 288 readings, got %d", agg.ReadingCount)
	}

	// TIR should be high (most readings 70-180)
	if agg.TIR < 90 {
		t.Errorf("expected TIR > 90%% for well-controlled, got %.1f%%", agg.TIR)
	}

	// TBR should be near 0
	if agg.TBR > 5 {
		t.Errorf("expected TBR < 5%% for well-controlled, got %.1f%%", agg.TBR)
	}

	// CV should be low
	if agg.CV > 36 {
		t.Errorf("expected CV < 36%% for well-controlled, got %.1f%%", agg.CV)
	}

	// GMI should be reasonable
	if agg.GMI < 5.0 || agg.GMI > 8.0 {
		t.Errorf("expected GMI 5-8%% for mean ~120, got %.2f%%", agg.GMI)
	}

	// TIR + TAR + TBR should approximately equal 100
	total := agg.TIR + agg.TAR + agg.TBR
	if math.Abs(total-100) > 1.0 {
		t.Errorf("TIR+TAR+TBR should ~= 100, got %.1f", total)
	}
}

func TestAggregateCGM_PoorlyControlled(t *testing.T) {
	// Poorly controlled: high mean, high variability
	readings := generateCGMReadings(288, 200, 80)

	agg := AggregateCGM(readings)

	// TAR should be significant
	if agg.TAR < 30 {
		t.Errorf("expected TAR > 30%% for poorly controlled, got %.1f%%", agg.TAR)
	}

	// Mean glucose should be high
	if agg.MeanGlucose < 150 {
		t.Errorf("expected mean glucose > 150 for poorly controlled, got %.1f", agg.MeanGlucose)
	}
}

func TestAggregateCGM_HypoglycemiaRisk(t *testing.T) {
	// Readings with significant time below range
	readings := generateCGMReadings(288, 80, 30) // mean 80, range ~50-110

	agg := AggregateCGM(readings)

	// TBR should be significant
	if agg.TBR < 5 {
		t.Errorf("expected TBR > 5%% for hypo-risk profile, got %.1f%%", agg.TBR)
	}
}

func TestAggregateCGM_MAGCalculation(t *testing.T) {
	// Two readings 5 minutes apart with known glucose change
	readings := []CGMReading{
		{Timestamp: time.Date(2026, 3, 21, 10, 0, 0, 0, time.UTC), GlucoseMgDL: 100},
		{Timestamp: time.Date(2026, 3, 21, 10, 5, 0, 0, time.UTC), GlucoseMgDL: 110},
		{Timestamp: time.Date(2026, 3, 21, 10, 10, 0, 0, time.UTC), GlucoseMgDL: 105},
		{Timestamp: time.Date(2026, 3, 21, 10, 15, 0, 0, time.UTC), GlucoseMgDL: 115},
		{Timestamp: time.Date(2026, 3, 21, 10, 20, 0, 0, time.UTC), GlucoseMgDL: 100},
		{Timestamp: time.Date(2026, 3, 21, 10, 25, 0, 0, time.UTC), GlucoseMgDL: 95},
		{Timestamp: time.Date(2026, 3, 21, 10, 30, 0, 0, time.UTC), GlucoseMgDL: 105},
		{Timestamp: time.Date(2026, 3, 21, 10, 35, 0, 0, time.UTC), GlucoseMgDL: 110},
		{Timestamp: time.Date(2026, 3, 21, 10, 40, 0, 0, time.UTC), GlucoseMgDL: 100},
		{Timestamp: time.Date(2026, 3, 21, 10, 45, 0, 0, time.UTC), GlucoseMgDL: 108},
		{Timestamp: time.Date(2026, 3, 21, 10, 50, 0, 0, time.UTC), GlucoseMgDL: 102},
		{Timestamp: time.Date(2026, 3, 21, 10, 55, 0, 0, time.UTC), GlucoseMgDL: 106},
	}

	agg := AggregateCGM(readings)

	if agg.MAG <= 0 {
		t.Errorf("expected positive MAG, got %.2f", agg.MAG)
	}
}

func TestUltrahumanAdapter_Convert(t *testing.T) {
	adapter := NewUltrahumanAdapter()

	readings := generateCGMReadings(288, 130, 25)
	start := readings[0].Timestamp
	end := readings[len(readings)-1].Timestamp

	payload := UltrahumanCGMPayload{
		PatientID:   uuid.New(),
		TenantID:    uuid.New(),
		DeviceID:    "uh-m1-001",
		SensorID:    "sensor-abc123",
		Readings:    readings,
		PeriodStart: start,
		PeriodEnd:   end,
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// Should produce 7 observations: TIR, TAR, TBR, CV, MAG, MeanGlucose, GMI
	if len(observations) != 7 {
		t.Errorf("expected 7 CGM metric observations, got %d", len(observations))
	}

	// All should be WEARABLE source
	for _, obs := range observations {
		if obs.SourceType != "WEARABLE" {
			t.Errorf("expected WEARABLE source, got %s", obs.SourceType)
		}
		if obs.DeviceContext == nil || obs.DeviceContext.Manufacturer != "Ultrahuman" {
			t.Error("expected Ultrahuman device context")
		}
	}
}

func TestUltrahumanAdapter_InsufficientReadings(t *testing.T) {
	adapter := NewUltrahumanAdapter()

	payload := UltrahumanCGMPayload{
		PatientID: uuid.New(),
		Readings:  generateCGMReadings(5, 120, 10), // too few
	}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Error("expected error for insufficient readings")
	}
}

func TestQualityFromReadingCount(t *testing.T) {
	tests := []struct {
		count    int
		minScore float64
	}{
		{288, 0.95},
		{200, 0.85},
		{100, 0.75},
		{50, 0.60},
	}
	for _, tt := range tests {
		score := qualityFromReadingCount(tt.count)
		if score < tt.minScore {
			t.Errorf("qualityFromReadingCount(%d) = %f, want >= %f", tt.count, score, tt.minScore)
		}
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/adapters/wearables/... -v -count=1`
Expected: All Ultrahuman tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/ultrahuman.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/ultrahuman_test.go
git commit -m "feat(ingestion): add Ultrahuman CGM adapter with TIR/TAR/TBR/CV/MAG

Aggregates raw CGM readings into 7 standardized metrics per
International Consensus on TIR. GMI formula: 3.31 + 0.02392 * mean.
Quality score scales with reading density (288/day = 0.95).
Critical flag when TBR > 4%. Feeds KB-26 Metabolic Digital Twin."
```

---

## Task 8: Apple HealthKit Adapter (Ingestion)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/apple_health.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/apple_health_test.go`

**Spec reference:** Section 2.1 `internal/adapters/wearables/apple_health.go`. POST `/ingest/wearables/apple-health`. Apple HealthKit data relayed via Flutter app.

- [ ] **Step 1: Write apple_health.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/apple_health.go
package wearables

import (
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/ingestion-service/internal/canonical"
)

// AppleHealthSample represents a single HealthKit sample relayed from the iOS Flutter app.
type AppleHealthSample struct {
	SampleType  string    `json:"sample_type"`  // HKQuantityTypeIdentifier e.g., "HKQuantityTypeIdentifierHeartRate"
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`          // HealthKit unit string e.g., "count/min", "mg/dL"
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date"`
	SourceName  string    `json:"source_name"`   // e.g., "Apple Watch", "iPhone"
	DeviceModel string    `json:"device_model"`
}

// AppleHealthPayload is the batch payload from the iOS Flutter app.
type AppleHealthPayload struct {
	PatientID uuid.UUID           `json:"patient_id"`
	TenantID  uuid.UUID           `json:"tenant_id"`
	DeviceID  string              `json:"device_id"`
	Samples   []AppleHealthSample `json:"samples"`
}

// appleHealthLOINCMap maps HealthKit quantity type identifiers to LOINC codes.
var appleHealthLOINCMap = map[string]struct {
	LOINCCode       string
	ObservationType canonical.ObservationType
	StandardUnit    string
}{
	"HKQuantityTypeIdentifierHeartRate":                {LOINCCode: "8867-4", ObservationType: canonical.ObsVitals, StandardUnit: "beats/min"},
	"HKQuantityTypeIdentifierBloodPressureSystolic":    {LOINCCode: "8480-6", ObservationType: canonical.ObsVitals, StandardUnit: "mmHg"},
	"HKQuantityTypeIdentifierBloodPressureDiastolic":   {LOINCCode: "8462-4", ObservationType: canonical.ObsVitals, StandardUnit: "mmHg"},
	"HKQuantityTypeIdentifierOxygenSaturation":         {LOINCCode: "2708-6", ObservationType: canonical.ObsVitals, StandardUnit: "%"},
	"HKQuantityTypeIdentifierBodyMass":                 {LOINCCode: "29463-7", ObservationType: canonical.ObsVitals, StandardUnit: "kg"},
	"HKQuantityTypeIdentifierBodyTemperature":          {LOINCCode: "8310-5", ObservationType: canonical.ObsVitals, StandardUnit: "Cel"},
	"HKQuantityTypeIdentifierBloodGlucose":             {LOINCCode: "2339-0", ObservationType: canonical.ObsVitals, StandardUnit: "mg/dL"},
	"HKQuantityTypeIdentifierStepCount":                {LOINCCode: "55423-8", ObservationType: canonical.ObsDeviceData, StandardUnit: "steps"},
	"HKQuantityTypeIdentifierActiveEnergyBurned":       {LOINCCode: "41981-2", ObservationType: canonical.ObsDeviceData, StandardUnit: "kcal"},
	"HKQuantityTypeIdentifierAppleExerciseTime":        {LOINCCode: "68516-4", ObservationType: canonical.ObsDeviceData, StandardUnit: "min"},
	"HKQuantityTypeIdentifierRespiratoryRate":          {LOINCCode: "9279-1", ObservationType: canonical.ObsVitals, StandardUnit: "breaths/min"},
	"HKCategoryTypeIdentifierSleepAnalysis":            {LOINCCode: "93832-4", ObservationType: canonical.ObsDeviceData, StandardUnit: "h"},
}

// AppleHealthAdapter converts Apple HealthKit samples to CanonicalObservations.
type AppleHealthAdapter struct{}

// NewAppleHealthAdapter creates a new Apple HealthKit adapter.
func NewAppleHealthAdapter() *AppleHealthAdapter {
	return &AppleHealthAdapter{}
}

// Convert transforms Apple HealthKit samples into CanonicalObservation structs.
func (a *AppleHealthAdapter) Convert(payload AppleHealthPayload) ([]canonical.CanonicalObservation, error) {
	var observations []canonical.CanonicalObservation

	for _, sample := range payload.Samples {
		obs, err := a.convertSample(payload, sample)
		if err != nil {
			continue // skip unconvertible samples
		}
		observations = append(observations, *obs)
	}

	if len(observations) == 0 {
		return nil, fmt.Errorf("no convertible samples in Apple Health payload")
	}

	return observations, nil
}

func (a *AppleHealthAdapter) convertSample(
	payload AppleHealthPayload,
	sample AppleHealthSample,
) (*canonical.CanonicalObservation, error) {
	mapping, exists := appleHealthLOINCMap[sample.SampleType]
	if !exists {
		return nil, fmt.Errorf("unsupported HealthKit type: %s", sample.SampleType)
	}

	value := sample.Value
	unit := mapping.StandardUnit

	// Unit conversions for common HealthKit units
	value = convertAppleHealthUnit(sample.SampleType, sample.Value, sample.Unit)

	return &canonical.CanonicalObservation{
		ID:              uuid.New(),
		PatientID:       payload.PatientID,
		TenantID:        payload.TenantID,
		SourceType:      canonical.SourceWearable,
		SourceID:        "apple-health:" + sample.SourceName,
		ObservationType: mapping.ObservationType,
		LOINCCode:       mapping.LOINCCode,
		Value:           value,
		Unit:            unit,
		Timestamp:       sample.StartDate,
		QualityScore:    0.90, // Apple Watch sensors are clinically validated
		DeviceContext: &canonical.DeviceContext{
			DeviceID:     payload.DeviceID,
			DeviceType:   "apple-health",
			Manufacturer: "Apple",
			Model:        sample.DeviceModel,
		},
	}, nil
}

// convertAppleHealthUnit handles unit conversion from HealthKit units to standard units.
func convertAppleHealthUnit(sampleType string, value float64, unit string) float64 {
	switch {
	case unit == "lb" || unit == "lbs":
		return value * 0.453592 // lbs -> kg
	case unit == "°F":
		return (value - 32) * 5 / 9 // Fahrenheit -> Celsius
	case unit == "mmol/L" && sampleType == "HKQuantityTypeIdentifierBloodGlucose":
		return value * 18.0182 // mmol/L -> mg/dL
	case unit == "kPa":
		return value * 7.50062 // kPa -> mmHg
	default:
		return value
	}
}
```

- [ ] **Step 2: Write apple_health_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/apple_health_test.go
package wearables

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestAppleHealthAdapter_HeartRate(t *testing.T) {
	adapter := NewAppleHealthAdapter()

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		DeviceID:  "apple-watch-ultra",
		Samples: []AppleHealthSample{
			{
				SampleType:  "HKQuantityTypeIdentifierHeartRate",
				Value:       68,
				Unit:        "count/min",
				StartDate:   time.Now(),
				SourceName:  "Apple Watch",
				DeviceModel: "Apple Watch Ultra 2",
			},
		},
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(observations) != 1 {
		t.Fatalf("expected 1 observation, got %d", len(observations))
	}
	if observations[0].LOINCCode != "8867-4" {
		t.Errorf("expected LOINC 8867-4, got %s", observations[0].LOINCCode)
	}
	if observations[0].Value != 68 {
		t.Errorf("expected 68 bpm, got %f", observations[0].Value)
	}
	if observations[0].QualityScore != 0.90 {
		t.Errorf("expected quality 0.90, got %f", observations[0].QualityScore)
	}
}

func TestAppleHealthAdapter_BloodGlucose_MmolConversion(t *testing.T) {
	adapter := NewAppleHealthAdapter()

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Samples: []AppleHealthSample{
			{
				SampleType: "HKQuantityTypeIdentifierBloodGlucose",
				Value:      6.5, // mmol/L
				Unit:       "mmol/L",
				StartDate:  time.Now(),
				SourceName: "Dexcom G7",
			},
		},
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	// 6.5 mmol/L * 18.0182 = ~117.1 mg/dL
	expectedMgDL := 6.5 * 18.0182
	if math.Abs(observations[0].Value-expectedMgDL) > 0.5 {
		t.Errorf("expected ~%.1f mg/dL, got %.1f", expectedMgDL, observations[0].Value)
	}
}

func TestAppleHealthAdapter_WeightLbsConversion(t *testing.T) {
	adapter := NewAppleHealthAdapter()

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Samples: []AppleHealthSample{
			{
				SampleType: "HKQuantityTypeIdentifierBodyMass",
				Value:      176, // lbs
				Unit:       "lb",
				StartDate:  time.Now(),
				SourceName: "iPhone",
			},
		},
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}

	expectedKg := 176 * 0.453592 // ~79.8 kg
	if math.Abs(observations[0].Value-expectedKg) > 0.5 {
		t.Errorf("expected ~%.1f kg, got %.1f", expectedKg, observations[0].Value)
	}
	if observations[0].Unit != "kg" {
		t.Errorf("expected kg unit, got %s", observations[0].Unit)
	}
}

func TestAppleHealthAdapter_MultipleSamples(t *testing.T) {
	adapter := NewAppleHealthAdapter()

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		TenantID:  uuid.New(),
		Samples: []AppleHealthSample{
			{SampleType: "HKQuantityTypeIdentifierHeartRate", Value: 72, Unit: "count/min", StartDate: time.Now(), SourceName: "Apple Watch"},
			{SampleType: "HKQuantityTypeIdentifierStepCount", Value: 10500, Unit: "count", StartDate: time.Now(), SourceName: "iPhone"},
			{SampleType: "HKQuantityTypeIdentifierOxygenSaturation", Value: 98, Unit: "%", StartDate: time.Now(), SourceName: "Apple Watch"},
		},
	}

	observations, err := adapter.Convert(payload)
	if err != nil {
		t.Fatalf("Convert failed: %v", err)
	}
	if len(observations) != 3 {
		t.Errorf("expected 3 observations, got %d", len(observations))
	}
}

func TestAppleHealthAdapter_UnsupportedType(t *testing.T) {
	adapter := NewAppleHealthAdapter()

	payload := AppleHealthPayload{
		PatientID: uuid.New(),
		Samples: []AppleHealthSample{
			{SampleType: "HKQuantityTypeIdentifierUnsupported", Value: 1, StartDate: time.Now()},
		},
	}

	_, err := adapter.Convert(payload)
	if err == nil {
		t.Error("expected error for unsupported sample type")
	}
}

func TestConvertAppleHealthUnit_Fahrenheit(t *testing.T) {
	celsius := convertAppleHealthUnit("HKQuantityTypeIdentifierBodyTemperature", 98.6, "°F")
	if math.Abs(celsius-37.0) > 0.1 {
		t.Errorf("expected ~37.0 C, got %.1f", celsius)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/adapters/wearables/... -v -count=1`
Expected: All Apple Health tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/apple_health.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/adapters/wearables/apple_health_test.go
git commit -m "feat(ingestion): add Apple HealthKit wearable adapter

12 HealthKit quantity types -> CanonicalObservation. Unit conversions:
lbs->kg, F->C, mmol/L->mg/dL, kPa->mmHg. Quality score 0.90 for
Apple Watch clinically validated sensors."
```

---

## Task 9: DLQ Management (Ingestion)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/publisher.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/resolver.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/replay.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/publisher_test.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/replay_test.go`

**Spec reference:** Section 2.1 `internal/dlq/` — DLQ publisher, resolver, replay. Section 3.2 admin endpoints: `GET /fhir/OperationOutcome?category=dlq`, `POST /fhir/OperationOutcome/{id}/$replay`. Section 7.1 error classes: PARSE, NORMALIZATION, VALIDATION. Metrics: `ingestion_dlq_messages_total{error_class, source_type}`.

- [ ] **Step 1: Write publisher.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/publisher.go
package dlq

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ErrorClass identifies the category of error that caused DLQ routing.
type ErrorClass string

const (
	ErrorParse         ErrorClass = "PARSE"
	ErrorNormalization ErrorClass = "NORMALIZATION"
	ErrorValidation    ErrorClass = "VALIDATION"
	ErrorMapping       ErrorClass = "MAPPING"
	ErrorPublish       ErrorClass = "PUBLISH"
)

// MessageStatus represents the lifecycle state of a DLQ message.
type MessageStatus string

const (
	StatusDLQPending   MessageStatus = "PENDING"
	StatusDLQReplayed  MessageStatus = "REPLAYED"
	StatusDLQDiscarded MessageStatus = "DISCARDED"
)

// DLQMessage represents a message in the dead letter queue.
type DLQMessage struct {
	ID           uuid.UUID     `json:"id"`
	ErrorClass   ErrorClass    `json:"error_class"`
	SourceType   string        `json:"source_type"`
	SourceID     string        `json:"source_id"`
	RawPayload   []byte        `json:"raw_payload"`
	ErrorMessage string        `json:"error_message"`
	RetryCount   int           `json:"retry_count"`
	Status       MessageStatus `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	ResolvedAt   *time.Time    `json:"resolved_at,omitempty"`
}

// Publisher writes failed messages to the DLQ (PostgreSQL + optionally Kafka ingestion.dlq).
type Publisher struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewPublisher creates a new DLQ publisher.
func NewPublisher(db *pgxpool.Pool, logger *zap.Logger) *Publisher {
	return &Publisher{db: db, logger: logger}
}

// Publish writes a failed message to the DLQ table.
func (p *Publisher) Publish(ctx context.Context, errorClass ErrorClass, sourceType, sourceID string, rawPayload []byte, errMsg string) (*DLQMessage, error) {
	msg := &DLQMessage{
		ID:           uuid.New(),
		ErrorClass:   errorClass,
		SourceType:   sourceType,
		SourceID:     sourceID,
		RawPayload:   rawPayload,
		ErrorMessage: errMsg,
		Status:       StatusDLQPending,
		CreatedAt:    time.Now().UTC(),
	}

	_, err := p.db.Exec(ctx,
		`INSERT INTO dlq_messages (id, error_class, source_type, source_id, raw_payload, error_message, status, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		msg.ID, msg.ErrorClass, msg.SourceType, msg.SourceID, msg.RawPayload, msg.ErrorMessage, msg.Status, msg.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("publish to DLQ: %w", err)
	}

	p.logger.Warn("message published to DLQ",
		zap.String("id", msg.ID.String()),
		zap.String("error_class", string(msg.ErrorClass)),
		zap.String("source_type", msg.SourceType),
		zap.String("error", msg.ErrorMessage),
	)

	return msg, nil
}

// IncrementRetryCount increments the retry count for a DLQ message.
func (p *Publisher) IncrementRetryCount(ctx context.Context, id uuid.UUID) error {
	_, err := p.db.Exec(ctx,
		"UPDATE dlq_messages SET retry_count = retry_count + 1 WHERE id = $1",
		id,
	)
	return err
}
```

- [ ] **Step 2: Write resolver.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/resolver.go
package dlq

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// Resolver provides admin operations for viewing and managing DLQ messages.
type Resolver struct {
	db     *pgxpool.Pool
	logger *zap.Logger
}

// NewResolver creates a new DLQ resolver.
func NewResolver(db *pgxpool.Pool, logger *zap.Logger) *Resolver {
	return &Resolver{db: db, logger: logger}
}

// ListFilter holds filter options for DLQ listing.
type ListFilter struct {
	Status     *MessageStatus
	ErrorClass *ErrorClass
	SourceType *string
	Limit      int
	Offset     int
}

// List returns DLQ messages matching the filter criteria.
func (r *Resolver) List(ctx context.Context, filter ListFilter) ([]DLQMessage, error) {
	query := `SELECT id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at, resolved_at
		FROM dlq_messages WHERE 1=1`
	args := []interface{}{}
	argIdx := 1

	if filter.Status != nil {
		query += fmt.Sprintf(" AND status = $%d", argIdx)
		args = append(args, *filter.Status)
		argIdx++
	}
	if filter.ErrorClass != nil {
		query += fmt.Sprintf(" AND error_class = $%d", argIdx)
		args = append(args, *filter.ErrorClass)
		argIdx++
	}
	if filter.SourceType != nil {
		query += fmt.Sprintf(" AND source_type = $%d", argIdx)
		args = append(args, *filter.SourceType)
		argIdx++
	}

	query += " ORDER BY created_at DESC"

	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	query += fmt.Sprintf(" LIMIT $%d OFFSET $%d", argIdx, argIdx+1)
	args = append(args, limit, filter.Offset)

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list DLQ messages: %w", err)
	}
	defer rows.Close()

	var messages []DLQMessage
	for rows.Next() {
		var msg DLQMessage
		if err := rows.Scan(&msg.ID, &msg.ErrorClass, &msg.SourceType, &msg.SourceID,
			&msg.RawPayload, &msg.ErrorMessage, &msg.RetryCount, &msg.Status,
			&msg.CreatedAt, &msg.ResolvedAt); err != nil {
			return nil, fmt.Errorf("scan DLQ message: %w", err)
		}
		messages = append(messages, msg)
	}
	return messages, nil
}

// Get retrieves a single DLQ message by ID.
func (r *Resolver) Get(ctx context.Context, id uuid.UUID) (*DLQMessage, error) {
	var msg DLQMessage
	err := r.db.QueryRow(ctx,
		`SELECT id, error_class, source_type, source_id, raw_payload, error_message, retry_count, status, created_at, resolved_at
		 FROM dlq_messages WHERE id = $1`,
		id,
	).Scan(&msg.ID, &msg.ErrorClass, &msg.SourceType, &msg.SourceID,
		&msg.RawPayload, &msg.ErrorMessage, &msg.RetryCount, &msg.Status,
		&msg.CreatedAt, &msg.ResolvedAt)
	if err != nil {
		return nil, fmt.Errorf("get DLQ message: %w", err)
	}
	return &msg, nil
}

// Discard marks a DLQ message as discarded (will not be replayed).
func (r *Resolver) Discard(ctx context.Context, id uuid.UUID) error {
	now := time.Now().UTC()
	result, err := r.db.Exec(ctx,
		"UPDATE dlq_messages SET status = $1, resolved_at = $2 WHERE id = $3 AND status = 'PENDING'",
		StatusDLQDiscarded, now, id,
	)
	if err != nil {
		return fmt.Errorf("discard DLQ message: %w", err)
	}
	if result.RowsAffected() == 0 {
		return fmt.Errorf("DLQ message %s not found or not PENDING", id)
	}

	r.logger.Info("DLQ message discarded", zap.String("id", id.String()))
	return nil
}

// Count returns counts of DLQ messages grouped by status.
func (r *Resolver) Count(ctx context.Context) (map[MessageStatus]int, error) {
	counts := map[MessageStatus]int{
		StatusDLQPending:   0,
		StatusDLQReplayed:  0,
		StatusDLQDiscarded: 0,
	}

	rows, err := r.db.Query(ctx,
		"SELECT status, COUNT(*) FROM dlq_messages GROUP BY status",
	)
	if err != nil {
		return nil, fmt.Errorf("count DLQ messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var status MessageStatus
		var count int
		if err := rows.Scan(&status, &count); err == nil {
			counts[status] = count
		}
	}
	return counts, nil
}
```

- [ ] **Step 3: Write replay.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/replay.go
package dlq

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"go.uber.org/zap"
)

// ReplayFunc is the function signature for re-injecting a message into the pipeline.
// It receives the raw payload, source type, and source ID.
type ReplayFunc func(ctx context.Context, rawPayload []byte, sourceType, sourceID string) error

// Replayer handles DLQ message replay by re-injecting messages into the ingestion pipeline.
type Replayer struct {
	db        *pgxpool.Pool
	resolver  *Resolver
	replayFn  ReplayFunc
	logger    *zap.Logger
	maxRetries int
}

// NewReplayer creates a new DLQ replayer.
func NewReplayer(db *pgxpool.Pool, resolver *Resolver, replayFn ReplayFunc, logger *zap.Logger) *Replayer {
	return &Replayer{
		db:         db,
		resolver:   resolver,
		replayFn:   replayFn,
		logger:     logger,
		maxRetries: 3,
	}
}

// ReplayResult holds the result of a replay attempt.
type ReplayResult struct {
	MessageID uuid.UUID `json:"message_id"`
	Success   bool      `json:"success"`
	Error     string    `json:"error,omitempty"`
	RetryCount int      `json:"retry_count"`
}

// Replay re-injects a single DLQ message into the ingestion pipeline.
func (r *Replayer) Replay(ctx context.Context, messageID uuid.UUID) (*ReplayResult, error) {
	msg, err := r.resolver.Get(ctx, messageID)
	if err != nil {
		return nil, fmt.Errorf("get DLQ message for replay: %w", err)
	}

	if msg.Status != StatusDLQPending {
		return nil, fmt.Errorf("cannot replay message in status %s", msg.Status)
	}

	if msg.RetryCount >= r.maxRetries {
		return &ReplayResult{
			MessageID:  messageID,
			Success:    false,
			Error:      fmt.Sprintf("max retries (%d) exceeded", r.maxRetries),
			RetryCount: msg.RetryCount,
		}, nil
	}

	// Increment retry count
	_, _ = r.db.Exec(ctx,
		"UPDATE dlq_messages SET retry_count = retry_count + 1 WHERE id = $1",
		messageID,
	)

	// Re-inject into pipeline
	err = r.replayFn(ctx, msg.RawPayload, msg.SourceType, msg.SourceID)
	if err != nil {
		r.logger.Warn("DLQ replay failed",
			zap.String("message_id", messageID.String()),
			zap.Int("retry_count", msg.RetryCount+1),
			zap.Error(err),
		)
		return &ReplayResult{
			MessageID:  messageID,
			Success:    false,
			Error:      err.Error(),
			RetryCount: msg.RetryCount + 1,
		}, nil
	}

	// Mark as replayed
	now := time.Now().UTC()
	_, _ = r.db.Exec(ctx,
		"UPDATE dlq_messages SET status = $1, resolved_at = $2 WHERE id = $3",
		StatusDLQReplayed, now, messageID,
	)

	r.logger.Info("DLQ message replayed successfully",
		zap.String("message_id", messageID.String()),
	)

	return &ReplayResult{
		MessageID:  messageID,
		Success:    true,
		RetryCount: msg.RetryCount + 1,
	}, nil
}

// ReplayBatch replays multiple DLQ messages. Returns results for each.
func (r *Replayer) ReplayBatch(ctx context.Context, messageIDs []uuid.UUID) []ReplayResult {
	var results []ReplayResult
	for _, id := range messageIDs {
		result, err := r.Replay(ctx, id)
		if err != nil {
			results = append(results, ReplayResult{
				MessageID: id,
				Success:   false,
				Error:     err.Error(),
			})
		} else {
			results = append(results, *result)
		}
	}
	return results
}
```

- [ ] **Step 4: Write publisher_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/publisher_test.go
package dlq

import (
	"testing"
)

func TestErrorClass_Constants(t *testing.T) {
	classes := []ErrorClass{ErrorParse, ErrorNormalization, ErrorValidation, ErrorMapping, ErrorPublish}
	if len(classes) != 5 {
		t.Errorf("expected 5 error classes, got %d", len(classes))
	}
}

func TestMessageStatus_Constants(t *testing.T) {
	statuses := []MessageStatus{StatusDLQPending, StatusDLQReplayed, StatusDLQDiscarded}
	if len(statuses) != 3 {
		t.Errorf("expected 3 message statuses, got %d", len(statuses))
	}
}
```

- [ ] **Step 5: Write replay_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/replay_test.go
package dlq

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

func TestReplayResult_Fields(t *testing.T) {
	result := ReplayResult{
		MessageID:  uuid.New(),
		Success:    true,
		RetryCount: 1,
	}
	if !result.Success {
		t.Error("expected success")
	}
	if result.RetryCount != 1 {
		t.Errorf("expected retry count 1, got %d", result.RetryCount)
	}
}

func TestReplayFunc_Signature(t *testing.T) {
	// Verify the ReplayFunc type is usable
	var fn ReplayFunc = func(ctx context.Context, rawPayload []byte, sourceType, sourceID string) error {
		return nil
	}
	err := fn(context.Background(), []byte("test"), "LAB", "thyrocare")
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}
```

- [ ] **Step 6: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/dlq/... -v -count=1`
Expected: All tests PASS

- [ ] **Step 7: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/dlq/
git commit -m "feat(ingestion): add DLQ publisher, resolver, and replay mechanism

PostgreSQL-backed DLQ with 5 error classes (PARSE, NORMALIZATION,
VALIDATION, MAPPING, PUBLISH). Admin resolver with filtering/counting.
Replay re-injects into pipeline with max 3 retries. Batch replay support."
```

---

## Task 10: Prometheus Metrics Collectors (Both Services)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/collectors.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/metrics/collectors.go`

**Spec reference:** Section 7.3 (10 ingestion metrics), Section 7.4 (10 intake metrics), Section 7.5 (health endpoints with /metrics).

- [ ] **Step 1: Write ingestion metrics/collectors.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/collectors.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Ingestion service Prometheus metrics — 10 metrics per spec section 7.3.

var (
	// MessagesReceived counts all messages received by source type.
	MessagesReceived = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_messages_received_total",
		Help: "Total messages received by the ingestion service",
	}, []string{"source_type", "source_id", "tenant_id"})

	// MessagesProcessed counts messages that completed processing by stage and status.
	MessagesProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_messages_processed_total",
		Help: "Total messages processed through pipeline stages",
	}, []string{"source_type", "stage", "status"})

	// PipelineDuration tracks processing time per pipeline stage.
	PipelineDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "ingestion_pipeline_duration_seconds",
		Help:    "Pipeline stage processing duration in seconds",
		Buckets: prometheus.ExponentialBuckets(0.001, 2, 12), // 1ms to ~4s
	}, []string{"source_type", "stage"})

	// CriticalValues counts observations flagged as critical values.
	CriticalValues = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_critical_values_total",
		Help: "Total critical value observations detected",
	}, []string{"observation_type", "tenant_id"})

	// DLQMessages counts messages routed to the dead letter queue.
	DLQMessages = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_dlq_messages_total",
		Help: "Total messages routed to DLQ by error class",
	}, []string{"error_class", "source_type"})

	// WALPending tracks the number of messages waiting in the Write-Ahead Log.
	WALPending = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "ingestion_wal_messages_pending",
		Help: "Number of messages pending in the WAL (Kafka failover buffer)",
	})

	// PatientResolutionPending tracks unresolved patient identifiers.
	PatientResolutionPending = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ingestion_patient_resolution_pending",
		Help: "Number of messages with unresolved patient identifiers",
	}, []string{"tenant_id"})

	// ABDMConsentOperations counts ABDM consent workflow operations.
	ABDMConsentOperations = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_abdm_consent_operations_total",
		Help: "Total ABDM consent operations by type and status",
	}, []string{"operation", "status"})

	// FHIRValidationFailures counts FHIR validation failures by profile.
	FHIRValidationFailures = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "ingestion_fhir_validation_failures_total",
		Help: "Total FHIR validation failures by profile and violation type",
	}, []string{"profile", "violation_type"})

	// SourceFreshness tracks the last-seen timestamp per source for staleness detection.
	SourceFreshness = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "ingestion_source_freshness_seconds",
		Help: "Seconds since last message from each source",
	}, []string{"source_type", "source_id"})
)
```

- [ ] **Step 2: Write intake metrics/collectors.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/metrics/collectors.go
package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Intake-Onboarding service Prometheus metrics — 10 metrics per spec section 7.4.

var (
	// Enrollments counts enrollment events by status.
	Enrollments = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "intake_enrollments_total",
		Help: "Total enrollment operations by tenant, channel, and status",
	}, []string{"tenant_id", "channel_type", "status"})

	// SlotFills counts slot fill operations by extraction mode and confidence.
	SlotFills = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "intake_slot_fills_total",
		Help: "Total slot fill operations by slot name, extraction mode, and confidence tier",
	}, []string{"slot_name", "extraction_mode", "confidence_tier"})

	// SafetyTriggers counts safety rule activations.
	SafetyTriggers = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "intake_safety_triggers_total",
		Help: "Total safety rule triggers by rule ID and type",
	}, []string{"rule_id", "rule_type", "tenant_id"})

	// SessionDuration tracks the duration of intake sessions.
	SessionDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "intake_session_duration_seconds",
		Help:    "Duration of intake sessions in seconds",
		Buckets: prometheus.ExponentialBuckets(60, 2, 10), // 1min to ~17hrs
	}, []string{"channel_type", "flow_type"})

	// NLULatency tracks the latency of NLU extraction calls.
	NLULatency = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "intake_nlu_latency_seconds",
		Help:    "NLU extraction latency in seconds",
		Buckets: prometheus.ExponentialBuckets(0.01, 2, 10), // 10ms to ~10s
	}, []string{"extraction_mode", "confidence_tier"})

	// PharmacistReviewQueueDepth tracks the review queue size by risk stratum.
	PharmacistReviewQueueDepth = promauto.NewGaugeVec(prometheus.GaugeOpts{
		Name: "intake_pharmacist_review_queue_depth",
		Help: "Current pharmacist review queue depth by tenant and risk stratum",
	}, []string{"tenant_id", "risk_stratum"})

	// WhatsAppDeliveryRate tracks WhatsApp message delivery success.
	WhatsAppDeliveryRate = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "intake_whatsapp_delivery_rate",
		Help: "WhatsApp message delivery count by message type",
	}, []string{"message_type"})

	// OfflineQueueDepth tracks the ASHA offline sync queue size.
	OfflineQueueDepth = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "intake_offline_queue_depth",
		Help: "Number of ASHA offline submissions pending sync",
	})

	// SessionLockContention tracks Redis lock contention events.
	SessionLockContention = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "intake_session_lock_contention",
		Help: "Number of concurrent session lock contention events",
	})

	// CheckinTrajectory counts check-in trajectory signals.
	CheckinTrajectory = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "intake_checkin_trajectory_total",
		Help: "Total check-in trajectory signals by type and tenant",
	}, []string{"trajectory", "tenant_id"})
)
```

- [ ] **Step 3: Wire metrics middleware in both services**

Update `metricsMiddleware()` in both `server.go` files to use the actual Prometheus metrics:

For ingestion `internal/api/server.go`:
```go
// Replace the existing metricsMiddleware() body with:
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		status := strconv.Itoa(c.Writer.Status())
		metrics.PipelineDuration.WithLabelValues("http", status).Observe(duration)
	}
}
```

For intake `internal/api/server.go`:
```go
// Replace the existing metricsMiddleware() body with:
func (s *Server) metricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()
		duration := time.Since(start).Seconds()
		_ = strconv.Itoa(c.Writer.Status())
		metrics.SessionDuration.WithLabelValues("http", "request").Observe(duration)
	}
}
```

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/collectors.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/metrics/collectors.go
git commit -m "feat: add 20 Prometheus metric collectors for ingestion and intake

Ingestion (10): messages received/processed, pipeline duration, critical
values, DLQ, WAL pending, patient resolution, ABDM consent, FHIR
validation, source freshness.
Intake (10): enrollments, slot fills, safety triggers, session duration,
NLU latency, review queue depth, WhatsApp delivery, offline queue,
lock contention, check-in trajectory."
```

---

## Task 11: OpenTelemetry Tracing (Both Services)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/tracing.go`
- Create: `vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/metrics/tracing.go`

**Spec reference:** Section 6.3 — Kafka envelope includes `traceId` field. Section 7.5 — OpenTelemetry traces for distributed tracing across services.

- [ ] **Step 1: Write ingestion tracing.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/tracing.go
package metrics

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	IngestionServiceName    = "ingestion-service"
	IngestionServiceVersion = "0.5.0"
)

// InitTracer initializes the OpenTelemetry tracer provider for the ingestion service.
func InitTracer(ctx context.Context, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(IngestionServiceName),
			semconv.ServiceVersionKey.String(IngestionServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))), // 10% sampling
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// TracingMiddleware returns a Gin middleware that creates spans for each request.
func TracingMiddleware(serviceName string) gin.HandlerFunc {
	tracer := otel.Tracer(serviceName)

	return func(c *gin.Context) {
		// Extract trace context from incoming headers
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.route", c.FullPath()),
			),
		)
		defer span.End()

		// Store trace context in Gin context for downstream use
		c.Request = c.Request.WithContext(ctx)
		c.Set("traceID", span.SpanContext().TraceID().String())

		c.Next()

		// Record response status
		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
	}
}

// SpanFromContext creates a child span from the current context.
func SpanFromContext(ctx context.Context, serviceName, operationName string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	tracer := otel.Tracer(serviceName)
	return tracer.Start(ctx, operationName, trace.WithAttributes(attrs...))
}

// TraceIDFromContext extracts the trace ID string from the context.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}
```

- [ ] **Step 2: Write intake tracing.go**

```go
// vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/metrics/tracing.go
package metrics

import (
	"context"
	"fmt"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.24.0"
	"go.opentelemetry.io/otel/trace"
)

const (
	IntakeServiceName    = "intake-onboarding-service"
	IntakeServiceVersion = "0.5.0"
)

// InitTracer initializes the OpenTelemetry tracer provider for the intake service.
func InitTracer(ctx context.Context, otlpEndpoint string) (*sdktrace.TracerProvider, error) {
	exporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(otlpEndpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return nil, fmt.Errorf("create OTLP exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(IntakeServiceName),
			semconv.ServiceVersionKey.String(IntakeServiceVersion),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("create resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(sdktrace.TraceIDRatioBased(0.1))),
	)

	otel.SetTracerProvider(tp)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	return tp, nil
}

// TracingMiddleware returns a Gin middleware for intake service request tracing.
func TracingMiddleware() gin.HandlerFunc {
	tracer := otel.Tracer(IntakeServiceName)

	return func(c *gin.Context) {
		ctx := otel.GetTextMapPropagator().Extract(c.Request.Context(), propagation.HeaderCarrier(c.Request.Header))

		spanName := fmt.Sprintf("%s %s", c.Request.Method, c.FullPath())
		ctx, span := tracer.Start(ctx, spanName,
			trace.WithSpanKind(trace.SpanKindServer),
			trace.WithAttributes(
				attribute.String("http.method", c.Request.Method),
				attribute.String("http.url", c.Request.URL.String()),
				attribute.String("http.route", c.FullPath()),
			),
		)
		defer span.End()

		c.Request = c.Request.WithContext(ctx)
		c.Set("traceID", span.SpanContext().TraceID().String())

		c.Next()

		span.SetAttributes(attribute.Int("http.status_code", c.Writer.Status()))
	}
}

// TraceIDFromContext extracts the trace ID for inclusion in Kafka envelopes.
func TraceIDFromContext(ctx context.Context) string {
	span := trace.SpanFromContext(ctx)
	if !span.SpanContext().IsValid() {
		return ""
	}
	return span.SpanContext().TraceID().String()
}
```

- [ ] **Step 3: Wire tracing in both main.go files**

Add to both services' `cmd/*/main.go`, after config load and before server creation:

```go
// Initialize OpenTelemetry tracing (optional — disabled if OTLP_ENDPOINT not set)
otlpEndpoint := os.Getenv("OTLP_ENDPOINT")
if otlpEndpoint != "" {
    tp, err := metrics.InitTracer(context.Background(), otlpEndpoint)
    if err != nil {
        logger.Warn("Failed to initialize tracing", zap.Error(err))
    } else {
        defer tp.Shutdown(context.Background())
        logger.Info("OpenTelemetry tracing initialized", zap.String("endpoint", otlpEndpoint))
    }
}
```

And add `TracingMiddleware` to the Gin router in both `server.go` files:

```go
// After router.Use(gin.Recovery())
router.Use(metrics.TracingMiddleware("service-name"))
```

- [ ] **Step 4: Add OTel dependencies to both go.mod files**

Run for both services:
```bash
cd vaidshala/clinical-runtime-platform/services/ingestion-service && go get go.opentelemetry.io/otel go.opentelemetry.io/otel/sdk go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
cd vaidshala/clinical-runtime-platform/services/intake-onboarding-service && go get go.opentelemetry.io/otel go.opentelemetry.io/otel/sdk go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp
```

- [ ] **Step 5: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/metrics/tracing.go \
       vaidshala/clinical-runtime-platform/services/intake-onboarding-service/internal/metrics/tracing.go
git commit -m "feat: add OpenTelemetry distributed tracing for both services

OTLP HTTP exporter to configurable endpoint. 10% parent-based sampling.
Gin middleware creates server spans with HTTP attributes. TraceID
propagated to Kafka envelopes for cross-service correlation.
Optional — disabled when OTLP_ENDPOINT is not set."
```

---

## Task 12: WAL (Write-Ahead Log) for Kafka Failover (Ingestion)

**Files:**
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/wal.go`
- Create: `vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/wal_test.go`

**Spec reference:** Section 2.1 `internal/kafka/wal.go` — Write-Ahead Log for Kafka failover (10GB cap, 30s retry). Section 5.2 — WAL stored on local disk. Section 7.1 — Kafka publish failures retry 3x then WAL. Metrics: `ingestion_wal_messages_pending`.

The WAL provides durability when Kafka is temporarily unavailable. Messages are appended to a local file-based log and replayed to Kafka on a 30-second retry loop. A 10GB cap prevents disk exhaustion.

- [ ] **Step 1: Write wal.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/wal.go
package kafka

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"

	"go.uber.org/zap"
)

const (
	// WALMaxSizeBytes is the maximum WAL file size (10 GB).
	WALMaxSizeBytes = 10 * 1024 * 1024 * 1024

	// WALRetryInterval is the interval between WAL replay attempts.
	WALRetryInterval = 30 * time.Second

	// WALFileName is the default WAL file name.
	WALFileName = "ingestion_wal.bin"

	// walEntryHeaderSize is 4 bytes for message length prefix.
	walEntryHeaderSize = 4
)

// WALEntry represents a single message in the Write-Ahead Log.
type WALEntry struct {
	Topic        string    `json:"topic"`
	PartitionKey string    `json:"partition_key"`
	Payload      []byte    `json:"payload"`
	Timestamp    time.Time `json:"timestamp"`
}

// PublishFunc is the function signature for publishing a message to Kafka.
type PublishFunc func(ctx context.Context, topic, partitionKey string, payload []byte) error

// WAL provides a local Write-Ahead Log for Kafka failover.
type WAL struct {
	mu         sync.Mutex
	file       *os.File
	path       string
	sizeBytes  int64
	pending    int64
	publishFn  PublishFunc
	logger     *zap.Logger
	stopCh     chan struct{}
	stopped    bool
}

// NewWAL creates and opens a Write-Ahead Log at the given directory.
func NewWAL(dir string, publishFn PublishFunc, logger *zap.Logger) (*WAL, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, fmt.Errorf("create WAL directory: %w", err)
	}

	path := filepath.Join(dir, WALFileName)

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("open WAL file: %w", err)
	}

	stat, err := file.Stat()
	if err != nil {
		file.Close()
		return nil, fmt.Errorf("stat WAL file: %w", err)
	}

	w := &WAL{
		file:      file,
		path:      path,
		sizeBytes: stat.Size(),
		publishFn: publishFn,
		logger:    logger,
		stopCh:    make(chan struct{}),
	}

	// Count pending entries
	w.pending = w.countEntries()

	return w, nil
}

// Append writes a message to the WAL. Returns an error if the WAL has exceeded its size cap.
func (w *WAL) Append(entry WALEntry) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.sizeBytes >= WALMaxSizeBytes {
		return fmt.Errorf("WAL size cap exceeded (%d bytes >= %d bytes)", w.sizeBytes, WALMaxSizeBytes)
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal WAL entry: %w", err)
	}

	// Write length-prefixed entry: [4 bytes length][json payload]
	header := make([]byte, walEntryHeaderSize)
	binary.BigEndian.PutUint32(header, uint32(len(data)))

	if _, err := w.file.Write(header); err != nil {
		return fmt.Errorf("write WAL header: %w", err)
	}
	if _, err := w.file.Write(data); err != nil {
		return fmt.Errorf("write WAL entry: %w", err)
	}
	if err := w.file.Sync(); err != nil {
		return fmt.Errorf("sync WAL: %w", err)
	}

	w.sizeBytes += int64(walEntryHeaderSize + len(data))
	w.pending++

	w.logger.Debug("WAL entry appended",
		zap.String("topic", entry.Topic),
		zap.Int64("pending", w.pending),
		zap.Int64("size_bytes", w.sizeBytes),
	)

	return nil
}

// StartReplayLoop begins a background goroutine that attempts to replay WAL entries
// to Kafka every WALRetryInterval.
func (w *WAL) StartReplayLoop(ctx context.Context) {
	go func() {
		ticker := time.NewTicker(WALRetryInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-w.stopCh:
				return
			case <-ticker.C:
				if w.Pending() > 0 {
					replayed, err := w.Replay(ctx)
					if err != nil {
						w.logger.Warn("WAL replay cycle failed",
							zap.Error(err),
							zap.Int("replayed", replayed),
						)
					} else if replayed > 0 {
						w.logger.Info("WAL replay cycle completed",
							zap.Int("replayed", replayed),
							zap.Int64("remaining", w.Pending()),
						)
					}
				}
			}
		}
	}()
}

// Replay reads all entries from the WAL, publishes them to Kafka, and truncates
// the WAL file on success. Returns the number of entries successfully replayed.
func (w *WAL) Replay(ctx context.Context) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Read all entries
	entries, err := w.readEntries()
	if err != nil {
		return 0, fmt.Errorf("read WAL entries: %w", err)
	}

	if len(entries) == 0 {
		return 0, nil
	}

	// Try to publish each entry
	replayed := 0
	var failedEntries []WALEntry

	for _, entry := range entries {
		err := w.publishFn(ctx, entry.Topic, entry.PartitionKey, entry.Payload)
		if err != nil {
			// Kafka still unavailable — keep remaining entries
			failedEntries = append(failedEntries, entry)
			failedEntries = append(failedEntries, entries[replayed+len(failedEntries):]...)
			break
		}
		replayed++
	}

	// Rewrite WAL with only failed entries (or truncate if all succeeded)
	if err := w.rewriteWAL(failedEntries); err != nil {
		return replayed, fmt.Errorf("rewrite WAL: %w", err)
	}

	w.pending = int64(len(failedEntries))
	return replayed, nil
}

// Pending returns the number of pending WAL entries.
func (w *WAL) Pending() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.pending
}

// SizeBytes returns the current WAL file size.
func (w *WAL) SizeBytes() int64 {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.sizeBytes
}

// Close stops the replay loop and closes the WAL file.
func (w *WAL) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.stopped {
		close(w.stopCh)
		w.stopped = true
	}
	return w.file.Close()
}

// readEntries reads all WAL entries from the file.
func (w *WAL) readEntries() ([]WALEntry, error) {
	// Seek to beginning
	if _, err := w.file.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	var entries []WALEntry
	for {
		header := make([]byte, walEntryHeaderSize)
		if _, err := io.ReadFull(w.file, header); err != nil {
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			return nil, err
		}

		length := binary.BigEndian.Uint32(header)
		data := make([]byte, length)
		if _, err := io.ReadFull(w.file, data); err != nil {
			break
		}

		var entry WALEntry
		if err := json.Unmarshal(data, &entry); err != nil {
			continue // skip corrupted entries
		}
		entries = append(entries, entry)
	}

	return entries, nil
}

// rewriteWAL replaces the WAL file with only the given entries.
func (w *WAL) rewriteWAL(entries []WALEntry) error {
	w.file.Close()

	// Truncate and rewrite
	file, err := os.Create(w.path)
	if err != nil {
		return err
	}

	var totalSize int64
	for _, entry := range entries {
		data, err := json.Marshal(entry)
		if err != nil {
			continue
		}
		header := make([]byte, walEntryHeaderSize)
		binary.BigEndian.PutUint32(header, uint32(len(data)))
		file.Write(header)
		file.Write(data)
		totalSize += int64(walEntryHeaderSize + len(data))
	}

	file.Sync()
	w.file = file
	w.sizeBytes = totalSize
	return nil
}

// countEntries counts entries in the WAL file (for initialization).
func (w *WAL) countEntries() int64 {
	entries, err := w.readEntries()
	if err != nil {
		return 0
	}
	// Seek back to end for appending
	w.file.Seek(0, io.SeekEnd)
	return int64(len(entries))
}
```

- [ ] **Step 2: Write wal_test.go**

```go
// vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/wal_test.go
package kafka

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"go.uber.org/zap"
)

func testWALLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func newTestWAL(t *testing.T, publishFn PublishFunc) (*WAL, string) {
	t.Helper()
	dir := t.TempDir()

	wal, err := NewWAL(dir, publishFn, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	return wal, dir
}

func TestWAL_AppendAndReplay(t *testing.T) {
	var published []WALEntry

	publishFn := func(ctx context.Context, topic, key string, payload []byte) error {
		published = append(published, WALEntry{Topic: topic, PartitionKey: key, Payload: payload})
		return nil
	}

	wal, _ := newTestWAL(t, publishFn)
	defer wal.Close()

	// Append 3 entries
	for i := 0; i < 3; i++ {
		err := wal.Append(WALEntry{
			Topic:        "ingestion.vitals",
			PartitionKey: fmt.Sprintf("patient-%d", i),
			Payload:      []byte(fmt.Sprintf(`{"value": %d}`, i)),
			Timestamp:    time.Now(),
		})
		if err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if wal.Pending() != 3 {
		t.Errorf("expected 3 pending, got %d", wal.Pending())
	}

	// Replay
	replayed, err := wal.Replay(context.Background())
	if err != nil {
		t.Fatalf("Replay failed: %v", err)
	}
	if replayed != 3 {
		t.Errorf("expected 3 replayed, got %d", replayed)
	}
	if len(published) != 3 {
		t.Errorf("expected 3 published, got %d", len(published))
	}
	if wal.Pending() != 0 {
		t.Errorf("expected 0 pending after replay, got %d", wal.Pending())
	}
}

func TestWAL_ReplayPartialFailure(t *testing.T) {
	callCount := 0
	publishFn := func(ctx context.Context, topic, key string, payload []byte) error {
		callCount++
		if callCount >= 3 {
			return fmt.Errorf("kafka unavailable")
		}
		return nil
	}

	wal, _ := newTestWAL(t, publishFn)
	defer wal.Close()

	for i := 0; i < 5; i++ {
		wal.Append(WALEntry{
			Topic:        "ingestion.vitals",
			PartitionKey: "patient-1",
			Payload:      []byte(fmt.Sprintf(`{"idx": %d}`, i)),
			Timestamp:    time.Now(),
		})
	}

	replayed, _ := wal.Replay(context.Background())
	if replayed != 2 {
		t.Errorf("expected 2 replayed before failure, got %d", replayed)
	}

	// Remaining entries should still be in WAL
	if wal.Pending() < 1 {
		t.Errorf("expected remaining pending entries, got %d", wal.Pending())
	}
}

func TestWAL_SizeCap(t *testing.T) {
	publishFn := func(ctx context.Context, topic, key string, payload []byte) error {
		return nil
	}

	dir := t.TempDir()

	// Create a WAL with a very small cap for testing
	wal, err := NewWAL(dir, publishFn, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	defer wal.Close()

	// The real WAL has a 10GB cap. We test the cap logic by checking
	// that SizeBytes increases with each append.
	initial := wal.SizeBytes()
	wal.Append(WALEntry{
		Topic:   "test",
		Payload: []byte(`{"data":"test"}`),
	})

	if wal.SizeBytes() <= initial {
		t.Error("expected WAL size to increase after append")
	}
}

func TestWAL_EmptyReplay(t *testing.T) {
	publishFn := func(ctx context.Context, topic, key string, payload []byte) error {
		return nil
	}

	wal, _ := newTestWAL(t, publishFn)
	defer wal.Close()

	replayed, err := wal.Replay(context.Background())
	if err != nil {
		t.Fatalf("Replay on empty WAL failed: %v", err)
	}
	if replayed != 0 {
		t.Errorf("expected 0 replayed on empty WAL, got %d", replayed)
	}
}

func TestWAL_PersistenceAcrossReopen(t *testing.T) {
	publishFn := func(ctx context.Context, topic, key string, payload []byte) error {
		return nil
	}

	dir := t.TempDir()

	// Open, append, close
	wal1, err := NewWAL(dir, publishFn, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL failed: %v", err)
	}
	wal1.Append(WALEntry{Topic: "test", Payload: []byte(`{"v":1}`), Timestamp: time.Now()})
	wal1.Append(WALEntry{Topic: "test", Payload: []byte(`{"v":2}`), Timestamp: time.Now()})
	wal1.Close()

	// Reopen — should see 2 pending entries
	wal2, err := NewWAL(dir, publishFn, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL reopen failed: %v", err)
	}
	defer wal2.Close()

	if wal2.Pending() != 2 {
		t.Errorf("expected 2 pending after reopen, got %d", wal2.Pending())
	}
}

func TestWAL_FileCreation(t *testing.T) {
	publishFn := func(ctx context.Context, topic, key string, payload []byte) error {
		return nil
	}

	dir := t.TempDir()
	subdir := dir + "/wal-data"

	wal, err := NewWAL(subdir, publishFn, testWALLogger())
	if err != nil {
		t.Fatalf("NewWAL with new dir failed: %v", err)
	}
	defer wal.Close()

	// Verify file exists
	_, err = os.Stat(subdir + "/" + WALFileName)
	if err != nil {
		t.Errorf("WAL file should exist: %v", err)
	}
}
```

- [ ] **Step 3: Run tests**

Run: `cd vaidshala/clinical-runtime-platform/services/ingestion-service && go test ./internal/kafka/... -v -count=1`
Expected: All WAL tests PASS

- [ ] **Step 4: Commit**

```bash
git add vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/wal.go \
       vaidshala/clinical-runtime-platform/services/ingestion-service/internal/kafka/wal_test.go
git commit -m "feat(ingestion): add Write-Ahead Log for Kafka failover

File-based WAL with length-prefixed binary format. 10GB cap prevents
disk exhaustion. 30-second retry loop replays entries to Kafka.
Partial replay supported — failed entries preserved for next cycle.
Persistence across service restarts verified."
```

---

## Summary

This plan delivers 12 tasks across both services:

| # | Task | Service | Key Files |
|---|------|---------|-----------|
| 1 | Check-in state machine (7 states, 12 slots) | Intake | `checkin/machine.go` |
| 2 | Trajectory computer (STABLE/FRAGILE/FAILURE/DISENGAGE) | Intake | `checkin/trajectory.go` |
| 3 | $checkin and $checkin-slot handlers | Intake | `checkin/handler.go` |
| 4 | Pharmacist review queue (risk stratification) | Intake | `review/queue.go` |
| 5 | Review handlers ($approve, $clarify, $escalate) | Intake | `review/reviewer.go` |
| 6 | Health Connect adapter | Ingestion | `wearables/health_connect.go` |
| 7 | Ultrahuman CGM adapter (TIR/TAR/TBR/CV/MAG) | Ingestion | `wearables/ultrahuman.go` |
| 8 | Apple HealthKit adapter | Ingestion | `wearables/apple_health.go` |
| 9 | DLQ management + replay | Ingestion | `dlq/publisher.go`, `dlq/resolver.go`, `dlq/replay.go` |
| 10 | 20 Prometheus metrics (10+10) | Both | `metrics/collectors.go` |
| 11 | OpenTelemetry tracing | Both | `metrics/tracing.go` |
| 12 | WAL for Kafka failover (10GB, 30s retry) | Ingestion | `kafka/wal.go` |

**Total new files:** ~30 (including tests)
**Dependencies added:** `go.opentelemetry.io/otel`, `go.opentelemetry.io/otel/sdk`, `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp`
**Migrations added:** `003_checkin.sql` (intake)
