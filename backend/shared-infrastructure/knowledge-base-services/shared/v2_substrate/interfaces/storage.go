// Package interfaces declares storage and transport contracts for the v2
// substrate. The canonical KB (kb-20 for actor entities) implements these
// interfaces; other KBs use them via clients.
package interfaces

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// ErrNotFound is returned by stores when a requested entity does not exist.
// Handlers should check errors.Is(err, ErrNotFound) to choose 404 vs 500.
var ErrNotFound = errors.New("v2_substrate: entity not found")

// ResidentStore is the canonical storage contract for Resident entities.
// kb-20-patient-profile is the only KB expected to implement this.
type ResidentStore interface {
	GetResident(ctx context.Context, id uuid.UUID) (*models.Resident, error)
	UpsertResident(ctx context.Context, r models.Resident) (*models.Resident, error)
	// ListResidentsByFacility returns residents at the given facility, paginated.
	// limit must be > 0 (caller's responsibility); offset >= 0. The implementation
	// may apply a maximum cap (e.g. 1000) but caller should not rely on that.
	ListResidentsByFacility(ctx context.Context, facilityID uuid.UUID, limit, offset int) ([]models.Resident, error)
}

// PersonStore is the canonical storage contract for Person entities.
type PersonStore interface {
	GetPerson(ctx context.Context, id uuid.UUID) (*models.Person, error)
	UpsertPerson(ctx context.Context, p models.Person) (*models.Person, error)
	GetPersonByHPII(ctx context.Context, hpii string) (*models.Person, error)
}

// RoleStore is the canonical storage contract for Role entities.
type RoleStore interface {
	GetRole(ctx context.Context, id uuid.UUID) (*models.Role, error)
	UpsertRole(ctx context.Context, r models.Role) (*models.Role, error)
	ListRolesByPerson(ctx context.Context, personID uuid.UUID) ([]models.Role, error)
	// ListActiveRolesByPersonAndFacility returns only roles where ValidFrom <= now <= ValidTo (or ValidTo is nil)
	// and (FacilityID is nil OR FacilityID == facilityID). Used by the future Authorisation evaluator.
	ListActiveRolesByPersonAndFacility(ctx context.Context, personID uuid.UUID, facilityID uuid.UUID) ([]models.Role, error)
}

// MedicineUseStore is the canonical storage contract for MedicineUse entities.
// kb-20-patient-profile is the only KB expected to implement this.
type MedicineUseStore interface {
	GetMedicineUse(ctx context.Context, id uuid.UUID) (*models.MedicineUse, error)
	UpsertMedicineUse(ctx context.Context, m models.MedicineUse) (*models.MedicineUse, error)
	ListMedicineUsesByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.MedicineUse, error)
}

// ObservationStore is the canonical storage contract for Observation entities.
// kb-20-patient-profile is the only KB expected to implement this. List
// methods take limit/offset; the implementation may apply a maximum cap
// (e.g. 1000) but caller should not rely on that.
//
// Implementations of UpsertObservation MUST compute Delta before insert via
// shared/v2_substrate/delta.ComputeDelta with an injected BaselineProvider;
// when the provider returns delta.ErrNoBaseline (or Value is nil or
// Kind=behavioural), the resulting Delta.DirectionalFlag must be
// DeltaFlagNoBaseline.
type ObservationStore interface {
	GetObservation(ctx context.Context, id uuid.UUID) (*models.Observation, error)
	UpsertObservation(ctx context.Context, o models.Observation) (*models.Observation, error)
	ListObservationsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Observation, error)
	ListObservationsByResidentAndKind(ctx context.Context, residentID uuid.UUID, kind string, limit, offset int) ([]models.Observation, error)
}

