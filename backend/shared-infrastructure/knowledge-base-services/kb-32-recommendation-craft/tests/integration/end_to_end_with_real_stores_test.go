// Package integration — end_to_end_with_real_stores_test.go wires every
// Phase 2-completion deliverable into a single pipeline run against a real
// Postgres instance and exercises the Sunday-night-fall scenario.
//
// Components composed:
//   - PostgresSubstrateClient (Task 1)   — Stage 1 substrate
//   - SubstrateBackedScorer  (Task 2)    — Stage 4 appropriateness gate
//   - PostgresRegistry       (Task 3)    — Stage 5b citation pinning
//   - CompositeEmitter(EthicsLog + Postgres) (Task 4) — Stage 7 audit trail
//   - PostgresOptOutStore    (Task 6)    — prescriber framing opt-out substrate
//
// Skipping: every test in this file skips cleanly when VAIDSHALA_TEST_DSN is
// unset, matching the precedent in substrate_client_test.go and
// evidence_trace_test.go. None of these tests fail in CI without a DB.
//
// Isolation: every seeded row is keyed by a freshly-generated UUID and torn
// down via t.Cleanup so parallel runs against the same database do not
// collide.
package integration

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"

	"github.com/cardiofit/kb32/internal/api"
	"github.com/cardiofit/kb32/internal/appropriateness"
	"github.com/cardiofit/kb32/internal/citations"
	kb32ctx "github.com/cardiofit/kb32/internal/context"
	"github.com/cardiofit/kb32/internal/framing"
	"github.com/cardiofit/kb32/internal/lifecycle"
	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/cardiofit/kb32/internal/store/postgres"
)

// ---------------------------------------------------------------------------
// Test-DB plumbing
// ---------------------------------------------------------------------------

// openE2ETestDB opens *sql.DB against VAIDSHALA_TEST_DSN or skips the test.
// The returned DB is closed via t.Cleanup so callers need not defer Close.
func openE2ETestDB(t *testing.T) *sql.DB {
	t.Helper()
	dsn := os.Getenv("VAIDSHALA_TEST_DSN")
	if dsn == "" {
		t.Skip("VAIDSHALA_TEST_DSN not set; skipping end-to-end integration test")
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

// snapshotSeed describes the kb-20 substrate rows seeded for a single
// resident. Only the fields populated by the seedClinicalSnapshot helper are
// listed here — extending this struct is the integration point for future
// substrate columns.
type snapshotSeed struct {
	cfs           int
	dbi           float64
	acb           int
	egfr          float64
	careIntensity string // kb-20 vocabulary (e.g. "active_treatment", "palliative")
	recentFall72h bool
}

// seedClinicalSnapshot inserts one row per relevant kb-20 table and registers
// a t.Cleanup that deletes every seeded row.
//
// NOTE: kb-20 lives in a separate service; we do not import a seed-helper
// package from it. Parameterised INSERTs against the kb-20 schema are the
// established pattern in this repo (see substrate_client_test.go's seed
// block) — no helper exists to delegate to today.
func seedClinicalSnapshot(t *testing.T, db *sql.DB, residentID uuid.UUID, seed snapshotSeed) {
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
		residentID.String(), seed.egfr, now)
	if seed.recentFall72h {
		mustExec(`INSERT INTO active_concerns
		            (resident_id, concern_type, started_at, expected_resolution_at, resolution_status)
		          VALUES ($1, 'post_fall_72h', $2, $3, 'open')`,
			residentID, now.Add(-1*time.Hour), now.Add(71*time.Hour))
	}

	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
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
	})
}

