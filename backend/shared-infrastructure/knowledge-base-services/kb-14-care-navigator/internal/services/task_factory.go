// Package services provides business logic for KB-14 Care Navigator
package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/clients"
	"kb-14-care-navigator/internal/models"
)

// TaskFactory creates tasks from external KB sources
type TaskFactory struct {
	taskService *TaskService
	kb3Client   *clients.KB3Client
	kb9Client   *clients.KB9Client
	kb12Client  *clients.KB12Client
	log         *logrus.Entry
}

// NewTaskFactory creates a new TaskFactory
func NewTaskFactory(
	taskService *TaskService,
	kb3Client *clients.KB3Client,
	kb9Client *clients.KB9Client,
	kb12Client *clients.KB12Client,
	log *logrus.Entry,
) *TaskFactory {
	return &TaskFactory{
		taskService: taskService,
		kb3Client:   kb3Client,
		kb9Client:   kb9Client,
		kb12Client:  kb12Client,
		log:         log.WithField("service", "task-factory"),
	}
}

// CreateFromTemporalAlert creates a task from a KB-3 temporal alert
func (f *TaskFactory) CreateFromTemporalAlert(ctx context.Context, alert *clients.TemporalAlert) (*models.Task, error) {
	// Determine task type based on protocol name and severity
	taskType := f.mapTemporalAlertToTaskType(alert)

	// Map severity to priority
	priority := models.TaskPriorityMedium
	switch strings.ToLower(alert.Severity) {
	case "critical":
		priority = models.TaskPriorityCritical
	case "high", "major":
		priority = models.TaskPriorityHigh
	case "low", "minor":
		priority = models.TaskPriorityLow
	}

	// Calculate SLA based on how overdue the alert is
	slaMinutes := taskType.GetDefaultSLAMinutes()
	if alert.TimeOverdue > 0 {
		// Already overdue, reduce SLA
		slaMinutes = 60 // 1 hour for overdue alerts
	}

	req := &models.CreateTaskRequest{
		Type:        taskType,
		Priority:    priority,
		Source:      models.TaskSourceKB3,
		SourceID:    alert.AlertID,
		PatientID:   alert.PatientID,
		EncounterID: alert.EncounterID,
		Title:       fmt.Sprintf("[%s] %s - %s", alert.ProtocolName, alert.Action, alert.Severity),
		Description: alert.Description,
		SLAMinutes:  slaMinutes,
		Metadata: map[string]interface{}{
			"protocol_id":    alert.ProtocolID,
			"protocol_name":  alert.ProtocolName,
			"constraint_id":  alert.ConstraintID,
			"alert_severity": alert.Severity,
			"deadline":       alert.Deadline,
			"time_overdue":   alert.TimeOverdue,
			"reference":      alert.Reference,
		},
	}

	task, err := f.taskService.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task from temporal alert: %w", err)
	}

	f.log.WithFields(logrus.Fields{
		"task_id":   task.ID,
		"alert_id":  alert.AlertID,
		"protocol":  alert.ProtocolName,
		"patient":   alert.PatientID,
	}).Info("Task created from KB-3 temporal alert")

	return task, nil
}

// mapTemporalAlertToTaskType maps a temporal alert to the appropriate task type
// based on protocol name patterns and severity
func (f *TaskFactory) mapTemporalAlertToTaskType(alert *clients.TemporalAlert) models.TaskType {
	protocolName := strings.ToLower(alert.ProtocolName)
	severity := strings.ToLower(alert.Severity)

	// Map based on protocol name patterns
	switch {
	case strings.Contains(protocolName, "abnormal result") || strings.Contains(protocolName, "abnormal"):
		return models.TaskTypeAbnormalResult
	case strings.Contains(protocolName, "critical value") || strings.Contains(protocolName, "critical lab"):
		return models.TaskTypeAcuteProtocolDeadline
	case strings.Contains(protocolName, "sepsis") || strings.Contains(protocolName, "acute"):
		return models.TaskTypeAcuteProtocolDeadline
	case strings.Contains(protocolName, "therapeutic") || strings.Contains(protocolName, "drug monitoring"):
		return models.TaskTypeTherapeuticChange
	case strings.Contains(protocolName, "missed appointment") || strings.Contains(protocolName, "no show"):
		return models.TaskTypeMissedAppointment
	case strings.Contains(protocolName, "follow-up") && strings.Contains(protocolName, "discharge"):
		return models.TaskTypeTransitionFollowup
	case strings.Contains(protocolName, "monitoring"):
		return models.TaskTypeMonitoringOverdue
	}

	// Fallback based on severity
	if severity == "critical" {
		return models.TaskTypeAcuteProtocolDeadline
	}

	return models.TaskTypeMonitoringOverdue
}

