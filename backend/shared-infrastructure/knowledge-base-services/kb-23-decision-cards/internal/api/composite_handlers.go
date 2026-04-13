package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// handleCompositeSynthesize handles POST /api/v1/composite-cards/synthesize/:patientId
//
// Trigger endpoint called by KB-26 after a successful BP context
// classification so that active cards created in the last 72 hours
// (masked HTN + medication timing + selection bias, etc.) are folded
// into a single CompositeCardSignal with the most-restrictive MCU gate.
//
// Contract: the handler is idempotent-ish — calling it multiple times
// for the same patient produces multiple composite rows, so callers
// should only invoke it when a classification has actually changed or
// been re-run. When no active cards exist the handler returns 200 OK
// with `{"composite_created": false}` rather than 404, because the
// caller (a batch job) cannot distinguish "no cards" from a real error.
func (s *Server) handleCompositeSynthesize(c *gin.Context) {
	patientIDRaw := c.Param("patientId")
	patientID, err := uuid.Parse(patientIDRaw)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_patient_id"})
		return
	}

	if s.compositeService == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "composite_service_unavailable"})
		return
	}

	composite, err := s.compositeService.Synthesize(c.Request.Context(), patientID)
	if err != nil {
		s.log.Error("composite synthesise failed",
			zap.String("patient_id", patientIDRaw), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "composite_synthesize_failed"})
		return
	}

	if composite == nil {
		c.JSON(http.StatusOK, gin.H{"composite_created": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"composite_created":     true,
		"composite_id":          composite.CompositeID.String(),
		"most_restrictive_gate": composite.MostRestrictiveGate,
		"urgency_upgraded":      composite.UrgencyUpgraded,
	})
}
