// Package clients provides HTTP clients for KB services.
//
// KB4HTTPClient implements the KB4Client interface for KB-4 Patient Safety Service.
// It provides comprehensive medication safety checking including:
// - Black box warnings (FDA/TGA/EMA)
// - Contraindications (drug-condition)
// - Dose limits (single, daily, population-specific)
// - Pregnancy safety (FDA PLLR, TGA categories)
// - Lactation safety (LactMed)
// - High-alert medications (ISMP)
// - Beers criteria (AGS geriatric)
// - Anticholinergic burden (ACB scale)
// - Lab monitoring requirements
// - STOPP/START criteria (European geriatric)
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client is used by KnowledgeSnapshotBuilder to populate SafetySnapshot.
// All safety checks are pre-computed at snapshot build time - engines NEVER
// call safety services directly at execution time.
//
// KB-4 API Endpoints Used:
// - POST /v1/check                  - Comprehensive safety check
// - POST /v1/check/comprehensive    - All safety checks combined
// - GET  /v1/blackbox               - Black box warnings
// - GET  /v1/high-alert             - ISMP high-alert status
// - GET  /v1/pregnancy              - Pregnancy safety info
// - GET  /v1/lactation              - Lactation safety info
// - GET  /v1/beers                  - Beers criteria
// - GET  /v1/anticholinergic        - ACB score
// - GET  /v1/labs                   - Lab requirements
// - POST /v1/limits/validate        - Dose limit validation
// - POST /v1/contraindications/check - Contraindication check
//
// Connects to: http://localhost:8088 (Docker: kb4-patient-safety-service)
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

// KB4HTTPClient implements KB4Client by calling the KB-4 Patient Safety Service REST API.
type KB4HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB4HTTPClient creates a new KB-4 HTTP client.
func NewKB4HTTPClient(baseURL string) *KB4HTTPClient {
	return &KB4HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB4HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB4HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB4HTTPClient {
	return &KB4HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB4Client Interface Implementation
// ============================================================================

// GetActiveAllergies returns allergy information for the patient.
// Calls KB-4 /api/v1/safety/allergies endpoint.
func (c *KB4HTTPClient) GetActiveAllergies(
	ctx context.Context,
	patient *contracts.PatientContext,
) ([]contracts.AllergyInfo, error) {

	req := kb4AllergyRequest{
		PatientID: patient.Demographics.PatientID,
	}

	resp, err := c.callKB4(ctx, "/api/v1/safety/allergies", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get allergies: %w", err)
	}

	var result kb4AllergyResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse allergy response: %w", err)
	}

	allergies := make([]contracts.AllergyInfo, 0, len(result.Allergies))
	for _, a := range result.Allergies {
		allergies = append(allergies, contracts.AllergyInfo{
			Allergen: contracts.ClinicalCode{
				System:  a.AllergenSystem,
				Code:    a.AllergenCode,
				Display: a.AllergenDisplay,
			},
			Category:    a.Category,
			Criticality: a.Criticality,
			Reactions:   a.Reactions,
		})
	}

	return allergies, nil
}

// CheckContraindications checks drug-condition contraindications.
// Calls KB-4 /v1/contraindications/check endpoint for each medication.
// KB-4 API expects one drug at a time with patient diagnoses.
func (c *KB4HTTPClient) CheckContraindications(
	ctx context.Context,
	meds []contracts.ClinicalCode,
	conditions []contracts.ClinicalCode,
) ([]contracts.ContraindicationInfo, error) {

	if len(meds) == 0 || len(conditions) == 0 {
		return []contracts.ContraindicationInfo{}, nil
	}

	// Build diagnoses list in KB-4 expected format
	diagnoses := make([]kb4Diagnosis, 0, len(conditions))
	for _, cond := range conditions {
		diagnoses = append(diagnoses, kb4Diagnosis{
			Code:        cond.Code,
			Description: cond.Display,
		})
	}

	var allContraindications []contracts.ContraindicationInfo

	// Check each medication against all conditions
	for _, med := range meds {
		req := kb4ContraindicationCheckRequest{
			RxNormCode: med.Code,
			Diagnoses:  diagnoses,
		}

		resp, err := c.callKB4(ctx, "/v1/contraindications/check", req)
		if err != nil {
			// Log but continue with other medications
			continue
		}

		var result kb4ContraindicationCheckResult
		if err := json.Unmarshal(resp, &result); err != nil {
			continue
		}

		// Process matches from the response
		for _, match := range result.Matches {
			allContraindications = append(allContraindications, contracts.ContraindicationInfo{
				Medication: contracts.ClinicalCode{
					System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
					Code:    med.Code,
					Display: med.Display,
				},
				Condition: contracts.ClinicalCode{
					System:  "http://snomed.info/sct",
					Code:    match.MatchedDiagnosis.Code,
					Display: match.MatchedDiagnosis.Description,
				},
				Severity:       match.Severity,
				Description:    match.Contraindication.Description,
				Recommendation: match.Contraindication.Management,
				Evidence:       match.Contraindication.EvidenceLevel,
			})
		}
	}

	return allContraindications, nil
}

// GetPregnancyStatus returns pregnancy information if applicable.
// Calls KB-4 /api/v1/safety/pregnancy endpoint.
func (c *KB4HTTPClient) GetPregnancyStatus(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.PregnancyInfo, error) {

	// Only check pregnancy for female patients
	if patient.Demographics.Gender != "female" && patient.Demographics.Gender != "F" {
		return nil, nil
	}

	req := kb4PregnancyRequest{
		PatientID: patient.Demographics.PatientID,
	}

	resp, err := c.callKB4(ctx, "/api/v1/safety/pregnancy", req)
	if err != nil {
		// Not found is not an error - patient may not be pregnant
		return nil, nil
	}

	var result kb4PregnancyResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, nil
	}

	if !result.IsPregnant {
		return nil, nil
	}

	return &contracts.PregnancyInfo{
		IsPregnant:      result.IsPregnant,
		Trimester:       result.Trimester,
		EstimatedWeeks:  result.EstimatedWeeks,
		LactationStatus: result.Lactating,
	}, nil
}

