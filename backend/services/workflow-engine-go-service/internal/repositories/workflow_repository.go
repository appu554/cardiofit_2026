package repositories

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"

	"github.com/clinical-synthesis-hub/workflow-engine-go-service/internal/domain"
)

// WorkflowRepository defines the interface for workflow data operations
type WorkflowRepository interface {
	Create(ctx context.Context, instance *domain.WorkflowInstance) error
	GetByID(ctx context.Context, id string) (*WorkflowInstance, error)
	UpdateStatus(ctx context.Context, id string, status domain.WorkflowStatus, message string) error
	List(ctx context.Context, options *WorkflowListOptions) ([]WorkflowInstance, int64, error)
	Delete(ctx context.Context, id string) error
}

// WorkflowInstance represents the database model for workflow instances
type WorkflowInstance struct {
	ID                 string                 `db:"id" json:"id"`
	DefinitionID       string                 `db:"definition_id" json:"definition_id"`
	PatientID          string                 `db:"patient_id" json:"patient_id"`
	Status             domain.WorkflowStatus  `db:"status" json:"status"`
	StartedAt          time.Time              `db:"started_at" json:"started_at"`
	CompletedAt        *time.Time             `db:"completed_at" json:"completed_at,omitempty"`
	CorrelationID      string                 `db:"correlation_id" json:"correlation_id"`
	Context            map[string]interface{} `db:"context" json:"context,omitempty"`
	ErrorMessage       *string                `db:"error_message" json:"error_message,omitempty"`
	CreatedAt          time.Time              `db:"created_at" json:"created_at"`
	UpdatedAt          time.Time              `db:"updated_at" json:"updated_at"`
}

// WorkflowListOptions defines filtering and pagination options
type WorkflowListOptions struct {
	PatientID     string    `json:"patient_id,omitempty"`
	Status        string    `json:"status,omitempty"`
	DefinitionID  string    `json:"definition_id,omitempty"`
	StartedAfter  time.Time `json:"started_after,omitempty"`
	StartedBefore time.Time `json:"started_before,omitempty"`
	Limit         int       `json:"limit"`
	Offset        int       `json:"offset"`
}

