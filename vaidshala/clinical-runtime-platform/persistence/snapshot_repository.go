// Package persistence provides storage implementations for clinical runtime artifacts.
//
// CRITICAL DESIGN (SaMD Compliance):
// Every clinical decision must be traceable to the exact knowledge state used.
// This package persists KnowledgeSnapshots for:
//   - Audit trail (regulatory requirement)
//   - Reproducibility (re-run with identical inputs)
//   - Debugging (exact patient + knowledge state)
//   - Analytics (population-level insights)
package persistence

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"

	"github.com/google/uuid"
)

// ============================================================================
// SNAPSHOT REPOSITORY INTERFACE
// ============================================================================

// SnapshotRepository defines the interface for KnowledgeSnapshot persistence.
type SnapshotRepository interface {
	// Save persists a KnowledgeSnapshot and returns the snapshot ID
	Save(ctx context.Context, patientID string, snapshot *contracts.KnowledgeSnapshot, opts SaveOptions) (string, error)

	// GetByID retrieves a snapshot by its ID
	GetByID(ctx context.Context, snapshotID string) (*StoredSnapshot, error)

	// GetByRequestID retrieves a snapshot by request ID (for replay)
	GetByRequestID(ctx context.Context, requestID string) (*StoredSnapshot, error)

	// GetLatest retrieves the most recent snapshot for a patient
	GetLatest(ctx context.Context, patientID string) (*StoredSnapshot, error)

	// GetForAudit retrieves snapshots within a time range for audit
	GetForAudit(ctx context.Context, patientID string, start, end time.Time) ([]*StoredSnapshot, error)

	// GetStats returns aggregate statistics about stored snapshots
	GetStats(ctx context.Context) (*SnapshotStats, error)

	// CleanupExpired removes expired snapshots (run periodically)
	CleanupExpired(ctx context.Context) (int, error)
}

// SaveOptions configures snapshot persistence behavior.
type SaveOptions struct {
	// RequestID links snapshot to a specific request (for replay)
	RequestID string

	// EncounterID links snapshot to a clinical encounter
	EncounterID string

	// Region for multi-region support
	Region string

	// CreatedBy user/system identifier
	CreatedBy string

	// ExpiresAt TTL for the snapshot (nil = keep forever)
	ExpiresAt *time.Time
}

