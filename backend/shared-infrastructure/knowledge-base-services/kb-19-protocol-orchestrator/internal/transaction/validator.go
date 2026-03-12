// Package transaction provides transaction validation for KB-19.
// Validator MOVED FROM: medication-advisor-engine/advisor/engine.go
// as part of V3 architecture refactoring.
package transaction

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// =============================================================================
// KB-5 CLIENT INTERFACE
// Allows injection of KB-5 DDI client for production use
// =============================================================================

// KB5DDIChecker interface for DDI checking via KB-5
type KB5DDIChecker interface {
	CheckInteractions(ctx context.Context, drugCodes []string) (*KB5DDIResult, error)
}

// =============================================================================
// V3: RISK PROFILE PROVIDER INTERFACE
// Med-Advisor = Judge (calculates risks), KB-19 = Clerk (makes decisions)
// =============================================================================

// RiskProfileProvider interface for getting risk profiles from Med-Advisor
type RiskProfileProvider interface {
	GetRiskProfile(ctx context.Context, req *RiskProfileRequest) (*RiskProfile, error)
}

// RiskProfileRequest contains the data needed to calculate risks
type RiskProfileRequest struct {
	PatientID   uuid.UUID          `json:"patient_id"`
	EncounterID uuid.UUID          `json:"encounter_id"`
	Medications []MedicationInput  `json:"medications"`
	PatientData PatientDataInput   `json:"patient_data"`
	LabValues   []LabValueInput    `json:"lab_values,omitempty"`
}

// MedicationInput represents a medication for risk assessment
type MedicationInput struct {
	RxNormCode      string  `json:"rxnorm_code"`
	DrugName        string  `json:"drug_name"`
	DoseValue       float64 `json:"dose_value,omitempty"`
	DoseUnit        string  `json:"dose_unit,omitempty"`
	IsProposed      bool    `json:"is_proposed"`
	RequiresDoseAdj bool    `json:"requires_dose_adjustment,omitempty"`
}

