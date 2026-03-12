package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
	"go.uber.org/zap"

	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/infrastructure/clients"
)

// RecipeResolverContextIntegration extends the Recipe Resolver with Context Gateway integration
// This implements the complete Phase 1 → Phase 2 workflow:
// Phase 1: Recipe Resolution → Phase 2: Context Assembly via Snapshot (TRANSFORMED)
type RecipeResolverContextIntegration struct {
	// Core services
	resolverIntegration *RecipeResolverIntegration
	contextGateway      *ContextGatewayService
	
	// Configuration
	config              ContextIntegrationConfig
	logger              *zap.Logger
	
	// Metrics
	metrics             *ContextIntegrationMetrics
}

// ContextIntegrationConfig contains configuration for the integrated workflow
type ContextIntegrationConfig struct {
	// Workflow settings
	EnableSnapshotCreation      bool                     `json:"enable_snapshot_creation"`
	AutoCreateSnapshots         bool                     `json:"auto_create_snapshots"`
	SnapshotCreationMode        string                   `json:"snapshot_creation_mode"` // sync, async, conditional
	
	// Performance settings
	MaxConcurrentSnapshots      int                      `json:"max_concurrent_snapshots"`
	SnapshotCreationTimeout     time.Duration            `json:"snapshot_creation_timeout"`
	
	// Quality gates
	MinResolutionQuality        float64                  `json:"min_resolution_quality"`
	RequireValidatedSnapshots   bool                     `json:"require_validated_snapshots"`
	
	// Failure handling
	ContinueOnSnapshotFailure   bool                     `json:"continue_on_snapshot_failure"`
	RetryFailedSnapshots        bool                     `json:"retry_failed_snapshots"`
	
	// Snapshot lifecycle
	EnableSnapshotSupersession  bool                     `json:"enable_snapshot_supersession"`
	CleanupSupersededSnapshots  bool                     `json:"cleanup_superseded_snapshots"`
}

// ContextIntegrationMetrics tracks the integrated workflow performance
type ContextIntegrationMetrics struct {
	// Workflow metrics
	TotalWorkflows              int64         `json:"total_workflows"`
	SuccessfulWorkflows         int64         `json:"successful_workflows"`
	FailedWorkflows             int64         `json:"failed_workflows"`
	
	// Resolution metrics
	ResolutionTime              time.Duration `json:"average_resolution_time"`
	ResolutionSuccessRate       float64       `json:"resolution_success_rate"`
	
	// Snapshot metrics
	SnapshotsCreated            int64         `json:"snapshots_created"`
	SnapshotCreationTime        time.Duration `json:"average_snapshot_creation_time"`
	SnapshotSuccessRate         float64       `json:"snapshot_success_rate"`
	SnapshotValidationRate      float64       `json:"snapshot_validation_rate"`
	
	// End-to-end metrics
	EndToEndLatency             time.Duration `json:"average_end_to_end_latency"`
	QualityScore               float64       `json:"average_quality_score"`
	
	// Performance buckets
	Under50ms                  int64         `json:"under_50ms"`
	Between50And100ms          int64         `json:"between_50_and_100ms"`
	Between100And250ms         int64         `json:"between_100_and_250ms"`
	Over250ms                  int64         `json:"over_250ms"`
	
	LastUpdated                time.Time     `json:"last_updated"`
}

// IntegratedWorkflowRequest contains the complete workflow request
type IntegratedWorkflowRequest struct {
	// Recipe resolution request
	RecipeResolutionRequest     entities.RecipeResolutionRequest `json:"recipe_resolution_request"`
	
	// Snapshot creation options
	CreateSnapshot              bool                             `json:"create_snapshot"`
	SnapshotType                string                           `json:"snapshot_type"`
	SnapshotPriority           string                           `json:"snapshot_priority"`
	CustomFreshnessReqs        map[string]time.Duration         `json:"custom_freshness_requirements,omitempty"`
	RequireValidation          bool                             `json:"require_validation"`
	
	// Workflow metadata
	WorkflowID                 uuid.UUID                        `json:"workflow_id"`
	RequestedBy                string                           `json:"requested_by"`
	ClientContext              map[string]string                `json:"client_context,omitempty"`
}

