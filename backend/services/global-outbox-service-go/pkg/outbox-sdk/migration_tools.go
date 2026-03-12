package outboxsdk

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sirupsen/logrus"
)

// MigrationTool provides utilities for managing outbox table schemas
type MigrationTool struct {
	pool   *pgxpool.Pool
	logger *logrus.Logger
}

// NewMigrationTool creates a new migration tool
func NewMigrationTool(databaseURL string, logger *logrus.Logger) (*MigrationTool, error) {
	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to create database pool: %w", err)
	}

	if err := pool.Ping(context.Background()); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	return &MigrationTool{
		pool:   pool,
		logger: logger,
	}, nil
}

// Close closes the migration tool
func (mt *MigrationTool) Close() {
	if mt.pool != nil {
		mt.pool.Close()
	}
}

// CreateOutboxTable creates the outbox table for a service
func (mt *MigrationTool) CreateOutboxTable(ctx context.Context, serviceName string) error {
	tableName := fmt.Sprintf("outbox_events_%s", strings.ReplaceAll(serviceName, "-", "_"))
	
	mt.logger.Infof("Creating outbox table: %s", tableName)

	query := fmt.Sprintf(`
		-- Create the outbox events table
		CREATE TABLE IF NOT EXISTS %s (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			service_name VARCHAR(255) NOT NULL,
			event_type VARCHAR(255) NOT NULL,
			event_data JSONB NOT NULL,
			topic VARCHAR(255) NOT NULL,
			correlation_id VARCHAR(255),
			priority INTEGER NOT NULL DEFAULT 5,
			metadata JSONB DEFAULT '{}',
			medical_context VARCHAR(50) NOT NULL DEFAULT 'routine',
			created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
			published_at TIMESTAMP WITH TIME ZONE,
			retry_count INTEGER NOT NULL DEFAULT 0,
			status VARCHAR(20) NOT NULL DEFAULT 'pending',
			error_message TEXT,
			next_retry_at TIMESTAMP WITH TIME ZONE,
			
			-- Constraints
			CONSTRAINT %s_priority_check CHECK (priority BETWEEN 1 AND 10),
			CONSTRAINT %s_medical_context_check CHECK (medical_context IN ('critical', 'urgent', 'routine', 'background')),
			CONSTRAINT %s_status_check CHECK (status IN ('pending', 'published', 'failed', 'dead_letter'))
		);

		-- Create indexes for optimal query performance
		CREATE INDEX IF NOT EXISTS idx_%s_status ON %s (status) WHERE status IN ('pending', 'failed');
		CREATE INDEX IF NOT EXISTS idx_%s_created_at ON %s (created_at);
		CREATE INDEX IF NOT EXISTS idx_%s_priority ON %s (priority DESC);
		CREATE INDEX IF NOT EXISTS idx_%s_medical_context ON %s (medical_context);
		CREATE INDEX IF NOT EXISTS idx_%s_next_retry ON %s (next_retry_at) WHERE next_retry_at IS NOT NULL;
		CREATE INDEX IF NOT EXISTS idx_%s_service_status ON %s (service_name, status);
		CREATE INDEX IF NOT EXISTS idx_%s_correlation ON %s (correlation_id) WHERE correlation_id IS NOT NULL;

		-- Create a partial index for pending events with priority
		CREATE INDEX IF NOT EXISTS idx_%s_pending_priority ON %s (
			CASE medical_context
				WHEN 'critical' THEN 1
				WHEN 'urgent' THEN 2
				WHEN 'routine' THEN 3
				ELSE 4
			END,
			priority DESC,
			created_at ASC
		) WHERE status = 'pending';

		-- Add comments for documentation
		COMMENT ON TABLE %s IS 'Transactional outbox events for service: %s';
		COMMENT ON COLUMN %s.id IS 'Unique event identifier';
		COMMENT ON COLUMN %s.service_name IS 'Name of the service that created this event';
		COMMENT ON COLUMN %s.event_type IS 'Type of event (e.g., user.created, order.updated)';
		COMMENT ON COLUMN %s.event_data IS 'JSON payload of the event';
		COMMENT ON COLUMN %s.topic IS 'Kafka topic to publish to';
		COMMENT ON COLUMN %s.medical_context IS 'Medical priority: critical, urgent, routine, background';
		COMMENT ON COLUMN %s.priority IS 'Event priority (1-10, 10 = highest)';
		COMMENT ON COLUMN %s.status IS 'Event status: pending, published, failed, dead_letter';
	`, 
		tableName, 
		tableName, tableName, tableName,           // Constraint names
		tableName, tableName,                      // Status index
		tableName, tableName,                      // Created_at index
		tableName, tableName,                      // Priority index  
		tableName, tableName,                      // Medical_context index
		tableName, tableName,                      // Next_retry index
		tableName, tableName,                      // Service_status index
		tableName, tableName,                      // Correlation index
		tableName, tableName,                      // Pending_priority index
		tableName, serviceName,                    // Table comment
		tableName, tableName, tableName, tableName, tableName, tableName, tableName, tableName, // Column comments
	)

	_, err := mt.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create outbox table %s: %w", tableName, err)
	}

	mt.logger.Infof("Successfully created outbox table: %s", tableName)
	return nil
}

