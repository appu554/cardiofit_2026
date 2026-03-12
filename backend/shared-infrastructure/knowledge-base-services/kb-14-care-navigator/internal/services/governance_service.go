// Package services provides the governance service for KB-14 Care Navigator
package services

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/models"
)

// =============================================================================
// GOVERNANCE SERVICE
// Central coordinator for audit logging, governance events, and compliance
// =============================================================================

// GovernanceService handles audit logging and governance events
type GovernanceService struct {
	auditRepo        *database.AuditRepository
	governanceRepo   *database.GovernanceRepository
	reasonCodeRepo   *database.ReasonCodeRepository
	intelligenceRepo *database.IntelligenceRepository
	log              *logrus.Entry
}

// NewGovernanceService creates a new GovernanceService
func NewGovernanceService(
	auditRepo *database.AuditRepository,
	governanceRepo *database.GovernanceRepository,
	reasonCodeRepo *database.ReasonCodeRepository,
	intelligenceRepo *database.IntelligenceRepository,
	log *logrus.Entry,
) *GovernanceService {
	return &GovernanceService{
		auditRepo:        auditRepo,
		governanceRepo:   governanceRepo,
		reasonCodeRepo:   reasonCodeRepo,
		intelligenceRepo: intelligenceRepo,
		log:              log.WithField("service", "governance"),
	}
}

// =============================================================================
// AUDIT EVENT PUBLISHING
// =============================================================================

// AuditContext contains context information for audit events
type AuditContext struct {
	ActorID   *uuid.UUID
	ActorType models.ActorType
	ActorName string
	ActorRole string
	IPAddress string
	UserAgent string
	SessionID string
}

// SystemAuditContext returns an audit context for system-initiated actions
func SystemAuditContext() *AuditContext {
	return &AuditContext{
		ActorType: models.ActorTypeSystem,
		ActorName: "KB-14 System",
	}
}

// PublishTaskCreated publishes a task created audit event
func (s *GovernanceService) PublishTaskCreated(ctx context.Context, task *models.Task, auditCtx *AuditContext, sourceEventID string) error {
	return s.publishAuditEvent(ctx, &models.TaskAuditLog{
		TaskID:        task.ID,
		TaskNumber:    task.TaskID,
		EventType:     models.AuditEventCreated,
		NewStatus:     &task.Status,
		ActorID:       auditCtx.ActorID,
		ActorType:     auditCtx.ActorType,
		ActorName:     auditCtx.ActorName,
		ActorRole:     auditCtx.ActorRole,
		PatientID:     task.PatientID,
		EncounterID:   task.EncounterID,
		SourceService: string(task.Source),
		SourceEventID: sourceEventID,
		NewValue: models.JSONMap{
			"type":        task.Type,
			"priority":    task.Priority,
			"title":       task.Title,
			"sla_minutes": task.SLAMinutes,
		},
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
		SessionID: auditCtx.SessionID,
	})
}

// PublishTaskAssigned publishes a task assigned audit event
func (s *GovernanceService) PublishTaskAssigned(ctx context.Context, task *models.Task, previousAssignee *uuid.UUID, auditCtx *AuditContext) error {
	eventType := models.AuditEventAssigned
	if previousAssignee != nil {
		eventType = models.AuditEventReassigned
	}

	previousValue := models.JSONMap{}
	if previousAssignee != nil {
		previousValue["assigned_to"] = previousAssignee.String()
	}

	return s.publishAuditEvent(ctx, &models.TaskAuditLog{
		TaskID:      task.ID,
		TaskNumber:  task.TaskID,
		EventType:   eventType,
		ActorID:     auditCtx.ActorID,
		ActorType:   auditCtx.ActorType,
		ActorName:   auditCtx.ActorName,
		ActorRole:   auditCtx.ActorRole,
		PatientID:   task.PatientID,
		EncounterID: task.EncounterID,
		PreviousValue: previousValue,
		NewValue: models.JSONMap{
			"assigned_to":   task.AssignedTo,
			"assigned_role": task.AssignedRole,
		},
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
		SessionID: auditCtx.SessionID,
	})
}

