// drill_through_composition_test.go — v1.0 Part 17 Category 5
// (drill-through). Cross-cutting tests for trajectory→observation,
// recommendation→citation, negative-evidence epistemic-humility
// framing, and substrate-confidence visibility.
package structural

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/drill_through"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// TestDrillThrough_FromTrajectory_ResolvesObservationSeries — click on
// a trajectory → GetTrajectoryHistory returns the observation series
// for that parameter.
func TestDrillThrough_FromTrajectory_ResolvesObservationSeries(t *testing.T) {
	rid := uuid.New()
	asOf := time.Date(2026, 5, 11, 0, 0, 0, 0, time.UTC)

	obs := []substrate_types.Observation{
		mkObsV9(rid, "egfr", 55, asOf.AddDate(-1, 0, 0)),
		mkObsV9(rid, "egfr", 45, asOf.AddDate(0, -6, 0)),
		mkObsV9(rid, "egfr", 38, asOf.AddDate(0, -1, 0)),
	}
	client := aggregation.NewInMemorySubstrateClient().WithObservations(obs...)

	hist, err := drill_through.GetTrajectoryHistory(context.Background(), client, rid, "egfr")
	if err != nil {
		t.Fatalf("GetTrajectoryHistory: %v", err)
	}
	if len(hist.Observations) != 3 {
		t.Errorf("expected 3 observations; got %d", len(hist.Observations))
	}
	if hist.Parameter != "egfr" {
		t.Errorf("Parameter mismatch: %q", hist.Parameter)
	}
	// Chronological order (oldest first).
	for i := 1; i < len(hist.Observations); i++ {
		if hist.Observations[i-1].ObservedAt.After(hist.Observations[i].ObservedAt) {
			t.Errorf("observation series not chronological at index %d", i)
		}
	}
}

// TestDrillThrough_FromRecommendationCitation_ResolvesCitation — click
// on a citation in a card → returns the SourceVersion + pinned
// RecommendationCitation. We assert via BuildPendingRecommendationCards
// that the citation is surfaced on the card (the drill-through itself
// is a frontend operation; here we verify the substrate is present).
func TestDrillThrough_FromRecommendationCitation_ResolvesCitation(t *testing.T) {
	rid := uuid.New()
	asOf := time.Now()
	pkt := mkPktV9(rid, "STOP", "red")
	cit := substrate_types.Citation{
		RecommendationID: pkt.RecommendationID.String(),
		SourceID:         "AMH-2025",
		Version:          "1.0.0",
		PinnedAt:         asOf.AddDate(0, -1, 0),
	}
	client := aggregation.NewInMemorySubstrateClient().
		WithPackets(pkt).
		WithCitations(pkt.RecommendationID, cit)

	cards, err := aggregation.BuildPendingRecommendationCards(context.Background(), client, rid, asOf)
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected 1 card; got %d", len(cards))
	}
	if len(cards[0].Citations) != 1 {
		t.Fatalf("expected 1 citation on card; got %d", len(cards[0].Citations))
	}
	got := cards[0].Citations[0]
	if got.SourceID != "AMH-2025" || got.Version != "1.0.0" {
		t.Errorf("citation drift: got %+v", got)
	}
}

// TestDrillThrough_NegativeEvidence_EpistemicHumilityFraming — negative
// evidence rendering uses "we searched for X; found nothing matching"
// not "X is absent" per v1.0 Part 10.4.
func TestDrillThrough_NegativeEvidence_EpistemicHumilityFraming(t *testing.T) {
	rendering := drill_through.RenderNegativeEvidence(drill_through.NegativeEvidenceSearch{
		Claim:             "no current indication for omeprazole",
		SearchedSources:   []string{"eNRMC indication field", "progress notes (24mo)"},
		UnsearchedSources: []string{"scanned discharge summaries"},
		SearchedAt:        time.Now(),
		Confidence:        "moderate",
	})

	// Framing rule: "in available records" should appear in the
	// Statement to preserve epistemic humility.
	if !strings.Contains(rendering.Statement, "available records") {
		t.Errorf("Statement should preserve epistemic humility ('in available records'); got %q", rendering.Statement)
	}
	// Statement must NOT use absolute-absence framing.
	if strings.Contains(strings.ToLower(rendering.Statement), "is absent") {
		t.Errorf("Statement uses absolute-absence framing — violates v1.0 Part 10.4: %q", rendering.Statement)
	}
	// Evidence lines must use "searched, no matching evidence" framing.
	for _, line := range rendering.EvidenceLines {
		if !strings.Contains(line, "searched") {
			t.Errorf("evidence line should describe a SEARCH, not an absence: %q", line)
		}
	}
	// Caveat must surface unsearched sources.
	if rendering.Caveat == "" {
		t.Error("Caveat should be populated when UnsearchedSources is non-empty")
	}
	if !strings.Contains(rendering.Caveat, "scanned discharge summaries") {
		t.Errorf("Caveat should name unsearched sources; got %q", rendering.Caveat)
	}
}

