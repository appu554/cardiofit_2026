package config

import (
	"fmt"
	"time"
)

// AdvancedOrchestrationConfig contains configuration for advanced orchestration features
type AdvancedOrchestrationConfig struct {
	// Core settings
	Enabled              bool          `yaml:"enabled"`
	MaxConcurrentRequests int          `yaml:"max_concurrent_requests"`
	RequestTimeout       time.Duration `yaml:"request_timeout"`
	
	// Batch processing configuration
	BatchProcessing      BatchProcessingConfig      `yaml:"batch_processing"`
	
	// Load balancing configuration
	LoadBalancing        LoadBalancingConfig        `yaml:"load_balancing"`
	
	// Routing configuration
	Routing              RoutingConfig              `yaml:"routing"`
	
	// Metrics and monitoring
	Metrics              OrchestrationMetricsConfig `yaml:"metrics"`
	
	// Performance optimization
	Performance          PerformanceConfig          `yaml:"performance"`
}

// BatchProcessingConfig defines batch processing parameters
type BatchProcessingConfig struct {
	Enabled           bool          `yaml:"enabled"`
	MaxBatchSize      int           `yaml:"max_batch_size"`
	BatchTimeout      time.Duration `yaml:"batch_timeout"`
	Concurrency       int           `yaml:"concurrency"`
	PatientGrouping   bool          `yaml:"patient_grouping"`
	SnapshotOptimized bool          `yaml:"snapshot_optimized"`
}

// LoadBalancingConfig defines load balancing strategies and parameters
type LoadBalancingConfig struct {
	Strategy                string             `yaml:"strategy"` // round_robin, least_loaded, performance_weighted, adaptive
	EnableHealthCheck       bool               `yaml:"enable_health_check"`
	HealthCheckInterval     time.Duration      `yaml:"health_check_interval"`
	AdaptiveWeightDecay     float64            `yaml:"adaptive_weight_decay"`
	PerformanceWindowSize   int                `yaml:"performance_window_size"`
	EngineSelectionCriteria SelectionCriteria  `yaml:"engine_selection_criteria"`
}

// SelectionCriteria defines engine selection parameters
type SelectionCriteria struct {
	MaxErrorRate         float64 `yaml:"max_error_rate"`
	MaxAverageLatencyMs  int64   `yaml:"max_average_latency_ms"`
	MinThroughputPerSec  float64 `yaml:"min_throughput_per_sec"`
	LoadScoreThreshold   float64 `yaml:"load_score_threshold"`
}

// RoutingConfig defines intelligent routing configuration
type RoutingConfig struct {
	EnableIntelligentRouting bool                     `yaml:"enable_intelligent_routing"`
	DefaultTier              string                   `yaml:"default_tier"`
	EnginePriorities         map[string]int           `yaml:"engine_priorities"`
	FallbackChains           map[string][]string      `yaml:"fallback_chains"`
	RoutingRules             []RoutingRuleConfig      `yaml:"routing_rules"`
	DynamicRuleEvaluation    bool                     `yaml:"dynamic_rule_evaluation"`
}

// RoutingRuleConfig defines individual routing rules
type RoutingRuleConfig struct {
	Name        string            `yaml:"name"`
	Description string            `yaml:"description"`
	Condition   ConditionConfig   `yaml:"condition"`
	TargetTier  string            `yaml:"target_tier"`
	Priority    int               `yaml:"priority"`
	Enabled     bool              `yaml:"enabled"`
	Metadata    map[string]string `yaml:"metadata,omitempty"`
}

// ConditionConfig defines routing rule conditions
type ConditionConfig struct {
	Field    string      `yaml:"field"`     // priority, action_type, patient_id, etc.
	Operator string      `yaml:"operator"`  // equals, contains, gt, lt, in, not_in
	Value    interface{} `yaml:"value"`     // The value to compare against
	
	// Complex conditions
	And []ConditionConfig `yaml:"and,omitempty"`
	Or  []ConditionConfig `yaml:"or,omitempty"`
	Not *ConditionConfig  `yaml:"not,omitempty"`
}

