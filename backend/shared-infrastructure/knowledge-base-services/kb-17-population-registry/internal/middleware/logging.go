// Package middleware provides HTTP middleware for KB-17 Population Registry
package middleware

import (
	"bytes"
	"io"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
)

// LoggingConfig holds logging configuration
type LoggingConfig struct {
	SkipPaths       []string
	LogRequestBody  bool
	LogResponseBody bool
	MaxBodyLogSize  int
}

// DefaultLoggingConfig returns default logging configuration
func DefaultLoggingConfig() *LoggingConfig {
	return &LoggingConfig{
		SkipPaths:       []string{"/health", "/ready", "/metrics"},
		LogRequestBody:  false, // Disable by default for PHI/PII concerns
		LogResponseBody: false,
		MaxBodyLogSize:  1024, // 1KB max for body logging
	}
}

// LoggingMiddleware creates request logging middleware
func LoggingMiddleware(config *LoggingConfig, logger *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Generate request ID
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		// Skip logging for certain paths
		path := c.Request.URL.Path
		for _, skipPath := range config.SkipPaths {
			if path == skipPath {
				c.Next()
				return
			}
		}

		startTime := time.Now()

		// Create request-scoped logger
		reqLogger := logger.WithFields(logrus.Fields{
			"request_id": requestID,
			"method":     c.Request.Method,
			"path":       path,
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		// Log request body if enabled
		var requestBody []byte
		if config.LogRequestBody && c.Request.Body != nil {
			requestBody, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(requestBody))

			if len(requestBody) > 0 && len(requestBody) <= config.MaxBodyLogSize {
				reqLogger = reqLogger.WithField("request_body", string(requestBody))
			}
		}

		// Wrap response writer to capture response
		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		if config.LogResponseBody {
			c.Writer = blw
		}

		// Log incoming request
		reqLogger.Info("Request received")

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(startTime)

		// Build response log fields
		responseFields := logrus.Fields{
			"status":      c.Writer.Status(),
			"duration_ms": duration.Milliseconds(),
			"size":        c.Writer.Size(),
		}

		// Log response body if enabled
		if config.LogResponseBody && blw.body.Len() > 0 && blw.body.Len() <= config.MaxBodyLogSize {
			responseFields["response_body"] = blw.body.String()
		}

		// Log any errors
		if len(c.Errors) > 0 {
			responseFields["errors"] = c.Errors.String()
		}

		// Log response
		responseLogger := reqLogger.WithFields(responseFields)

		status := c.Writer.Status()
		switch {
		case status >= 500:
			responseLogger.Error("Request completed with server error")
		case status >= 400:
			responseLogger.Warn("Request completed with client error")
		default:
			responseLogger.Info("Request completed successfully")
		}
	}
}

// bodyLogWriter wraps gin.ResponseWriter to capture response body
type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

// AuditLogMiddleware creates HIPAA-compliant audit logging middleware
func AuditLogMiddleware(logger *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Skip non-data-access endpoints
		if !isDataAccessEndpoint(c.Request.URL.Path) {
			c.Next()
			return
		}

		startTime := time.Now()

		// Capture pre-request state
		auditEntry := &AuditLogEntry{
			Timestamp:   startTime,
			RequestID:   c.GetString("request_id"),
			UserID:      c.GetString("user_id"),
			ServiceName: c.GetString("service_name"),
			Method:      c.Request.Method,
			Path:        c.Request.URL.Path,
			ClientIP:    c.ClientIP(),
		}

		// Process request
		c.Next()

		// Complete audit entry
		auditEntry.Duration = time.Since(startTime)
		auditEntry.StatusCode = c.Writer.Status()
		auditEntry.Success = c.Writer.Status() < 400

		// Extract patient ID if present
		if patientID := c.Param("patient_id"); patientID != "" {
			auditEntry.PatientID = patientID
		}

		// Log audit entry
		auditLogger := logger.WithFields(logrus.Fields{
			"audit":        true,
			"request_id":   auditEntry.RequestID,
			"user_id":      auditEntry.UserID,
			"service_name": auditEntry.ServiceName,
			"patient_id":   auditEntry.PatientID,
			"method":       auditEntry.Method,
			"path":         auditEntry.Path,
			"status_code":  auditEntry.StatusCode,
			"success":      auditEntry.Success,
			"duration_ms":  auditEntry.Duration.Milliseconds(),
		})

		if auditEntry.Success {
			auditLogger.Info("Audit: Data access completed")
		} else {
			auditLogger.Warn("Audit: Data access failed")
		}
	}
}

// AuditLogEntry represents an audit log entry
type AuditLogEntry struct {
	Timestamp   time.Time
	RequestID   string
	UserID      string
	ServiceName string
	PatientID   string
	Method      string
	Path        string
	ClientIP    string
	StatusCode  int
	Success     bool
	Duration    time.Duration
}

// isDataAccessEndpoint checks if path accesses patient data
func isDataAccessEndpoint(path string) bool {
	dataAccessPatterns := []string{
		"/api/v1/patients",
		"/api/v1/enrollments",
		"/api/v1/evaluate",
	}

	for _, pattern := range dataAccessPatterns {
		if len(path) >= len(pattern) && path[:len(pattern)] == pattern {
			return true
		}
	}
	return false
}

// CorrelationMiddleware adds correlation IDs for distributed tracing
func CorrelationMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check for incoming correlation ID
		correlationID := c.GetHeader("X-Correlation-ID")
		if correlationID == "" {
			correlationID = uuid.New().String()
		}

		// Set correlation ID
		c.Set("correlation_id", correlationID)
		c.Header("X-Correlation-ID", correlationID)

		// Check for span ID
		spanID := c.GetHeader("X-Span-ID")
		if spanID == "" {
			spanID = uuid.New().String()[:8]
		}
		c.Set("span_id", spanID)
		c.Header("X-Span-ID", spanID)

		c.Next()
	}
}
