package overrides_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/cardiofit/kb32/internal/overrides"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// ---------------------------------------------------------------------------
// fakeLogger — injectable EthicsLogger for tests
// ---------------------------------------------------------------------------

type fakeLogger struct {
	entries []ethics_log.Entry
	err     error
}

func (f *fakeLogger) Append(_ context.Context, e ethics_log.Entry) error {
	if f.err != nil {
		return f.err
	}
	f.entries = append(f.entries, e)
	return nil
}

// ---------------------------------------------------------------------------
// Helper: build an InMemoryStore pre-loaded with overrides for a rule.
//
// appropriate + inappropriate + mixed must sum to total.
// All overrides are captured "now" (within the since window).
// ---------------------------------------------------------------------------

func buildStore(t *testing.T, ruleID string, appropriate, inappropriate, mixed int) *overrides.InMemoryStore {
	t.Helper()
	s := overrides.NewInMemoryStore()
	ctx := context.Background()

	load := func(flag string, n int) {
		for i := 0; i < n; i++ {
			_, err := s.CreateForRule(ctx, overrides.OverrideReason{
				RecommendationID:    "rec-test",
				ReasonCode:          "alert_fatigue",
				AppropriatenessFlag: flag,
				Reasoning:           "test",
				CapturedBy:          "user",
			}, ruleID)
			if err != nil {
				t.Fatalf("CreateForRule %s: %v", flag, err)
			}
		}
	}

	load("appropriate_override", appropriate)
	load("inappropriate_override", inappropriate)
	load("mixed", mixed)
	return s
}

// since is far enough in the past to include all test records.
var testSince = time.Now().Add(-7 * 24 * time.Hour)

// ---------------------------------------------------------------------------
// Test: 50 total, 35 inappropriate → ratio 0.70 ≥ 0.60 → log entry emitted
// ---------------------------------------------------------------------------

func TestDetector_Scan_AboveThresholdAndFloor_EmitsEntry(t *testing.T) {
	// 50 overrides, 35 inappropriate → ratio = 0.70
	store := buildStore(t, "rule-A", 15, 35, 0)
	logger := &fakeLogger{}
	d := overrides.NewDetector(store, logger)

	if err := d.Scan(context.Background(), "rule-A", testSince); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(logger.entries) != 1 {
		t.Fatalf("expected 1 ethics log entry; got %d", len(logger.entries))
	}
	e := logger.entries[0]
	if e.EntryType != ethics_log.EntryTypePatternDetected {
		t.Errorf("EntryType = %q; want %q", e.EntryType, ethics_log.EntryTypePatternDetected)
	}
	if e.Severity != 3 {
		t.Errorf("Severity = %d; want 3", e.Severity)
	}
	if e.Description == "" {
		t.Error("Description must not be empty")
	}
	// Description should mention the rule ID and counts.
	if !containsAll(e.Description, "rule-A", "50", "35") {
		t.Errorf("Description %q missing expected tokens (rule-A / 50 / 35)", e.Description)
	}
}

// ---------------------------------------------------------------------------
// Test: 50 total, 20 inappropriate → ratio 0.40 < 0.60 → no log entry
// ---------------------------------------------------------------------------

func TestDetector_Scan_BelowFloor_NoEntry(t *testing.T) {
	// 50 overrides, 20 inappropriate → ratio = 0.40
	store := buildStore(t, "rule-B", 30, 20, 0)
	logger := &fakeLogger{}
	d := overrides.NewDetector(store, logger)

	if err := d.Scan(context.Background(), "rule-B", testSince); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(logger.entries) != 0 {
		t.Errorf("expected 0 entries (ratio below floor); got %d", len(logger.entries))
	}
}

// ---------------------------------------------------------------------------
// Test: 20 total (below threshold) → no log entry regardless of ratio
// ---------------------------------------------------------------------------

func TestDetector_Scan_BelowThreshold_NoEntry(t *testing.T) {
	// 20 overrides, 18 inappropriate → ratio = 0.90, but total < 30
	store := buildStore(t, "rule-C", 2, 18, 0)
	logger := &fakeLogger{}
	d := overrides.NewDetector(store, logger)

	if err := d.Scan(context.Background(), "rule-C", testSince); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(logger.entries) != 0 {
		t.Errorf("expected 0 entries (below threshold); got %d", len(logger.entries))
	}
}

// ---------------------------------------------------------------------------
// Test: exactly 30 total, 18 inappropriate → ratio 0.60 exactly → entry emitted
// ---------------------------------------------------------------------------

func TestDetector_Scan_ExactlyAtThresholdAndFloor_EmitsEntry(t *testing.T) {
	// 30 overrides, 18 inappropriate → ratio = 0.60 exactly
	store := buildStore(t, "rule-D", 12, 18, 0)
	logger := &fakeLogger{}
	d := overrides.NewDetector(store, logger)

	if err := d.Scan(context.Background(), "rule-D", testSince); err != nil {
		t.Fatalf("Scan: %v", err)
	}
	if len(logger.entries) != 1 {
		t.Fatalf("expected 1 entry at exact threshold+floor; got %d", len(logger.entries))
	}
}

// ---------------------------------------------------------------------------
// Test: store error propagates
// ---------------------------------------------------------------------------

func TestDetector_Scan_StoreError_Propagates(t *testing.T) {
	storeErr := errors.New("store: connection refused")
	store := &errorStore{err: storeErr}
	logger := &fakeLogger{}
	d := overrides.NewDetector(store, logger)

	err := d.Scan(context.Background(), "rule-E", testSince)
	if err == nil {
		t.Fatal("expected error from store; got nil")
	}
	if !errors.Is(err, storeErr) {
		t.Errorf("error = %v; want wrapping %v", err, storeErr)
	}
}

// ---------------------------------------------------------------------------
// Test: logger error propagates
// ---------------------------------------------------------------------------

func TestDetector_Scan_LoggerError_Propagates(t *testing.T) {
	// Trigger a log entry (50 total, 35 inappropriate).
	store := buildStore(t, "rule-F", 15, 35, 0)
	logErr := errors.New("logger: write failed")
	logger := &fakeLogger{err: logErr}
	d := overrides.NewDetector(store, logger)

	err := d.Scan(context.Background(), "rule-F", testSince)
	if err == nil {
		t.Fatal("expected error from logger; got nil")
	}
	if !errors.Is(err, logErr) {
		t.Errorf("error = %v; want wrapping %v", err, logErr)
	}
}

// ---------------------------------------------------------------------------
// errorStore — Store that always fails PatternSummary
// ---------------------------------------------------------------------------

type errorStore struct {
	err error
}

func (e *errorStore) Create(_ context.Context, r overrides.OverrideReason) (overrides.OverrideReason, error) {
	return overrides.OverrideReason{}, e.err
}
func (e *errorStore) Get(_ context.Context, _ string) (overrides.OverrideReason, error) {
	return overrides.OverrideReason{}, e.err
}
func (e *errorStore) ListByRule(_ context.Context, _ string) ([]overrides.OverrideReason, error) {
	return nil, e.err
}
func (e *errorStore) PatternSummary(_ context.Context, _ string, _ time.Time) (map[string]int, error) {
	return nil, e.err
}

// ---------------------------------------------------------------------------
// containsAll checks that s contains all of the given substrings.
// ---------------------------------------------------------------------------

func containsAll(s string, subs ...string) bool {
	for _, sub := range subs {
		found := false
		for i := 0; i <= len(s)-len(sub); i++ {
			if s[i:i+len(sub)] == sub {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
