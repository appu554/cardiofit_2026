package aggregation

import (
	"context"
	"fmt"
	"sort"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// LifecycleState names the kb-32 lifecycle states surfaced in the S2
// pending-recommendations panel per S2 v1.0 Part 6.2.
type LifecycleState string

// Canonical lifecycle states. The set is exhaustive per v1.0 Part 6.2.
const (
	LifecycleDetected         LifecycleState = "detected"
	LifecycleDrafted          LifecycleState = "drafted"
	LifecycleSubmitted        LifecycleState = "submitted"
	LifecycleViewed           LifecycleState = "viewed"
	LifecycleDecided          LifecycleState = "decided"
	LifecycleMonitoringActive LifecycleState = "monitoring_active"
)

// ConfidenceDimensions is the two-axis confidence rendering per S2 v1.0
// Part 6.3 (substrate confidence + clinical confidence). Values are in
// [0.0, 1.0]. Phase 1 surfacing rule per Part 6.3: high-substrate-
// confidence recommendations are surfaced primarily; medium with
// "verify substrate" prompts; low typically suppressed at the renderer.
// The aggregator does NOT suppress here — surfacing policy is a UI-tier
// concern in Phase 1.
type ConfidenceDimensions struct {
	SubstrateConfidence float64
	ClinicalConfidence  float64
}

// PendingRecommendationCard is the per-card rendering unit for the S2
// pending-recommendations panel per v1.0 Part 6.1. Each card composes:
//   - kb-32 Packet content (Layer 1/2/3 framing body)
//   - kb-32 Assessment scores (5-dimension rubric)
//   - kb-32 Citation pins (fire-time source references)
//   - kb-32 Override history (decision log per recommendation)
//   - paired restraint signal (Phase 1 advisory-only)
//   - SubstrateRefs (verification-not-belief discipline)
//
// HoldReason is non-empty when the appropriateness gate held the
// recommendation in `detected` state — surfaced to the pharmacist per
// v1.0 Part 6.2.
type PendingRecommendationCard struct {
	RecommendationID     uuid.UUID
	Type                 string
	Urgency              string
	Layer1Body           string
	Layer2Body           string
	Layer3Body           string
	Lifecycle            LifecycleState
	Confidence           ConfidenceDimensions
	PairedRestraintSignal *substrate_types.RestraintSignal
	HoldReason           string
	Citations            []substrate_types.Citation
	OverrideHistory      []substrate_types.OverrideReason
	SubstrateRefs        []SubstrateRef
}

// typeRank mirrors kb-32 ordering.typeRank (kb-32/internal/ordering/orderer.go).
// Lower values sort earlier: STOP > MONITOR > DOSE_CHANGE > ADD.
var typeRank = map[string]int{
	"STOP":        0,
	"MONITOR":     1,
	"DOSE_CHANGE": 2,
	"ADD":         3,
}

// urgencyRank orders urgency tiers per v1.0 Part 6.1: red > amber > green.
var urgencyRank = map[string]int{
	"red":   0,
	"amber": 1,
	"green": 2,
}

// BuildPendingRecommendationCards composes the per-resident pending-
// recommendation panel per S2 v1.0 Part 6. Each kb-32 Packet on this
// resident is enriched with assessment scores, citation pins, override
// history, and any paired restraint signal.
//
// Empty-state per v1.0 Part 6.5: when no packets are pending, the
// returned slice is a non-nil empty slice — callers must distinguish
// "no data fetched" (nil/error) from "fetched, zero" (empty slice).
//
// Sort order matches kb-32 ordering.Order: STOP > MONITOR > DOSE_CHANGE
// > ADD by type, then urgency (red > amber > green), then by
// RecommendationID for stability.
//
// Phase 1 commitment: clinical content (Layer 1/2/3 body framing) comes
// from kb-32; the aggregator does NOT author clinical text here.
func BuildPendingRecommendationCards(
	ctx context.Context,
	client SubstrateClient,
	residentID uuid.UUID,
	asOf time.Time,
) ([]PendingRecommendationCard, error) {
	if client == nil {
		return nil, fmt.Errorf("BuildPendingRecommendationCards: nil SubstrateClient")
	}
	_ = asOf // reserved for time-travel queries in Task 8

	packets, err := client.PendingRecommendations(ctx, residentID)
	if err != nil {
		return nil, fmt.Errorf("PendingRecommendations: %w", err)
	}

	// Fetch active restraint signals once so we can pair them in-loop.
	signals, err := client.ActiveRestraintSignals(ctx, residentID)
	if err != nil {
		return nil, fmt.Errorf("ActiveRestraintSignals: %w", err)
	}
	signalByRec := make(map[uuid.UUID]*substrate_types.RestraintSignal, len(signals))
	for i := range signals {
		s := signals[i]
		if s.PairedRecommendationID != uuid.Nil {
			signalByRec[s.PairedRecommendationID] = &s
		}
	}

	// Empty-state per v1.0 Part 6.5: non-nil empty slice.
	out := make([]PendingRecommendationCard, 0, len(packets))

	for _, pkt := range packets {
		card, err := buildOneCard(ctx, client, pkt, signalByRec[pkt.RecommendationID])
		if err != nil {
			return nil, err
		}
		out = append(out, card)
	}

	sortCards(out)
	return out, nil
}

// buildOneCard assembles a single PendingRecommendationCard from a packet
// + paired restraint signal. Fetches assessment, citations, and override
// history via the SubstrateClient.
func buildOneCard(
	ctx context.Context,
	client SubstrateClient,
	pkt substrate_types.RecommendationPacket,
	pairedSignal *substrate_types.RestraintSignal,
) (PendingRecommendationCard, error) {
	card := PendingRecommendationCard{
		RecommendationID: pkt.RecommendationID,
		Type:             pkt.Type,
		Urgency:          pkt.AppliedRule.Urgency,
		// Layer 1/2/3 body text comes from kb-32 Sections. The keys below
		// match the v1.0 framing-body convention; if kb-32 has not yet
		// populated a key (Tasks 7/10/12 of kb-32 fill them), the value
		// is empty and the renderer must handle that gracefully.
		Layer1Body: pkt.Sections["layer_1"],
		Layer2Body: pkt.Sections["layer_2"],
		Layer3Body: pkt.Sections["layer_3"],
		// Default lifecycle: detected. kb-32 Packet does not carry the
		// lifecycle column directly (that lives in the persistence
		// layer); the SubstrateClient adapter (Task 8) will populate
		// this. Fake clients leave it default-detected.
		Lifecycle:             LifecycleDetected,
		PairedRestraintSignal: pairedSignal,
	}

	assessment, err := client.RecommendationAssessment(ctx, pkt.RecommendationID)
	if err != nil {
		return card, fmt.Errorf("RecommendationAssessment(%s): %w", pkt.RecommendationID, err)
	}
	card.Confidence = confidenceFromAssessment(assessment)
	card.HoldReason = holdReasonFromAssessment(assessment)

	cits, err := client.RecommendationCitations(ctx, pkt.RecommendationID)
	if err != nil {
		return card, fmt.Errorf("RecommendationCitations(%s): %w", pkt.RecommendationID, err)
	}
	card.Citations = cits

	ors, err := client.RecommendationOverrides(ctx, pkt.RecommendationID)
	if err != nil {
		return card, fmt.Errorf("RecommendationOverrides(%s): %w", pkt.RecommendationID, err)
	}
	card.OverrideHistory = ors

	card.SubstrateRefs = buildCardSubstrateRefs(pkt, cits, pairedSignal)

	return card, nil
}

// confidenceFromAssessment derives the two-axis ConfidenceDimensions from
// the 5-dimension assessment per v1.0 Part 6.3 + Style Guide Part 8.
//
// Mapping (Phase 1 conservative placeholder, pending senior pharmacist
// authoring per S2 Addendum Part 6.1):
//   - SubstrateConfidence: EvidenceSolidity / 5.0
//   - ClinicalConfidence:  mean(ClinicalWarrant, AlternativesConsidered,
//                                RestraintConsidered, GoalsOfCareAlignment) / 5.0
//
// Zero-valued AssessmentScores (no assessment on record) produce zero
// confidence — the renderer should surface "assessment pending" rather
// than treat as low-confidence-surfaced-anyway.
func confidenceFromAssessment(a substrate_types.AssessmentScores) ConfidenceDimensions {
	if a == (substrate_types.AssessmentScores{}) {
		return ConfidenceDimensions{}
	}
	clinSum := a.ClinicalWarrant + a.AlternativesConsidered + a.RestraintConsidered + a.GoalsOfCareAlignment
	return ConfidenceDimensions{
		SubstrateConfidence: float64(a.EvidenceSolidity) / 5.0,
		ClinicalConfidence:  float64(clinSum) / (4.0 * 5.0),
	}
}

// holdReasonFromAssessment mirrors kb-32 appropriateness.Check: any
// dimension ≤2 holds the recommendation in `detected`. Returns a short
// human-readable reason naming the first failing dimension; empty when
// no dimension is in hold range or when no assessment is on record.
func holdReasonFromAssessment(a substrate_types.AssessmentScores) string {
	if a == (substrate_types.AssessmentScores{}) {
		return ""
	}
	const holdThreshold = 2
	dims := []struct {
		name  string
		score int
	}{
		{"clinical_warrant", a.ClinicalWarrant},
		{"evidence_solidity", a.EvidenceSolidity},
		{"alternatives_considered", a.AlternativesConsidered},
		{"restraint_considered", a.RestraintConsidered},
		{"goals_of_care_alignment", a.GoalsOfCareAlignment},
	}
	for _, d := range dims {
		if d.score >= 1 && d.score <= holdThreshold {
			return fmt.Sprintf("appropriateness hold: %s score %d ≤ threshold %d", d.name, d.score, holdThreshold)
		}
	}
	return ""
}

// buildCardSubstrateRefs assembles the verification-not-belief refs
// for the card: the recommendation itself + each citation pin + the
// paired restraint signal (if any). Each card carries ≥1 ref.
func buildCardSubstrateRefs(
	pkt substrate_types.RecommendationPacket,
	cits []substrate_types.Citation,
	paired *substrate_types.RestraintSignal,
) []SubstrateRef {
	refs := []SubstrateRef{{
		Source:      "kb-32",
		ID:          pkt.RecommendationID,
		Description: fmt.Sprintf("%s recommendation (rule %s)", pkt.Type, pkt.AppliedRule.RuleID),
	}}
	for _, c := range cits {
		refs = append(refs, SubstrateRef{
			Source:      "kb-32-citation",
			ID:          pkt.RecommendationID,
			Description: fmt.Sprintf("citation %s@%s pinned %s", c.SourceID, c.Version, c.PinnedAt.Format("2006-01-02")),
		})
	}
	if paired != nil {
		refs = append(refs, SubstrateRef{
			Source:      "kb-32-restraint",
			ID:          paired.SignalID,
			Description: fmt.Sprintf("paired restraint signal %s (severity %d)", paired.Type, paired.Severity),
		})
	}
	return refs
}

// sortCards applies the v1.0 Part 6.1 + kb-32 ordering.Order priority:
// STOP > MONITOR > DOSE_CHANGE > ADD, then red > amber > green, then by
// RecommendationID for stability. Unknown type/urgency sorts to the end.
func sortCards(cards []PendingRecommendationCard) {
	sort.SliceStable(cards, func(i, j int) bool {
		ri, oki := typeRank[cards[i].Type]
		if !oki {
			ri = len(typeRank)
		}
		rj, okj := typeRank[cards[j].Type]
		if !okj {
			rj = len(typeRank)
		}
		if ri != rj {
			return ri < rj
		}
		ui, oku := urgencyRank[cards[i].Urgency]
		if !oku {
			ui = len(urgencyRank)
		}
		uj, oku := urgencyRank[cards[j].Urgency]
		if !oku {
			uj = len(urgencyRank)
		}
		if ui != uj {
			return ui < uj
		}
		return cards[i].RecommendationID.String() < cards[j].RecommendationID.String()
	})
}