// PublishStatusChange publishes a status change audit event
func (s *GovernanceService) PublishStatusChange(ctx context.Context, task *models.Task, previousStatus models.TaskStatus, auditCtx *AuditContext, reasonCode, reasonText string) error {
	eventType := s.getEventTypeForStatusChange(task.Status)

	return s.publishAuditEvent(ctx, &models.TaskAuditLog{
		TaskID:         task.ID,
		TaskNumber:     task.TaskID,
		EventType:      eventType,
		PreviousStatus: &previousStatus,
		NewStatus:      &task.Status,
		ActorID:        auditCtx.ActorID,
		ActorType:      auditCtx.ActorType,
		ActorName:      auditCtx.ActorName,
		ActorRole:      auditCtx.ActorRole,
		PatientID:      task.PatientID,
		EncounterID:    task.EncounterID,
		ReasonCode:     reasonCode,
		ReasonText:     reasonText,
		IPAddress:      auditCtx.IPAddress,
		UserAgent:      auditCtx.UserAgent,
		SessionID:      auditCtx.SessionID,
	})
}

// PublishEscalation publishes an escalation audit event
func (s *GovernanceService) PublishEscalation(ctx context.Context, task *models.Task, escalation *models.Escalation, auditCtx *AuditContext) error {
	return s.publishAuditEvent(ctx, &models.TaskAuditLog{
		TaskID:      task.ID,
		TaskNumber:  task.TaskID,
		EventType:   models.AuditEventEscalated,
		ActorID:     auditCtx.ActorID,
		ActorType:   auditCtx.ActorType,
		ActorName:   auditCtx.ActorName,
		ActorRole:   auditCtx.ActorRole,
		PatientID:   task.PatientID,
		EncounterID: task.EncounterID,
		ReasonCode:  "ESCALATION_" + string(escalation.Level),
		ReasonText:  escalation.Reason,
		NewValue: models.JSONMap{
			"escalation_level":   escalation.Level,
			"escalated_to":       escalation.EscalatedTo,
			"escalated_to_role":  escalation.EscalatedToRole,
			"sla_elapsed_percent": escalation.SLAElapsedPercent,
		},
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
		SessionID: auditCtx.SessionID,
	})
}

// PublishNoteAdded publishes a note added audit event
func (s *GovernanceService) PublishNoteAdded(ctx context.Context, task *models.Task, note *models.TaskNote, auditCtx *AuditContext) error {
	return s.publishAuditEvent(ctx, &models.TaskAuditLog{
		TaskID:      task.ID,
		TaskNumber:  task.TaskID,
		EventType:   models.AuditEventNoteAdded,
		ActorID:     auditCtx.ActorID,
		ActorType:   auditCtx.ActorType,
		ActorName:   auditCtx.ActorName,
		ActorRole:   auditCtx.ActorRole,
		PatientID:   task.PatientID,
		EncounterID: task.EncounterID,
		NewValue: models.JSONMap{
			"note_id": note.NoteID,
			"author":  note.Author,
		},
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
		SessionID: auditCtx.SessionID,
	})
}

// PublishPriorityChange publishes a priority change audit event
func (s *GovernanceService) PublishPriorityChange(ctx context.Context, task *models.Task, previousPriority models.TaskPriority, auditCtx *AuditContext, reason string) error {
	return s.publishAuditEvent(ctx, &models.TaskAuditLog{
		TaskID:      task.ID,
		TaskNumber:  task.TaskID,
		EventType:   models.AuditEventPriorityChanged,
		ActorID:     auditCtx.ActorID,
		ActorType:   auditCtx.ActorType,
		ActorName:   auditCtx.ActorName,
		ActorRole:   auditCtx.ActorRole,
		PatientID:   task.PatientID,
		EncounterID: task.EncounterID,
		ReasonText:  reason,
		PreviousValue: models.JSONMap{
			"priority": previousPriority,
		},
		NewValue: models.JSONMap{
			"priority": task.Priority,
		},
		IPAddress: auditCtx.IPAddress,
		UserAgent: auditCtx.UserAgent,
		SessionID: auditCtx.SessionID,
	})
}

