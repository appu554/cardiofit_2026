package performance

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/internal/cache"
	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/internal/config"
	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/pkg/logger"
	"github.com/clinical-synthesis-hub/cardiofit/safety-gateway-platform/pkg/types"
)

// PerformanceOptimizationEngine coordinates all performance optimization components
type PerformanceOptimizationEngine struct {
	// Core components
	resourceManager    *AdaptiveResourceManager
	cacheManager       *cache.AdvancedCacheManager
	preWarmingSystem   *cache.PredictivePreWarmingSystem
	
	// Optimization engines
	memoryOptimizer    *MemoryOptimizer
	cpuOptimizer       *CPUOptimizer
	networkOptimizer   *NetworkOptimizer
	latencyOptimizer   *LatencyOptimizer
	
	// Configuration and control
	config             *config.PerformanceOptimizationConfig
	logger             *logger.Logger
	
	// State management
	isRunning          int32 // atomic
	currentProfile     *PerformanceProfile
	optimizationState  *OptimizationState
	stopCh             chan struct{}
	mu                 sync.RWMutex
	
	// Metrics and monitoring
	metrics            *PerformanceOptimizationMetrics
	optimizer          *PerformanceOptimizerCore
}

// PerformanceProfile represents the current performance configuration
type PerformanceProfile struct {
	Name               string                 `json:"name"`
	Mode               OptimizationMode       `json:"mode"`
	Targets            *PerformanceTargets    `json:"targets"`
	Constraints        *ResourceConstraints   `json:"constraints"`
	Strategies         []OptimizationStrategy `json:"strategies"`
	CreatedAt          time.Time              `json:"created_at"`
	LastUpdated        time.Time              `json:"last_updated"`
	IsActive           bool                   `json:"is_active"`
}

type OptimizationMode string

const (
	ModeBalanced     OptimizationMode = "balanced"
	ModePerformance  OptimizationMode = "performance"
	ModeEfficiency   OptimizationMode = "efficiency"
	ModeStability    OptimizationMode = "stability"
	ModeAdaptive     OptimizationMode = "adaptive"
)

// PerformanceTargets defines performance objectives
type PerformanceTargets struct {
	MaxResponseTimeMs    int     `json:"max_response_time_ms"`
	MinThroughputRPS     int     `json:"min_throughput_rps"`
	MaxErrorRate         float64 `json:"max_error_rate"`
	MinCacheHitRate      float64 `json:"min_cache_hit_rate"`
	MaxMemoryUsage       float64 `json:"max_memory_usage"`
	MaxCPUUsage          float64 `json:"max_cpu_usage"`
	MinSystemStability   float64 `json:"min_system_stability"`
}

// ResourceConstraints defines resource usage limits
type ResourceConstraints struct {
	MaxMemoryMB          int           `json:"max_memory_mb"`
	MaxCPUCores          int           `json:"max_cpu_cores"`
	MaxConnections       int           `json:"max_connections"`
	MaxGoroutines        int           `json:"max_goroutines"`
	MaxCacheSize         int           `json:"max_cache_size"`
	MaxBandwidthMBps     int           `json:"max_bandwidth_mbps"`
	OptimizationInterval time.Duration `json:"optimization_interval"`
}

type OptimizationStrategy string

const (
	StrategyPreemptive   OptimizationStrategy = "preemptive"
	StrategyReactive     OptimizationStrategy = "reactive"
	StrategyPredictive   OptimizationStrategy = "predictive"
	StrategyAggressive   OptimizationStrategy = "aggressive"
	StrategyConservative OptimizationStrategy = "conservative"
)

// OptimizationState tracks current optimization activities
type OptimizationState struct {
	CurrentMode           OptimizationMode          `json:"current_mode"`
	ActiveOptimizations   map[string]*Optimization  `json:"active_optimizations"`
	ScheduledOptimizations []*ScheduledOptimization `json:"scheduled_optimizations"`
	PerformanceScore      float64                   `json:"performance_score"`
	EfficiencyScore       float64                   `json:"efficiency_score"`
	StabilityScore        float64                   `json:"stability_score"`
	LastOptimization      time.Time                 `json:"last_optimization"`
	OptimizationCycle     int64                     `json:"optimization_cycle"`
	
	mu sync.RWMutex
}

// Optimization represents an active optimization process
type Optimization struct {
	ID                string                 `json:"id"`
	Type              OptimizationType       `json:"type"`
	Status            OptimizationStatus     `json:"status"`
	StartTime         time.Time              `json:"start_time"`
	EstimatedDuration time.Duration          `json:"estimated_duration"`
	Progress          float64                `json:"progress"`
	Impact            *OptimizationImpact    `json:"impact"`
	Parameters        map[string]interface{} `json:"parameters"`
	Results           *OptimizationResult    `json:"results,omitempty"`
}

