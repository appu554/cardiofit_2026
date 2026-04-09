package services

import "kb-23-decision-cards/internal/models"

// ---------------------------------------------------------------------------
// InertiaCard — decision card for therapeutic inertia
// ---------------------------------------------------------------------------

// InertiaCard represents a clinical decision card raised for therapeutic
// inertia, including severity-mapped urgency and evidence chain.
type InertiaCard struct {
	CardType      string        `json:"card_type"`
	Urgency       string        `json:"urgency"`
	Title         string        `json:"title"`
	Rationale     string        `json:"rationale"`
	EvidenceChain EvidenceChain `json:"evidence_chain"`
}

// Card type constants
const (
	CardTypeTherapeuticInertia = "THERAPEUTIC_INERTIA"
	CardTypeDualDomainInertia  = "DUAL_DOMAIN_INERTIA"
)

// ---------------------------------------------------------------------------
// GenerateInertiaCards — produce decision cards from patient inertia report
// ---------------------------------------------------------------------------

// GenerateInertiaCards creates one THERAPEUTIC_INERTIA card per detected
// verdict, with severity-mapped urgency. If dual-domain inertia is present,
// an additional DUAL_DOMAIN_INERTIA card with IMMEDIATE urgency is appended.
func GenerateInertiaCards(report models.PatientInertiaReport) []InertiaCard {
	var cards []InertiaCard

	for _, v := range report.Verdicts {
		if !v.Detected {
			continue
		}
		// Skip the synthetic dual-domain verdict — handled separately below.
		if v.Pattern == models.PatternDualDomainInertia {
			continue
		}

		evidence := BuildEvidenceChain(v)
		urgency := mapSeverityToUrgency(v.Severity)

		card := InertiaCard{
			CardType:      CardTypeTherapeuticInertia,
			Urgency:       urgency,
			Title:         "Therapeutic inertia detected: " + string(v.Domain),
			Rationale:     evidence.Summary,
			EvidenceChain: evidence,
		}
		cards = append(cards, card)
	}

	// Dual-domain inertia card — always IMMEDIATE urgency
	if report.HasDualDomainInertia {
		card := InertiaCard{
			CardType:  CardTypeDualDomainInertia,
			Urgency:   UrgencyImmediate,
			Title:     "Dual-domain therapeutic inertia — concordant uncontrolled status",
			Rationale: "Both glycaemic and hemodynamic domains show unaddressed therapeutic inertia, compounding cardiovascular risk",
		}
		cards = append(cards, card)
	}

	return cards
}

// ---------------------------------------------------------------------------
// mapSeverityToUrgency — severity bracket to card urgency
// ---------------------------------------------------------------------------

func mapSeverityToUrgency(severity models.InertiaSeverity) string {
	switch severity {
	case models.SeverityCritical:
		return UrgencyImmediate
	case models.SeveritySevere, models.SeverityModerate:
		return UrgencyUrgent
	case models.SeverityMild:
		return UrgencyRoutine
	default:
		return UrgencyScheduled
	}
}
