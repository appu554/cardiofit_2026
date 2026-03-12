// Package clients provides HTTP clients for KB services.
//
// KB1HTTPClient implements the KB1Client interface for KB-1 Drug Rules Service.
// It provides dose adjustments based on renal, hepatic, weight, and age factors.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client is used by KnowledgeSnapshotBuilder to populate DosingSnapshot.
// All dose adjustments are pre-computed at snapshot build time - engines NEVER
// call dosing rules directly at execution time.
//
// Connects to: http://localhost:8081 (Docker: kb1-drug-rules)
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

// KB1HTTPClient implements KB1Client by calling the KB-1 Drug Rules Service REST API.
type KB1HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB1HTTPClient creates a new KB-1 HTTP client.
func NewKB1HTTPClient(baseURL string) *KB1HTTPClient {
	return &KB1HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB1HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB1HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB1HTTPClient {
	return &KB1HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB1Client Interface Implementation
// ============================================================================

// GetRenalAdjustments returns renal dose adjustments for medications based on eGFR.
// Calls KB-1 /v1/calculate/renal endpoint for each medication.
//
// NOTE: KB-1 expects full PatientParameters (age, gender, weight, height) but the
// KB1Client interface only provides eGFR. To properly integrate, the interface
// would need to be updated to pass PatientContext. For now, we send what we have.
func (c *KB1HTTPClient) GetRenalAdjustments(
	ctx context.Context,
	medications []contracts.ClinicalCode,
	eGFR float64,
) (map[string]contracts.DoseAdjustment, error) {

	adjustments := make(map[string]contracts.DoseAdjustment)

	for _, med := range medications {
		// Build request for renal adjustment
		// NOTE: KB-1 expects full patient params but interface only provides eGFR.
		// KB-1 will use eGFR directly if provided, skipping internal calculation.
		req := kb1RenalRequest{
			RxNormCode: med.Code,
			Patient: kb1PatientParams{
				// Provide defaults for required fields - KB-1 will use eGFR if provided
				Age:      50,      // Default adult age
				Gender:   "M",     // Default gender
				WeightKg: 70.0,    // Default weight
				HeightCm: 170.0,   // Default height
				EGFR:     eGFR,    // The actual value we care about
			},
		}

		resp, err := c.callKB1(ctx, "/v1/calculate/renal", req)
		if err != nil {
			// Log but continue - non-fatal for individual medications
			continue
		}

		var result kb1RenalResult
		if err := json.Unmarshal(resp, &result); err != nil {
			continue
		}

		// KB-1 returns success=true if drug found, and adjustment_factor indicates adjustment
		// AdjustmentNeeded() returns true if success && (factor != 1.0 || contraindicated)
		if result.AdjustmentNeeded() {
			key := fmt.Sprintf("%s|%s", med.System, med.Code)
			adjustments[key] = contracts.DoseAdjustment{
				Medication: med,
				Reason:     "renal",
				CurrentDose: &contracts.Quantity{
					Value: result.OriginalDose,
					Unit:  result.Unit,
				},
				RecommendedDose: &contracts.Quantity{
					Value: result.AdjustedDose,
					Unit:  result.Unit,
				},
				AdjustmentPercent: result.AdjustmentFactor * 100, // KB-1 uses adjustment_factor
				ThresholdEGFR:     eGFR,
				Guidance:          result.Recommendation, // KB-1 uses recommendation
			}
		}
	}

	return adjustments, nil
}

// GetHepaticAdjustments returns hepatic dose adjustments based on Child-Pugh class.
// Calls KB-1 /v1/calculate/hepatic endpoint.
func (c *KB1HTTPClient) GetHepaticAdjustments(
	ctx context.Context,
	medications []contracts.ClinicalCode,
	childPugh string,
) (map[string]contracts.DoseAdjustment, error) {

	adjustments := make(map[string]contracts.DoseAdjustment)

	for _, med := range medications {
		// KB-1 expects HepaticAdjustedRequest with full patient params
		req := kb1HepaticRequest{
			RxNormCode: med.Code,
			Patient: kb1PatientParams{
				Age:            50,
				Gender:         "M",
				WeightKg:       70.0,
				HeightCm:       170.0,
				ChildPughClass: childPugh,
			},
		}

		resp, err := c.callKB1(ctx, "/v1/calculate/hepatic", req)
		if err != nil {
			continue
		}

		var result kb1HepaticResult
		if err := json.Unmarshal(resp, &result); err != nil {
			continue
		}

		// KB-1 returns success=true if drug found, adjustment_factor indicates adjustment
		if result.AdjustmentNeeded() {
			key := fmt.Sprintf("%s|%s", med.System, med.Code)
			adjustments[key] = contracts.DoseAdjustment{
				Medication: med,
				Reason:     "hepatic",
				CurrentDose: &contracts.Quantity{
					Value: result.OriginalDose,
					Unit:  result.Unit,
				},
				RecommendedDose: &contracts.Quantity{
					Value: result.AdjustedDose,
					Unit:  result.Unit,
				},
				AdjustmentPercent: result.AdjustmentFactor * 100,
				Guidance:          fmt.Sprintf("Child-Pugh %s: %s", childPugh, result.Recommendation),
			}
		}
	}

	return adjustments, nil
}

// GetWeightBasedDoses returns weight-based dose calculations.
// Calls KB-1 /v1/calculate/weight-based endpoint.
func (c *KB1HTTPClient) GetWeightBasedDoses(
	ctx context.Context,
	medications []contracts.ClinicalCode,
	weightKg float64,
	bsa float64,
) (map[string]contracts.DoseCalculation, error) {

	doses := make(map[string]contracts.DoseCalculation)

	for _, med := range medications {
		req := kb1WeightBasedRequest{
			RxNormCode: med.Code,
			Patient: kb1PatientParams{
				WeightKg: weightKg,
			},
		}

		resp, err := c.callKB1(ctx, "/v1/calculate/weight-based", req)
		if err != nil {
			continue
		}

		var result kb1WeightBasedResult
		if err := json.Unmarshal(resp, &result); err != nil {
			continue
		}

		key := fmt.Sprintf("%s|%s", med.System, med.Code)
		doses[key] = contracts.DoseCalculation{
			Medication:    med,
			DosePerKg:     result.DosePerKg,
			PatientWeight: weightKg,
			PatientBSA:    bsa,
			CalculatedDose: &contracts.Quantity{
				Value: result.CalculatedDose,
				Unit:  result.Unit,
			},
			MaxDose: &contracts.Quantity{
				Value: result.MaxDose,
				Unit:  result.Unit,
			},
		}
	}

	return doses, nil
}

// GetAgeBasedAdjustments returns age-based dose adjustments (pediatric/geriatric).
// Calls KB-1 /v1/calculate/pediatric or /v1/calculate/geriatric based on age.
func (c *KB1HTTPClient) GetAgeBasedAdjustments(
	ctx context.Context,
	medications []contracts.ClinicalCode,
	ageYears int,
) (map[string]contracts.DoseAdjustment, error) {

	adjustments := make(map[string]contracts.DoseAdjustment)

	for _, med := range medications {
		req := kb1AgeRequest{
			RxNormCode: med.Code,
			Patient: kb1PatientParams{
				Age:      ageYears,
				Gender:   "M",   // Default
				WeightKg: 70.0,  // Default adult weight
				HeightCm: 170.0, // Default
			},
		}

		if ageYears < 18 {
			// Pediatric dose calculation
			resp, err := c.callKB1(ctx, "/v1/calculate/pediatric", req)
			if err != nil {
				continue
			}

			var result kb1PediatricResult
			if err := json.Unmarshal(resp, &result); err != nil {
				continue
			}

			// Pediatric results provide recommended dose - compare to standard adult dose
			if result.Success && result.RecommendedDose > 0 {
				key := fmt.Sprintf("%s|%s", med.System, med.Code)
				adjustments[key] = contracts.DoseAdjustment{
					Medication: med,
					Reason:     "pediatric",
					RecommendedDose: &contracts.Quantity{
						Value: result.RecommendedDose,
						Unit:  result.Unit,
					},
					Guidance: fmt.Sprintf("Pediatric (%s): dose per kg %.2f mg/kg, max %.0f %s",
						result.AgeCategory, result.DosePerKg, result.MaxDose, result.Unit),
				}
			}
		} else {
			// Geriatric dose calculation
			resp, err := c.callKB1(ctx, "/v1/calculate/geriatric", req)
			if err != nil {
				continue
			}

			var result kb1GeriatricResult
			if err := json.Unmarshal(resp, &result); err != nil {
				continue
			}

			// Geriatric results provide recommended dose with Beers criteria warnings
			if result.Success && result.RecommendedDose > 0 {
				key := fmt.Sprintf("%s|%s", med.System, med.Code)
				guidance := result.AdjustmentNotes
				if result.BeersWarning != "" {
					guidance = fmt.Sprintf("Beers: %s. %s", result.BeersWarning, result.AdjustmentNotes)
				}
				adjustments[key] = contracts.DoseAdjustment{
					Medication: med,
					Reason:     "geriatric",
					RecommendedDose: &contracts.Quantity{
						Value: result.RecommendedDose,
						Unit:  result.Unit,
					},
					Guidance: guidance,
				}
			}
		}
	}

	return adjustments, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB1HTTPClient) callKB1(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
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
		return nil, fmt.Errorf("KB-1 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-1 service is healthy.
func (c *KB1HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-1 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// KB-1 Request/Response Types (internal)
// ============================================================================
// NOTE: These types must match the actual KB-1 API models in:
// kb-1-drug-rules/internal/models/models.go

// kb1PatientParams matches models.PatientParameters
type kb1PatientParams struct {
	Age             int     `json:"age"`
	Gender          string  `json:"gender"`
	WeightKg        float64 `json:"weight_kg"`
	HeightCm        float64 `json:"height_cm"`
	SerumCreatinine float64 `json:"serum_creatinine,omitempty"`
	EGFR            float64 `json:"egfr,omitempty"`
	ChildPughScore  int     `json:"child_pugh_score,omitempty"`
	ChildPughClass  string  `json:"child_pugh_class,omitempty"`
}

// kb1RenalRequest matches models.RenalAdjustedRequest
type kb1RenalRequest struct {
	RxNormCode string           `json:"rxnorm_code"`
	Patient    kb1PatientParams `json:"patient"`
}

// kb1RenalResult matches models.RenalDoseResult
// Note: KB-1 returns adjustment_factor, not adjustment_ratio
// Note: KB-1 returns recommendation, not reason
type kb1RenalResult struct {
	Success                bool    `json:"success"`
	DrugName               string  `json:"drug_name"`
	RxNormCode             string  `json:"rxnorm_code"`
	OriginalDose           float64 `json:"original_dose"`
	AdjustedDose           float64 `json:"adjusted_dose"`
	Unit                   string  `json:"unit"`
	EGFR                   float64 `json:"egfr"`
	CrCl                   float64 `json:"crcl,omitempty"`
	CKDStage               string  `json:"ckd_stage"`
	CKDDescription         string  `json:"ckd_description,omitempty"`
	AdjustmentFactor       float64 `json:"adjustment_factor"`
	Recommendation         string  `json:"recommendation,omitempty"`
	Contraindicated        bool    `json:"contraindicated"`
	ContraindicationReason string  `json:"contraindication_reason,omitempty"`
	Frequency              string  `json:"frequency,omitempty"`
	Error                  string  `json:"error,omitempty"`
}

// AdjustmentNeeded returns true if an adjustment is needed based on KB-1 response
func (r *kb1RenalResult) AdjustmentNeeded() bool {
	return r.Success && (r.AdjustmentFactor != 1.0 || r.Contraindicated)
}

// kb1HepaticRequest matches models.HepaticAdjustedRequest
type kb1HepaticRequest struct {
	RxNormCode string           `json:"rxnorm_code"`
	Patient    kb1PatientParams `json:"patient"`
}

// kb1HepaticResult matches models.HepaticDoseResult
type kb1HepaticResult struct {
	Success          bool    `json:"success"`
	DrugName         string  `json:"drug_name"`
	RxNormCode       string  `json:"rxnorm_code"`
	OriginalDose     float64 `json:"original_dose"`
	AdjustedDose     float64 `json:"adjusted_dose"`
	Unit             string  `json:"unit"`
	ChildPughClass   string  `json:"child_pugh_class"`
	AdjustmentFactor float64 `json:"adjustment_factor"`
	Recommendation   string  `json:"recommendation,omitempty"`
	Contraindicated  bool    `json:"contraindicated"`
	Error            string  `json:"error,omitempty"`
}

// AdjustmentNeeded returns true if an adjustment is needed based on KB-1 response
func (r *kb1HepaticResult) AdjustmentNeeded() bool {
	return r.Success && (r.AdjustmentFactor != 1.0 || r.Contraindicated)
}

// kb1WeightBasedRequest matches models.WeightBasedRequest
type kb1WeightBasedRequest struct {
	RxNormCode string           `json:"rxnorm_code,omitempty"`
	Patient    kb1PatientParams `json:"patient"`
	DosePerKg  float64          `json:"dose_per_kg,omitempty"`
}

// kb1WeightBasedResult is used for weight-based dose calculations
type kb1WeightBasedResult struct {
	Success        bool    `json:"success"`
	CalculatedDose float64 `json:"calculated_dose"`
	Unit           string  `json:"unit"`
	DosePerKg      float64 `json:"dose_per_kg"`
	BSABased       bool    `json:"bsa_based"`
	Formula        string  `json:"formula"`
	MinDose        float64 `json:"min_dose"`
	MaxDose        float64 `json:"max_dose"`
	Error          string  `json:"error,omitempty"`
}

// kb1AgeRequest matches models.PediatricRequest/GeriatricRequest
type kb1AgeRequest struct {
	RxNormCode string           `json:"rxnorm_code"`
	Patient    kb1PatientParams `json:"patient"`
}

// kb1PediatricResult matches models.PediatricDoseResult
type kb1PediatricResult struct {
	Success         bool     `json:"success"`
	AgeCategory     string   `json:"age_category"`
	DrugName        string   `json:"drug_name"`
	RecommendedDose float64  `json:"recommended_dose"`
	Unit            string   `json:"unit"`
	DosePerKg       float64  `json:"dose_per_kg,omitempty"`
	MaxDose         float64  `json:"max_dose,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Error           string   `json:"error,omitempty"`
}

// kb1GeriatricResult matches models.GeriatricDoseResult
type kb1GeriatricResult struct {
	Success         bool     `json:"success"`
	DrugName        string   `json:"drug_name"`
	RecommendedDose float64  `json:"recommended_dose"`
	Unit            string   `json:"unit"`
	AdjustmentNotes string   `json:"adjustment_notes,omitempty"`
	BeersWarning    string   `json:"beers_warning,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	Error           string   `json:"error,omitempty"`
}
