package dsl

import (
	"errors"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ruleDocument is the on-disk YAML wrapper. We accept both the wrapped
// `authorisation_rule:` form (per Layer 3 v2 doc Part 4.5.2) and the
// unwrapped form for round-trip convenience.
type ruleDocument struct {
	AuthorisationRule *AuthorisationRule `yaml:"authorisation_rule"`
}

// ParseRule parses a YAML document (wrapped or unwrapped) into an
// AuthorisationRule and validates cross-field invariants. Returns an
// error with the YAML line/column when the document is malformed.
func ParseRule(data []byte) (*AuthorisationRule, error) {
	if len(data) == 0 {
		return nil, errors.New("empty rule document")
	}

	// Try wrapped form first.
	var doc ruleDocument
	if err := yaml.Unmarshal(data, &doc); err == nil && doc.AuthorisationRule != nil {
		if err := ValidateSchema(*doc.AuthorisationRule); err != nil {
			return nil, fmt.Errorf("schema validation: %w", err)
		}
		return doc.AuthorisationRule, nil
	}

	// Fallback: try unwrapped form.
	var rule AuthorisationRule
	if err := yaml.Unmarshal(data, &rule); err != nil {
		return nil, fmt.Errorf("yaml unmarshal: %w", err)
	}
	if rule.RuleID == "" {
		return nil, errors.New("missing rule_id (document is neither wrapped nor unwrapped AuthorisationRule)")
	}
	if err := ValidateSchema(rule); err != nil {
		return nil, fmt.Errorf("schema validation: %w", err)
	}
	return &rule, nil
}

// MarshalRule serialises the rule back to wrapped YAML for round-trip.
func MarshalRule(rule *AuthorisationRule) ([]byte, error) {
	return yaml.Marshal(ruleDocument{AuthorisationRule: rule})
}

// ValidateSchema applies cross-field invariants beyond the YAML schema.
//
// Invariants enforced:
//   - rule_id is required and non-empty
//   - jurisdiction is required (ISO-style, e.g. "AU" or "AU/VIC")
//   - effective_period.start_date is required (non-zero)
//   - end_date, when set, must be strictly after start_date
//   - grace_period_days, when set, must be >= 0; only meaningful when
//     end_date is set OR start_date is in the future (deferred enforcement)
//   - applies_to.role is required
//   - applies_to.action_class is required and one of the known enums
//   - evaluation.decision is required and one of the known enums
//   - fallback_eligible_roles is only meaningful when fallback_required=true
//   - if_any_condition_fails is only meaningful when conditions are non-empty
//   - audit.legislative_reference is required (regulator-defensible)
//   - recordkeeping_period_years, when set, must be > 0
func ValidateSchema(r AuthorisationRule) error {
	var errs []string

	if strings.TrimSpace(r.RuleID) == "" {
		errs = append(errs, "rule_id is required")
	}
	if strings.TrimSpace(r.Jurisdiction) == "" {
		errs = append(errs, "jurisdiction is required (ISO-style, e.g. \"AU\" or \"AU/VIC\")")
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
		errs = append(errs, fmt.Sprintf("applies_to.action_class %q is invalid (expected one of: administer, prescribe, observe, recommend, consent_witness, view_profile)", r.AppliesTo.ActionClass))
	}

	if !isValidDecision(r.Evaluation.Decision) {
		errs = append(errs, fmt.Sprintf("evaluation.decision %q is invalid (expected one of: granted, granted_with_conditions, denied)", r.Evaluation.Decision))
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
		return fmt.Errorf("invalid AuthorisationRule: %s", strings.Join(errs, "; "))
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
