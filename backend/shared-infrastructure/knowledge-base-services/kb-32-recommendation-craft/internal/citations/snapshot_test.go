package citations

import (
	"context"
	"errors"
	"testing"
	"time"
)

// pinTime is the reference "recommendation fire time" used across snapshot tests.
var pinTime = time.Date(2026, 5, 8, 14, 0, 0, 0, time.UTC)

// ---------------------------------------------------------------------------
// Helper: register a source version active at a given time window
// ---------------------------------------------------------------------------

func mustRegister(t *testing.T, reg *InMemoryRegistry, sv SourceVersion) {
	t.Helper()
	ctx := context.Background()
	if err := reg.Register(ctx, sv); err != nil {
		t.Fatalf("mustRegister: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Empty anchors → empty (non-nil) slice, no error
// ---------------------------------------------------------------------------

func TestPinAtFireTime_EmptyAnchors(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	citations, err := PinAtFireTime(ctx, reg, "rec-empty", nil, pinTime)
	if err != nil {
		t.Fatalf("PinAtFireTime(nil anchors): unexpected error: %v", err)
	}
	if citations == nil {
		t.Fatal("PinAtFireTime(nil anchors): returned nil slice, want non-nil empty slice")
	}
	if len(citations) != 0 {
		t.Errorf("PinAtFireTime(nil anchors): got %d citations, want 0", len(citations))
	}

	// Explicit empty slice.
	citations, err = PinAtFireTime(ctx, reg, "rec-empty2", []string{}, pinTime)
	if err != nil {
		t.Fatalf("PinAtFireTime([] anchors): unexpected error: %v", err)
	}
	if citations == nil {
		t.Fatal("PinAtFireTime([] anchors): returned nil slice, want non-nil empty slice")
	}
}

// ---------------------------------------------------------------------------
// Multiple anchors all get correct versions
// ---------------------------------------------------------------------------

func TestPinAtFireTime_MultipleAnchors(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	// Register three distinct sources, each with one active version at pinTime.
	sources := []struct {
		id      string
		version string
	}{
		{"guideline-A", "3"},
		{"drug-db-B", "2024-01"},
		{"safety-ref-C", "1"},
	}
	for _, s := range sources {
		mustRegister(t, reg, SourceVersion{
			SourceID:      s.id,
			Version:       s.version,
			EffectiveFrom: pinTime.Add(-24 * time.Hour),
			Status:        StatusActive,
		})
	}

	anchors := []string{"guideline-A", "drug-db-B", "safety-ref-C"}
	cits, err := PinAtFireTime(ctx, reg, "rec-multi", anchors, pinTime)
	if err != nil {
		t.Fatalf("PinAtFireTime: %v", err)
	}
	if len(cits) != 3 {
		t.Fatalf("PinAtFireTime: got %d citations, want 3", len(cits))
	}

	// Each citation should match the expected version.
	bySource := make(map[string]RecommendationCitation, len(cits))
	for _, c := range cits {
		bySource[c.SourceID] = c
	}
	for _, s := range sources {
		c, ok := bySource[s.id]
		if !ok {
			t.Errorf("citation missing for source %q", s.id)
			continue
		}
		if c.Version != s.version {
			t.Errorf("source %q: version = %q, want %q", s.id, c.Version, s.version)
		}
		if c.RecommendationID != "rec-multi" {
			t.Errorf("source %q: recommendation_id = %q, want %q", s.id, c.RecommendationID, "rec-multi")
		}
		if !c.PinnedAt.Equal(pinTime) {
			t.Errorf("source %q: pinned_at = %v, want %v", s.id, c.PinnedAt, pinTime)
		}
	}
}

// ---------------------------------------------------------------------------
// Pin then Amend — the core audit-defensibility invariant
//
// Scenario: recommendation R cites source S at version v1 (fire time = pinTime).
// Source S is later amended to v2. The original citation for R must still
// resolve to v1, not v2.
// ---------------------------------------------------------------------------

func TestPinAtFireTime_PinThenAmend(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	amendTime := pinTime.Add(30 * 24 * time.Hour) // 30 days after fire time

	// Register v1 of source, active at pinTime.
	mustRegister(t, reg, SourceVersion{
		SourceID:      "src-amend",
		Version:       "1",
		EffectiveFrom: pinTime.Add(-24 * time.Hour),
		Status:        StatusActive,
		ContentHash:   "hash-v1",
	})

	// Pin recommendation at fire time.
	cits, err := PinAtFireTime(ctx, reg, "rec-audit", []string{"src-amend"}, pinTime)
	if err != nil {
		t.Fatalf("PinAtFireTime: %v", err)
	}
	if len(cits) != 1 {
		t.Fatalf("PinAtFireTime: got %d citations, want 1", len(cits))
	}
	pinnedVersion := cits[0].Version
	if pinnedVersion != "1" {
		t.Fatalf("pinned version = %q, want %q", pinnedVersion, "1")
	}

	// Amend source to v2 (30 days later — AFTER the recommendation was fired).
	if err := reg.Amend(ctx, "src-amend", "2", "hash-v2", amendTime); err != nil {
		t.Fatalf("Amend: %v", err)
	}

	// CORE AUDIT INVARIANT: the original citation must still point to v1.
	got, err := reg.GetCitation(ctx, "rec-audit", "src-amend", pinnedVersion)
	if err != nil {
		t.Fatalf("GetCitation after amend: %v", err)
	}
	if got.Version != "1" {
		t.Errorf("after amend: citation version = %q, want %q (amendment must not retroactively change pinned citation)", got.Version, "1")
	}

	// v1 is now amended; verify the registry reflects amendment without
	// affecting the already-pinned citation.
	v1, err := reg.Get(ctx, "src-amend", "1")
	if err != nil {
		t.Fatalf("Get v1 after amend: %v", err)
	}
	if v1.Status != StatusAmended {
		t.Errorf("v1.Status = %q after amend, want %q", v1.Status, StatusAmended)
	}

	// Confirming: the pinned citation's version field is unchanged.
	if got.Version != pinnedVersion {
		t.Errorf("pinned citation version changed from %q to %q (must be immutable)", pinnedVersion, got.Version)
	}
}

// ---------------------------------------------------------------------------
// Pin then Retract — citation persists and surfaces the retracted status
// ---------------------------------------------------------------------------

func TestPinAtFireTime_PinThenRetract(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	mustRegister(t, reg, SourceVersion{
		SourceID:      "src-retract",
		Version:       "1",
		EffectiveFrom: pinTime.Add(-24 * time.Hour),
		Status:        StatusActive,
		ContentHash:   "hash-v1",
	})

	// Pin first.
	cits, err := PinAtFireTime(ctx, reg, "rec-retract", []string{"src-retract"}, pinTime)
	if err != nil {
		t.Fatalf("PinAtFireTime: %v", err)
	}
	if len(cits) != 1 {
		t.Fatalf("PinAtFireTime: got %d citations, want 1", len(cits))
	}
	pinnedVersion := cits[0].Version

	// Retract source AFTER fire time.
	if err := reg.Retract(ctx, "src-retract", "safety concern"); err != nil {
		t.Fatalf("Retract: %v", err)
	}

	// The citation still exists and points to the same version.
	got, err := reg.GetCitation(ctx, "rec-retract", "src-retract", pinnedVersion)
	if err != nil {
		t.Fatalf("GetCitation after retract: %v", err)
	}
	if got.Version != pinnedVersion {
		t.Errorf("citation version changed after retract: got %q, want %q", got.Version, pinnedVersion)
	}

	// The SourceVersion itself is now retracted — the caller can surface this
	// as a "retracted source" flag on the dashboard.
	sv, err := reg.Get(ctx, "src-retract", pinnedVersion)
	if err != nil {
		t.Fatalf("Get source version after retract: %v", err)
	}
	if sv.Status != StatusRetracted {
		t.Errorf("source version status = %q after retract, want %q", sv.Status, StatusRetracted)
	}
}

// ---------------------------------------------------------------------------
// Source has no active version at asOf → ErrNoActiveVersion
// ---------------------------------------------------------------------------

func TestPinAtFireTime_NoActiveVersion(t *testing.T) {
	ctx := context.Background()
	reg := NewInMemoryRegistry()

	// Register a source that only becomes active AFTER pinTime.
	futureStart := pinTime.Add(24 * time.Hour)
	mustRegister(t, reg, SourceVersion{
		SourceID:      "src-future",
		Version:       "1",
		EffectiveFrom: futureStart,
		Status:        StatusActive,
	})

	// Attempt to pin at pinTime — src-future is not yet active.
	_, err := PinAtFireTime(ctx, reg, "rec-future", []string{"src-future"}, pinTime)
	if err == nil {
		t.Fatal("PinAtFireTime: expected error for no active version, got nil")
	}
	if !errors.Is(err, ErrNoActiveVersion) {
		t.Errorf("PinAtFireTime: error = %v, want wrapping ErrNoActiveVersion", err)
	}
}