// NeedsRenalDoseAdjustment checks if renal adjustment needed based on eGFR.
// Returns true if eGFR < 60 (CKD Stage 3 or worse).
func (c *KB4HTTPClient) NeedsRenalDoseAdjustment(
	ctx context.Context,
	eGFR float64,
) (bool, error) {
	// CKD Stage classification:
	// Stage 1: eGFR >= 90 (normal or high)
	// Stage 2: eGFR 60-89 (mildly decreased)
	// Stage 3a: eGFR 45-59 (mildly to moderately decreased)
	// Stage 3b: eGFR 30-44 (moderately to severely decreased)
	// Stage 4: eGFR 15-29 (severely decreased)
	// Stage 5: eGFR < 15 (kidney failure)

	// Need adjustment at Stage 3 or worse
	return eGFR < 60, nil
}

// NeedsHepaticDoseAdjustment checks if hepatic adjustment needed.
func (c *KB4HTTPClient) NeedsHepaticDoseAdjustment(
	ctx context.Context,
	patient *contracts.PatientContext,
) (bool, error) {

	// Check for liver disease conditions
	liverCodes := []string{
		"235856003", // Hepatic impairment
		"19943007",  // Cirrhosis
		"197321007", // Chronic hepatitis
		"328383001", // Chronic liver disease
	}

	for _, cond := range patient.ActiveConditions {
		for _, code := range liverCodes {
			if cond.Code.Code == code {
				return true, nil
			}
		}
	}

	return false, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB4HTTPClient) callKB4(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
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
		return nil, fmt.Errorf("KB-4 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-4 service is healthy.
func (c *KB4HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-4 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// KB-4 Request/Response Types (internal)
// ============================================================================

type kb4AllergyRequest struct {
	PatientID string `json:"patient_id"`
}

type kb4AllergyResult struct {
	Allergies []kb4Allergy `json:"allergies"`
}

type kb4Allergy struct {
	AllergenSystem  string   `json:"allergen_system"`
	AllergenCode    string   `json:"allergen_code"`
	AllergenDisplay string   `json:"allergen_display"`
	Category        string   `json:"category"`    // food, medication, environment
	Criticality     string   `json:"criticality"` // low, high, unable-to-assess
	Reactions       []string `json:"reactions"`
}

// kb4Diagnosis matches the KB-4 API's expected diagnosis format.
type kb4Diagnosis struct {
	Code        string `json:"code"`
	Description string `json:"description,omitempty"`
}

// kb4ContraindicationCheckRequest matches the KB-4 /v1/contraindications/check API.
type kb4ContraindicationCheckRequest struct {
	RxNormCode string         `json:"rxnormCode"`
	Diagnoses  []kb4Diagnosis `json:"diagnoses"`
}

// kb4ContraindicationCheckResult is the response from /v1/contraindications/check.
type kb4ContraindicationCheckResult struct {
	RxNormCode          string                     `json:"rxnormCode"`
	HasContraindication bool                       `json:"hasContraindication"`
	MatchCount          int                        `json:"matchCount"`
	Matches             []kb4ContraindicationMatch `json:"matches"`
}

// kb4ContraindicationMatch represents a matched contraindication.
type kb4ContraindicationMatch struct {
	Contraindication kb4ContraindicationDetail `json:"contraindication"`
	MatchedDiagnosis kb4Diagnosis              `json:"matchedDiagnosis"`
	Severity         string                    `json:"severity"`
}

// kb4ContraindicationDetail contains contraindication details from KB-4.
type kb4ContraindicationDetail struct {
	Severity       string   `json:"severity"`
	Description    string   `json:"description"`
	Management     string   `json:"management,omitempty"`
	EvidenceLevel  string   `json:"evidenceLevel,omitempty"`
	ConditionCodes []string `json:"conditionCodes,omitempty"`
}

type kb4PregnancyRequest struct {
	PatientID string `json:"patient_id"`
}

type kb4PregnancyResult struct {
	IsPregnant     bool `json:"is_pregnant"`
	Trimester      int  `json:"trimester"`
	EstimatedWeeks int  `json:"estimated_weeks"`
	Lactating      bool `json:"lactating"`
}

// ============================================================================
// ENHANCED KB-4 SAFETY METHODS
// ============================================================================
// These methods provide comprehensive safety checking using KB-4's full API.
// They are used by KnowledgeSnapshotBuilder to populate EnhancedSafetySnapshot.

// CheckMedicationSafety performs comprehensive safety evaluation for a medication.
// Calls KB-4 POST /v1/check endpoint with full patient context.
// Returns all applicable safety alerts (black box, contraindications, dose limits, etc.)
func (c *KB4HTTPClient) CheckMedicationSafety(
	ctx context.Context,
	drug *contracts.ClinicalCode,
	proposedDose float64,
	doseUnit string,
	patient *contracts.PatientContext,
) (*KB4SafetyCheckResponse, error) {

	// Build request matching KB-4's SafetyCheckRequest format
	req := kb4SafetyCheckRequest{
		Drug: kb4DrugInfo{
			RxNormCode: drug.Code,
			DrugName:   drug.Display,
		},
		ProposedDose: proposedDose,
		DoseUnit:     doseUnit,
		Patient:      buildKB4PatientContext(patient),
	}

	resp, err := c.callKB4(ctx, "/v1/check", req)
	if err != nil {
		return nil, fmt.Errorf("safety check failed: %w", err)
	}

	var result KB4SafetyCheckResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse safety check response: %w", err)
	}

	return &result, nil
}

// GetBlackBoxWarnings retrieves FDA/TGA black box warnings for medications.
// Calls KB-4 GET /v1/blackbox?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetBlackBoxWarnings(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4BlackBoxWarning, error) {

	warnings := make(map[string]*KB4BlackBoxWarning)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/blackbox?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			// No warning found is not an error
			continue
		}

		var warning KB4BlackBoxWarning
		if err := json.Unmarshal(resp, &warning); err != nil {
			continue
		}

		if warning.RxNormCode != "" {
			warnings[code] = &warning
		}
	}

	return warnings, nil
}

