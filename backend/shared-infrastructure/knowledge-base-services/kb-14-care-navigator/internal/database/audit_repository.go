// Package database provides audit and governance repository operations
package database

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"kb-14-care-navigator/internal/models"
)

// =============================================================================
// AUDIT LOG REPOSITORY
// Append-only repository for immutable audit records
// =============================================================================

// AuditRepository handles audit log database operations
type AuditRepository struct {
	db *gorm.DB
}

// NewAuditRepository creates a new AuditRepository
func NewAuditRepository(db *gorm.DB) *AuditRepository {
	return &AuditRepository{db: db}
}

// Create creates a new audit log entry (append-only)
func (r *AuditRepository) Create(ctx context.Context, log *models.TaskAuditLog) error {
	// Get the previous hash for chain integrity
	var lastLog models.TaskAuditLog
	err := r.db.WithContext(ctx).
		Order("sequence_number DESC").
		First(&lastLog).Error

	if err != nil && err != gorm.ErrRecordNotFound {
		return fmt.Errorf("failed to get last audit log: %w", err)
	}

	// Set previous hash (empty string for genesis record)
	if err == nil {
		log.PreviousHash = lastLog.RecordHash
	} else {
		log.PreviousHash = ""
	}

	// Set event category
	log.EventCategory = log.EventType.GetEventCategory()

	// Set event timestamp if not provided
	if log.EventTimestamp.IsZero() {
		log.EventTimestamp = time.Now().UTC()
	}

	// Truncate timestamp to microsecond precision for consistent hashing
	// PostgreSQL stores timestamps with microsecond precision, so we need to
	// truncate before hashing to ensure consistent hash verification after read
	log.EventTimestamp = log.EventTimestamp.Truncate(time.Microsecond)

	// Calculate record hash
	log.RecordHash = log.CalculateHash()

	// Create the record
	if err := r.db.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// FindByTask retrieves all audit logs for a task
func (r *AuditRepository) FindByTask(ctx context.Context, taskID uuid.UUID) ([]models.TaskAuditLog, error) {
	var logs []models.TaskAuditLog
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("sequence_number ASC").
		Find(&logs).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find audit logs: %w", err)
	}

	return logs, nil
}

// FindByPatient retrieves all audit logs for a patient
func (r *AuditRepository) FindByPatient(ctx context.Context, patientID string, limit int) ([]models.TaskAuditLog, error) {
	var logs []models.TaskAuditLog
	query := r.db.WithContext(ctx).
		Where("patient_id = ?", patientID).
		Order("event_timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to find audit logs: %w", err)
	}

	return logs, nil
}

// FindByActor retrieves all audit logs for an actor
func (r *AuditRepository) FindByActor(ctx context.Context, actorID uuid.UUID, limit int) ([]models.TaskAuditLog, error) {
	var logs []models.TaskAuditLog
	query := r.db.WithContext(ctx).
		Where("actor_id = ?", actorID).
		Order("event_timestamp DESC")

	if limit > 0 {
		query = query.Limit(limit)
	}

	if err := query.Find(&logs).Error; err != nil {
		return nil, fmt.Errorf("failed to find audit logs: %w", err)
	}

	return logs, nil
}

// FindByQuery retrieves audit logs based on query parameters
func (r *AuditRepository) FindByQuery(ctx context.Context, query *models.AuditLogQuery) ([]models.TaskAuditLog, int64, error) {
	var logs []models.TaskAuditLog
	var total int64

	db := r.db.WithContext(ctx).Model(&models.TaskAuditLog{})

	// Apply filters
	if query.TaskID != nil {
		db = db.Where("task_id = ?", *query.TaskID)
	}
	if query.PatientID != "" {
		db = db.Where("patient_id = ?", query.PatientID)
	}
	if query.ActorID != nil {
		db = db.Where("actor_id = ?", *query.ActorID)
	}
	if query.EventType != nil {
		db = db.Where("event_type = ?", *query.EventType)
	}
	if query.EventCategory != nil {
		db = db.Where("event_category = ?", *query.EventCategory)
	}
	if query.StartDate != nil {
		db = db.Where("event_timestamp >= ?", *query.StartDate)
	}
	if query.EndDate != nil {
		db = db.Where("event_timestamp <= ?", *query.EndDate)
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Apply pagination
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// Order by timestamp descending
	db = db.Order("event_timestamp DESC")

	if err := db.Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find audit logs: %w", err)
	}

	return logs, total, nil
}

