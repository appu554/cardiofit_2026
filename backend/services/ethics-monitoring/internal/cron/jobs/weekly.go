package jobs

import (
	"context"
	"fmt"

	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// WeeklyContentVariationJob runs at 03:00 every Monday. Phase 1c does not yet
// ship a content-variation detector primitive (corpus diversity scoring is
// scheduled for Task 2 of the tightened Phase 3 plan). Until that primitive
// lands this job emits a low-severity scheduled-audit entry so operators can
// see the cadence is firing and so the EBA register reflects the cadence
// commitment.
type WeeklyContentVariationJob struct {
	Logger *ethics_log.Logger
}

// Name implements cron.Job.
func (WeeklyContentVariationJob) Name() string { return "weekly_content_variation" }

// Schedule implements cron.Job — 03:00 every Monday.
func (WeeklyContentVariationJob) Schedule() string { return "0 3 * * 1" }

// Run logs a scheduled-audit placeholder entry. NOTE: ethics_log.EntryType has
// no canonical "audit_scheduled" value; we use EntryTypeReviewRequested with a
// descriptive message and severity 1 so the cadence is visible without
// false-positive ringing.
func (j WeeklyContentVariationJob) Run(ctx context.Context) error {
	entry := ethics_log.Entry{
		EntryType:   ethics_log.EntryTypeReviewRequested,
		Severity:    1,
		Status:      ethics_log.StatusOpen,
		Description: "weekly content-variation scan scheduled — primitive deferred to Phase 3 Task 2 (corpus diversity)",
	}
	if err := j.Logger.Append(ctx, entry); err != nil {
		return fmt.Errorf("log emit: %w", err)
	}
	return nil
}
