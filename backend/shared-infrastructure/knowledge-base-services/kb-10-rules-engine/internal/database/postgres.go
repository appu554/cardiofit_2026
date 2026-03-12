// Package database provides PostgreSQL operations for the Clinical Rules Engine
package database

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/kb-10-rules-engine/internal/config"
	"github.com/cardiofit/kb-10-rules-engine/internal/models"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// PostgresDB provides database operations
type PostgresDB struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger
}

// NewPostgresDB creates a new PostgreSQL database connection
func NewPostgresDB(cfg *config.DatabaseConfig, logger *logrus.Logger) (*PostgresDB, error) {
	connString := fmt.Sprintf(
		"postgres://%s:%s@%s:%d/%s?sslmode=%s&pool_max_conns=%d&pool_min_conns=%d",
		cfg.User, cfg.Password, cfg.Host, cfg.Port, cfg.Name, cfg.SSLMode,
		cfg.MaxConnections, cfg.MinConnections,
	)

	poolConfig, err := pgxpool.ParseConfig(connString)
	if err != nil {
		return nil, fmt.Errorf("failed to parse connection string: %w", err)
	}

	poolConfig.MaxConnLifetime = cfg.MaxConnLifetime
	poolConfig.MaxConnIdleTime = cfg.MaxConnIdleTime

	pool, err := pgxpool.NewWithConfig(context.Background(), poolConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"host": cfg.Host,
		"port": cfg.Port,
		"db":   cfg.Name,
	}).Info("Connected to PostgreSQL")

	return &PostgresDB{
		pool:   pool,
		logger: logger,
	}, nil
}

// Close closes the database connection pool
func (db *PostgresDB) Close() {
	db.pool.Close()
}

// Ping checks the database connection
func (db *PostgresDB) Ping(ctx context.Context) error {
	return db.pool.Ping(ctx)
}

