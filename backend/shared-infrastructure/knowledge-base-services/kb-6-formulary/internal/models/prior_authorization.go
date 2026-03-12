package models

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// PA CRITERIA TYPES
// =============================================================================

// PACriterionType defines the type of PA criterion
type PACriterionType string

const (
	PACriterionDiagnosis       PACriterionType = "DIAGNOSIS"
	PACriterionLab             PACriterionType = "LAB"
	PACriterionPriorTherapy    PACriterionType = "PRIOR_THERAPY"
	PACriterionAge             PACriterionType = "AGE"
	PACriterionContraindication PACriterionType = "CONTRAINDICATION"
	PACriterionCustom          PACriterionType = "CUSTOM"
)

// PAStatus defines the status of a PA submission
type PAStatus string

const (
	PAStatusPending     PAStatus = "PENDING"
	PAStatusUnderReview PAStatus = "UNDER_REVIEW"
	PAStatusApproved    PAStatus = "APPROVED"
	PAStatusDenied      PAStatus = "DENIED"
	PAStatusNeedInfo    PAStatus = "NEED_INFO"
	PAStatusExpired     PAStatus = "EXPIRED"
	PAStatusCancelled   PAStatus = "CANCELLED"
)

// PAUrgencyLevel defines urgency levels for PA processing
type PAUrgencyLevel string

const (
	PAUrgencyStandard  PAUrgencyLevel = "STANDARD"
	PAUrgencyUrgent    PAUrgencyLevel = "URGENT"
	PAUrgencyExpedited PAUrgencyLevel = "EXPEDITED"
)

// =============================================================================
// PA CRITERIA MODELS
// =============================================================================

// PACriterion represents a single clinical criterion for PA evaluation
type PACriterion struct {
	Type              PACriterionType `json:"type"`
	Codes             []string        `json:"codes,omitempty"`               // Diagnosis codes (ICD-10, etc.)
	CodeSystem        string          `json:"code_system,omitempty"`         // ICD10, CPT, SNOMED
	Test              string          `json:"test,omitempty"`                // Lab test name
	LOINC             string          `json:"loinc,omitempty"`               // LOINC code for lab
	Operator          string          `json:"operator,omitempty"`            // >, <, =, >=, <=
	Value             float64         `json:"value,omitempty"`               // Threshold value
	Unit              string          `json:"unit,omitempty"`                // Unit of measure
	MaxAgeDays        int             `json:"max_age_days,omitempty"`        // Max age of lab result
	DrugClass         string          `json:"drug_class,omitempty"`          // Drug class name
	RxNormCodes       []string        `json:"rxnorm_codes,omitempty"`        // Specific drug RxNorm codes
	MinDurationDays   int             `json:"min_duration_days,omitempty"`   // Minimum therapy duration
	OrContraindication bool           `json:"or_contraindication,omitempty"` // Accept contraindication as alternative
	Conditions        []string        `json:"conditions,omitempty"`          // Contraindication conditions
	Action            string          `json:"action,omitempty"`              // deny, exempt, etc.
	Check             string          `json:"check,omitempty"`               // Custom check name
	Description       string          `json:"description"`                   // Human-readable description
	Required          bool            `json:"required,omitempty"`            // If criterion is mandatory
}

// PARequirement represents PA requirements for a drug
type PARequirement struct {
	ID                   uuid.UUID      `json:"id" db:"id"`
	DrugRxNorm           string         `json:"drug_rxnorm" db:"drug_rxnorm"`
	DrugName             string         `json:"drug_name" db:"drug_name"`
	PayerID              *string        `json:"payer_id" db:"payer_id"`
	PlanID               *string        `json:"plan_id" db:"plan_id"`
	Criteria             []PACriterion  `json:"criteria"`
	ApprovalDurationDays int            `json:"approval_duration_days" db:"approval_duration_days"`
	RenewalAllowed       bool           `json:"renewal_allowed" db:"renewal_allowed"`
	MaxRenewals          *int           `json:"max_renewals" db:"max_renewals"`
	RequiredDocuments    []string       `json:"required_documents" db:"required_documents"`
	UrgencyLevels        []string       `json:"urgency_levels" db:"urgency_levels"`
	StandardReviewHours  int            `json:"standard_review_hours" db:"standard_review_hours"`
	UrgentReviewHours    int            `json:"urgent_review_hours" db:"urgent_review_hours"`
	ExpeditedReviewHours int            `json:"expedited_review_hours" db:"expedited_review_hours"`
	EffectiveDate        time.Time      `json:"effective_date" db:"effective_date"`
	TerminationDate      *time.Time     `json:"termination_date" db:"termination_date"`
	Version              int            `json:"version" db:"version"`
	CreatedAt            time.Time      `json:"created_at" db:"created_at"`
	UpdatedAt            time.Time      `json:"updated_at" db:"updated_at"`
}

