package api

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"kb-23-decision-cards/internal/models"
	"kb-23-decision-cards/internal/services"
)

// resolveCohort reads the ?cohort= query param. Behaviour:
//   - "all"           → empty string (no filter)
//   - "" + useDefault → server.cfg.DefaultCohort (empty string if unset)
//   - "" + !useDefault → empty string (no filter)
//   - anything else   → that exact cohort id
//
// The pilot endpoint sets useDefault=true so HCF instances return HCF data by
// default; the system/clinician endpoints leave it false so they stay global
// unless the caller opts in.
func (s *Server) resolveCohort(c *gin.Context, useDefault bool) string {
	raw := c.Query("cohort")
	if raw == "all" {
		return ""
	}
	if raw == "" && useDefault && s.cfg != nil {
		return s.cfg.DefaultCohort
	}
	return raw
}

// applyCohortFilter appends a WHERE cohort_id = ? clause when cohort is set.
// Returns the slice of lifecycles and a cohort-echo string for the response.
func (s *Server) loadLifecyclesByWindowAndCohort(windowDays int, cohort string, extraWhere string, extraArgs ...interface{}) []models.DetectionLifecycle {
	since := time.Now().AddDate(0, 0, -windowDays)
	q := s.db.DB.Where("detected_at > ?", since)
	if cohort != "" {
		q = q.Where("cohort_id = ?", cohort)
	}
	if extraWhere != "" {
		q = q.Where(extraWhere, extraArgs...)
	}
	var lifecycles []models.DetectionLifecycle
	q.Find(&lifecycles)
	return lifecycles
}

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
// GET /api/v1/metrics/clinician/:clinicianId?window=30&cohort=<id|all>
func (s *Server) getClinicianMetrics(c *gin.Context) {
	clinicianID := c.Param("clinicianId")
	window := parseWindow(c.Query("window"), 30)
	cohort := s.resolveCohort(c, false)

	lifecycles := s.loadLifecyclesByWindowAndCohort(window, cohort,
		"assigned_clinician_id = ?", clinicianID)

	svc := s.responseMetricsService
	if svc == nil {
		svc = services.NewResponseMetricsService(nil)
	}
	metrics := svc.ComputeClinicianMetrics(lifecycles, clinicianID, cohort, window)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": metrics})
}

// getSystemMetrics returns system-wide aggregate response metrics.
// GET /api/v1/metrics/system?window=30&cohort=<id|all>
func (s *Server) getSystemMetrics(c *gin.Context) {
	window := parseWindow(c.Query("window"), 30)
	cohort := s.resolveCohort(c, false)
	lifecycles := s.loadLifecyclesByWindowAndCohort(window, cohort, "")

	svc := s.responseMetricsService
	if svc == nil {
		svc = services.NewResponseMetricsService(nil)
	}
	metrics := svc.ComputeSystemMetrics(lifecycles, cohort, window)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": metrics})
}

// getPilotMetrics returns pilot-specific KPIs. Unlike the system endpoint,
// this one defaults the cohort filter to the server's DEFAULT_COHORT so
// an HCF-pilot deployment returns HCF-only numbers without the caller
// having to pass ?cohort=hcf_catalyst_chf every time.
// GET /api/v1/metrics/pilot?window=90&cohort=<id|all>
func (s *Server) getPilotMetrics(c *gin.Context) {
	window := parseWindow(c.Query("window"), 90)
	cohort := s.resolveCohort(c, true)
	lifecycles := s.loadLifecyclesByWindowAndCohort(window, cohort, "")

	svc := s.responseMetricsService
	if svc == nil {
		svc = services.NewResponseMetricsService(nil)
	}
	metrics := svc.ComputePilotMetrics(lifecycles, cohort, window)
	c.JSON(http.StatusOK, gin.H{"success": true, "data": metrics})
}

// resolveLifecycleRequest is the payload for POST /api/v1/tracking/resolve,
// called by KB-26 (acute event resolution) and potentially other services
// that observe outcome signals.
type resolveLifecycleRequest struct {
	PatientID          string    `json:"patient_id" binding:"required"`
	DetectionType      string    `json:"detection_type,omitempty"` // optional narrowing
	ResolvedAt         time.Time `json:"resolved_at" binding:"required"`
	OutcomeDescription string    `json:"outcome_description,omitempty"`
}

// handleResolveLifecycle attributes an observed outcome to the most recent
// actioned-but-unresolved lifecycle for the patient, closing T4.
// POST /api/v1/tracking/resolve
//
// Sprint 1 attribution rule: "most recent actioned detection for this patient
// (optionally matching detection_type) wins." A proper outcome attribution
// engine replaces this in Sprint 2.
func (s *Server) handleResolveLifecycle(c *gin.Context) {
	var req resolveLifecycleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if s.lifecycleTracker == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "lifecycle tracker not initialized"})
		return
	}
	lc, err := s.lifecycleTracker.FindMostRecentActionedByPatient(req.PatientID, req.DetectionType)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "no actioned-unresolved lifecycle matched",
		})
		return
	}
	s.lifecycleTracker.RecordT4(lc, req.OutcomeDescription, req.ResolvedAt)
	c.JSON(http.StatusOK, gin.H{
		"success":      true,
		"lifecycle_id": lc.ID,
		"patient_id":   lc.PatientID,
		"resolved_at":  req.ResolvedAt,
	})
}

func parseWindow(raw string, fallback int) int {
	if raw == "" {
		return fallback
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v <= 0 {
		return fallback
	}
	return v
}
