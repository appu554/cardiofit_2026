// Package models provides domain models for KB-19 Protocol Orchestrator.
package models

import (
	"time"

	"github.com/google/uuid"
)

// RecommendationBundle is the complete output of the KB-19 arbitration engine.
// It contains all decisions, their evidence, and a human-readable narrative.
// This is what gets returned to the calling system and stored for audit.
type RecommendationBundle struct {
	// Unique identifier for this bundle
	ID uuid.UUID `json:"id"`

	// Patient and encounter context
	PatientID   uuid.UUID `json:"patient_id"`
	EncounterID uuid.UUID `json:"encounter_id"`

	// When this bundle was generated
	Timestamp time.Time `json:"timestamp"`

	// All arbitrated decisions in priority order
	Decisions []ArbitratedDecision `json:"decisions"`

	// Protocol evaluations that led to these decisions
	ProtocolEvaluations []ProtocolEvaluation `json:"protocol_evaluations"`

	// Human-readable narrative summary
	NarrativeSummary string `json:"narrative_summary"`

	// Structured executive summary
	ExecutiveSummary ExecutiveSummary `json:"executive_summary"`

	// Execution plan with bindings to KB-3, KB-12, KB-14
	ExecutionPlan ExecutionPlan `json:"execution_plan"`

	// Conflicts that were identified and resolved
	ConflictsResolved []ConflictResolution `json:"conflicts_resolved"`

	// Safety gates that were applied
	SafetyGatesApplied []SafetyGate `json:"safety_gates_applied"`

	// Overall risk assessment
	RiskAssessment RiskAssessment `json:"risk_assessment"`

	// Alerts that should be surfaced to the clinical team
	Alerts []Alert `json:"alerts"`

	// Versions of all KB services used
	ServiceVersions map[string]string `json:"service_versions"`

	// Processing metrics
	ProcessingMetrics ProcessingMetrics `json:"processing_metrics"`

	// Status of this bundle
	Status BundleStatus `json:"status"`

	// If there was an error, details here
	Error *BundleError `json:"error,omitempty"`
}

// ExecutiveSummary provides a structured summary for quick review.
type ExecutiveSummary struct {
	// Total number of protocols evaluated
	ProtocolsEvaluated int `json:"protocols_evaluated"`

	// Number of protocols that were applicable
	ProtocolsApplicable int `json:"protocols_applicable"`

	// Number of conflicts detected
	ConflictsDetected int `json:"conflicts_detected"`

	// Number of safety blocks triggered
	SafetyBlocks int `json:"safety_blocks"`

	// Breakdown of decisions by type
	DecisionsByType map[DecisionType]int `json:"decisions_by_type"`

	// Highest urgency action
	HighestUrgency ActionUrgency `json:"highest_urgency"`

	// Key recommendations (top 3-5)
	KeyRecommendations []string `json:"key_recommendations"`

	// Critical warnings
	CriticalWarnings []string `json:"critical_warnings"`
}

// ExecutionPlan contains bindings to execution services.
type ExecutionPlan struct {
	// KB-3 Temporal bindings (scheduling, deadlines)
	TemporalBindings []TemporalBinding `json:"temporal_bindings"`

	// KB-12 OrderSet activations
	OrderSetActivations []OrderSetActivation `json:"orderset_activations"`

	// KB-14 Governance task creations
	GovernanceTasks []GovernanceTask `json:"governance_tasks"`
}

// TemporalBinding represents a KB-3 scheduling binding.
type TemporalBinding struct {
	DecisionID  uuid.UUID  `json:"decision_id"`
	ActionID    string     `json:"action_id"`
	ScheduledAt time.Time  `json:"scheduled_at"`
	Deadline    *time.Time `json:"deadline,omitempty"`
	Recurring   bool       `json:"recurring"`
	RecurFreq   string     `json:"recur_freq,omitempty"` // e.g., "daily", "q6h"
	AlertBefore *int       `json:"alert_before,omitempty"` // minutes before deadline
}

// OrderSetActivation represents a KB-12 order set activation.
type OrderSetActivation struct {
	DecisionID    uuid.UUID              `json:"decision_id"`
	OrderSetID    string                 `json:"orderset_id"`
	OrderSetName  string                 `json:"orderset_name"`
	Parameters    map[string]interface{} `json:"parameters"`
	ActivatedAt   time.Time              `json:"activated_at"`
	IndividualOrders []string            `json:"individual_orders"`
}

// GovernanceTask represents a KB-14 task creation.
type GovernanceTask struct {
	DecisionID  uuid.UUID `json:"decision_id"`
	TaskType    string    `json:"task_type"`    // REVIEW, APPROVAL, ESCALATION
	AssignedTo  string    `json:"assigned_to"`  // Role or specific user
	Priority    string    `json:"priority"`     // LOW, MEDIUM, HIGH, CRITICAL
	DueAt       time.Time `json:"due_at"`
	Description string    `json:"description"`
}

