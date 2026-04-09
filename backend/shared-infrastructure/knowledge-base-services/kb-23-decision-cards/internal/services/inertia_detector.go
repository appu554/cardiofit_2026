package services

import (
	"time"

	"kb-23-decision-cards/internal/models"
)

// ---------------------------------------------------------------------------
// DomainInertiaInput — input data for a single clinical domain evaluation
// ---------------------------------------------------------------------------

// DomainInertiaInput captures the current state of a patient in one clinical
// domain, providing the evidence needed for therapeutic inertia detection.
type DomainInertiaInput struct {
	AtTarget            bool
	CurrentValue        float64
	TargetValue         float64
	DaysUncontrolled    int
	ConsecutiveReadings int
	DataSource          string     // "HBA1C", "CGM_TIR", "HOME_BP"
	LastIntervention    *time.Time
	CurrentMeds         []string
	AtMaxDose           bool
}

// InertiaDetectorInput aggregates domain inputs for a full patient evaluation.
type InertiaDetectorInput struct {
	PatientID   string
	Glycaemic   *DomainInertiaInput
	Hemodynamic *DomainInertiaInput
	Renal       *DomainInertiaInput
}

// ---------------------------------------------------------------------------
// Constants — thresholds derived from Khunti et al. clinical evidence
// ---------------------------------------------------------------------------

const (
	// gracePeriodDays is the minimum time after an intervention before
	// inertia can be declared (6 weeks titration window).
	gracePeriodDays = 42

	// hba1cMinDays is the minimum uncontrolled duration for HbA1c-based
	// inertia detection (12 weeks — one HbA1c cycle).
	hba1cMinDays = 84

	// cgmMinDays is the minimum uncontrolled duration for CGM-based
	// inertia detection (14 days — AGP reporting period).
	cgmMinDays = 14

	// bpMinDays is the minimum uncontrolled duration for blood pressure
	// inertia detection (4 weeks).
	bpMinDays = 28

	// Severity bracket thresholds in days. Approximate week boundaries
	// rounded to clinically meaningful day counts to avoid truncation
	// artefacts at boundary cases (e.g. 180 days ≈ 26 weeks).
	mildDays     = 84  // ~12 weeks
	moderateDays = 180 // ~26 weeks
	severeDays   = 364 // ~52 weeks
	criticalDays = 546 // ~78 weeks
)

// ---------------------------------------------------------------------------
// DetectInertia — top-level entry point
// ---------------------------------------------------------------------------

// DetectInertia evaluates all provided clinical domains for therapeutic
// inertia, detects dual-domain patterns, and computes overall urgency.
func DetectInertia(input InertiaDetectorInput) models.PatientInertiaReport {
	now := time.Now()
	report := models.PatientInertiaReport{
		PatientID:   input.PatientID,
		EvaluatedAt: now,
	}

	var verdicts []models.InertiaVerdict
	var detectedDomains []models.InertiaDomain

	// Evaluate each domain if provided.
	if input.Glycaemic != nil {
		if v := evaluateDomainInertia(models.DomainGlycaemic, input.Glycaemic); v != nil {
			verdicts = append(verdicts, *v)
			if v.Detected {
				detectedDomains = append(detectedDomains, v.Domain)
			}
		}
	}
	if input.Hemodynamic != nil {
		if v := evaluateDomainInertia(models.DomainHemodynamic, input.Hemodynamic); v != nil {
			verdicts = append(verdicts, *v)
			if v.Detected {
				detectedDomains = append(detectedDomains, v.Domain)
			}
		}
	}
	if input.Renal != nil {
		if v := evaluateDomainInertia(models.DomainRenal, input.Renal); v != nil {
			verdicts = append(verdicts, *v)
			if v.Detected {
				detectedDomains = append(detectedDomains, v.Domain)
			}
		}
	}

	// Dual-domain detection.
	if len(detectedDomains) >= 2 {
		report.HasDualDomainInertia = true
		dualVerdict := models.InertiaVerdict{
			Domain:   detectedDomains[0], // primary domain
			Pattern:  models.PatternDualDomainInertia,
			Detected: true,
			Severity: models.SeverityCritical,
		}
		verdicts = append(verdicts, dualVerdict)
	}

	report.Verdicts = verdicts

	// Determine HasAnyInertia and MostSevere.
	var mostSevere *models.InertiaVerdict
	for i := range verdicts {
		if verdicts[i].Detected {
			report.HasAnyInertia = true
			if mostSevere == nil || severityRank(verdicts[i].Severity) > severityRank(mostSevere.Severity) {
				mostSevere = &verdicts[i]
			}
		}
	}
	report.MostSevere = mostSevere

	// Overall urgency mapping.
	report.OverallUrgency = determineOverallUrgency(report)

	return report
}

