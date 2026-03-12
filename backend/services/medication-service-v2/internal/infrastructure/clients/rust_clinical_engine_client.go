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
	"go.uber.org/zap"
)

// RustClinicalEngineClient handles communication with the Rust Clinical Engine
type RustClinicalEngineClient struct {
	baseURL        string
	httpClient     *http.Client
	logger         *zap.Logger
	timeout        time.Duration
	maxRetries     int
	retryDelay     time.Duration
	metricsEnabled bool
}

// RustClinicalEngineConfig holds configuration for the Rust engine client
type RustClinicalEngineConfig struct {
	BaseURL        string        `json:"base_url" mapstructure:"base_url"`
	Timeout        time.Duration `json:"timeout" mapstructure:"timeout"`
	MaxRetries     int           `json:"max_retries" mapstructure:"max_retries"`
	RetryDelay     time.Duration `json:"retry_delay" mapstructure:"retry_delay"`
	MetricsEnabled bool          `json:"metrics_enabled" mapstructure:"metrics_enabled"`
}

// NewRustClinicalEngineClient creates a new Rust Clinical Engine client
func NewRustClinicalEngineClient(config *RustClinicalEngineConfig, logger *zap.Logger) *RustClinicalEngineClient {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Second
	}
	if config.MaxRetries == 0 {
		config.MaxRetries = 3
	}
	if config.RetryDelay == 0 {
		config.RetryDelay = 100 * time.Millisecond
	}

	return &RustClinicalEngineClient{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger:         logger.Named("rust-clinical-engine-client"),
		timeout:        config.Timeout,
		maxRetries:     config.MaxRetries,
		retryDelay:     config.RetryDelay,
		metricsEnabled: config.MetricsEnabled,
	}
}

// Clinical Engine Request Types

// DrugInteractionRequest represents a request for drug interaction analysis
type DrugInteractionRequest struct {
	RequestID       string                 `json:"request_id"`
	PatientID       string                 `json:"patient_id"`
	Medications     []MedicationForAnalysis `json:"medications"`
	ClinicalContext map[string]interface{} `json:"clinical_context"`
	AnalysisDepth   string                 `json:"analysis_depth"` // shallow, deep, comprehensive
	Priority        string                 `json:"priority"`       // low, normal, high, critical
}

// MedicationForAnalysis represents medication data for analysis
type MedicationForAnalysis struct {
	MedicationCode string                 `json:"medication_code"`
	Name           string                 `json:"name"`
	Dose           string                 `json:"dose"`
	Route          string                 `json:"route"`
	Frequency      string                 `json:"frequency"`
	StartDate      *time.Time             `json:"start_date,omitempty"`
	EndDate        *time.Time             `json:"end_date,omitempty"`
	IsActive       bool                   `json:"is_active"`
	MetaData       map[string]interface{} `json:"metadata,omitempty"`
}

// DosageCalculationRequest represents a request for dosage calculations
type DosageCalculationRequest struct {
	RequestID       string                 `json:"request_id"`
	PatientID       string                 `json:"patient_id"`
	MedicationCode  string                 `json:"medication_code"`
	PatientWeight   float64                `json:"patient_weight"`
	PatientAge      int                    `json:"patient_age"`
	KidneyFunction  *KidneyFunctionData    `json:"kidney_function,omitempty"`
	LiverFunction   *LiverFunctionData     `json:"liver_function,omitempty"`
	ClinicalContext map[string]interface{} `json:"clinical_context"`
	CalculationType string                 `json:"calculation_type"` // standard, pediatric, geriatric, renal_impaired
}

// KidneyFunctionData represents kidney function parameters
type KidneyFunctionData struct {
	CreatinineLevel float64 `json:"creatinine_level"`
	CreatinineClearance float64 `json:"creatinine_clearance"`
	GFR             float64 `json:"gfr"`
	Stage           string  `json:"stage"` // normal, mild, moderate, severe, dialysis
}

// LiverFunctionData represents liver function parameters  
type LiverFunctionData struct {
	ALTLevel    float64 `json:"alt_level"`
	ASTLevel    float64 `json:"ast_level"`
	BilirubinLevel float64 `json:"bilirubin_level"`
	AlbuminLevel   float64 `json:"albumin_level"`
	ChildPughScore int     `json:"child_pugh_score"`
}

// SafetyValidationRequest represents a request for safety validation
type SafetyValidationRequest struct {
	RequestID       string                 `json:"request_id"`
	PatientID       string                 `json:"patient_id"`
	ProposedMedication *MedicationForAnalysis `json:"proposed_medication"`
	CurrentMedications []MedicationForAnalysis `json:"current_medications"`
	Allergies       []AllergyInfo          `json:"allergies"`
	Conditions      []ConditionInfo        `json:"conditions"`
	ClinicalContext map[string]interface{} `json:"clinical_context"`
	ValidationLevel string                 `json:"validation_level"` // basic, standard, comprehensive
}

