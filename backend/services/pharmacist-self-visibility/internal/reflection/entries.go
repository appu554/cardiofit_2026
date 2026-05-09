package reflection

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/google/uuid"
)

// VisibilityClass: POA — only the authoring pharmacist can read these entries.
// Pattern detection is explicitly forbidden on this entity.
var ErrNotAuthorized = errors.New("reflection: not authorized")

type Entry struct {
	ID           uuid.UUID
	PharmacistID uuid.UUID
	PromptID     *uuid.UUID
	Body         string
	Tags         []string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

type Store interface {
	Create(ctx context.Context, e Entry) (Entry, error)
	Get(ctx context.Context, requester uuid.UUID, id uuid.UUID) (*Entry, error)
	ListByAuthor(ctx context.Context, pharmacistID uuid.UUID, limit int) ([]Entry, error)
}

type InMemoryStore struct {
	mu      sync.RWMutex
	entries map[uuid.UUID]Entry
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{entries: make(map[uuid.UUID]Entry)}
}

func (s *InMemoryStore) Create(_ context.Context, e Entry) (Entry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	now := time.Now().UTC()
	e.CreatedAt = now
	e.UpdatedAt = now
	s.entries[e.ID] = e
	return e, nil
}

func (s *InMemoryStore) Get(_ context.Context, requester, id uuid.UUID) (*Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	e, ok := s.entries[id]
	if !ok {
		return nil, ErrNotAuthorized // do NOT leak existence to non-authors
	}
	if e.PharmacistID != requester {
		return nil, ErrNotAuthorized
	}
	return &e, nil
}

func (s *InMemoryStore) ListByAuthor(_ context.Context, pharmacistID uuid.UUID, limit int) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, 0)
	for _, e := range s.entries {
		if e.PharmacistID == pharmacistID {
			out = append(out, e)
			if len(out) >= limit {
				break
			}
		}
	}
	return out, nil
}
