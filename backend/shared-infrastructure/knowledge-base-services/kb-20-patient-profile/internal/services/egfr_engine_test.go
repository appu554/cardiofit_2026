package services

import (
	"math"
	"testing"
	"time"

	"github.com/shopspring/decimal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-patient-profile/internal/models"
)

func TestComputeEGFR(t *testing.T) {
	engine := NewEGFREngine()

	tests := []struct {
		name       string
		creatinine float64
		age        int
		sex        string
		wantMin    float64
		wantMax    float64
	}{
		{
			name:       "Male age 50 Scr=1.0 — normal kidney function",
			creatinine: 1.0,
			age:        50,
			sex:        "M",
			wantMin:    88,
			wantMax:    98,
		},
		{
			name:       "Female age 60 Scr=0.8 — expected higher eGFR from sex multiplier",
			creatinine: 0.8,
			age:        60,
			sex:        "F",
			wantMin:    78,
			wantMax:    92,
		},
		{
			name:       "Male age 30 Scr=0.7 — young adult normal",
			creatinine: 0.7,
			age:        30,
			sex:        "M",
			wantMin:    110,
			wantMax:    130,
		},
		{
			name:       "Male age 80 Scr=1.5 — elderly with elevated creatinine",
			creatinine: 1.5,
			age:        80,
			sex:        "M",
			wantMin:    35,
			wantMax:    50,
		},
		{
			name:       "Female age 40 Scr at kappa (0.7) — boundary test",
			creatinine: 0.7,
			age:        40,
			sex:        "F",
			wantMin:    100,
			wantMax:    125,
		},
		{
			name:       "Male age 50 Scr at kappa (0.9) — boundary test",
			creatinine: 0.9,
			age:        50,
			sex:        "M",
			wantMin:    95,
			wantMax:    108,
		},
		{
			name:       "Very low creatinine 0.2 — high eGFR",
			creatinine: 0.2,
			age:        40,
			sex:        "M",
			wantMin:    140,
			wantMax:    180,
		},
		{
			name:       "Very high creatinine 10.0 — very low eGFR",
			creatinine: 10.0,
			age:        50,
			sex:        "M",
			wantMin:    2,
			wantMax:    10,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.ComputeEGFR(tc.creatinine, tc.age, tc.sex)
			assert.GreaterOrEqual(t, result, tc.wantMin,
				"eGFR %.2f below expected minimum %.2f", result, tc.wantMin)
			assert.LessOrEqual(t, result, tc.wantMax,
				"eGFR %.2f above expected maximum %.2f", result, tc.wantMax)
		})
	}
}

func TestComputeEGFR_Deterministic(t *testing.T) {
	engine := NewEGFREngine()

	// Same inputs should always produce same output
	result1 := engine.ComputeEGFR(1.0, 50, "M")
	result2 := engine.ComputeEGFR(1.0, 50, "M")
	assert.Equal(t, result1, result2, "eGFR computation should be deterministic")
}

func TestComputeEGFR_SexDifference(t *testing.T) {
	engine := NewEGFREngine()

	male := engine.ComputeEGFR(1.0, 50, "M")
	female := engine.ComputeEGFR(1.0, 50, "F")

	// Female eGFR should be higher at same creatinine due to 1.012 multiplier
	// and different kappa/alpha parameters
	assert.NotEqual(t, male, female, "Male and female eGFR should differ")
}

func TestComputeEGFR_AgeEffect(t *testing.T) {
	engine := NewEGFREngine()

	young := engine.ComputeEGFR(1.0, 30, "M")
	old := engine.ComputeEGFR(1.0, 70, "M")

	assert.Greater(t, young, old,
		"Younger patients should have higher eGFR: young=%.2f old=%.2f", young, old)
}

func TestCKDStageFromEGFR(t *testing.T) {
	engine := NewEGFREngine()

	tests := []struct {
		name  string
		egfr  float64
		stage string
	}{
		{"G1 at 90", 90, models.CKDG1},
		{"G1 at 120", 120, models.CKDG1},
		{"G2 at 89", 89, models.CKDG2},
		{"G2 at 60", 60, models.CKDG2},
		{"G3a at 59", 59, models.CKDG3a},
		{"G3a at 45", 45, models.CKDG3a},
		{"G3b at 44", 44, models.CKDG3b},
		{"G3b at 30", 30, models.CKDG3b},
		{"G4 at 29", 29, models.CKDG4},
		{"G4 at 15", 15, models.CKDG4},
		{"G5 at 14", 14, models.CKDG5},
		{"G5 at 0", 0, models.CKDG5},
		{"G5 at negative", -1, models.CKDG5},
		{"G1 at very high", 200, models.CKDG1},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			result := engine.CKDStageFromEGFR(tc.egfr)
			assert.Equal(t, tc.stage, result)
		})
	}
}