// RunMigrations runs the database migrations
func (db *PostgresDB) RunMigrations() error {
	ctx := context.Background()

	migrations := []string{
		// Rules table
		`CREATE TABLE IF NOT EXISTS rules (
			id VARCHAR(100) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			description TEXT,
			type VARCHAR(50) NOT NULL,
			category VARCHAR(100) NOT NULL,
			severity VARCHAR(50),
			status VARCHAR(50) DEFAULT 'ACTIVE',
			priority INTEGER DEFAULT 100,
			version VARCHAR(50),
			conditions JSONB NOT NULL,
			condition_logic VARCHAR(10) DEFAULT 'AND',
			actions JSONB NOT NULL,
			evidence JSONB,
			tags TEXT[],
			metadata JSONB,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Rules indexes
		`CREATE INDEX IF NOT EXISTS idx_rules_type ON rules(type)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_category ON rules(category)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_severity ON rules(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_status ON rules(status)`,
		`CREATE INDEX IF NOT EXISTS idx_rules_tags ON rules USING GIN(tags)`,

		// Alerts table
		`CREATE TABLE IF NOT EXISTS alerts (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			rule_id VARCHAR(100) NOT NULL,
			rule_name VARCHAR(255),
			patient_id VARCHAR(100) NOT NULL,
			encounter_id VARCHAR(100),
			severity VARCHAR(50) NOT NULL,
			category VARCHAR(100),
			message TEXT NOT NULL,
			details TEXT,
			context JSONB,
			status VARCHAR(50) DEFAULT 'active',
			priority VARCHAR(50),
			acknowledged_by VARCHAR(100),
			acknowledged_at TIMESTAMP WITH TIME ZONE,
			resolved_by VARCHAR(100),
			resolved_at TIMESTAMP WITH TIME ZONE,
			resolution TEXT,
			expires_at TIMESTAMP WITH TIME ZONE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Alerts indexes
		`CREATE INDEX IF NOT EXISTS idx_alerts_patient ON alerts(patient_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_status ON alerts(status)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_severity ON alerts(severity)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_rule ON alerts(rule_id)`,
		`CREATE INDEX IF NOT EXISTS idx_alerts_created ON alerts(created_at)`,

		// Rule executions table (audit trail)
		`CREATE TABLE IF NOT EXISTS rule_executions (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			rule_id VARCHAR(100) NOT NULL,
			rule_name VARCHAR(255),
			patient_id VARCHAR(100) NOT NULL,
			encounter_id VARCHAR(100),
			triggered BOOLEAN NOT NULL,
			context JSONB,
			result JSONB,
			execution_time_ms FLOAT,
			cache_hit BOOLEAN DEFAULT FALSE,
			error TEXT,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,

		// Executions indexes
		`CREATE INDEX IF NOT EXISTS idx_executions_rule ON rule_executions(rule_id)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_patient ON rule_executions(patient_id)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_created ON rule_executions(created_at)`,
		`CREATE INDEX IF NOT EXISTS idx_executions_triggered ON rule_executions(triggered)`,

		// Updated_at trigger function
		`CREATE OR REPLACE FUNCTION update_updated_at_column()
		RETURNS TRIGGER AS $$
		BEGIN
			NEW.updated_at = NOW();
			RETURN NEW;
		END;
		$$ LANGUAGE plpgsql`,

		// Triggers for updated_at
		`DROP TRIGGER IF EXISTS update_rules_updated_at ON rules`,
		`CREATE TRIGGER update_rules_updated_at
			BEFORE UPDATE ON rules
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column()`,

		`DROP TRIGGER IF EXISTS update_alerts_updated_at ON alerts`,
		`CREATE TRIGGER update_alerts_updated_at
			BEFORE UPDATE ON alerts
			FOR EACH ROW
			EXECUTE FUNCTION update_updated_at_column()`,
	}

	for _, migration := range migrations {
		if _, err := db.pool.Exec(ctx, migration); err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}

	db.logger.Info("Database migrations completed")
	return nil
}

// --- Rule Operations ---

// SaveRule saves a rule to the database
func (db *PostgresDB) SaveRule(ctx context.Context, rule *models.Rule) error {
	conditionsJSON, err := json.Marshal(rule.Conditions)
	if err != nil {
		return fmt.Errorf("failed to marshal conditions: %w", err)
	}

	actionsJSON, err := json.Marshal(rule.Actions)
	if err != nil {
		return fmt.Errorf("failed to marshal actions: %w", err)
	}

	evidenceJSON, err := json.Marshal(rule.Evidence)
	if err != nil {
		return fmt.Errorf("failed to marshal evidence: %w", err)
	}

	metadataJSON, err := json.Marshal(rule.Metadata)
	if err != nil {
		return fmt.Errorf("failed to marshal metadata: %w", err)
	}

	query := `
		INSERT INTO rules (id, name, description, type, category, severity, status, priority, version,
			conditions, condition_logic, actions, evidence, tags, metadata, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17)
		ON CONFLICT (id) DO UPDATE SET
			name = EXCLUDED.name,
			description = EXCLUDED.description,
			type = EXCLUDED.type,
			category = EXCLUDED.category,
			severity = EXCLUDED.severity,
			status = EXCLUDED.status,
			priority = EXCLUDED.priority,
			version = EXCLUDED.version,
			conditions = EXCLUDED.conditions,
			condition_logic = EXCLUDED.condition_logic,
			actions = EXCLUDED.actions,
			evidence = EXCLUDED.evidence,
			tags = EXCLUDED.tags,
			metadata = EXCLUDED.metadata,
			updated_at = NOW()`

	_, err = db.pool.Exec(ctx, query,
		rule.ID, rule.Name, rule.Description, rule.Type, rule.Category,
		rule.Severity, rule.Status, rule.Priority, rule.Version,
		conditionsJSON, rule.ConditionLogic, actionsJSON, evidenceJSON,
		rule.Tags, metadataJSON, rule.CreatedAt, rule.UpdatedAt,
	)

	return err
}

// GetRule retrieves a rule by ID
func (db *PostgresDB) GetRule(ctx context.Context, id string) (*models.Rule, error) {
	query := `
		SELECT id, name, description, type, category, severity, status, priority, version,
			conditions, condition_logic, actions, evidence, tags, metadata, created_at, updated_at
		FROM rules
		WHERE id = $1`

	row := db.pool.QueryRow(ctx, query, id)
	return scanRule(row)
}

// ListRules retrieves rules with optional filters
func (db *PostgresDB) ListRules(ctx context.Context, filter *models.Filter) ([]*models.Rule, error) {
	query := `
		SELECT id, name, description, type, category, severity, status, priority, version,
			conditions, condition_logic, actions, evidence, tags, metadata, created_at, updated_at
		FROM rules
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	if filter != nil {
		if len(filter.Types) > 0 {
			query += fmt.Sprintf(" AND type = ANY($%d)", argIndex)
			args = append(args, filter.Types)
			argIndex++
		}
		if len(filter.Categories) > 0 {
			query += fmt.Sprintf(" AND category = ANY($%d)", argIndex)
			args = append(args, filter.Categories)
			argIndex++
		}
		if len(filter.Statuses) > 0 {
			query += fmt.Sprintf(" AND status = ANY($%d)", argIndex)
			args = append(args, filter.Statuses)
			argIndex++
		}
	}

	query += " ORDER BY priority ASC, created_at DESC"

	if filter != nil {
		if filter.Limit > 0 {
			query += fmt.Sprintf(" LIMIT %d", filter.Limit)
		}
		if filter.Offset > 0 {
			query += fmt.Sprintf(" OFFSET %d", filter.Offset)
		}
	}

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var rules []*models.Rule
	for rows.Next() {
		rule, err := scanRuleFromRows(rows)
		if err != nil {
			return nil, err
		}
		rules = append(rules, rule)
	}

	return rules, rows.Err()
}

// DeleteRule deletes a rule by ID
func (db *PostgresDB) DeleteRule(ctx context.Context, id string) error {
	_, err := db.pool.Exec(ctx, "DELETE FROM rules WHERE id = $1", id)
	return err
}

// --- Alert Operations ---

// CreateAlert creates a new alert
func (db *PostgresDB) CreateAlert(ctx context.Context, alert *models.Alert) error {
	if alert.ID == "" {
		alert.ID = uuid.New().String()
	}

	contextJSON, err := json.Marshal(alert.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	query := `
		INSERT INTO alerts (id, rule_id, rule_name, patient_id, encounter_id, severity, category,
			message, details, context, status, priority, expires_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`

	_, err = db.pool.Exec(ctx, query,
		alert.ID, alert.RuleID, alert.RuleName, alert.PatientID, alert.EncounterID,
		alert.Severity, alert.Category, alert.Message, alert.Details, contextJSON,
		alert.Status, alert.Priority, alert.ExpiresAt, alert.CreatedAt, alert.UpdatedAt,
	)

	return err
}

// GetAlert retrieves an alert by ID
func (db *PostgresDB) GetAlert(ctx context.Context, id string) (*models.Alert, error) {
	query := `
		SELECT id, rule_id, rule_name, patient_id, encounter_id, severity, category, message, details,
			context, status, priority, acknowledged_by, acknowledged_at, resolved_by, resolved_at,
			resolution, expires_at, created_at, updated_at
		FROM alerts
		WHERE id = $1`

	row := db.pool.QueryRow(ctx, query, id)
	return scanAlert(row)
}

// ListAlerts retrieves alerts with optional filters
func (db *PostgresDB) ListAlerts(ctx context.Context, status string, severity string, limit, offset int) ([]*models.Alert, error) {
	query := `
		SELECT id, rule_id, rule_name, patient_id, encounter_id, severity, category, message, details,
			context, status, priority, acknowledged_by, acknowledged_at, resolved_by, resolved_at,
			resolution, expires_at, created_at, updated_at
		FROM alerts
		WHERE 1=1`

	args := []interface{}{}
	argIndex := 1

	if status != "" {
		query += fmt.Sprintf(" AND status = $%d", argIndex)
		args = append(args, status)
		argIndex++
	}
	if severity != "" {
		query += fmt.Sprintf(" AND severity = $%d", argIndex)
		args = append(args, severity)
		argIndex++
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT %d", limit)
	}
	if offset > 0 {
		query += fmt.Sprintf(" OFFSET %d", offset)
	}

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		alert, err := scanAlertFromRows(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

// GetPatientAlerts retrieves alerts for a specific patient
func (db *PostgresDB) GetPatientAlerts(ctx context.Context, patientID string, status string) ([]*models.Alert, error) {
	query := `
		SELECT id, rule_id, rule_name, patient_id, encounter_id, severity, category, message, details,
			context, status, priority, acknowledged_by, acknowledged_at, resolved_by, resolved_at,
			resolution, expires_at, created_at, updated_at
		FROM alerts
		WHERE patient_id = $1`

	args := []interface{}{patientID}
	if status != "" {
		query += " AND status = $2"
		args = append(args, status)
	}

	query += " ORDER BY created_at DESC"

	rows, err := db.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var alerts []*models.Alert
	for rows.Next() {
		alert, err := scanAlertFromRows(rows)
		if err != nil {
			return nil, err
		}
		alerts = append(alerts, alert)
	}

	return alerts, rows.Err()
}

// AcknowledgeAlert marks an alert as acknowledged
func (db *PostgresDB) AcknowledgeAlert(ctx context.Context, id, acknowledgedBy string) error {
	now := time.Now()
	_, err := db.pool.Exec(ctx,
		`UPDATE alerts SET status = $1, acknowledged_by = $2, acknowledged_at = $3 WHERE id = $4`,
		models.AlertStatusAcknowledged, acknowledgedBy, now, id,
	)
	return err
}

// ResolveAlert marks an alert as resolved
func (db *PostgresDB) ResolveAlert(ctx context.Context, id, resolvedBy, resolution string) error {
	now := time.Now()
	_, err := db.pool.Exec(ctx,
		`UPDATE alerts SET status = $1, resolved_by = $2, resolved_at = $3, resolution = $4 WHERE id = $5`,
		models.AlertStatusResolved, resolvedBy, now, resolution, id,
	)
	return err
}

// --- Audit Operations ---

// RecordExecution records a rule execution for audit purposes
func (db *PostgresDB) RecordExecution(ctx context.Context, exec *models.RuleExecution) error {
	if exec.ID == "" {
		exec.ID = uuid.New().String()
	}

	contextJSON, err := json.Marshal(exec.Context)
	if err != nil {
		return fmt.Errorf("failed to marshal context: %w", err)
	}

	query := `
		INSERT INTO rule_executions (id, rule_id, rule_name, patient_id, encounter_id, triggered,
			context, result, execution_time_ms, cache_hit, error, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)`

	_, err = db.pool.Exec(ctx, query,
		exec.ID, exec.RuleID, exec.RuleName, exec.PatientID, exec.EncounterID,
		exec.Triggered, contextJSON, exec.Result, exec.ExecutionTimeMs, exec.CacheHit,
		exec.Error, exec.CreatedAt,
	)

	return err
}

// GetExecutionStats retrieves execution statistics for a rule
func (db *PostgresDB) GetExecutionStats(ctx context.Context, ruleID string) (*models.ExecutionStats, error) {
	query := `
		SELECT
			rule_id,
			COUNT(*) as total_executions,
			SUM(CASE WHEN triggered THEN 1 ELSE 0 END) as trigger_count,
			AVG(execution_time_ms) as avg_execution_ms,
			SUM(CASE WHEN cache_hit THEN 1 ELSE 0 END)::float / NULLIF(COUNT(*), 0) as cache_hit_rate,
			MAX(created_at) as last_executed_at,
			MAX(CASE WHEN triggered THEN created_at END) as last_triggered_at
		FROM rule_executions
		WHERE rule_id = $1
		GROUP BY rule_id`

	var stats models.ExecutionStats
	err := db.pool.QueryRow(ctx, query, ruleID).Scan(
		&stats.RuleID,
		&stats.TotalExecutions,
		&stats.TriggerCount,
		&stats.AvgExecutionMs,
		&stats.CacheHitRate,
		&stats.LastExecutedAt,
		&stats.LastTriggeredAt,
	)

	if err == pgx.ErrNoRows {
		return &models.ExecutionStats{RuleID: ruleID}, nil
	}

	if err != nil {
		return nil, err
	}

	if stats.TotalExecutions > 0 {
		stats.TriggerRate = float64(stats.TriggerCount) / float64(stats.TotalExecutions)
	}

	return &stats, nil
}

// --- Helper Functions ---

func scanRule(row pgx.Row) (*models.Rule, error) {
	var rule models.Rule
	var conditionsJSON, actionsJSON, evidenceJSON, metadataJSON []byte

	err := row.Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.Type, &rule.Category,
		&rule.Severity, &rule.Status, &rule.Priority, &rule.Version,
		&conditionsJSON, &rule.ConditionLogic, &actionsJSON, &evidenceJSON,
		&rule.Tags, &metadataJSON, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(actionsJSON, &rule.Actions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(evidenceJSON, &rule.Evidence); err != nil {
		return nil, err
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &rule.Metadata); err != nil {
			return nil, err
		}
	}

	return &rule, nil
}

func scanRuleFromRows(rows pgx.Rows) (*models.Rule, error) {
	var rule models.Rule
	var conditionsJSON, actionsJSON, evidenceJSON, metadataJSON []byte

	err := rows.Scan(
		&rule.ID, &rule.Name, &rule.Description, &rule.Type, &rule.Category,
		&rule.Severity, &rule.Status, &rule.Priority, &rule.Version,
		&conditionsJSON, &rule.ConditionLogic, &actionsJSON, &evidenceJSON,
		&rule.Tags, &metadataJSON, &rule.CreatedAt, &rule.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(conditionsJSON, &rule.Conditions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(actionsJSON, &rule.Actions); err != nil {
		return nil, err
	}
	if err := json.Unmarshal(evidenceJSON, &rule.Evidence); err != nil {
		return nil, err
	}
	if len(metadataJSON) > 0 {
		if err := json.Unmarshal(metadataJSON, &rule.Metadata); err != nil {
			return nil, err
		}
	}

	return &rule, nil
}

func scanAlert(row pgx.Row) (*models.Alert, error) {
	var alert models.Alert
	var contextJSON []byte

	err := row.Scan(
		&alert.ID, &alert.RuleID, &alert.RuleName, &alert.PatientID, &alert.EncounterID,
		&alert.Severity, &alert.Category, &alert.Message, &alert.Details,
		&contextJSON, &alert.Status, &alert.Priority, &alert.AcknowledgedBy,
		&alert.AcknowledgedAt, &alert.ResolvedBy, &alert.ResolvedAt,
		&alert.Resolution, &alert.ExpiresAt, &alert.CreatedAt, &alert.UpdatedAt,
	)

	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	if len(contextJSON) > 0 {
		if err := json.Unmarshal(contextJSON, &alert.Context); err != nil {
			return nil, err
		}
	}

	return &alert, nil
}

func scanAlertFromRows(rows pgx.Rows) (*models.Alert, error) {
	var alert models.Alert
	var contextJSON []byte

	err := rows.Scan(
		&alert.ID, &alert.RuleID, &alert.RuleName, &alert.PatientID, &alert.EncounterID,
		&alert.Severity, &alert.Category, &alert.Message, &alert.Details,
		&contextJSON, &alert.Status, &alert.Priority, &alert.AcknowledgedBy,
		&alert.AcknowledgedAt, &alert.ResolvedBy, &alert.ResolvedAt,
		&alert.Resolution, &alert.ExpiresAt, &alert.CreatedAt, &alert.UpdatedAt,
	)

	if err != nil {
		return nil, err
	}

	if len(contextJSON) > 0 {
		if err := json.Unmarshal(contextJSON, &alert.Context); err != nil {
			return nil, err
		}
	}

	return &alert, nil
}
