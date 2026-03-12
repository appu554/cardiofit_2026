// Package api provides the HTTP API server for the Clinical Rules Engine
package api

import (
	"context"
	"net/http"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/database"
	"github.com/cardiofit/kb-10-rules-engine/internal/engine"
	"github.com/cardiofit/kb-10-rules-engine/internal/loader"
	"github.com/cardiofit/kb-10-rules-engine/internal/metrics"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

// Server provides the HTTP API server
type Server struct {
	config     *config.Config
	engine     *engine.RulesEngine
	store      *models.RuleStore
	db         *database.PostgresDB
	loader     *loader.YAMLLoader
	logger     *logrus.Logger
	metrics    *metrics.Collector
	router     *gin.Engine
	startTime  time.Time
}

// NewServer creates a new API server
func NewServer(
	cfg *config.Config,
	rulesEngine *engine.RulesEngine,
	store *models.RuleStore,
	db *database.PostgresDB,
	yamlLoader *loader.YAMLLoader,
	logger *logrus.Logger,
	metricsCollector *metrics.Collector,
) *Server {
	// Set Gin mode based on log level
	if cfg.Logging.Level == "debug" {
		gin.SetMode(gin.DebugMode)
	} else {
		gin.SetMode(gin.ReleaseMode)
	}

	server := &Server{
		config:    cfg,
		engine:    rulesEngine,
		store:     store,
		db:        db,
		loader:    yamlLoader,
		logger:    logger,
		metrics:   metricsCollector,
		startTime: time.Now(),
	}

	server.setupRouter()
	return server
}

// Router returns the gin router
func (s *Server) Router() *gin.Engine {
	return s.router
}

// setupRouter configures all routes
func (s *Server) setupRouter() {
	s.router = gin.New()

	// Middleware
	s.router.Use(gin.Recovery())
	s.router.Use(s.requestIDMiddleware())
	s.router.Use(s.loggingMiddleware())
	s.router.Use(s.corsMiddleware())

	// Health endpoints
	s.router.GET("/health", s.healthHandler)
	s.router.GET("/ready", s.readyHandler)
	s.router.GET("/metrics", gin.WrapH(promhttp.Handler()))

	// API v1
	v1 := s.router.Group("/api/v1")
	{
		// Evaluation endpoints
		v1.POST("/evaluate", s.evaluateHandler)
		v1.POST("/evaluate/rules", s.evaluateSpecificHandler)
		v1.POST("/evaluate/type/:type", s.evaluateByTypeHandler)
		v1.POST("/evaluate/category/:category", s.evaluateByCategoryHandler)
		v1.POST("/evaluate/tags", s.evaluateByTagsHandler)

		// Rules management endpoints
		v1.GET("/rules", s.listRulesHandler)
		v1.GET("/rules/:id", s.getRuleHandler)
		v1.POST("/rules", s.createRuleHandler)
		v1.PUT("/rules/:id", s.updateRuleHandler)
		v1.DELETE("/rules/:id", s.deleteRuleHandler)
		v1.POST("/rules/reload", s.reloadRulesHandler)
		v1.GET("/rules/stats", s.ruleStatsHandler)
		v1.GET("/rules/types", s.ruleTypesHandler)
		v1.GET("/rules/categories", s.ruleCategoriesHandler)
		v1.GET("/rules/tags", s.ruleTagsHandler)

		// Alerts endpoints
		v1.GET("/alerts", s.listAlertsHandler)
		v1.GET("/alerts/:id", s.getAlertHandler)
		v1.POST("/alerts/:id/acknowledge", s.acknowledgeAlertHandler)
		v1.POST("/alerts/:id/resolve", s.resolveAlertHandler)
		v1.GET("/alerts/patient/:patientId", s.patientAlertsHandler)

		// Cache endpoints
		v1.GET("/cache/stats", s.cacheStatsHandler)
		v1.POST("/cache/clear", s.cacheClearHandler)
		v1.POST("/cache/invalidate/:patientId", s.cacheInvalidateHandler)
	}
}

// Middleware

func (s *Server) requestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		c.Next()
	}
}

