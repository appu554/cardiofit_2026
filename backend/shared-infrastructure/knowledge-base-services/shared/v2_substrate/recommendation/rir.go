package recommendation

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// RIRResult is the v3 §11 line 588 Layer-C operational North Star metric:
// the Recommendation Implementation Rate over a rolling window.
//
// Implementation semantics (resolves Task 3 review concern):
//
// "Actioned" means the recommendation's CURRENT state is one of
// `implemented`, `monitoring-active`, or `outcome-recorded`. A
// recommendation that went `submitted → decided (no-action) → closed`
// does NOT count toward the actioned count because no implementation
// occurred. RIR is an IMPLEMENTATION rate, not a "touched" rate.
//
// Note: the recommendations table does not preserve the full state
// history, so a `closed` row cannot be distinguished between
// closed-after-implemented vs closed-via-decided-no-action by table data
// alone. For the substrate-level computation, `closed` is conservatively
// UNCOUNTED. The full lifecycle history lives in EvidenceTrace.
//
// See RIR_SEMANTICS.md in this package directory for the full rationale
// and the divergence note vs migration 023's matview.
type RIRResult struct {
	AuthorID    uuid.UUID
	Window      time.Duration
	Submitted   int
	Actioned    int
	RatePercent float64
}

// ComputeRIR returns the rolling-window RIR for one author. Window is
// typically 28 days (per Ramsey 2025 measurement basis).
//
// Submitted = recommendations authored by authorID with submitted_at
//
//	within the window.
//
// Actioned  = subset of Submitted whose current state is one of
//
//	`implemented`, `monitoring-active`, `outcome-recorded`,
//	with decided_at populated within the window.
//
// This Go function is the AUTHORITATIVE RIR computation per v3 spec.
// The matview `recommendation_rir_28d` from migration 023 uses a looser
// "actioned" set and is documented as deprecated-pending-tightening in
// RIR_SEMANTICS.md. New consumers must use ComputeRIR.
func ComputeRIR(ctx context.Context, db *sql.DB, authorID uuid.UUID,
	window time.Duration) (RIRResult, error) {
	wind := durationToInterval(window)
	const q = `
WITH eligible AS (
  SELECT id, state, submitted_at, decided_at
  FROM recommendations
  WHERE author_id = $1
    AND submitted_at IS NOT NULL
    AND submitted_at >= NOW() - $2::interval
)
SELECT
  COUNT(*),
  COUNT(*) FILTER (WHERE
    state IN ('implemented','monitoring-active','outcome-recorded')
    AND decided_at IS NOT NULL
    AND decided_at <= submitted_at + $3::interval
  )
FROM eligible`
	row := db.QueryRowContext(ctx, q, authorID, wind, wind)

	var submitted, actioned int
	if err := row.Scan(&submitted, &actioned); err != nil {
		return RIRResult{}, fmt.Errorf("scan rir: %w", err)
	}
	rate := 0.0
	if submitted > 0 {
		rate = 100.0 * float64(actioned) / float64(submitted)
	}
	return RIRResult{
		AuthorID:    authorID,
		Window:      window,
		Submitted:   submitted,
		Actioned:    actioned,
		RatePercent: rate,
	}, nil
}

// durationToInterval renders a Go time.Duration as a Postgres interval
// string. Day-aligned durations render as "N days"; otherwise hour-aligned.
func durationToInterval(d time.Duration) string {
	if d >= 24*time.Hour && d%(24*time.Hour) == 0 {
		days := int(d / (24 * time.Hour))
		return fmt.Sprintf("%d days", days)
	}
	hours := int(d / time.Hour)
	return fmt.Sprintf("%d hours", hours)
}
