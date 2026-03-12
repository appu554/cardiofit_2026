package config

import (
	"fmt"
	"time"

	"github.com/spf13/viper"
)

// Config holds all configuration for the medication service
type Config struct {
	Service          ServiceConfig          `mapstructure:"service"`
	Server           ServerConfig           `mapstructure:"server"`
	Database         DatabaseConfig         `mapstructure:"database"`
	Redis            RedisConfig            `mapstructure:"redis"`
	ExternalServices ExternalServicesConfig `mapstructure:"external_services"`
	RecipeResolver   RecipeResolverConfig   `mapstructure:"recipe_resolver"`
	ContextGateway   ContextGatewayConfig   `mapstructure:"context_gateway"`
	ContextIntegration ContextIntegrationConfig `mapstructure:"context_integration"`
	ClinicalEngine   ClinicalEngineConfig   `mapstructure:"clinical_engine"`
	KnowledgeBases   KnowledgeBasesConfig   `mapstructure:"knowledge_bases"`
	GoogleFHIR       GoogleFHIRConfig       `mapstructure:"google_fhir"`

	// 4-Phase Workflow Orchestration Configuration
	WorkflowOrchestrator WorkflowOrchestratorConfig `mapstructure:"workflow_orchestrator"`
	ClinicalIntelligence ClinicalIntelligenceConfig `mapstructure:"clinical_intelligence"`
	ProposalGeneration   ProposalGenerationConfig   `mapstructure:"proposal_generation"`
	WorkflowState        WorkflowStateServiceConfig `mapstructure:"workflow_state"`
	MetricsService       MetricsServiceConfig       `mapstructure:"metrics_service"`
	
	Monitoring       MonitoringConfig       `mapstructure:"monitoring"`
	Logging          LoggingConfig          `mapstructure:"logging"`
	Performance      PerformanceConfig      `mapstructure:"performance"`
}

// ServiceConfig contains basic service information
type ServiceConfig struct {
	Name    string `mapstructure:"name" default:"medication-service-v2"`
	Version string `mapstructure:"version" default:"1.0.0"`
	Port    string `mapstructure:"port" default:"8005"`
}

// ServerConfig contains server-specific configuration
type ServerConfig struct {
	HTTP HTTPServerConfig `mapstructure:"http"`
	GRPC GRPCServerConfig `mapstructure:"grpc"`
}

// HTTPServerConfig contains HTTP server configuration
type HTTPServerConfig struct {
	Port         string        `mapstructure:"port" default:"8005"`
	Host         string        `mapstructure:"host" default:"0.0.0.0"`
	ReadTimeout  time.Duration `mapstructure:"read_timeout" default:"30s"`
	WriteTimeout time.Duration `mapstructure:"write_timeout" default:"30s"`
	IdleTimeout  time.Duration `mapstructure:"idle_timeout" default:"120s"`
	CORS         CORSConfig    `mapstructure:"cors"`
}

// GRPCServerConfig contains gRPC server configuration
type GRPCServerConfig struct {
	Port string `mapstructure:"port" default:"50005"`
	Host string `mapstructure:"host" default:"0.0.0.0"`
}

// CORSConfig contains CORS configuration
type CORSConfig struct {
	AllowedOrigins   []string      `mapstructure:"allowed_origins"`
	AllowedMethods   []string      `mapstructure:"allowed_methods"`
	AllowedHeaders   []string      `mapstructure:"allowed_headers"`
	AllowCredentials bool          `mapstructure:"allow_credentials"`
	MaxAge           time.Duration `mapstructure:"max_age" default:"24h"`
}

// DatabaseConfig contains database configuration
type DatabaseConfig struct {
	URL             string        `mapstructure:"url"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" default:"25"`
	MaxIdleConns    int           `mapstructure:"max_idle_conns" default:"5"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" default:"1h"`
	ConnMaxIdleTime time.Duration `mapstructure:"conn_max_idle_time" default:"30m"`
	MigrationsPath  string        `mapstructure:"migrations_path" default:"./migrations"`
}

