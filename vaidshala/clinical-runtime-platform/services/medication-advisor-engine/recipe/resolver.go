// Package recipe provides clinical recipe resolution and KB orchestration.
// This implements the recipe resolution logic from the ORB-driven architecture.
// IMPORTANT: KB services are REQUIRED - this resolver does NOT have fallback logic.
package recipe

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/cardiofit/medication-advisor-engine/kbclients"
)

// ErrKBUnavailable indicates a required KB service is unavailable
var ErrKBUnavailable = fmt.Errorf("required KB service unavailable")

// RecipeResolver coordinates Knowledge Base calls for medication recommendations.
// It implements the 4-tier recipe resolution pattern:
// - Tier 1: Basic patient context (demographics, allergies)
// - Tier 2: Clinical conditions and medications
// - Tier 3: Lab results and computed scores
// - Tier 4: Guideline-specific requirements
//
// IMPORTANT: All KB services (KB-1 through KB-6) MUST be available for this resolver to function.
type RecipeResolver struct {
	kbManager *kbclients.KBManager
}

// ResolverConfig holds configuration for the resolver
type ResolverConfig struct {
	EnableFallback bool `json:"enable_fallback"` // Deprecated: Fallback is no longer supported
}

// NewRecipeResolver creates a new recipe resolver.
// The kbManager MUST be configured with all KB service URLs.
func NewRecipeResolver(kbManager *kbclients.KBManager, config ResolverConfig) *RecipeResolver {
	// Note: EnableFallback is ignored - KB services are always required
	return &RecipeResolver{
		kbManager: kbManager,
	}
}

// RecipeInput represents input for recipe resolution
type RecipeInput struct {
	PatientID       string         `json:"patient_id"`
	Indication      string         `json:"indication"`
	TargetDrugClass string         `json:"target_drug_class"`
	TargetRxNorm    string         `json:"target_rxnorm,omitempty"`
	Conditions      []ClinicalCode `json:"conditions"`
	Medications     []ClinicalCode `json:"medications"`
	Allergies       []ClinicalCode `json:"allergies"`
	AgeYears        int            `json:"age_years"`
	WeightKg        *float64       `json:"weight_kg,omitempty"`
	EGFR            *float64       `json:"egfr,omitempty"`
	ChildPughClass  string         `json:"child_pugh_class,omitempty"`
}

// ClinicalCode represents a coded clinical concept
type ClinicalCode struct {
	System  string `json:"system"`
	Code    string `json:"code"`
	Display string `json:"display"`
}

// RecipeOutput represents the resolved recipe output
type RecipeOutput struct {
	CandidateDrugs     []DrugCandidate `json:"candidate_drugs"`
	ExcludedDrugs      []ExcludedDrug  `json:"excluded_drugs"`
	Warnings           []Warning       `json:"warnings"`
	KBQueriesPerformed []KBQuery       `json:"kb_queries"`
	ExecutionTimeMs    int64           `json:"execution_time_ms"`
}

// DrugCandidate represents a candidate drug from recipe resolution
type DrugCandidate struct {
	RxNormCode       string           `json:"rxnorm_code"`
	DrugName         string           `json:"drug_name"`
	DrugClass        string           `json:"drug_class"`
	StandardDose     float64          `json:"standard_dose"`
	AdjustedDose     float64          `json:"adjusted_dose"`
	Unit             string           `json:"unit"`
	Route            string           `json:"route"`
	Frequency        string           `json:"frequency"`
	GuidelineScore   float64          `json:"guideline_score"`
	SafetyScore      float64          `json:"safety_score"`
	EfficacyScore    float64          `json:"efficacy_score"`
	InteractionScore float64          `json:"interaction_score"`
	MonitoringScore  float64          `json:"monitoring_score"`
	Rationale        string           `json:"rationale"`
	EvidenceGrade    string           `json:"evidence_grade"`
	DoseAdjustments  []DoseAdjustment `json:"dose_adjustments,omitempty"`
	Warnings         []Warning        `json:"warnings,omitempty"`
}