// publishAuditEvent is the internal method that creates audit log entries
func (s *GovernanceService) publishAuditEvent(ctx context.Context, log *models.TaskAuditLog) error {
	if err := s.auditRepo.Create(ctx, log); err != nil {
		s.log.WithError(err).WithFields(logrus.Fields{
			"task_id":    log.TaskID,
			"event_type": log.EventType,
		}).Error("Failed to publish audit event")
		return err
	}

	s.log.WithFields(logrus.Fields{
		"task_id":    log.TaskID,
		"event_type": log.EventType,
		"actor_type": log.ActorType,
	}).Debug("Audit event published")

	return nil
}

// getEventTypeForStatusChange maps status to audit event type
func (s *GovernanceService) getEventTypeForStatusChange(status models.TaskStatus) models.AuditEventType {
	switch status {
	case models.TaskStatusInProgress:
		return models.AuditEventStarted
	case models.TaskStatusCompleted:
		return models.AuditEventCompleted
	case models.TaskStatusVerified:
		return models.AuditEventVerified
	case models.TaskStatusDeclined:
		return models.AuditEventDeclined
	case models.TaskStatusCancelled:
		return models.AuditEventCancelled
	case models.TaskStatusEscalated:
		return models.AuditEventEscalated
	default:
		return models.AuditEventType(status)
	}
}

// =============================================================================
// GOVERNANCE EVENTS
// =============================================================================

// PublishSLAWarning publishes an SLA warning governance event
func (s *GovernanceService) PublishSLAWarning(ctx context.Context, task *models.Task, slaElapsed float64) error {
	event := &models.GovernanceEvent{
		EventType:   models.GovernanceEventSLABreach,
		Severity:    models.GovernanceSeverityWarning,
		TaskID:      &task.ID,
		PatientID:   task.PatientID,
		Title:       fmt.Sprintf("SLA Warning: Task %s approaching deadline", task.TaskID),
		Description: fmt.Sprintf("Task has consumed %.1f%% of allocated SLA time", slaElapsed*100),
		RiskScore:   floatPtr(slaElapsed * 50), // Warning = up to 50 risk
		RequiresAction: true,
		TriggeredBy: "SYSTEM",
		Evidence: models.JSONMap{
			"sla_elapsed_percent": slaElapsed * 100,
			"sla_minutes":         task.SLAMinutes,
			"task_type":           task.Type,
			"priority":            task.Priority,
		},
	}

	return s.governanceRepo.Create(ctx, event)
}

// PublishSLABreach publishes an SLA breach governance event
func (s *GovernanceService) PublishSLABreach(ctx context.Context, task *models.Task, slaElapsed float64) error {
	severity := models.GovernanceSeverityCritical
	if slaElapsed > 1.25 {
		severity = models.GovernanceSeverityAlert
	}

	deadline := time.Now().UTC().Add(1 * time.Hour)

	event := &models.GovernanceEvent{
		EventType:      models.GovernanceEventSLABreach,
		Severity:       severity,
		TaskID:         &task.ID,
		PatientID:      task.PatientID,
		Title:          fmt.Sprintf("SLA Breach: Task %s overdue", task.TaskID),
		Description:    fmt.Sprintf("Task is %.1f%% past SLA deadline", (slaElapsed-1)*100),
		RiskScore:      floatPtr(50 + (slaElapsed-1)*50), // Breach = 50-100 risk
		RequiresAction: true,
		ActionDeadline: &deadline,
		TriggeredBy:    "SYSTEM",
		Evidence: models.JSONMap{
			"sla_elapsed_percent": slaElapsed * 100,
			"sla_minutes":         task.SLAMinutes,
			"task_type":           task.Type,
			"priority":            task.Priority,
			"assigned_to":         task.AssignedTo,
		},
	}

	return s.governanceRepo.Create(ctx, event)
}

