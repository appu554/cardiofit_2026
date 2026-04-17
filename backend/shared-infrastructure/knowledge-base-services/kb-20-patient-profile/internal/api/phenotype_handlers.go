package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-patient-profile/internal/models"
	"kb-patient-profile/internal/services"
)

// DecisionCardWebhookRequest is the minimal projection KB-23 sends after
// persisting a decision card. Phase 10 Gap 9 FHIR write-back.
type DecisionCardWebhookRequest struct {
	PatientID        string `json:"patient_id" binding:"required"`
	CardID           string `json:"card_id" binding:"required"`
	TemplateID       string `json:"template_id"`
	ClinicianSummary string `json:"clinician_summary"`
	SafetyTier       string `json:"safety_tier"`
	MCUGate          string `json:"mcu_gate"`
}

func (s *Server) handleDecisionCardWebhook(c *gin.Context) {
	var req DecisionCardWebhookRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Republish on the internal event bus → FHIR publisher picks it up.
	s.eventBus.Publish(models.EventDecisionCardGenerated, req.PatientID, map[string]interface{}{
		"card_id":           req.CardID,
		"template_id":      req.TemplateID,
		"clinician_summary": req.ClinicianSummary,
		"safety_tier":      req.SafetyTier,
		"mcu_gate":         req.MCUGate,
	})

	c.JSON(http.StatusOK, gin.H{"success": true})
}

// PhenotypeClusterRequest is the JSON body from the Python clustering pipeline.
type PhenotypeClusterRequest struct {
	RawClusterLabel   string  `json:"raw_cluster_label" binding:"required"`
	MembershipProb    float64 `json:"membership_prob"`
	SeparabilityRatio float64 `json:"separability_ratio"`
	IsNoise           bool    `json:"is_noise"`
	RunID             string  `json:"run_id" binding:"required"`
	DomainDriver      string  `json:"domain_driver,omitempty"`
}

func (s *Server) patchPhenotypeCluster(c *gin.Context) {
	patientID := c.Param("id")

	var req PhenotypeClusterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Load current patient profile
	var profile models.PatientProfile
	if err := s.db.DB.Where("patient_id = ?", patientID).First(&profile).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "patient not found"})
		return
	}

	now := time.Now().UTC()

	// Build current state from profile fields
	var currentState *models.PatientClusterState
	if profile.PhenotypeCluster != "" {
		var confidence float64
		if profile.PhenotypeConfidence != nil {
			confidence = *profile.PhenotypeConfidence
		}
		currentState = &models.PatientClusterState{
			PatientID:            patientID,
			CurrentStableCluster: profile.PhenotypeCluster,
			Confidence:           confidence,
		}
	}

	// Build stability input with config loaded from phenotype_stability.yaml
	input := services.StabilityInput{
		PatientID:         patientID,
		RawClusterLabel:   req.RawClusterLabel,
		MembershipProb:    req.MembershipProb,
		SeparabilityRatio: req.SeparabilityRatio,
		IsNoise:           req.IsNoise,
		RunDate:           now,
		CurrentState:      currentState,
		DomainDriver:      req.DomainDriver,
		Config:            s.stabilityConfig,
	}

	// Evaluate through stability engine
	decision := s.stabilityEngine.Evaluate(input)

	// Persist raw assignment record
	assignmentRecord := models.ClusterAssignmentRecord{
		ID:                uuid.New(),
		PatientID:         patientID,
		RunID:             req.RunID,
		RunDate:           now,
		RawClusterLabel:   req.RawClusterLabel,
		MembershipProb:    req.MembershipProb,
		SeparabilityRatio: req.SeparabilityRatio,
		IsNoise:           req.IsNoise,
		StableCluster:     decision.StableClusterLabel,
		WasOverridden:     decision.Decision != models.DecisionAccept,
		OverrideReason:    decision.Reason,
	}
	if err := s.db.DB.Create(&assignmentRecord).Error; err != nil {
		s.logger.Error("failed to persist cluster assignment", zap.Error(err))
	}

	// Update profile with the STABLE cluster (not the raw one)
	confidence := decision.Confidence
	updates := map[string]interface{}{
		"phenotype_cluster":        decision.StableClusterLabel,
		"phenotype_confidence":     &confidence,
		"phenotype_cluster_origin": "STABILITY_ENGINE",
	}
	s.db.DB.Model(&models.PatientProfile{}).Where("patient_id = ?", patientID).Updates(updates)

	// If stable cluster changed, persist a transition record
	if decision.Decision == models.DecisionAccept && decision.TransitionType != "" && profile.PhenotypeCluster != "" && profile.PhenotypeCluster != decision.StableClusterLabel {
		transitionRecord := models.ClusterTransitionRecord{
			ID:                   uuid.New(),
			PatientID:            patientID,
			TransitionDate:       now,
			PreviousCluster:      profile.PhenotypeCluster,
			NewCluster:           decision.StableClusterLabel,
			TransitionType:       decision.TransitionType,
			Classification:       services.ClassifyTransition(profile.PhenotypeCluster, decision.StableClusterLabel, input.OverrideEvents, nil, false, false),
			ConfidenceInNew:      decision.Confidence,
			TriggerEvent:         decision.TriggerEvent,
			DominantDomainDriver: decision.DomainDriver,
		}
		if err := s.db.DB.Create(&transitionRecord).Error; err != nil {
			s.logger.Error("failed to persist cluster transition", zap.Error(err))
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"decision":       decision.Decision,
		"stable_cluster": decision.StableClusterLabel,
		"raw_cluster":    decision.RawClusterLabel,
		"reason":         decision.Reason,
	})
}
