package services

import (
	"math"

	"go.uber.org/zap"
)

// AnnualAttribution holds the 12-month aggregated attribution for the Annual Health Narrative.
type AnnualAttribution struct {
	PatientID       string  `json:"patient_id"`
	TargetVar       string  `json:"target_variable"`
	TotalDelta      float64 `json:"total_delta"`
	LifestyleFrac   float64 `json:"lifestyle_fraction"`
	MedicationFrac  float64 `json:"medication_fraction"`
	UnexplainedFrac float64 `json:"unexplained_fraction"`
	Quarters        int     `json:"quarters_aggregated"`
}

type AnnualAttributionEngine struct {
	logger *zap.Logger
}

func NewAnnualAttributionEngine(logger *zap.Logger) *AnnualAttributionEngine {
	return &AnnualAttributionEngine{logger: logger}
}

// AggregateAnnual combines quarterly AttributionResults into a single annual summary.
// Fractions are weighted by absolute delta magnitude (larger improvements carry more weight).
func (e *AnnualAttributionEngine) AggregateAnnual(quarterly []AttributionResult) AnnualAttribution {
	if len(quarterly) == 0 {
		return AnnualAttribution{}
	}

	var totalDelta, weightedLS, weightedMed, weightedUnex, totalWeight float64

	for _, q := range quarterly {
		totalDelta += q.TotalDelta
		w := math.Abs(q.TotalDelta)
		if w == 0 {
			w = 1
		}
		weightedLS += q.LifestyleFrac * w
		weightedMed += q.MedicationFrac * w
		weightedUnex += q.UnexplainedFrac * w
		totalWeight += w
	}

	if totalWeight == 0 {
		totalWeight = 1
	}

	return AnnualAttribution{
		PatientID:       quarterly[0].PatientID,
		TargetVar:       quarterly[0].TargetVar,
		TotalDelta:      totalDelta,
		LifestyleFrac:   weightedLS / totalWeight,
		MedicationFrac:  weightedMed / totalWeight,
		UnexplainedFrac: weightedUnex / totalWeight,
		Quarters:        len(quarterly),
	}
}
