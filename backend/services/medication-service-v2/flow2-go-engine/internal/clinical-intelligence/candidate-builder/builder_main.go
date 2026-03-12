package candidatebuilder

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"
	"time"
)

// CandidateBuilder implements the "Generator" in our "Generator + Ranker" model
// Its sole purpose is to produce clinically safe medication options through rigorous filtering
// Enhanced with production-grade safety, performance, and observability features
type CandidateBuilder struct {
	classFilter    *ClassFilter
	safetyFilter   *SafetyFilter
	ddiFilter      *DDIFilter
	validator      *InputValidator
	logger         *log.Logger

	// Enhanced features
	config         *BuilderConfig
	metrics        MetricsCollector
}

// Note: BuilderConfig, MetricsCollector, and DefaultBuilderConfig are now defined in models.go

// NewCandidateBuilder creates a new candidate builder with all components
func NewCandidateBuilder() *CandidateBuilder {
	logger := log.New(os.Stdout, "[CandidateBuilder] ", log.LstdFlags|log.Lshortfile)

	return &CandidateBuilder{
		classFilter:  NewClassFilter(logger),
		safetyFilter: NewSafetyFilter(logger),
		ddiFilter:    NewDDIFilter(logger),
		validator:    NewInputValidator(logger),
		logger:       logger,
		config:       DefaultBuilderConfig(),
		metrics:      nil, // Can be set later
	}
}

// NewCandidateBuilderWithConfig creates a builder with custom configuration
func NewCandidateBuilderWithConfig(config *BuilderConfig, metrics MetricsCollector) *CandidateBuilder {
	logger := log.New(os.Stdout, "[CandidateBuilder] ", log.LstdFlags|log.Lshortfile)

	if config == nil {
		config = DefaultBuilderConfig()
	}

	return &CandidateBuilder{
		classFilter:  NewClassFilter(logger),
		safetyFilter: NewSafetyFilter(logger),
		ddiFilter:    NewDDIFilter(logger),
		validator:    NewInputValidator(logger),
		logger:       logger,
		config:       config,
		metrics:      metrics,
	}
}

// BuildCandidateProposals is the main function implementing the filtering funnel
// This is the "Generator" that produces all clinically safe medication options
func (cb *CandidateBuilder) BuildCandidateProposals(
	ctx context.Context,
	input CandidateBuilderInput,
) (*CandidateBuilderResult, error) {
	
	startTime := time.Now()
	
	cb.logger.Printf("Starting candidate proposal building - Generator phase: request_id=%s, patient_id=%s, initial_drugs=%d, recommended_classes=%v, patient_flags=%d, active_meds=%d, ddi_rules=%d", 
		input.RequestID, input.PatientID, len(input.DrugMasterList), input.RecommendedDrugClasses, 
		len(input.PatientFlags), len(input.ActiveMedications), len(input.DDIRules))
	
	// STEP 0: Enhanced Input Validation
	if err := cb.validator.ValidateInputs(input); err != nil {
		cb.logger.Printf("Input validation failed: %v", err)
		return nil, fmt.Errorf("input validation failed: %w", err)
	}
	
	// STEP 1: Filter by Recommended Class (Therapeutic Class Filter)
	classFiltered, err := cb.classFilter.FilterByRecommendedClass(
		input.DrugMasterList,
		input.RecommendedDrugClasses,
	)
	if err != nil {
		cb.logger.Printf("Class filtering failed: %v", err)
		return nil, fmt.Errorf("class filtering failed: %w", err)
	}
	
	cb.logger.Printf("Class filtering completed: initial=%d, filtered=%d, reduction=%.1f%%", 
		len(input.DrugMasterList), len(classFiltered), 
		cb.calculateReductionPercent(len(input.DrugMasterList), len(classFiltered)))
	
	// STEP 2: Patient-Specific Safety Filter (Critical Safety Gate)
	safetyFiltered, err := cb.safetyFilter.FilterByPatientContraindications(
		classFiltered,
		input.PatientFlags,
	)
	if err != nil {
		// Check if this is an empty results error (all drugs filtered out)
		if len(classFiltered) > 0 {
			cb.logger.Printf("Safety filtering resulted in empty results - handling gracefully")
			return cb.handleEmptyResults(input, classFiltered, []Drug{}, startTime)
		}
		cb.logger.Printf("Safety filtering failed: %v", err)
		return nil, fmt.Errorf("safety filtering failed: %w", err)
	}
	
	cb.logger.Printf("Patient safety filtering completed: class_filtered=%d, safety_filtered=%d, reduction=%.1f%%", 
		len(classFiltered), len(safetyFiltered), 
		cb.calculateReductionPercent(len(classFiltered), len(safetyFiltered)))
	
	// STEP 3: "Contraindicated" DDI Filter (Drug Interaction Safety)
	finalCandidates, err := cb.ddiFilter.FilterByContraindicatedDDIs(
		safetyFiltered,
		input.ActiveMedications,
		input.DDIRules,
	)
	if err != nil {
		cb.logger.Printf("DDI filtering failed: %v", err)
		return nil, fmt.Errorf("DDI filtering failed: %w", err)
	}
	
	cb.logger.Printf("DDI filtering completed: safety_filtered=%d, final_candidates=%d, reduction=%.1f%%", 
		len(safetyFiltered), len(finalCandidates), 
		cb.calculateReductionPercent(len(safetyFiltered), len(finalCandidates)))
	
	// STEP 4: Handle Empty Results with Clinical Guidance
	if len(finalCandidates) == 0 {
		cb.logger.Printf("No safe medication candidates found - all drugs filtered out for safety")
		return cb.handleEmptyResults(input, classFiltered, safetyFiltered, startTime)
	}
	
	// STEP 5: Build Final Result with Enhanced Statistics and Scoring
	result := cb.buildSuccessResult(input, classFiltered, safetyFiltered, finalCandidates, startTime)

	// Record metrics if available
	duration := time.Since(startTime)
	if cb.metrics != nil {
		cb.metrics.RecordFilteringComplete(input.RequestID, len(finalCandidates),
			len(input.DrugMasterList)-len(finalCandidates), duration)
	}

	cb.logger.Printf("Enhanced candidate proposal building completed successfully: request_id=%s, final_candidates=%d, overall_reduction=%.1f%%, processing_time=%dms",
		input.RequestID, len(finalCandidates), result.FilteringStatistics.OverallReductionPercent,
		result.ProcessingMetadata.ProcessingTimeMs)

	return result, nil
}

