// Package api provides the HTTP API server for KB-12
package api

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

// setupRoutes configures all API routes
func (s *Server) setupRoutes() {
	// Health check endpoints
	s.router.GET("/health", s.healthCheck)
	s.router.GET("/health/live", s.livenessCheck)
	s.router.GET("/health/ready", s.readinessCheck)

	// Metrics endpoint
	s.router.GET("/metrics", s.metricsHandler)

	// API v1 routes
	v1 := s.router.Group("/api/v1")
	{
		// Order Set Template endpoints
		templates := v1.Group("/templates")
		{
			templates.GET("", s.listTemplates)
			templates.GET("/:id", s.getTemplate)
			templates.GET("/search", s.searchTemplates)
			templates.GET("/category/:category", s.getTemplatesByCategory)
		}

		// Order Set Activation endpoints
		activate := v1.Group("/activate")
		{
			activate.POST("", s.activateOrderSet)
		}

		// Order Set Instance endpoints
		instances := v1.Group("/instances")
		{
			instances.GET("", s.listInstances)
			instances.GET("/:id", s.getInstance)
			instances.PUT("/:id/status", s.updateInstanceStatus)
			instances.GET("/:id/constraints", s.getInstanceConstraints)
			instances.PUT("/:id/order/:order_id", s.updateOrderStatus)
		}

		// Care Plan Template endpoints
		careplans := v1.Group("/careplans")
		{
			careplans.GET("", s.listCarePlanTemplates)
			careplans.GET("/:id", s.getCarePlanTemplate)
			careplans.POST("", s.activateCarePlan)
		}

		// Care Plan Instance endpoints
		careplanInstances := v1.Group("/careplan-instances")
		{
			careplanInstances.GET("", s.listCarePlanInstances)
			careplanInstances.GET("/:id", s.getCarePlanInstance)
			careplanInstances.PUT("/:id/status", s.updateCarePlanStatus)
			careplanInstances.PUT("/:id/progress", s.updateCarePlanProgress)
			careplanInstances.GET("/:id/goals", s.getCarePlanGoals)
			careplanInstances.PUT("/:id/goals/:goal_id", s.updateGoalProgress)
			careplanInstances.GET("/:id/activities", s.getCarePlanActivities)
			careplanInstances.PUT("/:id/activities/:activity_id", s.updateActivityStatus)
		}

		// Temporal Constraint endpoints
		constraints := v1.Group("/constraints")
		{
			constraints.GET("", s.listConstraints)
			constraints.POST("/evaluate", s.evaluateConstraints)
			constraints.GET("/overdue", s.getOverdueConstraints)
		}

		// FHIR Resource Generation endpoints
		fhir := v1.Group("/fhir")
		{
			fhir.GET("/bundle/:instance_id", s.getFHIRBundle)
			fhir.GET("/plandefinition/:template_id", s.getFHIRPlanDefinition)
			fhir.GET("/careplan/:instance_id", s.getFHIRCarePlan)
			fhir.GET("/medicationrequest/:order_id", s.getFHIRMedicationRequest)
			fhir.GET("/servicerequest/:order_id", s.getFHIRServiceRequest)
		}

		// CPOE Integration endpoints
		cpoe := v1.Group("/cpoe")
		{
			cpoe.POST("/submit", s.submitOrders)
			cpoe.POST("/drafts", s.createDraftSession)
			cpoe.GET("/drafts/:id", s.getDraftSession)
			cpoe.PUT("/drafts/:id", s.updateDraftSession)
			cpoe.POST("/drafts/:id/submit", s.submitDraftSession)
			cpoe.DELETE("/drafts/:id", s.cancelDraftSession)
			cpoe.POST("/safety-check", s.performSafetyCheck)
		}

		// Patient-specific endpoints
		patient := v1.Group("/patient/:patient_id")
		{
			patient.GET("/ordersets", s.getPatientOrderSets)
			patient.GET("/careplans", s.getPatientCarePlans)
			patient.GET("/orders", s.getPatientOrders)
			patient.GET("/constraints", s.getPatientConstraints)
		}

		// Workflow endpoints
		workflow := v1.Group("/workflow")
		{
			workflow.GET("/tasks", s.listTasks)
			workflow.GET("/tasks/:id", s.getTask)
			workflow.PUT("/tasks/:id/status", s.updateTaskStatus)
			workflow.GET("/overdue", s.getOverdueTasks)
			workflow.GET("/deadlines", s.getUpcomingDeadlines)
		}
	}

	// CDS Hooks endpoints (separate path per spec)
	cds := s.router.Group("/cds-services")
	{
		cds.GET("", s.getCDSServices)
		cds.POST("/order-select", s.orderSelectHook)
		cds.POST("/order-sign", s.orderSignHook)
		cds.POST("/patient-view", s.patientViewHook)
		cds.POST("/encounter-start", s.encounterStartHook)
		cds.POST("/encounter-discharge", s.encounterDischargeHook)
	}
}

