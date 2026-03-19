package api

import (
	"fmt"
	"net/http"

	"kb-21-behavioral-intelligence/internal/services"

	"github.com/gin-gonic/gin"
)

// getMRIMessage generates a patient-facing MRI message based on score change.
// GET /api/v1/patient/:patient_id/mri-message?current_score=X&previous_score=Y&category=Z&top_driver=W
func (s *Server) getMRIMessage(c *gin.Context) {
	currentScore := parseFloatParam(c.Query("current_score"), 0)
	previousScore := parseFloatParam(c.Query("previous_score"), 0)
	category := c.DefaultQuery("category", "")
	topDriver := c.DefaultQuery("top_driver", "")

	if category == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "category is required"})
		return
	}

	msg := services.GenerateMRIMessage(currentScore, previousScore, category, topDriver)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": msg})
}

func parseFloatParam(s string, def float64) float64 {
	if s == "" {
		return def
	}
	var v float64
	fmt.Sscanf(s, "%f", &v)
	return v
}
