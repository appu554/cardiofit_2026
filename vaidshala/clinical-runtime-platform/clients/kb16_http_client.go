// Package clients provides HTTP/GraphQL clients for KB services.
//
// KB16HTTPClient implements the KB16Client interface for KB-16 Lab Interpretation Service.
// It provides reference ranges, critical value thresholds, and lab panel interpretations.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client IS PART of KnowledgeSnapshotBuilder (Category A - SNAPSHOT KB).
// Lab reference ranges are pre-computed at snapshot build time - CQL evaluates against
// frozen lab interpretation data. Engines NEVER call KB-16 directly at execution time.
//
// "CQL does not need a terminology ENGINE at runtime — it needs a terminology ANSWER."
//
// Connects to: http://localhost:8098 (Docker: kb16-lab-interpretation)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// ============================================================================
// KB-16 CLIENT INTERFACE (SNAPSHOT KB - Category A)
// ============================================================================

// KB16Client defines the interface for Lab Interpretation service interactions.
// This is a SNAPSHOT KB - used at build time to produce LabInterpretationSnapshot.
//
// ARCHITECTURE: KB-16 is called during KnowledgeSnapshotBuilder.Build() to
// pre-compute lab interpretations, reference ranges, and critical value thresholds.
// CQL consumes these frozen values from the snapshot.
type KB16Client interface {
	// InterpretLabResult interprets a single lab result with patient context
	// Returns interpretation including abnormality flags, clinical significance
	InterpretLabResult(ctx context.Context, labResult LabResult, patientContext PatientDemographics) (*LabInterpretation, error)

	// GetReferenceRange returns age/sex appropriate reference range for a LOINC code
	// Reference ranges are patient-specific based on demographics
	GetReferenceRange(ctx context.Context, loincCode string, patientContext PatientDemographics) (*ReferenceRange, error)

	// InterpretLabPanel interprets a panel of related labs (e.g., BMP, CMP, Lipid Panel)
	// Provides holistic interpretation considering relationships between values
	InterpretLabPanel(ctx context.Context, panelType string, labs []LabResult, patientContext PatientDemographics) (*PanelInterpretation, error)

	// GetCriticalValues returns critical value thresholds for a LOINC code
	// Critical values trigger immediate clinical action
	GetCriticalValues(ctx context.Context, loincCode string) (*CriticalValues, error)

	// GetReferenceRangesForPatient returns all applicable reference ranges for patient
	// Used at snapshot build time to pre-compute all lab thresholds efficiently
	GetReferenceRangesForPatient(ctx context.Context, patientContext PatientDemographics, loincCodes []string) (map[string]*ReferenceRange, error)

	// BatchInterpretLabs interprets multiple lab results in a single call
	// More efficient than individual InterpretLabResult calls
	BatchInterpretLabs(ctx context.Context, labs []LabResult, patientContext PatientDemographics) ([]LabInterpretation, error)

	// GetDeltaCheck returns expected change thresholds for labs
	// Used to detect clinically significant changes between measurements
	GetDeltaCheck(ctx context.Context, loincCode string, previousValue float64, currentValue float64, hoursElapsed float64) (*DeltaCheckResult, error)

	// HealthCheck verifies KB-16 service is operational
	HealthCheck(ctx context.Context) error
}

// ============================================================================
// DATA TYPES FOR KB-16 (Lab Interpretation Domain)
// ============================================================================

// LabResult represents a single laboratory measurement.
type LabResult struct {
	// LOINC code for the lab test
	LOINCCode string `json:"loinc_code"`

	// Display name of the lab test
	DisplayName string `json:"display_name"`

	// Numeric value of the result
	Value float64 `json:"value"`

	// Unit of measurement (e.g., "mg/dL", "mmol/L")
	Unit string `json:"unit"`

	// Timestamp when the lab was collected
	CollectedAt time.Time `json:"collected_at"`

	// Timestamp when the lab was resulted
	ResultedAt time.Time `json:"resulted_at"`

	// Optional: String value for non-numeric results
	StringValue string `json:"string_value,omitempty"`

	// Optional: Indicates if result is flagged by lab
	LabFlag string `json:"lab_flag,omitempty"` // "H", "L", "HH", "LL", "A", "AA"
}

