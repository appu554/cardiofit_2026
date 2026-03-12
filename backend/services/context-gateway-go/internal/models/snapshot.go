// Package models provides core data structures for the Context Gateway
package models

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SnapshotStatus represents the current state of a clinical snapshot
type SnapshotStatus int

const (
	SnapshotStatusUnknown SnapshotStatus = iota
	SnapshotStatusActive
	SnapshotStatusExpired
	SnapshotStatusInvalidated
)

// SignatureMethod represents the cryptographic signature method used
type SignatureMethod int

const (
	SignatureMethodUnknown SignatureMethod = iota
	SignatureMethodMock
	SignatureMethodRSA2048
	SignatureMethodECDSAP256
)

// ClinicalSnapshot represents an immutable clinical context snapshot
// with cryptographic integrity verification
type ClinicalSnapshot struct {
	ID                string                 `json:"id" bson:"_id"`
	PatientID         string                 `json:"patient_id" bson:"patient_id"`
	RecipeID          string                 `json:"recipe_id" bson:"recipe_id"`
	ContextID         string                 `json:"context_id" bson:"context_id"`
	Data              map[string]interface{} `json:"data" bson:"data"`
	CompletenessScore float64                `json:"completeness_score" bson:"completeness_score"`
	Checksum          string                 `json:"checksum" bson:"checksum"`
	Signature         string                 `json:"signature" bson:"signature"`
	SignatureMethod   SignatureMethod        `json:"signature_method" bson:"signature_method"`
	CreatedAt         time.Time              `json:"created_at" bson:"created_at"`
	ExpiresAt         time.Time              `json:"expires_at" bson:"expires_at"`
	Status            SnapshotStatus         `json:"status" bson:"status"`
	ProviderID        *string                `json:"provider_id,omitempty" bson:"provider_id,omitempty"`
	EncounterID       *string                `json:"encounter_id,omitempty" bson:"encounter_id,omitempty"`
	AssemblyMetadata  map[string]interface{} `json:"assembly_metadata" bson:"assembly_metadata"`
	EvidenceEnvelope  map[string]interface{} `json:"evidence_envelope" bson:"evidence_envelope"`
	AccessedCount     int32                  `json:"accessed_count" bson:"accessed_count"`
	LastAccessedAt    *time.Time             `json:"last_accessed_at,omitempty" bson:"last_accessed_at,omitempty"`
	AllowLiveFetch    bool                   `json:"allow_live_fetch" bson:"allow_live_fetch"`
	AllowedLiveFields []string               `json:"allowed_live_fields,omitempty" bson:"allowed_live_fields,omitempty"`
}

// NewClinicalSnapshot creates a new clinical snapshot with generated ID
func NewClinicalSnapshot(patientID, recipeID string, data map[string]interface{}) *ClinicalSnapshot {
	now := time.Now().UTC()
	
	snapshot := &ClinicalSnapshot{
		ID:                uuid.New().String(),
		PatientID:         patientID,
		RecipeID:          recipeID,
		ContextID:         uuid.New().String(),
		Data:              data,
		CompletenessScore: 0.0,
		CreatedAt:         now,
		Status:            SnapshotStatusActive,
		AssemblyMetadata:  make(map[string]interface{}),
		EvidenceEnvelope:  make(map[string]interface{}),
		AccessedCount:     0,
		AllowLiveFetch:    false,
	}
	
	// Calculate checksum
	snapshot.Checksum = snapshot.CalculateChecksum()
	
	return snapshot
}

// CalculateChecksum computes SHA-256 checksum of the clinical data
func (s *ClinicalSnapshot) CalculateChecksum() string {
	// Convert data to canonical JSON for consistent hashing
	dataBytes, err := json.Marshal(s.Data)
	if err != nil {
		// If marshaling fails, return empty checksum
		return ""
	}
	
	hash := sha256.Sum256(dataBytes)
	return fmt.Sprintf("%x", hash)
}

// VerifyChecksum verifies the integrity of the snapshot data
func (s *ClinicalSnapshot) VerifyChecksum() bool {
	expectedChecksum := s.CalculateChecksum()
	return s.Checksum == expectedChecksum
}

// IsExpired checks if the snapshot has expired
func (s *ClinicalSnapshot) IsExpired() bool {
	return time.Now().UTC().After(s.ExpiresAt)
}

// MarkAccessed increments the access count and updates last accessed time
func (s *ClinicalSnapshot) MarkAccessed() {
	s.AccessedCount++
	now := time.Now().UTC()
	s.LastAccessedAt = &now
}

