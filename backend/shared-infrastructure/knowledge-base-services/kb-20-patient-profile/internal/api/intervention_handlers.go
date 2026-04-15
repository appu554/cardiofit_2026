package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/services"
)

// getInterventionTimeline returns the most recent clinical intervention
// per therapeutic-inertia domain for the given patient. Phase 7 P7-D:
// consumed by KB-23's InertiaInputAssembler to populate the
// LastIntervention field on each DomainInertiaInput before running
// DetectInertia.
func (s *Server) getInterventionTimeline(c *gin.Context) {
	patientID := c.Param("id")

	svc := services.NewInterventionTimelineService(s.db.DB, s.logger)
	result, err := svc.BuildTimeline(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build intervention timeline: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
