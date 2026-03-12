// Package factory provides orchestrator wiring for the clinical runtime platform.
//
// OrchestratorWiring connects all components:
// - KB HTTP Clients → KnowledgeSnapshotBuilder
// - KnowledgeSnapshotBuilder → ExecutionContextFactory
// - Engines (CQL, Measure, Medication) → EngineOrchestrator
//
// ARCHITECTURE (CTO/CMO Directive):
// 1. KB clients pre-compute answers at snapshot build time
// 2. Engines consume frozen ClinicalExecutionContext
// 3. Engines NEVER call KB services directly
package factory

import (
	"context"

	"vaidshala/clinical-runtime-platform/adapters"
	"vaidshala/clinical-runtime-platform/builders"
	"vaidshala/clinical-runtime-platform/clients"
	"vaidshala/clinical-runtime-platform/contracts"
	"vaidshala/clinical-runtime-platform/engines"
)

// WiringConfig configures the complete orchestrator wiring.
type WiringConfig struct {
	// KB client configuration
	KBConfig clients.KBClientConfig

	// Factory configuration
	FactoryConfig FactoryConfig

	// Orchestrator configuration
	OrchestratorConfig OrchestratorConfig

	// Engine configurations
	MedicationEngineConfig engines.MedicationEngineConfig

	// Feature flags
	EnableCQLEngine      bool
	EnableMeasureEngine  bool
	EnableMedicationEngine bool
}

// DefaultWiringConfig returns production-ready defaults.
func DefaultWiringConfig() WiringConfig {
	return WiringConfig{
		KBConfig:               clients.DefaultKBClientConfig(),
		FactoryConfig:          DefaultFactoryConfig(),
		OrchestratorConfig:     DefaultOrchestratorConfig(),
		MedicationEngineConfig: engines.DefaultMedicationEngineConfig(),
		EnableCQLEngine:        true,
		EnableMeasureEngine:    true,
		EnableMedicationEngine: true,
	}
}

// WiredOrchestrator contains the fully-wired orchestrator and its dependencies.
type WiredOrchestrator struct {
	// Orchestrator is the main entry point for clinical execution
	Orchestrator *EngineOrchestrator

	// Factory builds ClinicalExecutionContext
	Factory *ExecutionContextFactory

	// KBClients for health checks and monitoring
	KBClients *clients.KBClients

	// SnapshotBuilder for direct snapshot operations
	SnapshotBuilder *builders.KnowledgeSnapshotBuilder

	// Engines for direct access if needed
	MedicationEngine *engines.MedicationEngine
}

// WireOrchestrator creates a fully-wired orchestrator with all dependencies.
//
// WIRING ORDER:
// 1. Create KB HTTP clients
// 2. Create KnowledgeSnapshotBuilder with KB clients
// 3. Create ExecutionContextFactory with snapshot builder
// 4. Create engines (CQL, Measure, Medication)
// 5. Create EngineOrchestrator with all engines
func WireOrchestrator(config WiringConfig) (*WiredOrchestrator, error) {
	// ========================================================================
	// STEP 1: Create KB HTTP Clients
	// ========================================================================
	kbClients := clients.NewKBClients(config.KBConfig)

	// ========================================================================
	// STEP 2: Create KnowledgeSnapshotBuilder
	// Uses FHIR client for terminology (CTO/CMO directive)
	// ========================================================================
	snapshotBuilder := builders.NewKnowledgeSnapshotBuilderFHIR(
		kbClients.KB7,  // KB-7 Terminology (FHIR)
		kbClients.KB8,  // KB-8 Calculator
		kbClients.KB4,  // KB-4 Patient Safety
		kbClients.KB5,  // KB-5 Drug Interactions
		kbClients.KB6,  // KB-6 Formulary
		kbClients.KB1,  // KB-1 Drug Rules
		nil,            // KB-11 CDI (not wired yet)
		nil,            // KB-16 Lab Interpretation (not wired yet)
		builders.DefaultKnowledgeSnapshotConfig(),
	)

	// ========================================================================
	// STEP 3: Create ExecutionContextFactory
	// ========================================================================
	// Create KB-2 adapters (for patient context assembly)
	kb2Adapter := adapters.NewKB2Adapter(nil, adapters.DefaultKB2AdapterConfig())
	kb2Intelligence := &noOpKB2Intelligence{}

	factory := NewExecutionContextFactory(
		kb2Adapter,
		kb2Intelligence,
		snapshotBuilder,
		config.FactoryConfig,
	)

	// ========================================================================
	// STEP 4: Create Engines
	// ========================================================================
	var medicationEngine *engines.MedicationEngine
	otherEngines := make([]Engine, 0)

	// Create MedicationEngine
	if config.EnableMedicationEngine {
		medicationEngine = engines.NewMedicationEngine(config.MedicationEngineConfig)
		otherEngines = append(otherEngines, medicationEngine)
	}

	// ========================================================================
	// STEP 5: Create EngineOrchestrator
	// ========================================================================
	var orchestrator *EngineOrchestrator

	if config.EnableCQLEngine || config.EnableMeasureEngine {
		// Full CQL → Measure flow with other engines
		// Note: CQL and Measure engines would be wired here when available
		orchestrator = NewEngineOrchestratorWithConfig(
			factory,
			config.OrchestratorConfig,
			nil, // CQL Engine (wire when available)
			nil, // Measure Engine (wire when available)
			otherEngines...,
		)
	} else {
		// Simple orchestrator without CQL → Measure flow
		orchestrator = NewSimpleOrchestrator(factory, otherEngines...)
	}

	return &WiredOrchestrator{
		Orchestrator:     orchestrator,
		Factory:          factory,
		KBClients:        kbClients,
		SnapshotBuilder:  snapshotBuilder,
		MedicationEngine: medicationEngine,
	}, nil
}

// WireDefaultOrchestrator creates orchestrator with default configuration.
func WireDefaultOrchestrator() (*WiredOrchestrator, error) {
	return WireOrchestrator(DefaultWiringConfig())
}

// WireOrchestratorFromEnv creates orchestrator with environment-based configuration.
// Uses KBClientConfigFromEnv() to read KB service URLs from environment variables.
func WireOrchestratorFromEnv() (*WiredOrchestrator, error) {
	config := DefaultWiringConfig()
	config.KBConfig = clients.KBClientConfigFromEnv()
	return WireOrchestrator(config)
}

// WireMedicationOnlyOrchestrator creates orchestrator with only MedicationEngine.
// Useful for medication-specific workflows without CQL/Measure overhead.
func WireMedicationOnlyOrchestrator() (*WiredOrchestrator, error) {
	config := DefaultWiringConfig()
	config.EnableCQLEngine = false
	config.EnableMeasureEngine = false
	config.EnableMedicationEngine = true
	return WireOrchestrator(config)
}

// ============================================================================
// NO-OP IMPLEMENTATIONS FOR MISSING ADAPTERS
// ============================================================================

// noOpKB2Intelligence is a no-op implementation of KB2Intelligence.
// Used when KB-2B enrichment is not available.
// Returns the base patient context unchanged.
type noOpKB2Intelligence struct{}

// Enrich returns the base patient context unchanged (no-op enrichment).
func (n *noOpKB2Intelligence) Enrich(ctx context.Context, base *contracts.PatientContext) (*contracts.PatientContext, error) {
	return base, nil
}

// Ensure noOpKB2Intelligence implements KB2Intelligence
var _ adapters.KB2Intelligence = (*noOpKB2Intelligence)(nil)
