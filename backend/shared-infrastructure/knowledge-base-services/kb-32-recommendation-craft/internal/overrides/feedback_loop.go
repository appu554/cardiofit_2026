package overrides

// feedback_loop.go — override→rule-tuning feedback loop (Phase 2b Task 4)
//
// VisibilityClass: AD — override audit per Guidelines §5
//
// Detector scans override patterns for a given rule and emits an EthicsLog
// entry when the volume and inappropriate-override ratio exceed the configured
// thresholds. This creates a data-driven signal for the rule-tuning pipeline
// to review and potentially adjust recommendation rules.
//
// Thresholds (per Plan §Task-4):
//   - OverrideThreshold         = 30   (minimum total overrides in window)
//   - InappropriateRatioFloor   = 0.6  (inappropriate/total ≥ floor → emit)

import (
	"context"
	"fmt"
	"time"

	"github.com/cardiofit/shared/v2_substrate/ethics/ethics_log"
)

// OverrideThreshold is the minimum total number of overrides in the scan
// window before the inappropriate-override ratio is evaluated. Windows with
// fewer overrides are ignored to avoid noisy signals from low-volume rules.
const OverrideThreshold = 30

// InappropriateRatioFloor is the minimum ratio of inappropriate_override
// records to total override records that triggers an EthicsLog entry.
// Must be in [0,1]. A ratio of exactly 0.6 meets the floor.
const InappropriateRatioFloor = 0.6

// EthicsLogger is the subset of ethics_log.Logger that Detector uses, making
// the dependency injectable for testing.
type EthicsLogger interface {
	Append(ctx context.Context, e ethics_log.Entry) error
}

// Detector holds the Store and EthicsLogger dependencies for the feedback loop.
type Detector struct {
	store  Store
	logger EthicsLogger
}

// NewDetector constructs a Detector. Both store and logger must be non-nil.
func NewDetector(store Store, logger EthicsLogger) *Detector {
	return &Detector{store: store, logger: logger}
}

// Scan examines override patterns for ruleID in the window [since, now).
//
// It calls store.PatternSummary to count overrides by appropriateness flag.
// When the total count meets OverrideThreshold AND the ratio of
// "inappropriate_override" to total meets InappropriateRatioFloor, Scan
// emits an EthicsLog entry with:
//   - EntryType: EntryTypePatternDetected
//   - Severity:  3
//   - Description: "<ruleID>: <total> overrides, <inappropriate> inappropriate
//     (ratio <ratio>); rule-tuning review recommended"
//
// Scan returns nil when no entry is emitted. It propagates errors from both
// the Store and the Logger unchanged.
func (d *Detector) Scan(ctx context.Context, ruleID string, since time.Time) error {
	summary, err := d.store.PatternSummary(ctx, ruleID, since)
	if err != nil {
		return fmt.Errorf("feedback_loop: pattern_summary: %w", err)
	}

	inappropriate := summary["inappropriate_override"]
	appropriate := summary["appropriate_override"]
	mixed := summary["mixed"]
	total := inappropriate + appropriate + mixed

	if total < OverrideThreshold {
		return nil
	}

	ratio := float64(inappropriate) / float64(total)
	if ratio < InappropriateRatioFloor {
		return nil
	}

	entry := ethics_log.Entry{
		EntryType: ethics_log.EntryTypePatternDetected,
		Severity:  3,
		Description: fmt.Sprintf(
			"%s: %d overrides, %d inappropriate (ratio %.2f); rule-tuning review recommended",
			ruleID, total, inappropriate, ratio,
		),
		Status: ethics_log.StatusOpen,
	}

	if err := d.logger.Append(ctx, entry); err != nil {
		return fmt.Errorf("feedback_loop: ethics_log append: %w", err)
	}
	return nil
}
