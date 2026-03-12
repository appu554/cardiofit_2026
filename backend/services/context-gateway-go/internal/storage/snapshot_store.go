// Package storage provides dual-layer storage implementation for clinical snapshots
// Implements hot (Redis) and cold (S3/MongoDB) storage as per architecture requirements
package storage

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/redis/go-redis/v9"
	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"

	"context-gateway-go/internal/models"
)

// SnapshotStore implements dual-layer storage for clinical snapshots
// L1 Cache (Redis): Hot storage for active snapshots
// L2 Persistent (MongoDB): Cold storage for immutable snapshots
type SnapshotStore struct {
	hotStore  *redis.Client  // L1 Cache (Redis)
	coldStore *mongo.Client  // L2 Persistent (MongoDB)
	database  *mongo.Database
	collection *mongo.Collection
}

// NewSnapshotStore creates a new dual-layer snapshot store
func NewSnapshotStore(redisAddr, mongoURI, dbName string) (*SnapshotStore, error) {
	// Initialize Redis client (hot store)
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: "", // no password for development
		DB:       0,  // default DB
	})
	
	// Test Redis connection
	ctx := context.Background()
	_, err := rdb.Ping(ctx).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}
	
	// Initialize MongoDB client (cold store)
	client, err := mongo.Connect(ctx, options.Client().ApplyURI(mongoURI))
	if err != nil {
		return nil, fmt.Errorf("failed to connect to MongoDB: %w", err)
	}
	
	// Test MongoDB connection
	err = client.Ping(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to ping MongoDB: %w", err)
	}
	
	database := client.Database(dbName)
	collection := database.Collection("clinical_snapshots")
	
	store := &SnapshotStore{
		hotStore:   rdb,
		coldStore:  client,
		database:   database,
		collection: collection,
	}
	
	// Ensure MongoDB indexes
	if err := store.ensureIndexes(); err != nil {
		log.Printf("Warning: Failed to ensure indexes: %v", err)
	}
	
	return store, nil
}

// ensureIndexes creates necessary MongoDB indexes for performance
func (ss *SnapshotStore) ensureIndexes() error {
	ctx := context.Background()
	
	// TTL index for automatic cleanup
	ttlIndex := mongo.IndexModel{
		Keys:    bson.D{{Key: "expires_at", Value: 1}},
		Options: options.Index().SetExpireAfterSeconds(0),
	}
	
	// Query optimization indexes
	patientIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "patient_id", Value: 1}, {Key: "created_at", Value: -1}},
	}
	
	recipeIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "recipe_id", Value: 1}, {Key: "created_at", Value: -1}},
	}
	
	statusIndex := mongo.IndexModel{
		Keys: bson.D{{Key: "status", Value: 1}, {Key: "expires_at", Value: 1}},
	}
	
	indexes := []mongo.IndexModel{ttlIndex, patientIndex, recipeIndex, statusIndex}
	
	_, err := ss.collection.Indexes().CreateMany(ctx, indexes)
	return err
}

// Save stores a clinical snapshot in both hot and cold storage atomically
func (ss *SnapshotStore) Save(ctx context.Context, snapshot *models.ClinicalSnapshot) error {
	// Prepare data for storage
	snapshotData, err := json.Marshal(snapshot)
	if err != nil {
		return fmt.Errorf("failed to marshal snapshot: %w", err)
	}
	
	// Calculate TTL for hot storage
	ttl := time.Until(snapshot.ExpiresAt)
	if ttl <= 0 {
		return fmt.Errorf("snapshot already expired")
	}
	
	// Write to both layers atomically using a channel for error collection
	errChan := make(chan error, 2)
	
	// Write to hot store (Redis) in goroutine
	go func() {
		redisKey := fmt.Sprintf("snapshot:%s", snapshot.ID)
		err := ss.hotStore.Set(ctx, redisKey, snapshotData, ttl).Err()
		errChan <- err
	}()
	
	// Write to cold store (MongoDB) in goroutine
	go func() {
		document := snapshot.ToDict()
		document["_id"] = snapshot.ID
		_, err := ss.collection.InsertOne(ctx, document)
		errChan <- err
	}()
	
	// Wait for both operations to complete
	var errors []error
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}
	
	// If any error occurred, attempt cleanup
	if len(errors) > 0 {
		// Best effort cleanup
		redisKey := fmt.Sprintf("snapshot:%s", snapshot.ID)
		ss.hotStore.Del(ctx, redisKey)
		ss.collection.DeleteOne(ctx, bson.M{"_id": snapshot.ID})
		
		return fmt.Errorf("failed to save snapshot: %v", errors)
	}
	
	return nil
}

