package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"
)

func TestBaseline_Day1_HospitalInfluenced(t *testing.T) {
	ctrl := NewBaselineAdjustmentController()

	stage := ctrl.DetermineBaselineStage(1, 0)

	if stage != models.BaselineStageHospitalInfluenced {
		t.Errorf("expected %s, got %s", models.BaselineStageHospitalInfluenced, stage)
	}
}

func TestBaseline_Day5_Building_FewReadings(t *testing.T) {
	ctrl := NewBaselineAdjustmentController()

	stage := ctrl.DetermineBaselineStage(5, 3)

	if stage != models.BaselineStageBuildingNew {
		t.Errorf("expected %s, got %s", models.BaselineStageBuildingNew, stage)
	}
}

func TestBaseline_Day15_Evolved(t *testing.T) {
	ctrl := NewBaselineAdjustmentController()

	stage := ctrl.DetermineBaselineStage(15, 8)

	if stage != models.BaselineStagePostDischargeEvolving {
		t.Errorf("expected %s, got %s", models.BaselineStagePostDischargeEvolving, stage)
	}
}

func TestBaseline_Day31_SteadyState(t *testing.T) {
	ctrl := NewBaselineAdjustmentController()

	// STEADY_STATE regardless of reading count
	stage := ctrl.DetermineBaselineStage(31, 0)

	if stage != models.BaselineStageSteadyState {
		t.Errorf("expected %s, got %s", models.BaselineStageSteadyState, stage)
	}
}

func TestBaseline_CriticalBypassesSuppression(t *testing.T) {
	ctrl := NewBaselineAdjustmentController()

	// Even during HOSPITAL_INFLUENCED, CRITICAL severity must NOT be suppressed
	suppressed := ctrl.ShouldSuppressDeviation(models.BaselineStageHospitalInfluenced, "CRITICAL")

	if suppressed {
		t.Error("expected CRITICAL to bypass suppression during HOSPITAL_INFLUENCED stage")
	}
}
