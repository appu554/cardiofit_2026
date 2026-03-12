// Package api provides HTTP handlers for KB-18 Governance Engine
package api

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"

	"kb-18-governance-engine/internal/config"
	"kb-18-governance-engine/pkg/engine"
	"kb-18-governance-engine/pkg/override"
	"kb-18-governance-engine/pkg/programs"
	"kb-18-governance-engine/pkg/types"
)

// Server represents the HTTP server for KB-18 Governance Engine
type Server struct {
	config *config.Config
	router *gin.Engine
	server *http.Server
	log    *logrus.Entry

	// Core components
	governanceEngine *engine.GovernanceEngine
	programStore     *programs.ProgramStore
	overrideStore    *override.OverrideStore
}

// NewServer creates a new HTTP server instance
func NewServer(cfg *config.Config) (*Server, error) {
	log := logrus.WithField("component", "api-server")

	// Set Gin mode based on environment
	if cfg.Server.Environment == "production" {
		gin.SetMode(gin.ReleaseMode)
	}

	// Initialize program store with pre-configured programs
	programStore := programs.NewProgramStore()
	log.WithField("programs", programStore.Count()).Info("Initialized program store")

	// Initialize governance engine
	governanceEngine := engine.NewGovernanceEngine(programStore)
	log.Info("Initialized governance engine")

	// Initialize override store
	overrideStore := override.NewOverrideStore()
	log.Info("Initialized override store")

	// Create server instance
	s := &Server{
		config:           cfg,
		log:              log,
		governanceEngine: governanceEngine,
		programStore:     programStore,
		overrideStore:    overrideStore,
	}

	// Setup router
	s.setupRouter()

	return s, nil
}

// setupRouter configures all HTTP routes
func (s *Server) setupRouter() {
	router := gin.New()

	// Apply middleware
	router.Use(gin.Recovery())
	router.Use(s.requestLogger())
	router.Use(s.corsMiddleware())
	router.Use(s.requestIDMiddleware())

	// Health endpoints
	router.GET("/health", s.healthCheck)
	router.GET("/ready", s.readinessCheck)
	router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1 routes
	v1 := router.Group("/api/v1")
	{
		// Core evaluation endpoints
		v1.POST("/evaluate", s.Evaluate)
		v1.POST("/evaluate/medication", s.EvaluateMedication)
		v1.POST("/evaluate/protocol", s.EvaluateProtocol)

		// Program management endpoints
		programs := v1.Group("/programs")
		{
			programs.GET("", s.ListPrograms)
			programs.GET("/:code", s.GetProgram)
		}

		// Override management endpoints
		overrides := v1.Group("/overrides")
		{
			overrides.GET("", s.ListOverrides)
			overrides.GET("/:id", s.GetOverride)
			overrides.POST("/request", s.RequestOverride)
			overrides.POST("/approve", s.ApproveOverride)
			overrides.POST("/deny", s.DenyOverride)
		}

		// Acknowledgment endpoints
		v1.GET("/acknowledgments", s.ListAcknowledgments)
		v1.POST("/acknowledge", s.RecordAcknowledgment)

		// Escalation management endpoints
		escalations := v1.Group("/escalations")
		{
			escalations.GET("", s.ListEscalations)
			escalations.GET("/:id", s.GetEscalation)
			escalations.POST("", s.CreateEscalation)
			escalations.POST("/:id/resolve", s.ResolveEscalation)
		}

		// Analytics and audit endpoints
		v1.GET("/stats", s.GetStats)
		v1.GET("/audit/pattern", s.GetPatternAnalysis)
		v1.GET("/audit/trail/:id", s.GetEvidenceTrail)
	}

	s.router = router
}

