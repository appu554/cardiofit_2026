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
	var req models.CombinedProjectionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, 400, "invalid request", "INVALID_REQUEST", nil)
		return
	}

	result := s.projectionEngine.ProjectCombined(req)
	sendSuccess(c, result, nil)
}
