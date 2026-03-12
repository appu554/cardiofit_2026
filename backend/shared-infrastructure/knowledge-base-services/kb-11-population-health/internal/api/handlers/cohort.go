// Package handlers provides HTTP request handlers for the KB-11 API.
package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"

	"github.com/cardiofit/kb-11-population-health/internal/cohort"
	"github.com/cardiofit/kb-11-population-health/internal/models"
)

// CohortHandler handles cohort management endpoints.
type CohortHandler struct {
	service *cohort.Service
	logger  *logrus.Entry
}

// NewCohortHandler creates a new cohort handler.
func NewCohortHandler(service *cohort.Service, logger *logrus.Entry) *CohortHandler {
	return &CohortHandler{
		service: service,
		logger:  logger.WithField("handler", "cohort"),
	}
}

// ──────────────────────────────────────────────────────────────────────────────
// Cohort CRUD Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// CreateStaticCohortRequest represents a request to create a static cohort.
type CreateStaticCohortRequest struct {
	Name        string `json:"name" binding:"required"`
	Description string `json:"description"`
	CreatedBy   string `json:"created_by" binding:"required"`
}

// CreateStaticCohort handles POST /v1/cohorts/static - create a static cohort.
func (h *CohortHandler) CreateStaticCohort(c *gin.Context) {
	var req CreateStaticCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	result, err := h.service.CreateStaticCohort(c.Request.Context(), req.Name, req.Description, req.CreatedBy)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create static cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create cohort",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}

// CreateDynamicCohortRequest represents a request to create a dynamic cohort.
type CreateDynamicCohortRequest struct {
	Name        string             `json:"name" binding:"required"`
	Description string             `json:"description"`
	CreatedBy   string             `json:"created_by" binding:"required"`
	Criteria    []cohort.Criterion `json:"criteria" binding:"required,min=1"`
}

// CreateDynamicCohort handles POST /v1/cohorts/dynamic - create a dynamic cohort.
func (h *CohortHandler) CreateDynamicCohort(c *gin.Context) {
	var req CreateDynamicCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	result, err := h.service.CreateDynamicCohort(
		c.Request.Context(),
		req.Name,
		req.Description,
		req.CreatedBy,
		req.Criteria,
	)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create dynamic cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create cohort",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}

// CreateSnapshotCohortRequest represents a request to create a snapshot cohort.
type CreateSnapshotCohortRequest struct {
	SourceCohortID string `json:"source_cohort_id" binding:"required,uuid"`
	CreatedBy      string `json:"created_by" binding:"required"`
}

// CreateSnapshotCohort handles POST /v1/cohorts/snapshot - create a snapshot of a cohort.
func (h *CohortHandler) CreateSnapshotCohort(c *gin.Context) {
	var req CreateSnapshotCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	sourceID, err := uuid.Parse(req.SourceCohortID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid source cohort ID",
			"INVALID_UUID",
			err.Error(),
		))
		return
	}

	result, err := h.service.CreateSnapshotCohort(c.Request.Context(), sourceID, req.CreatedBy)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create snapshot cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create snapshot",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}

// GetCohort handles GET /v1/cohorts/:id - get a cohort by ID.
func (h *CohortHandler) GetCohort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	result, err := h.service.GetCohort(c.Request.Context(), id)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get cohort",
			"GET_ERROR",
			err.Error(),
		))
		return
	}

	if result == nil {
		c.JSON(http.StatusNotFound, models.NewErrorResponse(
			"Cohort not found",
			"NOT_FOUND",
			"",
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cohort": result,
	})
}

// ListCohorts handles GET /v1/cohorts - list all cohorts.
func (h *CohortHandler) ListCohorts(c *gin.Context) {
	filter := &cohort.CohortFilter{}

	// Parse optional filters
	if typeStr := c.Query("type"); typeStr != "" {
		filter.Type = models.CohortType(typeStr)
	}
	if createdBy := c.Query("created_by"); createdBy != "" {
		filter.CreatedBy = createdBy
	}
	if limitStr := c.Query("limit"); limitStr != "" {
		if limit, err := strconv.Atoi(limitStr); err == nil {
			filter.Limit = limit
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if offset, err := strconv.Atoi(offsetStr); err == nil {
			filter.Offset = offset
		}
	}

	results, err := h.service.ListCohorts(c.Request.Context(), filter)
	if err != nil {
		h.logger.WithError(err).Error("Failed to list cohorts")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to list cohorts",
			"LIST_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cohorts": results,
		"count":   len(results),
	})
}

