package domain

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// SnapshotStatus represents the lifecycle status of a snapshot
type SnapshotStatus string

const (
	SnapshotStatusCreated   SnapshotStatus = "created"
	SnapshotStatusActive    SnapshotStatus = "active"
	SnapshotStatusExpired   SnapshotStatus = "expired"
	SnapshotStatusArchived  SnapshotStatus = "archived"
	SnapshotStatusCorrupted SnapshotStatus = "corrupted"
)

// WorkflowPhase represents the phase in which a snapshot was created
type WorkflowPhase string

const (
	WorkflowPhaseCalculate WorkflowPhase = "calculate"
	WorkflowPhaseValidate  WorkflowPhase = "validate"
	WorkflowPhaseCommit    WorkflowPhase = "commit"
	WorkflowPhaseOverride  WorkflowPhase = "override"
)

// SnapshotReference represents an immutable reference to clinical data
type SnapshotReference struct {
	SnapshotID     string                 `json:"snapshot_id" db:"snapshot_id"`
	Checksum       string                 `json:"checksum" db:"checksum"`
	CreatedAt      time.Time              `json:"created_at" db:"created_at"`
	ExpiresAt      time.Time              `json:"expires_at" db:"expires_at"`
	Status         SnapshotStatus         `json:"status" db:"status"`
	PhaseCreated   WorkflowPhase          `json:"phase_created" db:"phase_created"`
	PatientID      string                 `json:"patient_id" db:"patient_id"`
	ContextVersion string                 `json:"context_version" db:"context_version"`
	Metadata       map[string]interface{} `json:"metadata" db:"metadata"`
	Data           map[string]interface{} `json:"data,omitempty" db:"data"`
}

// NewSnapshotReference creates a new snapshot reference
func NewSnapshotReference(
	patientID string,
	phaseCreated WorkflowPhase,
	contextVersion string,
	expiresAt time.Time,
	data map[string]interface{},
) (*SnapshotReference, error) {
	snapshotID := fmt.Sprintf("snap_%s", uuid.New().String())
	checksum, err := calculateChecksum(data)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	return &SnapshotReference{
		SnapshotID:     snapshotID,
		Checksum:       checksum,
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      expiresAt,
		Status:         SnapshotStatusCreated,
		PhaseCreated:   phaseCreated,
		PatientID:      patientID,
		ContextVersion: contextVersion,
		Metadata:       make(map[string]interface{}),
		Data:           data,
	}, nil
}

// IsValid checks if the snapshot is still valid and not expired
func (s *SnapshotReference) IsValid() bool {
	now := time.Now().UTC()
	return s.Status == SnapshotStatusActive && s.ExpiresAt.After(now)
}

// ValidateIntegrity validates the snapshot data integrity using checksum
func (s *SnapshotReference) ValidateIntegrity(data map[string]interface{}) bool {
	calculatedChecksum, err := calculateChecksum(data)
	if err != nil {
		return false
	}
	return calculatedChecksum == s.Checksum
}

// Activate sets the snapshot status to active
func (s *SnapshotReference) Activate() {
	s.Status = SnapshotStatusActive
}

// Expire sets the snapshot status to expired
func (s *SnapshotReference) Expire() {
	s.Status = SnapshotStatusExpired
}

// Archive sets the snapshot status to archived
func (s *SnapshotReference) Archive() {
	s.Status = SnapshotStatusArchived
}

// MarkCorrupted sets the snapshot status to corrupted
func (s *SnapshotReference) MarkCorrupted() {
	s.Status = SnapshotStatusCorrupted
}