// GetHighAlertStatus retrieves ISMP high-alert medication status.
// Calls KB-4 GET /v1/high-alert?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetHighAlertStatus(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4HighAlertMedication, error) {

	highAlerts := make(map[string]*KB4HighAlertMedication)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/high-alert?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var alert KB4HighAlertMedication
		if err := json.Unmarshal(resp, &alert); err != nil {
			continue
		}

		if alert.RxNormCode != "" {
			highAlerts[code] = &alert
		}
	}

	return highAlerts, nil
}

// GetPregnancySafetyInfo retrieves comprehensive pregnancy safety information.
// Calls KB-4 GET /v1/pregnancy?rxnorm=XXX endpoint.
// Returns FDA PLLR data, TGA categories, teratogenicity info, alternatives.
func (c *KB4HTTPClient) GetPregnancySafetyInfo(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4PregnancySafety, error) {

	pregnancyInfo := make(map[string]*KB4PregnancySafety)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/pregnancy?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var info KB4PregnancySafety
		if err := json.Unmarshal(resp, &info); err != nil {
			continue
		}

		if info.RxNormCode != "" {
			pregnancyInfo[code] = &info
		}
	}

	return pregnancyInfo, nil
}

// GetLactationSafetyInfo retrieves LactMed lactation safety information.
// Calls KB-4 GET /v1/lactation?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetLactationSafetyInfo(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4LactationSafety, error) {

	lactationInfo := make(map[string]*KB4LactationSafety)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/lactation?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var info KB4LactationSafety
		if err := json.Unmarshal(resp, &info); err != nil {
			continue
		}

		if info.RxNormCode != "" {
			lactationInfo[code] = &info
		}
	}

	return lactationInfo, nil
}

// GetBeersCriteria retrieves AGS Beers Criteria entries for geriatric patients.
// Calls KB-4 GET /v1/beers?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetBeersCriteria(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4BeersEntry, error) {

	beersEntries := make(map[string]*KB4BeersEntry)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/beers?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var entry KB4BeersEntry
		if err := json.Unmarshal(resp, &entry); err != nil {
			continue
		}

		if entry.RxNormCode != "" {
			beersEntries[code] = &entry
		}
	}

	return beersEntries, nil
}

// GetAnticholinergicBurden retrieves ACB scores for medications.
// Calls KB-4 GET /v1/anticholinergic?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetAnticholinergicBurden(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4AnticholinergicBurden, error) {

	acbScores := make(map[string]*KB4AnticholinergicBurden)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/anticholinergic?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var acb KB4AnticholinergicBurden
		if err := json.Unmarshal(resp, &acb); err != nil {
			continue
		}

		if acb.RxNormCode != "" {
			acbScores[code] = &acb
		}
	}

	return acbScores, nil
}

