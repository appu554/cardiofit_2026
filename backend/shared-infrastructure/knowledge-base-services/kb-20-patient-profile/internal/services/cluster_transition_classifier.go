package services

import "kb-patient-profile/internal/models"

// ClassifyTransition labels a cluster transition for downstream card generation.
// It is a pure function — no state, no side effects.
//
// Priority order:
//  1. Override events present         → GENUINE_TRANSITION
//  2. Engagement collapse             → UNCERTAIN  (data-quality confounder)
//  3. Seasonal window active          → UNCERTAIN  (environmental confounder)
//  4. Insufficient history (<2 records) → UNCERTAIN
//  5. Oscillation detected (A↔B)       → PROBABLE_FLAP
//  6. Default directional move         → GENUINE_TRANSITION
func ClassifyTransition(
	fromCluster, toCluster string,
	overrideEvents []models.OverrideEvent,
	recentHistory []models.ClusterTransitionRecord,
	seasonalActive bool,
	engagementCollapse bool,
) string {
	// 1. Clinical override evidence trumps everything.
	if len(overrideEvents) > 0 {
		return models.ClassificationGenuine
	}

	// 2. Engagement collapse → data quality issue.
	if engagementCollapse {
		return models.ClassificationUncertain
	}

	// 3. Seasonal window → environmental confounder.
	if seasonalActive {
		return models.ClassificationUncertain
	}

	// 4. Insufficient history.
	if len(recentHistory) < 2 {
		return models.ClassificationUncertain
	}

	// 5. Oscillation detection: check if the pair (from, to) has appeared
	//    in reverse in recent history.
	if isOscillating(fromCluster, toCluster, recentHistory) {
		return models.ClassificationFlap
	}

	// 6. Directional move with sufficient history and no confounders.
	return models.ClassificationGenuine
}

// isOscillating returns true if the reverse pair (to→from) appears anywhere
// in recentHistory, indicating the patient is bouncing between two clusters.
func isOscillating(from, to string, history []models.ClusterTransitionRecord) bool {
	for _, rec := range history {
		if rec.PreviousCluster == to && rec.NewCluster == from {
			return true
		}
	}
	return false
}
