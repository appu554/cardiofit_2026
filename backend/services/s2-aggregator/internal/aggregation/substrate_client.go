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

	// PendingRecommendations returns the kb-32 generator.Packet rows that
	// are currently in a non-terminal lifecycle state (detected, drafted,
	// submitted, viewed) for this resident. Returns an empty slice (never
	// nil) when none are pending.
	PendingRecommendations(ctx context.Context, residentID uuid.UUID) ([]substrate_types.RecommendationPacket, error)

	// RecommendationAssessment returns the 5-dimension appropriateness
	// scores attached to a recommendation per kb-32 Phase 2-completion
	// Task 2. Returns the zero Assessment when no assessment is on record.
	RecommendationAssessment(ctx context.Context, recID uuid.UUID) (substrate_types.AssessmentScores, error)

	// RecommendationCitations returns the fire-time citation pins per
	// kb-32 Guidelines §6. Returns an empty slice when no citations are
	// pinned (rare; usually indicates the packet pre-dates citation
	// pinning).
	RecommendationCitations(ctx context.Context, recID uuid.UUID) ([]substrate_types.Citation, error)

	// RecommendationOverrides returns the override history for a
	// recommendation per kb-32 Phase 2-completion Task 5. Ordering is
	// chronological (oldest first). Empty slice when no overrides exist.
	RecommendationOverrides(ctx context.Context, recID uuid.UUID) ([]substrate_types.OverrideReason, error)

	// ActiveRestraintSignals returns all currently-active restraint
	// signals for this resident. Phase 1 commitment per S2 v1.0 Part 7.3:
	// the SubstrateClient MUST NOT filter signals based on transition
	// criteria — that is informational-only in Phase 1.
	ActiveRestraintSignals(ctx context.Context, residentID uuid.UUID) ([]substrate_types.RestraintSignal, error)

	// FailedInterventionHistory returns the resident's failed-intervention
	// records since `since`. Per S2 v1.0 Part 8.4 default the panel
	// supplies a 12-month `since` for the standard view.
	//
	// KNOWN GAP (Step 4 Task B): the canonical Store writes
	// ResidentID=uuid.Nil today, so the production adapter (Task 8)
	// returns an empty slice until kb-32 ships the RecommendationID→
	// ResidentID JOIN-resolver. The fake in this package returns
	// seeded rows so tests can exercise the populated panel.
	FailedInterventionHistory(ctx context.Context, residentID uuid.UUID, since time.Time) ([]substrate_types.FailedInterventionRecord, error)

	// FailedInterventionRetrievalAvailable reports whether FIR retrieval
	// by resident-id is wired in this environment. The s2 panel uses
	// the result to render the gap-badge per Step 4 Task B's documented
	// limitation. Phase 1 production adapter returns false; the in-mem
	// fake returns true.
	FailedInterventionRetrievalAvailable() bool

	// CurrentGoalsOfCare returns the most-recent documented GoC state
	// for the resident, or nil when none is on record.
	CurrentGoalsOfCare(ctx context.Context, residentID uuid.UUID) (*substrate_types.GoalsOfCareEntry, error)

	// GoalsOfCareHistory returns the resident's GoC entries oldest-first.
	// Returns an empty slice when none exist; never nil.
	GoalsOfCareHistory(ctx context.Context, residentID uuid.UUID) ([]substrate_types.GoalsOfCareEntry, error)

	// CurrentCareIntensity returns the most-recent care-intensity entry,
	// or nil when none is on record.
	CurrentCareIntensity(ctx context.Context, residentID uuid.UUID) (*substrate_types.CareIntensityEntry, error)

	// CareIntensityHistory returns the resident's care-intensity history
	// oldest-first. Returns empty slice when none; never nil.
	CareIntensityHistory(ctx context.Context, residentID uuid.UUID) ([]substrate_types.CareIntensityEntry, error)

	// LastFamilyMeetingDate returns the most-recent family meeting date
	// for the resident, or nil when none on record. Layer 1 scope per
	// v1.0 Part 9.3 + Addendum Part 3.2 — full chronology deferred to
	// Layer 2.
	LastFamilyMeetingDate(ctx context.Context, residentID uuid.UUID) (*time.Time, error)
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

	// Task 4 additions: pending-recommendation pipeline substrate.
	packets        []substrate_types.RecommendationPacket
	assessments    map[uuid.UUID]substrate_types.AssessmentScores
	citations      map[uuid.UUID][]substrate_types.Citation
	overrides      map[uuid.UUID][]substrate_types.OverrideReason
	restraintByRes map[uuid.UUID][]substrate_types.RestraintSignal

	// Task 5 additions: FIR + GoC + care intensity + family communication.
	firByRes              map[uuid.UUID][]substrate_types.FailedInterventionRecord
	firRetrievalAvailable bool
	gocByRes              map[uuid.UUID][]substrate_types.GoalsOfCareEntry
	intensityByRes        map[uuid.UUID][]substrate_types.CareIntensityEntry
	lastFamilyMeeting     map[uuid.UUID]time.Time
}

