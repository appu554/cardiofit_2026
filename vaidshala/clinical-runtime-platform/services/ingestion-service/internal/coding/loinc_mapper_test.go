package coding

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLookupLOINC_Found(t *testing.T) {
	entry, ok := LookupLOINC("33914-3")
	require.True(t, ok)
	assert.Equal(t, "egfr", entry.Analyte)
	assert.Equal(t, "mL/min/1.73m2", entry.StdUnit)
	assert.Equal(t, "LABS", entry.Category)
}

func TestLookupLOINC_NotFound(t *testing.T) {
	_, ok := LookupLOINC("99999-9")
	assert.False(t, ok)
}

func TestLookupLOINCByAnalyte_Found(t *testing.T) {
	code, ok := LookupLOINCByAnalyte("fasting_glucose")
	require.True(t, ok)
	assert.Equal(t, "1558-6", code)
}

func TestLookupLOINCByAnalyte_NotFound(t *testing.T) {
	_, ok := LookupLOINCByAnalyte("unknown_analyte")
	assert.False(t, ok)
}

func TestLookupSNOMED_Found(t *testing.T) {
	entry, ok := LookupSNOMED("diabetes_mellitus_2")
	require.True(t, ok)
	assert.Equal(t, "44054006", entry.Code)
}

func TestLookupSNOMED_NotFound(t *testing.T) {
	_, ok := LookupSNOMED("nonexistent")
	assert.False(t, ok)
}

func TestLOINCRegistry_CoversCoreAnalytes(t *testing.T) {
	coreAnalytes := []string{
		"fasting_glucose", "hba1c", "egfr", "creatinine", "potassium",
		"cholesterol", "hdl", "ldl", "triglycerides",
		"systolic_bp", "diastolic_bp", "heart_rate", "weight",
	}
	for _, a := range coreAnalytes {
		code, ok := LookupLOINCByAnalyte(a)
		assert.True(t, ok, "missing LOINC mapping for analyte %q", a)
		_, found := LookupLOINC(code)
		assert.True(t, found, "LOINC code %s for analyte %q not in registry", code, a)
	}
}
