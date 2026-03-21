package models

import "time"

// MealSignalRequest represents a patient meal log submission (S4).
type MealSignalRequest struct {
	MealType     string    `json:"meal_type" binding:"required"` // breakfast, lunch, dinner, snack
	Description  string    `json:"description"`
	ProteinG     float64   `json:"protein_g"`
	CarbsG       float64   `json:"carbs_g"`
	FatG         float64   `json:"fat_g"`
	CaloriesKcal float64   `json:"calories_kcal"`
	MeasuredAt   time.Time `json:"measured_at" binding:"required"`
}

// ActivitySignalRequest represents a patient activity submission (S16).
type ActivitySignalRequest struct {
	StepCount    int       `json:"step_count"`
	ExerciseType string    `json:"exercise_type"` // walking, cycling, swimming, etc.
	DurationMin  int       `json:"duration_min"`
	Intensity    string    `json:"intensity"` // light, moderate, vigorous
	MeasuredAt   time.Time `json:"measured_at" binding:"required"`
}

// WaistSignalRequest represents a waist circumference measurement (S15).
type WaistSignalRequest struct {
	ValueCm    float64   `json:"value_cm" binding:"required"`
	MeasuredAt time.Time `json:"measured_at" binding:"required"`
}

// AdherenceSignalRequest represents a medication adherence submission (S20).
type AdherenceSignalRequest struct {
	DrugClass  string    `json:"drug_class" binding:"required"`
	Taken      bool      `json:"taken"`
	MissedDose bool      `json:"missed_dose"`
	Reason     string    `json:"reason"` // forgot, side_effect, intentional, etc.
	MeasuredAt time.Time `json:"measured_at" binding:"required"`
}

// SymptomSignalRequest represents a patient-reported symptom (S18).
type SymptomSignalRequest struct {
	SymptomCode string    `json:"symptom_code" binding:"required"` // headache, dizziness, fatigue, etc.
	Severity    int       `json:"severity" binding:"required,min=1,max=10"` // 1-10 scale
	Duration    string    `json:"duration"`                        // "2 hours", "3 days"
	Description string    `json:"description"`
	MeasuredAt  time.Time `json:"measured_at" binding:"required"`
}

// AdverseEventSignalRequest represents a patient-reported adverse event (S19).
type AdverseEventSignalRequest struct {
	DrugClass   string    `json:"drug_class" binding:"required"`
	EventType   string    `json:"event_type" binding:"required"` // rash, gi_upset, cough, etc.
	Severity    string    `json:"severity" binding:"required"`   // mild, moderate, severe
	Description string    `json:"description"`
	OnsetAt     time.Time `json:"onset_at" binding:"required"`
}

// ResolutionSignalRequest represents a symptom/ADR resolution report (S21).
type ResolutionSignalRequest struct {
	OriginalEventType string    `json:"original_event_type" binding:"required"` // SYMPTOM or ADVERSE_EVENT
	OriginalEventID   string    `json:"original_event_id"`
	Resolution        string    `json:"resolution" binding:"required"` // resolved, improved, unchanged, worsened
	ResolvedAt        time.Time `json:"resolved_at" binding:"required"`
}

// HospitalisationSignalRequest represents a hospitalisation report (S22).
type HospitalisationSignalRequest struct {
	Reason       string     `json:"reason" binding:"required"`
	Facility     string     `json:"facility"`
	AdmittedAt   time.Time  `json:"admitted_at" binding:"required"`
	DischargedAt *time.Time `json:"discharged_at"`
}
