// Package clients provides HTTP/GraphQL clients for KB services.
//
// KB8HTTPClient implements the KB8Client interface for KB-8 Calculator Service.
// It extracts parameters from PatientContext and calls the KB-8 REST API
// to compute clinical calculations (eGFR, ASCVD, CHA2DS2-VASc, etc.).
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// This client is used by KnowledgeSnapshotBuilder to populate CalculatorSnapshot.
// All calculations are pre-computed at snapshot build time - engines NEVER call
// calculators directly at execution time.
//
// Connects to: http://localhost:8088 (configurable)
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

// KB8HTTPClient implements KB8Client by calling the KB-8 Calculator Service REST API.
type KB8HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB8HTTPClient creates a new KB-8 HTTP client.
func NewKB8HTTPClient(baseURL string) *KB8HTTPClient {
	return &KB8HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB8HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB8HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB8HTTPClient {
	return &KB8HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB8Client Interface Implementation
// ============================================================================

// CalculateEGFR calculates Estimated Glomerular Filtration Rate.
// Requires: serumCreatinine (from labs), age, sex
func (c *KB8HTTPClient) CalculateEGFR(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {

	// Extract required parameters
	creatinine := c.extractLabValue(patient, "2160-0") // LOINC: Serum Creatinine
	if creatinine == nil {
		return nil, fmt.Errorf("missing serum creatinine for eGFR calculation")
	}

	age := c.extractAge(patient)
	if age <= 0 {
		return nil, fmt.Errorf("missing or invalid age for eGFR calculation")
	}

	sex := c.mapGenderToSex(patient.Demographics.Gender)
	if sex == "" {
		return nil, fmt.Errorf("missing sex for eGFR calculation")
	}

	// Build request
	req := map[string]interface{}{
		"serumCreatinine": *creatinine,
		"ageYears":        age,
		"sex":             sex,
	}

	// Call KB-8
	resp, err := c.callCalculator(ctx, "/api/v1/calculate/egfr", req)
	if err != nil {
		return nil, fmt.Errorf("eGFR calculation failed: %w", err)
	}

	// Parse response
	var result kb8EGFRResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse eGFR response: %w", err)
	}

	// Convert to contracts.CalculationResult
	return &contracts.CalculationResult{
		Name:     "eGFR",
		Value:    result.Value,
		Unit:     result.Unit,
		Category: string(result.CKDStage),
		Formula:  result.Equation,
		Inputs: map[string]interface{}{
			"serumCreatinine": *creatinine,
			"ageYears":        age,
			"sex":             sex,
		},
		CalculatedAt: result.Provenance.CalculatedAt,
		Warnings:     result.Provenance.Caveats,
	}, nil
}

// CalculateASCVD calculates 10-year ASCVD risk using Pooled Cohort Equations.
// Requires: age, sex, totalCholesterol, hdlCholesterol, systolicBP
func (c *KB8HTTPClient) CalculateASCVD(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {

	// Extract required parameters
	age := c.extractAge(patient)
	if age < 40 || age > 79 {
		return nil, fmt.Errorf("ASCVD calculator valid for ages 40-79, got %d", age)
	}

	sex := c.mapGenderToSex(patient.Demographics.Gender)
	if sex == "" {
		return nil, fmt.Errorf("missing sex for ASCVD calculation")
	}

	totalChol := c.extractLabValue(patient, "2093-3") // LOINC: Total Cholesterol
	if totalChol == nil {
		return nil, fmt.Errorf("missing total cholesterol for ASCVD calculation")
	}

	hdlChol := c.extractLabValue(patient, "2085-9") // LOINC: HDL Cholesterol
	if hdlChol == nil {
		return nil, fmt.Errorf("missing HDL cholesterol for ASCVD calculation")
	}

	systolicBP := c.extractVitalValue(patient, "8480-6") // LOINC: Systolic BP
	if systolicBP == nil {
		// Try component value from blood pressure panel
		systolicBP = c.extractBPComponent(patient, "8480-6")
		if systolicBP == nil {
			return nil, fmt.Errorf("missing systolic BP for ASCVD calculation")
		}
	}

	// Optional parameters
	onBPTreatment := c.hasCondition(patient, "38341003") || // SNOMED: Hypertensive disorder
		c.hasMedicationClass(patient, "antihypertensive")
	hasDiabetes := c.hasCondition(patient, "73211009") || // SNOMED: Diabetes mellitus
		c.hasCondition(patient, "44054006") // Type 2 DM
	isSmoker := c.hasCondition(patient, "77176002") || // SNOMED: Smoker
		c.hasCondition(patient, "449868002") // Current every day smoker

	// Build request
	req := map[string]interface{}{
		"ageYears":         age,
		"sex":              sex,
		"totalCholesterol": *totalChol,
		"hdlCholesterol":   *hdlChol,
		"systolicBP":       *systolicBP,
		"onBPTreatment":    onBPTreatment,
		"hasDiabetes":      hasDiabetes,
		"isSmoker":         isSmoker,
	}

	// Add race if available (defaults to white coefficients if not specified)
	if patient.Demographics.Region == "US" {
		req["race"] = "white" // Would need ethnicity field for African American
	}

	// Call KB-8
	resp, err := c.callCalculator(ctx, "/api/v1/calculate/ascvd", req)
	if err != nil {
		return nil, fmt.Errorf("ASCVD calculation failed: %w", err)
	}

	// Parse response
	var result kb8ASCVDResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse ASCVD response: %w", err)
	}

	return &contracts.CalculationResult{
		Name:     "ASCVD 10-Year Risk",
		Value:    result.RiskPercent,
		Unit:     "%",
		Category: string(result.RiskCategory),
		Formula:  "Pooled Cohort Equations (2013/2018)",
		Inputs: map[string]interface{}{
			"ageYears":         age,
			"sex":              sex,
			"totalCholesterol": *totalChol,
			"hdlCholesterol":   *hdlChol,
			"systolicBP":       *systolicBP,
			"onBPTreatment":    onBPTreatment,
			"hasDiabetes":      hasDiabetes,
			"isSmoker":         isSmoker,
		},
		CalculatedAt: result.Provenance.CalculatedAt,
		Warnings:     result.Provenance.Caveats,
	}, nil
}

// CalculateCHA2DS2VASc calculates stroke risk in AFib.
// Requires: age, sex; optional condition flags
func (c *KB8HTTPClient) CalculateCHA2DS2VASc(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {

	age := c.extractAge(patient)
	if age <= 0 {
		return nil, fmt.Errorf("missing age for CHA2DS2-VASc calculation")
	}

	sex := c.mapGenderToSex(patient.Demographics.Gender)
	if sex == "" {
		return nil, fmt.Errorf("missing sex for CHA2DS2-VASc calculation")
	}

	// Detect conditions from patient's active conditions
	hasChf := c.hasCondition(patient, "42343007")    // SNOMED: Congestive heart failure
	hasHtn := c.hasCondition(patient, "38341003")    // SNOMED: Hypertensive disorder
	hasDm := c.hasCondition(patient, "73211009")     // SNOMED: Diabetes mellitus
	hasStroke := c.hasCondition(patient, "230690007") // SNOMED: Stroke
	hasVascular := c.hasCondition(patient, "22298006") || // SNOMED: MI
		c.hasCondition(patient, "400047006") // Peripheral arterial disease

	req := map[string]interface{}{
		"ageYears":                  age,
		"sex":                       sex,
		"hasCongestiveHeartFailure": hasChf,
		"hasHypertension":           hasHtn,
		"hasDiabetes":               hasDm,
		"hasStrokeTIA":              hasStroke,
		"hasVascularDisease":        hasVascular,
	}

	resp, err := c.callCalculator(ctx, "/api/v1/calculate/cha2ds2vasc", req)
	if err != nil {
		return nil, fmt.Errorf("CHA2DS2-VASc calculation failed: %w", err)
	}

	var result kb8CHA2DS2VAScResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse CHA2DS2-VASc response: %w", err)
	}

	return &contracts.CalculationResult{
		Name:     "CHA2DS2-VASc",
		Value:    float64(result.Total),
		Unit:     "points",
		Category: string(result.RiskCategory),
		Formula:  "CHA2DS2-VASc Score (2010)",
		Inputs: map[string]interface{}{
			"ageYears":                  age,
			"sex":                       sex,
			"hasCongestiveHeartFailure": hasChf,
			"hasHypertension":           hasHtn,
			"hasDiabetes":               hasDm,
			"hasStrokeTIA":              hasStroke,
			"hasVascularDisease":        hasVascular,
		},
		CalculatedAt: result.Provenance.CalculatedAt,
		Warnings:     result.Provenance.Caveats,
	}, nil
}

// CalculateHASBLED calculates bleeding risk.
func (c *KB8HTTPClient) CalculateHASBLED(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {

	age := c.extractAge(patient)

	// Detect HAS-BLED factors
	hasUncontrolledHTN := c.hasSBPAbove(patient, 160)
	hasAbnormalRenal := c.hasCondition(patient, "90688005") || // SNOMED: CKD
		c.hasLabValueAbove(patient, "2160-0", 2.26) // Creatinine > 2.26
	hasAbnormalLiver := c.hasCondition(patient, "19943007") || // SNOMED: Cirrhosis
		c.hasCondition(patient, "235856003") // Hepatic impairment
	hasStrokeHistory := c.hasCondition(patient, "230690007") // SNOMED: Stroke
	hasBleedingHistory := c.hasCondition(patient, "50960005") // SNOMED: Hemorrhage
	onAntiplatelet := c.hasMedicationClass(patient, "antiplatelet") ||
		c.hasMedicationClass(patient, "nsaid")
	excessiveAlcohol := c.hasCondition(patient, "15167005") // SNOMED: Alcohol abuse

	req := map[string]interface{}{
		"hasUncontrolledHypertension": hasUncontrolledHTN,
		"hasAbnormalRenalFunction":    hasAbnormalRenal,
		"hasAbnormalLiverFunction":    hasAbnormalLiver,
		"hasStrokeHistory":            hasStrokeHistory,
		"hasBleedingHistory":          hasBleedingHistory,
		"hasLabileINR":                false, // Would need TTR data
		"ageYears":                    age,
		"takingAntiplateletOrNSAID":   onAntiplatelet,
		"excessiveAlcohol":            excessiveAlcohol,
	}

	resp, err := c.callCalculator(ctx, "/api/v1/calculate/hasbled", req)
	if err != nil {
		return nil, fmt.Errorf("HAS-BLED calculation failed: %w", err)
	}

	var result kb8HASBLEDResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse HAS-BLED response: %w", err)
	}

	return &contracts.CalculationResult{
		Name:     "HAS-BLED",
		Value:    float64(result.Total),
		Unit:     "points",
		Category: string(result.RiskCategory),
		Formula:  "HAS-BLED Score (2010)",
		Inputs:   req,
		CalculatedAt: result.Provenance.CalculatedAt,
		Warnings:     result.Provenance.Caveats,
	}, nil
}

// CalculateBMI calculates Body Mass Index.
func (c *KB8HTTPClient) CalculateBMI(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {

	weight := c.extractVitalValue(patient, "29463-7") // LOINC: Body weight
	if weight == nil {
		return nil, fmt.Errorf("missing weight for BMI calculation")
	}

	height := c.extractVitalValue(patient, "8302-2") // LOINC: Body height
	if height == nil {
		return nil, fmt.Errorf("missing height for BMI calculation")
	}

	req := map[string]interface{}{
		"weightKg": *weight,
		"heightCm": *height,
		"region":   c.mapRegion(patient.Demographics.Region),
	}

	resp, err := c.callCalculator(ctx, "/api/v1/calculate/bmi", req)
	if err != nil {
		return nil, fmt.Errorf("BMI calculation failed: %w", err)
	}

	var result kb8BMIResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse BMI response: %w", err)
	}

	// Use regional category
	category := result.CategoryWestern
	if patient.Demographics.Region == "IN" || patient.Demographics.Region == "SG" {
		category = result.CategoryAsian
	}

	return &contracts.CalculationResult{
		Name:     "BMI",
		Value:    result.Value,
		Unit:     result.Unit,
		Category: category,
		Formula:  "weight(kg) / height(m)²",
		Inputs: map[string]interface{}{
			"weightKg": *weight,
			"heightCm": *height,
		},
		CalculatedAt: result.Provenance.CalculatedAt,
		Warnings:     result.Provenance.Caveats,
	}, nil
}

// CalculateChildPugh calculates liver function score.
// NOTE: KB-8 does not yet expose Child-Pugh endpoint - returns nil.
func (c *KB8HTTPClient) CalculateChildPugh(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {
	// Child-Pugh endpoint not yet implemented in KB-8
	// Return nil to indicate calculation not available
	return nil, nil
}

// CalculateMELD calculates Model for End-Stage Liver Disease.
// NOTE: KB-8 does not yet expose MELD endpoint - returns nil.
func (c *KB8HTTPClient) CalculateMELD(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {
	// MELD endpoint not yet implemented in KB-8
	// Return nil to indicate calculation not available
	return nil, nil
}

// CalculateSOFA calculates Sequential Organ Failure Assessment.
func (c *KB8HTTPClient) CalculateSOFA(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {

	req := make(map[string]interface{})

	// Respiration: PaO2/FiO2 ratio
	if pao2 := c.extractLabValue(patient, "2703-7"); pao2 != nil { // LOINC: PaO2
		if fio2 := c.extractLabValue(patient, "3150-0"); fio2 != nil { // LOINC: FiO2
			if *fio2 > 0 {
				req["pao2fio2Ratio"] = (*pao2 / *fio2) * 100
			}
		}
	}

	// Coagulation: Platelets
	if platelets := c.extractLabValue(patient, "777-3"); platelets != nil { // LOINC: Platelets
		req["platelets"] = *platelets
	}

	// Liver: Bilirubin
	if bilirubin := c.extractLabValue(patient, "1975-2"); bilirubin != nil { // LOINC: Total Bilirubin
		req["bilirubin"] = *bilirubin
	}

	// Cardiovascular: MAP
	if map_ := c.extractVitalValue(patient, "8478-0"); map_ != nil { // LOINC: MAP
		req["map"] = *map_
	}

	// CNS: Glasgow Coma Scale
	if gcs := c.extractVitalValueInt(patient, "9269-2"); gcs != nil { // LOINC: GCS
		req["glasgowComaScale"] = *gcs
	}

	// Renal: Creatinine
	if creatinine := c.extractLabValue(patient, "2160-0"); creatinine != nil {
		req["creatinine"] = *creatinine
	}

	resp, err := c.callCalculator(ctx, "/api/v1/calculate/sofa", req)
	if err != nil {
		return nil, fmt.Errorf("SOFA calculation failed: %w", err)
	}

	var result kb8SOFAResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse SOFA response: %w", err)
	}

	return &contracts.CalculationResult{
		Name:     "SOFA",
		Value:    float64(result.Total),
		Unit:     "points",
		Category: string(result.RiskLevel),
		Formula:  "SOFA Score (1996)",
		Inputs:   req,
		CalculatedAt: result.Provenance.CalculatedAt,
		Warnings:     result.Provenance.Caveats,
	}, nil
}

// CalculateQSOFA calculates Quick SOFA.
func (c *KB8HTTPClient) CalculateQSOFA(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.CalculationResult, error) {

	req := make(map[string]interface{})

	// Respiratory rate
	if rr := c.extractVitalValueInt(patient, "9279-1"); rr != nil { // LOINC: Respiratory rate
		req["respiratoryRate"] = *rr
	}

	// Systolic BP
	if sbp := c.extractVitalValue(patient, "8480-6"); sbp != nil { // LOINC: Systolic BP
		req["systolicBP"] = int(*sbp)
	} else if sbp := c.extractBPComponent(patient, "8480-6"); sbp != nil {
		req["systolicBP"] = int(*sbp)
	}

	// GCS for altered mentation
	if gcs := c.extractVitalValueInt(patient, "9269-2"); gcs != nil { // LOINC: GCS
		req["glasgowComaScale"] = *gcs
	}

	resp, err := c.callCalculator(ctx, "/api/v1/calculate/qsofa", req)
	if err != nil {
		return nil, fmt.Errorf("qSOFA calculation failed: %w", err)
	}

	var result kb8QSOFAResult
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse qSOFA response: %w", err)
	}

	return &contracts.CalculationResult{
		Name:     "qSOFA",
		Value:    float64(result.Total),
		Unit:     "points",
		Category: string(result.RiskLevel),
		Formula:  "Quick SOFA (Sepsis-3, 2016)",
		Inputs:   req,
		CalculatedAt: result.Provenance.CalculatedAt,
		Warnings:     result.Provenance.Caveats,
	}, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB8HTTPClient) callCalculator(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
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
		return nil, fmt.Errorf("KB-8 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-8 service is healthy.
func (c *KB8HTTPClient) HealthCheck(ctx context.Context) error {
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
		return fmt.Errorf("KB-8 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// Patient Data Extraction Helpers
// ============================================================================

func (c *KB8HTTPClient) extractAge(patient *contracts.PatientContext) int {
	if patient.Demographics.BirthDate == nil {
		return 0
	}
	return int(time.Since(*patient.Demographics.BirthDate).Hours() / 24 / 365)
}

func (c *KB8HTTPClient) mapGenderToSex(gender string) string {
	switch gender {
	case "male", "Male", "M":
		return "male"
	case "female", "Female", "F":
		return "female"
	default:
		return ""
	}
}

func (c *KB8HTTPClient) mapRegion(region string) string {
	switch region {
	case "IN":
		return "asia"
	case "AU", "US", "UK", "EU":
		return "global"
	default:
		return "global"
	}
}

func (c *KB8HTTPClient) extractLabValue(patient *contracts.PatientContext, loincCode string) *float64 {
	for _, lab := range patient.RecentLabResults {
		if lab.Code.Code == loincCode && lab.Value != nil {
			return &lab.Value.Value
		}
	}
	return nil
}

func (c *KB8HTTPClient) hasLabValueAbove(patient *contracts.PatientContext, loincCode string, threshold float64) bool {
	val := c.extractLabValue(patient, loincCode)
	return val != nil && *val > threshold
}

func (c *KB8HTTPClient) extractVitalValue(patient *contracts.PatientContext, loincCode string) *float64 {
	for _, vital := range patient.RecentVitalSigns {
		if vital.Code.Code == loincCode && vital.Value != nil {
			return &vital.Value.Value
		}
	}
	return nil
}

func (c *KB8HTTPClient) extractVitalValueInt(patient *contracts.PatientContext, loincCode string) *int {
	val := c.extractVitalValue(patient, loincCode)
	if val != nil {
		intVal := int(*val)
		return &intVal
	}
	return nil
}

func (c *KB8HTTPClient) extractBPComponent(patient *contracts.PatientContext, loincCode string) *float64 {
	// Look for blood pressure panel and extract component
	for _, vital := range patient.RecentVitalSigns {
		for _, comp := range vital.ComponentValues {
			if comp.Code.Code == loincCode && comp.Value != nil {
				return &comp.Value.Value
			}
		}
	}
	return nil
}

func (c *KB8HTTPClient) hasSBPAbove(patient *contracts.PatientContext, threshold float64) bool {
	sbp := c.extractVitalValue(patient, "8480-6")
	if sbp == nil {
		sbp = c.extractBPComponent(patient, "8480-6")
	}
	return sbp != nil && *sbp > threshold
}

func (c *KB8HTTPClient) hasCondition(patient *contracts.PatientContext, snomedCode string) bool {
	for _, cond := range patient.ActiveConditions {
		if cond.Code.Code == snomedCode {
			return true
		}
	}
	return false
}

func (c *KB8HTTPClient) hasMedicationClass(patient *contracts.PatientContext, drugClass string) bool {
	// This would ideally query KB-7 for medication classification
	// For now, return false as a placeholder
	// In production, this should check medication codes against drug class ValueSets
	_ = drugClass
	return false
}

// ============================================================================
// KB-8 Response Types (internal)
// ============================================================================

type kb8Provenance struct {
	CalculatorType string    `json:"calculatorType"`
	Version        string    `json:"version"`
	CalculatedAt   time.Time `json:"calculatedAt"`
	Caveats        []string  `json:"caveats"`
}

type kb8EGFRResult struct {
	Value      float64       `json:"value"`
	Unit       string        `json:"unit"`
	CKDStage   string        `json:"ckdStage"`
	Equation   string        `json:"equation"`
	Provenance kb8Provenance `json:"provenance"`
}

type kb8ASCVDResult struct {
	RiskPercent  float64       `json:"riskPercent"`
	RiskCategory string        `json:"riskCategory"`
	Provenance   kb8Provenance `json:"provenance"`
}

type kb8CHA2DS2VAScResult struct {
	Total        int           `json:"total"`
	RiskCategory string        `json:"riskCategory"`
	Provenance   kb8Provenance `json:"provenance"`
}

type kb8HASBLEDResult struct {
	Total        int           `json:"total"`
	RiskCategory string        `json:"riskCategory"`
	Provenance   kb8Provenance `json:"provenance"`
}

type kb8BMIResult struct {
	Value          float64       `json:"value"`
	Unit           string        `json:"unit"`
	CategoryWestern string       `json:"categoryWestern"`
	CategoryAsian   string       `json:"categoryAsian"`
	Provenance     kb8Provenance `json:"provenance"`
}

type kb8SOFAResult struct {
	Total      int           `json:"total"`
	RiskLevel  string        `json:"riskLevel"`
	Provenance kb8Provenance `json:"provenance"`
}

type kb8QSOFAResult struct {
	Total      int           `json:"total"`
	RiskLevel  string        `json:"riskLevel"`
	Provenance kb8Provenance `json:"provenance"`
}
