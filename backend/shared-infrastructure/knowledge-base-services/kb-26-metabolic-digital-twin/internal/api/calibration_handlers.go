package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func (s *Server) calibrate(c *gin.Context) {
	var req struct {
		PatientID        string  `json:"patient_id" binding:"required"`
		InterventionCode string  `json:"intervention_code" binding:"required"`
		TargetVariable   string  `json:"target_variable" binding:"required"`
		PopulationEffect float64 `json:"population_effect"`
		ObservedEffect   float64 `json:"observed_effect"`
		ObservationSD    float64 `json:"observation_sd"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request", "INVALID_REQUEST", nil)
		return
	}
	if req.ObservationSD <= 0 {
		req.ObservationSD = 2.0
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	result, err := s.calibrator.Calibrate(
		patientID, req.InterventionCode, req.TargetVariable,
		req.PopulationEffect, req.ObservedEffect, req.ObservationSD,
	)
	if err != nil {
		sendError(c, http.StatusInternalServerError, "calibration failed", "CALIBRATION_ERROR", nil)
		return
	}

	sendSuccess(c, result, nil)
}