// SnapshotChainTracker tracks snapshots across all workflow phases
type SnapshotChainTracker struct {
	WorkflowID        string             `json:"workflow_id" db:"workflow_id"`
	CalculateSnapshot *SnapshotReference `json:"calculate_snapshot,omitempty" db:"calculate_snapshot"`
	ValidateSnapshot  *SnapshotReference `json:"validate_snapshot,omitempty" db:"validate_snapshot"`
	CommitSnapshot    *SnapshotReference `json:"commit_snapshot,omitempty" db:"commit_snapshot"`
	OverrideSnapshot  *SnapshotReference `json:"override_snapshot,omitempty" db:"override_snapshot"`
	ChainCreatedAt    time.Time          `json:"chain_created_at" db:"chain_created_at"`
}

// NewSnapshotChainTracker creates a new snapshot chain tracker
func NewSnapshotChainTracker(workflowID string) *SnapshotChainTracker {
	return &SnapshotChainTracker{
		WorkflowID:     workflowID,
		ChainCreatedAt: time.Now().UTC(),
	}
}

// AddPhaseSnapshot adds a snapshot reference for a specific workflow phase
func (s *SnapshotChainTracker) AddPhaseSnapshot(phase WorkflowPhase, snapshot *SnapshotReference) {
	switch phase {
	case WorkflowPhaseCalculate:
		s.CalculateSnapshot = snapshot
	case WorkflowPhaseValidate:
		s.ValidateSnapshot = snapshot
	case WorkflowPhaseCommit:
		s.CommitSnapshot = snapshot
	case WorkflowPhaseOverride:
		s.OverrideSnapshot = snapshot
	}
}

// ValidateChainConsistency validates that all snapshots in the chain are consistent
func (s *SnapshotChainTracker) ValidateChainConsistency() bool {
	var snapshots []*SnapshotReference

	if s.CalculateSnapshot != nil {
		snapshots = append(snapshots, s.CalculateSnapshot)
	}
	if s.ValidateSnapshot != nil {
		snapshots = append(snapshots, s.ValidateSnapshot)
	}
	if s.CommitSnapshot != nil {
		snapshots = append(snapshots, s.CommitSnapshot)
	}

	if len(snapshots) < 2 {
		return true // Single or no snapshots are consistent by definition
	}

	// Check that all snapshots have the same patient_id and context_version
	baseSnapshot := snapshots[0]
	for _, snapshot := range snapshots[1:] {
		if snapshot.PatientID != baseSnapshot.PatientID ||
			snapshot.ContextVersion != baseSnapshot.ContextVersion {
			return false
		}
	}

	return true
}

// GetPrimarySnapshot returns the primary snapshot used for this workflow
func (s *SnapshotChainTracker) GetPrimarySnapshot() *SnapshotReference {
	if s.CalculateSnapshot != nil {
		return s.CalculateSnapshot
	}
	if s.ValidateSnapshot != nil {
		return s.ValidateSnapshot
	}
	if s.CommitSnapshot != nil {
		return s.CommitSnapshot
	}
	return s.OverrideSnapshot
}

// RecipeReference represents a reference to a clinical recipe
type RecipeReference struct {
	RecipeID         string                 `json:"recipe_id" db:"recipe_id"`
	Version          string                 `json:"version" db:"version"`
	ResolvedAt       time.Time              `json:"resolved_at" db:"resolved_at"`
	ResolutionSource string                 `json:"resolution_source" db:"resolution_source"` // "cache", "service", "fallback"
	Metadata         map[string]interface{} `json:"metadata" db:"metadata"`
}

// NewRecipeReference creates a new recipe reference
func NewRecipeReference(recipeID, version, resolutionSource string) *RecipeReference {
	return &RecipeReference{
		RecipeID:         recipeID,
		Version:          version,
		ResolvedAt:       time.Now().UTC(),
		ResolutionSource: resolutionSource,
		Metadata:         make(map[string]interface{}),
	}
}

