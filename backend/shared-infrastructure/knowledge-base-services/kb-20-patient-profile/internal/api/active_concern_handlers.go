package api

// ActiveConcern REST handlers (Wave 2.3 of Layer 2 substrate plan).
//
// Mounted via:
//
//	h := NewActiveConcernHandlers(activeConcernStore)
//	h.RegisterRoutes(router.Group("/v2"))
//
// Endpoints (mounted under /v2 by the caller):
//
//	POST   /residents/:id/active-concerns        — open a new concern
//	GET    /residents/:id/active-concerns        — list (optional ?status=)
//	PATCH  /active-concerns/:id                  — update resolution
//	GET    /active-concerns/expiring?within=Xh   — cron-friendly listing

import (
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"

	"kb-patient-profile/internal/storage"
)

// ActiveConcernHandlers serves the REST endpoints backed by an
// ActiveConcernStore.
type ActiveConcernHandlers struct {
	store *storage.ActiveConcernStore
}

// NewActiveConcernHandlers constructs a handler set bound to store.
func NewActiveConcernHandlers(store *storage.ActiveConcernStore) *ActiveConcernHandlers {
	return &ActiveConcernHandlers{store: store}
}

// RegisterRoutes wires the active-concern endpoints onto g (typically the
// /v2 router group).
func (h *ActiveConcernHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/residents/:resident_id/active-concerns", h.createForResident)
	g.GET("/residents/:resident_id/active-concerns", h.listForResident)
	g.PATCH("/active-concerns/:id", h.patchResolution)
	g.GET("/active-concerns/expiring", h.listExpiring)
}

// createForResidentBody is the request shape for POST. ResidentID is taken
// from the path and overrides any body value.
type createForResidentBody struct {
	ConcernType                string     `json:"concern_type"`
	StartedAt                  time.Time  `json:"started_at"`
	StartedByEventRef          *uuid.UUID `json:"started_by_event_ref,omitempty"`
	ExpectedResolutionAt       time.Time  `json:"expected_resolution_at"`
	OwnerRoleRef               *uuid.UUID `json:"owner_role_ref,omitempty"`
	RelatedMonitoringPlanRef   *uuid.UUID `json:"related_monitoring_plan_ref,omitempty"`
	Notes                      string     `json:"notes,omitempty"`
}

func (h *ActiveConcernHandlers) createForResident(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	var body createForResidentBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in := models.ActiveConcern{
		ResidentID:               rid,
		ConcernType:              body.ConcernType,
		StartedAt:                body.StartedAt,
		StartedByEventRef:        body.StartedByEventRef,
		ExpectedResolutionAt:     body.ExpectedResolutionAt,
		OwnerRoleRef:             body.OwnerRoleRef,
		RelatedMonitoringPlanRef: body.RelatedMonitoringPlanRef,
		ResolutionStatus:         models.ResolutionStatusOpen,
		Notes:                    body.Notes,
	}
	// Validate before persistence so 400 errors don't reach the DB.
	if err := validation.ValidateActiveConcern(in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.CreateActiveConcern(c.Request.Context(), in)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ActiveConcernHandlers) listForResident(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	status := c.Query("status")
	if status != "" && !models.IsValidResolutionStatus(status) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid status"})
		return
	}
	out, err := h.store.ListActiveConcernsByResident(c.Request.Context(), rid, status)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.ActiveConcern{}
	}
	c.JSON(http.StatusOK, out)
}

// patchResolutionBody is the request shape for PATCH /active-concerns/:id.
type patchResolutionBody struct {
	ResolutionStatus            string     `json:"resolution_status"`
	ResolvedAt                  time.Time  `json:"resolved_at"`
	ResolutionEvidenceTraceRef  *uuid.UUID `json:"resolution_evidence_trace_ref,omitempty"`
}

func (h *ActiveConcernHandlers) patchResolution(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	var body patchResolutionBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.ResolvedAt.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resolved_at is required"})
		return
	}
	if !models.IsValidResolutionStatus(body.ResolutionStatus) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resolution_status"})
		return
	}
	out, err := h.store.UpdateResolution(c.Request.Context(), id,
		body.ResolutionStatus, body.ResolvedAt, body.ResolutionEvidenceTraceRef)
	if err != nil {
		// Distinguish illegal-transition (400) from not-found (404) and
		// generic errors (500). The store wraps "transition" errors with
		// a sentinel string — we recognise it via the wrapped validation
		// error.
		if errors.Is(err, interfaces.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		// Best-effort 400 vs 500 split: any error that isn't ErrNotFound
		// is treated as a client-side issue (most likely an illegal
		// transition or stale resolved_at). Production wiring may refine
		// this once we add a typed sentinel for transition rejections.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ActiveConcernHandlers) listExpiring(c *gin.Context) {
	withinStr := c.DefaultQuery("within", "0h")
	within, err := time.ParseDuration(withinStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid within (expect duration like 24h)"})
		return
	}
	if within < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "within must be >= 0"})
		return
	}
	out, err := h.store.ListExpiringConcerns(c.Request.Context(), within)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.ActiveConcern{}
	}
	c.JSON(http.StatusOK, out)
}
