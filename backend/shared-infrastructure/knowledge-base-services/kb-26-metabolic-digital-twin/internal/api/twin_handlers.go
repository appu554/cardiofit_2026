package api

import (
	"net/http"
	"strconv"
	"time"

	"kb-26-metabolic-digital-twin/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// getTwin returns the latest twin state for a patient.
// GET /api/v1/kb26/twin/:patientId
func (s *Server) getTwin(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found", "TWIN_NOT_FOUND", nil)
		return
	}

	sendSuccess(c, twin, map[string]interface{}{
		"patient_id":    patientID.String(),
		"state_version": twin.StateVersion,
	})
}

// getTwinHistory returns the N most recent twin state snapshots for a patient.
// GET /api/v1/kb26/twin/:patientId/history?limit=10
func (s *Server) getTwinHistory(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	limit := 10
	if limitStr := c.Query("limit"); limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	states, err := s.twinUpdater.GetHistory(patientID, limit)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "failed to retrieve history", "HISTORY_ERROR", nil)
		return
	}

	sendSuccess(c, states, map[string]interface{}{
		"patient_id": patientID.String(),
		"count":      len(states),
	})
}

// syncTwin re-derives Tier 2 fields from the latest twin state and persists a new snapshot.
// POST /api/v1/kb26/sync/:patientId
func (s *Server) syncTwin(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	existing, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found — create initial state first", "TWIN_NOT_FOUND", nil)
		return
	}

	// Re-derive MAP from SBP/DBP if both are present
	newTwin := *existing
	newTwin.ID = uuid.New()
	newTwin.UpdateSource = "sync"
	newTwin.UpdatedAt = time.Now().UTC()

	if existing.SBP14dMean != nil && existing.DBP14dMean != nil {
		mapVal := services.ComputeMAP(*existing.SBP14dMean, *existing.DBP14dMean)
		newTwin.MAPValue = &mapVal
	}

	if err := s.twinUpdater.CreateSnapshot(&newTwin); err != nil {
		sendError(c, http.StatusInternalServerError, "failed to create twin snapshot", "SNAPSHOT_ERROR", nil)
		return
	}

	sendSuccess(c, newTwin, map[string]interface{}{
		"patient_id":    patientID.String(),
		"state_version": newTwin.StateVersion,
		"source":        "sync",
	})
}
