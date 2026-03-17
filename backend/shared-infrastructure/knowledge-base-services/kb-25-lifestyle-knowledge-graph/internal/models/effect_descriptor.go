package models

type EffectDescriptor struct {
	EffectSize         float64           `json:"effect_size"`
	EffectUnit         string            `json:"effect_unit"`
	ConfidenceInterval [2]float64        `json:"confidence_interval"`
	DoseResponse       DoseResponseCurve `json:"dose_response"`
	OnsetDays          int               `json:"onset_days"`
	PeakEffectDays     int               `json:"peak_effect_days"`
	SteadyStateDays    int               `json:"steady_state_days"`
	EvidenceGrade      string            `json:"evidence_grade"` // A|B|C|D
	SourcePMIDs        []string          `json:"source_pmids"`
	EffectModifiers    []ModifierRef     `json:"effect_modifiers,omitempty"`
	Contraindications  []ContraRef       `json:"contraindications,omitempty"`
}

type DoseResponseCurve struct {
	Type       string    `json:"type"` // linear|logarithmic|sigmoid|threshold
	Parameters []float64 `json:"parameters"`
	MinDose    float64   `json:"min_dose"`
	MaxDose    float64   `json:"max_dose"`
	DoseUnit   string    `json:"dose_unit"`
}

type ModifierRef struct {
	ContextCode string  `json:"context_code"`
	Multiplier  float64 `json:"multiplier"`
	Condition   string  `json:"condition"`
}

type ContraRef struct {
	RuleCode    string `json:"rule_code"`
	Condition   string `json:"condition"`
	Severity    string `json:"severity"`     // HARD_STOP|WARNING
	Description string `json:"description"`
}
