// Package integration provides the bridge between Go CandidateBuilder and Rust JIT Safety Engine
package integration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/hashicorp/go-retryablehttp"
	"github.com/sirupsen/logrus"

	candidatebuilder "flow2-go-engine/internal/clinical-intelligence/candidate-builder"
	"flow2-go-engine/internal/models"
)

// ==================== Enhanced JIT Safety Service Client ====================

// EnhancedJITSafetyClient provides interface to the Rust JIT Safety Engine with rich data transformation
type EnhancedJITSafetyClient struct {
	baseURL    string
	httpClient *retryablehttp.Client
	logger     *logrus.Logger
	timeout    time.Duration
}

// NewEnhancedJITSafetyClient creates a new enhanced JIT safety client
func NewEnhancedJITSafetyClient(baseURL string, logger *logrus.Logger) *EnhancedJITSafetyClient {
	retryClient := retryablehttp.NewClient()
	retryClient.RetryMax = 3
	retryClient.RetryWaitMin = 100 * time.Millisecond
	retryClient.RetryWaitMax = 1 * time.Second
	retryClient.Logger = nil // Use our own logger

	return &EnhancedJITSafetyClient{
		baseURL:    baseURL,
		httpClient: retryClient,
		logger:     logger,
		timeout:    5 * time.Second,
	}
}

// ==================== Enhanced Request/Response Models ====================

// EnhancedJITSafetyRequest represents the request to the enhanced JIT safety engine
type EnhancedJITSafetyRequest struct {
	Candidate      DrugCandidateDTO  `json:"candidate"`
	PatientContext PatientContextDTO `json:"patient_context"`
	RequestID      string            `json:"request_id"`
}

// DrugCandidateDTO represents a drug candidate for JIT checking
type DrugCandidateDTO struct {
	DrugID           string            `json:"drug_id"`
	Name             string            `json:"name"`
	GenericName      string            `json:"generic_name"`
	ProposedDoseMg   float64           `json:"proposed_dose_mg"`
	ProposedDoseUnit string            `json:"proposed_dose_unit"`
	Route            string            `json:"route"`
	Frequency        FrequencyDTO      `json:"frequency"`
	DurationDays     *int              `json:"duration_days,omitempty"`
	Formulation      *string           `json:"formulation,omitempty"`
	Provenance       map[string]string `json:"provenance"`
}

// FrequencyDTO represents dosing frequency
type FrequencyDTO struct {
	TimesPerDay int     `json:"times_per_day"`
	Schedule    *string `json:"schedule,omitempty"`
	WithFood    *bool   `json:"with_food,omitempty"`
}

// PatientContextDTO represents patient clinical context
type PatientContextDTO struct {
	PatientID         string                     `json:"patient_id"`
	AgeYears          int                        `json:"age_years"`
	Sex               string                     `json:"sex"`
	WeightKg          float64                    `json:"weight_kg"`
	HeightCm          float64                    `json:"height_cm"`
	PregnancyStatus   PregnancyStatusDTO         `json:"pregnancy_status"`
	Breastfeeding     bool                       `json:"breastfeeding"`
	Labs              LabResultsDTO              `json:"labs"`
	Conditions        []string                   `json:"conditions"`
	RecentProcedures  []ProcedureDTO             `json:"recent_procedures"`
	ActiveMedications []ActiveMedicationDTO      `json:"active_medications"`
	Allergies         []AllergyDTO               `json:"allergies"`
	Pharmacogenomics  *PharmacogenomicProfileDTO `json:"pharmacogenomics,omitempty"`
	KBVersions        map[string]string          `json:"kb_versions"`
	Timestamp         time.Time                  `json:"timestamp"`
}

// PregnancyStatusDTO represents pregnancy status
type PregnancyStatusDTO struct {
	Status    string `json:"status"` // "not_pregnant", "pregnant", "unknown"
	Trimester *int   `json:"trimester,omitempty"`
}

