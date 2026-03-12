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

// KB6FormularyClient provides HTTP client for KB-6 Formulary service
type KB6FormularyClient struct {
	baseURL    string
	httpClient *http.Client
	config     config.KBClientConfig
	log        *logrus.Entry
}

// FormularyCheckRequest represents a request to check formulary status
type FormularyCheckRequest struct {
	DrugCode     string `json:"drug_code"`     // RxNorm code
	DrugName     string `json:"drug_name"`
	FormularyID  string `json:"formulary_id,omitempty"`
	PayerID      string `json:"payer_id,omitempty"`
	PlanID       string `json:"plan_id,omitempty"`
	PatientID    string `json:"patient_id,omitempty"`
}

// FormularyCheckResponse represents formulary check result
type FormularyCheckResponse struct {
	Success         bool                `json:"success"`
	DrugCode        string              `json:"drug_code"`
	DrugName        string              `json:"drug_name"`
	FormularyStatus FormularyStatus     `json:"formulary_status"`
	Alternatives    []DrugAlternative   `json:"alternatives,omitempty"`
	PARequired      bool                `json:"pa_required"`
	StepTherapy     *StepTherapyInfo    `json:"step_therapy,omitempty"`
	QuantityLimits  *QuantityLimits     `json:"quantity_limits,omitempty"`
	CopayInfo       *CopayInfo          `json:"copay_info,omitempty"`
	ErrorMessage    string              `json:"error_message,omitempty"`
}

// FormularyStatus represents the formulary tier status
type FormularyStatus struct {
	OnFormulary   bool   `json:"on_formulary"`
	Tier          int    `json:"tier"`
	TierName      string `json:"tier_name"` // generic, preferred, non-preferred, specialty
	Restrictions  []string `json:"restrictions,omitempty"`
	EffectiveDate time.Time `json:"effective_date"`
	ExpiryDate    time.Time `json:"expiry_date,omitempty"`
}

// DrugAlternative represents a formulary alternative drug
type DrugAlternative struct {
	DrugCode       string  `json:"drug_code"`
	DrugName       string  `json:"drug_name"`
	GenericName    string  `json:"generic_name"`
	Tier           int     `json:"tier"`
	IsGeneric      bool    `json:"is_generic"`
	IsPreferred    bool    `json:"is_preferred"`
	TherapyClass   string  `json:"therapy_class"`
	EstimatedCopay float64 `json:"estimated_copay,omitempty"`
	PARequired     bool    `json:"pa_required"`
}

// StepTherapyInfo represents step therapy requirements
type StepTherapyInfo struct {
	Required       bool     `json:"required"`
	CurrentStep    int      `json:"current_step"`
	TotalSteps     int      `json:"total_steps"`
	RequiredDrugs  []string `json:"required_drugs"`
	MinDuration    int      `json:"min_duration_days"`
	Criteria       string   `json:"criteria"`
}

// QuantityLimits represents quantity limit information
type QuantityLimits struct {
	MaxQuantity       int    `json:"max_quantity"`
	MaxDaysSupply     int    `json:"max_days_supply"`
	QuantityPerFill   int    `json:"quantity_per_fill"`
	RefillsAllowed    int    `json:"refills_allowed"`
	RequiresOverride  bool   `json:"requires_override"`
}

// CopayInfo represents copay information
type CopayInfo struct {
	CopayAmount    float64 `json:"copay_amount"`
	CoinsurancePct float64 `json:"coinsurance_pct"`
	DeductibleApplies bool  `json:"deductible_applies"`
	OOPAccumulator float64 `json:"oop_accumulator"`
	OOPMax         float64 `json:"oop_max"`
}

// DrugInteractionRequest represents a drug-drug interaction check request
type DrugInteractionRequest struct {
	DrugCodes []string `json:"drug_codes"` // List of RxNorm codes
	PatientID string   `json:"patient_id,omitempty"`
}

// DrugInteractionResponse represents interaction check results
type DrugInteractionResponse struct {
	Success       bool               `json:"success"`
	Interactions  []DrugInteraction  `json:"interactions"`
	TotalChecked  int                `json:"total_checked"`
	ErrorMessage  string             `json:"error_message,omitempty"`
}

// DrugInteraction represents a detected drug-drug interaction
type DrugInteraction struct {
	Drug1Code     string `json:"drug1_code"`
	Drug1Name     string `json:"drug1_name"`
	Drug2Code     string `json:"drug2_code"`
	Drug2Name     string `json:"drug2_name"`
	Severity      string `json:"severity"` // severe, moderate, minor
	InteractionType string `json:"interaction_type"`
	Description   string `json:"description"`
	ClinicalEffect string `json:"clinical_effect"`
	Management    string `json:"management"`
	Reference     string `json:"reference"`
}

// NewKB6FormularyClient creates a new KB-6 Formulary HTTP client
func NewKB6FormularyClient(cfg config.KBClientConfig) *KB6FormularyClient {
	transport := &http.Transport{
		MaxIdleConns:        cfg.MaxIdleConns,
		MaxIdleConnsPerHost: cfg.MaxIdleConns,
		IdleConnTimeout:     cfg.IdleConnTimeout,
		DisableKeepAlives:   false,
	}

	return &KB6FormularyClient{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout:   cfg.Timeout,
			Transport: transport,
		},
		config: cfg,
		log:    logrus.WithField("client", "kb6-formulary"),
	}
}

