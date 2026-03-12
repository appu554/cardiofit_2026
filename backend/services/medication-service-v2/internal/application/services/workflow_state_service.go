package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// WorkflowState represents the persistent state of a workflow execution
type WorkflowState struct {
	WorkflowID       uuid.UUID                      `json:"workflow_id" db:"workflow_id"`
	RequestID        string                         `json:"request_id" db:"request_id"`
	PatientID        string                         `json:"patient_id" db:"patient_id"`
	Status           WorkflowStatus                 `json:"status" db:"status"`
	CurrentPhase     WorkflowPhase                  `json:"current_phase" db:"current_phase"`
	PhaseResults     map[WorkflowPhase]*PhaseResult `json:"phase_results" db:"phase_results"`
	ExecutionContext *WorkflowExecutionState        `json:"execution_context" db:"execution_context"`
	Metadata         map[string]interface{}         `json:"metadata" db:"metadata"`
	CreatedAt        time.Time                      `json:"created_at" db:"created_at"`
	UpdatedAt        time.Time                      `json:"updated_at" db:"updated_at"`
	CompletedAt      *time.Time                     `json:"completed_at,omitempty" db:"completed_at"`
	ExpiresAt        time.Time                      `json:"expires_at" db:"expires_at"`
	Version          int                            `json:"version" db:"version"`
}

// WorkflowExecutionState represents detailed execution state
type WorkflowExecutionState struct {
	StartTime        time.Time                      `json:"start_time"`
	LastActivity     time.Time                      `json:"last_activity"`
	RetryCount       int                            `json:"retry_count"`
	ErrorHistory     []WorkflowError                `json:"error_history"`
	WarningHistory   []WorkflowWarning              `json:"warning_history"`
	AuditTrail       []WorkflowAuditEntry           `json:"audit_trail"`
	PerformanceData  *WorkflowPerformanceData       `json:"performance_data"`
	ResourceUsage    *ResourceUsageData             `json:"resource_usage"`
	Configuration    *WorkflowConfiguration         `json:"configuration"`
}

// WorkflowPerformanceData tracks performance metrics during execution
type WorkflowPerformanceData struct {
	PhaseTimings        map[WorkflowPhase]time.Duration `json:"phase_timings"`
	TotalExecutionTime  time.Duration                   `json:"total_execution_time"`
	QueueTime           time.Duration                   `json:"queue_time"`
	ProcessingTime      time.Duration                   `json:"processing_time"`
	IOTime              time.Duration                   `json:"io_time"`
	ThrottleTime        time.Duration                   `json:"throttle_time"`
	CacheHits           int                             `json:"cache_hits"`
	CacheMisses         int                             `json:"cache_misses"`
	APICallsTotal       int                             `json:"api_calls_total"`
	APICallsSuccessful  int                             `json:"api_calls_successful"`
	APICallsFailed      int                             `json:"api_calls_failed"`
}

// ResourceUsageData tracks resource usage during execution
type ResourceUsageData struct {
	MemoryUsagePeak     int64                          `json:"memory_usage_peak"`
	MemoryUsageAverage  int64                          `json:"memory_usage_average"`
	CPUUsagePeak        float64                        `json:"cpu_usage_peak"`
	CPUUsageAverage     float64                        `json:"cpu_usage_average"`
	NetworkBytesIn      int64                          `json:"network_bytes_in"`
	NetworkBytesOut     int64                          `json:"network_bytes_out"`
	DiskReadsTotal      int64                          `json:"disk_reads_total"`
	DiskWritesTotal     int64                          `json:"disk_writes_total"`
	ConnectionsActive   int                            `json:"connections_active"`
	ConnectionsTotal    int                            `json:"connections_total"`
}

// WorkflowConfiguration captures workflow execution configuration
type WorkflowConfiguration struct {
	TimeoutPerPhase      time.Duration          `json:"timeout_per_phase"`
	MaxRetries           int                    `json:"max_retries"`
	EnableParallelPhases bool                   `json:"enable_parallel_phases"`
	QualityThreshold     float64                `json:"quality_threshold"`
	EnableAuditTrail     bool                   `json:"enable_audit_trail"`
	RetryStrategy        string                 `json:"retry_strategy"`
	FailurePolicy        string                 `json:"failure_policy"`
	Parameters           map[string]interface{} `json:"parameters"`
}

