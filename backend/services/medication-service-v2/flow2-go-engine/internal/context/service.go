package context

import (
	"context"
	"time"

	"flow2-go-engine/orb"
)

// ContextIntegrationService is the main interface for the Context Integration Service
// This service replaces the external Python Context Service with a high-performance Go module
type ContextIntegrationService interface {
	// Core functionality
	AssembleContext(ctx context.Context, manifest *orb.IntentManifest) (*CompleteContextPayload, error)
	
	// Cache management
	InvalidateCache(patientID string) error
	InvalidateCachePattern(pattern string) error
	GetCacheStats() CacheStatistics
	
	// Health and monitoring
	HealthCheck() error
	GetMetrics() IntegrationMetrics
	
	// Configuration
	UpdateConfiguration(config ServiceConfiguration) error
	GetConfiguration() ServiceConfiguration
}

// KnowledgeBaseClient interface for communicating with KB services
type KnowledgeBaseClient interface {
	// Fetch data from specified Knowledge Bases
	FetchKnowledgeData(ctx context.Context, kbHints []string, patientID string, medicationCode string) (KnowledgeContext, error)
	
	// Get all available KB identifiers
	GetAllKBIdentifiers() []string
	
	// Health check for KB services
	HealthCheck() map[string]bool
	
	// Get KB service endpoints
	GetKBEndpoints() map[string]string
}

// ContextGatewayClient interface for communicating with existing Context Gateway Service
type ContextGatewayClient interface {
	// Fetch patient clinical data
	FetchPatientData(ctx context.Context, patientID string, dataRequirements []string) (PatientContext, error)
	
	// Health check for Context Gateway
	HealthCheck() error
	
	// Get service endpoint
	GetEndpoint() string
}

// CacheManager interface for L3 Redis caching
type CacheManager interface {
	// Core cache operations
	Get(ctx context.Context, key string) (*CompleteContextPayload, error)
	Set(ctx context.Context, key string, payload *CompleteContextPayload, ttl time.Duration) error
	
	// Stale-while-revalidate operations
	GetStale(ctx context.Context, key string) (*CompleteContextPayload, error)
	SetWithStale(ctx context.Context, key string, payload *CompleteContextPayload, ttl time.Duration, staleTTL time.Duration) error
	
	// Cache management
	Delete(ctx context.Context, key string) error
	DeletePattern(ctx context.Context, pattern string) error
	Exists(ctx context.Context, key string) bool
	
	// Statistics and monitoring
	GetStats() CacheStatistics
	HealthCheck() error
}

// CircuitBreaker interface for resilience patterns
type CircuitBreaker interface {
	// Execute a function with circuit breaker protection
	Execute(ctx context.Context, operation func() (interface{}, error)) (interface{}, error)
	
	// Get circuit breaker state
	GetState() CircuitBreakerState
	
	// Get circuit breaker metrics
	GetMetrics() CircuitBreakerMetrics
	
	// Reset circuit breaker
	Reset()
}

// Supporting data structures

// CacheStatistics contains cache performance metrics
type CacheStatistics struct {
	HitCount        int64   `json:"hit_count"`
	MissCount       int64   `json:"miss_count"`
	HitRate         float64 `json:"hit_rate"`
	StaleHitCount   int64   `json:"stale_hit_count"`
	EvictionCount   int64   `json:"eviction_count"`
	TotalKeys       int64   `json:"total_keys"`
	MemoryUsage     int64   `json:"memory_usage_bytes"`
	AverageLatency  time.Duration `json:"average_latency"`
}

// IntegrationMetrics contains service performance metrics
type IntegrationMetrics struct {
	// Request metrics
	TotalRequests       int64   `json:"total_requests"`
	SuccessfulRequests  int64   `json:"successful_requests"`
	FailedRequests      int64   `json:"failed_requests"`
	SuccessRate         float64 `json:"success_rate"`
	
	// Performance metrics
	AverageLatency      time.Duration `json:"average_latency"`
	P95Latency          time.Duration `json:"p95_latency"`
	P99Latency          time.Duration `json:"p99_latency"`
	
	// Cache metrics
	CacheStats          CacheStatistics `json:"cache_stats"`
	
	// Knowledge Base metrics
	KBCallCount         map[string]int64 `json:"kb_call_count"`
	KBLatency           map[string]time.Duration `json:"kb_latency"`
	KBErrorRate         map[string]float64 `json:"kb_error_rate"`
	
	// Context Gateway metrics
	ContextGatewayLatency time.Duration `json:"context_gateway_latency"`
	ContextGatewayErrors  int64         `json:"context_gateway_errors"`
	
	// Circuit breaker metrics
	CircuitBreakerStates map[string]CircuitBreakerState `json:"circuit_breaker_states"`
}