// seedSourceVersion inserts one active SourceVersion via PostgresRegistry and
// registers a t.Cleanup that removes it from source_versions.
func seedSourceVersion(t *testing.T, db *sql.DB, registry *citations.PostgresRegistry, sourceID string) citations.SourceVersion {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	sv := citations.SourceVersion{
		SourceID:      sourceID,
		Version:       "1",
		EffectiveFrom: time.Now().UTC().Add(-1 * time.Hour),
		EffectiveTo:   nil,
		ContentHash:   "e2e-seed-hash",
		Status:        citations.StatusActive,
	}
	if err := registry.Register(ctx, sv); err != nil {
		t.Fatalf("seed source_versions: %v", err)
	}
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx, `DELETE FROM source_versions WHERE source_id = $1`, sourceID)
		_, _ = db.ExecContext(ctx, `DELETE FROM recommendation_citations WHERE source_id = $1`, sourceID)
	})
	return sv
}

// ---------------------------------------------------------------------------
// Stub HAPI clients (local; we reuse the in-memory pattern from
// sunday_night_fall_test.go without importing test-private types from it).
// ---------------------------------------------------------------------------

type stubHAPIClientE2E struct {
	rules map[string]*reasoning.EvaluateRuleResult
}

func (c *stubHAPIClientE2E) EvaluateRule(_ context.Context, ruleID string, _ uuid.UUID) (*reasoning.EvaluateRuleResult, error) {
	if r, ok := c.rules[ruleID]; ok {
		res := *r
		res.RuleID = ruleID
		return &res, nil
	}
	return &reasoning.EvaluateRuleResult{RuleID: ruleID, Triggered: false}, nil
}

// ---------------------------------------------------------------------------
// Test 1 — Sunday-night-fall E2E with all-Postgres-backed deps.
// ---------------------------------------------------------------------------

