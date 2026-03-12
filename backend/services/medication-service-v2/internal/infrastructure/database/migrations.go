package database

import (
	"context"
	"database/sql"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"
	"go.uber.org/zap"
)

// Migration represents a database migration
type Migration struct {
	Version     string
	Name        string
	Up          string
	Down        string
	AppliedAt   *time.Time
	Checksum    string
	Description string
}

// MigrationManager handles database migrations with up/down support
type MigrationManager struct {
	db               *PostgreSQL
	logger           *zap.Logger
	migrationsTable  string
	migrationsPath   string
	enableChecksums  bool
	enableTransactions bool
}

// NewMigrationManager creates a new migration manager with enhanced features
func NewMigrationManager(db *PostgreSQL, logger *zap.Logger, opts ...MigrationOption) *MigrationManager {
	mm := &MigrationManager{
		db:               db,
		logger:           logger,
		migrationsTable:  "schema_migrations",
		migrationsPath:   "./migrations",
		enableChecksums:  true,
		enableTransactions: true,
	}

	for _, opt := range opts {
		opt(mm)
	}

	return mm
}

// MigrationOption allows customization of migration manager
type MigrationOption func(*MigrationManager)

// WithMigrationsTable sets custom migrations table name
func WithMigrationsTable(tableName string) MigrationOption {
	return func(mm *MigrationManager) {
		mm.migrationsTable = tableName
	}
}

// WithMigrationsPath sets custom migrations directory path
func WithMigrationsPath(path string) MigrationOption {
	return func(mm *MigrationManager) {
		mm.migrationsPath = path
	}
}

// WithChecksums enables/disables checksum validation
func WithChecksums(enabled bool) MigrationOption {
	return func(mm *MigrationManager) {
		mm.enableChecksums = enabled
	}
}

// WithTransactions enables/disables transactional migrations
func WithTransactions(enabled bool) MigrationOption {
	return func(mm *MigrationManager) {
		mm.enableTransactions = enabled
	}
}

// MigrationResult contains the result of a migration operation
type MigrationResult struct {
	Applied       []Migration `json:"applied"`
	Failed        []Migration `json:"failed"`
	Skipped       []Migration `json:"skipped"`
	ExecutionTime time.Duration `json:"execution_time"`
	TotalCount    int         `json:"total_count"`
	Success       bool        `json:"success"`
	ErrorMessage  string      `json:"error_message,omitempty"`
}

