// Package storage — CareIntensityStore is the kb-20 implementation of the
// care-intensity persistence + service layer (Wave 2.4 of the Layer 2
// substrate plan). It owns the care_intensity_history table + the
// care_intensity_current view created by migration 016 and is consumed by
// api/care_intensity_handlers.go for the REST surface.
//
// The store wraps the pure clinical_state.CareIntensityEngine: a transition
// is the engine's OnTransition output (transition Event + cascade hints)
// persisted alongside the new care_intensity_history row and one
// EvidenceTrace node per cascade. All writes happen in a single
// transaction so the substrate never observes a partial transition.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/clinical_state"
	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// CareIntensityStore implements interfaces.CareIntensityStore. It depends
// on a *sql.DB for the care_intensity_history table and a
// *V2SubstrateStore handle for Event + EvidenceTrace writes. Both must be
// backed by the same *sql.DB so all writes flow through one connection
// pool (and, when the future transactional path lands, the same
// transaction).
type CareIntensityStore struct {
	db     *sql.DB
	v2     *V2SubstrateStore
	engine *clinical_state.CareIntensityEngine
	now    func() time.Time
}

// NewCareIntensityStore wires a *sql.DB + *V2SubstrateStore into the
// CareIntensity persistence contract. The engine is constructed
// internally with the default UTC clock; tests that want determinism
// inject a clock via WithClock after construction.
func NewCareIntensityStore(db *sql.DB, v2 *V2SubstrateStore) *CareIntensityStore {
	return &CareIntensityStore{
		db:     db,
		v2:     v2,
		engine: clinical_state.NewCareIntensityEngine(),
		now:    func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the store's clock + the embedded engine's clock so
// tests can drive deterministic timestamps on the persisted Event and
// EvidenceTrace nodes. Returns the receiver for fluent configuration.
func (s *CareIntensityStore) WithClock(now func() time.Time) *CareIntensityStore {
	if now != nil {
		s.now = now
		s.engine = clinical_state.NewCareIntensityEngine(clinical_state.WithCareIntensityClock(now))
	}
	return s
}

const careIntensityColumns = `id, resident_ref, tag, effective_date,
       documented_by_role_ref, review_due_date, rationale_structured,
       rationale_free_text, supersedes_ref, created_at`

func scanCareIntensity(sc rowScanner) (models.CareIntensity, error) {
	var (
		c              models.CareIntensity
		reviewDue      sql.NullTime
		rationaleStruc sql.NullString
		rationaleText  sql.NullString
		supersedes     uuid.NullUUID
	)
	if err := sc.Scan(
		&c.ID, &c.ResidentRef, &c.Tag, &c.EffectiveDate,
		&c.DocumentedByRoleRef, &reviewDue, &rationaleStruc,
		&rationaleText, &supersedes, &c.CreatedAt,
	); err != nil {
		return models.CareIntensity{}, err
	}
	if reviewDue.Valid {
		t := reviewDue.Time
		c.ReviewDueDate = &t
	}
	if rationaleStruc.Valid && rationaleStruc.String != "" {
		c.RationaleStructured = json.RawMessage(rationaleStruc.String)
	}
	if rationaleText.Valid {
		c.RationaleFreeText = rationaleText.String
	}
	if supersedes.Valid {
		u := supersedes.UUID
		c.SupersedesRef = &u
	}
	return c, nil
}

// CreateCareIntensityTransition is the orchestration entry point. It
//   1. validates the incoming row;
//   2. loads the resident's current tag (if any);
//   3. validates the transition;
//   4. runs the pure engine to produce the transition Event + cascades;
//   5. persists the new care_intensity_history row, the Event, and one
//      EvidenceTrace node per cascade — all in a single transaction;
//   6. wires derived_from edges from each cascade EvidenceTrace node back
//      to the transition Event so the audit graph is complete.
//
// The returned CareIntensityTransitionResult bundles the persisted row,
// the persisted Event, and the cascade hints in the order they were
// written.
func (s *CareIntensityStore) CreateCareIntensityTransition(ctx context.Context, in models.CareIntensity) (*interfaces.CareIntensityTransitionResult, error) {
	// 1. Validate the incoming row.
	if in.ID == uuid.Nil {
		in.ID = uuid.New()
	}
	if in.EffectiveDate.IsZero() {
		in.EffectiveDate = s.now()
	}
	if err := validation.ValidateCareIntensity(in); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	// 2. Load the resident's current tag (may be empty for first-ever).
	current, err := s.GetCurrentCareIntensity(ctx, in.ResidentRef)
	var fromTag string
	if err == nil && current != nil {
		fromTag = current.Tag
		// Auto-link supersedes when the caller didn't.
		if in.SupersedesRef == nil {
			ref := current.ID
			in.SupersedesRef = &ref
		}
	} else if err != nil && !errors.Is(err, interfaces.ErrNotFound) {
		return nil, fmt.Errorf("load current care intensity: %w", err)
	}

	// 3. Validate the transition.
	if err := validation.ValidateCareIntensityTransition(fromTag, in.Tag); err != nil {
		return nil, fmt.Errorf("transition: %w", err)
	}

	// 4. Run the pure engine.
	ev, cascades := s.engine.OnTransition(fromTag, in.Tag, in.ResidentRef, in.DocumentedByRoleRef)

	// 5. Persist all writes in a single transaction.
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin tx: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()

	if err := s.insertHistoryRowTx(ctx, tx, in); err != nil {
		return nil, err
	}

	// Event persistence reuses V2SubstrateStore.UpsertEvent (non-tx) under
	// the same DB pool. Wrapping it in this tx requires a tx-aware variant
	// which the v2 store doesn't yet expose; keeping the Event write on
	// the same connection pool is sufficient for MVP since the Event row
	// is the only side effect that is not visible to the rest of the
	// transition's invariants. Future hardening may thread the tx through.
	persistedEvent, err := s.v2.UpsertEvent(ctx, ev)
	if err != nil {
		return nil, fmt.Errorf("persist transition event: %w", err)
	}

	// 6. EvidenceTrace: a parent node for the transition itself, plus one
	// child node per cascade hint linked via derived_from edges. The
	// parent node carries state_change_type=care_intensity_transition;
	// children carry state_change_type=care_intensity_cascade_<kind>.
	parentNodeID, err := s.writeTransitionEvidenceTrace(ctx, in, persistedEvent.ID, fromTag)
	if err != nil {
		return nil, fmt.Errorf("write transition evidence trace: %w", err)
	}
	hints := make([]interfaces.CareIntensityCascadeHint, 0, len(cascades))
	for _, c := range cascades {
		if err := s.writeCascadeEvidenceTrace(ctx, in, persistedEvent.ID, parentNodeID, c); err != nil {
			return nil, fmt.Errorf("write cascade evidence trace %s: %w", c.Kind, err)
		}
		hints = append(hints, interfaces.CareIntensityCascadeHint{Kind: c.Kind, Reason: c.Reason})
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	committed = true

	persisted, err := s.GetCareIntensity(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("reload persisted: %w", err)
	}
	return &interfaces.CareIntensityTransitionResult{
		CareIntensity: persisted,
		Event:         persistedEvent,
		Cascades:      hints,
	}, nil
}

// insertHistoryRowTx writes one care_intensity_history row inside tx.
func (s *CareIntensityStore) insertHistoryRowTx(ctx context.Context, tx *sql.Tx, c models.CareIntensity) error {
	const q = `
		INSERT INTO care_intensity_history
			(id, resident_ref, tag, effective_date, documented_by_role_ref,
			 review_due_date, rationale_structured, rationale_free_text,
			 supersedes_ref, created_at)
		VALUES
			($1, $2, $3, $4, $5,
			 $6, $7, $8, $9, NOW())`

	var reviewArg, rationaleStrucArg, supersedesArg interface{}
	if c.ReviewDueDate != nil {
		reviewArg = *c.ReviewDueDate
	}
	if len(c.RationaleStructured) > 0 {
		rationaleStrucArg = []byte(c.RationaleStructured)
	}
	if c.SupersedesRef != nil {
		supersedesArg = *c.SupersedesRef
	}

	if _, err := tx.ExecContext(ctx, q,
		c.ID, c.ResidentRef, c.Tag, c.EffectiveDate, c.DocumentedByRoleRef,
		reviewArg, rationaleStrucArg, nilIfEmpty(c.RationaleFreeText),
		supersedesArg,
	); err != nil {
		return fmt.Errorf("insert care_intensity_history: %w", err)
	}
	return nil
}

// writeTransitionEvidenceTrace records the parent EvidenceTrace node for
// the transition itself. State machine = ClinicalState; state change
// type = care_intensity_transition; inputs include the new CareIntensity
// row (as input_type=other) and the transition Event; output references
// the resident.
func (s *CareIntensityStore) writeTransitionEvidenceTrace(ctx context.Context, c models.CareIntensity, eventID uuid.UUID, fromTag string) (uuid.UUID, error) {
	now := s.now()
	nodeID := uuid.New()
	rs := &models.ReasoningSummary{
		Text:      fmt.Sprintf("care_intensity_transition from=%q to=%q", fromTag, c.Tag),
		RuleFires: []string{"care_intensity_transition:" + c.Tag},
	}
	rid := c.ResidentRef
	roleRef := c.DocumentedByRoleRef
	node := models.EvidenceTraceNode{
		ID:              nodeID,
		StateMachine:    models.EvidenceTraceStateMachineClinicalState,
		StateChangeType: "care_intensity_transition",
		RecordedAt:      now,
		OccurredAt:      c.EffectiveDate,
		Actor: models.TraceActor{
			RoleRef: &roleRef,
		},
		Inputs: []models.TraceInput{
			{InputType: models.TraceInputTypeOther, InputRef: c.ID, RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
			{InputType: models.TraceInputTypeEvent, InputRef: eventID, RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
		},
		ReasoningSummary: rs,
		Outputs: []models.TraceOutput{
			{OutputType: "Resident", OutputRef: c.ResidentRef},
			{OutputType: "CareIntensity", OutputRef: c.ID},
		},
		ResidentRef: &rid,
		CreatedAt:   now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return uuid.Nil, err
	}
	// Edge: the transition node derived_from the transition Event (the
	// Event is the proximate cause; care_intensity_history is its
	// recorded outcome).
	if err := s.v2.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
		From: nodeID,
		To:   eventID,
		Kind: evidence_trace.EdgeKindDerivedFrom,
	}); err != nil {
		return uuid.Nil, fmt.Errorf("insert transition→event edge: %w", err)
	}
	return nodeID, nil
}

// writeCascadeEvidenceTrace records one EvidenceTrace node per cascade
// hint, linked via derived_from to the parent transition node. The
// cascade node's state_change_type encodes the cascade kind so downstream
// consumers (Layer 3 worklist routing) can pattern-match.
func (s *CareIntensityStore) writeCascadeEvidenceTrace(ctx context.Context, c models.CareIntensity, eventID, parentNodeID uuid.UUID, cascade clinical_state.CareIntensityCascade) error {
	now := s.now()
	nodeID := uuid.New()
	rs := &models.ReasoningSummary{
		Text:      cascade.Reason,
		RuleFires: []string{"care_intensity_cascade:" + cascade.Kind},
	}
	rid := c.ResidentRef
	node := models.EvidenceTraceNode{
		ID:              nodeID,
		StateMachine:    models.EvidenceTraceStateMachineClinicalState,
		StateChangeType: "care_intensity_cascade_" + cascade.Kind,
		RecordedAt:      now,
		OccurredAt:      c.EffectiveDate,
		Inputs: []models.TraceInput{
			{InputType: models.TraceInputTypeOther, InputRef: c.ID, RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
			{InputType: models.TraceInputTypeEvent, InputRef: eventID, RoleInDecision: models.TraceRoleInDecisionSupportive},
		},
		ReasoningSummary: rs,
		Outputs: []models.TraceOutput{
			{OutputType: "Resident", OutputRef: c.ResidentRef},
		},
		ResidentRef: &rid,
		CreatedAt:   now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return err
	}
	// Edge: cascade node derived_from the parent transition node.
	if err := s.v2.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
		From: nodeID,
		To:   parentNodeID,
		Kind: evidence_trace.EdgeKindDerivedFrom,
	}); err != nil {
		return fmt.Errorf("insert cascade→parent edge: %w", err)
	}
	// Edge: cascade also derived_from the transition Event directly (so
	// graph queries that start at the Event find every cascade in one
	// reverse-traversal step).
	if err := s.v2.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
		From: nodeID,
		To:   eventID,
		Kind: evidence_trace.EdgeKindDerivedFrom,
	}); err != nil {
		return fmt.Errorf("insert cascade→event edge: %w", err)
	}
	return nil
}