type OptimizationType string

const (
	TypeMemoryOptimization    OptimizationType = "memory"
	TypeCPUOptimization      OptimizationType = "cpu"
	TypeCacheOptimization    OptimizationType = "cache"
	TypeNetworkOptimization  OptimizationType = "network"
	TypeLatencyOptimization  OptimizationType = "latency"
	TypeThroughputOptimization OptimizationType = "throughput"
)

type OptimizationStatus string

const (
	StatusPending    OptimizationStatus = "pending"
	StatusRunning    OptimizationStatus = "running"
	StatusCompleted  OptimizationStatus = "completed"
	StatusFailed     OptimizationStatus = "failed"
	StatusCancelled  OptimizationStatus = "cancelled"
)

// ScheduledOptimization represents a future optimization
type ScheduledOptimization struct {
	ID                string            `json:"id"`
	Type              OptimizationType  `json:"type"`
	ScheduledTime     time.Time         `json:"scheduled_time"`
	Priority          int               `json:"priority"`
	Condition         string            `json:"condition"`
	Parameters        map[string]interface{} `json:"parameters"`
	CreatedAt         time.Time         `json:"created_at"`
}

// OptimizationResult represents the outcome of an optimization
type OptimizationResult struct {
	Success              bool              `json:"success"`
	ImprovementPercent   float64           `json:"improvement_percent"`
	ResourceSavings      map[string]float64 `json:"resource_savings"`
	PerformanceGains     map[string]float64 `json:"performance_gains"`
	SideEffects          []string          `json:"side_effects"`
	Recommendations      []string          `json:"recommendations"`
	CompletedAt          time.Time         `json:"completed_at"`
}

// OptimizationImpact measures expected or actual impact
type OptimizationImpact struct {
	PerformanceImprovement float64 `json:"performance_improvement"`
	ResourceReduction      float64 `json:"resource_reduction"`
	StabilityChange        float64 `json:"stability_change"`
	RiskLevel             float64 `json:"risk_level"`
	Confidence            float64 `json:"confidence"`
}

// PerformanceOptimizationMetrics tracks optimization performance
type PerformanceOptimizationMetrics struct {
	TotalOptimizations    int64                    `json:"total_optimizations"`
	SuccessfulOptimizations int64                  `json:"successful_optimizations"`
	FailedOptimizations   int64                    `json:"failed_optimizations"`
	
	AverageImprovement    float64                  `json:"average_improvement"`
	TotalResourceSavings  map[string]float64       `json:"total_resource_savings"`
	
	OptimizationsByType   map[OptimizationType]int64 `json:"optimizations_by_type"`
	OptimizationHistory   []*OptimizationHistoryEntry `json:"optimization_history"`
	
	CurrentPerformanceScore float64                  `json:"current_performance_score"`
	BaselinePerformanceScore float64                 `json:"baseline_performance_score"`
	
	mu sync.RWMutex
}

// OptimizationHistoryEntry represents a historical optimization record
type OptimizationHistoryEntry struct {
	Timestamp       time.Time         `json:"timestamp"`
	Type            OptimizationType  `json:"type"`
	Success         bool              `json:"success"`
	Improvement     float64           `json:"improvement"`
	Duration        time.Duration     `json:"duration"`
	ResourcesBefore map[string]float64 `json:"resources_before"`
	ResourcesAfter  map[string]float64 `json:"resources_after"`
}

// NewPerformanceOptimizationEngine creates a new performance optimization engine
func NewPerformanceOptimizationEngine(
	resourceManager *AdaptiveResourceManager,
	cacheManager *cache.AdvancedCacheManager,
	preWarmingSystem *cache.PredictivePreWarmingSystem,
	config *config.PerformanceOptimizationConfig,
	logger *logger.Logger,
) *PerformanceOptimizationEngine {
	
	engine := &PerformanceOptimizationEngine{
		resourceManager:  resourceManager,
		cacheManager:     cacheManager,
		preWarmingSystem: preWarmingSystem,
		config:           config,
		logger:           logger,
		stopCh:           make(chan struct{}),
		metrics:          NewPerformanceOptimizationMetrics(),
		currentProfile:   createDefaultPerformanceProfile(),
		optimizationState: NewOptimizationState(),
	}
	
	// Initialize optimizers
	engine.memoryOptimizer = NewMemoryOptimizer(config.Memory, logger)
	engine.cpuOptimizer = NewCPUOptimizer(config.CPU, logger)
	engine.networkOptimizer = NewNetworkOptimizer(config.Network, logger)
	engine.latencyOptimizer = NewLatencyOptimizer(config.Latency, logger)
	engine.optimizer = NewPerformanceOptimizerCore(engine, logger)
	
	return engine
}

