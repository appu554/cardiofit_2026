// Wave 6.2 — Monitoring state-machine integration test.
//
// Layer 2 doc §4.3: "expected-vs-received observation gap detection. A
// monitoring trajectory that misses N consecutive expected captures
// transitions to 'abnormal'."
package state_machine_integration

import (
	"testing"
	"time"
)

// monitoringTrajectory is the mock-Layer-3 trajectory state. The
// substrate exposes the raw observation timestamps; Layer 3's monitoring
// state machine computes gap detection on top.
type monitoringTrajectory struct {
	ResidentRef       string
	ExpectedCadence   time.Duration // e.g. 24h for daily weight
	GapToleranceCount int           // how many consecutive misses → abnormal
	LastObservations  []time.Time
}

// State returns "abnormal" when the most recent gap-tolerance window of
// expected captures has zero received observations.
func (m monitoringTrajectory) State(now time.Time) string {
	thresholdStart := now.Add(-time.Duration(m.GapToleranceCount) * m.ExpectedCadence)
	for _, t := range m.LastObservations {
		if t.After(thresholdStart) {
			return "normal"
		}
	}
	return "abnormal"
}

func TestMonitoring_GapDetectionTransitionsToAbnormal(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	traj := monitoringTrajectory{
		ResidentRef:       "resident-1",
		ExpectedCadence:   24 * time.Hour,
		GapToleranceCount: 3,
		// Last observation 5 days ago — gap exceeds tolerance.
		LastObservations: []time.Time{now.Add(-5 * 24 * time.Hour)},
	}
	if got := traj.State(now); got != "abnormal" {
		t.Fatalf("want abnormal after 5-day gap (tolerance 3); got %s", got)
	}
}

func TestMonitoring_RecentObservationKeepsNormal(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	traj := monitoringTrajectory{
		ResidentRef:       "resident-1",
		ExpectedCadence:   24 * time.Hour,
		GapToleranceCount: 3,
		LastObservations:  []time.Time{now.Add(-12 * time.Hour)},
	}
	if got := traj.State(now); got != "normal" {
		t.Fatalf("want normal with 12h-old obs; got %s", got)
	}
}

func TestMonitoring_BoundaryAtTolerance(t *testing.T) {
	now := time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC)
	// Observation exactly at the boundary (3 cadences ago) is OUTSIDE the
	// "after" window → abnormal. Just-inside (3 cadences minus 1ms) →
	// normal.
	traj := monitoringTrajectory{
		ResidentRef:       "resident-1",
		ExpectedCadence:   24 * time.Hour,
		GapToleranceCount: 3,
		LastObservations:  []time.Time{now.Add(-3 * 24 * time.Hour)},
	}
	if got := traj.State(now); got != "abnormal" {
		t.Fatalf("at the exact boundary: want abnormal; got %s", got)
	}
	traj.LastObservations = []time.Time{now.Add(-3*24*time.Hour + time.Millisecond)}
	if got := traj.State(now); got != "normal" {
		t.Fatalf("just inside boundary: want normal; got %s", got)
	}
}
