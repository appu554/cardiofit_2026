package instability_chronology

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestIsValidInstabilityPrimitive(t *testing.T) {
	t.Run("happy_path_all_canonical_primitives", func(t *testing.T) {
		for _, p := range ValidInstabilityPrimitives {
			if !IsValidInstabilityPrimitive(string(p)) {
				t.Errorf("expected %q to be valid", p)
			}
		}
	})

	t.Run("rejects_empty_string", func(t *testing.T) {
		if IsValidInstabilityPrimitive("") {
			t.Error("empty string must not be a valid primitive")
		}
	})

	t.Run("rejects_unknown", func(t *testing.T) {
		bad := []string{"medication", "Medication_Change", "MEDICATION_CHANGE", "fall ", " fall", "unknown_primitive"}
		for _, s := range bad {
			if IsValidInstabilityPrimitive(s) {
				t.Errorf("expected %q to be invalid", s)
			}
		}
	})
}

func TestIsValidAudienceClass(t *testing.T) {
	t.Run("happy_path_all_canonical_audiences", func(t *testing.T) {
		for _, a := range ValidAudienceClasses {
			if !IsValidAudienceClass(string(a)) {
				t.Errorf("expected %q to be valid", a)
			}
		}
	})

	t.Run("rejects_empty_string", func(t *testing.T) {
		if IsValidAudienceClass("") {
			t.Error("empty string must not be a valid audience class")
		}
	})

	t.Run("rejects_unknown", func(t *testing.T) {
		bad := []string{"Pharmacist", "PHARMACIST", "doctor", "nurse", "rach-operator"}
		for _, s := range bad {
			if IsValidAudienceClass(s) {
				t.Errorf("expected %q to be invalid", s)
			}
		}
	})
}

// TestValidInstabilityPrimitivesMatchesConstants guards against drift: if
// someone adds a Primitive* constant but forgets to extend the slice (or
// vice versa), this test fires.
func TestValidInstabilityPrimitivesMatchesConstants(t *testing.T) {
	expected := []InstabilityPrimitive{
		PrimitiveMedicationChange,
		PrimitiveIntakeDecline,
		PrimitiveFall,
		PrimitiveConfusionOnset,
		PrimitiveOrthostaticInstability,
		PrimitiveSedation,
	}
	if len(ValidInstabilityPrimitives) != len(expected) {
		t.Fatalf("ValidInstabilityPrimitives length = %d, want %d", len(ValidInstabilityPrimitives), len(expected))
	}
	for i, p := range expected {
		if ValidInstabilityPrimitives[i] != p {
			t.Errorf("ValidInstabilityPrimitives[%d] = %q, want %q", i, ValidInstabilityPrimitives[i], p)
		}
	}
}

func TestValidAudienceClassesMatchesConstants(t *testing.T) {
	expected := []AudienceClass{
		AudiencePharmacist,
		AudienceRACHOperator,
		AudienceGovernance,
		AudienceFamilyCommunication,
		AudienceAuditDefensibility,
	}
	if len(ValidAudienceClasses) != len(expected) {
		t.Fatalf("ValidAudienceClasses length = %d, want %d", len(ValidAudienceClasses), len(expected))
	}
	for i, a := range expected {
		if ValidAudienceClasses[i] != a {
			t.Errorf("ValidAudienceClasses[%d] = %q, want %q", i, ValidAudienceClasses[i], a)
		}
	}
}

func TestTimeWindowContains(t *testing.T) {
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	w := TimeWindow{Start: start, End: end}

	cases := []struct {
		name string
		t    time.Time
		want bool
	}{
		{"equal_start_is_inclusive", start, true},
		{"equal_end_is_exclusive", end, false},
		{"before_start", start.Add(-time.Nanosecond), false},
		{"after_end", end.Add(time.Nanosecond), false},
		{"inside", start.Add(72 * time.Hour), true},
		{"one_nanosecond_before_end", end.Add(-time.Nanosecond), true},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := w.Contains(c.t); got != c.want {
				t.Errorf("Contains(%v) = %v, want %v", c.t, got, c.want)
			}
		})
	}
}

