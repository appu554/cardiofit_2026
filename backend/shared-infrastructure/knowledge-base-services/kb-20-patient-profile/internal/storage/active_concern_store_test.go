package storage

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

func openTestActiveConcernStore(t *testing.T) (*ActiveConcernStore, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated active_concern store test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	return NewActiveConcernStore(db), db
}

func TestActiveConcernStore_CreateGet_RoundTrip(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()

	rid := uuid.New()
	startedBy := uuid.New()
	owner := uuid.New()
	in := models.ActiveConcern{
		ResidentID:           rid,
		ConcernType:          models.ActiveConcernPostFall72h,
		StartedAt:            time.Now().UTC().Truncate(time.Second),
		StartedByEventRef:    &startedBy,
		ExpectedResolutionAt: time.Now().UTC().Add(72 * time.Hour).Truncate(time.Second),
		OwnerRoleRef:         &owner,
		ResolutionStatus:     models.ResolutionStatusOpen,
		Notes:                "post-fall watch",
	}
	out, err := s.CreateActiveConcern(context.Background(), in)
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if out.ID == uuid.Nil {
		t.Errorf("expected server-generated ID")
	}
	if out.ConcernType != in.ConcernType {
		t.Errorf("ConcernType drift")
	}
	if out.StartedByEventRef == nil || *out.StartedByEventRef != startedBy {
		t.Errorf("StartedByEventRef drift")
	}

	got, err := s.GetActiveConcern(context.Background(), out.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if got.ID != out.ID {
		t.Errorf("ID drift on read")
	}
	if got.Notes != "post-fall watch" {
		t.Errorf("Notes drift: got %q", got.Notes)
	}
}

func TestActiveConcernStore_GetMissing_ReturnsErrNotFound(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()
	_, err := s.GetActiveConcern(context.Background(), uuid.New())
	if !errors.Is(err, interfaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

func TestActiveConcernStore_CreateRejectsInvalid(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()
	bad := models.ActiveConcern{
		ResidentID:           uuid.New(),
		ConcernType:          "bogus_type",
		StartedAt:            time.Now().UTC(),
		ExpectedResolutionAt: time.Now().UTC().Add(time.Hour),
		ResolutionStatus:     models.ResolutionStatusOpen,
	}
	if _, err := s.CreateActiveConcern(context.Background(), bad); err == nil {
		t.Errorf("expected error for invalid concern_type")
	}
}

func TestActiveConcernStore_ListByResident_FiltersByStatus(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()
	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	openOne, err := s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernPostFall72h,
		StartedAt: now.Add(-2 * time.Hour), ExpectedResolutionAt: now.Add(70 * time.Hour),
		ResolutionStatus: models.ResolutionStatusOpen,
	})
	if err != nil {
		t.Fatalf("seed open: %v", err)
	}
	resolvedAt := now.Add(-time.Hour)
	resolvedOne, err := s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernAcuteInfectionActive,
		StartedAt: now.Add(-3 * time.Hour), ExpectedResolutionAt: now.Add(69 * time.Hour),
		ResolutionStatus: models.ResolutionStatusResolvedStopCriteria,
		ResolvedAt:       &resolvedAt,
	})
	if err != nil {
		t.Fatalf("seed resolved: %v", err)
	}

	all, err := s.ListActiveConcernsByResident(context.Background(), rid, "")
	if err != nil {
		t.Fatalf("list all: %v", err)
	}
	if len(all) != 2 {
		t.Errorf("expected 2 rows, got %d", len(all))
	}

	openOnly, err := s.ListActiveConcernsByResident(context.Background(), rid, models.ResolutionStatusOpen)
	if err != nil {
		t.Fatalf("list open: %v", err)
	}
	if len(openOnly) != 1 || openOnly[0].ID != openOne.ID {
		t.Errorf("open filter wrong: %+v", openOnly)
	}

	resOnly, err := s.ListActiveConcernsByResident(context.Background(), rid, models.ResolutionStatusResolvedStopCriteria)
	if err != nil {
		t.Fatalf("list resolved: %v", err)
	}
	if len(resOnly) != 1 || resOnly[0].ID != resolvedOne.ID {
		t.Errorf("resolved filter wrong: %+v", resOnly)
	}
}

func TestActiveConcernStore_ListActiveByResidentAndType(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()
	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	if _, err := s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernPostFall72h,
		StartedAt: now.Add(-2 * time.Hour), ExpectedResolutionAt: now.Add(70 * time.Hour),
		ResolutionStatus: models.ResolutionStatusOpen,
	}); err != nil {
		t.Fatalf("seed 1: %v", err)
	}
	if _, err := s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernAcuteInfectionActive,
		StartedAt: now.Add(-2 * time.Hour), ExpectedResolutionAt: now.Add(70 * time.Hour),
		ResolutionStatus: models.ResolutionStatusOpen,
	}); err != nil {
		t.Fatalf("seed 2: %v", err)
	}

	got, err := s.ListActiveByResidentAndType(context.Background(), rid, []string{
		models.ActiveConcernPostFall72h, models.ActiveConcernPostFall24h,
	})
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(got) != 1 || got[0].ConcernType != models.ActiveConcernPostFall72h {
		t.Errorf("expected just post_fall_72h; got %+v", got)
	}
}

