package health

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/types"
)

// HealthStatus represents the health status of a component
type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
	HealthStatusUnknown   HealthStatus = "unknown"
)

// ComponentHealth represents the health of a single component
type ComponentHealth struct {
	Name         string                 `json:"name"`
	Status       HealthStatus           `json:"status"`
	LastChecked  time.Time              `json:"last_checked"`
	ResponseTime time.Duration          `json:"response_time"`
	Error        string                 `json:"error,omitempty"`
	Details      map[string]interface{} `json:"details,omitempty"`
	Version      string                 `json:"version,omitempty"`
}

// SystemHealth represents the overall system health
type SystemHealth struct {
	Status      HealthStatus                `json:"status"`
	Timestamp   time.Time                   `json:"timestamp"`
	Version     string                      `json:"version"`
	Uptime      time.Duration               `json:"uptime"`
	Components  map[string]ComponentHealth  `json:"components"`
	Summary     HealthSummary               `json:"summary"`
}

// HealthSummary provides a summary of system health
type HealthSummary struct {
	TotalComponents   int `json:"total_components"`
	HealthyComponents int `json:"healthy_components"`
	DegradedComponents int `json:"degraded_components"`
	UnhealthyComponents int `json:"unhealthy_components"`
	UnknownComponents int `json:"unknown_components"`
}

// HealthChecker interface for components that can be health checked
type HealthChecker interface {
	HealthCheck() error
	Name() string
}

// HealthManager manages health checks for all system components
type HealthManager struct {
	components    map[string]HealthChecker
	componentHealth map[string]ComponentHealth
	mutex         sync.RWMutex
	logger        *zap.Logger
	startTime     time.Time
	version       string
	
	// Configuration
	checkInterval time.Duration
	timeout       time.Duration
	
	// Background checking
	stopChan      chan struct{}
	running       bool
}

// NewHealthManager creates a new health manager
func NewHealthManager(logger *zap.Logger, version string) *HealthManager {
	return &HealthManager{
		components:      make(map[string]HealthChecker),
		componentHealth: make(map[string]ComponentHealth),
		logger:          logger,
		startTime:       time.Now(),
		version:         version,
		checkInterval:   30 * time.Second,
		timeout:         5 * time.Second,
		stopChan:        make(chan struct{}),
	}
}

// RegisterComponent registers a component for health checking
func (hm *HealthManager) RegisterComponent(component HealthChecker) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	
	name := component.Name()
	hm.components[name] = component
	hm.componentHealth[name] = ComponentHealth{
		Name:        name,
		Status:      HealthStatusUnknown,
		LastChecked: time.Time{},
	}
	
	hm.logger.Info("Registered health check component", zap.String("component", name))
}

// UnregisterComponent unregisters a component
func (hm *HealthManager) UnregisterComponent(name string) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	
	delete(hm.components, name)
	delete(hm.componentHealth, name)
	
	hm.logger.Info("Unregistered health check component", zap.String("component", name))
}

// CheckComponent performs a health check on a specific component
func (hm *HealthManager) CheckComponent(name string) ComponentHealth {
	hm.mutex.RLock()
	component, exists := hm.components[name]
	hm.mutex.RUnlock()
	
	if !exists {
		return ComponentHealth{
			Name:        name,
			Status:      HealthStatusUnknown,
			LastChecked: time.Now(),
			Error:       "Component not found",
		}
	}
	
	return hm.performHealthCheck(component)
}

// CheckAllComponents performs health checks on all registered components
func (hm *HealthManager) CheckAllComponents() map[string]ComponentHealth {
	hm.mutex.RLock()
	components := make(map[string]HealthChecker)
	for name, component := range hm.components {
		components[name] = component
	}
	hm.mutex.RUnlock()
	
	results := make(map[string]ComponentHealth)
	var wg sync.WaitGroup
	var resultMutex sync.Mutex
	
	for name, component := range components {
		wg.Add(1)
		go func(n string, c HealthChecker) {
			defer wg.Done()
			result := hm.performHealthCheck(c)
			
			resultMutex.Lock()
			results[n] = result
			hm.mutex.Lock()
			hm.componentHealth[n] = result
			hm.mutex.Unlock()
			resultMutex.Unlock()
		}(name, component)
	}
	
	wg.Wait()
	return results
}

// performHealthCheck performs a health check on a single component
func (hm *HealthManager) performHealthCheck(component HealthChecker) ComponentHealth {
	startTime := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), hm.timeout)
	defer cancel()
	
	// Create a channel to receive the health check result
	resultChan := make(chan error, 1)
	
	// Run health check in a goroutine
	go func() {
		resultChan <- component.HealthCheck()
	}()
	
	var err error
	select {
	case err = <-resultChan:
		// Health check completed
	case <-ctx.Done():
		// Health check timed out
		err = fmt.Errorf("health check timed out after %v", hm.timeout)
	}
	
	responseTime := time.Since(startTime)
	
	health := ComponentHealth{
		Name:         component.Name(),
		LastChecked:  time.Now(),
		ResponseTime: responseTime,
	}
	
	if err != nil {
		health.Status = HealthStatusUnhealthy
		health.Error = err.Error()
	} else {
		health.Status = HealthStatusHealthy
	}
	
	// Add performance details
	health.Details = map[string]interface{}{
		"response_time_ms": responseTime.Milliseconds(),
		"timeout_ms":       hm.timeout.Milliseconds(),
	}
	
	return health
}

