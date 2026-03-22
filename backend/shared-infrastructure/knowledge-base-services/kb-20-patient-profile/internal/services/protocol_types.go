package services

// DrugStep defines one step in a protocol's drug sequencing algorithm.
// The TitrationEngine titrates within the current step until the
// escalation trigger fires, then the protocol advances to the next step.
type DrugStep struct {
	StepOrder           int      `json:"step_order"`
	DrugClass           string   `json:"drug_class"`
	DrugName            string   `json:"drug_name"`
	StartingDoseMg      float64  `json:"starting_dose_mg"`
	DoseIncrementMg     float64  `json:"dose_increment_mg"`
	MaxDoseMg           float64  `json:"max_dose_mg"`
	FrequencyPerDay     int      `json:"frequency_per_day"`
	EscalationTrigger   string   `json:"escalation_trigger"`
	DeescalationTrigger string   `json:"deescalation_trigger,omitempty"`
	ChannelBGuards      []string `json:"channel_b_guards"`
	ChannelCGuards      []string `json:"channel_c_guards"`
	Notes               string   `json:"notes,omitempty"`
	OwningProtocol      string   `json:"owning_protocol,omitempty"` // for shared drugs: which protocol owns dose decisions
}

// TargetRange defines the goal range for a protocol's target metric.
type TargetRange struct {
	Metric                string                `json:"metric"` // HbA1c, SBP, eGFR, LDL
	Unit                  string                `json:"unit"`
	DefaultLow            float64               `json:"default_low"`
	DefaultHigh           float64               `json:"default_high"`
	IndividualisedTargets []IndividualisedTarget `json:"individualised_targets,omitempty"`
}

// IndividualisedTarget overrides the default range for a specific patient archetype.
type IndividualisedTarget struct {
	Archetype string  `json:"archetype"` // ElderlyFrail, CKDProgressor, GoodResponder, VisceralObese
	Low       float64 `json:"low"`
	High      float64 `json:"high"`
	Rationale string  `json:"rationale"`
}

// TargetFor returns the individualised target for the given archetype,
// or falls back to the default range.
func (tr TargetRange) TargetFor(archetype string) IndividualisedTarget {
	for _, it := range tr.IndividualisedTargets {
		if it.Archetype == archetype {
			return it
		}
	}
	return IndividualisedTarget{
		Archetype: archetype,
		Low:       tr.DefaultLow,
		High:      tr.DefaultHigh,
		Rationale: "default target",
	}
}

// SuccessMode determines how graduation criteria are evaluated.
type SuccessMode string

const (
	SuccessModeAll      SuccessMode = "ALL"       // all criteria must be met (M3-PRP)
	SuccessModeAny      SuccessMode = "ANY"       // any criterion suffices (M3-VFRP)
	SuccessModeNever    SuccessMode = "NEVER"      // lifelong protocol, no graduation (GLYC-1, HTN-1)
	SuccessModeCardOnly SuccessMode = "CARD_ONLY"  // generates decision cards, no titration (LIPID-1)
)
