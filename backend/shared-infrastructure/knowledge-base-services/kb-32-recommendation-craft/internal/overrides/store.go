// Package overrides implements the override-reason taxonomy for clinical-safety
// audit capture.
//
// VisibilityClass: AD — override audit per Guidelines §5
//
// This file provides the persistence boundary (Store interface) and two
// implementations: PostgresStore (production) and InMemoryStore (tests/dev).
// The InMemoryStore uses sync.RWMutex per Phase 1a convention.
package overrides

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/failed_interventions"
)

// ---------------------------------------------------------------------------
// Store interface
// ---------------------------------------------------------------------------

// Store is the persistence boundary for OverrideReason records.
// Implementations must be safe for concurrent use.
type Store interface {
	// Create persists r and returns the record with ID populated.
	Create(ctx context.Context, r OverrideReason) (OverrideReason, error)

	// Get returns the OverrideReason for the given id, or (zero, ErrNotFound)
	// when no record exists.
	Get(ctx context.Context, id string) (OverrideReason, error)

	// ListByRule returns all overrides whose recommendation belongs to the
	// given ruleID. The order is by CapturedAt ascending.
	//
	// Note: this requires the persistence layer to know which recommendations
	// belong to which rules. The Postgres implementation joins against the
	// recommendations table (rule_id column assumed per Plan 0.1). The InMemory
	// implementation uses a ruleID field embedded in test OverrideReasons via
	// the RuleID auxiliary field.
	ListByRule(ctx context.Context, ruleID string) ([]OverrideReason, error)

	// PatternSummary returns a count of override records for ruleID in the
	// time window [since, now), keyed by AppropriatenessFlag:
	//   "appropriate_override" → count
	//   "inappropriate_override" → count
	//   "mixed"                  → count
	//
	// Keys with zero count are omitted from the map. The Task 4 feedback loop
	// uses this exact shape to determine whether a rule should be flagged for
	// tuning review.
	PatternSummary(ctx context.Context, ruleID string, since time.Time) (map[string]int, error)
}

// ErrNotFound is returned by Get when no record exists for the given id.
var ErrNotFound = fmt.Errorf("overrides: record not found")

// ---------------------------------------------------------------------------
// Compile-time interface assertions
// ---------------------------------------------------------------------------

var _ Store = (*PostgresStore)(nil)
var _ Store = (*InMemoryStore)(nil)

// ---------------------------------------------------------------------------
// InMemoryStore
// ---------------------------------------------------------------------------

// storedOverride extends OverrideReason with an internal RuleID field used by
// the InMemory implementation to support ListByRule and PatternSummary without
// a real recommendations table.
type storedOverride struct {
	OverrideReason
	RuleID string
}

// InMemoryStore is a thread-safe in-memory Store intended for testing and
// development. It is NOT suitable for production use — data is lost on restart.
//
// The RuleID is stored as a separate field on the internal record; callers
// should use CreateForRule (instead of Create) when they need ListByRule to
// work correctly. Create stores an empty RuleID.
//
// firStore (optional) is the Failed Intervention History writer. When non-nil,
// the rule-aware createInternal path (i.e. CreateForRule with a non-empty
// ruleID) will attempt a best-effort FIR write after the override is persisted
// — see createInternal for the hook semantics. The plain Create path does NOT
// attempt FIR writes because it lacks rule context (the FIR
// InterventionType vocabulary is derived from the rule ID via
// failed_interventions.ClassifyInterventionType).
type InMemoryStore struct {
	mu       sync.RWMutex
	records  map[string]storedOverride
	firStore failed_interventions.Store
}

// NewInMemoryStore returns an empty, ready-to-use InMemoryStore.
func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{records: make(map[string]storedOverride)}
}

// WithFailedInterventionStore wires the FIR writer used by the rule-aware
// CreateForRule path. Safe to call once at construction; not safe for
// concurrent reconfiguration during request handling.
func (s *InMemoryStore) WithFailedInterventionStore(fir failed_interventions.Store) *InMemoryStore {
	s.firStore = fir
	return s
}

// Create persists r, assigning a new UUID as ID and setting CapturedAt if
// zero. Returns the populated record.
func (s *InMemoryStore) Create(ctx context.Context, r OverrideReason) (OverrideReason, error) {
	return s.createInternal(ctx, r, "")
}

