// Package tests provides unit tests for Phase 3b.6 conditional reference ranges
package tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/types"
)

// =============================================================================
// CONDITIONAL REFERENCE RANGE TESTS
// =============================================================================

func TestConditionalReferenceRange_Interpret(t *testing.T) {
	// Setup: Pregnancy T3 Hemoglobin range (10.5-14.0)
	lowNormal := 10.5
	highNormal := 14.0
	critLow := 7.0
	critHigh := 16.0
	panicLow := 5.0
	panicHigh := 18.0

	pregnantT3Range := &reference.ConditionalReferenceRange{
		ID:            uuid.New(),
		LowNormal:     &lowNormal,
		HighNormal:    &highNormal,
		CriticalLow:   &critLow,
		CriticalHigh:  &critHigh,
		PanicLow:      &panicLow,
		PanicHigh:     &panicHigh,
		Authority:     "ACOG",
		AuthorityRef:  "ACOG Practice Bulletin: Anemia in Pregnancy 2021",
		SpecificityScore: 5,
		RangeConditions: reference.RangeConditions{
			IsPregnant: boolPtr(true),
			Trimester:  intPtr(3),
		},
	}

	tests := []struct {
		name           string
		value          float64
		expectedFlag   reference.InterpretationFlag
		expectDeviation bool
	}{
		{
			name:         "Normal value in pregnancy T3",
			value:        12.0,
			expectedFlag: reference.FlagNormal,
		},
		{
			name:         "Low boundary (still normal)",
			value:        10.5,
			expectedFlag: reference.FlagNormal,
		},
		{
			name:           "Below normal (LOW)",
			value:          9.5,
			expectedFlag:   reference.FlagLow,
			expectDeviation: true,
		},
		{
			name:           "Critical low",
			value:          6.5,
			expectedFlag:   reference.FlagCriticalLow,
			expectDeviation: true,
		},
		{
			name:           "Panic low",
			value:          4.0,
			expectedFlag:   reference.FlagPanicLow,
			expectDeviation: true,
		},
		{
			name:           "Above normal (HIGH)",
			value:          14.5,
			expectedFlag:   reference.FlagHigh,
			expectDeviation: true,
		},
		{
			name:           "Critical high",
			value:          17.0,
			expectedFlag:   reference.FlagCriticalHigh,
			expectDeviation: true,
		},
		{
			name:           "Panic high",
			value:          19.0,
			expectedFlag:   reference.FlagPanicHigh,
			expectDeviation: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := pregnantT3Range.Interpret(tt.value, "g/dL")

			assert.Equal(t, tt.expectedFlag, result.Flag)
			assert.Equal(t, tt.value, result.Value)
			assert.Equal(t, "g/dL", result.Unit)
			assert.Equal(t, "ACOG", result.Authority)
			assert.Equal(t, 5, result.SpecificityScore)

			if tt.expectDeviation {
				assert.NotNil(t, result.DeviationPercent)
				assert.NotEmpty(t, result.DeviationDirection)
			}
		})
	}
}

func TestConditionalReferenceRange_ContextDescription(t *testing.T) {
	tests := []struct {
		name     string
		range_   reference.ConditionalReferenceRange
		expected string
	}{
		{
			name: "Pregnancy T3",
			range_: reference.ConditionalReferenceRange{
				RangeConditions: reference.RangeConditions{
					IsPregnant: boolPtr(true),
					Trimester:  intPtr(3),
				},
			},
			expected: "Pregnancy T3",
		},
		{
			name: "CKD Stage 4",
			range_: reference.ConditionalReferenceRange{
				RangeConditions: reference.RangeConditions{
					CKDStage: intPtr(4),
				},
			},
			expected: "CKD Stage 4",
		},
		{
			name: "CKD Stage 5 Dialysis",
			range_: reference.ConditionalReferenceRange{
				RangeConditions: reference.RangeConditions{
					CKDStage:     intPtr(5),
					IsOnDialysis: boolPtr(true),
				},
			},
			expected: "CKD Stage 5, Dialysis",
		},
		{
			name: "Adult Male",
			range_: reference.ConditionalReferenceRange{
				RangeConditions: reference.RangeConditions{
					Gender:      stringPtr("M"),
					AgeMinYears: float64Ptr(18),
					AgeMaxYears: float64Ptr(120),
				},
			},
			expected: "Male, Adult",
		},
		{
			name: "Standard (no conditions)",
			range_: reference.ConditionalReferenceRange{
				RangeConditions: reference.RangeConditions{},
			},
			expected: "Standard",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.range_.ContextDescription()
			assert.Equal(t, tt.expected, got)
		})
	}
}

