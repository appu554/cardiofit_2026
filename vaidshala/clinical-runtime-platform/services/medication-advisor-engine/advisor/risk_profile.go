// Package advisor provides the core Medication Advisor Engine.
// This file contains the V3 RiskProfile endpoint.
// V3 Architecture: Med-Advisor calculates risks ONLY, KB-19 makes decisions.
package advisor

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// V3 RISK PROFILE API
// Med-Advisor (8095) = Risk Calculator (Judge) - calculates risks
// KB-19 (8119) = Transaction Authority (Clerk) - makes decisions
// =============================================================================

// RiskProfileRequest is the V3 request for risk calculation.
// Unlike CalculateRequest, this ONLY calculates risks - NO decisions.
type RiskProfileRequest struct {
	PatientID   uuid.UUID           `json:"patient_id"`
	EncounterID uuid.UUID           `json:"encounter_id"`
	Medications []MedicationInput   `json:"medications"`
	PatientData PatientDataInput    `json:"patient_data"`
	LabValues   []LabValueInput     `json:"lab_values,omitempty"`
	Options     RiskCalculationOpts `json:"options,omitempty"`
}

// MedicationInput represents a medication for risk calculation.
type MedicationInput struct {
	RxNormCode   string  `json:"rxnorm_code"`
	DrugName     string  `json:"drug_name"`
	DrugClass    string  `json:"drug_class,omitempty"`
	DoseMg       float64 `json:"dose_mg,omitempty"`
	Unit         string  `json:"unit,omitempty"`
	Route        string  `json:"route,omitempty"`
	Frequency    string  `json:"frequency,omitempty"`
	Indication   string  `json:"indication,omitempty"`
	IsProposed   bool    `json:"is_proposed"`         // true = new med, false = current med
	IsRenalAdj   bool    `json:"is_renal_adjusted,omitempty"`
	IsHepaticAdj bool    `json:"is_hepatic_adjusted,omitempty"`
}

// PatientDataInput contains patient context for risk calculation.
type PatientDataInput struct {
	Sex                string             `json:"sex"`
	Age                int                `json:"age"`
	WeightKg           float64            `json:"weight_kg,omitempty"`
	HeightCm           float64            `json:"height_cm,omitempty"`
	BSA                float64            `json:"bsa,omitempty"`        // Body Surface Area
	EGFR               float64            `json:"egfr,omitempty"`       // Renal function
	ChildPughScore     string             `json:"child_pugh,omitempty"` // A, B, C
	IsPregnant         bool               `json:"is_pregnant"`
	PregnancyTrimester int                `json:"pregnancy_trimester,omitempty"`
	IsLactating        bool               `json:"is_lactating"`
	Conditions         []ConditionRefInput `json:"conditions,omitempty"`
	Allergies          []AllergyRefInput   `json:"allergies,omitempty"`
}

// ConditionRefInput is a reference to a patient condition.
type ConditionRefInput struct {
	ICD10Code  string `json:"icd10_code,omitempty"`
	SNOMEDCode string `json:"snomed_code,omitempty"`
	Display    string `json:"display"`
}

// AllergyRefInput is a reference to a patient allergy.
type AllergyRefInput struct {
	AllergenCode string `json:"allergen_code,omitempty"`
	AllergenType string `json:"allergen_type"` // drug, food, environmental
	Severity     string `json:"severity"`
}

// LabValueInput represents a lab value for risk calculation.
type LabValueInput struct {
	LOINCCode   string      `json:"loinc_code"`
	TestName    string      `json:"test_name"`
	Value       interface{} `json:"value"`
	Unit        string      `json:"unit"`
	CollectedAt time.Time   `json:"collected_at"`
	IsCritical  bool        `json:"is_critical,omitempty"`
}

// RiskCalculationOpts contains options for risk calculation.
type RiskCalculationOpts struct {
	IncludeDDI         bool `json:"include_ddi"`
	IncludeLabContra   bool `json:"include_lab_contra"`
	IncludePregnancy   bool `json:"include_pregnancy"`
	IncludeAllergy     bool `json:"include_allergy"`
	IncludeRenalDosing bool `json:"include_renal_dosing"`
	IncludeHepaticDose bool `json:"include_hepatic_dosing"`
	IncludeBeers       bool `json:"include_beers"`
	StrictMode         bool `json:"strict_mode"`
}