// WorkflowStateQuery represents a query for workflow states
type WorkflowStateQuery struct {
	WorkflowIDs   []uuid.UUID        `json:"workflow_ids,omitempty"`
	PatientIDs    []string           `json:"patient_ids,omitempty"`
	Statuses      []WorkflowStatus   `json:"statuses,omitempty"`
	Phases        []WorkflowPhase    `json:"phases,omitempty"`
	CreatedAfter  *time.Time         `json:"created_after,omitempty"`
	CreatedBefore *time.Time         `json:"created_before,omitempty"`
	UpdatedAfter  *time.Time         `json:"updated_after,omitempty"`
	UpdatedBefore *time.Time         `json:"updated_before,omitempty"`
	IncludeExpired bool              `json:"include_expired"`
	Limit         int                `json:"limit"`
	Offset        int                `json:"offset"`
	OrderBy       string             `json:"order_by"`
	OrderDesc     bool               `json:"order_desc"`
}

// WorkflowStateUpdate represents an update to workflow state
type WorkflowStateUpdate struct {
	WorkflowID       uuid.UUID                      `json:"workflow_id"`
	Status           *WorkflowStatus                `json:"status,omitempty"`
	CurrentPhase     *WorkflowPhase                 `json:"current_phase,omitempty"`
	PhaseResults     map[WorkflowPhase]*PhaseResult `json:"phase_results,omitempty"`
	ExecutionContext *WorkflowExecutionState        `json:"execution_context,omitempty"`
	Metadata         map[string]interface{}         `json:"metadata,omitempty"`
	CompletedAt      *time.Time                     `json:"completed_at,omitempty"`
	Version          int                            `json:"version"`
}

// WorkflowStateRepository defines the interface for workflow state persistence
type WorkflowStateRepository interface {
	Create(ctx context.Context, state *WorkflowState) error
	GetByID(ctx context.Context, workflowID uuid.UUID) (*WorkflowState, error)
	Update(ctx context.Context, update *WorkflowStateUpdate) error
	Delete(ctx context.Context, workflowID uuid.UUID) error
	Query(ctx context.Context, query *WorkflowStateQuery) ([]*WorkflowState, int, error)
	CleanupExpired(ctx context.Context, before time.Time) (int, error)
	GetStatistics(ctx context.Context) (*WorkflowStateStatistics, error)
}

// WorkflowStateStatistics provides statistics about workflow states
type WorkflowStateStatistics struct {
	TotalWorkflows      int64                          `json:"total_workflows"`
	ActiveWorkflows     int64                          `json:"active_workflows"`
	CompletedWorkflows  int64                          `json:"completed_workflows"`
	FailedWorkflows     int64                          `json:"failed_workflows"`
	StatusDistribution  map[WorkflowStatus]int64       `json:"status_distribution"`
	PhaseDistribution   map[WorkflowPhase]int64        `json:"phase_distribution"`
	AverageExecutionTime time.Duration                 `json:"average_execution_time"`
	SuccessRate         float64                        `json:"success_rate"`
	RetryRate           float64                        `json:"retry_rate"`
	LastUpdated         time.Time                      `json:"last_updated"`
}

// WorkflowStateServiceConfig contains configuration for the workflow state service
type WorkflowStateServiceConfig struct {
	DefaultTTL           time.Duration `mapstructure:"default_ttl" default:"24h"`
	CleanupInterval      time.Duration `mapstructure:"cleanup_interval" default:"1h"`
	MaxRetainedStates    int           `mapstructure:"max_retained_states" default:"10000"`
	EnableCompression    bool          `mapstructure:"enable_compression" default:"true"`
	EnableEncryption     bool          `mapstructure:"enable_encryption" default:"true"`
	EncryptionKey        string        `mapstructure:"encryption_key"`
	StatsCacheInterval   time.Duration `mapstructure:"stats_cache_interval" default:"5m"`
	EnableAuditLogging   bool          `mapstructure:"enable_audit_logging" default:"true"`
	BackupInterval       time.Duration `mapstructure:"backup_interval" default:"6h"`
	BackupRetention      time.Duration `mapstructure:"backup_retention" default:"30d"`
}

// WorkflowStateService manages workflow state persistence and lifecycle
type WorkflowStateService struct {
	repository     WorkflowStateRepository
	cacheService   *CacheService
	auditService   *AuditService
	metricsService *MetricsService
	
	config         WorkflowStateServiceConfig
	logger         *zap.Logger
	
	// Internal state
	statisticsCache *WorkflowStateStatistics
	lastStatsUpdate time.Time
}

