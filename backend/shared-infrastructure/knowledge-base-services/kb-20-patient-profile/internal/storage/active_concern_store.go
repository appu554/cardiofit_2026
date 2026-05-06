// Package storage — ActiveConcernStore is the kb-20 implementation of the
// active-concern lifecycle persistence layer (Layer 2 substrate plan, Wave
// 2.3). It owns the active_concerns + concern_type_triggers tables created
// by migration 015 and is consumed by:
//
//   - api/active_concern_handlers.go (REST CRUD + lifecycle)
//   - main.go cron loop (calls ListExpiringConcerns then engine.SweepExpired)
//   - storage/baseline_store.go (consults ListActiveByResidentAndType for
//     the wave-2.2 ExcludeDuringActiveConcerns filter)
//
// The store is intentionally narrow: status-transition correctness is
// checked at the application boundary via
// validation.ValidateActiveConcernResolutionTransition before the UPDATE
// is dispatched. The DB-level CHECK constraint is a backstop, not the
// primary defence.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"

	"github.com/cardiofit/shared/v2_substrate/clinical_state"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"
)

// ActiveConcernStore implements interfaces.ActiveConcernStore +
// interfaces.ConcernTriggerLookupStore against the active_concerns and
// concern_type_triggers tables (migration 015).
type ActiveConcernStore struct {
	db *sql.DB
}

// NewActiveConcernStore wires a *sql.DB into the ActiveConcern persistence
// contract. The caller owns the database lifecycle.
func NewActiveConcernStore(db *sql.DB) *ActiveConcernStore {
	return &ActiveConcernStore{db: db}
}

const activeConcernColumns = `id, resident_id, concern_type, started_at,
       started_by_event_ref, expected_resolution_at, owner_role_ref,
       related_monitoring_plan_ref, resolution_status, resolved_at,
       resolution_evidence_trace_ref, notes, created_at, updated_at`

func scanActiveConcern(sc rowScanner) (models.ActiveConcern, error) {
	var (
		c            models.ActiveConcern
		startedBy    uuid.NullUUID
		ownerRole    uuid.NullUUID
		relatedPlan  uuid.NullUUID
		resolvedAt   sql.NullTime
		evidenceRef  uuid.NullUUID
		notes        sql.NullString
	)
	if err := sc.Scan(
		&c.ID, &c.ResidentID, &c.ConcernType, &c.StartedAt,
		&startedBy, &c.ExpectedResolutionAt, &ownerRole,
		&relatedPlan, &c.ResolutionStatus, &resolvedAt,
		&evidenceRef, &notes, &c.CreatedAt, &c.UpdatedAt,
	); err != nil {
		return models.ActiveConcern{}, err
	}
	if startedBy.Valid {
		u := startedBy.UUID
		c.StartedByEventRef = &u
	}
	if ownerRole.Valid {
		u := ownerRole.UUID
		c.OwnerRoleRef = &u
	}
	if relatedPlan.Valid {
		u := relatedPlan.UUID
		c.RelatedMonitoringPlanRef = &u
	}
	if resolvedAt.Valid {
		t := resolvedAt.Time
		c.ResolvedAt = &t
	}
	if evidenceRef.Valid {
		u := evidenceRef.UUID
		c.ResolutionEvidenceTraceRef = &u
	}
	if notes.Valid {
		c.Notes = notes.String
	}
	return c, nil
}

// CreateActiveConcern inserts a new row. The supplied entity is validated
// before INSERT. ID is generated server-side when c.ID == uuid.Nil so the
// caller can omit it in REST payloads.
func (s *ActiveConcernStore) CreateActiveConcern(ctx context.Context, c models.ActiveConcern) (*models.ActiveConcern, error) {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	if c.ResolutionStatus == "" {
		c.ResolutionStatus = models.ResolutionStatusOpen
	}
	if err := validation.ValidateActiveConcern(c); err != nil {
		return nil, fmt.Errorf("validate: %w", err)
	}

	const q = `
		INSERT INTO active_concerns
			(id, resident_id, concern_type, started_at, started_by_event_ref,
			 expected_resolution_at, owner_role_ref, related_monitoring_plan_ref,
			 resolution_status, resolved_at, resolution_evidence_trace_ref, notes,
			 created_at, updated_at)
		VALUES
			($1, $2, $3, $4, $5,
			 $6, $7, $8,
			 $9, $10, $11, $12,
			 NOW(), NOW())`

	var startedByArg, ownerArg, planArg, evidenceArg interface{}
	if c.StartedByEventRef != nil {
		startedByArg = *c.StartedByEventRef
	}
	if c.OwnerRoleRef != nil {
		ownerArg = *c.OwnerRoleRef
	}
	if c.RelatedMonitoringPlanRef != nil {
		planArg = *c.RelatedMonitoringPlanRef
	}
	if c.ResolutionEvidenceTraceRef != nil {
		evidenceArg = *c.ResolutionEvidenceTraceRef
	}
	var resolvedArg interface{}
	if c.ResolvedAt != nil {
		resolvedArg = *c.ResolvedAt
	}

	if _, err := s.db.ExecContext(ctx, q,
		c.ID, c.ResidentID, c.ConcernType, c.StartedAt, startedByArg,
		c.ExpectedResolutionAt, ownerArg, planArg,
		c.ResolutionStatus, resolvedArg, evidenceArg, nilIfEmpty(c.Notes),
	); err != nil {
		return nil, fmt.Errorf("insert active_concern: %w", err)
	}
	return s.GetActiveConcern(ctx, c.ID)
}

