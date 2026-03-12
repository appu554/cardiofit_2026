// Package interpretation provides clinical interpretation algorithms
// neonatal_bilirubin.go implements AAP 2022 hour-specific bilirubin nomogram
package interpretation

import (
	"math"
)

// =============================================================================
// NEONATAL BILIRUBIN ASSESSMENT (AAP 2022 Clinical Practice Guideline)
// =============================================================================

// NeonatalBilirubinAssessment contains complete bilirubin assessment results
type NeonatalBilirubinAssessment struct {
	// Input Values
	TotalBilirubin   float64  `json:"totalBilirubin"`   // mg/dL
	HoursOfLife      float64  `json:"hoursOfLife"`      // 0-168 (7 days)
	GestationalAge   int      `json:"gestationalAge"`   // weeks (35-42+)
	BirthWeight      float64  `json:"birthWeight"`      // grams
	RiskFactors      []string `json:"riskFactors"`

	// Calculated Values
	RiskZone              string  `json:"riskZone"`              // LOW, LOW_INTERMEDIATE, HIGH_INTERMEDIATE, HIGH
	PhototherapyThreshold float64 `json:"phototherapyThreshold"` // mg/dL
	ExchangeThreshold     float64 `json:"exchangeThreshold"`     // mg/dL
	RateOfRise            float64 `json:"rateOfRise,omitempty"`  // mg/dL/hr (if prior measurement)

	// Clinical Interpretation
	Recommendation     string   `json:"recommendation"`
	FollowUpHours      int      `json:"followUpHours"`
	UrgencyLevel       string   `json:"urgencyLevel"`       // ROUTINE, URGENT, EMERGENT
	ActionRequired     []string `json:"actionRequired"`
	RiskAssessment     string   `json:"riskAssessment"`

	// Governance
	Governance BilirubinGovernance `json:"governance"`
}

// BilirubinGovernance tracks clinical authority
type BilirubinGovernance struct {
	GuidelineSource   string `json:"guidelineSource"`
	GuidelineRef      string `json:"guidelineRef"`
	EvidenceLevel     string `json:"evidenceLevel"`
	PhotoThresholdRef string `json:"photoThresholdRef"`
	NomogramVersion   string `json:"nomogramVersion"`
}

// =============================================================================
// AAP 2022 RISK FACTORS
// =============================================================================

// Neurotoxicity Risk Factors (AAP 2022 - Table 1)
const (
	RiskFactorIsoimmune       = "isoimmune_hemolytic_disease"    // ABO, Rh incompatibility
	RiskFactorG6PD            = "g6pd_deficiency"
	RiskFactorAsphyxia        = "perinatal_asphyxia"
	RiskFactorSepsis          = "sepsis"
	RiskFactorAcidosis        = "significant_acidosis"
	RiskFactorAlbumin         = "albumin_less_than_3"
	RiskFactorLethargy        = "lethargy"
	RiskFactorTemperature     = "temperature_instability"
	RiskFactorDirectBili      = "elevated_direct_bilirubin"
	RiskFactorPriorSibling    = "sibling_with_jaundice"
	RiskFactorEastAsian       = "east_asian_ethnicity"
	RiskFactorCephalohematoma = "cephalohematoma_bruising"
	RiskFactorExclusiveBF     = "exclusive_breastfeeding"
	RiskFactorPreterm         = "late_preterm_35_36"
)

// Hyperbilirubinemia risk categories
const (
	RiskZoneLow              = "LOW"
	RiskZoneLowIntermediate  = "LOW_INTERMEDIATE"
	RiskZoneHighIntermediate = "HIGH_INTERMEDIATE"
	RiskZoneHigh             = "HIGH"
)

// =============================================================================
// AAP 2022 PHOTOTHERAPY THRESHOLDS (mg/dL)
// Based on gestational age and risk factors
// =============================================================================

