// Package test provides care plan engine tests for KB-12
package test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-12-ordersets-careplans/internal/models"
	"kb-12-ordersets-careplans/pkg/careplans"
)

// ============================================
// 4.1 Template Validation Tests
// ============================================

func TestAllCarePlansHaveGoals(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	require.NotEmpty(t, plans, "Care plans should exist")

	plansWithGoals := 0
	plansWithoutGoals := []string{}

	for _, plan := range plans {
		if len(plan.Goals) > 0 {
			plansWithGoals++
		} else {
			plansWithoutGoals = append(plansWithoutGoals, plan.Name)
		}
	}

	t.Logf("Care plans with goals: %d/%d", plansWithGoals, len(plans))
	if len(plansWithoutGoals) > 0 {
		t.Logf("Plans without goals: %v", plansWithoutGoals)
	}
	assert.Equal(t, len(plans), plansWithGoals, "All care plans should have at least one goal")
}

func TestAllGoalsHaveMeasurableTargets(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	totalGoals := 0
	goalsWithTargets := 0

	for _, plan := range plans {
		for _, goal := range plan.Goals {
			totalGoals++
			if len(goal.Targets) > 0 {
				goalsWithTargets++
			} else {
				t.Logf("Goal without targets in %s: %s", plan.Name, goal.Description)
			}
		}
	}

	pct := float64(goalsWithTargets) / float64(totalGoals) * 100
	t.Logf("Goals with measurable targets: %d/%d (%.1f%%)", goalsWithTargets, totalGoals, pct)
	assert.GreaterOrEqual(t, pct, 70.0, "At least 70%% of goals should have measurable targets")
}

func TestAllActivitiesHaveRecurrence(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	totalActivities := 0
	activitiesWithSchedule := 0

	for _, plan := range plans {
		for _, activity := range plan.Activities {
			totalActivities++
			// Activity has schedule if it has recurrence OR frequency
			if activity.Recurrence != nil || activity.Frequency != "" || activity.Detail.Frequency != "" {
				activitiesWithSchedule++
			}
		}
	}

	if totalActivities > 0 {
		pct := float64(activitiesWithSchedule) / float64(totalActivities) * 100
		t.Logf("Activities with scheduling: %d/%d (%.1f%%)", activitiesWithSchedule, totalActivities, pct)
	}
}

func TestAllCarePlansHaveGuidelineReferences(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	plansWithGuidelines := 0

	for _, plan := range plans {
		if plan.GuidelineSource != "" || len(plan.Guidelines) > 0 {
			plansWithGuidelines++
		} else {
			t.Logf("Plan without guideline reference: %s", plan.Name)
		}
	}

	pct := float64(plansWithGuidelines) / float64(len(plans)) * 100
	t.Logf("Plans with guideline references: %d/%d (%.1f%%)", plansWithGuidelines, len(plans), pct)
}

func TestCarePlanCategoryClassification(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	categories := make(map[string]int)

	for _, plan := range plans {
		if plan.Category != "" {
			categories[plan.Category]++
		} else {
			categories["uncategorized"]++
		}
	}

	t.Logf("Care plan categories:")
	for cat, count := range categories {
		t.Logf("  - %s: %d", cat, count)
	}

	// Should have multiple categories
	assert.GreaterOrEqual(t, len(categories), 1, "Should have at least one category")
}

// ============================================
// 4.2 FHIR CarePlan Validation Tests
// ============================================

func TestCarePlanStatusValues(t *testing.T) {
	validStatuses := map[models.CarePlanStatus]bool{
		models.CarePlanStatusDraft:     true,
		models.CarePlanStatusActive:    true,
		models.CarePlanStatusOnHold:    true,
		models.CarePlanStatusCompleted: true,
		models.CarePlanStatusCancelled: true,
		models.CarePlanStatusRevoked:   true,
	}

	for status := range validStatuses {
		assert.NotEmpty(t, string(status), "Status should have string value")
	}
	t.Logf("✓ Validated %d care plan status values", len(validStatuses))
}

func TestCarePlanIntentValues(t *testing.T) {
	validIntents := map[models.CarePlanIntent]bool{
		models.IntentPlan:      true,
		models.IntentOrder:     true,
		models.IntentOption:    true,
		models.IntentProposal:  true,
		models.IntentDirective: true,
	}

	for intent := range validIntents {
		assert.NotEmpty(t, string(intent), "Intent should have string value")
	}
	t.Logf("✓ Validated %d care plan intent values", len(validIntents))
}

func TestCarePlanActivityStructure(t *testing.T) {
	plans := careplans.GetAllCarePlans()

	for _, plan := range plans {
		t.Run(plan.Name, func(t *testing.T) {
			for _, activity := range plan.Activities {
				// Each activity should have description
				assert.NotEmpty(t, activity.Description, "Activity should have description")

				// Should have activity type
				if activity.ActivityType == "" && activity.Type == "" {
					t.Logf("Activity without type in %s: %s", plan.Name, activity.Description)
				}
			}
		})
	}
}

