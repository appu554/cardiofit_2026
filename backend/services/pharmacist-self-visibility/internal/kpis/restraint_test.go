package kpis

import "testing"

// TestRestraintOverridePattern_CountsByType is the verbatim plan test.
func TestRestraintOverridePattern_CountsByType(t *testing.T) {
	overrides := []RestraintOverride{
		{RecommendationType: "deprescribe", Reasoning: "patient preference"},
		{RecommendationType: "dose_reduce", Reasoning: "renal impairment considered"},
		{RecommendationType: "deprescribe", Reasoning: "falls risk"},
	}
	got := RestraintOverridePattern(overrides)

	if got["deprescribe"] != 2 {
		t.Errorf("expected deprescribe=2, got %d", got["deprescribe"])
	}
	if got["dose_reduce"] != 1 {
		t.Errorf("expected dose_reduce=1, got %d", got["dose_reduce"])
	}
}

// TestRestraintOverridePattern_Empty verifies an empty (non-nil) map is returned
// for empty input.
func TestRestraintOverridePattern_Empty(t *testing.T) {
	got := RestraintOverridePattern([]RestraintOverride{})
	if got == nil {
		t.Error("expected non-nil map for empty input, got nil")
	}
	if len(got) != 0 {
		t.Errorf("expected empty map, got %v", got)
	}
}

// TestRestraintOverridePattern_SingleEntry verifies a single override is counted.
func TestRestraintOverridePattern_SingleEntry(t *testing.T) {
	overrides := []RestraintOverride{
		{RecommendationType: "switch", Reasoning: "formulary change"},
	}
	got := RestraintOverridePattern(overrides)
	if got["switch"] != 1 {
		t.Errorf("expected switch=1, got %d", got["switch"])
	}
}
