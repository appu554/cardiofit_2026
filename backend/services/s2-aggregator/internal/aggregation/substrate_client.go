package aggregation

import (
	"context"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/s2-aggregator/internal/substrate_types"
)

// SubstrateClient is the minimal interface s2-aggregator needs to read
// substrate. It decouples the aggregator from kb-32's internal/store/postgres
// PostgresSubstrateClient (which is internal/ and therefore unimportable)
// and from kb-20's GORM models. Task 8 of the build plan wires this
// interface to a production adapter — for Layer 1 + tests, the in-memory
// fake below is sufficient.
//
// Architectural framing per v1.0 Part 2: the ViewBuilder is a pure
// aggregation layer; SubstrateClient is the read port.
type SubstrateClient interface {
	// SnapshotFor returns the current ClinicalSnapshot-equivalent values
	// for a resident as of asOf. Returned values are the most-recent
	// observation per parameter at-or-before asOf; nil values indicate
	// "no observation on record" per v1.0 Part 5.3 "Missing data category".
	SnapshotFor(ctx context.Context, residentID uuid.UUID, asOf time.Time) (Snapshot, error)

	// TrajectoryHistory returns the observation series for one parameter,
	// ordered chronologically (oldest first). Returns an empty slice when
	// no observations exist; never returns nil.
	TrajectoryHistory(ctx context.Context, residentID uuid.UUID, parameter string) ([]substrate_types.Observation, error)

	// RecentPRNAdministrations returns the administration stream for a
	// (resident, class) pair across the trailing window CAPE Phase 1
	// requires (120 days). Caller (BuildTrajectories) passes this to
	// prn_velocity.Compute equivalents.
	RecentPRNAdministrations(ctx context.Context, residentID uuid.UUID, class substrate_types.PRNClass, asOf time.Time) ([]substrate_types.PRNAdministration, error)
}

// Snapshot is the s2-aggregator's view of kb-20's ClinicalSnapshot. Fields
// are pointer-to-float64 so "no observation on record" is distinguishable
// from "value is zero" — v1.0 Part 5.3 requires this distinction.
type Snapshot struct {
	ResidentID uuid.UUID
	AsOf       time.Time

	EGFR        *float64 // mL/min/1.73m²
	DBI         *float64 // Drug Burden Index (unit-less)
	ACB         *float64 // Anticholinergic Cognitive Burden (unit-less)
	CFS         *float64 // Clinical Frailty Scale (1–9)
	Weight      *float64 // kg
	BPSystolic  *float64 // mmHg
	BPDiastolic *float64 // mmHg
}

// inMemorySubstrateClient is a test-facing SubstrateClient backed by a
// flat slice. It is exported via NewInMemorySubstrateClient for tests in
// this package and (Task 8) any integration tests that want to drive the
// aggregator without a database.
type inMemorySubstrateClient struct {
	mu              sync.RWMutex
	observations    []substrate_types.Observation
	administrations []substrate_types.PRNAdministration
}

// NewInMemorySubstrateClient returns an empty in-memory fake. Use
// WithObservations / WithAdministrations to seed it.
func NewInMemorySubstrateClient() *inMemorySubstrateClient {
	return &inMemorySubstrateClient{}
}

// WithObservations seeds the fake with observation rows. Returns the
// receiver for fluent chaining.
func (c *inMemorySubstrateClient) WithObservations(obs ...substrate_types.Observation) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.observations = append(c.observations, obs...)
	return c
}

// WithAdministrations seeds the fake with PRN administration rows.
func (c *inMemorySubstrateClient) WithAdministrations(adm ...substrate_types.PRNAdministration) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.administrations = append(c.administrations, adm...)
	return c
}

// SnapshotFor implements SubstrateClient by returning the most-recent
// observation per parameter at-or-before asOf.
func (c *inMemorySubstrateClient) SnapshotFor(_ context.Context, residentID uuid.UUID, asOf time.Time) (Snapshot, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	snap := Snapshot{ResidentID: residentID, AsOf: asOf}
	latest := map[string]substrate_types.Observation{}
	for _, o := range c.observations {
		if o.ResidentID != residentID {
			continue
		}
		if o.ObservedAt.After(asOf) {
			continue
		}
		cur, ok := latest[o.Parameter]
		if !ok || o.ObservedAt.After(cur.ObservedAt) {
			latest[o.Parameter] = o
		}
	}

	assign := func(p string, dst **float64) {
		if o, ok := latest[p]; ok {
			v := o.Value
			*dst = &v
		}
	}
	assign("egfr", &snap.EGFR)
	assign("dbi", &snap.DBI)
	assign("acb", &snap.ACB)
	assign("cfs", &snap.CFS)
	assign("weight", &snap.Weight)
	assign("bp_systolic", &snap.BPSystolic)
	assign("bp_diastolic", &snap.BPDiastolic)
	return snap, nil
}

// TrajectoryHistory implements SubstrateClient by returning the parameter's
// observation series sorted oldest-first.
func (c *inMemorySubstrateClient) TrajectoryHistory(_ context.Context, residentID uuid.UUID, parameter string) ([]substrate_types.Observation, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]substrate_types.Observation, 0)
	for _, o := range c.observations {
		if o.ResidentID == residentID && o.Parameter == parameter {
			out = append(out, o)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ObservedAt.Before(out[j].ObservedAt) })
	return out, nil
}

// RecentPRNAdministrations implements SubstrateClient by returning the
// (resident, class) administration stream over the trailing 120-day
// window (matches prn_velocity.Compute's outer window per CAPE Guidelines).
func (c *inMemorySubstrateClient) RecentPRNAdministrations(_ context.Context, residentID uuid.UUID, class substrate_types.PRNClass, asOf time.Time) ([]substrate_types.PRNAdministration, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cutoff := asOf.Add(-120 * 24 * time.Hour)
	out := make([]substrate_types.PRNAdministration, 0)
	for _, a := range c.administrations {
		if a.ResidentID != residentID || a.Class != class {
			continue
		}
		if a.AdministeredAt.After(cutoff) && !a.AdministeredAt.After(asOf) {
			out = append(out, a)
		}
	}
	return out, nil
}
