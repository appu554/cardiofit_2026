// result_mapper.go converts production TitrationCycleResult + SafetyTrace back
// to simulation TitrationCycleResult, and provides a rule ID normalization table
// that maps between simulation and production rule IDs (they diverged during
// production development).
package bridge

import (
	"fmt"

	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
	simtypes "vaidshala/simulation/pkg/types"
)

// ---------------------------------------------------------------------------
// Rule ID normalization
// ---------------------------------------------------------------------------

// NormDirection indicates the mapping direction for rule ID normalization.
type NormDirection int

const (
	DirectionSimToProduction NormDirection = iota
	DirectionProdToSimulation
)

// ruleIDSimToProd maps simulation rule IDs to production rule IDs.
// The two rule sets diverged during production development; this table
// is the canonical reconciliation point.
var ruleIDSimToProd = map[string]string{
	"B-01":       "B-01",
	"B-02":       "B-07",
	"B-03":       "B-04",
	"B-04":       "B-03",
	"B-04+PG-14": "B-03-RAAS-SUPPRESSED",
	"B-05":       "B-06",
	"B-06":       "DA-01",
	"B-07":       "B-08",
	"B-08":       "B-05",
	"B-09":       "B-09",
	"B-10":       "DA-06",
	"B-11":       "DA-07",
	"B-12":       "B-12",
	"B-13":       "B-13",
	"B-14":       "B-14",
	"B-15":       "B-15",
	"B-16":       "B-16",
	"B-17":       "B-17",
	"B-18":       "B-18",
	"PG-01":      "PG-01",
	"PG-02":      "PG-02",
	"PG-03":      "PG-03",
	"PG-04":      "PG-04",
	"PG-05":      "PG-05",
	"PG-06":      "PG-06",
	"PG-07":      "PG-07",
	"PG-08":      "PG-08",
	"PG-14":      "PG-14",
	"B-20":       "B-20",
	"PG-17-A3":   "PG-17-A3",
	"PG-17-A2":   "PG-17-A2",
}

// productionOnlyRules: rules in production with NO simulation equivalent.
// IMPORTANT: Do NOT add IDs that appear as VALUES in ruleIDSimToProd.
// e.g., DA-01 maps FROM sim B-06, so it's NOT production-only.
// Production B-10/B-11 are DIFFERENT rules from simulation B-10/B-11.
var productionOnlyRules = map[string]bool{
	"B-10": true, "B-11": true, "B-19": true, "B-21": true,
	"DA-02": true, "DA-03": true, "DA-04": true, "DA-05": true, "DA-08": true,
	"PG-08-DUAL-RAAS": true,
	"PG-09":           true, "PG-10": true, "PG-11": true, "PG-12": true, "PG-13": true,
	"PG-15": true, "PG-16": true,
	"PG-18": true, "PG-19": true,
}

// NormalizeRuleID maps a rule ID between simulation and production namespaces.
// Panics on unknown IDs — callers must ensure inputs are valid rule IDs.
func NormalizeRuleID(ruleID string, direction NormDirection) string {
	if direction == DirectionSimToProduction {
		prod, ok := ruleIDSimToProd[ruleID]
		if !ok {
			panic(fmt.Sprintf("bridge: unknown simulation rule ID: %q", ruleID))
		}
		return prod
	}
	// Production → simulation: reverse lookup
	if productionOnlyRules[ruleID] {
		return "PRODUCTION_ONLY"
	}
	for simID, prodID := range ruleIDSimToProd {
		if prodID == ruleID {
			return simID
		}
	}
	panic(fmt.Sprintf("bridge: unknown production rule ID: %q", ruleID))
}

// ---------------------------------------------------------------------------
// Result mapping: production → simulation
// ---------------------------------------------------------------------------

// ToSimulationResult converts a production TitrationCycleResult (with optional
// SafetyTrace) back to the simulation TitrationCycleResult type.
//
// Production uses string-based DominantChannel ("A", "B", "C", "NONE") while
// simulation uses Channel constants ("MCU_GATE", "PHYSIO_GATE", "PROTOCOL_GATE").
// Production uses *float64 for DoseApplied/DoseDelta while simulation uses
// bool + float64.
func ToSimulationResult(prod *vt.TitrationCycleResult) simtypes.TitrationCycleResult {
	doseApplied := prod.DoseApplied != nil
	doseDelta := derefFloat64(prod.DoseDelta)

	// Map production channel string to simulation Channel type
	domChannel := mapDominantChannel(prod.Arbiter.DominantChannel)

	return simtypes.TitrationCycleResult{
		FinalGate:         GateSignalToSimulation(prod.Arbiter.FinalGate),
		DominantChannel:   domChannel,
		DoseApplied:       doseApplied,
		DoseDelta:         doseDelta,
		BlockedBy:         prod.BlockedBy,
		PhysioRuleFired:   prod.ChannelB.RuleFired,
		ProtocolRuleFired: prod.ChannelC.RuleID,
		SafetyTrace: simtypes.SafetyTrace{
			MCUGate:      GateSignalToSimulation(prod.ChannelA.Gate),
			PhysioGate:   GateSignalToSimulation(prod.ChannelB.Gate),
			ProtocolGate: GateSignalToSimulation(prod.ChannelC.Gate),
			FinalGate:    GateSignalToSimulation(prod.Arbiter.FinalGate),
			DoseApplied:  doseApplied,
			DoseDelta:    doseDelta,
			GainFactor:   prod.ChannelA.GainFactor,
		},
	}
}

// mapDominantChannel converts a production DominantChannel string ("A", "B",
// "C", "NONE") to the simulation Channel constant.
func mapDominantChannel(prodChannel string) simtypes.Channel {
	switch prodChannel {
	case "A":
		return simtypes.ChannelA
	case "B":
		return simtypes.ChannelB
	case "C":
		return simtypes.ChannelC
	default:
		return simtypes.ChannelB // NONE → safe default (physio monitor)
	}
}
