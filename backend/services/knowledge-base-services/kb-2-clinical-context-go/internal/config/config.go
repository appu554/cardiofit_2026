package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

type Config struct {
	// Server configuration
	Port        int    `json:"port"`
	Environment string `json:"environment"`
	
	// Database configuration
	DatabaseURL   string `json:"database_url"`
	DatabaseName  string `json:"database_name"`
	
	// Redis configuration
	RedisURL      string `json:"redis_url"`
	RedisPassword string `json:"redis_password"`
	RedisDB       int    `json:"redis_db"`
	
	// CEL Engine configuration
	CELTimeout                int `json:"cel_timeout"`
	MaxCELExpressionComplexity int `json:"max_cel_expression_complexity"`
	
	// Performance configuration
	BatchSize             int `json:"batch_size"`
	MaxConcurrentRequests int `json:"max_concurrent_requests"`
	CacheTimeout          int `json:"cache_timeout"`
	
	// Multi-Tier Cache Configuration
	L1CacheMaxSize       int64         `json:"l1_cache_max_size"`        // 100MB
	L1CacheDefaultTTL    time.Duration `json:"l1_cache_default_ttl"`     // 5 minutes
	L1CacheMaxItems      int           `json:"l1_cache_max_items"`       // 10,000
	L1CacheHitRateTarget float64       `json:"l1_cache_hit_rate_target"` // 0.85
	
	L2CacheMaxMemory     int64         `json:"l2_cache_max_memory"`      // 1GB
	L2CacheDefaultTTL    time.Duration `json:"l2_cache_default_ttl"`     // 1 hour
	L2CacheCompression   bool          `json:"l2_cache_compression"`     // true
	L2CacheHitRateTarget float64       `json:"l2_cache_hit_rate_target"` // 0.95
	
	L3CacheBaseURL       string        `json:"l3_cache_base_url"`        // CDN URL
	L3CacheVersionPrefix string        `json:"l3_cache_version_prefix"`  // "v1"
	L3CacheEnabled       bool          `json:"l3_cache_enabled"`         // true
	
	// Cache warming configuration
	CacheWarmingEnabled   bool          `json:"cache_warming_enabled"`    // true
	CacheWarmingInterval  time.Duration `json:"cache_warming_interval"`   // 15 minutes
	CacheWarmingBatchSize int           `json:"cache_warming_batch_size"` // 100
	
	// Performance targets
	TargetLatencyP50     int `json:"target_latency_p50"`     // 5ms
	TargetLatencyP95     int `json:"target_latency_p95"`     // 25ms
	TargetLatencyP99     int `json:"target_latency_p99"`     // 100ms
	TargetThroughputRPS  int `json:"target_throughput_rps"`  // 10,000 RPS
	TargetBatchTime      int `json:"target_batch_time"`      // 1000ms for 1000 patients
	
	// Feature flags
	EnableCaching       bool `json:"enable_caching"`
	EnableMetrics       bool `json:"enable_metrics"`
	StrictValidation    bool `json:"strict_validation"`
	EnableTracing       bool `json:"enable_tracing"`
	EnableMultiTierCache bool `json:"enable_multi_tier_cache"`
	
	// Knowledge base paths
	PhenotypesPath         string `json:"phenotypes_path"`
	RiskModelsPath         string `json:"risk_models_path"`
	TreatmentPreferencesPath string `json:"treatment_preferences_path"`
	
	// SLA configuration
	PhenotypeEvaluationSLA     int `json:"phenotype_evaluation_sla"`     // 100ms
	PhenotypeExplanationSLA    int `json:"phenotype_explanation_sla"`    // 150ms
	RiskAssessmentSLA          int `json:"risk_assessment_sla"`           // 200ms
	TreatmentPreferencesSLA    int `json:"treatment_preferences_sla"`     // 50ms
	ContextAssemblySLA         int `json:"context_assembly_sla"`          // 200ms
}

