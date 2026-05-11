// Package appropriateness — substrate_scorer.go implements SubstrateBackedScorer,
// the production replacement for DefaultAppropriatenessSource. It scores each of
// the five appropriateness dimensions against the ClinicalSnapshot + Packet +
// ApplicableRule actually produced by Stages 1–3 of the craft pipeline.
//
// VisibilityClass: AD (audit-defensible) — drives the Stage 4 clinical-safety gate.
//
// Scoring philosophy: when a dimension cannot be evaluated from available
// substrate (e.g. the current ApplicableRule schema does not yet carry the
// metadata a dimension would inspect), the scorer returns 3 (neutral) rather
// than inflating to 5. Each such branch is annotated below. Neutral defaults
// keep the gate honest: a 3 still passes (HoldThreshold=2) but does not falsely
// claim evidence of strong appropriateness.
//
// Phase 2-completion Task 2.
package appropriateness

import (
	"context"
	"strings"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/reasoning"
)

// SubstrateBackedScorer implements api.AppropriatenessSource using the live
// ClinicalSnapshot, Packet, and ApplicableRule produced by earlier pipeline
// stages. The zero value is usable; NewSubstrateBackedScorer is provided for
// symmetry with the rest of the package.
type SubstrateBackedScorer struct{}

// NewSubstrateBackedScorer returns a SubstrateBackedScorer ready for use.
func NewSubstrateBackedScorer() *SubstrateBackedScorer {
	return &SubstrateBackedScorer{}
}

// Assess scores the five appropriateness dimensions for the supplied draft.
// It never returns an error in the current implementation (all branches reduce
// to deterministic substrate inspection), but preserves the (Assessment, error)
// signature so future I/O-backed enrichments (e.g. evidence-anchor freshness
// lookups against the citations registry) can propagate failures.
func (s *SubstrateBackedScorer) Assess(
	_ context.Context,
	pkt *generator.Packet,
	snap kb32ctx.ClinicalSnapshot,
	rule reasoning.ApplicableRule,
) (Assessment, error) {
	return Assessment{
		ClinicalWarrant:        s.scoreClinicalWarrant(rule, snap),
		EvidenceSolidity:       s.scoreEvidenceSolidity(pkt, rule),
		AlternativesConsidered: s.scoreAlternativesConsidered(rule),
		RestraintConsidered:    s.scoreRestraintConsidered(snap, rule),
		GoalsOfCareAlignment:   s.scoreGoalsOfCareAlignment(rule, snap),
	}, nil
}

// scoreClinicalWarrant — does the recommendation address a real clinical
// issue at this resident's substrate state?
//
//	5: rule type matches a substrate signal (STOP with ACB ≥ 3, STOP with DBI ≥ 1,
//	   MONITOR with eGFR < 30, DOSE_CHANGE with eGFR 30–60).
//	3: rule fires but no direct substrate corroboration (neutral default).
//	1: rule fires against a contraindicated state (ADD aggressive for end_of_life,
//	   STOP psychotropic on a newly-admitted resident with no other signals).
func (s *SubstrateBackedScorer) scoreClinicalWarrant(
	rule reasoning.ApplicableRule, snap kb32ctx.ClinicalSnapshot,
) int {
	switch rule.Type {
	case "STOP":
		// Contraindicated: STOP psychotropic-class rule on a freshly-admitted
		// resident with no anticholinergic or polypharmacy signal — likely
		// pre-emptive stopping during transition; clinically risky.
		if snap.RecentAdmission72h && snap.ACB < 2 && snap.DBI < 1.0 {
			if isPsychotropicRule(rule.RuleID) {
				return 1
			}
		}
		if snap.ACB >= 3 || snap.DBI >= 1.0 {
			return 5
		}
	case "MONITOR":
		// Strong renal signal supports active monitoring.
		if snap.EGFR > 0 && snap.EGFR < 30 {
			return 5
		}
	case "DOSE_CHANGE":
		if snap.EGFR >= 30 && snap.EGFR < 60 {
			return 5
		}
	case "ADD":
		// Contraindicated: aggressive ADD intervention in an end_of_life or
		// palliative resident (mirrors GoalsOfCare misalignment but is also a
		// warrant problem — the substrate state itself argues against acting).
		if snap.CareIntensity == "end_of_life" {
			return 1
		}
	}
	// Default neutral — rule fired but substrate does not corroborate or
	// contraindicate at the level the scorer can detect.
	return 3
}

