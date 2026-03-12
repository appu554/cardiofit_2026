package rules

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"kb-1-drug-rules/internal/models"
)

// =============================================================================
// REPOSITORY - Database-backed drug rule access
// =============================================================================

// Repository provides database-backed drug rule access with caching
type Repository struct {
	db    *sql.DB
	log   *logrus.Entry
	cache Cache
}

// Cache interface for rule caching (Redis implementation)
type Cache interface {
	Get(ctx context.Context, key string) ([]byte, error)
	Set(ctx context.Context, key string, value []byte, ttl time.Duration) error
	Delete(ctx context.Context, key string) error
}

// NewRepository creates a new database-backed repository
func NewRepository(db *sql.DB, cache Cache, log *logrus.Entry) *Repository {
	return &Repository{
		db:    db,
		cache: cache,
		log:   log.WithField("component", "rules-repository"),
	}
}

// =============================================================================
// RULE RETRIEVAL
// =============================================================================

// GetByRxNorm retrieves a governed rule by RxNorm code and jurisdiction
// Falls back to GLOBAL jurisdiction if specific jurisdiction not found
// CRITICAL: Only returns ACTIVE rules for clinical safety
func (r *Repository) GetByRxNorm(ctx context.Context, rxnormCode, jurisdiction string) (*models.GovernedDrugRule, error) {
	return r.GetByRxNormWithStatus(ctx, rxnormCode, jurisdiction, true) // activeOnly = true
}

// GetByRxNormWithStatus retrieves a rule with optional approval status filtering
// Set activeOnly=true for production queries, false for admin/review queries
func (r *Repository) GetByRxNormWithStatus(ctx context.Context, rxnormCode, jurisdiction string, activeOnly bool) (*models.GovernedDrugRule, error) {
	// Try cache first (only for active rules)
	cacheKey := fmt.Sprintf("drug_rule:%s:%s", jurisdiction, rxnormCode)
	if activeOnly && r.cache != nil {
		if cached, err := r.cache.Get(ctx, cacheKey); err == nil && cached != nil {
			var rule models.GovernedDrugRule
			if err := json.Unmarshal(cached, &rule); err == nil {
				r.log.WithFields(logrus.Fields{
					"rxnorm_code":  rxnormCode,
					"jurisdiction": jurisdiction,
				}).Debug("Cache hit for drug rule")
				return &rule, nil
			}
		}
	}

	// Query database with jurisdiction fallback: specific -> GLOBAL
	// CRITICAL: Filter by approval_status = 'ACTIVE' for production safety
	var query string
	if activeOnly {
		query = `
			SELECT rule_data
			FROM drug_rules
			WHERE rxnorm_code = $1
			  AND jurisdiction IN ($2, 'GLOBAL')
			  AND (approval_status = 'ACTIVE' OR approval_status IS NULL)
			ORDER BY CASE jurisdiction WHEN $2 THEN 0 ELSE 1 END
			LIMIT 1
		`
	} else {
		query = `
			SELECT rule_data
			FROM drug_rules
			WHERE rxnorm_code = $1
			  AND jurisdiction IN ($2, 'GLOBAL')
			ORDER BY CASE jurisdiction WHEN $2 THEN 0 ELSE 1 END
			LIMIT 1
		`
	}

	var ruleJSON []byte
	err := r.db.QueryRowContext(ctx, query, rxnormCode, jurisdiction).Scan(&ruleJSON)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to query drug rule: %w", err)
	}

	var rule models.GovernedDrugRule
	if err := json.Unmarshal(ruleJSON, &rule); err != nil {
		return nil, fmt.Errorf("failed to unmarshal rule: %w", err)
	}

	// Cache the result (only for active rules)
	if activeOnly && r.cache != nil {
		r.cache.Set(ctx, cacheKey, ruleJSON, 5*time.Minute)
	}

	return &rule, nil
}

