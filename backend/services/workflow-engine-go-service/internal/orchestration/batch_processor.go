package orchestration

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.uber.org/zap"
)

// BatchProcessor handles grouped medication commits as described in document 13_9.2
type BatchProcessor struct {
	logger     *zap.Logger
	config     *BatchConfiguration
	activeBatches map[string]*BatchExecution
	mu         sync.RWMutex
}

// BatchConfiguration defines batch processing behavior
type BatchConfiguration struct {
	MaxBatchSize        int           `json:"max_batch_size"`        // Maximum proposals per batch
	BatchTimeout        time.Duration `json:"batch_timeout"`         // Maximum time to wait for batch completion
	ParallelProcessing  bool          `json:"parallel_processing"`   // Enable parallel processing within batch
	MaxConcurrency      int           `json:"max_concurrency"`       // Maximum concurrent operations
	FailureHandling     string        `json:"failure_handling"`      // "FAIL_FAST", "CONTINUE", "ROLLBACK_ALL"
	RetryPolicy         *RetryPolicy  `json:"retry_policy"`
}

// RetryPolicy defines retry behavior for batch operations
type RetryPolicy struct {
	MaxRetries    int           `json:"max_retries"`
	RetryDelay    time.Duration `json:"retry_delay"`
	BackoffFactor float64       `json:"backoff_factor"`
}

// BatchRequest represents a request to process multiple proposals as a batch
type BatchRequest struct {
	BatchID         string           `json:"batch_id"`
	BatchType       BatchType        `json:"batch_type"`
	Proposals       []ProposalItem   `json:"proposals"`
	ClinicalContext map[string]interface{} `json:"clinical_context"`
	RequestedBy     string           `json:"requested_by"`
	Priority        BatchPriority    `json:"priority"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

// ProposalItem represents a single proposal in a batch
type ProposalItem struct {
	ProposalID      string                 `json:"proposal_id"`
	WorkflowID      string                 `json:"workflow_id"`
	MedicationCode  string                 `json:"medication_code"`
	SafetyVerdict   SafetyVerdict          `json:"safety_verdict"`
	OverrideAction  *OverrideDecision      `json:"override_action,omitempty"`
	Context         map[string]interface{} `json:"context"`
	Dependencies    []string               `json:"dependencies,omitempty"`
}

// BatchResult represents the outcome of batch processing
type BatchResult struct {
	BatchID           string                    `json:"batch_id"`
	Status            BatchStatus               `json:"status"`
	ProcessedItems    []ProposalResult          `json:"processed_items"`
	Summary           *BatchSummary             `json:"summary"`
	ExecutionMetrics  *BatchExecutionMetrics    `json:"execution_metrics"`
	RollbackToken     string                    `json:"rollback_token,omitempty"`
	RollbackExpiresAt *time.Time                `json:"rollback_expires_at,omitempty"`
	AuditTrail        []AuditEntry              `json:"audit_trail"`
	ErrorDetails      []BatchError              `json:"error_details,omitempty"`
}

// ProposalResult represents the result of processing a single proposal in batch
type ProposalResult struct {
	ProposalID      string                 `json:"proposal_id"`
	Status          ProposalStatus         `json:"status"`
	CommitID        string                 `json:"commit_id,omitempty"`
	ErrorMessage    string                 `json:"error_message,omitempty"`
	ExecutionTime   time.Duration          `json:"execution_time"`
	RetryCount      int                    `json:"retry_count"`
	Context         map[string]interface{} `json:"context,omitempty"`
}

// BatchExecution tracks an active batch processing operation
type BatchExecution struct {
	BatchID         string                `json:"batch_id"`
	StartTime       time.Time             `json:"start_time"`
	Status          BatchStatus           `json:"status"`
	TotalItems      int                   `json:"total_items"`
	ProcessedItems  int                   `json:"processed_items"`
	SuccessfulItems int                   `json:"successful_items"`
	FailedItems     int                   `json:"failed_items"`
	ActiveWorkers   int                   `json:"active_workers"`
	Context         map[string]interface{} `json:"context"`
}

// Enums and supporting types
type BatchType string

const (
	BatchTypeAdmissionOrderSet    BatchType = "ADMISSION_ORDER_SET"
	BatchTypeDischargeOrders      BatchType = "DISCHARGE_ORDERS"
	BatchTypeRoutineMedications   BatchType = "ROUTINE_MEDICATIONS"
	BatchTypeEmergencyProtocol    BatchType = "EMERGENCY_PROTOCOL"
	BatchTypeCustomBatch          BatchType = "CUSTOM_BATCH"
)

type BatchPriority string

const (
	BatchPriorityLow      BatchPriority = "LOW"
	BatchPriorityNormal   BatchPriority = "NORMAL"
	BatchPriorityHigh     BatchPriority = "HIGH"
	BatchPriorityCritical BatchPriority = "CRITICAL"
)

type BatchStatus string

const (
	BatchStatusPending     BatchStatus = "PENDING"
	BatchStatusProcessing  BatchStatus = "PROCESSING"
	BatchStatusCompleted   BatchStatus = "COMPLETED"
	BatchStatusFailed      BatchStatus = "FAILED"
	BatchStatusPartial     BatchStatus = "PARTIAL"
	BatchStatusRolledBack  BatchStatus = "ROLLED_BACK"
)

type ProposalStatus string

const (
	ProposalStatusPending   ProposalStatus = "PENDING"
	ProposalStatusProcessing ProposalStatus = "PROCESSING"
	ProposalStatusSuccess   ProposalStatus = "SUCCESS"
	ProposalStatusFailed    ProposalStatus = "FAILED"
	ProposalStatusSkipped   ProposalStatus = "SKIPPED"
)

type BatchSummary struct {
	TotalProposals    int                    `json:"total_proposals"`
	SuccessfulCommits int                    `json:"successful_commits"`
	FailedCommits     int                    `json:"failed_commits"`
	SkippedProposals  int                    `json:"skipped_proposals"`
	OverriddenItems   int                    `json:"overridden_items"`
	ProcessingTime    time.Duration          `json:"processing_time"`
	AverageItemTime   time.Duration          `json:"average_item_time"`
	BatchEfficiency   float64                `json:"batch_efficiency"`
	CriticalFailures  []string               `json:"critical_failures,omitempty"`
}

type BatchExecutionMetrics struct {
	StartTime         time.Time             `json:"start_time"`
	EndTime           time.Time             `json:"end_time"`
	TotalDuration     time.Duration         `json:"total_duration"`
	ParallelismFactor float64               `json:"parallelism_factor"`
	ThroughputPerMin  float64               `json:"throughput_per_minute"`
	PhaseBreakdown    map[string]time.Duration `json:"phase_breakdown"`
	ResourceUsage     map[string]interface{} `json:"resource_usage"`
}

type BatchError struct {
	ErrorID     string                 `json:"error_id"`
	ProposalID  string                 `json:"proposal_id,omitempty"`
	ErrorType   string                 `json:"error_type"`
	Message     string                 `json:"message"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Recoverable bool                   `json:"recoverable"`
	Timestamp   time.Time              `json:"timestamp"`
}