// IntegratedWorkflowResponse contains the complete workflow response
type IntegratedWorkflowResponse struct {
	// Workflow metadata
	WorkflowID                 uuid.UUID                        `json:"workflow_id"`
	Status                     string                           `json:"status"`
	ProcessingTime             time.Duration                    `json:"processing_time"`
	
	// Phase 1: Recipe Resolution
	RecipeResolution           *entities.RecipeResolution       `json:"recipe_resolution"`
	ResolutionTime             time.Duration                    `json:"resolution_time"`
	ResolutionQuality          float64                          `json:"resolution_quality"`
	
	// Phase 2: Context Snapshot
	SnapshotResult             *SnapshotCreationResult          `json:"snapshot_result,omitempty"`
	SnapshotCreationTime       time.Duration                    `json:"snapshot_creation_time"`
	
	// Overall results
	QualityScore               float64                          `json:"overall_quality_score"`
	Warnings                   []string                         `json:"warnings,omitempty"`
	Errors                     []string                         `json:"errors,omitempty"`
	
	// Audit information
	CreatedAt                  time.Time                        `json:"created_at"`
	CompletedAt                time.Time                        `json:"completed_at"`
	ProcessedBy                string                           `json:"processed_by"`
}

// NewRecipeResolverContextIntegration creates a new integrated workflow service
func NewRecipeResolverContextIntegration(
	resolverIntegration *RecipeResolverIntegration,
	contextGateway *ContextGatewayService,
	logger *zap.Logger,
	config ContextIntegrationConfig,
) *RecipeResolverContextIntegration {
	return &RecipeResolverContextIntegration{
		resolverIntegration: resolverIntegration,
		contextGateway:      contextGateway,
		config:              config,
		logger:              logger,
		metrics:             &ContextIntegrationMetrics{LastUpdated: time.Now()},
	}
}

