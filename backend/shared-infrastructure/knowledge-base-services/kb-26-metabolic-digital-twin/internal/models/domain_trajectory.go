package models

import (
	"time"

	"github.com/google/uuid"
)

// MHRIDomain identifies each of the four MHRI domains.
type MHRIDomain string

const (
	DomainGlucose    MHRIDomain = "GLUCOSE"
	DomainCardio     MHRIDomain = "CARDIO"
	DomainBodyComp   MHRIDomain = "BODY_COMP"
	DomainBehavioral MHRIDomain = "BEHAVIORAL"
)

// AllMHRIDomains lists all four domains for iteration.
var AllMHRIDomains = []MHRIDomain{DomainGlucose, DomainCardio, DomainBodyComp, DomainBehavioral}

// Trend classifications (used in DomainSlope.Trend and DecomposedTrajectory.CompositeTrend).
const (
	TrendRapidImproving  = "RAPID_IMPROVING"
	TrendImproving       = "IMPROVING"
	TrendStable          = "STABLE"
	TrendDeclining       = "DECLINING"
	TrendRapidDeclining  = "RAPID_DECLINING"
	TrendInsufficient    = "INSUFFICIENT_DATA"
)

// Confidence levels for OLS R² goodness-of-fit.
const (
	ConfidenceHigh     = "HIGH"
	ConfidenceModerate = "MODERATE"
	ConfidenceLow      = "LOW"
)

// Direction values for DomainCategoryCrossing.
const (
	DirectionWorsened = "WORSENED"
	DirectionImproved = "IMPROVED"
)

// DomainTrajectoryPoint stores a single snapshot of all domain scores at a point in time.
type DomainTrajectoryPoint struct {
	Timestamp       time.Time `json:"timestamp"`
	CompositeScore  float64   `json:"composite_score"`
	GlucoseScore    float64   `json:"glucose_score"`
	CardioScore     float64   `json:"cardio_score"`
	BodyCompScore   float64   `json:"body_comp_score"`
	BehavioralScore float64   `json:"behavioral_score"`
}

// DomainSlope captures the OLS regression result for a single domain.
type DomainSlope struct {
	Domain      MHRIDomain `json:"domain"`
	SlopePerDay float64    `json:"slope_per_day"`
	Trend       string     `json:"trend"`
	StartScore  float64    `json:"start_score"`
	EndScore    float64    `json:"end_score"`
	DeltaScore  float64    `json:"delta_score"`
	R2          float64    `json:"r_squared"`
	Confidence  string     `json:"confidence"`
}

// DivergencePattern describes when two domains move in opposite directions.
type DivergencePattern struct {
	ImprovingDomain   MHRIDomain `json:"improving_domain"`
	DecliningDomain   MHRIDomain `json:"declining_domain"`
	ImprovingSlope    float64    `json:"improving_slope"`
	DecliningSlope    float64    `json:"declining_slope"`
	DivergenceRate    float64    `json:"divergence_rate"`
	ClinicalConcern   string     `json:"clinical_concern"`
	PossibleMechanism string     `json:"possible_mechanism"`
}

// LeadingIndicator describes when behavioral domain decline precedes
// clinical domain decline.
//
// Phase 10 V4-5 completion: LeadDays and CausalChain fields added
// to support temporal lead-lag analysis (Scenario 4 from the spec:
// "body composition deteriorated first, then glucose, then cardio —
// pointing to weight gain as the root cause").
type LeadingIndicator struct {
	LeadingDomain  MHRIDomain   `json:"leading_domain"`
	LaggingDomains []MHRIDomain `json:"lagging_domains"`
	LeadDays       int          `json:"lead_days"`        // estimated days the leading domain leads by
	CausalChain    []MHRIDomain `json:"causal_chain,omitempty"` // ordered: first decliner → second → third
	Confidence     string       `json:"confidence"`
	Interpretation string       `json:"interpretation"`
}

