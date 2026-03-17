package services

type ExerciseRx struct {
	PatientID    string  `json:"patient_id"`
	Prescription string  `json:"prescription"`
	METTarget    float64 `json:"met_target"`
	MinutesWeek  int     `json:"minutes_per_week"`
	Compliance   float64 `json:"compliance"`
}

func GenerateExerciseRx(patientID string) *ExerciseRx {
	return &ExerciseRx{
		PatientID:    patientID,
		Prescription: "150 min/week moderate aerobic + 2 sessions resistance",
		METTarget:    3.5,
		MinutesWeek:  150,
		Compliance:   0.0,
	}
}