// DropOutboxTable drops the outbox table for a service (use with caution)
func (mt *MigrationTool) DropOutboxTable(ctx context.Context, serviceName string, confirm bool) error {
	if !confirm {
		return fmt.Errorf("must set confirm=true to drop table")
	}

	tableName := fmt.Sprintf("outbox_events_%s", strings.ReplaceAll(serviceName, "-", "_"))
	
	mt.logger.Warnf("Dropping outbox table: %s", tableName)

	query := fmt.Sprintf("DROP TABLE IF EXISTS %s CASCADE", tableName)
	_, err := mt.pool.Exec(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to drop table %s: %w", tableName, err)
	}

	mt.logger.Warnf("Successfully dropped outbox table: %s", tableName)
	return nil
}

// UpgradeOutboxTable upgrades an existing outbox table to the latest schema
func (mt *MigrationTool) UpgradeOutboxTable(ctx context.Context, serviceName string) error {
	tableName := fmt.Sprintf("outbox_events_%s", strings.ReplaceAll(serviceName, "-", "_"))
	
	mt.logger.Infof("Upgrading outbox table: %s", tableName)

	// Check if table exists
	exists, err := mt.tableExists(ctx, tableName)
	if err != nil {
		return fmt.Errorf("failed to check if table exists: %w", err)
	}

	if !exists {
		mt.logger.Infof("Table %s doesn't exist, creating new table", tableName)
		return mt.CreateOutboxTable(ctx, serviceName)
	}

	// Get current table schema
	columns, err := mt.getTableColumns(ctx, tableName)
	if err != nil {
		return fmt.Errorf("failed to get table columns: %w", err)
	}

	// Apply schema migrations based on missing columns
	migrations := mt.generateMigrations(tableName, columns)
	
	for _, migration := range migrations {
		mt.logger.Infof("Applying migration: %s", migration.Description)
		_, err := mt.pool.Exec(ctx, migration.Query)
		if err != nil {
			return fmt.Errorf("migration failed: %s - %w", migration.Description, err)
		}
	}

	if len(migrations) > 0 {
		mt.logger.Infof("Successfully applied %d migrations to table: %s", len(migrations), tableName)
	} else {
		mt.logger.Infof("Table %s is already up to date", tableName)
	}

	return nil
}

// GetOutboxTableInfo returns information about an outbox table
func (mt *MigrationTool) GetOutboxTableInfo(ctx context.Context, serviceName string) (*TableInfo, error) {
	tableName := fmt.Sprintf("outbox_events_%s", strings.ReplaceAll(serviceName, "-", "_"))

	// Check if table exists
	exists, err := mt.tableExists(ctx, tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to check table existence: %w", err)
	}

	info := &TableInfo{
		ServiceName: serviceName,
		TableName:   tableName,
		Exists:      exists,
	}

	if !exists {
		return info, nil
	}

	// Get table size information
	sizeQuery := `
		SELECT 
			pg_total_relation_size($1) as total_size,
			pg_relation_size($1) as table_size,
			(SELECT reltuples::bigint FROM pg_class WHERE relname = $2) as estimated_rows
	`
	
	err = mt.pool.QueryRow(ctx, sizeQuery, tableName, tableName).Scan(
		&info.TotalSize,
		&info.TableSize,
		&info.EstimatedRows,
	)
	if err != nil {
		mt.logger.Warnf("Failed to get table size info: %v", err)
	}

	// Get index information
	indexQuery := `
		SELECT indexname, indexdef 
		FROM pg_indexes 
		WHERE tablename = $1 
		ORDER BY indexname
	`
	
	rows, err := mt.pool.Query(ctx, indexQuery, tableName)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var indexName, indexDef string
			if err := rows.Scan(&indexName, &indexDef); err == nil {
				info.Indexes = append(info.Indexes, IndexInfo{
					Name:       indexName,
					Definition: indexDef,
				})
			}
		}
	}

	// Get column information
	columns, err := mt.getTableColumns(ctx, tableName)
	if err == nil {
		info.Columns = columns
	}

	return info, nil
}