// PublishPolicyViolation publishes a policy violation governance event
func (s *GovernanceService) PublishPolicyViolation(ctx context.Context, taskID *uuid.UUID, patientID, title, description string, riskScore float64) error {
	event := &models.GovernanceEvent{
		EventType:      models.GovernanceEventPolicyViolation,
		Severity:       models.GovernanceSeverityCritical,
		TaskID:         taskID,
		PatientID:      patientID,
		Title:          title,
		Description:    description,
		RiskScore:      &riskScore,
		RequiresAction: true,
		TriggeredBy:    "SYSTEM",
	}

	return s.governanceRepo.Create(ctx, event)
}

// PublishIntelligenceGap publishes an intelligence gap governance event
func (s *GovernanceService) PublishIntelligenceGap(ctx context.Context, patientID string, orphanedCount int) error {
	event := &models.GovernanceEvent{
		EventType:   models.GovernanceEventIntelligenceGap,
		Severity:    models.GovernanceSeverityWarning,
		PatientID:   patientID,
		Title:       fmt.Sprintf("Intelligence Gap: %d unprocessed items for patient", orphanedCount),
		Description: "Clinical intelligence has not been converted to tasks or dispositioned",
		RiskScore:   floatPtr(float64(orphanedCount) * 10),
		RequiresAction: true,
		TriggeredBy: "SYSTEM",
		Evidence: models.JSONMap{
			"orphaned_count": orphanedCount,
		},
	}

	return s.governanceRepo.Create(ctx, event)
}

// =============================================================================
// INTELLIGENCE TRACKING
// =============================================================================

// TrackIntelligence tracks incoming clinical intelligence
func (s *GovernanceService) TrackIntelligence(ctx context.Context, source string, sourceID string, sourceType models.IntelligenceSourceType, patientID string, snapshot map[string]interface{}) (*models.IntelligenceTracking, error) {
	// Check if already tracked
	existing, err := s.intelligenceRepo.GetBySourceID(ctx, source, sourceID)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return existing, nil // Already tracked
	}

	intel := &models.IntelligenceTracking{
		SourceService:        source,
		SourceID:             sourceID,
		SourceType:           sourceType,
		PatientID:            patientID,
		Status:               models.IntelligenceStatusReceived,
		IntelligenceSnapshot: snapshot,
	}

	if err := s.intelligenceRepo.Create(ctx, intel); err != nil {
		return nil, err
	}

	s.log.WithFields(logrus.Fields{
		"source_service": source,
		"source_id":      sourceID,
		"source_type":    sourceType,
		"patient_id":     patientID,
	}).Debug("Intelligence tracked")

	return intel, nil
}

// LinkIntelligenceToTask links intelligence to a created task
func (s *GovernanceService) LinkIntelligenceToTask(ctx context.Context, intelligenceID, taskID uuid.UUID) error {
	return s.intelligenceRepo.UpdateStatus(ctx, intelligenceID, models.IntelligenceStatusTaskCreated, &taskID)
}

// DispositionIntelligence records a disposition for intelligence that won't become a task
func (s *GovernanceService) DispositionIntelligence(ctx context.Context, intelligenceID uuid.UUID, code, reason string, by uuid.UUID) error {
	// Validate reason code
	valid, reasonCode, err := s.reasonCodeRepo.ValidateCode(ctx, code)
	if err != nil {
		return err
	}
	if !valid {
		return fmt.Errorf("invalid or inactive reason code: %s", code)
	}

	// Check if justification is required
	if reasonCode != nil && reasonCode.RequiresJustification && reason == "" {
		return fmt.Errorf("reason code %s requires justification", code)
	}

	return s.intelligenceRepo.Disposition(ctx, intelligenceID, code, reason, by)
}

// CheckOrphanedIntelligence checks for intelligence that hasn't been processed
func (s *GovernanceService) CheckOrphanedIntelligence(ctx context.Context, timeoutMinutes int) error {
	orphaned, err := s.intelligenceRepo.FindOrphanedIntelligence(ctx, timeoutMinutes)
	if err != nil {
		return err
	}

	// Group by patient
	patientCounts := make(map[string]int)
	for _, intel := range orphaned {
		patientCounts[intel.PatientID]++
	}

	// Publish governance events for patients with orphaned intelligence
	for patientID, count := range patientCounts {
		if err := s.PublishIntelligenceGap(ctx, patientID, count); err != nil {
			s.log.WithError(err).WithField("patient_id", patientID).Warn("Failed to publish intelligence gap event")
		}
	}

	if len(orphaned) > 0 {
		s.log.WithField("orphaned_count", len(orphaned)).Warn("Found orphaned intelligence")
	}

	return nil
}