// PhotoThresholds contains hour-specific phototherapy thresholds
// Organized by gestational age and risk category
var PhotoThresholds = map[int]map[string][]struct {
	Hours     float64
	Threshold float64
}{
	// ≥38 weeks without neurotoxicity risk factors
	38: {
		"standard": {
			{12, 13.0}, {24, 15.0}, {36, 17.5}, {48, 18.5},
			{60, 19.5}, {72, 20.0}, {84, 21.0}, {96, 21.0},
		},
		// ≥38 weeks WITH neurotoxicity risk factors
		"high_risk": {
			{12, 10.5}, {24, 12.5}, {36, 14.5}, {48, 15.5},
			{60, 16.5}, {72, 17.5}, {84, 18.0}, {96, 18.0},
		},
	},
	// 35-37 weeks without neurotoxicity risk factors
	35: {
		"standard": {
			{12, 10.0}, {24, 12.5}, {36, 14.0}, {48, 15.5},
			{60, 16.5}, {72, 17.5}, {84, 18.0}, {96, 18.5},
		},
		// 35-37 weeks WITH neurotoxicity risk factors
		"high_risk": {
			{12, 8.0}, {24, 10.0}, {36, 11.5}, {48, 12.5},
			{60, 13.5}, {72, 14.5}, {84, 15.0}, {96, 15.5},
		},
	},
}

// ExchangeThresholds contains hour-specific exchange transfusion thresholds
var ExchangeThresholds = map[int]map[string][]struct {
	Hours     float64
	Threshold float64
}{
	// ≥38 weeks
	38: {
		"standard": {
			{12, 21.0}, {24, 22.0}, {36, 23.0}, {48, 24.0},
			{60, 24.5}, {72, 25.0}, {84, 25.0}, {96, 25.0},
		},
		"high_risk": {
			{12, 17.0}, {24, 18.5}, {36, 20.0}, {48, 21.0},
			{60, 21.5}, {72, 22.0}, {84, 22.5}, {96, 22.5},
		},
	},
	// 35-37 weeks
	35: {
		"standard": {
			{12, 17.0}, {24, 18.5}, {36, 19.5}, {48, 21.0},
			{60, 22.0}, {72, 22.5}, {84, 23.0}, {96, 23.0},
		},
		"high_risk": {
			{12, 14.0}, {24, 15.0}, {36, 16.0}, {48, 17.0},
			{60, 18.0}, {72, 18.5}, {84, 19.0}, {96, 19.0},
		},
	},
}

// =============================================================================
// BHUTANI NOMOGRAM - Risk Zone Classification
// =============================================================================

// BhutaniNomogram defines hour-specific TSB percentile thresholds
var BhutaniNomogram = []struct {
	Hours            float64
	Low              float64 // <40th percentile
	LowIntermediate  float64 // 40th-75th percentile
	HighIntermediate float64 // 75th-95th percentile
	High             float64 // >95th percentile
}{
	{24, 4.0, 5.5, 7.5, 9.0},
	{36, 6.0, 7.5, 9.5, 11.5},
	{48, 7.5, 9.5, 12.0, 14.0},
	{60, 9.0, 11.0, 13.5, 16.0},
	{72, 10.0, 12.0, 14.5, 17.5},
	{84, 10.5, 12.5, 15.0, 18.5},
	{96, 11.0, 13.0, 15.5, 19.0},
	{108, 11.0, 13.0, 15.5, 19.5},
	{120, 11.0, 13.0, 15.5, 19.5},
}

// =============================================================================
// ASSESSMENT FUNCTION
// =============================================================================

// AssessNeonatalBilirubin performs comprehensive bilirubin risk assessment per AAP 2022
func AssessNeonatalBilirubin(bili, hoursOfLife float64, gestAge int, riskFactors []string) *NeonatalBilirubinAssessment {
	assessment := &NeonatalBilirubinAssessment{
		TotalBilirubin: bili,
		HoursOfLife:    hoursOfLife,
		GestationalAge: gestAge,
		RiskFactors:    riskFactors,
		Governance: BilirubinGovernance{
			GuidelineSource:   "AAP.Bilirubin",
			GuidelineRef:      "AAP Clinical Practice Guideline: Hyperbilirubinemia in the Newborn (2022)",
			EvidenceLevel:     "STRONG (1A)",
			PhotoThresholdRef: "AAP 2022 Figure 2: Phototherapy Thresholds",
			NomogramVersion:   "Bhutani 2004 (updated AAP 2022)",
		},
	}

	// Determine risk category
	hasNeurotoxicityRisk := assessment.hasNeurotoxicityRiskFactors()

	// Calculate thresholds based on gestational age and risk
	assessment.calculateThresholds(hasNeurotoxicityRisk)

	// Determine risk zone using Bhutani nomogram
	assessment.determineRiskZone()

	// Generate clinical interpretation
	assessment.interpretResults(hasNeurotoxicityRisk)

	return assessment
}

