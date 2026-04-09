package services

import (
	"fmt"
	"math"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// ActiveMedication — input to the gating engine
// ---------------------------------------------------------------------------

// ActiveMedication represents a single medication on the patient's active list.
type ActiveMedication struct {
	DrugClass    string  `json:"drug_class"`
	DrugName     string  `json:"drug_name"`
	CurrentDoseMg float64 `json:"current_dose_mg"`
	IsNew        bool    `json:"is_new"`
}

// ---------------------------------------------------------------------------
// RenalDoseGate — core gating engine
// ---------------------------------------------------------------------------

// RenalDoseGate evaluates medications against renal function thresholds.
type RenalDoseGate struct {
	formulary *RenalFormulary
}

// NewRenalDoseGate creates a gating engine backed by the given formulary.
func NewRenalDoseGate(formulary *RenalFormulary) *RenalDoseGate {
	return &RenalDoseGate{formulary: formulary}
}

// ---------------------------------------------------------------------------
// Evaluate — single-medication verdict
// ---------------------------------------------------------------------------

// Evaluate returns a gating result for one medication against renal status.
// Evaluation order (first match wins):
//  1. Stale eGFR (>CriticalDays → INSUFFICIENT_DATA)
//  2. Potassium co-gating (high K+ → CONTRAINDICATED; nil K+ at risk → MONITOR_ESCALATE)
//  3. Hard contraindication by eGFR
//  4. Efficacy cliff (→ DOSE_REDUCE with substitute)
//  5. Dose reduction zone
//  6. Monitor escalation zone
//  7. CLEARED
func (g *RenalDoseGate) Evaluate(med ActiveMedication, rs models.RenalStatus) models.MedicationGatingResult {
	now := time.Now()

	base := models.MedicationGatingResult{
		DrugClass:   med.DrugClass,
		DrugName:    med.DrugName,
		CurrentDoseMg: med.CurrentDoseMg,
		EGFR:        rs.EGFR,
		EvaluatedAt: now,
	}

	rule := g.formulary.GetRule(med.DrugClass)
	if rule == nil {
		base.Verdict = models.VerdictCleared
		base.Reason = "no renal rule defined for drug class"
		base.ClinicalAction = "continue current therapy"
		return base
	}

	base.SourceGuideline = rule.SourceGuideline
	base.MonitoringRequired = buildMonitoringList(rule)

	// ------- 1. Stale eGFR check -------
	daysSinceMeasurement := int(math.Round(now.Sub(rs.EGFRMeasuredAt).Hours() / 24))
	if daysSinceMeasurement > g.formulary.StaleEGFR.CriticalDays {
		base.Verdict = models.VerdictInsufficientData
		base.Reason = fmt.Sprintf("eGFR is %d days old (critical threshold: %d days)",
			daysSinceMeasurement, g.formulary.StaleEGFR.CriticalDays)
		base.ClinicalAction = "order urgent renal function panel before prescribing"
		return base
	}

	// ------- 2. Potassium co-gating -------
	if rule.RequiresPotassiumCheck {
		if rs.Potassium != nil && *rs.Potassium >= rule.PotassiumContraAbove {
			base.Verdict = models.VerdictContraindicated
			base.Reason = fmt.Sprintf("K+ %.1f >= %.1f (potassium contraindication for %s)",
				*rs.Potassium, rule.PotassiumContraAbove, med.DrugClass)
			base.ClinicalAction = fmt.Sprintf("discontinue %s; consider alternative if K+ normalises", med.DrugClass)
			if rule.SubstituteClass != "" {
				base.SubstituteClass = rule.SubstituteClass
			}
			return base
		}
		// Nil potassium at low eGFR → need monitoring before proceeding
		if rs.Potassium == nil && rs.EGFR < rule.MonitorEscalateBelow {
			base.Verdict = models.VerdictMonitorEscalate
			base.Reason = fmt.Sprintf("no potassium data available; eGFR %.1f < %.1f requires K+ monitoring for %s",
				rs.EGFR, rule.MonitorEscalateBelow, med.DrugClass)
			base.ClinicalAction = "order potassium level before dose adjustment; increase renal monitoring"
			base.MonitoringFrequency = "weekly"
			return base
		}
	}

	// ------- 3. Hard contraindication -------
	if rule.ContraindicatedBelow > 0 && rs.EGFR < rule.ContraindicatedBelow {
		base.Verdict = models.VerdictContraindicated
		base.Reason = fmt.Sprintf("eGFR %.1f below %.1f — contraindicated for %s",
			rs.EGFR, rule.ContraindicatedBelow, med.DrugClass)
		base.ClinicalAction = fmt.Sprintf("discontinue %s", med.DrugClass)
		if rule.SubstituteClass != "" {
			base.SubstituteClass = rule.SubstituteClass
			base.ClinicalAction += fmt.Sprintf("; consider %s", rule.SubstituteClass)
		}
		return base
	}

	// ------- 4. Efficacy cliff -------
	if rule.EfficacyCliffBelow > 0 && rs.EGFR < rule.EfficacyCliffBelow {
		base.Verdict = models.VerdictDoseReduce
		base.Reason = fmt.Sprintf("eGFR %.1f below efficacy cliff %.1f for %s",
			rs.EGFR, rule.EfficacyCliffBelow, med.DrugClass)
		base.ClinicalAction = fmt.Sprintf("reduced efficacy at this eGFR; switch to %s", rule.SubstituteClass)
		base.SubstituteClass = rule.SubstituteClass
		return base
	}

	// ------- 5. Dose reduction -------
	if rule.DoseReduceBelow > 0 && rs.EGFR < rule.DoseReduceBelow {
		base.Verdict = models.VerdictDoseReduce
		base.Reason = fmt.Sprintf("eGFR %.1f below dose-reduce threshold %.1f for %s",
			rs.EGFR, rule.DoseReduceBelow, med.DrugClass)
		if rule.MaxDoseReducedMg > 0 {
			base.MaxSafeDoseMg = &rule.MaxDoseReducedMg
			base.ClinicalAction = fmt.Sprintf("reduce dose to max %.0f mg", rule.MaxDoseReducedMg)
		} else {
			base.ClinicalAction = "reduce dose per specialist guidance"
		}
		base.MonitoringFrequency = "monthly"
		return base
	}

	// ------- 6. Monitor escalation -------
	if rule.MonitorEscalateBelow > 0 && rs.EGFR < rule.MonitorEscalateBelow {
		base.Verdict = models.VerdictMonitorEscalate
		base.Reason = fmt.Sprintf("eGFR %.1f below monitor-escalate threshold %.1f for %s",
			rs.EGFR, rule.MonitorEscalateBelow, med.DrugClass)
		base.ClinicalAction = "increase monitoring frequency; reassess at next visit"
		base.MonitoringFrequency = "monthly"
		return base
	}

	// ------- 7. Cleared -------
	base.Verdict = models.VerdictCleared
	base.Reason = fmt.Sprintf("eGFR %.1f is above all thresholds for %s", rs.EGFR, med.DrugClass)
	base.ClinicalAction = "continue current therapy"
	return base
}

// ---------------------------------------------------------------------------
// EvaluatePatient — multi-medication report
// ---------------------------------------------------------------------------

// EvaluatePatient evaluates all active medications for a patient and returns
// an aggregate report with urgency classification.
func (g *RenalDoseGate) EvaluatePatient(patientID string, rs models.RenalStatus, meds []ActiveMedication) models.PatientGatingReport {
	report := models.PatientGatingReport{
		PatientID:   patientID,
		RenalStatus: rs,
	}

	now := time.Now()
	daysSince := int(math.Round(now.Sub(rs.EGFRMeasuredAt).Hours() / 24))
	if daysSince > g.formulary.StaleEGFR.CriticalDays {
		report.StaleEGFR = true
		report.StaleEGFRDays = daysSince
	}

	results := make([]models.MedicationGatingResult, 0, len(meds))
	for _, med := range meds {
		r := g.Evaluate(med, rs)
		results = append(results, r)

		switch r.Verdict {
		case models.VerdictContraindicated:
			report.HasContraindicated = true
		case models.VerdictDoseReduce:
			report.HasDoseReduce = true
		}
	}
	report.MedicationResults = results

	// Determine overall urgency
	switch {
	case report.HasContraindicated:
		report.OverallUrgency = "IMMEDIATE"
	case report.HasDoseReduce || report.StaleEGFR:
		report.OverallUrgency = "URGENT"
	default:
		report.OverallUrgency = "ROUTINE"
	}

	return report
}

// ---------------------------------------------------------------------------
// BlockRecommendation — hard gate for card builder safety
// ---------------------------------------------------------------------------

// BlockRecommendation checks whether a drug class should be blocked from
// recommendation given current renal status. Returns (blocked, reason).
// It checks both hard contraindication and PBS/initiation thresholds.
func (g *RenalDoseGate) BlockRecommendation(drugClass string, rs models.RenalStatus) (bool, string) {
	rule := g.formulary.GetRule(drugClass)
	if rule == nil {
		return false, ""
	}

	// Hard contraindication
	if rule.ContraindicatedBelow > 0 && rs.EGFR < rule.ContraindicatedBelow {
		return true, fmt.Sprintf("%s contraindicated at eGFR %.1f (threshold: %.1f)",
			drugClass, rs.EGFR, rule.ContraindicatedBelow)
	}

	// PBS initiation threshold (for new prescriptions)
	if rule.InitiationMinEGFR > 0 && rs.EGFR < rule.InitiationMinEGFR {
		return true, fmt.Sprintf("%s initiation blocked at eGFR %.1f (PBS min: %.1f)",
			drugClass, rs.EGFR, rule.InitiationMinEGFR)
	}

	// Potassium hard gate
	if rule.RequiresPotassiumCheck && rs.Potassium != nil && *rs.Potassium >= rule.PotassiumContraAbove {
		return true, fmt.Sprintf("%s blocked: K+ %.1f >= %.1f",
			drugClass, *rs.Potassium, rule.PotassiumContraAbove)
	}

	return false, ""
}

// ---------------------------------------------------------------------------
// buildMonitoringList — helper
// ---------------------------------------------------------------------------

func buildMonitoringList(rule *models.RenalDrugRule) []string {
	monitors := []string{"eGFR", "serum creatinine"}
	if rule.RequiresPotassiumCheck {
		monitors = append(monitors, "serum potassium")
	}
	return monitors
}