func (s *Server) loggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		s.logger.WithFields(logrus.Fields{
			"request_id": c.GetString("request_id"),
			"method":     c.Request.Method,
			"path":       c.Request.URL.Path,
			"status":     c.Writer.Status(),
			"duration":   time.Since(start).String(),
			"client_ip":  c.ClientIP(),
		}).Info("Request completed")
	}
}

func (s *Server) corsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusOK)
			return
		}

		c.Next()
	}
}

// Health Handlers

func (s *Server) healthHandler(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "healthy",
		"service":   "kb-10-rules-engine",
		"version":   "1.0.0",
		"uptime":    time.Since(s.startTime).String(),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (s *Server) readyHandler(c *gin.Context) {
	// Check database connectivity (if database is configured)
	if s.db != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := s.db.Ping(ctx); err != nil {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status": "not ready",
				"error":  "database connection failed",
			})
			return
		}
	}

	// Check if rules are loaded
	if s.store.Count() == 0 {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "not ready",
			"error":  "no rules loaded",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":      "ready",
		"rules_count": s.store.Count(),
	})
}

// Evaluation Handlers

func (s *Server) evaluateHandler(c *gin.Context) {
	var evalCtx models.EvaluationContext
	if err := c.ShouldBindJSON(&evalCtx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if evalCtx.PatientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id is required"})
		return
	}

	evalCtx.RequestID = c.GetString("request_id")
	startTime := time.Now()

	results, err := s.engine.Evaluate(c.Request.Context(), &evalCtx)
	if err != nil {
		s.logger.WithError(err).Error("Evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := engine.BuildEvaluateResponse(&evalCtx, results, startTime)
	c.JSON(http.StatusOK, response)
}

func (s *Server) evaluateSpecificHandler(c *gin.Context) {
	var req struct {
		RuleIDs []string                  `json:"rule_ids" binding:"required"`
		Context models.EvaluationContext `json:"context" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Context.PatientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "context.patient_id is required"})
		return
	}

	req.Context.RequestID = c.GetString("request_id")
	startTime := time.Now()

	results, err := s.engine.EvaluateSpecific(c.Request.Context(), req.RuleIDs, &req.Context)
	if err != nil {
		s.logger.WithError(err).Error("Evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := engine.BuildEvaluateResponse(&req.Context, results, startTime)
	c.JSON(http.StatusOK, response)
}

func (s *Server) evaluateByTypeHandler(c *gin.Context) {
	ruleType := c.Param("type")

	var evalCtx models.EvaluationContext
	if err := c.ShouldBindJSON(&evalCtx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if evalCtx.PatientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id is required"})
		return
	}

	evalCtx.RequestID = c.GetString("request_id")
	startTime := time.Now()

	results, err := s.engine.EvaluateByType(c.Request.Context(), ruleType, &evalCtx)
	if err != nil {
		s.logger.WithError(err).Error("Evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := engine.BuildEvaluateResponse(&evalCtx, results, startTime)
	c.JSON(http.StatusOK, response)
}

func (s *Server) evaluateByCategoryHandler(c *gin.Context) {
	category := c.Param("category")

	var evalCtx models.EvaluationContext
	if err := c.ShouldBindJSON(&evalCtx); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if evalCtx.PatientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "patient_id is required"})
		return
	}

	evalCtx.RequestID = c.GetString("request_id")
	startTime := time.Now()

	results, err := s.engine.EvaluateByCategory(c.Request.Context(), category, &evalCtx)
	if err != nil {
		s.logger.WithError(err).Error("Evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := engine.BuildEvaluateResponse(&evalCtx, results, startTime)
	c.JSON(http.StatusOK, response)
}

func (s *Server) evaluateByTagsHandler(c *gin.Context) {
	var req struct {
		Tags    []string                  `json:"tags" binding:"required"`
		Context models.EvaluationContext `json:"context" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if req.Context.PatientID == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "context.patient_id is required"})
		return
	}

	req.Context.RequestID = c.GetString("request_id")
	startTime := time.Now()

	results, err := s.engine.EvaluateByTags(c.Request.Context(), req.Tags, &req.Context)
	if err != nil {
		s.logger.WithError(err).Error("Evaluation failed")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	response := engine.BuildEvaluateResponse(&req.Context, results, startTime)
	c.JSON(http.StatusOK, response)
}

// Rules Management Handlers

func (s *Server) listRulesHandler(c *gin.Context) {
	filter := &models.Filter{}

	if types := c.QueryArray("type"); len(types) > 0 {
		filter.Types = types
	}
	if categories := c.QueryArray("category"); len(categories) > 0 {
		filter.Categories = categories
	}
	if statuses := c.QueryArray("status"); len(statuses) > 0 {
		filter.Statuses = statuses
	}
	if tags := c.QueryArray("tag"); len(tags) > 0 {
		filter.Tags = tags
	}

	rules := s.store.Query(filter)

	c.JSON(http.StatusOK, gin.H{
		"count": len(rules),
		"rules": rules,
	})
}

func (s *Server) getRuleHandler(c *gin.Context) {
	id := c.Param("id")

	rule, exists := s.store.Get(id)
	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	c.JSON(http.StatusOK, rule)
}

func (s *Server) createRuleHandler(c *gin.Context) {
	var rule models.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := rule.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Check for duplicate ID
	if s.store.HasRule(rule.ID) {
		c.JSON(http.StatusConflict, gin.H{"error": "rule with this ID already exists"})
		return
	}

	now := time.Now()
	rule.CreatedAt = now
	rule.UpdatedAt = now

	// Save to database (if database is configured)
	if s.db != nil {
		if err := s.db.SaveRule(c.Request.Context(), &rule); err != nil {
			s.logger.WithError(err).Error("Failed to save rule")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save rule"})
			return
		}
	}

	// Add to store
	s.store.Add(&rule)
	s.store.SortByPriority()

	// Invalidate cache
	s.engine.GetCache().Clear()

	c.JSON(http.StatusCreated, rule)
}

func (s *Server) updateRuleHandler(c *gin.Context) {
	id := c.Param("id")

	if !s.store.HasRule(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	var rule models.Rule
	if err := c.ShouldBindJSON(&rule); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	rule.ID = id
	rule.UpdatedAt = time.Now()

	if err := rule.Validate(); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Update in database (if database is configured)
	if s.db != nil {
		if err := s.db.SaveRule(c.Request.Context(), &rule); err != nil {
			s.logger.WithError(err).Error("Failed to update rule")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update rule"})
			return
		}
	}

	// Update in store
	s.store.Remove(id)
	s.store.Add(&rule)
	s.store.SortByPriority()

	// Invalidate cache for this rule
	s.engine.GetCache().InvalidateRule(id)

	c.JSON(http.StatusOK, rule)
}

func (s *Server) deleteRuleHandler(c *gin.Context) {
	id := c.Param("id")

	if !s.store.HasRule(id) {
		c.JSON(http.StatusNotFound, gin.H{"error": "rule not found"})
		return
	}

	// Delete from database (if database is configured)
	if s.db != nil {
		if err := s.db.DeleteRule(c.Request.Context(), id); err != nil {
			s.logger.WithError(err).Error("Failed to delete rule")
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete rule"})
			return
		}
	}

	// Remove from store
	s.store.Remove(id)

	// Invalidate cache
	s.engine.GetCache().InvalidateRule(id)

	c.JSON(http.StatusOK, gin.H{"message": "rule deleted", "id": id})
}

func (s *Server) reloadRulesHandler(c *gin.Context) {
	startTime := time.Now()

	// Check if loader is configured
	if s.loader == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "rules loader not configured"})
		return
	}

	if err := s.loader.Reload(); err != nil {
		s.logger.WithError(err).Error("Failed to reload rules")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Clear cache after reload
	s.engine.GetCache().Clear()

	duration := time.Since(startTime)
	s.store.SetReloadMetadata(time.Now(), duration)

	c.JSON(http.StatusOK, gin.H{
		"message":          "rules reloaded",
		"rules_count":      s.store.Count(),
		"reload_duration":  duration.String(),
	})
}

