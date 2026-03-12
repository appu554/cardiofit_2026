package http

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/config"
	"global-outbox-service-go/internal/services"
)

// RegistryEndpoints provides HTTP endpoints for service registration and management
type RegistryEndpoints struct {
	registry *services.ServiceRegistry
	logger   *logrus.Logger
	config   *config.PlatformConfig
}

// NewRegistryEndpoints creates new registry endpoints
func NewRegistryEndpoints(registry *services.ServiceRegistry, config *config.PlatformConfig, logger *logrus.Logger) *RegistryEndpoints {
	return &RegistryEndpoints{
		registry: registry,
		logger:   logger,
		config:   config,
	}
}

// RegisterRoutes registers all service registry routes
func (re *RegistryEndpoints) RegisterRoutes(app *fiber.App) {
	// Service registry management
	registry := app.Group("/registry")
	
	registry.Post("/services", re.registerService)
	registry.Delete("/services/:serviceName", re.unregisterService)
	registry.Put("/services/:serviceName/heartbeat", re.updateHeartbeat)
	registry.Get("/services", re.listServices)
	registry.Get("/services/:serviceName", re.getService)
	registry.Get("/services/:serviceName/health", re.checkServiceHealth)
	
	// Service discovery and management
	discovery := app.Group("/discovery")
	
	discovery.Post("/scan", re.scanForServices)
	discovery.Get("/tables", re.listOutboxTables)
	discovery.Post("/services/:serviceName/table", re.createServiceTable)
}

// registerService handles service registration
func (re *RegistryEndpoints) registerService(c *fiber.Ctx) error {
	var req ServiceRegistrationRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid request body",
			"detail": err.Error(),
		})
	}

	// Validate request
	if err := re.validateRegistrationRequest(&req); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "Invalid registration request",
			"detail": err.Error(),
		})
	}

	// Create service registration
	registration := &config.ServiceRegistration{
		Name:           req.ServiceName,
		DatabaseURL:    req.DatabaseURL,
		Priority:       req.Priority,
		HealthcheckURL: req.HealthcheckURL,
		Metadata:       req.Metadata,
	}

	// Register the service
	if err := re.registry.RegisterService(c.Context(), registration); err != nil {
		re.logger.Errorf("Failed to register service %s: %v", req.ServiceName, err)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to register service",
			"detail": err.Error(),
		})
	}

	re.logger.Infof("Service registered via API: %s", req.ServiceName)
	
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"message": "Service registered successfully",
		"service": registration,
	})
}

// unregisterService handles service unregistration
func (re *RegistryEndpoints) unregisterService(c *fiber.Ctx) error {
	serviceName := c.Params("serviceName")
	
	if err := re.registry.UnregisterService(c.Context(), serviceName); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Service not found",
				"service": serviceName,
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to unregister service",
			"detail": err.Error(),
		})
	}

	re.logger.Infof("Service unregistered via API: %s", serviceName)
	
	return c.JSON(fiber.Map{
		"message": "Service unregistered successfully",
		"service": serviceName,
	})
}

// updateHeartbeat updates service heartbeat
func (re *RegistryEndpoints) updateHeartbeat(c *fiber.Ctx) error {
	serviceName := c.Params("serviceName")
	
	if err := re.registry.UpdateServiceHeartbeat(c.Context(), serviceName); err != nil {
		if strings.Contains(err.Error(), "not found") {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": "Service not found",
				"service": serviceName,
			})
		}
		
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "Failed to update heartbeat",
			"detail": err.Error(),
		})
	}

	return c.JSON(fiber.Map{
		"message": "Heartbeat updated successfully",
		"service": serviceName,
		"timestamp": time.Now(),
	})
}

// listServices lists all registered services
func (re *RegistryEndpoints) listServices(c *fiber.Ctx) error {
	activeOnly := c.Query("active") == "true"
	
	var services map[string]*config.ServiceRegistration
	
	if activeOnly {
		services = re.registry.GetActiveServices()
	} else {
		services = re.registry.GetRegisteredServices()
	}

	// Convert to response format
	var serviceList []ServiceInfo
	for _, service := range services {
		serviceList = append(serviceList, ServiceInfo{
			Name:           service.Name,
			Status:         service.Status,
			Priority:       service.Priority,
			HealthcheckURL: service.HealthcheckURL,
			CreatedAt:      service.CreatedAt,
			LastSeen:       service.LastSeen,
			Metadata:       service.Metadata,
		})
	}

	return c.JSON(fiber.Map{
		"total_services": len(serviceList),
		"active_only": activeOnly,
		"services": serviceList,
	})
}

// getService gets information about a specific service
func (re *RegistryEndpoints) getService(c *fiber.Ctx) error {
	serviceName := c.Params("serviceName")
	
	service, exists := re.registry.GetServiceRegistration(serviceName)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
			"service": serviceName,
		})
	}

	// Get additional service information
	serviceInfo := ServiceInfo{
		Name:           service.Name,
		Status:         service.Status,
		Priority:       service.Priority,
		HealthcheckURL: service.HealthcheckURL,
		CreatedAt:      service.CreatedAt,
		LastSeen:       service.LastSeen,
		Metadata:       service.Metadata,
	}

	// Add configuration overrides if available
	if override := re.config.GetServiceOverride(serviceName); override != nil {
		serviceInfo.ConfigOverrides = map[string]interface{}{
			"poll_interval": override.PollInterval,
			"batch_size": override.BatchSize,
			"max_workers": override.MaxWorkers,
			"circuit_breaker": override.CircuitBreaker,
			"priority": override.Priority,
			"topic_prefix": override.CustomTopicPrefix,
		}
	}

	return c.JSON(serviceInfo)
}