// PatientDemographics contains patient info needed for reference range selection.
type PatientDemographics struct {
	// Age in years
	AgeYears int `json:"age_years"`

	// Biological sex (male, female)
	Sex string `json:"sex"`

	// Pregnancy status
	IsPregnant bool `json:"is_pregnant,omitempty"`

	// Trimester if pregnant (1, 2, 3)
	Trimester int `json:"trimester,omitempty"`

	// Race/Ethnicity (may affect some reference ranges)
	Ethnicity string `json:"ethnicity,omitempty"`

	// Region for regional reference ranges
	Region string `json:"region,omitempty"`

	// Fasting status (affects glucose, lipids)
	IsFasting bool `json:"is_fasting,omitempty"`
}

// ReferenceRange contains the normal range for a lab value.
type ReferenceRange struct {
	// LOINC code
	LOINCCode string `json:"loinc_code"`

	// Low end of normal range
	LowNormal float64 `json:"low_normal"`

	// High end of normal range
	HighNormal float64 `json:"high_normal"`

	// Unit of measurement
	Unit string `json:"unit"`

	// Critical low (panic value)
	CriticalLow *float64 `json:"critical_low,omitempty"`

	// Critical high (panic value)
	CriticalHigh *float64 `json:"critical_high,omitempty"`

	// Age range this reference applies to
	AgeMin int `json:"age_min,omitempty"`
	AgeMax int `json:"age_max,omitempty"`

	// Sex this reference applies to (empty = both)
	Sex string `json:"sex,omitempty"`

	// Pregnancy-specific range
	IsPregnancyRange bool `json:"is_pregnancy_range,omitempty"`

	// Source of the reference range
	Source string `json:"source,omitempty"`
}

// CriticalValues contains critical/panic value thresholds.
type CriticalValues struct {
	// LOINC code
	LOINCCode string `json:"loinc_code"`

	// Display name
	DisplayName string `json:"display_name"`

	// Critical low threshold
	CriticalLow *float64 `json:"critical_low,omitempty"`

	// Critical high threshold
	CriticalHigh *float64 `json:"critical_high,omitempty"`

	// Unit of measurement
	Unit string `json:"unit"`

	// Required action for critical values
	RequiredAction string `json:"required_action,omitempty"`

	// Time to notify in minutes
	NotifyWithinMinutes int `json:"notify_within_minutes,omitempty"`

	// Source/Guideline reference
	Source string `json:"source,omitempty"`
}

// LabInterpretation provides clinical interpretation of a lab result.
type LabInterpretation struct {
	// Original lab result
	LabResult LabResult `json:"lab_result"`

	// Reference range used
	ReferenceRange ReferenceRange `json:"reference_range"`

	// Abnormality level
	AbnormalityLevel string `json:"abnormality_level"` // "normal", "low", "high", "critical_low", "critical_high"

	// Abnormality flag for display
	Flag string `json:"flag"` // "", "L", "H", "LL", "HH"

	// Is this a critical/panic value?
	IsCritical bool `json:"is_critical"`

	// Clinical significance
	ClinicalSignificance string `json:"clinical_significance"` // "none", "mild", "moderate", "severe"

	// Possible causes for abnormality
	PossibleCauses []string `json:"possible_causes,omitempty"`

	// Suggested follow-up actions
	SuggestedActions []string `json:"suggested_actions,omitempty"`

	// Related conditions to consider
	RelatedConditions []string `json:"related_conditions,omitempty"`

	// Trending direction if historical data available
	Trend string `json:"trend,omitempty"` // "stable", "improving", "worsening", "unknown"

	// Interpretation narrative
	Narrative string `json:"narrative,omitempty"`
}

// PanelInterpretation provides holistic interpretation of a lab panel.
type PanelInterpretation struct {
	// Panel type (e.g., "BMP", "CMP", "CBC", "LipidPanel")
	PanelType string `json:"panel_type"`

	// Panel display name
	DisplayName string `json:"display_name"`

	// Individual lab interpretations
	LabInterpretations []LabInterpretation `json:"lab_interpretations"`

	// Overall panel assessment
	OverallAssessment string `json:"overall_assessment"` // "normal", "abnormal", "critical"

	// Panel-level findings (relationships between values)
	PanelFindings []PanelFinding `json:"panel_findings,omitempty"`

	// Suggested differential diagnoses
	DifferentialDiagnoses []string `json:"differential_diagnoses,omitempty"`

	// Recommended follow-up tests
	RecommendedFollowUp []string `json:"recommended_follow_up,omitempty"`

	// Panel-level narrative
	Narrative string `json:"narrative,omitempty"`
}

// PanelFinding represents a finding from analyzing multiple labs together.
type PanelFinding struct {
	// Finding type
	Type string `json:"type"` // "pattern", "ratio", "correlation"

	// Finding description
	Description string `json:"description"`

	// Labs involved in this finding
	InvolvedLabs []string `json:"involved_labs"`

	// Clinical significance
	Significance string `json:"significance"` // "informational", "notable", "concerning"

	// Possible interpretation
	Interpretation string `json:"interpretation,omitempty"`
}

