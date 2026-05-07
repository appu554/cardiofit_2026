package store

import (
	"context"
	"database/sql"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-authorisation-evaluator/internal/dsl"
)

func newSampleRule(ruleID, juri string, start time.Time) dsl.AuthorisationRule {
	return dsl.AuthorisationRule{
		RuleID:          ruleID,
		Jurisdiction:    juri,
		EffectivePeriod: dsl.EffectivePeriod{StartDate: start},
		AppliesTo:       dsl.AppliesToScope{Role: "rn", ActionClass: dsl.ActionPrescribe},
		Evaluation:      dsl.EvaluationBlock{Decision: dsl.DecisionGranted, Reason: "ok"},
		Audit:           dsl.AuditBlock{LegislativeReference: "test legislation"},
	}
}

func TestMemoryStore_InsertAndGet(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	rule := newSampleRule("R1", "AU", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	id, err := s.Insert(ctx, rule, []byte("yaml-blob-1"))
	require.NoError(t, err)

	got, err := s.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "R1", got.Rule.RuleID)
	assert.Equal(t, 1, got.Version)
	assert.NotEmpty(t, got.ContentSHA)
}

func TestMemoryStore_VersionAutoIncrement(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	r := newSampleRule("R1", "AU", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	for i := 1; i <= 3; i++ {
		_, err := s.Insert(ctx, r, []byte("v"))
		require.NoError(t, err)
	}
	lineage, err := s.Lineage(ctx, "R1")
	require.NoError(t, err)
	require.Len(t, lineage, 3)
	assert.Equal(t, 1, lineage[0].Version)
	assert.Equal(t, 3, lineage[2].Version)
}

func TestMemoryStore_ActiveForJurisdiction_TimeFiltering(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()

	earlyStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	earlyEnd := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	expired := newSampleRule("EXPIRED", "AU", earlyStart)
	expired.EffectivePeriod.EndDate = &earlyEnd
	_, err := s.Insert(ctx, expired, []byte("y"))
	require.NoError(t, err)

	current := newSampleRule("CURRENT", "AU", earlyStart)
	_, err = s.Insert(ctx, current, []byte("y"))
	require.NoError(t, err)

	future := newSampleRule("FUTURE", "AU", time.Date(2030, 1, 1, 0, 0, 0, 0, time.UTC))
	_, err = s.Insert(ctx, future, []byte("y"))
	require.NoError(t, err)

	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	active, err := s.ActiveForJurisdiction(ctx, "AU", now)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "CURRENT", active[0].Rule.RuleID)
}

func TestMemoryStore_ActiveForJurisdiction_PrefixMatch(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)

	auRule := newSampleRule("AU-WIDE", "AU", start)
	_, err := s.Insert(ctx, auRule, []byte("y"))
	require.NoError(t, err)

	vicRule := newSampleRule("VIC-ONLY", "AU/VIC", start)
	_, err = s.Insert(ctx, vicRule, []byte("y"))
	require.NoError(t, err)

	tasRule := newSampleRule("TAS-ONLY", "AU/TAS", start)
	_, err = s.Insert(ctx, tasRule, []byte("y"))
	require.NoError(t, err)

	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	vicActive, err := s.ActiveForJurisdiction(ctx, "AU/VIC", now)
	require.NoError(t, err)
	ids := ruleIDs(vicActive)
	assert.Contains(t, ids, "AU-WIDE")
	assert.Contains(t, ids, "VIC-ONLY")
	assert.NotContains(t, ids, "TAS-ONLY")
}

func TestMemoryStore_VersionPrecedence_LatestVersionWins(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	r := newSampleRule("R1", "AU", start)
	r.Evaluation.Reason = "v1"
	_, err := s.Insert(ctx, r, []byte("y"))
	require.NoError(t, err)
	r.Evaluation.Reason = "v2"
	_, err = s.Insert(ctx, r, []byte("y"))
	require.NoError(t, err)

	now := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	active, err := s.ActiveForJurisdiction(ctx, "AU", now)
	require.NoError(t, err)
	require.Len(t, active, 1)
	assert.Equal(t, "v2", active[0].Rule.Evaluation.Reason)
	assert.Equal(t, 2, active[0].Version)
}

func TestMemoryStore_RegisterSupersession(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	start := time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC)
	r := newSampleRule("R1", "AU", start)
	oldID, err := s.Insert(ctx, r, []byte("old"))
	require.NoError(t, err)
	r.Evaluation.Reason = "updated"
	newID, err := s.Insert(ctx, r, []byte("new"))
	require.NoError(t, err)

	require.NoError(t, s.RegisterSupersession(ctx, oldID, newID))
	got, err := s.GetByID(ctx, newID)
	require.NoError(t, err)
	require.NotNil(t, got.SupersedesRef)
	assert.Equal(t, oldID, *got.SupersedesRef)
}

func TestMemoryStore_RejectsInvalidRule(t *testing.T) {
	s := NewMemoryStore()
	ctx := context.Background()
	_, err := s.Insert(ctx, dsl.AuthorisationRule{}, []byte(""))
	require.Error(t, err)
}

func TestJurisdictionMatch(t *testing.T) {
	cases := []struct {
		ruleJuri, queryJuri string
		want                bool
	}{
		{"AU", "AU", true},
		{"AU", "AU/VIC", true},
		{"AU/VIC", "AU/VIC", true},
		{"AU/VIC", "AU/TAS", false},
		{"AU/VIC", "AU", false},
		{"AU", "AUS", false}, // no false-prefix match
	}
	for _, c := range cases {
		assert.Equal(t, c.want, jurisdictionMatch(c.ruleJuri, c.queryJuri),
			"rule=%s query=%s", c.ruleJuri, c.queryJuri)
	}
}

// ----- Postgres-gated tests --------------------------------------------------

func TestPostgresStore_DBGated(t *testing.T) {
	dsn := os.Getenv("KB30_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("KB30_TEST_DATABASE_URL not set; skipping Postgres store tests")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	defer db.Close()

	require.NoError(t, db.Ping())
	// Apply schema.
	migrationSQL, err := os.ReadFile("../../migrations/001_authorisation_rules.sql")
	require.NoError(t, err)
	_, err = db.Exec(string(migrationSQL))
	require.NoError(t, err)
	_, err = db.Exec(`TRUNCATE authorisation_rules`)
	require.NoError(t, err)

	ps := NewPostgresStore(db)
	ctx := context.Background()
	r := newSampleRule("PG-1", "AU", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC))
	id, err := ps.Insert(ctx, r, []byte("yaml-payload"))
	require.NoError(t, err)
	got, err := ps.GetByID(ctx, id)
	require.NoError(t, err)
	assert.Equal(t, "PG-1", got.Rule.RuleID)
	assert.Equal(t, 1, got.Version)

	// Insert v2 + verify lineage.
	_, err = ps.Insert(ctx, r, []byte("yaml-payload-v2"))
	require.NoError(t, err)
	lineage, err := ps.Lineage(ctx, "PG-1")
	require.NoError(t, err)
	assert.Len(t, lineage, 2)
}

func ruleIDs(rs []StoredRule) []string {
	out := make([]string, len(rs))
	for i, r := range rs {
		out[i] = r.Rule.RuleID
	}
	return out
}
