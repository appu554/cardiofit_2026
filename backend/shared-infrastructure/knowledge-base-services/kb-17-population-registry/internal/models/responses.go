// Package models contains domain models for KB-17 Population Registry
package models

import "time"

// APIResponse is a generic API response wrapper
type APIResponse struct {
	Success   bool        `json:"success"`
	Data      interface{} `json:"data,omitempty"`
	Error     string      `json:"error,omitempty"`
	Message   string      `json:"message,omitempty"`
	Timestamp time.Time   `json:"timestamp"`
}

// NewSuccessResponse creates a success response
func NewSuccessResponse(data interface{}) *APIResponse {
	return &APIResponse{
		Success:   true,
		Data:      data,
		Timestamp: time.Now().UTC(),
	}
}

// NewErrorResponse creates an error response
func NewErrorResponse(err string) *APIResponse {
	return &APIResponse{
		Success:   false,
		Error:     err,
		Timestamp: time.Now().UTC(),
	}
}

// NewMessageResponse creates a message response
func NewMessageResponse(message string) *APIResponse {
	return &APIResponse{
		Success:   true,
		Message:   message,
		Timestamp: time.Now().UTC(),
	}
}

// HealthResponse represents the health check response
type HealthResponse struct {
	Status      string            `json:"status"`
	Service     string            `json:"service"`
	Version     string            `json:"version"`
	Uptime      string            `json:"uptime"`
	Checks      map[string]string `json:"checks,omitempty"`
	Timestamp   time.Time         `json:"timestamp"`
}

// PaginatedResponse wraps paginated data
type PaginatedResponse struct {
	Success    bool        `json:"success"`
	Data       interface{} `json:"data"`
	Total      int64       `json:"total"`
	Limit      int         `json:"limit"`
	Offset     int         `json:"offset"`
	HasMore    bool        `json:"has_more"`
	Error      string      `json:"error,omitempty"`
}

// NewPaginatedResponse creates a paginated response
func NewPaginatedResponse(data interface{}, total int64, limit, offset int) *PaginatedResponse {
	hasMore := int64(offset+limit) < total
	return &PaginatedResponse{
		Success: true,
		Data:    data,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: hasMore,
	}
}

// HighRiskPatientSummary represents a high-risk patient summary
type HighRiskPatientSummary struct {
	PatientID       string           `json:"patient_id"`
	RegistryCode    RegistryCode     `json:"registry_code"`
	RiskTier        RiskTier         `json:"risk_tier"`
	CareGapCount    int              `json:"care_gap_count"`
	KeyMetrics      map[string]interface{} `json:"key_metrics,omitempty"`
	EnrolledAt      time.Time        `json:"enrolled_at"`
	LastEvaluatedAt *time.Time       `json:"last_evaluated_at,omitempty"`
}

// HighRiskResponse wraps high-risk patient data
type HighRiskResponse struct {
	Success   bool                     `json:"success"`
	Data      []HighRiskPatientSummary `json:"data"`
	Total     int64                    `json:"total"`
	ByTier    map[RiskTier]int64       `json:"by_tier"`
	Error     string                   `json:"error,omitempty"`
}

// CareGapSummary represents a patient with care gaps
type CareGapSummary struct {
	PatientID    string       `json:"patient_id"`
	RegistryCode RegistryCode `json:"registry_code"`
	CareGaps     []string     `json:"care_gaps"`
	RiskTier     RiskTier     `json:"risk_tier"`
	EnrolledAt   time.Time    `json:"enrolled_at"`
}

// CareGapResponse wraps care gap data
type CareGapResponse struct {
	Success       bool             `json:"success"`
	Data          []CareGapSummary `json:"data"`
	Total         int64            `json:"total"`
	ByRegistry    map[RegistryCode]int64 `json:"by_registry"`
	Error         string           `json:"error,omitempty"`
}

// StatsResponse wraps registry statistics
type StatsResponse struct {
	Success bool           `json:"success"`
	Data    *RegistryStats `json:"data,omitempty"`
	Error   string         `json:"error,omitempty"`
}

// AllStatsResponse wraps all registry statistics
type AllStatsResponse struct {
	Success  bool            `json:"success"`
	Data     []RegistryStats `json:"data,omitempty"`
	Summary  *StatsSummary   `json:"summary,omitempty"`
	Error    string          `json:"error,omitempty"`
}

// StatsSummary provides aggregate statistics
type StatsSummary struct {
	TotalRegistries   int   `json:"total_registries"`
	TotalEnrollments  int64 `json:"total_enrollments"`
	ActiveEnrollments int64 `json:"active_enrollments"`
	HighRiskPatients  int64 `json:"high_risk_patients"`
	PatientsWithGaps  int64 `json:"patients_with_care_gaps"`
}

// PatientRegistriesResponse wraps patient registry data
type PatientRegistriesResponse struct {
	Success     bool              `json:"success"`
	PatientID   string            `json:"patient_id"`
	Enrollments []RegistryPatient `json:"enrollments"`
	Total       int               `json:"total"`
	Error       string            `json:"error,omitempty"`
}

// ValidationError represents a field validation error
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ValidationErrorResponse wraps validation errors
type ValidationErrorResponse struct {
	Success bool              `json:"success"`
	Errors  []ValidationError `json:"errors"`
}

// NewValidationErrorResponse creates a validation error response
func NewValidationErrorResponse(errors []ValidationError) *ValidationErrorResponse {
	return &ValidationErrorResponse{
		Success: false,
		Errors:  errors,
	}
}
