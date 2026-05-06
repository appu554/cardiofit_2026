package delta

import (
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/cardiofit/shared/v2_substrate/models"
)

func ptr(f float64) *float64 { return &f }

func TestComputeDelta_Cases(t *testing.T) {
	bl := &Baseline{
		BaselineValue: 120.0,
		StdDev:        10.0,
		SampleSize:    50,
		ComputedAt:    time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC),
	}

	cases := []struct {
		name      string
		obs       models.Observation
		baseline  *Baseline
		wantFlag  string
	}{
		{
			name: "within_baseline_value_equals_baseline",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(120.0), ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagWithinBaseline,
		},
		{
			name: "within_baseline_one_stddev_high",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(130.0), ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagWithinBaseline, // |dev|=1.0 → within (boundary inclusive)
		},
		{
			name: "elevated_1pt5_stddev",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(135.0), ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagElevated,
		},
		{
			name: "severely_elevated_3_stddev",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(150.0), ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagSeverelyElevated,
		},
		{
			name: "low_1pt5_stddev_below",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(105.0), ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagLow,
		},
		{
			name: "severely_low_3_stddev_below",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(90.0), ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagSeverelyLow,
		},
		{
			name: "no_baseline_when_baseline_nil",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(120.0), ObservedAt: time.Now()},
			baseline: nil,
			wantFlag: models.DeltaFlagNoBaseline,
		},
		{
			name: "no_baseline_when_value_nil",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindBehavioural, ValueText: "agitation", ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagNoBaseline,
		},
		{
			name: "no_baseline_when_kind_behavioural",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindBehavioural, Value: ptr(1.0), ObservedAt: time.Now()},
			baseline: bl,
			wantFlag: models.DeltaFlagNoBaseline,
		},
		{
			name: "no_baseline_when_stddev_zero_guards_div0",
			obs:  models.Observation{ResidentID: uuid.New(), Kind: models.ObservationKindVital, Value: ptr(150.0), ObservedAt: time.Now()},
			baseline: &Baseline{BaselineValue: 120.0, StdDev: 0.0, SampleSize: 1, ComputedAt: time.Now()},
			wantFlag: models.DeltaFlagNoBaseline,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			d := ComputeDelta(c.obs, c.baseline)
			if d.DirectionalFlag != c.wantFlag {
				t.Errorf("ComputeDelta flag: got %q want %q (case %s)", d.DirectionalFlag, c.wantFlag, c.name)
			}
			if c.wantFlag != models.DeltaFlagNoBaseline {
				if d.BaselineValue != c.baseline.BaselineValue {
					t.Errorf("BaselineValue: got %v want %v", d.BaselineValue, c.baseline.BaselineValue)
				}
			} else {
				if d.BaselineValue != 0 || d.DeviationStdDev != 0 {
					t.Errorf("no_baseline must zero numeric fields, got BL=%v dev=%v", d.BaselineValue, d.DeviationStdDev)
				}
			}
			if d.ComputedAt.IsZero() {
				t.Errorf("ComputedAt must be set, got zero")
			}
		})
	}
}
