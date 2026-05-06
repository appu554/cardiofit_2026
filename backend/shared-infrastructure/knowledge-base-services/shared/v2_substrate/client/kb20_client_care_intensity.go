package client

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
)

// CreateCareIntensityRequest is the payload for POST
// /v2/residents/:id/care-intensity. Mirrors the handler's request body.
// ResidentRef is taken from the path; the body's ResidentRef field, if
// any, is ignored.
type CreateCareIntensityRequest struct {
	Tag                 string          `json:"tag"`
	EffectiveDate       time.Time       `json:"effective_date"`
	DocumentedByRoleRef uuid.UUID       `json:"documented_by_role_ref"`
	ReviewDueDate       *time.Time      `json:"review_due_date,omitempty"`
	RationaleStructured json.RawMessage `json:"rationale_structured,omitempty"`
	RationaleFreeText   string          `json:"rationale_free_text,omitempty"`
}

// CreateCareIntensity records a new care-intensity transition for
// residentID. Returns the persisted CareIntensity row, the transition
// Event, and the cascade hints surfaced by the engine.
func (c *KB20Client) CreateCareIntensity(ctx context.Context, residentID uuid.UUID, req CreateCareIntensityRequest) (*interfaces.CareIntensityTransitionResult, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/care-intensity"
	return doJSON[interfaces.CareIntensityTransitionResult](ctx, c.http, http.MethodPost, u, req)
}

// GetCurrentCareIntensity returns the latest CareIntensity row for
// residentID. The HTTP layer returns 404 (interfaces.ErrNotFound at the
// store boundary) when the resident has no history rows yet.
func (c *KB20Client) GetCurrentCareIntensity(ctx context.Context, residentID uuid.UUID) (*models.CareIntensity, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/care-intensity/current"
	return doJSON[models.CareIntensity](ctx, c.http, http.MethodGet, u, nil)
}

// ListCareIntensityHistory returns the full history for residentID,
// newest-first by effective_date.
func (c *KB20Client) ListCareIntensityHistory(ctx context.Context, residentID uuid.UUID) ([]models.CareIntensity, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/care-intensity/history"
	out, err := doJSON[[]models.CareIntensity](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}