// IsValid performs comprehensive validation of the snapshot
func (s *ClinicalSnapshot) IsValid() (bool, []string) {
	var errors []string
	
	// Check basic fields
	if s.ID == "" {
		errors = append(errors, "snapshot ID is required")
	}
	if s.PatientID == "" {
		errors = append(errors, "patient ID is required")
	}
	if s.RecipeID == "" {
		errors = append(errors, "recipe ID is required")
	}
	if s.Data == nil {
		errors = append(errors, "clinical data is required")
	}
	
	// Check expiration
	if s.IsExpired() {
		errors = append(errors, "snapshot has expired")
	}
	
	// Check status
	if s.Status == SnapshotStatusInvalidated {
		errors = append(errors, "snapshot has been invalidated")
	}
	
	// Verify checksum
	if !s.VerifyChecksum() {
		errors = append(errors, "data integrity check failed - checksum mismatch")
	}
	
	return len(errors) == 0, errors
}

// ToDict converts the snapshot to a map for serialization
func (s *ClinicalSnapshot) ToDict() map[string]interface{} {
	result := map[string]interface{}{
		"id":                  s.ID,
		"patient_id":          s.PatientID,
		"recipe_id":           s.RecipeID,
		"context_id":          s.ContextID,
		"data":                s.Data,
		"completeness_score":  s.CompletenessScore,
		"checksum":            s.Checksum,
		"signature":           s.Signature,
		"signature_method":    s.SignatureMethod,
		"created_at":          s.CreatedAt,
		"expires_at":          s.ExpiresAt,
		"status":              s.Status,
		"assembly_metadata":   s.AssemblyMetadata,
		"evidence_envelope":   s.EvidenceEnvelope,
		"accessed_count":      s.AccessedCount,
		"allow_live_fetch":    s.AllowLiveFetch,
		"allowed_live_fields": s.AllowedLiveFields,
	}
	
	if s.ProviderID != nil {
		result["provider_id"] = *s.ProviderID
	}
	if s.EncounterID != nil {
		result["encounter_id"] = *s.EncounterID
	}
	if s.LastAccessedAt != nil {
		result["last_accessed_at"] = *s.LastAccessedAt
	}
	
	return result
}

// SnapshotRequest represents a request to create a clinical snapshot
type SnapshotRequest struct {
	PatientID       string          `json:"patient_id"`
	RecipeID        string          `json:"recipe_id"`
	ProviderID      *string         `json:"provider_id,omitempty"`
	EncounterID     *string         `json:"encounter_id,omitempty"`
	ForceRefresh    bool            `json:"force_refresh"`
	TTLHours        int32           `json:"ttl_hours"`
	SignatureMethod SignatureMethod `json:"signature_method"`
}

// Validate validates the snapshot request
func (r *SnapshotRequest) Validate() error {
	if r.PatientID == "" {
		return fmt.Errorf("patient_id is required")
	}
	if r.RecipeID == "" {
		return fmt.Errorf("recipe_id is required")
	}
	if r.TTLHours <= 0 {
		return fmt.Errorf("ttl_hours must be greater than 0")
	}
	if r.TTLHours > 168 { // 7 days maximum
		return fmt.Errorf("ttl_hours cannot exceed 168 (7 days)")
	}
	return nil
}

// SnapshotSummary provides a lightweight view of a snapshot for listing operations
type SnapshotSummary struct {
	ID                string         `json:"id"`
	PatientID         string         `json:"patient_id"`
	RecipeID          string         `json:"recipe_id"`
	Status            SnapshotStatus `json:"status"`
	CreatedAt         time.Time      `json:"created_at"`
	ExpiresAt         time.Time      `json:"expires_at"`
	CompletenessScore float64        `json:"completeness_score"`
	AccessedCount     int32          `json:"accessed_count"`
	ProviderID        *string        `json:"provider_id,omitempty"`
	EncounterID       *string        `json:"encounter_id,omitempty"`
}

// SnapshotMetrics provides metrics about snapshot usage and performance
type SnapshotMetrics struct {
	TotalSnapshots      int64                  `json:"total_snapshots"`
	ActiveSnapshots     int64                  `json:"active_snapshots"`
	ExpiredSnapshots    int64                  `json:"expired_snapshots"`
	AverageCompleteness float64                `json:"average_completeness"`
	AverageTTLHours     float64                `json:"average_ttl_hours"`
	CreationRatePerHour float64                `json:"creation_rate_per_hour"`
	AccessRatePerHour   float64                `json:"access_rate_per_hour"`
	TopRecipes          []map[string]interface{} `json:"top_recipes"`
	TopProviders        []map[string]interface{} `json:"top_providers"`
}

// ValidationResult represents the result of snapshot validation
type ValidationResult struct {
	SnapshotID             string    `json:"snapshot_id"`
	Valid                  bool      `json:"valid"`
	ChecksumValid          bool      `json:"checksum_valid"`
	SignatureValid         bool      `json:"signature_valid"`
	NotExpired             bool      `json:"not_expired"`
	Errors                 []string  `json:"errors"`
	Warnings               []string  `json:"warnings,omitempty"`
	ValidationDurationMs   float64   `json:"validation_duration_ms"`
	ValidatedAt            time.Time `json:"validated_at"`
}