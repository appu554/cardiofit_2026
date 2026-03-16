package services

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"kb-patient-profile/internal/models"
)

// ──────────────────────────────────────────────────────────────────────────
// Helper: mock KB-7 concept lookup
// ──────────────────────────────────────────────────────────────────────────

type mockKB7Lookup struct {
	concepts map[string]*KB7ConceptResult
}

func (m *mockKB7Lookup) LookupConcept(loincCode string) (*KB7ConceptResult, error) {
	if c, ok := m.concepts[loincCode]; ok {
		return c, nil
	}
	return nil, nil
}

func newMockKB7() *mockKB7Lookup {
	return &mockKB7Lookup{
		concepts: map[string]*KB7ConceptResult{
			"2160-0":  {Code: "2160-0", Display: "Creatinine [Mass/volume] in Serum or Plasma"},
			"33914-3": {Code: "33914-3", Display: "Glomerular filtration rate/1.73 sq M.predicted"},
			"1558-6":  {Code: "1558-6", Display: "Fasting glucose [Mass/volume] in Serum or Plasma"},
			"4548-4":  {Code: "4548-4", Display: "Hemoglobin A1c/Hemoglobin.total in Blood"},
			"8480-6":  {Code: "8480-6", Display: "Systolic blood pressure"},
			"8462-4":  {Code: "8462-4", Display: "Diastolic blood pressure"},
			"6298-4":  {Code: "6298-4", Display: "Potassium [Moles/volume] in Serum or Plasma"},
			"2951-2":  {Code: "2951-2", Display: "Sodium [Moles/volume] in Serum or Plasma"},
			"8867-4":  {Code: "8867-4", Display: "Heart rate"},
			"29463-7": {Code: "29463-7", Display: "Body weight"},
			"9318-7":  {Code: "9318-7", Display: "Albumin/Creatinine [Mass Ratio] in Urine"},
		},
	}
}

// ──────────────────────────────────────────────────────────────────────────
// LOINC Registry tests
// ──────────────────────────────────────────────────────────────────────────

func TestLOINCRegistry_Initialize(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock := newMockKB7()
	reg := NewLOINCRegistry(mock, logger)

	reg.Initialize(context.Background())

	assert.True(t, reg.IsReady())
	assert.Contains(t, reg.VerificationSummary(), "11/11")
}

func TestLOINCRegistry_LOINCForLabType(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock := newMockKB7()
	reg := NewLOINCRegistry(mock, logger)
	reg.Initialize(context.Background())

	tests := []struct {
		labType  string
		expected string
	}{
		{models.LabTypeCreatinine, "2160-0"},
		{models.LabTypeEGFR, "33914-3"},
		{models.LabTypeFBG, "1558-6"},
		{models.LabTypeHbA1c, "4548-4"},
		{models.LabTypeSBP, "8480-6"},
		{models.LabTypeDBP, "8462-4"},
		{models.LabTypePotassium, "6298-4"},
		{models.LabTypeSodium, "2951-2"},
		{"HEART_RATE", "8867-4"},
		{"WEIGHT", "29463-7"},
		{models.LabTypeACR, "9318-7"},
		{"UNKNOWN_TYPE", ""},
	}

	for _, tc := range tests {
		t.Run(tc.labType, func(t *testing.T) {
			assert.Equal(t, tc.expected, reg.LOINCForLabType(tc.labType))
		})
	}
}

func TestLOINCRegistry_LabTypeForLOINC(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock := newMockKB7()
	reg := NewLOINCRegistry(mock, logger)
	reg.Initialize(context.Background())

	assert.Equal(t, models.LabTypeCreatinine, reg.LabTypeForLOINC("2160-0"))
	assert.Equal(t, models.LabTypeSodium, reg.LabTypeForLOINC("2951-2"))
	assert.Equal(t, "", reg.LabTypeForLOINC("UNKNOWN"))
}

