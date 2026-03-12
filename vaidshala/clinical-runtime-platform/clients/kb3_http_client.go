// Package clients provides HTTP clients for KB services.
//
// KB3HTTPClient implements the KB3Client interface for KB-3 Clinical Guidelines Service.
// It provides access to clinical protocols, treatment pathways, and guideline recommendations.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// - KB-3 provides PROTOCOLS and PATHWAYS, not direct drug recommendations
// - Drug recommendations come from KB-1 (Drug Rules) and KB-6 (Formulary)
// - CQL CLASSIFIES guideline applicability; KB-3 provides pathway context
// - KB-19 coordinates guideline recommendations with ICU Dominance respect
//
// Key Endpoints:
//   - /v1/protocols/condition/{condition} - Get protocols for a clinical condition
//   - /v1/protocols/search?q={query} - Search protocols by keyword
//   - /v1/protocols/{type}/{id} - Get specific protocol details
//   - /v1/guidelines/applicable - Get applicable guidelines for patient context
//   - /v1/guidelines/{id}/compliance - Check compliance with specific guideline
//
// Connects to: http://localhost:8087 (Docker: kb3-guidelines-service)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// ============================================================================
// KB-3 GUIDELINE TYPES
// ============================================================================

// Guideline represents a clinical practice guideline.
type Guideline struct {
	// GuidelineID is the unique identifier
	GuidelineID string `json:"guideline_id"`

	// Name is the guideline title
	Name string `json:"name"`

	// Source organization (e.g., "AHA", "ACC", "SSC", "KDIGO")
	Source string `json:"source"`

	// Version of the guideline (e.g., "2024", "2021")
	Version string `json:"version"`

	// PublicationDate when guideline was published
	PublicationDate *time.Time `json:"publication_date,omitempty"`

	// Conditions the guideline applies to (SNOMED codes)
	Conditions []contracts.ClinicalCode `json:"conditions,omitempty"`

	// Summary is a brief description
	Summary string `json:"summary,omitempty"`

	// EvidenceLevel overall evidence grade
	EvidenceLevel string `json:"evidence_level,omitempty"`

	// Recommendations count
	RecommendationCount int `json:"recommendation_count,omitempty"`

	// URL to original guideline
	URL string `json:"url,omitempty"`
}

// Protocol represents a clinical treatment protocol.
type Protocol struct {
	// ProtocolID is the unique identifier
	ProtocolID string `json:"protocol_id"`

	// Name is the protocol title
	Name string `json:"name"`

	// Type is the protocol type (acute, chronic, screening, monitoring)
	Type string `json:"type"`

	// GuidelineSource is the source guideline
	GuidelineSource string `json:"guideline_source"`

	// Description of the protocol
	Description string `json:"description,omitempty"`

	// Stages for acute protocols
	Stages []ProtocolStage `json:"stages,omitempty"`

	// MonitoringItems for chronic/monitoring protocols
	MonitoringItems []MonitoringItem `json:"monitoring_items,omitempty"`

	// Triggers that activate this protocol
	Triggers []ProtocolTrigger `json:"triggers,omitempty"`

	// Contraindications to protocol activation
	Contraindications []string `json:"contraindications,omitempty"`

	// Priority ordering when multiple protocols apply
	Priority int `json:"priority,omitempty"`
}

// ProtocolStage represents a stage in an acute protocol.
type ProtocolStage struct {
	// StageID is the unique identifier
	StageID string `json:"stage_id"`

	// Name of the stage
	Name string `json:"name"`

	// Order within the protocol
	Order int `json:"order"`

	// Duration expected for this stage
	Duration string `json:"duration,omitempty"`

	// Actions to perform in this stage
	Actions []ProtocolAction `json:"actions,omitempty"`

	// Completion criteria for this stage
	CompletionCriteria []string `json:"completion_criteria,omitempty"`

	// EscalationCriteria to next stage
	EscalationCriteria []string `json:"escalation_criteria,omitempty"`
}

