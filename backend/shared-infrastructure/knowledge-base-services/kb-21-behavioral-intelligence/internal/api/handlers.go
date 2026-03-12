package api

import (
	"encoding/json"
	"io"
	"net/http"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// --- Interaction handlers ---

func (s *Server) recordInteraction(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patient_id is required", "INVALID_REQUEST", nil)
		return
	}

	var req models.RecordInteractionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "Invalid request body", "VALIDATION_ERROR",
			map[string]interface{}{"error": err.Error()})
		return
	}
	req.PatientID = patientID

	event, err := s.adherenceService.RecordInteraction(req)
	if err != nil {
		s.logger.Error("failed to record interaction", zap.Error(err))
		sendError(c, http.StatusInternalServerError, "Failed to record interaction", "RECORD_FAILED", nil)
		return
	}

	// Evaluate hypo risk asynchronously after recording
	go func() {
		if _, err := s.hypoRiskService.EvaluateHypoRisk(patientID); err != nil {
			s.logger.Error("hypo risk evaluation failed", zap.Error(err))
		}
	}()

	sendSuccess(c, event, map[string]interface{}{
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// --- Adherence handlers ---

func (s *Server) getAdherence(c *gin.Context) {
	patientID := c.Param("patient_id")
	states, err := s.adherenceService.GetAdherence(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get adherence", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, states, map[string]interface{}{
		"patient_id": patientID,
		"count":      len(states),
	})
}

func (s *Server) recomputeAdherence(c *gin.Context) {
	patientID := c.Param("patient_id")
	drugClass := c.Query("drug_class")

	if drugClass == "" {
		// Recompute for all drug classes
		states, err := s.adherenceService.GetAdherence(patientID)
		if err != nil {
			sendError(c, http.StatusInternalServerError, "Failed to get adherence states", "QUERY_FAILED", nil)
			return
		}
		for _, state := range states {
			if err := s.adherenceService.RecomputeAdherence(patientID, state.DrugClass); err != nil {
				s.logger.Error("adherence recomputation failed",
					zap.String("drug_class", state.DrugClass), zap.Error(err))
			}
		}
	} else {
		if err := s.adherenceService.RecomputeAdherence(patientID, drugClass); err != nil {
			sendError(c, http.StatusInternalServerError, "Recomputation failed", "COMPUTE_FAILED", nil)
			return
		}
	}

	sendSuccess(c, gin.H{"status": "recomputed"}, nil)
}

// getAdherenceWeights returns adherence-adjusted weights for KB-22 (Finding F-06).
// KB-22 queries this alongside KB-20's stratum query and applies the adjustment
// to scale drug-ADR context modifier magnitudes.
// Formula: adjusted_weight = min(1.0, adherence_score / 0.70)
func (s *Server) getAdherenceWeights(c *gin.Context) {
	patientID := c.Param("patient_id")
	weights, err := s.adherenceService.GetAdherenceWeights(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get adherence weights", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, weights, map[string]interface{}{
		"purpose":              "KB-22 CM activation weight adjustment",
		"adherence_threshold":  0.70,
		"latency_budget_notes": "Fire in parallel with KB-20 stratum query (50ms constraint)",
	})
}

// --- Engagement handlers ---

func (s *Server) getEngagementProfile(c *gin.Context) {
	patientID := c.Param("patient_id")
	profile, err := s.engagementService.GetEngagementProfile(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get engagement profile", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, profile, nil)
}

func (s *Server) recomputeEngagement(c *gin.Context) {
	patientID := c.Param("patient_id")
	if err := s.engagementService.RecomputeProfile(patientID); err != nil {
		sendError(c, http.StatusInternalServerError, "Engagement recomputation failed", "COMPUTE_FAILED", nil)
		return
	}

	profile, _ := s.engagementService.GetEngagementProfile(patientID)
	sendSuccess(c, profile, gin.H{"status": "recomputed"})
}

// getLoopTrust returns the composite loop trust score for V-MCU (Finding F-01).
// V-MCU uses this to gate correction loop control authority.
func (s *Server) getLoopTrust(c *gin.Context) {
	patientID := c.Param("patient_id")
	trust, err := s.engagementService.GetLoopTrust(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get loop trust", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, trust, map[string]interface{}{
		"note": "Thresholds are informational. V-MCU owns control authority decisions.",
	})
}

// --- Outcome correlation handlers (Finding F-04) ---

