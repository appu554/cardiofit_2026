// Package clients provides HTTP clients for KB services.
//
// KB13HTTPClient implements the KB13Client interface for KB-13 Quality Measures Service.
// It provides quality measure evaluation, reporting, and analytics.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-13 is a RUNTIME KB - called during workflow execution, NOT during snapshot build.
// It executes quality measure logic and produces HEDIS/CMS measure reports.
//
// Workflow Pattern:
// 1. CQL evaluates clinical facts → produces clinical truths
// 2. KB-13 consumes facts → determines measure populations
// 3. KB-9 tracks resulting care gaps
//
// Connects to: http://localhost:8093 (Docker: kb13-quality-measures)
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

// KB13HTTPClient implements KB13Client by calling the KB-13 Quality Measures Service REST API.
type KB13HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB13HTTPClient creates a new KB-13 HTTP client.
func NewKB13HTTPClient(baseURL string) *KB13HTTPClient {
	return &KB13HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 60 * time.Second, // Measure evaluation can take longer
		},
	}
}

// NewKB13HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB13HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB13HTTPClient {
	return &KB13HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB13Client Interface Implementation (RUNTIME)
// ============================================================================

// EvaluateMeasure evaluates a quality measure for a single patient.
// This is the core measure evaluation function.
//
// Parameters:
// - patientID: FHIR Patient.id for the patient
// - measureID: Quality measure identifier (e.g., "CMS122", "CMS165", "CMS2")
//
// Returns:
// - MeasureResult with population membership and care gap status
func (c *KB13HTTPClient) EvaluateMeasure(
	ctx context.Context,
	patientID string,
	measureID string,
) (*contracts.MeasureResult, error) {

	req := kb13EvaluateRequest{
		PatientID:   patientID,
		MeasureID:   measureID,
		EvaluatedAt: time.Now().UTC(),
	}

	resp, err := c.callKB13(ctx, "/api/v1/measures/evaluate", req)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate measure: %w", err)
	}

	var result kb13EvaluateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse measure result: %w", err)
	}

	return &contracts.MeasureResult{
		MeasureID:              result.MeasureID,
		MeasureName:            result.MeasureName,
		InInitialPopulation:    result.InInitialPopulation,
		InDenominator:          result.InDenominator,
		InNumerator:            result.InNumerator,
		InDenominatorExclusion: result.InDenominatorExclusion,
		InDenominatorException: result.InDenominatorException,
		CareGapIdentified:      result.CareGapIdentified,
		MeasureVersion:         result.MeasureVersion,
		LogicVersion:           result.LogicVersion,
		ELMCorrespondence:      result.ELMCorrespondence,
		EvaluatedAt:            result.EvaluatedAt,
		Rationale:              result.Rationale,
	}, nil
}

// GetMeasureDefinition returns the specification for a quality measure.
// Used for displaying measure details and requirements.
//
// Parameters:
// - measureID: Quality measure identifier (e.g., "CMS122")
//
// Returns:
// - MeasureDefinition with full measure specification
func (c *KB13HTTPClient) GetMeasureDefinition(
	ctx context.Context,
	measureID string,
) (*MeasureDefinition, error) {

	resp, err := c.callKB13Get(ctx, fmt.Sprintf("/api/v1/measures/%s/definition", measureID))
	if err != nil {
		return nil, fmt.Errorf("failed to get measure definition: %w", err)
	}

	var result kb13MeasureDefResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse measure definition: %w", err)
	}

	return &MeasureDefinition{
		MeasureID:              result.MeasureID,
		MeasureName:            result.MeasureName,
		MeasureVersion:         result.MeasureVersion,
		Description:            result.Description,
		Category:               result.Category,
		MeasureType:            result.MeasureType,
		NQFNumber:              result.NQFNumber,
		CMSNumber:              result.CMSNumber,
		EligibilityCriteria:    result.EligibilityCriteria,
		NumeratorCriteria:      result.NumeratorCriteria,
		DenominatorCriteria:    result.DenominatorCriteria,
		ExclusionCriteria:      result.ExclusionCriteria,
		ExceptionCriteria:      result.ExceptionCriteria,
		RecommendedActions:     result.RecommendedActions,
		EvidenceGrade:          result.EvidenceGrade,
		SourceURL:              result.SourceURL,
	}, nil
}

