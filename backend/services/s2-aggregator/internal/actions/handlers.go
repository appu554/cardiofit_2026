package actions

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/cardiofit/s2-aggregator/internal/aggregation"
	"github.com/google/uuid"
)

// ActionStore is the persistence contract for the s2-aggregator-local
// pharmacist_actions audit table. Task 8 wires a Postgres-backed
// implementation; in-package tests use InMemoryActionStore.
type ActionStore interface {
	Record(ctx context.Context, req ActionRequest, ackID uuid.UUID) error
}

// OverrideForwarder forwards override actions to kb-32's override
// store (Phase 2-completion Task 5 endpoint POST
// /v1/craft/override/:recommendation_id). Decoupling via interface
// keeps s2-aggregator free of an HTTP dependency on kb-32 inside the
// in-process path; the production HTTP adapter lands with Task 8.
type OverrideForwarder interface {
	Forward(ctx context.Context, req ActionRequest) error
}

// Handler dispatches the eleven pharmacist actions. It is the single
// ingress point used by the (Task 8) HTTP handlers and by the
// in-process test harness.
type Handler struct {
	store             ActionStore
	sessions          SessionStore
	overrideForwarder OverrideForwarder
	viewBuilder       aggregation.S2ViewBuilder
}

// NewHandler returns a Handler wired with the supplied dependencies.
// All four collaborators are required; nil values panic at construction
// rather than at first call so wiring bugs surface at boot.
func NewHandler(
	store ActionStore,
	sessions SessionStore,
	overrideForwarder OverrideForwarder,
	viewBuilder aggregation.S2ViewBuilder,
) *Handler {
	if store == nil || sessions == nil || overrideForwarder == nil || viewBuilder == nil {
		panic("actions.NewHandler: all dependencies must be non-nil")
	}
	return &Handler{
		store:             store,
		sessions:          sessions,
		overrideForwarder: overrideForwarder,
		viewBuilder:       viewBuilder,
	}
}

// Execute runs the supplied ActionRequest through reasoning validation,
// per-action side effects, and the audit/session bookkeeping common to
// all eleven actions.
//
// Per-action behaviour:
//
//   - ActionOverride: forwards to kb-32 via OverrideForwarder in addition
//     to local store.Record. The local row is the s2-aggregator audit
//     trail; kb-32 is the system of record for override taxonomy.
//
//   - ActionOpenComplexWorkspace: invokes
//     S2ViewBuilder.EscalateToLayer(1→3, req). In Phase 1 Layer 3
//     returns the "not yet implemented" sentinel per Addendum Part 6
//     content-deferral; that sentinel is propagated as the Execute
//     return error so the caller learns the escalation was unfulfillable
//     while the audit row is still written.
//
//   - ActionInvokeSafetyCriticalBypass: local store.Record only (the
//     audit-prioritised handling is a function of how the audit row is
//     consumed downstream, not of the recording path itself).
//
//   - All others: local store.Record only.
//
// Session bookkeeping (RecordActionInSession) runs on every successful
// validation regardless of which action ran, so session ActionCount
// reflects the full pharmacist activity.
func (h *Handler) Execute(ctx context.Context, req ActionRequest) (ActionAcknowledgment, error) {
	if err := ValidateReasoning(req); err != nil {
		return ActionAcknowledgment{}, err
	}

	if req.Timestamp.IsZero() {
		req.Timestamp = time.Now().UTC()
	}

	// For override, normalize the dual-vocab codes onto req so the row
	// persisted to the local audit table is always canonical.
	if req.Action == ActionOverride {
		snake, short, err := NormalizeOverrideCodes(req.OverrideReasonCode, req.OverrideReasonCodeShort)
		if err != nil {
			return ActionAcknowledgment{}, err
		}
		req.OverrideReasonCode = snake
		req.OverrideReasonCodeShort = short
	}

	ack := ActionAcknowledgment{
		ActionID:     uuid.New(),
		AcceptedAt:   time.Now().UTC(),
		AuditTraceID: uuid.New(), // Task 7 wires EvidenceTrace; placeholder for now.
	}

	if err := h.store.Record(ctx, req, ack.ActionID); err != nil {
		return ActionAcknowledgment{}, err
	}
	if err := h.sessions.RecordActionInSession(ctx, req.SessionID, req.Action); err != nil {
		return ActionAcknowledgment{}, err
	}

	// Per-action side effects after the audit row + session counter are
	// safely persisted, so a downstream failure does not lose the audit.
	switch req.Action {
	case ActionOverride:
		if err := h.overrideForwarder.Forward(ctx, req); err != nil {
			return ack, err
		}
	case ActionOpenComplexWorkspace:
		workspaceReq := aggregation.WorkspaceRequest{
			ResidentID:   req.ResidentID,
			PharmacistID: req.PharmacistID,
			SessionID:    req.SessionID,
			AsOf:         req.Timestamp,
		}
		if _, err := h.viewBuilder.EscalateToLayer(ctx, 1, 3, workspaceReq); err != nil {
			// Propagate the Addendum Part 6 deferral sentinel verbatim;
			// the audit row is already written.
			return ack, err
		}
	}

	return ack, nil
}

// InMemoryActionStore is a test-facing ActionStore that retains the
// recorded rows so handler tests can assert on them.
type InMemoryActionStore struct {
	mu   sync.Mutex
	rows []recordedAction
}

type recordedAction struct {
	Req   ActionRequest
	AckID uuid.UUID
}

// NewInMemoryActionStore returns an empty in-memory store.
func NewInMemoryActionStore() *InMemoryActionStore {
	return &InMemoryActionStore{}
}

// Record appends the action to the in-memory log.
func (s *InMemoryActionStore) Record(_ context.Context, req ActionRequest, ackID uuid.UUID) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows = append(s.rows, recordedAction{Req: req, AckID: ackID})
	return nil
}

// Count returns the number of rows recorded.
func (s *InMemoryActionStore) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.rows)
}

// Last returns the most-recently-recorded row, or an empty record + false
// when the store is empty.
func (s *InMemoryActionStore) Last() (ActionRequest, uuid.UUID, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.rows) == 0 {
		return ActionRequest{}, uuid.Nil, false
	}
	r := s.rows[len(s.rows)-1]
	return r.Req, r.AckID, true
}

// InMemoryOverrideForwarder is a test-facing OverrideForwarder that
// records forwarded payloads so handler tests can verify override
// actions did fan out to kb-32.
type InMemoryOverrideForwarder struct {
	mu        sync.Mutex
	forwarded []ActionRequest
	failNext  bool
}

// NewInMemoryOverrideForwarder returns an empty in-memory forwarder.
func NewInMemoryOverrideForwarder() *InMemoryOverrideForwarder {
	return &InMemoryOverrideForwarder{}
}

// Forward records the override forward.
func (f *InMemoryOverrideForwarder) Forward(_ context.Context, req ActionRequest) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.failNext {
		f.failNext = false
		return errors.New("simulated forward failure")
	}
	f.forwarded = append(f.forwarded, req)
	return nil
}

// FailNext arms the forwarder to fail on the next call (test helper).
func (f *InMemoryOverrideForwarder) FailNext() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.failNext = true
}

// Count returns the number of successful forwards recorded.
func (f *InMemoryOverrideForwarder) Count() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.forwarded)
}
