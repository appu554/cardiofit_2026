package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/models"
)

func (s *Server) addLab(c *gin.Context) {
	patientID := c.Param("id")
	var req models.AddLabRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	entry, err := s.labService.AddLab(patientID, req)
	if err != nil {
		// Lab validation rejection returns 422
		c.JSON(http.StatusUnprocessableEntity, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": entry})
}

func (s *Server) getLabs(c *gin.Context) {
	patientID := c.Param("id")
	labType := c.Query("lab_type")

	labs, err := s.labService.GetLabs(patientID, labType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": labs})
}

func (s *Server) getEGFRHistory(c *gin.Context) {
	patientID := c.Param("id")

	trajectory, err := s.labService.GetEGFRTrajectory(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": trajectory})
}
