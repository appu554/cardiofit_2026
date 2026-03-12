// Package clients provides HTTP clients for KB services.
//
// KB11HTTPClient implements the KB11Client interface for KB-11 Clinical Intelligence Orchestrator.
// It provides clinical documentation intelligence including:
//   - Clinical fact extraction from notes
//   - Coding opportunity identification
//   - Documentation query generation
//   - Risk score aggregation
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-11 is a SNAPSHOT category KB. All CDI facts are pre-computed at snapshot build time.
// Engines NEVER call KB-11 directly at execution time - they read from CDIFacts in the snapshot.
//
// Per the CTO/CMO spec:
//   "CQL explains. KB-19 recommends. ICU decides."
//
// KB-11 provides the clinical intelligence that informs CQL execution,
// but it does NOT make recommendations or decisions.
//
// Connects to: http://localhost:8111 (Docker: kb11-clinical-intelligence)
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

// KB11HTTPClient implements KB11Client by calling the KB-11 Clinical Intelligence REST API.
type KB11HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB11HTTPClient creates a new KB-11 HTTP client.
func NewKB11HTTPClient(baseURL string) *KB11HTTPClient {
	return &KB11HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB11HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB11HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB11HTTPClient {
	return &KB11HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB11Client Interface Implementation
// ============================================================================

// GetCDIFacts retrieves clinical documentation intelligence facts for snapshot building.
// This is called by KnowledgeSnapshotBuilder to populate CDI section of the snapshot.
func (c *KB11HTTPClient) GetCDIFacts(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CDIFacts, error) {

	// Build request with patient context
	req := kb11CDIRequest{
		PatientID:     patient.Demographics.PatientID,
		Region:        patient.Demographics.Region,
		ConditionIDs:  extractConditionCodes(patient.ActiveConditions),
		MedicationIDs: extractMedicationCodes(patient.ActiveMedications),
	}

	resp, err := c.callKB11(ctx, "/api/v1/cdi/facts", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get CDI facts: %w", err)
	}

	var result kb11CDIResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse CDI response: %w", err)
	}

	return c.mapToCDIFacts(&result), nil
}

// ExtractClinicalFacts extracts clinical facts from unstructured clinical notes.
// Returns facts with confidence scores for NLP-extracted information.
func (c *KB11HTTPClient) ExtractClinicalFacts(
	ctx context.Context,
	patientID string,
	clinicalNotes []string,
) ([]contracts.CDIFact, error) {

	req := kb11ExtractionRequest{
		PatientID:     patientID,
		ClinicalNotes: clinicalNotes,
	}

	resp, err := c.callKB11(ctx, "/api/v1/cdi/extract", req)
	if err != nil {
		return nil, fmt.Errorf("failed to extract clinical facts: %w", err)
	}

	var result kb11ExtractionResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse extraction response: %w", err)
	}

	facts := make([]contracts.CDIFact, 0, len(result.ExtractedFacts))
	for _, f := range result.ExtractedFacts {
		facts = append(facts, contracts.CDIFact{
			FactType:    f.FactType,
			Code:        mapToCode(f.CodeSystem, f.CodeValue, f.CodeDisplay),
			Description: f.Description,
			Confidence:  f.Confidence,
			SourceText:  f.SourceText,
		})
	}

	return facts, nil
}

// IdentifyCodingOpportunities analyzes patient data for potential coding improvements.
// Returns opportunities for better ICD-10 specificity or DRG optimization.
func (c *KB11HTTPClient) IdentifyCodingOpportunities(
	ctx context.Context,
	patient *contracts.PatientContext,
) ([]contracts.CodingOpportunity, error) {

	req := kb11CodingRequest{
		PatientID:    patient.Demographics.PatientID,
		Conditions:   extractConditionCodes(patient.ActiveConditions),
		Medications:  extractMedicationCodes(patient.ActiveMedications),
		LabResults:   extractLabCodes(patient.RecentLabResults),
	}

	resp, err := c.callKB11(ctx, "/api/v1/cdi/coding-opportunities", req)
	if err != nil {
		return nil, fmt.Errorf("failed to identify coding opportunities: %w", err)
	}

	var result kb11CodingResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse coding response: %w", err)
	}

	opportunities := make([]contracts.CodingOpportunity, 0, len(result.Opportunities))
	for _, o := range result.Opportunities {
		opportunities = append(opportunities, contracts.CodingOpportunity{
			CurrentCode:    mapToCode(o.CurrentCodeSystem, o.CurrentCodeValue, o.CurrentCodeDisplay),
			SuggestedCode:  mapToCode(o.SuggestedCodeSystem, o.SuggestedCodeValue, o.SuggestedCodeDisplay),
			Reason:         o.Reason,
			ImpactEstimate: o.ImpactEstimate,
		})
	}

	return opportunities, nil
}