// workflowRepositoryImpl implements WorkflowRepository
type workflowRepositoryImpl struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewWorkflowRepository creates a new workflow repository
func NewWorkflowRepository(db *sqlx.DB, logger *zap.Logger) WorkflowRepository {
	return &workflowRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// Create inserts a new workflow instance
func (r *workflowRepositoryImpl) Create(ctx context.Context, instance *domain.WorkflowInstance) error {
	query := `
		INSERT INTO workflow_instances (
			id, definition_id, patient_id, status, started_at, 
			correlation_id, context, created_at, updated_at
		) VALUES (
			:id, :definition_id, :patient_id, :status, :started_at,
			:correlation_id, :context, :created_at, :updated_at
		)`

	dbInstance := &WorkflowInstance{
		ID:            instance.ID,
		DefinitionID:  instance.DefinitionID,
		PatientID:     instance.PatientID,
		Status:        instance.Status,
		StartedAt:     instance.StartedAt,
		CorrelationID: instance.CorrelationID,
		Context:       instance.Context,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	_, err := r.db.NamedExecContext(ctx, query, dbInstance)
	if err != nil {
		r.logger.Error("Failed to create workflow instance",
			zap.String("workflow_id", instance.ID),
			zap.Error(err))
		return fmt.Errorf("failed to create workflow instance: %w", err)
	}

	r.logger.Info("Created workflow instance",
		zap.String("workflow_id", instance.ID),
		zap.String("patient_id", instance.PatientID),
		zap.String("correlation_id", instance.CorrelationID))

	return nil
}

// GetByID retrieves a workflow instance by ID
func (r *workflowRepositoryImpl) GetByID(ctx context.Context, id string) (*WorkflowInstance, error) {
	query := `
		SELECT id, definition_id, patient_id, status, started_at, completed_at,
			   correlation_id, context, error_message, created_at, updated_at
		FROM workflow_instances 
		WHERE id = $1`

	var instance WorkflowInstance
	err := r.db.GetContext(ctx, &instance, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("workflow instance not found: %s", id)
		}
		r.logger.Error("Failed to get workflow instance",
			zap.String("workflow_id", id),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get workflow instance: %w", err)
	}

	return &instance, nil
}

// UpdateStatus updates the workflow instance status and completion time
func (r *workflowRepositoryImpl) UpdateStatus(ctx context.Context, id string, status domain.WorkflowStatus, message string) error {
	var query string
	var args []interface{}

	// Determine if this is a completion status
	isComplete := status == domain.WorkflowStatusCompleted || 
		status == domain.WorkflowStatusCompletedWithWarnings || 
		status == domain.WorkflowStatusFailed

	if isComplete {
		query = `
			UPDATE workflow_instances 
			SET status = $1, completed_at = $2, error_message = $3, updated_at = $4
			WHERE id = $5`
		
		var errorMessage *string
		if message != "" {
			errorMessage = &message
		}
		
		args = []interface{}{status, time.Now(), errorMessage, time.Now(), id}
	} else {
		query = `
			UPDATE workflow_instances 
			SET status = $1, updated_at = $2
			WHERE id = $3`
		args = []interface{}{status, time.Now(), id}
	}

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		r.logger.Error("Failed to update workflow status",
			zap.String("workflow_id", id),
			zap.String("status", string(status)),
			zap.Error(err))
		return fmt.Errorf("failed to update workflow status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Warn("Could not get rows affected for workflow update",
			zap.String("workflow_id", id),
			zap.Error(err))
	} else if rowsAffected == 0 {
		return fmt.Errorf("workflow instance not found: %s", id)
	}

	r.logger.Info("Updated workflow status",
		zap.String("workflow_id", id),
		zap.String("status", string(status)),
		zap.Bool("completed", isComplete))

	return nil
}

// List retrieves workflow instances with filtering and pagination
func (r *workflowRepositoryImpl) List(ctx context.Context, options *WorkflowListOptions) ([]WorkflowInstance, int64, error) {
	// Build WHERE clause and arguments
	whereClause := "WHERE 1=1"
	args := []interface{}{}
	argIndex := 1

	if options.PatientID != "" {
		whereClause += fmt.Sprintf(" AND patient_id = $%d", argIndex)
		args = append(args, options.PatientID)
		argIndex++
	}

	if options.Status != "" {
		whereClause += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, options.Status)
		argIndex++
	}

	if options.DefinitionID != "" {
		whereClause += fmt.Sprintf(" AND definition_id = $%d", argIndex)
		args = append(args, options.DefinitionID)
		argIndex++
	}

	if !options.StartedAfter.IsZero() {
		whereClause += fmt.Sprintf(" AND started_at >= $%d", argIndex)
		args = append(args, options.StartedAfter)
		argIndex++
	}

	if !options.StartedBefore.IsZero() {
		whereClause += fmt.Sprintf(" AND started_at <= $%d", argIndex)
		args = append(args, options.StartedBefore)
		argIndex++
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM workflow_instances %s", whereClause)
	var total int64
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		r.logger.Error("Failed to count workflow instances", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to count workflow instances: %w", err)
	}

	// Get paginated results
	listQuery := fmt.Sprintf(`
		SELECT id, definition_id, patient_id, status, started_at, completed_at,
			   correlation_id, context, error_message, created_at, updated_at
		FROM workflow_instances 
		%s
		ORDER BY started_at DESC
		LIMIT $%d OFFSET $%d`,
		whereClause, argIndex, argIndex+1)

	args = append(args, options.Limit, options.Offset)

	var instances []WorkflowInstance
	err = r.db.SelectContext(ctx, &instances, listQuery, args...)
	if err != nil {
		r.logger.Error("Failed to list workflow instances", zap.Error(err))
		return nil, 0, fmt.Errorf("failed to list workflow instances: %w", err)
	}

	r.logger.Info("Retrieved workflow instances",
		zap.Int("count", len(instances)),
		zap.Int64("total", total),
		zap.Int("limit", options.Limit),
		zap.Int("offset", options.Offset))

	return instances, total, nil
}

// Delete removes a workflow instance (soft delete by updating status)
func (r *workflowRepositoryImpl) Delete(ctx context.Context, id string) error {
	query := `
		UPDATE workflow_instances 
		SET status = $1, updated_at = $2
		WHERE id = $3`

	result, err := r.db.ExecContext(ctx, query, domain.WorkflowStatusDeleted, time.Now(), id)
	if err != nil {
		r.logger.Error("Failed to delete workflow instance",
			zap.String("workflow_id", id),
			zap.Error(err))
		return fmt.Errorf("failed to delete workflow instance: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Warn("Could not get rows affected for workflow deletion",
			zap.String("workflow_id", id),
			zap.Error(err))
	} else if rowsAffected == 0 {
		return fmt.Errorf("workflow instance not found: %s", id)
	}

	r.logger.Info("Deleted workflow instance", zap.String("workflow_id", id))
	return nil
}