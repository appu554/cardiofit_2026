// Package integration runs end-to-end ScopeRule round-trip tests:
// load the bundled YAML files, insert into the in-memory store, query
// for the deployment jurisdictions, and assert the expected rules
// surface (or correctly stay hidden, in the DRAFT case).
package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-scope-rules/internal/dsl"
	"kb-scope-rules/internal/parser"
	"kb-scope-rules/internal/store"
)

// findDataDir walks up to find the kb-31 data/ directory.
func findDataDir(t *testing.T) string {
	t.Helper()
	dir, err := os.Getwd()
	require.NoError(t, err)
	for i := 0; i < 6; i++ {
		candidate := filepath.Join(dir, "data")
		if info, err := os.Stat(candidate); err == nil && info.IsDir() {
			return candidate
		}
		dir = filepath.Dir(dir)
	}
	t.Fatalf("could not find data/ directory")
	return ""
}

func loadAllIntoMemoryStore(t *testing.T) *store.MemoryStore {
	t.Helper()
	dataDir := findDataDir(t)
	loaded, errs := parser.LoadDir(dataDir)
	for _, e := range errs {
		t.Errorf("parse error: %v", e)
	}
	require.GreaterOrEqual(t, len(loaded), 4,
		"expected at least 4 bundled ScopeRules (Vic PCW, NMBA DRNP, ACOP APC, Tas pilot)")
	s := store.NewMemoryStore()
	for _, lr := range loaded {
		_, err := s.Insert(context.Background(), *lr.Rule, lr.PayloadYAML)
		require.NoError(t, err, "insert %s", lr.Path)
	}
	return s
}

// ---------------------------------------------------------------------------
// End-to-end: 4 deployment ScopeRules round-trip.
// ---------------------------------------------------------------------------

func TestRoundTrip_VictorianPCWVisibleAfterStartDate(t *testing.T) {
	s := loadAllIntoMemoryStore(t)
	// Query the day enforcement begins (29 Sep 2026, end of grace period).
	atTime := time.Date(2026, 9, 29, 0, 0, 0, 0, time.UTC)
	results, err := s.ActiveForJurisdiction(context.Background(), "AU/VIC", atTime)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, r := range results {
		got[r.Rule.RuleID] = true
	}
	assert.True(t, got["AUS-VIC-PCW-S4-EXCLUSION-2026-07-01"],
		"Victorian PCW exclusion ScopeRule must surface for AU/VIC on 29 Sep 2026")
}

func TestRoundTrip_VictorianPCWHiddenBeforeStartDate(t *testing.T) {
	s := loadAllIntoMemoryStore(t)
	// Day before commencement.
	atTime := time.Date(2026, 6, 30, 0, 0, 0, 0, time.UTC)
	results, err := s.ActiveForJurisdiction(context.Background(), "AU/VIC", atTime)
	require.NoError(t, err)
	for _, r := range results {
		assert.NotEqual(t, "AUS-VIC-PCW-S4-EXCLUSION-2026-07-01", r.Rule.RuleID,
			"Victorian PCW ScopeRule must NOT surface before 1 Jul 2026 commencement")
	}
}

func TestRoundTrip_NMBADRNPVisibleAtNationalQuery(t *testing.T) {
	s := loadAllIntoMemoryStore(t)
	atTime := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	results, err := s.ActiveForJurisdiction(context.Background(), "AU", atTime)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, r := range results {
		got[r.Rule.RuleID] = true
	}
	assert.True(t, got["AUS-NMBA-DRNP-PRESCRIBING-AGREEMENT-2025-09-30"],
		"NMBA DRNP ScopeRule must surface for AU on or after 30 Sep 2025")
}

func TestRoundTrip_NMBADRNPCascadesToVIC(t *testing.T) {
	// AU national rules cascade to AU/VIC queries.
	s := loadAllIntoMemoryStore(t)
	atTime := time.Date(2026, 5, 6, 0, 0, 0, 0, time.UTC)
	results, err := s.ActiveForJurisdiction(context.Background(), "AU/VIC", atTime)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, r := range results {
		got[r.Rule.RuleID] = true
	}
	assert.True(t, got["AUS-NMBA-DRNP-PRESCRIBING-AGREEMENT-2025-09-30"],
		"AU national NMBA DRNP must cascade to AU/VIC queries")
}

func TestRoundTrip_TasmanianPilotIsDraftAndExcluded(t *testing.T) {
	s := loadAllIntoMemoryStore(t)
	atTime := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	results, err := s.ActiveForJurisdiction(context.Background(), "AU/TAS", atTime)
	require.NoError(t, err)
	for _, r := range results {
		assert.NotEqual(t, "AUS-TAS-PHARMACIST-COPRESCRIBE-PILOT-2026", r.Rule.RuleID,
			"Tasmanian pilot is DRAFT and MUST NOT appear in ActiveForJurisdiction")
	}

	// AllForJurisdiction must include the DRAFT rule for audit/visibility.
	all, err := s.AllForJurisdiction(context.Background(), "AU/TAS")
	require.NoError(t, err)
	gotDraft := false
	var draftRule dsl.ScopeRule
	for _, r := range all {
		if r.Rule.RuleID == "AUS-TAS-PHARMACIST-COPRESCRIBE-PILOT-2026" {
			gotDraft = true
			draftRule = r.Rule
		}
	}
	require.True(t, gotDraft, "DRAFT Tasmanian pilot must be visible to audit/visibility queries")
	assert.Equal(t, dsl.StatusDraft, draftRule.Status)
	assert.NotEmpty(t, draftRule.ActivationGate, "DRAFT pilot must document activation_gate")
}

func TestRoundTrip_ACOPAPCVisibleAfterStartDate(t *testing.T) {
	s := loadAllIntoMemoryStore(t)
	atTime := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	results, err := s.ActiveForJurisdiction(context.Background(), "AU", atTime)
	require.NoError(t, err)
	got := map[string]bool{}
	for _, r := range results {
		got[r.Rule.RuleID] = true
	}
	assert.True(t, got["AUS-ACOP-APC-CREDENTIAL-2026-07-01"],
		"ACOP APC credential ScopeRule must surface for AU on 1 Jul 2026")
}

func TestRoundTrip_LegislativeCitationsArePresent(t *testing.T) {
	s := loadAllIntoMemoryStore(t)
	atTime := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	all, err := s.AllForJurisdiction(context.Background(), "AU")
	require.NoError(t, err)
	allVic, err := s.AllForJurisdiction(context.Background(), "AU/VIC")
	require.NoError(t, err)
	allTas, err := s.AllForJurisdiction(context.Background(), "AU/TAS")
	require.NoError(t, err)

	// Every bundled rule must carry a non-empty legislative_reference and
	// source_url. This is the regulator-defensible audit chain test.
	for _, set := range [][]store.StoredRule{all, allVic, allTas} {
		for _, r := range set {
			assert.NotEmpty(t, r.Rule.Audit.LegislativeReference,
				"%s must carry legislative_reference", r.Rule.RuleID)
			assert.NotEmpty(t, r.Rule.Audit.SourceURL,
				"%s must carry source_url for the published reference", r.Rule.RuleID)
		}
	}
	_ = atTime
}