// GenerateDocumentationQueries generates documentation improvement queries.
// Returns questions that can be sent to clinical staff for clarification.
func (c *KB11HTTPClient) GenerateDocumentationQueries(
	ctx context.Context,
	patient *contracts.PatientContext,
) ([]contracts.QueryOpportunity, error) {

	req := kb11QueryRequest{
		PatientID:   patient.Demographics.PatientID,
		Conditions:  extractConditionCodes(patient.ActiveConditions),
		LabResults:  extractLabCodes(patient.RecentLabResults),
		VitalSigns:  extractVitalSignCodes(patient.RecentVitalSigns),
	}

	resp, err := c.callKB11(ctx, "/api/v1/cdi/documentation-queries", req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation queries: %w", err)
	}

	var result kb11QueryResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse query response: %w", err)
	}

	queries := make([]contracts.QueryOpportunity, 0, len(result.Queries))
	for _, q := range result.Queries {
		queries = append(queries, contracts.QueryOpportunity{
			QueryType: q.QueryType,
			Question:  q.Question,
			Context:   q.Context,
			Priority:  q.Priority,
		})
	}

	return queries, nil
}

// GetRiskScores retrieves aggregated risk scores for the patient.
// This aggregates scores from KB-8 and other sources for clinical intelligence.
func (c *KB11HTTPClient) GetRiskScores(
	ctx context.Context,
	patient *contracts.PatientContext,
) (map[string]contracts.RiskScore, error) {

	req := kb11RiskRequest{
		PatientID: patient.Demographics.PatientID,
		Region:    patient.Demographics.Region,
	}

	resp, err := c.callKB11(ctx, "/api/v1/cdi/risk-scores", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk scores: %w", err)
	}

	var result kb11RiskResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse risk response: %w", err)
	}

	scores := make(map[string]contracts.RiskScore)
	for _, s := range result.Scores {
		scores[s.Name] = contracts.RiskScore{
			Name:            s.Name,
			Value:           s.Value,
			Category:        s.Category,
			Confidence:      s.Confidence,
			ComponentScores: s.Components,
		}
	}

	return scores, nil
}

