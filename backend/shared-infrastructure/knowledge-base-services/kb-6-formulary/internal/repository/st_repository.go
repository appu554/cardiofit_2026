// Package repository provides database access layer for KB-6 Formulary Service.
package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/lib/pq"
	"kb-formulary/internal/models"
)

// STRepository handles Step Therapy database operations
type STRepository struct {
	db *sql.DB
}

// NewSTRepository creates a new STRepository instance
func NewSTRepository(db *sql.DB) *STRepository {
	return &STRepository{db: db}
}

// GetRules retrieves ST rules for a drug
func (r *STRepository) GetRules(ctx context.Context, rxnormCode string, payerID, planID *string) (*models.StepTherapyRule, error) {
	query := `
		SELECT id, target_drug_rxnorm, target_drug_name, payer_id, plan_id, steps,
		       override_criteria, exception_diagnosis_codes,
		       protocol_name, protocol_version, evidence_level,
		       effective_date, termination_date, version, created_at, updated_at
		FROM step_therapy_rules
		WHERE target_drug_rxnorm = $1
		  AND (payer_id = $2 OR payer_id IS NULL)
		  AND (plan_id = $3 OR plan_id IS NULL)
		  AND (termination_date IS NULL OR termination_date > CURRENT_DATE)
		  AND effective_date <= CURRENT_DATE
		ORDER BY payer_id NULLS LAST, plan_id NULLS LAST
		LIMIT 1
	`

	var rule models.StepTherapyRule
	var stepsJSON []byte
	var overrideCriteria, exceptionCodes []string

	err := r.db.QueryRowContext(ctx, query, rxnormCode, payerID, planID).Scan(
		&rule.ID, &rule.TargetDrugRxNorm, &rule.TargetDrugName, &rule.PayerID, &rule.PlanID, &stepsJSON,
		pq.Array(&overrideCriteria), pq.Array(&exceptionCodes),
		&rule.ProtocolName, &rule.ProtocolVersion, &rule.EvidenceLevel,
		&rule.EffectiveDate, &rule.TerminationDate, &rule.Version, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No ST rule found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ST rules: %w", err)
	}

	// Unmarshal JSONB steps
	if err := json.Unmarshal(stepsJSON, &rule.Steps); err != nil {
		return nil, fmt.Errorf("failed to unmarshal steps: %w", err)
	}

	rule.OverrideCriteria = overrideCriteria
	rule.ExceptionDiagnosisCodes = exceptionCodes

	return &rule, nil
}

