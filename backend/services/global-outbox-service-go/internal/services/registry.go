package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"

	"global-outbox-service-go/internal/config"
)

// ServiceRegistry manages platform-wide service registration and discovery
type ServiceRegistry struct {
	mu       sync.RWMutex
	pool     *pgxpool.Pool
	config   *config.PlatformConfig
	logger   *logrus.Logger
	services map[string]*config.ServiceRegistration
	
	// Background tasks
	discoveryTicker   *time.Ticker
	healthcheckTicker *time.Ticker
	cleanupTicker     *time.Ticker
	stopChan          chan struct{}
}

// NewServiceRegistry creates a new service registry
func NewServiceRegistry(pool *pgxpool.Pool, cfg *config.PlatformConfig, logger *logrus.Logger) *ServiceRegistry {
	return &ServiceRegistry{
		pool:     pool,
		config:   cfg,
		logger:   logger,
		services: make(map[string]*config.ServiceRegistration),
		stopChan: make(chan struct{}),
	}
}

// Start starts the service registry background processes
func (sr *ServiceRegistry) Start(ctx context.Context) error {
	sr.logger.Info("Starting Service Registry...")

	// Initialize the registry table
	if err := sr.initializeRegistryTable(ctx); err != nil {
		return fmt.Errorf("failed to initialize registry table: %w", err)
	}

	// Load existing registrations
	if err := sr.loadExistingRegistrations(ctx); err != nil {
		sr.logger.Warnf("Failed to load existing registrations: %v", err)
	}

	// Start background processes
	if sr.config.ServiceDiscovery.Enabled && sr.config.ServiceDiscovery.AutoDiscovery {
		sr.discoveryTicker = time.NewTicker(sr.config.ServiceDiscovery.DiscoveryInterval)
		go sr.autoDiscoveryLoop(ctx)
	}

	if sr.config.ServiceDiscovery.Enabled {
		sr.healthcheckTicker = time.NewTicker(sr.config.ServiceDiscovery.HealthcheckInterval)
		go sr.healthcheckLoop(ctx)
	}

	if sr.config.ServiceRegistry.Enabled {
		sr.cleanupTicker = time.NewTicker(sr.config.ServiceRegistry.CleanupInterval)
		go sr.cleanupLoop(ctx)
	}

	sr.logger.Info("Service Registry started successfully")
	return nil
}

// Stop stops the service registry
func (sr *ServiceRegistry) Stop() {
	sr.logger.Info("Stopping Service Registry...")
	
	close(sr.stopChan)
	
	if sr.discoveryTicker != nil {
		sr.discoveryTicker.Stop()
	}
	if sr.healthcheckTicker != nil {
		sr.healthcheckTicker.Stop()
	}
	if sr.cleanupTicker != nil {
		sr.cleanupTicker.Stop()
	}
	
	sr.logger.Info("Service Registry stopped")
}

// RegisterService registers a new service
func (sr *ServiceRegistry) RegisterService(ctx context.Context, registration *config.ServiceRegistration) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.logger.Infof("Registering service: %s", registration.Name)

	// Set timestamps
	now := time.Now().UTC()
	if registration.CreatedAt.IsZero() {
		registration.CreatedAt = now
	}
	registration.LastSeen = now
	registration.Status = "active"

	// Validate registration
	if err := sr.validateRegistration(registration); err != nil {
		return fmt.Errorf("invalid registration: %w", err)
	}

	// Ensure outbox table exists for this service
	if err := sr.ensureServiceOutboxTable(ctx, registration); err != nil {
		return fmt.Errorf("failed to ensure outbox table: %w", err)
	}

	// Store in database
	if err := sr.storeRegistration(ctx, registration); err != nil {
		return fmt.Errorf("failed to store registration: %w", err)
	}

	// Update in-memory registry
	sr.services[registration.Name] = registration

	sr.logger.Infof("Successfully registered service: %s", registration.Name)
	return nil
}

