package main

import (
	"fmt"
	"os"
	"text/tabwriter"
	"time"

	"vaidshala/simulation/pkg/harness"
	"vaidshala/simulation/pkg/patient"
	"vaidshala/simulation/pkg/types"
)

func main() {
	fmt.Println("VAIDSHALA V-MCU Simulation Harness v1.0")
	fmt.Println("Three-Channel Safety Architecture Validation")
	fmt.Println("=============================================")
	fmt.Println()

	scenarios := patient.AllScenarios()
	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(w, "SCENARIO\tGATE\tDOSE\tDELTA\tB-RULE\tC-RULE\tSTATUS\n")
	fmt.Fprintf(w, "--------\t----\t----\t-----\t------\t------\t------\n")

	passed, failed := 0, 0
	for _, vp := range scenarios {
		engine := harness.NewVMCUEngine()
		engine.LastDoseChangeTime = time.Now().Add(-72 * time.Hour)
		result := engine.RunCycle(vp.ToTitrationInput(1))
		dose := "NO"
		if result.DoseApplied { dose = "YES" }
		ok := validate(vp.Archetype, result)
		st := "PASS"
		if !ok { st = "FAIL"; failed++ } else { passed++ }
		fmt.Fprintf(w, "%s\t%s\t%s\t%.1f\t%s\t%s\t%s\n",
			trunc(vp.Archetype, 28), result.FinalGate, dose, result.DoseDelta,
			trunc(result.PhysioRuleFired, 10), trunc(result.ProtocolRuleFired, 10), st)
	}
	w.Flush()
	fmt.Printf("\nResults: %d passed, %d failed, %d total\n", passed, failed, len(scenarios))
	if failed > 0 { fmt.Println("SAFETY FAILURES DETECTED"); os.Exit(1) }
	fmt.Println("All scenarios passed.")

	fmt.Println("\nArbiter Exhaustive Sweep (125 combinations):")
	sigs := []types.GateSignal{types.CLEAR, types.MODIFY, types.PAUSE, types.HOLD_DATA, types.HALT}
	ok := 0
	for _, a := range sigs { for _, b := range sigs { for _, c := range sigs {
		r := types.Arbitrate(types.ArbiterInput{MCUGate: a, PhysioGate: b, ProtocolGate: c})
		if r.FinalGate == types.MostRestrictive(a, types.MostRestrictive(b, c)) { ok++ }
	}}}
	fmt.Printf("Arbiter: %d/125 correct\n", ok)
}

func validate(arch string, r types.TitrationCycleResult) bool {
	switch arch {
	case "active_hypoglycaemia": return r.FinalGate == types.HALT && !r.DoseApplied
	case "aki_mid_titration": return r.FinalGate == types.HALT && !r.DoseApplied
	case "raas_creatinine_tolerance": return r.FinalGate != types.HALT
	case "data_dropout": return r.FinalGate >= types.HOLD_DATA && !r.DoseApplied
	case "non_adherent": return r.DoseDelta <= 0
	case "jcurve_ckd3b": return r.FinalGate >= types.PAUSE
	case "dual_raas": return r.FinalGate == types.HALT
	case "hyponatraemia_thiazide": return r.FinalGate == types.HALT
	case "green_trajectory": return r.FinalGate == types.CLEAR && r.DoseApplied
	case "metformin_ckd4": return r.FinalGate == types.HALT
	default: return true
	}
}

func trunc(s string, n int) string {
	if len(s) <= n { return s }
	return s[:n-2] + ".."
}