// CheckSTRequired checks if ST is required for a drug
func (r *STRepository) CheckSTRequired(ctx context.Context, rxnormCode string, payerID, planID *string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM step_therapy_rules
			WHERE target_drug_rxnorm = $1
			  AND (payer_id = $2 OR payer_id IS NULL)
			  AND (plan_id = $3 OR plan_id IS NULL)
			  AND (termination_date IS NULL OR termination_date > CURRENT_DATE)
			  AND effective_date <= CURRENT_DATE
		)
	`

	var required bool
	err := r.db.QueryRowContext(ctx, query, rxnormCode, payerID, planID).Scan(&required)
	if err != nil {
		return false, fmt.Errorf("failed to check ST required: %w", err)
	}

	return required, nil
}

// SaveCheck saves a step therapy check result
func (r *STRepository) SaveCheck(ctx context.Context, check *models.StepTherapyCheck) error {
	query := `
		INSERT INTO step_therapy_checks (
			id, patient_id, provider_id, target_drug_rxnorm, target_drug_name,
			payer_id, plan_id, drug_history,
			step_therapy_required, total_steps, steps_satisfied, current_step,
			approved, override_requested, override_reason, override_approved,
			message, next_required_step, checked_at, rule_id
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, $18, $19, $20
		)
	`

	check.ID = uuid.New()
	check.CheckedAt = time.Now()

	drugHistoryJSON, err := json.Marshal(check.DrugHistory)
	if err != nil {
		return fmt.Errorf("failed to marshal drug history: %w", err)
	}

	var nextStepJSON []byte
	if check.NextRequiredStep != nil {
		nextStepJSON, err = json.Marshal(check.NextRequiredStep)
		if err != nil {
			return fmt.Errorf("failed to marshal next step: %w", err)
		}
	}

	_, err = r.db.ExecContext(ctx, query,
		check.ID, check.PatientID, check.ProviderID, check.TargetDrugRxNorm, check.TargetDrugName,
		check.PayerID, check.PlanID, drugHistoryJSON,
		check.StepTherapyRequired, check.TotalSteps, pq.Array(check.StepsSatisfied), check.CurrentStep,
		check.Approved, check.OverrideRequested, check.OverrideReason, check.OverrideApproved,
		check.Message, nextStepJSON, check.CheckedAt, check.RuleID,
	)

	if err != nil {
		return fmt.Errorf("failed to save ST check: %w", err)
	}

	return nil
}

// GetCheck retrieves a step therapy check by ID
func (r *STRepository) GetCheck(ctx context.Context, checkID uuid.UUID) (*models.StepTherapyCheck, error) {
	query := `
		SELECT id, patient_id, provider_id, target_drug_rxnorm, target_drug_name,
		       payer_id, plan_id, drug_history,
		       step_therapy_required, total_steps, steps_satisfied, current_step,
		       approved, override_requested, override_reason, override_approved,
		       message, next_required_step, checked_at, rule_id
		FROM step_therapy_checks
		WHERE id = $1
	`

	var check models.StepTherapyCheck
	var drugHistoryJSON, nextStepJSON []byte
	var stepsSatisfied []int64

	err := r.db.QueryRowContext(ctx, query, checkID).Scan(
		&check.ID, &check.PatientID, &check.ProviderID, &check.TargetDrugRxNorm, &check.TargetDrugName,
		&check.PayerID, &check.PlanID, &drugHistoryJSON,
		&check.StepTherapyRequired, &check.TotalSteps, pq.Array(&stepsSatisfied), &check.CurrentStep,
		&check.Approved, &check.OverrideRequested, &check.OverrideReason, &check.OverrideApproved,
		&check.Message, &nextStepJSON, &check.CheckedAt, &check.RuleID,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ST check: %w", err)
	}

	if err := json.Unmarshal(drugHistoryJSON, &check.DrugHistory); err != nil {
		return nil, fmt.Errorf("failed to unmarshal drug history: %w", err)
	}

	if len(nextStepJSON) > 0 {
		var step models.Step
		if err := json.Unmarshal(nextStepJSON, &step); err != nil {
			return nil, fmt.Errorf("failed to unmarshal next step: %w", err)
		}
		check.NextRequiredStep = &step
	}

	// Convert []int64 to []int
	check.StepsSatisfied = make([]int, len(stepsSatisfied))
	for i, v := range stepsSatisfied {
		check.StepsSatisfied[i] = int(v)
	}

	return &check, nil
}

// CreateOverride creates a new step therapy override request
func (r *STRepository) CreateOverride(ctx context.Context, override *models.StepTherapyOverride) error {
	query := `
		INSERT INTO step_therapy_overrides (
			id, check_id, patient_id, provider_id, target_drug_rxnorm,
			override_reason, clinical_justification, supporting_documentation,
			status, submitted_at, submitted_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11
		)
	`

	override.ID = uuid.New()
	override.Status = models.STOverridePending
	override.SubmittedAt = time.Now()

	supportingDocJSON, err := json.Marshal(override.SupportingDocumentation)
	if err != nil {
		return fmt.Errorf("failed to marshal supporting documentation: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		override.ID, override.CheckID, override.PatientID, override.ProviderID, override.TargetDrugRxNorm,
		override.OverrideReason, override.ClinicalJustification, supportingDocJSON,
		override.Status, override.SubmittedAt, override.SubmittedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create ST override: %w", err)
	}

	return nil
}

// GetOverride retrieves an override request by ID
func (r *STRepository) GetOverride(ctx context.Context, overrideID uuid.UUID) (*models.StepTherapyOverride, error) {
	query := `
		SELECT id, check_id, patient_id, provider_id, target_drug_rxnorm,
		       override_reason, clinical_justification, supporting_documentation,
		       status, decision_reason, submitted_at, reviewed_at, decision_at, expires_at,
		       submitted_by, reviewed_by
		FROM step_therapy_overrides
		WHERE id = $1
	`

	var override models.StepTherapyOverride
	var supportingDocJSON []byte

	err := r.db.QueryRowContext(ctx, query, overrideID).Scan(
		&override.ID, &override.CheckID, &override.PatientID, &override.ProviderID, &override.TargetDrugRxNorm,
		&override.OverrideReason, &override.ClinicalJustification, &supportingDocJSON,
		&override.Status, &override.DecisionReason, &override.SubmittedAt, &override.ReviewedAt, &override.DecisionAt, &override.ExpiresAt,
		&override.SubmittedBy, &override.ReviewedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get ST override: %w", err)
	}

	if len(supportingDocJSON) > 0 {
		if err := json.Unmarshal(supportingDocJSON, &override.SupportingDocumentation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal supporting documentation: %w", err)
		}
	}

	return &override, nil
}

// GetActiveOverride retrieves an active override for a patient/drug combination
func (r *STRepository) GetActiveOverride(ctx context.Context, patientID, rxnormCode string) (*models.StepTherapyOverride, error) {
	query := `
		SELECT id, check_id, patient_id, provider_id, target_drug_rxnorm,
		       override_reason, clinical_justification, supporting_documentation,
		       status, decision_reason, submitted_at, reviewed_at, decision_at, expires_at,
		       submitted_by, reviewed_by
		FROM step_therapy_overrides
		WHERE patient_id = $1
		  AND target_drug_rxnorm = $2
		  AND status = 'APPROVED'
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY decision_at DESC
		LIMIT 1
	`

	var override models.StepTherapyOverride
	var supportingDocJSON []byte

	err := r.db.QueryRowContext(ctx, query, patientID, rxnormCode).Scan(
		&override.ID, &override.CheckID, &override.PatientID, &override.ProviderID, &override.TargetDrugRxNorm,
		&override.OverrideReason, &override.ClinicalJustification, &supportingDocJSON,
		&override.Status, &override.DecisionReason, &override.SubmittedAt, &override.ReviewedAt, &override.DecisionAt, &override.ExpiresAt,
		&override.SubmittedBy, &override.ReviewedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active override: %w", err)
	}

	if len(supportingDocJSON) > 0 {
		if err := json.Unmarshal(supportingDocJSON, &override.SupportingDocumentation); err != nil {
			return nil, fmt.Errorf("failed to unmarshal supporting documentation: %w", err)
		}
	}

	return &override, nil
}

// UpdateOverrideStatus updates the status of an override request
func (r *STRepository) UpdateOverrideStatus(ctx context.Context, overrideID uuid.UUID, status models.STOverrideStatus, reason string, reviewedBy string) error {
	query := `
		UPDATE step_therapy_overrides
		SET status = $2,
		    decision_reason = $3,
		    reviewed_by = $4,
		    reviewed_at = NOW(),
		    decision_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, overrideID, status, reason, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to update override status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("override not found: %s", overrideID)
	}

	return nil
}

// ListPendingOverrides lists all pending override requests
func (r *STRepository) ListPendingOverrides(ctx context.Context, limit, offset int) ([]models.StepTherapyOverride, error) {
	query := `
		SELECT id, check_id, patient_id, provider_id, target_drug_rxnorm,
		       override_reason, clinical_justification, status, submitted_at
		FROM step_therapy_overrides
		WHERE status = 'PENDING'
		ORDER BY submitted_at ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending overrides: %w", err)
	}
	defer rows.Close()

	var overrides []models.StepTherapyOverride
	for rows.Next() {
		var override models.StepTherapyOverride
		if err := rows.Scan(
			&override.ID, &override.CheckID, &override.PatientID, &override.ProviderID, &override.TargetDrugRxNorm,
			&override.OverrideReason, &override.ClinicalJustification, &override.Status, &override.SubmittedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan override: %w", err)
		}
		overrides = append(overrides, override)
	}

	return overrides, nil
}
