package services

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReconciliation_NewAnticoagulant_CriticalRisk(t *testing.T) {
	pre := []MedicationEntry{
		{DrugName: "metoprolol", DrugClass: "BETA_BLOCKER", DoseMg: 50, Frequency: "BD"},
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
	}
	discharge := []MedicationEntry{
		{DrugName: "metoprolol", DrugClass: "BETA_BLOCKER", DoseMg: 50, Frequency: "BD"},
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
		{DrugName: "warfarin", DrugClass: "ANTICOAGULANT", DoseMg: 5, Frequency: "OD"},
	}

	report := ReconcileRegimens(pre, discharge, nil, "AF")

	// warfarin must be classified NEW with CRITICAL risk
	require.Len(t, report.NewMedications, 1)
	assert.Equal(t, "warfarin", report.NewMedications[0].DrugName)
	assert.Equal(t, "NEW", report.NewMedications[0].ReconciliationStatus)
	assert.Equal(t, "CRITICAL", report.NewMedications[0].ClinicalRiskLevel)

	// continued drugs
	assert.Len(t, report.ContinuedMedications, 2)
	assert.Len(t, report.StoppedMedications, 0)

	assert.Equal(t, 1, report.DiscrepanciesFound)
	assert.Equal(t, 1, report.HighRiskChanges)
	assert.Equal(t, "HIGH_RISK_URGENT", report.ReconciliationOutcome)
}

func TestReconciliation_StoppedBetaBlocker_PostMI_HighRisk(t *testing.T) {
	pre := []MedicationEntry{
		{DrugName: "metoprolol", DrugClass: "BETA_BLOCKER", DoseMg: 50, Frequency: "BD"},
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
		{DrugName: "ramipril", DrugClass: "ACEi", DoseMg: 5, Frequency: "OD"},
	}
	discharge := []MedicationEntry{
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
		{DrugName: "ramipril", DrugClass: "ACEi", DoseMg: 5, Frequency: "OD"},
	}

	report := ReconcileRegimens(pre, discharge, nil, "MI")

	require.Len(t, report.StoppedMedications, 1)
	assert.Equal(t, "metoprolol", report.StoppedMedications[0].DrugName)
	assert.Equal(t, "STOPPED", report.StoppedMedications[0].ReconciliationStatus)
	assert.Equal(t, "HIGH", report.StoppedMedications[0].ClinicalRiskLevel)

	assert.Len(t, report.ContinuedMedications, 2)
	assert.Equal(t, 1, report.DiscrepanciesFound)
	assert.Equal(t, 1, report.HighRiskChanges)
	assert.Equal(t, "DISCREPANCIES_CLINICIAN_REVIEW", report.ReconciliationOutcome)
}

func TestReconciliation_ContinuedStatin_NotFlagged(t *testing.T) {
	pre := []MedicationEntry{
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
	}
	discharge := []MedicationEntry{
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
	}

	report := ReconcileRegimens(pre, discharge, nil, "")

	require.Len(t, report.ContinuedMedications, 1)
	assert.Equal(t, "CONTINUED", report.ContinuedMedications[0].ReconciliationStatus)
	assert.Equal(t, "LOW", report.ContinuedMedications[0].ClinicalRiskLevel)

	assert.Len(t, report.NewMedications, 0)
	assert.Len(t, report.StoppedMedications, 0)
	assert.Len(t, report.ChangedMedications, 0)
	assert.Len(t, report.UnclearMedications, 0)

	assert.Equal(t, 0, report.DiscrepanciesFound)
	assert.Equal(t, 0, report.HighRiskChanges)
	assert.Equal(t, "CLEAN", report.ReconciliationOutcome)
}

func TestReconciliation_UnclearCardiacMeds_Clarification(t *testing.T) {
	pre := []MedicationEntry{
		{DrugName: "metoprolol", DrugClass: "BETA_BLOCKER", DoseMg: 50, Frequency: "BD"},
		{DrugName: "ramipril", DrugClass: "ACEi", DoseMg: 5, Frequency: "OD"},
		{DrugName: "amlodipine", DrugClass: "CCB", DoseMg: 5, Frequency: "OD"},
	}
	discharge := []MedicationEntry{
		{DrugName: "cardiac medications", DrugClass: "", DoseMg: 0, Frequency: ""},
	}

	report := ReconcileRegimens(pre, discharge, nil, "")

	require.Len(t, report.UnclearMedications, 1)
	assert.Equal(t, "UNCLEAR", report.UnclearMedications[0].ReconciliationStatus)

	// pre-admission drugs not matched → STOPPED
	assert.Len(t, report.StoppedMedications, 3)

	assert.Equal(t, "UNCLEAR_INSUFFICIENT_DATA", report.ReconciliationOutcome)
}

func TestReconciliation_NewMetforminLowEGFR_HighRisk(t *testing.T) {
	pre := []MedicationEntry{
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
	}
	discharge := []MedicationEntry{
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
		{DrugName: "metformin", DrugClass: "BIGUANIDE", DoseMg: 500, Frequency: "BD"},
	}

	egfr := 35.0
	report := ReconcileRegimens(pre, discharge, &egfr, "T2DM")

	require.Len(t, report.NewMedications, 1)
	assert.Equal(t, "metformin", report.NewMedications[0].DrugName)
	assert.Equal(t, "NEW", report.NewMedications[0].ReconciliationStatus)
	assert.Equal(t, "HIGH", report.NewMedications[0].ClinicalRiskLevel)

	assert.Equal(t, 1, report.DiscrepanciesFound)
	assert.Equal(t, 1, report.HighRiskChanges)
	assert.Equal(t, "DISCREPANCIES_CLINICIAN_REVIEW", report.ReconciliationOutcome)
}

func TestReconciliation_CleanReconciliation_NoIssues(t *testing.T) {
	meds := []MedicationEntry{
		{DrugName: "metoprolol", DrugClass: "BETA_BLOCKER", DoseMg: 50, Frequency: "BD"},
		{DrugName: "atorvastatin", DrugClass: "STATIN", DoseMg: 40, Frequency: "OD"},
		{DrugName: "ramipril", DrugClass: "ACEi", DoseMg: 5, Frequency: "OD"},
	}

	report := ReconcileRegimens(meds, meds, nil, "HTN")

	assert.Len(t, report.ContinuedMedications, 3)
	assert.Len(t, report.NewMedications, 0)
	assert.Len(t, report.StoppedMedications, 0)
	assert.Len(t, report.ChangedMedications, 0)
	assert.Len(t, report.UnclearMedications, 0)

	assert.Equal(t, 0, report.DiscrepanciesFound)
	assert.Equal(t, 0, report.HighRiskChanges)
	assert.Equal(t, "CLEAN", report.ReconciliationOutcome)
}
