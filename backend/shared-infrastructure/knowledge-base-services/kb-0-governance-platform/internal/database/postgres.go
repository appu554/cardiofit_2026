// Package database provides PostgreSQL persistence for the KB-0 governance platform.
package database

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// POSTGRES STORE
// =============================================================================

// Store provides PostgreSQL persistence for knowledge items.
type Store struct {
	db *sql.DB
}

// NewStore creates a new PostgreSQL store.
func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

// GetItem retrieves a knowledge item by ID.
func (s *Store) GetItem(ctx context.Context, itemID string) (*models.KnowledgeItem, error) {
	query := `
		SELECT
			item_id, kb, item_type, name, description,
			content_ref, content_hash,
			source_authority, source_document, source_section, source_url,
			source_jurisdiction, source_effective_date, source_expiration_date,
			risk_level, workflow_template, requires_dual_review,
			risk_flags, state, version,
			governance_trail,
			created_at, updated_at, activated_at, retired_at
		FROM knowledge_items
		WHERE item_id = $1
	`

	item := &models.KnowledgeItem{}
	var riskFlagsJSON, governanceJSON []byte
	var effectiveDate, expirationDate sql.NullString
	var activatedAt, retiredAt sql.NullTime

	err := s.db.QueryRowContext(ctx, query, itemID).Scan(
		&item.ID, &item.KB, &item.Type, &item.Name, &item.Description,
		&item.ContentRef, &item.ContentHash,
		&item.Source.Authority, &item.Source.Document, &item.Source.Section, &item.Source.URL,
		&item.Source.Jurisdiction, &effectiveDate, &expirationDate,
		&item.RiskLevel, &item.WorkflowTemplate, &item.RequiresDualReview,
		&riskFlagsJSON, &item.State, &item.Version,
		&governanceJSON,
		&item.CreatedAt, &item.UpdatedAt, &activatedAt, &retiredAt,
	)
	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("item not found: %s", itemID)
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get item: %w", err)
	}

	// Parse JSON fields
	if len(riskFlagsJSON) > 0 {
		json.Unmarshal(riskFlagsJSON, &item.RiskFlags)
	}
	if len(governanceJSON) > 0 {
		json.Unmarshal(governanceJSON, &item.Governance)
	}

	// Parse nullable fields
	if effectiveDate.Valid {
		item.Source.EffectiveDate = effectiveDate.String
	}
	if expirationDate.Valid {
		item.Source.ExpirationDate = expirationDate.String
	}
	if activatedAt.Valid {
		item.ActiveAt = &activatedAt.Time
	}
	if retiredAt.Valid {
		item.RetiredAt = &retiredAt.Time
	}

	return item, nil
}

// UpdateItem updates a knowledge item.
func (s *Store) UpdateItem(ctx context.Context, item *models.KnowledgeItem) error {
	riskFlagsJSON, _ := json.Marshal(item.RiskFlags)
	governanceJSON, _ := json.Marshal(item.Governance)

	query := `
		UPDATE knowledge_items SET
			state = $2,
			risk_flags = $3,
			governance_trail = $4,
			updated_at = $5,
			activated_at = $6,
			retired_at = $7
		WHERE item_id = $1
	`

	_, err := s.db.ExecContext(ctx, query,
		item.ID,
		item.State,
		riskFlagsJSON,
		governanceJSON,
		item.UpdatedAt,
		item.ActiveAt,
		item.RetiredAt,
	)
	if err != nil {
		return fmt.Errorf("failed to update item: %w", err)
	}

	return nil
}

// CreateItem creates a new knowledge item.
func (s *Store) CreateItem(ctx context.Context, item *models.KnowledgeItem) error {
	riskFlagsJSON, _ := json.Marshal(item.RiskFlags)
	governanceJSON, _ := json.Marshal(item.Governance)

	query := `
		INSERT INTO knowledge_items (
			item_id, kb, item_type, name, description,
			content_ref, content_hash,
			source_authority, source_document, source_section, source_url,
			source_jurisdiction, source_effective_date, source_expiration_date,
			risk_level, workflow_template, requires_dual_review,
			risk_flags, state, version,
			created_by, governance_trail,
			created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, $5,
			$6, $7,
			$8, $9, $10, $11,
			$12, $13, $14,
			$15, $16, $17,
			$18, $19, $20,
			$21, $22,
			$23, $24
		)
	`

	_, err := s.db.ExecContext(ctx, query,
		item.ID, item.KB, item.Type, item.Name, item.Description,
		item.ContentRef, item.ContentHash,
		item.Source.Authority, item.Source.Document, item.Source.Section, item.Source.URL,
		item.Source.Jurisdiction, item.Source.EffectiveDate, item.Source.ExpirationDate,
		item.RiskLevel, item.WorkflowTemplate, item.RequiresDualReview,
		riskFlagsJSON, item.State, item.Version,
		item.Governance.CreatedBy, governanceJSON,
		item.CreatedAt, item.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to create item: %w", err)
	}

	return nil
}

