package reconciliation

import (
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestBuildWorklistInputs_SkipsUnchangedAndAppliesDueWindow(t *testing.T) {
	docRef := uuid.New()
	resRef := uuid.New()
	assigned := uuid.New()
	facility := uuid.New()
	dischargeAt := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)

	pre := []DiffEntry{
		{Class: DiffUnchanged},
		{Class: DiffNewMedication, DischargeLineMedicine: &DischargeLineSummary{IndicationText: "post-op infection"}},
		{Class: DiffCeasedMedication},
	}

	got := BuildWorklistInputs(docRef, resRef, &assigned, &facility, dischargeAt, 0, pre, nil)

	if got.DischargeDocumentRef != docRef || got.ResidentRef != resRef {
		t.Fatalf("ref mismatch")
	}
	if got.DueAt != dischargeAt.Add(DefaultWorklistDueWindow) {
		t.Fatalf("due window mismatch: got %v", got.DueAt)
	}
	if len(got.Decisions) != 2 {
		t.Fatalf("unchanged should be filtered, want 2 decisions got %d", len(got.Decisions))
	}
	// First decision is the new_medication with acute keyword text.
	if got.Decisions[0].IntentClass != IntentAcuteTemporary {
		t.Errorf("classifier should fire acute on post-op infection, got %s",
			got.Decisions[0].IntentClass)
	}
	if got.Decisions[1].IntentClass != IntentUnclear {
		t.Errorf("ceased should classify unclear, got %s", got.Decisions[1].IntentClass)
	}
}

func TestBuildWorklistInputs_CustomTextResolver(t *testing.T) {
	dischargeAt := time.Date(2025, 6, 1, 10, 0, 0, 0, time.UTC)
	d := DiffEntry{Class: DiffNewMedication, DischargeLineMedicine: &DischargeLineSummary{}}
	resolver := func(_ DiffEntry) string { return "started for ongoing chronic management" }

	got := BuildWorklistInputs(uuid.New(), uuid.New(), nil, nil, dischargeAt, time.Hour, []DiffEntry{d}, resolver)
	if got.DueAt != dischargeAt.Add(time.Hour) {
		t.Fatalf("custom dueWindow ignored: got %v", got.DueAt)
	}
	if len(got.Decisions) != 1 {
		t.Fatalf("want 1 decision")
	}
	if got.Decisions[0].IntentClass != IntentNewChronic {
		t.Errorf("resolver text should classify new_chronic, got %s", got.Decisions[0].IntentClass)
	}
}