func (s *Server) ruleStatsHandler(c *gin.Context) {
	stats := s.store.GetStats()
	c.JSON(http.StatusOK, stats)
}

func (s *Server) ruleTypesHandler(c *gin.Context) {
	types := s.store.GetTypes()
	c.JSON(http.StatusOK, gin.H{"types": types})
}

func (s *Server) ruleCategoriesHandler(c *gin.Context) {
	categories := s.store.GetCategories()
	c.JSON(http.StatusOK, gin.H{"categories": categories})
}

func (s *Server) ruleTagsHandler(c *gin.Context) {
	tags := s.store.GetTags()
	c.JSON(http.StatusOK, gin.H{"tags": tags})
}

// Alerts Handlers

func (s *Server) listAlertsHandler(c *gin.Context) {
	status := c.Query("status")
	severity := c.Query("severity")
	limit := 100 // Default limit
	offset := 0

	alerts, err := s.db.ListAlerts(c.Request.Context(), status, severity, limit, offset)
	if err != nil {
		s.logger.WithError(err).Error("Failed to list alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"count":  len(alerts),
		"alerts": alerts,
	})
}

func (s *Server) getAlertHandler(c *gin.Context) {
	id := c.Param("id")

	alert, err := s.db.GetAlert(c.Request.Context(), id)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if alert == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "alert not found"})
		return
	}

	c.JSON(http.StatusOK, alert)
}

