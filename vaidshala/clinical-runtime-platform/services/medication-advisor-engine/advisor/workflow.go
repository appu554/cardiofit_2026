package advisor

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/medication-advisor-engine/evidence"
	"github.com/cardiofit/medication-advisor-engine/kbclients"
	"github.com/cardiofit/medication-advisor-engine/recipe"
	"github.com/cardiofit/medication-advisor-engine/snapshot"
)

// ErrKBUnavailable indicates a required Knowledge Base service is unavailable
var ErrKBUnavailable = fmt.Errorf("required KB service unavailable")

// WorkflowOrchestrator orchestrates the 4-phase medication advisory workflow.
// REQUIRES: All KB services (KB-1 through KB-6) must be available.
type WorkflowOrchestrator struct {
	config EngineConfig

	// KB Manager for KB service calls - REQUIRED
	kbManager *kbclients.KBManager

	// Recipe Resolver for coordinated KB orchestration - REQUIRED
	recipeResolver *recipe.RecipeResolver

	// Phase timing tracking
	phaseTimings map[string]time.Duration
}

// WorkflowInput represents input to the workflow
type WorkflowInput struct {
	Snapshot       *snapshot.ClinicalSnapshot
	Question       ClinicalQuestion
	PatientContext PatientContext
	EnvelopeID     uuid.UUID
}

// WorkflowResult represents the result of workflow execution
type WorkflowResult struct {
	Candidates       []MedicationCandidate
	ExcludedDrugs    []ExcludedDrug
	InferenceChain   []evidence.InferenceStep
	PhaseTimings     map[string]time.Duration
}

// MedicationCandidate represents a candidate medication from the workflow
type MedicationCandidate struct {
	Medication     ClinicalCode
	Dosage         Dosage
	Scores         QualityFactors
	TotalScore     float64
	Rationale      string
	Warnings       []Warning
	KBSources      []string
}

// ExcludedDrug represents a drug that was excluded during the workflow
type ExcludedDrug struct {
	Medication ClinicalCode
	Reason     string
	KBSource   string
	RuleID     string
	Severity   string // contraindication, interaction, allergy
}

// NewWorkflowOrchestrator creates a new workflow orchestrator.
// Returns error if KB services cannot be initialized - KB services are REQUIRED.
func NewWorkflowOrchestrator(config EngineConfig) (*WorkflowOrchestrator, error) {
	wo := &WorkflowOrchestrator{
		config:       config,
		phaseTimings: make(map[string]time.Duration),
	}

	// Validate required KB URLs
	if config.KB1URL == "" || config.KB2URL == "" || config.KB3URL == "" ||
		config.KB4URL == "" || config.KB5URL == "" || config.KB6URL == "" {
		return nil, fmt.Errorf("all KB service URLs are required: KB1-KB6")
	}

	// Initialize KB Manager - REQUIRED
	kbConfig := kbclients.KBManagerConfig{
		KB1URL:        config.KB1URL,
		KB2URL:        config.KB2URL,
		KB3URL:        config.KB3URL,
		KB4URL:        config.KB4URL,
		KB5URL:        config.KB5URL,
		KB6URL:        config.KB6URL,
		Timeout:       30 * time.Second,
		RetryAttempts: 3,
	}

	kbMgr, err := kbclients.NewKBManager(kbConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize KB Manager: %w", err)
	}
	wo.kbManager = kbMgr

	// Initialize Recipe Resolver - REQUIRED, no fallback
	wo.recipeResolver = recipe.NewRecipeResolver(kbMgr, recipe.ResolverConfig{
		EnableFallback: false, // NO FALLBACK - KB services required
	})

	return wo, nil
}

