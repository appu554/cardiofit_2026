// Package api provides the HTTP API server
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-17-population-registry/internal/models"
)

// Health handlers

func (s *Server) healthHandler(c *gin.Context) {
	checks := map[string]string{
		"database": "healthy",
	}

	if err := s.db.Health(); err != nil {
		checks["database"] = "unhealthy"
	}

	status := "healthy"
	for _, v := range checks {
		if v != "healthy" {
			status = "degraded"
			break
		}
	}

	c.JSON(http.StatusOK, models.HealthResponse{
		Status:    status,
		Service:   "kb-17-population-registry",
		Version:   s.config.Server.Version,
		Uptime:    time.Since(s.startTime).String(),
		Checks:    checks,
		Timestamp: time.Now().UTC(),
	})
}

func (s *Server) readyHandler(c *gin.Context) {
	if err := s.db.Health(); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"ready": false, "error": "database unhealthy"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ready": true})
}

// Registry handlers

func (s *Server) listRegistriesHandler(c *gin.Context) {
	activeOnly := c.Query("active") == "true"

	registries, err := s.repo.ListRegistries(activeOnly)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, &models.RegistryListResponse{
		Success: true,
		Data:    registries,
		Total:   int64(len(registries)),
	})
}

func (s *Server) getRegistryHandler(c *gin.Context) {
	code := models.RegistryCode(c.Param("code"))

	registry, err := s.repo.GetRegistry(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	if registry == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse("Registry not found"))
		return
	}

	c.JSON(http.StatusOK, &models.RegistryResponse{
		Success: true,
		Data:    registry,
	})
}

func (s *Server) createRegistryHandler(c *gin.Context) {
	var req models.CreateRegistryRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	registry := &models.Registry{
		Code:               req.Code,
		Name:               req.Name,
		Description:        req.Description,
		Category:           req.Category,
		AutoEnroll:         req.AutoEnroll,
		Active:             true,
		InclusionCriteria:  req.InclusionCriteria,
		ExclusionCriteria:  req.ExclusionCriteria,
		RiskStratification: req.RiskStratification,
		CareGapMeasures:    req.CareGapMeasures,
	}

	if err := s.repo.CreateRegistry(registry); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, &models.RegistryResponse{
		Success: true,
		Data:    registry,
	})
}

