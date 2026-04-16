package services

import (
	"fmt"
	"strings"
)

// PhenotypeStabilityDecision mirrors the relevant fields from KB-20's StabilityDecision.
// This avoids a direct import dependency between KB-23 and KB-20.
type PhenotypeStabilityDecision struct {
	PatientID          string
	RawClusterLabel    string
	StableClusterLabel string
	Decision           string   // ACCEPT | HOLD_DWELL | HOLD_FLAP | OVERRIDE_EVENT
	TransitionType     string   // GENUINE | FLAP_DAMPENED | OVERRIDE | INITIAL | ""
	DomainDriver       string
	Confidence         float64
	PreviousCluster    string   // the cluster before the decision
	FlapPair           []string // populated when Decision=HOLD_FLAP
}

// PhenotypeTransitionCard represents a card generated from a phenotype stability decision.
type PhenotypeTransitionCard struct {
	TemplateID      string
	PatientID       string
	PreviousCluster string
	NewCluster      string
	DomainDriver    string
	Confidence      float64
	FlapPair        string // formatted "A <-> B" for the flap warning
	SuppressInertia bool   // true when patient is stable-good -> skip inertia detection
}

// EvaluatePhenotypeTransition generates card(s) based on a stability decision.
// It uses the PhenotypeStabilityDecision from KB-20 (passed by the caller —
// KB-23 does not call KB-20 directly for this).
func EvaluatePhenotypeTransition(decision PhenotypeStabilityDecision) []PhenotypeTransitionCard {
	switch decision.Decision {
	case "ACCEPT":
		return evaluateAccept(decision)
	case "HOLD_FLAP":
		return evaluateHoldFlap(decision)
	case "HOLD_DWELL":
		// Silent hold — the engine is just waiting, nothing to report.
		return nil
	default:
		return nil
	}
}

func evaluateAccept(d PhenotypeStabilityDecision) []PhenotypeTransitionCard {
	// INITIAL assignment — first phenotype, nothing to compare against.
	if d.TransitionType == "INITIAL" {
		return nil
	}

	// No transition (same cluster confirmed) — check for stable-good suppression.
	if d.TransitionType == "" {
		if d.StableClusterLabel == "STABLE_CONTROLLED" {
			return []PhenotypeTransitionCard{{
				TemplateID:      "",
				PatientID:       d.PatientID,
				NewCluster:      d.StableClusterLabel,
				SuppressInertia: true,
			}}
		}
		return nil
	}

	// Genuine transition — previous cluster differs from the new stable cluster.
	if d.PreviousCluster != d.StableClusterLabel {
		return []PhenotypeTransitionCard{{
			TemplateID:      "dc-phenotype-transition-v1",
			PatientID:       d.PatientID,
			PreviousCluster: d.PreviousCluster,
			NewCluster:      d.StableClusterLabel,
			DomainDriver:    d.DomainDriver,
			Confidence:      d.Confidence,
		}}
	}

	return nil
}

func evaluateHoldFlap(d PhenotypeStabilityDecision) []PhenotypeTransitionCard {
	pair := formatFlapPair(d.FlapPair)
	return []PhenotypeTransitionCard{{
		TemplateID:   "dc-phenotype-flap-warning-v1",
		PatientID:    d.PatientID,
		FlapPair:     pair,
		DomainDriver: d.DomainDriver,
		Confidence:   d.Confidence,
	}}
}

func formatFlapPair(pair []string) string {
	if len(pair) == 2 {
		return fmt.Sprintf("%s ↔ %s", pair[0], pair[1])
	}
	return strings.Join(pair, " ↔ ")
}
