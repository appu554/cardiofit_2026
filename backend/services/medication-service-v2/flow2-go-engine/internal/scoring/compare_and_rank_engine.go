// Package scoring provides enhanced compare-and-rank functionality for medication recommendations
package scoring

import (
	"context"
	"fmt"
	"math"
	"sort"
	"time"

	"flow2-go-engine/internal/models"

	"github.com/sirupsen/logrus"
)

// CompareAndRankEngine interface for enhanced medication proposal ranking
type CompareAndRankEngine interface {
	CompareAndRank(ctx context.Context, request *models.CompareAndRankRequest) (*models.CompareAndRankResponse, error)
	UpdateWeightProfile(profile string, weights WeightProfile) error
	UpdatePenaltiesProfile(profile string, penalties PenaltiesProfile) error
	GetWeightProfile(profile string) (WeightProfile, error)
	GetPenaltiesProfile(profile string) (PenaltiesProfile, error)
}

// compareAndRankEngine implements CompareAndRankEngine
type compareAndRankEngine struct {
	weightProfiles    map[string]WeightProfile
	penaltiesProfiles map[string]PenaltiesProfile
	kbConfigLoader    *KBConfigLoader
	logger            *logrus.Logger
}

// WeightProfile represents phenotype-specific weight configurations
type WeightProfile struct {
	Efficacy    float64 `json:"efficacy" yaml:"efficacy"`
	Safety      float64 `json:"safety" yaml:"safety"`
	Availability float64 `json:"availability" yaml:"availability"`
	Cost        float64 `json:"cost" yaml:"cost"`
	Adherence   float64 `json:"adherence" yaml:"adherence"`
	Preference  float64 `json:"preference" yaml:"preference"`
}

// PenaltiesProfile represents configurable penalty values
type PenaltiesProfile struct {
	Safety      SafetyPenalties      `json:"safety" yaml:"safety"`
	Adherence   AdherencePenalties   `json:"adherence" yaml:"adherence"`
	Availability AvailabilityPenalties `json:"availability" yaml:"availability"`
}

// SafetyPenalties represents safety-related penalties
type SafetyPenalties struct {
	ResidualDDI map[string]float64 `json:"residual_ddi" yaml:"residual_ddi"`
	Hypo        map[string]float64 `json:"hypo" yaml:"hypo"`
	WeightGain  float64            `json:"weight_gain" yaml:"weight_gain"`
}

// AdherencePenalties represents adherence-related penalties and bonuses
type AdherencePenalties struct {
	Base                    float64            `json:"base" yaml:"base"`
	FrequencyBonus          map[string]float64 `json:"frequency_bonus" yaml:"frequency_bonus"`
	FDCBonus               float64            `json:"fdc_bonus" yaml:"fdc_bonus"`
	InjectablePenalty      float64            `json:"injectable_penalty" yaml:"injectable_penalty"`
	WeeklyInjectableBonus  float64            `json:"weekly_injectable_bonus" yaml:"weekly_injectable_bonus"`
	DeviceTrainingPenalty  float64            `json:"device_training_penalty" yaml:"device_training_penalty"`
}

// AvailabilityPenalties represents availability-related factors
type AvailabilityPenalties struct {
	TierFactor           map[int]float64 `json:"tier_factor" yaml:"tier_factor"`
	OutOfStockMultiplier float64         `json:"out_of_stock_multiplier" yaml:"out_of_stock_multiplier"`
}

// NewCompareAndRankEngine creates a new enhanced compare-and-rank engine
func NewCompareAndRankEngine(configPath string, logger *logrus.Logger) CompareAndRankEngine {
	if logger == nil {
		logger = logrus.New()
	}

	// Create KB config loader
	kbLoader := NewKBConfigLoader(configPath, logger)

	engine := &compareAndRankEngine{
		weightProfiles:    make(map[string]WeightProfile),
		penaltiesProfiles: make(map[string]PenaltiesProfile),
		kbConfigLoader:    kbLoader,
		logger:            logger,
	}

	// Load KB configuration
	if err := engine.loadKBConfiguration(); err != nil {
		logger.WithError(err).Warn("Failed to load KB configuration, using defaults")
		engine.initializeDefaultProfiles()
	}

	return engine
}

// NewCompareAndRankEngineWithDefaults creates engine with default configuration
func NewCompareAndRankEngineWithDefaults(logger *logrus.Logger) CompareAndRankEngine {
	if logger == nil {
		logger = logrus.New()
	}

	engine := &compareAndRankEngine{
		weightProfiles:    make(map[string]WeightProfile),
		penaltiesProfiles: make(map[string]PenaltiesProfile),
		kbConfigLoader:    nil, // No KB loader, use defaults only
		logger:            logger,
	}

	// Initialize default profiles
	engine.initializeDefaultProfiles()

	return engine
}