// GetItemsByState returns items in specified states for a KB.
func (s *Store) GetItemsByState(ctx context.Context, kb models.KB, states []models.ItemState) ([]*models.KnowledgeItem, error) {
	// Build state list
	stateStrings := make([]interface{}, len(states))
	placeholders := ""
	for i, state := range states {
		stateStrings[i] = string(state)
		if i > 0 {
			placeholders += ", "
		}
		placeholders += fmt.Sprintf("$%d", i+2)
	}

	query := fmt.Sprintf(`
		SELECT
			item_id, kb, item_type, name,
			source_authority, source_jurisdiction,
			risk_level, workflow_template, requires_dual_review,
			state, version,
			created_at, updated_at
		FROM knowledge_items
		WHERE kb = $1 AND state IN (%s)
		ORDER BY risk_level DESC, created_at ASC
	`, placeholders)

	args := append([]interface{}{kb}, stateStrings...)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query items by state: %w", err)
	}
	defer rows.Close()

	var items []*models.KnowledgeItem
	for rows.Next() {
		item := &models.KnowledgeItem{}
		err := rows.Scan(
			&item.ID, &item.KB, &item.Type, &item.Name,
			&item.Source.Authority, &item.Source.Jurisdiction,
			&item.RiskLevel, &item.WorkflowTemplate, &item.RequiresDualReview,
			&item.State, &item.Version,
			&item.CreatedAt, &item.UpdatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to scan item: %w", err)
		}
		items = append(items, item)
	}

	return items, nil
}

// GetMetrics returns governance metrics for a KB.
func (s *Store) GetMetrics(ctx context.Context, kb models.KB) (*Metrics, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE state = 'ACTIVE') AS active_count,
			COUNT(*) FILTER (WHERE state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE')) AS pending_review_count,
			COUNT(*) FILTER (WHERE state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL')) AS pending_approval_count,
			COUNT(*) FILTER (WHERE state = 'HOLD') AS hold_count,
			COUNT(*) FILTER (WHERE state = 'EMERGENCY_ACTIVE') AS emergency_count,
			COUNT(*) FILTER (WHERE state = 'RETIRED') AS retired_count,
			COUNT(*) FILTER (WHERE state = 'REJECTED') AS rejected_count,
			COUNT(*) FILTER (WHERE state = 'ACTIVE' AND risk_level = 'HIGH') AS high_risk_active,
			COUNT(*) AS total_count
		FROM knowledge_items
		WHERE kb = $1
	`

	metrics := &Metrics{KB: kb}
	err := s.db.QueryRowContext(ctx, query, kb).Scan(
		&metrics.ActiveCount,
		&metrics.PendingReviewCount,
		&metrics.PendingApprovalCount,
		&metrics.HoldCount,
		&metrics.EmergencyCount,
		&metrics.RetiredCount,
		&metrics.RejectedCount,
		&metrics.HighRiskActiveCount,
		&metrics.TotalCount,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get metrics: %w", err)
	}

	return metrics, nil
}

// GetCrossKBMetrics returns metrics across all KBs.
func (s *Store) GetCrossKBMetrics(ctx context.Context) (*CrossKBMetrics, error) {
	query := `
		SELECT
			COUNT(*) FILTER (WHERE state = 'ACTIVE') AS total_active,
			COUNT(*) FILTER (WHERE state IN ('DRAFT', 'PRIMARY_REVIEW', 'SECONDARY_REVIEW', 'REVISE')) AS total_pending_review,
			COUNT(*) FILTER (WHERE state IN ('REVIEWED', 'CMO_APPROVAL', 'DIRECTOR_APPROVAL', 'LEAD_APPROVAL')) AS total_pending_approval,
			COUNT(*) FILTER (WHERE state = 'HOLD') AS total_hold,
			COUNT(*) FILTER (WHERE state = 'EMERGENCY_ACTIVE') AS total_emergency,
			COUNT(*) FILTER (WHERE state = 'ACTIVE' AND risk_level = 'HIGH') AS total_high_risk_active,
			COUNT(DISTINCT kb) AS active_kbs,
			COUNT(*) AS total_items
		FROM knowledge_items
	`

	metrics := &CrossKBMetrics{}
	err := s.db.QueryRowContext(ctx, query).Scan(
		&metrics.TotalActive,
		&metrics.TotalPendingReview,
		&metrics.TotalPendingApproval,
		&metrics.TotalHold,
		&metrics.TotalEmergency,
		&metrics.TotalHighRiskActive,
		&metrics.ActiveKBs,
		&metrics.TotalItems,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get cross-KB metrics: %w", err)
	}

	return metrics, nil
}

// Metrics contains governance metrics for a single KB.
type Metrics struct {
	KB                  models.KB `json:"kb"`
	ActiveCount         int       `json:"active_count"`
	PendingReviewCount  int       `json:"pending_review_count"`
	PendingApprovalCount int      `json:"pending_approval_count"`
	HoldCount           int       `json:"hold_count"`
	EmergencyCount      int       `json:"emergency_count"`
	RetiredCount        int       `json:"retired_count"`
	RejectedCount       int       `json:"rejected_count"`
	HighRiskActiveCount int       `json:"high_risk_active_count"`
	TotalCount          int       `json:"total_count"`
}

// CrossKBMetrics contains aggregated metrics across all KBs.
type CrossKBMetrics struct {
	TotalActive         int       `json:"total_active"`
	TotalPendingReview  int       `json:"total_pending_review"`
	TotalPendingApproval int      `json:"total_pending_approval"`
	TotalHold           int       `json:"total_hold"`
	TotalEmergency      int       `json:"total_emergency"`
	TotalHighRiskActive int       `json:"total_high_risk_active"`
	ActiveKBs           int       `json:"active_kbs"`
	TotalItems          int       `json:"total_items"`
	GeneratedAt         time.Time `json:"generated_at"`
}