// ProtocolAction represents an action within a protocol stage.
type ProtocolAction struct {
	// ActionID is the unique identifier
	ActionID string `json:"action_id"`

	// Type of action (medication, lab, vital, procedure, assessment)
	Type string `json:"type"`

	// Medication name if type is medication
	Medication string `json:"medication,omitempty"`

	// DrugClass if type is medication
	DrugClass string `json:"drug_class,omitempty"`

	// RxNormCode if type is medication
	RxNormCode string `json:"rxnorm_code,omitempty"`

	// LabCode if type is lab (LOINC)
	LabCode string `json:"lab_code,omitempty"`

	// Description of the action
	Description string `json:"description,omitempty"`

	// Dose if type is medication
	Dose float64 `json:"dose,omitempty"`

	// DoseUnit if type is medication
	DoseUnit string `json:"dose_unit,omitempty"`

	// Route if type is medication
	Route string `json:"route,omitempty"`

	// Frequency if recurring
	Frequency string `json:"frequency,omitempty"`

	// IsCritical marks time-sensitive actions
	IsCritical bool `json:"is_critical,omitempty"`

	// TimeWindow for critical actions (e.g., "1 hour")
	TimeWindow string `json:"time_window,omitempty"`

	// EvidenceGrade for this action
	EvidenceGrade string `json:"evidence_grade,omitempty"`
}

// MonitoringItem represents an item in a chronic monitoring schedule.
type MonitoringItem struct {
	// ItemID is the unique identifier
	ItemID string `json:"item_id"`

	// Name of the monitoring item
	Name string `json:"name"`

	// Type of monitoring (lab, vital, screening, assessment, medication)
	Type string `json:"type"`

	// Code (LOINC for labs, SNOMED for procedures)
	Code string `json:"code,omitempty"`

	// Recurrence schedule
	Recurrence MonitoringRecurrence `json:"recurrence"`

	// Priority of this monitoring item
	Priority string `json:"priority,omitempty"`

	// Rationale for monitoring
	Rationale string `json:"rationale,omitempty"`
}

// MonitoringRecurrence defines the schedule for a monitoring item.
type MonitoringRecurrence struct {
	// Frequency (daily, weekly, monthly, quarterly, annually)
	Frequency string `json:"frequency"`

	// Interval (e.g., 1 for every month, 3 for every 3 months)
	Interval int `json:"interval"`

	// DaysFromBaseline for one-time follow-ups
	DaysFromBaseline int `json:"days_from_baseline,omitempty"`
}

// ProtocolTrigger defines when a protocol should be activated.
type ProtocolTrigger struct {
	// TriggerID is the unique identifier
	TriggerID string `json:"trigger_id"`

	// Type of trigger (condition, lab_result, vital_sign, medication)
	Type string `json:"type"`

	// Code for the trigger (SNOMED, LOINC, RxNorm)
	Code string `json:"code,omitempty"`

	// Threshold for numeric triggers
	Threshold *TriggerThreshold `json:"threshold,omitempty"`

	// Description of the trigger
	Description string `json:"description,omitempty"`
}

// TriggerThreshold defines threshold conditions for triggers.
type TriggerThreshold struct {
	// Operator (gt, lt, gte, lte, eq, ne, between)
	Operator string `json:"operator"`

	// Value for comparison
	Value float64 `json:"value"`

	// UpperValue for "between" operator
	UpperValue float64 `json:"upper_value,omitempty"`

	// Unit of measurement
	Unit string `json:"unit,omitempty"`
}

