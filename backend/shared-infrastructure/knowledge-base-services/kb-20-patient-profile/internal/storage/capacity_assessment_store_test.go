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

func openTestCapacityStore(t *testing.T) (*CapacityAssessmentStore, *sql.DB) {
	t.Helper()
	dsn := os.Getenv("KB20_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB20_TEST_DATABASE_URL not set; skipping DB-gated capacity store test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Fatalf("ping: %v", err)
	}
	v2 := NewV2SubstrateStoreWithDB(db)
	return NewCapacityAssessmentStore(db, v2), db
}

func validCA(rid uuid.UUID, domain, outcome, duration string, assessedAt time.Time) models.CapacityAssessment {
	return models.CapacityAssessment{
		ResidentRef:     rid,
		AssessedAt:      assessedAt,
		AssessorRoleRef: uuid.New(),
		Domain:          domain,
		Outcome:         outcome,
		Duration:        duration,
	}
}

func TestCapacityStore_InsertAndGetCurrentPerDomain(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()

	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	in := validCA(rid, models.CapacityDomainMedical, models.CapacityOutcomeIntact, models.CapacityDurationPermanent, now)
	out, err := s.CreateCapacityAssessment(context.Background(), in)
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.Assessment == nil || out.Assessment.Domain != models.CapacityDomainMedical {
		t.Fatalf("Assessment drift: %+v", out.Assessment)
	}
	// intact+medical → no Event (only impaired+medical triggers Event).
	if out.Event != nil {
		t.Errorf("expected no Event for intact+medical; got %+v", out.Event)
	}
	if out.EvidenceTraceNodeRef == uuid.Nil {
		t.Errorf("expected EvidenceTraceNodeRef set")
	}

	cur, err := s.GetCurrentCapacity(context.Background(), rid, models.CapacityDomainMedical)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if cur.ID != out.Assessment.ID {
		t.Errorf("current ID drift")
	}
}

func TestCapacityStore_HistoryDescendingPerDomain(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()

	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	for i, at := range []time.Time{now.Add(-72 * time.Hour), now.Add(-24 * time.Hour), now} {
		_ = i
		if _, err := s.CreateCapacityAssessment(context.Background(), validCA(
			rid, models.CapacityDomainFinancial,
			models.CapacityOutcomeIntact, models.CapacityDurationPermanent, at,
		)); err != nil {
			t.Fatalf("seed at %v: %v", at, err)
		}
	}

	hist, err := s.ListCapacityHistory(context.Background(), rid, models.CapacityDomainFinancial)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(hist) != 3 {
		t.Fatalf("expected 3 rows; got %d", len(hist))
	}
	if !hist[0].AssessedAt.Equal(now) {
		t.Errorf("expected newest first; got %v", hist[0].AssessedAt)
	}
}

func TestCapacityStore_DomainsAreIndependent(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()

	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	// Impaired medical capacity.
	if _, err := s.CreateCapacityAssessment(context.Background(), validCA(
		rid, models.CapacityDomainMedical,
		models.CapacityOutcomeImpaired, models.CapacityDurationPermanent, now,
	)); err != nil {
		t.Fatalf("seed medical: %v", err)
	}
	// Intact financial capacity at the same moment.
	if _, err := s.CreateCapacityAssessment(context.Background(), validCA(
		rid, models.CapacityDomainFinancial,
		models.CapacityOutcomeIntact, models.CapacityDurationPermanent, now,
	)); err != nil {
		t.Fatalf("seed financial: %v", err)
	}

	med, err := s.GetCurrentCapacity(context.Background(), rid, models.CapacityDomainMedical)
	if err != nil {
		t.Fatalf("get medical: %v", err)
	}
	if med.Outcome != models.CapacityOutcomeImpaired {
		t.Errorf("medical drift: %s", med.Outcome)
	}
	fin, err := s.GetCurrentCapacity(context.Background(), rid, models.CapacityDomainFinancial)
	if err != nil {
		t.Fatalf("get financial: %v", err)
	}
	if fin.Outcome != models.CapacityOutcomeIntact {
		t.Errorf("financial drift — impaired-medical leaked into financial: %s", fin.Outcome)
	}

	// Listing current returns both rows, one per domain.
	all, err := s.ListCurrentCapacityByResident(context.Background(), rid)
	if err != nil {
		t.Fatalf("list current: %v", err)
	}
	if len(all) != 2 {
		t.Fatalf("expected 2 current rows (one per domain); got %d", len(all))
	}
}