// GetClinicalFlags retrieves boolean clinical flags for the patient.
// These flags indicate conditions like "is_diabetic", "is_sepsis_risk", etc.
func (c *KB11HTTPClient) GetClinicalFlags(
	ctx context.Context,
	patient *contracts.PatientContext,
) (map[string]bool, error) {

	req := kb11FlagsRequest{
		PatientID:  patient.Demographics.PatientID,
		Conditions: extractConditionCodes(patient.ActiveConditions),
	}

	resp, err := c.callKB11(ctx, "/api/v1/cdi/clinical-flags", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get clinical flags: %w", err)
	}

	var result kb11FlagsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse flags response: %w", err)
	}

	return result.Flags, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB11HTTPClient) callKB11(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-11 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-11 service is healthy.
func (c *KB11HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-11 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// Mapping Functions
// ============================================================================

func (c *KB11HTTPClient) mapToCDIFacts(result *kb11CDIResult) *contracts.CDIFacts {
	cdi := &contracts.CDIFacts{
		ExtractedFacts:      make([]contracts.CDIFact, 0, len(result.ExtractedFacts)),
		CodingOpportunities: make([]contracts.CodingOpportunity, 0, len(result.CodingOpportunities)),
		QueryOpportunities:  make([]contracts.QueryOpportunity, 0, len(result.QueryOpportunities)),
	}

	for _, f := range result.ExtractedFacts {
		cdi.ExtractedFacts = append(cdi.ExtractedFacts, contracts.CDIFact{
			FactType:    f.FactType,
			Code:        mapToCode(f.CodeSystem, f.CodeValue, f.CodeDisplay),
			Description: f.Description,
			Confidence:  f.Confidence,
			SourceText:  f.SourceText,
		})
	}

	for _, o := range result.CodingOpportunities {
		cdi.CodingOpportunities = append(cdi.CodingOpportunities, contracts.CodingOpportunity{
			CurrentCode:    mapToCode(o.CurrentCodeSystem, o.CurrentCodeValue, o.CurrentCodeDisplay),
			SuggestedCode:  mapToCode(o.SuggestedCodeSystem, o.SuggestedCodeValue, o.SuggestedCodeDisplay),
			Reason:         o.Reason,
			ImpactEstimate: o.ImpactEstimate,
		})
	}

	for _, q := range result.QueryOpportunities {
		cdi.QueryOpportunities = append(cdi.QueryOpportunities, contracts.QueryOpportunity{
			QueryType: q.QueryType,
			Question:  q.Question,
			Context:   q.Context,
			Priority:  q.Priority,
		})
	}

	return cdi
}

func mapToCode(system, code, display string) contracts.ClinicalCode {
	return contracts.ClinicalCode{
		System:  system,
		Code:    code,
		Display: display,
	}
}

func extractConditionCodes(conditions []contracts.ClinicalCondition) []string {
	codes := make([]string, 0, len(conditions))
	for _, c := range conditions {
		codes = append(codes, c.Code.Code)
	}
	return codes
}

func extractMedicationCodes(medications []contracts.Medication) []string {
	codes := make([]string, 0, len(medications))
	for _, m := range medications {
		codes = append(codes, m.Code.Code)
	}
	return codes
}

func extractLabCodes(labs []contracts.LabResult) []string {
	codes := make([]string, 0, len(labs))
	for _, l := range labs {
		codes = append(codes, l.Code.Code)
	}
	return codes
}

func extractVitalSignCodes(vitals []contracts.VitalSign) []string {
	codes := make([]string, 0, len(vitals))
	for _, v := range vitals {
		codes = append(codes, v.Code.Code)
	}
	return codes
}

// ============================================================================
// KB-11 Request/Response Types (internal)
// ============================================================================

type kb11CDIRequest struct {
	PatientID     string   `json:"patient_id"`
	Region        string   `json:"region,omitempty"`
	ConditionIDs  []string `json:"condition_ids,omitempty"`
	MedicationIDs []string `json:"medication_ids,omitempty"`
}

type kb11CDIResult struct {
	ExtractedFacts      []kb11Fact              `json:"extracted_facts"`
	CodingOpportunities []kb11CodingOpportunity `json:"coding_opportunities"`
	QueryOpportunities  []kb11QueryOpportunity  `json:"query_opportunities"`
}

type kb11ExtractionRequest struct {
	PatientID     string   `json:"patient_id"`
	ClinicalNotes []string `json:"clinical_notes"`
}

type kb11ExtractionResult struct {
	ExtractedFacts []kb11Fact `json:"extracted_facts"`
}

type kb11Fact struct {
	FactType    string  `json:"fact_type"`    // diagnosis, procedure, symptom, finding
	CodeSystem  string  `json:"code_system"`  // http://snomed.info/sct, http://hl7.org/fhir/sid/icd-10
	CodeValue   string  `json:"code_value"`
	CodeDisplay string  `json:"code_display"`
	Description string  `json:"description"`
	Confidence  float64 `json:"confidence"`  // 0.0-1.0
	SourceText  string  `json:"source_text"` // Original text that was extracted from
}

type kb11CodingRequest struct {
	PatientID   string   `json:"patient_id"`
	Conditions  []string `json:"conditions"`
	Medications []string `json:"medications"`
	LabResults  []string `json:"lab_results"`
}

type kb11CodingResult struct {
	Opportunities []kb11CodingOpportunity `json:"opportunities"`
}

type kb11CodingOpportunity struct {
	CurrentCodeSystem   string `json:"current_code_system,omitempty"`
	CurrentCodeValue    string `json:"current_code_value,omitempty"`
	CurrentCodeDisplay  string `json:"current_code_display,omitempty"`
	SuggestedCodeSystem string `json:"suggested_code_system"`
	SuggestedCodeValue  string `json:"suggested_code_value"`
	SuggestedCodeDisplay string `json:"suggested_code_display"`
	Reason              string `json:"reason"`
	ImpactEstimate      string `json:"impact_estimate,omitempty"` // DRG impact, compliance, etc.
}

type kb11QueryRequest struct {
	PatientID  string   `json:"patient_id"`
	Conditions []string `json:"conditions"`
	LabResults []string `json:"lab_results"`
	VitalSigns []string `json:"vital_signs"`
}

type kb11QueryResult struct {
	Queries []kb11QueryOpportunity `json:"queries"`
}

type kb11QueryOpportunity struct {
	QueryType string `json:"query_type"` // specificity, clarification, missing_doc
	Question  string `json:"question"`
	Context   string `json:"context,omitempty"`
	Priority  string `json:"priority"` // high, medium, low
}

type kb11RiskRequest struct {
	PatientID string `json:"patient_id"`
	Region    string `json:"region,omitempty"`
}

type kb11RiskResult struct {
	Scores []kb11RiskScore `json:"scores"`
}

type kb11RiskScore struct {
	Name       string             `json:"name"`
	Value      float64            `json:"value"`
	Category   string             `json:"category,omitempty"`
	Confidence float64            `json:"confidence,omitempty"`
	Components map[string]float64 `json:"components,omitempty"`
}

type kb11FlagsRequest struct {
	PatientID  string   `json:"patient_id"`
	Conditions []string `json:"conditions"`
}

type kb11FlagsResult struct {
	Flags map[string]bool `json:"flags"`
}
