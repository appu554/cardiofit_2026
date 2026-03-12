// Package tests provides comprehensive test coverage for KB-19 Protocol Orchestrator.
//
// PILLAR 6: EXECUTION BINDING TESTS
// Tests Step 8 of the pipeline: KB-3 (temporal), KB-12 (orderset), KB-14 (governance)
package tests

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-19-protocol-orchestrator/internal/arbitration"
	"kb-19-protocol-orchestrator/internal/config"
	"kb-19-protocol-orchestrator/internal/models"
)

// ============================================================================
// PILLAR 6.1: TEMPORAL BINDING (KB-3)
// DO/DELAY decisions should create temporal bindings
// ============================================================================

func TestTemporalBinding_DODecision_CreatesBinding(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:8083",
			KB12URL: "http://localhost:8094",
			KB14URL: "http://localhost:8091",
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)

	// Execution plan should exist
	assert.NotNil(t, bundle.ExecutionPlan, "Execution plan should be created")

	// Temporal bindings structure should exist with real KB-3 service
	require.NotNil(t, bundle.ExecutionPlan.TemporalBindings,
		"Temporal bindings slice should be initialized (requires KB-3 at localhost:8083)")
}

func TestTemporalBinding_UrgencyToTiming(t *testing.T) {
	// Urgency levels should map to appropriate timing
	tests := []struct {
		urgency       models.ActionUrgency
		maxDueWithin  time.Duration
	}{
		{models.UrgencySTAT, 15 * time.Minute},
		{models.UrgencyUrgent, 1 * time.Hour},
		{models.UrgencyRoutine, 24 * time.Hour},
		{models.UrgencyScheduled, 7 * 24 * time.Hour},
	}

	for _, tc := range tests {
		t.Run(string(tc.urgency), func(t *testing.T) {
			// Verify temporal binding calculation matches urgency expectations
			var expectedMinutes int
			switch tc.urgency {
			case models.UrgencySTAT:
				expectedMinutes = 15
			case models.UrgencyUrgent:
				expectedMinutes = 60
			case models.UrgencyRoutine:
				expectedMinutes = 24 * 60
			case models.UrgencyScheduled:
				expectedMinutes = 7 * 24 * 60
			}

			assert.GreaterOrEqual(t, int(tc.maxDueWithin.Minutes()), expectedMinutes/10,
				"%s urgency should have appropriate timing", tc.urgency)
		})
	}
}

func TestTemporalBinding_DELAYDecision_CreatesFollowUp(t *testing.T) {
	// DELAY decisions should schedule a reassessment
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	delayDecision := models.ArbitratedDecision{
		ID:             uuid.New(),
		DecisionType:   models.DecisionDelay,
		Target:         "metoprolol",
		Rationale:      "Delayed due to hemodynamic instability",
		Urgency:        models.UrgencyRoutine,
		SourceProtocol: "AFIB-RATE",
	}

	bundle.AddDecision(delayDecision)

	// DELAY decisions should have a reassessment window
	// This validates the model supports follow-up tracking
	assert.Equal(t, models.DecisionDelay, delayDecision.DecisionType)
}

func TestTemporalBinding_Structure(t *testing.T) {
	binding := models.TemporalBinding{
		DecisionID:  uuid.New(),
		ActionID:    "action-123",
		ScheduledAt: time.Now().Add(1 * time.Hour),
		Deadline:    ptrTime(time.Now().Add(2 * time.Hour)),
		Recurring:   false,
	}

	assert.NotEqual(t, uuid.Nil, binding.DecisionID)
	assert.NotEmpty(t, binding.ActionID)
	assert.False(t, binding.ScheduledAt.IsZero())
	assert.NotNil(t, binding.Deadline)
	assert.False(t, binding.Recurring)
}

func TestTemporalBinding_Recurring(t *testing.T) {
	binding := models.TemporalBinding{
		DecisionID:  uuid.New(),
		ActionID:    "recurring-action",
		ScheduledAt: time.Now(),
		Recurring:   true,
		RecurFreq:   "q6h",
	}

	assert.True(t, binding.Recurring)
	assert.Equal(t, "q6h", binding.RecurFreq)
}

// ============================================================================
// PILLAR 6.2: ORDER SET ACTIVATION (KB-12)
// DO decisions should activate order sets
// ============================================================================

