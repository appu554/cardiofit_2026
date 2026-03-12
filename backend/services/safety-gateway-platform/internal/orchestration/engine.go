package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/internal/config"
	"safety-gateway-platform/internal/registry"
	"safety-gateway-platform/pkg/logger"
	"safety-gateway-platform/pkg/types"
)

// OrchestrationEngine coordinates safety engine execution
type OrchestrationEngine struct {
	registry        *registry.EngineRegistry
	contextService  ContextAssemblyService
	responseBuilder *ResponseBuilder
	circuitBreaker  *CircuitBreaker
	config          *config.Config
	logger          *logger.Logger
}

// ContextAssemblyService interface for context assembly
type ContextAssemblyService interface {
	AssembleContext(ctx context.Context, patientID string) (*types.ClinicalContext, error)
}

// NewOrchestrationEngine creates a new orchestration engine
func NewOrchestrationEngine(
	registry *registry.EngineRegistry,
	contextService ContextAssemblyService,
	cfg *config.Config,
	logger *logger.Logger,
) *OrchestrationEngine {
	return &OrchestrationEngine{
		registry:        registry,
		contextService:  contextService,
		responseBuilder: NewResponseBuilder(logger),
		circuitBreaker:  NewCircuitBreaker(cfg.CircuitBreaker, logger),
		config:          cfg,
		logger:          logger,
	}
}

// ProcessSafetyRequest processes a safety validation request
func (o *OrchestrationEngine) ProcessSafetyRequest(ctx context.Context, req *types.SafetyRequest) (*types.SafetyResponse, error) {
	startTime := time.Now()
	requestLogger := o.logger.WithRequestID(req.RequestID).WithPatientID(req.PatientID)

	requestLogger.Info("Processing safety request",
		zap.String("action_type", req.ActionType),
		zap.String("priority", req.Priority),
		zap.Int("medication_count", len(req.MedicationIDs)),
		zap.Int("condition_count", len(req.ConditionIDs)),
	)

	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, o.config.GetRequestTimeout())
	defer cancel()

	// Assemble clinical context (if enabled)
	var clinicalContext *types.ClinicalContext
	var err error

	if o.config.ContextAssembly.Enabled {
		clinicalContext, err = o.assembleContext(ctx, req.PatientID, requestLogger)
		if err != nil {
			requestLogger.Error("Context assembly failed", zap.Error(err))
			return o.createErrorResponse(req, fmt.Errorf("context assembly failed: %w", err), startTime), nil
		}
	} else {
		// Skip context assembly - let CAE handle its own context
		requestLogger.Debug("Context assembly disabled - skipping patient data fetch")
		clinicalContext = &types.ClinicalContext{
			PatientID: req.PatientID,
			// Empty context - CAE will handle its own data fetching
		}
	}

	// Get applicable engines
	engines := o.registry.GetEnginesForRequest(req)
	if len(engines) == 0 {
		requestLogger.Warn("No engines available for request")
		return o.createErrorResponse(req, fmt.Errorf("no engines available"), startTime), nil
	}

	requestLogger.Debug("Selected engines for execution",
		zap.Int("engine_count", len(engines)),
		zap.Strings("engines", o.getEngineIDs(engines)),
	)

	// Execute engines in parallel with timeout
	engineCtx, engineCancel := context.WithTimeout(ctx, o.config.GetEngineExecutionTimeout())
	defer engineCancel()

	results := o.executeEnginesParallel(engineCtx, engines, req, clinicalContext, requestLogger)

	// Aggregate results
	response := o.responseBuilder.AggregateResults(req, results, clinicalContext)
	response.ProcessingTime = time.Since(startTime)

	// Log final decision
	requestLogger.LogSafetyDecision(
		req.RequestID,
		req.PatientID,
		string(response.Status),
		response.RiskScore,
		response.ProcessingTime.Milliseconds(),
		o.getEngineResultIDs(results),
	)

	requestLogger.Info("Safety request processed",
		zap.String("status", string(response.Status)),
		zap.Float64("risk_score", response.RiskScore),
		zap.Int64("processing_time_ms", response.ProcessingTime.Milliseconds()),
		zap.Int("engines_executed", len(results)),
	)

	return response, nil
}

