package kpis

import "testing"

// TestContextTime_MedianAcrossRolling30 is the verbatim plan test:
// median of [5, 8, 12, 7, 9] = 8.
func TestContextTime_MedianAcrossRolling30(t *testing.T) {
	got := MedianContextTime([]float64{5, 8, 12, 7, 9})
	if got != 8.0 {
		t.Errorf("expected 8.0, got %v", got)
	}
}

// TestContextTime_EvenLength verifies that an even-length input returns the
// average of the two middle values: median([1,2,3,4]) = 2.5.
func TestContextTime_EvenLength(t *testing.T) {
	got := MedianContextTime([]float64{1, 2, 3, 4})
	if got != 2.5 {
		t.Errorf("expected 2.5, got %v", got)
	}
}

// TestContextTime_SingleValue verifies that a single-element slice returns
// that element unchanged.
func TestContextTime_SingleValue(t *testing.T) {
	got := MedianContextTime([]float64{7.5})
	if got != 7.5 {
		t.Errorf("expected 7.5, got %v", got)
	}
}

// TestContextTime_Empty verifies the zero sentinel for empty input.
func TestContextTime_Empty(t *testing.T) {
	got := MedianContextTime([]float64{})
	if got != 0 {
		t.Errorf("expected 0, got %v", got)
	}
}

// TestContextTime_DoesNotMutateInput verifies the input slice is not sorted in place.
func TestContextTime_DoesNotMutateInput(t *testing.T) {
	input := []float64{9, 1, 5}
	_ = MedianContextTime(input)
	if input[0] != 9 || input[1] != 1 || input[2] != 5 {
		t.Errorf("input slice was mutated: %v", input)
	}
}
