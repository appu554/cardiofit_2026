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
	"context"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"
	"github.com/cardiofit/shared/v2_substrate/interfaces"
	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"

	"kb-patient-profile/internal/storage"
)

// V2SubstrateHandlers serves v2 substrate REST endpoints for kb-20.
//
// onMedicineUseChanged is an optional best-effort callback fired after a
// successful UpsertMedicineUse. Wave 2.6 wires the kb-20 ScoringStore's
// RecomputeDrugBurden through here so DBI/ACB are recomputed on every
// MedicineUse insert/update/end. The callback runs synchronously in the
// request goroutine but its return value is intentionally ignored —
// recompute MUST NOT fail the underlying MedicineUse write. TODO: move
// to outbox-driven async in production once the outbox supports
// per-resident coalescing.
type V2SubstrateHandlers struct {
	store                *storage.V2SubstrateStore
	onMedicineUseChanged func(ctx context.Context, residentRef uuid.UUID)
}

// NewV2SubstrateHandlers constructs a handler set bound to the given store.
func NewV2SubstrateHandlers(store *storage.V2SubstrateStore) *V2SubstrateHandlers {
	return &V2SubstrateHandlers{store: store}
}

// SetOnMedicineUseChanged registers a best-effort post-write callback
// fired on successful UpsertMedicineUse. Wave 2.6 wires
// ScoringStore.RecomputeDrugBurden through here.
func (h *V2SubstrateHandlers) SetOnMedicineUseChanged(cb func(ctx context.Context, residentRef uuid.UUID)) {
	h.onMedicineUseChanged = cb
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

	g.POST("/events", h.upsertEvent)
	g.GET("/events/:id", h.getEvent)
	g.GET("/residents/:resident_id/events", h.listEventsByResident)
	g.GET("/events", h.listEventsByType)

	g.POST("/evidence-trace/nodes", h.upsertEvidenceTraceNode)
	g.GET("/evidence-trace/nodes/:id", h.getEvidenceTraceNode)
	g.POST("/evidence-trace/edges", h.insertEvidenceTraceEdge)
	g.GET("/evidence-trace/:id/forward", h.traceEvidenceForward)
	g.GET("/evidence-trace/:id/backward", h.traceEvidenceBackward)

	// Wave 5.2 — Layer 3 / regulator-audit query API.
	g.GET("/evidence-trace/recommendations/:id/lineage", h.getRecommendationLineage)
	g.GET("/evidence-trace/observations/:id/consequences", h.getObservationConsequences)
	g.GET("/residents/:resident_id/reasoning-window", h.getReasoningWindow)
}

// maxEvidenceTraceDepth is the server-side cap on traversal depth, to
// prevent abuse via a runaway depth query parameter.
const maxEvidenceTraceDepth = 50

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
	// Wave 2.6: best-effort DBI/ACB recompute. Errors are swallowed so
	// the underlying MedicineUse write always commits.
	if h.onMedicineUseChanged != nil && out != nil {
		h.onMedicineUseChanged(c.Request.Context(), out.ResidentID)
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

// ---------------------------------------------------------------------------
// Event
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertEvent(c *gin.Context) {
	var e models.Event
	if err := c.BindJSON(&e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateEvent(e); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.UpsertEvent(c.Request.Context(), e)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getEvent(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	e, err := h.store.GetEvent(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, e)
}

func (h *V2SubstrateHandlers) listEventsByResident(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit <= 0 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be in (0, 1000]"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}
	out, err := h.store.ListEventsByResident(c.Request.Context(), residentID, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// listEventsByType serves GET /v2/events?type=&from=&to=&limit=&offset=
//   - type    is required and must be a valid EventType
//   - from/to are RFC3339 datetimes; either may be omitted (no bound)
//   - to must be > from when both are set (returns 400 otherwise)
func (h *V2SubstrateHandlers) listEventsByType(c *gin.Context) {
	eventType := c.Query("type")
	if eventType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "type query parameter required"})
		return
	}
	if !models.IsValidEventType(eventType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid event type"})
		return
	}
	var from, to time.Time
	if s := c.Query("from"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from (expect RFC3339)"})
			return
		}
		from = t
	}
	if s := c.Query("to"); s != "" {
		t, err := time.Parse(time.RFC3339, s)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to (expect RFC3339)"})
			return
		}
		to = t
	}
	if !from.IsZero() && !to.IsZero() && !to.After(from) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "to must be after from"})
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "100"))
	if err != nil || limit <= 0 || limit > 1000 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "limit must be in (0, 1000]"})
		return
	}
	offset, err := strconv.Atoi(c.DefaultQuery("offset", "0"))
	if err != nil || offset < 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid offset"})
		return
	}
	out, err := h.store.ListEventsByType(c.Request.Context(), eventType, from, to, limit, offset)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// ---------------------------------------------------------------------------
// EvidenceTrace
// ---------------------------------------------------------------------------

func (h *V2SubstrateHandlers) upsertEvidenceTraceNode(c *gin.Context) {
	var n models.EvidenceTraceNode
	if err := c.BindJSON(&n); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := validation.ValidateEvidenceTraceNode(n); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.UpsertEvidenceTraceNode(c.Request.Context(), n)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) getEvidenceTraceNode(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	n, err := h.store.GetEvidenceTraceNode(c.Request.Context(), id)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, n)
}

// edgeRequest is the wire-format payload for POST /v2/evidence-trace/edges.
// Mirrors evidence_trace.Edge with explicit JSON tags so callers can use
// either the raw struct or this shape.
type edgeRequest struct {
	From string `json:"from_node"`
	To   string `json:"to_node"`
	Kind string `json:"edge_kind"`
}

func (h *V2SubstrateHandlers) insertEvidenceTraceEdge(c *gin.Context) {
	var req edgeRequest
	if err := c.BindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	from, err := uuid.Parse(req.From)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from_node"})
		return
	}
	to, err := uuid.Parse(req.To)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to_node"})
		return
	}
	if !evidence_trace.IsValidEdgeKind(req.Kind) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid edge_kind"})
		return
	}
	edge := evidence_trace.Edge{From: from, To: to, Kind: evidence_trace.EdgeKind(req.Kind)}
	if err := h.store.InsertEvidenceTraceEdge(c.Request.Context(), edge); err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// parseTraversalDepth reads the ?depth= query parameter, defaults to 10,
// caps at maxEvidenceTraceDepth. Returns (depth, ok). On failure writes
// the 400 response and returns ok=false.
func parseTraversalDepth(c *gin.Context) (int, bool) {
	depth, err := strconv.Atoi(c.DefaultQuery("depth", "10"))
	if err != nil || depth <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "depth must be a positive integer"})
		return 0, false
	}
	if depth > maxEvidenceTraceDepth {
		depth = maxEvidenceTraceDepth
	}
	return depth, true
}

func (h *V2SubstrateHandlers) traceEvidenceForward(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	depth, ok := parseTraversalDepth(c)
	if !ok {
		return
	}
	out, err := h.store.TraceForward(c.Request.Context(), id, depth)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.EvidenceTraceNode{}
	}
	c.JSON(http.StatusOK, out)
}

func (h *V2SubstrateHandlers) traceEvidenceBackward(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	depth, ok := parseTraversalDepth(c)
	if !ok {
		return
	}
	out, err := h.store.TraceBackward(c.Request.Context(), id, depth)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.EvidenceTraceNode{}
	}
	c.JSON(http.StatusOK, out)
}
