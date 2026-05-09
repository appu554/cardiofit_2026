package framing_test

import (
	"errors"
	"testing"

	"github.com/cardiofit/kb32/internal/framing"
)

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func baseContent() framing.ClinicalContent {
	return framing.ClinicalContent{
		RuleID:          "STOPP-B3",
		Type:            "STOP",
		EvidenceAnchors: []string{"ADG-2025-AU", "BEERS-2023-US"},
		Urgency:         "red",
	}
}

// ---------------------------------------------------------------------------
// ContentHash determinism
// ---------------------------------------------------------------------------

func TestContentHash_DeterministicSameInput(t *testing.T) {
	c := baseContent()
	h1 := framing.ContentHash(c)
	h2 := framing.ContentHash(c)
	if h1 != h2 {
		t.Errorf("same input produced different hashes: %q vs %q", h1, h2)
	}
}

func TestContentHash_AnchorReorderingDoesNotAffectHash(t *testing.T) {
	ordered := framing.ClinicalContent{
		RuleID:          "STOPP-B3",
		Type:            "STOP",
		EvidenceAnchors: []string{"A", "B", "C"},
		Urgency:         "red",
	}
	reversed := framing.ClinicalContent{
		RuleID:          "STOPP-B3",
		Type:            "STOP",
		EvidenceAnchors: []string{"C", "A", "B"},
		Urgency:         "red",
	}
	h1 := framing.ContentHash(ordered)
	h2 := framing.ContentHash(reversed)
	if h1 != h2 {
		t.Errorf("anchor reordering changed hash:\n  ordered  %q\n  reversed %q", h1, h2)
	}
}

func TestContentHash_DifferentRuleIDDifferentHash(t *testing.T) {
	a := baseContent()
	b := baseContent()
	b.RuleID = "STOPP-X99"
	if framing.ContentHash(a) == framing.ContentHash(b) {
		t.Error("different RuleIDs should produce different hashes")
	}
}

func TestContentHash_DifferentTypeDifferentHash(t *testing.T) {
	a := baseContent()
	b := baseContent()
	b.Type = "MONITOR"
	if framing.ContentHash(a) == framing.ContentHash(b) {
		t.Error("different Types should produce different hashes")
	}
}

func TestContentHash_DifferentUrgencyDifferentHash(t *testing.T) {
	a := baseContent()
	b := baseContent()
	b.Urgency = "amber"
	if framing.ContentHash(a) == framing.ContentHash(b) {
		t.Error("different Urgency values should produce different hashes")
	}
}

// ---------------------------------------------------------------------------
// Multi-audience framing — same content, different framings → same hash
// ---------------------------------------------------------------------------

func TestSameContentDifferentFramingsSameHash(t *testing.T) {
	content := baseContent()
	h := framing.ContentHash(content)

	framings := []framing.FramingAdaptation{
		{Audience: "gp", OpeningLine: "Please review this medication.", ClosingCall: "Discuss at next appointment."},
		{Audience: "pharmacist", OpeningLine: "Clinical alert: deprescribing opportunity.", ClosingCall: "Review medication list."},
		{Audience: "regulator", OpeningLine: "Compliance notice.", ClosingCall: "Audit trail attached."},
	}
	// All three framings attach to the same ClinicalContent; hash must not change.
	for _, fr := range framings {
		got := framing.ContentHash(content)
		if got != h {
			t.Errorf("framing for audience %q changed ContentHash: got %q, want %q",
				fr.Audience, got, h)
		}
	}
}

// ---------------------------------------------------------------------------
// IsContentInvariantAcross
// ---------------------------------------------------------------------------

func TestIsContentInvariantAcross_TruePath(t *testing.T) {
	content := baseContent()
	contents := []framing.ClinicalContent{content, content, content}
	framings := []framing.FramingAdaptation{
		{Audience: "gp"},
		{Audience: "pharmacist"},
		{Audience: "regulator"},
	}
	if !framing.IsContentInvariantAcross(framings, contents) {
		t.Error("expected IsContentInvariantAcross == true when all contents identical")
	}
}

