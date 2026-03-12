package testutils

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/mongodb"
	redisContainer "github.com/testcontainers/testcontainers-go/modules/redis"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.uber.org/zap"

	"kb-clinical-context/internal/cache"
	"kb-clinical-context/internal/config"
	"kb-clinical-context/internal/database"
)

// TestContainer holds test container instances
type TestContainer struct {
	MongoContainer testcontainers.Container
	RedisContainer testcontainers.Container
	MongoDB        *database.Database
	RedisClient    *cache.CacheClient
	Config         *config.Config
	Logger         *zap.Logger
}

// SetupTestContainers initializes MongoDB and Redis containers for integration testing
func SetupTestContainers(t *testing.T) (*TestContainer, error) {
	ctx := context.Background()

	// Create logger for testing
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("failed to create logger: %w", err)
	}

	// Setup MongoDB container
	mongoContainer, err := mongodb.RunContainer(ctx,
		testcontainers.WithImage("mongo:6.0"),
		mongodb.WithUsername("testuser"),
		mongodb.WithPassword("testpass"),
		mongodb.WithDatabase("kb_clinical_context_test"),
		testcontainers.WithWaitStrategy(
			testcontainers.NewLogStrategy("Waiting for connections").
				WithStartupTimeout(60*time.Second).
				WithPollInterval(1*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create MongoDB container: %w", err)
	}

	// Get MongoDB connection details
	mongoEndpoint, err := mongoContainer.ConnectionString(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get MongoDB connection string: %w", err)
	}

	// Setup Redis container
	redisContainerInstance, err := redisContainer.RunContainer(ctx,
		testcontainers.WithImage("redis:7-alpine"),
		redisContainer.WithConfigFile(""),
		testcontainers.WithWaitStrategy(
			testcontainers.NewLogStrategy("Ready to accept connections").
				WithStartupTimeout(30*time.Second).
				WithPollInterval(1*time.Second),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create Redis container: %w", err)
	}

	// Get Redis connection details
	redisHost, err := redisContainerInstance.Host(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis host: %w", err)
	}

	redisPort, err := redisContainerInstance.MappedPort(ctx, "6379")
	if err != nil {
		return nil, fmt.Errorf("failed to get Redis port: %w", err)
	}

	// Create test configuration
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port:        "8082",
			Environment: "test",
		},
		MongoDB: config.MongoDBConfig{
			URI:            mongoEndpoint,
			Database:       "kb_clinical_context_test",
			MaxPoolSize:    10,
			ConnectTimeout: 10 * time.Second,
		},
		Redis: config.RedisConfig{
			Address:     fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
			Password:    "",
			Database:    0,
			MaxRetries:  3,
			PoolSize:    10,
			PoolTimeout: 30 * time.Second,
		},
		CEL: config.CELConfig{
			MaxEvaluationTime:   5 * time.Second,
			MaxExpressionLength: 10000,
			EnableSafetyMode:    true,
		},
		Phenotype: config.PhenotypeConfig{
			Directory:           "../phenotypes",
			ReloadInterval:      5 * time.Minute,
			ValidationEnabled:   true,
			ConfidenceThreshold: 0.7,
		},
	}

	// Initialize MongoDB connection
	dbClient, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoEndpoint))
	if err != nil {
		mongoContainer.Terminate(ctx)
		redisContainerInstance.Terminate(ctx)
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test MongoDB connection
	if err := dbClient.Ping(ctx, nil); err != nil {
		mongoContainer.Terminate(ctx)
		redisContainerInstance.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	mongoDB := &database.Database{
		Client: dbClient,
		DB:     dbClient.Database("kb_clinical_context_test"),
	}

	// Initialize Redis client
	redisClient := redis.NewClient(&redis.Options{
		Addr:        fmt.Sprintf("%s:%s", redisHost, redisPort.Port()),
		Password:    "",
		DB:          0,
		MaxRetries:  3,
		PoolSize:    10,
		PoolTimeout: 30 * time.Second,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		mongoContainer.Terminate(ctx)
		redisContainerInstance.Terminate(ctx)
		return nil, fmt.Errorf("failed to ping Redis: %w", err)
	}

	cacheClient := &cache.CacheClient{
		Client: redisClient,
		Logger: logger,
	}

	return &TestContainer{
		MongoContainer: mongoContainer,
		RedisContainer: redisContainerInstance,
		MongoDB:        mongoDB,
		RedisClient:    cacheClient,
		Config:         cfg,
		Logger:         logger,
	}, nil
}

// Cleanup terminates all test containers
func (tc *TestContainer) Cleanup() {
	ctx := context.Background()
	
	if tc.MongoDB != nil && tc.MongoDB.Client != nil {
		tc.MongoDB.Client.Disconnect(ctx)
	}
	
	if tc.RedisClient != nil && tc.RedisClient.Client != nil {
		tc.RedisClient.Client.Close()
	}
	
	if tc.MongoContainer != nil {
		tc.MongoContainer.Terminate(ctx)
	}
	
	if tc.RedisContainer != nil {
		tc.RedisContainer.Terminate(ctx)
	}
}

// SeedTestData populates test containers with sample data
func (tc *TestContainer) SeedTestData(t *testing.T) error {
	ctx := context.Background()
	
	// Seed MongoDB with phenotype definitions
	phenotypeCollection := tc.MongoDB.DB.Collection("phenotype_definitions")
	
	samplePhenotypes := []interface{}{
		map[string]interface{}{
			"phenotype_id": "hypertension_stage_1",
			"name":        "Hypertension Stage 1",
			"version":     "1.0.0",
			"description": "Stage 1 hypertension based on ACC/AHA guidelines",
			"status":      "active",
			"criteria": map[string]interface{}{
				"required_conditions": []map[string]interface{}{},
				"required_labs": []map[string]interface{}{
					{
						"loinc_code":  "8480-6",
						"operator":    ">=",
						"value":       130.0,
						"unit":        "mmHg",
						"time_window": "30d",
					},
				},
				"required_medications": []map[string]interface{}{},
				"exclusion_criteria":   []string{},
			},
		},
		map[string]interface{}{
			"phenotype_id": "diabetes_uncontrolled",
			"name":        "Uncontrolled Diabetes",
			"version":     "1.0.0",
			"description": "Type 2 diabetes with poor glycemic control",
			"status":      "active",
			"criteria": map[string]interface{}{
				"required_conditions": []map[string]interface{}{
					{
						"type":            "diagnosis",
						"codes":           []string{"E11.9", "E11.65", "E11.69"},
						"time_window":     "",
						"min_occurrences": 1,
					},
				},
				"required_labs": []map[string]interface{}{
					{
						"loinc_code":  "4548-4",
						"operator":    ">",
						"value":       7.0,
						"unit":        "%",
						"time_window": "90d",
					},
				},
				"required_medications": []map[string]interface{}{},
				"exclusion_criteria":   []string{},
			},
		},
		map[string]interface{}{
			"phenotype_id": "ckd_stage_3",
			"name":        "Chronic Kidney Disease Stage 3",
			"version":     "1.0.0",
			"description": "Moderate decrease in GFR (30-59 mL/min/1.73m²)",
			"status":      "active",
			"criteria": map[string]interface{}{
				"required_conditions": []map[string]interface{}{},
				"required_labs": []map[string]interface{}{
					{
						"loinc_code":  "33914-3",
						"operator":    ">=",
						"value":       30.0,
						"unit":        "mL/min/1.73m2",
						"time_window": "60d",
					},
					{
						"loinc_code":  "33914-3",
						"operator":    "<",
						"value":       60.0,
						"unit":        "mL/min/1.73m2",
						"time_window": "60d",
					},
				},
				"required_medications": []map[string]interface{}{},
				"exclusion_criteria":   []string{},
			},
		},
	}
	
	_, err := phenotypeCollection.InsertMany(ctx, samplePhenotypes)
	if err != nil {
		return fmt.Errorf("failed to seed phenotype definitions: %w", err)
	}
	
	t.Logf("Seeded %d phenotype definitions", len(samplePhenotypes))
	
	return nil
}

// ClearTestData removes all test data from containers
func (tc *TestContainer) ClearTestData() error {
	ctx := context.Background()
	
	// Clear MongoDB collections
	collections := []string{
		"phenotype_definitions",
		"patient_contexts",
		"detection_results",
	}
	
	for _, collName := range collections {
		collection := tc.MongoDB.DB.Collection(collName)
		if _, err := collection.DeleteMany(ctx, map[string]interface{}{}); err != nil {
			return fmt.Errorf("failed to clear collection %s: %w", collName, err)
		}
	}
	
	// Clear Redis data
	if err := tc.RedisClient.Client.FlushDB(ctx).Err(); err != nil {
		return fmt.Errorf("failed to clear Redis: %w", err)
	}
	
	return nil
}

// WaitForContainers waits for all containers to be ready
func (tc *TestContainer) WaitForContainers(timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Wait for MongoDB
	mongoReady := make(chan bool)
	go func() {
		for {
			if err := tc.MongoDB.Client.Ping(ctx, nil); err == nil {
				mongoReady <- true
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}()

	// Wait for Redis
	redisReady := make(chan bool)
	go func() {
		for {
			if err := tc.RedisClient.Client.Ping(ctx).Err(); err == nil {
				redisReady <- true
				return
			}
			select {
			case <-ctx.Done():
				return
			case <-time.After(1 * time.Second):
				continue
			}
		}
	}()

	// Wait for both services
	mongoReadyFlag := false
	redisReadyFlag := false

	for !mongoReadyFlag || !redisReadyFlag {
		select {
		case <-mongoReady:
			mongoReadyFlag = true
		case <-redisReady:
			redisReadyFlag = true
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for containers to be ready")
		}
	}

	return nil
}