// ListAllOutboxTables lists all outbox tables in the database
func (mt *MigrationTool) ListAllOutboxTables(ctx context.Context) ([]string, error) {
	query := `
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = 'public' 
		  AND table_name LIKE 'outbox_events_%'
		ORDER BY table_name
	`

	rows, err := mt.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to query outbox tables: %w", err)
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			continue
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}

// Helper types and methods

// Migration represents a database migration
type Migration struct {
	Description string
	Query       string
}

// TableInfo contains information about an outbox table
type TableInfo struct {
	ServiceName   string       `json:"service_name"`
	TableName     string       `json:"table_name"`
	Exists        bool         `json:"exists"`
	TotalSize     int64        `json:"total_size"`
	TableSize     int64        `json:"table_size"`
	EstimatedRows int64        `json:"estimated_rows"`
	Columns       []ColumnInfo `json:"columns"`
	Indexes       []IndexInfo  `json:"indexes"`
}

// ColumnInfo contains information about a table column
type ColumnInfo struct {
	Name         string `json:"name"`
	DataType     string `json:"data_type"`
	IsNullable   string `json:"is_nullable"`
	DefaultValue string `json:"default_value"`
}

// IndexInfo contains information about a table index
type IndexInfo struct {
	Name       string `json:"name"`
	Definition string `json:"definition"`
}

func (mt *MigrationTool) tableExists(ctx context.Context, tableName string) (bool, error) {
	query := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_schema = 'public' 
			  AND table_name = $1
		)
	`
	
	var exists bool
	err := mt.pool.QueryRow(ctx, query, tableName).Scan(&exists)
	return exists, err
}

func (mt *MigrationTool) getTableColumns(ctx context.Context, tableName string) ([]ColumnInfo, error) {
	query := `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			COALESCE(column_default, '') as column_default
		FROM information_schema.columns 
		WHERE table_schema = 'public' 
		  AND table_name = $1
		ORDER BY ordinal_position
	`

	rows, err := mt.pool.Query(ctx, query, tableName)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []ColumnInfo
	for rows.Next() {
		var col ColumnInfo
		if err := rows.Scan(&col.Name, &col.DataType, &col.IsNullable, &col.DefaultValue); err != nil {
			continue
		}
		columns = append(columns, col)
	}

	return columns, nil
}

func (mt *MigrationTool) generateMigrations(tableName string, existingColumns []ColumnInfo) []Migration {
	var migrations []Migration

	// Check for missing columns and add them
	requiredColumns := map[string]string{
		"id":              "UUID PRIMARY KEY DEFAULT gen_random_uuid()",
		"service_name":    "VARCHAR(255) NOT NULL",
		"event_type":      "VARCHAR(255) NOT NULL",
		"event_data":      "JSONB NOT NULL",
		"topic":           "VARCHAR(255) NOT NULL",
		"correlation_id":  "VARCHAR(255)",
		"priority":        "INTEGER NOT NULL DEFAULT 5",
		"metadata":        "JSONB DEFAULT '{}'",
		"medical_context": "VARCHAR(50) NOT NULL DEFAULT 'routine'",
		"created_at":      "TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()",
		"published_at":    "TIMESTAMP WITH TIME ZONE",
		"retry_count":     "INTEGER NOT NULL DEFAULT 0",
		"status":          "VARCHAR(20) NOT NULL DEFAULT 'pending'",
		"error_message":   "TEXT",
		"next_retry_at":   "TIMESTAMP WITH TIME ZONE",
	}

	// Create a map of existing columns
	existingMap := make(map[string]bool)
	for _, col := range existingColumns {
		existingMap[col.Name] = true
	}

	// Add missing columns
	for columnName, columnDef := range requiredColumns {
		if !existingMap[columnName] {
			migrations = append(migrations, Migration{
				Description: fmt.Sprintf("Add column %s", columnName),
				Query:       fmt.Sprintf("ALTER TABLE %s ADD COLUMN IF NOT EXISTS %s %s", tableName, columnName, columnDef),
			})
		}
	}

	return migrations
}