// initializeDefaultProfiles sets up the default phenotype-aware weight profiles
func (e *compareAndRankEngine) initializeDefaultProfiles() {
	// ASCVD profile - prioritizes efficacy and safety
	e.weightProfiles["ASCVD"] = WeightProfile{
		Efficacy:     0.38,
		Safety:       0.22,
		Availability: 0.10,
		Cost:         0.08,
		Adherence:    0.12,
		Preference:   0.10,
	}

	// HF profile - prioritizes safety and efficacy
	e.weightProfiles["HF"] = WeightProfile{
		Efficacy:     0.36,
		Safety:       0.24,
		Availability: 0.12,
		Cost:         0.06,
		Adherence:    0.12,
		Preference:   0.10,
	}

	// CKD profile - prioritizes safety and efficacy
	e.weightProfiles["CKD"] = WeightProfile{
		Efficacy:     0.36,
		Safety:       0.24,
		Availability: 0.12,
		Cost:         0.06,
		Adherence:    0.12,
		Preference:   0.10,
	}

	// NONE profile - balanced approach
	e.weightProfiles["NONE"] = WeightProfile{
		Efficacy:     0.30,
		Safety:       0.22,
		Availability: 0.14,
		Cost:         0.14,
		Adherence:    0.12,
		Preference:   0.08,
	}

	// BUDGET_MODE profile - prioritizes cost and availability
	e.weightProfiles["BUDGET_MODE"] = WeightProfile{
		Efficacy:     0.28,
		Safety:       0.22,
		Availability: 0.16,
		Cost:         0.22,
		Adherence:    0.08,
		Preference:   0.04,
	}

	// Initialize default penalties profile
	e.penaltiesProfiles["default"] = PenaltiesProfile{
		Safety: SafetyPenalties{
			ResidualDDI: map[string]float64{
				"none":     0.0,
				"moderate": 0.15,
				"major":    0.30,
			},
			Hypo: map[string]float64{
				"low":  0.00,
				"med":  0.15,
				"high": 0.25,
			},
			WeightGain: 0.05,
		},
		Adherence: AdherencePenalties{
			Base: 0.50,
			FrequencyBonus: map[string]float64{
				"od":  0.20,
				"bid": 0.05,
				"tid": -0.05,
				"qid": -0.10,
			},
			FDCBonus:              0.15,
			InjectablePenalty:     0.10,
			WeeklyInjectableBonus: 0.05,
			DeviceTrainingPenalty: 0.05,
		},
		Availability: AvailabilityPenalties{
			TierFactor: map[int]float64{
				1:   1.0,
				2:   0.8,
				3:   0.6,
				4:   0.4,
				999: 0.2, // Out of network
			},
			OutOfStockMultiplier: 0.2,
		},
	}
}

// CompareAndRank performs comprehensive comparison and ranking of proposals
func (e *compareAndRankEngine) CompareAndRank(
	ctx context.Context,
	request *models.CompareAndRankRequest,
) (*models.CompareAndRankResponse, error) {
	startTime := time.Now()

	e.logger.WithFields(logrus.Fields{
		"request_id":        request.RequestID,
		"candidate_count":   len(request.Candidates),
		"risk_phenotype":    request.PatientContext.RiskPhenotype,
		"weight_profile":    request.ConfigRef.WeightProfile,
	}).Info("Starting compare-and-rank process")

	// Step 1: Dominance pruning (Pareto optimization)
	pruned := e.dominancePrune(request.Candidates)
	
	// Step 2: Apply knockout rules
	eligible := e.applyKnockoutRules(pruned, request.PatientContext)

	// Step 3: Calculate normalization ranges
	normRanges := e.calculateNormalizationRanges(eligible)

	// Step 4: Score and rank proposals
	ranked, err := e.scoreAndRankProposals(eligible, request.PatientContext, request.ConfigRef, normRanges)
	if err != nil {
		return nil, fmt.Errorf("failed to score and rank proposals: %w", err)
	}

	// Step 5: Build audit information
	audit := models.RankingAuditInfo{
		NormalizationRanges: normRanges,
		ProfileUsed: models.ProfileUsedInfo{
			Weights:   request.ConfigRef.WeightProfile,
			Penalties: request.ConfigRef.PenaltiesProfile,
		},
		ProcessingTime:      time.Since(startTime),
		CandidatesProcessed: len(eligible),
		CandidatesPruned:    len(request.Candidates) - len(pruned),
	}

	e.logger.WithFields(logrus.Fields{
		"request_id":         request.RequestID,
		"candidates_pruned":  len(request.Candidates) - len(pruned),
		"candidates_ranked":  len(ranked),
		"processing_time_ms": time.Since(startTime).Milliseconds(),
		"top_therapy":        ranked[0].TherapyID,
		"top_score":          ranked[0].FinalScore,
	}).Info("Compare-and-rank process completed")

	return &models.CompareAndRankResponse{
		Ranked: ranked,
		Audit:  audit,
	}, nil
}

// dominancePrune removes dominated proposals using Pareto optimization
func (e *compareAndRankEngine) dominancePrune(candidates []models.EnhancedProposal) []models.EnhancedProposal {
	if len(candidates) <= 1 {
		return candidates
	}

	var nonDominated []models.EnhancedProposal

	for i, candidateA := range candidates {
		isDominated := false

		for j, candidateB := range candidates {
			if i == j {
				continue
			}

			// Check if B dominates A
			if e.dominates(candidateB, candidateA) {
				isDominated = true
				break
			}
		}

		if !isDominated {
			nonDominated = append(nonDominated, candidateA)
		}
	}

	e.logger.WithField("pruned_count", len(candidates)-len(nonDominated)).Debug("Dominance pruning completed")
	return nonDominated
}

