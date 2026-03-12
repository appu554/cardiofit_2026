package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"
)

// MongoConnection manages MongoDB connections for clinical context data
type MongoConnection struct {
	client   *mongo.Client
	database *mongo.Database
	config   MongoConfig
}

// MongoConfig holds MongoDB connection configuration
type MongoConfig struct {
	URI                string
	Database           string
	ConnectTimeout     time.Duration
	ServerTimeout      time.Duration
	MaxPoolSize        uint64
	MinPoolSize        uint64
	MaxConnIdleTime    time.Duration
	ApplicationName    string
}

// NewMongoConnection creates a new MongoDB connection
func NewMongoConnection(config MongoConfig) (*MongoConnection, error) {
	ctx, cancel := context.WithTimeout(context.Background(), config.ConnectTimeout)
	defer cancel()

	// Set client options
	clientOptions := options.Client().ApplyURI(config.URI)
	
	if config.MaxPoolSize > 0 {
		clientOptions.SetMaxPoolSize(config.MaxPoolSize)
	}
	if config.MinPoolSize > 0 {
		clientOptions.SetMinPoolSize(config.MinPoolSize)
	}
	if config.MaxConnIdleTime > 0 {
		clientOptions.SetMaxConnIdleTime(config.MaxConnIdleTime)
	}
	if config.ApplicationName != "" {
		clientOptions.SetAppName(config.ApplicationName)
	}

	// Set server selection timeout
	if config.ServerTimeout > 0 {
		clientOptions.SetServerSelectionTimeout(config.ServerTimeout)
	}

	// Connect to MongoDB
	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(config.Database)
	
	log.Printf("Connected to MongoDB database: %s", config.Database)

	connection := &MongoConnection{
		client:   client,
		database: database,
		config:   config,
	}

	// Initialize collections and indexes
	if err := connection.initializeCollections(ctx); err != nil {
		return nil, fmt.Errorf("failed to initialize collections: %w", err)
	}

	return connection, nil
}

// Close closes the MongoDB connection
func (m *MongoConnection) Close(ctx context.Context) error {
	return m.client.Disconnect(ctx)
}

// GetCollection returns a MongoDB collection
func (m *MongoConnection) GetCollection(collectionName string) *mongo.Collection {
	return m.database.Collection(collectionName)
}

// GetDatabase returns the MongoDB database
func (m *MongoConnection) GetDatabase() *mongo.Database {
	return m.database
}

// Health check for MongoDB connection
func (m *MongoConnection) HealthCheck(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	err := m.client.Ping(ctx, readpref.Primary())
	if err != nil {
		return fmt.Errorf("MongoDB health check failed: %w", err)
	}

	return nil
}

// initializeCollections creates collections and indexes
func (m *MongoConnection) initializeCollections(ctx context.Context) error {
	// Collections for KB-2 Clinical Context Service
	collections := []string{
		"clinical_contexts",
		"phenotype_definitions", 
		"patient_profiles",
		"contextual_insights",
		"population_cohorts",
		"risk_stratifications",
		"clinical_patterns",
		"phenotype_matches",
		"context_cache",
	}

	// Create collections if they don't exist
	for _, collectionName := range collections {
		// Check if collection exists
		filter := map[string]interface{}{"name": collectionName}
		collections, err := m.database.ListCollectionNames(ctx, filter)
		if err != nil {
			return fmt.Errorf("failed to list collections: %w", err)
		}

		if len(collections) == 0 {
			// Create collection
			err = m.database.CreateCollection(ctx, collectionName)
			if err != nil {
				log.Printf("Warning: failed to create collection %s: %v", collectionName, err)
			} else {
				log.Printf("Created collection: %s", collectionName)
			}
		}
	}

	// TODO: Fix index creation with bson.D format
	// Skip index creation for now to get service running
	// if err := m.createIndexes(ctx); err != nil {
	// 	return fmt.Errorf("failed to create indexes: %w", err)
	// }

	return nil
}