// GuidelineRecommendation represents a specific recommendation from a guideline.
type GuidelineRecommendation struct {
	// RecommendationID is the unique identifier
	RecommendationID string `json:"recommendation_id"`

	// GuidelineID parent guideline
	GuidelineID string `json:"guideline_id"`

	// Text of the recommendation
	Text string `json:"text"`

	// ClassOfRecommendation (I, IIa, IIb, III)
	ClassOfRecommendation string `json:"class_of_recommendation"`

	// LevelOfEvidence (A, B-R, B-NR, C-LD, C-EO)
	LevelOfEvidence string `json:"level_of_evidence"`

	// Category (treatment, prevention, screening, monitoring)
	Category string `json:"category,omitempty"`

	// Medications mentioned in this recommendation
	Medications []string `json:"medications,omitempty"`

	// Conditions this applies to
	Conditions []string `json:"conditions,omitempty"`

	// IsFirstLine if this is a first-line recommendation
	IsFirstLine bool `json:"is_first_line,omitempty"`

	// ContraindicatedWith conditions that contraindicate this
	ContraindicatedWith []string `json:"contraindicated_with,omitempty"`
}

// GuidelineCompliance represents compliance status with a guideline.
type GuidelineCompliance struct {
	// GuidelineID the compliance relates to
	GuidelineID string `json:"guideline_id"`

	// GuidelineName for display
	GuidelineName string `json:"guideline_name"`

	// OverallCompliance percentage (0-100)
	OverallCompliance float64 `json:"overall_compliance"`

	// ComplianceStatus (compliant, partial, non_compliant)
	ComplianceStatus string `json:"compliance_status"`

	// RecommendationCompliance per-recommendation status
	RecommendationCompliance []RecommendationComplianceItem `json:"recommendation_compliance,omitempty"`

	// GapsIdentified specific gaps in compliance
	GapsIdentified []ComplianceGap `json:"gaps_identified,omitempty"`

	// AssessedAt when compliance was assessed
	AssessedAt time.Time `json:"assessed_at"`
}

// RecommendationComplianceItem represents compliance with a single recommendation.
type RecommendationComplianceItem struct {
	// RecommendationID of the recommendation
	RecommendationID string `json:"recommendation_id"`

	// RecommendationText for display
	RecommendationText string `json:"recommendation_text"`

	// IsCompliant whether currently compliant
	IsCompliant bool `json:"is_compliant"`

	// Reason for non-compliance if applicable
	Reason string `json:"reason,omitempty"`

	// Evidence supporting compliance assessment
	Evidence []string `json:"evidence,omitempty"`
}

// ComplianceGap represents a specific gap in guideline compliance.
type ComplianceGap struct {
	// GapID is the unique identifier
	GapID string `json:"gap_id"`

	// Description of the gap
	Description string `json:"description"`

	// RecommendationID the gap relates to
	RecommendationID string `json:"recommendation_id,omitempty"`

	// Severity of the gap (critical, high, medium, low)
	Severity string `json:"severity"`

	// RecommendedAction to close the gap
	RecommendedAction string `json:"recommended_action"`

	// Priority for addressing
	Priority int `json:"priority"`
}

// DrugRecommendation represents a drug recommendation from guidelines.
type DrugRecommendation struct {
	// DrugName generic drug name
	DrugName string `json:"drug_name"`

	// RxNormCode if available
	RxNormCode string `json:"rxnorm_code,omitempty"`

	// DrugClass therapeutic class
	DrugClass string `json:"drug_class,omitempty"`

	// GuidelineSource recommending guideline
	GuidelineSource string `json:"guideline_source"`

	// RecommendationID from guideline
	RecommendationID string `json:"recommendation_id,omitempty"`

	// EvidenceGrade for this recommendation
	EvidenceGrade string `json:"evidence_grade"`

	// ClassOfRecommendation (I, IIa, IIb, III)
	ClassOfRecommendation string `json:"class_of_recommendation,omitempty"`

	// IsFirstLine if this is a first-line choice
	IsFirstLine bool `json:"is_first_line"`

	// Indication for the drug
	Indication string `json:"indication,omitempty"`

	// Rationale for recommendation
	Rationale string `json:"rationale,omitempty"`

	// TypicalDose suggested dose range
	TypicalDose string `json:"typical_dose,omitempty"`

	// Contraindications to this drug
	Contraindications []string `json:"contraindications,omitempty"`
}

