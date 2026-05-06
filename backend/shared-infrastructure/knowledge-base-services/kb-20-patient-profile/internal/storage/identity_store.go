// Package storage — identity matching persistence + service layer.
//
// IdentityStore is kb-20's binding between the pure-package matcher
// (shared/v2_substrate/identity) and the database. It implements:
//
//   - identity.IdentityCandidateLookup so the matcher can resolve
//     candidates without depending on a *sql.DB,
//   - interfaces.IdentityMappingStore  for identity_mappings CRUD,
//   - interfaces.IdentityReviewQueueStore for identity_review_queue CRUD.
//
// On top of those, MatchAndPersist is the service-level entry point
// that calls Match, writes the EvidenceTrace audit node (every match
// decision — HIGH, MEDIUM, LOW, NONE — produces one), persists the
// mapping for HIGH/MEDIUM tiers, and enqueues LOW/NONE entries for
// reviewer attention. ResolveReview is the symmetric promotion path
// with post-hoc re-routing (Layer 2 §3.3).
package storage

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/identity"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// IdentityStore is the kb-20 persistence + service layer for identity
// matching. It owns a *sql.DB shared with V2SubstrateStore (the
// EvidenceTrace writer) and a *V2SubstrateStore handle for the
// EvidenceTrace upsert path.
type IdentityStore struct {
	db    *sql.DB
	v2    *V2SubstrateStore // for EvidenceTrace node writes
}

// NewIdentityStore constructs an IdentityStore. The provided
// V2SubstrateStore must be backed by the same *sql.DB so the audit
// writes happen in the same connection pool (and, for callers that
// later thread a transaction in, the same transaction).
func NewIdentityStore(db *sql.DB, v2 *V2SubstrateStore) *IdentityStore {
	return &IdentityStore{db: db, v2: v2}
}

// ---------------------------------------------------------------------------
// IdentityCandidateLookup
// ---------------------------------------------------------------------------

// LookupByIHI consults two sources in order:
//
//  1. patient_profiles.ihi — the source of truth for IHI on the
//     Resident itself (added by migration 008_part1).
//  2. identity_mappings where identifier_kind='ihi' — secondary
//     mappings recorded post-hoc (e.g. an IHI surfaced from an
//     external source that subsequently matched a known resident
//     via a fuzzy path).
//
// Returns interfaces.ErrNotFound when no mapping exists. The matcher's
// isNotFoundErr recognises this sentinel and falls through.
func (s *IdentityStore) LookupByIHI(ctx context.Context, ihi string) (*uuid.UUID, error) {
	if ihi == "" {
		return nil, fmt.Errorf("lookup_by_ihi: %w", interfaces.ErrNotFound)
	}
	// 1) Direct on patient_profiles.
	const q1 = `SELECT id FROM patient_profiles WHERE ihi = $1 LIMIT 1`
	var id uuid.UUID
	err := s.db.QueryRowContext(ctx, q1, ihi).Scan(&id)
	if err == nil {
		return &id, nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return nil, fmt.Errorf("lookup_by_ihi pp: %w", err)
	}
	// 2) Fall back to identity_mappings.
	const q2 = `SELECT resident_ref FROM identity_mappings
	             WHERE identifier_kind = 'ihi' AND identifier_value = $1
	             ORDER BY created_at DESC LIMIT 1`
	if err := s.db.QueryRowContext(ctx, q2, ihi).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("lookup_by_ihi: %w", interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("lookup_by_ihi map: %w", err)
	}
	return &id, nil
}

