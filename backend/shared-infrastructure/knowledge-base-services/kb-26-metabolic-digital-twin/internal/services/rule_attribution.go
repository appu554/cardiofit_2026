package services

import (
	"fmt"

	"github.com/google/uuid"
	"kb-26-metabolic-digital-twin/internal/models"
)

// AttributionInput carries everything the rule-based engine needs.
// Built from a ConsolidatedAlertRecord (kb-23 Task 3).
type AttributionInput struct {
	ConsolidatedRecordID uuid.UUID
	PatientID            string
	CohortID             string

	TreatmentStrategy    string
	OutcomeOccurred      *bool
	OutcomeType          string
	HorizonDays          int

	PreAlertRiskScore    float64
	PreAlertRiskTier     string
}

// ComputeAttribution produces a rule-based AttributionVerdict for one consolidated
// alert record. The counterfactual is the patient's own pre-alert risk score — no
// cohort mean, no propensity model. Sprint 2 replaces this function with IPW/DR
// estimators in KB-28; the returned struct stays identical.
func ComputeAttribution(in AttributionInput) models.AttributionVerdict {
	verdict := models.AttributionVerdict{
		ConsolidatedRecordID: in.ConsolidatedRecordID,
		PatientID:            in.PatientID,
		CohortID:             in.CohortID,
		CounterfactualRisk:   in.PreAlertRiskScore,
		PredictionWindowDays: in.HorizonDays,
		AttributionMethod:    "RULE_BASED",
		MethodVersion:        "sprint1-v1",
	}

	if in.OutcomeOccurred == nil {
		verdict.ClinicianLabel = string(models.LabelInconclusive)
		verdict.TechnicalLabel = "outcome_missing"
		verdict.Rationale = "Outcome status not available — attribution cannot be computed."
		return verdict
	}

	occurred := *in.OutcomeOccurred
	verdict.ObservedOutcome = occurred
	tier := in.PreAlertRiskTier
	ts := in.TreatmentStrategy

	switch {
	case isIntervention(ts) && !occurred && tier == "HIGH":
		verdict.ClinicianLabel = string(models.LabelPrevented)
		verdict.TechnicalLabel = "rule_prevented_high_risk_no_outcome"
		// RiskDifference = baseline − 0 (observed outcome did not occur, so observed
		// risk realization = 0). RiskReductionPct equals RiskDifference for this rule-based
		// case; Sprint 2 IPW/DR will decouple them via estimated Ŷ(0).
		verdict.RiskDifference = in.PreAlertRiskScore
		verdict.RiskReductionPct = in.PreAlertRiskScore
		verdict.Rationale = fmt.Sprintf(
			"High pre-alert risk (%.0f/100); intervention taken; outcome did not occur within %d-day window.",
			in.PreAlertRiskScore, in.HorizonDays)

	// ALREADY_ADDRESSED: alert fired but action was pre-existing. When outcome
	// occurs, the pre-existing action clearly failed — same causal story as
	// intervention failure. When outcome does NOT occur, causal ambiguity is too
	// high for rule-based attribution (was it the pre-existing action, or was
	// the patient never going to have the outcome?) — falls through to inconclusive.
	case (isIntervention(ts) || ts == "ALREADY_ADDRESSED") && occurred:
		verdict.ClinicianLabel = string(models.LabelOutcomeDespiteIntervention)
		verdict.TechnicalLabel = "rule_outcome_despite_intervention"
		verdict.RiskDifference = 0
		verdict.Rationale = fmt.Sprintf(
			"Intervention taken but outcome occurred within %d-day window.", in.HorizonDays)

	case isIntervention(ts) && !occurred && (tier == "MODERATE" || tier == "LOW"):
		verdict.ClinicianLabel = string(models.LabelNoEffectDetected)
		verdict.TechnicalLabel = "rule_low_baseline_no_attribution"
		verdict.RiskDifference = 0
		verdict.Rationale = fmt.Sprintf(
			"Pre-alert risk not high enough (%.0f/100) to credibly attribute non-occurrence to intervention.",
			in.PreAlertRiskScore)

	case (ts == "OVERRIDE_WITH_REASON" || ts == "NO_RESPONSE") && !occurred && tier == "HIGH":
		verdict.ClinicianLabel = string(models.LabelFragileEstimate)
		if ts == "NO_RESPONSE" {
			verdict.TechnicalLabel = "rule_noresponse_high_risk_no_outcome"
		} else {
			verdict.TechnicalLabel = "rule_override_high_risk_no_outcome"
		}
		// Halved: fragile estimate cannot support full-score attribution; /2 is a
		// conservative discount pending propensity adjustment in Sprint 2.
		verdict.RiskDifference = in.PreAlertRiskScore / 2
		verdict.Rationale = "High-risk alert overridden/unresponded; outcome did not occur but attribution is fragile without propensity adjustment."

	case (ts == "OVERRIDE_WITH_REASON" || ts == "NO_RESPONSE") && !occurred && (tier == "MODERATE" || tier == "LOW"):
		verdict.ClinicianLabel = string(models.LabelInconclusive)
		verdict.TechnicalLabel = "rule_override_low_risk_no_attribution"
		verdict.Rationale = fmt.Sprintf(
			"Alert overridden/unresponded at non-high risk tier (%.0f/100); rule-based attribution cannot establish signal.",
			in.PreAlertRiskScore)

	default:
		verdict.ClinicianLabel = string(models.LabelInconclusive)
		verdict.TechnicalLabel = "rule_no_matching_case"
		verdict.Rationale = "Insufficient data for rule-based attribution."
	}

	return verdict
}

func isIntervention(strategy string) bool {
	return strategy == "INTERVENTION_TAKEN"
}
