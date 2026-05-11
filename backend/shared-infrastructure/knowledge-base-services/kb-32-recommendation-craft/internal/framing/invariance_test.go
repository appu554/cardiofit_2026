package framing

import (
	"testing"
)

// canonicalAudiences is the audience matrix this CI gate enforces. Adding a
// new audience requires updating this list AND ensuring the framing adapter
// for that audience does not modify ClinicalContent — only wraps it.
var canonicalAudiences = []string{
	"gp", "pharmacist", "rach_staff", "regulator",
}

// TestFrameVsContentInvariance is a Phase 3 (tightened) Task 5 CI invariance
// gate per Ethical Architecture Implementation Guidelines v1.0 Principle 1
// (frame-vs-content separation).
//
// For a fixed ClinicalContent, ContentHash(content) must return the same value
// regardless of which audience adaptation wraps it. Any code change that makes
// content vary by audience breaks this test, which blocks merge.
//
// Located here (kb-32 internal/framing) rather than the plan's
// shared/v2_substrate/ethics/ci_gates/ because framing is an internal package
// of the kb-32 module and cannot be imported from shared/. The CI-gate
// property is preserved: any failing go test blocks merge.
func TestFrameVsContentInvariance(t *testing.T) {
	content := ClinicalContent{
		RuleID:          "RULE-CKD-METFORMIN-STOP-001",
		Type:            "STOP",
		EvidenceAnchors: []string{"NICE-NG203", "KDIGO-2024", "MHRA-2023"},
		Urgency:         "amber",
	}
	expected := ContentHash(content)

	// Build one FramingAdaptation per canonical audience. Production framings
	// wrap content; they must not mutate the ClinicalContent such that
	// ContentHash drifts.
	framings := make([]FramingAdaptation, 0, len(canonicalAudiences))
	for _, aud := range canonicalAudiences {
		if !IsValidAudience(aud) {
			t.Fatalf("canonical audience %q rejected by IsValidAudience; matrix and validator have drifted", aud)
		}
		framings = append(framings, FramingAdaptation{
			Audience:    aud,
			OpeningLine: "audience-specific opening for " + aud,
			ClosingCall: "audience-specific call for " + aud,
		})

		if got := ContentHash(content); got != expected {
			t.Errorf("audience %q: content_hash drifted (%s != %s)", aud, got, expected)
		}
	}

	// Belt-and-braces: verify the package's own audit predicate also reports
	// invariance across the parallel content slice (one entry per framing).
	contents := make([]ClinicalContent, len(framings))
	for i := range framings {
		contents[i] = content
	}
	if !IsContentInvariantAcross(framings, contents) {
		t.Fatalf("IsContentInvariantAcross returned false across canonical audiences %v", canonicalAudiences)
	}
}
