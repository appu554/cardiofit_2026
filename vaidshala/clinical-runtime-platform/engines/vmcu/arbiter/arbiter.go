// Package arbiter implements the Safety Arbiter (SA-01).
//
// CRITICAL CONSTRAINT: Pure function. No I/O. No external dependencies. < 1ms.
//
// The arbiter applies the 1-out-of-3 (1oo3) veto rule: the most restrictive
// gate signal from any channel wins. This is the same pattern used in
// IEC 61508 safety-critical systems (nuclear, aviation).
//
// Architectural parallel: Vaidshala ICU DominanceEngine uses the same
// severity-ordered veto pattern. The arbiter is a titration-specific
// specialization of that concept.
package arbiter

import vt "vaidshala/clinical-runtime-platform/engines/vmcu/types"

// Arbitrate applies the 1oo3 veto rule: the most restrictive gate wins.
// Severity: HALT > HOLD_DATA > PAUSE > MODIFY > CLEAR
//
// This function has NO side effects, NO I/O, NO external calls.
// It is a pure function of its inputs.
func Arbitrate(input vt.ArbiterInput) vt.ArbiterOutput {
	signals := []vt.GateSignal{input.MCUGate, input.PhysioGate, input.ProtocolGate}
	final := mostRestrictive(signals)
	dominant := dominantChannel(input, final)
	rationale := buildRationale(input, final, dominant)

	return vt.ArbiterOutput{
		FinalGate:       final,
		DominantChannel: dominant,
		AllChannels:     input,
		RationaleCode:   rationale,
	}
}

// mostRestrictive picks the signal with the highest severity level.
func mostRestrictive(signals []vt.GateSignal) vt.GateSignal {
	max := vt.GateClear
	for _, s := range signals {
		if s.Level() > max.Level() {
			max = s
		}
	}
	return max
}

// dominantChannel identifies which channel drove the final decision.
// When multiple channels agree on the same level, attribution priority is:
//
//	B (physiology) > C (protocol) > A (diagnostic)
//
// Safety channels take attribution priority over diagnostic because
// physiological danger is the most fundamental safety concern.
// When all channels agree on CLEAR, returns "NONE".
func dominantChannel(input vt.ArbiterInput, final vt.GateSignal) string {
	if final == vt.GateClear {
		return "NONE"
	}
	if input.PhysioGate == final {
		return "B"
	}
	if input.ProtocolGate == final {
		return "C"
	}
	if input.MCUGate == final {
		return "A"
	}
	return "NONE"
}

// buildRationale generates a human-readable rationale code.
func buildRationale(input vt.ArbiterInput, final vt.GateSignal, dominant string) string {
	if final == vt.GateClear {
		return "ALL_CLEAR"
	}

	code := string(final) + "_BY_CH" + dominant

	// Count how many channels agree on the restrictive signal
	agreeing := 0
	if input.MCUGate == final {
		agreeing++
	}
	if input.PhysioGate == final {
		agreeing++
	}
	if input.ProtocolGate == final {
		agreeing++
	}
	if agreeing > 1 {
		code += "_UNANIMOUS"
	}

	return code
}