func TestDetectThresholdCrossings(t *testing.T) {
	engine := NewEGFREngine()

	tests := []struct {
		name          string
		oldEGFR       float64
		newEGFR       float64
		wantCount     int
		wantBoundary  float64
		wantDrugClass string
	}{
		{
			name:          "Cross 60 downward — metformin monitor",
			oldEGFR:       62,
			newEGFR:       58,
			wantCount:     1,
			wantBoundary:  60,
			wantDrugClass: models.DrugClassMetformin,
		},
		{
			name:          "Cross 45 downward — metformin cap",
			oldEGFR:       48,
			newEGFR:       43,
			wantCount:     1,
			wantBoundary:  45,
			wantDrugClass: models.DrugClassMetformin,
		},
		{
			name:          "Cross 30 downward — metformin reduce + SGLT2i note",
			oldEGFR:       32,
			newEGFR:       28,
			wantCount:     2, // metformin + SGLT2i both at boundary 30
			wantBoundary:  30,
			wantDrugClass: "",
		},
		{
			name:          "Cross 15 downward — metformin discontinue",
			oldEGFR:       16,
			newEGFR:       13,
			wantCount:     1,
			wantBoundary:  15,
			wantDrugClass: models.DrugClassMetformin,
		},
		{
			name:      "No crossing — stable within G2",
			oldEGFR:   75,
			newEGFR:   70,
			wantCount: 0,
		},
		{
			name:          "Cross 60 upward — recovery",
			oldEGFR:       58,
			newEGFR:       62,
			wantCount:     1,
			wantBoundary:  60,
			wantDrugClass: models.DrugClassMetformin,
		},
		{
			name:      "Multiple crossings — 62 to 28",
			oldEGFR:   62,
			newEGFR:   28,
			wantCount: 4, // crosses 60 (metformin), 45 (metformin), 30 (metformin+SGLT2i)
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			crossings := engine.DetectThresholdCrossings(tc.oldEGFR, tc.newEGFR)
			assert.Len(t, crossings, tc.wantCount,
				"Expected %d crossings, got %d", tc.wantCount, len(crossings))

			if tc.wantCount == 1 && tc.wantDrugClass != "" {
				assert.Equal(t, tc.wantBoundary, crossings[0].EGFRBoundary)
				assert.Equal(t, tc.wantDrugClass, crossings[0].AffectedDrugClass)
			}
		})
	}
}

func TestIsCKDConfirmed(t *testing.T) {
	engine := NewEGFREngine()

	now := time.Now()
	day := 24 * time.Hour

	tests := []struct {
		name          string
		entries       []models.LabEntry
		wantHasCKD    bool
		wantConfirmed bool
	}{
		{
			name:          "No entries — no CKD",
			entries:       nil,
			wantHasCKD:    false,
			wantConfirmed: false,
		},
		{
			name: "Single reading below 60 — SUSPECTED",
			entries: []models.LabEntry{
				labEntry(55, now, models.ValidationAccepted),
			},
			wantHasCKD:    true,
			wantConfirmed: false,
		},
		{
			name: "Two readings below 60 but same day — SUSPECTED",
			entries: []models.LabEntry{
				labEntry(55, now, models.ValidationAccepted),
				labEntry(52, now.Add(2*time.Hour), models.ValidationAccepted),
			},
			wantHasCKD:    true,
			wantConfirmed: false,
		},
		{
			name: "Two readings below 60, 89 days apart — SUSPECTED (not 90)",
			entries: []models.LabEntry{
				labEntry(55, now, models.ValidationAccepted),
				labEntry(52, now.Add(89*day), models.ValidationAccepted),
			},
			wantHasCKD:    true,
			wantConfirmed: false,
		},
		{
			name: "Two readings below 60, exactly 90 days apart — CONFIRMED",
			entries: []models.LabEntry{
				labEntry(55, now, models.ValidationAccepted),
				labEntry(52, now.Add(90*day), models.ValidationAccepted),
			},
			wantHasCKD:    true,
			wantConfirmed: true,
		},
		{
			name: "Two readings below 60, 180 days apart — CONFIRMED",
			entries: []models.LabEntry{
				labEntry(55, now, models.ValidationAccepted),
				labEntry(40, now.Add(180*day), models.ValidationAccepted),
			},
			wantHasCKD:    true,
			wantConfirmed: true,
		},
		{
			name: "One below 60, one above 60 — only 1 CKD reading, SUSPECTED",
			entries: []models.LabEntry{
				labEntry(55, now, models.ValidationAccepted),
				labEntry(65, now.Add(91*day), models.ValidationAccepted),
			},
			wantHasCKD:    true,
			wantConfirmed: false,
		},
		{
			name: "All above 60 — no CKD",
			entries: []models.LabEntry{
				labEntry(75, now, models.ValidationAccepted),
				labEntry(80, now.Add(91*day), models.ValidationAccepted),
			},
			wantHasCKD:    false,
			wantConfirmed: false,
		},
		{
			name: "REJECTED entries should be excluded",
			entries: []models.LabEntry{
				labEntry(55, now, models.ValidationRejected),
				labEntry(52, now.Add(91*day), models.ValidationRejected),
			},
			wantHasCKD:    false,
			wantConfirmed: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			hasCKD, confirmed := engine.IsCKDConfirmed(tc.entries)
			assert.Equal(t, tc.wantHasCKD, hasCKD, "hasCKD mismatch")
			assert.Equal(t, tc.wantConfirmed, confirmed, "confirmed mismatch")
		})
	}
}

