// Package clients provides HTTP clients for KB services.
//
// KB17HTTPClient implements the KB17Client interface for KB-17 Population Registry.
// It provides patient registration, cohort management, program enrollment tracking,
// and longitudinal patient timeline data.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-17 is a RUNTIME category KB (Category B). It is called during workflow execution,
// NOT during snapshot build time. It does NOT provide data to CQL - it consumes CQL outputs.
//
// Per the CTO/CMO spec:
//   "CQL explains. KB-19 recommends. ICU decides."
//
// KB-17 provides population registry services that workflows (KB-14, KB-19) consume
// AFTER CQL classification and ICU veto checks have passed.
//
// Workflow Pattern:
//   1. CQL evaluates → produces classifications
//   2. ICU veto check → pass/reject
//   3. KB-17 called → register patients, manage cohorts
//
// Connects to: http://localhost:8097 (Docker: kb17-population-registry)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// KB17HTTPClient implements KB17Client by calling the KB-17 Population Registry REST API.
type KB17HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB17HTTPClient creates a new KB-17 HTTP client.
func NewKB17HTTPClient(baseURL string) *KB17HTTPClient {
	return &KB17HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB17HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB17HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB17HTTPClient {
	return &KB17HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB17Client Interface Implementation
// ============================================================================

// RegisterPatient adds or updates a patient in the population registry.
// Called during patient onboarding or demographic updates.
func (c *KB17HTTPClient) RegisterPatient(
	ctx context.Context,
	patient PatientRegistration,
) error {

	req := kb17RegisterRequest{
		PatientID:    patient.PatientID,
		Demographics: patient.Demographics,
		Region:       patient.Region,
		PrimaryCare:  patient.PrimaryCareProvider,
		Facility:     patient.FacilityID,
		RegistryTags: patient.Tags,
	}

	_, err := c.callKB17(ctx, "POST", "/api/v1/registry/patients", req)
	if err != nil {
		return fmt.Errorf("failed to register patient: %w", err)
	}

	return nil
}

// GetRegisteredPatients returns patients matching registry criteria.
// Used for cohort identification and population queries.
func (c *KB17HTTPClient) GetRegisteredPatients(
	ctx context.Context,
	criteria RegistryCriteria,
) ([]RegisteredPatient, error) {

	req := kb17QueryRequest{
		Conditions:   criteria.ConditionCodes,
		AgeRange:     criteria.AgeRange,
		Gender:       criteria.Gender,
		Region:       criteria.Region,
		Tags:         criteria.Tags,
		RiskLevel:    criteria.RiskLevel,
		Limit:        criteria.Limit,
		Offset:       criteria.Offset,
	}

	resp, err := c.callKB17(ctx, "POST", "/api/v1/registry/patients/search", req)
	if err != nil {
		return nil, fmt.Errorf("failed to query registry: %w", err)
	}

	var result kb17QueryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse registry response: %w", err)
	}

	patients := make([]RegisteredPatient, 0, len(result.Patients))
	for _, p := range result.Patients {
		patients = append(patients, RegisteredPatient{
			PatientID:           p.PatientID,
			RegistrationDate:    p.RegistrationDate,
			Demographics:        p.Demographics,
			RiskLevel:           p.RiskLevel,
			ActivePrograms:      p.ActivePrograms,
			PrimaryCareProvider: p.PrimaryCare,
			LastEncounterDate:   p.LastEncounter,
		})
	}

	return patients, nil
}

// GetPatientPrograms returns programs a patient is enrolled in.
// Used for care coordination and program management.
func (c *KB17HTTPClient) GetPatientPrograms(
	ctx context.Context,
	patientID string,
) ([]ProgramEnrollment, error) {

	endpoint := fmt.Sprintf("/api/v1/registry/patients/%s/programs", patientID)

	resp, err := c.callKB17(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient programs: %w", err)
	}

	var result kb17ProgramsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse programs response: %w", err)
	}

	programs := make([]ProgramEnrollment, 0, len(result.Programs))
	for _, p := range result.Programs {
		programs = append(programs, ProgramEnrollment{
			ProgramID:      p.ProgramID,
			ProgramName:    p.ProgramName,
			EnrollmentDate: p.EnrollmentDate,
			Status:         p.Status,
			CareManager:    p.CareManager,
			NextReviewDate: p.NextReview,
			Goals:          p.Goals,
		})
	}

	return programs, nil
}

