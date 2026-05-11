package aggregation

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildCAPEContextBand_EmptyForNonWorklist(t *testing.T) {
	paths := []EntryPath{EntryPathSearch, EntryPathNotification, EntryPathCrossReference, EntryPath("")}
	for _, p := range paths {
		band, err := BuildCAPEContextBand(EntryPathMetadata{Path: p})
		if err != nil {
			t.Errorf("path %q: unexpected error %v", p, err)
		}
		if len(band.Signals) != 0 || len(band.SubstrateRefs) != 0 {
			t.Errorf("path %q: expected empty band, got %+v", p, band)
		}
	}
}

func TestBuildCAPEContextBand_RendersKnownSignals(t *testing.T) {
	triaged := time.Now().Add(-2 * time.Minute).UTC()
	band, err := BuildCAPEContextBand(EntryPathMetadata{
		Path: EntryPathWorklist,
		Context: WorklistContext{
			PrimarySignals: []string{
				"acute_event_severity_5_fall",
				"trajectory_velocity_4_egfr_decline",
			},
			CAPEScore: 0.91,
			TriagedAt: triaged,
		},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(band.Signals) != 2 {
		t.Fatalf("expected 2 signals, got %d", len(band.Signals))
	}
	if band.Signals[0].HumanReadable == band.Signals[0].Code {
		t.Errorf("known signal %q rendered verbatim; expected human-readable mapping", band.Signals[0].Code)
	}
	if band.Signals[0].Severity != 5 {
		t.Errorf("severity = %d, want 5", band.Signals[0].Severity)
	}
	if band.CAPEScore != 0.91 {
		t.Errorf("CAPEScore not preserved")
	}
	if !band.TriagedAt.Equal(triaged) {
		t.Errorf("TriagedAt not preserved")
	}
	// Verification-not-belief: every signal carries at least one substrate ref.
	if len(band.SubstrateRefs) != len(band.Signals) {
		t.Errorf("expected one SubstrateRef per signal; got %d refs for %d signals", len(band.SubstrateRefs), len(band.Signals))
	}
}

func TestBuildCAPEContextBand_RendersUnknownSignalsVerbatim(t *testing.T) {
	band, err := BuildCAPEContextBand(EntryPathMetadata{
		Path: EntryPathWorklist,
		Context: WorklistContext{
			PrimarySignals: []string{"some_future_signal_kb33_will_emit"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(band.Signals) != 1 {
		t.Fatal("expected one signal")
	}
	if band.Signals[0].HumanReadable != "some_future_signal_kb33_will_emit" {
		t.Errorf("unknown signal should render verbatim; got %q", band.Signals[0].HumanReadable)
	}
	if band.Signals[0].Severity != 0 {
		t.Errorf("unknown signal severity should be 0; got %d", band.Signals[0].Severity)
	}
}

func TestBuildCAPEContextBand_RejectsWrongContextType(t *testing.T) {
	_, err := BuildCAPEContextBand(EntryPathMetadata{
		Path:    EntryPathWorklist,
		Context: SearchContext{Query: "x"},
	})
	if err == nil {
		t.Fatal("expected error when worklist path has non-WorklistContext")
	}
}

// TestSubstrateRefSentinelPresentForEachSignal asserts the
// verification-not-belief discipline (v1.0 Part 10) holds even when kb-33
// has not yet supplied real substrate IDs.
func TestSubstrateRefSentinelPresentForEachSignal(t *testing.T) {
	band, _ := BuildCAPEContextBand(EntryPathMetadata{
		Path: EntryPathWorklist,
		Context: WorklistContext{
			PrimarySignals: []string{"acute_event_severity_5_fall", "monitoring_overdue_lithium_level"},
		},
	})
	if len(band.SubstrateRefs) < len(band.Signals) {
		t.Fatalf("fewer SubstrateRefs (%d) than signals (%d) — verification-not-belief violated", len(band.SubstrateRefs), len(band.Signals))
	}
	for _, r := range band.SubstrateRefs {
		if r.Source == "" {
			t.Errorf("SubstrateRef missing Source")
		}
	}
	_ = uuid.Nil // keep import in case future tests need it
}
