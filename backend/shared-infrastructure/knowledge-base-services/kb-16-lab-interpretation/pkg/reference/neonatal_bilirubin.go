// Package reference provides neonatal bilirubin interpretation per AAP 2022 guidelines
// Phase 3b.6: Bhutani Nomogram implementation with hour-of-life interpolation
package reference

import (
	"context"
	"fmt"
	"math"

	"gorm.io/gorm"

	"kb-16-lab-interpretation/pkg/types"
)

// BilirubinInterpreter handles neonatal bilirubin interpretation using AAP 2022 Bhutani nomogram
type BilirubinInterpreter struct {
	db *gorm.DB
}

// NewBilirubinInterpreter creates a new BilirubinInterpreter instance
func NewBilirubinInterpreter(db *gorm.DB) *BilirubinInterpreter {
	return &BilirubinInterpreter{db: db}
}

// BilirubinZone represents risk zones on the Bhutani nomogram
type BilirubinZone string

const (
	ZoneLowRisk           BilirubinZone = "LOW_RISK"
	ZoneLowIntermediate   BilirubinZone = "LOW_INTERMEDIATE"
	ZoneHighIntermediate  BilirubinZone = "HIGH_INTERMEDIATE"
	ZoneHighRisk          BilirubinZone = "HIGH_RISK"
	ZonePhototherapy      BilirubinZone = "PHOTOTHERAPY"
	ZoneExchange          BilirubinZone = "EXCHANGE_TRANSFUSION"
)

// InterpretBilirubin evaluates neonatal bilirubin against AAP 2022 thresholds
func (b *BilirubinInterpreter) InterpretBilirubin(ctx context.Context, value float64, patient *types.PatientContext) (*BilirubinInterpretation, error) {
	// Validate patient context
	if !patient.IsNeonate {
		return nil, fmt.Errorf("bilirubin nomogram only applies to neonates")
	}
	if patient.HoursOfLife <= 0 {
		return nil, fmt.Errorf("hours of life must be positive")
	}
	if patient.GestationalAgeAtBirth <= 0 {
		return nil, fmt.Errorf("gestational age at birth is required")
	}

	// Determine risk category
	riskCategory := b.determineRiskCategory(patient)

	// Get thresholds with interpolation
	photoThreshold, exchangeThreshold, interpolated, err := b.getThresholds(ctx, patient.GestationalAgeAtBirth, riskCategory, patient.HoursOfLife)
	if err != nil {
		return nil, fmt.Errorf("failed to get bilirubin thresholds: %w", err)
	}

	// Determine zone and treatment needs
	zone := b.determineZone(value, photoThreshold, exchangeThreshold)
	needsPhoto := value >= photoThreshold
	needsExchange := exchangeThreshold != nil && value >= *exchangeThreshold

	interpretation := &BilirubinInterpretation{
		Value:             value,
		Unit:              "mg/dL",
		HoursOfLife:       patient.HoursOfLife,
		GestationalAge:    patient.GestationalAgeAtBirth,
		RiskCategory:      riskCategory,
		PhotoThreshold:    photoThreshold,
		ExchangeThreshold: exchangeThreshold,
		NeedsPhototherapy: needsPhoto,
		NeedsExchange:     needsExchange,
		Zone:              string(zone),
		Interpolated:      interpolated,
		Authority:         "AAP",
		AuthorityRef:      "AAP Clinical Practice Guideline: Management of Hyperbilirubinemia 2022",
	}

	return interpretation, nil
}

// determineRiskCategory classifies neonate into risk category
func (b *BilirubinInterpreter) determineRiskCategory(patient *types.PatientContext) RiskCategory {
	// First, use explicit risk category if set
	if patient.NeonatalRiskCategory != "" {
		return RiskCategory(patient.NeonatalRiskCategory)
	}

	// Otherwise, determine from gestational age
	ga := patient.GestationalAgeAtBirth
	switch {
	case ga >= 38:
		// Check for risk factors that would upgrade to MEDIUM
		if b.hasHighRiskFactors(patient) {
			return RiskMedium
		}
		return RiskLow
	case ga >= 35:
		return RiskMedium
	default:
		return RiskHigh
	}
}

// hasHighRiskFactors checks for risk factors that modify bilirubin thresholds
func (b *BilirubinInterpreter) hasHighRiskFactors(patient *types.PatientContext) bool {
	// Risk factors per AAP 2022:
	// - Isoimmune hemolytic disease (ABO, Rh incompatibility)
	// - G6PD deficiency
	// - Asphyxia
	// - Significant lethargy
	// - Temperature instability
	// - Sepsis
	// - Acidosis
	// - Albumin < 3.0 g/dL

	riskConditions := []string{
		"P55.0", // Rh isoimmunization
		"P55.1", // ABO isoimmunization
		"D55.0", // G6PD deficiency
		"P21",   // Birth asphyxia
		"P36",   // Bacterial sepsis of newborn
	}

	return patient.HasRiskFactor(riskConditions...)
}

