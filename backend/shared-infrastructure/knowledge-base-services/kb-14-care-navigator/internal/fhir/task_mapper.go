// Package fhir provides FHIR R4 resource mapping for KB-14 Care Navigator
package fhir

import (
	"fmt"
	"time"

	"kb-14-care-navigator/internal/models"
)

// TaskMapper handles conversion between KB-14 Tasks and FHIR R4 Task resources
type TaskMapper struct {
	baseURL string
}

// NewTaskMapper creates a new TaskMapper
func NewTaskMapper(baseURL string) *TaskMapper {
	return &TaskMapper{
		baseURL: baseURL,
	}
}

// ToFHIR converts a KB-14 Task to a FHIR R4 Task resource
func (m *TaskMapper) ToFHIR(task *models.Task) *models.FHIRTask {
	fhirTask := &models.FHIRTask{
		ResourceType: "Task",
		ID:           task.ID.String(),
		Status:       m.mapStatusToFHIR(task.Status),
		Intent:       "order",
		Priority:     m.mapPriorityToFHIR(task.Priority),
		Description:  task.Title,
	}

	// Set code (task type)
	fhirTask.Code = &models.FHIRCodeableConcept{
		Coding: []models.FHIRCoding{{
			System:  "http://cardiofit.health/fhir/CodeSystem/task-type",
			Code:    string(task.Type),
			Display: m.getTaskTypeDisplay(task.Type),
		}},
		Text: m.getTaskTypeDisplay(task.Type),
	}

	// Set patient reference
	fhirTask.For = &models.FHIRReference{
		Reference: fmt.Sprintf("Patient/%s", task.PatientID),
		Type:      "Patient",
	}

	// Set encounter reference if available
	if task.EncounterID != "" {
		fhirTask.Encounter = &models.FHIRReference{
			Reference: fmt.Sprintf("Encounter/%s", task.EncounterID),
			Type:      "Encounter",
		}
	}

	// Set execution period
	fhirTask.ExecutionPeriod = &models.FHIRPeriod{
		Start: task.CreatedAt.Format(time.RFC3339),
	}
	if task.CompletedAt != nil {
		fhirTask.ExecutionPeriod.End = task.CompletedAt.Format(time.RFC3339)
	}

	// Set owner (assignee)
	if task.AssignedTo != nil {
		fhirTask.Owner = &models.FHIRReference{
			Reference: fmt.Sprintf("Practitioner/%s", task.AssignedTo.String()),
			Type:      "Practitioner",
		}
	}

	// Set requester (source system)
	fhirTask.Requester = &models.FHIRReference{
		Display: string(task.Source),
	}

	// Set restriction period (due date)
	if task.DueDate != nil {
		fhirTask.Restriction = &models.FHIRTaskRestriction{
			Period: &models.FHIRPeriod{
				End: task.DueDate.Format(time.RFC3339),
			},
		}
	}

	// Set authored on
	authoredOn := task.CreatedAt.Format(time.RFC3339)
	fhirTask.AuthoredOn = &authoredOn

	// Set last modified
	lastModified := task.UpdatedAt.Format(time.RFC3339)
	fhirTask.LastModified = &lastModified

	// Set notes
	if len(task.Notes) > 0 {
		fhirTask.Note = make([]models.FHIRAnnotation, len(task.Notes))
		for i, note := range task.Notes {
			fhirTask.Note[i] = models.FHIRAnnotation{
				AuthorString: note.Author,
				Time:         note.CreatedAt.Format(time.RFC3339),
				Text:         note.Content,
			}
		}
	}

	// Set business status (escalation level)
	if task.EscalationLevel > 0 {
		fhirTask.BusinessStatus = &models.FHIRCodeableConcept{
			Coding: []models.FHIRCoding{{
				System:  "http://cardiofit.health/fhir/CodeSystem/escalation-level",
				Code:    fmt.Sprintf("level-%d", task.EscalationLevel),
				Display: m.getEscalationDisplay(task.EscalationLevel),
			}},
		}
	}

	// Add identifier
	fhirTask.Identifier = []models.FHIRIdentifier{{
		System: "http://cardiofit.health/fhir/identifier/task-id",
		Value:  task.TaskID,
	}}

	return fhirTask
}

// FromFHIR converts a FHIR R4 Task resource to a KB-14 Task
func (m *TaskMapper) FromFHIR(fhirTask *models.FHIRTask) *models.Task {
	task := &models.Task{
		Title:       fhirTask.Description,
		Status:      m.mapStatusFromFHIR(fhirTask.Status),
		Priority:    m.mapPriorityFromFHIR(fhirTask.Priority),
		Description: fhirTask.Description,
	}

	// Extract task type from code
	if fhirTask.Code != nil && len(fhirTask.Code.Coding) > 0 {
		task.Type = models.TaskType(fhirTask.Code.Coding[0].Code)
	}

	// Extract patient ID
	if fhirTask.For != nil {
		task.PatientID = extractIDFromReference(fhirTask.For.Reference)
	}

	// Extract encounter ID
	if fhirTask.Encounter != nil {
		task.EncounterID = extractIDFromReference(fhirTask.Encounter.Reference)
	}

	return task
}

