// Package integration — end_to_end_s2_layer_1_test.go is the headline
// end-to-end gate for the S2 Layer 1 build (Task 10 of the build plan).
//
// What this file proves
// ---------------------
//
// The s2-aggregator's Layer 1 view assembly works end-to-end against the
// Step 1–4 substrate: kb-20 ClinicalSnapshot tables, kb-32
// source_versions + recommendation_citations, and the
// shared/migrations/047 failed_intervention_records table. Three
// scenarios are covered:
//
//   - TestS2Layer1_E2E_FullViewAssembly: Worklist entry → every panel
//     populates against real Postgres seeds, citation pin set surfaces,
//     FIR record retrieves, ADD-on-palliative goals-conflict detected,
//     audit emitter receives EventViewRender, EscalationEvent emitter
//     NOT triggered.
//
//   - TestS2Layer1_E2E_OverrideRoundTrip: action handler executes
//     `override`, writes a row to s2-aggregator's pharmacist_actions
//     table (real DB), forwards to a stubbed OverrideForwarder with
//     normalized dual-vocab codes, and emits the
//     EventPharmacistAction audit row.
//
//   - TestS2Layer1_E2E_DrillThrough: seeded observation drill-throughs
//     via GetSubstrateObservation + GetTrajectoryHistory return the
//     seeded rows; AuditedDrillThrough emits EventDrillThrough rows.
//
// Skip discipline
// ---------------
//
// All three tests skip cleanly when VAIDSHALA_TEST_DSN is unset. The
// pattern mirrors kb-32-recommendation-craft/tests/integration/
// end_to_end_with_real_stores_test.go (Phase 2-completion Task 8) —
// CI without a DB still runs `go test ./...` clean.
//
// Isolation
// ---------
//
// Every seeded row is UUID-keyed and torn down via t.Cleanup. Parallel
// runs against the same database do not collide. Cleanup is FK-order-
// correct: recommendation_citations before source_versions, etc.
//
// Hard-constraint notes
// ---------------------
//
//   - DO NOT import kb-32 or shared internal packages. Substrate
//     seeding is raw parameterized SQL only (matches the Task 8 kb-32
//     pattern). The s2-aggregator's own SubstrateClient interface is
//     satisfied by a test-local adapter (postgresAdapter below) — the
//     production adapter is future-task wiring.
//
//   - DO NOT make real cross-service HTTP calls. The OverrideForwarder
//     test double records calls in memory; the assertion is "forwarder
//     was invoked with normalized codes," not "kb-32 received it."
//
//   - DO NOT modify production code outside this file. The adapter and
//     fetcher live inside the test file as test-private types.
package integration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/s2-aggregator/internal/actions"
	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/cardiofit/s2-aggregator/internal/audit"
	"github.com/cardiofit/s2-aggregator/internal/drill_through"
	"github.com/cardiofit/s2-aggregator/internal/entry_paths"
	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// ---------------------------------------------------------------------------
// Test-DB plumbing
// ---------------------------------------------------------------------------

// openE2ETestDB opens *sql.DB against VAIDSHALA_TEST_DSN or skips the test.
// db.Close is registered via t.Cleanup so callers need not defer Close.
func openE2ETestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping S2 Layer 1 end-to-end integration test")
	}
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		t.Fatalf("sql.Open: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		t.Fatalf("db.Ping: %v", err)
	}
	return db
}

// ---------------------------------------------------------------------------
// kb-20 / kb-32 / shared substrate seeding (raw SQL)
//
// Schema discovery: Phase 2-completion Task 1's PostgresSubstrateClient
// documents the following kb-20 tables — we seed against them directly
// rather than importing kb-20 GORM models (the s2-aggregator is a
// separate Go module and kb-20 internals are unimportable). The same
// table names + columns are used by kb-32's
// end_to_end_with_real_stores_test.go seed helper (Phase 2-completion
// Task 8).
//
//   - cfs_scores            (resident_ref, assessed_at, score, ...)
//   - dbi_scores            (resident_ref, computed_at, score, ...)
//   - acb_scores            (resident_ref, computed_at, score)
//   - lab_entries           (patient_id::TEXT, lab_type, value, unit, measured_at)
//   - care_intensity_history(resident_ref, tag, effective_date, documented_by_role_ref)
//   - active_concerns       (resident_id, concern_type, started_at, ...)
//
//   - kb-32: source_versions, recommendation_citations (migration 043)
//   - shared: failed_intervention_records (migration 047)
//   - s2-aggregator: pharmacist_actions (s2 migration 002)
// ---------------------------------------------------------------------------

// e2eSeedSnapshot describes the kb-20 substrate rows we seed for one
// resident across the three tests. Fields chosen so each test can
// exercise the relevant panel without bringing in unrelated columns.
type e2eSeedSnapshot struct {
	cfs           int
	dbi           float64
	acb           int
	egfr          float64
	careIntensity string // "active_treatment" | "rehabilitation" | "comfort_focused" | "palliative"
}

