package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// classifyBPContext handles POST /api/v1/kb26/bp-context/:patientId.
// Body is empty — the patient ID in the path is sufficient; all other
// inputs are fetched from KB-20 and KB-21 by the orchestrator.
func (s *Server) classifyBPContext(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patientId is required", "MISSING_PATIENT_ID", nil)
		return
	}

	result, err := s.bpContextOrchestrator.Classify(c.Request.Context(), patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "BP context classification failed", "BP_CONTEXT_FAILED", map[string]interface{}{
			"error": err.Error(),
		})
		return
	}

	sendSuccess(c, result, map[string]interface{}{
		"patient_id": patientID,
	})
}