func TestActiveConcernStore_UpdateResolution_OpenToResolved(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()
	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	created, err := s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernPostFall72h,
		StartedAt: now.Add(-time.Hour), ExpectedResolutionAt: now.Add(71 * time.Hour),
		ResolutionStatus: models.ResolutionStatusOpen,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}

	traceRef := uuid.New()
	resolved, err := s.UpdateResolution(context.Background(), created.ID,
		models.ResolutionStatusResolvedStopCriteria, now, &traceRef)
	if err != nil {
		t.Fatalf("update: %v", err)
	}
	if resolved.ResolutionStatus != models.ResolutionStatusResolvedStopCriteria {
		t.Errorf("status not updated: got %s", resolved.ResolutionStatus)
	}
	if resolved.ResolvedAt == nil || !resolved.ResolvedAt.Equal(now) {
		t.Errorf("resolved_at drift")
	}
	if resolved.ResolutionEvidenceTraceRef == nil || *resolved.ResolutionEvidenceTraceRef != traceRef {
		t.Errorf("trace ref not persisted")
	}
}

func TestActiveConcernStore_UpdateResolution_RejectsTerminalSource(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()
	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)
	resolvedAt := now.Add(-time.Minute)
	created, err := s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernEndOfLifeRecognition,
		StartedAt: now.Add(-time.Hour), ExpectedResolutionAt: now.Add(719 * time.Hour),
		ResolutionStatus: models.ResolutionStatusExpiredUnresolved,
		ResolvedAt:       &resolvedAt,
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	// Try to "reopen" — must fail.
	if _, err := s.UpdateResolution(context.Background(), created.ID,
		models.ResolutionStatusOpen, now, nil); err == nil {
		t.Errorf("expected error on terminal→open transition")
	}
}

func TestActiveConcernStore_ListExpiringConcerns(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()
	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	expired, err := s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernPostFall72h,
		StartedAt: now.Add(-100 * time.Hour), ExpectedResolutionAt: now.Add(-time.Hour),
		ResolutionStatus: models.ResolutionStatusOpen,
	})
	if err != nil {
		t.Fatalf("seed expired: %v", err)
	}
	_, err = s.CreateActiveConcern(context.Background(), models.ActiveConcern{
		ResidentID: rid, ConcernType: models.ActiveConcernAcuteInfectionActive,
		StartedAt: now.Add(-time.Hour), ExpectedResolutionAt: now.Add(50 * time.Hour),
		ResolutionStatus: models.ResolutionStatusOpen,
	})
	if err != nil {
		t.Fatalf("seed not-yet-expired: %v", err)
	}

	// within=0 → only past-due rows.
	pastDue, err := s.ListExpiringConcerns(context.Background(), 0)
	if err != nil {
		t.Fatalf("list past-due: %v", err)
	}
	foundExpired := false
	for _, c := range pastDue {
		if c.ID == expired.ID {
			foundExpired = true
		}
	}
	if !foundExpired {
		t.Errorf("expected expired concern in past-due list")
	}
}

func TestActiveConcernStore_LookupTriggers_FromSeed(t *testing.T) {
	s, db := openTestActiveConcernStore(t)
	defer db.Close()

	// Event-driven trigger.
	got, err := s.LookupConcernTriggersByEventType(context.Background(), models.EventTypeFall)
	if err != nil {
		t.Fatalf("lookup by event: %v", err)
	}
	// Seed has 2 fall rows: post_fall_72h + post_fall_24h.
	if len(got) != 2 {
		t.Errorf("expected 2 fall triggers, got %d: %+v", len(got), got)
	}

	// ATC-driven trigger.
	atcGot, err := s.LookupConcernTriggersByMedATC(context.Background(), "J01CA04", "treatment")
	if err != nil {
		t.Fatalf("lookup by med_atc: %v", err)
	}
	if len(atcGot) != 1 || atcGot[0].ConcernType != models.ActiveConcernAntibioticCourseActive {
		t.Errorf("expected antibiotic_course_active for J01CA04; got %+v", atcGot)
	}

	// N05 (psychotropic) — no intent restriction in seed.
	psyGot, err := s.LookupConcernTriggersByMedATC(context.Background(), "N05AH04", "")
	if err != nil {
		t.Fatalf("lookup N05: %v", err)
	}
	if len(psyGot) != 1 || psyGot[0].ConcernType != models.ActiveConcernNewPsychotropicTitration {
		t.Errorf("expected new_psychotropic_titration_window for N05AH04; got %+v", psyGot)
	}

	// Unknown ATC → no match.
	noneGot, _ := s.LookupConcernTriggersByMedATC(context.Background(), "Z99", "")
	if len(noneGot) != 0 {
		t.Errorf("expected no triggers for unknown ATC; got %+v", noneGot)
	}
}
