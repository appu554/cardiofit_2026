package consent

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// ExpirySweeper periodically scans the consents table for active consents
// whose valid_until has passed and transitions them to expired via the
// Lifecycle engine (so an EvidenceTrace edge is emitted, just as a
// human-initiated transition would).
//
// Operational note: production deployment wires this on a 1-hour ticker.
// The Postgres partial index idx_consents_expiry_sweep makes the
// per-sweep query cheap. The sweeper does NOT bypass the Lifecycle —
// every expiry produces a regulator-auditable trail.
type ExpirySweeper struct {
	store *PostgresStore
	lc    *Lifecycle
	now   func() time.Time
}

// NewExpirySweeper constructs a sweeper. now() is injected for
// deterministic tests; production callers pass time.Now.
func NewExpirySweeper(store *PostgresStore, lc *Lifecycle, now func() time.Time) *ExpirySweeper {
	return &ExpirySweeper{store: store, lc: lc, now: now}
}

// RunOnce performs a single sweep. Each expired consent receives one
// Lifecycle.Transition call (active → expired) with a system actor.
// Per-row failures do NOT abort the sweep — production deployments
// rely on the next tick to retry; partial drains are operationally
// acceptable for this idempotent operation.
func (s *ExpirySweeper) RunOnce(ctx context.Context) error {
	const q = `
SELECT id FROM consents
WHERE state = 'active' AND valid_until IS NOT NULL AND valid_until < $1
LIMIT 500`
	rows, err := s.store.db.QueryContext(ctx, q, s.now())
	if err != nil {
		return err
	}
	defer rows.Close()

	var ids []uuid.UUID
	for rows.Next() {
		var id uuid.UUID
		if err := rows.Scan(&id); err != nil {
			return err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return err
	}

	systemActor := uuid.Nil // system-generated transition
	for _, id := range ids {
		if err := s.lc.Transition(ctx, TransitionRequest{
			ConsentID:  id,
			ToState:    models.ConsentStateExpired,
			ActorID:    systemActor,
			ActorRole:  "system_expiry_sweeper",
			OccurredAt: s.now(),
			Notes:      "automatic expiry on valid_until passed",
		}); err != nil {
			// Per-row failure: log + continue. The next sweep retries.
			// Aborting the whole sweep on first failure would leave
			// other expired consents unprocessed for a full tick.
			continue
		}
	}
	return nil
}
