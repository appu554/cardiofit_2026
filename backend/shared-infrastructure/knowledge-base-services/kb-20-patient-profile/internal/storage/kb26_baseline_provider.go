package storage

import (
	"context"
	"sync"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/delta"
)

// InMemoryBaselineProvider is the MVP delta.BaselineProvider implementation.
// It serves baselines from an in-memory map keyed by (residentID, vitalType)
// and is suitable for unit tests + early-pilot deployments where kb-26 may
// not yet expose its AcuteRepository over the network.
//
// Production wiring (deferred — non-blocking for β.2-C exit): replace with
// a thin adapter over kb-26's AcuteRepository.FetchBaseline(patientID,
// vitalType) using the kb-26 internal HTTP/gRPC API. The interface contract
// (delta.BaselineProvider) does not change; this seam is exactly where the
// migration lands.
type InMemoryBaselineProvider struct {
	mu        sync.RWMutex
	baselines map[string]delta.Baseline
}

// NewInMemoryBaselineProvider returns an empty provider; seed with Seed().
func NewInMemoryBaselineProvider() *InMemoryBaselineProvider {
	return &InMemoryBaselineProvider{baselines: map[string]delta.Baseline{}}
}

func keyFor(residentID uuid.UUID, vitalType string) string {
	return residentID.String() + "::" + vitalType
}

// Seed inserts or replaces a baseline for (residentID, vitalType). Test-only
// helper; production code populates via the (deferred) kb-26 adapter.
func (p *InMemoryBaselineProvider) Seed(residentID uuid.UUID, vitalType string, b delta.Baseline) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.baselines[keyFor(residentID, vitalType)] = b
}

// FetchBaseline implements delta.BaselineProvider. Returns delta.ErrNoBaseline
// when no entry exists for (residentID, vitalType).
func (p *InMemoryBaselineProvider) FetchBaseline(ctx context.Context, residentID uuid.UUID, vitalType string) (*delta.Baseline, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	if b, ok := p.baselines[keyFor(residentID, vitalType)]; ok {
		return &b, nil
	}
	return nil, delta.ErrNoBaseline
}