// dominates checks if proposal A dominates proposal B (Pareto dominance)
func (e *compareAndRankEngine) dominates(a, b models.EnhancedProposal) bool {
	// A dominates B if A is >= B on safety and efficacy, and no worse on adherence and availability/cost
	
	safetyA := e.getSafetyValue(a)
	safetyB := e.getSafetyValue(b)
	
	efficacyA := a.Efficacy.ExpectedA1cDropPct
	efficacyB := b.Efficacy.ExpectedA1cDropPct
	
	adherenceA := e.getAdherenceValue(a)
	adherenceB := e.getAdherenceValue(b)
	
	costA := a.Cost.MonthlyEstimate
	costB := b.Cost.MonthlyEstimate
	
	availabilityA := float64(a.Availability.Tier)
	availabilityB := float64(b.Availability.Tier)

	return safetyA >= safetyB && 
		   efficacyA >= efficacyB && 
		   adherenceA >= adherenceB && 
		   (costA <= costB || availabilityA <= availabilityB) &&
		   (safetyA > safetyB || efficacyA > efficacyB) // At least one strict improvement
}

// Helper functions for dominance checking
func (e *compareAndRankEngine) getSafetyValue(proposal models.EnhancedProposal) float64 {
	base := 1.0
	
	// Apply DDI penalty
	switch proposal.Safety.ResidualDDI {
	case "major":
		base -= 0.30
	case "moderate":
		base -= 0.15
	}
	
	// Apply hypoglycemia penalty
	switch proposal.Safety.HypoPropensity {
	case "high":
		base -= 0.25
	case "med":
		base -= 0.15
	}
	
	return math.Max(0.0, base)
}

func (e *compareAndRankEngine) getAdherenceValue(proposal models.EnhancedProposal) float64 {
	base := 0.5

	// Frequency bonus
	if proposal.Adherence.DosesPerDay == 1 {
		base += 0.2
	} else if proposal.Adherence.DosesPerDay == 2 {
		base += 0.05
	} else if proposal.Adherence.DosesPerDay > 2 {
		base -= 0.1
	}

	// FDC bonus
	if proposal.Regimen.IsFDC {
		base += 0.15
	}

	return math.Max(0.0, math.Min(1.0, base))
}

// applyKnockoutRules applies contextual knockout rules before scoring
func (e *compareAndRankEngine) applyKnockoutRules(
	candidates []models.EnhancedProposal,
	context models.PatientRiskContext,
) []models.EnhancedProposal {
	var eligible []models.EnhancedProposal

	for _, candidate := range candidates {
		// Stock knockout for top slot
		if candidate.Availability.OnHand == 0 && candidate.Availability.LeadTimeDays > 7 {
			// Demote to alternatives only - for now, we'll skip this logic
			// In full implementation, this would mark as "alternatives_only"
		}

		// Preference knockout
		if context.Preferences.AvoidInjectables && e.isInjectable(candidate) {
			// Skip injectable if patient refuses and no clinical override
			continue
		}

		// Budget mode knockout
		if context.ResourceTier == "minimal" {
			// Apply budget constraints - for now, simple tier check
			if candidate.Availability.Tier > 2 {
				continue
			}
		}

		eligible = append(eligible, candidate)
	}

	e.logger.WithField("eligible_count", len(eligible)).Debug("Knockout rules applied")
	return eligible
}

// isInjectable checks if a proposal involves injectable medication
func (e *compareAndRankEngine) isInjectable(proposal models.EnhancedProposal) bool {
	route := proposal.Dose.Route
	return route == "sc" || route == "im" || route == "iv"
}

// calculateNormalizationRanges calculates min/max ranges for normalization
func (e *compareAndRankEngine) calculateNormalizationRanges(
	candidates []models.EnhancedProposal,
) map[string]models.NormalizationRange {
	if len(candidates) == 0 {
		return make(map[string]models.NormalizationRange)
	}

	// Initialize with first candidate
	minCost := candidates[0].Cost.MonthlyEstimate
	maxCost := candidates[0].Cost.MonthlyEstimate
	minA1c := candidates[0].Efficacy.ExpectedA1cDropPct
	maxA1c := candidates[0].Efficacy.ExpectedA1cDropPct

	// Find min/max values
	for _, candidate := range candidates {
		if candidate.Cost.MonthlyEstimate < minCost {
			minCost = candidate.Cost.MonthlyEstimate
		}
		if candidate.Cost.MonthlyEstimate > maxCost {
			maxCost = candidate.Cost.MonthlyEstimate
		}
		if candidate.Efficacy.ExpectedA1cDropPct < minA1c {
			minA1c = candidate.Efficacy.ExpectedA1cDropPct
		}
		if candidate.Efficacy.ExpectedA1cDropPct > maxA1c {
			maxA1c = candidate.Efficacy.ExpectedA1cDropPct
		}
	}

	return map[string]models.NormalizationRange{
		"cost": {Min: minCost, Max: maxCost},
		"a1c":  {Min: minA1c, Max: maxA1c},
	}
}