// RiskProfileResponse is the V3 response containing risk calculations.
// NOTE: No decisions or blocks - just risk information for KB-19 to process.
type RiskProfileResponse struct {
	RequestID    string    `json:"request_id"`
	PatientID    uuid.UUID `json:"patient_id"`
	EncounterID  uuid.UUID `json:"encounter_id"`
	CalculatedAt time.Time `json:"calculated_at"`

	// Risk assessments (no decisions - KB-19 decides)
	MedicationRisks []MedicationRisk `json:"medication_risks"`
	DDIRisks        []DDIRisk        `json:"ddi_risks,omitempty"`
	LabRisks        []LabRisk        `json:"lab_risks,omitempty"`
	AllergyRisks    []AllergyRisk    `json:"allergy_risks,omitempty"`

	// Dosing recommendations (no enforcement - KB-19 decides)
	DoseRecommendations []DoseRecommendation `json:"dose_recommendations,omitempty"`

	// Provenance and audit
	KBSourcesUsed []string `json:"kb_sources_used"`
	ProcessingMs  int64    `json:"processing_ms"`
}

// MedicationRisk represents a risk assessment for a single medication.
type MedicationRisk struct {
	Medication   MedicationInput `json:"medication"`
	RiskScore    float64         `json:"risk_score"`    // 0.0-1.0
	RiskCategory string          `json:"risk_category"` // LOW, MODERATE, HIGH, CRITICAL
	RiskFactors  []RiskFactor    `json:"risk_factors"`
	IsHighAlert  bool            `json:"is_high_alert"` // ISMP high-alert medication
	HasBlackBox  bool            `json:"has_black_box"`
	BlackBoxText string          `json:"black_box_text,omitempty"`
}

// RiskFactor represents a contributing factor to medication risk.
type RiskFactor struct {
	Type        string `json:"type"`        // DDI, LAB, ALLERGY, RENAL, HEPATIC, AGE, PREGNANCY
	Code        string `json:"code"`
	Description string `json:"description"`
	Severity    string `json:"severity"` // mild, moderate, severe, life-threatening
	Evidence    string `json:"evidence"` // KB source + rule ID
}

// DDIRisk represents a drug-drug interaction risk.
// Field names match KB-19's expected format for proper mapping.
type DDIRisk struct {
	Drug1Code          string `json:"drug1_code"`
	Drug1Name          string `json:"drug1_name"`
	Drug2Code          string `json:"drug2_code"`
	Drug2Name          string `json:"drug2_name"`
	InteractionID      string `json:"interaction_id"`
	Severity           string `json:"severity"`            // mild, moderate, severe, contraindicated
	InteractionType    string `json:"interaction_type"`
	Mechanism          string `json:"mechanism,omitempty"`
	ClinicalEffect     string `json:"clinical_effect"`
	ManagementStrategy string `json:"management_strategy,omitempty"`
	EvidenceLevel      string `json:"evidence_level,omitempty"`
	KBSource           string `json:"kb_source"`           // KB-5
	RuleID             string `json:"rule_id"`             // DDI rule ID
}

// LabRisk represents a lab-drug contraindication risk.
// Field names match KB-19's expected format for proper mapping.
type LabRisk struct {
	RxNormCode     string  `json:"rxnorm_code"`
	DrugName       string  `json:"drug_name"`
	LOINCCode      string  `json:"loinc_code"`
	LabName        string  `json:"lab_name"`
	CurrentValue   float64 `json:"current_value"`
	ThresholdValue float64 `json:"threshold_value"`
	ThresholdOp    string  `json:"threshold_op"`     // <, >, <=, >=, ==
	Severity       string  `json:"severity"`
	ClinicalRisk   string  `json:"clinical_risk"`
	Recommendation string  `json:"recommendation"`
	KBSource       string  `json:"kb_source"`        // KB-16
	RuleID         string  `json:"rule_id"`
}

// AllergyRisk represents an allergy risk.
// Field names match KB-19's expected format for proper mapping.
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