// ConflictResolution captures details of a protocol conflict and its resolution.
type ConflictResolution struct {
	// IDs of the conflicting protocols
	ProtocolA string `json:"protocol_a"`
	ProtocolB string `json:"protocol_b"`

	// Type of conflict
	ConflictType ConflictType `json:"conflict_type"`

	// Which protocol won
	Winner string `json:"winner"`
	Loser  string `json:"loser"`

	// Rule that determined the winner
	ResolutionRule string `json:"resolution_rule"`

	// Explanation of the resolution
	Explanation string `json:"explanation"`

	// What happened to the loser (DELAY, AVOID, etc.)
	LoserOutcome DecisionType `json:"loser_outcome"`

	// Confidence in this resolution
	Confidence float64 `json:"confidence"`
}

// ConflictType categorizes types of protocol conflicts.
type ConflictType string

const (
	// ConflictHemodynamic - Conflicting hemodynamic goals (e.g., sepsis fluids vs HF diuresis)
	ConflictHemodynamic ConflictType = "HEMODYNAMIC"

	// ConflictAnticoagulation - Bleeding vs clotting risk
	ConflictAnticoagulation ConflictType = "ANTICOAGULATION"

	// ConflictNephrotoxic - Nephrotoxic drug vs renal protection
	ConflictNephrotoxic ConflictType = "NEPHROTOXIC"

	// ConflictPregnancy - Teratogenic risk
	ConflictPregnancy ConflictType = "PREGNANCY"

	// ConflictNeurological - CNS effects or ICH risk
	ConflictNeurological ConflictType = "NEUROLOGICAL"

	// ConflictMetabolic - Metabolic derangement
	ConflictMetabolic ConflictType = "METABOLIC"

	// ConflictTiming - Temporal incompatibility
	ConflictTiming ConflictType = "TIMING"
)

// SafetyGate represents a safety check that was applied.
type SafetyGate struct {
	// Name of the safety gate
	Name string `json:"name"`

	// Source system
	Source string `json:"source"` // ICU_INTELLIGENCE, KB4_SAFETY, etc.

	// Was the gate triggered?
	Triggered bool `json:"triggered"`

	// Result of the gate check
	Result string `json:"result"` // PASS, WARN, BLOCK

	// Details of what was checked
	Details string `json:"details"`

	// Decisions affected by this gate
	AffectedDecisions []uuid.UUID `json:"affected_decisions"`
}

// RiskAssessment provides an overall risk assessment.
type RiskAssessment struct {
	// Overall risk level
	OverallRisk string `json:"overall_risk"` // LOW, MODERATE, HIGH, CRITICAL

	// Component risk scores
	MortalityRisk    float64 `json:"mortality_risk"`    // 0-1
	ComplicationRisk float64 `json:"complication_risk"` // 0-1
	ReadmissionRisk  float64 `json:"readmission_risk"`  // 0-1

	// Risk factors identified
	RiskFactors []string `json:"risk_factors"`

	// Protective factors
	ProtectiveFactors []string `json:"protective_factors"`
}

