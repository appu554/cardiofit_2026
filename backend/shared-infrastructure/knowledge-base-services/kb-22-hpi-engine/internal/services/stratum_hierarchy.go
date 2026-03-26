package services

// stratumParent maps each stratum to its direct parent in the hierarchy.
//
// Hierarchy (nested):
//
//	DM_HTN_base
//	├── DM_HTN
//	│   └── DM_HTN_CKD
//	│       ├── DM_HTN_CKD_HF
//	│       │   ├── DM_HTN_CKD_HF_REDUCED   (V4: HF subtyping — ESC 2024)
//	│       │   └── DM_HTN_CKD_HF_PRESERVED  (V4: HF subtyping — ESC 2024)
//	│       ├── DM_HTN_CKD_3a               (V4: CKD substaging — KDIGO 2024)
//	│       ├── DM_HTN_CKD_3b               (V4: CKD substaging — KDIGO 2024)
//	│       └── DM_HTN_CKD_A3               (V4: CKD substaging — KDIGO 2024)
//	├── DM_ONLY
//	└── HTN_ONLY
var stratumParent = map[string]string{
	// V3 strata
	"DM_HTN":        "DM_HTN_base",
	"DM_HTN_CKD":    "DM_HTN",
	"DM_HTN_CKD_HF": "DM_HTN_CKD",
	"DM_ONLY":       "DM_HTN_base",
	"HTN_ONLY":      "DM_HTN_base",
	// V4: CKD substaging (KDIGO 2024)
	"DM_HTN_CKD_3a": "DM_HTN_CKD",
	"DM_HTN_CKD_3b": "DM_HTN_CKD",
	"DM_HTN_CKD_A3": "DM_HTN_CKD",
	// V4: HF subtyping (ESC 2024)
	"DM_HTN_CKD_HF_REDUCED":   "DM_HTN_CKD_HF",
	"DM_HTN_CKD_HF_PRESERVED": "DM_HTN_CKD_HF",
}

const maxAncestorDepth = 4

// StratumMatches returns true if patientStratum is accepted by any stratum
// in nodeStrata, accounting for the hierarchy. A node declaring "DM_HTN_base"
// accepts any descendant (DM_HTN, DM_HTN_CKD, DM_HTN_CKD_HF, DM_ONLY, HTN_ONLY).
// A node declaring "DM_HTN" accepts DM_HTN, DM_HTN_CKD, and DM_HTN_CKD_HF
// but NOT DM_ONLY or HTN_ONLY.
func StratumMatches(patientStratum string, nodeStrata []string) bool {
	for _, supported := range nodeStrata {
		if supported == patientStratum {
			return true
		}
	}
	// Walk up the ancestor chain
	current := patientStratum
	for depth := 0; depth < maxAncestorDepth; depth++ {
		parent, ok := stratumParent[current]
		if !ok {
			return false
		}
		for _, supported := range nodeStrata {
			if supported == parent {
				return true
			}
		}
		current = parent
	}
	return false
}
