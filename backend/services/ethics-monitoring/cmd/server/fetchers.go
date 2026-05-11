package main

import (
	"context"

	"github.com/cardiofit/shared/v2_substrate/ethics/pattern_detection"
)

// emptyPatternFetcher is the Phase 3 Task 1 placeholder — it returns no
// snapshot pairs so the daily acceptance/appropriateness job runs to
// completion as a no-op until the Postgres-backed fetcher lands in a
// follow-up task.
type emptyPatternFetcher struct{}

func (emptyPatternFetcher) LatestRuleSnapshots(_ context.Context) (
	[]pattern_detection.RuleSnapshot, []pattern_detection.RuleSnapshot, error,
) {
	return nil, nil, nil
}

// emptySuppressionFetcher is the Phase 3 Task 1 placeholder — it returns no
// inputs so the daily suppression scan job runs as a no-op until a
// Postgres-backed fetcher lands.
type emptySuppressionFetcher struct{}

func (emptySuppressionFetcher) SuppressionInputs(_ context.Context) (
	[]pattern_detection.SuppressionInputs, error,
) {
	return nil, nil
}