// scoreAndRankProposals performs comprehensive scoring and ranking
func (e *compareAndRankEngine) scoreAndRankProposals(
	candidates []models.EnhancedProposal,
	context models.PatientRiskContext,
	configRef models.ConfigReference,
	normRanges map[string]models.NormalizationRange,
) ([]models.EnhancedScoredProposal, error) {
	if len(candidates) == 0 {
		return []models.EnhancedScoredProposal{}, nil
	}

	// Get weight and penalties profiles
	weights, err := e.GetWeightProfile(configRef.WeightProfile)
	if err != nil {
		return nil, fmt.Errorf("failed to get weight profile: %w", err)
	}

	penalties, err := e.GetPenaltiesProfile(configRef.PenaltiesProfile)
	if err != nil {
		// Use default if not found
		penalties = e.penaltiesProfiles["default"]
	}

	var scored []models.EnhancedScoredProposal

	for _, candidate := range candidates {
		// Calculate detailed sub-scores
		subScores := e.calculateDetailedSubScores(candidate, penalties, normRanges)

		// Calculate final weighted score
		finalScore := e.calculateFinalScore(subScores, weights)

		// Generate contributions
		contributions := e.generateContributions(subScores, weights)

		// Determine eligibility
		eligibilityFlags := e.determineEligibility(candidate, context)

		// Generate explanatory notes
		notes := e.generateNotes(subScores, candidate, context)

		// Build audit info
		auditInfo := models.ProposalAuditInfo{
			KBVersions:          candidate.Provenance.KBVersions,
			RawInputs:           e.buildRawInputs(candidate),
			NormalizationRanges: normRanges,
			ProcessingTime:      time.Since(time.Now()), // Placeholder
		}

		scoredProposal := models.EnhancedScoredProposal{
			TherapyID:        candidate.TherapyID,
			SafetyVerified:   models.SafetyVerifiedProposal{}, // Would be populated from candidate
			FinalScore:       finalScore,
			Rank:             0, // Will be set after sorting
			SubScores:        subScores,
			Contributions:    contributions,
			EligibilityFlags: eligibilityFlags,
			Notes:            notes,
			AuditInfo:        auditInfo,
			ScoredAt:         time.Now(),
		}

		scored = append(scored, scoredProposal)
	}

	// Sort by final score (highest first)
	sort.Slice(scored, func(i, j int) bool {
		if math.Abs(scored[i].FinalScore-scored[j].FinalScore) < 0.001 {
			// Apply tie-breakers
			return e.applyTieBreakers(scored[i], scored[j])
		}
		return scored[i].FinalScore > scored[j].FinalScore
	})

	// Assign rankings
	for i := range scored {
		scored[i].Rank = i + 1
	}

	return scored, nil
}

// UpdateWeightProfile updates a weight profile
func (e *compareAndRankEngine) UpdateWeightProfile(profile string, weights WeightProfile) error {
	e.weightProfiles[profile] = weights
	e.logger.WithField("profile", profile).Info("Weight profile updated")
	return nil
}

// UpdatePenaltiesProfile updates a penalties profile
func (e *compareAndRankEngine) UpdatePenaltiesProfile(profile string, penalties PenaltiesProfile) error {
	e.penaltiesProfiles[profile] = penalties
	e.logger.WithField("profile", profile).Info("Penalties profile updated")
	return nil
}

// GetWeightProfile retrieves a weight profile
func (e *compareAndRankEngine) GetWeightProfile(profile string) (WeightProfile, error) {
	weights, exists := e.weightProfiles[profile]
	if !exists {
		return WeightProfile{}, fmt.Errorf("weight profile '%s' not found", profile)
	}
	return weights, nil
}

// GetPenaltiesProfile retrieves a penalties profile
func (e *compareAndRankEngine) GetPenaltiesProfile(profile string) (PenaltiesProfile, error) {
	penalties, exists := e.penaltiesProfiles[profile]
	if !exists {
		return PenaltiesProfile{}, fmt.Errorf("penalties profile '%s' not found", profile)
	}
	return penalties, nil
}

// calculateDetailedSubScores calculates detailed sub-scores for all components
func (e *compareAndRankEngine) calculateDetailedSubScores(
	candidate models.EnhancedProposal,
	penalties PenaltiesProfile,
	normRanges map[string]models.NormalizationRange,
) models.EnhancedComponentScores {
	return models.EnhancedComponentScores{
		Efficacy:     e.calculateEfficacyScore(candidate),
		Safety:       e.calculateSafetyScore(candidate, penalties),
		Availability: e.calculateAvailabilityScore(candidate, penalties),
		Cost:         e.calculateCostScore(candidate, normRanges),
		Adherence:    e.calculateAdherenceScore(candidate, penalties),
		Preference:   e.calculatePreferenceScore(candidate),
	}
}

