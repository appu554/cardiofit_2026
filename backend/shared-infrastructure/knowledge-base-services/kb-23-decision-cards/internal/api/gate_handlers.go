package api

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MCUGateResumeRequest is the JSON body for clinician gate-resume requests.
type MCUGateResumeRequest struct {
	ClinicianID string `json:"clinician_id" binding:"required"`
	Reason      string `json:"reason"`
}

// handleMCUGateResume handles POST /api/v1/cards/:id/mcu-gate-resume
// A clinician resumes a PAUSE or HALT gate, transitioning it to MODIFY.
func (s *Server) handleMCUGateResume(c *gin.Context) {
	// 1. Parse and validate the card ID path parameter.
	rawID := c.Param("id")
	cardID, err := uuid.Parse(rawID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_card_id",
			"message": "card ID must be a valid UUID",
		})
		return
	}

	// 2. Parse and validate the request body.
	var req MCUGateResumeRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid_payload",
			"message": err.Error(),
		})
		return
	}

	s.log.Info("MCU gate resume requested",
		zap.String("card_id", cardID.String()),
		zap.String("clinician_id", req.ClinicianID),
		zap.String("reason", req.Reason),
	)

	// 3. Delegate to the CardLifecycle service.
	if err := s.cardLifecycle.ResumeGate(c.Request.Context(), cardID, req.ClinicianID); err != nil {
		s.log.Error("gate resume failed",
			zap.String("card_id", cardID.String()),
			zap.Error(err),
		)

		errMsg := err.Error()
		switch {
		case strings.Contains(errMsg, "not found"):
			c.JSON(http.StatusNotFound, gin.H{
				"error":   "card_not_found",
				"message": errMsg,
			})
		case strings.Contains(errMsg, "not ACTIVE"),
			strings.Contains(errMsg, "only PAUSE or HALT"),
			strings.Contains(errMsg, "invalid state"):
			c.JSON(http.StatusConflict, gin.H{
				"error":   "invalid_state",
				"message": errMsg,
			})
		default:
			c.JSON(http.StatusInternalServerError, gin.H{
				"error":   "internal_error",
				"message": errMsg,
			})
		}
		return
	}

	// 4. Return success.
	c.JSON(http.StatusOK, gin.H{
		"status":  "resumed",
		"card_id": cardID.String(),
	})
}