// ExecuteIntegratedWorkflow executes the complete Phase 1 → Phase 2 workflow
func (r *RecipeResolverContextIntegration) ExecuteIntegratedWorkflow(
	ctx context.Context,
	request *IntegratedWorkflowRequest,
) (*IntegratedWorkflowResponse, error) {
	startTime := time.Now()
	
	// Initialize response
	response := &IntegratedWorkflowResponse{
		WorkflowID:    request.WorkflowID,
		Status:        "processing",
		CreatedAt:     startTime,
		ProcessedBy:   "medication-service-v2",
		Warnings:      make([]string, 0),
		Errors:        make([]string, 0),
	}
	
	// Validate request
	if err := r.validateWorkflowRequest(request); err != nil {
		return r.handleWorkflowError(response, err, "request validation failed")
	}

	r.logger.Info("Starting integrated workflow",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.String("recipe_id", request.RecipeResolutionRequest.RecipeID),
		zap.String("patient_id", request.RecipeResolutionRequest.PatientContext.PatientID),
		zap.Bool("create_snapshot", request.CreateSnapshot),
	)

	// Phase 1: Recipe Resolution
	phase1Start := time.Now()
	resolution, err := r.resolverIntegration.ResolveRecipeWithIntegration(
		ctx,
		request.RecipeResolutionRequest,
	)
	if err != nil {
		return r.handleWorkflowError(response, err, "recipe resolution failed")
	}
	
	response.RecipeResolution = resolution
	response.ResolutionTime = time.Since(phase1Start)
	response.ResolutionQuality = r.calculateResolutionQuality(resolution)
	
	r.logger.Debug("Phase 1 completed",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.Duration("resolution_time", response.ResolutionTime),
		zap.Float64("quality", response.ResolutionQuality),
	)

	// Check quality gate
	if response.ResolutionQuality < r.config.MinResolutionQuality {
		warning := fmt.Sprintf("Resolution quality %.2f below threshold %.2f",
			response.ResolutionQuality, r.config.MinResolutionQuality)
		response.Warnings = append(response.Warnings, warning)
		
		r.logger.Warn("Resolution quality below threshold",
			zap.String("workflow_id", request.WorkflowID.String()),
			zap.Float64("quality", response.ResolutionQuality),
			zap.Float64("threshold", r.config.MinResolutionQuality),
		)
	}

	// Phase 2: Context Snapshot Creation (if requested and enabled)
	if request.CreateSnapshot && r.config.EnableSnapshotCreation {
		phase2Start := time.Now()
		
		snapshotRequest := &SnapshotCreationRequest{
			PatientID:         uuid.MustParse(request.RecipeResolutionRequest.PatientContext.PatientID),
			RecipeID:          uuid.MustParse(request.RecipeResolutionRequest.RecipeID),
			RecipeResolution:  resolution,
			PatientContext:    request.RecipeResolutionRequest.PatientContext,
			SnapshotType:      request.SnapshotType,
			Priority:          request.SnapshotPriority,
			CreatedBy:         request.RequestedBy,
			RequireValidation: request.RequireValidation || r.config.RequireValidatedSnapshots,
			CustomFreshness:   request.CustomFreshnessReqs,
		}
		
		snapshotResult, err := r.contextGateway.CreateSnapshotFromResolution(ctx, snapshotRequest)
		if err != nil {
			if r.config.ContinueOnSnapshotFailure {
				// Log error but continue
				errorMsg := fmt.Sprintf("Snapshot creation failed: %s", err.Error())
				response.Errors = append(response.Errors, errorMsg)
				
				r.logger.Error("Snapshot creation failed, continuing workflow",
					zap.Error(err),
					zap.String("workflow_id", request.WorkflowID.String()),
				)
			} else {
				return r.handleWorkflowError(response, err, "snapshot creation failed")
			}
		} else {
			response.SnapshotResult = snapshotResult
			response.SnapshotCreationTime = time.Since(phase2Start)
			
			r.logger.Debug("Phase 2 completed",
				zap.String("workflow_id", request.WorkflowID.String()),
				zap.String("snapshot_id", snapshotResult.SnapshotID),
				zap.Duration("snapshot_time", response.SnapshotCreationTime),
				zap.Float64("snapshot_quality", snapshotResult.QualityScore),
			)
		}
	}

	// Calculate overall metrics
	response.ProcessingTime = time.Since(startTime)
	response.CompletedAt = time.Now()
	response.QualityScore = r.calculateOverallQuality(response)
	response.Status = "completed"

	// Update metrics
	r.updateMetrics(response)

	r.logger.Info("Integrated workflow completed",
		zap.String("workflow_id", request.WorkflowID.String()),
		zap.Duration("total_time", response.ProcessingTime),
		zap.Float64("overall_quality", response.QualityScore),
		zap.Int("warnings", len(response.Warnings)),
		zap.Int("errors", len(response.Errors)),
	)

	return response, nil
}

// SupersedeSnapshot creates a new snapshot and marks the old one as superseded
func (r *RecipeResolverContextIntegration) SupersedeSnapshot(
	ctx context.Context,
	oldSnapshotID string,
	request *IntegratedWorkflowRequest,
	reason string,
) (*IntegratedWorkflowResponse, error) {
	if !r.config.EnableSnapshotSupersession {
		return nil, fmt.Errorf("snapshot supersession is disabled")
	}

	// Execute new workflow
	response, err := r.ExecuteIntegratedWorkflow(ctx, request)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create superseding workflow")
	}

	// If snapshot was created successfully, supersede the old one
	if response.SnapshotResult != nil && response.SnapshotResult.SnapshotID != "" {
		err = r.contextGateway.SupersedeSnapshot(
			ctx,
			oldSnapshotID,
			response.SnapshotResult.SnapshotID,
			reason,
			request.RequestedBy,
		)
		if err != nil {
			r.logger.Error("Failed to supersede snapshot",
				zap.Error(err),
				zap.String("old_snapshot", oldSnapshotID),
				zap.String("new_snapshot", response.SnapshotResult.SnapshotID),
			)
			response.Warnings = append(response.Warnings, 
				fmt.Sprintf("Failed to supersede old snapshot: %s", err.Error()))
		}
	}

	return response, nil
}

