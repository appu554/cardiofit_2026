package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// STEP THERAPY TYPES
// =============================================================================

// STOverrideReason defines valid override reasons for step therapy
type STOverrideReason string

const (
	STOverrideContraindication    STOverrideReason = "contraindication"
	STOverrideAdverseReaction     STOverrideReason = "adverse_reaction"
	STOverrideTreatmentFailure    STOverrideReason = "treatment_failure"
	STOverrideMedicalNecessity    STOverrideReason = "medical_necessity"
	STOverrideDrugInteraction     STOverrideReason = "drug_interaction"
	STOverrideRenalImpairment     STOverrideReason = "renal_impairment"
	STOverrideHepaticImpairment   STOverrideReason = "hepatic_impairment"
	STOverridePregnancy           STOverrideReason = "pregnancy"
	STOverrideAgeRestriction      STOverrideReason = "age_restriction"
	STOverrideOther               STOverrideReason = "other"
)

// STOverrideStatus defines the status of an override request
type STOverrideStatus string

const (
	STOverridePending   STOverrideStatus = "PENDING"
	STOverrideApproved  STOverrideStatus = "APPROVED"
	STOverrideDenied    STOverrideStatus = "DENIED"
	STOverrideExpired   STOverrideStatus = "EXPIRED"
	STOverrideCancelled STOverrideStatus = "CANCELLED"
)

// =============================================================================
// STEP THERAPY MODELS
// =============================================================================

// Step represents a single step in step therapy
type Step struct {
	StepNumber      int      `json:"step_number"`
	DrugClass       string   `json:"drug_class"`
	Description     string   `json:"description"`
	RxNormCodes     []string `json:"rxnorm_codes,omitempty"`
	MinDurationDays int      `json:"min_duration_days"`
	MaxDurationDays *int     `json:"max_duration_days,omitempty"`
	RequiredDose    *string  `json:"required_dose,omitempty"`
	AllowAnyInClass bool     `json:"allow_any_in_class,omitempty"`
	Rationale       string   `json:"rationale,omitempty"`
}

// StepTherapyRule represents step therapy requirements for a drug
type StepTherapyRule struct {
	ID                      uuid.UUID  `json:"id" db:"id"`
	TargetDrugRxNorm        string     `json:"target_drug_rxnorm" db:"target_drug_rxnorm"`
	TargetDrugName          string     `json:"target_drug_name" db:"target_drug_name"`
	PayerID                 *string    `json:"payer_id" db:"payer_id"`
	PlanID                  *string    `json:"plan_id" db:"plan_id"`
	Steps                   []Step     `json:"steps"`
	OverrideCriteria        []string   `json:"override_criteria" db:"override_criteria"`
	ExceptionDiagnosisCodes []string   `json:"exception_diagnosis_codes" db:"exception_diagnosis_codes"`
	ProtocolName            *string    `json:"protocol_name" db:"protocol_name"`
	ProtocolVersion         *string    `json:"protocol_version" db:"protocol_version"`
	EvidenceLevel           *string    `json:"evidence_level" db:"evidence_level"`
	EffectiveDate           time.Time  `json:"effective_date" db:"effective_date"`
	TerminationDate         *time.Time `json:"termination_date" db:"termination_date"`
	Version                 int        `json:"version" db:"version"`
	CreatedAt               time.Time  `json:"created_at" db:"created_at"`
	UpdatedAt               time.Time  `json:"updated_at" db:"updated_at"`
}

// StepTherapyCheck represents a step therapy check result
type StepTherapyCheck struct {
	ID                  uuid.UUID   `json:"id" db:"id"`
	PatientID           string      `json:"patient_id" db:"patient_id"`
	ProviderID          *string     `json:"provider_id" db:"provider_id"`
	TargetDrugRxNorm    string      `json:"target_drug_rxnorm" db:"target_drug_rxnorm"`
	TargetDrugName      string      `json:"target_drug_name" db:"target_drug_name"`
	PayerID             *string     `json:"payer_id" db:"payer_id"`
	PlanID              *string     `json:"plan_id" db:"plan_id"`
	DrugHistory         []DrugHistory `json:"drug_history"`
	StepTherapyRequired bool        `json:"step_therapy_required" db:"step_therapy_required"`
	TotalSteps          *int        `json:"total_steps" db:"total_steps"`
	StepsSatisfied      []int       `json:"steps_satisfied" db:"steps_satisfied"`
	CurrentStep         *int        `json:"current_step" db:"current_step"`
	Approved            bool        `json:"approved" db:"approved"`
	OverrideRequested   bool        `json:"override_requested" db:"override_requested"`
	OverrideReason      *string     `json:"override_reason" db:"override_reason"`
	OverrideApproved    *bool       `json:"override_approved" db:"override_approved"`
	Message             string      `json:"message" db:"message"`
	NextRequiredStep    *Step       `json:"next_required_step"`
	RuleID              *uuid.UUID  `json:"rule_id" db:"rule_id"`
	CheckedAt           time.Time   `json:"checked_at" db:"checked_at"`
}

