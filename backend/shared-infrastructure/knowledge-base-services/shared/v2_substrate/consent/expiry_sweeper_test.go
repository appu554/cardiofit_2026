package consent

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func TestExpirySweeper_TransitionsExpiredConsents(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	sweeper := NewExpirySweeper(store, lc, func() time.Time { return frozen })
	ctx := context.Background()

	resident := uuid.New()
	pastDue := frozen.Add(-24 * time.Hour) // 1 day before frozen now
	expired := models.Consent{
		ID:            uuid.New(),
		ResidentID:    resident,
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "sdm",
		ValidFrom:     frozen.Add(-30 * 24 * time.Hour),
		ValidUntil:    &pastDue,
		CreatedAt:     frozen.Add(-30 * 24 * time.Hour),
		UpdatedAt:     frozen.Add(-30 * 24 * time.Hour),
	}
	if err := store.Create(ctx, &expired); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE id = $1", expired.ID)
	})

	if err := sweeper.RunOnce(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	got, err := store.Get(ctx, expired.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.State != models.ConsentStateExpired {
		t.Errorf("expected expired; got %q", got.State)
	}
	if got.ExpiredAt == nil {
		t.Errorf("expired_at must be auto-populated by Lifecycle/Store")
	}
	if len(edges.emitted) != 1 {
		t.Errorf("expected 1 EvidenceEdge emitted by sweeper transition; got %d",
			len(edges.emitted))
	}
	if len(edges.emitted) > 0 {
		ev := edges.emitted[0]
		if ev.ActorRole != "system_expiry_sweeper" {
			t.Errorf("ActorRole = %q want system_expiry_sweeper", ev.ActorRole)
		}
		if ev.FromState != models.ConsentStateActive ||
			ev.ToState != models.ConsentStateExpired {
			t.Errorf("edge states wrong: %+v", ev)
		}
	}
}

func TestExpirySweeper_LeavesActiveAndFutureUntouched(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	edges := &fakeEdgeStore{}
	lc := NewLifecycle(store, edges)
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	sweeper := NewExpirySweeper(store, lc, func() time.Time { return frozen })
	ctx := context.Background()

	resident := uuid.New()
	futureDue := frozen.Add(+24 * time.Hour)

	openEnded := models.Consent{
		ID:            uuid.New(),
		ResidentID:    resident,
		Class:         models.ConsentClassPsychotropic,
		State:         models.ConsentStateActive,
		GrantedByID:   uuid.New(),
		GrantedByRole: "sdm",
		ValidFrom:     frozen.Add(-30 * 24 * time.Hour),
		ValidUntil:    nil, // open-ended
		CreatedAt:     frozen.Add(-30 * 24 * time.Hour),
		UpdatedAt:     frozen.Add(-30 * 24 * time.Hour),
	}
	notYetDue := openEnded
	notYetDue.ID = uuid.New()
	notYetDue.ValidUntil = &futureDue

	for _, c := range []models.Consent{openEnded, notYetDue} {
		cc := c
		if err := store.Create(ctx, &cc); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM consents WHERE resident_id = $1", resident)
	})

	if err := sweeper.RunOnce(ctx); err != nil {
		t.Fatalf("sweep: %v", err)
	}

	for _, id := range []uuid.UUID{openEnded.ID, notYetDue.ID} {
		got, _ := store.Get(ctx, id)
		if got.State != models.ConsentStateActive {
			t.Errorf("consent %v should still be active; got %q", id, got.State)
		}
	}
	if len(edges.emitted) != 0 {
		t.Errorf("expected zero edges (no expiries); got %d", len(edges.emitted))
	}
}
