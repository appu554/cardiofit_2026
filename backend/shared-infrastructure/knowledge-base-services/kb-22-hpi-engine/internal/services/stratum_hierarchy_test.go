package services

import "testing"

func TestStratumMatches(t *testing.T) {
	tests := []struct {
		name           string
		patientStratum string
		nodeStrata     []string
		want           bool
	}{
		// Direct match
		{"direct match", "DM_HTN", []string{"DM_HTN"}, true},
		// Ancestor walk: DM_HTN → parent DM_HTN_base
		{"child matches base", "DM_HTN", []string{"DM_HTN_base"}, true},
		// 2-level walk: DM_HTN_CKD → DM_HTN → DM_HTN_base
		{"grandchild matches base", "DM_HTN_CKD", []string{"DM_HTN_base"}, true},
		// 3-level walk: DM_HTN_CKD_HF → DM_HTN_CKD → DM_HTN → DM_HTN_base
		{"great-grandchild matches base", "DM_HTN_CKD_HF", []string{"DM_HTN_base"}, true},
		// Nested: DM_HTN_CKD is child of DM_HTN
		{"child matches parent", "DM_HTN_CKD", []string{"DM_HTN"}, true},
		// DM_HTN_CKD_HF walks to DM_HTN_CKD
		{"grandchild matches mid-level", "DM_HTN_CKD_HF", []string{"DM_HTN_CKD"}, true},
		// Parent cannot match child
		{"parent does not match child", "DM_HTN", []string{"DM_HTN_CKD"}, false},
		// DM_ONLY → DM_HTN_base (sibling, not under DM_HTN)
		{"DM_ONLY matches base", "DM_ONLY", []string{"DM_HTN_base"}, true},
		{"DM_ONLY does not match DM_HTN", "DM_ONLY", []string{"DM_HTN"}, false},
		// HTN_ONLY → DM_HTN_base
		{"HTN_ONLY matches base", "HTN_ONLY", []string{"DM_HTN_base"}, true},
		{"HTN_ONLY does not match DM_HTN", "HTN_ONLY", []string{"DM_HTN"}, false},
		// Unknown stratum
		{"unknown stratum", "NONE", []string{"DM_HTN_base"}, false},
		// Empty strata list
		{"empty strata list", "DM_HTN", []string{}, false},
		// Multiple strata in list — match any
		{"multi-strata direct", "DM_ONLY", []string{"DM_HTN", "DM_ONLY"}, true},
		{"multi-strata ancestor", "DM_HTN_CKD_HF", []string{"DM_ONLY", "DM_HTN"}, true},

		// V4: CKD substaging (KDIGO 2024) — children of DM_HTN_CKD
		{"CKD_3a matches DM_HTN_CKD", "DM_HTN_CKD_3a", []string{"DM_HTN_CKD"}, true},
		{"CKD_3b matches DM_HTN_CKD", "DM_HTN_CKD_3b", []string{"DM_HTN_CKD"}, true},
		{"CKD_A3 matches DM_HTN_CKD", "DM_HTN_CKD_A3", []string{"DM_HTN_CKD"}, true},
		{"CKD_3a matches DM_HTN", "DM_HTN_CKD_3a", []string{"DM_HTN"}, true},
		{"CKD_3a matches DM_HTN_base", "DM_HTN_CKD_3a", []string{"DM_HTN_base"}, true},
		{"CKD_3a does not match DM_ONLY", "DM_HTN_CKD_3a", []string{"DM_ONLY"}, false},
		{"CKD_3a does not match DM_HTN_CKD_HF", "DM_HTN_CKD_3a", []string{"DM_HTN_CKD_HF"}, false},

		// V4: HF subtyping (ESC 2024) — children of DM_HTN_CKD_HF (4-level chain)
		{"HF_REDUCED matches DM_HTN_CKD_HF", "DM_HTN_CKD_HF_REDUCED", []string{"DM_HTN_CKD_HF"}, true},
		{"HF_PRESERVED matches DM_HTN_CKD_HF", "DM_HTN_CKD_HF_PRESERVED", []string{"DM_HTN_CKD_HF"}, true},
		{"HF_REDUCED matches DM_HTN_CKD", "DM_HTN_CKD_HF_REDUCED", []string{"DM_HTN_CKD"}, true},
		{"HF_REDUCED matches DM_HTN", "DM_HTN_CKD_HF_REDUCED", []string{"DM_HTN"}, true},
		{"HF_REDUCED matches DM_HTN_base", "DM_HTN_CKD_HF_REDUCED", []string{"DM_HTN_base"}, true},
		{"HF_PRESERVED matches DM_HTN_base", "DM_HTN_CKD_HF_PRESERVED", []string{"DM_HTN_base"}, true},
		{"HF_REDUCED does not match DM_ONLY", "DM_HTN_CKD_HF_REDUCED", []string{"DM_ONLY"}, false},
		{"HF_REDUCED does not match CKD_3a", "DM_HTN_CKD_HF_REDUCED", []string{"DM_HTN_CKD_3a"}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StratumMatches(tt.patientStratum, tt.nodeStrata)
			if got != tt.want {
				t.Errorf("StratumMatches(%q, %v) = %v, want %v",
					tt.patientStratum, tt.nodeStrata, got, tt.want)
			}
		})
	}
}

// TestHierarchyCoversAllKnownStrata validates that the stratumParent map
// covers every stratum constant known in the system. If KB-20 adds a new
// stratum, this test will fail until the hierarchy is updated.
// Mirrors KB-20 constants from kb-20-patient-profile/internal/models/stratum.go.
func TestHierarchyCoversAllKnownStrata(t *testing.T) {
	// These must match KB-20's exported stratum constants.
	// Update this list when KB-20 adds new strata.
	kb20Strata := []string{
		// V3 strata
		"DM_HTN",
		"DM_HTN_CKD",
		"DM_HTN_CKD_HF",
		"DM_ONLY",
		"HTN_ONLY",
		// V4: CKD substaging (KDIGO 2024)
		"DM_HTN_CKD_3a",
		"DM_HTN_CKD_3b",
		"DM_HTN_CKD_A3",
		// V4: HF subtyping (ESC 2024)
		"DM_HTN_CKD_HF_REDUCED",
		"DM_HTN_CKD_HF_PRESERVED",
	}
	for _, s := range kb20Strata {
		if _, ok := stratumParent[s]; !ok {
			t.Errorf("KB-20 stratum %q is missing from stratumParent hierarchy map — add it to stratum_hierarchy.go", s)
		}
	}
}