// RedisConfig contains Redis configuration
type RedisConfig struct {
	URL         string        `mapstructure:"url"`
	Password    string        `mapstructure:"password"`
	DB          int           `mapstructure:"db" default:"0"`
	PoolSize    int           `mapstructure:"pool_size" default:"10"`
	MaxRetries  int           `mapstructure:"max_retries" default:"3"`
	DialTimeout time.Duration `mapstructure:"dial_timeout" default:"5s"`
	
	// Multi-level caching configuration
	MultiLevelCache MultiLevelCacheConfig `mapstructure:"multi_level_cache"`
}

// MultiLevelCacheConfig contains multi-level cache configuration
type MultiLevelCacheConfig struct {
	Enabled            bool          `mapstructure:"enabled" default:"true"`
	L1CacheSize        int64         `mapstructure:"l1_cache_size" default:"1000"`
	L1TTL              time.Duration `mapstructure:"l1_ttl" default:"5m"`
	L2TTL              time.Duration `mapstructure:"l2_ttl" default:"1h"`
	PromotionThreshold int64         `mapstructure:"promotion_threshold" default:"3"`
	DemotionTimeout    time.Duration `mapstructure:"demotion_timeout" default:"15m"`
	EncryptionEnabled  bool          `mapstructure:"encryption_enabled" default:"true"`
	AuditEnabled       bool          `mapstructure:"audit_enabled" default:"true"`
	
	// Performance optimization
	PerformanceOpt    bool   `mapstructure:"performance_optimization" default:"true"`
	OptimizeForLatency bool  `mapstructure:"optimize_for_latency" default:"true"`
	HotCacheSize      int64  `mapstructure:"hot_cache_size" default:"500"`
	HotCacheTTL       time.Duration `mapstructure:"hot_cache_ttl" default:"10m"`
	
	// Analytics and monitoring
	AnalyticsEnabled  bool `mapstructure:"analytics_enabled" default:"true"`
	MonitoringEnabled bool `mapstructure:"monitoring_enabled" default:"true"`
	
	// Cache warming
	WarmupEnabled     bool          `mapstructure:"warmup_enabled" default:"true"`
	WarmupInterval    time.Duration `mapstructure:"warmup_interval" default:"15m"`
}

// ExternalServicesConfig contains external service configurations
type ExternalServicesConfig struct {
	ContextGateway    ExternalServiceConfig `mapstructure:"context_gateway"`
	ApolloFederation  ExternalServiceConfig `mapstructure:"apollo_federation"`
	RustEngine        ExternalServiceConfig `mapstructure:"rust_engine"`
	Flow2GoEngine     ExternalServiceConfig `mapstructure:"flow2_go_engine"`
	SafetyGateway     ExternalServiceConfig `mapstructure:"safety_gateway"`
}

