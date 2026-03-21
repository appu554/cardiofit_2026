package coding

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConvertGlucose_MmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("mmol/L", "mg/dL", 5.5, "glucose")
	require.NoError(t, err)
	assert.InDelta(t, 99.0, result, 0.5) // 5.5 * 18.0 = 99.0
}

func TestConvertGlucose_MgdLToMmol(t *testing.T) {
	result, err := ConvertUnit("mg/dL", "mmol/L", 126.0, "glucose")
	require.NoError(t, err)
	assert.InDelta(t, 7.0, result, 0.01) // 126 / 18.0 = 7.0
}

func TestConvertBP_KPaToMmHg(t *testing.T) {
	result, err := ConvertUnit("kPa", "mmHg", 16.0, "blood_pressure")
	require.NoError(t, err)
	assert.InDelta(t, 120.0, result, 0.5) // 16.0 * 7.50062 ~ 120.01
}

func TestConvertTemp_FahrenheitToCelsius(t *testing.T) {
	result, err := ConvertUnit("degF", "degC", 98.6, "temperature")
	require.NoError(t, err)
	assert.InDelta(t, 37.0, result, 0.01) // (98.6 - 32) * 5/9 = 37.0
}

func TestConvertTemp_CelsiusToFahrenheit(t *testing.T) {
	result, err := ConvertUnit("degC", "degF", 37.0, "temperature")
	require.NoError(t, err)
	assert.InDelta(t, 98.6, result, 0.01)
}

func TestConvertCholesterol_MmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("mmol/L", "mg/dL", 5.0, "cholesterol")
	require.NoError(t, err)
	assert.InDelta(t, 193.3, result, 0.5) // 5.0 * 38.67 = 193.35
}

func TestConvertTriglycerides_MmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("mmol/L", "mg/dL", 1.7, "triglycerides")
	require.NoError(t, err)
	assert.InDelta(t, 150.5, result, 0.5) // 1.7 * 88.57 = 150.57
}

func TestConvertCreatinine_UmolToMgdL(t *testing.T) {
	result, err := ConvertUnit("umol/L", "mg/dL", 88.4, "creatinine")
	require.NoError(t, err)
	assert.InDelta(t, 1.0, result, 0.01) // 88.4 / 88.4 = 1.0
}

func TestConvertHbA1c_MmolMolToPercent(t *testing.T) {
	result, err := ConvertUnit("mmol/mol", "%", 48.0, "hba1c")
	require.NoError(t, err)
	assert.InDelta(t, 6.5, result, 0.1) // (48 / 10.929) + 2.15 ~ 6.54
}

func TestConvertUnit_SameUnit(t *testing.T) {
	result, err := ConvertUnit("mg/dL", "mg/dL", 126.0, "glucose")
	require.NoError(t, err)
	assert.Equal(t, 126.0, result) // No conversion needed
}

func TestConvertUnit_UnsupportedConversion(t *testing.T) {
	_, err := ConvertUnit("furlongs", "mg/dL", 1.0, "glucose")
	assert.Error(t, err)
}

func TestConvertWeight_KgToLbs(t *testing.T) {
	result, err := ConvertUnit("kg", "lbs", 70.0, "weight")
	require.NoError(t, err)
	assert.InDelta(t, 154.32, result, 0.1) // 70 * 2.20462
}

func TestConvertWeight_LbsToKg(t *testing.T) {
	result, err := ConvertUnit("lbs", "kg", 154.32, "weight")
	require.NoError(t, err)
	assert.InDelta(t, 70.0, result, 0.1)
}

func TestNormalizeToStandardUnit(t *testing.T) {
	tests := []struct {
		name     string
		fromUnit string
		value    float64
		analyte  string
		wantVal  float64
		wantUnit string
	}{
		{"glucose mmol->mg/dL", "mmol/L", 7.0, "glucose", 126.0, "mg/dL"},
		{"glucose mg/dL stays", "mg/dL", 126.0, "glucose", 126.0, "mg/dL"},
		{"BP kPa->mmHg", "kPa", 16.0, "blood_pressure", 120.0, "mmHg"},
		{"temp degF->degC", "degF", 98.6, "temperature", 37.0, "degC"},
		{"cholesterol mmol->mg/dL", "mmol/L", 5.0, "cholesterol", 193.3, "mg/dL"},
		{"creatinine umol->mg/dL", "umol/L", 88.4, "creatinine", 1.0, "mg/dL"},
		{"hba1c mmol/mol->%", "mmol/mol", 48.0, "hba1c", 6.5, "%"},
		{"weight lbs->kg", "lbs", 154.32, "weight", 70.0, "kg"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			val, unit, err := NormalizeToStandardUnit(tt.fromUnit, tt.value, tt.analyte)
			require.NoError(t, err)
			assert.Equal(t, tt.wantUnit, unit)
			assert.InDelta(t, tt.wantVal, val, 0.5)
			_ = math.Abs(0) // suppress unused import
		})
	}
}
