package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-23-decision-cards/internal/models"
	"kb-23-decision-cards/internal/services"
)

// getDetectionLifecycle returns the full lifecycle for a single detection.
// GET /api/v1/tracking/detection/:id
func (s *Server) getDetectionLifecycle(c *gin.Context) {
	id := c.Param("id")
	var lc models.DetectionLifecycle
	if err := s.db.DB.Where("id = ?", id).First(&lc).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "lifecycle not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"success": true, "data": lc})
}

// getPatientLifecycles returns recent lifecycles for a patient.
// GET /api/v1/tracking/patient/:patientId
func (s *Server) getPatientLifecycles(c *gin.Context) {
	patientID := c.Param("patientId")
	var lifecycles []models.DetectionLifecycle
	s.db.DB.Where("patient_id = ?", patientID).
		Order("detected_at DESC").Limit(50).Find(&lifecycles)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": lifecycles})
}

// getClinicianMetrics returns computed response metrics for a clinician.
// GET /api/v1/metrics/clinician/:clinicianId?window=30
func (s *Server) getClinicianMetrics(c *gin.Context) {
	clinicianID := c.Param("clinicianId")
	window := 30
	if w := c.Query("window"); w != "" {
		if parsed, err := strconv.Atoi(w); err == nil && parsed > 0 {
			window = parsed
		}
	}

	var lifecycles []models.DetectionLifecycle
	since := time.Now().AddDate(0, 0, -window)
	s.db.DB.Where("assigned_clinician_id = ? AND detected_at > ?", clinicianID, since).
		Find(&lifecycles)

	svc := services.NewResponseMetricsService()
	metrics := svc.ComputeClinicianMetrics(lifecycles, clinicianID, window)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": metrics})
}

// getSystemMetrics returns system-wide aggregate response metrics.
// GET /api/v1/metrics/system?window=30
func (s *Server) getSystemMetrics(c *gin.Context) {
	window := 30
	if w := c.Query("window"); w != "" {
		if parsed, err := strconv.Atoi(w); err == nil && parsed > 0 {
			window = parsed
		}
	}

	var lifecycles []models.DetectionLifecycle
	since := time.Now().AddDate(0, 0, -window)
	s.db.DB.Where("detected_at > ?", since).Find(&lifecycles)

	svc := services.NewResponseMetricsService()
	metrics := svc.ComputeSystemMetrics(lifecycles, window)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": metrics})
}

// getPilotMetrics returns HCF CHF pilot-specific KPIs.
// GET /api/v1/metrics/pilot?window=90
func (s *Server) getPilotMetrics(c *gin.Context) {
	window := 90
	if w := c.Query("window"); w != "" {
		if parsed, err := strconv.Atoi(w); err == nil && parsed > 0 {
			window = parsed
		}
	}

	var lifecycles []models.DetectionLifecycle
	since := time.Now().AddDate(0, 0, -window)
	s.db.DB.Where("detected_at > ?", since).Find(&lifecycles)

	svc := services.NewResponseMetricsService()
	metrics := svc.ComputePilotMetrics(lifecycles)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": metrics})
}