// scoreEvidenceSolidity — strength of evidence anchors.
//
//	5: 2+ AU-jurisdiction anchors detectable in the packet's evidence section.
//	3: 1 AU anchor, OR 2+ international anchors, OR no anchors discernible at
//	   Stage 4 (Stage 5 builds the canonical anchor list — neutral default).
//	1: explicit retracted/superseded markers in the evidence text.
//
// NOTE: The generator.Packet does not yet carry a structured EvidenceAnchors
// slice at Stage 4 — anchors are assembled into framing.ClinicalContent at
// Stage 5. This scorer inspects the rendered evidence section text plus the
// RuleID prefix as a best-effort substrate. When the future Packet schema gains
// structured anchors, replace the string-scan with field reads.
func (s *SubstrateBackedScorer) scoreEvidenceSolidity(
	pkt *generator.Packet, rule reasoning.ApplicableRule,
) int {
	if pkt == nil {
		return 3
	}
	evidence := strings.ToUpper(pkt.Sections["evidence"])

	if strings.Contains(evidence, "RETRACTED") || strings.Contains(evidence, "SUPERSEDED") {
		return 1
	}

	auMarkers := []string{"ADG-", "AU-", "-AU", "TGA-", "RACGP-"}
	intlMarkers := []string{"NICE-", "KDIGO-", "STOPP-", "BEERS-", "WHO-", "MHRA-"}

	auHits := countMarkers(evidence, auMarkers)
	intlHits := countMarkers(evidence, intlMarkers)

	// RuleID prefix can also hint at AU jurisdiction when the evidence section
	// is still template.NA (Stage 4 is before evidence enrichment).
	if auHits == 0 && hasAUPrefix(rule.RuleID) {
		auHits = 1
	}

	if auHits >= 2 {
		return 5
	}
	if auHits == 1 || intlHits >= 2 {
		return 3
	}
	// Cannot discern anchors at Stage 4 (evidence section often still NA).
	// Default neutral rather than penalise — Stage 5 / Task 3 will pin actual
	// citations and a future iteration of this scorer can read them back.
	return 3
}

// scoreAlternativesConsidered — has the rule's underlying CQL evaluated alternatives?
//
//	5: rule explicitly checked alternative interventions (RuleID convention:
//	   suffix "-ALT" or "+ALT" — used by deprescribing-with-substitute rules).
//	3: rule has alternative-check naming but unfilled metadata.
//	1: rule has no alternative-consideration metadata at all.
//
// NOTE: The current ApplicableRule schema does not carry structured
// alternative-check metadata — only RuleID/Type/Urgency. We infer from RuleID
// naming conventions used by kb-cql-runtime authors. ADD-class rules without
// an "-ALT" suffix are treated as score=1 because by definition an ADD without
// a documented alternative-check has not considered restraint or substitution.
// Other types default to neutral 3.
func (s *SubstrateBackedScorer) scoreAlternativesConsidered(
	rule reasoning.ApplicableRule,
) int {
	id := strings.ToUpper(rule.RuleID)
	// Check unfilled-metadata markers FIRST: "ALT-PENDING" / "ALT-TBD" also
	// contain "-ALT" as a substring, so order matters.
	if strings.Contains(id, "ALT-PENDING") || strings.Contains(id, "ALT-TBD") {
		return 3
	}
	if strings.Contains(id, "-ALT") || strings.HasSuffix(id, "+ALT") {
		return 5
	}
	if rule.Type == "ADD" {
		// ADD without alternative-check naming = no documented consideration
		// of less-invasive options. Clinically equivalent to "rule has no
		// alternative-consideration metadata".
		return 1
	}
	// STOP/MONITOR/DOSE_CHANGE without explicit metadata: cannot decide from
	// current ApplicableRule schema → neutral default. Promote to 5 once the
	// CQL runtime starts emitting structured alternative-check results.
	return 3
}

