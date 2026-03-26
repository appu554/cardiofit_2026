package models

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPatientProfile_V4FieldsExist(t *testing.T) {
	p := PatientProfile{}
	// BP Variability fields
	assert.Nil(t, p.ARVSBP7d)
	assert.Empty(t, p.DipClassification)
	assert.Empty(t, p.BPControlStatus)
	// MHRI fields
	assert.Nil(t, p.MHRIScore)
	assert.Empty(t, p.MHRITrajectory)
	// Engagement fields
	assert.Nil(t, p.EngagementComposite)
	assert.Empty(t, p.EngagementStatus)
	// CKM stage
	assert.Equal(t, 0, p.CKMStage)
	// Data tier default
	assert.Equal(t, "", p.DataTier) // GORM default applies at DB level
}

func TestCKMStage_Constants(t *testing.T) {
	assert.Equal(t, 0, CKMStage0)
	assert.Equal(t, 1, CKMStage1)
	assert.Equal(t, 2, CKMStage2)
	assert.Equal(t, 3, CKMStage3)
	assert.Equal(t, 4, CKMStage4)
}

func TestComputeCKMStage_Stage0_NoRiskFactors(t *testing.T) {
	p := PatientProfile{
		BMI:                22.0,
		WaistToHeightRatio: floatPtr(0.45),
		HbA1c:              floatPtr(5.4),
		EGFR:               floatPtr(95.0),
		UACR:               floatPtr(15.0),
		MHRIScore:          floatPtr(85.0),
	}
	assert.Equal(t, CKMStage0, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage1_ExcessAdiposityOnly(t *testing.T) {
	p := PatientProfile{
		BMI:                28.0,
		WaistToHeightRatio: floatPtr(0.58),
		HbA1c:              floatPtr(5.5),
		EGFR:               floatPtr(90.0),
		UACR:               floatPtr(20.0),
	}
	assert.Equal(t, CKMStage1, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage2_T2DM(t *testing.T) {
	p := PatientProfile{
		BMI:                30.0,
		WaistToHeightRatio: floatPtr(0.60),
		HbA1c:              floatPtr(7.5),
		EGFR:               floatPtr(55.0),
		UACR:               floatPtr(150.0),
		DiabetesYears:      intPtr(5),
	}
	assert.Equal(t, CKMStage2, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage3_HighASCVDRisk(t *testing.T) {
	p := PatientProfile{
		BMI:                31.0,
		WaistToHeightRatio: floatPtr(0.62),
		HbA1c:              floatPtr(8.0),
		EGFR:               floatPtr(45.0),
		UACR:               floatPtr(300.0),
		ASCVDRisk10y:       floatPtr(22.0),
	}
	assert.Equal(t, CKMStage3, ComputeCKMStage(p))
}

func TestComputeCKMStage_Stage4_ClinicalCVDEvent(t *testing.T) {
	p := PatientProfile{
		BMI:            29.0,
		HasClinicalCVD: true,
	}
	assert.Equal(t, CKMStage4, ComputeCKMStage(p))
}

// floatPtr is defined in stratum.go (same package)
func intPtr(i int) *int { return &i }