// GetPerformanceRate returns aggregate performance for a measure across a population.
// Used for quality dashboards and reporting.
//
// Parameters:
// - populationID: Identifier for the population cohort
// - measureID: Quality measure identifier
//
// Returns:
// - PerformanceRate with numerator/denominator and rate calculation
func (c *KB13HTTPClient) GetPerformanceRate(
	ctx context.Context,
	populationID string,
	measureID string,
) (*PerformanceRate, error) {

	req := kb13PerformanceRequest{
		PopulationID: populationID,
		MeasureID:    measureID,
	}

	resp, err := c.callKB13(ctx, "/api/v1/measures/performance", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get performance rate: %w", err)
	}

	var result kb13PerformanceResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse performance rate: %w", err)
	}

	return &PerformanceRate{
		MeasureID:          result.MeasureID,
		MeasureName:        result.MeasureName,
		PopulationID:       result.PopulationID,
		InitialPopulation:  result.InitialPopulation,
		Denominator:        result.Denominator,
		Numerator:          result.Numerator,
		Exclusions:         result.Exclusions,
		Exceptions:         result.Exceptions,
		PerformanceRate:    result.PerformanceRate,
		BenchmarkRate:      result.BenchmarkRate,
		BenchmarkPercentile: result.BenchmarkPercentile,
		TrendDirection:     result.TrendDirection,
		PriorPeriodRate:    result.PriorPeriodRate,
		CalculatedAt:       result.CalculatedAt,
	}, nil
}

// GetMeasureReport generates a formal measure report for a reporting period.
// Used for regulatory submission and quality reporting programs.
//
// Parameters:
// - measureID: Quality measure identifier
// - reportingPeriod: Date range for the report
//
// Returns:
// - MeasureReport formatted for QRDA III or CMS submission
func (c *KB13HTTPClient) GetMeasureReport(
	ctx context.Context,
	measureID string,
	reportingPeriod DateRange,
) (*MeasureReport, error) {

	req := kb13ReportRequest{
		MeasureID:       measureID,
		ReportingPeriod: reportingPeriod,
	}

	resp, err := c.callKB13(ctx, "/api/v1/measures/report", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get measure report: %w", err)
	}

	var result kb13ReportResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse measure report: %w", err)
	}

	stratifications := make([]Stratification, 0, len(result.Stratifications))
	for _, s := range result.Stratifications {
		stratifications = append(stratifications, Stratification{
			StratificationID: s.StratificationID,
			Description:      s.Description,
			Numerator:        s.Numerator,
			Denominator:      s.Denominator,
			Rate:             s.Rate,
		})
	}

	return &MeasureReport{
		ReportID:           result.ReportID,
		MeasureID:          result.MeasureID,
		MeasureName:        result.MeasureName,
		ReportingPeriod:    result.ReportingPeriod,
		InitialPopulation:  result.InitialPopulation,
		Denominator:        result.Denominator,
		Numerator:          result.Numerator,
		Exclusions:         result.Exclusions,
		Exceptions:         result.Exceptions,
		PerformanceRate:    result.PerformanceRate,
		Stratifications:    stratifications,
		ImprovementNotation: result.ImprovementNotation,
		GeneratedAt:        result.GeneratedAt,
		SubmissionReady:    result.SubmissionReady,
	}, nil
}