// DoseRecommendation represents a dosing recommendation (not enforcement).
type DoseRecommendation struct {
	Medication      MedicationInput `json:"medication"`
	RecommendedDose float64         `json:"recommended_dose"`
	DoseUnit        string          `json:"dose_unit"`
	MaxDailyDose    float64         `json:"max_daily_dose,omitempty"`
	AdjustmentType  string          `json:"adjustment_type"` // RENAL, HEPATIC, AGE, WEIGHT
	AdjustmentRatio float64         `json:"adjustment_ratio"`
	Rationale       string          `json:"rationale"`
	Evidence        string          `json:"evidence"` // KB-1 rule ID
}

// =============================================================================
// V3 RISK PROFILE METHOD
// =============================================================================

// RiskProfile calculates medication risks WITHOUT making decisions.
// This is the V3 API - KB-19 calls this, then KB-19 makes block decisions.
//
// Key difference from Calculate():
// - NO HardBlocks (KB-19 creates those)
// - NO GovernanceEvents (KB-19 creates those)
// - NO Disposition (KB-19 determines that)
// - ONLY risk assessments and dosing recommendations
func (e *MedicationAdvisorEngine) RiskProfile(ctx context.Context, req *RiskProfileRequest) (*RiskProfileResponse, error) {
	startTime := time.Now()
	requestID := uuid.New().String()

	log.Printf("[V3-RiskProfile] Request ID: %s, Patient: %s, Encounter: %s", requestID, req.PatientID, req.EncounterID)
	log.Printf("[V3-RiskProfile] Total medications received: %d", len(req.Medications))
	for i, med := range req.Medications {
		log.Printf("[V3-RiskProfile]   Med[%d]: %s (RxNorm: %s) IsProposed: %v", i, med.DrugName, med.RxNormCode, med.IsProposed)
	}
	log.Printf("[V3-RiskProfile] Lab values received: %d", len(req.LabValues))

	// Convert input types to internal types
	patientContext := e.convertToPatientContext(req.PatientData, req.LabValues)
	proposedMeds := e.convertToMedicationCodes(req.Medications, true)  // proposed meds
	currentMeds := e.convertToMedicationCodes(req.Medications, false) // current meds
	labValues := e.convertToLabValues(req.LabValues)

	log.Printf("[V3-RiskProfile] Proposed meds: %d, Current meds: %d, Labs: %d", len(proposedMeds), len(currentMeds), len(labValues))

	// Initialize response
	response := &RiskProfileResponse{
		RequestID:       requestID,
		PatientID:       req.PatientID,
		EncounterID:     req.EncounterID,
		CalculatedAt:    time.Now(),
		MedicationRisks: []MedicationRisk{},
		DDIRisks:        []DDIRisk{},
		LabRisks:        []LabRisk{},
		AllergyRisks:    []AllergyRisk{},
		KBSourcesUsed:   []string{},
	}

	// Set defaults for options
	opts := req.Options
	if !opts.IncludeDDI && !opts.IncludeLabContra && !opts.IncludeAllergy {
		// If no options specified, include everything
		opts.IncludeDDI = true
		opts.IncludeLabContra = true
		opts.IncludeAllergy = true
		opts.IncludeRenalDosing = true
		opts.IncludePregnancy = true
	}

	// 1. DDI Risk Assessment (KB-5)
	log.Printf("[V3-RiskProfile] DDI check: IncludeDDI=%v, proposedMeds=%d, currentMeds=%d", opts.IncludeDDI, len(proposedMeds), len(currentMeds))
	if opts.IncludeDDI && len(proposedMeds) > 0 && len(currentMeds) > 0 {
		ddiRisks := e.calculateDDIRisks(proposedMeds, currentMeds)
		response.DDIRisks = ddiRisks
		log.Printf("[V3-RiskProfile] DDI risks found: %d", len(ddiRisks))
		for i, ddi := range ddiRisks {
			log.Printf("[V3-RiskProfile]   DDI[%d]: %s + %s → %s (Severity: %s)", i, ddi.Drug1Name, ddi.Drug2Name, ddi.ClinicalEffect, ddi.Severity)
		}
		if len(ddiRisks) > 0 {
			response.KBSourcesUsed = appendUnique(response.KBSourcesUsed, "KB-5")
		}
	} else {
		log.Printf("[V3-RiskProfile] DDI check SKIPPED: IncludeDDI=%v, proposedMeds=%d, currentMeds=%d", opts.IncludeDDI, len(proposedMeds), len(currentMeds))
	}

	// 2. Lab Contraindication Risk Assessment (KB-16)
	log.Printf("[V3-RiskProfile] Lab check: IncludeLabContra=%v, proposedMeds=%d, labValues=%d", opts.IncludeLabContra, len(proposedMeds), len(labValues))
	if opts.IncludeLabContra && len(proposedMeds) > 0 && len(labValues) > 0 {
		labRisks := e.calculateLabRisks(proposedMeds, labValues)
		response.LabRisks = labRisks
		log.Printf("[V3-RiskProfile] Lab risks found: %d", len(labRisks))
		for i, lab := range labRisks {
			log.Printf("[V3-RiskProfile]   Lab[%d]: %s vs %s: %.2f %s (Severity: %s)", i, lab.DrugName, lab.LabName, lab.CurrentValue, lab.ThresholdOp, lab.Severity)
		}
		if len(labRisks) > 0 {
			response.KBSourcesUsed = appendUnique(response.KBSourcesUsed, "KB-16")
		}
	} else {
		log.Printf("[V3-RiskProfile] Lab check SKIPPED: IncludeLabContra=%v, proposedMeds=%d, labValues=%d", opts.IncludeLabContra, len(proposedMeds), len(labValues))
	}

	// 3. Allergy Risk Assessment (KB-4)
	log.Printf("[V3-RiskProfile] Allergy check: IncludeAllergy=%v, proposedMeds=%d, allergies=%d", opts.IncludeAllergy, len(proposedMeds), len(patientContext.Allergies))
	if opts.IncludeAllergy && len(proposedMeds) > 0 && len(patientContext.Allergies) > 0 {
		allergyRisks := e.calculateAllergyRisks(proposedMeds, patientContext.Allergies)
		response.AllergyRisks = allergyRisks
		log.Printf("[V3-RiskProfile] Allergy risks found: %d", len(allergyRisks))
		if len(allergyRisks) > 0 {
			response.KBSourcesUsed = appendUnique(response.KBSourcesUsed, "KB-4")
		}
	}

	// 4. Build overall medication risk assessments
	response.MedicationRisks = e.buildMedicationRisks(
		req.Medications,
		response.DDIRisks,
		response.LabRisks,
		response.AllergyRisks,
		patientContext,
		opts,
	)

	// 5. Dosing recommendations (KB-1)
	if opts.IncludeRenalDosing || opts.IncludeHepaticDose {
		response.DoseRecommendations = e.calculateDoseRecommendations(req.Medications, patientContext)
		if len(response.DoseRecommendations) > 0 {
			response.KBSourcesUsed = appendUnique(response.KBSourcesUsed, "KB-1")
		}
	}

	response.ProcessingMs = time.Since(startTime).Milliseconds()
	return response, nil
}