// GetAuditSummary retrieves summary for a task
func (r *AuditRepository) GetAuditSummary(ctx context.Context, taskID uuid.UUID) (*models.AuditSummary, error) {
	var summary models.AuditSummary

	// Get task info
	var task models.Task
	if err := r.db.WithContext(ctx).First(&task, "id = ?", taskID).Error; err != nil {
		return nil, fmt.Errorf("failed to find task: %w", err)
	}

	summary.TaskID = task.ID
	summary.TaskNumber = task.TaskID
	summary.PatientID = task.PatientID
	summary.TaskType = task.Type
	summary.CurrentStatus = task.Status

	// Get audit stats
	var stats struct {
		TotalEvents  int64
		FirstEvent   time.Time
		LastEvent    time.Time
		UniqueActors int
	}

	r.db.WithContext(ctx).
		Model(&models.TaskAuditLog{}).
		Select("COUNT(*) as total_events, MIN(event_timestamp) as first_event, MAX(event_timestamp) as last_event, COUNT(DISTINCT actor_id) as unique_actors").
		Where("task_id = ?", taskID).
		Scan(&stats)

	summary.TotalEvents = stats.TotalEvents
	summary.FirstEvent = stats.FirstEvent
	summary.LastEvent = stats.LastEvent
	summary.UniqueActors = stats.UniqueActors

	// Get unique event types
	var eventTypes []models.AuditEventType
	r.db.WithContext(ctx).
		Model(&models.TaskAuditLog{}).
		Distinct("event_type").
		Where("task_id = ?", taskID).
		Pluck("event_type", &eventTypes)

	summary.EventTypes = eventTypes

	// Check for reason codes
	var hasReasonCodes int64
	r.db.WithContext(ctx).
		Model(&models.TaskAuditLog{}).
		Where("task_id = ? AND reason_code IS NOT NULL AND reason_code != ''", taskID).
		Count(&hasReasonCodes)

	summary.HasReasonCodes = hasReasonCodes > 0

	return &summary, nil
}

// VerifyHashChain verifies the integrity of the hash chain for a task
func (r *AuditRepository) VerifyHashChain(ctx context.Context, taskID uuid.UUID) (bool, []string, error) {
	logs, err := r.FindByTask(ctx, taskID)
	if err != nil {
		return false, nil, err
	}

	var errors []string
	for i, log := range logs {
		// Verify hash calculation
		expectedHash := log.CalculateHash()
		if log.RecordHash != expectedHash {
			errors = append(errors, fmt.Sprintf("Record %d: hash mismatch (expected: %s, got: %s)", i, expectedHash, log.RecordHash))
		}

		// Verify chain link (skip first record)
		if i > 0 {
			if log.PreviousHash != logs[i-1].RecordHash {
				errors = append(errors, fmt.Sprintf("Record %d: chain break (expected previous: %s, got: %s)", i, logs[i-1].RecordHash, log.PreviousHash))
			}
		}
	}

	return len(errors) == 0, errors, nil
}

// =============================================================================
// GOVERNANCE EVENT REPOSITORY
// =============================================================================

// GovernanceRepository handles governance event database operations
type GovernanceRepository struct {
	db *gorm.DB
}

// NewGovernanceRepository creates a new GovernanceRepository
func NewGovernanceRepository(db *gorm.DB) *GovernanceRepository {
	return &GovernanceRepository{db: db}
}

// Create creates a new governance event
func (r *GovernanceRepository) Create(ctx context.Context, event *models.GovernanceEvent) error {
	if err := r.db.WithContext(ctx).Create(event).Error; err != nil {
		return fmt.Errorf("failed to create governance event: %w", err)
	}
	return nil
}

