package decision_metadata

import (
	"context"
	"sync"

	"github.com/google/uuid"
)

// InMemoryStore is a map-backed Store suitable for unit tests.
// It is safe for concurrent use; all mutations and reads are guarded
// by an RWMutex, matching the Phase 1a Task 3 pattern.
type InMemoryStore struct {
	mu sync.RWMutex
	m  map[uuid.UUID]Metadata
}

// compile-time interface satisfaction assertion.
var _ Store = (*InMemoryStore)(nil)

// NewInMemoryStore returns an initialised InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{m: map[uuid.UUID]Metadata{}}
}

// Put stores m under its DecisionID. Overwrites any prior entry with the same key.
func (s *InMemoryStore) Put(_ context.Context, m Metadata) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.m[m.DecisionID] = m
	return nil
}

// Get returns a pointer to the stored Metadata for id, or (nil, nil) when
// no entry exists.
func (s *InMemoryStore) Get(_ context.Context, id uuid.UUID) (*Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if m, ok := s.m[id]; ok {
		cp := m
		return &cp, nil
	}
	return nil, nil
}

// QueryBySubject returns all Metadata records whose AffectedSubjectID equals
// subjectID. Order is not guaranteed.
func (s *InMemoryStore) QueryBySubject(_ context.Context, subjectID string) ([]Metadata, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := []Metadata{}
	for _, m := range s.m {
		if m.AffectedSubjectID == subjectID {
			out = append(out, m)
		}
	}
	return out, nil
}
