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

// CreateCapacityAssessmentRequest is the payload for POST
// /v2/residents/:id/capacity. ResidentRef is taken from the path; the
// body's ResidentRef field, if any, is ignored.
type CreateCapacityAssessmentRequest struct {
	AssessedAt          time.Time       `json:"assessed_at"`
	AssessorRoleRef     uuid.UUID       `json:"assessor_role_ref"`
	Domain              string          `json:"domain"`
	Instrument          string          `json:"instrument,omitempty"`
	Score               *float64        `json:"score,omitempty"`
	Outcome             string          `json:"outcome"`
	Duration            string          `json:"duration"`
	ExpectedReviewDate  *time.Time      `json:"expected_review_date,omitempty"`
	RationaleStructured json.RawMessage `json:"rationale_structured,omitempty"`
	RationaleFreeText   string          `json:"rationale_free_text,omitempty"`
	SupersedesRef       *uuid.UUID      `json:"supersedes_ref,omitempty"`
}

// CreateCapacityAssessment records a new capacity assessment for
// residentID. Returns the persisted row, the optional capacity_change
// Event (set only when impaired+medical_decisions), and the
// EvidenceTrace node id that was written.
func (c *KB20Client) CreateCapacityAssessment(ctx context.Context, residentID uuid.UUID, req CreateCapacityAssessmentRequest) (*interfaces.CapacityAssessmentResult, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/capacity"
	return doJSON[interfaces.CapacityAssessmentResult](ctx, c.http, http.MethodPost, u, req)
}

// GetCurrentCapacityForDomain returns the latest CapacityAssessment for
// (residentID, domain). The HTTP layer returns 404 when no assessment
// exists for that pair.
func (c *KB20Client) GetCurrentCapacityForDomain(ctx context.Context, residentID uuid.UUID, domain string) (*models.CapacityAssessment, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/capacity/current/" + domain
	return doJSON[models.CapacityAssessment](ctx, c.http, http.MethodGet, u, nil)
}

// ListCurrentCapacityByResident returns one row per domain present for
// residentID (latest by assessed_at within each domain). Empty slice
// when the resident has no assessments yet.
func (c *KB20Client) ListCurrentCapacityByResident(ctx context.Context, residentID uuid.UUID) ([]models.CapacityAssessment, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/capacity/current"
	out, err := doJSON[[]models.CapacityAssessment](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}

// ListCapacityHistoryForDomain returns the full history for (residentID,
// domain), newest-first by assessed_at.
func (c *KB20Client) ListCapacityHistoryForDomain(ctx context.Context, residentID uuid.UUID, domain string) ([]models.CapacityAssessment, error) {
	u := c.baseURL + "/v2/residents/" + residentID.String() + "/capacity/history/" + domain
	out, err := doJSON[[]models.CapacityAssessment](ctx, c.http, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	return *out, nil
}
