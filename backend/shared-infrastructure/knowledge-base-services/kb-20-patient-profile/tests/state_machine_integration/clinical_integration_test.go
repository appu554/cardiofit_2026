// Wave 6.2 — Clinical state-machine integration.
//
// Layer 2 doc §4.4: "internal — already covered by Wave 2 tests."
//
// This file is a doc-only stub per the Wave 6.2 plan task (which
// explicitly lists this slot as "pointer to existing Wave 2 tests").
// Maintaining the slot ensures the five-state-machine pack is visibly
// complete; future test additions for the clinical state machine should
// land here so the conventions stay consistent.
package state_machine_integration

import "testing"

func TestClinical_PointerToWave2(t *testing.T) {
	t.Log("Clinical state machine integration is exercised by Wave 2 tests:")
	t.Log("  - shared/v2_substrate/clinical_state/* (concern lifecycle)")
	t.Log("  - kb-20 internal/storage/active_concern_store_test.go")
	t.Log("  - kb-20 internal/storage/care_intensity_store_test.go")
	t.Log("  - shared/v2_substrate/scoring/cfs_capture_test.go")
	t.Log("New Clinical state machine integration coverage should land in this file.")
}