// calculateEfficacyScore calculates detailed efficacy score
func (e *compareAndRankEngine) calculateEfficacyScore(candidate models.EnhancedProposal) models.EfficacyScoreDetail {
	// Base score from A1c drop (0% -> 0.0, 2.0% -> 1.0)
	a1cNorm := math.Min(candidate.Efficacy.ExpectedA1cDropPct/2.0, 1.0)

	// Phenotype bonuses
	phenotypeBonus := 0.0
	if candidate.Efficacy.CVBenefit {
		phenotypeBonus += 0.10
	}
	if candidate.Efficacy.HFBenefit || candidate.Efficacy.CKDBenefit {
		phenotypeBonus += 0.15
	}

	finalScore := math.Min(a1cNorm+phenotypeBonus, 1.0)

	return models.EfficacyScoreDetail{
		Score:              finalScore,
		ExpectedA1cDropPct: candidate.Efficacy.ExpectedA1cDropPct,
		CVBenefit:         candidate.Efficacy.CVBenefit,
		HFBenefit:         candidate.Efficacy.HFBenefit,
		CKDBenefit:        candidate.Efficacy.CKDBenefit,
		PhenotypeBonus:    phenotypeBonus,
		EvidenceLevel:     "high", // Placeholder
	}
}

// calculateSafetyScore calculates detailed safety score
func (e *compareAndRankEngine) calculateSafetyScore(
	candidate models.EnhancedProposal,
	penalties PenaltiesProfile,
) models.SafetyScoreDetail {
	score := 1.0
	var safetyPenalties []models.SafetyPenalty

	// DDI penalty
	if ddiPenalty, exists := penalties.Safety.ResidualDDI[candidate.Safety.ResidualDDI]; exists {
		score -= ddiPenalty
		if ddiPenalty > 0 {
			safetyPenalties = append(safetyPenalties, models.SafetyPenalty{
				Type:        "ddi",
				Description: fmt.Sprintf("Residual DDI: %s", candidate.Safety.ResidualDDI),
				Penalty:     ddiPenalty,
			})
		}
	}

	// Hypoglycemia penalty
	if hypoPenalty, exists := penalties.Safety.Hypo[candidate.Safety.HypoPropensity]; exists {
		score -= hypoPenalty
		if hypoPenalty > 0 {
			safetyPenalties = append(safetyPenalties, models.SafetyPenalty{
				Type:        "hypoglycemia",
				Description: fmt.Sprintf("Hypoglycemia risk: %s", candidate.Safety.HypoPropensity),
				Penalty:     hypoPenalty,
			})
		}
	}

	// Weight gain penalty
	if candidate.Safety.WeightEffect == "gain" {
		score -= penalties.Safety.WeightGain
		safetyPenalties = append(safetyPenalties, models.SafetyPenalty{
			Type:        "weight_gain",
			Description: "Weight gain risk",
			Penalty:     penalties.Safety.WeightGain,
		})
	}

	score = math.Max(0.0, score)

	return models.SafetyScoreDetail{
		Score:           score,
		ResidualDDI:     candidate.Safety.ResidualDDI,
		HypoPropensity:  candidate.Safety.HypoPropensity,
		WeightEffect:    candidate.Safety.WeightEffect,
		RenalFit:        candidate.Suitability.RenalFit,
		HepaticFit:      candidate.Suitability.HepaticFit,
		SafetyPenalties: safetyPenalties,
	}
}

// calculateAvailabilityScore calculates detailed availability score
func (e *compareAndRankEngine) calculateAvailabilityScore(
	candidate models.EnhancedProposal,
	penalties PenaltiesProfile,
) models.AvailabilityScoreDetail {
	// Tier factor
	tierFactor := 0.2 // Default for unknown tiers
	if factor, exists := penalties.Availability.TierFactor[candidate.Availability.Tier]; exists {
		tierFactor = factor
	}

	// Stock factor
	stockFactor := 1.0
	if candidate.Availability.OnHand == 0 {
		stockFactor = penalties.Availability.OutOfStockMultiplier
	}

	score := tierFactor * stockFactor

	return models.AvailabilityScoreDetail{
		Score:        score,
		FormularyTier: candidate.Availability.Tier,
		TierFactor:   tierFactor,
		OnHand:       candidate.Availability.OnHand,
		StockFactor:  stockFactor,
		LeadTimeDays: candidate.Availability.LeadTimeDays,
	}
}

// calculateCostScore calculates detailed cost score
func (e *compareAndRankEngine) calculateCostScore(
	candidate models.EnhancedProposal,
	normRanges map[string]models.NormalizationRange,
) models.CostScoreDetail {
	costRange, exists := normRanges["cost"]
	if !exists || costRange.Max == costRange.Min {
		return models.CostScoreDetail{
			Score:           0.5, // Default middle score
			MonthlyEstimate: candidate.Cost.MonthlyEstimate,
			Currency:        candidate.Cost.Currency,
			PatientCopay:    candidate.Cost.PatientCopay,
			NormalizedCost:  0.5,
		}
	}

	// Higher score = cheaper (inverted normalization)
	normalizedCost := (candidate.Cost.MonthlyEstimate - costRange.Min) / (costRange.Max - costRange.Min)
	score := 1.0 - normalizedCost // Invert so lower cost = higher score

	return models.CostScoreDetail{
		Score:           math.Max(0.0, math.Min(1.0, score)),
		MonthlyEstimate: candidate.Cost.MonthlyEstimate,
		Currency:        candidate.Cost.Currency,
		PatientCopay:    candidate.Cost.PatientCopay,
		NormalizedCost:  normalizedCost,
	}
}