// ---------------------------------------------------------------------------
// evaluateDomainInertia — single domain evaluation
// ---------------------------------------------------------------------------

func evaluateDomainInertia(domain models.InertiaDomain, input *DomainInertiaInput) *models.InertiaVerdict {
	// If at target, no inertia.
	if input.AtTarget {
		return nil
	}

	// Determine minimum duration and pattern based on data source.
	minDays, pattern := domainMinDaysAndPattern(domain, input.DataSource)

	// Check minimum uncontrolled duration.
	if input.DaysUncontrolled < minDays {
		return nil
	}

	// Grace period: if intervention happened within grace period, suppress.
	if input.LastIntervention != nil {
		daysSinceIntervention := int(time.Since(*input.LastIntervention).Hours() / 24)
		if daysSinceIntervention < gracePeriodDays {
			return nil
		}
	}

	// Compute inertia duration: days uncontrolled minus grace period (if
	// there was an intervention within the uncontrolled window).
	inertiaDays := input.DaysUncontrolled
	if input.LastIntervention != nil {
		daysSinceIntervention := int(time.Since(*input.LastIntervention).Hours() / 24)
		if daysSinceIntervention < input.DaysUncontrolled {
			// Intervention was within the uncontrolled window; inertia
			// only counts from when the grace period ended.
			inertiaDays = daysSinceIntervention - gracePeriodDays
			if inertiaDays < 0 {
				inertiaDays = 0
			}
		}
	}

	severity := classifyInertiaSeverity(inertiaDays)

	verdict := &models.InertiaVerdict{
		Domain:              domain,
		Pattern:             pattern,
		Detected:            true,
		InertiaDurationDays: inertiaDays,
		TargetValue:         input.TargetValue,
		CurrentValue:        input.CurrentValue,
		ConsecutiveReadings: input.ConsecutiveReadings,
		DataSource:          input.DataSource,
		CurrentMedications:  input.CurrentMeds,
		AtMaxDose:           input.AtMaxDose,
		Severity:            severity,
	}

	if input.LastIntervention != nil {
		verdict.LastInterventionDate = input.LastIntervention
		verdict.DaysSinceIntervention = int(time.Since(*input.LastIntervention).Hours() / 24)
	}

	// FirstExceedanceDate approximation from DaysUncontrolled.
	verdict.FirstExceedanceDate = time.Now().AddDate(0, 0, -input.DaysUncontrolled)

	return verdict
}

// ---------------------------------------------------------------------------
// domainMinDaysAndPattern — returns minimum days and pattern for a domain
// ---------------------------------------------------------------------------

func domainMinDaysAndPattern(domain models.InertiaDomain, dataSource string) (int, models.InertiaPattern) {
	switch domain {
	case models.DomainGlycaemic:
		switch dataSource {
		case "CGM_TIR":
			return cgmMinDays, models.PatternCGMInertia
		default: // HBA1C and others
			return hba1cMinDays, models.PatternHbA1cInertia
		}
	case models.DomainHemodynamic:
		return bpMinDays, models.PatternBPInertia
	case models.DomainRenal:
		return hba1cMinDays, models.PatternRenalProgressionInertia
	default:
		return hba1cMinDays, models.PatternHbA1cInertia
	}
}

// ---------------------------------------------------------------------------
// classifyInertiaSeverity — Khunti-bracket weeks-based classification
// ---------------------------------------------------------------------------

func classifyInertiaSeverity(days int) models.InertiaSeverity {
	switch {
	case days >= criticalDays:
		return models.SeverityCritical
	case days >= severeDays:
		return models.SeveritySevere
	case days >= moderateDays:
		return models.SeverityModerate
	default:
		return models.SeverityMild
	}
}

// ---------------------------------------------------------------------------
// determineOverallUrgency — maps inertia findings to urgency level
// ---------------------------------------------------------------------------

func determineOverallUrgency(report models.PatientInertiaReport) string {
	if !report.HasAnyInertia {
		return UrgencyScheduled
	}
	if report.HasDualDomainInertia {
		return UrgencyImmediate
	}
	if report.MostSevere != nil {
		switch report.MostSevere.Severity {
		case models.SeverityCritical:
			return UrgencyImmediate
		case models.SeveritySevere, models.SeverityModerate:
			return UrgencyUrgent
		case models.SeverityMild:
			return UrgencyRoutine
		}
	}
	return UrgencyScheduled
}

// ---------------------------------------------------------------------------
// severityRank — numeric rank for severity comparison
// ---------------------------------------------------------------------------

func severityRank(s models.InertiaSeverity) int {
	switch s {
	case models.SeverityCritical:
		return 4
	case models.SeveritySevere:
		return 3
	case models.SeverityModerate:
		return 2
	case models.SeverityMild:
		return 1
	default:
		return 0
	}
}