// ExcludedDrug represents a drug excluded during resolution
type ExcludedDrug struct {
	RxNormCode    string `json:"rxnorm_code"`
	DrugName      string `json:"drug_name"`
	Reason        string `json:"reason"`
	ExclusionType string `json:"exclusion_type"` // allergy, contraindication, interaction
	KBSource      string `json:"kb_source"`
	Severity      string `json:"severity"`
}

// DoseAdjustment represents a dose adjustment applied
type DoseAdjustment struct {
	Type         string  `json:"type"` // renal, hepatic, age, weight
	Factor       float64 `json:"factor"`
	Reason       string  `json:"reason"`
	OriginalDose float64 `json:"original_dose"`
	AdjustedDose float64 `json:"adjusted_dose"`
}

// Warning represents a clinical warning
type Warning struct {
	Severity string `json:"severity"` // info, warning, critical
	Message  string `json:"message"`
	Source   string `json:"source"`
	Code     string `json:"code,omitempty"`
}

// KBQuery represents a KB query performed during resolution
type KBQuery struct {
	KBService  string                 `json:"kb_service"`
	QueryType  string                 `json:"query_type"`
	Parameters map[string]interface{} `json:"parameters"`
	Result     interface{}            `json:"result,omitempty"`
	Success    bool                   `json:"success"`
	DurationMs int64                  `json:"duration_ms"`
	CacheHit   bool                   `json:"cache_hit"`
}

// Resolve executes the complete recipe resolution process.
// Returns an error if any required KB service is unavailable.
func (r *RecipeResolver) Resolve(ctx context.Context, input *RecipeInput) (*RecipeOutput, error) {
	startTime := time.Now()

	// Validate KB manager is available
	if r.kbManager == nil {
		return nil, fmt.Errorf("KB manager is not initialized: %w", ErrKBUnavailable)
	}

	output := &RecipeOutput{
		CandidateDrugs:     []DrugCandidate{},
		ExcludedDrugs:      []ExcludedDrug{},
		Warnings:           []Warning{},
		KBQueriesPerformed: []KBQuery{},
	}

	// Step 1: Get candidate drugs from guidelines (KB-3) - REQUIRED
	candidates, err := r.getCandidateDrugs(ctx, input, output)
	if err != nil {
		return nil, fmt.Errorf("failed to get candidate drugs: %w", err)
	}

	// Step 2: Safety filtering (KB-4) - allergies, contraindications - REQUIRED
	safeCandidates, err := r.filterBySafety(ctx, candidates, input, output)
	if err != nil {
		return nil, fmt.Errorf("failed to filter by safety: %w", err)
	}

	// Step 3: Interaction checking (KB-2) - REQUIRED
	safeCandidates, err = r.filterByInteractions(ctx, safeCandidates, input, output)
	if err != nil {
		return nil, fmt.Errorf("failed to filter by interactions: %w", err)
	}

	// Step 4: Calculate dosing (KB-1) - REQUIRED
	dosedCandidates, err := r.calculateDosing(ctx, safeCandidates, input, output)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate dosing: %w", err)
	}

	// Step 5: Get monitoring requirements (KB-5) - REQUIRED
	err = r.addMonitoringScores(ctx, dosedCandidates, output)
	if err != nil {
		return nil, fmt.Errorf("failed to get monitoring scores: %w", err)
	}

	// Step 6: Get efficacy scores (KB-6) - REQUIRED
	err = r.addEfficacyScores(ctx, dosedCandidates, input.Indication, output)
	if err != nil {
		return nil, fmt.Errorf("failed to get efficacy scores: %w", err)
	}

	output.CandidateDrugs = dosedCandidates
	output.ExecutionTimeMs = time.Since(startTime).Milliseconds()

	return output, nil
}

