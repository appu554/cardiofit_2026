package services

import "kb-26-metabolic-digital-twin/internal/models"

// ConfidenceAnalysis holds the overall confidence assessment for a patient twin.
type ConfidenceAnalysis struct {
	PatientID         string                   `json:"patient_id"`
	OverallConfidence float64                  `json:"overall_confidence"`
	Variables         []VariableConfidence     `json:"variables"`
	Recommended       []RecommendedMeasurement `json:"recommended_measurements"`
}

// VariableConfidence describes confidence for a single twin variable.
type VariableConfidence struct {
	Name       string  `json:"name"`
	Tier       int     `json:"tier"`
	Confidence float64 `json:"confidence"`
}

// RecommendedMeasurement suggests a measurement to improve twin confidence.
type RecommendedMeasurement struct {
	Measurement  string  `json:"measurement"`
	ExpectedGain float64 `json:"expected_confidence_gain"`
	Priority     string  `json:"priority"`
	Reason       string  `json:"reason"`
}

// AnalyzeConfidence evaluates how well-populated a twin state is and recommends
// measurements that would most improve the twin's fidelity.
func AnalyzeConfidence(twin *models.TwinState) *ConfidenceAnalysis {
	analysis := &ConfidenceAnalysis{
		PatientID: twin.PatientID.String(),
	}

	vars := []VariableConfidence{
		{Name: "FBG", Tier: 1, Confidence: boolToConf(twin.FBG7dMean != nil, 0.95)},
		{Name: "HbA1c", Tier: 1, Confidence: boolToConf(twin.HbA1c != nil, 0.95)},
		{Name: "SBP", Tier: 1, Confidence: boolToConf(twin.SBP14dMean != nil, 0.90)},
		{Name: "eGFR", Tier: 1, Confidence: boolToConf(twin.EGFR != nil, 0.90)},
		{Name: "VisceralFatProxy", Tier: 2, Confidence: boolToConf(twin.VisceralFatProxy != nil, 0.75)},
		{Name: "GlycemicVariability", Tier: 2, Confidence: boolToConf(twin.GlycemicVariability != nil, 0.70)},
	}
	analysis.Variables = vars

	total := 0.0
	for _, v := range vars {
		total += v.Confidence
	}
	analysis.OverallConfidence = total / float64(len(vars))

	// Recommend measurements for missing high-value variables.
	if twin.HbA1c == nil {
		analysis.Recommended = append(analysis.Recommended, RecommendedMeasurement{
			Measurement: "HbA1c", ExpectedGain: 0.20, Priority: "HIGH", Reason: "No HbA1c on file",
		})
	}
	if twin.EGFR == nil {
		analysis.Recommended = append(analysis.Recommended, RecommendedMeasurement{
			Measurement: "Serum Creatinine", ExpectedGain: 0.15, Priority: "HIGH", Reason: "No eGFR available",
		})
	}
	if twin.VisceralFatProxy == nil {
		analysis.Recommended = append(analysis.Recommended, RecommendedMeasurement{
			Measurement: "Waist Circumference + Lipid Panel", ExpectedGain: 0.10, Priority: "MEDIUM", Reason: "VF proxy requires waist + TG/HDL",
		})
	}

	return analysis
}

func boolToConf(available bool, maxConf float64) float64 {
	if available {
		return maxConf
	}
	return 0.10
}
