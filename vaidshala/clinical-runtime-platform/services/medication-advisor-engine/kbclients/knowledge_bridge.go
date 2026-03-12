// Package kbclients provides integration with the parent clinical-runtime-platform
// Knowledge Base services via the KnowledgeSnapshotBuilder pattern.
//
// ARCHITECTURE (CTO/CMO Directive):
// - Engines see ANSWERS, not questions - no KB calls at execution time
// - Snapshot is IMMUTABLE - built once, read-only thereafter
// - All pre-computed answers come from KnowledgeSnapshotBuilder
//
// This file bridges the medication-advisor-engine with the parent module's
// KnowledgeSnapshotBuilder and real HTTP clients.
package kbclients

import (
	"context"
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/builders"
	"vaidshala/clinical-runtime-platform/clients"
	"vaidshala/clinical-runtime-platform/contracts"
)

// KnowledgeBridge provides access to pre-computed KB answers via KnowledgeSnapshotBuilder.
// This is the ONLY way engines should access KB data at execution time.
type KnowledgeBridge struct {
	builder     *builders.KnowledgeSnapshotBuilder
	kbClients   *clients.KBClients
	snapshot    *contracts.KnowledgeSnapshot
	lastBuilt   time.Time
	config      KnowledgeBridgeConfig
}

// KnowledgeBridgeConfig holds configuration for the knowledge bridge.
type KnowledgeBridgeConfig struct {
	// KB service URLs (use defaults if empty)
	KB1URL string // Drug Rules (default: http://localhost:8081)
	KB4URL string // Patient Safety (default: http://localhost:8088)
	KB5URL string // Drug Interactions (default: http://localhost:8095)
	KB6URL string // Formulary (default: http://localhost:8087 HTTP, 8086 is gRPC)
	KB7URL string // Terminology (default: http://localhost:8092)
	KB8URL string // Calculator (default: http://localhost:8097)

	// Timeout for KB service calls
	Timeout time.Duration

	// SnapshotTTL is the time-to-live for cached snapshots
	SnapshotTTL time.Duration
}

// DefaultKnowledgeBridgeConfig returns default configuration for local Docker setup.
func DefaultKnowledgeBridgeConfig() KnowledgeBridgeConfig {
	return KnowledgeBridgeConfig{
		KB1URL:      "http://localhost:8081",
		KB4URL:      "http://localhost:8088",
		KB5URL:      "http://localhost:8095",
		KB6URL:      "http://localhost:8087",
		KB7URL:      "http://localhost:8092",
		KB8URL:      "http://localhost:8097",
		Timeout:     30 * time.Second,
		SnapshotTTL: 30 * time.Minute,
	}
}

// NewKnowledgeBridge creates a new knowledge bridge with real KB HTTP clients.
// This is the REQUIRED way to create KB access for the medication advisor engine.
func NewKnowledgeBridge(config KnowledgeBridgeConfig) (*KnowledgeBridge, error) {
	// Create KB client configuration
	kbConfig := clients.KBClientConfig{
		KB1BaseURL: config.KB1URL,
		KB4BaseURL: config.KB4URL,
		KB5BaseURL: config.KB5URL,
		KB6BaseURL: config.KB6URL,
		KB7BaseURL: config.KB7URL,
		KB8BaseURL: config.KB8URL,
		Timeout:    config.Timeout,
	}

	// Apply defaults for empty URLs
	if kbConfig.KB1BaseURL == "" {
		kbConfig.KB1BaseURL = "http://localhost:8081"
	}
	if kbConfig.KB4BaseURL == "" {
		kbConfig.KB4BaseURL = "http://localhost:8088"
	}
	if kbConfig.KB5BaseURL == "" {
		kbConfig.KB5BaseURL = "http://localhost:8095"
	}
	if kbConfig.KB6BaseURL == "" {
		kbConfig.KB6BaseURL = "http://localhost:8087"
	}
	if kbConfig.KB7BaseURL == "" {
		kbConfig.KB7BaseURL = "http://localhost:8092"
	}
	if kbConfig.KB8BaseURL == "" {
		kbConfig.KB8BaseURL = "http://localhost:8097"
	}
	if kbConfig.Timeout == 0 {
		kbConfig.Timeout = 30 * time.Second
	}

	// Create real KB HTTP clients from parent module
	kbClients := clients.NewKBClients(kbConfig)

	// Create KnowledgeSnapshotBuilder with real clients
	// Note: Using NewKnowledgeSnapshotBuilderFHIR for clinical execution
	builderConfig := builders.DefaultKnowledgeSnapshotConfig()
	builderConfig.Region = "AU"
	builderConfig.ParallelQueries = true
	builderConfig.QueryTimeout = 5 * time.Second

	builder := builders.NewKnowledgeSnapshotBuilderFHIR(
		kbClients.KB7,  // KB-7 FHIR Terminology
		kbClients.KB8,  // KB-8 Calculator
		kbClients.KB4,  // KB-4 Safety
		kbClients.KB5,  // KB-5 Interactions
		kbClients.KB6,  // KB-6 Formulary
		kbClients.KB1,  // KB-1 Dosing
		nil,            // KB-11 CDI (optional)
		nil,            // KB-16 Lab Interpretation (optional)
		builderConfig,
	)

	return &KnowledgeBridge{
		builder:   builder,
		kbClients: kbClients,
		config:    config,
	}, nil
}

