package evidence

import (
	"testing"
)

// ---------------------------------------------------------------------------
// IsValidJurisdiction
// ---------------------------------------------------------------------------

func TestIsValidJurisdiction(t *testing.T) {
	valid := []string{"AU", "US", "EU", "INTL"}
	for _, j := range valid {
		if !IsValidJurisdiction(j) {
			t.Errorf("IsValidJurisdiction(%q) = false, want true", j)
		}
	}

	invalid := []string{"au", "uk", "CA", "", "INVALID", "us"}
	for _, j := range invalid {
		if IsValidJurisdiction(j) {
			t.Errorf("IsValidJurisdiction(%q) = true, want false", j)
		}
	}
}

// ---------------------------------------------------------------------------
// Select: empty and single-element cases
// ---------------------------------------------------------------------------

func TestSelect_EmptyInput(t *testing.T) {
	result := Select([]Anchor{})
	if result == nil {
		t.Fatal("Select(empty) returned nil; want non-nil empty slice")
	}
	if len(result) != 0 {
		t.Errorf("Select(empty) len = %d, want 0", len(result))
	}
	// Sliceability: confirm we can sub-slice without panic.
	_ = result[0:0]
}

func TestSelect_OnlyOneCandidate(t *testing.T) {
	in := []Anchor{{SourceID: "ADG-2025-AU", Title: "Aust Deprescribing Guideline", Jurisdiction: "AU", Rank: 1}}
	out := Select(in)
	if len(out) != 1 {
		t.Fatalf("Select(1 candidate) len = %d, want 1", len(out))
	}
	if out[0].SourceID != "ADG-2025-AU" {
		t.Errorf("SourceID = %q, want ADG-2025-AU", out[0].SourceID)
	}
}

// ---------------------------------------------------------------------------
// Select: AU-first ordering
// ---------------------------------------------------------------------------

func TestSelect_AUFirstWhenMixed(t *testing.T) {
	candidates := []Anchor{
		{SourceID: "BEERS-2023-US", Title: "Beers Criteria 2023", Jurisdiction: "US", Rank: 1},
		{SourceID: "ADG-2025-AU", Title: "Australian Deprescribing Guideline 2025", Jurisdiction: "AU", Rank: 2},
		{SourceID: "STOPP-2023-EU", Title: "STOPP/START 2023", Jurisdiction: "EU", Rank: 1},
	}
	out := Select(candidates)

	if len(out) != MaxAnchorsPerRec {
		t.Fatalf("len = %d, want %d", len(out), MaxAnchorsPerRec)
	}
	if out[0].Jurisdiction != "AU" {
		t.Errorf("first anchor jurisdiction = %q, want AU", out[0].Jurisdiction)
	}
}

// ---------------------------------------------------------------------------
// Select: max-cap enforcement
// ---------------------------------------------------------------------------

func TestSelect_MaxTwoCap(t *testing.T) {
	candidates := []Anchor{
		{SourceID: "A", Jurisdiction: "AU", Rank: 1},
		{SourceID: "B", Jurisdiction: "AU", Rank: 2},
		{SourceID: "C", Jurisdiction: "US", Rank: 1},
		{SourceID: "D", Jurisdiction: "US", Rank: 2},
		{SourceID: "E", Jurisdiction: "INTL", Rank: 1},
	}
	out := Select(candidates)
	if len(out) != MaxAnchorsPerRec {
		t.Errorf("len = %d, want %d (MaxAnchorsPerRec)", len(out), MaxAnchorsPerRec)
	}
}

// ---------------------------------------------------------------------------
// Select: SourceID tie-break (deterministic ordering)
// ---------------------------------------------------------------------------

func TestSelect_SourceIDTieBreak(t *testing.T) {
	// Two AU anchors with identical Rank — SourceID must break the tie alphabetically.
	candidates := []Anchor{
		{SourceID: "ZULU-AU", Jurisdiction: "AU", Rank: 1},
		{SourceID: "ALPHA-AU", Jurisdiction: "AU", Rank: 1},
	}
	out := Select(candidates)
	if len(out) != 2 {
		t.Fatalf("len = %d, want 2", len(out))
	}
	if out[0].SourceID != "ALPHA-AU" {
		t.Errorf("first SourceID = %q, want ALPHA-AU", out[0].SourceID)
	}
	if out[1].SourceID != "ZULU-AU" {
		t.Errorf("second SourceID = %q, want ZULU-AU", out[1].SourceID)
	}
}

// ---------------------------------------------------------------------------
// Select: Rank respected within same jurisdiction
// ---------------------------------------------------------------------------

func TestSelect_RankRespected(t *testing.T) {
	// Three US sources at different ranks — Rank 1 must beat Rank 3.
	candidates := []Anchor{
		{SourceID: "US-RANK3", Jurisdiction: "US", Rank: 3},
		{SourceID: "US-RANK1", Jurisdiction: "US", Rank: 1},
		{SourceID: "US-RANK2", Jurisdiction: "US", Rank: 2},
	}
	out := Select(candidates)
	if len(out) != MaxAnchorsPerRec {
		t.Fatalf("len = %d, want %d", len(out), MaxAnchorsPerRec)
	}
	if out[0].SourceID != "US-RANK1" {
		t.Errorf("first SourceID = %q, want US-RANK1", out[0].SourceID)
	}
	if out[1].SourceID != "US-RANK2" {
		t.Errorf("second SourceID = %q, want US-RANK2", out[1].SourceID)
	}
}

// ---------------------------------------------------------------------------
// Select: input not mutated
// ---------------------------------------------------------------------------

func TestSelect_DoesNotMutateInput(t *testing.T) {
	original := []Anchor{
		{SourceID: "B", Jurisdiction: "US", Rank: 1},
		{SourceID: "A", Jurisdiction: "AU", Rank: 1},
	}
	before := make([]Anchor, len(original))
	copy(before, original)

	Select(original)

	for i := range original {
		if original[i] != before[i] {
			t.Errorf("input mutated at index %d: got %+v, want %+v", i, original[i], before[i])
		}
	}
}
