// Package api provides route definitions for KB-3 Guidelines Service
package api

import (
	"github.com/gin-gonic/gin"
)

// RegisterRoutes registers all API routes
func RegisterRoutes(r *gin.Engine, handler *Handler) {
	// Health & Status
	r.GET("/health", handler.Health)
	r.GET("/metrics", handler.Metrics)
	r.GET("/version", handler.Version)

	// API v1 group
	v1 := r.Group("/v1")
	{
		// ===== Protocol Management =====
		protocols := v1.Group("/protocols")
		{
			protocols.GET("", handler.ListProtocols)
			protocols.GET("/acute", handler.ListAcuteProtocols)
			protocols.GET("/chronic", handler.ListChronicSchedules)
			protocols.GET("/preventive", handler.ListPreventiveSchedules)
			protocols.GET("/search", handler.SearchProtocols)
			protocols.GET("/condition/:condition", handler.GetProtocolsByCondition)
			protocols.GET("/:type/:id", handler.GetProtocol)
		}

		// ===== Pathway Operations =====
		pathways := v1.Group("/pathways")
		{
			pathways.POST("/start", handler.StartPathway)
			pathways.GET("/:id", handler.GetPathwayStatus)
			pathways.GET("/:id/pending", handler.GetPendingActions)
			pathways.GET("/:id/overdue", handler.GetOverdueActions)
			pathways.GET("/:id/constraints", handler.EvaluateConstraints)
			pathways.GET("/:id/audit", handler.GetPathwayAudit)
			pathways.POST("/:id/advance", handler.AdvanceStage)
			pathways.POST("/:id/complete-action", handler.CompleteAction)
			pathways.POST("/:id/suspend", handler.SuspendPathway)
			pathways.POST("/:id/resume", handler.ResumePathway)
			pathways.POST("/:id/cancel", handler.CancelPathway)
		}

		// ===== Patient Operations =====
		patients := v1.Group("/patients")
		{
			patients.GET("/:id/pathways", handler.GetPatientPathways)
			patients.GET("/:id/schedule", handler.GetPatientSchedule)
			patients.GET("/:id/schedule-summary", handler.GetScheduleSummary)
			patients.GET("/:id/overdue", handler.GetPatientOverdue)
			patients.GET("/:id/upcoming", handler.GetPatientUpcoming)
			patients.GET("/:id/export", handler.ExportPatientData)
			patients.POST("/:id/start-protocol", handler.StartProtocolForPatient)
		}

		// ===== Scheduling Operations =====
		schedule := v1.Group("/schedule")
		{
			schedule.GET("/:patientId", handler.GetSchedule)
			schedule.GET("/:patientId/pending", handler.GetSchedulePending)
			schedule.POST("/:patientId/add", handler.AddScheduledItem)
			schedule.POST("/:patientId/complete", handler.CompleteScheduledItem)
		}

		// ===== Temporal Operations =====
		temporal := v1.Group("/temporal")
		{
			temporal.POST("/evaluate", handler.EvaluateTemporalRelation)
			temporal.POST("/next-occurrence", handler.CalculateNextOccurrence)
			temporal.POST("/validate-constraint", handler.ValidateConstraintTiming)
		}

		// ===== Alert Management =====
		alerts := v1.Group("/alerts")
		{
			alerts.POST("/process", handler.ProcessAlerts)
			alerts.GET("/overdue", handler.GetAllOverdue)
		}

		// ===== Batch Operations =====
		batch := v1.Group("/batch")
		{
			batch.POST("/start-protocols", handler.BatchStartProtocols)
		}

		// ===== Governance (from TypeScript conversion) =====
		guidelines := v1.Group("/guidelines")
		{
			guidelines.GET("", handler.GetGuidelines)
			guidelines.GET("/:id", handler.GetGuideline)
		}

		conflicts := v1.Group("/conflicts")
		{
			conflicts.POST("/resolve", handler.ResolveConflict)
		}

		safety := v1.Group("/safety-overrides")
		{
			safety.GET("", handler.GetSafetyOverrides)
			safety.POST("", handler.CreateSafetyOverride)
		}

		versions := v1.Group("/versions")
		{
			versions.POST("", handler.CreateVersion)
			versions.POST("/:id/approve", handler.ProcessApproval)
		}
	}
}

// Middleware provides common middleware functions
type Middleware struct{}

// NewMiddleware creates a new middleware instance
func NewMiddleware() *Middleware {
	return &Middleware{}
}

// CORS adds CORS headers
func (m *Middleware) CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

// RequestLogger logs incoming requests
func (m *Middleware) RequestLogger() gin.HandlerFunc {
	return gin.Logger()
}

// Recovery handles panics
func (m *Middleware) Recovery() gin.HandlerFunc {
	return gin.Recovery()
}

// RateLimiter provides basic rate limiting (placeholder for production implementation)
func (m *Middleware) RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		// TODO: Implement actual rate limiting with Redis
		c.Next()
	}
}