// getThresholds retrieves phototherapy/exchange thresholds with interpolation
func (b *BilirubinInterpreter) getThresholds(ctx context.Context, gaWeeks int, risk RiskCategory, hoursOfLife int) (float64, *float64, bool, error) {
	// Clamp hours of life to valid range
	if hoursOfLife < 12 {
		hoursOfLife = 12 // Below 12h, use 12h threshold
	}
	if hoursOfLife > 120 {
		hoursOfLife = 120 // Beyond 120h, use 120h threshold
	}

	// Determine GA range for query
	gaMin, gaMax := b.getGARange(gaWeeks)

	// Try exact match first
	var exactThreshold NeonatalBilirubinThreshold
	err := b.db.WithContext(ctx).
		Where("gestational_age_weeks_min = ? AND gestational_age_weeks_max = ?", gaMin, gaMax).
		Where("risk_category = ?", string(risk)).
		Where("hour_of_life = ?", hoursOfLife).
		First(&exactThreshold).Error

	if err == nil {
		// Exact match found
		return exactThreshold.PhotoThreshold, exactThreshold.ExchangeThreshold, false, nil
	}

	if err != gorm.ErrRecordNotFound {
		return 0, nil, false, err
	}

	// No exact match - interpolate between bracketing hours
	return b.interpolateThresholds(ctx, gaMin, gaMax, risk, hoursOfLife)
}

// getGARange returns the appropriate GA range for the nomogram
func (b *BilirubinInterpreter) getGARange(gaWeeks int) (int, int) {
	switch {
	case gaWeeks >= 38:
		return 38, 45
	case gaWeeks >= 35:
		return 35, 37
	default:
		return 28, 34
	}
}

// interpolateThresholds performs linear interpolation between hour-of-life points
func (b *BilirubinInterpreter) interpolateThresholds(ctx context.Context, gaMin, gaMax int, risk RiskCategory, hoursOfLife int) (float64, *float64, bool, error) {
	// Get lower bracket
	var lower NeonatalBilirubinThreshold
	err := b.db.WithContext(ctx).
		Where("gestational_age_weeks_min = ? AND gestational_age_weeks_max = ?", gaMin, gaMax).
		Where("risk_category = ?", string(risk)).
		Where("hour_of_life < ?", hoursOfLife).
		Order("hour_of_life DESC").
		First(&lower).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return 0, nil, false, fmt.Errorf("no lower threshold found for interpolation")
		}
		return 0, nil, false, err
	}

	// Get upper bracket
	var upper NeonatalBilirubinThreshold
	err = b.db.WithContext(ctx).
		Where("gestational_age_weeks_min = ? AND gestational_age_weeks_max = ?", gaMin, gaMax).
		Where("risk_category = ?", string(risk)).
		Where("hour_of_life > ?", hoursOfLife).
		Order("hour_of_life ASC").
		First(&upper).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			// No upper bound - use lower threshold
			return lower.PhotoThreshold, lower.ExchangeThreshold, false, nil
		}
		return 0, nil, false, err
	}

	// Calculate interpolation factor
	factor := float64(hoursOfLife-lower.HourOfLife) / float64(upper.HourOfLife-lower.HourOfLife)

	// Interpolate phototherapy threshold
	photoThreshold := lower.PhotoThreshold + factor*(upper.PhotoThreshold-lower.PhotoThreshold)
	photoThreshold = math.Round(photoThreshold*10) / 10 // Round to 1 decimal

	// Interpolate exchange threshold (if both exist)
	var exchangeThreshold *float64
	if lower.ExchangeThreshold != nil && upper.ExchangeThreshold != nil {
		interpolatedExchange := *lower.ExchangeThreshold + factor*(*upper.ExchangeThreshold-*lower.ExchangeThreshold)
		interpolatedExchange = math.Round(interpolatedExchange*10) / 10
		exchangeThreshold = &interpolatedExchange
	} else if lower.ExchangeThreshold != nil {
		exchangeThreshold = lower.ExchangeThreshold
	}

	return photoThreshold, exchangeThreshold, true, nil
}

// determineZone classifies the bilirubin value into a risk zone
func (b *BilirubinInterpreter) determineZone(value, photoThreshold float64, exchangeThreshold *float64) BilirubinZone {
	// Check exchange threshold first (most severe)
	if exchangeThreshold != nil && value >= *exchangeThreshold {
		return ZoneExchange
	}

	// Check phototherapy threshold
	if value >= photoThreshold {
		return ZonePhototherapy
	}

	// Determine percentile zone relative to photo threshold
	percentOfPhoto := (value / photoThreshold) * 100

	switch {
	case percentOfPhoto >= 85:
		return ZoneHighRisk
	case percentOfPhoto >= 75:
		return ZoneHighIntermediate
	case percentOfPhoto >= 50:
		return ZoneLowIntermediate
	default:
		return ZoneLowRisk
	}
}