// AllergyInfo represents allergy information
type AllergyInfo struct {
	AllergenCode string `json:"allergen_code"`
	AllergenName string `json:"allergen_name"`
	Severity     string `json:"severity"`
	Reaction     string `json:"reaction"`
	OnsetDate    *time.Time `json:"onset_date,omitempty"`
}

// ConditionInfo represents medical condition information
type ConditionInfo struct {
	ConditionCode string     `json:"condition_code"`
	ConditionName string     `json:"condition_name"`
	Severity      string     `json:"severity"`
	Status        string     `json:"status"`
	OnsetDate     *time.Time `json:"onset_date,omitempty"`
}

// Clinical Rule Evaluation Request
type ClinicalRuleEvaluationRequest struct {
	RequestID       string                 `json:"request_id"`
	PatientID       string                 `json:"patient_id"`
	RuleSet         string                 `json:"rule_set"` // drug_rules, guideline_evidence, safety_rules
	EvaluationContext map[string]interface{} `json:"evaluation_context"`
	RuleFilters     []string               `json:"rule_filters,omitempty"`
	Priority        string                 `json:"priority"`
}

// Clinical Engine Response Types

// DrugInteractionResponse represents the response from drug interaction analysis
type DrugInteractionResponse struct {
	RequestID       string               `json:"request_id"`
	PatientID       string               `json:"patient_id"`
	Interactions    []DrugInteraction    `json:"interactions"`
	OverallRisk     string               `json:"overall_risk"`
	ProcessingTime  time.Duration        `json:"processing_time"`
	Recommendations []InteractionRecommendation `json:"recommendations"`
	Success         bool                 `json:"success"`
	Error           string               `json:"error,omitempty"`
}

// Removed duplicate DrugInteraction type - defined in apollo_federation_types.go

// InteractionRecommendation represents a recommendation for handling interactions
type InteractionRecommendation struct {
	RecommendationID string    `json:"recommendation_id"`
	InteractionID    string    `json:"interaction_id"`
	Action           string    `json:"action"`
	Description      string    `json:"description"`
	Priority         string    `json:"priority"`
	Evidence         string    `json:"evidence"`
}

// DosageCalculationResponse represents the response from dosage calculations
type DosageCalculationResponse struct {
	RequestID           string                `json:"request_id"`
	PatientID           string                `json:"patient_id"`
	MedicationCode      string                `json:"medication_code"`
	RecommendedDosage   *DosageRecommendation `json:"recommended_dosage"`
	AlternativeDosages  []DosageRecommendation `json:"alternative_dosages"`
	CalculationDetails  *CalculationDetails   `json:"calculation_details"`
	SafetyConsiderations []SafetyNote         `json:"safety_considerations"`
	ProcessingTime      time.Duration         `json:"processing_time"`
	Success             bool                  `json:"success"`
	Error               string                `json:"error,omitempty"`
}

// DosageRecommendation represents a dosage recommendation
type DosageRecommendation struct {
	DosageID      string  `json:"dosage_id"`
	Amount        float64 `json:"amount"`
	Unit          string  `json:"unit"`
	Route         string  `json:"route"`
	Frequency     string  `json:"frequency"`
	Duration      string  `json:"duration"`
	Instructions  string  `json:"instructions"`
	Confidence    float64 `json:"confidence"`
	Rationale     string  `json:"rationale"`
}

// CalculationDetails represents calculation methodology details
type CalculationDetails struct {
	Formula            string                 `json:"formula"`
	InputParameters    map[string]interface{} `json:"input_parameters"`
	Adjustments        []DosageAdjustment     `json:"adjustments"`
	ReferenceStandards []string               `json:"reference_standards"`
}

// DosageAdjustment represents a dosage adjustment factor
type DosageAdjustment struct {
	AdjustmentType string  `json:"adjustment_type"`
	Factor         float64 `json:"factor"`
	Reason         string  `json:"reason"`
	Applied        bool    `json:"applied"`
}

// SafetyNote represents a safety consideration
type SafetyNote struct {
	NoteID      string `json:"note_id"`
	Category    string `json:"category"`
	Severity    string `json:"severity"`
	Description string `json:"description"`
	Action      string `json:"action"`
	Reference   string `json:"reference,omitempty"`
}