// assembleContext assembles clinical context with timeout
func (o *OrchestrationEngine) assembleContext(ctx context.Context, patientID string, logger *logger.Logger) (*types.ClinicalContext, error) {
	contextCtx, cancel := context.WithTimeout(ctx, o.config.GetContextAssemblyTimeout())
	defer cancel()

	startTime := time.Now()
	
	context, err := o.contextService.AssembleContext(contextCtx, patientID)
	duration := time.Since(startTime)

	if err != nil {
		logger.Error("Context assembly failed",
			zap.String("patient_id", patientID),
			zap.Int64("duration_ms", duration.Milliseconds()),
			zap.Error(err),
		)
		return nil, err
	}

	logger.Debug("Context assembled successfully",
		zap.String("patient_id", patientID),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.String("context_version", context.ContextVersion),
		zap.Strings("data_sources", context.DataSources),
	)

	return context, nil
}

// executeEnginesParallel executes engines in parallel
func (o *OrchestrationEngine) executeEnginesParallel(
	ctx context.Context,
	engines []*registry.EngineInfo,
	req *types.SafetyRequest,
	clinicalContext *types.ClinicalContext,
	logger *logger.Logger,
) []types.EngineResult {
	resultsChan := make(chan types.EngineResult, len(engines))
	var wg sync.WaitGroup

	// Execute each engine in a separate goroutine
	for _, engine := range engines {
		wg.Add(1)
		go func(eng *registry.EngineInfo) {
			defer wg.Done()
			result := o.executeEngineInProcess(ctx, eng, req, clinicalContext, logger)
			resultsChan <- result
		}(engine)
	}

	// Wait for all engines to complete or timeout
	go func() {
		wg.Wait()
		close(resultsChan)
	}()

	// Collect results
	var results []types.EngineResult
	for {
		select {
		case result, ok := <-resultsChan:
			if !ok {
				// Channel closed, all engines completed
				return results
			}
			results = append(results, result)
		case <-ctx.Done():
			// Timeout - return what we have
			logger.Warn("Engine execution timeout, returning partial results",
				zap.Int("completed_engines", len(results)),
				zap.Int("total_engines", len(engines)),
			)
			return results
		}
	}
}

// executeEngineInProcess executes a single engine in-process
func (o *OrchestrationEngine) executeEngineInProcess(
	ctx context.Context,
	engineInfo *registry.EngineInfo,
	req *types.SafetyRequest,
	clinicalContext *types.ClinicalContext,
	logger *logger.Logger,
) types.EngineResult {
	return o.executeEngineInProcessInternal(ctx, engineInfo, req, clinicalContext, nil, logger)
}

// executeEngineInProcessWithSnapshot executes a single engine with snapshot data
func (o *OrchestrationEngine) executeEngineInProcessWithSnapshot(
	ctx context.Context,
	engineInfo *registry.EngineInfo,
	req *types.SafetyRequest,
	snapshot *types.ClinicalSnapshot,
	logger *logger.Logger,
) types.EngineResult {
	return o.executeEngineInProcessInternal(ctx, engineInfo, req, nil, snapshot, logger)
}