// CreateForRule persists r associated with ruleID. This is a test-helper that
// allows ListByRule and PatternSummary to work correctly without a real
// recommendations table.
func (s *InMemoryStore) CreateForRule(ctx context.Context, r OverrideReason, ruleID string) (OverrideReason, error) {
	return s.createInternal(ctx, r, ruleID)
}

func (s *InMemoryStore) createInternal(ctx context.Context, r OverrideReason, ruleID string) (OverrideReason, error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.CapturedAt.IsZero() {
		r.CapturedAt = time.Now().UTC()
	}
	// Mirror PostgresStore: backfill the missing vocabulary form so reads
	// always return both ReasonCode and ReasonCodeShort populated.
	if r.ReasonCodeShort == "" {
		if short, ok := ToShortCode(r.ReasonCode); ok {
			r.ReasonCodeShort = short
		}
	} else if r.ReasonCode == "" {
		if snake, ok := ToReasonCode(r.ReasonCodeShort); ok {
			r.ReasonCode = snake
		}
	}
	s.mu.Lock()
	s.records[r.ID] = storedOverride{OverrideReason: r, RuleID: ruleID}
	s.mu.Unlock()

	// CAPE substrate hook: when called via CreateForRule with a non-empty
	// ruleID AND the override's ReasonCode signals a documented reversal
	// (one of the failure-outcome ACOP extensions), write a
	// FailedInterventionRecord. Best-effort: log on error but do NOT fail
	// the override write — override capture is the primary audit path.
	//
	// Known gap: ResidentID is not carried on OverrideReason in Phase 2-
	// completion (the HTTP boundary at /v1/craft/override/:recommendation_id
	// only receives recommendation_id, not resident_id). Until the override
	// flow is widened to carry ResidentID, FIR rows are written with
	// uuid.Nil; CAPE Layer 4 readers must resolve the resident via the
	// recommendations table join. This is acceptable substrate-prerequisite
	// behaviour; kb-33 hardens the join.
	if ruleID != "" && s.firStore != nil {
		writeFIRBestEffort(ctx, s.firStore, r, ruleID)
	}
	return r, nil
}

// reversalReasonCodes is the set of override ReasonCodes that classify the
// override as a documented clinical reversal — these populate the CAPE
// Layer 4 Failed Intervention History. All are ACOP extensions from the
// taxonomy (see taxonomy.go); Wright/McCoy foundation codes (alert fatigue,
// workflow constraint, etc.) describe alert handling, NOT clinical
// reversal, and so do NOT generate FIR rows.
var reversalReasonCodes = map[string]string{
	"goals_of_care_aligned":    failed_interventions.OutcomeGoalsOfCareAligned,
	"frailty_consideration":    failed_interventions.OutcomeReversedDueToFrailty,
	"deprescribing_underway":   failed_interventions.OutcomeReversedDueToClinicalDecline,
	"family_consensus_pending": failed_interventions.OutcomeReversedDueToFamilyRequest,
	"trial_period_active":      failed_interventions.OutcomeReversedDueToClinicalDecline,
}

// writeFIRBestEffort attempts a CAPE Layer-4 FIR write. Errors are logged
// (not propagated) per the design note in createInternal / PostgresStore
// CreateForRule. ctx may be context.Background() for the InMemoryStore path
// since the in-memory implementation ignores it.
func writeFIRBestEffort(ctx context.Context, fir failed_interventions.Store, r OverrideReason, ruleID string) {
	interventionType, classified := failed_interventions.ClassifyInterventionType(ruleID)
	if !classified {
		return
	}
	outcome, ok := reversalReasonCodes[r.ReasonCode]
	if !ok {
		return
	}
	authorID, err := uuid.Parse(r.CapturedBy)
	if err != nil {
		authorID = uuid.Nil
	}
	now := time.Now().UTC()
	rec := failed_interventions.FailedInterventionRecord{
		// ResidentID intentionally uuid.Nil — see Known gap note in
		// createInternal. The OverrideReason struct does not currently
		// carry ResidentID; widening it is out of scope for this task.
		ResidentID:        uuid.Nil,
		InterventionType:  interventionType,
		AttemptDate:       now,
		Outcome:           outcome,
		DocumentedReason:  r.Reasoning,
		RetryEligibleDate: now.Add(failed_interventions.DefaultRetryWindow),
		DocumentedBy:      authorID,
	}
	if err := fir.Record(ctx, rec); err != nil {
		log.Printf("overrides: failed to record FailedInterventionRecord (rule=%s, reason=%s): %v",
			ruleID, r.ReasonCode, err)
	}
}

