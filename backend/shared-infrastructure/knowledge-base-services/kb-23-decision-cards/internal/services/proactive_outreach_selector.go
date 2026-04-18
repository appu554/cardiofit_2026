package services

import "sort"

// ProactiveOutreachItem represents a patient eligible for proactive outreach.
type ProactiveOutreachItem struct {
	PatientID               string   `json:"patient_id"`
	PatientName             string   `json:"patient_name"`
	RiskScore               float64  `json:"risk_score"`
	RiskTier                string   `json:"risk_tier"`
	PrimaryReason           string   `json:"primary_reason"`
	RecommendedAction       string   `json:"recommended_action"`
	CounterfactualReduction float64  `json:"counterfactual_reduction"`
	ModifiableDrivers       []string `json:"modifiable_drivers"`
}

// PredictedRiskSummary is a simplified prediction consumed by the outreach selector.
// Mirrors KB-26's PredictedRisk output (Option α coupling — no cross-service import).
type PredictedRiskSummary struct {
	PatientID               string
	RiskScore               float64
	RiskTier                string
	RiskSummary             string
	RecommendedAction       string
	CounterfactualReduction float64
	ModifiableDriverNames   []string
}

// SelectProactiveOutreach returns patients eligible for proactive outreach.
// Filters: risk >= minScore, PAI not in excludeTiers, cooldown met.
// Sorts by risk score descending, truncates to maxItems.
func SelectProactiveOutreach(
	predictions []PredictedRiskSummary,
	currentPAITiers map[string]string,
	lastContactDays map[string]int,
	maxItems int,
	minRiskScore float64,
	excludePAITiers []string,
	cooldownDays int,
) []ProactiveOutreachItem {
	excludeSet := make(map[string]bool, len(excludePAITiers))
	for _, tier := range excludePAITiers {
		excludeSet[tier] = true
	}

	var candidates []ProactiveOutreachItem

	for _, p := range predictions {
		// Filter: risk score must meet minimum threshold.
		if p.RiskScore < minRiskScore {
			continue
		}

		// Filter: exclude patients whose current PAI tier is in the urgent set.
		if paiTier, ok := currentPAITiers[p.PatientID]; ok && excludeSet[paiTier] {
			continue
		}

		// Filter: cooldown — skip patients contacted too recently.
		if daysSince, ok := lastContactDays[p.PatientID]; ok && daysSince < cooldownDays {
			continue
		}

		candidates = append(candidates, ProactiveOutreachItem{
			PatientID:               p.PatientID,
			PatientName:             p.PatientID, // placeholder — resolved from KB-20 in Sprint 2
			RiskScore:               p.RiskScore,
			RiskTier:                p.RiskTier,
			PrimaryReason:           p.RiskSummary,
			RecommendedAction:       p.RecommendedAction,
			CounterfactualReduction: p.CounterfactualReduction,
			ModifiableDrivers:       p.ModifiableDriverNames,
		})
	}

	// Sort by risk score descending (highest risk first).
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].RiskScore > candidates[j].RiskScore
	})

	// Truncate to daily cap.
	if len(candidates) > maxItems {
		candidates = candidates[:maxItems]
	}

	return candidates
}
