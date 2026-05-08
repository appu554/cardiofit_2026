package recommendation

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// TestComputeRIR_OnlyImplementedCounts verifies that the Go RIR formula
// counts a recommendation as "actioned" ONLY when its current state is
// `implemented`, `monitoring-active`, or `outcome-recorded` — NOT when
// it was decided-no-action and closed without implementation, and NOT
// when it remains in earlier lifecycle states. This resolves the Task 3
// review concern.
//
// Scenario: 5 recommendations submitted within the 28-day window:
//   - 1 implemented (counts)
//   - 1 monitoring-active (counts)
//   - 1 outcome-recorded (counts)
//   - 1 closed (does NOT count — table can't distinguish closed-after-
//     implemented from closed-via-decided-no-action; conservatively excluded)
//   - 1 still submitted, never actioned (does NOT count)
//
// Expected RIR: 3/5 = 60.0%
func TestComputeRIR_OnlyImplementedCounts(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()
	store := NewPostgresStore(db)
	author := uuid.New()
	now := time.Now().UTC()

	mk := func(state string, submittedDaysAgo, decidedDaysAgo, closedDaysAgo int) {
		rec := models.Recommendation{
			ID:              uuid.New(),
			ResidentID:      uuid.New(),
			AuthorID:        author,
			State:           state,
			Type:            models.RecommendationTypeStop,
			Urgency:         models.RecommendationUrgencyAmber,
			Title:           "test",
			ClinicalContent: models.ClinicalContent{Issue: "x"},
			CreatedAt:       now,
			UpdatedAt:       now,
		}
		if submittedDaysAgo >= 0 {
			ts := now.Add(-time.Duration(submittedDaysAgo) * 24 * time.Hour)
			rec.SubmittedAt = &ts
		}
		if decidedDaysAgo >= 0 {
			ts := now.Add(-time.Duration(decidedDaysAgo) * 24 * time.Hour)
			rec.DecidedAt = &ts
		}
		if closedDaysAgo >= 0 {
			ts := now.Add(-time.Duration(closedDaysAgo) * 24 * time.Hour)
			rec.ClosedAt = &ts
		}
		if err := store.Create(ctx, &rec); err != nil {
			t.Fatalf("seed: %v", err)
		}
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(),
				"DELETE FROM recommendations WHERE id = $1", rec.ID)
		})
	}

	// Counted (3): current state is in the actioned set, decided_at within window.
	mk(models.RecommendationStateImplemented, 10, 5, -1)
	mk(models.RecommendationStateMonitoringActive, 8, 3, -1)
	mk(models.RecommendationStateOutcomeRecorded, 12, 7, -1)
	// Not counted: closed (table can't distinguish path).
	mk(models.RecommendationStateClosed, 7, 3, 2)
	// Not counted: still submitted.
	mk(models.RecommendationStateSubmitted, 5, -1, -1)

	got, err := ComputeRIR(ctx, db, author, 28*24*time.Hour)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if got.Submitted != 5 {
		t.Errorf("Submitted = %d want 5", got.Submitted)
	}
	if got.Actioned != 3 {
		t.Errorf("Actioned = %d want 3 (implementation rate excludes decided-no-action and closed)", got.Actioned)
	}
	if got.RatePercent < 59.9 || got.RatePercent > 60.1 {
		t.Errorf("RatePercent = %.2f want ~60.0", got.RatePercent)
	}
}

// TestComputeRIR_EmptyAuthor verifies the zero-row case returns a
// well-formed zero result rather than NaN or error.
func TestComputeRIR_EmptyAuthor(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()
	got, err := ComputeRIR(ctx, db, uuid.New(), 28*24*time.Hour)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if got.Submitted != 0 || got.Actioned != 0 || got.RatePercent != 0 {
		t.Errorf("empty author should produce zero result; got %+v", got)
	}
}

// TestComputeRIR_OutsideWindowExcluded verifies the window boundary —
// a recommendation submitted 60 days ago must be excluded from a 28-day
// rolling RIR.
func TestComputeRIR_OutsideWindowExcluded(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	ctx := context.Background()
	store := NewPostgresStore(db)
	author := uuid.New()
	now := time.Now().UTC()

	old := now.Add(-60 * 24 * time.Hour)
	rec := models.Recommendation{
		ID:              uuid.New(),
		ResidentID:      uuid.New(),
		AuthorID:        author,
		State:           models.RecommendationStateImplemented,
		Type:            models.RecommendationTypeStop,
		Urgency:         models.RecommendationUrgencyAmber,
		Title:           "old",
		ClinicalContent: models.ClinicalContent{Issue: "x"},
		SubmittedAt:     &old,
		DecidedAt:       &old,
		CreatedAt:       now,
		UpdatedAt:       now,
	}
	if err := store.Create(ctx, &rec); err != nil {
		t.Fatalf("seed: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM recommendations WHERE id = $1", rec.ID)
	})

	got, err := ComputeRIR(ctx, db, author, 28*24*time.Hour)
	if err != nil {
		t.Fatalf("compute: %v", err)
	}
	if got.Submitted != 0 {
		t.Errorf("recommendation submitted 60d ago should be outside 28d window; got Submitted=%d", got.Submitted)
	}
}