// StoredSnapshot wraps KnowledgeSnapshot with storage metadata.
type StoredSnapshot struct {
	// ID unique identifier for this stored snapshot
	ID string `json:"id"`

	// PatientID the snapshot belongs to
	PatientID string `json:"patient_id"`

	// RequestID that triggered this snapshot
	RequestID string `json:"request_id,omitempty"`

	// EncounterID this snapshot relates to
	EncounterID string `json:"encounter_id,omitempty"`

	// Region where snapshot was created
	Region string `json:"region"`

	// Snapshot the actual KnowledgeSnapshot
	Snapshot *contracts.KnowledgeSnapshot `json:"snapshot"`

	// CreatedAt when the snapshot was stored
	CreatedAt time.Time `json:"created_at"`

	// CreatedBy who/what created the snapshot
	CreatedBy string `json:"created_by,omitempty"`

	// ExpiresAt when the snapshot can be deleted (nil = never)
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// SnapshotStats provides aggregate statistics about stored snapshots.
type SnapshotStats struct {
	TotalSnapshots     int64   `json:"total_snapshots"`
	SnapshotsToday     int64   `json:"snapshots_today"`
	SnapshotsThisWeek  int64   `json:"snapshots_this_week"`
	AverageEGFR        float64 `json:"average_egfr"`
	PatientsWithCKD    int64   `json:"patients_with_ckd"`
	PatientsOnAnticoag int64   `json:"patients_on_anticoagulation"`
}

// ============================================================================
// POSTGRESQL IMPLEMENTATION
// ============================================================================

// PostgresSnapshotRepository implements SnapshotRepository using PostgreSQL.
type PostgresSnapshotRepository struct {
	db *sql.DB
}

// NewPostgresSnapshotRepository creates a new PostgreSQL-backed repository.
func NewPostgresSnapshotRepository(db *sql.DB) *PostgresSnapshotRepository {
	return &PostgresSnapshotRepository{db: db}
}

// Save persists a KnowledgeSnapshot to PostgreSQL.
func (r *PostgresSnapshotRepository) Save(
	ctx context.Context,
	patientID string,
	snapshot *contracts.KnowledgeSnapshot,
	opts SaveOptions,
) (string, error) {

	// Generate unique ID
	snapshotID := uuid.New().String()

	// Serialize snapshot to JSON
	snapshotJSON, err := json.Marshal(snapshot)
	if err != nil {
		return "", fmt.Errorf("failed to serialize snapshot: %w", err)
	}

	// Serialize KB versions
	kbVersionsJSON, err := json.Marshal(snapshot.KBVersions)
	if err != nil {
		return "", fmt.Errorf("failed to serialize KB versions: %w", err)
	}

	// Serialize clinical flags
	clinicalFlagsJSON, err := json.Marshal(snapshot.Terminology.ValueSetMemberships)
	if err != nil {
		return "", fmt.Errorf("failed to serialize clinical flags: %w", err)
	}

	// Extract calculator values for indexing
	var egfrValue, egfrCategory sql.NullString
	var cha2Score, hasBledScore sql.NullInt32

	if snapshot.Calculators.EGFR != nil {
		egfrValue = sql.NullString{String: fmt.Sprintf("%.2f", snapshot.Calculators.EGFR.Value), Valid: true}
		egfrCategory = sql.NullString{String: snapshot.Calculators.EGFR.Category, Valid: true}
	}
	if snapshot.Calculators.CHA2DS2VASc != nil {
		cha2Score = sql.NullInt32{Int32: int32(snapshot.Calculators.CHA2DS2VASc.Value), Valid: true}
	}
	if snapshot.Calculators.HASBLED != nil {
		hasBledScore = sql.NullInt32{Int32: int32(snapshot.Calculators.HASBLED.Value), Valid: true}
	}

	// Count conditions and medications
	conditionCount := len(snapshot.Terminology.PatientConditionCodes)
	medicationCount := len(snapshot.Terminology.PatientMedicationCodes)

	// Default region
	region := opts.Region
	if region == "" {
		region = "AU"
	}

	// Insert snapshot
	query := `
		INSERT INTO knowledge_snapshots (
			id, patient_id, region, request_id, encounter_id,
			snapshot_jsonb, kb_versions, snapshot_version, snapshot_timestamp,
			egfr_value, egfr_category, cha2ds2vasc_score, hasbled_score,
			condition_count, medication_count, clinical_flags,
			created_at, created_by, expires_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7, $8, $9,
			$10, $11, $12, $13,
			$14, $15, $16,
			$17, $18, $19
		)
	`

	_, err = r.db.ExecContext(ctx, query,
		snapshotID, patientID, region, nullString(opts.RequestID), nullString(opts.EncounterID),
		snapshotJSON, kbVersionsJSON, snapshot.SnapshotVersion, snapshot.SnapshotTimestamp,
		egfrValue, egfrCategory, cha2Score, hasBledScore,
		conditionCount, medicationCount, clinicalFlagsJSON,
		time.Now(), nullString(opts.CreatedBy), opts.ExpiresAt,
	)

	if err != nil {
		return "", fmt.Errorf("failed to insert snapshot: %w", err)
	}

	return snapshotID, nil
}

// GetByID retrieves a snapshot by its ID.
func (r *PostgresSnapshotRepository) GetByID(ctx context.Context, snapshotID string) (*StoredSnapshot, error) {
	query := `
		SELECT id, patient_id, request_id, encounter_id, region,
		       snapshot_jsonb, created_at, created_by, expires_at
		FROM knowledge_snapshots
		WHERE id = $1
	`

	return r.scanStoredSnapshot(r.db.QueryRowContext(ctx, query, snapshotID))
}

// GetByRequestID retrieves a snapshot by request ID.
func (r *PostgresSnapshotRepository) GetByRequestID(ctx context.Context, requestID string) (*StoredSnapshot, error) {
	query := `
		SELECT id, patient_id, request_id, encounter_id, region,
		       snapshot_jsonb, created_at, created_by, expires_at
		FROM knowledge_snapshots
		WHERE request_id = $1
		LIMIT 1
	`

	return r.scanStoredSnapshot(r.db.QueryRowContext(ctx, query, requestID))
}

// GetLatest retrieves the most recent snapshot for a patient.
func (r *PostgresSnapshotRepository) GetLatest(ctx context.Context, patientID string) (*StoredSnapshot, error) {
	query := `
		SELECT id, patient_id, request_id, encounter_id, region,
		       snapshot_jsonb, created_at, created_by, expires_at
		FROM knowledge_snapshots
		WHERE patient_id = $1
		ORDER BY created_at DESC
		LIMIT 1
	`

	return r.scanStoredSnapshot(r.db.QueryRowContext(ctx, query, patientID))
}

// GetForAudit retrieves snapshots within a time range for audit.
func (r *PostgresSnapshotRepository) GetForAudit(
	ctx context.Context,
	patientID string,
	start, end time.Time,
) ([]*StoredSnapshot, error) {

	query := `
		SELECT id, patient_id, request_id, encounter_id, region,
		       snapshot_jsonb, created_at, created_by, expires_at
		FROM knowledge_snapshots
		WHERE patient_id = $1
		  AND created_at BETWEEN $2 AND $3
		ORDER BY created_at DESC
	`

	rows, err := r.db.QueryContext(ctx, query, patientID, start, end)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer rows.Close()

	var snapshots []*StoredSnapshot
	for rows.Next() {
		snapshot, err := r.scanStoredSnapshotFromRows(rows)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, rows.Err()
}

// GetStats returns aggregate statistics about stored snapshots.
func (r *PostgresSnapshotRepository) GetStats(ctx context.Context) (*SnapshotStats, error) {
	query := `SELECT * FROM get_snapshot_stats()`

	var stats SnapshotStats
	var avgEGFR sql.NullFloat64

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalSnapshots,
		&stats.SnapshotsToday,
		&stats.SnapshotsThisWeek,
		&avgEGFR,
		&stats.PatientsWithCKD,
		&stats.PatientsOnAnticoag,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get snapshot stats: %w", err)
	}

	if avgEGFR.Valid {
		stats.AverageEGFR = avgEGFR.Float64
	}

	return &stats, nil
}

// CleanupExpired removes expired snapshots.
func (r *PostgresSnapshotRepository) CleanupExpired(ctx context.Context) (int, error) {
	query := `SELECT cleanup_expired_snapshots()`

	var count int
	err := r.db.QueryRowContext(ctx, query).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to cleanup expired snapshots: %w", err)
	}

	return count, nil
}