// NewBatchProcessor creates a new batch processor
func NewBatchProcessor(logger *zap.Logger) *BatchProcessor {
	return &BatchProcessor{
		logger:        logger,
		config:        DefaultBatchConfiguration(),
		activeBatches: make(map[string]*BatchExecution),
	}
}

// DefaultBatchConfiguration returns default batch processing configuration
func DefaultBatchConfiguration() *BatchConfiguration {
	return &BatchConfiguration{
		MaxBatchSize:       50,                // Maximum 50 proposals per batch
		BatchTimeout:       10 * time.Minute, // 10-minute timeout for batch completion
		ParallelProcessing: true,              // Enable parallel processing
		MaxConcurrency:     5,                 // Process up to 5 proposals concurrently
		FailureHandling:    "CONTINUE",        // Continue processing despite individual failures
		RetryPolicy: &RetryPolicy{
			MaxRetries:    2,
			RetryDelay:    1 * time.Second,
			BackoffFactor: 2.0,
		},
	}
}

// ProcessBatch processes a batch of medication proposals
func (b *BatchProcessor) ProcessBatch(ctx context.Context, request *BatchRequest, commitOrchestrator *CommitOrchestrator) (*BatchResult, error) {
	startTime := time.Now()

	b.logger.Info("Starting batch processing",
		zap.String("batch_id", request.BatchID),
		zap.String("batch_type", string(request.BatchType)),
		zap.Int("proposal_count", len(request.Proposals)),
		zap.String("priority", string(request.Priority)))

	// Validate batch request
	if err := b.validateBatchRequest(request); err != nil {
		return b.buildFailureResult(request.BatchID, startTime, fmt.Sprintf("Validation failed: %v", err)), err
	}

	// Create batch execution tracking
	execution := &BatchExecution{
		BatchID:    request.BatchID,
		StartTime:  startTime,
		Status:     BatchStatusProcessing,
		TotalItems: len(request.Proposals),
		Context:    request.ClinicalContext,
	}

	b.trackBatchExecution(request.BatchID, execution)
	defer b.untrackBatchExecution(request.BatchID)

	// Create audit trail
	auditTrail := []AuditEntry{{
		EntryID:   fmt.Sprintf("%s_start", request.BatchID),
		Action:    "BATCH_PROCESSING_STARTED",
		Actor:     request.RequestedBy,
		Context:   map[string]interface{}{"batch_type": request.BatchType, "proposal_count": len(request.Proposals)},
		Timestamp: startTime,
	}}

	// Resolve dependencies and determine processing order
	processingOrder := b.resolveDependencies(request.Proposals)

	// Process proposals based on configuration
	var results []ProposalResult
	var processingErrors []BatchError

	if b.config.ParallelProcessing {
		results, processingErrors = b.processProposalsParallel(ctx, processingOrder, request, commitOrchestrator, execution)
	} else {
		results, processingErrors = b.processProposalsSequential(ctx, processingOrder, request, commitOrchestrator, execution)
	}

	// Build batch summary
	summary := b.buildBatchSummary(results, startTime)

	// Determine overall batch status
	batchStatus := b.determineBatchStatus(summary, processingErrors)

	// Create rollback token for the entire batch if successful
	var rollbackToken string
	var rollbackExpiry *time.Time
	if batchStatus == BatchStatusCompleted || batchStatus == BatchStatusPartial {
		// For batch operations, provide longer rollback window (10 minutes)
		token := fmt.Sprintf("batch_rollback_%s_%d", request.BatchID, time.Now().Unix())
		expiry := time.Now().Add(10 * time.Minute)
		rollbackToken = token
		rollbackExpiry = &expiry
	}

	// Final audit entry
	auditTrail = append(auditTrail, AuditEntry{
		EntryID: fmt.Sprintf("%s_complete", request.BatchID),
		Action:  "BATCH_PROCESSING_COMPLETED",
		Actor:   request.RequestedBy,
		Context: map[string]interface{}{
			"status":             batchStatus,
			"successful_commits": summary.SuccessfulCommits,
			"failed_commits":     summary.FailedCommits,
			"total_time":         summary.ProcessingTime.String(),
		},
		Timestamp: time.Now(),
	})

	// Build final result
	result := &BatchResult{
		BatchID:           request.BatchID,
		Status:            batchStatus,
		ProcessedItems:    results,
		Summary:           summary,
		RollbackToken:     rollbackToken,
		RollbackExpiresAt: rollbackExpiry,
		AuditTrail:        auditTrail,
		ErrorDetails:      processingErrors,
		ExecutionMetrics: &BatchExecutionMetrics{
			StartTime:         startTime,
			EndTime:           time.Now(),
			TotalDuration:     time.Since(startTime),
			ParallelismFactor: b.calculateParallelismFactor(execution),
			ThroughputPerMin:  b.calculateThroughput(summary),
			PhaseBreakdown: map[string]time.Duration{
				"dependency_resolution": 100 * time.Millisecond, // Placeholder
				"processing":            summary.ProcessingTime,
				"summary_generation":    50 * time.Millisecond, // Placeholder
			},
		},
	}

	b.logger.Info("Batch processing completed",
		zap.String("batch_id", request.BatchID),
		zap.String("status", string(batchStatus)),
		zap.Int("successful", summary.SuccessfulCommits),
		zap.Int("failed", summary.FailedCommits),
		zap.Duration("total_time", result.ExecutionMetrics.TotalDuration))

	return result, nil
}

