// Package channel_b — TreatmentPerturbation (Amendment 2).
//
// When an antihypertensive drug is started, stopped, or uptitrated, certain
// lab signals are expected to shift for a pharmacologically-defined window.
// Channel B must not fire false safety gates on expected shifts during these
// perturbation windows.
//
// This file defines:
//   - TreatmentPerturbation struct (one active perturbation instance)
//   - PerturbationSpec (static registry entry)
//   - PerturbationRegistry (lookup table for all antihypertensive classes)
//   - IsPerturbationActive helper
//   - FindActivePerturbation helper for monitor rules
package channel_b

import "time"

// DrugClass identifies an antihypertensive drug class for perturbation tracking.
type DrugClass string

const (
	DrugClassSGLT2i      DrugClass = "SGLT2I"
	DrugClassACEiARB     DrugClass = "ACEI_ARB"
	DrugClassThiazide    DrugClass = "THIAZIDE"
	DrugClassBetaBlocker DrugClass = "BETA_BLOCKER"
	DrugClassCCB         DrugClass = "CCB"
)

// ChangeType identifies what happened to the drug.
type ChangeType string

const (
	ChangeStarted   ChangeType = "STARTED"
	ChangeIncreased ChangeType = "INCREASED"
	ChangeStopped   ChangeType = "STOPPED"
)

// TreatmentPerturbation represents an active perturbation window for a patient.
// Populated by the orchestrator from KB-20 medication events.
type TreatmentPerturbation struct {
	Drug           DrugClass  `json:"drug_class"`
	Change         ChangeType `json:"change_type"`
	StartedAt      time.Time  `json:"started_at"`
	WindowDuration time.Duration `json:"window_duration"`

	// Affected physiology signals and expected direction.
	AffectedSignals      []string `json:"affected_signals"`
	ExpectedDirection    string   `json:"expected_direction"` // "UP" | "DOWN"
	ExpectedMagnitudeMin float64  `json:"expected_magnitude_min"`
	ExpectedMagnitudeMax float64  `json:"expected_magnitude_max"`
	CausalNote           string   `json:"causal_note"`
}

// IsActive returns true if the perturbation window has not yet expired.
func (tp *TreatmentPerturbation) IsActive(now time.Time) bool {
	return now.Before(tp.StartedAt.Add(tp.WindowDuration))
}

// PerturbationSpec is a static registry entry defining the expected perturbation
// for a (DrugClass, ChangeType) pair. These are pharmacologically invariant.
type PerturbationSpec struct {
	Drug                 DrugClass
	Change               ChangeType
	WindowDuration       time.Duration
	AffectedSignals      []string
	ExpectedDirection    string
	ExpectedMagnitudeMin float64
	ExpectedMagnitudeMax float64
	CausalNote           string
}

