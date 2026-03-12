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

// PARepository handles Prior Authorization database operations
type PARepository struct {
	db *sql.DB
}

// NewPARepository creates a new PARepository instance
func NewPARepository(db *sql.DB) *PARepository {
	return &PARepository{db: db}
}

// GetRequirements retrieves PA requirements for a drug
func (r *PARepository) GetRequirements(ctx context.Context, rxnormCode string, payerID, planID *string) (*models.PARequirement, error) {
	query := `
		SELECT id, drug_rxnorm, drug_name, payer_id, plan_id, criteria,
		       approval_duration_days, renewal_allowed, max_renewals,
		       required_documents, urgency_levels,
		       standard_review_hours, urgent_review_hours, expedited_review_hours,
		       effective_date, termination_date, version, created_at, updated_at
		FROM pa_requirements
		WHERE drug_rxnorm = $1
		  AND (payer_id = $2 OR payer_id IS NULL)
		  AND (plan_id = $3 OR plan_id IS NULL)
		  AND (termination_date IS NULL OR termination_date > CURRENT_DATE)
		  AND effective_date <= CURRENT_DATE
		ORDER BY payer_id NULLS LAST, plan_id NULLS LAST
		LIMIT 1
	`

	var req models.PARequirement
	var criteriaJSON []byte
	var requiredDocsArray, urgencyLevelsArray []string

	err := r.db.QueryRowContext(ctx, query, rxnormCode, payerID, planID).Scan(
		&req.ID, &req.DrugRxNorm, &req.DrugName, &req.PayerID, &req.PlanID, &criteriaJSON,
		&req.ApprovalDurationDays, &req.RenewalAllowed, &req.MaxRenewals,
		pq.Array(&requiredDocsArray), pq.Array(&urgencyLevelsArray),
		&req.StandardReviewHours, &req.UrgentReviewHours, &req.ExpeditedReviewHours,
		&req.EffectiveDate, &req.TerminationDate, &req.Version, &req.CreatedAt, &req.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No PA requirement found
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get PA requirements: %w", err)
	}

	// Unmarshal JSONB criteria
	if err := json.Unmarshal(criteriaJSON, &req.Criteria); err != nil {
		return nil, fmt.Errorf("failed to unmarshal criteria: %w", err)
	}

	req.RequiredDocuments = requiredDocsArray
	req.UrgencyLevels = urgencyLevelsArray

	return &req, nil
}

// CheckPARequired checks if PA is required for a drug
func (r *PARepository) CheckPARequired(ctx context.Context, rxnormCode string, payerID, planID *string) (bool, error) {
	query := `
		SELECT EXISTS(
			SELECT 1 FROM pa_requirements
			WHERE drug_rxnorm = $1
			  AND (payer_id = $2 OR payer_id IS NULL)
			  AND (plan_id = $3 OR plan_id IS NULL)
			  AND (termination_date IS NULL OR termination_date > CURRENT_DATE)
			  AND effective_date <= CURRENT_DATE
		)
	`

	var required bool
	err := r.db.QueryRowContext(ctx, query, rxnormCode, payerID, planID).Scan(&required)
	if err != nil {
		return false, fmt.Errorf("failed to check PA required: %w", err)
	}

	return required, nil
}

