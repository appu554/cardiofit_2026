// Package api provides HTTP handlers for KB-14 Care Navigator
package api

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"

	"kb-14-care-navigator/internal/models"
)

// SearchFHIRTasks searches for FHIR Task resources
func (s *Server) SearchFHIRTasks(c *gin.Context) {
	// Parse FHIR search parameters
	var filters models.WorklistFilters

	// Patient reference (_for parameter in FHIR)
	if patient := c.Query("patient"); patient != "" {
		filters.PatientID = patient
	}

	// Status parameter
	if status := c.Query("status"); status != "" {
		// Map FHIR status to internal status
		fhirStatus := mapFHIRStatusToInternal(status)
		if fhirStatus != "" {
			filters.Statuses = []models.TaskStatus{models.TaskStatus(fhirStatus)}
		}
	}

	// Priority parameter
	if priority := c.Query("priority"); priority != "" {
		fhirPriority := mapFHIRPriorityToInternal(priority)
		if fhirPriority != "" {
			filters.Priorities = []models.TaskPriority{models.TaskPriority(fhirPriority)}
		}
	}

	// Owner (assignee)
	if owner := c.Query("owner"); owner != "" {
		uid, err := uuid.Parse(owner)
		if err == nil {
			filters.UserID = &uid
		}
	}

	// Pagination (_count and _offset)
	count, _ := strconv.Atoi(c.DefaultQuery("_count", "20"))
	offset, _ := strconv.Atoi(c.DefaultQuery("_offset", "0"))

	if count <= 0 || count > 100 {
		count = 20
	}

	page := (offset / count) + 1
	filters.Page = page
	filters.PageSize = count

	// Get tasks
	response, err := s.worklistService.GetWorklist(c.Request.Context(), filters)
	if err != nil {
		s.log.WithError(err).Error("Failed to search FHIR tasks")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "exception",
				"details":  gin.H{"text": err.Error()},
			}},
		})
		return
	}

	// Convert to FHIR Bundle
	bundle := s.fhirMapper.CreateBundle(response.Data, int(response.Total))
	c.JSON(http.StatusOK, bundle)
}

// GetFHIRTask retrieves a single FHIR Task resource
func (s *Server) GetFHIRTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "invalid",
				"details":  gin.H{"text": "invalid task ID format"},
			}},
		})
		return
	}

	task, err := s.taskService.GetByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "not-found",
				"details":  gin.H{"text": "task not found"},
			}},
		})
		return
	}

	fhirTask := s.fhirMapper.ToFHIR(task)
	c.JSON(http.StatusOK, fhirTask)
}

// CreateFHIRTask creates a new task from a FHIR Task resource
func (s *Server) CreateFHIRTask(c *gin.Context) {
	var fhirTask models.FHIRTask
	if err := c.ShouldBindJSON(&fhirTask); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "invalid",
				"details":  gin.H{"text": err.Error()},
			}},
		})
		return
	}

	// Validate resource type
	if fhirTask.ResourceType != "Task" {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "invalid",
				"details":  gin.H{"text": "resourceType must be 'Task'"},
			}},
		})
		return
	}

	// Convert to internal task
	task := s.fhirMapper.FromFHIR(&fhirTask)

	// Create the task
	createdTask, err := s.taskService.Create(c.Request.Context(), &models.CreateTaskRequest{
		Type:        task.Type,
		Priority:    task.Priority,
		Source:      models.TaskSourceManual,
		PatientID:   task.PatientID,
		Title:       task.Title,
		Description: task.Description,
	})
	if err != nil {
		s.log.WithError(err).Error("Failed to create FHIR task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "exception",
				"details":  gin.H{"text": err.Error()},
			}},
		})
		return
	}

	resultFHIR := s.fhirMapper.ToFHIR(createdTask)
	c.JSON(http.StatusCreated, resultFHIR)
}

// UpdateFHIRTask updates a task from a FHIR Task resource
func (s *Server) UpdateFHIRTask(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "invalid",
				"details":  gin.H{"text": "invalid task ID format"},
			}},
		})
		return
	}

	var fhirTask models.FHIRTask
	if err := c.ShouldBindJSON(&fhirTask); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "invalid",
				"details":  gin.H{"text": err.Error()},
			}},
		})
		return
	}

	// Convert to internal task
	task := s.fhirMapper.FromFHIR(&fhirTask)

	// Build update request
	updateReq := &models.UpdateTaskRequest{
		Status:      &task.Status,
		Priority:    &task.Priority,
		Title:       &task.Title,
		Description: &task.Description,
	}

	updatedTask, err := s.taskService.Update(c.Request.Context(), id, updateReq)
	if err != nil {
		s.log.WithError(err).Error("Failed to update FHIR task")
		c.JSON(http.StatusInternalServerError, gin.H{
			"resourceType": "OperationOutcome",
			"issue": []gin.H{{
				"severity": "error",
				"code":     "exception",
				"details":  gin.H{"text": err.Error()},
			}},
		})
		return
	}

	resultFHIR := s.fhirMapper.ToFHIR(updatedTask)
	c.JSON(http.StatusOK, resultFHIR)
}

// mapFHIRStatusToInternal maps FHIR Task status to internal status
func mapFHIRStatusToInternal(fhirStatus string) string {
	switch fhirStatus {
	case "draft":
		return "CREATED"
	case "requested":
		return "CREATED"
	case "accepted":
		return "ASSIGNED"
	case "in-progress":
		return "IN_PROGRESS"
	case "completed":
		return "COMPLETED"
	case "cancelled":
		return "CANCELLED"
	case "on-hold":
		return "BLOCKED"
	case "failed":
		return "DECLINED"
	default:
		return ""
	}
}

// mapFHIRPriorityToInternal maps FHIR Task priority to internal priority
func mapFHIRPriorityToInternal(fhirPriority string) string {
	switch fhirPriority {
	case "stat":
		return "CRITICAL"
	case "asap":
		return "HIGH"
	case "urgent":
		return "HIGH"
	case "routine":
		return "MEDIUM"
	default:
		return ""
	}
}