func TestTimeWindowDuration(t *testing.T) {
	t.Run("positive_window", func(t *testing.T) {
		w := TimeWindow{
			Start: time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
		}
		want := 14 * 24 * time.Hour
		if got := w.Duration(); got != want {
			t.Errorf("Duration = %v, want %v", got, want)
		}
	})

	t.Run("negative_window_allowed", func(t *testing.T) {
		// End < Start is permitted (documented sentinel use). Duration is
		// simply End.Sub(Start) with no normalisation.
		w := TimeWindow{
			Start: time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC),
			End:   time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
		}
		if got := w.Duration(); got >= 0 {
			t.Errorf("Duration = %v, want negative", got)
		}
	})

	t.Run("zero_window", func(t *testing.T) {
		now := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
		w := TimeWindow{Start: now, End: now}
		if got := w.Duration(); got != 0 {
			t.Errorf("Duration = %v, want 0", got)
		}
	})
}

// TestInstabilityChronologyJSONRoundTrip verifies that a fully-populated
// chronology survives a marshal/unmarshal cycle. Particular attention:
//   - map[AudienceClass]ChronologyRendering — Go encodes the underlying
//     string keys, so this must round-trip cleanly.
//   - time.Time fields — UTC-normalised to avoid location-pointer inequality.
func TestInstabilityChronologyJSONRoundTrip(t *testing.T) {
	residentID := uuid.New()
	evt1ID := uuid.New()
	evt2ID := uuid.New()
	subRefID := uuid.New()
	start := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)

	original := InstabilityChronology{
		ResidentID: residentID,
		TimeWindow: TimeWindow{Start: start, End: end},
		Events: []ChronologyEvent{
			{
				EventID:       evt1ID,
				Timestamp:     start.Add(24 * time.Hour),
				EventType:     "medication_change",
				PrimitiveType: PrimitiveMedicationChange,
				Severity:      3,
				Description:   "Frusemide dose increased 40mg → 80mg daily",
				SubstrateRefs: []SubstrateReference{
					{SubstrateType: "medicineuse_change", ReferenceID: subRefID, Description: "GP letter 2026-05-01"},
				},
				SuspectedCauses: []string{"diuretic_escalation"},
				RelatedEvents:   []uuid.UUID{evt2ID},
			},
			{
				EventID:         evt2ID,
				Timestamp:       start.Add(72 * time.Hour),
				EventType:       "intake_decline",
				PrimitiveType:   PrimitiveIntakeDecline,
				Severity:        2,
				Description:     "Reduced PO intake first noted",
				SubstrateRefs:   []SubstrateReference{},
				SuspectedCauses: []string{},
				RelatedEvents:   []uuid.UUID{evt1ID},
			},
		},
		Patterns: []TemporalPattern{
			{
				PatternID:     "volume_contraction_cascade",
				EventSequence: []uuid.UUID{evt1ID, evt2ID},
				Reasoning:     "Diuretic escalation followed by reduced intake",
				Confidence:    0.65,
			},
		},
		Severity: 4,
		AudienceAdaptations: map[AudienceClass]ChronologyRendering{
			AudiencePharmacist: {
				Audience:          AudiencePharmacist,
				Headline:          "Volume-contraction cascade — frusemide origin",
				Narrative:         "The cascade origin is the frusemide dose change ...",
				HighlightedEvents: []uuid.UUID{evt1ID},
			},
			AudienceRACHOperator: {
				Audience:          AudienceRACHOperator,
				Headline:          "Mobility decline + confusion onset",
				Narrative:         "Operational response: monitoring escalation ...",
				HighlightedEvents: []uuid.UUID{evt2ID},
			},
		},
	}

	encoded, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("marshal failed: %v", err)
	}

	var decoded InstabilityChronology
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// Spot-check survival of each field that could plausibly lose information.
	if decoded.ResidentID != residentID {
		t.Errorf("ResidentID drift: %v vs %v", decoded.ResidentID, residentID)
	}
	if !decoded.TimeWindow.Start.Equal(start) || !decoded.TimeWindow.End.Equal(end) {
		t.Errorf("TimeWindow drift: %+v vs %+v", decoded.TimeWindow, original.TimeWindow)
	}
	if len(decoded.Events) != 2 {
		t.Fatalf("Events length = %d, want 2", len(decoded.Events))
	}
	if decoded.Events[0].PrimitiveType != PrimitiveMedicationChange {
		t.Errorf("Events[0].PrimitiveType drift: %q", decoded.Events[0].PrimitiveType)
	}
	if !decoded.Events[0].Timestamp.Equal(original.Events[0].Timestamp) {
		t.Errorf("Events[0].Timestamp drift: %v vs %v", decoded.Events[0].Timestamp, original.Events[0].Timestamp)
	}
	if len(decoded.Events[0].SubstrateRefs) != 1 || decoded.Events[0].SubstrateRefs[0].ReferenceID != subRefID {
		t.Errorf("SubstrateRefs drift: %+v", decoded.Events[0].SubstrateRefs)
	}
	if len(decoded.Events[0].RelatedEvents) != 1 || decoded.Events[0].RelatedEvents[0] != evt2ID {
		t.Errorf("RelatedEvents drift: %+v", decoded.Events[0].RelatedEvents)
	}
	if len(decoded.Patterns) != 1 || decoded.Patterns[0].Confidence != 0.65 {
		t.Errorf("Patterns drift: %+v", decoded.Patterns)
	}

	// AudienceAdaptations: verify keys survive as canonical AudienceClass values
	if len(decoded.AudienceAdaptations) != 2 {
		t.Fatalf("AudienceAdaptations length = %d, want 2", len(decoded.AudienceAdaptations))
	}
	pharm, ok := decoded.AudienceAdaptations[AudiencePharmacist]
	if !ok {
		t.Fatal("AudiencePharmacist key did not survive round-trip")
	}
	if pharm.Audience != AudiencePharmacist || pharm.Headline == "" {
		t.Errorf("Pharmacist rendering drift: %+v", pharm)
	}
	rach, ok := decoded.AudienceAdaptations[AudienceRACHOperator]
	if !ok {
		t.Fatal("AudienceRACHOperator key did not survive round-trip")
	}
	if len(rach.HighlightedEvents) != 1 || rach.HighlightedEvents[0] != evt2ID {
		t.Errorf("RACH HighlightedEvents drift: %+v", rach.HighlightedEvents)
	}
}