// =============================================================================
// REASON CODE VALIDATION
// =============================================================================

// ValidateReasonCode validates a reason code and returns requirements
func (s *GovernanceService) ValidateReasonCode(ctx context.Context, code string) (bool, bool, bool, error) {
	valid, reasonCode, err := s.reasonCodeRepo.ValidateCode(ctx, code)
	if err != nil {
		return false, false, false, err
	}
	if !valid || reasonCode == nil {
		return false, false, false, nil
	}

	return true, reasonCode.RequiresJustification, reasonCode.RequiresSupervisorApproval, nil
}

// GetReasonCodesByCategory retrieves reason codes by category
func (s *GovernanceService) GetReasonCodesByCategory(ctx context.Context, category models.ReasonCodeCategory) ([]models.ReasonCode, error) {
	return s.reasonCodeRepo.GetByCategory(ctx, category)
}

// GetAllReasonCodes retrieves all active reason codes
func (s *GovernanceService) GetAllReasonCodes(ctx context.Context) ([]models.ReasonCode, error) {
	return s.reasonCodeRepo.GetAll(ctx)
}

// =============================================================================
// AUDIT QUERIES
// =============================================================================

// GetTaskAuditTrail retrieves the complete audit trail for a task
func (s *GovernanceService) GetTaskAuditTrail(ctx context.Context, taskID uuid.UUID) ([]models.TaskAuditLog, error) {
	return s.auditRepo.FindByTask(ctx, taskID)
}

// GetPatientAuditTrail retrieves audit trail for a patient
func (s *GovernanceService) GetPatientAuditTrail(ctx context.Context, patientID string, limit int) ([]models.TaskAuditLog, error) {
	return s.auditRepo.FindByPatient(ctx, patientID, limit)
}

// GetActorAuditTrail retrieves audit trail for an actor
func (s *GovernanceService) GetActorAuditTrail(ctx context.Context, actorID uuid.UUID, limit int) ([]models.TaskAuditLog, error) {
	return s.auditRepo.FindByActor(ctx, actorID, limit)
}

// QueryAuditLogs queries audit logs based on parameters
func (s *GovernanceService) QueryAuditLogs(ctx context.Context, query *models.AuditLogQuery) ([]models.TaskAuditLog, int64, error) {
	return s.auditRepo.FindByQuery(ctx, query)
}

// GetAuditSummary retrieves summary statistics for a task's audit trail
func (s *GovernanceService) GetAuditSummary(ctx context.Context, taskID uuid.UUID) (*models.AuditSummary, error) {
	return s.auditRepo.GetAuditSummary(ctx, taskID)
}

// VerifyAuditIntegrity verifies the hash chain integrity for a task
func (s *GovernanceService) VerifyAuditIntegrity(ctx context.Context, taskID uuid.UUID) (bool, []string, error) {
	return s.auditRepo.VerifyHashChain(ctx, taskID)
}

// =============================================================================
// GOVERNANCE QUERIES
// =============================================================================

// GetUnresolvedGovernanceEvents retrieves all unresolved governance events
func (s *GovernanceService) GetUnresolvedGovernanceEvents(ctx context.Context) ([]models.GovernanceEvent, error) {
	return s.governanceRepo.FindUnresolved(ctx)
}

// GetEventsRequiringAction retrieves governance events requiring action
func (s *GovernanceService) GetEventsRequiringAction(ctx context.Context) ([]models.GovernanceEvent, error) {
	return s.governanceRepo.FindRequiringAction(ctx)
}

