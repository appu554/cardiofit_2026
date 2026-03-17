package api

import (
	"net/http"

	"kb-25-lifestyle-knowledge-graph/internal/models"
	"kb-25-lifestyle-knowledge-graph/internal/services"

	"github.com/gin-gonic/gin"
)

func (s *Server) compareInterventions(c *gin.Context) {
	var req models.ComparisonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request", "INVALID_REQUEST", nil)
		return
	}
	if req.TimeHorizon == 0 {
		req.TimeHorizon = 90
	}

	patient, _ := s.kb20Client.GetPatientSnapshot(req.PatientID)
	hba1c := 0.0
	sbp := 0.0
	if patient != nil {
		hba1c = patient.HbA1c
		sbp = patient.SBP
	}

	recommendation := services.ApplyDecisionRule(hba1c, sbp)

	var compared []models.ComparedOption
	for _, opt := range req.Options {
		compared = append(compared, models.ComparedOption{
			Option:        opt,
			EvidenceGrade: "B",
			SafetyScore:   0.9,
		})
	}
	compared = services.RankOptions(compared)

	result := models.ComparisonResult{
		PatientID:      req.PatientID,
		TargetVar:      req.TargetVar,
		Options:        compared,
		Recommendation: recommendation,
		Rationale:      "Based on current HbA1c and SBP thresholds",
	}

	sendSuccess(c, result, nil)
}

func (s *Server) projectCombined(c *gin.Context) {
	var req struct {
		PatientID     string `json:"patient_id" binding:"required"`
		LifestyleCode string `json:"lifestyle_code" binding:"required"`
		MedChange     string `json:"med_change" binding:"required"`
		Days          int    `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request", "INVALID_REQUEST", nil)
		return
	}
	if req.Days == 0 {
		req.Days = 90
	}

	sendSuccess(c, gin.H{
		"patient_id":      req.PatientID,
		"lifestyle_code":  req.LifestyleCode,
		"med_change":      req.MedChange,
		"projection_days": req.Days,
		"status":          "projection_engine_pending",
	}, nil)
}
