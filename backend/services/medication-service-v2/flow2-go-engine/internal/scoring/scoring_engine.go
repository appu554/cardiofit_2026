// Package scoring provides multi-factor scoring for medication recommendations
package scoring

import (
	"context"
	"math"
	"sort"
	"time"

	"flow2-go-engine/internal/models"
	"github.com/sirupsen/logrus"
)

// ScoringEngine interface for medication proposal scoring
type ScoringEngine interface {
	ScoreAndRankProposals(ctx context.Context, proposals []*models.SafetyVerifiedProposal) ([]*models.ScoredProposal, error)
	UpdateScoringWeights(weights ScoringWeights) error
	GetScoringWeights() ScoringWeights
}

// scoringEngine implements ScoringEngine
type scoringEngine struct {
	config ScoringConfig
	logger *logrus.Logger
}

// ScoringConfig holds configuration for the scoring engine
type ScoringConfig struct {
	Weights           ScoringWeights `json:"weights" yaml:"weights"`
	EfficacyDataSource string        `json:"efficacy_data_source" yaml:"efficacy_data_source"`
	CostDataSource     string        `json:"cost_data_source" yaml:"cost_data_source"`
	GuidelineSource    string        `json:"guideline_source" yaml:"guideline_source"`
}

// ScoringWeights defines the weights for different scoring components
type ScoringWeights struct {
	Safety             float64 `json:"safety" yaml:"safety"`
	Efficacy           float64 `json:"efficacy" yaml:"efficacy"`
	Cost               float64 `json:"cost" yaml:"cost"`
	Convenience        float64 `json:"convenience" yaml:"convenience"`
	PatientPreference  float64 `json:"patient_preference" yaml:"patient_preference"`
	GuidelineAdherence float64 `json:"guideline_adherence" yaml:"guideline_adherence"`
}

// DefaultScoringWeights returns default scoring weights
func DefaultScoringWeights() ScoringWeights {
	return ScoringWeights{
		Safety:             0.30, // Highest priority
		Efficacy:           0.25, // Second highest
		Cost:               0.15,
		Convenience:        0.10,
		PatientPreference:  0.10,
		GuidelineAdherence: 0.10,
	}
}

// SafetyFirstWeights returns safety-prioritized weights
func SafetyFirstWeights() ScoringWeights {
	return ScoringWeights{
		Safety:             0.50, // Heavily prioritize safety
		Efficacy:           0.20,
		Cost:               0.10,
		Convenience:        0.05,
		PatientPreference:  0.05,
		GuidelineAdherence: 0.10,
	}
}

// CostConsciousWeights returns cost-conscious weights
func CostConsciousWeights() ScoringWeights {
	return ScoringWeights{
		Safety:             0.25,
		Efficacy:           0.20,
		Cost:               0.30, // Prioritize cost
		Convenience:        0.10,
		PatientPreference:  0.05,
		GuidelineAdherence: 0.10,
	}
}

// GuidelineAdherentWeights returns guideline-adherent weights
func GuidelineAdherentWeights() ScoringWeights {
	return ScoringWeights{
		Safety:             0.25,
		Efficacy:           0.20,
		Cost:               0.10,
		Convenience:        0.05,
		PatientPreference:  0.05,
		GuidelineAdherence: 0.35, // Prioritize guidelines
	}
}

// NewScoringEngine creates a new scoring engine
func NewScoringEngine(config ScoringConfig, logger *logrus.Logger) ScoringEngine {
	if logger == nil {
		logger = logrus.New()
	}

	// Validate and normalize weights
	config.Weights = normalizeWeights(config.Weights)

	return &scoringEngine{
		config: config,
		logger: logger,
	}
}

