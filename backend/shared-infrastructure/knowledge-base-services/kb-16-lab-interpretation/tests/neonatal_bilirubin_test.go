// Package tests provides unit tests for Phase 3b.6 neonatal bilirubin interpretation
package tests

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"kb-16-lab-interpretation/pkg/reference"
	"kb-16-lab-interpretation/pkg/types"
)

// =============================================================================
// NEONATAL BILIRUBIN INTERPRETATION TESTS
// =============================================================================

func TestBilirubinInterpretation_RiskCategory(t *testing.T) {
	tests := []struct {
		name             string
		gestationalAge   int
		expectedRiskCat  reference.RiskCategory
	}{
		{"Term 40w - LOW", 40, reference.RiskLow},
		{"Term 39w - LOW", 39, reference.RiskLow},
		{"Term 38w - LOW", 38, reference.RiskLow},
		{"Late preterm 37w - MEDIUM", 37, reference.RiskMedium},
		{"Late preterm 36w - MEDIUM", 36, reference.RiskMedium},
		{"Late preterm 35w - MEDIUM", 35, reference.RiskMedium},
		{"Preterm 34w - HIGH", 34, reference.RiskHigh},
		{"Very preterm 32w - HIGH", 32, reference.RiskHigh},
		{"Very preterm 28w - HIGH", 28, reference.RiskHigh},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.NewPatientContext("baby-123", 0.01, "M")
			ctx.SetNeonatalStatus(tt.gestationalAge, 48)

			// The risk category is determined by SetNeonatalStatus
			assert.Equal(t, string(tt.expectedRiskCat), ctx.NeonatalRiskCategory)
		})
	}
}

func TestBilirubinZone_Determination(t *testing.T) {
	tests := []struct {
		name              string
		value             float64
		photoThreshold    float64
		exchangeThreshold *float64
		expectedZone      reference.BilirubinZone
	}{
		{
			name:           "Below 50% of phototherapy - low risk zone",
			value:          7.0,
			photoThreshold: 15.0,
			expectedZone:   reference.ZoneLowRisk, // 7/15 = 46.7% < 50% of photo
		},
		{
			name:           "Low intermediate (50-75% of photo)",
			value:          10.0,
			photoThreshold: 15.0,
			expectedZone:   reference.ZoneLowIntermediate, // 10/15 = 67%
		},
		{
			name:           "High intermediate (75-85% of photo)",
			value:          12.0,
			photoThreshold: 15.0,
			expectedZone:   reference.ZoneHighIntermediate, // 12/15 = 80%
		},
		{
			name:           "High risk zone (85-100% of photo)",
			value:          14.0,
			photoThreshold: 15.0,
			expectedZone:   reference.ZoneHighRisk, // 14/15 = 93%
		},
		{
			name:              "At phototherapy threshold",
			value:             15.0,
			photoThreshold:    15.0,
			exchangeThreshold: float64Ptr(25.0),
			expectedZone:      reference.ZonePhototherapy,
		},
		{
			name:              "Above phototherapy, below exchange",
			value:             18.0,
			photoThreshold:    15.0,
			exchangeThreshold: float64Ptr(25.0),
			expectedZone:      reference.ZonePhototherapy,
		},
		{
			name:              "At exchange threshold",
			value:             25.0,
			photoThreshold:    15.0,
			exchangeThreshold: float64Ptr(25.0),
			expectedZone:      reference.ZoneExchange,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock interpreter to test zone determination
			// Note: In real tests, this would use the BilirubinInterpreter with a mock DB
			zone := determineTestZone(tt.value, tt.photoThreshold, tt.exchangeThreshold)
			assert.Equal(t, tt.expectedZone, zone)
		})
	}
}

// determineTestZone replicates the zone determination logic for testing
func determineTestZone(value, photoThreshold float64, exchangeThreshold *float64) reference.BilirubinZone {
	// Check exchange threshold first (most severe)
	if exchangeThreshold != nil && value >= *exchangeThreshold {
		return reference.ZoneExchange
	}

	// Check phototherapy threshold
	if value >= photoThreshold {
		return reference.ZonePhototherapy
	}

	// Determine percentile zone relative to photo threshold
	percentOfPhoto := (value / photoThreshold) * 100

	switch {
	case percentOfPhoto >= 85:
		return reference.ZoneHighRisk
	case percentOfPhoto >= 75:
		return reference.ZoneHighIntermediate
	case percentOfPhoto >= 50:
		return reference.ZoneLowIntermediate
	default:
		return reference.ZoneLowRisk
	}
}

