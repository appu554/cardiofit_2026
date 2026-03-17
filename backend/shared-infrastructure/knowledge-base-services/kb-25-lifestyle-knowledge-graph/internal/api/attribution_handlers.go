package api

import (
	"net/http"

	"kb-25-lifestyle-knowledge-graph/internal/services"

	"github.com/gin-gonic/gin"
)

func (s *Server) attributeOutcome(c *gin.Context) {
	var req struct {
		PatientID  string  `json:"patient_id" binding:"required"`
		TargetVar  string  `json:"target_variable" binding:"required"`
		TotalDelta float64 `json:"total_delta" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		sendError(c, http.StatusBadRequest, "invalid request", "INVALID_REQUEST", nil)
		return
	}

	result := services.AttributeOutcome(req.PatientID, req.TargetVar, req.TotalDelta)
	sendSuccess(c, result, nil)
}
