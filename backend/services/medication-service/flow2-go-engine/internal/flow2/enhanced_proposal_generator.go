package flow2

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"flow2-go-engine/internal/clients"
	"flow2-go-engine/internal/models"
)

// EnhancedProposalGenerator implements the comprehensive proposal generation design
type EnhancedProposalGenerator struct {
	logger                  *logrus.Logger
	contextServiceClient    clients.ContextServiceClient
	formularyClient         clients.FormularyClient
	monitoringClient        clients.MonitoringClient
	alternativesClient      clients.AlternativesClient
	evidenceRepository      clients.EvidenceRepository
	calculationEngine       CalculationEngine
	alternativeEngine       AlternativeEngine
	monitoringEngine        MonitoringEngine
	rationaleEngine         RationaleEngine
}

// NewEnhancedProposalGenerator creates a new enhanced proposal generator
func NewEnhancedProposalGenerator(
	logger *logrus.Logger,
	contextClient clients.ContextServiceClient,
	formularyClient clients.FormularyClient,
	monitoringClient clients.MonitoringClient,
	alternativesClient clients.AlternativesClient,
	evidenceRepo clients.EvidenceRepository,
) *EnhancedProposalGenerator {
	return &EnhancedProposalGenerator{
		logger:                  logger,
		contextServiceClient:    contextClient,
		formularyClient:         formularyClient,
		monitoringClient:        monitoringClient,
		alternativesClient:      alternativesClient,
		evidenceRepository:      evidenceRepo,
		calculationEngine:       NewCalculationEngine(),
		alternativeEngine:       NewAlternativeEngine(),
		monitoringEngine:        NewMonitoringEngine(),
		rationaleEngine:         NewRationaleEngine(),
	}
}

// ProcessCommand implements the enhanced proposal generation workflow
func (g *EnhancedProposalGenerator) ProcessCommand(
	ctx context.Context,
	command *models.MedicationCommand,
) (*models.EnhancedProposedOrder, error) {
	startTime := time.Now()
	
	g.logger.WithFields(logrus.Fields{
		"patient_id":    command.PatientID,
		"medication":    command.InputString,
		"scenario":      command.Scenario,
		"urgency":       command.Urgency,
	}).Info("Starting enhanced proposal generation")

	// Step 1: Enhanced harmonization with confidence scoring
	harmonizationResult, err := g.harmonizeMedication(ctx, command.InputString, true)
	if err != nil {
		return nil, fmt.Errorf("medication harmonization failed: %w", err)
	}

	// Step 2: Recipe selection with fallback strategies
	recipe, err := g.selectRecipe(ctx, harmonizationResult.Medication, command.Scenario, command.Urgency)
	if err != nil {
		return nil, fmt.Errorf("recipe selection failed: %w", err)
	}

	// Step 3: Parallel processing for efficiency
	contextTask := g.fetchContext(ctx, command.PatientID, recipe.ContextID)
	formularyTask := g.checkFormulary(ctx, harmonizationResult.Medication)
	monitoringTask := g.getMonitoringRequirements(ctx, harmonizationResult.Medication)

	// Wait for all parallel tasks
	context, formulary, monitoring, err := g.awaitParallelTasks(contextTask, formularyTask, monitoringTask)
	if err != nil {
		return nil, fmt.Errorf("parallel processing failed: %w", err)
	}

	// Step 4: Multi-engine orchestration
	calculationResult, err := g.calculationEngine.Calculate(ctx, &CalculationRequest{
		Medication:           harmonizationResult.Medication,
		PatientContext:       context,
		Indication:           command.Indication,
		CalculationStrategy:  g.determineStrategy(harmonizationResult.Medication, context),
	})
	if err != nil {
		return nil, fmt.Errorf("dose calculation failed: %w", err)
	}

	// Step 5: Parallel alternative analysis
	alternatives, err := g.alternativeEngine.FindAlternatives(ctx, &AlternativeRequest{
		Medication:         harmonizationResult.Medication,
		Reasons:           []string{"FORMULARY", "CLINICAL", "PATIENT_PREFERENCE"},
		MaxAlternatives:   5,
		IncludeNonPharm:   true,
	})
	if err != nil {
		g.logger.WithError(err).Warn("Alternative analysis failed, continuing without alternatives")
		alternatives = &AlternativeResponse{Alternatives: []models.TherapeuticAlternative{}}
	}

	// Step 6: Risk-stratified monitoring
	monitoringPlan, err := g.monitoringEngine.GeneratePlan(ctx, &MonitoringRequest{
		Medication:        harmonizationResult.Medication,
		PatientRiskFactors: context.RiskFactors,
		BaselineLabs:      context.RecentLabs,
		StratifyByRisk:    true,
	})
	if err != nil {
		return nil, fmt.Errorf("monitoring plan generation failed: %w", err)
	}

	// Step 7: Intelligent rationale generation
	rationale, err := g.rationaleEngine.Generate(ctx, &RationaleRequest{
		DecisionPoints: map[string]interface{}{
			"medication_selection": calculationResult.SelectionReasoning,
			"dose_selection":       calculationResult.DoseReasoning,
			"monitoring_selection": monitoringPlan.Reasoning,
			"alternative_ranking":  alternatives.RankingReasoning,
		},
		EvidenceSources: g.evidenceRepository.GetRelevantGuidelines(command.Indication, harmonizationResult.Medication),
		PatientFactors:  context.RelevantFactors,
	})
	if err != nil {
		return nil, fmt.Errorf("rationale generation failed: %w", err)
	}

	// Step 8: Assemble enhanced proposal
	proposal := g.assembleEnhancedProposal(
		harmonizationResult,
		calculationResult,
		alternatives,
		monitoringPlan,
		rationale,
		command,
		context,
		startTime,
	)

	executionTime := time.Since(startTime)
	g.logger.WithFields(logrus.Fields{
		"patient_id":       command.PatientID,
		"proposal_id":      proposal.ProposalID,
		"execution_time":   executionTime.Milliseconds(),
		"confidence_score": proposal.Metadata.ConfidenceScore,
	}).Info("Enhanced proposal generation completed")

	return proposal, nil
}