// executeEngineInProcessInternal executes a single engine with either legacy context or snapshot
func (o *OrchestrationEngine) executeEngineInProcessInternal(
	ctx context.Context,
	engineInfo *registry.EngineInfo,
	req *types.SafetyRequest,
	clinicalContext *types.ClinicalContext,
	snapshot *types.ClinicalSnapshot,
	logger *logger.Logger,
) types.EngineResult {
	startTime := time.Now()
	engineLogger := logger.WithEngine(engineInfo.ID)

	// Create engine-specific context with timeout
	engineCtx, cancel := context.WithTimeout(ctx, engineInfo.Timeout)
	defer cancel()

	// Determine processing mode
	processingMode := "legacy"
	if snapshot != nil {
		processingMode = "snapshot_based"
	}

	engineLogger.Debug("Executing engine",
		zap.String("engine_name", engineInfo.Name),
		zap.String("tier", string(engineInfo.Tier)),
		zap.String("processing_mode", processingMode),
		zap.Int64("timeout_ms", engineInfo.Timeout.Milliseconds()),
	)

	// Execute engine through circuit breaker
	var result *types.EngineResult
	var err error

	circuitBreakerErr := o.circuitBreaker.Execute(engineInfo.ID, func() error {
		// Choose execution path based on data availability
		if snapshot != nil {
			// Try snapshot-aware execution first
			if snapshotEngine, ok := engineInfo.Instance.(types.SnapshotAwareEngine); ok {
				engineLogger.Debug("Using snapshot-aware execution")
				result, err = snapshotEngine.EvaluateWithSnapshot(engineCtx, req, snapshot)
			} else {
				// Fallback to legacy execution with snapshot's clinical data
				engineLogger.Debug("Using legacy execution with snapshot data")
				result, err = engineInfo.Instance.Evaluate(engineCtx, req, snapshot.Data)
			}
		} else {
			// Legacy execution
			engineLogger.Debug("Using legacy execution")
			result, err = engineInfo.Instance.Evaluate(engineCtx, req, clinicalContext)
		}
		return err
	})

	duration := time.Since(startTime)

	// Handle circuit breaker errors
	if circuitBreakerErr != nil {
		engineLogger.Warn("Circuit breaker prevented engine execution",
			zap.Error(circuitBreakerErr),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)
		return o.createEngineErrorResult(engineInfo, circuitBreakerErr, duration)
	}

	// Handle engine execution errors
	if err != nil {
		engineLogger.Error("Engine execution failed",
			zap.Error(err),
			zap.Int64("duration_ms", duration.Milliseconds()),
		)
		return o.createEngineErrorResult(engineInfo, err, duration)
	}

	// Handle nil result
	if result == nil {
		err := fmt.Errorf("engine returned nil result")
		engineLogger.Error("Engine returned nil result", zap.Int64("duration_ms", duration.Milliseconds()))
		return o.createEngineErrorResult(engineInfo, err, duration)
	}

	// Populate result metadata
	result.EngineID = engineInfo.ID
	result.EngineName = engineInfo.Name
	result.Duration = duration
	result.Tier = engineInfo.Tier

	engineLogger.Debug("Engine execution completed",
		zap.String("status", string(result.Status)),
		zap.Float64("risk_score", result.RiskScore),
		zap.Int64("duration_ms", duration.Milliseconds()),
		zap.Int("violations", len(result.Violations)),
	)

	// Log engine execution for audit
	logger.LogEngineExecution(
		engineInfo.ID,
		string(result.Status),
		duration.Milliseconds(),
		result.Error,
	)

	return *result
}

// createEngineErrorResult creates an error result for a failed engine
func (o *OrchestrationEngine) createEngineErrorResult(
	engineInfo *registry.EngineInfo,
	err error,
	duration time.Duration,
) types.EngineResult {
	// Determine status based on engine tier
	var status types.SafetyStatus
	if engineInfo.Tier == types.TierVetoCritical {
		status = types.SafetyStatusUnsafe // Fail closed for critical engines
	} else {
		status = types.SafetyStatusWarning // Degraded for advisory engines
	}

	return types.EngineResult{
		EngineID:   engineInfo.ID,
		EngineName: engineInfo.Name,
		Status:     status,
		RiskScore:  1.0, // Maximum risk for failed engines
		Violations: []string{fmt.Sprintf("Engine execution failed: %s", err.Error())},
		Confidence: 0.0, // No confidence in failed results
		Duration:   duration,
		Tier:       engineInfo.Tier,
		Error:      err.Error(),
	}
}

// createErrorResponse creates an error response
func (o *OrchestrationEngine) createErrorResponse(req *types.SafetyRequest, err error, startTime time.Time) *types.SafetyResponse {
	return &types.SafetyResponse{
		RequestID:      req.RequestID,
		Status:         types.SafetyStatusError,
		RiskScore:      1.0,
		EngineResults:  []types.EngineResult{},
		ProcessingTime: time.Since(startTime),
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"error": err.Error(),
		},
	}
}

// getEngineIDs extracts engine IDs from engine info slice
func (o *OrchestrationEngine) getEngineIDs(engines []*registry.EngineInfo) []string {
	ids := make([]string, len(engines))
	for i, engine := range engines {
		ids[i] = engine.ID
	}
	return ids
}

// getEngineResultIDs extracts engine IDs from engine results
func (o *OrchestrationEngine) getEngineResultIDs(results []types.EngineResult) []string {
	ids := make([]string, len(results))
	for i, result := range results {
		ids[i] = result.EngineID
	}
	return ids
}
