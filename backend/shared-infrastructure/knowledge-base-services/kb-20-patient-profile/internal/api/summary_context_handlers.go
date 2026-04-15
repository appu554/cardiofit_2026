package api

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"kb-patient-profile/internal/services"
)

// getSummaryContext returns the cross-cutting patient snapshot KB-23's
// card-generation pipeline needs to evaluate gate templates, detect
// inertia, and populate decision cards. Phase 8 P8-1.
//
// This is the handler that every Phase 7 card path was implicitly
// calling since Phase 6 — FetchSummaryContext in the KB-23 client
// hits /patient/:id/summary-context, but no handler existed until now.
// Every card generation silently 404'd and produced nothing for real
// patients. This endpoint closes that loop.
//
// Response envelope: { "success": true, "data": SummaryContext }.
// Missing patients return 404. Internal errors return 500 with the
// error surfaced in the message so downstream debugging is possible.
func (s *Server) getSummaryContext(c *gin.Context) {
	patientID := c.Param("id")

	// Phase 8 P8-3: the CGM fetcher is injected from main.go via
	// SetKB26CGMFetcher after the KB-26 HTTP client is constructed.
	// Phase 8 P8-5: the safety event recorder is wired at server
	// construction time and derives the confounder flags from the
	// safety_events table.
	//
	// Both dependencies are nil-safe — local dev, tests, or
	// deployments where KB-26 / safety_events are unavailable all
	// degrade cleanly (HasCGM=false, confounder flags all false,
	// falling back to HbA1c glycaemic path + no MCU gate confounder
	// overrides).
	svc := services.NewSummaryContextService(
		s.db.DB,
		s.kb26CGMFetcher,
		s.safetyRecorder,
		s.logger,
	)
	summary, err := svc.BuildContext(c.Request.Context(), patientID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to build summary context: " + err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": summary})
}
