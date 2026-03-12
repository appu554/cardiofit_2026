package contextrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// =============================================================================
// KB-16 Lab Interpretation Service Client
// =============================================================================
// This client connects Context Router to KB-16 for LOINC reference range lookups.
// KB-16 queries the loinc_reference_ranges table in the shared canonical_facts DB
// which contains 6041 LOINC codes with reference ranges.
//
// Architecture:
//   Context Router → KB-16 API → canonical_facts DB → loinc_reference_ranges
//
// Use Cases:
//   1. Validate patient lab values against reference ranges
//   2. Get critical/panic thresholds for DDI context evaluation
//   3. Enrich PatientContext with LOINC metadata
//   4. Get DDI-relevant LOINC codes for threshold validation
// =============================================================================

// KB16Client provides access to KB-16 Lab Interpretation Service
type KB16Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
	enabled    bool
}

// KB16Config holds configuration for KB-16 client
type KB16Config struct {
	BaseURL string        `json:"base_url"`
	Timeout time.Duration `json:"timeout"`
	Enabled bool          `json:"enabled"`
}

// DefaultKB16Config returns default configuration
func DefaultKB16Config() KB16Config {
	return KB16Config{
		BaseURL: "http://localhost:8095",
		Timeout: 10 * time.Second,
		Enabled: true,
	}
}

// NewKB16Client creates a new KB-16 client
func NewKB16Client(config KB16Config, logger *zap.Logger) *KB16Client {
	if logger == nil {
		logger, _ = zap.NewProduction()
	}

	return &KB16Client{
		baseURL: config.BaseURL,
		httpClient: &http.Client{
			Timeout: config.Timeout,
		},
		logger:  logger.With(zap.String("client", "kb16")),
		enabled: config.Enabled,
	}
}

// =============================================================================
// Response Models
// =============================================================================

// LOINCReferenceRange represents a reference range from KB-16
// Field names match 006_expanded_loinc_reference_ranges.sql migration columns
type LOINCReferenceRange struct {
	LOINCCode              string   `json:"loinc_code"`
	Component              string   `json:"component"`
	LongName               string   `json:"long_name"`
	Unit                   string   `json:"unit"`
	LowNormal              *float64 `json:"low_normal"`
	HighNormal             *float64 `json:"high_normal"`
	CriticalLow            *float64 `json:"critical_low"`
	CriticalHigh           *float64 `json:"critical_high"`
	ClinicalCategory       string   `json:"clinical_category"`
	AgeGroup               string   `json:"age_group"`
	Sex                    string   `json:"sex"`
	InterpretationGuidance string   `json:"interpretation_guidance,omitempty"`
}

// LOINCStats represents repository statistics
type LOINCStats struct {
	TotalCodes       int      `json:"total_codes"`
	DDIRelevantCodes int      `json:"ddi_relevant_codes"`
	CategoryCount    int      `json:"category_count"`
	Categories       []string `json:"categories"`
}

// KB16APIResponse is the standard KB-16 API response wrapper
type KB16APIResponse struct {
	Success bool            `json:"success"`
	Data    json.RawMessage `json:"data,omitempty"`
	Error   *KB16APIError   `json:"error,omitempty"`
	Meta    *KB16APIMeta    `json:"meta,omitempty"`
}

// KB16APIError represents an API error
type KB16APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// KB16APIMeta contains response metadata
type KB16APIMeta struct {
	Total int `json:"total,omitempty"`
}

// =============================================================================
// API Methods
// =============================================================================

// GetReferenceRange retrieves reference range for a LOINC code
func (c *KB16Client) GetReferenceRange(ctx context.Context, loincCode string) (*LOINCReferenceRange, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/loinc/reference-ranges/%s", c.baseURL, loincCode)
	resp, err := c.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to get LOINC reference range: %w", err)
	}

	if !resp.Success {
		if resp.Error != nil && resp.Error.Code == "NOT_FOUND" {
			return nil, nil // Not found is not an error
		}
		return nil, fmt.Errorf("KB-16 error: %s", resp.Error.Message)
	}

	var ref LOINCReferenceRange
	if err := json.Unmarshal(resp.Data, &ref); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reference range: %w", err)
	}

	return &ref, nil
}

