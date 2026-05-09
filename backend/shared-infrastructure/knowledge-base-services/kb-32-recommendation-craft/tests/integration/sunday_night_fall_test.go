// Package integration provides end-to-end tests for the kb-32 recommendation
// craft engine pipeline.
//
// The Sunday-night-fall scenario validates the complete six-stage pipeline
// with in-memory fakes for the substrate client and HAPI client, exercising
// the path from a resident with RecentFall72h=true through to a drafted
// recommendation with a non-empty ContentHash.
package integration

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/api"
	"github.com/cardiofit/kb32/internal/appropriateness"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/reasoning"
)

// ---------------------------------------------------------------------------
// In-memory fakes
// ---------------------------------------------------------------------------

// inMemorySubstrateClient returns a fixed ClinicalSnapshot keyed by residentID.
type inMemorySubstrateClient struct {
	snapshots map[uuid.UUID]kb32ctx.ClinicalSnapshot
}

func (c *inMemorySubstrateClient) SnapshotFor(_ context.Context, residentID uuid.UUID) (kb32ctx.ClinicalSnapshot, error) {
	if snap, ok := c.snapshots[residentID]; ok {
		return snap, nil
	}
	// Return a minimal passing snapshot for unregistered residents.
	return kb32ctx.ClinicalSnapshot{
		ResidentID:    residentID,
		CareIntensity: "active",
		AssessedAt:    time.Now(),
	}, nil
}

// stubHAPIClient returns a fixed EvaluateRuleResult for a given ruleID.
type stubHAPIClient struct {
	rules map[string]*reasoning.EvaluateRuleResult
}

func (c *stubHAPIClient) EvaluateRule(_ context.Context, ruleID string, _ uuid.UUID) (*reasoning.EvaluateRuleResult, error) {
	if result, ok := c.rules[ruleID]; ok {
		res := *result
		res.RuleID = ruleID
		return &res, nil
	}
	return &reasoning.EvaluateRuleResult{RuleID: ruleID, Triggered: false}, nil
}

// defaultAppSrc delegates to api.DefaultAppropriatenessSource.
type defaultAppSrc struct{}

func (defaultAppSrc) Assess(_ context.Context, _ *generator.Packet,
	_ kb32ctx.ClinicalSnapshot, _ reasoning.ApplicableRule) (appropriateness.Assessment, error) {
	return api.DefaultAppropriatenessSource{}.Assess(context.Background(), nil, kb32ctx.ClinicalSnapshot{}, reasoning.ApplicableRule{})
}

// ---------------------------------------------------------------------------
// Sunday-night-fall E2E test
// ---------------------------------------------------------------------------

