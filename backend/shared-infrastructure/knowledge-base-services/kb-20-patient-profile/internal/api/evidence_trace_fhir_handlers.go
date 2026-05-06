// Wave 5.3 — FHIR egress for one EvidenceTraceNode.
//
// GET /v2/evidence-trace/:id/fhir — fetches the node, dispatches via
// fhir.MapEvidenceTrace, and returns the single resource (Provenance OR
// AuditEvent) per the Layer 2 doc §1.6 dual-resource pattern.
package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/fhir"
)

// getEvidenceTraceFHIR returns the FHIR resource for one EvidenceTraceNode.
func (h *V2SubstrateHandlers) getEvidenceTraceFHIR(c *gin.Context) {
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
	rt, res, err := fhir.MapEvidenceTrace(*n)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Header("X-FHIR-Resource-Type", rt)
	c.JSON(http.StatusOK, res)
}
