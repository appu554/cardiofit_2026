package services

import "math"

// CGMGlucoseInput collects the signals needed for CGM-aware glucose domain scoring.
// When HasCGM is true and SufficientData is true, the scorer uses TIR/CV/GRI/TBR.
// Otherwise it falls back to snapshot scoring from FBG/HbA1c.
// CGMGlucoseScoringResult is the output of ComputeGlucoseDomainScore.
type CGMGlucoseScoringResult struct {
	Score      float64 `json:"score"`
	Confidence string  `json:"confidence"` // HIGH, MODERATE, LOW
	DataSource string  `json:"data_source"` // CGM, SNAPSHOT
	GMIDiscrepancyDetected bool `json:"gmi_discrepancy_detected,omitempty"`
}

type CGMGlucoseInput struct {
	HasCGM         bool
	SufficientData bool
	TIRPct         float64
	CVPct          float64
	GRI            float64
	TBRL2Pct       float64
	GMI            float64  // CGM-derived GMI (only meaningful when HasCGM)
	FBG            *float64
	PPBG           *float64
	HbA1c          *float64 // lab HbA1c — used for cross-check against GMI
}

// GMIDiscrepancyResult reports whether the CGM-derived GMI diverges
// significantly from the laboratory-measured HbA1c.
type GMIDiscrepancyResult struct {
	GMI      float64 `json:"gmi"`
	LabHbA1c float64 `json:"lab_hba1c"`
	Delta    float64 `json:"delta"`
	Flagged  bool    `json:"flagged"`
	Reason   string  `json:"reason,omitempty"`
}

// CGM-aware glucose domain sub-score weights.
const (
	cgmWeightTIR  = 0.40
	cgmWeightCV   = 0.20
	cgmWeightGRI  = 0.25
	cgmWeightTBR  = 0.15
)

// ComputeGlucoseDomainScore returns a 0-100 score for the glucose domain.
// Higher is better (well-managed glucose).
//
// When CGM data is available and sufficient:
//   - TIR component  (40%): linear 0→100 mapped from 0→70% TIR
//   - CV component   (20%): 100 if CV ≤36%, linear decay to 0 at CV=60%
//   - GRI component  (25%): inverse — 100 at GRI=0, 0 at GRI=100
//   - TBR safety     (15%): 100 if TBRL2=0, hard penalty for severe hypo
//
// When no CGM: snapshot scoring from FBG and HbA1c.
//
// GMI-HbA1c cross-check: if CGM-based GMI and lab HbA1c diverge by >0.5%,
// the confidence is downgraded to MODERATE (data integrity concern —
// hemoglobin variant, anemia, or assay interference possible).
func ComputeGlucoseDomainScore(input CGMGlucoseInput) float64 {
	return ComputeGlucoseDomainScoreWithConfidence(input).Score
}

// ComputeGlucoseDomainScoreWithConfidence returns score + confidence + discrepancy flag.
func ComputeGlucoseDomainScoreWithConfidence(input CGMGlucoseInput) CGMGlucoseScoringResult {
	if input.HasCGM && input.SufficientData {
		score := computeCGMScore(input)
		confidence := "HIGH"
		discrepancy := false

		// Cross-check GMI against lab HbA1c when both available
		if input.GMI > 0 && input.HbA1c != nil {
			disc := DetectGMIDiscrepancy(input.GMI, *input.HbA1c)
			if disc.Flagged {
				confidence = "MODERATE"
				discrepancy = true
			}
		}

		return CGMGlucoseScoringResult{
			Score:                  score,
			Confidence:             confidence,
			DataSource:             "CGM",
			GMIDiscrepancyDetected: discrepancy,
		}
	}

	score := computeSnapshotScore(input)
	confidence := "MODERATE"
	if input.FBG == nil && input.HbA1c == nil {
		confidence = "LOW"
	}
	return CGMGlucoseScoringResult{
		Score:      score,
		Confidence: confidence,
		DataSource: "SNAPSHOT",
	}
}

func computeCGMScore(input CGMGlucoseInput) float64 {
	// TIR component: linear 0-100 mapped from 0-70% TIR (capped at 70)
	tirClamped := math.Min(input.TIRPct, 70.0)
	tirScore := (tirClamped / 70.0) * 100.0

	// CV component: 100 if ≤36%, linear decay to 0 at 60%
	var cvScore float64
	switch {
	case input.CVPct <= 36.0:
		cvScore = 100.0
	case input.CVPct >= 60.0:
		cvScore = 0.0
	default:
		cvScore = (1.0 - (input.CVPct-36.0)/(60.0-36.0)) * 100.0
	}

	// GRI component: inverse linear — 100 at 0, 0 at 100
	griClamped := math.Min(math.Max(input.GRI, 0), 100.0)
	griScore := (1.0 - griClamped/100.0) * 100.0

	// TBR safety: 100 if TBRL2=0, severe penalty otherwise
	var tbrScore float64
	switch {
	case input.TBRL2Pct <= 0:
		tbrScore = 100.0
	case input.TBRL2Pct <= 1.0:
		tbrScore = 50.0
	default:
		// Each additional % above 1 loses 20 points from 50
		tbrScore = math.Max(0, 50.0-((input.TBRL2Pct-1.0)*20.0))
	}

	composite := cgmWeightTIR*tirScore +
		cgmWeightCV*cvScore +
		cgmWeightGRI*griScore +
		cgmWeightTBR*tbrScore

	return math.Max(0, math.Min(100, composite))
}

func computeSnapshotScore(input CGMGlucoseInput) float64 {
	var total, weights float64

	// FBG component: 100 at ≤100 mg/dL, 0 at ≥200 mg/dL
	if input.FBG != nil {
		fbg := *input.FBG
		var fbgScore float64
		switch {
		case fbg <= 100:
			fbgScore = 100.0
		case fbg >= 200:
			fbgScore = 0.0
		default:
			fbgScore = (1.0 - (fbg-100.0)/100.0) * 100.0
		}
		total += fbgScore * 0.40
		weights += 0.40
	}

	// HbA1c component: 100 at ≤5.7%, 0 at ≥10%
	if input.HbA1c != nil {
		a1c := *input.HbA1c
		var a1cScore float64
		switch {
		case a1c <= 5.7:
			a1cScore = 100.0
		case a1c >= 10.0:
			a1cScore = 0.0
		default:
			a1cScore = (1.0 - (a1c-5.7)/(10.0-5.7)) * 100.0
		}
		total += a1cScore * 0.60
		weights += 0.60
	}

	if weights == 0 {
		return 50.0 // indeterminate
	}
	return math.Max(0, math.Min(100, total/weights))
}

// DetectGMIDiscrepancy flags when |GMI - lab HbA1c| > 0.5%.
// A significant discrepancy may indicate glycation rate variants,
// haemoglobinopathies, or non-steady-state glucose conditions.
func DetectGMIDiscrepancy(gmi, labHbA1c float64) GMIDiscrepancyResult {
	delta := math.Abs(gmi - labHbA1c)
	result := GMIDiscrepancyResult{
		GMI:      gmi,
		LabHbA1c: labHbA1c,
		Delta:    math.Round(delta*100) / 100, // round to 2 dp
	}
	if delta > 0.5 {
		result.Flagged = true
		result.Reason = "GMI-HbA1c discrepancy exceeds 0.5% threshold; consider glycation rate variant or haemoglobinopathy screen"
	}
	return result
}