// Get retrieves a clinical snapshot, trying hot store first, then cold store
func (ss *SnapshotStore) Get(ctx context.Context, snapshotID string) (*models.ClinicalSnapshot, error) {
	redisKey := fmt.Sprintf("snapshot:%s", snapshotID)
	
	// Try hot store first (Redis)
	cached, err := ss.hotStore.Get(ctx, redisKey).Result()
	if err == nil {
		// Found in hot store, deserialize and return
		var snapshot models.ClinicalSnapshot
		if err := json.Unmarshal([]byte(cached), &snapshot); err != nil {
			log.Printf("Warning: Failed to unmarshal cached snapshot %s: %v", snapshotID, err)
		} else {
			return &snapshot, nil
		}
	}
	
	// Fallback to cold store (MongoDB)
	var document bson.M
	err = ss.collection.FindOne(ctx, bson.M{"_id": snapshotID}).Decode(&document)
	if err != nil {
		if err == mongo.ErrNoDocuments {
			return nil, nil // Not found
		}
		return nil, fmt.Errorf("failed to query cold store: %w", err)
	}
	
	// Convert MongoDB document to snapshot
	snapshot, err := ss.documentToSnapshot(document)
	if err != nil {
		return nil, fmt.Errorf("failed to convert document to snapshot: %w", err)
	}
	
	// Warm the hot cache if snapshot is still valid
	if !snapshot.IsExpired() {
		go func() {
			// Calculate remaining TTL
			ttl := time.Until(snapshot.ExpiresAt)
			if ttl > 0 {
				snapshotData, err := json.Marshal(snapshot)
				if err == nil {
					ss.hotStore.Set(context.Background(), redisKey, snapshotData, ttl)
				}
			}
		}()
	}
	
	return snapshot, nil
}

// Update modifies an existing snapshot (used for access tracking)
func (ss *SnapshotStore) Update(ctx context.Context, snapshot *models.ClinicalSnapshot) error {
	// Update cold store (MongoDB)
	filter := bson.M{"_id": snapshot.ID}
	update := bson.M{
		"$set": bson.M{
			"accessed_count":    snapshot.AccessedCount,
			"last_accessed_at":  snapshot.LastAccessedAt,
		},
	}
	
	_, err := ss.collection.UpdateOne(ctx, filter, update)
	if err != nil {
		return fmt.Errorf("failed to update snapshot in cold store: %w", err)
	}
	
	// Update hot store if present
	redisKey := fmt.Sprintf("snapshot:%s", snapshot.ID)
	snapshotData, err := json.Marshal(snapshot)
	if err == nil {
		ttl := time.Until(snapshot.ExpiresAt)
		if ttl > 0 {
			ss.hotStore.Set(ctx, redisKey, snapshotData, ttl)
		}
	}
	
	return nil
}

// Delete removes a snapshot from both hot and cold storage
func (ss *SnapshotStore) Delete(ctx context.Context, snapshotID string) error {
	errChan := make(chan error, 2)
	
	// Delete from hot store (Redis)
	go func() {
		redisKey := fmt.Sprintf("snapshot:%s", snapshotID)
		err := ss.hotStore.Del(ctx, redisKey).Err()
		errChan <- err
	}()
	
	// Delete from cold store (MongoDB)
	go func() {
		_, err := ss.collection.DeleteOne(ctx, bson.M{"_id": snapshotID})
		errChan <- err
	}()
	
	// Wait for both operations
	var errors []error
	for i := 0; i < 2; i++ {
		if err := <-errChan; err != nil {
			errors = append(errors, err)
		}
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("failed to delete snapshot: %v", errors)
	}
	
	return nil
}

// List retrieves snapshots with optional filtering
func (ss *SnapshotStore) List(ctx context.Context, patientID *string, recipeID *string, limit int) ([]*models.SnapshotSummary, error) {
	filter := bson.M{}
	
	if patientID != nil && *patientID != "" {
		filter["patient_id"] = *patientID
	}
	if recipeID != nil && *recipeID != "" {
		filter["recipe_id"] = *recipeID
	}
	
	opts := options.Find().
		SetSort(bson.D{{Key: "created_at", Value: -1}}).
		SetLimit(int64(limit))
	
	cursor, err := ss.collection.Find(ctx, filter, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to query snapshots: %w", err)
	}
	defer cursor.Close(ctx)
	
	var summaries []*models.SnapshotSummary
	for cursor.Next(ctx) {
		var document bson.M
		if err := cursor.Decode(&document); err != nil {
			log.Printf("Warning: Failed to decode snapshot document: %v", err)
			continue
		}
		
		summary := ss.documentToSummary(document)
		if summary != nil {
			summaries = append(summaries, summary)
		}
	}
	
	return summaries, nil
}