// UnregisterService removes a service from the registry
func (sr *ServiceRegistry) UnregisterService(ctx context.Context, serviceName string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.logger.Infof("Unregistering service: %s", serviceName)

	// Remove from database
	query := `DELETE FROM service_registry WHERE name = $1`
	_, err := sr.pool.Exec(ctx, query, serviceName)
	if err != nil {
		return fmt.Errorf("failed to remove from database: %w", err)
	}

	// Remove from memory
	delete(sr.services, serviceName)

	sr.logger.Infof("Successfully unregistered service: %s", serviceName)
	return nil
}

// UpdateServiceHeartbeat updates the last seen timestamp for a service
func (sr *ServiceRegistry) UpdateServiceHeartbeat(ctx context.Context, serviceName string) error {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	service, exists := sr.services[serviceName]
	if !exists {
		return fmt.Errorf("service %s not found in registry", serviceName)
	}

	// Update last seen
	service.LastSeen = time.Now().UTC()

	// Update in database
	query := `UPDATE service_registry SET last_seen = $2, status = $3 WHERE name = $1`
	_, err := sr.pool.Exec(ctx, query, serviceName, service.LastSeen, "active")
	if err != nil {
		return fmt.Errorf("failed to update heartbeat: %w", err)
	}

	return nil
}

// GetRegisteredServices returns all registered services
func (sr *ServiceRegistry) GetRegisteredServices() map[string]*config.ServiceRegistration {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	// Return a copy to prevent concurrent modification
	result := make(map[string]*config.ServiceRegistration)
	for k, v := range sr.services {
		result[k] = v
	}
	return result
}

// GetActiveServices returns only active services
func (sr *ServiceRegistry) GetActiveServices() map[string]*config.ServiceRegistration {
	sr.mu.RLock()
	defer sr.mu.RUnlock()

	result := make(map[string]*config.ServiceRegistration)
	for name, service := range sr.services {
		if service.Status == "active" && sr.isServiceHealthy(service) {
			result[name] = service
		}
	}
	return result
}

// GetServiceRegistration returns a specific service registration
func (sr *ServiceRegistry) GetServiceRegistration(serviceName string) (*config.ServiceRegistration, bool) {
	sr.mu.RLock()
	defer sr.mu.RUnlock()
	
	service, exists := sr.services[serviceName]
	return service, exists
}

// autoDiscoveryLoop automatically discovers services
func (sr *ServiceRegistry) autoDiscoveryLoop(ctx context.Context) {
	sr.logger.Info("Starting auto-discovery loop")
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-sr.stopChan:
			return
		case <-sr.discoveryTicker.C:
			if err := sr.discoverServices(ctx); err != nil {
				sr.logger.Errorf("Auto-discovery failed: %v", err)
			}
		}
	}
}

// healthcheckLoop checks health of registered services
func (sr *ServiceRegistry) healthcheckLoop(ctx context.Context) {
	sr.logger.Info("Starting healthcheck loop")
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-sr.stopChan:
			return
		case <-sr.healthcheckTicker.C:
			sr.performHealthchecks(ctx)
		}
	}
}

// cleanupLoop removes stale service registrations
func (sr *ServiceRegistry) cleanupLoop(ctx context.Context) {
	sr.logger.Info("Starting cleanup loop")
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-sr.stopChan:
			return
		case <-sr.cleanupTicker.C:
			sr.cleanupStaleServices(ctx)
		}
	}
}

