package clinical_state

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

// CareIntensityCascade is one worklist hint produced by a care-intensity
// transition. Per Layer 2 doc §2.4 — care intensity transitions propagate
// through the substrate (active concerns may resolve, recommendations may
// be re-evaluated, monitoring plans may be revised, consent may need
// refresh). The cascade list is intentionally hint-shaped: the engine
// declares what should be reviewed; persistence of those reviews is
// downstream work (Layer 3 worklist surfaces, recommendation lifecycle).
//
// Kind values are a closed vocabulary so downstream consumers can
// pattern-match (e.g. ACOP pharmacist queue for review_preventive_medications).
type CareIntensityCascade struct {
	Kind   string `json:"kind"`
	Reason string `json:"reason"`
}

// Care-intensity cascade kinds. The set is closed; downstream worklist
// routing keys off these strings.
const (
	CareIntensityCascadeReviewPreventiveMedications = "review_preventive_medications"
	CareIntensityCascadeRevisitMonitoringPlan       = "revisit_monitoring_plan"
	CareIntensityCascadeConsentRefreshNeeded        = "consent_refresh_needed"
)

// CareIntensityEngine is the pure (IO-free) lifecycle engine for
// care-intensity transitions. It produces the transition Event + a list
// of cascade hints for the worklist surface; the caller is responsible
// for persistence (care_intensity_history INSERT, events INSERT,
// EvidenceTrace nodes/edges).
//
// Construct via NewCareIntensityEngine; the zero value is unusable
// because the clock is nil.
type CareIntensityEngine struct {
	now func() time.Time
}

// CareIntensityEngineOption configures a CareIntensityEngine at
// construction time. Use functional options so future knobs (e.g. a
// pluggable cascade rule table for site-specific overrides) can be
// added without breaking the constructor signature.
type CareIntensityEngineOption func(*CareIntensityEngine)

// WithCareIntensityClock overrides the engine's clock (default:
// time.Now().UTC()). Used in tests to drive deterministic timestamps on
// the produced Event.
func WithCareIntensityClock(now func() time.Time) CareIntensityEngineOption {
	return func(e *CareIntensityEngine) {
		if now != nil {
			e.now = now
		}
	}
}

// NewCareIntensityEngine returns a CareIntensityEngine. The default clock
// is time.Now().UTC(); pass WithCareIntensityClock for tests.
func NewCareIntensityEngine(opts ...CareIntensityEngineOption) *CareIntensityEngine {
	e := &CareIntensityEngine{
		now: func() time.Time { return time.Now().UTC() },
	}
	for _, opt := range opts {
		opt(e)
	}
	return e
}

// careIntensityCascadeRule is one row in the cascade rules table. From=""
// matches any source tag (used for the "transitioning into X regardless
// of source" case). The engine prefers the most-specific rule (From+To
// match) over the generic (To-only) rule when both exist for the same To.
type careIntensityCascadeRule struct {
	From     string
	To       string
	Cascades []CareIntensityCascade
}

// careIntensityCascadeRules is the rule table. Per Layer 2 doc §2.4:
//
//   - Transition into palliative: deprescribing of all non-symptom medications
//     expected; restrictive-practice authorisations re-examined; monitoring
//     plans revised. Three cascades.
//
//   - Transition into comfort_focused: deprescribing of preventive medications
//     appropriate; reduce monitoring intensity. Two cascades.
//
//   - Specific palliative → comfort_focused (intensity stepping back up):
//     revisit monitoring plan only — preventive deprescribing is already
//     in place from the prior palliative tagging, so the intensity-up
//     transition does not re-fire that cascade.
//
//   - active_treatment → rehabilitation, etc.: no automatic cascades. The
//     transition Event is still emitted but produces no worklist hints
//     by default.
//
// The engine matches longest-prefix: a (From, To) rule beats a (_, To)
// rule when both apply.
var careIntensityCascadeRules = []careIntensityCascadeRule{
	// Specific: palliative → comfort_focused. Stepping intensity back up.
	{
		From: models.CareIntensityTagPalliative,
		To:   models.CareIntensityTagComfortFocused,
		Cascades: []CareIntensityCascade{
			{Kind: CareIntensityCascadeRevisitMonitoringPlan, Reason: "Care intensity stepping back up from palliative to comfort_focused; review monitoring plan"},
		},
	},
	// Generic: any → palliative.
	{
		To: models.CareIntensityTagPalliative,
		Cascades: []CareIntensityCascade{
			{Kind: CareIntensityCascadeReviewPreventiveMedications, Reason: "Palliative tagging implies deprescribing preventive medications"},
			{Kind: CareIntensityCascadeRevisitMonitoringPlan, Reason: "Routine BP/lipid monitoring may no longer be relevant under palliative care"},
			{Kind: CareIntensityCascadeConsentRefreshNeeded, Reason: "Restrictive-practice authorisations need re-examination under palliative care"},
		},
	},
	// Generic: any → comfort_focused.
	{
		To: models.CareIntensityTagComfortFocused,
		Cascades: []CareIntensityCascade{
			{Kind: CareIntensityCascadeReviewPreventiveMedications, Reason: "Comfort-focused care: deprescribe primary prevention medications"},
			{Kind: CareIntensityCascadeRevisitMonitoringPlan, Reason: "Reduce monitoring intensity in line with care goals"},
		},
	},
}

