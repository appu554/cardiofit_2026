package models

import "time"

// ObservationEvent represents an incoming lab or vital-sign observation.
type ObservationEvent struct {
	PatientID string    `json:"patient_id"`
	Type      string    `json:"type"`
	Code      string    `json:"code"`
	Value     float64   `json:"value"`
	Unit      string    `json:"unit"`
	Timestamp time.Time `json:"timestamp"`
}

// CheckinEvent represents a patient self-reported daily check-in.
type CheckinEvent struct {
	PatientID    string    `json:"patient_id"`
	MealQuality  float64   `json:"meal_quality,omitempty"`
	ExerciseDone bool      `json:"exercise_done,omitempty"`
	StepCount    int       `json:"step_count,omitempty"`
	Timestamp    time.Time `json:"timestamp"`
}

// MedChangeEvent represents a medication change notification.
type MedChangeEvent struct {
	PatientID string    `json:"patient_id"`
	DrugClass string    `json:"drug_class"`
	Action    string    `json:"action"`
	NewDose   string    `json:"new_dose,omitempty"`
	Timestamp time.Time `json:"timestamp"`
}