// ScoreAndRankProposals scores and ranks medication proposals
func (s *scoringEngine) ScoreAndRankProposals(
	ctx context.Context,
	proposals []*models.SafetyVerifiedProposal,
) ([]*models.ScoredProposal, error) {
	if len(proposals) == 0 {
		return []*models.ScoredProposal{}, nil
	}

	s.logger.WithField("proposal_count", len(proposals)).Debug("Starting proposal scoring")

	var scoredProposals []*models.ScoredProposal

	for _, proposal := range proposals {
		// Calculate component scores
		componentScores := s.calculateComponentScores(proposal)

		// Calculate weighted total score
		totalScore := s.calculateTotalScore(componentScores)

		// Create scored proposal
		scored := &models.ScoredProposal{
			SafetyVerified:  *proposal,
			TotalScore:      totalScore,
			ComponentScores: componentScores,
			Ranking:         0, // Will be set after sorting
			ScoredAt:        time.Now(),
		}

		scoredProposals = append(scoredProposals, scored)
	}

	// Sort by total score (highest first)
	sort.Slice(scoredProposals, func(i, j int) bool {
		return scoredProposals[i].TotalScore > scoredProposals[j].TotalScore
	})

	// Assign rankings
	for i, scored := range scoredProposals {
		scored.Ranking = i + 1
	}

	s.logger.WithFields(logrus.Fields{
		"scored_proposals": len(scoredProposals),
		"top_score":        scoredProposals[0].TotalScore,
		"top_medication":   scoredProposals[0].SafetyVerified.Original.MedicationName,
	}).Info("Proposal scoring completed")

	return scoredProposals, nil
}

// calculateComponentScores calculates individual component scores
func (s *scoringEngine) calculateComponentScores(proposal *models.SafetyVerifiedProposal) models.ComponentScores {
	return models.ComponentScores{
		SafetyScore:             s.calculateSafetyScore(proposal),
		EfficacyScore:           s.calculateEfficacyScore(proposal),
		CostScore:               s.calculateCostScore(proposal),
		ConvenienceScore:        s.calculateConvenienceScore(proposal),
		PatientPreferenceScore:  s.calculatePatientPreferenceScore(proposal),
		GuidelineAdherenceScore: s.calculateGuidelineAdherenceScore(proposal),
	}
}

// calculateSafetyScore calculates safety score from JIT Safety results
func (s *scoringEngine) calculateSafetyScore(proposal *models.SafetyVerifiedProposal) float64 {
	// Base score from JIT Safety
	baseScore := proposal.SafetyScore

	// Apply penalties for DDI warnings
	ddiPenalty := 0.0
	for _, ddi := range proposal.DDIWarnings {
		switch ddi.Severity {
		case "major":
			ddiPenalty += 0.15
		case "moderate":
			ddiPenalty += 0.10
		case "minor":
			ddiPenalty += 0.05
		}
	}

	// Apply penalties for safety reasons
	reasonPenalty := 0.0
	for _, reason := range proposal.SafetyReasons {
		switch reason.Severity {
		case "warn":
			reasonPenalty += 0.05
		case "error":
			reasonPenalty += 0.10
		}
	}

	finalScore := baseScore - ddiPenalty - reasonPenalty
	if finalScore < 0.0 {
		finalScore = 0.0
	}

	return finalScore
}

// calculateEfficacyScore calculates efficacy score based on clinical evidence
func (s *scoringEngine) calculateEfficacyScore(proposal *models.SafetyVerifiedProposal) float64 {
	// Placeholder implementation - in real system, this would query clinical databases
	// For now, use medication class and indication to estimate efficacy

	medicationName := proposal.Original.MedicationName
	
	// Simple efficacy mapping based on medication type
	switch {
	case contains(medicationName, "lisinopril"):
		return 0.85 // ACE inhibitors are highly effective for HTN
	case contains(medicationName, "metformin"):
		return 0.90 // Metformin is first-line for T2DM
	case contains(medicationName, "empagliflozin"):
		return 0.80 // SGLT2 inhibitors are effective with CV benefits
	case contains(medicationName, "insulin"):
		return 0.95 // Insulin is highly effective for glucose control
	default:
		return 0.75 // Default moderate efficacy
	}
}

// calculateCostScore calculates cost score based on formulary and pricing
func (s *scoringEngine) calculateCostScore(proposal *models.SafetyVerifiedProposal) float64 {
	// Placeholder implementation - in real system, this would query formulary databases
	
	medicationName := proposal.Original.MedicationName
	
	// Simple cost mapping (higher score = lower cost)
	switch {
	case contains(medicationName, "lisinopril"):
		return 0.95 // Generic ACE inhibitor - very low cost
	case contains(medicationName, "metformin"):
		return 0.95 // Generic - very low cost
	case contains(medicationName, "empagliflozin"):
		return 0.30 // Brand SGLT2 inhibitor - high cost
	case contains(medicationName, "insulin"):
		return 0.60 // Moderate cost depending on type
	default:
		return 0.70 // Default moderate cost
	}
}