// UpdateCohortRequest represents a request to update a cohort.
type UpdateCohortRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

// UpdateCohort handles PATCH /v1/cohorts/:id - update a cohort.
func (h *CohortHandler) UpdateCohort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	var req UpdateCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	result, err := h.service.UpdateCohort(c.Request.Context(), id, req.Name, req.Description)
	if err != nil {
		h.logger.WithError(err).Error("Failed to update cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to update cohort",
			"UPDATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"cohort": result,
	})
}

// DeleteCohort handles DELETE /v1/cohorts/:id - delete a cohort.
func (h *CohortHandler) DeleteCohort(c *gin.Context) {
	idStr := c.Param("id")
	id, err := uuid.Parse(idStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	if err := h.service.DeleteCohort(c.Request.Context(), id); err != nil {
		h.logger.WithError(err).Error("Failed to delete cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to delete cohort",
			"DELETE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Cohort deleted successfully",
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Membership Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// AddMemberRequest represents a request to add a member to a static cohort.
type AddMemberRequest struct {
	PatientID     string `json:"patient_id" binding:"required,uuid"`
	FHIRPatientID string `json:"fhir_patient_id" binding:"required"`
}

// AddMember handles POST /v1/cohorts/:id/members - add a member to a static cohort.
func (h *CohortHandler) AddMember(c *gin.Context) {
	cohortIDStr := c.Param("id")
	cohortID, err := uuid.Parse(cohortIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	var req AddMemberRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid patient ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	if err := h.service.AddMemberToStaticCohort(c.Request.Context(), cohortID, patientID, req.FHIRPatientID); err != nil {
		h.logger.WithError(err).Error("Failed to add member")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to add member",
			"ADD_MEMBER_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Member added successfully",
	})
}

// RemoveMember handles DELETE /v1/cohorts/:id/members/:patientId - remove a member.
func (h *CohortHandler) RemoveMember(c *gin.Context) {
	cohortIDStr := c.Param("id")
	cohortID, err := uuid.Parse(cohortIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	patientIDStr := c.Param("patientId")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid patient ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	if err := h.service.RemoveMemberFromStaticCohort(c.Request.Context(), cohortID, patientID); err != nil {
		h.logger.WithError(err).Error("Failed to remove member")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to remove member",
			"REMOVE_MEMBER_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Member removed successfully",
	})
}

// GetMembers handles GET /v1/cohorts/:id/members - get members of a cohort.
func (h *CohortHandler) GetMembers(c *gin.Context) {
	cohortIDStr := c.Param("id")
	cohortID, err := uuid.Parse(cohortIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	limit := 100
	offset := 0

	if limitStr := c.Query("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
		}
	}
	if offsetStr := c.Query("offset"); offsetStr != "" {
		if o, err := strconv.Atoi(offsetStr); err == nil {
			offset = o
		}
	}

	members, err := h.service.GetCohortMembers(c.Request.Context(), cohortID, limit, offset)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get members")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get members",
			"GET_MEMBERS_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"members": members,
		"count":   len(members),
	})
}

// CheckMembership handles GET /v1/cohorts/:id/members/:patientId - check membership.
func (h *CohortHandler) CheckMembership(c *gin.Context) {
	cohortIDStr := c.Param("id")
	cohortID, err := uuid.Parse(cohortIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	patientIDStr := c.Param("patientId")
	patientID, err := uuid.Parse(patientIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid patient ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	isMember, err := h.service.IsMemberOfCohort(c.Request.Context(), cohortID, patientID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to check membership")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to check membership",
			"MEMBERSHIP_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"is_member": isMember,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Refresh & Analytics Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// RefreshCohort handles POST /v1/cohorts/:id/refresh - refresh a dynamic cohort.
func (h *CohortHandler) RefreshCohort(c *gin.Context) {
	cohortIDStr := c.Param("id")
	cohortID, err := uuid.Parse(cohortIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	result, err := h.service.RefreshDynamicCohort(c.Request.Context(), cohortID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to refresh cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to refresh cohort",
			"REFRESH_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"refresh_result": result,
	})
}

// GetCohortStats handles GET /v1/cohorts/:id/stats - get cohort statistics.
func (h *CohortHandler) GetCohortStats(c *gin.Context) {
	cohortIDStr := c.Param("id")
	cohortID, err := uuid.Parse(cohortIDStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	stats, err := h.service.GetCohortStats(c.Request.Context(), cohortID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to get cohort stats")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to get stats",
			"STATS_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"stats": stats,
	})
}

// CompareCohorts handles GET /v1/cohorts/compare - compare two cohorts.
func (h *CohortHandler) CompareCohorts(c *gin.Context) {
	cohort1Str := c.Query("cohort1")
	cohort2Str := c.Query("cohort2")

	if cohort1Str == "" || cohort2Str == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Both cohort1 and cohort2 query parameters are required",
			"MISSING_PARAMS",
			"",
		))
		return
	}

	cohort1ID, err := uuid.Parse(cohort1Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort1 ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	cohort2ID, err := uuid.Parse(cohort2Str)
	if err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid cohort2 ID",
			"INVALID_UUID",
			"",
		))
		return
	}

	comparison, err := h.service.CompareCohorts(c.Request.Context(), cohort1ID, cohort2ID)
	if err != nil {
		h.logger.WithError(err).Error("Failed to compare cohorts")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to compare cohorts",
			"COMPARE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"comparison": comparison,
	})
}