// Start begins the performance optimization engine
func (e *PerformanceOptimizationEngine) Start(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 0, 1) {
		return fmt.Errorf("performance optimization engine is already running")
	}
	
	e.logger.Info("Starting performance optimization engine")
	
	// Start component optimizers
	if err := e.memoryOptimizer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start memory optimizer: %w", err)
	}
	
	if err := e.cpuOptimizer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start CPU optimizer: %w", err)
	}
	
	if err := e.networkOptimizer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start network optimizer: %w", err)
	}
	
	if err := e.latencyOptimizer.Start(ctx); err != nil {
		return fmt.Errorf("failed to start latency optimizer: %w", err)
	}
	
	// Start main optimization loop
	go e.optimizationLoop(ctx)
	
	// Start performance monitoring
	go e.performanceMonitoringLoop(ctx)
	
	e.logger.Info("Performance optimization engine started")
	return nil
}

// Stop gracefully shuts down the performance optimization engine
func (e *PerformanceOptimizationEngine) Stop(ctx context.Context) error {
	if !atomic.CompareAndSwapInt32(&e.isRunning, 1, 0) {
		return nil
	}
	
	e.logger.Info("Stopping performance optimization engine")
	
	close(e.stopCh)
	
	// Stop component optimizers
	e.latencyOptimizer.Stop(ctx)
	e.networkOptimizer.Stop(ctx)
	e.cpuOptimizer.Stop(ctx)
	e.memoryOptimizer.Stop(ctx)
	
	e.logger.Info("Performance optimization engine stopped")
	return nil
}

// optimizationLoop runs the main optimization cycle
func (e *PerformanceOptimizationEngine) optimizationLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.OptimizationInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.runOptimizationCycle()
		}
	}
}

// performanceMonitoringLoop continuously monitors performance metrics
func (e *PerformanceOptimizationEngine) performanceMonitoringLoop(ctx context.Context) {
	ticker := time.NewTicker(e.config.MonitoringInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ctx.Done():
			return
		case <-e.stopCh:
			return
		case <-ticker.C:
			e.updatePerformanceScores()
		}
	}
}

// runOptimizationCycle executes a complete optimization cycle
func (e *PerformanceOptimizationEngine) runOptimizationCycle() {
	e.optimizationState.mu.Lock()
	cycle := e.optimizationState.OptimizationCycle + 1
	e.optimizationState.OptimizationCycle = cycle
	e.optimizationState.mu.Unlock()
	
	e.logger.Debug("Starting optimization cycle", "cycle", cycle)
	
	// Collect current performance data
	resourceStats := e.resourceManager.GetCurrentStats()
	if resourceStats == nil {
		e.logger.Warn("No resource statistics available for optimization")
		return
	}
	
	// Analyze performance and identify optimization opportunities
	opportunities := e.identifyOptimizationOpportunities(resourceStats)
	
	// Prioritize and execute optimizations
	for _, opportunity := range opportunities {
		if e.shouldExecuteOptimization(opportunity) {
			e.executeOptimization(opportunity)
		}
	}
	
	// Update optimization state
	e.optimizationState.mu.Lock()
	e.optimizationState.LastOptimization = time.Now()
	e.optimizationState.mu.Unlock()
	
	e.logger.Debug("Optimization cycle completed", 
		"cycle", cycle,
		"opportunities", len(opportunities))
}

