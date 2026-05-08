// Package api — runtime_handlers.go
//
// REST endpoints under /v2/runtime/* that the kb-cql-runtime Java service
// (Plan 0.5) calls during CQL evaluation. Each endpoint is a thin read-only
// composition over existing kb-20 stores.
//
// The Java side's SubstrateExternalFunctions class (Plan 0.5 Task 3) maps
// each Vaidshala.Substrate.X CQL function to one of these endpoints.
//
// Endpoints:
//
//	GET /v2/runtime/baseline?resident_id=X&type=potassium
//	    → {baseline_value, baseline_confidence, baseline_n_observations}
//	GET /v2/runtime/active-concerns?resident_id=X
//	    → ["post_fall_72h", "antibiotic_course_active", ...]
//	GET /v2/runtime/care-intensity?resident_id=X
//	    → {tag: "active_treatment" | ... }
//	GET /v2/runtime/medicine-use?resident_id=X
//	    → list of MedicineUse summaries
//	GET /v2/runtime/observations?resident_id=X&type=potassium&limit=10
//	    → list of recent observations of the given type

package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// RuntimeProviders abstracts the five substrate reads the runtime API needs.
// Composed of method signatures that match the underlying kb-20 store APIs
// closely enough to delegate. Tests inject a fake; production passes a
// concrete adapter wired to the actual stores.
type RuntimeProviders interface {
	GetBaseline(residentID uuid.UUID, observationType string) (
		baseline float64, confidence string, n int, err error)
	GetActiveConcerns(residentID uuid.UUID) ([]string, error)
	GetCareIntensity(residentID uuid.UUID) (string, error)
	GetMedicineUse(residentID uuid.UUID) ([]map[string]any, error)
	GetObservations(residentID uuid.UUID, observationType string, limit int) (
		[]map[string]any, error)
}

// RuntimeHandlers serves the substrate REST endpoints consumed by
// kb-cql-runtime.
type RuntimeHandlers struct {
	providers RuntimeProviders
}

// NewRuntimeHandlers constructs a handler set bound to the supplied
// providers.
func NewRuntimeHandlers(p RuntimeProviders) *RuntimeHandlers {
	return &RuntimeHandlers{providers: p}
}

// RegisterRoutes mounts the five GET endpoints on g (typically /v2).
func (h *RuntimeHandlers) RegisterRoutes(g *gin.RouterGroup) {
	rt := g.Group("/runtime")
	rt.GET("/baseline", h.getBaseline)
	rt.GET("/active-concerns", h.getActiveConcerns)
	rt.GET("/care-intensity", h.getCareIntensity)
	rt.GET("/medicine-use", h.getMedicineUse)
	rt.GET("/observations", h.getObservations)
}

func (h *RuntimeHandlers) getBaseline(c *gin.Context) {
	residentID, ok := parseRuntimeResidentID(c)
	if !ok {
		return
	}
	obsType := c.Query("type")
	if obsType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'type' query parameter"})
		return
	}
	v, conf, n, err := h.providers.GetBaseline(residentID, obsType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"baseline_value":          v,
		"baseline_confidence":     conf,
		"baseline_n_observations": n,
	})
}

func (h *RuntimeHandlers) getActiveConcerns(c *gin.Context) {
	residentID, ok := parseRuntimeResidentID(c)
	if !ok {
		return
	}
	concerns, err := h.providers.GetActiveConcerns(residentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if concerns == nil {
		concerns = []string{}
	}
	c.JSON(http.StatusOK, concerns)
}

func (h *RuntimeHandlers) getCareIntensity(c *gin.Context) {
	residentID, ok := parseRuntimeResidentID(c)
	if !ok {
		return
	}
	tag, err := h.providers.GetCareIntensity(residentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"tag": tag})
}

func (h *RuntimeHandlers) getMedicineUse(c *gin.Context) {
	residentID, ok := parseRuntimeResidentID(c)
	if !ok {
		return
	}
	meds, err := h.providers.GetMedicineUse(residentID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if meds == nil {
		meds = []map[string]any{}
	}
	c.JSON(http.StatusOK, meds)
}

func (h *RuntimeHandlers) getObservations(c *gin.Context) {
	residentID, ok := parseRuntimeResidentID(c)
	if !ok {
		return
	}
	obsType := c.Query("type")
	if obsType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'type' query parameter"})
		return
	}
	limit := 10
	if l := c.Query("limit"); l != "" {
		if n, err := strconv.Atoi(l); err == nil && n > 0 {
			limit = n
		}
	}
	obs, err := h.providers.GetObservations(residentID, obsType, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	if obs == nil {
		obs = []map[string]any{}
	}
	c.JSON(http.StatusOK, obs)
}

// parseRuntimeResidentID extracts and parses the resident_id query param.
// Writes an error response and returns false if missing or malformed.
func parseRuntimeResidentID(c *gin.Context) (uuid.UUID, bool) {
	raw := c.Query("resident_id")
	if raw == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing 'resident_id' query parameter"})
		return uuid.Nil, false
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id: " + err.Error()})
		return uuid.Nil, false
	}
	return id, true
}

// EmptyRuntimeProviders is a no-op RuntimeProviders that returns empty
// results. Registered by main.go so the /v2/runtime/* endpoints are
// reachable from boot; Plan 0.5 Task 3 swaps it for a real adapter.
type EmptyRuntimeProviders struct{}

func (*EmptyRuntimeProviders) GetBaseline(_ uuid.UUID, _ string) (float64, string, int, error) {
	return 0, "insufficient_data", 0, nil
}
func (*EmptyRuntimeProviders) GetActiveConcerns(_ uuid.UUID) ([]string, error) { return nil, nil }
func (*EmptyRuntimeProviders) GetCareIntensity(_ uuid.UUID) (string, error)    { return "", nil }
func (*EmptyRuntimeProviders) GetMedicineUse(_ uuid.UUID) ([]map[string]any, error) {
	return nil, nil
}
func (*EmptyRuntimeProviders) GetObservations(_ uuid.UUID, _ string, _ int) (
	[]map[string]any, error) {
	return nil, nil
}