// DomainCategoryCrossing detects when a domain crosses an MHRI category boundary.
type DomainCategoryCrossing struct {
	Domain       MHRIDomain `json:"domain"`
	PrevCategory string     `json:"prev_category"`
	CurrCategory string     `json:"curr_category"`
	Direction    string     `json:"direction"`
	CrossingDate time.Time  `json:"crossing_date"`
}

// DecomposedTrajectory is the full output of the domain decomposition engine.
type DecomposedTrajectory struct {
	PatientID               string                     `json:"patient_id"`
	WindowDays              int                        `json:"window_days"`
	DataPoints              int                        `json:"data_points"`
	ComputedAt              time.Time                  `json:"computed_at"`
	CompositeSlope          float64                    `json:"composite_slope_per_day"`
	CompositeTrend          string                     `json:"composite_trend"`
	CompositeStartScore     float64                    `json:"composite_start_score"`
	CompositeEndScore       float64                    `json:"composite_end_score"`
	DomainSlopes            map[MHRIDomain]DomainSlope `json:"domain_slopes"`
	DominantDriver          *MHRIDomain                `json:"dominant_driver,omitempty"`
	DriverContribution      float64                    `json:"driver_contribution"`
	Divergences             []DivergencePattern        `json:"divergences,omitempty"`
	LeadingIndicators       []LeadingIndicator         `json:"leading_indicators,omitempty"`
	DomainCrossings         []DomainCategoryCrossing   `json:"domain_crossings,omitempty"`
	HasDiscordantTrend      bool                       `json:"has_discordant_trend"`
	ConcordantDeterioration bool                       `json:"concordant_deterioration"`
	DomainsDeteriorating    int                        `json:"domains_deteriorating"`

	// Phase 10 V4-5 completion: seasonal suppression (India-specific).
	// When a seasonal window is active and the declining domain
	// matches the window's affected_domains list, this field carries
	// the seasonal context so downstream card generators can
	// downgrade urgency rather than alerting on expected patterns.
	SeasonalSuppression *SeasonalSuppressionContext `json:"seasonal_suppression,omitempty"`
}

// SeasonalSuppressionContext describes an active seasonal window
// that may explain a domain decline. Phase 10 V4-5 completion.
type SeasonalSuppressionContext struct {
	WindowName      string       `json:"window_name"`       // e.g. "diwali", "ramadan", "summer_heat"
	AffectedDomains []MHRIDomain `json:"affected_domains"`
	Mode            string       `json:"mode"`              // DOWNGRADE_URGENCY | SUPPRESS
	Rationale       string       `json:"rationale"`
}

// DomainTrajectoryHistory stores decomposed snapshots for trend-over-time analysis.
type DomainTrajectoryHistory struct {
	ID              string    `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID       uuid.UUID `gorm:"type:uuid;index:idx_dth_patient,priority:1;uniqueIndex:uq_dth_patient_date,priority:1;not null" json:"patient_id"`
	SnapshotDate    time.Time `gorm:"type:date;index:idx_dth_patient,priority:2,sort:desc;uniqueIndex:uq_dth_patient_date,priority:2;not null" json:"snapshot_date"`
	WindowDays      int       `gorm:"not null" json:"window_days"`
	CompositeSlope  float64   `gorm:"type:decimal(6,3)" json:"composite_slope"`
	GlucoseSlope    float64   `gorm:"type:decimal(6,3)" json:"glucose_slope"`
	CardioSlope     float64   `gorm:"type:decimal(6,3)" json:"cardio_slope"`
	BodyCompSlope   float64   `gorm:"type:decimal(6,3)" json:"body_comp_slope"`
	BehavioralSlope float64   `gorm:"type:decimal(6,3)" json:"behavioral_slope"`
	HasDiscordance  bool      `gorm:"default:false" json:"has_discordance"`
	DominantDriver  string    `gorm:"size:20" json:"dominant_driver,omitempty"`
	CreatedAt       time.Time `gorm:"not null;default:now()" json:"created_at"`
}

func (DomainTrajectoryHistory) TableName() string { return "domain_trajectory_history" }