// TestE2E_AllPostgresBackedDeps_SundayNightFall wires PostgresSubstrateClient
// + SubstrateBackedScorer + PostgresRegistry + CompositeEmitter(EthicsLog +
// Postgres) into a single Pipeline and exercises the canonical
// Sunday-night-fall fixture against a real database.
//
// Assertions:
//   - pipeline returns no error
//   - HoldReason is empty (gate passes)
//   - UrgencyTag == "red"
//   - Packet.Type == "MONITOR"
//   - EthicsLog InMemoryStore received exactly one EntryTypeDecision entry
//   - evidence_trace_entries has exactly one row for this RecommendationID
func TestE2E_AllPostgresBackedDeps_SundayNightFall(t *testing.T) {
	db := openE2ETestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	residentID := uuid.New()
	authorID := uuid.New()

	// Sunday-night-fall fixture per the brief: eGFR 55, DBI 0.3, ACB 1, CFS 5,
	// active treatment, RecentFall72h=true.
	seedClinicalSnapshot(t, db, residentID, snapshotSeed{
		cfs:           5,
		dbi:           0.3,
		acb:           1,
		egfr:          55.0,
		careIntensity: "active_treatment",
		recentFall72h: true,
	})

	// Wire components.
	substrateClient := postgres.NewPostgresSubstrateClient(db)
	assembler := kb32ctx.NewAssembler(substrateClient)

	hapi := &stubHAPIClientE2E{
		rules: map[string]*reasoning.EvaluateRuleResult{
			"PostFall": {Triggered: true, Type: "MONITOR", Urgency: "red"},
		},
	}
	chain := reasoning.NewChainBuilder(hapi)

	scorer := appropriateness.NewSubstrateBackedScorer()

	registry := citations.NewPostgresRegistry(db)
	_ = seedSourceVersion(t, db, registry, "E2E-SUNDAY-FALL-"+uuid.NewString())

	// Stage 7 — composite emitter over EthicsLog (in-memory) + Postgres.
	ethicsStore := ethics_log.NewInMemoryStore()
	ethicsEmitter := lifecycle.NewEthicsLogEmitter(ethics_log.NewLogger(ethicsStore))
	pgEmitter := lifecycle.NewPostgresEmitter(db)
	tracer := lifecycle.NewCompositeEmitter(ethicsEmitter, pgEmitter)

	pipeline := api.NewPipelineWithRegistry(assembler, chain, scorer, nil, registry).
		WithEvidenceTracer(tracer)

	result, err := pipeline.Run(ctx, "PostFall", residentID, authorID)
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}

	// --- Pipeline outcome -------------------------------------------------
	if result.HoldReason != "" {
		t.Errorf("expected gate to pass; got HoldReason=%q", result.HoldReason)
	}
	if result.UrgencyTag != "red" {
		t.Errorf("UrgencyTag = %q; want red", result.UrgencyTag)
	}
	if result.Packet == nil {
		t.Fatal("Packet is nil")
	}
	if result.Packet.Type != "MONITOR" {
		t.Errorf("Packet.Type = %q; want MONITOR", result.Packet.Type)
	}

	// --- Cleanup the Stage 7 row before we make assertions on it (we want
	// the cleanup registered now so a failed assertion still tears the row
	// down).
	recID := result.Packet.RecommendationID
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx,
			`DELETE FROM evidence_trace_entries WHERE recommendation_id = $1`, recID)
	})

	// --- EthicsLog assertion ---------------------------------------------
	entries, err := ethicsStore.List(ctx)
	if err != nil {
		t.Fatalf("ethicsStore.List: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("expected exactly 1 EthicsLog entry; got %d", len(entries))
	}
	if entries[0].EntryType != ethics_log.EntryTypeDecision {
		t.Errorf("EthicsLog entry type = %q; want %q",
			entries[0].EntryType, ethics_log.EntryTypeDecision)
	}

	// --- Postgres evidence_trace_entries assertion -----------------------
	var (
		ruleID  string
		firedAt time.Time
		count   int
	)
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM evidence_trace_entries WHERE recommendation_id = $1`,
		recID,
	).Scan(&count); err != nil {
		t.Fatalf("count evidence_trace_entries: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected exactly 1 evidence_trace_entries row; got %d", count)
	}
	if err := db.QueryRowContext(ctx,
		`SELECT rule_id, fired_at FROM evidence_trace_entries WHERE recommendation_id = $1`,
		recID,
	).Scan(&ruleID, &firedAt); err != nil {
		t.Fatalf("read evidence_trace_entries row: %v", err)
	}
	if ruleID != "PostFall" {
		t.Errorf("evidence_trace_entries.rule_id = %q; want PostFall", ruleID)
	}
	if firedAt.IsZero() {
		t.Error("evidence_trace_entries.fired_at is zero")
	}
}

// ---------------------------------------------------------------------------
// Test 2 — Appropriateness hold path emits no Stage 7 trace.
// ---------------------------------------------------------------------------

// TestE2E_AllPostgresBackedDeps_AppropriatenessHold proves the
// hold-path-no-emission invariant under real-Postgres conditions: when the
// SubstrateBackedScorer scores GoalsOfCareAlignment=1 (ADD recommendation on a
// palliative resident — the canonical anti-pattern) the pipeline holds and
// MUST NOT write a Stage 7 record either to the EthicsLog or to
// evidence_trace_entries.
func TestE2E_AllPostgresBackedDeps_AppropriatenessHold(t *testing.T) {
	db := openE2ETestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	residentID := uuid.New()
	authorID := uuid.New()

	// Palliative resident — combined with an ADD-type rule, this drives
	// GoalsOfCareAlignment=1 (see substrate_scorer.go scoreGoalsOfCareAlignment).
	seedClinicalSnapshot(t, db, residentID, snapshotSeed{
		cfs:           7,
		dbi:           0.5,
		acb:           2,
		egfr:          40.0,
		careIntensity: "palliative",
		recentFall72h: false,
	})

	substrateClient := postgres.NewPostgresSubstrateClient(db)
	assembler := kb32ctx.NewAssembler(substrateClient)

	// ADD on palliative => scoreGoalsOfCareAlignment returns 1 => Stage 4 holds.
	hapi := &stubHAPIClientE2E{
		rules: map[string]*reasoning.EvaluateRuleResult{
			"AddOnPalliative": {Triggered: true, Type: "ADD", Urgency: "amber"},
		},
	}
	chain := reasoning.NewChainBuilder(hapi)
	scorer := appropriateness.NewSubstrateBackedScorer()

	registry := citations.NewPostgresRegistry(db)

	ethicsStore := ethics_log.NewInMemoryStore()
	ethicsEmitter := lifecycle.NewEthicsLogEmitter(ethics_log.NewLogger(ethicsStore))
	pgEmitter := lifecycle.NewPostgresEmitter(db)
	tracer := lifecycle.NewCompositeEmitter(ethicsEmitter, pgEmitter)

	pipeline := api.NewPipelineWithRegistry(assembler, chain, scorer, nil, registry).
		WithEvidenceTracer(tracer)

	result, err := pipeline.Run(ctx, "AddOnPalliative", residentID, authorID)
	if err != nil {
		t.Fatalf("pipeline.Run: %v", err)
	}

	if result.HoldReason == "" {
		t.Fatalf("expected appropriateness hold; got empty HoldReason "+
			"(assessment=%+v)", result.Assessment)
	}
	if result.Assessment.GoalsOfCareAlignment > appropriateness.HoldThreshold {
		t.Errorf("expected GoalsOfCareAlignment ≤ HoldThreshold; got %d",
			result.Assessment.GoalsOfCareAlignment)
	}

	// --- EthicsLog must be empty (no Stage 7 emission on hold) -----------
	entries, err := ethicsStore.List(ctx)
	if err != nil {
		t.Fatalf("ethicsStore.List: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("expected zero EthicsLog entries on hold; got %d", len(entries))
	}

	// --- evidence_trace_entries must have zero rows for this recommendation
	// (use the packet's RecommendationID — even on hold the generator
	// produced a packet).
	if result.Packet == nil {
		t.Fatal("Packet unexpectedly nil on hold")
	}
	var count int
	if err := db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM evidence_trace_entries WHERE recommendation_id = $1`,
		result.Packet.RecommendationID,
	).Scan(&count); err != nil {
		t.Fatalf("count evidence_trace_entries: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 evidence_trace_entries rows on hold; got %d", count)
	}
}