// GetByRxNormWithProvenance retrieves rule with full governance metadata
func (r *Repository) GetByRxNormWithProvenance(ctx context.Context, rxnormCode, jurisdiction string) (*models.GovernedDrugRule, *RuleProvenance, error) {
	query := `
		SELECT
			rule_data,
			authority,
			document_name,
			document_url,
			version,
			approved_by,
			approved_at,
			source_set_id,
			source_hash,
			ingested_at
		FROM drug_rules
		WHERE rxnorm_code = $1
		  AND jurisdiction IN ($2, 'GLOBAL')
		ORDER BY CASE jurisdiction WHEN $2 THEN 0 ELSE 1 END
		LIMIT 1
	`

	var (
		ruleJSON   []byte
		provenance RuleProvenance
	)

	err := r.db.QueryRowContext(ctx, query, rxnormCode, jurisdiction).Scan(
		&ruleJSON,
		&provenance.Authority,
		&provenance.Document,
		&provenance.URL,
		&provenance.Version,
		&provenance.ApprovedBy,
		&provenance.ApprovedAt,
		&provenance.SourceSetID,
		&provenance.SourceHash,
		&provenance.IngestedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil, nil
	}
	if err != nil {
		return nil, nil, fmt.Errorf("failed to query drug rule: %w", err)
	}

	var rule models.GovernedDrugRule
	if err := json.Unmarshal(ruleJSON, &rule); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal rule: %w", err)
	}

	return &rule, &provenance, nil
}

// RuleProvenance contains provenance information for audit
type RuleProvenance struct {
	Authority   string
	Document    string
	URL         string
	Version     string
	ApprovedBy  string
	ApprovedAt  time.Time
	SourceSetID string
	SourceHash  string
	IngestedAt  time.Time
}

// =============================================================================
// SEARCH
// =============================================================================

// SearchFilters for drug search
type SearchFilters struct {
	HighAlertOnly bool
	Category      string
	Authority     string
	HasBlackBox   bool
}

