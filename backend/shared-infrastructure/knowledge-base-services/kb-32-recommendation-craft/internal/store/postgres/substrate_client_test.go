package postgres

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	kb32ctx "github.com/cardiofit/kb32/internal/context"
)

// Compile-time interface conformance — does not require a database.
var _ kb32ctx.SubstrateClient = (*PostgresSubstrateClient)(nil)

// TestPostgresSubstrateClient_InterfaceConformance is a unit-level guard
// that does not require a live database. It catches signature drift between
// the kb32ctx.SubstrateClient port and PostgresSubstrateClient.SnapshotFor
// at test-discovery time.
func TestPostgresSubstrateClient_InterfaceConformance(t *testing.T) {
	t.Parallel()

	// NewPostgresSubstrateClient(nil) must succeed — error surfaces are
	// per-call, not per-constructor, so a nil *sql.DB is a valid (if
	// useless) construction.
	client := NewPostgresSubstrateClient(nil)
	if client == nil {
		t.Fatal("NewPostgresSubstrateClient(nil) returned nil; expected usable struct")
	}

	var _ kb32ctx.SubstrateClient = client
}

// TestTranslateCareIntensity exercises the kb-20→kb-32 vocabulary
// translation in isolation. No database required.
func TestTranslateCareIntensity(t *testing.T) {
	t.Parallel()

	cases := []struct {
		in, want string
	}{
		{"active_treatment", "active"},
		{"rehabilitation", "active"},
		{"comfort_focused", "comfort"},
		{"palliative", "palliative"},
		// Unknown values must pass through rather than be silently dropped.
		{"end_of_life", "end_of_life"},
		{"", ""},
	}
	for _, tc := range cases {
		if got := translateCareIntensity(tc.in); got != tc.want {
			t.Errorf("translateCareIntensity(%q) = %q; want %q", tc.in, got, tc.want)
		}
	}
}

// TestPostgresSubstrateClient_RoundTrip is the integration test. It seeds a
// small kb-20 fixture, queries through PostgresSubstrateClient, and asserts
// the ClinicalSnapshot is populated correctly.
//
// Skipped automatically when VAIDSHALA_TEST_DSN is unset so the suite
// remains green in environments without a Postgres instance available.
func TestPostgresSubstrateClient_RoundTrip(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}

	residentID := uuid.New()
	now := time.Now().UTC()

	// Best-effort cleanup so a re-run is idempotent.
	cleanup := func() {
		for _, stmt := range []string{
			`DELETE FROM cfs_scores WHERE resident_ref = $1`,
			`DELETE FROM dbi_scores WHERE resident_ref = $1`,
			`DELETE FROM acb_scores WHERE resident_ref = $1`,
			`DELETE FROM care_intensity_history WHERE resident_ref = $1`,
			`DELETE FROM capacity_assessments WHERE resident_ref = $1`,
			`DELETE FROM active_concerns WHERE resident_id = $1`,
		} {
			_, _ = db.ExecContext(ctx, stmt, residentID)
		}
		_, _ = db.ExecContext(ctx, `DELETE FROM lab_entries WHERE patient_id = $1`, residentID.String())
	}
	cleanup()
	defer cleanup()

	// Seed one row per relevant table.
	mustExec := func(stmt string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, stmt, args...); err != nil {
			t.Fatalf("seed (%s): %v", stmt, err)
		}
	}

	roleRef := uuid.New()

	mustExec(`INSERT INTO cfs_scores (resident_ref, assessed_at, assessor_role_ref, instrument_version, score)
	          VALUES ($1, $2, $3, 'rockwood-2020', 6)`,
		residentID, now, roleRef)
	mustExec(`INSERT INTO dbi_scores (resident_ref, computed_at, score, anticholinergic_component, sedative_component)
	          VALUES ($1, $2, 2.5, 1.5, 1.0)`,
		residentID, now)
	mustExec(`INSERT INTO acb_scores (resident_ref, computed_at, score)
	          VALUES ($1, $2, 3)`,
		residentID, now)
	mustExec(`INSERT INTO care_intensity_history
	            (resident_ref, tag, effective_date, documented_by_role_ref)
	          VALUES ($1, 'comfort_focused', $2, $3)`,
		residentID, now, roleRef)
	mustExec(`INSERT INTO lab_entries (patient_id, lab_type, value, unit, measured_at)
	          VALUES ($1, 'egfr', 42.0, 'mL/min/1.73m2', $2)`,
		residentID.String(), now)
	mustExec(`INSERT INTO active_concerns
	            (resident_id, concern_type, started_at, expected_resolution_at, resolution_status)
	          VALUES ($1, 'post_fall_72h', $2, $3, 'open')`,
		residentID, now.Add(-1*time.Hour), now.Add(71*time.Hour))

	client := NewPostgresSubstrateClient(db)
	snap, err := client.SnapshotFor(ctx, residentID)
	if err != nil {
		t.Fatalf("SnapshotFor: %v", err)
	}

	if snap.ResidentID != residentID {
		t.Errorf("ResidentID = %v; want %v", snap.ResidentID, residentID)
	}
	if snap.CFS != 6 {
		t.Errorf("CFS = %d; want 6", snap.CFS)
	}
	if snap.DBI != 2.5 {
		t.Errorf("DBI = %v; want 2.5", snap.DBI)
	}
	if snap.ACB != 3 {
		t.Errorf("ACB = %d; want 3", snap.ACB)
	}
	if snap.EGFR != 42.0 {
		t.Errorf("EGFR = %v; want 42.0", snap.EGFR)
	}
	if snap.CareIntensity != "comfort" {
		t.Errorf("CareIntensity = %q; want %q", snap.CareIntensity, "comfort")
	}
	if !snap.RecentFall72h {
		t.Error("RecentFall72h = false; want true")
	}
	if snap.RecentAdmission72h {
		t.Error("RecentAdmission72h = true; want false (no seed)")
	}
	if snap.AssessedAt.IsZero() {
		t.Error("AssessedAt is zero; want CFS assessed_at value")
	}
}

// TestPostgresSubstrateClient_MissingResident verifies the documented
// graceful-degradation contract: an unknown resident yields a zero-value
// ClinicalSnapshot (other than ResidentID and AssessedAt fallback), never
// an error.
func TestPostgresSubstrateClient_MissingResident(t *testing.T) {
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping Postgres integration test")
	}

	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	defer db.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	client := NewPostgresSubstrateClient(db)
	snap, err := client.SnapshotFor(ctx, uuid.New())
	if err != nil {
		t.Fatalf("SnapshotFor on missing resident: %v", err)
	}
	if snap.CFS != 0 || snap.DBI != 0 || snap.ACB != 0 || snap.EGFR != 0 {
		t.Errorf("expected zero-value scoring fields; got CFS=%d DBI=%v ACB=%d EGFR=%v",
			snap.CFS, snap.DBI, snap.ACB, snap.EGFR)
	}
	if snap.CareIntensity != "" {
		t.Errorf("CareIntensity = %q; want empty", snap.CareIntensity)
	}
	if snap.RecentFall72h || snap.RecentAdmission72h || snap.CapacityLapse {
		t.Error("expected all boolean signals false for missing resident")
	}
}
