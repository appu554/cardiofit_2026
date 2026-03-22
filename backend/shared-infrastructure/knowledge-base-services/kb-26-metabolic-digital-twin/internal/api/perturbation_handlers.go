package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// perturbationAnalysis runs a perturbation-suppression analysis for a patient.
// POST /api/v1/kb26/perturbation
// Stub: full implementation pending Track 3 perturbation-suppression task.
func (s *Server) perturbationAnalysis(c *gin.Context) {
	sendError(c, http.StatusNotImplemented, "perturbation analysis not yet implemented", "NOT_IMPLEMENTED", nil)
}
