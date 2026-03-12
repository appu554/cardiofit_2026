// Package repository provides database access layer for KB-6 Formulary Service.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"kb-formulary/internal/models"
)

// QLRepository handles Quantity Limit database operations
type QLRepository struct {
	db *sql.DB
}

// NewQLRepository creates a new QLRepository instance
func NewQLRepository(db *sql.DB) *QLRepository {
	return &QLRepository{db: db}
}

// GetFormularyLimits retrieves quantity limits for a drug from formulary entries
func (r *QLRepository) GetFormularyLimits(ctx context.Context, rxnormCode string, payerID, planID *string) (*models.ExtendedQuantityLimit, string, error) {
	query := `
		SELECT drug_name, quantity_limit
		FROM formulary_entries
		WHERE drug_rxnorm = $1
		  AND (payer_id = $2 OR payer_id IS NULL)
		  AND (plan_id = $3 OR plan_id IS NULL)
		  AND status = 'active'
		ORDER BY payer_id NULLS LAST, plan_id NULLS LAST
		LIMIT 1
	`

	var drugName string
	var limitJSON []byte

	err := r.db.QueryRowContext(ctx, query, rxnormCode, payerID, planID).Scan(&drugName, &limitJSON)
	if err == sql.ErrNoRows {
		return nil, "", nil // No entry found
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to get formulary limits: %w", err)
	}

	if len(limitJSON) == 0 {
		return nil, drugName, nil // Entry found but no limits
	}

	var limits models.ExtendedQuantityLimit
	if err := json.Unmarshal(limitJSON, &limits); err != nil {
		return nil, drugName, fmt.Errorf("failed to unmarshal quantity limits: %w", err)
	}

	return &limits, drugName, nil
}

// GetActiveOverride retrieves an active quantity limit override for a patient/drug
func (r *QLRepository) GetActiveOverride(ctx context.Context, patientID, rxnormCode string, payerID *string) (*models.QLOverride, error) {
	query := `
		SELECT approved_quantity, approved_days_supply, approved_fills_year,
		       override_reason, approved_by, approved_at, expires_at
		FROM quantity_limit_overrides
		WHERE patient_id = $1
		  AND drug_rxnorm = $2
		  AND (payer_id = $3 OR $3 IS NULL)
		  AND status = 'APPROVED'
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY approved_at DESC
		LIMIT 1
	`

	var override models.QLOverride
	err := r.db.QueryRowContext(ctx, query, patientID, rxnormCode, payerID).Scan(
		&override.ApprovedQuantity, &override.ApprovedDaysSupply, &override.ApprovedFillsYear,
		&override.OverrideReason, &override.ApprovedBy, &override.ApprovedAt, &override.ExpiresAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active override: %w", err)
	}

	return &override, nil
}