// NewInMemorySubstrateClient returns an empty in-memory fake. Use
// WithObservations / WithAdministrations / WithPackets / WithAssessment /
// WithCitations / WithOverrides / WithRestraintSignals to seed it.
func NewInMemorySubstrateClient() *inMemorySubstrateClient {
	return &inMemorySubstrateClient{
		assessments:           map[uuid.UUID]substrate_types.AssessmentScores{},
		citations:             map[uuid.UUID][]substrate_types.Citation{},
		overrides:             map[uuid.UUID][]substrate_types.OverrideReason{},
		restraintByRes:        map[uuid.UUID][]substrate_types.RestraintSignal{},
		firByRes:              map[uuid.UUID][]substrate_types.FailedInterventionRecord{},
		firRetrievalAvailable: true, // fake exposes wired retrieval; production adapter overrides to false
		gocByRes:              map[uuid.UUID][]substrate_types.GoalsOfCareEntry{},
		intensityByRes:        map[uuid.UUID][]substrate_types.CareIntensityEntry{},
		lastFamilyMeeting:     map[uuid.UUID]time.Time{},
	}
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

// WithPackets seeds the fake with kb-32 recommendation packets. Each
// packet's SnapshotRef is used as the resident-id key for
// PendingRecommendations lookup.
func (c *inMemorySubstrateClient) WithPackets(pkts ...substrate_types.RecommendationPacket) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.packets = append(c.packets, pkts...)
	return c
}

// WithAssessment seeds an appropriateness assessment for a recommendation.
func (c *inMemorySubstrateClient) WithAssessment(recID uuid.UUID, a substrate_types.AssessmentScores) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.assessments[recID] = a
	return c
}

// WithCitations seeds the citation pin set for a recommendation.
func (c *inMemorySubstrateClient) WithCitations(recID uuid.UUID, cits ...substrate_types.Citation) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.citations[recID] = append(c.citations[recID], cits...)
	return c
}

// WithOverrides seeds the override history for a recommendation.
func (c *inMemorySubstrateClient) WithOverrides(recID uuid.UUID, ors ...substrate_types.OverrideReason) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.overrides[recID] = append(c.overrides[recID], ors...)
	return c
}

// WithRestraintSignals seeds restraint signals for a resident.
func (c *inMemorySubstrateClient) WithRestraintSignals(residentID uuid.UUID, sigs ...substrate_types.RestraintSignal) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.restraintByRes[residentID] = append(c.restraintByRes[residentID], sigs...)
	return c
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

// PendingRecommendations returns packets seeded for this resident. The
// fake uses Packet.SnapshotRef as the resident-id key (matching kb-32's
// generator.Generate behaviour of populating SnapshotRef from the
// snapshot's ResidentID).
func (c *inMemorySubstrateClient) PendingRecommendations(_ context.Context, residentID uuid.UUID) ([]substrate_types.RecommendationPacket, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]substrate_types.RecommendationPacket, 0)
	for _, p := range c.packets {
		if p.SnapshotRef == residentID {
			out = append(out, p)
		}
	}
	return out, nil
}

// RecommendationAssessment returns the assessment seeded for recID, or
// the zero AssessmentScores if none was seeded.
func (c *inMemorySubstrateClient) RecommendationAssessment(_ context.Context, recID uuid.UUID) (substrate_types.AssessmentScores, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if a, ok := c.assessments[recID]; ok {
		return a, nil
	}
	return substrate_types.AssessmentScores{}, nil
}

// RecommendationCitations returns the citations seeded for recID.
func (c *inMemorySubstrateClient) RecommendationCitations(_ context.Context, recID uuid.UUID) ([]substrate_types.Citation, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	cits := c.citations[recID]
	out := make([]substrate_types.Citation, len(cits))
	copy(out, cits)
	return out, nil
}

// RecommendationOverrides returns the override history seeded for recID,
// chronological (oldest first).
func (c *inMemorySubstrateClient) RecommendationOverrides(_ context.Context, recID uuid.UUID) ([]substrate_types.OverrideReason, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	ors := c.overrides[recID]
	out := make([]substrate_types.OverrideReason, len(ors))
	copy(out, ors)
	sort.Slice(out, func(i, j int) bool { return out[i].CapturedAt.Before(out[j].CapturedAt) })
	return out, nil
}

// ActiveRestraintSignals returns the restraint signals seeded for
// residentID. Per S2 v1.0 Part 7.3 the fake does NOT filter by
// transition criteria (advisory-only Phase 1 commitment).
func (c *inMemorySubstrateClient) ActiveRestraintSignals(_ context.Context, residentID uuid.UUID) ([]substrate_types.RestraintSignal, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	sigs := c.restraintByRes[residentID]
	out := make([]substrate_types.RestraintSignal, len(sigs))
	copy(out, sigs)
	return out, nil
}

// WithFailedInterventions seeds the fake with FIR rows for a resident.
func (c *inMemorySubstrateClient) WithFailedInterventions(residentID uuid.UUID, recs ...substrate_types.FailedInterventionRecord) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.firByRes[residentID] = append(c.firByRes[residentID], recs...)
	return c
}

