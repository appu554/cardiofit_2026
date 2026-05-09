// Package clinical_safety_test validates the immutability guarantee of
// fire-time citation pinning: amending a source after a recommendation fires
// must NOT retroactively change the pinned version.
//
// Recommendation Craft Guidelines Part 13 — clinical safety test category.
// VisibilityClass: AD — citation versioning per Guidelines §6 audit defensibility
package clinical_safety_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/citations"
)

// TestCitationVersioning_PinThenAmendDoesNotRetroactivelyChange asserts the
// core audit-defensibility invariant: a recommendation pinned at fire time to
// source version v1 must still resolve to v1 even after the source is amended
// to v2.
//
// This test uses InMemoryRegistry to avoid any Postgres dependency and is safe
// for execution in standard CI without infrastructure.
//
// Guidelines §6 hard cap: pinned citations are immutable — subsequent amendments
// or retractions of the source MUST NOT modify already-pinned recommendations.
func TestCitationVersioning_PinThenAmendDoesNotRetroactivelyChange(t *testing.T) {
	t.Parallel()

	reg := citations.NewInMemoryRegistry()
	ctx := context.Background()
	recID := uuid.New().String()
	sourceID := "ADGuideline2025"

	// ── Step 1: Register source at v1 with a fixed fire time ─────────────────
	fireTime := time.Date(2025, 6, 1, 12, 0, 0, 0, time.UTC)
	v1 := citations.SourceVersion{
		SourceID:      sourceID,
		Version:       "v1",
		EffectiveFrom: fireTime.Add(-24 * time.Hour), // active before fire time
		EffectiveTo:   nil,
		ContentHash:   "hash-v1",
		Status:        citations.StatusActive,
	}
	if err := reg.Register(ctx, v1); err != nil {
		t.Fatalf("Register v1: %v", err)
	}

	// ── Step 2: Pin citations at fire time ────────────────────────────────────
	citationsList, err := citations.PinAtFireTime(ctx, reg, recID, []string{sourceID}, fireTime)
	if err != nil {
		t.Fatalf("PinAtFireTime: %v", err)
	}
	if len(citationsList) != 1 {
		t.Fatalf("expected 1 citation pinned; got %d", len(citationsList))
	}
	if citationsList[0].Version != "v1" {
		t.Fatalf("expected citation pinned to v1; got %q", citationsList[0].Version)
	}

	// ── Step 3: Amend source to v2 (after fire time) ──────────────────────────
	amendTime := fireTime.Add(1 * time.Hour)
	if err := reg.Amend(ctx, sourceID, "v2", "hash-v2", amendTime); err != nil {
		t.Fatalf("Amend to v2: %v", err)
	}

	// ── Step 4: Verify original citation still resolves to v1 ────────────────
	// The pinned citation record is immutable; its Version field must not change
	// because of the amendment.
	pinned := citationsList[0]
	if pinned.Version != "v1" {
		t.Errorf(
			"amendment retroactively changed pinned citation: expected v1, got %q — "+
				"audit-defensibility invariant violated (Guidelines §6)",
			pinned.Version,
		)
	}

	// ── Step 5: Confirm the registry lookup also returns the original pin ─────
	// GetCitation must find the v1 record because SaveCitation persisted it at
	// fire time. If the in-memory store mutated it, this will fail.
	retrieved, err := reg.GetCitation(ctx, recID, sourceID, "v1")
	if err != nil {
		t.Fatalf("GetCitation(v1): %v — stored citation must remain retrievable after amendment", err)
	}
	if retrieved.Version != "v1" {
		t.Errorf("retrieved citation version is %q; expected v1", retrieved.Version)
	}
	if retrieved.PinnedAt != fireTime {
		t.Errorf("retrieved citation PinnedAt is %v; expected %v", retrieved.PinnedAt, fireTime)
	}
}

// TestCitationVersioning_NewRecommendationGetsAmendedVersion asserts the
// complementary property: a NEW recommendation generated after the amendment
// DOES receive the updated v2 citation.
//
// This is the inverse of the immutability test: existing pins stay at v1;
// new fire events see v2.
func TestCitationVersioning_NewRecommendationGetsAmendedVersion(t *testing.T) {
	t.Parallel()

	reg := citations.NewInMemoryRegistry()
	ctx := context.Background()
	sourceID := "ADGuideline2025"

	// Register v1.
	v1Start := time.Date(2025, 6, 1, 0, 0, 0, 0, time.UTC)
	v1 := citations.SourceVersion{
		SourceID:      sourceID,
		Version:       "v1",
		EffectiveFrom: v1Start,
		ContentHash:   "hash-v1",
		Status:        citations.StatusActive,
	}
	if err := reg.Register(ctx, v1); err != nil {
		t.Fatalf("Register v1: %v", err)
	}

	// Amend to v2.
	amendTime := time.Date(2025, 6, 10, 0, 0, 0, 0, time.UTC)
	if err := reg.Amend(ctx, sourceID, "v2", "hash-v2", amendTime); err != nil {
		t.Fatalf("Amend to v2: %v", err)
	}

	// New recommendation fires after amendment.
	newRecID := uuid.New().String()
	newFireTime := amendTime.Add(1 * time.Hour)
	newCitations, err := citations.PinAtFireTime(ctx, reg, newRecID, []string{sourceID}, newFireTime)
	if err != nil {
		t.Fatalf("PinAtFireTime for new recommendation: %v", err)
	}
	if len(newCitations) != 1 {
		t.Fatalf("expected 1 citation for new recommendation; got %d", len(newCitations))
	}
	if newCitations[0].Version != "v2" {
		t.Errorf(
			"new recommendation should be pinned to v2; got %q — "+
				"forward citation update not working correctly",
			newCitations[0].Version,
		)
	}
}
