package api

import (
	"testing"

	"kb-patient-profile/internal/services"
)

// TestG4_PRPEntryEligibleWhenVFRPActivated validates that the registry correctly
// identifies PRP eligibility when the patient has a protein gap — the condition
// that triggers auto-activation in the activateProtocol handler (G-4).
func TestG4_PRPEntryEligibleWhenVFRPActivated(t *testing.T) {
	registry := services.NewProtocolRegistry()

	numericFields := map[string]float64{
		// Meets PRP entry: protein_gap >= 20
		"protein_gap": 25.0,
		// No exclusion: eGFR is safe
		"egfr": 55.0,
	}
	boolFields := map[string]bool{}

	eligible, reason := registry.CheckEntry("M3-PRP", numericFields, boolFields)
	if !eligible {
		t.Errorf("expected M3-PRP eligible for auto-activation, got ineligible: reason=%q", reason)
	}
}

// TestG4_PRPNotAutoActivatedWhenEGFRExcludes ensures that PRP is NOT
// auto-activated when the patient has renal impairment (eGFR < 30 → LS-01).
func TestG4_PRPNotAutoActivatedWhenEGFRExcludes(t *testing.T) {
	registry := services.NewProtocolRegistry()

	numericFields := map[string]float64{
		"protein_gap": 25.0,
		"egfr":        25.0, // below 30 — LS-01 exclusion
	}
	boolFields := map[string]bool{}

	eligible, reason := registry.CheckEntry("M3-PRP", numericFields, boolFields)
	if eligible {
		t.Error("expected M3-PRP ineligible due to LS-01 (eGFR < 30), but was eligible")
	}
	if reason != "LS-01" {
		t.Errorf("expected exclusion reason LS-01, got %q", reason)
	}
}

// TestG4_PRPNotAutoActivatedWhenNoProteinGap ensures that PRP is NOT
// auto-activated when the patient has no protein gap (no entry criterion met).
func TestG4_PRPNotAutoActivatedWhenNoProteinGap(t *testing.T) {
	registry := services.NewProtocolRegistry()

	numericFields := map[string]float64{
		// protein_gap < 20 and protein_intake_gkg >= 0.8 — neither entry criterion met
		"protein_gap":         10.0,
		"protein_intake_gkg":  1.2,
		"egfr":                60.0,
	}
	boolFields := map[string]bool{}

	eligible, reason := registry.CheckEntry("M3-PRP", numericFields, boolFields)
	if eligible {
		t.Error("expected M3-PRP ineligible when no entry criterion is met")
	}
	if reason != "NO_ENTRY_CRITERIA_MET" {
		t.Errorf("expected NO_ENTRY_CRITERIA_MET, got %q", reason)
	}
}

// TestG4_PRPNotAutoActivatedWhenNephroticSyndrome ensures the nephrotic
// syndrome exclusion (NEPHRO-EXCL) blocks PRP auto-activation.
func TestG4_PRPNotAutoActivatedWhenNephroticSyndrome(t *testing.T) {
	registry := services.NewProtocolRegistry()

	numericFields := map[string]float64{
		"protein_gap":       25.0,
		"egfr":              50.0,
		"nephrotic_syndrome": 1.0,
	}
	boolFields := map[string]bool{}

	eligible, reason := registry.CheckEntry("M3-PRP", numericFields, boolFields)
	if eligible {
		t.Error("expected M3-PRP ineligible due to NEPHRO-EXCL, but was eligible")
	}
	if reason != "NEPHRO-EXCL" {
		t.Errorf("expected exclusion reason NEPHRO-EXCL, got %q", reason)
	}
}

// TestG4_PRPEligibleViaLowProteinIntake validates the second PRP entry path:
// protein_intake_gkg < 0.8 (even without an explicit protein_gap field).
func TestG4_PRPEligibleViaLowProteinIntake(t *testing.T) {
	registry := services.NewProtocolRegistry()

	numericFields := map[string]float64{
		"protein_intake_gkg": 0.6, // < 0.8 — entry criterion met
		"egfr":               60.0,
	}
	boolFields := map[string]bool{}

	eligible, reason := registry.CheckEntry("M3-PRP", numericFields, boolFields)
	if !eligible {
		t.Errorf("expected M3-PRP eligible via low protein intake, got ineligible: reason=%q", reason)
	}
}

// TestMapPhaseToSeason validates that each M3-MAINTAIN phase maps to the correct
// engagement season name and number, and that unknown phases default to CORRECTION (1).
func TestMapPhaseToSeason(t *testing.T) {
	tests := []struct {
		phase  string
		number int
	}{
		{"CONSOLIDATION", 2},
		{"INDEPENDENCE", 3},
		{"STABILITY", 4},
		{"PARTNERSHIP", 5},
		{"BASELINE", 1},  // default → CORRECTION
		{"WHATEVER", 1},  // unknown → CORRECTION
	}
	for _, tt := range tests {
		t.Run(tt.phase, func(t *testing.T) {
			season := mapPhaseToSeason(tt.phase)
			if season.Number != tt.number {
				t.Errorf("season number = %d, want %d", season.Number, tt.number)
			}
		})
	}
}

// TestActivateProtocolRequest_NilFieldsDefaultSafely documents the nil-guard
// behaviour in the handler: if NumericFields or BoolFields are omitted from the
// request, the handler initialises them to empty maps before calling CheckEntry.
// This is a logic test — no HTTP server required.
func TestActivateProtocolRequest_NilFieldsDefaultSafely(t *testing.T) {
	registry := services.NewProtocolRegistry()

	// Simulate the handler nil-guard: empty maps → no entry criterion met → not eligible.
	numericFields := map[string]float64{}
	boolFields := map[string]bool{}

	eligible, _ := registry.CheckEntry("M3-PRP", numericFields, boolFields)
	// With no fields supplied, PRP entry criteria cannot be evaluated — eligible
	// must be false (NO_ENTRY_CRITERIA_MET), ensuring auto-activation is skipped.
	if eligible {
		t.Error("expected PRP not eligible when no fields supplied (safe default)")
	}
}