// NewWorkflowStateService creates a new workflow state service
func NewWorkflowStateService(
	repository WorkflowStateRepository,
	cacheService *CacheService,
	auditService *AuditService,
	metricsService *MetricsService,
	config WorkflowStateServiceConfig,
	logger *zap.Logger,
) *WorkflowStateService {
	service := &WorkflowStateService{
		repository:     repository,
		cacheService:   cacheService,
		auditService:   auditService,
		metricsService: metricsService,
		config:         config,
		logger:         logger,
	}
	
	// Start background tasks
	go service.startCleanupRoutine()
	go service.startStatsRefreshRoutine()
	
	return service
}

// CreateState creates a new workflow state
func (w *WorkflowStateService) CreateState(ctx context.Context, workflowID uuid.UUID, request *WorkflowExecutionRequest) (*WorkflowState, error) {
	state := &WorkflowState{
		WorkflowID:   workflowID,
		RequestID:    request.RequestID,
		PatientID:    request.PatientID,
		Status:       WorkflowStatusPending,
		CurrentPhase: PhaseRecipeResolution,
		PhaseResults: make(map[WorkflowPhase]*PhaseResult),
		ExecutionContext: &WorkflowExecutionState{
			StartTime:        time.Now(),
			LastActivity:     time.Now(),
			RetryCount:       0,
			ErrorHistory:     []WorkflowError{},
			WarningHistory:   []WorkflowWarning{},
			AuditTrail:       []WorkflowAuditEntry{},
			PerformanceData:  &WorkflowPerformanceData{PhaseTimings: make(map[WorkflowPhase]time.Duration)},
			ResourceUsage:    &ResourceUsageData{},
			Configuration:    w.buildWorkflowConfiguration(request),
		},
		Metadata:  make(map[string]interface{}),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(w.config.DefaultTTL),
		Version:   1,
	}
	
	// Add request metadata
	state.Metadata["requested_by"] = request.RequestedBy
	state.Metadata["recipe_id"] = request.RecipeID
	
	if request.Options != nil {
		state.Metadata["enable_parallel_phases"] = request.Options.EnableParallelPhases
		state.Metadata["fail_fast"] = request.Options.FailFast
	}
	
	// Create in repository
	if err := w.repository.Create(ctx, state); err != nil {
		w.logger.Error("Failed to create workflow state",
			zap.String("workflow_id", workflowID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to create workflow state: %w", err)
	}
	
	// Cache the state
	if w.cacheService != nil {
		cacheKey := fmt.Sprintf("workflow_state:%s", workflowID.String())
		if cacheErr := w.cacheService.Set(ctx, cacheKey, state, w.config.DefaultTTL); cacheErr != nil {
			w.logger.Warn("Failed to cache workflow state", zap.Error(cacheErr))
		}
	}
	
	// Audit state creation
	if w.config.EnableAuditLogging && w.auditService != nil {
		w.auditEvent(ctx, "workflow_state_created", request.RequestedBy, map[string]interface{}{
			"workflow_id": workflowID.String(),
			"patient_id":  request.PatientID,
			"recipe_id":   request.RecipeID,
		})
	}
	
	// Update metrics
	if w.metricsService != nil {
		w.metricsService.RecordWorkflowStateCreated(state.Status)
	}
	
	w.logger.Info("Created workflow state",
		zap.String("workflow_id", workflowID.String()),
		zap.String("patient_id", request.PatientID),
	)
	
	return state, nil
}

// GetState retrieves a workflow state by ID
func (w *WorkflowStateService) GetState(ctx context.Context, workflowID uuid.UUID) (*WorkflowState, error) {
	// Try cache first
	if w.cacheService != nil {
		cacheKey := fmt.Sprintf("workflow_state:%s", workflowID.String())
		var cachedState WorkflowState
		if cacheErr := w.cacheService.Get(ctx, cacheKey, &cachedState); cacheErr == nil {
			w.logger.Debug("Retrieved workflow state from cache",
				zap.String("workflow_id", workflowID.String()),
			)
			return &cachedState, nil
		}
	}
	
	// Get from repository
	state, err := w.repository.GetByID(ctx, workflowID)
	if err != nil {
		w.logger.Error("Failed to get workflow state",
			zap.String("workflow_id", workflowID.String()),
			zap.Error(err),
		)
		return nil, fmt.Errorf("failed to get workflow state: %w", err)
	}
	
	if state == nil {
		return nil, fmt.Errorf("workflow state not found: %s", workflowID.String())
	}
	
	// Cache the state
	if w.cacheService != nil {
		cacheKey := fmt.Sprintf("workflow_state:%s", workflowID.String())
		if cacheErr := w.cacheService.Set(ctx, cacheKey, state, time.Until(state.ExpiresAt)); cacheErr != nil {
			w.logger.Warn("Failed to cache workflow state", zap.Error(cacheErr))
		}
	}
	
	w.logger.Debug("Retrieved workflow state from repository",
		zap.String("workflow_id", workflowID.String()),
		zap.String("status", fmt.Sprintf("%d", state.Status)),
		zap.Int("phase", int(state.CurrentPhase)),
	)
	
	return state, nil
}

// UpdateState updates an existing workflow state
func (w *WorkflowStateService) UpdateState(ctx context.Context, update *WorkflowStateUpdate) error {
	// Validate update
	if update.WorkflowID == uuid.Nil {
		return fmt.Errorf("workflow ID is required")
	}
	
	// Update in repository
	if err := w.repository.Update(ctx, update); err != nil {
		w.logger.Error("Failed to update workflow state",
			zap.String("workflow_id", update.WorkflowID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to update workflow state: %w", err)
	}
	
	// Invalidate cache
	if w.cacheService != nil {
		cacheKey := fmt.Sprintf("workflow_state:%s", update.WorkflowID.String())
		if cacheErr := w.cacheService.Delete(ctx, cacheKey); cacheErr != nil {
			w.logger.Warn("Failed to invalidate workflow state cache", zap.Error(cacheErr))
		}
	}
	
	// Audit state update
	if w.config.EnableAuditLogging && w.auditService != nil {
		w.auditEvent(ctx, "workflow_state_updated", "system", map[string]interface{}{
			"workflow_id": update.WorkflowID.String(),
			"version":     update.Version,
		})
	}
	
	// Update metrics
	if w.metricsService != nil && update.Status != nil {
		w.metricsService.RecordWorkflowStateUpdated(*update.Status)
	}
	
	w.logger.Debug("Updated workflow state",
		zap.String("workflow_id", update.WorkflowID.String()),
		zap.Int("version", update.Version),
	)
	
	return nil
}

// PersistState persists a workflow state (create or update)
func (w *WorkflowStateService) PersistState(ctx context.Context, state *WorkflowState) error {
	// Check if state exists
	existingState, err := w.repository.GetByID(ctx, state.WorkflowID)
	if err != nil && err.Error() != "workflow state not found" {
		return fmt.Errorf("failed to check existing state: %w", err)
	}
	
	// Update timestamps
	state.UpdatedAt = time.Now()
	if state.ExecutionContext != nil {
		state.ExecutionContext.LastActivity = time.Now()
	}
	
	if existingState == nil {
		// Create new state
		return w.repository.Create(ctx, state)
	} else {
		// Update existing state
		update := &WorkflowStateUpdate{
			WorkflowID:       state.WorkflowID,
			Status:           &state.Status,
			CurrentPhase:     &state.CurrentPhase,
			PhaseResults:     state.PhaseResults,
			ExecutionContext: state.ExecutionContext,
			Metadata:         state.Metadata,
			CompletedAt:      state.CompletedAt,
			Version:          existingState.Version + 1,
		}
		return w.UpdateState(ctx, update)
	}
}

// DeleteState deletes a workflow state
func (w *WorkflowStateService) DeleteState(ctx context.Context, workflowID uuid.UUID) error {
	// Delete from repository
	if err := w.repository.Delete(ctx, workflowID); err != nil {
		w.logger.Error("Failed to delete workflow state",
			zap.String("workflow_id", workflowID.String()),
			zap.Error(err),
		)
		return fmt.Errorf("failed to delete workflow state: %w", err)
	}
	
	// Remove from cache
	if w.cacheService != nil {
		cacheKey := fmt.Sprintf("workflow_state:%s", workflowID.String())
		if cacheErr := w.cacheService.Delete(ctx, cacheKey); cacheErr != nil {
			w.logger.Warn("Failed to remove workflow state from cache", zap.Error(cacheErr))
		}
	}
	
	// Audit state deletion
	if w.config.EnableAuditLogging && w.auditService != nil {
		w.auditEvent(ctx, "workflow_state_deleted", "system", map[string]interface{}{
			"workflow_id": workflowID.String(),
		})
	}
	
	w.logger.Info("Deleted workflow state",
		zap.String("workflow_id", workflowID.String()),
	)
	
	return nil
}

// QueryStates queries workflow states based on criteria
func (w *WorkflowStateService) QueryStates(ctx context.Context, query *WorkflowStateQuery) ([]*WorkflowState, int, error) {
	states, total, err := w.repository.Query(ctx, query)
	if err != nil {
		w.logger.Error("Failed to query workflow states", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to query workflow states: %w", err)
	}
	
	w.logger.Debug("Queried workflow states",
		zap.Int("count", len(states)),
		zap.Int("total", total),
	)
	
	return states, total, nil
}

// GetActiveStates returns all active workflow states
func (w *WorkflowStateService) GetActiveStates(ctx context.Context) ([]*WorkflowState, error) {
	query := &WorkflowStateQuery{
		Statuses: []WorkflowStatus{
			WorkflowStatusPending,
			WorkflowStatusInProgress,
		},
		IncludeExpired: false,
		Limit:          1000, // Reasonable limit for active states
		OrderBy:        "created_at",
		OrderDesc:      false,
	}
	
	states, _, err := w.QueryStates(ctx, query)
	return states, err
}

// GetStatistics returns workflow state statistics
func (w *WorkflowStateService) GetStatistics(ctx context.Context) (*WorkflowStateStatistics, error) {
	// Check cached statistics
	if w.statisticsCache != nil && time.Since(w.lastStatsUpdate) < w.config.StatsCacheInterval {
		return w.statisticsCache, nil
	}
	
	// Get fresh statistics
	stats, err := w.repository.GetStatistics(ctx)
	if err != nil {
		w.logger.Error("Failed to get workflow state statistics", zap.Error(err))
		return nil, fmt.Errorf("failed to get statistics: %w", err)
	}
	
	// Cache statistics
	w.statisticsCache = stats
	w.lastStatsUpdate = time.Now()
	
	return stats, nil
}

// CleanupExpiredStates removes expired workflow states
func (w *WorkflowStateService) CleanupExpiredStates(ctx context.Context) (int, error) {
	before := time.Now().Add(-w.config.DefaultTTL)
	
	deletedCount, err := w.repository.CleanupExpired(ctx, before)
	if err != nil {
		w.logger.Error("Failed to cleanup expired workflow states", zap.Error(err))
		return 0, fmt.Errorf("failed to cleanup expired states: %w", err)
	}
	
	if deletedCount > 0 {
		w.logger.Info("Cleaned up expired workflow states",
			zap.Int("deleted_count", deletedCount),
			zap.Time("before", before),
		)
		
		// Update metrics
		if w.metricsService != nil {
			w.metricsService.RecordWorkflowStatesCleanup(deletedCount)
		}
	}
	
	return deletedCount, nil
}

// GetWorkflowProgress returns the progress of a workflow
func (w *WorkflowStateService) GetWorkflowProgress(ctx context.Context, workflowID uuid.UUID) (*WorkflowProgress, error) {
	state, err := w.GetState(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	
	progress := &WorkflowProgress{
		WorkflowID:          state.WorkflowID,
		Status:              state.Status,
		CurrentPhase:        state.CurrentPhase,
		CompletedPhases:     w.countCompletedPhases(state.PhaseResults),
		TotalPhases:         4, // Total phases in medication workflow
		ProgressPercentage:  w.calculateProgressPercentage(state),
		EstimatedCompletion: w.estimateCompletion(state),
		LastUpdate:          state.UpdatedAt,
	}
	
	if state.ExecutionContext != nil {
		progress.ElapsedTime = time.Since(state.ExecutionContext.StartTime)
		if len(state.ExecutionContext.ErrorHistory) > 0 {
			progress.HasErrors = true
			progress.ErrorCount = len(state.ExecutionContext.ErrorHistory)
		}
		if len(state.ExecutionContext.WarningHistory) > 0 {
			progress.HasWarnings = true
			progress.WarningCount = len(state.ExecutionContext.WarningHistory)
		}
	}
	
	return progress, nil
}

// WorkflowProgress represents the progress of a workflow
type WorkflowProgress struct {
	WorkflowID          uuid.UUID        `json:"workflow_id"`
	Status              WorkflowStatus   `json:"status"`
	CurrentPhase        WorkflowPhase    `json:"current_phase"`
	CompletedPhases     int              `json:"completed_phases"`
	TotalPhases         int              `json:"total_phases"`
	ProgressPercentage  float64          `json:"progress_percentage"`
	ElapsedTime         time.Duration    `json:"elapsed_time"`
	EstimatedCompletion *time.Time       `json:"estimated_completion,omitempty"`
	LastUpdate          time.Time        `json:"last_update"`
	HasErrors           bool             `json:"has_errors"`
	HasWarnings         bool             `json:"has_warnings"`
	ErrorCount          int              `json:"error_count"`
	WarningCount        int              `json:"warning_count"`
}

// Helper methods

func (w *WorkflowStateService) buildWorkflowConfiguration(request *WorkflowExecutionRequest) *WorkflowConfiguration {
	config := &WorkflowConfiguration{
		TimeoutPerPhase:      30 * time.Second, // Default
		MaxRetries:           3,
		EnableParallelPhases: false,
		QualityThreshold:     0.8,
		EnableAuditTrail:     true,
		RetryStrategy:        "exponential_backoff",
		FailurePolicy:        "continue",
		Parameters:           make(map[string]interface{}),
	}
	
	if request.Options != nil {
		config.TimeoutPerPhase = request.Options.TimeoutPerPhase
		config.MaxRetries = request.Options.MaxRetries
		config.EnableParallelPhases = request.Options.EnableParallelPhases
		config.EnableAuditTrail = request.Options.EnableAuditTrail
		
		if request.Options.FailFast {
			config.FailurePolicy = "fail_fast"
		}
	}
	
	return config
}

func (w *WorkflowStateService) countCompletedPhases(phaseResults map[WorkflowPhase]*PhaseResult) int {
	count := 0
	for _, result := range phaseResults {
		if result.Status == StatusCompleted {
			count++
		}
	}
	return count
}

func (w *WorkflowStateService) calculateProgressPercentage(state *WorkflowState) float64 {
	completedPhases := float64(w.countCompletedPhases(state.PhaseResults))
	totalPhases := 4.0 // Total phases in medication workflow
	
	// Add partial progress for current phase
	if state.Status == WorkflowStatusInProgress {
		completedPhases += 0.5 // Assume 50% progress for current phase
	}
	
	return (completedPhases / totalPhases) * 100.0
}

func (w *WorkflowStateService) estimateCompletion(state *WorkflowState) *time.Time {
	if state.Status == WorkflowStatusCompleted || state.Status == WorkflowStatusFailed {
		return nil
	}
	
	if state.ExecutionContext == nil {
		return nil
	}
	
	// Simple estimation based on elapsed time and progress
	elapsedTime := time.Since(state.ExecutionContext.StartTime)
	progressPercentage := w.calculateProgressPercentage(state)
	
	if progressPercentage <= 0 {
		return nil
	}
	
	totalEstimatedTime := time.Duration(float64(elapsedTime) * (100.0 / progressPercentage))
	estimatedCompletion := state.ExecutionContext.StartTime.Add(totalEstimatedTime)
	
	return &estimatedCompletion
}

func (w *WorkflowStateService) startCleanupRoutine() {
	ticker := time.NewTicker(w.config.CleanupInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if _, err := w.CleanupExpiredStates(ctx); err != nil {
				w.logger.Error("Scheduled cleanup failed", zap.Error(err))
			}
			cancel()
		}
	}
}

func (w *WorkflowStateService) startStatsRefreshRoutine() {
	ticker := time.NewTicker(w.config.StatsCacheInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			if _, err := w.GetStatistics(ctx); err != nil {
				w.logger.Error("Scheduled stats refresh failed", zap.Error(err))
			}
			cancel()
		}
	}
}

func (w *WorkflowStateService) auditEvent(ctx context.Context, eventType, actor string, data interface{}) {
	if w.auditService == nil {
		return
	}
	
	auditData, _ := json.Marshal(data)
	w.auditService.LogEvent(ctx, &AuditEvent{
		EventType: eventType,
		ActorID:   actor,
		Data:      string(auditData),
		Timestamp: time.Now(),
	})
}

// IsHealthy returns the health status of the service
func (w *WorkflowStateService) IsHealthy(ctx context.Context) bool {
	// Test repository connection
	_, err := w.repository.GetStatistics(ctx)
	return err == nil
}