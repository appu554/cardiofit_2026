package snapshot

import (
	"time"
)

// SnapshotRequest represents a request to create a clinical snapshot
type SnapshotRequest struct {
	PatientID       string `json:"patient_id" binding:"required"`
	RecipeID        string `json:"recipe_id" binding:"required"`
	ProviderID      string `json:"provider_id,omitempty"`
	EncounterID     string `json:"encounter_id,omitempty"`
	TTLMinutes      int    `json:"ttl_minutes" binding:"min=1,max=60"`
	ForceRefresh    bool   `json:"force_refresh"`
	SignatureMethod string `json:"signature_method,omitempty"`
}

// SnapshotInfo contains information about the snapshot used
type SnapshotInfo struct {
	SnapshotID        string    `json:"snapshot_id"`
	RecipeID          string    `json:"recipe_id"`
	CreatedAt         time.Time `json:"created_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	CompletenessScore float64   `json:"completeness_score"`
	Checksum          string    `json:"checksum"`
	AccessedCount     int       `json:"accessed_count"`
}

// SnapshotValidationResult represents the result of snapshot validation
type SnapshotValidationResult struct {
	SnapshotID           string    `json:"snapshot_id"`
	Valid                bool      `json:"valid"`
	ChecksumValid        bool      `json:"checksum_valid"`
	SignatureValid       bool      `json:"signature_valid"`
	NotExpired           bool      `json:"not_expired"`
	Errors               []string  `json:"errors"`
	Warnings             []string  `json:"warnings"`
	ValidatedAt          time.Time `json:"validated_at"`
	ValidationDurationMs float64   `json:"validation_duration_ms"`
}

// SnapshotFilters represents filtering options for listing snapshots
type SnapshotFilters struct {
	PatientID  string `json:"patient_id,omitempty"`
	ProviderID string `json:"provider_id,omitempty"`
	RecipeID   string `json:"recipe_id,omitempty"`
	Status     string `json:"status,omitempty"`
	Limit      int    `json:"limit,omitempty"`
}

// SnapshotSummary represents a summary view of a clinical snapshot
type SnapshotSummary struct {
	ID                string    `json:"id"`
	PatientID         string    `json:"patient_id"`
	RecipeID          string    `json:"recipe_id"`
	Status            string    `json:"status"`
	CreatedAt         time.Time `json:"created_at"`
	ExpiresAt         time.Time `json:"expires_at"`
	CompletenessScore float64   `json:"completeness_score"`
	AccessedCount     int       `json:"accessed_count"`
	ProviderID        string    `json:"provider_id,omitempty"`
	EncounterID       string    `json:"encounter_id,omitempty"`
}

// SnapshotMetrics represents snapshot service performance metrics
type SnapshotMetrics struct {
	TotalSnapshots      int                      `json:"total_snapshots"`
	ActiveSnapshots     int                      `json:"active_snapshots"`
	ExpiredSnapshots    int                      `json:"expired_snapshots"`
	AverageCompleteness float64                  `json:"average_completeness"`
	AverageTTLMinutes   float64                  `json:"average_ttl_minutes"`
	CreationRatePerHour float64                  `json:"creation_rate_per_hour"`
	AccessRatePerHour   float64                  `json:"access_rate_per_hour"`
	TopRecipes          []map[string]interface{} `json:"top_recipes"`
	TopProviders        []map[string]interface{} `json:"top_providers"`
}

// SignatureMethod constants
const (
	SignatureMethodMock      = "mock"
	SignatureMethodRSA2048   = "rsa-2048"
	SignatureMethodECDSAP256 = "ecdsa-p256"
	SignatureMethodSHA256    = "sha256"
)

// Performance constants aligned with FDA SaMD requirements
const (
	// Default snapshot TTL is 30 minutes per FDA guidance
	DefaultTTLMinutes = 30
	MaxTTLMinutes     = 60

	// Minimum completeness score for valid snapshot
	MinCompletenessScore = 0.7

	// Required integrity checks for FDA compliance
	RequiredIntegrityChecks = 2 // checksum + signature

	// Target performance times in milliseconds
	TargetSnapshotRetrievalTimeMs = 5
	TargetTotalSnapshotTimeMs     = 100
)

// CleanupResult represents the result of snapshot cleanup operation
type CleanupResult struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	DeletedCount int    `json:"deleted_count"`
	CleanedAt    string `json:"cleaned_at"`
}
