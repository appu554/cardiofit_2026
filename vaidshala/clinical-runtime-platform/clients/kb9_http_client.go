// Package clients provides HTTP clients for KB services.
//
// KB9HTTPClient implements the KB9Client interface for KB-9 Care Gaps Service.
// It provides care gap identification, tracking, and closure management.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-9 is a RUNTIME KB - called during workflow execution, NOT during snapshot build.
// It tracks care gaps identified by CQL evaluation and manages their lifecycle.
//
// Workflow Pattern:
// 1. CQL evaluates quality measures → produces measure results
// 2. KB-9 tracks care gaps from measure results → provides intervention tracking
// 3. Care gaps flow to KB-14 (Navigator) for workflow orchestration
//
// Connects to: http://localhost:8089 (Docker: kb9-care-gaps)
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

// KB9HTTPClient implements KB9Client by calling the KB-9 Care Gaps Service REST API.
type KB9HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB9HTTPClient creates a new KB-9 HTTP client.
func NewKB9HTTPClient(baseURL string) *KB9HTTPClient {
	return &KB9HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB9HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB9HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB9HTTPClient {
	return &KB9HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB9Client Interface Implementation (RUNTIME)
// ============================================================================

// GetActiveGaps returns open care gaps for a patient.
// These are gaps that have been identified but not yet closed.
//
// Parameters:
// - patientID: FHIR Patient.id for the patient
//
// Returns:
// - List of open care gaps with recommended actions
func (c *KB9HTTPClient) GetActiveGaps(
	ctx context.Context,
	patientID string,
) ([]contracts.CareGap, error) {

	req := kb9ActiveGapsRequest{
		PatientID:  patientID,
		ActiveOnly: true,
	}

	resp, err := c.callKB9(ctx, "/api/v1/gaps/active", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get active gaps: %w", err)
	}

	var result kb9GapsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse gaps response: %w", err)
	}

	gaps := make([]contracts.CareGap, 0, len(result.Gaps))
	for _, g := range result.Gaps {
		gaps = append(gaps, contracts.CareGap{
			MeasureID:         g.MeasureID,
			Description:       g.Description,
			Priority:          g.Priority,
			RecommendedAction: g.RecommendedAction,
		})
	}

	return gaps, nil
}

// GetMeasureStatus returns quality measure status for a patient and specific measure.
// Provides detailed information about measure compliance.
//
// Parameters:
// - patientID: FHIR Patient.id for the patient
// - measureID: Quality measure identifier (e.g., "CMS122", "CMS165")
//
// Returns:
// - MeasureStatus with population membership and gap status
func (c *KB9HTTPClient) GetMeasureStatus(
	ctx context.Context,
	patientID string,
	measureID string,
) (*MeasureStatus, error) {

	req := kb9MeasureStatusRequest{
		PatientID: patientID,
		MeasureID: measureID,
	}

	resp, err := c.callKB9(ctx, "/api/v1/measures/status", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get measure status: %w", err)
	}

	var result kb9MeasureStatusResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse measure status response: %w", err)
	}

	return &MeasureStatus{
		MeasureID:              result.MeasureID,
		MeasureName:            result.MeasureName,
		InInitialPopulation:    result.InInitialPopulation,
		InDenominator:          result.InDenominator,
		InNumerator:            result.InNumerator,
		InDenominatorExclusion: result.InDenominatorExclusion,
		InDenominatorException: result.InDenominatorException,
		HasGap:                 result.HasGap,
		GapDescription:         result.GapDescription,
		RecommendedActions:     result.RecommendedActions,
		LastEvaluatedAt:        result.LastEvaluatedAt,
	}, nil
}

// GetDueInterventions returns interventions due for a patient.
// Includes both scheduled and overdue care activities.
//
// Parameters:
// - patientID: FHIR Patient.id for the patient
//
// Returns:
// - List of due interventions with priority and due dates
func (c *KB9HTTPClient) GetDueInterventions(
	ctx context.Context,
	patientID string,
) ([]DueIntervention, error) {

	req := kb9InterventionsRequest{
		PatientID:     patientID,
		IncludeOverdue: true,
		IncludeDueSoon: true,
		DueSoonDays:   30,
	}

	resp, err := c.callKB9(ctx, "/api/v1/interventions/due", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get due interventions: %w", err)
	}

	var result kb9InterventionsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse interventions response: %w", err)
	}

	interventions := make([]DueIntervention, 0, len(result.Interventions))
	for _, i := range result.Interventions {
		interventions = append(interventions, DueIntervention{
			InterventionID:   i.InterventionID,
			MeasureID:        i.MeasureID,
			InterventionType: i.InterventionType,
			Description:      i.Description,
			Priority:         i.Priority,
			DueDate:          i.DueDate,
			IsOverdue:        i.IsOverdue,
			OverdueDays:      i.OverdueDays,
			SuggestedActions: i.SuggestedActions,
		})
	}

	return interventions, nil
}