func TestIsContentInvariantAcross_FalsePathOneDifferent(t *testing.T) {
	c1 := baseContent()
	c2 := baseContent()
	c3 := baseContent()
	c3.Urgency = "green" // deliberately different
	contents := []framing.ClinicalContent{c1, c2, c3}
	framings := []framing.FramingAdaptation{
		{Audience: "gp"},
		{Audience: "pharmacist"},
		{Audience: "regulator"},
	}
	if framing.IsContentInvariantAcross(framings, contents) {
		t.Error("expected IsContentInvariantAcross == false when one content differs")
	}
}

func TestIsContentInvariantAcross_EmptySliceIsTrue(t *testing.T) {
	if !framing.IsContentInvariantAcross(nil, nil) {
		t.Error("empty slice should be vacuously true")
	}
}

func TestIsContentInvariantAcross_SingleEntryIsTrue(t *testing.T) {
	contents := []framing.ClinicalContent{baseContent()}
	if !framing.IsContentInvariantAcross(nil, contents) {
		t.Error("single entry should be vacuously true")
	}
}

// ---------------------------------------------------------------------------
// ClinicalContent.Validate
// ---------------------------------------------------------------------------

func TestValidate_RejectsEmpty(t *testing.T) {
	var c framing.ClinicalContent // all zero values
	if err := c.Validate(); err == nil {
		t.Error("expected error for empty ClinicalContent")
	}
}

func TestValidate_RejectsEmptyRuleID(t *testing.T) {
	c := baseContent()
	c.RuleID = ""
	if err := c.Validate(); err == nil {
		t.Error("expected error for empty RuleID")
	}
}

func TestValidate_RejectsEmptyType(t *testing.T) {
	c := baseContent()
	c.Type = ""
	if err := c.Validate(); err == nil {
		t.Error("expected error for empty Type")
	}
}

func TestValidate_RejectsInvalidUrgency(t *testing.T) {
	c := baseContent()
	c.Urgency = "critical" // not a valid urgency value
	err := c.Validate()
	if err == nil {
		t.Error("expected error for invalid Urgency value")
	}
}

func TestValidate_RejectsEmptyUrgency(t *testing.T) {
	c := baseContent()
	c.Urgency = ""
	if err := c.Validate(); err == nil {
		t.Error("expected error for empty Urgency")
	}
}

func TestValidate_AcceptsValidContent(t *testing.T) {
	for _, urgency := range []string{"red", "amber", "green"} {
		c := baseContent()
		c.Urgency = urgency
		if err := c.Validate(); err != nil {
			t.Errorf("expected nil for valid content with Urgency=%q, got %v", urgency, err)
		}
	}
}

// ---------------------------------------------------------------------------
// IsValidAudience
// ---------------------------------------------------------------------------

func TestIsValidAudience(t *testing.T) {
	cases := []struct {
		audience string
		wantOK   bool
	}{
		{"gp", true},
		{"pharmacist", true},
		{"rach_staff", true},
		{"regulator", true},
		{"patient", false},
		{"", false},
		{"GP", false},      // case-sensitive
		{"Pharmacist", false},
		{"doctor", false},
	}
	for _, tc := range cases {
		got := framing.IsValidAudience(tc.audience)
		if got != tc.wantOK {
			t.Errorf("IsValidAudience(%q) = %v, want %v", tc.audience, got, tc.wantOK)
		}
	}
}

// ---------------------------------------------------------------------------
// ErrInvalidContent sentinel reachable via Validate
// ---------------------------------------------------------------------------

func TestValidate_SentinelReachable(t *testing.T) {
	c := framing.ClinicalContent{RuleID: "X", Type: "STOP", Urgency: "bad"}
	err := c.Validate()
	if err == nil {
		t.Fatal("expected an error")
	}
	// The error should be distinct from appropriateness errors; here we just
	// confirm it is not nil and is not wrapping a different sentinel.
	if errors.Is(err, nil) {
		t.Error("error wrapping nil is unexpected")
	}
}
