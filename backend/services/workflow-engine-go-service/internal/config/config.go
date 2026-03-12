package config

import (
	"fmt"
	"time"

	"github.com/kelseyhightower/envconfig"
)

// DatabaseConfig holds database connection configuration
type DatabaseConfig struct {
	URL             string        `envconfig:"DATABASE_URL" required:"true"`
	MaxConnections  int           `envconfig:"DATABASE_MAX_CONNECTIONS" default:"10"`
	MinConnections  int           `envconfig:"DATABASE_MIN_CONNECTIONS" default:"5"`
	AcquireTimeout  time.Duration `envconfig:"DATABASE_ACQUIRE_TIMEOUT" default:"30s"`
	IdleTimeout     time.Duration `envconfig:"DATABASE_IDLE_TIMEOUT" default:"600s"`
	MaxLifetime     time.Duration `envconfig:"DATABASE_MAX_LIFETIME" default:"1800s"`
}

// ExternalServicesConfig holds external service URLs
type ExternalServicesConfig struct {
	AuthServiceURL       string `envconfig:"AUTH_SERVICE_URL" default:"http://localhost:8001"`
	MedicationServiceURL string `envconfig:"MEDICATION_SERVICE_URL" default:"http://localhost:8004"`
	SafetyGatewayURL     string `envconfig:"SAFETY_GATEWAY_URL" default:"http://localhost:8018"`
	Flow2GoURL           string `envconfig:"FLOW2_GO_URL" default:"http://localhost:8080"`
	Flow2RustURL         string `envconfig:"FLOW2_RUST_URL" default:"http://localhost:8090"`
	ContextGatewayURL    string `envconfig:"CONTEXT_GATEWAY_URL" default:"http://localhost:8016"`
}

// WorkflowConfig holds workflow engine configuration
type WorkflowConfig struct {
	ExecutionTimeout      time.Duration `envconfig:"WORKFLOW_EXECUTION_TIMEOUT" default:"1h"`
	TaskAssignmentTimeout time.Duration `envconfig:"WORKFLOW_TASK_ASSIGNMENT_TIMEOUT" default:"24h"`
	EventPollingInterval  time.Duration `envconfig:"WORKFLOW_EVENT_POLLING_INTERVAL" default:"30s"`
	TaskPollingInterval   time.Duration `envconfig:"WORKFLOW_TASK_POLLING_INTERVAL" default:"10s"`
	MockMode              bool          `envconfig:"WORKFLOW_MOCK_MODE" default:"false"`
	EnableWebhooks        bool          `envconfig:"WORKFLOW_ENABLE_WEBHOOKS" default:"true"`
	EnableFHIRMonitoring  bool          `envconfig:"WORKFLOW_ENABLE_FHIR_MONITORING" default:"true"`
}

// PerformanceConfig holds performance target configuration
type PerformanceConfig struct {
	CalculateTargetMs int64 `envconfig:"PERFORMANCE_CALCULATE_TARGET_MS" default:"175"`
	ValidateTargetMs  int64 `envconfig:"PERFORMANCE_VALIDATE_TARGET_MS" default:"100"`
	CommitTargetMs    int64 `envconfig:"PERFORMANCE_COMMIT_TARGET_MS" default:"50"`
	TotalTargetMs     int64 `envconfig:"PERFORMANCE_TOTAL_TARGET_MS" default:"325"`
}

// MonitoringConfig holds monitoring and observability configuration
type MonitoringConfig struct {
	PrometheusEnabled     bool          `envconfig:"MONITORING_PROMETHEUS_ENABLED" default:"true"`
	JaegerEndpoint        string        `envconfig:"MONITORING_JAEGER_ENDPOINT" default:"http://localhost:14268/api/traces"`
	MetricsPort           int           `envconfig:"MONITORING_METRICS_PORT" default:"9090"`
	HealthCheckInterval   time.Duration `envconfig:"MONITORING_HEALTH_CHECK_INTERVAL" default:"30s"`
}

// GoogleCloudConfig holds Google Healthcare API configuration
type GoogleCloudConfig struct {
	ProjectID       string `envconfig:"GOOGLE_CLOUD_PROJECT" default:"cardiofit-905a8"`
	Location        string `envconfig:"GOOGLE_CLOUD_LOCATION" default:"asia-south1"`
	Dataset         string `envconfig:"GOOGLE_CLOUD_DATASET" default:"clinical-synthesis-hub"`
	FHIRStore       string `envconfig:"GOOGLE_CLOUD_FHIR_STORE" default:"fhir-store"`
	CredentialsPath string `envconfig:"GOOGLE_APPLICATION_CREDENTIALS" default:"credentials/google-credentials.json"`
	Enabled         bool   `envconfig:"USE_GOOGLE_HEALTHCARE_API" default:"true"`
}

