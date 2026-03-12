// Package scoring provides an enhanced scoring engine that combines the best features
// of both the Compare-and-Rank model and the comprehensive scoring engine
package scoring

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"flow2-go-engine/internal/models"

	"github.com/sirupsen/logrus"
	"go.uber.org/zap"
)

// EnhancedScoringEngine combines comprehensive scoring with compare-and-rank intelligence
type EnhancedScoringEngine struct {
	// Core engines
	compareAndRankEngine CompareAndRankEngine
	
	// External services
	efficacyService  EfficacyDataService
	costService      CostDataService
	historyService   PatientHistoryService
	
	// Configuration
	config *ScoringConfig
	logger *logrus.Logger
	zapLogger *zap.Logger
	
	// Caching
	efficacyCache map[string]*EfficacyData
	cacheMu       sync.RWMutex
}

// ScoringConfig defines comprehensive scoring configuration
type ScoringConfig struct {
	Weights    ScoringWeights    `json:"weights"`
	Parameters ScoringParameters `json:"parameters"`
	Features   ScoringFeatures   `json:"features"`
}

// ScoringWeights defines the relative importance of each scoring dimension
type ScoringWeights struct {
	Safety                 float64 `json:"safety"`
	Efficacy               float64 `json:"efficacy"`
	Tolerability           float64 `json:"tolerability"`
	Convenience            float64 `json:"convenience"`
	Cost                   float64 `json:"cost"`
	PatientPreference      float64 `json:"patient_preference"`
	GuidelineAdherence     float64 `json:"guideline_adherence"`
	DrugInteractionProfile float64 `json:"drug_interaction_profile"`
}

// ScoringParameters defines thresholds and scaling factors
type ScoringParameters struct {
	MaxAcceptableCostPerMonth float64 `json:"max_acceptable_cost_per_month"`
	CostSensitivity           float64 `json:"cost_sensitivity"`
	MinimumEfficacyThreshold  float64 `json:"minimum_efficacy_threshold"`
	EfficacyDataQualityWeight float64 `json:"efficacy_data_quality_weight"`
	IdealDosesPerDay          int     `json:"ideal_doses_per_day"`
	PillBurdenPenaltyFactor   float64 `json:"pill_burden_penalty_factor"`
	PreferGeneric             bool    `json:"prefer_generic"`
	PreferOralRoute           bool    `json:"prefer_oral_route"`
	AvoidInjectables          bool    `json:"avoid_injectables"`
	GuidelineSource           string  `json:"guideline_source"`
	PreferFirstLine           bool    `json:"prefer_first_line"`
}

// ScoringFeatures defines which advanced features are enabled
type ScoringFeatures struct {
	UseMLPredictions       bool `json:"use_ml_predictions"`
	UsePatientHistory      bool `json:"use_patient_history"`
	UsePopulationOutcomes  bool `json:"use_population_outcomes"`
	UseRealWorldEvidence   bool `json:"use_real_world_evidence"`
	UsePredictiveAdherence bool `json:"use_predictive_adherence"`
	UsePharmacogenomics    bool `json:"use_pharmacogenomics"`
	UseCompareAndRank      bool `json:"use_compare_and_rank"`
}

// External service interfaces
type EfficacyDataService interface {
	GetEfficacyData(ctx context.Context, drugID string, indication string) (*EfficacyData, error)
	GetComparativeEffectiveness(ctx context.Context, drugIDs []string, indication string) (map[string]float64, error)
}

type CostDataService interface {
	GetMedicationCost(ctx context.Context, drugID string, formularyID string) (*CostData, error)
	GetPatientCopay(ctx context.Context, drugID string, insuranceID string) (float64, error)
}

type PatientHistoryService interface {
	GetMedicationHistory(ctx context.Context, patientID string) ([]MedicationHistoryEntry, error)
	GetAdherenceHistory(ctx context.Context, patientID string) (*AdherenceProfile, error)
	GetOutcomeHistory(ctx context.Context, patientID string) ([]OutcomeEntry, error)
}

