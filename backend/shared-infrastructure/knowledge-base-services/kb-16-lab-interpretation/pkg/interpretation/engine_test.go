package interpretation

import (
	"context"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/types"
)

func TestEngine_Interpret(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	refDB := reference.NewDatabase()
	engine := NewEngine(refDB, nil, log)

	tests := []struct {
		name           string
		result         *types.LabResult
		patientCtx     *types.PatientContext
		expectedFlag   types.InterpretationFlag
		expectedPanic  bool
		expectedCrit   bool
	}{
		{
			name: "Normal potassium",
			result: &types.LabResult{
				Code:         "2823-3",
				Name:         "Potassium",
				ValueNumeric: testFloatPtr(4.0),
				Unit:         "mmol/L",
			},
			patientCtx:    nil,
			expectedFlag:  types.FlagNormal,
			expectedPanic: false,
			expectedCrit:  false,
		},
		{
			name: "High potassium",
			result: &types.LabResult{
				Code:         "2823-3",
				Name:         "Potassium",
				ValueNumeric: testFloatPtr(5.5),
				Unit:         "mmol/L",
			},
			patientCtx:    nil,
			expectedFlag:  types.FlagHigh,
			expectedPanic: false,
			expectedCrit:  false,
		},
		{
			name: "Critical high potassium (above critical threshold)",
			result: &types.LabResult{
				Code:         "2823-3",
				Name:         "Potassium",
				ValueNumeric: testFloatPtr(6.2),
				Unit:         "mmol/L",
			},
			patientCtx:    nil,
			expectedFlag:  types.FlagHigh, // Flag stays HIGH, but isCritical is true
			expectedPanic: false,
			expectedCrit:  true,
		},
		{
			name: "Panic high potassium (above panic threshold)",
			result: &types.LabResult{
				Code:         "2823-3",
				Name:         "Potassium",
				ValueNumeric: testFloatPtr(7.0),
				Unit:         "mmol/L",
			},
			patientCtx:    nil,
			expectedFlag:  types.FlagPanicHigh, // Panic overrides flag
			expectedPanic: true,
			expectedCrit:  true, // Also above critical threshold
		},
		{
			name: "Normal glucose",
			result: &types.LabResult{
				Code:         "2345-7",
				Name:         "Glucose",
				ValueNumeric: testFloatPtr(95.0),
				Unit:         "mg/dL",
			},
			patientCtx:    nil,
			expectedFlag:  types.FlagNormal,
			expectedPanic: false,
			expectedCrit:  false,
		},
		{
			name: "Panic low glucose (below panic threshold)",
			result: &types.LabResult{
				Code:         "2345-7",
				Name:         "Glucose",
				ValueNumeric: testFloatPtr(35.0),
				Unit:         "mg/dL",
			},
			patientCtx:    nil,
			expectedFlag:  types.FlagPanicLow,
			expectedPanic: true,
			expectedCrit:  true, // Also below critical threshold (50)
		},
		{
			name: "Low hemoglobin",
			result: &types.LabResult{
				Code:         "718-7",
				Name:         "Hemoglobin",
				ValueNumeric: testFloatPtr(10.5),
				Unit:         "g/dL",
			},
			patientCtx: &types.PatientContext{
				Age: 40,
				Sex: "male",
			},
			expectedFlag:  types.FlagLow,
			expectedPanic: false,
			expectedCrit:  false,
		},
		{
			name: "Critical low hemoglobin (below critical threshold)",
			result: &types.LabResult{
				Code:         "718-7",
				Name:         "Hemoglobin",
				ValueNumeric: testFloatPtr(6.5),
				Unit:         "g/dL",
			},
			patientCtx:    nil,
			expectedFlag:  types.FlagLow, // Flag stays LOW, but isCritical is true
			expectedPanic: false,
			expectedCrit:  true,
		},
	}

	ctx := context.Background()

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := engine.Interpret(ctx, tt.result, tt.patientCtx)
			require.NoError(t, err)
			require.NotNil(t, result)

			assert.Equal(t, tt.expectedFlag, result.Interpretation.Flag, "Flag mismatch")
			assert.Equal(t, tt.expectedPanic, result.Interpretation.IsPanic, "IsPanic mismatch")
			assert.Equal(t, tt.expectedCrit, result.Interpretation.IsCritical, "IsCritical mismatch")

			// Verify clinical comment is generated
			assert.NotEmpty(t, result.Interpretation.ClinicalComment, "Clinical comment should not be empty")
		})
	}
}

