package testsetup

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"runtime"
	"testing"
	"time"

	"go.uber.org/zap"
	
	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/repositories"
	"medication-service-v2/internal/infrastructure/database"
	"medication-service-v2/internal/infrastructure/redis"
	"medication-service-v2/internal/infrastructure/monitoring"
	"medication-service-v2/internal/interfaces/grpc/auth"
)

// Test database configuration
const (
	TestDBHost     = "localhost"
	TestDBPort     = "5434"
	TestDBUser     = "test_user"
	TestDBPassword = "test_password"
	TestDBName     = "medication_service_test"
	TestDBSSLMode  = "disable"
)

// Test Redis configuration
const (
	TestRedisHost     = "localhost"
	TestRedisPort     = "6381"
	TestRedisPassword = ""
	TestRedisDB       = 0
)

// SetupTestDatabase creates and configures a test database
func SetupTestDatabase(t *testing.T) *database.Client {
	if testing.Short() {
		t.Skip("Skipping database tests in short mode")
	}
	
	// Check if test database is available
	if !isDatabaseAvailable() {
		t.Skip("Test database not available")
	}
	
	// Create database connection
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		TestDBHost, TestDBPort, TestDBUser, TestDBPassword, TestDBName, TestDBSSLMode)
	
	db, err := database.NewClient(dsn)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	
	// Run migrations
	err = runTestMigrations(db)
	if err != nil {
		t.Fatalf("Failed to run test migrations: %v", err)
	}
	
	// Register cleanup
	t.Cleanup(func() {
		CleanupTestDatabase(t, db)
	})
	
	return db
}

// SetupOptimizedTestDatabase creates an optimized test database for performance tests
func SetupOptimizedTestDatabase(t *testing.T) *database.Client {
	// Use in-memory database for performance tests
	db := SetupTestDatabase(t)
	
	// Configure for performance
	_, err := db.Exec(context.Background(), "SET synchronous_commit = OFF")
	if err != nil {
		t.Logf("Warning: Could not disable synchronous commit: %v", err)
	}
	
	_, err = db.Exec(context.Background(), "SET checkpoint_segments = 32")
	if err != nil {
		t.Logf("Warning: Could not configure checkpoint segments: %v", err)
	}
	
	return db
}

// SetupTestRedis creates and configures a test Redis client
func SetupTestRedis(t *testing.T) *redis.Client {
	if testing.Short() {
		t.Skip("Skipping Redis tests in short mode")
	}
	
	// Check if test Redis is available
	if !isRedisAvailable() {
		t.Skip("Test Redis not available")
	}
	
	config := &redis.Config{
		Host:     TestRedisHost,
		Port:     TestRedisPort,
		Password: TestRedisPassword,
		DB:       TestRedisDB,
		PoolSize: 10,
	}
	
	client, err := redis.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to connect to test Redis: %v", err)
	}
	
	// Register cleanup
	t.Cleanup(func() {
		CleanupTestRedis(t, client)
	})
	
	return client
}

// SetupOptimizedTestRedis creates an optimized Redis client for performance tests
func SetupOptimizedTestRedis(t *testing.T) *redis.Client {
	config := &redis.Config{
		Host:     TestRedisHost,
		Port:     TestRedisPort,
		Password: TestRedisPassword,
		DB:       TestRedisDB,
		PoolSize: 100, // Larger pool for performance testing
		MaxRetries: 3,
		MinIdleConns: 10,
	}
	
	client, err := redis.NewClient(config)
	if err != nil {
		t.Fatalf("Failed to connect to optimized test Redis: %v", err)
	}
	
	t.Cleanup(func() {
		CleanupTestRedis(t, client)
	})
	
	return client
}

// SetupMedicationRepository creates a test medication repository
func SetupMedicationRepository(db *database.Client) repositories.MedicationRepository {
	return database.NewMedicationRepository(db)
}

// SetupOptimizedMedicationRepository creates an optimized medication repository
func SetupOptimizedMedicationRepository(db *database.Client) repositories.MedicationRepository {
	return database.NewMedicationRepository(db)
}

// SetupRecipeRepository creates a test recipe repository
func SetupRecipeRepository(db *database.Client) repositories.RecipeRepository {
	return database.NewRecipeRepository(db)
}

// SetupOptimizedRecipeRepository creates an optimized recipe repository
func SetupOptimizedRecipeRepository(db *database.Client) repositories.RecipeRepository {
	return database.NewRecipeRepository(db)
}

// SetupRustEngine creates a test Rust clinical engine client
func SetupRustEngine(t *testing.T) services.RustClinicalEngineClient {
	// For testing, use mock or test implementation
	if os.Getenv("USE_REAL_RUST_ENGINE") == "true" {
		return setupRealRustEngine(t)
	}
	return setupMockRustEngine(t)
}

// SetupOptimizedRustEngine creates an optimized Rust engine for performance tests
func SetupOptimizedRustEngine(t *testing.T) services.RustClinicalEngineClient {
	return SetupRustEngine(t) // Same as regular for now
}

// SetupTestRustEngine creates a test-specific Rust engine
func SetupTestRustEngine(t *testing.T) services.RustClinicalEngineClient {
	return setupMockRustEngine(t)
}