// NewDefaultKnowledgeBridge creates a knowledge bridge with default local Docker configuration.
func NewDefaultKnowledgeBridge() (*KnowledgeBridge, error) {
	return NewKnowledgeBridge(DefaultKnowledgeBridgeConfig())
}

// BuildSnapshot builds a fresh KnowledgeSnapshot for the given patient context.
// This pre-computes ALL KB answers at build time - engines read from the snapshot.
func (kb *KnowledgeBridge) BuildSnapshot(ctx context.Context, patientCtx *contracts.PatientContext) (*contracts.KnowledgeSnapshot, error) {
	if patientCtx == nil {
		return nil, fmt.Errorf("patient context is required")
	}

	// Build the snapshot using the parent module's builder
	snapshot, err := kb.builder.Build(ctx, patientCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to build knowledge snapshot: %w", err)
	}

	// Cache the snapshot
	kb.snapshot = snapshot
	kb.lastBuilt = time.Now()

	return snapshot, nil
}

// GetSnapshot returns the current cached snapshot.
// Returns nil if no snapshot has been built or if it's expired.
func (kb *KnowledgeBridge) GetSnapshot() *contracts.KnowledgeSnapshot {
	if kb.snapshot == nil {
		return nil
	}

	// Check if snapshot is expired
	if kb.config.SnapshotTTL > 0 && time.Since(kb.lastBuilt) > kb.config.SnapshotTTL {
		return nil
	}

	return kb.snapshot
}

// GetOrBuildSnapshot returns the cached snapshot or builds a new one if needed.
func (kb *KnowledgeBridge) GetOrBuildSnapshot(ctx context.Context, patientCtx *contracts.PatientContext) (*contracts.KnowledgeSnapshot, error) {
	if snap := kb.GetSnapshot(); snap != nil {
		return snap, nil
	}
	return kb.BuildSnapshot(ctx, patientCtx)
}

// ============================================================================
// Pre-computed answer accessors
// These methods read from the snapshot - NO KB calls at execution time
// ============================================================================

// GetSafetyInfo returns pre-computed safety information from KB-4.
func (kb *KnowledgeBridge) GetSafetyInfo() *contracts.SafetySnapshot {
	if kb.snapshot == nil {
		return nil
	}
	return &kb.snapshot.Safety
}

// GetInteractions returns pre-computed drug interactions from KB-5.
func (kb *KnowledgeBridge) GetInteractions() *contracts.InteractionSnapshot {
	if kb.snapshot == nil {
		return nil
	}
	return &kb.snapshot.Interactions
}

// GetFormularyInfo returns pre-computed formulary status from KB-6.
func (kb *KnowledgeBridge) GetFormularyInfo() *contracts.FormularySnapshot {
	if kb.snapshot == nil {
		return nil
	}
	return &kb.snapshot.Formulary
}

// GetDosingInfo returns pre-computed dosing adjustments from KB-1.
func (kb *KnowledgeBridge) GetDosingInfo() *contracts.DosingSnapshot {
	if kb.snapshot == nil {
		return nil
	}
	return &kb.snapshot.Dosing
}

// GetTerminology returns pre-computed terminology from KB-7.
func (kb *KnowledgeBridge) GetTerminology() *contracts.TerminologySnapshot {
	if kb.snapshot == nil {
		return nil
	}
	return &kb.snapshot.Terminology
}