// GetWorkflowMetrics returns current workflow metrics
func (r *RecipeResolverContextIntegration) GetWorkflowMetrics() *ContextIntegrationMetrics {
	r.metrics.LastUpdated = time.Now()
	return r.metrics
}

// HealthCheck performs a comprehensive health check
func (r *RecipeResolverContextIntegration) HealthCheck(ctx context.Context) (map[string]interface{}, error) {
	health := map[string]interface{}{
		"service": "recipe_resolver_context_integration",
		"status":  "healthy",
	}

	// Check resolver integration health
	resolverHealth, err := r.resolverIntegration.GetIntegrationHealth(ctx)
	if err != nil {
		health["resolver_integration"] = "unhealthy"
		health["resolver_error"] = err.Error()
		health["status"] = "degraded"
	} else {
		health["resolver_integration"] = resolverHealth.Overall
		if resolverHealth.Overall != "healthy" {
			health["status"] = "degraded"
		}
	}

	// Check context gateway health
	contextHealth, err := r.contextGateway.HealthCheck(ctx)
	if err != nil {
		health["context_gateway"] = "unhealthy"
		health["context_error"] = err.Error()
		health["status"] = "unhealthy"
	} else {
		health["context_gateway"] = contextHealth.Status
		if contextHealth.Status != "healthy" {
			health["status"] = "degraded"
		}
	}

	// Add metrics
	health["metrics"] = r.metrics

	return health, nil
}

// Helper methods

func (r *RecipeResolverContextIntegration) validateWorkflowRequest(request *IntegratedWorkflowRequest) error {
	if request.WorkflowID == uuid.Nil {
		return fmt.Errorf("workflow_id is required")
	}

	if request.RecipeResolutionRequest.RecipeID == "" {
		return fmt.Errorf("recipe_id is required")
	}

	if request.RecipeResolutionRequest.PatientContext.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}

	if request.RequestedBy == "" {
		request.RequestedBy = "medication-service-v2"
	}

	// Set defaults for snapshot creation
	if request.CreateSnapshot {
		if request.SnapshotType == "" {
			request.SnapshotType = "calculation"
		}
		if request.SnapshotPriority == "" {
			request.SnapshotPriority = "normal"
		}
	}

	return nil
}

func (r *RecipeResolverContextIntegration) handleWorkflowError(
	response *IntegratedWorkflowResponse,
	err error,
	context string,
) (*IntegratedWorkflowResponse, error) {
	response.Status = "failed"
	response.CompletedAt = time.Now()
	response.ProcessingTime = time.Since(response.CreatedAt)
	response.Errors = append(response.Errors, fmt.Sprintf("%s: %s", context, err.Error()))

	// Update failure metrics
	r.metrics.FailedWorkflows++
	r.metrics.TotalWorkflows++

	r.logger.Error("Workflow failed",
		zap.String("workflow_id", response.WorkflowID.String()),
		zap.String("context", context),
		zap.Error(err),
		zap.Duration("processing_time", response.ProcessingTime),
	)

	return response, errors.Wrap(err, context)
}

