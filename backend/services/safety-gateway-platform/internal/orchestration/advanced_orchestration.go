package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// AdvancedOrchestrationEngine provides enhanced orchestration capabilities
type AdvancedOrchestrationEngine struct {
	*SnapshotOrchestrationEngine
	batchProcessor    *BatchProcessor
	loadBalancer      *IntelligentLoadBalancer
	routingEngine     *RoutingEngine
	metricsCollector  *OrchestrationMetrics
	config            *config.AdvancedOrchestrationConfig
	logger            *logger.Logger
}

// BatchProcessor handles batch processing of safety requests
type BatchProcessor struct {
	maxBatchSize     int
	batchTimeout     time.Duration
	concurrency      int
	pendingRequests  []*types.SafetyRequest
	pendingResponses chan *types.BatchProcessingResult
	mu               sync.RWMutex
	logger           *logger.Logger
}

// IntelligentLoadBalancer manages engine load balancing
type IntelligentLoadBalancer struct {
	engineMetrics    map[string]*EngineMetrics
	routingStrategy  string
	adaptiveWeights  map[string]float64
	mu               sync.RWMutex
	logger           *logger.Logger
}

// RoutingEngine handles intelligent request routing
type RoutingEngine struct {
	routingRules     []RoutingRule
	enginePriorities map[string]int
	fallbackChains   map[string][]string
	logger           *logger.Logger
}

// OrchestrationMetrics collects advanced orchestration metrics
type OrchestrationMetrics struct {
	TotalRequests         int64
	BatchedRequests       int64
	CacheHitRatio         float64
	AverageResponseTime   time.Duration
	EngineUtilization     map[string]float64
	LoadBalancingDecisions int64
	RoutingDecisions      int64
	mu                    sync.RWMutex
}

// EngineMetrics tracks individual engine performance
type EngineMetrics struct {
	RequestCount     int64
	ErrorCount       int64
	AverageLatency   time.Duration
	ThroughputPerSec float64
	LoadScore        float64
	LastUpdated      time.Time
	mu               sync.RWMutex
}

// RoutingRule defines request routing logic
type RoutingRule struct {
	Name        string
	Condition   func(*types.SafetyRequest) bool
	TargetTier  types.EngineTier
	Priority    int
	Enabled     bool
}

// BatchProcessingResult contains batch processing outcomes
type BatchProcessingResult struct {
	BatchID       string
	Responses     []*types.SafetyResponse
	ProcessedAt   time.Time
	TotalDuration time.Duration
	SuccessCount  int
	ErrorCount    int
	CacheHitCount int
}

// NewAdvancedOrchestrationEngine creates an advanced orchestration engine
func NewAdvancedOrchestrationEngine(
	snapshotEngine *SnapshotOrchestrationEngine,
	cfg *config.AdvancedOrchestrationConfig,
	logger *logger.Logger,
) *AdvancedOrchestrationEngine {
	// Initialize batch processor
	batchProcessor := &BatchProcessor{
		maxBatchSize:     cfg.BatchProcessing.MaxBatchSize,
		batchTimeout:     cfg.BatchProcessing.BatchTimeout,
		concurrency:      cfg.BatchProcessing.Concurrency,
		pendingRequests:  make([]*types.SafetyRequest, 0),
		pendingResponses: make(chan *types.BatchProcessingResult, cfg.BatchProcessing.MaxBatchSize),
		logger:           logger,
	}

	// Initialize load balancer
	loadBalancer := &IntelligentLoadBalancer{
		engineMetrics:   make(map[string]*EngineMetrics),
		routingStrategy: cfg.LoadBalancing.Strategy,
		adaptiveWeights: make(map[string]float64),
		logger:          logger,
	}

	// Initialize routing engine with default rules
	routingEngine := &RoutingEngine{
		routingRules:     createDefaultRoutingRules(),
		enginePriorities: cfg.Routing.EnginePriorities,
		fallbackChains:   cfg.Routing.FallbackChains,
		logger:           logger,
	}

	// Initialize metrics collector
	metricsCollector := &OrchestrationMetrics{
		EngineUtilization: make(map[string]float64),
	}

	return &AdvancedOrchestrationEngine{
		SnapshotOrchestrationEngine: snapshotEngine,
		batchProcessor:              batchProcessor,
		loadBalancer:                loadBalancer,
		routingEngine:               routingEngine,
		metricsCollector:            metricsCollector,
		config:                      cfg,
		logger:                      logger,
	}
}

