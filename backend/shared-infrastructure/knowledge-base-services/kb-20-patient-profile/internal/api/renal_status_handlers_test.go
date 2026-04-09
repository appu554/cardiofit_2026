package api

import "testing"

// ---------------------------------------------------------------------------
// TestClassifyCKDStage — KDIGO eGFR-based CKD staging
// ---------------------------------------------------------------------------

func TestClassifyCKDStage(t *testing.T) {
	tests := []struct {
		name     string
		egfr     float64
		expected string
	}{
		{name: "G1 — normal/high", egfr: 95, expected: "G1"},
		{name: "G2 — mildly decreased", egfr: 72, expected: "G2"},
		{name: "G3a — mild-moderate", egfr: 52, expected: "G3a"},
		{name: "G3b — moderate-severe", egfr: 38, expected: "G3b"},
		{name: "G4 — severely decreased", egfr: 22, expected: "G4"},
		{name: "G5 — kidney failure", egfr: 10, expected: "G5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCKDStage(tc.egfr)
			if got != tc.expected {
				t.Errorf("classifyCKDStage(%.0f) = %s, want %s", tc.egfr, got, tc.expected)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// TestClassifyCKDStage_Boundaries — edge cases at stage boundaries
// ---------------------------------------------------------------------------

func TestClassifyCKDStage_Boundaries(t *testing.T) {
	tests := []struct {
		name     string
		egfr     float64
		expected string
	}{
		{name: "exactly 90", egfr: 90, expected: "G1"},
		{name: "exactly 60", egfr: 60, expected: "G2"},
		{name: "exactly 45", egfr: 45, expected: "G3a"},
		{name: "exactly 30", egfr: 30, expected: "G3b"},
		{name: "exactly 15", egfr: 15, expected: "G4"},
		{name: "exactly 0", egfr: 0, expected: "G5"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := classifyCKDStage(tc.egfr)
			if got != tc.expected {
				t.Errorf("classifyCKDStage(%.0f) = %s, want %s", tc.egfr, got, tc.expected)
			}
		})
	}
}