// seedClinicalSnapshot inserts kb-20 rows + the matching care-intensity
// entry that the s2 postgresAdapter reads as the goals-of-care state.
// Cleanup tears every row down. The signature matches the kb-32
// reference helper byte-for-byte for vocabulary parity.
func seedClinicalSnapshot(t *testing.T, db *sql.DB, residentID uuid.UUID, seed e2eSeedSnapshot) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	roleRef := uuid.New()

	mustExec := func(stmt string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, stmt, args...); err != nil {
			t.Fatalf("seed (%s): %v", stmt, err)
		}
	}

	mustExec(`INSERT INTO cfs_scores
	            (resident_ref, assessed_at, assessor_role_ref, instrument_version, score)
	          VALUES ($1, $2, $3, 'rockwood-2020', $4)`,
		residentID, now, roleRef, seed.cfs)
	mustExec(`INSERT INTO dbi_scores
	            (resident_ref, computed_at, score, anticholinergic_component, sedative_component)
	          VALUES ($1, $2, $3, 0, 0)`,
		residentID, now, seed.dbi)
	mustExec(`INSERT INTO acb_scores (resident_ref, computed_at, score)
	          VALUES ($1, $2, $3)`,
		residentID, now, seed.acb)
	mustExec(`INSERT INTO care_intensity_history
	            (resident_ref, tag, effective_date, documented_by_role_ref)
	          VALUES ($1, $2, $3, $4)`,
		residentID, seed.careIntensity, now, roleRef)
	mustExec(`INSERT INTO lab_entries (patient_id, lab_type, value, unit, measured_at)
	          VALUES ($1, 'egfr', $2, 'mL/min/1.73m2', $3)`,
		residentID.String(), seed.egfr, now.Add(-24*time.Hour))

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		for _, stmt := range []string{
			`DELETE FROM cfs_scores WHERE resident_ref = $1`,
			`DELETE FROM dbi_scores WHERE resident_ref = $1`,
			`DELETE FROM acb_scores WHERE resident_ref = $1`,
			`DELETE FROM care_intensity_history WHERE resident_ref = $1`,
		} {
			_, _ = db.ExecContext(ctx, stmt, residentID)
		}
		_, _ = db.ExecContext(ctx, `DELETE FROM lab_entries WHERE patient_id = $1`, residentID.String())
	})
}

// seedSourceVersionAndCitation inserts an active source_versions row and
// pins a recommendation_citations row against it. Cleanup tears both
// down. The kb-32 recommendations packet itself stays in-memory: kb-32
// does not maintain a recommendations table (the FK in migration 043 is
// only to source_versions per the migration comment) — packets are
// runtime artefacts.
func seedSourceVersionAndCitation(
	t *testing.T,
	db *sql.DB,
	sourceID string,
	recID uuid.UUID,
) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO source_versions
		(source_id, version, effective_from, effective_to, content_hash, status)
		VALUES ($1, '1', $2, NULL, 'e2e-s2-hash', 'active')`,
		sourceID, now.Add(-1*time.Hour),
	); err != nil {
		t.Fatalf("seed source_versions: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO recommendation_citations
		(recommendation_id, source_id, version, pinned_at)
		VALUES ($1, $2, '1', $3)`,
		recID, sourceID, now,
	); err != nil {
		t.Fatalf("seed recommendation_citations: %v", err)
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		// FK-order: citations first, then source_versions.
		_, _ = db.ExecContext(ctx,
			`DELETE FROM recommendation_citations WHERE source_id = $1`, sourceID)
		_, _ = db.ExecContext(ctx,
			`DELETE FROM source_versions WHERE source_id = $1`, sourceID)
	})
}

// seedFailedInterventionRecord inserts a FIR row keyed by residentID so
// the s2 postgresAdapter's FailedInterventionHistory can retrieve it.
// In production this row's resident_id is uuid.Nil (the Step 4 Task B
// documented gap); seeding it directly with the real resident-id lets
// us exercise the populated-panel path the production adapter cannot.
func seedFailedInterventionRecord(t *testing.T, db *sql.DB, residentID uuid.UUID) uuid.UUID {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	rowID := uuid.New()
	now := time.Now().UTC()
	if _, err := db.ExecContext(ctx, `INSERT INTO failed_intervention_records
		(id, resident_id, intervention_type, attempt_date, outcome, documented_reason,
		 retry_eligible_date, documented_by)
		VALUES ($1, $2, 'antipsychotic_deprescribing', $3, 'reversed_due_to_BPSD_recurrence',
		        'e2e seed', $4, $5)`,
		rowID, residentID, now.AddDate(0, -2, 0), now.AddDate(0, 10, 0), uuid.New(),
	); err != nil {
		t.Fatalf("seed failed_intervention_records: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx,
			`DELETE FROM failed_intervention_records WHERE id = $1`, rowID)
	})
	return rowID
}

// ---------------------------------------------------------------------------
// Postgres-backed SubstrateClient adapter (test-private)
//
// This is the minimal adapter the integration tests need to drive the
// aggregation pipeline against a real database. The production adapter
// is future-task wiring (per Task 10 instructions: "the production
// adapter is future work; for the test, a one-off adapter is fine").
//
// Design notes:
//   - Pending recommendations are NOT read from a kb-32 recommendations
//     table because no such table exists. They're carried in-memory and
//     keyed by SnapshotRef==residentID so the SubstrateClient contract
//     is honoured.
//   - Citations and FIR are read from real Postgres rows.
//   - GoC + care-intensity are derived from care_intensity_history per
//     the vocabulary-discovery comment in
//     internal/substrate_types/goals_of_care.go.
//   - Trajectories come from lab_entries (patient_id, lab_type, value,
//     measured_at).
// ---------------------------------------------------------------------------

type postgresAdapter struct {
	db *sql.DB

	// In-memory carry-over for substrate types kb-20/kb-32 does not yet
	// expose via the same table: pending recommendations, restraint
	// signals, override history, family meeting dates, PRN
	// administrations. These can be seeded per-test via the With*
	// methods so each test composes its own scenario.
	packets           []substrate_types.RecommendationPacket
	overrides         map[uuid.UUID][]substrate_types.OverrideReason
	restraintByRes    map[uuid.UUID][]substrate_types.RestraintSignal
	administrations   []substrate_types.PRNAdministration
	lastFamilyMeeting map[uuid.UUID]time.Time
	firAvailable      bool
}

