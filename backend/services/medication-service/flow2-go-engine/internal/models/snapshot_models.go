package models

import (
	"time"
)

// SnapshotRequest represents a request to create a clinical snapshot
type SnapshotRequest struct {
	PatientID       string `json:"patient_id" binding:"required"`
	RecipeID        string `json:"recipe_id" binding:"required"`
	ProviderID      string `json:"provider_id,omitempty"`
	EncounterID     string `json:"encounter_id,omitempty"`
	TTLHours        int    `json:"ttl_hours" binding:"min=1,max=24"`
	ForceRefresh    bool   `json:"force_refresh"`
	SignatureMethod string `json:"signature_method,omitempty"`
}

// ClinicalSnapshot represents an immutable clinical snapshot
type ClinicalSnapshot struct {
	ID                string                 `json:"id"`
	PatientID         string                 `json:"patient_id"`
	RecipeID          string                 `json:"recipe_id"`
	ContextID         string                 `json:"context_id"`
	Data              map[string]interface{} `json:"data"`
	CompletenessScore float64                `json:"completeness_score"`
	Checksum          string                 `json:"checksum"`
	Signature         string                 `json:"signature"`
	SignatureMethod   string                 `json:"signature_method"`
	Status            string                 `json:"status"`
	CreatedAt         time.Time              `json:"created_at"`
	ExpiresAt         time.Time              `json:"expires_at"`
	AccessedCount     int                    `json:"accessed_count"`
	LastAccessedAt    *time.Time             `json:"last_accessed_at,omitempty"`
	ProviderID        string                 `json:"provider_id,omitempty"`
	EncounterID       string                 `json:"encounter_id,omitempty"`
	AssemblyMetadata  map[string]interface{} `json:"assembly_metadata"`
	EvidenceEnvelope  map[string]interface{} `json:"evidence_envelope"`
}

// IsExpired checks if the snapshot has expired
func (cs *ClinicalSnapshot) IsExpired() bool {
	return time.Now().After(cs.ExpiresAt)
}

// IsValid checks if the snapshot is valid (not expired and active)
func (cs *ClinicalSnapshot) IsValid() bool {
	return cs.Status == "active" && !cs.IsExpired()
}

// SnapshotValidationResult represents the result of snapshot validation
type SnapshotValidationResult struct {
	SnapshotID            string    `json:"snapshot_id"`
	Valid                 bool      `json:"valid"`
	ChecksumValid         bool      `json:"checksum_valid"`
	SignatureValid        bool      `json:"signature_valid"`
	NotExpired           bool      `json:"not_expired"`
	Errors               []string  `json:"errors"`
	Warnings             []string  `json:"warnings"`
	ValidatedAt          time.Time `json:"validated_at"`
	ValidationDurationMs float64   `json:"validation_duration_ms"`
}

// SnapshotBasedFlow2Request represents a snapshot-based Flow2 execution request
type SnapshotBasedFlow2Request struct {
	// Option 1: Use existing snapshot
	SnapshotID string `json:"snapshot_id,omitempty"`

	// Option 2: Create new snapshot
	PatientID         string   `json:"patient_id" binding:"required"`
	RecipeID          string   `json:"recipe_id,omitempty"`
	MedicationCode    string   `json:"medication_code" binding:"required"`
	MedicationName    string   `json:"medication_name,omitempty"`
	Indication        string   `json:"indication,omitempty"`
	PatientConditions []string `json:"patient_conditions,omitempty"`
	Priority          string   `json:"priority,omitempty"`

	// Snapshot creation options (when creating new snapshot)
	ProviderID   string `json:"provider_id,omitempty"`
	EncounterID  string `json:"encounter_id,omitempty"`
	TTLHours     int    `json:"ttl_hours,omitempty"`
	ForceRefresh bool   `json:"force_refresh,omitempty"`

	// Processing options
	ProcessingHints map[string]interface{} `json:"processing_hints,omitempty"`
}