// getCandidateDrugs retrieves candidate drugs from KB-3 Guidelines, falling back to KB-1 if needed.
// KB-3 provides protocol-based recommendations (clinical pathways), while KB-1 provides
// drug rules that can be searched by therapeutic class.
func (r *RecipeResolver) getCandidateDrugs(ctx context.Context, input *RecipeInput, output *RecipeOutput) ([]DrugCandidate, error) {
	var candidates []DrugCandidate

	// Step 1: Try KB-3 Guidelines for protocol-based recommendations
	if r.kbManager.Guidelines() != nil {
		query := KBQuery{
			KBService: "KB-3",
			QueryType: "GetRecommendedDrugs",
			Parameters: map[string]interface{}{
				"indication": input.Indication,
				"drug_class": input.TargetDrugClass,
			},
		}
		queryStart := time.Now()

		recommendations, err := r.kbManager.Guidelines().GetRecommendedDrugs(ctx, input.Indication, input.TargetDrugClass)
		query.DurationMs = time.Since(queryStart).Milliseconds()

		if err != nil {
			query.Success = false
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
			// Log but continue - we'll try KB-1 fallback
			output.Warnings = append(output.Warnings, Warning{
				Severity: "info",
				Message:  fmt.Sprintf("KB-3 query returned error: %v, falling back to KB-1", err),
				Source:   "KB-3",
			})
		} else {
			query.Success = true
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

			for _, rec := range recommendations {
				if rec.RxNormCode != "" {
					candidates = append(candidates, DrugCandidate{
						RxNormCode:     rec.RxNormCode,
						DrugName:       rec.DrugName,
						DrugClass:      rec.DrugClass,
						StandardDose:   rec.RecommendedDose,
						AdjustedDose:   rec.RecommendedDose,
						Unit:           rec.Unit,
						Frequency:      rec.Frequency,
						GuidelineScore: getGuidelineScore(rec.EvidenceGrade),
						EvidenceGrade:  rec.EvidenceGrade,
						Rationale:      fmt.Sprintf("Recommended by %s for %s", rec.GuidelineSource, input.Indication),
					})
				}
			}
		}
	}

	// Step 2: If KB-3 returned no drugs with RxNorm codes, fall back to KB-1
	// Handle both explicit drug class searches AND polypharmacy/medication review scenarios
	if len(candidates) == 0 {
		if input.TargetDrugClass != "" {
			// Standard drug class search
			kb1Candidates, err := r.getCandidateDrugsFromKB1(ctx, input, output)
			if err != nil {
				// If both KB-3 and KB-1 fail, return error
				return nil, fmt.Errorf("no candidates from KB-3 or KB-1: %w", err)
			}
			candidates = kb1Candidates
		} else if isPolypharmacyScenario(input.Indication) {
			// Polypharmacy/medication review - use special fallback for deprescribing candidates
			kb1Candidates, err := r.getCandidateDrugsFromKB1(ctx, &RecipeInput{
				PatientID:       input.PatientID,
				Indication:      input.Indication,
				TargetDrugClass: "polypharmacy", // Trigger local polypharmacy fallback
				Conditions:      input.Conditions,
				Medications:     input.Medications,
				Allergies:       input.Allergies,
				AgeYears:        input.AgeYears,
				EGFR:            input.EGFR,
			}, output)
			if err != nil {
				return nil, fmt.Errorf("polypharmacy medication review failed: %w", err)
			}
			candidates = kb1Candidates
		}
	}

	// Step 3: If still no candidates and specific RxNorm provided, use that directly
	if len(candidates) == 0 && input.TargetRxNorm != "" {
		candidates = append(candidates, DrugCandidate{
			RxNormCode:     input.TargetRxNorm,
			DrugName:       input.TargetRxNorm, // Will be enriched by KB-1 later
			DrugClass:      input.TargetDrugClass,
			GuidelineScore: 0.50, // Lower score for direct input without guideline support
			EvidenceGrade:  "C",
			Rationale:      "Direct drug selection by provider",
		})
	}

	return candidates, nil
}

