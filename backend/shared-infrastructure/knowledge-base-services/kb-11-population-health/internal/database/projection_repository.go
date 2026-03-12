// Package database provides PostgreSQL database connectivity for KB-11.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// ProjectionRepository handles database operations for patient projections.
// IMPORTANT: This is a read-through cache repository. Patient data is NOT authoritative.
type ProjectionRepository struct {
	db     *DB
	logger *logrus.Entry
}

// NewProjectionRepository creates a new projection repository.
func NewProjectionRepository(db *DB, logger *logrus.Entry) *ProjectionRepository {
	return &ProjectionRepository{
		db:     db,
		logger: logger.WithField("repository", "projection"),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Patient Projection Operations
// ──────────────────────────────────────────────────────────────────────────────

// GetByFHIRID retrieves a patient projection by FHIR ID.
func (r *ProjectionRepository) GetByFHIRID(ctx context.Context, fhirID string) (*models.PatientProjection, error) {
	query := `
		SELECT id, fhir_id, kb17_patient_id, mrn,
			   first_name, last_name, date_of_birth, gender,
			   attributed_pcp, attributed_practice, attribution_date,
			   current_risk_tier, latest_risk_score, care_gap_count,
			   last_synced_at, sync_source, sync_version,
			   created_at, updated_at
		FROM patient_projections
		WHERE fhir_id = $1`

	return r.scanPatientProjection(r.db.QueryRowContext(ctx, query, fhirID))
}

// GetByID retrieves a patient projection by internal ID.
func (r *ProjectionRepository) GetByID(ctx context.Context, id uuid.UUID) (*models.PatientProjection, error) {
	query := `
		SELECT id, fhir_id, kb17_patient_id, mrn,
			   first_name, last_name, date_of_birth, gender,
			   attributed_pcp, attributed_practice, attribution_date,
			   current_risk_tier, latest_risk_score, care_gap_count,
			   last_synced_at, sync_source, sync_version,
			   created_at, updated_at
		FROM patient_projections
		WHERE id = $1`

	return r.scanPatientProjection(r.db.QueryRowContext(ctx, query, id))
}

// Query retrieves patient projections with filtering and pagination.
func (r *ProjectionRepository) Query(ctx context.Context, req *models.PatientQueryRequest) ([]*models.PatientProjection, int, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.RiskTier != nil {
		conditions = append(conditions, fmt.Sprintf("current_risk_tier = $%d", argIndex))
		args = append(args, string(*req.RiskTier))
		argIndex++
	}

	if req.AttributedPCP != nil {
		conditions = append(conditions, fmt.Sprintf("attributed_pcp = $%d", argIndex))
		args = append(args, *req.AttributedPCP)
		argIndex++
	}

	if req.AttributedPractice != nil {
		conditions = append(conditions, fmt.Sprintf("attributed_practice = $%d", argIndex))
		args = append(args, *req.AttributedPractice)
		argIndex++
	}

	if req.MinCareGaps != nil {
		conditions = append(conditions, fmt.Sprintf("care_gap_count >= $%d", argIndex))
		args = append(args, *req.MinCareGaps)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	// Build ORDER BY clause
	orderColumn := "latest_risk_score"
	switch req.SortBy {
	case "care_gaps":
		orderColumn = "care_gap_count"
	case "last_synced":
		orderColumn = "last_synced_at"
	case "created_at":
		orderColumn = "created_at"
	}

	orderDir := "DESC"
	if req.SortOrder == "asc" {
		orderDir = "ASC"
	}

	// Get total count
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM patient_projections %s", whereClause)
	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("failed to count projections: %w", err)
	}

	// Get results with pagination
	query := fmt.Sprintf(`
		SELECT id, fhir_id, kb17_patient_id, mrn,
			   first_name, last_name, date_of_birth, gender,
			   attributed_pcp, attributed_practice, attribution_date,
			   current_risk_tier, latest_risk_score, care_gap_count,
			   last_synced_at, sync_source, sync_version,
			   created_at, updated_at
		FROM patient_projections
		%s
		ORDER BY %s %s NULLS LAST
		LIMIT $%d OFFSET $%d`,
		whereClause, orderColumn, orderDir, argIndex, argIndex+1)

	args = append(args, req.Limit, req.Offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query projections: %w", err)
	}
	defer rows.Close()

	projections := []*models.PatientProjection{}
	for rows.Next() {
		pp, err := r.scanPatientProjectionFromRows(rows)
		if err != nil {
			return nil, 0, err
		}
		projections = append(projections, pp)
	}

	return projections, total, rows.Err()
}

// Upsert creates or updates a patient projection.
// NOTE: This is used for syncing data from upstream sources (FHIR, KB-17).
func (r *ProjectionRepository) Upsert(ctx context.Context, pp *models.PatientProjection) error {
	query := `
		INSERT INTO patient_projections (
			id, fhir_id, kb17_patient_id, mrn,
			first_name, last_name, date_of_birth, gender,
			attributed_pcp, attributed_practice, attribution_date,
			current_risk_tier, latest_risk_score, care_gap_count,
			last_synced_at, sync_source, sync_version,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
			$12, $13, $14, $15, $16, $17, $18, $19
		)
		ON CONFLICT (fhir_id) DO UPDATE SET
			kb17_patient_id = EXCLUDED.kb17_patient_id,
			mrn = EXCLUDED.mrn,
			first_name = EXCLUDED.first_name,
			last_name = EXCLUDED.last_name,
			date_of_birth = EXCLUDED.date_of_birth,
			gender = EXCLUDED.gender,
			last_synced_at = EXCLUDED.last_synced_at,
			sync_source = EXCLUDED.sync_source,
			sync_version = patient_projections.sync_version + 1,
			updated_at = NOW()`

	_, err := r.db.ExecContext(ctx, query,
		pp.ID, pp.FHIRID, pp.KB17PatientID, pp.MRN,
		pp.FirstName, pp.LastName, pp.DateOfBirth, genderPtr(pp.Gender),
		pp.AttributedPCP, pp.AttributedPractice, pp.AttributionDate,
		string(pp.CurrentRiskTier), pp.LatestRiskScore, pp.CareGapCount,
		pp.LastSyncedAt, string(pp.SyncSource), pp.SyncVersion,
		pp.CreatedAt, pp.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert projection: %w", err)
	}

	return nil
}

// UpdateRiskTier updates only the risk tier and score for a patient.
// KB-11 OWNS this data - this is where risk calculations are stored.
func (r *ProjectionRepository) UpdateRiskTier(ctx context.Context, fhirID string, tier models.RiskTier, score float64) error {
	query := `
		UPDATE patient_projections
		SET current_risk_tier = $2, latest_risk_score = $3, updated_at = NOW()
		WHERE fhir_id = $1`

	result, err := r.db.ExecContext(ctx, query, fhirID, string(tier), score)
	if err != nil {
		return fmt.Errorf("failed to update risk tier: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("patient not found: %s", fhirID)
	}

	return nil
}

// UpdateCareGapCount updates the aggregated care gap count from KB-13.
// NOTE: This is a cached count, NOT source of truth.
func (r *ProjectionRepository) UpdateCareGapCount(ctx context.Context, fhirID string, count int) error {
	query := `
		UPDATE patient_projections
		SET care_gap_count = $2, updated_at = NOW()
		WHERE fhir_id = $1`

	_, err := r.db.ExecContext(ctx, query, fhirID, count)
	return err
}

// UpdateAttribution updates patient attribution data.
// KB-11 OWNS attribution data.
func (r *ProjectionRepository) UpdateAttribution(ctx context.Context, req *models.AttributionUpdateRequest) error {
	query := `
		UPDATE patient_projections
		SET attributed_pcp = COALESCE($2, attributed_pcp),
			attributed_practice = COALESCE($3, attributed_practice),
			attribution_date = COALESCE($4, attribution_date),
			updated_at = NOW()
		WHERE fhir_id = $1`

	result, err := r.db.ExecContext(ctx, query,
		req.PatientFHIRID, req.AttributedPCP, req.AttributedPractice, req.AttributionDate)
	if err != nil {
		return fmt.Errorf("failed to update attribution: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("patient not found: %s", req.PatientFHIRID)
	}

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Risk Assessment Operations
// ──────────────────────────────────────────────────────────────────────────────

// GetLatestRiskAssessment retrieves the latest risk assessment for a patient.
func (r *ProjectionRepository) GetLatestRiskAssessment(ctx context.Context, patientFHIRID, modelName string) (*models.RiskAssessment, error) {
	query := `
		SELECT id, patient_fhir_id, model_name, model_version,
			   score, risk_tier, contributing_factors,
			   input_hash, calculation_hash, governance_event_id,
			   calculated_at, valid_until
		FROM risk_assessments
		WHERE patient_fhir_id = $1 AND model_name = $2`

	return r.scanRiskAssessment(r.db.QueryRowContext(ctx, query, patientFHIRID, modelName))
}

// SaveRiskAssessment saves a new risk assessment and archives the old one.
func (r *ProjectionRepository) SaveRiskAssessment(ctx context.Context, ra *models.RiskAssessment) error {
	return r.db.WithTransaction(ctx, func(tx *sql.Tx) error {
		// Archive existing assessment if present
		archiveQuery := `
			INSERT INTO risk_assessment_history (
				id, assessment_id, patient_fhir_id, model_name, model_version,
				score, risk_tier, contributing_factors,
				input_hash, calculation_hash, governance_event_id,
				calculated_at, archived_at
			)
			SELECT gen_random_uuid(), id, patient_fhir_id, model_name, model_version,
				   score, risk_tier, contributing_factors,
				   input_hash, calculation_hash, governance_event_id,
				   calculated_at, NOW()
			FROM risk_assessments
			WHERE patient_fhir_id = $1 AND model_name = $2`

		_, err := tx.ExecContext(ctx, archiveQuery, ra.PatientFHIRID, ra.ModelName)
		if err != nil {
			return fmt.Errorf("failed to archive assessment: %w", err)
		}

		// Convert contributing factors to JSON
		factorsJSON, err := json.Marshal(ra.ContributingFactors)
		if err != nil {
			return fmt.Errorf("failed to marshal contributing factors: %w", err)
		}

		// Insert new assessment (upsert)
		insertQuery := `
			INSERT INTO risk_assessments (
				id, patient_fhir_id, model_name, model_version,
				score, risk_tier, contributing_factors,
				input_hash, calculation_hash, governance_event_id,
				calculated_at, valid_until
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
			ON CONFLICT (patient_fhir_id, model_name) DO UPDATE SET
				model_version = EXCLUDED.model_version,
				score = EXCLUDED.score,
				risk_tier = EXCLUDED.risk_tier,
				contributing_factors = EXCLUDED.contributing_factors,
				input_hash = EXCLUDED.input_hash,
				calculation_hash = EXCLUDED.calculation_hash,
				governance_event_id = EXCLUDED.governance_event_id,
				calculated_at = EXCLUDED.calculated_at,
				valid_until = EXCLUDED.valid_until`

		_, err = tx.ExecContext(ctx, insertQuery,
			ra.ID, ra.PatientFHIRID, ra.ModelName, ra.ModelVersion,
			ra.Score, string(ra.RiskTier), factorsJSON,
			ra.InputHash, ra.CalculationHash, ra.GovernanceEventID,
			ra.CalculatedAt, ra.ValidUntil,
		)
		if err != nil {
			return fmt.Errorf("failed to save assessment: %w", err)
		}

		return nil
	})
}

// GetRiskHistory retrieves the risk assessment history for a patient.
func (r *ProjectionRepository) GetRiskHistory(ctx context.Context, patientFHIRID string, limit int) ([]*models.RiskAssessmentHistory, error) {
	query := `
		SELECT id, assessment_id, patient_fhir_id, model_name, model_version,
			   score, risk_tier, contributing_factors,
			   input_hash, calculation_hash, governance_event_id,
			   calculated_at, archived_at
		FROM risk_assessment_history
		WHERE patient_fhir_id = $1
		ORDER BY calculated_at DESC
		LIMIT $2`

	rows, err := r.db.QueryContext(ctx, query, patientFHIRID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query risk history: %w", err)
	}
	defer rows.Close()

	history := []*models.RiskAssessmentHistory{}
	for rows.Next() {
		var h models.RiskAssessmentHistory
		var factorsJSON []byte
		var tier string

		err := rows.Scan(
			&h.ID, &h.AssessmentID, &h.PatientFHIRID, &h.ModelName, &h.ModelVersion,
			&h.Score, &tier, &factorsJSON,
			&h.InputHash, &h.CalculationHash, &h.GovernanceEventID,
			&h.CalculatedAt, &h.ArchivedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan risk history: %w", err)
		}

		h.RiskTier = models.RiskTier(tier)
		if factorsJSON != nil {
			json.Unmarshal(factorsJSON, &h.ContributingFactors)
		}

		history = append(history, &h)
	}

	return history, rows.Err()
}

// ──────────────────────────────────────────────────────────────────────────────
// Population Metrics Operations
// ──────────────────────────────────────────────────────────────────────────────

// GetPopulationMetrics calculates population-level metrics.
// This is the CORE PURPOSE of KB-11 - answering population-level questions.
func (r *ProjectionRepository) GetPopulationMetrics(ctx context.Context, req *models.PopulationMetricsRequest) (*models.PopulationMetrics, error) {
	// Build WHERE clause
	conditions := []string{}
	args := []interface{}{}
	argIndex := 1

	if req.AttributedPCP != nil {
		conditions = append(conditions, fmt.Sprintf("attributed_pcp = $%d", argIndex))
		args = append(args, *req.AttributedPCP)
		argIndex++
	}

	if req.AttributedPractice != nil {
		conditions = append(conditions, fmt.Sprintf("attributed_practice = $%d", argIndex))
		args = append(args, *req.AttributedPractice)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + strings.Join(conditions, " AND ")
	}

	metrics := &models.PopulationMetrics{
		RiskDistribution:    make(map[models.RiskTier]int),
		CareGapDistribution: make(map[string]int),
		CalculatedAt:        time.Now(),
	}

	// Get total count and average risk score
	summaryQuery := fmt.Sprintf(`
		SELECT COUNT(*), COALESCE(AVG(latest_risk_score), 0)
		FROM patient_projections
		%s`, whereClause)

	err := r.db.QueryRowContext(ctx, summaryQuery, args...).Scan(&metrics.TotalPatients, &metrics.AverageRiskScore)
	if err != nil {
		return nil, fmt.Errorf("failed to get population summary: %w", err)
	}

	// Get risk tier distribution
	tierQuery := fmt.Sprintf(`
		SELECT current_risk_tier, COUNT(*)
		FROM patient_projections
		%s
		GROUP BY current_risk_tier`, whereClause)

	rows, err := r.db.QueryContext(ctx, tierQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get tier distribution: %w", err)
	}
	defer rows.Close()

	highRiskCount := 0
	for rows.Next() {
		var tier string
		var count int
		if err := rows.Scan(&tier, &count); err != nil {
			return nil, err
		}
		riskTier := models.RiskTier(tier)
		metrics.RiskDistribution[riskTier] = count

		if riskTier == models.RiskTierHigh || riskTier == models.RiskTierVeryHigh {
			highRiskCount += count
		}
		if riskTier == models.RiskTierRising {
			metrics.RisingRiskCount = count
		}
	}

	if metrics.TotalPatients > 0 {
		metrics.HighRiskPercentage = float64(highRiskCount) / float64(metrics.TotalPatients) * 100
	}

	// Get groupings if requested
	if req.GroupBy == "practice" {
		metrics.ByPractice, err = r.getGroupCounts(ctx, "attributed_practice", whereClause, args)
		if err != nil {
			return nil, err
		}
	}

	if req.GroupBy == "pcp" {
		metrics.ByPCP, err = r.getGroupCounts(ctx, "attributed_pcp", whereClause, args)
		if err != nil {
			return nil, err
		}
	}

	return metrics, nil
}

func (r *ProjectionRepository) getGroupCounts(ctx context.Context, column, whereClause string, args []interface{}) (map[string]int, error) {
	query := fmt.Sprintf(`
		SELECT COALESCE(%s, 'Unassigned'), COUNT(*)
		FROM patient_projections
		%s
		GROUP BY %s`, column, whereClause, column)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var group string
		var count int
		if err := rows.Scan(&group, &count); err != nil {
			return nil, err
		}
		counts[group] = count
	}

	return counts, rows.Err()
}

// ──────────────────────────────────────────────────────────────────────────────
// Sync Status Operations
// ──────────────────────────────────────────────────────────────────────────────

// GetSyncStatus retrieves the sync status for a source.
func (r *ProjectionRepository) GetSyncStatus(ctx context.Context, source models.SyncSource) (*models.SyncStatusRecord, error) {
	query := `
		SELECT id, source, last_sync_started, last_sync_completed,
			   last_sync_status, records_synced, error_message,
			   created_at, updated_at
		FROM sync_status
		WHERE source = $1`

	var ss models.SyncStatusRecord
	var status string

	err := r.db.QueryRowContext(ctx, query, string(source)).Scan(
		&ss.ID, &status, &ss.LastSyncStarted, &ss.LastSyncCompleted,
		&status, &ss.RecordsSynced, &ss.ErrorMessage,
		&ss.CreatedAt, &ss.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get sync status: %w", err)
	}

	ss.Source = models.SyncSource(source)
	ss.LastSyncStatus = models.SyncStatus(status)

	return &ss, nil
}

// UpdateSyncStatus updates the sync status for a source.
func (r *ProjectionRepository) UpdateSyncStatus(ctx context.Context, source models.SyncSource, status models.SyncStatus, recordsSynced int, errMsg *string) error {
	query := `
		UPDATE sync_status
		SET last_sync_status = $2,
			last_sync_completed = CASE WHEN $2 IN ('SUCCESS', 'FAILED') THEN NOW() ELSE last_sync_completed END,
			records_synced = $3,
			error_message = $4,
			updated_at = NOW()
		WHERE source = $1`

	_, err := r.db.ExecContext(ctx, query, string(source), string(status), recordsSynced, errMsg)
	return err
}

// StartSync marks a sync operation as started.
func (r *ProjectionRepository) StartSync(ctx context.Context, source models.SyncSource) error {
	query := `
		UPDATE sync_status
		SET last_sync_started = NOW(),
			last_sync_status = 'IN_PROGRESS',
			error_message = NULL,
			updated_at = NOW()
		WHERE source = $1`

	_, err := r.db.ExecContext(ctx, query, string(source))
	return err
}

// ──────────────────────────────────────────────────────────────────────────────
// Helper Functions
// ──────────────────────────────────────────────────────────────────────────────

func (r *ProjectionRepository) scanPatientProjection(row *sql.Row) (*models.PatientProjection, error) {
	var pp models.PatientProjection
	var tier, syncSource string
	var gender *string

	err := row.Scan(
		&pp.ID, &pp.FHIRID, &pp.KB17PatientID, &pp.MRN,
		&pp.FirstName, &pp.LastName, &pp.DateOfBirth, &gender,
		&pp.AttributedPCP, &pp.AttributedPractice, &pp.AttributionDate,
		&tier, &pp.LatestRiskScore, &pp.CareGapCount,
		&pp.LastSyncedAt, &syncSource, &pp.SyncVersion,
		&pp.CreatedAt, &pp.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan patient projection: %w", err)
	}

	pp.CurrentRiskTier = models.RiskTier(tier)
	pp.SyncSource = models.SyncSource(syncSource)
	if gender != nil {
		g := models.Gender(*gender)
		pp.Gender = &g
	}

	return &pp, nil
}

func (r *ProjectionRepository) scanPatientProjectionFromRows(rows *sql.Rows) (*models.PatientProjection, error) {
	var pp models.PatientProjection
	var tier, syncSource string
	var gender *string

	err := rows.Scan(
		&pp.ID, &pp.FHIRID, &pp.KB17PatientID, &pp.MRN,
		&pp.FirstName, &pp.LastName, &pp.DateOfBirth, &gender,
		&pp.AttributedPCP, &pp.AttributedPractice, &pp.AttributionDate,
		&tier, &pp.LatestRiskScore, &pp.CareGapCount,
		&pp.LastSyncedAt, &syncSource, &pp.SyncVersion,
		&pp.CreatedAt, &pp.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to scan patient projection: %w", err)
	}

	pp.CurrentRiskTier = models.RiskTier(tier)
	pp.SyncSource = models.SyncSource(syncSource)
	if gender != nil {
		g := models.Gender(*gender)
		pp.Gender = &g
	}

	return &pp, nil
}

func (r *ProjectionRepository) scanRiskAssessment(row *sql.Row) (*models.RiskAssessment, error) {
	var ra models.RiskAssessment
	var tier string
	var factorsJSON []byte

	err := row.Scan(
		&ra.ID, &ra.PatientFHIRID, &ra.ModelName, &ra.ModelVersion,
		&ra.Score, &tier, &factorsJSON,
		&ra.InputHash, &ra.CalculationHash, &ra.GovernanceEventID,
		&ra.CalculatedAt, &ra.ValidUntil,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to scan risk assessment: %w", err)
	}

	ra.RiskTier = models.RiskTier(tier)
	if factorsJSON != nil {
		json.Unmarshal(factorsJSON, &ra.ContributingFactors)
	}

	return &ra, nil
}

func genderPtr(g *models.Gender) *string {
	if g == nil {
		return nil
	}
	s := string(*g)
	return &s
}

// ──────────────────────────────────────────────────────────────────────────────
// Analytics Support Methods (Phase D)
// ──────────────────────────────────────────────────────────────────────────────

// GetRiskDistribution returns the count of patients in each risk tier.
func (r *ProjectionRepository) GetRiskDistribution(ctx context.Context) (map[models.RiskTier]int, error) {
	query := `
		SELECT current_risk_tier, COUNT(*)
		FROM patient_projections
		GROUP BY current_risk_tier`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get risk distribution: %w", err)
	}
	defer rows.Close()

	distribution := make(map[models.RiskTier]int)
	for rows.Next() {
		var tier string
		var count int
		if err := rows.Scan(&tier, &count); err != nil {
			return nil, err
		}
		distribution[models.RiskTier(tier)] = count
	}

	return distribution, rows.Err()
}

// GetAverageRiskScore returns the average risk score across all patients.
func (r *ProjectionRepository) GetAverageRiskScore(ctx context.Context) (float64, error) {
	query := `SELECT COALESCE(AVG(latest_risk_score), 0) FROM patient_projections WHERE latest_risk_score IS NOT NULL`

	var avg float64
	if err := r.db.QueryRowContext(ctx, query).Scan(&avg); err != nil {
		return 0, fmt.Errorf("failed to get average risk score: %w", err)
	}
	return avg, nil
}

// GetHighRiskPercentage returns the percentage of patients in high/very-high risk tiers.
func (r *ProjectionRepository) GetHighRiskPercentage(ctx context.Context) (float64, error) {
	query := `
		SELECT
			CASE WHEN COUNT(*) > 0
				THEN (SUM(CASE WHEN current_risk_tier IN ('HIGH', 'VERY_HIGH') THEN 1 ELSE 0 END)::float / COUNT(*)::float) * 100
				ELSE 0
			END
		FROM patient_projections`

	var pct float64
	if err := r.db.QueryRowContext(ctx, query).Scan(&pct); err != nil {
		return 0, fmt.Errorf("failed to get high risk percentage: %w", err)
	}
	return pct, nil
}

// GetPatientsByPCP returns patient counts grouped by PCP.
func (r *ProjectionRepository) GetPatientsByPCP(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT COALESCE(attributed_pcp, 'Unattributed'), COUNT(*)
		FROM patient_projections
		GROUP BY attributed_pcp`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get patients by PCP: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var pcp string
		var count int
		if err := rows.Scan(&pcp, &count); err != nil {
			return nil, err
		}
		counts[pcp] = count
	}

	return counts, rows.Err()
}

// GetPatientsByPractice returns patient counts grouped by practice.
func (r *ProjectionRepository) GetPatientsByPractice(ctx context.Context) (map[string]int, error) {
	query := `
		SELECT COALESCE(attributed_practice, 'Unattributed'), COUNT(*)
		FROM patient_projections
		GROUP BY attributed_practice`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get patients by practice: %w", err)
	}
	defer rows.Close()

	counts := make(map[string]int)
	for rows.Next() {
		var practice string
		var count int
		if err := rows.Scan(&practice, &count); err != nil {
			return nil, err
		}
		counts[practice] = count
	}

	return counts, rows.Err()
}

// GetUnattributedCount returns the count of patients without PCP attribution.
func (r *ProjectionRepository) GetUnattributedCount(ctx context.Context) (int, error) {
	query := `SELECT COUNT(*) FROM patient_projections WHERE attributed_pcp IS NULL`

	var count int
	if err := r.db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, fmt.Errorf("failed to get unattributed count: %w", err)
	}
	return count, nil
}

// GetRisingRiskPatients returns patients with rising risk tier.
func (r *ProjectionRepository) GetRisingRiskPatients(ctx context.Context, limit int) ([]*models.PatientProjection, error) {
	query := `
		SELECT id, fhir_id, kb17_patient_id, mrn,
			   first_name, last_name, date_of_birth, gender,
			   attributed_pcp, attributed_practice, attribution_date,
			   current_risk_tier, latest_risk_score, care_gap_count,
			   last_synced_at, sync_source, sync_version,
			   created_at, updated_at
		FROM patient_projections
		WHERE current_risk_tier = 'RISING'
		ORDER BY latest_risk_score DESC NULLS LAST
		LIMIT $1`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to get rising risk patients: %w", err)
	}
	defer rows.Close()

	var patients []*models.PatientProjection
	for rows.Next() {
		pp, err := r.scanPatientProjectionFromRows(rows)
		if err != nil {
			return nil, err
		}
		patients = append(patients, pp)
	}

	return patients, rows.Err()
}

// GetPatientsByAttributedPCP returns all patients attributed to a specific PCP.
func (r *ProjectionRepository) GetPatientsByAttributedPCP(ctx context.Context, pcp string) ([]*models.PatientProjection, error) {
	query := `
		SELECT id, fhir_id, kb17_patient_id, mrn,
			   first_name, last_name, date_of_birth, gender,
			   attributed_pcp, attributed_practice, attribution_date,
			   current_risk_tier, latest_risk_score, care_gap_count,
			   last_synced_at, sync_source, sync_version,
			   created_at, updated_at
		FROM patient_projections
		WHERE attributed_pcp = $1`

	rows, err := r.db.QueryContext(ctx, query, pcp)
	if err != nil {
		return nil, fmt.Errorf("failed to get patients by PCP: %w", err)
	}
	defer rows.Close()

	var patients []*models.PatientProjection
	for rows.Next() {
		pp, err := r.scanPatientProjectionFromRows(rows)
		if err != nil {
			return nil, err
		}
		patients = append(patients, pp)
	}

	return patients, rows.Err()
}

// GetPatientsByAttributedPractice returns all patients attributed to a specific practice.
func (r *ProjectionRepository) GetPatientsByAttributedPractice(ctx context.Context, practice string) ([]*models.PatientProjection, error) {
	query := `
		SELECT id, fhir_id, kb17_patient_id, mrn,
			   first_name, last_name, date_of_birth, gender,
			   attributed_pcp, attributed_practice, attribution_date,
			   current_risk_tier, latest_risk_score, care_gap_count,
			   last_synced_at, sync_source, sync_version,
			   created_at, updated_at
		FROM patient_projections
		WHERE attributed_practice = $1`

	rows, err := r.db.QueryContext(ctx, query, practice)
	if err != nil {
		return nil, fmt.Errorf("failed to get patients by practice: %w", err)
	}
	defer rows.Close()

	var patients []*models.PatientProjection
	for rows.Next() {
		pp, err := r.scanPatientProjectionFromRows(rows)
		if err != nil {
			return nil, err
		}
		patients = append(patients, pp)
	}

	return patients, rows.Err()
}

// ──────────────────────────────────────────────────────────────────────────────
// Historical Snapshot (for Trend Analysis)
// ──────────────────────────────────────────────────────────────────────────────

// HistoricalSnapshot represents a point-in-time population snapshot.
type HistoricalSnapshot struct {
	SnapshotDate     time.Time `json:"snapshot_date"`
	TotalPatients    int       `json:"total_patients"`
	HighRiskCount    int       `json:"high_risk_count"`
	RisingRiskCount  int       `json:"rising_risk_count"`
	AverageRiskScore float64   `json:"average_risk_score"`
	CareGapCount     int       `json:"care_gap_count"`
}

// GetHistoricalSnapshot retrieves a historical snapshot for a given date.
// Falls back to current data if no historical record exists.
func (r *ProjectionRepository) GetHistoricalSnapshot(ctx context.Context, date time.Time) (*HistoricalSnapshot, error) {
	// First try to get from historical snapshots table
	query := `
		SELECT snapshot_date, total_patients, high_risk_count,
			   rising_risk_count, average_risk_score, care_gap_count
		FROM population_snapshots
		WHERE DATE(snapshot_date) = DATE($1)
		ORDER BY snapshot_date DESC
		LIMIT 1`

	snapshot := &HistoricalSnapshot{}
	err := r.db.QueryRowContext(ctx, query, date).Scan(
		&snapshot.SnapshotDate, &snapshot.TotalPatients, &snapshot.HighRiskCount,
		&snapshot.RisingRiskCount, &snapshot.AverageRiskScore, &snapshot.CareGapCount,
	)

	if err == sql.ErrNoRows {
		// No historical data, compute current stats
		return r.computeCurrentSnapshot(ctx)
	}
	if err != nil {
		// Table might not exist, fall back to current computation
		return r.computeCurrentSnapshot(ctx)
	}

	return snapshot, nil
}

// computeCurrentSnapshot calculates current population metrics.
func (r *ProjectionRepository) computeCurrentSnapshot(ctx context.Context) (*HistoricalSnapshot, error) {
	query := `
		SELECT
			COUNT(*) as total,
			SUM(CASE WHEN current_risk_tier IN ('HIGH', 'VERY_HIGH') THEN 1 ELSE 0 END) as high_risk,
			SUM(CASE WHEN current_risk_tier = 'RISING' THEN 1 ELSE 0 END) as rising,
			COALESCE(AVG(latest_risk_score), 0) as avg_score,
			SUM(care_gap_count) as total_gaps
		FROM patient_projections`

	snapshot := &HistoricalSnapshot{
		SnapshotDate: time.Now(),
	}

	err := r.db.QueryRowContext(ctx, query).Scan(
		&snapshot.TotalPatients, &snapshot.HighRiskCount,
		&snapshot.RisingRiskCount, &snapshot.AverageRiskScore, &snapshot.CareGapCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to compute current snapshot: %w", err)
	}

	return snapshot, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Utilization Data (for Utilization Reporting)
// ──────────────────────────────────────────────────────────────────────────────

// UtilizationData contains utilization metrics for reporting.
type UtilizationData struct {
	TotalPatients       int
	TotalEncounters     int
	InpatientAdmissions int
	EDVisits            int
	OutpatientVisits    int
	Readmissions30Day   int
	AvgLengthOfStay     float64
	ByRiskTier          map[string]*UtilizationMetrics
	ByProvider          map[string]*UtilizationMetrics
	ByPractice          map[string]*UtilizationMetrics
	MonthlyTrend        []MonthlyUtilization
}

// UtilizationMetrics contains detailed utilization metrics.
type UtilizationMetrics struct {
	PatientCount       int
	EncounterCount     int
	InpatientRate      float64
	EDRate             float64
	ReadmissionRate    float64
	AvgEncountersPerPt float64
}

// MonthlyUtilization represents utilization for a specific month.
type MonthlyUtilization struct {
	Month               string `json:"month"`
	InpatientAdmissions int    `json:"inpatient_admissions"`
	EDVisits            int    `json:"ed_visits"`
	OutpatientVisits    int    `json:"outpatient_visits"`
	Readmissions        int    `json:"readmissions"`
}

// GetUtilizationData retrieves utilization data for a date range.
func (r *ProjectionRepository) GetUtilizationData(ctx context.Context, startDate, endDate time.Time) (*UtilizationData, error) {
	data := &UtilizationData{
		ByRiskTier: make(map[string]*UtilizationMetrics),
		ByProvider: make(map[string]*UtilizationMetrics),
		ByPractice: make(map[string]*UtilizationMetrics),
	}

	// Get basic patient count
	countQuery := `SELECT COUNT(*) FROM patient_projections`
	if err := r.db.QueryRowContext(ctx, countQuery).Scan(&data.TotalPatients); err != nil {
		return nil, fmt.Errorf("failed to get patient count: %w", err)
	}

	// Get utilization by risk tier
	tierQuery := `
		SELECT current_risk_tier, COUNT(*)
		FROM patient_projections
		GROUP BY current_risk_tier`

	rows, err := r.db.QueryContext(ctx, tierQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to get tier utilization: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var tier string
		var count int
		if err := rows.Scan(&tier, &count); err != nil {
			continue
		}
		data.ByRiskTier[tier] = &UtilizationMetrics{
			PatientCount: count,
		}
	}

	// Calculate rates per tier
	for tier, metrics := range data.ByRiskTier {
		if metrics.PatientCount > 0 {
			// Simulated rates based on tier (would come from encounter data in production)
			switch tier {
			case "VERY_HIGH":
				metrics.InpatientRate = 150.0
				metrics.EDRate = 300.0
				metrics.ReadmissionRate = 25.0
			case "HIGH":
				metrics.InpatientRate = 100.0
				metrics.EDRate = 200.0
				metrics.ReadmissionRate = 15.0
			case "MODERATE":
				metrics.InpatientRate = 50.0
				metrics.EDRate = 100.0
				metrics.ReadmissionRate = 8.0
			default:
				metrics.InpatientRate = 20.0
				metrics.EDRate = 50.0
				metrics.ReadmissionRate = 3.0
			}
		}
	}

	return data, nil
}