// DeltaCheckResult contains analysis of lab value changes over time.
type DeltaCheckResult struct {
	// LOINC code
	LOINCCode string `json:"loinc_code"`

	// Previous value
	PreviousValue float64 `json:"previous_value"`

	// Current value
	CurrentValue float64 `json:"current_value"`

	// Absolute change
	AbsoluteChange float64 `json:"absolute_change"`

	// Percent change
	PercentChange float64 `json:"percent_change"`

	// Time elapsed in hours
	HoursElapsed float64 `json:"hours_elapsed"`

	// Is this change clinically significant?
	IsClinicallySignificant bool `json:"is_clinically_significant"`

	// Rate of change per hour
	RatePerHour float64 `json:"rate_per_hour"`

	// Delta check threshold that was exceeded (if any)
	ThresholdExceeded *float64 `json:"threshold_exceeded,omitempty"`

	// Narrative interpretation
	Interpretation string `json:"interpretation,omitempty"`
}

// ============================================================================
// KB-16 HTTP CLIENT IMPLEMENTATION
// ============================================================================

// KB16HTTPClient implements KB16Client by calling the KB-16 Lab Interpretation REST API.
type KB16HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB16HTTPClient creates a new KB-16 HTTP client with default timeout.
func NewKB16HTTPClient(baseURL string) *KB16HTTPClient {
	return &KB16HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB16HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB16HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB16HTTPClient {
	return &KB16HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB16Client Interface Implementation
// ============================================================================

// InterpretLabResult interprets a single lab result with patient context.
func (c *KB16HTTPClient) InterpretLabResult(
	ctx context.Context,
	labResult LabResult,
	patientContext PatientDemographics,
) (*LabInterpretation, error) {

	req := kb16InterpretLabRequest{
		LabResult:      labResult,
		PatientContext: patientContext,
	}

	respBody, err := c.callKB16(ctx, "/api/v1/interpret", req)
	if err != nil {
		return nil, fmt.Errorf("interpret lab result failed: %w", err)
	}

	var resp kb16InterpretLabResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse interpret response: %w", err)
	}

	return &resp.Interpretation, nil
}

// GetReferenceRange returns age/sex appropriate reference range for a LOINC code.
func (c *KB16HTTPClient) GetReferenceRange(
	ctx context.Context,
	loincCode string,
	patientContext PatientDemographics,
) (*ReferenceRange, error) {

	req := kb16ReferenceRangeRequest{
		LOINCCode:      loincCode,
		PatientContext: patientContext,
	}

	respBody, err := c.callKB16(ctx, "/api/v1/reference-range", req)
	if err != nil {
		return nil, fmt.Errorf("get reference range failed: %w", err)
	}

	var resp kb16ReferenceRangeResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse reference range response: %w", err)
	}

	return &resp.ReferenceRange, nil
}

// InterpretLabPanel interprets a panel of related labs.
func (c *KB16HTTPClient) InterpretLabPanel(
	ctx context.Context,
	panelType string,
	labs []LabResult,
	patientContext PatientDemographics,
) (*PanelInterpretation, error) {

	req := kb16InterpretPanelRequest{
		PanelType:      panelType,
		Labs:           labs,
		PatientContext: patientContext,
	}

	respBody, err := c.callKB16(ctx, "/api/v1/interpret-panel", req)
	if err != nil {
		return nil, fmt.Errorf("interpret lab panel failed: %w", err)
	}

	var resp kb16InterpretPanelResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse panel interpretation response: %w", err)
	}

	return &resp.PanelInterpretation, nil
}

// GetCriticalValues returns critical value thresholds for a LOINC code.
func (c *KB16HTTPClient) GetCriticalValues(
	ctx context.Context,
	loincCode string,
) (*CriticalValues, error) {

	respBody, err := c.callKB16(ctx, fmt.Sprintf("/api/v1/critical-values/%s", loincCode), nil)
	if err != nil {
		return nil, fmt.Errorf("get critical values failed: %w", err)
	}

	var resp kb16CriticalValuesResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse critical values response: %w", err)
	}

	return &resp.CriticalValues, nil
}