// calculateAdherenceScore calculates detailed adherence score
func (e *compareAndRankEngine) calculateAdherenceScore(
	candidate models.EnhancedProposal,
	penalties PenaltiesProfile,
) models.AdherenceScoreDetail {
	score := penalties.Adherence.Base

	// Frequency bonus
	frequencyBonus := 0.0
	switch candidate.Adherence.DosesPerDay {
	case 1:
		if bonus, exists := penalties.Adherence.FrequencyBonus["od"]; exists {
			frequencyBonus = bonus
		}
	case 2:
		if bonus, exists := penalties.Adherence.FrequencyBonus["bid"]; exists {
			frequencyBonus = bonus
		}
	case 3:
		if bonus, exists := penalties.Adherence.FrequencyBonus["tid"]; exists {
			frequencyBonus = bonus
		}
	case 4:
		if bonus, exists := penalties.Adherence.FrequencyBonus["qid"]; exists {
			frequencyBonus = bonus
		}
	}
	score += frequencyBonus

	// FDC bonus
	fdcBonus := 0.0
	if candidate.Regimen.IsFDC {
		fdcBonus = penalties.Adherence.FDCBonus
		score += fdcBonus
	}

	// Injectable penalty
	injectablePenalty := 0.0
	if e.isInjectable(candidate) {
		injectablePenalty = penalties.Adherence.InjectablePenalty
		score -= injectablePenalty

		// Weekly injectable bonus (offset some penalty)
		if candidate.Adherence.DosesPerDay <= 1 { // Assuming weekly or less frequent
			score += penalties.Adherence.WeeklyInjectableBonus
		}
	}

	// Device training penalty
	deviceTrainingPenalty := 0.0
	if candidate.Adherence.RequiresTraining {
		deviceTrainingPenalty = penalties.Adherence.DeviceTrainingPenalty
		score -= deviceTrainingPenalty
	}

	score = math.Max(0.0, math.Min(1.0, score))

	return models.AdherenceScoreDetail{
		Score:                 score,
		BaseScore:             penalties.Adherence.Base,
		FrequencyBonus:        frequencyBonus,
		FDCBonus:              fdcBonus,
		InjectablePenalty:     injectablePenalty,
		DeviceTrainingPenalty: deviceTrainingPenalty,
		PillBurden:            candidate.Adherence.PillBurden,
		DosesPerDay:           candidate.Adherence.DosesPerDay,
		IsFDC:                 candidate.Regimen.IsFDC,
		RequiresDevice:        candidate.Adherence.RequiresDevice,
	}
}

// calculatePreferenceScore calculates detailed preference score
func (e *compareAndRankEngine) calculatePreferenceScore(candidate models.EnhancedProposal) models.PreferenceScoreDetail {
	score := 1.0
	var violatedPreferences []models.ViolatedPreference

	// Injectable preference violation
	if candidate.Preferences.AvoidInjectables && e.isInjectable(candidate) {
		penalty := 0.3 // Strong preference violation
		score -= penalty
		violatedPreferences = append(violatedPreferences, models.ViolatedPreference{
			Type:        "strong",
			Description: "Patient prefers to avoid injectables",
			Penalty:     penalty,
		})
	}

	// Once daily preference
	if candidate.Preferences.OnceDailyPreferred && candidate.Adherence.DosesPerDay > 1 {
		penalty := 0.1 // Soft preference violation
		score -= penalty
		violatedPreferences = append(violatedPreferences, models.ViolatedPreference{
			Type:        "soft",
			Description: "Patient prefers once-daily dosing",
			Penalty:     penalty,
		})
	}

	score = math.Max(0.0, score)

	return models.PreferenceScoreDetail{
		Score:               score,
		BaseScore:           1.0,
		ViolatedPreferences: violatedPreferences,
	}
}

// calculateFinalScore calculates the weighted final score
func (e *compareAndRankEngine) calculateFinalScore(
	subScores models.EnhancedComponentScores,
	weights WeightProfile,
) float64 {
	finalScore := (subScores.Efficacy.Score * weights.Efficacy) +
		(subScores.Safety.Score * weights.Safety) +
		(subScores.Availability.Score * weights.Availability) +
		(subScores.Cost.Score * weights.Cost) +
		(subScores.Adherence.Score * weights.Adherence) +
		(subScores.Preference.Score * weights.Preference)

	return math.Max(0.0, math.Min(1.0, finalScore))
}

