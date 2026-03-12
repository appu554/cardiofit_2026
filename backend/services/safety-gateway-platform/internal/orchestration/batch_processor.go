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

// BatchRequest represents a batch of safety requests
type BatchRequest struct {
	BatchID     string
	Requests    []*types.SafetyRequest
	SubmittedAt time.Time
	Priority    string
	Context     map[string]interface{}
}

// BatchResponse contains the results of batch processing
type BatchResponse struct {
	BatchID       string                     `json:"batch_id"`
	Responses     []*types.SafetyResponse    `json:"responses"`
	Summary       *BatchSummary              `json:"summary"`
	ProcessedAt   time.Time                  `json:"processed_at"`
	TotalDuration time.Duration              `json:"total_duration"`
	Metadata      map[string]interface{}     `json:"metadata,omitempty"`
}

// BatchSummary provides aggregate statistics for the batch
type BatchSummary struct {
	TotalRequests     int                    `json:"total_requests"`
	SuccessfulResults int                    `json:"successful_results"`
	ErrorResults      int                    `json:"error_results"`
	WarningResults    int                    `json:"warning_results"`
	UnsafeResults     int                    `json:"unsafe_results"`
	CacheHitCount     int                    `json:"cache_hit_count"`
	AverageRiskScore  float64                `json:"average_risk_score"`
	ProcessingStats   *ProcessingStatistics  `json:"processing_stats"`
}

// ProcessingStatistics contains detailed processing metrics
type ProcessingStatistics struct {
	SnapshotRetrievals   int           `json:"snapshot_retrievals"`
	CacheHits            int           `json:"cache_hits"`
	CacheMisses          int           `json:"cache_misses"`
	EngineExecutions     int           `json:"engine_executions"`
	AverageEngineLatency time.Duration `json:"average_engine_latency"`
	ParallelismAchieved  float64       `json:"parallelism_achieved"`
	ResourceUtilization  map[string]float64 `json:"resource_utilization"`
}

// EnhancedBatchProcessor provides advanced batch processing capabilities
type EnhancedBatchProcessor struct {
	orchestrator    *SnapshotOrchestrationEngine
	config          *config.BatchProcessingConfig
	logger          *logger.Logger
	
	// Batch management
	activeBatches   map[string]*BatchRequest
	batchResults    map[string]*BatchResponse
	
	// Concurrency control
	semaphore       chan struct{}
	workerPool      chan func()
	
	// Metrics
	metrics         *BatchProcessingMetrics
	
	// Synchronization
	mu              sync.RWMutex
}

// BatchProcessingMetrics tracks batch processing performance
type BatchProcessingMetrics struct {
	TotalBatches        int64
	TotalRequests       int64
	AverageBatchSize    float64
	AverageProcessingTime time.Duration
	ThroughputPerSecond float64
	ErrorRate           float64
	CacheHitRatio       float64
	ParallelismEfficiency float64
	mu                  sync.RWMutex
}

// NewEnhancedBatchProcessor creates a new enhanced batch processor
func NewEnhancedBatchProcessor(
	orchestrator *SnapshotOrchestrationEngine,
	cfg *config.BatchProcessingConfig,
	logger *logger.Logger,
) *EnhancedBatchProcessor {
	processor := &EnhancedBatchProcessor{
		orchestrator:  orchestrator,
		config:        cfg,
		logger:        logger,
		activeBatches: make(map[string]*BatchRequest),
		batchResults:  make(map[string]*BatchResponse),
		semaphore:     make(chan struct{}, cfg.Concurrency),
		workerPool:    make(chan func(), cfg.Concurrency*2),
		metrics:       &BatchProcessingMetrics{},
	}

	// Start worker pool
	for i := 0; i < cfg.Concurrency; i++ {
		go processor.worker()
	}

	return processor
}

