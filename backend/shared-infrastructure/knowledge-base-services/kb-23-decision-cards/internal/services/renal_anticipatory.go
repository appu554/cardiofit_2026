package services

import (
	"fmt"
	"math"
)

// ---------------------------------------------------------------------------
// AnticipatoryAlert — proactive threshold crossing warning
// ---------------------------------------------------------------------------

// AnticipatoryAlert warns that a medication will need action if the eGFR
// trajectory continues its current trend.
type AnticipatoryAlert struct {
	DrugClass         string  `json:"drug_class"`
	ThresholdType     string  `json:"threshold_type"`      // CONTRAINDICATION | DOSE_REDUCE | EFFICACY_CLIFF
	ThresholdValue    float64 `json:"threshold_value"`
	MonthsToThreshold float64 `json:"months_to_threshold"`
	RecommendedAction string  `json:"recommended_action"`
	SourceGuideline   string  `json:"source_guideline"`
}

// ---------------------------------------------------------------------------
// projectTimeToThreshold — local copy (cross-module import not possible)
// ---------------------------------------------------------------------------

// projectTimeToThreshold returns the number of months until currentEGFR
// reaches threshold given slopePerYear (mL/min/1.73m²/year).
// Returns nil if slope is non-negative (improving/stable) or already below threshold.
func projectTimeToThreshold(currentEGFR, slopePerYear, threshold float64) *float64 {
	if slopePerYear >= 0 {
		return nil
	}
	if currentEGFR <= threshold {
		return nil
	}
	gap := currentEGFR - threshold
	years := gap / math.Abs(slopePerYear)
	months := years * 12.0
	return &months
}

// ---------------------------------------------------------------------------
// FindApproachingThresholds — anticipatory alert finder
// ---------------------------------------------------------------------------

// FindApproachingThresholds checks whether the eGFR trajectory will cross
// clinically significant thresholds for each active medication within the
// rule's lookahead horizon (AnticipateMonths, default 6).
func FindApproachingThresholds(
	formulary *RenalFormulary,
	currentEGFR, slopePerYear float64,
	meds []ActiveMedication,
) []AnticipatoryAlert {
	var alerts []AnticipatoryAlert

	for _, med := range meds {
		rule := formulary.GetRule(med.DrugClass)
		if rule == nil {
			continue
		}

		horizon := float64(rule.AnticipateMonths)
		if horizon <= 0 {
			horizon = 6.0 // default lookahead
		}

		// Check each threshold type in severity order.
		type thresholdCheck struct {
			name      string
			value     float64
			action    string
		}

		checks := []thresholdCheck{}

		if rule.ContraindicatedBelow > 0 {
			checks = append(checks, thresholdCheck{
				name:  "CONTRAINDICATION",
				value: rule.ContraindicatedBelow,
				action: fmt.Sprintf("plan discontinuation of %s; consider %s",
					med.DrugClass, rule.SubstituteClass),
			})
		}
		if rule.DoseReduceBelow > 0 {
			checks = append(checks, thresholdCheck{
				name:  "DOSE_REDUCE",
				value: rule.DoseReduceBelow,
				action: fmt.Sprintf("prepare dose reduction for %s", med.DrugClass),
			})
		}
		if rule.EfficacyCliffBelow > 0 {
			checks = append(checks, thresholdCheck{
				name:  "EFFICACY_CLIFF",
				value: rule.EfficacyCliffBelow,
				action: fmt.Sprintf("reduced efficacy approaching for %s; consider switch to %s",
					med.DrugClass, rule.SubstituteClass),
			})
		}

		for _, chk := range checks {
			months := projectTimeToThreshold(currentEGFR, slopePerYear, chk.value)
			if months == nil {
				continue
			}
			if *months <= horizon {
				alerts = append(alerts, AnticipatoryAlert{
					DrugClass:         med.DrugClass,
					ThresholdType:     chk.name,
					ThresholdValue:    chk.value,
					MonthsToThreshold: *months,
					RecommendedAction: chk.action,
					SourceGuideline:   rule.SourceGuideline,
				})
			}
		}
	}

	return alerts
}