// TestDrillThrough_NegativeEvidence_NoUnsearched_NoCaveat — when no
// unsearched sources are supplied, the Caveat is empty (we don't
// invent humility theatre).
func TestDrillThrough_NegativeEvidence_NoUnsearched_NoCaveat(t *testing.T) {
	rendering := drill_through.RenderNegativeEvidence(drill_through.NegativeEvidenceSearch{
		Claim:           "no documented penicillin allergy",
		SearchedSources: []string{"allergy module"},
	})
	if rendering.Caveat != "" {
		t.Errorf("Caveat should be empty when no unsearched sources; got %q", rendering.Caveat)
	}
}

// TestDrillThrough_SubstrateConfidenceVisible — every drill-through
// result carries the substrate confidence value per v1.0 Part 10.3.
func TestDrillThrough_SubstrateConfidenceVisible(t *testing.T) {
	rid := uuid.New()
	obsID := uuid.New()
	asOf := time.Now()
	obs := substrate_types.Observation{
		ID:         obsID,
		ResidentID: rid,
		Parameter:  "egfr",
		Value:      40,
		Unit:       "mL/min/1.73m²",
		ObservedAt: asOf.AddDate(0, -1, 0),
		Source:     "kb-20",
		Confidence: "moderate",
	}
	fetcher := &fixedFetcher{obs: obs}
	ref := aggregation.SubstrateRef{Source: "kb-20", ID: obsID, Description: "egfr=40"}
	so, err := drill_through.GetSubstrateObservation(context.Background(), fetcher, ref, nil)
	if err != nil {
		t.Fatalf("GetSubstrateObservation: %v", err)
	}
	if so.SubstrateConfidence != "moderate" {
		t.Errorf("SubstrateConfidence should be 'moderate'; got %q", so.SubstrateConfidence)
	}
	if so.Observation.ID != obsID {
		t.Error("Observation ID mismatch in drill-through result")
	}
}

// TestDrillThrough_BackTrail_Preserved — the back-trail of claims is
// preserved verbatim through the drill-through call so the renderer
// can show the "one-click-back-to-claim" path per v1.0 Part 10.1.
func TestDrillThrough_BackTrail_Preserved(t *testing.T) {
	rid := uuid.New()
	obsID := uuid.New()
	obs := substrate_types.Observation{
		ID: obsID, ResidentID: rid, Parameter: "weight", Value: 65, Unit: "kg",
		ObservedAt: time.Now(), Source: "kb-20", Confidence: "high",
	}
	fetcher := &fixedFetcher{obs: obs}
	backTrail := []aggregation.SubstrateRef{
		{Source: "trajectory", ID: rid, Description: "weight trajectory"},
		{Source: "panel", ID: rid, Description: "trajectories panel"},
	}
	ref := aggregation.SubstrateRef{Source: "kb-20", ID: obsID, Description: "weight=65"}
	so, err := drill_through.GetSubstrateObservation(context.Background(), fetcher, ref, backTrail)
	if err != nil {
		t.Fatalf("GetSubstrateObservation: %v", err)
	}
	if len(so.ClaimBackTrail) != 2 {
		t.Errorf("ClaimBackTrail length: got %d want 2", len(so.ClaimBackTrail))
	}
}

// fixedFetcher is an inline ObservationFetcher returning a fixed obs.
type fixedFetcher struct{ obs substrate_types.Observation }

func (f *fixedFetcher) GetObservationByID(_ context.Context, _ uuid.UUID) (substrate_types.Observation, error) {
	return f.obs, nil
}
