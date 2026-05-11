package entry_paths

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestFromWorklist_HappyPath(t *testing.T) {
	pharm := uuid.New()
	res := uuid.New()
	triaged := time.Now().Add(-5 * time.Minute).UTC()
	meta, err := FromWorklist(context.Background(), pharm, res, WorklistContext{
		PrimarySignals: []string{"acute_event_severity_5_fall"},
		CAPEScore:      0.87,
		TriagedAt:      triaged,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if meta.Path != EntryPathWorklist {
		t.Errorf("Path = %q, want worklist", meta.Path)
	}
	if meta.PharmacistID != pharm || meta.ResidentID != res {
		t.Errorf("identity not preserved")
	}
	wc, ok := meta.Context.(WorklistContext)
	if !ok {
		t.Fatalf("Context not WorklistContext: %T", meta.Context)
	}
	if !wc.TriagedAt.Equal(triaged) {
		t.Errorf("TriagedAt overwritten: got %v want %v", wc.TriagedAt, triaged)
	}
	if wc.Kind() != EntryPathWorklist {
		t.Errorf("Kind() = %q", wc.Kind())
	}
}

func TestFromWorklist_DefaultsTriagedAt(t *testing.T) {
	meta, err := FromWorklist(context.Background(), uuid.New(), uuid.New(), WorklistContext{
		PrimarySignals: []string{"trajectory_velocity_4_egfr_decline"},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	wc := meta.Context.(WorklistContext)
	if wc.TriagedAt.IsZero() {
		t.Errorf("TriagedAt should have defaulted to now")
	}
}

func TestFromWorklist_RejectsZeroPharmacist(t *testing.T) {
	_, err := FromWorklist(context.Background(), uuid.Nil, uuid.New(), WorklistContext{
		PrimarySignals: []string{"x"},
	})
	if err == nil {
		t.Fatal("expected error for zero pharmacist_id")
	}
}

func TestFromWorklist_RejectsZeroResident(t *testing.T) {
	_, err := FromWorklist(context.Background(), uuid.New(), uuid.Nil, WorklistContext{
		PrimarySignals: []string{"x"},
	})
	if err == nil {
		t.Fatal("expected error for zero resident_id")
	}
}

func TestFromWorklist_RejectsEmptySignals(t *testing.T) {
	_, err := FromWorklist(context.Background(), uuid.New(), uuid.New(), WorklistContext{})
	if err == nil {
		t.Fatal("expected error for empty PrimarySignals — worklist entry without signals is nonsense")
	}
}
