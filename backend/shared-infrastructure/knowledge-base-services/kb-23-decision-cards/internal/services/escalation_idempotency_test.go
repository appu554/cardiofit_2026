package services

import (
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/google/uuid"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	"kb-23-decision-cards/internal/models"
)

var sqliteTestDBCounter atomic.Int64

// setupEscalationTestDB creates an in-memory SQLite with an escalation_events
// table whose columns match the fields AcknowledgePendingEscalation /
// RecordActionOnAcknowledgedEscalation touch. We don't use GORM AutoMigrate
// because EscalationEvent's GORM tags use Postgres-only defaults
// (e.g. gen_random_uuid()) that SQLite would reject; a hand-rolled CREATE
// TABLE is faster and more explicit about what's being tested.
//
// Each call uses a unique named shared-cache in-memory DSN so goroutines in
// a concurrency test can all reach the same in-memory database (plain
// ":memory:" is per-connection) while different tests remain isolated.
func setupEscalationTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	// _busy_timeout makes SQLite's per-database write lock queue rather than
	// immediately error with SQLITE_BUSY when a second goroutine contends for
	// the write. Without this, concurrent ACKs race with the whole-DB lock
	// instead of reaching the state predicate, and we can't exercise the
	// idempotency guarantee we actually want to verify.
	dsn := fmt.Sprintf("file:esc_test_%d?mode=memory&cache=shared&_busy_timeout=5000", sqliteTestDBCounter.Add(1))
	db, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	// Keep at most one active connection so the shared-cache in-memory DSN
	// addresses a single logical database (multi-connection + shared cache
	// can still produce spurious locks on some platforms).
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatalf("get sql.DB: %v", err)
	}
	sqlDB.SetMaxOpenConns(1)
	err = db.Exec(`
		CREATE TABLE escalation_events (
			id TEXT PRIMARY KEY,
			patient_id TEXT NOT NULL,
			card_id TEXT,
			trigger_type TEXT NOT NULL,
			escalation_tier TEXT NOT NULL,
			current_state TEXT NOT NULL DEFAULT 'PENDING',
			assigned_clinician_id TEXT,
			assigned_clinician_role TEXT,
			channels TEXT,
			delivery_attempts INTEGER DEFAULT 0,
			created_at DATETIME NOT NULL,
			delivered_at DATETIME,
			acknowledged_at DATETIME,
			acknowledged_by TEXT,
			acted_at DATETIME,
			action_type TEXT,
			action_detail TEXT,
			resolved_at DATETIME,
			resolution_reason TEXT,
			escalated_at DATETIME,
			escalation_level INTEGER NOT NULL DEFAULT 1,
			timeout_at DATETIME,
			previous_event_id TEXT,
			pai_score_at_trigger REAL,
			pai_tier_at_trigger TEXT,
			primary_reason TEXT,
			suggested_action TEXT,
			suggested_timeframe TEXT,
			updated_at DATETIME
		)
	`).Error
	if err != nil {
		t.Fatalf("create escalation_events: %v", err)
	}
	return db
}

func insertPendingEscalation(t *testing.T, db *gorm.DB, patientID string) *models.EscalationEvent {
	t.Helper()
	ev := &models.EscalationEvent{
		ID:              uuid.New(),
		PatientID:       patientID,
		TriggerType:     string(models.TriggerCardGenerated),
		EscalationTier:  string(models.TierSafety),
		CurrentState:    string(models.StatePending),
		EscalationLevel: 1,
		CreatedAt:       time.Now().UTC(),
	}
	if err := db.Create(ev).Error; err != nil {
		t.Fatalf("seed escalation_events: %v", err)
	}
	return ev
}

// TestAcknowledgePendingEscalation_SequentialIsIdempotent exercises the
// dual-channel ACK scenario (WhatsApp + worklist): two clinicians acknowledge
// the same escalation in quick succession. The first ACK must succeed and
// stamp T2; the second must be a no-op (ErrRecordNotFound) so the first
// clinician's identity and timestamp are preserved, not overwritten.
func TestAcknowledgePendingEscalation_SequentialIsIdempotent(t *testing.T) {
	db := setupEscalationTestDB(t)
	tracker := NewAcknowledgmentTracker(nil)

	seeded := insertPendingEscalation(t, db, "patient-ack-1")

	first, err := AcknowledgePendingEscalation(db, tracker, "patient-ack-1", "clinician-A")
	if err != nil {
		t.Fatalf("first ACK should succeed: %v", err)
	}
	if first == nil {
		t.Fatal("first ACK returned nil event")
	}
	if first.CurrentState != string(models.StateAcknowledged) {
		t.Errorf("first ACK state = %s, want ACKNOWLEDGED", first.CurrentState)
	}
	if first.AcknowledgedBy != "clinician-A" {
		t.Errorf("first ACK recorded by = %q, want clinician-A", first.AcknowledgedBy)
	}
	if first.AcknowledgedAt == nil {
		t.Fatal("first ACK did not stamp T2")
	}
	firstT2 := *first.AcknowledgedAt

	second, err := AcknowledgePendingEscalation(db, tracker, "patient-ack-1", "clinician-B")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("second ACK should return ErrRecordNotFound (state predicate guard), got err=%v event=%v", err, second)
	}
	if second != nil {
		t.Errorf("second ACK should return nil event, got %+v", second)
	}

	var reloaded models.EscalationEvent
	if err := db.Where("id = ?", seeded.ID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload escalation: %v", err)
	}
	if reloaded.AcknowledgedBy != "clinician-A" {
		t.Errorf("T2 identity overwritten: AcknowledgedBy = %q, want clinician-A", reloaded.AcknowledgedBy)
	}
	if reloaded.AcknowledgedAt == nil || !reloaded.AcknowledgedAt.Equal(firstT2) {
		t.Errorf("T2 timestamp overwritten: got %v, want %v", reloaded.AcknowledgedAt, firstT2)
	}
}

