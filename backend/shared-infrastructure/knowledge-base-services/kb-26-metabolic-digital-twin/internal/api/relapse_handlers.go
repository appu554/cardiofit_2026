package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// GET /api/v1/kb26/relapse/:patientId/nadir
func (s *Server) getNadir(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	nadir, err := s.relapseDetector.GetNadir(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "no nadir data", "NADIR_NOT_FOUND", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"nadir":      nadir,
	})
}

// POST /api/v1/kb26/relapse/:patientId/check
func (s *Server) checkRelapse(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	event, err := s.relapseDetector.CheckRelapse(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), "RELAPSE_CHECK_ERROR", nil)
		return
	}

	if event == nil {
		c.JSON(http.StatusOK, gin.H{
			"patient_id": patientID,
			"relapse":    false,
			"message":    "no relapse detected — requires 2 consecutive quarters of sustained MRI rise >15 from nadir",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"relapse":    true,
		"event":      event,
	})
}

// GET /api/v1/kb26/relapse/:patientId/history
func (s *Server) getRelapseHistory(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	events, err := s.relapseDetector.GetRelapseHistory(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, err.Error(), "RELAPSE_HISTORY_ERROR", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"count":      len(events),
		"events":     events,
	})
}
