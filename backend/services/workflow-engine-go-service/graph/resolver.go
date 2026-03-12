package graph

import (
	"context"
	"encoding/base64"
	"fmt"
	"strconv"
	"time"

	"go.uber.org/zap"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/graph/model"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/orchestration"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/services"
)

// Resolver is the root GraphQL resolver
type Resolver struct {
	orchestrationService *services.OrchestrationService
	logger               *zap.Logger
}

// NewResolver creates a new GraphQL resolver
func NewResolver(orchestrationService *services.OrchestrationService, logger *zap.Logger) *Resolver {
	return &Resolver{
		orchestrationService: orchestrationService,
		logger:               logger,
	}
}

// Query resolver implementation
type queryResolver struct{ *Resolver }

// Mutation resolver implementation  
type mutationResolver struct{ *Resolver }

// Subscription resolver implementation
type subscriptionResolver struct{ *Resolver }

// Patient resolver implementation for federation
type patientResolver struct{ *Resolver }

// Query returns QueryResolver interface
func (r *Resolver) Query() QueryResolver { return &queryResolver{r} }

// Mutation returns MutationResolver interface
func (r *Resolver) Mutation() MutationResolver { return &mutationResolver{r} }

// Subscription returns SubscriptionResolver interface
func (r *Resolver) Subscription() SubscriptionResolver { return &subscriptionResolver{r} }

// Patient returns PatientResolver interface for federation
func (r *Resolver) Patient() PatientResolver { return &patientResolver{r} }

// Query resolver implementations

