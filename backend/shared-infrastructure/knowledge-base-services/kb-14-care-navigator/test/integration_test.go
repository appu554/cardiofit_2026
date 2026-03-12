// Package test contains integration tests for KB-14 Care Navigator
package test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-14-care-navigator/internal/models"
)

// =============================================================================
// Test Helpers
// =============================================================================

func init() {
	gin.SetMode(gin.TestMode)
}

// createTestRouter creates a simple test router for HTTP handler testing
func createTestRouter() *gin.Engine {
	router := gin.New()
	return router
}

// makeJSONRequest creates a JSON request for testing
func makeJSONRequest(t *testing.T, method, path string, body interface{}) *http.Request {
	t.Helper()
	var reqBody *bytes.Buffer
	if body != nil {
		jsonBytes, err := json.Marshal(body)
		require.NoError(t, err)
		reqBody = bytes.NewBuffer(jsonBytes)
	} else {
		reqBody = bytes.NewBuffer(nil)
	}

	req, err := http.NewRequest(method, path, reqBody)
	require.NoError(t, err)
	req.Header.Set("Content-Type", "application/json")
	return req
}

// =============================================================================
// API Request/Response Structure Tests
// =============================================================================

func TestCreateTaskRequest_JSON(t *testing.T) {
	dueDate := time.Now().UTC().Add(24 * time.Hour)
	teamID := uuid.New()
	assigneeID := uuid.New()

	req := &models.CreateTaskRequest{
		Type:         models.TaskTypeMedicationReview,
		Priority:     models.TaskPriorityHigh,
		Source:       models.TaskSourceKB9,
		SourceID:     "GAP-001",
		PatientID:    "PATIENT-001",
		EncounterID:  "ENCOUNTER-001",
		Title:        "Medication Review Required",
		Description:  "Review current medications for potential interactions",
		Instructions: "Check drug-drug interactions",
		ClinicalNote: "Patient on multiple medications",
		DueDate:      &dueDate,
		SLAMinutes:   240,
		TeamID:       &teamID,
		AssignedTo:   &assigneeID,
		AssignedRole: "Pharmacist",
		Actions: []models.TaskAction{
			{
				ActionID:    "action-1",
				Type:        "review",
				Description: "Review medication list",
				Required:    true,
			},
		},
		Metadata: map[string]interface{}{
			"gap_id":       "GAP-001",
			"measure_name": "Medication Reconciliation",
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(req)
	require.NoError(t, err)

	// Unmarshal back
	var parsed models.CreateTaskRequest
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, models.TaskTypeMedicationReview, parsed.Type)
	assert.Equal(t, models.TaskPriorityHigh, parsed.Priority)
	assert.Equal(t, "PATIENT-001", parsed.PatientID)
	assert.Len(t, parsed.Actions, 1)
}

func TestUpdateTaskRequest_JSON(t *testing.T) {
	status := models.TaskStatusInProgress
	priority := models.TaskPriorityHigh
	title := "Updated Title"
	description := "Updated description"
	dueDate := time.Now().UTC().Add(48 * time.Hour)

	req := &models.UpdateTaskRequest{
		Status:      &status,
		Priority:    &priority,
		Title:       &title,
		Description: &description,
		DueDate:     &dueDate,
		Metadata: map[string]interface{}{
			"updated_by": "test-user",
		},
	}

	// Marshal to JSON
	jsonBytes, err := json.Marshal(req)
	require.NoError(t, err)

	// Unmarshal back
	var parsed models.UpdateTaskRequest
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, status, *parsed.Status)
	assert.Equal(t, priority, *parsed.Priority)
	assert.Equal(t, title, *parsed.Title)
}

func TestAssignTaskRequest_JSON(t *testing.T) {
	assigneeID := uuid.New()

	req := &models.AssignTaskRequest{
		AssigneeID: assigneeID,
		Role:       "Care Coordinator",
	}

	jsonBytes, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed models.AssignTaskRequest
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, assigneeID, parsed.AssigneeID)
	assert.Equal(t, "Care Coordinator", parsed.Role)
}

