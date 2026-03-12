package model

import (
	"time"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/domain"
	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/orchestration"
)

// WorkflowInstance represents a workflow instance in GraphQL
type WorkflowInstance struct {
	ID            string                 `json:"id"`
	DefinitionID  string                 `json:"definitionId"`
	PatientID     string                 `json:"patientId"`
	Status        WorkflowStatus         `json:"status"`
	StartedAt     time.Time              `json:"startedAt"`
	CompletedAt   *time.Time             `json:"completedAt,omitempty"`
	CorrelationID string                 `json:"correlationId"`
	CurrentPhase  string                 `json:"currentPhase"`
	Progress      float64                `json:"progress"`
	ErrorMessage  *string                `json:"errorMessage,omitempty"`
	Context       interface{}            `json:"context,omitempty"`
}

// Patient represents a patient entity for federation
type Patient struct {
	ID string `json:"id"`
}

// MedicationWorkflowInput represents the input for medication workflow execution
type MedicationWorkflowInput struct {
	PatientID         string          `json:"patientId"`
	CorrelationID     *string         `json:"correlationId,omitempty"`
	MedicationRequest interface{}     `json:"medicationRequest"`
	ClinicalIntent    interface{}     `json:"clinicalIntent"`
	ProviderContext   interface{}     `json:"providerContext"`
	ExecutionMode     *ExecutionMode  `json:"executionMode,omitempty"`
	ValidationLevel   *ValidationLevel `json:"validationLevel,omitempty"`
	CommitMode        *CommitMode     `json:"commitMode,omitempty"`
}

// WorkflowConnection represents a paginated list of workflows
type WorkflowConnection struct {
	Edges      []*WorkflowEdge `json:"edges"`
	PageInfo   *PageInfo       `json:"pageInfo"`
	TotalCount int             `json:"totalCount"`
}

// WorkflowEdge represents an edge in the workflow connection
type WorkflowEdge struct {
	Node   *WorkflowInstance `json:"node"`
	Cursor string            `json:"cursor"`
}

// PageInfo represents pagination information
type PageInfo struct {
	HasNextPage     bool     `json:"hasNextPage"`
	HasPreviousPage bool     `json:"hasPreviousPage"`
	StartCursor     *string  `json:"startCursor,omitempty"`
	EndCursor       *string  `json:"endCursor,omitempty"`
}

// Helper functions to convert between domain and GraphQL types

// FromDomainWorkflowInstance converts domain WorkflowInstance to GraphQL type
func FromDomainWorkflowInstance(instance *domain.WorkflowInstance, currentPhase string, progress float64) *WorkflowInstance {
	result := &WorkflowInstance{
		ID:            instance.ID,
		DefinitionID:  instance.DefinitionID,
		PatientID:     instance.PatientID,
		Status:        WorkflowStatusFromDomain(instance.Status),
		StartedAt:     instance.StartedAt,
		CompletedAt:   instance.CompletedAt,
		CorrelationID: instance.CorrelationID,
		CurrentPhase:  currentPhase,
		Progress:      progress,
		Context:       instance.Context,
	}

	if instance.ErrorMessage != nil {
		result.ErrorMessage = instance.ErrorMessage
	}

	return result
}

// WorkflowStatusFromDomain converts domain WorkflowStatus to GraphQL enum
func WorkflowStatusFromDomain(status domain.WorkflowStatus) WorkflowStatus {
	switch status {
	case domain.WorkflowStatusPending:
		return WorkflowStatusPending
	case domain.WorkflowStatusRunning:
		return WorkflowStatusRunning
	case domain.WorkflowStatusCompleted:
		return WorkflowStatusCompleted
	case domain.WorkflowStatusCompletedWithWarnings:
		return WorkflowStatusCompletedWithWarnings
	case domain.WorkflowStatusFailed:
		return WorkflowStatusFailed
	case domain.WorkflowStatusCancelled:
		return WorkflowStatusCancelled
	case domain.WorkflowStatusDeleted:
		return WorkflowStatusDeleted
	default:
		return WorkflowStatusPending
	}
}

// ToMedicationOrchestrationRequest converts GraphQL input to orchestration request
func (input *MedicationWorkflowInput) ToOrchestrationRequest() *orchestration.OrchestrationRequest {
	request := &orchestration.OrchestrationRequest{
		PatientID:         input.PatientID,
		MedicationRequest: input.MedicationRequest.(map[string]interface{}),
		ClinicalIntent:    input.ClinicalIntent.(map[string]interface{}),
		ProviderContext:   input.ProviderContext.(map[string]interface{}),
	}

	if input.CorrelationID != nil {
		request.CorrelationID = *input.CorrelationID
	}

	if input.ExecutionMode != nil {
		switch *input.ExecutionMode {
		case ExecutionModeBasic:
			request.ExecutionMode = "basic"
		case ExecutionModeStandard:
			request.ExecutionMode = "standard"
		case ExecutionModeAdvanced:
			request.ExecutionMode = "advanced"
		}
	}

	if input.ValidationLevel != nil {
		switch *input.ValidationLevel {
		case ValidationLevelBasic:
			request.ValidationLevel = "basic"
		case ValidationLevelComprehensive:
			request.ValidationLevel = "comprehensive"
		case ValidationLevelCritical:
			request.ValidationLevel = "critical"
		}
	}

	if input.CommitMode != nil {
		switch *input.CommitMode {
		case CommitModeImmediate:
			request.CommitMode = "immediate"
		case CommitModeConditional:
			request.CommitMode = "conditional"
		case CommitModeSafeOnly:
			request.CommitMode = "safe_only"
		case CommitModeNever:
			request.CommitMode = "never"
		}
	}

	return request
}

// Reference resolver for federation
func (r *Patient) WorkflowsResolver() []*WorkflowInstance {
	// This would be implemented to fetch workflows for a patient
	// Left as placeholder for federation resolver
	return []*WorkflowInstance{}
}