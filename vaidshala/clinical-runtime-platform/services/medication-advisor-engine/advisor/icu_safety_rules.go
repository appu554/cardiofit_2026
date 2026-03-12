// Package advisor provides ICU Safety Intelligence Rules.
// This file implements Tier-10 Phase 2: ICU-specific medication safety rules.
//
// The ICU Safety Rules Engine evaluates medications against 8 clinical dimensions:
// - Hemodynamic safety (vasopressor compatibility, hypotension risk)
// - Respiratory safety (respiratory depression, bronchospasm)
// - Renal safety (nephrotoxicity, CRRT drug removal)
// - Hepatic safety (hepatotoxicity, CYP interactions)
// - Coagulation safety (bleeding risk, anticoagulant interactions)
// - Neurological safety (sedation, seizure risk)
// - Fluid balance safety (volume effects, electrolytes)
// - Infection safety (antibiotic stewardship, sepsis protocols)
package advisor

import (
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ============================================================================
// ICU Safety Rule Definitions
// ============================================================================

// ICUSafetyRule represents a single ICU-specific safety rule
type ICUSafetyRule struct {
	ID              string            `json:"id"`
	Name            string            `json:"name"`
	Category        ICURuleCategory   `json:"category"`
	Dimension       string            `json:"dimension"`        // Which ICU dimension it applies to
	Severity        RuleSeverity      `json:"severity"`         // Rule severity if triggered
	TargetDrugs     []string          `json:"target_drugs"`     // RxNorm codes or drug classes
	DrugClass       string            `json:"drug_class"`       // Alternative: match by class
	Condition       ICURuleCondition  `json:"condition"`        // When to trigger
	Action          ICURuleAction     `json:"action"`           // What action to take
	Recommendation  string            `json:"recommendation"`   // Clinical recommendation
	AlternativeDrugs []string         `json:"alternative_drugs,omitempty"`
	EvidenceLevel   string            `json:"evidence_level"`   // FDA, Guideline, Expert
	KBSource        string            `json:"kb_source"`        // KB-X reference
	Active          bool              `json:"active"`
}

// ICURuleCategory categorizes safety rules
type ICURuleCategory string

const (
	RuleCategoryHemodynamic   ICURuleCategory = "HEMODYNAMIC"
	RuleCategoryRespiratory   ICURuleCategory = "RESPIRATORY"
	RuleCategoryRenal         ICURuleCategory = "RENAL"
	RuleCategoryHepatic       ICURuleCategory = "HEPATIC"
	RuleCategoryCoagulation   ICURuleCategory = "COAGULATION"
	RuleCategoryNeurological  ICURuleCategory = "NEUROLOGICAL"
	RuleCategoryFluidBalance  ICURuleCategory = "FLUID_BALANCE"
	RuleCategoryInfection     ICURuleCategory = "INFECTION"
	RuleCategoryCRRT          ICURuleCategory = "CRRT"
	RuleCategoryMultiOrgan    ICURuleCategory = "MULTI_ORGAN"
)

// RuleSeverity indicates the impact of rule violation
type RuleSeverity string

const (
	SeverityInfo     RuleSeverity = "INFO"       // Informational only
	SeverityCaution  RuleSeverity = "CAUTION"    // Use with caution
	SeverityWarning  RuleSeverity = "WARNING"    // Requires acknowledgment
	SeverityBlock    RuleSeverity = "BLOCK"      // Hard block - requires override
	SeverityCritical RuleSeverity = "CRITICAL"   // Absolute contraindication
)

// ICURuleCondition defines when a rule triggers
type ICURuleCondition struct {
	ConditionType    string             `json:"condition_type"`    // threshold, state, combo
	Dimension        string             `json:"dimension"`
	Parameter        string             `json:"parameter"`         // Specific parameter to check
	Operator         string             `json:"operator"`          // gt, lt, gte, lte, eq, in
	Value            interface{}        `json:"value"`             // Threshold value or state
	SecondaryCheck   *ICURuleCondition  `json:"secondary_check,omitempty"` // AND condition
	OrCondition      *ICURuleCondition  `json:"or_condition,omitempty"`    // OR condition
}

// ICURuleAction defines what happens when rule triggers
type ICURuleAction struct {
	ActionType     string   `json:"action_type"`     // block, warn, adjust, monitor
	BlockLevel     string   `json:"block_level"`     // hard, soft
	RequiresAck    bool     `json:"requires_ack"`
	DoseAdjustment *float64 `json:"dose_adjustment,omitempty"` // Percentage adjustment
	MonitoringReqs []string `json:"monitoring_reqs,omitempty"`
	TaskGeneration bool     `json:"task_generation"`
}

// ============================================================================
// ICU Safety Rules Engine
// ============================================================================

// ICUSafetyRulesEngine evaluates medications against ICU safety rules
type ICUSafetyRulesEngine struct {
	rules              []ICUSafetyRule
	drugClassMappings  map[string][]string // drug code -> classes
	enabled            bool
}

// NewICUSafetyRulesEngine creates a new ICU safety rules engine with default rules
func NewICUSafetyRulesEngine() *ICUSafetyRulesEngine {
	engine := &ICUSafetyRulesEngine{
		rules:             loadDefaultICURules(),
		drugClassMappings: loadDrugClassMappings(),
		enabled:           true,
	}
	return engine
}

// EvaluateMedication evaluates a single medication against all ICU rules
func (e *ICUSafetyRulesEngine) EvaluateMedication(
	medication ClinicalCode,
	icuState *ICUClinicalState,
) []ICURuleViolation {
	if !e.enabled || icuState == nil {
		return nil
	}

	violations := []ICURuleViolation{}

	for _, rule := range e.rules {
		if !rule.Active {
			continue
		}

		// Check if rule applies to this medication
		if !e.ruleAppliesToDrug(rule, medication) {
			continue
		}

		// Evaluate rule condition against ICU state
		if e.evaluateCondition(rule.Condition, icuState) {
			violation := ICURuleViolation{
				ID:             uuid.New(),
				RuleID:         rule.ID,
				RuleName:       rule.Name,
				Category:       rule.Category,
				Dimension:      rule.Dimension,
				Severity:       rule.Severity,
				Medication:     medication,
				TriggerValue:   e.getTriggerValue(rule.Condition, icuState),
				Threshold:      fmt.Sprintf("%v", rule.Condition.Value),
				Recommendation: rule.Recommendation,
				Alternatives:   e.getAlternatives(rule),
				Action:         rule.Action,
				KBSource:       rule.KBSource,
				EvidenceLevel:  rule.EvidenceLevel,
				Timestamp:      time.Now(),
			}
			violations = append(violations, violation)
		}
	}

	return violations
}

// EvaluateMultipleMedications evaluates multiple medications for combined risks
func (e *ICUSafetyRulesEngine) EvaluateMultipleMedications(
	medications []ClinicalCode,
	icuState *ICUClinicalState,
) ICUSafetyEvaluation {
	evaluation := ICUSafetyEvaluation{
		ID:              uuid.New(),
		ICUStateID:      icuState.ID,
		EvaluatedAt:     time.Now(),
		Violations:      []ICURuleViolation{},
		HardBlocks:      []ICUMedBlock{},
		Warnings:        []ICUMedWarning{},
		DoseAdjustments: []ICUDoseAdjustment{},
		SafetyScore:     100.0,
	}

	// Evaluate each medication
	for _, med := range medications {
		violations := e.EvaluateMedication(med, icuState)
		evaluation.Violations = append(evaluation.Violations, violations...)

		// Process violations into blocks/warnings
		for _, v := range violations {
			switch v.Severity {
			case SeverityCritical, SeverityBlock:
				evaluation.HardBlocks = append(evaluation.HardBlocks, e.violationToBlock(v))
			case SeverityWarning:
				evaluation.Warnings = append(evaluation.Warnings, e.violationToWarning(v))
			}

			// Check for dose adjustments
			if v.Action.DoseAdjustment != nil {
				adjustment := ICUDoseAdjustment{
					Medication:        v.Medication,
					OriginalDose:      100.0, // Placeholder - actual dose from order
					AdjustedPercent:   *v.Action.DoseAdjustment,
					Reason:            v.Recommendation,
					Dimension:         v.Dimension,
					MonitoringRequired: v.Action.MonitoringReqs,
				}
				evaluation.DoseAdjustments = append(evaluation.DoseAdjustments, adjustment)
			}
		}
	}

	// Calculate composite safety score
	evaluation.SafetyScore = e.calculateSafetyScore(evaluation)

	// Determine overall disposition
	evaluation.Disposition = e.determineDisposition(evaluation)

	return evaluation
}

// ============================================================================
// Rule Evaluation Logic
// ============================================================================

// ruleAppliesToDrug checks if a rule applies to a specific medication
func (e *ICUSafetyRulesEngine) ruleAppliesToDrug(rule ICUSafetyRule, med ClinicalCode) bool {
	// Check direct drug code match
	for _, target := range rule.TargetDrugs {
		if strings.EqualFold(med.Code, target) {
			return true
		}
	}

	// Check drug class match
	if rule.DrugClass != "" {
		drugClasses := e.drugClassMappings[strings.ToLower(med.Code)]
		for _, class := range drugClasses {
			if strings.EqualFold(class, rule.DrugClass) {
				return true
			}
		}
		// Also check by drug name patterns
		if e.matchesDrugClassByName(med.Display, rule.DrugClass) {
			return true
		}
	}

	return false
}

// evaluateCondition evaluates a rule condition against ICU state
func (e *ICUSafetyRulesEngine) evaluateCondition(cond ICURuleCondition, state *ICUClinicalState) bool {
	result := e.evaluateSingleCondition(cond, state)

	// Check secondary AND condition
	if result && cond.SecondaryCheck != nil {
		result = result && e.evaluateCondition(*cond.SecondaryCheck, state)
	}

	// Check OR condition
	if !result && cond.OrCondition != nil {
		result = e.evaluateCondition(*cond.OrCondition, state)
	}

	return result
}

// evaluateSingleCondition evaluates one condition without AND/OR chaining
func (e *ICUSafetyRulesEngine) evaluateSingleCondition(cond ICURuleCondition, state *ICUClinicalState) bool {
	paramValue := e.getParameterValue(cond.Dimension, cond.Parameter, state)
	if paramValue == nil {
		return false
	}

	switch cond.Operator {
	case "gt":
		return compareFloat(paramValue, cond.Value, ">")
	case "lt":
		return compareFloat(paramValue, cond.Value, "<")
	case "gte":
		return compareFloat(paramValue, cond.Value, ">=")
	case "lte":
		return compareFloat(paramValue, cond.Value, "<=")
	case "eq":
		return compareValue(paramValue, cond.Value)
	case "neq":
		return !compareValue(paramValue, cond.Value)
	case "in":
		return valueIn(paramValue, cond.Value)
	case "bool":
		if b, ok := paramValue.(bool); ok {
			if expected, ok := cond.Value.(bool); ok {
				return b == expected
			}
		}
	}

	return false
}

// getParameterValue extracts parameter value from ICU state
func (e *ICUSafetyRulesEngine) getParameterValue(dimension, parameter string, state *ICUClinicalState) interface{} {
	switch dimension {
	case "hemodynamic":
		return e.getHemodynamicParam(parameter, state.Hemodynamic)
	case "respiratory":
		return e.getRespiratoryParam(parameter, state.Respiratory)
	case "renal":
		return e.getRenalParam(parameter, state.Renal)
	case "hepatic":
		return e.getHepaticParam(parameter, state.Hepatic)
	case "coagulation":
		return e.getCoagulationParam(parameter, state.Coagulation)
	case "neurological":
		return e.getNeurologicalParam(parameter, state.Neurological)
	case "fluid_balance":
		return e.getFluidBalanceParam(parameter, state.FluidBalance)
	case "infection":
		return e.getInfectionParam(parameter, state.Infection)
	case "composite":
		return e.getCompositeParam(parameter, state)
	}
	return nil
}

// Dimension-specific parameter getters
func (e *ICUSafetyRulesEngine) getHemodynamicParam(param string, h HemodynamicState) interface{} {
	switch param {
	case "map":
		return h.MAP
	case "systolic_bp":
		return h.SystolicBP
	case "shock_state":
		return string(h.ShockState)
	case "vasopressor_req":
		return string(h.VasopressorReq)
	case "stability":
		return string(h.Stability)
	case "on_vasopressors":
		return h.VasopressorReq != VasopressorNone
	case "hemodynamic_score":
		return h.HemodynamicScore
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getRespiratoryParam(param string, r RespiratoryState) interface{} {
	switch param {
	case "spo2":
		return r.SpO2
	case "fio2":
		return r.FiO2
	case "pf_ratio":
		if r.PaO2FiO2Ratio != nil {
			return *r.PaO2FiO2Ratio
		}
		return nil
	case "ventilator_mode":
		return string(r.VentilatorMode)
	case "on_mechanical_vent":
		return r.VentilatorMode != VentModeNone && r.VentilatorMode != VentModeNC
	case "ards_severity":
		if r.ARDSSeverity != nil {
			return string(*r.ARDSSeverity)
		}
		return nil
	case "on_ecmo":
		return r.ECMO
	case "respiratory_score":
		return r.RespiratoryScore
	case "oxygenation_risk":
		return string(r.OxygenationRisk)
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getRenalParam(param string, r RenalState) interface{} {
	switch param {
	case "egfr":
		return r.EGFR
	case "creatinine":
		return r.Creatinine
	case "potassium":
		return r.Potassium
	case "sodium":
		return r.Sodium
	case "urine_output_kg":
		return r.UrineOutputKg
	case "aki_stage":
		if r.AKIStage != nil {
			return string(*r.AKIStage)
		}
		return nil
	case "rrt_status":
		return string(r.RRTStatus)
	case "on_crrt":
		return r.RRTStatus == RRTCRRT
	case "renal_score":
		return r.RenalScore
	case "nephrotoxic_risk":
		return string(r.NephrotoxicRisk)
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getHepaticParam(param string, h HepaticState) interface{} {
	switch param {
	case "total_bilirubin":
		return h.TotalBilirubin
	case "ast":
		return h.AST
	case "alt":
		return h.ALT
	case "inr":
		return h.INR
	case "albumin":
		return h.Albumin
	case "child_pugh_class":
		if h.ChildPughClass != nil {
			return *h.ChildPughClass
		}
		return nil
	case "meld_score":
		if h.MELDScore != nil {
			return *h.MELDScore
		}
		return nil
	case "he_grade":
		if h.HEGrade != nil {
			return string(*h.HEGrade)
		}
		return nil
	case "metabolism_impaired":
		return h.MetabolismImpaired
	case "cyp3a4_status":
		return h.CYP3A4Status
	case "hepatic_score":
		return h.HepaticScore
	case "hepatotoxic_risk":
		return string(h.HepatotoxicRisk)
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getCoagulationParam(param string, c CoagulationState) interface{} {
	switch param {
	case "inr":
		return c.INR
	case "ptt":
		return c.PTT
	case "platelets":
		return c.Platelets
	case "hemoglobin":
		return c.Hemoglobin
	case "fibrinogen":
		if c.Fibrinogen != nil {
			return *c.Fibrinogen
		}
		return nil
	case "bleeding_risk":
		return string(c.BleedingRisk)
	case "thrombosis_risk":
		return string(c.ThrombosisRisk)
	case "anticoag_status":
		return string(c.AnticoagStatus)
	case "hit_risk":
		return c.HITRisk
	case "coag_score":
		return c.CoagScore
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getNeurologicalParam(param string, n NeurologicalState) interface{} {
	switch param {
	case "gcs":
		return n.GCS
	case "rass":
		if n.RASSScore != nil {
			return *n.RASSScore
		}
		return nil
	case "cam_icu_positive":
		if n.CAMICUPositive != nil {
			return *n.CAMICUPositive
		}
		return nil
	case "icp_monitored":
		return n.ICPMonitored
	case "icp_value":
		if n.ICPValue != nil {
			return *n.ICPValue
		}
		return nil
	case "seizure_recent":
		return n.SeizureRecent
	case "neurological_score":
		return n.NeurologicalScore
	case "delirium_risk":
		return string(n.DeliriumRisk)
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getFluidBalanceParam(param string, f FluidBalanceState) interface{} {
	switch param {
	case "net_balance_24h":
		return f.NetBalance24h
	case "cumulative_balance":
		return f.CumulativeBalance
	case "volume_status":
		return string(f.VolumeStatus)
	case "edema_grade":
		if f.EdemaGrade != nil {
			return *f.EdemaGrade
		}
		return nil
	case "overload_risk":
		return string(f.OverloadRisk)
	case "fluid_score":
		return f.FluidScore
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getInfectionParam(param string, i InfectionState) interface{} {
	switch param {
	case "temperature":
		return i.Temperature
	case "wbc":
		return i.WBC
	case "procalcitonin":
		if i.Procalcitonin != nil {
			return *i.Procalcitonin
		}
		return nil
	case "lactate":
		if i.Lactate != nil {
			return *i.Lactate
		}
		return nil
	case "sepsis_status":
		return string(i.SepsisStatus)
	case "septic_shock":
		return i.SepticShock
	case "qsofa":
		if i.qSOFA != nil {
			return *i.qSOFA
		}
		return nil
	case "on_antibiotics":
		return i.OnAntibiotics
	case "antibiotic_days":
		return i.AntibioticDays
	case "sepsis_risk":
		return string(i.SepsisRisk)
	case "infection_score":
		return i.InfectionScore
	}
	return nil
}

func (e *ICUSafetyRulesEngine) getCompositeParam(param string, state *ICUClinicalState) interface{} {
	switch param {
	case "icu_acuity_score":
		return state.ICUAcuityScore
	case "is_critical":
		return state.IsCritical()
	case "is_high_acuity":
		return state.IsHighAcuity()
	case "trend_direction":
		return string(state.TrendDirection)
	case "sofa_score":
		if state.SOFAScore != nil {
			return *state.SOFAScore
		}
		return nil
	case "apache_ii_score":
		if state.APACHEIIScore != nil {
			return *state.APACHEIIScore
		}
		return nil
	case "on_crrt":
		return state.RequiresCRRTDoseAdjustment()
	case "on_vasopressors":
		return state.HasActiveVasopressors()
	case "on_mechanical_vent":
		return state.IsOnMechanicalVentilation()
	case "has_sepsis":
		return state.HasActiveSepsis()
	}
	return nil
}

// ============================================================================
// Helper Functions
// ============================================================================

func (e *ICUSafetyRulesEngine) getTriggerValue(cond ICURuleCondition, state *ICUClinicalState) string {
	val := e.getParameterValue(cond.Dimension, cond.Parameter, state)
	if val == nil {
		return "unknown"
	}
	return fmt.Sprintf("%v", val)
}

func (e *ICUSafetyRulesEngine) getAlternatives(rule ICUSafetyRule) []ClinicalCode {
	alternatives := []ClinicalCode{}
	for _, alt := range rule.AlternativeDrugs {
		alternatives = append(alternatives, ClinicalCode{
			System:  "http://www.nlm.nih.gov/research/umls/rxnorm",
			Code:    alt,
			Display: alt, // Would be resolved from drug database
		})
	}
	return alternatives
}

func (e *ICUSafetyRulesEngine) violationToBlock(v ICURuleViolation) ICUMedBlock {
	return ICUMedBlock{
		ID:               uuid.New(),
		BlockReason:      e.categoryToBlockReason(v.Category),
		Medication:       v.Medication,
		TriggerDimension: v.Dimension,
		TriggerValue:     v.TriggerValue,
		SafetyRationale:  v.Recommendation,
		Alternative:      firstOrNil(v.Alternatives),
		RequiresAck:      v.Action.RequiresAck,
		KBSource:         v.KBSource,
		RuleID:           v.RuleID,
		CreatedAt:        time.Now(),
	}
}

func (e *ICUSafetyRulesEngine) violationToWarning(v ICURuleViolation) ICUMedWarning {
	return ICUMedWarning{
		ID:             uuid.New(),
		WarningType:    ICUWarningType(v.Category),
		Medication:     v.Medication,
		Dimension:      v.Dimension,
		Message:        v.Recommendation,
		TriggerValue:   v.TriggerValue,
		Severity:       string(v.Severity),
		RequiresAck:    v.Action.RequiresAck,
		MonitoringReqs: v.Action.MonitoringReqs,
		CreatedAt:      time.Now(),
	}
}

func (e *ICUSafetyRulesEngine) categoryToBlockReason(cat ICURuleCategory) ICUBlockReason {
	switch cat {
	case RuleCategoryHemodynamic:
		return BlockHemodynamicInstability
	case RuleCategoryRespiratory:
		return BlockRespiratoryConcern
	case RuleCategoryRenal:
		return BlockRenalContraindication
	case RuleCategoryHepatic:
		return BlockHepaticContraindication
	case RuleCategoryCoagulation:
		return BlockCoagContraindication
	case RuleCategoryNeurological:
		return BlockNeurologicalRisk
	case RuleCategoryInfection:
		return BlockSepsisProtocol
	case RuleCategoryCRRT:
		return BlockCRRTInteraction
	default:
		return BlockHemodynamicInstability
	}
}

func (e *ICUSafetyRulesEngine) calculateSafetyScore(eval ICUSafetyEvaluation) float64 {
	score := 100.0

	// Deduct for hard blocks (critical)
	score -= float64(len(eval.HardBlocks)) * 25.0

	// Deduct for warnings
	score -= float64(len(eval.Warnings)) * 10.0

	// Deduct for dose adjustments needed
	score -= float64(len(eval.DoseAdjustments)) * 5.0

	if score < 0 {
		score = 0
	}
	return score
}

func (e *ICUSafetyRulesEngine) determineDisposition(eval ICUSafetyEvaluation) DispositionCode {
	if len(eval.HardBlocks) > 0 {
		return DispositionICUCriticalHardStop
	}
	if len(eval.Warnings) > 0 {
		return DispositionICUHighRisk
	}
	if len(eval.DoseAdjustments) > 0 {
		return DispositionICUDoseAdjustment
	}
	return DispositionICUSafe
}

func (e *ICUSafetyRulesEngine) matchesDrugClassByName(drugName, classPattern string) bool {
	name := strings.ToLower(drugName)
	pattern := strings.ToLower(classPattern)

	classPatterns := map[string][]string{
		"nsaid":          {"ibuprofen", "naproxen", "ketorolac", "diclofenac", "indomethacin", "celecoxib"},
		"aminoglycoside": {"gentamicin", "tobramycin", "amikacin", "neomycin", "streptomycin"},
		"ace_inhibitor":  {"lisinopril", "enalapril", "captopril", "ramipril", "benazepril", "quinapril"},
		"arb":            {"losartan", "valsartan", "olmesartan", "candesartan", "irbesartan", "telmisartan"},
		"beta_blocker":   {"metoprolol", "atenolol", "propranolol", "carvedilol", "bisoprolol", "labetalol"},
		"calcium_channel":{"amlodipine", "nifedipine", "diltiazem", "verapamil", "nicardipine", "clevidipine"},
		"opioid":         {"morphine", "fentanyl", "hydromorphone", "oxycodone", "methadone", "hydrocodone"},
		"benzodiazepine": {"midazolam", "lorazepam", "diazepam", "alprazolam", "clonazepam"},
		"anticoagulant":  {"heparin", "warfarin", "enoxaparin", "rivaroxaban", "apixaban", "dabigatran"},
		"vasopressor":    {"norepinephrine", "epinephrine", "vasopressin", "phenylephrine", "dopamine"},
		"sedative":       {"propofol", "dexmedetomidine", "ketamine", "midazolam", "lorazepam"},
		"fluoroquinolone":{"ciprofloxacin", "levofloxacin", "moxifloxacin", "ofloxacin"},
		"contrast_agent": {"iohexol", "iopamidol", "iodixanol", "ioversol", "gadolinium"},
		"potassium_sparing": {"spironolactone", "eplerenone", "amiloride", "triamterene"},
	}

	if drugs, ok := classPatterns[pattern]; ok {
		for _, drug := range drugs {
			if strings.Contains(name, drug) {
				return true
			}
		}
	}
	return false
}

// Comparison helper functions
func compareFloat(val, threshold interface{}, op string) bool {
	v, ok1 := toFloat64(val)
	t, ok2 := toFloat64(threshold)
	if !ok1 || !ok2 {
		return false
	}
	switch op {
	case ">":
		return v > t
	case "<":
		return v < t
	case ">=":
		return v >= t
	case "<=":
		return v <= t
	case "==":
		return v == t
	}
	return false
}

func compareValue(val, expected interface{}) bool {
	return fmt.Sprintf("%v", val) == fmt.Sprintf("%v", expected)
}

func valueIn(val interface{}, set interface{}) bool {
	if arr, ok := set.([]string); ok {
		valStr := fmt.Sprintf("%v", val)
		for _, item := range arr {
			if valStr == item {
				return true
			}
		}
	}
	return false
}

func toFloat64(val interface{}) (float64, bool) {
	switch v := val.(type) {
	case float64:
		return v, true
	case float32:
		return float64(v), true
	case int:
		return float64(v), true
	case int64:
		return float64(v), true
	}
	return 0, false
}

func firstOrNil(codes []ClinicalCode) *ClinicalCode {
	if len(codes) > 0 {
		return &codes[0]
	}
	return nil
}

// ============================================================================
// Output Types
// ============================================================================

// ICURuleViolation represents a triggered safety rule
type ICURuleViolation struct {
	ID             uuid.UUID        `json:"id"`
	RuleID         string           `json:"rule_id"`
	RuleName       string           `json:"rule_name"`
	Category       ICURuleCategory  `json:"category"`
	Dimension      string           `json:"dimension"`
	Severity       RuleSeverity     `json:"severity"`
	Medication     ClinicalCode     `json:"medication"`
	TriggerValue   string           `json:"trigger_value"`
	Threshold      string           `json:"threshold"`
	Recommendation string           `json:"recommendation"`
	Alternatives   []ClinicalCode   `json:"alternatives,omitempty"`
	Action         ICURuleAction    `json:"action"`
	KBSource       string           `json:"kb_source"`
	EvidenceLevel  string           `json:"evidence_level"`
	Timestamp      time.Time        `json:"timestamp"`
}

// ICUMedWarning represents a non-blocking medication warning
type ICUMedWarning struct {
	ID             uuid.UUID      `json:"id"`
	WarningType    ICUWarningType `json:"warning_type"`
	Medication     ClinicalCode   `json:"medication"`
	Dimension      string         `json:"dimension"`
	Message        string         `json:"message"`
	TriggerValue   string         `json:"trigger_value"`
	Severity       string         `json:"severity"`
	RequiresAck    bool           `json:"requires_ack"`
	MonitoringReqs []string       `json:"monitoring_reqs,omitempty"`
	CreatedAt      time.Time      `json:"created_at"`
}

// ICUWarningType represents warning categories
type ICUWarningType string

// ICUDoseAdjustment represents a recommended dose adjustment
type ICUDoseAdjustment struct {
	Medication        ClinicalCode `json:"medication"`
	OriginalDose      float64      `json:"original_dose"`
	AdjustedPercent   float64      `json:"adjusted_percent"`  // e.g., 50 = reduce to 50%
	Reason            string       `json:"reason"`
	Dimension         string       `json:"dimension"`
	MonitoringRequired []string    `json:"monitoring_required,omitempty"`
}

// ICUSafetyEvaluation represents complete safety evaluation results
type ICUSafetyEvaluation struct {
	ID              uuid.UUID          `json:"id"`
	ICUStateID      uuid.UUID          `json:"icu_state_id"`
	EvaluatedAt     time.Time          `json:"evaluated_at"`
	Violations      []ICURuleViolation `json:"violations"`
	HardBlocks      []ICUMedBlock      `json:"hard_blocks"`
	Warnings        []ICUMedWarning    `json:"warnings"`
	DoseAdjustments []ICUDoseAdjustment `json:"dose_adjustments"`
	SafetyScore     float64            `json:"safety_score"` // 0-100
	Disposition     DispositionCode    `json:"disposition"`
}

// ============================================================================
// Default ICU Safety Rules (Phase 2 Core Rules)
// ============================================================================

func loadDefaultICURules() []ICUSafetyRule {
	return []ICUSafetyRule{
		// === HEMODYNAMIC RULES ===
		{
			ID:       "ICU-HEMO-001",
			Name:     "Beta-blocker in hypotension",
			Category: RuleCategoryHemodynamic,
			Dimension: "hemodynamic",
			Severity: SeverityBlock,
			DrugClass: "beta_blocker",
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "hemodynamic",
				Parameter:     "map",
				Operator:      "lt",
				Value:         65.0,
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Beta-blockers contraindicated with MAP < 65 mmHg. Consider vasopressor support first.",
			AlternativeDrugs: []string{},
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-HEMO-002",
			Name:     "Vasodilator in shock",
			Category: RuleCategoryHemodynamic,
			Dimension: "hemodynamic",
			Severity: SeverityCritical,
			DrugClass: "calcium_channel",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "hemodynamic",
				Parameter:     "shock_state",
				Operator:      "in",
				Value:         []string{"SEVERE", "REFRACTORY"},
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Calcium channel blockers contraindicated in severe/refractory shock.",
			EvidenceLevel:   "FDA",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-HEMO-003",
			Name:     "Negative inotrope with vasopressor dependency",
			Category: RuleCategoryHemodynamic,
			Dimension: "hemodynamic",
			Severity: SeverityBlock,
			DrugClass: "beta_blocker",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "hemodynamic",
				Parameter:     "vasopressor_req",
				Operator:      "in",
				Value:         []string{"HIGH_DOSE", "MAXIMAL_DOSE", "MULTIPLE"},
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Beta-blockers contraindicated with high-dose vasopressor requirements.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},

		// === RESPIRATORY RULES ===
		{
			ID:       "ICU-RESP-001",
			Name:     "Sedative with severe hypoxemia",
			Category: RuleCategoryRespiratory,
			Dimension: "respiratory",
			Severity: SeverityBlock,
			DrugClass: "sedative",
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "respiratory",
				Parameter:     "spo2",
				Operator:      "lt",
				Value:         88.0,
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "soft",
				RequiresAck:    true,
				TaskGeneration: true,
				MonitoringReqs: []string{"Continuous SpO2", "Respiratory rate q15min"},
			},
			Recommendation:  "Sedatives require caution with SpO2 < 88%. Ensure airway protection.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-RESP-002",
			Name:     "Opioid without secured airway",
			Category: RuleCategoryRespiratory,
			Dimension: "respiratory",
			Severity: SeverityWarning,
			DrugClass: "opioid",
			Condition: ICURuleCondition{
				ConditionType: "combo",
				Dimension:     "respiratory",
				Parameter:     "on_mechanical_vent",
				Operator:      "bool",
				Value:         false,
				SecondaryCheck: &ICURuleCondition{
					ConditionType: "threshold",
					Dimension:     "respiratory",
					Parameter:     "fio2",
					Operator:      "gte",
					Value:         0.5,
				},
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				DoseAdjustment: floatPtr(50.0), // 50% dose reduction
				MonitoringReqs: []string{"RR q1h", "SpO2 continuous", "Capnography if available"},
			},
			Recommendation:  "High FiO2 without intubation - consider reduced opioid dosing with enhanced monitoring.",
			EvidenceLevel:   "Expert",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-RESP-003",
			Name:     "Neuromuscular blocker in severe ARDS",
			Category: RuleCategoryRespiratory,
			Dimension: "respiratory",
			Severity: SeverityInfo,
			TargetDrugs: []string{"cisatracurium", "rocuronium", "vecuronium"},
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "respiratory",
				Parameter:     "ards_severity",
				Operator:      "eq",
				Value:         "SEVERE",
			},
			Action: ICURuleAction{
				ActionType:     "monitor",
				MonitoringReqs: []string{"Train-of-four q4h", "Daily awakening trial when stable"},
			},
			Recommendation:  "NMBAs may benefit severe ARDS (ACURASYS trial). Monitor for ICU-acquired weakness.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-6",
			Active:          true,
		},

		// === RENAL RULES ===
		{
			ID:       "ICU-RENAL-001",
			Name:     "Aminoglycoside in AKI Stage 2+",
			Category: RuleCategoryRenal,
			Dimension: "renal",
			Severity: SeverityBlock,
			DrugClass: "aminoglycoside",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "renal",
				Parameter:     "aki_stage",
				Operator:      "in",
				Value:         []string{"STAGE_2", "STAGE_3"},
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Aminoglycosides contraindicated in AKI Stage 2-3. Consider alternative antibiotics.",
			AlternativeDrugs: []string{"ceftriaxone", "piperacillin-tazobactam", "meropenem"},
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-16",
			Active:          true,
		},
		{
			ID:       "ICU-RENAL-002",
			Name:     "NSAID in renal impairment",
			Category: RuleCategoryRenal,
			Dimension: "renal",
			Severity: SeverityBlock,
			DrugClass: "nsaid",
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "renal",
				Parameter:     "egfr",
				Operator:      "lt",
				Value:         30.0,
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "NSAIDs contraindicated with eGFR < 30. Consider acetaminophen or non-nephrotoxic alternatives.",
			AlternativeDrugs: []string{"acetaminophen"},
			EvidenceLevel:   "FDA",
			KBSource:        "KB-16",
			Active:          true,
		},
		{
			ID:       "ICU-RENAL-003",
			Name:     "ACE/ARB in severe AKI",
			Category: RuleCategoryRenal,
			Dimension: "renal",
			Severity: SeverityBlock,
			DrugClass: "ace_inhibitor",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "renal",
				Parameter:     "aki_stage",
				Operator:      "eq",
				Value:         "STAGE_3",
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "soft",
				RequiresAck:    true,
			},
			Recommendation:  "Hold ACE inhibitors in AKI Stage 3. Reassess after renal function recovery.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-RENAL-004",
			Name:     "Contrast agent in AKI",
			Category: RuleCategoryRenal,
			Dimension: "renal",
			Severity: SeverityBlock,
			DrugClass: "contrast_agent",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "renal",
				Parameter:     "aki_stage",
				Operator:      "neq",
				Value:         "NONE",
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "IV contrast contraindicated in AKI. Consider alternative imaging or defer until renal recovery.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-16",
			Active:          true,
		},
		{
			ID:       "ICU-RENAL-005",
			Name:     "Potassium-sparing diuretic with hyperkalemia",
			Category: RuleCategoryRenal,
			Dimension: "renal",
			Severity: SeverityCritical,
			DrugClass: "potassium_sparing",
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "renal",
				Parameter:     "potassium",
				Operator:      "gt",
				Value:         5.5,
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Potassium-sparing diuretics contraindicated with K+ > 5.5 mmol/L. Treat hyperkalemia first.",
			EvidenceLevel:   "FDA",
			KBSource:        "KB-16",
			Active:          true,
		},

		// === CRRT RULES ===
		{
			ID:       "ICU-CRRT-001",
			Name:     "Drug dosing adjustment for CRRT",
			Category: RuleCategoryCRRT,
			Dimension: "renal",
			Severity: SeverityWarning,
			TargetDrugs: []string{"vancomycin", "piperacillin-tazobactam", "meropenem", "cefepime"},
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "renal",
				Parameter:     "on_crrt",
				Operator:      "bool",
				Value:         true,
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				MonitoringReqs: []string{"Drug levels", "CRRT effluent rate monitoring"},
			},
			Recommendation:  "CRRT alters drug clearance. Consult pharmacy for CRRT-specific dosing. Monitor drug levels.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-1",
			Active:          true,
		},

		// === HEPATIC RULES ===
		{
			ID:       "ICU-HEP-001",
			Name:     "Acetaminophen in liver failure",
			Category: RuleCategoryHepatic,
			Dimension: "hepatic",
			Severity: SeverityBlock,
			TargetDrugs: []string{"325383", "161", "acetaminophen"}, // RxNorm codes
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "hepatic",
				Parameter:     "total_bilirubin",
				Operator:      "gt",
				Value:         5.0,
				SecondaryCheck: &ICURuleCondition{
					ConditionType: "threshold",
					Dimension:     "hepatic",
					Parameter:     "inr",
					Operator:      "gt",
					Value:         2.0,
				},
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Acetaminophen contraindicated in severe liver failure (bili > 5, INR > 2). Consider opioid alternatives.",
			EvidenceLevel:   "FDA",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-HEP-002",
			Name:     "Sedative in hepatic encephalopathy",
			Category: RuleCategoryHepatic,
			Dimension: "hepatic",
			Severity: SeverityBlock,
			DrugClass: "benzodiazepine",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "hepatic",
				Parameter:     "he_grade",
				Operator:      "in",
				Value:         []string{"GRADE_2", "GRADE_3", "GRADE_4"},
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
			},
			Recommendation:  "Benzodiazepines worsen hepatic encephalopathy. Use propofol or dexmedetomidine if sedation required.",
			AlternativeDrugs: []string{"propofol", "dexmedetomidine"},
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-HEP-003",
			Name:     "CYP3A4 substrate with impaired metabolism",
			Category: RuleCategoryHepatic,
			Dimension: "hepatic",
			Severity: SeverityWarning,
			TargetDrugs: []string{"midazolam", "fentanyl", "tacrolimus", "cyclosporine"},
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "hepatic",
				Parameter:     "cyp3a4_status",
				Operator:      "eq",
				Value:         "severely_impaired",
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				DoseAdjustment: floatPtr(50.0), // Reduce to 50%
				MonitoringReqs: []string{"Drug levels if applicable", "Enhanced sedation monitoring"},
			},
			Recommendation:  "CYP3A4 impairment - reduce dose by 50% and monitor for toxicity.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-2",
			Active:          true,
		},

		// === COAGULATION RULES ===
		{
			ID:       "ICU-COAG-001",
			Name:     "Anticoagulant with severe thrombocytopenia",
			Category: RuleCategoryCoagulation,
			Dimension: "coagulation",
			Severity: SeverityBlock,
			DrugClass: "anticoagulant",
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "coagulation",
				Parameter:     "platelets",
				Operator:      "lt",
				Value:         50.0,
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Anticoagulation contraindicated with platelets < 50K. Address thrombocytopenia before anticoagulation.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-COAG-002",
			Name:     "Heparin with HIT risk",
			Category: RuleCategoryCoagulation,
			Dimension: "coagulation",
			Severity: SeverityCritical,
			TargetDrugs: []string{"heparin", "5224"}, // Heparin RxNorm
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "coagulation",
				Parameter:     "hit_risk",
				Operator:      "bool",
				Value:         true,
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Heparin contraindicated with HIT risk. Use argatroban or bivalirudin.",
			AlternativeDrugs: []string{"argatroban", "bivalirudin"},
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-COAG-003",
			Name:     "NSAID with high bleeding risk",
			Category: RuleCategoryCoagulation,
			Dimension: "coagulation",
			Severity: SeverityBlock,
			DrugClass: "nsaid",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "coagulation",
				Parameter:     "bleeding_risk",
				Operator:      "in",
				Value:         []string{"HIGH", "CRITICAL"},
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
			},
			Recommendation:  "NSAIDs increase bleeding risk. Use acetaminophen or consider IV ketorolac only if essential.",
			AlternativeDrugs: []string{"acetaminophen"},
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},

		// === NEUROLOGICAL RULES ===
		{
			ID:       "ICU-NEURO-001",
			Name:     "Sedative with low GCS",
			Category: RuleCategoryNeurological,
			Dimension: "neurological",
			Severity: SeverityWarning,
			DrugClass: "sedative",
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "neurological",
				Parameter:     "gcs",
				Operator:      "lte",
				Value:         8,
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				MonitoringReqs: []string{"GCS q2h", "Pupillary checks q2h", "Consider sedation vacation"},
			},
			Recommendation:  "GCS ≤ 8 requires secured airway. If intubated, continue sedation per protocol. Daily awakening trials.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-NEURO-002",
			Name:     "Fluoroquinolone with seizure history",
			Category: RuleCategoryNeurological,
			Dimension: "neurological",
			Severity: SeverityWarning,
			DrugClass: "fluoroquinolone",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "neurological",
				Parameter:     "seizure_recent",
				Operator:      "bool",
				Value:         true,
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				MonitoringReqs: []string{"Seizure precautions", "EEG if available"},
			},
			Recommendation:  "Fluoroquinolones lower seizure threshold. Consider alternative antibiotic if possible.",
			AlternativeDrugs: []string{"ceftriaxone", "azithromycin"},
			EvidenceLevel:   "FDA",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-NEURO-003",
			Name:     "Anticholinergic with delirium",
			Category: RuleCategoryNeurological,
			Dimension: "neurological",
			Severity: SeverityWarning,
			TargetDrugs: []string{"diphenhydramine", "promethazine", "hydroxyzine", "scopolamine"},
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "neurological",
				Parameter:     "cam_icu_positive",
				Operator:      "bool",
				Value:         true,
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
			},
			Recommendation:  "Anticholinergics worsen delirium. Avoid if possible. Consider non-pharmacologic interventions.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-NEURO-004",
			Name:     "Vasodilator with elevated ICP",
			Category: RuleCategoryNeurological,
			Dimension: "neurological",
			Severity: SeverityCritical,
			DrugClass: "calcium_channel",
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "neurological",
				Parameter:     "icp_value",
				Operator:      "gt",
				Value:         20.0,
			},
			Action: ICURuleAction{
				ActionType:     "block",
				BlockLevel:     "hard",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Vasodilators contraindicated with ICP > 20 mmHg. May worsen cerebral edema.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},

		// === FLUID BALANCE RULES ===
		{
			ID:       "ICU-FLUID-001",
			Name:     "IV fluid bolus in fluid overload",
			Category: RuleCategoryFluidBalance,
			Dimension: "fluid_balance",
			Severity: SeverityWarning,
			TargetDrugs: []string{"normal_saline", "lactated_ringers", "albumin"},
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "fluid_balance",
				Parameter:     "volume_status",
				Operator:      "eq",
				Value:         "HYPERVOLEMIC",
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				MonitoringReqs: []string{"I/O q1h", "Daily weight", "Consider diuresis"},
			},
			Recommendation:  "Patient is fluid overloaded. Avoid additional IV fluids. Consider diuresis if hemodynamically stable.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-FLUID-002",
			Name:     "Large volume in severe edema",
			Category: RuleCategoryFluidBalance,
			Dimension: "fluid_balance",
			Severity: SeverityWarning,
			TargetDrugs: []string{"normal_saline", "lactated_ringers"},
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "fluid_balance",
				Parameter:     "edema_grade",
				Operator:      "gte",
				Value:         3,
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
			},
			Recommendation:  "Severe edema (grade 3+) - minimize crystalloid use. Consider albumin if volume needed.",
			EvidenceLevel:   "Expert",
			KBSource:        "KB-4",
			Active:          true,
		},

		// === INFECTION/SEPSIS RULES ===
		{
			ID:       "ICU-SEPSIS-001",
			Name:     "Broad-spectrum abx deescalation overdue",
			Category: RuleCategoryInfection,
			Dimension: "infection",
			Severity: SeverityInfo,
			TargetDrugs: []string{"meropenem", "piperacillin-tazobactam", "vancomycin"},
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "infection",
				Parameter:     "antibiotic_days",
				Operator:      "gt",
				Value:         72,
				SecondaryCheck: &ICURuleCondition{
					ConditionType: "state",
					Dimension:     "infection",
					Parameter:     "sepsis_status",
					Operator:      "neq",
					Value:         "SHOCK",
				},
			},
			Action: ICURuleAction{
				ActionType:     "monitor",
				TaskGeneration: true,
				MonitoringReqs: []string{"Review cultures", "Procalcitonin trend", "Consider deescalation"},
			},
			Recommendation:  "Broad-spectrum antibiotics > 72h. Review cultures and consider deescalation per stewardship.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-3",
			Active:          true,
		},
		{
			ID:       "ICU-SEPSIS-002",
			Name:     "Vasopressor in septic shock",
			Category: RuleCategoryInfection,
			Dimension: "infection",
			Severity: SeverityInfo,
			DrugClass: "vasopressor",
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "infection",
				Parameter:     "septic_shock",
				Operator:      "bool",
				Value:         true,
			},
			Action: ICURuleAction{
				ActionType:     "monitor",
				MonitoringReqs: []string{"MAP target > 65", "Lactate q2-4h", "Urine output hourly"},
			},
			Recommendation:  "Vasopressor use in septic shock - follow Surviving Sepsis Campaign targets.",
			EvidenceLevel:   "Guideline",
			KBSource:        "KB-3",
			Active:          true,
		},

		// === MULTI-ORGAN RULES ===
		{
			ID:       "ICU-MULTI-001",
			Name:     "High acuity medication review",
			Category: RuleCategoryMultiOrgan,
			Dimension: "composite",
			Severity: SeverityWarning,
			TargetDrugs: []string{}, // All drugs
			DrugClass:  "",          // All drugs
			Condition: ICURuleCondition{
				ConditionType: "threshold",
				Dimension:     "composite",
				Parameter:     "icu_acuity_score",
				Operator:      "gte",
				Value:         80.0,
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				TaskGeneration: true,
				MonitoringReqs: []string{"Pharmacy review", "ICU team discussion"},
			},
			Recommendation:  "Very high ICU acuity (≥80). All medication orders require pharmacy and ICU team review.",
			EvidenceLevel:   "Expert",
			KBSource:        "KB-4",
			Active:          true,
		},
		{
			ID:       "ICU-MULTI-002",
			Name:     "Medication in critical deterioration",
			Category: RuleCategoryMultiOrgan,
			Dimension: "composite",
			Severity: SeverityWarning,
			TargetDrugs: []string{}, // All drugs
			DrugClass:  "",          // All drugs
			Condition: ICURuleCondition{
				ConditionType: "state",
				Dimension:     "composite",
				Parameter:     "trend_direction",
				Operator:      "eq",
				Value:         "CRITICAL",
			},
			Action: ICURuleAction{
				ActionType:     "warn",
				RequiresAck:    true,
				TaskGeneration: true,
			},
			Recommendation:  "Patient in critical deterioration. Urgent medication review and goals of care discussion.",
			EvidenceLevel:   "Expert",
			KBSource:        "KB-4",
			Active:          true,
		},
	}
}

// loadDrugClassMappings returns drug code to class mappings
func loadDrugClassMappings() map[string][]string {
	return map[string][]string{
		// Beta-blockers
		"metoprolol":  {"beta_blocker"},
		"68950":       {"beta_blocker"}, // metoprolol succinate
		"atenolol":    {"beta_blocker"},
		"carvedilol":  {"beta_blocker"},
		"labetalol":   {"beta_blocker"},
		"propranolol": {"beta_blocker"},

		// Calcium channel blockers
		"amlodipine":   {"calcium_channel"},
		"17767":        {"calcium_channel"}, // amlodipine
		"diltiazem":    {"calcium_channel"},
		"nifedipine":   {"calcium_channel"},
		"nicardipine":  {"calcium_channel"},
		"clevidipine":  {"calcium_channel"},

		// Opioids
		"morphine":      {"opioid"},
		"7052":          {"opioid"}, // morphine
		"fentanyl":      {"opioid"},
		"4337":          {"opioid"}, // fentanyl
		"hydromorphone": {"opioid"},

		// Aminoglycosides
		"gentamicin": {"aminoglycoside"},
		"4750":       {"aminoglycoside"},
		"tobramycin": {"aminoglycoside"},
		"amikacin":   {"aminoglycoside"},

		// NSAIDs
		"ibuprofen":    {"nsaid"},
		"5640":         {"nsaid"},
		"ketorolac":    {"nsaid"},
		"6691":         {"nsaid"},
		"naproxen":     {"nsaid"},
		"indomethacin": {"nsaid"},

		// ACE Inhibitors
		"lisinopril": {"ace_inhibitor"},
		"29046":      {"ace_inhibitor"},
		"enalapril":  {"ace_inhibitor"},
		"captopril":  {"ace_inhibitor"},

		// ARBs
		"losartan":  {"arb"},
		"52175":     {"arb"},
		"valsartan": {"arb"},

		// Benzodiazepines
		"midazolam": {"benzodiazepine", "sedative"},
		"6960":      {"benzodiazepine", "sedative"},
		"lorazepam": {"benzodiazepine", "sedative"},
		"diazepam":  {"benzodiazepine", "sedative"},

		// Sedatives (non-benzo)
		"propofol":        {"sedative"},
		"8745":            {"sedative"},
		"dexmedetomidine": {"sedative"},
		"ketamine":        {"sedative"},

		// Anticoagulants
		"heparin":     {"anticoagulant"},
		"5224":        {"anticoagulant"},
		"enoxaparin":  {"anticoagulant"},
		"67108":       {"anticoagulant"},
		"warfarin":    {"anticoagulant"},
		"11289":       {"anticoagulant"},
		"rivaroxaban": {"anticoagulant"},
		"apixaban":    {"anticoagulant"},

		// Vasopressors
		"norepinephrine": {"vasopressor"},
		"7512":           {"vasopressor"},
		"epinephrine":    {"vasopressor"},
		"vasopressin":    {"vasopressor"},
		"phenylephrine":  {"vasopressor"},
		"dopamine":       {"vasopressor"},

		// Fluoroquinolones
		"ciprofloxacin":  {"fluoroquinolone"},
		"2551":           {"fluoroquinolone"},
		"levofloxacin":   {"fluoroquinolone"},
		"moxifloxacin":   {"fluoroquinolone"},

		// Potassium-sparing
		"spironolactone": {"potassium_sparing"},
		"9997":           {"potassium_sparing"},
		"eplerenone":     {"potassium_sparing"},
	}
}

// Helper for creating float pointers
func floatPtr(v float64) *float64 {
	return &v
}
