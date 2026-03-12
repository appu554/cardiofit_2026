package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// handleGenerateCard handles POST /api/v1/decision-cards
// Receives HPI_COMPLETE from KB-22 and generates a DecisionCard.
func (s *Server) handleGenerateCard(c *gin.Context) {
	start := time.Now()

	var event models.HPICompleteEvent
	if err := c.ShouldBindJSON(&event); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	s.log.Info("HPI_COMPLETE received from KB-22",
		zap.String("session_id", event.SessionID.String()),
		zap.String("patient_id", event.PatientID.String()),
		zap.String("top_diagnosis", event.TopDiagnosis),
		zap.Float64("top_posterior", event.TopPosterior),
		zap.Int("differentials", len(event.RankedDifferentials)),
		zap.Int("safety_flags", len(event.SafetyFlags)),
	)

	// 1. Select matching template
	tmpl := s.templateSelector.SelectBest(event.TopDiagnosis, event.NodeID)
	if tmpl == nil {
		s.log.Warn("no matching template found",
			zap.String("top_diagnosis", event.TopDiagnosis),
			zap.String("node_id", event.NodeID),
		)
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":   "no_matching_template",
			"message": "no CardTemplate matches the top differential",
		})
		return
	}

	// 2. Fetch patient context from KB-20
	patientCtx, err := s.kb20Client.FetchSummaryContext(c.Request.Context(), event.PatientID.String())
	if err != nil {
		s.log.Warn("KB-20 fetch failed, proceeding with limited context", zap.Error(err))
	}

	// 3. Build decision card
	card, err := s.cardBuilder.Build(c.Request.Context(), tmpl, &event, patientCtx)
	if err != nil {
		s.log.Error("card build failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "card_build_failed",
			"message": err.Error(),
		})
		return
	}

	// 4. Write MCU gate to cache
	if err := s.mcuGateCache.WriteGate(card); err != nil {
		s.log.Error("MCU gate cache write failed", zap.Error(err))
	}

	// 5. Publish MCU_GATE_CHANGED to KB-19
	go s.kb19Publisher.PublishGateChanged(card)

	// 6. Publish SAFETY_ALERT for IMMEDIATE safety flags (< 2s SLA)
	for _, flag := range event.SafetyFlags {
		if flag.Severity == "IMMEDIATE" {
			go s.kb19Publisher.PublishSafetyAlert(card.PatientID, card.SessionID, flag)
		}
	}

	// 7. Record metrics
	s.metrics.CardsGenerated.WithLabelValues(string(card.CardSource), card.TemplateID).Inc()
	s.metrics.CardGenerationLatency.Observe(float64(time.Since(start).Milliseconds()))

	c.JSON(http.StatusCreated, card)
}

// handleGetCard handles GET /api/v1/cards/:id
func (s *Server) handleGetCard(c *gin.Context) {
	id := c.Param("id")
	cardID, err := uuid.Parse(id)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_card_id"})
		return
	}

	var card models.DecisionCard
	result := s.db.DB.Preload("Recommendations").First(&card, "card_id = ?", cardID)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "card_not_found"})
		return
	}

	c.JSON(http.StatusOK, card)
}

// handleHypoglycaemiaAlert handles POST /api/v1/safety/hypoglycaemia-alert
// Phase 2 minimal handler: writes gate immediately without hysteresis.
func (s *Server) handleHypoglycaemiaAlert(c *gin.Context) {
	start := time.Now()

	var req models.SafetyAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	s.log.Warn("hypoglycaemia alert received",
		zap.String("patient_id", req.PatientID.String()),
		zap.String("source", req.Source),
		zap.Float64("glucose_mmol_l", req.GlucoseMmolL),
		zap.String("severity", req.Severity),
	)

	card, err := s.hypoHandler.HandleAlert(c.Request.Context(), &req)
	if err != nil {
		s.log.Error("hypoglycaemia handler failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "handler_failed",
			"message": err.Error(),
		})
		return
	}

	s.metrics.HypoglycaemiaAlerts.WithLabelValues(req.Severity, req.Source).Inc()
	s.metrics.SafetyAlertLatency.Observe(float64(time.Since(start).Milliseconds()))

	c.JSON(http.StatusCreated, card)
}

// handleBehavioralGapAlert handles POST /api/v1/safety/behavioral-gap-alert
// Phase 2 minimal handler: KB-21 G-01 BEHAVIORAL_GAP/DISCORDANT -> gate write.
func (s *Server) handleBehavioralGapAlert(c *gin.Context) {
	start := time.Now()

	var req models.SafetyAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	s.log.Info("behavioral gap alert received from KB-21",
		zap.String("patient_id", req.PatientID.String()),
		zap.String("response_class", req.TreatmentResponseClass),
		zap.Float64("adherence", req.MeanAdherenceScore),
		zap.Float64("hba1c_delta", req.HbA1cDelta),
	)

	card, err := s.behavioralHandler.HandleAlert(c.Request.Context(), &req)
	if err != nil {
		s.log.Error("behavioral gap handler failed", zap.Error(err))
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "handler_failed",
			"message": err.Error(),
		})
		return
	}

	s.metrics.BehavioralGapAlerts.WithLabelValues(req.TreatmentResponseClass).Inc()
	s.metrics.SafetyAlertLatency.Observe(float64(time.Since(start).Milliseconds()))

	c.JSON(http.StatusCreated, card)
}