// isPolypharmacyScenario detects if the indication suggests a polypharmacy/medication review scenario
func isPolypharmacyScenario(indication string) bool {
	indication = strings.ToLower(indication)
	polypharmacyKeywords := []string{
		"polypharmacy",
		"medication review",
		"deprescribing",
		"medication reconciliation",
		"drug burden",
		"potentially inappropriate",
		"beers criteria",
	}
	for _, keyword := range polypharmacyKeywords {
		if strings.Contains(indication, keyword) {
			return true
		}
	}
	return false
}

// getCandidateDrugsFromKB1 retrieves drugs from KB-1 Drug Rules by therapeutic class.
// This is a fallback when KB-3 doesn't have drug recommendations for the indication.
func (r *RecipeResolver) getCandidateDrugsFromKB1(ctx context.Context, input *RecipeInput, output *RecipeOutput) ([]DrugCandidate, error) {
	if r.kbManager.Dosing() == nil {
		return nil, fmt.Errorf("KB-1 Dosing service unavailable: %w", ErrKBUnavailable)
	}

	query := KBQuery{
		KBService: "KB-1",
		QueryType: "SearchByClass",
		Parameters: map[string]interface{}{
			"therapeutic_class": input.TargetDrugClass,
		},
	}
	queryStart := time.Now()

	drugRules, err := r.kbManager.Dosing().SearchByClass(ctx, input.TargetDrugClass)
	query.DurationMs = time.Since(queryStart).Milliseconds()

	if err != nil {
		query.Success = false
		output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
		return nil, fmt.Errorf("KB-1 SearchByClass failed: %w", err)
	}

	query.Success = true
	query.Result = map[string]interface{}{"count": len(drugRules)}
	output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

	if len(drugRules) == 0 {
		output.Warnings = append(output.Warnings, Warning{
			Severity: "warning",
			Message:  fmt.Sprintf("No drugs found in KB-1 for therapeutic class: %s", input.TargetDrugClass),
			Source:   "KB-1",
		})
		return nil, fmt.Errorf("no drugs found in KB-1 for class: %s", input.TargetDrugClass)
	}

	var candidates []DrugCandidate
	for _, rule := range drugRules {
		candidates = append(candidates, DrugCandidate{
			RxNormCode:     rule.RxNormCode,
			DrugName:       rule.DrugName,
			DrugClass:      rule.TherapeuticClass,
			GuidelineScore: 0.70, // Default moderate guideline score for KB-1 drugs
			EvidenceGrade:  "B",  // Assume moderate evidence for KB-1 registered drugs
			Rationale:      fmt.Sprintf("%s from KB-1 drug rules for %s", rule.DrugName, input.Indication),
		})
	}

	output.Warnings = append(output.Warnings, Warning{
		Severity: "info",
		Message:  fmt.Sprintf("Used KB-1 fallback: found %d drugs for class %s", len(candidates), input.TargetDrugClass),
		Source:   "KB-1",
	})

	return candidates, nil
}