func (q *queryResolver) Workflow(ctx context.Context, id string) (*model.WorkflowInstance, error) {
	q.logger.Info("GraphQL: Getting workflow", zap.String("workflow_id", id))

	response, err := q.orchestrationService.GetWorkflowStatus(ctx, id)
	if err != nil {
		q.logger.Error("Failed to get workflow", zap.String("workflow_id", id), zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve workflow: %w", err)
	}

	return &model.WorkflowInstance{
		ID:            response.WorkflowInstanceID,
		DefinitionID:  response.DefinitionID,
		PatientID:     response.PatientID,
		Status:        model.WorkflowStatus(response.Status),
		StartedAt:     response.StartedAt,
		CompletedAt:   response.CompletedAt,
		CorrelationID: response.CorrelationID,
		CurrentPhase:  response.CurrentPhase,
		Progress:      response.Progress,
		ErrorMessage:  &response.ErrorMessage,
		Context:       response.Context,
	}, nil
}

func (q *queryResolver) Workflows(ctx context.Context, filters *model.WorkflowListFilters) (*model.WorkflowConnection, error) {
	q.logger.Info("GraphQL: Listing workflows")

	// Convert GraphQL filters to service filters
	serviceFilters := &services.WorkflowListFilters{
		Limit:  50,
		Offset: 0,
	}

	if filters != nil {
		if filters.PatientID != nil {
			serviceFilters.PatientID = *filters.PatientID
		}
		if filters.Status != nil {
			serviceFilters.Status = string(*filters.Status)
		}
		if filters.DefinitionID != nil {
			serviceFilters.DefinitionID = *filters.DefinitionID
		}
		if filters.StartedAfter != nil {
			serviceFilters.StartedAfter = *filters.StartedAfter
		}
		if filters.StartedBefore != nil {
			serviceFilters.StartedBefore = *filters.StartedBefore
		}
		if filters.Limit != nil && *filters.Limit > 0 && *filters.Limit <= 1000 {
			serviceFilters.Limit = *filters.Limit
		}
		if filters.Offset != nil && *filters.Offset >= 0 {
			serviceFilters.Offset = *filters.Offset
		}
	}

	response, err := q.orchestrationService.ListWorkflowInstances(ctx, serviceFilters)
	if err != nil {
		q.logger.Error("Failed to list workflows", zap.Error(err))
		return nil, fmt.Errorf("failed to retrieve workflows: %w", err)
	}

	// Convert to GraphQL connection format
	edges := make([]*model.WorkflowEdge, len(response.Workflows))
	for i, workflow := range response.Workflows {
		cursor := encodeCursor(workflow.WorkflowInstanceID, workflow.StartedAt)
		edges[i] = &model.WorkflowEdge{
			Node: &model.WorkflowInstance{
				ID:            workflow.WorkflowInstanceID,
				DefinitionID:  workflow.DefinitionID,
				PatientID:     workflow.PatientID,
				Status:        model.WorkflowStatus(workflow.Status),
				StartedAt:     workflow.StartedAt,
				CompletedAt:   workflow.CompletedAt,
				CorrelationID: workflow.CorrelationID,
				CurrentPhase:  workflow.CurrentPhase,
				Progress:      workflow.Progress,
				ErrorMessage:  &workflow.ErrorMessage,
			},
			Cursor: cursor,
		}
	}

	pageInfo := &model.PageInfo{
		HasNextPage:     response.HasMore,
		HasPreviousPage: response.Offset > 0,
	}

	if len(edges) > 0 {
		startCursor := edges[0].Cursor
		endCursor := edges[len(edges)-1].Cursor
		pageInfo.StartCursor = &startCursor
		pageInfo.EndCursor = &endCursor
	}

	return &model.WorkflowConnection{
		Edges:      edges,
		PageInfo:   pageInfo,
		TotalCount: int(response.Total),
	}, nil
}

func (q *queryResolver) Patient(ctx context.Context, id string) (*model.Patient, error) {
	// Federation resolver - just return the patient with ID
	return &model.Patient{ID: id}, nil
}

func (q *queryResolver) Health(ctx context.Context) (*model.SystemHealth, error) {
	q.logger.Info("GraphQL: Getting system health")

	response := q.orchestrationService.GetSystemHealth(ctx)

	return &model.SystemHealth{
		Status:            model.HealthStatus(response.Status),
		Service:           response.Service,
		DatabaseConnected: response.DatabaseConnected,
		ExternalServices:  response.ExternalServices,
		CheckedAt:         response.CheckedAt,
	}, nil
}

func (q *queryResolver) Service(ctx context.Context) (*model.Service, error) {
	// Apollo Federation service discovery
	return &model.Service{
		SDL: getSDL(), // This would return the schema definition
	}, nil
}

// Mutation resolver implementations

func (m *mutationResolver) ExecuteMedicationWorkflow(ctx context.Context, input model.MedicationWorkflowInput) (*model.MedicationWorkflowResult, error) {
	m.logger.Info("GraphQL: Executing medication workflow",
		zap.String("patient_id", input.PatientID),
		zap.Any("correlation_id", input.CorrelationID))

	// Convert GraphQL input to orchestration request
	request := input.ToOrchestrationRequest()

	// Execute workflow
	response, err := m.orchestrationService.ExecuteMedicationWorkflow(ctx, request)
	if err != nil {
		m.logger.Error("Medication workflow execution failed", zap.Error(err))
		return nil, fmt.Errorf("workflow execution failed: %w", err)
	}

	// Convert orchestration response to GraphQL response
	result := &model.MedicationWorkflowResult{
		WorkflowInstanceID: response.WorkflowInstanceID,
		SnapshotID:        response.SnapshotID,
		ProposalSetID:     response.ProposalSetID,
		ValidationID:      response.ValidationID,
		MedicationOrderID: &response.MedicationOrderID,
		Status:           model.WorkflowExecutionStatus(response.Status),
		Message:          &response.Message,
		Errors:           response.Errors,
	}

	// Convert ranked proposals
	proposals := make([]*model.MedicationProposal, len(response.RankedProposals))
	for i, proposal := range response.RankedProposals {
		proposals[i] = &model.MedicationProposal{
			ProposalID: fmt.Sprintf("proposal-%d", i),
			Medication: proposal,
			// These would be extracted from the proposal data structure
			Dosing:             proposal,
			ClinicalRationale:  "Clinical rationale from proposal",
			ConfidenceScore:    0.85, // Would come from proposal
			RiskScore:          0.15, // Would come from proposal
			Contraindications:  []string{},
			Warnings:           []string{},
		}
	}
	result.RankedProposals = proposals

	// Convert validation result
	if response.ValidationResult != nil {
		result.ValidationResult = &model.ValidationSummary{
			Verdict:           model.ValidationVerdict(response.ValidationResult.Verdict),
			OverallRiskScore:  response.ValidationResult.OverallRiskScore,
			FindingsCount:     response.ValidationResult.FindingsCount,
			ExecutedEngines:   response.ValidationResult.ExecutedEngines,
			OverrideTokens:    response.ValidationResult.OverrideTokens,
		}

		// Convert critical findings
		criticalFindings := make([]*model.ValidationFinding, len(response.ValidationResult.CriticalFindings))
		for i, finding := range response.ValidationResult.CriticalFindings {
			criticalFindings[i] = &model.ValidationFinding{
				FindingID:            finding.FindingID,
				Severity:             model.ValidationSeverity(finding.Severity),
				Category:             finding.Category,
				Description:          finding.Description,
				ClinicalSignificance: finding.ClinicalSignificance,
				Recommendation:       finding.Recommendation,
				ConfidenceScore:      finding.ConfidenceScore,
				Source:               finding.Source,
				Evidence:             finding.Evidence,
				Overridable:          finding.Overridable,
			}
		}
		result.ValidationResult.CriticalFindings = criticalFindings
	}

	// Convert commit result
	if response.CommitResult != nil {
		result.CommitResult = &model.CommitSummary{
			MedicationOrderID:      response.CommitResult.MedicationOrderID,
			FhirResourceID:         response.CommitResult.FHIRResourceID,
			PersistenceStatus:      response.CommitResult.PersistenceStatus,
			EventPublicationStatus: response.CommitResult.EventPublicationStatus,
			AuditTrailID:           response.CommitResult.AuditTrailID,
			CommittedAt:            response.CommitResult.CommittedAt,
		}
	}

	// Convert execution metrics
	if response.ExecutionMetrics != nil {
		result.ExecutionMetrics = &model.ExecutionMetrics{
			TotalDurationMs:    int(response.ExecutionMetrics.TotalDuration.Milliseconds()),
			CalculateDurationMs: int(response.ExecutionMetrics.CalculateDuration.Milliseconds()),
			ValidateDurationMs:  int(response.ExecutionMetrics.ValidateDuration.Milliseconds()),
			CommitDurationMs:    intPtr(int(response.ExecutionMetrics.CommitDuration.Milliseconds())),
			PhaseBreakdown:      response.ExecutionMetrics.PhaseBreakdown,
		}
	}

	return result, nil
}

func (m *mutationResolver) CancelWorkflow(ctx context.Context, workflowID string) (*model.WorkflowInstance, error) {
	m.logger.Info("GraphQL: Cancelling workflow", zap.String("workflow_id", workflowID))

	// This would be implemented to cancel a workflow
	// For now, return a placeholder implementation
	return nil, fmt.Errorf("workflow cancellation not implemented")
}

// Subscription resolver implementations

func (s *subscriptionResolver) WorkflowUpdates(ctx context.Context, workflowID string) (<-chan *model.WorkflowInstance, error) {
	s.logger.Info("GraphQL: Starting workflow updates subscription", zap.String("workflow_id", workflowID))
	
	// Create a channel for workflow updates
	updates := make(chan *model.WorkflowInstance)
	
	// This would be implemented with actual subscription logic
	// For now, close the channel immediately
	close(updates)
	
	return updates, nil
}

func (s *subscriptionResolver) WorkflowsByPatient(ctx context.Context, patientID string) (<-chan *model.WorkflowInstance, error) {
	s.logger.Info("GraphQL: Starting patient workflow updates subscription", zap.String("patient_id", patientID))
	
	// Create a channel for workflow updates
	updates := make(chan *model.WorkflowInstance)
	
	// This would be implemented with actual subscription logic
	// For now, close the channel immediately
	close(updates)
	
	return updates, nil
}

// Patient resolver implementation for federation

func (p *patientResolver) Workflows(ctx context.Context, obj *model.Patient) ([]*model.WorkflowInstance, error) {
	p.logger.Info("GraphQL: Getting workflows for patient", zap.String("patient_id", obj.ID))

	filters := &services.WorkflowListFilters{
		PatientID: obj.ID,
		Limit:     100, // Reasonable default for federation
		Offset:    0,
	}

	response, err := p.orchestrationService.ListWorkflowInstances(ctx, filters)
	if err != nil {
		p.logger.Error("Failed to get patient workflows", zap.String("patient_id", obj.ID), zap.Error(err))
		return []*model.WorkflowInstance{}, nil // Return empty list instead of error for federation
	}

	workflows := make([]*model.WorkflowInstance, len(response.Workflows))
	for i, workflow := range response.Workflows {
		workflows[i] = &model.WorkflowInstance{
			ID:            workflow.WorkflowInstanceID,
			DefinitionID:  workflow.DefinitionID,
			PatientID:     workflow.PatientID,
			Status:        model.WorkflowStatus(workflow.Status),
			StartedAt:     workflow.StartedAt,
			CompletedAt:   workflow.CompletedAt,
			CorrelationID: workflow.CorrelationID,
			CurrentPhase:  workflow.CurrentPhase,
			Progress:      workflow.Progress,
			ErrorMessage:  &workflow.ErrorMessage,
		}
	}

	return workflows, nil
}

// Helper functions

func encodeCursor(id string, timestamp time.Time) string {
	cursor := fmt.Sprintf("%s:%d", id, timestamp.Unix())
	return base64.StdEncoding.EncodeToString([]byte(cursor))
}

func intPtr(i int) *int {
	return &i
}

func getSDL() string {
	// This would return the actual schema definition for Apollo Federation
	// For now, return a placeholder
	return "# Workflow Engine Service SDL"
}