// Package storage — CapacityAssessmentStore is the kb-20 implementation
// of the per-domain capacity persistence + service layer (Wave 2.5 of
// the Layer 2 substrate plan; Layer 2 doc §2.5). It owns the
// capacity_assessments table + the capacity_current view created by
// migration 017 and is consumed by api/capacity_handlers.go for the
// REST surface.
//
// Service-layer behaviour (per plan):
//
//   - Every CapacityAssessment write produces one EvidenceTrace node so
//     the audit graph captures every assessment.
//   - When Outcome=impaired AND Domain=medical_decisions, the store
//     additionally emits an Event of type capacity_change AND tags the
//     EvidenceTrace node with state_machine=Consent. Layer 3's Consent
//     state machine consumes the Event to re-evaluate consent paths
//     (resident-self vs SDM-authorised).
//   - For all other (domain, outcome) combinations the EvidenceTrace
//     node is tagged with state_machine=ClinicalState; no Event is
//     emitted (informational only, does not cascade in this wave).
//
// The history is append-only — never UPDATE rows.
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// CapacityAssessmentStore implements interfaces.CapacityAssessmentStore.
// Depends on a *sql.DB for the capacity_assessments table and a
// *V2SubstrateStore handle for Event + EvidenceTrace writes. Both must
// be backed by the same *sql.DB so all writes flow through one
// connection pool.
type CapacityAssessmentStore struct {
	db  *sql.DB
	v2  *V2SubstrateStore
	now func() time.Time
}

