// Package kb4 provides a client for KB-4 Patient Safety Service integration.
// KB-4 performs comprehensive safety checks on medication orders before dispensing.
//
// Integration Flow:
//
//	KB-1 (Dose Calculation) → KB-4 (Safety Check) → Final Dose Response
//
// KB-4 Checks Performed:
//   - Black Box Warnings (FDA/TGA)
//   - Contraindications (patient-specific)
//   - Dose Limits (max single/daily)
//   - Age Limits (pediatric/geriatric)
//   - Pregnancy Safety (FDA categories)
//   - Lactation Safety (LactMed)
//   - High-Alert Status (ISMP)
//   - Beers Criteria (geriatric PIMs)
//   - Anticholinergic Burden (ACB scale)
//   - Lab Requirements (monitoring)
package kb4

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"
)

// =============================================================================
// CLIENT CONFIGURATION
// =============================================================================

// Client provides access to KB-4 Patient Safety Service
type Client struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
	enabled    bool
}

// Config holds KB-4 client configuration
type Config struct {
	BaseURL    string
	Timeout    time.Duration
	MaxRetries int
	RetryDelay time.Duration
	Enabled    bool
}

// DefaultConfig returns default KB-4 client configuration
func DefaultConfig() Config {
	return Config{
		BaseURL:    "http://localhost:8088",
		Timeout:    30 * time.Second,
		MaxRetries: 3,
		RetryDelay: 500 * time.Millisecond,
		Enabled:    true,
	}
}

// NewClient creates a new KB-4 patient safety client
func NewClient(baseURL string, log *logrus.Entry) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		log:     log.WithField("component", "kb4-client"),
		enabled: true,
	}
}

// NewClientWithConfig creates a client with custom configuration
func NewClientWithConfig(cfg Config, log *logrus.Entry) *Client {
	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: cfg.Timeout,
		},
		log:     log.WithField("component", "kb4-client"),
		enabled: cfg.Enabled,
	}
}

// IsEnabled returns whether KB-4 integration is enabled
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// =============================================================================
// HEALTH CHECK
// =============================================================================

