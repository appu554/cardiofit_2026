// Package harness provides the multi-cycle simulation runner.
// This simulates a full 90-day correction loop trajectory.
package harness

import (
	"fmt"
	"math"
	"math/rand"

	"vaidshala/simulation/pkg/types"
)

// SimulationConfig controls the multi-cycle simulation parameters.
type SimulationConfig struct {
	TotalCycles       int     // Number of titration cycles to simulate
	CycleDurationDays float64 // Days between cycles (e.g., 1.0 for daily)
	NoiseStdDev       float64 // Gaussian noise on glucose readings (mmol/L)
}

// DefaultConfig returns the standard 90-day simulation configuration.
func DefaultConfig() SimulationConfig {
	return SimulationConfig{
		TotalCycles:       90,  // 90 days
		CycleDurationDays: 1.0, // Daily cycles
		NoiseStdDev:       0.5, // ±0.5 mmol/L measurement noise
	}
}

// SimulationResult captures the outcome of a multi-cycle simulation.
type SimulationResult struct {
	PatientID     string
	Archetype     string
	TotalCycles   int
	HALTCount     int
	PAUSECount    int
	HOLDDataCount int
	CLEARCount    int
	MODIFYCount   int
	DosesApplied  int
	TotalDoseDelta float64
	FinalDose     float64
	InitialDose   float64
	GlucoseStart  float64
	GlucoseEnd    float64
	Traces        []types.SafetyTrace
	CycleResults  []types.TitrationCycleResult
}

// PhysiologyState is the evolving patient state during multi-cycle simulation.
type PhysiologyState struct {
	Glucose    float64 // Current FBG (mmol/L)
	HbA1c     float64
	SBP        int
	DBP        int
	EGFR       float64
	Creatinine float64
	Potassium  float64
	Sodium     float64
	Weight     float64
	HeartRate  int

	// Drug effects
	InsulinDose     float64
	InsulinSensitivity float64 // How much 1U lowers glucose (mmol/L)
	BasalGlucose    float64    // Hepatic glucose output without insulin

	// Progression
	NaturalGlucoseRise float64 // mmol/L per day (disease progression)
	NaturalEGFRDecline float64 // mL/min/1.73m² per day
}

// RunMultiCycle runs a complete multi-cycle simulation with physiological evolution.
func RunMultiCycle(engine *VMCUEngine, initial PhysiologyState, config SimulationConfig) SimulationResult {
	state := initial
	result := SimulationResult{
		GlucoseStart: state.Glucose,
		InitialDose:  state.InsulinDose,
	}

	prevGlucose := state.Glucose
	prevCreatinine := state.Creatinine
	prevWeight := state.Weight

	for cycle := 1; cycle <= config.TotalCycles; cycle++ {
		// Add measurement noise
		measuredGlucose := state.Glucose + rand.NormFloat64()*config.NoiseStdDev
		if measuredGlucose < 1.5 {
			measuredGlucose = 1.5 // Floor to prevent impossible values
		}

		// Build patient data for this cycle
		labs := &types.RawPatientData{
			PatientID:          fmt.Sprintf("SIM-MC-%s", result.PatientID),
			GlucoseCurrent:     measuredGlucose,
			GlucosePrevious:    prevGlucose,
			CreatinineCurrent:  state.Creatinine,
			CreatininePrevious: prevCreatinine,
			EGFR:               state.EGFR,
			PotassiumCurrent:   state.Potassium,
			SBP:                state.SBP,
			DBP:                state.DBP,
			HeartRate:          state.HeartRate,
			SodiumCurrent:      state.Sodium,
			Weight:             state.Weight,
			WeightPrevious:     prevWeight,
			HeartRateRegularity: "REGULAR",
		}

		ctx := &types.TitrationContext{
			CurrentDose:   state.InsulinDose,
			EGFRCurrent:   state.EGFR,
			InsulinActive: state.InsulinDose > 0,
		}

		input := types.TitrationCycleInput{
			PatientID:        result.PatientID,
			CycleNumber:      cycle,
			RawLabs:          labs,
			TitrationContext: ctx,
			MCUGate:          types.CLEAR,
			AdherenceScore:   0.85,
			LoopTrustScore:   0.80,
		}

		// Run V-MCU cycle
		cycleResult := engine.RunCycle(input)
		result.CycleResults = append(result.CycleResults, cycleResult)

		// Count gate outcomes
		switch cycleResult.FinalGate {
		case types.HALT:
			result.HALTCount++
		case types.PAUSE:
			result.PAUSECount++
		case types.HOLD_DATA:
			result.HOLDDataCount++
		case types.MODIFY:
			result.MODIFYCount++
		case types.CLEAR:
			result.CLEARCount++
		}

		if cycleResult.DoseApplied {
			result.DosesApplied++
			result.TotalDoseDelta += cycleResult.DoseDelta
			state.InsulinDose += cycleResult.DoseDelta
		}

		// Evolve physiology
		prevGlucose = measuredGlucose
		prevCreatinine = state.Creatinine
		prevWeight = state.Weight

		// Glucose response to insulin change
		insulinEffect := state.InsulinDose * state.InsulinSensitivity
		state.Glucose = state.BasalGlucose - insulinEffect + state.NaturalGlucoseRise*float64(cycle)

		// Clamp glucose to physiological range
		state.Glucose = math.Max(2.5, math.Min(30.0, state.Glucose))

		// Natural disease progression
		state.EGFR -= state.NaturalEGFRDecline
		if state.EGFR < 5 {
			state.EGFR = 5
		}
	}

	result.TotalCycles = config.TotalCycles
	result.GlucoseEnd = state.Glucose
	result.FinalDose = state.InsulinDose
	result.Traces = engine.Traces
	return result
}