// IsEnabled returns whether the KB-6 client is enabled
func (c *KB6FormularyClient) IsEnabled() bool {
	return c.config.Enabled
}

// Health checks if KB-6 service is healthy
func (c *KB6FormularyClient) Health(ctx context.Context) error {
	if !c.config.Enabled {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, "GET", c.baseURL+"/health", nil)
	if err != nil {
		return fmt.Errorf("failed to create health request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("KB-6 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-6 unhealthy: status %d", resp.StatusCode)
	}

	return nil
}

// CheckFormulary checks if a drug is on formulary
func (c *KB6FormularyClient) CheckFormulary(ctx context.Context, req *FormularyCheckRequest) (*FormularyCheckResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-6 client disabled, returning default formulary response")
		return &FormularyCheckResponse{
			Success:  true,
			DrugCode: req.DrugCode,
			DrugName: req.DrugName,
			FormularyStatus: FormularyStatus{
				OnFormulary: true,
				Tier:        2,
				TierName:    "preferred",
			},
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *FormularyCheckResponse
	err = c.doRequest(ctx, "POST", "/api/v1/formulary/check", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// GetAlternatives retrieves formulary alternatives for a drug
func (c *KB6FormularyClient) GetAlternatives(ctx context.Context, drugCode string, formularyID string) ([]DrugAlternative, error) {
	if !c.config.Enabled {
		return []DrugAlternative{}, nil
	}

	var resp struct {
		Alternatives []DrugAlternative `json:"alternatives"`
	}
	endpoint := fmt.Sprintf("/api/v1/formulary/alternatives?drug_code=%s&formulary_id=%s", drugCode, formularyID)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp.Alternatives, nil
}

// CheckInteractions checks for drug-drug interactions
func (c *KB6FormularyClient) CheckInteractions(ctx context.Context, req *DrugInteractionRequest) (*DrugInteractionResponse, error) {
	if !c.config.Enabled {
		c.log.Debug("KB-6 client disabled, returning empty interactions")
		return &DrugInteractionResponse{
			Success:      true,
			Interactions: []DrugInteraction{},
			TotalChecked: len(req.DrugCodes),
		}, nil
	}

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp *DrugInteractionResponse
	err = c.doRequest(ctx, "POST", "/api/v1/interactions/check", body, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// ValidatePriorAuth checks if prior authorization is required
func (c *KB6FormularyClient) ValidatePriorAuth(ctx context.Context, drugCode string, patientID string, payerID string) (bool, string, error) {
	if !c.config.Enabled {
		return false, "", nil
	}

	req := map[string]string{
		"drug_code":  drugCode,
		"patient_id": patientID,
		"payer_id":   payerID,
	}

	body, err := json.Marshal(req)
	if err != nil {
		return false, "", fmt.Errorf("failed to marshal request: %w", err)
	}

	var resp struct {
		PARequired bool   `json:"pa_required"`
		Reason     string `json:"reason,omitempty"`
	}
	err = c.doRequest(ctx, "POST", "/api/v1/formulary/prior-auth/check", body, &resp)
	if err != nil {
		return false, "", err
	}

	return resp.PARequired, resp.Reason, nil
}

// GetQuantityLimits retrieves quantity limits for a drug
func (c *KB6FormularyClient) GetQuantityLimits(ctx context.Context, drugCode string, formularyID string) (*QuantityLimits, error) {
	if !c.config.Enabled {
		return &QuantityLimits{
			MaxQuantity:     90,
			MaxDaysSupply:   30,
			QuantityPerFill: 30,
			RefillsAllowed:  3,
		}, nil
	}

	var resp *QuantityLimits
	endpoint := fmt.Sprintf("/api/v1/formulary/quantity-limits?drug_code=%s&formulary_id=%s", drugCode, formularyID)
	err := c.doRequest(ctx, "GET", endpoint, nil, &resp)
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// doRequest performs an HTTP request with retry logic
func (c *KB6FormularyClient) doRequest(ctx context.Context, method, endpoint string, body []byte, result interface{}) error {
	var lastErr error

	for attempt := 0; attempt <= c.config.MaxRetries; attempt++ {
		if attempt > 0 {
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
			c.log.WithError(err).WithField("attempt", attempt+1).Warn("KB-6 request failed, retrying")
			continue
		}
		defer resp.Body.Close()

		respBody, err := io.ReadAll(resp.Body)
		if err != nil {
			lastErr = fmt.Errorf("failed to read response body: %w", err)
			continue
		}

		if resp.StatusCode >= 500 {
			lastErr = fmt.Errorf("KB-6 server error: %d - %s", resp.StatusCode, string(respBody))
			c.log.WithField("status", resp.StatusCode).WithField("attempt", attempt+1).Warn("KB-6 server error, retrying")
			continue
		}

		if resp.StatusCode >= 400 {
			return fmt.Errorf("KB-6 client error: %d - %s", resp.StatusCode, string(respBody))
		}

		if result != nil {
			if err := json.Unmarshal(respBody, result); err != nil {
				return fmt.Errorf("failed to unmarshal response: %w", err)
			}
		}

		return nil
	}

	return fmt.Errorf("KB-6 request failed after %d retries: %w", c.config.MaxRetries+1, lastErr)
}
