package services

import (
	"sort"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// Sort-score weights
// ---------------------------------------------------------------------------

func escalationTierWeight(tier string) float64 {
	switch tier {
	case "SAFETY":
		return 5000
	case "IMMEDIATE":
		return 4000
	case "URGENT":
		return 3000
	case "ROUTINE":
		return 1000
	default:
		return 0
	}
}

func trajectoryBoost(trend string) float64 {
	switch trend {
	case "RISING":
		return 50
	case "STABLE":
		return 0
	case "FALLING":
		return -20
	default:
		return 0
	}
}

func attentionGapBoost(lastDays int) float64 {
	return float64(lastDays) * 2
}

func transitionBoost(tags []string) float64 {
	for _, t := range tags {
		if t == "POST_DISCHARGE" {
			return 30
		}
	}
	return 0
}

// computeSortScore returns the composite sort score for a worklist item.
func computeSortScore(item models.WorklistItem) float64 {
	return escalationTierWeight(item.EscalationTier) +
		item.PAIScore*10 +
		trajectoryBoost(item.PAITrend) +
		attentionGapBoost(item.LastClinicianDays) +
		transitionBoost(item.ContextTags)
}

// urgencyTierFromScore maps a sort score to a display urgency tier.
func urgencyTierFromScore(score float64) string {
	switch {
	case score >= 4000:
		return models.WorklistTierCritical
	case score >= 3000:
		return models.WorklistTierHigh
	case score >= 1000:
		return models.WorklistTierModerate
	default:
		return models.WorklistTierLow
	}
}

// SortAndTierWorklist sorts items by composite score, assigns urgency tiers,
// truncates to maxItems (never truncating CRITICAL), and returns a WorklistView
// with tier counts.
func SortAndTierWorklist(items []models.WorklistItem, maxItems int) models.WorklistView {
	type scored struct {
		item  models.WorklistItem
		score float64
	}

	scored_items := make([]scored, len(items))
	for i, it := range items {
		scored_items[i] = scored{item: it, score: computeSortScore(it)}
	}

	// Stable sort descending by score.
	sort.SliceStable(scored_items, func(i, j int) bool {
		return scored_items[i].score > scored_items[j].score
	})

	// Preserve aggregator's urgency tier — do NOT re-derive from score.
	// The aggregator assigned tiers based on clinical cascade logic
	// (SAFETY escalation → CRITICAL, PAI CRITICAL → CRITICAL, etc.).
	// The score is used only for ordering within tiers, not for
	// tier reassignment. Re-deriving would downgrade PAI-CRITICAL
	// patients (no escalation, low score) from CRITICAL to LOW.

	// Truncate to maxItems, never removing CRITICAL items.
	var result []models.WorklistItem
	nonCriticalCount := 0
	maxNonCritical := maxItems // we'll compute after counting criticals

	// First pass: count criticals.
	criticals := 0
	for _, s := range scored_items {
		if s.item.UrgencyTier == models.WorklistTierCritical {
			criticals++
		}
	}
	if maxItems > criticals {
		maxNonCritical = maxItems - criticals
	} else {
		maxNonCritical = 0
	}

	for _, s := range scored_items {
		if s.item.UrgencyTier == models.WorklistTierCritical {
			result = append(result, s.item)
		} else {
			if nonCriticalCount < maxNonCritical {
				result = append(result, s.item)
				nonCriticalCount++
			}
		}
	}

	// Compute tier counts.
	view := models.WorklistView{
		Items:         result,
		TotalCount:    len(result),
		LastRefreshed: time.Now(),
	}
	for _, it := range result {
		switch it.UrgencyTier {
		case models.WorklistTierCritical:
			view.CriticalCount++
		case models.WorklistTierHigh:
			view.HighCount++
		case models.WorklistTierModerate:
			view.ModerateCount++
		case models.WorklistTierLow:
			view.LowCount++
		}
	}

	return view
}
