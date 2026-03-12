package services

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/domain/repositories"
	"medication-service-v2/internal/infrastructure/redis"
)

// RecipeResolverIntegration coordinates all recipe resolution components
type RecipeResolverIntegration struct {
	// Core services
	resolverService     *RecipeResolverServiceImpl
	templateService     *RecipeTemplateServiceImpl
	cacheService        *RecipeCacheService
	ruleEngine          *ConditionalRuleEngine
	
	// Protocol resolvers
	protocolRegistry    *ProtocolResolverRegistry
	
	// Configuration
	config              IntegrationConfig
	
	// Performance tracking
	performanceTracker  *PerformanceTracker
	
	// Synchronization
	mu                  sync.RWMutex
	initialized         bool
}

// IntegrationConfig contains configuration for the integration
type IntegrationConfig struct {
	// Performance settings
	PerformanceTarget     time.Duration `json:"performance_target"`
	EnableParallelProcessing bool       `json:"enable_parallel_processing"`
	MaxConcurrentResolvers   int        `json:"max_concurrent_resolvers"`
	
	// Caching settings
	DefaultCacheTTL       time.Duration `json:"default_cache_ttl"`
	EnableCaching         bool          `json:"enable_caching"`
	CacheCompressionEnabled bool        `json:"cache_compression_enabled"`
	
	// Rule engine settings
	MaxRulesPerProtocol   int           `json:"max_rules_per_protocol"`
	RuleEvaluationTimeout time.Duration `json:"rule_evaluation_timeout"`
	
	// Template settings
	MaxTemplatesPerProtocol int         `json:"max_templates_per_protocol"`
	TemplateValidationLevel string      `json:"template_validation_level"`
	
	// Feature flags
	EnableConditionalRules  bool         `json:"enable_conditional_rules"`
	EnableProtocolResolvers bool         `json:"enable_protocol_resolvers"`
	EnableFieldMerging      bool         `json:"enable_field_merging"`
	EnableFreshnessChecks   bool         `json:"enable_freshness_checks"`
}

// PerformanceTracker tracks overall system performance
type PerformanceTracker struct {
	TotalResolutions    int64         `json:"total_resolutions"`
	SuccessfulResolutions int64       `json:"successful_resolutions"`
	FailedResolutions   int64         `json:"failed_resolutions"`
	AverageResolutionTime time.Duration `json:"average_resolution_time"`
	CacheHitRate        float64       `json:"cache_hit_rate"`
	PerformanceTarget   time.Duration `json:"performance_target"`
	TargetMeetRate      float64       `json:"target_meet_rate"`
	LastUpdated         time.Time     `json:"last_updated"`
	
	// Performance buckets
	Under10ms           int64         `json:"under_10ms"`
	Between10And50ms    int64         `json:"between_10_and_50ms"`
	Between50And100ms   int64         `json:"between_50_and_100ms"`
	Over100ms           int64         `json:"over_100ms"`
}

// IntegrationHealthStatus represents the health of the integration
type IntegrationHealthStatus struct {
	Overall             string            `json:"overall"`
	Components          map[string]string `json:"components"`
	PerformanceMetrics  *PerformanceTracker `json:"performance_metrics"`
	LastHealthCheck     time.Time         `json:"last_health_check"`
	Issues              []HealthIssue     `json:"issues,omitempty"`
}

// HealthIssue represents a health issue
type HealthIssue struct {
	Component   string    `json:"component"`
	Severity    string    `json:"severity"`
	Message     string    `json:"message"`
	Timestamp   time.Time `json:"timestamp"`
	Suggestion  string    `json:"suggestion,omitempty"`
}

