package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/models"
)

func (s *Server) addMedication(c *gin.Context) {
	patientID := c.Param("id")
	var req models.AddMedicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	med, err := s.medicationService.Add(patientID, req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": med})
}

func (s *Server) updateMedication(c *gin.Context) {
	patientID := c.Param("id")
	medID := c.Param("med_id")

	var req models.UpdateMedicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.medicationService.Update(patientID, medID, req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}

func (s *Server) getMedications(c *gin.Context) {
	patientID := c.Param("id")

	meds, err := s.medicationService.GetActive(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true, "data": meds})
}