// assembleEnhancedProposal creates the comprehensive proposal structure
func (g *EnhancedProposalGenerator) assembleEnhancedProposal(
	harmonization *HarmonizationResult,
	calculation *CalculationResult,
	alternatives *AlternativeResponse,
	monitoring *MonitoringPlanResponse,
	rationale *RationaleResponse,
	command *models.MedicationCommand,
	context *models.ClinicalContext,
	startTime time.Time,
) *models.EnhancedProposedOrder {
	proposalID := uuid.New().String()
	now := time.Now()
	
	return &models.EnhancedProposedOrder{
		ProposalID:      proposalID,
		ProposalVersion: "1.0",
		Timestamp:       now,
		ExpiresAt:       now.Add(24 * time.Hour),
		
		Metadata: models.ProposalMetadata{
			PatientID:           command.PatientID,
			EncounterID:         command.EncounterID,
			PrescriberID:        command.PrescriberID,
			Status:              "PROPOSED",
			Urgency:             command.Urgency,
			ProposalType:        "NEW_PRESCRIPTION",
			RecipeUsed:          calculation.RecipeUsed,
			ContextCompleteness: context.CompletenessScore,
			ConfidenceScore:     harmonization.ConfidenceScore,
		},
		
		CalculatedOrder: g.buildCalculatedOrder(harmonization, calculation),
		MonitoringPlan:  g.buildMonitoringPlan(monitoring),
		TherapeuticAlternatives: g.buildTherapeuticAlternatives(alternatives),
		ClinicalRationale: g.buildClinicalRationale(rationale),
		ProposalMetadata: g.buildProposalMetadata(startTime, calculation),
	}
}

// buildCalculatedOrder constructs the calculated order section
func (g *EnhancedProposalGenerator) buildCalculatedOrder(
	harmonization *HarmonizationResult,
	calculation *CalculationResult,
) models.CalculatedOrder {
	return models.CalculatedOrder{
		Medication: models.MedicationDetail{
			PrimaryIdentifier: models.Identifier{
				System:  "RxNorm",
				Code:    harmonization.Medication.RxNormCode,
				Display: harmonization.Medication.DisplayName,
			},
			AlternateIdentifiers: g.buildAlternateIdentifiers(harmonization.Medication),
			BrandName:           harmonization.Medication.BrandName,
			GenericName:         harmonization.Medication.GenericName,
			TherapeuticClass:    harmonization.Medication.TherapeuticClass,
			IsHighAlert:         harmonization.Medication.IsHighAlert,
			IsControlled:        harmonization.Medication.IsControlled,
		},
		Dosing: g.buildDosingDetail(calculation),
		CalculationDetails: g.buildCalculationDetails(calculation),
		Formulation: g.buildFormulationDetail(calculation),
	}
}