// SnapshotBasedFlow2Response represents the response from snapshot-based execution
type SnapshotBasedFlow2Response struct {
	RequestID      string                        `json:"request_id"`
	PatientID      string                        `json:"patient_id"`
	SnapshotInfo   *SnapshotInfo                 `json:"snapshot_info"`
	IntentManifest *IntentManifestResponse       `json:"intent_manifest"`
	MedicationProposal *MedicationProposal       `json:"medication_proposal"`
	EvidenceEnvelope *SnapshotEvidenceEnvelope   `json:"evidence_envelope"`
	OverallStatus    string                      `json:"overall_status"`
	PerformanceMetrics *SnapshotPerformanceMetrics `json:"performance_metrics"`
	Timestamp        time.Time                   `json:"timestamp"`
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

// SnapshotEvidenceEnvelope contains comprehensive evidence for audit trails
type SnapshotEvidenceEnvelope struct {
	SnapshotEvidence   map[string]interface{} `json:"snapshot_evidence"`
	ProcessingEvidence map[string]interface{} `json:"processing_evidence"`
	AuditTrail         SnapshotAuditTrail     `json:"audit_trail"`
}

// SnapshotAuditTrail contains detailed audit information
type SnapshotAuditTrail struct {
	SnapshotCreated time.Time `json:"snapshot_created"`
	WorkflowExecuted time.Time `json:"workflow_executed"`
	DataSources     []string  `json:"data_sources"`
	IntegrityChecks []string  `json:"integrity_checks"`
	ProcessingSteps []string  `json:"processing_steps"`
}

// SnapshotPerformanceMetrics contains performance metrics for snapshot-based execution
type SnapshotPerformanceMetrics struct {
	TotalExecutionTimeMs    int64   `json:"total_execution_time_ms"`
	SnapshotRetrievalTimeMs int64   `json:"snapshot_retrieval_time_ms"`
	ORBEvaluationTimeMs     int64   `json:"orb_evaluation_time_ms"`
	RustExecutionTimeMs     int64   `json:"rust_execution_time_ms"`
	NetworkHops             int     `json:"network_hops"`
	ArchitectureType        string  `json:"architecture_type"`
	DataFreshness          float64 `json:"data_freshness_minutes"`
	IntegrityVerified      bool    `json:"integrity_verified"`
}

// AdvancedSnapshotRequest represents an advanced snapshot workflow request
type AdvancedSnapshotRequest struct {
	PatientID         string   `json:"patient_id" binding:"required"`
	MedicationCode    string   `json:"medication_code" binding:"required"`
	MedicationName    string   `json:"medication_name,omitempty"`
	Indication        string   `json:"indication,omitempty"`
	PatientConditions []string `json:"patient_conditions,omitempty"`
	Priority          string   `json:"priority,omitempty"`
	ProviderID        string   `json:"provider_id,omitempty"`
	EncounterID       string   `json:"encounter_id,omitempty"`
	TTLHours          int      `json:"ttl_hours,omitempty"`
	ForceRefresh      bool     `json:"force_refresh,omitempty"`
}

// BatchSnapshotRequest represents a batch snapshot execution request
type BatchSnapshotRequest struct {
	Requests []SnapshotBasedFlow2Request `json:"requests" binding:"required,min=1,max=10"`
}

// BatchSnapshotResponse represents the response from batch snapshot execution
type BatchSnapshotResponse struct {
	BatchID             string                        `json:"batch_id"`
	TotalRequests       int                           `json:"total_requests"`
	SuccessCount        int                           `json:"success_count"`
	FailureCount        int                           `json:"failure_count"`
	SuccessfulResults   []SnapshotBasedFlow2Response  `json:"successful_results"`
	FailedResults       []BatchFailureResult          `json:"failed_results"`
	StartedAt           time.Time                     `json:"started_at"`
	CompletedAt         time.Time                     `json:"completed_at"`
	TotalExecutionTimeMs int64                        `json:"total_execution_time_ms"`
}

// BatchFailureResult represents a failed request in batch processing
type BatchFailureResult struct {
	Index   int                       `json:"index"`
	Request SnapshotBasedFlow2Request `json:"request"`
	Error   string                    `json:"error"`
}

// SnapshotBasedRustRequest represents a request to Rust engine with snapshot data
type SnapshotBasedRustRequest struct {
	RequestID       string                 `json:"request_id"`
	SnapshotID      string                 `json:"snapshot_id"`
	RecipeID        string                 `json:"recipe_id"`
	MedicationCode  string                 `json:"medication_code"`
	ClinicalData    map[string]interface{} `json:"clinical_data"`
	ProcessingHints map[string]interface{} `json:"processing_hints"`
}

// RustRecipeResponse represents the response from Rust engine recipe execution
type RustRecipeResponse struct {
	RequestID          string              `json:"request_id"`
	MedicationProposal *MedicationProposal `json:"medication_proposal"`
	SafetyStatus       string              `json:"safety_status"`
	ExecutionTimeMs    int64               `json:"execution_time_ms"`
	ProcessingEvidence map[string]interface{} `json:"processing_evidence"`
}

// SignatureMethod constants
const (
	SignatureMethodMock     = "mock"
	SignatureMethodRSA2048  = "rsa-2048"
	SignatureMethodECDSAP256 = "ecdsa-p256"
)

// Snapshot workflow constants
const (
	SnapshotWorkflowTypeBasic    = "basic"
	SnapshotWorkflowTypeAdvanced = "advanced"
	SnapshotWorkflowTypeBatch    = "batch"
)

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
	TotalSnapshots       int                            `json:"total_snapshots"`
	ActiveSnapshots      int                            `json:"active_snapshots"`
	ExpiredSnapshots     int                            `json:"expired_snapshots"`
	AverageCompleteness  float64                        `json:"average_completeness"`
	AverageTTLHours      float64                        `json:"average_ttl_hours"`
	CreationRatePerHour  float64                        `json:"creation_rate_per_hour"`
	AccessRatePerHour    float64                        `json:"access_rate_per_hour"`
	TopRecipes           []map[string]interface{}       `json:"top_recipes"`
	TopProviders         []map[string]interface{}       `json:"top_providers"`
}

