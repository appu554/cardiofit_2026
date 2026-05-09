// Package jobs contains the concrete cron jobs run by the ethics-monitoring
// orchestrator. Each job composes a Phase 1c pure-function detector from
// shared/v2_substrate/ethics/pattern_detection with the ethics_log substrate.
//
// Schedules follow Ethical Architecture Guidelines §10.1:
//   - Daily detection (acceptance-appropriateness divergence, suppression scan)
//   - Weekly detection (content variation — placeholder until corpus indexer ships)
//   - Monthly detection (bias disparity — gated on Task 2 stratification pipeline)
//
// Jobs intentionally hold no persistent state; collaborators are injected via
// constructors so unit tests can use in-memory fakes.
//
// VisibilityClass: AD
package jobs

import (
	"context"
	"errors"
	"fmt"

	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
	"github.com/cardiofit/shared/v2_substrate/ethics/pattern_detection"
)

// PatternFetcher supplies the rolling rule snapshot pairs (prior, current)
// that the daily acceptance-appropriateness detector consumes. Implementations
// are expected to query the recommendations / appropriateness substrate over
// fixed rolling windows (typically 30-day prior vs. current). The fetcher is
// the seam between this scheduling layer and Phase 2's substrate-backed
// telemetry pipeline.
type PatternFetcher interface {
	// LatestRuleSnapshots returns matched (prior, current) snapshot pairs for
	// every rule that has data in both windows. The slices MUST be the same
	// length; element i in prior corresponds to element i in current.
	LatestRuleSnapshots(ctx context.Context) (prior, current []pattern_detection.RuleSnapshot, err error)
}

// SuppressionFetcher supplies per-rule deferral statistics for the daily
// suppression scan.
type SuppressionFetcher interface {
	// SuppressionInputs returns one inputs record per active rule for the
	// rolling observation window.
	SuppressionInputs(ctx context.Context) ([]pattern_detection.SuppressionInputs, error)
}

// DailyAcceptanceAppropriatenessJob runs at 02:00 daily and emits an ethics
// log entry for every rule whose acceptance has risen ≥ 10 percentage points
// without a parallel rise in appropriateness mean (Guidelines §1 Principle 2,
// §10 daily detection).
type DailyAcceptanceAppropriatenessJob struct {
	Fetcher PatternFetcher
	Logger  *ethics_log.Logger
	// Threshold is the acceptance-rise percentage-point trigger. Defaults to
	// 0.10 (10 pp) per Guidelines §10.
	Threshold float64
}

// Name implements cron.Job.
func (DailyAcceptanceAppropriatenessJob) Name() string {
	return "daily_acceptance_appropriateness"
}

// Schedule implements cron.Job — 02:00 every day.
func (DailyAcceptanceAppropriatenessJob) Schedule() string { return "0 2 * * *" }

// Run executes the detector. It is safe to call directly from tests.
//
// Errors from individual Logger.Append calls are accumulated via errors.Join
// rather than aborting on the first failure, so a transient log-store error
// for one rule does not silently drop subsequent divergent rules in the same
// batch. The fetch error is still terminal (without inputs there is no work).
func (j DailyAcceptanceAppropriatenessJob) Run(ctx context.Context) error {
	threshold := j.Threshold
	if threshold == 0 {
		threshold = 0.10
	}
	prior, current, err := j.Fetcher.LatestRuleSnapshots(ctx)
	if err != nil {
		return fmt.Errorf("fetch snapshots: %w", err)
	}
	if len(prior) != len(current) {
		return fmt.Errorf("snapshot length mismatch: prior=%d current=%d", len(prior), len(current))
	}
	var errs []error
	for i := range prior {
		if !pattern_detection.DetectDivergence(prior[i], current[i], threshold) {
			continue
		}
		entry := ethics_log.Entry{
			EntryType: ethics_log.EntryTypePatternDetected,
			Severity:  3,
			Status:    ethics_log.StatusOpen,
			Description: fmt.Sprintf(
				"acceptance-appropriateness divergence detected for rule %q: "+
					"acceptance rose ≥ %.0f pp without parallel appropriateness rise",
				current[i].RuleID, threshold*100,
			),
		}
		if err := j.Logger.Append(ctx, entry); err != nil {
			errs = append(errs, fmt.Errorf("log emit %s: %w", current[i].RuleID, err))
		}
	}
	return errors.Join(errs...)
}

// DailySuppressionScanJob runs at 02:15 daily and emits a pattern_detected
// entry for every rule whose recommendations are being deferred at high rate
// without documented clinical reasoning (Guidelines §10).
type DailySuppressionScanJob struct {
	Fetcher SuppressionFetcher
	Logger  *ethics_log.Logger
	// DeferralThreshold is the minimum deferral rate that triggers
	// suspicion. Defaults to 0.40 (40 %) — a conservative starting point;
	// tuneable once shadow-deployment telemetry is available.
	DeferralThreshold float64
	// UndocumentedThreshold per the substrate semantics: flag fires when
	// undocumented rate ≥ (1 - threshold). Defaults to 0.20 (i.e. ≥ 80 %
	// undocumented triggers).
	UndocumentedThreshold float64
}

// Name implements cron.Job.
func (DailySuppressionScanJob) Name() string { return "daily_suppression_scan" }

// Schedule implements cron.Job — 02:15 every day, offset from the
// acceptance/appropriateness job to spread DB load.
func (DailySuppressionScanJob) Schedule() string { return "15 2 * * *" }

// Run executes the suppression detector. As with the divergence job, per-rule
// log-emit failures are accumulated via errors.Join rather than aborting.
func (j DailySuppressionScanJob) Run(ctx context.Context) error {
	deferral := j.DeferralThreshold
	if deferral == 0 {
		deferral = 0.40
	}
	undoc := j.UndocumentedThreshold
	if undoc == 0 {
		undoc = 0.20
	}
	inputs, err := j.Fetcher.SuppressionInputs(ctx)
	if err != nil {
		return fmt.Errorf("fetch suppression inputs: %w", err)
	}
	var errs []error
	for _, in := range inputs {
		if !pattern_detection.DetectSuppression(in, deferral, undoc) {
			continue
		}
		entry := ethics_log.Entry{
			EntryType: ethics_log.EntryTypePatternDetected,
			Severity:  3,
			Status:    ethics_log.StatusOpen,
			Description: fmt.Sprintf(
				"systematic suppression detected for rule %q: deferral rate ≥ %.0f%% "+
					"with ≥ %.0f%% undocumented",
				in.RuleID, deferral*100, (1-undoc)*100,
			),
		}
		if err := j.Logger.Append(ctx, entry); err != nil {
			errs = append(errs, fmt.Errorf("log emit %s: %w", in.RuleID, err))
		}
	}
	return errors.Join(errs...)
}