// ---------------------------------------------------------------------------
// Test 3 — Postgres OptOutStore round-trip.
// ---------------------------------------------------------------------------

// TestE2E_OptOutStore_RoundTrip exercises Register → IsOptedOut → Revoke →
// IsOptedOut → re-Register → IsOptedOut against prescriber_framing_optout
// (migration 047). Validates idempotent re-register semantics under real
// Postgres.
func TestE2E_OptOutStore_RoundTrip(t *testing.T) {
	db := openE2ETestDB(t)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	store := framing.NewPostgresOptOutStore(db)
	gpID := uuid.New()
	t.Cleanup(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, _ = db.ExecContext(ctx,
			`DELETE FROM prescriber_framing_optout WHERE gp_id = $1`, gpID)
	})

	// Register → true.
	if err := store.RegisterOptOut(ctx, gpID, "e2e test"); err != nil {
		t.Fatalf("RegisterOptOut: %v", err)
	}
	if got, err := store.IsOptedOut(ctx, gpID); err != nil || !got {
		t.Fatalf("IsOptedOut after Register = (%v, %v); want (true, nil)", got, err)
	}

	// Revoke → false.
	if err := store.RevokeOptOut(ctx, gpID); err != nil {
		t.Fatalf("RevokeOptOut: %v", err)
	}
	if got, err := store.IsOptedOut(ctx, gpID); err != nil || got {
		t.Fatalf("IsOptedOut after Revoke = (%v, %v); want (false, nil)", got, err)
	}

	// Re-Register → true (idempotent flip-back).
	if err := store.RegisterOptOut(ctx, gpID, "e2e test refresh"); err != nil {
		t.Fatalf("RegisterOptOut (re-register): %v", err)
	}
	if got, err := store.IsOptedOut(ctx, gpID); err != nil || !got {
		t.Fatalf("IsOptedOut after re-Register = (%v, %v); want (true, nil)", got, err)
	}
}