// EvidenceEnvelope contains clinical evidence generated during workflow execution
type EvidenceEnvelope struct {
	EvidenceID      string                 `json:"evidence_id" db:"evidence_id"`
	SnapshotID      string                 `json:"snapshot_id" db:"snapshot_id"`
	Phase           WorkflowPhase          `json:"phase" db:"phase"`
	EvidenceType    string                 `json:"evidence_type" db:"evidence_type"` // "clinical_reasoning", "safety_assessment", "decision_support"
	Content         map[string]interface{} `json:"content" db:"content"`
	ConfidenceScore float64                `json:"confidence_score" db:"confidence_score"`
	GeneratedAt     time.Time              `json:"generated_at" db:"generated_at"`
	Source          string                 `json:"source" db:"source"` // "flow2_engine", "safety_gateway", "clinical_rules"
}

// NewEvidenceEnvelope creates a new evidence envelope
func NewEvidenceEnvelope(
	snapshotID string,
	phase WorkflowPhase,
	evidenceType string,
	content map[string]interface{},
	confidenceScore float64,
	source string,
) *EvidenceEnvelope {
	return &EvidenceEnvelope{
		EvidenceID:      fmt.Sprintf("evidence_%s", uuid.New().String()),
		SnapshotID:      snapshotID,
		Phase:           phase,
		EvidenceType:    evidenceType,
		Content:         content,
		ConfidenceScore: confidenceScore,
		GeneratedAt:     time.Now().UTC(),
		Source:          source,
	}
}

// ClinicalOverride represents a provider override decision
type ClinicalOverride struct {
	OverrideID        string                 `json:"override_id" db:"override_id"`
	WorkflowID        string                 `json:"workflow_id" db:"workflow_id"`
	SnapshotID        string                 `json:"snapshot_id" db:"snapshot_id"`
	OverrideType      string                 `json:"override_type" db:"override_type"` // "warning_override", "safety_override", "protocol_override"
	OriginalVerdict   string                 `json:"original_verdict" db:"original_verdict"`
	OverriddenTo      string                 `json:"overridden_to" db:"overridden_to"`
	ClinicianID       string                 `json:"clinician_id" db:"clinician_id"`
	Justification     string                 `json:"justification" db:"justification"`
	OverrideTokens    []string               `json:"override_tokens" db:"override_tokens"`
	OverrideTimestamp time.Time              `json:"override_timestamp" db:"override_timestamp"`
	PatientContext    map[string]interface{} `json:"patient_context" db:"patient_context"`
}

// NewClinicalOverride creates a new clinical override
func NewClinicalOverride(
	workflowID, snapshotID, overrideType, originalVerdict, overriddenTo, clinicianID, justification string,
	overrideTokens []string,
	patientContext map[string]interface{},
) *ClinicalOverride {
	return &ClinicalOverride{
		OverrideID:        fmt.Sprintf("override_%s", uuid.New().String()),
		WorkflowID:        workflowID,
		SnapshotID:        snapshotID,
		OverrideType:      overrideType,
		OriginalVerdict:   originalVerdict,
		OverriddenTo:      overriddenTo,
		ClinicianID:       clinicianID,
		Justification:     justification,
		OverrideTokens:    overrideTokens,
		OverrideTimestamp: time.Now().UTC(),
		PatientContext:    patientContext,
	}
}

// ProposalWithSnapshot represents an enhanced proposal response with snapshot metadata
type ProposalWithSnapshot struct {
	ProposalSetID     string                 `json:"proposal_set_id"`
	SnapshotReference *SnapshotReference     `json:"snapshot_reference"`
	RankedProposals   []map[string]interface{} `json:"ranked_proposals"`
	ClinicalEvidence  map[string]interface{} `json:"clinical_evidence"`
	MonitoringPlan    map[string]interface{} `json:"monitoring_plan"`
	RecipeReference   *RecipeReference       `json:"recipe_reference,omitempty"`
	ExecutionMetrics  map[string]interface{} `json:"execution_metrics"`
}