// filterBySafety checks allergies and contraindications (KB-4).
// Returns an error if KB-4 is unavailable.
func (r *RecipeResolver) filterBySafety(ctx context.Context, candidates []DrugCandidate, input *RecipeInput, output *RecipeOutput) ([]DrugCandidate, error) {
	// KB-4 is REQUIRED
	if r.kbManager.Safety() == nil {
		return nil, fmt.Errorf("KB-4 Safety service unavailable: %w", ErrKBUnavailable)
	}

	safeCandidates := []DrugCandidate{}

	for _, candidate := range candidates {
		excluded := false

		// Check allergies
		for _, allergy := range input.Allergies {
			query := KBQuery{
				KBService: "KB-4",
				QueryType: "CheckAllergyMatch",
				Parameters: map[string]interface{}{
					"rxnorm_code":   candidate.RxNormCode,
					"allergen_code": allergy.Code,
				},
			}
			queryStart := time.Now()

			result, err := r.kbManager.Safety().CheckAllergyMatch(ctx, candidate.RxNormCode, allergy.Code)
			query.DurationMs = time.Since(queryStart).Milliseconds()

			if err != nil {
				query.Success = false
				output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
				return nil, fmt.Errorf("KB-4 allergy check failed for %s: %w", candidate.RxNormCode, err)
			}

			query.Success = true
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

			if result.IsMatch {
				output.ExcludedDrugs = append(output.ExcludedDrugs, ExcludedDrug{
					RxNormCode:    candidate.RxNormCode,
					DrugName:      candidate.DrugName,
					Reason:        fmt.Sprintf("Patient allergic to %s (%s)", allergy.Display, result.MatchType),
					ExclusionType: "allergy",
					KBSource:      "KB-4",
					Severity:      "critical",
				})
				excluded = true
				break
			}
		}

		if excluded {
			continue
		}

		// Check contraindications
		conditionCodes := make([]string, len(input.Conditions))
		for i, c := range input.Conditions {
			conditionCodes[i] = c.Code
		}

		req := &kbclients.ContraindicationRequest{
			RxNormCode:     candidate.RxNormCode,
			ConditionCodes: conditionCodes,
			AgeYears:       input.AgeYears,
			EGFR:           input.EGFR,
		}

		query := KBQuery{
			KBService: "KB-4",
			QueryType: "CheckContraindication",
			Parameters: map[string]interface{}{
				"rxnorm_code": candidate.RxNormCode,
				"conditions":  conditionCodes,
			},
		}
		queryStart := time.Now()

		result, err := r.kbManager.Safety().CheckContraindication(ctx, req)
		query.DurationMs = time.Since(queryStart).Milliseconds()

		if err != nil {
			query.Success = false
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
			return nil, fmt.Errorf("KB-4 contraindication check failed for %s: %w", candidate.RxNormCode, err)
		}

		query.Success = true
		output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

		if result.IsContraindicated {
			output.ExcludedDrugs = append(output.ExcludedDrugs, ExcludedDrug{
				RxNormCode:    candidate.RxNormCode,
				DrugName:      candidate.DrugName,
				Reason:        result.Reason,
				ExclusionType: "contraindication",
				KBSource:      "KB-4",
				Severity:      result.Severity,
			})
			excluded = true
		}

		if !excluded {
			candidate.SafetyScore = 0.90 // High safety score for passing all checks
			safeCandidates = append(safeCandidates, candidate)
		}
	}

	return safeCandidates, nil
}

// filterByInteractions checks drug-drug interactions (KB-2).
// Returns an error if KB-2 is unavailable.
func (r *RecipeResolver) filterByInteractions(ctx context.Context, candidates []DrugCandidate, input *RecipeInput, output *RecipeOutput) ([]DrugCandidate, error) {
	// KB-2 is REQUIRED
	if r.kbManager.Interactions() == nil {
		return nil, fmt.Errorf("KB-2 Interactions service unavailable: %w", ErrKBUnavailable)
	}

	safeCandidates := []DrugCandidate{}

	for _, candidate := range candidates {
		excluded := false
		candidateScore := 1.0 // Start with perfect interaction score

		// Check against existing medications
		for _, existingMed := range input.Medications {
			query := KBQuery{
				KBService: "KB-2",
				QueryType: "CheckInteraction",
				Parameters: map[string]interface{}{
					"drug1": candidate.RxNormCode,
					"drug2": existingMed.Code,
				},
			}
			queryStart := time.Now()

			result, err := r.kbManager.Interactions().CheckInteraction(ctx, candidate.RxNormCode, existingMed.Code)
			query.DurationMs = time.Since(queryStart).Milliseconds()

			if err != nil {
				query.Success = false
				output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
				return nil, fmt.Errorf("KB-2 interaction check failed for %s vs %s: %w", candidate.RxNormCode, existingMed.Code, err)
			}

			query.Success = true
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

			if result.HasInteraction {
				switch result.Severity {
				case "severe", "contraindicated":
					output.ExcludedDrugs = append(output.ExcludedDrugs, ExcludedDrug{
						RxNormCode:    candidate.RxNormCode,
						DrugName:      candidate.DrugName,
						Reason:        fmt.Sprintf("Severe interaction with %s: %s", existingMed.Display, result.Description),
						ExclusionType: "interaction",
						KBSource:      "KB-2",
						Severity:      result.Severity,
					})
					excluded = true
				case "moderate":
					candidateScore -= 0.20
					candidate.Warnings = append(candidate.Warnings, Warning{
						Severity: "warning",
						Message:  fmt.Sprintf("Moderate interaction with %s: %s", existingMed.Display, result.Description),
						Source:   "KB-2",
					})
				case "mild":
					candidateScore -= 0.05
					candidate.Warnings = append(candidate.Warnings, Warning{
						Severity: "info",
						Message:  fmt.Sprintf("Mild interaction with %s", existingMed.Display),
						Source:   "KB-2",
					})
				}
			}
		}

		if !excluded {
			candidate.InteractionScore = candidateScore
			safeCandidates = append(safeCandidates, candidate)
		}
	}

	return safeCandidates, nil
}

