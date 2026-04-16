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

	// Build stability input with default config
	input := services.StabilityInput{
		PatientID:         patientID,
		RawClusterLabel:   req.RawClusterLabel,
		MembershipProb:    req.MembershipProb,
		SeparabilityRatio: req.SeparabilityRatio,
		IsNoise:           req.IsNoise,
		RunDate:           now,
		CurrentState:      currentState,
		DomainDriver:      req.DomainDriver,
		Config: services.StabilityConfig{
			DwellMinWeeks:          4,
			DwellExtendedWeeks:     8,
			FlapLookbackDays:       90,
			FlapMinOscillations:    2,
			HighMembershipProb:     0.7,
			ModerateMembershipProb: 0.4,
			CGMStartGraceWeeks:     2,
			CGMStopGraceWeeks:      4,
			ConservatismRank: map[string]int{
				"STABLE_CONTROLLED":     1,
				"STABLE_MEDICATED":      2,
				"PROGRESSIVE_GLYCAEMIC": 3,
				"CARDIORENAL_COMPLEX":   4,
				"HIGH_RISK_UNSTABLE":    5,
				"NOISE":                 6,
			},
		},
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
		"phenotype_cluster":    decision.StableClusterLabel,
		"phenotype_confidence": &confidence,
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
