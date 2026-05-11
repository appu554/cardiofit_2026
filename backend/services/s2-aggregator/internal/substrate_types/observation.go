// Substrate observation shape — minimal copy used by s2-aggregator's
// trajectory pipeline. This shape is intentionally simple (numeric series
// only) because Layer 1 trajectory rendering deals in numeric series per
// S2 v1.0 Part 5.1. kb-20 returns richer rows; the s2-aggregator does
// not need the richer shape for trajectory aggregation.
package substrate_types

import (
	"time"

	"github.com/google/uuid"
)

// Observation is a single observation row used to build trajectories.
// Parameter names the clinical parameter (e.g., "egfr", "dbi", "weight",
// "bp_systolic"). Value is the numeric value in the parameter's canonical
// units. Source names the substrate origin for drill-through.
//
// SOURCE OF TRUTH (for parameter shape): kb-20 ClinicalSnapshot fields +
// kb-20 longitudinal observation tables. The s2-aggregator consumes the
// numeric series via the SubstrateClient interface; it does NOT need to
// know kb-20's full GORM model.
type Observation struct {
	ID         uuid.UUID
	ResidentID uuid.UUID
	Parameter  string
	Value      float64
	Unit       string
	ObservedAt time.Time
	Source     string // e.g., "kb-20", "pathology_lab", "nursing_assessment"
	Confidence string // "high" | "moderate" | "low" — per v1.0 Part 10.3
}