// EnrollInProgram enrolls a patient in a care program.
// Called when eligibility is confirmed (after CQL classification).
func (c *KB17HTTPClient) EnrollInProgram(
	ctx context.Context,
	patientID string,
	programID string,
	enrollmentData ProgramEnrollmentRequest,
) (*ProgramEnrollment, error) {

	req := kb17EnrollRequest{
		PatientID:     patientID,
		ProgramID:     programID,
		CareManager:   enrollmentData.CareManagerID,
		ReferralSource: enrollmentData.ReferralSource,
		Goals:         enrollmentData.Goals,
		Notes:         enrollmentData.Notes,
	}

	resp, err := c.callKB17(ctx, "POST", "/api/v1/registry/enrollments", req)
	if err != nil {
		return nil, fmt.Errorf("failed to enroll patient: %w", err)
	}

	var result kb17EnrollResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse enrollment response: %w", err)
	}

	return &ProgramEnrollment{
		ProgramID:      result.ProgramID,
		ProgramName:    result.ProgramName,
		EnrollmentDate: result.EnrollmentDate,
		Status:         result.Status,
		CareManager:    result.CareManager,
	}, nil
}

// GetProgramEligibility determines if a patient is eligible for a program.
// Consumes CQL classification outputs to determine eligibility.
func (c *KB17HTTPClient) GetProgramEligibility(
	ctx context.Context,
	patientID string,
	programID string,
) (*EligibilityResult, error) {

	endpoint := fmt.Sprintf("/api/v1/registry/patients/%s/eligibility/%s", patientID, programID)

	resp, err := c.callKB17(ctx, "GET", endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to check eligibility: %w", err)
	}

	var result kb17EligibilityResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse eligibility response: %w", err)
	}

	return &EligibilityResult{
		IsEligible:      result.Eligible,
		Reason:          result.Reason,
		MetCriteria:     result.MetCriteria,
		UnmetCriteria:   result.UnmetCriteria,
		RecommendedDate: result.RecommendedEnrollmentDate,
	}, nil
}

// GetPatientTimeline returns longitudinal patient data for care coordination.
// Provides historical view of patient's care journey.
func (c *KB17HTTPClient) GetPatientTimeline(
	ctx context.Context,
	patientID string,
	startDate, endDate time.Time,
) (*PatientTimeline, error) {

	req := kb17TimelineRequest{
		PatientID: patientID,
		StartDate: startDate.Format(time.RFC3339),
		EndDate:   endDate.Format(time.RFC3339),
	}

	resp, err := c.callKB17(ctx, "POST", "/api/v1/registry/patients/timeline", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get patient timeline: %w", err)
	}

	var result kb17TimelineResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse timeline response: %w", err)
	}

	events := make([]TimelineEvent, 0, len(result.Events))
	for _, e := range result.Events {
		events = append(events, TimelineEvent{
			EventID:     e.EventID,
			EventType:   e.EventType,
			EventDate:   e.EventDate,
			Description: e.Description,
			Provider:    e.Provider,
			Facility:    e.Facility,
			Metadata:    e.Metadata,
		})
	}

	return &PatientTimeline{
		PatientID:    result.PatientID,
		TimelineSpan: Period{Start: &startDate, End: &endDate},
		Events:       events,
		TotalEvents:  result.TotalEvents,
	}, nil
}

// GetCohortMembers returns patients belonging to a defined cohort.
// Used for population health management and outreach.
func (c *KB17HTTPClient) GetCohortMembers(
	ctx context.Context,
	cohortID string,
	limit int,
) ([]string, error) {

	req := kb17CohortRequest{
		CohortID: cohortID,
		Limit:    limit,
	}

	resp, err := c.callKB17(ctx, "POST", "/api/v1/registry/cohorts/members", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get cohort members: %w", err)
	}

	var result kb17CohortResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cohort response: %w", err)
	}

	return result.PatientIDs, nil
}

// CreateCohort creates a new patient cohort based on criteria.
// Used for population segmentation and targeted interventions.
func (c *KB17HTTPClient) CreateCohort(
	ctx context.Context,
	cohort CohortDefinition,
) (*CohortInfo, error) {

	req := kb17CreateCohortRequest{
		Name:        cohort.Name,
		Description: cohort.Description,
		Criteria:    cohort.Criteria,
		Tags:        cohort.Tags,
		Owner:       cohort.OwnerID,
	}

	resp, err := c.callKB17(ctx, "POST", "/api/v1/registry/cohorts", req)
	if err != nil {
		return nil, fmt.Errorf("failed to create cohort: %w", err)
	}

	var result kb17CohortInfoResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse cohort info: %w", err)
	}

	return &CohortInfo{
		CohortID:    result.CohortID,
		Name:        result.Name,
		Description: result.Description,
		MemberCount: result.MemberCount,
		CreatedAt:   result.CreatedAt,
		UpdatedAt:   result.UpdatedAt,
	}, nil
}

