package reflection

import (
	"context"
	"errors"
	"reflect"
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Shared fake Signals implementation
// ---------------------------------------------------------------------------

type fakeSignals struct {
	recentOverrides int
	typeMix         map[string]int
}

func (f *fakeSignals) RestraintOverridesIn(_ context.Context, _ uuid.UUID, _ int) (int, error) {
	return f.recentOverrides, nil
}
func (f *fakeSignals) RecommendationTypeMix(_ context.Context, _ uuid.UUID, _ int) (map[string]int, error) {
	return f.typeMix, nil
}

// ---------------------------------------------------------------------------
// Plan-provided tests (verbatim from Task 2 spec)
// ---------------------------------------------------------------------------

func TestPromptSelector_RotatesMonthly(t *testing.T) {
	lib := DefaultPromptLibrary()
	if len(lib) < 4 {
		t.Fatalf("library should have ≥4 curated prompts, got %d", len(lib))
	}
	sel := NewSelector(lib, &fakeSignals{recentOverrides: 0, typeMix: nil})

	pharmacist := uuid.New()
	got1, _ := sel.Select(context.Background(), pharmacist, 2026, 5)
	got2, _ := sel.Select(context.Background(), pharmacist, 2026, 6)
	if got1.ID == got2.ID {
		t.Errorf("expected different prompts in different months")
	}
}

func TestPromptSelector_AdaptsToRestraintOverrideSignal(t *testing.T) {
	lib := DefaultPromptLibrary()
	sel := NewSelector(lib, &fakeSignals{recentOverrides: 5, typeMix: nil})
	got, _ := sel.Select(context.Background(), uuid.New(), 2026, 5)
	if !got.HasTag("restraint") {
		t.Errorf("expected restraint-themed prompt for override-active pharmacist; got tags=%v", got.Tags)
	}
}

// ---------------------------------------------------------------------------
// Augmented tests
// ---------------------------------------------------------------------------

// TestPromptSelector_DeterministicForSameInputs verifies that Select() returns
// the same Prompt when called twice with identical (pharmacist, year, month)
// and no signal trigger. The FNV-1a hash must be stable across calls.
func TestPromptSelector_DeterministicForSameInputs(t *testing.T) {
	lib := DefaultPromptLibrary()
	sig := &fakeSignals{recentOverrides: 0, typeMix: nil}
	sel := NewSelector(lib, sig)

	pharmacist := uuid.New()
	got1, err1 := sel.Select(context.Background(), pharmacist, 2026, 5)
	got2, err2 := sel.Select(context.Background(), pharmacist, 2026, 5)
	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected errors: %v / %v", err1, err2)
	}
	if got1.ID != got2.ID {
		t.Errorf("Select is not deterministic: first=%s second=%s", got1.ID, got2.ID)
	}
}

// TestPromptSelector_DifferentPharmacistsDifferentPrompts verifies that the
// hash-based rotation produces diversity across pharmacist IDs. A cohort of 20
// distinct pharmacists is sampled; at least 2 distinct prompts must appear to
// confirm the hash is not degenerate.
func TestPromptSelector_DifferentPharmacistsDifferentPrompts(t *testing.T) {
	lib := DefaultPromptLibrary()
	sig := &fakeSignals{recentOverrides: 0, typeMix: nil}
	sel := NewSelector(lib, sig)

	seen := make(map[uuid.UUID]struct{})
	const cohort = 20
	for i := 0; i < cohort; i++ {
		got, err := sel.Select(context.Background(), uuid.New(), 2026, 5)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		seen[got.ID] = struct{}{}
	}
	if len(seen) < 2 {
		t.Errorf("expected diversity across %d pharmacists, got only %d distinct prompt(s)", cohort, len(seen))
	}
}

// TestPromptSelector_RestraintTriggerOverridesRotation verifies that a strong
// restraint-override signal selects the restraint-tagged prompt regardless of
// (year, month). Tested for two different months to rule out a coincidental
// alignment with the default rotation result.
func TestPromptSelector_RestraintTriggerOverridesRotation(t *testing.T) {
	lib := DefaultPromptLibrary()
	sig := &fakeSignals{recentOverrides: 5, typeMix: nil}
	sel := NewSelector(lib, sig)

	pharmacist := uuid.New()
	months := []int{3, 9} // two different months
	for _, m := range months {
		got, err := sel.Select(context.Background(), pharmacist, 2026, m)
		if err != nil {
			t.Fatalf("month=%d unexpected error: %v", m, err)
		}
		if !got.HasTag("restraint") {
			t.Errorf("month=%d: expected restraint-tagged prompt when recentOverrides=5, got tags=%v", m, got.Tags)
		}
	}
}

// TestPromptSelector_NoLibraryPanics verifies that an empty prompt library
// returns ErrEmptyLibrary and does NOT panic.
func TestPromptSelector_NoLibraryPanics(t *testing.T) {
	sel := NewSelector([]Prompt{}, &fakeSignals{recentOverrides: 0})
	_, err := sel.Select(context.Background(), uuid.New(), 2026, 5)
	if !errors.Is(err, ErrEmptyLibrary) {
		t.Errorf("expected ErrEmptyLibrary for empty library, got %v", err)
	}
}

// TestPromptSelector_DoesNotConsultReflectiveEntries is a structural interface
// test asserting that the Signals interface has exactly two methods and does
// NOT include any method that would allow access to reflective entries.
//
// This enforces the POA isolation constraint from Self-Visibility Guidelines
// §6.4: prompt selection must never read reflective entries. The entries are
// Pharmacist-Only-Always (POA) — safe-space reflective writing that must
// remain architecturally isolated from all algorithmic selection.
//
// If Signals ever acquired a reflective-entry method (e.g. GetReflectiveEntries,
// ListEntries), this test will fail, alerting the implementer to the violation.
func TestPromptSelector_DoesNotConsultReflectiveEntries(t *testing.T) {
	// Compile-time proof: fakeSignals must satisfy Signals without any
	// reflective-entry method. If Signals grew such a method, fakeSignals
	// would fail to compile here, surfacing the violation at build time.
	var _ Signals = (*fakeSignals)(nil)

	// Runtime proof: assert the Signals interface has exactly 2 methods.
	// The only permitted methods are the two activity-signal accessors below.
	permittedMethods := map[string]bool{
		"RestraintOverridesIn":  true,
		"RecommendationTypeMix": true,
	}

	sigType := reflect.TypeOf((*Signals)(nil)).Elem()
	if sigType.NumMethod() != len(permittedMethods) {
		t.Errorf("Signals interface has %d methods (expected %d). "+
			"POA isolation: Signals must NOT include any reflective-entry method. "+
			"See Self-Visibility Guidelines §6.4.",
			sigType.NumMethod(), len(permittedMethods))
	}
	for i := 0; i < sigType.NumMethod(); i++ {
		name := sigType.Method(i).Name
		if !permittedMethods[name] {
			t.Errorf("Signals interface contains unexpected method %q — "+
				"this may violate POA isolation (Guidelines §6.4)", name)
		}
	}
}
