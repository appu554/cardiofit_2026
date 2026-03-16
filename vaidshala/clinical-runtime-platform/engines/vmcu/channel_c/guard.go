package channel_c

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// ProtocolGuard evaluates pre-compiled protocol rules against a TitrationContext.
// Rules are loaded once at startup from protocol_rules.yaml.
// MUST NOT make any network calls — all data comes from TitrationContext.
type ProtocolGuard struct {
	rules     []compiledRule
	rulesHash string // SHA-256 of protocol_rules.yaml for SafetyTrace
}

// compiledRule is the in-memory representation of a protocol rule.
type compiledRule struct {
	RuleID       string
	Description  string
	GuidelineRef string
	Gate         ProtocolGate
	evaluate     func(ctx *TitrationContext) bool
}

// ruleFile is the YAML structure of protocol_rules.yaml.
type ruleFile struct {
	Version string     `yaml:"version"`
	Rules   []ruleSpec `yaml:"rules"`
}

type ruleSpec struct {
	RuleID       string        `yaml:"rule_id"`
	Description  string        `yaml:"description"`
	GuidelineRef string        `yaml:"guideline_ref"`
	Condition    ruleCondition `yaml:"condition"`
	Gate         string        `yaml:"gate"`
}

type ruleCondition struct {
	Field            string      `yaml:"field"`
	Operator         string      `yaml:"operator"`
	Value            interface{} `yaml:"value"`
	MedicationActive string      `yaml:"medication_active,omitempty"`
	ActionType       string      `yaml:"action_type,omitempty"`
}

// LoadRules parses protocol_rules.yaml and compiles rules into evaluators.
// Called once during V-MCU initialization — not at runtime.
func LoadRules(path string) (*ProtocolGuard, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read protocol rules: %w", err)
	}

	hash := sha256.Sum256(data)
	hashStr := fmt.Sprintf("%x", hash)

	var rf ruleFile
	if err := yaml.Unmarshal(data, &rf); err != nil {
		return nil, fmt.Errorf("failed to parse protocol rules: %w", err)
	}

	guard := &ProtocolGuard{
		rulesHash: hashStr,
	}

	for _, spec := range rf.Rules {
		cr, err := compileRule(spec)
		if err != nil {
			return nil, fmt.Errorf("failed to compile rule %s: %w", spec.RuleID, err)
		}
		guard.rules = append(guard.rules, cr)
	}

	return guard, nil
}

// RulesHash returns the SHA-256 of the loaded protocol_rules.yaml.
func (g *ProtocolGuard) RulesHash() string {
	return g.rulesHash
}

// RuleCount returns the number of compiled rules.
func (g *ProtocolGuard) RuleCount() int {
	return len(g.rules)
}

// Evaluate checks all rules against the current titration context.
// Returns the most restrictive gate from any matching rule.
// MUST NOT make any network calls.
func (g *ProtocolGuard) Evaluate(ctx *TitrationContext) ProtocolResult {
	result := ProtocolResult{
		Gate:        ProtoClear,
		RuleVersion: g.rulesHash,
	}

	for _, rule := range g.rules {
		if rule.evaluate(ctx) {
			gate := rule.Gate
			if gateLevel(gate) > gateLevel(result.Gate) {
				result.Gate = gate
				result.RuleID = rule.RuleID
				result.GuidelineRef = rule.GuidelineRef
			}
		}
	}

	return result
}

// gateLevel returns severity rank for ProtocolGate.
// MODIFY (1) sits between CLEAR (0) and PAUSE (2): it constrains but doesn't freeze.
func gateLevel(g ProtocolGate) int {
	switch g {
	case ProtoHalt:
		return 3
	case ProtoPause:
		return 2
	case ProtoModify:
		return 1
	case ProtoClear:
		return 0
	default:
		return 0
	}
}

// compileRule converts a YAML rule spec into a compiled evaluator function.
func compileRule(spec ruleSpec) (compiledRule, error) {
	gate := ProtocolGate(spec.Gate)
	if gate != ProtoHalt && gate != ProtoPause && gate != ProtoModify && gate != ProtoClear {
		return compiledRule{}, fmt.Errorf("invalid gate %q", spec.Gate)
	}

	eval, err := buildEvaluator(spec.Condition)
	if err != nil {
		return compiledRule{}, err
	}

	return compiledRule{
		RuleID:       spec.RuleID,
		Description:  spec.Description,
		GuidelineRef: spec.GuidelineRef,
		Gate:         gate,
		evaluate:     eval,
	}, nil
}