// ServiceConfiguration contains service configuration
type ServiceConfiguration struct {
	// Cache configuration
	Cache struct {
		DefaultTTL      time.Duration `json:"default_ttl"`
		StaleTTL        time.Duration `json:"stale_ttl"`
		MaxKeys         int64         `json:"max_keys"`
		CompressionEnabled bool       `json:"compression_enabled"`
	} `json:"cache"`
	
	// Circuit breaker configuration
	CircuitBreaker struct {
		FailureThreshold int           `json:"failure_threshold"`
		Timeout          time.Duration `json:"timeout"`
		RecoveryTimeout  time.Duration `json:"recovery_timeout"`
	} `json:"circuit_breaker"`
	
	// Knowledge Base configuration
	KnowledgeBases struct {
		Endpoints map[string]string `json:"endpoints"`
		Timeout   time.Duration     `json:"timeout"`
		Retries   int               `json:"retries"`
	} `json:"knowledge_bases"`
	
	// Context Gateway configuration
	ContextGateway struct {
		Endpoint string        `json:"endpoint"`
		Timeout  time.Duration `json:"timeout"`
		Retries  int           `json:"retries"`
	} `json:"context_gateway"`
	
	// Performance configuration
	Performance struct {
		MaxConcurrentRequests int           `json:"max_concurrent_requests"`
		RequestTimeout        time.Duration `json:"request_timeout"`
		ParallelismEnabled    bool          `json:"parallelism_enabled"`
	} `json:"performance"`
}

// CircuitBreakerState represents the state of a circuit breaker
type CircuitBreakerState string

const (
	CircuitBreakerClosed   CircuitBreakerState = "closed"
	CircuitBreakerOpen     CircuitBreakerState = "open"
	CircuitBreakerHalfOpen CircuitBreakerState = "half_open"
)

// CircuitBreakerMetrics contains circuit breaker performance data
type CircuitBreakerMetrics struct {
	State           CircuitBreakerState `json:"state"`
	FailureCount    int64               `json:"failure_count"`
	SuccessCount    int64               `json:"success_count"`
	LastFailureTime time.Time           `json:"last_failure_time"`
	LastSuccessTime time.Time           `json:"last_success_time"`
	OpenedAt        time.Time           `json:"opened_at"`
}

// Error types for the Context Integration Service
type ContextIntegrationError struct {
	Type    string `json:"type"`
	Message string `json:"message"`
	Code    int    `json:"code"`
	Details map[string]interface{} `json:"details,omitempty"`
}

func (e *ContextIntegrationError) Error() string {
	return e.Message
}

// Common error types
var (
	ErrCacheUnavailable     = &ContextIntegrationError{Type: "cache_unavailable", Message: "cache service unavailable", Code: 1001}
	ErrKBServiceUnavailable = &ContextIntegrationError{Type: "kb_service_unavailable", Message: "knowledge base service unavailable", Code: 1002}
	ErrContextGatewayUnavailable = &ContextIntegrationError{Type: "context_gateway_unavailable", Message: "context gateway service unavailable", Code: 1003}
	ErrInvalidManifest      = &ContextIntegrationError{Type: "invalid_manifest", Message: "invalid intent manifest", Code: 1004}
	ErrTimeout              = &ContextIntegrationError{Type: "timeout", Message: "request timeout", Code: 1005}
	ErrCircuitBreakerOpen   = &ContextIntegrationError{Type: "circuit_breaker_open", Message: "circuit breaker is open", Code: 1006}
)

// Helper functions for creating specific errors
func NewCacheError(message string, details map[string]interface{}) *ContextIntegrationError {
	return &ContextIntegrationError{
		Type:    "cache_error",
		Message: message,
		Code:    1001,
		Details: details,
	}
}

func NewKBServiceError(service string, message string, details map[string]interface{}) *ContextIntegrationError {
	if details == nil {
		details = make(map[string]interface{})
	}
	details["service"] = service
	
	return &ContextIntegrationError{
		Type:    "kb_service_error",
		Message: message,
		Code:    1002,
		Details: details,
	}
}

func NewContextGatewayError(message string, details map[string]interface{}) *ContextIntegrationError {
	return &ContextIntegrationError{
		Type:    "context_gateway_error",
		Message: message,
		Code:    1003,
		Details: details,
	}
}

func NewTimeoutError(operation string, timeout time.Duration) *ContextIntegrationError {
	return &ContextIntegrationError{
		Type:    "timeout",
		Message: "operation timed out",
		Code:    1005,
		Details: map[string]interface{}{
			"operation": operation,
			"timeout":   timeout.String(),
		},
	}
}

// Constants for cache strategies
const (
	CacheStrategyAggressive = "aggressive"
	CacheStrategyStandard   = "standard"
	CacheStrategyMinimal    = "minimal"
	CacheStrategyNone       = "none"
)

// Constants for Knowledge Base identifiers (matching intent_manifest.go)
const (
	KBDrugMaster         = "kb_drug_master_v1"
	KBDosingRules        = "kb_dosing_rules_v1"
	KBDrugInteractions   = "kb_ddi_v1"
	KBFormularyStock     = "kb_formulary_stock_v1"
	KBPatientSafetyChecks = "kb_patient_safe_checks_v1"
	KBGuidelineEvidence  = "kb_guideline_evidence_v1"
	KBResistanceProfiles = "kb_resistance_profiles_v1"
)

// GetAllKBIdentifiers returns all valid Knowledge Base identifiers
func GetAllKBIdentifiers() []string {
	return []string{
		KBDrugMaster,
		KBDosingRules,
		KBDrugInteractions,
		KBFormularyStock,
		KBPatientSafetyChecks,
		KBGuidelineEvidence,
		KBResistanceProfiles,
	}
}
