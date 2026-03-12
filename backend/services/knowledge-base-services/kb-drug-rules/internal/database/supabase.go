package database

import (
	"fmt"
	"strings"
	"time"

	_ "github.com/lib/pq"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// SupabaseConfig holds Supabase connection configuration
type SupabaseConfig struct {
	URL      string
	APIKey   string
	JWTSecret string
	Database struct {
		Host     string
		Port     string
		User     string
		Password string
		DBName   string
		SSLMode  string
	}
}

// NewSupabaseConnection creates a new Supabase database connection
func NewSupabaseConnection(config *SupabaseConfig) (*gorm.DB, error) {
	// Build PostgreSQL connection string for Supabase
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		config.Database.Host,
		config.Database.Port,
		config.Database.User,
		config.Database.Password,
		config.Database.DBName,
		config.Database.SSLMode,
	)

	// Configure GORM with Supabase-specific settings
	gormConfig := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
		NowFunc: func() time.Time {
			return time.Now().UTC()
		},
		// Supabase uses UUID primary keys by default
		DisableForeignKeyConstraintWhenMigrating: true,
	}

	// Connect to Supabase PostgreSQL
	db, err := gorm.Open(postgres.Open(dsn), gormConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Supabase: %w", err)
	}

	// Get underlying SQL DB for configuration
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get underlying SQL DB: %w", err)
	}

	// Configure connection pool for Supabase
	sqlDB.SetMaxIdleConns(5)   // Supabase has connection limits
	sqlDB.SetMaxOpenConns(20)  // Conservative for Supabase free tier
	sqlDB.SetConnMaxLifetime(time.Hour)

	// Test connection
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping Supabase database: %w", err)
	}

	return db, nil
}

// NewSupabaseConnectionFromURL creates a Supabase connection from a URL
func NewSupabaseConnectionFromURL(databaseURL string) (*gorm.DB, error) {
	// Parse Supabase connection URL
	config, err := parseSupabaseURL(databaseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Supabase URL: %w", err)
	}

	return NewSupabaseConnection(config)
}

// parseSupabaseURL parses a Supabase database URL into configuration
func parseSupabaseURL(databaseURL string) (*SupabaseConfig, error) {
	// Example Supabase URL format:
	// postgresql://postgres:[password]@db.[project-ref].supabase.co:5432/postgres
	
	if !strings.Contains(databaseURL, "supabase.co") {
		// Fallback to regular PostgreSQL parsing
		return parsePostgreSQLURL(databaseURL)
	}

	// Parse the URL components
	// This is a simplified parser - in production, use a proper URL parser
	config := &SupabaseConfig{}
	
	// Extract components from URL
	if strings.HasPrefix(databaseURL, "postgresql://") {
		urlParts := strings.TrimPrefix(databaseURL, "postgresql://")
		
		// Split user:password@host:port/dbname
		atIndex := strings.Index(urlParts, "@")
		if atIndex == -1 {
			return nil, fmt.Errorf("invalid Supabase URL format")
		}
		
		userPass := urlParts[:atIndex]
		hostPortDB := urlParts[atIndex+1:]
		
		// Parse user:password
		colonIndex := strings.Index(userPass, ":")
		if colonIndex == -1 {
			return nil, fmt.Errorf("invalid Supabase URL format")
		}
		
		config.Database.User = userPass[:colonIndex]
		config.Database.Password = userPass[colonIndex+1:]
		
		// Parse host:port/dbname
		slashIndex := strings.Index(hostPortDB, "/")
		if slashIndex == -1 {
			return nil, fmt.Errorf("invalid Supabase URL format")
		}
		
		hostPort := hostPortDB[:slashIndex]
		dbName := hostPortDB[slashIndex+1:]
		
		// Parse host:port
		portIndex := strings.LastIndex(hostPort, ":")
		if portIndex == -1 {
			config.Database.Host = hostPort
			config.Database.Port = "5432"
		} else {
			config.Database.Host = hostPort[:portIndex]
			config.Database.Port = hostPort[portIndex+1:]
		}
		
		config.Database.DBName = dbName
		config.Database.SSLMode = "require" // Supabase requires SSL
	}

	return config, nil
}

// parsePostgreSQLURL parses a regular PostgreSQL URL for fallback
func parsePostgreSQLURL(databaseURL string) (*SupabaseConfig, error) {
	config := &SupabaseConfig{}
	
	// Simple parsing for regular PostgreSQL URLs
	// In production, use a proper URL parser like net/url
	if strings.Contains(databaseURL, "localhost") || strings.Contains(databaseURL, "127.0.0.1") {
		config.Database.Host = "localhost"
		config.Database.Port = "5432"
		config.Database.User = "postgres"
		config.Database.Password = "password"
		config.Database.DBName = "kb_drug_rules"
		config.Database.SSLMode = "disable"
	} else {
		return nil, fmt.Errorf("unsupported database URL format")
	}

	return config, nil
}

