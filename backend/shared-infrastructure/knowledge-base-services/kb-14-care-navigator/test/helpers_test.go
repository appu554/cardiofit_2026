// Package test provides shared test helpers for KB-14 Care Navigator
package test

import (
	"os"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"kb-14-care-navigator/internal/clients"
	"kb-14-care-navigator/internal/config"
	"kb-14-care-navigator/internal/database"
	"kb-14-care-navigator/internal/services"
)

// =============================================================================
// ENVIRONMENT HELPERS
// =============================================================================

// getEnvOrDefault returns environment variable value or default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// =============================================================================
// TEST CONFIG HELPERS
// =============================================================================

// TestConfig holds test environment configuration
type TestConfig struct {
	DatabaseURL string
	RedisURL    string
	KB3URL      string
	KB9URL      string
	KB12URL     string
}

// LoadTestConfig loads test configuration from environment
func LoadTestConfig() *TestConfig {
	return &TestConfig{
		DatabaseURL: getEnvOrDefault("DATABASE_URL", "postgres://kb14user:kb14password@localhost:5438/kb_care_navigator_test?sslmode=disable"),
		RedisURL:    getEnvOrDefault("REDIS_URL", "redis://localhost:6386/1"),
		KB3URL:      getEnvOrDefault("KB3_URL", "http://localhost:8087"),
		KB9URL:      getEnvOrDefault("KB9_URL", "http://localhost:8089"),
		KB12URL:     getEnvOrDefault("KB12_URL", "http://localhost:8090"),
	}
}

// =============================================================================
// DATABASE HELPERS
// =============================================================================

// SetupTestDB creates a test database connection
func SetupTestDB(cfg *TestConfig, log *logrus.Entry) (*gorm.DB, *database.Database, error) {
	db, err := gorm.Open(postgres.Open(cfg.DatabaseURL), &gorm.Config{})
	if err != nil {
		return nil, nil, err
	}
	dbWrapper := &database.Database{DB: db}
	return db, dbWrapper, nil
}