// checkServiceHealth checks the health of a specific service
func (re *RegistryEndpoints) checkServiceHealth(c *fiber.Ctx) error {
	serviceName := c.Params("serviceName")
	
	service, exists := re.registry.GetServiceRegistration(serviceName)
	if !exists {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Service not found",
			"service": serviceName,
		})
	}

	// Calculate health status
	healthy := re.isServiceHealthy(service)
	
	healthStatus := ServiceHealthStatus{
		ServiceName: serviceName,
		Healthy:     healthy,
		Status:      service.Status,
		LastSeen:    service.LastSeen,
		TimeSinceLastSeen: time.Since(service.LastSeen),
	}

	if service.HealthcheckURL != "" {
		// TODO: Implement actual HTTP health check
		healthStatus.HealthcheckURL = service.HealthcheckURL
		healthStatus.HealthcheckStatus = "not_implemented"
	}

	statusCode := fiber.StatusOK
	if !healthy {
		statusCode = fiber.StatusServiceUnavailable
	}

	return c.Status(statusCode).JSON(healthStatus)
}

// scanForServices manually triggers service discovery
func (re *RegistryEndpoints) scanForServices(c *fiber.Ctx) error {
	// This would trigger the auto-discovery process
	// For now, we'll return a placeholder response
	
	re.logger.Info("Manual service discovery scan triggered via API")
	
	return c.JSON(fiber.Map{
		"message": "Service discovery scan initiated",
		"timestamp": time.Now(),
		"note": "Check logs for discovery results",
	})
}

// listOutboxTables lists all outbox tables in the database
func (re *RegistryEndpoints) listOutboxTables(c *fiber.Ctx) error {
	// This would query the database for outbox tables
	// For now, return a placeholder
	
	return c.JSON(fiber.Map{
		"message": "Outbox table listing not yet implemented",
		"note": "This will show all outbox_events_* tables",
	})
}

// createServiceTable creates an outbox table for a service
func (re *RegistryEndpoints) createServiceTable(c *fiber.Ctx) error {
	serviceName := c.Params("serviceName")
	
	// This would create the outbox table for the service
	re.logger.Infof("Request to create outbox table for service: %s", serviceName)
	
	return c.JSON(fiber.Map{
		"message": fmt.Sprintf("Outbox table creation for service %s not yet implemented", serviceName),
		"service": serviceName,
	})
}

// Helper methods and types

// ServiceRegistrationRequest represents a service registration request
type ServiceRegistrationRequest struct {
	ServiceName    string            `json:"service_name" validate:"required"`
	DatabaseURL    string            `json:"database_url" validate:"required"`
	Priority       int               `json:"priority" validate:"min=1,max=10"`
	HealthcheckURL string            `json:"healthcheck_url,omitempty"`
	Metadata       map[string]string `json:"metadata,omitempty"`
}

// ServiceInfo represents service information in API responses
type ServiceInfo struct {
	Name            string                 `json:"name"`
	Status          string                 `json:"status"`
	Priority        int                    `json:"priority"`
	HealthcheckURL  string                 `json:"healthcheck_url,omitempty"`
	CreatedAt       time.Time              `json:"created_at"`
	LastSeen        time.Time              `json:"last_seen"`
	Metadata        map[string]string      `json:"metadata,omitempty"`
	ConfigOverrides map[string]interface{} `json:"config_overrides,omitempty"`
}

// ServiceHealthStatus represents service health status
type ServiceHealthStatus struct {
	ServiceName         string        `json:"service_name"`
	Healthy             bool          `json:"healthy"`
	Status              string        `json:"status"`
	LastSeen            time.Time     `json:"last_seen"`
	TimeSinceLastSeen   time.Duration `json:"time_since_last_seen"`
	HealthcheckURL      string        `json:"healthcheck_url,omitempty"`
	HealthcheckStatus   string        `json:"healthcheck_status,omitempty"`
}

func (re *RegistryEndpoints) validateRegistrationRequest(req *ServiceRegistrationRequest) error {
	if req.ServiceName == "" {
		return fmt.Errorf("service_name is required")
	}
	
	if req.DatabaseURL == "" {
		return fmt.Errorf("database_url is required")
	}
	
	if req.Priority < 1 || req.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	
	// Validate service name format
	if strings.Contains(req.ServiceName, " ") || strings.Contains(req.ServiceName, "_") {
		return fmt.Errorf("service_name should use kebab-case (e.g., 'patient-service')")
	}
	
	return nil
}

func (re *RegistryEndpoints) isServiceHealthy(service *config.ServiceRegistration) bool {
	// Consider a service healthy if it was seen recently
	ttl := re.config.ServiceRegistry.RegistrationTTL
	return time.Since(service.LastSeen) < ttl && service.Status == "active"
}