// processProposalsParallel processes proposals concurrently
func (b *BatchProcessor) processProposalsParallel(
	ctx context.Context,
	proposals []ProposalItem,
	request *BatchRequest,
	commitOrchestrator *CommitOrchestrator,
	execution *BatchExecution,
) ([]ProposalResult, []BatchError) {

	results := make([]ProposalResult, len(proposals))
	errors := make([]BatchError, 0)

	// Create semaphore for concurrency control
	semaphore := make(chan struct{}, b.config.MaxConcurrency)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i, proposal := range proposals {
		wg.Add(1)
		go func(index int, item ProposalItem) {
			defer wg.Done()

			// Acquire semaphore
			semaphore <- struct{}{}
			defer func() { <-semaphore }()

			// Update active workers count
			b.mu.Lock()
			execution.ActiveWorkers++
			b.mu.Unlock()

			defer func() {
				b.mu.Lock()
				execution.ActiveWorkers--
				execution.ProcessedItems++
				b.mu.Unlock()
			}()

			result := b.processProposal(ctx, item, request, commitOrchestrator)

			mu.Lock()
			results[index] = result
			if result.Status == ProposalStatusSuccess {
				execution.SuccessfulItems++
			} else if result.Status == ProposalStatusFailed {
				execution.FailedItems++
				errors = append(errors, BatchError{
					ErrorID:     fmt.Sprintf("%s_%s", request.BatchID, item.ProposalID),
					ProposalID:  item.ProposalID,
					ErrorType:   "PROCESSING_ERROR",
					Message:     result.ErrorMessage,
					Recoverable: true,
					Timestamp:   time.Now(),
				})
			}
			mu.Unlock()
		}(i, proposal)
	}

	wg.Wait()
	return results, errors
}