// SafetyValidationResponse represents the response from safety validation
type SafetyValidationResponse struct {
	RequestID           string            `json:"request_id"`
	PatientID           string            `json:"patient_id"`
	ValidationResult    string            `json:"validation_result"` // safe, caution, contraindicated
	SafetyAlerts        []SafetyAlert     `json:"safety_alerts"`
	ContraindicationChecks []ContraindicationCheck `json:"contraindication_checks"`
	AllergyChecks       []AllergyCheck    `json:"allergy_checks"`
	OverallRiskScore    float64           `json:"overall_risk_score"`
	ProcessingTime      time.Duration     `json:"processing_time"`
	Success             bool              `json:"success"`
	Error               string            `json:"error,omitempty"`
}

// SafetyAlert represents a safety alert
type SafetyAlert struct {
	AlertID       string `json:"alert_id"`
	Category      string `json:"category"`
	Severity      string `json:"severity"`
	Title         string `json:"title"`
	Description   string `json:"description"`
	Action        string `json:"action"`
	Reference     string `json:"reference"`
	Override      bool   `json:"override"`
}

// ContraindicationCheck represents a contraindication check result
type ContraindicationCheck struct {
	CheckID         string `json:"check_id"`
	ConditionCode   string `json:"condition_code"`
	ConditionName   string `json:"condition_name"`
	MedicationCode  string `json:"medication_code"`
	IsContraindicated bool `json:"is_contraindicated"`
	Severity        string `json:"severity"`
	Evidence        string `json:"evidence"`
}

// AllergyCheck represents an allergy check result
type AllergyCheck struct {
	CheckID        string `json:"check_id"`
	AllergenCode   string `json:"allergen_code"`
	AllergenName   string `json:"allergen_name"`
	MedicationCode string `json:"medication_code"`
	CrossReactivity bool  `json:"cross_reactivity"`
	RiskLevel      string `json:"risk_level"`
	Recommendation string `json:"recommendation"`
}

// ClinicalRuleEvaluationResponse represents the response from clinical rule evaluation
type ClinicalRuleEvaluationResponse struct {
	RequestID       string                    `json:"request_id"`
	PatientID       string                    `json:"patient_id"`
	RuleSet         string                    `json:"rule_set"`
	EvaluatedRules  []EvaluatedRule          `json:"evaluated_rules"`
	OverallScore    float64                   `json:"overall_score"`
	Recommendations []ClinicalRecommendation  `json:"recommendations"`
	ProcessingTime  time.Duration             `json:"processing_time"`
	Success         bool                      `json:"success"`
	Error           string                    `json:"error,omitempty"`
}

// EvaluatedRule represents an evaluated clinical rule
type EvaluatedRule struct {
	RuleID          string                 `json:"rule_id"`
	RuleName        string                 `json:"rule_name"`
	RuleType        string                 `json:"rule_type"`
	Triggered       bool                   `json:"triggered"`
	Score           float64                `json:"score"`
	Evidence        string                 `json:"evidence"`
	Parameters      map[string]interface{} `json:"parameters"`
	ExecutionTime   time.Duration          `json:"execution_time"`
}

// ClinicalRecommendation represents a clinical recommendation
type ClinicalRecommendation struct {
	RecommendationID string  `json:"recommendation_id"`
	Category         string  `json:"category"`
	Priority         string  `json:"priority"`
	Title            string  `json:"title"`
	Description      string  `json:"description"`
	Action           string  `json:"action"`
	Evidence         string  `json:"evidence"`
	Confidence       float64 `json:"confidence"`
	References       []string `json:"references"`
}

// Client Methods

// AnalyzeDrugInteractions performs drug interaction analysis via Rust engine
func (c *RustClinicalEngineClient) AnalyzeDrugInteractions(ctx context.Context, request *DrugInteractionRequest) (*DrugInteractionResponse, error) {
	startTime := time.Now()
	c.logger.Info("Starting drug interaction analysis",
		zap.String("request_id", request.RequestID),
		zap.String("patient_id", request.PatientID),
		zap.Int("medication_count", len(request.Medications)))

	response, err := c.makeRequest(ctx, "POST", "/api/v1/drug-interactions", request)
	if err != nil {
		return nil, fmt.Errorf("drug interaction analysis failed: %w", err)
	}

	var result DrugInteractionResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal drug interaction response: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	c.logger.Info("Drug interaction analysis completed",
		zap.String("request_id", request.RequestID),
		zap.Duration("processing_time", result.ProcessingTime),
		zap.Int("interactions_found", len(result.Interactions)))

	return &result, nil
}