// GetReferenceRangeWithContext retrieves reference range with age/sex specificity
func (c *KB16Client) GetReferenceRangeWithContext(ctx context.Context, loincCode string, age int, sex string) (*LOINCReferenceRange, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/loinc/reference-ranges/%s/context?age=%d&sex=%s", c.baseURL, loincCode, age, sex)
	resp, err := c.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to get LOINC reference range with context: %w", err)
	}

	if !resp.Success {
		if resp.Error != nil && resp.Error.Code == "NOT_FOUND" {
			return nil, nil
		}
		return nil, fmt.Errorf("KB-16 error: %s", resp.Error.Message)
	}

	var ref LOINCReferenceRange
	if err := json.Unmarshal(resp.Data, &ref); err != nil {
		return nil, fmt.Errorf("failed to unmarshal reference range: %w", err)
	}

	return &ref, nil
}

// GetDDIRelevantRanges retrieves all DDI-relevant LOINC reference ranges
func (c *KB16Client) GetDDIRelevantRanges(ctx context.Context) ([]LOINCReferenceRange, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/loinc/ddi-relevant", c.baseURL)
	resp, err := c.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to get DDI-relevant LOINC ranges: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-16 error: %s", resp.Error.Message)
	}

	var refs []LOINCReferenceRange
	if err := json.Unmarshal(resp.Data, &refs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal DDI-relevant ranges: %w", err)
	}

	return refs, nil
}

// GetStats retrieves LOINC repository statistics
func (c *KB16Client) GetStats(ctx context.Context) (*LOINCStats, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/loinc/stats", c.baseURL)
	resp, err := c.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to get LOINC stats: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-16 error: %s", resp.Error.Message)
	}

	var stats LOINCStats
	if err := json.Unmarshal(resp.Data, &stats); err != nil {
		return nil, fmt.Errorf("failed to unmarshal LOINC stats: %w", err)
	}

	return &stats, nil
}

// SearchByName searches LOINC codes by component name
func (c *KB16Client) SearchByName(ctx context.Context, query string, limit int) ([]LOINCReferenceRange, error) {
	if !c.enabled {
		return nil, nil
	}

	url := fmt.Sprintf("%s/api/v1/loinc/search?q=%s&limit=%d", c.baseURL, query, limit)
	resp, err := c.doRequest(ctx, "GET", url)
	if err != nil {
		return nil, fmt.Errorf("failed to search LOINC: %w", err)
	}

	if !resp.Success {
		return nil, fmt.Errorf("KB-16 error: %s", resp.Error.Message)
	}

	var refs []LOINCReferenceRange
	if err := json.Unmarshal(resp.Data, &refs); err != nil {
		return nil, fmt.Errorf("failed to unmarshal search results: %w", err)
	}

	return refs, nil
}

// Health checks KB-16 service health
func (c *KB16Client) Health(ctx context.Context) (bool, error) {
	if !c.enabled {
		return false, nil
	}

	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return false, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, err
	}
	defer resp.Body.Close()

	return resp.StatusCode == http.StatusOK, nil
}

// =============================================================================
// Context Router Integration Methods
// =============================================================================