// UpdatePatientRiskLevel updates a patient's risk stratification in registry.
// Called after KB-11 population health analysis.
func (c *KB17HTTPClient) UpdatePatientRiskLevel(
	ctx context.Context,
	patientID string,
	riskLevel string,
	riskFactors []string,
) error {

	req := kb17RiskUpdateRequest{
		PatientID:   patientID,
		RiskLevel:   riskLevel,
		RiskFactors: riskFactors,
		UpdatedAt:   time.Now(),
	}

	_, err := c.callKB17(ctx, "PUT", fmt.Sprintf("/api/v1/registry/patients/%s/risk", patientID), req)
	if err != nil {
		return fmt.Errorf("failed to update risk level: %w", err)
	}

	return nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB17HTTPClient) callKB17(ctx context.Context, method, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, method, url, bodyReader)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-17 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-17 service is healthy.
func (c *KB17HTTPClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-17 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// Contract Types (for KB-17 specific operations)
// ============================================================================

// PatientRegistration contains data for registering a patient.
type PatientRegistration struct {
	PatientID           string
	Demographics        contracts.PatientDemographics
	Region              string
	PrimaryCareProvider string
	FacilityID          string
	Tags                []string
}

// RegistryCriteria defines search criteria for registry queries.
type RegistryCriteria struct {
	ConditionCodes []string
	AgeRange       *AgeRange
	Gender         string
	Region         string
	Tags           []string
	RiskLevel      string
	Limit          int
	Offset         int
}

// AgeRange represents an age range filter.
type AgeRange struct {
	MinAge int `json:"min_age,omitempty"`
	MaxAge int `json:"max_age,omitempty"`
}

// RegisteredPatient represents a patient in the registry.
type RegisteredPatient struct {
	PatientID           string
	RegistrationDate    time.Time
	Demographics        contracts.PatientDemographics
	RiskLevel           string
	ActivePrograms      []string
	PrimaryCareProvider string
	LastEncounterDate   *time.Time
}

// ProgramEnrollment represents a patient's enrollment in a care program.
type ProgramEnrollment struct {
	ProgramID      string
	ProgramName    string
	EnrollmentDate time.Time
	Status         string // active, completed, withdrawn, pending
	CareManager    string
	NextReviewDate *time.Time
	Goals          []string
}

// ProgramEnrollmentRequest contains data for enrolling a patient.
type ProgramEnrollmentRequest struct {
	CareManagerID  string
	ReferralSource string
	Goals          []string
	Notes          string
}

// EligibilityResult contains program eligibility determination.
type EligibilityResult struct {
	IsEligible      bool
	Reason          string
	MetCriteria     []string
	UnmetCriteria   []string
	RecommendedDate *time.Time
}

// PatientTimeline represents longitudinal patient data.
type PatientTimeline struct {
	PatientID    string
	TimelineSpan Period
	Events       []TimelineEvent
	TotalEvents  int
}

// Period represents a time period.
type Period struct {
	Start *time.Time
	End   *time.Time
}

// TimelineEvent represents an event in patient timeline.
type TimelineEvent struct {
	EventID     string
	EventType   string // encounter, procedure, diagnosis, medication, lab, etc.
	EventDate   time.Time
	Description string
	Provider    string
	Facility    string
	Metadata    map[string]interface{}
}

// CohortDefinition contains criteria for creating a cohort.
type CohortDefinition struct {
	Name        string
	Description string
	Criteria    RegistryCriteria
	Tags        []string
	OwnerID     string
}

// CohortInfo contains cohort metadata.
type CohortInfo struct {
	CohortID    string
	Name        string
	Description string
	MemberCount int
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// ============================================================================
// KB-17 Request/Response Types (internal)
// ============================================================================

type kb17RegisterRequest struct {
	PatientID    string                        `json:"patient_id"`
	Demographics contracts.PatientDemographics `json:"demographics"`
	Region       string                        `json:"region"`
	PrimaryCare  string                        `json:"primary_care,omitempty"`
	Facility     string                        `json:"facility,omitempty"`
	RegistryTags []string                      `json:"tags,omitempty"`
}

type kb17QueryRequest struct {
	Conditions []string  `json:"conditions,omitempty"`
	AgeRange   *AgeRange `json:"age_range,omitempty"`
	Gender     string    `json:"gender,omitempty"`
	Region     string    `json:"region,omitempty"`
	Tags       []string  `json:"tags,omitempty"`
	RiskLevel  string    `json:"risk_level,omitempty"`
	Limit      int       `json:"limit,omitempty"`
	Offset     int       `json:"offset,omitempty"`
}

type kb17QueryResponse struct {
	Patients   []kb17Patient `json:"patients"`
	TotalCount int           `json:"total_count"`
}

type kb17Patient struct {
	PatientID        string                        `json:"patient_id"`
	RegistrationDate time.Time                     `json:"registration_date"`
	Demographics     contracts.PatientDemographics `json:"demographics"`
	RiskLevel        string                        `json:"risk_level"`
	ActivePrograms   []string                      `json:"active_programs"`
	PrimaryCare      string                        `json:"primary_care"`
	LastEncounter    *time.Time                    `json:"last_encounter,omitempty"`
}

type kb17ProgramsResponse struct {
	Programs []kb17Program `json:"programs"`
}

type kb17Program struct {
	ProgramID      string     `json:"program_id"`
	ProgramName    string     `json:"program_name"`
	EnrollmentDate time.Time  `json:"enrollment_date"`
	Status         string     `json:"status"`
	CareManager    string     `json:"care_manager"`
	NextReview     *time.Time `json:"next_review,omitempty"`
	Goals          []string   `json:"goals,omitempty"`
}

type kb17EnrollRequest struct {
	PatientID      string   `json:"patient_id"`
	ProgramID      string   `json:"program_id"`
	CareManager    string   `json:"care_manager,omitempty"`
	ReferralSource string   `json:"referral_source,omitempty"`
	Goals          []string `json:"goals,omitempty"`
	Notes          string   `json:"notes,omitempty"`
}

type kb17EnrollResponse struct {
	EnrollmentID   string    `json:"enrollment_id"`
	ProgramID      string    `json:"program_id"`
	ProgramName    string    `json:"program_name"`
	EnrollmentDate time.Time `json:"enrollment_date"`
	Status         string    `json:"status"`
	CareManager    string    `json:"care_manager"`
}

type kb17EligibilityResponse struct {
	Eligible                  bool       `json:"eligible"`
	Reason                    string     `json:"reason"`
	MetCriteria               []string   `json:"met_criteria"`
	UnmetCriteria             []string   `json:"unmet_criteria"`
	RecommendedEnrollmentDate *time.Time `json:"recommended_enrollment_date,omitempty"`
}

type kb17TimelineRequest struct {
	PatientID string `json:"patient_id"`
	StartDate string `json:"start_date"`
	EndDate   string `json:"end_date"`
}

type kb17TimelineResponse struct {
	PatientID   string            `json:"patient_id"`
	Events      []kb17Event       `json:"events"`
	TotalEvents int               `json:"total_events"`
}

type kb17Event struct {
	EventID     string                 `json:"event_id"`
	EventType   string                 `json:"event_type"`
	EventDate   time.Time              `json:"event_date"`
	Description string                 `json:"description"`
	Provider    string                 `json:"provider,omitempty"`
	Facility    string                 `json:"facility,omitempty"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type kb17CohortRequest struct {
	CohortID string `json:"cohort_id"`
	Limit    int    `json:"limit,omitempty"`
}

type kb17CohortResponse struct {
	CohortID   string   `json:"cohort_id"`
	CohortName string   `json:"cohort_name"`
	PatientIDs []string `json:"patient_ids"`
	TotalCount int      `json:"total_count"`
}

type kb17CreateCohortRequest struct {
	Name        string           `json:"name"`
	Description string           `json:"description"`
	Criteria    RegistryCriteria `json:"criteria"`
	Tags        []string         `json:"tags,omitempty"`
	Owner       string           `json:"owner,omitempty"`
}

type kb17CohortInfoResponse struct {
	CohortID    string    `json:"cohort_id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	MemberCount int       `json:"member_count"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type kb17RiskUpdateRequest struct {
	PatientID   string    `json:"patient_id"`
	RiskLevel   string    `json:"risk_level"`
	RiskFactors []string  `json:"risk_factors"`
	UpdatedAt   time.Time `json:"updated_at"`
}