// BatchEvaluate evaluates multiple measures for a patient in a single call.
// More efficient than multiple individual calls.
//
// Parameters:
// - patientID: FHIR Patient.id for the patient
// - measureIDs: List of quality measure identifiers to evaluate
//
// Returns:
// - List of MeasureResults for all requested measures
func (c *KB13HTTPClient) BatchEvaluate(
	ctx context.Context,
	patientID string,
	measureIDs []string,
) ([]contracts.MeasureResult, error) {

	req := kb13BatchRequest{
		PatientID:   patientID,
		MeasureIDs:  measureIDs,
		EvaluatedAt: time.Now().UTC(),
	}

	resp, err := c.callKB13(ctx, "/api/v1/measures/batch-evaluate", req)
	if err != nil {
		return nil, fmt.Errorf("failed to batch evaluate measures: %w", err)
	}

	var result kb13BatchResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse batch results: %w", err)
	}

	results := make([]contracts.MeasureResult, 0, len(result.Results))
	for _, r := range result.Results {
		results = append(results, contracts.MeasureResult{
			MeasureID:              r.MeasureID,
			MeasureName:            r.MeasureName,
			InInitialPopulation:    r.InInitialPopulation,
			InDenominator:          r.InDenominator,
			InNumerator:            r.InNumerator,
			InDenominatorExclusion: r.InDenominatorExclusion,
			InDenominatorException: r.InDenominatorException,
			CareGapIdentified:      r.CareGapIdentified,
			MeasureVersion:         r.MeasureVersion,
			LogicVersion:           r.LogicVersion,
			EvaluatedAt:            r.EvaluatedAt,
			Rationale:              r.Rationale,
		})
	}

	return results, nil
}

// GetAvailableMeasures returns all measures available for evaluation.
// Used for configuration and measure set selection.
func (c *KB13HTTPClient) GetAvailableMeasures(
	ctx context.Context,
	category string,
) ([]MeasureSummary, error) {

	endpoint := "/api/v1/measures"
	if category != "" {
		endpoint = fmt.Sprintf("%s?category=%s", endpoint, category)
	}

	resp, err := c.callKB13Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get available measures: %w", err)
	}

	var result kb13MeasureListResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse measure list: %w", err)
	}

	measures := make([]MeasureSummary, 0, len(result.Measures))
	for _, m := range result.Measures {
		measures = append(measures, MeasureSummary{
			MeasureID:      m.MeasureID,
			MeasureName:    m.MeasureName,
			Category:       m.Category,
			MeasureType:    m.MeasureType,
			Version:        m.Version,
			IsActive:       m.IsActive,
			Description:    m.Description,
		})
	}

	return measures, nil
}

