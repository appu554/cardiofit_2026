package services

import (
	"testing"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// TestGenerateInertiaCards_SingleDomain
// ---------------------------------------------------------------------------

// Verifies that a single glycaemic verdict with MODERATE severity produces
// one THERAPEUTIC_INERTIA card with URGENT urgency.
func TestGenerateInertiaCards_SingleDomain(t *testing.T) {
	report := models.PatientInertiaReport{
		PatientID:   "patient-inertia-01",
		EvaluatedAt: time.Now(),
		Verdicts: []models.InertiaVerdict{
			{
				Domain:              models.DomainGlycaemic,
				Pattern:             models.PatternHbA1cInertia,
				Detected:            true,
				InertiaDurationDays: 200,
				CurrentValue:        8.5,
				TargetValue:         7.0,
				Severity:            models.SeverityModerate,
			},
		},
		HasAnyInertia:        true,
		HasDualDomainInertia: false,
	}

	cards := GenerateInertiaCards(report)

	if len(cards) != 1 {
		t.Fatalf("want 1 card, got %d", len(cards))
	}

	card := cards[0]
	if card.CardType != CardTypeTherapeuticInertia {
		t.Errorf("CardType: want %s, got %s", CardTypeTherapeuticInertia, card.CardType)
	}
	if card.Urgency != UrgencyUrgent {
		t.Errorf("Urgency: want URGENT, got %s", card.Urgency)
	}
	if card.Title == "" {
		t.Error("Title: want non-empty")
	}
	if card.EvidenceChain.Summary == "" {
		t.Error("EvidenceChain.Summary: want non-empty")
	}
}

// ---------------------------------------------------------------------------
// TestGenerateInertiaCards_DualDomain
// ---------------------------------------------------------------------------

// Verifies that when HasDualDomainInertia is true, a DUAL_DOMAIN_INERTIA
// card with IMMEDIATE urgency is generated in addition to per-verdict cards.
func TestGenerateInertiaCards_DualDomain(t *testing.T) {
	report := models.PatientInertiaReport{
		PatientID:   "patient-inertia-02",
		EvaluatedAt: time.Now(),
		Verdicts: []models.InertiaVerdict{
			{
				Domain:              models.DomainGlycaemic,
				Pattern:             models.PatternHbA1cInertia,
				Detected:            true,
				InertiaDurationDays: 200,
				CurrentValue:        8.5,
				TargetValue:         7.0,
				Severity:            models.SeverityModerate,
			},
			{
				Domain:              models.DomainHemodynamic,
				Pattern:             models.PatternBPInertia,
				Detected:            true,
				InertiaDurationDays: 150,
				CurrentValue:        155,
				TargetValue:         130,
				Severity:            models.SeverityMild,
			},
			{
				// Synthetic dual-domain verdict (produced by DetectInertia)
				Domain:   models.DomainGlycaemic,
				Pattern:  models.PatternDualDomainInertia,
				Detected: true,
				Severity: models.SeverityCritical,
			},
		},
		HasAnyInertia:        true,
		HasDualDomainInertia: true,
	}

	cards := GenerateInertiaCards(report)

	// Expect: 1 glycaemic + 1 hemodynamic + 1 dual-domain = 3 cards
	if len(cards) != 3 {
		t.Fatalf("want 3 cards, got %d", len(cards))
	}

	// Find the dual-domain card
	var dualCard *InertiaCard
	for i, c := range cards {
		if c.CardType == CardTypeDualDomainInertia {
			dualCard = &cards[i]
			break
		}
	}
	if dualCard == nil {
		t.Fatal("DUAL_DOMAIN_INERTIA card not found")
	}
	if dualCard.Urgency != UrgencyImmediate {
		t.Errorf("DUAL_DOMAIN_INERTIA urgency: want IMMEDIATE, got %s", dualCard.Urgency)
	}
}
