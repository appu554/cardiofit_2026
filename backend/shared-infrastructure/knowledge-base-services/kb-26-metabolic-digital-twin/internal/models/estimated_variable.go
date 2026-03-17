package models

// EstimatedVariable represents a Tier 3 estimated metabolic parameter
// stored as JSONB within TwinState.
type EstimatedVariable struct {
	Value          float64 `json:"value"`
	Classification string  `json:"classification"`
	Confidence     float64 `json:"confidence"`
	Method         string  `json:"method"`
}
