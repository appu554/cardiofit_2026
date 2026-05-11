package substrate_types

import (
	"reflect"
	"testing"
)

// TestGoalsOfCareEntryFieldPinning pins the s2-side GoalsOfCareEntry
// shape. Phase 1: there is NO canonical type in kb-20 — the GoC
// substrate signal is care_intensity_history per the vocabulary
// discovery note in goals_of_care.go. Pin the local shape so any
// downstream consumer (Layer 1 panel builder, drill-through) breaks at
// CI when fields rename.
func TestGoalsOfCareEntryFieldPinning(t *testing.T) {
	want := []string{
		"State",
		"EffectiveFrom",
		"EffectiveTo",
		"DocumentedBy",
		"FreshnessFlag",
		"SubstrateID",
	}
	got := fieldNames(GoalsOfCareEntry{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("GoalsOfCareEntry fields drifted: want %v got %v", want, got)
	}
}

// TestCareIntensityEntryFieldPinning pins the s2-side CareIntensityEntry
// shape against kb-20's care_intensity_history row shape (Wave 2.4).
// SOURCE OF TRUTH: shared/v2_substrate/models/care_intensity.go.
func TestCareIntensityEntryFieldPinning(t *testing.T) {
	want := []string{
		"Tag",
		"EffectiveDate",
		"DocumentedBy",
		"FreshnessFlag",
		"SubstrateID",
	}
	got := fieldNames(CareIntensityEntry{})
	if !reflect.DeepEqual(want, got) {
		t.Fatalf("CareIntensityEntry fields drifted: want %v got %v", want, got)
	}
}

// TestGoCStateConstants pins the GoC state vocabulary. The Wave 2.4
// kb-20 vocabulary is the authoritative source for the first four
// values; GoCStateEndOfLife is the legacy short-form passthrough.
func TestGoCStateConstants(t *testing.T) {
	cases := []struct {
		got, want, name string
	}{
		{GoCStateActiveTreatment, "active_treatment", "ActiveTreatment"},
		{GoCStateRehabilitation, "rehabilitation", "Rehabilitation"},
		{GoCStateComfortFocused, "comfort_focused", "ComfortFocused"},
		{GoCStatePalliative, "palliative", "Palliative"},
		{GoCStateEndOfLife, "end_of_life", "EndOfLife"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("GoCState %s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}

// TestCareIntensityTagConstants pins the Wave 2.4 closed-set tag
// vocabulary against kb-20.
func TestCareIntensityTagConstants(t *testing.T) {
	cases := []struct {
		got, want, name string
	}{
		{CareIntensityTagActiveTreatment, "active_treatment", "ActiveTreatment"},
		{CareIntensityTagRehabilitation, "rehabilitation", "Rehabilitation"},
		{CareIntensityTagComfortFocused, "comfort_focused", "ComfortFocused"},
		{CareIntensityTagPalliative, "palliative", "Palliative"},
	}
	for _, tc := range cases {
		if tc.got != tc.want {
			t.Errorf("CareIntensityTag %s = %q, want %q", tc.name, tc.got, tc.want)
		}
	}
}
