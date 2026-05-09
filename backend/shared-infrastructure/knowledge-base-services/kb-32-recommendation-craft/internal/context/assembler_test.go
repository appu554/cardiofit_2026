package context_test

import (
	"context"
	"errors"
	"testing"
	"time"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// InMemorySubstrateClient — test fixture
//
// Implements SubstrateClient with a pre-configured snapshot or a fixed error.
// Other test packages (e.g. generator / Task 6) may embed this type via a
// test-import alias to avoid re-defining the pattern.
// ---------------------------------------------------------------------------

// InMemorySubstrateClient is an in-process test double for SubstrateClient.
// It returns a single pre-loaded ClinicalSnapshot or propagates a fixed error.
type InMemorySubstrateClient struct {
	Snapshot kb32ctx.ClinicalSnapshot
	Err      error
}

// SnapshotFor implements SubstrateClient.
func (c *InMemorySubstrateClient) SnapshotFor(_ context.Context, _ uuid.UUID) (kb32ctx.ClinicalSnapshot, error) {
	if c.Err != nil {
		return kb32ctx.ClinicalSnapshot{}, c.Err
	}
	return c.Snapshot, nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func freshSnapshot(residentID uuid.UUID) kb32ctx.ClinicalSnapshot {
	return kb32ctx.ClinicalSnapshot{
		ResidentID:          residentID,
		EGFR:                52.4,
		DBI:                 1.1,
		ACB:                 3,
		CFS:                 6,
		CareIntensity:       "active",
		RecentFall72h:       true,
		RecentAdmission72h:  false,
		AssessedAt:          time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

func TestAssembler_HappyPath(t *testing.T) {
	rid := uuid.New()
	snap := freshSnapshot(rid)

	client := &InMemorySubstrateClient{Snapshot: snap}
	a := kb32ctx.NewAssembler(client)

	got, err := a.Assemble(context.Background(), rid)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.ResidentID != rid {
		t.Errorf("ResidentID mismatch: want %v, got %v", rid, got.ResidentID)
	}
	if got.EGFR != snap.EGFR {
		t.Errorf("EGFR mismatch: want %v, got %v", snap.EGFR, got.EGFR)
	}
	if got.CareIntensity != snap.CareIntensity {
		t.Errorf("CareIntensity mismatch: want %v, got %v", snap.CareIntensity, got.CareIntensity)
	}
	if !got.RecentFall72h {
		t.Error("expected RecentFall72h = true")
	}
}

func TestAssembler_PropagatesSourceError(t *testing.T) {
	sentinel := errors.New("substrate unavailable")
	client := &InMemorySubstrateClient{Err: sentinel}
	a := kb32ctx.NewAssembler(client)

	_, err := a.Assemble(context.Background(), uuid.New())
	if !errors.Is(err, sentinel) {
		t.Errorf("expected sentinel error, got %v", err)
	}
}

func TestAssembler_ContextCancellation(t *testing.T) {
	// Use a client that respects context so we can test cancellation.
	// InMemorySubstrateClient ignores context by design (it's synchronous),
	// so we test that the assembler correctly forwards a pre-cancelled context
	// to the source and that the source can choose to honour it.
	// Here we test that the assembler itself does NOT add latency beyond the
	// source: if a cancelled context is passed, the assembler checks it
	// before calling the source.
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	rid := uuid.New()
	client := &InMemorySubstrateClient{Snapshot: freshSnapshot(rid)}
	a := kb32ctx.NewAssembler(client)

	_, err := a.Assemble(ctx, rid)
	if err == nil {
		t.Error("expected error for cancelled context, got nil")
	}
	if !errors.Is(err, context.Canceled) {
		t.Errorf("expected context.Canceled, got %v", err)
	}
}

func TestSnapshot_Stale(t *testing.T) {
	ttl := 5 * time.Minute

	// Fresh snapshot — assessed just now.
	fresh := kb32ctx.ClinicalSnapshot{AssessedAt: time.Now()}
	if fresh.Stale(ttl) {
		t.Error("freshly-assessed snapshot should not be stale")
	}

	// Stale snapshot — assessed longer than TTL ago.
	old := kb32ctx.ClinicalSnapshot{AssessedAt: time.Now().Add(-10 * time.Minute)}
	if !old.Stale(ttl) {
		t.Error("snapshot assessed 10 min ago should be stale with 5-min TTL")
	}

	// Boundary: assessed exactly at TTL — should be stale (strictly greater).
	boundary := kb32ctx.ClinicalSnapshot{AssessedAt: time.Now().Add(-ttl - time.Millisecond)}
	if !boundary.Stale(ttl) {
		t.Error("snapshot at boundary should be stale")
	}
}

func TestIsValidCareIntensity(t *testing.T) {
	valid := []string{"active", "comfort", "palliative", "end_of_life"}
	for _, v := range valid {
		if !kb32ctx.IsValidCareIntensity(v) {
			t.Errorf("expected IsValidCareIntensity(%q) = true", v)
		}
	}

	invalid := []string{"", "Active", "COMFORT", "curative", "unknown", "hospice"}
	for _, v := range invalid {
		if kb32ctx.IsValidCareIntensity(v) {
			t.Errorf("expected IsValidCareIntensity(%q) = false", v)
		}
	}
}