// scoreRestraintConsidered — did the rule run restraint signal detectors?
//
//	5: restraint signaler ran (snapshot carries the kb-20 restraint substrate
//	   fields — FamilyDistress, CapacityLapse, FrailtyStepIncrease30d,
//	   RestrictivePracticeActive) AND the rule's type acknowledges them
//	   (STOP/MONITOR aligned with active restrictive practice).
//	3: restraint signaler ran but output ignored (signals present, rule type
//	   unrelated — e.g. DOSE_CHANGE on a resident with RestrictivePracticeActive).
//	1: restraint signaler not invoked (none of the kb-20 restraint substrate
//	   fields are populated — signals never reached the snapshot).
//
// The four kb-20 restraint substrate fields are the canonical "did the
// signaler run" tell. The user has confirmed they will load these from
// substrate; we do not defend against missing/zero values.
func (s *SubstrateBackedScorer) scoreRestraintConsidered(
	snap kb32ctx.ClinicalSnapshot, rule reasoning.ApplicableRule,
) int {
	signalerRan := snap.FamilyDistress ||
		snap.CapacityLapse ||
		snap.FrailtyStepIncrease30d ||
		snap.RestrictivePracticeActive

	if !signalerRan {
		return 1
	}

	// Signaler ran. Was the output considered by the rule?
	// Heuristic: STOP and MONITOR rules acting on a resident with an active
	// restrictive practice are the canonical pattern for "restraint output
	// considered" (recommend stopping the chemical restraint, or monitor it).
	if snap.RestrictivePracticeActive && (rule.Type == "STOP" || rule.Type == "MONITOR") {
		return 5
	}
	// Family-distress / capacity-lapse signals also align well with STOP.
	if rule.Type == "STOP" && (snap.FamilyDistress || snap.CapacityLapse) {
		return 5
	}
	// Signaler ran but the recommendation type does not engage with the
	// restraint signal — output produced but not acted on.
	return 3
}

// scoreGoalsOfCareAlignment — does the recommendation align with documented
// care intensity?
//
//	5: STOP/MONITOR aligned with comfort/palliative/end_of_life,
//	   OR ADD aligned with active treatment.
//	3: neutral (unrecognised care intensity, or DOSE_CHANGE on any intensity).
//	1: misaligned (ADD on palliative/end_of_life — the canonical anti-pattern
//	   from the plan; STOP on active when no other deprescribing signal exists
//	   is allowed at 3 because deprescribing is appropriate at any intensity).
func (s *SubstrateBackedScorer) scoreGoalsOfCareAlignment(
	rule reasoning.ApplicableRule, snap kb32ctx.ClinicalSnapshot,
) int {
	intensity := snap.CareIntensity
	switch rule.Type {
	case "ADD":
		if intensity == "palliative" || intensity == "end_of_life" {
			return 1 // canonical misalignment — drives the plan's gate-holds test
		}
		if intensity == "active" {
			return 5
		}
		// "comfort": ADD is suspect but not categorically wrong (e.g. analgesic).
		// Neutral.
		return 3
	case "STOP", "MONITOR":
		if intensity == "comfort" || intensity == "palliative" || intensity == "end_of_life" {
			return 5
		}
		// STOP/MONITOR on active intensity is normal deprescribing work — neutral.
		return 3
	default:
		// DOSE_CHANGE and any future types — cannot tell directionally. Neutral.
		return 3
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// isPsychotropicRule returns true when the RuleID naming convention marks the
// rule as a psychotropic-class rule. kb-cql-runtime authors use the prefixes
// "PSYCH-", "ANTIPSY-", or "BZD-" for these.
func isPsychotropicRule(ruleID string) bool {
	id := strings.ToUpper(ruleID)
	return strings.HasPrefix(id, "PSYCH-") ||
		strings.HasPrefix(id, "ANTIPSY-") ||
		strings.HasPrefix(id, "BZD-")
}

// hasAUPrefix returns true when the RuleID naming convention marks the rule as
// originating from an AU-jurisdiction guideline (ADG / RACGP / TGA).
func hasAUPrefix(ruleID string) bool {
	id := strings.ToUpper(ruleID)
	return strings.HasPrefix(id, "ADG-") ||
		strings.HasPrefix(id, "RACGP-") ||
		strings.HasPrefix(id, "TGA-") ||
		strings.HasPrefix(id, "AU-")
}

// countMarkers returns the number of distinct markers from needles that appear
// in haystack. Used to count AU vs international evidence anchors in the
// rendered evidence section.
func countMarkers(haystack string, needles []string) int {
	n := 0
	for _, m := range needles {
		if strings.Contains(haystack, m) {
			n++
		}
	}
	return n
}
