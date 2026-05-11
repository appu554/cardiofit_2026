package failed_interventions_test

import (
	"context"
	"database/sql"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	fi "github.com/cardiofit/shared/v2_substrate/failed_interventions"
)

// -----------------------------------------------------------------------
// InMemoryStore — always runs, no DSN required.
// -----------------------------------------------------------------------

func TestInMemoryStore_RecordAndList(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := fi.NewInMemoryStore()
	rid := uuid.New()
	doc := uuid.New()
	now := time.Now().UTC()

	rec := fi.FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "antipsychotic_deprescribing",
		AttemptDate:       now.Add(-30 * 24 * time.Hour),
		Outcome:           fi.OutcomeReversedDueToBPSDRecurrence,
		DocumentedReason:  "BPSD returned within 14d of stop",
		RetryEligibleDate: now.Add(335 * 24 * time.Hour),
		DocumentedBy:      doc,
	}
	if err := s.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}
	got, err := s.ListByResident(ctx, rid)
	if err != nil {
		t.Fatalf("ListByResident: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("ListByResident: got %d records, want 1", len(got))
	}
	if got[0].InterventionType != rec.InterventionType || got[0].DocumentedBy != doc {
		t.Errorf("round-trip mismatch: got %+v want %+v", got[0], rec)
	}
}

func TestInMemoryStore_IsVetoActive(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := fi.NewInMemoryStore()
	rid := uuid.New()
	otherRid := uuid.New()
	doc := uuid.New()
	now := time.Now().UTC()

	mustRecord := func(r fi.FailedInterventionRecord) {
		if err := s.Record(ctx, r); err != nil {
			t.Fatalf("Record: %v", err)
		}
	}
	mustRecord(fi.FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "antipsychotic_deprescribing",
		RetryEligibleDate: now.Add(60 * 24 * time.Hour),
		DocumentedBy:      doc,
	})
	mustRecord(fi.FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "benzodiazepine_deprescribing",
		RetryEligibleDate: now.Add(-30 * 24 * time.Hour),
		DocumentedBy:      doc,
	})

	cases := []struct {
		name             string
		residentID       uuid.UUID
		interventionType string
		want             bool
	}{
		{"active veto", rid, "antipsychotic_deprescribing", true},
		{"expired veto", rid, "benzodiazepine_deprescribing", false},
		{"wrong intervention type", rid, "dose_reduction", false},
		{"no records for resident", otherRid, "antipsychotic_deprescribing", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			active, err := s.IsVetoActive(ctx, tc.residentID, tc.interventionType, now)
			if err != nil {
				t.Fatalf("IsVetoActive: %v", err)
			}
			if active != tc.want {
				t.Errorf("got %v want %v", active, tc.want)
			}
		})
	}
}

func TestInMemoryStore_ConcurrentRecord(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	s := fi.NewInMemoryStore()
	rid := uuid.New()
	doc := uuid.New()
	now := time.Now().UTC()

	const N = 100
	var wg sync.WaitGroup
	for i := 0; i < N; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			_ = s.Record(ctx, fi.FailedInterventionRecord{
				ResidentID:        rid,
				InterventionType:  "antipsychotic_deprescribing",
				AttemptDate:       now,
				RetryEligibleDate: now.Add(time.Duration(i) * time.Hour),
				DocumentedBy:      doc,
			})
		}(i)
	}
	wg.Wait()
	got, err := s.ListByResident(ctx, rid)
	if err != nil {
		t.Fatalf("ListByResident: %v", err)
	}
	if len(got) != N {
		t.Errorf("got %d records after concurrent Record, want %d", len(got), N)
	}
}

// -----------------------------------------------------------------------
// PostgresStore — skipped when VAIDSHALA_TEST_DSN is unset.
//
// Mirrors the kb-32 substrate_client_test.go DSN-skip pattern.
// -----------------------------------------------------------------------

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set — skipping Postgres-backed test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	if err := db.Ping(); err != nil {
		t.Skipf("Postgres unreachable at VAIDSHALA_TEST_DSN: %v", err)
	}
	return db
}

func TestPostgresStore_RecordAndList(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()
	s := fi.NewPostgresStore(db)
	rid := uuid.New()
	doc := uuid.New()
	now := time.Now().UTC()

	rec := fi.FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "antipsychotic_deprescribing",
		AttemptDate:       now.Add(-30 * 24 * time.Hour),
		Outcome:           fi.OutcomeReversedDueToBPSDRecurrence,
		DocumentedReason:  "test reversal",
		RetryEligibleDate: now.Add(335 * 24 * time.Hour),
		DocumentedBy:      doc,
	}
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM failed_intervention_records WHERE resident_id = $1`, rid)
	})
	if err := s.Record(ctx, rec); err != nil {
		t.Fatalf("Record: %v", err)
	}
	got, err := s.ListByResident(ctx, rid)
	if err != nil {
		t.Fatalf("ListByResident: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("got %d records, want 1", len(got))
	}
	if got[0].InterventionType != rec.InterventionType {
		t.Errorf("intervention_type mismatch: got %q want %q", got[0].InterventionType, rec.InterventionType)
	}
}

func TestPostgresStore_IsVetoActive(t *testing.T) {
	db := openTestDB(t)
	defer db.Close()
	ctx := context.Background()
	s := fi.NewPostgresStore(db)
	rid := uuid.New()
	doc := uuid.New()
	now := time.Now().UTC()
	t.Cleanup(func() {
		_, _ = db.ExecContext(ctx, `DELETE FROM failed_intervention_records WHERE resident_id = $1`, rid)
	})

	if err := s.Record(ctx, fi.FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "antipsychotic_deprescribing",
		AttemptDate:       now.Add(-30 * 24 * time.Hour),
		Outcome:           fi.OutcomeReversedDueToBPSDRecurrence,
		RetryEligibleDate: now.Add(60 * 24 * time.Hour),
		DocumentedBy:      doc,
	}); err != nil {
		t.Fatalf("Record active: %v", err)
	}
	if err := s.Record(ctx, fi.FailedInterventionRecord{
		ResidentID:        rid,
		InterventionType:  "benzodiazepine_deprescribing",
		AttemptDate:       now.Add(-400 * 24 * time.Hour),
		Outcome:           fi.OutcomeReversedDueToFrailty,
		RetryEligibleDate: now.Add(-30 * 24 * time.Hour),
		DocumentedBy:      doc,
	}); err != nil {
		t.Fatalf("Record expired: %v", err)
	}

	cases := []struct {
		name             string
		interventionType string
		want             bool
	}{
		{"active veto", "antipsychotic_deprescribing", true},
		{"case-insensitive active", "Antipsychotic_Deprescribing", true},
		{"expired veto", "benzodiazepine_deprescribing", false},
		{"unrelated intervention", "dose_reduction", false},
		{"empty intervention type", "", false},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			active, err := s.IsVetoActive(ctx, rid, tc.interventionType, now)
			if err != nil {
				t.Fatalf("IsVetoActive: %v", err)
			}
			if active != tc.want {
				t.Errorf("got %v want %v", active, tc.want)
			}
		})
	}
}
