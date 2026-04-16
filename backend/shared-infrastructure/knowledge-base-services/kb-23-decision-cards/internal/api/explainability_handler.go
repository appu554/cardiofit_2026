package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-23-decision-cards/internal/services"
)

// handleGetExplainability returns the full evidence trail for a
// decision card. Phase 10 Gap 10. Route: GET /api/v1/cards/:id/explainability
func (s *Server) handleGetExplainability(c *gin.Context) {
	cardID := c.Param("id")

	svc := services.NewExplainabilityService(s.db.DB, s.log)
	trail, err := svc.BuildTrail(cardID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "failed to build explainability trail: " + err.Error(),
		})
		return
	}
	if trail == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": "card not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data":    trail,
	})
}
