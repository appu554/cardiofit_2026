package recommendation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// listOverdueFakeStore lets the test inject a fixed list to be returned.
type listOverdueFakeStore struct {
	fakeStore
	list []models.Recommendation
}

func (l *listOverdueFakeStore) ListDeferredOverdue(_ context.Context,
	_ time.Time) ([]models.Recommendation, error) {
	return l.list, nil
}

// recordingEvents captures all events the escalator emits.
type recordingEvents struct {
	emitted []EscalationEvent
}

func (r *recordingEvents) Emit(_ context.Context, ev EscalationEvent) error {
	r.emitted = append(r.emitted, ev)
	return nil
}

func TestDeferredEscalator_EmitsEventForOverdue(t *testing.T) {
	overdueID := uuid.New()
	residentID := uuid.New()
	authorID := uuid.New()
	dueAt := time.Now().Add(-1 * time.Hour).UTC()

	store := &listOverdueFakeStore{
		list: []models.Recommendation{
			{
				ID:          overdueID,
				ResidentID:  residentID,
				AuthorID:    authorID,
				State:       models.RecommendationStateDeferred,
				ReviewDueAt: &dueAt,
			},
		},
	}
	events := &recordingEvents{}
	frozen := time.Date(2026, 5, 7, 12, 0, 0, 0, time.UTC)
	esc := NewDeferredEscalator(store, events, func() time.Time { return frozen })

	if err := esc.RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(events.emitted) != 1 {
		t.Fatalf("expected 1 escalation event; got %d", len(events.emitted))
	}
	got := events.emitted[0]
	if got.RecommendationID != overdueID {
		t.Errorf("RecommendationID = %v want %v", got.RecommendationID, overdueID)
	}
	if got.ResidentID != residentID {
		t.Errorf("ResidentID not propagated")
	}
	if got.AuthorID != authorID {
		t.Errorf("AuthorID not propagated")
	}
	if !got.OriginalDueAt.Equal(dueAt) {
		t.Errorf("OriginalDueAt = %v want %v", got.OriginalDueAt, dueAt)
	}
	if !got.EmittedAt.Equal(frozen) {
		t.Errorf("EmittedAt = %v want frozen %v", got.EmittedAt, frozen)
	}
}

func TestDeferredEscalator_NoOverdueNoEvents(t *testing.T) {
	store := &listOverdueFakeStore{list: nil}
	events := &recordingEvents{}
	esc := NewDeferredEscalator(store, events, time.Now)

	if err := esc.RunOnce(context.Background()); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(events.emitted) != 0 {
		t.Errorf("expected zero events; got %d", len(events.emitted))
	}
}

// TestPostgresStore_ListDeferredOverdue is an integration test that exercises
// the SQL implementation. Seeds rows in three states (deferred-overdue,
// deferred-not-yet-due, non-deferred-but-old), runs ListDeferredOverdue,
// and verifies only the deferred-overdue row is returned.
func TestPostgresStore_ListDeferredOverdue(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	now := time.Now().UTC()
	pastDue := now.Add(-2 * time.Hour)
	futureDue := now.Add(+2 * time.Hour)

	overdueID := uuid.New()
	notYetDueID := uuid.New()
	nonDeferredID := uuid.New()

	for _, rec := range []models.Recommendation{
		{
			ID: overdueID, ResidentID: uuid.New(), AuthorID: uuid.New(),
			State:           models.RecommendationStateDeferred,
			Type:            models.RecommendationTypeStop,
			Urgency:         models.RecommendationUrgencyAmber,
			Title:           "overdue",
			ClinicalContent: models.ClinicalContent{Issue: "x"},
			ReviewDueAt:     &pastDue, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: notYetDueID, ResidentID: uuid.New(), AuthorID: uuid.New(),
			State:           models.RecommendationStateDeferred,
			Type:            models.RecommendationTypeStop,
			Urgency:         models.RecommendationUrgencyAmber,
			Title:           "not yet due",
			ClinicalContent: models.ClinicalContent{Issue: "x"},
			ReviewDueAt:     &futureDue, CreatedAt: now, UpdatedAt: now,
		},
		{
			ID: nonDeferredID, ResidentID: uuid.New(), AuthorID: uuid.New(),
			State:           models.RecommendationStateSubmitted,
			Type:            models.RecommendationTypeStop,
			Urgency:         models.RecommendationUrgencyAmber,
			Title:           "submitted",
			ClinicalContent: models.ClinicalContent{Issue: "x"},
			ReviewDueAt:     &pastDue, CreatedAt: now, UpdatedAt: now,
		},
	} {
		r := rec
		if err := store.Create(ctx, &r); err != nil {
			t.Fatalf("seed %v: %v", r.ID, err)
		}
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(),
				"DELETE FROM recommendations WHERE id = $1", r.ID)
		})
	}

	got, err := store.ListDeferredOverdue(ctx, now)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 overdue row; got %d", len(got))
	}
	if got[0].ID != overdueID {
		t.Errorf("got %v want %v", got[0].ID, overdueID)
	}
}