// Get returns the OverrideReason for id, or (zero, ErrNotFound) if absent.
func (s *InMemoryStore) Get(_ context.Context, id string) (OverrideReason, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	rec, ok := s.records[id]
	if !ok {
		return OverrideReason{}, ErrNotFound
	}
	return rec.OverrideReason, nil
}

// ListByRule returns all overrides associated with ruleID, ordered by
// CapturedAt ascending.
func (s *InMemoryStore) ListByRule(_ context.Context, ruleID string) ([]OverrideReason, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	var out []OverrideReason
	for _, rec := range s.records {
		if rec.RuleID == ruleID {
			out = append(out, rec.OverrideReason)
		}
	}
	// Sort by CapturedAt ascending (simple insertion sort is fine for tests).
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].CapturedAt.Before(out[j-1].CapturedAt); j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out, nil
}

// PatternSummary counts overrides for ruleID since the given time, keyed by
// AppropriatenessFlag. Zero-count flags are omitted.
func (s *InMemoryStore) PatternSummary(_ context.Context, ruleID string, since time.Time) (map[string]int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	counts := make(map[string]int)
	for _, rec := range s.records {
		if rec.RuleID != ruleID {
			continue
		}
		if rec.CapturedAt.Before(since) {
			continue
		}
		counts[rec.AppropriatenessFlag]++
	}
	return counts, nil
}

// ---------------------------------------------------------------------------
// PostgresStore
// ---------------------------------------------------------------------------

// PostgresStore is a production-grade Store backed by a PostgreSQL database.
// It requires migration 042 to have been applied (table
// recommendation_override_reasons and materialised view rule_override_patterns).
//
// The materialised view rule_override_patterns is used by PatternSummary for
// bulk aggregation; for small windows the direct table query is used instead.
//
// firStore (optional) is the Failed Intervention History writer, mirroring
// InMemoryStore's hook. CAPE substrate writes only flow from CreateForRule;
// the plain Create path lacks rule context and is intentionally inert
// w.r.t. FIR.
type PostgresStore struct {
	db       *sql.DB
	firStore failed_interventions.Store
}

// NewPostgresStore constructs a PostgresStore from an open *sql.DB.
// The caller retains ownership of db and must close it after use.
func NewPostgresStore(db *sql.DB) *PostgresStore {
	return &PostgresStore{db: db}
}

// WithFailedInterventionStore wires the FIR writer used by the rule-aware
// CreateForRule path. Safe to call once at construction.
func (s *PostgresStore) WithFailedInterventionStore(fir failed_interventions.Store) *PostgresStore {
	s.firStore = fir
	return s
}

// CreateForRule persists r and, when applicable, writes a CAPE Failed
// Intervention History row. The override insert reuses Create's path; the
// FIR write is best-effort (logged but non-fatal). The recommendations
// table is expected to already carry the (recommendation_id, rule_id)
// linkage so ListByRule and PatternSummary continue to function against
// the plain Create path — this CreateForRule overload exists to give the
// CAPE substrate writer access to ruleID at insert time. See InMemoryStore
// docstring for the symmetric design.
func (s *PostgresStore) CreateForRule(ctx context.Context, r OverrideReason, ruleID string) (OverrideReason, error) {
	out, err := s.Create(ctx, r)
	if err != nil {
		return out, err
	}
	if ruleID != "" && s.firStore != nil {
		writeFIRBestEffort(ctx, s.firStore, out, ruleID)
	}
	return out, nil
}

