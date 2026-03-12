// Package clients provides HTTP clients for KB-19 to communicate with upstream services.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// VaidshalaClient is the HTTP client for the Vaidshala CQL Engine.
// The CQL Engine is the source of clinical truth - it evaluates CQL expressions
// against patient data and returns boolean flags indicating clinical conditions.
type VaidshalaClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewVaidshalaClient creates a new VaidshalaClient.
func NewVaidshalaClient(baseURL string, timeout time.Duration, log *logrus.Entry) *VaidshalaClient {
	return &VaidshalaClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.WithField("client", "vaidshala"),
	}
}

// CQLEvaluationRequest is the request to evaluate CQL expressions.
type CQLEvaluationRequest struct {
	PatientID   uuid.UUID `json:"patient_id"`
	EncounterID uuid.UUID `json:"encounter_id,omitempty"`
	LibraryIDs  []string  `json:"library_ids,omitempty"`
	Expressions []string  `json:"expressions,omitempty"`
}

// CQLEvaluationResponse is the response from CQL evaluation.
type CQLEvaluationResponse struct {
	PatientID    uuid.UUID              `json:"patient_id"`
	Results      map[string]interface{} `json:"results"`
	TruthFlags   map[string]bool        `json:"truth_flags"`
	Errors       []string               `json:"errors,omitempty"`
	EvaluatedAt  time.Time              `json:"evaluated_at"`
	EngineVersion string                `json:"engine_version"`
}

// ClinicalExecutionContext is the complete clinical context for a patient.
type ClinicalExecutionContext struct {
	PatientID        uuid.UUID              `json:"patient_id"`
	EncounterID      uuid.UUID              `json:"encounter_id"`
	Demographics     Demographics           `json:"demographics"`
	ActiveConditions []Condition            `json:"active_conditions"`
	ActiveMedications []Medication          `json:"active_medications"`
	RecentLabs       []LabResult            `json:"recent_labs"`
	RecentVitals     []VitalSign            `json:"recent_vitals"`
	CQLTruthFlags    map[string]bool        `json:"cql_truth_flags"`
	CalculatorScores map[string]float64     `json:"calculator_scores"`
	ICUState         *ICUState              `json:"icu_state,omitempty"`
	PregnancyStatus  *PregnancyStatus       `json:"pregnancy_status,omitempty"`
	Timestamp        time.Time              `json:"timestamp"`
}

// Demographics represents patient demographic information.
type Demographics struct {
	Age         int     `json:"age"`
	Sex         string  `json:"sex"`
	Weight      float64 `json:"weight_kg"`
	Height      float64 `json:"height_cm"`
	BSA         float64 `json:"bsa"`
	Ethnicity   string  `json:"ethnicity,omitempty"`
}

// Condition represents an active clinical condition.
type Condition struct {
	Code        string    `json:"code"`
	System      string    `json:"system"` // SNOMED, ICD-10
	Display     string    `json:"display"`
	OnsetDate   time.Time `json:"onset_date"`
	IsActive    bool      `json:"is_active"`
	Severity    string    `json:"severity,omitempty"`
}

// Medication represents an active medication.
type Medication struct {
	Code        string    `json:"code"`
	System      string    `json:"system"` // RxNorm
	Display     string    `json:"display"`
	Dose        string    `json:"dose"`
	Route       string    `json:"route"`
	Frequency   string    `json:"frequency"`
	StartDate   time.Time `json:"start_date"`
}

// LabResult represents a laboratory result.
type LabResult struct {
	Code        string    `json:"code"`
	System      string    `json:"system"` // LOINC
	Display     string    `json:"display"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	ReferenceRange string `json:"reference_range"`
	IsAbnormal  bool      `json:"is_abnormal"`
	CollectedAt time.Time `json:"collected_at"`
}

// VitalSign represents a vital sign measurement.
type VitalSign struct {
	Code        string    `json:"code"`
	Display     string    `json:"display"`
	Value       float64   `json:"value"`
	Unit        string    `json:"unit"`
	RecordedAt  time.Time `json:"recorded_at"`
}

// ICUState represents ICU-specific clinical state.
type ICUState struct {
	IsICU           bool    `json:"is_icu"`
	ShockState      string  `json:"shock_state"`      // NONE, COMPENSATED, UNCOMPENSATED
	VentilatorMode  string  `json:"ventilator_mode"`
	SOFAScore       int     `json:"sofa_score"`
	AKIStage        int     `json:"aki_stage"`
	DICScore        int     `json:"dic_score"`
	PlateletsLow    bool    `json:"platelets_low"`
	BleedingRisk    string  `json:"bleeding_risk"`    // LOW, MODERATE, HIGH
	VasopressorCount int    `json:"vasopressor_count"`
}

// PregnancyStatus represents pregnancy information.
type PregnancyStatus struct {
	IsPregnant      bool      `json:"is_pregnant"`
	GestationalAge  int       `json:"gestational_age_weeks"`
	Trimester       int       `json:"trimester"`
	HighRisk        bool      `json:"high_risk"`
	ExpectedDueDate time.Time `json:"expected_due_date"`
}

// EvaluateCQL evaluates CQL expressions for a patient and returns truth flags.
func (c *VaidshalaClient) EvaluateCQL(ctx context.Context, req CQLEvaluationRequest) (*CQLEvaluationResponse, error) {
	c.log.WithField("patient_id", req.PatientID).Debug("Evaluating CQL expressions")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v1/cql/evaluate", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("CQL evaluation failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result CQLEvaluationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"patient_id":   req.PatientID,
		"truth_flags":  len(result.TruthFlags),
	}).Debug("CQL evaluation complete")

	return &result, nil
}

// GetClinicalContext retrieves the complete clinical execution context for a patient.
func (c *VaidshalaClient) GetClinicalContext(ctx context.Context, patientID uuid.UUID) (*ClinicalExecutionContext, error) {
	c.log.WithField("patient_id", patientID).Debug("Getting clinical context")

	url := fmt.Sprintf("%s/api/v1/clinical-context/%s", c.baseURL, patientID.String())
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get clinical context failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	var result ClinicalExecutionContext
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"patient_id":    patientID,
		"conditions":    len(result.ActiveConditions),
		"medications":   len(result.ActiveMedications),
		"truth_flags":   len(result.CQLTruthFlags),
	}).Debug("Clinical context retrieved")

	return &result, nil
}

// GetTruthFlags retrieves just the CQL truth flags for a patient.
// This is a convenience method that extracts truth flags from the clinical context.
func (c *VaidshalaClient) GetTruthFlags(ctx context.Context, patientID uuid.UUID) (map[string]bool, error) {
	clinicalCtx, err := c.GetClinicalContext(ctx, patientID)
	if err != nil {
		return nil, err
	}
	return clinicalCtx.CQLTruthFlags, nil
}

// Health checks if the Vaidshala CQL Engine is healthy.
func (c *VaidshalaClient) Health(ctx context.Context) error {
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("health check returned status %d", resp.StatusCode)
	}

	return nil
}