// ──────────────────────────────────────────────────────────────────────────────
// Predefined Cohort Creation Endpoints
// ──────────────────────────────────────────────────────────────────────────────

// CreatePredefinedCohortRequest represents a request for predefined cohorts.
type CreatePredefinedCohortRequest struct {
	CreatedBy string `json:"created_by" binding:"required"`
	Value     string `json:"value,omitempty"` // Optional: for PCP name, practice name, min gaps count
}

// CreateHighRiskCohort handles POST /v1/cohorts/predefined/high-risk.
func (h *CohortHandler) CreateHighRiskCohort(c *gin.Context) {
	var req CreatePredefinedCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	result, err := h.service.CreateHighRiskCohort(c.Request.Context(), req.CreatedBy)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create high-risk cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create cohort",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}

// CreateRisingRiskCohort handles POST /v1/cohorts/predefined/rising-risk.
func (h *CohortHandler) CreateRisingRiskCohort(c *gin.Context) {
	var req CreatePredefinedCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	result, err := h.service.CreateRisingRiskCohort(c.Request.Context(), req.CreatedBy)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create rising-risk cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create cohort",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}

// CreateCareGapCohort handles POST /v1/cohorts/predefined/care-gap.
func (h *CohortHandler) CreateCareGapCohort(c *gin.Context) {
	var req CreatePredefinedCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	minGaps := 3 // Default
	if req.Value != "" {
		if mg, err := strconv.Atoi(req.Value); err == nil {
			minGaps = mg
		}
	}

	result, err := h.service.CreateCareGapCohort(c.Request.Context(), minGaps, req.CreatedBy)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create care-gap cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create cohort",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}

// CreatePCPCohort handles POST /v1/cohorts/predefined/pcp.
func (h *CohortHandler) CreatePCPCohort(c *gin.Context) {
	var req CreatePredefinedCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	if req.Value == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"PCP name required in 'value' field",
			"MISSING_VALUE",
			"",
		))
		return
	}

	result, err := h.service.CreatePCPCohort(c.Request.Context(), req.Value, req.CreatedBy)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create PCP cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create cohort",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}

// CreatePracticeCohort handles POST /v1/cohorts/predefined/practice.
func (h *CohortHandler) CreatePracticeCohort(c *gin.Context) {
	var req CreatePredefinedCohortRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Invalid request body",
			"INVALID_REQUEST",
			err.Error(),
		))
		return
	}

	if req.Value == "" {
		c.JSON(http.StatusBadRequest, models.NewErrorResponse(
			"Practice name required in 'value' field",
			"MISSING_VALUE",
			"",
		))
		return
	}

	result, err := h.service.CreatePracticeCohort(c.Request.Context(), req.Value, req.CreatedBy)
	if err != nil {
		h.logger.WithError(err).Error("Failed to create practice cohort")
		c.JSON(http.StatusInternalServerError, models.NewErrorResponse(
			"Failed to create cohort",
			"CREATE_ERROR",
			err.Error(),
		))
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"cohort": result,
	})
}