// GetByID retrieves a governance event by ID
func (r *GovernanceRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.GovernanceEvent, error) {
	var event models.GovernanceEvent
	if err := r.db.WithContext(ctx).First(&event, "id = ?", id).Error; err != nil {
		return nil, fmt.Errorf("failed to find governance event: %w", err)
	}
	return &event, nil
}

// FindByTask retrieves all governance events for a task
func (r *GovernanceRepository) FindByTask(ctx context.Context, taskID uuid.UUID) ([]models.GovernanceEvent, error) {
	var events []models.GovernanceEvent
	err := r.db.WithContext(ctx).
		Where("task_id = ?", taskID).
		Order("created_at DESC").
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find governance events: %w", err)
	}

	return events, nil
}

// FindUnresolved retrieves all unresolved governance events
func (r *GovernanceRepository) FindUnresolved(ctx context.Context) ([]models.GovernanceEvent, error) {
	var events []models.GovernanceEvent
	err := r.db.WithContext(ctx).
		Where("resolved = ?", false).
		Order("severity DESC, created_at ASC").
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find unresolved governance events: %w", err)
	}

	return events, nil
}

// FindRequiringAction retrieves events requiring action
func (r *GovernanceRepository) FindRequiringAction(ctx context.Context) ([]models.GovernanceEvent, error) {
	var events []models.GovernanceEvent
	err := r.db.WithContext(ctx).
		Where("resolved = ? AND requires_action = ?", false, true).
		Order("action_deadline ASC, severity DESC").
		Find(&events).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find events requiring action: %w", err)
	}

	return events, nil
}

// FindByQuery retrieves governance events based on query parameters
func (r *GovernanceRepository) FindByQuery(ctx context.Context, query *models.GovernanceEventQuery) ([]models.GovernanceEvent, int64, error) {
	var events []models.GovernanceEvent
	var total int64

	db := r.db.WithContext(ctx).Model(&models.GovernanceEvent{})

	// Apply filters
	if query.EventType != nil {
		db = db.Where("event_type = ?", *query.EventType)
	}
	if query.Severity != nil {
		db = db.Where("severity = ?", *query.Severity)
	}
	if query.TaskID != nil {
		db = db.Where("task_id = ?", *query.TaskID)
	}
	if query.PatientID != "" {
		db = db.Where("patient_id = ?", query.PatientID)
	}
	if query.Resolved != nil {
		db = db.Where("resolved = ?", *query.Resolved)
	}
	if query.RequiresAction != nil {
		db = db.Where("requires_action = ?", *query.RequiresAction)
	}
	if query.StartDate != nil {
		db = db.Where("created_at >= ?", *query.StartDate)
	}
	if query.EndDate != nil {
		db = db.Where("created_at <= ?", *query.EndDate)
	}

	// Get total count
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count governance events: %w", err)
	}

	// Apply pagination
	if query.Limit > 0 {
		db = db.Limit(query.Limit)
	}
	if query.Offset > 0 {
		db = db.Offset(query.Offset)
	}

	// Order by created_at descending
	db = db.Order("created_at DESC")

	if err := db.Find(&events).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to find governance events: %w", err)
	}

	return events, total, nil
}

// Resolve marks a governance event as resolved
func (r *GovernanceRepository) Resolve(ctx context.Context, id uuid.UUID, resolvedBy uuid.UUID, notes string) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.GovernanceEvent{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"resolved":         true,
			"resolved_by":      resolvedBy,
			"resolved_at":      now,
			"resolution_notes": notes,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to resolve governance event: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return fmt.Errorf("governance event not found: %s", id)
	}

	return nil
}