// createIndexes creates necessary indexes for performance
func (m *MongoConnection) createIndexes(ctx context.Context) error {
	indexModels := map[string][]mongo.IndexModel{
		"clinical_contexts": {
			{
				Keys: map[string]int{
					"patient_id": 1,
					"context_type": 1,
				},
				Options: options.Index().SetUnique(false).SetName("patient_context_idx"),
			},
			{
				Keys: map[string]int{
					"created_at": -1,
				},
				Options: options.Index().SetName("created_at_idx"),
			},
			{
				Keys: map[string]int{
					"clinical_indicators.condition_codes": 1,
				},
				Options: options.Index().SetName("condition_codes_idx"),
			},
		},
		"phenotype_definitions": {
			{
				Keys: map[string]int{
					"phenotype_id": 1,
				},
				Options: options.Index().SetUnique(true).SetName("phenotype_id_unique"),
			},
			{
				Keys: map[string]int{
					"category": 1,
					"severity": 1,
				},
				Options: options.Index().SetName("category_severity_idx"),
			},
			{
				Keys: map[string]int{
					"icd10_codes": 1,
				},
				Options: options.Index().SetName("icd10_codes_idx"),
			},
		},
		"patient_profiles": {
			{
				Keys: map[string]int{
					"patient_id": 1,
				},
				Options: options.Index().SetUnique(true).SetName("patient_id_unique"),
			},
			{
				Keys: map[string]int{
					"demographics.age_range": 1,
					"demographics.gender": 1,
				},
				Options: options.Index().SetName("demographics_idx"),
			},
			{
				Keys: map[string]int{
					"phenotypes.phenotype_id": 1,
				},
				Options: options.Index().SetName("phenotypes_idx"),
			},
		},
		"contextual_insights": {
			{
				Keys: map[string]int{
					"patient_id": 1,
					"insight_type": 1,
				},
				Options: options.Index().SetName("patient_insight_idx"),
			},
			{
				Keys: map[string]int{
					"confidence_score": -1,
				},
				Options: options.Index().SetName("confidence_score_idx"),
			},
			{
				Keys: map[string]int{
					"generated_at": -1,
				},
				Options: options.Index().SetName("generated_at_idx"),
			},
		},
		"population_cohorts": {
			{
				Keys: map[string]int{
					"cohort_id": 1,
				},
				Options: options.Index().SetUnique(true).SetName("cohort_id_unique"),
			},
			{
				Keys: map[string]int{
					"criteria.phenotypes": 1,
				},
				Options: options.Index().SetName("cohort_phenotypes_idx"),
			},
			{
				Keys: map[string]int{
					"criteria.age_range.min": 1,
					"criteria.age_range.max": 1,
				},
				Options: options.Index().SetName("age_range_idx"),
			},
		},
		"clinical_patterns": {
			{
				Keys: map[string]int{
					"pattern_type": 1,
					"frequency": -1,
				},
				Options: options.Index().SetName("pattern_frequency_idx"),
			},
			{
				Keys: map[string]int{
					"phenotype_combinations": 1,
				},
				Options: options.Index().SetName("phenotype_combinations_idx"),
			},
		},
		"context_cache": {
			{
				Keys: map[string]int{
					"cache_key": 1,
				},
				Options: options.Index().SetUnique(true).SetName("cache_key_unique"),
			},
			{
				Keys: map[string]int{
					"expires_at": 1,
				},
				Options: options.Index().SetExpireAfterSeconds(0).SetName("ttl_idx"),
			},
		},
	}

	// Create indexes for each collection
	for collectionName, indexes := range indexModels {
		collection := m.database.Collection(collectionName)
		
		for _, indexModel := range indexes {
			_, err := collection.Indexes().CreateOne(ctx, indexModel)
			if err != nil {
				// Log warning but continue - indexes might already exist
				log.Printf("Warning: failed to create index %s on collection %s: %v", 
					*indexModel.Options.Name, collectionName, err)
			} else {
				log.Printf("Created index %s on collection %s", 
					*indexModel.Options.Name, collectionName)
			}
		}
	}

	return nil
}

// GetCollectionStats returns statistics about MongoDB collections
func (m *MongoConnection) GetCollectionStats(ctx context.Context) (map[string]interface{}, error) {
	stats := make(map[string]interface{})
	
	collections := []string{
		"clinical_contexts",
		"phenotype_definitions", 
		"patient_profiles",
		"contextual_insights",
		"population_cohorts",
	}

	for _, collectionName := range collections {
		collection := m.database.Collection(collectionName)
		
		// Get document count
		count, err := collection.CountDocuments(ctx, map[string]interface{}{})
		if err != nil {
			log.Printf("Warning: failed to get count for %s: %v", collectionName, err)
			continue
		}

		stats[collectionName] = map[string]interface{}{
			"document_count": count,
		}
	}

	return stats, nil
}

// Transaction helper for MongoDB operations
func (m *MongoConnection) WithTransaction(ctx context.Context, fn func(sessCtx mongo.SessionContext) error) error {
	session, err := m.client.StartSession()
	if err != nil {
		return fmt.Errorf("failed to start session: %w", err)
	}
	defer session.EndSession(ctx)

	_, err = session.WithTransaction(ctx, func(sessCtx mongo.SessionContext) (interface{}, error) {
		return nil, fn(sessCtx)
	})

	return err
}

// Aggregation helper
func (m *MongoConnection) Aggregate(ctx context.Context, collectionName string, pipeline []interface{}) (*mongo.Cursor, error) {
	collection := m.database.Collection(collectionName)
	return collection.Aggregate(ctx, pipeline)
}

// BulkWrite helper for efficient batch operations
func (m *MongoConnection) BulkWrite(ctx context.Context, collectionName string, models []mongo.WriteModel) (*mongo.BulkWriteResult, error) {
	collection := m.database.Collection(collectionName)
	opts := options.BulkWrite().SetOrdered(false) // Unordered for better performance
	return collection.BulkWrite(ctx, models, opts)
}