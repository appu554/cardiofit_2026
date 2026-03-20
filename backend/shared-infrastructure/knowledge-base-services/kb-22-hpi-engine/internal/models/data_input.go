package models

import "time"

// RequiredInput defines a scalar input needed by a PM/MD node.
type RequiredInput struct {
	Field           string `yaml:"field" json:"field"`
	Source          string `yaml:"source" json:"source"` // KB-20 | KB-26 | DEVICE | TIER1_CHECKIN
	Unit            string `yaml:"unit" json:"unit"`
	MinObservations int    `yaml:"min_observations" json:"min_observations"`
	LookbackDays    int    `yaml:"lookback_days" json:"lookback_days"`
	Optional        bool   `yaml:"optional" json:"optional"`
	Description     string `yaml:"description" json:"description"`
}

// AggregatedInputDef defines a time-series input requiring pre-computed aggregation.
// Used by PM-04 through PM-09 which need statistical summaries (mean, stdev, CV)
// over time-series data. DataResolver fetches the series, computes the aggregate,
// and stores the scalar result in ResolvedData.Fields.
type AggregatedInputDef struct {
	Field        string `yaml:"field" json:"field"`
	Source       string `yaml:"source" json:"source"`
	LookbackDays int    `yaml:"lookback_days" json:"lookback_days"`
	Aggregation  string `yaml:"aggregation" json:"aggregation"` // MEAN | STDEV | COUNT | MAX | MIN | CV | RAW
	Optional     bool   `yaml:"optional" json:"optional"`
	Description  string `yaml:"description" json:"description"`
}

// ResolvedData contains all data fetched and computed for a node evaluation.
type ResolvedData struct {
	Fields          map[string]float64           `json:"fields"`
	TimeSeries      map[string][]TimeSeriesPoint `json:"time_series,omitempty"` // raw series for TrajectoryComputer
	FieldTimestamps map[string]time.Time         `json:"field_timestamps,omitempty"`
	Sufficiency     DataSufficiency              `json:"sufficiency"`
	MissingFields   []string                     `json:"missing_fields,omitempty"`
	Sources         map[string]string            `json:"sources,omitempty"` // field → source used
}

// DataSufficiency indicates the completeness of resolved data.
type DataSufficiency string

const (
	DataSufficient   DataSufficiency = "SUFFICIENT"
	DataPartial      DataSufficiency = "PARTIAL"
	DataInsufficient DataSufficiency = "INSUFFICIENT"
)

// TwinStateView is a flattened view of KB-26 twin state for engine consumption.
type TwinStateView struct {
	IS  EstimatedValue `json:"is"`
	HGO EstimatedValue `json:"hgo"`
	MM  EstimatedValue `json:"mm"`

	VF      float64 `json:"vf"`
	VFTrend string  `json:"vf_trend"`

	VR EstimatedValue `json:"vr"` // from KB-26 JSONB or Tier 2 derivation
	RR EstimatedValue `json:"rr"` // from KB-26 JSONB or Tier 2 derivation

	RenalSlope  float64   `json:"renal_slope"`
	EGFR        *float64  `json:"egfr"`
	GlycemicVar float64   `json:"glycemic_var"`
	DailySteps  *float64  `json:"daily_steps"`
	RestingHR   *float64  `json:"resting_hr"`
	LastUpdated time.Time `json:"last_updated"` // for staleness check
}

// EstimatedValue pairs a numeric estimate with its confidence.
type EstimatedValue struct {
	Value      float64 `json:"value"`
	Confidence float64 `json:"confidence"`
}

// TimeSeriesPoint is a single timestamped value for trajectory computation.
type TimeSeriesPoint struct {
	Timestamp time.Time `json:"timestamp"`
	Value     float64   `json:"value"`
}

// StalenessRule defines the maximum age of a data field before it is considered stale.
type StalenessRule struct {
	FieldPattern string        // field name or prefix for HasPrefix matching (e.g. "fbg", "sbp", "egfr")
	MaxAge       time.Duration // data older than this → DataPartial
}