// Start starts the HTTP server
func (s *Server) Start() error {
	addr := fmt.Sprintf(":%s", s.config.Server.Port)

	s.server = &http.Server{
		Addr:         addr,
		Handler:      s.router,
		ReadTimeout:  s.config.Server.ReadTimeout,
		WriteTimeout: s.config.Server.WriteTimeout,
		IdleTimeout:  s.config.Server.IdleTimeout,
	}

	s.log.WithField("port", s.config.Server.Port).Info("Starting KB-18 Governance Engine server")

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return fmt.Errorf("failed to start server: %w", err)
	}

	return nil
}

// Shutdown gracefully shuts down the server
func (s *Server) Shutdown(ctx context.Context) error {
	s.log.Info("Shutting down server...")

	if s.server != nil {
		if err := s.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("server shutdown failed: %w", err)
		}
	}

	s.log.Info("Server shutdown complete")
	return nil
}

// Router returns the HTTP handler for testing purposes
func (s *Server) Router() http.Handler {
	return s.router
}

// ==================== Middleware ====================

// requestLogger logs HTTP requests
func (s *Server) requestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		raw := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		statusCode := c.Writer.Status()

		entry := s.log.WithFields(logrus.Fields{
			"status":    statusCode,
			"method":    c.Request.Method,
			"path":      path,
			"query":     raw,
			"latency":   latency.String(),
			"client_ip": c.ClientIP(),
		})

		if statusCode >= 500 {
			entry.Error("Server error")
		} else if statusCode >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request completed")
		}
	}
}

// corsMiddleware handles CORS headers
func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// requestIDMiddleware adds a request ID to each request
func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = fmt.Sprintf("kb18-%d", time.Now().UnixNano())
		}
		c.Set("request_id", requestID)
		c.Writer.Header().Set("X-Request-ID", requestID)
		c.Next()
	}
}

// ==================== Health Endpoints ====================

// healthCheck returns health status
func (s *Server) healthCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "kb-18-governance-engine",
		"version":   "1.0.0",
		"timestamp": time.Now().UTC().Format(time.RFC3339),
	})
}

// readinessCheck checks if the service is ready
func (s *Server) readinessCheck(c *gin.Context) {
	// Check governance engine is initialized
	if s.governanceEngine == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "governance engine not initialized",
		})
		return
	}

	// Check program store has programs loaded
	if s.programStore.Count() == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "no programs loaded",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":        "ready",
		"programs":      s.programStore.Count(),
		"engine_status": "operational",
	})
}

// ==================== Evaluation Endpoints ====================

