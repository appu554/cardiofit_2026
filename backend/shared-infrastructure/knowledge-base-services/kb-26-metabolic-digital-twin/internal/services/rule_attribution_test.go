package services

import (
	"testing"

	"github.com/google/uuid"
	"kb-26-metabolic-digital-twin/internal/models"
)

func attrInput(strategy string, outcomeOccurred bool, tier string) AttributionInput {
	occurred := outcomeOccurred
	return AttributionInput{
		ConsolidatedRecordID: uuid.New(),
		PatientID:            "P-test",
		CohortID:             "hcf_catalyst_chf",
		TreatmentStrategy:    strategy,
		OutcomeOccurred:      &occurred,
		OutcomeType:          "READMISSION_30D",
		HorizonDays:          30,
		PreAlertRiskScore:    62.0,
		PreAlertRiskTier:     tier,
	}
}

func TestAttribution_HighRiskInterventionNoOutcome_Prevented(t *testing.T) {
	v := ComputeAttribution(attrInput("INTERVENTION_TAKEN", false, "HIGH"))
	if v.ClinicianLabel != string(models.LabelPrevented) {
		t.Fatalf("expected prevented, got %s", v.ClinicianLabel)
	}
	if v.RiskReductionPct <= 0 {
		t.Fatalf("expected positive risk reduction, got %f", v.RiskReductionPct)
	}
}

func TestAttribution_InterventionOutcomeOccurred_Despite(t *testing.T) {
	v := ComputeAttribution(attrInput("INTERVENTION_TAKEN", true, "HIGH"))
	if v.ClinicianLabel != string(models.LabelOutcomeDespiteIntervention) {
		t.Fatalf("expected outcome_despite_intervention, got %s", v.ClinicianLabel)
	}
}

func TestAttribution_LowRiskInterventionNoOutcome_NoEffect(t *testing.T) {
	v := ComputeAttribution(attrInput("INTERVENTION_TAKEN", false, "LOW"))
	if v.ClinicianLabel != string(models.LabelNoEffectDetected) {
		t.Fatalf("expected no_effect_detected, got %s", v.ClinicianLabel)
	}
}

func TestAttribution_HighRiskOverrideNoOutcome_Fragile(t *testing.T) {
	v := ComputeAttribution(attrInput("OVERRIDE_WITH_REASON", false, "HIGH"))
	if v.ClinicianLabel != string(models.LabelFragileEstimate) {
		t.Fatalf("expected fragile_estimate, got %s", v.ClinicianLabel)
	}
}
