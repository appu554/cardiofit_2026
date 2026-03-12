package infrastructure

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// ApolloFederationFactory creates and manages Apollo Federation client instances
type ApolloFederationFactory struct {
	config        *ApolloFederationConfig
	logger        *zap.Logger
	clients       map[string]*ApolloFederationClient
	clientsMu     sync.RWMutex
	healthChecker *FederationHealthChecker
	metrics       *FederationMetrics
}

// FederationHealthChecker monitors Apollo Federation gateway health
type FederationHealthChecker struct {
	factory       *ApolloFederationFactory
	logger        *zap.Logger
	checkInterval time.Duration
	timeout       time.Duration
	healthy       map[string]bool
	healthMu      sync.RWMutex
	lastCheck     map[string]time.Time
}

// FederationMetrics tracks Apollo Federation performance metrics
type FederationMetrics struct {
	QueryCount        map[string]int64
	QueryDuration     map[string]time.Duration
	QueryErrors       map[string]int64
	CacheHits         int64
	CacheMisses       int64
	CircuitBreakerEvents map[string]int64
	mu               sync.RWMutex
}

// NewApolloFederationFactory creates a new factory for Apollo Federation clients
func NewApolloFederationFactory(config *ApolloFederationConfig, logger *zap.Logger) *ApolloFederationFactory {
	factory := &ApolloFederationFactory{
		config:  config,
		logger:  logger,
		clients: make(map[string]*ApolloFederationClient),
		metrics: NewFederationMetrics(),
	}

	// Initialize health checker if enabled
	if config.HealthCheckEnabled {
		factory.healthChecker = &FederationHealthChecker{
			factory:       factory,
			logger:        logger,
			checkInterval: config.HealthCheckInterval,
			timeout:       5 * time.Second,
			healthy:       make(map[string]bool),
			lastCheck:     make(map[string]time.Time),
		}
		
		go factory.healthChecker.Start()
	}

	return factory
}

// GetClient returns an Apollo Federation client instance (singleton per configuration)
func (f *ApolloFederationFactory) GetClient(ctx context.Context) (*ApolloFederationClient, error) {
	return f.GetNamedClient(ctx, "default")
}

// GetNamedClient returns a named Apollo Federation client instance
func (f *ApolloFederationFactory) GetNamedClient(ctx context.Context, name string) (*ApolloFederationClient, error) {
	f.clientsMu.RLock()
	if client, exists := f.clients[name]; exists {
		f.clientsMu.RUnlock()
		return client, nil
	}
	f.clientsMu.RUnlock()

	// Create new client with write lock
	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := f.clients[name]; exists {
		return client, nil
	}

	// Create new client
	client := NewApolloFederationClient(
		f.config.URL,
		f.config.Timeout,
		f.logger,
	)

	// Configure client with additional settings
	f.configureClient(client)

	f.clients[name] = client

	f.logger.Info("Created new Apollo Federation client",
		zap.String("name", name),
		zap.String("url", f.config.URL),
	)

	return client, nil
}

// configureClient applies additional configuration to the client
func (f *ApolloFederationFactory) configureClient(client *ApolloFederationClient) {
	// Configure retry settings
	client.maxRetries = f.config.MaxRetries
	client.retryDelay = f.config.RetryDelay

	// Add performance monitoring if enabled
	if f.config.MetricsEnabled {
		// Client will use the factory's metrics
	}
}

// CreateClientWithCustomConfig creates a client with custom configuration
func (f *ApolloFederationFactory) CreateClientWithCustomConfig(
	ctx context.Context,
	customConfig *ApolloFederationConfig,
	name string,
) (*ApolloFederationClient, error) {
	if err := customConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid custom config: %w", err)
	}

	client := NewApolloFederationClient(
		customConfig.URL,
		customConfig.Timeout,
		f.logger,
	)

	// Apply custom configuration
	client.maxRetries = customConfig.MaxRetries
	client.retryDelay = customConfig.RetryDelay

	// Store with custom name to avoid conflicts
	customName := fmt.Sprintf("custom_%s", name)
	f.clientsMu.Lock()
	f.clients[customName] = client
	f.clientsMu.Unlock()

	f.logger.Info("Created custom Apollo Federation client",
		zap.String("name", customName),
		zap.String("url", customConfig.URL),
	)

	return client, nil
}