// SetupApolloClient creates a test Apollo Federation client
func SetupApolloClient(t *testing.T) services.ApolloFederationClient {
	if os.Getenv("USE_REAL_APOLLO") == "true" {
		return setupRealApolloClient(t)
	}
	return setupMockApolloClient(t)
}

// SetupOptimizedApolloClient creates an optimized Apollo client for performance tests
func SetupOptimizedApolloClient(t *testing.T) services.ApolloFederationClient {
	return SetupApolloClient(t) // Same as regular for now
}

// SetupTestApolloClient creates a test-specific Apollo client
func SetupTestApolloClient(t *testing.T) services.ApolloFederationClient {
	return setupMockApolloClient(t)
}

// SetupContextGateway creates a test Context Gateway client
func SetupContextGateway(t *testing.T) services.ContextGatewayClient {
	if os.Getenv("USE_REAL_CONTEXT_GATEWAY") == "true" {
		return setupRealContextGateway(t)
	}
	return setupMockContextGateway(t)
}

// SetupOptimizedContextGateway creates an optimized Context Gateway for performance tests
func SetupOptimizedContextGateway(t *testing.T) services.ContextGatewayClient {
	return SetupContextGateway(t) // Same as regular for now
}

// SetupTestContextGateway creates a test-specific Context Gateway
func SetupTestContextGateway(t *testing.T) services.ContextGatewayClient {
	return setupMockContextGateway(t)
}

// SetupTestAuthService creates a test authentication service
func SetupTestAuthService(t *testing.T) *auth.Service {
	config := &auth.Config{
		JWTSecret:     "test-secret-key-for-testing",
		TokenExpiry:   1 * time.Hour,
		RefreshExpiry: 24 * time.Hour,
		Issuer:        "clinical-platform-test",
	}
	
	return auth.NewService(config)
}

// TestLogger returns a test-appropriate logger
func TestLogger() *zap.Logger {
	if testing.Verbose() {
		logger, _ := zap.NewDevelopment()
		return logger
	}
	return zap.NewNop()
}

// TestMetrics returns a test metrics instance
func TestMetrics() *monitoring.Metrics {
	return monitoring.NewMetrics()
}

// Cleanup functions

// CleanupTestDatabase cleans up test database resources
func CleanupTestDatabase(t *testing.T, db *database.Client) {
	if db != nil {
		CleanupTestData(t, db)
		db.Close()
	}
}

// CleanupTestData removes test data from database
func CleanupTestData(t *testing.T, db *database.Client) {
	ctx := context.Background()
	
	// Clean up tables in reverse dependency order
	tables := []string{
		"medication_proposals",
		"recipes",
		"snapshots",
		"audit_events",
	}
	
	for _, table := range tables {
		_, err := db.Exec(ctx, fmt.Sprintf("DELETE FROM %s", table))
		if err != nil {
			t.Logf("Warning: Could not clean up table %s: %v", table, err)
		}
	}
}

// CleanupTestRedis cleans up test Redis resources
func CleanupTestRedis(t *testing.T, client *redis.Client) {
	if client != nil {
		ctx := context.Background()
		err := client.FlushDB(ctx)
		if err != nil {
			t.Logf("Warning: Could not flush test Redis: %v", err)
		}
		client.Close()
	}
}

// Utility functions

// isDatabaseAvailable checks if test database is accessible
func isDatabaseAvailable() bool {
	dsn := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		TestDBHost, TestDBPort, TestDBUser, TestDBPassword, TestDBName, TestDBSSLMode)
	
	db, err := sql.Open("postgres", dsn)
	if err != nil {
		return false
	}
	defer db.Close()
	
	err = db.Ping()
	return err == nil
}

// isRedisAvailable checks if test Redis is accessible
func isRedisAvailable() bool {
	config := &redis.Config{
		Host:     TestRedisHost,
		Port:     TestRedisPort,
		Password: TestRedisPassword,
		DB:       TestRedisDB,
	}
	
	client, err := redis.NewClient(config)
	if err != nil {
		return false
	}
	defer client.Close()
	
	ctx := context.Background()
	_, err = client.Ping(ctx)
	return err == nil
}

