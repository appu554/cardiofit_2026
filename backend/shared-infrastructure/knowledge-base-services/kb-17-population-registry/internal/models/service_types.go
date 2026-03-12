// Package models contains domain models for KB-17 Population Registry
package models

import (
	"time"
)

// Additional enrollment source constant used by services
const (
	EnrollmentSourceAutomatic EnrollmentSource = "AUTOMATIC"
	EnrollmentSourceImport    EnrollmentSource = "IMPORT"
)

// EnrollmentRequest represents a request to enroll a patient (used by service layer)
type EnrollmentRequest struct {
	PatientID    string               `json:"patient_id" binding:"required"`
	RegistryCode RegistryCode         `json:"registry_code" binding:"required"`
	Source       EnrollmentSource     `json:"source,omitempty"`
	Notes        string               `json:"notes,omitempty"`
	PatientData  *PatientClinicalData `json:"patient_data,omitempty"`
}

// EnrollmentFilters for querying enrollments
type EnrollmentFilters struct {
	RegistryCode RegistryCode     `json:"registry_code,omitempty"`
	Status       EnrollmentStatus `json:"status,omitempty"`
	RiskTiers    []RiskTier       `json:"risk_tiers,omitempty"`
	HasCareGaps  bool             `json:"has_care_gaps,omitempty"`
	Limit        int              `json:"limit,omitempty"`
	Offset       int              `json:"offset,omitempty"`
}

// BulkEnrollmentError represents a failed enrollment in a bulk operation
type BulkEnrollmentError struct {
	PatientID    string       `json:"patient_id"`
	RegistryCode RegistryCode `json:"registry_code"`
	Error        string       `json:"error"`
}

// EligibilityResult contains the result of patient eligibility evaluation
type EligibilityResult struct {
	PatientID           string                `json:"patient_id"`
	EvaluatedAt         time.Time             `json:"evaluated_at"`
	RegistryEligibility []RegistryEligibility `json:"registry_eligibility"`
	EvaluationDuration  time.Duration         `json:"evaluation_duration"`
}

// RegistryEligibility represents eligibility for a single registry
type RegistryEligibility struct {
	RegistryCode      RegistryCode     `json:"registry_code"`
	RegistryName      string           `json:"registry_name"`
	Eligible          bool             `json:"eligible"`
	MatchedCriteria   []string         `json:"matched_criteria,omitempty"`
	SuggestedRiskTier RiskTier         `json:"suggested_risk_tier,omitempty"`
	ConfidenceScore   float64          `json:"confidence_score,omitempty"`
	EvaluationError   string           `json:"evaluation_error,omitempty"`
	EvaluationDetails *CriteriaEvaluationResult `json:"evaluation_details,omitempty"`
}

// RiskAssessment represents a patient's risk assessment
type RiskAssessment struct {
	PatientID       string       `json:"patient_id"`
	RegistryCode    RegistryCode `json:"registry_code"`
	AssessedAt      time.Time    `json:"assessed_at"`
	RiskTier        RiskTier     `json:"risk_tier"`
	RiskScore       float64      `json:"risk_score"`
	RiskFactors     []string     `json:"risk_factors,omitempty"`
	ConfidenceScore float64      `json:"confidence_score"`
}

// ValidationResult represents the result of data validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Warnings []string `json:"warnings,omitempty"`
	Errors   []string `json:"errors,omitempty"`
}

// BatchEvaluationRequest represents a request to evaluate a patient
type BatchEvaluationRequest struct {
	PatientID   string               `json:"patient_id"`
	PatientData *PatientClinicalData `json:"patient_data"`
}

// BatchEvaluationResult represents the result of batch evaluation
type BatchEvaluationResult struct {
	TotalEvaluated int                    `json:"total_evaluated"`
	SuccessCount   int                    `json:"success_count"`
	FailedCount    int                    `json:"failed_count"`
	Results        []EligibilityResult    `json:"results"`
	Errors         []BatchEvaluationError `json:"errors,omitempty"`
}

