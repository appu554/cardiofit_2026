// Package evidence provides Evidence Envelope management for FDA SaMD compliance.
// The Evidence Envelope captures the complete audit trail of medication decisions.
package evidence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
)

// EvidenceEnvelope captures the complete audit trail for a medication decision
// Required for FDA SaMD Class IIa compliance
type EvidenceEnvelope struct {
	ID              uuid.UUID              `json:"id"`
	SnapshotID      uuid.UUID              `json:"snapshot_id"`
	PatientID       uuid.UUID              `json:"patient_id"`
	ProviderID      string                 `json:"provider_id"`
	SessionID       string                 `json:"session_id"`
	Environment     string                 `json:"environment"`

	// Version tracking for KB services used
	KBVersions      map[string]string      `json:"kb_versions"`
	VersionSetName  string                 `json:"version_set_name"`
	ActivatedAt     time.Time              `json:"activated_at"`

	// Inference chain for explainability
	InferenceChain  []InferenceStep        `json:"inference_chain"`

	// Usage tracking
	UsedVersions    map[string]VersionUsage `json:"used_versions"`

	// Decision trail
	Decisions       []Decision             `json:"decisions"`
	Overrides       []Override             `json:"overrides"`

	// Integrity
	Checksum        string                 `json:"checksum"`
	ChecksumAlgo    string                 `json:"checksum_algo"`
	Finalized       bool                   `json:"finalized"`
	FinalizedAt     *time.Time             `json:"finalized_at,omitempty"`
	FinalizedBy     string                 `json:"finalized_by,omitempty"`

	// Timestamps
	CreatedAt       time.Time              `json:"created_at"`
	UpdatedAt       time.Time              `json:"updated_at"`
}

// VersionUsage tracks how each KB version was used
type VersionUsage struct {
	Version    string    `json:"version"`
	AccessedAt time.Time `json:"accessed_at"`
	QueryCount int       `json:"query_count"`
	CacheHits  int       `json:"cache_hits"`
}

// Decision represents a decision made during the medication advisory process
type Decision struct {
	ID          uuid.UUID   `json:"id"`
	Phase       string      `json:"phase"` // calculate, validate, commit
	Type        string      `json:"type"`  // include, exclude, adjust, recommend
	Target      string      `json:"target"` // medication code/name
	Reason      string      `json:"reason"`
	Confidence  float64     `json:"confidence"`
	KBSource    string      `json:"kb_source"`
	RuleID      string      `json:"rule_id,omitempty"`
	Timestamp   time.Time   `json:"timestamp"`
	Automated   bool        `json:"automated"`
}

// Override represents a provider override of system recommendation
type Override struct {
	ID              uuid.UUID   `json:"id"`
	DecisionID      uuid.UUID   `json:"decision_id"`
	OriginalAction  string      `json:"original_action"`
	OverrideAction  string      `json:"override_action"`
	Reason          string      `json:"reason"`
	ProviderID      string      `json:"provider_id"`
	ProviderRole    string      `json:"provider_role"`
	Acknowledged    bool        `json:"acknowledged"`
	AcknowledgedAt  *time.Time  `json:"acknowledged_at,omitempty"`
	Timestamp       time.Time   `json:"timestamp"`
}

// EvidenceEnvelopeManager manages Evidence Envelopes for medication decisions
type EvidenceEnvelopeManager struct {
	store            EnvelopeStore
	environment      string
	checksumAlgo     string

	// Active envelopes by session
	activeEnvelopes  map[string]*EvidenceEnvelope

	// Concurrency control
	mutex            sync.RWMutex

	// Metrics
	metrics          *EnvelopeMetrics
}

// EnvelopeStore interface for envelope persistence
type EnvelopeStore interface {
	Save(ctx context.Context, envelope *EvidenceEnvelope) error
	Get(ctx context.Context, id string) (*EvidenceEnvelope, error)
	GetBySnapshot(ctx context.Context, snapshotID string) (*EvidenceEnvelope, error)
	List(ctx context.Context, patientID string, limit int) ([]*EvidenceEnvelope, error)
}

// EnvelopeMetrics tracks envelope usage and performance
type EnvelopeMetrics struct {
	EnvelopesCreated   int64
	EnvelopesFinalized int64
	OverridesRecorded  int64
	AverageChainLength float64
}

// NewEvidenceEnvelopeManager creates a new Evidence Envelope manager
func NewEvidenceEnvelopeManager(store EnvelopeStore, environment string) *EvidenceEnvelopeManager {
	return &EvidenceEnvelopeManager{
		store:           store,
		environment:     environment,
		checksumAlgo:    "sha256",
		activeEnvelopes: make(map[string]*EvidenceEnvelope),
		metrics:         &EnvelopeMetrics{},
	}
}

