package api

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// EscalationMetrics aggregates escalation performance data.
type EscalationMetrics struct {
	TotalByTier     map[string]int     `json:"total_by_tier"`
	TotalByState    map[string]int     `json:"total_by_state"`
	AvgResponseMins map[string]float64 `json:"avg_response_minutes_by_tier"`
	TimeoutRate     map[string]float64 `json:"timeout_rate_by_tier"`
}

// acknowledgeEscalation handles POST /api/v1/escalation/:id/acknowledge
func (s *Server) acknowledgeEscalation(c *gin.Context) {
	if s.escalationManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "escalation_manager_not_configured"})
		return
	}

	idStr := c.Param("id")
	eventID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id", "message": "escalation ID must be a valid UUID"})
		return
	}

	var body struct {
		ClinicianID string `json:"clinician_id" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_payload", "message": err.Error()})
		return
	}

	var event models.EscalationEvent
	if err := s.db.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		s.log.Warn("escalation event not found", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	now := time.Now()
	event.AcknowledgedAt = &now
	event.AcknowledgedBy = body.ClinicianID
	event.CurrentState = string(models.StateAcknowledged)

	if err := s.db.DB.Save(&event).Error; err != nil {
		s.log.Error("failed to save acknowledged escalation", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save_failed"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// recordEscalationAction handles POST /api/v1/escalation/:id/action
func (s *Server) recordEscalationAction(c *gin.Context) {
	if s.escalationManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "escalation_manager_not_configured"})
		return
	}

	idStr := c.Param("id")
	eventID, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_id", "message": "escalation ID must be a valid UUID"})
		return
	}

	var body struct {
		ActionType   string `json:"action_type" binding:"required"`
		ActionDetail string `json:"action_detail"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_payload", "message": err.Error()})
		return
	}

	var event models.EscalationEvent
	if err := s.db.DB.Where("id = ?", eventID).First(&event).Error; err != nil {
		s.log.Warn("escalation event not found", zap.String("id", idStr), zap.Error(err))
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	now := time.Now()
	event.ActedAt = &now
	event.ActionType = body.ActionType
	event.ActionDetail = body.ActionDetail
	event.CurrentState = string(models.StateActed)

	if err := s.db.DB.Save(&event).Error; err != nil {
		s.log.Error("failed to save escalation action", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "save_failed"})
		return
	}

	c.JSON(http.StatusOK, event)
}

// getPatientEscalations handles GET /api/v1/escalation/patient/:patientId
func (s *Server) getPatientEscalations(c *gin.Context) {
	if s.escalationManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "escalation_manager_not_configured"})
		return
	}

	patientID := c.Param("patientId")
	if patientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_patient_id"})
		return
	}

	var events []models.EscalationEvent
	if err := s.db.DB.Where("patient_id = ?", patientID).
		Order("created_at DESC").
		Limit(50).
		Find(&events).Error; err != nil {
		s.log.Error("failed to query patient escalations", zap.String("patient_id", patientID), zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "query_failed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"count":      len(events),
		"events":     events,
	})
}

// getEscalationMetrics handles GET /api/v1/escalation/metrics
func (s *Server) getEscalationMetrics(c *gin.Context) {
	if s.escalationManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "escalation_manager_not_configured"})
		return
	}

	metrics := EscalationMetrics{
		TotalByTier:     make(map[string]int),
		TotalByState:    make(map[string]int),
		AvgResponseMins: make(map[string]float64),
		TimeoutRate:     make(map[string]float64),
	}

	// Count by tier
	type tierCount struct {
		EscalationTier string
		Count          int
	}
	var tierCounts []tierCount
	s.db.DB.Model(&models.EscalationEvent{}).
		Select("escalation_tier, COUNT(*) as count").
		Group("escalation_tier").
		Scan(&tierCounts)
	for _, tc := range tierCounts {
		metrics.TotalByTier[tc.EscalationTier] = tc.Count
	}

	// Count by state
	type stateCount struct {
		CurrentState string
		Count        int
	}
	var stateCounts []stateCount
	s.db.DB.Model(&models.EscalationEvent{}).
		Select("current_state, COUNT(*) as count").
		Group("current_state").
		Scan(&stateCounts)
	for _, sc := range stateCounts {
		metrics.TotalByState[sc.CurrentState] = sc.Count
	}

	// Average response time (created_at -> acknowledged_at) by tier
	type avgResponse struct {
		EscalationTier string
		AvgMins        float64
	}
	var avgResponses []avgResponse
	s.db.DB.Model(&models.EscalationEvent{}).
		Select("escalation_tier, AVG(EXTRACT(EPOCH FROM (acknowledged_at - created_at)) / 60) as avg_mins").
		Where("acknowledged_at IS NOT NULL").
		Group("escalation_tier").
		Scan(&avgResponses)
	for _, ar := range avgResponses {
		metrics.AvgResponseMins[ar.EscalationTier] = ar.AvgMins
	}

	// Timeout rate by tier: expired / total
	for tier, total := range metrics.TotalByTier {
		if total == 0 {
			continue
		}
		var expiredCount int64
		s.db.DB.Model(&models.EscalationEvent{}).
			Where("escalation_tier = ? AND current_state = ?", tier, models.StateExpired).
			Count(&expiredCount)
		metrics.TimeoutRate[tier] = float64(expiredCount) / float64(total)
	}

	c.JSON(http.StatusOK, metrics)
}

// upsertClinicianPreferences handles POST /api/v1/clinician/:clinicianId/preferences
func (s *Server) upsertClinicianPreferences(c *gin.Context) {
	if s.escalationManager == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "escalation_manager_not_configured"})
		return
	}

	clinicianID := c.Param("clinicianId")
	if clinicianID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_clinician_id"})
		return
	}

	var prefs models.ClinicianPreferences
	if err := c.ShouldBindJSON(&prefs); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_payload", "message": err.Error()})
		return
	}
	prefs.ClinicianID = clinicianID

	// SAFETY tier cannot be opted out: ensure preferred_channels does not
	// exclude all SAFETY-capable channels.
	channels := strings.ToUpper(prefs.PreferredChannels)
	if channels != "" && !strings.Contains(channels, "SMS") && !strings.Contains(channels, "PUSH") && !strings.Contains(channels, "PAGER") {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "safety_channel_required",
			"message": "preferred_channels must include at least one SAFETY-capable channel (sms, push, or pager)",
		})
		return
	}

	result := s.db.DB.Where("clinician_id = ?", clinicianID).
		Assign(models.ClinicianPreferences{
			PreferredChannels:       prefs.PreferredChannels,
			QuietHoursStart:         prefs.QuietHoursStart,
			QuietHoursEnd:           prefs.QuietHoursEnd,
			Timezone:                prefs.Timezone,
			MaxNotificationsPerHour: prefs.MaxNotificationsPerHour,
		}).
		FirstOrCreate(&prefs)
	if result.Error != nil {
		s.log.Error("failed to upsert clinician preferences", zap.String("clinician_id", clinicianID), zap.Error(result.Error))
		c.JSON(http.StatusInternalServerError, gin.H{"error": "upsert_failed"})
		return
	}

	c.JSON(http.StatusOK, prefs)
}
