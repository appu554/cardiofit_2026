// Package models provides domain models for KB-19 Protocol Orchestrator.
package models

import (
	"time"

	"github.com/google/uuid"
)

// ArbitratedDecision represents the final decision made by the arbitration engine
// for a specific clinical action. This is the "heart object" of KB-19.
//
// Key characteristics:
// - Every decision has exactly one EvidenceEnvelope (legal protection)
// - Every decision is attributed to a source protocol
// - Every decision has an explicit rationale
// - Decisions that "lost" arbitration are marked with ArbitrationReason
type ArbitratedDecision struct {
	// Unique identifier for this decision
	ID uuid.UUID `json:"id"`

	// Type of decision (DO, DELAY, AVOID, CONSIDER)
	DecisionType DecisionType `json:"decision_type"`

	// Target of the decision (drug name, procedure, action)
	Target string `json:"target"`

	// RxNorm code if applicable
	TargetRxNorm string `json:"target_rxnorm,omitempty"`

	// SNOMED code if applicable
	TargetSNOMED string `json:"target_snomed,omitempty"`

	// Human-readable rationale for this decision
	Rationale string `json:"rationale"`

	// Safety flags that affected this decision
	SafetyFlags []SafetyFlag `json:"safety_flags,omitempty"`

	// Decision IDs that this decision depends on
	// (e.g., "start anticoagulant" depends on "stop aspirin" completing)
	Dependencies []uuid.UUID `json:"dependencies,omitempty"`

	// Full evidence envelope (legal protection)
	Evidence EvidenceEnvelope `json:"evidence"`

	// Protocol that generated this decision
	SourceProtocol   string `json:"source_protocol"`
	SourceProtocolID string `json:"source_protocol_id"`

	// Why this decision was chosen (especially relevant for conflict losers)
	ArbitrationReason string `json:"arbitration_reason"`

	// If this decision "lost" a conflict, what won instead
	ConflictedWith string `json:"conflicted_with,omitempty"`
	ConflictType   string `json:"conflict_type,omitempty"`

	// Urgency of this decision
	Urgency ActionUrgency `json:"urgency"`

	// Specific actions to take (bound to KB-3/KB-12)
	Actions []BoundAction `json:"actions"`

	// Monitoring requirements
	MonitoringPlan []MonitoringItem `json:"monitoring_plan,omitempty"`

	// When this decision expires (for temporal validity)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`

	// Patient and encounter context
	PatientID   uuid.UUID `json:"patient_id"`
	EncounterID uuid.UUID `json:"encounter_id"`

	// Creation timestamp
	CreatedAt time.Time `json:"created_at"`

	// Whether this decision has been acknowledged by a clinician
	Acknowledged   bool       `json:"acknowledged"`
	AcknowledgedBy string     `json:"acknowledged_by,omitempty"`
	AcknowledgedAt *time.Time `json:"acknowledged_at,omitempty"`
}

// DecisionType categorizes what the arbitration engine decided.
type DecisionType string

const (
	// DecisionDo - Proceed with this action
	DecisionDo DecisionType = "DO"

	// DecisionDelay - Delay this action (due to conflict or timing)
	DecisionDelay DecisionType = "DELAY"

	// DecisionAvoid - Avoid this action (contraindicated or harmful)
	DecisionAvoid DecisionType = "AVOID"

	// DecisionConsider - Consider this action (optional, Class IIb)
	DecisionConsider DecisionType = "CONSIDER"
)

// SafetyFlag represents a safety concern identified during arbitration.
type SafetyFlag struct {
	// Type of safety flag
	Type SafetyFlagType `json:"type"`

	// Severity of the concern
	Severity string `json:"severity"` // WARNING, CAUTION, HARD_BLOCK

	// Reason for this flag
	Reason string `json:"reason"`

	// Source that generated this flag
	Source string `json:"source"` // e.g., "ICU_SAFETY_ENGINE", "PREGNANCY_CHECKER"

	// Whether this flag was overridden
	Overridden   bool   `json:"overridden"`
	OverrideNote string `json:"override_note,omitempty"`
}

// SafetyFlagType categorizes safety concerns.
type SafetyFlagType string

const (
	// FlagICUHardBlock - ICU safety engine hard block
	FlagICUHardBlock SafetyFlagType = "ICU_HARD_BLOCK"

	// FlagPregnancy - Pregnancy-related safety concern
	FlagPregnancy SafetyFlagType = "PREGNANCY"

	// FlagRenal - Renal dosing concern
	FlagRenal SafetyFlagType = "RENAL"

	// FlagHepatic - Hepatic dosing concern
	FlagHepatic SafetyFlagType = "HEPATIC"

	// FlagBleeding - Bleeding risk concern
	FlagBleeding SafetyFlagType = "BLEEDING"

	// FlagAllergy - Known allergy
	FlagAllergy SafetyFlagType = "ALLERGY"

	// FlagInteraction - Drug-drug interaction
	FlagInteraction SafetyFlagType = "INTERACTION"

	// FlagAgeRelated - Age-related concern (pediatric, geriatric)
	FlagAgeRelated SafetyFlagType = "AGE_RELATED"

	// FlagHighAlert - High-alert medication
	FlagHighAlert SafetyFlagType = "HIGH_ALERT"
)