func (s *Server) getRegistryPatientsHandler(c *gin.Context) {
	code := models.RegistryCode(c.Param("code"))

	var query struct {
		Limit  int `form:"limit,default=50"`
		Offset int `form:"offset,default=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	patients, total, err := s.repo.GetRegistryPatients(code, query.Limit, query.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.NewPaginatedResponse(patients, total, query.Limit, query.Offset))
}

// Enrollment handlers

func (s *Server) listEnrollmentsHandler(c *gin.Context) {
	var query models.EnrollmentQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	enrollments, total, err := s.repo.ListEnrollments(&query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, &models.EnrollmentListResponse{
		Success: true,
		Data:    enrollments,
		Total:   total,
	})
}

func (s *Server) createEnrollmentHandler(c *gin.Context) {
	var req models.EnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	// Check if already enrolled
	existing, err := s.repo.GetEnrollmentByPatientRegistry(req.PatientID, req.RegistryCode)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	if existing != nil && existing.Status.IsActive() {
		c.JSON(http.StatusConflict, models.NewErrorResponse("Patient already enrolled in this registry"))
		return
	}

	enrollment := &models.RegistryPatient{
		RegistryCode:     req.RegistryCode,
		PatientID:        req.PatientID,
		Status:           models.EnrollmentStatusActive,
		EnrollmentSource: req.EnrollmentSource,
		SourceEventID:    req.SourceEventID,
		RiskTier:         req.RiskTier,
		Metrics:          req.Metrics,
		Notes:            req.Notes,
		EnrolledBy:       req.EnrolledBy,
		EnrolledAt:       time.Now().UTC(),
		Metadata:         req.Metadata,
	}

	if enrollment.RiskTier == "" {
		enrollment.RiskTier = models.RiskTierModerate
	}

	if err := s.repo.CreateEnrollment(enrollment); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	// Produce enrollment event
	if s.producer != nil {
		s.producer.ProduceEnrollmentEvent(c.Request.Context(), enrollment)
	}

	c.JSON(http.StatusCreated, &models.EnrollmentResponse{
		Success: true,
		Data:    enrollment,
	})
}

func (s *Server) getEnrollmentHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("Invalid enrollment ID"))
		return
	}

	enrollment, err := s.repo.GetEnrollment(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	if enrollment == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse("Enrollment not found"))
		return
	}

	c.JSON(http.StatusOK, &models.EnrollmentResponse{
		Success: true,
		Data:    enrollment,
	})
}

func (s *Server) updateEnrollmentHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("Invalid enrollment ID"))
		return
	}

	enrollment, err := s.repo.GetEnrollment(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	if enrollment == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse("Enrollment not found"))
		return
	}

	var req models.UpdateEnrollmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	// Handle risk tier change
	if req.RiskTier != nil && *req.RiskTier != enrollment.RiskTier {
		oldTier := enrollment.RiskTier
		if err := s.repo.UpdateEnrollmentRiskTier(id, oldTier, *req.RiskTier, "api"); err != nil {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
			return
		}
		enrollment.RiskTier = *req.RiskTier

		// Produce risk change event
		if s.producer != nil {
			s.producer.ProduceRiskChangedEvent(c.Request.Context(), enrollment, oldTier, *req.RiskTier)
		}
	}

	// Handle status change
	if req.Status != nil && *req.Status != enrollment.Status {
		oldStatus := enrollment.Status
		if err := s.repo.UpdateEnrollmentStatus(id, oldStatus, *req.Status, "", "api"); err != nil {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
			return
		}
		enrollment.Status = *req.Status
	}

	// Handle metrics update
	if req.Metrics != nil {
		if err := s.repo.UpdateEnrollmentMetrics(id, req.Metrics); err != nil {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
			return
		}
		enrollment.Metrics = req.Metrics
	}

	// Handle care gaps update
	if req.CareGaps != nil {
		if err := s.repo.UpdateEnrollmentCareGaps(id, req.CareGaps); err != nil {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
			return
		}
		enrollment.CareGaps = req.CareGaps
	}

	c.JSON(http.StatusOK, &models.EnrollmentResponse{
		Success: true,
		Data:    enrollment,
	})
}

func (s *Server) deleteEnrollmentHandler(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse("Invalid enrollment ID"))
		return
	}

	var req models.DisenrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	enrollment, err := s.repo.GetEnrollment(id)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	if enrollment == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse("Enrollment not found"))
		return
	}

	if err := s.repo.DeleteEnrollment(id, req.Reason, req.DisenrolledBy); err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	// Produce disenrollment event
	if s.producer != nil {
		s.producer.ProduceDisenrollmentEvent(c.Request.Context(), enrollment, req.Reason)
	}

	c.JSON(http.StatusOK, models.NewMessageResponse("Patient disenrolled successfully"))
}

func (s *Server) bulkEnrollHandler(c *gin.Context) {
	var req models.BulkEnrollRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	enrollments := make([]models.RegistryPatient, len(req.PatientIDs))
	for i, patientID := range req.PatientIDs {
		enrollments[i] = models.RegistryPatient{
			RegistryCode:     req.RegistryCode,
			PatientID:        patientID,
			Status:           models.EnrollmentStatusActive,
			EnrollmentSource: req.EnrollmentSource,
			RiskTier:         req.RiskTier,
			EnrolledBy:       req.EnrolledBy,
			EnrolledAt:       time.Now().UTC(),
		}
		if enrollments[i].RiskTier == "" {
			enrollments[i].RiskTier = models.RiskTierModerate
		}
	}

	result, err := s.repo.BulkEnroll(enrollments)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, &models.BulkEnrollmentResponse{
		Success: true,
		Data:    result,
	})
}

// Patient-centric handlers

func (s *Server) getPatientRegistriesHandler(c *gin.Context) {
	patientID := c.Param("id")

	enrollments, err := s.repo.GetPatientRegistries(patientID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, &models.PatientRegistriesResponse{
		Success:     true,
		PatientID:   patientID,
		Enrollments: enrollments,
		Total:       len(enrollments),
	})
}

func (s *Server) getPatientEnrollmentHandler(c *gin.Context) {
	patientID := c.Param("id")
	code := models.RegistryCode(c.Param("code"))

	enrollment, err := s.repo.GetEnrollmentByPatientRegistry(patientID, code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	if enrollment == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse("Enrollment not found"))
		return
	}

	c.JSON(http.StatusOK, &models.EnrollmentResponse{
		Success: true,
		Data:    enrollment,
	})
}

// Evaluation handler

func (s *Server) evaluateHandler(c *gin.Context) {
	var req models.EvaluateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	// Get patient data if not provided
	patientData := req.PatientData
	if patientData == nil {
		patientData = &models.PatientClinicalData{PatientID: req.PatientID}

		// Try to fetch from KB-2
		if s.kb2Client != nil {
			data, err := s.kb2Client.GetPatientContext(c.Request.Context(), req.PatientID)
			if err == nil && data != nil {
				patientData = data
			}
		}
	}

	var results []models.CriteriaEvaluationResult
	var err error

	if req.RegistryCode != "" {
		// Evaluate specific registry
		registry, err := s.repo.GetRegistry(req.RegistryCode)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
			return
		}
		if registry == nil {
			c.JSON(http.StatusNotFound, models.NewErrorResponse("Registry not found"))
			return
		}

		result, err := s.engine.Evaluate(patientData, registry)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
			return
		}
		results = []models.CriteriaEvaluationResult{*result}
	} else {
		// Evaluate all registries
		results, err = s.engine.EvaluateAll(patientData)
		if err != nil {
			c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
			return
		}
	}

	c.JSON(http.StatusOK, &models.EvaluateResponse{
		Success: true,
		Data:    results,
	})
}

// Analytics handlers

func (s *Server) getAllStatsHandler(c *gin.Context) {
	registries, err := s.repo.ListRegistries(true)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	stats := make([]models.RegistryStats, 0, len(registries))
	summary := &models.StatsSummary{
		TotalRegistries: len(registries),
	}

	for _, reg := range registries {
		regStats, err := s.repo.GetRegistryStats(reg.Code)
		if err != nil {
			continue
		}
		stats = append(stats, *regStats)
		summary.TotalEnrollments += regStats.TotalEnrolled
		summary.ActiveEnrollments += regStats.ActiveCount
		summary.HighRiskPatients += regStats.HighRiskCount + regStats.CriticalCount
		summary.PatientsWithGaps += regStats.CareGapCount
	}

	c.JSON(http.StatusOK, &models.AllStatsResponse{
		Success: true,
		Data:    stats,
		Summary: summary,
	})
}

func (s *Server) getRegistryStatsHandler(c *gin.Context) {
	code := models.RegistryCode(c.Param("code"))

	stats, err := s.repo.GetRegistryStats(code)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, &models.StatsResponse{
		Success: true,
		Data:    stats,
	})
}

func (s *Server) getHighRiskPatientsHandler(c *gin.Context) {
	var query struct {
		Limit  int `form:"limit,default=50"`
		Offset int `form:"offset,default=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	patients, total, err := s.repo.GetHighRiskPatients(query.Limit, query.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	// Convert to summaries
	summaries := make([]models.HighRiskPatientSummary, len(patients))
	byTier := make(map[models.RiskTier]int64)

	for i, p := range patients {
		summaries[i] = models.HighRiskPatientSummary{
			PatientID:       p.PatientID,
			RegistryCode:    p.RegistryCode,
			RiskTier:        p.RiskTier,
			CareGapCount:    len(p.CareGaps),
			EnrolledAt:      p.EnrolledAt,
			LastEvaluatedAt: p.LastEvaluatedAt,
		}
		byTier[p.RiskTier]++
	}

	c.JSON(http.StatusOK, &models.HighRiskResponse{
		Success: true,
		Data:    summaries,
		Total:   total,
		ByTier:  byTier,
	})
}

func (s *Server) getCareGapPatientsHandler(c *gin.Context) {
	var query struct {
		Limit  int `form:"limit,default=50"`
		Offset int `form:"offset,default=0"`
	}
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	patients, total, err := s.repo.GetPatientsWithCareGaps(query.Limit, query.Offset)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	// Convert to summaries
	summaries := make([]models.CareGapSummary, len(patients))
	byRegistry := make(map[models.RegistryCode]int64)

	for i, p := range patients {
		summaries[i] = models.CareGapSummary{
			PatientID:    p.PatientID,
			RegistryCode: p.RegistryCode,
			CareGaps:     p.CareGaps,
			RiskTier:     p.RiskTier,
			EnrolledAt:   p.EnrolledAt,
		}
		byRegistry[p.RegistryCode]++
	}

	c.JSON(http.StatusOK, &models.CareGapResponse{
		Success:    true,
		Data:       summaries,
		Total:      total,
		ByRegistry: byRegistry,
	})
}

// Event handler

func (s *Server) processEventHandler(c *gin.Context) {
	var req models.ProcessEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(err.Error()))
		return
	}

	if s.consumer == nil {
		c.JSON(http.StatusServiceUnavailable, models.NewErrorResponse("Event processing not available"))
		return
	}

	response, err := s.consumer.ProcessManualEvent(c.Request.Context(), &req)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(err.Error()))
		return
	}

	c.JSON(http.StatusOK, response)
}
