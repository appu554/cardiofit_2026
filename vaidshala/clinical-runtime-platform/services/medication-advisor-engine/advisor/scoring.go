package advisor

import (
	"sort"

	"github.com/google/uuid"
)

// ProposalScoringEngine handles weighted scoring and ranking of medication proposals
type ProposalScoringEngine struct {
	weights ScoringWeights
}

// ScoringWeights defines the weights for each quality factor
type ScoringWeights struct {
	Guideline   float64 `json:"guideline"`   // 30%
	Safety      float64 `json:"safety"`      // 25%
	Efficacy    float64 `json:"efficacy"`    // 20%
	Interaction float64 `json:"interaction"` // 15%
	Monitoring  float64 `json:"monitoring"`  // 10%
}

// DefaultWeights returns the default scoring weights
func DefaultWeights() ScoringWeights {
	return ScoringWeights{
		Guideline:   0.30,
		Safety:      0.25,
		Efficacy:    0.20,
		Interaction: 0.15,
		Monitoring:  0.10,
	}
}

// NewProposalScoringEngine creates a new scoring engine with default weights
func NewProposalScoringEngine() *ProposalScoringEngine {
	return &ProposalScoringEngine{
		weights: DefaultWeights(),
	}
}

// NewProposalScoringEngineWithWeights creates a scoring engine with custom weights
func NewProposalScoringEngineWithWeights(weights ScoringWeights) *ProposalScoringEngine {
	return &ProposalScoringEngine{
		weights: weights,
	}
}

// RankProposals scores and ranks medication candidates
func (pse *ProposalScoringEngine) RankProposals(candidates []MedicationCandidate) []MedicationProposal {
	proposals := make([]MedicationProposal, len(candidates))

	for i, candidate := range candidates {
		// Calculate weighted score
		score := pse.calculateWeightedScore(candidate.Scores)

		proposals[i] = MedicationProposal{
			ID:             uuid.New(),
			Medication:     candidate.Medication,
			Dosage:         candidate.Dosage,
			QualityScore:   score,
			QualityFactors: candidate.Scores,
			Rationale:      candidate.Rationale,
			Warnings:       candidate.Warnings,
		}
	}

	// Sort by quality score descending
	sort.Slice(proposals, func(i, j int) bool {
		return proposals[i].QualityScore > proposals[j].QualityScore
	})

	// Assign ranks
	for i := range proposals {
		proposals[i].Rank = i + 1
	}

	return proposals
}

// calculateWeightedScore computes the weighted quality score
func (pse *ProposalScoringEngine) calculateWeightedScore(factors QualityFactors) float64 {
	return factors.Guideline*pse.weights.Guideline +
		factors.Safety*pse.weights.Safety +
		factors.Efficacy*pse.weights.Efficacy +
		factors.Interaction*pse.weights.Interaction +
		factors.Monitoring*pse.weights.Monitoring
}

// ScoreBreakdown provides detailed scoring breakdown
type ScoreBreakdown struct {
	GuidelineScore      float64 `json:"guideline_score"`
	GuidelineWeighted   float64 `json:"guideline_weighted"`
	SafetyScore         float64 `json:"safety_score"`
	SafetyWeighted      float64 `json:"safety_weighted"`
	EfficacyScore       float64 `json:"efficacy_score"`
	EfficacyWeighted    float64 `json:"efficacy_weighted"`
	InteractionScore    float64 `json:"interaction_score"`
	InteractionWeighted float64 `json:"interaction_weighted"`
	MonitoringScore     float64 `json:"monitoring_score"`
	MonitoringWeighted  float64 `json:"monitoring_weighted"`
	TotalScore          float64 `json:"total_score"`
}

