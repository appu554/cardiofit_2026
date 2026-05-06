package api

// Scoring REST handlers (Wave 2.6 of Layer 2 substrate plan; Layer 2
// doc §2.4 / §2.6).
//
// Mounted via:
//
//	h := NewScoringHandlers(scoringStore)
//	h.RegisterRoutes(router.Group("/v2"))
//
// Endpoints (mounted under /v2 by the caller):
//
//	POST /residents/:id/cfs              — record a CFS score (1-9). Returns
//	                                       {cfs_score, care_intensity_hint?, evidence_trace_node_ref}
//	POST /residents/:id/akps             — record an AKPS score (0-100, %10).
//	                                       Same response shape as CFS.
//	GET  /residents/:id/scores/current   — latest CFS/AKPS/DBI/ACB combined
//	GET  /residents/:id/cfs/history      — full CFS history (newest-first)
//	GET  /residents/:id/akps/history     — full AKPS history
//	GET  /residents/:id/dbi/history      — full DBI history (computed)
//	GET  /residents/:id/acb/history      — full ACB history (computed)

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
	"github.com/cardiofit/shared/v2_substrate/validation"

	"kb-patient-profile/internal/storage"
)

// ScoringHandlers serves the Wave 2.6 scoring REST endpoints backed by
// a ScoringStore.
type ScoringHandlers struct {
	store *storage.ScoringStore
}

// NewScoringHandlers constructs a handler set bound to store.
func NewScoringHandlers(store *storage.ScoringStore) *ScoringHandlers {
	return &ScoringHandlers{store: store}
}

// RegisterRoutes wires the scoring endpoints onto g (typically the /v2
// router group).
func (h *ScoringHandlers) RegisterRoutes(g *gin.RouterGroup) {
	g.POST("/residents/:resident_id/cfs", h.createCFS)
	g.POST("/residents/:resident_id/akps", h.createAKPS)
	g.GET("/residents/:resident_id/scores/current", h.getCurrentScores)
	g.GET("/residents/:resident_id/cfs/history", h.getCFSHistory)
	g.GET("/residents/:resident_id/akps/history", h.getAKPSHistory)
	g.GET("/residents/:resident_id/dbi/history", h.getDBIHistory)
	g.GET("/residents/:resident_id/acb/history", h.getACBHistory)
}

// createCFSBody mirrors client.CreateCFSScoreRequest. ResidentRef is
// taken from the path and overrides any body value.
type createCFSBody struct {
	AssessedAt        time.Time `json:"assessed_at"`
	AssessorRoleRef   uuid.UUID `json:"assessor_role_ref"`
	InstrumentVersion string    `json:"instrument_version"`
	Score             int       `json:"score"`
	Rationale         string    `json:"rationale,omitempty"`
}

func (h *ScoringHandlers) createCFS(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	var body createCFSBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in := models.CFSScore{
		ResidentRef:       rid,
		AssessedAt:        body.AssessedAt,
		AssessorRoleRef:   body.AssessorRoleRef,
		InstrumentVersion: body.InstrumentVersion,
		Score:             body.Score,
		Rationale:         body.Rationale,
	}
	if in.AssessedAt.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assessed_at is required"})
		return
	}
	if err := validation.ValidateCFSScore(in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.CreateCFSScore(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

// createAKPSBody mirrors client.CreateAKPSScoreRequest.
type createAKPSBody struct {
	AssessedAt        time.Time `json:"assessed_at"`
	AssessorRoleRef   uuid.UUID `json:"assessor_role_ref"`
	InstrumentVersion string    `json:"instrument_version"`
	Score             int       `json:"score"`
	Rationale         string    `json:"rationale,omitempty"`
}

func (h *ScoringHandlers) createAKPS(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	var body createAKPSBody
	if err := c.BindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	in := models.AKPSScore{
		ResidentRef:       rid,
		AssessedAt:        body.AssessedAt,
		AssessorRoleRef:   body.AssessorRoleRef,
		InstrumentVersion: body.InstrumentVersion,
		Score:             body.Score,
		Rationale:         body.Rationale,
	}
	if in.AssessedAt.IsZero() {
		c.JSON(http.StatusBadRequest, gin.H{"error": "assessed_at is required"})
		return
	}
	if err := validation.ValidateAKPSScore(in); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	out, err := h.store.CreateAKPSScore(c.Request.Context(), in)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ScoringHandlers) getCurrentScores(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.CurrentScoresByResident(c.Request.Context(), rid)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

func (h *ScoringHandlers) getCFSHistory(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.ListCFSHistory(c.Request.Context(), rid)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.CFSScore{}
	}
	c.JSON(http.StatusOK, out)
}

func (h *ScoringHandlers) getAKPSHistory(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.ListAKPSHistory(c.Request.Context(), rid)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.AKPSScore{}
	}
	c.JSON(http.StatusOK, out)
}

func (h *ScoringHandlers) getDBIHistory(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.ListDBIHistory(c.Request.Context(), rid)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.DBIScore{}
	}
	c.JSON(http.StatusOK, out)
}

func (h *ScoringHandlers) getACBHistory(c *gin.Context) {
	rid, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	out, err := h.store.ListACBHistory(c.Request.Context(), rid)
	if err != nil {
		respondError(c, err)
		return
	}
	if out == nil {
		out = []models.ACBScore{}
	}
	c.JSON(http.StatusOK, out)
}
