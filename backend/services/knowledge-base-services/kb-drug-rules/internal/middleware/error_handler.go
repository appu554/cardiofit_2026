package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"
	"time"

	"kb-drug-rules/internal/errors"
	"kb-drug-rules/internal/metrics"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ErrorHandlerMiddleware provides comprehensive error handling for TOML operations
func ErrorHandlerMiddleware(tomlMetrics *metrics.TOMLMetrics) gin.HandlerFunc {
	return gin.CustomRecovery(func(c *gin.Context, recovered interface{}) {
		// Generate request ID if not present
		requestID := c.GetString("request_id")
		if requestID == "" {
			requestID = uuid.New().String()
			c.Set("request_id", requestID)
		}

		// Record panic metrics
		tomlMetrics.RecordError("panic", c.Request.Method)

		var err *errors.TOMLError

		// Handle different types of recovered errors
		switch v := recovered.(type) {
		case *errors.TOMLError:
			err = v
		case error:
			err = &errors.TOMLError{
				Code:       "INTERNAL_SERVER_ERROR",
				Message:    v.Error(),
				HTTPStatus: http.StatusInternalServerError,
			}
		case string:
			err = &errors.TOMLError{
				Code:       "INTERNAL_SERVER_ERROR",
				Message:    v,
				HTTPStatus: http.StatusInternalServerError,
			}
		default:
			err = &errors.TOMLError{
				Code:       "UNKNOWN_ERROR",
				Message:    fmt.Sprintf("Unknown error occurred: %v", v),
				HTTPStatus: http.StatusInternalServerError,
			}
		}

		// Add stack trace for internal errors
		if err.HTTPStatus >= 500 {
			err.Details = map[string]interface{}{
				"stack_trace": string(debug.Stack()),
			}
		}

		// Create error response
		errorResponse := errors.NewErrorResponse(
			err,
			requestID,
			c.Request.URL.Path,
			c.Request.Method,
		)

		// Log error details
		logError(c, err, errorResponse)

		// Send error response
		c.JSON(err.HTTPStatus, errorResponse)
		c.Abort()
	})
}

// TOMLErrorHandler handles TOML-specific errors
func TOMLErrorHandler(tomlMetrics *metrics.TOMLMetrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check if there are any errors
		if len(c.Errors) > 0 {
			// Get the last error
			lastError := c.Errors.Last()
			
			// Generate request ID if not present
			requestID := c.GetString("request_id")
			if requestID == "" {
				requestID = uuid.New().String()
				c.Set("request_id", requestID)
			}

			var tomlErr *errors.TOMLError

			// Convert error to TOMLError
			switch err := lastError.Err.(type) {
			case *errors.TOMLError:
				tomlErr = err
			default:
				tomlErr = &errors.TOMLError{
					Code:       "GENERIC_ERROR",
					Message:    err.Error(),
					HTTPStatus: http.StatusInternalServerError,
				}
			}

			// Record error metrics
			tomlMetrics.RecordError(tomlErr.Code, c.Request.Method)

			// Create error response
			errorResponse := errors.NewErrorResponse(
				tomlErr,
				requestID,
				c.Request.URL.Path,
				c.Request.Method,
			)

			// Log error
			logError(c, tomlErr, errorResponse)

			// Send error response if not already sent
			if !c.Writer.Written() {
				c.JSON(tomlErr.HTTPStatus, errorResponse)
			}
		}
	}
}

// ValidationErrorHandler handles validation errors specifically
func ValidationErrorHandler() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Check for validation errors in context
		if validationErrors, exists := c.Get("validation_errors"); exists {
			if errCollection, ok := validationErrors.(*errors.ValidationErrorCollection); ok {
				requestID := c.GetString("request_id")
				if requestID == "" {
					requestID = uuid.New().String()
				}

				// Create comprehensive validation error response
				response := map[string]interface{}{
					"success":    false,
					"request_id": requestID,
					"timestamp":  time.Now().Unix(),
					"path":       c.Request.URL.Path,
					"method":     c.Request.Method,
					"validation": map[string]interface{}{
						"errors":        errCollection.Errors,
						"warnings":      errCollection.Warnings,
						"error_count":   len(errCollection.Errors),
						"warning_count": len(errCollection.Warnings),
						"has_errors":    errCollection.HasErrors(),
						"has_warnings":  errCollection.HasWarnings(),
					},
				}

				statusCode := http.StatusBadRequest
				if !errCollection.HasErrors() {
					statusCode = http.StatusOK
				}

				c.JSON(statusCode, response)
				c.Abort()
			}
		}
	}
}

