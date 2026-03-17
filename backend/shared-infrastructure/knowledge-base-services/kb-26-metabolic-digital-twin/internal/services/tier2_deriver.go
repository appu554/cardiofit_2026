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
	sd := math.Sqrt(sumSq / float64(n))

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