// ProcessBatch processes a batch of safety requests with advanced optimizations
func (bp *EnhancedBatchProcessor) ProcessBatch(
	ctx context.Context,
	batch *BatchRequest,
) (*BatchResponse, error) {
	startTime := time.Now()

	bp.logger.Info("Processing enhanced batch request",
		zap.String("batch_id", batch.BatchID),
		zap.Int("request_count", len(batch.Requests)),
		zap.String("priority", batch.Priority),
	)

	// Update metrics
	bp.updateBatchMetrics(batch)

	// Validate batch
	if err := bp.validateBatch(batch); err != nil {
		return nil, fmt.Errorf("batch validation failed: %w", err)
	}

	// Store active batch
	bp.mu.Lock()
	bp.activeBatches[batch.BatchID] = batch
	bp.mu.Unlock()

	defer func() {
		bp.mu.Lock()
		delete(bp.activeBatches, batch.BatchID)
		bp.mu.Unlock()
	}()

	// Optimize batch processing strategy
	strategy := bp.determineBatchStrategy(batch)
	
	var response *BatchResponse
	var err error

	switch strategy {
	case "patient_grouped":
		response, err = bp.processPatientGroupedBatch(ctx, batch)
	case "snapshot_optimized":
		response, err = bp.processSnapshotOptimizedBatch(ctx, batch)
	case "parallel_direct":
		response, err = bp.processParallelBatch(ctx, batch)
	default:
		response, err = bp.processStandardBatch(ctx, batch)
	}

	if err != nil {
		bp.logger.Error("Batch processing failed",
			zap.String("batch_id", batch.BatchID),
			zap.String("strategy", strategy),
			zap.Error(err),
		)
		return nil, err
	}

	// Calculate final metrics and statistics
	response.TotalDuration = time.Since(startTime)
	response.Summary.ProcessingStats = bp.calculateProcessingStatistics(response)

	// Store result
	bp.mu.Lock()
	bp.batchResults[batch.BatchID] = response
	bp.mu.Unlock()

	bp.logger.Info("Enhanced batch processing completed",
		zap.String("batch_id", batch.BatchID),
		zap.Int("total_requests", response.Summary.TotalRequests),
		zap.Int("successful_results", response.Summary.SuccessfulResults),
		zap.Int("cache_hits", response.Summary.CacheHitCount),
		zap.Int64("total_duration_ms", response.TotalDuration.Milliseconds()),
		zap.String("strategy", strategy),
	)

	return response, nil
}

