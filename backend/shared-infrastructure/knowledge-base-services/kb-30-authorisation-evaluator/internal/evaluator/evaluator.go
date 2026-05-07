// Package evaluator is the runtime decision engine for the kb-30
// Authorisation evaluator. Given an action query and a rule store, it
// resolves which rules apply, evaluates their conditions via an injected
// ConditionResolver, and combines the results per Layer 3 v2 doc Part 5.5.4
// (most-restrictive wins; explicit denied overrides; granted_with_conditions
// only when all conditions met).
package evaluator

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"kb-authorisation-evaluator/internal/dsl"
	"kb-authorisation-evaluator/internal/store"
)

// Query is a runtime authorisation request.
type Query struct {
	Jurisdiction       string
	Role               string
	ActionClass        dsl.ActionClass
	MedicationSchedule string
	MedicationClass    string
	ResidentRef        uuid.UUID
	ActorRef           uuid.UUID
	ActionDate         time.Time
}

// CacheKey is the canonical lookup key for the evaluator's cache layer.
// Format: auth:v1:{jurisdiction}:{role}:{action_class}:{schedule}:{class}:{resident}:{date}
func (q Query) CacheKey() string {
	return fmt.Sprintf("auth:v1:%s:%s:%s:%s:%s:%s:%s",
		q.Jurisdiction, q.Role, q.ActionClass,
		q.MedicationSchedule, q.MedicationClass,
		q.ResidentRef.String(),
		q.ActionDate.UTC().Format(time.RFC3339),
	)
}

// ConditionResult captures the outcome of one rule condition.
type ConditionResult struct {
	Condition string `json:"condition"`
	Check     string `json:"check"`
	Passed    bool   `json:"passed"`
	Detail    string `json:"detail,omitempty"`
}

// Result is the evaluation outcome for a query.
type Result struct {
	Decision             dsl.Decision      `json:"decision"`
	Reason               string            `json:"reason"`
	RuleID               string            `json:"rule_id,omitempty"`
	RuleVersion          int               `json:"rule_version,omitempty"`
	Conditions           []ConditionResult `json:"conditions,omitempty"`
	FallbackEligible     []string          `json:"fallback_eligible,omitempty"`
	LegislativeReference string            `json:"legislative_reference,omitempty"`
	GraceModeActive      bool              `json:"grace_mode_active,omitempty"`
	EvaluatedAt          time.Time         `json:"evaluated_at"`
	EvaluationDurationMs int64             `json:"evaluation_duration_ms"`
}

// ConditionResolver is the injectable substrate-aware predicate evaluator.
// It is called once per Condition.Check string. Implementations consult
// kb-20 (Roles + Credentials), kb-22 (PrescribingAgreements), and the
// resident's consent state to return Passed=true|false plus a detail string
// for the audit trail.
type ConditionResolver interface {
	Resolve(ctx context.Context, q Query, c dsl.Condition) (ConditionResult, error)
}

// ConditionResolverFunc adapts a plain function to the ConditionResolver
// interface for tests and lightweight wirings.
type ConditionResolverFunc func(ctx context.Context, q Query, c dsl.Condition) (ConditionResult, error)

func (f ConditionResolverFunc) Resolve(ctx context.Context, q Query, c dsl.Condition) (ConditionResult, error) {
	return f(ctx, q, c)
}

// AlwaysPassResolver passes every condition. Useful for fixture-driven
// integration tests where rule applicability — not condition logic — is
// the focus.
var AlwaysPassResolver ConditionResolver = ConditionResolverFunc(
	func(_ context.Context, _ Query, c dsl.Condition) (ConditionResult, error) {
		return ConditionResult{Condition: c.Condition, Check: c.Check, Passed: true, Detail: "stub: always pass"}, nil
	},
)

// Evaluator combines rule store + condition resolver.
type Evaluator struct {
	Store      store.Store
	Conditions ConditionResolver
}

// New builds an Evaluator.
func New(s store.Store, c ConditionResolver) *Evaluator {
	if c == nil {
		c = AlwaysPassResolver
	}
	return &Evaluator{Store: s, Conditions: c}
}