func TestEngine_ClassifyResult(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	refDB := reference.NewDatabase()
	engine := NewEngine(refDB, nil, log)

	tests := []struct {
		name         string
		value        float64
		ranges       *types.ReferenceRange
		expectedFlag types.InterpretationFlag
	}{
		{
			name:  "Within normal range",
			value: 4.0,
			ranges: &types.ReferenceRange{
				Low:  testFloatPtr(3.5),
				High: testFloatPtr(5.0),
			},
			expectedFlag: types.FlagNormal,
		},
		{
			name:  "Below normal",
			value: 3.0,
			ranges: &types.ReferenceRange{
				Low:  testFloatPtr(3.5),
				High: testFloatPtr(5.0),
			},
			expectedFlag: types.FlagLow,
		},
		{
			name:  "Above normal",
			value: 5.5,
			ranges: &types.ReferenceRange{
				Low:  testFloatPtr(3.5),
				High: testFloatPtr(5.0),
			},
			expectedFlag: types.FlagHigh,
		},
		{
			name:  "Critical low",
			value: 2.0,
			ranges: &types.ReferenceRange{
				Low:         testFloatPtr(3.5),
				High:        testFloatPtr(5.0),
				CriticalLow: testFloatPtr(2.5),
			},
			expectedFlag: types.FlagCriticalLow,
		},
		{
			name:  "Critical high",
			value: 7.0,
			ranges: &types.ReferenceRange{
				Low:          testFloatPtr(3.5),
				High:         testFloatPtr(5.0),
				CriticalHigh: testFloatPtr(6.0),
			},
			expectedFlag: types.FlagCriticalHigh,
		},
		{
			name:  "Panic low",
			value: 1.5,
			ranges: &types.ReferenceRange{
				Low:      testFloatPtr(3.5),
				High:     testFloatPtr(5.0),
				PanicLow: testFloatPtr(2.0),
			},
			expectedFlag: types.FlagPanicLow,
		},
		{
			name:  "Panic high",
			value: 8.0,
			ranges: &types.ReferenceRange{
				Low:       testFloatPtr(3.5),
				High:      testFloatPtr(5.0),
				PanicHigh: testFloatPtr(7.0),
			},
			expectedFlag: types.FlagPanicHigh,
		},
		{
			name:         "Nil ranges returns normal",
			value:        5.0,
			ranges:       nil,
			expectedFlag: types.FlagNormal,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flag := engine.classifyResult(tt.value, tt.ranges)
			assert.Equal(t, tt.expectedFlag, flag)
		})
	}
}

func TestEngine_DetermineSeverity(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	refDB := reference.NewDatabase()
	engine := NewEngine(refDB, nil, log)

	tests := []struct {
		name             string
		flag             types.InterpretationFlag
		isPanic          bool
		isCritical       bool
		delta            *types.DeltaCheckResult
		expectedSeverity types.Severity
	}{
		{
			name:             "Panic value is critical severity",
			flag:             types.FlagPanicHigh,
			isPanic:          true,
			isCritical:       false,
			delta:            nil,
			expectedSeverity: types.SeverityCritical,
		},
		{
			name:             "Critical value is high severity",
			flag:             types.FlagCriticalHigh,
			isPanic:          false,
			isCritical:       true,
			delta:            nil,
			expectedSeverity: types.SeverityHigh,
		},
		{
			name:       "Significant delta is medium severity",
			flag:       types.FlagNormal,
			isPanic:    false,
			isCritical: false,
			delta: &types.DeltaCheckResult{
				IsSignificant: true,
			},
			expectedSeverity: types.SeverityMedium,
		},
		{
			name:             "Abnormal flag is low severity",
			flag:             types.FlagHigh,
			isPanic:          false,
			isCritical:       false,
			delta:            nil,
			expectedSeverity: types.SeverityLow,
		},
		{
			name:             "Normal flag is low severity",
			flag:             types.FlagNormal,
			isPanic:          false,
			isCritical:       false,
			delta:            nil,
			expectedSeverity: types.SeverityLow,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			severity := engine.determineSeverity(tt.flag, tt.isPanic, tt.isCritical, tt.delta)
			assert.Equal(t, tt.expectedSeverity, severity)
		})
	}
}

func TestEngine_GenerateRecommendations(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	refDB := reference.NewDatabase()
	engine := NewEngine(refDB, nil, log)

	// Test panic value recommendations
	result := &types.LabResult{
		Code:         "2823-3",
		Name:         "Potassium",
		ValueNumeric: testFloatPtr(7.0),
		Unit:         "mmol/L",
	}

	recs := engine.generateRecommendations(result, types.FlagPanicHigh, true, false, nil)
	assert.NotEmpty(t, recs)
	assert.True(t, containsRecommendation(recs, "URGENT: Notify physician immediately"), "Should contain urgent notification")

	// Test critical value recommendations
	recs = engine.generateRecommendations(result, types.FlagCriticalHigh, false, true, nil)
	assert.NotEmpty(t, recs)
	assert.True(t, containsRecommendation(recs, "Review with ordering clinician within 30 minutes"), "Should contain clinician review")

	// Test potassium-specific recommendations
	recs = engine.generateRecommendations(result, types.FlagCriticalHigh, false, true, nil)
	assert.True(t, containsRecommendation(recs, "Order ECG to assess for cardiac effects"), "Should contain ECG recommendation")
}

func TestEngine_InterpretNonNumeric(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	refDB := reference.NewDatabase()
	engine := NewEngine(refDB, nil, log)

	result := &types.LabResult{
		Code:        "test-code",
		Name:        "Test Result",
		ValueString: "Positive",
	}

	interpreted, err := engine.Interpret(context.Background(), result, nil)
	require.NoError(t, err)
	require.NotNil(t, interpreted)

	assert.Equal(t, types.FlagNormal, interpreted.Interpretation.Flag)
	assert.False(t, interpreted.Interpretation.IsCritical)
	assert.False(t, interpreted.Interpretation.IsPanic)
	assert.Contains(t, interpreted.Interpretation.ClinicalComment, "Positive")
}

// Helper function to create float64 pointers
func testFloatPtr(f float64) *float64 {
	return &f
}

// containsRecommendation checks if any recommendation has the given description
func containsRecommendation(recs []types.Recommendation, description string) bool {
	for _, rec := range recs {
		if rec.Description == description {
			return true
		}
	}
	return false
}