// ProcessSafetyRequestAdvanced processes requests with advanced orchestration
func (a *AdvancedOrchestrationEngine) ProcessSafetyRequestAdvanced(
	ctx context.Context,
	req *types.SafetyRequest,
) (*types.SafetyResponse, error) {
	startTime := time.Now()
	requestLogger := a.logger.WithRequestID(req.RequestID).WithPatientID(req.PatientID)

	requestLogger.Info("Processing advanced safety request",
		zap.String("orchestration_mode", "advanced"),
		zap.String("action_type", req.ActionType),
		zap.String("priority", req.Priority),
	)

	// Update metrics
	a.metricsCollector.mu.Lock()
	a.metricsCollector.TotalRequests++
	a.metricsCollector.mu.Unlock()

	// 1. Intelligent routing decision
	routingDecision, err := a.routingEngine.DetermineRouting(req)
	if err != nil {
		return a.createErrorResponse(req, fmt.Errorf("routing decision failed: %w", err), startTime), nil
	}

	requestLogger.Debug("Routing decision made",
		zap.String("target_tier", string(routingDecision.TargetTier)),
		zap.Int("priority", routingDecision.Priority),
		zap.String("rule", routingDecision.RuleName),
	)

	// 2. Check for batch processing eligibility
	if a.shouldBatchProcess(req) {
		return a.processBatchRequest(ctx, req, routingDecision)
	}

	// 3. Load balancing for engine selection
	engines := a.registry.GetEnginesForTier(routingDecision.TargetTier)
	selectedEngines, err := a.loadBalancer.SelectEngines(engines, req)
	if err != nil {
		return a.createErrorResponse(req, fmt.Errorf("load balancing failed: %w", err), startTime), nil
	}

	requestLogger.Debug("Load balancing completed",
		zap.Int("available_engines", len(engines)),
		zap.Int("selected_engines", len(selectedEngines)),
		zap.String("strategy", a.loadBalancer.routingStrategy),
	)

	// 4. Execute with enhanced orchestration
	response, err := a.executeWithAdvancedOrchestration(ctx, req, selectedEngines, routingDecision)
	if err != nil {
		return a.createErrorResponse(req, err, startTime), nil
	}

	// 5. Update metrics and engine performance tracking
	duration := time.Since(startTime)
	a.updateEngineMetrics(selectedEngines, response, duration)
	a.updateOrchestrationMetrics(response, duration)

	response.ProcessingTime = duration
	response.Metadata["orchestration_mode"] = "advanced"
	response.Metadata["routing_rule"] = routingDecision.RuleName
	response.Metadata["load_balancing_strategy"] = a.loadBalancer.routingStrategy

	requestLogger.Info("Advanced safety request processed",
		zap.String("status", string(response.Status)),
		zap.Float64("risk_score", response.RiskScore),
		zap.Int64("processing_time_ms", response.ProcessingTime.Milliseconds()),
		zap.Int("engines_executed", len(response.EngineResults)),
	)

	return response, nil
}

// ProcessBatchRequests handles batch processing of multiple safety requests
func (a *AdvancedOrchestrationEngine) ProcessBatchRequests(
	ctx context.Context,
	requests []*types.SafetyRequest,
) (*BatchProcessingResult, error) {
	startTime := time.Now()
	batchID := fmt.Sprintf("batch_%d_%d", time.Now().Unix(), len(requests))

	a.logger.Info("Processing batch requests",
		zap.String("batch_id", batchID),
		zap.Int("request_count", len(requests)),
		zap.Int("max_batch_size", a.batchProcessor.maxBatchSize),
	)

	// Update metrics
	a.metricsCollector.mu.Lock()
	a.metricsCollector.BatchedRequests += int64(len(requests))
	a.metricsCollector.mu.Unlock()

	// Group requests by patient ID for snapshot optimization
	patientGroups := a.groupRequestsByPatient(requests)

	// Process groups concurrently
	resultChan := make(chan *types.SafetyResponse, len(requests))
	var wg sync.WaitGroup

	for patientID, patientRequests := range patientGroups {
		wg.Add(1)
		go func(pid string, reqs []*types.SafetyRequest) {
			defer wg.Done()
			a.processPatientBatch(ctx, pid, reqs, resultChan)
		}(patientID, patientRequests)
	}

	// Collect results
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	var responses []*types.SafetyResponse
	var successCount, errorCount, cacheHitCount int

	for response := range resultChan {
		responses = append(responses, response)
		
		switch response.Status {
		case types.SafetyStatusSafe, types.SafetyStatusWarning:
			successCount++
		case types.SafetyStatusUnsafe, types.SafetyStatusError:
			errorCount++
		}

		// Check for cache hits in metadata
		if mode, exists := response.Metadata["processing_mode"]; exists && mode == "snapshot_based" {
			cacheHitCount++
		}
	}

	result := &BatchProcessingResult{
		BatchID:       batchID,
		Responses:     responses,
		ProcessedAt:   startTime,
		TotalDuration: time.Since(startTime),
		SuccessCount:  successCount,
		ErrorCount:    errorCount,
		CacheHitCount: cacheHitCount,
	}

	a.logger.Info("Batch processing completed",
		zap.String("batch_id", batchID),
		zap.Int("total_requests", len(requests)),
		zap.Int("successful_responses", successCount),
		zap.Int("error_responses", errorCount),
		zap.Int("cache_hits", cacheHitCount),
		zap.Int64("total_duration_ms", result.TotalDuration.Milliseconds()),
	)

	return result, nil
}