// ValidateLabValue validates a patient lab value against KB-16 reference ranges
// Returns validation result indicating if the value is within normal, critical, or panic ranges
func (c *KB16Client) ValidateLabValue(ctx context.Context, loincCode string, value float64) (*LabValidationResult, error) {
	ref, err := c.GetReferenceRange(ctx, loincCode)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return &LabValidationResult{
			LOINCCode:    loincCode,
			Value:        value,
			Status:       "UNKNOWN",
			Message:      "No reference range found for LOINC code",
			HasReference: false,
		}, nil
	}

	result := &LabValidationResult{
		LOINCCode:     loincCode,
		Value:         value,
		Unit:          ref.Unit,
		HasReference:  true,
		LowNormal:     ref.LowNormal,
		HighNormal:    ref.HighNormal,
		CriticalLow:   ref.CriticalLow,
		CriticalHigh:  ref.CriticalHigh,
		Component:     ref.Component,
	}

	// Check panic/critical values first
	if ref.CriticalLow != nil && value < *ref.CriticalLow {
		result.Status = "CRITICAL_LOW"
		result.Message = fmt.Sprintf("%s %.2f is below critical low threshold %.2f",
			ref.Component, value, *ref.CriticalLow)
		result.IsCritical = true
		return result, nil
	}
	if ref.CriticalHigh != nil && value > *ref.CriticalHigh {
		result.Status = "CRITICAL_HIGH"
		result.Message = fmt.Sprintf("%s %.2f exceeds critical high threshold %.2f",
			ref.Component, value, *ref.CriticalHigh)
		result.IsCritical = true
		return result, nil
	}

	// Check normal ranges
	if ref.LowNormal != nil && value < *ref.LowNormal {
		result.Status = "LOW"
		result.Message = fmt.Sprintf("%s %.2f is below normal range (%.2f - %.2f)",
			ref.Component, value, *ref.LowNormal, safeFloat(ref.HighNormal))
		return result, nil
	}
	if ref.HighNormal != nil && value > *ref.HighNormal {
		result.Status = "HIGH"
		result.Message = fmt.Sprintf("%s %.2f exceeds normal range (%.2f - %.2f)",
			ref.Component, value, safeFloat(ref.LowNormal), *ref.HighNormal)
		return result, nil
	}

	result.Status = "NORMAL"
	result.Message = fmt.Sprintf("%s %.2f is within normal range", ref.Component, value)
	return result, nil
}

// EnrichPatientContext validates all labs in patient context against KB-16 reference ranges
// Adds validation results to each lab value
func (c *KB16Client) EnrichPatientContext(ctx context.Context, patientCtx *PatientContext) (*EnrichedPatientContext, error) {
	enriched := &EnrichedPatientContext{
		PatientContext:    patientCtx,
		LabValidations:    make(map[string]*LabValidationResult),
		CriticalLabCount:  0,
		AbnormalLabCount:  0,
		NormalLabCount:    0,
		UnknownLabCount:   0,
	}

	if patientCtx.Labs == nil {
		return enriched, nil
	}

	for loincCode, labValue := range patientCtx.Labs {
		validation, err := c.ValidateLabValue(ctx, loincCode, labValue.Value)
		if err != nil {
			c.logger.Warn("Failed to validate lab value",
				zap.String("loinc_code", loincCode),
				zap.Error(err))
			continue
		}

		enriched.LabValidations[loincCode] = validation

		switch validation.Status {
		case "CRITICAL_LOW", "CRITICAL_HIGH":
			enriched.CriticalLabCount++
			enriched.AbnormalLabCount++
		case "LOW", "HIGH":
			enriched.AbnormalLabCount++
		case "NORMAL":
			enriched.NormalLabCount++
		default:
			enriched.UnknownLabCount++
		}
	}

	return enriched, nil
}

