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

// SnapshotRepository defines the interface for snapshot data operations
type SnapshotRepository interface {
	Create(ctx context.Context, snapshot *domain.SnapshotReference) error
	GetByID(ctx context.Context, snapshotID string) (*domain.SnapshotReference, error)
	GetByPatientID(ctx context.Context, patientID string, limit int) ([]*domain.SnapshotReference, error)
	UpdateStatus(ctx context.Context, snapshotID string, status domain.SnapshotStatus) error
	ExpireSnapshots(ctx context.Context, before time.Time) (int64, error)
	ValidateChecksum(ctx context.Context, snapshotID, expectedChecksum string) (bool, error)
}

// snapshotRepositoryImpl implements SnapshotRepository
type snapshotRepositoryImpl struct {
	db     *sqlx.DB
	logger *zap.Logger
}

// NewSnapshotRepository creates a new snapshot repository
func NewSnapshotRepository(db *sqlx.DB, logger *zap.Logger) SnapshotRepository {
	return &snapshotRepositoryImpl{
		db:     db,
		logger: logger,
	}
}

// Create inserts a new snapshot reference
func (r *snapshotRepositoryImpl) Create(ctx context.Context, snapshot *domain.SnapshotReference) error {
	query := `
		INSERT INTO snapshots (
			snapshot_id, checksum, created_at, expires_at, status,
			phase_created, patient_id, context_version, metadata
		) VALUES (
			:snapshot_id, :checksum, :created_at, :expires_at, :status,
			:phase_created, :patient_id, :context_version, :metadata
		)`

	_, err := r.db.NamedExecContext(ctx, query, snapshot)
	if err != nil {
		r.logger.Error("Failed to create snapshot reference",
			zap.String("snapshot_id", snapshot.SnapshotID),
			zap.String("patient_id", snapshot.PatientID),
			zap.Error(err))
		return fmt.Errorf("failed to create snapshot reference: %w", err)
	}

	r.logger.Info("Created snapshot reference",
		zap.String("snapshot_id", snapshot.SnapshotID),
		zap.String("patient_id", snapshot.PatientID),
		zap.String("phase", string(snapshot.PhaseCreated)))

	return nil
}

// GetByID retrieves a snapshot reference by ID
func (r *snapshotRepositoryImpl) GetByID(ctx context.Context, snapshotID string) (*domain.SnapshotReference, error) {
	query := `
		SELECT snapshot_id, checksum, created_at, expires_at, status,
			   phase_created, patient_id, context_version, metadata
		FROM snapshots 
		WHERE snapshot_id = $1`

	var snapshot domain.SnapshotReference
	err := r.db.GetContext(ctx, &snapshot, query, snapshotID)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("snapshot not found: %s", snapshotID)
		}
		r.logger.Error("Failed to get snapshot reference",
			zap.String("snapshot_id", snapshotID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get snapshot reference: %w", err)
	}

	return &snapshot, nil
}

// GetByPatientID retrieves recent snapshots for a patient
func (r *snapshotRepositoryImpl) GetByPatientID(ctx context.Context, patientID string, limit int) ([]*domain.SnapshotReference, error) {
	query := `
		SELECT snapshot_id, checksum, created_at, expires_at, status,
			   phase_created, patient_id, context_version, metadata
		FROM snapshots 
		WHERE patient_id = $1 AND status != $2
		ORDER BY created_at DESC
		LIMIT $3`

	var snapshots []*domain.SnapshotReference
	err := r.db.SelectContext(ctx, &snapshots, query, patientID, domain.SnapshotStatusExpired, limit)
	if err != nil {
		r.logger.Error("Failed to get snapshots for patient",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return nil, fmt.Errorf("failed to get snapshots for patient: %w", err)
	}

	r.logger.Info("Retrieved snapshots for patient",
		zap.String("patient_id", patientID),
		zap.Int("count", len(snapshots)))

	return snapshots, nil
}

// UpdateStatus updates the snapshot status
func (r *snapshotRepositoryImpl) UpdateStatus(ctx context.Context, snapshotID string, status domain.SnapshotStatus) error {
	query := `UPDATE snapshots SET status = $1 WHERE snapshot_id = $2`

	result, err := r.db.ExecContext(ctx, query, status, snapshotID)
	if err != nil {
		r.logger.Error("Failed to update snapshot status",
			zap.String("snapshot_id", snapshotID),
			zap.String("status", string(status)),
			zap.Error(err))
		return fmt.Errorf("failed to update snapshot status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Warn("Could not get rows affected for snapshot update",
			zap.String("snapshot_id", snapshotID),
			zap.Error(err))
	} else if rowsAffected == 0 {
		return fmt.Errorf("snapshot not found: %s", snapshotID)
	}

	r.logger.Info("Updated snapshot status",
		zap.String("snapshot_id", snapshotID),
		zap.String("status", string(status)))

	return nil
}

// ExpireSnapshots marks expired snapshots as expired
func (r *snapshotRepositoryImpl) ExpireSnapshots(ctx context.Context, before time.Time) (int64, error) {
	query := `
		UPDATE snapshots 
		SET status = $1 
		WHERE expires_at < $2 AND status = $3`

	result, err := r.db.ExecContext(ctx, query, domain.SnapshotStatusExpired, before, domain.SnapshotStatusActive)
	if err != nil {
		r.logger.Error("Failed to expire snapshots", zap.Error(err))
		return 0, fmt.Errorf("failed to expire snapshots: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		r.logger.Warn("Could not get rows affected for snapshot expiration", zap.Error(err))
		return 0, nil
	}

	if rowsAffected > 0 {
		r.logger.Info("Expired snapshots",
			zap.Int64("count", rowsAffected),
			zap.Time("before", before))
	}

	return rowsAffected, nil
}

// ValidateChecksum verifies the checksum of a snapshot
func (r *snapshotRepositoryImpl) ValidateChecksum(ctx context.Context, snapshotID, expectedChecksum string) (bool, error) {
	query := `SELECT checksum FROM snapshots WHERE snapshot_id = $1`

	var actualChecksum string
	err := r.db.GetContext(ctx, &actualChecksum, query, snapshotID)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, fmt.Errorf("snapshot not found: %s", snapshotID)
		}
		r.logger.Error("Failed to validate snapshot checksum",
			zap.String("snapshot_id", snapshotID),
			zap.Error(err))
		return false, fmt.Errorf("failed to validate snapshot checksum: %w", err)
	}

	isValid := actualChecksum == expectedChecksum
	
	if !isValid {
		r.logger.Warn("Snapshot checksum mismatch",
			zap.String("snapshot_id", snapshotID),
			zap.String("expected", expectedChecksum),
			zap.String("actual", actualChecksum))
	}

	return isValid, nil
}