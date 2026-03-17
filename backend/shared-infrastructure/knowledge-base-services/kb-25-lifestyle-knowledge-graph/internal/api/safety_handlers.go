package api

import (
	"net/http"

	"kb-25-lifestyle-knowledge-graph/internal/clients"
	"kb-25-lifestyle-knowledge-graph/internal/models"

	"github.com/gin-gonic/gin"
)

func (s *Server) checkSafety(c *gin.Context) {
	var req models.SafetyCheckRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST", nil)
		return
	}

	patient, err := s.kb20Client.GetPatientSnapshot(req.PatientID)
	if err != nil {
		s.logger.Warn("KB-20 unavailable, using empty patient context")
		patient = &clients.PatientSnapshot{PatientID: req.PatientID}
	}

	result, err := s.safetyEngine.CheckSafety(c.Request.Context(), patient, req.Interventions, req.Medications)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "safety check failed", "SAFETY_ERROR", nil)
		return
	}

	sendSuccess(c, result, nil)
}