// DetermineRouting makes intelligent routing decisions
func (r *RoutingEngine) DetermineRouting(req *types.SafetyRequest) (*RoutingDecision, error) {
	// Sort rules by priority
	activeRules := make([]RoutingRule, 0)
	for _, rule := range r.routingRules {
		if rule.Enabled {
			activeRules = append(activeRules, rule)
		}
	}

	// Evaluate rules in priority order
	for _, rule := range activeRules {
		if rule.Condition(req) {
			r.logger.Debug("Routing rule matched",
				zap.String("rule_name", rule.Name),
				zap.String("target_tier", string(rule.TargetTier)),
				zap.Int("priority", rule.Priority),
			)

			return &RoutingDecision{
				RuleName:   rule.Name,
				TargetTier: rule.TargetTier,
				Priority:   rule.Priority,
			}, nil
		}
	}

	// Default routing
	r.logger.Debug("No routing rule matched, using default",
		zap.String("default_tier", string(types.TierVetoCritical)),
	)

	return &RoutingDecision{
		RuleName:   "default",
		TargetTier: types.TierVetoCritical,
		Priority:   0,
	}, nil
}

// SelectEngines performs intelligent load balancing
func (lb *IntelligentLoadBalancer) SelectEngines(
	availableEngines []*registry.EngineInfo,
	req *types.SafetyRequest,
) ([]*registry.EngineInfo, error) {
	if len(availableEngines) == 0 {
		return nil, fmt.Errorf("no engines available")
	}

	switch lb.routingStrategy {
	case "round_robin":
		return lb.selectRoundRobin(availableEngines)
	case "least_loaded":
		return lb.selectLeastLoaded(availableEngines)
	case "performance_weighted":
		return lb.selectPerformanceWeighted(availableEngines)
	case "adaptive":
		return lb.selectAdaptive(availableEngines, req)
	default:
		// Default to all engines
		return availableEngines, nil
	}
}

// selectAdaptive uses adaptive load balancing based on real-time metrics
func (lb *IntelligentLoadBalancer) selectAdaptive(
	engines []*registry.EngineInfo,
	req *types.SafetyRequest,
) ([]*registry.EngineInfo, error) {
	lb.mu.RLock()
	defer lb.mu.RUnlock()

	// Calculate adaptive weights based on performance metrics
	totalWeight := 0.0
	engineWeights := make(map[string]float64)

	for _, engine := range engines {
		metrics, exists := lb.engineMetrics[engine.ID]
		if !exists {
			// New engine, assign default weight
			engineWeights[engine.ID] = 1.0
		} else {
			// Calculate weight based on performance metrics
			errorRate := float64(metrics.ErrorCount) / float64(metrics.RequestCount+1)
			latencyScore := 1.0 / (metrics.AverageLatency.Seconds() + 0.001)
			throughputScore := metrics.ThroughputPerSec
			
			// Weighted combination (lower error rate, lower latency, higher throughput = higher weight)
			weight := (1.0 - errorRate) * latencyScore * (throughputScore + 1.0)
			engineWeights[engine.ID] = weight
		}
		totalWeight += engineWeights[engine.ID]
	}

	// Select engines based on weighted probabilities
	// For simplicity, select top 50% of engines by weight
	var selectedEngines []*registry.EngineInfo
	for _, engine := range engines {
		weight := engineWeights[engine.ID]
		if weight >= totalWeight*0.3 { // Top engines with weight above 30% of total
			selectedEngines = append(selectedEngines, engine)
		}
	}

	// Ensure at least one engine is selected
	if len(selectedEngines) == 0 {
		selectedEngines = engines[:1]
	}

	lb.logger.Debug("Adaptive load balancing completed",
		zap.Int("total_engines", len(engines)),
		zap.Int("selected_engines", len(selectedEngines)),
		zap.Float64("total_weight", totalWeight),
	)

	return selectedEngines, nil
}