// GetReferenceRangesForPatient returns all applicable reference ranges for a patient.
// This is the primary method called during snapshot build time.
func (c *KB16HTTPClient) GetReferenceRangesForPatient(
	ctx context.Context,
	patientContext PatientDemographics,
	loincCodes []string,
) (map[string]*ReferenceRange, error) {

	req := kb16BatchReferenceRangeRequest{
		PatientContext: patientContext,
		LOINCCodes:     loincCodes,
	}

	respBody, err := c.callKB16(ctx, "/api/v1/reference-ranges/batch", req)
	if err != nil {
		return nil, fmt.Errorf("get batch reference ranges failed: %w", err)
	}

	var resp kb16BatchReferenceRangeResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse batch reference range response: %w", err)
	}

	return resp.ReferenceRanges, nil
}

// BatchInterpretLabs interprets multiple lab results in a single call.
func (c *KB16HTTPClient) BatchInterpretLabs(
	ctx context.Context,
	labs []LabResult,
	patientContext PatientDemographics,
) ([]LabInterpretation, error) {

	req := kb16BatchInterpretRequest{
		Labs:           labs,
		PatientContext: patientContext,
	}

	respBody, err := c.callKB16(ctx, "/api/v1/interpret/batch", req)
	if err != nil {
		return nil, fmt.Errorf("batch interpret labs failed: %w", err)
	}

	var resp kb16BatchInterpretResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse batch interpret response: %w", err)
	}

	return resp.Interpretations, nil
}

// GetDeltaCheck returns expected change thresholds for labs.
func (c *KB16HTTPClient) GetDeltaCheck(
	ctx context.Context,
	loincCode string,
	previousValue float64,
	currentValue float64,
	hoursElapsed float64,
) (*DeltaCheckResult, error) {

	req := kb16DeltaCheckRequest{
		LOINCCode:     loincCode,
		PreviousValue: previousValue,
		CurrentValue:  currentValue,
		HoursElapsed:  hoursElapsed,
	}

	respBody, err := c.callKB16(ctx, "/api/v1/delta-check", req)
	if err != nil {
		return nil, fmt.Errorf("delta check failed: %w", err)
	}

	var resp kb16DeltaCheckResponse
	if err := json.Unmarshal(respBody, &resp); err != nil {
		return nil, fmt.Errorf("failed to parse delta check response: %w", err)
	}

	return &resp.DeltaCheck, nil
}

// HealthCheck verifies KB-16 service is operational.
func (c *KB16HTTPClient) HealthCheck(ctx context.Context) error {
	respBody, err := c.callKB16(ctx, "/health", nil)
	if err != nil {
		return fmt.Errorf("KB-16 health check failed: %w", err)
	}

	var health struct {
		Status string `json:"status"`
	}
	if err := json.Unmarshal(respBody, &health); err != nil {
		return fmt.Errorf("failed to parse health response: %w", err)
	}

	if health.Status != "healthy" && health.Status != "ok" {
		return fmt.Errorf("KB-16 unhealthy: %s", health.Status)
	}

	return nil
}

// ============================================================================
// INTERNAL HTTP HELPER
// ============================================================================