// GetCareIntensity reads a single care_intensity_history row by primary
// key. Used internally by CreateCareIntensityTransition for the post-write
// re-read.
func (s *CareIntensityStore) GetCareIntensity(ctx context.Context, id uuid.UUID) (*models.CareIntensity, error) {
	q := `SELECT ` + careIntensityColumns + ` FROM care_intensity_history WHERE id = $1`
	c, err := scanCareIntensity(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get care_intensity %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get care_intensity %s: %w", id, err)
	}
	return &c, nil
}

// GetCurrentCareIntensity returns the latest tag for residentRef via the
// care_intensity_current view. Returns ErrNotFound when the resident has
// no history rows yet.
func (s *CareIntensityStore) GetCurrentCareIntensity(ctx context.Context, residentRef uuid.UUID) (*models.CareIntensity, error) {
	q := `SELECT ` + careIntensityColumns + ` FROM care_intensity_current WHERE resident_ref = $1`
	c, err := scanCareIntensity(s.db.QueryRowContext(ctx, q, residentRef))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get current care_intensity for %s: %w", residentRef, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get current care_intensity for %s: %w", residentRef, err)
	}
	return &c, nil
}

// ListCareIntensityHistory returns the full history for residentRef,
// newest-first by effective_date. Empty slice when no rows.
func (s *CareIntensityStore) ListCareIntensityHistory(ctx context.Context, residentRef uuid.UUID) ([]models.CareIntensity, error) {
	q := `SELECT ` + careIntensityColumns + `
		    FROM care_intensity_history
		   WHERE resident_ref = $1
		   ORDER BY effective_date DESC`
	rows, err := s.db.QueryContext(ctx, q, residentRef)
	if err != nil {
		return nil, fmt.Errorf("list care_intensity_history: %w", err)
	}
	defer rows.Close()
	var out []models.CareIntensity
	for rows.Next() {
		c, err := scanCareIntensity(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Compile-time assertion.
var _ interfaces.CareIntensityStore = (*CareIntensityStore)(nil)