func newPostgresAdapter(db *sql.DB) *postgresAdapter {
	return &postgresAdapter{
		db:                db,
		overrides:         map[uuid.UUID][]substrate_types.OverrideReason{},
		restraintByRes:    map[uuid.UUID][]substrate_types.RestraintSignal{},
		lastFamilyMeeting: map[uuid.UUID]time.Time{},
		firAvailable:      true, // tests seed real rows; flag mirrors production-adapter knob
	}
}

func (a *postgresAdapter) withPackets(pkts ...substrate_types.RecommendationPacket) *postgresAdapter {
	a.packets = append(a.packets, pkts...)
	return a
}

func (a *postgresAdapter) SnapshotFor(_ context.Context, residentID uuid.UUID, asOf time.Time) (aggregation.Snapshot, error) {
	snap := aggregation.Snapshot{ResidentID: residentID, AsOf: asOf}
	// eGFR from lab_entries.
	var egfr sql.NullFloat64
	if err := a.db.QueryRow(`SELECT value FROM lab_entries
		WHERE patient_id = $1 AND lab_type = 'egfr' AND measured_at <= $2
		ORDER BY measured_at DESC LIMIT 1`,
		residentID.String(), asOf,
	).Scan(&egfr); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return snap, fmt.Errorf("SnapshotFor.egfr: %w", err)
	}
	if egfr.Valid {
		v := egfr.Float64
		snap.EGFR = &v
	}
	// CFS / DBI / ACB from their dedicated tables.
	var cfs sql.NullFloat64
	_ = a.db.QueryRow(`SELECT score::float FROM cfs_scores
		WHERE resident_ref = $1 AND assessed_at <= $2
		ORDER BY assessed_at DESC LIMIT 1`, residentID, asOf).Scan(&cfs)
	if cfs.Valid {
		v := cfs.Float64
		snap.CFS = &v
	}
	var dbiv sql.NullFloat64
	_ = a.db.QueryRow(`SELECT score::float FROM dbi_scores
		WHERE resident_ref = $1 AND computed_at <= $2
		ORDER BY computed_at DESC LIMIT 1`, residentID, asOf).Scan(&dbiv)
	if dbiv.Valid {
		v := dbiv.Float64
		snap.DBI = &v
	}
	var acbv sql.NullFloat64
	_ = a.db.QueryRow(`SELECT score::float FROM acb_scores
		WHERE resident_ref = $1 AND computed_at <= $2
		ORDER BY computed_at DESC LIMIT 1`, residentID, asOf).Scan(&acbv)
	if acbv.Valid {
		v := acbv.Float64
		snap.ACB = &v
	}
	return snap, nil
}

func (a *postgresAdapter) TrajectoryHistory(_ context.Context, residentID uuid.UUID, parameter string) ([]substrate_types.Observation, error) {
	out := []substrate_types.Observation{}
	switch parameter {
	case "egfr":
		rows, err := a.db.Query(`SELECT value, unit, measured_at
			FROM lab_entries WHERE patient_id = $1 AND lab_type = $2
			ORDER BY measured_at ASC`, residentID.String(), parameter)
		if err != nil {
			return nil, fmt.Errorf("TrajectoryHistory: %w", err)
		}
		defer rows.Close()
		for rows.Next() {
			var v float64
			var unit string
			var at time.Time
			if err := rows.Scan(&v, &unit, &at); err != nil {
				return nil, err
			}
			out = append(out, substrate_types.Observation{
				ID:         uuid.New(), // synthesised — lab_entries does not expose a stable UUID via this view
				ResidentID: residentID,
				Parameter:  parameter,
				Value:      v,
				Unit:       unit,
				ObservedAt: at,
				Source:     "kb-20",
				Confidence: "high",
			})
		}
		return out, rows.Err()
	case "cfs":
		rows, err := a.db.Query(`SELECT score::float, assessed_at FROM cfs_scores
			WHERE resident_ref = $1 ORDER BY assessed_at ASC`, residentID)
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var v float64
			var at time.Time
			if err := rows.Scan(&v, &at); err != nil {
				return nil, err
			}
			out = append(out, substrate_types.Observation{
				ID: uuid.New(), ResidentID: residentID, Parameter: parameter,
				Value: v, Unit: "rockwood-2020", ObservedAt: at,
				Source: "kb-20", Confidence: "high",
			})
		}
		return out, rows.Err()
	}
	// Parameters without a backing table return empty.
	return out, nil
}

func (a *postgresAdapter) RecentPRNAdministrations(_ context.Context, residentID uuid.UUID, class substrate_types.PRNClass, asOf time.Time) ([]substrate_types.PRNAdministration, error) {
	cutoff := asOf.Add(-120 * 24 * time.Hour)
	out := []substrate_types.PRNAdministration{}
	for _, adm := range a.administrations {
		if adm.ResidentID != residentID || adm.Class != class {
			continue
		}
		if adm.AdministeredAt.After(cutoff) && !adm.AdministeredAt.After(asOf) {
			out = append(out, adm)
		}
	}
	return out, nil
}

func (a *postgresAdapter) PendingRecommendations(_ context.Context, residentID uuid.UUID) ([]substrate_types.RecommendationPacket, error) {
	out := []substrate_types.RecommendationPacket{}
	for _, p := range a.packets {
		if p.SnapshotRef == residentID {
			out = append(out, p)
		}
	}
	return out, nil
}