// =============================================================================
// HELPER METHODS FOR RISK PROFILE
// =============================================================================

// convertToPatientContext converts input types to internal PatientContext
func (e *MedicationAdvisorEngine) convertToPatientContext(data PatientDataInput, labs []LabValueInput) PatientContext {
	ctx := PatientContext{
		Age: data.Age,
		Sex: data.Sex,
	}

	if data.WeightKg > 0 {
		ctx.WeightKg = &data.WeightKg
	}
	if data.HeightCm > 0 {
		ctx.HeightCm = &data.HeightCm
	}

	// Convert conditions
	for _, cond := range data.Conditions {
		code := ClinicalCode{
			Code:    cond.SNOMEDCode,
			Display: cond.Display,
			System:  "SNOMED",
		}
		if cond.ICD10Code != "" {
			code.Code = cond.ICD10Code
			code.System = "ICD-10"
		}
		ctx.Conditions = append(ctx.Conditions, code)
	}

	// Add pregnancy condition if pregnant
	if data.IsPregnant {
		ctx.Conditions = append(ctx.Conditions, ClinicalCode{
			System:  "SNOMED",
			Code:    "77386006",
			Display: "Pregnancy",
		})
	}

	// Convert allergies
	for _, allergy := range data.Allergies {
		ctx.Allergies = append(ctx.Allergies, ClinicalCode{
			Code:    allergy.AllergenCode,
			Display: allergy.AllergenType + ": " + allergy.Severity,
			System:  "RxNorm",
		})
	}

	// Convert labs
	for _, lab := range labs {
		ctx.LabResults = append(ctx.LabResults, LabValue{
			Code:     lab.LOINCCode,
			Display:  lab.TestName,
			Value:    lab.Value,
			Unit:     lab.Unit,
			Critical: lab.IsCritical,
		})
	}

	// Set computed scores
	if data.EGFR > 0 {
		egfr := data.EGFR
		ctx.ComputedScores.EGFR = &egfr
	}

	return ctx
}