// OrchestrationMetricsConfig defines metrics collection and reporting
type OrchestrationMetricsConfig struct {
	EnableMetrics          bool          `yaml:"enable_metrics"`
	MetricsInterval        time.Duration `yaml:"metrics_interval"`
	HistoryRetentionPeriod time.Duration `yaml:"history_retention_period"`
	
	// Metric categories
	EnablePerformanceMetrics  bool `yaml:"enable_performance_metrics"`
	EnableLoadMetrics         bool `yaml:"enable_load_metrics"`
	EnableRoutingMetrics      bool `yaml:"enable_routing_metrics"`
	EnableBatchMetrics        bool `yaml:"enable_batch_metrics"`
	
	// Export configuration
	ExportPrometheus      bool   `yaml:"export_prometheus"`
	PrometheusNamespace   string `yaml:"prometheus_namespace"`
	ExportJSON            bool   `yaml:"export_json"`
	JSONExportPath        string `yaml:"json_export_path"`
}

// PerformanceConfig defines performance optimization settings
type PerformanceConfig struct {
	EnablePerformanceOptimization bool          `yaml:"enable_performance_optimization"`
	AdaptiveThrottling           bool          `yaml:"adaptive_throttling"`
	CircuitBreakerThreshold      int           `yaml:"circuit_breaker_threshold"`
	PreemptiveScaling            bool          `yaml:"preemptive_scaling"`
	MemoryOptimization           bool          `yaml:"memory_optimization"`
	CPUOptimization              bool          `yaml:"cpu_optimization"`
	
	// Resource limits
	MaxMemoryMB              int           `yaml:"max_memory_mb"`
	MaxCPUCores              int           `yaml:"max_cpu_cores"`
	GoroutinePoolSize        int           `yaml:"goroutine_pool_size"`
	
	// Timeouts and intervals
	OptimizationInterval     time.Duration `yaml:"optimization_interval"`
	ResourceCheckInterval    time.Duration `yaml:"resource_check_interval"`
}

// GetDefaultAdvancedOrchestrationConfig returns default advanced orchestration configuration
func GetDefaultAdvancedOrchestrationConfig() *AdvancedOrchestrationConfig {
	return &AdvancedOrchestrationConfig{
		Enabled:              false, // Disabled by default
		MaxConcurrentRequests: 1000,
		RequestTimeout:       10 * time.Second,
		
		BatchProcessing: BatchProcessingConfig{
			Enabled:           true,
			MaxBatchSize:      50,
			BatchTimeout:      100 * time.Millisecond,
			Concurrency:       10,
			PatientGrouping:   true,
			SnapshotOptimized: true,
		},
		
		LoadBalancing: LoadBalancingConfig{
			Strategy:             "adaptive",
			EnableHealthCheck:    true,
			HealthCheckInterval:  30 * time.Second,
			AdaptiveWeightDecay:  0.1,
			PerformanceWindowSize: 100,
			EngineSelectionCriteria: SelectionCriteria{
				MaxErrorRate:        0.05, // 5% max error rate
				MaxAverageLatencyMs: 1000, // 1 second max latency
				MinThroughputPerSec: 1.0,  // 1 request per second minimum
				LoadScoreThreshold:  0.8,  // 80% load threshold
			},
		},
		
		Routing: RoutingConfig{
			EnableIntelligentRouting: true,
			DefaultTier:              "veto_critical",
			EnginePriorities:         getDefaultEnginePriorities(),
			FallbackChains:           getDefaultFallbackChains(),
			RoutingRules:             getDefaultRoutingRules(),
			DynamicRuleEvaluation:    true,
		},
		
		Metrics: OrchestrationMetricsConfig{
			EnableMetrics:             true,
			MetricsInterval:           10 * time.Second,
			HistoryRetentionPeriod:    24 * time.Hour,
			EnablePerformanceMetrics:  true,
			EnableLoadMetrics:         true,
			EnableRoutingMetrics:      true,
			EnableBatchMetrics:        true,
			ExportPrometheus:          false,
			PrometheusNamespace:       "safety_gateway",
			ExportJSON:                true,
			JSONExportPath:            "/tmp/orchestration_metrics.json",
		},
		
		Performance: PerformanceConfig{
			EnablePerformanceOptimization: true,
			AdaptiveThrottling:           true,
			CircuitBreakerThreshold:      10,
			PreemptiveScaling:            false,
			MemoryOptimization:           true,
			CPUOptimization:              true,
			MaxMemoryMB:                  1024, // 1GB
			MaxCPUCores:                  4,
			GoroutinePoolSize:            100,
			OptimizationInterval:         1 * time.Minute,
			ResourceCheckInterval:        10 * time.Second,
		},
	}
}

// Helper functions for default configurations

func getDefaultEnginePriorities() map[string]int {
	return map[string]int{
		"drug_interaction_engine":    100,
		"allergy_check_engine":       90,
		"dosage_validation_engine":   80,
		"contraindication_engine":    70,
		"clinical_advisory_engine":   50,
	}
}