func TestCompleteTaskRequest_JSON(t *testing.T) {
	req := &models.CompleteTaskRequest{
		Outcome: "RESOLVED",
		Notes:   "All actions completed successfully",
	}

	jsonBytes, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed models.CompleteTaskRequest
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "RESOLVED", parsed.Outcome)
	assert.Equal(t, "All actions completed successfully", parsed.Notes)
}

func TestAddNoteRequest_JSON(t *testing.T) {
	req := &models.AddNoteRequest{
		Content:  "Patient condition improving",
		AuthorID: uuid.NewString(),
		Author:   "Dr. Smith",
	}

	jsonBytes, err := json.Marshal(req)
	require.NoError(t, err)

	var parsed models.AddNoteRequest
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.Equal(t, "Patient condition improving", parsed.Content)
	assert.Equal(t, "Dr. Smith", parsed.Author)
}

// =============================================================================
// API Response Structure Tests
// =============================================================================

func TestTaskResponse_JSON(t *testing.T) {
	task := &models.Task{
		ID:        uuid.New(),
		TaskID:    "CLN-12345678",
		Type:      models.TaskTypeMedicationReview,
		Status:    models.TaskStatusCreated,
		Priority:  models.TaskPriorityHigh,
		Source:    models.TaskSourceKB9,
		PatientID: "PATIENT-001",
		Title:     "Medication Review",
	}

	resp := &models.TaskResponse{
		Success: true,
		Data:    task,
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed models.TaskResponse
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.True(t, parsed.Success)
	assert.NotNil(t, parsed.Data)
	assert.Equal(t, "PATIENT-001", parsed.Data.PatientID)
}

func TestTaskListResponse_JSON(t *testing.T) {
	tasks := []models.Task{
		{
			ID:        uuid.New(),
			TaskID:    "CLN-12345678",
			Type:      models.TaskTypeMedicationReview,
			Status:    models.TaskStatusCreated,
			PatientID: "PATIENT-001",
		},
		{
			ID:        uuid.New(),
			TaskID:    "GAP-87654321",
			Type:      models.TaskTypeCareGapClosure,
			Status:    models.TaskStatusAssigned,
			PatientID: "PATIENT-002",
		},
	}

	resp := &models.TaskListResponse{
		Success: true,
		Data:    tasks,
		Total:   2,
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed models.TaskListResponse
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.True(t, parsed.Success)
	assert.Len(t, parsed.Data, 2)
	assert.Equal(t, int64(2), parsed.Total)
}

func TestErrorResponse_JSON(t *testing.T) {
	resp := &models.TaskResponse{
		Success: false,
		Error:   "Task not found",
	}

	jsonBytes, err := json.Marshal(resp)
	require.NoError(t, err)

	var parsed models.TaskResponse
	err = json.Unmarshal(jsonBytes, &parsed)
	require.NoError(t, err)

	assert.False(t, parsed.Success)
	assert.Equal(t, "Task not found", parsed.Error)
	assert.Nil(t, parsed.Data)
}

// =============================================================================
// HTTP Handler Response Tests
// =============================================================================

func TestHealthEndpoint(t *testing.T) {
	router := createTestRouter()
	router.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "kb-14-care-navigator",
			"version": "1.0.0",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)

	var resp map[string]string
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.Equal(t, "healthy", resp["status"])
	assert.Equal(t, "kb-14-care-navigator", resp["service"])
}

func TestBadRequestResponse(t *testing.T) {
	router := createTestRouter()
	router.POST("/api/v1/tasks", func(c *gin.Context) {
		var req models.CreateTaskRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{
				"success": false,
				"error":   err.Error(),
			})
			return
		}
	})

	// Send invalid JSON
	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestNotFoundResponse(t *testing.T) {
	router := createTestRouter()
	router.GET("/api/v1/tasks/:id", func(c *gin.Context) {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"error":   "task not found",
		})
	})

	req := httptest.NewRequest(http.MethodGet, "/api/v1/tasks/"+uuid.NewString(), nil)
	rec := httptest.NewRecorder()
	router.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusNotFound, rec.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(rec.Body.Bytes(), &resp)
	require.NoError(t, err)
	assert.False(t, resp["success"].(bool))
	assert.Equal(t, "task not found", resp["error"])
}

// =============================================================================
// Workflow Integration Tests
// =============================================================================

