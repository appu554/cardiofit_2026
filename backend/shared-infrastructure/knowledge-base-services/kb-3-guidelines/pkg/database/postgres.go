// Package database provides PostgreSQL database operations for KB-3 Guidelines
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-3-guidelines/pkg/models"
)

// PostgresService provides database operations
type PostgresService struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewPostgresService creates a new PostgreSQL service
func NewPostgresService(databaseURL string) (*PostgresService, error) {
	logger := logrus.New()
	logger.SetFormatter(&logrus.JSONFormatter{})

	db, err := sql.Open("pgx", databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Configure connection pool
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	// Verify connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.Info("PostgreSQL connection established")

	return &PostgresService{
		db:     db,
		logger: logger,
	}, nil
}

// Close closes the database connection
func (s *PostgresService) Close() error {
	return s.db.Close()
}

// Health checks database health
func (s *PostgresService) Health(ctx context.Context) error {
	return s.db.PingContext(ctx)
}

// BeginTx starts a new transaction
func (s *PostgresService) BeginTx(ctx context.Context) (*sql.Tx, error) {
	return s.db.BeginTx(ctx, nil)
}

// ===== Guideline Operations =====

// GetActiveGuidelines retrieves all active guidelines
func (s *PostgresService) GetActiveGuidelines(ctx context.Context) ([]models.Guideline, error) {
	query := `
		SELECT guideline_id, name, source, version, effective_date, status,
		       domain, evidence_grade, recommendations, active, created_at, updated_at
		FROM guidelines
		WHERE active = true
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query guidelines: %w", err)
	}
	defer rows.Close()

	var guidelines []models.Guideline
	for rows.Next() {
		var g models.Guideline
		var recsJSON []byte

		err := rows.Scan(
			&g.GuidelineID, &g.Name, &g.Source, &g.Version, &g.EffectiveDate,
			&g.Status, &g.Domain, &g.EvidenceGrade, &recsJSON, &g.Active,
			&g.CreatedAt, &g.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline: %w", err)
		}

		if len(recsJSON) > 0 {
			if err := json.Unmarshal(recsJSON, &g.Recommendations); err != nil {
				s.logger.WithError(err).Warn("Failed to unmarshal recommendations")
			}
		}

		guidelines = append(guidelines, g)
	}

	return guidelines, nil
}

// GetGuidelineByID retrieves a guideline by ID
func (s *PostgresService) GetGuidelineByID(ctx context.Context, guidelineID string) (*models.Guideline, error) {
	query := `
		SELECT guideline_id, name, source, version, effective_date, status,
		       domain, evidence_grade, recommendations, active, created_at, updated_at
		FROM guidelines
		WHERE guideline_id = $1
	`

	var g models.Guideline
	var recsJSON []byte

	err := s.db.QueryRowContext(ctx, query, guidelineID).Scan(
		&g.GuidelineID, &g.Name, &g.Source, &g.Version, &g.EffectiveDate,
		&g.Status, &g.Domain, &g.EvidenceGrade, &recsJSON, &g.Active,
		&g.CreatedAt, &g.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get guideline: %w", err)
	}

	if len(recsJSON) > 0 {
		json.Unmarshal(recsJSON, &g.Recommendations)
	}

	return &g, nil
}

// CreateGuideline creates a new guideline
func (s *PostgresService) CreateGuideline(ctx context.Context, g *models.Guideline) error {
	recsJSON, _ := json.Marshal(g.Recommendations)

	query := `
		INSERT INTO guidelines (guideline_id, name, source, version, effective_date, status,
		                        domain, evidence_grade, recommendations, active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.ExecContext(ctx, query,
		g.GuidelineID, g.Name, g.Source, g.Version, g.EffectiveDate,
		g.Status, g.Domain, g.EvidenceGrade, recsJSON, g.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to create guideline: %w", err)
	}

	return nil
}

// ===== Conflict Operations =====

// GetConflicts retrieves conflicts, optionally filtered by status
func (s *PostgresService) GetConflicts(ctx context.Context, status string) ([]models.Conflict, error) {
	query := `
		SELECT conflict_id, guideline1_id, guideline2_id, recommendation1, recommendation2,
		       type, severity, domain, status, detected_at, resolved_at
		FROM conflicts
		WHERE ($1 = '' OR status = $1)
		ORDER BY detected_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, status)
	if err != nil {
		return nil, fmt.Errorf("failed to query conflicts: %w", err)
	}
	defer rows.Close()

	var conflicts []models.Conflict
	for rows.Next() {
		var c models.Conflict
		var rec1JSON, rec2JSON []byte

		err := rows.Scan(
			&c.ConflictID, &c.Guideline1ID, &c.Guideline2ID, &rec1JSON, &rec2JSON,
			&c.Type, &c.Severity, &c.Domain, &c.Status, &c.DetectedAt, &c.ResolvedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conflict: %w", err)
		}

		json.Unmarshal(rec1JSON, &c.Recommendation1)
		json.Unmarshal(rec2JSON, &c.Recommendation2)

		conflicts = append(conflicts, c)
	}

	return conflicts, nil
}

// CreateConflict creates a new conflict record
func (s *PostgresService) CreateConflict(ctx context.Context, c *models.Conflict) error {
	rec1JSON, _ := json.Marshal(c.Recommendation1)
	rec2JSON, _ := json.Marshal(c.Recommendation2)

	query := `
		INSERT INTO conflicts (conflict_id, guideline1_id, guideline2_id, recommendation1,
		                       recommendation2, type, severity, domain, status, detected_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.ExecContext(ctx, query,
		c.ConflictID, c.Guideline1ID, c.Guideline2ID, rec1JSON, rec2JSON,
		c.Type, c.Severity, c.Domain, c.Status, c.DetectedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create conflict: %w", err)
	}

	return nil
}

// ResolveConflict marks a conflict as resolved
func (s *PostgresService) ResolveConflict(ctx context.Context, conflictID string, resolution *models.Resolution) error {
	resJSON, _ := json.Marshal(resolution)

	query := `
		UPDATE conflicts
		SET status = 'resolved', resolved_at = $2, resolution = $3
		WHERE conflict_id = $1
	`

	_, err := s.db.ExecContext(ctx, query, conflictID, time.Now(), resJSON)
	if err != nil {
		return fmt.Errorf("failed to resolve conflict: %w", err)
	}

	return nil
}

// ===== Safety Override Operations =====

// GetActiveSafetyOverrides retrieves all active safety overrides
func (s *PostgresService) GetActiveSafetyOverrides(ctx context.Context) ([]models.SafetyOverride, error) {
	query := `
		SELECT override_id, name, description, trigger_conditions, override_action,
		       priority, active, affected_guidelines, requires_signature, clinical_rationale
		FROM safety_overrides
		WHERE active = true
		ORDER BY priority ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query safety overrides: %w", err)
	}
	defer rows.Close()

	var overrides []models.SafetyOverride
	for rows.Next() {
		var o models.SafetyOverride
		var triggerJSON, actionJSON, guidelinesJSON []byte

		err := rows.Scan(
			&o.OverrideID, &o.Name, &o.Description, &triggerJSON, &actionJSON,
			&o.Priority, &o.Active, &guidelinesJSON, &o.RequiresSignature, &o.ClinicalRationale,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan safety override: %w", err)
		}

		json.Unmarshal(triggerJSON, &o.TriggerConditions)
		json.Unmarshal(actionJSON, &o.OverrideAction)
		json.Unmarshal(guidelinesJSON, &o.AffectedGuidelines)

		overrides = append(overrides, o)
	}

	return overrides, nil
}

// CreateSafetyOverride creates a new safety override
func (s *PostgresService) CreateSafetyOverride(ctx context.Context, o *models.SafetyOverride) error {
	triggerJSON, _ := json.Marshal(o.TriggerConditions)
	actionJSON, _ := json.Marshal(o.OverrideAction)
	guidelinesJSON, _ := json.Marshal(o.AffectedGuidelines)

	query := `
		INSERT INTO safety_overrides (override_id, name, description, trigger_conditions,
		                              override_action, priority, active, affected_guidelines,
		                              requires_signature, clinical_rationale)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	_, err := s.db.ExecContext(ctx, query,
		o.OverrideID, o.Name, o.Description, triggerJSON, actionJSON,
		o.Priority, o.Active, guidelinesJSON, o.RequiresSignature, o.ClinicalRationale,
	)
	if err != nil {
		return fmt.Errorf("failed to create safety override: %w", err)
	}

	return nil
}

// ===== Protocol Operations =====

// SaveProtocol saves a protocol definition
func (s *PostgresService) SaveProtocol(ctx context.Context, p *models.Protocol) error {
	defJSON, _ := json.Marshal(p)

	query := `
		INSERT INTO protocols (protocol_id, name, protocol_type, guideline_source, definition, active)
		VALUES ($1, $2, $3, $4, $5, $6)
		ON CONFLICT (protocol_id) DO UPDATE SET
			name = EXCLUDED.name,
			guideline_source = EXCLUDED.guideline_source,
			definition = EXCLUDED.definition,
			active = EXCLUDED.active,
			updated_at = NOW()
	`

	_, err := s.db.ExecContext(ctx, query,
		p.ProtocolID, p.Name, p.Type, p.GuidelineSource, defJSON, p.Active,
	)
	if err != nil {
		return fmt.Errorf("failed to save protocol: %w", err)
	}

	return nil
}

// GetProtocol retrieves a protocol by ID
func (s *PostgresService) GetProtocol(ctx context.Context, protocolID string) (*models.Protocol, error) {
	query := `SELECT definition FROM protocols WHERE protocol_id = $1`

	var defJSON []byte
	err := s.db.QueryRowContext(ctx, query, protocolID).Scan(&defJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get protocol: %w", err)
	}

	var p models.Protocol
	if err := json.Unmarshal(defJSON, &p); err != nil {
		return nil, fmt.Errorf("failed to unmarshal protocol: %w", err)
	}

	return &p, nil
}

// ===== Pathway Instance Operations =====

// SavePathwayInstance saves a pathway instance
func (s *PostgresService) SavePathwayInstance(ctx context.Context, instance *models.PathwayInstance) error {
	contextJSON, _ := json.Marshal(instance.Context)

	query := `
		INSERT INTO pathway_instances (instance_id, pathway_id, patient_id, current_stage,
		                               status, context, started_at, completed_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (instance_id) DO UPDATE SET
			current_stage = EXCLUDED.current_stage,
			status = EXCLUDED.status,
			context = EXCLUDED.context,
			completed_at = EXCLUDED.completed_at
	`

	_, err := s.db.ExecContext(ctx, query,
		instance.InstanceID, instance.PathwayID, instance.PatientID, instance.CurrentStage,
		instance.Status, contextJSON, instance.StartedAt, instance.CompletedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save pathway instance: %w", err)
	}

	// Save actions
	for _, action := range instance.Actions {
		if err := s.SavePathwayAction(ctx, instance.InstanceID, &action); err != nil {
			s.logger.WithError(err).Warn("Failed to save pathway action")
		}
	}

	return nil
}

// SavePathwayAction saves a pathway action
func (s *PostgresService) SavePathwayAction(ctx context.Context, instanceID string, action *models.PathwayAction) error {
	query := `
		INSERT INTO pathway_actions (action_id, instance_id, name, action_type, status,
		                             deadline, grace_period, completed_at, completed_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		ON CONFLICT (action_id) DO UPDATE SET
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at,
			completed_by = EXCLUDED.completed_by
	`

	_, err := s.db.ExecContext(ctx, query,
		action.ActionID, instanceID, action.Name, action.Type, action.Status,
		action.Deadline, action.GracePeriod, action.CompletedAt, action.CompletedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to save pathway action: %w", err)
	}

	return nil
}

// GetPatientPathways retrieves pathway instances for a patient
func (s *PostgresService) GetPatientPathways(ctx context.Context, patientID string) ([]models.PathwayInstance, error) {
	query := `
		SELECT instance_id, pathway_id, patient_id, current_stage, status,
		       context, started_at, completed_at
		FROM pathway_instances
		WHERE patient_id = $1
		ORDER BY started_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pathway instances: %w", err)
	}
	defer rows.Close()

	var instances []models.PathwayInstance
	for rows.Next() {
		var i models.PathwayInstance
		var contextJSON []byte

		err := rows.Scan(
			&i.InstanceID, &i.PathwayID, &i.PatientID, &i.CurrentStage, &i.Status,
			&contextJSON, &i.StartedAt, &i.CompletedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pathway instance: %w", err)
		}

		json.Unmarshal(contextJSON, &i.Context)

		// Load actions for this instance
		actions, err := s.GetPathwayActions(ctx, i.InstanceID)
		if err == nil {
			i.Actions = actions
		}

		instances = append(instances, i)
	}

	return instances, nil
}

// GetPathwayActions retrieves actions for a pathway instance
func (s *PostgresService) GetPathwayActions(ctx context.Context, instanceID string) ([]models.PathwayAction, error) {
	query := `
		SELECT action_id, name, action_type, status, deadline, grace_period,
		       completed_at, completed_by
		FROM pathway_actions
		WHERE instance_id = $1
		ORDER BY deadline ASC
	`

	rows, err := s.db.QueryContext(ctx, query, instanceID)
	if err != nil {
		return nil, fmt.Errorf("failed to query pathway actions: %w", err)
	}
	defer rows.Close()

	var actions []models.PathwayAction
	for rows.Next() {
		var a models.PathwayAction
		err := rows.Scan(
			&a.ActionID, &a.Name, &a.Type, &a.Status, &a.Deadline, &a.GracePeriod,
			&a.CompletedAt, &a.CompletedBy,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan pathway action: %w", err)
		}
		actions = append(actions, a)
	}

	return actions, nil
}

// ===== Scheduled Item Operations =====

// SaveScheduledItem saves a scheduled item
func (s *PostgresService) SaveScheduledItem(ctx context.Context, item *models.ScheduledItem) error {
	recurrenceJSON, _ := json.Marshal(item.Recurrence)

	query := `
		INSERT INTO scheduled_items (item_id, patient_id, item_type, name, due_date,
		                             priority, is_recurring, recurrence, status,
		                             completed_at, source_protocol)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (item_id) DO UPDATE SET
			due_date = EXCLUDED.due_date,
			status = EXCLUDED.status,
			completed_at = EXCLUDED.completed_at
	`

	_, err := s.db.ExecContext(ctx, query,
		item.ItemID, item.PatientID, item.Type, item.Name, item.DueDate,
		item.Priority, item.IsRecurring, recurrenceJSON, item.Status,
		item.CompletedAt, item.SourceProtocol,
	)
	if err != nil {
		return fmt.Errorf("failed to save scheduled item: %w", err)
	}

	return nil
}

// GetPatientSchedule retrieves scheduled items for a patient
func (s *PostgresService) GetPatientSchedule(ctx context.Context, patientID string) ([]models.ScheduledItem, error) {
	query := `
		SELECT item_id, patient_id, item_type, name, due_date, priority,
		       is_recurring, recurrence, status, completed_at, source_protocol,
		       created_at, updated_at
		FROM scheduled_items
		WHERE patient_id = $1
		ORDER BY due_date ASC
	`

	rows, err := s.db.QueryContext(ctx, query, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query scheduled items: %w", err)
	}
	defer rows.Close()

	var items []models.ScheduledItem
	for rows.Next() {
		var item models.ScheduledItem
		var recurrenceJSON []byte

		err := rows.Scan(
			&item.ItemID, &item.PatientID, &item.Type, &item.Name, &item.DueDate,
			&item.Priority, &item.IsRecurring, &recurrenceJSON, &item.Status,
			&item.CompletedAt, &item.SourceProtocol, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled item: %w", err)
		}

		if len(recurrenceJSON) > 0 {
			var rec models.RecurrencePattern
			if err := json.Unmarshal(recurrenceJSON, &rec); err == nil {
				item.Recurrence = &rec
			}
		}

		items = append(items, item)
	}

	return items, nil
}

// GetOverdueItems retrieves all overdue scheduled items
func (s *PostgresService) GetOverdueItems(ctx context.Context) ([]models.ScheduledItem, error) {
	query := `
		SELECT item_id, patient_id, item_type, name, due_date, priority,
		       is_recurring, recurrence, status, completed_at, source_protocol,
		       created_at, updated_at
		FROM scheduled_items
		WHERE status = 'pending' AND due_date < NOW()
		ORDER BY priority ASC, due_date ASC
	`

	rows, err := s.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query overdue items: %w", err)
	}
	defer rows.Close()

	var items []models.ScheduledItem
	for rows.Next() {
		var item models.ScheduledItem
		var recurrenceJSON []byte

		err := rows.Scan(
			&item.ItemID, &item.PatientID, &item.Type, &item.Name, &item.DueDate,
			&item.Priority, &item.IsRecurring, &recurrenceJSON, &item.Status,
			&item.CompletedAt, &item.SourceProtocol, &item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan scheduled item: %w", err)
		}

		item.Status = models.ScheduleOverdue // Update status

		if len(recurrenceJSON) > 0 {
			var rec models.RecurrencePattern
			json.Unmarshal(recurrenceJSON, &rec)
			item.Recurrence = &rec
		}

		items = append(items, item)
	}

	return items, nil
}

// ===== Version Operations =====

// SaveGuidelineVersion saves a guideline version
func (s *PostgresService) SaveGuidelineVersion(ctx context.Context, v *models.GuidelineVersion) error {
	changesJSON, _ := json.Marshal(v.Changes)
	impactJSON, _ := json.Marshal(v.ClinicalImpact)
	approvalJSON, _ := json.Marshal(v.ApprovalChain)
	transitionJSON, _ := json.Marshal(v.TransitionPlan)

	query := `
		INSERT INTO guideline_versions (version_id, guideline_id, version, change_type,
		                                changes, clinical_impact, approval_chain,
		                                transition_plan, status, created_by, effective_date)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		ON CONFLICT (version_id) DO UPDATE SET
			status = EXCLUDED.status,
			approval_chain = EXCLUDED.approval_chain,
			effective_date = EXCLUDED.effective_date
	`

	_, err := s.db.ExecContext(ctx, query,
		v.VersionID, v.GuidelineID, v.Version, v.ChangeType,
		changesJSON, impactJSON, approvalJSON, transitionJSON,
		v.Status, v.CreatedBy, v.EffectiveDate,
	)
	if err != nil {
		return fmt.Errorf("failed to save guideline version: %w", err)
	}

	return nil
}

// GetGuidelineVersions retrieves versions for a guideline
func (s *PostgresService) GetGuidelineVersions(ctx context.Context, guidelineID string) ([]models.GuidelineVersion, error) {
	query := `
		SELECT version_id, guideline_id, version, change_type, changes,
		       clinical_impact, approval_chain, transition_plan, status,
		       created_by, created_at, effective_date
		FROM guideline_versions
		WHERE guideline_id = $1
		ORDER BY created_at DESC
	`

	rows, err := s.db.QueryContext(ctx, query, guidelineID)
	if err != nil {
		return nil, fmt.Errorf("failed to query guideline versions: %w", err)
	}
	defer rows.Close()

	var versions []models.GuidelineVersion
	for rows.Next() {
		var v models.GuidelineVersion
		var changesJSON, impactJSON, approvalJSON, transitionJSON []byte

		err := rows.Scan(
			&v.VersionID, &v.GuidelineID, &v.Version, &v.ChangeType, &changesJSON,
			&impactJSON, &approvalJSON, &transitionJSON, &v.Status,
			&v.CreatedBy, &v.CreatedAt, &v.EffectiveDate,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan guideline version: %w", err)
		}

		json.Unmarshal(changesJSON, &v.Changes)
		json.Unmarshal(impactJSON, &v.ClinicalImpact)
		json.Unmarshal(approvalJSON, &v.ApprovalChain)
		if len(transitionJSON) > 0 {
			var tp models.TransitionPlan
			json.Unmarshal(transitionJSON, &tp)
			v.TransitionPlan = &tp
		}

		versions = append(versions, v)
	}

	return versions, nil
}

// ===== Audit Operations =====

// LogAuditEntry logs an audit entry
func (s *PostgresService) LogAuditEntry(ctx context.Context, entry *models.AuditEntry) error {
	detailsJSON, _ := json.Marshal(entry.Details)

	query := `
		INSERT INTO audit_log (entry_id, action, user_id, checksum, timestamp, details)
		VALUES ($1, $2, $3, $4, $5, $6)
	`

	_, err := s.db.ExecContext(ctx, query,
		entry.EntryID, entry.Action, entry.UserID, entry.Checksum, entry.Timestamp, detailsJSON,
	)
	if err != nil {
		return fmt.Errorf("failed to log audit entry: %w", err)
	}

	return nil
}