func TestLOINCRegistry_AllMappingsVerified(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	mock := newMockKB7()
	reg := NewLOINCRegistry(mock, logger)
	reg.Initialize(context.Background())

	mappings := reg.AllMappings()
	assert.Len(t, mappings, 11)

	for _, m := range mappings {
		assert.True(t, m.Verified, "Expected %s (%s) to be verified", m.LabType, m.LOINCCode)
		assert.NotEmpty(t, m.Display, "Expected display name for %s", m.LabType)
	}
}

func TestLOINCRegistry_GracefulDegradation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	// Empty mock — all lookups return nil (simulating KB-7 down)
	mock := &mockKB7Lookup{concepts: map[string]*KB7ConceptResult{}}
	reg := NewLOINCRegistry(mock, logger)
	reg.Initialize(context.Background())

	assert.True(t, reg.IsReady())
	assert.Contains(t, reg.VerificationSummary(), "0/11")

	// Codes should still be available (unverified)
	assert.Equal(t, "2160-0", reg.LOINCForLabType(models.LabTypeCreatinine))
	m := reg.GetMapping(models.LabTypeCreatinine)
	require.NotNil(t, m)
	assert.False(t, m.Verified)
}

// ──────────────────────────────────────────────────────────────────────────
// Pure logic tests (no database needed)
// ──────────────────────────────────────────────────────────────────────────

func TestHasDrugClass_WithFDC(t *testing.T) {
	meds := []models.MedicationState{
		{DrugClass: "METFORMIN", IsActive: true},
		{DrugClass: "FDC", IsActive: true, FDCComponents: []string{"ACE_INHIBITOR", "DIURETIC"}},
	}

	assert.True(t, hasDrugClass(meds, models.DrugClassACEInhibitor), "FDC should decompose to ACE_INHIBITOR")
	assert.True(t, hasDrugClass(meds, models.DrugClassDiuretic), "FDC should decompose to DIURETIC")
	assert.True(t, hasDrugClass(meds, models.DrugClassMetformin), "Direct metformin match")
	assert.False(t, hasDrugClass(meds, models.DrugClassBetaBlocker), "Beta blocker not present")
}

func TestEffectiveDrugClasses_Dedup(t *testing.T) {
	meds := []models.MedicationState{
		{DrugClass: "ACE_INHIBITOR", IsActive: true},
		{DrugClass: "FDC", IsActive: true, FDCComponents: []string{"ACE_INHIBITOR", "DIURETIC"}},
	}

	classes := effectiveDrugClasses(meds)
	assert.Len(t, classes, 2, "ACE_INHIBITOR should be deduped + DIURETIC + FDC parent excluded")
	assert.Contains(t, classes, "ACE_INHIBITOR")
	assert.Contains(t, classes, "DIURETIC")
}

func TestJCurveSBPFloor(t *testing.T) {
	tests := []struct {
		stage    string
		expected *float64
	}{
		{"3a", float64Ptr(120)},
		{"3b", float64Ptr(125)},
		{"4", float64Ptr(130)},
		{"5", float64Ptr(135)},
		{"1", nil},
		{"2", nil},
		{"", nil},
	}

	for _, tc := range tests {
		t.Run("stage_"+tc.stage, func(t *testing.T) {
			result := jCurveSBPFloor(tc.stage)
			if tc.expected == nil {
				assert.Nil(t, result)
			} else {
				require.NotNil(t, result)
				assert.Equal(t, *tc.expected, *result)
			}
		})
	}
}

// ──────────────────────────────────────────────────────────────────────────
// Staleness model tests
// ──────────────────────────────────────────────────────────────────────────

func TestStalenessThresholds(t *testing.T) {
	assert.Equal(t, 90, models.StalenessThresholdEGFR)
	assert.Equal(t, 90, models.StalenessThresholdHbA1c)
	assert.Equal(t, 14, models.StalenessThresholdCreatinine)
	assert.Equal(t, 14, models.StalenessThresholdPotassium)
}