// healthCheck handles GET /health
func (s *Server) healthCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check database
	dbHealth := s.db.HealthCheck(ctx)

	// Check cache
	cacheHealth := s.cache.HealthCheck(ctx)

	// Check KB services
	var kb1Status, kb3Status, kb6Status, kb7Status string
	if err := s.kb1Client.Health(ctx); err != nil {
		kb1Status = "unhealthy: " + err.Error()
	} else {
		kb1Status = "healthy"
	}
	if err := s.kb3Client.Health(ctx); err != nil {
		kb3Status = "unhealthy: " + err.Error()
	} else {
		kb3Status = "healthy"
	}
	if err := s.kb6Client.Health(ctx); err != nil {
		kb6Status = "unhealthy: " + err.Error()
	} else {
		kb6Status = "healthy"
	}
	if err := s.kb7Client.Health(ctx); err != nil {
		kb7Status = "unhealthy: " + err.Error()
	} else {
		kb7Status = "healthy"
	}

	overallStatus := "healthy"
	httpStatus := http.StatusOK
	if dbHealth.Status != "healthy" || cacheHealth.Status != "healthy" {
		overallStatus = "degraded"
		httpStatus = http.StatusServiceUnavailable
	}

	c.JSON(httpStatus, gin.H{
		"status":      overallStatus,
		"service":     "kb-12-ordersets-careplans",
		"version":     "1.0.0",
		"environment": s.config.Server.Environment,
		"components": gin.H{
			"database": dbHealth,
			"cache":    cacheHealth,
			"kb1_dosing":     kb1Status,
			"kb3_temporal":   kb3Status,
			"kb6_formulary":  kb6Status,
			"kb7_terminology": kb7Status,
		},
	})
}

// livenessCheck handles GET /health/live
func (s *Server) livenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status": "alive",
	})
}

// readinessCheck handles GET /health/ready
func (s *Server) readinessCheck(c *gin.Context) {
	ctx := c.Request.Context()

	// Check database connectivity
	if err := s.db.Health(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "database unavailable",
		})
		return
	}

	// Check cache connectivity
	if err := s.cache.Health(ctx); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"reason": "cache unavailable",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status": "ready",
	})
}

// metricsHandler handles GET /metrics (Prometheus format)
func (s *Server) metricsHandler(c *gin.Context) {
	ctx := c.Request.Context()

	dbHealth := s.db.HealthCheck(ctx)
	cacheHealth := s.cache.HealthCheck(ctx)

	// Simple Prometheus-style metrics
	metrics := `# HELP kb12_up KB-12 service health status
# TYPE kb12_up gauge
kb12_up 1

# HELP kb12_db_connections_open Number of open database connections
# TYPE kb12_db_connections_open gauge
kb12_db_connections_open ` + fmt.Sprintf("%d", dbHealth.OpenConns) + `

# HELP kb12_db_connections_in_use Number of database connections in use
# TYPE kb12_db_connections_in_use gauge
kb12_db_connections_in_use ` + fmt.Sprintf("%d", dbHealth.InUse) + `

# HELP kb12_cache_hits Total cache hits
# TYPE kb12_cache_hits counter
kb12_cache_hits ` + fmt.Sprintf("%d", cacheHealth.Hits) + `

# HELP kb12_cache_misses Total cache misses
# TYPE kb12_cache_misses counter
kb12_cache_misses ` + fmt.Sprintf("%d", cacheHealth.Misses) + `
`
	c.String(http.StatusOK, metrics)
}
