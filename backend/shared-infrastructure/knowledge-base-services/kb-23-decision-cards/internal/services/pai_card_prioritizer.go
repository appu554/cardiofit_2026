package services

import "sort"

// PAICardEntry pairs a pending decision card with its patient's PAI score.
type PAICardEntry struct {
	CardID    string  `json:"card_id"`
	PatientID string  `json:"patient_id"`
	PAIScore  float64 `json:"pai_score"`
	PAITier   string  `json:"pai_tier,omitempty"`
	HasPAI    bool    `json:"has_pai"`
}

// PrioritizeCardsByPAI sorts pending cards by patient PAI score descending.
// Cards without PAI scores are sorted to the end.
// Uses stable sort to preserve original order for equal PAI scores.
func PrioritizeCardsByPAI(cards []PAICardEntry) []PAICardEntry {
	if len(cards) == 0 {
		return cards
	}

	// Make a copy to avoid mutating the input
	result := make([]PAICardEntry, len(cards))
	copy(result, cards)

	sort.SliceStable(result, func(i, j int) bool {
		// Cards with PAI always before cards without
		if result[i].HasPAI != result[j].HasPAI {
			return result[i].HasPAI
		}
		// Among cards with PAI, higher score first
		return result[i].PAIScore > result[j].PAIScore
	})

	return result
}

// AnnotateCardWithPAI adds PAI context to a card entry.
func AnnotateCardWithPAI(entry *PAICardEntry, score float64, tier string) {
	entry.PAIScore = score
	entry.PAITier = tier
	entry.HasPAI = true
}