// perturbationRegistry is the static lookup table per Amendment 2.
// Keyed by DrugClass + ChangeType.
var perturbationRegistry = []PerturbationSpec{
	// ── SGLT2 inhibitors ──
	{
		Drug: DrugClassSGLT2i, Change: ChangeStarted,
		WindowDuration:       4 * 7 * 24 * time.Hour, // 4 weeks
		AffectedSignals:      []string{"SBP"},
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 3.0, ExpectedMagnitudeMax: 5.0,
		CausalNote: "SGLT2i natriuresis and osmotic diuresis lower SBP 3-5 mmHg over 4 weeks",
	},
	{
		Drug: DrugClassSGLT2i, Change: ChangeStopped,
		WindowDuration:       4 * 7 * 24 * time.Hour, // 4 weeks
		AffectedSignals:      []string{"SBP"},
		ExpectedDirection:    "UP",
		ExpectedMagnitudeMin: 3.0, ExpectedMagnitudeMax: 5.0,
		CausalNote: "SGLT2i withdrawal removes natriuretic effect; SBP rebounds 3-5 mmHg",
	},

	// ── ACEi / ARB ──
	{
		Drug: DrugClassACEiARB, Change: ChangeStarted,
		WindowDuration:       14 * 24 * time.Hour, // 14 days
		AffectedSignals:      []string{"CREATININE", "EGFR"},
		ExpectedDirection:    "UP", // creatinine UP, eGFR DOWN
		ExpectedMagnitudeMin: 10.0, ExpectedMagnitudeMax: 30.0,
		CausalNote: "RAAS blockade reduces efferent arteriolar tone; creatinine rise 10-30% expected within 14d",
	},
	{
		Drug: DrugClassACEiARB, Change: ChangeIncreased,
		WindowDuration:       14 * 24 * time.Hour, // 14 days
		AffectedSignals:      []string{"CREATININE", "EGFR"},
		ExpectedDirection:    "UP",
		ExpectedMagnitudeMin: 10.0, ExpectedMagnitudeMax: 30.0,
		CausalNote: "RAAS blockade uptitration; creatinine rise 10-30% expected within 14d",
	},
	{
		Drug: DrugClassACEiARB, Change: ChangeStopped,
		WindowDuration:       7 * 24 * time.Hour, // 7 days
		AffectedSignals:      []string{"CREATININE", "POTASSIUM"},
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 0.0, ExpectedMagnitudeMax: 0.0, // variable
		CausalNote: "RAAS blockade withdrawal; creatinine and K+ drop variably within 7d",
	},

	// ── Thiazide diuretics ──
	{
		Drug: DrugClassThiazide, Change: ChangeStarted,
		WindowDuration:       3 * 7 * 24 * time.Hour, // 3 weeks
		AffectedSignals:      []string{"POTASSIUM"},
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 0.3, ExpectedMagnitudeMax: 0.5,
		CausalNote: "Thiazide kaliuresis lowers K+ 0.3-0.5 mmol/L over 3 weeks",
	},

	// ── Beta-blockers ──
	{
		Drug: DrugClassBetaBlocker, Change: ChangeStarted,
		WindowDuration:       2 * 7 * 24 * time.Hour, // 2 weeks
		AffectedSignals:      []string{"HR", "GLUCOSE"},
		ExpectedDirection:    "DOWN", // HR DOWN, glucose may UP
		ExpectedMagnitudeMin: 0.0, ExpectedMagnitudeMax: 0.0, // variable
		CausalNote: "Beta-blocker negative chronotropy lowers HR; hepatic gluconeogenesis blunted → glucose may rise",
	},
	{
		Drug: DrugClassBetaBlocker, Change: ChangeIncreased,
		WindowDuration:       2 * 7 * 24 * time.Hour, // 2 weeks
		AffectedSignals:      []string{"HR", "GLUCOSE"},
		ExpectedDirection:    "DOWN",
		ExpectedMagnitudeMin: 0.0, ExpectedMagnitudeMax: 0.0,
		CausalNote: "Beta-blocker uptitration; additional HR reduction expected over 2 weeks",
	},

	// ── Calcium channel blockers ──
	{
		Drug: DrugClassCCB, Change: ChangeStarted,
		WindowDuration:       2 * 7 * 24 * time.Hour, // 2 weeks
		AffectedSignals:      []string{"SBP", "HR"},
		ExpectedDirection:    "DOWN", // SBP DOWN, HR may reflex UP
		ExpectedMagnitudeMin: 5.0, ExpectedMagnitudeMax: 10.0,
		CausalNote: "CCB vasodilation lowers SBP 5-10 mmHg; reflex tachycardia possible with DHP CCBs",
	},
}

// LookupPerturbationSpec returns the static spec for a given drug class and change type.
// Returns nil if no spec exists for the combination.
func LookupPerturbationSpec(drug DrugClass, change ChangeType) *PerturbationSpec {
	for i := range perturbationRegistry {
		if perturbationRegistry[i].Drug == drug && perturbationRegistry[i].Change == change {
			return &perturbationRegistry[i]
		}
	}
	return nil
}

// NewTreatmentPerturbation creates a TreatmentPerturbation from a registry spec
// and a concrete event time. Returns nil if no spec exists.
func NewTreatmentPerturbation(drug DrugClass, change ChangeType, startedAt time.Time) *TreatmentPerturbation {
	spec := LookupPerturbationSpec(drug, change)
	if spec == nil {
		return nil
	}
	return &TreatmentPerturbation{
		Drug:                 spec.Drug,
		Change:               spec.Change,
		StartedAt:            startedAt,
		WindowDuration:       spec.WindowDuration,
		AffectedSignals:      spec.AffectedSignals,
		ExpectedDirection:    spec.ExpectedDirection,
		ExpectedMagnitudeMin: spec.ExpectedMagnitudeMin,
		ExpectedMagnitudeMax: spec.ExpectedMagnitudeMax,
		CausalNote:           spec.CausalNote,
	}
}

// IsPerturbationActive checks whether a perturbation window is active for
// the given drug class and change type at the specified time.
// Searches the provided list of active perturbations.
func IsPerturbationActive(perturbations []TreatmentPerturbation, drug DrugClass, change ChangeType, now time.Time) bool {
	for i := range perturbations {
		if perturbations[i].Drug == drug && perturbations[i].Change == change && perturbations[i].IsActive(now) {
			return true
		}
	}
	return false
}

// FindActivePerturbation returns the first active perturbation matching the
// drug class (any change type) at the specified time. Returns nil if none found.
func FindActivePerturbation(perturbations []TreatmentPerturbation, drug DrugClass, now time.Time) *TreatmentPerturbation {
	for i := range perturbations {
		if perturbations[i].Drug == drug && perturbations[i].IsActive(now) {
			return &perturbations[i]
		}
	}
	return nil
}

// HasActiveSignalPerturbation returns true if any active perturbation affects
// the specified signal (e.g., "SBP", "POTASSIUM", "CREATININE", "HR").
func HasActiveSignalPerturbation(perturbations []TreatmentPerturbation, signal string, now time.Time) bool {
	for i := range perturbations {
		if !perturbations[i].IsActive(now) {
			continue
		}
		for _, s := range perturbations[i].AffectedSignals {
			if s == signal {
				return true
			}
		}
	}
	return false
}
