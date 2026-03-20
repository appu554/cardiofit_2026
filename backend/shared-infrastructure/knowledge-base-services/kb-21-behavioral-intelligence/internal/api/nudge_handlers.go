package api

import (
	"net/http"

	"kb-21-behavioral-intelligence/internal/models"
	"kb-21-behavioral-intelligence/internal/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// selectNudge chooses the best coaching technique for a patient.
// POST /api/v1/patient/:patient_id/nudge/select
func (s *Server) selectNudge(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patient_id is required", "INVALID_REQUEST", nil)
		return
	}

	// Build nudge request from patient's current state
	var body struct {
		Channel         models.InteractionChannel `json:"channel"`
		Language        string                    `json:"language"`
		Season          models.EngagementSeason   `json:"season,omitempty"`
		HasTriggerEvent bool                      `json:"has_trigger_event,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		body.Channel = models.ChannelWhatsApp
		body.Language = "hi"
	}

	// Get current behavioral state (log errors but proceed with defaults)
	profile, profileErr := s.engagementService.GetEngagementProfile(patientID)
	if profileErr != nil {
		s.logger.Warn("engagement profile unavailable for nudge selection", zap.String("patient_id", patientID), zap.Error(profileErr))
	}
	states, adherenceErr := s.adherenceService.GetAdherence(patientID)
	if adherenceErr != nil {
		s.logger.Warn("adherence data unavailable for nudge selection", zap.String("patient_id", patientID), zap.Error(adherenceErr))
	}

	var adherenceScore, adherenceScore7d float64
	var adherenceTrend models.AdherenceTrend
	if len(states) > 0 {
		adherenceScore = states[0].AdherenceScore
		adherenceScore7d = states[0].AdherenceScore7d
		adherenceTrend = states[0].AdherenceTrend
	}

	phenotype := models.PhenotypeSteady
	if profile != nil {
		phenotype = profile.Phenotype
	}

	req := services.NudgeRequest{
		PatientID:        patientID,
		Channel:          body.Channel,
		Language:         body.Language,
		AdherenceScore:   adherenceScore,
		AdherenceScore7d: adherenceScore7d,
		AdherenceTrend:   adherenceTrend,
		Phenotype:        phenotype,
		Season:           body.Season,
		HasTriggerEvent:  body.HasTriggerEvent,
		Signals: services.BarrierSignals{
			DrugClassCount: len(states),
		},
	}

	result, err := s.nudgeEngine.SelectNudge(req)
	if err != nil {
		s.logger.Error("nudge selection failed", zap.Error(err))
		sendError(c, http.StatusInternalServerError, "Nudge selection failed", "SELECT_FAILED", nil)
		return
	}

	if result == nil {
		sendSuccess(c, gin.H{
			"patient_id": patientID,
			"status":     "SKIPPED",
			"reason":     "Daily limit reached or all techniques fatigued",
		}, nil)
		return
	}

	// Record delivery
	record, err := s.nudgeEngine.RecordDelivery(patientID, result, body.Channel, body.Language)
	if err != nil {
		s.logger.Error("nudge delivery recording failed", zap.Error(err))
	}

	sendSuccess(c, gin.H{
		"patient_id":     patientID,
		"technique":      result.Technique,
		"technique_name": result.TechniqueName,
		"nudge_type":     result.NudgeType,
		"phase":          result.Phase,
		"barrier":        result.Barrier,
		"reason":         result.Reason,
		"nudge_record":   record,
	}, nil)
}

// observeNudgeOutcome records whether a nudge led to improved adherence.
// POST /api/v1/patient/:patient_id/nudge/outcome
func (s *Server) observeNudgeOutcome(c *gin.Context) {
	patientID := c.Param("patient_id")

	var body struct {
		Technique models.TechniqueID `json:"technique" binding:"required"`
		Success   bool               `json:"success"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request", "VALIDATION_ERROR",
			map[string]interface{}{"error": err.Error()})
		return
	}

	if err := s.nudgeEngine.ObserveOutcome(patientID, body.Technique, body.Success); err != nil {
		sendError(c, http.StatusInternalServerError, "Outcome recording failed", "OBSERVE_FAILED", nil)
		return
	}

	sendSuccess(c, gin.H{
		"patient_id": patientID,
		"technique":  body.Technique,
		"success":    body.Success,
		"status":     "posterior_updated",
	}, nil)
}

// getTechniqueEffectiveness returns the Bayesian posteriors for all techniques.
// GET /api/v1/patient/:patient_id/techniques
func (s *Server) getTechniqueEffectiveness(c *gin.Context) {
	patientID := c.Param("patient_id")
	records, err := s.nudgeEngine.GetPatientTechniques(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get techniques", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, records, map[string]interface{}{
		"patient_id": patientID,
		"count":      len(records),
	})
}

// getMotivationPhase returns the patient's current motivation phase (E5).
// GET /api/v1/patient/:patient_id/motivation-phase
func (s *Server) getMotivationPhase(c *gin.Context) {
	patientID := c.Param("patient_id")
	phase, err := s.nudgeEngine.GetPatientPhase(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get phase", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, phase, nil)
}