// Health checks KB-4 service health
func (c *Client) Health(ctx context.Context) error {
	if !c.enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-4 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// =============================================================================
// DATA TYPES - Mirrors KB-4 safety/types.go
// =============================================================================

// Severity levels for safety alerts
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityModerate Severity = "MODERATE"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

// AlertType categorizes safety alerts
type AlertType string

const (
	AlertTypeBlackBox         AlertType = "BLACK_BOX_WARNING"
	AlertTypeContraindication AlertType = "CONTRAINDICATION"
	AlertTypeAgeLimit         AlertType = "AGE_LIMIT"
	AlertTypeDoseLimit        AlertType = "DOSE_LIMIT"
	AlertTypePregnancy        AlertType = "PREGNANCY"
	AlertTypeLactation        AlertType = "LACTATION"
	AlertTypeHighAlert        AlertType = "HIGH_ALERT"
	AlertTypeBeers            AlertType = "BEERS_CRITERIA"
	AlertTypeAnticholinergic  AlertType = "ANTICHOLINERGIC"
	AlertTypeLabRequired      AlertType = "LAB_REQUIRED"
)

// DrugInfo identifies a medication for safety checking
type DrugInfo struct {
	RxNormCode string `json:"rxnormCode"`
	DrugName   string `json:"drugName,omitempty"`
	DrugClass  string `json:"drugClass,omitempty"`
	ATCCode    string `json:"atcCode,omitempty"`
}

// PatientContext provides patient-specific information for safety evaluation
type PatientContext struct {
	Age                  float64  `json:"age,omitempty"`
	AgeUnit              string   `json:"ageUnit,omitempty"` // years, months, days
	WeightKg             float64  `json:"weightKg,omitempty"`
	HeightCm             float64  `json:"heightCm,omitempty"`
	Gender               string   `json:"gender,omitempty"`
	IsPregnant           bool     `json:"isPregnant,omitempty"`
	PregnancyTrimester   int      `json:"pregnancyTrimester,omitempty"`
	IsLactating          bool     `json:"isLactating,omitempty"`
	CreatinineClearance  float64  `json:"creatinineClearance,omitempty"`  // mL/min
	EGFR                 float64  `json:"eGFR,omitempty"`                 // mL/min/1.73m²
	ChildPughScore       string   `json:"childPughScore,omitempty"`       // A, B, C
	Conditions           []string `json:"conditions,omitempty"`           // ICD-10 codes
	CurrentMedications   []string `json:"currentMedications,omitempty"`   // RxNorm codes
	Allergies            []string `json:"allergies,omitempty"`            // RxNorm codes
}

// SafetyCheckRequest is the input for KB-4 comprehensive safety check
type SafetyCheckRequest struct {
	Drug         DrugInfo       `json:"drug"`
	ProposedDose float64        `json:"proposedDose,omitempty"`
	DoseUnit     string         `json:"doseUnit,omitempty"`
	Frequency    string         `json:"frequency,omitempty"`
	Route        string         `json:"route,omitempty"`
	Patient      PatientContext `json:"patient"`
	CheckTypes   []AlertType    `json:"checkTypes,omitempty"` // Empty = all checks
}

// SafetyAlert represents an individual safety finding
type SafetyAlert struct {
	ID                     string    `json:"id,omitempty"`
	Type                   AlertType `json:"type"`
	Severity               Severity  `json:"severity"`
	Title                  string    `json:"title"`
	Message                string    `json:"message"`
	RequiresAcknowledgment bool      `json:"requiresAcknowledgment"`
	CanOverride            bool      `json:"canOverride"`
	ClinicalRationale      string    `json:"clinicalRationale,omitempty"`
	Recommendations        []string  `json:"recommendations,omitempty"`
	References             []string  `json:"references,omitempty"`
	DrugInfo               *DrugInfo `json:"drugInfo,omitempty"`
	CreatedAt              time.Time `json:"createdAt,omitempty"`
}

// SafetyCheckResponse is the result of KB-4 safety evaluation
type SafetyCheckResponse struct {
	Safe                       bool          `json:"safe"`
	RequiresAction             bool          `json:"requiresAction"`
	BlockPrescribing           bool          `json:"blockPrescribing"`
	CriticalAlerts             int           `json:"criticalAlerts"`
	HighAlerts                 int           `json:"highAlerts"`
	ModerateAlerts             int           `json:"moderateAlerts"`
	LowAlerts                  int           `json:"lowAlerts"`
	TotalAlerts                int           `json:"totalAlerts"`
	Alerts                     []SafetyAlert `json:"alerts"`
	IsHighAlertDrug            bool          `json:"isHighAlertDrug"`
	AnticholinergicBurdenTotal int           `json:"anticholinergicBurdenTotal,omitempty"`
	CheckedAt                  time.Time     `json:"checkedAt"`
	RequestID                  string        `json:"requestId,omitempty"`
}

// SafetyVerdict provides a summary for inclusion in dose calculation responses
type SafetyVerdict struct {
	Safe             bool          `json:"safe"`
	BlockPrescribing bool          `json:"blockPrescribing"`
	RequiresAction   bool          `json:"requiresAction"`
	IsHighAlertDrug  bool          `json:"isHighAlertDrug"`
	TotalAlerts      int           `json:"totalAlerts"`
	CriticalAlerts   int           `json:"criticalAlerts"`
	HighAlerts       int           `json:"highAlerts"`
	Alerts           []SafetyAlert `json:"alerts,omitempty"`
	CheckedAt        time.Time     `json:"checkedAt"`
	KB4RequestID     string        `json:"kb4RequestId,omitempty"`
}

// =============================================================================
// SAFETY CHECK METHODS
// =============================================================================

// Check performs a comprehensive safety check via KB-4
func (c *Client) Check(ctx context.Context, req *SafetyCheckRequest) (*SafetyCheckResponse, error) {
	if !c.enabled {
		c.log.Debug("KB-4 integration disabled, skipping safety check")
		return &SafetyCheckResponse{
			Safe:      true,
			CheckedAt: time.Now(),
		}, nil
	}

	if req.Drug.RxNormCode == "" && req.Drug.DrugName == "" {
		return nil, fmt.Errorf("either rxnormCode or drugName is required")
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.baseURL + "/v1/check"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	c.log.WithFields(logrus.Fields{
		"rxnorm_code": req.Drug.RxNormCode,
		"drug_name":   req.Drug.DrugName,
		"patient_age": req.Patient.Age,
	}).Debug("Calling KB-4 safety check")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.WithError(err).Warn("KB-4 safety check request failed")
		return nil, fmt.Errorf("safety check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		c.log.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(bodyBytes),
		}).Warn("KB-4 safety check returned non-OK status")
		return nil, fmt.Errorf("KB-4 returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result SafetyCheckResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"safe":              result.Safe,
		"block_prescribing": result.BlockPrescribing,
		"total_alerts":      result.TotalAlerts,
		"critical_alerts":   result.CriticalAlerts,
	}).Debug("KB-4 safety check completed")

	return &result, nil
}