// GetMeasureHistory returns historical measure results for a patient.
// Used for trending and patient progress tracking.
func (c *KB13HTTPClient) GetMeasureHistory(
	ctx context.Context,
	patientID string,
	measureID string,
	startDate time.Time,
	endDate time.Time,
) ([]MeasureHistoryEntry, error) {

	req := kb13HistoryRequest{
		PatientID: patientID,
		MeasureID: measureID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	resp, err := c.callKB13(ctx, "/api/v1/measures/history", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get measure history: %w", err)
	}

	var result kb13HistoryResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse measure history: %w", err)
	}

	history := make([]MeasureHistoryEntry, 0, len(result.History))
	for _, h := range result.History {
		history = append(history, MeasureHistoryEntry{
			EvaluatedAt:     h.EvaluatedAt,
			InNumerator:     h.InNumerator,
			InDenominator:   h.InDenominator,
			CareGap:         h.CareGap,
			ChangeReason:    h.ChangeReason,
		})
	}

	return history, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB13HTTPClient) callKB13(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
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
		return nil, fmt.Errorf("KB-13 returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *KB13HTTPClient) callKB13Get(ctx context.Context, endpoint string) ([]byte, error) {
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
		return nil, fmt.Errorf("KB-13 returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ============================================================================
// KB-13 Specific Types
// ============================================================================

// MeasureDefinition contains the specification for a quality measure.
type MeasureDefinition struct {
	MeasureID           string   `json:"measureId"`
	MeasureName         string   `json:"measureName"`
	MeasureVersion      string   `json:"measureVersion"`
	Description         string   `json:"description"`
	Category            string   `json:"category"`
	MeasureType         string   `json:"measureType"`
	NQFNumber           string   `json:"nqfNumber,omitempty"`
	CMSNumber           string   `json:"cmsNumber,omitempty"`
	EligibilityCriteria string   `json:"eligibilityCriteria"`
	NumeratorCriteria   string   `json:"numeratorCriteria"`
	DenominatorCriteria string   `json:"denominatorCriteria"`
	ExclusionCriteria   string   `json:"exclusionCriteria,omitempty"`
	ExceptionCriteria   string   `json:"exceptionCriteria,omitempty"`
	RecommendedActions  []string `json:"recommendedActions,omitempty"`
	EvidenceGrade       string   `json:"evidenceGrade,omitempty"`
	SourceURL           string   `json:"sourceUrl,omitempty"`
}

// PerformanceRate contains aggregate measure performance.
type PerformanceRate struct {
	MeasureID           string    `json:"measureId"`
	MeasureName         string    `json:"measureName"`
	PopulationID        string    `json:"populationId"`
	InitialPopulation   int       `json:"initialPopulation"`
	Denominator         int       `json:"denominator"`
	Numerator           int       `json:"numerator"`
	Exclusions          int       `json:"exclusions"`
	Exceptions          int       `json:"exceptions"`
	PerformanceRate     float64   `json:"performanceRate"`
	BenchmarkRate       float64   `json:"benchmarkRate,omitempty"`
	BenchmarkPercentile int       `json:"benchmarkPercentile,omitempty"`
	TrendDirection      string    `json:"trendDirection,omitempty"`
	PriorPeriodRate     float64   `json:"priorPeriodRate,omitempty"`
	CalculatedAt        time.Time `json:"calculatedAt"`
}

// DateRange represents a time period for reporting.
type DateRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

// MeasureReport is a formal quality measure report.
type MeasureReport struct {
	ReportID            string           `json:"reportId"`
	MeasureID           string           `json:"measureId"`
	MeasureName         string           `json:"measureName"`
	ReportingPeriod     DateRange        `json:"reportingPeriod"`
	InitialPopulation   int              `json:"initialPopulation"`
	Denominator         int              `json:"denominator"`
	Numerator           int              `json:"numerator"`
	Exclusions          int              `json:"exclusions"`
	Exceptions          int              `json:"exceptions"`
	PerformanceRate     float64          `json:"performanceRate"`
	Stratifications     []Stratification `json:"stratifications,omitempty"`
	ImprovementNotation string           `json:"improvementNotation"`
	GeneratedAt         time.Time        `json:"generatedAt"`
	SubmissionReady     bool             `json:"submissionReady"`
}

// Stratification represents a measure stratification result.
type Stratification struct {
	StratificationID string  `json:"stratificationId"`
	Description      string  `json:"description"`
	Numerator        int     `json:"numerator"`
	Denominator      int     `json:"denominator"`
	Rate             float64 `json:"rate"`
}

// MeasureSummary is a brief summary of a quality measure.
type MeasureSummary struct {
	MeasureID   string `json:"measureId"`
	MeasureName string `json:"measureName"`
	Category    string `json:"category"`
	MeasureType string `json:"measureType"`
	Version     string `json:"version"`
	IsActive    bool   `json:"isActive"`
	Description string `json:"description,omitempty"`
}

// MeasureHistoryEntry represents a historical measure evaluation.
type MeasureHistoryEntry struct {
	EvaluatedAt   time.Time `json:"evaluatedAt"`
	InNumerator   bool      `json:"inNumerator"`
	InDenominator bool      `json:"inDenominator"`
	CareGap       bool      `json:"careGap"`
	ChangeReason  string    `json:"changeReason,omitempty"`
}

// ============================================================================
// Internal Request/Response Types
// ============================================================================

type kb13EvaluateRequest struct {
	PatientID   string    `json:"patientId"`
	MeasureID   string    `json:"measureId"`
	EvaluatedAt time.Time `json:"evaluatedAt"`
}

type kb13EvaluateResponse struct {
	MeasureID              string    `json:"measureId"`
	MeasureName            string    `json:"measureName"`
	InInitialPopulation    bool      `json:"inInitialPopulation"`
	InDenominator          bool      `json:"inDenominator"`
	InNumerator            bool      `json:"inNumerator"`
	InDenominatorExclusion bool      `json:"inDenominatorExclusion"`
	InDenominatorException bool      `json:"inDenominatorException"`
	CareGapIdentified      bool      `json:"careGapIdentified"`
	MeasureVersion         string    `json:"measureVersion"`
	LogicVersion           string    `json:"logicVersion"`
	ELMCorrespondence      string    `json:"elmCorrespondence"`
	EvaluatedAt            time.Time `json:"evaluatedAt"`
	Rationale              string    `json:"rationale"`
}

type kb13MeasureDefResponse struct {
	MeasureID           string   `json:"measureId"`
	MeasureName         string   `json:"measureName"`
	MeasureVersion      string   `json:"measureVersion"`
	Description         string   `json:"description"`
	Category            string   `json:"category"`
	MeasureType         string   `json:"measureType"`
	NQFNumber           string   `json:"nqfNumber"`
	CMSNumber           string   `json:"cmsNumber"`
	EligibilityCriteria string   `json:"eligibilityCriteria"`
	NumeratorCriteria   string   `json:"numeratorCriteria"`
	DenominatorCriteria string   `json:"denominatorCriteria"`
	ExclusionCriteria   string   `json:"exclusionCriteria"`
	ExceptionCriteria   string   `json:"exceptionCriteria"`
	RecommendedActions  []string `json:"recommendedActions"`
	EvidenceGrade       string   `json:"evidenceGrade"`
	SourceURL           string   `json:"sourceUrl"`
}

type kb13PerformanceRequest struct {
	PopulationID string `json:"populationId"`
	MeasureID    string `json:"measureId"`
}

type kb13PerformanceResponse struct {
	MeasureID           string    `json:"measureId"`
	MeasureName         string    `json:"measureName"`
	PopulationID        string    `json:"populationId"`
	InitialPopulation   int       `json:"initialPopulation"`
	Denominator         int       `json:"denominator"`
	Numerator           int       `json:"numerator"`
	Exclusions          int       `json:"exclusions"`
	Exceptions          int       `json:"exceptions"`
	PerformanceRate     float64   `json:"performanceRate"`
	BenchmarkRate       float64   `json:"benchmarkRate"`
	BenchmarkPercentile int       `json:"benchmarkPercentile"`
	TrendDirection      string    `json:"trendDirection"`
	PriorPeriodRate     float64   `json:"priorPeriodRate"`
	CalculatedAt        time.Time `json:"calculatedAt"`
}

type kb13ReportRequest struct {
	MeasureID       string    `json:"measureId"`
	ReportingPeriod DateRange `json:"reportingPeriod"`
}

type kb13ReportResponse struct {
	ReportID        string    `json:"reportId"`
	MeasureID       string    `json:"measureId"`
	MeasureName     string    `json:"measureName"`
	ReportingPeriod DateRange `json:"reportingPeriod"`
	InitialPopulation   int   `json:"initialPopulation"`
	Denominator     int       `json:"denominator"`
	Numerator       int       `json:"numerator"`
	Exclusions      int       `json:"exclusions"`
	Exceptions      int       `json:"exceptions"`
	PerformanceRate float64   `json:"performanceRate"`
	Stratifications []struct {
		StratificationID string  `json:"stratificationId"`
		Description      string  `json:"description"`
		Numerator        int     `json:"numerator"`
		Denominator      int     `json:"denominator"`
		Rate             float64 `json:"rate"`
	} `json:"stratifications"`
	ImprovementNotation string    `json:"improvementNotation"`
	GeneratedAt         time.Time `json:"generatedAt"`
	SubmissionReady     bool      `json:"submissionReady"`
}

type kb13BatchRequest struct {
	PatientID   string    `json:"patientId"`
	MeasureIDs  []string  `json:"measureIds"`
	EvaluatedAt time.Time `json:"evaluatedAt"`
}

type kb13BatchResponse struct {
	Results []kb13EvaluateResponse `json:"results"`
}

type kb13MeasureListResponse struct {
	Measures []struct {
		MeasureID   string `json:"measureId"`
		MeasureName string `json:"measureName"`
		Category    string `json:"category"`
		MeasureType string `json:"measureType"`
		Version     string `json:"version"`
		IsActive    bool   `json:"isActive"`
		Description string `json:"description"`
	} `json:"measures"`
}

type kb13HistoryRequest struct {
	PatientID string    `json:"patientId"`
	MeasureID string    `json:"measureId"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type kb13HistoryResponse struct {
	History []struct {
		EvaluatedAt   time.Time `json:"evaluatedAt"`
		InNumerator   bool      `json:"inNumerator"`
		InDenominator bool      `json:"inDenominator"`
		CareGap       bool      `json:"careGap"`
		ChangeReason  string    `json:"changeReason"`
	} `json:"history"`
}

// HealthCheck verifies KB-13 service is healthy.
func (c *KB13HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-13 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}