// Create inserts r into recommendation_override_reasons, assigning a new UUID
// and setting captured_at to NOW() if the caller left CapturedAt zero.
// Returns the persisted record with ID populated.
func (s *PostgresStore) Create(ctx context.Context, r OverrideReason) (OverrideReason, error) {
	if r.ID == "" {
		r.ID = uuid.New().String()
	}
	if r.CapturedAt.IsZero() {
		r.CapturedAt = time.Now().UTC()
	}
	// Defensive normalisation: if the caller skipped Validate() and only set
	// ReasonCode, derive ReasonCodeShort here so the NOT NULL column is
	// satisfied. If only ReasonCodeShort is set, derive ReasonCode in turn.
	// Inconsistent pairs are left as-is and will be rejected by the DB CHECK.
	if r.ReasonCodeShort == "" {
		if short, ok := ToShortCode(r.ReasonCode); ok {
			r.ReasonCodeShort = short
		}
	} else if r.ReasonCode == "" {
		if snake, ok := ToReasonCode(r.ReasonCodeShort); ok {
			r.ReasonCode = snake
		}
	}

	const q = `
		INSERT INTO recommendation_override_reasons
			(id, recommendation_id, reason_code, reason_code_short,
			 appropriateness_flag, reasoning, captured_at, captured_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, recommendation_id, reason_code, reason_code_short,
			appropriateness_flag, reasoning, captured_at, captured_by`

	row := s.db.QueryRowContext(ctx, q,
		r.ID, r.RecommendationID, r.ReasonCode, r.ReasonCodeShort,
		r.AppropriatenessFlag, r.Reasoning, r.CapturedAt, r.CapturedBy,
	)
	var out OverrideReason
	if err := row.Scan(
		&out.ID, &out.RecommendationID, &out.ReasonCode, &out.ReasonCodeShort,
		&out.AppropriatenessFlag, &out.Reasoning, &out.CapturedAt, &out.CapturedBy,
	); err != nil {
		return OverrideReason{}, fmt.Errorf("overrides: create: %w", err)
	}
	return out, nil
}

// Get returns the OverrideReason for id, or (zero, ErrNotFound) when absent.
func (s *PostgresStore) Get(ctx context.Context, id string) (OverrideReason, error) {
	const q = `
		SELECT id, recommendation_id, reason_code, reason_code_short,
			appropriateness_flag, reasoning, captured_at, captured_by
		FROM recommendation_override_reasons
		WHERE id = $1`

	row := s.db.QueryRowContext(ctx, q, id)
	var out OverrideReason
	err := row.Scan(
		&out.ID, &out.RecommendationID, &out.ReasonCode, &out.ReasonCodeShort,
		&out.AppropriatenessFlag, &out.Reasoning, &out.CapturedAt, &out.CapturedBy,
	)
	if err == sql.ErrNoRows {
		return OverrideReason{}, ErrNotFound
	}
	if err != nil {
		return OverrideReason{}, fmt.Errorf("overrides: get: %w", err)
	}
	return out, nil
}

// ListByRule returns all overrides for recommendations whose rule_id equals
// ruleID, ordered by captured_at ascending. Requires the recommendations
// table (columns: id, rule_id) per Plan 0.1.
func (s *PostgresStore) ListByRule(ctx context.Context, ruleID string) ([]OverrideReason, error) {
	const q = `
		SELECT r.id, r.recommendation_id, r.reason_code, r.reason_code_short,
			r.appropriateness_flag, r.reasoning, r.captured_at, r.captured_by
		FROM recommendation_override_reasons r
		JOIN recommendations rec ON rec.id = r.recommendation_id
		WHERE rec.rule_id = $1
		ORDER BY r.captured_at ASC`

	rows, err := s.db.QueryContext(ctx, q, ruleID)
	if err != nil {
		return nil, fmt.Errorf("overrides: list_by_rule: %w", err)
	}
	defer rows.Close()

	var out []OverrideReason
	for rows.Next() {
		var rec OverrideReason
		if err := rows.Scan(
			&rec.ID, &rec.RecommendationID, &rec.ReasonCode, &rec.ReasonCodeShort,
			&rec.AppropriatenessFlag, &rec.Reasoning, &rec.CapturedAt, &rec.CapturedBy,
		); err != nil {
			return nil, fmt.Errorf("overrides: list_by_rule scan: %w", err)
		}
		out = append(out, rec)
	}
	return out, rows.Err()
}

// PatternSummary queries recommendation_override_reasons directly (not the
// materialised view, which lags) to count overrides for ruleID since the given
// time, grouped by appropriateness_flag. Zero-count flags are omitted.
func (s *PostgresStore) PatternSummary(ctx context.Context, ruleID string, since time.Time) (map[string]int, error) {
	const q = `
		SELECT r.appropriateness_flag, COUNT(*) AS cnt
		FROM recommendation_override_reasons r
		JOIN recommendations rec ON rec.id = r.recommendation_id
		WHERE rec.rule_id = $1
		  AND r.captured_at >= $2
		GROUP BY r.appropriateness_flag`

	rows, err := s.db.QueryContext(ctx, q, ruleID, since)
	if err != nil {
		return nil, fmt.Errorf("overrides: pattern_summary: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var flag string
		var cnt int
		if err := rows.Scan(&flag, &cnt); err != nil {
			return nil, fmt.Errorf("overrides: pattern_summary scan: %w", err)
		}
		counts[flag] = cnt
	}
	return counts, rows.Err()
}