// =============================================================================
// PA SUBMISSION MODELS
// =============================================================================

// LabResult represents a patient lab result for PA evaluation
type LabResult struct {
	Test   string    `json:"test"`
	LOINC  string    `json:"loinc,omitempty"`
	Value  float64   `json:"value"`
	Unit   string    `json:"unit,omitempty"`
	Date   time.Time `json:"date"`
}

// DrugHistory represents patient medication history
type DrugHistory struct {
	RxNormCode  string     `json:"rxnorm_code"`
	DrugName    string     `json:"drug_name"`
	StartDate   time.Time  `json:"start_date"`
	EndDate     *time.Time `json:"end_date,omitempty"`
	DurationDays int       `json:"duration_days,omitempty"`
	Dose        string     `json:"dose,omitempty"`
	Frequency   string     `json:"frequency,omitempty"`
}

// DiagnosisCode represents a patient diagnosis
type DiagnosisCode struct {
	Code        string    `json:"code"`
	System      string    `json:"system"` // ICD10, ICD9, SNOMED
	Description string    `json:"description,omitempty"`
	Date        time.Time `json:"date,omitempty"`
	Primary     bool      `json:"primary,omitempty"`
}

// ClinicalDocumentation represents clinical documentation for PA
type ClinicalDocumentation struct {
	Diagnoses     []DiagnosisCode `json:"diagnoses,omitempty"`
	LabResults    []LabResult     `json:"lab_results,omitempty"`
	PriorTherapy  []DrugHistory   `json:"prior_therapy,omitempty"`
	ClinicalNotes string          `json:"clinical_notes,omitempty"`
}

// PASubmission represents a PA submission request
type PASubmission struct {
	ID                    uuid.UUID             `json:"id" db:"id"`
	ExternalID            *string               `json:"external_id" db:"external_id"`
	PatientID             string                `json:"patient_id" db:"patient_id"`
	ProviderID            string                `json:"provider_id" db:"provider_id"`
	ProviderNPI           *string               `json:"provider_npi" db:"provider_npi"`
	DrugRxNorm            string                `json:"drug_rxnorm" db:"drug_rxnorm"`
	DrugName              string                `json:"drug_name" db:"drug_name"`
	Quantity              int                   `json:"quantity" db:"quantity"`
	DaysSupply            int                   `json:"days_supply" db:"days_supply"`
	ClinicalDocumentation ClinicalDocumentation `json:"clinical_documentation"`
	PayerID               *string               `json:"payer_id" db:"payer_id"`
	PlanID                *string               `json:"plan_id" db:"plan_id"`
	MemberID              *string               `json:"member_id" db:"member_id"`
	Status                PAStatus              `json:"status" db:"status"`
	UrgencyLevel          PAUrgencyLevel        `json:"urgency_level" db:"urgency_level"`
	DecisionReason        *string               `json:"decision_reason" db:"decision_reason"`
	ApprovedQuantity      *int                  `json:"approved_quantity" db:"approved_quantity"`
	ApprovedDaysSupply    *int                  `json:"approved_days_supply" db:"approved_days_supply"`
	SubmittedAt           time.Time             `json:"submitted_at" db:"submitted_at"`
	ReviewedAt            *time.Time            `json:"reviewed_at" db:"reviewed_at"`
	DecisionAt            *time.Time            `json:"decision_at" db:"decision_at"`
	ExpiresAt             *time.Time            `json:"expires_at" db:"expires_at"`
	CreatedBy             *string               `json:"created_by" db:"created_by"`
	ReviewedBy            *string               `json:"reviewed_by" db:"reviewed_by"`
}

// =============================================================================
// PA EVALUATION MODELS
// =============================================================================

// CriterionEvaluation represents the evaluation of a single criterion
type CriterionEvaluation struct {
	Criterion   PACriterion `json:"criterion"`
	Met         bool        `json:"met"`
	Evidence    interface{} `json:"evidence,omitempty"`
	Notes       string      `json:"notes,omitempty"`
	EvaluatedAt time.Time   `json:"evaluated_at"`
}

// PAEvaluation represents the full PA evaluation result
type PAEvaluation struct {
	SubmissionID    uuid.UUID             `json:"submission_id"`
	RequirementID   uuid.UUID             `json:"requirement_id"`
	AllCriteriaMet  bool                  `json:"all_criteria_met"`
	CriteriaResults []CriterionEvaluation `json:"criteria_results"`
	RecommendedStatus PAStatus            `json:"recommended_status"`
	EvaluatedAt     time.Time             `json:"evaluated_at"`
	Notes           string                `json:"notes,omitempty"`
}

// =============================================================================
// REQUEST/RESPONSE MODELS
// =============================================================================