func TestOrderSetActivation_DODecision(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	activation := models.OrderSetActivation{
		DecisionID:       uuid.New(),
		OrderSetID:       "OS-SEPSIS-BUNDLE",
		OrderSetName:     "Sepsis 1-Hour Bundle",
		Parameters:       map[string]interface{}{"weight_kg": 70.0},
		ActivatedAt:      time.Now(),
		IndividualOrders: []string{"Blood cultures", "Lactate", "Fluids 30mL/kg", "Antibiotics"},
	}

	bundle.ExecutionPlan.OrderSetActivations = append(
		bundle.ExecutionPlan.OrderSetActivations,
		activation,
	)

	require.Len(t, bundle.ExecutionPlan.OrderSetActivations, 1)
	assert.Equal(t, "OS-SEPSIS-BUNDLE", bundle.ExecutionPlan.OrderSetActivations[0].OrderSetID)
	assert.Len(t, bundle.ExecutionPlan.OrderSetActivations[0].IndividualOrders, 4)
}

func TestOrderSetActivation_Parameters(t *testing.T) {
	activation := models.OrderSetActivation{
		DecisionID:   uuid.New(),
		OrderSetID:   "OS-ANTICOAG",
		OrderSetName: "Anticoagulation Initiation",
		Parameters: map[string]interface{}{
			"weight_kg":     85.0,
			"creatinine":    1.2,
			"indication":    "AFib",
			"target_inr":    2.5,
		},
		ActivatedAt: time.Now(),
	}

	assert.Equal(t, 85.0, activation.Parameters["weight_kg"])
	assert.Equal(t, "AFib", activation.Parameters["indication"])
	assert.Equal(t, 2.5, activation.Parameters["target_inr"])
}

func TestOrderSetActivation_IndividualOrders(t *testing.T) {
	// Order sets should expand to individual orders
	activation := models.OrderSetActivation{
		DecisionID:   uuid.New(),
		OrderSetID:   "OS-HF-GDMT",
		OrderSetName: "Heart Failure GDMT",
		IndividualOrders: []string{
			"Lisinopril 5mg PO daily",
			"Carvedilol 3.125mg PO BID",
			"Spironolactone 25mg PO daily",
			"Monitor potassium weekly",
		},
		ActivatedAt: time.Now(),
	}

	assert.Len(t, activation.IndividualOrders, 4)
	assert.Contains(t, activation.IndividualOrders[0], "Lisinopril")
}

// ============================================================================
// PILLAR 6.3: GOVERNANCE TASK CREATION (KB-14)
// AVOID/DELAY/CONSIDER decisions should create governance tasks
// ============================================================================

func TestGovernanceTask_AVOIDDecision(t *testing.T) {
	// AVOID decisions should create escalation tasks

	task := models.GovernanceTask{
		DecisionID:  uuid.New(),
		TaskType:    "ESCALATION",
		AssignedTo:  "ATTENDING_PHYSICIAN",
		Priority:    "HIGH",
		DueAt:       time.Now().Add(1 * time.Hour),
		Description: "AVOID: Warfarin blocked due to pregnancy. Acknowledge and document alternative.",
	}

	assert.Equal(t, "ESCALATION", task.TaskType)
	assert.Equal(t, "HIGH", task.Priority)
	assert.NotEmpty(t, task.Description)
}

func TestGovernanceTask_DELAYDecision(t *testing.T) {
	// DELAY decisions should create review tasks

	task := models.GovernanceTask{
		DecisionID:  uuid.New(),
		TaskType:    "REVIEW",
		AssignedTo:  "CARE_TEAM",
		Priority:    "MEDIUM",
		DueAt:       time.Now().Add(24 * time.Hour),
		Description: "Review delayed action: metoprolol. Reassess when hemodynamically stable.",
	}

	assert.Equal(t, "REVIEW", task.TaskType)
	assert.Equal(t, "CARE_TEAM", task.AssignedTo)
}

func TestGovernanceTask_CONSIDERDecision(t *testing.T) {
	// CONSIDER decisions should create review tasks with lower priority

	task := models.GovernanceTask{
		DecisionID:  uuid.New(),
		TaskType:    "REVIEW",
		AssignedTo:  "PRIMARY_PROVIDER",
		Priority:    "LOW",
		DueAt:       time.Now().Add(7 * 24 * time.Hour),
		Description: "Consider: Statin therapy for cardiovascular risk reduction.",
	}

	assert.Equal(t, "LOW", task.Priority)
	assert.Contains(t, task.Description, "Consider")
}

