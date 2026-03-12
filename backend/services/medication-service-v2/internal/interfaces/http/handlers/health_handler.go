package handlers

import (
	"net/http"
	"time"

	"medication-service-v2/internal/application/services"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
)

// HealthHandler handles health check endpoints
type HealthHandler struct {
	service *services.HealthService
	logger  *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(service *services.HealthService, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		service: service,
		logger:  logger,
	}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string                    `json:"status"`
	Timestamp time.Time                 `json:"timestamp"`
	Services  map[string]ServiceHealth  `json:"services,omitempty"`
	Version   string                    `json:"version,omitempty"`
	Uptime    string                    `json:"uptime,omitempty"`
}

// ServiceHealth represents the health status of a service component
type ServiceHealth struct {
	Name           string    `json:"name"`
	Status         string    `json:"status"`
	Message        string    `json:"message,omitempty"`
	LastCheck      time.Time `json:"last_check"`
	ResponseTimeMs int64     `json:"response_time_ms"`
}

// HealthCheck performs a basic health check
// @Summary Health check
// @Description Returns the health status of the service
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health [get]
func (h *HealthHandler) HealthCheck(c *gin.Context) {
	deepCheck := c.Query("deep") == "true"
	
	health := h.service.CheckHealth(c.Request.Context(), deepCheck)
	
	// Convert service health
	services := make(map[string]ServiceHealth)
	for name, serviceHealth := range health.ServiceHealth {
		services[name] = ServiceHealth{
			Name:           serviceHealth.Name,
			Status:         serviceHealth.Status,
			Message:        serviceHealth.Message,
			LastCheck:      serviceHealth.LastCheck,
			ResponseTimeMs: serviceHealth.ResponseTimeMs,
		}
	}
	
	response := HealthResponse{
		Status:    health.Status,
		Timestamp: time.Now(),
		Services:  services,
		Version:   "2.0.0", // This would come from build info
	}
	
	// Set appropriate HTTP status code
	statusCode := http.StatusOK
	if health.Status != "healthy" {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, response)
}

// ReadinessCheck checks if the service is ready to accept traffic
// @Summary Readiness check
// @Description Returns whether the service is ready to accept traffic
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Failure 503 {object} HealthResponse
// @Router /health/ready [get]
func (h *HealthHandler) ReadinessCheck(c *gin.Context) {
	health := h.service.CheckReadiness(c.Request.Context())
	
	response := HealthResponse{
		Status:    health.Status,
		Timestamp: time.Now(),
	}
	
	statusCode := http.StatusOK
	if health.Status != "ready" {
		statusCode = http.StatusServiceUnavailable
	}
	
	c.JSON(statusCode, response)
}

// LivenessCheck checks if the service is alive
// @Summary Liveness check
// @Description Returns whether the service is alive (basic ping)
// @Tags health
// @Produce json
// @Success 200 {object} HealthResponse
// @Router /health/live [get]
func (h *HealthHandler) LivenessCheck(c *gin.Context) {
	response := HealthResponse{
		Status:    "alive",
		Timestamp: time.Now(),
	}
	
	c.JSON(http.StatusOK, response)
}