// GetDashboard retrieves dashboard statistics
func (r *GovernanceRepository) GetDashboard(ctx context.Context, days int) ([]models.GovernanceDashboard, error) {
	var dashboard []models.GovernanceDashboard

	startDate := time.Now().UTC().AddDate(0, 0, -days)

	err := r.db.WithContext(ctx).
		Model(&models.GovernanceEvent{}).
		Select(`
			DATE(created_at) as date,
			event_type,
			severity,
			COUNT(*) as event_count,
			COUNT(*) FILTER (WHERE resolved = true) as resolved_count,
			COUNT(*) FILTER (WHERE resolved = false AND requires_action = true) as pending_action_count,
			AVG(compliance_score) as avg_compliance_score,
			AVG(risk_score) as avg_risk_score
		`).
		Where("created_at >= ?", startDate).
		Group("DATE(created_at), event_type, severity").
		Order("date DESC, severity DESC").
		Scan(&dashboard).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get governance dashboard: %w", err)
	}

	return dashboard, nil
}

// =============================================================================
// REASON CODE REPOSITORY
// =============================================================================

// ReasonCodeRepository handles reason code database operations
type ReasonCodeRepository struct {
	db *gorm.DB
}

// NewReasonCodeRepository creates a new ReasonCodeRepository
func NewReasonCodeRepository(db *gorm.DB) *ReasonCodeRepository {
	return &ReasonCodeRepository{db: db}
}

// GetAll retrieves all active reason codes
func (r *ReasonCodeRepository) GetAll(ctx context.Context) ([]models.ReasonCode, error) {
	var codes []models.ReasonCode
	err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("category, sort_order").
		Find(&codes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get reason codes: %w", err)
	}

	return codes, nil
}

// GetByCategory retrieves reason codes by category
func (r *ReasonCodeRepository) GetByCategory(ctx context.Context, category models.ReasonCodeCategory) ([]models.ReasonCode, error) {
	var codes []models.ReasonCode
	err := r.db.WithContext(ctx).
		Where("category = ? AND is_active = ?", category, true).
		Order("sort_order").
		Find(&codes).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get reason codes: %w", err)
	}

	return codes, nil
}

// GetByCode retrieves a reason code by code
func (r *ReasonCodeRepository) GetByCode(ctx context.Context, code string) (*models.ReasonCode, error) {
	var reasonCode models.ReasonCode
	if err := r.db.WithContext(ctx).First(&reasonCode, "code = ?", code).Error; err != nil {
		return nil, fmt.Errorf("failed to find reason code: %w", err)
	}
	return &reasonCode, nil
}

// ValidateCode validates a reason code exists and is active
func (r *ReasonCodeRepository) ValidateCode(ctx context.Context, code string) (bool, *models.ReasonCode, error) {
	reasonCode, err := r.GetByCode(ctx, code)
	if err != nil {
		return false, nil, nil
	}

	if !reasonCode.IsActive {
		return false, reasonCode, nil
	}

	return true, reasonCode, nil
}

// =============================================================================
// INTELLIGENCE TRACKING REPOSITORY
// =============================================================================

// IntelligenceRepository handles intelligence tracking database operations
type IntelligenceRepository struct {
	db *gorm.DB
}

// NewIntelligenceRepository creates a new IntelligenceRepository
func NewIntelligenceRepository(db *gorm.DB) *IntelligenceRepository {
	return &IntelligenceRepository{db: db}
}

// Create creates a new intelligence tracking record
func (r *IntelligenceRepository) Create(ctx context.Context, intel *models.IntelligenceTracking) error {
	if err := r.db.WithContext(ctx).Create(intel).Error; err != nil {
		return fmt.Errorf("failed to create intelligence tracking: %w", err)
	}
	return nil
}

// GetBySourceID retrieves intelligence by source service and ID
func (r *IntelligenceRepository) GetBySourceID(ctx context.Context, service, sourceID string) (*models.IntelligenceTracking, error) {
	var intel models.IntelligenceTracking
	err := r.db.WithContext(ctx).
		Where("source_service = ? AND source_id = ?", service, sourceID).
		First(&intel).Error

	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to find intelligence: %w", err)
	}

	return &intel, nil
}