// QueryGovernanceEvents queries governance events based on parameters
func (s *GovernanceService) QueryGovernanceEvents(ctx context.Context, query *models.GovernanceEventQuery) ([]models.GovernanceEvent, int64, error) {
	return s.governanceRepo.FindByQuery(ctx, query)
}

// ResolveGovernanceEvent resolves a governance event
func (s *GovernanceService) ResolveGovernanceEvent(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID, notes string) error {
	return s.governanceRepo.Resolve(ctx, id, resolvedBy, notes)
}

// GetGovernanceDashboard retrieves governance dashboard statistics
func (s *GovernanceService) GetGovernanceDashboard(ctx context.Context, days int) ([]models.GovernanceDashboard, error) {
	return s.governanceRepo.GetDashboard(ctx, days)
}

// GetIntelligenceAccountability retrieves intelligence accountability statistics
func (s *GovernanceService) GetIntelligenceAccountability(ctx context.Context, days int) ([]models.IntelligenceAccountability, error) {
	return s.intelligenceRepo.GetAccountability(ctx, days)
}

// =============================================================================
// COMPLIANCE SCORING
// =============================================================================

// ComplianceScore represents a compliance score with breakdown
type ComplianceScore struct {
	OverallScore     float64            `json:"overall_score"`
	SLACompliance    float64            `json:"sla_compliance"`
	EscalationRate   float64            `json:"escalation_rate"`
	IntelligenceGaps float64            `json:"intelligence_gaps"`
	GovernanceEvents int                `json:"governance_events"`
	RiskLevel        string             `json:"risk_level"`
	Breakdown        map[string]float64 `json:"breakdown"`
}

// CalculateComplianceScore calculates the overall compliance score
func (s *GovernanceService) CalculateComplianceScore(ctx context.Context, days int) (*ComplianceScore, error) {
	// Get governance dashboard data
	dashboard, err := s.governanceRepo.GetDashboard(ctx, days)
	if err != nil {
		return nil, err
	}

	// Get intelligence accountability
	accountability, err := s.intelligenceRepo.GetAccountability(ctx, days)
	if err != nil {
		return nil, err
	}

	// Calculate metrics
	var totalEvents, resolvedEvents, slaBreaches int64
	var avgRisk float64
	for _, d := range dashboard {
		totalEvents += d.EventCount
		resolvedEvents += d.ResolvedCount
		if d.EventType == models.GovernanceEventSLABreach {
			slaBreaches += d.EventCount
		}
		if d.AvgRiskScore != nil {
			avgRisk += *d.AvgRiskScore
		}
	}

	// Calculate intelligence gap score
	var totalIntel, pendingIntel int64
	for _, a := range accountability {
		totalIntel += a.Count
		pendingIntel += a.Pending
	}

	// Calculate scores
	slaCompliance := 100.0
	if totalEvents > 0 {
		slaCompliance = 100.0 - (float64(slaBreaches)/float64(totalEvents))*100
	}

	escalationRate := 0.0
	if totalEvents > 0 {
		escalationRate = float64(resolvedEvents) / float64(totalEvents) * 100
	}

	intelligenceGaps := 100.0
	if totalIntel > 0 {
		intelligenceGaps = 100.0 - (float64(pendingIntel)/float64(totalIntel))*100
	}

	// Overall score (weighted average)
	overallScore := (slaCompliance*0.4 + escalationRate*0.3 + intelligenceGaps*0.3)

	// Determine risk level
	riskLevel := "LOW"
	if overallScore < 70 {
		riskLevel = "HIGH"
	} else if overallScore < 85 {
		riskLevel = "MEDIUM"
	}

	return &ComplianceScore{
		OverallScore:     overallScore,
		SLACompliance:    slaCompliance,
		EscalationRate:   escalationRate,
		IntelligenceGaps: intelligenceGaps,
		GovernanceEvents: int(totalEvents),
		RiskLevel:        riskLevel,
		Breakdown: map[string]float64{
			"sla_weight":          0.4,
			"escalation_weight":   0.3,
			"intelligence_weight": 0.3,
		},
	}, nil
}

// Helper functions
func floatPtr(f float64) *float64 {
	return &f
}