// mapCareGapToTaskType maps a care gap to the appropriate task type
// based on gap type, category, and measure name
func (f *TaskFactory) mapCareGapToTaskType(gap *clients.CareGap) models.TaskType {
	gapType := strings.ToLower(gap.GapType)
	measureName := strings.ToLower(gap.MeasureName)
	category := strings.ToLower(gap.Category)

	// First check measure name for specific task types
	switch {
	case strings.Contains(measureName, "annual wellness") || strings.Contains(measureName, "wellness visit"):
		return models.TaskTypeAnnualWellness
	case strings.Contains(measureName, "colonoscopy") || strings.Contains(measureName, "screening"):
		return models.TaskTypeScreeningOutreach
	case strings.Contains(measureName, "hba1c") || strings.Contains(measureName, "diabetes"):
		return models.TaskTypeCareGapClosure
	case strings.Contains(measureName, "eye exam") || strings.Contains(measureName, "retinal"):
		return models.TaskTypeCareGapClosure
	case strings.Contains(measureName, "chronic care") || strings.Contains(category, "chronic"):
		return models.TaskTypeChronicCareMgmt
	}

	// Then check gap type
	switch gapType {
	case "screening":
		return models.TaskTypeScreeningOutreach
	case "follow_up":
		return models.TaskTypeTransitionFollowup
	case "monitoring":
		return models.TaskTypeMonitoringOverdue
	case "wellness":
		return models.TaskTypeAnnualWellness
	case "chronic_care":
		return models.TaskTypeChronicCareMgmt
	}

	return models.TaskTypeCareGapClosure
}

// CreateFromProtocolDeadline creates a task from a KB-3 protocol deadline
func (f *TaskFactory) CreateFromProtocolDeadline(ctx context.Context, deadline *clients.ProtocolDeadline) (*models.Task, error) {
	// Map priority
	priority := models.TaskPriorityMedium
	switch deadline.Priority {
	case "critical":
		priority = models.TaskPriorityCritical
	case "high":
		priority = models.TaskPriorityHigh
	case "low":
		priority = models.TaskPriorityLow
	}

	dueDate := deadline.Deadline

	req := &models.CreateTaskRequest{
		Type:        models.TaskTypeAcuteProtocolDeadline,
		Priority:    priority,
		Source:      models.TaskSourceKB3,
		SourceID:    deadline.DeadlineID,
		PatientID:   deadline.PatientID,
		EncounterID: deadline.EncounterID,
		Title:       fmt.Sprintf("[%s] %s - %s", deadline.ProtocolName, deadline.StageName, deadline.ActionName),
		Description: fmt.Sprintf("Protocol stage deadline: %s - %s", deadline.StageName, deadline.ActionName),
		DueDate:     &dueDate,
		SLAMinutes:  deadline.SLAMinutes,
		Metadata: map[string]interface{}{
			"protocol_id":   deadline.ProtocolID,
			"protocol_name": deadline.ProtocolName,
			"stage_id":      deadline.StageID,
			"stage_name":    deadline.StageName,
			"action_id":     deadline.ActionID,
			"action_name":   deadline.ActionName,
		},
	}

	task, err := f.taskService.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task from protocol deadline: %w", err)
	}

	f.log.WithFields(logrus.Fields{
		"task_id":     task.ID,
		"deadline_id": deadline.DeadlineID,
		"protocol":    deadline.ProtocolName,
		"patient":     deadline.PatientID,
	}).Info("Task created from KB-3 protocol deadline")

	return task, nil
}

