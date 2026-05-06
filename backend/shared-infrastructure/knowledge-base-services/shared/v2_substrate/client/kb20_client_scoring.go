package client

// KB20 scoring instrument methods (Wave 2.6 of Layer 2 substrate plan;
// Layer 2 doc §2.4 / §2.6). Wraps the four /v2 scoring endpoints exposed
// by kb-20-patient-profile:
//
//   POST /v2/residents/:id/cfs              → CreateCFSScore
//   POST /v2/residents/:id/akps             → CreateAKPSScore
//   GET  /v2/residents/:id/scores/current   → GetCurrentScores
//   GET  /v2/residents/:id/cfs/history      → ListCFSHistory
//   GET  /v2/residents/:id/akps/history     → ListAKPSHistory
//   GET  /v2/residents/:id/dbi/history      → ListDBIHistory
//   GET  /v2/residents/:id/acb/history      → ListACBHistory
//
// DBI / ACB are computed on every MedicineUse write and surfaced via
// /scores/current + /history; there is no POST endpoint for them.

import (
	"context"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// CreateCFSScoreRequest is the body for POST /v2/residents/:id/cfs.
// ResidentRef is taken from the path; the body's residence value (if any)
// is ignored.
type CreateCFSScoreRequest struct {
	AssessedAt        time.Time `json:"assessed_at"`
	AssessorRoleRef   uuid.UUID `json:"assessor_role_ref"`
	InstrumentVersion string    `json:"instrument_version"`
	Score             int       `json:"score"`
	Rationale         string    `json:"rationale,omitempty"`
}

// CreateCFSScore records a new CFS assessment for residentID. Returns the
// persisted row + optional CareIntensityReviewHint (set when Score >= 7).
func (c *KB20Client) CreateCFSScore(ctx context.Context, residentID uuid.UUID, req CreateCFSScoreRequest) (*interfaces.ScoringResult, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/cfs"
	return doJSON[interfaces.ScoringResult](ctx, c.http, http.MethodPost, u, req)
}

// CreateAKPSScoreRequest is the body for POST /v2/residents/:id/akps.
type CreateAKPSScoreRequest struct {
	AssessedAt        time.Time `json:"assessed_at"`
	AssessorRoleRef   uuid.UUID `json:"assessor_role_ref"`
	InstrumentVersion string    `json:"instrument_version"`
	Score             int       `json:"score"`
	Rationale         string    `json:"rationale,omitempty"`
}

// CreateAKPSScore records a new AKPS assessment for residentID. Returns
// the persisted row + optional CareIntensityReviewHint (set when Score <= 40).
func (c *KB20Client) CreateAKPSScore(ctx context.Context, residentID uuid.UUID, req CreateAKPSScoreRequest) (*interfaces.ScoringResult, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/akps"
	return doJSON[interfaces.ScoringResult](ctx, c.http, http.MethodPost, u, req)
}

// GetCurrentScores returns the latest CFS / AKPS / DBI / ACB rows for
// residentID. Any field can be nil when no score of that instrument has
// been recorded for the resident.
func (c *KB20Client) GetCurrentScores(ctx context.Context, residentID uuid.UUID) (*interfaces.CurrentScores, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/scores/current"
	return doJSON[interfaces.CurrentScores](ctx, c.http, http.MethodGet, u, nil)
}

// ListCFSHistory returns the full CFS history for residentID, newest-first.
func (c *KB20Client) ListCFSHistory(ctx context.Context, residentID uuid.UUID) ([]models.CFSScore, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/cfs/history"
	out, err := doJSON[[]models.CFSScore](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ListAKPSHistory returns the full AKPS history for residentID, newest-first.
func (c *KB20Client) ListAKPSHistory(ctx context.Context, residentID uuid.UUID) ([]models.AKPSScore, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/akps/history"
	out, err := doJSON[[]models.AKPSScore](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ListDBIHistory returns the full DBI recompute history for residentID,
// newest-first.
func (c *KB20Client) ListDBIHistory(ctx context.Context, residentID uuid.UUID) ([]models.DBIScore, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/dbi/history"
	out, err := doJSON[[]models.DBIScore](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ListACBHistory returns the full ACB recompute history for residentID,
// newest-first.
func (c *KB20Client) ListACBHistory(ctx context.Context, residentID uuid.UUID) ([]models.ACBScore, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/acb/history"
	out, err := doJSON[[]models.ACBScore](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}