// Data structures
type EfficacyData struct {
	DrugID              string                 `json:"drug_id"`
	Indication          string                 `json:"indication"`
	EfficacyScore       float64                `json:"efficacy_score"`
	ClinicalTrials      []ClinicalTrialSummary `json:"clinical_trials"`
	MetaAnalyses        []MetaAnalysis         `json:"meta_analyses"`
	NumberNeededToTreat *int                   `json:"nnt,omitempty"`
	TimeToEffect        string                 `json:"time_to_effect"`
	DurableResponse     bool                   `json:"durable_response"`
}

type CostData struct {
	DrugID                 string  `json:"drug_id"`
	AWPPerMonth            float64 `json:"awp_per_month"`
	PatientCopayPerMonth   float64 `json:"patient_copay_per_month"`
	FormularyTier          int     `json:"formulary_tier"`
	GenericAvailable       bool    `json:"generic_available"`
	PatientAssistance      bool    `json:"patient_assistance_available"`
	CostEffectivenessRatio float64 `json:"cost_effectiveness_ratio"`
}

type ClinicalTrialSummary struct {
	TrialID         string  `json:"trial_id"`
	Name            string  `json:"name"`
	SampleSize      int     `json:"sample_size"`
	PrimaryOutcome  string  `json:"primary_outcome"`
	EffectSize      float64 `json:"effect_size"`
	PValue          float64 `json:"p_value"`
	NumberNeededToTreat int `json:"nnt"`
}

type MetaAnalysis struct {
	Title            string  `json:"title"`
	StudiesIncluded  int     `json:"studies_included"`
	TotalPatients    int     `json:"total_patients"`
	PooledEffectSize float64 `json:"pooled_effect_size"`
	Heterogeneity    string  `json:"heterogeneity"`
	Publication      string  `json:"publication"`
	Year             int     `json:"year"`
}

type MedicationHistoryEntry struct {
	DrugID            string     `json:"drug_id"`
	StartDate         time.Time  `json:"start_date"`
	EndDate           *time.Time `json:"end_date,omitempty"`
	ReasonForStopping *string    `json:"reason_for_stopping,omitempty"`
	Efficacy          *string    `json:"efficacy,omitempty"`
	Tolerability      *string    `json:"tolerability,omitempty"`
}

type AdherenceProfile struct {
	OverallAdherence    float64            `json:"overall_adherence"`
	MedicationAdherence map[string]float64 `json:"medication_adherence"`
	PredictedAdherence  float64            `json:"predicted_adherence"`
	AdherenceFactors    []string           `json:"adherence_factors"`
}

type OutcomeEntry struct {
	Date         time.Time `json:"date"`
	OutcomeType  string    `json:"outcome_type"`
	Value        string    `json:"value"`
	MedicationID string    `json:"medication_id"`
}

// NewEnhancedScoringEngine creates a new enhanced scoring engine
func NewEnhancedScoringEngine(
	compareAndRankEngine CompareAndRankEngine,
	efficacyService EfficacyDataService,
	costService CostDataService,
	historyService PatientHistoryService,
	config *ScoringConfig,
	logger *logrus.Logger,
) *EnhancedScoringEngine {
	if config == nil {
		config = DefaultScoringConfig()
	}
	if logger == nil {
		logger = logrus.New()
	}

	// Create zap logger for compatibility
	zapLogger, _ := zap.NewProduction()

	return &EnhancedScoringEngine{
		compareAndRankEngine: compareAndRankEngine,
		efficacyService:      efficacyService,
		costService:          costService,
		historyService:       historyService,
		config:               config,
		logger:               logger,
		zapLogger:            zapLogger,
		efficacyCache:        make(map[string]*EfficacyData),
	}
}

