// Goals-of-care and care intensity substrate shapes — S2 v1.0 Part 9 +
// Addendum Part 4.5 (GoC + care intensity as shared primitive inherited
// across all layers).
//
// VOCABULARY DISCOVERY: The Task 5 build brief speculated a separate
// goals-of-care state machine in kb-20 with vocabulary
// `curative | rehabilitative | maintenance | comfort_focused |
// palliative | end_of_life`. Inspection of kb-20 (and of
// shared/v2_substrate/models/care_intensity.go) shows NO such separate
// state machine exists in Phase 1 — kb-20's only documented
// care-philosophy substrate is the four-tag CareIntensity Wave 2.4
// enum:
//
//	active_treatment | rehabilitation | comfort_focused | palliative
//
// kb-32's appropriateness scorer (substrate_scorer.go
// scoreGoalsOfCareAlignment) reads `snap.CareIntensity` using the
// LEGACY short-form values `active | comfort | palliative | end_of_life`
// — translation between the legacy short forms and the Wave 2.4 tags is
// handled by LegacyCareIntensityForTag in the canonical models package.
//
// For the S2 GoC panel we therefore treat care_intensity as the
// goals-of-care substrate signal (per CAPE Phase 2-completion Task 2's
// scoreGoalsOfCareAlignment doing the same), surface the Wave 2.4 tag
// vocabulary, and tolerate the legacy short forms in conflict-detection
// equality checks. When kb-20 grows a distinct GoC state machine with
// its own vocabulary, this file is the integration point.
//
// SOURCE OF TRUTH (care intensity tags):
// shared/v2_substrate/models/care_intensity.go
// (CareIntensityTagActiveTreatment / Rehabilitation / ComfortFocused /
// Palliative).
package substrate_types

import (
	"time"

	"github.com/google/uuid"
)

// GoalsOfCareEntry is the s2-side projection of a documented
// goals-of-care state. In Phase 1 this is sourced from the kb-20
// care_intensity_history table (see vocabulary discovery note above);
// when kb-20 grows a distinct GoC state machine the shape stays the
// same — only the SubstrateClient adapter changes.
//
// State carries the care-intensity tag (active_treatment |
// rehabilitation | comfort_focused | palliative). FreshnessFlag is
// set by the panel builder (not the substrate adapter) based on
// EffectiveFrom age.
type GoalsOfCareEntry struct {
	State         string
	EffectiveFrom time.Time
	EffectiveTo   *time.Time
	DocumentedBy  uuid.UUID
	FreshnessFlag bool
	SubstrateID   uuid.UUID
}

// CareIntensityEntry is the s2-side projection of a kb-20
// care_intensity_history row.
type CareIntensityEntry struct {
	Tag           string
	EffectiveDate time.Time
	DocumentedBy  uuid.UUID
	FreshnessFlag bool
	SubstrateID   uuid.UUID
}

// Goals-of-care / care-intensity state constants. The Phase 1
// vocabulary mirrors kb-20's Wave 2.4 CareIntensityTag* set; legacy
// short forms are accepted in conflict-detection equality.
//
// SOURCE OF TRUTH: shared/v2_substrate/models/care_intensity.go.
const (
	// GoCStateActiveTreatment — curative / disease-modifying treatment
	// posture. Maps to kb-20 CareIntensityTagActiveTreatment and to
	// legacy short form "active".
	GoCStateActiveTreatment = "active_treatment"

	// GoCStateRehabilitation — restorative posture (e.g., post-acute
	// rehab). Maps to kb-20 CareIntensityTagRehabilitation.
	GoCStateRehabilitation = "rehabilitation"

	// GoCStateComfortFocused — comfort-oriented posture (kb-20 Wave 2.4
	// canonical form). Maps to legacy short form "comfort".
	GoCStateComfortFocused = "comfort_focused"

	// GoCStatePalliative — palliative posture. Maps verbatim to kb-20
	// CareIntensityTagPalliative.
	GoCStatePalliative = "palliative"

	// GoCStateEndOfLife — terminal-care posture. NOT in kb-20's Wave 2.4
	// closed set; surfaced here only to honour the legacy short-form
	// value that kb-32's scoreGoalsOfCareAlignment recognises. When a
	// resident's substrate carries "end_of_life" it indicates a
	// pre-Wave-2.4 record or a future state-machine extension; the
	// panel surfaces the value verbatim.
	GoCStateEndOfLife = "end_of_life"
)

// Care intensity tag constants — alias the kb-20 Wave 2.4 vocabulary.
// Held separately from GoC* so that future divergence (if kb-20 grows
// a distinct GoC enum) does not require renaming.
const (
	CareIntensityTagActiveTreatment = "active_treatment"
	CareIntensityTagRehabilitation  = "rehabilitation"
	CareIntensityTagComfortFocused  = "comfort_focused"
	CareIntensityTagPalliative      = "palliative"
)