// LabResultsDTO represents laboratory results
type LabResultsDTO struct {
	EgfrMlMin           *LabValueDTO `json:"egfr_ml_min,omitempty"`
	SerumCreatinineMgDl *LabValueDTO `json:"serum_creatinine_mg_dl,omitempty"`
	SerumPotassiumMmolL *LabValueDTO `json:"serum_potassium_mmol_l,omitempty"`
	SerumSodiumMmolL    *LabValueDTO `json:"serum_sodium_mmol_l,omitempty"`
	AltUL               *LabValueDTO `json:"alt_u_l,omitempty"`
	AstUL               *LabValueDTO `json:"ast_u_l,omitempty"`
	HbA1cPercent        *LabValueDTO `json:"hba1c_percent,omitempty"`
	QtcMs               *LabValueDTO `json:"qtc_ms,omitempty"`
}

// LabValueDTO represents a laboratory value with timestamp
type LabValueDTO struct {
	Value     float64   `json:"value"`
	Timestamp time.Time `json:"timestamp"`
	Unit      string    `json:"unit"`
}

// ProcedureDTO represents a recent procedure
type ProcedureDTO struct {
	Name           string    `json:"name"`
	Code           string    `json:"code"`
	Date           time.Time `json:"date"`
	RequiresNPO    bool      `json:"requires_npo"`
	AnesthesiaType *string   `json:"anesthesia_type,omitempty"`
}

// ActiveMedicationDTO represents an active medication
type ActiveMedicationDTO struct {
	DrugID            string       `json:"drug_id"`
	Name              string       `json:"name"`
	GenericName       string       `json:"generic_name"`
	DoseMg            float64      `json:"dose_mg"`
	Route             string       `json:"route"`
	Frequency         FrequencyDTO `json:"frequency"`
	StartDate         time.Time    `json:"start_date"`
	LastTaken         *time.Time   `json:"last_taken,omitempty"`
	TherapeuticClass  []string     `json:"therapeutic_class"`
	MechanismOfAction []string     `json:"mechanism_of_action"`
}

// AllergyDTO represents an allergy
type AllergyDTO struct {
	Allergen     string `json:"allergen"`
	ReactionType string `json:"reaction_type"`
	Severity     string `json:"severity"`
}

// PharmacogenomicProfileDTO represents pharmacogenomic data
type PharmacogenomicProfileDTO struct {
	CYP2D6Status  *string `json:"cyp2d6_status,omitempty"`
	CYP2C19Status *string `json:"cyp2c19_status,omitempty"`
	CYP2C9Status  *string `json:"cyp2c9_status,omitempty"`
	CYP3A4Status  *string `json:"cyp3a4_status,omitempty"`
	SLCO1B1Status *string `json:"slco1b1_status,omitempty"`
	HLAB5701      *bool   `json:"hla_b5701,omitempty"`
	TPMTStatus    *string `json:"tpmt_status,omitempty"`
}

// EnhancedJITSafetyResponse represents the response from the enhanced JIT safety engine
type EnhancedJITSafetyResponse struct {
	Action          SafetyActionDTO             `json:"action"`
	Score           SafetyScoreDTO              `json:"score"`
	Findings        []SafetyFindingDTO          `json:"findings"`
	Recommendations []ClinicalRecommendationDTO `json:"recommendations"`
	AuditTrail      AuditTrailDTO               `json:"audit_trail"`
}

// SafetyActionDTO represents the recommended action
type SafetyActionDTO struct {
	Type               string   `json:"type"`
	Parameters         []string `json:"parameters,omitempty"`
	RecommendedDoseMg  *float64 `json:"recommended_dose_mg,omitempty"`
	Reason             *string  `json:"reason,omitempty"`
	Urgency            *string  `json:"urgency,omitempty"`
	AlternativeDrugIDs []string `json:"alternative_drug_ids,omitempty"`
	Specialty          *string  `json:"specialty,omitempty"`
}

// SafetyScoreDTO represents safety scoring
type SafetyScoreDTO struct {
	Overall    float64            `json:"overall"`
	Components map[string]float64 `json:"components"`
}

// SafetyFindingDTO represents a safety finding
type SafetyFindingDTO struct {
	FindingID            string            `json:"finding_id"`
	Category             string            `json:"category"`
	Severity             string            `json:"severity"`
	Code                 string            `json:"code"`
	Message              string            `json:"message"`
	ClinicalSignificance string            `json:"clinical_significance"`
	EvidenceLevel        string            `json:"evidence_level"`
	References           []string          `json:"references"`
	Details              map[string]string `json:"details"`
}