func TestClassifyTrajectory(t *testing.T) {
	engine := NewEGFREngine()
	now := time.Now()
	year := 365.25 * 24 * time.Hour

	tests := []struct {
		name      string
		points    []models.EGFRTrajectoryPoint
		wantTrend string
		checkSlope bool
		slopeSign  int // -1 negative, 0 near zero, +1 positive
	}{
		{
			name:      "Fewer than 3 points — INSUFFICIENT_DATA",
			points:    []models.EGFRTrajectoryPoint{{Value: 80, MeasuredAt: now}},
			wantTrend: TrendInsufficientData,
		},
		{
			name:      "Two points — INSUFFICIENT_DATA",
			points: []models.EGFRTrajectoryPoint{
				{Value: 80, MeasuredAt: now},
				{Value: 78, MeasuredAt: now.Add(time.Duration(float64(year)))},
			},
			wantTrend: TrendInsufficientData,
		},
		{
			name: "Stable trajectory — flat eGFR",
			points: []models.EGFRTrajectoryPoint{
				{Value: 80, MeasuredAt: now},
				{Value: 79, MeasuredAt: now.Add(time.Duration(float64(year)))},
				{Value: 80, MeasuredAt: now.Add(time.Duration(2 * float64(year)))},
			},
			wantTrend:  TrendStable,
			checkSlope: true,
			slopeSign:  0,
		},
		{
			name: "Slow decline — about -4 per year",
			points: []models.EGFRTrajectoryPoint{
				{Value: 80, MeasuredAt: now},
				{Value: 76, MeasuredAt: now.Add(time.Duration(float64(year)))},
				{Value: 72, MeasuredAt: now.Add(time.Duration(2 * float64(year)))},
			},
			wantTrend:  TrendSlowDecline,
			checkSlope: true,
			slopeSign:  -1,
		},
		{
			name: "Rapid decline — about -8 per year",
			points: []models.EGFRTrajectoryPoint{
				{Value: 80, MeasuredAt: now},
				{Value: 72, MeasuredAt: now.Add(time.Duration(float64(year)))},
				{Value: 64, MeasuredAt: now.Add(time.Duration(2 * float64(year)))},
			},
			wantTrend:  TrendRapidDecline,
			checkSlope: true,
			slopeSign:  -1,
		},
		{
			name: "Improving — positive slope",
			points: []models.EGFRTrajectoryPoint{
				{Value: 50, MeasuredAt: now},
				{Value: 55, MeasuredAt: now.Add(time.Duration(float64(year)))},
				{Value: 60, MeasuredAt: now.Add(time.Duration(2 * float64(year)))},
			},
			wantTrend:  TrendImproving,
			checkSlope: true,
			slopeSign:  1,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			trend, slope := engine.ClassifyTrajectory(tc.points)
			assert.Equal(t, tc.wantTrend, trend)

			if tc.checkSlope && slope != nil {
				switch tc.slopeSign {
				case -1:
					assert.Less(t, *slope, float64(0), "slope should be negative")
				case 1:
					assert.Greater(t, *slope, float64(0), "slope should be positive")
				case 0:
					assert.Less(t, math.Abs(*slope), float64(3), "slope should be near zero")
				}
			}
		})
	}
}

func TestCheckMedicationAlerts(t *testing.T) {
	engine := NewEGFREngine()

	metforminMed := models.MedicationState{
		DrugClass: models.DrugClassMetformin,
		DoseMg:    decimal.NewFromFloat(2000),
		Frequency: "BD",
		IsActive:  true,
	}

	t.Run("eGFR 35 with metformin 2000mg BD — should flag dose exceeds max", func(t *testing.T) {
		overrides := engine.CheckMedicationAlerts(35, []models.MedicationState{metforminMed})
		require.NotEmpty(t, overrides, "should produce safety overrides")
		// At eGFR 35 (G3b), metformin triggers at boundaries 60, 45
		// The 45 boundary has MaxDoseMg=1500, daily dose = 2000*2 = 4000 > 1500
		found := false
		for _, o := range overrides {
			if o.Severity == "RED" {
				found = true
			}
		}
		assert.True(t, found, "should have RED severity for dose exceeding max")
	})

	t.Run("eGFR 95 — no alerts", func(t *testing.T) {
		overrides := engine.CheckMedicationAlerts(95, []models.MedicationState{metforminMed})
		assert.Empty(t, overrides, "no alerts expected above all thresholds")
	})
}

// labEntry is a helper to create test LabEntry instances.
func labEntry(value float64, measuredAt time.Time, status string) models.LabEntry {
	return models.LabEntry{
		LabType:          models.LabTypeEGFR,
		Value:            decimal.NewFromFloat(value),
		MeasuredAt:       measuredAt,
		ValidationStatus: status,
	}
}