// ============================================================================
// KB-3 HTTP CLIENT
// ============================================================================

// KB3HTTPClient implements KB3Client by calling the KB-3 Clinical Guidelines Service REST API.
type KB3HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB3HTTPClient creates a new KB-3 HTTP client.
func NewKB3HTTPClient(baseURL string) *KB3HTTPClient {
	return &KB3HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB3HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB3HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB3HTTPClient {
	return &KB3HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// GUIDELINE RETRIEVAL METHODS
// ============================================================================

// GetApplicableGuidelines returns guidelines applicable to patient conditions.
// Calls KB-3 /v1/guidelines/applicable endpoint.
func (c *KB3HTTPClient) GetApplicableGuidelines(
	ctx context.Context,
	patient *contracts.PatientContext,
) ([]Guideline, error) {

	// Extract condition codes
	conditionCodes := make([]string, 0, len(patient.ActiveConditions))
	for _, cond := range patient.ActiveConditions {
		conditionCodes = append(conditionCodes, cond.Code.Code)
	}

	req := kb3ApplicableGuidelinesRequest{
		PatientID:      patient.Demographics.PatientID,
		ConditionCodes: conditionCodes,
		Region:         patient.Demographics.Region,
	}

	resp, err := c.callKB3(ctx, "/v1/guidelines/applicable", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get applicable guidelines: %w", err)
	}

	var result kb3GuidelinesResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse guidelines response: %w", err)
	}

	return result.Guidelines, nil
}

// GetGuidelineDetail returns detailed information about a specific guideline.
// Calls KB-3 /v1/guidelines/{id} endpoint.
func (c *KB3HTTPClient) GetGuidelineDetail(
	ctx context.Context,
	guidelineID string,
) (*Guideline, []GuidelineRecommendation, error) {

	reqURL := fmt.Sprintf("%s/v1/guidelines/%s", c.baseURL, url.PathEscape(guidelineID))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result kb3GuidelineDetailResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, nil, fmt.Errorf("failed to parse guideline detail: %w", err)
	}

	return &result.Guideline, result.Recommendations, nil
}

// GetGuidelinesByCondition returns guidelines for a specific condition.
// Calls KB-3 /v1/guidelines/condition/{code} endpoint.
func (c *KB3HTTPClient) GetGuidelinesByCondition(
	ctx context.Context,
	conditionCode string,
) ([]Guideline, error) {

	reqURL := fmt.Sprintf("%s/v1/guidelines/condition/%s", c.baseURL, url.PathEscape(conditionCode))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result kb3GuidelinesResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse guidelines: %w", err)
	}

	return result.Guidelines, nil
}

// ============================================================================
// PROTOCOL METHODS
// ============================================================================

// GetProtocolsForCondition returns treatment protocols for a condition.
// Calls KB-3 /v1/protocols/condition/{condition} endpoint.
func (c *KB3HTTPClient) GetProtocolsForCondition(
	ctx context.Context,
	conditionCode string,
) ([]Protocol, error) {

	// Normalize condition name for URL
	normalizedCondition := strings.ToLower(strings.ReplaceAll(conditionCode, " ", "-"))
	reqURL := fmt.Sprintf("%s/v1/protocols/condition/%s", c.baseURL, url.PathEscape(normalizedCondition))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode == http.StatusNotFound {
		// Condition not found - return empty list (not an error)
		return []Protocol{}, nil
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result kb3ProtocolsResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse protocols: %w", err)
	}

	return result.Protocols, nil
}

// GetProtocolDetail returns detailed information about a specific protocol.
// Calls KB-3 /v1/protocols/{type}/{id} endpoint.
func (c *KB3HTTPClient) GetProtocolDetail(
	ctx context.Context,
	protocolType string,
	protocolID string,
) (*Protocol, error) {

	reqURL := fmt.Sprintf("%s/v1/protocols/%s/%s", c.baseURL, url.PathEscape(protocolType), url.PathEscape(protocolID))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var protocol Protocol
	if err := json.Unmarshal(body, &protocol); err != nil {
		return nil, fmt.Errorf("failed to parse protocol detail: %w", err)
	}

	return &protocol, nil
}