// ValidationResult represents an enhanced validation result with snapshot consistency
type ValidationResult struct {
	ValidationID      string                 `json:"validation_id"`
	SnapshotReference *SnapshotReference     `json:"snapshot_reference"`
	Verdict           string                 `json:"verdict"` // "SAFE", "WARNING", "UNSAFE"
	Findings          []map[string]interface{} `json:"findings"`
	EvidenceEnvelope  *EvidenceEnvelope      `json:"evidence_envelope"`
	OverrideTokens    []string               `json:"override_tokens,omitempty"`
	ApprovalRequirements map[string]interface{} `json:"approval_requirements,omitempty"`
	ValidationMetrics map[string]interface{} `json:"validation_metrics"`
}

// CommitResult represents an enhanced commit result with snapshot audit trail
type CommitResult struct {
	MedicationOrderID string                 `json:"medication_order_id"`
	SnapshotReference *SnapshotReference     `json:"snapshot_reference"`
	AuditTrailID      string                 `json:"audit_trail_id"`
	PersistenceStatus string                 `json:"persistence_status"`
	EventPublicationStatus string           `json:"event_publication_status"`
	SnapshotChain     *SnapshotChainTracker  `json:"snapshot_chain"`
	CommitTimestamp   time.Time              `json:"commit_timestamp"`
}

// calculateChecksum calculates a SHA256 checksum for the given data
func calculateChecksum(data map[string]interface{}) (string, error) {
	// Convert to JSON for consistent hashing
	jsonData, err := json.Marshal(data)
	if err != nil {
		return "", fmt.Errorf("failed to marshal data: %w", err)
	}

	// Calculate SHA256 hash
	hash := sha256.Sum256(jsonData)
	return fmt.Sprintf("%x", hash), nil
}

// SnapshotError represents snapshot-related errors
type SnapshotError struct {
	Type    string
	Message string
	Details map[string]interface{}
}

func (e *SnapshotError) Error() string {
	return fmt.Sprintf("%s: %s", e.Type, e.Message)
}

// NewSnapshotExpiredError creates a snapshot expired error
func NewSnapshotExpiredError(snapshotID string, expiredAt time.Time) *SnapshotError {
	return &SnapshotError{
		Type:    "SnapshotExpiredError",
		Message: fmt.Sprintf("Snapshot %s expired at %s", snapshotID, expiredAt.Format(time.RFC3339)),
		Details: map[string]interface{}{
			"snapshot_id": snapshotID,
			"expired_at":  expiredAt,
		},
	}
}

// NewSnapshotIntegrityError creates a snapshot integrity error
func NewSnapshotIntegrityError(snapshotID, expectedChecksum, actualChecksum string) *SnapshotError {
	return &SnapshotError{
		Type: "SnapshotIntegrityError",
		Message: fmt.Sprintf("Snapshot %s integrity check failed: expected %s, got %s",
			snapshotID, expectedChecksum, actualChecksum),
		Details: map[string]interface{}{
			"snapshot_id":       snapshotID,
			"expected_checksum": expectedChecksum,
			"actual_checksum":   actualChecksum,
		},
	}
}

// NewSnapshotNotFoundError creates a snapshot not found error
func NewSnapshotNotFoundError(snapshotID string) *SnapshotError {
	return &SnapshotError{
		Type:    "SnapshotNotFoundError",
		Message: fmt.Sprintf("Snapshot %s not found", snapshotID),
		Details: map[string]interface{}{
			"snapshot_id": snapshotID,
		},
	}
}

// NewSnapshotConsistencyError creates a snapshot consistency error
func NewSnapshotConsistencyError(message string, snapshotChain *SnapshotChainTracker) *SnapshotError {
	details := map[string]interface{}{
		"message": message,
	}
	if snapshotChain != nil {
		details["workflow_id"] = snapshotChain.WorkflowID
		details["chain_created_at"] = snapshotChain.ChainCreatedAt
	}

	return &SnapshotError{
		Type:    "SnapshotConsistencyError",
		Message: message,
		Details: details,
	}
}