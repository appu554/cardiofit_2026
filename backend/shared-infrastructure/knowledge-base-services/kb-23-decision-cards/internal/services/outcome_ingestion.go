package services

import (
	"fmt"
	"time"

	"kb-23-decision-cards/internal/models"
)

// ReconcileOutcomes takes one or more OutcomeRecords for the same (patient, outcome_type)
// and returns a single authoritative record with Reconciliation set to RESOLVED,
// CONFLICTED, or PENDING. Source agreement on OutcomeOccurred resolves; disagreement
// conflicts; insufficient sources remain pending.
func ReconcileOutcomes(records []models.OutcomeRecord, tolerance time.Duration, minSources int) (models.OutcomeRecord, error) {
	if len(records) == 0 {
		return models.OutcomeRecord{}, fmt.Errorf("no records to reconcile")
	}
	if len(records) < minSources {
		r := records[0]
		r.Reconciliation = string(models.ReconciliationPending)
		return r, nil
	}

	firstOccurred := records[0].OutcomeOccurred
	allAgree := true
	for _, r := range records[1:] {
		if r.OutcomeOccurred != firstOccurred {
			allAgree = false
			break
		}
	}

	result := records[0]
	if !allAgree {
		result.Reconciliation = string(models.ReconciliationConflicted)
		result.Notes = fmt.Sprintf("conflict across %d sources", len(records))
		return result, nil
	}

	// All agree on occurrence — check timestamp agreement within tolerance.
	if firstOccurred && len(records) > 1 {
		for _, r := range records[1:] {
			if r.OccurredAt != nil && result.OccurredAt != nil {
				diff := r.OccurredAt.Sub(*result.OccurredAt)
				if diff < -tolerance || diff > tolerance {
					result.Reconciliation = string(models.ReconciliationConflicted)
					result.Notes = fmt.Sprintf("timestamp disagreement beyond %s", tolerance)
					return result, nil
				}
			}
		}
	}

	result.Reconciliation = string(models.ReconciliationResolved)
	return result, nil
}