// EvidenceTraceStore is the canonical storage contract for EvidenceTrace
// nodes + edges. kb-20-patient-profile is the only KB expected to
// implement this. Per Layer 2 doc §1.6 — the architectural moat.
//
// Forward and backward traversal MUST be supported from day 1
// (Recommendation 3 of Part 7). Implementations should use a recursive
// CTE (or equivalent) over the edges table; depth-cap traversal is
// non-negotiable to prevent runaway queries.
type EvidenceTraceStore interface {
	UpsertEvidenceTraceNode(ctx context.Context, n models.EvidenceTraceNode) (*models.EvidenceTraceNode, error)
	GetEvidenceTraceNode(ctx context.Context, id uuid.UUID) (*models.EvidenceTraceNode, error)
	InsertEvidenceTraceEdge(ctx context.Context, e evidence_trace.Edge) error
	// TraceForward returns the distinct EvidenceTrace nodes reachable from
	// startNode by following outgoing edges, capped at maxDepth hops.
	TraceForward(ctx context.Context, startNode uuid.UUID, maxDepth int) ([]models.EvidenceTraceNode, error)
	// TraceBackward is the symmetric reverse traversal: nodes reachable by
	// following incoming edges (ancestors), capped at maxDepth hops.
	TraceBackward(ctx context.Context, startNode uuid.UUID, maxDepth int) ([]models.EvidenceTraceNode, error)
}