// CheckComprehensive performs all safety checks with detailed reporting
func (c *Client) CheckComprehensive(ctx context.Context, req *SafetyCheckRequest) (*SafetyCheckResponse, error) {
	if !c.enabled {
		return &SafetyCheckResponse{Safe: true, CheckedAt: time.Now()}, nil
	}

	// Force all check types
	req.CheckTypes = nil

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.baseURL + "/v1/check/comprehensive"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("comprehensive check request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-4 returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	// Comprehensive endpoint wraps response in additional metadata
	var wrapper struct {
		SafetyCheck SafetyCheckResponse `json:"safetyCheck"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &wrapper.SafetyCheck, nil
}

// ToVerdict converts a full SafetyCheckResponse to a summary SafetyVerdict
func (r *SafetyCheckResponse) ToVerdict() SafetyVerdict {
	return SafetyVerdict{
		Safe:             r.Safe,
		BlockPrescribing: r.BlockPrescribing,
		RequiresAction:   r.RequiresAction,
		IsHighAlertDrug:  r.IsHighAlertDrug,
		TotalAlerts:      r.TotalAlerts,
		CriticalAlerts:   r.CriticalAlerts,
		HighAlerts:       r.HighAlerts,
		Alerts:           r.Alerts,
		CheckedAt:        r.CheckedAt,
		KB4RequestID:     r.RequestID,
	}
}

// =============================================================================
// INDIVIDUAL SAFETY LOOKUPS
// =============================================================================

// GetBlackBoxWarning retrieves black box warning for a drug
func (c *Client) GetBlackBoxWarning(ctx context.Context, rxnormCode string) (*SafetyAlert, error) {
	if !c.enabled {
		return nil, nil
	}

	endpoint := fmt.Sprintf("%s/v1/blackbox?rxnorm=%s", c.baseURL, rxnormCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil // No black box warning
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result SafetyAlert
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// IsHighAlertDrug checks if a drug is on the ISMP high-alert list
func (c *Client) IsHighAlertDrug(ctx context.Context, rxnormCode string) (bool, error) {
	if !c.enabled {
		return false, nil
	}

	endpoint := fmt.Sprintf("%s/v1/high-alert?rxnorm=%s", c.baseURL, rxnormCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return false, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return false, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return false, nil
	}

	if resp.StatusCode != http.StatusOK {
		return false, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result struct {
		IsHighAlert bool `json:"isHighAlert"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return false, fmt.Errorf("failed to decode response: %w", err)
	}

	return result.IsHighAlert, nil
}

// GetPregnancySafety retrieves pregnancy safety information for a drug
func (c *Client) GetPregnancySafety(ctx context.Context, rxnormCode string) (*SafetyAlert, error) {
	if !c.enabled {
		return nil, nil
	}

	endpoint := fmt.Sprintf("%s/v1/pregnancy?rxnorm=%s", c.baseURL, rxnormCode)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status: %d", resp.StatusCode)
	}

	var result SafetyAlert
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}

// =============================================================================
// DOSE VALIDATION
// =============================================================================

// DoseValidationRequest for validating proposed doses
type DoseValidationRequest struct {
	Drug         DrugInfo       `json:"drug"`
	ProposedDose float64        `json:"proposedDose"`
	DoseUnit     string         `json:"doseUnit"`
	Patient      PatientContext `json:"patient"`
}

// DoseValidationResponse from KB-4 dose limit validation
type DoseValidationResponse struct {
	IsValid       bool    `json:"isValid"`
	ExceedsSingle bool    `json:"exceedsSingle"`
	ExceedsDaily  bool    `json:"exceedsDaily"`
	MaxAllowed    float64 `json:"maxAllowed,omitempty"`
	Message       string  `json:"message,omitempty"`
}

// ValidateDose checks if a proposed dose exceeds safety limits
func (c *Client) ValidateDose(ctx context.Context, req *DoseValidationRequest) (*DoseValidationResponse, error) {
	if !c.enabled {
		return &DoseValidationResponse{IsValid: true}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	endpoint := c.baseURL + "/v1/limits/validate"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("dose validation request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-4 returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result DoseValidationResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &result, nil
}
