package models

// OverlapBand is the (floor, ceiling) outside which a propensity value is considered
// to fail the overlap check. Populated per cohort from cate_parameters.yaml and
// consumed by the propensity-based overlap diagnostic in Task 4.
//
// Spec §6.1: "This is a hard guard and cannot be disabled." A propensity value
// outside [floor, ceiling] short-circuits CATE estimation to OverlapBelowFloor or
// OverlapAboveCeiling, regardless of the CATE point value.
type OverlapBand struct {
	Floor   float64
	Ceiling float64
}