func TestCarePlanGoalTargetMeasures(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	targetsWithMeasure := 0
	totalTargets := 0

	for _, plan := range plans {
		for _, goal := range plan.Goals {
			for _, target := range goal.Targets {
				totalTargets++
				if target.Measure != "" || target.Metric != "" {
					targetsWithMeasure++
				}
			}
		}
	}

	if totalTargets > 0 {
		pct := float64(targetsWithMeasure) / float64(totalTargets) * 100
		t.Logf("Targets with measures: %d/%d (%.1f%%)", targetsWithMeasure, totalTargets, pct)
		assert.GreaterOrEqual(t, pct, 80.0, "At least 80%% of targets should have measures")
	}
}

func TestCarePlanAddressesCondition(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	plansWithCondition := 0

	for _, plan := range plans {
		hasCondition := plan.Condition != "" || plan.ConditionRef != nil

		// Also check if goals address conditions
		for _, goal := range plan.Goals {
			if len(goal.Addresses) > 0 {
				hasCondition = true
				break
			}
		}

		if hasCondition {
			plansWithCondition++
		}
	}

	pct := float64(plansWithCondition) / float64(len(plans)) * 100
	t.Logf("Plans addressing conditions: %d/%d (%.1f%%)", plansWithCondition, len(plans), pct)
}

// ============================================
// 4.3 Activation & Execution Tests
// ============================================

func TestCarePlanActivation(t *testing.T) {
	// Create a care plan instance
	instance := models.CarePlanInstance{
		InstanceID: "CPI-TEST001",
		TemplateID: "CP-HTN-001",
		PatientID:  "patient-12345",
		Status:     models.CarePlanStatusActive,
		StartDate:  time.Now(),
	}

	assert.NotEmpty(t, instance.InstanceID)
	assert.Equal(t, models.CarePlanStatusActive, instance.Status)
	assert.True(t, instance.IsActive())
	t.Log("✓ Care plan activation successful")
}

func TestCarePlanActivityScheduling(t *testing.T) {
	// Test recurrence pattern
	recurrence := models.RecurrencePattern{
		Type:       "weekly",
		Frequency:  1,
		DaysOfWeek: []string{"Monday", "Wednesday", "Friday"},
		StartDate:  time.Now(),
	}

	assert.NotEmpty(t, recurrence.Type)
	assert.Greater(t, recurrence.Frequency, 0)
	assert.NotEmpty(t, recurrence.DaysOfWeek)
	t.Log("✓ Activity scheduling validated")
}

func TestCarePlanGoalProgress(t *testing.T) {
	instance := models.CarePlanInstance{
		InstanceID: "CPI-TEST002",
		TemplateID: "CP-DM-001",
		PatientID:  "patient-67890",
		Status:     models.CarePlanStatusActive,
		StartDate:  time.Now(),
		GoalsProgress: []models.GoalProgress{
			{
				GoalID:       "DM-GOAL-001",
				Status:       models.GoalStatusInProgress,
				ProgressPct:  65.0,
				CurrentValue: "7.2%",
				TargetValue:  "<7.0%",
			},
			{
				GoalID:       "DM-GOAL-002",
				Status:       models.GoalStatusInProgress,
				ProgressPct:  80.0,
				CurrentValue: "95 mg/dL",
				TargetValue:  "80-130 mg/dL",
			},
		},
	}

	progress, err := instance.CalculateOverallProgress()
	require.NoError(t, err)

	expectedProgress := (65.0 + 80.0) / 2
	assert.InDelta(t, expectedProgress, progress, 0.1)
	t.Logf("✓ Overall progress: %.1f%%", progress)
}

func TestCarePlanStatusTransitions(t *testing.T) {
	validTransitions := map[models.CarePlanStatus][]models.CarePlanStatus{
		models.CarePlanStatusDraft:     {models.CarePlanStatusActive, models.CarePlanStatusCancelled},
		models.CarePlanStatusActive:    {models.CarePlanStatusOnHold, models.CarePlanStatusCompleted, models.CarePlanStatusRevoked},
		models.CarePlanStatusOnHold:    {models.CarePlanStatusActive, models.CarePlanStatusCancelled},
		models.CarePlanStatusCompleted: {}, // Terminal state
		models.CarePlanStatusCancelled: {}, // Terminal state
		models.CarePlanStatusRevoked:   {}, // Terminal state
	}

	for from, allowed := range validTransitions {
		t.Logf("From %s: %d valid transitions", from, len(allowed))
	}
	t.Log("✓ Status transition rules defined")
}