// identifyOptimizationOpportunities analyzes current state and identifies opportunities
func (e *PerformanceOptimizationEngine) identifyOptimizationOpportunities(stats *ResourceStats) []*OptimizationOpportunity {
	var opportunities []*OptimizationOpportunity
	
	// Memory optimization opportunities
	if stats.MemoryPercent > 80 {
		opportunities = append(opportunities, &OptimizationOpportunity{
			Type:        TypeMemoryOptimization,
			Priority:    e.calculatePriority(stats.MemoryPercent, 80, 100),
			Urgency:     e.calculateUrgency(stats.MemoryPercent, 90),
			Impact:      &OptimizationImpact{
				PerformanceImprovement: 15.0,
				ResourceReduction:      20.0,
				RiskLevel:             0.3,
				Confidence:            0.8,
			},
			Triggers:    []string{"high_memory_usage"},
			Parameters:  map[string]interface{}{"current_usage": stats.MemoryPercent},
		})
	}
	
	// CPU optimization opportunities
	if stats.CPUUsage > 75 {
		opportunities = append(opportunities, &OptimizationOpportunity{
			Type:        TypeCPUOptimization,
			Priority:    e.calculatePriority(stats.CPUUsage, 75, 100),
			Urgency:     e.calculateUrgency(stats.CPUUsage, 85),
			Impact:      &OptimizationImpact{
				PerformanceImprovement: 12.0,
				ResourceReduction:      15.0,
				RiskLevel:             0.4,
				Confidence:            0.7,
			},
			Triggers:    []string{"high_cpu_usage"},
			Parameters:  map[string]interface{}{"current_usage": stats.CPUUsage},
		})
	}
	
	// Cache optimization opportunities
	if e.cacheManager != nil {
		cacheStats := e.cacheManager.GetStatistics()
		if cacheStats.HitRatio < 0.7 {
			opportunities = append(opportunities, &OptimizationOpportunity{
				Type:        TypeCacheOptimization,
				Priority:    e.calculatePriority(cacheStats.HitRatio, 0.5, 0.9),
				Urgency:     e.calculateUrgency(1.0 - cacheStats.HitRatio, 0.4),
				Impact:      &OptimizationImpact{
					PerformanceImprovement: 25.0,
					ResourceReduction:      10.0,
					RiskLevel:             0.2,
					Confidence:            0.9,
				},
				Triggers:    []string{"low_cache_hit_ratio"},
				Parameters:  map[string]interface{}{"hit_ratio": cacheStats.HitRatio},
			})
		}
	}
	
	// Network optimization opportunities
	if stats.NetworkLatency > 100*time.Millisecond {
		opportunities = append(opportunities, &OptimizationOpportunity{
			Type:        TypeNetworkOptimization,
			Priority:    e.calculatePriority(float64(stats.NetworkLatency.Milliseconds()), 50, 200),
			Urgency:     e.calculateUrgency(float64(stats.NetworkLatency.Milliseconds()), 150),
			Impact:      &OptimizationImpact{
				PerformanceImprovement: 30.0,
				ResourceReduction:      5.0,
				RiskLevel:             0.3,
				Confidence:            0.75,
			},
			Triggers:    []string{"high_network_latency"},
			Parameters:  map[string]interface{}{"latency_ms": stats.NetworkLatency.Milliseconds()},
		})
	}
	
	// Response time optimization
	if stats.ResponseTime > time.Duration(e.currentProfile.Targets.MaxResponseTimeMs)*time.Millisecond {
		opportunities = append(opportunities, &OptimizationOpportunity{
			Type:        TypeLatencyOptimization,
			Priority:    e.calculatePriority(float64(stats.ResponseTime.Milliseconds()), 
				float64(e.currentProfile.Targets.MaxResponseTimeMs), 
				float64(e.currentProfile.Targets.MaxResponseTimeMs)*2),
			Urgency:     e.calculateUrgency(float64(stats.ResponseTime.Milliseconds()), 
				float64(e.currentProfile.Targets.MaxResponseTimeMs)*1.5),
			Impact:      &OptimizationImpact{
				PerformanceImprovement: 40.0,
				ResourceReduction:      8.0,
				RiskLevel:             0.25,
				Confidence:            0.85,
			},
			Triggers:    []string{"high_response_time"},
			Parameters:  map[string]interface{}{"response_time_ms": stats.ResponseTime.Milliseconds()},
		})
	}
	
	// Sort opportunities by priority and urgency
	e.sortOptimizationOpportunities(opportunities)
	
	return opportunities
}