// matchCareIntensityCascades returns the cascade list for the (from, to)
// transition. Specific (From+To) rules beat generic (To-only) rules; if
// no rule matches, the slice is nil.
func matchCareIntensityCascades(from, to string) []CareIntensityCascade {
	// First pass: prefer a specific From+To match.
	for _, r := range careIntensityCascadeRules {
		if r.From != "" && r.From == from && r.To == to {
			return r.Cascades
		}
	}
	// Second pass: generic To-only.
	for _, r := range careIntensityCascadeRules {
		if r.From == "" && r.To == to {
			return r.Cascades
		}
	}
	return nil
}

// careIntensityTransitionDescription is the structured description JSON
// emitted on the transition Event. Marshalled into Event.DescriptionStructured.
type careIntensityTransitionDescription struct {
	From     string                 `json:"from,omitempty"` // empty for the resident's first transition
	To       string                 `json:"to"`
	Cascades []CareIntensityCascade `json:"cascades,omitempty"`
}

// OnTransition produces the transition Event + cascade hints for moving a
// resident's care intensity from `from` to `to`. Pure: no IO, no calls
// out to the database. The caller writes both to storage in a single
// transaction along with the new care_intensity_history row.
//
// `from` may be empty string when the resident has no prior CareIntensity
// row (first-ever tagging); validation is the caller's responsibility
// (see validation.ValidateCareIntensityTransition).
//
// The Event has:
//   - EventType = care_intensity_transition (Care-transitions bucket;
//     routes to FHIR Encounter on egress)
//   - OccurredAt = engine clock
//   - ResidentID = residentRef
//   - ReportedByRef = documentedByRoleRef
//   - Severity = moderate when transitioning into palliative or
//     comfort_focused (because the cascade list is non-empty and the
//     transition implies treatment-intensity reduction); minor otherwise
//   - DescriptionStructured = {from, to, cascades} JSON for downstream
//     pattern-matching.
//
// If marshalling DescriptionStructured fails (it should not — the shape
// is closed), OnTransition panics: the caller cannot produce a valid
// Event without it, and silent error swallowing would mask a programmer
// bug.
func (e *CareIntensityEngine) OnTransition(
	from, to string,
	residentRef, documentedByRoleRef uuid.UUID,
) (models.Event, []CareIntensityCascade) {
	cascades := matchCareIntensityCascades(from, to)
	desc := careIntensityTransitionDescription{
		From:     from,
		To:       to,
		Cascades: cascades,
	}
	descJSON, err := json.Marshal(desc)
	if err != nil {
		// Programmer error — every field is a closed vocabulary.
		panic(fmt.Sprintf("care_intensity: marshal description failed: %v", err))
	}
	severity := models.EventSeverityMinor
	if to == models.CareIntensityTagPalliative ||
		to == models.CareIntensityTagComfortFocused {
		severity = models.EventSeverityModerate
	}
	ev := models.Event{
		ID:                    uuid.New(),
		EventType:             models.EventTypeCareIntensityTransition,
		OccurredAt:            e.now(),
		ResidentID:            residentRef,
		ReportedByRef:         documentedByRoleRef,
		Severity:              severity,
		DescriptionStructured: descJSON,
	}
	return ev, cascades
}