// ============================================================================
// HELPER METHODS
// ============================================================================

// scanStoredSnapshot scans a single row into a StoredSnapshot.
func (r *PostgresSnapshotRepository) scanStoredSnapshot(row *sql.Row) (*StoredSnapshot, error) {
	var stored StoredSnapshot
	var snapshotJSON []byte
	var requestID, encounterID, createdBy sql.NullString
	var expiresAt sql.NullTime

	err := row.Scan(
		&stored.ID,
		&stored.PatientID,
		&requestID,
		&encounterID,
		&stored.Region,
		&snapshotJSON,
		&stored.CreatedAt,
		&createdBy,
		&expiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan snapshot: %w", err)
	}

	// Parse nullable fields
	if requestID.Valid {
		stored.RequestID = requestID.String
	}
	if encounterID.Valid {
		stored.EncounterID = encounterID.String
	}
	if createdBy.Valid {
		stored.CreatedBy = createdBy.String
	}
	if expiresAt.Valid {
		stored.ExpiresAt = &expiresAt.Time
	}

	// Deserialize snapshot JSON
	stored.Snapshot = &contracts.KnowledgeSnapshot{}
	if err := json.Unmarshal(snapshotJSON, stored.Snapshot); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}

	return &stored, nil
}

// scanStoredSnapshotFromRows scans a row from *sql.Rows into a StoredSnapshot.
func (r *PostgresSnapshotRepository) scanStoredSnapshotFromRows(rows *sql.Rows) (*StoredSnapshot, error) {
	var stored StoredSnapshot
	var snapshotJSON []byte
	var requestID, encounterID, createdBy sql.NullString
	var expiresAt sql.NullTime

	err := rows.Scan(
		&stored.ID,
		&stored.PatientID,
		&requestID,
		&encounterID,
		&stored.Region,
		&snapshotJSON,
		&stored.CreatedAt,
		&createdBy,
		&expiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan snapshot row: %w", err)
	}

	// Parse nullable fields
	if requestID.Valid {
		stored.RequestID = requestID.String
	}
	if encounterID.Valid {
		stored.EncounterID = encounterID.String
	}
	if createdBy.Valid {
		stored.CreatedBy = createdBy.String
	}
	if expiresAt.Valid {
		stored.ExpiresAt = &expiresAt.Time
	}

	// Deserialize snapshot JSON
	stored.Snapshot = &contracts.KnowledgeSnapshot{}
	if err := json.Unmarshal(snapshotJSON, stored.Snapshot); err != nil {
		return nil, fmt.Errorf("failed to deserialize snapshot: %w", err)
	}

	return &stored, nil
}

// nullString converts an empty string to sql.NullString.
func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}