// NewRecipeResolverIntegration creates a new recipe resolver integration
func NewRecipeResolverIntegration(
	recipeRepo repositories.RecipeRepository,
	medicationRepo repositories.MedicationRepository,
	templateRepo TemplateRepository,
	ruleRepo ConditionalRuleRepository,
	redisClient redis.Client,
	config IntegrationConfig,
) *RecipeResolverIntegration {
	
	// Create cache service
	cacheConfig := CacheConfig{
		DefaultTTL:         config.DefaultCacheTTL,
		PerformanceTarget:  config.PerformanceTarget,
		CompressionEnabled: config.CacheCompressionEnabled,
	}
	cacheService := NewRecipeCacheService(redisClient, cacheConfig)
	
	// Create resolver service
	resolverService := NewRecipeResolverService(recipeRepo, medicationRepo, redisClient)
	
	// Create template service
	templateService := NewRecipeTemplateService(recipeRepo, templateRepo)
	
	// Create rule engine
	ruleEngine := NewConditionalRuleEngine(ruleRepo)
	
	// Create protocol registry
	protocolRegistry := NewProtocolResolverRegistry()
	
	integration := &RecipeResolverIntegration{
		resolverService:     resolverService,
		templateService:     templateService,
		cacheService:        cacheService,
		ruleEngine:          ruleEngine,
		protocolRegistry:    protocolRegistry,
		config:              config,
		performanceTracker:  &PerformanceTracker{PerformanceTarget: config.PerformanceTarget},
		initialized:         false,
	}
	
	return integration
}

// Initialize initializes the integration and all components
func (i *RecipeResolverIntegration) Initialize(ctx context.Context) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	if i.initialized {
		return nil
	}
	
	// Register protocol resolvers with the resolver service
	if i.config.EnableProtocolResolvers {
		for _, protocolID := range i.protocolRegistry.List() {
			resolver, err := i.protocolRegistry.Get(protocolID)
			if err != nil {
				return errors.Wrapf(err, "failed to get protocol resolver for %s", protocolID)
			}
			i.resolverService.RegisterProtocolResolver(protocolID, resolver)
		}
	}
	
	// Initialize performance tracking
	i.performanceTracker.LastUpdated = time.Now()
	
	i.initialized = true
	return nil
}

// ResolveRecipeWithIntegration performs integrated recipe resolution
func (i *RecipeResolverIntegration) ResolveRecipeWithIntegration(ctx context.Context, request entities.RecipeResolutionRequest) (*entities.RecipeResolution, error) {
	startTime := time.Now()
	
	// Ensure initialization
	if !i.initialized {
		if err := i.Initialize(ctx); err != nil {
			return nil, errors.Wrap(err, "failed to initialize integration")
		}
	}
	
	// Update performance tracking
	defer func() {
		processingTime := time.Since(startTime)
		i.updatePerformanceMetrics(processingTime, true)
	}()
	
	// Check if caching is enabled and try cache first
	if i.config.EnableCaching && request.Options.UseCache {
		if cached, err := i.cacheService.GetRecipeResolution(ctx, request.RecipeID, request.PatientContext.PatientID); err == nil {
			return cached, nil
		}
	}
	
	// Resolve recipe using main service
	resolution, err := i.resolverService.ResolveRecipe(ctx, request)
	if err != nil {
		i.updatePerformanceMetrics(time.Since(startTime), false)
		return nil, errors.Wrap(err, "recipe resolution failed")
	}
	
	// Cache result if caching is enabled
	if i.config.EnableCaching && request.Options.UseCache {
		cacheTTL := request.Options.CacheTTL
		if cacheTTL == 0 {
			cacheTTL = i.config.DefaultCacheTTL
		}
		
		// Cache asynchronously to not impact performance
		go func() {
			if err := i.cacheService.SetRecipeResolution(context.Background(), request.RecipeID, request.PatientContext.PatientID, resolution, cacheTTL); err != nil {
				// Log error but don't fail the request
			}
		}()
	}
	
	return resolution, nil
}

