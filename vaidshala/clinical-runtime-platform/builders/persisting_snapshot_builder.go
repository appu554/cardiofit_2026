// Package builders provides builder implementations for constructing
// the KnowledgeSnapshot component of ClinicalExecutionContext.
package builders

import (
	"context"
	"fmt"

	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/persistence"
)

// ============================================================================
// PERSISTING SNAPSHOT BUILDER
// ============================================================================

// PersistingKnowledgeSnapshotBuilder wraps KnowledgeSnapshotBuilder to automatically
// persist snapshots after building them.
//
// DESIGN (SaMD Compliance):
// Every clinical decision must be traceable to the exact knowledge state.
// This builder ensures snapshots are persisted for:
//   - Audit trail (regulatory requirement)
//   - Reproducibility (re-run CQL with identical inputs)
//   - Debugging (exact patient + knowledge state)
//
// USAGE:
//   builder := NewPersistingKnowledgeSnapshotBuilder(innerBuilder, repository)
//   snapshot, snapshotID, err := builder.BuildAndPersist(ctx, patient, opts)
//   // snapshotID can be used for replay: repository.GetByID(snapshotID)
type PersistingKnowledgeSnapshotBuilder struct {
	// inner builder that actually constructs the snapshot
	inner *KnowledgeSnapshotBuilder

	// repository for persisting snapshots
	repository persistence.SnapshotRepository

	// defaultRegion for persistence if not specified
	defaultRegion string
}

// NewPersistingKnowledgeSnapshotBuilder creates a builder that persists snapshots.
func NewPersistingKnowledgeSnapshotBuilder(
	inner *KnowledgeSnapshotBuilder,
	repository persistence.SnapshotRepository,
) *PersistingKnowledgeSnapshotBuilder {
	return &PersistingKnowledgeSnapshotBuilder{
		inner:         inner,
		repository:    repository,
		defaultRegion: "AU",
	}
}

// WithRegion sets the default region for persistence.
func (b *PersistingKnowledgeSnapshotBuilder) WithRegion(region string) *PersistingKnowledgeSnapshotBuilder {
	b.defaultRegion = region
	return b
}

// BuildPersistOptions configures snapshot persistence.
type BuildPersistOptions struct {
	// RequestID links snapshot to a specific request
	RequestID string

	// EncounterID links snapshot to a clinical encounter
	EncounterID string

	// Region for multi-region support (defaults to builder's defaultRegion)
	Region string

	// CreatedBy user/system identifier
	CreatedBy string

	// Persist if false, skips persistence (useful for testing)
	Persist bool
}

// DefaultBuildPersistOptions returns sensible defaults.
func DefaultBuildPersistOptions() BuildPersistOptions {
	return BuildPersistOptions{
		Persist: true,
	}
}

// BuildAndPersist builds the KnowledgeSnapshot and persists it.
// Returns the snapshot, the snapshot ID for future retrieval, and any error.
func (b *PersistingKnowledgeSnapshotBuilder) BuildAndPersist(
	ctx context.Context,
	patient *contracts.PatientContext,
	opts BuildPersistOptions,
) (*contracts.KnowledgeSnapshot, string, error) {

	// Build the snapshot using the inner builder
	snapshot, err := b.inner.Build(ctx, patient)
	if err != nil {
		return nil, "", fmt.Errorf("failed to build snapshot: %w", err)
	}

	// Skip persistence if disabled
	if !opts.Persist || b.repository == nil {
		return snapshot, "", nil
	}

	// Determine region
	region := opts.Region
	if region == "" {
		region = b.defaultRegion
	}

	// Get patient ID from context
	patientID := patient.Demographics.PatientID
	if patientID == "" {
		patientID = "unknown"
	}

	// Persist the snapshot
	saveOpts := persistence.SaveOptions{
		RequestID:   opts.RequestID,
		EncounterID: opts.EncounterID,
		Region:      region,
		CreatedBy:   opts.CreatedBy,
	}

	snapshotID, err := b.repository.Save(ctx, patientID, snapshot, saveOpts)
	if err != nil {
		// Log error but don't fail the clinical flow
		// Persistence failure should not block clinical decisions
		return snapshot, "", fmt.Errorf("snapshot built but persistence failed: %w", err)
	}

	return snapshot, snapshotID, nil
}

// Build delegates to the inner builder (for backward compatibility).
// NOTE: This does NOT persist - use BuildAndPersist for persistence.
func (b *PersistingKnowledgeSnapshotBuilder) Build(
	ctx context.Context,
	patient *contracts.PatientContext,
) (*contracts.KnowledgeSnapshot, error) {
	return b.inner.Build(ctx, patient)
}

// ============================================================================
// REPLAY SUPPORT
// ============================================================================

// ReplayFromSnapshot creates a ClinicalExecutionContext from a stored snapshot.
// This enables exact replay of clinical decisions for debugging or audit.
//
// USAGE:
//   stored, _ := repository.GetByID(snapshotID)
//   execCtx := builder.ReplayFromSnapshot(stored)
//   // Re-run CQL/engines with exact same knowledge state
func ReplayFromSnapshot(stored *persistence.StoredSnapshot) *contracts.ClinicalExecutionContext {
	if stored == nil || stored.Snapshot == nil {
		return nil
	}

	return &contracts.ClinicalExecutionContext{
		Knowledge: *stored.Snapshot,
		// PatientContext and RuntimeContext would need to be reconstructed
		// or stored separately for full replay capability
	}
}

// ============================================================================
// CONVENIENCE CONSTRUCTORS
// ============================================================================

// NewPersistingKnowledgeSnapshotBuilderFHIR creates a persisting builder using FHIR client.
// This is the RECOMMENDED constructor for production use.
func NewPersistingKnowledgeSnapshotBuilderFHIR(
	kb7FHIR KB7FHIRClient,
	kb8 KB8Client,
	kb4 KB4Client,
	kb5 KB5Client,
	kb6 KB6Client,
	kb1 KB1Client,
	kb11 KB11Client,
	kb16 KB16Client,
	config KnowledgeSnapshotConfig,
	repository persistence.SnapshotRepository,
) *PersistingKnowledgeSnapshotBuilder {

	inner := NewKnowledgeSnapshotBuilderFHIR(
		kb7FHIR, kb8, kb4, kb5, kb6, kb1, kb11, kb16, config,
	)

	return NewPersistingKnowledgeSnapshotBuilder(inner, repository)
}
