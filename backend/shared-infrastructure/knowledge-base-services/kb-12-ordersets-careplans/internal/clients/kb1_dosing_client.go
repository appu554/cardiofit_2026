// Package clients provides HTTP clients for KB service integrations
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"kb-12-ordersets-careplans/internal/config"
)

// KB1DosingClient provides HTTP client for KB-1 Drug Rules/Dosing service
type KB1DosingClient struct {
	baseURL    string
	httpClient *http.Client
	config     config.KBClientConfig
	log        *logrus.Entry
}

// PatientParameters represents patient data for KB-1 API (used in various requests)
type PatientParameters struct {
	Age             int     `json:"age"`
	Gender          string  `json:"gender"` // M, F, male, female
	WeightKg        float64 `json:"weight_kg"`
	HeightCm        float64 `json:"height_cm"`
	SerumCreatinine float64 `json:"serum_creatinine,omitempty"` // in mg/dL
	EGFR            float64 `json:"egfr,omitempty"`             // pre-calculated if available
	ChildPughScore  int     `json:"child_pugh_score,omitempty"` // 5-15 for hepatic adjustment
}

// DoseCalculationRequest represents a request to calculate drug dosing
// NOTE: KB-1 actual implementation uses flat structure (not nested patient object)
type DoseCalculationRequest struct {
	RxNormCode      string  `json:"rxnorm_code"`               // RxNorm code
	Age             int     `json:"age"`                       // Patient age
	Gender          string  `json:"gender"`                    // M, F
	WeightKg        float64 `json:"weight_kg"`                 // Weight in kg
	HeightCm        float64 `json:"height_cm"`                 // Height in cm
	SerumCreatinine float64 `json:"serum_creatinine,omitempty"` // in mg/dL
	Indication      string  `json:"indication,omitempty"`      // Clinical indication
	// Helper field for backward compatibility with nested patient requests
	Patient PatientParameters `json:"-"` // Ignored in JSON, used internally
}

// DoseCalculationRequestLegacy provides backward compatibility for callers using old format
type DoseCalculationRequestLegacy struct {
	DrugCode     string            `json:"drug_code"`     // RxNorm code
	DrugName     string            `json:"drug_name"`     // Drug name
	PatientID    string            `json:"patient_id"`    // Patient identifier
	Weight       float64           `json:"weight_kg"`     // Weight in kg
	WeightUnit   string            `json:"weight_unit"`   // kg or lb
	Height       float64           `json:"height_cm"`     // Height in cm
	Age          int               `json:"age_years"`     // Age in years
	Gender       string            `json:"gender"`        // male/female
	BSA          float64           `json:"bsa_m2"`        // Body surface area
	CrCl         float64           `json:"crcl_ml_min"`   // Creatinine clearance
	Indication   string            `json:"indication"`    // Clinical indication
	Route        string            `json:"route"`         // Administration route
	Frequency    string            `json:"frequency"`     // Dosing frequency
	RenalStatus  string            `json:"renal_status"`  // normal, mild, moderate, severe, esrd
	HepaticStatus string           `json:"hepatic_status"`// normal, mild, moderate, severe
	Parameters   map[string]interface{} `json:"parameters,omitempty"` // Additional parameters
}

