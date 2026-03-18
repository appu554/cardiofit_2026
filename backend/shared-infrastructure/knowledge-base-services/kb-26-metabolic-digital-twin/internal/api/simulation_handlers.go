package api

import (
	"net/http"

	"kb-26-metabolic-digital-twin/internal/models"
	"kb-26-metabolic-digital-twin/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) simulate(c *gin.Context) {
	var req struct {
		PatientID    string              `json:"patient_id" binding:"required"`
		Intervention models.Intervention `json:"intervention" binding:"required"`
		Days         int                 `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request", "INVALID_REQUEST", nil)
		return
	}
	if req.Days == 0 {
		req.Days = 90
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}
	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found", "NOT_FOUND", nil)
		return
	}

	initial := services.TwinToSimState(twin)
	projected := services.RunSimulation(initial, req.Intervention, req.Days)

	sendSuccess(c, gin.H{
		"patient_id":      req.PatientID,
		"intervention":    req.Intervention,
		"projection_days": req.Days,
		"projected":       projected,
	}, nil)
}

func (s *Server) simulateComparison(c *gin.Context) {
	var req struct {
		PatientID     string                `json:"patient_id" binding:"required"`
		Interventions []models.Intervention `json:"interventions" binding:"required"`
		Days          int                   `json:"days"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request", "INVALID_REQUEST", nil)
		return
	}
	if req.Days == 0 {
		req.Days = 90
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}
	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found", "NOT_FOUND", nil)
		return
	}

	initial := services.TwinToSimState(twin)
	comparisons := make(map[string][]models.ProjectedState)
	for _, iv := range req.Interventions {
		projected := services.RunSimulation(initial, iv, req.Days)
		comparisons[iv.Code] = projected
	}

	sendSuccess(c, gin.H{
		"patient_id":  req.PatientID,
		"comparisons": comparisons,
	}, nil)
}