// GetHealthyClient returns a healthy client instance
func (f *ApolloFederationFactory) GetHealthyClient(ctx context.Context) (*ApolloFederationClient, error) {
	if f.healthChecker == nil {
		// Health checking not enabled, return default client
		return f.GetClient(ctx)
	}

	// Check if default client is healthy
	if f.healthChecker.IsHealthy("default") {
		return f.GetNamedClient(ctx, "default")
	}

	// If default is not healthy, try other clients
	f.clientsMu.RLock()
	clientNames := make([]string, 0, len(f.clients))
	for name := range f.clients {
		clientNames = append(clientNames, name)
	}
	f.clientsMu.RUnlock()

	for _, name := range clientNames {
		if f.healthChecker.IsHealthy(name) {
			client, err := f.GetNamedClient(ctx, name)
			if err == nil {
				return client, nil
			}
		}
	}

	return nil, fmt.Errorf("no healthy Apollo Federation clients available")
}

// GetMetrics returns current federation metrics
func (f *ApolloFederationFactory) GetMetrics() *FederationMetrics {
	return f.metrics
}

// GetHealthStatus returns health status for all clients
func (f *ApolloFederationFactory) GetHealthStatus() map[string]interface{} {
	status := make(map[string]interface{})

	if f.healthChecker == nil {
		status["health_checking_enabled"] = false
		return status
	}

	f.healthChecker.healthMu.RLock()
	defer f.healthChecker.healthMu.RUnlock()

	status["health_checking_enabled"] = true
	status["clients"] = make(map[string]interface{})

	clientsStatus := status["clients"].(map[string]interface{})
	for name, healthy := range f.healthChecker.healthy {
		clientsStatus[name] = map[string]interface{}{
			"healthy":    healthy,
			"last_check": f.healthChecker.lastCheck[name],
		}
	}

	return status
}

// Shutdown gracefully shuts down all clients and background processes
func (f *ApolloFederationFactory) Shutdown(ctx context.Context) error {
	f.logger.Info("Shutting down Apollo Federation factory")

	// Stop health checker
	if f.healthChecker != nil {
		f.healthChecker.Stop()
	}

	f.clientsMu.Lock()
	defer f.clientsMu.Unlock()

	// Clear clients
	for name := range f.clients {
		delete(f.clients, name)
	}

	f.logger.Info("Apollo Federation factory shutdown complete")
	return nil
}

// RecordQueryMetric records a query performance metric
func (f *ApolloFederationFactory) RecordQueryMetric(queryType string, duration time.Duration, success bool) {
	if !f.config.MetricsEnabled {
		return
	}

	f.metrics.RecordQuery(queryType, duration, success)
}

// NewFederationMetrics creates a new metrics instance
func NewFederationMetrics() *FederationMetrics {
	return &FederationMetrics{
		QueryCount:           make(map[string]int64),
		QueryDuration:        make(map[string]time.Duration),
		QueryErrors:          make(map[string]int64),
		CircuitBreakerEvents: make(map[string]int64),
	}
}

// RecordQuery records query metrics
func (m *FederationMetrics) RecordQuery(queryType string, duration time.Duration, success bool) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueryCount[queryType]++
	m.QueryDuration[queryType] += duration

	if !success {
		m.QueryErrors[queryType]++
	}
}

// RecordCacheHit records a cache hit
func (m *FederationMetrics) RecordCacheHit() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheHits++
}

// RecordCacheMiss records a cache miss
func (m *FederationMetrics) RecordCacheMiss() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CacheMisses++
}