// GetSystemHealth returns the overall system health
func (hm *HealthManager) GetSystemHealth() SystemHealth {
	componentHealth := hm.CheckAllComponents()
	
	summary := HealthSummary{
		TotalComponents: len(componentHealth),
	}
	
	overallStatus := HealthStatusHealthy
	
	for _, health := range componentHealth {
		switch health.Status {
		case HealthStatusHealthy:
			summary.HealthyComponents++
		case HealthStatusDegraded:
			summary.DegradedComponents++
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		case HealthStatusUnhealthy:
			summary.UnhealthyComponents++
			overallStatus = HealthStatusUnhealthy
		case HealthStatusUnknown:
			summary.UnknownComponents++
			if overallStatus == HealthStatusHealthy {
				overallStatus = HealthStatusDegraded
			}
		}
	}
	
	return SystemHealth{
		Status:     overallStatus,
		Timestamp:  time.Now(),
		Version:    hm.version,
		Uptime:     time.Since(hm.startTime),
		Components: componentHealth,
		Summary:    summary,
	}
}

// StartBackgroundChecking starts background health checking
func (hm *HealthManager) StartBackgroundChecking() {
	hm.mutex.Lock()
	if hm.running {
		hm.mutex.Unlock()
		return
	}
	hm.running = true
	hm.mutex.Unlock()
	
	hm.logger.Info("Starting background health checking", 
		zap.Duration("interval", hm.checkInterval))
	
	go func() {
		ticker := time.NewTicker(hm.checkInterval)
		defer ticker.Stop()
		
		for {
			select {
			case <-ticker.C:
				hm.CheckAllComponents()
			case <-hm.stopChan:
				hm.logger.Info("Stopping background health checking")
				return
			}
		}
	}()
}

// StopBackgroundChecking stops background health checking
func (hm *HealthManager) StopBackgroundChecking() {
	hm.mutex.Lock()
	if !hm.running {
		hm.mutex.Unlock()
		return
	}
	hm.running = false
	hm.mutex.Unlock()
	
	close(hm.stopChan)
}

// SetCheckInterval sets the background check interval
func (hm *HealthManager) SetCheckInterval(interval time.Duration) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.checkInterval = interval
}

// SetTimeout sets the health check timeout
func (hm *HealthManager) SetTimeout(timeout time.Duration) {
	hm.mutex.Lock()
	defer hm.mutex.Unlock()
	hm.timeout = timeout
}

// HealthHandler provides HTTP handlers for health endpoints
type HealthHandler struct {
	healthManager *HealthManager
	logger        *zap.Logger
}

// NewHealthHandler creates a new health handler
func NewHealthHandler(healthManager *HealthManager, logger *zap.Logger) *HealthHandler {
	return &HealthHandler{
		healthManager: healthManager,
		logger:        logger,
	}
}

// LivenessHandler handles liveness probe requests
func (hh *HealthHandler) LivenessHandler(w http.ResponseWriter, r *http.Request) {
	// Liveness check - just verify the service is running
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	response := map[string]interface{}{
		"status":    "alive",
		"timestamp": time.Now(),
		"uptime":    time.Since(hh.healthManager.startTime),
	}
	
	json.NewEncoder(w).Encode(response)
}

// ReadinessHandler handles readiness probe requests
func (hh *HealthHandler) ReadinessHandler(w http.ResponseWriter, r *http.Request) {
	systemHealth := hh.healthManager.GetSystemHealth()
	
	w.Header().Set("Content-Type", "application/json")
	
	// Return 200 if healthy or degraded, 503 if unhealthy
	if systemHealth.Status == HealthStatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	
	json.NewEncoder(w).Encode(systemHealth)
}

// HealthHandler handles detailed health check requests
func (hh *HealthHandler) HealthHandler(w http.ResponseWriter, r *http.Request) {
	systemHealth := hh.healthManager.GetSystemHealth()
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	
	json.NewEncoder(w).Encode(systemHealth)
}

// ComponentHealthHandler handles health check requests for specific components
func (hh *HealthHandler) ComponentHealthHandler(w http.ResponseWriter, r *http.Request) {
	componentName := r.URL.Query().Get("component")
	if componentName == "" {
		http.Error(w, "component parameter is required", http.StatusBadRequest)
		return
	}
	
	componentHealth := hh.healthManager.CheckComponent(componentName)
	
	w.Header().Set("Content-Type", "application/json")
	
	if componentHealth.Status == HealthStatusUnhealthy {
		w.WriteHeader(http.StatusServiceUnavailable)
	} else {
		w.WriteHeader(http.StatusOK)
	}
	
	json.NewEncoder(w).Encode(componentHealth)
}

// RegisterRoutes registers health check routes with an HTTP mux
func (hh *HealthHandler) RegisterRoutes(mux *http.ServeMux) {
	mux.HandleFunc("/health/live", hh.LivenessHandler)
	mux.HandleFunc("/health/ready", hh.ReadinessHandler)
	mux.HandleFunc("/health", hh.HealthHandler)
	mux.HandleFunc("/health/component", hh.ComponentHealthHandler)
}

// EngineHealthChecker implements HealthChecker for safety engines
type EngineHealthChecker struct {
	engine types.SafetyEngine
}

// NewEngineHealthChecker creates a new engine health checker
func NewEngineHealthChecker(engine types.SafetyEngine) *EngineHealthChecker {
	return &EngineHealthChecker{engine: engine}
}

// HealthCheck performs a health check on the engine
func (ehc *EngineHealthChecker) HealthCheck() error {
	return ehc.engine.HealthCheck()
}

// Name returns the engine name
func (ehc *EngineHealthChecker) Name() string {
	return fmt.Sprintf("engine_%s", ehc.engine.ID())
}
