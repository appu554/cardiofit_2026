// Package metric_integrity_test validates the override-pattern suppression
// detector's ability to emit an EthicsLog pattern_detected entry when the
// inappropriate-override ratio exceeds the configured floor.
//
// Recommendation Craft Guidelines Part 13 — metric integrity test category.
// VisibilityClass: AD — override audit per Guidelines §5
package metric_integrity_test

import (
	"context"
	"testing"
	"time"

	"github.com/cardiofit/kb32/internal/overrides"
	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// ---------------------------------------------------------------------------
// mockStore — a Store implementation whose PatternSummary returns fixed counts.
//
// This avoids the Postgres+materialised-view dependency that PatternSummary
// requires in production (InMemoryStore.PatternSummary requires CreateForRule
// to associate records with a ruleID, but the map lookup from recommendation
// to rule requires Postgres in the full implementation).
// ---------------------------------------------------------------------------

type mockStore struct {
	summary map[string]int
}

func (m *mockStore) Create(_ context.Context, r overrides.OverrideReason) (overrides.OverrideReason, error) {
	return r, nil
}

func (m *mockStore) Get(_ context.Context, _ string) (overrides.OverrideReason, error) {
	return overrides.OverrideReason{}, nil
}

func (m *mockStore) ListByRule(_ context.Context, _ string) ([]overrides.OverrideReason, error) {
	return nil, nil
}

func (m *mockStore) PatternSummary(_ context.Context, _ string, _ time.Time) (map[string]int, error) {
	return m.summary, nil
}

var _ overrides.Store = (*mockStore)(nil)

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestSuppressionDetector_FlagsHighOverrideInappropriateRate asserts that when
// the mock store returns:
//   - 35 inappropriate_override records
//   - 15 appropriate_override records
//   - Total: 50 (≥ OverrideThreshold=30)
//   - Ratio: 70% (≥ InappropriateRatioFloor=0.6)
//
// the Detector emits exactly one EthicsLog entry with EntryType=pattern_detected.
//
// Guidelines §13 metric integrity cap: the suppression signal must fire reliably
// when inappropriate-override rate meets or exceeds the configured floor.
func TestSuppressionDetector_FlagsHighOverrideInappropriateRate(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		summary: map[string]int{
			"inappropriate_override": 35, // 70% of 50 — above InappropriateRatioFloor (0.6)
			"appropriate_override":   15,
		},
	}

	logStore := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(logStore)
	detector := overrides.NewDetector(store, logger)

	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	if err := detector.Scan(ctx, "TestRule_HighRate", since); err != nil {
		t.Fatalf("Scan returned unexpected error: %v", err)
	}

	entries, err := logStore.List(ctx)
	if err != nil {
		t.Fatalf("List ethics log entries: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("expected 1 ethics log entry; got %d", len(entries))
	}

	got := entries[0]
	if got.EntryType != ethics_log.EntryTypePatternDetected {
		t.Errorf("expected EntryType=%q; got %q", ethics_log.EntryTypePatternDetected, got.EntryType)
	}
	if got.Severity != 3 {
		t.Errorf("expected Severity=3; got %d", got.Severity)
	}
	if got.Status != ethics_log.StatusOpen {
		t.Errorf("expected Status=%q; got %q", ethics_log.StatusOpen, got.Status)
	}
}

// TestSuppressionDetector_DoesNotFlagBelowThreshold asserts that a rule with
// only 29 total overrides (below OverrideThreshold=30) does NOT trigger an
// EthicsLog entry even if the inappropriate ratio would otherwise qualify.
//
// Low-volume windows are deliberately ignored to prevent noisy signals from
// rules with insufficient data.
func TestSuppressionDetector_DoesNotFlagBelowThreshold(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		summary: map[string]int{
			"inappropriate_override": 25, // 86% but total is 29 — below OverrideThreshold
			"appropriate_override":   4,
		},
	}

	logStore := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(logStore)
	detector := overrides.NewDetector(store, logger)

	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	if err := detector.Scan(ctx, "TestRule_LowVolume", since); err != nil {
		t.Fatalf("Scan returned unexpected error: %v", err)
	}

	entries, err := logStore.List(ctx)
	if err != nil {
		t.Fatalf("List ethics log entries: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 ethics log entries for low-volume window; got %d", len(entries))
	}
}

// TestSuppressionDetector_DoesNotFlagBelowRatioFloor asserts that a rule with
// enough total overrides but an inappropriate ratio below the floor
// (InappropriateRatioFloor=0.6) does NOT trigger an EthicsLog entry.
func TestSuppressionDetector_DoesNotFlagBelowRatioFloor(t *testing.T) {
	t.Parallel()

	store := &mockStore{
		summary: map[string]int{
			"inappropriate_override": 17, // 34% of 50 — below InappropriateRatioFloor
			"appropriate_override":   33,
		},
	}

	logStore := ethics_log.NewInMemoryStore()
	logger := ethics_log.NewLogger(logStore)
	detector := overrides.NewDetector(store, logger)

	ctx := context.Background()
	since := time.Now().Add(-24 * time.Hour)

	if err := detector.Scan(ctx, "TestRule_LowRatio", since); err != nil {
		t.Fatalf("Scan returned unexpected error: %v", err)
	}

	entries, err := logStore.List(ctx)
	if err != nil {
		t.Fatalf("List ethics log entries: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("expected 0 ethics log entries for below-floor ratio; got %d", len(entries))
	}
}
