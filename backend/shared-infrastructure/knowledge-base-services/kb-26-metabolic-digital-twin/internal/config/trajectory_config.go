package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"

	"kb-26-metabolic-digital-twin/internal/models"
)

// TrajectoryThresholds holds all tunable thresholds for the trajectory engine.
type TrajectoryThresholds struct {
	Trend              TrendThresholds              `yaml:"trend_thresholds"`
	Divergence         DivergenceThresholds         `yaml:"divergence"`
	LeadingIndicator   LeadingIndicatorThresholds   `yaml:"leading_indicator"`
	Concordant         ConcordantThresholds         `yaml:"concordant"`
	Driver             DriverThresholds             `yaml:"driver"`
	RSquared           R2Thresholds                 `yaml:"r_squared"`
	CategoryBoundaries CategoryBoundaries           `yaml:"category_boundaries"`
}

type TrendThresholds struct {
	RapidImproving float64 `yaml:"rapid_improving"`
	Improving      float64 `yaml:"improving"`
	Declining      float64 `yaml:"declining"`
	RapidDeclining float64 `yaml:"rapid_declining"`
}

type DivergenceThresholds struct {
	MinDivergenceRate float64 `yaml:"min_divergence_rate"`
	MinImprovingSlope float64 `yaml:"min_improving_slope"`
	MinDecliningSlope float64 `yaml:"min_declining_slope"`
}

type LeadingIndicatorThresholds struct {
	MinDataPoints             int     `yaml:"min_data_points"`
	MinBehavioralDeclineSlope float64 `yaml:"min_behavioral_decline_slope"`
}

type ConcordantThresholds struct {
	MinDomainsDeclining int     `yaml:"min_domains_declining"`
	MinSlopePerDomain   float64 `yaml:"min_slope_per_domain"`
}

type DriverThresholds struct {
	MinContributionPct float64                       `yaml:"min_contribution_pct"`
	WeightMap          map[models.MHRIDomain]float64 `yaml:"weight_map"`
}

type R2Thresholds struct {
	High     float64 `yaml:"high"`
	Moderate float64 `yaml:"moderate"`
}

type CategoryBoundaries struct {
	Optimal  float64 `yaml:"optimal"`
	Mild     float64 `yaml:"mild"`
	Moderate float64 `yaml:"moderate"`
}

// DefaultTrajectoryThresholds returns the canonical defaults matching the
// values that were previously hardcoded in mri_domain_trajectory.go.
func DefaultTrajectoryThresholds() TrajectoryThresholds {
	return TrajectoryThresholds{
		Trend: TrendThresholds{
			RapidImproving: 1.0,
			Improving:      0.3,
			Declining:      -0.3,
			RapidDeclining: -1.0,
		},
		Divergence: DivergenceThresholds{
			MinDivergenceRate: 0.5,
			MinImprovingSlope: 0.3,
			MinDecliningSlope: -0.3,
		},
		LeadingIndicator: LeadingIndicatorThresholds{
			MinDataPoints:             5,
			MinBehavioralDeclineSlope: -0.5,
		},
		Concordant: ConcordantThresholds{
			MinDomainsDeclining: 2,
			MinSlopePerDomain:   -0.3,
		},
		Driver: DriverThresholds{
			MinContributionPct: 40.0,
			WeightMap: map[models.MHRIDomain]float64{
				models.DomainGlucose:    0.35,
				models.DomainCardio:     0.25,
				models.DomainBodyComp:   0.25,
				models.DomainBehavioral: 0.15,
			},
		},
		RSquared: R2Thresholds{
			High:     0.5,
			Moderate: 0.25,
		},
		CategoryBoundaries: CategoryBoundaries{
			Optimal:  70.0,
			Mild:     55.0,
			Moderate: 40.0,
		},
	}
}

// LoadTrajectoryThresholds parses a YAML file and returns thresholds.
// Returns DefaultTrajectoryThresholds and a warning error if the file
// is missing — startup should not fail on missing config.
func LoadTrajectoryThresholds(path string) (TrajectoryThresholds, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultTrajectoryThresholds(), nil
		}
		return TrajectoryThresholds{}, fmt.Errorf("read trajectory thresholds: %w", err)
	}

	var t TrajectoryThresholds
	if err := yaml.Unmarshal(data, &t); err != nil {
		return TrajectoryThresholds{}, fmt.Errorf("parse trajectory thresholds: %w", err)
	}

	return t, nil
}