// CreateRecipeFromTemplateWithIntegration creates a recipe from template with full integration
func (i *RecipeResolverIntegration) CreateRecipeFromTemplateWithIntegration(ctx context.Context, templateID uuid.UUID, customization RecipeCustomization) (*entities.Recipe, error) {
	// Validate template exists and is active
	template, err := i.templateService.GetTemplate(ctx, templateID)
	if err != nil {
		return nil, errors.Wrap(err, "failed to get template")
	}
	
	if !template.IsActive {
		return nil, errors.New("template is not active")
	}
	
	// Create recipe from template
	recipe, err := i.templateService.CreateRecipeFromTemplate(ctx, templateID, customization)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create recipe from template")
	}
	
	// Clear related caches
	if i.config.EnableCaching {
		go func() {
			_ = i.cacheService.InvalidateProtocolCache(context.Background(), recipe.ProtocolID)
		}()
	}
	
	return recipe, nil
}

// EvaluateRulesWithIntegration evaluates rules with full integration support
func (i *RecipeResolverIntegration) EvaluateRulesWithIntegration(ctx context.Context, protocolID string, patientContext entities.PatientContext) ([]*EvaluationResult, error) {
	if !i.config.EnableConditionalRules {
		return []*EvaluationResult{}, nil
	}
	
	return i.ruleEngine.EvaluateRules(ctx, protocolID, patientContext)
}

// GetIntegrationHealth returns the health status of the integration
func (i *RecipeResolverIntegration) GetIntegrationHealth(ctx context.Context) (*IntegrationHealthStatus, error) {
	health := &IntegrationHealthStatus{
		Overall:         "healthy",
		Components:      make(map[string]string),
		PerformanceMetrics: i.performanceTracker,
		LastHealthCheck: time.Now(),
		Issues:          make([]HealthIssue, 0),
	}
	
	// Check cache health
	if i.cacheService.IsCacheHealthy() {
		health.Components["cache"] = "healthy"
	} else {
		health.Components["cache"] = "degraded"
		health.Issues = append(health.Issues, HealthIssue{
			Component: "cache",
			Severity:  "warning",
			Message:   "Cache performance below target",
			Timestamp: time.Now(),
			Suggestion: "Check Redis performance and memory usage",
		})
	}
	
	// Check performance metrics
	if i.performanceTracker.TargetMeetRate < 0.9 {
		health.Components["performance"] = "degraded"
		health.Issues = append(health.Issues, HealthIssue{
			Component: "performance",
			Severity:  "warning",
			Message:   fmt.Sprintf("Performance target meet rate is %.2f%%, below 90%%", i.performanceTracker.TargetMeetRate*100),
			Timestamp: time.Now(),
			Suggestion: "Review resolution logic and consider performance optimizations",
		})
	} else {
		health.Components["performance"] = "healthy"
	}
	
	// Check protocol resolvers
	protocolCount := len(i.protocolRegistry.List())
	if protocolCount == 0 {
		health.Components["protocols"] = "warning"
		health.Issues = append(health.Issues, HealthIssue{
			Component: "protocols",
			Severity:  "info",
			Message:   "No protocol resolvers registered",
			Timestamp: time.Now(),
			Suggestion: "Register protocol-specific resolvers for better performance",
		})
	} else {
		health.Components["protocols"] = "healthy"
	}
	
	// Overall health determination
	if len(health.Issues) > 0 {
		hasWarnings := false
		hasErrors := false
		
		for _, issue := range health.Issues {
			if issue.Severity == "warning" {
				hasWarnings = true
			}
			if issue.Severity == "error" {
				hasErrors = true
			}
		}
		
		if hasErrors {
			health.Overall = "unhealthy"
		} else if hasWarnings {
			health.Overall = "degraded"
		}
	}
	
	return health, nil
}

// GetPerformanceMetrics returns current performance metrics
func (i *RecipeResolverIntegration) GetPerformanceMetrics() *PerformanceTracker {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	// Create a copy to avoid race conditions
	metrics := *i.performanceTracker
	return &metrics
}