// CreateFromCareGap creates a task from a KB-9 care gap
func (f *TaskFactory) CreateFromCareGap(ctx context.Context, gap *clients.CareGap) (*models.Task, error) {
	// Determine task type based on gap type and measure name
	taskType := f.mapCareGapToTaskType(gap)

	// Map priority
	priority := models.TaskPriorityMedium
	switch gap.Priority {
	case "high":
		priority = models.TaskPriorityHigh
	case "low":
		priority = models.TaskPriorityLow
	}

	// Build action items from interventions
	var actions []models.TaskAction
	for i, intervention := range gap.Interventions {
		action := models.TaskAction{
			ActionID:    fmt.Sprintf("action-%d", i+1),
			Type:        intervention.Type,
			Description: intervention.Description,
			Required:    true,
			Completed:   false,
		}
		actions = append(actions, action)
	}

	req := &models.CreateTaskRequest{
		Type:        taskType,
		Priority:    priority,
		Source:      models.TaskSourceKB9,
		SourceID:    gap.GapID,
		PatientID:   gap.PatientID,
		Title:       fmt.Sprintf("[Care Gap] %s - %s", gap.MeasureName, gap.Category),
		Description: gap.Description,
		DueDate:     gap.DueDate,
		Actions:     actions,
		Metadata: map[string]interface{}{
			"measure_id":      gap.MeasureID,
			"measure_name":    gap.MeasureName,
			"gap_category":    gap.Category,
			"gap_type":        gap.GapType,
			"rationale":       gap.Rationale,
			"evidence_source": gap.EvidenceSource,
			"detected_date":   gap.DetectedDate,
		},
	}

	task, err := f.taskService.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task from care gap: %w", err)
	}

	// Link the task back to KB-9
	if f.kb9Client.IsEnabled() {
		_ = f.kb9Client.LinkCareGapToTask(ctx, gap.GapID, task.ID.String())
	}

	f.log.WithFields(logrus.Fields{
		"task_id":  task.ID,
		"gap_id":   gap.GapID,
		"measure":  gap.MeasureName,
		"patient":  gap.PatientID,
	}).Info("Task created from KB-9 care gap")

	return task, nil
}

// CreateFromCarePlanActivity creates a task from a KB-12 care plan activity
func (f *TaskFactory) CreateFromCarePlanActivity(ctx context.Context, planID string, activity *clients.Activity) (*models.Task, error) {
	// Determine task type based on activity type
	taskType := models.TaskTypeCarePlanReview
	switch activity.Type {
	case "medication":
		taskType = models.TaskTypeMedicationRefill
	case "lab":
		taskType = models.TaskTypeMonitoringOverdue
	case "procedure":
		taskType = models.TaskTypeCarePlanReview
	case "education":
		taskType = models.TaskTypeCareGapClosure
	case "referral":
		taskType = models.TaskTypeReferralProcessing
	case "follow_up":
		taskType = models.TaskTypeTransitionFollowup
	}

	// Map priority
	priority := models.TaskPriorityMedium
	switch activity.Priority {
	case "high":
		priority = models.TaskPriorityHigh
	case "low":
		priority = models.TaskPriorityLow
	}

	// Get patient from care plan - fallback to activity if KB12 unavailable
	var patientID string
	var carePlanTitle string

	// Try to fetch from KB12, but fallback to activity data if unavailable
	if f.kb12Client != nil {
		carePlan, err := f.kb12Client.GetCarePlan(ctx, planID)
		if err == nil && carePlan != nil {
			patientID = carePlan.PatientID
			carePlanTitle = carePlan.Title
		}
	}

	// Fallback to activity data if KB12 lookup failed
	if patientID == "" {
		patientID = activity.PatientID
	}
	if patientID == "" {
		return nil, fmt.Errorf("patient ID required but not found in care plan or activity")
	}
	if carePlanTitle == "" {
		carePlanTitle = fmt.Sprintf("Care Plan %s", planID)
	}

	var dueDate *time.Time
	if activity.DueDate != nil {
		dueDate = activity.DueDate
	} else if activity.ScheduledDate != nil {
		dueDate = activity.ScheduledDate
	}

	// Assign to specified role if present
	var teamID *uuid.UUID
	assignedRole := activity.AssignedRole
	if assignedRole == "" {
		assignedRole = taskType.GetDefaultRole()
	}

	req := &models.CreateTaskRequest{
		Type:        taskType,
		Priority:    priority,
		Source:      models.TaskSourceKB12,
		SourceID:    activity.ActivityID,
		PatientID:   patientID,
		Title:       fmt.Sprintf("[Care Plan] %s", activity.Title),
		Description: activity.Description,
		DueDate:     dueDate,
		TeamID:      teamID,
		AssignedRole: assignedRole,
		Metadata: map[string]interface{}{
			"care_plan_id":   planID,
			"care_plan_title": carePlanTitle,
			"activity_type":  activity.Type,
			"frequency":      activity.Frequency,
			"order_set_id":   activity.OrderSetID,
		},
	}

	task, err := f.taskService.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task from care plan activity: %w", err)
	}

	// Link the task back to KB-12
	if f.kb12Client.IsEnabled() {
		_ = f.kb12Client.LinkActivityToTask(ctx, activity.ActivityID, task.ID.String())
	}

	f.log.WithFields(logrus.Fields{
		"task_id":      task.ID,
		"activity_id":  activity.ActivityID,
		"care_plan":    planID,
		"patient":      patientID,
	}).Info("Task created from KB-12 care plan activity")

	return task, nil
}