// TestAcknowledgePendingEscalation_Concurrent is the true idempotency
// regression: launch N goroutines racing to ACK the same escalation and
// verify exactly ONE of them records T2 — the rest see the state predicate
// reject them. Without the transactional state guard, Gap 19 latency metrics
// would be corrupted by duplicate T2 writes, and the SAFETY escalation
// timeout cancellation would fire ambiguously.
func TestAcknowledgePendingEscalation_Concurrent(t *testing.T) {
	db := setupEscalationTestDB(t)
	tracker := NewAcknowledgmentTracker(nil)

	const N = 10
	insertPendingEscalation(t, db, "patient-race")

	var (
		wg         sync.WaitGroup
		mu         sync.Mutex
		successes  int
		idempotent int
		other      []error
	)
	wg.Add(N)
	for i := 0; i < N; i++ {
		clinician := "clinician-" + string(rune('A'+i))
		go func() {
			defer wg.Done()
			ev, err := AcknowledgePendingEscalation(db, tracker, "patient-race", clinician)
			mu.Lock()
			defer mu.Unlock()
			switch {
			case err == nil && ev != nil:
				successes++
			case errors.Is(err, gorm.ErrRecordNotFound):
				idempotent++
			default:
				other = append(other, err)
			}
		}()
	}
	wg.Wait()

	if successes != 1 {
		t.Errorf("exactly 1 goroutine should succeed, got %d successes + %d idempotent no-ops", successes, idempotent)
	}
	if successes+idempotent != N {
		t.Errorf("every goroutine should land in success or idempotent path: %d+%d != %d (errors: %v)", successes, idempotent, N, other)
	}
}

// TestRecordActionOnAcknowledgedEscalation_Idempotent mirrors the ACK test
// for T3 — if a clinician taps an action button twice (or two clinicians act
// on the same escalation from different surfaces), only one T3 transition
// should land.
func TestRecordActionOnAcknowledgedEscalation_Idempotent(t *testing.T) {
	db := setupEscalationTestDB(t)
	tracker := NewAcknowledgmentTracker(nil)

	seeded := insertPendingEscalation(t, db, "patient-act-1")
	// Move it to ACKNOWLEDGED first (pre-condition for T3).
	if _, err := AcknowledgePendingEscalation(db, tracker, "patient-act-1", "clinician-A"); err != nil {
		t.Fatalf("seed ACK: %v", err)
	}

	first, err := RecordActionOnAcknowledgedEscalation(db, tracker, "patient-act-1", "CALL_PATIENT", "called the mobile")
	if err != nil {
		t.Fatalf("first action should succeed: %v", err)
	}
	if first == nil || first.CurrentState != string(models.StateActed) {
		t.Fatalf("first action should transition to ACTED, got %+v", first)
	}
	if first.ActionType != "CALL_PATIENT" {
		t.Errorf("ActionType = %q, want CALL_PATIENT", first.ActionType)
	}

	second, err := RecordActionOnAcknowledgedEscalation(db, tracker, "patient-act-1", "MEDICATION_REVIEW", "follow-up attempt")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("second action should return ErrRecordNotFound, got err=%v event=%v", err, second)
	}

	// Verify the first action's details survived — no overwrite by the second caller.
	var reloaded models.EscalationEvent
	if err := db.Where("id = ?", seeded.ID).First(&reloaded).Error; err != nil {
		t.Fatalf("reload: %v", err)
	}
	if reloaded.ActionType != "CALL_PATIENT" {
		t.Errorf("first action's type overwritten: got %q, want CALL_PATIENT", reloaded.ActionType)
	}
	if reloaded.ActionDetail != "called the mobile" {
		t.Errorf("first action's detail overwritten: got %q", reloaded.ActionDetail)
	}
}

// TestAcknowledgePendingEscalation_NoEligibleRow exercises the path where no
// pending escalation exists for the patient — the helper should return
// (nil, gorm.ErrRecordNotFound) rather than panicking, since callers treat
// this as the "already acknowledged / nothing to do" signal.
func TestAcknowledgePendingEscalation_NoEligibleRow(t *testing.T) {
	db := setupEscalationTestDB(t)
	tracker := NewAcknowledgmentTracker(nil)

	ev, err := AcknowledgePendingEscalation(db, tracker, "patient-absent", "clinician-X")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound for missing escalation, got err=%v", err)
	}
	if ev != nil {
		t.Errorf("expected nil event, got %+v", ev)
	}
}
