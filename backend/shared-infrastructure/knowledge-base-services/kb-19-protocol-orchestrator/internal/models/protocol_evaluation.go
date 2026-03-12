// Package models provides domain models for KB-19 Protocol Orchestrator.
package models

// ProtocolEvaluation represents the result of evaluating a single protocol
// against a patient's clinical context. It captures whether the protocol
// is applicable, any contraindications, and recommended actions.
type ProtocolEvaluation struct {
	// Protocol being evaluated
	ProtocolID   string `json:"protocol_id"`
	ProtocolName string `json:"protocol_name"`

	// Whether this protocol is applicable to the patient
	IsApplicable        bool   `json:"is_applicable"`
	ApplicabilityReason string `json:"applicability_reason"`

	// Whether this protocol is contraindicated
	Contraindicated         bool     `json:"contraindicated"`
	ContraindicationReasons []string `json:"contraindication_reasons,omitempty"`

	// Actions recommended by this protocol
	RecommendedActions []AbstractAction `json:"recommended_actions"`

	// Priority class inherited from ProtocolDescriptor
	PriorityClass PriorityClass `json:"priority_class"`

	// Impact on patient risk scores (positive = increased risk, negative = decreased risk)
	RiskScoreImpact float64 `json:"risk_score_impact"`

	// CQL facts that were used in this evaluation (for audit trail)
	CQLFactsUsed []string `json:"cql_facts_used"`

	// Calculator scores that were used in this evaluation
	CalculatorsUsed map[string]float64 `json:"calculators_used,omitempty"`

	// Confidence in this evaluation (0.0 - 1.0)
	Confidence float64 `json:"confidence"`

	// Notes or warnings from the evaluation
	Notes []string `json:"notes,omitempty"`
}

// AbstractAction represents a recommended clinical action from a protocol.
// This is "abstract" because it doesn't specify exact dosing or timing -
// those details come from KB-1 (dosing) and KB-3 (timing).
type AbstractAction struct {
	// Unique ID for this action within the protocol
	ActionID string `json:"action_id"`

	// Type of action
	ActionType ActionType `json:"action_type"`

	// Target of the action (e.g., drug name, procedure code)
	Target string `json:"target"`

	// RxNorm code if this is a medication action
	RxNormCode string `json:"rxnorm_code,omitempty"`

	// SNOMED code for procedures
	SNOMEDCode string `json:"snomed_code,omitempty"`

	// Description of the action
	Description string `json:"description"`

	// Urgency of this action
	Urgency ActionUrgency `json:"urgency"`

	// Whether this action is conditional on something else
	IsConditional bool   `json:"is_conditional"`
	Condition     string `json:"condition,omitempty"`

	// Expected outcome of this action
	ExpectedOutcome string `json:"expected_outcome,omitempty"`

	// Monitoring requirements
	MonitoringRequired []string `json:"monitoring_required,omitempty"`
}

// ActionType categorizes clinical actions.
type ActionType string

const (
	// ActionMedicationStart - Start a new medication
	ActionMedicationStart ActionType = "MEDICATION_START"

	// ActionMedicationStop - Discontinue a medication
	ActionMedicationStop ActionType = "MEDICATION_STOP"

	// ActionMedicationModify - Modify dosing of existing medication
	ActionMedicationModify ActionType = "MEDICATION_MODIFY"

	// ActionLabOrder - Order laboratory tests
	ActionLabOrder ActionType = "LAB_ORDER"

	// ActionProcedure - Perform a procedure
	ActionProcedure ActionType = "PROCEDURE"

	// ActionConsult - Request specialty consultation
	ActionConsult ActionType = "CONSULT"

	// ActionMonitor - Set up monitoring
	ActionMonitor ActionType = "MONITOR"

	// ActionEducation - Patient education
	ActionEducation ActionType = "EDUCATION"

	// ActionAlert - Generate alert for clinical team
	ActionAlert ActionType = "ALERT"

	// ActionDisposition - Change patient disposition (admit, discharge, transfer)
	ActionDisposition ActionType = "DISPOSITION"
)

// ActionUrgency indicates how quickly an action should be performed.
type ActionUrgency string

const (
	// UrgencySTAT - Within minutes (life-threatening)
	UrgencySTAT ActionUrgency = "STAT"

	// UrgencyUrgent - Within hours
	UrgencyUrgent ActionUrgency = "URGENT"

	// UrgencyRoutine - Within 24 hours
	UrgencyRoutine ActionUrgency = "ROUTINE"

	// UrgencyScheduled - Can be scheduled at convenience
	UrgencyScheduled ActionUrgency = "SCHEDULED"
)

// NewProtocolEvaluation creates a new ProtocolEvaluation.
func NewProtocolEvaluation(protocolID, protocolName string) *ProtocolEvaluation {
	return &ProtocolEvaluation{
		ProtocolID:         protocolID,
		ProtocolName:       protocolName,
		RecommendedActions: make([]AbstractAction, 0),
		CQLFactsUsed:       make([]string, 0),
		CalculatorsUsed:    make(map[string]float64),
		Confidence:         1.0,
	}
}

// MarkApplicable marks the protocol as applicable with a reason.
func (pe *ProtocolEvaluation) MarkApplicable(reason string) {
	pe.IsApplicable = true
	pe.ApplicabilityReason = reason
}

// MarkNotApplicable marks the protocol as not applicable with a reason.
func (pe *ProtocolEvaluation) MarkNotApplicable(reason string) {
	pe.IsApplicable = false
	pe.ApplicabilityReason = reason
}

// AddContraindication adds a contraindication reason.
func (pe *ProtocolEvaluation) AddContraindication(reason string) {
	pe.Contraindicated = true
	pe.ContraindicationReasons = append(pe.ContraindicationReasons, reason)
}

// AddAction adds a recommended action.
func (pe *ProtocolEvaluation) AddAction(action AbstractAction) {
	pe.RecommendedActions = append(pe.RecommendedActions, action)
}

// RecordCQLFact records a CQL fact that was used in this evaluation.
func (pe *ProtocolEvaluation) RecordCQLFact(factID string) {
	pe.CQLFactsUsed = append(pe.CQLFactsUsed, factID)
}

// RecordCalculator records a calculator score that was used.
func (pe *ProtocolEvaluation) RecordCalculator(calculatorID string, score float64) {
	pe.CalculatorsUsed[calculatorID] = score
}

// HasSTATActions returns true if any action requires STAT urgency.
func (pe *ProtocolEvaluation) HasSTATActions() bool {
	for _, action := range pe.RecommendedActions {
		if action.Urgency == UrgencySTAT {
			return true
		}
	}
	return false
}

// GetMedicationActions returns all medication-related actions.
func (pe *ProtocolEvaluation) GetMedicationActions() []AbstractAction {
	var meds []AbstractAction
	for _, action := range pe.RecommendedActions {
		if action.ActionType == ActionMedicationStart ||
			action.ActionType == ActionMedicationStop ||
			action.ActionType == ActionMedicationModify {
			meds = append(meds, action)
		}
	}
	return meds
}
