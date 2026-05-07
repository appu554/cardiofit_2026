// Package dsl defines the AuthorisationRule format for the kb-30 evaluator.
//
// The schema mirrors Layer 3 v2 doc Part 4.5.2 — declarative, jurisdiction-aware,
// time-aware rules used by the Authorisation evaluator runtime to make
// administer / prescribe / observe / recommend decisions.
package dsl

import (
	"time"
)

// Decision is the outcome of evaluating an AuthorisationRule against a query.
type Decision string

const (
	// DecisionGranted means the action is unconditionally permitted.
	DecisionGranted Decision = "granted"
	// DecisionGrantedWithConditions means the action is permitted only if
	// every entry in EvaluationBlock.Conditions evaluates true.
	DecisionGrantedWithConditions Decision = "granted_with_conditions"
	// DecisionDenied means the action is forbidden in this jurisdiction.
	DecisionDenied Decision = "denied"
)

// ActionClass is the high-level kind of action the rule applies to.
type ActionClass string

const (
	ActionAdminister     ActionClass = "administer"
	ActionPrescribe      ActionClass = "prescribe"
	ActionObserve        ActionClass = "observe"
	ActionRecommend      ActionClass = "recommend"
	ActionConsentWitness ActionClass = "consent_witness"
	ActionViewProfile    ActionClass = "view_profile"
)

// AuthorisationRule is the top-level YAML document. The wrapper key
// `authorisation_rule:` in source YAML is unwrapped during parsing.
type AuthorisationRule struct {
	RuleID          string          `yaml:"rule_id" json:"rule_id"`
	Jurisdiction    string          `yaml:"jurisdiction" json:"jurisdiction"`
	EffectivePeriod EffectivePeriod `yaml:"effective_period" json:"effective_period"`
	AppliesTo       AppliesToScope  `yaml:"applies_to" json:"applies_to"`
	Evaluation      EvaluationBlock `yaml:"evaluation" json:"evaluation"`
	Audit           AuditBlock      `yaml:"audit" json:"audit"`
}

// EffectivePeriod governs when the rule is active. EndDate=nil means "in
// force until amended". GracePeriodDays applies AFTER StartDate to defer
// hard enforcement (e.g. Victorian PCW exclusion 1 Jul 2026 + 90 days
// grace -> hard enforcement 29 Sep 2026).
type EffectivePeriod struct {
	StartDate       time.Time  `yaml:"start_date" json:"start_date"`
	EndDate         *time.Time `yaml:"end_date,omitempty" json:"end_date,omitempty"`
	GracePeriodDays *int       `yaml:"grace_period_days,omitempty" json:"grace_period_days,omitempty"`
}

// AppliesToScope filters which actions the rule fires on.
type AppliesToScope struct {
	Role                      string      `yaml:"role" json:"role"`
	ActionClass               ActionClass `yaml:"action_class" json:"action_class"`
	MedicationSchedule        []string    `yaml:"medication_schedule,omitempty" json:"medication_schedule,omitempty"`
	MedicationClassIncludes   []string    `yaml:"medication_class_includes,omitempty" json:"medication_class_includes,omitempty"`
	ResidentSelfAdministering *bool       `yaml:"resident_self_administering,omitempty" json:"resident_self_administering,omitempty"`
}

// EvaluationBlock describes the decision logic.
type EvaluationBlock struct {
	Decision              Decision      `yaml:"decision" json:"decision"`
	Reason                string        `yaml:"reason,omitempty" json:"reason,omitempty"`
	Conditions            []Condition   `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	FallbackRequired      bool          `yaml:"fallback_required,omitempty" json:"fallback_required,omitempty"`
	FallbackEligibleRoles []string      `yaml:"fallback_eligible_roles,omitempty" json:"fallback_eligible_roles,omitempty"`
	IfAnyConditionFails   *FailureBlock `yaml:"if_any_condition_fails,omitempty" json:"if_any_condition_fails,omitempty"`
}

// Condition is a single named predicate that must evaluate true for a
// granted_with_conditions decision to hold.
type Condition struct {
	Condition string `yaml:"condition" json:"condition"`
	Check     string `yaml:"check" json:"check"`
}

// FailureBlock is the fallback decision when any condition fails.
type FailureBlock struct {
	Decision Decision `yaml:"decision" json:"decision"`
	Reason   string   `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// AuditBlock carries the regulatory provenance of the rule.
type AuditBlock struct {
	LegislativeReference     string `yaml:"legislative_reference" json:"legislative_reference"`
	SourceID                 string `yaml:"source_id,omitempty" json:"source_id,omitempty"`
	SourceVersion            string `yaml:"source_version,omitempty" json:"source_version,omitempty"`
	RecordkeepingRequired    bool   `yaml:"recordkeeping_required" json:"recordkeeping_required"`
	RecordkeepingPeriodYears int    `yaml:"recordkeeping_period_years,omitempty" json:"recordkeeping_period_years,omitempty"`
}

// IsActiveAt reports whether the rule is in force at the given time, taking
// into account both StartDate and (when set) EndDate. GracePeriodDays does
// NOT change activeness — it changes whether enforcement is hard or soft,
// which is a runtime concern for the evaluator, not the parser.
func (r *AuthorisationRule) IsActiveAt(t time.Time) bool {
	if t.Before(r.EffectivePeriod.StartDate) {
		return false
	}
	if r.EffectivePeriod.EndDate != nil && !t.Before(*r.EffectivePeriod.EndDate) {
		return false
	}
	return true
}

// InGracePeriod reports whether t falls inside the rule's grace window
// (StartDate .. StartDate + GracePeriodDays). When GracePeriodDays is nil
// or zero, this is always false.
func (r *AuthorisationRule) InGracePeriod(t time.Time) bool {
	if r.EffectivePeriod.GracePeriodDays == nil || *r.EffectivePeriod.GracePeriodDays <= 0 {
		return false
	}
	graceEnd := r.EffectivePeriod.StartDate.AddDate(0, 0, *r.EffectivePeriod.GracePeriodDays)
	return !t.Before(r.EffectivePeriod.StartDate) && t.Before(graceEnd)
}