// Evaluate resolves the authorisation decision for q.
//
// Combination logic (Layer 3 v2 doc Part 5.5.4):
//   - if no rules apply: granted (open-by-default for action classes
//     without a regulatory rule; the caller may layer simple RBAC on top)
//   - if any rule decides denied: denied wins
//   - if any rule is granted_with_conditions and any condition fails:
//     denied (most restrictive)
//   - if every applicable rule grants (with all conditions met where
//     applicable): granted_with_conditions when conditions exist, else
//     granted
func (e *Evaluator) Evaluate(ctx context.Context, q Query) (Result, error) {
	startedAt := time.Now()
	rules, err := e.Store.ActiveForJurisdiction(ctx, q.Jurisdiction, q.ActionDate)
	if err != nil {
		return Result{}, fmt.Errorf("load active rules: %w", err)
	}

	// Filter to rules whose applies_to scope matches the query.
	var matching []store.StoredRule
	for _, r := range rules {
		if ruleMatchesQuery(r.Rule, q) {
			matching = append(matching, r)
		}
	}

	if len(matching) == 0 {
		return Result{
			Decision:             dsl.DecisionGranted,
			Reason:               "no applicable rule; default-grant for action class without jurisdictional rule",
			EvaluatedAt:          startedAt.UTC(),
			EvaluationDurationMs: time.Since(startedAt).Milliseconds(),
		}, nil
	}

	// Evaluate each matching rule. Track strongest negative outcome.
	var (
		denied             *Result
		grantedWithCond    *Result
		grantedUnconditional *Result
	)
	for _, sr := range matching {
		res, err := e.evaluateOne(ctx, sr, q)
		if err != nil {
			return Result{}, err
		}
		switch res.Decision {
		case dsl.DecisionDenied:
			r := res
			denied = &r
		case dsl.DecisionGrantedWithConditions:
			r := res
			grantedWithCond = &r
		case dsl.DecisionGranted:
			r := res
			grantedUnconditional = &r
		}
	}

	// Pick winner: denied > granted_with_conditions > granted.
	var winner Result
	switch {
	case denied != nil:
		winner = *denied
	case grantedWithCond != nil:
		winner = *grantedWithCond
	case grantedUnconditional != nil:
		winner = *grantedUnconditional
	default:
		// Should not occur given the len() check above.
		winner = Result{Decision: dsl.DecisionDenied, Reason: "internal: no winner"}
	}

	winner.EvaluatedAt = startedAt.UTC()
	winner.EvaluationDurationMs = time.Since(startedAt).Milliseconds()
	return winner, nil
}

// evaluateOne returns the outcome of a single rule against the query.
func (e *Evaluator) evaluateOne(ctx context.Context, sr store.StoredRule, q Query) (Result, error) {
	r := sr.Rule
	base := Result{
		RuleID:               r.RuleID,
		RuleVersion:          sr.Version,
		LegislativeReference: r.Audit.LegislativeReference,
		FallbackEligible:     r.Evaluation.FallbackEligibleRoles,
		GraceModeActive:      r.InGracePeriod(q.ActionDate),
	}

	switch r.Evaluation.Decision {
	case dsl.DecisionGranted:
		base.Decision = dsl.DecisionGranted
		base.Reason = r.Evaluation.Reason
		return base, nil

	case dsl.DecisionDenied:
		base.Decision = dsl.DecisionDenied
		base.Reason = r.Evaluation.Reason
		return base, nil

	case dsl.DecisionGrantedWithConditions:
		conds := make([]ConditionResult, 0, len(r.Evaluation.Conditions))
		allPassed := true
		for _, c := range r.Evaluation.Conditions {
			cr, err := e.Conditions.Resolve(ctx, q, c)
			if err != nil {
				return Result{}, fmt.Errorf("resolve condition %q: %w", c.Condition, err)
			}
			conds = append(conds, cr)
			if !cr.Passed {
				allPassed = false
			}
		}
		base.Conditions = conds
		if allPassed {
			base.Decision = dsl.DecisionGrantedWithConditions
			base.Reason = r.Evaluation.Reason
			return base, nil
		}
		// Apply if_any_condition_fails (default to denied if absent).
		base.Decision = dsl.DecisionDenied
		base.Reason = "at least one condition failed"
		if r.Evaluation.IfAnyConditionFails != nil {
			base.Decision = r.Evaluation.IfAnyConditionFails.Decision
			base.Reason = r.Evaluation.IfAnyConditionFails.Reason
		}
		return base, nil
	}
	return Result{}, fmt.Errorf("unknown decision %q for rule %s", r.Evaluation.Decision, r.RuleID)
}

func ruleMatchesQuery(r dsl.AuthorisationRule, q Query) bool {
	if r.AppliesTo.Role != "" && r.AppliesTo.Role != q.Role {
		return false
	}
	if r.AppliesTo.ActionClass != "" && r.AppliesTo.ActionClass != q.ActionClass {
		return false
	}
	if len(r.AppliesTo.MedicationSchedule) > 0 {
		if !contains(r.AppliesTo.MedicationSchedule, q.MedicationSchedule) {
			return false
		}
	}
	if len(r.AppliesTo.MedicationClassIncludes) > 0 {
		if !contains(r.AppliesTo.MedicationClassIncludes, q.MedicationClass) {
			return false
		}
	}
	return true
}

func contains(haystack []string, needle string) bool {
	for _, s := range haystack {
		if s == needle {
			return true
		}
	}
	return false
}
