package services

import "fmt"

// ProtocolTemplate defines a registered protocol template.
type ProtocolTemplate struct {
	ProtocolID        string             `json:"protocol_id"`
	ProtocolName      string             `json:"protocol_name"`
	Version           string             `json:"version"`
	Category          string             `json:"category"`
	Subcategory       string             `json:"subcategory"`
	Phases            []PhaseDefinition  `json:"phases"`
	EntryCriteria     []Criterion        `json:"entry_criteria"`
	ExclusionCriteria []ExclusionRule    `json:"exclusion_criteria"`
	ConcurrentWith    []string           `json:"concurrent_with"`
	SuccessCriteria   []Criterion        `json:"success_criteria"`
	EscalationTrigger string             `json:"escalation_trigger"`
}

// PhaseDefinition describes one phase in a protocol.
type PhaseDefinition struct {
	ID           string `json:"id"`
	DurationDays int    `json:"duration_days"`
	ExtendableTo int    `json:"extendable_to"`
	AutoAdvance  bool   `json:"auto_advance"`
}

// Criterion is a numeric comparison for entry/success evaluation.
type Criterion struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"` // >= | <= | < | > | ==
	Value    float64 `json:"value"`
}

// ExclusionRule blocks protocol entry.
type ExclusionRule struct {
	Field    string  `json:"field"`
	Operator string  `json:"operator"`
	Value    float64 `json:"value"`
	RuleCode string  `json:"rule_code"` // LS-01, LS-14, LS-15, etc.
}

// ProtocolRegistry holds all registered protocol templates.
type ProtocolRegistry struct {
	templates map[string]*ProtocolTemplate
}

// NewProtocolRegistry creates a registry pre-loaded with M3-PRP and M3-VFRP.
func NewProtocolRegistry() *ProtocolRegistry {
	r := &ProtocolRegistry{templates: make(map[string]*ProtocolTemplate)}
	r.registerPRP()
	r.registerVFRP()
	return r
}

// GetTemplate returns a protocol template by ID.
func (r *ProtocolRegistry) GetTemplate(protocolID string) (*ProtocolTemplate, error) {
	tmpl, ok := r.templates[protocolID]
	if !ok {
		return nil, fmt.Errorf("protocol %s not registered", protocolID)
	}
	return tmpl, nil
}

// CheckEntry evaluates entry criteria and exclusion rules.
// Returns (eligible, reason). reason is empty if eligible, or the blocking rule code.
func (r *ProtocolRegistry) CheckEntry(protocolID string, numericFields map[string]float64, boolFields map[string]bool) (bool, string) {
	tmpl, err := r.GetTemplate(protocolID)
	if err != nil {
		return false, "UNKNOWN_PROTOCOL"
	}

	// Check exclusion criteria first
	for _, excl := range tmpl.ExclusionCriteria {
		val, ok := numericFields[excl.Field]
		if !ok {
			// Check bool fields
			if boolVal, boolOk := boolFields[excl.Field]; boolOk && boolVal {
				return false, excl.RuleCode
			}
			continue
		}
		if evaluateNumeric(val, excl.Operator, excl.Value) {
			return false, excl.RuleCode
		}
	}

	// Check at least one entry criterion is met
	anyMet := false
	for _, entry := range tmpl.EntryCriteria {
		val, ok := numericFields[entry.Field]
		if !ok {
			continue
		}
		if evaluateNumeric(val, entry.Operator, entry.Value) {
			anyMet = true
			break
		}
	}

	if !anyMet {
		return false, "NO_ENTRY_CRITERIA_MET"
	}

	return true, ""
}

func evaluateNumeric(actual float64, operator string, threshold float64) bool {
	switch operator {
	case ">=":
		return actual >= threshold
	case "<=":
		return actual <= threshold
	case ">":
		return actual > threshold
	case "<":
		return actual < threshold
	case "==":
		return actual == threshold
	default:
		return false
	}
}

