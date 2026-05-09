package algorithmic_distinction

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ── Plan tests (verbatim from Task 3) ────────────────────────────────────────

func TestObservation_FourClasses(t *testing.T) {
	for _, c := range []Class{ClassSubstrateFact, ClassPlatformSuggestion, ClassPharmacistReflection, ClassHybrid} {
		o := Observation{ID: uuid.New(), Class: c, Body: "x"}
		if !o.Class.Valid() {
			t.Errorf("class %v should be valid", c)
		}
	}
	// Unknown class invalid.
	if Class("nope").Valid() {
		t.Errorf("unknown class should be invalid")
	}
}

func TestObservation_ConfirmTransitionsSuggestionToHybrid(t *testing.T) {
	o := Observation{ID: uuid.New(), Class: ClassPlatformSuggestion, Body: "Your deprescribing acceptance is changing."}
	confirmed, err := o.Confirm(uuid.New(), time.Now())
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Class != ClassHybrid {
		t.Errorf("class after confirm = %v, want hybrid", confirmed.Class)
	}
	if confirmed.ConfirmedBy == nil || confirmed.ConfirmedAt == nil {
		t.Errorf("expected ConfirmedBy and ConfirmedAt set")
	}
}

func TestObservation_ConfirmRejectsNonSuggestion(t *testing.T) {
	o := Observation{ID: uuid.New(), Class: ClassSubstrateFact}
	if _, err := o.Confirm(uuid.New(), time.Now()); err == nil {
		t.Errorf("confirm on substrate-fact should reject")
	}
}

func TestMarkers_RenderEmoji(t *testing.T) {
	if Marker(ClassSubstrateFact) != "🔵" {
		t.Errorf("substrate-fact marker = %q", Marker(ClassSubstrateFact))
	}
	if Marker(ClassHybrid) != "🟣" {
		t.Errorf("hybrid marker = %q", Marker(ClassHybrid))
	}
	_ = context.Background()
}

// ── Augmentation tests ────────────────────────────────────────────────────────

// Augmentation 1: IsValidClass package-level helper mirrors Phase 1a conventions.
func TestIsValidClass(t *testing.T) {
	valid := []string{"substrate_fact", "platform_suggestion", "pharmacist_reflection", "hybrid"}
	for _, s := range valid {
		if !IsValidClass(s) {
			t.Errorf("IsValidClass(%q) = false, want true", s)
		}
	}
	invalid := []string{"", "unknown", "Hybrid", "SUBSTRATE_FACT"}
	for _, s := range invalid {
		if IsValidClass(s) {
			t.Errorf("IsValidClass(%q) = true, want false", s)
		}
	}
}

// Augmentation 3: IsConfirmed helper returns true only for hybrid with non-nil fields.
func TestObservation_IsConfirmed(t *testing.T) {
	byID := uuid.New()
	at := time.Now().UTC()

	hybrid := Observation{
		ID:          uuid.New(),
		Class:       ClassHybrid,
		ConfirmedBy: &byID,
		ConfirmedAt: &at,
	}
	if !hybrid.IsConfirmed() {
		t.Errorf("hybrid with ConfirmedBy+ConfirmedAt should be confirmed")
	}

	// Hybrid with nil fields is not confirmed.
	partialHybrid := Observation{ID: uuid.New(), Class: ClassHybrid}
	if partialHybrid.IsConfirmed() {
		t.Errorf("hybrid without ConfirmedBy/ConfirmedAt should not be confirmed")
	}

	// Platform suggestion is never confirmed even with fields set.
	suggestion := Observation{
		ID:          uuid.New(),
		Class:       ClassPlatformSuggestion,
		ConfirmedBy: &byID,
		ConfirmedAt: &at,
	}
	if suggestion.IsConfirmed() {
		t.Errorf("platform suggestion should not be considered confirmed")
	}
}

// Augmentation 4: Marker with unknown class returns "" (no panic).
func TestMarker_Unknown(t *testing.T) {
	got := Marker(Class("garbage"))
	if got != "" {
		t.Errorf("Marker(unknown) = %q, want empty string", got)
	}
}