// hasNeurotoxicityRiskFactors checks for AAP 2022 neurotoxicity risk factors
func (a *NeonatalBilirubinAssessment) hasNeurotoxicityRiskFactors() bool {
	neurotoxicityRisks := map[string]bool{
		RiskFactorIsoimmune:   true,
		RiskFactorG6PD:        true,
		RiskFactorAsphyxia:    true,
		RiskFactorSepsis:      true,
		RiskFactorAcidosis:    true,
		RiskFactorAlbumin:     true,
		RiskFactorLethargy:    true,
		RiskFactorTemperature: true,
	}

	for _, rf := range a.RiskFactors {
		if neurotoxicityRisks[rf] {
			return true
		}
	}

	// Late preterm (35-36 weeks) is also high risk
	if a.GestationalAge >= 35 && a.GestationalAge <= 36 {
		return true
	}

	return false
}

// calculateThresholds determines phototherapy and exchange thresholds
func (a *NeonatalBilirubinAssessment) calculateThresholds(highRisk bool) {
	// Select gestational age category
	gaCategory := 38
	if a.GestationalAge < 38 {
		gaCategory = 35
	}

	// Select risk category
	riskCategory := "standard"
	if highRisk {
		riskCategory = "high_risk"
	}

	// Get phototherapy threshold
	photoThresholds := PhotoThresholds[gaCategory][riskCategory]
	a.PhototherapyThreshold = interpolateThreshold(a.HoursOfLife, photoThresholds)

	// Get exchange threshold
	exchangeThresholds := ExchangeThresholds[gaCategory][riskCategory]
	a.ExchangeThreshold = interpolateThreshold(a.HoursOfLife, exchangeThresholds)
}

// interpolateThreshold calculates threshold for exact hour of life
func interpolateThreshold(hours float64, thresholds []struct {
	Hours     float64
	Threshold float64
}) float64 {
	if len(thresholds) == 0 {
		return 0
	}

	// Before first threshold point
	if hours <= thresholds[0].Hours {
		return thresholds[0].Threshold
	}

	// After last threshold point
	if hours >= thresholds[len(thresholds)-1].Hours {
		return thresholds[len(thresholds)-1].Threshold
	}

	// Linear interpolation between points
	for i := 1; i < len(thresholds); i++ {
		if hours <= thresholds[i].Hours {
			prev := thresholds[i-1]
			curr := thresholds[i]
			ratio := (hours - prev.Hours) / (curr.Hours - prev.Hours)
			return prev.Threshold + ratio*(curr.Threshold-prev.Threshold)
		}
	}

	return thresholds[len(thresholds)-1].Threshold
}

// determineRiskZone classifies bilirubin level using Bhutani nomogram
func (a *NeonatalBilirubinAssessment) determineRiskZone() {
	// Find nearest nomogram values
	var lowThresh, lowIntThresh, highIntThresh, highThresh float64

	for i := 1; i < len(BhutaniNomogram); i++ {
		if a.HoursOfLife <= BhutaniNomogram[i].Hours {
			prev := BhutaniNomogram[i-1]
			curr := BhutaniNomogram[i]

			if a.HoursOfLife < prev.Hours {
				// Before first timepoint
				lowThresh = prev.Low
				lowIntThresh = prev.LowIntermediate
				highIntThresh = prev.HighIntermediate
				highThresh = prev.High
			} else {
				// Interpolate
				ratio := (a.HoursOfLife - prev.Hours) / (curr.Hours - prev.Hours)
				lowThresh = prev.Low + ratio*(curr.Low-prev.Low)
				lowIntThresh = prev.LowIntermediate + ratio*(curr.LowIntermediate-prev.LowIntermediate)
				highIntThresh = prev.HighIntermediate + ratio*(curr.HighIntermediate-prev.HighIntermediate)
				highThresh = prev.High + ratio*(curr.High-prev.High)
			}
			break
		}
	}

	// If beyond nomogram range, use last values
	if lowThresh == 0 {
		last := BhutaniNomogram[len(BhutaniNomogram)-1]
		lowThresh = last.Low
		lowIntThresh = last.LowIntermediate
		highIntThresh = last.HighIntermediate
		highThresh = last.High
	}

	// Classify
	if a.TotalBilirubin >= highThresh {
		a.RiskZone = RiskZoneHigh
	} else if a.TotalBilirubin >= highIntThresh {
		a.RiskZone = RiskZoneHighIntermediate
	} else if a.TotalBilirubin >= lowIntThresh {
		a.RiskZone = RiskZoneLowIntermediate
	} else {
		a.RiskZone = RiskZoneLow
	}
}

