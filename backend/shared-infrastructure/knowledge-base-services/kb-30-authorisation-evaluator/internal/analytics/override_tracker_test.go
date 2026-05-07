package analytics

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newFixedTracker(now time.Time) *Tracker {
	return NewTracker().withClock(func() time.Time { return now })
}

func TestTracker_RecordAndStats(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	tr := newFixedTracker(now)
	for i := 0; i < 8; i++ {
		tr.Record("RULE_A", EventFire, now.Add(-time.Duration(i)*time.Hour))
	}
	for i := 0; i < 2; i++ {
		tr.Record("RULE_A", EventOverride, now.Add(-time.Duration(i)*time.Hour))
	}
	stats := tr.Stats(30, 0)
	require.Len(t, stats, 1)
	s := stats[0]
	assert.Equal(t, "RULE_A", s.RuleID)
	assert.Equal(t, 8, s.FireCount)
	assert.Equal(t, 2, s.OverrideCount)
	assert.InDelta(t, 0.20, s.OverrideRate, 0.001)
	assert.False(t, s.FlagRetire, "0.20 < 0.70 threshold; should not flag")
}

func TestTracker_FlagRetireAboveThreshold(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	tr := newFixedTracker(now)
	for i := 0; i < 2; i++ {
		tr.Record("RULE_NOISY", EventFire, now.Add(-time.Duration(i)*time.Hour))
	}
	for i := 0; i < 8; i++ {
		tr.Record("RULE_NOISY", EventOverride, now.Add(-time.Duration(i)*time.Hour))
	}
	stats := tr.Stats(30, 0)
	require.Len(t, stats, 1)
	assert.True(t, stats[0].FlagRetire,
		"override rate 0.80 should flag retirement")
}

func TestTracker_BelowMinFiresDoesNotFlag(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	tr := newFixedTracker(now)
	// Only 4 events total (< MinFiresForRetire=5).
	tr.Record("RULE_LOW", EventFire, now)
	for i := 0; i < 3; i++ {
		tr.Record("RULE_LOW", EventOverride, now.Add(-time.Duration(i)*time.Hour))
	}
	stats := tr.Stats(30, 0)
	require.Len(t, stats, 1)
	assert.False(t, stats[0].FlagRetire,
		"insufficient sample (<5 events) must not flag retirement even at 75% override rate")
}

func TestTracker_OutsideWindowExcluded(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	tr := newFixedTracker(now)
	tr.Record("RULE_OLD", EventFire, now.AddDate(0, 0, -45))
	tr.Record("RULE_OLD", EventOverride, now.AddDate(0, 0, -40))
	tr.Record("RULE_OLD", EventFire, now.AddDate(0, 0, -1))
	stats := tr.Stats(30, 0)
	require.Len(t, stats, 1)
	assert.Equal(t, 1, stats[0].FireCount)
	assert.Equal(t, 0, stats[0].OverrideCount)
}

func TestTracker_LibraryWideRate(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	tr := newFixedTracker(now)
	for i := 0; i < 95; i++ {
		tr.Record("RULE_A", EventFire, now.AddDate(0, 0, -i%10))
	}
	for i := 0; i < 5; i++ {
		tr.Record("RULE_A", EventOverride, now.AddDate(0, 0, -i))
	}
	rate := tr.LibraryWideOverrideRate(30)
	assert.InDelta(t, 0.05, rate, 0.001,
		"library-wide override rate at the Wave 6 < 5%% target")
	assert.LessOrEqual(t, rate, LibraryWideTargetMaxRate+0.001)
}

func TestTracker_RetirementCandidates(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	tr := newFixedTracker(now)
	// Quiet rule.
	for i := 0; i < 10; i++ {
		tr.Record("RULE_QUIET", EventFire, now.Add(-time.Duration(i)*time.Hour))
	}
	// Noisy rule.
	for i := 0; i < 2; i++ {
		tr.Record("RULE_NOISY", EventFire, now.Add(-time.Duration(i)*time.Hour))
	}
	for i := 0; i < 8; i++ {
		tr.Record("RULE_NOISY", EventOverride, now.Add(-time.Duration(i)*time.Hour))
	}
	candidates := tr.RetirementCandidates(30, 0)
	require.Len(t, candidates, 1)
	assert.Equal(t, "RULE_NOISY", candidates[0].RuleID)
}

func TestTracker_RecordRejectsZeroValues(t *testing.T) {
	tr := NewTracker()
	tr.Record("", EventFire, time.Now())
	tr.Record("RULE", EventFire, time.Time{})
	stats := tr.Stats(30, 0)
	assert.Empty(t, stats)
}