// ExternalServiceConfig contains individual external service configuration
type ExternalServiceConfig struct {
	URL        string        `mapstructure:"url"`
	Timeout    time.Duration `mapstructure:"timeout" default:"30s"`
	MaxRetries int           `mapstructure:"max_retries" default:"3"`
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// CircuitBreakerConfig contains circuit breaker configuration
type CircuitBreakerConfig struct {
	Enabled           bool          `mapstructure:"enabled" default:"true"`
	MaxRequests       uint32        `mapstructure:"max_requests" default:"3"`
	Interval          time.Duration `mapstructure:"interval" default:"60s"`
	Timeout           time.Duration `mapstructure:"timeout" default:"10s"`
	FailureThreshold  float64       `mapstructure:"failure_threshold" default:"0.6"`
}

// RecipeResolverConfig contains recipe resolver configuration
type RecipeResolverConfig struct {
	CacheEnabled    bool          `mapstructure:"cache_enabled" default:"true"`
	CacheTTL        time.Duration `mapstructure:"cache_ttl" default:"10m"`
	DefaultTTL      time.Duration `mapstructure:"default_ttl" default:"1h"`
	MaxRecipeSize   int           `mapstructure:"max_recipe_size" default:"1048576"` // 1MB
	ValidationLevel string        `mapstructure:"validation_level" default:"strict"`
}

// ClinicalEngineConfig contains clinical engine configuration
type ClinicalEngineConfig struct {
	RustEngineURL         string        `mapstructure:"rust_engine_url" default:"http://localhost:8095"`
	Timeout               time.Duration `mapstructure:"timeout" default:"30s"`
	MaxRetries            int           `mapstructure:"max_retries" default:"3"`
	MaxConcurrentRequests int           `mapstructure:"max_concurrent_requests" default:"100"`
	PerformanceTargets    PerformanceTargetsConfig `mapstructure:"performance_targets"`
}

// PerformanceTargetsConfig contains performance target configuration
type PerformanceTargetsConfig struct {
	EndToEndLatencyP95    time.Duration `mapstructure:"end_to_end_latency_p95" default:"250ms"`
	RecipeResolution      time.Duration `mapstructure:"recipe_resolution" default:"10ms"`
	SnapshotCreation      time.Duration `mapstructure:"snapshot_creation" default:"100ms"`
	ClinicalCalculations  time.Duration `mapstructure:"clinical_calculations" default:"50ms"`
	TargetThroughputRPS   int           `mapstructure:"target_throughput_rps" default:"1000"`
}

// KnowledgeBasesConfig contains knowledge base configuration
type KnowledgeBasesConfig struct {
	DrugRulesURL           string `mapstructure:"drug_rules_url" default:"http://localhost:8086"`
	GuidelinesURL          string `mapstructure:"guidelines_url" default:"http://localhost:8089"`
	ApolloFederationURL    string `mapstructure:"apollo_federation_url" default:"http://localhost:4000/graphql"`
	CacheEnabled           bool   `mapstructure:"cache_enabled" default:"true"`
	CacheTTL               time.Duration `mapstructure:"cache_ttl" default:"30m"`
}

// MonitoringConfig contains monitoring configuration
type MonitoringConfig struct {
	Enabled         bool   `mapstructure:"enabled" default:"true"`
	MetricsPath     string `mapstructure:"metrics_path" default:"/metrics"`
	HealthPath      string `mapstructure:"health_path" default:"/health"`
	JaegerEndpoint  string `mapstructure:"jaeger_endpoint"`
	SampleRate      float64 `mapstructure:"sample_rate" default:"0.1"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level       string `mapstructure:"level" default:"info"`
	Format      string `mapstructure:"format" default:"json"`
	Development bool   `mapstructure:"development" default:"false"`
	AuditTrail  AuditTrailConfig `mapstructure:"audit_trail"`
}

// AuditTrailConfig contains audit trail configuration for HIPAA compliance
type AuditTrailConfig struct {
	Enabled    bool   `mapstructure:"enabled" default:"true"`
	LogPath    string `mapstructure:"log_path" default:"./logs/audit.log"`
	MaxSize    int    `mapstructure:"max_size" default:"100"` // MB
	MaxBackups int    `mapstructure:"max_backups" default:"30"`
	MaxAge     int    `mapstructure:"max_age" default:"90"` // days
}

// PerformanceConfig contains performance optimization configuration
type PerformanceConfig struct {
	MaxConcurrentCalculations int           `mapstructure:"max_concurrent_calculations" default:"100"`
	CacheTTL                  time.Duration `mapstructure:"cache_ttl" default:"300s"`
	SnapshotExpiryHours       int           `mapstructure:"snapshot_expiry_hours" default:"24"`
	ConnectionPooling         ConnectionPoolingConfig `mapstructure:"connection_pooling"`
}

// ConnectionPoolingConfig contains connection pooling configuration
type ConnectionPoolingConfig struct {
	MaxIdleConns    int           `mapstructure:"max_idle_conns" default:"10"`
	MaxOpenConns    int           `mapstructure:"max_open_conns" default:"100"`
	ConnMaxLifetime time.Duration `mapstructure:"conn_max_lifetime" default:"1h"`
}

// ContextGatewayConfig contains Context Gateway service configuration
type ContextGatewayConfig struct {
	// Snapshot settings
	DefaultSnapshotTTL          time.Duration            `mapstructure:"default_snapshot_ttl" default:"24h"`
	FreshnessRequirements       map[string]time.Duration `mapstructure:"freshness_requirements"`
	SnapshotCreationTimeout     time.Duration            `mapstructure:"snapshot_creation_timeout" default:"30s"`
	
	// Retry settings
	MaxRetries                  int                      `mapstructure:"max_retries" default:"3"`
	RetryBackoffMultiplier      float64                  `mapstructure:"retry_backoff_multiplier" default:"2.0"`
	InitialRetryDelay          time.Duration            `mapstructure:"initial_retry_delay" default:"1s"`
	
	// Quality settings
	MinRequiredQualityScore     float64                  `mapstructure:"min_required_quality_score" default:"0.7"`
	RequiredFields             []string                 `mapstructure:"required_fields"`
	OptionalFields             []string                 `mapstructure:"optional_fields"`
	
	// Performance settings
	EnableAsyncSnapshotCreation bool                     `mapstructure:"enable_async_snapshot_creation" default:"false"`
	SnapshotCreationWorkers     int                      `mapstructure:"snapshot_creation_workers" default:"5"`
	
	// Validation settings
	EnableSnapshotValidation    bool                     `mapstructure:"enable_snapshot_validation" default:"true"`
	ValidationLevel            string                   `mapstructure:"validation_level" default:"standard"`
}

// ContextIntegrationConfig contains Context Integration workflow configuration
type ContextIntegrationConfig struct {
	// Workflow settings
	EnableSnapshotCreation      bool                     `mapstructure:"enable_snapshot_creation" default:"true"`
	AutoCreateSnapshots         bool                     `mapstructure:"auto_create_snapshots" default:"true"`
	SnapshotCreationMode        string                   `mapstructure:"snapshot_creation_mode" default:"sync"`
	
	// Performance settings
	MaxConcurrentSnapshots      int                      `mapstructure:"max_concurrent_snapshots" default:"10"`
	SnapshotCreationTimeout     time.Duration            `mapstructure:"snapshot_creation_timeout" default:"30s"`
	
	// Quality gates
	MinResolutionQuality        float64                  `mapstructure:"min_resolution_quality" default:"0.6"`
	RequireValidatedSnapshots   bool                     `mapstructure:"require_validated_snapshots" default:"false"`
	
	// Failure handling
	ContinueOnSnapshotFailure   bool                     `mapstructure:"continue_on_snapshot_failure" default:"true"`
	RetryFailedSnapshots        bool                     `mapstructure:"retry_failed_snapshots" default:"true"`
	
	// Snapshot lifecycle
	EnableSnapshotSupersession  bool                     `mapstructure:"enable_snapshot_supersession" default:"true"`
	CleanupSupersededSnapshots  bool                     `mapstructure:"cleanup_superseded_snapshots" default:"false"`
}

// Load loads configuration from files and environment variables
func Load() (*Config, error) {
	v := viper.New()

	// Set configuration file name and paths
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./config")
	v.AddConfigPath(".")

	// Set environment variable prefix
	v.SetEnvPrefix("MEDICATION_SERVICE")
	v.AutomaticEnv()

	// Set default values
	setDefaults(v)

	// Read configuration file
	if err := v.ReadInConfig(); err != nil {
		// Configuration file is optional, continue with defaults and env vars
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("error reading config file: %w", err)
		}
	}

	// Unmarshal configuration
	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

	// Validate configuration
	if err := validate(&cfg); err != nil {
		return nil, fmt.Errorf("config validation failed: %w", err)
	}

	return &cfg, nil
}

// setDefaults sets default configuration values
func setDefaults(v *viper.Viper) {
	// Service defaults
	v.SetDefault("service.name", "medication-service-v2")
	v.SetDefault("service.version", "1.0.0")
	v.SetDefault("service.port", "8005")

	// Server defaults
	v.SetDefault("server.http.port", "8005")
	v.SetDefault("server.http.host", "0.0.0.0")
	v.SetDefault("server.grpc.port", "50005")

	// External services defaults
	v.SetDefault("external_services.context_gateway.url", "http://localhost:8020")
	v.SetDefault("external_services.apollo_federation.url", "http://localhost:4000/graphql")
	v.SetDefault("external_services.rust_engine.url", "http://localhost:8095")
	v.SetDefault("external_services.flow2_go_engine.url", "http://localhost:8085")
	v.SetDefault("external_services.safety_gateway.url", "http://localhost:8030")

	// Performance defaults
	v.SetDefault("performance.max_concurrent_calculations", 100)
	v.SetDefault("performance.cache_ttl", "300s")
	v.SetDefault("performance.snapshot_expiry_hours", 24)

	// Context Gateway defaults
	v.SetDefault("context_gateway.default_snapshot_ttl", "24h")
	v.SetDefault("context_gateway.snapshot_creation_timeout", "30s")
	v.SetDefault("context_gateway.max_retries", 3)
	v.SetDefault("context_gateway.retry_backoff_multiplier", 2.0)
	v.SetDefault("context_gateway.initial_retry_delay", "1s")
	v.SetDefault("context_gateway.min_required_quality_score", 0.7)
	v.SetDefault("context_gateway.required_fields", []string{"demographics", "medications", "allergies"})
	v.SetDefault("context_gateway.optional_fields", []string{"vital_signs", "lab_results", "conditions"})
	v.SetDefault("context_gateway.enable_async_snapshot_creation", false)
	v.SetDefault("context_gateway.snapshot_creation_workers", 5)
	v.SetDefault("context_gateway.enable_snapshot_validation", true)
	v.SetDefault("context_gateway.validation_level", "standard")
	
	// Context Gateway freshness requirements
	v.SetDefault("context_gateway.freshness_requirements.demographics", "168h")      // 7 days
	v.SetDefault("context_gateway.freshness_requirements.vital_signs", "4h")         // 4 hours  
	v.SetDefault("context_gateway.freshness_requirements.lab_results", "24h")        // 24 hours
	v.SetDefault("context_gateway.freshness_requirements.medications", "1h")         // 1 hour
	v.SetDefault("context_gateway.freshness_requirements.allergies", "720h")         // 30 days
	v.SetDefault("context_gateway.freshness_requirements.conditions", "168h")        // 7 days

	// Context Integration defaults
	v.SetDefault("context_integration.enable_snapshot_creation", true)
	v.SetDefault("context_integration.auto_create_snapshots", true)
	v.SetDefault("context_integration.snapshot_creation_mode", "sync")
	v.SetDefault("context_integration.max_concurrent_snapshots", 10)
	v.SetDefault("context_integration.snapshot_creation_timeout", "30s")
	v.SetDefault("context_integration.min_resolution_quality", 0.6)
	v.SetDefault("context_integration.require_validated_snapshots", false)
	v.SetDefault("context_integration.continue_on_snapshot_failure", true)
	v.SetDefault("context_integration.retry_failed_snapshots", true)
	v.SetDefault("context_integration.enable_snapshot_supersession", true)
	v.SetDefault("context_integration.cleanup_superseded_snapshots", false)

	// 4-Phase Workflow Orchestration defaults
	v.SetDefault("workflow_orchestrator.default_timeout_per_phase", "30s")
	v.SetDefault("workflow_orchestrator.max_concurrent_workflows", 50)
	v.SetDefault("workflow_orchestrator.enable_parallel_phases", true)
	v.SetDefault("workflow_orchestrator.default_max_retries", 3)
	v.SetDefault("workflow_orchestrator.performance_target", "250ms")
	v.SetDefault("workflow_orchestrator.quality_threshold", 0.8)
	v.SetDefault("workflow_orchestrator.enable_state_persistence", true)
	v.SetDefault("workflow_orchestrator.state_cleanup_interval", "1h")
	v.SetDefault("workflow_orchestrator.max_retained_states", 1000)
	
	// Clinical Intelligence defaults
	v.SetDefault("clinical_intelligence.enable_rule_engines", []string{"rust_engine", "knowledge_base"})
	v.SetDefault("clinical_intelligence.rust_engine_url", "http://localhost:8095")
	v.SetDefault("clinical_intelligence.default_quality_threshold", 0.8)
	v.SetDefault("clinical_intelligence.enable_risk_assessment", true)
	v.SetDefault("clinical_intelligence.enable_safety_checks", true)
	v.SetDefault("clinical_intelligence.max_rule_engines", 5)
	v.SetDefault("clinical_intelligence.rule_evaluation_timeout", "10s")
	v.SetDefault("clinical_intelligence.risk_assessment_timeout", "5s")
	v.SetDefault("clinical_intelligence.safety_check_timeout", "5s")
	v.SetDefault("clinical_intelligence.enable_parallel_processing", true)
	v.SetDefault("clinical_intelligence.max_concurrent_processors", 3)
	
	// Proposal Generation defaults
	v.SetDefault("proposal_generation.max_proposals", 5)
	v.SetDefault("proposal_generation.min_quality_threshold", 0.7)
	v.SetDefault("proposal_generation.enable_alternative_analysis", true)
	v.SetDefault("proposal_generation.enable_fhir_validation", true)
	v.SetDefault("proposal_generation.fhir_validation_profile", "us-core")
	v.SetDefault("proposal_generation.enable_cost_analysis", false)
	v.SetDefault("proposal_generation.proposal_generation_timeout", "15s")
	v.SetDefault("proposal_generation.safety_check_timeout", "5s")
	v.SetDefault("proposal_generation.fhir_validation_timeout", "5s")
	v.SetDefault("proposal_generation.alternative_analysis_timeout", "10s")
	v.SetDefault("proposal_generation.enable_parallel_generation", true)
	v.SetDefault("proposal_generation.max_concurrent_generators", 3)
	v.SetDefault("proposal_generation.require_evidence_based", true)
	
	// Workflow State Service defaults
	v.SetDefault("workflow_state.default_ttl", "24h")
	v.SetDefault("workflow_state.cleanup_interval", "1h")
	v.SetDefault("workflow_state.max_retained_states", 10000)
	v.SetDefault("workflow_state.enable_compression", true)
	v.SetDefault("workflow_state.enable_encryption", true)
	v.SetDefault("workflow_state.stats_cache_interval", "5m")
	v.SetDefault("workflow_state.enable_audit_logging", true)
	v.SetDefault("workflow_state.backup_interval", "6h")
	v.SetDefault("workflow_state.backup_retention", "30d")
	
	// Metrics Service defaults
	v.SetDefault("metrics_service.collection_interval", "30s")
	v.SetDefault("metrics_service.aggregation_window", "5m")
	v.SetDefault("metrics_service.retention_period", "24h")
	v.SetDefault("metrics_service.max_sample_size", 1000)
	v.SetDefault("metrics_service.enable_detailed_metrics", true)
	v.SetDefault("metrics_service.export_interval", "1m")
	v.SetDefault("metrics_service.export_enabled", false)

	// CORS defaults for healthcare environments
	v.SetDefault("server.http.cors.allowed_origins", []string{"http://localhost:4200", "https://*.cardiofit.health"})
	v.SetDefault("server.http.cors.allowed_methods", []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"})
	v.SetDefault("server.http.cors.allowed_headers", []string{"Content-Type", "Authorization", "X-Requested-With"})
	v.SetDefault("server.http.cors.allow_credentials", true)
}

// validate validates the configuration
func validate(cfg *Config) error {
	// Validate required fields
	if cfg.Database.URL == "" {
		return fmt.Errorf("database.url is required")
	}

	if cfg.Redis.URL == "" {
		return fmt.Errorf("redis.url is required")
	}

	// Validate performance targets are realistic for healthcare
	if cfg.ClinicalEngine.PerformanceTargets.EndToEndLatencyP95 > 1*time.Second {
		return fmt.Errorf("end-to-end latency target exceeds healthcare requirements (>1s)")
	}

	// Validate service ports don't conflict
	if cfg.Server.HTTP.Port == cfg.Server.GRPC.Port {
		return fmt.Errorf("HTTP and gRPC ports cannot be the same")
	}

	// Validate external service URLs
	if cfg.ExternalServices.RustEngine.URL == "" {
		return fmt.Errorf("rust engine URL is required")
	}

	return nil
}

// Missing config types for workflow orchestration
type WorkflowOrchestratorConfig struct {
	Enabled         bool          `mapstructure:"enabled"`
	MaxConcurrency  int           `mapstructure:"max_concurrency"`
	ProcessingQueue QueueConfig   `mapstructure:"processing_queue"`
	RetryPolicy     RetryConfig   `mapstructure:"retry_policy"`
}

type ClinicalIntelligenceConfig struct {
	Enabled               bool                    `mapstructure:"enabled"`
	AIModelEndpoint       string                  `mapstructure:"ai_model_endpoint"`
	ConfidenceThreshold   float64                 `mapstructure:"confidence_threshold"`
	MaxProcessingTime     time.Duration           `mapstructure:"max_processing_time"`
	ClinicalRulesEngine   ClinicalRulesConfig     `mapstructure:"clinical_rules_engine"`
}

type ProposalGenerationConfig struct {
	Enabled                 bool                  `mapstructure:"enabled"`
	MaxProposalsPerRequest  int                   `mapstructure:"max_proposals_per_request"`
	RankingAlgorithm        string                `mapstructure:"ranking_algorithm"`
	QualityFilters          QualityFilterConfig   `mapstructure:"quality_filters"`
}

type WorkflowStateServiceConfig struct {
	Enabled             bool            `mapstructure:"enabled"`
	StateStore          StateStoreConfig `mapstructure:"state_store"`
	StatePersistence    bool            `mapstructure:"state_persistence"`
	MaxStateRetention   time.Duration   `mapstructure:"max_state_retention"`
}

type MetricsServiceConfig struct {
	Enabled         bool            `mapstructure:"enabled"`
	MetricsBackend  string          `mapstructure:"metrics_backend"`
	ReportingPeriod time.Duration   `mapstructure:"reporting_period"`
	CustomMetrics   []string        `mapstructure:"custom_metrics"`
}

// Supporting config types
type QueueConfig struct {
	Type        string `mapstructure:"type"`
	MaxSize     int    `mapstructure:"max_size"`
	Workers     int    `mapstructure:"workers"`
}

type RetryConfig struct {
	MaxRetries  int           `mapstructure:"max_retries"`
	BackoffTime time.Duration `mapstructure:"backoff_time"`
}

type ClinicalRulesConfig struct {
	RulesEngine string `mapstructure:"rules_engine"`
	RulesPath   string `mapstructure:"rules_path"`
}

type QualityFilterConfig struct {
	MinConfidence   float64 `mapstructure:"min_confidence"`
	RequireEvidence bool    `mapstructure:"require_evidence"`
}

type StateStoreConfig struct {
	Type       string `mapstructure:"type"`
	Connection string `mapstructure:"connection"`
	TTL        time.Duration `mapstructure:"ttl"`
}

// GoogleFHIRConfig contains Google Cloud Healthcare API configuration
type GoogleFHIRConfig struct {
	Enabled         bool   `mapstructure:"enabled" default:"true"`
	ProjectID       string `mapstructure:"project_id" default:"cardiofit-905a8"`
	Location        string `mapstructure:"location" default:"asia-south1"`
	DatasetID       string `mapstructure:"dataset_id" default:"clinical-synthesis-hub"`
	FHIRStoreID     string `mapstructure:"fhir_store_id" default:"fhir-store"`
	CredentialsPath string `mapstructure:"credentials_path" default:"credentials/google-credentials.json"`
}