func (a *postgresAdapter) RecommendationAssessment(_ context.Context, _ uuid.UUID) (substrate_types.AssessmentScores, error) {
	return substrate_types.AssessmentScores{}, nil
}

func (a *postgresAdapter) RecommendationCitations(_ context.Context, recID uuid.UUID) ([]substrate_types.Citation, error) {
	rows, err := a.db.Query(`SELECT source_id, version, pinned_at
		FROM recommendation_citations WHERE recommendation_id = $1
		ORDER BY pinned_at ASC`, recID)
	if err != nil {
		return nil, fmt.Errorf("RecommendationCitations: %w", err)
	}
	defer rows.Close()
	out := []substrate_types.Citation{}
	for rows.Next() {
		var sourceID, version string
		var pinnedAt time.Time
		if err := rows.Scan(&sourceID, &version, &pinnedAt); err != nil {
			return nil, err
		}
		out = append(out, substrate_types.Citation{
			RecommendationID: recID.String(),
			SourceID:         sourceID,
			Version:          version,
			PinnedAt:         pinnedAt,
		})
	}
	return out, rows.Err()
}

func (a *postgresAdapter) RecommendationOverrides(_ context.Context, recID uuid.UUID) ([]substrate_types.OverrideReason, error) {
	ors := a.overrides[recID]
	out := make([]substrate_types.OverrideReason, len(ors))
	copy(out, ors)
	sort.Slice(out, func(i, j int) bool { return out[i].CapturedAt.Before(out[j].CapturedAt) })
	return out, nil
}

func (a *postgresAdapter) ActiveRestraintSignals(_ context.Context, residentID uuid.UUID) ([]substrate_types.RestraintSignal, error) {
	sigs := a.restraintByRes[residentID]
	out := make([]substrate_types.RestraintSignal, len(sigs))
	copy(out, sigs)
	return out, nil
}

