package cdss

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"kb-7-terminology/internal/models"

	"github.com/lib/pq"
	"github.com/sirupsen/logrus"
)

// ============================================================================
// Rule Repository Interface
// ============================================================================

// RuleRepository provides database operations for clinical rules
type RuleRepository interface {
	// GetAllEnabledRules returns all enabled rules from the database
	GetAllEnabledRules(ctx context.Context) ([]ClinicalRule, error)

	// GetRulesByDomain returns enabled rules for a specific domain
	GetRulesByDomain(ctx context.Context, domain models.ClinicalDomain) ([]ClinicalRule, error)

	// GetRuleByID returns a specific rule by ID
	GetRuleByID(ctx context.Context, id string) (*ClinicalRule, error)

	// SaveRule inserts or updates a rule (upsert)
	SaveRule(ctx context.Context, rule *ClinicalRule) error

	// SaveRules saves multiple rules in a batch
	SaveRules(ctx context.Context, rules []ClinicalRule) error

	// DeleteRule removes a rule by ID
	DeleteRule(ctx context.Context, id string) error

	// RuleCount returns the number of rules in the database
	RuleCount(ctx context.Context) (int, error)

	// TableExists checks if the clinical_rules table exists
	TableExists(ctx context.Context) (bool, error)
}

// ============================================================================
// PostgreSQL Implementation
// ============================================================================

type postgresRuleRepository struct {
	db     *sql.DB
	logger *logrus.Logger
}

// NewPostgresRuleRepository creates a new PostgreSQL rule repository
func NewPostgresRuleRepository(db *sql.DB, logger *logrus.Logger) RuleRepository {
	return &postgresRuleRepository{
		db:     db,
		logger: logger,
	}
}

// TableExists checks if the clinical_rules table exists
func (r *postgresRuleRepository) TableExists(ctx context.Context) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'clinical_rules'
		)
	`
	var exists bool
	err := r.db.QueryRowContext(ctx, query).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("failed to check table existence: %w", err)
	}
	return exists, nil
}

// RuleCount returns the number of rules in the database
func (r *postgresRuleRepository) RuleCount(ctx context.Context) (int, error) {
	var count int
	err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM clinical_rules").Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("failed to count rules: %w", err)
	}
	return count, nil
}

// GetAllEnabledRules returns all enabled rules from the database
func (r *postgresRuleRepository) GetAllEnabledRules(ctx context.Context) ([]ClinicalRule, error) {
	query := `
		SELECT id, name, description, version, domain, severity, category,
			   conditions, alert_title, alert_description, recommendations,
			   guideline_references, enabled, priority, created_at, updated_at
		FROM clinical_rules
		WHERE enabled = true
		ORDER BY priority ASC, domain ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query enabled rules: %w", err)
	}
	defer rows.Close()

	return r.scanRules(rows)
}

// GetRulesByDomain returns enabled rules for a specific domain
func (r *postgresRuleRepository) GetRulesByDomain(ctx context.Context, domain models.ClinicalDomain) ([]ClinicalRule, error) {
	query := `
		SELECT id, name, description, version, domain, severity, category,
			   conditions, alert_title, alert_description, recommendations,
			   guideline_references, enabled, priority, created_at, updated_at
		FROM clinical_rules
		WHERE domain = $1 AND enabled = true
		ORDER BY priority ASC, created_at ASC
	`

	rows, err := r.db.QueryContext(ctx, query, string(domain))
	if err != nil {
		return nil, fmt.Errorf("failed to query rules by domain: %w", err)
	}
	defer rows.Close()

	return r.scanRules(rows)
}

// GetRuleByID returns a specific rule by ID
func (r *postgresRuleRepository) GetRuleByID(ctx context.Context, id string) (*ClinicalRule, error) {
	query := `
		SELECT id, name, description, version, domain, severity, category,
			   conditions, alert_title, alert_description, recommendations,
			   guideline_references, enabled, priority, created_at, updated_at
		FROM clinical_rules
		WHERE id = $1
	`

	row := r.db.QueryRowContext(ctx, query, id)
	rule, err := r.scanRule(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get rule by ID: %w", err)
	}
	return rule, nil
}

// SaveRule inserts or updates a rule (upsert)
func (r *postgresRuleRepository) SaveRule(ctx context.Context, rule *ClinicalRule) error {
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
	}

	query := `
		INSERT INTO clinical_rules (
			id, name, description, version, domain, severity, category,
			conditions, alert_title, alert_description, recommendations,
			guideline_references, enabled, priority, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			version = EXCLUDED.version,
			domain = EXCLUDED.domain,
			severity = EXCLUDED.severity,
			category = EXCLUDED.category,
			conditions = EXCLUDED.conditions,
			alert_title = EXCLUDED.alert_title,
			alert_description = EXCLUDED.alert_description,
			recommendations = EXCLUDED.recommendations,
			guideline_references = EXCLUDED.guideline_references,
			enabled = EXCLUDED.enabled,
			priority = EXCLUDED.priority,
			updated_at = NOW()
	`

	_, err = r.db.ExecContext(ctx, query,
		rule.ID,
		rule.Name,
		rule.Description,
		rule.Version,
		string(rule.Domain),
		string(rule.Severity),
		rule.Category,
		conditionsJSON,
		rule.AlertTitle,
		rule.AlertDescription,
		pq.Array(rule.Recommendations),
		pq.Array(rule.GuidelineReferences),
		rule.Enabled,
		rule.Priority,
		rule.CreatedAt,
		rule.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save rule: %w", err)
	}

	r.logger.WithFields(logrus.Fields{
		"rule_id": rule.ID,
		"name":    rule.Name,
	}).Debug("Saved clinical rule to database")

	return nil
}