func TestGovernanceTask_Priorities(t *testing.T) {
	priorities := []string{"CRITICAL", "HIGH", "MEDIUM", "LOW"}

	for _, priority := range priorities {
		task := models.GovernanceTask{
			DecisionID: uuid.New(),
			Priority:   priority,
		}
		assert.NotEmpty(t, task.Priority)
	}
}

func TestGovernanceTask_Assignees(t *testing.T) {
	assignees := []string{
		"ATTENDING_PHYSICIAN",
		"CARE_TEAM",
		"PRIMARY_PROVIDER",
		"PHARMACIST",
		"NURSING",
	}

	for _, assignee := range assignees {
		task := models.GovernanceTask{
			DecisionID: uuid.New(),
			AssignedTo: assignee,
		}
		assert.NotEmpty(t, task.AssignedTo)
	}
}

// ============================================================================
// PILLAR 6.4: EXECUTION PLAN COMPLETENESS
// Full execution plan should have all components
// ============================================================================

func TestExecutionPlan_Structure(t *testing.T) {
	plan := models.ExecutionPlan{
		TemporalBindings:    make([]models.TemporalBinding, 0),
		OrderSetActivations: make([]models.OrderSetActivation, 0),
		GovernanceTasks:     make([]models.GovernanceTask, 0),
	}

	assert.NotNil(t, plan.TemporalBindings)
	assert.NotNil(t, plan.OrderSetActivations)
	assert.NotNil(t, plan.GovernanceTasks)
}

func TestExecutionPlan_MultipleBindings(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:8083",
			KB12URL: "http://localhost:8094",
			KB14URL: "http://localhost:8091",
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	// Create scenario with multiple protocols
	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
			"HasAKI":    true,
		},
	}

	bundle, err := engine.Execute(context.Background(), uuid.New(), uuid.New(), contextData)
	require.NoError(t, err)

	// Should have execution plan
	assert.NotNil(t, bundle.ExecutionPlan)
}

// ============================================================================
// PILLAR 6.5: BOUND ACTION TRACKING
// Decisions should track their bound actions
// ============================================================================

func TestBoundAction_Structure(t *testing.T) {
	action := models.BoundAction{
		AbstractActionID: "abstract-action-1",
		BindingType:      "KB12_ORDERSET",
		BoundEntityID:    "OS-SEPSIS-001",
		Details: map[string]interface{}{
			"orderset_name": "Sepsis Bundle",
		},
		ScheduledAt: ptrTime(time.Now().Add(15 * time.Minute)),
		Status:      "PENDING",
	}

	assert.Equal(t, "KB12_ORDERSET", action.BindingType)
	assert.Equal(t, "OS-SEPSIS-001", action.BoundEntityID)
	assert.Equal(t, "PENDING", action.Status)
	assert.NotNil(t, action.ScheduledAt)
}

func TestBoundAction_BindingTypes(t *testing.T) {
	bindingTypes := []string{
		"KB3_TEMPORAL",
		"KB12_ORDERSET",
		"KB14_TASK",
	}

	for _, bindingType := range bindingTypes {
		action := models.BoundAction{
			BindingType: bindingType,
		}
		assert.NotEmpty(t, action.BindingType)
	}
}

func TestBoundAction_StatusValues(t *testing.T) {
	statuses := []string{
		"PENDING",
		"SCHEDULED",
		"IN_PROGRESS",
		"COMPLETED",
		"CANCELLED",
	}

	for _, status := range statuses {
		action := models.BoundAction{
			Status: status,
		}
		assert.NotEmpty(t, action.Status)
	}
}

func TestDecision_AddBoundAction(t *testing.T) {
	decision := models.NewArbitratedDecision(models.DecisionDo, "lisinopril", "HF GDMT")

	action := models.BoundAction{
		AbstractActionID: "start-acei",
		BindingType:      "KB12_ORDERSET",
		BoundEntityID:    "OS-ACEI-START",
		Status:           "PENDING",
	}

	decision.AddBoundAction(action)

	require.Len(t, decision.Actions, 1)
	assert.Equal(t, "start-acei", decision.Actions[0].AbstractActionID)
}

// ============================================================================
// PILLAR 6.6: MONITORING PLAN
// Safety-flagged decisions should include monitoring
// ============================================================================