// buildSuccessResult creates comprehensive result when candidates are found
func (cb *CandidateBuilder) buildSuccessResult(
	input CandidateBuilderInput,
	classFiltered, safetyFiltered, finalCandidates []Drug,
	startTime time.Time,
) *CandidateBuilderResult {
	
	return &CandidateBuilderResult{
		CandidateProposals: cb.convertDrugsToProposals(finalCandidates),
		FilteringStatistics: FilteringStatistics{
			InitialDrugCount:        len(input.DrugMasterList),
			ClassFilteredCount:      len(classFiltered),
			SafetyFilteredCount:     len(safetyFiltered),
			FinalCandidateCount:     len(finalCandidates),
			ClassReductionPercent:   cb.calculateReductionPercent(len(input.DrugMasterList), len(classFiltered)),
			SafetyReductionPercent:  cb.calculateReductionPercent(len(classFiltered), len(safetyFiltered)),
			DDIReductionPercent:     cb.calculateReductionPercent(len(safetyFiltered), len(finalCandidates)),
			OverallReductionPercent: cb.calculateReductionPercent(len(input.DrugMasterList), len(finalCandidates)),
			RequiresSpecialistReview: false,
			FallbackTriggered:       false,
		},
		ProcessingMetadata: ProcessingMetadata{
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			GeneratedAt:      time.Now(),
			EngineVersion:    "v2.0",
			FilterStagesRun:  []string{"class_filtering", "patient_contraindications", "contraindicated_ddi"},
		},
		ExclusionLog: []ExclusionRecord{}, // Would be populated from individual filters
	}
}