// TestSundayNightFall_E2E exercises the complete six-stage pipeline with a
// synthetic resident who has had a fall within 72 hours.
//
// Preconditions:
//   - Resident has RecentFall72h=true
//   - The "PostFall" CQL rule triggers with Type="MONITOR", Urgency="red"
//   - DefaultAppropriatenessSource returns passing scores (3s across all dims)
//
// Assertions:
//   - response.State == "drafted"
//   - response.ContentHash non-empty
//   - Layer 1 signal mentions "fall" (from the generated issue section)
//   - appropriateness scoring (default 3s) passes the gate
//   - urgency tagger derives "red" from RecentFall72h=true
//   - orderer + urgency tagger output match (urgency = red)
func TestSundayNightFall_E2E(t *testing.T) {
	residentID := uuid.New()
	authorID := uuid.New()

	// Synthetic resident: RecentFall72h=true, active care intensity.
	snap := kb32ctx.ClinicalSnapshot{
		ResidentID:    residentID,
		EGFR:          55.0,
		DBI:           0.3,
		ACB:           1,
		CFS:           5,
		CareIntensity: "active",
		RecentFall72h: true,
		AssessedAt:    time.Now(),
	}

	substrateClient := &inMemorySubstrateClient{
		snapshots: map[uuid.UUID]kb32ctx.ClinicalSnapshot{
			residentID: snap,
		},
	}

	// Stub HAPI: "PostFall" rule fires with MONITOR type and red urgency.
	hapiClient := &stubHAPIClient{
		rules: map[string]*reasoning.EvaluateRuleResult{
			"PostFall": {
				Triggered: true,
				Type:      "MONITOR",
				Urgency:   "red",
			},
		},
	}

	assembler := kb32ctx.NewAssembler(substrateClient)
	chain := reasoning.NewChainBuilder(hapiClient)
	appSrc := &defaultAppSrc{}

	pipeline := api.NewPipeline(assembler, chain, appSrc, nil)

	// Run pipeline.
	result, err := pipeline.Run(context.Background(), "PostFall", residentID, authorID)
	if err != nil {
		t.Fatalf("pipeline.Run error: %v", err)
	}

	// ---- Assertion 1: State should be "drafted" (gate passes) ----
	if result.HoldReason != "" {
		t.Errorf("expected gate to pass (HoldReason empty); got HoldReason=%q", result.HoldReason)
	}

	// ---- Assertion 2: ContentHash non-empty ----
	if result.ContentHash == "" {
		t.Error("expected non-empty ContentHash for drafted recommendation")
	}

	// ---- Assertion 3: Layer 1 signal mentions "fall" ----
	l1Lower := strings.ToLower(result.LayerOutput.L1Signal)
	if !strings.Contains(l1Lower, "fall") {
		t.Errorf("Layer 1 signal %q does not mention 'fall'", result.LayerOutput.L1Signal)
	}

	// ---- Assertion 4: Appropriateness passes (all dims at 3 via DefaultScorer) ----
	if err := appropriateness.Check(result.Assessment); err != nil {
		t.Errorf("appropriateness gate should pass; got %v", err)
	}
	for i, score := range []int{
		result.Assessment.ClinicalWarrant,
		result.Assessment.EvidenceSolidity,
		result.Assessment.AlternativesConsidered,
		result.Assessment.RestraintConsidered,
		result.Assessment.GoalsOfCareAlignment,
	} {
		if score <= appropriateness.HoldThreshold {
			t.Errorf("assessment dimension %d score %d ≤ HoldThreshold %d", i, score, appropriateness.HoldThreshold)
		}
	}

	// ---- Assertion 5: UrgencyTag is "red" (RecentFall72h=true) ----
	if result.UrgencyTag != "red" {
		t.Errorf("UrgencyTag = %q; want red (RecentFall72h=true)", result.UrgencyTag)
	}

	// ---- Assertion 6: Packet type matches applied rule (MONITOR) ----
	if result.Packet == nil {
		t.Fatal("Packet is nil")
	}
	if result.Packet.Type != "MONITOR" {
		t.Errorf("Packet.Type = %q; want MONITOR", result.Packet.Type)
	}
	if result.Packet.AppliedRule.RuleID != "PostFall" {
		t.Errorf("Packet.AppliedRule.RuleID = %q; want PostFall", result.Packet.AppliedRule.RuleID)
	}

	// ---- Assertion 7: Packet ResidentID matches ----
	if result.Packet.SnapshotRef != residentID {
		t.Errorf("Packet.SnapshotRef = %v; want %v", result.Packet.SnapshotRef, residentID)
	}

	// ---- Assertion 8: ContentHash is 64-character hex (SHA-256) ----
	if len(result.ContentHash) != 64 {
		t.Errorf("ContentHash length = %d; want 64", len(result.ContentHash))
	}
	for _, c := range result.ContentHash {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("ContentHash contains non-hex character %q", c)
			break
		}
	}

	// ---- Assertion 9: Layer 1 within word budget ----
	if wc := len(strings.Fields(result.LayerOutput.L1Signal)); wc > 25 {
		t.Errorf("Layer 1 word count = %d; want ≤ 25", wc)
	}

	// ---- Assertion 10: Layer 2 within word budget ----
	if wc := len(strings.Fields(result.LayerOutput.L2Reasoning)); wc > 100 {
		t.Errorf("Layer 2 word count = %d; want ≤ 100", wc)
	}

	// ---- Assertion 11: L3Provenance contains "PostFall" ----
	found := false
	for _, ref := range result.LayerOutput.L3Provenance {
		if ref == "PostFall" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("L3Provenance %v does not contain PostFall rule reference", result.LayerOutput.L3Provenance)
	}

	t.Logf("Sunday-night-fall E2E passed:")
	t.Logf("  RecommendationID: %s", result.Packet.RecommendationID)
	t.Logf("  State: drafted")
	t.Logf("  ContentHash: %s", result.ContentHash)
	t.Logf("  UrgencyTag: %s", result.UrgencyTag)
	t.Logf("  L1Signal: %s", result.LayerOutput.L1Signal)
	t.Logf("  L3Provenance: %v", result.LayerOutput.L3Provenance)
}

// TestSundayNightFall_AppropriatenessHold validates that a below-threshold
// assessment correctly holds the recommendation in detected state.
func TestSundayNightFall_AppropriatenessHold(t *testing.T) {
	residentID := uuid.New()
	authorID := uuid.New()

	snap := kb32ctx.ClinicalSnapshot{
		ResidentID:    residentID,
		CareIntensity: "active",
		RecentFall72h: true,
		AssessedAt:    time.Now(),
	}

	substrateClient := &inMemorySubstrateClient{
		snapshots: map[uuid.UUID]kb32ctx.ClinicalSnapshot{residentID: snap},
	}
	hapiClient := &stubHAPIClient{
		rules: map[string]*reasoning.EvaluateRuleResult{
			"PostFall": {Triggered: true, Type: "MONITOR", Urgency: "red"},
		},
	}

	// Use a hold-triggering AppropriatenessSource.
	holdSrc := &holdingAppSrc{}
	assembler := kb32ctx.NewAssembler(substrateClient)
	chain := reasoning.NewChainBuilder(hapiClient)
	pipeline := api.NewPipeline(assembler, chain, holdSrc, nil)

	result, err := pipeline.Run(context.Background(), "PostFall", residentID, authorID)
	if err != nil {
		t.Fatalf("pipeline.Run error: %v", err)
	}

	if result.HoldReason == "" {
		t.Error("expected HoldReason to be set when gate holds")
	}
	if result.ContentHash != "" {
		t.Errorf("ContentHash should be empty when gate holds; got %q", result.ContentHash)
	}

	t.Logf("Hold scenario: HoldReason=%q", result.HoldReason)
}

// holdingAppSrc returns an assessment with all dimensions at 1 (holds the gate).
type holdingAppSrc struct{}

func (holdingAppSrc) Assess(_ context.Context, _ *generator.Packet,
	_ kb32ctx.ClinicalSnapshot, _ reasoning.ApplicableRule) (appropriateness.Assessment, error) {
	return appropriateness.Assessment{
		ClinicalWarrant:        1,
		EvidenceSolidity:       1,
		AlternativesConsidered: 1,
		RestraintConsidered:    1,
		GoalsOfCareAlignment:   1,
	}, nil
}