// IdentityMapping is the persistence-layer row for an
// identifier-kind/value -> Resident mapping. The pure matcher package
// (shared/v2_substrate/identity) deals in MatchResult; the storage
// layer persists the chosen mapping as one of these rows so future
// inbound identifiers reuse the same Resident.
//
// IdentifierKind is constrained at the DB level to a closed set
// ({ihi, medicare, dva, facility_internal, hospital_mrn,
// dispensing_pharmacy, gp_system}); see migration 010.
type IdentityMapping struct {
	ID              uuid.UUID  `json:"id"`
	IdentifierKind  string     `json:"identifier_kind"`
	IdentifierValue string     `json:"identifier_value"`
	ResidentRef     uuid.UUID  `json:"resident_ref"`
	Confidence      string     `json:"confidence"` // high|medium|low
	MatchPath       string     `json:"match_path"`
	Source          string     `json:"source"`
	VerifiedBy      *uuid.UUID `json:"verified_by,omitempty"`
	VerifiedAt      *time.Time `json:"verified_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
}

// IdentityMappingStore is the canonical storage contract for the
// identity_mappings table. kb-20-patient-profile is the only KB
// expected to implement this.
type IdentityMappingStore interface {
	// InsertIdentityMapping writes (or updates by the unique key
	// (identifier_kind, identifier_value, resident_ref)) one mapping.
	// Returns the persisted row.
	InsertIdentityMapping(ctx context.Context, m IdentityMapping) (*IdentityMapping, error)
	// ListIdentityMappingsByResident returns every mapping pointing at
	// resident_ref, newest-first. Used by the manual-override re-route
	// to find rows that need re-pointing.
	ListIdentityMappingsByResident(ctx context.Context, residentRef uuid.UUID) ([]IdentityMapping, error)
	// ReassignIdentityMappingsByResidentSince repoints every mapping
	// whose resident_ref == fromRef AND created_at >= since onto toRef.
	// Returns the count of rows affected. Used by ResolveIdentityReview
	// for the post-hoc re-routing requirement at Layer 2 §3.3.
	ReassignIdentityMappingsByResidentSince(ctx context.Context, fromRef, toRef uuid.UUID, since time.Time) (int, error)
}

// IdentityReviewQueueEntry is the persistence-layer row for the
// identity_review_queue table. Low-confidence and no-match decisions
// land here pending human verification.
type IdentityReviewQueueEntry struct {
	ID                   uuid.UUID       `json:"id"`
	IncomingIdentifier   json.RawMessage `json:"incoming_identifier"` // serialized identity.IncomingIdentifier
	CandidateResidentRefs []uuid.UUID    `json:"candidate_resident_refs"`
	BestCandidate        *uuid.UUID      `json:"best_candidate,omitempty"`
	BestDistance         *int            `json:"best_distance,omitempty"`
	MatchPath            string          `json:"match_path"`
	Confidence           string          `json:"confidence"` // low|none
	Source               string          `json:"source"`
	Status               string          `json:"status"` // pending|resolved|rejected
	ResolvedResidentRef  *uuid.UUID      `json:"resolved_resident_ref,omitempty"`
	ResolvedBy           *uuid.UUID      `json:"resolved_by,omitempty"`
	ResolvedAt           *time.Time      `json:"resolved_at,omitempty"`
	ResolutionNote       string          `json:"resolution_note,omitempty"`
	EvidenceTraceNodeRef *uuid.UUID      `json:"evidence_trace_node_ref,omitempty"`
	CreatedAt            time.Time       `json:"created_at"`
}

// IdentityReviewQueueStore is the canonical storage contract for the
// identity_review_queue table. kb-20-patient-profile is the only KB
// expected to implement this.
type IdentityReviewQueueStore interface {
	// InsertIdentityReviewQueueEntry creates a new pending entry. The
	// caller is responsible for setting Confidence, MatchPath,
	// IncomingIdentifier, BestCandidate/BestDistance/Candidates, and
	// EvidenceTraceNodeRef before calling.
	InsertIdentityReviewQueueEntry(ctx context.Context, e IdentityReviewQueueEntry) (*IdentityReviewQueueEntry, error)
	// GetIdentityReviewQueueEntry reads one entry by id.
	GetIdentityReviewQueueEntry(ctx context.Context, id uuid.UUID) (*IdentityReviewQueueEntry, error)
	// ListIdentityReviewQueue paginates entries filtered by status
	// (empty string -> any status), newest-first.
	ListIdentityReviewQueue(ctx context.Context, status string, limit, offset int) ([]IdentityReviewQueueEntry, error)
	// UpdateIdentityReviewQueueEntryResolution marks an entry resolved
	// (or rejected when resolvedRef == uuid.Nil) and records the
	// reviewer + note + resolved_at = NOW(). Returns the updated row.
	UpdateIdentityReviewQueueEntryResolution(ctx context.Context, id uuid.UUID, status string, resolvedRef *uuid.UUID, resolvedBy uuid.UUID, note string) (*IdentityReviewQueueEntry, error)
}

// ActiveConcernStore is the canonical storage contract for ActiveConcern
// entities. kb-20-patient-profile is the only KB expected to implement
// this. Per Layer 2 doc §2.3 (Wave 2.3) — open clinical questions that
// gate downstream rule firing.
//
// Status transitions are validated at the storage boundary: implementers
// MUST reject UpdateResolution calls whose source status is terminal
// (resolved_stop_criteria, escalated, expired_unresolved). Use
// validation.ValidateActiveConcernResolutionTransition for the legal-
// transitions check.
type ActiveConcernStore interface {
	// CreateActiveConcern inserts a new row. Returns the persisted entity.
	// Implementations must reject inputs that fail
	// validation.ValidateActiveConcern.
	CreateActiveConcern(ctx context.Context, c models.ActiveConcern) (*models.ActiveConcern, error)
	// GetActiveConcern reads a single ActiveConcern by primary key.
	GetActiveConcern(ctx context.Context, id uuid.UUID) (*models.ActiveConcern, error)
	// ListActiveConcernsByResident returns all concerns for a resident,
	// optionally filtered by resolution_status (empty string => any),
	// newest-first by started_at.
	ListActiveConcernsByResident(ctx context.Context, residentID uuid.UUID, status string) ([]models.ActiveConcern, error)
	// ListActiveByResidentAndType returns the open concerns for a resident
	// matching any of the supplied concern types. Used by the Wave 2.3
	// baseline-exclusion query path.
	ListActiveByResidentAndType(ctx context.Context, residentID uuid.UUID, types []string) ([]models.ActiveConcern, error)
	// ListExpiringConcerns returns open concerns whose
	// expected_resolution_at < now() + within. Pass within=0 for
	// already-expired concerns. Ordered by expected_resolution_at ASC
	// (most-overdue first) so the cron can prioritise.
	ListExpiringConcerns(ctx context.Context, within time.Duration) ([]models.ActiveConcern, error)
	// UpdateResolution transitions an ActiveConcern from open to a terminal
	// status. The implementation MUST reject illegal transitions per
	// validation.ValidateActiveConcernResolutionTransition.
	UpdateResolution(ctx context.Context, id uuid.UUID, status string, resolvedAt time.Time, evidenceTraceRef *uuid.UUID) (*models.ActiveConcern, error)
}

// ConcernTriggerLookupStore is the storage-layer view of the
// concern_type_triggers seed table (migration 015). Mirrors
// clinical_state.ConcernTriggerLookup but lives in the interfaces package
// to avoid circular imports between storage and clinical_state.
//
// kb-20-patient-profile implements this and adapts to
// clinical_state.ConcernTriggerLookup at engine wiring time.
type ConcernTriggerLookupStore interface {
	LookupConcernTriggersByEventType(ctx context.Context, eventType string) ([]ConcernTriggerEntry, error)
	LookupConcernTriggersByMedATC(ctx context.Context, atc, intent string) ([]ConcernTriggerEntry, error)
}

// ConcernTriggerEntry is a wire-shape mirror of
// clinical_state.TriggerEntry (kept in interfaces to avoid the import
// cycle).
type ConcernTriggerEntry struct {
	ConcernType        string
	DefaultWindowHours int
}

// CareIntensityStore is the canonical storage contract for CareIntensity
// entities (Wave 2.4 of Layer 2 substrate plan; Layer 2 doc §2.4).
// kb-20-patient-profile is the only KB expected to implement this.
//
// The history is append-only — never UPDATE rows. New transitions are
// recorded via fresh INSERT calls; the latest row by EffectiveDate per
// ResidentRef is the current tag (queried via the care_intensity_current
// view).
//
// CreateCareIntensityTransition is the orchestration entry point: it
// runs the pure clinical_state.CareIntensityEngine, persists the new
// CareIntensity row, the transition Event, and one EvidenceTrace node
// per cascade (linked via derived_from edges to the transition Event's
// EvidenceTrace node). Implementations MUST run all writes in a single
// transaction so the substrate never observes a partial transition.
type CareIntensityStore interface {
	// CreateCareIntensityTransition records a transition from the resident's
	// current tag (loaded inside the call) to `incoming.Tag`. Returns the
	// persisted CareIntensity row, the persisted transition Event, and the
	// cascade hints the engine produced (in the same order the EvidenceTrace
	// nodes were written).
	CreateCareIntensityTransition(ctx context.Context, incoming models.CareIntensity) (*CareIntensityTransitionResult, error)
	// GetCurrentCareIntensity returns the latest tag for residentRef, or
	// ErrNotFound when the resident has no history rows yet.
	GetCurrentCareIntensity(ctx context.Context, residentRef uuid.UUID) (*models.CareIntensity, error)
	// ListCareIntensityHistory returns the full history for residentRef,
	// ordered by EffectiveDate DESC (newest first). Empty slice when the
	// resident has no history rows.
	ListCareIntensityHistory(ctx context.Context, residentRef uuid.UUID) ([]models.CareIntensity, error)
}

// CareIntensityTransitionResult is the return shape from
// CareIntensityStore.CreateCareIntensityTransition. It bundles the
// persisted CareIntensity row, the transition Event, and the cascade hints
// so callers can return all three to the REST client in a single payload.
type CareIntensityTransitionResult struct {
	CareIntensity *models.CareIntensity `json:"care_intensity"`
	Event         *models.Event         `json:"event"`
	Cascades      []CareIntensityCascadeHint `json:"cascades"`
}

// CareIntensityCascadeHint mirrors clinical_state.CareIntensityCascade in
// the interfaces package to avoid an import cycle (clinical_state already
// imports models; interfaces is upstream of both clinical_state and the
// kb-20 storage layer that produces these hints).
type CareIntensityCascadeHint struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

// CapacityAssessmentStore is the canonical storage contract for
// CapacityAssessment entities (Wave 2.5 of Layer 2 substrate plan;
// Layer 2 doc §2.5). kb-20-patient-profile is the only KB expected to
// implement this.
//
// The history is append-only — never UPDATE rows. New assessments are
// recorded via fresh INSERT calls; the latest row by AssessedAt per
// (ResidentRef, Domain) is the current assessment for that domain
// (queried via the capacity_current view).
//
// CreateCapacityAssessment is the orchestration entry point: it
// validates the incoming row, persists it, writes one EvidenceTrace
// node for the assessment, and conditionally emits an Event of type
// capacity_change when Outcome=impaired AND Domain=medical_decisions
// (the only combination that cascades to Consent in Layer 3). All
// writes happen against the same connection pool so future
// transactional hardening can wrap them in a single tx.
type CapacityAssessmentStore interface {
	// CreateCapacityAssessment persists `incoming` and (conditionally) emits
	// the capacity_change Event. Returns the persisted row, the optional
	// Event (nil for non-medical or non-impaired), and the EvidenceTrace
	// node id that was written.
	CreateCapacityAssessment(ctx context.Context, incoming models.CapacityAssessment) (*CapacityAssessmentResult, error)
	// GetCapacityAssessment reads a single row by primary key.
	GetCapacityAssessment(ctx context.Context, id uuid.UUID) (*models.CapacityAssessment, error)
	// GetCurrentCapacity returns the latest assessment for (residentRef,
	// domain), or ErrNotFound when no rows exist for that pair.
	GetCurrentCapacity(ctx context.Context, residentRef uuid.UUID, domain string) (*models.CapacityAssessment, error)
	// ListCurrentCapacityByResident returns one row per domain present for
	// residentRef (the latest by AssessedAt for each domain). Empty slice
	// when the resident has no assessments at all.
	ListCurrentCapacityByResident(ctx context.Context, residentRef uuid.UUID) ([]models.CapacityAssessment, error)
	// ListCapacityHistory returns the full history for (residentRef,
	// domain), ordered by AssessedAt DESC.
	ListCapacityHistory(ctx context.Context, residentRef uuid.UUID, domain string) ([]models.CapacityAssessment, error)
}

// CapacityAssessmentResult is the return shape from
// CapacityAssessmentStore.CreateCapacityAssessment. Event is nil when
// the assessment did not trigger a capacity_change Event (i.e. anything
// other than impaired+medical_decisions). EvidenceTraceNodeRef is always
// set — every assessment writes one EvidenceTrace node so the audit
// graph is complete.
type CapacityAssessmentResult struct {
	Assessment           *models.CapacityAssessment `json:"assessment"`
	Event                *models.Event              `json:"event,omitempty"`
	EvidenceTraceNodeRef uuid.UUID                  `json:"evidence_trace_node_ref"`
}

// EventStore is the canonical storage contract for Event entities.
// kb-20-patient-profile is the only KB expected to implement this. List
// methods take limit/offset; the implementation may apply a maximum cap
// (e.g. 1000) but caller should not rely on that.
type EventStore interface {
	GetEvent(ctx context.Context, id uuid.UUID) (*models.Event, error)
	UpsertEvent(ctx context.Context, e models.Event) (*models.Event, error)
	ListEventsByResident(ctx context.Context, residentID uuid.UUID, limit, offset int) ([]models.Event, error)
	// ListEventsByType returns events of a given event_type whose occurred_at
	// falls inside [from, to). A zero `from` means no lower bound; a zero `to`
	// means no upper bound. Results are ordered by occurred_at DESC.
	ListEventsByType(ctx context.Context, eventType string, from, to time.Time, limit, offset int) ([]models.Event, error)
}
