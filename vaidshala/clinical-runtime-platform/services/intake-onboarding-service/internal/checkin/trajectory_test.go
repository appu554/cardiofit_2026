package checkin

import (
	"testing"
)

func TestComputeTrajectory_Stable(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 95, "ppbg": 130, "hba1c": 6.5,
			"systolic_bp": 125, "diastolic_bp": 75, "egfr": 75,
			"weight": 78, "medication_adherence": 90,
			"physical_activity_minutes": 180, "sleep_hours": 7.5,
			"symptom_severity": 1, "side_effects": 0,
		},
		PreviousValues: map[string]float64{
			"fbg": 110, "ppbg": 155, "hba1c": 7.2,
			"systolic_bp": 140, "diastolic_bp": 85, "egfr": 70,
			"weight": 80, "medication_adherence": 85,
			"physical_activity_minutes": 150, "sleep_hours": 7.0,
			"symptom_severity": 3, "side_effects": 2,
		},
		CycleNumber:   3,
		SlotsFilled:   12,
		SlotsRequired: 12,
		DaysSinceSchedule: 0,
	}

	result := ComputeTrajectory(input)
	if result.Signal != STABLE {
		t.Fatalf("expected STABLE, got %s (improving=%v, worsening=%v)", result.Signal, result.ImprovingSlots, result.WorseningSlots)
	}
	if result.Confidence < 0.5 {
		t.Fatalf("expected confidence >= 0.5, got %.2f", result.Confidence)
	}
}

func TestComputeTrajectory_Fragile(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 130, "ppbg": 180, "hba1c": 7.8,
			"systolic_bp": 120, "diastolic_bp": 75, "egfr": 72,
			"weight": 79, "medication_adherence": 88,
			"physical_activity_minutes": 160, "sleep_hours": 7.5,
			"symptom_severity": 2, "side_effects": 1,
		},
		PreviousValues: map[string]float64{
			"fbg": 100, "ppbg": 135, "hba1c": 6.8,
			"systolic_bp": 125, "diastolic_bp": 78, "egfr": 70,
			"weight": 80, "medication_adherence": 85,
			"physical_activity_minutes": 150, "sleep_hours": 7.0,
			"symptom_severity": 1, "side_effects": 0,
		},
		CycleNumber:   4,
		SlotsFilled:   12,
		SlotsRequired: 12,
		DaysSinceSchedule: 1,
	}

	result := ComputeTrajectory(input)
	if result.Signal != FRAGILE && result.Signal != FAILURE {
		t.Fatalf("expected FRAGILE or FAILURE, got %s", result.Signal)
	}
}

func TestComputeTrajectory_Failure(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 170, "ppbg": 240, "hba1c": 10.5,
			"systolic_bp": 165, "diastolic_bp": 100, "egfr": 35,
			"weight": 95, "medication_adherence": 30,
			"physical_activity_minutes": 20, "sleep_hours": 4,
			"symptom_severity": 8, "side_effects": 7,
		},
		PreviousValues: map[string]float64{
			"fbg": 100, "ppbg": 135, "hba1c": 6.8,
			"systolic_bp": 125, "diastolic_bp": 75, "egfr": 70,
			"weight": 80, "medication_adherence": 90,
			"physical_activity_minutes": 200, "sleep_hours": 8,
			"symptom_severity": 1, "side_effects": 0,
		},
		CycleNumber:   5,
		SlotsFilled:   12,
		SlotsRequired: 12,
		DaysSinceSchedule: 0,
	}

	result := ComputeTrajectory(input)
	if result.Signal != FAILURE {
		t.Fatalf("expected FAILURE, got %s (worsening=%v)", result.Signal, result.WorseningSlots)
	}
}

func TestComputeTrajectory_Disengage(t *testing.T) {
	// 0 slots filled, 5 days late
	input := TrajectoryInput{
		CurrentValues:     map[string]float64{},
		PreviousValues:    map[string]float64{},
		CycleNumber:       2,
		SlotsFilled:       0,
		SlotsRequired:     12,
		DaysSinceSchedule: 5,
	}

	result := ComputeTrajectory(input)
	if result.Signal != DISENGAGE {
		t.Fatalf("expected DISENGAGE, got %s", result.Signal)
	}
}

func TestComputeTrajectory_ConsecutiveFragileEscalation(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 125, "ppbg": 170, "hba1c": 7.5,
			"systolic_bp": 118, "diastolic_bp": 74, "egfr": 72,
			"weight": 79, "medication_adherence": 87,
			"physical_activity_minutes": 160, "sleep_hours": 7.5,
			"symptom_severity": 2, "side_effects": 1,
		},
		PreviousValues: map[string]float64{
			"fbg": 100, "ppbg": 135, "hba1c": 6.8,
			"systolic_bp": 122, "diastolic_bp": 76, "egfr": 70,
			"weight": 80, "medication_adherence": 85,
			"physical_activity_minutes": 150, "sleep_hours": 7.0,
			"symptom_severity": 1, "side_effects": 0,
		},
		CycleNumber:        4,
		SlotsFilled:        12,
		SlotsRequired:      12,
		DaysSinceSchedule:  1,
		PreviousTrajectory: FRAGILE,
	}

	result := ComputeTrajectory(input)
	// With consecutive FRAGILE, should not be STABLE
	if result.Signal == STABLE {
		t.Fatalf("expected escalation from consecutive FRAGILE, got STABLE")
	}
}

func TestComputeTrajectory_FirstCheckin(t *testing.T) {
	input := TrajectoryInput{
		CurrentValues: map[string]float64{
			"fbg": 95, "ppbg": 130, "hba1c": 6.5,
			"systolic_bp": 125, "diastolic_bp": 75, "egfr": 75,
			"weight": 78, "medication_adherence": 90,
			"physical_activity_minutes": 180, "sleep_hours": 7.5,
			"symptom_severity": 1, "side_effects": 0,
		},
		BaselineValues: map[string]float64{
			"fbg": 140, "ppbg": 200, "hba1c": 8.5,
			"systolic_bp": 155, "diastolic_bp": 95, "egfr": 55,
			"weight": 85, "medication_adherence": 60,
			"physical_activity_minutes": 30, "sleep_hours": 5,
			"symptom_severity": 6, "side_effects": 4,
		},
		CycleNumber:   1,
		SlotsFilled:   12,
		SlotsRequired: 12,
		DaysSinceSchedule: 0,
	}

	result := ComputeTrajectory(input)
	if result.Signal != STABLE {
		t.Fatalf("expected STABLE for good improvement from baseline, got %s", result.Signal)
	}
}

func TestScoreSlotChange_WeightLoss(t *testing.T) {
	target := ClinicalTarget{
		SlotName:       "weight",
		Domain:         "anthropometric",
		LowOptimal:     50,
		HighOptimal:    90,
		LowCritical:    30,
		HighCritical:   200,
		ImprovementDir: "lower",
	}

	// 5% weight loss (80 → 76): should score > 0.7
	score5pctLoss := scoreSlotChange(76, 80, target)
	if score5pctLoss <= 0.7 {
		t.Fatalf("expected 5%% weight loss score > 0.7, got %.3f", score5pctLoss)
	}

	// 10% weight gain (80 → 88): should score < 0.3
	score10pctGain := scoreSlotChange(88, 80, target)
	if score10pctGain >= 0.3 {
		t.Fatalf("expected 10%% weight gain score < 0.3, got %.3f", score10pctGain)
	}
}
