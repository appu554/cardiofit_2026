package arbiter

import (
	"testing"

	vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"
)

func TestAllClear(t *testing.T) {
	result := Arbitrate(vt.ArbiterInput{
		MCUGate:      vt.GateClear,
		PhysioGate:   vt.GateClear,
		ProtocolGate: vt.GateClear,
	})
	if result.FinalGate != vt.GateClear {
		t.Errorf("all CLEAR should yield CLEAR, got %s", result.FinalGate)
	}
	if result.DominantChannel != "NONE" {
		t.Errorf("all CLEAR dominant should be NONE, got %s", result.DominantChannel)
	}
	if result.RationaleCode != "ALL_CLEAR" {
		t.Errorf("rationale should be ALL_CLEAR, got %s", result.RationaleCode)
	}
}

func TestChannelBHalt(t *testing.T) {
	result := Arbitrate(vt.ArbiterInput{
		MCUGate:      vt.GateClear,
		PhysioGate:   vt.GateHalt,
		ProtocolGate: vt.GateClear,
	})
	if result.FinalGate != vt.GateHalt {
		t.Errorf("B=HALT should yield HALT, got %s", result.FinalGate)
	}
	if result.DominantChannel != "B" {
		t.Errorf("dominant should be B, got %s", result.DominantChannel)
	}
}

func TestChannelCPause(t *testing.T) {
	result := Arbitrate(vt.ArbiterInput{
		MCUGate:      vt.GateModify,
		PhysioGate:   vt.GateClear,
		ProtocolGate: vt.GatePause,
	})
	if result.FinalGate != vt.GatePause {
		t.Errorf("C=PAUSE should dominate A=MODIFY, got %s", result.FinalGate)
	}
	if result.DominantChannel != "C" {
		t.Errorf("dominant should be C, got %s", result.DominantChannel)
	}
}

func TestAllHalt_PhysioTakesAttribution(t *testing.T) {
	result := Arbitrate(vt.ArbiterInput{
		MCUGate:      vt.GateHalt,
		PhysioGate:   vt.GateHalt,
		ProtocolGate: vt.GateHalt,
	})
	if result.FinalGate != vt.GateHalt {
		t.Errorf("all HALT should yield HALT, got %s", result.FinalGate)
	}
	if result.DominantChannel != "B" {
		t.Errorf("all HALT → dominant should be B (physiology priority), got %s", result.DominantChannel)
	}
}

func TestHoldDataFromChannelA(t *testing.T) {
	result := Arbitrate(vt.ArbiterInput{
		MCUGate:      vt.GateHoldData,
		PhysioGate:   vt.GatePause,
		ProtocolGate: vt.GateClear,
	})
	if result.FinalGate != vt.GateHoldData {
		t.Errorf("HOLD_DATA > PAUSE, got %s", result.FinalGate)
	}
	if result.DominantChannel != "A" {
		t.Errorf("dominant should be A, got %s", result.DominantChannel)
	}
}

func TestHaltBeatsHoldData(t *testing.T) {
	result := Arbitrate(vt.ArbiterInput{
		MCUGate:      vt.GateHoldData,
		PhysioGate:   vt.GateClear,
		ProtocolGate: vt.GateHalt,
	})
	if result.FinalGate != vt.GateHalt {
		t.Errorf("HALT > HOLD_DATA, got %s", result.FinalGate)
	}
	if result.DominantChannel != "C" {
		t.Errorf("dominant should be C, got %s", result.DominantChannel)
	}
}

func TestModifyFromChannelA(t *testing.T) {
	result := Arbitrate(vt.ArbiterInput{
		MCUGate:      vt.GateModify,
		PhysioGate:   vt.GateClear,
		ProtocolGate: vt.GateClear,
	})
	if result.FinalGate != vt.GateModify {
		t.Errorf("A=MODIFY should yield MODIFY, got %s", result.FinalGate)
	}
	if result.DominantChannel != "A" {
		t.Errorf("dominant should be A, got %s", result.DominantChannel)
	}
}

func TestSeverityHierarchy(t *testing.T) {
	signals := []vt.GateSignal{
		vt.GateClear,
		vt.GateModify,
		vt.GatePause,
		vt.GateHoldData,
		vt.GateHalt,
	}
	for i := 0; i < len(signals)-1; i++ {
		if signals[i].Level() >= signals[i+1].Level() {
			t.Errorf("severity ordering broken: %s (level %d) should be < %s (level %d)",
				signals[i], signals[i].Level(), signals[i+1], signals[i+1].Level())
		}
	}
}

func TestExhaustiveGateCombinations(t *testing.T) {
	// Full 5x5x5 = 125 combinations verifying the most-restrictive-wins invariant.
	signals := []vt.GateSignal{
		vt.GateClear, vt.GateModify, vt.GatePause, vt.GateHoldData, vt.GateHalt,
	}

	for _, a := range signals {
		for _, b := range signals {
			for _, c := range signals {
				result := Arbitrate(vt.ArbiterInput{
					MCUGate:      a,
					PhysioGate:   b,
					ProtocolGate: c,
				})

				// Invariant 1: FinalGate must be the most restrictive of the three
				maxLevel := a.Level()
				if b.Level() > maxLevel {
					maxLevel = b.Level()
				}
				if c.Level() > maxLevel {
					maxLevel = c.Level()
				}
				if result.FinalGate.Level() != maxLevel {
					t.Errorf("A=%s B=%s C=%s: FinalGate=%s (level %d) != max level %d",
						a, b, c, result.FinalGate, result.FinalGate.Level(), maxLevel)
				}

				// Invariant 2: If all CLEAR, dominant must be NONE
				if a == vt.GateClear && b == vt.GateClear && c == vt.GateClear {
					if result.DominantChannel != "NONE" {
						t.Errorf("all CLEAR: dominant should be NONE, got %s", result.DominantChannel)
					}
				}

				// Invariant 3: DominantChannel must actually have the final gate signal
				switch result.DominantChannel {
				case "A":
					if a != result.FinalGate {
						t.Errorf("A=%s B=%s C=%s: dominant=A but A gate=%s != final=%s",
							a, b, c, a, result.FinalGate)
					}
				case "B":
					if b != result.FinalGate {
						t.Errorf("A=%s B=%s C=%s: dominant=B but B gate=%s != final=%s",
							a, b, c, b, result.FinalGate)
					}
				case "C":
					if c != result.FinalGate {
						t.Errorf("A=%s B=%s C=%s: dominant=C but C gate=%s != final=%s",
							a, b, c, c, result.FinalGate)
					}
				case "NONE":
					// valid only when all CLEAR
				}

				// Invariant 4: Attribution priority B > C > A
				if b == result.FinalGate && result.FinalGate != vt.GateClear {
					if result.DominantChannel != "B" {
						t.Errorf("A=%s B=%s C=%s: B matches final=%s but dominant=%s (should be B)",
							a, b, c, result.FinalGate, result.DominantChannel)
					}
				}
			}
		}
	}
}

func TestGateSignalIsBlocking(t *testing.T) {
	if vt.GateClear.IsBlocking() {
		t.Error("CLEAR should not be blocking")
	}
	if vt.GateModify.IsBlocking() {
		t.Error("MODIFY should not be blocking")
	}
	if !vt.GatePause.IsBlocking() {
		t.Error("PAUSE should be blocking")
	}
	if !vt.GateHoldData.IsBlocking() {
		t.Error("HOLD_DATA should be blocking")
	}
	if !vt.GateHalt.IsBlocking() {
		t.Error("HALT should be blocking")
	}
}
