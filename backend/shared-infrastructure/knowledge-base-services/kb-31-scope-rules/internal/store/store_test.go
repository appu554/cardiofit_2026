package store

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-scope-rules/internal/dsl"
)

func newSampleRule(ruleID, juri, category string, status dsl.Status, start time.Time) dsl.ScopeRule {
	r := dsl.ScopeRule{
		RuleID:          ruleID,
		Jurisdiction:    juri,
		Category:        category,
		Status:          status,
		EffectivePeriod: dsl.EffectivePeriod{StartDate: start},
		AppliesTo:       dsl.AppliesToScope{Role: "rn", ActionClass: dsl.ActionPrescribe},
		Evaluation:      dsl.EvaluationBlock{Decision: dsl.DecisionGranted, Reason: "ok"},
		Audit:           dsl.AuditBlock{LegislativeReference: "test"},
	}
	if status == dsl.StatusDraft {
		r.ActivationGate = "test gate"
	}
	return r
}

func TestMemoryStore_InsertAndGet(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	rule := newSampleRule("R1", "AU", "prescriber_scope", dsl.StatusActive,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	id, err := s.Insert(ctx, rule, []byte("yaml-payload"))
	require.NoError(t, err)
	got, err := s.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "R1", got.Rule.RuleID)
	assert.Equal(t, 1, got.Version)
}

func TestMemoryStore_VersionAutoIncrements(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	r := newSampleRule("R1", "AU", "prescriber_scope", dsl.StatusActive,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err := s.Insert(ctx, r, []byte("v1"))
	require.NoError(t, err)
	id2, err := s.Insert(ctx, r, []byte("v2"))
	require.NoError(t, err)
	v2, err := s.GetByID(ctx, id2)
	require.NoError(t, err)
	assert.Equal(t, 2, v2.Version)
}

func TestMemoryStore_ActiveForJurisdiction_HierarchyMatch(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	// AU national rule should apply to AU/VIC query.
	auRule := newSampleRule("R-AU", "AU", "prescriber_scope", dsl.StatusActive,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err := s.Insert(ctx, auRule, []byte("y"))
	require.NoError(t, err)

	// AU/VIC rule applies to AU/VIC query.
	vicRule := newSampleRule("R-VIC", "AU/VIC", "medication_administration_scope_restriction",
		dsl.StatusActive, time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	_, err = s.Insert(ctx, vicRule, []byte("y"))
	require.NoError(t, err)

	// AU/TAS rule should NOT apply to AU/VIC query.
	tasRule := newSampleRule("R-TAS", "AU/TAS", "prescriber_scope", dsl.StatusActive,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err = s.Insert(ctx, tasRule, []byte("y"))
	require.NoError(t, err)

	results, err := s.ActiveForJurisdiction(ctx, "AU/VIC", now)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, r := range results {
		got[r.Rule.RuleID] = true
	}
	assert.True(t, got["R-AU"], "AU national rule should apply to AU/VIC")
	assert.True(t, got["R-VIC"], "AU/VIC rule should apply to AU/VIC")
	assert.False(t, got["R-TAS"], "AU/TAS rule must not apply to AU/VIC")
}

func TestMemoryStore_DraftExcludedFromActive(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)

	draft := newSampleRule("R-DRAFT", "AU/TAS", "prescriber_scope", dsl.StatusDraft,
		time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err := s.Insert(ctx, draft, []byte("y"))
	require.NoError(t, err)

	results, err := s.ActiveForJurisdiction(ctx, "AU/TAS", now)
	require.NoError(t, err)
	for _, r := range results {
		assert.NotEqual(t, "R-DRAFT", r.Rule.RuleID, "DRAFT rule must be excluded from Active query")
	}

	all, err := s.AllForJurisdiction(ctx, "AU/TAS")
	require.NoError(t, err)
	gotDraft := false
	for _, r := range all {
		if r.Rule.RuleID == "R-DRAFT" {
			gotDraft = true
		}
	}
	assert.True(t, gotDraft, "DRAFT rule MUST appear in AllForJurisdiction (audit query)")
}

func TestMemoryStore_OutsideEffectivePeriodExcluded(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	beforeStart := time.Date(2025, 12, 1, 0, 0, 0, 0, time.UTC)
	r := newSampleRule("R1", "AU", "prescriber_scope", dsl.StatusActive,
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))
	_, err := s.Insert(ctx, r, []byte("y"))
	require.NoError(t, err)

	results, err := s.ActiveForJurisdiction(ctx, "AU", beforeStart)
	require.NoError(t, err)
	assert.Empty(t, results)
}

func TestMemoryStore_Lineage(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	r := newSampleRule("R1", "AU", "prescriber_scope", dsl.StatusActive,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	_, _ = s.Insert(ctx, r, []byte("v1"))
	_, _ = s.Insert(ctx, r, []byte("v2"))
	_, _ = s.Insert(ctx, r, []byte("v3"))
	lineage, err := s.Lineage(ctx, "R1")
	require.NoError(t, err)
	require.Len(t, lineage, 3)
	assert.Equal(t, 1, lineage[0].Version)
	assert.Equal(t, 3, lineage[2].Version)
}

// ---------------------------------------------------------------------------
// PostgresStore — DB-gated; skip cleanly without KB31_TEST_DATABASE_URL.
// ---------------------------------------------------------------------------

func newPostgresStore(t *testing.T) (*PostgresStore, func()) {
	t.Helper()
	dsnURL := os.Getenv("KB31_TEST_DATABASE_URL")
	if dsnURL == "" {
		t.Skip("KB31_TEST_DATABASE_URL not set; skipping Postgres tests")
	}
	db, err := sql.Open("postgres", dsnURL)
	require.NoError(t, err)
	return NewPostgresStore(db), func() { _ = db.Close() }
}

func TestPostgresStore_Insert(t *testing.T) {
	s, cleanup := newPostgresStore(t)
	defer cleanup()
	ctx := context.Background()
	rule := newSampleRule("R-PG-1", "AU", "prescriber_scope", dsl.StatusActive,
		time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	id, err := s.Insert(ctx, rule, []byte("yaml"))
	require.NoError(t, err)
	got, err := s.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "R-PG-1", got.Rule.RuleID)
}