// WithFIRRetrievalAvailable overrides the fake's retrieval-available
// flag so tests can exercise both the populated panel and the
// gap-badge degraded state.
func (c *inMemorySubstrateClient) WithFIRRetrievalAvailable(available bool) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.firRetrievalAvailable = available
	return c
}

// WithGoalsOfCare seeds GoC entries for a resident.
func (c *inMemorySubstrateClient) WithGoalsOfCare(residentID uuid.UUID, entries ...substrate_types.GoalsOfCareEntry) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.gocByRes[residentID] = append(c.gocByRes[residentID], entries...)
	return c
}

// WithCareIntensity seeds care-intensity entries for a resident.
func (c *inMemorySubstrateClient) WithCareIntensity(residentID uuid.UUID, entries ...substrate_types.CareIntensityEntry) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.intensityByRes[residentID] = append(c.intensityByRes[residentID], entries...)
	return c
}

// WithLastFamilyMeeting seeds a most-recent family meeting date for a
// resident.
func (c *inMemorySubstrateClient) WithLastFamilyMeeting(residentID uuid.UUID, when time.Time) *inMemorySubstrateClient {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.lastFamilyMeeting[residentID] = when
	return c
}

// FailedInterventionHistory returns FIRs for a resident with
// AttemptDate ≥ since, sorted most-recent-first. Empty slice (not nil)
// when none. When retrieval is unavailable the fake STILL returns the
// seeded rows — the panel surfaces the gap via the separate
// FailedInterventionRetrievalAvailable() check.
func (c *inMemorySubstrateClient) FailedInterventionHistory(_ context.Context, residentID uuid.UUID, since time.Time) ([]substrate_types.FailedInterventionRecord, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows := c.firByRes[residentID]
	out := make([]substrate_types.FailedInterventionRecord, 0, len(rows))
	for _, r := range rows {
		if r.AttemptDate.Before(since) {
			continue
		}
		out = append(out, r)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].AttemptDate.After(out[j].AttemptDate) })
	return out, nil
}

// FailedInterventionRetrievalAvailable reports whether resident-id
// retrieval is wired. Fake default: true; production adapter (Task 8)
// returns false until kb-32 ships the JOIN-resolver.
func (c *inMemorySubstrateClient) FailedInterventionRetrievalAvailable() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.firRetrievalAvailable
}

// CurrentGoalsOfCare returns the most-recent GoC entry (by
// EffectiveFrom) for the resident, or nil when none on record.
func (c *inMemorySubstrateClient) CurrentGoalsOfCare(_ context.Context, residentID uuid.UUID) (*substrate_types.GoalsOfCareEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows := c.gocByRes[residentID]
	if len(rows) == 0 {
		return nil, nil
	}
	idx := 0
	for i := 1; i < len(rows); i++ {
		if rows[i].EffectiveFrom.After(rows[idx].EffectiveFrom) {
			idx = i
		}
	}
	cp := rows[idx]
	return &cp, nil
}

// GoalsOfCareHistory returns all GoC entries oldest-first.
func (c *inMemorySubstrateClient) GoalsOfCareHistory(_ context.Context, residentID uuid.UUID) ([]substrate_types.GoalsOfCareEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows := c.gocByRes[residentID]
	out := make([]substrate_types.GoalsOfCareEntry, len(rows))
	copy(out, rows)
	sort.Slice(out, func(i, j int) bool { return out[i].EffectiveFrom.Before(out[j].EffectiveFrom) })
	return out, nil
}

// CurrentCareIntensity returns the most-recent care-intensity entry
// (by EffectiveDate), or nil when none on record.
func (c *inMemorySubstrateClient) CurrentCareIntensity(_ context.Context, residentID uuid.UUID) (*substrate_types.CareIntensityEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows := c.intensityByRes[residentID]
	if len(rows) == 0 {
		return nil, nil
	}
	idx := 0
	for i := 1; i < len(rows); i++ {
		if rows[i].EffectiveDate.After(rows[idx].EffectiveDate) {
			idx = i
		}
	}
	cp := rows[idx]
	return &cp, nil
}

// CareIntensityHistory returns all care-intensity entries oldest-first.
func (c *inMemorySubstrateClient) CareIntensityHistory(_ context.Context, residentID uuid.UUID) ([]substrate_types.CareIntensityEntry, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	rows := c.intensityByRes[residentID]
	out := make([]substrate_types.CareIntensityEntry, len(rows))
	copy(out, rows)
	sort.Slice(out, func(i, j int) bool { return out[i].EffectiveDate.Before(out[j].EffectiveDate) })
	return out, nil
}

// LastFamilyMeetingDate returns the most-recent meeting date, or nil
// when none on record.
func (c *inMemorySubstrateClient) LastFamilyMeetingDate(_ context.Context, residentID uuid.UUID) (*time.Time, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	t, ok := c.lastFamilyMeeting[residentID]
	if !ok {
		return nil, nil
	}
	return &t, nil
}