func getDefaultFallbackChains() map[string][]string {
	return map[string][]string{
		"veto_critical": {
			"drug_interaction_engine",
			"allergy_check_engine",
			"contraindication_engine",
		},
		"advisory": {
			"clinical_advisory_engine",
			"dosage_validation_engine",
		},
	}
}

func getDefaultRoutingRules() []RoutingRuleConfig {
	return []RoutingRuleConfig{
		{
			Name:        "critical_priority_routing",
			Description: "Route critical priority requests to veto engines",
			Condition: ConditionConfig{
				Field:    "priority",
				Operator: "in",
				Value:    []string{"critical", "high"},
			},
			TargetTier: "veto_critical",
			Priority:   100,
			Enabled:    true,
		},
		{
			Name:        "medication_interaction_routing",
			Description: "Route medication interaction checks to specialized engines",
			Condition: ConditionConfig{
				And: []ConditionConfig{
					{
						Field:    "action_type",
						Operator: "equals",
						Value:    "medication_interaction",
					},
					{
						Field:    "medication_count",
						Operator: "gt",
						Value:    1,
					},
				},
			},
			TargetTier: "veto_critical",
			Priority:   90,
			Enabled:    true,
		},
		{
			Name:        "routine_advisory_routing",
			Description: "Route routine requests to advisory engines",
			Condition: ConditionConfig{
				Field:    "priority",
				Operator: "in",
				Value:    []string{"routine", "low"},
			},
			TargetTier: "advisory",
			Priority:   10,
			Enabled:    true,
		},
		{
			Name:        "patient_specific_routing",
			Description: "Route specific patients to dedicated engine pools",
			Condition: ConditionConfig{
				Field:    "patient_id",
				Operator: "contains",
				Value:    "VIP_",
			},
			TargetTier: "veto_critical",
			Priority:   80,
			Enabled:    false, // Disabled by default
			Metadata: map[string]string{
				"engine_pool": "dedicated_vip",
				"sla_tier":    "premium",
			},
		},
	}
}

// Validation methods

func (c *AdvancedOrchestrationConfig) Validate() error {
	if c.MaxConcurrentRequests <= 0 {
		return fmt.Errorf("max_concurrent_requests must be positive, got %d", c.MaxConcurrentRequests)
	}
	
	if c.RequestTimeout <= 0 {
		return fmt.Errorf("request_timeout must be positive, got %v", c.RequestTimeout)
	}
	
	if err := c.BatchProcessing.Validate(); err != nil {
		return fmt.Errorf("batch_processing config validation failed: %w", err)
	}
	
	if err := c.LoadBalancing.Validate(); err != nil {
		return fmt.Errorf("load_balancing config validation failed: %w", err)
	}
	
	if err := c.Routing.Validate(); err != nil {
		return fmt.Errorf("routing config validation failed: %w", err)
	}
	
	if err := c.Performance.Validate(); err != nil {
		return fmt.Errorf("performance config validation failed: %w", err)
	}
	
	return nil
}

func (c *BatchProcessingConfig) Validate() error {
	if c.MaxBatchSize <= 0 {
		return fmt.Errorf("max_batch_size must be positive, got %d", c.MaxBatchSize)
	}
	
	if c.BatchTimeout <= 0 {
		return fmt.Errorf("batch_timeout must be positive, got %v", c.BatchTimeout)
	}
	
	if c.Concurrency <= 0 {
		return fmt.Errorf("concurrency must be positive, got %d", c.Concurrency)
	}
	
	return nil
}

