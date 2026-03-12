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

// =============================================================================
// MEDICATION ADVISOR CLIENT
// Calls Med-Advisor's /risk-profile endpoint to get risk assessments
// V3 Architecture: Med-Advisor = Judge (calculates risks), KB-19 = Clerk (makes decisions)
// =============================================================================

// MedicationAdvisorClient is the HTTP client for the Medication Advisor Engine.
type MedicationAdvisorClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewMedicationAdvisorClient creates a new MedicationAdvisorClient.
func NewMedicationAdvisorClient(baseURL string, timeout time.Duration, log *logrus.Entry) *MedicationAdvisorClient {
	return &MedicationAdvisorClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.WithField("client", "medication-advisor"),
	}
}

// =============================================================================
// REQUEST TYPES (matching Med-Advisor's RiskProfileRequest)
// =============================================================================

// RiskProfileRequest is the request to get a medication risk profile.
type RiskProfileRequest struct {
	PatientID   uuid.UUID            `json:"patient_id"`
	EncounterID uuid.UUID            `json:"encounter_id"`
	Medications []MedicationInput    `json:"medications"`
	PatientData PatientDataInput     `json:"patient_data"`
	LabValues   []LabValueInput      `json:"lab_values,omitempty"`
	Options     RiskCalculationOpts  `json:"options,omitempty"`
}

// MedicationInput represents a single medication for risk assessment.
type MedicationInput struct {
	RxNormCode       string  `json:"rxnorm_code"`
	DrugName         string  `json:"drug_name"`
	DoseValue        float64 `json:"dose_value,omitempty"`
	DoseUnit         string  `json:"dose_unit,omitempty"`
	Route            string  `json:"route,omitempty"`
	Frequency        string  `json:"frequency,omitempty"`
	IsProposed       bool    `json:"is_proposed"`
	RequiresDoseAdj  bool    `json:"requires_dose_adjustment,omitempty"`
}

// PatientDataInput contains patient context for risk calculations.
// Matches Med-Advisor's expected PatientDataInput format
type PatientDataInput struct {
	Sex                string              `json:"sex"`
	Age                int                 `json:"age"`
	WeightKg           float64             `json:"weight_kg,omitempty"`
	HeightCm           float64             `json:"height_cm,omitempty"`
	EGFR               float64             `json:"egfr,omitempty"`
	ChildPughScore     string              `json:"child_pugh,omitempty"`
	IsPregnant         bool                `json:"is_pregnant"`
	PregnancyTrimester int                 `json:"pregnancy_trimester,omitempty"`
	IsLactating        bool                `json:"is_lactating"`
	Conditions         []ConditionRefInput `json:"conditions,omitempty"`
	Allergies          []AllergyRefInput   `json:"allergies,omitempty"`
}

// ConditionRefInput represents a patient condition for risk calculations
// Matches Med-Advisor's expected ConditionRefInput format
type ConditionRefInput struct {
	ICD10Code  string `json:"icd10_code,omitempty"`
	SNOMEDCode string `json:"snomed_code,omitempty"`
	Display    string `json:"display"`
}

// AllergyRefInput represents a patient allergy for risk calculations
// Matches Med-Advisor's expected AllergyRefInput format
type AllergyRefInput struct {
	AllergenCode string `json:"allergen_code,omitempty"`
	AllergenType string `json:"allergen_type"` // drug, food, environmental
	Severity     string `json:"severity"`
}

// LabValueInput represents a single lab result.
type LabValueInput struct {
	LOINCCode    string    `json:"loinc_code"`
	Value        float64   `json:"value"`
	Unit         string    `json:"unit"`
	ResultDate   time.Time `json:"result_date,omitempty"`
	IsCritical   bool      `json:"is_critical,omitempty"`
}

// RiskCalculationOpts controls which risk types to calculate.
type RiskCalculationOpts struct {
	IncludeDDI        bool `json:"include_ddi"`
	IncludeLab        bool `json:"include_lab"`
	IncludeAllergy    bool `json:"include_allergy"`
	IncludeDosing     bool `json:"include_dosing"`
	IncludeBlackBox   bool `json:"include_black_box"`
	IncludeHighAlert  bool `json:"include_high_alert"`
}

// =============================================================================
// RESPONSE TYPES (matching Med-Advisor's RiskProfileResponse)
// =============================================================================

// RiskProfileResponse is the response from the risk profile endpoint.
type RiskProfileResponse struct {
	RequestID           string               `json:"request_id"`
	PatientID           uuid.UUID            `json:"patient_id"`
	EncounterID         uuid.UUID            `json:"encounter_id"`
	CalculatedAt        time.Time            `json:"calculated_at"`

	// Risk assessments (no decisions - KB-19 decides)
	MedicationRisks     []MedicationRisk     `json:"medication_risks"`
	DDIRisks            []DDIRisk            `json:"ddi_risks,omitempty"`
	LabRisks            []LabRisk            `json:"lab_risks,omitempty"`
	AllergyRisks        []AllergyRisk        `json:"allergy_risks,omitempty"`

	// Dosing recommendations (no enforcement - KB-19 decides)
	DoseRecommendations []DoseRecommendation `json:"dose_recommendations,omitempty"`

	// Provenance and audit
	KBSourcesUsed       []string             `json:"kb_sources_used"`
	ProcessingMs        int64                `json:"processing_ms"`
}

// MedicationRisk represents the aggregate risk for a single medication.
type MedicationRisk struct {
	RxNormCode      string       `json:"rxnorm_code"`
	DrugName        string       `json:"drug_name"`
	OverallRisk     float64      `json:"overall_risk"`      // 0.0 - 1.0
	RiskCategory    string       `json:"risk_category"`     // LOW, MODERATE, HIGH, CRITICAL
	RiskFactors     []RiskFactor `json:"risk_factors"`
	IsHighAlert     bool         `json:"is_high_alert"`
	HasBlackBoxWarn bool         `json:"has_black_box_warning"`
}

