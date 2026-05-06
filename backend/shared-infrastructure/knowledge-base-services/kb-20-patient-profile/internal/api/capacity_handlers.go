package api

// Capacity REST handlers (Wave 2.5 of Layer 2 substrate plan; Layer 2
// doc §2.5).
//
// Mounted via:
//
//	h := NewCapacityHandlers(capacityStore)
//	h.RegisterRoutes(router.Group("/v2"))
//
// Endpoints (mounted under /v2 by the caller):
//
//	POST /residents/:id/capacity
//	    body: full CapacityAssessment payload (resident_ref taken from path)
//	    → 200 {assessment, event?, evidence_trace_node_ref}; 400 on
//	      validator failure
//
//	GET  /residents/:id/capacity/current
//	    → 200 [CapacityAssessment, ...] — one row per domain present
//
//	GET  /residents/:id/capacity/current/:domain
//	    → 200 {CapacityAssessment}; 404 if no assessment for that domain
//
//	GET  /residents/:id/capacity/history/:domain
//	    → 200 [CapacityAssessment, ...] — descending by assessed_at

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

// CapacityHandlers serves the REST endpoints backed by a
// CapacityAssessmentStore.
type CapacityHandlers struct {
	store *storage.CapacityAssessmentStore
}

// NewCapacityHandlers constructs a handler set bound to store.
func NewCapacityHandlers(store *storage.CapacityAssessmentStore) *CapacityHandlers {
	return &CapacityHandlers{store: store}
}

// RegisterRoutes wires the capacity endpoints onto g (typically the /v2
// router group).
func (h *CapacityHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/residents/:resident_id/capacity", h.createAssessment)
	g.GET("/residents/:resident_id/capacity/current", h.listCurrent)
	g.GET("/residents/:resident_id/capacity/current/:domain", h.getCurrentByDomain)
	g.GET("/residents/:resident_id/capacity/history/:domain", h.getHistoryByDomain)
}

// createCapacityBody is the request shape for POST. ResidentRef is taken
// from the path and overrides any body value.
type createCapacityBody struct {
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

func (h *CapacityHandlers) createAssessment(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	var body createCapacityBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in := models.CapacityAssessment{
		ResidentRef:         rid,
		AssessedAt:          body.AssessedAt,
		AssessorRoleRef:     body.AssessorRoleRef,
		Domain:              body.Domain,
		Instrument:          body.Instrument,
		Score:               body.Score,
		Outcome:             body.Outcome,
		Duration:            body.Duration,
		ExpectedReviewDate:  body.ExpectedReviewDate,
		RationaleStructured: body.RationaleStructured,
		RationaleFreeText:   body.RationaleFreeText,
		SupersedesRef:       body.SupersedesRef,
	}
	if in.AssessedAt.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assessed_at is required"})
		return
	}
	// Surface validation 400s before reaching the DB.
	if err := validation.ValidateCapacityAssessment(in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.CreateCapacityAssessment(c.Request.Context(), in)
	if err != nil {
		// Validation errors (CHECK constraint, unique, etc.) → 400; others → 500.
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *CapacityHandlers) listCurrent(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.ListCurrentCapacityByResident(c.Request.Context(), rid)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.CapacityAssessment{}
	}
	c.JSON(http.StatusOK, out)
}

func (h *CapacityHandlers) getCurrentByDomain(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	domain := c.Param("domain")
	if !models.IsValidCapacityDomain(domain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain"})
		return
	}
	out, err := h.store.GetCurrentCapacity(c.Request.Context(), rid, domain)
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

func (h *CapacityHandlers) getHistoryByDomain(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	domain := c.Param("domain")
	if !models.IsValidCapacityDomain(domain) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid domain"})
		return
	}
	out, err := h.store.ListCapacityHistory(c.Request.Context(), rid, domain)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.CapacityAssessment{}
	}
	c.JSON(http.StatusOK, out)
}