func (a *postgresAdapter) FailedInterventionHistory(_ context.Context, residentID uuid.UUID, since time.Time) ([]substrate_types.FailedInterventionRecord, error) {
	rows, err := a.db.Query(`SELECT resident_id, intervention_type, attempt_date, outcome,
		documented_reason, retry_eligible_date, documented_by
		FROM failed_intervention_records
		WHERE resident_id = $1 AND attempt_date >= $2
		ORDER BY attempt_date DESC`, residentID, since)
	if err != nil {
		return nil, fmt.Errorf("FailedInterventionHistory: %w", err)
	}
	defer rows.Close()
	out := []substrate_types.FailedInterventionRecord{}
	for rows.Next() {
		var r substrate_types.FailedInterventionRecord
		if err := rows.Scan(&r.ResidentID, &r.InterventionType, &r.AttemptDate,
			&r.Outcome, &r.DocumentedReason, &r.RetryEligibleDate, &r.DocumentedBy); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

func (a *postgresAdapter) FailedInterventionRetrievalAvailable() bool { return a.firAvailable }

// CurrentGoalsOfCare derives GoC from the most-recent
// care_intensity_history row (per goals_of_care.go vocabulary-discovery
// note: kb-20 has no separate GoC state machine; care_intensity IS the
// substrate signal in Phase 1).
func (a *postgresAdapter) CurrentGoalsOfCare(_ context.Context, residentID uuid.UUID) (*substrate_types.GoalsOfCareEntry, error) {
	var tag string
	var effective time.Time
	var docBy uuid.UUID
	err := a.db.QueryRow(`SELECT tag, effective_date, documented_by_role_ref
		FROM care_intensity_history WHERE resident_ref = $1
		ORDER BY effective_date DESC LIMIT 1`, residentID,
	).Scan(&tag, &effective, &docBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("CurrentGoalsOfCare: %w", err)
	}
	return &substrate_types.GoalsOfCareEntry{
		State:         tag,
		EffectiveFrom: effective,
		DocumentedBy:  docBy,
		SubstrateID:   residentID, // synthesised stable ref for substrate-ref linkage
	}, nil
}

func (a *postgresAdapter) GoalsOfCareHistory(_ context.Context, residentID uuid.UUID) ([]substrate_types.GoalsOfCareEntry, error) {
	rows, err := a.db.Query(`SELECT tag, effective_date, documented_by_role_ref
		FROM care_intensity_history WHERE resident_ref = $1
		ORDER BY effective_date ASC`, residentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []substrate_types.GoalsOfCareEntry{}
	for rows.Next() {
		var tag string
		var eff time.Time
		var docBy uuid.UUID
		if err := rows.Scan(&tag, &eff, &docBy); err != nil {
			return nil, err
		}
		out = append(out, substrate_types.GoalsOfCareEntry{
			State: tag, EffectiveFrom: eff, DocumentedBy: docBy, SubstrateID: residentID,
		})
	}
	return out, rows.Err()
}

func (a *postgresAdapter) CurrentCareIntensity(_ context.Context, residentID uuid.UUID) (*substrate_types.CareIntensityEntry, error) {
	var tag string
	var effective time.Time
	var docBy uuid.UUID
	err := a.db.QueryRow(`SELECT tag, effective_date, documented_by_role_ref
		FROM care_intensity_history WHERE resident_ref = $1
		ORDER BY effective_date DESC LIMIT 1`, residentID,
	).Scan(&tag, &effective, &docBy)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &substrate_types.CareIntensityEntry{
		Tag: tag, EffectiveDate: effective, DocumentedBy: docBy, SubstrateID: residentID,
	}, nil
}

func (a *postgresAdapter) CareIntensityHistory(_ context.Context, residentID uuid.UUID) ([]substrate_types.CareIntensityEntry, error) {
	rows, err := a.db.Query(`SELECT tag, effective_date, documented_by_role_ref
		FROM care_intensity_history WHERE resident_ref = $1
		ORDER BY effective_date ASC`, residentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []substrate_types.CareIntensityEntry{}
	for rows.Next() {
		var tag string
		var eff time.Time
		var docBy uuid.UUID
		if err := rows.Scan(&tag, &eff, &docBy); err != nil {
			return nil, err
		}
		out = append(out, substrate_types.CareIntensityEntry{
			Tag: tag, EffectiveDate: eff, DocumentedBy: docBy, SubstrateID: residentID,
		})
	}
	return out, rows.Err()
}

func (a *postgresAdapter) LastFamilyMeetingDate(_ context.Context, residentID uuid.UUID) (*time.Time, error) {
	t, ok := a.lastFamilyMeeting[residentID]
	if !ok {
		return nil, nil
	}
	return &t, nil
}

// ---------------------------------------------------------------------------
// PostgresActionStore (test-private, satisfies actions.ActionStore)
//
// Writes a row to s2-aggregator's pharmacist_actions table (s2
// migration 002). The production store is Task-8 wiring; this inline
// implementation is sufficient to exercise the end-to-end override
// round-trip without depending on future work.
// ---------------------------------------------------------------------------

type postgresActionStore struct{ db *sql.DB }

func (s *postgresActionStore) Record(ctx context.Context, req actions.ActionRequest, ackID uuid.UUID) error {
	_, err := s.db.ExecContext(ctx, `INSERT INTO pharmacist_actions
		(id, pharmacist_id, resident_id, session_id, subject_id, action,
		 reasoning, override_reason_code, override_reason_code_short,
		 appropriateness_flag, note_body, captured_at, audit_trace_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)`,
		ackID, req.PharmacistID, req.ResidentID, req.SessionID,
		nullableUUID(req.SubjectID), string(req.Action),
		req.Reasoning, req.OverrideReasonCode, req.OverrideReasonCodeShort,
		req.AppropriatenessFlag, req.NoteBody, req.Timestamp, uuid.New(),
	)
	return err
}

func nullableUUID(u uuid.UUID) any {
	if u == uuid.Nil {
		return nil
	}
	return u
}

// ---------------------------------------------------------------------------
// Test-private OverrideForwarder + ObservationFetcher doubles
// ---------------------------------------------------------------------------

type recordingForwarder struct {
	mu     sync.Mutex
	calls  []actions.ActionRequest
}

func (f *recordingForwarder) Forward(_ context.Context, req actions.ActionRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.calls = append(f.calls, req)
	return nil
}
func (f *recordingForwarder) count() int { f.mu.Lock(); defer f.mu.Unlock(); return len(f.calls) }
func (f *recordingForwarder) last() (actions.ActionRequest, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if len(f.calls) == 0 {
		return actions.ActionRequest{}, false
	}
	return f.calls[len(f.calls)-1], true
}

// observationFetcherFn adapts a closure to drill_through.ObservationFetcher.
type observationFetcherFn func(ctx context.Context, id uuid.UUID) (substrate_types.Observation, error)

func (f observationFetcherFn) GetObservationByID(ctx context.Context, id uuid.UUID) (substrate_types.Observation, error) {
	return f(ctx, id)
}

// ---------------------------------------------------------------------------
// Test 1 — Full view assembly happy path
// ---------------------------------------------------------------------------

// TestS2Layer1_E2E_FullViewAssembly wires the s2-aggregator end-to-end
// against the Step 1–4 substrate and exercises the Worklist-entry happy
// path:
//
//   - Substrate (cfs/dbi/acb/lab_entries/care_intensity_history) is
//     seeded such that the resident is on palliative care intensity.
//   - One pending recommendation (Type=ADD) is registered with the
//     adapter and a citation is pinned via real source_versions +
//     recommendation_citations rows.
//   - A FIR row is seeded via failed_intervention_records.
//   - The Layer 1 view is assembled by calling each panel builder in
//     turn (mirrors tests/structural/verification_not_belief_test.go
//     buildTestS2View — there's no single AssembleView() entry point
//     in Phase 1; that unification is the
//     `TODO(layer 1 type unification)` in that file).
//
// Assertions:
//   - All panels populate (CAPE band, trajectories, pending recs, FIR
//     panel, GoC + care intensity, GoalsConflict).
//   - The pending-recommendation card carries the seeded citation pin.
//   - FIR retrieval availability is true (we seeded against the real
//     table with real resident_id) and the seeded row surfaces.
//   - GoalsConflict for ADD on palliative is detected per the v1.0
//     Part 9.4 / kb-32 substrate_scorer.go canonical anti-pattern.
//   - The audit MemoryEmitter received one EventViewRender row (via
//     the BuildLayer1Baseline path with WithViewRenderEmitter).
//   - The EscalationEventEmitter is NOT triggered (no Layer 1→3
//     escalation in this scenario).
func TestS2Layer1_E2E_FullViewAssembly(t *testing.T) {
	db := openE2ETestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	residentID := uuid.New()
	pharmID := uuid.New()
	sessionID := uuid.New()
	recID := uuid.New()

	// 1. Substrate: palliative resident with declining eGFR.
	seedClinicalSnapshot(t, db, residentID, e2eSeedSnapshot{
		cfs:           7,
		dbi:           0.5,
		acb:           3,
		egfr:          32,
		careIntensity: "palliative",
	})

	// 2. Citation pin set: real source_versions + recommendation_citations.
	sourceID := "E2E-S2-L1-" + uuid.NewString()
	seedSourceVersionAndCitation(t, db, sourceID, recID)

	// 3. FIR record.
	_ = seedFailedInterventionRecord(t, db, residentID)

	// 4. SubstrateClient adapter + in-memory pending packet (ADD on
	// palliative is the canonical anti-pattern that triggers
	// GoalsConflict per v1.0 Part 9.4).
	adapter := newPostgresAdapter(db).withPackets(substrate_types.RecommendationPacket{
		RecommendationID: recID,
		AuthorID:         pharmID,
		Type:             "ADD",
		Sections:         map[string]string{"layer_1": "ADD body L1", "layer_2": "ADD body L2", "layer_3": "ADD body L3"},
		AppliedRule:      substrate_types.AppliedRule{RuleID: "add-rule-1", Type: "ADD", Urgency: "green"},
		SnapshotRef:      residentID,
	})

	// 5. Entry path: Worklist with CAPE signal.
	meta, err := entry_paths.FromWorklist(ctx, pharmID, residentID, aggregation.WorklistContext{
		PrimarySignals: []string{"trajectory_velocity_4_egfr_decline"},
		CAPEScore:      0.78,
		TriagedAt:      time.Now().UTC().Add(-1 * time.Hour),
	})
	if err != nil {
		t.Fatalf("FromWorklist: %v", err)
	}

	asOf := time.Now().UTC()
	req := aggregation.WorkspaceRequest{
		ResidentID: residentID, PharmacistID: pharmID, SessionID: sessionID,
		AsOf: asOf, EntryPath: aggregation.EntryPathWorklist, EntryMetadata: meta,
	}

	// 6. Wire view-render audit emitter (MemoryEmitter for assertion).
	memEmitter := audit.NewMemoryEmitter()
	vb := aggregation.NewDefaultViewBuilder()
	vb = aggregation.WithViewRenderEmitter(vb, audit.NewViewRenderAdapter(memEmitter))

	if _, err := vb.BuildLayer1Baseline(ctx, req); err != nil {
		t.Fatalf("BuildLayer1Baseline: %v", err)
	}

	// 7. CAPE context band.
	band, err := aggregation.BuildCAPEContextBand(meta)
	if err != nil {
		t.Fatalf("BuildCAPEContextBand: %v", err)
	}
	if len(band.Signals) == 0 {
		t.Error("CAPE band: expected ≥1 signal from worklist entry")
	}

	// 8. Trajectories: eGFR + CFS series populate from real lab_entries +
	// cfs_scores.
	trs, err := aggregation.BuildTrajectories(ctx, adapter, residentID, asOf)
	if err != nil {
		t.Fatalf("BuildTrajectories: %v", err)
	}
	if len(trs) == 0 {
		t.Error("expected ≥1 trajectory rendered against real lab_entries")
	}

	// 9. Pending recommendation cards — assert citation pin surfaces.
	cards, err := aggregation.BuildPendingRecommendationCards(ctx, adapter, residentID, asOf)
	if err != nil {
		t.Fatalf("BuildPendingRecommendationCards: %v", err)
	}
	if len(cards) != 1 {
		t.Fatalf("expected exactly 1 pending recommendation card; got %d", len(cards))
	}
	if len(cards[0].Citations) != 1 {
		t.Errorf("expected 1 citation pin from real recommendation_citations; got %d", len(cards[0].Citations))
	} else if cards[0].Citations[0].SourceID != sourceID {
		t.Errorf("citation source mismatch: got %s want %s", cards[0].Citations[0].SourceID, sourceID)
	}

	// 10. FIR panel: real row should surface via the adapter.
	firPanel, err := aggregation.BuildFailedInterventionPanel(ctx, adapter, residentID, asOf)
	if err != nil {
		t.Fatalf("BuildFailedInterventionPanel: %v", err)
	}
	if !firPanel.RetrievalAvailable {
		t.Error("FIR retrieval should be available (adapter seeded with real rows)")
	}
	if len(firPanel.Cards) != 1 {
		t.Errorf("expected exactly 1 FIR card; got %d", len(firPanel.Cards))
	}

	// 11. GoC + care-intensity panels.
	gocPanel, err := aggregation.BuildGoalsOfCarePanel(ctx, adapter, residentID)
	if err != nil {
		t.Fatalf("BuildGoalsOfCarePanel: %v", err)
	}
	if gocPanel.Current == nil || gocPanel.Current.State != "palliative" {
		t.Errorf("GoC current state = %v; want palliative", gocPanel.Current)
	}
	ciPanel, err := aggregation.BuildCareIntensityPanel(ctx, adapter, residentID)
	if err != nil {
		t.Fatalf("BuildCareIntensityPanel: %v", err)
	}
	if ciPanel.Current == nil || ciPanel.Current.Tag != "palliative" {
		t.Errorf("care intensity current = %v; want palliative", ciPanel.Current)
	}

	// 12. Goals conflict — ADD on palliative MUST be detected per v1.0
	// Part 9.4 / kb-32 substrate_scorer.go anti-pattern.
	conflicts := aggregation.DetectGoalsConflicts(cards, gocPanel.Current)
	if len(conflicts) != 1 {
		t.Fatalf("expected 1 goals-of-care conflict (ADD on palliative); got %d", len(conflicts))
	}
	if conflicts[0].RecommendationID != recID {
		t.Errorf("conflict RecommendationID mismatch: got %s want %s", conflicts[0].RecommendationID, recID)
	}

	// 13. Audit: view-render row should have been emitted.
	viewRenders := memEmitter.EventsOfType(audit.EventViewRender)
	if len(viewRenders) != 1 {
		t.Errorf("expected exactly 1 EventViewRender; got %d", len(viewRenders))
	}
	// Escalation MemoryEmitter is NOT triggered in this scenario.
	if escs := memEmitter.EventsOfType(audit.EventCognitiveEscalation); len(escs) != 0 {
		t.Errorf("expected zero EventCognitiveEscalation rows; got %d", len(escs))
	}

	// 14. Smoke-test that the override pathway is reachable end-to-end
	// (validates the action handler composes with the in-process
	// adapters built above). The detailed assertions live in Test 2.
	actionStore := &postgresActionStore{db: db}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx,
			`DELETE FROM pharmacist_actions WHERE pharmacist_id = $1`, pharmID)
	})
	sessions := actions.NewInMemorySessionStore()
	_, err = actions.StartSession(ctx, pharmID, sessions)
	if err != nil {
		t.Fatalf("StartSession: %v", err)
	}
	// Re-seat the session under our deterministic sessionID so the row
	// the override writes carries it.
	if err := sessions.Create(ctx, actions.SessionContext{SessionID: sessionID, PharmacistID: pharmID, StartedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("session Create: %v", err)
	}
	fwd := &recordingForwarder{}
	h := actions.NewHandler(actionStore, sessions, fwd, vb)
	if _, err := h.Execute(ctx, actions.ActionRequest{
		Action: actions.ActionOverride, PharmacistID: pharmID, ResidentID: residentID,
		SessionID: sessionID, SubjectID: recID,
		Reasoning: "smoke-test from full view assembly path",
		OverrideReasonCode: "clinical_judgment", OverrideReasonCodeShort: "CJG",
		AppropriatenessFlag: "appropriate",
	}); err != nil {
		t.Fatalf("override smoke-test Execute: %v", err)
	}
	if fwd.count() != 1 {
		t.Errorf("override smoke-test: forwarder count = %d; want 1", fwd.count())
	}
}

// ---------------------------------------------------------------------------
// Test 2 — Override round-trip
// ---------------------------------------------------------------------------

// TestS2Layer1_E2E_OverrideRoundTrip exercises the override pathway
// end-to-end:
//
//   - Substrate seeded same as Test 1 (active resident this time so the
//     conflict-detection logic does not interfere with the action path)
//   - actions.Handler.Execute(override) is invoked with a Guidelines
//     Part 5 short code so the dual-vocab normalizer fires.
//   - Assertions: pharmacist_actions row was persisted with normalized
//     codes; OverrideForwarder was invoked exactly once with the
//     normalized codes; the action audit emitter received an
//     EventPharmacistAction row keyed by AuditTraceID.
func TestS2Layer1_E2E_OverrideRoundTrip(t *testing.T) {
	db := openE2ETestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	residentID := uuid.New()
	pharmID := uuid.New()
	sessionID := uuid.New()
	recID := uuid.New()

	seedClinicalSnapshot(t, db, residentID, e2eSeedSnapshot{
		cfs: 5, dbi: 0.3, acb: 1, egfr: 55, careIntensity: "active_treatment",
	})
	sourceID := "E2E-S2-L1-OVR-" + uuid.NewString()
	seedSourceVersionAndCitation(t, db, sourceID, recID)

	// Cleanup any pharmacist_actions row this test writes.
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx,
			`DELETE FROM pharmacist_actions WHERE pharmacist_id = $1`, pharmID)
	})

	store := &postgresActionStore{db: db}
	sessions := actions.NewInMemorySessionStore()
	if err := sessions.Create(ctx, actions.SessionContext{SessionID: sessionID, PharmacistID: pharmID, StartedAt: time.Now().UTC()}); err != nil {
		t.Fatalf("session Create: %v", err)
	}
	fwd := &recordingForwarder{}
	memEmitter := audit.NewMemoryEmitter()
	vb := aggregation.NewDefaultViewBuilder()

	h := actions.NewHandler(store, sessions, fwd, vb).WithAuditEmitter(memEmitter)

	// Use the 3-letter Guidelines Part 5 short code "ALF" (alert fatigue)
	// to exercise the dual-vocab normalizer. NormalizeOverrideCodes
	// expects exactly one of snake or short to be supplied; passing the
	// short form alone is the most ergonomic UI shape.
	req := actions.ActionRequest{
		Action:                  actions.ActionOverride,
		PharmacistID:            pharmID,
		ResidentID:              residentID,
		SessionID:               sessionID,
		SubjectID:               recID,
		Reasoning:               "alert fatigue — declining repeat prompts on this resident",
		OverrideReasonCodeShort: "ALF",
		AppropriatenessFlag:     "inappropriate",
	}
	ack, err := h.Execute(ctx, req)
	if err != nil {
		t.Fatalf("Execute(override): %v", err)
	}

	// 1. Forwarder invoked once.
	if fwd.count() != 1 {
		t.Fatalf("forwarder calls = %d; want 1", fwd.count())
	}
	fwded, _ := fwd.last()
	if fwded.OverrideReasonCodeShort != "ALF" {
		t.Errorf("forwarded short code = %q; want ALF", fwded.OverrideReasonCodeShort)
	}
	if fwded.OverrideReasonCode == "" {
		t.Errorf("forwarded snake_case code should be populated post-normalization; got empty")
	} else if !strings.Contains(strings.ToLower(fwded.OverrideReasonCode), "alert_fatigue") {
		t.Errorf("forwarded snake_case code = %q; expected to contain 'alert_fatigue'",
			fwded.OverrideReasonCode)
	}

	// 2. pharmacist_actions row persisted with normalized codes.
	var (
		gotAction, gotSnake, gotShort string
	)
	if err := db.QueryRowContext(ctx, `SELECT action, override_reason_code, override_reason_code_short
		FROM pharmacist_actions WHERE id = $1`, ack.ActionID,
	).Scan(&gotAction, &gotSnake, &gotShort); err != nil {
		t.Fatalf("read pharmacist_actions: %v", err)
	}
	if gotAction != string(actions.ActionOverride) {
		t.Errorf("row action = %q; want override", gotAction)
	}
	if gotShort != "ALF" || !strings.Contains(strings.ToLower(gotSnake), "alert_fatigue") {
		t.Errorf("row codes = (%q, %q); expected ('alert_fatigue', 'ALF')", gotSnake, gotShort)
	}

	// 3. Audit emitter received exactly one EventPharmacistAction.
	pa := memEmitter.EventsOfType(audit.EventPharmacistAction)
	if len(pa) != 1 {
		t.Fatalf("expected 1 EventPharmacistAction; got %d", len(pa))
	}
	if pa[0].Subject != string(actions.ActionOverride) {
		t.Errorf("audit Subject = %q; want override", pa[0].Subject)
	}
	if pa[0].TraceID != ack.AuditTraceID {
		t.Errorf("audit TraceID mismatch: got %s want %s", pa[0].TraceID, ack.AuditTraceID)
	}
}

