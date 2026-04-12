package services

import "kb-23-decision-cards/internal/models"

// ---------------------------------------------------------------------------
// EnrichedConflictReport — combined safety output for card pipeline
// ---------------------------------------------------------------------------

// HFBlockResult records an HF-contraindicated drug that was blocked.
type HFBlockResult struct {
	DrugClass string `json:"drug_class"`
	Reason    string `json:"reason"`
}

// RenalHFConflict represents a cross-domain conflict where a medication is
// simultaneously mandated by HF guidelines and contraindicated by renal status.
// This generates a compound referral card rather than two contradictory cards.
type RenalHFConflict struct {
	DrugClass             string `json:"drug_class"`
	HFMandateReason       string `json:"hf_mandate_reason"`
	RenalBlockReason      string `json:"renal_block_reason"`
	ResolutionRecommendation string `json:"resolution_recommendation"`
	Urgency               string `json:"urgency"`
}

// EnrichedConflictReport aggregates renal gating, anticipatory alerts, and
// stale-eGFR detection into a single safety report for the card builder.
type EnrichedConflictReport struct {
	RenalGating         *models.PatientGatingReport `json:"renal_gating,omitempty"`
	AnticipatoryAlerts  []AnticipatoryAlert          `json:"anticipatory_alerts,omitempty"`
	StaleEGFR           *StaleEGFRResult             `json:"stale_egfr,omitempty"`
	HFContraindications []HFBlockResult              `json:"hf_contraindications,omitempty"`
	RenalHFConflicts    []RenalHFConflict            `json:"renal_hf_conflicts,omitempty"`
	MandatoryMedGaps    []MandatoryMedGap            `json:"mandatory_med_gaps,omitempty"`
	HasSafetyBlock      bool                         `json:"has_safety_block"`
	HasCriticalConflict bool                         `json:"has_critical_conflict"` // true when HFpEF has no available GDMT
	BlockedDrugClasses  []string                     `json:"blocked_drug_classes,omitempty"`
}

// ---------------------------------------------------------------------------
// DetectAllConflicts — unified safety pipeline
// ---------------------------------------------------------------------------

// DetectAllConflicts runs the full renal safety pipeline for a patient:
//  1. Renal gating via EvaluatePatient (medication-level verdicts)
//  2. Anticipatory alerts via FindApproachingThresholds (trajectory warnings)
//  3. Stale eGFR via DetectStaleEGFR (data freshness check)
//  4. HF medication gating (Stage 4c)
//  5. Mandatory medication gap detection (4a/4b/4c)
//  6. Renal-HF cross-domain conflict resolution
//
// HasSafetyBlock is set if any medication is CONTRAINDICATED or DOSE_REDUCE,
// or if the stale-eGFR severity is CRITICAL.
// HasCriticalConflict is set when HFpEF loses its only disease-modifying therapy.
func DetectAllConflicts(
	gate *RenalDoseGate,
	formulary *RenalFormulary,
	patientID string,
	renal models.RenalStatus,
	meds []ActiveMedication,
	egfrSlope float64,
	ckmStage string,
	hfType string,
	ctx ...ClinicalContext,
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

	// Determine safety block and blocked classes from renal gating
	renalBlockedClasses := make([]string, 0)
	for _, r := range gatingReport.MedicationResults {
		if r.Verdict == models.VerdictContraindicated || r.Verdict == models.VerdictDoseReduce {
			report.HasSafetyBlock = true
			report.BlockedDrugClasses = append(report.BlockedDrugClasses, r.DrugClass)
			if r.Verdict == models.VerdictContraindicated {
				renalBlockedClasses = append(renalBlockedClasses, r.DrugClass)
			}
		}
	}

	// Critical stale eGFR also triggers safety block
	if staleResult.Severity == "CRITICAL" {
		report.HasSafetyBlock = true
	}

	// 4. HF medication gating (Stage 4c)
	if ckmStage == "4c" {
		hfGate := NewHFMedicationGate()
		for _, m := range meds {
			blocked, reason := hfGate.CheckContraindication(m.DrugClass, ckmStage, hfType)
			if blocked {
				report.HasSafetyBlock = true
				report.HFContraindications = append(report.HFContraindications, HFBlockResult{
					DrugClass: m.DrugClass,
					Reason:    reason,
				})
				report.BlockedDrugClasses = append(report.BlockedDrugClasses, m.DrugClass)
			}
		}
	}

	// 5. Mandatory medication gap detection for Stage 4
	if ckmStage == "4a" || ckmStage == "4b" || ckmStage == "4c" {
		checker := NewMandatoryMedChecker()
		activeClasses := make([]string, 0, len(meds))
		for _, m := range meds {
			activeClasses = append(activeClasses, m.DrugClass)
		}

		var clinCtx ClinicalContext
		if len(ctx) > 0 {
			clinCtx = ctx[0]
		}
		// Inject renal-blocked classes (from active meds) into ClinicalContext
		clinCtx.BlockedByRenal = append(clinCtx.BlockedByRenal, renalBlockedClasses...)

		// Prospective renal check: for HF GDMT classes the patient is NOT on,
		// probe the renal rule to see if it would be contraindicated at current eGFR.
		// This catches "should add MRA (HFrEF mandate) but eGFR=28 blocks it" cases.
		prospectiveClasses := []string{"MRA", "SGLT2i", "ACEi", "ARB", "SACUBITRIL_VALSARTAN"}
		activeSet := make(map[string]bool, len(activeClasses))
		for _, c := range activeClasses {
			activeSet[c] = true
		}
		for _, cls := range prospectiveClasses {
			if activeSet[cls] {
				continue // already handled by actual renal gating above
			}
			rule := formulary.GetRule(cls)
			if rule == nil {
				continue
			}
			if rule.ContraindicatedBelow > 0 && renal.EGFR < rule.ContraindicatedBelow {
				clinCtx.BlockedByRenal = append(clinCtx.BlockedByRenal, cls)
			}
		}

		gaps := checker.CheckMandatory(ckmStage, hfType, activeClasses, clinCtx)
		report.MandatoryMedGaps = gaps

		// 6. Cross-domain resolution: compound cards for suppressed gaps
		for _, g := range gaps {
			if !g.Suppressed {
				continue
			}
			conflict := RenalHFConflict{
				DrugClass:        g.MissingClass,
				HFMandateReason:  g.Rationale,
				RenalBlockReason: "Renal gate contraindicates this class at current eGFR",
				ResolutionRecommendation: "Refer for renal-cardio shared decision-making — " +
					"guideline-mandated therapy cannot be initiated at current eGFR. " +
					"Consider specialist consultation for risk-benefit evaluation.",
				Urgency: "URGENT",
			}
			// HFpEF with SGLT2i suppressed is a CRITICAL conflict —
			// SGLT2i is the ONLY proven disease-modifying therapy.
			if ckmStage == "4c" && hfType == "HFpEF" && g.MissingClass == "SGLT2i" {
				conflict.Urgency = "IMMEDIATE"
				conflict.ResolutionRecommendation = "CRITICAL: SGLT2i is the ONLY proven " +
					"disease-modifying therapy for HFpEF and is now contraindicated at current eGFR. " +
					"This patient has no available GDMT — urgent cardiology-nephrology co-management required."
				report.HasCriticalConflict = true
				report.HasSafetyBlock = true
			}
			report.RenalHFConflicts = append(report.RenalHFConflicts, conflict)
		}
	}

	return report
}