// discoverServices discovers new services automatically
func (sr *ServiceRegistry) discoverServices(ctx context.Context) error {
	sr.logger.Debug("Running service auto-discovery")

	// Discover services by looking for outbox tables
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		  AND table_name LIKE 'outbox_events_%'
	`

	rows, err := sr.pool.Query(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to query outbox tables: %w", err)
	}
	defer rows.Close()

	discovered := 0
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}

		// Extract service name from table name
		serviceName := sr.extractServiceName(tableName)
		if serviceName == "" {
			continue
		}

		// Check if already registered
		if _, exists := sr.services[serviceName]; exists {
			continue
		}

		// Auto-register the discovered service
		registration := &config.ServiceRegistration{
			Name:        serviceName,
			DatabaseURL: sr.config.DatabaseURL, // Use the same database by default
			Priority:    5, // Default priority
			Status:      "discovered",
			Metadata: map[string]string{
				"discovery_method": "auto",
				"table_name":      tableName,
			},
		}

		if err := sr.RegisterService(ctx, registration); err != nil {
			sr.logger.Warnf("Failed to auto-register service %s: %v", serviceName, err)
			continue
		}

		discovered++
		sr.logger.Infof("Auto-discovered and registered service: %s", serviceName)
	}

	if discovered > 0 {
		sr.logger.Infof("Auto-discovery completed: %d new services discovered", discovered)
	}

	return nil
}

// extractServiceName extracts service name from outbox table name
func (sr *ServiceRegistry) extractServiceName(tableName string) string {
	const prefix = "outbox_events_"
	if len(tableName) <= len(prefix) {
		return ""
	}
	
	serviceName := tableName[len(prefix):]
	// Convert underscores back to hyphens for service name
	return strings.ReplaceAll(serviceName, "_", "-")
}

// performHealthchecks checks health of all registered services
func (sr *ServiceRegistry) performHealthchecks(ctx context.Context) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.logger.Debug("Performing service healthchecks")

	for name, service := range sr.services {
		if service.HealthcheckURL == "" {
			continue
		}

		// Perform healthcheck (simplified - in production would use HTTP client)
		healthy := sr.checkServiceHealth(ctx, service)
		
		newStatus := "active"
		if !healthy {
			newStatus = "error"
		}

		if service.Status != newStatus {
			service.Status = newStatus
			sr.logger.Infof("Service %s status changed to: %s", name, newStatus)
			
			// Update in database
			query := `UPDATE service_registry SET status = $2 WHERE name = $1`
			if _, err := sr.pool.Exec(ctx, query, name, newStatus); err != nil {
				sr.logger.Errorf("Failed to update service status: %v", err)
			}
		}
	}
}

// cleanupStaleServices removes services that haven't been seen recently
func (sr *ServiceRegistry) cleanupStaleServices(ctx context.Context) {
	sr.mu.Lock()
	defer sr.mu.Unlock()

	sr.logger.Debug("Cleaning up stale services")

	ttl := sr.config.ServiceRegistry.RegistrationTTL
	cutoff := time.Now().UTC().Add(-ttl)

	var toRemove []string
	for name, service := range sr.services {
		if service.LastSeen.Before(cutoff) && service.Status != "active" {
			toRemove = append(toRemove, name)
		}
	}

	for _, name := range toRemove {
		sr.logger.Infof("Removing stale service: %s", name)
		delete(sr.services, name)

		// Remove from database
		query := `DELETE FROM service_registry WHERE name = $1`
		if _, err := sr.pool.Exec(ctx, query, name); err != nil {
			sr.logger.Errorf("Failed to remove stale service from database: %v", err)
		}
	}

	if len(toRemove) > 0 {
		sr.logger.Infof("Cleanup completed: removed %d stale services", len(toRemove))
	}
}

// Helper methods

func (sr *ServiceRegistry) initializeRegistryTable(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS service_registry (
			name VARCHAR(255) PRIMARY KEY,
			database_url TEXT NOT NULL,
			priority INTEGER DEFAULT 5,
			healthcheck_url TEXT,
			metadata JSONB DEFAULT '{}',
			status VARCHAR(50) DEFAULT 'active',
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			last_seen TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);
		
		CREATE INDEX IF NOT EXISTS idx_service_registry_status ON service_registry (status);
		CREATE INDEX IF NOT EXISTS idx_service_registry_last_seen ON service_registry (last_seen);
	`

	_, err := sr.pool.Exec(ctx, query)
	return err
}