// CreateEnvelope creates a new Evidence Envelope for a medication decision
func (eem *EvidenceEnvelopeManager) CreateEnvelope(
	ctx context.Context,
	snapshotID uuid.UUID,
	patientID uuid.UUID,
	providerID string,
	sessionID string,
	kbVersions map[string]string,
) (*EvidenceEnvelope, error) {

	now := time.Now()

	envelope := &EvidenceEnvelope{
		ID:             uuid.New(),
		SnapshotID:     snapshotID,
		PatientID:      patientID,
		ProviderID:     providerID,
		SessionID:      sessionID,
		Environment:    eem.environment,
		KBVersions:     kbVersions,
		VersionSetName: fmt.Sprintf("%s-%d", eem.environment, now.Unix()),
		ActivatedAt:    now,
		InferenceChain: []InferenceStep{},
		UsedVersions:   make(map[string]VersionUsage),
		Decisions:      []Decision{},
		Overrides:      []Override{},
		ChecksumAlgo:   eem.checksumAlgo,
		Finalized:      false,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	// Save to store
	if err := eem.store.Save(ctx, envelope); err != nil {
		return nil, fmt.Errorf("failed to save envelope: %w", err)
	}

	// Track active envelope
	eem.mutex.Lock()
	eem.activeEnvelopes[sessionID] = envelope
	eem.metrics.EnvelopesCreated++
	eem.mutex.Unlock()

	return envelope, nil
}

// AddInferenceStep adds an inference step to the envelope's chain
func (eem *EvidenceEnvelopeManager) AddInferenceStep(
	ctx context.Context,
	envelopeID uuid.UUID,
	step InferenceStep,
) error {

	eem.mutex.Lock()
	defer eem.mutex.Unlock()

	// Find envelope
	var envelope *EvidenceEnvelope
	for _, env := range eem.activeEnvelopes {
		if env.ID == envelopeID {
			envelope = env
			break
		}
	}

	if envelope == nil {
		return fmt.Errorf("envelope not found: %s", envelopeID)
	}

	if envelope.Finalized {
		return fmt.Errorf("cannot modify finalized envelope")
	}

	step.StepNumber = len(envelope.InferenceChain) + 1
	step.Timestamp = time.Now()
	envelope.InferenceChain = append(envelope.InferenceChain, step)
	envelope.UpdatedAt = time.Now()

	return nil
}

// RecordDecision records a decision in the envelope
func (eem *EvidenceEnvelopeManager) RecordDecision(
	ctx context.Context,
	envelopeID uuid.UUID,
	decision Decision,
) error {

	eem.mutex.Lock()
	defer eem.mutex.Unlock()

	var envelope *EvidenceEnvelope
	for _, env := range eem.activeEnvelopes {
		if env.ID == envelopeID {
			envelope = env
			break
		}
	}

	if envelope == nil {
		return fmt.Errorf("envelope not found: %s", envelopeID)
	}

	if envelope.Finalized {
		return fmt.Errorf("cannot modify finalized envelope")
	}

	decision.ID = uuid.New()
	decision.Timestamp = time.Now()
	envelope.Decisions = append(envelope.Decisions, decision)
	envelope.UpdatedAt = time.Now()

	return nil
}

// RecordOverride records a provider override
func (eem *EvidenceEnvelopeManager) RecordOverride(
	ctx context.Context,
	envelopeID uuid.UUID,
	override Override,
) error {

	eem.mutex.Lock()
	defer eem.mutex.Unlock()

	var envelope *EvidenceEnvelope
	for _, env := range eem.activeEnvelopes {
		if env.ID == envelopeID {
			envelope = env
			break
		}
	}

	if envelope == nil {
		return fmt.Errorf("envelope not found: %s", envelopeID)
	}

	if envelope.Finalized {
		return fmt.Errorf("cannot modify finalized envelope")
	}

	override.ID = uuid.New()
	override.Timestamp = time.Now()
	envelope.Overrides = append(envelope.Overrides, override)
	envelope.UpdatedAt = time.Now()

	eem.metrics.OverridesRecorded++

	return nil
}

// RecordKBUsage records usage of a KB service
func (eem *EvidenceEnvelopeManager) RecordKBUsage(
	ctx context.Context,
	envelopeID uuid.UUID,
	kbName string,
	cacheHit bool,
) error {

	eem.mutex.Lock()
	defer eem.mutex.Unlock()

	var envelope *EvidenceEnvelope
	for _, env := range eem.activeEnvelopes {
		if env.ID == envelopeID {
			envelope = env
			break
		}
	}

	if envelope == nil {
		return fmt.Errorf("envelope not found: %s", envelopeID)
	}

	version, ok := envelope.KBVersions[kbName]
	if !ok {
		version = "unknown"
	}

	usage := envelope.UsedVersions[kbName]
	usage.Version = version
	usage.AccessedAt = time.Now()
	usage.QueryCount++
	if cacheHit {
		usage.CacheHits++
	}
	envelope.UsedVersions[kbName] = usage

	return nil
}

// Finalize finalizes the envelope with checksum (called at Commit phase)
func (eem *EvidenceEnvelopeManager) Finalize(
	ctx context.Context,
	envelopeID uuid.UUID,
	finalizedBy string,
) error {

	eem.mutex.Lock()
	defer eem.mutex.Unlock()

	var envelope *EvidenceEnvelope
	var sessionKey string
	for key, env := range eem.activeEnvelopes {
		if env.ID == envelopeID {
			envelope = env
			sessionKey = key
			break
		}
	}

	if envelope == nil {
		return fmt.Errorf("envelope not found: %s", envelopeID)
	}

	if envelope.Finalized {
		return fmt.Errorf("envelope already finalized")
	}

	// Compute checksum
	checksum, err := eem.computeChecksum(envelope)
	if err != nil {
		return fmt.Errorf("failed to compute checksum: %w", err)
	}

	now := time.Now()
	envelope.Checksum = checksum
	envelope.Finalized = true
	envelope.FinalizedAt = &now
	envelope.FinalizedBy = finalizedBy
	envelope.UpdatedAt = now

	// Save to store
	if err := eem.store.Save(ctx, envelope); err != nil {
		return fmt.Errorf("failed to save finalized envelope: %w", err)
	}

	// Remove from active
	delete(eem.activeEnvelopes, sessionKey)
	eem.metrics.EnvelopesFinalized++

	return nil
}

// GetEnvelope retrieves an envelope by ID
func (eem *EvidenceEnvelopeManager) GetEnvelope(ctx context.Context, id string) (*EvidenceEnvelope, error) {
	return eem.store.Get(ctx, id)
}

// GetEnvelopeBySnapshot retrieves an envelope by snapshot ID
func (eem *EvidenceEnvelopeManager) GetEnvelopeBySnapshot(ctx context.Context, snapshotID string) (*EvidenceEnvelope, error) {
	return eem.store.GetBySnapshot(ctx, snapshotID)
}

// VerifyIntegrity verifies the envelope's checksum
func (eem *EvidenceEnvelopeManager) VerifyIntegrity(ctx context.Context, id string) (bool, error) {
	envelope, err := eem.store.Get(ctx, id)
	if err != nil {
		return false, err
	}

	if !envelope.Finalized {
		return false, fmt.Errorf("envelope not finalized")
	}

	checksum, err := eem.computeChecksum(envelope)
	if err != nil {
		return false, err
	}

	return checksum == envelope.Checksum, nil
}

// computeChecksum computes SHA256 checksum of envelope data
func (eem *EvidenceEnvelopeManager) computeChecksum(envelope *EvidenceEnvelope) (string, error) {
	// Create data structure for hashing (excluding checksum fields)
	data := struct {
		ID             uuid.UUID        `json:"id"`
		SnapshotID     uuid.UUID        `json:"snapshot_id"`
		PatientID      uuid.UUID        `json:"patient_id"`
		KBVersions     map[string]string `json:"kb_versions"`
		InferenceChain []InferenceStep  `json:"inference_chain"`
		Decisions      []Decision       `json:"decisions"`
		Overrides      []Override       `json:"overrides"`
	}{
		ID:             envelope.ID,
		SnapshotID:     envelope.SnapshotID,
		PatientID:      envelope.PatientID,
		KBVersions:     envelope.KBVersions,
		InferenceChain: envelope.InferenceChain,
		Decisions:      envelope.Decisions,
		Overrides:      envelope.Overrides,
	}

	jsonBytes, err := json.Marshal(data)
	if err != nil {
		return "", err
	}

	hash := sha256.Sum256(jsonBytes)
	return hex.EncodeToString(hash[:]), nil
}

// GetMetrics returns envelope manager metrics
func (eem *EvidenceEnvelopeManager) GetMetrics() *EnvelopeMetrics {
	eem.mutex.RLock()
	defer eem.mutex.RUnlock()

	metrics := *eem.metrics

	// Calculate average chain length
	totalLength := 0
	count := 0
	for _, env := range eem.activeEnvelopes {
		totalLength += len(env.InferenceChain)
		count++
	}
	if count > 0 {
		metrics.AverageChainLength = float64(totalLength) / float64(count)
	}

	return &metrics
}