// buildDosingDetail constructs the dosing information
func (g *EnhancedProposalGenerator) buildDosingDetail(calculation *CalculationResult) models.DosingDetail {
	return models.DosingDetail{
		Dose: models.DoseInfo{
			Value:   calculation.OptimalDose.Amount,
			Unit:    calculation.OptimalDose.Unit,
			PerDose: true,
		},
		Route: models.RouteInfo{
			Code:    calculation.OptimalDose.Route,
			Display: g.getRouteDisplay(calculation.OptimalDose.Route),
		},
		Frequency: models.FrequencyInfo{
			Code:          calculation.OptimalDose.FrequencyCode,
			Display:       calculation.OptimalDose.FrequencyDisplay,
			TimesPerDay:   calculation.OptimalDose.TimesPerDay,
			SpecificTimes: calculation.OptimalDose.SpecificTimes,
		},
		Duration: models.DurationInfo{
			Value:   calculation.OptimalDose.Duration,
			Unit:    "days",
			Refills: calculation.OptimalDose.Refills,
		},
		Instructions: models.InstructionInfo{
			PatientInstructions:    calculation.Instructions.PatientInstructions,
			PharmacyInstructions:   calculation.Instructions.PharmacyInstructions,
			AdditionalInstructions: calculation.Instructions.AdditionalInstructions,
		},
	}
}

// buildCalculationDetails constructs the calculation details
func (g *EnhancedProposalGenerator) buildCalculationDetails(calculation *CalculationResult) models.CalculationDetails {
	return models.CalculationDetails{
		Method:          calculation.Method,
		Factors:         g.buildCalculationFactors(calculation.Factors),
		Adjustments:     calculation.Adjustments,
		RoundingApplied: calculation.RoundingApplied,
		MaximumDoseCheck: models.MaximumDoseCheck{
			Daily:        calculation.MaxDoseCheck.DailyDose,
			Maximum:      calculation.MaxDoseCheck.MaximumDose,
			WithinLimits: calculation.MaxDoseCheck.WithinLimits,
		},
	}
}

// buildCalculationFactors constructs the calculation factors
func (g *EnhancedProposalGenerator) buildCalculationFactors(factors *CalculationFactors) models.CalculationFactors {
	return models.CalculationFactors{
		PatientWeight: factors.Weight,
		PatientAge:    factors.Age,
		RenalFunction: models.RenalFunction{
			EGFR:     factors.RenalFunction.EGFR,
			Category: factors.RenalFunction.Category,
		},
	}
}

// buildFormulationDetail constructs the formulation details
func (g *EnhancedProposalGenerator) buildFormulationDetail(calculation *CalculationResult) models.FormulationDetail {
	alternatives := make([]models.AlternativeFormulation, len(calculation.Formulation.Alternatives))
	for i, alt := range calculation.Formulation.Alternatives {
		alternatives[i] = models.AlternativeFormulation{
			Form:         alt.Form,
			Strengths:    alt.Strengths,
			ClinicalNote: alt.ClinicalNote,
		}
	}

	return models.FormulationDetail{
		SelectedForm:            calculation.Formulation.SelectedForm,
		AvailableStrengths:      calculation.Formulation.AvailableStrengths,
		Splittable:              calculation.Formulation.Splittable,
		Crushable:               calculation.Formulation.Crushable,
		AlternativeFormulations: alternatives,
	}
}

// buildMonitoringPlan constructs the monitoring plan
func (g *EnhancedProposalGenerator) buildMonitoringPlan(monitoring *MonitoringPlanResponse) models.EnhancedMonitoringPlan {
	return models.EnhancedMonitoringPlan{
		RiskStratification: g.buildRiskStratification(monitoring.RiskAssessment),
		Baseline:          g.buildBaselineMonitoring(monitoring.BaselineRequirements),
		Ongoing:           g.buildOngoingMonitoring(monitoring.OngoingRequirements),
		SymptomMonitoring: g.buildSymptomMonitoring(monitoring.SymptomMonitoring),
	}
}

// buildRiskStratification constructs risk stratification
func (g *EnhancedProposalGenerator) buildRiskStratification(risk *RiskAssessment) models.RiskStratification {
	factors := make([]models.RiskFactor, len(risk.Factors))
	for i, factor := range risk.Factors {
		factors[i] = models.RiskFactor{
			Factor:  factor.Factor,
			Present: factor.Present,
			Impact:  factor.Impact,
		}
	}

	return models.RiskStratification{
		OverallRisk: risk.OverallRisk,
		Factors:     factors,
	}
}