// calculateConvenienceScore calculates convenience score based on dosing frequency
func (s *scoringEngine) calculateConvenienceScore(proposal *models.SafetyVerifiedProposal) float64 {
	// Base score on dosing frequency
	intervalH := proposal.FinalDose.IntervalH
	
	switch {
	case intervalH >= 24: // Once daily
		return 1.0
	case intervalH >= 12: // Twice daily
		return 0.8
	case intervalH >= 8:  // Three times daily
		return 0.6
	case intervalH >= 6:  // Four times daily
		return 0.4
	default:              // More frequent
		return 0.2
	}
}

// calculatePatientPreferenceScore calculates patient preference score
func (s *scoringEngine) calculatePatientPreferenceScore(proposal *models.SafetyVerifiedProposal) float64 {
	// Placeholder implementation - in real system, this would consider:
	// - Patient's previous medication history
	// - Route preferences (oral vs injection)
	// - Side effect tolerance
	// - Lifestyle factors
	
	route := proposal.FinalDose.Route
	
	switch route {
	case "po": // Oral - generally preferred
		return 0.9
	case "sc": // Subcutaneous - moderate preference
		return 0.6
	case "iv": // Intravenous - less preferred for outpatient
		return 0.3
	default:
		return 0.7
	}
}

// calculateGuidelineAdherenceScore calculates guideline adherence score
func (s *scoringEngine) calculateGuidelineAdherenceScore(proposal *models.SafetyVerifiedProposal) float64 {
	// Placeholder implementation - in real system, this would check against:
	// - ADA guidelines for diabetes
	// - ACC/AHA guidelines for cardiovascular disease
	// - KDIGO guidelines for kidney disease
	// - Local institutional guidelines
	
	medicationName := proposal.Original.MedicationName
	
	// Simple guideline adherence mapping
	switch {
	case contains(medicationName, "metformin"):
		return 1.0 // First-line per ADA guidelines
	case contains(medicationName, "lisinopril"):
		return 0.9 // Recommended for HTN and CKD
	case contains(medicationName, "empagliflozin"):
		return 0.85 // Recommended for T2DM with CV benefits
	case contains(medicationName, "insulin"):
		return 0.8 // Appropriate when indicated
	default:
		return 0.7 // Default moderate adherence
	}
}

// calculateTotalScore calculates weighted total score
func (s *scoringEngine) calculateTotalScore(scores models.ComponentScores) float64 {
	weights := s.config.Weights
	
	totalScore := (scores.SafetyScore * weights.Safety) +
		(scores.EfficacyScore * weights.Efficacy) +
		(scores.CostScore * weights.Cost) +
		(scores.ConvenienceScore * weights.Convenience) +
		(scores.PatientPreferenceScore * weights.PatientPreference) +
		(scores.GuidelineAdherenceScore * weights.GuidelineAdherence)
	
	// Ensure score is between 0 and 1
	return math.Max(0.0, math.Min(1.0, totalScore))
}

// UpdateScoringWeights updates the scoring weights
func (s *scoringEngine) UpdateScoringWeights(weights ScoringWeights) error {
	s.config.Weights = normalizeWeights(weights)
	s.logger.WithField("weights", s.config.Weights).Info("Scoring weights updated")
	return nil
}

// GetScoringWeights returns current scoring weights
func (s *scoringEngine) GetScoringWeights() ScoringWeights {
	return s.config.Weights
}

// Helper functions

// normalizeWeights ensures weights sum to 1.0
func normalizeWeights(weights ScoringWeights) ScoringWeights {
	total := weights.Safety + weights.Efficacy + weights.Cost + 
		weights.Convenience + weights.PatientPreference + weights.GuidelineAdherence
	
	if total == 0 {
		return DefaultScoringWeights()
	}
	
	return ScoringWeights{
		Safety:             weights.Safety / total,
		Efficacy:           weights.Efficacy / total,
		Cost:               weights.Cost / total,
		Convenience:        weights.Convenience / total,
		PatientPreference:  weights.PatientPreference / total,
		GuidelineAdherence: weights.GuidelineAdherence / total,
	}
}

// contains checks if a string contains a substring (case-insensitive)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && 
		(s == substr || 
		 (len(s) > len(substr) && 
		  (s[:len(substr)] == substr || 
		   s[len(s)-len(substr):] == substr ||
		   containsSubstring(s, substr))))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
