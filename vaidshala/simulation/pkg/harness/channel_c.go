package harness

import (
	"fmt"
	"math"

	"vaidshala/simulation/pkg/types"
)

// ProtocolGuard evaluates Channel C rules against titration context.
// Rules are pre-compiled from protocol_rules.yaml — no runtime network calls.
type ProtocolGuard struct{}

func NewProtocolGuard() *ProtocolGuard { return &ProtocolGuard{} }

type ChannelCResult struct {
	Gate      types.GateSignal
	RuleFired string
	Details   string
}

// Evaluate runs all Protocol Guard rules.
func (pg *ProtocolGuard) Evaluate(ctx *types.TitrationContext, labs *types.RawPatientData) ChannelCResult {
	results := []ChannelCResult{
		pg.rulePG01(ctx),
		pg.rulePG02(ctx),
		pg.rulePG03(labs),
		pg.rulePG04(ctx, labs),
		pg.rulePG05(ctx),
		pg.rulePG06(ctx),
		pg.rulePG07(labs, ctx),
		pg.rulePG08(ctx),
		pg.rulePG14(ctx, labs),
		pg.rulePG17(ctx),
	}

	worst := ChannelCResult{Gate: types.CLEAR, RuleFired: "NONE"}
	for _, r := range results {
		if r.Gate > worst.Gate {
			worst = r
		}
	}
	return worst
}

