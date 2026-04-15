package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-26-metabolic-digital-twin/internal/services"
)

// getCGMLatest returns the most recent CGMPeriodReport for the given
// patient, or 404 when no report exists. Phase 7 P7-E Milestone 2:
// consumed by KB-23's InertiaInputAssembler to populate the CGM_TIR
// branch of the glycaemic domain inertia input.
func (s *Server) getCGMLatest(c *gin.Context) {
	patientID := c.Param("patientId")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patientId is required", "BAD_REQUEST", nil)
		return
	}

	repo := services.NewCGMPeriodReportRepository(s.db.DB, s.logger)
	report, err := repo.FetchLatestPeriodReport(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "failed to fetch CGM period report: "+err.Error(), "FETCH_FAILED", nil)
		return
	}
	if report == nil {
		sendError(c, http.StatusNotFound, "no CGM period report found for patient", "NOT_FOUND", nil)
		return
	}

	sendSuccess(c, report, nil)
}
