package api

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"kb-patient-profile/internal/models"
)

func (s *Server) healthHandler(c *gin.Context) {
	dbErr := s.db.HealthCheck()
	cacheErr := s.cache.HealthCheck()

	status := http.StatusOK
	dbOK := dbErr == nil
	cacheOK := cacheErr == nil
	if !dbOK {
		status = http.StatusServiceUnavailable
	}

	c.JSON(status, gin.H{
		"status":   statusString(dbOK && cacheOK),
		"service":  "kb-20-patient-profile",
		"version":  "1.0.0",
		"database": dbOK,
		"cache":    cacheOK,
	})
}

func (s *Server) readinessHandler(c *gin.Context) {
	if err := s.db.HealthCheck(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

func statusString(ok bool) string {
	if ok {
		return "healthy"
	}
	return "degraded"
}

func (s *Server) createPatient(c *gin.Context) {
	var profile models.PatientProfile
	if err := c.ShouldBindJSON(&profile); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.patientService.Create(&profile); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"success": true, "data": profile})
}

func (s *Server) getProfile(c *gin.Context) {
	patientID := c.Param("id")
	response, err := s.patientService.GetFullProfile(patientID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": response})
}

func (s *Server) updatePatient(c *gin.Context) {
	patientID := c.Param("id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.patientService.Update(patientID, updates); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"success": true})
}
