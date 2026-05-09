// Package ethics_log — see logger.go for package-level documentation.
package ethics_log

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// Querier provides read-only query methods over the ethics log Store.
type Querier struct{ store Store }

// NewQuerier returns a Querier backed by s.
func NewQuerier(s Store) *Querier { return &Querier{store: s} }

// ByDecision returns all entries whose DecisionID matches id.
func (q *Querier) ByDecision(ctx context.Context, id uuid.UUID) ([]Entry, error) {
	all, err := q.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range all {
		if e.DecisionID == id {
			out = append(out, e)
		}
	}
	return out, nil
}

// OpenAtSeverity returns all entries whose Status is StatusOpen and whose
// Severity equals sev.
func (q *Querier) OpenAtSeverity(ctx context.Context, sev int) ([]Entry, error) {
	all, err := q.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range all {
		if e.Severity == sev && e.Status == StatusOpen {
			out = append(out, e)
		}
	}
	return out, nil
}

// ByTimeWindow returns all entries whose CreatedAt is within [since, until] inclusive.
func (q *Querier) ByTimeWindow(ctx context.Context, since, until time.Time) ([]Entry, error) {
	all, err := q.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range all {
		if !e.CreatedAt.Before(since) && !e.CreatedAt.After(until) {
			out = append(out, e)
		}
	}
	return out, nil
}

// ByEntryType returns all entries whose EntryType equals t.
func (q *Querier) ByEntryType(ctx context.Context, t EntryType) ([]Entry, error) {
	all, err := q.store.List(ctx)
	if err != nil {
		return nil, err
	}
	out := []Entry{}
	for _, e := range all {
		if e.EntryType == t {
			out = append(out, e)
		}
	}
	return out, nil
}
