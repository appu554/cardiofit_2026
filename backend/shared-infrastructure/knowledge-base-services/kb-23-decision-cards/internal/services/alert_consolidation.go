package services

import (
	"fmt"
	"strings"

	"kb-23-decision-cards/internal/models"
)

// BuildConsolidatedRecord joins a DetectionLifecycle (Gap 19), an OutcomeRecord (Gap 21 Task 1),
// and the pre-alert PredictedRisk snapshot (Gap 20) into a single TTE-ready record.
// Time-zero is anchored at DetectedAt (T0) — the moment the alert fired is the
// earliest plausible point where the alert could have changed clinician behavior.
// Anchoring later (T1/T2/T3) would introduce immortal-time bias: the patient
// must survive long enough to be delivered/acknowledged/actioned, selecting out
// the very outcomes the alert should be credited for.
func BuildConsolidatedRecord(
	lc models.DetectionLifecycle,
	outcome *models.OutcomeRecord,
	preAlertRiskScore float64,
	preAlertRiskTier string,
	predictionModelID string,
	horizonDays int,
) (models.ConsolidatedAlertRecord, error) {
	strategy := classifyTreatmentStrategy(lc)
	overrideReason := ""
	if strategy == models.TreatmentOverrideReason {
		overrideReason = extractOverrideReason(lc.ActionDetail)
		if overrideReason == "already_addressed" {
			strategy = models.TreatmentAlreadyAddressed
		}
	}

	record := models.ConsolidatedAlertRecord{
		LifecycleID:       lc.ID,
		PatientID:         lc.PatientID,
		CohortID:          lc.CohortID,
		PreAlertRiskScore: preAlertRiskScore,
		PreAlertRiskTier:  preAlertRiskTier,
		PredictionModelID: predictionModelID,
		DetectedAt:        lc.DetectedAt,
		DeliveredAt:       lc.DeliveredAt,
		AcknowledgedAt:    lc.AcknowledgedAt,
		ActionedAt:        lc.ActionedAt,
		ResolvedAt:        lc.ResolvedAt,
		TimeZero:          lc.DetectedAt,
		TreatmentStrategy: string(strategy),
		ActionType:        lc.ActionType,
		OverrideReason:    overrideReason,
		HorizonDays:       horizonDays,
	}

	if outcome != nil {
		if outcome.Reconciliation != string(models.ReconciliationResolved) &&
			outcome.Reconciliation != string(models.ReconciliationHorizonExp) {
			return record, fmt.Errorf("outcome record not resolved: %s", outcome.Reconciliation)
		}
		record.OutcomeRecordID = &outcome.ID
		occurred := outcome.OutcomeOccurred
		record.OutcomeOccurred = &occurred
		record.OutcomeType = outcome.OutcomeType
	}

	return record, nil
}

func classifyTreatmentStrategy(lc models.DetectionLifecycle) models.TreatmentStrategy {
	if lc.ActionType == "OVERRIDE" {
		return models.TreatmentOverrideReason
	}
	if lc.ActionedAt != nil && lc.ActionType != "" {
		return models.TreatmentInterventionTaken
	}
	return models.TreatmentNoResponse
}

func extractOverrideReason(detail string) string {
	// ActionDetail format (from Gap 18 worklist): "reason=<override_code>[;...]"
	for _, part := range strings.Split(detail, ";") {
		part = strings.TrimSpace(part)
		if strings.HasPrefix(part, "reason=") {
			return strings.TrimPrefix(part, "reason=")
		}
	}
	return ""
}