// processProposalsSequential processes proposals one by one
func (b *BatchProcessor) processProposalsSequential(
	ctx context.Context,
	proposals []ProposalItem,
	request *BatchRequest,
	commitOrchestrator *CommitOrchestrator,
	execution *BatchExecution,
) ([]ProposalResult, []BatchError) {

	results := make([]ProposalResult, 0, len(proposals))
	errors := make([]BatchError, 0)

	for _, proposal := range proposals {
		execution.ActiveWorkers = 1

		result := b.processProposal(ctx, proposal, request, commitOrchestrator)
		results = append(results, result)

		execution.ProcessedItems++
		if result.Status == ProposalStatusSuccess {
			execution.SuccessfulItems++
		} else if result.Status == ProposalStatusFailed {
			execution.FailedItems++
			batchError := BatchError{
				ErrorID:     fmt.Sprintf("%s_%s", request.BatchID, proposal.ProposalID),
				ProposalID:  proposal.ProposalID,
				ErrorType:   "PROCESSING_ERROR",
				Message:     result.ErrorMessage,
				Recoverable: true,
				Timestamp:   time.Now(),
			}
			errors = append(errors, batchError)

			// Handle failure based on configuration
			if b.config.FailureHandling == "FAIL_FAST" {
				b.logger.Error("Failing fast due to proposal failure",
					zap.String("batch_id", request.BatchID),
					zap.String("proposal_id", proposal.ProposalID))
				break
			}
		}

		execution.ActiveWorkers = 0
	}

	return results, errors
}

// processProposal processes a single proposal within a batch context
func (b *BatchProcessor) processProposal(
	ctx context.Context,
	proposal ProposalItem,
	batchRequest *BatchRequest,
	commitOrchestrator *CommitOrchestrator,
) ProposalResult {
	startTime := time.Now()

	b.logger.Debug("Processing batch proposal",
		zap.String("batch_id", batchRequest.BatchID),
		zap.String("proposal_id", proposal.ProposalID))

	// Create commit request from proposal
	commitRequest := &CommitRequest{
		ProposalID:      proposal.ProposalID,
		WorkflowID:      proposal.WorkflowID,
		SafetyVerdict:   proposal.SafetyVerdict,
		ClinicalContext: proposal.Context,
		RequestedBy:     batchRequest.RequestedBy,
		BatchID:         batchRequest.BatchID,
	}

	// Process the proposal through commit orchestrator
	commitResult, err := commitOrchestrator.ExecuteCommitPhase(ctx, commitRequest)

	result := ProposalResult{
		ProposalID:    proposal.ProposalID,
		ExecutionTime: time.Since(startTime),
		Context:       proposal.Context,
	}

	if err != nil {
		result.Status = ProposalStatusFailed
		result.ErrorMessage = err.Error()
		b.logger.Error("Proposal processing failed in batch",
			zap.String("proposal_id", proposal.ProposalID),
			zap.Error(err))
		return result
	}

	// Handle different commit results
	switch commitResult.Status {
	case string(CommitStatusCommitted), string(CommitStatusOverridden):
		result.Status = ProposalStatusSuccess
		result.CommitID = commitResult.CommitID
	case string(CommitStatusAwaitingOverride):
		// Handle override for batch context
		if proposal.OverrideAction != nil {
			overrideResult, err := commitOrchestrator.HandleOverrideDecision(ctx, proposal.OverrideAction)
			if err != nil {
				result.Status = ProposalStatusFailed
				result.ErrorMessage = fmt.Sprintf("Override processing failed: %v", err)
			} else if overrideResult.Status == string(CommitStatusOverridden) {
				result.Status = ProposalStatusSuccess
				result.CommitID = overrideResult.CommitID
			} else {
				result.Status = ProposalStatusFailed
				result.ErrorMessage = "Override was not approved"
			}
		} else {
			result.Status = ProposalStatusSkipped
			result.ErrorMessage = "Override required but not provided in batch context"
		}
	case string(CommitStatusCancelled):
		result.Status = ProposalStatusSkipped
		result.ErrorMessage = "Proposal was cancelled"
	default:
		result.Status = ProposalStatusFailed
		result.ErrorMessage = fmt.Sprintf("Unexpected commit status: %s", commitResult.Status)
	}

	return result
}

