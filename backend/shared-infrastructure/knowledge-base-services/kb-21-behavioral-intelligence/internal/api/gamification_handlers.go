package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// getPatientStreaks returns all active streaks for a patient.
// GET /api/v1/patient/:patient_id/streaks
func (s *Server) getPatientStreaks(c *gin.Context) {
	patientID := c.Param("patient_id")
	if s.gamificationEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Gamification not enabled", "FEATURE_DISABLED", nil)
		return
	}

	streaks, err := s.gamificationEngine.GetPatientStreaks(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get streaks", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, streaks, map[string]interface{}{
		"patient_id": patientID,
		"count":      len(streaks),
	})
}

// getPatientMilestones returns all milestones for a patient.
// GET /api/v1/patient/:patient_id/milestones
func (s *Server) getPatientMilestones(c *gin.Context) {
	patientID := c.Param("patient_id")
	if s.gamificationEngine == nil {
		sendError(c, http.StatusServiceUnavailable, "Gamification not enabled", "FEATURE_DISABLED", nil)
		return
	}

	milestones, err := s.gamificationEngine.GetPatientMilestones(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get milestones", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, milestones, map[string]interface{}{
		"patient_id": patientID,
		"count":      len(milestones),
	})
}

// getOptimalDeliveryTime returns the best nudge delivery time for a patient.
// GET /api/v1/patient/:patient_id/optimal-timing
func (s *Server) getOptimalDeliveryTime(c *gin.Context) {
	patientID := c.Param("patient_id")
	if s.timingBandit == nil {
		sendError(c, http.StatusServiceUnavailable, "Timing optimization not enabled", "FEATURE_DISABLED", nil)
		return
	}

	slot, err := s.timingBandit.GetOptimalTime(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get timing", "QUERY_FAILED", nil)
		return
	}

	profiles, _ := s.timingBandit.EnsurePatientProfiles(patientID)

	sendSuccess(c, gin.H{
		"patient_id":   patientID,
		"optimal_slot": slot,
		"all_profiles": profiles,
	}, nil)
}