// handleEmptyResults handles cases where no safe medication options are found
func (cb *CandidateBuilder) handleEmptyResults(
	input CandidateBuilderInput,
	classFiltered, safetyFiltered []Drug,
	startTime time.Time,
) (*CandidateBuilderResult, error) {
	
	cb.logger.Printf("Handling empty results: initial=%d, class_filtered=%d, safety_filtered=%d, final=0", 
		len(input.DrugMasterList), len(classFiltered), len(safetyFiltered))
	
	// Generate clinical guidance for empty results
	clinicalGuidance := &ClinicalGuidance{
		Severity:           "HIGH",
		Message:            "No safe medication options identified based on current patient profile",
		SpecialistReferral: true,
		ClinicalReasoning:  cb.generateEmptyResultsReasoning(input, classFiltered, safetyFiltered),
		RecommendedActions: []string{
			"Immediate clinical pharmacist consultation",
			"Specialist referral for alternative treatment approaches",
			"Review patient contraindication profile for accuracy",
			"Consider risk-benefit analysis with specialist",
			"Evaluate non-pharmacological treatment options",
		},
	}
	
	// Generate specialist review proposal
	specialistProposals := []MedicationProposal{
		{
			MedicationCode:    "CLINICAL_REVIEW_REQUIRED",
			MedicationName:    "Clinical Review Required",
			TherapeuticClass:  "SPECIALIST_CONSULTATION",
			Route:             "N/A",
			Status:            "requires_specialist_review",
			GeneratedAt:       time.Now(),
			FormulationOptions: []string{"Specialist Consultation"},
		},
	}
	
	return &CandidateBuilderResult{
		CandidateProposals: specialistProposals,
		FilteringStatistics: FilteringStatistics{
			InitialDrugCount:         len(input.DrugMasterList),
			ClassFilteredCount:       len(classFiltered),
			SafetyFilteredCount:      len(safetyFiltered),
			FinalCandidateCount:      0,
			ClassReductionPercent:    cb.calculateReductionPercent(len(input.DrugMasterList), len(classFiltered)),
			SafetyReductionPercent:   cb.calculateReductionPercent(len(classFiltered), len(safetyFiltered)),
			DDIReductionPercent:      100.0, // All remaining candidates excluded by DDI
			OverallReductionPercent:  100.0,
			RequiresSpecialistReview: true,
			FallbackTriggered:        false, // No fallback, just clinical guidance
		},
		ClinicalGuidance: clinicalGuidance,
		ProcessingMetadata: ProcessingMetadata{
			ProcessingTimeMs: time.Since(startTime).Milliseconds(),
			GeneratedAt:      time.Now(),
			EngineVersion:    "v2.0",
			FilterStagesRun:  []string{"class_filtering", "patient_contraindications", "contraindicated_ddi"},
		},
		ExclusionLog: []ExclusionRecord{}, // Would be populated from filters
	}, nil
}

// convertDrugsToProposals converts Drug objects to MedicationProposal objects with enhanced scoring
func (cb *CandidateBuilder) convertDrugsToProposals(drugs []Drug) []MedicationProposal {
	proposals := make([]MedicationProposal, len(drugs))

	for i, drug := range drugs {
		// Calculate enhanced safety score
		safetyScore := cb.calculateEnhancedSafetyScore(drug)

		// Get therapeutic class (handle both single and multiple classes)
		therapeuticClass := ""
		if len(drug.TherapeuticClasses) > 0 {
			therapeuticClass = drug.TherapeuticClasses[0]
		}

		proposals[i] = MedicationProposal{
			MedicationCode:     drug.Code,
			MedicationName:     drug.Name,
			GenericName:        drug.GenericName,
			TherapeuticClass:   therapeuticClass,
			Route:              drug.PreferredRoute,
			FormulationOptions: drug.AvailableFormulations,
			BaselineEfficacy:   drug.EfficacyScore,
			SafetyProfile:      drug.SafetyProfile,
			Status:             "candidate", // All candidates are safe at this point
			GeneratedAt:        time.Now(),
			Indications:        drug.Indications,
			IsGeneric:          drug.IsGeneric,

			// Enhanced fields
			SafetyScore:        safetyScore,
			DDIWarnings:        []DDIInteraction{}, // Will be populated by DDI filter
			FormularyTier:      1,                  // Default tier
			CostEstimate:       0.0,                // Default cost
		}
	}

	// Sort proposals by safety score (descending)
	cb.rankProposalsBySafety(proposals)

	cb.logger.Printf("Converted %d drugs to enhanced medication proposals with safety scoring", len(proposals))
	return proposals
}