// RiskFactor represents a single risk factor contributing to medication risk.
type RiskFactor struct {
	Type        string `json:"type"`        // DDI, LAB, ALLERGY, RENAL, HEPATIC, AGE, PREGNANCY
	Severity    string `json:"severity"`    // mild, moderate, severe, life-threatening
	Description string `json:"description"`
	KBSource    string `json:"kb_source"`
	RuleID      string `json:"rule_id"`
}

// DDIRisk represents a drug-drug interaction risk.
type DDIRisk struct {
	Drug1Code          string `json:"drug1_code"`
	Drug1Name          string `json:"drug1_name"`
	Drug2Code          string `json:"drug2_code"`
	Drug2Name          string `json:"drug2_name"`
	Severity           string `json:"severity"`           // mild, moderate, severe, contraindicated
	InteractionType    string `json:"interaction_type"`
	Mechanism          string `json:"mechanism"`
	ClinicalEffect     string `json:"clinical_effect"`
	ManagementStrategy string `json:"management_strategy"`
	EvidenceLevel      string `json:"evidence_level"`
	KBSource           string `json:"kb_source"`
	RuleID             string `json:"rule_id"`
}

// LabRisk represents a lab-based contraindication risk.
type LabRisk struct {
	RxNormCode       string  `json:"rxnorm_code"`
	DrugName         string  `json:"drug_name"`
	LOINCCode        string  `json:"loinc_code"`
	LabName          string  `json:"lab_name"`
	CurrentValue     float64 `json:"current_value"`
	ThresholdValue   float64 `json:"threshold_value"`
	ThresholdOp      string  `json:"threshold_op"`       // <, >, <=, >=, ==
	Severity         string  `json:"severity"`
	ClinicalRisk     string  `json:"clinical_risk"`
	Recommendation   string  `json:"recommendation"`
	KBSource         string  `json:"kb_source"`
	RuleID           string  `json:"rule_id"`
}

// AllergyRisk represents an allergy-based risk.
type AllergyRisk struct {
	RxNormCode      string `json:"rxnorm_code"`
	DrugName        string `json:"drug_name"`
	AllergenCode    string `json:"allergen_code"`
	AllergenName    string `json:"allergen_name"`
	IsCrossReactive bool   `json:"is_cross_reactive"`
	Severity        string `json:"severity"`
	ReactionType    string `json:"reaction_type"`
	KBSource        string `json:"kb_source"`
	RuleID          string `json:"rule_id"`
}

// DoseRecommendation represents a dosing adjustment recommendation.
type DoseRecommendation struct {
	RxNormCode      string  `json:"rxnorm_code"`
	DrugName        string  `json:"drug_name"`
	OriginalDose    float64 `json:"original_dose"`
	AdjustedDose    float64 `json:"adjusted_dose"`
	DoseUnit        string  `json:"dose_unit"`
	AdjustmentType  string  `json:"adjustment_type"`   // RENAL, HEPATIC, AGE, WEIGHT
	AdjustmentRatio float64 `json:"adjustment_ratio"`  // 0.0 - 1.0
	Reason          string  `json:"reason"`
	KBSource        string  `json:"kb_source"`
	RuleID          string  `json:"rule_id"`
}

// =============================================================================
// CLIENT METHODS
// =============================================================================

// GetRiskProfile calls Med-Advisor's /risk-profile endpoint to get risk assessments.
// This is the V3 pattern: Med-Advisor calculates risks, KB-19 makes decisions.
func (c *MedicationAdvisorClient) GetRiskProfile(ctx context.Context, req *RiskProfileRequest) (*RiskProfileResponse, error) {
	c.log.WithFields(logrus.Fields{
		"patient_id":      req.PatientID,
		"medication_count": len(req.Medications),
	}).Info("Getting risk profile from Med-Advisor")

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/risk-profile", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.WithError(err).Error("Med-Advisor request failed")
		return nil, fmt.Errorf("med-advisor request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.log.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(respBody),
		}).Error("Med-Advisor returned error")
		return nil, fmt.Errorf("med-advisor error: %s", string(respBody))
	}

	var result RiskProfileResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"medication_risks": len(result.MedicationRisks),
		"ddi_risks":        len(result.DDIRisks),
		"lab_risks":        len(result.LabRisks),
		"allergy_risks":    len(result.AllergyRisks),
		"processing_ms":    result.ProcessingMs,
	}).Info("Risk profile received from Med-Advisor")

	return &result, nil
}

// Health checks if Med-Advisor is healthy.
func (c *MedicationAdvisorClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("med-advisor unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

// =============================================================================
// RISK TO HARD BLOCK CONVERSION
// KB-19's responsibility: Convert risk assessments into governance decisions
// =============================================================================

// RiskSeverityThresholds defines when risks become hard blocks
type RiskSeverityThresholds struct {
	DDISeverities     []string // severities that trigger hard blocks
	LabSeverities     []string // severities that trigger hard blocks
	AllergySeverities []string // severities that trigger hard blocks
	OverallRiskCutoff float64  // overall risk score that triggers hard block
}

// DefaultRiskThresholds returns conservative default thresholds
func DefaultRiskThresholds() RiskSeverityThresholds {
	return RiskSeverityThresholds{
		DDISeverities:     []string{"contraindicated", "severe"},
		LabSeverities:     []string{"contraindicated", "severe", "life-threatening"},
		AllergySeverities: []string{"severe", "life-threatening"},
		OverallRiskCutoff: 0.8, // CRITICAL category
	}
}

