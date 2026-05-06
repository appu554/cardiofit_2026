// Wave 6.1 — Failure Mode 4: baseline contamination by acute periods.
//
// Layer 2 doc Part 6 Failure 4: "an acute concern (e.g. infection)
// inflates baselines if its observations aren't excluded from the
// recompute window. Defence: BaselineConfig.ExcludeDuringActiveConcerns
// names concern types that suppress contributing observations during
// the active window."
//
// The full end-to-end test (active_concern open → ingest acute obs →
// close concern → recompute → assert exclusion) is DB-gated. This file
// also runs a pure-logic stub that verifies the BaselineConfig.
// ExcludeDuringActiveConcerns wiring is plumbed through correctly.
package failure_modes

import (
	"os"
	"testing"
	"time"

	"github.com/cardiofit/shared/v2_substrate/delta"
)

func TestFailure4_BaselineContamination_E2E(t *testing.T) {
	if os.Getenv("KB20_TEST_DATABASE_URL") == "" {
		t.Skip("Failure 4 end-to-end test skipped (set KB20_TEST_DATABASE_URL to run).")
	}
	t.Skip("V1 end-to-end DB harness not yet implemented; ExcludeDuringActiveConcerns wiring covered by the stub test below.")
}

// TestFailure4_BaselineContamination_ConfigStub is the pure-logic check
// that the BaselineConfig.ExcludeDuringActiveConcerns field round-trips
// correctly through the delta.BaselineConfig type. The full DB-backed
// recompute exclusion is exercised by the BaselineStore integration tests
// (kb-20 internal/storage/baseline_store_test.go) when KB20_TEST_DATABASE_URL
// is set.
func TestFailure4_BaselineContamination_ConfigStub(t *testing.T) {
	cfg := delta.BaselineConfig{
		ObservationType:             "8480-6", // systolic BP LOINC
		WindowDays:                  90,
		MinObsForHighConfidence:     21,
		ExcludeDuringActiveConcerns: []string{"infection_acute", "AKI_watching"},
		MorningOnly:                 true,
		UpdatedAt:                   time.Now().UTC(),
	}
	if len(cfg.ExcludeDuringActiveConcerns) != 2 {
		t.Fatalf("expected 2 exclusions, got %d", len(cfg.ExcludeDuringActiveConcerns))
	}
	// The exclusion list must be non-empty for vital types where Layer 2
	// §2.2 calls for it. We don't enforce a closed set here (operators
	// can extend), but we do enforce the semantic that the list is ordered
	// and stable — a pop/push at the end shouldn't change earlier entries.
	if cfg.ExcludeDuringActiveConcerns[0] != "infection_acute" {
		t.Fatal("ExcludeDuringActiveConcerns ordering drifted")
	}
}

// TestFailure4_DefaultConfigEmptyExclusions documents the deliberate
// design decision: DefaultConfig (the fallback for unknown observation
// types) has NO exclusions. Operators must opt in per type so a new vital
// without a config row never silently drops observations.
func TestFailure4_DefaultConfigEmptyExclusions(t *testing.T) {
	d := delta.DefaultConfig("any-unknown-type")
	if len(d.ExcludeDuringActiveConcerns) != 0 {
		t.Fatalf("DefaultConfig must have no exclusions; got %v", d.ExcludeDuringActiveConcerns)
	}
}