// Alert represents an alert to surface to the clinical team.
type Alert struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`     // INFO, WARNING, CRITICAL
	Severity    string    `json:"severity"` // LOW, MEDIUM, HIGH, CRITICAL
	Message     string    `json:"message"`
	DecisionRef uuid.UUID `json:"decision_ref,omitempty"`
	RequiresAck bool      `json:"requires_ack"`
}

// ProcessingMetrics captures performance metrics for the arbitration.
type ProcessingMetrics struct {
	StartTime            time.Time `json:"start_time"`
	EndTime              time.Time `json:"end_time"`
	TotalDurationMs      int64     `json:"total_duration_ms"`
	CQLEvaluationMs      int64     `json:"cql_evaluation_ms"`
	ProtocolMatchingMs   int64     `json:"protocol_matching_ms"`
	ConflictResolutionMs int64     `json:"conflict_resolution_ms"`
	SafetyCheckMs        int64     `json:"safety_check_ms"`
	NarrativeGenerationMs int64    `json:"narrative_generation_ms"`
}

// BundleStatus represents the status of a recommendation bundle.
type BundleStatus string

const (
	StatusPending   BundleStatus = "PENDING"
	StatusCompleted BundleStatus = "COMPLETED"
	StatusPartial   BundleStatus = "PARTIAL"  // Some evaluations failed
	StatusFailed    BundleStatus = "FAILED"
)

// BundleError captures error information if the bundle generation failed.
type BundleError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details string `json:"details,omitempty"`
}

// NewRecommendationBundle creates a new RecommendationBundle with initialized values.
func NewRecommendationBundle(patientID, encounterID uuid.UUID) *RecommendationBundle {
	return &RecommendationBundle{
		ID:                  uuid.New(),
		PatientID:           patientID,
		EncounterID:         encounterID,
		Timestamp:           time.Now(),
		Decisions:           make([]ArbitratedDecision, 0),
		ProtocolEvaluations: make([]ProtocolEvaluation, 0),
		ConflictsResolved:   make([]ConflictResolution, 0),
		SafetyGatesApplied:  make([]SafetyGate, 0),
		Alerts:              make([]Alert, 0),
		ServiceVersions:     make(map[string]string),
		ExecutiveSummary: ExecutiveSummary{
			DecisionsByType:    make(map[DecisionType]int),
			KeyRecommendations: make([]string, 0),
			CriticalWarnings:   make([]string, 0),
		},
		ExecutionPlan: ExecutionPlan{
			TemporalBindings:    make([]TemporalBinding, 0),
			OrderSetActivations: make([]OrderSetActivation, 0),
			GovernanceTasks:     make([]GovernanceTask, 0),
		},
		Status: StatusPending,
		ProcessingMetrics: ProcessingMetrics{
			StartTime: time.Now(),
		},
	}
}

// AddDecision adds an arbitrated decision to the bundle.
func (rb *RecommendationBundle) AddDecision(decision ArbitratedDecision) {
	rb.Decisions = append(rb.Decisions, decision)
	rb.ExecutiveSummary.DecisionsByType[decision.DecisionType]++
}

// AddProtocolEvaluation adds a protocol evaluation.
func (rb *RecommendationBundle) AddProtocolEvaluation(eval ProtocolEvaluation) {
	rb.ProtocolEvaluations = append(rb.ProtocolEvaluations, eval)
	rb.ExecutiveSummary.ProtocolsEvaluated++
	if eval.IsApplicable {
		rb.ExecutiveSummary.ProtocolsApplicable++
	}
}

// AddConflictResolution adds a conflict resolution record.
func (rb *RecommendationBundle) AddConflictResolution(resolution ConflictResolution) {
	rb.ConflictsResolved = append(rb.ConflictsResolved, resolution)
	rb.ExecutiveSummary.ConflictsDetected++
}

// AddSafetyGate adds a safety gate record.
func (rb *RecommendationBundle) AddSafetyGate(gate SafetyGate) {
	rb.SafetyGatesApplied = append(rb.SafetyGatesApplied, gate)
	if gate.Result == "BLOCK" {
		rb.ExecutiveSummary.SafetyBlocks++
	}
}

// AddAlert adds an alert.
func (rb *RecommendationBundle) AddAlert(alertType, severity, message string, requiresAck bool) {
	alert := Alert{
		ID:          uuid.New(),
		Type:        alertType,
		Severity:    severity,
		Message:     message,
		RequiresAck: requiresAck,
	}
	rb.Alerts = append(rb.Alerts, alert)

	if severity == "CRITICAL" {
		rb.ExecutiveSummary.CriticalWarnings = append(rb.ExecutiveSummary.CriticalWarnings, message)
	}
}

// Finalize completes the bundle processing.
func (rb *RecommendationBundle) Finalize() {
	rb.ProcessingMetrics.EndTime = time.Now()
	rb.ProcessingMetrics.TotalDurationMs = rb.ProcessingMetrics.EndTime.Sub(rb.ProcessingMetrics.StartTime).Milliseconds()
	rb.Status = StatusCompleted

	// Determine highest urgency
	highestUrgency := UrgencyScheduled
	for _, decision := range rb.Decisions {
		if decision.Urgency == UrgencySTAT {
			highestUrgency = UrgencySTAT
			break
		} else if decision.Urgency == UrgencyUrgent && highestUrgency != UrgencySTAT {
			highestUrgency = UrgencyUrgent
		} else if decision.Urgency == UrgencyRoutine && highestUrgency == UrgencyScheduled {
			highestUrgency = UrgencyRoutine
		}
	}
	rb.ExecutiveSummary.HighestUrgency = highestUrgency
}

// GetDODecisions returns all DO decisions.
func (rb *RecommendationBundle) GetDODecisions() []ArbitratedDecision {
	var result []ArbitratedDecision
	for _, d := range rb.Decisions {
		if d.DecisionType == DecisionDo {
			result = append(result, d)
		}
	}
	return result
}

// GetAVOIDDecisions returns all AVOID decisions.
func (rb *RecommendationBundle) GetAVOIDDecisions() []ArbitratedDecision {
	var result []ArbitratedDecision
	for _, d := range rb.Decisions {
		if d.DecisionType == DecisionAvoid {
			result = append(result, d)
		}
	}
	return result
}

// HasCriticalAlerts returns true if there are critical alerts.
func (rb *RecommendationBundle) HasCriticalAlerts() bool {
	for _, alert := range rb.Alerts {
		if alert.Severity == "CRITICAL" {
			return true
		}
	}
	return false
}
