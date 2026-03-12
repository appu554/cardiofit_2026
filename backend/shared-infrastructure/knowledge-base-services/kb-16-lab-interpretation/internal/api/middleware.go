// Package api provides HTTP API handlers and middleware for KB-16
package api

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/sirupsen/logrus"
)

// Prometheus metrics
var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "kb16_http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "endpoint", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "kb16_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "endpoint"},
	)

	criticalValuesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kb16_critical_values_total",
			Help: "Total number of critical lab values detected",
		},
	)

	panicValuesTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kb16_panic_values_total",
			Help: "Total number of panic lab values detected",
		},
	)

	interpretationsTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "kb16_interpretations_total",
			Help: "Total number of lab result interpretations",
		},
	)
)

func init() {
	// Register metrics
	prometheus.MustRegister(
		httpRequestsTotal,
		httpRequestDuration,
		criticalValuesTotal,
		panicValuesTotal,
		interpretationsTotal,
	)
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
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

// LoggingMiddleware logs HTTP requests
func LoggingMiddleware(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		entry := log.WithFields(logrus.Fields{
			"request_id": c.GetString("request_id"),
			"method":     c.Request.Method,
			"path":       path,
			"query":      query,
			"status":     status,
			"latency_ms": latency.Milliseconds(),
			"client_ip":  c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
		})

		if len(c.Errors) > 0 {
			entry.Error(c.Errors.String())
		} else if status >= 500 {
			entry.Error("Server error")
		} else if status >= 400 {
			entry.Warn("Client error")
		} else {
			entry.Info("Request completed")
		}
	}
}

// MetricsMiddleware records Prometheus metrics for requests
func MetricsMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		c.Next()

		status := c.Writer.Status()
		duration := time.Since(start).Seconds()

		httpRequestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			http.StatusText(status),
		).Inc()

		httpRequestDuration.WithLabelValues(
			c.Request.Method,
			path,
		).Observe(duration)
	}
}

// RecoveryMiddleware recovers from panics and logs them
func RecoveryMiddleware(log *logrus.Entry) gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				log.WithFields(logrus.Fields{
					"request_id": c.GetString("request_id"),
					"error":      err,
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
				}).Error("Panic recovered")

				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"success": false,
					"error": gin.H{
						"code":    "INTERNAL_ERROR",
						"message": "An internal error occurred",
					},
				})
			}
		}()
		c.Next()
	}
}

// CORSMiddleware handles CORS headers
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Accept, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		c.Header("Access-Control-Max-Age", "86400")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// TimeoutMiddleware adds request timeout
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set deadline on context
		// Note: actual timeout handling should be done in handlers
		c.Set("timeout", timeout)
		c.Next()
	}
}

// ClientServiceMiddleware extracts client service identifier
func ClientServiceMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		clientService := c.GetHeader("X-Client-Service")
		if clientService != "" {
			c.Set("client_service", clientService)
		}
		c.Next()
	}
}

// RecordCriticalValue records a critical value metric
func RecordCriticalValue() {
	criticalValuesTotal.Inc()
}

// RecordPanicValue records a panic value metric
func RecordPanicValue() {
	panicValuesTotal.Inc()
}

// RecordInterpretation records an interpretation metric
func RecordInterpretation() {
	interpretationsTotal.Inc()
}