func (c *LoadBalancingConfig) Validate() error {
	validStrategies := []string{"round_robin", "least_loaded", "performance_weighted", "adaptive"}
	valid := false
	for _, strategy := range validStrategies {
		if c.Strategy == strategy {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid load balancing strategy: %s, must be one of %v", c.Strategy, validStrategies)
	}
	
	if c.AdaptiveWeightDecay < 0 || c.AdaptiveWeightDecay > 1 {
		return fmt.Errorf("adaptive_weight_decay must be between 0 and 1, got %f", c.AdaptiveWeightDecay)
	}
	
	if c.PerformanceWindowSize <= 0 {
		return fmt.Errorf("performance_window_size must be positive, got %d", c.PerformanceWindowSize)
	}
	
	return c.EngineSelectionCriteria.Validate()
}

func (c *SelectionCriteria) Validate() error {
	if c.MaxErrorRate < 0 || c.MaxErrorRate > 1 {
		return fmt.Errorf("max_error_rate must be between 0 and 1, got %f", c.MaxErrorRate)
	}
	
	if c.MaxAverageLatencyMs <= 0 {
		return fmt.Errorf("max_average_latency_ms must be positive, got %d", c.MaxAverageLatencyMs)
	}
	
	if c.MinThroughputPerSec < 0 {
		return fmt.Errorf("min_throughput_per_sec cannot be negative, got %f", c.MinThroughputPerSec)
	}
	
	if c.LoadScoreThreshold < 0 || c.LoadScoreThreshold > 1 {
		return fmt.Errorf("load_score_threshold must be between 0 and 1, got %f", c.LoadScoreThreshold)
	}
	
	return nil
}

func (c *RoutingConfig) Validate() error {
	validTiers := []string{"veto_critical", "veto_advisory", "advisory"}
	valid := false
	for _, tier := range validTiers {
		if c.DefaultTier == tier {
			valid = true
			break
		}
	}
	if !valid {
		return fmt.Errorf("invalid default_tier: %s, must be one of %v", c.DefaultTier, validTiers)
	}
	
	// Validate routing rules
	for i, rule := range c.RoutingRules {
		if err := rule.Validate(); err != nil {
			return fmt.Errorf("routing rule %d validation failed: %w", i, err)
		}
	}
	
	return nil
}

func (c *RoutingRuleConfig) Validate() error {
	if c.Name == "" {
		return fmt.Errorf("routing rule name cannot be empty")
	}
	
	if c.Priority < 0 {
		return fmt.Errorf("routing rule priority cannot be negative, got %d", c.Priority)
	}
	
	return c.Condition.Validate()
}

func (c *ConditionConfig) Validate() error {
	if c.Field == "" && len(c.And) == 0 && len(c.Or) == 0 && c.Not == nil {
		return fmt.Errorf("condition must have either field or logical operators (and/or/not)")
	}
	
	if c.Field != "" {
		validOperators := []string{"equals", "contains", "gt", "lt", "gte", "lte", "in", "not_in"}
		valid := false
		for _, op := range validOperators {
			if c.Operator == op {
				valid = true
				break
			}
		}
		if !valid {
			return fmt.Errorf("invalid operator: %s, must be one of %v", c.Operator, validOperators)
		}
	}
	
	// Recursively validate nested conditions
	for i, cond := range c.And {
		if err := cond.Validate(); err != nil {
			return fmt.Errorf("and condition %d validation failed: %w", i, err)
		}
	}
	
	for i, cond := range c.Or {
		if err := cond.Validate(); err != nil {
			return fmt.Errorf("or condition %d validation failed: %w", i, err)
		}
	}
	
	if c.Not != nil {
		if err := c.Not.Validate(); err != nil {
			return fmt.Errorf("not condition validation failed: %w", err)
		}
	}
	
	return nil
}

func (c *PerformanceConfig) Validate() error {
	if c.MaxMemoryMB <= 0 {
		return fmt.Errorf("max_memory_mb must be positive, got %d", c.MaxMemoryMB)
	}
	
	if c.MaxCPUCores <= 0 {
		return fmt.Errorf("max_cpu_cores must be positive, got %d", c.MaxCPUCores)
	}
	
	if c.GoroutinePoolSize <= 0 {
		return fmt.Errorf("goroutine_pool_size must be positive, got %d", c.GoroutinePoolSize)
	}
	
	if c.OptimizationInterval <= 0 {
		return fmt.Errorf("optimization_interval must be positive, got %v", c.OptimizationInterval)
	}
	
	if c.ResourceCheckInterval <= 0 {
		return fmt.Errorf("resource_check_interval must be positive, got %v", c.ResourceCheckInterval)
	}
	
	return nil
}

// IsAdvancedOrchestrationEnabled returns whether advanced orchestration is enabled and properly configured
func (c *AdvancedOrchestrationConfig) IsAdvancedOrchestrationEnabled() bool {
	return c.Enabled
}

// GetBatchSize returns the configured batch size
func (c *AdvancedOrchestrationConfig) GetBatchSize() int {
	return c.BatchProcessing.MaxBatchSize
}

// GetLoadBalancingStrategy returns the configured load balancing strategy
func (c *AdvancedOrchestrationConfig) GetLoadBalancingStrategy() string {
	return c.LoadBalancing.Strategy
}

// IsIntelligentRoutingEnabled returns whether intelligent routing is enabled
func (c *AdvancedOrchestrationConfig) IsIntelligentRoutingEnabled() bool {
	return c.Routing.EnableIntelligentRouting
}