// buildEvaluator creates an evaluator function from a rule condition.
func buildEvaluator(cond ruleCondition) (func(*TitrationContext) bool, error) {
	switch cond.Field {
	case "egfr":
		threshold, ok := toFloat64(cond.Value)
		if !ok {
			return nil, fmt.Errorf("egfr threshold must be numeric")
		}
		medClass := cond.MedicationActive
		return func(ctx *TitrationContext) bool {
			if medClass != "" && !hasMedication(ctx.ActiveMedications, medClass) {
				return false
			}
			return compareFloat(ctx.EGFR, cond.Operator, threshold)
		}, nil

	case "aki_detected":
		return func(ctx *TitrationContext) bool {
			return ctx.AKIDetected
		}, nil

	case "active_hypoglycaemia":
		actionType := cond.ActionType
		return func(ctx *TitrationContext) bool {
			if !ctx.ActiveHypoglycaemia {
				return false
			}
			if actionType != "" {
				return ctx.ProposedAction == actionType
			}
			return true
		}, nil

	case "dose_delta_percent":
		threshold, ok := toFloat64(cond.Value)
		if !ok {
			return nil, fmt.Errorf("dose_delta_percent threshold must be numeric")
		}
		return func(ctx *TitrationContext) bool {
			return compareFloat(ctx.DoseDeltaPercent, cond.Operator, threshold)
		}, nil

	case "hypoglycaemia_within_7d":
		actionType := cond.ActionType
		return func(ctx *TitrationContext) bool {
			if !ctx.HypoglycaemiaWithin7d {
				return false
			}
			if actionType != "" {
				return ctx.ProposedAction == actionType
			}
			return true
		}, nil

	// ── HTN co-management boolean fields (Wave 1 P0) ──
	// Each field is pre-computed by the orchestrator as a composite condition.

	case "acei_arb_hyperk_declining_egfr":
		// PG-08: ACEi/ARB + K+ ≥5.5 + declining eGFR → HALT uptitration
		actionType := cond.ActionType
		return func(ctx *TitrationContext) bool {
			if !ctx.ACEiARBHyperKDecliningEGFR {
				return false
			}
			if actionType != "" {
				return ctx.ProposedAction == actionType
			}
			return true
		}, nil

	case "beta_blocker_insulin_active":
		// PG-09: Beta-blocker + insulin → MODIFY (hypo masking risk)
		return func(ctx *TitrationContext) bool {
			return ctx.BetaBlockerInsulinActive
		}, nil

	case "resistant_htn_detected":
		// PG-10: ≥3 antihypertensives at max + uncontrolled → PAUSE
		return func(ctx *TitrationContext) bool {
			return ctx.ResistantHTNDetected
		}, nil

	case "thiazide_hyponatraemia":
		// PG-11: Thiazide + Na+ <130 → HALT
		return func(ctx *TitrationContext) bool {
			return ctx.ThiazideHyponatraemia
		}, nil

	case "mra_hyperk_low_egfr":
		// PG-12: MRA + K+ >5.0 + eGFR <45 → MODIFY (dose cap)
		return func(ctx *TitrationContext) bool {
			return ctx.MRAHyperKLowEGFR
		}, nil

	case "ccb_excessive_response":
		// PG-13: CCB + SBP <110 + recent dose increase → MODIFY
		return func(ctx *TitrationContext) bool {
			return ctx.CCBExcessiveResponse
		}, nil

	case "raas_creatinine_tolerant":
		// PG-14: RAAS tolerance active → suppress B-03 HALT (handled in Channel B)
		// Channel C records this as CLEAR with audit trail
		return func(ctx *TitrationContext) bool {
			return ctx.RAASCreatinineTolerant
		}, nil

	// ── HTN co-management fields (Wave 2 P1) ──

	case "acei_induced_cough_probability":
		// PG-15: ACEi cough posterior from KB-22 P03 > 0.70 → MODIFY (ARB switch)
		threshold, ok := toFloat64(cond.Value)
		if !ok {
			return nil, fmt.Errorf("acei_induced_cough_probability threshold must be numeric")
		}
		return func(ctx *TitrationContext) bool {
			return compareFloat(ctx.ACEiInducedCoughProbability, cond.Operator, threshold)
		}, nil

	case "af_confirmed_no_anticoagulation":
		// PG-16: AF confirmed (B-16) + no anticoagulation → PAUSE
		return func(ctx *TitrationContext) bool {
			return ctx.AFConfirmedNoAnticoagulation
		}, nil

	case "dual_raas_active":
		// PG-08-DUAL-RAAS: ACEi AND ARB simultaneously (contraindicated)
		return func(ctx *TitrationContext) bool {
			return ctx.DualRAASActive
		}, nil

	case "ckd_stage4_deprescribing_blocked":
		// AD-09: CKD Stage 4 deprescribing hard block
		return func(ctx *TitrationContext) bool {
			return ctx.CKDStage4DeprescribingBlocked
		}, nil

	// ── ACR-based RAAS escalation (PG-17) ──

	case "acr_a3_no_raas":
		// PG-17-A3: ACR category A3 AND NOT on ACEi/ARB → HALT
		return func(ctx *TitrationContext) bool {
			return ctx.ACRA3NoRAAS
		}, nil

	case "acr_a2_no_raas":
		// PG-17-A2: ACR category A2 AND NOT on ACEi/ARB → MODIFY
		return func(ctx *TitrationContext) bool {
			return ctx.ACRA2NoRAAS
		}, nil

	default:
		return nil, fmt.Errorf("unknown field %q", cond.Field)
	}
}

func hasMedication(meds []string, target string) bool {
	upper := strings.ToUpper(target)
	for _, m := range meds {
		if strings.ToUpper(m) == upper {
			return true
		}
	}
	return false
}

func compareFloat(val float64, op string, threshold float64) bool {
	switch op {
	case "lt":
		return val < threshold
	case "gt":
		return val > threshold
	case "lte":
		return val <= threshold
	case "gte":
		return val >= threshold
	case "eq":
		return val == threshold
	default:
		return false
	}
}

func toFloat64(v interface{}) (float64, bool) {
	switch n := v.(type) {
	case float64:
		return n, true
	case int:
		return float64(n), true
	case int64:
		return float64(n), true
	default:
		return 0, false
	}
}