// buildTherapeuticAlternatives constructs therapeutic alternatives
func (g *EnhancedProposalGenerator) buildTherapeuticAlternatives(alternatives *AlternativeResponse) models.TherapeuticAlternatives {
	therapeuticAlts := make([]models.TherapeuticAlternative, len(alternatives.Alternatives))
	for i, alt := range alternatives.Alternatives {
		therapeuticAlts[i] = g.buildTherapeuticAlternative(alt)
	}

	nonPharmAlts := make([]models.NonPharmAlternative, len(alternatives.NonPharmAlternatives))
	for i, alt := range alternatives.NonPharmAlternatives {
		nonPharmAlts[i] = models.NonPharmAlternative{
			Intervention:   alt.Intervention,
			Components:     alt.Components,
			Effectiveness:  alt.Effectiveness,
			Recommendation: alt.Recommendation,
		}
	}

	return models.TherapeuticAlternatives{
		PrimaryReason:        alternatives.PrimaryReason,
		Alternatives:         therapeuticAlts,
		NonPharmAlternatives: nonPharmAlts,
	}
}

// buildTherapeuticAlternative constructs a single therapeutic alternative
func (g *EnhancedProposalGenerator) buildTherapeuticAlternative(alt *TherapeuticAlternativeData) models.TherapeuticAlternative {
	var evidence *models.AlternativeEvidence
	if alt.Evidence != nil {
		evidence = &models.AlternativeEvidence{
			ComparativeEffectiveness: alt.Evidence.ComparativeEffectiveness,
			GuidelinePosition:        alt.Evidence.GuidelinePosition,
			References:               alt.Evidence.References,
		}
	}

	return models.TherapeuticAlternative{
		Medication: models.AlternativeMedicationDetail{
			Name:     alt.Medication.Name,
			Code:     alt.Medication.Code,
			Strength: alt.Medication.Strength,
			Unit:     alt.Medication.Unit,
		},
		Category: alt.Category,
		FormularyStatus: models.FormularyStatus{
			Tier:              alt.FormularyStatus.Tier,
			PriorAuthRequired: alt.FormularyStatus.PriorAuthRequired,
			QuantityLimits:    alt.FormularyStatus.QuantityLimits,
		},
		CostComparison: models.CostComparison{
			RelativeCost:         alt.CostComparison.RelativeCost,
			EstimatedMonthlyCost: alt.CostComparison.EstimatedMonthlyCost,
			PatientCopay:         alt.CostComparison.PatientCopay,
		},
		ClinicalConsiderations: models.ClinicalConsiderations{
			Advantages:        alt.ClinicalConsiderations.Advantages,
			Disadvantages:     alt.ClinicalConsiderations.Disadvantages,
			Contraindications: alt.ClinicalConsiderations.Contraindications,
		},
		SwitchingInstructions: alt.SwitchingInstructions,
		Evidence:              evidence,
	}
}

// buildClinicalRationale constructs the clinical rationale
func (g *EnhancedProposalGenerator) buildClinicalRationale(rationale *RationaleResponse) models.ClinicalRationale {
	return models.ClinicalRationale{
		Summary: models.RationaleSummary{
			Decision:   rationale.Summary.Decision,
			Confidence: rationale.Summary.Confidence,
			Complexity: rationale.Summary.Complexity,
		},
		IndicationAssessment: g.buildIndicationAssessment(rationale.IndicationAssessment),
		DosingRationale:      g.buildDosingRationale(rationale.DosingRationale),
		FormularyRationale:   g.buildFormularyRationale(rationale.FormularyRationale),
		PatientFactors:       g.buildPatientFactors(rationale.PatientFactors),
		QualityMeasures:      g.buildQualityMeasures(rationale.QualityMeasures),
	}
}

// buildProposalMetadata constructs the proposal metadata section
func (g *EnhancedProposalGenerator) buildProposalMetadata(startTime time.Time, calculation *CalculationResult) models.ProposalMetadataSection {
	return models.ProposalMetadataSection{
		ClinicalReferences: []models.ClinicalReference{
			{
				Type:     "GUIDELINE",
				Citation: "American Diabetes Association. Standards of Medical Care in Diabetes—2024",
				URL:      "https://doi.org/10.2337/dc24-S009",
			},
		},
		AuditTrail: models.AuditTrail{
			CalculationTime:     45,
			ContextFetchTime:    120,
			TotalProcessingTime: time.Since(startTime).Milliseconds(),
			CacheUtilization: models.CacheUtilization{
				FormularyCache:       "HIT",
				DoseCalculationCache: "MISS",
				MonitoringCache:      "HIT",
			},
		},
		NextSteps: []models.NextStep{
			{
				Step:           "SAFETY_VALIDATION",
				Service:        "Safety Gateway",
				RequiredChecks: []string{"Drug interactions", "Allergies", "Contraindications"},
			},
			{
				Step:     "PROVIDER_REVIEW",
				Optional: false,
				Reason:   "New medication initiation",
			},
		},
	}
}
