package appropriateness_test

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/api"
	"github.com/cardiofit/kb32/internal/appropriateness"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/generator"
	"github.com/cardiofit/kb32/internal/reasoning"
)

// Compile-time conformance: SubstrateBackedScorer satisfies the
// api.AppropriatenessSource interface. Lives in package appropriateness_test
// to avoid an api ↔ appropriateness import cycle (api imports appropriateness;
// appropriateness cannot import api).
var _ api.AppropriatenessSource = (*appropriateness.SubstrateBackedScorer)(nil)

// makePacket is a tiny helper that builds a Packet via the real generator so
// the test exercises the same construction path the pipeline uses.
func makePacket(t *testing.T, snap kb32ctx.ClinicalSnapshot, rule reasoning.ApplicableRule) *generator.Packet {
	t.Helper()
	pkt, err := generator.Generate(snap, []reasoning.ApplicableRule{rule}, uuid.New())
	if err != nil {
		t.Fatalf("generator.Generate: %v", err)
	}
	return pkt
}

// ---------------------------------------------------------------------------
// Per-dimension table tests
// ---------------------------------------------------------------------------

func TestScorer_ClinicalWarrant(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	cases := []struct {
		name string
		snap kb32ctx.ClinicalSnapshot
		rule reasoning.ApplicableRule
		want int
	}{
		{
			name: "STOP with ACB>=3 scores 5",
			snap: kb32ctx.ClinicalSnapshot{ACB: 4, CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "STOPP-A1", Type: "STOP", Urgency: "ROUTINE"},
			want: 5,
		},
		{
			name: "STOP psychotropic on freshly-admitted clean substrate scores 1 (contraindicated)",
			snap: kb32ctx.ClinicalSnapshot{RecentAdmission72h: true, ACB: 0, DBI: 0, CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "PSYCH-001", Type: "STOP", Urgency: "ROUTINE"},
			want: 1,
		},
		{
			name: "ADD aggressive on end_of_life scores 1 (contraindicated)",
			snap: kb32ctx.ClinicalSnapshot{CareIntensity: "end_of_life"},
			rule: reasoning.ApplicableRule{RuleID: "ADD-OXY", Type: "ADD", Urgency: "ROUTINE"},
			want: 1,
		},
		{
			name: "STOP without substrate signal scores 3 (neutral default)",
			snap: kb32ctx.ClinicalSnapshot{CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "STOPP-Q1", Type: "STOP", Urgency: "ROUTINE"},
			want: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkt := makePacket(t, tc.snap, tc.rule)
			got, err := scorer.Assess(context.Background(), pkt, tc.snap, tc.rule)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			if got.ClinicalWarrant != tc.want {
				t.Errorf("ClinicalWarrant = %d, want %d", got.ClinicalWarrant, tc.want)
			}
		})
	}
}

func TestScorer_EvidenceSolidity(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	cases := []struct {
		name        string
		evidenceTxt string
		ruleID      string
		want        int
	}{
		{
			name:        "two AU markers scores 5",
			evidenceTxt: "Based on ADG-2025-AU and RACGP-2024 guidance.",
			ruleID:      "ADG-001",
			want:        5,
		},
		{
			name:        "one AU marker in evidence (non-AU ruleID) scores 3",
			evidenceTxt: "Per TGA-2025 guidance.",
			ruleID:      "GEN-001",
			want:        3,
		},
		{
			name:        "two international markers scores 3",
			evidenceTxt: "Refs: NICE-NG203 and KDIGO-2024.",
			ruleID:      "GEN-001",
			want:        3,
		},
		{
			name:        "retracted marker scores 1",
			evidenceTxt: "Source RETRACTED in 2024.",
			ruleID:      "GEN-001",
			want:        1,
		},
		{
			name:        "no anchors discernible scores 3 (neutral default)",
			evidenceTxt: "N/A",
			ruleID:      "GEN-001",
			want:        3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := kb32ctx.ClinicalSnapshot{CareIntensity: "active"}
			rule := reasoning.ApplicableRule{RuleID: tc.ruleID, Type: "MONITOR", Urgency: "ROUTINE"}
			pkt := makePacket(t, snap, rule)
			pkt.Sections["evidence"] = tc.evidenceTxt
			got, err := scorer.Assess(context.Background(), pkt, snap, rule)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			if got.EvidenceSolidity != tc.want {
				t.Errorf("EvidenceSolidity = %d, want %d", got.EvidenceSolidity, tc.want)
			}
		})
	}
}

