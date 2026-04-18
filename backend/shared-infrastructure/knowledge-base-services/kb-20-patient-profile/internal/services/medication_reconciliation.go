package services

import (
	"strings"

	"kb-patient-profile/internal/models"
)

// MedicationEntry is one drug on a medication list (pre-admission or discharge).
type MedicationEntry struct {
	DrugName  string
	DrugClass string
	DoseMg    float64
	Frequency string
}

// highRiskNewDrugClasses are drug classes that get CRITICAL risk when newly added.
var highRiskNewDrugClasses = map[string]bool{
	"ANTICOAGULANT":    true,
	"INSULIN":          true,
	"OPIOID":           true,
	"DIGOXIN":          true,
	"AMIODARONE":       true,
	"IMMUNOSUPPRESSANT": true,
}

// cardioprotectiveClasses are drug classes whose withdrawal post-MI/ACS/HF is risky.
var cardioprotectiveClasses = map[string]bool{
	"BETA_BLOCKER": true,
	"ACEi":         true,
	"ARB":          true,
	"STATIN":       true,
}

// unclearDrugTerms are generic/unclear medication references requiring clarification.
var unclearDrugTerms = []string{
	"cardiac medications",
	"heart medications",
	"current medications",
}

// ReconcileRegimens compares pre-admission vs discharge medication lists.
// patientEGFR: current eGFR for renal appropriateness checks (nil if unavailable).
// diagnosis: primary admission diagnosis for context-aware risk assessment.
func ReconcileRegimens(
	preAdmission []MedicationEntry,
	discharge []MedicationEntry,
	patientEGFR *float64,
	diagnosis string,
) models.MedicationReconciliationReport {
	report := models.MedicationReconciliationReport{}

	// Build lookup of pre-admission drugs keyed by normalised name.
	preMap := make(map[string]MedicationEntry, len(preAdmission))
	preMatched := make(map[string]bool, len(preAdmission))
	for _, m := range preAdmission {
		key := normaliseDrugName(m.DrugName)
		preMap[key] = m
		preMatched[key] = false
	}

	// Classify each discharge drug.
	for _, d := range discharge {
		// Check for unclear/generic entries first.
		if isUnclearEntry(d.DrugName) {
			dm := toDischargeMed(d, models.MedStatusUnclear, "MEDIUM", "")
			report.UnclearMedications = append(report.UnclearMedications, dm)
			continue
		}

		key := normaliseDrugName(d.DrugName)
		pre, found := preMap[key]
		if !found {
			// NEW drug on discharge.
			risk := classifyNewDrugRisk(d, patientEGFR)
			dm := toDischargeMed(d, models.MedStatusNew, risk, "")
			report.NewMedications = append(report.NewMedications, dm)
		} else {
			preMatched[key] = true
			if pre.DoseMg == d.DoseMg && strings.EqualFold(pre.Frequency, d.Frequency) {
				// CONTINUED — same drug, same dose, same frequency.
				dm := toDischargeMed(d, models.MedStatusContinued, "LOW", pre.DrugName)
				report.ContinuedMedications = append(report.ContinuedMedications, dm)
			} else {
				// CHANGED_DOSE — drug present but dose or frequency differs.
				dm := toDischargeMed(d, models.MedStatusChangedDose, "MEDIUM", pre.DrugName)
				report.ChangedMedications = append(report.ChangedMedications, dm)
			}
		}
	}

	// Any pre-admission drug not matched → STOPPED.
	for _, m := range preAdmission {
		key := normaliseDrugName(m.DrugName)
		if !preMatched[key] {
			risk := classifyStoppedDrugRisk(m, diagnosis)
			dm := toDischargeMed(m, models.MedStatusStopped, risk, m.DrugName)
			report.StoppedMedications = append(report.StoppedMedications, dm)
		}
	}

	// Compute summary counts.
	report.DiscrepanciesFound = len(report.NewMedications) +
		len(report.StoppedMedications) +
		len(report.ChangedMedications) +
		len(report.UnclearMedications)

	report.HighRiskChanges = countHighRisk(report)
	report.ReconciliationOutcome = determineOutcome(report)

	return report
}

