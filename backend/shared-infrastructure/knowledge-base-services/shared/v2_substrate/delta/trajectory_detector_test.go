package delta

import "testing"

func TestDetectTrajectory_RisingThreeInARow(t *testing.T) {
	// values are observed_at-DESC: most recent first.
	// ASC: 100, 105, 110, 115 → all "up", 3 trailing ups.
	v := []float64{115, 110, 105, 100}
	snap := DetectTrajectory(v)
	if !snap.IsTrending {
		t.Errorf("IsTrending = false, want true")
	}
	if snap.ConsecutiveSameDirection != 3 {
		t.Errorf("ConsecutiveSameDirection = %d, want 3", snap.ConsecutiveSameDirection)
	}
	if snap.Direction != "up" {
		t.Errorf("Direction = %q, want up", snap.Direction)
	}
}

func TestDetectTrajectory_FallingFourInARow(t *testing.T) {
	// ASC: 60, 55, 50, 45, 40 → 4 trailing downs.
	v := []float64{40, 45, 50, 55, 60}
	snap := DetectTrajectory(v)
	if !snap.IsTrending {
		t.Errorf("IsTrending = false, want true")
	}
	if snap.ConsecutiveSameDirection != 4 {
		t.Errorf("ConsecutiveSameDirection = %d, want 4", snap.ConsecutiveSameDirection)
	}
	if snap.Direction != "down" {
		t.Errorf("Direction = %q, want down", snap.Direction)
	}
}

func TestDetectTrajectory_TwoConsecutiveNotTrending(t *testing.T) {
	// ASC: 100, 100, 105, 110 → flat, up, up. Last is "up", count=2.
	v := []float64{110, 105, 100, 100}
	snap := DetectTrajectory(v)
	if snap.IsTrending {
		t.Errorf("IsTrending = true, want false (only 2 trailing ups)")
	}
	if snap.ConsecutiveSameDirection != 2 {
		t.Errorf("ConsecutiveSameDirection = %d, want 2", snap.ConsecutiveSameDirection)
	}
	if snap.Direction != "up" {
		t.Errorf("Direction = %q, want up", snap.Direction)
	}
}

func TestDetectTrajectory_MixedDirection(t *testing.T) {
	// ASC: 100, 105, 100, 105, 100 → up, down, up, down. Last "down", count=1.
	v := []float64{100, 105, 100, 105, 100}
	snap := DetectTrajectory(v)
	if snap.IsTrending {
		t.Errorf("IsTrending = true, want false on alternating direction")
	}
	if snap.ConsecutiveSameDirection != 1 {
		t.Errorf("ConsecutiveSameDirection = %d, want 1", snap.ConsecutiveSameDirection)
	}
}

func TestDetectTrajectory_TrailingFlatKillsTrend(t *testing.T) {
	// ASC: 100, 105, 110, 110 → up, up, flat. Trailing flat → reset.
	v := []float64{110, 110, 105, 100}
	snap := DetectTrajectory(v)
	if snap.IsTrending {
		t.Errorf("IsTrending = true, want false on trailing flat")
	}
	if snap.ConsecutiveSameDirection != 0 {
		t.Errorf("ConsecutiveSameDirection = %d, want 0 on trailing flat", snap.ConsecutiveSameDirection)
	}
}

func TestDetectTrajectory_InsufficientData(t *testing.T) {
	cases := [][]float64{nil, {100}, {100, 105}}
	for _, v := range cases {
		snap := DetectTrajectory(v)
		if snap.IsTrending || snap.ConsecutiveSameDirection != 0 {
			t.Errorf("DetectTrajectory(%v) = %+v, want zero", v, snap)
		}
	}
}

func TestDetectTrajectory_StableValues(t *testing.T) {
	// All identical → all "flat" → not trending.
	v := []float64{120, 120, 120, 120, 120}
	snap := DetectTrajectory(v)
	if snap.IsTrending {
		t.Errorf("IsTrending = true on stable values")
	}
	if snap.ConsecutiveSameDirection != 0 {
		t.Errorf("ConsecutiveSameDirection = %d on stable values", snap.ConsecutiveSameDirection)
	}
}

func TestDetectTrajectory_DoesNotMutateInput(t *testing.T) {
	v := []float64{40, 45, 50, 55, 60}
	orig := make([]float64, len(v))
	copy(orig, v)
	_ = DetectTrajectory(v)
	for i := range v {
		if v[i] != orig[i] {
			t.Errorf("DetectTrajectory mutated input at index %d: got %v, want %v", i, v[i], orig[i])
		}
	}
}