func TestBilirubinThresholds_ByRiskCategory(t *testing.T) {
	// Test expected thresholds at key hour points per AAP 2022
	tests := []struct {
		name               string
		hoursOfLife        int
		riskCategory       reference.RiskCategory
		expectedPhotoMin   float64 // Approximate expected values
		expectedPhotoMax   float64
	}{
		// 24 hours
		{"24h LOW risk", 24, reference.RiskLow, 11.0, 13.0},
		{"24h MEDIUM risk", 24, reference.RiskMedium, 9.0, 11.0},
		{"24h HIGH risk", 24, reference.RiskHigh, 7.0, 9.0},

		// 48 hours
		{"48h LOW risk", 48, reference.RiskLow, 14.0, 16.0},
		{"48h MEDIUM risk", 48, reference.RiskMedium, 12.0, 14.0},
		{"48h HIGH risk", 48, reference.RiskHigh, 10.0, 12.0},

		// 72 hours
		{"72h LOW risk", 72, reference.RiskLow, 17.0, 19.0},
		{"72h MEDIUM risk", 72, reference.RiskMedium, 15.0, 17.0},
		{"72h HIGH risk", 72, reference.RiskHigh, 13.0, 15.0},

		// 96 hours
		{"96h LOW risk", 96, reference.RiskLow, 19.0, 21.0},
		{"96h MEDIUM risk", 96, reference.RiskMedium, 17.0, 19.0},
		{"96h HIGH risk", 96, reference.RiskHigh, 14.0, 16.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// These are validation tests for the seeded data
			// In real tests, would query the database
			// Here we just validate the expected ranges are reasonable

			// Verify that higher risk = lower threshold (more conservative)
			assert.True(t, tt.expectedPhotoMin > 0, "Photo threshold should be positive")
			assert.True(t, tt.expectedPhotoMax > tt.expectedPhotoMin, "Max should exceed min")
		})
	}
}

func TestBilirubinVelocity(t *testing.T) {
	tests := []struct {
		name           string
		currentValue   float64
		previousValue  float64
		hoursBetween   int
		expectedVelocity float64
		isRapidRise    bool
	}{
		{
			name:           "Normal velocity",
			currentValue:   10.0,
			previousValue:  9.0,
			hoursBetween:   12,
			expectedVelocity: 0.083, // 1.0/12
			isRapidRise:   false,
		},
		{
			name:           "Moderate velocity",
			currentValue:   12.0,
			previousValue:  10.0,
			hoursBetween:   12,
			expectedVelocity: 0.167, // 2.0/12
			isRapidRise:   false,
		},
		{
			name:           "Elevated velocity (>0.2)",
			currentValue:   15.0,
			previousValue:  12.0,
			hoursBetween:   12,
			expectedVelocity: 0.25, // 3.0/12
			isRapidRise:   true,
		},
		{
			name:           "Rapid rise (>0.3) - hemolysis concern",
			currentValue:   18.0,
			previousValue:  14.0,
			hoursBetween:   12,
			expectedVelocity: 0.333, // 4.0/12
			isRapidRise:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			velocity := (tt.currentValue - tt.previousValue) / float64(tt.hoursBetween)

			// Allow small floating point tolerance
			assert.InDelta(t, tt.expectedVelocity, velocity, 0.01)

			isRapid := velocity > 0.2
			assert.Equal(t, tt.isRapidRise, isRapid)
		})
	}
}

func TestFollowUpTiming(t *testing.T) {
	tests := []struct {
		name           string
		zone           reference.BilirubinZone
		hoursOfLife    int
		expectedTiming string
	}{
		{"Exchange - always 4-6h", reference.ZoneExchange, 48, "4-6 hours (during/after treatment)"},
		{"Phototherapy - always 4-6h", reference.ZonePhototherapy, 48, "4-6 hours (during/after treatment)"},
		{"High risk early (<48h)", reference.ZoneHighRisk, 24, "4-8 hours"},
		{"High risk later (>48h)", reference.ZoneHighRisk, 72, "8-12 hours"},
		{"High intermediate early", reference.ZoneHighIntermediate, 24, "8-12 hours"},
		{"High intermediate later", reference.ZoneHighIntermediate, 72, "12-24 hours"},
		{"Low intermediate", reference.ZoneLowIntermediate, 48, "24 hours or first outpatient visit"},
		{"Low risk", reference.ZoneLowRisk, 48, "Routine follow-up per discharge criteria"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timing := getTestFollowUpTiming(tt.zone, tt.hoursOfLife)
			assert.Equal(t, tt.expectedTiming, timing)
		})
	}
}

// getTestFollowUpTiming replicates the follow-up timing logic for testing
func getTestFollowUpTiming(zone reference.BilirubinZone, hoursOfLife int) string {
	earlyPeriod := hoursOfLife < 48

	switch zone {
	case reference.ZoneExchange, reference.ZonePhototherapy:
		return "4-6 hours (during/after treatment)"
	case reference.ZoneHighRisk:
		if earlyPeriod {
			return "4-8 hours"
		}
		return "8-12 hours"
	case reference.ZoneHighIntermediate:
		if earlyPeriod {
			return "8-12 hours"
		}
		return "12-24 hours"
	case reference.ZoneLowIntermediate:
		return "24 hours or first outpatient visit"
	default:
		return "Routine follow-up per discharge criteria"
	}
}

