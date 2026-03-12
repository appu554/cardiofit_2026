package models

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistryCode_IsValid(t *testing.T) {
	testCases := []struct {
		code     RegistryCode
		expected bool
	}{
		{RegistryDiabetes, true},
		{RegistryHypertension, true},
		{RegistryHeartFailure, true},
		{RegistryCKD, true},
		{RegistryCOPD, true},
		{RegistryPregnancy, true},
		{RegistryOpioidUse, true},
		{RegistryAnticoagulation, true},
		{RegistryCode("INVALID"), false},
		{RegistryCode(""), false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.code), func(t *testing.T) {
			result := tc.code.IsValid()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRegistryCode_String(t *testing.T) {
	testCases := []struct {
		code     RegistryCode
		expected string
	}{
		{RegistryDiabetes, "DIABETES"},
		{RegistryHypertension, "HYPERTENSION"},
		{RegistryHeartFailure, "HEART_FAILURE"},
	}

	for _, tc := range testCases {
		t.Run(string(tc.code), func(t *testing.T) {
			result := tc.code.String()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRiskTier_IsValid(t *testing.T) {
	testCases := []struct {
		tier     RiskTier
		expected bool
	}{
		{RiskTierLow, true},
		{RiskTierModerate, true},
		{RiskTierHigh, true},
		{RiskTierCritical, true},
		{RiskTier("INVALID"), false},
		{RiskTier(""), false},
	}

	for _, tc := range testCases {
		t.Run(string(tc.tier), func(t *testing.T) {
			result := tc.tier.IsValid()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRiskTier_Priority(t *testing.T) {
	testCases := []struct {
		tier     RiskTier
		expected int
	}{
		{RiskTierLow, 1},
		{RiskTierModerate, 2},
		{RiskTierHigh, 3},
		{RiskTierCritical, 4},
		{RiskTier("INVALID"), 0},
	}

	for _, tc := range testCases {
		t.Run(string(tc.tier), func(t *testing.T) {
			result := tc.tier.Priority()
			assert.Equal(t, tc.expected, result)
		})
	}
}

func TestRiskTier_Comparison(t *testing.T) {
	assert.True(t, RiskTierCritical.Priority() > RiskTierHigh.Priority())
	assert.True(t, RiskTierHigh.Priority() > RiskTierModerate.Priority())
	assert.True(t, RiskTierModerate.Priority() > RiskTierLow.Priority())
}

func TestRegistry_JSONMarshaling(t *testing.T) {
	registry := Registry{
		Code:        RegistryDiabetes,
		Name:        "Diabetes Mellitus Registry",
		Description: "Type 1 and Type 2 Diabetes",
		Active:      true,
		InclusionCriteria: CriteriaGroupSlice{
			{
				ID:       "diabetes-dx",
				Operator: LogicalOr,
				Criteria: []Criterion{
					{
						Type:       CriteriaTypeDiagnosis,
						Field:      "code",
						Operator:   OperatorStartsWith,
						Value:      "E11",
						CodeSystem: CodeSystemICD10,
					},
				},
			},
		},
	}

	// Marshal
	data, err := json.Marshal(registry)
	require.NoError(t, err)
	assert.Contains(t, string(data), "DIABETES")
	assert.Contains(t, string(data), "Diabetes Mellitus Registry")

	// Unmarshal
	var decoded Registry
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)
	assert.Equal(t, registry.Code, decoded.Code)
	assert.Equal(t, registry.Name, decoded.Name)
	assert.Equal(t, registry.Active, decoded.Active)
	assert.Len(t, decoded.InclusionCriteria, 1)
}

func TestRegistryStats_JSONMarshaling(t *testing.T) {
	stats := RegistryStats{
		RegistryCode:  RegistryDiabetes,
		TotalEnrolled: 1000,
		ActiveCount:   950,
		LowRiskCount:  200,
		ModerateCount: 400,
		HighRiskCount: 250,
		CriticalCount: 100,
	}

	data, err := json.Marshal(stats)
	require.NoError(t, err)

	var decoded RegistryStats
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, stats.RegistryCode, decoded.RegistryCode)
	assert.Equal(t, stats.TotalEnrolled, decoded.TotalEnrolled)
	assert.Equal(t, stats.HighRiskCount, decoded.HighRiskCount)
}

func TestCriteriaGroupSlice_Value(t *testing.T) {
	slice := CriteriaGroupSlice{
		{ID: "test", Operator: LogicalAnd},
	}

	val, err := slice.Value()
	require.NoError(t, err)
	assert.NotNil(t, val)
}

func TestStringSlice_Value(t *testing.T) {
	slice := StringSlice{"a", "b", "c"}

	val, err := slice.Value()
	require.NoError(t, err)
	assert.NotNil(t, val)
}

func TestJSONMap_Value(t *testing.T) {
	m := JSONMap{"key": "value"}

	val, err := m.Value()
	require.NoError(t, err)
	assert.NotNil(t, val)
}