// OptimizePerformance performs performance optimizations
func (i *RecipeResolverIntegration) OptimizePerformance(ctx context.Context) error {
	// Optimize cache
	if err := i.cacheService.OptimizeCache(ctx); err != nil {
		return errors.Wrap(err, "cache optimization failed")
	}
	
	// Clear expired cache entries
	if err := i.cacheService.ClearExpiredEntries(ctx); err != nil {
		return errors.Wrap(err, "failed to clear expired cache entries")
	}
	
	return nil
}

// RegisterProtocolResolver registers a new protocol resolver
func (i *RecipeResolverIntegration) RegisterProtocolResolver(protocolID string, resolver entities.ProtocolResolver) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	// Register with registry
	i.protocolRegistry.Register(protocolID, resolver)
	
	// Register with resolver service if initialized
	if i.initialized {
		i.resolverService.RegisterProtocolResolver(protocolID, resolver)
	}
	
	return nil
}

// GetAvailableProtocols returns available protocol resolvers
func (i *RecipeResolverIntegration) GetAvailableProtocols() []string {
	return i.protocolRegistry.List()
}

// GetConfiguration returns current configuration
func (i *RecipeResolverIntegration) GetConfiguration() IntegrationConfig {
	i.mu.RLock()
	defer i.mu.RUnlock()
	
	return i.config
}

// UpdateConfiguration updates the integration configuration
func (i *RecipeResolverIntegration) UpdateConfiguration(ctx context.Context, config IntegrationConfig) error {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	i.config = config
	i.performanceTracker.PerformanceTarget = config.PerformanceTarget
	
	return nil
}

// Helper methods

func (i *RecipeResolverIntegration) updatePerformanceMetrics(processingTime time.Duration, success bool) {
	i.mu.Lock()
	defer i.mu.Unlock()
	
	i.performanceTracker.TotalResolutions++
	
	if success {
		i.performanceTracker.SuccessfulResolutions++
	} else {
		i.performanceTracker.FailedResolutions++
	}
	
	// Update average processing time
	totalTime := time.Duration(i.performanceTracker.TotalResolutions) * i.performanceTracker.AverageResolutionTime
	i.performanceTracker.AverageResolutionTime = (totalTime + processingTime) / time.Duration(i.performanceTracker.TotalResolutions)
	
	// Update performance buckets
	if processingTime < 10*time.Millisecond {
		i.performanceTracker.Under10ms++
	} else if processingTime < 50*time.Millisecond {
		i.performanceTracker.Between10And50ms++
	} else if processingTime < 100*time.Millisecond {
		i.performanceTracker.Between50And100ms++
	} else {
		i.performanceTracker.Over100ms++
	}
	
	// Update target meet rate
	targetMeets := i.performanceTracker.Under10ms
	if i.performanceTracker.PerformanceTarget > 10*time.Millisecond {
		targetMeets += i.performanceTracker.Between10And50ms
		if i.performanceTracker.PerformanceTarget > 50*time.Millisecond {
			targetMeets += i.performanceTracker.Between50And100ms
		}
	}
	
	if i.performanceTracker.TotalResolutions > 0 {
		i.performanceTracker.TargetMeetRate = float64(targetMeets) / float64(i.performanceTracker.TotalResolutions)
	}
	
	i.performanceTracker.LastUpdated = time.Now()
}

// DefaultIntegrationConfig returns default configuration
func DefaultIntegrationConfig() IntegrationConfig {
	return IntegrationConfig{
		PerformanceTarget:       10 * time.Millisecond,
		EnableParallelProcessing: true,
		MaxConcurrentResolvers:  10,
		DefaultCacheTTL:         5 * time.Minute,
		EnableCaching:           true,
		CacheCompressionEnabled: false,
		MaxRulesPerProtocol:     100,
		RuleEvaluationTimeout:   5 * time.Second,
		MaxTemplatesPerProtocol: 50,
		TemplateValidationLevel: "strict",
		EnableConditionalRules:  true,
		EnableProtocolResolvers: true,
		EnableFieldMerging:      true,
		EnableFreshnessChecks:   true,
	}
}