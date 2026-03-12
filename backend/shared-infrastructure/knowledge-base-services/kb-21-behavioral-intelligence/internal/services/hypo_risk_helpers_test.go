package services

import (
	"testing"
)

// ---------------------------------------------------------------------------
// uniqueStrings tests
// ---------------------------------------------------------------------------

func TestUniqueStrings_NoDuplicates(t *testing.T) {
	input := []string{"INSULIN", "SULFONYLUREA", "BASAL_INSULIN"}
	got := uniqueStrings(input)
	if len(got) != 3 {
		t.Errorf("len = %d, want 3", len(got))
	}
}

func TestUniqueStrings_WithDuplicates(t *testing.T) {
	input := []string{"INSULIN", "SULFONYLUREA", "INSULIN", "BASAL_INSULIN", "SULFONYLUREA"}
	got := uniqueStrings(input)
	if len(got) != 3 {
		t.Errorf("len = %d, want 3 (deduplicated)", len(got))
	}
	// Order preserved: first occurrence wins
	expected := []string{"INSULIN", "SULFONYLUREA", "BASAL_INSULIN"}
	for i, s := range expected {
		if got[i] != s {
			t.Errorf("got[%d] = %q, want %q", i, got[i], s)
		}
	}
}

func TestUniqueStrings_AllSame(t *testing.T) {
	input := []string{"INSULIN", "INSULIN", "INSULIN"}
	got := uniqueStrings(input)
	if len(got) != 1 {
		t.Errorf("len = %d, want 1", len(got))
	}
	if got[0] != "INSULIN" {
		t.Errorf("got[0] = %q, want INSULIN", got[0])
	}
}

func TestUniqueStrings_Empty(t *testing.T) {
	got := uniqueStrings([]string{})
	if len(got) != 0 {
		t.Errorf("len = %d, want 0 for empty input", len(got))
	}
}

func TestUniqueStrings_Nil(t *testing.T) {
	got := uniqueStrings(nil)
	if len(got) != 0 {
		t.Errorf("len = %d, want 0 for nil input", len(got))
	}
}

func TestUniqueStrings_SingleElement(t *testing.T) {
	got := uniqueStrings([]string{"CCB"})
	if len(got) != 1 || got[0] != "CCB" {
		t.Errorf("single element: got %v, want [CCB]", got)
	}
}
