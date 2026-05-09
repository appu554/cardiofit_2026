package pattern_detection

import "testing"

// ---------------------------------------------------------------------------
// Plan-verbatim tests (Task 9 — surveillance)
// ---------------------------------------------------------------------------

func TestSurveillance_FlagsAboveP95IndividualQueries(t *testing.T) {
	if !DetectSurveillanceP95(IndividualQueryRate{Employer: "A", QueryCountP95: 50, EmployerQueryCount: 120}) {
		t.Errorf("expected p95 flag")
	}
}

func TestSurveillance_FlagsReidentificationRisk(t *testing.T) {
	if !DetectReidentificationRisk(AggregationSubset{PharmacistCount: 3}, 5) {
		t.Errorf("subset of 3 below floor of 5 should flag")
	}
}

// ---------------------------------------------------------------------------
// Augmentations
// ---------------------------------------------------------------------------

// TestSurveillanceP95_BoundaryEqualsDoesNotFlag documents that DetectSurveillanceP95
// uses strict greater-than. An employer whose query count equals the P95
// exactly is at, but not above, the threshold and must NOT be flagged.
func TestSurveillanceP95_BoundaryEqualsDoesNotFlag(t *testing.T) {
	r := IndividualQueryRate{
		Employer:           "B",
		QueryCountP95:      50,
		EmployerQueryCount: 50, // exactly at P95
	}
	if DetectSurveillanceP95(r) {
		t.Errorf("EmployerQueryCount == QueryCountP95 should NOT flag (strict > required)")
	}
}

// TestReidentificationRisk_BoundaryAtFloor documents that DetectReidentificationRisk
// uses strict less-than. A cohort of exactly floor pharmacists meets the minimum
// permissible size and must NOT be flagged.
func TestReidentificationRisk_BoundaryAtFloor(t *testing.T) {
	if DetectReidentificationRisk(AggregationSubset{PharmacistCount: 5}, 5) {
		t.Errorf("PharmacistCount == floor should NOT flag (strict < required)")
	}
}
