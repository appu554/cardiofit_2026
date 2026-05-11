package actions

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// SessionContext is the per-pharmacist session per v1.0 Part 12.4. A
// session opens when the pharmacist begins their review and closes when
// they explicitly end it; every action captured between open/close is
// tagged with the SessionID so the audit pipeline can reconstruct
// reasoning trails per resident-review session.
type SessionContext struct {
	SessionID         uuid.UUID
	PharmacistID      uuid.UUID
	StartedAt         time.Time
	EndedAt           *time.Time
	ResidentsReviewed []uuid.UUID
	ActionCount       int
}

// Sentinel errors for session lifecycle.
var (
	// ErrSessionNotFound indicates a session lookup or update targeted a
	// SessionID that the store has no record of.
	ErrSessionNotFound = errors.New("actions: session not found")

	// ErrSessionAlreadyEnded indicates EndSession was called on a session
	// that already has an EndedAt timestamp.
	ErrSessionAlreadyEnded = errors.New("actions: session already ended")
)

// SessionStore is the persistence contract for pharmacist session
// records. The InMemorySessionStore satisfies it for tests; the
// production Postgres-backed implementation lands with Task 8.
type SessionStore interface {
	Create(ctx context.Context, s SessionContext) error
	Update(ctx context.Context, s SessionContext) error
	Get(ctx context.Context, sessionID uuid.UUID) (SessionContext, error)
	RecordActionInSession(ctx context.Context, sessionID uuid.UUID, action Action) error
}

// StartSession opens a new pharmacist session, persists the initial
// SessionContext to the supplied store, and returns the populated record.
func StartSession(ctx context.Context, pharmacistID uuid.UUID, store SessionStore) (SessionContext, error) {
	s := SessionContext{
		SessionID:         uuid.New(),
		PharmacistID:      pharmacistID,
		StartedAt:         time.Now().UTC(),
		ResidentsReviewed: []uuid.UUID{},
	}
	if err := store.Create(ctx, s); err != nil {
		return SessionContext{}, err
	}
	return s, nil
}

// EndSession closes an open pharmacist session by stamping EndedAt and
// persisting the change. Returns ErrSessionAlreadyEnded when the
// session's EndedAt is already populated.
func EndSession(ctx context.Context, sessionID uuid.UUID, store SessionStore) (SessionContext, error) {
	s, err := store.Get(ctx, sessionID)
	if err != nil {
		return SessionContext{}, err
	}
	if s.EndedAt != nil {
		return SessionContext{}, ErrSessionAlreadyEnded
	}
	now := time.Now().UTC()
	s.EndedAt = &now
	if err := store.Update(ctx, s); err != nil {
		return SessionContext{}, err
	}
	return s, nil
}

// InMemorySessionStore is a goroutine-safe in-memory SessionStore used
// by tests and the action-handler test harness. It is NOT intended for
// production use — Task 8 wires the Postgres-backed implementation.
type InMemorySessionStore struct {
	mu       sync.Mutex
	sessions map[uuid.UUID]SessionContext
}

// NewInMemorySessionStore returns an empty in-memory session store.
func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{sessions: map[uuid.UUID]SessionContext{}}
}

// Create persists a new session. Returns an error if SessionID collides
// with an existing record (which would indicate a uuid collision or a
// caller bug).
func (s *InMemorySessionStore) Create(_ context.Context, sc SessionContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sessions[sc.SessionID]; exists {
		return errors.New("actions: session already exists")
	}
	s.sessions[sc.SessionID] = sc
	return nil
}

// Update overwrites an existing session record.
func (s *InMemorySessionStore) Update(_ context.Context, sc SessionContext) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, exists := s.sessions[sc.SessionID]; !exists {
		return ErrSessionNotFound
	}
	s.sessions[sc.SessionID] = sc
	return nil
}

// Get returns the SessionContext for the supplied SessionID or
// ErrSessionNotFound.
func (s *InMemorySessionStore) Get(_ context.Context, sessionID uuid.UUID) (SessionContext, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.sessions[sessionID]
	if !ok {
		return SessionContext{}, ErrSessionNotFound
	}
	return sc, nil
}

// RecordActionInSession increments ActionCount on the session. The
// action argument is accepted so a future implementation can index
// per-action-type counters without changing the interface.
func (s *InMemorySessionStore) RecordActionInSession(_ context.Context, sessionID uuid.UUID, _ Action) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	sc, ok := s.sessions[sessionID]
	if !ok {
		return ErrSessionNotFound
	}
	sc.ActionCount++
	s.sessions[sessionID] = sc
	return nil
}