// SearchProtocols searches for protocols by keyword.
// Calls KB-3 /v1/protocols/search endpoint.
func (c *KB3HTTPClient) SearchProtocols(
	ctx context.Context,
	query string,
	protocolType string,
) ([]Protocol, error) {

	reqURL := fmt.Sprintf("%s/v1/protocols/search?q=%s", c.baseURL, url.QueryEscape(query))
	if protocolType != "" {
		reqURL += "&type=" + url.QueryEscape(protocolType)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result kb3ProtocolsResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse protocol search results: %w", err)
	}

	return result.Protocols, nil
}

// ============================================================================
// COMPLIANCE METHODS
// ============================================================================

// CheckGuidelineCompliance checks patient compliance with a specific guideline.
// Calls KB-3 /v1/guidelines/{id}/compliance endpoint.
func (c *KB3HTTPClient) CheckGuidelineCompliance(
	ctx context.Context,
	guidelineID string,
	patient *contracts.PatientContext,
) (*GuidelineCompliance, error) {

	// Extract medication codes
	medCodes := make([]string, 0, len(patient.ActiveMedications))
	for _, med := range patient.ActiveMedications {
		medCodes = append(medCodes, med.Code.Code)
	}

	// Extract condition codes
	condCodes := make([]string, 0, len(patient.ActiveConditions))
	for _, cond := range patient.ActiveConditions {
		condCodes = append(condCodes, cond.Code.Code)
	}

	// Extract recent labs
	labResults := make([]kb3LabValue, 0, len(patient.RecentLabResults))
	for _, lab := range patient.RecentLabResults {
		if lab.Value != nil {
			labResults = append(labResults, kb3LabValue{
				Code:  lab.Code.Code,
				Value: lab.Value.Value,
				Unit:  lab.Value.Unit,
				Date:  lab.EffectiveDateTime,
			})
		}
	}

	req := kb3ComplianceRequest{
		GuidelineID:     guidelineID,
		PatientID:       patient.Demographics.PatientID,
		MedicationCodes: medCodes,
		ConditionCodes:  condCodes,
		LabResults:      labResults,
	}

	resp, err := c.callKB3(ctx, fmt.Sprintf("/v1/guidelines/%s/compliance", guidelineID), req)
	if err != nil {
		return nil, fmt.Errorf("failed to check compliance: %w", err)
	}

	var result GuidelineCompliance
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse compliance response: %w", err)
	}

	return &result, nil
}

// GetComplianceGaps returns identified gaps in guideline compliance.
// Calls KB-3 /v1/guidelines/compliance/gaps endpoint.
func (c *KB3HTTPClient) GetComplianceGaps(
	ctx context.Context,
	patient *contracts.PatientContext,
) ([]ComplianceGap, error) {

	// Extract condition codes
	condCodes := make([]string, 0, len(patient.ActiveConditions))
	for _, cond := range patient.ActiveConditions {
		condCodes = append(condCodes, cond.Code.Code)
	}

	// Extract medication codes
	medCodes := make([]string, 0, len(patient.ActiveMedications))
	for _, med := range patient.ActiveMedications {
		medCodes = append(medCodes, med.Code.Code)
	}

	req := kb3GapsRequest{
		PatientID:       patient.Demographics.PatientID,
		ConditionCodes:  condCodes,
		MedicationCodes: medCodes,
		Region:          patient.Demographics.Region,
	}

	resp, err := c.callKB3(ctx, "/v1/guidelines/compliance/gaps", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get compliance gaps: %w", err)
	}

	var result kb3GapsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse gaps response: %w", err)
	}

	return result.Gaps, nil
}

// ============================================================================
// DRUG RECOMMENDATION METHODS
// ============================================================================