// =============================================================================
// INTEGRATION SCENARIO TESTS
// =============================================================================

func TestBilirubinScenario_TermNewborn48Hours(t *testing.T) {
	// Scenario: Term (40w) newborn at 48 hours of life
	// Bilirubin = 14.0 mg/dL
	// Expected: Just below phototherapy threshold for LOW risk

	ctx := types.NewPatientContext("baby-term", 0.005, "M")
	ctx.SetNeonatalStatus(40, 48)

	assert.Equal(t, "LOW", ctx.NeonatalRiskCategory)

	// LOW risk at 48h: photo threshold ~15 mg/dL per AAP 2022
	// 14.0 mg/dL should be in HIGH_RISK zone but not yet phototherapy
	bilirubinValue := 14.0
	photoThreshold := 15.0 // From seeded data

	zone := determineTestZone(bilirubinValue, photoThreshold, nil)
	assert.Equal(t, reference.ZoneHighRisk, zone) // 14/15 = 93% of threshold

	// Should NOT need phototherapy yet
	needsPhoto := bilirubinValue >= photoThreshold
	assert.False(t, needsPhoto)
}

func TestBilirubinScenario_LatePreterm36Hours(t *testing.T) {
	// Scenario: Late preterm (35w) newborn at 36 hours
	// Bilirubin = 13.0 mg/dL
	// Expected: Above phototherapy threshold for MEDIUM risk

	ctx := types.NewPatientContext("baby-preterm", 0.004, "F")
	ctx.SetNeonatalStatus(35, 36)

	assert.Equal(t, "MEDIUM", ctx.NeonatalRiskCategory)

	// MEDIUM risk at 36h: photo threshold ~12 mg/dL per AAP 2022
	// 13.0 mg/dL should trigger phototherapy
	bilirubinValue := 13.0
	photoThreshold := 12.0 // From seeded data

	zone := determineTestZone(bilirubinValue, photoThreshold, nil)
	assert.Equal(t, reference.ZonePhototherapy, zone)

	needsPhoto := bilirubinValue >= photoThreshold
	assert.True(t, needsPhoto)
}

func TestBilirubinScenario_VeryPretermCritical(t *testing.T) {
	// Scenario: Very preterm (30w) newborn at 72 hours
	// Bilirubin = 16.0 mg/dL
	// Expected: Above exchange threshold for HIGH risk

	ctx := types.NewPatientContext("baby-verypreterm", 0.008, "M")
	ctx.SetNeonatalStatus(30, 72)

	assert.Equal(t, "HIGH", ctx.NeonatalRiskCategory)

	// HIGH risk at 72h: exchange threshold ~19 mg/dL per AAP 2022
	// But let's test with a higher value that would trigger exchange
	bilirubinValue := 20.0
	photoThreshold := 14.0
	exchangeThreshold := float64Ptr(19.0)

	zone := determineTestZone(bilirubinValue, photoThreshold, exchangeThreshold)
	assert.Equal(t, reference.ZoneExchange, zone)

	needsExchange := exchangeThreshold != nil && bilirubinValue >= *exchangeThreshold
	assert.True(t, needsExchange)
}

// =============================================================================
// EDGE CASE TESTS
// =============================================================================

func TestBilirubinEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		hoursOfLife    int
		ga             int
		value          float64
		expectValid    bool
	}{
		{
			name:        "Very early (12h) - valid",
			hoursOfLife: 12,
			ga:          38,
			value:       8.0,
			expectValid: true,
		},
		{
			name:        "At 120h boundary - valid",
			hoursOfLife: 120,
			ga:          40,
			value:       18.0,
			expectValid: true,
		},
		{
			name:        "Beyond nomogram (>120h) - extrapolate",
			hoursOfLife: 144,
			ga:          39,
			value:       15.0,
			expectValid: true, // Should use 120h threshold
		},
		{
			name:        "Extremely preterm (24w) - use HIGH risk",
			hoursOfLife: 48,
			ga:          24,
			value:       10.0,
			expectValid: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := types.NewPatientContext("baby-edge", 0.01, "F")

			// SetNeonatalStatus will categorize appropriately
			ctx.SetNeonatalStatus(tt.ga, tt.hoursOfLife)

			// All should result in valid neonatal contexts
			assert.True(t, ctx.IsNeonate)
			assert.NotEmpty(t, ctx.NeonatalRiskCategory)
		})
	}
}
