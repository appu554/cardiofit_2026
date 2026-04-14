package services

import (
	"testing"
	"time"
)

func TestSteadyStateWindow_KnownDrugs(t *testing.T) {
	cases := []struct {
		class    string
		expected time.Duration
	}{
		{"AMLODIPINE", 8 * 24 * time.Hour},
		{"FELODIPINE", 7 * 24 * time.Hour},
		{"NIFEDIPINE", 3 * 24 * time.Hour},
		{"LOSARTAN", 6 * 24 * time.Hour},
		{"VALSARTAN", 4 * 24 * time.Hour},
		{"TELMISARTAN", 7 * 24 * time.Hour},
		{"LISINOPRIL", 5 * 24 * time.Hour},
		{"RAMIPRIL", 5 * 24 * time.Hour},
		{"ENALAPRIL", 4 * 24 * time.Hour},
		{"METOPROLOL", 2 * 24 * time.Hour},
		{"ATENOLOL", 2 * 24 * time.Hour},
		{"BISOPROLOL", 3 * 24 * time.Hour},
		{"HCTZ", 7 * 24 * time.Hour},
		{"INDAPAMIDE", 7 * 24 * time.Hour},
	}
	for _, tc := range cases {
		if got := SteadyStateWindow(tc.class); got != tc.expected {
			t.Errorf("%s: expected %v, got %v", tc.class, tc.expected, got)
		}
	}
}

func TestSteadyStateWindow_UnknownDrug_Default(t *testing.T) {
	if got := SteadyStateWindow("UNKNOWN_DRUG"); got != defaultSteadyStateWindow {
		t.Errorf("expected default %v, got %v", defaultSteadyStateWindow, got)
	}
}

func TestSteadyStateWindow_EmptyString_Default(t *testing.T) {
	if got := SteadyStateWindow(""); got != defaultSteadyStateWindow {
		t.Errorf("expected default %v for empty class, got %v", defaultSteadyStateWindow, got)
	}
}

func TestSteadyStateWindow_CaseInsensitive(t *testing.T) {
	if got := SteadyStateWindow("amlodipine"); got != 8*24*time.Hour {
		t.Errorf("expected lowercase to match, got %v", got)
	}
	if got := SteadyStateWindow("  Metoprolol  "); got != 2*24*time.Hour {
		t.Errorf("expected whitespace+mixed-case to match, got %v", got)
	}
}