// BatchEvaluationError represents a failed evaluation
type BatchEvaluationError struct {
	PatientID string `json:"patient_id"`
	Error     string `json:"error"`
}

// PatientClinicalEvent represents a clinical event for a patient
type PatientClinicalEvent struct {
	PatientID    string               `json:"patient_id"`
	EventType    string               `json:"event_type"`
	EventTime    time.Time            `json:"event_time"`
	ClinicalData *PatientClinicalData `json:"clinical_data"`
}

// Note: RegistryStats is defined in registry.go

// DailyEnrollment represents daily enrollment counts
type DailyEnrollment struct {
	Date  time.Time `json:"date"`
	Count int64     `json:"count"`
}

// Note: StatsSummary is defined in responses.go

// HighRiskSummary represents high-risk patient summary
type HighRiskSummary struct {
	CalculatedAt  time.Time               `json:"calculated_at"`
	TotalHighRisk int64                   `json:"total_high_risk"`
	ByRegistry    map[RegistryCode]int64  `json:"by_registry"`
	ByRiskTier    map[RiskTier]int64      `json:"by_risk_tier"`
}

// CareGapsSummary represents care gaps summary
type CareGapsSummary struct {
	CalculatedAt          time.Time               `json:"calculated_at"`
	TotalPatientsWithGaps int64                   `json:"total_patients_with_gaps"`
	TotalGaps             int64                   `json:"total_gaps"`
	ByRegistry            map[RegistryCode]int64  `json:"by_registry"`
	ByGapType             map[string]int64        `json:"by_gap_type"`
}

// EnrollmentTimeline represents enrollment timeline data
type EnrollmentTimeline struct {
	RegistryCode    RegistryCode      `json:"registry_code"`
	StartDate       time.Time         `json:"start_date"`
	EndDate         time.Time         `json:"end_date"`
	Enrollments     []DailyEnrollment `json:"enrollments"`
	Disenrollments  []DailyEnrollment `json:"disenrollments"`
	NetChange       int64             `json:"net_change"`
}

// RiskTransition represents a risk tier change
type RiskTransition struct {
	PatientID    string       `json:"patient_id"`
	RegistryCode RegistryCode `json:"registry_code"`
	OldTier      RiskTier     `json:"old_tier"`
	NewTier      RiskTier     `json:"new_tier"`
	ChangedAt    time.Time    `json:"changed_at"`
	Reason       string       `json:"reason,omitempty"`
}

// CareGapFrequency represents care gap frequency
type CareGapFrequency struct {
	MeasureID    string `json:"measure_id"`
	MeasureName  string `json:"measure_name"`
	PatientCount int64  `json:"patient_count"`
}

// RegistryComparison represents registry comparison data
type RegistryComparison struct {
	CalculatedAt time.Time                `json:"calculated_at"`
	Registries   []RegistryComparisonItem `json:"registries"`
}

// RegistryComparisonItem represents a single registry in comparison
type RegistryComparisonItem struct {
	RegistryCode      RegistryCode `json:"registry_code"`
	RegistryName      string       `json:"registry_name"`
	TotalEnrolled     int64        `json:"total_enrolled"`
	ActiveEnrollments int64        `json:"active_enrollments"`
	HighRiskPercent   float64      `json:"high_risk_percent"`
	CareGapPercent    float64      `json:"care_gap_percent"`
}

// Note: HighRiskPatientSummary and CareGapSummary are defined in responses.go

// CareGap represents a single care gap
type CareGap struct {
	MeasureID   string    `json:"measure_id"`
	MeasureName string    `json:"measure_name"`
	DueDate     time.Time `json:"due_date,omitempty"`
	Priority    string    `json:"priority,omitempty"`
	Status      string    `json:"status"`
}

// RegistryPatient extensions - add EligibilityData field
func (rp *RegistryPatient) SetEligibilityData(data *CriteriaEvaluationResult) {
	// Store eligibility data in metadata
	if rp.Metadata == nil {
		rp.Metadata = make(JSONMap)
	}
	rp.Metadata["eligibility_data"] = data
}
