// Package models provides domain models for KB-11 Population Health Engine.
package models

import (
	"time"

	"github.com/google/uuid"
)

// ──────────────────────────────────────────────────────────────────────────────
// API Request Models
// ──────────────────────────────────────────────────────────────────────────────

// PatientQueryRequest represents query parameters for patient projection searches.
type PatientQueryRequest struct {
	// Pagination
	Limit  int `form:"limit" binding:"omitempty,min=1,max=1000"`
	Offset int `form:"offset" binding:"omitempty,min=0"`

	// Filters
	RiskTier           *RiskTier `form:"risk_tier" binding:"omitempty"`
	AttributedPCP      *string   `form:"attributed_pcp" binding:"omitempty"`
	AttributedPractice *string   `form:"attributed_practice" binding:"omitempty"`
	MinCareGaps        *int      `form:"min_care_gaps" binding:"omitempty,min=0"`

	// Sort
	SortBy    string `form:"sort_by" binding:"omitempty,oneof=risk_score care_gaps last_synced created_at"`
	SortOrder string `form:"sort_order" binding:"omitempty,oneof=asc desc"`
}

// SetDefaults applies default values for missing parameters.
func (r *PatientQueryRequest) SetDefaults() {
	if r.Limit == 0 {
		r.Limit = 50
	}
	if r.SortBy == "" {
		r.SortBy = "risk_score"
	}
	if r.SortOrder == "" {
		r.SortOrder = "desc"
	}
}

// RiskCalculationRequest represents a request to calculate risk for a patient.
type RiskCalculationRequest struct {
	PatientFHIRID string        `json:"patient_fhir_id" binding:"required"`
	ModelName     RiskModelType `json:"model_name" binding:"required"`
	ForceRecalc   bool          `json:"force_recalc"`
}

// BatchRiskCalculationRequest represents a request to calculate risk for multiple patients.
type BatchRiskCalculationRequest struct {
	PatientFHIRIDs []string      `json:"patient_fhir_ids" binding:"required,min=1,max=100"`
	ModelName      RiskModelType `json:"model_name" binding:"required"`
	ForceRecalc    bool          `json:"force_recalc"`
}

// SyncRequest represents a request to sync patient data from an upstream source.
type SyncRequest struct {
	Source     SyncSource `json:"source" binding:"required"`
	FullSync   bool       `json:"full_sync"`
	MaxRecords int        `json:"max_records" binding:"omitempty,min=1,max=10000"`
}

// AttributionUpdateRequest represents a request to update patient attribution.
type AttributionUpdateRequest struct {
	PatientFHIRID      string     `json:"patient_fhir_id" binding:"required"`
	AttributedPCP      *string    `json:"attributed_pcp"`
	AttributedPractice *string    `json:"attributed_practice"`
	AttributionDate    *time.Time `json:"attribution_date"`
}

// BatchAttributionUpdateRequest represents a request to update multiple patient attributions.
type BatchAttributionUpdateRequest struct {
	Updates []AttributionUpdateRequest `json:"updates" binding:"required,min=1,max=100"`
}

// PopulationMetricsRequest represents query parameters for population metrics.
type PopulationMetricsRequest struct {
	// Filters
	AttributedPCP      *string `form:"attributed_pcp" binding:"omitempty"`
	AttributedPractice *string `form:"attributed_practice" binding:"omitempty"`

	// Grouping
	GroupBy string `form:"group_by" binding:"omitempty,oneof=practice pcp risk_tier"`
}

// CohortCriteria represents criteria for cohort definition.
type CohortCriteria struct {
	Field    string           `json:"field" binding:"required"`
	Operator CriteriaOperator `json:"operator" binding:"required"`
	Value    interface{}      `json:"value" binding:"required"`
}

// CohortDefinitionRequest represents a request to create a cohort.
type CohortDefinitionRequest struct {
	Name        string           `json:"name" binding:"required,min=3,max=100"`
	Description string           `json:"description" binding:"max=500"`
	Type        CohortType       `json:"type" binding:"required"`
	Criteria    []CohortCriteria `json:"criteria" binding:"required,min=1"`
}

// ──────────────────────────────────────────────────────────────────────────────
// API Response Models
// ──────────────────────────────────────────────────────────────────────────────

