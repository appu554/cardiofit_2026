package models

// StopCriteria.Spec JSONB shapes — documented per-pattern struct shapes that
// callers may use for type safety. The actual storage is JSON.RawMessage in
// MedicineUse.StopCriteria.Spec at the top level.

// StopCriteriaReviewSpec — for time-bounded review obligation.
//
// Used when StopCriteria.Triggers contains StopTriggerReviewDue and the
// review is time-based.
//
// Example: {"review_after_days": 30, "review_owner": "ACOP"}
type StopCriteriaReviewSpec struct {
	ReviewAfterDays int    `json:"review_after_days"`
	ReviewOwner     string `json:"review_owner,omitempty"` // RN|GP|ACOP|pharmacist
}

// StopCriteriaThresholdSpec — for criterion based on an observation threshold.
//
// Used when stop should be triggered if a specific observation crosses a
// threshold (e.g., stop ACE inhibitor if eGFR drops below 30).
//
// Example: {"observation_kind": "vital", "loinc_code": "8867-4", "operator": "<", "value": 50}
type StopCriteriaThresholdSpec struct {
	ObservationKind string  `json:"observation_kind"` // vital|lab|behavioural|mobility|weight
	LOINCCode       string  `json:"loinc_code,omitempty"`
	SNOMEDCode      string  `json:"snomed_code,omitempty"`
	Operator        string  `json:"operator"` // < <= = >= >
	Value           float64 `json:"value"`
}