// LookupByMedicare goes through identity_mappings (kind='medicare')
// joined to residents_v2 for the candidate fields the matcher needs.
// Empty result is NOT an error — the matcher treats it as "no
// medicare mapping" and falls through.
func (s *IdentityStore) LookupByMedicare(ctx context.Context, medicare string) ([]identity.ResidentCandidate, error) {
	if medicare == "" {
		return nil, nil
	}
	const q = `SELECT r.id, COALESCE(r.given_name,''), COALESCE(r.family_name,''), COALESCE(r.dob, '0001-01-01'::date)
	             FROM identity_mappings m
	             JOIN residents_v2 r ON r.id = m.resident_ref
	            WHERE m.identifier_kind = 'medicare'
	              AND m.identifier_value = $1`
	rows, err := s.db.QueryContext(ctx, q, medicare)
	if err != nil {
		return nil, fmt.Errorf("lookup_by_medicare: %w", err)
	}
	defer rows.Close()
	var out []identity.ResidentCandidate
	for rows.Next() {
		var c identity.ResidentCandidate
		if err := rows.Scan(&c.ID, &c.GivenName, &c.FamilyName, &c.DOB); err != nil {
			return nil, fmt.Errorf("lookup_by_medicare scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// LookupByFacilityAndDOB queries residents_v2 directly. We compare on
// the dob date column, not date_trunc on a timestamp, so a midnight-
// UTC dob from the matcher matches a stored DATE row exactly.
func (s *IdentityStore) LookupByFacilityAndDOB(ctx context.Context, facility uuid.UUID, dob time.Time) ([]identity.ResidentCandidate, error) {
	const q = `SELECT id, COALESCE(given_name,''), COALESCE(family_name,''), COALESCE(dob, '0001-01-01'::date)
	             FROM residents_v2
	            WHERE facility_id = $1 AND dob = $2::date`
	rows, err := s.db.QueryContext(ctx, q, facility, dob.UTC().Format("2006-01-02"))
	if err != nil {
		return nil, fmt.Errorf("lookup_by_facility_dob: %w", err)
	}
	defer rows.Close()
	var out []identity.ResidentCandidate
	for rows.Next() {
		var c identity.ResidentCandidate
		if err := rows.Scan(&c.ID, &c.GivenName, &c.FamilyName, &c.DOB); err != nil {
			return nil, fmt.Errorf("lookup_by_facility_dob scan: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ---------------------------------------------------------------------------
// IdentityMappingStore
// ---------------------------------------------------------------------------

// InsertIdentityMapping upserts on (identifier_kind, identifier_value,
// resident_ref). The unique constraint guarantees idempotency for
// repeated writes of the same canonical mapping.
func (s *IdentityStore) InsertIdentityMapping(ctx context.Context, m interfaces.IdentityMapping) (*interfaces.IdentityMapping, error) {
	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}
	const q = `
		INSERT INTO identity_mappings
			(id, identifier_kind, identifier_value, resident_ref,
			 confidence, match_path, source, verified_by, verified_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW())
		ON CONFLICT (identifier_kind, identifier_value, resident_ref) DO UPDATE SET
			confidence  = EXCLUDED.confidence,
			match_path  = EXCLUDED.match_path,
			source      = EXCLUDED.source,
			verified_by = EXCLUDED.verified_by,
			verified_at = EXCLUDED.verified_at
		RETURNING id, identifier_kind, identifier_value, resident_ref,
		          confidence, match_path, source, verified_by, verified_at, created_at`
	var (
		out         interfaces.IdentityMapping
		verifiedBy  uuid.NullUUID
		verifiedAt  sql.NullTime
		verArg      interface{}
		verAtArg    interface{}
	)
	if m.VerifiedBy != nil {
		verArg = *m.VerifiedBy
	}
	if m.VerifiedAt != nil {
		verAtArg = *m.VerifiedAt
	}
	if err := s.db.QueryRowContext(ctx, q,
		m.ID, m.IdentifierKind, m.IdentifierValue, m.ResidentRef,
		m.Confidence, m.MatchPath, m.Source, verArg, verAtArg,
	).Scan(
		&out.ID, &out.IdentifierKind, &out.IdentifierValue, &out.ResidentRef,
		&out.Confidence, &out.MatchPath, &out.Source, &verifiedBy, &verifiedAt, &out.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("insert_identity_mapping: %w", err)
	}
	if verifiedBy.Valid {
		v := verifiedBy.UUID
		out.VerifiedBy = &v
	}
	if verifiedAt.Valid {
		t := verifiedAt.Time
		out.VerifiedAt = &t
	}
	return &out, nil
}

// ListIdentityMappingsByResident returns every mapping pointing at
// residentRef, newest-first. Used by ResolveReview to enumerate rows
// that may need to be repointed.
func (s *IdentityStore) ListIdentityMappingsByResident(ctx context.Context, residentRef uuid.UUID) ([]interfaces.IdentityMapping, error) {
	const q = `SELECT id, identifier_kind, identifier_value, resident_ref,
	                  confidence, match_path, source, verified_by, verified_at, created_at
	             FROM identity_mappings
	            WHERE resident_ref = $1
	            ORDER BY created_at DESC`
	rows, err := s.db.QueryContext(ctx, q, residentRef)
	if err != nil {
		return nil, fmt.Errorf("list_identity_mappings: %w", err)
	}
	defer rows.Close()
	var out []interfaces.IdentityMapping
	for rows.Next() {
		var (
			m          interfaces.IdentityMapping
			verifiedBy uuid.NullUUID
			verifiedAt sql.NullTime
		)
		if err := rows.Scan(
			&m.ID, &m.IdentifierKind, &m.IdentifierValue, &m.ResidentRef,
			&m.Confidence, &m.MatchPath, &m.Source, &verifiedBy, &verifiedAt, &m.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("list_identity_mappings scan: %w", err)
		}
		if verifiedBy.Valid {
			v := verifiedBy.UUID
			m.VerifiedBy = &v
		}
		if verifiedAt.Valid {
			t := verifiedAt.Time
			m.VerifiedAt = &t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// ReassignIdentityMappingsByResidentSince repoints every mapping whose
// resident_ref == fromRef AND created_at >= since onto toRef. The
// time floor scopes the re-route to mappings created after the
// queued decision (per the conservative re-route policy at Layer 2
// §3.3); pre-existing mappings to the wrong resident need a
// separate ops process. Returns the count of rows affected.
//
// We avoid violating the UNIQUE (kind, value, resident_ref) by
// DELETE-then-INSERT semantics: if a (kind, value, toRef) row already
// exists we delete the from-row rather than collide.
func (s *IdentityStore) ReassignIdentityMappingsByResidentSince(ctx context.Context, fromRef, toRef uuid.UUID, since time.Time) (int, error) {
	if fromRef == toRef {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, fmt.Errorf("reassign begin: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	// Step 1: delete any from-rows whose (kind, value) already has a to-row.
	const qDelDup = `
		DELETE FROM identity_mappings m1
		 WHERE m1.resident_ref = $1
		   AND m1.created_at  >= $3
		   AND EXISTS (
		       SELECT 1 FROM identity_mappings m2
		        WHERE m2.identifier_kind  = m1.identifier_kind
		          AND m2.identifier_value = m1.identifier_value
		          AND m2.resident_ref     = $2
		   )`
	if _, err := tx.ExecContext(ctx, qDelDup, fromRef, toRef, since); err != nil {
		return 0, fmt.Errorf("reassign dedup: %w", err)
	}

	// Step 2: update remaining from-rows to point at toRef.
	const qUpd = `
		UPDATE identity_mappings
		   SET resident_ref = $2
		 WHERE resident_ref = $1
		   AND created_at  >= $3`
	res, err := tx.ExecContext(ctx, qUpd, fromRef, toRef, since)
	if err != nil {
		return 0, fmt.Errorf("reassign update: %w", err)
	}
	n, _ := res.RowsAffected()
	if err := tx.Commit(); err != nil {
		return 0, fmt.Errorf("reassign commit: %w", err)
	}
	return int(n), nil
}

// ---------------------------------------------------------------------------
// IdentityReviewQueueStore
// ---------------------------------------------------------------------------

// InsertIdentityReviewQueueEntry creates a pending entry. Caller is
// responsible for filling the JSONB IncomingIdentifier (the matcher's
// IncomingIdentifier marshalled).
func (s *IdentityStore) InsertIdentityReviewQueueEntry(ctx context.Context, e interfaces.IdentityReviewQueueEntry) (*interfaces.IdentityReviewQueueEntry, error) {
	if e.ID == uuid.Nil {
		e.ID = uuid.New()
	}
	if e.Status == "" {
		e.Status = "pending"
	}
	const q = `
		INSERT INTO identity_review_queue
			(id, incoming_identifier, candidate_resident_refs,
			 best_candidate, best_distance, match_path, confidence,
			 source, status, evidence_trace_node_ref, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, NOW())
		RETURNING id, incoming_identifier, candidate_resident_refs,
		          best_candidate, best_distance, match_path, confidence,
		          source, status, resolved_resident_ref, resolved_by,
		          resolved_at, resolution_note, evidence_trace_node_ref, created_at`
	var (
		bestCand uuid.NullUUID
		bestDist sql.NullInt64
		bestArg  interface{}
		distArg  interface{}
		etRef    uuid.NullUUID
		etArg    interface{}
	)
	if e.BestCandidate != nil {
		bestArg = *e.BestCandidate
	}
	if e.BestDistance != nil {
		distArg = *e.BestDistance
	}
	if e.EvidenceTraceNodeRef != nil {
		etArg = *e.EvidenceTraceNodeRef
	}
	candRefStrings := make([]string, len(e.CandidateResidentRefs))
	for i, u := range e.CandidateResidentRefs {
		candRefStrings[i] = u.String()
	}

	var (
		out          interfaces.IdentityReviewQueueEntry
		incoming     []byte
		candArr      pq.StringArray
		resolvedRef  uuid.NullUUID
		resolvedBy   uuid.NullUUID
		resolvedAt   sql.NullTime
		resolutionN  sql.NullString
	)
	if err := s.db.QueryRowContext(ctx, q,
		e.ID, []byte(e.IncomingIdentifier), pq.Array(candRefStrings),
		bestArg, distArg, e.MatchPath, e.Confidence,
		e.Source, e.Status, etArg,
	).Scan(
		&out.ID, &incoming, &candArr,
		&bestCand, &bestDist, &out.MatchPath, &out.Confidence,
		&out.Source, &out.Status, &resolvedRef, &resolvedBy,
		&resolvedAt, &resolutionN, &etRef, &out.CreatedAt,
	); err != nil {
		return nil, fmt.Errorf("insert_review_queue: %w", err)
	}
	out.IncomingIdentifier = json.RawMessage(incoming)
	out.CandidateResidentRefs = parseStringUUIDs(candArr)
	if bestCand.Valid {
		v := bestCand.UUID
		out.BestCandidate = &v
	}
	if bestDist.Valid {
		d := int(bestDist.Int64)
		out.BestDistance = &d
	}
	if resolvedRef.Valid {
		v := resolvedRef.UUID
		out.ResolvedResidentRef = &v
	}
	if resolvedBy.Valid {
		v := resolvedBy.UUID
		out.ResolvedBy = &v
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		out.ResolvedAt = &t
	}
	if resolutionN.Valid {
		out.ResolutionNote = resolutionN.String
	}
	if etRef.Valid {
		v := etRef.UUID
		out.EvidenceTraceNodeRef = &v
	}
	return &out, nil
}

const reviewQueueColumns = `id, incoming_identifier, candidate_resident_refs,
       best_candidate, best_distance, match_path, confidence,
       source, status, resolved_resident_ref, resolved_by,
       resolved_at, resolution_note, evidence_trace_node_ref, created_at`

func scanReviewQueueRow(sc rowScanner) (interfaces.IdentityReviewQueueEntry, error) {
	var (
		out          interfaces.IdentityReviewQueueEntry
		incoming     []byte
		candArr      pq.StringArray
		bestCand     uuid.NullUUID
		bestDist     sql.NullInt64
		resolvedRef  uuid.NullUUID
		resolvedBy   uuid.NullUUID
		resolvedAt   sql.NullTime
		resolutionN  sql.NullString
		etRef        uuid.NullUUID
	)
	if err := sc.Scan(
		&out.ID, &incoming, &candArr,
		&bestCand, &bestDist, &out.MatchPath, &out.Confidence,
		&out.Source, &out.Status, &resolvedRef, &resolvedBy,
		&resolvedAt, &resolutionN, &etRef, &out.CreatedAt,
	); err != nil {
		return interfaces.IdentityReviewQueueEntry{}, err
	}
	out.IncomingIdentifier = json.RawMessage(incoming)
	out.CandidateResidentRefs = parseStringUUIDs(candArr)
	if bestCand.Valid {
		v := bestCand.UUID
		out.BestCandidate = &v
	}
	if bestDist.Valid {
		d := int(bestDist.Int64)
		out.BestDistance = &d
	}
	if resolvedRef.Valid {
		v := resolvedRef.UUID
		out.ResolvedResidentRef = &v
	}
	if resolvedBy.Valid {
		v := resolvedBy.UUID
		out.ResolvedBy = &v
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		out.ResolvedAt = &t
	}
	if resolutionN.Valid {
		out.ResolutionNote = resolutionN.String
	}
	if etRef.Valid {
		v := etRef.UUID
		out.EvidenceTraceNodeRef = &v
	}
	return out, nil
}

// GetIdentityReviewQueueEntry reads one entry by id.
func (s *IdentityStore) GetIdentityReviewQueueEntry(ctx context.Context, id uuid.UUID) (*interfaces.IdentityReviewQueueEntry, error) {
	q := `SELECT ` + reviewQueueColumns + ` FROM identity_review_queue WHERE id = $1`
	out, err := scanReviewQueueRow(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get_review_queue %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get_review_queue %s: %w", id, err)
	}
	return &out, nil
}

// ListIdentityReviewQueue paginates entries filtered by status.
// status="" means any status. Newest-first ordering.
func (s *IdentityStore) ListIdentityReviewQueue(ctx context.Context, status string, limit, offset int) ([]interfaces.IdentityReviewQueueEntry, error) {
	q := `SELECT ` + reviewQueueColumns + ` FROM identity_review_queue`
	args := []interface{}{}
	if status != "" {
		q += ` WHERE status = $1`
		args = append(args, status)
	}
	q += ` ORDER BY created_at DESC LIMIT $` + fmt.Sprintf("%d", len(args)+1) +
		` OFFSET $` + fmt.Sprintf("%d", len(args)+2)
	args = append(args, limit, offset)

	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("list_review_queue: %w", err)
	}
	defer rows.Close()
	var out []interfaces.IdentityReviewQueueEntry
	for rows.Next() {
		e, err := scanReviewQueueRow(rows)
		if err != nil {
			return nil, fmt.Errorf("list_review_queue scan: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// UpdateIdentityReviewQueueEntryResolution marks an entry resolved or
// rejected. Pass status="resolved" with resolvedRef pointing at the
// canonical Resident; pass status="rejected" with resolvedRef nil to
// signal "this is bad data, no resident here".
func (s *IdentityStore) UpdateIdentityReviewQueueEntryResolution(ctx context.Context, id uuid.UUID, status string, resolvedRef *uuid.UUID, resolvedBy uuid.UUID, note string) (*interfaces.IdentityReviewQueueEntry, error) {
	if status != "resolved" && status != "rejected" {
		return nil, fmt.Errorf("update_review_resolution: status must be resolved|rejected, got %q", status)
	}
	const q = `UPDATE identity_review_queue
	             SET status = $2,
	                 resolved_resident_ref = $3,
	                 resolved_by = $4,
	                 resolved_at = NOW(),
	                 resolution_note = $5
	           WHERE id = $1
	         RETURNING ` + reviewQueueColumns
	var resolvedArg interface{}
	if resolvedRef != nil {
		resolvedArg = *resolvedRef
	}
	row := s.db.QueryRowContext(ctx, q, id, status, resolvedArg, resolvedBy, nilIfEmpty(note))
	out, err := scanReviewQueueRow(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("update_review_resolution %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("update_review_resolution %s: %w", id, err)
	}
	return &out, nil
}

// ---------------------------------------------------------------------------
// Service-level: MatchAndPersist + ResolveReview
// ---------------------------------------------------------------------------

// MatchAndPersistResult bundles the matcher's output with the audit
// node id (always written) and the queue entry id (written when
// RequiresReview, else nil).
type MatchAndPersistResult struct {
	Match                identity.MatchResult
	EvidenceTraceNodeRef uuid.UUID
	ReviewQueueEntryID   *uuid.UUID
}

// MatchAndPersist is the canonical service entry point. It:
//
//  1. Calls the IdentityMatcher (this store implements the lookup).
//  2. Writes an EvidenceTrace node (state_machine=ClinicalState,
//     state_change_type=identity_match) recording the inputs, the
//     resolved ResidentRef (if any), and a reasoning summary that
//     captures Path / Confidence / NameDistance.
//  3. For HIGH/MEDIUM: persists an identity_mappings row keyed on
//     the strongest identifier present in the IncomingIdentifier
//     (IHI > Medicare > facility_internal — but facility_internal
//     mappings are only written when the reviewer resolves a queued
//     decision, so the auto-write skips them).
//  4. For LOW/NONE: enqueues an identity_review_queue entry.
//
// Every match decision produces an EvidenceTrace node — that's the
// architectural moat. The persistence side-effects degrade gracefully
// (logged, returned as warnings via the result struct) but the audit
// node is required and a failure to write it propagates to the caller.
func (s *IdentityStore) MatchAndPersist(ctx context.Context, incoming identity.IncomingIdentifier) (*MatchAndPersistResult, error) {
	matcher := identity.NewMatcher(s)
	res, err := matcher.Match(ctx, incoming)
	if err != nil {
		return nil, fmt.Errorf("match: %w", err)
	}

	// Build & write the EvidenceTrace node FIRST so the audit precedes
	// any persistence side-effects. Failure here is fatal — we'd rather
	// surface a 5xx than a silently un-audited match decision.
	nodeID, err := s.writeMatchEvidenceTrace(ctx, incoming, res)
	if err != nil {
		return nil, fmt.Errorf("evidence_trace: %w", err)
	}

	out := &MatchAndPersistResult{Match: res, EvidenceTraceNodeRef: nodeID}

	switch res.Confidence {
	case identity.ConfidenceHigh, identity.ConfidenceMedium:
		// Auto-accept: persist identity_mappings on the strongest identifier.
		if res.ResidentRef == nil {
			return nil, fmt.Errorf("invariant violation: %s confidence with nil ResidentRef", res.Confidence)
		}
		if err := s.persistAutoMapping(ctx, incoming, res); err != nil {
			return nil, fmt.Errorf("persist_mapping: %w", err)
		}
	case identity.ConfidenceLow, identity.ConfidenceNone:
		// Enqueue for human verification.
		entry, err := s.enqueueReview(ctx, incoming, res, nodeID)
		if err != nil {
			return nil, fmt.Errorf("enqueue_review: %w", err)
		}
		out.ReviewQueueEntryID = &entry.ID
	}
	return out, nil
}

// writeMatchEvidenceTrace builds an EvidenceTraceNode summarising the
// match decision and writes it through the V2SubstrateStore. Returns
// the node id so callers can cross-reference it (e.g. on the review
// queue entry).
func (s *IdentityStore) writeMatchEvidenceTrace(ctx context.Context, incoming identity.IncomingIdentifier, res identity.MatchResult) (uuid.UUID, error) {
	now := time.Now().UTC()
	nodeID := uuid.New()

	// Reasoning summary captures the algorithmic path + confidence + distance
	// for after-the-fact diagnostics. We stash the full Path/Confidence text
	// in ReasoningSummary.Text and treat the rule-id slot as the path
	// identifier so downstream rule-engine consumers can filter by it.
	rs := &models.ReasoningSummary{
		Text:      fmt.Sprintf("identity_match path=%s confidence=%s distance=%d", res.Path, res.Confidence, res.NameDistance),
		RuleFires: []string{"identity_match:" + string(res.Path)},
	}

	// Inputs: IHI (treated as InputType=other since there's no IHI-shaped
	// substrate entity) plus any candidates surfaced by the LOW path so the
	// reviewer can drill back to specific residents from the audit graph.
	var inputs []models.TraceInput
	for _, c := range res.Candidates {
		inputs = append(inputs, models.TraceInput{
			InputType:      models.TraceInputTypeOther,
			InputRef:       c,
			RoleInDecision: models.TraceRoleInDecisionSecondaryEvidence,
		})
	}

	// Outputs: the matched resident (when any). We use TraceOutput.OutputType
	// = "Resident" — Layer 2 doc §1.6 leaves OutputType free-form.
	var outputs []models.TraceOutput
	if res.ResidentRef != nil {
		outputs = append(outputs, models.TraceOutput{
			OutputType: "Resident",
			OutputRef:  *res.ResidentRef,
		})
	}

	node := models.EvidenceTraceNode{
		ID:               nodeID,
		StateMachine:     models.EvidenceTraceStateMachineClinicalState,
		StateChangeType:  "identity_match",
		RecordedAt:       now,
		OccurredAt:       now,
		Inputs:           inputs,
		ReasoningSummary: rs,
		Outputs:          outputs,
		ResidentRef:      res.ResidentRef,
		CreatedAt:        now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return uuid.Nil, err
	}
	_ = incoming // reserved for future enrichment of inputs (e.g. a TraceInput referencing the source/IncomingIdentifier blob once that lives in a substrate table).
	return nodeID, nil
}

// persistAutoMapping records an identity_mappings row for a HIGH or
// MEDIUM match. We pick the strongest identifier present:
//  - HIGH: always the IHI (the matcher's path guarantees IHI != "")
//  - MEDIUM: always the Medicare number (path guarantees Medicare != "")
func (s *IdentityStore) persistAutoMapping(ctx context.Context, incoming identity.IncomingIdentifier, res identity.MatchResult) error {
	var kind, value string
	switch res.Path {
	case identity.MatchPathIHI:
		kind, value = "ihi", incoming.IHI
	case identity.MatchPathMedicareNameDOB:
		kind, value = "medicare", incoming.Medicare
	default:
		return fmt.Errorf("auto-persist not supported for path %s", res.Path)
	}
	if value == "" {
		return fmt.Errorf("invariant: path %s but identifier value is empty", res.Path)
	}
	_, err := s.InsertIdentityMapping(ctx, interfaces.IdentityMapping{
		IdentifierKind:  kind,
		IdentifierValue: value,
		ResidentRef:     *res.ResidentRef,
		Confidence:      string(res.Confidence),
		MatchPath:       string(res.Path),
		Source:          incoming.Source,
	})
	return err
}

// enqueueReview marshals the IncomingIdentifier and persists a
// review-queue entry, cross-linked to the EvidenceTrace node.
func (s *IdentityStore) enqueueReview(ctx context.Context, incoming identity.IncomingIdentifier, res identity.MatchResult, nodeID uuid.UUID) (*interfaces.IdentityReviewQueueEntry, error) {
	blob, err := json.Marshal(incoming)
	if err != nil {
		return nil, fmt.Errorf("marshal incoming: %w", err)
	}
	entry := interfaces.IdentityReviewQueueEntry{
		IncomingIdentifier:    json.RawMessage(blob),
		CandidateResidentRefs: res.Candidates,
		MatchPath:             string(res.Path),
		Confidence:            string(res.Confidence),
		Source:                incoming.Source,
		Status:                "pending",
		EvidenceTraceNodeRef:  &nodeID,
	}
	if res.ResidentRef != nil {
		v := *res.ResidentRef
		entry.BestCandidate = &v
	}
	if res.NameDistance > 0 || res.ResidentRef != nil {
		d := res.NameDistance
		entry.BestDistance = &d
	}
	return s.InsertIdentityReviewQueueEntry(ctx, entry)
}

// ResolveReview promotes a pending queue entry to resolved. It:
//
//  1. Marks the queue entry resolved with the reviewer + note.
//  2. Writes a new identity_mappings row pointing the IncomingIdentifier
//     at the resolved Resident (kind chosen the same way as
//     persistAutoMapping; for LOW path on facility_internal we use
//     'facility_internal' as the kind with a synthetic value of the
//     queue entry id so it is uniquely keyed).
//  3. Re-routes any subsequent identity_mappings written against the
//     prior best_candidate (Layer 2 §3.3 post-hoc correction).
//  4. Writes an EvidenceTrace node recording the resolution + re-route.
//
// All four steps run in sequence; the migration leaves them as
// independent statements (no FKs across boundaries), so a
// connection-level error mid-flight surfaces as a partial state to the
// caller. Future hardening can wrap them in a single transaction.
func (s *IdentityStore) ResolveReview(ctx context.Context, queueEntryID, resolvedRef, resolvedBy uuid.UUID, note string) (*interfaces.IdentityReviewQueueEntry, int, error) {
	if resolvedRef == uuid.Nil {
		return nil, 0, fmt.Errorf("resolve_review: resolvedRef must not be nil; use Reject for bad-data dispositions")
	}

	// Read the entry first so we know best_candidate (the prior auto-
	// chosen resident) for re-routing AND so we have the
	// IncomingIdentifier blob for writing the promoted mapping.
	entry, err := s.GetIdentityReviewQueueEntry(ctx, queueEntryID)
	if err != nil {
		return nil, 0, fmt.Errorf("read entry: %w", err)
	}

	// Step 1: mark resolved.
	updated, err := s.UpdateIdentityReviewQueueEntryResolution(ctx, queueEntryID, "resolved", &resolvedRef, resolvedBy, note)
	if err != nil {
		return nil, 0, fmt.Errorf("update entry: %w", err)
	}

	// Step 2: write the verified mapping.
	if err := s.persistResolvedMapping(ctx, *entry, resolvedRef, resolvedBy); err != nil {
		return updated, 0, fmt.Errorf("persist resolved mapping: %w", err)
	}

	// Step 3: re-route subsequent mappings written against best_candidate.
	rerouted := 0
	if entry.BestCandidate != nil && *entry.BestCandidate != resolvedRef {
		n, err := s.ReassignIdentityMappingsByResidentSince(ctx, *entry.BestCandidate, resolvedRef, entry.CreatedAt)
		if err != nil {
			return updated, 0, fmt.Errorf("reassign mappings: %w", err)
		}
		rerouted = n
	}

	// Step 4: audit the resolution.
	if err := s.writeResolutionEvidenceTrace(ctx, *entry, resolvedRef, resolvedBy, rerouted); err != nil {
		return updated, rerouted, fmt.Errorf("evidence_trace resolution: %w", err)
	}

	return updated, rerouted, nil
}

// persistResolvedMapping writes the promoted identity_mappings row.
// The identifier kind is derived from the queue entry's match_path:
// MatchPathIHI -> 'ihi', MatchPathMedicareNameDOB -> 'medicare',
// MatchPathNameDOBFacility / MatchPathNoMatch -> 'facility_internal'
// with the queue entry id as the synthetic identifier value (so the
// UNIQUE constraint stays well-defined).
func (s *IdentityStore) persistResolvedMapping(ctx context.Context, entry interfaces.IdentityReviewQueueEntry, resolvedRef, resolvedBy uuid.UUID) error {
	// Decode the IncomingIdentifier blob to recover the original
	// identifiers; we cannot reconstruct them from the queue row alone.
	var incoming identity.IncomingIdentifier
	if len(entry.IncomingIdentifier) > 0 {
		if err := json.Unmarshal(entry.IncomingIdentifier, &incoming); err != nil {
			return fmt.Errorf("unmarshal incoming: %w", err)
		}
	}

	var kind, value string
	switch identity.MatchPath(entry.MatchPath) {
	case identity.MatchPathIHI:
		kind, value = "ihi", incoming.IHI
	case identity.MatchPathMedicareNameDOB:
		kind, value = "medicare", incoming.Medicare
	default:
		// LOW or NONE path. Use the queue entry id as a stable synthetic
		// identifier so reruns of the resolution are idempotent.
		kind, value = "facility_internal", entry.ID.String()
	}
	if value == "" {
		// Fallback when the strongest-tier identifier was empty (e.g.
		// MEDIUM path with empty Medicare — defensive only).
		kind, value = "facility_internal", entry.ID.String()
	}
	now := time.Now().UTC()
	_, err := s.InsertIdentityMapping(ctx, interfaces.IdentityMapping{
		IdentifierKind:  kind,
		IdentifierValue: value,
		ResidentRef:     resolvedRef,
		Confidence:      "low", // verified-by-human, but the underlying signal was LOW
		MatchPath:       entry.MatchPath,
		Source:          entry.Source,
		VerifiedBy:      &resolvedBy,
		VerifiedAt:      &now,
	})
	return err
}

// writeResolutionEvidenceTrace records a second EvidenceTrace node
// capturing the resolution event itself, with a derived_from edge
// linking it to the original match-decision node so the graph
// preserves the full audit chain.
func (s *IdentityStore) writeResolutionEvidenceTrace(ctx context.Context, entry interfaces.IdentityReviewQueueEntry, resolvedRef, resolvedBy uuid.UUID, rerouted int) error {
	now := time.Now().UTC()
	nodeID := uuid.New()
	rs := &models.ReasoningSummary{
		Text:      fmt.Sprintf("identity_review resolution: rerouted=%d", rerouted),
		RuleFires: []string{"identity_match_resolution"},
	}
	node := models.EvidenceTraceNode{
		ID:              nodeID,
		StateMachine:    models.EvidenceTraceStateMachineClinicalState,
		StateChangeType: "identity_match_resolved",
		RecordedAt:      now,
		OccurredAt:      now,
		Actor: models.TraceActor{
			PersonRef: &resolvedBy,
		},
		Outputs: []models.TraceOutput{
			{OutputType: "Resident", OutputRef: resolvedRef},
		},
		ReasoningSummary: rs,
		ResidentRef:      &resolvedRef,
		CreatedAt:        now,
	}
	if _, err := s.v2.UpsertEvidenceTraceNode(ctx, node); err != nil {
		return err
	}
	// Edge: resolution derived_from the original match-decision node (if any).
	if entry.EvidenceTraceNodeRef != nil {
		if err := s.v2.InsertEvidenceTraceEdge(ctx, evidence_trace.Edge{
			From: nodeID,
			To:   *entry.EvidenceTraceNodeRef,
			Kind: evidence_trace.EdgeKindDerivedFrom,
		}); err != nil {
			return fmt.Errorf("insert resolution edge: %w", err)
		}
	}
	return nil
}

// ---------------------------------------------------------------------------
// Compile-time interface assertions
// ---------------------------------------------------------------------------

var (
	_ identity.IdentityCandidateLookup    = (*IdentityStore)(nil)
	_ interfaces.IdentityMappingStore     = (*IdentityStore)(nil)
	_ interfaces.IdentityReviewQueueStore = (*IdentityStore)(nil)
)