// FindByPatient retrieves all intelligence for a patient
func (r *IntelligenceRepository) FindByPatient(ctx context.Context, patientID string) ([]models.IntelligenceTracking, error) {
	var intel []models.IntelligenceTracking
	err := r.db.WithContext(ctx).
		Where("patient_id = ?", patientID).
		Order("received_at DESC").
		Find(&intel).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find intelligence: %w", err)
	}

	return intel, nil
}

// FindPending retrieves all pending (unprocessed) intelligence
func (r *IntelligenceRepository) FindPending(ctx context.Context) ([]models.IntelligenceTracking, error) {
	var intel []models.IntelligenceTracking
	err := r.db.WithContext(ctx).
		Where("status = ?", models.IntelligenceStatusReceived).
		Order("received_at ASC").
		Find(&intel).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find pending intelligence: %w", err)
	}

	return intel, nil
}

// UpdateStatus updates the status of an intelligence record
func (r *IntelligenceRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status models.IntelligenceStatus, taskID *uuid.UUID) error {
	now := time.Now().UTC()
	updates := map[string]interface{}{
		"status":       status,
		"processed_at": now,
	}

	if taskID != nil {
		updates["task_id"] = *taskID
	}

	result := r.db.WithContext(ctx).
		Model(&models.IntelligenceTracking{}).
		Where("id = ?", id).
		Updates(updates)

	if result.Error != nil {
		return fmt.Errorf("failed to update intelligence status: %w", result.Error)
	}

	return nil
}

// Disposition records a disposition for intelligence that didn't become a task
func (r *IntelligenceRepository) Disposition(ctx context.Context, id uuid.UUID, code, reason string, by uuid.UUID) error {
	now := time.Now().UTC()
	result := r.db.WithContext(ctx).
		Model(&models.IntelligenceTracking{}).
		Where("id = ?", id).
		Updates(map[string]interface{}{
			"status":             models.IntelligenceStatusDeclined,
			"disposition_code":   code,
			"disposition_reason": reason,
			"disposition_by":     by,
			"disposition_at":     now,
			"processed_at":       now,
		})

	if result.Error != nil {
		return fmt.Errorf("failed to disposition intelligence: %w", result.Error)
	}

	return nil
}

// GetAccountability retrieves intelligence accountability statistics
func (r *IntelligenceRepository) GetAccountability(ctx context.Context, days int) ([]models.IntelligenceAccountability, error) {
	var accountability []models.IntelligenceAccountability

	startDate := time.Now().UTC().AddDate(0, 0, -days)

	err := r.db.WithContext(ctx).
		Model(&models.IntelligenceTracking{}).
		Select(`
			source_service,
			source_type,
			status,
			COUNT(*) as count,
			COUNT(*) FILTER (WHERE task_id IS NOT NULL) as tasks_created,
			COUNT(*) FILTER (WHERE disposition_code IS NOT NULL) as dispositioned,
			COUNT(*) FILTER (WHERE status = 'RECEIVED' AND processed_at IS NULL) as pending
		`).
		Where("received_at >= ?", startDate).
		Group("source_service, source_type, status").
		Order("source_service, count DESC").
		Scan(&accountability).Error

	if err != nil {
		return nil, fmt.Errorf("failed to get intelligence accountability: %w", err)
	}

	return accountability, nil
}

// FindOrphanedIntelligence finds intelligence that hasn't been processed within timeout
func (r *IntelligenceRepository) FindOrphanedIntelligence(ctx context.Context, timeoutMinutes int) ([]models.IntelligenceTracking, error) {
	var intel []models.IntelligenceTracking
	cutoff := time.Now().UTC().Add(-time.Duration(timeoutMinutes) * time.Minute)

	err := r.db.WithContext(ctx).
		Where("status = ? AND received_at < ?", models.IntelligenceStatusReceived, cutoff).
		Order("received_at ASC").
		Find(&intel).Error

	if err != nil {
		return nil, fmt.Errorf("failed to find orphaned intelligence: %w", err)
	}

	return intel, nil
}
