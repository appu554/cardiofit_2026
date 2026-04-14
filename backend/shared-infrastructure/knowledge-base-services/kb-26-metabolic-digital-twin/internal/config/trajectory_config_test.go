package config

import (
	"os"
	"path/filepath"
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestLoadTrajectoryThresholds_DefaultsOnMissingFile(t *testing.T) {
	tmp := t.TempDir()
	missing := filepath.Join(tmp, "nope.yaml")

	got, err := LoadTrajectoryThresholds(missing)
	if err != nil {
		t.Fatalf("expected nil error on missing file, got %v", err)
	}

	defaults := DefaultTrajectoryThresholds()
	if got.Trend.RapidImproving != defaults.Trend.RapidImproving {
		t.Errorf("expected default RapidImproving %v, got %v", defaults.Trend.RapidImproving, got.Trend.RapidImproving)
	}
	if got.Driver.WeightMap[models.DomainGlucose] != 0.35 {
		t.Errorf("expected default glucose weight 0.35, got %v", got.Driver.WeightMap[models.DomainGlucose])
	}
}

func TestLoadTrajectoryThresholds_ParsesValidYAML(t *testing.T) {
	tmp := t.TempDir()
	path := filepath.Join(tmp, "thresholds.yaml")

	yaml := `
trend_thresholds:
  rapid_improving: 1.5
  improving: 0.4
  declining: -0.4
  rapid_declining: -1.2

divergence:
  min_divergence_rate: 0.6
  min_improving_slope: 0.4
  min_declining_slope: -0.4

leading_indicator:
  min_data_points: 6
  min_behavioral_decline_slope: -0.6

concordant:
  min_domains_declining: 3
  min_slope_per_domain: -0.4

driver:
  min_contribution_pct: 50.0
  weight_map:
    GLUCOSE: 0.40
    CARDIO: 0.30
    BODY_COMP: 0.20
    BEHAVIORAL: 0.10

r_squared:
  high: 0.6
  moderate: 0.3

category_boundaries:
  optimal: 75
  mild: 60
  moderate: 45
`
	if err := os.WriteFile(path, []byte(yaml), 0644); err != nil {
		t.Fatalf("write test yaml: %v", err)
	}

	got, err := LoadTrajectoryThresholds(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}

	if got.Trend.RapidImproving != 1.5 {
		t.Errorf("expected RapidImproving 1.5, got %v", got.Trend.RapidImproving)
	}
	if got.Driver.WeightMap[models.DomainGlucose] != 0.40 {
		t.Errorf("expected glucose weight 0.40, got %v", got.Driver.WeightMap[models.DomainGlucose])
	}
	if got.CategoryBoundaries.Optimal != 75 {
		t.Errorf("expected optimal boundary 75, got %v", got.CategoryBoundaries.Optimal)
	}
}