// NewCapacityAssessmentStore wires a *sql.DB + *V2SubstrateStore into
// the CapacityAssessment persistence contract.
func NewCapacityAssessmentStore(db *sql.DB, v2 *V2SubstrateStore) *CapacityAssessmentStore {
	return &CapacityAssessmentStore{
		db:  db,
		v2:  v2,
		now: func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the store's clock so tests can drive deterministic
// timestamps on the persisted Event and EvidenceTrace nodes.
func (s *CapacityAssessmentStore) WithClock(now func() time.Time) *CapacityAssessmentStore {
	if now != nil {
		s.now = now
	}
	return s
}

const capacityAssessmentColumns = `id, resident_ref, assessed_at, assessor_role_ref,
       domain, instrument, score, outcome, duration, expected_review_date,
       rationale_structured, rationale_free_text, supersedes_ref, created_at`

func scanCapacityAssessment(sc rowScanner) (models.CapacityAssessment, error) {
	var (
		c              models.CapacityAssessment
		instrument     sql.NullString
		score          sql.NullFloat64
		expectedReview sql.NullTime
		rationaleStruc sql.NullString
		rationaleText  sql.NullString
		supersedes     uuid.NullUUID
	)
	if err := sc.Scan(
		&c.ID, &c.ResidentRef, &c.AssessedAt, &c.AssessorRoleRef,
		&c.Domain, &instrument, &score, &c.Outcome, &c.Duration, &expectedReview,
		&rationaleStruc, &rationaleText, &supersedes, &c.CreatedAt,
	); err != nil {
		return models.CapacityAssessment{}, err
	}
	if instrument.Valid {
		c.Instrument = instrument.String
	}
	if score.Valid {
		f := score.Float64
		c.Score = &f
	}
	if expectedReview.Valid {
		t := expectedReview.Time
		c.ExpectedReviewDate = &t
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

// CreateCapacityAssessment is the orchestration entry point. It:
//  1. validates the incoming row;
//  2. auto-links supersedes_ref to the prior current assessment for the
//     same (resident, domain) pair when not explicitly supplied;
//  3. inserts the capacity_assessments row;
//  4. writes the EvidenceTrace node (state_machine = Consent for
//     impaired+medical_decisions, else ClinicalState);
//  5. emits the capacity_change Event when impaired+medical_decisions;
//  6. wires a derived_from edge from the EvidenceTrace node to the Event
//     when the Event was emitted.
func (s *CapacityAssessmentStore) CreateCapacityAssessment(ctx context.Context, in models.CapacityAssessment) (*interfaces.CapacityAssessmentResult, error) {
	// 1. Validate.
	if in.ID == uuid.Nil {
		in.ID = uuid.New()
	}
	if in.AssessedAt.IsZero() {
		in.AssessedAt = s.now()
	}
	if err := validation.ValidateCapacityAssessment(in); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	// 2. Auto-link supersedes_ref.
	if in.SupersedesRef == nil {
		current, err := s.GetCurrentCapacity(ctx, in.ResidentRef, in.Domain)
		if err == nil && current != nil {
			ref := current.ID
			in.SupersedesRef = &ref
		} else if err != nil && !errors.Is(err, interfaces.ErrNotFound) {
			return nil, fmt.Errorf("load current capacity: %w", err)
		}
	}

	// 3. Insert.
	if err := s.insertCapacityRow(ctx, in); err != nil {
		return nil, err
	}

	// 4. EvidenceTrace + 5. optional Event.
	isMedicalImpaired := in.Domain == models.CapacityDomainMedical &&
		in.Outcome == models.CapacityOutcomeImpaired

	var (
		persistedEvent *models.Event
		err            error
	)
	if isMedicalImpaired {
		persistedEvent, err = s.emitCapacityChangeEvent(ctx, in)
		if err != nil {
			return nil, fmt.Errorf("emit capacity_change event: %w", err)
		}
	}

	traceNodeID, err := s.writeAssessmentEvidenceTrace(ctx, in, persistedEvent, isMedicalImpaired)
	if err != nil {
		return nil, fmt.Errorf("write evidence trace: %w", err)
	}

	persisted, err := s.GetCapacityAssessment(ctx, in.ID)
	if err != nil {
		return nil, fmt.Errorf("reload persisted: %w", err)
	}
	return &interfaces.CapacityAssessmentResult{
		Assessment:           persisted,
		Event:                persistedEvent,
		EvidenceTraceNodeRef: traceNodeID,
	}, nil
}

// insertCapacityRow writes one capacity_assessments row.
func (s *CapacityAssessmentStore) insertCapacityRow(ctx context.Context, c models.CapacityAssessment) error {
	const q = `
		INSERT INTO capacity_assessments
			(id, resident_ref, assessed_at, assessor_role_ref, domain,
			 instrument, score, outcome, duration, expected_review_date,
			 rationale_structured, rationale_free_text, supersedes_ref, created_at)
		VALUES
			($1, $2, $3, $4, $5,
			 $6, $7, $8, $9, $10,
			 $11, $12, $13, NOW())`

	var (
		instrumentArg, scoreArg, reviewArg, rationaleStrucArg, supersedesArg interface{}
	)
	if c.Instrument != "" {
		instrumentArg = c.Instrument
	}
	if c.Score != nil {
		scoreArg = *c.Score
	}
	if c.ExpectedReviewDate != nil {
		reviewArg = *c.ExpectedReviewDate
	}
	if len(c.RationaleStructured) > 0 {
		rationaleStrucArg = []byte(c.RationaleStructured)
	}
	if c.SupersedesRef != nil {
		supersedesArg = *c.SupersedesRef
	}

	if _, err := s.db.ExecContext(ctx, q,
		c.ID, c.ResidentRef, c.AssessedAt, c.AssessorRoleRef, c.Domain,
		instrumentArg, scoreArg, c.Outcome, c.Duration, reviewArg,
		rationaleStrucArg, nilIfEmpty(c.RationaleFreeText), supersedesArg,
	); err != nil {
		return fmt.Errorf("insert capacity_assessment: %w", err)
	}
	return nil
}

// emitCapacityChangeEvent persists the capacity_change Event for an
// impaired+medical_decisions assessment.
func (s *CapacityAssessmentStore) emitCapacityChangeEvent(ctx context.Context, c models.CapacityAssessment) (*models.Event, error) {
	descr, _ := json.Marshal(map[string]interface{}{
		"capacity_assessment_ref": c.ID.String(),
		"domain":                  c.Domain,
		"outcome":                 c.Outcome,
		"duration":                c.Duration,
	})
	ev := models.Event{
		ID:                    uuid.New(),
		EventType:             models.EventTypeCapacityChange,
		OccurredAt:            c.AssessedAt,
		ResidentID:            c.ResidentRef,
		ReportedByRef:         c.AssessorRoleRef,
		Severity:              models.EventSeverityModerate,
		DescriptionStructured: descr,
		DescriptionFreeText:   "Medical capacity impaired — Consent state machine should re-evaluate consent paths",
		CreatedAt:             s.now(),
		UpdatedAt:             s.now(),
	}
	persisted, err := s.v2.UpsertEvent(ctx, ev)
	if err != nil {
		return nil, err
	}
	return persisted, nil
}

// writeAssessmentEvidenceTrace records one EvidenceTrace node per
// assessment. When the assessment is impaired+medical_decisions, the
// node is tagged with state_machine=Consent and a derived_from edge is
// wired from the node to the capacity_change Event. Otherwise the node
// is tagged with state_machine=ClinicalState and no edge is wired (no
// Event was emitted).
func (s *CapacityAssessmentStore) writeAssessmentEvidenceTrace(
	ctx context.Context,
	c models.CapacityAssessment,
	ev *models.Event,
	isMedicalImpaired bool,
) (uuid.UUID, error) {
	now := s.now()
	nodeID := uuid.New()
	stateMachine := models.EvidenceTraceStateMachineClinicalState
	stateChangeType := "capacity_assessment_recorded"
	if isMedicalImpaired {
		stateMachine = models.EvidenceTraceStateMachineConsent
		stateChangeType = "capacity_change_medical_impaired"
	}
	rs := &models.ReasoningSummary{
		Text: fmt.Sprintf("capacity_assessment domain=%q outcome=%q duration=%q",
			c.Domain, c.Outcome, c.Duration),
		RuleFires: []string{"capacity_assessment:" + c.Domain + ":" + c.Outcome},
	}
	rid := c.ResidentRef
	roleRef := c.AssessorRoleRef
	inputs := []models.TraceInput{
		{
			InputType:      models.TraceInputTypeOther,
			InputRef:       c.ID,
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		},
	}
	if ev != nil {
		inputs = append(inputs, models.TraceInput{
			InputType:      models.TraceInputTypeEvent,
			InputRef:       ev.ID,
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		})
	}
	node := models.EvidenceTraceNode{
		ID:              nodeID,
		StateMachine:    stateMachine,
		StateChangeType: stateChangeType,
		RecordedAt:      now,
		OccurredAt:      c.AssessedAt,
		Actor: models.TraceActor{
			RoleRef: &roleRef,
		},
		Inputs:           inputs,
		ReasoningSummary: rs,
		Outputs: []models.TraceOutput{
			{OutputType: "Resident", OutputRef: c.ResidentRef},
			{OutputType: "CapacityAssessment", OutputRef: c.ID},
		},
		ResidentRef: &rid,
		CreatedAt:   now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return uuid.Nil, err
	}
	if ev != nil {
		// Edge: assessment node derived_from the capacity_change Event
		// (the Event is the proximate cause of the Consent re-evaluation;
		// the assessment row is the recorded outcome).
		if err := s.v2.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
			From: nodeID,
			To:   ev.ID,
			Kind: evidence_trace.EdgeKindDerivedFrom,
		}); err != nil {
			return uuid.Nil, fmt.Errorf("insert assessment→event edge: %w", err)
		}
	}
	return nodeID, nil
}

// GetCapacityAssessment reads a single capacity_assessments row by id.
func (s *CapacityAssessmentStore) GetCapacityAssessment(ctx context.Context, id uuid.UUID) (*models.CapacityAssessment, error) {
	q := `SELECT ` + capacityAssessmentColumns + ` FROM capacity_assessments WHERE id = $1`
	c, err := scanCapacityAssessment(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get capacity_assessment %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get capacity_assessment %s: %w", id, err)
	}
	return &c, nil
}

// GetCurrentCapacity returns the latest assessment for (residentRef,
// domain) via the capacity_current view. Returns ErrNotFound when no
// rows exist for that pair.
func (s *CapacityAssessmentStore) GetCurrentCapacity(ctx context.Context, residentRef uuid.UUID, domain string) (*models.CapacityAssessment, error) {
	q := `SELECT ` + capacityAssessmentColumns + ` FROM capacity_current
	      WHERE resident_ref = $1 AND domain = $2`
	c, err := scanCapacityAssessment(s.db.QueryRowContext(ctx, q, residentRef, domain))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get current capacity for %s/%s: %w", residentRef, domain, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get current capacity for %s/%s: %w", residentRef, domain, err)
	}
	return &c, nil
}

