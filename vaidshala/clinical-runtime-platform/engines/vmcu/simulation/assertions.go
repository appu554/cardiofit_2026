// assertions.go provides custom assertion helpers for simulation tests.
package simulation

import (
	"fmt"
	"testing"

	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

// AssertAllTracesPresent verifies every cycle produced a SafetyTrace.
func AssertAllTracesPresent(t *testing.T, result *SimulationResult) {
	t.Helper()
	missing := 0
	for i, c := range result.Cycles {
		if c.Trace == nil {
			missing++
			if missing <= 5 { // limit noise
				t.Errorf("cycle %d (day %d, cycle %d): SafetyTrace is nil", i, c.Day, c.Cycle)
			}
		}
	}
	if missing > 0 {
		t.Errorf("total missing SafetyTraces: %d / %d", missing, len(result.Cycles))
	}
}

// AssertNoHALTInRange verifies no HALT gates in the given day range [start, end).
func AssertNoHALTInRange(t *testing.T, result *SimulationResult, startDay, endDay int) {
	t.Helper()
	for _, c := range result.CyclesInRange(startDay, endDay) {
		if c.Result != nil && c.Result.Arbiter.FinalGate == vt.GateHalt {
			t.Errorf("unexpected HALT at day %d cycle %d: rule=%s",
				c.Day, c.Cycle, c.Result.ChannelB.RuleFired)
		}
	}
}

// AssertHALTInRange verifies at least one HALT in the day range.
func AssertHALTInRange(t *testing.T, result *SimulationResult, startDay, endDay int) {
	t.Helper()
	for _, c := range result.CyclesInRange(startDay, endDay) {
		if c.Result != nil && c.Result.Arbiter.FinalGate == vt.GateHalt {
			return // found one
		}
	}
	t.Errorf("expected at least one HALT in days [%d, %d), found none", startDay, endDay)
}

// AssertGateTransition verifies a gate change occurs in the day range.
func AssertGateTransition(t *testing.T, result *SimulationResult, startDay, endDay int, from, to vt.GateSignal) {
	t.Helper()
	cycles := result.CyclesInRange(startDay, endDay)
	for i := 1; i < len(cycles); i++ {
		prev := cycles[i-1].Result.Arbiter.FinalGate
		curr := cycles[i].Result.Arbiter.FinalGate
		if prev == from && curr == to {
			return
		}
	}
	t.Errorf("expected gate transition %s → %s in days [%d, %d), not found", from, to, startDay, endDay)
}

// AssertDoseInRange verifies the dose stays within bounds during [startDay, endDay).
func AssertDoseInRange(t *testing.T, result *SimulationResult, startDay, endDay int, minDose, maxDose float64) {
	t.Helper()
	for _, c := range result.CyclesInRange(startDay, endDay) {
		if c.Dose < minDose || c.Dose > maxDose {
			t.Errorf("dose %.1f out of range [%.1f, %.1f] at day %d cycle %d",
				c.Dose, minDose, maxDose, c.Day, c.Cycle)
		}
	}
}

// AssertDoseDecreasing verifies dose is non-increasing across the day range.
func AssertDoseDecreasing(t *testing.T, result *SimulationResult, startDay, endDay int) {
	t.Helper()
	cycles := result.CyclesInRange(startDay, endDay)
	for i := 1; i < len(cycles); i++ {
		if cycles[i].Dose > cycles[i-1].Dose+0.01 { // tolerance for float comparison
			t.Errorf("dose increased from %.1f to %.1f at day %d cycle %d (expected non-increasing)",
				cycles[i-1].Dose, cycles[i].Dose, cycles[i].Day, cycles[i].Cycle)
			return
		}
	}
}

// AssertBlockedByPrefix verifies at least one cycle is blocked with given prefix.
func AssertBlockedByPrefix(t *testing.T, result *SimulationResult, prefix string) {
	t.Helper()
	blocked := result.CyclesBlockedBy(prefix)
	if len(blocked) == 0 {
		t.Errorf("expected at least one cycle blocked by %q, found none", prefix)
	}
}

// AssertFinalDoseInRange checks the final dose is within bounds.
func AssertFinalDoseInRange(t *testing.T, result *SimulationResult, minDose, maxDose float64) {
	t.Helper()
	if result.FinalDose < minDose || result.FinalDose > maxDose {
		t.Errorf("final dose %.1f out of expected range [%.1f, %.1f]",
			result.FinalDose, minDose, maxDose)
	}
}

// AssertChannelBRuleFiredInRange checks that a specific Channel B rule fired in range.
func AssertChannelBRuleFiredInRange(t *testing.T, result *SimulationResult, startDay, endDay int, ruleID string) {
	t.Helper()
	for _, c := range result.CyclesInRange(startDay, endDay) {
		if c.Result != nil && c.Result.ChannelB.RuleFired == ruleID {
			return
		}
	}
	t.Errorf("expected Channel B rule %s to fire in days [%d, %d), not found", ruleID, startDay, endDay)
}

// AssertDominantChannel checks that a specific channel dominates in the range.
func AssertDominantChannel(t *testing.T, result *SimulationResult, startDay, endDay int, channel string) {
	t.Helper()
	for _, c := range result.CyclesInRange(startDay, endDay) {
		if c.Result != nil && c.Result.Arbiter.DominantChannel == channel {
			return
		}
	}
	t.Errorf("expected channel %s to be dominant in days [%d, %d), not found", channel, startDay, endDay)
}

// PrintSimulationSummary logs the simulation summary for debugging.
func PrintSimulationSummary(t *testing.T, result *SimulationResult) {
	t.Helper()
	t.Log(result.Summary())

	// Log notable events
	for _, c := range result.Cycles {
		if c.Result == nil {
			continue
		}
		gate := c.Result.Arbiter.FinalGate
		if gate == vt.GateHalt || gate == vt.GatePause || len(c.Result.BlockedBy) > 0 {
			t.Logf("  Day %d C%d: gate=%s blocked=%s rule=%s note=%s",
				c.Day, c.Cycle, gate, c.Result.BlockedBy,
				c.Result.ChannelB.RuleFired, c.Notes)
		}
	}

	// Dose trajectory (sample every 5 days)
	t.Log("  Dose trajectory:")
	for _, dh := range result.DoseHistory() {
		if dh.Day%5 == 0 || dh.Day == TotalDays-1 {
			t.Logf("    Day %3d: %.1f", dh.Day, dh.Dose)
		}
	}
}

// CountGateOccurrences returns how many times each gate appears.
func CountGateOccurrences(result *SimulationResult) map[vt.GateSignal]int {
	counts := make(map[vt.GateSignal]int)
	for _, c := range result.Cycles {
		if c.Result != nil {
			counts[c.Result.Arbiter.FinalGate]++
		}
	}
	return counts
}

// FormatGateCounts returns a readable string of gate counts.
func FormatGateCounts(counts map[vt.GateSignal]int) string {
	return fmt.Sprintf("CLEAR=%d MODIFY=%d PAUSE=%d HALT=%d HOLD_DATA=%d",
		counts[vt.GateClear], counts[vt.GateModify],
		counts[vt.GatePause], counts[vt.GateHalt],
		counts[vt.GateHoldData])
}
