package api

import (
	"encoding/json"
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
	// Bind raw JSON for validation before model binding
	var rawPayload []map[string]interface{}
	if err := c.ShouldBindJSON(&rawPayload); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// L3 intake validation
	validationErrors := validateL3Payload(rawPayload)
	if len(validationErrors) > 0 {
		c.JSON(http.StatusUnprocessableEntity, gin.H{
			"error":             "L3 validation failed",
			"validation_errors": validationErrors,
		})
		return
	}

	// Re-marshal and bind to typed models for upsert
	jsonBytes, _ := json.Marshal(rawPayload)
	var profiles []models.AdverseReactionProfile
	if err := json.Unmarshal(jsonBytes, &profiles); err != nil {
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