// ServiceStatus represents Context Gateway service status
type ServiceStatus struct {
	Service          string                 `json:"service"`
	Status           string                 `json:"status"`
	Version          string                 `json:"version"`
	Features         []string               `json:"features"`
	Endpoints        []string               `json:"endpoints"`
	CurrentMetrics   map[string]interface{} `json:"current_metrics"`
	Timestamp        string                 `json:"timestamp"`
}

// BatchSnapshotResult represents the result of batch snapshot creation
type BatchSnapshotResult struct {
	TotalRequested int                    `json:"total_requested"`
	Successful     []map[string]interface{} `json:"successful"`
	Failed         []map[string]interface{} `json:"failed"`
	CreatedAt      string                 `json:"created_at"`
}

// CleanupResult represents the result of snapshot cleanup operation
type CleanupResult struct {
	Status       string `json:"status"`
	Message      string `json:"message"`
	DeletedCount int    `json:"deleted_count"`
	CleanedAt    string `json:"cleaned_at"`
}

// Performance improvement constants
const (
	// Target performance improvements with snapshot architecture
	TargetSnapshotRetrievalTimeMs = 5
	TargetORBEvaluationTimeMs     = 1
	TargetRustExecutionTimeMs     = 50
	TargetTotalSnapshotTimeMs     = 100 // vs 250ms traditional
	
	// Data integrity constants
	DefaultTTLHours            = 1
	MaxTTLHours               = 24
	MinCompletenessScore      = 0.7
	RequiredIntegrityChecks   = 2 // checksum + signature
)