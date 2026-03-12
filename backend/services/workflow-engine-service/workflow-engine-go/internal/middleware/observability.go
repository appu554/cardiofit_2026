package middleware

import (
	"runtime"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"workflow-engine/internal/monitoring"
)

// ObservabilityMiddleware combines metrics, tracing, and logging
type ObservabilityMiddleware struct {
	metrics *monitoring.Metrics
	tracer  *monitoring.TracingProvider
	logger  *zap.Logger
}

// NewObservabilityMiddleware creates new observability middleware
func NewObservabilityMiddleware(
	metrics *monitoring.Metrics,
	tracer *monitoring.TracingProvider,
	logger *zap.Logger,
) *ObservabilityMiddleware {
	return &ObservabilityMiddleware{
		metrics: metrics,
		tracer:  tracer,
		logger:  logger,
	}
}

// MetricsMiddleware records HTTP request metrics
func (o *ObservabilityMiddleware) MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method

		// Get request size
		requestSize := c.Request.ContentLength
		if requestSize < 0 {
			requestSize = 0
		}

		c.Next()

		// Record metrics after request completion
		duration := time.Since(start)
		statusCode := c.Writer.Status()
		responseSize := int64(c.Writer.Size())

		// Record HTTP metrics
		o.metrics.RecordHTTPRequest(method, path, duration, statusCode, requestSize, responseSize)

		// Log performance warnings for slow requests
		if duration > 5*time.Second {
			o.logger.Warn("Slow HTTP request detected",
				zap.String("method", method),
				zap.String("path", path),
				zap.Duration("duration", duration),
				zap.Int("status_code", statusCode))
		}
	}
}

// TracingMiddleware adds distributed tracing to HTTP requests
func (o *ObservabilityMiddleware) TracingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		method := c.Request.Method

		// Start HTTP span
		ctx, span := o.tracer.StartHTTPSpan(c.Request.Context(), method, path)
		c.Request = c.Request.WithContext(ctx)

		// Add request attributes
		o.tracer.AddWorkflowAttributes(span, map[string]interface{}{
			"http.method":      method,
			"http.url":         c.Request.URL.String(),
			"http.user_agent":  c.GetHeader("User-Agent"),
			"http.remote_addr": c.ClientIP(),
		})

		// Add correlation ID if present
		if correlationID := c.GetHeader("X-Correlation-ID"); correlationID != "" {
			o.tracer.AddWorkflowAttributes(span, map[string]interface{}{
				"correlation.id": correlationID,
			})
		}

		// Add user context if available
		if user, exists := c.Get("user"); exists {
			if userContext, ok := user.(*UserContext); ok {
				o.tracer.AddClinicalAttributes(span, "", userContext.ProviderID, "")
				o.tracer.AddWorkflowAttributes(span, map[string]interface{}{
					"user.id":        userContext.UserID,
					"user.is_provider": userContext.IsProvider,
				})
			}
		}

		c.Next()

		// Finish span with status
		statusCode := c.Writer.Status()
		success := statusCode < 400
		
		o.tracer.AddWorkflowAttributes(span, map[string]interface{}{
			"http.status_code":   statusCode,
			"http.response_size": c.Writer.Size(),
		})

		monitoring.FinishSpanWithStatus(span, success, statusCode)
	}
}

// ResourceMonitoringMiddleware tracks system resources
func (o *ObservabilityMiddleware) ResourceMonitoringMiddleware() gin.HandlerFunc {
	// Update resource metrics periodically
	go o.updateResourceMetrics()

	return func(c *gin.Context) {
		c.Next()
	}
}

// updateResourceMetrics periodically updates system resource metrics
func (o *ObservabilityMiddleware) updateResourceMetrics() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		// Update memory metrics
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)

		o.metrics.MemoryUsage.WithLabelValues("heap").Set(float64(memStats.HeapAlloc))
		o.metrics.MemoryUsage.WithLabelValues("stack").Set(float64(memStats.StackInuse))
		o.metrics.MemoryUsage.WithLabelValues("sys").Set(float64(memStats.Sys))

		// Update goroutine count
		o.metrics.GoroutineCount.Set(float64(runtime.NumGoroutine()))

		o.logger.Debug("Updated resource metrics",
			zap.Uint64("heap_alloc", memStats.HeapAlloc),
			zap.Uint64("stack_inuse", memStats.StackInuse),
			zap.Uint64("sys", memStats.Sys),
			zap.Int("goroutines", runtime.NumGoroutine()))
	}
}

// WorkflowObservabilityMiddleware adds workflow-specific observability
func (o *ObservabilityMiddleware) WorkflowObservabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to workflow orchestration endpoints
		if !isWorkflowEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()

		// Add workflow-specific logging context
		contextLogger := o.logger.With(
			zap.String("endpoint_type", "workflow_orchestration"),
			zap.String("path", c.Request.URL.Path),
		)

		// Set enhanced logger in context
		c.Set("logger", contextLogger)

		c.Next()

		// Record workflow endpoint metrics
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// Log workflow request completion
		logLevel := zap.InfoLevel
		if statusCode >= 400 {
			logLevel = zap.WarnLevel
		}
		if statusCode >= 500 {
			logLevel = zap.ErrorLevel
		}

		contextLogger.Log(logLevel, "Workflow endpoint request completed",
			zap.Duration("duration", duration),
			zap.Int("status_code", statusCode),
			zap.String("correlation_id", c.GetHeader("X-Correlation-ID")))
	}
}