// interpretResults generates clinical recommendations
func (a *NeonatalBilirubinAssessment) interpretResults(highRisk bool) {
	a.ActionRequired = []string{}

	// Check against exchange threshold first (most critical)
	if a.TotalBilirubin >= a.ExchangeThreshold {
		a.UrgencyLevel = "EMERGENT"
		a.Recommendation = "EXCHANGE TRANSFUSION indicated - TSB at or above exchange threshold"
		a.FollowUpHours = 0 // Immediate action
		a.ActionRequired = []string{
			"IMMEDIATE: Prepare for exchange transfusion",
			"Start intensive phototherapy while preparing",
			"Notify NICU and blood bank urgently",
			"Recheck TSB in 2 hours during phototherapy",
			"Consider IVIG if isoimmune hemolysis",
			"Ensure hydration and temperature stability",
		}
		a.RiskAssessment = "CRITICAL - Bilirubin at exchange threshold. High risk for bilirubin encephalopathy."
		return
	}

	// Check against phototherapy threshold
	if a.TotalBilirubin >= a.PhototherapyThreshold {
		a.UrgencyLevel = "URGENT"
		a.Recommendation = "PHOTOTHERAPY indicated - TSB at or above phototherapy threshold"
		a.FollowUpHours = 4
		a.ActionRequired = []string{
			"Start phototherapy immediately",
			"Ensure adequate irradiance (≥30 μW/cm²/nm)",
			"Maximize skin exposure",
			"Recheck TSB in 4-6 hours",
			"Continue feeding to promote elimination",
			"Monitor temperature, hydration, intake/output",
		}

		// Distance from exchange threshold
		margin := a.ExchangeThreshold - a.TotalBilirubin
		if margin < 2.0 {
			a.ActionRequired = append(a.ActionRequired,
				"WARNING: Within 2 mg/dL of exchange threshold - monitor closely",
				"Consider intensive phototherapy (multiple devices)")
			a.RiskAssessment = "HIGH - Close to exchange threshold. Intensive monitoring required."
		} else {
			a.RiskAssessment = "MODERATE - Phototherapy indicated. Monitor for response."
		}
		return
	}

	// Below phototherapy threshold - assess risk zone
	switch a.RiskZone {
	case RiskZoneHigh:
		a.UrgencyLevel = "URGENT"
		a.Recommendation = "High-risk zone - Close monitoring required, approach phototherapy threshold"
		a.FollowUpHours = 6
		a.ActionRequired = []string{
			"Recheck TSB in 6-8 hours or before discharge",
			"Assess for risk factors and feeding adequacy",
			"Do not discharge until repeat TSB shows stable/decreasing trend",
			"Ensure follow-up appointment within 24-48 hours after discharge",
		}
		a.RiskAssessment = "HIGH RISK ZONE - Likely to require intervention. Close monitoring essential."

	case RiskZoneHighIntermediate:
		a.UrgencyLevel = "ROUTINE"
		a.Recommendation = "High-intermediate risk zone - Monitoring indicated"
		a.FollowUpHours = 12
		a.ActionRequired = []string{
			"Recheck TSB in 12-24 hours",
			"Evaluate feeding and hydration",
			"Schedule outpatient follow-up within 48-72 hours",
			"Provide parent education on jaundice warning signs",
		}
		a.RiskAssessment = "MODERATE RISK - May progress to needing phototherapy. Ensure follow-up."

	case RiskZoneLowIntermediate:
		a.UrgencyLevel = "ROUTINE"
		a.Recommendation = "Low-intermediate risk zone - Routine monitoring"
		a.FollowUpHours = 24
		if highRisk {
			a.FollowUpHours = 12
		}
		a.ActionRequired = []string{
			"Recheck TSB based on clinical judgment",
			"Ensure adequate feeding",
			"Schedule routine follow-up",
			"Provide parent education",
		}
		a.RiskAssessment = "LOW-MODERATE RISK - Low likelihood of requiring treatment."

	case RiskZoneLow:
		a.UrgencyLevel = "ROUTINE"
		a.Recommendation = "Low-risk zone - Routine care"
		a.FollowUpHours = 48
		a.ActionRequired = []string{
			"Routine newborn care",
			"Ensure adequate feeding (8-12 feeds per 24 hours)",
			"Parent education on jaundice",
			"Routine outpatient follow-up",
		}
		a.RiskAssessment = "LOW RISK - Unlikely to require intervention."
	}

	// Adjust for risk factors
	if highRisk {
		a.ActionRequired = append(a.ActionRequired,
			"Note: Neurotoxicity risk factors present - use lower thresholds",
			"Consider closer follow-up intervals")
	}
}