// processPatientGroupedBatch optimizes processing by grouping requests by patient
func (bp *EnhancedBatchProcessor) processPatientGroupedBatch(
	ctx context.Context,
	batch *BatchRequest,
) (*BatchResponse, error) {
	// Group requests by patient ID
	patientGroups := make(map[string][]*types.SafetyRequest)
	for _, req := range batch.Requests {
		patientGroups[req.PatientID] = append(patientGroups[req.PatientID], req)
	}

	bp.logger.Debug("Processing patient-grouped batch",
		zap.String("batch_id", batch.BatchID),
		zap.Int("patient_count", len(patientGroups)),
		zap.Int("total_requests", len(batch.Requests)),
	)

	// Process each patient group concurrently
	type patientResult struct {
		PatientID string
		Responses []*types.SafetyResponse
		Error     error
	}

	resultChan := make(chan patientResult, len(patientGroups))
	var wg sync.WaitGroup

	// Process patient groups
	for patientID, requests := range patientGroups {
		wg.Add(1)
		bp.workerPool <- func() {
			defer wg.Done()
			responses, err := bp.processPatientGroup(ctx, patientID, requests)
			resultChan <- patientResult{
				PatientID: patientID,
				Responses: responses,
				Error:     err,
			}
		}
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	var allResponses []*types.SafetyResponse
	var processingErrors []error

	for result := range resultChan {
		if result.Error != nil {
			processingErrors = append(processingErrors, result.Error)
			bp.logger.Warn("Patient group processing failed",
				zap.String("patient_id", result.PatientID),
				zap.Error(result.Error),
			)
		} else {
			allResponses = append(allResponses, result.Responses...)
		}
	}

	// Handle partial failures
	if len(processingErrors) > 0 && len(allResponses) == 0 {
		return nil, fmt.Errorf("all patient groups failed: %v", processingErrors[0])
	}

	response := bp.buildBatchResponse(batch, allResponses)
	response.Metadata["processing_strategy"] = "patient_grouped"
	response.Metadata["patient_groups"] = len(patientGroups)
	response.Metadata["partial_failures"] = len(processingErrors)

	return response, nil
}

// processSnapshotOptimizedBatch optimizes processing using snapshot caching
func (bp *EnhancedBatchProcessor) processSnapshotOptimizedBatch(
	ctx context.Context,
	batch *BatchRequest,
) (*BatchResponse, error) {
	// Pre-fetch all unique snapshots
	snapshotIDs := bp.extractUniqueSnapshots(batch.Requests)
	
	bp.logger.Debug("Processing snapshot-optimized batch",
		zap.String("batch_id", batch.BatchID),
		zap.Int("unique_snapshots", len(snapshotIDs)),
		zap.Int("total_requests", len(batch.Requests)),
	)

	// Pre-warm snapshot cache
	if err := bp.preWarmSnapshots(ctx, snapshotIDs); err != nil {
		bp.logger.Warn("Snapshot pre-warming failed", zap.Error(err))
		// Continue with regular processing
	}

	// Process requests with warmed cache
	return bp.processParallelBatch(ctx, batch)
}

// processParallelBatch processes requests in parallel without special grouping
func (bp *EnhancedBatchProcessor) processParallelBatch(
	ctx context.Context,
	batch *BatchRequest,
) (*BatchResponse, error) {
	bp.logger.Debug("Processing parallel batch",
		zap.String("batch_id", batch.BatchID),
		zap.Int("request_count", len(batch.Requests)),
		zap.Int("concurrency", bp.config.Concurrency),
	)

	responseChan := make(chan *types.SafetyResponse, len(batch.Requests))
	var wg sync.WaitGroup

	// Process requests concurrently
	for _, req := range batch.Requests {
		wg.Add(1)
		bp.workerPool <- func() {
			defer wg.Done()
			
			// Acquire semaphore for concurrency control
			bp.semaphore <- struct{}{}
			defer func() { <-bp.semaphore }()
			
			response, err := bp.orchestrator.ProcessSafetyRequest(ctx, req)
			if err != nil {
				// Create error response
				response = bp.createErrorResponse(req, err)
			}
			responseChan <- response
		}
	}

	// Wait for completion
	go func() {
		wg.Wait()
		close(responseChan)
	}()

	// Collect responses
	var responses []*types.SafetyResponse
	for response := range responseChan {
		responses = append(responses, response)
	}

	batchResponse := bp.buildBatchResponse(batch, responses)
	batchResponse.Metadata["processing_strategy"] = "parallel_direct"
	batchResponse.Metadata["concurrency_used"] = bp.config.Concurrency

	return batchResponse, nil
}

// processStandardBatch processes requests using standard sequential method
func (bp *EnhancedBatchProcessor) processStandardBatch(
	ctx context.Context,
	batch *BatchRequest,
) (*BatchResponse, error) {
	responses := make([]*types.SafetyResponse, 0, len(batch.Requests))

	for _, req := range batch.Requests {
		response, err := bp.orchestrator.ProcessSafetyRequest(ctx, req)
		if err != nil {
			response = bp.createErrorResponse(req, err)
		}
		responses = append(responses, response)
	}

	batchResponse := bp.buildBatchResponse(batch, responses)
	batchResponse.Metadata["processing_strategy"] = "standard_sequential"

	return batchResponse, nil
}

// Helper methods

func (bp *EnhancedBatchProcessor) validateBatch(batch *BatchRequest) error {
	if batch.BatchID == "" {
		return fmt.Errorf("batch ID is required")
	}

	if len(batch.Requests) == 0 {
		return fmt.Errorf("batch cannot be empty")
	}

	if len(batch.Requests) > bp.config.MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(batch.Requests), bp.config.MaxBatchSize)
	}

	return nil
}

func (bp *EnhancedBatchProcessor) determineBatchStrategy(batch *BatchRequest) string {
	// Decision logic for batch processing strategy
	uniquePatients := bp.countUniquePatients(batch.Requests)
	hasSnapshots := bp.hasSnapshotReferences(batch.Requests)
	
	if bp.config.PatientGrouping && uniquePatients < len(batch.Requests)*0.7 {
		return "patient_grouped"
	}
	
	if bp.config.SnapshotOptimized && hasSnapshots {
		return "snapshot_optimized"
	}
	
	if len(batch.Requests) >= bp.config.Concurrency {
		return "parallel_direct"
	}
	
	return "standard"
}