func (s *Server) getLatestCorrelation(c *gin.Context) {
	patientID := c.Param("patient_id")
	corr, err := s.correlationService.GetLatestCorrelation(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get correlation", "QUERY_FAILED", nil)
		return
	}
	if corr == nil {
		sendSuccess(c, nil, map[string]interface{}{
			"message": "No outcome correlation data yet. Requires at least one HbA1c result.",
		})
		return
	}

	sendSuccess(c, corr, nil)
}

func (s *Server) getCorrelationHistory(c *gin.Context) {
	patientID := c.Param("patient_id")
	history, err := s.correlationService.GetCorrelationHistory(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to get correlation history", "QUERY_FAILED", nil)
		return
	}

	sendSuccess(c, history, map[string]interface{}{
		"count": len(history),
	})
}

// --- Hypoglycemia risk handler (Finding F-03) ---

func (s *Server) evaluateHypoRisk(c *gin.Context) {
	patientID := c.Param("patient_id")
	event, err := s.hypoRiskService.EvaluateHypoRisk(patientID)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "Hypo risk evaluation failed", "EVAL_FAILED", nil)
		return
	}

	if event == nil {
		sendSuccess(c, gin.H{
			"patient_id": patientID,
			"risk_level": "NORMAL",
			"message":    "No elevated hypoglycemia risk factors detected",
		}, nil)
		return
	}

	sendSuccess(c, event, map[string]interface{}{
		"consumers": []string{"KB-19 (Protocol Orchestrator)", "KB-4 (Patient Safety)"},
	})
}

// --- Webhook handlers (dev mode event ingestion) ---

func (s *Server) webhookLabResult(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Failed to read body", "READ_FAILED", nil)
		return
	}
	defer c.Request.Body.Close()

	if err := s.eventSubscriber.HandleLabResult(body); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to process lab result", "PROCESS_FAILED",
			map[string]interface{}{"error": err.Error()})
		return
	}

	sendSuccess(c, gin.H{"status": "processed"}, nil)
}

func (s *Server) webhookMedicationChanged(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		sendError(c, http.StatusBadRequest, "Failed to read body", "READ_FAILED", nil)
		return
	}
	defer c.Request.Body.Close()

	if err := s.eventSubscriber.HandleMedicationChanged(body); err != nil {
		sendError(c, http.StatusInternalServerError, "Failed to process medication change", "PROCESS_FAILED",
			map[string]interface{}{"error": err.Error()})
		return
	}

	sendSuccess(c, gin.H{"status": "processed"}, nil)
}

// --- Answer reliability handler (KB-22 contract: R-03) ---

// getAnswerReliability returns the patient's answer reliability metrics.
// KB-22's session_context_provider calls this endpoint at session start to
// obtain a reliability_modifier that scales likelihood-ratio updates.
// When no behavioral data exists, conservative defaults are returned.
func (s *Server) getAnswerReliability(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patient_id is required", "INVALID_REQUEST", nil)
		return
	}

	// Attempt to derive reliability from existing engagement and adherence data.
	var reliabilityScore float64 = 0.85
	var sampleSize int
	var confidenceLevel string = "MODERATE"

	profile, profileErr := s.engagementService.GetEngagementProfile(patientID)
	states, adherenceErr := s.adherenceService.GetAdherence(patientID)

	if profileErr == nil && profile != nil {
		// Use engagement consistency as a proxy for answer reliability.
		// Higher engagement score correlates with more reliable self-reports.
		if profile.EngagementScore > 0 {
			reliabilityScore = profile.EngagementScore
		}
		sampleSize += profile.TotalInteractions
	}

	if adherenceErr == nil && len(states) > 0 {
		// Factor in adherence data volume.
		for _, state := range states {
			sampleSize += state.TotalCheckIns
		}
	}

	// Determine confidence level from sample size.
	switch {
	case sampleSize >= 30:
		confidenceLevel = "HIGH"
	case sampleSize >= 10:
		confidenceLevel = "MODERATE"
	default:
		confidenceLevel = "LOW"
	}

	// Clamp reliability score to [0.1, 1.0].
	if reliabilityScore < 0.1 {
		reliabilityScore = 0.1
	}
	if reliabilityScore > 1.0 {
		reliabilityScore = 1.0
	}

	sendSuccess(c, gin.H{
		"patient_id":          patientID,
		"reliability_score":   reliabilityScore,
		"reliability_modifier": reliabilityScore, // KB-22 contract field
		"sample_size":         sampleSize,
		"confidence_level":    confidenceLevel,
	}, map[string]interface{}{
		"source":  "kb-21-behavioral-intelligence",
		"purpose": "KB-22 HPI session reliability modifier (R-03)",
	})
}