// SafeMigrate handles GORM AutoMigrate while respecting PostgreSQL views
// The migrations create views (v_active_escalations, v_escalation_stats) that depend on the
// escalations table. GORM AutoMigrate cannot modify columns used by views.
// This function checks if tables already exist (from SQL migrations) and skips AutoMigrate if so.
func SafeMigrate(db *gorm.DB, models ...interface{}) error {
	// Check if core tables already exist (indicating migrations were applied)
	var tableExists bool
	err := db.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_schema = 'public'
			AND table_name = 'tasks'
		)
	`).Scan(&tableExists).Error
	if err != nil {
		return err
	}

	if tableExists {
		// Tables exist from migrations - skip AutoMigrate to avoid view conflicts
		// SQL migrations handle schema properly
		return nil
	}

	// Tables don't exist - need to drop views, migrate, then recreate views
	// First, drop dependent views
	viewDropStatements := []string{
		"DROP VIEW IF EXISTS v_active_escalations CASCADE",
		"DROP VIEW IF EXISTS v_escalation_stats CASCADE",
	}
	for _, stmt := range viewDropStatements {
		if err := db.Exec(stmt).Error; err != nil {
			// Ignore errors if views don't exist
		}
	}

	// Run AutoMigrate
	if err := db.AutoMigrate(models...); err != nil {
		return err
	}

	// Recreate views
	viewCreateStatements := []string{
		`CREATE OR REPLACE VIEW v_active_escalations AS
		SELECT
			e.id AS escalation_id,
			e.level,
			e.reason,
			e.sla_elapsed_percent,
			e.time_overdue AS time_overdue_minutes,
			e.created_at AS escalated_at,
			e.acknowledged,
			t.id AS task_id,
			t.task_id AS task_number,
			t.type AS task_type,
			t.priority AS task_priority,
			t.title AS task_title,
			t.patient_id,
			t.assigned_to,
			t.team_id,
			t.due_date
		FROM escalations e
		JOIN tasks t ON e.task_id = t.id
		WHERE e.acknowledged = false
		  AND t.status NOT IN ('COMPLETED', 'VERIFIED', 'CANCELLED')
		ORDER BY e.level DESC, e.created_at ASC`,
		`CREATE OR REPLACE VIEW v_escalation_stats AS
		SELECT
			DATE_TRUNC('day', created_at) AS date,
			level,
			COUNT(*) AS total_escalations,
			COUNT(*) FILTER (WHERE acknowledged = true) AS acknowledged_count,
			AVG(EXTRACT(EPOCH FROM (acknowledged_at - created_at)) / 60)
				FILTER (WHERE acknowledged = true) AS avg_response_minutes
		FROM escalations
		GROUP BY DATE_TRUNC('day', created_at), level
		ORDER BY date DESC, level`,
	}
	for _, stmt := range viewCreateStatements {
		if err := db.Exec(stmt).Error; err != nil {
			return err
		}
	}

	return nil
}

// =============================================================================
// SERVICE FACTORY HELPERS
// =============================================================================

// TestServices holds initialized test services
type TestServices struct {
	TaskRepo          *database.TaskRepository
	TeamRepo          *database.TeamRepository
	EscalationRepo    *database.EscalationRepository
	AuditRepo         *database.AuditRepository
	GovernanceRepo    *database.GovernanceRepository
	ReasonCodeRepo    *database.ReasonCodeRepository
	IntelligenceRepo  *database.IntelligenceRepository
	TaskService       *services.TaskService
	GovernanceService *services.GovernanceService
	NotificationSvc   *services.NotificationService
	AssignmentEngine  *services.AssignmentEngine
	EscalationEngine  *services.EscalationEngine
	WorklistService   *services.WorklistService
	TaskFactory       *services.TaskFactory
	KB3Client         *clients.KB3Client
	KB9Client         *clients.KB9Client
	KB12Client        *clients.KB12Client
}

// SetupTestServices initializes all test services
func SetupTestServices(db *gorm.DB, dbWrapper *database.Database, cfg *TestConfig, log *logrus.Entry) *TestServices {
	svc := &TestServices{}

	// Initialize repositories
	svc.TaskRepo = database.NewTaskRepository(dbWrapper, log)
	svc.TeamRepo = database.NewTeamRepository(dbWrapper, log)
	svc.EscalationRepo = database.NewEscalationRepository(dbWrapper, log)
	svc.AuditRepo = database.NewAuditRepository(db)
	svc.GovernanceRepo = database.NewGovernanceRepository(db)
	svc.ReasonCodeRepo = database.NewReasonCodeRepository(db)
	svc.IntelligenceRepo = database.NewIntelligenceRepository(db)

	// Initialize KB clients
	kbClientConfig := config.KBClientConfig{
		URL:     cfg.KB3URL,
		Timeout: 30 * time.Second,
		Enabled: true,
	}
	svc.KB3Client = clients.NewKB3Client(kbClientConfig)
	kbClientConfig.URL = cfg.KB9URL
	svc.KB9Client = clients.NewKB9Client(kbClientConfig)
	kbClientConfig.URL = cfg.KB12URL
	svc.KB12Client = clients.NewKB12Client(kbClientConfig)

	// Initialize services
	svc.NotificationSvc = services.NewNotificationService(log)
	svc.GovernanceService = services.NewGovernanceService(
		svc.AuditRepo,
		svc.GovernanceRepo,
		svc.ReasonCodeRepo,
		svc.IntelligenceRepo,
		log,
	)
	svc.TaskService = services.NewTaskService(
		svc.TaskRepo,
		svc.TeamRepo,
		svc.EscalationRepo,
		svc.GovernanceService,
		log,
	)
	svc.TaskFactory = services.NewTaskFactory(
		svc.TaskService,
		svc.KB3Client,
		svc.KB9Client,
		svc.KB12Client,
		log,
	)
	svc.AssignmentEngine = services.NewAssignmentEngine(svc.TaskRepo, svc.TeamRepo, log)
	svc.EscalationEngine = services.NewEscalationEngine(
		svc.TaskRepo,
		svc.TeamRepo,
		svc.EscalationRepo,
		svc.NotificationSvc,
		log,
	)
	svc.WorklistService = services.NewWorklistService(svc.TaskRepo, svc.TeamRepo, log)

	return svc
}

// =============================================================================
// UUID HELPERS
// =============================================================================

// ParseUserID parses a string UserID to uuid.UUID (panics if invalid, for test use only)
func ParseUserID(userID string) uuid.UUID {
	id, err := uuid.Parse(userID)
	if err != nil {
		panic("invalid UserID: " + userID)
	}
	return id
}

// UUIDPtr returns a pointer to a UUID
func UUIDPtr(id uuid.UUID) *uuid.UUID {
	return &id
}

// NewUUIDPtr creates a new UUID and returns a pointer
func NewUUIDPtr() *uuid.UUID {
	id := uuid.New()
	return &id
}

// =============================================================================
// TIME HELPERS
// =============================================================================

// TimePtr returns a pointer to a time.Time
func TimePtr(t time.Time) *time.Time {
	return &t
}

// NowPtr returns a pointer to current time
func NowPtr() *time.Time {
	now := time.Now()
	return &now
}

// =============================================================================
// LOGGER HELPERS
// =============================================================================

// NewTestLogger creates a test logger
func NewTestLogger(testName string) *logrus.Entry {
	logger := logrus.New()
	logger.SetLevel(logrus.DebugLevel)
	return logrus.NewEntry(logger).WithField("test", testName)
}