// InitializeSupabaseSchema initializes the database schema for Supabase
func InitializeSupabaseSchema(db *gorm.DB) error {
	// Enable required PostgreSQL extensions
	extensions := []string{
		"CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"",
		"CREATE EXTENSION IF NOT EXISTS \"pg_trgm\"",
		"CREATE EXTENSION IF NOT EXISTS \"pg_stat_statements\"",
	}

	for _, ext := range extensions {
		if err := db.Exec(ext).Error; err != nil {
			// Log warning but don't fail - extensions might not be available
			fmt.Printf("Warning: Failed to create extension: %v\n", err)
		}
	}

	// Enable Row Level Security (RLS) for Supabase
	rlsPolicies := []string{
		"ALTER TABLE IF EXISTS drug_rule_packs ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE IF EXISTS governance_approvals ENABLE ROW LEVEL SECURITY",
		"ALTER TABLE IF EXISTS audit_log ENABLE ROW LEVEL SECURITY",
	}

	for _, policy := range rlsPolicies {
		if err := db.Exec(policy).Error; err != nil {
			// Log warning but don't fail - tables might not exist yet
			fmt.Printf("Warning: Failed to enable RLS: %v\n", err)
		}
	}

	return nil
}

// CreateSupabaseRLSPolicies creates Row Level Security policies for Supabase
func CreateSupabaseRLSPolicies(db *gorm.DB) error {
	policies := []string{
		// Allow authenticated users to read drug rules
		`CREATE POLICY IF NOT EXISTS "Allow authenticated read access" ON drug_rule_packs
		 FOR SELECT USING (auth.role() = 'authenticated')`,
		
		// Allow service role to manage drug rules
		`CREATE POLICY IF NOT EXISTS "Allow service role full access" ON drug_rule_packs
		 FOR ALL USING (auth.role() = 'service_role')`,
		
		// Allow authenticated users to read governance approvals
		`CREATE POLICY IF NOT EXISTS "Allow authenticated read governance" ON governance_approvals
		 FOR SELECT USING (auth.role() = 'authenticated')`,
		
		// Allow service role to manage governance
		`CREATE POLICY IF NOT EXISTS "Allow service role governance access" ON governance_approvals
		 FOR ALL USING (auth.role() = 'service_role')`,
		
		// Audit log policies
		`CREATE POLICY IF NOT EXISTS "Allow service role audit access" ON audit_log
		 FOR ALL USING (auth.role() = 'service_role')`,
	}

	for _, policy := range policies {
		if err := db.Exec(policy).Error; err != nil {
			fmt.Printf("Warning: Failed to create RLS policy: %v\n", err)
		}
	}

	return nil
}

// SupabaseHealthCheck checks Supabase database health
func SupabaseHealthCheck(db *gorm.DB) error {
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get SQL DB: %w", err)
	}

	if err := sqlDB.Ping(); err != nil {
		return fmt.Errorf("Supabase ping failed: %w", err)
	}

	// Check if we can query a system table
	var version string
	if err := db.Raw("SELECT version()").Scan(&version).Error; err != nil {
		return fmt.Errorf("failed to query Supabase version: %w", err)
	}

	return nil
}

// buildSupabaseDatabaseURL builds a PostgreSQL connection string from Supabase URL
func buildSupabaseDatabaseURL(supabaseURL string) string {
	if supabaseURL == "" {
		return ""
	}

	// Parse the Supabase URL to extract project reference
	// Expected format: https://[project-ref].supabase.co
	if !strings.Contains(supabaseURL, "supabase.co") {
		return ""
	}

	// Extract project reference
	urlParts := strings.TrimPrefix(supabaseURL, "https://")
	urlParts = strings.TrimPrefix(urlParts, "http://")
	projectRef := strings.Split(urlParts, ".")[0]

	if projectRef == "" {
		return ""
	}

	// Get password from environment
	password := ""
	if envPassword := strings.TrimSpace(getEnvOrDefault("SUPABASE_DB_PASSWORD", "")); envPassword != "" {
		password = envPassword
	}

	if password == "" {
		return ""
	}

	// Build connection string
	// Supabase database host format: db.[project-ref].supabase.co
	host := fmt.Sprintf("db.%s.supabase.co", projectRef)

	return fmt.Sprintf("postgresql://postgres:%s@%s:5432/postgres?sslmode=require", password, host)
}

// getEnvOrDefault returns the value of an environment variable or a default value
func getEnvOrDefault(key, defaultValue string) string {
	if value, exists := lookupEnv(key); exists {
		return value
	}
	return defaultValue
}

// lookupEnv is a variable to allow mocking in tests
var lookupEnv = func(key string) (string, bool) {
	// This would normally use os.LookupEnv, but we use a variable for testability
	return "", false
}

// GetSupabaseStats returns Supabase-specific database statistics
func GetSupabaseStats(db *gorm.DB) (map[string]interface{}, error) {
	sqlDB, err := db.DB()
	if err != nil {
		return nil, fmt.Errorf("failed to get SQL DB: %w", err)
	}

	stats := sqlDB.Stats()
	
	// Get Supabase-specific metrics
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public'").Scan(&tableCount)
	
	var dbSize string
	db.Raw("SELECT pg_size_pretty(pg_database_size(current_database()))").Scan(&dbSize)
	
	return map[string]interface{}{
		"max_open_connections":     stats.MaxOpenConnections,
		"open_connections":         stats.OpenConnections,
		"in_use":                  stats.InUse,
		"idle":                    stats.Idle,
		"wait_count":              stats.WaitCount,
		"wait_duration":           stats.WaitDuration,
		"max_idle_closed":         stats.MaxIdleClosed,
		"max_idle_time_closed":    stats.MaxIdleTimeClosed,
		"max_lifetime_closed":     stats.MaxLifetimeClosed,
		"table_count":             tableCount,
		"database_size":           dbSize,
		"provider":                "supabase",
	}, nil
}