// generateContributions generates score contribution breakdown
func (e *compareAndRankEngine) generateContributions(
	subScores models.EnhancedComponentScores,
	weights WeightProfile,
) []models.ScoreContribution {
	return []models.ScoreContribution{
		{
			Factor:       "efficacy",
			Value:        subScores.Efficacy.Score,
			Weight:       weights.Efficacy,
			Contribution: subScores.Efficacy.Score * weights.Efficacy,
			Note:         e.generateEfficacyNote(subScores.Efficacy),
		},
		{
			Factor:       "safety",
			Value:        subScores.Safety.Score,
			Weight:       weights.Safety,
			Contribution: subScores.Safety.Score * weights.Safety,
			Note:         e.generateSafetyNote(subScores.Safety),
		},
		{
			Factor:       "availability",
			Value:        subScores.Availability.Score,
			Weight:       weights.Availability,
			Contribution: subScores.Availability.Score * weights.Availability,
			Note:         e.generateAvailabilityNote(subScores.Availability),
		},
		{
			Factor:       "cost",
			Value:        subScores.Cost.Score,
			Weight:       weights.Cost,
			Contribution: subScores.Cost.Score * weights.Cost,
			Note:         e.generateCostNote(subScores.Cost),
		},
		{
			Factor:       "adherence",
			Value:        subScores.Adherence.Score,
			Weight:       weights.Adherence,
			Contribution: subScores.Adherence.Score * weights.Adherence,
			Note:         e.generateAdherenceNote(subScores.Adherence),
		},
		{
			Factor:       "preference",
			Value:        subScores.Preference.Score,
			Weight:       weights.Preference,
			Contribution: subScores.Preference.Score * weights.Preference,
			Note:         e.generatePreferenceNote(subScores.Preference),
		},
	}
}

// determineEligibility determines top-slot eligibility
func (e *compareAndRankEngine) determineEligibility(
	candidate models.EnhancedProposal,
	context models.PatientRiskContext,
) models.EligibilityFlags {
	eligible := true
	var reasons []string

	// Stock eligibility
	if candidate.Availability.OnHand == 0 && candidate.Availability.LeadTimeDays > 7 {
		eligible = false
		reasons = append(reasons, "Out of stock with extended lead time")
	}

	// Preference eligibility
	if context.Preferences.AvoidInjectables && e.isInjectable(candidate) {
		eligible = false
		reasons = append(reasons, "Patient preference against injectables")
	}

	// Budget eligibility
	if context.ResourceTier == "minimal" && candidate.Availability.Tier > 2 {
		eligible = false
		reasons = append(reasons, "Exceeds budget constraints")
	}

	return models.EligibilityFlags{
		TopSlotEligible: eligible,
		Reasons:         reasons,
	}
}

// generateNotes generates explanatory notes
func (e *compareAndRankEngine) generateNotes(
	subScores models.EnhancedComponentScores,
	candidate models.EnhancedProposal,
	context models.PatientRiskContext,
) []string {
	var notes []string

	// Top contributing factors
	contributions := []struct {
		name  string
		score float64
	}{
		{"efficacy", subScores.Efficacy.Score},
		{"safety", subScores.Safety.Score},
		{"availability", subScores.Availability.Score},
		{"cost", subScores.Cost.Score},
		{"adherence", subScores.Adherence.Score},
		{"preference", subScores.Preference.Score},
	}

	// Sort by score (highest first)
	sort.Slice(contributions, func(i, j int) bool {
		return contributions[i].score > contributions[j].score
	})

	// Generate notes for top 3 factors
	for i := 0; i < 3 && i < len(contributions); i++ {
		factor := contributions[i]
		switch factor.name {
		case "efficacy":
			if subScores.Efficacy.CVBenefit || subScores.Efficacy.HFBenefit || subScores.Efficacy.CKDBenefit {
				notes = append(notes, "Strong efficacy with cardiovascular/renal benefits")
			} else {
				notes = append(notes, fmt.Sprintf("Good efficacy (%.1f%% A1c reduction)", subScores.Efficacy.ExpectedA1cDropPct))
			}
		case "safety":
			if len(subScores.Safety.SafetyPenalties) == 0 {
				notes = append(notes, "Excellent safety profile")
			} else {
				notes = append(notes, "Acceptable safety with minor concerns")
			}
		case "availability":
			if subScores.Availability.FormularyTier <= 2 {
				notes = append(notes, "Preferred formulary status")
			}
		case "cost":
			notes = append(notes, "Cost-effective option")
		case "adherence":
			if subScores.Adherence.DosesPerDay == 1 {
				notes = append(notes, "Convenient once-daily dosing")
			}
		case "preference":
			if len(subScores.Preference.ViolatedPreferences) == 0 {
				notes = append(notes, "Aligns with patient preferences")
			}
		}
	}

	return notes
}

// Note generation helper methods
func (e *compareAndRankEngine) generateEfficacyNote(efficacy models.EfficacyScoreDetail) string {
	if efficacy.CVBenefit || efficacy.HFBenefit || efficacy.CKDBenefit {
		return fmt.Sprintf("%.1f%% A1c reduction with CV/renal benefits", efficacy.ExpectedA1cDropPct)
	}
	return fmt.Sprintf("%.1f%% expected A1c reduction", efficacy.ExpectedA1cDropPct)
}

func (e *compareAndRankEngine) generateSafetyNote(safety models.SafetyScoreDetail) string {
	if len(safety.SafetyPenalties) == 0 {
		return "Excellent safety profile"
	}
	return fmt.Sprintf("Good safety with %d minor concerns", len(safety.SafetyPenalties))
}

func (e *compareAndRankEngine) generateAvailabilityNote(availability models.AvailabilityScoreDetail) string {
	if availability.FormularyTier <= 2 {
		return fmt.Sprintf("Tier %d formulary status", availability.FormularyTier)
	}
	return fmt.Sprintf("Tier %d formulary", availability.FormularyTier)
}

