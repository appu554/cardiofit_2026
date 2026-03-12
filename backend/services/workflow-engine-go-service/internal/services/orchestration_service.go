package services

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/orchestration"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/repositories"
)

// OrchestrationService provides business logic for workflow orchestration
type OrchestrationService struct {
	strategicOrchestrator *orchestration.StrategicOrchestrator
	workflowRepo          repositories.WorkflowRepository
	logger                *zap.Logger
}

// NewOrchestrationService creates a new orchestration service
func NewOrchestrationService(
	strategicOrchestrator *orchestration.StrategicOrchestrator,
	workflowRepo repositories.WorkflowRepository,
	logger *zap.Logger,
) *OrchestrationService {
	return &OrchestrationService{
		strategicOrchestrator: strategicOrchestrator,
		workflowRepo:          workflowRepo,
		logger:                logger,
	}
}

// ExecuteMedicationWorkflow handles medication workflow orchestration with business logic
func (s *OrchestrationService) ExecuteMedicationWorkflow(ctx context.Context, request *orchestration.OrchestrationRequest) (*orchestration.OrchestrationResponse, error) {
	// Validate request
	if err := s.validateOrchestrationRequest(request); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Add request timing context
	ctx = context.WithValue(ctx, "start_time", time.Now())

	s.logger.Info("Processing medication workflow request",
		zap.String("correlation_id", request.CorrelationID),
		zap.String("patient_id", request.PatientID),
		zap.String("execution_mode", request.ExecutionMode))

	// Execute orchestration
	response, err := s.strategicOrchestrator.ExecuteMedicationWorkflow(ctx, request)
	if err != nil {
		s.logger.Error("Medication workflow execution failed",
			zap.String("correlation_id", request.CorrelationID),
			zap.Error(err))
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Log execution summary
	s.logExecutionSummary(response)

	return response, nil
}

// GetWorkflowStatus retrieves the current status of a workflow instance
func (s *OrchestrationService) GetWorkflowStatus(ctx context.Context, workflowInstanceID string) (*WorkflowStatusResponse, error) {
	instance, err := s.workflowRepo.GetByID(ctx, workflowInstanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to retrieve workflow: %w", err)
	}

	response := &WorkflowStatusResponse{
		WorkflowInstanceID: instance.ID,
		DefinitionID:      instance.DefinitionID,
		PatientID:         instance.PatientID,
		Status:            string(instance.Status),
		StartedAt:         instance.StartedAt,
		CompletedAt:       instance.CompletedAt,
		CorrelationID:     instance.CorrelationID,
		CurrentPhase:      s.determineCurrentPhase(instance),
		Progress:          s.calculateProgress(instance),
		Context:           instance.Context,
	}

	if instance.ErrorMessage != nil {
		response.ErrorMessage = *instance.ErrorMessage
	}

	return response, nil
}

// ListWorkflowInstances retrieves workflow instances with filtering options
func (s *OrchestrationService) ListWorkflowInstances(ctx context.Context, filters *WorkflowListFilters) (*WorkflowListResponse, error) {
	// Set default filters if not provided
	if filters == nil {
		filters = &WorkflowListFilters{
			Limit:  50,
			Offset: 0,
		}
	}

	instances, total, err := s.workflowRepo.List(ctx, repositoryFiltersFromService(filters))
	if err != nil {
		return nil, fmt.Errorf("failed to list workflows: %w", err)
	}

	// Convert to response format
	workflowSummaries := make([]WorkflowSummary, len(instances))
	for i, instance := range instances {
		workflowSummaries[i] = WorkflowSummary{
			WorkflowInstanceID: instance.ID,
			DefinitionID:      instance.DefinitionID,
			PatientID:         instance.PatientID,
			Status:            string(instance.Status),
			StartedAt:         instance.StartedAt,
			CompletedAt:       instance.CompletedAt,
			CorrelationID:     instance.CorrelationID,
			CurrentPhase:      s.determineCurrentPhase(&instance),
			Progress:          s.calculateProgress(&instance),
		}

		if instance.ErrorMessage != nil {
			workflowSummaries[i].ErrorMessage = *instance.ErrorMessage
		}
	}

	return &WorkflowListResponse{
		Workflows:   workflowSummaries,
		Total:       total,
		Limit:       filters.Limit,
		Offset:      filters.Offset,
		HasMore:     int64(filters.Offset+len(instances)) < total,
	}, nil
}

// GetSystemHealth checks the health of all orchestration components
func (s *OrchestrationService) GetSystemHealth(ctx context.Context) *SystemHealthResponse {
	healthResults := s.strategicOrchestrator.HealthCheck(ctx)

	response := &SystemHealthResponse{
		Status:            "healthy",
		Service:           "workflow-engine-service",
		DatabaseConnected: true, // This would be checked against the database
		ExternalServices:  healthResults,
		CheckedAt:         time.Now(),
	}

	// Determine overall health status
	for serviceName, status := range healthResults {
		if status != "healthy" {
			response.Status = "degraded"
			s.logger.Warn("External service unhealthy",
				zap.String("service", serviceName),
				zap.String("status", status))
		}
	}

	return response
}

// Helper methods

func (s *OrchestrationService) validateOrchestrationRequest(request *orchestration.OrchestrationRequest) error {
	if request.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}

	if request.MedicationRequest == nil || len(request.MedicationRequest) == 0 {
		return fmt.Errorf("medication_request is required")
	}

	// Validate execution mode if provided
	if request.ExecutionMode != "" {
		validModes := map[string]bool{
			"basic":    true,
			"standard": true,
			"advanced": true,
		}
		if !validModes[request.ExecutionMode] {
			return fmt.Errorf("invalid execution_mode: %s", request.ExecutionMode)
		}
	}

	// Validate validation level if provided
	if request.ValidationLevel != "" {
		validLevels := map[string]bool{
			"basic":         true,
			"comprehensive": true,
			"critical":      true,
		}
		if !validLevels[request.ValidationLevel] {
			return fmt.Errorf("invalid validation_level: %s", request.ValidationLevel)
		}
	}

	// Validate commit mode if provided
	if request.CommitMode != "" {
		validModes := map[string]bool{
			"immediate":   true,
			"conditional": true,
			"safe_only":   true,
			"never":       true,
		}
		if !validModes[request.CommitMode] {
			return fmt.Errorf("invalid commit_mode: %s", request.CommitMode)
		}
	}

	return nil
}

