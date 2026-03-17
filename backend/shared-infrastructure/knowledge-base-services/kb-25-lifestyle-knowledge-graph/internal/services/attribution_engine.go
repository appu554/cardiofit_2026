package services

type AttributionResult struct {
	PatientID       string  `json:"patient_id"`
	TargetVar       string  `json:"target_variable"`
	TotalDelta      float64 `json:"total_delta"`
	LifestyleFrac   float64 `json:"lifestyle_fraction"`
	MedicationFrac  float64 `json:"medication_fraction"`
	UnexplainedFrac float64 `json:"unexplained_fraction"`
}

func AttributeOutcome(patientID, targetVar string, totalDelta float64) *AttributionResult {
	return &AttributionResult{
		PatientID:       patientID,
		TargetVar:       targetVar,
		TotalDelta:      totalDelta,
		LifestyleFrac:   0.5,
		MedicationFrac:  0.4,
		UnexplainedFrac: 0.1,
	}
}