// Search searches for drug rules by name, class, or other criteria
func (r *Repository) Search(ctx context.Context, query string, jurisdiction string, filters SearchFilters) ([]*models.DrugRuleSummary, error) {
	sqlQuery := `
		SELECT
			rxnorm_code,
			drug_name,
			generic_name,
			drug_class,
			jurisdiction,
			is_high_alert,
			is_narrow_ti,
			has_black_box,
			authority,
			version
		FROM drug_rules
		WHERE jurisdiction IN ($1, 'GLOBAL')
		  AND (
			  drug_name ILIKE '%' || $2 || '%'
			  OR generic_name ILIKE '%' || $2 || '%'
			  OR drug_class ILIKE '%' || $2 || '%'
			  OR rxnorm_code = $2
		  )
	`

	args := []interface{}{jurisdiction, query}
	argNum := 3

	if filters.HighAlertOnly {
		sqlQuery += fmt.Sprintf(" AND is_high_alert = $%d", argNum)
		args = append(args, true)
		argNum++
	}

	if filters.HasBlackBox {
		sqlQuery += fmt.Sprintf(" AND has_black_box = $%d", argNum)
		args = append(args, true)
		argNum++
	}

	if filters.Category != "" {
		sqlQuery += fmt.Sprintf(" AND drug_class ILIKE '%%' || $%d || '%%'", argNum)
		args = append(args, filters.Category)
		argNum++
	}

	if filters.Authority != "" {
		sqlQuery += fmt.Sprintf(" AND authority = $%d", argNum)
		args = append(args, filters.Authority)
		argNum++
	}

	sqlQuery += " ORDER BY drug_name LIMIT 100"

	rows, err := r.db.QueryContext(ctx, sqlQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to search drugs: %w", err)
	}
	defer rows.Close()

	var results []*models.DrugRuleSummary
	for rows.Next() {
		var summary models.DrugRuleSummary
		if err := rows.Scan(
			&summary.RxNormCode,
			&summary.DrugName,
			&summary.GenericName,
			&summary.DrugClass,
			&summary.Jurisdiction,
			&summary.IsHighAlert,
			&summary.IsNarrowTI,
			&summary.HasBlackBox,
			&summary.Authority,
			&summary.Version,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		results = append(results, &summary)
	}

	return results, rows.Err()
}

// =============================================================================
// STATISTICS
// =============================================================================

// RepositoryStats contains repository statistics
type RepositoryStats struct {
	TotalDrugs     int        `json:"total_drugs"`
	USCount        int        `json:"us_count"`
	AUCount        int        `json:"au_count"`
	INCount        int        `json:"in_count"`
	GlobalCount    int        `json:"global_count"`
	HighAlertCount int        `json:"high_alert_count"`
	BlackBoxCount  int        `json:"black_box_count"`
	NarrowTICount  int        `json:"narrow_ti_count"`
	LastIngestion  *time.Time `json:"last_ingestion"`
}

// GetStats returns repository statistics
func (r *Repository) GetStats(ctx context.Context) (*RepositoryStats, error) {
	var stats RepositoryStats

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE jurisdiction = 'US') as us_count,
			COUNT(*) FILTER (WHERE jurisdiction = 'AU') as au_count,
			COUNT(*) FILTER (WHERE jurisdiction = 'IN') as in_count,
			COUNT(*) FILTER (WHERE jurisdiction = 'GLOBAL') as global_count,
			COUNT(*) FILTER (WHERE is_high_alert = TRUE) as high_alert_count,
			COUNT(*) FILTER (WHERE has_black_box = TRUE) as black_box_count,
			COUNT(*) FILTER (WHERE is_narrow_ti = TRUE) as narrow_ti_count,
			MAX(ingested_at) as last_ingestion
		FROM drug_rules
	`

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalDrugs,
		&stats.USCount,
		&stats.AUCount,
		&stats.INCount,
		&stats.GlobalCount,
		&stats.HighAlertCount,
		&stats.BlackBoxCount,
		&stats.NarrowTICount,
		&stats.LastIngestion,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get stats: %w", err)
	}

	return &stats, nil
}

// =============================================================================
// WRITE OPERATIONS (for ingestion)
// =============================================================================

// UpsertRule inserts or updates a drug rule
// CRITICAL: This method now includes approval workflow columns for clinical safety
func (r *Repository) UpsertRule(ctx context.Context, rule *models.GovernedDrugRule, runID string) error {
	ruleJSON, err := json.Marshal(rule)
	if err != nil {
		return fmt.Errorf("failed to marshal rule: %w", err)
	}

	// Convert extraction warnings to JSON for storage
	var warningsJSON []byte
	if len(rule.Governance.ExtractionWarnings) > 0 {
		warningsJSON, _ = json.Marshal(rule.Governance.ExtractionWarnings)
	}

	query := `
		INSERT INTO drug_rules (
			rxnorm_code, jurisdiction, drug_name, generic_name, drug_class,
			atc_code, snomed_code, rule_data, authority, document_name,
			document_section, document_url, evidence_level, source_hash,
			ingestion_run_id, approved_by, approved_at, version,
			is_high_alert, is_narrow_ti, has_black_box, is_beers_list,
			source_set_id,
			approval_status, risk_level, extraction_confidence, extraction_warnings,
			requires_manual_review
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10,
			$11, $12, $13, $14, $15::uuid, $16, $17, $18, $19, $20, $21, $22, $23,
			$24, $25, $26, $27, $28
		)
		ON CONFLICT (rxnorm_code, jurisdiction) DO UPDATE SET
			drug_name = EXCLUDED.drug_name,
			generic_name = EXCLUDED.generic_name,
			drug_class = EXCLUDED.drug_class,
			atc_code = EXCLUDED.atc_code,
			snomed_code = EXCLUDED.snomed_code,
			rule_data = EXCLUDED.rule_data,
			authority = EXCLUDED.authority,
			document_name = EXCLUDED.document_name,
			document_section = EXCLUDED.document_section,
			document_url = EXCLUDED.document_url,
			evidence_level = EXCLUDED.evidence_level,
			source_hash = EXCLUDED.source_hash,
			ingestion_run_id = EXCLUDED.ingestion_run_id,
			version = EXCLUDED.version,
			is_high_alert = EXCLUDED.is_high_alert,
			is_narrow_ti = EXCLUDED.is_narrow_ti,
			has_black_box = EXCLUDED.has_black_box,
			is_beers_list = EXCLUDED.is_beers_list,
			source_set_id = EXCLUDED.source_set_id,
			approval_status = EXCLUDED.approval_status,
			risk_level = EXCLUDED.risk_level,
			extraction_confidence = EXCLUDED.extraction_confidence,
			extraction_warnings = EXCLUDED.extraction_warnings,
			requires_manual_review = EXCLUDED.requires_manual_review
	`

	var runIDPtr *string
	if runID != "" {
		runIDPtr = &runID
	}

	// Determine beers_list status from geriatric dosing
	isBeers := false
	if rule.Dosing.Geriatric != nil && rule.Dosing.Geriatric.BeersListStatus != "" {
		isBeers = true
	}

	// Convert approval status and risk level to strings for database
	approvalStatus := string(rule.Governance.ApprovalStatus)
	if approvalStatus == "" {
		approvalStatus = "DRAFT" // Default to DRAFT for safety
	}
	riskLevel := string(rule.Governance.RiskLevel)
	if riskLevel == "" {
		riskLevel = "STANDARD"
	}

	_, err = r.db.ExecContext(ctx, query,
		rule.Drug.RxNormCode,
		rule.Governance.Jurisdiction,
		rule.Drug.Name,
		rule.Drug.GenericName,
		rule.Drug.DrugClass,
		rule.Drug.ATCCode,
		rule.Drug.SNOMEDCode,
		ruleJSON,
		rule.Governance.Authority,
		rule.Governance.Document,
		rule.Governance.Section,
		rule.Governance.URL,
		rule.Governance.EvidenceLevel,
		rule.Governance.SourceHash,
		runIDPtr,
		rule.Governance.ApprovedBy,
		rule.Governance.ApprovedAt,
		rule.Governance.Version,
		rule.Safety.HighAlertDrug,
		rule.Safety.NarrowTherapeuticIndex,
		rule.Safety.BlackBoxWarning,
		isBeers,
		rule.Governance.SourceSetID,
		approvalStatus,
		riskLevel,
		rule.Governance.ExtractionConfidence,
		warningsJSON,
		rule.Governance.RequiresManualReview,
	)

	if err != nil {
		return fmt.Errorf("failed to upsert rule: %w", err)
	}

	// Invalidate cache
	if r.cache != nil {
		cacheKey := fmt.Sprintf("drug_rule:%s:%s", rule.Governance.Jurisdiction, rule.Drug.RxNormCode)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// DeleteRule deletes a drug rule
func (r *Repository) DeleteRule(ctx context.Context, rxnormCode, jurisdiction string) error {
	query := `DELETE FROM drug_rules WHERE rxnorm_code = $1 AND jurisdiction = $2`

	result, err := r.db.ExecContext(ctx, query, rxnormCode, jurisdiction)
	if err != nil {
		return fmt.Errorf("failed to delete rule: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return sql.ErrNoRows
	}

	// Invalidate cache
	if r.cache != nil {
		cacheKey := fmt.Sprintf("drug_rule:%s:%s", jurisdiction, rxnormCode)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// =============================================================================
// INGESTION RUN TRACKING
// =============================================================================

// CreateIngestionRun creates a new ingestion run record
func (r *Repository) CreateIngestionRun(ctx context.Context, authority, jurisdiction, triggeredBy, triggerType string) (string, error) {
	runID := uuid.New().String()

	query := `
		INSERT INTO ingestion_runs (id, authority, jurisdiction, triggered_by, trigger_type)
		VALUES ($1, $2, $3, $4, $5)
	`

	_, err := r.db.ExecContext(ctx, query, runID, authority, jurisdiction, triggeredBy, triggerType)
	if err != nil {
		return "", fmt.Errorf("failed to create ingestion run: %w", err)
	}

	return runID, nil
}

// UpdateIngestionRun updates an ingestion run with final statistics
func (r *Repository) UpdateIngestionRun(ctx context.Context, runID string, status string, stats IngestionStats, errorMsg string) error {
	query := `
		UPDATE ingestion_runs
		SET
			status = $2,
			completed_at = NOW(),
			total_drugs_processed = $3,
			drugs_added = $4,
			drugs_updated = $5,
			drugs_unchanged = $6,
			drugs_failed = $7,
			error_message = $8
		WHERE id = $1
	`

	_, err := r.db.ExecContext(ctx, query,
		runID,
		status,
		stats.TotalProcessed,
		stats.Added,
		stats.Updated,
		stats.Unchanged,
		stats.Failed,
		errorMsg,
	)
	if err != nil {
		return fmt.Errorf("failed to update ingestion run: %w", err)
	}

	return nil
}

// IngestionStats tracks ingestion statistics
type IngestionStats struct {
	TotalProcessed int
	Added          int
	Updated        int
	Unchanged      int
	Failed         int
}

// LogIngestionItem logs a single drug processing result
func (r *Repository) LogIngestionItem(ctx context.Context, runID, rxnormCode, drugName, status, action, errorMsg string, processingTimeMs int) error {
	query := `
		INSERT INTO ingestion_items (ingestion_run_id, rxnorm_code, drug_name, status, action, error_message, processing_time_ms)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
	`

	_, err := r.db.ExecContext(ctx, query, runID, rxnormCode, drugName, status, action, errorMsg, processingTimeMs)
	if err != nil {
		return fmt.Errorf("failed to log ingestion item: %w", err)
	}

	return nil
}

// GetIngestionRuns retrieves recent ingestion runs
func (r *Repository) GetIngestionRuns(ctx context.Context, limit int) ([]IngestionRunSummary, error) {
	query := `
		SELECT
			id, authority, jurisdiction, status,
			started_at, completed_at,
			total_drugs_processed, drugs_added, drugs_updated, drugs_failed,
			triggered_by, trigger_type
		FROM ingestion_runs
		ORDER BY started_at DESC
		LIMIT $1
	`

	rows, err := r.db.QueryContext(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query ingestion runs: %w", err)
	}
	defer rows.Close()

	var runs []IngestionRunSummary
	for rows.Next() {
		var run IngestionRunSummary
		if err := rows.Scan(
			&run.ID,
			&run.Authority,
			&run.Jurisdiction,
			&run.Status,
			&run.StartedAt,
			&run.CompletedAt,
			&run.TotalProcessed,
			&run.Added,
			&run.Updated,
			&run.Failed,
			&run.TriggeredBy,
			&run.TriggerType,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		runs = append(runs, run)
	}

	return runs, rows.Err()
}

// IngestionRunSummary for listing ingestion runs
type IngestionRunSummary struct {
	ID             string
	Authority      string
	Jurisdiction   string
	Status         string
	StartedAt      time.Time
	CompletedAt    *time.Time
	TotalProcessed int
	Added          int
	Updated        int
	Failed         int
	TriggeredBy    string
	TriggerType    string
}

// =============================================================================
// AUDIT TRAIL
// =============================================================================

// GetRuleHistory retrieves change history for a drug rule
func (r *Repository) GetRuleHistory(ctx context.Context, rxnormCode string, limit int) ([]RuleHistoryEntry, error) {
	query := `
		SELECT
			id, drug_rule_id, jurisdiction, change_type,
			changed_fields, changed_by, change_reason, changed_at
		FROM drug_rule_history
		WHERE rxnorm_code = $1
		ORDER BY changed_at DESC
		LIMIT $2
	`

	rows, err := r.db.QueryContext(ctx, query, rxnormCode, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query rule history: %w", err)
	}
	defer rows.Close()

	var history []RuleHistoryEntry
	for rows.Next() {
		var entry RuleHistoryEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.DrugRuleID,
			&entry.Jurisdiction,
			&entry.ChangeType,
			&entry.ChangedFields,
			&entry.ChangedBy,
			&entry.ChangeReason,
			&entry.ChangedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		history = append(history, entry)
	}

	return history, rows.Err()
}

// RuleHistoryEntry represents a change history entry
type RuleHistoryEntry struct {
	ID            string
	DrugRuleID    *string
	Jurisdiction  string
	ChangeType    string
	ChangedFields []string
	ChangedBy     *string
	ChangeReason  *string
	ChangedAt     time.Time
}

// =============================================================================
// BULK OPERATIONS
// =============================================================================

// GetAllRxNormCodes retrieves all RxNorm codes for a jurisdiction
func (r *Repository) GetAllRxNormCodes(ctx context.Context, jurisdiction string) ([]string, error) {
	query := `
		SELECT rxnorm_code
		FROM drug_rules
		WHERE jurisdiction = $1
		ORDER BY rxnorm_code
	`

	rows, err := r.db.QueryContext(ctx, query, jurisdiction)
	if err != nil {
		return nil, fmt.Errorf("failed to query rxnorm codes: %w", err)
	}
	defer rows.Close()

	var codes []string
	for rows.Next() {
		var code string
		if err := rows.Scan(&code); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		codes = append(codes, code)
	}

	return codes, rows.Err()
}

// CheckSourceHash checks if a source document has changed
func (r *Repository) CheckSourceHash(ctx context.Context, rxnormCode, jurisdiction, sourceHash string) (bool, error) {
	query := `
		SELECT source_hash
		FROM drug_rules
		WHERE rxnorm_code = $1 AND jurisdiction = $2
	`

	var existingHash string
	err := r.db.QueryRowContext(ctx, query, rxnormCode, jurisdiction).Scan(&existingHash)
	if err == sql.ErrNoRows {
		return true, nil // New drug, needs processing
	}
	if err != nil {
		return false, fmt.Errorf("failed to check source hash: %w", err)
	}

	return existingHash != sourceHash, nil // True if hash changed
}

// =============================================================================
// APPROVAL WORKFLOW OPERATIONS
// =============================================================================

// PendingReviewFilter for filtering pending review queue
type PendingReviewFilter struct {
	RiskLevel    string
	MinConfidence int
	MaxConfidence int
	Jurisdiction string
	Limit        int
}

// GetPendingReviews retrieves rules pending pharmacist review
func (r *Repository) GetPendingReviews(ctx context.Context, filter PendingReviewFilter) ([]PendingReviewItem, error) {
	query := `
		SELECT
			id, rxnorm_code, drug_name, generic_name, drug_class,
			jurisdiction, authority, approval_status, risk_level,
			extraction_confidence, is_high_alert, has_black_box,
			source_set_id, document_url, ingested_at
		FROM drug_rules
		WHERE approval_status IN ('DRAFT', 'REVIEWED')
		  AND requires_manual_review = TRUE
	`

	args := []interface{}{}
	argNum := 1

	if filter.RiskLevel != "" {
		query += fmt.Sprintf(" AND risk_level = $%d", argNum)
		args = append(args, filter.RiskLevel)
		argNum++
	}

	if filter.Jurisdiction != "" {
		query += fmt.Sprintf(" AND jurisdiction = $%d", argNum)
		args = append(args, filter.Jurisdiction)
		argNum++
	}

	if filter.MinConfidence > 0 {
		query += fmt.Sprintf(" AND extraction_confidence >= $%d", argNum)
		args = append(args, filter.MinConfidence)
		argNum++
	}

	if filter.MaxConfidence > 0 {
		query += fmt.Sprintf(" AND extraction_confidence <= $%d", argNum)
		args = append(args, filter.MaxConfidence)
		argNum++
	}

	query += `
		ORDER BY
			CASE risk_level
				WHEN 'CRITICAL' THEN 0
				WHEN 'HIGH' THEN 1
				WHEN 'STANDARD' THEN 2
				ELSE 3
			END,
			extraction_confidence ASC NULLS FIRST,
			ingested_at DESC
	`

	limit := filter.Limit
	if limit <= 0 {
		limit = 100
	}
	query += fmt.Sprintf(" LIMIT $%d", argNum)
	args = append(args, limit)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query pending reviews: %w", err)
	}
	defer rows.Close()

	var items []PendingReviewItem
	for rows.Next() {
		var item PendingReviewItem
		if err := rows.Scan(
			&item.ID,
			&item.RxNormCode,
			&item.DrugName,
			&item.GenericName,
			&item.DrugClass,
			&item.Jurisdiction,
			&item.Authority,
			&item.ApprovalStatus,
			&item.RiskLevel,
			&item.ExtractionConfidence,
			&item.IsHighAlert,
			&item.HasBlackBox,
			&item.SourceSetID,
			&item.DocumentURL,
			&item.IngestedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		items = append(items, item)
	}

	return items, rows.Err()
}

// PendingReviewItem represents a rule awaiting pharmacist review
type PendingReviewItem struct {
	ID                   string
	RxNormCode           string
	DrugName             string
	GenericName          *string
	DrugClass            *string
	Jurisdiction         string
	Authority            string
	ApprovalStatus       string
	RiskLevel            *string
	ExtractionConfidence *int
	IsHighAlert          bool
	HasBlackBox          bool
	SourceSetID          *string
	DocumentURL          *string
	IngestedAt           time.Time
}

// ApproveRule transitions a rule to ACTIVE status
func (r *Repository) ApproveRule(ctx context.Context, ruleID, approvedBy, reviewNotes string, skipVerification bool) error {
	// Get current state
	var currentStatus, riskLevel string
	var rxnormCode, jurisdiction string
	err := r.db.QueryRowContext(ctx, `
		SELECT approval_status, risk_level, rxnorm_code, jurisdiction
		FROM drug_rules WHERE id = $1
	`, ruleID).Scan(&currentStatus, &riskLevel, &rxnormCode, &jurisdiction)

	if err == sql.ErrNoRows {
		return fmt.Errorf("rule not found: %s", ruleID)
	}
	if err != nil {
		return fmt.Errorf("failed to get rule state: %w", err)
	}

	// Validate transition
	if currentStatus != "DRAFT" && currentStatus != "REVIEWED" {
		return fmt.Errorf("cannot approve rule in status: %s", currentStatus)
	}

	// CRITICAL/HIGH risk drugs require explicit verification
	if (riskLevel == "CRITICAL" || riskLevel == "HIGH") && !skipVerification {
		return fmt.Errorf("high-risk drug (%s) requires explicit verification flag", riskLevel)
	}

	// Update the rule
	_, err = r.db.ExecContext(ctx, `
		UPDATE drug_rules
		SET approval_status = 'ACTIVE',
		    approved_by = $2,
		    approved_at = NOW(),
		    reviewed_by = $2,
		    reviewed_at = NOW(),
		    review_notes = $3,
		    requires_manual_review = FALSE
		WHERE id = $1
	`, ruleID, approvedBy, reviewNotes)

	if err != nil {
		return fmt.Errorf("failed to approve rule: %w", err)
	}

	// Log the approval
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO drug_rule_approvals (
			drug_rule_id, rxnorm_code, jurisdiction,
			previous_status, new_status,
			changed_by, change_reason, review_notes,
			risk_level, verified_against_source
		)
		VALUES ($1, $2, $3, $4, 'ACTIVE', $5, 'Approved for clinical use', $6, $7, $8)
	`, ruleID, rxnormCode, jurisdiction, currentStatus, approvedBy, reviewNotes, riskLevel, skipVerification)

	if err != nil {
		r.log.WithError(err).Warn("Failed to log approval audit")
	}

	// Invalidate cache
	if r.cache != nil {
		cacheKey := fmt.Sprintf("drug_rule:%s:%s", jurisdiction, rxnormCode)
		r.cache.Delete(ctx, cacheKey)
	}

	return nil
}