// DefaultScoringConfig returns a balanced default configuration
func DefaultScoringConfig() *ScoringConfig {
	return &ScoringConfig{
		Weights: ScoringWeights{
			Safety:                 0.30,
			Efficacy:               0.25,
			Tolerability:           0.15,
			Convenience:            0.10,
			Cost:                   0.10,
			PatientPreference:      0.05,
			GuidelineAdherence:     0.03,
			DrugInteractionProfile: 0.02,
		},
		Parameters: ScoringParameters{
			MaxAcceptableCostPerMonth: 500.0,
			CostSensitivity:           0.5,
			MinimumEfficacyThreshold:  0.6,
			EfficacyDataQualityWeight: 0.8,
			IdealDosesPerDay:          1,
			PillBurdenPenaltyFactor:   0.1,
			PreferGeneric:             true,
			PreferOralRoute:           true,
			AvoidInjectables:          false,
			GuidelineSource:           "AHA",
			PreferFirstLine:           true,
		},
		Features: ScoringFeatures{
			UseMLPredictions:       false,
			UsePatientHistory:      true,
			UsePopulationOutcomes:  true,
			UseRealWorldEvidence:   false,
			UsePredictiveAdherence: false,
			UsePharmacogenomics:    true,
			UseCompareAndRank:      true, // Enable enhanced compare-and-rank
		},
	}
}

// ScoreAndRankProposals performs comprehensive scoring with optional compare-and-rank enhancement
func (e *EnhancedScoringEngine) ScoreAndRankProposals(
	ctx context.Context,
	proposals []*models.SafetyVerifiedProposal,
	patientContext *models.ClinicalContext,
	indication string,
) ([]*models.EnhancedScoredProposal, error) {
	if len(proposals) == 0 {
		return []*models.EnhancedScoredProposal{}, nil
	}

	startTime := time.Now()
	e.logger.WithFields(logrus.Fields{
		"proposal_count": len(proposals),
		"indication":     indication,
		"patient_id":     patientContext.PatientID,
	}).Info("Starting enhanced scoring process")

	// If compare-and-rank is enabled and available, use it
	if e.config.Features.UseCompareAndRank && e.compareAndRankEngine != nil {
		return e.scoreWithCompareAndRank(ctx, proposals, patientContext, indication)
	}

	// Otherwise, use traditional comprehensive scoring
	return e.scoreWithTraditionalMethod(ctx, proposals, patientContext, indication)
}

// scoreWithCompareAndRank uses the enhanced compare-and-rank engine
func (e *EnhancedScoringEngine) scoreWithCompareAndRank(
	ctx context.Context,
	proposals []*models.SafetyVerifiedProposal,
	patientContext *models.ClinicalContext,
	indication string,
) ([]*models.EnhancedScoredProposal, error) {
	// Convert to enhanced proposals with enriched data
	enhancedProposals, err := e.enrichProposalsWithExternalData(ctx, proposals, patientContext, indication)
	if err != nil {
		e.logger.WithError(err).Warn("Failed to enrich proposals, falling back to basic data")
		enhancedProposals = e.convertToBasicEnhancedProposals(proposals, patientContext)
	}

	// Extract patient risk context
	patientRiskContext := e.extractPatientRiskContext(patientContext)

	// Create compare-and-rank request
	request := &models.CompareAndRankRequest{
		PatientContext: patientRiskContext,
		Candidates:     enhancedProposals,
		ConfigRef: models.ConfigReference{
			WeightProfile:    patientRiskContext.RiskPhenotype,
			PenaltiesProfile: "default",
		},
		RequestID: fmt.Sprintf("enhanced-scoring-%d", time.Now().Unix()),
		Timestamp: time.Now(),
	}

	// Execute compare-and-rank
	response, err := e.compareAndRankEngine.CompareAndRank(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("compare-and-rank failed: %w", err)
	}

	e.logger.WithFields(logrus.Fields{
		"candidates_ranked":  len(response.Ranked),
		"candidates_pruned":  response.Audit.CandidatesPruned,
		"weight_profile":     response.Audit.ProfileUsed.Weights,
		"processing_time_ms": response.Audit.ProcessingTime.Milliseconds(),
	}).Info("Enhanced scoring with compare-and-rank completed")

	return response.Ranked, nil
}

