package kpis

// ConfirmedActivity represents a single confirmed CPD activity.
//
// Naming note: the plan draft used lowercase "confirmedActivity" but Go convention
// for cross-package use requires an exported identifier. ConfirmedActivity is used
// throughout this package and its callers.
//
// VisibilityClass:
//   - Completion status / hours by category: WO — employer can see compliance totals.
//   - Any reflective content attached to an activity: POA — pharmacist alone.
type ConfirmedActivity struct {
	// Category is the CPD category label (e.g. "clinical", "communication", "management").
	Category string

	// Hours is the number of CPD hours credited for this activity.
	Hours float64
}

// CPDHoursByCategory sums CPD hours for each category across the provided
// confirmed activities.
//
// Returns an empty (non-nil) map for empty input.
func CPDHoursByCategory(acts []ConfirmedActivity) map[string]float64 {
	totals := make(map[string]float64)
	for _, a := range acts {
		totals[a.Category] += a.Hours
	}
	return totals
}