func (c *KB16HTTPClient) callKB16(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	var req *http.Request
	var err error

	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		req, err = http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonBody))
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
		req.Header.Set("Content-Type", "application/json")
	} else {
		req, err = http.NewRequestWithContext(ctx, "GET", url, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to create request: %w", err)
		}
	}

	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("KB-16 returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// ============================================================================
// INTERNAL REQUEST/RESPONSE TYPES
// ============================================================================

type kb16InterpretLabRequest struct {
	LabResult      LabResult           `json:"lab_result"`
	PatientContext PatientDemographics `json:"patient_context"`
}

type kb16InterpretLabResponse struct {
	Interpretation LabInterpretation `json:"interpretation"`
}

type kb16ReferenceRangeRequest struct {
	LOINCCode      string              `json:"loinc_code"`
	PatientContext PatientDemographics `json:"patient_context"`
}

type kb16ReferenceRangeResponse struct {
	ReferenceRange ReferenceRange `json:"reference_range"`
}

type kb16InterpretPanelRequest struct {
	PanelType      string              `json:"panel_type"`
	Labs           []LabResult         `json:"labs"`
	PatientContext PatientDemographics `json:"patient_context"`
}

type kb16InterpretPanelResponse struct {
	PanelInterpretation PanelInterpretation `json:"panel_interpretation"`
}

type kb16CriticalValuesResponse struct {
	CriticalValues CriticalValues `json:"critical_values"`
}

type kb16BatchReferenceRangeRequest struct {
	PatientContext PatientDemographics `json:"patient_context"`
	LOINCCodes     []string            `json:"loinc_codes"`
}

type kb16BatchReferenceRangeResponse struct {
	ReferenceRanges map[string]*ReferenceRange `json:"reference_ranges"`
}

type kb16BatchInterpretRequest struct {
	Labs           []LabResult         `json:"labs"`
	PatientContext PatientDemographics `json:"patient_context"`
}

type kb16BatchInterpretResponse struct {
	Interpretations []LabInterpretation `json:"interpretations"`
}

type kb16DeltaCheckRequest struct {
	LOINCCode     string  `json:"loinc_code"`
	PreviousValue float64 `json:"previous_value"`
	CurrentValue  float64 `json:"current_value"`
	HoursElapsed  float64 `json:"hours_elapsed"`
}

type kb16DeltaCheckResponse struct {
	DeltaCheck DeltaCheckResult `json:"delta_check"`
}

// ============================================================================
// COMMON LAB LOINC CODES (Reference for snapshot builder)
// ============================================================================

// CommonLabLOINCCodes contains frequently used lab LOINC codes for snapshot building.
// The KnowledgeSnapshotBuilder uses these to pre-fetch reference ranges.
var CommonLabLOINCCodes = []string{
	// Basic Metabolic Panel
	"2345-7",  // Glucose
	"2160-0",  // Creatinine
	"3094-0",  // BUN
	"2951-2",  // Sodium
	"2823-3",  // Potassium
	"2075-0",  // Chloride
	"1963-8",  // Bicarbonate (CO2)
	"17861-6", // Calcium
	"33037-3", // Anion Gap

	// Complete Blood Count
	"718-7",   // Hemoglobin
	"4544-3",  // Hematocrit
	"6690-2",  // WBC
	"777-3",   // Platelet Count
	"26515-7", // Platelets
	"789-8",   // RBC
	"787-2",   // MCV
	"788-0",   // MCH
	"786-4",   // MCHC

	// Lipid Panel
	"2093-3", // Total Cholesterol
	"2085-9", // HDL
	"13457-7", // LDL (calculated)
	"2571-8", // Triglycerides

	// Liver Function
	"1742-6", // ALT
	"1920-8", // AST
	"1975-2", // Bilirubin Total
	"1968-7", // Bilirubin Direct
	"6768-6", // Alkaline Phosphatase
	"1751-7", // Albumin

	// Coagulation
	"5902-2",  // Prothrombin Time
	"6301-6",  // INR
	"3173-2",  // PTT
	"30385-9", // D-dimer

	// Cardiac
	"10839-9", // Troponin I
	"49563-0", // Troponin T
	"42719-5", // BNP
	"33762-6", // NT-proBNP

	// Renal/Electrolytes
	"33914-3", // eGFR
	"14959-1", // Microalbumin/Creatinine Ratio
	"2777-1",  // Phosphorus
	"19123-9", // Magnesium

	// Thyroid
	"3016-3", // TSH
	"3053-6", // Free T4
	"3051-0", // Free T3

	// Inflammatory
	"1988-5", // CRP
	"30522-7", // hsCRP
	"4537-7", // ESR

	// Diabetes
	"4548-4", // HbA1c
	"14749-6", // Fasting Glucose

	// Infectious Disease
	"5767-9", // Procalcitonin
	"49498-9", // Lactate
}

// ============================================================================
// INTERFACE COMPLIANCE DOCUMENTATION
// ============================================================================
//
// This file implements:
//   - KB16Client interface for Lab Interpretation Service (Category A - SNAPSHOT KB)
//
// Integration Points:
//   - KnowledgeSnapshotBuilder.Build() → buildLabInterpretationSnapshot()
//   - LabInterpretationSnapshot is consumed by CQL via frozen snapshot
//   - Tier 1.5 CQL: LabReferenceRanges.cql uses this snapshot
//
// API Endpoints Called:
//   - POST /api/v1/interpret - Single lab interpretation
//   - POST /api/v1/interpret/batch - Batch lab interpretation
//   - POST /api/v1/interpret-panel - Panel interpretation
//   - POST /api/v1/reference-range - Single reference range
//   - POST /api/v1/reference-ranges/batch - Batch reference ranges
//   - GET  /api/v1/critical-values/{loinc} - Critical value thresholds
//   - POST /api/v1/delta-check - Delta check analysis
//   - GET  /health - Health check
//
// CTO/CMO Directive Compliance:
//   - ✅ SNAPSHOT KB (Category A) - Pre-computed at build time
//   - ✅ Frozen data for CQL - No runtime KB calls from CQL
//   - ✅ O(1) lookups - Reference ranges keyed by LOINC
//   - ✅ Deterministic - Same inputs produce same outputs
// ============================================================================
