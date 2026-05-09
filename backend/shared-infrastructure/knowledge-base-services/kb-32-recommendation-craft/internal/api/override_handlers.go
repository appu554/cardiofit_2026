package api

// override_handlers.go — POST /v1/craft/override/{recommendation_id}
//
// Captures a clinician override of a recommendation, persisting a structured
// OverrideReason record via the injected Store.
//
// PDP middleware: NOT mounted on this route. The permissions PDP class
// enforcement is deferred to Phase 2-completion. The route is wired without
// middleware wrapping; the TODO below tracks the follow-up.
//
//   TODO(phase-2-completion): wrap /v1/craft/override/:id with the shared
//   permissions.Middleware(AD class) once the PDP enforcement task lands.
//   See docs/superpowers/plans/2026-05-09-phase-2b-clinical-safety-and-audit-moat.md
//   §"PDP middleware deferral".

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/kb32/internal/overrides"
	"github.com/cardiofit/shared/v2_substrate/permissions"
)

// ---------------------------------------------------------------------------
// Request / Response shapes
// ---------------------------------------------------------------------------

// OverrideCaptureRequest is the JSON body for POST /v1/craft/override/{id}.
//
// recommendation_id is captured from the URL path parameter, not the body.
// captured_by is populated from the JWT viewer role (permissions.ViewerRoleFrom).
type OverrideCaptureRequest struct {
	// ReasonCode is one of the 20 canonical override reason codes (Guidelines §5).
	ReasonCode string `json:"reason_code" binding:"required"`

	// AppropriatenessFlag classifies the override as "appropriate_override",
	// "inappropriate_override", or "mixed".
	AppropriatenessFlag string `json:"appropriateness_flag" binding:"required"`

	// Reasoning is mandatory free-text capturing why the override was made.
	Reasoning string `json:"reasoning" binding:"required"`
}

// OverrideCaptureResponse is the JSON response body returned on 201 Created.
type OverrideCaptureResponse struct {
	// ID is the UUID of the persisted override record.
	ID string `json:"id"`

	// RecommendationID echoes the path parameter for caller convenience.
	RecommendationID string `json:"recommendation_id"`

	// ReasonCode echoes the persisted reason code.
	ReasonCode string `json:"reason_code"`

	// AppropriatenessFlag echoes the persisted flag.
	AppropriatenessFlag string `json:"appropriateness_flag"`
}

// ErrorEnvelope wraps a single error message for non-2xx responses.
type ErrorEnvelope struct {
	Error string `json:"error"`
}

// ---------------------------------------------------------------------------
// OverrideHandler
// ---------------------------------------------------------------------------

// OverrideHandler holds the Store dependency for the override capture endpoint.
type OverrideHandler struct {
	store overrides.Store
}

// NewOverrideHandler constructs an OverrideHandler backed by the given Store.
func NewOverrideHandler(store overrides.Store) *OverrideHandler {
	return &OverrideHandler{store: store}
}

// HandleCapture is the Gin handler for POST /v1/craft/override/:recommendation_id.
//
// HTTP status codes:
//   - 201 Created:            override persisted successfully (body: OverrideCaptureResponse)
//   - 400 Bad Request:        malformed JSON or missing required field
//   - 422 Unprocessable Entity: validation failure (bad reason_code, bad flag, empty reasoning,
//     or malformed recommendation_id UUID); body: ErrorEnvelope
//   - 500 Internal Server Error: store failure
//
// captured_by is read from the JWT context via permissions.ViewerRoleFrom. When
// the PDP middleware is not mounted (current Phase 2b state), the context will
// not carry a viewer role; in that case captured_by defaults to "anonymous" so
// the override is still captured for audit purposes. The TODO above tracks
// mounting the middleware.
func (h *OverrideHandler) HandleCapture(c *gin.Context) {
	// Validate recommendation_id from URL path.
	rawID := c.Param("recommendation_id")
	if _, err := uuid.Parse(rawID); err != nil {
		c.JSON(http.StatusUnprocessableEntity, ErrorEnvelope{
			Error: "recommendation_id: " + err.Error(),
		})
		return
	}

	var req OverrideCaptureRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorEnvelope{Error: err.Error()})
		return
	}

	// Build the OverrideReason and run Validate() from Task 1's taxonomy.
	reason := overrides.OverrideReason{
		RecommendationID:    rawID,
		ReasonCode:          req.ReasonCode,
		AppropriatenessFlag: req.AppropriatenessFlag,
		Reasoning:           req.Reasoning,
	}

	// captured_by from JWT context; fall back to "anonymous" when PDP
	// middleware is not yet mounted (Phase 2-completion follow-up).
	if viewerID, ok := permissions.ViewerRoleFrom(c.Request.Context()); ok {
		reason.CapturedBy = viewerID.String()
	} else {
		reason.CapturedBy = "anonymous"
	}

	if err := reason.Validate(); err != nil {
		var msg string
		switch {
		case errors.Is(err, overrides.ErrInvalidReasonCode):
			msg = "reason_code: " + err.Error()
		case errors.Is(err, overrides.ErrInvalidFlag):
			msg = "appropriateness_flag: " + err.Error()
		case errors.Is(err, overrides.ErrEmptyReasoning):
			msg = "reasoning: " + err.Error()
		default:
			msg = err.Error()
		}
		c.JSON(http.StatusUnprocessableEntity, ErrorEnvelope{Error: msg})
		return
	}

	persisted, err := h.store.Create(c.Request.Context(), reason)
	if err != nil {
		c.JSON(http.StatusInternalServerError, ErrorEnvelope{Error: "store: " + err.Error()})
		return
	}

	c.JSON(http.StatusCreated, OverrideCaptureResponse{
		ID:                  persisted.ID,
		RecommendationID:    persisted.RecommendationID,
		ReasonCode:          persisted.ReasonCode,
		AppropriatenessFlag: persisted.AppropriatenessFlag,
	})
}
