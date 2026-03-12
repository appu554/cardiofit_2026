package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/models"
)

func (s *Server) batchWriteModifiers(c *gin.Context) {
	var modifiers []models.ContextModifier
	if err := c.ShouldBindJSON(&modifiers); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.pipelineService.BatchWriteModifiers(modifiers)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}

func (s *Server) batchWriteADRProfiles(c *gin.Context) {
	var profiles []models.AdverseReactionProfile
	if err := c.ShouldBindJSON(&profiles); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := s.pipelineService.BatchWriteADRProfiles(profiles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": result})
}