func (bp *EnhancedBatchProcessor) processPatientGroup(
	ctx context.Context,
	patientID string,
	requests []*types.SafetyRequest,
) ([]*types.SafetyResponse, error) {
	responses := make([]*types.SafetyResponse, 0, len(requests))
	
	for _, req := range requests {
		response, err := bp.orchestrator.ProcessSafetyRequest(ctx, req)
		if err != nil {
			response = bp.createErrorResponse(req, err)
		}
		responses = append(responses, response)
	}
	
	return responses, nil
}

func (bp *EnhancedBatchProcessor) extractUniqueSnapshots(requests []*types.SafetyRequest) []string {
	snapshotMap := make(map[string]bool)
	
	for _, req := range requests {
		if snapshotID, exists := req.Context["snapshot_id"]; exists && snapshotID != "" {
			snapshotMap[snapshotID] = true
		}
	}
	
	snapshots := make([]string, 0, len(snapshotMap))
	for id := range snapshotMap {
		snapshots = append(snapshots, id)
	}
	
	return snapshots
}

func (bp *EnhancedBatchProcessor) preWarmSnapshots(ctx context.Context, snapshotIDs []string) error {
	// Pre-warm snapshot cache by fetching all snapshots
	var wg sync.WaitGroup
	errors := make(chan error, len(snapshotIDs))
	
	for _, snapshotID := range snapshotIDs {
		wg.Add(1)
		go func(id string) {
			defer wg.Done()
			
			_, err := bp.orchestrator.contextClient.GetSnapshot(ctx, id)
			if err != nil {
				errors <- fmt.Errorf("failed to pre-warm snapshot %s: %w", id, err)
			}
		}(snapshotID)
	}
	
	wg.Wait()
	close(errors)
	
	// Log any errors but don't fail the batch
	for err := range errors {
		bp.logger.Warn("Snapshot pre-warming error", zap.Error(err))
	}
	
	return nil
}

func (bp *EnhancedBatchProcessor) buildBatchResponse(
	batch *BatchRequest,
	responses []*types.SafetyResponse,
) *BatchResponse {
	summary := bp.calculateBatchSummary(responses)
	
	return &BatchResponse{
		BatchID:     batch.BatchID,
		Responses:   responses,
		Summary:     summary,
		ProcessedAt: time.Now(),
		Metadata:    make(map[string]interface{}),
	}
}

func (bp *EnhancedBatchProcessor) calculateBatchSummary(responses []*types.SafetyResponse) *BatchSummary {
	summary := &BatchSummary{
		TotalRequests: len(responses),
		ProcessingStats: &ProcessingStatistics{
			ResourceUtilization: make(map[string]float64),
		},
	}
	
	var totalRiskScore float64
	cacheHits := 0
	
	for _, response := range responses {
		switch response.Status {
		case types.SafetyStatusSafe:
			summary.SuccessfulResults++
		case types.SafetyStatusWarning:
			summary.WarningResults++
		case types.SafetyStatusUnsafe:
			summary.UnsafeResults++
		case types.SafetyStatusError:
			summary.ErrorResults++
		}
		
		totalRiskScore += response.RiskScore
		
		// Check for cache hits
		if mode, exists := response.Metadata["processing_mode"]; exists && mode == "snapshot_based" {
			cacheHits++
		}
	}
	
	if len(responses) > 0 {
		summary.AverageRiskScore = totalRiskScore / float64(len(responses))
	}
	
	summary.CacheHitCount = cacheHits
	
	return summary
}