// enrichProposalsWithExternalData enriches proposals with external service data
func (e *EnhancedScoringEngine) enrichProposalsWithExternalData(
	ctx context.Context,
	proposals []*models.SafetyVerifiedProposal,
	patientContext *models.ClinicalContext,
	indication string,
) ([]models.EnhancedProposal, error) {
	var enhanced []models.EnhancedProposal

	for _, proposal := range proposals {
		// Get efficacy data
		efficacyData, err := e.getEfficacyData(ctx, proposal.Original.MedicationCode, indication)
		if err != nil {
			e.logger.WithError(err).WithField("drug_id", proposal.Original.MedicationCode).
				Warn("Failed to get efficacy data")
		}

		// Get cost data
		costData, err := e.getCostData(ctx, proposal.Original.MedicationCode, patientContext.FormularyKBId)
		if err != nil {
			e.logger.WithError(err).WithField("drug_id", proposal.Original.MedicationCode).
				Warn("Failed to get cost data")
		}

		// Create enhanced proposal with enriched data
		enhancedProposal := models.EnhancedProposal{
			TherapyID: proposal.Original.MedicationCode,
			Class:     proposal.Original.TherapeuticClass,
			Agent:     proposal.Original.GenericName,
			Regimen: models.RegimenDetail{
				Form:      e.inferFormulation(proposal.Original.MedicationName),
				Frequency: e.inferFrequency(proposal.FinalDose.IntervalH),
				IsFDC:     e.isFDC(proposal.Original.MedicationName),
				PillCount: e.inferPillCount(proposal.Original.MedicationName),
			},
			Dose: models.DoseDetail{
				Amount:    proposal.FinalDose.DoseMg,
				Unit:      "mg",
				Frequency: e.inferFrequency(proposal.FinalDose.IntervalH),
				Route:     proposal.FinalDose.Route,
				Rationale: "JIT safety verified dose with external data enrichment",
			},
			Efficacy: models.EfficacyDetail{
				ExpectedA1cDropPct: e.extractEfficacyScore(efficacyData),
				CVBenefit:         e.hasCVBenefit(proposal.Original.MedicationName, efficacyData),
				HFBenefit:         e.hasHFBenefit(proposal.Original.MedicationName, efficacyData),
				CKDBenefit:        e.hasCKDBenefit(proposal.Original.MedicationName, efficacyData),
			},
			Safety: models.SafetyDetail{
				ResidualDDI:    e.mapDDISeverity(proposal.DDIWarnings),
				HypoPropensity: e.mapHypoglycemiaRisk(proposal.Original.MedicationName),
				WeightEffect:   e.mapWeightEffect(proposal.Original.MedicationName),
			},
			Suitability: models.SuitabilityDetail{
				RenalFit:   e.assessRenalFit(proposal, patientContext),
				HepaticFit: e.assessHepaticFit(proposal, patientContext),
			},
			Adherence: models.AdherenceDetail{
				DosesPerDay:      e.calculateDosesPerDay(proposal.FinalDose.IntervalH),
				PillBurden:       e.inferPillCount(proposal.Original.MedicationName),
				RequiresDevice:   proposal.FinalDose.Route != "po",
				RequiresTraining: e.requiresTraining(proposal.FinalDose.Route),
			},
			Availability: models.AvailabilityDetail{
				Tier:         e.extractFormularyTier(costData),
				OnHand:       100, // Default - would come from inventory system
				LeadTimeDays: 0,   // Default
			},
			Cost: models.CostDetail{
				MonthlyEstimate: e.extractMonthlyCost(costData),
				Currency:        "USD",
				PatientCopay:    e.extractPatientCopay(costData),
			},
			Preferences: models.PreferencesDetail{
				AvoidInjectables:   e.config.Parameters.AvoidInjectables,
				OnceDailyPreferred: true,
				CostSensitivity:    e.inferCostSensitivity(e.config.Parameters.CostSensitivity),
			},
			Provenance: models.ProvenanceDetail{
				KBVersions: map[string]string{
					"jit_safety":     "v1.0",
					"efficacy_data":  "v2.1",
					"cost_data":      "v1.5",
					"enhanced_scoring": "v1.0",
				},
			},
		}

		enhanced = append(enhanced, enhancedProposal)
	}

	return enhanced, nil
}