// CreateFromMonitoringOverdue creates a task from a KB-3 monitoring overdue item
func (f *TaskFactory) CreateFromMonitoringOverdue(ctx context.Context, overdue *clients.MonitoringOverdue) (*models.Task, error) {
	// Map priority based on severity
	priority := models.TaskPriorityMedium
	switch overdue.Severity {
	case "critical":
		priority = models.TaskPriorityCritical
	case "high":
		priority = models.TaskPriorityHigh
	case "low":
		priority = models.TaskPriorityLow
	}

	// Calculate SLA - urgent for very overdue items
	slaMinutes := 1440 // 24 hours default
	if overdue.DaysOverdue > 7 {
		slaMinutes = 60 // 1 hour for very overdue
	} else if overdue.DaysOverdue > 3 {
		slaMinutes = 240 // 4 hours
	}

	dueDate := overdue.DueDate

	req := &models.CreateTaskRequest{
		Type:       models.TaskTypeMonitoringOverdue,
		Priority:   priority,
		Source:     models.TaskSourceKB3,
		SourceID:   overdue.OverdueID,
		PatientID:  overdue.PatientID,
		Title:      fmt.Sprintf("[%s] %s - %d days overdue", overdue.ProtocolName, overdue.MonitoringType, overdue.DaysOverdue),
		Description: overdue.Description,
		DueDate:    &dueDate,
		SLAMinutes: slaMinutes,
		Metadata: map[string]interface{}{
			"protocol_id":     overdue.ProtocolID,
			"protocol_name":   overdue.ProtocolName,
			"monitoring_type": overdue.MonitoringType,
			"days_overdue":    overdue.DaysOverdue,
			"last_performed":  overdue.LastPerformed,
		},
	}

	task, err := f.taskService.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task from monitoring overdue: %w", err)
	}

	f.log.WithFields(logrus.Fields{
		"task_id":    task.ID,
		"overdue_id": overdue.OverdueID,
		"protocol":   overdue.ProtocolName,
		"patient":    overdue.PatientID,
	}).Info("Task created from KB-3 monitoring overdue")

	return task, nil
}

// CreateFromCareGapModel creates a task from a CareGap model (wrapper for API handler use)
func (f *TaskFactory) CreateFromCareGapModel(ctx context.Context, gap *models.CareGap) (*models.Task, error) {
	// Convert model to client type
	clientGap := &clients.CareGap{
		GapID:       gap.GapID,
		PatientID:   gap.PatientID,
		GapType:     gap.GapType,
		Category:    gap.GapCategory,
		MeasureID:   "",
		MeasureName: gap.Title,
		Description: gap.Description,
		Priority:    gap.Priority,
		DueDate:     gap.DueDate,
		Rationale:   "",
	}

	// Convert interventions
	for _, intervention := range gap.Interventions {
		clientGap.Interventions = append(clientGap.Interventions, clients.Intervention{
			Type:        intervention.Type,
			Description: intervention.Description,
			Code:        intervention.Code,
			CodeSystem:  intervention.CodeSystem,
		})
	}

	return f.CreateFromCareGap(ctx, clientGap)
}

// CreateFromTemporalAlertModel creates a task from a TemporalAlert model (wrapper for API handler use)
func (f *TaskFactory) CreateFromTemporalAlertModel(ctx context.Context, alert *models.TemporalAlert) (*models.Task, error) {
	// Convert model to client type
	var deadline time.Time
	if alert.Deadline != nil {
		deadline = *alert.Deadline
	}

	var alertTime time.Time
	if alert.AlertTime != nil {
		alertTime = *alert.AlertTime
	}

	clientAlert := &clients.TemporalAlert{
		AlertID:      alert.AlertID,
		PatientID:    alert.PatientID,
		EncounterID:  alert.EncounterID,
		ProtocolID:   alert.ProtocolID,
		ProtocolName: alert.ProtocolName,
		ConstraintID: alert.ConstraintID,
		Action:       alert.Action,
		Severity:     alert.Severity,
		Status:       alert.Status,
		Deadline:     deadline,
		TimeOverdue:  alert.TimeOverdue,
		AlertTime:    alertTime,
		Description:  alert.Description,
		Reference:    alert.Reference,
		Acknowledged: alert.Acknowledged,
	}

	return f.CreateFromTemporalAlert(ctx, clientAlert)
}

