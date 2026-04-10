package services

import (
	"testing"
	"time"

	"kb-patient-profile/internal/models"
)

func floatP(f float64) *float64 { return &f }
func intP(i int) *int           { return &i }

func TestClassifyCKM_Stage0_NoRiskFactors(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 30, BMI: 22.0, EGFR: 95,
	})
	if result.Stage != models.CKMStageV2_0 {
		t.Errorf("expected Stage 0, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage1_Adiposity(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 35, BMI: 27.0, EGFR: 90,
	})
	if result.Stage != models.CKMStageV2_1 {
		t.Errorf("expected Stage 1, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage1_AsianBMI(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 35, BMI: 24.0, EGFR: 90, AsianBMICutoffs: true,
	})
	if result.Stage != models.CKMStageV2_1 {
		t.Errorf("expected Stage 1 with Asian BMI cutoff 23, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage2_Diabetes(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 50, BMI: 28.0, EGFR: 75, HasDiabetes: true,
	})
	if result.Stage != models.CKMStageV2_2 {
		t.Errorf("expected Stage 2, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage2_CKD(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 60, BMI: 24.0, EGFR: 48, ACR: floatP(45.0),
	})
	if result.Stage != models.CKMStageV2_2 {
		t.Errorf("expected Stage 2 with eGFR <60, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage3_HighRisk(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 55, BMI: 30.0, EGFR: 65, HasDiabetes: true, HasHTN: true,
		PREVENTScore: floatP(15.0),
	})
	if result.Stage != models.CKMStageV2_3 {
		t.Errorf("expected Stage 3, got %s", result.Stage)
	}
}

func TestClassifyCKM_Stage4a_SubclinicalCAC(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 58, BMI: 29.0, EGFR: 60, HasDiabetes: true, HasHTN: true,
		CACScore: floatP(350.0),
	})
	if result.Stage != models.CKMStageV2_4a {
		t.Errorf("expected Stage 4a, got %s", result.Stage)
	}
	if result.Metadata.CACScore == nil || *result.Metadata.CACScore != 350.0 {
		t.Error("expected CACScore=350 in metadata")
	}
}

func TestClassifyCKM_Stage4b_PriorMI(t *testing.T) {
	miDate := time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 62, BMI: 27.0, EGFR: 55, HasDiabetes: true,
		ASCVDEvents: []models.ASCVDEvent{
			{EventType: "MI", EventDate: miDate, Details: "STEMI anterior"},
		},
	})
	if result.Stage != models.CKMStageV2_4b {
		t.Errorf("expected Stage 4b, got %s", result.Stage)
	}
	if len(result.Metadata.ASCVDEvents) != 1 {
		t.Error("expected 1 ASCVD event in metadata")
	}
}

func TestClassifyCKM_Stage4c_HFrEF(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 58, BMI: 30.0, EGFR: 45,
		HasHeartFailure: true, LVEF: floatP(35.0), NYHAClass: "II",
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected Stage 4c, got %s", result.Stage)
	}
	if result.Metadata.HFClassification != models.HFTypeReduced {
		t.Errorf("expected HFrEF, got %s", result.Metadata.HFClassification)
	}
}

func TestClassifyCKM_Stage4c_HFpEF(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 68, BMI: 35.0, EGFR: 50, HasDiabetes: true,
		HasHeartFailure: true, LVEF: floatP(55.0), NYHAClass: "III",
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected Stage 4c, got %s", result.Stage)
	}
	if result.Metadata.HFClassification != models.HFTypePreserved {
		t.Errorf("expected HFpEF, got %s", result.Metadata.HFClassification)
	}
}

func TestClassifyCKM_Stage4c_Rheumatic(t *testing.T) {
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 32, BMI: 22.0, EGFR: 80,
		HasHeartFailure: true, LVEF: floatP(45.0), NYHAClass: "II",
		HFEtiology: "RHEUMATIC", RheumaticEtiology: true,
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected Stage 4c, got %s", result.Stage)
	}
	if !result.Metadata.RheumaticEtiology {
		t.Error("expected RheumaticEtiology=true")
	}
}

func TestClassifyCKM_Hierarchy_4c_Trumps_4b(t *testing.T) {
	// Patient with both HF and prior MI → 4c wins (higher severity)
	miDate := time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
	result := ClassifyCKMStage(CKMClassifierInput{
		Age: 65, BMI: 28.0, EGFR: 40,
		HasHeartFailure: true, LVEF: floatP(30.0), NYHAClass: "III",
		ASCVDEvents: []models.ASCVDEvent{{EventType: "MI", EventDate: miDate}},
	})
	if result.Stage != models.CKMStageV2_4c {
		t.Errorf("expected 4c to trump 4b, got %s", result.Stage)
	}
}
