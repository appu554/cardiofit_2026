// Wave 6.1 — Failure Mode 1: compute-on-write performance.
//
// Layer 2 doc Part 6 Failure 1: "the substrate computes baselines and
// scoring on-write rather than on-read; under sustained load this could
// pile up. Defence: outbox-driven async recompute with p95 lag <30s on
// 2,000 obs/day/facility × 5 facilities concurrent".
//
// This file documents the target and ships a DB-gated test that skips
// cleanly without KB20_TEST_DATABASE_URL — the actual load run requires a
// real PostgreSQL + Kafka stack which is provisioned at V1.
package failure_modes

import (
	"os"
	"testing"
)

// TargetP95LagSeconds is the Layer 2 doc Failure 1 target.
const TargetP95LagSeconds = 30

// TargetObsPerDayPerFacility is the load profile from the failure-mode
// statement. Combined with TargetConcurrentFacilities below, the total is
// 10,000 obs/day across the cluster.
const (
	TargetObsPerDayPerFacility = 2000
	TargetConcurrentFacilities = 5
)

// TestFailure1_ComputeOnWritePerf is DB-gated. With KB20_TEST_DATABASE_URL
// unset it skips and documents the SLO. With it set, it would drive the
// observation-ingest endpoint at the target rate and assert p95 recompute
// lag from outbox emission to baseline persistence.
func TestFailure1_ComputeOnWritePerf(t *testing.T) {
	if os.Getenv("KB20_TEST_DATABASE_URL") == "" {
		t.Skipf("Failure 1 load test skipped (set KB20_TEST_DATABASE_URL to run). Target: p95 recompute lag <%ds at %d obs/day/facility × %d facilities concurrent.",
			TargetP95LagSeconds, TargetObsPerDayPerFacility, TargetConcurrentFacilities)
	}
	// V1: actual load harness goes here. Drives kb-20 with synthetic
	// observations, monitors outbox processing lag via the Kafka-style
	// metric exposed by V2SubstrateStore, asserts p95 < TargetP95LagSeconds.
	t.Skip("V1 load harness not yet implemented; the SLO target is documented above.")
}