// CreateOverride creates a new quantity limit override
func (r *QLRepository) CreateOverride(ctx context.Context, req *models.QLOverrideRequest, approved bool) (*models.QLOverride, error) {
	// First check if the table exists (it may need to be created)
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS quantity_limit_overrides (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			patient_id VARCHAR(100) NOT NULL,
			drug_rxnorm VARCHAR(20) NOT NULL,
			payer_id VARCHAR(50),
			plan_id VARCHAR(100),
			approved_quantity INTEGER,
			approved_days_supply INTEGER,
			approved_fills_year INTEGER,
			override_reason TEXT NOT NULL,
			clinical_notes TEXT,
			status VARCHAR(20) NOT NULL DEFAULT 'PENDING',
			submitted_by VARCHAR(100),
			approved_by VARCHAR(100),
			submitted_at TIMESTAMPTZ DEFAULT NOW(),
			approved_at TIMESTAMPTZ,
			expires_at TIMESTAMPTZ,
			CONSTRAINT unique_ql_override UNIQUE (patient_id, drug_rxnorm, payer_id)
		)
	`
	_, _ = r.db.ExecContext(ctx, createTableQuery) // Ignore error if table exists

	insertQuery := `
		INSERT INTO quantity_limit_overrides (
			id, patient_id, drug_rxnorm, payer_id, plan_id,
			approved_quantity, approved_days_supply,
			override_reason, clinical_notes,
			status, submitted_by, approved_by, approved_at, expires_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14
		)
		ON CONFLICT (patient_id, drug_rxnorm, payer_id)
		DO UPDATE SET
			approved_quantity = EXCLUDED.approved_quantity,
			approved_days_supply = EXCLUDED.approved_days_supply,
			override_reason = EXCLUDED.override_reason,
			clinical_notes = EXCLUDED.clinical_notes,
			status = EXCLUDED.status,
			approved_by = EXCLUDED.approved_by,
			approved_at = EXCLUDED.approved_at,
			expires_at = EXCLUDED.expires_at
	`

	id := uuid.New()
	now := time.Now()
	var expiresAt *time.Time
	status := "PENDING"
	var approvedBy *string
	var approvedAt *time.Time

	if approved {
		status = "APPROVED"
		approvedBy = &req.ProviderID
		approvedAt = &now
		exp := now.AddDate(1, 0, 0) // 1 year
		expiresAt = &exp
	}

	_, err := r.db.ExecContext(ctx, insertQuery,
		id, req.PatientID, req.DrugRxNorm, req.PayerID, req.PlanID,
		req.RequestedQuantity, req.RequestedDaysSupply,
		req.OverrideReason, req.ClinicalNotes,
		status, req.ProviderID, approvedBy, approvedAt, expiresAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to create override: %w", err)
	}

	if approved {
		return &models.QLOverride{
			ApprovedQuantity:   req.RequestedQuantity,
			ApprovedDaysSupply: req.RequestedDaysSupply,
			OverrideReason:     req.OverrideReason,
			ApprovedBy:         req.ProviderID,
			ApprovedAt:         now,
			ExpiresAt:          expiresAt,
		}, nil
	}

	return nil, nil
}

// GetPatientFillCount retrieves the number of fills this year for a patient/drug
func (r *QLRepository) GetPatientFillCount(ctx context.Context, patientID, rxnormCode string, payerID *string) (int, error) {
	query := `
		SELECT COUNT(*)
		FROM prescription_fills
		WHERE patient_id = $1
		  AND drug_rxnorm = $2
		  AND (payer_id = $3 OR $3 IS NULL)
		  AND fill_date >= date_trunc('year', CURRENT_DATE)
	`

	var count int
	err := r.db.QueryRowContext(ctx, query, patientID, rxnormCode, payerID).Scan(&count)
	if err != nil {
		// Table might not exist, return 0
		return 0, nil
	}

	return count, nil
}

// SaveCheckLog saves a quantity limit check for audit purposes
func (r *QLRepository) SaveCheckLog(ctx context.Context, req *models.QLCheckRequest, response *models.QLCheckResponse) error {
	// Create table if it doesn't exist
	createTableQuery := `
		CREATE TABLE IF NOT EXISTS quantity_limit_checks (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			patient_id VARCHAR(100),
			drug_rxnorm VARCHAR(20) NOT NULL,
			payer_id VARCHAR(50),
			plan_id VARCHAR(100),
			requested_quantity INTEGER NOT NULL,
			requested_days_supply INTEGER NOT NULL,
			within_limits BOOLEAN NOT NULL,
			violations JSONB,
			checked_at TIMESTAMPTZ DEFAULT NOW()
		)
	`
	_, _ = r.db.ExecContext(ctx, createTableQuery)

	insertQuery := `
		INSERT INTO quantity_limit_checks (
			patient_id, drug_rxnorm, payer_id, plan_id,
			requested_quantity, requested_days_supply,
			within_limits, violations
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	violationsJSON, _ := json.Marshal(response.Violations)

	_, err := r.db.ExecContext(ctx, insertQuery,
		req.PatientID, req.DrugRxNorm, req.PayerID, req.PlanID,
		req.Quantity, req.DaysSupply,
		response.WithinLimits, violationsJSON,
	)

	return err // Log errors but don't fail the check
}