// PaginatedResponse wraps paginated results.
type PaginatedResponse struct {
	Data       interface{} `json:"data"`
	Total      int         `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	HasMore    bool        `json:"has_more"`
}

// NewPaginatedResponse creates a new paginated response.
func NewPaginatedResponse(data interface{}, total, limit, offset int) *PaginatedResponse {
	return &PaginatedResponse{
		Data:    data,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: offset+limit < total,
	}
}

// PatientProjectionResponse represents a patient projection in API responses.
type PatientProjectionResponse struct {
	ID                 uuid.UUID  `json:"id"`
	FHIRID             string     `json:"fhir_id"`
	MRN                *string    `json:"mrn,omitempty"`
	FullName           string     `json:"full_name"`
	Age                *int       `json:"age,omitempty"`
	Gender             *Gender    `json:"gender,omitempty"`
	AttributedPCP      *string    `json:"attributed_pcp,omitempty"`
	AttributedPractice *string    `json:"attributed_practice,omitempty"`
	CurrentRiskTier    RiskTier   `json:"current_risk_tier"`
	LatestRiskScore    *float64   `json:"latest_risk_score,omitempty"`
	CareGapCount       int        `json:"care_gap_count"`
	IsHighRisk         bool       `json:"is_high_risk"`
	LastSyncedAt       time.Time  `json:"last_synced_at"`
}

// FromPatientProjection converts a domain model to an API response.
func FromPatientProjection(pp *PatientProjection) *PatientProjectionResponse {
	return &PatientProjectionResponse{
		ID:                 pp.ID,
		FHIRID:             pp.FHIRID,
		MRN:                pp.MRN,
		FullName:           pp.FullName(),
		Age:                pp.Age(),
		Gender:             pp.Gender,
		AttributedPCP:      pp.AttributedPCP,
		AttributedPractice: pp.AttributedPractice,
		CurrentRiskTier:    pp.CurrentRiskTier,
		LatestRiskScore:    pp.LatestRiskScore,
		CareGapCount:       pp.CareGapCount,
		IsHighRisk:         pp.IsHighRisk(),
		LastSyncedAt:       pp.LastSyncedAt,
	}
}

// RiskAssessmentResponse represents a risk assessment in API responses.
type RiskAssessmentResponse struct {
	ID                  uuid.UUID          `json:"id"`
	PatientFHIRID       string             `json:"patient_fhir_id"`
	ModelName           string             `json:"model_name"`
	ModelVersion        string             `json:"model_version"`
	Score               float64            `json:"score"`
	RiskTier            RiskTier           `json:"risk_tier"`
	ContributingFactors map[string]float64 `json:"contributing_factors,omitempty"`
	CalculatedAt        time.Time          `json:"calculated_at"`
	ValidUntil          *time.Time         `json:"valid_until,omitempty"`
	GovernanceEventID   *uuid.UUID         `json:"governance_event_id,omitempty"`
}

// FromRiskAssessment converts a domain model to an API response.
func FromRiskAssessment(ra *RiskAssessment) *RiskAssessmentResponse {
	return &RiskAssessmentResponse{
		ID:                  ra.ID,
		PatientFHIRID:       ra.PatientFHIRID,
		ModelName:           ra.ModelName,
		ModelVersion:        ra.ModelVersion,
		Score:               ra.Score,
		RiskTier:            ra.RiskTier,
		ContributingFactors: ra.ContributingFactors,
		CalculatedAt:        ra.CalculatedAt,
		ValidUntil:          ra.ValidUntil,
		GovernanceEventID:   ra.GovernanceEventID,
	}
}

// PopulationMetricsResponse represents population metrics in API responses.
type PopulationMetricsResponse struct {
	TotalPatients       int                    `json:"total_patients"`
	RiskDistribution    map[string]int         `json:"risk_distribution"`
	HighRiskPercentage  float64                `json:"high_risk_percentage"`
	RisingRiskCount     int                    `json:"rising_risk_count"`
	AverageRiskScore    float64                `json:"average_risk_score"`
	CareGapDistribution map[string]int         `json:"care_gap_distribution,omitempty"`
	ByPractice          map[string]int         `json:"by_practice,omitempty"`
	ByPCP               map[string]int         `json:"by_pcp,omitempty"`
	CalculatedAt        time.Time              `json:"calculated_at"`
}

// FromPopulationMetrics converts a domain model to an API response.
func FromPopulationMetrics(pm *PopulationMetrics) *PopulationMetricsResponse {
	// Convert RiskTier keys to strings for JSON
	riskDist := make(map[string]int)
	for tier, count := range pm.RiskDistribution {
		riskDist[string(tier)] = count
	}

	return &PopulationMetricsResponse{
		TotalPatients:       pm.TotalPatients,
		RiskDistribution:    riskDist,
		HighRiskPercentage:  pm.HighRiskPercentage,
		RisingRiskCount:     pm.RisingRiskCount,
		AverageRiskScore:    pm.AverageRiskScore,
		CareGapDistribution: pm.CareGapDistribution,
		ByPractice:          pm.ByPractice,
		ByPCP:               pm.ByPCP,
		CalculatedAt:        pm.CalculatedAt,
	}
}

// SyncStatusResponse represents sync status in API responses.
type SyncStatusResponse struct {
	Source            SyncSource `json:"source"`
	Status            SyncStatus `json:"status"`
	LastSyncStarted   *time.Time `json:"last_sync_started,omitempty"`
	LastSyncCompleted *time.Time `json:"last_sync_completed,omitempty"`
	RecordsSynced     int        `json:"records_synced"`
	ErrorMessage      *string    `json:"error_message,omitempty"`
}

// FromSyncStatusRecord converts a domain model to an API response.
func FromSyncStatusRecord(ssr *SyncStatusRecord) *SyncStatusResponse {
	return &SyncStatusResponse{
		Source:            ssr.Source,
		Status:            ssr.LastSyncStatus,
		LastSyncStarted:   ssr.LastSyncStarted,
		LastSyncCompleted: ssr.LastSyncCompleted,
		RecordsSynced:     ssr.RecordsSynced,
		ErrorMessage:      ssr.ErrorMessage,
	}
}

// HealthResponse represents the health check response.
type HealthResponse struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Environment string            `json:"environment"`
	Uptime      string            `json:"uptime"`
	Checks      map[string]string `json:"checks"`
}

// ErrorResponse represents an API error response.
type ErrorResponse struct {
	Error   string `json:"error"`
	Code    string `json:"code,omitempty"`
	Details string `json:"details,omitempty"`
}

// NewErrorResponse creates a new error response.
func NewErrorResponse(err, code, details string) *ErrorResponse {
	return &ErrorResponse{
		Error:   err,
		Code:    code,
		Details: details,
	}
}