// GetScoreBreakdown returns detailed scoring breakdown for a candidate
func (pse *ProposalScoringEngine) GetScoreBreakdown(factors QualityFactors) ScoreBreakdown {
	return ScoreBreakdown{
		GuidelineScore:      factors.Guideline,
		GuidelineWeighted:   factors.Guideline * pse.weights.Guideline,
		SafetyScore:         factors.Safety,
		SafetyWeighted:      factors.Safety * pse.weights.Safety,
		EfficacyScore:       factors.Efficacy,
		EfficacyWeighted:    factors.Efficacy * pse.weights.Efficacy,
		InteractionScore:    factors.Interaction,
		InteractionWeighted: factors.Interaction * pse.weights.Interaction,
		MonitoringScore:     factors.Monitoring,
		MonitoringWeighted:  factors.Monitoring * pse.weights.Monitoring,
		TotalScore:          pse.calculateWeightedScore(factors),
	}
}

// CompareProposals compares two proposals and returns the better one
func (pse *ProposalScoringEngine) CompareProposals(a, b MedicationProposal) int {
	if a.QualityScore > b.QualityScore {
		return 1
	} else if a.QualityScore < b.QualityScore {
		return -1
	}
	return 0
}

// FilterByMinScore filters proposals that meet minimum score threshold
func (pse *ProposalScoringEngine) FilterByMinScore(proposals []MedicationProposal, minScore float64) []MedicationProposal {
	filtered := []MedicationProposal{}

	for _, p := range proposals {
		if p.QualityScore >= minScore {
			filtered = append(filtered, p)
		}
	}

	return filtered
}

// GetTopN returns the top N proposals
func (pse *ProposalScoringEngine) GetTopN(proposals []MedicationProposal, n int) []MedicationProposal {
	if n >= len(proposals) {
		return proposals
	}
	return proposals[:n]
}

// RescoreWithCustomWeights re-ranks proposals with different weights
func (pse *ProposalScoringEngine) RescoreWithCustomWeights(
	proposals []MedicationProposal,
	weights ScoringWeights,
) []MedicationProposal {
	// Create new scoring engine with custom weights
	customScorer := NewProposalScoringEngineWithWeights(weights)

	// Convert back to candidates for rescoring
	candidates := make([]MedicationCandidate, len(proposals))
	for i, p := range proposals {
		candidates[i] = MedicationCandidate{
			Medication: p.Medication,
			Dosage:     p.Dosage,
			Scores:     p.QualityFactors,
			Rationale:  p.Rationale,
			Warnings:   p.Warnings,
		}
	}

	return customScorer.RankProposals(candidates)
}

// ScoringSummary provides a summary of the scoring for a proposal set
type ScoringSummary struct {
	TotalProposals      int     `json:"total_proposals"`
	AverageScore        float64 `json:"average_score"`
	HighestScore        float64 `json:"highest_score"`
	LowestScore         float64 `json:"lowest_score"`
	ScoreStdDev         float64 `json:"score_std_dev"`
	AboveThresholdCount int     `json:"above_threshold_count"`
}

// GetSummary returns a scoring summary for a set of proposals
func (pse *ProposalScoringEngine) GetSummary(proposals []MedicationProposal, threshold float64) ScoringSummary {
	if len(proposals) == 0 {
		return ScoringSummary{}
	}

	var total, highest, lowest float64
	lowest = 1.0
	aboveThreshold := 0

	for _, p := range proposals {
		total += p.QualityScore
		if p.QualityScore > highest {
			highest = p.QualityScore
		}
		if p.QualityScore < lowest {
			lowest = p.QualityScore
		}
		if p.QualityScore >= threshold {
			aboveThreshold++
		}
	}

	avg := total / float64(len(proposals))

	// Calculate standard deviation
	var sumSquares float64
	for _, p := range proposals {
		diff := p.QualityScore - avg
		sumSquares += diff * diff
	}
	stdDev := 0.0
	if len(proposals) > 1 {
		stdDev = sumSquares / float64(len(proposals)-1)
	}

	return ScoringSummary{
		TotalProposals:      len(proposals),
		AverageScore:        avg,
		HighestScore:        highest,
		LowestScore:         lowest,
		ScoreStdDev:         stdDev,
		AboveThresholdCount: aboveThreshold,
	}
}