// PatientDataInput contains patient context for risk calculations
type PatientDataInput struct {
	Age            int                   `json:"age"`
	Gender         string                `json:"gender"`
	WeightKg       float64               `json:"weight_kg,omitempty"`
	HeightCm       float64               `json:"height_cm,omitempty"`
	EGFR           float64               `json:"egfr,omitempty"`
	ChildPughScore string                `json:"child_pugh_score,omitempty"`
	IsPregnant     bool                  `json:"is_pregnant,omitempty"`
	IsLactating    bool                  `json:"is_lactating,omitempty"`
	Conditions     []ConditionRefInput   `json:"conditions,omitempty"`
	Allergies      []AllergyRefInput     `json:"allergies,omitempty"`
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

// LabValueInput represents a lab value for risk calculations
type LabValueInput struct {
	LOINCCode  string  `json:"loinc_code"`
	Value      float64 `json:"value"`
	Unit       string  `json:"unit"`
	IsCritical bool    `json:"is_critical,omitempty"`
}

// KB5DDIResult represents DDI check results from KB-5
type KB5DDIResult struct {
	InteractionsFound []KB5Interaction `json:"interactions_found"`
	HighestSeverity   string           `json:"highest_severity"`
	TotalInteractions int              `json:"total_interactions"`
}

// KB5Interaction represents a single DDI from KB-5
type KB5Interaction struct {
	InteractionID    string `json:"interaction_id"`
	DrugACode        string `json:"drug_a_code"`
	DrugAName        string `json:"drug_a_name"`
	DrugBCode        string `json:"drug_b_code"`
	DrugBName        string `json:"drug_b_name"`
	Severity         string `json:"severity"`
	ClinicalEffect   string `json:"clinical_effect"`
	ManagementStrategy string `json:"management_strategy"`
}

// =============================================================================
// VALIDATOR
// Coordinates hard block evaluation from multiple sources
// =============================================================================

// Validator evaluates medication transactions for safety blocks
type Validator struct {
	// Configuration for validation
	config ValidatorConfig
	// Optional KB-5 client for DDI checking (if nil, uses local rules)
	kb5Client KB5DDIChecker
	// V3: Optional Med-Advisor client for risk profiles (preferred path)
	riskProvider RiskProfileProvider
	// V3: Risk thresholds for converting risks to hard blocks
	riskThresholds RiskThresholds
}

// ValidatorConfig holds configuration for the validator
type ValidatorConfig struct {
	EnableDDIChecks     bool
	EnableLabChecks     bool
	EnableExclusionChks bool
	StrictMode          bool   // If true, any warning becomes a hard block
	UseKB5ForDDI        bool   // If true and kb5Client set, call KB-5 instead of local rules
	KB5BaseURL          string // KB-5 service URL
	// V3 configuration
	UseV3RiskProfiles   bool   // If true and riskProvider set, use V3 workflow (preferred)
	MedAdvisorURL       string // Medication Advisor service URL
}

// NewValidator creates a new validator with default configuration
func NewValidator() *Validator {
	return &Validator{
		config: ValidatorConfig{
			EnableDDIChecks:     true,
			EnableLabChecks:     true,
			EnableExclusionChks: true,
			StrictMode:          false,
			UseKB5ForDDI:        false,
			UseV3RiskProfiles:   false, // Default to legacy mode
		},
		riskThresholds: DefaultRiskThresholds(),
	}
}

// NewValidatorWithConfig creates a validator with custom configuration
func NewValidatorWithConfig(cfg ValidatorConfig) *Validator {
	return &Validator{
		config:         cfg,
		riskThresholds: DefaultRiskThresholds(),
	}
}

// SetKB5Client sets the KB-5 DDI client for external DDI checking
func (v *Validator) SetKB5Client(client KB5DDIChecker) {
	v.kb5Client = client
	v.config.UseKB5ForDDI = true
}

// SetRiskProvider sets the Med-Advisor client for V3 risk profile workflow
func (v *Validator) SetRiskProvider(provider RiskProfileProvider) {
	v.riskProvider = provider
	v.config.UseV3RiskProfiles = true
}

// SetRiskThresholds sets custom risk thresholds for block determination
func (v *Validator) SetRiskThresholds(thresholds RiskThresholds) {
	v.riskThresholds = thresholds
}

// IsV3Enabled returns true if the V3 risk profile workflow is enabled
func (v *Validator) IsV3Enabled() bool {
	return v.config.UseV3RiskProfiles && v.riskProvider != nil
}

// GetRiskProvider returns the configured risk profile provider (Med-Advisor)
func (v *Validator) GetRiskProvider() RiskProfileProvider {
	return v.riskProvider
}

// =============================================================================
// V3: RISK PROFILE BASED VALIDATION
// Med-Advisor calculates risks, KB-19 converts to governance decisions
// =============================================================================

// ValidateTransactionV3 performs validation using Med-Advisor risk profiles.
// This is the preferred V3 workflow: Med-Advisor = Judge, KB-19 = Clerk
func (v *Validator) ValidateTransactionV3(
	ctx context.Context,
	txn *Transaction,
	riskProfile *RiskProfile,
) error {
	// Reset any existing blocks
	txn.HardBlocks = []HardBlock{}

	// Convert DDI risks to hard blocks
	ddiBlocks := v.convertDDIRisksToBlocks(riskProfile.DDIRisks)
	txn.HardBlocks = append(txn.HardBlocks, ddiBlocks...)

	// Convert Lab risks to hard blocks
	labBlocks := v.convertLabRisksToBlocks(riskProfile.LabRisks)
	txn.HardBlocks = append(txn.HardBlocks, labBlocks...)

	// Convert Allergy risks to hard blocks
	allergyBlocks := v.convertAllergyRisksToBlocks(riskProfile.AllergyRisks)
	txn.HardBlocks = append(txn.HardBlocks, allergyBlocks...)

	// Convert high-risk medications to blocks (CRITICAL category)
	medBlocks := v.convertMedicationRisksToBlocks(riskProfile.MedicationRisks)
	txn.HardBlocks = append(txn.HardBlocks, medBlocks...)

	// Update transaction state based on blocks
	if len(txn.HardBlocks) > 0 {
		txn.State = StateBlocked
	} else {
		txn.State = StateValidated
	}

	return nil
}

// convertDDIRisksToBlocks converts DDI risk assessments to hard blocks
func (v *Validator) convertDDIRisksToBlocks(ddiRisks []DDIRisk) []HardBlock {
	var blocks []HardBlock

	for _, risk := range ddiRisks {
		// Check if severity triggers a hard block
		if !v.severityTriggersBlock(risk.Severity, v.riskThresholds.DDISeverities) {
			continue
		}

		block := HardBlock{
			ID:        uuid.New(),
			BlockType: "DDI_SEVERE",
			Severity:  risk.Severity,
			Medication: ClinicalCode{
				System:  "RxNorm",
				Code:    risk.Drug1Code,
				Display: risk.Drug1Name,
			},
			TriggerCondition: ClinicalCode{
				System:  "RxNorm",
				Code:    risk.Drug2Code,
				Display: risk.Drug2Name,
			},
			Reason:      fmt.Sprintf("DDI: %s + %s - %s", risk.Drug1Name, risk.Drug2Name, risk.ClinicalEffect),
			KBSource:    risk.KBSource,
			RuleID:      risk.RuleID,
			RequiresAck: true,
			AckText: fmt.Sprintf("I acknowledge the %s drug-drug interaction between %s and %s. "+
				"Clinical effect: %s. Management: %s. "+
				"I have reviewed the risks and take full clinical responsibility for any override decision.",
				risk.Severity, risk.Drug1Name, risk.Drug2Name,
				risk.ClinicalEffect, risk.ManagementStrategy),
		}
		blocks = append(blocks, block)
	}

	return blocks
}

// convertLabRisksToBlocks converts Lab risk assessments to hard blocks
func (v *Validator) convertLabRisksToBlocks(labRisks []LabRisk) []HardBlock {
	var blocks []HardBlock

	for _, risk := range labRisks {
		// Check if severity triggers a hard block
		if !v.severityTriggersBlock(risk.Severity, v.riskThresholds.LabSeverities) {
			continue
		}

		block := HardBlock{
			ID:        uuid.New(),
			BlockType: "LAB_CONTRAINDICATION",
			Severity:  risk.Severity,
			Medication: ClinicalCode{
				System:  "RxNorm",
				Code:    risk.RxNormCode,
				Display: risk.DrugName,
			},
			TriggerCondition: ClinicalCode{
				System:  "LOINC",
				Code:    risk.LOINCCode,
				Display: risk.LabName,
			},
			Reason:      fmt.Sprintf("Lab contraindication: %s (%s %s %v) - %s", risk.LabName, risk.ThresholdOp, fmt.Sprintf("%.2f", risk.ThresholdValue), risk.CurrentValue, risk.ClinicalRisk),
			KBSource:    risk.KBSource,
			RuleID:      risk.RuleID,
			RequiresAck: true,
			AckText: fmt.Sprintf("I acknowledge the lab-based contraindication for %s. "+
				"Lab: %s, Current value: %.2f, Threshold: %s %.2f. "+
				"Clinical risk: %s. Recommendation: %s. "+
				"I have reviewed the risks and take full clinical responsibility for any override decision.",
				risk.DrugName, risk.LabName, risk.CurrentValue, risk.ThresholdOp, risk.ThresholdValue,
				risk.ClinicalRisk, risk.Recommendation),
		}
		blocks = append(blocks, block)
	}

	return blocks
}

// convertAllergyRisksToBlocks converts Allergy risk assessments to hard blocks
func (v *Validator) convertAllergyRisksToBlocks(allergyRisks []AllergyRisk) []HardBlock {
	var blocks []HardBlock

	for _, risk := range allergyRisks {
		// Check if severity triggers a hard block
		if !v.severityTriggersBlock(risk.Severity, v.riskThresholds.AllergySeverities) {
			continue
		}

		crossReactiveNote := ""
		if risk.IsCrossReactive {
			crossReactiveNote = " (cross-reactive)"
		}

		block := HardBlock{
			ID:        uuid.New(),
			BlockType: "ALLERGY_CONTRAINDICATION",
			Severity:  risk.Severity,
			Medication: ClinicalCode{
				System:  "RxNorm",
				Code:    risk.RxNormCode,
				Display: risk.DrugName,
			},
			TriggerCondition: ClinicalCode{
				System:  "Allergen",
				Code:    risk.AllergenCode,
				Display: risk.AllergenName,
			},
			Reason:      fmt.Sprintf("Allergy: Patient allergic to %s%s - %s", risk.AllergenName, crossReactiveNote, risk.ReactionType),
			KBSource:    risk.KBSource,
			RuleID:      risk.RuleID,
			RequiresAck: true,
			AckText: fmt.Sprintf("I acknowledge the allergy contraindication for %s. "+
				"Patient has documented allergy to %s%s. "+
				"Expected reaction type: %s. Severity: %s. "+
				"I have reviewed the risks and take full clinical responsibility for any override decision.",
				risk.DrugName, risk.AllergenName, crossReactiveNote, risk.ReactionType, risk.Severity),
		}
		blocks = append(blocks, block)
	}

	return blocks
}

// convertMedicationRisksToBlocks converts high-risk medication assessments to hard blocks
func (v *Validator) convertMedicationRisksToBlocks(medRisks []MedicationRisk) []HardBlock {
	var blocks []HardBlock

	for _, risk := range medRisks {
		// Only create blocks for CRITICAL risk category
		if risk.OverallRisk < v.riskThresholds.OverallRiskCutoff {
			continue
		}

		// Build risk factors description
		riskFactorDesc := ""
		for i, rf := range risk.RiskFactors {
			if i > 0 {
				riskFactorDesc += "; "
			}
			riskFactorDesc += fmt.Sprintf("%s (%s): %s", rf.Type, rf.Severity, rf.Description)
		}

		blockType := "HIGH_RISK_MEDICATION"
		if risk.HasBlackBoxWarn {
			blockType = "BLACK_BOX_WARNING"
		}
		if risk.IsHighAlert {
			blockType = "HIGH_ALERT_MEDICATION"
		}

		block := HardBlock{
			ID:        uuid.New(),
			BlockType: blockType,
			Severity:  risk.RiskCategory,
			Medication: ClinicalCode{
				System:  "RxNorm",
				Code:    risk.RxNormCode,
				Display: risk.DrugName,
			},
			Reason:      fmt.Sprintf("High-risk medication (%.0f%% risk): %s", risk.OverallRisk*100, riskFactorDesc),
			KBSource:    "MedAdvisor",
			RuleID:      fmt.Sprintf("MED_RISK_%s", risk.RxNormCode),
			RequiresAck: true,
			AckText: fmt.Sprintf("I acknowledge the high-risk medication warning for %s. "+
				"Overall risk score: %.0f%% (%s). "+
				"Risk factors: %s. "+
				"I have reviewed the risks and take full clinical responsibility for any override decision.",
				risk.DrugName, risk.OverallRisk*100, risk.RiskCategory, riskFactorDesc),
		}
		blocks = append(blocks, block)
	}

	return blocks
}

// severityTriggersBlock checks if a severity level should trigger a hard block
func (v *Validator) severityTriggersBlock(severity string, triggerSeverities []string) bool {
	severityLower := strings.ToLower(severity)
	for _, trigger := range triggerSeverities {
		if strings.ToLower(trigger) == severityLower {
			return true
		}
	}
	return false
}

// NOTE: Legacy ValidateTransaction function REMOVED in V3 architecture.
// V3 requires Med-Advisor for risk calculation. Use ValidateTransactionV3 instead.
// See manager.go ValidateTransaction() which enforces V3-only path.

// NOTE: Legacy evaluateDDIViaKB5 function REMOVED in V3 architecture.
// V3 DDI checking is done via Med-Advisor which calls KB-5 internally.
// This removes the silent fallback behavior that could mask service failures.

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 893-950
// Refactored: Changed from receiver method to Validator method
// =============================================================================

// ExcludedDrug represents a drug excluded during workflow
type ExcludedDrug struct {
	Medication ClinicalCode `json:"medication"`
	Reason     string       `json:"reason"`
	Severity   string       `json:"severity"`
	KBSource   string       `json:"kb_source"`
	RuleID     string       `json:"rule_id"`
}

// PatientContext represents patient clinical context for validation
type PatientContext struct {
	PatientID  uuid.UUID      `json:"patient_id"`
	Sex        string         `json:"sex"`
	Age        int            `json:"age"`
	WeightKg   float64        `json:"weight_kg"`
	Conditions []ClinicalCode `json:"conditions"`
	Allergies  []ClinicalCode `json:"allergies"`
	IsPregnant bool           `json:"is_pregnant"`
	EGFR       float64        `json:"egfr"`
}

// EvaluateExcludedDrugs converts workflow excluded drugs to hard blocks
// MOVED FROM: medication-advisor-engine/advisor/engine.go processExcludedDrugs()
func (v *Validator) EvaluateExcludedDrugs(
	excluded []ExcludedDrug,
	patientContext PatientContext,
) ([]HardBlock, []ExcludedDrugInfo) {
	var hardBlocks []HardBlock
	var excludedDrugs []ExcludedDrugInfo

	// Find pregnancy condition for hard block context
	var pregnancyCondition *ClinicalCode
	for _, cond := range patientContext.Conditions {
		if isPregnancyCode(cond.Code) {
			pregnancyCondition = &cond
			break
		}
	}

	for _, ex := range excluded {
		// Determine if this is a hard block based on severity
		isHardBlock := isHardBlockSeverity(ex.Severity)

		// Convert to ExcludedDrugInfo
		excludedDrug := ExcludedDrugInfo{
			Medication:  ex.Medication,
			Reason:      ex.Reason,
			Severity:    ex.Severity,
			BlockType:   ex.Severity, // Use severity as block type for now
			KBSource:    ex.KBSource,
			RuleID:      ex.RuleID,
			IsHardBlock: isHardBlock,
		}
		excludedDrugs = append(excludedDrugs, excludedDrug)

		// Generate hard block if severity warrants it
		if isHardBlock {
			hardBlock := HardBlock{
				ID:          uuid.New(),
				BlockType:   mapToHardBlockType(ex.Severity),
				Severity:    ex.Severity,
				Medication:  ex.Medication,
				Reason:      ex.Reason,
				KBSource:    ex.KBSource,
				RuleID:      ex.RuleID,
				RequiresAck: true,
				AckText:     generateAckText(ex.Medication.Display, ex.Reason),
			}

			// Add trigger condition if pregnancy-related
			if pregnancyCondition != nil && isPregnancyRelatedBlock(ex.Reason) {
				hardBlock.TriggerCondition = *pregnancyCondition
				hardBlock.FDACategory = extractFDACategory(ex.Reason)
			}

			hardBlocks = append(hardBlocks, hardBlock)
		}
	}

	return hardBlocks, excludedDrugs
}

// =============================================================================
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 747-780
// =============================================================================

// MedicationProposal represents a medication proposal for disposition determination
type MedicationProposal struct {
	Medication     ClinicalCode     `json:"medication"`
	Warnings       []ProposalWarning `json:"warnings"`
	QualityFactors QualityFactors   `json:"quality_factors"`
}

// ProposalWarning represents a warning on a proposal
type ProposalWarning struct {
	Type     string `json:"type"`
	Severity string `json:"severity"`
	Message  string `json:"message"`
}

// QualityFactors contains quality metrics for a proposal
type QualityFactors struct {
	Safety      float64 `json:"safety"`
	Efficacy    float64 `json:"efficacy"`
	CostValue   float64 `json:"cost_value"`
	Convenience float64 `json:"convenience"`
}

// WorkflowResult represents results from the risk calculation workflow
type WorkflowResult struct {
	ExcludedDrugs  []ExcludedDrug   `json:"excluded_drugs"`
	InferenceChain []InferenceStep  `json:"inference_chain"`
}

// InferenceStep represents a step in the inference chain
type InferenceStep struct {
	RuleID      string `json:"rule_id"`
	KBSource    string `json:"kb_source"`
	Description string `json:"description"`
}

// DetermineDisposition determines the recommended next action based on results
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 747-780
func (v *Validator) DetermineDisposition(
	hardBlocks []HardBlock,
	proposals []MedicationProposal,
	workflowResult *WorkflowResult,
) DispositionCode {
	// Hard blocks always result in HARD_STOP
	if len(hardBlocks) > 0 {
		return DispositionHardStop
	}

	// No proposals means we need more information
	if len(proposals) == 0 {
		return DispositionRecalculate
	}

	// Check for severe interactions or warnings
	for _, proposal := range proposals {
		for _, warning := range proposal.Warnings {
			if warning.Severity == "critical" {
				return DispositionHoldForReview
			}
		}
	}

	// Check quality scores - low safety score requires review
	for _, proposal := range proposals {
		if proposal.QualityFactors.Safety < 0.6 {
			return DispositionHoldForApproval
		}
	}

	// All clear - safe to proceed
	return DispositionDispense
}

// =============================================================================
// HELPER FUNCTIONS
// MOVED FROM: medication-advisor-engine/advisor/engine.go lines 952-1065
// =============================================================================

// isHardBlockSeverity determines if the severity level requires a hard block
func isHardBlockSeverity(severity string) bool {
	hardBlockSeverities := map[string]bool{
		"absolute":         true,
		"contraindicated":  true,
		"life_threatening": true,
		"severe":           true,
	}
	return hardBlockSeverities[severity]
}

// mapToHardBlockType converts severity to a standardized block type
func mapToHardBlockType(severity string) string {
	switch severity {
	case "absolute", "contraindicated":
		return "CONTRAINDICATION"
	case "life_threatening":
		return "LIFE_THREATENING"
	case "severe":
		return "DDI_SEVERE"
	default:
		return "CONTRAINDICATION"
	}
}

// isPregnancyCode checks if a SNOMED code represents pregnancy
func isPregnancyCode(code string) bool {
	pregnancyCodes := map[string]bool{
		"77386006":          true, // Pregnancy
		"72892002":          true, // Normal pregnancy
		"237238006":         true, // Gestational diabetes mellitus
		"48194001":          true, // Pregnancy-induced hypertension
		"10746341000119109": true, // High risk pregnancy
	}
	return pregnancyCodes[code]
}

// isPregnancyRelatedBlock checks if the block reason mentions pregnancy
func isPregnancyRelatedBlock(reason string) bool {
	// Check for pregnancy-related keywords in reason
	keywords := []string{"pregnancy", "pregnant", "teratogenic", "fetal", "gestational", "FDA Category D", "FDA Category X"}
	for _, kw := range keywords {
		if containsIgnoreCase(reason, kw) {
			return true
		}
	}
	return false
}

// containsIgnoreCase checks if s contains substr (case-insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		(s == substr ||
			len(s) >= len(substr) && containsLower(s, substr))
}