// CalculateDosage performs dosage calculations via Rust engine
func (c *RustClinicalEngineClient) CalculateDosage(ctx context.Context, request *DosageCalculationRequest) (*DosageCalculationResponse, error) {
	startTime := time.Now()
	c.logger.Info("Starting dosage calculation",
		zap.String("request_id", request.RequestID),
		zap.String("patient_id", request.PatientID),
		zap.String("medication_code", request.MedicationCode))

	response, err := c.makeRequest(ctx, "POST", "/api/v1/dosage-calculation", request)
	if err != nil {
		return nil, fmt.Errorf("dosage calculation failed: %w", err)
	}

	var result DosageCalculationResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal dosage calculation response: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	c.logger.Info("Dosage calculation completed",
		zap.String("request_id", request.RequestID),
		zap.Duration("processing_time", result.ProcessingTime))

	return &result, nil
}

// ValidateSafety performs safety validation via Rust engine
func (c *RustClinicalEngineClient) ValidateSafety(ctx context.Context, request *SafetyValidationRequest) (*SafetyValidationResponse, error) {
	startTime := time.Now()
	c.logger.Info("Starting safety validation",
		zap.String("request_id", request.RequestID),
		zap.String("patient_id", request.PatientID),
		zap.String("validation_level", request.ValidationLevel))

	response, err := c.makeRequest(ctx, "POST", "/api/v1/safety-validation", request)
	if err != nil {
		return nil, fmt.Errorf("safety validation failed: %w", err)
	}

	var result SafetyValidationResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal safety validation response: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	c.logger.Info("Safety validation completed",
		zap.String("request_id", request.RequestID),
		zap.Duration("processing_time", result.ProcessingTime),
		zap.String("result", result.ValidationResult),
		zap.Float64("risk_score", result.OverallRiskScore))

	return &result, nil
}

// EvaluateRules performs clinical rule evaluation via Rust engine
func (c *RustClinicalEngineClient) EvaluateRules(ctx context.Context, request *ClinicalRuleEvaluationRequest) (*ClinicalRuleEvaluationResponse, error) {
	startTime := time.Now()
	c.logger.Info("Starting clinical rule evaluation",
		zap.String("request_id", request.RequestID),
		zap.String("patient_id", request.PatientID),
		zap.String("rule_set", request.RuleSet))

	response, err := c.makeRequest(ctx, "POST", "/api/v1/evaluate-rules", request)
	if err != nil {
		return nil, fmt.Errorf("clinical rule evaluation failed: %w", err)
	}

	var result ClinicalRuleEvaluationResponse
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule evaluation response: %w", err)
	}

	result.ProcessingTime = time.Since(startTime)
	c.logger.Info("Clinical rule evaluation completed",
		zap.String("request_id", request.RequestID),
		zap.Duration("processing_time", result.ProcessingTime),
		zap.Int("rules_evaluated", len(result.EvaluatedRules)),
		zap.Float64("overall_score", result.OverallScore))

	return &result, nil
}

// Health check for Rust engine
func (c *RustClinicalEngineClient) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	response, err := c.makeRequest(ctx, "GET", "/health", nil)
	if err != nil {
		return nil, fmt.Errorf("health check failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal health check response: %w", err)
	}

	return result, nil
}

// Metrics retrieval from Rust engine
func (c *RustClinicalEngineClient) GetMetrics(ctx context.Context) (map[string]interface{}, error) {
	response, err := c.makeRequest(ctx, "GET", "/metrics", nil)
	if err != nil {
		return nil, fmt.Errorf("metrics retrieval failed: %w", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(response, &result); err != nil {
		return nil, fmt.Errorf("failed to unmarshal metrics response: %w", err)
	}

	return result, nil
}

// Internal helper methods

func (c *RustClinicalEngineClient) makeRequest(ctx context.Context, method, endpoint string, payload interface{}) ([]byte, error) {
	url := c.baseURL + endpoint
	
	var body io.Reader
	if payload != nil {
		jsonData, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal request: %w", err)
		}
		body = bytes.NewBuffer(jsonData)
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetries; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(c.retryDelay * time.Duration(attempt)):
			}
		}

		req, err := http.NewRequestWithContext(ctx, method, url, body)
		if err != nil {
			lastErr = fmt.Errorf("failed to create request: %w", err)
			continue
		}

		if payload != nil {
			req.Header.Set("Content-Type", "application/json")
		}
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Request-ID", uuid.New().String())

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = fmt.Errorf("HTTP request failed: %w", err)
			continue
		}

		responseData, err := io.ReadAll(resp.Body)
		resp.Body.Close()

		if err != nil {
			lastErr = fmt.Errorf("failed to read response: %w", err)
			continue
		}

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return responseData, nil
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("server error (status %d): %s", resp.StatusCode, string(responseData))
			continue
		}

		return nil, fmt.Errorf("request failed (status %d): %s", resp.StatusCode, string(responseData))
	}

	return nil, fmt.Errorf("request failed after %d attempts: %w", c.maxRetries+1, lastErr)
}