// convertToBasicEnhancedProposals creates basic enhanced proposals without external data
func (e *EnhancedScoringEngine) convertToBasicEnhancedProposals(
	proposals []*models.SafetyVerifiedProposal,
	patientContext *models.ClinicalContext,
) []models.EnhancedProposal {
	var enhanced []models.EnhancedProposal

	for _, proposal := range proposals {
		enhancedProposal := models.EnhancedProposal{
			TherapyID: proposal.Original.MedicationCode,
			Class:     proposal.Original.TherapeuticClass,
			Agent:     proposal.Original.GenericName,
			Regimen: models.RegimenDetail{
				Form:      "tablet",
				Frequency: "daily",
				IsFDC:     false,
				PillCount: 1,
			},
			Dose: models.DoseDetail{
				Amount:    proposal.FinalDose.DoseMg,
				Unit:      "mg",
				Frequency: "daily",
				Route:     proposal.FinalDose.Route,
				Rationale: "Basic enhanced proposal",
			},
			Efficacy: models.EfficacyDetail{
				ExpectedA1cDropPct: e.estimateBasicEfficacy(proposal.Original.MedicationName),
				CVBenefit:         e.hasBasicCVBenefit(proposal.Original.MedicationName),
				HFBenefit:         e.hasBasicHFBenefit(proposal.Original.MedicationName),
				CKDBenefit:        e.hasBasicCKDBenefit(proposal.Original.MedicationName),
			},
			Safety: models.SafetyDetail{
				ResidualDDI:    e.mapDDISeverity(proposal.DDIWarnings),
				HypoPropensity: e.mapHypoglycemiaRisk(proposal.Original.MedicationName),
				WeightEffect:   e.mapWeightEffect(proposal.Original.MedicationName),
			},
			Suitability: models.SuitabilityDetail{
				RenalFit:   true,
				HepaticFit: true,
			},
			Adherence: models.AdherenceDetail{
				DosesPerDay:      e.calculateDosesPerDay(proposal.FinalDose.IntervalH),
				PillBurden:       1,
				RequiresDevice:   proposal.FinalDose.Route != "po",
				RequiresTraining: e.requiresTraining(proposal.FinalDose.Route),
			},
			Availability: models.AvailabilityDetail{
				Tier:         proposal.Original.FormularyTier,
				OnHand:       100,
				LeadTimeDays: 0,
			},
			Cost: models.CostDetail{
				MonthlyEstimate: proposal.Original.CostEstimate,
				Currency:        "USD",
			},
			Preferences: models.PreferencesDetail{
				AvoidInjectables:   e.config.Parameters.AvoidInjectables,
				OnceDailyPreferred: true,
				CostSensitivity:    "medium",
			},
			Provenance: models.ProvenanceDetail{
				KBVersions: map[string]string{
					"jit_safety": "v1.0",
					"basic_data": "v1.0",
				},
			},
		}

		enhanced = append(enhanced, enhancedProposal)
	}

	return enhanced
}