// ListCurrentCapacityByResident returns one row per domain present for
// residentRef (the latest by AssessedAt within each domain).
func (s *CapacityAssessmentStore) ListCurrentCapacityByResident(ctx context.Context, residentRef uuid.UUID) ([]models.CapacityAssessment, error) {
	q := `SELECT ` + capacityAssessmentColumns + ` FROM capacity_current
	      WHERE resident_ref = $1
	      ORDER BY domain ASC`
	rows, err := s.db.QueryContext(ctx, q, residentRef)
	if err != nil {
		return nil, fmt.Errorf("list current capacity: %w", err)
	}
	defer rows.Close()
	var out []models.CapacityAssessment
	for rows.Next() {
		c, err := scanCapacityAssessment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListCapacityHistory returns the full history for (residentRef,
// domain), newest-first by assessed_at.
func (s *CapacityAssessmentStore) ListCapacityHistory(ctx context.Context, residentRef uuid.UUID, domain string) ([]models.CapacityAssessment, error) {
	q := `SELECT ` + capacityAssessmentColumns + `
	        FROM capacity_assessments
	       WHERE resident_ref = $1 AND domain = $2
	       ORDER BY assessed_at DESC`
	rows, err := s.db.QueryContext(ctx, q, residentRef, domain)
	if err != nil {
		return nil, fmt.Errorf("list capacity_assessments history: %w", err)
	}
	defer rows.Close()
	var out []models.CapacityAssessment
	for rows.Next() {
		c, err := scanCapacityAssessment(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// Compile-time assertion.
var _ interfaces.CapacityAssessmentStore = (*CapacityAssessmentStore)(nil)
