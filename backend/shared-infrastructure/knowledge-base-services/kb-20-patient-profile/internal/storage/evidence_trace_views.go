// Package storage — Wave 5.1: read bindings for the EvidenceTrace
// materialised views (migration 022). Keep these read-only; the views are
// computed by refresh_evidence_trace_views() in PostgreSQL.
package storage

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/lib/pq"
)

// RecommendationLineageView is one row of mv_recommendation_lineage.
//
// All slice fields are typed as []uuid.UUID so callers don't have to deal
// with pq.UUIDArray scan plumbing. Empty slices are represented as nil.
type RecommendationLineageView struct {
	RecommendationID         uuid.UUID   `json:"recommendation_id"`
	ResidentRef              *uuid.UUID  `json:"resident_ref,omitempty"`
	UpstreamEvidenceCount    int         `json:"upstream_evidence_count"`
	UpstreamObservationRefs  []uuid.UUID `json:"upstream_observation_refs,omitempty"`
	UpstreamEventRefs        []uuid.UUID `json:"upstream_event_refs,omitempty"`
	DecisionOutcome          string      `json:"decision_outcome"`
	DownstreamOutcomeRefs    []uuid.UUID `json:"downstream_outcome_refs,omitempty"`
}

// ObservationConsequencesView is one row of mv_observation_consequences.
type ObservationConsequencesView struct {
	ObservationID                 uuid.UUID   `json:"observation_id"`
	ResidentRef                   *uuid.UUID  `json:"resident_ref,omitempty"`
	DownstreamRecommendationCount int         `json:"downstream_recommendation_count"`
	DownstreamRecommendations     []uuid.UUID `json:"downstream_recommendations,omitempty"`
	DownstreamActedCount          int         `json:"downstream_acted_count"`
}

// ResidentReasoningSummaryView is one row of mv_resident_reasoning_summary.
type ResidentReasoningSummaryView struct {
	ResidentRef                       uuid.UUID `json:"resident_ref"`
	Last30DRecommendationCount        int       `json:"last_30d_recommendation_count"`
	Last30DDecisionCount              int       `json:"last_30d_decision_count"`
	AverageEvidencePerRecommendation  float64   `json:"average_evidence_per_recommendation"`
}

// EvidenceTraceViewsStore is the read-only binding to the Wave 5.1
// materialised views.
type EvidenceTraceViewsStore struct {
	db *sql.DB
}

// NewEvidenceTraceViewsStore wraps a *sql.DB for the views.
func NewEvidenceTraceViewsStore(db *sql.DB) *EvidenceTraceViewsStore {
	return &EvidenceTraceViewsStore{db: db}
}

// ErrViewRowNotFound is returned by single-row lookups when no view row
// matches.
var ErrViewRowNotFound = errors.New("evidence_trace_views: row not found")

// GetRecommendationLineage reads one mv_recommendation_lineage row by id.
func (s *EvidenceTraceViewsStore) GetRecommendationLineage(ctx context.Context, recID uuid.UUID) (*RecommendationLineageView, error) {
	const q = `
		SELECT recommendation_id, resident_ref,
		       upstream_evidence_count, upstream_observation_refs,
		       upstream_event_refs, decision_outcome,
		       downstream_outcome_refs
		FROM mv_recommendation_lineage
		WHERE recommendation_id = $1
	`
	var (
		out          RecommendationLineageView
		residentRef  sql.NullString
		obsArr       pq.StringArray
		evtArr       pq.StringArray
		outcomeArr   pq.StringArray
		decisionOut  sql.NullString
	)
	err := s.db.QueryRowContext(ctx, q, recID).Scan(
		&out.RecommendationID, &residentRef,
		&out.UpstreamEvidenceCount, &obsArr, &evtArr,
		&decisionOut, &outcomeArr,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrViewRowNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query mv_recommendation_lineage: %w", err)
	}
	if residentRef.Valid {
		if u, e := uuid.Parse(residentRef.String); e == nil {
			out.ResidentRef = &u
		}
	}
	if decisionOut.Valid {
		out.DecisionOutcome = decisionOut.String
	}
	out.UpstreamObservationRefs = parseUUIDArray(obsArr)
	out.UpstreamEventRefs = parseUUIDArray(evtArr)
	out.DownstreamOutcomeRefs = parseUUIDArray(outcomeArr)
	return &out, nil
}

// GetObservationConsequences reads one mv_observation_consequences row by id.
func (s *EvidenceTraceViewsStore) GetObservationConsequences(ctx context.Context, obsID uuid.UUID) (*ObservationConsequencesView, error) {
	const q = `
		SELECT observation_id, resident_ref,
		       downstream_recommendation_count,
		       downstream_recommendations, downstream_acted_count
		FROM mv_observation_consequences
		WHERE observation_id = $1
	`
	var (
		out         ObservationConsequencesView
		residentRef sql.NullString
		recArr      pq.StringArray
	)
	err := s.db.QueryRowContext(ctx, q, obsID).Scan(
		&out.ObservationID, &residentRef,
		&out.DownstreamRecommendationCount, &recArr,
		&out.DownstreamActedCount,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrViewRowNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query mv_observation_consequences: %w", err)
	}
	if residentRef.Valid {
		if u, e := uuid.Parse(residentRef.String); e == nil {
			out.ResidentRef = &u
		}
	}
	out.DownstreamRecommendations = parseUUIDArray(recArr)
	return &out, nil
}

// GetResidentReasoningSummary reads one mv_resident_reasoning_summary row.
func (s *EvidenceTraceViewsStore) GetResidentReasoningSummary(ctx context.Context, residentRef uuid.UUID) (*ResidentReasoningSummaryView, error) {
	const q = `
		SELECT resident_ref, last_30d_recommendation_count,
		       last_30d_decision_count, average_evidence_per_recommendation
		FROM mv_resident_reasoning_summary
		WHERE resident_ref = $1
	`
	var out ResidentReasoningSummaryView
	err := s.db.QueryRowContext(ctx, q, residentRef).Scan(
		&out.ResidentRef, &out.Last30DRecommendationCount,
		&out.Last30DDecisionCount, &out.AverageEvidencePerRecommendation,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrViewRowNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("query mv_resident_reasoning_summary: %w", err)
	}
	return &out, nil
}

// RefreshEvidenceTraceViews invokes the refresh_evidence_trace_views()
// stored procedure shipped by migration 022. Suitable for a worker process
// or a manual invocation by ops.
//
// Production refresh strategy lock-in is a TODO for V1 (see migration 022
// header for the candidate strategies).
func (s *EvidenceTraceViewsStore) RefreshEvidenceTraceViews(ctx context.Context) error {
	_, err := s.db.ExecContext(ctx, `SELECT refresh_evidence_trace_views()`)
	if err != nil {
		return fmt.Errorf("refresh_evidence_trace_views: %w", err)
	}
	return nil
}

// parseUUIDArray converts a pq.StringArray to []uuid.UUID, dropping any
// element that fails to parse. Returns nil for empty / all-bad input so
// JSON marshalling produces "null"/"[]" cleanly.
func parseUUIDArray(in pq.StringArray) []uuid.UUID {
	if len(in) == 0 {
		return nil
	}
	out := make([]uuid.UUID, 0, len(in))
	for _, s := range in {
		if s == "" {
			continue
		}
		if u, err := uuid.Parse(s); err == nil {
			out = append(out, u)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}
