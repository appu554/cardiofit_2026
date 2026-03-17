package harness

import (
	"fmt"
	"math"
	"time"

	"vaidshala/simulation/pkg/types"
)

// VMCUEngine simulates the V-MCU 19-step titration pipeline.
// This is the simulation equivalent of engines/vmcu/vmcu_engine.go.
type VMCUEngine struct {
	ChannelB   *PhysiologySafetyMonitor
	ChannelC   *ProtocolGuard
	Integrator *types.IntegratorState
	Traces     []types.SafetyTrace
	CycleCount int

	// Configuration
	AutonomyLimitPct      float64 // ±20% of physician-last-approved dose
	CumulativeLimitPct    float64 // ±50% cumulative from physician-approved dose
	CooldownBasalH        float64 // 48h between basal insulin changes
	CooldownRapidH        float64 // 6h between rapid-acting changes
	PostResumePct         float64 // 50% delta reduction post-resume
	ReentryPhases         int     // 3 phases
	ReentryCyclesEach     int     // 3 cycles per phase
	LastDoseChangeTime    time.Time
}

func NewVMCUEngine() *VMCUEngine {
	return &VMCUEngine{
		ChannelB:          NewPhysiologySafetyMonitor(),
		ChannelC:          NewProtocolGuard(),
		Integrator:         &types.IntegratorState{CurrentDose: 10.0, LastApprovedDose: 10.0},
		AutonomyLimitPct:   20.0,
		CumulativeLimitPct: 50.0,
		CooldownBasalH:    48.0,
		CooldownRapidH:    6.0,
		PostResumePct:     50.0,
		ReentryPhases:     3,
		ReentryCyclesEach: 3,
	}
}