// convertToMedicationCodes converts medication inputs to clinical codes
func (e *MedicationAdvisorEngine) convertToMedicationCodes(meds []MedicationInput, isProposed bool) []ClinicalCode {
	var result []ClinicalCode
	for _, med := range meds {
		if med.IsProposed == isProposed {
			result = append(result, ClinicalCode{
				System:  "RxNorm",
				Code:    med.RxNormCode,
				Display: med.DrugName,
			})
		}
	}
	return result
}

// convertToLabValues converts lab inputs to internal lab values
func (e *MedicationAdvisorEngine) convertToLabValues(labs []LabValueInput) []LabValue {
	var result []LabValue
	for _, lab := range labs {
		result = append(result, LabValue{
			Code:     lab.LOINCCode,
			Display:  lab.TestName,
			Value:    lab.Value,
			Unit:     lab.Unit,
			Critical: lab.IsCritical,
		})
	}
	return result
}

// calculateDDIRisks identifies drug-drug interaction risks
func (e *MedicationAdvisorEngine) calculateDDIRisks(proposed, current []ClinicalCode) []DDIRisk {
	var risks []DDIRisk

	// Use existing DDI checking logic
	ddiBlocks := e.processDDIHardBlocks(proposed, current)

	for _, block := range ddiBlocks {
		// Convert HardBlock to DDIRisk (flat fields for KB-19 compatibility)
		risks = append(risks, DDIRisk{
			Drug1Code:          block.Medication.Code,
			Drug1Name:          block.Medication.Display,
			Drug2Code:          block.TriggerCondition.Code,
			Drug2Name:          block.TriggerCondition.Display,
			InteractionID:      block.RuleID,
			Severity:           mapBlockSeverityToDDISeverity(block.Severity),
			InteractionType:    "pharmacodynamic",
			ClinicalEffect:     block.Reason,
			ManagementStrategy: "Avoid combination or monitor closely",
			KBSource:           "KB-5",
			RuleID:             block.RuleID,
		})
	}

	return risks
}

// calculateLabRisks identifies lab-drug contraindication risks
func (e *MedicationAdvisorEngine) calculateLabRisks(proposed []ClinicalCode, labs []LabValue) []LabRisk {
	var risks []LabRisk

	// Use existing lab checking logic
	labBlocks := e.processLabHardBlocks(proposed, labs)

	for _, block := range labBlocks {
		// Convert HardBlock to LabRisk (flat fields for KB-19 compatibility)
		risks = append(risks, LabRisk{
			RxNormCode:     block.Medication.Code,
			DrugName:       block.Medication.Display,
			LOINCCode:      block.TriggerCondition.Code,
			LabName:        block.TriggerCondition.Display,
			CurrentValue:   0, // Will be populated from actual lab value
			ThresholdValue: 0, // Will be populated from rule
			ThresholdOp:    "<",
			Severity:       block.Severity,
			ClinicalRisk:   block.Reason,
			Recommendation: "Avoid or adjust dose based on lab values",
			KBSource:       "KB-16",
			RuleID:         block.RuleID,
		})
	}

	return risks
}