// =============================================================================
// RATE OF RISE CALCULATION
// =============================================================================

// CalculateRateOfRise calculates bilirubin rise rate between measurements
func CalculateRateOfRise(currentBili, priorBili, hoursElapsed float64) float64 {
	if hoursElapsed <= 0 {
		return 0
	}
	return (currentBili - priorBili) / hoursElapsed
}

// InterpretRateOfRise assesses significance of bilirubin rise rate
func InterpretRateOfRise(ratePerHour float64, hoursOfLife float64) string {
	// AAP 2022: Rate >0.2 mg/dL/hr is concerning
	// Rate >0.3 mg/dL/hr suggests hemolysis

	if ratePerHour > 0.3 {
		return "CRITICAL: Rate >0.3 mg/dL/hr - strongly suggests hemolysis. Evaluate for isoimmune disease, G6PD."
	} else if ratePerHour > 0.2 {
		return "ELEVATED: Rate >0.2 mg/dL/hr - increased risk, closer monitoring needed"
	} else if ratePerHour > 0.15 && hoursOfLife < 48 {
		return "CAUTION: Rate elevated for age, monitor trend"
	}
	return "NORMAL: Rate of rise within expected range"
}

// =============================================================================
// PHOTOTHERAPY EFFECTIVENESS
// =============================================================================

// PhototherapyResponse assesses response to phototherapy
type PhototherapyResponse struct {
	PreTreatmentTSB  float64 `json:"preTreatmentTsb"`
	PostTreatmentTSB float64 `json:"postTreatmentTsb"`
	HoursOfTherapy   float64 `json:"hoursOfTherapy"`
	DeclineRate      float64 `json:"declineRate"`     // mg/dL/hr
	PercentDecline   float64 `json:"percentDecline"`
	Response         string  `json:"response"`        // GOOD, SUBOPTIMAL, POOR, NO_RESPONSE
	Recommendation   string  `json:"recommendation"`
}

// AssessPhototherapyResponse evaluates response to phototherapy
func AssessPhototherapyResponse(preTSB, postTSB, hours float64) *PhototherapyResponse {
	response := &PhototherapyResponse{
		PreTreatmentTSB:  preTSB,
		PostTreatmentTSB: postTSB,
		HoursOfTherapy:   hours,
	}

	if hours <= 0 {
		response.Response = "UNABLE_TO_ASSESS"
		response.Recommendation = "Insufficient time elapsed"
		return response
	}

	response.DeclineRate = (preTSB - postTSB) / hours
	if preTSB > 0 {
		response.PercentDecline = ((preTSB - postTSB) / preTSB) * 100
	}

	// Expected decline: 1-2 mg/dL in first 4-6 hours with intensive phototherapy
	// Subsequently: 30-40% decline in 24 hours is good response

	if hours <= 6 {
		// Early response assessment
		if response.DeclineRate >= 0.2 {
			response.Response = "GOOD"
			response.Recommendation = "Good response to phototherapy. Continue treatment, recheck in 4-6 hours."
		} else if response.DeclineRate >= 0.1 {
			response.Response = "SUBOPTIMAL"
			response.Recommendation = "Suboptimal response. Verify phototherapy technique, consider intensive phototherapy."
		} else if response.DeclineRate >= 0 {
			response.Response = "POOR"
			response.Recommendation = "Poor response. Evaluate for hemolysis, check irradiance, consider additional lights."
		} else {
			response.Response = "NO_RESPONSE"
			response.Recommendation = "TSB rising despite phototherapy. Urgent evaluation for underlying cause. Consider exchange transfusion preparation."
		}
	} else {
		// Longer-term response
		if response.PercentDecline >= 30 {
			response.Response = "GOOD"
			response.Recommendation = "Good response. Consider stopping phototherapy if below threshold."
		} else if response.PercentDecline >= 15 {
			response.Response = "SUBOPTIMAL"
			response.Recommendation = "Suboptimal response. Continue phototherapy, investigate underlying cause."
		} else {
			response.Response = "POOR"
			response.Recommendation = "Poor response despite prolonged therapy. Evaluate for hemolysis, consider IVIG."
		}
	}

	return response
}