// extractPatientRiskContext extracts patient risk context from clinical context
func (e *EnhancedScoringEngine) extractPatientRiskContext(clinicalContext *models.ClinicalContext) models.PatientRiskContext {
	riskPhenotype := "NONE"

	// Determine risk phenotype based on conditions
	for _, condition := range clinicalContext.Conditions {
		switch condition.Code {
		case "I25.9", "I21.9": // CAD, MI
			riskPhenotype = "ASCVD"
		case "I50.9": // Heart failure
			riskPhenotype = "HF"
		case "N18.6": // CKD
			riskPhenotype = "CKD"
		}
	}

	return models.PatientRiskContext{
		RiskPhenotype: riskPhenotype,
		ResourceTier:  "standard",
		Preferences: models.JITPatientPreferences{
			AvoidInjectables:   e.config.Parameters.AvoidInjectables,
			OnceDailyPreferred: true,
			CostSensitivity:    e.inferCostSensitivity(e.config.Parameters.CostSensitivity),
		},
	}
}

// Helper methods for data extraction and inference

// getEfficacyData retrieves efficacy data from external service
func (e *EnhancedScoringEngine) getEfficacyData(ctx context.Context, drugID, indication string) (*EfficacyData, error) {
	if e.efficacyService == nil {
		return nil, fmt.Errorf("efficacy service not available")
	}

	// Check cache first
	e.cacheMu.RLock()
	cached, exists := e.efficacyCache[drugID]
	e.cacheMu.RUnlock()

	if exists {
		return cached, nil
	}

	// Fetch from service
	data, err := e.efficacyService.GetEfficacyData(ctx, drugID, indication)
	if err != nil {
		return nil, err
	}

	// Cache the result
	e.cacheMu.Lock()
	e.efficacyCache[drugID] = data
	e.cacheMu.Unlock()

	return data, nil
}

// getCostData retrieves cost data from external service
func (e *EnhancedScoringEngine) getCostData(ctx context.Context, drugID, formularyID string) (*CostData, error) {
	if e.costService == nil {
		return nil, fmt.Errorf("cost service not available")
	}

	return e.costService.GetMedicationCost(ctx, drugID, formularyID)
}

// Data extraction methods
func (e *EnhancedScoringEngine) extractEfficacyScore(data *EfficacyData) float64 {
	if data == nil {
		return 1.0 // Default efficacy
	}
	return data.EfficacyScore
}

func (e *EnhancedScoringEngine) extractMonthlyCost(data *CostData) float64 {
	if data == nil {
		return 100.0 // Default cost
	}
	return data.PatientCopayPerMonth
}

func (e *EnhancedScoringEngine) extractPatientCopay(data *CostData) float64 {
	if data == nil {
		return 25.0 // Default copay
	}
	return data.PatientCopayPerMonth
}

func (e *EnhancedScoringEngine) extractFormularyTier(data *CostData) int {
	if data == nil {
		return 2 // Default tier
	}
	return data.FormularyTier
}

// Inference methods
func (e *EnhancedScoringEngine) inferFormulation(medicationName string) string {
	// Simple inference based on medication name
	if contains(medicationName, "injection") || contains(medicationName, "injectable") {
		return "injection"
	}
	if contains(medicationName, "patch") {
		return "patch"
	}
	return "tablet"
}

func (e *EnhancedScoringEngine) inferFrequency(intervalH uint32) string {
	if intervalH == 0 {
		return "daily"
	}
	switch intervalH {
	case 24:
		return "daily"
	case 12:
		return "twice daily"
	case 8:
		return "three times daily"
	case 6:
		return "four times daily"
	case 168: // Weekly
		return "weekly"
	default:
		return "daily"
	}
}

func (e *EnhancedScoringEngine) isFDC(medicationName string) bool {
	// Check for common FDC indicators
	return contains(medicationName, "/") ||
		   contains(medicationName, "combination") ||
		   contains(medicationName, "plus")
}

func (e *EnhancedScoringEngine) inferPillCount(medicationName string) int {
	if e.isFDC(medicationName) {
		return 1 // FDC is typically one pill
	}
	return 1 // Default
}

func (e *EnhancedScoringEngine) calculateDosesPerDay(intervalH uint32) int {
	if intervalH == 0 {
		return 1
	}
	return int(24 / intervalH)
}