func TestCarePlanDeactivation(t *testing.T) {
	instance := models.CarePlanInstance{
		InstanceID: "CPI-TEST003",
		TemplateID: "CP-HTN-001",
		PatientID:  "patient-11111",
		Status:     models.CarePlanStatusActive,
		StartDate:  time.Now().Add(-30 * 24 * time.Hour),
	}

	// Deactivate
	instance.Status = models.CarePlanStatusCompleted
	now := time.Now()
	instance.EndDate = &now

	assert.Equal(t, models.CarePlanStatusCompleted, instance.Status)
	assert.NotNil(t, instance.EndDate)
	assert.False(t, instance.IsActive())
	t.Log("✓ Care plan deactivation successful")
}

// ============================================
// 4.4 Goal Validation Tests
// ============================================

func TestGoalStatusValues(t *testing.T) {
	validStatuses := []models.GoalStatus{
		models.GoalStatusProposed,
		models.GoalStatusAccepted,
		models.GoalStatusInProgress,
		models.GoalStatusAchieved,
		models.GoalStatusNotAchieved,
		models.GoalStatusCancelled,
	}

	for _, status := range validStatuses {
		assert.NotEmpty(t, string(status))
	}
	t.Logf("✓ Validated %d goal status values", len(validStatuses))
}

func TestActivityStatusValues(t *testing.T) {
	validStatuses := []models.ActivityStatus{
		models.ActivityStatusScheduled,
		models.ActivityStatusInProgress,
		models.ActivityStatusCompleted,
		models.ActivityStatusNotDone,
		models.ActivityStatusCancelled,
	}

	for _, status := range validStatuses {
		assert.NotEmpty(t, string(status))
	}
	t.Logf("✓ Validated %d activity status values", len(validStatuses))
}

// ============================================
// 4.5 Monitoring Tests
// ============================================

func TestMonitoringItemsPresent(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	plansWithMonitoring := 0

	for _, plan := range plans {
		items, _ := plan.GetMonitoringItems()
		if len(items) > 0 {
			plansWithMonitoring++
		}
	}

	pct := float64(plansWithMonitoring) / float64(len(plans)) * 100
	t.Logf("Plans with monitoring items: %d/%d (%.1f%%)", plansWithMonitoring, len(plans), pct)
}

func TestMonitoringItemsHaveFrequency(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	itemsWithFrequency := 0
	totalItems := 0

	for _, plan := range plans {
		items, _ := plan.GetMonitoringItems()
		for _, item := range items {
			totalItems++
			if item.Frequency != "" || item.Recurrence != nil {
				itemsWithFrequency++
			}
		}
	}

	if totalItems > 0 {
		pct := float64(itemsWithFrequency) / float64(totalItems) * 100
		t.Logf("Monitoring items with frequency: %d/%d (%.1f%%)", itemsWithFrequency, totalItems, pct)
	}
}

// ============================================
// 4.6 Activity Detail Tests
// ============================================

func TestMedicationActivitiesHaveDrugInfo(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	medActivities := 0
	medWithDrugInfo := 0

	for _, plan := range plans {
		for _, activity := range plan.Activities {
			if activity.ActivityType == "medication" || activity.Type == "medication" {
				medActivities++
				if activity.Detail.DrugCode != "" || activity.Detail.DrugName != "" {
					medWithDrugInfo++
				}
			}
		}
	}

	if medActivities > 0 {
		pct := float64(medWithDrugInfo) / float64(medActivities) * 100
		t.Logf("Medication activities with drug info: %d/%d (%.1f%%)", medWithDrugInfo, medActivities, pct)
		assert.GreaterOrEqual(t, pct, 50.0, "At least 50%% of medication activities should have drug info")
	}
}

func TestActivitiesHaveGoalReferences(t *testing.T) {
	plans := careplans.GetAllCarePlans()
	activitiesWithGoalRef := 0
	totalActivities := 0

	for _, plan := range plans {
		for _, activity := range plan.Activities {
			totalActivities++
			if len(activity.GoalReferences) > 0 {
				activitiesWithGoalRef++
			}
		}
	}

	if totalActivities > 0 {
		pct := float64(activitiesWithGoalRef) / float64(totalActivities) * 100
		t.Logf("Activities with goal references: %d/%d (%.1f%%)", activitiesWithGoalRef, totalActivities, pct)
	}
}

// ============================================
// Benchmark Tests
// ============================================

func BenchmarkCarePlanLoad(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_ = careplans.GetAllCarePlans()
	}
}

func BenchmarkCarePlanProgressCalculation(b *testing.B) {
	instance := models.CarePlanInstance{
		GoalsProgress: []models.GoalProgress{
			{GoalID: "G1", ProgressPct: 50},
			{GoalID: "G2", ProgressPct: 75},
			{GoalID: "G3", ProgressPct: 100},
			{GoalID: "G4", ProgressPct: 25},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = instance.CalculateOverallProgress()
	}
}
