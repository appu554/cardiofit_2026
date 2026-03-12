package services

import (
	"testing"

	"github.com/lib/pq"
	"github.com/stretchr/testify/assert"

	"kb-patient-profile/internal/models"
)

func TestEffectiveDrugClasses_SingleDrug(t *testing.T) {
	tests := []struct {
		name      string
		drugClass string
	}{
		{"Metformin", models.DrugClassMetformin},
		{"SGLT2i", models.DrugClassSGLT2I},
		{"DPP4i", models.DrugClassDPP4I},
		{"Sulfonylurea", models.DrugClassSulfonylurea},
		{"CCB", models.DrugClassCCB},
		{"ARB", models.DrugClassARB},
		{"ACE Inhibitor", models.DrugClassACEInhibitor},
		{"Insulin", models.DrugClassInsulin},
		{"Statin", models.DrugClassStatin},
		{"Beta Blocker", models.DrugClassBetaBlocker},
		{"Diuretic", models.DrugClassDiuretic},
		{"GLP1RA", models.DrugClassGLP1RA},
		{"Thiazolidinedione", models.DrugClassThiazolidinedione},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			med := models.MedicationState{
				DrugClass: tc.drugClass,
			}
			classes := med.EffectiveDrugClasses()
			assert.Equal(t, []string{tc.drugClass}, classes,
				"Single drug should return its own class")
		})
	}
}

func TestEffectiveDrugClasses_IndiaSpecific(t *testing.T) {
	indiaClasses := []string{
		models.DrugClassTeneligliptin,
		models.DrugClassSaroglitazar,
		models.DrugClassRemogliflozin,
		models.DrugClassDualPPAR,
		models.DrugClassVoglibose,
	}

	for _, dc := range indiaClasses {
		t.Run(dc, func(t *testing.T) {
			med := models.MedicationState{DrugClass: dc}
			classes := med.EffectiveDrugClasses()
			assert.Equal(t, []string{dc}, classes,
				"India-specific drug class should be recognized")
		})
	}
}

func TestEffectiveDrugClasses_FDCDecomposition(t *testing.T) {
	tests := []struct {
		name           string
		drugClass      string
		fdcComponents  []string
		wantClasses    []string
	}{
		{
			name:          "Metformin + Glimepiride FDC",
			drugClass:     "FDC",
			fdcComponents: []string{models.DrugClassMetformin, models.DrugClassSulfonylurea},
			wantClasses:   []string{models.DrugClassMetformin, models.DrugClassSulfonylurea},
		},
		{
			name:          "Metformin + Vildagliptin FDC",
			drugClass:     "FDC",
			fdcComponents: []string{models.DrugClassMetformin, models.DrugClassDPP4I},
			wantClasses:   []string{models.DrugClassMetformin, models.DrugClassDPP4I},
		},
		{
			name:          "Triple FDC — Metformin + Glimepiride + Voglibose",
			drugClass:     "FDC",
			fdcComponents: []string{models.DrugClassMetformin, models.DrugClassSulfonylurea, models.DrugClassVoglibose},
			wantClasses:   []string{models.DrugClassMetformin, models.DrugClassSulfonylurea, models.DrugClassVoglibose},
		},
		{
			name:          "ARB + CCB FDC",
			drugClass:     "FDC",
			fdcComponents: []string{models.DrugClassARB, models.DrugClassCCB},
			wantClasses:   []string{models.DrugClassARB, models.DrugClassCCB},
		},
		{
			name:          "India-specific FDC — Teneligliptin + Metformin",
			drugClass:     "FDC",
			fdcComponents: []string{models.DrugClassTeneligliptin, models.DrugClassMetformin},
			wantClasses:   []string{models.DrugClassTeneligliptin, models.DrugClassMetformin},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			med := models.MedicationState{
				DrugClass:     tc.drugClass,
				FDCComponents: pq.StringArray(tc.fdcComponents),
			}
			classes := med.EffectiveDrugClasses()
			assert.Equal(t, tc.wantClasses, classes,
				"FDC should decompose into all component drug classes")
		})
	}
}

func TestEffectiveDrugClasses_EmptyFDC(t *testing.T) {
	med := models.MedicationState{
		DrugClass:     models.DrugClassMetformin,
		FDCComponents: pq.StringArray{},
	}
	classes := med.EffectiveDrugClasses()
	assert.Equal(t, []string{models.DrugClassMetformin}, classes,
		"Empty FDC components should return the drug's own class")
}

func TestEffectiveDrugClasses_NilFDC(t *testing.T) {
	med := models.MedicationState{
		DrugClass:     models.DrugClassStatin,
		FDCComponents: nil,
	}
	classes := med.EffectiveDrugClasses()
	assert.Equal(t, []string{models.DrugClassStatin}, classes,
		"Nil FDC components should return the drug's own class")
}

func TestDrugClassConstants_Exist(t *testing.T) {
	// Verify all drug class constants are non-empty and distinct
	allClasses := []string{
		models.DrugClassMetformin,
		models.DrugClassSGLT2I,
		models.DrugClassDPP4I,
		models.DrugClassSulfonylurea,
		models.DrugClassCCB,
		models.DrugClassARB,
		models.DrugClassACEInhibitor,
		models.DrugClassInsulin,
		models.DrugClassStatin,
		models.DrugClassBetaBlocker,
		models.DrugClassDiuretic,
		models.DrugClassGLP1RA,
		models.DrugClassThiazolidinedione,
		// India-specific
		models.DrugClassTeneligliptin,
		models.DrugClassSaroglitazar,
		models.DrugClassRemogliflozin,
		models.DrugClassDualPPAR,
		models.DrugClassVoglibose,
	}

	seen := make(map[string]bool)
	for _, dc := range allClasses {
		assert.NotEmpty(t, dc, "Drug class constant should not be empty")
		assert.False(t, seen[dc], "Duplicate drug class constant: %s", dc)
		seen[dc] = true
	}

	assert.Len(t, seen, 18, "Should have 18 distinct drug class constants (13 standard + 5 India-specific)")
}