// Helper methods

func (b *BatchProcessor) validateBatchRequest(request *BatchRequest) error {
	if request.BatchID == "" {
		return fmt.Errorf("batch ID is required")
	}

	if len(request.Proposals) == 0 {
		return fmt.Errorf("batch must contain at least one proposal")
	}

	if len(request.Proposals) > b.config.MaxBatchSize {
		return fmt.Errorf("batch size %d exceeds maximum %d", len(request.Proposals), b.config.MaxBatchSize)
	}

	if request.RequestedBy == "" {
		return fmt.Errorf("requested_by is required")
	}

	return nil
}

func (b *BatchProcessor) resolveDependencies(proposals []ProposalItem) []ProposalItem {
	// Simple dependency resolution - in practice this would be more sophisticated
	// For now, just return proposals in original order
	return proposals
}

func (b *BatchProcessor) buildBatchSummary(results []ProposalResult, startTime time.Time) *BatchSummary {
	summary := &BatchSummary{
		TotalProposals: len(results),
		ProcessingTime: time.Since(startTime),
	}

	totalTime := time.Duration(0)
	for _, result := range results {
		totalTime += result.ExecutionTime

		switch result.Status {
		case ProposalStatusSuccess:
			summary.SuccessfulCommits++
		case ProposalStatusFailed:
			summary.FailedCommits++
		case ProposalStatusSkipped:
			summary.SkippedProposals++
		}
	}

	if len(results) > 0 {
		summary.AverageItemTime = totalTime / time.Duration(len(results))
	}

	if summary.TotalProposals > 0 {
		summary.BatchEfficiency = float64(summary.SuccessfulCommits) / float64(summary.TotalProposals)
	}

	return summary
}

func (b *BatchProcessor) determineBatchStatus(summary *BatchSummary, errors []BatchError) BatchStatus {
	if summary.FailedCommits == 0 && summary.SkippedProposals == 0 {
		return BatchStatusCompleted
	}

	if summary.SuccessfulCommits == 0 {
		return BatchStatusFailed
	}

	return BatchStatusPartial
}

func (b *BatchProcessor) calculateParallelismFactor(execution *BatchExecution) float64 {
	if execution.TotalItems <= 1 {
		return 1.0
	}

	return float64(b.config.MaxConcurrency) / float64(execution.TotalItems)
}

func (b *BatchProcessor) calculateThroughput(summary *BatchSummary) float64 {
	if summary.ProcessingTime.Minutes() == 0 {
		return 0
	}

	return float64(summary.TotalProposals) / summary.ProcessingTime.Minutes()
}

func (b *BatchProcessor) trackBatchExecution(batchID string, execution *BatchExecution) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.activeBatches[batchID] = execution
}

func (b *BatchProcessor) untrackBatchExecution(batchID string) {
	b.mu.Lock()
	defer b.mu.Unlock()
	delete(b.activeBatches, batchID)
}

func (b *BatchProcessor) buildFailureResult(batchID string, startTime time.Time, errorMessage string) *BatchResult {
	return &BatchResult{
		BatchID: batchID,
		Status:  BatchStatusFailed,
		Summary: &BatchSummary{
			ProcessingTime: time.Since(startTime),
		},
		ErrorDetails: []BatchError{{
			ErrorID:     fmt.Sprintf("%s_validation", batchID),
			ErrorType:   "VALIDATION_ERROR",
			Message:     errorMessage,
			Recoverable: false,
			Timestamp:   time.Now(),
		}},
		ExecutionMetrics: &BatchExecutionMetrics{
			StartTime:     startTime,
			EndTime:       time.Now(),
			TotalDuration: time.Since(startTime),
		},
	}
}