// CamundaConfig holds Camunda workflow engine configuration
type CamundaConfig struct {
	EngineURL                    string `envconfig:"CAMUNDA_ENGINE_URL" default:"http://localhost:8080/engine-rest"`
	UseCloudVersion              bool   `envconfig:"USE_CAMUNDA_CLOUD" default:"false"`
	CloudClientID                string `envconfig:"CAMUNDA_CLOUD_CLIENT_ID"`
	CloudClientSecret            string `envconfig:"CAMUNDA_CLOUD_CLIENT_SECRET"`
	CloudClusterID               string `envconfig:"CAMUNDA_CLOUD_CLUSTER_ID"`
	CloudRegion                  string `envconfig:"CAMUNDA_CLOUD_REGION"`
	CloudAuthorizationServerURL  string `envconfig:"CAMUNDA_CLOUD_AUTHORIZATION_SERVER_URL" default:"https://login.cloud.camunda.io/oauth/token"`
}

// Config holds all application configuration
type Config struct {
	// Service Configuration
	ServiceName string `envconfig:"SERVICE_NAME" default:"workflow-engine-service"`
	ServicePort int    `envconfig:"SERVICE_PORT" default:"8017"`
	Debug       bool   `envconfig:"DEBUG" default:"true"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`

	// Component configurations
	Database         DatabaseConfig         `envconfig:""`
	ExternalServices ExternalServicesConfig `envconfig:""`
	Workflow         WorkflowConfig         `envconfig:""`
	Performance      PerformanceConfig      `envconfig:""`
	Monitoring       MonitoringConfig       `envconfig:""`
	GoogleCloud      GoogleCloudConfig      `envconfig:""`
	Camunda          CamundaConfig          `envconfig:""`

	// Security
	JWTSecret       string        `envconfig:"JWT_SECRET" default:"your-secret-key"`
	JWTExpiration   time.Duration `envconfig:"JWT_EXPIRATION" default:"24h"`
	
	// Rate limiting
	RateLimitEnabled bool `envconfig:"RATE_LIMIT_ENABLED" default:"true"`
	RateLimitRPS     int  `envconfig:"RATE_LIMIT_RPS" default:"100"`
}

// Load loads configuration from environment variables
func Load() (*Config, error) {
	var config Config
	if err := envconfig.Process("", &config); err != nil {
		return nil, fmt.Errorf("failed to process environment configuration: %w", err)
	}

	return &config, nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is required")
	}

	if c.ServicePort <= 0 || c.ServicePort > 65535 {
		return fmt.Errorf("invalid service port: %d", c.ServicePort)
	}

	if c.Database.MaxConnections < c.Database.MinConnections {
		return fmt.Errorf("database max connections (%d) must be >= min connections (%d)",
			c.Database.MaxConnections, c.Database.MinConnections)
	}

	if c.Performance.CalculateTargetMs <= 0 {
		return fmt.Errorf("calculate target must be positive")
	}

	if c.Performance.ValidateTargetMs <= 0 {
		return fmt.Errorf("validate target must be positive")
	}

	if c.Performance.CommitTargetMs <= 0 {
		return fmt.Errorf("commit target must be positive")
	}

	if c.Camunda.UseCloudVersion {
		if c.Camunda.CloudClientID == "" {
			return fmt.Errorf("Camunda Cloud client ID is required when cloud version is enabled")
		}
		if c.Camunda.CloudClientSecret == "" {
			return fmt.Errorf("Camunda Cloud client secret is required when cloud version is enabled")
		}
		if c.Camunda.CloudClusterID == "" {
			return fmt.Errorf("Camunda Cloud cluster ID is required when cloud version is enabled")
		}
	}

	return nil
}

// GetPerformanceTargets returns performance targets as a map
func (c *Config) GetPerformanceTargets() map[string]int64 {
	return map[string]int64{
		"calculate_ms": c.Performance.CalculateTargetMs,
		"validate_ms":  c.Performance.ValidateTargetMs,
		"commit_ms":    c.Performance.CommitTargetMs,
		"total_ms":     c.Performance.TotalTargetMs,
	}
}