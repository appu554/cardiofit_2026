package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

// Enhanced configuration for platform-wide event publishing

// ServiceRegistration represents a registered microservice
type ServiceRegistration struct {
	Name             string            `json:"name" mapstructure:"name"`
	DatabaseURL      string            `json:"database_url" mapstructure:"database_url"`
	Priority         int               `json:"priority" mapstructure:"priority"`
	HealthcheckURL   string            `json:"healthcheck_url" mapstructure:"healthcheck_url"`
	Metadata         map[string]string `json:"metadata" mapstructure:"metadata"`
	CreatedAt        time.Time         `json:"created_at" mapstructure:"created_at"`
	LastSeen         time.Time         `json:"last_seen" mapstructure:"last_seen"`
	Status           string            `json:"status" mapstructure:"status"` // active, inactive, error
}

// PlatformConfig holds enhanced platform-wide configuration
type PlatformConfig struct {
	*Config // Embed the original config

	// Enhanced Service Discovery
	ServiceDiscovery ServiceDiscoveryConfig `mapstructure:"service_discovery"`
	
	// Service Registration
	ServiceRegistry ServiceRegistryConfig `mapstructure:"service_registry"`
	
	// Multi-tenancy Support
	MultiTenant MultiTenantConfig `mapstructure:"multi_tenant"`
	
	// Enhanced Monitoring
	Monitoring MonitoringConfig `mapstructure:"monitoring"`
	
	// SDK Configuration
	SDK SDKConfig `mapstructure:"sdk"`
}

// ServiceDiscoveryConfig configures how services are discovered
type ServiceDiscoveryConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	AutoDiscovery     bool          `mapstructure:"auto_discovery"`
	DiscoveryInterval time.Duration `mapstructure:"discovery_interval"`
	HealthcheckInterval time.Duration `mapstructure:"healthcheck_interval"`
	DatabasePatterns  []string      `mapstructure:"database_patterns"`
	ServicePatterns   []string      `mapstructure:"service_patterns"`
}

// ServiceRegistryConfig configures the service registry
type ServiceRegistryConfig struct {
	Enabled           bool          `mapstructure:"enabled"`
	RegistrationTTL   time.Duration `mapstructure:"registration_ttl"`
	HeartbeatInterval time.Duration `mapstructure:"heartbeat_interval"`
	CleanupInterval   time.Duration `mapstructure:"cleanup_interval"`
	RequireAuth       bool          `mapstructure:"require_auth"`
}

// MultiTenantConfig enables multi-tenant capabilities
type MultiTenantConfig struct {
	Enabled                bool                       `mapstructure:"enabled"`
	IsolationLevel        string                     `mapstructure:"isolation_level"` // service, database, namespace
	DefaultConfiguration  map[string]interface{}     `mapstructure:"default_configuration"`
	ServiceOverrides      map[string]ServiceOverride `mapstructure:"service_overrides"`
}

// ServiceOverride allows per-service configuration overrides
type ServiceOverride struct {
	PollInterval      *time.Duration `mapstructure:"poll_interval"`
	BatchSize         *int           `mapstructure:"batch_size"`
	MaxWorkers        *int           `mapstructure:"max_workers"`
	CircuitBreaker    *bool          `mapstructure:"circuit_breaker"`
	Priority          *int           `mapstructure:"priority"`
	CustomTopicPrefix *string        `mapstructure:"custom_topic_prefix"`
}

// MonitoringConfig enhances monitoring capabilities
type MonitoringConfig struct {
	EnablePerServiceMetrics bool          `mapstructure:"enable_per_service_metrics"`
	MetricsRetentionPeriod  time.Duration `mapstructure:"metrics_retention_period"`
	AlertingEnabled         bool          `mapstructure:"alerting_enabled"`
	AlertThresholds         AlertThresholds `mapstructure:"alert_thresholds"`
	TracingEnabled          bool          `mapstructure:"tracing_enabled"`
	TracingSampleRate       float64       `mapstructure:"tracing_sample_rate"`
}

// AlertThresholds defines when to trigger alerts
type AlertThresholds struct {
	QueueDepthWarning   int           `mapstructure:"queue_depth_warning"`
	QueueDepthCritical  int           `mapstructure:"queue_depth_critical"`
	ProcessingLatency   time.Duration `mapstructure:"processing_latency"`
	ErrorRate           float64       `mapstructure:"error_rate"`
	DeadLetterThreshold int           `mapstructure:"dead_letter_threshold"`
}

// SDKConfig configures the client SDK
type SDKConfig struct {
	Version              string        `mapstructure:"version"`
	DefaultTimeout       time.Duration `mapstructure:"default_timeout"`
	RetryAttempts        int           `mapstructure:"retry_attempts"`
	ConnectionPoolSize   int           `mapstructure:"connection_pool_size"`
	EnableCircuitBreaker bool          `mapstructure:"enable_circuit_breaker"`
	EnableTracing        bool          `mapstructure:"enable_tracing"`
}

// LoadPlatformConfig loads the enhanced platform configuration
func LoadPlatformConfig() (*PlatformConfig, error) {
	// Load base config first
	baseConfig, err := Load()
	if err != nil {
		return nil, fmt.Errorf("failed to load base config: %w", err)
	}

	// Set enhanced defaults
	setEnhancedDefaults()

	var platformConfig PlatformConfig
	if err := viper.Unmarshal(&platformConfig); err != nil {
		return nil, fmt.Errorf("error unmarshaling platform config: %w", err)
	}

	// Set the base config
	platformConfig.Config = baseConfig

	// Validate enhanced configuration
	if err := platformConfig.Validate(); err != nil {
		return nil, fmt.Errorf("invalid platform configuration: %w", err)
	}

	return &platformConfig, nil
}

