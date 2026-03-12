// Package cohort provides cohort management for KB-11 Population Health.
package cohort

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// Repository provides database operations for cohorts.
type Repository struct {
	db     *sqlx.DB
	logger *logrus.Entry
}

// NewRepository creates a new cohort repository.
func NewRepository(db *sqlx.DB, logger *logrus.Entry) *Repository {
	return &Repository{
		db:     db,
		logger: logger.WithField("component", "cohort-repository"),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort CRUD Operations
// ──────────────────────────────────────────────────────────────────────────────

// Create creates a new cohort.
func (r *Repository) Create(ctx context.Context, cohort *Cohort) error {
	query := `
		INSERT INTO cohorts (
			id, name, description, type, criteria, member_count,
			last_refreshed, snapshot_date, source_cohort_id,
			created_by, created_at, updated_at, is_active
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13
		)`

	_, err := r.db.ExecContext(ctx, query,
		cohort.ID,
		cohort.Name,
		cohort.Description,
		cohort.Type,
		cohort.CriteriaJSON,
		cohort.MemberCount,
		cohort.LastRefreshed,
		cohort.SnapshotDate,
		cohort.SourceCohortID,
		cohort.CreatedBy,
		cohort.CreatedAt,
		cohort.UpdatedAt,
		cohort.IsActive,
	)

	if err != nil {
		r.logger.WithError(err).WithField("cohort_id", cohort.ID).Error("Failed to create cohort")
		return fmt.Errorf("failed to create cohort: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"cohort_id": cohort.ID,
		"name":      cohort.Name,
		"type":      cohort.Type,
	}).Info("Cohort created")

	return nil
}

// GetByID retrieves a cohort by ID.
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Cohort, error) {
	query := `
		SELECT id, name, description, type, criteria, member_count,
			   last_refreshed, snapshot_date, source_cohort_id,
			   created_by, created_at, updated_at, is_active
		FROM cohorts
		WHERE id = $1 AND is_active = true`

	cohort := &Cohort{}
	err := r.db.GetContext(ctx, cohort, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cohort: %w", err)
	}

	// Load criteria from JSON
	if err := cohort.LoadCriteria(); err != nil {
		r.logger.WithError(err).Warn("Failed to load criteria")
	}

	return cohort, nil
}

// GetByName retrieves a cohort by name.
func (r *Repository) GetByName(ctx context.Context, name string) (*Cohort, error) {
	query := `
		SELECT id, name, description, type, criteria, member_count,
			   last_refreshed, snapshot_date, source_cohort_id,
			   created_by, created_at, updated_at, is_active
		FROM cohorts
		WHERE name = $1 AND is_active = true`

	cohort := &Cohort{}
	err := r.db.GetContext(ctx, cohort, query, name)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get cohort by name: %w", err)
	}

	if err := cohort.LoadCriteria(); err != nil {
		r.logger.WithError(err).Warn("Failed to load criteria")
	}

	return cohort, nil
}

// List retrieves all active cohorts with optional filtering.
func (r *Repository) List(ctx context.Context, filter *CohortFilter) ([]*Cohort, error) {
	query := `
		SELECT id, name, description, type, criteria, member_count,
			   last_refreshed, snapshot_date, source_cohort_id,
			   created_by, created_at, updated_at, is_active
		FROM cohorts
		WHERE is_active = true`

	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if filter.Type != "" {
			query += fmt.Sprintf(" AND type = $%d", argIndex)
			args = append(args, filter.Type)
			argIndex++
		}
		if filter.CreatedBy != "" {
			query += fmt.Sprintf(" AND created_by = $%d", argIndex)
			args = append(args, filter.CreatedBy)
			argIndex++
		}
	}

	query += " ORDER BY created_at DESC"

	if filter != nil && filter.Limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, filter.Limit)
		argIndex++
	}

	if filter != nil && filter.Offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, filter.Offset)
	}

	cohorts := []*Cohort{}
	err := r.db.SelectContext(ctx, &cohorts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to list cohorts: %w", err)
	}

	// Load criteria for each cohort
	for _, c := range cohorts {
		if err := c.LoadCriteria(); err != nil {
			r.logger.WithError(err).WithField("cohort_id", c.ID).Warn("Failed to load criteria")
		}
	}

	return cohorts, nil
}