// CreateSubmission creates a new PA submission
func (r *PARepository) CreateSubmission(ctx context.Context, submission *models.PASubmission) error {
	query := `
		INSERT INTO pa_submissions (
			id, external_id, patient_id, provider_id, provider_npi,
			drug_rxnorm, drug_name, quantity, days_supply,
			clinical_documentation, payer_id, plan_id, member_id,
			status, urgency_level, submitted_at, created_by
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	submission.ID = uuid.New()
	submission.SubmittedAt = time.Now()

	clinicalDocJSON, err := json.Marshal(submission.ClinicalDocumentation)
	if err != nil {
		return fmt.Errorf("failed to marshal clinical documentation: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		submission.ID, submission.ExternalID, submission.PatientID, submission.ProviderID, submission.ProviderNPI,
		submission.DrugRxNorm, submission.DrugName, submission.Quantity, submission.DaysSupply,
		clinicalDocJSON, submission.PayerID, submission.PlanID, submission.MemberID,
		submission.Status, submission.UrgencyLevel, submission.SubmittedAt, submission.CreatedBy,
	)

	if err != nil {
		return fmt.Errorf("failed to create PA submission: %w", err)
	}

	return nil
}

// GetSubmission retrieves a PA submission by ID
func (r *PARepository) GetSubmission(ctx context.Context, submissionID uuid.UUID) (*models.PASubmission, error) {
	query := `
		SELECT id, external_id, patient_id, provider_id, provider_npi,
		       drug_rxnorm, drug_name, quantity, days_supply,
		       clinical_documentation, payer_id, plan_id, member_id,
		       status, urgency_level, decision_reason,
		       approved_quantity, approved_days_supply,
		       submitted_at, reviewed_at, decision_at, expires_at,
		       created_by, reviewed_by
		FROM pa_submissions
		WHERE id = $1
	`

	var sub models.PASubmission
	var clinicalDocJSON []byte

	err := r.db.QueryRowContext(ctx, query, submissionID).Scan(
		&sub.ID, &sub.ExternalID, &sub.PatientID, &sub.ProviderID, &sub.ProviderNPI,
		&sub.DrugRxNorm, &sub.DrugName, &sub.Quantity, &sub.DaysSupply,
		&clinicalDocJSON, &sub.PayerID, &sub.PlanID, &sub.MemberID,
		&sub.Status, &sub.UrgencyLevel, &sub.DecisionReason,
		&sub.ApprovedQuantity, &sub.ApprovedDaysSupply,
		&sub.SubmittedAt, &sub.ReviewedAt, &sub.DecisionAt, &sub.ExpiresAt,
		&sub.CreatedBy, &sub.ReviewedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get PA submission: %w", err)
	}

	if err := json.Unmarshal(clinicalDocJSON, &sub.ClinicalDocumentation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clinical documentation: %w", err)
	}

	return &sub, nil
}

// GetActiveApproval retrieves an active PA approval for a patient/drug combination
func (r *PARepository) GetActiveApproval(ctx context.Context, patientID, rxnormCode string, payerID *string) (*models.PASubmission, error) {
	query := `
		SELECT id, external_id, patient_id, provider_id, provider_npi,
		       drug_rxnorm, drug_name, quantity, days_supply,
		       clinical_documentation, payer_id, plan_id, member_id,
		       status, urgency_level, decision_reason,
		       approved_quantity, approved_days_supply,
		       submitted_at, reviewed_at, decision_at, expires_at,
		       created_by, reviewed_by
		FROM pa_submissions
		WHERE patient_id = $1
		  AND drug_rxnorm = $2
		  AND (payer_id = $3 OR $3 IS NULL)
		  AND status = 'APPROVED'
		  AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY decision_at DESC
		LIMIT 1
	`

	var sub models.PASubmission
	var clinicalDocJSON []byte

	err := r.db.QueryRowContext(ctx, query, patientID, rxnormCode, payerID).Scan(
		&sub.ID, &sub.ExternalID, &sub.PatientID, &sub.ProviderID, &sub.ProviderNPI,
		&sub.DrugRxNorm, &sub.DrugName, &sub.Quantity, &sub.DaysSupply,
		&clinicalDocJSON, &sub.PayerID, &sub.PlanID, &sub.MemberID,
		&sub.Status, &sub.UrgencyLevel, &sub.DecisionReason,
		&sub.ApprovedQuantity, &sub.ApprovedDaysSupply,
		&sub.SubmittedAt, &sub.ReviewedAt, &sub.DecisionAt, &sub.ExpiresAt,
		&sub.CreatedBy, &sub.ReviewedBy,
	)

	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get active approval: %w", err)
	}

	if err := json.Unmarshal(clinicalDocJSON, &sub.ClinicalDocumentation); err != nil {
		return nil, fmt.Errorf("failed to unmarshal clinical documentation: %w", err)
	}

	return &sub, nil
}

// UpdateSubmissionStatus updates the status of a PA submission
func (r *PARepository) UpdateSubmissionStatus(ctx context.Context, submissionID uuid.UUID, status models.PAStatus, reason string, reviewedBy string) error {
	query := `
		UPDATE pa_submissions
		SET status = $2,
		    decision_reason = $3,
		    reviewed_by = $4,
		    reviewed_at = NOW(),
		    decision_at = NOW()
		WHERE id = $1
	`

	result, err := r.db.ExecContext(ctx, query, submissionID, status, reason, reviewedBy)
	if err != nil {
		return fmt.Errorf("failed to update PA submission status: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("PA submission not found: %s", submissionID)
	}

	return nil
}

// SaveCriteriaEvaluation saves the evaluation of PA criteria
func (r *PARepository) SaveCriteriaEvaluation(ctx context.Context, submissionID uuid.UUID, evaluation models.CriterionEvaluation) error {
	query := `
		INSERT INTO pa_criteria_evaluations (
			id, submission_id, criterion_type, criterion_json,
			met, evidence, notes, evaluated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	criterionJSON, err := json.Marshal(evaluation.Criterion)
	if err != nil {
		return fmt.Errorf("failed to marshal criterion: %w", err)
	}

	evidenceJSON, err := json.Marshal(evaluation.Evidence)
	if err != nil {
		return fmt.Errorf("failed to marshal evidence: %w", err)
	}

	_, err = r.db.ExecContext(ctx, query,
		uuid.New(), submissionID, evaluation.Criterion.Type, criterionJSON,
		evaluation.Met, evidenceJSON, evaluation.Notes, evaluation.EvaluatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to save criteria evaluation: %w", err)
	}

	return nil
}

// GetCriteriaEvaluations retrieves all criteria evaluations for a submission
func (r *PARepository) GetCriteriaEvaluations(ctx context.Context, submissionID uuid.UUID) ([]models.CriterionEvaluation, error) {
	query := `
		SELECT criterion_type, criterion_json, met, evidence, notes, evaluated_at
		FROM pa_criteria_evaluations
		WHERE submission_id = $1
		ORDER BY evaluated_at
	`

	rows, err := r.db.QueryContext(ctx, query, submissionID)
	if err != nil {
		return nil, fmt.Errorf("failed to get criteria evaluations: %w", err)
	}
	defer rows.Close()

	var evaluations []models.CriterionEvaluation
	for rows.Next() {
		var eval models.CriterionEvaluation
		var criterionJSON, evidenceJSON []byte
		var criterionType string

		if err := rows.Scan(&criterionType, &criterionJSON, &eval.Met, &evidenceJSON, &eval.Notes, &eval.EvaluatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan criteria evaluation: %w", err)
		}

		if err := json.Unmarshal(criterionJSON, &eval.Criterion); err != nil {
			return nil, fmt.Errorf("failed to unmarshal criterion: %w", err)
		}

		if len(evidenceJSON) > 0 {
			if err := json.Unmarshal(evidenceJSON, &eval.Evidence); err != nil {
				return nil, fmt.Errorf("failed to unmarshal evidence: %w", err)
			}
		}

		evaluations = append(evaluations, eval)
	}

	return evaluations, nil
}

// ListPendingSubmissions lists all pending PA submissions
func (r *PARepository) ListPendingSubmissions(ctx context.Context, limit, offset int) ([]models.PASubmission, error) {
	query := `
		SELECT id, patient_id, provider_id, drug_rxnorm, drug_name,
		       quantity, days_supply, payer_id, plan_id,
		       status, urgency_level, submitted_at
		FROM pa_submissions
		WHERE status IN ('PENDING', 'UNDER_REVIEW', 'NEED_INFO')
		ORDER BY
			CASE urgency_level
				WHEN 'EXPEDITED' THEN 1
				WHEN 'URGENT' THEN 2
				ELSE 3
			END,
			submitted_at ASC
		LIMIT $1 OFFSET $2
	`

	rows, err := r.db.QueryContext(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to list pending submissions: %w", err)
	}
	defer rows.Close()

	var submissions []models.PASubmission
	for rows.Next() {
		var sub models.PASubmission
		if err := rows.Scan(
			&sub.ID, &sub.PatientID, &sub.ProviderID, &sub.DrugRxNorm, &sub.DrugName,
			&sub.Quantity, &sub.DaysSupply, &sub.PayerID, &sub.PlanID,
			&sub.Status, &sub.UrgencyLevel, &sub.SubmittedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan submission: %w", err)
		}
		submissions = append(submissions, sub)
	}

	return submissions, nil
}

// Note: This file uses github.com/lib/pq for PostgreSQL array support
// Import should be: pq "github.com/lib/pq"