func (s *OrchestrationService) logExecutionSummary(response *orchestration.OrchestrationResponse) {
	metrics := response.ExecutionMetrics
	
	s.logger.Info("Workflow execution summary",
		zap.String("workflow_instance_id", response.WorkflowInstanceID),
		zap.String("status", response.Status),
		zap.Duration("total_duration", metrics.TotalDuration),
		zap.Duration("calculate_duration", metrics.CalculateDuration),
		zap.Duration("validate_duration", metrics.ValidateDuration),
		zap.Duration("commit_duration", metrics.CommitDuration),
		zap.Int("error_count", len(response.Errors)))

	// Log performance warnings if phases exceed targets
	if metrics.CalculateDuration > 175*time.Millisecond {
		s.logger.Warn("Calculate phase exceeded target",
			zap.Duration("duration", metrics.CalculateDuration),
			zap.Duration("target", 175*time.Millisecond))
	}

	if metrics.ValidateDuration > 100*time.Millisecond {
		s.logger.Warn("Validate phase exceeded target",
			zap.Duration("duration", metrics.ValidateDuration),
			zap.Duration("target", 100*time.Millisecond))
	}

	if metrics.CommitDuration > 50*time.Millisecond {
		s.logger.Warn("Commit phase exceeded target",
			zap.Duration("duration", metrics.CommitDuration),
			zap.Duration("target", 50*time.Millisecond))
	}

	if metrics.TotalDuration > 325*time.Millisecond {
		s.logger.Warn("Total workflow exceeded target",
			zap.Duration("duration", metrics.TotalDuration),
			zap.Duration("target", 325*time.Millisecond))
	}
}