// GetRecommendedDrugs returns guideline-recommended drugs for an indication.
// Calls KB-3 /v1/guidelines/drugs/recommended endpoint.
//
// NOTE: KB-3 provides guideline context for drug selection, not direct formulary.
// For actual prescribing, coordinate with KB-1 (dosing) and KB-6 (formulary).
func (c *KB3HTTPClient) GetRecommendedDrugs(
	ctx context.Context,
	indication string,
	drugClass string,
) ([]DrugRecommendation, error) {

	req := kb3DrugRecommendationRequest{
		Indication: indication,
		DrugClass:  drugClass,
	}

	resp, err := c.callKB3(ctx, "/v1/guidelines/drugs/recommended", req)
	if err != nil {
		// Fallback: try protocol-based lookup
		return c.getRecommendationsFromProtocols(ctx, indication, drugClass)
	}

	var result kb3DrugRecommendationsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse drug recommendations: %w", err)
	}

	return result.Recommendations, nil
}

// GetFirstLineDrugs returns first-line drug recommendations for an indication.
// Calls KB-3 /v1/guidelines/drugs/first-line endpoint.
func (c *KB3HTTPClient) GetFirstLineDrugs(
	ctx context.Context,
	indication string,
) ([]DrugRecommendation, error) {

	reqURL := fmt.Sprintf("%s/v1/guidelines/drugs/first-line?indication=%s",
		c.baseURL, url.QueryEscape(indication))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result kb3DrugRecommendationsResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse first-line drugs: %w", err)
	}

	// Filter for first-line only
	firstLine := make([]DrugRecommendation, 0)
	for _, rec := range result.Recommendations {
		if rec.IsFirstLine {
			firstLine = append(firstLine, rec)
		}
	}

	return firstLine, nil
}

// GetGuidelineSupport returns guideline evidence for a specific drug.
// Calls KB-3 /v1/guidelines/drugs/{rxnorm}/evidence endpoint.
func (c *KB3HTTPClient) GetGuidelineSupport(
	ctx context.Context,
	rxnormCode string,
	indication string,
) (*GuidelineEvidence, error) {

	reqURL := fmt.Sprintf("%s/v1/guidelines/drugs/%s/evidence?indication=%s",
		c.baseURL, url.PathEscape(rxnormCode), url.QueryEscape(indication))

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result GuidelineEvidence
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse guideline evidence: %w", err)
	}

	return &result, nil
}

// GuidelineEvidence represents evidence from guidelines for a drug.
type GuidelineEvidence struct {
	// RxNormCode of the drug
	RxNormCode string `json:"rxnorm_code"`

	// DrugName generic name
	DrugName string `json:"drug_name"`

	// Indication being evaluated
	Indication string `json:"indication"`

	// SupportingGuidelines that recommend this drug
	SupportingGuidelines []GuidelineSupport `json:"supporting_guidelines,omitempty"`

	// HighestEvidenceGrade among supporting guidelines
	HighestEvidenceGrade string `json:"highest_evidence_grade"`

	// HighestRecommendationClass (I, IIa, IIb, III)
	HighestRecommendationClass string `json:"highest_recommendation_class"`

	// IsGuidelineSupported whether any guideline supports this
	IsGuidelineSupported bool `json:"is_guideline_supported"`
}

// GuidelineSupport represents support from a single guideline.
type GuidelineSupport struct {
	// GuidelineID of the supporting guideline
	GuidelineID string `json:"guideline_id"`

	// GuidelineName for display
	GuidelineName string `json:"guideline_name"`

	// Source organization
	Source string `json:"source"`

	// ClassOfRecommendation (I, IIa, IIb, III)
	ClassOfRecommendation string `json:"class_of_recommendation"`

	// LevelOfEvidence (A, B-R, B-NR, C-LD, C-EO)
	LevelOfEvidence string `json:"level_of_evidence"`

	// RecommendationText the specific recommendation
	RecommendationText string `json:"recommendation_text"`

	// IsFirstLine if recommended as first-line
	IsFirstLine bool `json:"is_first_line"`
}

