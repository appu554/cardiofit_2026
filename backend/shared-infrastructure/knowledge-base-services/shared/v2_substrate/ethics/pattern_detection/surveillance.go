package pattern_detection

// IndividualQueryRate captures an employer's individual-pharmacist query
// volume relative to the platform-wide 95th percentile (Guidelines §9.7).
type IndividualQueryRate struct {
	// Employer is the identifier of the employing organisation.
	Employer string

	// EmployerQueryCount is the number of individual-pharmacist queries this
	// employer submitted in the observation window.
	EmployerQueryCount int

	// QueryCountP95 is the 95th-percentile individual-pharmacist query count
	// across all employers during the same observation window.
	QueryCountP95 int
}

// DetectSurveillanceP95 flags an employer whose individual-pharmacist query
// volume exceeds the platform-wide 95th percentile per Guidelines §9.7.
//
// The comparison is strict greater-than: an employer whose count exactly equals
// the P95 value is NOT flagged (see TestSurveillanceP95_BoundaryEqualsDoesNotFlag).
func DetectSurveillanceP95(r IndividualQueryRate) bool {
	return r.EmployerQueryCount > r.QueryCountP95
}

// AggregationSubset describes the size of the pharmacist cohort in an
// aggregated visibility query. Aggregation with very small cohorts risks
// re-identification of individuals through combination of disclosed attributes.
type AggregationSubset struct {
	// PharmacistCount is the number of distinct pharmacists whose data is
	// included in the aggregation result.
	PharmacistCount int
}

// DetectReidentificationRisk flags an aggregation query whose cohort size
// falls below the configured floor per Guidelines §9.7.
//
// Risk is flagged when PharmacistCount < floor (strict less-than). A cohort
// exactly equal to the floor is NOT flagged — the floor is the minimum
// permissible cohort size, and meeting it exactly is acceptable.
// See TestReidentificationRisk_BoundaryAtFloor.
func DetectReidentificationRisk(s AggregationSubset, floor int) bool {
	return s.PharmacistCount < floor
}