func (e *EnhancedScoringEngine) requiresTraining(route string) bool {
	return route == "sc" || route == "im" || route == "iv"
}

func (e *EnhancedScoringEngine) inferCostSensitivity(sensitivity float64) string {
	if sensitivity < 0.3 {
		return "low"
	} else if sensitivity < 0.7 {
		return "medium"
	}
	return "high"
}

// Clinical assessment methods
func (e *EnhancedScoringEngine) hasCVBenefit(medicationName string, data *EfficacyData) bool {
	// Check external data first
	if data != nil {
		for _, trial := range data.ClinicalTrials {
			if contains(trial.PrimaryOutcome, "cardiovascular") || contains(trial.PrimaryOutcome, "CV") {
				return true
			}
		}
	}

	// Fallback to basic inference
	return e.hasBasicCVBenefit(medicationName)
}

func (e *EnhancedScoringEngine) hasHFBenefit(medicationName string, data *EfficacyData) bool {
	// Check external data first
	if data != nil {
		for _, trial := range data.ClinicalTrials {
			if contains(trial.PrimaryOutcome, "heart failure") || contains(trial.PrimaryOutcome, "HF") {
				return true
			}
		}
	}

	// Fallback to basic inference
	return e.hasBasicHFBenefit(medicationName)
}

func (e *EnhancedScoringEngine) hasCKDBenefit(medicationName string, data *EfficacyData) bool {
	// Check external data first
	if data != nil {
		for _, trial := range data.ClinicalTrials {
			if contains(trial.PrimaryOutcome, "kidney") || contains(trial.PrimaryOutcome, "renal") {
				return true
			}
		}
	}

	// Fallback to basic inference
	return e.hasBasicCKDBenefit(medicationName)
}

func (e *EnhancedScoringEngine) assessRenalFit(proposal *models.SafetyVerifiedProposal, context *models.ClinicalContext) bool {
	// Check for renal contraindications in safety warnings
	for _, warning := range proposal.Warnings {
		if contains(warning, "renal") || contains(warning, "kidney") {
			return false
		}
	}
	return true
}

func (e *EnhancedScoringEngine) assessHepaticFit(proposal *models.SafetyVerifiedProposal, context *models.ClinicalContext) bool {
	// Check for hepatic contraindications in safety warnings
	for _, warning := range proposal.Warnings {
		if contains(warning, "hepatic") || contains(warning, "liver") {
			return false
		}
	}
	return true
}

// Basic inference methods (fallback when external data unavailable)
func (e *EnhancedScoringEngine) estimateBasicEfficacy(medicationName string) float64 {
	switch {
	case contains(medicationName, "metformin"):
		return 1.0
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		return 1.2
	case contains(medicationName, "insulin"):
		return 1.8
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		return 1.5
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		return 0.8
	default:
		return 1.0
	}
}

func (e *EnhancedScoringEngine) hasBasicCVBenefit(medicationName string) bool {
	cvBenefitMeds := []string{"semaglutide", "liraglutide", "empagliflozin", "canagliflozin"}
	for _, med := range cvBenefitMeds {
		if contains(medicationName, med) {
			return true
		}
	}
	return false
}

func (e *EnhancedScoringEngine) hasBasicHFBenefit(medicationName string) bool {
	hfBenefitMeds := []string{"empagliflozin", "dapagliflozin"}
	for _, med := range hfBenefitMeds {
		if contains(medicationName, med) {
			return true
		}
	}
	return false
}

func (e *EnhancedScoringEngine) hasBasicCKDBenefit(medicationName string) bool {
	ckdBenefitMeds := []string{"empagliflozin", "canagliflozin"}
	for _, med := range ckdBenefitMeds {
		if contains(medicationName, med) {
			return true
		}
	}
	return false
}