// GetActiveConcern reads a single ActiveConcern by primary key.
func (s *ActiveConcernStore) GetActiveConcern(ctx context.Context, id uuid.UUID) (*models.ActiveConcern, error) {
	q := `SELECT ` + activeConcernColumns + ` FROM active_concerns WHERE id = $1`
	c, err := scanActiveConcern(s.db.QueryRowContext(ctx, q, id))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("get active_concern %s: %w", id, interfaces.ErrNotFound)
		}
		return nil, fmt.Errorf("get active_concern %s: %w", id, err)
	}
	return &c, nil
}

// ListActiveConcernsByResident returns all concerns for a resident,
// newest-first by started_at. Filter by resolution_status when status is
// non-empty.
func (s *ActiveConcernStore) ListActiveConcernsByResident(ctx context.Context, residentID uuid.UUID, status string) ([]models.ActiveConcern, error) {
	var (
		q    string
		args []interface{}
	)
	if status == "" {
		q = `SELECT ` + activeConcernColumns + `
			   FROM active_concerns
			  WHERE resident_id = $1
			  ORDER BY started_at DESC`
		args = []interface{}{residentID}
	} else {
		q = `SELECT ` + activeConcernColumns + `
			   FROM active_concerns
			  WHERE resident_id = $1 AND resolution_status = $2
			  ORDER BY started_at DESC`
		args = []interface{}{residentID, status}
	}
	rows, err := s.db.QueryContext(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ActiveConcern
	for rows.Next() {
		c, err := scanActiveConcern(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListActiveByResidentAndType returns the open concerns for a resident
// matching any of the supplied concern types. Used by the baseline
// exclusion path. Empty types slice returns nil.
func (s *ActiveConcernStore) ListActiveByResidentAndType(ctx context.Context, residentID uuid.UUID, types []string) ([]models.ActiveConcern, error) {
	if len(types) == 0 {
		return nil, nil
	}
	q := `SELECT ` + activeConcernColumns + `
		    FROM active_concerns
		   WHERE resident_id = $1
			 AND resolution_status = 'open'
			 AND concern_type = ANY($2::text[])
		   ORDER BY started_at DESC`
	rows, err := s.db.QueryContext(ctx, q, residentID, pq.Array(types))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ActiveConcern
	for rows.Next() {
		c, err := scanActiveConcern(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ListExpiringConcerns returns open concerns whose expected_resolution_at
// is within `within` of now. Pass within=0 to find concerns already past
// their deadline.
func (s *ActiveConcernStore) ListExpiringConcerns(ctx context.Context, within time.Duration) ([]models.ActiveConcern, error) {
	q := `SELECT ` + activeConcernColumns + `
		    FROM active_concerns
		   WHERE resolution_status = 'open'
			 AND expected_resolution_at < NOW() + ($1::text)::interval
		   ORDER BY expected_resolution_at ASC`
	// Pass duration as a string interval to avoid numeric/interval coercion
	// surprises across pq driver versions.
	rows, err := s.db.QueryContext(ctx, q, fmt.Sprintf("%d seconds", int64(within.Seconds())))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []models.ActiveConcern
	for rows.Next() {
		c, err := scanActiveConcern(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// UpdateResolution transitions an open ActiveConcern to a terminal status.
// Validates the transition before issuing the UPDATE; rejects calls
// whose current status is terminal (i.e. attempts to "reopen" or move
// between terminal states).
func (s *ActiveConcernStore) UpdateResolution(ctx context.Context, id uuid.UUID, status string, resolvedAt time.Time, evidenceTraceRef *uuid.UUID) (*models.ActiveConcern, error) {
	current, err := s.GetActiveConcern(ctx, id)
	if err != nil {
		return nil, err
	}
	if err := validation.ValidateActiveConcernResolutionTransition(current.ResolutionStatus, status); err != nil {
		return nil, fmt.Errorf("transition: %w", err)
	}
	if resolvedAt.IsZero() {
		return nil, errors.New("resolved_at is required")
	}
	if resolvedAt.Before(current.StartedAt) {
		return nil, errors.New("resolved_at must be >= started_at")
	}

	const q = `UPDATE active_concerns SET
		resolution_status              = $2,
		resolved_at                    = $3,
		resolution_evidence_trace_ref  = $4,
		updated_at                     = NOW()
	  WHERE id = $1
		AND resolution_status = 'open'`
	var evidenceArg interface{}
	if evidenceTraceRef != nil {
		evidenceArg = *evidenceTraceRef
	}
	res, err := s.db.ExecContext(ctx, q, id, status, resolvedAt, evidenceArg)
	if err != nil {
		return nil, fmt.Errorf("update active_concern resolution: %w", err)
	}
	rows, _ := res.RowsAffected()
	if rows == 0 {
		// Either the row no longer exists or someone else closed it
		// between our read and write. Surface as not-found.
		return nil, fmt.Errorf("update active_concern %s: %w", id, interfaces.ErrNotFound)
	}
	return s.GetActiveConcern(ctx, id)
}

// ============================================================================
// ConcernTriggerLookupStore — lookups against concern_type_triggers
// ============================================================================

// LookupConcernTriggersByEventType returns the trigger entries whose
// trigger_event_type matches eventType. Mirrors the engine's
// LookupByEventType signature.
func (s *ActiveConcernStore) LookupConcernTriggersByEventType(ctx context.Context, eventType string) ([]interfaces.ConcernTriggerEntry, error) {
	const q = `SELECT concern_type, default_window_hours
				 FROM concern_type_triggers
				WHERE trigger_event_type = $1`
	rows, err := s.db.QueryContext(ctx, q, eventType)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []interfaces.ConcernTriggerEntry
	for rows.Next() {
		var e interfaces.ConcernTriggerEntry
		if err := rows.Scan(&e.ConcernType, &e.DefaultWindowHours); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// LookupConcernTriggersByMedATC matches any concern_type_triggers row
// whose trigger_med_atc is a prefix of atc AND whose trigger_med_intent
// is either NULL or equal to intent. Performs the prefix match in SQL
// via LEFT(...) so the index on trigger_med_atc is usable when the
// caller passes the full ATC code.
func (s *ActiveConcernStore) LookupConcernTriggersByMedATC(ctx context.Context, atc, intent string) ([]interfaces.ConcernTriggerEntry, error) {
	const q = `SELECT concern_type, default_window_hours
				 FROM concern_type_triggers
				WHERE trigger_med_atc IS NOT NULL
				  AND $1 LIKE trigger_med_atc || '%'
				  AND (trigger_med_intent IS NULL OR trigger_med_intent = $2)`
	rows, err := s.db.QueryContext(ctx, q, atc, intent)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []interfaces.ConcernTriggerEntry
	for rows.Next() {
		var e interfaces.ConcernTriggerEntry
		if err := rows.Scan(&e.ConcernType, &e.DefaultWindowHours); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// ============================================================================
// Engine adapter
// ============================================================================

// AsEngineLookup adapts an ActiveConcernStore to the
// clinical_state.ConcernTriggerLookup interface so the pure engine can
// drive lookups against the persistent table without depending on the
// interfaces package directly.
func (s *ActiveConcernStore) AsEngineLookup() clinical_state.ConcernTriggerLookup {
	return &concernTriggerEngineAdapter{store: s}
}

type concernTriggerEngineAdapter struct {
	store *ActiveConcernStore
}

func (a *concernTriggerEngineAdapter) LookupByEventType(ctx context.Context, eventType string) ([]clinical_state.TriggerEntry, error) {
	rows, err := a.store.LookupConcernTriggersByEventType(ctx, eventType)
	if err != nil {
		return nil, err
	}
	out := make([]clinical_state.TriggerEntry, len(rows))
	for i, r := range rows {
		out[i] = clinical_state.TriggerEntry{ConcernType: r.ConcernType, DefaultWindowHours: r.DefaultWindowHours}
	}
	return out, nil
}

func (a *concernTriggerEngineAdapter) LookupByMedATC(ctx context.Context, atc, intent string) ([]clinical_state.TriggerEntry, error) {
	rows, err := a.store.LookupConcernTriggersByMedATC(ctx, atc, intent)
	if err != nil {
		return nil, err
	}
	out := make([]clinical_state.TriggerEntry, len(rows))
	for i, r := range rows {
		out[i] = clinical_state.TriggerEntry{ConcernType: r.ConcernType, DefaultWindowHours: r.DefaultWindowHours}
	}
	return out, nil
}

// Compile-time assertions.
var (
	_ interfaces.ActiveConcernStore        = (*ActiveConcernStore)(nil)
	_ interfaces.ConcernTriggerLookupStore = (*ActiveConcernStore)(nil)
)