// CalculateTotalAnticholinergicBurden calculates cumulative ACB score.
// Returns total score and risk level (Low: 1-2, Moderate: 3-4, High: 5+).
func (c *KB4HTTPClient) CalculateTotalAnticholinergicBurden(
	ctx context.Context,
	rxnormCodes []string,
) (*KB4ACBCalculation, error) {

	acbScores, err := c.GetAnticholinergicBurden(ctx, rxnormCodes)
	if err != nil {
		return nil, err
	}

	totalScore := 0
	medications := make([]KB4AnticholinergicBurden, 0)

	for _, acb := range acbScores {
		if acb != nil {
			totalScore += acb.ACBScore
			medications = append(medications, *acb)
		}
	}

	riskLevel := "Low"
	recommendation := "Low anticholinergic burden. Monitor as usual."

	if totalScore >= 5 {
		riskLevel = "High"
		recommendation = "High anticholinergic burden (≥5). Consider deprescribing anticholinergic medications. Monitor for cognitive effects."
	} else if totalScore >= 3 {
		riskLevel = "Moderate"
		recommendation = "Moderate anticholinergic burden (3-4). Review for opportunities to reduce burden, especially in older adults."
	}

	return &KB4ACBCalculation{
		TotalScore:     totalScore,
		RiskLevel:      riskLevel,
		Medications:    medications,
		Recommendation: recommendation,
	}, nil
}

// GetLabRequirements retrieves required lab monitoring for medications.
// Calls KB-4 GET /v1/labs?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetLabRequirements(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4LabRequirement, error) {

	labReqs := make(map[string]*KB4LabRequirement)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/labs?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var req KB4LabRequirement
		if err := json.Unmarshal(resp, &req); err != nil {
			continue
		}

		if req.RxNormCode != "" {
			labReqs[code] = &req
		}
	}

	return labReqs, nil
}

// GetDoseLimits retrieves dose limit information for medications.
// Calls KB-4 GET /v1/limits?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetDoseLimits(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4DoseLimit, error) {

	doseLimits := make(map[string]*KB4DoseLimit)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/limits?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var limit KB4DoseLimit
		if err := json.Unmarshal(resp, &limit); err != nil {
			continue
		}

		if limit.RxNormCode != "" {
			doseLimits[code] = &limit
		}
	}

	return doseLimits, nil
}

// ValidateDose validates a proposed dose against KB-4 dose limits.
// Calls KB-4 POST /v1/limits/validate endpoint.
func (c *KB4HTTPClient) ValidateDose(
	ctx context.Context,
	drug *contracts.ClinicalCode,
	proposedDose float64,
	doseUnit string,
	patient *contracts.PatientContext,
) (*KB4DoseValidation, error) {

	req := kb4DoseValidationRequest{
		Drug: kb4DrugInfo{
			RxNormCode: drug.Code,
			DrugName:   drug.Display,
		},
		ProposedDose: proposedDose,
		DoseUnit:     doseUnit,
		Patient:      buildKB4PatientContext(patient),
	}

	resp, err := c.callKB4(ctx, "/v1/limits/validate", req)
	if err != nil {
		return nil, fmt.Errorf("dose validation failed: %w", err)
	}

	var result KB4DoseValidation
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse dose validation response: %w", err)
	}

	return &result, nil
}

// GetAgeLimits retrieves age restriction information for medications.
// Calls KB-4 GET /v1/age-limits?rxnorm=XXX endpoint.
func (c *KB4HTTPClient) GetAgeLimits(
	ctx context.Context,
	rxnormCodes []string,
) (map[string]*KB4AgeLimit, error) {

	ageLimits := make(map[string]*KB4AgeLimit)

	for _, code := range rxnormCodes {
		endpoint := fmt.Sprintf("/v1/age-limits?rxnorm=%s", url.QueryEscape(code))
		resp, err := c.callKB4GET(ctx, endpoint)
		if err != nil {
			continue
		}

		var limit KB4AgeLimit
		if err := json.Unmarshal(resp, &limit); err != nil {
			continue
		}

		if limit.RxNormCode != "" {
			ageLimits[code] = &limit
		}
	}

	return ageLimits, nil
}

// ============================================================================
// HTTP Helper Methods (Extended)
// ============================================================================