// DoseCalculationResponse represents the calculated dose response from KB-1
// NOTE: This matches the actual KB-1 /v1/calculate response format
type DoseCalculationResponse struct {
	Success           bool              `json:"success"`
	DrugCode          string            `json:"drug_code,omitempty"`         // Legacy field
	RxNormCode        string            `json:"rxnorm_code,omitempty"`       // Actual KB-1 field
	DrugName          string            `json:"drug_name"`
	RecommendedDose   float64           `json:"recommended_dose"`            // Simple number from KB-1
	DoseUnit          string            `json:"dose_unit"`
	Frequency         string            `json:"frequency"`
	DosingMethod      string            `json:"dosing_method,omitempty"`
	CalculationBasis  string            `json:"calculation_basis,omitempty"`
	RenalAdjustment   *RenalAdjustment  `json:"renal_adjustment,omitempty"`
	PatientParameters map[string]interface{} `json:"patient_parameters,omitempty"`
	MonitoringRequired []string         `json:"monitoring_required,omitempty"`
	Adjustments       []DoseAdjustment  `json:"adjustments,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
	Contraindications []string          `json:"contraindications,omitempty"`
	RuleSource        string            `json:"rule_source,omitempty"`
	CalculatedAt      time.Time         `json:"calculated_at,omitempty"`
	ErrorMessage      string            `json:"error_message,omitempty"`
}

// RenalAdjustment represents renal dosing adjustments from KB-1
type RenalAdjustment struct {
	Applied    bool    `json:"applied"`
	Reason     string  `json:"reason,omitempty"`
	Multiplier float64 `json:"multiplier,omitempty"`
	MaxDoseCap float64 `json:"max_dose_cap,omitempty"`
	Notes      string  `json:"notes,omitempty"`
}

// DoseRecommendation represents recommended dosing
type DoseRecommendation struct {
	Dose         float64 `json:"dose"`
	Unit         string  `json:"unit"`         // mg, mcg, units, etc.
	DosePerKg    float64 `json:"dose_per_kg,omitempty"`
	DosePerBSA   float64 `json:"dose_per_bsa,omitempty"`
	MinDose      float64 `json:"min_dose,omitempty"`
	MaxDose      float64 `json:"max_dose,omitempty"`
	Frequency    string  `json:"frequency"`    // q4h, q6h, daily, etc.
	Route        string  `json:"route"`        // IV, PO, IM, etc.
	Duration     string  `json:"duration,omitempty"`
	Instructions string  `json:"instructions,omitempty"`
}

// DoseAdjustment represents a dose modification
type DoseAdjustment struct {
	Reason         string  `json:"reason"`
	AdjustmentType string  `json:"adjustment_type"` // increase, decrease, contraindicated
	Factor         float64 `json:"factor,omitempty"`
	NewDose        float64 `json:"new_dose,omitempty"`
	Unit           string  `json:"unit,omitempty"`
	Reference      string  `json:"reference"`
}

// DrugRuleRequest represents a request to get drug rules
type DrugRuleRequest struct {
	DrugCode  string `json:"drug_code"`
	RuleType  string `json:"rule_type,omitempty"` // dosing, monitoring, interaction
	Condition string `json:"condition,omitempty"`
}

// DrugRuleResponse represents drug rule information
type DrugRuleResponse struct {
	Success    bool       `json:"success"`
	DrugCode   string     `json:"drug_code"`
	DrugName   string     `json:"drug_name"`
	Rules      []DrugRule `json:"rules"`
	ErrorMessage string   `json:"error_message,omitempty"`
}

// DrugRule represents an individual drug rule
type DrugRule struct {
	RuleID       string            `json:"rule_id"`
	RuleType     string            `json:"rule_type"`
	Description  string            `json:"description"`
	Conditions   []RuleCondition   `json:"conditions"`
	Actions      []RuleAction      `json:"actions"`
	Severity     string            `json:"severity"`
	Reference    string            `json:"reference"`
	EffectiveDate time.Time        `json:"effective_date"`
	ExpiryDate   time.Time         `json:"expiry_date,omitempty"`
	Active       bool              `json:"active"`
}

// RuleCondition represents a condition for rule application
type RuleCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

// RuleAction represents an action when rule conditions are met
type RuleAction struct {
	ActionType string      `json:"action_type"`
	Target     string      `json:"target"`
	Value      interface{} `json:"value"`
}

// NewKB1DosingClient creates a new KB-1 Dosing HTTP client
func NewKB1DosingClient(cfg config.KBClientConfig) *KB1DosingClient {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableKeepAlives:   false,
	}

	return &KB1DosingClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		config: cfg,
		log:    logrus.WithField("client", "kb1-dosing"),
	}
}

// IsEnabled returns whether the KB-1 client is enabled
func (c *KB1DosingClient) IsEnabled() bool {
	return c.config.Enabled
}

// Health checks if KB-1 service is healthy
func (c *KB1DosingClient) Health(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-1 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-1 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// CalculateDose calculates drug dosing based on patient parameters
func (c *KB1DosingClient) CalculateDose(ctx context.Context, req *DoseCalculationRequest) (*DoseCalculationResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-1 client disabled, skipping dose calculation")
		return &DoseCalculationResponse{
			Success:      true,
			DrugCode:     req.RxNormCode,
			CalculatedAt: time.Now(),
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *DoseCalculationResponse
	err = c.doRequest(ctx, "POST", "/v1/calculate", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CalculateDoseLegacy provides backward compatibility for callers using old flat format
func (c *KB1DosingClient) CalculateDoseLegacy(ctx context.Context, req *DoseCalculationRequestLegacy) (*DoseCalculationResponse, error) {
	// Convert legacy format to new KB-1 format
	newReq := &DoseCalculationRequest{
		RxNormCode: req.DrugCode,
		Patient: PatientParameters{
			Age:             req.Age,
			Gender:          req.Gender,
			WeightKg:        req.Weight,
			HeightCm:        req.Height,
			SerumCreatinine: 1.0, // Default if not provided
		},
		Indication: req.Indication,
	}
	return c.CalculateDose(ctx, newReq)
}

// GetDrugRules retrieves drug rules for a specific medication
func (c *KB1DosingClient) GetDrugRules(ctx context.Context, req *DrugRuleRequest) (*DrugRuleResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-1 client disabled, skipping drug rules lookup")
		return &DrugRuleResponse{
			Success:  true,
			DrugCode: req.DrugCode,
			Rules:    []DrugRule{},
		}, nil
	}

	var resp *DrugRuleResponse
	// Use GET with RxNorm code in path
	endpoint := fmt.Sprintf("/v1/rules/%s", req.DrugCode)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ValidateDoseRange validates if a dose is within acceptable range
// Uses GET /v1/validate/max-dose to get limits, then validates locally
func (c *KB1DosingClient) ValidateDoseRange(ctx context.Context, drugCode string, dose float64, unit string, route string) (bool, []string, error) {
	if !c.config.Enabled {
		return true, nil, nil
	}

	// Use the simpler max-dose endpoint that doesn't require patient info
	var resp struct {
		Success       bool    `json:"success"`
		DrugName      string  `json:"drug_name"`
		MaxSingleDose float64 `json:"max_single_dose"`
		MaxDailyDose  float64 `json:"max_daily_dose"`
		Unit          string  `json:"unit"`
	}

	endpoint := fmt.Sprintf("/v1/validate/max-dose?rxnorm=%s", drugCode)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return false, nil, err
	}

	// Validate the proposed dose against max limits
	var warnings []string
	valid := true

	if resp.MaxSingleDose > 0 && dose > resp.MaxSingleDose {
		valid = false
		warnings = append(warnings, fmt.Sprintf("Dose %.1f%s exceeds maximum single dose of %.1f%s for %s",
			dose, unit, resp.MaxSingleDose, resp.Unit, resp.DrugName))
	} else if resp.MaxSingleDose > 0 && dose > resp.MaxSingleDose*0.8 {
		warnings = append(warnings, fmt.Sprintf("Dose %.1f%s is near maximum single dose of %.1f%s for %s",
			dose, unit, resp.MaxSingleDose, resp.Unit, resp.DrugName))
	}

	return valid, warnings, nil
}

// WeightBasedDosingRequest represents a weight-based dosing request (KB-1 format)
type WeightBasedDosingRequest struct {
	RxNormCode string            `json:"rxnorm_code,omitempty"`
	Patient    PatientParameters `json:"patient"`
	DosePerKg  float64           `json:"dose_per_kg,omitempty"`
}

// GetWeightBasedDosing calculates weight-based dosing for a drug
func (c *KB1DosingClient) GetWeightBasedDosing(ctx context.Context, drugCode string, indication string) (*WeightBasedDosingResponse, error) {
	if !c.config.Enabled {
		return &WeightBasedDosingResponse{Success: true, DrugCode: drugCode}, nil
	}

	// KB-1 /v1/calculate/weight-based is POST and needs patient info
	// For simple lookups without patient, use /v1/rules/{rxnorm} instead
	var resp *WeightBasedDosingResponse
	endpoint := fmt.Sprintf("/v1/rules/%s", drugCode)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CalculateWeightBasedDose calculates weight-based dosing with full patient parameters
func (c *KB1DosingClient) CalculateWeightBasedDose(ctx context.Context, req *WeightBasedDosingRequest) (*DoseCalculationResponse, error) {
	if !c.config.Enabled {
		return &DoseCalculationResponse{Success: true, DrugCode: req.RxNormCode}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *DoseCalculationResponse
	err = c.doRequest(ctx, "POST", "/v1/calculate/weight-based", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// WeightBasedDosingResponse represents weight-based dosing parameters
type WeightBasedDosingResponse struct {
	Success       bool    `json:"success"`
	DrugCode      string  `json:"drug_code"`
	DrugName      string  `json:"drug_name"`
	Indication    string  `json:"indication"`
	DosePerKg     float64 `json:"dose_per_kg"`
	DoseUnit      string  `json:"dose_unit"`
	MinDose       float64 `json:"min_dose"`
	MaxDose       float64 `json:"max_dose"`
	Frequency     string  `json:"frequency"`
	Route         string  `json:"route"`
	AgeRestrictions string `json:"age_restrictions,omitempty"`
	Reference     string  `json:"reference"`
	ErrorMessage  string  `json:"error_message,omitempty"`
}

// doRequest performs an HTTP request with retry logic
func (c *KB1DosingClient) doRequest(ctx context.Context, method, endpoint string, body []byte, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
			// Exponential backoff
			waitTime := c.config.RetryWaitMin * time.Duration(1<<uint(attempt-1))
			if waitTime > c.config.RetryWaitMax {
				waitTime = c.config.RetryWaitMax
			}
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(waitTime):
			}
		}

		var req *http.Request
		var err error

		if body != nil {
			req, err = http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, bytes.NewReader(body))
		} else {
			req, err = http.NewRequestWithContext(ctx, method, c.baseURL+endpoint, nil)
		}
		if err != nil {
			return fmt.Errorf("failed to create request: %w", err)
		}

		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("X-Client-Service", "kb-12-ordersets")

		resp, err := c.httpClient.Do(req)
		if err != nil {
			lastErr = err
			c.log.WithError(err).WithField("attempt", attempt+1).Warn("KB-1 request failed, retrying")
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("KB-1 server error: %d - %s", resp.StatusCode, string(respBody))
			c.log.WithField("status", resp.StatusCode).WithField("attempt", attempt+1).Warn("KB-1 server error, retrying")
			continue
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("KB-1 client error: %d - %s", resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("KB-1 request failed after %d retries: %w", c.config.MaxRetries+1, lastErr)
}

// InteractionCheckRequest represents a request to check drug interactions
type InteractionCheckRequest struct {
	DrugCodes     []string `json:"drug_codes"`     // List of RxNorm codes
	PatientID     string   `json:"patient_id"`     // Patient identifier
	IncludeSevere bool     `json:"include_severe"` // Include severe interactions only
}

// InteractionCheckResponse represents drug interaction check results
type InteractionCheckResponse struct {
	Success      bool          `json:"success"`
	Interactions []Interaction `json:"interactions"`
	ErrorMessage string        `json:"error_message,omitempty"`
}

// Interaction represents a drug interaction
type Interaction struct {
	Drug1       string `json:"drug_1"`
	Drug2       string `json:"drug_2"`
	Severity    string `json:"severity"`    // severe, moderate, minor
	Description string `json:"description"`
	Mechanism   string `json:"mechanism,omitempty"`
	Reference   string `json:"reference"`
}

// CheckInteraction checks for drug-drug interactions
func (c *KB1DosingClient) CheckInteraction(ctx context.Context, req *InteractionCheckRequest) (*InteractionCheckResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-1 client disabled, skipping interaction check")
		return &InteractionCheckResponse{
			Success:      true,
			Interactions: []Interaction{},
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *InteractionCheckResponse
	err = c.doRequest(ctx, "POST", "/v1/validate/interactions", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// DoseValidationRequest represents a dose validation request
// NOTE: KB-1 actual implementation uses flat structure
type DoseValidationRequest struct {
	RxNormCode   string  `json:"rxnorm_code"`
	ProposedDose float64 `json:"proposed_dose"`
	Unit         string  `json:"unit,omitempty"` // defaults to "mg"
	Age          int     `json:"age"`
	Gender       string  `json:"gender"`
	WeightKg     float64 `json:"weight_kg"`
	HeightCm     float64 `json:"height_cm"`
}

// DoseValidationRequestLegacy provides backward compatibility
type DoseValidationRequestLegacy struct {
	DrugCode  string  `json:"drug_code"`
	DrugName  string  `json:"drug_name"`
	Dose      float64 `json:"dose"`
	DoseUnit  string  `json:"dose_unit"`
	Route     string  `json:"route"`
	Frequency string  `json:"frequency"`
	PatientID string  `json:"patient_id,omitempty"`
	Weight    float64 `json:"weight_kg,omitempty"`
	Age       int     `json:"age_years,omitempty"`
}

// DoseValidationResponse represents dose validation results
type DoseValidationResponse struct {
	Success          bool     `json:"success"`
	Valid            bool     `json:"valid"`
	DrugName         string   `json:"drug_name,omitempty"`
	ProposedDose     float64  `json:"proposed_dose,omitempty"`
	RecommendedDose  float64  `json:"recommended_dose,omitempty"`
	MaxSingleDose    float64  `json:"max_single_dose,omitempty"`
	MaxDailyDose     float64  `json:"max_daily_dose,omitempty"`
	ValidationStatus string   `json:"validation_status,omitempty"` // safe, caution, warning, contraindicated
	Warnings         []string `json:"warnings,omitempty"`
	Errors           []string `json:"errors,omitempty"`
	Reasons          []string `json:"reasons,omitempty"`
	RecommendedMin   float64  `json:"recommended_min,omitempty"`
	RecommendedMax   float64  `json:"recommended_max,omitempty"`
	Reference        string   `json:"reference,omitempty"`
	ErrorMessage     string   `json:"error_message,omitempty"`
}

// ValidateDose validates a drug dose against clinical guidelines
func (c *KB1DosingClient) ValidateDose(ctx context.Context, req *DoseValidationRequest) (*DoseValidationResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-1 client disabled, skipping dose validation")
		return &DoseValidationResponse{
			Success: true,
			Valid:   true,
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *DoseValidationResponse
	err = c.doRequest(ctx, "POST", "/v1/validate/dose", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ValidateDoseLegacy provides backward compatibility for callers using old format
func (c *KB1DosingClient) ValidateDoseLegacy(ctx context.Context, req *DoseValidationRequestLegacy) (*DoseValidationResponse, error) {
	newReq := &DoseValidationRequest{
		RxNormCode:   req.DrugCode,
		ProposedDose: req.Dose,
		Unit:         req.DoseUnit,
		Age:          req.Age,
		WeightKg:     req.Weight,
		Gender:       "M",  // Default if not provided
		HeightCm:     170,  // Default if not provided
	}
	return c.ValidateDose(ctx, newReq)
}

// RenalDosingRequest represents a renal dosing check request
// NOTE: KB-1 actual implementation uses flat structure
type RenalDosingRequest struct {
	RxNormCode      string  `json:"rxnorm_code"`
	Age             int     `json:"age"`
	Gender          string  `json:"gender"`
	WeightKg        float64 `json:"weight_kg"`
	HeightCm        float64 `json:"height_cm"`
	SerumCreatinine float64 `json:"serum_creatinine,omitempty"`
	EGFR            float64 `json:"egfr,omitempty"`
}

// RenalDosingRequestLegacy provides backward compatibility
type RenalDosingRequestLegacy struct {
	DrugCode    string  `json:"drug_code"`
	DrugName    string  `json:"drug_name"`
	CrCl        float64 `json:"crcl_ml_min"` // Creatinine clearance
	GFR         float64 `json:"gfr,omitempty"`
	CurrentDose float64 `json:"current_dose"`
	DoseUnit    string  `json:"dose_unit"`
	Route       string  `json:"route"`
	Frequency   string  `json:"frequency"`
}

// RenalDosingResponse represents renal dosing adjustment results
type RenalDosingResponse struct {
	Success              bool     `json:"success"`
	DrugName             string   `json:"drug_name,omitempty"`
	OriginalDose         float64  `json:"original_dose,omitempty"`
	AdjustedDose         float64  `json:"adjusted_dose,omitempty"`
	Unit                 string   `json:"unit,omitempty"`
	EGFR                 float64  `json:"egfr,omitempty"`
	CrCl                 float64  `json:"crcl,omitempty"`
	CKDStage             string   `json:"ckd_stage,omitempty"` // G1-G5
	AdjustmentFactor     float64  `json:"adjustment_factor,omitempty"`
	Recommendation       string   `json:"recommendation,omitempty"`
	Contraindicated      bool     `json:"contraindicated"`
	ContraindicationReason string `json:"contraindication_reason,omitempty"`
	RequiresAdjust       bool     `json:"requires_adjustment"`
	RenalStage           string   `json:"renal_stage"` // normal, mild, moderate, severe, esrd
	AdjustedFrequency    string   `json:"adjusted_frequency,omitempty"`
	ReductionPercent     float64  `json:"reduction_percent,omitempty"`
	Warnings             []string `json:"warnings,omitempty"`
	Reference            string   `json:"reference"`
	ErrorMessage         string   `json:"error_message,omitempty"`
}

// CheckRenalDosing checks if renal dose adjustment is needed
func (c *KB1DosingClient) CheckRenalDosing(ctx context.Context, req *RenalDosingRequest) (*RenalDosingResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-1 client disabled, skipping renal dosing check")
		return &RenalDosingResponse{
			Success:        true,
			RequiresAdjust: false,
			RenalStage:     "unknown",
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *RenalDosingResponse
	err = c.doRequest(ctx, "POST", "/v1/calculate/renal", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// CheckRenalDosingLegacy provides backward compatibility for callers using old format
func (c *KB1DosingClient) CheckRenalDosingLegacy(ctx context.Context, req *RenalDosingRequestLegacy) (*RenalDosingResponse, error) {
	newReq := &RenalDosingRequest{
		RxNormCode:      req.DrugCode,
		Age:             65,   // Default
		Gender:          "M",  // Default
		WeightKg:        70,   // Default
		HeightCm:        170,  // Default
		SerumCreatinine: 1.0,  // Default
		EGFR:            req.GFR,
	}
	return c.CheckRenalDosing(ctx, newReq)
}