// ============================================================================
// ICU DOMINANCE INTEGRATION
// ============================================================================

// GetICUProtocols returns protocols specifically designed for ICU settings.
// These protocols may override general ward protocols when ICU Dominance is active.
func (c *KB3HTTPClient) GetICUProtocols(
	ctx context.Context,
	conditionCodes []string,
) ([]Protocol, error) {

	req := kb3ICUProtocolRequest{
		ConditionCodes: conditionCodes,
		Setting:        "ICU",
	}

	resp, err := c.callKB3(ctx, "/v1/protocols/icu", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get ICU protocols: %w", err)
	}

	var result kb3ProtocolsResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ICU protocols: %w", err)
	}

	return result.Protocols, nil
}

// GetCriticalCareGuidelines returns guidelines for critical care conditions.
// Prioritizes sepsis, AKI, ARDS, and other ICU-focused guidelines.
func (c *KB3HTTPClient) GetCriticalCareGuidelines(
	ctx context.Context,
) ([]Guideline, error) {

	reqURL := fmt.Sprintf("%s/v1/guidelines/category/critical-care", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	var result kb3GuidelinesResult
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("failed to parse critical care guidelines: %w", err)
	}

	return result.Guidelines, nil
}

// ============================================================================
// HEALTH CHECK
// ============================================================================

// HealthCheck verifies KB-3 service availability.
func (c *KB3HTTPClient) HealthCheck(ctx context.Context) error {
	reqURL := fmt.Sprintf("%s/health", c.baseURL)

	httpReq, err := http.NewRequestWithContext(ctx, "GET", reqURL, nil)
	if err != nil {
		return fmt.Errorf("failed to create health check request: %w", err)
	}

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("KB-3 health check failed: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-3 unhealthy: status %d", httpResp.StatusCode)
	}

	return nil
}

// ============================================================================
// PRIVATE HELPER METHODS
// ============================================================================

// callKB3 makes a POST request to KB-3 service.
func (c *KB3HTTPClient) callKB3(ctx context.Context, endpoint string, payload interface{}) ([]byte, error) {
	reqURL := fmt.Sprintf("%s%s", c.baseURL, endpoint)

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", reqURL, bytes.NewReader(jsonPayload))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Accept", "application/json")

	httpResp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer httpResp.Body.Close()

	body, err := io.ReadAll(httpResp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if httpResp.StatusCode < 200 || httpResp.StatusCode >= 300 {
		return nil, fmt.Errorf("KB-3 returned status %d: %s", httpResp.StatusCode, string(body))
	}

	return body, nil
}

// getRecommendationsFromProtocols extracts drug recommendations from protocols.
// Fallback when direct drug recommendation endpoint is unavailable.
func (c *KB3HTTPClient) getRecommendationsFromProtocols(
	ctx context.Context,
	indication string,
	drugClass string,
) ([]DrugRecommendation, error) {

	protocols, err := c.GetProtocolsForCondition(ctx, indication)
	if err != nil {
		return []DrugRecommendation{}, nil // Not an error - just no recommendations
	}

	var recommendations []DrugRecommendation

	for _, protocol := range protocols {
		// Check acute protocol stages
		for _, stage := range protocol.Stages {
			for _, action := range stage.Actions {
				if action.Type == "medication" {
					// Filter by drug class if specified
					if drugClass != "" && action.DrugClass != drugClass {
						continue
					}

					rec := DrugRecommendation{
						DrugName:        action.Medication,
						RxNormCode:      action.RxNormCode,
						DrugClass:       action.DrugClass,
						GuidelineSource: protocol.GuidelineSource,
						EvidenceGrade:   action.EvidenceGrade,
						IsFirstLine:     stage.Order == 1, // First stage = first line
						Indication:      indication,
						TypicalDose:     fmt.Sprintf("%v %s", action.Dose, action.DoseUnit),
					}
					recommendations = append(recommendations, rec)
				}
			}
		}

		// Check monitoring items for medication type
		for _, item := range protocol.MonitoringItems {
			if item.Type == "medication" {
				rec := DrugRecommendation{
					DrugName:        item.Name,
					GuidelineSource: protocol.GuidelineSource,
					EvidenceGrade:   "B",
					IsFirstLine:     true,
					Indication:      indication,
				}
				recommendations = append(recommendations, rec)
			}
		}
	}

	return recommendations, nil
}

