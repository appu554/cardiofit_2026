package monitoring

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func testDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN unset; skipping DB integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.PingContext(context.Background()); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return db
}

// seedPlan returns a fully-populated test MonitoringPlan with one observation
// obligation. Caller is responsible for store.Create + t.Cleanup of the row.
func seedPlan(state string, recID uuid.UUID) models.MonitoringPlan {
	now := time.Now().UTC()
	return models.MonitoringPlan{
		ID:               uuid.New(),
		RecommendationID: recID,
		ResidentID:       uuid.New(),
		State:            state,
		Obligations: []models.MonitoringObligation{
			{
				Type:            models.MonitoringObligationTypeObservation,
				ObservationCode: "potassium",
				FrequencyHours:  24,
				DueAt:           now.Add(2 * time.Hour),
				ThresholdSpec:   "value > 5.5",
			},
		},
		StartedAt:           now,
		ExpectedEndAt:       now.Add(7 * 24 * time.Hour),
		EscalateAfterMissed: 2,
		CreatedAt:           now,
		UpdatedAt:           now,
	}
}

func TestPostgresStore_CreateAndGet(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	p := seedPlan(models.MonitoringPlanStateActive, uuid.New())
	if err := store.Create(ctx, &p); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", p.ID)
	})

	got, err := store.Get(ctx, p.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != p.ID || got.State != p.State {
		t.Errorf("scalars mismatch: %+v", got)
	}
	if len(got.Obligations) != 1 {
		t.Fatalf("obligations lost: got %d", len(got.Obligations))
	}
	if got.Obligations[0].ObservationCode != "potassium" {
		t.Errorf("obligation detail lost: %+v", got.Obligations[0])
	}
}

func TestPostgresStore_GetNotFound(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	_, err := store.Get(context.Background(), uuid.New())
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound; got %v", err)
	}
}

func TestPostgresStore_UpdateState(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	p := seedPlan(models.MonitoringPlanStateActive, uuid.New())
	if err := store.Create(ctx, &p); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", p.ID)
	})

	if err := store.UpdateState(ctx, p.ID, models.MonitoringPlanStateCompleted); err != nil {
		t.Fatalf("update: %v", err)
	}
	got, _ := store.Get(ctx, p.ID)
	if got.State != models.MonitoringPlanStateCompleted {
		t.Errorf("state = %q want completed", got.State)
	}
}

func TestPostgresStore_MarkObligationFulfilled(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	p := seedPlan(models.MonitoringPlanStateActive, uuid.New())
	if err := store.Create(ctx, &p); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", p.ID)
	})

	obsID := uuid.New()
	fulfilled := time.Date(2026, 5, 7, 14, 0, 0, 0, time.UTC)
	if err := store.MarkObligationFulfilled(ctx, p.ID, 0, obsID, fulfilled); err != nil {
		t.Fatalf("mark fulfilled: %v", err)
	}

	got, _ := store.Get(ctx, p.ID)
	if got.Obligations[0].FulfilledAt == nil {
		t.Fatalf("FulfilledAt nil after MarkObligationFulfilled")
	}
	if !got.Obligations[0].FulfilledAt.Equal(fulfilled) {
		t.Errorf("FulfilledAt = %v want %v", *got.Obligations[0].FulfilledAt, fulfilled)
	}
	if got.Obligations[0].FulfilledByObsID == nil ||
		*got.Obligations[0].FulfilledByObsID != obsID {
		t.Errorf("FulfilledByObsID lost: %v", got.Obligations[0].FulfilledByObsID)
	}
}

func TestPostgresStore_MarkThresholdCrossed(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	p := seedPlan(models.MonitoringPlanStateActive, uuid.New())
	if err := store.Create(ctx, &p); err != nil {
		t.Fatalf("create: %v", err)
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(context.Background(),
			"DELETE FROM monitoring_plans WHERE id = $1", p.ID)
	})

	crossed := time.Date(2026, 5, 7, 16, 30, 0, 0, time.UTC)
	if err := store.MarkThresholdCrossed(ctx, p.ID, 0, crossed); err != nil {
		t.Fatalf("mark crossed: %v", err)
	}
	got, _ := store.Get(ctx, p.ID)
	if got.Obligations[0].ThresholdCrossedAt == nil {
		t.Fatalf("ThresholdCrossedAt nil")
	}
	if !got.Obligations[0].ThresholdCrossedAt.Equal(crossed) {
		t.Errorf("ThresholdCrossedAt = %v want %v",
			*got.Obligations[0].ThresholdCrossedAt, crossed)
	}
}

func TestPostgresStore_ListActiveOverdue(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()
	now := time.Now().UTC()
	cutoff := now

	overdueRec := uuid.New()
	overdue := seedPlan(models.MonitoringPlanStateActive, overdueRec)
	overdue.ExpectedEndAt = now.Add(-2 * time.Hour)

	notYetDue := seedPlan(models.MonitoringPlanStateActive, uuid.New())
	notYetDue.ExpectedEndAt = now.Add(+2 * time.Hour)

	completed := seedPlan(models.MonitoringPlanStateCompleted, uuid.New())
	completed.ExpectedEndAt = now.Add(-2 * time.Hour) // overdue but completed

	for _, p := range []models.MonitoringPlan{overdue, notYetDue, completed} {
		pp := p
		if err := store.Create(ctx, &pp); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	t.Cleanup(func() {
		for _, id := range []uuid.UUID{overdue.ID, notYetDue.ID, completed.ID} {
			_, _ = db.ExecContext(context.Background(),
				"DELETE FROM monitoring_plans WHERE id = $1", id)
		}
	})

	got, err := store.ListActiveOverdue(ctx, cutoff)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	// Filter by recommendation_id we control to avoid pollution
	count := 0
	for _, p := range got {
		if p.RecommendationID == overdueRec {
			count++
		}
	}
	if count != 1 {
		t.Errorf("expected exactly 1 overdue active plan with our recID; got %d (total returned %d)",
			count, len(got))
	}
}

func TestPostgresStore_ListByRecommendation(t *testing.T) {
	db := testDB(t)
	defer db.Close()
	store := NewPostgresStore(db)
	ctx := context.Background()

	recID := uuid.New()
	otherRecID := uuid.New()

	for _, p := range []models.MonitoringPlan{
		seedPlan(models.MonitoringPlanStateActive, recID),
		seedPlan(models.MonitoringPlanStateCompleted, recID),
		seedPlan(models.MonitoringPlanStateActive, otherRecID),
	} {
		pp := p
		if err := store.Create(ctx, &pp); err != nil {
			t.Fatalf("seed: %v", err)
		}
		id := pp.ID
		t.Cleanup(func() {
			_, _ = db.ExecContext(context.Background(),
				"DELETE FROM monitoring_plans WHERE id = $1", id)
		})
	}

	got, err := store.ListByRecommendation(ctx, recID)
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected 2 plans for recID; got %d", len(got))
	}
}