// CheckSuccess evaluates if graduation criteria are met.
// Returns (graduated, unmetCriteria). For PRP: ALL criteria must be met. For VFRP: ANY criterion suffices.
func (r *ProtocolRegistry) CheckSuccess(protocolID string, numericFields map[string]float64) (bool, string) {
	tmpl, err := r.GetTemplate(protocolID)
	if err != nil {
		return false, "UNKNOWN_PROTOCOL"
	}
	if len(tmpl.SuccessCriteria) == 0 {
		return false, "NO_SUCCESS_CRITERIA"
	}

	// PRP requires ALL criteria met (strict graduation)
	if protocolID == "M3-PRP" {
		for _, c := range tmpl.SuccessCriteria {
			val, ok := numericFields[c.Field]
			if !ok || !evaluateNumeric(val, c.Operator, c.Value) {
				return false, c.Field
			}
		}
		return true, ""
	}

	// VFRP requires ANY criterion met (flexible graduation)
	for _, c := range tmpl.SuccessCriteria {
		val, ok := numericFields[c.Field]
		if ok && evaluateNumeric(val, c.Operator, c.Value) {
			return true, ""
		}
	}
	return false, "NO_SUCCESS_CRITERIA_MET"
}

func (r *ProtocolRegistry) registerPRP() {
	r.templates["M3-PRP"] = &ProtocolTemplate{
		ProtocolID:   "M3-PRP",
		ProtocolName: "Metabolic Lever 3: Protein Restoration Protocol",
		Version:      "1.0.0",
		Category:     "lifestyle",
		Subcategory:  "nutrition",
		Phases: []PhaseDefinition{
			{ID: "BASELINE", DurationDays: 1, AutoAdvance: false},
			{ID: "STABILIZATION", DurationDays: 14, ExtendableTo: 21},
			{ID: "RESTORATION", DurationDays: 28, ExtendableTo: 42},
			{ID: "OPTIMIZATION", DurationDays: 42, ExtendableTo: 56},
		},
		EntryCriteria: []Criterion{
			{Field: "protein_gap", Operator: ">=", Value: 20},
			{Field: "protein_intake_gkg", Operator: "<", Value: 0.8},
		},
		ExclusionCriteria: []ExclusionRule{
			{Field: "egfr", Operator: "<", Value: 30, RuleCode: "LS-01"},
			{Field: "eating_disorder_flag", Operator: "==", Value: 1, RuleCode: "LS-14"},
			{Field: "nephrotic_syndrome", Operator: "==", Value: 1, RuleCode: "NEPHRO-EXCL"},
		},
		ConcurrentWith: []string{"M3-VFRP", "PROTOCOL-A", "PROTOCOL-B", "PROTOCOL-C", "V-MCU"},
		SuccessCriteria: []Criterion{
			{Field: "protein_intake_gkg", Operator: ">=", Value: 0.9},
			{Field: "lifestyle_attribution_pct", Operator: ">=", Value: 15},
		},
		EscalationTrigger: "trajectory_RED_at_day_63",
	}
}

func (r *ProtocolRegistry) registerVFRP() {
	r.templates["M3-VFRP"] = &ProtocolTemplate{
		ProtocolID:   "M3-VFRP",
		ProtocolName: "Metabolic Lever 3: Visceral Fat Reduction Protocol",
		Version:      "1.0.0",
		Category:     "lifestyle",
		Subcategory:  "body_composition",
		Phases: []PhaseDefinition{
			{ID: "BASELINE", DurationDays: 1, AutoAdvance: false},
			{ID: "METABOLIC_STABILIZATION", DurationDays: 14, ExtendableTo: 21},
			{ID: "FAT_MOBILIZATION", DurationDays: 28, ExtendableTo: 42},
			{ID: "SUSTAINED_REDUCTION", DurationDays: 42, ExtendableTo: 56},
		},
		EntryCriteria: []Criterion{
			{Field: "waist_cm", Operator: ">=", Value: 90},
			{Field: "waist_cm_female", Operator: ">=", Value: 80},
			{Field: "waist_trend_8wk_delta", Operator: ">", Value: 2},
			{Field: "triglycerides", Operator: ">", Value: 200},
		},
		ExclusionCriteria: []ExclusionRule{
			{Field: "bmi", Operator: "<", Value: 22, RuleCode: "LS-15"},
			{Field: "bmr_kcal", Operator: "<", Value: 1200, RuleCode: "LS-12"},
			{Field: "eating_disorder_flag", Operator: "==", Value: 1, RuleCode: "LS-14"},
			{Field: "pregnancy_status", Operator: "==", Value: 1, RuleCode: "LS-08"},
		},
		ConcurrentWith: []string{"M3-PRP", "PROTOCOL-A", "PROTOCOL-B", "PROTOCOL-C", "V-MCU"},
		SuccessCriteria: []Criterion{
			{Field: "waist_delta_cm", Operator: ">=", Value: 3},
			{Field: "tg_reduction_pct", Operator: ">=", Value: 15},
		},
		EscalationTrigger: "trajectory_RED_at_day_63",
	}
}