// ============================================================================
// REQUEST/RESPONSE TYPES (PRIVATE)
// ============================================================================

type kb3ApplicableGuidelinesRequest struct {
	PatientID      string   `json:"patient_id"`
	ConditionCodes []string `json:"condition_codes"`
	Region         string   `json:"region,omitempty"`
}

type kb3GuidelinesResult struct {
	Guidelines []Guideline `json:"guidelines"`
}

type kb3GuidelineDetailResult struct {
	Guideline       Guideline                 `json:"guideline"`
	Recommendations []GuidelineRecommendation `json:"recommendations"`
}

type kb3ProtocolsResult struct {
	Protocols []Protocol `json:"protocols"`
}

type kb3ComplianceRequest struct {
	GuidelineID     string        `json:"guideline_id"`
	PatientID       string        `json:"patient_id"`
	MedicationCodes []string      `json:"medication_codes"`
	ConditionCodes  []string      `json:"condition_codes"`
	LabResults      []kb3LabValue `json:"lab_results,omitempty"`
}

type kb3LabValue struct {
	Code  string     `json:"code"`
	Value float64    `json:"value"`
	Unit  string     `json:"unit"`
	Date  *time.Time `json:"date,omitempty"`
}

type kb3GapsRequest struct {
	PatientID       string   `json:"patient_id"`
	ConditionCodes  []string `json:"condition_codes"`
	MedicationCodes []string `json:"medication_codes"`
	Region          string   `json:"region,omitempty"`
}

type kb3GapsResult struct {
	Gaps []ComplianceGap `json:"gaps"`
}

type kb3DrugRecommendationRequest struct {
	Indication string `json:"indication"`
	DrugClass  string `json:"drug_class,omitempty"`
}

type kb3DrugRecommendationsResult struct {
	Recommendations []DrugRecommendation `json:"recommendations"`
}

type kb3ICUProtocolRequest struct {
	ConditionCodes []string `json:"condition_codes"`
	Setting        string   `json:"setting"`
}

// ============================================================================
// INTERFACE COMPLIANCE DOCUMENTATION
// ============================================================================
//
// KB3HTTPClient implements the following interface methods:
//
// Guideline Methods:
//   - GetApplicableGuidelines(ctx, patient) → []Guideline
//   - GetGuidelineDetail(ctx, guidelineID) → *Guideline, []GuidelineRecommendation
//   - GetGuidelinesByCondition(ctx, conditionCode) → []Guideline
//   - GetCriticalCareGuidelines(ctx) → []Guideline
//
// Protocol Methods:
//   - GetProtocolsForCondition(ctx, conditionCode) → []Protocol
//   - GetProtocolDetail(ctx, type, id) → *Protocol
//   - SearchProtocols(ctx, query, type) → []Protocol
//   - GetICUProtocols(ctx, conditionCodes) → []Protocol
//
// Compliance Methods:
//   - CheckGuidelineCompliance(ctx, guidelineID, patient) → *GuidelineCompliance
//   - GetComplianceGaps(ctx, patient) → []ComplianceGap
//
// Drug Recommendation Methods (Guideline Context):
//   - GetRecommendedDrugs(ctx, indication, drugClass) → []DrugRecommendation
//   - GetFirstLineDrugs(ctx, indication) → []DrugRecommendation
//   - GetGuidelineSupport(ctx, rxnormCode, indication) → *GuidelineEvidence
//
// Health:
//   - HealthCheck(ctx) → error
// ============================================================================
