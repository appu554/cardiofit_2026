package api

import (
	"net/http"

	"kb-26-metabolic-digital-twin/internal/services"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// getPREVENT returns the latest PREVENT 10-year CVD risk score for a patient.
// If no persisted score exists, it computes one from the current twin state.
// GET /api/v1/kb26/prevent/:patientId
func (s *Server) getPREVENT(c *gin.Context) {
	patientID, err := uuid.Parse(c.Param("patientId"))
	if err != nil {
		sendError(c, http.StatusBadRequest, "invalid patient ID", "INVALID_PATIENT_ID", nil)
		return
	}

	// Try to return the latest persisted score first.
	if s.preventScorer != nil {
		existing, err := s.preventScorer.GetLatest(patientID)
		if err == nil && existing != nil {
			sendSuccess(c, existing, map[string]interface{}{
				"patient_id": patientID.String(),
				"source":     "persisted",
			})
			return
		}
	}

	// No persisted score — compute from current twin state.
	if s.preventScorer == nil {
		sendError(c, http.StatusServiceUnavailable, "PREVENT scorer not available", "SERVICE_UNAVAILABLE", nil)
		return
	}

	twin, err := s.twinUpdater.GetLatest(patientID)
	if err != nil {
		sendError(c, http.StatusNotFound, "twin state not found", "TWIN_NOT_FOUND", nil)
		return
	}

	input := services.TwinToPREVENTInput(twin)
	result := s.preventScorer.ComputePREVENT(input)

	// Persist the computed score.
	persisted, err := s.preventScorer.PersistScore(patientID, input, result, &twin.ID)
	if err != nil {
		s.logger.Error("failed to persist PREVENT score", zap.Error(err))
	}

	resp := gin.H{
		"ten_year_risk": result.TenYearRisk,
		"risk_percent":  result.RiskPercent,
		"category":      result.Category,
	}
	if persisted != nil {
		resp["id"] = persisted.ID.String()
		resp["computed_at"] = persisted.ComputedAt
	}

	sendSuccess(c, resp, map[string]interface{}{
		"patient_id": patientID.String(),
		"source":     "computed",
	})
}