func (r *RecipeResolverContextIntegration) calculateResolutionQuality(resolution *entities.RecipeResolution) float64 {
	// Quality based on field completeness and processing success
	score := 0.0
	
	// Base score for successful resolution
	score += 0.3
	
	// Field completeness (0.4 max)
	if resolution.ResolvedFields != nil {
		totalPossibleFields := 20.0 // Estimate
		resolvedFields := float64(len(resolution.ResolvedFields))
		score += (resolvedFields / totalPossibleFields) * 0.4
	}
	
	// Processing time quality (0.3 max)
	if resolution.ProcessingMetadata != nil {
		if resolution.ProcessingMetadata.ProcessingTime < 10*time.Millisecond {
			score += 0.3
		} else if resolution.ProcessingMetadata.ProcessingTime < 50*time.Millisecond {
			score += 0.2
		} else if resolution.ProcessingMetadata.ProcessingTime < 100*time.Millisecond {
			score += 0.1
		}
	}
	
	return score
}

func (r *RecipeResolverContextIntegration) calculateOverallQuality(response *IntegratedWorkflowResponse) float64 {
	// Weighted combination of resolution and snapshot quality
	score := response.ResolutionQuality * 0.6 // 60% weight to resolution
	
	if response.SnapshotResult != nil {
		score += response.SnapshotResult.QualityScore * 0.4 // 40% weight to snapshot
	} else {
		// If no snapshot, give full weight to resolution
		score = response.ResolutionQuality
	}
	
	return score
}

func (r *RecipeResolverContextIntegration) updateMetrics(response *IntegratedWorkflowResponse) {
	r.metrics.TotalWorkflows++
	
	if response.Status == "completed" {
		r.metrics.SuccessfulWorkflows++
	} else {
		r.metrics.FailedWorkflows++
	}
	
	// Update timing metrics
	r.updateTimingMetrics(response.ProcessingTime)
	r.metrics.EndToEndLatency = time.Duration(
		(int64(r.metrics.EndToEndLatency)*r.metrics.TotalWorkflows + int64(response.ProcessingTime)) /
		(r.metrics.TotalWorkflows + 1))
	
	// Update quality metrics
	if r.metrics.TotalWorkflows == 1 {
		r.metrics.QualityScore = response.QualityScore
	} else {
		r.metrics.QualityScore = (r.metrics.QualityScore*float64(r.metrics.TotalWorkflows-1) + response.QualityScore) / float64(r.metrics.TotalWorkflows)
	}
	
	// Update success rates
	r.metrics.ResolutionSuccessRate = float64(r.metrics.SuccessfulWorkflows) / float64(r.metrics.TotalWorkflows)
	
	if response.SnapshotResult != nil {
		r.metrics.SnapshotsCreated++
		r.metrics.SnapshotSuccessRate = float64(r.metrics.SnapshotsCreated) / float64(r.metrics.TotalWorkflows)
	}
	
	r.metrics.LastUpdated = time.Now()
}

func (r *RecipeResolverContextIntegration) updateTimingMetrics(processingTime time.Duration) {
	if processingTime < 50*time.Millisecond {
		r.metrics.Under50ms++
	} else if processingTime < 100*time.Millisecond {
		r.metrics.Between50And100ms++
	} else if processingTime < 250*time.Millisecond {
		r.metrics.Between100And250ms++
	} else {
		r.metrics.Over250ms++
	}
}

// DefaultContextIntegrationConfig returns default configuration
func DefaultContextIntegrationConfig() ContextIntegrationConfig {
	return ContextIntegrationConfig{
		EnableSnapshotCreation:      true,
		AutoCreateSnapshots:         true,
		SnapshotCreationMode:        "sync",
		MaxConcurrentSnapshots:      10,
		SnapshotCreationTimeout:     30 * time.Second,
		MinResolutionQuality:        0.6,
		RequireValidatedSnapshots:   false,
		ContinueOnSnapshotFailure:   true,
		RetryFailedSnapshots:        true,
		EnableSnapshotSupersession:  true,
		CleanupSupersededSnapshots:  false,
	}
}