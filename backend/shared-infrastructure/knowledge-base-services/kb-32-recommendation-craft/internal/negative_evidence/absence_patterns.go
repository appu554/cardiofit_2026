// Package negative_evidence implements Stage 3 absence-pattern queries for
// negative-evidence audit defensibility.
//
// VisibilityClass: AD — negative-evidence audit per Guidelines §7
//
// The three CQL absence-query patterns answer a regulator's question:
// "What evidence does an absent observation support?"
//
//  1. Bounded-window absence — "no fall in the past 90 days".
//     WindowDays MUST be > 0 for this pattern.
//
//  2. Periodic-review absence — "no medication review in the past 12 months".
//     WindowDays is conventionally 365; callers MAY pass 0 (the querier then
//     applies the conventional 365-day window internally).
//
//  3. Indication-documentation absence — "no documented indication for PPI".
//     WindowDays is typically irrelevant; pass 0.
//
// Every AbsenceResult carries an EvidenceText — a human-readable defensibility
// statement suitable for direct inclusion in a recommendation audit trail.
package negative_evidence

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// AbsencePattern enum
// ---------------------------------------------------------------------------

// AbsencePattern identifies which CQL absence-query template is executed.
type AbsencePattern int

const (
	// PatternBoundedWindow corresponds to a "no <observation> in the past N days"
	// CQL query. WindowDays must be > 0.
	PatternBoundedWindow AbsencePattern = iota + 1

	// PatternPeriodicReview corresponds to a "no <observation> in the past 12 months"
	// CQL query. WindowDays is conventionally 365; the querier may apply 365
	// internally when the caller passes 0.
	PatternPeriodicReview

	// PatternIndicationDocumentation corresponds to a "no documented indication
	// for <observation>" CQL query. WindowDays is typically irrelevant (pass 0).
	PatternIndicationDocumentation
)

// String returns the database / audit-log string representation of the pattern.
func (p AbsencePattern) String() string {
	switch p {
	case PatternBoundedWindow:
		return "bounded_window"
	case PatternPeriodicReview:
		return "periodic_review"
	case PatternIndicationDocumentation:
		return "indication_documentation"
	default:
		return fmt.Sprintf("unknown_pattern_%d", int(p))
	}
}

// IsValidPattern reports whether p is one of the three recognised AbsencePattern
// constants. The check is exhaustive — only the three defined enum values pass.
func IsValidPattern(p AbsencePattern) bool {
	switch p {
	case PatternBoundedWindow, PatternPeriodicReview, PatternIndicationDocumentation:
		return true
	}
	return false
}

// ---------------------------------------------------------------------------
// AbsenceQuery
// ---------------------------------------------------------------------------

// AbsenceQuery describes a single negative-evidence lookup request.
type AbsenceQuery struct {
	// Pattern selects the CQL absence-query template to execute.
	Pattern AbsencePattern

	// ResidentID identifies the patient/resident being queried.
	ResidentID uuid.UUID

	// ObservationKind is the observation category (e.g. "fall", "ppi_indication",
	// "benzodiazepine_review"). Must not be empty.
	ObservationKind string

	// WindowDays is the look-back window.
	//   - PatternBoundedWindow: required; must be > 0.
	//   - PatternPeriodicReview: optional; 0 is treated as "use conventional 365".
	//   - PatternIndicationDocumentation: not meaningful; pass 0.
	WindowDays int
}

// Validate returns a non-nil error when the query is structurally invalid.
//
// Rules:
//   - Pattern must be one of the three defined constants (IsValidPattern).
//   - ObservationKind must not be empty.
//   - WindowDays must be > 0 for PatternBoundedWindow; other patterns may use 0.
func (q AbsenceQuery) Validate() error {
	if !IsValidPattern(q.Pattern) {
		return fmt.Errorf("negative_evidence: invalid pattern %d", int(q.Pattern))
	}
	if q.ObservationKind == "" {
		return errors.New("negative_evidence: ObservationKind must not be empty")
	}
	if q.Pattern == PatternBoundedWindow && q.WindowDays <= 0 {
		return fmt.Errorf(
			"negative_evidence: WindowDays must be > 0 for bounded-window pattern, got %d",
			q.WindowDays,
		)
	}
	return nil
}

// ---------------------------------------------------------------------------
// AbsenceResult
// ---------------------------------------------------------------------------

// AbsenceResult is the outcome of a single QueryAbsence call.
type AbsenceResult struct {
	// Confirmed is true when the observation is genuinely absent (no record found
	// within the query window). False means the observation was found — the
	// "absence" claim cannot be made.
	Confirmed bool

	// LastSeenAt is the timestamp of the most recent observation found within the
	// query window. Nil when Confirmed=true (no observation found).
	LastSeenAt *time.Time

	// QueriedAt is the UTC timestamp at which the query was executed.
	QueriedAt time.Time

	// EvidenceText is a human-readable, audit-defensible statement describing the
	// result. It is non-empty in all cases (both Confirmed=true and Confirmed=false).
	EvidenceText string
}