// =============================================================================
// REBOUND ASSESSMENT
// =============================================================================

// ReboundRisk assesses risk of rebound hyperbilirubinemia after stopping phototherapy
type ReboundRisk struct {
	TSBAtDiscontinuation float64  `json:"tsbAtDiscontinuation"`
	Threshold            float64  `json:"threshold"`
	MarginBelowThreshold float64  `json:"marginBelowThreshold"`
	RiskLevel            string   `json:"riskLevel"`       // LOW, MODERATE, HIGH
	RiskFactors          []string `json:"riskFactors"`
	FollowUpNeeded       bool     `json:"followUpNeeded"`
	RecommendedFollowUp  int      `json:"recommendedFollowUp"` // hours
}

// AssessReboundRisk evaluates rebound risk after phototherapy discontinuation
func AssessReboundRisk(tsbAtStop, threshold float64, hoursOfLife float64, riskFactors []string) *ReboundRisk {
	risk := &ReboundRisk{
		TSBAtDiscontinuation: tsbAtStop,
		Threshold:            threshold,
		MarginBelowThreshold: threshold - tsbAtStop,
		RiskFactors:          riskFactors,
	}

	// Rebound risk factors
	hasHemolyticRisk := false
	for _, rf := range riskFactors {
		if rf == RiskFactorIsoimmune || rf == RiskFactorG6PD {
			hasHemolyticRisk = true
			break
		}
	}

	// Calculate risk
	if hasHemolyticRisk || risk.MarginBelowThreshold < 2.0 || hoursOfLife < 48 {
		risk.RiskLevel = "HIGH"
		risk.FollowUpNeeded = true
		risk.RecommendedFollowUp = 12
	} else if risk.MarginBelowThreshold < 3.0 {
		risk.RiskLevel = "MODERATE"
		risk.FollowUpNeeded = true
		risk.RecommendedFollowUp = 24
	} else {
		risk.RiskLevel = "LOW"
		risk.FollowUpNeeded = hoursOfLife < 72
		risk.RecommendedFollowUp = 48
	}

	return risk
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// EstimateProjectedTSB estimates future TSB based on rate of rise
func EstimateProjectedTSB(currentTSB, ratePerHour, hoursAhead float64) float64 {
	projected := currentTSB + (ratePerHour * hoursAhead)
	return math.Max(0, projected)
}

// GetDischargeEligibility determines if discharge is appropriate
func GetDischargeEligibility(assessment *NeonatalBilirubinAssessment) (bool, string) {
	if assessment.UrgencyLevel == "EMERGENT" {
		return false, "Not eligible for discharge - urgent intervention required"
	}

	if assessment.UrgencyLevel == "URGENT" {
		return false, "Not eligible for discharge - monitoring and likely intervention required"
	}

	if assessment.RiskZone == RiskZoneHigh {
		return false, "Not eligible for discharge - high-risk zone, repeat TSB required"
	}

	if assessment.RiskZone == RiskZoneHighIntermediate {
		return false, "Discharge with caution - ensure follow-up within 24-48 hours"
	}

	return true, "Eligible for discharge with routine follow-up and parent education"
}

// GetRiskFactorCount returns count of neurotoxicity risk factors
func GetRiskFactorCount(riskFactors []string) int {
	neurotoxicity := []string{
		RiskFactorIsoimmune, RiskFactorG6PD, RiskFactorAsphyxia,
		RiskFactorSepsis, RiskFactorAcidosis, RiskFactorAlbumin,
		RiskFactorLethargy, RiskFactorTemperature,
	}

	count := 0
	for _, rf := range riskFactors {
		for _, nr := range neurotoxicity {
			if rf == nr {
				count++
				break
			}
		}
	}
	return count
}