// RecordCircuitBreakerEvent records a circuit breaker event
func (m *FederationMetrics) RecordCircuitBreakerEvent(eventType string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.CircuitBreakerEvents[eventType]++
}

// GetSnapshot returns a snapshot of current metrics
func (m *FederationMetrics) GetSnapshot() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	snapshot := make(map[string]interface{})
	snapshot["query_count"] = copyInt64Map(m.QueryCount)
	snapshot["query_errors"] = copyInt64Map(m.QueryErrors)
	snapshot["cache_hits"] = m.CacheHits
	snapshot["cache_misses"] = m.CacheMisses
	snapshot["circuit_breaker_events"] = copyInt64Map(m.CircuitBreakerEvents)

	// Calculate average durations
	avgDurations := make(map[string]float64)
	for queryType, totalDuration := range m.QueryDuration {
		if count := m.QueryCount[queryType]; count > 0 {
			avgDurations[queryType] = float64(totalDuration.Milliseconds()) / float64(count)
		}
	}
	snapshot["avg_query_duration_ms"] = avgDurations

	// Calculate cache hit rate
	totalCacheOps := m.CacheHits + m.CacheMisses
	if totalCacheOps > 0 {
		snapshot["cache_hit_rate"] = float64(m.CacheHits) / float64(totalCacheOps)
	} else {
		snapshot["cache_hit_rate"] = 0.0
	}

	return snapshot
}

// Reset resets all metrics
func (m *FederationMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.QueryCount = make(map[string]int64)
	m.QueryDuration = make(map[string]time.Duration)
	m.QueryErrors = make(map[string]int64)
	m.CacheHits = 0
	m.CacheMisses = 0
	m.CircuitBreakerEvents = make(map[string]int64)
}

// Health Checker Methods

// Start starts the health checker background process
func (h *FederationHealthChecker) Start() {
	ticker := time.NewTicker(h.checkInterval)
	defer ticker.Stop()

	// Initial health check
	h.performHealthCheck()

	for {
		select {
		case <-ticker.C:
			h.performHealthCheck()
		}
	}
}

// Stop stops the health checker (placeholder - would use context in real implementation)
func (h *FederationHealthChecker) Stop() {
	// In a real implementation, this would use a context to signal shutdown
	h.logger.Info("Health checker stopped")
}

// IsHealthy returns the health status for a named client
func (h *FederationHealthChecker) IsHealthy(name string) bool {
	h.healthMu.RLock()
	defer h.healthMu.RUnlock()
	
	healthy, exists := h.healthy[name]
	return exists && healthy
}

// performHealthCheck performs health checks on all clients
func (h *FederationHealthChecker) performHealthCheck() {
	h.factory.clientsMu.RLock()
	clientNames := make([]string, 0, len(h.factory.clients))
	clients := make([]*ApolloFederationClient, 0, len(h.factory.clients))
	
	for name, client := range h.factory.clients {
		clientNames = append(clientNames, name)
		clients = append(clients, client)
	}
	h.factory.clientsMu.RUnlock()

	// Perform health checks concurrently
	var wg sync.WaitGroup
	for i, client := range clients {
		wg.Add(1)
		go func(name string, c *ApolloFederationClient) {
			defer wg.Done()
			
			ctx, cancel := context.WithTimeout(context.Background(), h.timeout)
			defer cancel()
			
			healthy := c.HealthCheck(ctx) == nil
			
			h.healthMu.Lock()
			h.healthy[name] = healthy
			h.lastCheck[name] = time.Now()
			h.healthMu.Unlock()
			
			if !healthy {
				h.logger.Warn("Apollo Federation client health check failed",
					zap.String("client_name", name),
				)
			}
		}(clientNames[i], client)
	}

	wg.Wait()
}

// Helper functions

func copyInt64Map(original map[string]int64) map[string]int64 {
	copy := make(map[string]int64)
	for k, v := range original {
		copy[k] = v
	}
	return copy
}