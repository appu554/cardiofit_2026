package services

import "kb-23-decision-cards/internal/models"

// ---------------------------------------------------------------------------
// EnrichedConflictReport — combined safety output for card pipeline
// ---------------------------------------------------------------------------

// EnrichedConflictReport aggregates renal gating, anticipatory alerts, and
// stale-eGFR detection into a single safety report for the card builder.
type EnrichedConflictReport struct {
	RenalGating        *models.PatientGatingReport `json:"renal_gating,omitempty"`
	AnticipatoryAlerts []AnticipatoryAlert          `json:"anticipatory_alerts,omitempty"`
	StaleEGFR          *StaleEGFRResult             `json:"stale_egfr,omitempty"`
	HasSafetyBlock     bool                         `json:"has_safety_block"`
	BlockedDrugClasses []string                     `json:"blocked_drug_classes,omitempty"`
}

// ---------------------------------------------------------------------------
// DetectAllConflicts — unified safety pipeline
// ---------------------------------------------------------------------------

// DetectAllConflicts runs the full renal safety pipeline for a patient:
//  1. Renal gating via EvaluatePatient (medication-level verdicts)
//  2. Anticipatory alerts via FindApproachingThresholds (trajectory warnings)
//  3. Stale eGFR via DetectStaleEGFR (data freshness check)
//
// HasSafetyBlock is set if any medication is CONTRAINDICATED or DOSE_REDUCE,
// or if the stale-eGFR severity is CRITICAL.
func DetectAllConflicts(
	gate *RenalDoseGate,
	formulary *RenalFormulary,
	patientID string,
	renal models.RenalStatus,
	meds []ActiveMedication,
	egfrSlope float64,
) EnrichedConflictReport {
	report := EnrichedConflictReport{}

	// 1. Renal gating — per-medication verdicts
	gatingReport := gate.EvaluatePatient(patientID, renal, meds)
	report.RenalGating = &gatingReport

	// 2. Anticipatory alerts — trajectory-based warnings
	report.AnticipatoryAlerts = FindApproachingThresholds(formulary, renal.EGFR, egfrSlope, meds)

	// 3. Stale eGFR detection
	onRenalSensitiveMed := len(meds) > 0 // conservative: any med → sensitive
	staleResult := DetectStaleEGFR(renal, formulary.StaleEGFR, onRenalSensitiveMed)
	report.StaleEGFR = &staleResult

	// Determine safety block and blocked classes
	for _, r := range gatingReport.MedicationResults {
		if r.Verdict == models.VerdictContraindicated || r.Verdict == models.VerdictDoseReduce {
			report.HasSafetyBlock = true
			report.BlockedDrugClasses = append(report.BlockedDrugClasses, r.DrugClass)
		}
	}

	// Critical stale eGFR also triggers safety block
	if staleResult.Severity == "CRITICAL" {
		report.HasSafetyBlock = true
	}

	return report
}