// GetCalculators returns pre-computed clinical calculations from KB-8.
func (kb *KnowledgeBridge) GetCalculators() *contracts.CalculatorSnapshot {
	if kb.snapshot == nil {
		return nil
	}
	return &kb.snapshot.Calculators
}

// ============================================================================
// Convenience methods for common lookups
// ============================================================================

// HasContraindication checks if there's a contraindication for the given medication.
func (kb *KnowledgeBridge) HasContraindication(rxnormCode string) bool {
	safety := kb.GetSafetyInfo()
	if safety == nil {
		return false
	}
	for _, ci := range safety.Contraindications {
		if ci.Medication.Code == rxnormCode {
			return true
		}
	}
	return false
}

// HasDrugInteraction checks if adding a drug would cause an interaction.
func (kb *KnowledgeBridge) HasDrugInteraction(rxnormCode string) bool {
	interactions := kb.GetInteractions()
	if interactions == nil {
		return false
	}
	for _, ddi := range interactions.PotentialDDIs {
		if ddi.Drug1.Code == rxnormCode || ddi.Drug2.Code == rxnormCode {
			return true
		}
	}
	return false
}

// GetDrugInteractionSeverity returns the maximum interaction severity for a drug.
func (kb *KnowledgeBridge) GetDrugInteractionSeverity(rxnormCode string) string {
	interactions := kb.GetInteractions()
	if interactions == nil {
		return "none"
	}
	maxSeverity := "none"
	for _, ddi := range interactions.PotentialDDIs {
		if ddi.Drug1.Code == rxnormCode || ddi.Drug2.Code == rxnormCode {
			if compareSeverity(ddi.Severity, maxSeverity) > 0 {
				maxSeverity = ddi.Severity
			}
		}
	}
	return maxSeverity
}

// NeedsRenalDoseAdjustment checks if the patient needs renal dose adjustment.
func (kb *KnowledgeBridge) NeedsRenalDoseAdjustment() bool {
	safety := kb.GetSafetyInfo()
	if safety == nil {
		return false
	}
	return safety.RenalDoseAdjustmentNeeded
}

// GetRenalDoseAdjustment returns the renal dose adjustment for a medication.
func (kb *KnowledgeBridge) GetRenalDoseAdjustment(rxnormCode string) *contracts.DoseAdjustment {
	dosing := kb.GetDosingInfo()
	if dosing == nil {
		return nil
	}
	key := fmt.Sprintf("http://www.nlm.nih.gov/research/umls/rxnorm|%s", rxnormCode)
	if adj, ok := dosing.RenalAdjustments[key]; ok {
		return &adj
	}
	return nil
}

// IsOnFormulary checks if a medication is on the formulary.
func (kb *KnowledgeBridge) IsOnFormulary(rxnormCode string) bool {
	formulary := kb.GetFormularyInfo()
	if formulary == nil {
		return true // Assume on formulary if no info
	}
	key := fmt.Sprintf("http://www.nlm.nih.gov/research/umls/rxnorm|%s", rxnormCode)
	if status, ok := formulary.MedicationStatus[key]; ok {
		return status.Status == "preferred" || status.Status == "non-preferred"
	}
	return true
}

// GetEGFR returns the pre-computed eGFR if available.
func (kb *KnowledgeBridge) GetEGFR() *float64 {
	calcs := kb.GetCalculators()
	if calcs == nil || calcs.EGFR == nil {
		return nil
	}
	return &calcs.EGFR.Value
}

// ============================================================================
// Health check and diagnostics
// ============================================================================

// HealthCheck verifies connectivity to all KB services.
func (kb *KnowledgeBridge) HealthCheck() *clients.KBHealthStatus {
	return kb.kbClients.CheckHealth()
}

// Close releases resources held by the knowledge bridge.
func (kb *KnowledgeBridge) Close() error {
	// KB clients use standard HTTP client which doesn't need explicit close
	kb.snapshot = nil
	return nil
}

// Helper function to compare severity levels
func compareSeverity(a, b string) int {
	order := map[string]int{
		"none":     0,
		"low":      1,
		"mild":     1,
		"moderate": 2,
		"medium":   2,
		"high":     3,
		"severe":   3,
		"critical": 4,
	}
	return order[a] - order[b]
}