// CloseGap marks a care gap as closed with evidence.
// Creates an audit trail for regulatory compliance.
//
// Parameters:
// - gapID: Unique identifier of the gap to close
// - closureReason: Reason for closure (e.g., "numerator_met", "exclusion_applied")
// - evidence: Evidence supporting gap closure (procedure codes, lab results, etc.)
func (c *KB9HTTPClient) CloseGap(
	ctx context.Context,
	gapID string,
	closureReason string,
	evidence GapEvidence,
) error {

	req := kb9CloseGapRequest{
		GapID:         gapID,
		ClosureReason: closureReason,
		Evidence:      evidence,
		ClosedAt:      time.Now().UTC(),
	}

	_, err := c.callKB9(ctx, "/api/v1/gaps/close", req)
	if err != nil {
		return fmt.Errorf("failed to close gap: %w", err)
	}

	return nil
}

// GetPopulationGaps returns aggregate gap analysis for a population.
// Used for population health dashboards and outreach prioritization.
//
// Parameters:
// - populationID: Identifier for the population cohort
//
// Returns:
// - PopulationGapSummary with aggregated gap statistics
func (c *KB9HTTPClient) GetPopulationGaps(
	ctx context.Context,
	populationID string,
) (*PopulationGapSummary, error) {

	req := kb9PopulationGapsRequest{
		PopulationID: populationID,
	}

	resp, err := c.callKB9(ctx, "/api/v1/population/gaps", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get population gaps: %w", err)
	}

	var result kb9PopulationGapsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse population gaps response: %w", err)
	}

	measureSummaries := make([]MeasureGapSummary, 0, len(result.MeasureSummaries))
	for _, m := range result.MeasureSummaries {
		measureSummaries = append(measureSummaries, MeasureGapSummary{
			MeasureID:          m.MeasureID,
			MeasureName:        m.MeasureName,
			TotalEligible:      m.TotalEligible,
			TotalWithGap:       m.TotalWithGap,
			GapRate:            m.GapRate,
			TrendDirection:     m.TrendDirection,
			PriorityPatients:   m.PriorityPatients,
		})
	}

	return &PopulationGapSummary{
		PopulationID:     result.PopulationID,
		PopulationName:   result.PopulationName,
		TotalPatients:    result.TotalPatients,
		PatientsWithGaps: result.PatientsWithGaps,
		TotalGaps:        result.TotalGaps,
		MeasureSummaries: measureSummaries,
		GeneratedAt:      result.GeneratedAt,
	}, nil
}

// GetGapHistory returns gap history for a patient including closed gaps.
// Used for audit trails and trend analysis.
func (c *KB9HTTPClient) GetGapHistory(
	ctx context.Context,
	patientID string,
	startDate time.Time,
	endDate time.Time,
) ([]GapHistoryEntry, error) {

	req := kb9GapHistoryRequest{
		PatientID: patientID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	resp, err := c.callKB9(ctx, "/api/v1/gaps/history", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get gap history: %w", err)
	}

	var result kb9GapHistoryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse gap history response: %w", err)
	}

	history := make([]GapHistoryEntry, 0, len(result.History))
	for _, h := range result.History {
		history = append(history, GapHistoryEntry{
			GapID:          h.GapID,
			MeasureID:      h.MeasureID,
			OpenedAt:       h.OpenedAt,
			ClosedAt:       h.ClosedAt,
			Status:         h.Status,
			ClosureReason:  h.ClosureReason,
			DaysOpen:       h.DaysOpen,
		})
	}

	return history, nil
}