// NewWorkflowOrchestratorWithKBManager creates a workflow orchestrator with a provided KB manager.
// This is primarily used for testing with mock KB clients.
func NewWorkflowOrchestratorWithKBManager(config EngineConfig, kbMgr *kbclients.KBManager) (*WorkflowOrchestrator, error) {
	wo := &WorkflowOrchestrator{
		config:       config,
		phaseTimings: make(map[string]time.Duration),
		kbManager:    kbMgr,
	}

	// Initialize Recipe Resolver with provided KB manager
	wo.recipeResolver = recipe.NewRecipeResolver(kbMgr, recipe.ResolverConfig{
		EnableFallback: false, // NO FALLBACK - KB services required
	})

	return wo, nil
}

// Execute runs the complete 4-phase workflow.
// All KB services must be available or execution will fail.
func (wo *WorkflowOrchestrator) Execute(ctx context.Context, input *WorkflowInput) (*WorkflowResult, error) {
	// Validate KB services are available
	if wo.kbManager == nil || wo.recipeResolver == nil {
		return nil, fmt.Errorf("workflow not properly initialized: KB services required")
	}

	result := &WorkflowResult{
		Candidates:     []MedicationCandidate{},
		ExcludedDrugs:  []ExcludedDrug{},
		InferenceChain: []evidence.InferenceStep{},
		PhaseTimings:   make(map[string]time.Duration),
	}

	chainBuilder := evidence.NewInferenceChainBuilder(input.EnvelopeID)

	// Phase 1: Recipe Resolution
	phase1Start := time.Now()
	recipeResult, err := wo.executePhase1(ctx, input, chainBuilder)
	if err != nil {
		return nil, fmt.Errorf("phase 1 (recipe resolution) failed: %w", err)
	}
	result.PhaseTimings["phase1_recipe"] = time.Since(phase1Start)

	// Phase 2: KB Orchestration (Safety Checks)
	phase2Start := time.Now()
	safetyResult, err := wo.executePhase2(ctx, input, recipeResult, chainBuilder)
	if err != nil {
		return nil, fmt.Errorf("phase 2 (safety checks) failed: %w", err)
	}
	result.PhaseTimings["phase2_safety"] = time.Since(phase2Start)
	result.ExcludedDrugs = append(result.ExcludedDrugs, safetyResult.Excluded...)

	// Phase 3: Dosing & Adjustments
	phase3Start := time.Now()
	dosageResult, err := wo.executePhase3(ctx, input, safetyResult, chainBuilder)
	if err != nil {
		return nil, fmt.Errorf("phase 3 (dosing) failed: %w", err)
	}
	result.PhaseTimings["phase3_dosing"] = time.Since(phase3Start)

	// Phase 4: Proposal Generation & Scoring
	phase4Start := time.Now()
	proposalResult, err := wo.executePhase4(ctx, input, dosageResult, chainBuilder)
	if err != nil {
		return nil, fmt.Errorf("phase 4 (scoring) failed: %w", err)
	}
	result.PhaseTimings["phase4_scoring"] = time.Since(phase4Start)
	result.Candidates = proposalResult.Candidates

	// Build inference chain
	result.InferenceChain = chainBuilder.Build()

	return result, nil
}

// Phase 1: Recipe Resolution - Gather required data
func (wo *WorkflowOrchestrator) executePhase1(
	ctx context.Context,
	input *WorkflowInput,
	chainBuilder *evidence.InferenceChainBuilder,
) (*RecipeResult, error) {

	result := &RecipeResult{
		DataRequirements: []string{
			"conditions",
			"medications",
			"allergies",
			"lab_results",
			"computed_scores",
		},
		ResolvedData: make(map[string]interface{}),
	}

	// Extract data from snapshot
	result.ResolvedData["conditions"] = input.Snapshot.ClinicalData.Conditions
	result.ResolvedData["medications"] = input.Snapshot.ClinicalData.Medications
	result.ResolvedData["allergies"] = input.Snapshot.ClinicalData.Allergies
	result.ResolvedData["lab_results"] = input.Snapshot.ClinicalData.LabResults
	result.ResolvedData["computed_scores"] = input.Snapshot.ComputedScores

	// Record in inference chain
	chainBuilder.AddRecipeStep(
		input.Snapshot.RecipeID.String(),
		result.DataRequirements,
		result.ResolvedData,
	)

	return result, nil
}