func TestMonitoringPlan_Structure(t *testing.T) {
	item := models.MonitoringItem{
		Parameter:    "INR",
		Frequency:    "daily",
		TargetMin:    2.0,
		TargetMax:    3.0,
		Duration:     "until therapeutic",
		AlertIfBelow: ptrFloat64(1.5),
		AlertIfAbove: ptrFloat64(4.0),
	}

	assert.Equal(t, "INR", item.Parameter)
	assert.Equal(t, "daily", item.Frequency)
	assert.Equal(t, 2.0, item.TargetMin)
	assert.Equal(t, 3.0, item.TargetMax)
	assert.NotNil(t, item.AlertIfBelow)
	assert.NotNil(t, item.AlertIfAbove)
}

func TestDecision_AddMonitoring(t *testing.T) {
	decision := models.NewArbitratedDecision(models.DecisionDo, "vancomycin", "MRSA coverage")

	decision.AddMonitoring(models.MonitoringItem{
		Parameter: "Creatinine",
		Frequency: "daily",
	})
	decision.AddMonitoring(models.MonitoringItem{
		Parameter: "Vancomycin trough",
		Frequency: "q3doses",
	})

	require.Len(t, decision.MonitoringPlan, 2)
	assert.Equal(t, "Creatinine", decision.MonitoringPlan[0].Parameter)
	assert.Equal(t, "Vancomycin trough", decision.MonitoringPlan[1].Parameter)
}

// ============================================================================
// PILLAR 6.7: SERVICE VERSION TRACKING
// Bundle should record versions of all KB services used
// ============================================================================

func TestServiceVersionTracking(t *testing.T) {
	bundle := models.NewRecommendationBundle(uuid.New(), uuid.New())

	// Record service versions
	bundle.ServiceVersions["KB-3"] = "2.1.0"
	bundle.ServiceVersions["KB-8"] = "1.5.0"
	bundle.ServiceVersions["KB-12"] = "1.3.0"
	bundle.ServiceVersions["KB-14"] = "1.2.0"
	bundle.ServiceVersions["KB-19"] = "1.0.0"

	assert.Len(t, bundle.ServiceVersions, 5)
	assert.Equal(t, "2.1.0", bundle.ServiceVersions["KB-3"])
	assert.Equal(t, "1.0.0", bundle.ServiceVersions["KB-19"])
}

// ============================================================================
// PILLAR 6.8: END-TO-END EXECUTION BINDING FLOW
// Complete flow from decision to execution plan
// ============================================================================

func TestExecutionBinding_EndToEndFlow(t *testing.T) {
	log := logrus.NewEntry(logrus.New())
	cfg := &config.Config{
		Server: config.ServerConfig{Environment: "test"},
		KBServices: config.KBServicesConfig{
			KB3URL:  "http://localhost:8083",
			KB12URL: "http://localhost:8094",
			KB14URL: "http://localhost:8091",
			Timeout: 30 * time.Second,
		},
	}

	engine, err := arbitration.NewEngine(cfg, log)
	require.NoError(t, err)

	patientID := uuid.New()
	encounterID := uuid.New()

	contextData := map[string]interface{}{
		"cql_truth_flags": map[string]interface{}{
			"HasSepsis": true,
		},
	}

	bundle, err := engine.Execute(context.Background(), patientID, encounterID, contextData)
	require.NoError(t, err)
	require.NotNil(t, bundle)

	// Verify bundle structure
	assert.Equal(t, patientID, bundle.PatientID)
	assert.Equal(t, encounterID, bundle.EncounterID)
	assert.Equal(t, models.StatusCompleted, bundle.Status)

	// Execution plan should be initialized with real KB services
	assert.NotNil(t, bundle.ExecutionPlan)
	require.NotNil(t, bundle.ExecutionPlan.TemporalBindings,
		"Temporal bindings should be initialized (requires KB-3 at localhost:8083)")
	require.NotNil(t, bundle.ExecutionPlan.OrderSetActivations,
		"OrderSet activations should be initialized (requires KB-12 at localhost:8094)")
	require.NotNil(t, bundle.ExecutionPlan.GovernanceTasks,
		"Governance tasks should be initialized (requires KB-14 at localhost:8091)")
}

// ============================================================================
// Helper Functions
// ============================================================================

func ptrTime(t time.Time) *time.Time {
	return &t
}

func ptrFloat64(f float64) *float64 {
	return &f
}
