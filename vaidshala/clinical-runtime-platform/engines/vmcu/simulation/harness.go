// harness.go implements a 90-day time-stepping simulation engine for V-MCU
// integration testing. Each simulation runs 360 cycles (6-hour intervals)
// and captures SafetyTrace + TitrationCycleResult at every step.
package simulation

import (
	"fmt"
	"time"

	"vaidshala/clinical-runtime-platform/engines/vmcu"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_b"
	"vaidshala/clinical-runtime-platform/engines/vmcu/channel_c"
	"vaidshala/clinical-runtime-platform/engines/vmcu/trace"
	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

const (
	// CyclesPerDay is the number of V-MCU cycles per day (every 6 hours).
	CyclesPerDay = 4
	// TotalDays is the simulation duration.
	TotalDays = 90
	// TotalCycles = 90 * 4 = 360
	TotalCycles = TotalDays * CyclesPerDay
	// CycleInterval is the time between cycles.
	CycleInterval = 6 * time.Hour
)

// CycleLog captures the result of a single simulation cycle.
type CycleLog struct {
	Day     int
	Cycle   int // 0-based within day
	Time    time.Time
	Input   vmcu.TitrationCycleInput
	Result  *vt.TitrationCycleResult
	Trace   *trace.SafetyTrace
	Dose    float64 // dose after this cycle
	Notes   string  // scenario-specific annotation
}

// SimulationResult holds the complete output of a simulation run.
type SimulationResult struct {
	ScenarioName string
	PatientID    string
	StartTime    time.Time
	EndTime      time.Time
	Cycles       []CycleLog
	FinalDose    float64
}

// CycleModifier is called before each cycle to let scenarios mutate the
// patient state (labs, Channel A result, context, etc.) based on where
// we are in the 90-day trajectory.
type CycleModifier func(day, cycleInDay int, simTime time.Time, patient *PatientState) (
	channelA vt.ChannelAResult,
	labs *channel_b.RawPatientData,
	ctx *channel_c.TitrationContext,
	note string,
)

// Harness orchestrates a 90-day simulation.
type Harness struct {
	engine *vmcu.VMCUEngine
}

// NewHarness creates a simulation harness with a fully configured V-MCU engine.
func NewHarness(engine *vmcu.VMCUEngine) *Harness {
	return &Harness{engine: engine}
}

// Run executes a complete 90-day simulation.
// The modifier function controls how patient state evolves each cycle.
func (h *Harness) Run(scenarioName string, patient *PatientState, modifier CycleModifier) *SimulationResult {
	startTime := time.Date(2026, 1, 1, 6, 0, 0, 0, time.UTC)

	result := &SimulationResult{
		ScenarioName: scenarioName,
		PatientID:    patient.ID,
		StartTime:    startTime,
		Cycles:       make([]CycleLog, 0, TotalCycles),
	}

	currentDose := patient.InitialDose

	for day := 0; day < TotalDays; day++ {
		for cycle := 0; cycle < CyclesPerDay; cycle++ {
			simTime := startTime.Add(time.Duration(day*CyclesPerDay+cycle) * CycleInterval)

			// Let scenario modify patient state for this cycle
			chA, labs, titCtx, note := modifier(day, cycle, simTime, patient)

			input := vmcu.TitrationCycleInput{
				PatientID: patient.ID,
				ChannelAResult: chA,
				RawLabs:        labs,
				TitrationContext: titCtx,
				CurrentDose:   currentDose,
				ProposedDelta: patient.ProposedDelta,
				MedClass:      patient.MedClass,
			}

			cycleResult, safetyTrace := h.engine.RunCycle(input)

			// Update dose from result
			if cycleResult.DoseApplied != nil {
				currentDose = *cycleResult.DoseApplied
			}

			result.Cycles = append(result.Cycles, CycleLog{
				Day:    day,
				Cycle:  cycle,
				Time:   simTime,
				Input:  input,
				Result: cycleResult,
				Trace:  safetyTrace,
				Dose:   currentDose,
				Notes:  note,
			})
		}
	}

	result.EndTime = startTime.Add(TotalCycles * CycleInterval)
	result.FinalDose = currentDose

	return result
}

// CyclesInRange returns cycle logs for a given day range [startDay, endDay).
func (r *SimulationResult) CyclesInRange(startDay, endDay int) []CycleLog {
	var out []CycleLog
	for _, c := range r.Cycles {
		if c.Day >= startDay && c.Day < endDay {
			out = append(out, c)
		}
	}
	return out
}

// CyclesWithGate returns cycles where the arbiter final gate matches.
func (r *SimulationResult) CyclesWithGate(gate vt.GateSignal) []CycleLog {
	var out []CycleLog
	for _, c := range r.Cycles {
		if c.Result != nil && c.Result.Arbiter.FinalGate == gate {
			out = append(out, c)
		}
	}
	return out
}

// CyclesBlockedBy returns cycles blocked by a specific prefix (e.g., "AUTONOMY:", "COOLDOWN:").
func (r *SimulationResult) CyclesBlockedBy(prefix string) []CycleLog {
	var out []CycleLog
	for _, c := range r.Cycles {
		if c.Result != nil && len(c.Result.BlockedBy) > 0 {
			if len(prefix) == 0 || c.Result.BlockedBy[:min(len(prefix), len(c.Result.BlockedBy))] == prefix {
				out = append(out, c)
			}
		}
	}
	return out
}

// TraceCount returns the number of cycles that produced a SafetyTrace.
func (r *SimulationResult) TraceCount() int {
	count := 0
	for _, c := range r.Cycles {
		if c.Trace != nil {
			count++
		}
	}
	return count
}

// DoseHistory returns (day, dose) pairs for charting.
func (r *SimulationResult) DoseHistory() []struct {
	Day  int
	Dose float64
} {
	seen := make(map[int]float64)
	for _, c := range r.Cycles {
		// Last cycle of each day
		seen[c.Day] = c.Dose
	}
	out := make([]struct {
		Day  int
		Dose float64
	}, 0, len(seen))
	for d := 0; d < TotalDays; d++ {
		if dose, ok := seen[d]; ok {
			out = append(out, struct {
				Day  int
				Dose float64
			}{d, dose})
		}
	}
	return out
}

// Summary returns a human-readable summary of the simulation.
func (r *SimulationResult) Summary() string {
	haltCycles := r.CyclesWithGate(vt.GateHalt)
	pauseCycles := r.CyclesWithGate(vt.GatePause)
	blockedCycles := r.CyclesBlockedBy("")

	return fmt.Sprintf(
		"Scenario: %s | Patient: %s | Days: %d | Cycles: %d | "+
			"HALTs: %d | PAUSEs: %d | Blocked: %d | Final dose: %.1f | Traces: %d/%d",
		r.ScenarioName, r.PatientID, TotalDays, len(r.Cycles),
		len(haltCycles), len(pauseCycles), len(blockedCycles),
		r.FinalDose, r.TraceCount(), len(r.Cycles),
	)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