// RejectRule marks a rule as rejected
func (r *Repository) RejectRule(ctx context.Context, ruleID, rejectedBy, rejectionReason string) error {
	var currentStatus, rxnormCode, jurisdiction string
	err := r.db.QueryRowContext(ctx, `
		SELECT approval_status, rxnorm_code, jurisdiction
		FROM drug_rules WHERE id = $1
	`, ruleID).Scan(&currentStatus, &rxnormCode, &jurisdiction)

	if err == sql.ErrNoRows {
		return fmt.Errorf("rule not found: %s", ruleID)
	}
	if err != nil {
		return fmt.Errorf("failed to get rule state: %w", err)
	}

	// Update the rule
	_, err = r.db.ExecContext(ctx, `
		UPDATE drug_rules
		SET approval_status = 'RETIRED',
		    reviewed_by = $2,
		    reviewed_at = NOW(),
		    review_notes = $3,
		    requires_manual_review = FALSE
		WHERE id = $1
	`, ruleID, rejectedBy, rejectionReason)

	if err != nil {
		return fmt.Errorf("failed to reject rule: %w", err)
	}

	// Log the rejection
	_, err = r.db.ExecContext(ctx, `
		INSERT INTO drug_rule_approvals (
			drug_rule_id, rxnorm_code, jurisdiction,
			previous_status, new_status,
			changed_by, change_reason
		)
		VALUES ($1, $2, $3, $4, 'RETIRED', $5, $6)
	`, ruleID, rxnormCode, jurisdiction, currentStatus, rejectedBy, rejectionReason)

	if err != nil {
		r.log.WithError(err).Warn("Failed to log rejection audit")
	}

	return nil
}

