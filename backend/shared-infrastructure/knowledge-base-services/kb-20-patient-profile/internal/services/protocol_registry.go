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
	DrugSequence      []DrugStep         `json:"drug_sequence,omitempty"`
	TargetMetric      string             `json:"target_metric,omitempty"`
	Targets           *TargetRange       `json:"targets,omitempty"`
	IsLifelong        bool               `json:"is_lifelong"`
	SuccessMode       SuccessMode        `json:"success_mode,omitempty"`
	GuidelineRef      string             `json:"guideline_ref,omitempty"`
}

// PhaseDefinition describes one phase in a protocol.
type PhaseDefinition struct {
	ID              string `json:"id"`
	Name            string `json:"name,omitempty"`
	DurationDays    int    `json:"duration_days"`
	ExtendableTo    int    `json:"extendable_to"`
	AutoAdvance     bool   `json:"auto_advance"`
	ActiveDrugSteps []int  `json:"active_drug_steps,omitempty"` // indices into DrugSequence
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

// NewProtocolRegistry creates a registry pre-loaded with all protocol templates.
func NewProtocolRegistry() *ProtocolRegistry {
	r := &ProtocolRegistry{templates: make(map[string]*ProtocolTemplate)}
	r.registerPRP()
	r.registerVFRP()
	r.registerGLYC1()
	r.registerHTN1()
	r.registerRENAL1()
	r.registerLIPID1()
	r.registerDEPRESC1()
	r.registerMAINTAIN()
	r.registerRECORRECTION()
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
// Behaviour is driven by the template's SuccessMode:
//   - ALL:       every criterion must be met (M3-PRP)
//   - ANY:       any single criterion suffices (M3-VFRP)
//   - NEVER:     lifelong protocol, never graduates (GLYC-1, HTN-1)
//   - CARD_ONLY: informational only, no titration (LIPID-1)
func (r *ProtocolRegistry) CheckSuccess(protocolID string, numericFields map[string]float64) (bool, string) {
	tmpl, err := r.GetTemplate(protocolID)
	if err != nil {
		return false, "UNKNOWN_PROTOCOL"
	}
	if tmpl.SuccessMode == SuccessModeNever {
		return false, "LIFELONG_PROTOCOL"
	}
	if tmpl.SuccessMode == SuccessModeCardOnly {
		return false, "CARD_ONLY_PROTOCOL"
	}
	if len(tmpl.SuccessCriteria) == 0 {
		return false, "NO_SUCCESS_CRITERIA"
	}

	switch tmpl.SuccessMode {
	case SuccessModeAll:
		for _, c := range tmpl.SuccessCriteria {
			val, ok := numericFields[c.Field]
			if !ok || !evaluateNumeric(val, c.Operator, c.Value) {
				return false, c.Field
			}
		}
		return true, ""
	case SuccessModeAny:
		for _, c := range tmpl.SuccessCriteria {
			val, ok := numericFields[c.Field]
			if ok && evaluateNumeric(val, c.Operator, c.Value) {
				return true, ""
			}
		}
		return false, "NO_SUCCESS_CRITERIA_MET"
	default:
		// Backwards compat for templates without explicit SuccessMode
		if protocolID == "M3-PRP" {
			for _, c := range tmpl.SuccessCriteria {
				val, ok := numericFields[c.Field]
				if !ok || !evaluateNumeric(val, c.Operator, c.Value) {
					return false, c.Field
				}
			}
			return true, ""
		}
		for _, c := range tmpl.SuccessCriteria {
			val, ok := numericFields[c.Field]
			if ok && evaluateNumeric(val, c.Operator, c.Value) {
				return true, ""
			}
		}
		return false, "NO_SUCCESS_CRITERIA_MET"
	}
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
		ConcurrentWith: []string{"M3-VFRP", "GLYC-1", "HTN-1", "RENAL-1", "LIPID-1", "DEPRESC-1", "V-MCU"},
		SuccessCriteria: []Criterion{
			{Field: "protein_intake_gkg", Operator: ">=", Value: 0.9},
			{Field: "lifestyle_attribution_pct", Operator: ">=", Value: 15},
		},
		EscalationTrigger: "trajectory_RED_at_day_63",
		SuccessMode:       SuccessModeAll,
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
		ConcurrentWith: []string{"M3-PRP", "GLYC-1", "HTN-1", "RENAL-1", "LIPID-1", "DEPRESC-1", "V-MCU"},
		SuccessCriteria: []Criterion{
			{Field: "waist_delta_cm", Operator: ">=", Value: 3},
			{Field: "tg_reduction_pct", Operator: ">=", Value: 15},
		},
		EscalationTrigger: "trajectory_RED_at_day_63",
		SuccessMode:       SuccessModeAny,
	}
}

func (r *ProtocolRegistry) registerMAINTAIN() {
	r.templates["M3-MAINTAIN"] = &ProtocolTemplate{
		ProtocolID:   "M3-MAINTAIN",
		ProtocolName: "Metabolic Maintenance Lifecycle",
		Version:      "1.0.0",
		Category:     "lifecycle",
		Subcategory:  "engagement",
		Phases: []PhaseDefinition{
			{ID: "CONSOLIDATION", DurationDays: 90, ExtendableTo: 120, AutoAdvance: false},
			{ID: "INDEPENDENCE", DurationDays: 90, ExtendableTo: 120, AutoAdvance: false},
			{ID: "STABILITY", DurationDays: 90, ExtendableTo: 120, AutoAdvance: false},
			{ID: "PARTNERSHIP", DurationDays: -1, ExtendableTo: -1, AutoAdvance: false},
		},
		EntryCriteria: []Criterion{
			{Field: "mri_score", Operator: "<", Value: 50},
		},
		ConcurrentWith:  []string{"GLYC-1", "HTN-1", "RENAL-1", "LIPID-1", "DEPRESC-1"},
		SuccessCriteria: nil,
		IsLifelong:      true,
		SuccessMode:     SuccessModeNever,
		GuidelineRef:    "Patient_Engagement_Loop_Specification_v1.0",
	}
}

func (r *ProtocolRegistry) registerRECORRECTION() {
	r.templates["M3-RECORRECTION"] = &ProtocolTemplate{
		ProtocolID:   "M3-RECORRECTION",
		ProtocolName: "Metabolic Re-Correction (Abbreviated Cycle)",
		Version:      "1.0.0",
		Category:     "lifecycle",
		Subcategory:  "re-engagement",
		Phases: []PhaseDefinition{
			{ID: "ASSESSMENT", DurationDays: 3, AutoAdvance: true},
			{ID: "CORRECTION", DurationDays: 45, ExtendableTo: 60, AutoAdvance: false},
		},
		EntryCriteria: []Criterion{
			{Field: "relapse_detected", Operator: "==", Value: 1},
		},
		ConcurrentWith: []string{"GLYC-1", "HTN-1", "RENAL-1", "DEPRESC-1"},
		SuccessCriteria: []Criterion{
			{Field: "mri_score", Operator: "<", Value: 50},
		},
		IsLifelong:   false,
		SuccessMode:  SuccessModeAll,
		GuidelineRef: "Patient_Engagement_Loop_Specification_v1.0_Section8",
	}
}

