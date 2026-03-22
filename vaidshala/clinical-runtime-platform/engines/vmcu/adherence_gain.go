package vmcu

// AdherenceToGainFactor converts a KB-21 adherence score (0.0–1.0) to a
// gain factor multiplier for V-MCU titration modulation.
//
// Thresholds (from Clinical Signal Capture Layer spec):
//   - adherence >= 0.70 → 1.0 (normal titration)
//   - adherence 0.40–0.70 → 0.5 (dampened titration)
//   - adherence < 0.40 → 0.0 (suppress titration entirely)
func AdherenceToGainFactor(adherence float64) float64 {
	switch {
	case adherence >= 0.70:
		return 1.0
	case adherence >= 0.40:
		return 0.5
	default:
		return 0.0
	}
}
