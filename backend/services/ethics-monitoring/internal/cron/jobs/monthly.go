package jobs

import (
	"context"
	"fmt"

	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// MonthlyBiasDisparityJob runs at 04:00 on the 1st of each month. The
// underlying disparity detector (pattern_detection.DetectBiasDisparity) is
// already shipped, but the *stratified inputs* — age band, sex, frailty
// tier, CALD background, socioeconomic indicator, facility type, geography —
// are gated on Phase 3 Task 2's stratification pipeline. Until that ships
// this job records the cadence commitment as a low-severity entry.
type MonthlyBiasDisparityJob struct {
	Logger *ethics_log.Logger
}

// Name implements cron.Job.
func (MonthlyBiasDisparityJob) Name() string { return "monthly_bias_disparity" }

// Schedule implements cron.Job — 04:00 on day 1 of every month.
func (MonthlyBiasDisparityJob) Schedule() string { return "0 4 1 * *" }

// Run logs a scheduled-audit placeholder. NOTE: ethics_log.EntryType has no
// canonical "audit_scheduled" value; we use EntryTypeReviewRequested with a
// descriptive message and severity 1 to flag the cadence without raising a
// false alarm.
func (j MonthlyBiasDisparityJob) Run() error {
	ctx := context.Background()
	entry := ethics_log.Entry{
		EntryType:   ethics_log.EntryTypeReviewRequested,
		Severity:    1,
		Status:      ethics_log.StatusOpen,
		Description: "monthly bias-disparity audit scheduled — gated on Phase 3 Task 2 stratification pipeline (age/sex/frailty/CALD/SEIFA/facility/geography)",
	}
	if err := j.Logger.Append(ctx, entry); err != nil {
		return fmt.Errorf("log emit: %w", err)
	}
	return nil
}