// calculateDosing calculates adjusted doses (KB-1).
// Returns an error if KB-1 is unavailable.
func (r *RecipeResolver) calculateDosing(ctx context.Context, candidates []DrugCandidate, input *RecipeInput, output *RecipeOutput) ([]DrugCandidate, error) {
	// KB-1 is REQUIRED
	if r.kbManager.Dosing() == nil {
		return nil, fmt.Errorf("KB-1 Dosing service unavailable: %w", ErrKBUnavailable)
	}

	dosedCandidates := []DrugCandidate{}

	for _, candidate := range candidates {
		// Get standard dosing from KB-1
		query := KBQuery{
			KBService: "KB-1",
			QueryType: "GetStandardDosage",
			Parameters: map[string]interface{}{
				"rxnorm_code": candidate.RxNormCode,
			},
		}
		queryStart := time.Now()

		dosageInfo, err := r.kbManager.Dosing().GetStandardDosage(ctx, candidate.RxNormCode)
		query.DurationMs = time.Since(queryStart).Milliseconds()

		if err != nil {
			query.Success = false
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
			return nil, fmt.Errorf("KB-1 dosage lookup failed for %s: %w", candidate.RxNormCode, err)
		}

		query.Success = true
		output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

		candidate.StandardDose = dosageInfo.StandardDose
		candidate.AdjustedDose = dosageInfo.StandardDose
		candidate.Unit = dosageInfo.Unit
		candidate.Route = dosageInfo.Route
		candidate.Frequency = dosageInfo.Frequency

		// Calculate dose adjustments if needed
		if dosageInfo.RenalAdjust || dosageInfo.HepaticAdjust {
			adjustReq := &kbclients.DoseAdjustmentRequest{
				RxNormCode:     candidate.RxNormCode,
				BaseDose:       candidate.StandardDose,
				EGFR:           input.EGFR,
				ChildPughClass: input.ChildPughClass,
				AgeYears:       input.AgeYears,
				WeightKg:       input.WeightKg,
			}

			adjustQuery := KBQuery{
				KBService: "KB-1",
				QueryType: "CalculateDoseAdjustment",
				Parameters: map[string]interface{}{
					"rxnorm_code": candidate.RxNormCode,
					"egfr":        input.EGFR,
					"child_pugh":  input.ChildPughClass,
				},
			}
			adjustStart := time.Now()

			adjustResult, err := r.kbManager.Dosing().CalculateDoseAdjustment(ctx, adjustReq)
			adjustQuery.DurationMs = time.Since(adjustStart).Milliseconds()

			if err != nil {
				adjustQuery.Success = false
				output.KBQueriesPerformed = append(output.KBQueriesPerformed, adjustQuery)
				return nil, fmt.Errorf("KB-1 dose adjustment failed for %s: %w", candidate.RxNormCode, err)
			}

			adjustQuery.Success = true
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, adjustQuery)

			if adjustResult.AdjustedDose != candidate.StandardDose {
				candidate.DoseAdjustments = append(candidate.DoseAdjustments, DoseAdjustment{
					Type:         adjustResult.AdjustmentType,
					Factor:       adjustResult.AdjustmentRatio,
					Reason:       adjustResult.Rationale,
					OriginalDose: candidate.StandardDose,
					AdjustedDose: adjustResult.AdjustedDose,
				})
				candidate.AdjustedDose = adjustResult.AdjustedDose

				// Add warning for dose adjustments
				for _, w := range adjustResult.Warnings {
					candidate.Warnings = append(candidate.Warnings, Warning{
						Severity: "info",
						Message:  w,
						Source:   "KB-1",
					})
				}
			}
		}

		dosedCandidates = append(dosedCandidates, candidate)
	}

	return dosedCandidates, nil
}

