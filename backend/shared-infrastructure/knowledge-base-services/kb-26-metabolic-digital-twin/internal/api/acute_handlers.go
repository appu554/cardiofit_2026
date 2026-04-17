package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"
)

// --- Acute-on-Chronic Detection Endpoints ---

// GET /api/v1/kb26/acute/:patientId — returns active (unresolved) acute events.
func (s *Server) getAcuteEvents(c *gin.Context) {
	patientID := c.Param("patientId")
	if s.acuteRepo == nil {
		sendError(c, http.StatusServiceUnavailable, "acute detection not configured", "ACUTE_NOT_CONFIGURED", nil)
		return
	}

	events, err := s.acuteRepo.FetchActiveEvents(patientID)
	if err != nil {
		s.logger.Error("failed to fetch acute events", zap.String("patient_id", patientID), zap.Error(err))
		sendError(c, http.StatusInternalServerError, "failed to fetch acute events", "ACUTE_FETCH_ERROR", nil)
		return
	}

	sendSuccess(c, gin.H{
		"patient_id": patientID,
		"count":      len(events),
		"events":     events,
	}, nil)
}

// GET /api/v1/kb26/acute/:patientId/baselines — returns current baseline snapshots.
func (s *Server) getPatientBaselines(c *gin.Context) {
	patientID := c.Param("patientId")
	if s.acuteRepo == nil {
		sendError(c, http.StatusServiceUnavailable, "acute detection not configured", "ACUTE_NOT_CONFIGURED", nil)
		return
	}

	baselines, err := s.acuteRepo.FetchAllBaselines(patientID)
	if err != nil {
		s.logger.Error("failed to fetch baselines", zap.String("patient_id", patientID), zap.Error(err))
		sendError(c, http.StatusInternalServerError, "failed to fetch baselines", "BASELINE_FETCH_ERROR", nil)
		return
	}

	sendSuccess(c, gin.H{
		"patient_id": patientID,
		"count":      len(baselines),
		"baselines":  baselines,
	}, nil)
}

// acuteReadingRequest is the JSON body for POST /acute/:patientId/reading.
type acuteReadingRequest struct {
	VitalType string  `json:"vital_type" binding:"required"`
	Value     float64 `json:"value" binding:"required"`
	Timestamp string  `json:"timestamp"` // RFC3339; defaults to now
}

// POST /api/v1/kb26/acute/:patientId/reading — trigger detection for a new reading.
func (s *Server) processAcuteReading(c *gin.Context) {
	patientID := c.Param("patientId")
	if s.acuteRepo == nil || s.acuteHandler == nil {
		sendError(c, http.StatusServiceUnavailable, "acute detection not configured", "ACUTE_NOT_CONFIGURED", nil)
		return
	}

	var req acuteReadingRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request body", "INVALID_REQUEST", map[string]interface{}{
			"details": err.Error(),
		})
		return
	}

	ts := time.Now().UTC()
	if req.Timestamp != "" {
		parsed, err := time.Parse(time.RFC3339, req.Timestamp)
		if err != nil {
			sendError(c, http.StatusBadRequest, "invalid timestamp format (expected RFC3339)", "INVALID_TIMESTAMP", nil)
			return
		}
		ts = parsed
	}

	// Fetch recent readings for baseline computation. For the API endpoint
	// we pass empty slices — the handler will use the persisted baseline if
	// available. In the streaming pipeline, readings are supplied directly.
	var readings []float64
	var readingTimestamps []time.Time

	// Fetch recent deviations for compound check (last 72 hours).
	since := ts.Add(-72 * time.Hour)
	recentEvents, err := s.acuteRepo.FetchRecentDeviations(patientID, since)
	if err != nil {
		s.logger.Warn("failed to fetch recent deviations", zap.Error(err))
		// Non-fatal — proceed with empty slice.
		recentEvents = nil
	}

	// Convert recent events to DeviationResult slice for compound detection.
	var recentDeviations []models.DeviationResult
	for _, ev := range recentEvents {
		recentDeviations = append(recentDeviations, models.DeviationResult{
			VitalSignType:        ev.VitalSignType,
			CurrentValue:         ev.CurrentValue,
			BaselineMedian:       ev.BaselineMedian,
			DeviationAbsolute:    ev.DeviationAbsolute,
			DeviationPercent:     ev.DeviationPercent,
			Direction:            ev.Direction,
			ClinicalSignificance: ev.Severity,
			GapAmplified:         ev.GapAmplified,
			ConfounderDampened:    ev.ConfounderDampened,
		})
	}

	// Fetch active events for resolution check.
	activeEvents, err := s.acuteRepo.FetchActiveEvents(patientID)
	if err != nil {
		s.logger.Warn("failed to fetch active events", zap.Error(err))
		activeEvents = nil
	}

	// Run the detection pipeline. Context fields are left empty — the
	// caller can supply richer context via the streaming pipeline.
	event, resolvedIDs := s.acuteHandler.HandleNewReading(
		patientID,
		req.VitalType,
		req.Value,
		ts,
		readings,
		readingTimestamps,
		services.DeviationContext{},
		recentDeviations,
		services.CompoundContext{},
		activeEvents,
	)

	if event == nil {
		c.JSON(http.StatusNoContent, nil)
		return
	}

	sendSuccess(c, gin.H{
		"event":        event,
		"resolved_ids": resolvedIDs,
	}, nil)
}