// callKB4GET performs GET requests to KB-4 endpoints.
func (c *KB4HTTPClient) callKB4GET(ctx context.Context, endpoint string) ([]byte, error) {
	fullURL := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", fullURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("not found")
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-4 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// buildKB4PatientContext converts contracts.PatientContext to KB-4's format.
func buildKB4PatientContext(patient *contracts.PatientContext) kb4PatientContext {
	if patient == nil {
		return kb4PatientContext{}
	}

	ctx := kb4PatientContext{
		PatientID:   patient.Demographics.PatientID,
		Gender:      patient.Demographics.Gender,
		IsPregnant:  false,
		IsLactating: false,
	}

	// Calculate age from BirthDate
	if patient.Demographics.BirthDate != nil {
		ctx.AgeYears = float64(calculateAgeYears(*patient.Demographics.BirthDate))
	}

	// Extract weight/height from recent vital signs
	for _, vital := range patient.RecentVitalSigns {
		if vital.Value != nil {
			switch vital.Code.Code {
			case "29463-7": // Body weight LOINC
				ctx.WeightKg = vital.Value.Value
			case "8302-2": // Body height LOINC
				ctx.HeightCm = vital.Value.Value
			}
		}
	}

	// Convert conditions to diagnoses
	for _, cond := range patient.ActiveConditions {
		ctx.Diagnoses = append(ctx.Diagnoses, kb4Diagnosis{
			Code:        cond.Code.Code,
			Description: cond.Code.Display,
		})
	}

	// Convert allergies
	for _, allergy := range patient.Allergies {
		ctx.Allergies = append(ctx.Allergies, kb4AllergyEntry{
			Substance:    allergy.Code.Display,
			Code:         allergy.Code.Code,
			ReactionType: allergy.Category,
			Severity:     allergy.Criticality,
		})
	}

	// Check for pregnancy in conditions
	pregnancyCodes := []string{"Z33.1", "77386006", "O00-O9A"} // ICD-10 & SNOMED
	for _, cond := range patient.ActiveConditions {
		for _, pc := range pregnancyCodes {
			if strings.HasPrefix(cond.Code.Code, pc) || cond.Code.Code == pc {
				ctx.IsPregnant = true
				break
			}
		}
	}

	// Extract renal function from recent lab results
	for _, lab := range patient.RecentLabResults {
		if lab.Value != nil {
			switch lab.Code.Code {
			case "33914-3": // eGFR LOINC
				if ctx.RenalFunction == nil {
					ctx.RenalFunction = &kb4RenalFunction{}
				}
				ctx.RenalFunction.EGFR = lab.Value.Value
			case "2160-0": // Serum creatinine LOINC
				if ctx.RenalFunction == nil {
					ctx.RenalFunction = &kb4RenalFunction{}
				}
				ctx.RenalFunction.Creatinine = lab.Value.Value
			}
		}
	}

	// Convert current medications
	for _, med := range patient.ActiveMedications {
		ctx.CurrentMedications = append(ctx.CurrentMedications, kb4DrugInfo{
			RxNormCode: med.Code.Code,
			DrugName:   med.Code.Display,
		})
	}

	return ctx
}

// calculateAgeYears calculates age in years from birth date.
func calculateAgeYears(birthDate time.Time) int {
	now := time.Now()
	years := now.Year() - birthDate.Year()
	if now.YearDay() < birthDate.YearDay() {
		years--
	}
	return years
}

// ============================================================================
// KB-4 Enhanced Types (matching KB-4 pkg/safety/types.go)
// ============================================================================

// kb4SafetyCheckRequest matches KB-4's SafetyCheckRequest.
type kb4SafetyCheckRequest struct {
	Drug         kb4DrugInfo       `json:"drug"`
	ProposedDose float64           `json:"proposedDose,omitempty"`
	DoseUnit     string            `json:"doseUnit,omitempty"`
	Frequency    string            `json:"frequency,omitempty"`
	Route        string            `json:"route,omitempty"`
	Patient      kb4PatientContext `json:"patient"`
	CheckTypes   []string          `json:"checkTypes,omitempty"` // Empty = all checks
}

// kb4DrugInfo matches KB-4's DrugInfo.
type kb4DrugInfo struct {
	RxNormCode string `json:"rxnormCode"`
	DrugName   string `json:"drugName"`
	NDC        string `json:"ndc,omitempty"`
	DrugClass  string `json:"drugClass,omitempty"`
}

// kb4PatientContext matches KB-4's PatientContext.
type kb4PatientContext struct {
	PatientID          string              `json:"patientId,omitempty"`
	AgeYears           float64             `json:"ageYears"`
	AgeMonths          float64             `json:"ageMonths,omitempty"`
	Gender             string              `json:"gender"`
	WeightKg           float64             `json:"weightKg,omitempty"`
	HeightCm           float64             `json:"heightCm,omitempty"`
	IsPregnant         bool                `json:"isPregnant"`
	IsLactating        bool                `json:"isLactating"`
	Trimester          int                 `json:"trimester,omitempty"`
	Diagnoses          []kb4Diagnosis      `json:"diagnoses,omitempty"`
	Allergies          []kb4AllergyEntry   `json:"allergies,omitempty"`
	RenalFunction      *kb4RenalFunction   `json:"renalFunction,omitempty"`
	HepaticFunction    *kb4HepaticFunction `json:"hepaticFunction,omitempty"`
	CurrentMedications []kb4DrugInfo       `json:"currentMedications,omitempty"`
}

// kb4AllergyEntry for patient allergies.
type kb4AllergyEntry struct {
	Substance    string `json:"substance"`
	Code         string `json:"code,omitempty"`
	ReactionType string `json:"reactionType,omitempty"`
	Severity     string `json:"severity,omitempty"`
}

// kb4RenalFunction matches KB-4's RenalFunction.
type kb4RenalFunction struct {
	Creatinine float64 `json:"creatinine,omitempty"`
	BUN        float64 `json:"bun,omitempty"`
	EGFR       float64 `json:"egfr,omitempty"`
	CrCl       float64 `json:"crcl,omitempty"`
	Stage      string  `json:"stage,omitempty"`
}

// kb4HepaticFunction matches KB-4's HepaticFunction.
type kb4HepaticFunction struct {
	AST            float64 `json:"ast,omitempty"`
	ALT            float64 `json:"alt,omitempty"`
	Bilirubin      float64 `json:"bilirubin,omitempty"`
	Albumin        float64 `json:"albumin,omitempty"`
	INR            float64 `json:"inr,omitempty"`
	ChildPughScore int     `json:"childPughScore,omitempty"`
	ChildPughClass string  `json:"childPughClass,omitempty"`
}

// kb4DoseValidationRequest for dose limit validation.
type kb4DoseValidationRequest struct {
	Drug         kb4DrugInfo       `json:"drug"`
	ProposedDose float64           `json:"proposedDose"`
	DoseUnit     string            `json:"doseUnit"`
	Patient      kb4PatientContext `json:"patient,omitempty"`
}

// ============================================================================
// KB-4 Response Types (exported for use by KnowledgeSnapshotBuilder)
// ============================================================================

// KB4SafetyCheckResponse represents comprehensive safety check results.
type KB4SafetyCheckResponse struct {
	Safe                       bool             `json:"safe"`
	RequiresAction             bool             `json:"requiresAction"`
	BlockPrescribing           bool             `json:"blockPrescribing"`
	CriticalAlerts             int              `json:"criticalAlerts"`
	HighAlerts                 int              `json:"highAlerts"`
	ModerateAlerts             int              `json:"moderateAlerts"`
	LowAlerts                  int              `json:"lowAlerts"`
	TotalAlerts                int              `json:"totalAlerts"`
	Alerts                     []KB4SafetyAlert `json:"alerts"`
	IsHighAlertDrug            bool             `json:"isHighAlertDrug"`
	AnticholinergicBurdenTotal int              `json:"anticholinergicBurdenTotal,omitempty"`
	CheckedAt                  time.Time        `json:"checkedAt"`
	RequestID                  string           `json:"requestId,omitempty"`
}

// KB4SafetyAlert represents an individual safety finding.
type KB4SafetyAlert struct {
	ID                     string    `json:"id,omitempty"`
	Type                   string    `json:"type"` // BLACK_BOX_WARNING, CONTRAINDICATION, etc.
	Severity               string    `json:"severity"` // CRITICAL, HIGH, MODERATE, LOW
	Title                  string    `json:"title"`
	Message                string    `json:"message"`
	RequiresAcknowledgment bool      `json:"requiresAcknowledgment"`
	CanOverride            bool      `json:"canOverride"`
	ClinicalRationale      string    `json:"clinicalRationale,omitempty"`
	Recommendations        []string  `json:"recommendations,omitempty"`
	References             []string  `json:"references,omitempty"`
	DrugInfo               *kb4DrugInfo `json:"drugInfo,omitempty"`
	CreatedAt              time.Time `json:"createdAt,omitempty"`
}

// KB4BlackBoxWarning represents FDA/TGA black box warning data.
type KB4BlackBoxWarning struct {
	RxNormCode       string           `json:"rxnormCode"`
	DrugName         string           `json:"drugName"`
	ATCCode          string           `json:"atcCode,omitempty"`
	DrugClass        string           `json:"drugClass,omitempty"`
	RiskCategories   []string         `json:"riskCategories"`
	WarningText      string           `json:"warningText"`
	Severity         string           `json:"severity"` // CRITICAL, HIGH
	HasREMS          bool             `json:"hasRems"`
	REMSProgram      string           `json:"remsProgram,omitempty"`
	REMSRequirements []string         `json:"remsRequirements,omitempty"`
	Governance       KB4Governance    `json:"governance"`
}

// KB4HighAlertMedication represents ISMP high-alert medication data.
type KB4HighAlertMedication struct {
	RxNormCode             string        `json:"rxnormCode"`
	DrugName               string        `json:"drugName"`
	ATCCode                string        `json:"atcCode,omitempty"`
	TallManName            string        `json:"tallManName,omitempty"`
	Category               string        `json:"category"` // ANTICOAGULANTS, INSULIN, OPIOIDS, etc.
	ISMPListType           string        `json:"ismpListType,omitempty"`
	Requirements           []string      `json:"requirements"`
	Safeguards             []string      `json:"safeguards"`
	DoubleCheck            bool          `json:"doubleCheck"`
	SmartPump              bool          `json:"smartPump"`
	IndependentDoubleCheck bool          `json:"independentDoubleCheck,omitempty"`
	Governance             KB4Governance `json:"governance"`
}

// KB4PregnancySafety represents pregnancy safety information.
type KB4PregnancySafety struct {
	RxNormCode             string        `json:"rxnormCode"`
	DrugName               string        `json:"drugName"`
	GenericName            string        `json:"genericName,omitempty"`
	ATCCode                string        `json:"atcCode,omitempty"`
	Category               string        `json:"category"` // A, B, C, D, X, N
	RiskCategory           string        `json:"riskCategory,omitempty"`
	PLLRRiskSummary        string        `json:"pllrRiskSummary,omitempty"`
	PLLRClinicalConsiderations string    `json:"pllrClinicalConsiderations,omitempty"`
	Teratogenic            bool          `json:"teratogenic"`
	TeratogenicEffects     []string      `json:"teratogenicEffects,omitempty"`
	TrimesterRisks         map[string]string `json:"trimesterRisks,omitempty"`
	Recommendation         string        `json:"recommendation"`
	AlternativeDrugs       []string      `json:"alternativeDrugs,omitempty"`
	MonitoringRequired     []string      `json:"monitoringRequired,omitempty"`
	Governance             KB4Governance `json:"governance"`
}

// KB4LactationSafety represents lactation safety information.
type KB4LactationSafety struct {
	RxNormCode        string        `json:"rxnormCode"`
	DrugName          string        `json:"drugName"`
	GenericName       string        `json:"genericName,omitempty"`
	ATCCode           string        `json:"atcCode,omitempty"`
	Risk              string        `json:"risk"` // COMPATIBLE, PROBABLY_COMPATIBLE, USE_WITH_CAUTION, CONTRAINDICATED, UNKNOWN
	RiskSummary       string        `json:"riskSummary,omitempty"`
	ExcretedInMilk    bool          `json:"excretedInMilk"`
	MilkPlasmaRatio   string        `json:"milkPlasmaRatio,omitempty"`
	InfantDosePercent float64       `json:"infantDosePercent,omitempty"`
	HalfLifeHours     float64       `json:"halfLifeHours"`
	InfantEffects     []string      `json:"infantEffects,omitempty"`
	InfantMonitoring  []string      `json:"infantMonitoring,omitempty"`
	Recommendation    string        `json:"recommendation"`
	AlternativeDrugs  []string      `json:"alternativeDrugs,omitempty"`
	TimingAdvice      string        `json:"timingAdvice,omitempty"`
	Governance        KB4Governance `json:"governance"`
}

// KB4BeersEntry represents AGS Beers Criteria entry.
type KB4BeersEntry struct {
	RxNormCode               string        `json:"rxnormCode"`
	DrugName                 string        `json:"drugName"`
	ATCCode                  string        `json:"atcCode,omitempty"`
	DrugClass                string        `json:"drugClass,omitempty"`
	Recommendation           string        `json:"recommendation"` // AVOID, AVOID_IN_CONDITION, USE_WITH_CAUTION
	BeersTable               string        `json:"beersTable,omitempty"`
	Rationale                string        `json:"rationale"`
	QualityOfEvidence        string        `json:"qualityOfEvidence"`
	StrengthOfRecommendation string        `json:"strengthOfRecommendation"`
	Conditions               []string      `json:"conditions,omitempty"`
	ConditionCodes           []string      `json:"conditionCodes,omitempty"`
	ACBScore                 int           `json:"acbScore,omitempty"`
	AlternativeDrugs         []string      `json:"alternativeDrugs,omitempty"`
	NonPharmacologic         []string      `json:"nonPharmacologic,omitempty"`
	AgeThreshold             int           `json:"ageThreshold,omitempty"`
	Governance               KB4Governance `json:"governance"`
}

// KB4AnticholinergicBurden represents ACB score data.
type KB4AnticholinergicBurden struct {
	RxNormCode        string        `json:"rxnormCode"`
	DrugName          string        `json:"drugName"`
	ATCCode           string        `json:"atcCode,omitempty"`
	ACBScore          int           `json:"acbScore"` // 1-3 scale
	RiskLevel         string        `json:"riskLevel"` // Low, Moderate, High
	ScaleUsed         string        `json:"scaleUsed,omitempty"`
	Effects           []string      `json:"effects,omitempty"`
	CognitiveRisk     string        `json:"cognitiveRisk,omitempty"`
	PeripheralEffects []string      `json:"peripheralEffects,omitempty"`
	GeriatricRisk     string        `json:"geriatricRisk,omitempty"`
	Governance        KB4Governance `json:"governance"`
}

// KB4ACBCalculation represents cumulative ACB calculation result.
type KB4ACBCalculation struct {
	TotalScore     int                        `json:"totalScore"`
	RiskLevel      string                     `json:"riskLevel"` // Low, Moderate, High, Very High
	Medications    []KB4AnticholinergicBurden `json:"medications"`
	Recommendation string                     `json:"recommendation"`
	CognitiveRisk  string                     `json:"cognitiveRisk,omitempty"`
}

// KB4LabRequirement represents lab monitoring requirements.
type KB4LabRequirement struct {
	RxNormCode         string            `json:"rxnormCode"`
	DrugName           string            `json:"drugName"`
	GenericName        string            `json:"genericName,omitempty"`
	ATCCode            string            `json:"atcCode,omitempty"`
	DrugClass          string            `json:"drugClass,omitempty"`
	MonitoringRequired bool              `json:"monitoringRequired"`
	CriticalMonitoring bool              `json:"criticalMonitoring,omitempty"`
	REMSProgram        string            `json:"remsProgram,omitempty"`
	RequiredLabs       []string          `json:"requiredLabs,omitempty"`
	LabCodes           []string          `json:"labCodes,omitempty"` // LOINC codes
	Frequency          string            `json:"frequency,omitempty"`
	BaselineRequired   bool              `json:"baselineRequired,omitempty"`
	InitialMonitoring  string            `json:"initialMonitoring,omitempty"`
	OngoingMonitoring  string            `json:"ongoingMonitoring,omitempty"`
	CriticalValues     map[string]string `json:"criticalValues,omitempty"`
	ActionRequired     string            `json:"actionRequired,omitempty"`
	Rationale          string            `json:"rationale,omitempty"`
	Governance         KB4Governance     `json:"governance"`
}

// KB4DoseLimit represents dose limit information.
type KB4DoseLimit struct {
	RxNormCode         string             `json:"rxnormCode"`
	DrugName           string             `json:"drugName"`
	ATCCode            string             `json:"atcCode,omitempty"`
	MaxSingleDose      float64            `json:"maxSingleDose"`
	MaxSingleDoseUnit  string             `json:"maxSingleDoseUnit"`
	MaxDailyDose       float64            `json:"maxDailyDose"`
	MaxDailyDoseUnit   string             `json:"maxDailyDoseUnit"`
	MaxCumulativeDose  float64            `json:"maxCumulativeDose,omitempty"`
	GeriatricMaxDose   float64            `json:"geriatricMaxDose,omitempty"`
	PediatricMaxDose   float64            `json:"pediatricMaxDose,omitempty"`
	RenalAdjustment    string             `json:"renalAdjustment,omitempty"`
	HepaticAdjustment  string             `json:"hepaticAdjustment,omitempty"`
	RenalDoseByEGFR    map[string]float64 `json:"renalDoseByEgfr,omitempty"`
	HepaticDoseByClass map[string]float64 `json:"hepaticDoseByClass,omitempty"`
	Governance         KB4Governance      `json:"governance"`
}

// KB4AgeLimit represents age restriction data.
type KB4AgeLimit struct {
	RxNormCode  string        `json:"rxnormCode"`
	DrugName    string        `json:"drugName"`
	MinAgeYears float64       `json:"minAgeYears,omitempty"`
	MaxAgeYears float64       `json:"maxAgeYears,omitempty"`
	Rationale   string        `json:"rationale"`
	Severity    string        `json:"severity"` // CRITICAL, HIGH, MODERATE, LOW
	Governance  KB4Governance `json:"governance"`
}

// KB4DoseValidation represents dose validation result.
type KB4DoseValidation struct {
	Drug          kb4DrugInfo `json:"drug"`
	ProposedDose  float64     `json:"proposedDose"`
	DoseUnit      string      `json:"doseUnit"`
	IsValid       bool        `json:"isValid"`
	ExceedsSingle bool        `json:"exceedsSingle"`
	ExceedsDaily  bool        `json:"exceedsDaily"`
	MaxAllowed    float64     `json:"maxAllowed,omitempty"`
	Message       string      `json:"message,omitempty"`
}

// KB4Governance contains clinical governance metadata.
type KB4Governance struct {
	SourceAuthority    string `json:"sourceAuthority"`    // FDA, TGA, ISMP, AGS, etc.
	SourceDocument     string `json:"sourceDocument"`
	SourceSection      string `json:"sourceSection,omitempty"`
	SourceURL          string `json:"sourceUrl,omitempty"`
	Jurisdiction       string `json:"jurisdiction"`       // US, AU, EU, IN, UK, GLOBAL
	EvidenceLevel      string `json:"evidenceLevel"`      // A, B, C, D, EXPERT
	EffectiveDate      string `json:"effectiveDate"`
	ReviewDate         string `json:"reviewDate,omitempty"`
	KnowledgeVersion   string `json:"knowledgeVersion"`
	ApprovalStatus     string `json:"approvalStatus"`     // DRAFT, REVIEWED, APPROVED, ACTIVE
}