// Evaluate performs general governance evaluation
func (s *Server) Evaluate(c *gin.Context) {
	var req types.EvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.PatientContext == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "patient_context is required",
		})
		return
	}

	ctx := c.Request.Context()
	response, err := s.governanceEngine.Evaluate(ctx, &req)
	if err != nil {
		s.log.WithError(err).Error("Evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "evaluation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// EvaluateMedication evaluates a medication order
func (s *Server) EvaluateMedication(c *gin.Context) {
	var req types.EvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields for medication evaluation
	if req.PatientContext == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "patient_context is required",
		})
		return
	}

	if req.MedicationOrder == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "medication_order is required for medication evaluation",
		})
		return
	}

	// Copy MedicationOrder to Order for engine compatibility
	// The engine uses req.Order, while the API accepts medicationOrder
	if req.Order == nil {
		req.Order = req.MedicationOrder
	}

	// Set evaluation type
	req.EvaluationType = types.EvalTypeMedicationOrder

	ctx := c.Request.Context()
	response, err := s.governanceEngine.Evaluate(ctx, &req)
	if err != nil {
		s.log.WithError(err).Error("Medication evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "medication evaluation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// EvaluateProtocol evaluates protocol compliance
func (s *Server) EvaluateProtocol(c *gin.Context) {
	var req types.EvaluationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.PatientContext == nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "patient_context is required",
		})
		return
	}

	// Set evaluation type
	req.EvaluationType = types.EvalTypeProtocolCompliance

	ctx := c.Request.Context()
	response, err := s.governanceEngine.Evaluate(ctx, &req)
	if err != nil {
		s.log.WithError(err).Error("Protocol evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "protocol evaluation failed",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, response)
}

// ==================== Program Endpoints ====================

// ListPrograms returns all available governance programs
func (s *Server) ListPrograms(c *gin.Context) {
	allPrograms := s.programStore.GetAll()

	programList := make([]gin.H, 0, len(allPrograms))
	for _, p := range allPrograms {
		programList = append(programList, gin.H{
			"code":        p.Code,
			"name":        p.Name,
			"description": p.Description,
			"category":    p.Category,
			"rules_count": len(p.Rules),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    len(programList),
		"programs": programList,
	})
}

// GetProgram returns details for a specific program
func (s *Server) GetProgram(c *gin.Context) {
	code := c.Param("code")
	if code == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "program code is required",
		})
		return
	}

	program := s.programStore.Get(code)
	if program == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": fmt.Sprintf("program '%s' not found", code),
		})
		return
	}

	// Build rules summary
	rulesSummary := make([]gin.H, 0, len(program.Rules))
	for _, r := range program.Rules {
		rulesSummary = append(rulesSummary, gin.H{
			"code":              r.GetCode(),
			"name":              r.Name,
			"description":       r.Description,
			"severity":          r.Severity,
			"enforcement_level": r.EnforcementLevel,
			"conditions_count":  len(r.Conditions),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"code":               program.Code,
		"name":               program.Name,
		"description":        program.Description,
		"category":           program.Category,
		"activation_criteria": program.ActivationCriteria,
		"accountability_chain": program.AccountabilityChain,
		"rules":              rulesSummary,
	})
}

// ==================== Override Endpoints ====================

// ListOverrides returns all override requests
func (s *Server) ListOverrides(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse filters from query params
	status := c.Query("status")
	patientID := c.Query("patient_id")
	requestorID := c.Query("requestor_id")

	overrides := s.overrideStore.ListOverrides(ctx)

	// Apply filters
	filtered := make([]*types.OverrideRequest, 0)
	for _, o := range overrides {
		if status != "" && string(o.Status) != status {
			continue
		}
		if patientID != "" && o.PatientID != patientID {
			continue
		}
		if requestorID != "" && o.RequestorID != requestorID {
			continue
		}
		filtered = append(filtered, o)
	}

	c.JSON(http.StatusOK, gin.H{
		"total":     len(filtered),
		"overrides": filtered,
	})
}

// GetOverride returns a specific override request
func (s *Server) GetOverride(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "override ID is required",
		})
		return
	}

	ctx := c.Request.Context()
	override, err := s.overrideStore.GetOverride(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, override)
}

// RequestOverride creates a new override request
func (s *Server) RequestOverride(c *gin.Context) {
	var req types.OverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if req.ViolationID == "" || req.RequestorID == "" || req.Reason == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "violation_id, requestor_id, and reason are required",
		})
		return
	}

	ctx := c.Request.Context()
	if err := s.overrideStore.RequestOverride(ctx, &req); err != nil {
		s.log.WithError(err).Error("Failed to create override request")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create override request",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":     "Override request created",
		"override_id": req.ID,
		"status":      req.Status,
	})
}

// ApproveOverrideRequest represents the request body for approving an override
type ApproveOverrideRequest struct {
	OverrideID string `json:"override_id" binding:"required"`
	ApproverID string `json:"approver_id" binding:"required"`
	Reason     string `json:"reason"`
}

// ApproveOverride approves an override request
func (s *Server) ApproveOverride(c *gin.Context) {
	var req ApproveOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	if err := s.overrideStore.ApproveOverride(ctx, req.OverrideID, req.ApproverID); err != nil {
		s.log.WithError(err).Error("Failed to approve override")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to approve override",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Override approved",
		"override_id": req.OverrideID,
		"approved_by": req.ApproverID,
	})
}