func (s *Server) acknowledgeAlertHandler(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		AcknowledgedBy string `json:"acknowledged_by" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.db.AcknowledgeAlert(c.Request.Context(), id, req.AcknowledgedBy); err != nil {
		s.logger.WithError(err).Error("Failed to acknowledge alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "alert acknowledged",
		"id":              id,
		"acknowledged_by": req.AcknowledgedBy,
	})
}

func (s *Server) resolveAlertHandler(c *gin.Context) {
	id := c.Param("id")

	var req struct {
		ResolvedBy string `json:"resolved_by" binding:"required"`
		Resolution string `json:"resolution"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := s.db.ResolveAlert(c.Request.Context(), id, req.ResolvedBy, req.Resolution); err != nil {
		s.logger.WithError(err).Error("Failed to resolve alert")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":     "alert resolved",
		"id":          id,
		"resolved_by": req.ResolvedBy,
	})
}

func (s *Server) patientAlertsHandler(c *gin.Context) {
	patientID := c.Param("patientId")
	status := c.Query("status")

	alerts, err := s.db.GetPatientAlerts(c.Request.Context(), patientID, status)
	if err != nil {
		s.logger.WithError(err).Error("Failed to get patient alerts")
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"patient_id": patientID,
		"count":      len(alerts),
		"alerts":     alerts,
	})
}

// Cache Handlers

func (s *Server) cacheStatsHandler(c *gin.Context) {
	stats := s.engine.GetCache().Stats()
	c.JSON(http.StatusOK, stats)
}

func (s *Server) cacheClearHandler(c *gin.Context) {
	s.engine.GetCache().Clear()
	c.JSON(http.StatusOK, gin.H{"message": "cache cleared"})
}

func (s *Server) cacheInvalidateHandler(c *gin.Context) {
	patientID := c.Param("patientId")
	s.engine.GetCache().Invalidate(patientID)
	c.JSON(http.StatusOK, gin.H{
		"message":    "cache invalidated for patient",
		"patient_id": patientID,
	})
}
