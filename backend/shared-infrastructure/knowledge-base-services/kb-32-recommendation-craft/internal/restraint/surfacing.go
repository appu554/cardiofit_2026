// surfacing.go — data-shape aggregation layer for restraint signals.
//
// VisibilityClass: AD — restraint signals per Guidelines §10
//
// SurfaceData is the inline-with-action data layer (no UI rendering).
// Phase 2b ships data shaping; UI rendering is deferred to Phase 4 Layer 4 surfaces.
package restraint

// SurfaceData is the aggregated view of all restraint signals for a given
// ClinicalSnapshot.  It is the data structure consumed by downstream UI
// layers and recommendation formatting stages.
type SurfaceData struct {
	// SignalCount is the number of signals that fired.
	SignalCount int
	// HighestSeverity is "red" if any signal is Red, "amber" if only Amber
	// signals fired, or "" if no signals fired.
	HighestSeverity Severity
	// Signals holds the full ordered list of signals; nil when empty.
	Signals []Signal
}

// Surface aggregates signals into the SurfaceData shape consumed by future
// UI layers.  When signals is nil or empty it returns a zero-value SurfaceData
// with SignalCount=0 and empty HighestSeverity, signalling "no restraint context".
func Surface(signals []Signal) SurfaceData {
	if len(signals) == 0 {
		return SurfaceData{}
	}

	sd := SurfaceData{
		SignalCount:     len(signals),
		Signals:         signals,
		HighestSeverity: SeverityAmber, // default; upgraded to Red below if needed
	}

	for _, s := range signals {
		if s.Severity == SeverityRed {
			sd.HighestSeverity = SeverityRed
			break
		}
	}

	return sd
}