// DenyOverrideRequest represents the request body for denying an override
type DenyOverrideRequest struct {
	OverrideID string `json:"override_id" binding:"required"`
	DenierID   string `json:"denier_id" binding:"required"`
	Reason     string `json:"reason" binding:"required"`
}

// DenyOverride denies an override request
func (s *Server) DenyOverride(c *gin.Context) {
	var req DenyOverrideRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	if err := s.overrideStore.DenyOverride(ctx, req.OverrideID, req.DenierID, req.Reason); err != nil {
		s.log.WithError(err).Error("Failed to deny override")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to deny override",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "Override denied",
		"override_id": req.OverrideID,
		"denied_by":   req.DenierID,
	})
}

// ==================== Acknowledgment Endpoints ====================

// ListAcknowledgments returns all acknowledgments
func (s *Server) ListAcknowledgments(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse filters
	userID := c.Query("user_id")
	violationID := c.Query("violation_id")

	acknowledgments := s.overrideStore.ListAcknowledgments(ctx)

	// Apply filters
	filtered := make([]*types.Acknowledgment, 0)
	for _, a := range acknowledgments {
		if userID != "" && a.UserID != userID {
			continue
		}
		if violationID != "" && a.ViolationID != violationID {
			continue
		}
		filtered = append(filtered, a)
	}

	c.JSON(http.StatusOK, gin.H{
		"total":           len(filtered),
		"acknowledgments": filtered,
	})
}

// RecordAcknowledgment records a new acknowledgment
func (s *Server) RecordAcknowledgment(c *gin.Context) {
	var ack types.Acknowledgment
	if err := c.ShouldBindJSON(&ack); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if ack.ViolationID == "" || ack.UserID == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "violation_id and user_id are required",
		})
		return
	}

	ctx := c.Request.Context()
	if err := s.overrideStore.RecordAcknowledgment(ctx, &ack); err != nil {
		s.log.WithError(err).Error("Failed to record acknowledgment")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to record acknowledgment",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":           "Acknowledgment recorded",
		"acknowledgment_id": ack.ID,
		"timestamp":         ack.Timestamp,
	})
}

// ==================== Escalation Endpoints ====================

// ListEscalations returns all escalations
func (s *Server) ListEscalations(c *gin.Context) {
	ctx := c.Request.Context()

	// Parse filters
	status := c.Query("status")
	level := c.Query("level")

	escalations := s.overrideStore.ListEscalations(ctx)

	// Apply filters
	filtered := make([]*types.Escalation, 0)
	for _, e := range escalations {
		if status != "" && string(e.Status) != status {
			continue
		}
		if level != "" && e.Level != level {
			continue
		}
		filtered = append(filtered, e)
	}

	c.JSON(http.StatusOK, gin.H{
		"total":       len(filtered),
		"escalations": filtered,
	})
}

// GetEscalation returns a specific escalation
func (s *Server) GetEscalation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "escalation ID is required",
		})
		return
	}

	ctx := c.Request.Context()
	escalation, err := s.overrideStore.GetEscalation(ctx, id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"error": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, escalation)
}

// CreateEscalation creates a new escalation
func (s *Server) CreateEscalation(c *gin.Context) {
	var esc types.Escalation
	if err := c.ShouldBindJSON(&esc); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	// Validate required fields
	if esc.ViolationID == "" || esc.Level == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "violation_id and level are required",
		})
		return
	}

	ctx := c.Request.Context()
	if err := s.overrideStore.CreateEscalation(ctx, &esc); err != nil {
		s.log.WithError(err).Error("Failed to create escalation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to create escalation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"message":       "Escalation created",
		"escalation_id": esc.ID,
		"status":        esc.Status,
	})
}

// ResolveEscalationRequest represents the request body for resolving an escalation
type ResolveEscalationRequest struct {
	ResolverID string `json:"resolver_id" binding:"required"`
	Resolution string `json:"resolution" binding:"required"`
}