func TestCapacityStore_ImpairedMedicalEmitsEventAndConsentTrace(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()

	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	out, err := s.CreateCapacityAssessment(context.Background(), validCA(
		rid, models.CapacityDomainMedical,
		models.CapacityOutcomeImpaired, models.CapacityDurationPermanent, now,
	))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.Event == nil {
		t.Fatalf("expected capacity_change Event for impaired+medical")
	}
	if out.Event.EventType != models.EventTypeCapacityChange {
		t.Errorf("Event type drift: %s", out.Event.EventType)
	}
	if out.EvidenceTraceNodeRef == uuid.Nil {
		t.Fatalf("expected EvidenceTraceNodeRef set")
	}
	// Verify the EvidenceTrace node carries state_machine=Consent.
	node, err := s.v2.GetEvidenceTraceNode(context.Background(), out.EvidenceTraceNodeRef)
	if err != nil {
		t.Fatalf("get evidence trace node: %v", err)
	}
	if node.StateMachine != models.EvidenceTraceStateMachineConsent {
		t.Errorf("expected state_machine=Consent for impaired+medical; got %s", node.StateMachine)
	}
}

func TestCapacityStore_ImpairedFinancialNoEventClinicalStateTrace(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()

	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	out, err := s.CreateCapacityAssessment(context.Background(), validCA(
		rid, models.CapacityDomainFinancial,
		models.CapacityOutcomeImpaired, models.CapacityDurationPermanent, now,
	))
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if out.Event != nil {
		t.Errorf("expected NO Event for impaired+financial; got %+v", out.Event)
	}
	if out.EvidenceTraceNodeRef == uuid.Nil {
		t.Fatalf("expected EvidenceTraceNodeRef set")
	}
	node, err := s.v2.GetEvidenceTraceNode(context.Background(), out.EvidenceTraceNodeRef)
	if err != nil {
		t.Fatalf("get evidence trace node: %v", err)
	}
	if node.StateMachine != models.EvidenceTraceStateMachineClinicalState {
		t.Errorf("expected state_machine=ClinicalState for non-medical impaired; got %s", node.StateMachine)
	}
}

func TestCapacityStore_AppendingNewAssessmentBecomesCurrent(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()

	rid := uuid.New()
	now := time.Now().UTC().Truncate(time.Second)

	first, err := s.CreateCapacityAssessment(context.Background(), validCA(
		rid, models.CapacityDomainMedical,
		models.CapacityOutcomeIntact, models.CapacityDurationPermanent, now.Add(-24*time.Hour),
	))
	if err != nil {
		t.Fatalf("first: %v", err)
	}
	second, err := s.CreateCapacityAssessment(context.Background(), validCA(
		rid, models.CapacityDomainMedical,
		models.CapacityOutcomeImpaired, models.CapacityDurationPermanent, now,
	))
	if err != nil {
		t.Fatalf("second: %v", err)
	}
	// supersedes_ref should auto-link to the first row.
	if second.Assessment.SupersedesRef == nil || *second.Assessment.SupersedesRef != first.Assessment.ID {
		t.Errorf("expected supersedes_ref to auto-link to first row")
	}
	cur, err := s.GetCurrentCapacity(context.Background(), rid, models.CapacityDomainMedical)
	if err != nil {
		t.Fatalf("GetCurrent: %v", err)
	}
	if cur.ID != second.Assessment.ID {
		t.Errorf("expected current=second; got %s", cur.ID)
	}
}

func TestCapacityStore_GetCurrentNoHistoryReturnsNotFound(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()

	_, err := s.GetCurrentCapacity(context.Background(), uuid.New(), models.CapacityDomainMedical)
	if !errors.Is(err, interfaces.ErrNotFound) {
		t.Errorf("expected ErrNotFound; got %v", err)
	}
}

func TestCapacityStore_RejectsInvalidCrossField(t *testing.T) {
	s, db := openTestCapacityStore(t)
	defer db.Close()
	// intact + temporary is rejected by validator.
	bad := models.CapacityAssessment{
		ResidentRef:     uuid.New(),
		AssessedAt:      time.Now().UTC(),
		AssessorRoleRef: uuid.New(),
		Domain:          models.CapacityDomainMedical,
		Outcome:         models.CapacityOutcomeIntact,
		Duration:        models.CapacityDurationTemporary,
	}
	rev := bad.AssessedAt.Add(48 * time.Hour)
	bad.ExpectedReviewDate = &rev
	if _, err := s.CreateCapacityAssessment(context.Background(), bad); err == nil {
		t.Errorf("expected error for intact+temporary")
	}
}