// SaveRules saves multiple rules in a batch
func (r *postgresRuleRepository) SaveRules(ctx context.Context, rules []ClinicalRule) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback()

	query := `
		INSERT INTO clinical_rules (
			id, name, description, version, domain, severity, category,
			conditions, alert_title, alert_description, recommendations,
			guideline_references, enabled, priority, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			version = EXCLUDED.version,
			domain = EXCLUDED.domain,
			severity = EXCLUDED.severity,
			category = EXCLUDED.category,
			conditions = EXCLUDED.conditions,
			alert_title = EXCLUDED.alert_title,
			alert_description = EXCLUDED.alert_description,
			recommendations = EXCLUDED.recommendations,
			guideline_references = EXCLUDED.guideline_references,
			enabled = EXCLUDED.enabled,
			priority = EXCLUDED.priority,
			updated_at = NOW()
	`

	stmt, err := tx.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare statement: %w", err)
	}
	defer stmt.Close()

	for _, rule := range rules {
		conditionsJSON, err := json.Marshal(rule.Conditions)
		if err != nil {
			return fmt.Errorf("failed to marshal conditions for rule %s: %w", rule.ID, err)
		}

		_, err = stmt.ExecContext(ctx,
			rule.ID,
			rule.Name,
			rule.Description,
			rule.Version,
			string(rule.Domain),
			string(rule.Severity),
			rule.Category,
			conditionsJSON,
			rule.AlertTitle,
			rule.AlertDescription,
			pq.Array(rule.Recommendations),
			pq.Array(rule.GuidelineReferences),
			rule.Enabled,
			rule.Priority,
			rule.CreatedAt,
			rule.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to save rule %s: %w", rule.ID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	r.logger.WithField("count", len(rules)).Info("Saved clinical rules batch to database")
	return nil
}

// DeleteRule removes a rule by ID
func (r *postgresRuleRepository) DeleteRule(ctx context.Context, id string) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM clinical_rules WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	r.logger.WithFields(logrus.Fields{
		"rule_id":       id,
		"rows_affected": rowsAffected,
	}).Debug("Deleted clinical rule from database")

	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func (r *postgresRuleRepository) scanRules(rows *sql.Rows) ([]ClinicalRule, error) {
	var rules []ClinicalRule

	for rows.Next() {
		rule, err := r.scanRuleFromRows(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, *rule)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rules: %w", err)
	}

	return rules, nil
}

func (r *postgresRuleRepository) scanRuleFromRows(rows *sql.Rows) (*ClinicalRule, error) {
	var rule ClinicalRule
	var domainStr, severityStr string
	var conditionsJSON []byte
	var recommendations, guidelineRefs []string
	var description, alertDesc sql.NullString
	var createdAt, updatedAt time.Time

	err := rows.Scan(
		&rule.ID,
		&rule.Name,
		&description,
		&rule.Version,
		&domainStr,
		&severityStr,
		&rule.Category,
		&conditionsJSON,
		&rule.AlertTitle,
		&alertDesc,
		pq.Array(&recommendations),
		pq.Array(&guidelineRefs),
		&rule.Enabled,
		&rule.Priority,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to scan rule: %w", err)
	}

	rule.Description = description.String
	rule.AlertDescription = alertDesc.String
	rule.Domain = models.ClinicalDomain(domainStr)
	rule.Severity = models.CDSSAlertSeverity(severityStr)
	rule.Recommendations = recommendations
	rule.GuidelineReferences = guidelineRefs
	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt

	// Parse conditions JSON
	if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conditions for rule %s: %w", rule.ID, err)
	}

	return &rule, nil
}

func (r *postgresRuleRepository) scanRule(row *sql.Row) (*ClinicalRule, error) {
	var rule ClinicalRule
	var domainStr, severityStr string
	var conditionsJSON []byte
	var recommendations, guidelineRefs []string
	var description, alertDesc sql.NullString
	var createdAt, updatedAt time.Time

	err := row.Scan(
		&rule.ID,
		&rule.Name,
		&description,
		&rule.Version,
		&domainStr,
		&severityStr,
		&rule.Category,
		&conditionsJSON,
		&rule.AlertTitle,
		&alertDesc,
		pq.Array(&recommendations),
		pq.Array(&guidelineRefs),
		&rule.Enabled,
		&rule.Priority,
		&createdAt,
		&updatedAt,
	)
	if err != nil {
		return nil, err
	}

	rule.Description = description.String
	rule.AlertDescription = alertDesc.String
	rule.Domain = models.ClinicalDomain(domainStr)
	rule.Severity = models.CDSSAlertSeverity(severityStr)
	rule.Recommendations = recommendations
	rule.GuidelineReferences = guidelineRefs
	rule.CreatedAt = createdAt
	rule.UpdatedAt = updatedAt

	// Parse conditions JSON
	if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
		return nil, fmt.Errorf("failed to unmarshal conditions for rule %s: %w", rule.ID, err)
	}

	return &rule, nil
}