// runTestMigrations runs database migrations for testing
func runTestMigrations(db *database.Client) error {
	migrations := []string{
		`CREATE TABLE IF NOT EXISTS recipes (
			id UUID PRIMARY KEY,
			protocol_id VARCHAR(255) NOT NULL,
			name VARCHAR(255) NOT NULL,
			version VARCHAR(50) NOT NULL,
			description TEXT,
			indication VARCHAR(500),
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_by VARCHAR(255) NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS medication_proposals (
			id UUID PRIMARY KEY,
			patient_id UUID NOT NULL,
			protocol_id VARCHAR(255) NOT NULL,
			indication VARCHAR(500) NOT NULL,
			status VARCHAR(50) NOT NULL,
			snapshot_id UUID,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			updated_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			created_by VARCHAR(255) NOT NULL,
			validated_by VARCHAR(255),
			validation_timestamp TIMESTAMP WITH TIME ZONE
		)`,
		`CREATE TABLE IF NOT EXISTS snapshots (
			id UUID PRIMARY KEY,
			patient_id UUID NOT NULL,
			recipe_id UUID,
			type VARCHAR(50) NOT NULL,
			status VARCHAR(50) NOT NULL,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
			expires_at TIMESTAMP WITH TIME ZONE
		)`,
		`CREATE TABLE IF NOT EXISTS audit_events (
			id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
			entity_type VARCHAR(100) NOT NULL,
			entity_id VARCHAR(255) NOT NULL,
			action VARCHAR(50) NOT NULL,
			user_id VARCHAR(255) NOT NULL,
			ip_address INET,
			user_agent TEXT,
			details JSONB,
			timestamp TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_proposals_patient_id ON medication_proposals(patient_id)`,
		`CREATE INDEX IF NOT EXISTS idx_proposals_status ON medication_proposals(status)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_entity ON audit_events(entity_type, entity_id)`,
		`CREATE INDEX IF NOT EXISTS idx_audit_events_timestamp ON audit_events(timestamp)`,
	}
	
	ctx := context.Background()
	for _, migration := range migrations {
		_, err := db.Exec(ctx, migration)
		if err != nil {
			return fmt.Errorf("migration failed: %w", err)
		}
	}
	
	return nil
}

// Mock service implementations for testing

func setupMockRustEngine(t *testing.T) services.RustClinicalEngineClient {
	return &MockRustEngine{t: t}
}

func setupMockApolloClient(t *testing.T) services.ApolloFederationClient {
	return &MockApolloClient{t: t}
}

func setupMockContextGateway(t *testing.T) services.ContextGatewayClient {
	return &MockContextGateway{t: t}
}

// Real service implementations (when external services are available)

func setupRealRustEngine(t *testing.T) services.RustClinicalEngineClient {
	// In real implementation, would connect to actual Rust service
	// For now, return mock
	return setupMockRustEngine(t)
}

func setupRealApolloClient(t *testing.T) services.ApolloFederationClient {
	// In real implementation, would connect to actual Apollo Federation
	// For now, return mock
	return setupMockApolloClient(t)
}

func setupRealContextGateway(t *testing.T) services.ContextGatewayClient {
	// In real implementation, would connect to actual Context Gateway
	// For now, return mock
	return setupMockContextGateway(t)
}

// Performance testing utilities

// GetMemoryUsageMB returns current memory usage in MB
func GetMemoryUsageMB() float64 {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)
	return float64(m.Alloc) / 1024 / 1024
}

// ForceGarbageCollection forces garbage collection
func ForceGarbageCollection() {
	runtime.GC()
	runtime.GC() // Call twice to ensure cleanup
}

// Mock implementations

type MockRustEngine struct {
	t *testing.T
}

func (m *MockRustEngine) CalculateDosage(ctx context.Context, request *services.RustCalculationRequest) (*services.RustCalculationResponse, error) {
	return &services.RustCalculationResponse{
		DoseMg:          1.4 * request.Parameters["bsa_m2"].(float64),
		ConfidenceScore: 0.95,
		Adjustments:     []string{},
		Warnings:        []string{},
	}, nil
}

func (m *MockRustEngine) ValidateSafety(ctx context.Context, request *services.RustSafetyRequest) (*services.RustSafetyResponse, error) {
	return &services.RustSafetyResponse{
		IsValid:     true,
		Violations:  []services.SafetyViolation{},
		Score:       0.98,
		Recommendations: []string{},
	}, nil
}

func (m *MockRustEngine) HealthCheck(ctx context.Context) error {
	return nil
}

type MockApolloClient struct {
	t *testing.T
}

func (m *MockApolloClient) QueryKnowledgeBase(ctx context.Context, query string, variables map[string]interface{}) (*services.GraphQLResponse, error) {
	return &services.GraphQLResponse{
		Data: map[string]interface{}{
			"drugInteractions": []interface{}{},
			"contraindications": []interface{}{},
		},
	}, nil
}

func (m *MockApolloClient) GetDrugInteractions(ctx context.Context, drugName string) (*services.DrugInteractionsResponse, error) {
	return &services.DrugInteractionsResponse{
		Interactions: []services.DrugInteraction{},
	}, nil
}

type MockContextGateway struct {
	t *testing.T
}

func (m *MockContextGateway) CreateSnapshot(ctx context.Context, request *services.ContextSnapshotRequest) (*services.ContextSnapshotResponse, error) {
	return &services.ContextSnapshotResponse{
		SnapshotID: request.PatientID.String() + "-snapshot",
		Status:     "active",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		Data: map[string]interface{}{
			"patient_id": request.PatientID,
			"timestamp":  time.Now(),
		},
	}, nil
}

func (m *MockContextGateway) GetSnapshot(ctx context.Context, snapshotID uuid.UUID) (*services.ContextSnapshotResponse, error) {
	return &services.ContextSnapshotResponse{
		SnapshotID: snapshotID.String(),
		Status:     "active",
		ExpiresAt:  time.Now().Add(1 * time.Hour),
		Data: map[string]interface{}{
			"cached": true,
		},
	}, nil
}