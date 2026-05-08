package models

import (
	"time"

	"github.com/google/uuid"
)

// MonitoringPlan is the v2/v3 substrate entity that ensures the outcome
// loop closes after a recommendation is implemented. Per v2 §3 line 136:
// "monitoring outlives the recommendation that triggered it. The cessation
// closes Monday; the monitoring plan ('watch for urinary retention 14 days,
// falls 30 days, cognition 30 days') runs for a month."
//
// MonitoringPlan can carry multiple Obligations of mixed type (an observation
// to land, a follow-up review to occur, a behavioural chart to populate).
// Threshold crossings produce new Events that re-enter the Recommendation
// trigger surface — closing the v3 §3 line 136 outcome loop.
//
// State transitions are governed by monitoring.Lifecycle (Plan 0.3 Task 4),
// which writes an EvidenceTrace edge per transition. Direct State mutation
// outside the Lifecycle engine is a contract violation.
//
// Canonical storage: migrations/025_monitoring_lifecycle.sql (table:
// monitoring_plans). Obligations are stored as JSONB on the parent row;
// see the migration's monitoring_obligations_unrolled VIEW for query-friendly
// scanning.
type MonitoringPlan struct {
	ID                  uuid.UUID              `json:"id"`
	RecommendationID    uuid.UUID              `json:"recommendation_id"`
	ResidentID          uuid.UUID              `json:"resident_id"`
	State               string                 `json:"state"` // see MonitoringPlanState*
	Obligations         []MonitoringObligation `json:"obligations"`
	StartedAt           time.Time              `json:"started_at"`
	ExpectedEndAt       time.Time              `json:"expected_end_at"`
	EscalateAfterMissed int                    `json:"escalate_after_missed"` // # missed before auto-escalate
	CreatedAt           time.Time              `json:"created_at"`
	UpdatedAt           time.Time              `json:"updated_at"`
}

// MonitoringObligation is one expected observation/review/chart entry the
// plan tracks. FulfilledAt is nullable; DueAt drives the escalator sweep
// (Plan 0.3 Task 6). ThresholdSpec is a small DSL (e.g. "value > 5.5 OR
// value < 3.0") evaluated by the threshold evaluator (Plan 0.3 Task 5).
type MonitoringObligation struct {
	Type               string     `json:"type"` // see MonitoringObligationType*
	ObservationCode    string     `json:"observation_code,omitempty"` // e.g. "blood_pressure"
	FrequencyHours     int        `json:"frequency_hours,omitempty"`  // 0 = one-shot
	DueAt              time.Time  `json:"due_at"`
	ThresholdSpec      string     `json:"threshold_spec,omitempty"` // CQL-evaluable string
	FulfilledAt        *time.Time `json:"fulfilled_at,omitempty"`
	FulfilledByObsID   *uuid.UUID `json:"fulfilled_by_obs_id,omitempty"`
	ThresholdCrossedAt *time.Time `json:"threshold_crossed_at,omitempty"`
}

// validMonitoringTransitions encodes the monitoring lifecycle DAG. A pair
// (from, to) is in the map iff the transition is permitted. Direct
// mutation outside monitoring.Lifecycle is a contract violation; this
// function exists so the Lifecycle engine and storage layer share one
// source of truth. Completed, Escalated, Abandoned are terminal (not
// present as keys).
var validMonitoringTransitions = map[string]map[string]bool{
	MonitoringPlanStatePending: {
		MonitoringPlanStateActive:    true,
		MonitoringPlanStateAbandoned: true,
	},
	MonitoringPlanStateActive: {
		MonitoringPlanStateCompleted: true,
		MonitoringPlanStateEscalated: true,
		MonitoringPlanStateAbandoned: true,
	},
	// completed/escalated/abandoned are terminal
}

// IsValidMonitoringTransition reports whether the lifecycle DAG permits
// from → to.
func IsValidMonitoringTransition(from, to string) bool {
	if !IsValidMonitoringPlanState(from) || !IsValidMonitoringPlanState(to) {
		return false
	}
	allowed, ok := validMonitoringTransitions[from]
	if !ok {
		return false
	}
	return allowed[to]
}