// ClinicalRecommendationDTO represents a clinical recommendation
type ClinicalRecommendationDTO struct {
	RecommendationType string  `json:"recommendation_type"`
	Description        string  `json:"description"`
	Priority           string  `json:"priority"`
	Timing             *string `json:"timing,omitempty"`
}

// AuditTrailDTO represents audit information
type AuditTrailDTO struct {
	RequestID        string            `json:"request_id"`
	Timestamp        time.Time         `json:"timestamp"`
	ChecksPerformed  []string          `json:"checks_performed"`
	KBVersions       map[string]string `json:"kb_versions"`
	ProcessingTimeMs int64             `json:"processing_time_ms"`
}

// ==================== Main Integration Function ====================

// RunEnhancedJITSafetyCheck performs enhanced JIT safety verification
func (c *EnhancedJITSafetyClient) RunEnhancedJITSafetyCheck(
	ctx context.Context,
	candidate candidatebuilder.CandidateProposal,
	patientContext models.PatientContext,
	proposedDoseMg float64,
	requestID string,
) (*models.SafetyVerifiedProposal, error) {
	// Convert to enhanced DTOs
	request := c.buildEnhancedRequest(candidate, patientContext, proposedDoseMg, requestID)

	// Marshal request
	reqBody, err := json.Marshal(request)
	if err != nil {
		c.logger.WithError(err).Error("Failed to marshal enhanced JIT request")
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := retryablehttp.NewRequestWithContext(
		ctx,
		"POST",
		fmt.Sprintf("%s/api/v1/safety/enhanced-jit-check", c.baseURL),
		bytes.NewReader(reqBody),
	)
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-Request-ID", requestID)

	// Add timeout
	ctx, cancel := context.WithTimeout(ctx, c.timeout)
	defer cancel()

	// Execute request
	resp, err := c.httpClient.Do(httpReq.WithContext(ctx))
	if err != nil {
		c.logger.WithFields(logrus.Fields{
			"error":   err,
			"drug_id": candidate.MedicationCode,
		}).Error("Enhanced JIT safety check request failed")
		return nil, fmt.Errorf("execute request: %w", err)
	}
	defer resp.Body.Close()

	// Read response
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	// Check status code
	if resp.StatusCode != http.StatusOK {
		c.logger.WithFields(logrus.Fields{
			"status_code": resp.StatusCode,
			"response":    string(respBody),
		}).Error("Enhanced JIT safety check returned error")
		return nil, fmt.Errorf("JIT check failed with status %d: %s", resp.StatusCode, string(respBody))
	}

	// Parse response
	var jitResp EnhancedJITSafetyResponse
	if err := json.Unmarshal(respBody, &jitResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	c.logger.WithFields(logrus.Fields{
		"drug_id":      candidate.MedicationCode,
		"action":       jitResp.Action.Type,
		"safety_score": jitResp.Score.Overall,
		"findings":     len(jitResp.Findings),
	}).Info("Enhanced JIT safety check completed")

	// Convert to SafetyVerifiedProposal
	safetyVerified := c.interpretEnhancedSafetyAction(&jitResp, candidate)

	return &safetyVerified, nil
}

// ==================== Data Transformation Functions ====================

// buildEnhancedRequest converts Go models to enhanced JIT Safety request
func (c *EnhancedJITSafetyClient) buildEnhancedRequest(
	candidate candidatebuilder.CandidateProposal,
	patientContext models.PatientContext,
	proposedDoseMg float64,
	requestID string,
) EnhancedJITSafetyRequest {
	// Build drug candidate DTO
	drugCandidate := DrugCandidateDTO{
		DrugID:           candidate.MedicationCode,
		Name:             candidate.MedicationName,
		GenericName:      candidate.GenericName,
		ProposedDoseMg:   proposedDoseMg,
		ProposedDoseUnit: "mg",
		Route:            candidate.Route,
		Frequency: FrequencyDTO{
			TimesPerDay: 1, // Would be calculated based on drug
		},
		Provenance: map[string]string{
			"source":    "candidate_builder",
			"version":   "1.0.0",
			"timestamp": time.Now().Format(time.RFC3339),
		},
	}

	// Build patient context DTO
	patientContextDTO := PatientContextDTO{
		PatientID:    requestID, // Using request ID as patient ID for now
		AgeYears:     int(patientContext.Demographics.Age),
		Sex:          c.mapBiologicalSex(patientContext.Demographics.Gender),
		WeightKg:     patientContext.Demographics.Weight,
		HeightCm:     patientContext.Demographics.Height,
		PregnancyStatus: PregnancyStatusDTO{
			Status: c.mapPregnancyStatus(patientContext.Demographics.IsPregnant),
		},
		Breastfeeding:     false, // Not available in current model
		Labs:              c.buildLabResults(patientContext.LabResults),
		Conditions:        c.extractConditions(patientContext.Conditions),
		ActiveMedications: c.buildActiveMedications(patientContext.ActiveMedications),
		Allergies:         c.buildAllergies(patientContext.Allergies),
		KBVersions: map[string]string{
			"kb_drug_rules": "v1.5.2",
			"kb_ddi":        "v1.3.0",
		},
		Timestamp: time.Now(),
	}

	return EnhancedJITSafetyRequest{
		Candidate:      drugCandidate,
		PatientContext: patientContextDTO,
		RequestID:      requestID,
	}
}

// mapBiologicalSex converts gender to biological sex
func (c *EnhancedJITSafetyClient) mapBiologicalSex(gender string) string {
	switch strings.ToLower(gender) {
	case "m", "male":
		return "Male"
	case "f", "female":
		return "Female"
	default:
		return "Other"
	}
}

// mapPregnancyStatus converts pregnancy boolean to status
func (c *EnhancedJITSafetyClient) mapPregnancyStatus(isPregnant bool) string {
	if isPregnant {
		return "pregnant"
	}
	return "not_pregnant"
}

// buildLabResults converts lab results to DTO format
func (c *EnhancedJITSafetyClient) buildLabResults(labs models.LabResults) LabResultsDTO {
	labsDTO := LabResultsDTO{}

	if labs.EGFR > 0 {
		labsDTO.EgfrMlMin = &LabValueDTO{
			Value:     labs.EGFR,
			Timestamp: time.Now().Add(-24 * time.Hour),
			Unit:      "mL/min/1.73m²",
		}
	}

	if labs.ALT > 0 {
		labsDTO.AltUL = &LabValueDTO{
			Value:     labs.ALT,
			Timestamp: time.Now().Add(-24 * time.Hour),
			Unit:      "U/L",
		}
	}

	if labs.AST > 0 {
		labsDTO.AstUL = &LabValueDTO{
			Value:     labs.AST,
			Timestamp: time.Now().Add(-24 * time.Hour),
			Unit:      "U/L",
		}
	}

	if labs.Potassium > 0 {
		labsDTO.SerumPotassiumMmolL = &LabValueDTO{
			Value:     labs.Potassium,
			Timestamp: time.Now().Add(-24 * time.Hour),
			Unit:      "mmol/L",
		}
	}

	if labs.HbA1c > 0 {
		labsDTO.HbA1cPercent = &LabValueDTO{
			Value:     labs.HbA1c,
			Timestamp: time.Now().Add(-24 * time.Hour),
			Unit:      "%",
		}
	}

	return labsDTO
}

// extractConditions converts conditions to ICD-10 codes
func (c *EnhancedJITSafetyClient) extractConditions(conditions []models.Condition) []string {
	conditionCodes := make([]string, len(conditions))
	for i, condition := range conditions {
		conditionCodes[i] = condition.Code
	}
	return conditionCodes
}

// buildActiveMedications converts active medications to DTO format
func (c *EnhancedJITSafetyClient) buildActiveMedications(meds []models.ActiveMedication) []ActiveMedicationDTO {
	activeMeds := make([]ActiveMedicationDTO, len(meds))

	for i, med := range meds {
		activeMeds[i] = ActiveMedicationDTO{
			DrugID:      med.MedicationCode,
			Name:        med.MedicationCode, // Using code as name for now
			GenericName: med.MedicationCode,
			DoseMg:      med.DoseAmount,
			Route:       "Oral", // Default route
			Frequency: FrequencyDTO{
				TimesPerDay: 24 / med.FrequencyHours, // Convert hours to times per day
			},
			StartDate:         time.Now().Add(-30 * 24 * time.Hour), // Assume started 30 days ago
			TherapeuticClass:  []string{med.TherapeuticClass},
			MechanismOfAction: []string{}, // Not available in current model
		}
	}

	return activeMeds
}

// buildAllergies converts allergies to DTO format
func (c *EnhancedJITSafetyClient) buildAllergies(allergies []models.Allergy) []AllergyDTO {
	allergyDTOs := make([]AllergyDTO, len(allergies))

	for i, allergy := range allergies {
		allergyDTOs[i] = AllergyDTO{
			Allergen:     allergy.AllergenCode,
			ReactionType: "Unknown", // Not available in current model
			Severity:     allergy.Severity,
		}
	}

	return allergyDTOs
}

// ==================== Safety Action Interpretation ====================

// interpretEnhancedSafetyAction converts enhanced JIT safety response to SafetyVerifiedProposal
func (c *EnhancedJITSafetyClient) interpretEnhancedSafetyAction(
	response *EnhancedJITSafetyResponse,
	candidate candidatebuilder.CandidateProposal,
) models.SafetyVerifiedProposal {
	// Convert reasons from findings
	reasons := make([]models.Reason, len(response.Findings))
	for i, finding := range response.Findings {
		severity := c.mapFindingSeverity(finding.Severity)
		reasons[i] = models.Reason{
			Code:     finding.Code,
			Severity: severity,
			Message:  finding.Message,
			Evidence: finding.References,
			RuleID:   finding.FindingID,
		}
	}

	// Convert DDI warnings
	ddiWarnings := make([]models.DdiFlag, 0)
	for _, finding := range response.Findings {
		if finding.Category == "DrugDrugInteraction" {
			ddiWarnings = append(ddiWarnings, models.DdiFlag{
				WithDrugID: finding.Details["interacting_drug"],
				Severity:   c.mapDDISeverity(finding.Severity),
				Action:     finding.ClinicalSignificance,
				Code:       finding.Code,
				RuleID:     finding.FindingID,
			})
		}
	}

	// Determine final dose based on action
	finalDose := models.ProposedDose{
		DrugID:    candidate.MedicationCode,
		DoseMg:    10.0, // Default dose - would be from candidate
		Route:     "po",
		IntervalH: 24,
	}

	if response.Action.RecommendedDoseMg != nil {
		finalDose.DoseMg = *response.Action.RecommendedDoseMg
	}

	// Determine action string
	action := c.mapSafetyAction(response.Action.Type)

	// Create provenance
	provenance := models.Provenance{
		EngineVersion: "enhanced-jit-1.0.0",
		KBVersions:    response.AuditTrail.KBVersions,
		EvaluationTrace: []models.EvalStep{
			{
				RuleID: "enhanced_jit_check",
				Result: "completed",
			},
		},
	}

	return models.SafetyVerifiedProposal{
		Original:      candidate,
		SafetyScore:   response.Score.Overall,
		FinalDose:     finalDose,
		SafetyReasons: reasons,
		DDIWarnings:   ddiWarnings,
		Action:        action,
		JITProvenance: provenance,
		ProcessedAt:   time.Now(),
	}
}

// mapFindingSeverity converts finding severity to our format
func (c *EnhancedJITSafetyClient) mapFindingSeverity(severity string) string {
	switch severity {
	case "Info":
		return "info"
	case "Warning":
		return "warn"
	case "Major":
		return "error"
	case "Critical", "Contraindicated":
		return "blocker"
	default:
		return "info"
	}
}

// mapDDISeverity converts DDI severity to our format
func (c *EnhancedJITSafetyClient) mapDDISeverity(severity string) string {
	switch severity {
	case "Info":
		return "minor"
	case "Warning":
		return "moderate"
	case "Major":
		return "major"
	case "Critical", "Contraindicated":
		return "contraindicated"
	default:
		return "minor"
	}
}

// mapSafetyAction converts safety action type to our format
func (c *EnhancedJITSafetyClient) mapSafetyAction(actionType string) string {
	switch actionType {
	case "Proceed", "ProceedWithMonitoring":
		return "CanProceed"
	case "AdjustDose":
		return "RequiresReview"
	case "HoldForClinician", "RequireSpecialistReview":
		return "RequiresReview"
	case "AbortAndSwitch":
		return "Contraindicated"
	default:
		return "RequiresReview"
	}
}