func setEnhancedDefaults() {
	// Service Discovery Configuration
	viper.SetDefault("service_discovery.enabled", true)
	viper.SetDefault("service_discovery.auto_discovery", true)
	viper.SetDefault("service_discovery.discovery_interval", "30s")
	viper.SetDefault("service_discovery.healthcheck_interval", "10s")
	viper.SetDefault("service_discovery.database_patterns", []string{
		"*-service",
		"*_service",
	})
	viper.SetDefault("service_discovery.service_patterns", []string{
		"outbox_events_*",
	})

	// Service Registry Configuration
	viper.SetDefault("service_registry.enabled", true)
	viper.SetDefault("service_registry.registration_ttl", "5m")
	viper.SetDefault("service_registry.heartbeat_interval", "30s")
	viper.SetDefault("service_registry.cleanup_interval", "1m")
	viper.SetDefault("service_registry.require_auth", false)

	// Multi-tenant Configuration
	viper.SetDefault("multi_tenant.enabled", true)
	viper.SetDefault("multi_tenant.isolation_level", "service")

	// Monitoring Configuration
	viper.SetDefault("monitoring.enable_per_service_metrics", true)
	viper.SetDefault("monitoring.metrics_retention_period", "24h")
	viper.SetDefault("monitoring.alerting_enabled", true)
	viper.SetDefault("monitoring.alert_thresholds.queue_depth_warning", 100)
	viper.SetDefault("monitoring.alert_thresholds.queue_depth_critical", 500)
	viper.SetDefault("monitoring.alert_thresholds.processing_latency", "5s")
	viper.SetDefault("monitoring.alert_thresholds.error_rate", 0.1)
	viper.SetDefault("monitoring.alert_thresholds.dead_letter_threshold", 10)
	viper.SetDefault("monitoring.tracing_enabled", true)
	viper.SetDefault("monitoring.tracing_sample_rate", 0.1)

	// SDK Configuration
	viper.SetDefault("sdk.version", "1.0.0")
	viper.SetDefault("sdk.default_timeout", "30s")
	viper.SetDefault("sdk.retry_attempts", 3)
	viper.SetDefault("sdk.connection_pool_size", 10)
	viper.SetDefault("sdk.enable_circuit_breaker", true)
	viper.SetDefault("sdk.enable_tracing", true)
}

// Validate validates the platform configuration
func (pc *PlatformConfig) Validate() error {
	// Validate service discovery
	if pc.ServiceDiscovery.Enabled {
		if pc.ServiceDiscovery.DiscoveryInterval < time.Second {
			return fmt.Errorf("service_discovery.discovery_interval must be at least 1 second")
		}
		if pc.ServiceDiscovery.HealthcheckInterval < time.Second {
			return fmt.Errorf("service_discovery.healthcheck_interval must be at least 1 second")
		}
	}

	// Validate service registry
	if pc.ServiceRegistry.Enabled {
		if pc.ServiceRegistry.RegistrationTTL < time.Minute {
			return fmt.Errorf("service_registry.registration_ttl must be at least 1 minute")
		}
		if pc.ServiceRegistry.HeartbeatInterval < time.Second {
			return fmt.Errorf("service_registry.heartbeat_interval must be at least 1 second")
		}
	}

	// Validate monitoring
	if pc.Monitoring.AlertThresholds.ErrorRate < 0 || pc.Monitoring.AlertThresholds.ErrorRate > 1 {
		return fmt.Errorf("monitoring.alert_thresholds.error_rate must be between 0 and 1")
	}
	if pc.Monitoring.TracingSampleRate < 0 || pc.Monitoring.TracingSampleRate > 1 {
		return fmt.Errorf("monitoring.tracing_sample_rate must be between 0 and 1")
	}

	return nil
}

// GetServiceOverride returns configuration override for a specific service
func (pc *PlatformConfig) GetServiceOverride(serviceName string) *ServiceOverride {
	if override, exists := pc.MultiTenant.ServiceOverrides[serviceName]; exists {
		return &override
	}
	return nil
}

// GetEffectivePollInterval returns the effective poll interval for a service
func (pc *PlatformConfig) GetEffectivePollInterval(serviceName string) time.Duration {
	if override := pc.GetServiceOverride(serviceName); override != nil && override.PollInterval != nil {
		return *override.PollInterval
	}
	return pc.PublisherPollInterval
}

// GetEffectiveBatchSize returns the effective batch size for a service
func (pc *PlatformConfig) GetEffectiveBatchSize(serviceName string) int {
	if override := pc.GetServiceOverride(serviceName); override != nil && override.BatchSize != nil {
		return *override.BatchSize
	}
	return pc.PublisherBatchSize
}

// IsServiceEnabled checks if a service should be processed
func (pc *PlatformConfig) IsServiceEnabled(serviceName string) bool {
	// If we have supported services list and it's not empty, check if service is in it
	if len(pc.SupportedServices) > 0 {
		for _, supported := range pc.SupportedServices {
			if supported == serviceName || supported == "all" || supported == "*" {
				return true
			}
		}
		return false
	}
	
	// If service discovery is enabled, all discovered services are enabled by default
	return pc.ServiceDiscovery.Enabled
}

// GetTopicPrefix returns the topic prefix for a service
func (pc *PlatformConfig) GetTopicPrefix(serviceName string) string {
	if override := pc.GetServiceOverride(serviceName); override != nil && override.CustomTopicPrefix != nil {
		return *override.CustomTopicPrefix
	}
	
	// Default topic prefix strategy
	return fmt.Sprintf("clinical.%s", strings.ReplaceAll(serviceName, "-", "_"))
}