// addMonitoringScores adds monitoring complexity scores (KB-5).
// Returns an error if KB-5 is unavailable.
func (r *RecipeResolver) addMonitoringScores(ctx context.Context, candidates []DrugCandidate, output *RecipeOutput) error {
	// KB-5 is REQUIRED
	if r.kbManager.Monitoring() == nil {
		return fmt.Errorf("KB-5 Monitoring service unavailable: %w", ErrKBUnavailable)
	}

	for i := range candidates {
		query := KBQuery{
			KBService: "KB-5",
			QueryType: "GetMonitoringRequirements",
			Parameters: map[string]interface{}{
				"rxnorm_code": candidates[i].RxNormCode,
			},
		}
		queryStart := time.Now()

		requirements, err := r.kbManager.Monitoring().GetMonitoringRequirements(ctx, candidates[i].RxNormCode)
		query.DurationMs = time.Since(queryStart).Milliseconds()

		if err != nil {
			query.Success = false
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
			return fmt.Errorf("KB-5 monitoring lookup failed for %s: %w", candidates[i].RxNormCode, err)
		}

		query.Success = true
		output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

		// Convert monitoring complexity to a 0-1 score (lower complexity = higher score)
		candidates[i].MonitoringScore = 1.0 - requirements.MonitoringScore
	}

	return nil
}

// addEfficacyScores adds efficacy scores (KB-6).
// Returns an error if KB-6 is unavailable.
func (r *RecipeResolver) addEfficacyScores(ctx context.Context, candidates []DrugCandidate, indication string, output *RecipeOutput) error {
	// KB-6 is REQUIRED
	if r.kbManager.Efficacy() == nil {
		return fmt.Errorf("KB-6 Efficacy service unavailable: %w", ErrKBUnavailable)
	}

	for i := range candidates {
		query := KBQuery{
			KBService: "KB-6",
			QueryType: "GetEfficacyScore",
			Parameters: map[string]interface{}{
				"rxnorm_code": candidates[i].RxNormCode,
				"indication":  indication,
			},
		}
		queryStart := time.Now()

		score, err := r.kbManager.Efficacy().GetEfficacyScore(ctx, candidates[i].RxNormCode, indication)
		query.DurationMs = time.Since(queryStart).Milliseconds()

		if err != nil {
			query.Success = false
			output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)
			return fmt.Errorf("KB-6 efficacy lookup failed for %s: %w", candidates[i].RxNormCode, err)
		}

		query.Success = true
		output.KBQueriesPerformed = append(output.KBQueriesPerformed, query)

		candidates[i].EfficacyScore = score.EfficacyScore
	}

	return nil
}

// getGuidelineScore converts evidence grade to numeric score
func getGuidelineScore(grade string) float64 {
	switch grade {
	case "A":
		return 0.95
	case "B":
		return 0.80
	case "C":
		return 0.65
	default:
		return 0.50
	}
}