func LoadConfig() (*Config, error) {
	return &Config{
		// Server defaults
		Port:        getIntEnv("PORT", 8088),
		Environment: getStringEnv("ENVIRONMENT", "development"),
		
		// Database defaults
		DatabaseURL:  getStringEnv("DATABASE_URL", "mongodb://localhost:27017"),
		DatabaseName: getStringEnv("DATABASE_NAME", "kb_clinical_context"),
		
		// Redis defaults
		RedisURL:      getStringEnv("REDIS_URL", "localhost:6379"),
		RedisPassword: getStringEnv("REDIS_PASSWORD", ""),
		RedisDB:       getIntEnv("REDIS_DB", 0),
		
		// CEL Engine defaults
		CELTimeout:                getIntEnv("CEL_TIMEOUT", 1000),
		MaxCELExpressionComplexity: getIntEnv("MAX_CEL_EXPRESSION_COMPLEXITY", 100),
		
		// Performance defaults
		BatchSize:             getIntEnv("BATCH_SIZE", 1000),
		MaxConcurrentRequests: getIntEnv("MAX_CONCURRENT_REQUESTS", 100),
		CacheTimeout:          getIntEnv("CACHE_TIMEOUT", 3600),
		
		// Multi-Tier Cache Configuration
		L1CacheMaxSize:       getInt64Env("L1_CACHE_MAX_SIZE", 100*1024*1024),        // 100MB
		L1CacheDefaultTTL:    getDurationEnv("L1_CACHE_DEFAULT_TTL", 5*time.Minute),   // 5 minutes
		L1CacheMaxItems:      getIntEnv("L1_CACHE_MAX_ITEMS", 10000),                // 10,000
		L1CacheHitRateTarget: getFloat64Env("L1_CACHE_HIT_RATE_TARGET", 0.85),       // 85%
		
		L2CacheMaxMemory:     getInt64Env("L2_CACHE_MAX_MEMORY", 1024*1024*1024),     // 1GB
		L2CacheDefaultTTL:    getDurationEnv("L2_CACHE_DEFAULT_TTL", time.Hour),      // 1 hour
		L2CacheCompression:   getBoolEnv("L2_CACHE_COMPRESSION", true),               // true
		L2CacheHitRateTarget: getFloat64Env("L2_CACHE_HIT_RATE_TARGET", 0.95),       // 95%
		
		L3CacheBaseURL:       getStringEnv("L3_CACHE_BASE_URL", "https://cdn.clinical-hub.com"),
		L3CacheVersionPrefix: getStringEnv("L3_CACHE_VERSION_PREFIX", "v1"),
		L3CacheEnabled:       getBoolEnv("L3_CACHE_ENABLED", true),
		
		// Cache warming configuration
		CacheWarmingEnabled:   getBoolEnv("CACHE_WARMING_ENABLED", true),
		CacheWarmingInterval:  getDurationEnv("CACHE_WARMING_INTERVAL", 15*time.Minute),
		CacheWarmingBatchSize: getIntEnv("CACHE_WARMING_BATCH_SIZE", 100),
		
		// Performance targets
		TargetLatencyP50:    getIntEnv("TARGET_LATENCY_P50", 5),     // 5ms
		TargetLatencyP95:    getIntEnv("TARGET_LATENCY_P95", 25),    // 25ms
		TargetLatencyP99:    getIntEnv("TARGET_LATENCY_P99", 100),   // 100ms
		TargetThroughputRPS: getIntEnv("TARGET_THROUGHPUT_RPS", 10000), // 10,000 RPS
		TargetBatchTime:     getIntEnv("TARGET_BATCH_TIME", 1000),   // 1000ms for 1000 patients
		
		// Feature flags defaults
		EnableCaching:        getBoolEnv("ENABLE_CACHING", true),
		EnableMetrics:        getBoolEnv("ENABLE_METRICS", true),
		StrictValidation:     getBoolEnv("STRICT_VALIDATION", true),
		EnableTracing:        getBoolEnv("ENABLE_TRACING", false),
		EnableMultiTierCache: getBoolEnv("ENABLE_MULTI_TIER_CACHE", true),
		
		// Knowledge base paths
		PhenotypesPath:         getStringEnv("PHENOTYPES_PATH", "./knowledge-base/phenotypes"),
		RiskModelsPath:         getStringEnv("RISK_MODELS_PATH", "./knowledge-base/risk-models"),
		TreatmentPreferencesPath: getStringEnv("TREATMENT_PREFERENCES_PATH", "./knowledge-base/treatment-preferences"),
		
		// SLA configuration (milliseconds)
		PhenotypeEvaluationSLA:     getIntEnv("PHENOTYPE_EVALUATION_SLA", 100),
		PhenotypeExplanationSLA:    getIntEnv("PHENOTYPE_EXPLANATION_SLA", 150),
		RiskAssessmentSLA:          getIntEnv("RISK_ASSESSMENT_SLA", 200),
		TreatmentPreferencesSLA:    getIntEnv("TREATMENT_PREFERENCES_SLA", 50),
		ContextAssemblySLA:         getIntEnv("CONTEXT_ASSEMBLY_SLA", 200),
	}, nil
}

// GetStringEnv helper method for multi-tier cache
func (c *Config) GetStringEnv(key, defaultValue string) string {
	return getStringEnv(key, defaultValue)
}