// TestPatternsSliceOrderPreservedInJSON documents Go's JSON guarantee for
// slice ordering and warns future consumers not to rely on map ordering
// instead. JSON arrays preserve order; JSON objects (Go maps) do not.
func TestPatternsSliceOrderPreservedInJSON(t *testing.T) {
	patterns := []TemporalPattern{
		{PatternID: "alpha", Confidence: 0.1},
		{PatternID: "beta", Confidence: 0.2},
		{PatternID: "gamma", Confidence: 0.3},
		{PatternID: "delta", Confidence: 0.4},
	}
	chronology := InstabilityChronology{
		ResidentID: uuid.New(),
		Patterns:   patterns,
	}

	for i := 0; i < 5; i++ {
		encoded, err := json.Marshal(chronology)
		if err != nil {
			t.Fatalf("marshal failed: %v", err)
		}
		var decoded InstabilityChronology
		if err := json.Unmarshal(encoded, &decoded); err != nil {
			t.Fatalf("unmarshal failed: %v", err)
		}
		if len(decoded.Patterns) != len(patterns) {
			t.Fatalf("Patterns length drift on iter %d", i)
		}
		for j, p := range patterns {
			if decoded.Patterns[j].PatternID != p.PatternID {
				t.Errorf("iter %d: Patterns[%d].PatternID = %q, want %q (slice order not preserved)",
					i, j, decoded.Patterns[j].PatternID, p.PatternID)
			}
		}
	}
}
