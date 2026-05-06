// Package api — Wave 5.2 EvidenceTrace query API REST handlers.
//
// These handlers shape the kb-20 V2SubstrateStore + EvidenceTraceEdgeAdapter
// pair through the pure-Go evidence_trace.LineageOf / ConsequencesOf /
// ReasoningWindow helpers and emit JSON suited for Layer 3 + ACQSC
// regulator-audit consumers.
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/evidence_trace"

	"kb-patient-profile/internal/storage"
)

// edgeAdapterFor wraps the handler's V2SubstrateStore as an EdgeStore so the
// pure helpers can traverse via OutEdges / InEdges.
func (h *V2SubstrateHandlers) edgeAdapterFor() evidence_trace.EdgeStore {
	return storage.EvidenceTraceEdgeAdapter{Store: h.store}
}

// GET /v2/evidence-trace/recommendations/:id/lineage[?depth=N]
// Returns the backward traversal — every upstream evidence node — for one
// Recommendation. Uses derived_from + evidence_for edges only.
func (h *V2SubstrateHandlers) getRecommendationLineage(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	depth, ok := parseTraversalDepth(c)
	if !ok {
		return
	}
	out, err := evidence_trace.LineageOf(c.Request.Context(), id, h.store, h.edgeAdapterFor(), depth)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// GET /v2/evidence-trace/observations/:id/consequences[?depth=N]
// Returns the forward traversal — every downstream node — for one
// observation-seed node. Uses led_to edges.
func (h *V2SubstrateHandlers) getObservationConsequences(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}
	depth, ok := parseTraversalDepth(c)
	if !ok {
		return
	}
	out, err := evidence_trace.ConsequencesOf(c.Request.Context(), id, h.store, h.edgeAdapterFor(), depth)
	if err != nil {
		respondError(c, err)
		return
	}
	c.JSON(http.StatusOK, out)
}

// GET /v2/residents/:resident_id/reasoning-window?from=RFC3339&to=RFC3339
// Regulator-audit window query: the per-resident rollup of every node in
// the recorded_at window.
func (h *V2SubstrateHandlers) getReasoningWindow(c *gin.Context) {
	residentID, err := uuid.Parse(c.Param("resident_id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid resident_id"})
		return
	}
	fromStr := c.Query("from")
	toStr := c.Query("to")
	if fromStr == "" || toStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "from and to query parameters are required (RFC3339)"})
		return
	}
	from, err := time.Parse(time.RFC3339, fromStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid from (want RFC3339)"})
		return
	}
	to, err := time.Parse(time.RFC3339, toStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid to (want RFC3339)"})
		return
	}
	out, err := evidence_trace.ReasoningWindow(c.Request.Context(), residentID, from, to, h.store)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, out)
}