// StepTherapyOverride represents an override request for step therapy
type StepTherapyOverride struct {
	ID                      uuid.UUID       `json:"id" db:"id"`
	CheckID                 *uuid.UUID      `json:"check_id" db:"check_id"`
	PatientID               string          `json:"patient_id" db:"patient_id"`
	ProviderID              string          `json:"provider_id" db:"provider_id"`
	TargetDrugRxNorm        string          `json:"target_drug_rxnorm" db:"target_drug_rxnorm"`
	OverrideReason          STOverrideReason `json:"override_reason" db:"override_reason"`
	ClinicalJustification   string          `json:"clinical_justification" db:"clinical_justification"`
	SupportingDocumentation interface{}     `json:"supporting_documentation" db:"supporting_documentation"`
	Status                  STOverrideStatus `json:"status" db:"status"`
	DecisionReason          *string         `json:"decision_reason" db:"decision_reason"`
	SubmittedAt             time.Time       `json:"submitted_at" db:"submitted_at"`
	ReviewedAt              *time.Time      `json:"reviewed_at" db:"reviewed_at"`
	DecisionAt              *time.Time      `json:"decision_at" db:"decision_at"`
	ExpiresAt               *time.Time      `json:"expires_at" db:"expires_at"`
	SubmittedBy             *string         `json:"submitted_by" db:"submitted_by"`
	ReviewedBy              *string         `json:"reviewed_by" db:"reviewed_by"`
}

// StepEvaluation represents the evaluation of a single step
type StepEvaluation struct {
	Step           Step          `json:"step"`
	Satisfied      bool          `json:"satisfied"`
	MatchingDrugs  []DrugHistory `json:"matching_drugs,omitempty"`
	DurationMet    bool          `json:"duration_met"`
	ActualDuration int           `json:"actual_duration,omitempty"`
	Notes          string        `json:"notes,omitempty"`
}

// =============================================================================
// REQUEST/RESPONSE MODELS
// =============================================================================

// STRequirementsRequest represents a request for ST requirements
type STRequirementsRequest struct {
	DrugRxNorm string  `json:"drug_rxnorm" binding:"required"`
	PayerID    *string `json:"payer_id,omitempty"`
	PlanID     *string `json:"plan_id,omitempty"`
}

// STRequirementsResponse represents ST requirements for a drug
type STRequirementsResponse struct {
	STRequired        bool           `json:"st_required"`
	DrugRxNorm        string         `json:"drug_rxnorm"`
	DrugName          string         `json:"drug_name"`
	Steps             []Step         `json:"steps,omitempty"`
	TotalSteps        int            `json:"total_steps"`
	OverrideCriteria  []string       `json:"override_criteria,omitempty"`
	ExceptionCodes    []string       `json:"exception_diagnosis_codes,omitempty"`
	ProtocolName      string         `json:"protocol_name,omitempty"`
	EvidenceLevel     string         `json:"evidence_level,omitempty"`

	// Enhancement #1: Policy Binding (Tier-7 Governance Integration)
	PolicyBinding     *PolicyBinding `json:"policy_binding,omitempty"`
}

// STCheckRequest represents a step therapy check request
type STCheckRequest struct {
	PatientID    string        `json:"patient_id" binding:"required"`
	DrugRxNorm   string        `json:"drug_rxnorm" binding:"required"`
	DrugHistory  []DrugHistory `json:"drug_history"`
	PayerID      *string       `json:"payer_id,omitempty"`
	PlanID       *string       `json:"plan_id,omitempty"`
	Diagnoses    []DiagnosisCode `json:"diagnoses,omitempty"`
}

// STCheckResponse represents the result of a step therapy check
type STCheckResponse struct {
	StepTherapyRequired bool             `json:"step_therapy_required"`
	Approved            bool             `json:"approved"`
	TotalSteps          int              `json:"total_steps"`
	CurrentStep         int              `json:"current_step"`
	StepsSatisfied      []int            `json:"steps_satisfied"`
	StepEvaluations     []StepEvaluation `json:"step_evaluations,omitempty"`
	NextRequiredStep    *Step            `json:"next_required_step,omitempty"`
	OverrideAvailable   bool             `json:"override_available"`
	OverrideCriteria    []string         `json:"override_criteria,omitempty"`
	ExceptionApplies    bool             `json:"exception_applies"`
	Message             string           `json:"message"`

	// Enhancement #1: Policy Binding (Tier-7 Governance Integration)
	PolicyBinding       *PolicyBinding   `json:"policy_binding,omitempty"`
}

// STOverrideRequest represents a step therapy override request
type STOverrideRequest struct {
	PatientID             string           `json:"patient_id" binding:"required"`
	ProviderID            string           `json:"provider_id" binding:"required"`
	DrugRxNorm            string           `json:"drug_rxnorm" binding:"required"`
	OverrideReason        STOverrideReason `json:"override_reason" binding:"required"`
	ClinicalJustification string           `json:"clinical_justification" binding:"required"`
	SupportingDocuments   []string         `json:"supporting_documents,omitempty"`
	CheckID               *uuid.UUID       `json:"check_id,omitempty"`
}

// STOverrideResponse represents a step therapy override response
type STOverrideResponse struct {
	Override StepTherapyOverride `json:"override"`
	Message  string              `json:"message"`
}

// =============================================================================
// JSON MARSHALING HELPERS
// =============================================================================

// UnmarshalSteps unmarshals JSONB steps to []Step
func UnmarshalSteps(data []byte) ([]Step, error) {
	var steps []Step
	if err := json.Unmarshal(data, &steps); err != nil {
		return nil, err
	}
	return steps, nil
}

// MarshalSteps marshals []Step to JSONB
func MarshalSteps(steps []Step) ([]byte, error) {
	return json.Marshal(steps)
}
