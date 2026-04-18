package services

import (
	"testing"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/stretchr/testify/assert"
)

func TestTemporal_FirstDeviation_Spike(t *testing.T) {
	// No prior deviations → SPIKE
	current := models.DeviationResult{Direction: "BELOW_BASELINE", ClinicalSignificance: "HIGH"}
	priors := []models.DeviationResult{} // empty

	result := ClassifyTemporal(current, priors)
	assert.Equal(t, "SPIKE", result)
}

func TestTemporal_ThreeConsecutive_Trend(t *testing.T) {
	// 2 prior deviations in same direction → TREND (current is 3rd)
	current := models.DeviationResult{Direction: "BELOW_BASELINE", ClinicalSignificance: "HIGH"}
	priors := []models.DeviationResult{
		{Direction: "BELOW_BASELINE", ClinicalSignificance: "MODERATE"},
		{Direction: "BELOW_BASELINE", ClinicalSignificance: "HIGH"},
	}

	result := ClassifyTemporal(current, priors)
	assert.Equal(t, "TREND", result)
}

func TestTemporal_Sustained24h_Persistence(t *testing.T) {
	// 5+ prior deviations in same direction → PERSISTENCE
	current := models.DeviationResult{Direction: "BELOW_BASELINE", ClinicalSignificance: "HIGH"}
	priors := make([]models.DeviationResult, 5)
	for i := range priors {
		priors[i] = models.DeviationResult{Direction: "BELOW_BASELINE", ClinicalSignificance: "MODERATE"}
	}

	result := ClassifyTemporal(current, priors)
	assert.Equal(t, "PERSISTENCE", result)
}

func TestTemporal_SeverityModulation(t *testing.T) {
	// SPIKE at CRITICAL → effective HIGH (downgrade 1)
	eff1 := ModulateSeverity("CRITICAL", "SPIKE")
	assert.Equal(t, "HIGH", eff1)

	// PERSISTENCE at MODERATE → effective HIGH (upgrade 1)
	eff2 := ModulateSeverity("MODERATE", "PERSISTENCE")
	assert.Equal(t, "HIGH", eff2)

	// TREND at HIGH → effective HIGH (no change)
	eff3 := ModulateSeverity("HIGH", "TREND")
	assert.Equal(t, "HIGH", eff3)

	// PERSISTENCE at CRITICAL → stays CRITICAL (already max)
	eff4 := ModulateSeverity("CRITICAL", "PERSISTENCE")
	assert.Equal(t, "CRITICAL", eff4)

	// SPIKE at MODERATE → effective MODERATE (MODERATE is the floor — can't go below)
	eff5 := ModulateSeverity("MODERATE", "SPIKE")
	assert.Equal(t, "MODERATE", eff5) // MODERATE is the floor — can't go below
}