// GetOutreachPriority returns prioritized list of patients for outreach.
// Used by care coordinators for efficient outreach targeting.
func (c *KB9HTTPClient) GetOutreachPriority(
	ctx context.Context,
	populationID string,
	maxPatients int,
) ([]OutreachPriority, error) {

	req := kb9OutreachRequest{
		PopulationID: populationID,
		MaxPatients:  maxPatients,
	}

	resp, err := c.callKB9(ctx, "/api/v1/outreach/priority", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get outreach priority: %w", err)
	}

	var result kb9OutreachResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse outreach response: %w", err)
	}

	priorities := make([]OutreachPriority, 0, len(result.Patients))
	for _, p := range result.Patients {
		priorities = append(priorities, OutreachPriority{
			PatientID:       p.PatientID,
			PriorityScore:   p.PriorityScore,
			PriorityReason:  p.PriorityReason,
			GapCount:        p.GapCount,
			HighPriorityGaps: p.HighPriorityGaps,
			LastOutreach:    p.LastOutreach,
			SuggestedTopic:  p.SuggestedTopic,
		})
	}

	return priorities, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB9HTTPClient) callKB9(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+endpoint, bytes.NewBuffer(jsonBody))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-9 returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *KB9HTTPClient) callKB9Get(ctx context.Context, endpoint string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-9 returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ============================================================================
// KB-9 Specific Types
// ============================================================================

// MeasureStatus represents the status of a quality measure for a patient.
type MeasureStatus struct {
	MeasureID              string    `json:"measureId"`
	MeasureName            string    `json:"measureName"`
	InInitialPopulation    bool      `json:"inInitialPopulation"`
	InDenominator          bool      `json:"inDenominator"`
	InNumerator            bool      `json:"inNumerator"`
	InDenominatorExclusion bool      `json:"inDenominatorExclusion"`
	InDenominatorException bool      `json:"inDenominatorException"`
	HasGap                 bool      `json:"hasGap"`
	GapDescription         string    `json:"gapDescription,omitempty"`
	RecommendedActions     []string  `json:"recommendedActions,omitempty"`
	LastEvaluatedAt        time.Time `json:"lastEvaluatedAt"`
}

// DueIntervention represents an intervention that is due or overdue.
type DueIntervention struct {
	InterventionID   string    `json:"interventionId"`
	MeasureID        string    `json:"measureId"`
	InterventionType string    `json:"interventionType"`
	Description      string    `json:"description"`
	Priority         string    `json:"priority"`
	DueDate          time.Time `json:"dueDate"`
	IsOverdue        bool      `json:"isOverdue"`
	OverdueDays      int       `json:"overdueDays,omitempty"`
	SuggestedActions []string  `json:"suggestedActions,omitempty"`
}

// GapEvidence represents evidence supporting gap closure.
type GapEvidence struct {
	EvidenceType   string                 `json:"evidenceType"`
	ResourceType   string                 `json:"resourceType,omitempty"`
	ResourceID     string                 `json:"resourceId,omitempty"`
	PerformedDate  *time.Time             `json:"performedDate,omitempty"`
	ResultValue    string                 `json:"resultValue,omitempty"`
	ProviderID     string                 `json:"providerId,omitempty"`
	DocumentedBy   string                 `json:"documentedBy,omitempty"`
	AdditionalData map[string]interface{} `json:"additionalData,omitempty"`
}

// PopulationGapSummary contains aggregate gap statistics for a population.
type PopulationGapSummary struct {
	PopulationID     string              `json:"populationId"`
	PopulationName   string              `json:"populationName"`
	TotalPatients    int                 `json:"totalPatients"`
	PatientsWithGaps int                 `json:"patientsWithGaps"`
	TotalGaps        int                 `json:"totalGaps"`
	MeasureSummaries []MeasureGapSummary `json:"measureSummaries"`
	GeneratedAt      time.Time           `json:"generatedAt"`
}

// MeasureGapSummary contains gap statistics for a single measure.
type MeasureGapSummary struct {
	MeasureID        string   `json:"measureId"`
	MeasureName      string   `json:"measureName"`
	TotalEligible    int      `json:"totalEligible"`
	TotalWithGap     int      `json:"totalWithGap"`
	GapRate          float64  `json:"gapRate"`
	TrendDirection   string   `json:"trendDirection,omitempty"`
	PriorityPatients []string `json:"priorityPatients,omitempty"`
}

// GapHistoryEntry represents a historical gap record.
type GapHistoryEntry struct {
	GapID         string     `json:"gapId"`
	MeasureID     string     `json:"measureId"`
	OpenedAt      time.Time  `json:"openedAt"`
	ClosedAt      *time.Time `json:"closedAt,omitempty"`
	Status        string     `json:"status"`
	ClosureReason string     `json:"closureReason,omitempty"`
	DaysOpen      int        `json:"daysOpen"`
}

// OutreachPriority represents a patient prioritized for outreach.
type OutreachPriority struct {
	PatientID        string     `json:"patientId"`
	PriorityScore    float64    `json:"priorityScore"`
	PriorityReason   string     `json:"priorityReason"`
	GapCount         int        `json:"gapCount"`
	HighPriorityGaps []string   `json:"highPriorityGaps,omitempty"`
	LastOutreach     *time.Time `json:"lastOutreach,omitempty"`
	SuggestedTopic   string     `json:"suggestedTopic,omitempty"`
}

// ============================================================================
// Internal Request/Response Types
// ============================================================================

type kb9ActiveGapsRequest struct {
	PatientID  string `json:"patientId"`
	ActiveOnly bool   `json:"activeOnly"`
}

type kb9GapsResponse struct {
	Gaps []struct {
		GapID             string    `json:"gapId"`
		MeasureID         string    `json:"measureId"`
		Description       string    `json:"description"`
		Priority          string    `json:"priority"`
		RecommendedAction string    `json:"recommendedAction"`
		OpenedAt          time.Time `json:"openedAt"`
	} `json:"gaps"`
}

type kb9MeasureStatusRequest struct {
	PatientID string `json:"patientId"`
	MeasureID string `json:"measureId"`
}

type kb9MeasureStatusResponse struct {
	MeasureID              string    `json:"measureId"`
	MeasureName            string    `json:"measureName"`
	InInitialPopulation    bool      `json:"inInitialPopulation"`
	InDenominator          bool      `json:"inDenominator"`
	InNumerator            bool      `json:"inNumerator"`
	InDenominatorExclusion bool      `json:"inDenominatorExclusion"`
	InDenominatorException bool      `json:"inDenominatorException"`
	HasGap                 bool      `json:"hasGap"`
	GapDescription         string    `json:"gapDescription"`
	RecommendedActions     []string  `json:"recommendedActions"`
	LastEvaluatedAt        time.Time `json:"lastEvaluatedAt"`
}

type kb9InterventionsRequest struct {
	PatientID      string `json:"patientId"`
	IncludeOverdue bool   `json:"includeOverdue"`
	IncludeDueSoon bool   `json:"includeDueSoon"`
	DueSoonDays    int    `json:"dueSoonDays"`
}

type kb9InterventionsResponse struct {
	Interventions []struct {
		InterventionID   string    `json:"interventionId"`
		MeasureID        string    `json:"measureId"`
		InterventionType string    `json:"interventionType"`
		Description      string    `json:"description"`
		Priority         string    `json:"priority"`
		DueDate          time.Time `json:"dueDate"`
		IsOverdue        bool      `json:"isOverdue"`
		OverdueDays      int       `json:"overdueDays"`
		SuggestedActions []string  `json:"suggestedActions"`
	} `json:"interventions"`
}

type kb9CloseGapRequest struct {
	GapID         string      `json:"gapId"`
	ClosureReason string      `json:"closureReason"`
	Evidence      GapEvidence `json:"evidence"`
	ClosedAt      time.Time   `json:"closedAt"`
}

type kb9PopulationGapsRequest struct {
	PopulationID string `json:"populationId"`
}

type kb9PopulationGapsResponse struct {
	PopulationID     string `json:"populationId"`
	PopulationName   string `json:"populationName"`
	TotalPatients    int    `json:"totalPatients"`
	PatientsWithGaps int    `json:"patientsWithGaps"`
	TotalGaps        int    `json:"totalGaps"`
	MeasureSummaries []struct {
		MeasureID        string   `json:"measureId"`
		MeasureName      string   `json:"measureName"`
		TotalEligible    int      `json:"totalEligible"`
		TotalWithGap     int      `json:"totalWithGap"`
		GapRate          float64  `json:"gapRate"`
		TrendDirection   string   `json:"trendDirection"`
		PriorityPatients []string `json:"priorityPatients"`
	} `json:"measureSummaries"`
	GeneratedAt time.Time `json:"generatedAt"`
}

type kb9GapHistoryRequest struct {
	PatientID string    `json:"patientId"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type kb9GapHistoryResponse struct {
	History []struct {
		GapID         string     `json:"gapId"`
		MeasureID     string     `json:"measureId"`
		OpenedAt      time.Time  `json:"openedAt"`
		ClosedAt      *time.Time `json:"closedAt"`
		Status        string     `json:"status"`
		ClosureReason string     `json:"closureReason"`
		DaysOpen      int        `json:"daysOpen"`
	} `json:"history"`
}

type kb9OutreachRequest struct {
	PopulationID string `json:"populationId"`
	MaxPatients  int    `json:"maxPatients"`
}

type kb9OutreachResponse struct {
	Patients []struct {
		PatientID        string     `json:"patientId"`
		PriorityScore    float64    `json:"priorityScore"`
		PriorityReason   string     `json:"priorityReason"`
		GapCount         int        `json:"gapCount"`
		HighPriorityGaps []string   `json:"highPriorityGaps"`
		LastOutreach     *time.Time `json:"lastOutreach"`
		SuggestedTopic   string     `json:"suggestedTopic"`
	} `json:"patients"`
}

// HealthCheck verifies KB-9 service is healthy.
func (c *KB9HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-9 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}
