package services

import "kb-26-metabolic-digital-twin/internal/models"

// ClassifyTemporal determines the temporal pattern of a deviation based on
// prior deviations for the same patient + vital sign + direction.
// priorDeviations should be from the last 7 days, same vital sign.
func ClassifyTemporal(
	current models.DeviationResult,
	priorDeviations []models.DeviationResult,
) string {
	// Count prior deviations in the same direction
	sameDirectionCount := 0
	for _, prior := range priorDeviations {
		if prior.Direction == current.Direction {
			sameDirectionCount++
		}
	}

	// PERSISTENCE: 5+ prior same-direction deviations (6+ total including current)
	if sameDirectionCount >= 5 {
		return string(models.TemporalPersistence)
	}

	// TREND: 2+ prior same-direction (3+ total including current)
	if sameDirectionCount >= 2 {
		return string(models.TemporalTrend)
	}

	// SPIKE: 0-1 prior same-direction
	return string(models.TemporalSpike)
}

// ModulateSeverity adjusts raw severity based on temporal classification.
// SPIKE downgrades by 1 (CRITICAL->HIGH, HIGH stays HIGH, MODERATE stays MODERATE).
// PERSISTENCE upgrades by 1 (MODERATE->HIGH, HIGH->CRITICAL, CRITICAL stays CRITICAL).
// TREND preserves raw severity.
func ModulateSeverity(rawSeverity, temporalClass string) string {
	switch temporalClass {
	case string(models.TemporalSpike):
		return deescalateSeverityOnce(rawSeverity)
	case string(models.TemporalPersistence):
		return escalateSeverityOnce(rawSeverity)
	default: // TREND
		return rawSeverity
	}
}

// escalateSeverityOnce raises severity by one level, capping at CRITICAL.
func escalateSeverityOnce(severity string) string {
	return escalateSeverity(severity, 1)
}

// deescalateSeverityOnce lowers severity by one level with MODERATE as floor.
func deescalateSeverityOnce(severity string) string {
	return deescalateSeverity(severity, 1, "MODERATE")
}