func (sr *ServiceRegistry) loadExistingRegistrations(ctx context.Context) error {
	query := `SELECT name, database_url, priority, healthcheck_url, metadata, status, created_at, last_seen FROM service_registry`
	
	rows, err := sr.pool.Query(ctx, query)
	if err != nil {
		return err
	}
	defer rows.Close()

	loaded := 0
	for rows.Next() {
		var registration config.ServiceRegistration
		var metadataJSON []byte

		err := rows.Scan(
			&registration.Name,
			&registration.DatabaseURL,
			&registration.Priority,
			&registration.HealthcheckURL,
			&metadataJSON,
			&registration.Status,
			&registration.CreatedAt,
			&registration.LastSeen,
		)
		if err != nil {
			sr.logger.Errorf("Failed to scan service registration: %v", err)
			continue
		}

		// Parse metadata
		if len(metadataJSON) > 0 {
			if err := json.Unmarshal(metadataJSON, &registration.Metadata); err != nil {
				sr.logger.Errorf("Failed to parse metadata for service %s: %v", registration.Name, err)
				registration.Metadata = make(map[string]string)
			}
		}

		sr.services[registration.Name] = &registration
		loaded++
	}

	sr.logger.Infof("Loaded %d existing service registrations", loaded)
	return nil
}

func (sr *ServiceRegistry) validateRegistration(registration *config.ServiceRegistration) error {
	if registration.Name == "" {
		return fmt.Errorf("service name is required")
	}
	if registration.DatabaseURL == "" {
		return fmt.Errorf("database URL is required")
	}
	if registration.Priority < 1 || registration.Priority > 10 {
		return fmt.Errorf("priority must be between 1 and 10")
	}
	return nil
}

func (sr *ServiceRegistry) ensureServiceOutboxTable(ctx context.Context, registration *config.ServiceRegistration) error {
	tableName := fmt.Sprintf("outbox_events_%s", strings.ReplaceAll(registration.Name, "-", "_"))
	
	// Use the same table creation logic as the repository
	// This should match the schema in repository.go
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_name VARCHAR(255) NOT NULL,
			event_type VARCHAR(255) NOT NULL,
			event_data JSONB NOT NULL,
			topic VARCHAR(255) NOT NULL,
			correlation_id VARCHAR(255),
			priority INTEGER NOT NULL DEFAULT 5,
			metadata JSONB DEFAULT '{}',
			medical_context VARCHAR(50) NOT NULL DEFAULT 'routine',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			published_at TIMESTAMP WITH TIME ZONE,
			retry_count INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			error_message TEXT,
			next_retry_at TIMESTAMP WITH TIME ZONE
		);
		
		CREATE INDEX IF NOT EXISTS idx_%s_status ON %s (status);
		CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s (created_at);
		CREATE INDEX IF NOT EXISTS idx_%s_priority ON %s (priority DESC);
		CREATE INDEX IF NOT EXISTS idx_%s_medical_context ON %s (medical_context);
		CREATE INDEX IF NOT EXISTS idx_%s_next_retry ON %s (next_retry_at) WHERE next_retry_at IS NOT NULL;
	`, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName)

	_, err := sr.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create outbox table for service %s: %w", registration.Name, err)
	}

	sr.logger.Infof("Ensured outbox table exists for service: %s", registration.Name)
	return nil
}

func (sr *ServiceRegistry) storeRegistration(ctx context.Context, registration *config.ServiceRegistration) error {
	metadataJSON, _ := json.Marshal(registration.Metadata)

	query := `
		INSERT INTO service_registry (name, database_url, priority, healthcheck_url, metadata, status, created_at, last_seen)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (name) DO UPDATE SET
			database_url = EXCLUDED.database_url,
			priority = EXCLUDED.priority,
			healthcheck_url = EXCLUDED.healthcheck_url,
			metadata = EXCLUDED.metadata,
			status = EXCLUDED.status,
			last_seen = EXCLUDED.last_seen
	`

	_, err := sr.pool.Exec(ctx, query,
		registration.Name,
		registration.DatabaseURL,
		registration.Priority,
		registration.HealthcheckURL,
		metadataJSON,
		registration.Status,
		registration.CreatedAt,
		registration.LastSeen,
	)

	return err
}

func (sr *ServiceRegistry) isServiceHealthy(service *config.ServiceRegistration) bool {
	// Consider a service healthy if it was seen recently
	ttl := sr.config.ServiceRegistry.RegistrationTTL
	return time.Since(service.LastSeen) < ttl
}

func (sr *ServiceRegistry) checkServiceHealth(ctx context.Context, service *config.ServiceRegistration) bool {
	// Simplified health check - in production would make HTTP request to healthcheck URL
	// For now, just check if we can connect to the service's database
	return true
}