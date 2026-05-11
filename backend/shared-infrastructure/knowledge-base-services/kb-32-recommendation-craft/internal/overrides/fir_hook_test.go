package overrides_test

// fir_hook_test.go — covers the CAPE substrate fan-out from the override
// store's rule-aware CreateForRule path into the failed_interventions
// substrate. Validates that:
//   - The hook fires only when ruleID is non-empty AND ClassifyInterventionType
//     succeeds AND the ReasonCode maps to a reversal outcome.
//   - The plain Create path (no ruleID) NEVER writes a FIR row.
//   - Unknown rule prefixes and non-reversal reason codes are silently skipped.

import (
	"context"
	"testing"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/overrides"
	fi "github.com/cardiofit/shared/v2_substrate/failed_interventions"
)

func newWiredStore() (*overrides.InMemoryStore, *fi.InMemoryStore) {
	firStore := fi.NewInMemoryStore()
	ovStore := overrides.NewInMemoryStore().WithFailedInterventionStore(firStore)
	return ovStore, firStore
}

func TestFIRHook_CreateForRule_ReversalReasonCode_WritesRecord(t *testing.T) {
	t.Parallel()
	ovStore, firStore := newWiredStore()
	ctx := context.Background()

	r := overrides.OverrideReason{
		RecommendationID:    uuid.New().String(),
		ReasonCode:          "goals_of_care_aligned",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "Family confirmed comfort-focused care; antipsychotic reinstated.",
		CapturedBy:          uuid.New().String(),
	}
	if _, err := ovStore.CreateForRule(ctx, r, "STOP_PSYCH_RISPERIDONE_BPSD"); err != nil {
		t.Fatalf("CreateForRule: %v", err)
	}
	// FIR rows are written with uuid.Nil ResidentID (documented limitation).
	got, err := firStore.ListByResident(ctx, uuid.Nil)
	if err != nil {
		t.Fatalf("ListByResident: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d FIR records; want 1", len(got))
	}
	if got[0].InterventionType != "antipsychotic_deprescribing" {
		t.Errorf("InterventionType=%q; want antipsychotic_deprescribing", got[0].InterventionType)
	}
	if got[0].Outcome != fi.OutcomeGoalsOfCareAligned {
		t.Errorf("Outcome=%q; want %q", got[0].Outcome, fi.OutcomeGoalsOfCareAligned)
	}
}

func TestFIRHook_CreateForRule_NonReversalReason_NoFIRWrite(t *testing.T) {
	t.Parallel()
	ovStore, firStore := newWiredStore()
	ctx := context.Background()
	r := overrides.OverrideReason{
		RecommendationID:    uuid.New().String(),
		ReasonCode:          "alert_fatigue", // Wright/McCoy foundation — NOT a reversal
		AppropriatenessFlag: "inappropriate_override",
		Reasoning:           "Alert fires too often.",
		CapturedBy:          uuid.New().String(),
	}
	if _, err := ovStore.CreateForRule(ctx, r, "STOP_PSYCH_RISPERIDONE_BPSD"); err != nil {
		t.Fatalf("CreateForRule: %v", err)
	}
	got, _ := firStore.ListByResident(ctx, uuid.Nil)
	if len(got) != 0 {
		t.Errorf("got %d FIR records; want 0 (non-reversal reason code)", len(got))
	}
}

func TestFIRHook_CreateForRule_UnclassifiableRule_NoFIRWrite(t *testing.T) {
	t.Parallel()
	ovStore, firStore := newWiredStore()
	ctx := context.Background()
	r := overrides.OverrideReason{
		RecommendationID:    uuid.New().String(),
		ReasonCode:          "goals_of_care_aligned",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "Some monitoring-rule override.",
		CapturedBy:          uuid.New().String(),
	}
	// MONITOR_* rules don't classify → no FIR write even with a reversal reason.
	if _, err := ovStore.CreateForRule(ctx, r, "MONITOR_LITHIUM_LEVEL"); err != nil {
		t.Fatalf("CreateForRule: %v", err)
	}
	got, _ := firStore.ListByResident(ctx, uuid.Nil)
	if len(got) != 0 {
		t.Errorf("got %d FIR records; want 0 (unclassified rule prefix)", len(got))
	}
}

func TestFIRHook_PlainCreate_NeverWritesFIR(t *testing.T) {
	t.Parallel()
	ovStore, firStore := newWiredStore()
	ctx := context.Background()
	r := overrides.OverrideReason{
		RecommendationID:    uuid.New().String(),
		ReasonCode:          "goals_of_care_aligned", // would be a reversal if rule context existed
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "GoC discussion.",
		CapturedBy:          uuid.New().String(),
	}
	// Plain Create — no ruleID — must NOT write a FIR row (CAPE substrate
	// design: rule context is required to derive InterventionType).
	if _, err := ovStore.Create(ctx, r); err != nil {
		t.Fatalf("Create: %v", err)
	}
	got, _ := firStore.ListByResident(ctx, uuid.Nil)
	if len(got) != 0 {
		t.Errorf("got %d FIR records via plain Create; want 0", len(got))
	}
}

func TestFIRHook_NilFIRStore_NoCrash(t *testing.T) {
	t.Parallel()
	// Bare InMemoryStore with no FIR wired — must not panic on CreateForRule.
	ovStore := overrides.NewInMemoryStore()
	ctx := context.Background()
	_, err := ovStore.CreateForRule(ctx, overrides.OverrideReason{
		RecommendationID:    uuid.New().String(),
		ReasonCode:          "goals_of_care_aligned",
		AppropriatenessFlag: "appropriate_override",
		Reasoning:           "test.",
		CapturedBy:          uuid.New().String(),
	}, "STOP_PSYCH_HALOPERIDOL")
	if err != nil {
		t.Fatalf("CreateForRule with nil FIR store: %v", err)
	}
}