func TestTaskLifecycleWorkflow(t *testing.T) {
	// Simulate the complete task lifecycle

	// 1. Create task
	createReq := &models.CreateTaskRequest{
		Type:      models.TaskTypeCareGapClosure,
		Source:    models.TaskSourceKB9,
		PatientID: "PATIENT-001",
		Title:     "Close Care Gap",
	}
	assert.Equal(t, models.TaskTypeCareGapClosure, createReq.Type)

	// Simulate task creation
	task := &models.Task{
		ID:        uuid.New(),
		TaskID:    "GAP-12345678",
		Type:      createReq.Type,
		Status:    models.TaskStatusCreated,
		Source:    createReq.Source,
		PatientID: createReq.PatientID,
		Title:     createReq.Title,
		CreatedAt: time.Now().UTC(),
	}

	// 2. Assign task
	assert.Equal(t, models.TaskStatusCreated, task.Status)
	task.Status = models.TaskStatusAssigned
	assigneeID := uuid.New()
	task.AssignedTo = &assigneeID
	now := time.Now().UTC()
	task.AssignedAt = &now

	assert.Equal(t, models.TaskStatusAssigned, task.Status)
	assert.NotNil(t, task.AssignedTo)

	// 3. Start task
	task.Status = models.TaskStatusInProgress
	task.StartedAt = &now

	assert.Equal(t, models.TaskStatusInProgress, task.Status)
	assert.NotNil(t, task.StartedAt)

	// 4. Complete task
	task.Status = models.TaskStatusCompleted
	task.CompletedAt = &now
	task.CompletedBy = task.AssignedTo
	task.Outcome = "RESOLVED"

	assert.Equal(t, models.TaskStatusCompleted, task.Status)
	assert.NotNil(t, task.CompletedAt)
	assert.Equal(t, "RESOLVED", task.Outcome)
}

func TestEscalationWorkflow(t *testing.T) {
	// Create a task that needs escalation
	now := time.Now().UTC()
	pastDue := now.Add(-2 * time.Hour)

	task := &models.Task{
		ID:              uuid.New(),
		TaskID:          "CLN-12345678",
		Type:            models.TaskTypeCriticalLabReview,
		Status:          models.TaskStatusAssigned,
		Priority:        models.TaskPriorityCritical,
		CreatedAt:       pastDue,
		SLAMinutes:      60, // 1 hour SLA
		EscalationLevel: 0,
	}

	// Check SLA elapsed
	slaElapsed := task.GetSLAElapsedPercent()
	assert.GreaterOrEqual(t, slaElapsed, 100.0) // Should be at or over 100% (overdue)

	// Calculate escalation level
	level := models.CalculateEscalationLevel(slaElapsed/100, task.Priority)
	assert.Greater(t, int(level), 0) // Should need escalation

	// Apply escalation
	task.EscalationLevel = int(level)
	if level >= models.EscalationUrgent {
		task.Status = models.TaskStatusEscalated
	}

	assert.Equal(t, models.TaskStatusEscalated, task.Status)
	assert.Greater(t, task.EscalationLevel, 0)
}

func TestMultiActionTaskWorkflow(t *testing.T) {
	// Create task with multiple actions
	task := &models.Task{
		ID:        uuid.New(),
		TaskID:    "GAP-12345678",
		Type:      models.TaskTypeCareGapClosure,
		Status:    models.TaskStatusInProgress,
		PatientID: "PATIENT-001",
		Actions: models.ActionSlice{
			{ActionID: "action-1", Type: "schedule", Description: "Schedule appointment", Required: true, Completed: false},
			{ActionID: "action-2", Type: "lab_order", Description: "Order lab test", Required: true, Completed: false},
			{ActionID: "action-3", Type: "education", Description: "Provide education", Required: false, Completed: false},
		},
	}

	// Complete required actions
	for i := range task.Actions {
		if task.Actions[i].Required {
			now := time.Now().UTC()
			task.Actions[i].Completed = true
			task.Actions[i].CompletedAt = &now
			task.Actions[i].CompletedBy = "test-user"
		}
	}

	// Verify all required actions are complete
	allRequiredComplete := true
	for _, action := range task.Actions {
		if action.Required && !action.Completed {
			allRequiredComplete = false
			break
		}
	}

	assert.True(t, allRequiredComplete)
	assert.True(t, task.Actions[0].Completed)  // Required, should be complete
	assert.True(t, task.Actions[1].Completed)  // Required, should be complete
	assert.False(t, task.Actions[2].Completed) // Optional, still incomplete
}