// RunCycle executes one complete V-MCU titration cycle (19 steps).
// This is the core simulation entry point.
func (e *VMCUEngine) RunCycle(input types.TitrationCycleInput) types.TitrationCycleResult {
	e.CycleCount++
	now := time.Now()

	// Step 1-2: Read inputs
	labs := input.RawLabs
	ctx := input.TitrationContext

	// Step 3: Channel A — MCU_GATE from KB-23 (provided in input)
	mcuGate := input.MCUGate

	// Step 4: Channel B — PhysiologySafetyMonitor
	channelB := e.ChannelB.Evaluate(labs)

	// Step 5: Channel C — ProtocolGuard
	channelC := e.ChannelC.Evaluate(ctx, labs)

	// Step 5a: Arbiter — 1oo3 veto, most restrictive wins
	arbiterOutput := types.Arbitrate(types.ArbiterInput{
		MCUGate:      mcuGate,
		PhysioGate:   channelB.Gate,
		ProtocolGate: channelC.Gate,
	})

	result := types.TitrationCycleResult{
		FinalGate:         arbiterOutput.FinalGate,
		DominantChannel:   arbiterOutput.DominantChannel,
		PhysioRuleFired:   channelB.RuleFired,
		ProtocolRuleFired: channelC.RuleFired,
	}

	// Step 6: Integrator freeze/resume
	if arbiterOutput.FinalGate >= types.PAUSE {
		if !e.Integrator.Frozen {
			e.Integrator.Frozen = true
			e.Integrator.FrozenSince = now
		}
		result.BlockedBy = fmt.Sprintf("%s:%s", arbiterOutput.DominantChannel, arbiterOutput.FinalGate)
		result.DoseApplied = false
		result.DoseDelta = 0
	} else {
		// Resume if was frozen
		if e.Integrator.Frozen {
			pauseH := now.Sub(e.Integrator.FrozenSince).Hours()
			e.Integrator.PauseDurationH = pauseH
			e.Integrator.Frozen = false
			e.Integrator.PostResumeCount = 0
			e.Integrator.PostResumeLimit = int(math.Ceil(pauseH / 24.0))
			if pauseH > 72 { // Extended hold → 3-phase re-entry
				e.Integrator.ReentryPhase = 1
				e.Integrator.ReentryCycles = 0
			}
		}

		// Step 7: Cooldown check
		timeSinceLastDose := now.Sub(e.LastDoseChangeTime).Hours()
		if timeSinceLastDose < e.CooldownBasalH {
			result.DoseApplied = false
			result.BlockedBy = fmt.Sprintf("cooldown: %.0fh since last change (<%.0fh)", timeSinceLastDose, e.CooldownBasalH)
		} else {
			// Step 8-9: Re-entry protocol + rate limiter
			gainFactor := computeGainFactor(input.AdherenceScore)
			proposedDelta := computeBaseDelta(labs, ctx)

			// Apply gain factor (adherence modulation)
			proposedDelta *= gainFactor

			// Apply re-entry dampening
			if e.Integrator.ReentryPhase > 0 {
				switch e.Integrator.ReentryPhase {
				case 1: // Monitoring only
					proposedDelta = 0
				case 2: // Conservative (50% gain)
					proposedDelta *= 0.5
				case 3: // Normal
					// no change
				}
				e.Integrator.ReentryCycles++
				if e.Integrator.ReentryCycles >= e.ReentryCyclesEach {
					e.Integrator.ReentryPhase++
					e.Integrator.ReentryCycles = 0
					if e.Integrator.ReentryPhase > e.ReentryPhases {
						e.Integrator.ReentryPhase = 0
					}
				}
			}

			// Apply post-resume rate limiter (50% for ceil(pause_hours/24) cycles)
			if e.Integrator.PostResumeCount < e.Integrator.PostResumeLimit {
				proposedDelta *= (e.PostResumePct / 100.0)
				e.Integrator.PostResumeCount++
			}

			// Apply MODIFY constraint
			if arbiterOutput.FinalGate == types.MODIFY {
				if proposedDelta > 0 {
					proposedDelta = 0 // MODIFY: no upward titration
				}
			}

			// Step 14: Autonomy limits (±20% of physician-last-approved dose)
			maxDelta := e.Integrator.LastApprovedDose * (e.AutonomyLimitPct / 100.0)
			if math.Abs(proposedDelta) > maxDelta {
				if proposedDelta > 0 {
					proposedDelta = maxDelta
				} else {
					proposedDelta = -maxDelta
				}
			}

			// Step 14b: Cumulative autonomy limit (±50% from physician-approved dose)
			if e.CumulativeLimitPct > 0 && e.Integrator.LastApprovedDose > 0 {
				maxCumulative := e.Integrator.LastApprovedDose * (e.CumulativeLimitPct / 100.0)
				currentDrift := e.Integrator.CurrentDose - e.Integrator.LastApprovedDose
				newDrift := currentDrift + proposedDelta
				if math.Abs(newDrift) > maxCumulative {
					// Clamp delta so cumulative drift stays within limit
					if newDrift > 0 {
						proposedDelta = maxCumulative - currentDrift
					} else {
						proposedDelta = -maxCumulative - currentDrift
					}
					if math.Abs(proposedDelta) < 0.01 {
						proposedDelta = 0
					}
				}
			}

			if proposedDelta != 0 {
				result.DoseApplied = true
				result.DoseDelta = proposedDelta
				e.Integrator.CurrentDose += proposedDelta
				e.LastDoseChangeTime = now
			}
		}
	}

	// Step 16: SafetyTrace
	trace := types.SafetyTrace{
		TraceID:           fmt.Sprintf("SIM-%s-%d", input.PatientID, e.CycleCount),
		PatientID:         input.PatientID,
		CycleTimestamp:    now,
		CycleNumber:       e.CycleCount,
		MCUGate:           mcuGate,
		PhysioGate:        channelB.Gate,
		PhysioRuleFired:   channelB.RuleFired,
		ProtocolGate:      channelC.Gate,
		ProtocolRuleFired: channelC.RuleFired,
		FinalGate:         arbiterOutput.FinalGate,
		DominantChannel:   arbiterOutput.DominantChannel,
		DoseApplied:       result.DoseApplied,
		DoseDelta:         result.DoseDelta,
		BlockedBy:         result.BlockedBy,
		GainFactor:        computeGainFactor(input.AdherenceScore),
		AdherenceSource:   "SIMULATION",
	}
	e.Traces = append(e.Traces, trace)
	result.SafetyTrace = trace

	return result
}

// computeGainFactor maps adherence score to gain factor per V-MCU spec.
// ≥0.85 → 1.0, 0.65-0.84 → 0.75, 0.40-0.64 → 0.50, <0.40 → 0.25
func computeGainFactor(adherence float64) float64 {
	switch {
	case adherence >= 0.85:
		return 1.0
	case adherence >= 0.65:
		return 0.75
	case adherence >= 0.40:
		return 0.50
	default:
		return 0.25
	}
}

// computeBaseDelta calculates the base dose delta from clinical inputs.
// Simplified: +2U if FBG >7.2 mmol/L (130 mg/dL), -2U if FBG <4.4 (80 mg/dL).
func computeBaseDelta(labs *types.RawPatientData, ctx *types.TitrationContext) float64 {
	if labs.GlucoseCurrent <= 0 {
		return 0
	}
	fbg := labs.GlucoseCurrent
	switch {
	case fbg > 10.0: // >180 mg/dL
		return 4.0
	case fbg > 7.8: // >140 mg/dL
		return 2.0
	case fbg > 7.2: // >130 mg/dL
		return 1.0
	case fbg < 4.0: // <72 mg/dL
		return -2.0
	case fbg < 4.4: // <80 mg/dL
		return -1.0
	default:
		return 0
	}
}