func (e *EnhancedScoringEngine) mapDDISeverity(ddiWarnings []models.DDIFlag) string {
	if len(ddiWarnings) == 0 {
		return "none"
	}

	// Check for major DDIs first
	for _, ddi := range ddiWarnings {
		if ddi.Severity == "major" {
			return "major"
		}
	}

	// Check for moderate DDIs
	for _, ddi := range ddiWarnings {
		if ddi.Severity == "moderate" {
			return "moderate"
		}
	}

	return "none"
}

func (e *EnhancedScoringEngine) mapHypoglycemiaRisk(medicationName string) string {
	switch {
	case contains(medicationName, "insulin"):
		return "high"
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		return "high"
	case contains(medicationName, "metformin"):
		return "low"
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		return "low"
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		return "low"
	default:
		return "med"
	}
}

func (e *EnhancedScoringEngine) mapWeightEffect(medicationName string) string {
	switch {
	case contains(medicationName, "insulin"):
		return "gain"
	case contains(medicationName, "glipizide"), contains(medicationName, "glyburide"):
		return "gain"
	case contains(medicationName, "metformin"):
		return "neutral"
	case contains(medicationName, "semaglutide"), contains(medicationName, "liraglutide"):
		return "loss"
	case contains(medicationName, "empagliflozin"), contains(medicationName, "dapagliflozin"):
		return "loss"
	default:
		return "neutral"
	}
}

// scoreWithTraditionalMethod provides fallback traditional scoring
func (e *EnhancedScoringEngine) scoreWithTraditionalMethod(
	ctx context.Context,
	proposals []*models.SafetyVerifiedProposal,
	patientContext *models.ClinicalContext,
	indication string,
) ([]*models.EnhancedScoredProposal, error) {
	// This would implement the traditional comprehensive scoring approach
	// For now, return a simple implementation
	var scored []*models.EnhancedScoredProposal

	for i, proposal := range proposals {
		// Create basic enhanced scored proposal
		enhancedScored := &models.EnhancedScoredProposal{
			TherapyID: proposal.Original.MedicationCode,
			FinalScore: 0.8 - float64(i)*0.1, // Simple decreasing score
			Rank: i + 1,
			SubScores: models.EnhancedComponentScores{
				Efficacy: models.EfficacyScoreDetail{
					Score: e.estimateBasicEfficacy(proposal.Original.MedicationName) / 2.0,
					ExpectedA1cDropPct: e.estimateBasicEfficacy(proposal.Original.MedicationName),
					CVBenefit: e.hasBasicCVBenefit(proposal.Original.MedicationName),
				},
				Safety: models.SafetyScoreDetail{
					Score: proposal.SafetyScore,
					ResidualDDI: e.mapDDISeverity(proposal.DDIWarnings),
					HypoPropensity: e.mapHypoglycemiaRisk(proposal.Original.MedicationName),
					WeightEffect: e.mapWeightEffect(proposal.Original.MedicationName),
				},
				Cost: models.CostScoreDetail{
					Score: math.Max(0.1, 1.0 - (proposal.Original.CostEstimate / 500.0)),
					MonthlyEstimate: proposal.Original.CostEstimate,
					Currency: "USD",
				},
				Adherence: models.AdherenceScoreDetail{
					Score: 0.8,
					DosesPerDay: e.calculateDosesPerDay(proposal.FinalDose.IntervalH),
				},
			},
			Notes: []string{
				fmt.Sprintf("Traditional scoring for %s", proposal.Original.MedicationName),
			},
			ScoredAt: time.Now(),
		}

		scored = append(scored, enhancedScored)
	}

	e.logger.WithFields(logrus.Fields{
		"candidates_scored": len(scored),
		"method": "traditional",
	}).Info("Traditional scoring completed")

	return scored, nil
}

// Utility functions
func contains(s, substr string) bool {
	return len(s) >= len(substr) &&
		   (s == substr ||
		    len(s) > len(substr) &&
		    (s[:len(substr)] == substr ||
		     s[len(s)-len(substr):] == substr ||
		     findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