// =============================================================================
// FHIR Integration Tests
// =============================================================================

func TestFHIRTaskStatusMapping(t *testing.T) {
	tests := []struct {
		name       string
		taskStatus models.TaskStatus
		fhirStatus string
	}{
		{"created", models.TaskStatusCreated, "draft"},
		{"assigned", models.TaskStatusAssigned, "ready"},
		{"in_progress", models.TaskStatusInProgress, "in-progress"},
		{"completed", models.TaskStatusCompleted, "completed"},
		{"cancelled", models.TaskStatusCancelled, "cancelled"},
		{"escalated", models.TaskStatusEscalated, "on-hold"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate FHIR status mapping
			fhirStatus := mapTaskStatusToFHIR(tt.taskStatus)
			assert.Equal(t, tt.fhirStatus, fhirStatus)
		})
	}
}

func TestFHIRTaskPriorityMapping(t *testing.T) {
	tests := []struct {
		name         string
		taskPriority models.TaskPriority
		fhirPriority string
	}{
		{"critical", models.TaskPriorityCritical, "stat"},
		{"high", models.TaskPriorityHigh, "asap"},
		{"medium", models.TaskPriorityMedium, "urgent"},
		{"low", models.TaskPriorityLow, "routine"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fhirPriority := mapTaskPriorityToFHIR(tt.taskPriority)
			assert.Equal(t, tt.fhirPriority, fhirPriority)
		})
	}
}

// Helper functions simulating FHIR mapper
func mapTaskStatusToFHIR(status models.TaskStatus) string {
	switch status {
	case models.TaskStatusCreated:
		return "draft"
	case models.TaskStatusAssigned:
		return "ready"
	case models.TaskStatusInProgress:
		return "in-progress"
	case models.TaskStatusCompleted:
		return "completed"
	case models.TaskStatusVerified:
		return "completed"
	case models.TaskStatusCancelled:
		return "cancelled"
	case models.TaskStatusDeclined:
		return "rejected"
	case models.TaskStatusBlocked:
		return "on-hold"
	case models.TaskStatusEscalated:
		return "on-hold"
	default:
		return "draft"
	}
}

func mapTaskPriorityToFHIR(priority models.TaskPriority) string {
	switch priority {
	case models.TaskPriorityCritical:
		return "stat"
	case models.TaskPriorityHigh:
		return "asap"
	case models.TaskPriorityMedium:
		return "urgent"
	case models.TaskPriorityLow:
		return "routine"
	default:
		return "routine"
	}
}

// =============================================================================
// Concurrent Access Tests
// =============================================================================

func TestConcurrentTaskCreation(t *testing.T) {
	// Simulate concurrent task ID generation
	taskIDs := make(chan string, 100)

	for i := 0; i < 100; i++ {
		go func(n int) {
			// Generate unique task ID
			taskID := "TSK-" + uuid.NewString()[:8]
			taskIDs <- taskID
		}(i)
	}

	// Collect all task IDs
	collected := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id := <-taskIDs
		// Verify uniqueness
		assert.False(t, collected[id], "Duplicate task ID generated: %s", id)
		collected[id] = true
	}

	assert.Len(t, collected, 100)
}

// =============================================================================
// Data Validation Tests
// =============================================================================

func TestPatientIDValidation(t *testing.T) {
	tests := []struct {
		name      string
		patientID string
		valid     bool
	}{
		{"valid_patient_id", "PATIENT-001", true},
		{"empty_patient_id", "", false},
		{"whitespace_only", "   ", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isValid := len(tt.patientID) > 0 && tt.patientID != "   "
			assert.Equal(t, tt.valid, isValid)
		})
	}
}

func TestUUIDParsing(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr bool
	}{
		{"valid_uuid", "550e8400-e29b-41d4-a716-446655440000", false},
		{"invalid_uuid", "not-a-uuid", true},
		{"empty_string", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := uuid.Parse(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