// OptimizationOpportunity represents a potential optimization
type OptimizationOpportunity struct {
	Type        OptimizationType     `json:"type"`
	Priority    float64              `json:"priority"`
	Urgency     float64              `json:"urgency"`
	Impact      *OptimizationImpact  `json:"impact"`
	Triggers    []string             `json:"triggers"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// calculatePriority calculates optimization priority based on current, threshold, and max values
func (e *PerformanceOptimizationEngine) calculatePriority(current, threshold, max float64) float64 {
	if current <= threshold {
		return 0.0
	}
	
	// Linear scaling from threshold to max
	if max <= threshold {
		return 1.0
	}
	
	priority := (current - threshold) / (max - threshold)
	if priority > 1.0 {
		priority = 1.0
	}
	
	return priority
}

// calculateUrgency calculates optimization urgency
func (e *PerformanceOptimizationEngine) calculateUrgency(current, critical float64) float64 {
	if current >= critical {
		return 1.0
	}
	
	// Exponential curve approaching critical
	ratio := current / critical
	return ratio * ratio
}

// sortOptimizationOpportunities sorts opportunities by combined priority and urgency
func (e *PerformanceOptimizationEngine) sortOptimizationOpportunities(opportunities []*OptimizationOpportunity) {
	// Sort by combined score (priority * 0.6 + urgency * 0.4)
	for i := 0; i < len(opportunities); i++ {
		for j := i + 1; j < len(opportunities); j++ {
			scoreI := opportunities[i].Priority*0.6 + opportunities[i].Urgency*0.4
			scoreJ := opportunities[j].Priority*0.6 + opportunities[j].Urgency*0.4
			
			if scoreI < scoreJ {
				opportunities[i], opportunities[j] = opportunities[j], opportunities[i]
			}
		}
	}
}

// shouldExecuteOptimization determines if an optimization should be executed
func (e *PerformanceOptimizationEngine) shouldExecuteOptimization(opportunity *OptimizationOpportunity) bool {
	// Check if we're already running too many optimizations
	e.optimizationState.mu.RLock()
	activeCount := len(e.optimizationState.ActiveOptimizations)
	e.optimizationState.mu.RUnlock()
	
	if activeCount >= e.config.MaxConcurrentOptimizations {
		return false
	}
	
	// Check minimum priority threshold
	if opportunity.Priority < e.config.MinOptimizationPriority {
		return false
	}
	
	// Check risk tolerance
	if opportunity.Impact.RiskLevel > e.config.MaxRiskLevel {
		return false
	}
	
	// Check confidence threshold
	if opportunity.Impact.Confidence < e.config.MinConfidence {
		return false
	}
	
	return true
}

// executeOptimization executes a specific optimization
func (e *PerformanceOptimizationEngine) executeOptimization(opportunity *OptimizationOpportunity) {
	optimizationID := fmt.Sprintf("%s_%d", opportunity.Type, time.Now().UnixNano())
	
	optimization := &Optimization{
		ID:                optimizationID,
		Type:              opportunity.Type,
		Status:            StatusPending,
		StartTime:         time.Now(),
		EstimatedDuration: e.getEstimatedDuration(opportunity.Type),
		Progress:          0.0,
		Impact:            opportunity.Impact,
		Parameters:        opportunity.Parameters,
	}
	
	// Add to active optimizations
	e.optimizationState.mu.Lock()
	e.optimizationState.ActiveOptimizations[optimizationID] = optimization
	e.optimizationState.mu.Unlock()
	
	// Execute optimization asynchronously
	go e.performOptimization(optimization)
	
	e.logger.Info("Started optimization", 
		"id", optimizationID,
		"type", opportunity.Type,
		"priority", opportunity.Priority)
}

// performOptimization performs the actual optimization work
func (e *PerformanceOptimizationEngine) performOptimization(optimization *Optimization) {
	startTime := time.Now()
	
	// Update status
	optimization.Status = StatusRunning
	
	var result *OptimizationResult
	var err error
	
	// Execute based on optimization type
	switch optimization.Type {
	case TypeMemoryOptimization:
		result, err = e.memoryOptimizer.Optimize(optimization.Parameters)
	case TypeCPUOptimization:
		result, err = e.cpuOptimizer.Optimize(optimization.Parameters)
	case TypeCacheOptimization:
		result, err = e.optimizeCachePerformance(optimization.Parameters)
	case TypeNetworkOptimization:
		result, err = e.networkOptimizer.Optimize(optimization.Parameters)
	case TypeLatencyOptimization:
		result, err = e.latencyOptimizer.Optimize(optimization.Parameters)
	default:
		err = fmt.Errorf("unknown optimization type: %s", optimization.Type)
	}
	
	// Update optimization with results
	if err != nil {
		optimization.Status = StatusFailed
		e.logger.Error("Optimization failed", 
			"id", optimization.ID,
			"type", optimization.Type,
			"error", err.Error())
	} else {
		optimization.Status = StatusCompleted
		optimization.Results = result
		optimization.Progress = 1.0
		
		e.logger.Info("Optimization completed", 
			"id", optimization.ID,
			"type", optimization.Type,
			"improvement", result.ImprovementPercent,
			"duration", time.Since(startTime))
	}
	
	// Remove from active optimizations
	e.optimizationState.mu.Lock()
	delete(e.optimizationState.ActiveOptimizations, optimization.ID)
	e.optimizationState.mu.Unlock()
	
	// Record in history
	e.recordOptimizationHistory(optimization, time.Since(startTime))
	
	// Update metrics
	e.updateOptimizationMetrics(optimization)
}

// optimizeCachePerformance optimizes cache performance
func (e *PerformanceOptimizationEngine) optimizeCachePerformance(parameters map[string]interface{}) (*OptimizationResult, error) {
	if e.cacheManager == nil {
		return nil, fmt.Errorf("cache manager not available")
	}
	
	beforeStats := e.cacheManager.GetStatistics()
	
	// Trigger cache optimization
	err := e.cacheManager.OptimizePerformance()
	if err != nil {
		return nil, fmt.Errorf("cache optimization failed: %w", err)
	}
	
	// Wait for optimization to take effect
	time.Sleep(2 * time.Second)
	
	afterStats := e.cacheManager.GetStatistics()
	
	// Calculate improvement
	hitRatioImprovement := (afterStats.HitRatio - beforeStats.HitRatio) / beforeStats.HitRatio * 100
	
	result := &OptimizationResult{
		Success:            true,
		ImprovementPercent: hitRatioImprovement,
		ResourceSavings: map[string]float64{
			"cache_memory": 0.0, // Would calculate actual savings
		},
		PerformanceGains: map[string]float64{
			"cache_hit_ratio": hitRatioImprovement,
		},
		CompletedAt: time.Now(),
	}
	
	return result, nil
}

// getEstimatedDuration returns estimated duration for optimization type
func (e *PerformanceOptimizationEngine) getEstimatedDuration(optimizationType OptimizationType) time.Duration {
	durations := map[OptimizationType]time.Duration{
		TypeMemoryOptimization:     30 * time.Second,
		TypeCPUOptimization:       45 * time.Second,
		TypeCacheOptimization:     15 * time.Second,
		TypeNetworkOptimization:   60 * time.Second,
		TypeLatencyOptimization:   90 * time.Second,
		TypeThroughputOptimization: 120 * time.Second,
	}
	
	if duration, exists := durations[optimizationType]; exists {
		return duration
	}
	
	return 60 * time.Second // Default
}

// updatePerformanceScores updates current performance scoring
func (e *PerformanceOptimizationEngine) updatePerformanceScores() {
	resourceStats := e.resourceManager.GetCurrentStats()
	if resourceStats == nil {
		return
	}
	
	performanceScore := e.calculatePerformanceScore(resourceStats)
	efficiencyScore := e.calculateEfficiencyScore(resourceStats)
	stabilityScore := e.calculateStabilityScore(resourceStats)
	
	e.optimizationState.mu.Lock()
	e.optimizationState.PerformanceScore = performanceScore
	e.optimizationState.EfficiencyScore = efficiencyScore
	e.optimizationState.StabilityScore = stabilityScore
	e.optimizationState.mu.Unlock()
	
	// Update metrics
	e.metrics.mu.Lock()
	e.metrics.CurrentPerformanceScore = performanceScore
	e.metrics.mu.Unlock()
}

// calculatePerformanceScore calculates overall performance score
func (e *PerformanceOptimizationEngine) calculatePerformanceScore(stats *ResourceStats) float64 {
	// Performance score based on response time and throughput
	responseTimeScore := e.normalizeScore(float64(stats.ResponseTime.Milliseconds()), 
		0, float64(e.currentProfile.Targets.MaxResponseTimeMs), true)
		
	throughputScore := e.normalizeScore(stats.RequestsPerSecond, 
		0, float64(e.currentProfile.Targets.MinThroughputRPS), false)
		
	errorRateScore := e.normalizeScore(stats.ErrorRate,
		0, e.currentProfile.Targets.MaxErrorRate, true)
	
	// Weighted average
	return responseTimeScore*0.4 + throughputScore*0.4 + errorRateScore*0.2
}

// calculateEfficiencyScore calculates resource efficiency score
func (e *PerformanceOptimizationEngine) calculateEfficiencyScore(stats *ResourceStats) float64 {
	// Efficiency based on resource utilization vs performance
	memoryScore := e.normalizeScore(stats.MemoryPercent, 
		0, e.currentProfile.Targets.MaxMemoryUsage, true)
		
	cpuScore := e.normalizeScore(stats.CPUUsage,
		0, e.currentProfile.Targets.MaxCPUUsage, true)
	
	return (memoryScore + cpuScore) / 2.0
}

// calculateStabilityScore calculates system stability score
func (e *PerformanceOptimizationEngine) calculateStabilityScore(stats *ResourceStats) float64 {
	// Stability based on error rate and resource variance
	errorStability := 1.0 - stats.ErrorRate
	
	// Get resource variance from recent history
	resourceVariance := e.calculateResourceVariance()
	varianceStability := 1.0 - resourceVariance
	
	return (errorStability + varianceStability) / 2.0
}

// normalizeScore normalizes a value to 0-1 score
func (e *PerformanceOptimizationEngine) normalizeScore(value, min, max float64, lowerIsBetter bool) float64 {
	if max <= min {
		return 1.0
	}
	
	normalized := (value - min) / (max - min)
	if normalized < 0 {
		normalized = 0
	} else if normalized > 1 {
		normalized = 1
	}
	
	if lowerIsBetter {
		return 1.0 - normalized
	}
	
	return normalized
}

// calculateResourceVariance calculates recent resource usage variance
func (e *PerformanceOptimizationEngine) calculateResourceVariance() float64 {
	metrics := e.resourceManager.GetMetrics()
	if len(metrics.StatsHistory) < 2 {
		return 0.0
	}
	
	// Calculate variance over last 10 data points
	historySize := len(metrics.StatsHistory)
	startIdx := historySize - 10
	if startIdx < 0 {
		startIdx = 0
	}
	
	var memoryValues, cpuValues []float64
	for i := startIdx; i < historySize; i++ {
		memoryValues = append(memoryValues, metrics.StatsHistory[i].MemoryPercent)
		cpuValues = append(cpuValues, metrics.StatsHistory[i].CPUUsage)
	}
	
	memoryVariance := e.calculateVariance(memoryValues)
	cpuVariance := e.calculateVariance(cpuValues)
	
	// Normalize variance to 0-1 scale
	normalizedVariance := (memoryVariance + cpuVariance) / 200.0
	if normalizedVariance > 1.0 {
		normalizedVariance = 1.0
	}
	
	return normalizedVariance
}

// calculateVariance calculates statistical variance
func (e *PerformanceOptimizationEngine) calculateVariance(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}
	
	// Calculate mean
	var sum float64
	for _, v := range values {
		sum += v
	}
	mean := sum / float64(len(values))
	
	// Calculate variance
	var varianceSum float64
	for _, v := range values {
		diff := v - mean
		varianceSum += diff * diff
	}
	
	return varianceSum / float64(len(values))
}

// recordOptimizationHistory records optimization in history
func (e *PerformanceOptimizationEngine) recordOptimizationHistory(optimization *Optimization, duration time.Duration) {
	entry := &OptimizationHistoryEntry{
		Timestamp:   time.Now(),
		Type:        optimization.Type,
		Success:     optimization.Status == StatusCompleted,
		Duration:    duration,
		// ResourcesBefore and ResourcesAfter would be collected during optimization
	}
	
	if optimization.Results != nil {
		entry.Improvement = optimization.Results.ImprovementPercent
	}
	
	e.metrics.mu.Lock()
	e.metrics.OptimizationHistory = append(e.metrics.OptimizationHistory, entry)
	
	// Keep only recent history
	if len(e.metrics.OptimizationHistory) > e.config.MaxHistorySize {
		start := len(e.metrics.OptimizationHistory) - e.config.MaxHistorySize
		e.metrics.OptimizationHistory = e.metrics.OptimizationHistory[start:]
	}
	e.metrics.mu.Unlock()
}

// updateOptimizationMetrics updates optimization metrics
func (e *PerformanceOptimizationEngine) updateOptimizationMetrics(optimization *Optimization) {
	e.metrics.mu.Lock()
	defer e.metrics.mu.Unlock()
	
	e.metrics.TotalOptimizations++
	
	if optimization.Status == StatusCompleted {
		e.metrics.SuccessfulOptimizations++
		if optimization.Results != nil {
			// Update average improvement
			currentAvg := e.metrics.AverageImprovement
			totalSuccess := float64(e.metrics.SuccessfulOptimizations)
			e.metrics.AverageImprovement = (currentAvg*(totalSuccess-1) + optimization.Results.ImprovementPercent) / totalSuccess
		}
	} else if optimization.Status == StatusFailed {
		e.metrics.FailedOptimizations++
	}
	
	// Update optimizations by type
	if e.metrics.OptimizationsByType == nil {
		e.metrics.OptimizationsByType = make(map[OptimizationType]int64)
	}
	e.metrics.OptimizationsByType[optimization.Type]++
}

// GetCurrentProfile returns the current performance profile
func (e *PerformanceOptimizationEngine) GetCurrentProfile() *PerformanceProfile {
	e.mu.RLock()
	defer e.mu.RUnlock()
	
	// Return a copy
	profile := *e.currentProfile
	return &profile
}

// SetPerformanceProfile sets a new performance profile
func (e *PerformanceOptimizationEngine) SetPerformanceProfile(profile *PerformanceProfile) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	
	// Validate profile
	if err := e.validatePerformanceProfile(profile); err != nil {
		return fmt.Errorf("invalid performance profile: %w", err)
	}
	
	// Deactivate current profile
	if e.currentProfile != nil {
		e.currentProfile.IsActive = false
	}
	
	// Activate new profile
	profile.IsActive = true
	profile.LastUpdated = time.Now()
	e.currentProfile = profile
	
	e.logger.Info("Performance profile updated", 
		"name", profile.Name,
		"mode", profile.Mode)
	
	return nil
}

// validatePerformanceProfile validates a performance profile
func (e *PerformanceOptimizationEngine) validatePerformanceProfile(profile *PerformanceProfile) error {
	if profile == nil {
		return fmt.Errorf("profile cannot be nil")
	}
	
	if profile.Name == "" {
		return fmt.Errorf("profile name is required")
	}
	
	if profile.Targets == nil {
		return fmt.Errorf("performance targets are required")
	}
	
	if profile.Constraints == nil {
		return fmt.Errorf("resource constraints are required")
	}
	
	// Validate targets
	if profile.Targets.MaxResponseTimeMs <= 0 {
		return fmt.Errorf("max response time must be positive")
	}
	
	if profile.Targets.MinThroughputRPS < 0 {
		return fmt.Errorf("min throughput cannot be negative")
	}
	
	// Validate constraints
	if profile.Constraints.MaxMemoryMB <= 0 {
		return fmt.Errorf("max memory must be positive")
	}
	
	return nil
}

// GetOptimizationState returns current optimization state
func (e *PerformanceOptimizationEngine) GetOptimizationState() *OptimizationState {
	e.optimizationState.mu.RLock()
	defer e.optimizationState.mu.RUnlock()
	
	// Return a copy
	state := &OptimizationState{
		CurrentMode:           e.optimizationState.CurrentMode,
		ActiveOptimizations:   make(map[string]*Optimization),
		ScheduledOptimizations: make([]*ScheduledOptimization, len(e.optimizationState.ScheduledOptimizations)),
		PerformanceScore:      e.optimizationState.PerformanceScore,
		EfficiencyScore:       e.optimizationState.EfficiencyScore,
		StabilityScore:        e.optimizationState.StabilityScore,
		LastOptimization:      e.optimizationState.LastOptimization,
		OptimizationCycle:     e.optimizationState.OptimizationCycle,
	}
	
	// Copy active optimizations
	for k, v := range e.optimizationState.ActiveOptimizations {
		opt := *v
		state.ActiveOptimizations[k] = &opt
	}
	
	// Copy scheduled optimizations
	copy(state.ScheduledOptimizations, e.optimizationState.ScheduledOptimizations)
	
	return state
}

// GetMetrics returns performance optimization metrics
func (e *PerformanceOptimizationEngine) GetMetrics() *PerformanceOptimizationMetrics {
	e.metrics.mu.RLock()
	defer e.metrics.mu.RUnlock()
	
	// Return a copy
	metrics := &PerformanceOptimizationMetrics{
		TotalOptimizations:        e.metrics.TotalOptimizations,
		SuccessfulOptimizations:   e.metrics.SuccessfulOptimizations,
		FailedOptimizations:       e.metrics.FailedOptimizations,
		AverageImprovement:        e.metrics.AverageImprovement,
		TotalResourceSavings:      make(map[string]float64),
		OptimizationsByType:       make(map[OptimizationType]int64),
		OptimizationHistory:       make([]*OptimizationHistoryEntry, len(e.metrics.OptimizationHistory)),
		CurrentPerformanceScore:   e.metrics.CurrentPerformanceScore,
		BaselinePerformanceScore:  e.metrics.BaselinePerformanceScore,
	}
	
	// Copy maps
	for k, v := range e.metrics.TotalResourceSavings {
		metrics.TotalResourceSavings[k] = v
	}
	for k, v := range e.metrics.OptimizationsByType {
		metrics.OptimizationsByType[k] = v
	}
	
	// Copy history
	copy(metrics.OptimizationHistory, e.metrics.OptimizationHistory)
	
	return metrics
}

// Helper functions and constructor functions

// createDefaultPerformanceProfile creates a default performance profile
func createDefaultPerformanceProfile() *PerformanceProfile {
	return &PerformanceProfile{
		Name: "default",
		Mode: ModeBalanced,
		Targets: &PerformanceTargets{
			MaxResponseTimeMs: 200,
			MinThroughputRPS:  100,
			MaxErrorRate:      0.01,
			MinCacheHitRate:   0.8,
			MaxMemoryUsage:    80.0,
			MaxCPUUsage:       75.0,
			MinSystemStability: 0.95,
		},
		Constraints: &ResourceConstraints{
			MaxMemoryMB:          1024,
			MaxCPUCores:          4,
			MaxConnections:       1000,
			MaxGoroutines:        2000,
			MaxCacheSize:         512,
			MaxBandwidthMBps:     100,
			OptimizationInterval: 30 * time.Second,
		},
		Strategies:  []OptimizationStrategy{StrategyReactive, StrategyPreemptive},
		CreatedAt:   time.Now(),
		LastUpdated: time.Now(),
		IsActive:    true,
	}
}

// NewOptimizationState creates a new optimization state
func NewOptimizationState() *OptimizationState {
	return &OptimizationState{
		CurrentMode:            ModeBalanced,
		ActiveOptimizations:    make(map[string]*Optimization),
		ScheduledOptimizations: make([]*ScheduledOptimization, 0),
		OptimizationCycle:      0,
	}
}

// NewPerformanceOptimizationMetrics creates new performance optimization metrics
func NewPerformanceOptimizationMetrics() *PerformanceOptimizationMetrics {
	return &PerformanceOptimizationMetrics{
		TotalResourceSavings:    make(map[string]float64),
		OptimizationsByType:     make(map[OptimizationType]int64),
		OptimizationHistory:     make([]*OptimizationHistoryEntry, 0),
		BaselinePerformanceScore: 0.0,
	}
}