package api

// CareIntensity REST handlers (Wave 2.4 of Layer 2 substrate plan).
//
// Mounted via:
//
//	h := NewCareIntensityHandlers(careIntensityStore)
//	h.RegisterRoutes(router.Group("/v2"))
//
// Endpoints (mounted under /v2 by the caller):
//
//	POST /residents/:id/care-intensity          — record a transition; returns
//	                                              {care_intensity, event, cascades}
//	GET  /residents/:id/care-intensity/current  — latest tag (404 if none)
//	GET  /residents/:id/care-intensity/history  — full history (newest first)

import (
	"encoding/json"
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

// CareIntensityHandlers serves the REST endpoints backed by a
// CareIntensityStore.
type CareIntensityHandlers struct {
	store *storage.CareIntensityStore
}

// NewCareIntensityHandlers constructs a handler set bound to store.
func NewCareIntensityHandlers(store *storage.CareIntensityStore) *CareIntensityHandlers {
	return &CareIntensityHandlers{store: store}
}

// RegisterRoutes wires the care-intensity endpoints onto g (typically the
// /v2 router group).
func (h *CareIntensityHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/residents/:resident_id/care-intensity", h.createTransition)
	g.GET("/residents/:resident_id/care-intensity/current", h.getCurrent)
	g.GET("/residents/:resident_id/care-intensity/history", h.getHistory)
}

// createTransitionBody is the request shape for POST. ResidentRef is
// taken from the path and overrides any body value.
type createTransitionBody struct {
	Tag                 string          `json:"tag"`
	EffectiveDate       time.Time       `json:"effective_date"`
	DocumentedByRoleRef uuid.UUID       `json:"documented_by_role_ref"`
	ReviewDueDate       *time.Time      `json:"review_due_date,omitempty"`
	RationaleStructured json.RawMessage `json:"rationale_structured,omitempty"`
	RationaleFreeText   string          `json:"rationale_free_text,omitempty"`
}

func (h *CareIntensityHandlers) createTransition(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	var body createTransitionBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in := models.CareIntensity{
		ResidentRef:         rid,
		Tag:                 body.Tag,
		EffectiveDate:       body.EffectiveDate,
		DocumentedByRoleRef: body.DocumentedByRoleRef,
		ReviewDueDate:       body.ReviewDueDate,
		RationaleStructured: body.RationaleStructured,
		RationaleFreeText:   body.RationaleFreeText,
	}
	// Surface validation 400s before reaching the DB.
	if in.EffectiveDate.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "effective_date is required"})
		return
	}
	if err := validation.ValidateCareIntensity(in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.CreateCareIntensityTransition(c.Request.Context(), in)
	if err != nil {
		// Validation / illegal-transition errors → 400; everything else → 500.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *CareIntensityHandlers) getCurrent(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.GetCurrentCareIntensity(c.Request.Context(), rid)
	if err != nil {
		if errors.Is(err, interfaces.ErrNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *CareIntensityHandlers) getHistory(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.ListCareIntensityHistory(c.Request.Context(), rid)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.CareIntensity{}
	}
	c.JSON(http.StatusOK, out)
}