// ResolveEscalation resolves an escalation
func (s *Server) ResolveEscalation(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "escalation ID is required",
		})
		return
	}

	var req ResolveEscalationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"error":   "invalid request body",
			"details": err.Error(),
		})
		return
	}

	ctx := c.Request.Context()
	if err := s.overrideStore.ResolveEscalation(ctx, id, req.ResolverID, req.Resolution); err != nil {
		s.log.WithError(err).Error("Failed to resolve escalation")
		c.JSON(http.StatusInternalServerError, gin.H{
			"error":   "failed to resolve escalation",
			"details": err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":       "Escalation resolved",
		"escalation_id": id,
		"resolved_by":   req.ResolverID,
	})
}

// ==================== Analytics Endpoints ====================

// GetStats returns engine statistics
func (s *Server) GetStats(c *gin.Context) {
	stats := s.governanceEngine.GetStats()
	ctx := c.Request.Context()

	// Get override statistics
	overrides := s.overrideStore.ListOverrides(ctx)
	pendingOverrides := 0
	approvedOverrides := 0
	deniedOverrides := 0
	for _, o := range overrides {
		switch o.Status {
		case types.OverrideStatusPending:
			pendingOverrides++
		case types.OverrideStatusApproved:
			approvedOverrides++
		case types.OverrideStatusDenied:
			deniedOverrides++
		}
	}

	// Get escalation statistics
	escalations := s.overrideStore.ListEscalations(ctx)
	openEscalations := 0
	resolvedEscalations := 0
	for _, e := range escalations {
		switch e.Status {
		case types.EscalationStatusOpen:
			openEscalations++
		case types.EscalationStatusResolved:
			resolvedEscalations++
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"engine": gin.H{
			"total_evaluations":    stats.TotalEvaluations,
			"total_violations":     stats.TotalViolations,
			"total_blocked":        stats.TotalBlocked,
			"total_allowed":        stats.TotalAllowed,
			"programs_evaluated":   stats.ProgramsEvaluated,
			"rules_evaluated":      stats.RulesEvaluated,
			"avg_evaluation_time":  stats.AvgEvaluationTime.String(),
			"last_evaluation_time": stats.LastEvaluationTime.Format(time.RFC3339),
		},
		"overrides": gin.H{
			"total":    len(overrides),
			"pending":  pendingOverrides,
			"approved": approvedOverrides,
			"denied":   deniedOverrides,
		},
		"escalations": gin.H{
			"total":    len(escalations),
			"open":     openEscalations,
			"resolved": resolvedEscalations,
		},
		"programs": gin.H{
			"total_loaded": s.programStore.Count(),
		},
	})
}

// GetPatternAnalysis returns override pattern analysis
func (s *Server) GetPatternAnalysis(c *gin.Context) {
	ctx := c.Request.Context()
	patterns := s.overrideStore.GetPatternAnalysis(ctx)

	patternList := make([]gin.H, 0)
	for key, pattern := range patterns {
		patternList = append(patternList, gin.H{
			"key":          key,
			"requestor_id": pattern.RequestorID,
			"rule_code":    pattern.RuleCode,
			"count_24h":    pattern.Count24h,
			"count_7d":     pattern.Count7d,
			"flagged":      pattern.Flagged,
			"last_request": pattern.LastRequest.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"total":    len(patternList),
		"patterns": patternList,
	})
}

// GetEvidenceTrail returns the evidence trail for a specific evaluation
func (s *Server) GetEvidenceTrail(c *gin.Context) {
	id := c.Param("id")
	if id == "" {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "evidence trail ID is required",
		})
		return
	}

	// In a full implementation, this would fetch from persistent storage
	// For now, return a placeholder indicating the feature
	c.JSON(http.StatusOK, gin.H{
		"id":      id,
		"message": "Evidence trail lookup requires persistent storage integration",
		"note":    "Each evaluation generates an immutable evidence trail with SHA-256 hash",
	})
}