func TestScorer_AlternativesConsidered(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	cases := []struct {
		name string
		rule reasoning.ApplicableRule
		want int
	}{
		{
			name: "rule with -ALT suffix scores 5",
			rule: reasoning.ApplicableRule{RuleID: "STOPP-PPI-ALT", Type: "STOP", Urgency: "ROUTINE"},
			want: 5,
		},
		{
			name: "rule with ALT-PENDING scores 3",
			rule: reasoning.ApplicableRule{RuleID: "STOPP-X-ALT-PENDING", Type: "STOP", Urgency: "ROUTINE"},
			want: 3,
		},
		{
			name: "ADD without alt metadata scores 1",
			rule: reasoning.ApplicableRule{RuleID: "ADD-OXY", Type: "ADD", Urgency: "ROUTINE"},
			want: 1,
		},
		{
			name: "STOP without alt metadata scores 3 (neutral)",
			rule: reasoning.ApplicableRule{RuleID: "STOPP-Q1", Type: "STOP", Urgency: "ROUTINE"},
			want: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			snap := kb32ctx.ClinicalSnapshot{CareIntensity: "active"}
			pkt := makePacket(t, snap, tc.rule)
			got, err := scorer.Assess(context.Background(), pkt, snap, tc.rule)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			if got.AlternativesConsidered != tc.want {
				t.Errorf("AlternativesConsidered = %d, want %d", got.AlternativesConsidered, tc.want)
			}
		})
	}
}

func TestScorer_RestraintConsidered(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	cases := []struct {
		name string
		snap kb32ctx.ClinicalSnapshot
		rule reasoning.ApplicableRule
		want int
	}{
		{
			name: "RestrictivePracticeActive + STOP scores 5 (output considered)",
			snap: kb32ctx.ClinicalSnapshot{RestrictivePracticeActive: true, CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "STOPP-Q1", Type: "STOP", Urgency: "ROUTINE"},
			want: 5,
		},
		{
			name: "signaler ran but DOSE_CHANGE ignores it scores 3",
			snap: kb32ctx.ClinicalSnapshot{RestrictivePracticeActive: true, CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "DOSE-A", Type: "DOSE_CHANGE", Urgency: "ROUTINE"},
			want: 3,
		},
		{
			name: "signaler did not run scores 1",
			snap: kb32ctx.ClinicalSnapshot{CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "STOPP-Q1", Type: "STOP", Urgency: "ROUTINE"},
			want: 1,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkt := makePacket(t, tc.snap, tc.rule)
			got, err := scorer.Assess(context.Background(), pkt, tc.snap, tc.rule)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			if got.RestraintConsidered != tc.want {
				t.Errorf("RestraintConsidered = %d, want %d", got.RestraintConsidered, tc.want)
			}
		})
	}
}