// ClinicalAuditMiddleware provides enhanced logging for clinical operations
func (o *ObservabilityMiddleware) ClinicalAuditMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to clinical endpoints
		if !isClinicalEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		start := time.Now()
		
		// Enhanced audit logging for clinical operations
		auditLogger := o.logger.With(
			zap.String("audit_type", "clinical_operation"),
			zap.String("endpoint", c.Request.URL.Path),
			zap.String("method", c.Request.Method),
			zap.Time("timestamp", start),
		)

		// Add user context for audit
		if user, exists := c.Get("user"); exists {
			if userContext, ok := user.(*UserContext); ok {
				auditLogger = auditLogger.With(
					zap.String("user_id", userContext.UserID),
					zap.String("provider_id", userContext.ProviderID),
					zap.String("user_email", userContext.Email),
					zap.Strings("user_roles", userContext.Roles),
				)
			}
		}

		// Add to context for use in handlers
		c.Set("audit_logger", auditLogger)

		auditLogger.Info("Clinical operation initiated")

		c.Next()

		// Log completion
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		auditLogger.Info("Clinical operation completed",
			zap.Duration("duration", duration),
			zap.Int("status_code", statusCode),
			zap.Bool("successful", statusCode < 400))
	}
}

// ErrorTrackingMiddleware captures and reports errors
func (o *ObservabilityMiddleware) ErrorTrackingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for errors
		statusCode := c.Writer.Status()
		if statusCode >= 400 {
			// Record error metrics
			errorType := "client_error"
			if statusCode >= 500 {
				errorType = "server_error"
			}

			path := c.Request.URL.Path
			component := "http_handler"

			if isWorkflowEndpoint(path) {
				component = "workflow_orchestration"
			} else if isClinicalEndpoint(path) {
				component = "clinical_operation"
			}

			o.metrics.WorkflowErrors.WithLabelValues(errorType, "api", component).Inc()

			// Enhanced error logging
			o.logger.Error("Request failed",
				zap.Int("status_code", statusCode),
				zap.String("method", c.Request.Method),
				zap.String("path", path),
				zap.String("error_type", errorType),
				zap.String("component", component),
				zap.String("remote_addr", c.ClientIP()),
				zap.String("user_agent", c.GetHeader("User-Agent")),
				zap.String("correlation_id", c.GetHeader("X-Correlation-ID")))
		}
	}
}

// GraphQLObservabilityMiddleware adds observability for GraphQL operations
func (o *ObservabilityMiddleware) GraphQLObservabilityMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Only apply to GraphQL endpoints
		if c.Request.URL.Path != "/graphql" {
			c.Next()
			return
		}

		start := time.Now()

		// Enhanced GraphQL logging
		gqlLogger := o.logger.With(
			zap.String("component", "graphql"),
			zap.String("path", c.Request.URL.Path),
		)

		c.Set("graphql_logger", gqlLogger)

		c.Next()

		// Record GraphQL metrics
		duration := time.Since(start)
		statusCode := c.Writer.Status()

		// This would need to be enhanced to extract actual GraphQL operation details
		operationType := "query" // Would be extracted from request body
		operationName := "unknown" // Would be extracted from request body

		o.metrics.GraphQLOperations.WithLabelValues(
			operationType, 
			operationName, 
			strconv.Itoa(statusCode),
		).Inc()

		gqlLogger.Info("GraphQL operation completed",
			zap.String("operation_type", operationType),
			zap.String("operation_name", operationName),
			zap.Duration("duration", duration),
			zap.Int("status_code", statusCode))
	}
}

// Helper functions

func isWorkflowEndpoint(path string) bool {
	return len(path) >= 17 && path[:17] == "/api/v1/orchestration" ||
		   len(path) >= 13 && path[:13] == "/api/v1/workflows"
}

func isClinicalEndpoint(path string) bool {
	return isWorkflowEndpoint(path) ||
		   len(path) >= 12 && path[:12] == "/api/v1/patients" ||
		   len(path) >= 12 && path[:12] == "/api/v1/snapshots"
}

// GetLoggerFromContext extracts the enhanced logger from context
func GetLoggerFromContext(c *gin.Context, fallback *zap.Logger) *zap.Logger {
	if logger, exists := c.Get("logger"); exists {
		if contextLogger, ok := logger.(*zap.Logger); ok {
			return contextLogger
		}
	}
	return fallback
}

// GetAuditLoggerFromContext extracts the audit logger from context
func GetAuditLoggerFromContext(c *gin.Context, fallback *zap.Logger) *zap.Logger {
	if logger, exists := c.Get("audit_logger"); exists {
		if auditLogger, ok := logger.(*zap.Logger); ok {
			return auditLogger
		}
	}
	return fallback
}