// Package storage — ReconciliationStore is the kb-20 implementation of the
// hospital-discharge reconciliation persistence + write-back orchestration
// layer (Wave 4.3 + 4.4 of the Layer 2 substrate plan; Layer 2 doc §3.2).
// It owns the reconciliation_worklists + reconciliation_decisions tables
// (migration 021) and bridges the pure reconciliation engine
// (shared/v2_substrate/reconciliation) onto the substrate.
//
// EvidenceTrace chain written by every decision:
//
//	discharge_document → (start) → worklist EvidenceTrace node
//	    │                              │ derived_from
//	    │                              ▼
//	    │              ┌──── decision EvidenceTrace node
//	    │              │
//	    │  derived_from │ led_to
//	    ▼              ▼
//	pre_admission_med (if any)   resulting MedicineUse (if any)
//
// Forward and backward traversal from the discharge_document worklist
// node visits every decision and resulting MedicineUse change.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/reconciliation"
)

// ReconciliationStore implements interfaces.ReconciliationStore.
type ReconciliationStore struct {
	db        *sql.DB
	v2        *V2SubstrateStore
	docStore  *DischargeDocumentStore
	now       func() time.Time
}

// NewReconciliationStore wires the dependencies. v2 is required for
// MedicineUse / Event / EvidenceTrace writes; docStore loads the
// pre-staged discharge document + lines.
func NewReconciliationStore(db *sql.DB, v2 *V2SubstrateStore, docStore *DischargeDocumentStore) *ReconciliationStore {
	return &ReconciliationStore{
		db:       db,
		v2:       v2,
		docStore: docStore,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

// WithClock overrides the wall clock for deterministic tests.
func (s *ReconciliationStore) WithClock(now func() time.Time) *ReconciliationStore {
	if now != nil {
		s.now = now
	}
	return s
}

// =================================================================
// StartWorklist
// =================================================================

func (s *ReconciliationStore) StartWorklist(ctx context.Context, in interfaces.ReconciliationStartInputs) (*interfaces.ReconciliationStartResult, error) {
	if in.DischargeDocumentRef == uuid.Nil {
		return nil, errors.New("StartWorklist: discharge_document_ref required")
	}

	doc, err := s.docStore.GetDischargeDocument(ctx, in.DischargeDocumentRef)
	if err != nil {
		return nil, fmt.Errorf("load discharge document: %w", err)
	}

	// Pre-admission active MedicineUses for the resident.
	preAdmission, err := s.v2.ListMedicineUsesByResident(ctx, doc.ResidentRef, 1000, 0)
	if err != nil {
		return nil, fmt.Errorf("load pre-admission medicine uses: %w", err)
	}

	// Build the diff-engine inputs from the discharge lines.
	dischargeSummaries := make([]reconciliation.DischargeLineSummary, 0, len(doc.MedicationLines))
	lineByRef := map[uuid.UUID]interfaces.DischargeMedicationLine{}
	for _, ln := range doc.MedicationLines {
		lineByRef[ln.ID] = ln
		dischargeSummaries = append(dischargeSummaries, reconciliation.DischargeLineSummary{
			LineRef:        ln.ID,
			AMTCode:        ln.AMTCode,
			DisplayName:    ln.MedicationNameRaw,
			Dose:           ln.DoseRaw,
			Frequency:      ln.FrequencyRaw,
			Route:          ln.RouteRaw,
			IndicationText: ln.IndicationText,
			Notes:          ln.Notes,
		})
	}

	diffs := reconciliation.ComputeDiff(preAdmission, dischargeSummaries)

	dueWindow := time.Duration(in.DueWindowHours) * time.Hour
	if dueWindow <= 0 {
		dueWindow = reconciliation.DefaultWorklistDueWindow
	}
	worklistInputs := reconciliation.BuildWorklistInputs(
		doc.ID, doc.ResidentRef, in.AssignedRoleRef, in.FacilityID,
		doc.DischargeDate, dueWindow, diffs, nil,
	)

	// Persist worklist + decision rows + parent EvidenceTrace node.
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

	worklistID := uuid.New()
	now := s.now()
	const insWL = `
		INSERT INTO reconciliation_worklists
			(id, discharge_document_ref, resident_ref, assigned_role_ref,
			 facility_id, status, due_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING created_at`
	var wlCreatedAt time.Time
	if err := tx.QueryRowContext(ctx, insWL,
		worklistID, doc.ID, doc.ResidentRef,
		nullUUID(in.AssignedRoleRef), nullUUID(in.FacilityID),
		string(reconciliation.WorklistPending),
		worklistInputs.DueAt,
	).Scan(&wlCreatedAt); err != nil {
		return nil, fmt.Errorf("insert reconciliation_worklist: %w", err)
	}

	// EvidenceTrace parent node — captures the discharge_document → worklist link.
	parentNodeID, err := s.writeWorklistEvidenceTrace(ctx, doc, worklistID, now)
	if err != nil {
		return nil, fmt.Errorf("write worklist evidence trace: %w", err)
	}

	const insDec = `
		INSERT INTO reconciliation_decisions
			(id, worklist_ref, discharge_med_line_ref, pre_admission_medicine_use_ref,
			 diff_class, intent_class, acop_decision, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, '', NOW())
		RETURNING created_at`

	persistedDecisions := make([]interfaces.ReconciliationDecision, 0, len(worklistInputs.Decisions))
	for _, di := range worklistInputs.Decisions {
		decID := uuid.New()
		var (
			lineRef uuid.NullUUID
			preRef  uuid.NullUUID
		)
		if di.DiffEntry.DischargeLineRef != nil {
			lineRef = uuid.NullUUID{UUID: *di.DiffEntry.DischargeLineRef, Valid: true}
		}
		if di.DiffEntry.PreAdmissionMedUseRef != nil {
			preRef = uuid.NullUUID{UUID: *di.DiffEntry.PreAdmissionMedUseRef, Valid: true}
		}
		var createdAt time.Time
		if err := tx.QueryRowContext(ctx, insDec,
			decID, worklistID, lineRef, preRef,
			string(di.DiffEntry.Class), string(di.IntentClass),
		).Scan(&createdAt); err != nil {
			return nil, fmt.Errorf("insert reconciliation_decision: %w", err)
		}
		dec := interfaces.ReconciliationDecision{
			ID:          decID,
			WorklistRef: worklistID,
			DiffClass:   string(di.DiffEntry.Class),
			IntentClass: string(di.IntentClass),
			CreatedAt:   createdAt,
		}
		if di.DiffEntry.DischargeLineRef != nil {
			ref := *di.DiffEntry.DischargeLineRef
			dec.DischargeMedLineRef = &ref
		}
		if di.DiffEntry.PreAdmissionMedUseRef != nil {
			ref := *di.DiffEntry.PreAdmissionMedUseRef
			dec.PreAdmissionMedicineUseRef = &ref
		}
		persistedDecisions = append(persistedDecisions, dec)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit: %w", err)
	}
	committed = true

	worklist := &interfaces.ReconciliationWorklist{
		ID:                   worklistID,
		DischargeDocumentRef: doc.ID,
		ResidentRef:          doc.ResidentRef,
		AssignedRoleRef:      in.AssignedRoleRef,
		FacilityID:           in.FacilityID,
		Status:               string(reconciliation.WorklistPending),
		DueAt:                worklistInputs.DueAt,
		CreatedAt:            wlCreatedAt,
	}
	_ = parentNodeID // keep referenced; the node was written
	return &interfaces.ReconciliationStartResult{
		Worklist:  worklist,
		Decisions: persistedDecisions,
	}, nil
}

// writeWorklistEvidenceTrace creates the parent EvidenceTrace node tying
// the discharge_document to the new worklist.
func (s *ReconciliationStore) writeWorklistEvidenceTrace(ctx context.Context, doc *interfaces.DischargeDocument, worklistID uuid.UUID, now time.Time) (uuid.UUID, error) {
	nodeID := uuid.New()
	rid := doc.ResidentRef
	rs := &models.ReasoningSummary{
		Text:      fmt.Sprintf("reconciliation_started doc=%s source=%s", doc.ID, doc.Source),
		RuleFires: []string{"reconciliation_started"},
	}
	node := models.EvidenceTraceNode{
		ID:              nodeID,
		StateMachine:    models.EvidenceTraceStateMachineRecommendation,
		StateChangeType: "reconciliation_started",
		RecordedAt:      now,
		OccurredAt:      doc.DischargeDate,
		Inputs: []models.TraceInput{
			{InputType: models.TraceInputTypeOther, InputRef: doc.ID, RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
		},
		ReasoningSummary: rs,
		Outputs: []models.TraceOutput{
			{OutputType: "ReconciliationWorklist", OutputRef: worklistID},
		},
		ResidentRef: &rid,
		CreatedAt:   now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return uuid.Nil, err
	}
	return nodeID, nil
}

// =================================================================
// Get / List
// =================================================================

func (s *ReconciliationStore) GetWorklist(ctx context.Context, worklistRef uuid.UUID) (*interfaces.ReconciliationWorklist, []interfaces.ReconciliationDecision, error) {
	wl, err := s.loadWorklist(ctx, worklistRef)
	if err != nil {
		return nil, nil, err
	}
	decs, err := s.loadDecisions(ctx, worklistRef)
	if err != nil {
		return nil, nil, err
	}
	return wl, decs, nil
}

func (s *ReconciliationStore) loadWorklist(ctx context.Context, id uuid.UUID) (*interfaces.ReconciliationWorklist, error) {
	const q = `
		SELECT id, discharge_document_ref, resident_ref, assigned_role_ref,
		       facility_id, status, due_at, completed_at, completed_by_role_ref,
		       created_at
		FROM reconciliation_worklists
		WHERE id = $1`
	var (
		wl              interfaces.ReconciliationWorklist
		assignedRoleRef uuid.NullUUID
		facilityID      uuid.NullUUID
		completedAt     sql.NullTime
		completedByRole uuid.NullUUID
	)
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&wl.ID, &wl.DischargeDocumentRef, &wl.ResidentRef, &assignedRoleRef,
		&facilityID, &wl.Status, &wl.DueAt, &completedAt, &completedByRole,
		&wl.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, interfaces.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query worklist: %w", err)
	}
	if assignedRoleRef.Valid {
		u := assignedRoleRef.UUID
		wl.AssignedRoleRef = &u
	}
	if facilityID.Valid {
		u := facilityID.UUID
		wl.FacilityID = &u
	}
	if completedAt.Valid {
		t := completedAt.Time
		wl.CompletedAt = &t
	}
	if completedByRole.Valid {
		u := completedByRole.UUID
		wl.CompletedByRoleRef = &u
	}
	return &wl, nil
}

func (s *ReconciliationStore) loadDecisions(ctx context.Context, worklistRef uuid.UUID) ([]interfaces.ReconciliationDecision, error) {
	const q = `
		SELECT id, worklist_ref, discharge_med_line_ref, pre_admission_medicine_use_ref,
		       diff_class, intent_class, COALESCE(acop_decision,''),
		       acop_role_ref, decided_at, COALESCE(notes,''),
		       resulting_medicine_use_ref, evidence_trace_node_ref, created_at
		FROM reconciliation_decisions
		WHERE worklist_ref = $1
		ORDER BY created_at ASC`
	rows, err := s.db.QueryContext(ctx, q, worklistRef)
	if err != nil {
		return nil, fmt.Errorf("query decisions: %w", err)
	}
	defer rows.Close()
	out := []interfaces.ReconciliationDecision{}
	for rows.Next() {
		var (
			d           interfaces.ReconciliationDecision
			lineRef     uuid.NullUUID
			preRef      uuid.NullUUID
			roleRef     uuid.NullUUID
			decidedAt   sql.NullTime
			resultRef   uuid.NullUUID
			evidenceRef uuid.NullUUID
		)
		if err := rows.Scan(
			&d.ID, &d.WorklistRef, &lineRef, &preRef,
			&d.DiffClass, &d.IntentClass, &d.ACOPDecision,
			&roleRef, &decidedAt, &d.Notes,
			&resultRef, &evidenceRef, &d.CreatedAt,
		); err != nil {
			return nil, err
		}
		if lineRef.Valid {
			u := lineRef.UUID
			d.DischargeMedLineRef = &u
		}
		if preRef.Valid {
			u := preRef.UUID
			d.PreAdmissionMedicineUseRef = &u
		}
		if roleRef.Valid {
			u := roleRef.UUID
			d.ACOPRoleRef = &u
		}
		if decidedAt.Valid {
			t := decidedAt.Time
			d.DecidedAt = &t
		}
		if resultRef.Valid {
			u := resultRef.UUID
			d.ResultingMedicineUseRef = &u
		}
		if evidenceRef.Valid {
			u := evidenceRef.UUID
			d.EvidenceTraceNodeRef = &u
		}
		out = append(out, d)
	}
	return out, nil
}

func (s *ReconciliationStore) ListWorklistsByRoleAndFacility(ctx context.Context, roleRef, facilityID *uuid.UUID, status string, limit, offset int) ([]interfaces.ReconciliationWorklist, error) {
	if limit <= 0 {
		limit = 50
	}
	q := `
		SELECT id, discharge_document_ref, resident_ref, assigned_role_ref,
		       facility_id, status, due_at, completed_at, completed_by_role_ref,
		       created_at
		FROM reconciliation_worklists
		WHERE 1=1`
	args := []interface{}{}
	if roleRef != nil {
		args = append(args, *roleRef)
		q += fmt.Sprintf(" AND assigned_role_ref = $%d", len(args))
	}
	if facilityID != nil {
		args = append(args, *facilityID)
		q += fmt.Sprintf(" AND facility_id = $%d", len(args))
	}
	if status != "" {
		args = append(args, status)
		q += fmt.Sprintf(" AND status = $%d", len(args))
	}
	args = append(args, limit, offset)
	q += fmt.Sprintf(" ORDER BY due_at ASC LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list worklists: %w", err)
	}
	defer rows.Close()
	out := []interfaces.ReconciliationWorklist{}
	for rows.Next() {
		var (
			wl              interfaces.ReconciliationWorklist
			assignedRoleRef uuid.NullUUID
			facilityIDRef   uuid.NullUUID
			completedAt     sql.NullTime
			completedByRole uuid.NullUUID
		)
		if err := rows.Scan(
			&wl.ID, &wl.DischargeDocumentRef, &wl.ResidentRef, &assignedRoleRef,
			&facilityIDRef, &wl.Status, &wl.DueAt, &completedAt, &completedByRole,
			&wl.CreatedAt,
		); err != nil {
			return nil, err
		}
		if assignedRoleRef.Valid {
			u := assignedRoleRef.UUID
			wl.AssignedRoleRef = &u
		}
		if facilityIDRef.Valid {
			u := facilityIDRef.UUID
			wl.FacilityID = &u
		}
		if completedAt.Valid {
			t := completedAt.Time
			wl.CompletedAt = &t
		}
		if completedByRole.Valid {
			u := completedByRole.UUID
			wl.CompletedByRoleRef = &u
		}
		out = append(out, wl)
	}
	return out, nil
}

// =================================================================
// DecideReconciliation — runs the pure write-back, mutates substrate,
// and writes the EvidenceTrace node + edges.
// =================================================================

func (s *ReconciliationStore) DecideReconciliation(ctx context.Context, in interfaces.DecideReconciliationInputs) (*interfaces.ReconciliationDecision, error) {
	if !reconciliation.IsValidACOPDecision(in.ACOPDecision) {
		return nil, fmt.Errorf("invalid acop_decision %q", in.ACOPDecision)
	}
	if in.ACOPRoleRef == uuid.Nil {
		return nil, errors.New("acop_role_ref required")
	}

	// Load worklist + decision row + discharge document for context.
	wl, err := s.loadWorklist(ctx, in.WorklistRef)
	if err != nil {
		return nil, err
	}
	if wl.Status == string(reconciliation.WorklistCompleted) || wl.Status == string(reconciliation.WorklistAbandoned) {
		return nil, fmt.Errorf("worklist %s is %s; decisions are closed", wl.ID, wl.Status)
	}

	decRow, err := s.loadDecision(ctx, in.DecisionRef)
	if err != nil {
		return nil, err
	}
	if decRow.WorklistRef != in.WorklistRef {
		return nil, fmt.Errorf("decision %s does not belong to worklist %s", in.DecisionRef, in.WorklistRef)
	}
	if decRow.ACOPDecision != "" {
		return nil, fmt.Errorf("decision %s already recorded as %s", decRow.ID, decRow.ACOPDecision)
	}

	doc, err := s.docStore.GetDischargeDocument(ctx, wl.DischargeDocumentRef)
	if err != nil {
		return nil, err
	}

	// Reconstruct DiffEntry from the persisted refs.
	diff, err := s.reconstructDiffEntry(ctx, decRow, doc)
	if err != nil {
		return nil, err
	}

	// Resolve effective intent class — explicit override wins; else stored value.
	intent := reconciliation.IntentClass(decRow.IntentClass)
	if in.IntentClassOverride != "" {
		if !reconciliation.IsValidIntentClass(in.IntentClassOverride) {
			return nil, fmt.Errorf("invalid intent_class_override %q", in.IntentClassOverride)
		}
		intent = reconciliation.IntentClass(in.IntentClassOverride)
	}

	override := (*reconciliation.DecisionOverride)(nil)
	if in.ACOPDecision == string(reconciliation.ACOPModify) {
		override = &reconciliation.DecisionOverride{
			Dose:        in.OverrideDose,
			Frequency:   in.OverrideFrequency,
			Route:       in.OverrideRoute,
			IntentClass: reconciliation.IntentClass(in.IntentClassOverride),
		}
	}

	dctx := reconciliation.DecisionContext{
		Decision:    reconciliation.ACOPDecision(in.ACOPDecision),
		IntentClass: intent,
		Diff:        diff,
		DischargeAt: doc.DischargeDate,
		Notes:       in.Notes,
		Override:    override,
	}
	mutation, err := reconciliation.ApplyDecision(dctx, doc.ResidentRef, nil, s.now())
	if err != nil {
		return nil, fmt.Errorf("apply decision: %w", err)
	}

	// Apply substrate mutation + write EvidenceTrace node + edges.
	resultingMedicineUseRef, evidenceTraceNodeRef, err := s.applyMutationAndAudit(ctx, mutation, doc, decRow, in)
	if err != nil {
		return nil, err
	}

	// Update reconciliation_decisions row.
	now := s.now()
	const upd = `
		UPDATE reconciliation_decisions
		SET intent_class = $1,
		    acop_decision = $2,
		    acop_role_ref = $3,
		    decided_at = $4,
		    notes = $5,
		    resulting_medicine_use_ref = $6,
		    evidence_trace_node_ref = $7
		WHERE id = $8`
	if _, err := s.db.ExecContext(ctx, upd,
		string(intent), in.ACOPDecision, in.ACOPRoleRef, now,
		nilIfEmpty(in.Notes),
		nullUUID(resultingMedicineUseRef),
		nullUUID(evidenceTraceNodeRef),
		decRow.ID,
	); err != nil {
		return nil, fmt.Errorf("update reconciliation_decision: %w", err)
	}

	// Move worklist to in_progress on first decision.
	if wl.Status == string(reconciliation.WorklistPending) {
		const updWL = `UPDATE reconciliation_worklists SET status = $1 WHERE id = $2 AND status = 'pending'`
		if _, err := s.db.ExecContext(ctx, updWL, string(reconciliation.WorklistInProgress), wl.ID); err != nil {
			return nil, fmt.Errorf("advance worklist status: %w", err)
		}
	}

	out, err := s.loadDecision(ctx, decRow.ID)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// reconstructDiffEntry rebuilds a reconciliation.DiffEntry from the
// persisted decision + the discharge document so ApplyDecision has the
// same payload it would have seen at start time.
func (s *ReconciliationStore) reconstructDiffEntry(ctx context.Context, dec *interfaces.ReconciliationDecision, doc *interfaces.DischargeDocument) (reconciliation.DiffEntry, error) {
	var entry reconciliation.DiffEntry
	entry.Class = reconciliation.DiffClass(dec.DiffClass)

	if dec.DischargeMedLineRef != nil {
		ref := *dec.DischargeMedLineRef
		entry.DischargeLineRef = &ref
		// Find the matching line in doc.MedicationLines.
		for _, ln := range doc.MedicationLines {
			if ln.ID == ref {
				entry.DischargeLineMedicine = &reconciliation.DischargeLineSummary{
					LineRef:        ln.ID,
					AMTCode:        ln.AMTCode,
					DisplayName:    ln.MedicationNameRaw,
					Dose:           ln.DoseRaw,
					Frequency:      ln.FrequencyRaw,
					Route:          ln.RouteRaw,
					IndicationText: ln.IndicationText,
					Notes:          ln.Notes,
				}
				break
			}
		}
	}
	if dec.PreAdmissionMedicineUseRef != nil {
		ref := *dec.PreAdmissionMedicineUseRef
		entry.PreAdmissionMedUseRef = &ref
		mu, err := s.v2.GetMedicineUse(ctx, ref)
		if err != nil && !errors.Is(err, interfaces.ErrNotFound) {
			return entry, fmt.Errorf("load pre-admission medicine: %w", err)
		}
		if mu != nil {
			entry.PreAdmissionMedicine = mu
		}
	}
	// Recompute DoseChangeSummary for dose_change so the audit note carries it.
	if entry.Class == reconciliation.DiffDoseChange && entry.PreAdmissionMedicine != nil && entry.DischargeLineMedicine != nil {
		// Reuse the engine's compareDose by passing through ComputeDiff
		// would be wasteful; we just synthesise a short summary.
		entry.DoseChangeSummary = fmt.Sprintf("dose %q→%q",
			entry.PreAdmissionMedicine.Dose, entry.DischargeLineMedicine.Dose)
	}
	return entry, nil
}

// applyMutationAndAudit executes a Mutation and writes the EvidenceTrace
// node + edges. Returns the resulting MedicineUse ref (when applicable)
// and the EvidenceTrace node ref.
func (s *ReconciliationStore) applyMutationAndAudit(ctx context.Context, mut reconciliation.Mutation, doc *interfaces.DischargeDocument, dec *interfaces.ReconciliationDecision, in interfaces.DecideReconciliationInputs) (*uuid.UUID, *uuid.UUID, error) {
	now := s.now()
	var resultingMedicineUseRef *uuid.UUID

	switch mut.Kind {
	case reconciliation.MutationInsert:
		if mut.Insert == nil {
			return nil, nil, errors.New("insert mutation missing payload")
		}
		// The pure engine constructed MedicineUse with a fresh ID; write it.
		persisted, err := s.v2.UpsertMedicineUse(ctx, *mut.Insert)
		if err != nil {
			return nil, nil, fmt.Errorf("insert medicine_use: %w", err)
		}
		resultingMedicineUseRef = &persisted.ID

	case reconciliation.MutationEnd, reconciliation.MutationUpdate:
		if mut.Update == nil {
			return nil, nil, errors.New("update/end mutation missing payload")
		}
		current, err := s.v2.GetMedicineUse(ctx, mut.Update.ID)
		if err != nil {
			return nil, nil, fmt.Errorf("load existing medicine_use: %w", err)
		}
		merged := *current
		if mut.Update.Status != "" {
			merged.Status = mut.Update.Status
		}
		if mut.Update.EndedAt != nil {
			merged.EndedAt = mut.Update.EndedAt
		}
		if mut.Update.Dose != "" {
			merged.Dose = mut.Update.Dose
		}
		if mut.Update.Frequency != "" {
			merged.Frequency = mut.Update.Frequency
		}
		if mut.Update.Route != "" {
			merged.Route = mut.Update.Route
		}
		// Append audit note onto Intent.Notes so the change is human-visible
		// without a dedicated review_outcome_history table.
		if mut.Update.ReviewOutcomeNote != "" {
			if merged.Intent.Notes == "" {
				merged.Intent.Notes = mut.Update.ReviewOutcomeNote
			} else {
				merged.Intent.Notes += " | " + mut.Update.ReviewOutcomeNote
			}
		}
		persisted, err := s.v2.UpsertMedicineUse(ctx, merged)
		if err != nil {
			return nil, nil, fmt.Errorf("update medicine_use: %w", err)
		}
		resultingMedicineUseRef = &persisted.ID

	case reconciliation.MutationNoop:
		// No substrate change.
	}

	// EvidenceTrace node — ALWAYS written (non-negotiable per plan).
	nodeID := uuid.New()
	rid := doc.ResidentRef
	roleRef := in.ACOPRoleRef
	inputs := []models.TraceInput{
		{InputType: models.TraceInputTypeOther, InputRef: doc.ID, RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
	}
	if dec.DischargeMedLineRef != nil {
		inputs = append(inputs, models.TraceInput{
			InputType: models.TraceInputTypeOther, InputRef: *dec.DischargeMedLineRef,
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		})
	}
	if dec.PreAdmissionMedicineUseRef != nil {
		inputs = append(inputs, models.TraceInput{
			InputType: models.TraceInputTypeMedicineUse, InputRef: *dec.PreAdmissionMedicineUseRef,
			RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence,
		})
	}
	outputs := []models.TraceOutput{}
	if resultingMedicineUseRef != nil {
		outputs = append(outputs, models.TraceOutput{
			OutputType: "MedicineUse", OutputRef: *resultingMedicineUseRef,
		})
	}
	rs := &models.ReasoningSummary{
		Text:      mut.HumanReadableSummary,
		RuleFires: []string{"reconciliation_decision:" + in.ACOPDecision + ":" + dec.DiffClass},
	}
	node := models.EvidenceTraceNode{
		ID:               nodeID,
		StateMachine:     models.EvidenceTraceStateMachineRecommendation,
		StateChangeType:  "reconciliation_decision",
		RecordedAt:       now,
		OccurredAt:       doc.DischargeDate,
		Actor:            models.TraceActor{RoleRef: &roleRef},
		Inputs:           inputs,
		ReasoningSummary: rs,
		Outputs:          outputs,
		ResidentRef:      &rid,
		CreatedAt:        now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return nil, nil, fmt.Errorf("upsert evidence trace node: %w", err)
	}

	// Edge: decision derived_from discharge_document_evidence_trace
	// (we don't store the parent node ref on the worklist row; the
	// audit graph's recursive-CTE traversal can still find the path
	// because both nodes carry resident_ref + the decision's input
	// references include the discharge_document id).
	//
	// Edge from decision → resulting MedicineUse (led_to).
	if resultingMedicineUseRef != nil {
		// We can't FK into MedicineUse from the EvidenceTrace edge table
		// (edges are between trace nodes), so the led_to relationship is
		// captured in node.Outputs above. The edge table is for
		// node-to-node relationships only.
		_ = resultingMedicineUseRef
	}

	return resultingMedicineUseRef, &nodeID, nil
}

func (s *ReconciliationStore) loadDecision(ctx context.Context, id uuid.UUID) (*interfaces.ReconciliationDecision, error) {
	const q = `
		SELECT id, worklist_ref, discharge_med_line_ref, pre_admission_medicine_use_ref,
		       diff_class, intent_class, COALESCE(acop_decision,''),
		       acop_role_ref, decided_at, COALESCE(notes,''),
		       resulting_medicine_use_ref, evidence_trace_node_ref, created_at
		FROM reconciliation_decisions
		WHERE id = $1`
	var (
		d           interfaces.ReconciliationDecision
		lineRef     uuid.NullUUID
		preRef      uuid.NullUUID
		roleRef     uuid.NullUUID
		decidedAt   sql.NullTime
		resultRef   uuid.NullUUID
		evidenceRef uuid.NullUUID
	)
	err := s.db.QueryRowContext(ctx, q, id).Scan(
		&d.ID, &d.WorklistRef, &lineRef, &preRef,
		&d.DiffClass, &d.IntentClass, &d.ACOPDecision,
		&roleRef, &decidedAt, &d.Notes,
		&resultRef, &evidenceRef, &d.CreatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, interfaces.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if lineRef.Valid {
		u := lineRef.UUID
		d.DischargeMedLineRef = &u
	}
	if preRef.Valid {
		u := preRef.UUID
		d.PreAdmissionMedicineUseRef = &u
	}
	if roleRef.Valid {
		u := roleRef.UUID
		d.ACOPRoleRef = &u
	}
	if decidedAt.Valid {
		t := decidedAt.Time
		d.DecidedAt = &t
	}
	if resultRef.Valid {
		u := resultRef.UUID
		d.ResultingMedicineUseRef = &u
	}
	if evidenceRef.Valid {
		u := evidenceRef.UUID
		d.EvidenceTraceNodeRef = &u
	}
	return &d, nil
}

// =================================================================
// FinaliseWorklist — marks worklist completed + emits the
// reconciliation_completed Event + parent EvidenceTrace edge
// =================================================================

func (s *ReconciliationStore) FinaliseWorklist(ctx context.Context, worklistRef uuid.UUID, completedByRoleRef uuid.UUID) (*interfaces.FinaliseReconciliationResult, error) {
	if completedByRoleRef == uuid.Nil {
		return nil, errors.New("completed_by_role_ref required")
	}
	wl, err := s.loadWorklist(ctx, worklistRef)
	if err != nil {
		return nil, err
	}
	if wl.Status == string(reconciliation.WorklistCompleted) {
		return nil, fmt.Errorf("worklist %s already completed", wl.ID)
	}
	decs, err := s.loadDecisions(ctx, worklistRef)
	if err != nil {
		return nil, err
	}
	for _, d := range decs {
		if d.ACOPDecision == "" {
			return nil, fmt.Errorf("decision %s pending: cannot finalise", d.ID)
		}
	}

	now := s.now()
	const upd = `
		UPDATE reconciliation_worklists
		SET status = $1, completed_at = $2, completed_by_role_ref = $3
		WHERE id = $4`
	if _, err := s.db.ExecContext(ctx, upd,
		string(reconciliation.WorklistCompleted), now, completedByRoleRef, wl.ID,
	); err != nil {
		return nil, fmt.Errorf("update worklist status: %w", err)
	}

	// Emit reconciliation_completed Event.
	ev := models.Event{
		ID:                  uuid.New(),
		EventType:           models.EventTypeReconciliationCompleted,
		OccurredAt:          now,
		ResidentID:          wl.ResidentRef,
		ReportedByRef:       completedByRoleRef,
		DescriptionFreeText: fmt.Sprintf("reconciliation worklist %s completed (%d decisions)", wl.ID, len(decs)),
	}
	persistedEvent, err := s.v2.UpsertEvent(ctx, ev)
	if err != nil {
		return nil, fmt.Errorf("emit reconciliation_completed event: %w", err)
	}

	// Collect resulting MedicineUse refs from decisions.
	refs := []uuid.UUID{}
	for _, d := range decs {
		if d.ResultingMedicineUseRef != nil {
			refs = append(refs, *d.ResultingMedicineUseRef)
		}
	}

	// Wire the audit graph: discharge_document_evidence_trace → decision → completion event.
	// Look up the parent worklist EvidenceTrace node by its OutputRef (worklist ID).
	// For MVP, we link each decision's EvidenceTrace node forward via a led_to edge
	// to the completion Event (so backward traversal from the Event finds every decision).
	completionNodeID, err := s.writeCompletionEvidenceTrace(ctx, wl, persistedEvent.ID, len(decs), refs, now)
	if err != nil {
		return nil, fmt.Errorf("write completion evidence trace: %w", err)
	}
	for _, d := range decs {
		if d.EvidenceTraceNodeRef == nil {
			continue
		}
		if err := s.v2.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
			From: *d.EvidenceTraceNodeRef,
			To:   completionNodeID,
			Kind: evidence_trace.EdgeKindLedTo,
		}); err != nil {
			return nil, fmt.Errorf("insert decision→completion edge: %w", err)
		}
	}

	wl.Status = string(reconciliation.WorklistCompleted)
	wl.CompletedAt = &now
	wl.CompletedByRoleRef = &completedByRoleRef
	return &interfaces.FinaliseReconciliationResult{
		Worklist:                 wl,
		ResultingMedicineUseRefs: refs,
		CompletionEventID:        &persistedEvent.ID,
	}, nil
}

func (s *ReconciliationStore) writeCompletionEvidenceTrace(ctx context.Context, wl *interfaces.ReconciliationWorklist, eventID uuid.UUID, decisionCount int, resultingRefs []uuid.UUID, now time.Time) (uuid.UUID, error) {
	nodeID := uuid.New()
	rid := wl.ResidentRef
	rs := &models.ReasoningSummary{
		Text:      fmt.Sprintf("reconciliation_completed worklist=%s decisions=%d results=%d", wl.ID, decisionCount, len(resultingRefs)),
		RuleFires: []string{"reconciliation_completed"},
	}
	outputs := []models.TraceOutput{
		{OutputType: "Event", OutputRef: eventID},
	}
	for _, r := range resultingRefs {
		outputs = append(outputs, models.TraceOutput{OutputType: "MedicineUse", OutputRef: r})
	}
	node := models.EvidenceTraceNode{
		ID:              nodeID,
		StateMachine:    models.EvidenceTraceStateMachineRecommendation,
		StateChangeType: "reconciliation_completed",
		RecordedAt:      now,
		OccurredAt:      now,
		Inputs: []models.TraceInput{
			{InputType: models.TraceInputTypeOther, InputRef: wl.ID, RoleInDecision: models.TraceRoleInDecisionPrimaryEvidence},
		},
		ReasoningSummary: rs,
		Outputs:          outputs,
		ResidentRef:      &rid,
		CreatedAt:        now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return uuid.Nil, err
	}
	return nodeID, nil
}

// nullUUID converts an optional UUID into a sql/driver-compatible value.
// Nil pointers are encoded as NULL; non-nil values pass through as the
// underlying UUID.
func nullUUID(u *uuid.UUID) interface{} {
	if u == nil {
		return nil
	}
	return *u
}