// PG-01: eGFR <30 AND metformin active → HALT (KDIGO absolute contraindication)
func (pg *ProtocolGuard) rulePG01(ctx *types.TitrationContext) ChannelCResult {
	if ctx.EGFRCurrent < 30 {
		for _, med := range ctx.ActiveMedications {
			if med.DrugClass == "METFORMIN" {
				return ChannelCResult{types.HALT, "PG-01",
					fmt.Sprintf("metformin_contraindicated: eGFR=%.0f (<30)", ctx.EGFRCurrent)}
			}
		}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-02: eGFR <45 AND SGLT2i active → PAUSE (KDIGO threshold)
func (pg *ProtocolGuard) rulePG02(ctx *types.TitrationContext) ChannelCResult {
	if ctx.EGFRCurrent < 45 && ctx.SGLT2iActive {
		return ChannelCResult{types.PAUSE, "PG-02",
			fmt.Sprintf("sglt2i_ckd_caution: eGFR=%.0f (<45)", ctx.EGFRCurrent)}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-03: AKI detected (cross-channel — Channel B HALT for creatinine) → HALT
func (pg *ProtocolGuard) rulePG03(labs *types.RawPatientData) ChannelCResult {
	if labs.CreatinineCurrent > 0 && labs.CreatininePrevious > 0 {
		delta := labs.CreatinineCurrent - labs.CreatininePrevious
		if delta > 26.0 && !labs.CreatinineRiseExplained {
			return ChannelCResult{types.HALT, "PG-03",
				fmt.Sprintf("aki_cross_channel: creatinine delta=%.0f µmol/L", delta)}
		}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-04: Active hypoglycaemia AND insulin increase proposed → HALT
func (pg *ProtocolGuard) rulePG04(ctx *types.TitrationContext, labs *types.RawPatientData) ChannelCResult {
	if labs.GlucoseCurrent > 0 && labs.GlucoseCurrent < 3.9 &&
		ctx.InsulinActive && ctx.ProposedDoseDelta > 0 {
		return ChannelCResult{types.HALT, "PG-04",
			"hypo_plus_insulin_increase: absolute clinical rule"}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-05: dose_delta >20% of current dose → PAUSE (algorithmic drift protection)
func (pg *ProtocolGuard) rulePG05(ctx *types.TitrationContext) ChannelCResult {
	if ctx.CurrentDose > 0 && ctx.ProposedDoseDelta != 0 {
		pctChange := math.Abs(ctx.ProposedDoseDelta) / ctx.CurrentDose * 100
		if pctChange > 20 {
			return ChannelCResult{types.PAUSE, "PG-05",
				fmt.Sprintf("dose_drift: proposed %.1f%% change (>20%%)", pctChange)}
		}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-06: 12 cycles with no HbA1c improvement → PAUSE (therapeutic futility)
func (pg *ProtocolGuard) rulePG06(ctx *types.TitrationContext) ChannelCResult {
	if ctx.DoseChangeCount >= 12 && ctx.CyclesSinceHbA1c >= 12 {
		return ChannelCResult{types.PAUSE, "PG-06",
			fmt.Sprintf("therapeutic_futility: %d cycles without HbA1c improvement", ctx.CyclesSinceHbA1c)}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-07: Hypoglycaemia within 7 days AND dose increase proposed → HALT
func (pg *ProtocolGuard) rulePG07(labs *types.RawPatientData, ctx *types.TitrationContext) ChannelCResult {
	// Simplified: if glucose was recently low and we're proposing increase
	if labs.GlucosePrevious > 0 && labs.GlucosePrevious < 3.9 && ctx.ProposedDoseDelta > 0 {
		return ChannelCResult{types.HALT, "PG-07",
			"post_hypo_safety_window: hypo within 7 days + dose increase proposed"}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-08: Dual RAAS (ACEi + ARB simultaneously) → HALT
func (pg *ProtocolGuard) rulePG08(ctx *types.TitrationContext) ChannelCResult {
	if ctx.ACEiActive && ctx.ARBActive {
		return ChannelCResult{types.HALT, "PG-08",
			"dual_raas: ACEi + ARB simultaneously — contraindicated"}
	}
	return ChannelCResult{Gate: types.CLEAR}
}

// PG-14: RAAS creatinine tolerance (HTN Amendment 1 — most critical)
// ACEi/ARB dose increased within 14 days AND creatinine rise <30% → PAUSE (not HALT)
// This is the causal suppression rule that prevents false AKI on every ACEi/ARB increase.
func (pg *ProtocolGuard) rulePG14(ctx *types.TitrationContext, labs *types.RawPatientData) ChannelCResult {
	if !ctx.RAASChangeWithin14Days {
		return ChannelCResult{Gate: types.CLEAR}
	}
	if labs.CreatinineCurrent <= 0 || ctx.PreRAASCreatinine <= 0 {
		return ChannelCResult{Gate: types.CLEAR}
	}

	risePct := (labs.CreatinineCurrent - ctx.PreRAASCreatinine) / ctx.PreRAASCreatinine * 100

	if risePct > 30 || labs.PotassiumCurrent > 5.5 {
		// Rise exceeds RAAS tolerance — genuine concern, escalate to HALT
		return ChannelCResult{types.HALT, "PG-14-ESCALATE",
			fmt.Sprintf("raas_tolerance_exceeded: creatinine rise %.0f%% (>30%%) or K+=%.1f (>5.5)",
				risePct, labs.PotassiumCurrent)}
	}

	if risePct > 0 && risePct <= 30 && labs.PotassiumCurrent <= 5.5 {
		// Expected RAAS response — PAUSE with monitoring, not HALT
		return ChannelCResult{types.PAUSE, "PG-14",
			fmt.Sprintf("expected_raas_response: creatinine rise %.0f%% (≤30%%), K+=%.1f (≤5.5). Monitor at 7d and 14d.",
				risePct, labs.PotassiumCurrent)}
	}

	return ChannelCResult{Gate: types.CLEAR}
}

// PG-17: ACR-based RAAS escalation
// A3 without RAAS → HALT (RAAS blockade required)
// A2 without RAAS → MODIFY (recommend RAAS initiation)
func (pg *ProtocolGuard) rulePG17(ctx *types.TitrationContext) ChannelCResult {
	if ctx.ACRCategory == "A3" && !ctx.ACEiActive && !ctx.ARBActive {
		return ChannelCResult{types.HALT, "PG-17-A3",
			"acr_a3_no_raas: RAAS blockade required"}
	}
	if ctx.ACRCategory == "A2" && !ctx.ACEiActive && !ctx.ARBActive {
		return ChannelCResult{types.MODIFY, "PG-17-A2",
			"acr_a2_no_raas: recommend RAAS initiation"}
	}
	return ChannelCResult{Gate: types.CLEAR}
}
