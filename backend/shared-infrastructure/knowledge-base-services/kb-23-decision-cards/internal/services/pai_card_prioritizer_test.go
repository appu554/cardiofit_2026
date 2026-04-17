package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// PAICardEntry is defined in pai_card_prioritizer.go.

func TestPrioritize_CriticalFirst(t *testing.T) {
	cards := []PAICardEntry{
		{CardID: "card-1", PatientID: "P001", PAIScore: 45},
		{CardID: "card-2", PatientID: "P002", PAIScore: 85},
		{CardID: "card-3", PatientID: "P003", PAIScore: 72},
	}

	sorted := PrioritizeCardsByPAI(cards)

	assert.Equal(t, "card-2", sorted[0].CardID, "highest PAI first")
	assert.Equal(t, "card-3", sorted[1].CardID)
	assert.Equal(t, "card-1", sorted[2].CardID, "lowest PAI last")
}

func TestPrioritize_EqualPAI_StableOrder(t *testing.T) {
	cards := []PAICardEntry{
		{CardID: "card-a", PatientID: "P001", PAIScore: 50},
		{CardID: "card-b", PatientID: "P002", PAIScore: 50},
		{CardID: "card-c", PatientID: "P003", PAIScore: 50},
	}

	sorted := PrioritizeCardsByPAI(cards)

	// Stable sort: original order preserved when PAI is equal
	assert.Equal(t, "card-a", sorted[0].CardID)
	assert.Equal(t, "card-b", sorted[1].CardID)
	assert.Equal(t, "card-c", sorted[2].CardID)
}

func TestPrioritize_MissingPAI_Last(t *testing.T) {
	cards := []PAICardEntry{
		{CardID: "card-1", PatientID: "P001", PAIScore: 0, HasPAI: false},
		{CardID: "card-2", PatientID: "P002", PAIScore: 60, HasPAI: true},
		{CardID: "card-3", PatientID: "P003", PAIScore: 0, HasPAI: false},
		{CardID: "card-4", PatientID: "P004", PAIScore: 40, HasPAI: true},
	}

	sorted := PrioritizeCardsByPAI(cards)

	// Cards with PAI first (sorted by score), then cards without PAI
	assert.Equal(t, "card-2", sorted[0].CardID)
	assert.Equal(t, "card-4", sorted[1].CardID)
	// Missing-PAI cards at end
	assert.False(t, sorted[2].HasPAI)
	assert.False(t, sorted[3].HasPAI)
}

func TestPrioritize_Empty_NoError(t *testing.T) {
	sorted := PrioritizeCardsByPAI(nil)
	assert.Empty(t, sorted)

	sorted2 := PrioritizeCardsByPAI([]PAICardEntry{})
	assert.Empty(t, sorted2)
}