// calculateAllergyRisks identifies allergy risks
func (e *MedicationAdvisorEngine) calculateAllergyRisks(proposed, allergies []ClinicalCode) []AllergyRisk {
	var risks []AllergyRisk

	// Simple allergy matching - check if proposed meds match any allergies
	for _, med := range proposed {
		for _, allergy := range allergies {
			// Direct match
			if med.Code == allergy.Code {
				risks = append(risks, AllergyRisk{
					RxNormCode:      med.Code,
					DrugName:        med.Display,
					AllergenCode:    allergy.Code,
					AllergenName:    allergy.Display,
					IsCrossReactive: false,
					Severity:        "severe",
					ReactionType:    "allergic reaction",
					KBSource:        "KB-4",
					RuleID:          "ALLERGY_DIRECT",
				})
			}
		}
	}

	return risks
}

// buildMedicationRisks builds overall risk assessments for each medication
func (e *MedicationAdvisorEngine) buildMedicationRisks(
	meds []MedicationInput,
	ddiRisks []DDIRisk,
	labRisks []LabRisk,
	allergyRisks []AllergyRisk,
	patientCtx PatientContext,
	opts RiskCalculationOpts,
) []MedicationRisk {
	var results []MedicationRisk

	for _, med := range meds {
		if !med.IsProposed {
			continue // Only assess proposed medications
		}

		risk := MedicationRisk{
			Medication:   med,
			RiskScore:    0.0,
			RiskCategory: "LOW",
			RiskFactors:  []RiskFactor{},
		}

		// Add DDI risk factors
		for _, ddi := range ddiRisks {
			if ddi.Drug1Code == med.RxNormCode {
				risk.RiskFactors = append(risk.RiskFactors, RiskFactor{
					Type:        "DDI",
					Code:        ddi.Drug2Code,
					Description: ddi.ClinicalEffect,
					Severity:    ddi.Severity,
					Evidence:    ddi.KBSource + ":" + ddi.RuleID,
				})
			}
		}

		// Add lab risk factors
		for _, lab := range labRisks {
			if lab.RxNormCode == med.RxNormCode {
				risk.RiskFactors = append(risk.RiskFactors, RiskFactor{
					Type:        "LAB",
					Code:        lab.LOINCCode,
					Description: lab.ClinicalRisk,
					Severity:    lab.Severity,
					Evidence:    lab.KBSource + ":" + lab.RuleID,
				})
			}
		}

		// Add allergy risk factors
		for _, allergy := range allergyRisks {
			if allergy.RxNormCode == med.RxNormCode {
				risk.RiskFactors = append(risk.RiskFactors, RiskFactor{
					Type:        "ALLERGY",
					Code:        allergy.AllergenCode,
					Description: "Allergic reaction risk: " + allergy.ReactionType,
					Severity:    allergy.Severity,
					Evidence:    allergy.KBSource + ":" + allergy.RuleID,
				})
			}
		}

		// Calculate overall risk score
		risk.RiskScore = e.calculateOverallRiskScore(risk.RiskFactors)
		risk.RiskCategory = mapRiskScoreToCategory(risk.RiskScore)

		// Check for high-alert and black box
		risk.IsHighAlert = isHighAlertMedication(med.RxNormCode)
		risk.HasBlackBox = hasBlackBoxWarning(med.RxNormCode)
		if risk.HasBlackBox {
			risk.BlackBoxText = getBlackBoxText(med.RxNormCode)
		}

		results = append(results, risk)
	}

	return results
}

// calculateDoseRecommendations calculates dosing adjustments
func (e *MedicationAdvisorEngine) calculateDoseRecommendations(meds []MedicationInput, ctx PatientContext) []DoseRecommendation {
	var recommendations []DoseRecommendation

	for _, med := range meds {
		if !med.IsProposed {
			continue
		}

		// Check for renal adjustment needed
		if ctx.ComputedScores.EGFR != nil && *ctx.ComputedScores.EGFR > 0 && *ctx.ComputedScores.EGFR < 60 {
			adjustment := e.calculateRenalAdjustment(med, *ctx.ComputedScores.EGFR)
			if adjustment != nil {
				recommendations = append(recommendations, *adjustment)
			}
		}
	}

	return recommendations
}

