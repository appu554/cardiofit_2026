// Package dsl defines the ScopeRule data model for the kb-31 service.
//
// Per Layer 3 v2 doc Part 5.5.2 (ScopeRules-as-data), the schema mirrors
// kb-30's AuthorisationRule (Part 4.5.2) field-for-field, with one
// addition: a free-form `category` discriminator. The two schemas share
// the same on-disk YAML grammar — a kb-30 AuthorisationRule and a kb-31
// ScopeRule differ only in semantic role:
//
//   - AuthorisationRule: runtime evaluator input (kb-30) — applied at
//     each prescribe/administer/observe action.
//   - ScopeRule:         regulator-defensible source-of-truth (kb-31) —
//     describes jurisdiction + temporal scope, fans out to one or more
//     AuthorisationRules consumed by the evaluator.
//
// The CompatibilityChecker (Layer 3 v2 doc Part 4.2 Event D) listens for
// ScopeRule changes and marks the AuthorisationRules that reference them
// (via authorisation_gating.scope_rule_refs[]) as STALE.
package dsl

import "time"

// Status describes whether a ScopeRule is fully live or staged.
type Status string

const (
	// StatusActive means the rule is in force from EffectivePeriod.StartDate
	// onward (subject to the grace period for hard enforcement).
	StatusActive Status = "ACTIVE"
	// StatusDraft means the rule is staged but not in force; the
	// evaluator must NOT consume it. Used for pilot scope rules whose
	// activation depends on an external gate.
	StatusDraft Status = "DRAFT"
)

// Decision is the outcome of evaluating a ScopeRule against a query.
// Mirror of kb-30 dsl.Decision (Layer 3 v2 doc Part 5.5.2).
type Decision string

const (
	DecisionGranted               Decision = "granted"
	DecisionGrantedWithConditions Decision = "granted_with_conditions"
	DecisionDenied                Decision = "denied"
)

// ActionClass mirrors kb-30 dsl.ActionClass (Part 5.5.2).
type ActionClass string

const (
	ActionAdminister     ActionClass = "administer"
	ActionPrescribe      ActionClass = "prescribe"
	ActionObserve        ActionClass = "observe"
	ActionRecommend      ActionClass = "recommend"
	ActionConsentWitness ActionClass = "consent_witness"
	ActionViewProfile    ActionClass = "view_profile"
)

// ScopeRule is the on-disk + in-memory ScopeRule (Part 5.5.2).
//
// The wrapper key `scope_rule:` in YAML is unwrapped during parsing.
type ScopeRule struct {
	RuleID          string          `yaml:"rule_id" json:"rule_id"`
	Jurisdiction    string          `yaml:"jurisdiction" json:"jurisdiction"`
	Category        string          `yaml:"category" json:"category"`
	Status          Status          `yaml:"status,omitempty" json:"status,omitempty"`
	ActivationGate  string          `yaml:"activation_gate,omitempty" json:"activation_gate,omitempty"`
	EffectivePeriod EffectivePeriod `yaml:"effective_period" json:"effective_period"`
	AppliesTo       AppliesToScope  `yaml:"applies_to" json:"applies_to"`
	Evaluation      EvaluationBlock `yaml:"evaluation" json:"evaluation"`
	Audit           AuditBlock      `yaml:"audit" json:"audit"`
}

// EffectivePeriod governs when the rule is in force. Mirrors kb-30.
type EffectivePeriod struct {
	StartDate       time.Time  `yaml:"start_date" json:"start_date"`
	EndDate         *time.Time `yaml:"end_date,omitempty" json:"end_date,omitempty"`
	GracePeriodDays *int       `yaml:"grace_period_days,omitempty" json:"grace_period_days,omitempty"`
}

// AppliesToScope filters which actions the rule fires on. Mirrors kb-30.
type AppliesToScope struct {
	Role                      string      `yaml:"role" json:"role"`
	ActionClass               ActionClass `yaml:"action_class" json:"action_class"`
	MedicationSchedule        []string    `yaml:"medication_schedule,omitempty" json:"medication_schedule,omitempty"`
	MedicationClassIncludes   []string    `yaml:"medication_class_includes,omitempty" json:"medication_class_includes,omitempty"`
	ResidentSelfAdministering *bool       `yaml:"resident_self_administering,omitempty" json:"resident_self_administering,omitempty"`
}

// EvaluationBlock describes the decision logic. Mirrors kb-30.
type EvaluationBlock struct {
	Decision              Decision      `yaml:"decision" json:"decision"`
	Reason                string        `yaml:"reason,omitempty" json:"reason,omitempty"`
	Conditions            []Condition   `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	FallbackRequired      bool          `yaml:"fallback_required,omitempty" json:"fallback_required,omitempty"`
	FallbackEligibleRoles []string      `yaml:"fallback_eligible_roles,omitempty" json:"fallback_eligible_roles,omitempty"`
	IfAnyConditionFails   *FailureBlock `yaml:"if_any_condition_fails,omitempty" json:"if_any_condition_fails,omitempty"`
}

// Condition is a single named predicate.
type Condition struct {
	Condition string `yaml:"condition" json:"condition"`
	Check     string `yaml:"check" json:"check"`
}

// FailureBlock is the fallback decision when any condition fails.
type FailureBlock struct {
	Decision Decision `yaml:"decision" json:"decision"`
	Reason   string   `yaml:"reason,omitempty" json:"reason,omitempty"`
}

// AuditBlock carries regulatory provenance.
type AuditBlock struct {
	LegislativeReference     string `yaml:"legislative_reference" json:"legislative_reference"`
	SourceID                 string `yaml:"source_id,omitempty" json:"source_id,omitempty"`
	SourceVersion            string `yaml:"source_version,omitempty" json:"source_version,omitempty"`
	SourceURL                string `yaml:"source_url,omitempty" json:"source_url,omitempty"`
	RecordkeepingRequired    bool   `yaml:"recordkeeping_required" json:"recordkeeping_required"`
	RecordkeepingPeriodYears int    `yaml:"recordkeeping_period_years,omitempty" json:"recordkeeping_period_years,omitempty"`
}

// IsActiveAt reports whether the rule is in force at t. Status=DRAFT
// always returns false: a draft rule is by contract not active even
// inside its declared effective period.
func (r *ScopeRule) IsActiveAt(t time.Time) bool {
	if r.Status == StatusDraft {
		return false
	}
	if t.Before(r.EffectivePeriod.StartDate) {
		return false
	}
	if r.EffectivePeriod.EndDate != nil && !t.Before(*r.EffectivePeriod.EndDate) {
		return false
	}
	return true
}

// InGracePeriod reports whether t falls inside the rule's grace window.
func (r *ScopeRule) InGracePeriod(t time.Time) bool {
	if r.EffectivePeriod.GracePeriodDays == nil || *r.EffectivePeriod.GracePeriodDays <= 0 {
		return false
	}
	graceEnd := r.EffectivePeriod.StartDate.AddDate(0, 0, *r.EffectivePeriod.GracePeriodDays)
	return !t.Before(r.EffectivePeriod.StartDate) && t.Before(graceEnd)
}