func (s *OrchestrationService) determineCurrentPhase(instance *repositories.WorkflowInstance) string {
	switch instance.Status {
	case "running":
		// Would need additional logic to determine exact phase from instance context
		// For now, return a default
		return "executing"
	case "completed", "completed_with_warnings":
		return "completed"
	case "failed":
		return "failed"
	default:
		return "unknown"
	}
}

func (s *OrchestrationService) calculateProgress(instance *repositories.WorkflowInstance) float64 {
	switch instance.Status {
	case "running":
		return 0.5 // 50% progress for running workflows
	case "completed", "completed_with_warnings":
		return 1.0 // 100% progress for completed workflows
	case "failed":
		return 0.0 // 0% progress for failed workflows
	default:
		return 0.0
	}
}

// Response types

type WorkflowStatusResponse struct {
	WorkflowInstanceID string                 `json:"workflow_instance_id"`
	DefinitionID       string                 `json:"definition_id"`
	PatientID          string                 `json:"patient_id"`
	Status             string                 `json:"status"`
	StartedAt          time.Time              `json:"started_at"`
	CompletedAt        *time.Time             `json:"completed_at,omitempty"`
	CorrelationID      string                 `json:"correlation_id"`
	CurrentPhase       string                 `json:"current_phase"`
	Progress           float64                `json:"progress"`
	ErrorMessage       string                 `json:"error_message,omitempty"`
	Context            map[string]interface{} `json:"context,omitempty"`
}

type WorkflowListFilters struct {
	PatientID     string    `json:"patient_id,omitempty"`
	Status        string    `json:"status,omitempty"`
	DefinitionID  string    `json:"definition_id,omitempty"`
	StartedAfter  time.Time `json:"started_after,omitempty"`
	StartedBefore time.Time `json:"started_before,omitempty"`
	Limit         int       `json:"limit"`
	Offset        int       `json:"offset"`
}

type WorkflowSummary struct {
	WorkflowInstanceID string     `json:"workflow_instance_id"`
	DefinitionID       string     `json:"definition_id"`
	PatientID          string     `json:"patient_id"`
	Status             string     `json:"status"`
	StartedAt          time.Time  `json:"started_at"`
	CompletedAt        *time.Time `json:"completed_at,omitempty"`
	CorrelationID      string     `json:"correlation_id"`
	CurrentPhase       string     `json:"current_phase"`
	Progress           float64    `json:"progress"`
	ErrorMessage       string     `json:"error_message,omitempty"`
}

type WorkflowListResponse struct {
	Workflows []WorkflowSummary `json:"workflows"`
	Total     int64             `json:"total"`
	Limit     int               `json:"limit"`
	Offset    int               `json:"offset"`
	HasMore   bool              `json:"has_more"`
}

type SystemHealthResponse struct {
	Status            string            `json:"status"`
	Service           string            `json:"service"`
	DatabaseConnected bool              `json:"database_connected"`
	ExternalServices  map[string]string `json:"external_services"`
	CheckedAt         time.Time         `json:"checked_at"`
}

// Helper function to convert service filters to repository filters
func repositoryFiltersFromService(filters *WorkflowListFilters) *repositories.WorkflowListOptions {
	return &repositories.WorkflowListOptions{
		PatientID:     filters.PatientID,
		Status:        filters.Status,
		DefinitionID:  filters.DefinitionID,
		StartedAfter:  filters.StartedAfter,
		StartedBefore: filters.StartedBefore,
		Limit:         filters.Limit,
		Offset:        filters.Offset,
	}
}