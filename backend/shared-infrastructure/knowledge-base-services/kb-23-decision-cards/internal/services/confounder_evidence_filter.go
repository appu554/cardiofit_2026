package services

import "fmt"

// IOROutcomeResult is the outcome data consumed by KB-23 for evidence display.
type IOROutcomeResult struct {
	DeltaValue      float64 `json:"delta_value"`
	ConfidenceLevel string  `json:"confidence_level"`
	ConfounderScore float64 `json:"confounder_score"`
	Narrative       string  `json:"narrative,omitempty"`
}

// FilterIORByConfidence filters IOR outcomes based on confounder confidence.
func FilterIORByConfidence(outcomes []IOROutcomeResult, minConfidence string) []IOROutcomeResult {
	confidenceRank := map[string]int{"HIGH": 3, "MODERATE": 2, "LOW": 1}
	minRank := confidenceRank[minConfidence]
	if minRank == 0 {
		minRank = 1
	}
	var filtered []IOROutcomeResult
	for _, o := range outcomes {
		rank := confidenceRank[o.ConfidenceLevel]
		if rank >= minRank {
			filtered = append(filtered, o)
		}
	}
	return filtered
}

// AnnotateEvidenceWithConfounderContext adds confounder context to card evidence chains.
func AnnotateEvidenceWithConfounderContext(
	evidenceSummary string,
	totalOutcomes int,
	highConfidenceCount int,
	moderateCount int,
	lowCount int,
) string {
	if totalOutcomes == 0 {
		return evidenceSummary
	}
	highPct := float64(highConfidenceCount) / float64(totalOutcomes) * 100
	if highPct >= 70 {
		return fmt.Sprintf("%s (evidence quality: HIGH — majority of outcomes have minimal confounding)", evidenceSummary)
	}
	if highPct >= 40 {
		return fmt.Sprintf("%s (evidence quality: MODERATE — some outcomes affected by confounders)", evidenceSummary)
	}
	return fmt.Sprintf("%s (evidence quality: LOW — significant confounding in outcome data; interpret with caution)", evidenceSummary)
}