func TestScorer_GoalsOfCareAlignment(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	cases := []struct {
		name string
		snap kb32ctx.ClinicalSnapshot
		rule reasoning.ApplicableRule
		want int
	}{
		{
			name: "STOP on palliative scores 5",
			snap: kb32ctx.ClinicalSnapshot{CareIntensity: "palliative"},
			rule: reasoning.ApplicableRule{RuleID: "STOPP-Q1", Type: "STOP", Urgency: "ROUTINE"},
			want: 5,
		},
		{
			name: "ADD on active scores 5",
			snap: kb32ctx.ClinicalSnapshot{CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "ADD-X", Type: "ADD", Urgency: "ROUTINE"},
			want: 5,
		},
		{
			name: "ADD on end_of_life scores 1 (canonical misalignment)",
			snap: kb32ctx.ClinicalSnapshot{CareIntensity: "end_of_life"},
			rule: reasoning.ApplicableRule{RuleID: "ADD-X", Type: "ADD", Urgency: "ROUTINE"},
			want: 1,
		},
		{
			name: "DOSE_CHANGE on any intensity scores 3 (neutral default)",
			snap: kb32ctx.ClinicalSnapshot{CareIntensity: "active"},
			rule: reasoning.ApplicableRule{RuleID: "DOSE-A", Type: "DOSE_CHANGE", Urgency: "ROUTINE"},
			want: 3,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			pkt := makePacket(t, tc.snap, tc.rule)
			got, err := scorer.Assess(context.Background(), pkt, tc.snap, tc.rule)
			if err != nil {
				t.Fatalf("Assess: %v", err)
			}
			if got.GoalsOfCareAlignment != tc.want {
				t.Errorf("GoalsOfCareAlignment = %d, want %d", got.GoalsOfCareAlignment, tc.want)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Integration: the plan's required gate scenario
// ---------------------------------------------------------------------------

// TestScorer_EndOfLifeAddRecommendation_GateHolds asserts the plan's explicit
// integration scenario: an end-of-life resident receiving an ADD recommendation
// scores ≤ 2 on GoalsOfCareAlignment and the gate (Check) holds.
func TestScorer_EndOfLifeAddRecommendation_GateHolds(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	snap := kb32ctx.ClinicalSnapshot{
		ResidentID:    uuid.New(),
		CareIntensity: "end_of_life",
	}
	rule := reasoning.ApplicableRule{RuleID: "ADD-AGGRESSIVE", Type: "ADD", Urgency: "ROUTINE"}
	pkt := makePacket(t, snap, rule)

	got, err := scorer.Assess(context.Background(), pkt, snap, rule)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if got.GoalsOfCareAlignment > 2 {
		t.Errorf("GoalsOfCareAlignment = %d, want ≤ 2 for ADD on end_of_life", got.GoalsOfCareAlignment)
	}
	if appropriateness.Check(got) == nil {
		t.Errorf("expected gate to HOLD for ADD on end_of_life, but Check returned nil. Assessment=%+v", got)
	}
}

// TestScorer_PalliativeAddRecommendation_GateHolds is the equivalent assertion
// for the "palliative" care-intensity vocabulary (the plan mentions both
// end_of_life and palliative as the comfort-focused equivalents).
func TestScorer_PalliativeAddRecommendation_GateHolds(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	snap := kb32ctx.ClinicalSnapshot{
		ResidentID:    uuid.New(),
		CareIntensity: "palliative",
	}
	rule := reasoning.ApplicableRule{RuleID: "ADD-AGGRESSIVE", Type: "ADD", Urgency: "ROUTINE"}
	pkt := makePacket(t, snap, rule)

	got, err := scorer.Assess(context.Background(), pkt, snap, rule)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if got.GoalsOfCareAlignment > 2 {
		t.Errorf("GoalsOfCareAlignment = %d, want ≤ 2 for ADD on palliative", got.GoalsOfCareAlignment)
	}
	if appropriateness.Check(got) == nil {
		t.Errorf("expected gate to HOLD for ADD on palliative. Assessment=%+v", got)
	}
}

// TestScorer_HappyPath_GatePasses confirms a clinically aligned scenario
// passes the gate end-to-end: ACB-driven STOP rule on an active resident with
// alternatives metadata and a restraint signaler that ran.
func TestScorer_HappyPath_GatePasses(t *testing.T) {
	scorer := appropriateness.NewSubstrateBackedScorer()
	snap := kb32ctx.ClinicalSnapshot{
		ResidentID:                uuid.New(),
		ACB:                       4,
		RestrictivePracticeActive: true,
		CareIntensity:             "comfort",
	}
	rule := reasoning.ApplicableRule{RuleID: "ADG-STOPP-PPI-ALT", Type: "STOP", Urgency: "ROUTINE"}
	pkt := makePacket(t, snap, rule)
	pkt.Sections["evidence"] = "Per ADG-2025-AU and RACGP-2024 guidance."

	got, err := scorer.Assess(context.Background(), pkt, snap, rule)
	if err != nil {
		t.Fatalf("Assess: %v", err)
	}
	if err := appropriateness.Check(got); err != nil {
		t.Errorf("expected gate to PASS for happy path, got hold. Assessment=%+v", got)
	}
}