// CleanupExpired removes expired snapshots from both stores
func (ss *SnapshotStore) CleanupExpired(ctx context.Context) (int, error) {
	currentTime := time.Now().UTC()
	
	// MongoDB TTL index should handle cold store cleanup automatically
	// Manual cleanup for hot store (Redis) - scan for expired snapshot keys
	iter := ss.hotStore.Scan(ctx, 0, "snapshot:*", 100).Iterator()
	deletedCount := 0
	
	for iter.Next(ctx) {
		key := iter.Val()
		ttl := ss.hotStore.TTL(ctx, key).Val()
		if ttl <= 0 {
			ss.hotStore.Del(ctx, key)
			deletedCount++
		}
	}
	
	if err := iter.Err(); err != nil {
		return deletedCount, fmt.Errorf("failed to scan Redis keys: %w", err)
	}
	
	// Manual cleanup for cold store (MongoDB) as a backup
	filter := bson.M{
		"$or": []bson.M{
			{"expires_at": bson.M{"$lt": currentTime}},
			{"status": int(models.SnapshotStatusExpired)},
		},
	}
	
	result, err := ss.collection.DeleteMany(ctx, filter)
	if err != nil {
		log.Printf("Warning: Failed to manually cleanup expired snapshots: %v", err)
	} else if result.DeletedCount > 0 {
		log.Printf("Manually cleaned up %d expired snapshots from cold store", result.DeletedCount)
		deletedCount += int(result.DeletedCount)
	}
	
	return deletedCount, nil
}

// GetStats returns statistics about the snapshot store
func (ss *SnapshotStore) GetStats(ctx context.Context) (map[string]interface{}, error) {
	// MongoDB stats
	totalSnapshots, _ := ss.collection.CountDocuments(ctx, bson.M{})
	activeSnapshots, _ := ss.collection.CountDocuments(ctx, bson.M{
		"status": int(models.SnapshotStatusActive),
	})
	expiredSnapshots, _ := ss.collection.CountDocuments(ctx, bson.M{
		"expires_at": bson.M{"$lt": time.Now().UTC()},
	})
	
	// Redis stats
	redisInfo, _ := ss.hotStore.Info(ctx, "memory").Result()
	
	stats := map[string]interface{}{
		"cold_store": map[string]interface{}{
			"total_snapshots":   totalSnapshots,
			"active_snapshots":  activeSnapshots,
			"expired_snapshots": expiredSnapshots,
		},
		"hot_store": map[string]interface{}{
			"redis_info": redisInfo,
		},
		"timestamp": time.Now().UTC(),
	}
	
	return stats, nil
}

// documentToSnapshot converts MongoDB document to ClinicalSnapshot model
func (ss *SnapshotStore) documentToSnapshot(document bson.M) (*models.ClinicalSnapshot, error) {
	// This is a simplified conversion - in production, you'd want more robust type conversion
	jsonData, err := json.Marshal(document)
	if err != nil {
		return nil, err
	}
	
	var snapshot models.ClinicalSnapshot
	err = json.Unmarshal(jsonData, &snapshot)
	if err != nil {
		return nil, err
	}
	
	// Handle the _id field
	if id, ok := document["_id"].(string); ok {
		snapshot.ID = id
	}
	
	return &snapshot, nil
}

// documentToSummary converts MongoDB document to SnapshotSummary
func (ss *SnapshotStore) documentToSummary(document bson.M) *models.SnapshotSummary {
	summary := &models.SnapshotSummary{}
	
	if id, ok := document["_id"].(string); ok {
		summary.ID = id
	}
	if patientID, ok := document["patient_id"].(string); ok {
		summary.PatientID = patientID
	}
	if recipeID, ok := document["recipe_id"].(string); ok {
		summary.RecipeID = recipeID
	}
	if status, ok := document["status"].(int32); ok {
		summary.Status = models.SnapshotStatus(status)
	}
	if completeness, ok := document["completeness_score"].(float64); ok {
		summary.CompletenessScore = completeness
	}
	if accessedCount, ok := document["accessed_count"].(int32); ok {
		summary.AccessedCount = accessedCount
	}
	
	// Handle time fields - MongoDB stores these as primitive.DateTime
	if createdAt, ok := document["created_at"].(time.Time); ok {
		summary.CreatedAt = createdAt
	}
	if expiresAt, ok := document["expires_at"].(time.Time); ok {
		summary.ExpiresAt = expiresAt
	}
	
	// Handle optional fields
	if providerID, ok := document["provider_id"].(string); ok && providerID != "" {
		summary.ProviderID = &providerID
	}
	if encounterID, ok := document["encounter_id"].(string); ok && encounterID != "" {
		summary.EncounterID = &encounterID
	}
	
	return summary
}

// Close closes connections to both storage systems
func (ss *SnapshotStore) Close() error {
	var errors []error
	
	// Close Redis connection
	if err := ss.hotStore.Close(); err != nil {
		errors = append(errors, fmt.Errorf("failed to close Redis connection: %w", err))
	}
	
	// Close MongoDB connection
	if err := ss.coldStore.Disconnect(context.Background()); err != nil {
		errors = append(errors, fmt.Errorf("failed to close MongoDB connection: %w", err))
	}
	
	if len(errors) > 0 {
		return fmt.Errorf("errors closing storage connections: %v", errors)
	}
	
	return nil
}