// CreateFromCarePlanActivityModel creates a task from a CarePlanActivity model (wrapper for API handler use)
func (f *TaskFactory) CreateFromCarePlanActivityModel(ctx context.Context, activity *models.CarePlanActivity) (*models.Task, error) {
	// Determine task type based on activity type
	taskType := models.TaskTypeCarePlanReview
	switch activity.Type {
	case "medication":
		taskType = models.TaskTypeMedicationRefill
	case "lab", "observation":
		taskType = models.TaskTypeMonitoringOverdue
	case "procedure":
		taskType = models.TaskTypeCarePlanReview
	case "education":
		taskType = models.TaskTypeCareGapClosure
	case "referral":
		taskType = models.TaskTypeReferralProcessing
	case "follow_up", "appointment":
		taskType = models.TaskTypeTransitionFollowup
	}

	// Map priority
	priority := models.TaskPriorityMedium
	switch activity.Priority {
	case "stat":
		priority = models.TaskPriorityCritical
	case "asap", "urgent":
		priority = models.TaskPriorityHigh
	case "routine":
		priority = models.TaskPriorityLow
	}

	req := &models.CreateTaskRequest{
		Type:        taskType,
		Priority:    priority,
		Source:      models.TaskSourceKB12,
		SourceID:    activity.ActivityID,
		PatientID:   activity.PatientID,
		EncounterID: activity.EncounterID,
		Title:       fmt.Sprintf("[Care Plan] %s", activity.Title),
		Description: activity.Description,
		DueDate:     activity.DueDate,
		Metadata: map[string]interface{}{
			"care_plan_id":   activity.CarePlanID,
			"activity_type":  activity.Type,
			"activity_status": activity.Status,
		},
	}

	task, err := f.taskService.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task from care plan activity: %w", err)
	}

	f.log.WithFields(logrus.Fields{
		"task_id":     task.ID,
		"activity_id": activity.ActivityID,
		"care_plan":   activity.CarePlanID,
		"patient":     activity.PatientID,
	}).Info("Task created from KB-12 care plan activity (model)")

	return task, nil
}

// CreateFromProtocolStep creates a task from a ProtocolStep model
func (f *TaskFactory) CreateFromProtocolStep(ctx context.Context, step *models.ProtocolStep) (*models.Task, error) {
	// Determine task type based on step type
	taskType := models.TaskTypeAcuteProtocolDeadline
	switch step.StepType {
	case "medication":
		taskType = models.TaskTypeMedicationReview
	case "lab":
		taskType = models.TaskTypeCriticalLabReview
	case "procedure":
		taskType = models.TaskTypeCarePlanReview
	case "action", "decision":
		taskType = models.TaskTypeAcuteProtocolDeadline
	}

	// Map priority
	priority := models.TaskPriorityMedium
	switch step.Priority {
	case "critical", "stat":
		priority = models.TaskPriorityCritical
	case "high", "asap":
		priority = models.TaskPriorityHigh
	case "low":
		priority = models.TaskPriorityLow
	}

	slaMinutes := step.SLAMinutes
	if slaMinutes == 0 {
		slaMinutes = taskType.GetDefaultSLAMinutes()
	}

	// Build actions from protocol step actions
	var actions []models.TaskAction
	for _, stepAction := range step.Actions {
		action := models.TaskAction{
			ActionID:    stepAction.ActionID,
			Type:        stepAction.Type,
			Description: stepAction.Description,
			Required:    stepAction.Required,
			Completed:   false,
		}
		actions = append(actions, action)
	}

	req := &models.CreateTaskRequest{
		Type:        taskType,
		Priority:    priority,
		Source:      models.TaskSourceKB12,
		SourceID:    step.StepID,
		PatientID:   step.PatientID,
		EncounterID: step.EncounterID,
		Title:       fmt.Sprintf("[%s] Step %d: %s", step.ProtocolName, step.StepNumber, step.Title),
		Description: step.Description,
		DueDate:     step.DueDate,
		SLAMinutes:  slaMinutes,
		Actions:     actions,
		Metadata: map[string]interface{}{
			"protocol_id":   step.ProtocolID,
			"protocol_name": step.ProtocolName,
			"step_type":     step.StepType,
			"step_number":   step.StepNumber,
			"step_status":   step.Status,
		},
	}

	task, err := f.taskService.Create(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to create task from protocol step: %w", err)
	}

	f.log.WithFields(logrus.Fields{
		"task_id":  task.ID,
		"step_id":  step.StepID,
		"protocol": step.ProtocolName,
		"patient":  step.PatientID,
	}).Info("Task created from KB-12 protocol step")

	return task, nil
}

