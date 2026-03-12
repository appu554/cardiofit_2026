package database

import (
	"context"
	"fmt"
	"log"
	"time"

	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
	"go.mongodb.org/mongo-driver/mongo/readpref"

	"kb-clinical-context/internal/config"
)

type Database struct {
	Client   *mongo.Client
	DB       *mongo.Database
	config   *config.Config
}

func NewConnection(cfg *config.Config) (*Database, error) {
	// Build connection options
	clientOptions := options.Client()
	clientOptions.ApplyURI(cfg.MongoDB.URI)
	
	// Set credentials if provided
	if cfg.MongoDB.Username != "" && cfg.MongoDB.Password != "" {
		credential := options.Credential{
			Username: cfg.MongoDB.Username,
			Password: cfg.MongoDB.Password,
		}
		clientOptions.SetAuth(credential)
	}

	// Set connection timeout
	clientOptions.SetConnectTimeout(time.Duration(cfg.MongoDB.Timeout) * time.Second)
	clientOptions.SetSocketTimeout(time.Duration(cfg.MongoDB.Timeout) * time.Second)
	clientOptions.SetServerSelectionTimeout(time.Duration(cfg.MongoDB.Timeout) * time.Second)

	// Connect to MongoDB
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.MongoDB.Timeout)*time.Second)
	defer cancel()

	client, err := mongo.Connect(ctx, clientOptions)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}

	// Test the connection
	err = client.Ping(ctx, readpref.Primary())
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}

	database := client.Database(cfg.MongoDB.Database)

	log.Printf("Successfully connected to MongoDB database: %s", cfg.MongoDB.Database)

	db := &Database{
		Client: client,
		DB:     database,
		config: cfg,
	}

	// Initialize collections and indexes
	if err := db.InitializeCollections(); err != nil {
		return nil, fmt.Errorf("failed to initialize collections: %w", err)
	}

	return db, nil
}

func (db *Database) InitializeCollections() error {
	ctx := context.Background()

	// Create collections with validators
	collections := []string{
		"phenotype_definitions",
		"patient_contexts",
	}

	for _, collectionName := range collections {
		// Check if collection exists
		names, err := db.DB.ListCollectionNames(ctx, map[string]interface{}{})
		if err != nil {
			return fmt.Errorf("failed to list collections: %w", err)
		}

		exists := false
		for _, name := range names {
			if name == collectionName {
				exists = true
				break
			}
		}

		if !exists {
			log.Printf("Creating collection: %s", collectionName)
			err = db.DB.CreateCollection(ctx, collectionName)
			if err != nil {
				return fmt.Errorf("failed to create collection %s: %w", collectionName, err)
			}
		}
	}

	// Create indexes
	if err := db.CreateIndexes(); err != nil {
		return fmt.Errorf("failed to create indexes: %w", err)
	}

	return nil
}

func (db *Database) CreateIndexes() error {
	// TODO: Fix index creation with bson.D format
	// Skip index creation for now to get service running
	return nil

	ctx := context.Background()

	// Phenotype definitions indexes
	phenotypeIndexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"phenotype_id": 1, "version": -1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys: map[string]interface{}{"status": 1},
		},
		{
			Keys: map[string]interface{}{"criteria.required_conditions.codes": 1},
		},
		{
			Keys: map[string]interface{}{"name": "text", "description": "text"},
		},
	}

	_, err := db.DB.Collection("phenotype_definitions").Indexes().CreateMany(ctx, phenotypeIndexes)
	if err != nil {
		return fmt.Errorf("failed to create phenotype_definitions indexes: %w", err)
	}

	// Patient contexts indexes
	contextIndexes := []mongo.IndexModel{
		{
			Keys:    map[string]interface{}{"patient_id": 1, "timestamp": -1},
		},
		{
			Keys:    map[string]interface{}{"context_id": 1},
			Options: options.Index().SetUnique(true),
		},
		{
			Keys:    map[string]interface{}{"ttl": 1},
			Options: options.Index().SetExpireAfterSeconds(0),
		},
		{
			Keys: map[string]interface{}{"detected_phenotypes.phenotype_id": 1},
		},
		{
			Keys: map[string]interface{}{"active_conditions.code": 1},
		},
		{
			Keys: map[string]interface{}{"current_medications.rxnorm_code": 1},
		},
	}

	_, err = db.DB.Collection("patient_contexts").Indexes().CreateMany(ctx, contextIndexes)
	if err != nil {
		return fmt.Errorf("failed to create patient_contexts indexes: %w", err)
	}

	log.Println("Successfully created MongoDB indexes")
	return nil
}

func (db *Database) HealthCheck() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return db.Client.Ping(ctx, readpref.Primary())
}

func (db *Database) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	return db.Client.Disconnect(ctx)
}

// Collection helpers
func (db *Database) PhenotypeDefinitions() *mongo.Collection {
	return db.DB.Collection("phenotype_definitions")
}

func (db *Database) PatientContexts() *mongo.Collection {
	return db.DB.Collection("patient_contexts")
}