// GetZoneRecommendation returns clinical recommendations based on zone
func (b *BilirubinInterpreter) GetZoneRecommendation(zone BilirubinZone, interpretation *BilirubinInterpretation) string {
	switch zone {
	case ZoneExchange:
		return "URGENT: Bilirubin at exchange transfusion threshold. Immediate pediatric/neonatology consultation required. Consider intensive phototherapy while preparing for exchange."

	case ZonePhototherapy:
		return "Start phototherapy immediately per AAP guidelines. Repeat bilirubin in 4-6 hours. Monitor for adequate hydration and feeding."

	case ZoneHighRisk:
		value := interpretation.Value
		threshold := interpretation.PhotoThreshold
		return fmt.Sprintf("HIGH RISK: Bilirubin %.1f approaching phototherapy threshold (%.1f). Repeat in 4-8 hours. Consider early phototherapy if risk factors present.", value, threshold)

	case ZoneHighIntermediate:
		return "HIGH-INTERMEDIATE ZONE: Repeat bilirubin in 6-12 hours. Ensure adequate feeding and hydration. Monitor for clinical jaundice progression."

	case ZoneLowIntermediate:
		return "LOW-INTERMEDIATE ZONE: Repeat bilirubin in 12-24 hours if still in hospital, or at first outpatient visit. Provide jaundice education to parents."

	case ZoneLowRisk:
		return "LOW RISK: Routine follow-up appropriate. Provide jaundice education and when to seek care."

	default:
		return "Follow institutional protocol for neonatal jaundice management."
	}
}

// GetBilirubinVelocity calculates rate of bilirubin rise (mg/dL per hour)
// Velocity > 0.2 mg/dL/hour suggests hemolysis
func (b *BilirubinInterpreter) GetBilirubinVelocity(currentValue, previousValue float64, hoursBetween int) (float64, string) {
	if hoursBetween <= 0 {
		return 0, "Invalid time interval"
	}

	velocity := (currentValue - previousValue) / float64(hoursBetween)

	var interpretation string
	switch {
	case velocity > 0.3:
		interpretation = "CRITICAL: Rapid rise (>0.3 mg/dL/hr) - likely hemolysis. Urgent evaluation for isoimmune disease, G6PD deficiency."
	case velocity > 0.2:
		interpretation = "ELEVATED velocity (>0.2 mg/dL/hr) - suggests ongoing hemolysis. Consider Coombs test, reticulocyte count."
	case velocity > 0.1:
		interpretation = "Moderate rise - continue monitoring closely."
	default:
		interpretation = "Normal velocity - reassuring trajectory."
	}

	return velocity, interpretation
}

// GetFollowUpTiming recommends follow-up interval based on current zone and hours of life
func (b *BilirubinInterpreter) GetFollowUpTiming(zone BilirubinZone, hoursOfLife int) string {
	// Early hours are higher risk - shorter follow-up
	earlyPeriod := hoursOfLife < 48

	switch zone {
	case ZoneExchange, ZonePhototherapy:
		return "4-6 hours (during/after treatment)"
	case ZoneHighRisk:
		if earlyPeriod {
			return "4-8 hours"
		}
		return "8-12 hours"
	case ZoneHighIntermediate:
		if earlyPeriod {
			return "8-12 hours"
		}
		return "12-24 hours"
	case ZoneLowIntermediate:
		return "24 hours or first outpatient visit"
	default:
		return "Routine follow-up per discharge criteria"
	}
}

// NeedsSpecialtyConsult determines if neonatology/pediatrics consultation is needed
func (b *BilirubinInterpreter) NeedsSpecialtyConsult(interpretation *BilirubinInterpretation) (bool, string) {
	zone := BilirubinZone(interpretation.Zone)

	switch {
	case interpretation.NeedsExchange:
		return true, "URGENT: Exchange transfusion threshold reached - immediate neonatology consultation"
	case interpretation.NeedsPhototherapy && interpretation.HoursOfLife < 24:
		return true, "Phototherapy needed in first 24 hours of life - neonatology consultation recommended"
	case zone == ZoneHighRisk && interpretation.RiskCategory == RiskHigh:
		return true, "High-risk neonate in high-risk zone - consider neonatology consultation"
	case interpretation.GestationalAge < 35:
		return true, "Premature infant (<35 weeks) - neonatology involvement recommended"
	default:
		return false, ""
	}
}