// SyncFromKB3 syncs overdue alerts from KB-3 and creates tasks
func (f *TaskFactory) SyncFromKB3(ctx context.Context) (int, error) {
	if !f.kb3Client.IsEnabled() {
		return 0, nil
	}

	// Get overdue alerts
	alerts, err := f.kb3Client.GetOverdueAlerts(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get KB-3 overdue alerts: %w", err)
	}

	created := 0
	for _, alert := range alerts {
		// Check if task already exists for this alert
		existing, _ := f.taskService.taskRepo.FindBySource(ctx, models.TaskSourceKB3, alert.AlertID)
		if len(existing) > 0 {
			continue // Skip if already exists
		}

		if _, err := f.CreateFromTemporalAlert(ctx, &alert); err != nil {
			f.log.WithError(err).WithField("alert_id", alert.AlertID).Warn("Failed to create task from alert")
			continue
		}
		created++
	}

	f.log.WithField("created", created).Info("KB-3 sync completed")
	return created, nil
}

// SyncFromKB9 syncs care gaps from KB-9 and creates tasks
func (f *TaskFactory) SyncFromKB9(ctx context.Context) (int, error) {
	if !f.kb9Client.IsEnabled() {
		return 0, nil
	}

	// Get high priority care gaps
	gaps, err := f.kb9Client.GetHighPriorityCareGaps(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get KB-9 care gaps: %w", err)
	}

	created := 0
	for _, gap := range gaps {
		// Check if task already exists for this gap
		existing, _ := f.taskService.taskRepo.FindBySource(ctx, models.TaskSourceKB9, gap.GapID)
		if len(existing) > 0 {
			continue
		}

		if _, err := f.CreateFromCareGap(ctx, &gap); err != nil {
			f.log.WithError(err).WithField("gap_id", gap.GapID).Warn("Failed to create task from care gap")
			continue
		}
		created++
	}

	f.log.WithField("created", created).Info("KB-9 sync completed")
	return created, nil
}

// SyncFromKB12 syncs overdue activities from KB-12 and creates tasks
func (f *TaskFactory) SyncFromKB12(ctx context.Context) (int, error) {
	if !f.kb12Client.IsEnabled() {
		return 0, nil
	}

	// Get overdue activities
	activities, err := f.kb12Client.GetOverdueActivities(ctx)
	if err != nil {
		return 0, fmt.Errorf("failed to get KB-12 overdue activities: %w", err)
	}

	created := 0
	for _, activity := range activities {
		// Check if task already exists for this activity
		existing, _ := f.taskService.taskRepo.FindBySource(ctx, models.TaskSourceKB12, activity.ActivityID)
		if len(existing) > 0 {
			continue
		}

		// Get the care plan to get patient info
		// Note: We'd need to add a method to get activity with care plan context
		// For now, skip if we can't get the care plan
		if activity.OrderSetID == "" {
			continue
		}

		// Create with minimal info - in practice, we'd need care plan context
		f.log.WithField("activity_id", activity.ActivityID).Debug("Would create task from activity")
		// created++
	}

	f.log.WithField("created", created).Info("KB-12 sync completed")
	return created, nil
}

// SyncAll synchronizes from all KB sources
func (f *TaskFactory) SyncAll(ctx context.Context) (map[string]int, error) {
	results := map[string]int{}

	kb3Count, err := f.SyncFromKB3(ctx)
	if err != nil {
		f.log.WithError(err).Warn("KB-3 sync failed")
	}
	results["kb3"] = kb3Count

	kb9Count, err := f.SyncFromKB9(ctx)
	if err != nil {
		f.log.WithError(err).Warn("KB-9 sync failed")
	}
	results["kb9"] = kb9Count

	kb12Count, err := f.SyncFromKB12(ctx)
	if err != nil {
		f.log.WithError(err).Warn("KB-12 sync failed")
	}
	results["kb12"] = kb12Count

	return results, nil
}