// Helper methods and supporting functions

type RoutingDecision struct {
	RuleName   string
	TargetTier types.EngineTier
	Priority   int
}

func createDefaultRoutingRules() []RoutingRule {
	return []RoutingRule{
		{
			Name:     "high_priority_critical",
			Condition: func(req *types.SafetyRequest) bool {
				return req.Priority == "critical" || req.Priority == "high"
			},
			TargetTier: types.TierVetoCritical,
			Priority:   100,
			Enabled:    true,
		},
		{
			Name:     "medication_interaction",
			Condition: func(req *types.SafetyRequest) bool {
				return req.ActionType == "medication_interaction" && len(req.MedicationIDs) > 1
			},
			TargetTier: types.TierVetoCritical,
			Priority:   90,
			Enabled:    true,
		},
		{
			Name:     "routine_advisory",
			Condition: func(req *types.SafetyRequest) bool {
				return req.Priority == "low" || req.Priority == "routine"
			},
			TargetTier: types.TierAdvisory,
			Priority:   10,
			Enabled:    true,
		},
	}
}

func (a *AdvancedOrchestrationEngine) shouldBatchProcess(req *types.SafetyRequest) bool {
	// Simple heuristic - could be enhanced with ML
	return req.Priority == "routine" || req.Priority == "low"
}

func (a *AdvancedOrchestrationEngine) groupRequestsByPatient(
	requests []*types.SafetyRequest,
) map[string][]*types.SafetyRequest {
	groups := make(map[string][]*types.SafetyRequest)
	for _, req := range requests {
		groups[req.PatientID] = append(groups[req.PatientID], req)
	}
	return groups
}

func (a *AdvancedOrchestrationEngine) processPatientBatch(
	ctx context.Context,
	patientID string,
	requests []*types.SafetyRequest,
	resultChan chan<- *types.SafetyResponse,
) {
	for _, req := range requests {
		response, err := a.SnapshotOrchestrationEngine.ProcessSafetyRequest(ctx, req)
		if err != nil {
			// Create error response
			response = a.createErrorResponse(req, err, time.Now())
		}
		resultChan <- response
	}
}

func (a *AdvancedOrchestrationEngine) executeWithAdvancedOrchestration(
	ctx context.Context,
	req *types.SafetyRequest,
	engines []*registry.EngineInfo,
	routingDecision *RoutingDecision,
) (*types.SafetyResponse, error) {
	// Use snapshot orchestration as base, but with selected engines
	if a.config.Enabled && a.hasSnapshotReference(req) {
		// Get snapshot and execute with selected engines
		snapshotRef := a.extractSnapshotReference(req)
		snapshot, err := a.getValidatedSnapshot(ctx, snapshotRef.SnapshotID, a.logger)
		if err != nil {
			return nil, fmt.Errorf("snapshot retrieval failed: %w", err)
		}

		// Execute only selected engines
		results := make([]types.EngineResult, 0)
		for _, engine := range engines {
			result := a.executeEngineInProcessWithSnapshot(ctx, engine, req, snapshot, a.logger)
			results = append(results, result)
		}

		response := a.aggregateWithSnapshot(req, results, snapshot)
		return response, nil
	}

	// Fallback to legacy with selected engines
	return a.SnapshotOrchestrationEngine.ProcessSafetyRequest(ctx, req)
}

func (a *AdvancedOrchestrationEngine) updateEngineMetrics(
	engines []*registry.EngineInfo,
	response *types.SafetyResponse,
	duration time.Duration,
) {
	a.loadBalancer.mu.Lock()
	defer a.loadBalancer.mu.Unlock()

	for _, engine := range engines {
		metrics, exists := a.loadBalancer.engineMetrics[engine.ID]
		if !exists {
			metrics = &EngineMetrics{}
			a.loadBalancer.engineMetrics[engine.ID] = metrics
		}

		metrics.mu.Lock()
		metrics.RequestCount++
		if response.Status == types.SafetyStatusError {
			metrics.ErrorCount++
		}
		
		// Update running average latency
		if metrics.RequestCount == 1 {
			metrics.AverageLatency = duration
		} else {
			metrics.AverageLatency = time.Duration(
				(int64(metrics.AverageLatency)*metrics.RequestCount + int64(duration)) / 
				(metrics.RequestCount + 1),
			)
		}
		
		metrics.LastUpdated = time.Now()
		metrics.mu.Unlock()
	}
}