// GetThresholdForDDI retrieves the recommended threshold for a specific DDI context evaluation
// Derives thresholds from critical ranges in the LOINC reference data
// This can be used to override or validate thresholds in DDIProjection
func (c *KB16Client) GetThresholdForDDI(ctx context.Context, loincCode string) (*DDIThreshold, error) {
	ref, err := c.GetReferenceRange(ctx, loincCode)
	if err != nil {
		return nil, err
	}
	if ref == nil {
		return nil, nil
	}

	// Check if this is a DDI-relevant category (electrolyte, renal, hepatic, coagulation, cardiac)
	isDDIRelevant := ref.ClinicalCategory == "electrolyte" ||
		ref.ClinicalCategory == "renal" ||
		ref.ClinicalCategory == "hepatic" ||
		ref.ClinicalCategory == "coagulation" ||
		ref.ClinicalCategory == "cardiac"

	// Derive threshold from critical ranges
	if ref.CriticalHigh != nil {
		return &DDIThreshold{
			LOINCCode:   loincCode,
			Threshold:   *ref.CriticalHigh,
			Operator:    ">",
			Source:      "KB16_CRITICAL_HIGH",
			DDIRelevant: isDDIRelevant,
		}, nil
	}
	if ref.CriticalLow != nil {
		return &DDIThreshold{
			LOINCCode:   loincCode,
			Threshold:   *ref.CriticalLow,
			Operator:    "<",
			Source:      "KB16_CRITICAL_LOW",
			DDIRelevant: isDDIRelevant,
		}, nil
	}

	// Fall back to normal range boundaries for DDI evaluation
	if isDDIRelevant {
		if ref.HighNormal != nil {
			return &DDIThreshold{
				LOINCCode:   loincCode,
				Threshold:   *ref.HighNormal,
				Operator:    ">",
				Source:      "KB16_HIGH_NORMAL",
				DDIRelevant: true,
			}, nil
		}
		if ref.LowNormal != nil {
			return &DDIThreshold{
				LOINCCode:   loincCode,
				Threshold:   *ref.LowNormal,
				Operator:    "<",
				Source:      "KB16_LOW_NORMAL",
				DDIRelevant: true,
			}, nil
		}
	}

	return nil, nil
}

// =============================================================================
// Helper Types
// =============================================================================

// LabValidationResult contains the validation result for a lab value
type LabValidationResult struct {
	LOINCCode    string   `json:"loinc_code"`
	Component    string   `json:"component"`
	Value        float64  `json:"value"`
	Unit         string   `json:"unit"`
	Status       string   `json:"status"` // NORMAL, LOW, HIGH, CRITICAL_LOW, CRITICAL_HIGH, UNKNOWN
	Message      string   `json:"message"`
	HasReference bool     `json:"has_reference"`
	IsCritical   bool     `json:"is_critical"`
	LowNormal    *float64 `json:"low_normal,omitempty"`
	HighNormal   *float64 `json:"high_normal,omitempty"`
	CriticalLow  *float64 `json:"critical_low,omitempty"`
	CriticalHigh *float64 `json:"critical_high,omitempty"`
}

// EnrichedPatientContext contains patient context with lab validations
type EnrichedPatientContext struct {
	*PatientContext
	LabValidations   map[string]*LabValidationResult `json:"lab_validations"`
	CriticalLabCount int                             `json:"critical_lab_count"`
	AbnormalLabCount int                             `json:"abnormal_lab_count"`
	NormalLabCount   int                             `json:"normal_lab_count"`
	UnknownLabCount  int                             `json:"unknown_lab_count"`
}

// DDIThreshold contains threshold information for DDI evaluation
type DDIThreshold struct {
	LOINCCode   string  `json:"loinc_code"`
	Threshold   float64 `json:"threshold"`
	Operator    string  `json:"operator"`
	Source      string  `json:"source"`       // KB16_CRITICAL_HIGH, KB16_CRITICAL_LOW, KB16_HIGH_NORMAL, KB16_LOW_NORMAL
	DDIRelevant bool    `json:"ddi_relevant"` // True if from DDI-relevant clinical category
}

// =============================================================================
// Internal Methods
// =============================================================================

func (c *KB16Client) doRequest(ctx context.Context, method, url string) (*KB16APIResponse, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Service-Client", "context-router")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Error("HTTP request failed",
			zap.String("url", url),
			zap.Error(err))
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var apiResp KB16APIResponse
	if err := json.Unmarshal(body, &apiResp); err != nil {
		// Try to parse as raw data if not wrapped
		apiResp.Success = resp.StatusCode >= 200 && resp.StatusCode < 300
		apiResp.Data = body
	}

	return &apiResp, nil
}

func safeFloat(ptr *float64) float64 {
	if ptr == nil {
		return 0
	}
	return *ptr
}
