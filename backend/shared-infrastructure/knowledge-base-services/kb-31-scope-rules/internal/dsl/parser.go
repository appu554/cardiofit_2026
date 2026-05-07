package dsl

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ruleDocument is the on-disk YAML wrapper. We accept both the wrapped
// `scope_rule:` form (Layer 3 v2 doc Part 5.5.2) and the unwrapped form
// for round-trip convenience.
type ruleDocument struct {
	ScopeRule *ScopeRule `yaml:"scope_rule"`
}

// ParseRule parses a YAML document (wrapped or unwrapped) into a
// ScopeRule and validates cross-field invariants.
func ParseRule(data []byte) (*ScopeRule, error) {
	if len(data) == 0 {
		return nil, errors.New("empty rule document")
	}

	var doc ruleDocument
	if err := yaml.Unmarshal(data, &doc); err == nil && doc.ScopeRule != nil {
		applyDefaults(doc.ScopeRule)
		if err := ValidateSchema(*doc.ScopeRule); err != nil {
			return nil, fmt.Errorf("schema validation: %w", err)
		}
		return doc.ScopeRule, nil
	}

	var rule ScopeRule
	if err := yaml.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	if rule.RuleID == "" {
		return nil, errors.New("missing rule_id (document is neither wrapped nor unwrapped ScopeRule)")
	}
	applyDefaults(&rule)
	if err := ValidateSchema(rule); err != nil {
		return nil, fmt.Errorf("schema validation: %w", err)
	}
	return &rule, nil
}

// MarshalRule serialises the rule back to wrapped YAML for round-trip.
func MarshalRule(rule *ScopeRule) ([]byte, error) {
	return yaml.Marshal(ruleDocument{ScopeRule: rule})
}

// applyDefaults fills in the default Status. ScopeRule status defaults
// to ACTIVE; DRAFT must be explicit (so authors can't accidentally
// activate a pilot rule by omission).
func applyDefaults(r *ScopeRule) {
	if r.Status == "" {
		r.Status = StatusActive
	}
}

// ValidateSchema enforces cross-field invariants beyond YAML schema.
func ValidateSchema(r ScopeRule) error {
	var errs []string

	if strings.TrimSpace(r.RuleID) == "" {
		errs = append(errs, "rule_id is required")
	}
	if strings.TrimSpace(r.Jurisdiction) == "" {
		errs = append(errs, "jurisdiction is required (ISO-style, e.g. \"AU\" or \"AU/VIC\")")
	}
	if strings.TrimSpace(r.Category) == "" {
		errs = append(errs, "category is required (e.g. medication_administration_scope_restriction, prescriber_scope, credential_scope)")
	}
	if !isValidStatus(r.Status) {
		errs = append(errs, fmt.Sprintf("status %q is invalid (expected ACTIVE or DRAFT)", r.Status))
	}
	if r.Status == StatusDraft && strings.TrimSpace(r.ActivationGate) == "" {
		errs = append(errs, "status=DRAFT requires a non-empty activation_gate (document the activation contract)")
	}
	if r.EffectivePeriod.StartDate.IsZero() {
		errs = append(errs, "effective_period.start_date is required")
	}
	if r.EffectivePeriod.EndDate != nil && !r.EffectivePeriod.EndDate.After(r.EffectivePeriod.StartDate) {
		errs = append(errs, "effective_period.end_date must be strictly after start_date")
	}
	if r.EffectivePeriod.GracePeriodDays != nil && *r.EffectivePeriod.GracePeriodDays < 0 {
		errs = append(errs, "effective_period.grace_period_days must be >= 0")
	}

	if strings.TrimSpace(r.AppliesTo.Role) == "" {
		errs = append(errs, "applies_to.role is required")
	}
	if !isValidActionClass(r.AppliesTo.ActionClass) {
		errs = append(errs, fmt.Sprintf("applies_to.action_class %q is invalid", r.AppliesTo.ActionClass))
	}

	if !isValidDecision(r.Evaluation.Decision) {
		errs = append(errs, fmt.Sprintf("evaluation.decision %q is invalid", r.Evaluation.Decision))
	}
	if r.Evaluation.Decision == DecisionGrantedWithConditions && len(r.Evaluation.Conditions) == 0 {
		errs = append(errs, "evaluation.decision=granted_with_conditions requires at least one condition")
	}
	if len(r.Evaluation.FallbackEligibleRoles) > 0 && !r.Evaluation.FallbackRequired {
		errs = append(errs, "fallback_eligible_roles only valid when fallback_required=true")
	}
	if r.Evaluation.IfAnyConditionFails != nil && len(r.Evaluation.Conditions) == 0 {
		errs = append(errs, "if_any_condition_fails only valid when conditions are non-empty")
	}
	if r.Evaluation.IfAnyConditionFails != nil && !isValidDecision(r.Evaluation.IfAnyConditionFails.Decision) {
		errs = append(errs, fmt.Sprintf("if_any_condition_fails.decision %q is invalid", r.Evaluation.IfAnyConditionFails.Decision))
	}
	for i, c := range r.Evaluation.Conditions {
		if strings.TrimSpace(c.Condition) == "" {
			errs = append(errs, fmt.Sprintf("evaluation.conditions[%d].condition is required", i))
		}
		if strings.TrimSpace(c.Check) == "" {
			errs = append(errs, fmt.Sprintf("evaluation.conditions[%d].check is required", i))
		}
	}

	if strings.TrimSpace(r.Audit.LegislativeReference) == "" {
		errs = append(errs, "audit.legislative_reference is required (regulator-defensible)")
	}
	if r.Audit.RecordkeepingPeriodYears < 0 {
		errs = append(errs, "audit.recordkeeping_period_years must be >= 0")
	}
	if r.Audit.RecordkeepingRequired && r.Audit.RecordkeepingPeriodYears == 0 {
		errs = append(errs, "audit.recordkeeping_period_years must be > 0 when recordkeeping_required=true")
	}

	if len(errs) > 0 {
		return fmt.Errorf("invalid ScopeRule: %s", strings.Join(errs, "; "))
	}
	return nil
}

func isValidActionClass(a ActionClass) bool {
	switch a {
	case ActionAdminister, ActionPrescribe, ActionObserve,
		ActionRecommend, ActionConsentWitness, ActionViewProfile:
		return true
	}
	return false
}

func isValidDecision(d Decision) bool {
	switch d {
	case DecisionGranted, DecisionGrantedWithConditions, DecisionDenied:
		return true
	}
	return false
}

func isValidStatus(s Status) bool {
	switch s {
	case StatusActive, StatusDraft:
		return true
	}
	return false
}
