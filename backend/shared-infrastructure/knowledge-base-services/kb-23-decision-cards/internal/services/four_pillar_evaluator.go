package services

import (
	"fmt"

	"kb-23-decision-cards/internal/models"
	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

// ---------------------------------------------------------------------------
// PillarStatus — evaluation outcome for a single pillar
// ---------------------------------------------------------------------------

// PillarStatus represents the gap classification for a care pillar.
type PillarStatus string

const (
	PillarOnTrack   PillarStatus = "ON_TRACK"
	PillarGap       PillarStatus = "GAP"
	PillarUrgentGap PillarStatus = "URGENT_GAP"
)

// ---------------------------------------------------------------------------
// Input / Output types
// ---------------------------------------------------------------------------

// MedicationPillarInput captures the medication adherence state for evaluation.
type MedicationPillarInput struct {
	OnGuidelineMeds bool    `json:"on_guideline_meds"`
	AdherencePct    float64 `json:"adherence_pct"`
}

// MonitoringPillarInput captures monitoring freshness for evaluation.
type MonitoringPillarInput struct {
	StaleEGFR *StaleEGFRResult `json:"stale_egfr,omitempty"`
}

// LifestylePillarInput captures lifestyle adherence for evaluation.
type LifestylePillarInput struct {
	AdherencePct float64 `json:"adherence_pct"`
}

// EducationPillarInput captures patient education completion.
type EducationPillarInput struct {
	Complete bool `json:"complete"`
}

// FourPillarInput is the combined input for four-pillar evaluation.
type FourPillarInput struct {
	PatientID       string                         `json:"patient_id"`
	DualDomainState string                         `json:"dual_domain_state"`
	Medication      MedicationPillarInput          `json:"medication"`
	Monitoring      MonitoringPillarInput          `json:"monitoring"`
	Lifestyle       LifestylePillarInput           `json:"lifestyle"`
	Education       EducationPillarInput           `json:"education"`
	RenalGating          *models.PatientGatingReport     `json:"renal_gating,omitempty"`
	InertiaReport        *models.PatientInertiaReport    `json:"inertia_report,omitempty"`
	BPContext            *models.BPContextClassification `json:"bp_context,omitempty"`
	DecomposedTrajectory *dtModels.DecomposedTrajectory  `json:"decomposed_trajectory,omitempty"`
}

// PillarResult is the evaluation output for a single pillar.
type PillarResult struct {
	Pillar  string       `json:"pillar"`
	Status  PillarStatus `json:"status"`
	Reason  string       `json:"reason"`
	Actions []string     `json:"actions,omitempty"`
}

// FourPillarResult is the aggregate evaluation across all pillars.
type FourPillarResult struct {
	Pillars     []PillarResult `json:"pillars"`
	OverallGap  bool           `json:"overall_gap"`
	UrgentCount int            `json:"urgent_count"`
}

// ---------------------------------------------------------------------------
// EvaluateFourPillars — core evaluator
// ---------------------------------------------------------------------------

// EvaluateFourPillars assesses each of the four care pillars and returns
// an aggregate result with gap counts.
//
// Pillar logic:
//   - Medication: renal contraindication → URGENT_GAP; else adherence/guideline checks
//   - Monitoring: stale eGFR CRITICAL → URGENT_GAP; WARNING → GAP
//   - Lifestyle: adherence < 50% → GAP
//   - Education: incomplete → GAP
func EvaluateFourPillars(input FourPillarInput) FourPillarResult {
	result := FourPillarResult{}

	// --- Medication pillar ---
	medPillar := evaluateMedicationPillar(input)
	result.Pillars = append(result.Pillars, medPillar)

	// --- Monitoring pillar ---
	monPillar := evaluateMonitoringPillar(input)
	result.Pillars = append(result.Pillars, monPillar)

	// --- Lifestyle pillar ---
	lifePillar := evaluateLifestylePillar(input)
	result.Pillars = append(result.Pillars, lifePillar)

	// --- Education pillar ---
	eduPillar := evaluateEducationPillar(input)
	result.Pillars = append(result.Pillars, eduPillar)

	// Aggregate
	for _, p := range result.Pillars {
		if p.Status == PillarUrgentGap {
			result.UrgentCount++
			result.OverallGap = true
		} else if p.Status == PillarGap {
			result.OverallGap = true
		}
	}

	return result
}

// ---------------------------------------------------------------------------
// Per-pillar evaluators
// ---------------------------------------------------------------------------

func evaluateMedicationPillar(input FourPillarInput) PillarResult {
	p := PillarResult{Pillar: "MEDICATION"}

	// Renal contraindication overrides everything
	if input.RenalGating != nil && input.RenalGating.HasContraindicated {
		p.Status = PillarUrgentGap
		p.Reason = "renal contraindication detected — medication safety block"
		p.Actions = []string{
			"review contraindicated medications immediately",
			"consult nephrology if eGFR rapidly declining",
		}
		return p
	}

	// Therapeutic inertia escalation
	if input.InertiaReport != nil {
		if input.InertiaReport.HasDualDomainInertia {
			p.Status = PillarUrgentGap
			p.Reason = "dual-domain therapeutic inertia — concordant uncontrolled status"
			p.Actions = []string{
				"escalate medication review for both glycaemic and hemodynamic domains",
				"consider combination intensification strategy",
			}
			return p
		}
		if input.InertiaReport.MostSevere != nil {
			sev := input.InertiaReport.MostSevere.Severity
			if sev == models.SeveritySevere || sev == models.SeverityCritical {
				p.Status = PillarUrgentGap
				p.Reason = "severe/critical therapeutic inertia in " + string(input.InertiaReport.MostSevere.Domain)
				p.Actions = []string{
					"urgent medication intensification required",
					"review barriers to treatment escalation",
				}
				return p
			}
		}
		// Mild/moderate inertia downgrades ON_TRACK to GAP (checked after
		// guideline and adherence evaluation below to avoid masking those).
	}

	// Masked hypertension: clinic BP looks fine but home BP is elevated.
	// White-coat HTN: opposite — avoid overtreatment based on clinic-only readings.
	if input.BPContext != nil {
		switch input.BPContext.Phenotype {
		case models.PhenotypeMaskedHTN, models.PhenotypeMaskedUncontrolled:
			p.Status = PillarGap
			if input.BPContext.DiabetesAmplification || input.BPContext.MorningSurgeCompound {
				p.Status = PillarUrgentGap
			}
			p.Reason = "masked hypertension — home BP elevated despite normal clinic readings"
			p.Actions = []string{
				"treat based on HOME BP targets, not clinic readings",
			}
			if input.BPContext.DiabetesAmplification {
				p.Actions = append(p.Actions,
					"DM + masked HTN: 3.2x target organ damage risk — immediate action")
			}
			return p

		case models.PhenotypeWhiteCoatHTN, models.PhenotypeWhiteCoatUncontrolled:
			p.Status = PillarGap
			p.Reason = "white-coat effect — do not intensify based on clinic BP alone"
			p.Actions = []string{
				"continue home monitoring; lifestyle intervention appropriate",
			}
			return p
		}
	}

	// Guideline adherence check
	if !input.Medication.OnGuidelineMeds {
		p.Status = PillarGap
		p.Reason = "patient not on guideline-recommended medications"
		p.Actions = []string{"review medication regimen against current guidelines"}
		return p
	}

	// Adherence check
	if input.Medication.AdherencePct < 80 {
		p.Status = PillarGap
		p.Reason = "medication adherence below 80%"
		p.Actions = []string{"assess barriers to adherence", "consider simplifying regimen"}
		return p
	}

	// Mild/moderate inertia downgrades ON_TRACK to GAP
	if input.InertiaReport != nil && input.InertiaReport.HasAnyInertia {
		p.Status = PillarGap
		p.Reason = "therapeutic inertia detected despite adequate adherence"
		p.Actions = []string{"review treatment targets and consider intensification"}
		return p
	}

	p.Status = PillarOnTrack
	p.Reason = "on guideline medications with adequate adherence"
	return p
}

func evaluateMonitoringPillar(input FourPillarInput) PillarResult {
	p := PillarResult{Pillar: "MONITORING"}

	if input.Monitoring.StaleEGFR != nil && input.Monitoring.StaleEGFR.IsStale {
		if input.Monitoring.StaleEGFR.Severity == "CRITICAL" {
			p.Status = PillarUrgentGap
			p.Reason = "eGFR critically overdue — data insufficient for safe prescribing"
			p.Actions = []string{"order urgent renal function panel"}
			return p
		}
		p.Status = PillarGap
		p.Reason = "eGFR measurement overdue"
		p.Actions = []string{"schedule renal function panel"}
		return p
	}

	// Trajectory-based monitoring recommendations.
	// Order mirrors trajectory card priority: concordant > discordant > behavioral lead.
	if input.DecomposedTrajectory != nil {
		dt := input.DecomposedTrajectory
		if dt.ConcordantDeterioration {
			p.Status = PillarUrgentGap
			p.Reason = fmt.Sprintf("concordant deterioration: %d domains declining — increase monitoring frequency", dt.DomainsDeteriorating)
			p.Actions = []string{"increase monitoring frequency across all domains"}
			return p
		}
		if dt.HasDiscordantTrend {
			p.Status = PillarGap
			p.Reason = "discordant trajectory: domains moving in opposite directions"
			p.Actions = []string{"investigate cross-domain medication effects"}
			return p
		}
		for _, lead := range dt.LeadingIndicators {
			if lead.LeadingDomain == dtModels.DomainBehavioral {
				p.Status = PillarGap
				p.Reason = "behavioral leading indicator: engagement collapse detected"
				p.Actions = []string{"clinical outreach recommended before clinical domains deteriorate further"}
				return p
			}
		}
	}

	p.Status = PillarOnTrack
	p.Reason = "monitoring up to date"
	return p
}

func evaluateLifestylePillar(input FourPillarInput) PillarResult {
	p := PillarResult{Pillar: "LIFESTYLE"}

	if input.Lifestyle.AdherencePct < 50 {
		p.Status = PillarGap
		p.Reason = "lifestyle adherence below 50%"
		p.Actions = []string{"reinforce lifestyle counselling", "consider behavioural support referral"}
		return p
	}

	p.Status = PillarOnTrack
	p.Reason = "lifestyle adherence adequate"
	return p
}

func evaluateEducationPillar(input FourPillarInput) PillarResult {
	p := PillarResult{Pillar: "EDUCATION"}

	if !input.Education.Complete {
		p.Status = PillarGap
		p.Reason = "patient education modules incomplete"
		p.Actions = []string{"assign outstanding education modules"}
		return p
	}

	p.Status = PillarOnTrack
	p.Reason = "education modules completed"
	return p
}
