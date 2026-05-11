package drill_through

import (
	"strings"
	"testing"
	"time"
)

func TestRenderNegativeEvidence_FullSearch(t *testing.T) {
	got := RenderNegativeEvidence(NegativeEvidenceSearch{
		Claim:           "No current indication identified for omeprazole",
		SearchedSources: []string{"eNRMC indication field", "progress notes (24mo)", "care plan"},
		UnsearchedSources: []string{"scanned discharge summaries"},
		SearchedAt:      time.Now(),
		Confidence:      "high",
	})

	if !strings.Contains(got.Statement, "in available records") {
		t.Errorf("statement must use 'in available records' framing per v1.0 Part 10.4; got %q", got.Statement)
	}
	if len(got.EvidenceLines) != 3 {
		t.Errorf("expected 3 evidence lines; got %d", len(got.EvidenceLines))
	}
	for _, l := range got.EvidenceLines {
		if !strings.HasPrefix(l, "✓ ") {
			t.Errorf("evidence line missing ✓ prefix: %q", l)
		}
	}
	if !strings.Contains(got.Caveat, "scanned discharge summaries") {
		t.Errorf("caveat must mention unsearched sources; got %q", got.Caveat)
	}
	if got.Confidence != "high" {
		t.Errorf("confidence passthrough failed")
	}
}

func TestRenderNegativeEvidence_NoUnsearched(t *testing.T) {
	got := RenderNegativeEvidence(NegativeEvidenceSearch{
		Claim:           "No active falls within 30 days",
		SearchedSources: []string{"nursing assessment", "incident reports"},
	})
	if got.Caveat != "" {
		t.Errorf("caveat should be empty when no unsearched sources; got %q", got.Caveat)
	}
	if got.Confidence != "moderate" {
		t.Errorf("default confidence should be 'moderate' per Substrate Query Feasibility Analysis; got %q", got.Confidence)
	}
}

func TestRenderNegativeEvidence_EmptyClaim(t *testing.T) {
	got := RenderNegativeEvidence(NegativeEvidenceSearch{})
	if got.Statement == "" {
		t.Error("statement must never be empty (epistemic humility default)")
	}
}

// TestRenderNegativeEvidence_StructuralFraming pins the v1.0 Part 10.4
// framing distinction: the rendering must NOT use absolute-absence
// language ("X is absent", "X does not exist").
func TestRenderNegativeEvidence_StructuralFraming(t *testing.T) {
	got := RenderNegativeEvidence(NegativeEvidenceSearch{
		Claim:           "No current indication for omeprazole",
		SearchedSources: []string{"eNRMC"},
	})
	forbidden := []string{"is absent", "does not exist", "no such"}
	for _, f := range forbidden {
		if strings.Contains(strings.ToLower(got.Statement), f) {
			t.Errorf("statement uses absolute-absence framing %q — violates v1.0 Part 10.4 epistemic humility", f)
		}
	}
}