// BoundAction represents an action that has been bound to execution services.
type BoundAction struct {
	// Original abstract action ID
	AbstractActionID string `json:"abstract_action_id"`

	// What type of binding
	BindingType string `json:"binding_type"` // KB3_TEMPORAL, KB12_ORDERSET, KB14_TASK

	// Reference to the bound entity
	BoundEntityID string `json:"bound_entity_id"`

	// Details of the binding
	Details map[string]interface{} `json:"details"`

	// Timing (from KB-3)
	ScheduledAt *time.Time `json:"scheduled_at,omitempty"`
	Deadline    *time.Time `json:"deadline,omitempty"`

	// Status of the bound action
	Status string `json:"status"` // PENDING, SCHEDULED, IN_PROGRESS, COMPLETED, CANCELLED
}

// MonitoringItem represents a monitoring requirement.
type MonitoringItem struct {
	// What to monitor (e.g., "INR", "Potassium", "Blood pressure")
	Parameter string `json:"parameter"`

	// Frequency of monitoring
	Frequency string `json:"frequency"` // e.g., "daily", "q6h", "weekly"

	// Target range
	TargetMin float64 `json:"target_min,omitempty"`
	TargetMax float64 `json:"target_max,omitempty"`

	// Duration of monitoring
	Duration string `json:"duration,omitempty"` // e.g., "7 days", "until stable"

	// Alert thresholds
	AlertIfBelow *float64 `json:"alert_if_below,omitempty"`
	AlertIfAbove *float64 `json:"alert_if_above,omitempty"`
}

// NewArbitratedDecision creates a new ArbitratedDecision with initialized values.
func NewArbitratedDecision(decisionType DecisionType, target, rationale string) *ArbitratedDecision {
	return &ArbitratedDecision{
		ID:           uuid.New(),
		DecisionType: decisionType,
		Target:       target,
		Rationale:    rationale,
		Evidence:     *NewEvidenceEnvelope(),
		SafetyFlags:  make([]SafetyFlag, 0),
		Actions:      make([]BoundAction, 0),
		CreatedAt:    time.Now(),
	}
}

// AddSafetyFlag adds a safety flag to the decision.
func (d *ArbitratedDecision) AddSafetyFlag(flagType SafetyFlagType, severity, reason, source string) {
	flag := SafetyFlag{
		Type:     flagType,
		Severity: severity,
		Reason:   reason,
		Source:   source,
	}
	d.SafetyFlags = append(d.SafetyFlags, flag)
}

// AddDependency adds a dependency on another decision.
func (d *ArbitratedDecision) AddDependency(decisionID uuid.UUID) {
	d.Dependencies = append(d.Dependencies, decisionID)
}

// SetConflict marks this decision as having lost a conflict.
func (d *ArbitratedDecision) SetConflict(winner, conflictType string) {
	d.ConflictedWith = winner
	d.ConflictType = conflictType
}

// AddBoundAction adds a bound action to this decision.
func (d *ArbitratedDecision) AddBoundAction(action BoundAction) {
	d.Actions = append(d.Actions, action)
}

// AddMonitoring adds a monitoring requirement.
func (d *ArbitratedDecision) AddMonitoring(item MonitoringItem) {
	d.MonitoringPlan = append(d.MonitoringPlan, item)
}

// HasHardBlock returns true if any safety flag is a hard block.
func (d *ArbitratedDecision) HasHardBlock() bool {
	for _, flag := range d.SafetyFlags {
		if flag.Severity == "HARD_BLOCK" && !flag.Overridden {
			return true
		}
	}
	return false
}

// IsActionable returns true if this is a DO or CONSIDER decision.
func (d *ArbitratedDecision) IsActionable() bool {
	return d.DecisionType == DecisionDo || d.DecisionType == DecisionConsider
}

// IsBlocked returns true if this is an AVOID decision or has hard blocks.
func (d *ArbitratedDecision) IsBlocked() bool {
	return d.DecisionType == DecisionAvoid || d.HasHardBlock()
}

// String returns the string representation of DecisionType.
func (dt DecisionType) String() string {
	switch dt {
	case DecisionDo:
		return "DO"
	case DecisionDelay:
		return "DELAY"
	case DecisionAvoid:
		return "AVOID"
	case DecisionConsider:
		return "CONSIDER"
	default:
		return "UNKNOWN"
	}
}