func (e *compareAndRankEngine) generateCostNote(cost models.CostScoreDetail) string {
	return fmt.Sprintf("$%.2f monthly cost", cost.MonthlyEstimate)
}

func (e *compareAndRankEngine) generateAdherenceNote(adherence models.AdherenceScoreDetail) string {
	if adherence.DosesPerDay == 1 {
		return "Once-daily dosing"
	}
	return fmt.Sprintf("%d times daily", adherence.DosesPerDay)
}

func (e *compareAndRankEngine) generatePreferenceNote(preference models.PreferenceScoreDetail) string {
	if len(preference.ViolatedPreferences) == 0 {
		return "Aligns with patient preferences"
	}
	return fmt.Sprintf("%d preference concerns", len(preference.ViolatedPreferences))
}

// applyTieBreakers applies tie-breaking rules for equal scores
func (e *compareAndRankEngine) applyTieBreakers(a, b models.EnhancedScoredProposal) bool {
	// 1. Higher safety score
	if math.Abs(a.SubScores.Safety.Score-b.SubScores.Safety.Score) > 0.001 {
		return a.SubScores.Safety.Score > b.SubScores.Safety.Score
	}

	// 2. Higher efficacy score
	if math.Abs(a.SubScores.Efficacy.Score-b.SubScores.Efficacy.Score) > 0.001 {
		return a.SubScores.Efficacy.Score > b.SubScores.Efficacy.Score
	}

	// 3. Lower cost (higher cost score)
	if math.Abs(a.SubScores.Cost.Score-b.SubScores.Cost.Score) > 0.001 {
		return a.SubScores.Cost.Score > b.SubScores.Cost.Score
	}

	// 4. Better adherence (fewer doses per day)
	if a.SubScores.Adherence.DosesPerDay != b.SubScores.Adherence.DosesPerDay {
		return a.SubScores.Adherence.DosesPerDay < b.SubScores.Adherence.DosesPerDay
	}

	// 5. Deterministic: lexicographic on therapy ID
	return a.TherapyID < b.TherapyID
}

// buildRawInputs builds raw input data for audit trail
func (e *compareAndRankEngine) buildRawInputs(candidate models.EnhancedProposal) map[string]interface{} {
	return map[string]interface{}{
		"therapy_id":         candidate.TherapyID,
		"class":              candidate.Class,
		"agent":              candidate.Agent,
		"expected_a1c_drop":  candidate.Efficacy.ExpectedA1cDropPct,
		"cv_benefit":         candidate.Efficacy.CVBenefit,
		"residual_ddi":       candidate.Safety.ResidualDDI,
		"hypo_propensity":    candidate.Safety.HypoPropensity,
		"formulary_tier":     candidate.Availability.Tier,
		"monthly_cost":       candidate.Cost.MonthlyEstimate,
		"doses_per_day":      candidate.Adherence.DosesPerDay,
		"is_fdc":             candidate.Regimen.IsFDC,
		"route":              candidate.Dose.Route,
	}
}

// loadKBConfiguration loads configuration from KB config file
func (e *compareAndRankEngine) loadKBConfiguration() error {
	if e.kbConfigLoader == nil {
		return fmt.Errorf("no KB config loader available")
	}

	config, err := e.kbConfigLoader.LoadConfig()
	if err != nil {
		return fmt.Errorf("failed to load KB configuration: %w", err)
	}

	// Load weight profiles
	for profileName, profileConfig := range config.Profiles {
		e.weightProfiles[profileName] = profileConfig.Weights
	}

	// Load penalties profile
	penalties := PenaltiesProfile{
		Safety: SafetyPenalties{
			ResidualDDI: config.Penalties.Safety.ResidualDDI,
			Hypo:        config.Penalties.Safety.Hypo,
			WeightGain:  config.Penalties.Safety.WeightGain,
		},
		Adherence: AdherencePenalties{
			Base:                   config.Penalties.Adherence.Base,
			FrequencyBonus:         config.Penalties.Adherence.FrequencyBonus,
			FDCBonus:              config.Penalties.Adherence.FDCBonus,
			InjectablePenalty:     config.Penalties.Adherence.InjectablePenalty,
			WeeklyInjectableBonus: config.Penalties.Adherence.WeeklyInjectableBonus,
			DeviceTrainingPenalty: config.Penalties.Adherence.DeviceTrainingPenalty,
		},
		Availability: AvailabilityPenalties{
			TierFactor:           config.Penalties.Availability.TierFactor,
			OutOfStockMultiplier: config.Penalties.Availability.OutOfStockMultiplier,
		},
	}
	e.penaltiesProfiles["default"] = penalties

	e.logger.WithFields(logrus.Fields{
		"weight_profiles_loaded":    len(e.weightProfiles),
		"penalties_profiles_loaded": len(e.penaltiesProfiles),
		"kb_version":               config.Metadata.Version,
	}).Info("KB configuration loaded successfully")

	return nil
}

// ReloadKBConfiguration reloads KB configuration if needed
func (e *compareAndRankEngine) ReloadKBConfiguration() error {
	if e.kbConfigLoader == nil {
		return nil // No KB loader, nothing to reload
	}

	return e.kbConfigLoader.ReloadIfNeeded()
}
