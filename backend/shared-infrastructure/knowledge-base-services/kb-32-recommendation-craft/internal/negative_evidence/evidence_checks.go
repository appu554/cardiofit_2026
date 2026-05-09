// Package negative_evidence — this file implements the integration layer that
// wires absence-pattern queries into the deprescribing recommendation pipeline.
//
// VisibilityClass: AD — negative-evidence audit per Guidelines §7
//
// Key exported symbols:
//
//   - DefaultAbsenceQueryForStopRule — maps a STOP rule ID to the appropriate
//     AbsenceQuery using the canonical rule-to-pattern table.
//
//   - AttachNegativeEvidence — augments a generator.Packet whose Type=="STOP"
//     with the EvidenceText from a successful absence confirmation.
//     For non-STOP packets the function is a no-op; for STOP packets where
//     presence is detected the evidence section is left unchanged.
package negative_evidence

import (
	"context"
	"fmt"
	"strings"

	"github.com/cardiofit/kb32/internal/generator"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// DefaultAbsenceQueryForStopRule
// ---------------------------------------------------------------------------

// DefaultAbsenceQueryForStopRule returns the canonical AbsenceQuery for a
// given STOP ruleID. The mapping covers the three common deprescribing patterns;
// unknown rule IDs fall back to a bounded-window query on "general_observation"
// with a 90-day window.
//
// Rule-to-pattern table:
//
//	PostFall / fall-related   → PatternBoundedWindow,          "fall",                 90 days
//	PPI                       → PatternIndicationDocumentation, "ppi_indication",       0 (not meaningful)
//	BenzodiazepineLongTerm    → PatternPeriodicReview,          "benzodiazepine_review", 365 days
//	<all other rule IDs>      → PatternBoundedWindow,           "general_observation",  90 days
func DefaultAbsenceQueryForStopRule(ruleID string, residentID uuid.UUID) AbsenceQuery {
	upper := strings.ToLower(ruleID)

	switch {
	case strings.Contains(upper, "postfall") || strings.Contains(upper, "fall"):
		return AbsenceQuery{
			Pattern:         PatternBoundedWindow,
			ResidentID:      residentID,
			ObservationKind: "fall",
			WindowDays:      90,
		}

	case strings.Contains(upper, "ppi"):
		return AbsenceQuery{
			Pattern:         PatternIndicationDocumentation,
			ResidentID:      residentID,
			ObservationKind: "ppi_indication",
			WindowDays:      0,
		}

	case strings.Contains(upper, "benzodiazepine"):
		return AbsenceQuery{
			Pattern:         PatternPeriodicReview,
			ResidentID:      residentID,
			ObservationKind: "benzodiazepine_review",
			WindowDays:      365,
		}

	default:
		return AbsenceQuery{
			Pattern:         PatternBoundedWindow,
			ResidentID:      residentID,
			ObservationKind: "general_observation",
			WindowDays:      90,
		}
	}
}

// ---------------------------------------------------------------------------
// AttachNegativeEvidence
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// QuerierAttacher — adapter satisfying generator.NegativeEvidenceAttacher
// ---------------------------------------------------------------------------

// QuerierAttacher wraps a Querier and exposes an AttachTo method that satisfies
// the generator.NegativeEvidenceAttacher interface. Use NewQuerierAttacher to
// construct one and pass it to generator.GenerateWithNegativeEvidence.
type QuerierAttacher struct {
	q Querier
}

// NewQuerierAttacher returns a QuerierAttacher backed by q.
func NewQuerierAttacher(q Querier) *QuerierAttacher {
	return &QuerierAttacher{q: q}
}

// AttachTo implements generator.NegativeEvidenceAttacher by delegating to
// AttachNegativeEvidence with the wrapped Querier.
func (a *QuerierAttacher) AttachTo(ctx context.Context, pkt *generator.Packet, residentID uuid.UUID) error {
	return AttachNegativeEvidence(ctx, a.q, pkt, residentID)
}

// ---------------------------------------------------------------------------
// AttachNegativeEvidence (standalone function)
// ---------------------------------------------------------------------------

// AttachNegativeEvidence augments packet.Sections["evidence"] with an absence
// defensibility statement when:
//
//  1. packet.Type == "STOP"
//  2. The absence query (derived from DefaultAbsenceQueryForStopRule) confirms
//     that the target observation is genuinely absent (result.Confirmed==true).
//
// When packet.Type != "STOP" the function returns nil immediately without
// touching the packet (safe no-op for MONITOR, DOSE_CHANGE, ADD packets).
//
// When the querier detects presence (result.Confirmed==false) the evidence
// section is left unchanged — the absence claim cannot be made.
//
// Querier errors are propagated directly to the caller.
func AttachNegativeEvidence(ctx context.Context, q Querier, packet *generator.Packet, residentID uuid.UUID) error {
	if packet.Type != "STOP" {
		return nil
	}

	query := DefaultAbsenceQueryForStopRule(packet.AppliedRule.RuleID, residentID)

	result, err := q.QueryAbsence(ctx, query)
	if err != nil {
		return fmt.Errorf("negative_evidence: query absence for rule %s: %w",
			packet.AppliedRule.RuleID, err)
	}

	if !result.Confirmed {
		// Presence detected — leave evidence section as-is; absence claim withheld.
		return nil
	}

	// Absence confirmed — write the human-readable defensibility text into the
	// evidence section, replacing the N/A placeholder set by generator.Generate.
	packet.Sections["evidence"] = result.EvidenceText
	return nil
}