// --- Antihypertensive adherence handlers (Amendment 4, Wave 2) ---

// getHTNAdherence returns the aggregate antihypertensive adherence state for a patient.
// KB-23 card_builder calls this to gate HYPERTENSION_REVIEW decision cards.
// Returns per-class breakdown, quality-weighted aggregate scores, dietary sodium context,
// and the primary non-adherence reason for intervention routing.
func (s *Server) getHTNAdherence(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patient_id is required", "INVALID_REQUEST", nil)
		return
	}

	resp, err := s.adherenceService.GetAntihypertensiveAdherence(patientID)
	if err != nil {
		s.logger.Error("failed to get HTN adherence",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		sendError(c, http.StatusInternalServerError, "Failed to get HTN adherence", "QUERY_FAILED", nil)
		return
	}

	c.JSON(http.StatusOK, resp)
}

// getHTNAdherenceGate evaluates the adherence-based gate action for HTN decision cards.
// Returns the gate action (STANDARD_ESCALATION, ADHERENCE_LEAD, ADHERENCE_INTERVENTION,
// or SIDE_EFFECT_HPI) along with the full adherence state used for the decision.
//
// Decision matrix (Amendment 4):
//
//	Any class SIDE_EFFECT barrier → SIDE_EFFECT_HPI (routes to KB-22 HPI node)
//	Aggregate score >= 0.85       → STANDARD_ESCALATION
//	Aggregate score 0.60-0.84     → ADHERENCE_LEAD
//	Aggregate score < 0.60        → ADHERENCE_INTERVENTION
func (s *Server) getHTNAdherenceGate(c *gin.Context) {
	patientID := c.Param("patient_id")
	if patientID == "" {
		sendError(c, http.StatusBadRequest, "patient_id is required", "INVALID_REQUEST", nil)
		return
	}

	action, adherence, err := s.adherenceService.EvaluateHTNAdherenceGate(patientID)
	if err != nil {
		s.logger.Error("failed to evaluate HTN adherence gate",
			zap.String("patient_id", patientID),
			zap.Error(err),
		)
		sendError(c, http.StatusInternalServerError, "Failed to evaluate HTN adherence gate", "EVAL_FAILED", nil)
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"action":    string(action),
		"adherence": adherence,
	})
}

// --- Analytics handlers (Finding F-11) ---

func (s *Server) getPhenotypeDistribution(c *gin.Context) {
	type phenotypeCount struct {
		Phenotype string `json:"phenotype"`
		Count     int64  `json:"count"`
	}

	var results []phenotypeCount
	s.db.DB.Model(&models.EngagementProfile{}).
		Select("phenotype, count(*) as count").
		Group("phenotype").
		Scan(&results)

	sendSuccess(c, results, nil)
}

func (s *Server) getQuestionEffectiveness(c *gin.Context) {
	var telemetry []models.QuestionTelemetry
	query := s.db.DB.Where("active = true").Order("information_yield DESC")

	if lang := c.Query("language"); lang != "" {
		query = query.Where("language = ?", lang)
	}
	if category := c.Query("category"); category != "" {
		query = query.Where("category = ?", category)
	}

	query.Find(&telemetry)

	sendSuccess(c, telemetry, map[string]interface{}{
		"count": len(telemetry),
	})
}

func (s *Server) getCohortSnapshots(c *gin.Context) {
	var snapshots []models.CohortSnapshot

	query := s.db.DB.Order("week_of DESC")
	if limit := c.Query("limit"); limit != "" {
		query = query.Limit(12) // default last 12 weeks
	}

	query.Find(&snapshots)

	// Also compute real-time phenotype distribution for current state
	var currentDist []struct {
		Phenotype string
		Count     int64
	}
	s.db.DB.Model(&models.EngagementProfile{}).
		Select("phenotype, count(*) as count").
		Group("phenotype").
		Scan(&currentDist)

	sendSuccess(c, gin.H{
		"snapshots":            snapshots,
		"current_distribution": currentDist,
	}, nil)
}

// --- Helper for JSON parsing ---

func parseJSON(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}