// PARequirementsRequest represents a request for PA requirements
type PARequirementsRequest struct {
	DrugRxNorm string  `json:"drug_rxnorm" binding:"required"`
	PayerID    *string `json:"payer_id,omitempty"`
	PlanID     *string `json:"plan_id,omitempty"`
}

// PARequirementsResponse represents PA requirements for a drug
type PARequirementsResponse struct {
	PARequired        bool           `json:"pa_required"`
	DrugRxNorm        string         `json:"drug_rxnorm"`
	DrugName          string         `json:"drug_name"`
	Criteria          []PACriterion  `json:"criteria,omitempty"`
	RequiredDocuments []string       `json:"required_documents,omitempty"`
	ApprovalDuration  int            `json:"approval_duration_days,omitempty"`
	UrgencyLevels     []string       `json:"urgency_levels,omitempty"`
	ReviewTimeframes  ReviewTimeframes `json:"review_timeframes,omitempty"`

	// Enhancement #1: Policy Binding (Tier-7 Governance Integration)
	PolicyBinding     *PolicyBinding `json:"policy_binding,omitempty"`
}

// ReviewTimeframes represents PA review timeframes
type ReviewTimeframes struct {
	StandardHours  int `json:"standard_hours"`
	UrgentHours    int `json:"urgent_hours"`
	ExpeditedHours int `json:"expedited_hours"`
}

// PACheckRequest represents a PA check request with patient context
type PACheckRequest struct {
	DrugRxNorm   string                `json:"drug_rxnorm" binding:"required"`
	PatientID    string                `json:"patient_id" binding:"required"`
	PayerID      *string               `json:"payer_id,omitempty"`
	PlanID       *string               `json:"plan_id,omitempty"`
	Diagnoses    []DiagnosisCode       `json:"diagnoses,omitempty"`
	LabResults   []LabResult           `json:"lab_results,omitempty"`
	PriorTherapy []DrugHistory         `json:"prior_therapy,omitempty"`
	PatientAge   *int                  `json:"patient_age,omitempty"`
}

// PACheckResponse represents the result of a PA check
type PACheckResponse struct {
	PARequired       bool                  `json:"pa_required"`
	PAStatus         string                `json:"pa_status"` // pre_approved, requires_submission, denied
	CriteriaMet      bool                  `json:"criteria_met"`
	CriteriaResults  []CriterionEvaluation `json:"criteria_results,omitempty"`
	MissingCriteria  []PACriterion         `json:"missing_criteria,omitempty"`
	RequiredDocuments []string             `json:"required_documents,omitempty"`
	Message          string                `json:"message"`
	ExistingApproval *PASubmission         `json:"existing_approval,omitempty"`

	// Enhancement #1: Policy Binding (Tier-7 Governance Integration)
	PolicyBinding    *PolicyBinding        `json:"policy_binding,omitempty"`
}

// PASubmitRequest represents a PA submission request
type PASubmitRequest struct {
	PatientID             string                `json:"patient_id" binding:"required"`
	ProviderID            string                `json:"provider_id" binding:"required"`
	ProviderNPI           *string               `json:"provider_npi,omitempty"`
	DrugRxNorm            string                `json:"drug_rxnorm" binding:"required"`
	Quantity              int                   `json:"quantity" binding:"required"`
	DaysSupply            int                   `json:"days_supply" binding:"required"`
	ClinicalDocumentation ClinicalDocumentation `json:"clinical_documentation" binding:"required"`
	PayerID               *string               `json:"payer_id,omitempty"`
	PlanID                *string               `json:"plan_id,omitempty"`
	MemberID              *string               `json:"member_id,omitempty"`
	UrgencyLevel          PAUrgencyLevel        `json:"urgency_level,omitempty"`
	ClinicalNotes         string                `json:"clinical_notes,omitempty"`
}

// PAStatusRequest represents a PA status check request
type PAStatusRequest struct {
	PAID string `json:"pa_id" binding:"required"`
}

// PAStatusResponse represents a PA status response
type PAStatusResponse struct {
	Submission      PASubmission          `json:"submission"`
	CriteriaResults []CriterionEvaluation `json:"criteria_results,omitempty"`
	Message         string                `json:"message"`
}

// =============================================================================
// JSON MARSHALING HELPERS
// =============================================================================

// UnmarshalCriteria unmarshals JSONB criteria to []PACriterion
func UnmarshalCriteria(data []byte) ([]PACriterion, error) {
	var criteria []PACriterion
	if err := json.Unmarshal(data, &criteria); err != nil {
		return nil, err
	}
	return criteria, nil
}

// MarshalCriteria marshals []PACriterion to JSONB
func MarshalCriteria(criteria []PACriterion) ([]byte, error) {
	return json.Marshal(criteria)
}