func TestConditionalReferenceRange_IsExpired(t *testing.T) {
	past := time.Now().Add(-24 * time.Hour)
	future := time.Now().Add(24 * time.Hour)

	tests := []struct {
		name           string
		expirationDate *time.Time
		expected       bool
	}{
		{
			name:           "No expiration date",
			expirationDate: nil,
			expected:       false,
		},
		{
			name:           "Expired (past date)",
			expirationDate: &past,
			expected:       true,
		},
		{
			name:           "Not expired (future date)",
			expirationDate: &future,
			expected:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := reference.ConditionalReferenceRange{ExpirationDate: tt.expirationDate}
			assert.Equal(t, tt.expected, r.IsExpired())
		})
	}
}

// =============================================================================
// RANGE CONDITIONS MATCHING TESTS
// =============================================================================

func TestCalculateSpecificityScore(t *testing.T) {
	tests := []struct {
		name       string
		conditions reference.RangeConditions
		expected   int
	}{
		{
			name:       "No conditions (default)",
			conditions: reference.RangeConditions{},
			expected:   0,
		},
		{
			name: "Gender only",
			conditions: reference.RangeConditions{
				Gender: stringPtr("F"),
			},
			expected: 1,
		},
		{
			name: "Gender + Age",
			conditions: reference.RangeConditions{
				Gender:      stringPtr("F"),
				AgeMinYears: float64Ptr(18),
				AgeMaxYears: float64Ptr(50),
			},
			expected: 2, // Gender + Age
		},
		{
			name: "Pregnant (no trimester)",
			conditions: reference.RangeConditions{
				Gender:     stringPtr("F"),
				IsPregnant: boolPtr(true),
			},
			expected: 3, // Gender + Pregnant(2)
		},
		{
			name: "Pregnancy T3",
			conditions: reference.RangeConditions{
				Gender:     stringPtr("F"),
				IsPregnant: boolPtr(true),
				Trimester:  intPtr(3),
			},
			expected: 4, // Gender + Pregnant(2) + Trimester(1)
		},
		{
			name: "CKD Stage 4",
			conditions: reference.RangeConditions{
				CKDStage: intPtr(4),
			},
			expected: 2, // CKD Stage(2)
		},
		{
			name: "CKD Stage 5 + Dialysis",
			conditions: reference.RangeConditions{
				CKDStage:     intPtr(5),
				IsOnDialysis: boolPtr(true),
			},
			expected: 5, // CKD Stage(2) + Dialysis(3)
		},
		{
			name: "Neonatal with GA and hours",
			conditions: reference.RangeConditions{
				AgeMinDays:             intPtr(0),
				AgeMaxDays:             intPtr(28),
				GestationalAgeWeeksMin: intPtr(35),
				GestationalAgeWeeksMax: intPtr(40),
				HoursOfLifeMin:         intPtr(24),
				HoursOfLifeMax:         intPtr(48),
			},
			expected: 4, // AgeDays(1) + GA(2) + Hours(1)
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := reference.CalculateSpecificityScore(&tt.conditions)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestValidateRangeConsistency(t *testing.T) {
	tests := []struct {
		name      string
		range_    reference.ConditionalReferenceRange
		expectErr bool
		errMsg    string
	}{
		{
			name: "Valid range",
			range_: reference.ConditionalReferenceRange{
				LowNormal:    float64Ptr(10.0),
				HighNormal:   float64Ptr(14.0),
				CriticalLow:  float64Ptr(7.0),
				CriticalHigh: float64Ptr(16.0),
				PanicLow:     float64Ptr(5.0),
				PanicHigh:    float64Ptr(18.0),
			},
			expectErr: false,
		},
		{
			name: "Low >= High (invalid)",
			range_: reference.ConditionalReferenceRange{
				LowNormal:  float64Ptr(14.0),
				HighNormal: float64Ptr(10.0),
			},
			expectErr: true,
			errMsg:    "low_normal",
		},
		{
			name: "Critical low >= Low normal (invalid)",
			range_: reference.ConditionalReferenceRange{
				LowNormal:   float64Ptr(10.0),
				HighNormal:  float64Ptr(14.0),
				CriticalLow: float64Ptr(11.0), // Should be < LowNormal
			},
			expectErr: true,
			errMsg:    "critical_low",
		},
		{
			name: "Invalid trimester",
			range_: reference.ConditionalReferenceRange{
				LowNormal:  float64Ptr(10.0),
				HighNormal: float64Ptr(14.0),
				RangeConditions: reference.RangeConditions{
					Trimester: intPtr(5), // Invalid: must be 1-3
				},
			},
			expectErr: true,
			errMsg:    "trimester",
		},
		{
			name: "Invalid CKD stage",
			range_: reference.ConditionalReferenceRange{
				LowNormal:  float64Ptr(10.0),
				HighNormal: float64Ptr(14.0),
				RangeConditions: reference.RangeConditions{
					CKDStage: intPtr(7), // Invalid: must be 1-5
				},
			},
			expectErr: true,
			errMsg:    "ckd_stage",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := reference.ValidateRangeConsistency(&tt.range_)
			if tt.expectErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// =============================================================================
// PATIENT CONTEXT TESTS
// =============================================================================

func TestPatientContext_SetPregnancy(t *testing.T) {
	tests := []struct {
		name              string
		gestationalWeek   int
		expectedTrimester int
	}{
		{"Week 5 - T1", 5, 1},
		{"Week 13 - T1", 13, 1},
		{"Week 14 - T2", 14, 2},
		{"Week 20 - T2", 20, 2},
		{"Week 27 - T2", 27, 2},
		{"Week 28 - T3", 28, 3},
		{"Week 36 - T3", 36, 3},
		{"Week 40 - T3", 40, 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.NewPatientContext("patient-123", 30, "F")
			ctx.SetPregnancy(true, tt.gestationalWeek)

			assert.True(t, ctx.IsPregnant)
			assert.Equal(t, tt.expectedTrimester, ctx.Trimester)
			assert.Equal(t, tt.gestationalWeek, ctx.GestationalWeek)
		})
	}
}

func TestPatientContext_SetEGFR(t *testing.T) {
	tests := []struct {
		name          string
		egfr          float64
		expectedStage int
	}{
		{"eGFR 100 - Stage 1", 100, 1},
		{"eGFR 90 - Stage 1", 90, 1},
		{"eGFR 89 - Stage 2", 89, 2},
		{"eGFR 60 - Stage 2", 60, 2},
		{"eGFR 59 - Stage 3", 59, 3},
		{"eGFR 45 - Stage 3", 45, 3},
		{"eGFR 30 - Stage 3", 30, 3},
		{"eGFR 29 - Stage 4", 29, 4},
		{"eGFR 15 - Stage 4", 15, 4},
		{"eGFR 14 - Stage 5", 14, 5},
		{"eGFR 5 - Stage 5", 5, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.NewPatientContext("patient-123", 60, "M")
			ctx.SetEGFR(tt.egfr)

			assert.Equal(t, tt.egfr, ctx.EGFR)
			assert.Equal(t, tt.expectedStage, ctx.CKDStage)
		})
	}
}

func TestPatientContext_SetNeonatalStatus(t *testing.T) {
	tests := []struct {
		name             string
		ga               int
		hoursOfLife      int
		expectedRiskCat  string
	}{
		{"Term 40w LOW risk", 40, 48, "LOW"},
		{"Term 38w LOW risk", 38, 24, "LOW"},
		{"Late preterm 37w MEDIUM risk", 37, 36, "MEDIUM"},
		{"Late preterm 35w MEDIUM risk", 35, 48, "MEDIUM"},
		{"Preterm 34w HIGH risk", 34, 24, "HIGH"},
		{"Very preterm 30w HIGH risk", 30, 12, "HIGH"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.NewPatientContext("baby-123", 0.01, "M")
			ctx.SetNeonatalStatus(tt.ga, tt.hoursOfLife)

			assert.True(t, ctx.IsNeonate)
			assert.Equal(t, tt.ga, ctx.GestationalAgeAtBirth)
			assert.Equal(t, tt.hoursOfLife, ctx.HoursOfLife)
			assert.Equal(t, tt.expectedRiskCat, ctx.NeonatalRiskCategory)
		})
	}
}

func TestPatientContext_GetAgeCategory(t *testing.T) {
	tests := []struct {
		name     string
		age      float64
		expected string
	}{
		{"Neonate", 0.01, "Neonate"},
		{"Infant", 0.5, "Infant"},
		{"Child", 5, "Pediatric"},
		{"Adolescent", 15, "Adolescent"},
		{"Young Adult", 25, "Adult"},
		{"Middle Adult", 45, "Adult"},
		{"Senior", 70, "Geriatric"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.NewPatientContext("patient-123", tt.age, "M")
			assert.Equal(t, tt.expected, ctx.GetAgeCategory())
		})
	}
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func boolPtr(b bool) *bool {
	return &b
}

func crIntPtr(i int) *int {
	return &i
}

func crFloat64Ptr(f float64) *float64 {
	return &f
}

// stringPtr is defined in arbitration_scenarios_test.go