// ---------------------------------------------------------------------------
// Test 3 — Drill-through
// ---------------------------------------------------------------------------

// TestS2Layer1_E2E_DrillThrough exercises the v1.0 Part 10 drill-through
// pattern end-to-end against real lab_entries + a substrate-observation
// fetcher backed by the real DB.
//
// Coverage:
//   - GetSubstrateObservation returns the seeded row via an
//     ObservationFetcher backed by lab_entries.
//   - GetTrajectoryHistory returns the chronological lab_entries series
//     for the resident.
//   - AuditedDrillThrough emits EventDrillThrough rows for both calls.
func TestS2Layer1_E2E_DrillThrough(t *testing.T) {
	db := openE2ETestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	residentID := uuid.New()
	pharmID := uuid.New()
	sessionID := uuid.New()

	// Seed two lab_entries rows so trajectory history has a real series.
	now := time.Now().UTC()
	mustExec := func(stmt string, args ...any) {
		t.Helper()
		if _, err := db.ExecContext(ctx, stmt, args...); err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	mustExec(`INSERT INTO lab_entries (patient_id, lab_type, value, unit, measured_at)
	          VALUES ($1, 'egfr', $2, 'mL/min/1.73m2', $3)`,
		residentID.String(), 55.0, now.Add(-60*24*time.Hour))
	mustExec(`INSERT INTO lab_entries (patient_id, lab_type, value, unit, measured_at)
	          VALUES ($1, 'egfr', $2, 'mL/min/1.73m2', $3)`,
		residentID.String(), 45.0, now.Add(-30*24*time.Hour))
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `DELETE FROM lab_entries WHERE patient_id = $1`, residentID.String())
	})

	adapter := newPostgresAdapter(db)
	memEmitter := audit.NewMemoryEmitter()
	auditor := drill_through.NewAuditedDrillThrough(memEmitter)

	// 1. Trajectory history drill-through.
	hist, err := drill_through.GetTrajectoryHistory(ctx, adapter, residentID, "egfr")
	if err != nil {
		t.Fatalf("GetTrajectoryHistory: %v", err)
	}
	if len(hist.Observations) != 2 {
		t.Fatalf("expected 2 observations in trajectory history; got %d", len(hist.Observations))
	}
	if !hist.Observations[0].ObservedAt.Before(hist.Observations[1].ObservedAt) {
		t.Errorf("trajectory history not chronological")
	}
	auditor.RecordTrajectoryHistory(ctx, pharmID, residentID, sessionID, "egfr")

	// 2. Substrate observation drill-through against a fetcher that reads
	// lab_entries by a synthesised id (we look up by parameter+resident
	// since lab_entries does not expose a stable UUID via this view).
	fetcher := observationFetcherFn(func(_ context.Context, _ uuid.UUID) (substrate_types.Observation, error) {
		var v float64
		var unit string
		var at time.Time
		err := db.QueryRow(`SELECT value, unit, measured_at FROM lab_entries
			WHERE patient_id = $1 AND lab_type = 'egfr'
			ORDER BY measured_at DESC LIMIT 1`, residentID.String(),
		).Scan(&v, &unit, &at)
		if err != nil {
			return substrate_types.Observation{}, err
		}
		return substrate_types.Observation{
			ID: uuid.New(), ResidentID: residentID, Parameter: "egfr",
			Value: v, Unit: unit, ObservedAt: at, Source: "kb-20", Confidence: "high",
		}, nil
	})
	ref := aggregation.SubstrateRef{Source: "kb-20", ID: uuid.New(), Description: "egfr most-recent"}
	obs, err := drill_through.GetSubstrateObservation(ctx, fetcher, ref, nil)
	if err != nil {
		t.Fatalf("GetSubstrateObservation: %v", err)
	}
	if obs.Observation.Value != 45.0 {
		t.Errorf("most-recent eGFR = %v; want 45.0", obs.Observation.Value)
	}
	auditor.RecordSubstrateObservation(ctx, pharmID, residentID, sessionID, ref)

	// 3. Audit emitter received both EventDrillThrough rows.
	drills := memEmitter.EventsOfType(audit.EventDrillThrough)
	if len(drills) != 2 {
		t.Errorf("expected 2 EventDrillThrough rows; got %d", len(drills))
	}
}