// GetApprovalStats returns statistics about approval workflow
func (r *Repository) GetApprovalStats(ctx context.Context) (*ApprovalStats, error) {
	var stats ApprovalStats

	query := `
		SELECT
			COUNT(*) as total,
			COUNT(*) FILTER (WHERE approval_status = 'DRAFT') as draft_count,
			COUNT(*) FILTER (WHERE approval_status = 'REVIEWED') as reviewed_count,
			COUNT(*) FILTER (WHERE approval_status = 'ACTIVE') as active_count,
			COUNT(*) FILTER (WHERE approval_status = 'RETIRED') as retired_count,
			COUNT(*) FILTER (WHERE risk_level = 'CRITICAL' AND approval_status IN ('DRAFT', 'REVIEWED')) as critical_pending,
			COUNT(*) FILTER (WHERE risk_level = 'HIGH' AND approval_status IN ('DRAFT', 'REVIEWED')) as high_pending,
			COUNT(*) FILTER (WHERE extraction_confidence < 50 AND approval_status IN ('DRAFT', 'REVIEWED')) as low_confidence_pending
		FROM drug_rules
	`

	err := r.db.QueryRowContext(ctx, query).Scan(
		&stats.TotalRules,
		&stats.DraftCount,
		&stats.ReviewedCount,
		&stats.ActiveCount,
		&stats.RetiredCount,
		&stats.CriticalPending,
		&stats.HighPending,
		&stats.LowConfidencePending,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get approval stats: %w", err)
	}

	stats.PendingReview = stats.DraftCount + stats.ReviewedCount

	return &stats, nil
}

// ApprovalStats contains approval workflow statistics
type ApprovalStats struct {
	TotalRules           int `json:"total_rules"`
	DraftCount           int `json:"draft_count"`
	ReviewedCount        int `json:"reviewed_count"`
	ActiveCount          int `json:"active_count"`
	RetiredCount         int `json:"retired_count"`
	PendingReview        int `json:"pending_review"`
	CriticalPending      int `json:"critical_pending"`
	HighPending          int `json:"high_pending"`
	LowConfidencePending int `json:"low_confidence_pending"`
}