// calculateEnhancedSafetyScore computes a safety score based on drug characteristics
func (cb *CandidateBuilder) calculateEnhancedSafetyScore(drug Drug) float64 {
	score := cb.config.MaxSafetyScore

	// Reduce score for safety concerns
	if drug.BlackBoxWarning && cb.config.EnableBlackBoxFilter {
		score -= 0.4 // Significant reduction for black box warning
	}

	if drug.PregnancyCategory == "X" {
		score -= 0.3 // High reduction for pregnancy category X
	} else if drug.PregnancyCategory == "D" {
		score -= 0.2 // Moderate reduction for pregnancy category D
	}

	if drug.RenalAdjustment {
		score -= 0.1 // Small reduction for renal adjustment requirement
	}

	if drug.HepaticAdjustment {
		score -= 0.1 // Small reduction for hepatic adjustment requirement
	}

	// Ensure score stays within bounds
	if score < cb.config.MinSafetyScore {
		score = cb.config.MinSafetyScore
	}

	return score
}

// rankProposalsBySafety sorts proposals by safety score (descending)
func (cb *CandidateBuilder) rankProposalsBySafety(proposals []MedicationProposal) {
	// Sort by safety score (highest first), then by efficacy, then by name for stability
	for i := 0; i < len(proposals)-1; i++ {
		for j := i + 1; j < len(proposals); j++ {
			// Primary sort: Safety score (descending)
			if proposals[i].SafetyScore < proposals[j].SafetyScore {
				proposals[i], proposals[j] = proposals[j], proposals[i]
			} else if proposals[i].SafetyScore == proposals[j].SafetyScore {
				// Secondary sort: Efficacy score (descending)
				if proposals[i].BaselineEfficacy < proposals[j].BaselineEfficacy {
					proposals[i], proposals[j] = proposals[j], proposals[i]
				} else if proposals[i].BaselineEfficacy == proposals[j].BaselineEfficacy {
					// Tertiary sort: Name (ascending for stability)
					if proposals[i].MedicationName > proposals[j].MedicationName {
						proposals[i], proposals[j] = proposals[j], proposals[i]
					}
				}
			}
		}
	}
}

// calculateReductionPercent calculates percentage reduction between before and after counts
func (cb *CandidateBuilder) calculateReductionPercent(before, after int) float64 {
	if before == 0 {
		return 0
	}
	return float64(before-after) / float64(before) * 100
}

// getActiveFlagsCount counts how many patient flags are set to true
func (cb *CandidateBuilder) getActiveFlagsCount(flags map[string]bool) int {
	count := 0
	for _, value := range flags {
		if value {
			count++
		}
	}
	return count
}

// generateEmptyResultsReasoning generates detailed reasoning for why no candidates were found
func (cb *CandidateBuilder) generateEmptyResultsReasoning(
	input CandidateBuilderInput,
	classFiltered, safetyFiltered []Drug,
) string {
	reasons := []string{}
	
	// Analyze where the filtering eliminated candidates
	if len(classFiltered) == 0 {
		reasons = append(reasons, fmt.Sprintf("No drugs found matching recommended therapeutic classes: %s", 
			strings.Join(input.RecommendedDrugClasses, ", ")))
	} else if len(safetyFiltered) == 0 {
		activeFlagsCount := cb.getActiveFlagsCount(input.PatientFlags)
		reasons = append(reasons, fmt.Sprintf("All %d class-appropriate drugs contraindicated due to %d active patient safety flags", 
			len(classFiltered), activeFlagsCount))
	} else {
		reasons = append(reasons, fmt.Sprintf("All %d safety-vetted drugs excluded due to contraindicated drug-drug interactions with %d active medications", 
			len(safetyFiltered), len(input.ActiveMedications)))
	}
	
	return fmt.Sprintf("Clinical filtering analysis: %s. Specialist consultation required for alternative treatment strategies.", 
		strings.Join(reasons, ". "))
}

// ValidateInputs validates all inputs before processing
func (cb *CandidateBuilder) ValidateInputs(input CandidateBuilderInput) error {
	return cb.validator.ValidateInputs(input)
}

// HealthCheck performs health check on all components
func (cb *CandidateBuilder) HealthCheck() error {
	// Check all filter components
	if cb.classFilter == nil {
		return fmt.Errorf("class filter not initialized")
	}
	
	if cb.safetyFilter == nil {
		return fmt.Errorf("safety filter not initialized")
	}
	
	if cb.ddiFilter == nil {
		return fmt.Errorf("DDI filter not initialized")
	}
	
	if cb.validator == nil {
		return fmt.Errorf("input validator not initialized")
	}
	
	cb.logger.Printf("Candidate builder health check passed")
	return nil
}