func containsLower(s, substr string) bool {
	// Manual lowercase comparison
	sLower := make([]byte, len(s))
	substrLower := make([]byte, len(substr))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			sLower[i] = c + 32
		} else {
			sLower[i] = c
		}
	}
	for i := 0; i < len(substr); i++ {
		c := substr[i]
		if c >= 'A' && c <= 'Z' {
			substrLower[i] = c + 32
		} else {
			substrLower[i] = c
		}
	}
	// Find substr in s
	for i := 0; i <= len(sLower)-len(substrLower); i++ {
		match := true
		for j := 0; j < len(substrLower); j++ {
			if sLower[i+j] != substrLower[j] {
				match = false
				break
			}
		}
		if match {
			return true
		}
	}
	return false
}

// extractFDACategory extracts FDA pregnancy category from reason text
func extractFDACategory(reason string) string {
	if containsIgnoreCase(reason, "Category D") || containsIgnoreCase(reason, "(D)") {
		return "D"
	}
	if containsIgnoreCase(reason, "Category X") || containsIgnoreCase(reason, "(X)") {
		return "X"
	}
	if containsIgnoreCase(reason, "Category C") || containsIgnoreCase(reason, "(C)") {
		return "C"
	}
	return ""
}

// generateAckText generates the acknowledgment text for a hard block
func generateAckText(drugName, reason string) string {
	return fmt.Sprintf("I acknowledge that %s is contraindicated for this patient due to: %s. "+
		"I understand the risks and take full clinical responsibility for any override decision.",
		drugName, reason)
}
