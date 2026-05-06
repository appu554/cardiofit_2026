package api

// V2 Substrate REST handlers (milestone 1B-β.1).
//
// Routes are not auto-registered onto the existing kb-20 Server; they live
// behind a separate, opt-in RouterGroup so the v2 substrate ships
// non-breakingly. Wire them from main.go (or the existing setupRoutes once
// a v2Store is constructed) via:
//
//	v2Handlers := NewV2SubstrateHandlers(v2Store)
//	v2Handlers.RegisterRoutes(router.Group("/v2"))
//
// See commit notes for the deliberate decision not to mutate
// internal/api/server.go / routes.go in this milestone.

import (
	"errors"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"

	"kb-patient-profile/internal/storage"
)

// V2SubstrateHandlers serves v2 substrate REST endpoints for kb-20.
type V2SubstrateHandlers struct {
	store *storage.V2SubstrateStore
}

// NewV2SubstrateHandlers constructs a handler set bound to the given store.
func NewV2SubstrateHandlers(store *storage.V2SubstrateStore) *V2SubstrateHandlers {
	return &V2SubstrateHandlers{store: store}
}

// RegisterRoutes wires the v2 substrate endpoints onto the given router group.
// Caller is expected to mount the group at "/v2".
func (h *V2SubstrateHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/residents", h.upsertResident)
	g.GET("/residents/:id", h.getResident)
	g.GET("/facilities/:facility_id/residents", h.listResidentsByFacility)

	g.POST("/persons", h.upsertPerson)
	g.GET("/persons/:id", h.getPerson)
	g.GET("/persons", h.getPersonByHPII)

	g.POST("/roles", h.upsertRole)
	g.GET("/roles/:id", h.getRole)
	g.GET("/persons/:person_id/roles", h.listRolesByPerson)
	g.GET("/persons/:person_id/active_roles", h.listActiveRolesByPersonAndFacility)

	g.POST("/medicine_uses", h.upsertMedicineUse)
	g.GET("/medicine_uses/:id", h.getMedicineUse)
	g.GET("/residents/:resident_id/medicine_uses", h.listMedicineUsesByResident)

	g.POST("/observations", h.upsertObservation)
	g.GET("/observations/:id", h.getObservation)
	g.GET("/residents/:resident_id/observations", h.listObservationsByResident)
	g.GET("/residents/:resident_id/observations/:kind", h.listObservationsByResidentAndKind)
}

// respondError dispatches not-found errors to 404 and everything else to 500.
func respondError(c *gin.Context, err error) {
	if errors.Is(err, interfaces.ErrNotFound) {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
}

// ---------------------------------------------------------------------------
// Resident
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertResident(c *gin.Context) {
	var r models.Resident
	if err := c.BindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateResident(r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.UpsertResident(c.Request.Context(), r)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getResident(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	r, err := h.store.GetResident(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *V2SubstrateHandlers) listResidentsByFacility(c *gin.Context) {
	facilityID, err := uuid.Parse(c.Param("facility_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid facility_id"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}
	if limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be > 0"})
		return
	}
	if limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be <= 1000"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}
	if offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be >= 0"})
		return
	}
	residents, err := h.store.ListResidentsByFacility(c.Request.Context(), facilityID, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, residents)
}

// ---------------------------------------------------------------------------
// Person
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertPerson(c *gin.Context) {
	var p models.Person
	if err := c.BindJSON(&p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidatePerson(p); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.UpsertPerson(c.Request.Context(), p)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getPerson(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	p, err := h.store.GetPerson(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

func (h *V2SubstrateHandlers) getPersonByHPII(c *gin.Context) {
	hpii := c.Query("hpii")
	if hpii == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "hpii query parameter required"})
		return
	}
	p, err := h.store.GetPersonByHPII(c.Request.Context(), hpii)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, p)
}

// ---------------------------------------------------------------------------
// Role
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertRole(c *gin.Context) {
	var r models.Role
	if err := c.BindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateRole(r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.UpsertRole(c.Request.Context(), r)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getRole(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	r, err := h.store.GetRole(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, r)
}

func (h *V2SubstrateHandlers) listRolesByPerson(c *gin.Context) {
	personID, err := uuid.Parse(c.Param("person_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid person_id"})
		return
	}
	roles, err := h.store.ListRolesByPerson(c.Request.Context(), personID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, roles)
}

func (h *V2SubstrateHandlers) listActiveRolesByPersonAndFacility(c *gin.Context) {
	personID, err := uuid.Parse(c.Param("person_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid person_id"})
		return
	}
	facilityIDStr := c.Query("facility_id")
	if facilityIDStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "facility_id query parameter required"})
		return
	}
	facilityID, err := uuid.Parse(facilityIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid facility_id"})
		return
	}
	roles, err := h.store.ListActiveRolesByPersonAndFacility(c.Request.Context(), personID, facilityID)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, roles)
}

// ---------------------------------------------------------------------------
// MedicineUse
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertMedicineUse(c *gin.Context) {
	var m models.MedicineUse
	if err := c.BindJSON(&m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateMedicineUse(m); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.UpsertMedicineUse(c.Request.Context(), m)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getMedicineUse(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	m, err := h.store.GetMedicineUse(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, m)
}

func (h *V2SubstrateHandlers) listMedicineUsesByResident(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}
	if limit <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be > 0"})
		return
	}
	if limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be <= 1000"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}
	if offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "offset must be >= 0"})
		return
	}
	uses, err := h.store.ListMedicineUsesByResident(c.Request.Context(), residentID, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, uses)
}

// ---------------------------------------------------------------------------
// Observation
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertObservation(c *gin.Context) {
	var o models.Observation
	if err := c.BindJSON(&o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateObservation(o); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.UpsertObservation(c.Request.Context(), o)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getObservation(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	o, err := h.store.GetObservation(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, o)
}

func (h *V2SubstrateHandlers) listObservationsByResident(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}
	if limit <= 0 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be in (0, 1000]"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}
	out, err := h.store.ListObservationsByResident(c.Request.Context(), residentID, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) listObservationsByResidentAndKind(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	kind := c.Param("kind")
	if !models.IsValidObservationKind(kind) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid kind"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid limit"})
		return
	}
	if limit <= 0 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be in (0, 1000]"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}
	out, err := h.store.ListObservationsByResidentAndKind(c.Request.Context(), residentID, kind, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}