// Phase 2: KB Orchestration - Safety checks (KB-2, KB-3, KB-4)
func (wo *WorkflowOrchestrator) executePhase2(
	ctx context.Context,
	input *WorkflowInput,
	recipeResult *RecipeResult,
	chainBuilder *evidence.InferenceChainBuilder,
) (*SafetyResult, error) {

	result := &SafetyResult{
		SafeDrugs: []ClinicalCode{},
		Excluded:  []ExcludedDrug{},
		Warnings:  []Warning{},
	}

	// Get candidate drugs from KB-3 via RecipeResolver - REQUIRED
	candidates, err := wo.getCandidateDrugs(ctx, input.Question)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate drugs from KB-3: %w", err)
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("no candidate drugs found for indication=%s, drugClass=%s",
			input.Question.Indication, input.Question.TargetDrugClass)
	}

	for _, candidate := range candidates {
		excluded := false

		// Check allergies via KB-4 - REQUIRED
		for _, allergy := range input.Snapshot.ClinicalData.Allergies {
			isMatch, err := wo.checkAllergyMatch(ctx, candidate, allergy)
			if err != nil {
				return nil, fmt.Errorf("KB-4 allergy check failed: %w", err)
			}

			if isMatch {
				result.Excluded = append(result.Excluded, ExcludedDrug{
					Medication: candidate,
					Reason:     fmt.Sprintf("Patient allergic to %s", allergy.Allergen),
					KBSource:   "KB-4",
					Severity:   "allergy",
				})

				chainBuilder.AddExclusionStep(
					candidate.Display,
					fmt.Sprintf("Excluded due to allergy: %s", allergy.Allergen),
					"KB-4",
					"allergy-check",
					"critical",
				)
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		// Check contraindications via KB-4 - REQUIRED
		contraindication, err := wo.checkContraindications(ctx, candidate, input.PatientContext)
		if err != nil {
			return nil, fmt.Errorf("KB-4 contraindication check failed: %w", err)
		}

		if contraindication != "" {
			result.Excluded = append(result.Excluded, ExcludedDrug{
				Medication: candidate,
				Reason:     contraindication,
				KBSource:   "KB-4",
				RuleID:     "contraindication",
				Severity:   "contraindication",
			})

			chainBuilder.AddContraindicationStep(
				candidate.Display,
				contraindication,
				"patient condition",
				"KB-4",
				"contraindication",
			)
			excluded = true
			continue
		}

		// Check drug-drug interactions via KB-2 - REQUIRED
		for _, existingMed := range input.Snapshot.ClinicalData.Medications {
			interaction, err := wo.checkDrugInteraction(ctx, candidate, ClinicalCode{
				Code:    existingMed.RxNormCode,
				Display: existingMed.MedicationName,
			})
			if err != nil {
				return nil, fmt.Errorf("KB-2 interaction check failed: %w", err)
			}

			if interaction != nil && interaction.Severity == "severe" {
				result.Excluded = append(result.Excluded, ExcludedDrug{
					Medication: candidate,
					Reason:     interaction.Description,
					KBSource:   "KB-2",
					RuleID:     "ddi-severe",
					Severity:   "interaction",
				})

				chainBuilder.AddInteractionStep(
					candidate.Display,
					existingMed.MedicationName,
					interaction.Type,
					interaction.Severity,
					interaction.Recommendation,
				)
				excluded = true
				break
			} else if interaction != nil {
				result.Warnings = append(result.Warnings, Warning{
					Severity: "warning",
					Message:  interaction.Description,
					Source:   "KB-2",
				})
			}
		}

		if !excluded {
			result.SafeDrugs = append(result.SafeDrugs, candidate)

			// Record KB query in chain
			chainBuilder.AddKBQueryStep(
				"KB-4",
				"safety_check",
				map[string]string{"medication": candidate.Code},
				map[string]bool{"safe": true},
				fmt.Sprintf("%s passed all safety checks", candidate.Display),
			)
		}
	}

	return result, nil
}

// Phase 3: Dosing & Adjustments (KB-1)
func (wo *WorkflowOrchestrator) executePhase3(
	ctx context.Context,
	input *WorkflowInput,
	safetyResult *SafetyResult,
	chainBuilder *evidence.InferenceChainBuilder,
) (*DosageResult, error) {

	result := &DosageResult{
		DosedDrugs: []DosedDrug{},
	}

	scores := input.Snapshot.ComputedScores

	for _, drug := range safetyResult.SafeDrugs {
		// Get standard dosage from KB-1 - REQUIRED
		dosage, err := wo.getStandardDosage(ctx, drug)
		if err != nil {
			return nil, fmt.Errorf("KB-1 dosage lookup failed for %s: %w", drug.Display, err)
		}

		adjustments := []string{}

		// Check renal adjustment
		if scores.EGFR != nil && scores.RequiresRenalDoseAdjustment {
			adjustedDose := dosage.Value * wo.getRenalAdjustmentFactor(*scores.EGFR)
			if adjustedDose != dosage.Value {
				chainBuilder.AddDoseAdjustmentStep(
					drug.Display,
					dosage.Value,
					adjustedDose,
					dosage.Unit,
					fmt.Sprintf("Renal adjustment for eGFR %.1f", *scores.EGFR),
					"KB-1",
				)
				dosage.Value = adjustedDose
				adjustments = append(adjustments, "renal")
			}
		}

		// Check hepatic adjustment
		if scores.ChildPughClass != "" && scores.RequiresHepaticDoseAdjustment {
			adjustedDose := dosage.Value * wo.getHepaticAdjustmentFactor(scores.ChildPughClass)
			if adjustedDose != dosage.Value {
				chainBuilder.AddDoseAdjustmentStep(
					drug.Display,
					dosage.Value,
					adjustedDose,
					dosage.Unit,
					fmt.Sprintf("Hepatic adjustment for Child-Pugh %s", scores.ChildPughClass),
					"KB-1",
				)
				dosage.Value = adjustedDose
				adjustments = append(adjustments, "hepatic")
			}
		}

		result.DosedDrugs = append(result.DosedDrugs, DosedDrug{
			Medication:  drug,
			Dosage:      dosage,
			Adjustments: adjustments,
			Warnings:    safetyResult.Warnings,
		})
	}

	return result, nil
}

// Phase 4: Proposal Generation & Scoring (KB-3, KB-5, KB-6)
func (wo *WorkflowOrchestrator) executePhase4(
	ctx context.Context,
	input *WorkflowInput,
	dosageResult *DosageResult,
	chainBuilder *evidence.InferenceChainBuilder,
) (*ProposalResult, error) {

	result := &ProposalResult{
		Candidates: []MedicationCandidate{},
	}

	for _, dosedDrug := range dosageResult.DosedDrugs {
		// Calculate quality factors using KB-3, KB-5, KB-6 - REQUIRED
		factors, err := wo.calculateQualityFactors(ctx, dosedDrug, input)
		if err != nil {
			return nil, fmt.Errorf("quality scoring failed for %s: %w", dosedDrug.Medication.Display, err)
		}

		// Calculate weighted score
		totalScore := factors.Guideline*0.30 +
			factors.Safety*0.25 +
			factors.Efficacy*0.20 +
			factors.Interaction*0.15 +
			factors.Monitoring*0.10

		// Record scoring in chain
		chainBuilder.AddScoringStep(
			dosedDrug.Medication.Display,
			map[string]float64{
				"guideline":   factors.Guideline,
				"safety":      factors.Safety,
				"efficacy":    factors.Efficacy,
				"interaction": factors.Interaction,
				"monitoring":  factors.Monitoring,
			},
			map[string]float64{
				"guideline":   0.30,
				"safety":      0.25,
				"efficacy":    0.20,
				"interaction": 0.15,
				"monitoring":  0.10,
			},
			totalScore,
			fmt.Sprintf("Quality score %.2f: guideline %.0f%%, safety %.0f%%, efficacy %.0f%%",
				totalScore, factors.Guideline*100, factors.Safety*100, factors.Efficacy*100),
		)

		result.Candidates = append(result.Candidates, MedicationCandidate{
			Medication: dosedDrug.Medication,
			Dosage:     dosedDrug.Dosage,
			Scores:     factors,
			TotalScore: totalScore,
			Rationale:  wo.generateRationale(dosedDrug, factors, input.Question),
			Warnings:   dosedDrug.Warnings,
			KBSources:  []string{"KB-1", "KB-2", "KB-3", "KB-4", "KB-5", "KB-6"},
		})
	}

	// Add ranking step
	if len(result.Candidates) > 0 {
		meds := make([]string, len(result.Candidates))
		scores := make([]float64, len(result.Candidates))
		rankings := make([]int, len(result.Candidates))

		for i, c := range result.Candidates {
			meds[i] = c.Medication.Display
			scores[i] = c.TotalScore
			rankings[i] = i + 1
		}

		chainBuilder.AddRankingStep(meds, scores, rankings,
			"Final ranking based on weighted quality factors")
	}

	return result, nil
}

// Helper types

type RecipeResult struct {
	DataRequirements []string
	ResolvedData     map[string]interface{}
}

type SafetyResult struct {
	SafeDrugs []ClinicalCode
	Excluded  []ExcludedDrug
	Warnings  []Warning
}

type DosageResult struct {
	DosedDrugs []DosedDrug
}

type DosedDrug struct {
	Medication  ClinicalCode
	Dosage      Dosage
	Adjustments []string
	Warnings    []Warning
}

type ProposalResult struct {
	Candidates []MedicationCandidate
}

type DrugInteraction struct {
	Type           string
	Severity       string
	Description    string
	Recommendation string
}

// getCandidateDrugs retrieves candidate drugs via RecipeResolver.
// The RecipeResolver first tries KB-3 Guidelines, then falls back to KB-1 Drug Rules
// if KB-3 doesn't provide drug recommendations for the indication/class.
func (wo *WorkflowOrchestrator) getCandidateDrugs(ctx context.Context, question ClinicalQuestion) ([]ClinicalCode, error) {
	if wo.recipeResolver == nil {
		return nil, fmt.Errorf("%w: RecipeResolver not initialized", ErrKBUnavailable)
	}

	recipeInput := &recipe.RecipeInput{
		Indication:      question.Indication,
		TargetDrugClass: question.TargetDrugClass,
		PatientID:       "",
	}

	recipeOutput, err := wo.recipeResolver.Resolve(ctx, recipeInput)
	if err != nil {
		return nil, fmt.Errorf("recipe resolution failed (KB-3/KB-1): %w", err)
	}

	if len(recipeOutput.CandidateDrugs) == 0 {
		return nil, fmt.Errorf("no drug candidates found for indication=%s, class=%s (checked KB-3 and KB-1)",
			question.Indication, question.TargetDrugClass)
	}

	candidates := make([]ClinicalCode, 0, len(recipeOutput.CandidateDrugs))
	for _, candidate := range recipeOutput.CandidateDrugs {
		candidates = append(candidates, ClinicalCode{
			System:  "RxNorm",
			Code:    candidate.RxNormCode,
			Display: candidate.DrugName,
		})
	}

	return candidates, nil
}

// checkAllergyMatch checks allergy match via KB-4 Safety service.
// Returns error if KB-4 is unavailable - NO FALLBACK.
func (wo *WorkflowOrchestrator) checkAllergyMatch(ctx context.Context, drug ClinicalCode, allergy snapshot.AllergyEntry) (bool, error) {
	if wo.kbManager == nil || wo.kbManager.Safety() == nil {
		return false, fmt.Errorf("%w: KB-4 Safety service not available", ErrKBUnavailable)
	}

	result, err := wo.kbManager.Safety().CheckAllergyMatch(ctx, drug.Code, allergy.Allergen)
	if err != nil {
		return false, fmt.Errorf("KB-4 allergy check error: %w", err)
	}

	return result.IsMatch, nil
}

// checkContraindications checks contraindications via KB-4 Safety service.
// Returns error if KB-4 is unavailable - NO FALLBACK.
func (wo *WorkflowOrchestrator) checkContraindications(ctx context.Context, drug ClinicalCode, patient PatientContext) (string, error) {
	if wo.kbManager == nil || wo.kbManager.Safety() == nil {
		return "", fmt.Errorf("%w: KB-4 Safety service not available", ErrKBUnavailable)
	}

	// Extract condition codes from patient context
	conditionCodes := make([]string, 0, len(patient.Conditions))
	for _, cond := range patient.Conditions {
		conditionCodes = append(conditionCodes, cond.Code)
	}

	req := &kbclients.ContraindicationRequest{
		RxNormCode:     drug.Code,
		DrugName:       drug.Display, // Pass drug name for name-based matching
		ConditionCodes: conditionCodes,
		AgeYears:       patient.Age,
		EGFR:           patient.ComputedScores.EGFR,
	}

	result, err := wo.kbManager.Safety().CheckContraindication(ctx, req)
	if err != nil {
		return "", fmt.Errorf("KB-4 contraindication check error: %w", err)
	}

	if result.IsContraindicated {
		return result.Reason, nil
	}

	return "", nil
}

// checkDrugInteraction checks drug-drug interactions via KB-2 Interactions service.
// Returns error if KB-2 is unavailable - NO FALLBACK.
func (wo *WorkflowOrchestrator) checkDrugInteraction(ctx context.Context, drug1, drug2 ClinicalCode) (*DrugInteraction, error) {
	if wo.kbManager == nil || wo.kbManager.Interactions() == nil {
		return nil, fmt.Errorf("%w: KB-2 Interactions service not available", ErrKBUnavailable)
	}

	result, err := wo.kbManager.Interactions().CheckInteraction(ctx, drug1.Code, drug2.Code)
	if err != nil {
		return nil, fmt.Errorf("KB-2 interaction check error: %w", err)
	}

	if result.HasInteraction {
		return &DrugInteraction{
			Type:           result.Type,
			Severity:       result.Severity,
			Description:    result.Description,
			Recommendation: result.Recommendation,
		}, nil
	}

	return nil, nil
}

// getStandardDosage retrieves standard dosage from KB-1 Dosing service.
// Returns error if KB-1 is unavailable - NO FALLBACK.
func (wo *WorkflowOrchestrator) getStandardDosage(ctx context.Context, drug ClinicalCode) (Dosage, error) {
	if wo.kbManager == nil || wo.kbManager.Dosing() == nil {
		return Dosage{}, fmt.Errorf("%w: KB-1 Dosing service not available", ErrKBUnavailable)
	}

	dosageInfo, err := wo.kbManager.Dosing().GetStandardDosage(ctx, drug.Code)
	if err != nil {
		return Dosage{}, fmt.Errorf("KB-1 dosage lookup error for %s: %w", drug.Code, err)
	}

	return Dosage{
		Value:     dosageInfo.StandardDose,
		Unit:      dosageInfo.Unit,
		Route:     dosageInfo.Route,
		Frequency: dosageInfo.Frequency,
	}, nil
}

// getRenalAdjustmentFactor returns dose adjustment factor based on eGFR.
// This is clinical logic, not KB-dependent.
func (wo *WorkflowOrchestrator) getRenalAdjustmentFactor(egfr float64) float64 {
	if egfr >= 60 {
		return 1.0
	} else if egfr >= 30 {
		return 0.5
	} else if egfr >= 15 {
		return 0.25
	}
	return 0.0 // Contraindicated
}

// getHepaticAdjustmentFactor returns dose adjustment factor based on Child-Pugh class.
// This is clinical logic, not KB-dependent.
func (wo *WorkflowOrchestrator) getHepaticAdjustmentFactor(childPugh string) float64 {
	switch childPugh {
	case "A":
		return 1.0
	case "B":
		return 0.5
	case "C":
		return 0.25
	}
	return 1.0
}

// calculateQualityFactors calculates quality scores from KB-3, KB-5, KB-6.
// Returns error if any required KB is unavailable - NO FALLBACK.
func (wo *WorkflowOrchestrator) calculateQualityFactors(ctx context.Context, drug DosedDrug, input *WorkflowInput) (QualityFactors, error) {
	factors := QualityFactors{}

	// Get guideline score from KB-3 - REQUIRED
	if wo.kbManager.Guidelines() == nil {
		return factors, fmt.Errorf("%w: KB-3 Guidelines service not available", ErrKBUnavailable)
	}

	recommendations, err := wo.kbManager.Guidelines().GetRecommendedDrugs(
		ctx, input.Question.Indication, input.Question.TargetDrugClass)
	if err != nil {
		return factors, fmt.Errorf("KB-3 guidelines query failed: %w", err)
	}

	for _, rec := range recommendations {
		if rec.RxNormCode == drug.Medication.Code {
			factors.Guideline = getEvidenceScore(rec.EvidenceGrade)
			break
		}
	}
	if factors.Guideline == 0 {
		factors.Guideline = 0.50 // Drug found but no specific guideline match
	}

	// Get efficacy score from KB-6 - REQUIRED
	if wo.kbManager.Efficacy() == nil {
		return factors, fmt.Errorf("%w: KB-6 Efficacy service not available", ErrKBUnavailable)
	}

	efficacyScore, err := wo.kbManager.Efficacy().GetEfficacyScore(
		ctx, drug.Medication.Code, input.Question.Indication)
	if err != nil {
		return factors, fmt.Errorf("KB-6 efficacy query failed: %w", err)
	}
	factors.Efficacy = efficacyScore.EfficacyScore

	// Get monitoring score from KB-5 - REQUIRED
	if wo.kbManager.Monitoring() == nil {
		return factors, fmt.Errorf("%w: KB-5 Monitoring service not available", ErrKBUnavailable)
	}

	monReqs, err := wo.kbManager.Monitoring().GetMonitoringRequirements(ctx, drug.Medication.Code)
	if err != nil {
		return factors, fmt.Errorf("KB-5 monitoring query failed: %w", err)
	}
	// Lower monitoring complexity = higher score
	factors.Monitoring = 1.0 - monReqs.MonitoringScore

	// Calculate safety score based on warnings (no KB call, derived from earlier checks)
	if len(drug.Warnings) == 0 {
		factors.Safety = 0.95
	} else {
		warningPenalty := float64(len(drug.Warnings)) * 0.05
		factors.Safety = max(0.50, 0.95-warningPenalty)
	}

	// Interaction score from KB-2 checks already done (no additional KB call)
	// If we got here, interactions were already checked in Phase 2
	factors.Interaction = 0.95 // High score for passing interaction checks
	if len(drug.Warnings) > 0 {
		// Reduce for any mild/moderate interactions that generated warnings
		factors.Interaction = 0.80
	}

	return factors, nil
}

// getEvidenceScore converts evidence grade to numeric score.
func getEvidenceScore(grade string) float64 {
	switch grade {
	case "A", "1A", "I-A":
		return 0.95
	case "B", "1B", "I-B":
		return 0.85
	case "C", "2A", "II-A":
		return 0.75
	case "D", "2B", "II-B":
		return 0.65
	default:
		return 0.50
	}
}

func (wo *WorkflowOrchestrator) generateRationale(drug DosedDrug, factors QualityFactors, question ClinicalQuestion) string {
	rationale := fmt.Sprintf("%s recommended for %s. ",
		drug.Medication.Display, question.Indication)

	if len(drug.Adjustments) > 0 {
		rationale += fmt.Sprintf("Dose adjusted for %v. ", drug.Adjustments)
	}

	rationale += fmt.Sprintf("Quality score: %.0f%% guideline, %.0f%% safety, %.0f%% efficacy.",
		factors.Guideline*100, factors.Safety*100, factors.Efficacy*100)

	return rationale
}
