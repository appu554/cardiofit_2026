package framing

// optout_store.go — read/write surface for the prescriber_framing_optout
// substrate (migration 047). This is the authoritative source for the
// "GP has opted out of per-GP framing learning" signal that toxicity guard
// #3 (Guidelines §8) enforces inside PerGPObserver.Suggest.
//
// The existing ObservationSource.HasOptedOut method is preserved unchanged
// — Postgres-backed ObservationSource implementations should delegate
// HasOptedOut to OptOutStore.IsOptedOut so there is exactly one source of
// truth for opt-out state. See per_gp_observer.go for the consumer wiring.

import (
	"context"
	"database/sql"
	"sync"
	"time"

	"github.com/google/uuid"
)

// OptOutStore is the read/write port for prescriber framing opt-out state.
// Implementations must be safe for concurrent use from multiple goroutines.
type OptOutStore interface {
	// RegisterOptOut records or refreshes an opt-out for the given GP.
	// The operation is idempotent: re-registering an already-opted-out GP
	// MUST NOT error and MUST leave the GP currently opted-out (revoked_at
	// reset to NULL on the Postgres implementation).
	//
	// reason may be empty; it is stored verbatim and is purely advisory.
	RegisterOptOut(ctx context.Context, gpID uuid.UUID, reason string) error

	// RevokeOptOut flips the opt-out off for the given GP. If the GP is
	// not currently opted-out, the call is a no-op (no error). The original
	// opt-out row is preserved (revoked_at is set, not deleted) so the
	// historical action remains auditable.
	RevokeOptOut(ctx context.Context, gpID uuid.UUID) error

	// IsOptedOut reports whether the GP is currently opted-out
	// (revoked_at IS NULL on the persisted row, or no row exists).
	IsOptedOut(ctx context.Context, gpID uuid.UUID) (bool, error)
}

// optOutRecord is the in-memory representation of one persisted row.
type optOutRecord struct {
	reason     string
	optedOutAt time.Time
	revokedAt  *time.Time
}

// ---------------------------------------------------------------------------
// In-memory implementation (dev / test)
// ---------------------------------------------------------------------------

// InMemoryOptOutStore is a thread-safe OptOutStore backed by an in-process
// map. Suitable for dev-mode boots and unit tests; not durable.
type InMemoryOptOutStore struct {
	mu      sync.RWMutex
	records map[uuid.UUID]optOutRecord
}

// Compile-time interface conformance.
var _ OptOutStore = (*InMemoryOptOutStore)(nil)

// NewInMemoryOptOutStore constructs an empty InMemoryOptOutStore.
func NewInMemoryOptOutStore() *InMemoryOptOutStore {
	return &InMemoryOptOutStore{records: make(map[uuid.UUID]optOutRecord)}
}

// RegisterOptOut writes (or refreshes) an opt-out for gpID. Idempotent.
func (s *InMemoryOptOutStore) RegisterOptOut(_ context.Context, gpID uuid.UUID, reason string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.records[gpID] = optOutRecord{
		reason:     reason,
		optedOutAt: time.Now().UTC(),
		revokedAt:  nil,
	}
	return nil
}

// RevokeOptOut sets revoked_at on the record for gpID. No-op if no active
// opt-out exists for the GP.
func (s *InMemoryOptOutStore) RevokeOptOut(_ context.Context, gpID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	rec, ok := s.records[gpID]
	if !ok || rec.revokedAt != nil {
		// No-op: nothing to revoke. Preserves audit semantics (the historical
		// opt-out row, if any, is left intact).
		return nil
	}
	now := time.Now().UTC()
	rec.revokedAt = &now
	s.records[gpID] = rec
	return nil
}

// IsOptedOut reports whether gpID is currently opted-out.
func (s *InMemoryOptOutStore) IsOptedOut(_ context.Context, gpID uuid.UUID) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[gpID]
	if !ok {
		return false, nil
	}
	return rec.revokedAt == nil, nil
}

// ---------------------------------------------------------------------------
// Postgres implementation
// ---------------------------------------------------------------------------

// PostgresOptOutStore is the Postgres-backed OptOutStore over migration 047.
type PostgresOptOutStore struct {
	db *sql.DB
}

// Compile-time interface conformance.
var _ OptOutStore = (*PostgresOptOutStore)(nil)

// NewPostgresOptOutStore constructs a PostgresOptOutStore over the supplied
// *sql.DB. db may be nil at construction time — errors surface per-call.
func NewPostgresOptOutStore(db *sql.DB) *PostgresOptOutStore {
	return &PostgresOptOutStore{db: db}
}

// RegisterOptOut inserts (or refreshes) an opt-out via INSERT ... ON CONFLICT.
// Re-registering after a revoke flips revoked_at back to NULL and refreshes
// opted_out_at + reason — that gives idempotent re-register semantics.
func (s *PostgresOptOutStore) RegisterOptOut(ctx context.Context, gpID uuid.UUID, reason string) error {
	const stmt = `
		INSERT INTO prescriber_framing_optout (gp_id, reason, opted_out_at, revoked_at)
		VALUES ($1, $2, now(), NULL)
		ON CONFLICT (gp_id) DO UPDATE
		    SET reason       = EXCLUDED.reason,
		        opted_out_at = now(),
		        revoked_at   = NULL
	`
	var reasonArg any
	if reason == "" {
		reasonArg = nil
	} else {
		reasonArg = reason
	}
	_, err := s.db.ExecContext(ctx, stmt, gpID, reasonArg)
	return err
}

// RevokeOptOut sets revoked_at on the currently-active row for gpID. If no
// active row exists the UPDATE affects zero rows and returns no error
// (idempotent no-op).
func (s *PostgresOptOutStore) RevokeOptOut(ctx context.Context, gpID uuid.UUID) error {
	const stmt = `
		UPDATE prescriber_framing_optout
		   SET revoked_at = now()
		 WHERE gp_id = $1
		   AND revoked_at IS NULL
	`
	_, err := s.db.ExecContext(ctx, stmt, gpID)
	return err
}

// IsOptedOut reports whether gpID has an active (revoked_at IS NULL) row.
func (s *PostgresOptOutStore) IsOptedOut(ctx context.Context, gpID uuid.UUID) (bool, error) {
	const stmt = `
		SELECT EXISTS (
		    SELECT 1
		      FROM prescriber_framing_optout
		     WHERE gp_id = $1
		       AND revoked_at IS NULL
		)
	`
	var exists bool
	if err := s.db.QueryRowContext(ctx, stmt, gpID).Scan(&exists); err != nil {
		return false, err
	}
	return exists, nil
}
