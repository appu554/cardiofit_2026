package services

import (
	"math"
)

// TimedValue pairs a measurement with its time offset for regression.
type TimedValue struct {
	Value         float64
	DaysSinceFirst float64
}

// ComputeMAP returns Mean Arterial Pressure: DBP + (SBP - DBP) / 3.
func ComputeMAP(sbp, dbp float64) float64 {
	return dbp + (sbp-dbp)/3.0
}

// ComputeEGFR_CKDEPI2021 computes eGFR using the CKD-EPI 2021 race-free equation.
// Sex: "M" or "F". Creatinine in mg/dL. Age in years.
// Returns eGFR in mL/min/1.73m².
func ComputeEGFR_CKDEPI2021(creatinine float64, sex string, age int) float64 {
	if creatinine <= 0 {
		return 0
	}

	var kappa, alpha float64
	var sexCoeff float64

	if sex == "F" {
		kappa = 0.7
		alpha = -0.241
		sexCoeff = 1.012
	} else {
		kappa = 0.9
		alpha = -0.302
		sexCoeff = 1.0
	}

	ratio := creatinine / kappa
	minRatio := math.Min(ratio, 1.0)
	maxRatio := math.Max(ratio, 1.0)

	return 142 * math.Pow(minRatio, alpha) * math.Pow(maxRatio, -1.200) * math.Pow(0.9938, float64(age)) * sexCoeff
}

// ComputeVisceralFatProxy returns a composite 0-1 score estimating visceral
// adiposity from waist circumference, height, triglycerides, and HDL.
//
// Formula: 0.5 * norm(waist) + 0.3 * norm(waist/height) + 0.2 * norm(TG/HDL)
//
// Normalization ranges (population-based):
//   - waist: 60-140 cm
//   - waist/height: 0.3-0.8
//   - TG/HDL: 0.5-6.0
func ComputeVisceralFatProxy(waistCm, heightCm, tg, hdl float64) float64 {
	if heightCm <= 0 || hdl <= 0 {
		return 0
	}

	normWaist := clamp((waistCm-60.0)/(140.0-60.0), 0, 1)
	whr := waistCm / heightCm
	normWHR := clamp((whr-0.3)/(0.8-0.3), 0, 1)
	tgHDL := tg / hdl
	normTGHDL := clamp((tgHDL-0.5)/(6.0-0.5), 0, 1)

	return clamp(0.5*normWaist+0.3*normWHR+0.2*normTGHDL, 0, 1)
}

// ComputeGlycemicVariability returns the coefficient of variation (CV%) over
// a set of glucose readings. Returns 0 if fewer than 2 readings.
func ComputeGlycemicVariability(readings []float64) float64 {
	n := len(readings)
	if n < 2 {
		return 0
	}

	var sum float64
	for _, v := range readings {
		sum += v
	}
	mean := sum / float64(n)
	if mean == 0 {
		return 0
	}

	var sumSq float64
	for _, v := range readings {
		d := v - mean
		sumSq += d * d
	}
	sd := math.Sqrt(sumSq / float64(n-1))

	return (sd / mean) * 100.0
}

// ComputeDawnPhenomenon returns true if the patient shows dawn phenomenon:
// among the last 5 FBG/PPBG pairs, at least 3 have FBG > PPBG AND at least
// 3 of the FBGs exceed 130 mg/dL.
func ComputeDawnPhenomenon(fbgs, ppbgs []float64) bool {
	n := len(fbgs)
	if n > 5 {
		fbgs = fbgs[n-5:]
		ppbgs = ppbgs[n-5:]
		n = 5
	}
	if n < 3 || len(ppbgs) < n {
		return false
	}

	fbgGtPPBG := 0
	fbgGt130 := 0
	for i := 0; i < n; i++ {
		if fbgs[i] > ppbgs[i] {
			fbgGtPPBG++
		}
		if fbgs[i] > 130 {
			fbgGt130++
		}
	}

	return fbgGtPPBG >= 3 && fbgGt130 >= 3
}

// ComputeTrigHDLRatio returns TG / HDL. Returns 0 if HDL is zero.
func ComputeTrigHDLRatio(tg, hdl float64) float64 {
	if hdl <= 0 {
		return 0
	}
	return tg / hdl
}

// ComputeRenalSlope performs OLS linear regression on eGFR readings over time,
// returning slope in mL/min/1.73m²/year. Requires at least 2 readings.
func ComputeRenalSlope(readings []TimedValue) float64 {
	n := len(readings)
	if n < 2 {
		return 0
	}

	var sumX, sumY, sumXX, sumXY float64
	for _, r := range readings {
		x := r.DaysSinceFirst / 365.25 // convert days to years
		y := r.Value
		sumX += x
		sumY += y
		sumXX += x * x
		sumXY += x * y
	}

	nf := float64(n)
	denom := nf*sumXX - sumX*sumX
	if denom == 0 {
		return 0
	}

	slope := (nf*sumXY - sumX*sumY) / denom
	return slope
}

// clamp restricts v to [min, max].
func clamp(v, min, max float64) float64 {
	if v < min {
		return min
	}
	if v > max {
		return max
	}
	return v
}

// ComputeLinearTrend returns slope in units/day via OLS regression.
// Returns 0 if fewer than 2 readings.
func ComputeLinearTrend(readings []TimedValue) float64 {
	n := len(readings)
	if n < 2 {
		return 0
	}

	var sumX, sumY, sumXX, sumXY float64
	for _, r := range readings {
		x := r.DaysSinceFirst
		y := r.Value
		sumX += x
		sumY += y
		sumXX += x * x
		sumXY += x * y
	}

	nf := float64(n)
	denom := nf*sumXX - sumX*sumX
	if denom == 0 {
		return 0
	}

	return (nf*sumXY - sumX*sumY) / denom
}

// ComputeWeightTrendPerMonth returns kg/month slope from weight readings.
func ComputeWeightTrendPerMonth(readings []TimedValue) float64 {
	slopePerDay := ComputeLinearTrend(readings)
	return slopePerDay * 30.44 // average days per month
}

// ComputeHbA1cTrendPerQuarter returns %/quarter slope from HbA1c readings.
func ComputeHbA1cTrendPerQuarter(readings []TimedValue) float64 {
	slopePerDay := ComputeLinearTrend(readings)
	return slopePerDay * 91.3 // average days per quarter
}

// ComputeSBPTrend4Weeks returns mmHg change over 4 weeks from SBP readings.
func ComputeSBPTrend4Weeks(readings []TimedValue) float64 {
	slopePerDay := ComputeLinearTrend(readings)
	return slopePerDay * 28
}

// ClassifyBPDipping classifies BP dipping from evening and morning SBP means.
// Spec §4.1: Dipper=10-20% dip, Non-dipper=0-10%, Reverse=<0%
func ClassifyBPDipping(eveningSBP, morningSBP float64) string {
	if eveningSBP <= 0 {
		return "UNKNOWN"
	}
	dipPercent := ((eveningSBP - morningSBP) / eveningSBP) * 100.0

	switch {
	case dipPercent < 0:
		return "REVERSE_DIPPER"
	case dipPercent < 10:
		return "NON_DIPPER"
	default:
		return "DIPPER"
	}
}