// calculateRenalAdjustment calculates renal dose adjustment
func (e *MedicationAdvisorEngine) calculateRenalAdjustment(med MedicationInput, egfr float64) *DoseRecommendation {
	// Simplified renal adjustment - would use KB-1 in production
	var ratio float64
	var rationale string

	if egfr < 15 {
		ratio = 0.25
		rationale = "Severe renal impairment (eGFR < 15): 75% dose reduction recommended"
	} else if egfr < 30 {
		ratio = 0.50
		rationale = "Moderate-to-severe renal impairment (eGFR 15-30): 50% dose reduction recommended"
	} else if egfr < 45 {
		ratio = 0.75
		rationale = "Moderate renal impairment (eGFR 30-45): 25% dose reduction recommended"
	} else if egfr < 60 {
		ratio = 0.85
		rationale = "Mild renal impairment (eGFR 45-60): 15% dose reduction recommended"
	} else {
		return nil // No adjustment needed
	}

	return &DoseRecommendation{
		Medication:      med,
		RecommendedDose: med.DoseMg * ratio,
		DoseUnit:        med.Unit,
		AdjustmentType:  "RENAL",
		AdjustmentRatio: ratio,
		Rationale:       rationale,
		Evidence:        "KB-1:RENAL_ADJ",
	}
}

// calculateOverallRiskScore calculates composite risk score
func (e *MedicationAdvisorEngine) calculateOverallRiskScore(factors []RiskFactor) float64 {
	if len(factors) == 0 {
		return 0.0
	}

	var score float64
	for _, factor := range factors {
		switch factor.Severity {
		case "life-threatening", "contraindicated":
			score += 0.9
		case "severe":
			score += 0.7
		case "moderate":
			score += 0.4
		case "mild":
			score += 0.2
		}
	}

	// Normalize to 0-1 range
	if score > 1.0 {
		score = 1.0
	}
	return score
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func mapBlockSeverityToDDISeverity(severity string) string {
	// Case-insensitive comparison since KB-5 returns uppercase (e.g., "SEVERE")
	switch strings.ToLower(severity) {
	case "absolute", "life_threatening", "life-threatening":
		return "contraindicated"
	case "severe", "major": // KB-5 uses "SEVERE" or "major" for severe DDIs
		return "severe"
	case "moderate":
		return "moderate"
	default:
		return "mild"
	}
}

func mapRiskScoreToCategory(score float64) string {
	if score >= 0.8 {
		return "CRITICAL"
	} else if score >= 0.6 {
		return "HIGH"
	} else if score >= 0.3 {
		return "MODERATE"
	}
	return "LOW"
}

func appendUnique(slice []string, item string) []string {
	for _, s := range slice {
		if s == item {
			return slice
		}
	}
	return append(slice, item)
}

// Placeholder functions for black box and high-alert checks
// In production, these would query KB-4

func isHighAlertMedication(rxnorm string) bool {
	// ISMP high-alert medications
	highAlert := map[string]bool{
		"161":    true, // Warfarin
		"10582":  true, // Insulin
		"6809":   true, // Metformin (with renal considerations)
		"1191":   true, // Aspirin (high-dose)
		"7052":   true, // Morphine
		"3638":   true, // Heparin
		"5521":   true, // Hydromorphone
	}
	return highAlert[rxnorm]
}

func hasBlackBoxWarning(rxnorm string) bool {
	// Drugs with FDA black box warnings
	blackBox := map[string]bool{
		"161":    true, // Warfarin - bleeding risk
		"29046":  true, // Pioglitazone - CHF risk
		"5640":   true, // Ibuprofen - CV risk
		"36567":  true, // Rosiglitazone - CV risk
		"1151131": true, // Suvorexant - CNS depression
	}
	return blackBox[rxnorm]
}

func getBlackBoxText(rxnorm string) string {
	blackBoxTexts := map[string]string{
		"161":    "BOXED WARNING: BLEEDING RISK. Can cause major or fatal bleeding. Monitor for signs and symptoms of bleeding.",
		"29046":  "BOXED WARNING: May cause or exacerbate congestive heart failure in some patients.",
		"5640":   "BOXED WARNING: NSAIDs cause an increased risk of serious cardiovascular thrombotic events.",
		"36567":  "BOXED WARNING: May cause or exacerbate congestive heart failure. Not recommended in patients with symptomatic heart failure.",
	}
	return blackBoxTexts[rxnorm]
}