// classifyNewDrugRisk assigns risk level for a newly added drug.
func classifyNewDrugRisk(d MedicationEntry, patientEGFR *float64) string {
	cls := strings.ToUpper(d.DrugClass)

	// Intrinsically high-risk drug classes → CRITICAL.
	if highRiskNewDrugClasses[cls] {
		return "CRITICAL"
	}

	// Renal-inappropriate prescribing checks.
	if patientEGFR != nil {
		if cls == "BIGUANIDE" && *patientEGFR < 45 {
			return "HIGH"
		}
		if cls == "SGLT2I" && *patientEGFR < 25 {
			return "HIGH"
		}
	}

	return "MEDIUM"
}

// classifyStoppedDrugRisk assigns risk level for a stopped drug.
func classifyStoppedDrugRisk(m MedicationEntry, diagnosis string) string {
	cls := strings.ToUpper(m.DrugClass)
	if cardioprotectiveClasses[cls] && diagnosisIndicatesCardioprotection(diagnosis) {
		return "HIGH"
	}
	return "LOW"
}

// diagnosisIndicatesCardioprotection returns true if the diagnosis context
// means cardioprotective agents should generally not be stopped.
func diagnosisIndicatesCardioprotection(diagnosis string) bool {
	upper := strings.ToUpper(diagnosis)
	for _, term := range []string{"MI", "ACS", "HEART_FAILURE"} {
		if strings.Contains(upper, term) {
			return true
		}
	}
	return false
}

// isUnclearEntry checks if the drug name is a generic/unclear reference.
func isUnclearEntry(drugName string) bool {
	lower := strings.ToLower(strings.TrimSpace(drugName))
	for _, term := range unclearDrugTerms {
		if lower == term {
			return true
		}
	}
	return false
}

// normaliseDrugName returns a canonical form for matching.
func normaliseDrugName(name string) string {
	return strings.ToLower(strings.TrimSpace(name))
}

// toDischargeMed converts a MedicationEntry into a DischargeMedication model.
func toDischargeMed(entry MedicationEntry, status, risk, preAdmName string) models.DischargeMedication {
	return models.DischargeMedication{
		DrugName:             entry.DrugName,
		DrugClass:            entry.DrugClass,
		DoseMg:               entry.DoseMg,
		Frequency:            entry.Frequency,
		ReconciliationStatus: status,
		ClinicalRiskLevel:    risk,
		PreAdmissionDrugName: preAdmName,
	}
}

// countHighRisk counts drugs classified as CRITICAL or HIGH risk across all categories.
func countHighRisk(r models.MedicationReconciliationReport) int {
	count := 0
	for _, lists := range [][]models.DischargeMedication{
		r.NewMedications,
		r.StoppedMedications,
		r.ChangedMedications,
		r.UnclearMedications,
	} {
		for _, m := range lists {
			if m.ClinicalRiskLevel == "CRITICAL" || m.ClinicalRiskLevel == "HIGH" {
				count++
			}
		}
	}
	return count
}

// determineOutcome selects the reconciliation outcome string.
// Priority: UNCLEAR > CRITICAL > HIGH > CLEAN.
func determineOutcome(r models.MedicationReconciliationReport) string {
	if r.DiscrepanciesFound == 0 {
		return "CLEAN"
	}
	if len(r.UnclearMedications) > 0 {
		return "UNCLEAR_INSUFFICIENT_DATA"
	}
	if hasCriticalRisk(r) {
		return "HIGH_RISK_URGENT"
	}
	if r.HighRiskChanges > 0 {
		return "DISCREPANCIES_CLINICIAN_REVIEW"
	}
	// Discrepancies exist but none are CRITICAL/HIGH — still needs review
	return "DISCREPANCIES_CLINICIAN_REVIEW"
}

// hasCriticalRisk checks if any drug across all lists has CRITICAL risk.
func hasCriticalRisk(r models.MedicationReconciliationReport) bool {
	for _, lists := range [][]models.DischargeMedication{
		r.NewMedications,
		r.StoppedMedications,
		r.ChangedMedications,
		r.UnclearMedications,
	} {
		for _, m := range lists {
			if m.ClinicalRiskLevel == "CRITICAL" {
				return true
			}
		}
	}
	return false
}