// RequestMetricsMiddleware tracks request metrics
func RequestMetricsMiddleware(tomlMetrics *metrics.TOMLMetrics) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// Increment active requests
		tomlMetrics.IncrementActiveRequests()
		
		// Process request
		c.Next()
		
		// Decrement active requests
		tomlMetrics.DecrementActiveRequests()
		
		// Record response time
		duration := time.Since(start)
		tomlMetrics.RecordResponseTime(duration)
		
		// Record operation-specific metrics
		path := c.Request.URL.Path
		method := c.Request.Method
		
		switch {
		case path == "/v1/validate" && method == "POST":
			success := c.Writer.Status() < 400
			// Note: validation score would need to be extracted from response
			tomlMetrics.RecordValidation(duration, success, 0) // Score would be set elsewhere
			
		case path == "/v1/convert" && method == "POST":
			success := c.Writer.Status() < 400
			tomlMetrics.RecordConversion(duration, success, 0, 0) // Sizes would be set elsewhere
			
		case path == "/v1/hotload" && method == "POST":
			success := c.Writer.Status() < 400
			tomlMetrics.RecordHotload(duration, success)
			
		case path == "/v1/batch-load" && method == "POST":
			success := c.Writer.Status() < 400
			tomlMetrics.RecordBatchLoad(duration, success, 0) // Item count would be set elsewhere
		}
	}
}

// RequestIDMiddleware adds a unique request ID to each request
func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Check if request ID is already present in headers
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = uuid.New().String()
		}
		
		// Set request ID in context and response header
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)
		
		c.Next()
	}
}

// CORSMiddleware handles CORS for TOML operations
func CORSMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Content-Length, Accept-Encoding, X-CSRF-Token, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")
		
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}
		
		c.Next()
	}
}

// SecurityHeadersMiddleware adds security headers
func SecurityHeadersMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("X-Frame-Options", "DENY")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Content-Security-Policy", "default-src 'self'")
		
		c.Next()
	}
}

// logError logs error details for monitoring and debugging
func logError(c *gin.Context, err *errors.TOMLError, response *errors.ErrorResponse) {
	// This would integrate with your logging system
	// For now, we'll use a simple log format
	
	logLevel := "ERROR"
	if err.HTTPStatus < 500 {
		logLevel = "WARN"
	}
	
	logData := map[string]interface{}{
		"level":      logLevel,
		"timestamp":  time.Now().Unix(),
		"request_id": response.RequestID,
		"method":     c.Request.Method,
		"path":       c.Request.URL.Path,
		"status":     err.HTTPStatus,
		"error_code": err.Code,
		"message":    err.Message,
		"user_agent": c.Request.UserAgent(),
		"ip":         c.ClientIP(),
	}
	
	if err.Details != nil {
		logData["details"] = err.Details
	}
	
	if err.Line > 0 {
		logData["line"] = err.Line
		logData["column"] = err.Column
	}
	
	// In a real implementation, this would use structured logging
	fmt.Printf("[%s] %s %s - %s: %s (Request ID: %s)\n", 
		logLevel, 
		c.Request.Method, 
		c.Request.URL.Path, 
		err.Code, 
		err.Message, 
		response.RequestID,
	)
}

// RateLimitMiddleware provides basic rate limiting
func RateLimitMiddleware(requestsPerMinute int) gin.HandlerFunc {
	// This is a simplified rate limiter
	// In production, you'd use Redis or a more sophisticated solution
	
	return func(c *gin.Context) {
		// Rate limiting logic would go here
		// For now, just pass through
		c.Next()
	}
}

// TimeoutMiddleware adds request timeout handling
func TimeoutMiddleware(timeout time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		// Set timeout context
		// This would be implemented with context.WithTimeout
		c.Next()
	}
}