// Initialize creates the migrations tracking infrastructure
func (mm *MigrationManager) Initialize(ctx context.Context) error {
	// Create enhanced migrations table
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(500) NOT NULL,
			checksum VARCHAR(64),
			applied_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			applied_by VARCHAR(255) DEFAULT CURRENT_USER,
			execution_time_ms INTEGER DEFAULT 0,
			description TEXT,
			migration_type VARCHAR(20) DEFAULT 'up' CHECK (migration_type IN ('up', 'down')),
			success BOOLEAN DEFAULT TRUE,
			error_message TEXT,
			-- Audit fields for HIPAA compliance
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		-- Indexes for performance
		CREATE INDEX IF NOT EXISTS idx_%s_applied_at ON %s(applied_at DESC);
		CREATE INDEX IF NOT EXISTS idx_%s_success ON %s(success, applied_at DESC);
		CREATE INDEX IF NOT EXISTS idx_%s_type ON %s(migration_type, applied_at DESC);
	`, mm.migrationsTable, strings.ReplaceAll(mm.migrationsTable, ".", "_"), mm.migrationsTable,
		strings.ReplaceAll(mm.migrationsTable, ".", "_"), mm.migrationsTable,
		strings.ReplaceAll(mm.migrationsTable, ".", "_"), mm.migrationsTable)

	_, err := mm.db.DB.ExecContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	mm.logger.Info("Migration tracking infrastructure initialized",
		zap.String("table", mm.migrationsTable))

	return nil
}

// GetAppliedMigrations returns all applied migrations
func (mm *MigrationManager) GetAppliedMigrations(ctx context.Context) ([]Migration, error) {
	query := fmt.Sprintf(`
		SELECT version, name, checksum, applied_at, execution_time_ms, 
		       description, migration_type, success, error_message
		FROM %s 
		ORDER BY version ASC`, mm.migrationsTable)

	var migrations []Migration
	rows, err := mm.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to get applied migrations: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var m Migration
		var executionTimeMs sql.NullInt32
		err := rows.Scan(&m.Version, &m.Name, &m.Checksum, &m.AppliedAt, 
			&executionTimeMs, &m.Description, &sql.NullString{}, &sql.NullBool{}, &sql.NullString{})
		if err != nil {
			continue
		}
		migrations = append(migrations, m)
	}

	return migrations, nil
}

// GetPendingMigrations returns migrations that haven't been applied
func (mm *MigrationManager) GetPendingMigrations(ctx context.Context) ([]Migration, error) {
	availableMigrations, err := mm.loadMigrationsFromFiles()
	if err != nil {
		return nil, err
	}

	appliedMigrations, err := mm.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	// Create map of applied versions
	appliedMap := make(map[string]bool)
	for _, applied := range appliedMigrations {
		appliedMap[applied.Version] = true
	}

	// Filter pending migrations
	var pending []Migration
	for _, available := range availableMigrations {
		if !appliedMap[available.Version] {
			pending = append(pending, available)
		}
	}

	// Sort by version
	sort.Slice(pending, func(i, j int) bool {
		return pending[i].Version < pending[j].Version
	})

	return pending, nil
}

// ApplyMigrations applies all pending migrations
func (mm *MigrationManager) ApplyMigrations(ctx context.Context, dryRun bool) (*MigrationResult, error) {
	startTime := time.Now()
	result := &MigrationResult{
		Applied: []Migration{},
		Failed:  []Migration{},
		Skipped: []Migration{},
	}

	pending, err := mm.GetPendingMigrations(ctx)
	if err != nil {
		result.Success = false
		result.ErrorMessage = err.Error()
		return result, err
	}

	result.TotalCount = len(pending)

	if len(pending) == 0 {
		mm.logger.Info("No pending migrations found")
		result.Success = true
		result.ExecutionTime = time.Since(startTime)
		return result, nil
	}

	mm.logger.Info("Applying migrations", zap.Int("count", len(pending)), zap.Bool("dry_run", dryRun))

	for _, migration := range pending {
		if dryRun {
			mm.logger.Info("DRY RUN: Would apply migration", 
				zap.String("version", migration.Version), 
				zap.String("name", migration.Name))
			result.Skipped = append(result.Skipped, migration)
			continue
		}

		migrationStart := time.Now()
		err := mm.applyMigration(ctx, migration)
		executionTime := time.Since(migrationStart)

		if err != nil {
			mm.logger.Error("Migration failed", 
				zap.String("version", migration.Version),
				zap.String("name", migration.Name),
				zap.Error(err))
			
			migration.Description = err.Error()
			result.Failed = append(result.Failed, migration)
			
			// Record failed migration
			mm.recordMigration(ctx, migration, false, executionTime, err.Error())
			
			result.Success = false
			result.ErrorMessage = fmt.Sprintf("Migration %s failed: %v", migration.Version, err)
			break
		}

		mm.logger.Info("Migration applied successfully", 
			zap.String("version", migration.Version),
			zap.String("name", migration.Name),
			zap.Duration("execution_time", executionTime))

		result.Applied = append(result.Applied, migration)
		mm.recordMigration(ctx, migration, true, executionTime, "")
	}

	result.ExecutionTime = time.Since(startTime)
	result.Success = len(result.Failed) == 0

	return result, nil
}

// RollbackMigration rolls back a specific migration
func (mm *MigrationManager) RollbackMigration(ctx context.Context, version string, dryRun bool) error {
	migration, err := mm.getMigrationByVersion(version)
	if err != nil {
		return err
	}

	if migration.Down == "" {
		return fmt.Errorf("migration %s has no down migration", version)
	}

	// Check if migration is applied
	isApplied, err := mm.IsMigrationApplied(ctx, version)
	if err != nil {
		return err
	}

	if !isApplied {
		return fmt.Errorf("migration %s is not applied", version)
	}

	if dryRun {
		mm.logger.Info("DRY RUN: Would rollback migration", 
			zap.String("version", version))
		return nil
	}

	mm.logger.Info("Rolling back migration", zap.String("version", version))

	// Execute rollback
	err = mm.executeSQL(ctx, migration.Down)
	if err != nil {
		return fmt.Errorf("failed to execute rollback for migration %s: %w", version, err)
	}

	// Remove from migrations table
	query := fmt.Sprintf("DELETE FROM %s WHERE version = $1", mm.migrationsTable)
	_, err = mm.db.DB.ExecContext(ctx, query, version)
	if err != nil {
		return fmt.Errorf("failed to remove migration record: %w", err)
	}

	mm.logger.Info("Migration rolled back successfully", zap.String("version", version))
	return nil
}

// applyMigration applies a single migration
func (mm *MigrationManager) applyMigration(ctx context.Context, migration Migration) error {
	if migration.Up == "" {
		return fmt.Errorf("migration %s has no up SQL", migration.Version)
	}

	// Validate checksum if enabled
	if mm.enableChecksums {
		expectedChecksum := mm.calculateChecksum(migration.Up)
		if migration.Checksum != "" && migration.Checksum != expectedChecksum {
			return fmt.Errorf("checksum mismatch for migration %s", migration.Version)
		}
		migration.Checksum = expectedChecksum
	}

	// Execute migration
	return mm.executeSQL(ctx, migration.Up)
}

// executeSQL executes SQL with optional transaction support
func (mm *MigrationManager) executeSQL(ctx context.Context, sql string) error {
	if !mm.enableTransactions {
		_, err := mm.db.DB.ExecContext(ctx, sql)
		return err
	}

	// Use transaction
	return mm.db.WithTransaction(ctx, func(tx *sqlx.Tx) error {
		_, err := tx.ExecContext(ctx, sql)
		return err
	})
}

// recordMigration records migration execution in tracking table
func (mm *MigrationManager) recordMigration(ctx context.Context, migration Migration, success bool, executionTime time.Duration, errorMsg string) {
	query := fmt.Sprintf(`
		INSERT INTO %s (version, name, checksum, execution_time_ms, description, migration_type, success, error_message)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (version) DO UPDATE SET
			applied_at = NOW(),
			execution_time_ms = $4,
			success = $7,
			error_message = $8,
			updated_at = NOW()`,
		mm.migrationsTable)

	_, err := mm.db.DB.ExecContext(ctx, query, 
		migration.Version, migration.Name, migration.Checksum, 
		int(executionTime.Milliseconds()), migration.Description, 
		"up", success, errorMsg)
	
	if err != nil {
		mm.logger.Error("Failed to record migration", zap.Error(err))
	}
}

// IsMigrationApplied checks if a migration is applied
func (mm *MigrationManager) IsMigrationApplied(ctx context.Context, version string) (bool, error) {
	query := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE version = $1 AND success = TRUE", mm.migrationsTable)
	
	var count int
	err := mm.db.DB.GetContext(ctx, &count, query, version)
	if err != nil {
		return false, fmt.Errorf("failed to check migration status: %w", err)
	}
	
	return count > 0, nil
}

// loadMigrationsFromFiles loads migrations from the file system
func (mm *MigrationManager) loadMigrationsFromFiles() ([]Migration, error) {
	var migrations []Migration

	// Walk through migrations directory
	err := filepath.WalkDir(mm.migrationsPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() || !strings.HasSuffix(d.Name(), ".sql") {
			return nil
		}

		migration, err := mm.parseMigrationFile(path)
		if err != nil {
			mm.logger.Warn("Failed to parse migration file", 
				zap.String("file", path), zap.Error(err))
			return nil // Continue with other files
		}

		migrations = append(migrations, migration)
		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to load migrations: %w", err)
	}

	// Sort by version
	sort.Slice(migrations, func(i, j int) bool {
		return migrations[i].Version < migrations[j].Version
	})

	return migrations, nil
}

// parseMigrationFile parses a migration file
func (mm *MigrationManager) parseMigrationFile(filePath string) (Migration, error) {
	content, err := fs.ReadFile(nil, filePath) // This needs to be adapted based on your file system
	if err != nil {
		return Migration{}, fmt.Errorf("failed to read migration file: %w", err)
	}

	fileName := filepath.Base(filePath)
	parts := strings.Split(fileName, "_")
	if len(parts) < 2 {
		return Migration{}, fmt.Errorf("invalid migration file name format: %s", fileName)
	}

	version := parts[0]
	name := strings.TrimSuffix(strings.Join(parts[1:], "_"), ".sql")

	// Parse content for up/down sections
	contentStr := string(content)
	upSQL, downSQL := mm.parseUpDownSQL(contentStr)

	migration := Migration{
		Version:     version,
		Name:        name,
		Up:          upSQL,
		Down:        downSQL,
		Description: fmt.Sprintf("Migration %s: %s", version, name),
	}

	if mm.enableChecksums {
		migration.Checksum = mm.calculateChecksum(upSQL)
	}

	return migration, nil
}

// parseUpDownSQL parses up and down SQL from content
func (mm *MigrationManager) parseUpDownSQL(content string) (up, down string) {
	lines := strings.Split(content, "\n")
	
	var upLines, downLines []string
	currentSection := "up"
	
	for _, line := range lines {
		line = strings.TrimSpace(line)
		
		if strings.HasPrefix(line, "-- +migrate Up") {
			currentSection = "up"
			continue
		}
		
		if strings.HasPrefix(line, "-- +migrate Down") {
			currentSection = "down"
			continue
		}
		
		if currentSection == "up" {
			upLines = append(upLines, line)
		} else {
			downLines = append(downLines, line)
		}
	}
	
	return strings.Join(upLines, "\n"), strings.Join(downLines, "\n")
}

// calculateChecksum calculates SHA-256 checksum
func (mm *MigrationManager) calculateChecksum(content string) string {
	// Implement SHA-256 checksum calculation
	// This is a placeholder - implement with crypto/sha256
	return fmt.Sprintf("sha256_%d", len(content))
}

// getMigrationByVersion gets migration by version
func (mm *MigrationManager) getMigrationByVersion(version string) (Migration, error) {
	migrations, err := mm.loadMigrationsFromFiles()
	if err != nil {
		return Migration{}, err
	}

	for _, migration := range migrations {
		if migration.Version == version {
			return migration, nil
		}
	}

	return Migration{}, fmt.Errorf("migration %s not found", version)
}

// GetMigrationStatus returns the current migration status
func (mm *MigrationManager) GetMigrationStatus(ctx context.Context) (map[string]interface{}, error) {
	appliedMigrations, err := mm.GetAppliedMigrations(ctx)
	if err != nil {
		return nil, err
	}

	pendingMigrations, err := mm.GetPendingMigrations(ctx)
	if err != nil {
		return nil, err
	}

	availableMigrations, err := mm.loadMigrationsFromFiles()
	if err != nil {
		return nil, err
	}

	var lastApplied *Migration
	if len(appliedMigrations) > 0 {
		lastApplied = &appliedMigrations[len(appliedMigrations)-1]
	}

	status := map[string]interface{}{
		"applied_count":   len(appliedMigrations),
		"pending_count":   len(pendingMigrations),
		"available_count": len(availableMigrations),
		"last_applied":    lastApplied,
		"is_up_to_date":   len(pendingMigrations) == 0,
		"migrations_table": mm.migrationsTable,
		"migrations_path":  mm.migrationsPath,
	}

	return status, nil
}

// ValidateMigrations validates all migrations for consistency
func (mm *MigrationManager) ValidateMigrations(ctx context.Context) error {
	migrations, err := mm.loadMigrationsFromFiles()
	if err != nil {
		return err
	}

	// Check for version conflicts
	versionMap := make(map[string]string)
	for _, migration := range migrations {
		if existing, exists := versionMap[migration.Version]; exists {
			return fmt.Errorf("duplicate version %s found in migrations: %s and %s", 
				migration.Version, existing, migration.Name)
		}
		versionMap[migration.Version] = migration.Name
	}

	// Check version ordering
	for i := 1; i < len(migrations); i++ {
		prev := migrations[i-1]
		curr := migrations[i]
		
		if curr.Version <= prev.Version {
			return fmt.Errorf("migration version %s should be greater than %s", 
				curr.Version, prev.Version)
		}
	}

	mm.logger.Info("Migration validation completed successfully", 
		zap.Int("total_migrations", len(migrations)))

	return nil
}

// RepairMigrations repairs inconsistent migration state
func (mm *MigrationManager) RepairMigrations(ctx context.Context, dryRun bool) error {
	appliedMigrations, err := mm.GetAppliedMigrations(ctx)
	if err != nil {
		return err
	}

	availableMigrations, err := mm.loadMigrationsFromFiles()
	if err != nil {
		return err
	}

	// Create map of available migrations
	availableMap := make(map[string]Migration)
	for _, migration := range availableMigrations {
		availableMap[migration.Version] = migration
	}

	repairActions := []string{}

	// Check for applied migrations that no longer exist
	for _, applied := range appliedMigrations {
		if _, exists := availableMap[applied.Version]; !exists {
			action := fmt.Sprintf("Remove orphaned migration record: %s", applied.Version)
			repairActions = append(repairActions, action)
			
			if !dryRun {
				query := fmt.Sprintf("DELETE FROM %s WHERE version = $1", mm.migrationsTable)
				_, err := mm.db.DB.ExecContext(ctx, query, applied.Version)
				if err != nil {
					return fmt.Errorf("failed to remove orphaned migration: %w", err)
				}
			}
		}
	}

	if dryRun {
		for _, action := range repairActions {
			mm.logger.Info("DRY RUN: Would repair", zap.String("action", action))
		}
	}

	if len(repairActions) == 0 {
		mm.logger.Info("No migration repairs needed")
	} else {
		mm.logger.Info("Migration repair completed", zap.Int("actions", len(repairActions)))
	}

	return nil
}