func (bp *EnhancedBatchProcessor) calculateProcessingStatistics(response *BatchResponse) *ProcessingStatistics {
	stats := &ProcessingStatistics{
		ResourceUtilization: make(map[string]float64),
	}
	
	// Calculate statistics based on response data
	totalEngineExecutions := 0
	var totalEngineLatency time.Duration
	
	for _, resp := range response.Responses {
		totalEngineExecutions += len(resp.EngineResults)
		for _, result := range resp.EngineResults {
			totalEngineLatency += result.Duration
		}
	}
	
	stats.EngineExecutions = totalEngineExecutions
	if totalEngineExecutions > 0 {
		stats.AverageEngineLatency = totalEngineLatency / time.Duration(totalEngineExecutions)
	}
	
	stats.CacheHits = response.Summary.CacheHitCount
	stats.CacheMisses = response.Summary.TotalRequests - response.Summary.CacheHitCount
	
	// Calculate parallelism efficiency
	if response.TotalDuration > 0 {
		idealParallelTime := stats.AverageEngineLatency
		if idealParallelTime > 0 {
			stats.ParallelismAchieved = float64(idealParallelTime) / float64(response.TotalDuration)
		}
	}
	
	return stats
}

func (bp *EnhancedBatchProcessor) createErrorResponse(req *types.SafetyRequest, err error) *types.SafetyResponse {
	return &types.SafetyResponse{
		RequestID:      req.RequestID,
		Status:         types.SafetyStatusError,
		RiskScore:      1.0,
		EngineResults:  []types.EngineResult{},
		ProcessingTime: 0,
		Timestamp:      time.Now(),
		Metadata: map[string]interface{}{
			"error": err.Error(),
		},
	}
}

func (bp *EnhancedBatchProcessor) countUniquePatients(requests []*types.SafetyRequest) int {
	patients := make(map[string]bool)
	for _, req := range requests {
		patients[req.PatientID] = true
	}
	return len(patients)
}

func (bp *EnhancedBatchProcessor) hasSnapshotReferences(requests []*types.SafetyRequest) bool {
	for _, req := range requests {
		if _, exists := req.Context["snapshot_id"]; exists {
			return true
		}
	}
	return false
}

func (bp *EnhancedBatchProcessor) updateBatchMetrics(batch *BatchRequest) {
	bp.metrics.mu.Lock()
	defer bp.metrics.mu.Unlock()
	
	bp.metrics.TotalBatches++
	bp.metrics.TotalRequests += int64(len(batch.Requests))
	
	// Update running average
	if bp.metrics.TotalBatches == 1 {
		bp.metrics.AverageBatchSize = float64(len(batch.Requests))
	} else {
		bp.metrics.AverageBatchSize = (bp.metrics.AverageBatchSize*float64(bp.metrics.TotalBatches-1) + 
			float64(len(batch.Requests))) / float64(bp.metrics.TotalBatches)
	}
}

func (bp *EnhancedBatchProcessor) worker() {
	for job := range bp.workerPool {
		job()
	}
}

// GetBatchResult retrieves a completed batch result
func (bp *EnhancedBatchProcessor) GetBatchResult(batchID string) (*BatchResponse, bool) {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	
	result, exists := bp.batchResults[batchID]
	return result, exists
}

// GetBatchStatus returns the current status of a batch
func (bp *EnhancedBatchProcessor) GetBatchStatus(batchID string) (string, bool) {
	bp.mu.RLock()
	defer bp.mu.RUnlock()
	
	if _, exists := bp.activeBatches[batchID]; exists {
		return "processing", true
	}
	
	if _, exists := bp.batchResults[batchID]; exists {
		return "completed", true
	}
	
	return "not_found", false
}

// GetMetrics returns batch processing metrics
func (bp *EnhancedBatchProcessor) GetMetrics() *BatchProcessingMetrics {
	bp.metrics.mu.RLock()
	defer bp.metrics.mu.RUnlock()
	
	// Return a copy to avoid race conditions
	return &BatchProcessingMetrics{
		TotalBatches:          bp.metrics.TotalBatches,
		TotalRequests:         bp.metrics.TotalRequests,
		AverageBatchSize:      bp.metrics.AverageBatchSize,
		AverageProcessingTime: bp.metrics.AverageProcessingTime,
		ThroughputPerSecond:   bp.metrics.ThroughputPerSecond,
		ErrorRate:             bp.metrics.ErrorRate,
		CacheHitRatio:         bp.metrics.CacheHitRatio,
		ParallelismEfficiency: bp.metrics.ParallelismEfficiency,
	}
}