// GetCacheConfig returns cache-specific configuration
func (c *Config) GetCacheConfig() CacheConfig {
	return CacheConfig{
		// L1 Cache (Memory)
		L1: L1CacheConfig{
			MaxSize:       c.L1CacheMaxSize,
			DefaultTTL:    c.L1CacheDefaultTTL,
			MaxItems:      c.L1CacheMaxItems,
			HitRateTarget: c.L1CacheHitRateTarget,
			EvictionRate:  0.1, // 10% eviction rate when full
		},
		
		// L2 Cache (Redis)
		L2: L2CacheConfig{
			MaxMemory:     c.L2CacheMaxMemory,
			DefaultTTL:    c.L2CacheDefaultTTL,
			Compression:   c.L2CacheCompression,
			HitRateTarget: c.L2CacheHitRateTarget,
			KeyPrefix:     "kb2:l2:",
		},
		
		// L3 Cache (CDN)
		L3: L3CacheConfig{
			BaseURL:       c.L3CacheBaseURL,
			VersionPrefix: c.L3CacheVersionPrefix,
			Enabled:       c.L3CacheEnabled,
			CacheHeaders: map[string]string{
				"Cache-Control": "public, max-age=86400, immutable",
				"ETag":          "\"clinical-definitions-v1\"",
			},
		},
		
		// Warming configuration
		Warming: CacheWarmingConfig{
			Enabled:   c.CacheWarmingEnabled,
			Interval:  c.CacheWarmingInterval,
			BatchSize: c.CacheWarmingBatchSize,
		},
		
		// Performance targets
		Performance: CachePerformanceConfig{
			LatencyP50:    time.Duration(c.TargetLatencyP50) * time.Millisecond,
			LatencyP95:    time.Duration(c.TargetLatencyP95) * time.Millisecond,
			LatencyP99:    time.Duration(c.TargetLatencyP99) * time.Millisecond,
			ThroughputRPS: c.TargetThroughputRPS,
			BatchTime:     time.Duration(c.TargetBatchTime) * time.Millisecond,
		},
	}
}

// CacheConfig represents comprehensive cache configuration
type CacheConfig struct {
	L1          L1CacheConfig          `json:"l1"`
	L2          L2CacheConfig          `json:"l2"`
	L3          L3CacheConfig          `json:"l3"`
	Warming     CacheWarmingConfig     `json:"warming"`
	Performance CachePerformanceConfig `json:"performance"`
}

// L1CacheConfig configures in-memory L1 cache
type L1CacheConfig struct {
	MaxSize       int64         `json:"max_size"`
	DefaultTTL    time.Duration `json:"default_ttl"`
	MaxItems      int           `json:"max_items"`
	HitRateTarget float64       `json:"hit_rate_target"`
	EvictionRate  float64       `json:"eviction_rate"`
}

// L2CacheConfig configures Redis L2 cache
type L2CacheConfig struct {
	MaxMemory     int64         `json:"max_memory"`
	DefaultTTL    time.Duration `json:"default_ttl"`
	Compression   bool          `json:"compression"`
	HitRateTarget float64       `json:"hit_rate_target"`
	KeyPrefix     string        `json:"key_prefix"`
}

// L3CacheConfig configures CDN L3 cache
type L3CacheConfig struct {
	BaseURL       string            `json:"base_url"`
	VersionPrefix string            `json:"version_prefix"`
	Enabled       bool              `json:"enabled"`
	CacheHeaders  map[string]string `json:"cache_headers"`
}

// CacheWarmingConfig configures cache warming
type CacheWarmingConfig struct {
	Enabled   bool          `json:"enabled"`
	Interval  time.Duration `json:"interval"`
	BatchSize int           `json:"batch_size"`
}

// CachePerformanceConfig defines performance targets
type CachePerformanceConfig struct {
	LatencyP50    time.Duration `json:"latency_p50"`
	LatencyP95    time.Duration `json:"latency_p95"`
	LatencyP99    time.Duration `json:"latency_p99"`
	ThroughputRPS int           `json:"throughput_rps"`
	BatchTime     time.Duration `json:"batch_time"`
}

// IsCacheOptimal checks if cache performance meets targets
func (cpc *CachePerformanceConfig) IsCacheOptimal(actualLatencyP95 time.Duration, actualThroughput int) bool {
	return actualLatencyP95 <= cpc.LatencyP95 && actualThroughput >= cpc.ThroughputRPS
}

// GetPerformanceScore calculates performance score (0.0 to 1.0)
func (cpc *CachePerformanceConfig) GetPerformanceScore(actualLatencyP95 time.Duration, actualThroughput int, hitRate float64) float64 {
	// Latency score (0.0 to 0.4)
	latencyScore := 0.0
	if actualLatencyP95 <= cpc.LatencyP95 {
		latencyScore = 0.4
	} else {
		// Partial score based on how close to target
		ratio := float64(cpc.LatencyP95) / float64(actualLatencyP95)
		latencyScore = 0.4 * ratio
	}
	
	// Throughput score (0.0 to 0.4)
	throughputScore := 0.0
	if actualThroughput >= cpc.ThroughputRPS {
		throughputScore = 0.4
	} else {
		// Partial score based on how close to target
		ratio := float64(actualThroughput) / float64(cpc.ThroughputRPS)
		throughputScore = 0.4 * ratio
	}
	
	// Hit rate score (0.0 to 0.2)
	hitRateScore := hitRate * 0.2
	
	return latencyScore + throughputScore + hitRateScore
}

func getStringEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intValue
		}
	}
	return defaultValue
}

func getFloat64Env(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if floatValue, err := strconv.ParseFloat(value, 64); err == nil {
			return floatValue
		}
	}
	return defaultValue
}

func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return strings.ToLower(value) == "true"
	}
	return defaultValue
}

func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}