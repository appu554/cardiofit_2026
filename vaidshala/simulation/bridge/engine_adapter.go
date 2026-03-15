// engine_adapter.go wraps the production VMCUEngine so simulation scenarios
// can invoke it through the bridge. It converts simulation TitrationCycleInput
// to production TitrationCycleInput, calls RunCycle(), and converts the result
// back to simulation types.
//
// Infrastructure dependencies (SafetyCache, TraceStore, EventPublisher, etc.)
// are left nil — the adapter runs the engine in "zero infrastructure" mode,
// which is safe because RunCycle gracefully skips all nil-guarded code paths.
package bridge

import (
	"fmt"
	"time"

	vmcu "vaidshala/clinical-runtime-platform/engines/vmcu"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
	simtypes "vaidshala/simulation/pkg/types"
)

// ProductionEngine wraps a production VMCUEngine for use in simulation.
type ProductionEngine struct {
	engine  *vmcu.VMCUEngine
	simTime time.Time
}

// EngineOption configures a ProductionEngine at construction time.
type EngineOption func(*engineConfig)

type engineConfig struct {
	protocolRulesPath string
	simTime           time.Time
}

// WithProtocolRulesPath overrides the default protocol_rules.yaml location.
func WithProtocolRulesPath(path string) EngineOption {
	return func(c *engineConfig) { c.protocolRulesPath = path }
}

// WithSimulatedTime sets the reference time used for lab timestamps when
// the simulation input does not provide explicit timestamps.
func WithSimulatedTime(t time.Time) EngineOption {
	return func(c *engineConfig) { c.simTime = t }
}

// NewProductionEngine constructs a ProductionEngine backed by a real VMCUEngine.
// All infrastructure dependencies (cache, events, traces) are left nil.
func NewProductionEngine(opts ...EngineOption) (*ProductionEngine, error) {
	cfg := &engineConfig{
		protocolRulesPath: "testdata/protocol_rules.yaml",
		simTime:           time.Now(),
	}
	for _, opt := range opts {
		opt(cfg)
	}

	vmcuCfg := vmcu.DefaultVMCUConfig()
	vmcuCfg.ProtocolRulesPath = cfg.protocolRulesPath

	engine, err := vmcu.NewVMCUEngine(vmcuCfg)
	if err != nil {
		return nil, fmt.Errorf("bridge: failed to create production engine: %w", err)
	}

	return &ProductionEngine{
		engine:  engine,
		simTime: cfg.simTime,
	}, nil
}

// RunCycle converts the simulation input to production types, runs the
// production engine, and converts the result back to simulation types.
func (pe *ProductionEngine) RunCycle(input simtypes.TitrationCycleInput) simtypes.TitrationCycleResult {
	// Build timestamps — use simulation-provided timestamps if available,
	// otherwise fall back to the adapter's simTime.
	ts := PatientTimestamps{
		LastGlucose:    pe.simTime,
		LastCreatinine: pe.simTime,
		LastPotassium:  pe.simTime,
		LastHbA1c:      pe.simTime,
		LastEGFR:       pe.simTime,
	}
	if input.RawLabs != nil {
		if !input.RawLabs.GlucoseTimestamp.IsZero() {
			ts.LastGlucose = input.RawLabs.GlucoseTimestamp
		}
		if !input.RawLabs.CreatinineTimestamp.IsZero() {
			ts.LastCreatinine = input.RawLabs.CreatinineTimestamp
		}
		if !input.RawLabs.PotassiumTimestamp.IsZero() {
			ts.LastPotassium = input.RawLabs.PotassiumTimestamp
		}
		if !input.RawLabs.HbA1cTimestamp.IsZero() {
			ts.LastHbA1c = input.RawLabs.HbA1cTimestamp
		}
		if !input.RawLabs.EGFRTimestamp.IsZero() {
			ts.LastEGFR = input.RawLabs.EGFRTimestamp
		}
	}

	prodInput := vmcu.TitrationCycleInput{
		PatientID: input.PatientID,
		ChannelAResult: vt.ChannelAResult{
			Gate:       GateSignalToProduction(input.MCUGate),
			GainFactor: 1.0,
		},
		RawLabs:          ToProductionRawLabs(input.RawLabs, input.TitrationContext, ts),
		TitrationContext: ToProductionContext(input.TitrationContext),
		CurrentDose:      input.TitrationContext.CurrentDose,
		ProposedDelta:    input.TitrationContext.ProposedDoseDelta,
		// MedClass and MetabolicInput left at zero values — safe defaults.
	}

	// RunCycle returns (*TitrationCycleResult, *SafetyTrace).
	// The simulation bridge only needs the TitrationCycleResult.
	prodResult, _ := pe.engine.RunCycle(prodInput)
	return ToSimulationResult(prodResult)
}