func (a *AdvancedOrchestrationEngine) updateOrchestrationMetrics(
	response *types.SafetyResponse,
	duration time.Duration,
) {
	a.metricsCollector.mu.Lock()
	defer a.metricsCollector.mu.Unlock()

	// Update running average response time
	if a.metricsCollector.TotalRequests == 1 {
		a.metricsCollector.AverageResponseTime = duration
	} else {
		totalTime := int64(a.metricsCollector.AverageResponseTime) * a.metricsCollector.TotalRequests
		a.metricsCollector.AverageResponseTime = time.Duration(
			(totalTime + int64(duration)) / a.metricsCollector.TotalRequests,
		)
	}
}

func (lb *IntelligentLoadBalancer) selectRoundRobin(engines []*registry.EngineInfo) ([]*registry.EngineInfo, error) {
	// Simple round robin - could maintain state for true round robin
	return engines, nil
}

func (lb *IntelligentLoadBalancer) selectLeastLoaded(engines []*registry.EngineInfo) ([]*registry.EngineInfo, error) {
	// Select engines with lowest request count
	minCount := int64(^uint64(0) >> 1) // Max int64
	var leastLoaded []*registry.EngineInfo

	lb.mu.RLock()
	defer lb.mu.RUnlock()

	for _, engine := range engines {
		if metrics, exists := lb.engineMetrics[engine.ID]; exists {
			if metrics.RequestCount < minCount {
				minCount = metrics.RequestCount
				leastLoaded = []*registry.EngineInfo{engine}
			} else if metrics.RequestCount == minCount {
				leastLoaded = append(leastLoaded, engine)
			}
		} else {
			// New engine with 0 requests
			if minCount > 0 {
				minCount = 0
				leastLoaded = []*registry.EngineInfo{engine}
			} else {
				leastLoaded = append(leastLoaded, engine)
			}
		}
	}

	return leastLoaded, nil
}

func (lb *IntelligentLoadBalancer) selectPerformanceWeighted(engines []*registry.EngineInfo) ([]*registry.EngineInfo, error) {
	// Select engines based on performance metrics
	return lb.selectAdaptive(engines, nil) // Reuse adaptive logic
}

// GetOrchestrationStats returns comprehensive orchestration statistics
func (a *AdvancedOrchestrationEngine) GetOrchestrationStats() map[string]interface{} {
	a.metricsCollector.mu.RLock()
	defer a.metricsCollector.mu.RUnlock()

	stats := make(map[string]interface{})
	
	// Basic metrics
	stats["total_requests"] = a.metricsCollector.TotalRequests
	stats["batched_requests"] = a.metricsCollector.BatchedRequests
	stats["average_response_time_ms"] = a.metricsCollector.AverageResponseTime.Milliseconds()
	stats["load_balancing_decisions"] = a.metricsCollector.LoadBalancingDecisions
	stats["routing_decisions"] = a.metricsCollector.RoutingDecisions

	// Engine utilization
	stats["engine_utilization"] = a.metricsCollector.EngineUtilization

	// Load balancer stats
	a.loadBalancer.mu.RLock()
	engineStats := make(map[string]interface{})
	for engineID, metrics := range a.loadBalancer.engineMetrics {
		metrics.mu.RLock()
		engineStats[engineID] = map[string]interface{}{
			"request_count":      metrics.RequestCount,
			"error_count":        metrics.ErrorCount,
			"average_latency_ms": metrics.AverageLatency.Milliseconds(),
			"throughput_per_sec": metrics.ThroughputPerSec,
			"load_score":         metrics.LoadScore,
		}
		metrics.mu.RUnlock()
	}
	a.loadBalancer.mu.RUnlock()
	
	stats["engine_metrics"] = engineStats
	stats["routing_strategy"] = a.loadBalancer.routingStrategy

	// Include snapshot stats from parent
	if snapshotStats := a.SnapshotOrchestrationEngine.GetSnapshotStats(); snapshotStats != nil {
		stats["snapshot_stats"] = snapshotStats
	}

	return stats
}