// Update updates an existing cohort.
func (r *Repository) Update(ctx context.Context, cohort *Cohort) error {
	// Serialize criteria before update
	if err := cohort.SaveCriteria(); err != nil {
		return fmt.Errorf("failed to serialize criteria: %w", err)
	}

	query := `
		UPDATE cohorts SET
			name = $1,
			description = $2,
			criteria = $3,
			member_count = $4,
			last_refreshed = $5,
			updated_at = $6
		WHERE id = $7 AND is_active = true`

	cohort.UpdatedAt = time.Now()

	result, err := r.db.ExecContext(ctx, query,
		cohort.Name,
		cohort.Description,
		cohort.CriteriaJSON,
		cohort.MemberCount,
		cohort.LastRefreshed,
		cohort.UpdatedAt,
		cohort.ID,
	)
	if err != nil {
		return fmt.Errorf("failed to update cohort: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("cohort not found: %s", cohort.ID)
	}

	return nil
}

// Delete soft-deletes a cohort.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE cohorts SET is_active = false, updated_at = $1 WHERE id = $2`

	result, err := r.db.ExecContext(ctx, query, time.Now(), id)
	if err != nil {
		return fmt.Errorf("failed to delete cohort: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("cohort not found: %s", id)
	}

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Member Operations
// ──────────────────────────────────────────────────────────────────────────────

// AddMember adds a patient to a cohort.
func (r *Repository) AddMember(ctx context.Context, member *CohortMember) error {
	query := `
		INSERT INTO cohort_members (
			id, cohort_id, patient_id, fhir_patient_id, joined_at, is_active, snapshot_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (cohort_id, patient_id)
		DO UPDATE SET is_active = true, removed_at = NULL, joined_at = EXCLUDED.joined_at`

	_, err := r.db.ExecContext(ctx, query,
		member.ID,
		member.CohortID,
		member.PatientID,
		member.FHIRPatientID,
		member.JoinedAt,
		member.IsActive,
		member.SnapshotData,
	)
	if err != nil {
		return fmt.Errorf("failed to add cohort member: %w", err)
	}

	return nil
}

// RemoveMember removes a patient from a cohort (soft delete).
func (r *Repository) RemoveMember(ctx context.Context, cohortID, patientID uuid.UUID) error {
	now := time.Now()
	query := `
		UPDATE cohort_members
		SET is_active = false, removed_at = $1
		WHERE cohort_id = $2 AND patient_id = $3 AND is_active = true`

	_, err := r.db.ExecContext(ctx, query, now, cohortID, patientID)
	if err != nil {
		return fmt.Errorf("failed to remove cohort member: %w", err)
	}

	return nil
}

// GetMembers retrieves all active members of a cohort.
func (r *Repository) GetMembers(ctx context.Context, cohortID uuid.UUID, limit, offset int) ([]*CohortMember, error) {
	query := `
		SELECT id, cohort_id, patient_id, fhir_patient_id, joined_at, removed_at, is_active, snapshot_data
		FROM cohort_members
		WHERE cohort_id = $1 AND is_active = true
		ORDER BY joined_at DESC`

	args := []interface{}{cohortID}

	if limit > 0 {
		query += " LIMIT $2"
		args = append(args, limit)
		if offset > 0 {
			query += " OFFSET $3"
			args = append(args, offset)
		}
	}

	members := []*CohortMember{}
	err := r.db.SelectContext(ctx, &members, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get cohort members: %w", err)
	}

	return members, nil
}

// GetMemberCount returns the number of active members in a cohort.
func (r *Repository) GetMemberCount(ctx context.Context, cohortID uuid.UUID) (int, error) {
	query := `SELECT COUNT(*) FROM cohort_members WHERE cohort_id = $1 AND is_active = true`

	var count int
	err := r.db.GetContext(ctx, &count, query, cohortID)
	if err != nil {
		return 0, fmt.Errorf("failed to get member count: %w", err)
	}

	return count, nil
}

// IsMember checks if a patient is a member of a cohort.
func (r *Repository) IsMember(ctx context.Context, cohortID, patientID uuid.UUID) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM cohort_members
			WHERE cohort_id = $1 AND patient_id = $2 AND is_active = true
		)`

	var exists bool
	err := r.db.GetContext(ctx, &exists, query, cohortID, patientID)
	if err != nil {
		return false, fmt.Errorf("failed to check membership: %w", err)
	}

	return exists, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Bulk Operations (for Dynamic Cohort Refresh)
// ──────────────────────────────────────────────────────────────────────────────

// BulkAddMembers adds multiple members to a cohort in a transaction.
func (r *Repository) BulkAddMembers(ctx context.Context, members []*CohortMember) error {
	if len(members) == 0 {
		return nil
	}

	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO cohort_members (
			id, cohort_id, patient_id, fhir_patient_id, joined_at, is_active, snapshot_data
		) VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (cohort_id, patient_id)
		DO UPDATE SET is_active = true, removed_at = NULL, joined_at = EXCLUDED.joined_at`)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, m := range members {
		_, err := stmt.ExecContext(ctx,
			m.ID, m.CohortID, m.PatientID, m.FHIRPatientID,
			m.JoinedAt, m.IsActive, m.SnapshotData,
		)
		if err != nil {
			return fmt.Errorf("failed to insert member: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// BulkRemoveMembers removes members not in the provided list.
func (r *Repository) BulkRemoveMembers(ctx context.Context, cohortID uuid.UUID, keepPatientIDs []uuid.UUID) (int, error) {
	if len(keepPatientIDs) == 0 {
		// Remove all members
		query := `
			UPDATE cohort_members
			SET is_active = false, removed_at = $1
			WHERE cohort_id = $2 AND is_active = true`
		result, err := r.db.ExecContext(ctx, query, time.Now(), cohortID)
		if err != nil {
			return 0, fmt.Errorf("failed to remove all members: %w", err)
		}
		rows, _ := result.RowsAffected()
		return int(rows), nil
	}

	query := `
		UPDATE cohort_members
		SET is_active = false, removed_at = $1
		WHERE cohort_id = $2 AND is_active = true AND patient_id != ALL($3)`

	result, err := r.db.ExecContext(ctx, query, time.Now(), cohortID, keepPatientIDs)
	if err != nil {
		return 0, fmt.Errorf("failed to remove members: %w", err)
	}

	rows, _ := result.RowsAffected()
	return int(rows), nil
}

// UpdateMemberCount updates the cached member count for a cohort.
func (r *Repository) UpdateMemberCount(ctx context.Context, cohortID uuid.UUID) error {
	query := `
		UPDATE cohorts
		SET member_count = (
			SELECT COUNT(*) FROM cohort_members
			WHERE cohort_id = $1 AND is_active = true
		),
		updated_at = $2
		WHERE id = $1`

	_, err := r.db.ExecContext(ctx, query, cohortID, time.Now())
	if err != nil {
		return fmt.Errorf("failed to update member count: %w", err)
	}

	return nil
}

// UpdateLastRefreshed updates the last_refreshed timestamp for a dynamic cohort.
func (r *Repository) UpdateLastRefreshed(ctx context.Context, cohortID uuid.UUID) error {
	now := time.Now()
	query := `UPDATE cohorts SET last_refreshed = $1, updated_at = $1 WHERE id = $2`

	_, err := r.db.ExecContext(ctx, query, now, cohortID)
	if err != nil {
		return fmt.Errorf("failed to update last_refreshed: %w", err)
	}

	return nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort Statistics
// ──────────────────────────────────────────────────────────────────────────────

// GetCohortStats retrieves statistics for a cohort by joining with patient projections.
func (r *Repository) GetCohortStats(ctx context.Context, cohortID uuid.UUID) (*CohortStats, error) {
	cohort, err := r.GetByID(ctx, cohortID)
	if err != nil {
		return nil, err
	}
	if cohort == nil {
		return nil, fmt.Errorf("cohort not found: %s", cohortID)
	}

	stats := &CohortStats{
		CohortID:         cohortID,
		CohortName:       cohort.Name,
		RiskDistribution: make(map[models.RiskTier]int),
		ByPractice:       make(map[string]int),
		ByPCP:            make(map[string]int),
		CalculatedAt:     time.Now(),
	}

	// Get member count
	stats.MemberCount, err = r.GetMemberCount(ctx, cohortID)
	if err != nil {
		return nil, err
	}

	// Get risk distribution by joining with patient_projections
	riskQuery := `
		SELECT pp.current_risk_tier, COUNT(*) as count
		FROM cohort_members cm
		JOIN patient_projections pp ON cm.fhir_patient_id = pp.fhir_patient_id
		WHERE cm.cohort_id = $1 AND cm.is_active = true AND pp.is_active = true
		GROUP BY pp.current_risk_tier`

	rows, err := r.db.QueryContext(ctx, riskQuery, cohortID)
	if err != nil {
		r.logger.WithError(err).Warn("Failed to get risk distribution")
	} else {
		defer rows.Close()
		var totalScore float64
		var totalCount int
		for rows.Next() {
			var tier models.RiskTier
			var count int
			if err := rows.Scan(&tier, &count); err == nil {
				stats.RiskDistribution[tier] = count
				if tier == models.RiskTierHigh || tier == models.RiskTierVeryHigh {
					stats.HighRiskCount += count
				}
				totalCount += count
			}
		}
		if totalCount > 0 {
			// Calculate average from projected scores
			avgQuery := `
				SELECT AVG(pp.current_risk_score)
				FROM cohort_members cm
				JOIN patient_projections pp ON cm.fhir_patient_id = pp.fhir_patient_id
				WHERE cm.cohort_id = $1 AND cm.is_active = true AND pp.is_active = true`
			r.db.GetContext(ctx, &totalScore, avgQuery, cohortID)
			stats.AverageRiskScore = totalScore
		}
	}

	// Get distribution by practice
	practiceQuery := `
		SELECT pp.attributed_practice, COUNT(*) as count
		FROM cohort_members cm
		JOIN patient_projections pp ON cm.fhir_patient_id = pp.fhir_patient_id
		WHERE cm.cohort_id = $1 AND cm.is_active = true AND pp.is_active = true
			AND pp.attributed_practice IS NOT NULL AND pp.attributed_practice != ''
		GROUP BY pp.attributed_practice`

	practiceRows, err := r.db.QueryContext(ctx, practiceQuery, cohortID)
	if err == nil {
		defer practiceRows.Close()
		for practiceRows.Next() {
			var practice string
			var count int
			if err := practiceRows.Scan(&practice, &count); err == nil {
				stats.ByPractice[practice] = count
			}
		}
	}

	// Get distribution by PCP
	pcpQuery := `
		SELECT pp.attributed_pcp, COUNT(*) as count
		FROM cohort_members cm
		JOIN patient_projections pp ON cm.fhir_patient_id = pp.fhir_patient_id
		WHERE cm.cohort_id = $1 AND cm.is_active = true AND pp.is_active = true
			AND pp.attributed_pcp IS NOT NULL AND pp.attributed_pcp != ''
		GROUP BY pp.attributed_pcp`

	pcpRows, err := r.db.QueryContext(ctx, pcpQuery, cohortID)
	if err == nil {
		defer pcpRows.Close()
		for pcpRows.Next() {
			var pcp string
			var count int
			if err := pcpRows.Scan(&pcp, &count); err == nil {
				stats.ByPCP[pcp] = count
			}
		}
	}

	return stats, nil
}

// ──────────────────────────────────────────────────────────────────────────────
// Dynamic Cohort Query Building
// ──────────────────────────────────────────────────────────────────────────────

// FindPatientsMatchingCriteria finds all patients matching cohort criteria.
// This is used for refreshing dynamic cohorts.
func (r *Repository) FindPatientsMatchingCriteria(ctx context.Context, criteria []Criterion) ([]PatientMatch, error) {
	if len(criteria) == 0 {
		return nil, fmt.Errorf("no criteria provided")
	}

	// Build the WHERE clause from criteria
	whereClause, args := buildCriteriaQuery(criteria)

	query := fmt.Sprintf(`
		SELECT id, fhir_patient_id
		FROM patient_projections
		WHERE is_active = true AND %s`, whereClause)

	var matches []PatientMatch
	err := r.db.SelectContext(ctx, &matches, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to find matching patients: %w", err)
	}

	return matches, nil
}

// PatientMatch represents a patient matching cohort criteria.
type PatientMatch struct {
	ID            uuid.UUID `db:"id"`
	FHIRPatientID string    `db:"fhir_patient_id"`
}

// buildCriteriaQuery builds a SQL WHERE clause from criteria.
func buildCriteriaQuery(criteria []Criterion) (string, []interface{}) {
	if len(criteria) == 0 {
		return "1=1", nil
	}

	var conditions []string
	var args []interface{}
	argIndex := 1

	for i, c := range criteria {
		condition, newArgs, newIndex := buildCriterionCondition(c, argIndex)
		conditions = append(conditions, condition)
		args = append(args, newArgs...)
		argIndex = newIndex

		// Add logic operator (except for last criterion)
		if i < len(criteria)-1 && c.Logic != "" {
			// The logic operator is applied between this criterion and the next
		}
	}

	// Combine conditions with appropriate logic
	result := ""
	for i, cond := range conditions {
		if i > 0 {
			logic := "AND"
			if i > 0 && criteria[i-1].Logic == "OR" {
				logic = "OR"
			}
			result += " " + logic + " "
		}
		result += "(" + cond + ")"
	}

	return result, args
}

// buildCriterionCondition builds a single criterion condition.
func buildCriterionCondition(c Criterion, argIndex int) (string, []interface{}, int) {
	var condition string
	var args []interface{}

	// Map criterion fields to database columns
	columnMap := map[string]string{
		"current_risk_tier":    "current_risk_tier",
		"risk_tier":            "current_risk_tier",
		"current_risk_score":   "current_risk_score",
		"risk_score":           "current_risk_score",
		"age":                  "age",
		"gender":               "gender",
		"attributed_pcp":       "attributed_pcp",
		"attributed_practice":  "attributed_practice",
		"care_gap_count":       "care_gap_count",
		"last_encounter_date":  "last_encounter_date",
	}

	column, ok := columnMap[c.Field]
	if !ok {
		column = c.Field // Use as-is if not in map
	}

	switch c.Operator {
	case models.OpEquals:
		condition = fmt.Sprintf("%s = $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpNotEquals:
		condition = fmt.Sprintf("%s != $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpGreaterThan:
		condition = fmt.Sprintf("%s > $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpGreaterEq:
		condition = fmt.Sprintf("%s >= $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpLessThan:
		condition = fmt.Sprintf("%s < $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpLessEq:
		condition = fmt.Sprintf("%s <= $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpIn:
		condition = fmt.Sprintf("%s = ANY($%d)", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpNotIn:
		condition = fmt.Sprintf("%s != ALL($%d)", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	case models.OpContains:
		condition = fmt.Sprintf("%s ILIKE $%d", column, argIndex)
		args = append(args, "%"+fmt.Sprint(c.Value)+"%")
		argIndex++

	case models.OpMatches:
		condition = fmt.Sprintf("%s ~ $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++

	default:
		condition = fmt.Sprintf("%s = $%d", column, argIndex)
		args = append(args, c.Value)
		argIndex++
	}

	return condition, args, argIndex
}

// ──────────────────────────────────────────────────────────────────────────────
// Filter Types
// ──────────────────────────────────────────────────────────────────────────────

// CohortFilter provides filtering options for listing cohorts.
type CohortFilter struct {
	Type      models.CohortType
	CreatedBy string
	Limit     int
	Offset    int
}
