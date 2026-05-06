package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// CreateActiveConcernRequest is the payload for POST
// /v2/residents/:id/active-concerns. Mirrors the handler's
// createForResidentBody.
type CreateActiveConcernRequest struct {
	ConcernType              string     `json:"concern_type"`
	StartedAt                time.Time  `json:"started_at"`
	StartedByEventRef        *uuid.UUID `json:"started_by_event_ref,omitempty"`
	ExpectedResolutionAt     time.Time  `json:"expected_resolution_at"`
	OwnerRoleRef             *uuid.UUID `json:"owner_role_ref,omitempty"`
	RelatedMonitoringPlanRef *uuid.UUID `json:"related_monitoring_plan_ref,omitempty"`
	Notes                    string     `json:"notes,omitempty"`
}

// PatchActiveConcernResolutionRequest is the payload for PATCH
// /v2/active-concerns/:id.
type PatchActiveConcernResolutionRequest struct {
	ResolutionStatus           string     `json:"resolution_status"`
	ResolvedAt                 time.Time  `json:"resolved_at"`
	ResolutionEvidenceTraceRef *uuid.UUID `json:"resolution_evidence_trace_ref,omitempty"`
}

// CreateActiveConcern opens a new ActiveConcern for resident `id`.
func (c *KB20Client) CreateActiveConcern(ctx context.Context, residentID uuid.UUID, req CreateActiveConcernRequest) (*models.ActiveConcern, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/active-concerns"
	return doJSON[models.ActiveConcern](ctx, c.http, http.MethodPost, u, req)
}

// ListActiveConcernsByResident lists concerns for `residentID`. status may
// be empty (any status) or one of models.ResolutionStatus*.
func (c *KB20Client) ListActiveConcernsByResident(ctx context.Context, residentID uuid.UUID, status string) ([]models.ActiveConcern, error) {
	q := url.Values{}
	if status != "" {
		q.Set("status", status)
	}
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/active-concerns"
	if encoded := q.Encode(); encoded != "" {
		u += "?" + encoded
	}
	out, err := doJSON[[]models.ActiveConcern](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// PatchActiveConcernResolution transitions an ActiveConcern from open to
// a terminal status.
func (c *KB20Client) PatchActiveConcernResolution(ctx context.Context, id uuid.UUID, req PatchActiveConcernResolutionRequest) (*models.ActiveConcern, error) {
	u := c.baseURL + "/v2/active-concerns/" + id.String()
	return doJSON[models.ActiveConcern](ctx, c.http, http.MethodPatch, u, req)
}

// ListExpiringActiveConcerns returns open concerns whose
// expected_resolution_at < now() + within. Used by cron-driven sweeps.
func (c *KB20Client) ListExpiringActiveConcerns(ctx context.Context, within time.Duration) ([]models.ActiveConcern, error) {
	q := url.Values{}
	q.Set("within", fmt.Sprintf("%dh", int(within.Hours())))
	u := c.baseURL + "/v2/active-concerns/expiring?" + q.Encode()
	out, err := doJSON[[]models.ActiveConcern](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}