// CreateBundle creates a FHIR Bundle containing Task resources
func (m *TaskMapper) CreateBundle(items []models.WorklistItem, total int) map[string]interface{} {
	entries := make([]map[string]interface{}, len(items))

	for i, item := range items {
		// Create minimal task representation from WorklistItem
		task := &models.Task{
			ID:              item.TaskID,
			TaskID:          item.TaskNumber,
			Type:            item.Type,
			Status:          item.Status,
			Priority:        item.Priority,
			Title:           item.Title,
			Description:     item.Description,
			PatientID:       item.PatientID,
			AssignedTo:      item.AssignedTo,
			DueDate:         item.DueDate,
			EscalationLevel: item.EscalationLevel,
			CreatedAt:       item.CreatedAt,
		}

		fhirTask := m.ToFHIR(task)
		entries[i] = map[string]interface{}{
			"fullUrl":  fmt.Sprintf("%s/fhir/Task/%s", m.baseURL, task.ID.String()),
			"resource": fhirTask,
		}
	}

	return map[string]interface{}{
		"resourceType": "Bundle",
		"type":         "searchset",
		"total":        total,
		"entry":        entries,
	}
}

// mapStatusToFHIR maps internal status to FHIR Task status
func (m *TaskMapper) mapStatusToFHIR(status models.TaskStatus) string {
	switch status {
	case models.TaskStatusCreated:
		return "requested"
	case models.TaskStatusAssigned:
		return "accepted"
	case models.TaskStatusInProgress:
		return "in-progress"
	case models.TaskStatusCompleted, models.TaskStatusVerified:
		return "completed"
	case models.TaskStatusCancelled:
		return "cancelled"
	case models.TaskStatusBlocked:
		return "on-hold"
	case models.TaskStatusDeclined:
		return "rejected"
	case models.TaskStatusEscalated:
		return "ready" // No direct mapping, using ready
	default:
		return "draft"
	}
}

// mapStatusFromFHIR maps FHIR Task status to internal status
func (m *TaskMapper) mapStatusFromFHIR(status string) models.TaskStatus {
	switch status {
	case "draft", "requested":
		return models.TaskStatusCreated
	case "received", "accepted":
		return models.TaskStatusAssigned
	case "in-progress":
		return models.TaskStatusInProgress
	case "completed":
		return models.TaskStatusCompleted
	case "cancelled":
		return models.TaskStatusCancelled
	case "on-hold":
		return models.TaskStatusBlocked
	case "rejected", "failed":
		return models.TaskStatusDeclined
	default:
		return models.TaskStatusCreated
	}
}

// mapPriorityToFHIR maps internal priority to FHIR Task priority
func (m *TaskMapper) mapPriorityToFHIR(priority models.TaskPriority) string {
	switch priority {
	case models.TaskPriorityCritical:
		return "stat"
	case models.TaskPriorityHigh:
		return "asap"
	case models.TaskPriorityMedium:
		return "urgent"
	case models.TaskPriorityLow:
		return "routine"
	default:
		return "routine"
	}
}

// mapPriorityFromFHIR maps FHIR Task priority to internal priority
func (m *TaskMapper) mapPriorityFromFHIR(priority string) models.TaskPriority {
	switch priority {
	case "stat":
		return models.TaskPriorityCritical
	case "asap":
		return models.TaskPriorityHigh
	case "urgent":
		return models.TaskPriorityMedium
	case "routine":
		return models.TaskPriorityLow
	default:
		return models.TaskPriorityMedium
	}
}

// getTaskTypeDisplay returns a human-readable display for task type
func (m *TaskMapper) getTaskTypeDisplay(taskType models.TaskType) string {
	switch taskType {
	case models.TaskTypeCriticalLabReview:
		return "Critical Lab Review"
	case models.TaskTypeMedicationReview:
		return "Medication Review"
	case models.TaskTypeAbnormalResult:
		return "Abnormal Result Review"
	case models.TaskTypeTherapeuticChange:
		return "Therapeutic Change"
	case models.TaskTypeCarePlanReview:
		return "Care Plan Review"
	case models.TaskTypeAcuteProtocolDeadline:
		return "Acute Protocol Deadline"
	case models.TaskTypeCareGapClosure:
		return "Care Gap Closure"
	case models.TaskTypeMonitoringOverdue:
		return "Monitoring Overdue"
	case models.TaskTypeTransitionFollowup:
		return "Transition Follow-up"
	case models.TaskTypeAnnualWellness:
		return "Annual Wellness Visit"
	case models.TaskTypeChronicCareMgmt:
		return "Chronic Care Management"
	case models.TaskTypeAppointmentRemind:
		return "Appointment Reminder"
	case models.TaskTypeMissedAppointment:
		return "Missed Appointment Follow-up"
	case models.TaskTypeScreeningOutreach:
		return "Screening Outreach"
	case models.TaskTypeMedicationRefill:
		return "Medication Refill"
	case models.TaskTypePriorAuthNeeded:
		return "Prior Authorization Needed"
	case models.TaskTypeReferralProcessing:
		return "Referral Processing"
	default:
		return string(taskType)
	}
}

// getEscalationDisplay returns a human-readable display for escalation level
func (m *TaskMapper) getEscalationDisplay(level int) string {
	switch level {
	case 1:
		return "Warning"
	case 2:
		return "Urgent"
	case 3:
		return "Critical"
	case 4:
		return "Executive"
	default:
		return fmt.Sprintf("Level %d", level)
	}
}

// extractIDFromReference extracts the ID from a FHIR reference string
func extractIDFromReference(reference string) string {
	// Reference format: "ResourceType/ID"
	if reference == "" {
		return ""
	}
	// Find the last "/" and return everything after it
	for i := len(reference) - 1; i >= 0; i-- {
		if reference[i] == '/' {
			return reference[i+1:]
		}
	}
	return reference
}