// ---------------------------------------------------------------------------
// Querier interface
// ---------------------------------------------------------------------------

// Querier is the persistence boundary for absence queries.
// All implementations must be safe for concurrent use.
type Querier interface {
	// QueryAbsence executes the absence query and returns the result.
	// It returns a non-nil error only on infrastructure failures (not on
	// "presence detected" — that is expressed via AbsenceResult.Confirmed=false).
	QueryAbsence(ctx context.Context, q AbsenceQuery) (AbsenceResult, error)
}

// ---------------------------------------------------------------------------
// InMemoryQuerier — test fixture
// ---------------------------------------------------------------------------

// InMemoryQuerier is a deterministic, thread-safe Querier intended for unit
// and integration tests. It mirrors the InMemory fixture pattern used by
// citations.InMemoryRegistry and other Phase 1b/2b test fakes.
//
// Construct with:
//   - NewInMemoryQuerier(nil)               — always returns Confirmed=true (absence)
//   - NewInMemoryQuerier(&lastSeenTime)     — always returns Confirmed=false (presence)
//   - NewInMemoryQuerierWithError(sentinel) — always returns (zero, sentinel)
type InMemoryQuerier struct {
	lastSeenAt *time.Time // nil = absence; non-nil = presence at this time
	err        error      // non-nil = always return this error
}

// NewInMemoryQuerier returns a querier that reports absence (Confirmed=true) when
// lastSeenAt is nil, or presence (Confirmed=false) when lastSeenAt is non-nil.
func NewInMemoryQuerier(lastSeenAt *time.Time) *InMemoryQuerier {
	return &InMemoryQuerier{lastSeenAt: lastSeenAt}
}

// NewInMemoryQuerierWithError returns a querier that always returns the given error.
// Used to test error-propagation paths in callers.
func NewInMemoryQuerierWithError(err error) *InMemoryQuerier {
	return &InMemoryQuerier{err: err}
}

// QueryAbsence implements Querier.
func (q *InMemoryQuerier) QueryAbsence(_ context.Context, query AbsenceQuery) (AbsenceResult, error) {
	if q.err != nil {
		return AbsenceResult{}, q.err
	}

	now := time.Now().UTC()

	if q.lastSeenAt == nil {
		// Absence confirmed.
		return AbsenceResult{
			Confirmed:    true,
			LastSeenAt:   nil,
			QueriedAt:    now,
			EvidenceText: buildAbsenceText(query, now),
		}, nil
	}

	// Presence detected.
	t := *q.lastSeenAt
	return AbsenceResult{
		Confirmed:    false,
		LastSeenAt:   &t,
		QueriedAt:    now,
		EvidenceText: buildPresenceText(query, t, now),
	}, nil
}

// ---------------------------------------------------------------------------
// Evidence text helpers
// ---------------------------------------------------------------------------

func buildAbsenceText(q AbsenceQuery, at time.Time) string {
	switch q.Pattern {
	case PatternBoundedWindow:
		return fmt.Sprintf(
			"No record of '%s' found in the past %d day(s) for resident %s as of %s. "+
				"Absence supports deprescribing defensibility (bounded-window query).",
			q.ObservationKind, q.WindowDays, q.ResidentID, at.Format(time.RFC3339),
		)
	case PatternPeriodicReview:
		window := q.WindowDays
		if window == 0 {
			window = 365 // conventional periodic-review window
		}
		return fmt.Sprintf(
			"No record of '%s' found in the past %d day(s) for resident %s as of %s. "+
				"Absence indicates periodic review is overdue (periodic-review query).",
			q.ObservationKind, window, q.ResidentID, at.Format(time.RFC3339),
		)
	case PatternIndicationDocumentation:
		return fmt.Sprintf(
			"No documented indication for '%s' found for resident %s as of %s. "+
				"Absence supports medication appropriateness challenge (indication-documentation query).",
			q.ObservationKind, q.ResidentID, at.Format(time.RFC3339),
		)
	default:
		return fmt.Sprintf(
			"Absence confirmed for '%s' (resident %s) at %s.",
			q.ObservationKind, q.ResidentID, at.Format(time.RFC3339),
		)
	}
}

func buildPresenceText(q AbsenceQuery, lastSeen, at time.Time) string {
	return fmt.Sprintf(
		"Observation '%s' was found for resident %s (last seen: %s) as of %s. "+
			"Absence cannot be claimed; deprescribing defensibility statement withheld.",
		q.ObservationKind, q.ResidentID,
		lastSeen.Format(time.RFC3339), at.Format(time.RFC3339),
	)
}

// Compile-time check: InMemoryQuerier implements Querier.
var _ Querier = (*InMemoryQuerier)(nil)
