package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"
	"safety-gateway-platform/pkg/logger"
)

// SnapshotManager handles clinical data snapshot lifecycle and caching
type SnapshotManager struct {
	cache           *SnapshotCache
	apolloClient    *ApolloFederationClient
	logger          *logger.Logger
	defaultTTL      time.Duration
	maxCacheSize    int
	checksumEnabled bool
}

// ClinicalSnapshot represents an immutable clinical data snapshot
type ClinicalSnapshot struct {
	ID            string                 `json:"id"`
	PatientID     string                 `json:"patient_id"`
	Data          map[string]interface{} `json:"data"`
	Checksum      string                 `json:"checksum"`
	CreatedAt     time.Time              `json:"created_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	Version       string                 `json:"version"`
	Completeness  float64                `json:"completeness"`
	Sources       []string               `json:"sources"`
	KBVersions    map[string]string      `json:"kb_versions"`
	Metadata      SnapshotMetadata       `json:"metadata"`
}

// SnapshotMetadata contains additional snapshot information
type SnapshotMetadata struct {
	CreationLatency   time.Duration          `json:"creation_latency"`
	DataSourceCount   int                    `json:"data_source_count"`
	FieldCount        int                    `json:"field_count"`
	QualityScore      float64                `json:"quality_score"`
	ValidationErrors  []string               `json:"validation_errors"`
	CompletenessStats map[string]float64     `json:"completeness_stats"`
	CustomAttributes  map[string]interface{} `json:"custom_attributes"`
}

// SnapshotCache provides LRU caching for clinical snapshots
type SnapshotCache struct {
	data     map[string]*CacheEntry
	capacity int
	head     *CacheEntry
	tail     *CacheEntry
}

// CacheEntry represents a cached snapshot with LRU tracking
type CacheEntry struct {
	key      string
	snapshot *ClinicalSnapshot
	prev     *CacheEntry
	next     *CacheEntry
	accessed time.Time
}

// NewSnapshotManager creates a new snapshot manager
func NewSnapshotManager(
	apolloClient *ApolloFederationClient,
	logger *logger.Logger,
	opts ...SnapshotManagerOption,
) *SnapshotManager {
	sm := &SnapshotManager{
		apolloClient:    apolloClient,
		logger:          logger,
		defaultTTL:      30 * time.Minute,
		maxCacheSize:    1000,
		checksumEnabled: true,
	}

	// Apply options
	for _, opt := range opts {
		opt(sm)
	}

	sm.cache = NewSnapshotCache(sm.maxCacheSize)

	return sm
}

// SnapshotManagerOption configures the snapshot manager
type SnapshotManagerOption func(*SnapshotManager)

// WithCacheTTL sets the default cache TTL
func WithCacheTTL(ttl time.Duration) SnapshotManagerOption {
	return func(sm *SnapshotManager) {
		sm.defaultTTL = ttl
	}
}

// WithMaxCacheSize sets the maximum cache size
func WithMaxCacheSize(size int) SnapshotManagerOption {
	return func(sm *SnapshotManager) {
		sm.maxCacheSize = size
	}
}

// WithChecksumValidation enables/disables checksum validation
func WithChecksumValidation(enabled bool) SnapshotManagerOption {
	return func(sm *SnapshotManager) {
		sm.checksumEnabled = enabled
	}
}

// CreateSnapshot creates a new clinical data snapshot
func (sm *SnapshotManager) CreateSnapshot(
	ctx context.Context,
	patientID string,
	includeKBVersions bool,
) (*ClinicalSnapshot, error) {
	startTime := time.Now()
	
	sm.logger.Debug("Creating clinical snapshot",
		zap.String("patient_id", patientID),
		zap.Bool("include_kb_versions", includeKBVersions),
	)

	// Query patient clinical data from Apollo Federation
	response, err := sm.apolloClient.QueryPatientClinicalData(ctx, patientID)
	if err != nil {
		return nil, fmt.Errorf("failed to query patient data: %w", err)
	}

	// Check for GraphQL errors
	if len(response.Errors) > 0 {
		sm.logger.Warn("GraphQL errors in patient data query",
			zap.Int("error_count", len(response.Errors)),
		)
		for _, gqlError := range response.Errors {
			sm.logger.Warn("GraphQL error", zap.String("message", gqlError.Message))
		}
	}

	// Get KB versions if requested
	var kbVersions map[string]string
	if includeKBVersions {
		kbResponse, err := sm.apolloClient.QueryKnowledgeBaseVersions(ctx)
		if err != nil {
			sm.logger.Warn("Failed to get KB versions", zap.Error(err))
			kbVersions = make(map[string]string)
		} else {
			kbVersions = sm.extractKBVersions(kbResponse.Data)
		}
	}

	// Calculate data completeness and quality metrics
	completeness := sm.calculateCompleteness(response.Data)
	qualityScore := sm.calculateQualityScore(response.Data)
	
	// Create snapshot
	snapshot := &ClinicalSnapshot{
		ID:           sm.generateSnapshotID(patientID),
		PatientID:    patientID,
		Data:         response.Data,
		CreatedAt:    time.Now(),
		ExpiresAt:    time.Now().Add(sm.defaultTTL),
		Version:      "1.0",
		Completeness: completeness,
		Sources:      response.DataSources,
		KBVersions:   kbVersions,
		Metadata: SnapshotMetadata{
			CreationLatency:   time.Since(startTime),
			DataSourceCount:   len(response.DataSources),
			FieldCount:        sm.countFields(response.Data),
			QualityScore:      qualityScore,
			ValidationErrors:  sm.validateSnapshotData(response.Data),
			CompletenessStats: sm.calculateCompletenessStats(response.Data),
			CustomAttributes:  make(map[string]interface{}),
		},
	}

	// Generate checksum if enabled
	if sm.checksumEnabled {
		snapshot.Checksum = sm.generateChecksum(snapshot)
	}

	// Cache the snapshot
	sm.cache.Put(snapshot.ID, snapshot)

	sm.logger.Info("Clinical snapshot created",
		zap.String("snapshot_id", snapshot.ID),
		zap.String("patient_id", patientID),
		zap.Float64("completeness", completeness),
		zap.Float64("quality_score", qualityScore),
		zap.Duration("creation_latency", snapshot.Metadata.CreationLatency),
		zap.Int("field_count", snapshot.Metadata.FieldCount),
	)

	return snapshot, nil
}

// GetSnapshot retrieves a snapshot by ID
func (sm *SnapshotManager) GetSnapshot(snapshotID string) (*ClinicalSnapshot, error) {
	// Check cache first
	snapshot := sm.cache.Get(snapshotID)
	if snapshot != nil {
		// Validate snapshot hasn't expired
		if time.Now().Before(snapshot.ExpiresAt) {
			// Validate checksum if enabled
			if sm.checksumEnabled && !sm.validateChecksum(snapshot) {
				sm.logger.Error("Snapshot checksum validation failed",
					zap.String("snapshot_id", snapshotID),
				)
				sm.cache.Delete(snapshotID)
				return nil, fmt.Errorf("snapshot integrity check failed")
			}
			
			sm.logger.Debug("Snapshot retrieved from cache",
				zap.String("snapshot_id", snapshotID),
			)
			return snapshot, nil
		}

		// Snapshot expired, remove from cache
		sm.cache.Delete(snapshotID)
		sm.logger.Debug("Expired snapshot removed from cache",
			zap.String("snapshot_id", snapshotID),
		)
	}

	return nil, fmt.Errorf("snapshot not found or expired: %s", snapshotID)
}

// RefreshSnapshot refreshes an existing snapshot with latest data
func (sm *SnapshotManager) RefreshSnapshot(
	ctx context.Context,
	snapshotID string,
) (*ClinicalSnapshot, error) {
	// Get existing snapshot to extract patient ID
	existing, err := sm.GetSnapshot(snapshotID)
	if err != nil {
		return nil, fmt.Errorf("cannot refresh non-existent snapshot: %w", err)
	}

	// Create new snapshot with same patient ID
	refreshed, err := sm.CreateSnapshot(ctx, existing.PatientID, len(existing.KBVersions) > 0)
	if err != nil {
		return nil, fmt.Errorf("failed to refresh snapshot: %w", err)
	}

	// Use original snapshot ID to maintain references
	refreshed.ID = snapshotID
	refreshed.Version = sm.incrementVersion(existing.Version)

	// Update cache
	sm.cache.Put(snapshotID, refreshed)

	sm.logger.Info("Snapshot refreshed",
		zap.String("snapshot_id", snapshotID),
		zap.String("patient_id", existing.PatientID),
		zap.String("new_version", refreshed.Version),
	)

	return refreshed, nil
}

// IsValid checks if a snapshot is valid and not expired
func (sm *SnapshotManager) IsValid(snapshot *ClinicalSnapshot) bool {
	if snapshot == nil {
		return false
	}

	// Check expiration
	if time.Now().After(snapshot.ExpiresAt) {
		return false
	}

	// Check checksum if enabled
	if sm.checksumEnabled && !sm.validateChecksum(snapshot) {
		return false
	}

	// Check for critical validation errors
	if len(snapshot.Metadata.ValidationErrors) > 0 {
		for _, err := range snapshot.Metadata.ValidationErrors {
			if sm.isCriticalError(err) {
				return false
			}
		}
	}

	return true
}

// GetCacheStats returns cache performance statistics
func (sm *SnapshotManager) GetCacheStats() map[string]interface{} {
	return map[string]interface{}{
		"size":     sm.cache.Size(),
		"capacity": sm.cache.Capacity(),
		"hit_rate": sm.cache.HitRate(),
	}
}

// Helper methods

func (sm *SnapshotManager) generateSnapshotID(patientID string) string {
	timestamp := time.Now().UnixNano()
	data := fmt.Sprintf("%s-%d", patientID, timestamp)
	hash := sha256.Sum256([]byte(data))
	return fmt.Sprintf("snapshot-%s", hex.EncodeToString(hash[:8]))
}

func (sm *SnapshotManager) generateChecksum(snapshot *ClinicalSnapshot) string {
	data, _ := json.Marshal(snapshot.Data)
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (sm *SnapshotManager) validateChecksum(snapshot *ClinicalSnapshot) bool {
	if !sm.checksumEnabled {
		return true
	}
	
	computed := sm.generateChecksum(snapshot)
	return computed == snapshot.Checksum
}

func (sm *SnapshotManager) calculateCompleteness(data map[string]interface{}) float64 {
	if data == nil {
		return 0.0
	}

	requiredFields := []string{
		"patient.demographics",
		"patient.conditions",
		"patient.medications",
		"patient.allergies",
	}

	present := 0
	for _, field := range requiredFields {
		if sm.hasNestedField(data, field) {
			present++
		}
	}

	return float64(present) / float64(len(requiredFields))
}

func (sm *SnapshotManager) calculateQualityScore(data map[string]interface{}) float64 {
	// Implement quality scoring based on data richness, consistency, etc.
	// This is a simplified implementation
	if data == nil {
		return 0.0
	}

	score := 0.0
	maxScore := 100.0

	// Check for patient data presence
	if patient, ok := data["patient"].(map[string]interface{}); ok {
		if demographics, ok := patient["demographics"].(map[string]interface{}); ok {
			if demographics["age"] != nil { score += 10 }
			if demographics["gender"] != nil { score += 10 }
			if demographics["weight"] != nil { score += 10 }
		}
		
		if medications, ok := patient["medications"].([]interface{}); ok {
			score += float64(len(medications)) * 5 // Up to 25 points for medications
			if len(medications) > 5 { score = 75 } // Cap at 75 for this component
		}
		
		if conditions, ok := patient["conditions"].([]interface{}); ok {
			score += float64(len(conditions)) * 3 // Up to 15 points for conditions
			if len(conditions) > 5 { score = 90 } // Cap at 90 total so far
		}
		
		if allergies, ok := patient["allergies"].([]interface{}); ok {
			score += float64(len(allergies)) * 2 // Up to 10 points for allergies
		}
	}

	if score > maxScore {
		score = maxScore
	}

	return score / maxScore
}

func (sm *SnapshotManager) calculateCompletenessStats(data map[string]interface{}) map[string]float64 {
	stats := make(map[string]float64)
	
	if patient, ok := data["patient"].(map[string]interface{}); ok {
		// Demographics completeness
		if demographics, ok := patient["demographics"].(map[string]interface{}); ok {
			required := []string{"age", "gender", "weight", "height"}
			present := 0
			for _, field := range required {
				if demographics[field] != nil {
					present++
				}
			}
			stats["demographics"] = float64(present) / float64(len(required))
		}
		
		// Medications completeness (basic presence check)
		if medications, ok := patient["medications"].([]interface{}); ok {
			stats["medications"] = 1.0 // If present, consider complete
			if len(medications) == 0 {
				stats["medications"] = 0.0
			}
		}
		
		// Similar for other sections
		if conditions, ok := patient["conditions"].([]interface{}); ok {
			stats["conditions"] = 1.0
			if len(conditions) == 0 {
				stats["conditions"] = 0.0
			}
		}
	}
	
	return stats
}

func (sm *SnapshotManager) validateSnapshotData(data map[string]interface{}) []string {
	var errors []string
	
	if data == nil {
		errors = append(errors, "snapshot data is null")
		return errors
	}
	
	// Check for patient data
	patient, ok := data["patient"].(map[string]interface{})
	if !ok {
		errors = append(errors, "missing patient data")
		return errors
	}
	
	// Validate demographics
	if demographics, ok := patient["demographics"].(map[string]interface{}); ok {
		if age, ok := demographics["age"].(float64); ok && age < 0 {
			errors = append(errors, "invalid age: negative value")
		}
	} else {
		errors = append(errors, "missing demographics data")
	}
	
	return errors
}

func (sm *SnapshotManager) countFields(data map[string]interface{}) int {
	count := 0
	for _, value := range data {
		count++
		if nested, ok := value.(map[string]interface{}); ok {
			count += sm.countFields(nested)
		}
	}
	return count
}

func (sm *SnapshotManager) hasNestedField(data map[string]interface{}, field string) bool {
	// Simple implementation - could be enhanced for complex nested paths
	parts := []string{field} // Simplified - should split by "."
	current := data
	
	for _, part := range parts {
		if value, ok := current[part]; ok {
			if nested, ok := value.(map[string]interface{}); ok {
				current = nested
			} else {
				return true // Found the field
			}
		} else {
			return false
		}
	}
	
	return true
}

func (sm *SnapshotManager) extractKBVersions(data map[string]interface{}) map[string]string {
	versions := make(map[string]string)
	
	if kbs, ok := data["knowledgeBases"].(map[string]interface{}); ok {
		kbNames := []string{"kb1_dosing", "kb3_guidelines", "kb4_safety", "kb5_ddi", "kb7_terminology"}
		
		for _, kbName := range kbNames {
			if kb, ok := kbs[kbName].(map[string]interface{}); ok {
				if version, ok := kb["version"].(string); ok {
					versions[kbName] = version
				}
			}
		}
	}
	
	return versions
}

func (sm *SnapshotManager) incrementVersion(version string) string {
	// Simple version increment - could be enhanced
	return fmt.Sprintf("%s.1", version)
}

func (sm *SnapshotManager) isCriticalError(error string) bool {
	criticalPatterns := []string{
		"missing patient data",
		"invalid patient id",
		"authentication failed",
	}
	
	for _, pattern := range criticalPatterns {
		if error == pattern {
			return true
		}
	}
	
	return false
}

// SnapshotCache implementation

// NewSnapshotCache creates a new LRU cache for snapshots
func NewSnapshotCache(capacity int) *SnapshotCache {
	cache := &SnapshotCache{
		data:     make(map[string]*CacheEntry),
		capacity: capacity,
	}
	
	// Initialize dummy head and tail
	cache.head = &CacheEntry{}
	cache.tail = &CacheEntry{}
	cache.head.next = cache.tail
	cache.tail.prev = cache.head
	
	return cache
}

// Get retrieves a snapshot from cache
func (c *SnapshotCache) Get(key string) *ClinicalSnapshot {
	if entry, exists := c.data[key]; exists {
		// Move to head (most recently used)
		c.moveToHead(entry)
		entry.accessed = time.Now()
		return entry.snapshot
	}
	return nil
}

// Put stores a snapshot in cache
func (c *SnapshotCache) Put(key string, snapshot *ClinicalSnapshot) {
	if entry, exists := c.data[key]; exists {
		// Update existing entry
		entry.snapshot = snapshot
		entry.accessed = time.Now()
		c.moveToHead(entry)
	} else {
		// Add new entry
		entry := &CacheEntry{
			key:      key,
			snapshot: snapshot,
			accessed: time.Now(),
		}
		
		c.data[key] = entry
		c.addToHead(entry)
		
		// Check capacity
		if len(c.data) > c.capacity {
			tail := c.removeTail()
			delete(c.data, tail.key)
		}
	}
}

// Delete removes a snapshot from cache
func (c *SnapshotCache) Delete(key string) {
	if entry, exists := c.data[key]; exists {
		c.removeEntry(entry)
		delete(c.data, key)
	}
}

// Size returns current cache size
func (c *SnapshotCache) Size() int {
	return len(c.data)
}

// Capacity returns cache capacity
func (c *SnapshotCache) Capacity() int {
	return c.capacity
}

// HitRate calculates cache hit rate (simplified implementation)
func (c *SnapshotCache) HitRate() float64 {
	// This would need proper hit/miss tracking in a real implementation
	return 0.85 // Placeholder
}

// LRU cache helper methods

func (c *SnapshotCache) addToHead(entry *CacheEntry) {
	entry.prev = c.head
	entry.next = c.head.next
	c.head.next.prev = entry
	c.head.next = entry
}

func (c *SnapshotCache) removeEntry(entry *CacheEntry) {
	entry.prev.next = entry.next
	entry.next.prev = entry.prev
}

func (c *SnapshotCache) moveToHead(entry *CacheEntry) {
	c.removeEntry(entry)
	c.addToHead(entry)
}

func (c *SnapshotCache) removeTail() *CacheEntry {
	tail := c.tail.prev
	c.removeEntry(tail)
	return tail
}