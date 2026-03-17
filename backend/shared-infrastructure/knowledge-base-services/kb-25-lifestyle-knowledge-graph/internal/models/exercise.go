package models

type Exercise struct {
	Code              string   `json:"code"`
	Name              string   `json:"name"`
	Category          string   `json:"category"`
	METValue          float64  `json:"met_value"`
	SafetyTier        string   `json:"safety_tier"`
	MinDurationMin    int      `json:"min_duration_min"`
	MaxDurationMin    int      `json:"max_duration_min"`
	FreqPerWeek       int      `json:"freq_per_week"`
	Equipment         []string `json:"equipment,omitempty"`
	Contraindications []string `json:"contraindications,omitempty"`
}
