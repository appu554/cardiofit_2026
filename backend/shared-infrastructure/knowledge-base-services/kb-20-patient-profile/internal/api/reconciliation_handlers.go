package api

// Reconciliation REST handlers (Wave 4 of Layer 2 substrate plan).
//
// Mounted via:
//
//	h := NewReconciliationHandlers(docStore, reconStore)
//	h.RegisterRoutes(router.Group("/v2"))
//
// Endpoints (mounted under /v2 by the caller):
//
//	POST   /discharge-documents                      — ingest a parsed discharge doc
//	GET    /discharge-documents/:id                  — fetch one + its lines
//	GET    /residents/:resident_id/discharge-documents — list for a resident
//	POST   /reconciliation/start                     — start a worklist for a doc
//	GET    /reconciliation/:worklist_id              — fetch a worklist + decisions
//	GET    /reconciliation                           — list worklists by role/facility/status
//	POST   /reconciliation/:worklist_id/lines/:decision_id/decide — record ACOP decision
//	POST   /reconciliation/:worklist_id/finalise     — close the worklist + write-back

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"

	"kb-patient-profile/internal/storage"
)

// ReconciliationHandlers serves the reconciliation REST endpoints.
type ReconciliationHandlers struct {
	docs *storage.DischargeDocumentStore
	recs *storage.ReconciliationStore
}

// NewReconciliationHandlers constructs the handler set.
func NewReconciliationHandlers(docs *storage.DischargeDocumentStore, recs *storage.ReconciliationStore) *ReconciliationHandlers {
	return &ReconciliationHandlers{docs: docs, recs: recs}
}

// RegisterRoutes wires the reconciliation endpoints onto g.
func (h *ReconciliationHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/discharge-documents", h.createDischargeDocument)
	g.GET("/discharge-documents/:id", h.getDischargeDocument)
	g.GET("/residents/:resident_id/discharge-documents", h.listDischargeDocumentsByResident)
	g.POST("/reconciliation/start", h.startReconciliation)
	g.GET("/reconciliation", h.listReconciliationWorklists)
	g.GET("/reconciliation/:worklist_id", h.getReconciliationWorklist)
	g.POST("/reconciliation/:worklist_id/lines/:decision_id/decide", h.decideReconciliation)
	g.POST("/reconciliation/:worklist_id/finalise", h.finaliseReconciliation)
}

// ----- discharge documents -----

func (h *ReconciliationHandlers) createDischargeDocument(c *gin.Context) {
	var body interfaces.DischargeDocument
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.docs.CreateDischargeDocument(c.Request.Context(), body)
	if err != nil {
		if errors.Is(err, storage.ErrDuplicateDocument) {
			c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
			return
		}
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ReconciliationHandlers) getDischargeDocument(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	out, err := h.docs.GetDischargeDocument(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ReconciliationHandlers) listDischargeDocumentsByResident(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	limit, offset := parsePaging(c)
	out, err := h.docs.ListDischargeDocumentsByResident(c.Request.Context(), rid, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// ----- reconciliation worklists -----

type startReconciliationBody struct {
	DischargeDocumentRef uuid.UUID  `json:"discharge_document_ref"`
	AssignedRoleRef      *uuid.UUID `json:"assigned_role_ref,omitempty"`
	FacilityID           *uuid.UUID `json:"facility_id,omitempty"`
	DueWindowHours       int        `json:"due_window_hours,omitempty"`
}

func (h *ReconciliationHandlers) startReconciliation(c *gin.Context) {
	var body startReconciliationBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if body.DischargeDocumentRef == uuid.Nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "discharge_document_ref required"})
		return
	}
	out, err := h.recs.StartWorklist(c.Request.Context(), interfaces.ReconciliationStartInputs{
		DischargeDocumentRef: body.DischargeDocumentRef,
		AssignedRoleRef:      body.AssignedRoleRef,
		FacilityID:           body.FacilityID,
		DueWindowHours:       body.DueWindowHours,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ReconciliationHandlers) getReconciliationWorklist(c *gin.Context) {
	wid, err := uuid.Parse(c.Param("worklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid worklist_id"})
		return
	}
	wl, decs, err := h.recs.GetWorklist(c.Request.Context(), wid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"worklist": wl, "decisions": decs})
}

func (h *ReconciliationHandlers) listReconciliationWorklists(c *gin.Context) {
	var roleRef, facilityID *uuid.UUID
	if v := c.Query("role_ref"); v != "" {
		u, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid role_ref"})
			return
		}
		roleRef = &u
	}
	if v := c.Query("facility_id"); v != "" {
		u, err := uuid.Parse(v)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid facility_id"})
			return
		}
		facilityID = &u
	}
	status := c.Query("status")
	limit, offset := parsePaging(c)
	out, err := h.recs.ListWorklistsByRoleAndFacility(c.Request.Context(), roleRef, facilityID, status, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

type decideReconciliationBody struct {
	ACOPDecision        string    `json:"acop_decision"`
	ACOPRoleRef         uuid.UUID `json:"acop_role_ref"`
	IntentClassOverride string    `json:"intent_class_override,omitempty"`
	Notes               string    `json:"notes,omitempty"`
	OverrideDose        string    `json:"override_dose,omitempty"`
	OverrideFrequency   string    `json:"override_frequency,omitempty"`
	OverrideRoute       string    `json:"override_route,omitempty"`
}

func (h *ReconciliationHandlers) decideReconciliation(c *gin.Context) {
	wid, err := uuid.Parse(c.Param("worklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid worklist_id"})
		return
	}
	did, err := uuid.Parse(c.Param("decision_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid decision_id"})
		return
	}
	var body decideReconciliationBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.recs.DecideReconciliation(c.Request.Context(), interfaces.DecideReconciliationInputs{
		WorklistRef:         wid,
		DecisionRef:         did,
		ACOPDecision:        body.ACOPDecision,
		ACOPRoleRef:         body.ACOPRoleRef,
		IntentClassOverride: body.IntentClassOverride,
		Notes:               body.Notes,
		OverrideDose:        body.OverrideDose,
		OverrideFrequency:   body.OverrideFrequency,
		OverrideRoute:       body.OverrideRoute,
	})
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

type finaliseReconciliationBody struct {
	CompletedByRoleRef uuid.UUID `json:"completed_by_role_ref"`
}

func (h *ReconciliationHandlers) finaliseReconciliation(c *gin.Context) {
	wid, err := uuid.Parse(c.Param("worklist_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid worklist_id"})
		return
	}
	var body finaliseReconciliationBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.recs.FinaliseWorklist(c.Request.Context(), wid, body.CompletedByRoleRef)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// parsePaging extracts limit / offset query params. Defaults: limit=50,
// offset=0; invalid values fall back to defaults silently.
func parsePaging(c *gin.Context) (int, int) {
	limit := 50
	offset := 0
	if v, err := strconv.Atoi(c.DefaultQuery("limit", "50")); err == nil && v > 0 {
		limit = v
	}
	if v, err := strconv.Atoi(c.DefaultQuery("offset", "0")); err == nil && v >= 0 {
		offset = v
	}
	return limit, offset
}
