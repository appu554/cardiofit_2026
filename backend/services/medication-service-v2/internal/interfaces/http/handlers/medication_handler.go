package handlers

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"medication-service-v2/internal/application/services"
	"medication-service-v2/internal/domain/entities"
	"medication-service-v2/internal/interfaces/http/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MedicationProposalHandler handles HTTP requests for medication proposals
type MedicationProposalHandler struct {
	service *services.MedicationService
	logger  *zap.Logger
}

// NewMedicationProposalHandler creates a new medication proposal handler
func NewMedicationProposalHandler(service *services.MedicationService, logger *zap.Logger) *MedicationProposalHandler {
	return &MedicationProposalHandler{
		service: service,
		logger:  logger,
	}
}

// CreateProposalRequest represents the request body for creating a medication proposal
type CreateProposalRequest struct {
	PatientID         string                         `json:"patient_id" binding:"required"`
	ProtocolID        string                         `json:"protocol_id" binding:"required"`
	Indication        string                         `json:"indication" binding:"required"`
	ClinicalContext   *entities.ClinicalContext      `json:"clinical_context" binding:"required"`
	MedicationDetails *entities.MedicationDetails    `json:"medication_details" binding:"required"`
}

// UpdateProposalRequest represents the request body for updating a medication proposal
type UpdateProposalRequest struct {
	Indication        *string                        `json:"indication,omitempty"`
	ClinicalContext   *entities.ClinicalContext      `json:"clinical_context,omitempty"`
	MedicationDetails *entities.MedicationDetails    `json:"medication_details,omitempty"`
	Status            *entities.ProposalStatus       `json:"status,omitempty"`
}

// ProposalResponse represents the response structure for medication proposals
type ProposalResponse struct {
	ID                    string                           `json:"id"`
	PatientID             string                           `json:"patient_id"`
	ProtocolID            string                           `json:"protocol_id"`
	Indication            string                           `json:"indication"`
	Status                entities.ProposalStatus          `json:"status"`
	ClinicalContext       *entities.ClinicalContext        `json:"clinical_context"`
	MedicationDetails     *entities.MedicationDetails      `json:"medication_details"`
	DosageRecommendations []entities.DosageRecommendation  `json:"dosage_recommendations"`
	SafetyConstraints     []entities.SafetyConstraint      `json:"safety_constraints"`
	SnapshotID            string                           `json:"snapshot_id"`
	CreatedAt             time.Time                        `json:"created_at"`
	UpdatedAt             time.Time                        `json:"updated_at"`
	CreatedBy             string                           `json:"created_by"`
	ValidatedBy           *string                          `json:"validated_by,omitempty"`
	ValidationTimestamp   *time.Time                       `json:"validation_timestamp,omitempty"`
}

// ListProposalsResponse represents the response for listing proposals
type ListProposalsResponse struct {
	Proposals    []ProposalResponse `json:"proposals"`
	TotalCount   int                `json:"total_count"`
	Page         int                `json:"page"`
	PageSize     int                `json:"page_size"`
	TotalPages   int                `json:"total_pages"`
	HasNext      bool               `json:"has_next"`
	HasPrevious  bool               `json:"has_previous"`
}

// ValidateProposalRequest represents the request for validating a proposal
type ValidateProposalRequest struct {
	ValidatedBy string `json:"validated_by" binding:"required"`
}

// ValidationResult represents validation results
type ValidationResult struct {
	Field    string `json:"field"`
	Message  string `json:"message"`
	Severity string `json:"severity"`
	Code     string `json:"code"`
}

// ValidateProposalResponse represents the validation response
type ValidateProposalResponse struct {
	Proposal          ProposalResponse    `json:"proposal"`
	ValidationResults []ValidationResult `json:"validation_results"`
	IsValid           bool                `json:"is_valid"`
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string      `json:"error"`
	Message string      `json:"message"`
	Code    string      `json:"code"`
	Details interface{} `json:"details,omitempty"`
}

// CreateProposal creates a new medication proposal
// @Summary Create a new medication proposal
// @Description Creates a new medication proposal with clinical context and medication details
// @Tags medication-proposals
// @Accept json
// @Produce json
// @Param proposal body CreateProposalRequest true "Medication proposal data"
// @Success 201 {object} ProposalResponse
// @Failure 400 {object} ErrorResponse
// @Failure 401 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/medication-proposals [post]
func (h *MedicationProposalHandler) CreateProposal(c *gin.Context) {
	var req CreateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		h.logger.Error("Invalid request body", zap.Error(err))
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Code:    "BAD_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Get authenticated user
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "authentication_required",
			Message: "Authentication required",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	// Parse patient ID
	patientID, err := uuid.Parse(req.PatientID)
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_patient_id",
			Message: "Invalid patient ID format",
			Code:    "BAD_REQUEST",
		})
		return
	}

	// Create proposal using service
	createReq := services.CreateProposalRequest{
		PatientID:         patientID,
		ProtocolID:        req.ProtocolID,
		Indication:        req.Indication,
		ClinicalContext:   req.ClinicalContext,
		MedicationDetails: req.MedicationDetails,
		CreatedBy:         authCtx.UserID,
	}

	proposal, err := h.service.CreateProposal(c.Request.Context(), createReq)
	if err != nil {
		h.logger.Error("Failed to create proposal", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "creation_failed",
			Message: "Failed to create medication proposal",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	response := h.convertToProposalResponse(proposal)
	c.JSON(http.StatusCreated, response)
}

// GetProposal retrieves a medication proposal by ID
// @Summary Get a medication proposal
// @Description Retrieves a medication proposal by its ID
// @Tags medication-proposals
// @Produce json
// @Param id path string true "Proposal ID"
// @Success 200 {object} ProposalResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/medication-proposals/{id} [get]
func (h *MedicationProposalHandler) GetProposal(c *gin.Context) {
	proposalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid proposal ID format",
			Code:    "BAD_REQUEST",
		})
		return
	}

	proposal, err := h.service.GetProposal(c.Request.Context(), proposalID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "proposal_not_found",
				Message: "Medication proposal not found",
				Code:    "NOT_FOUND",
			})
			return
		}

		h.logger.Error("Failed to get proposal", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "retrieval_failed",
			Message: "Failed to retrieve medication proposal",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	response := h.convertToProposalResponse(proposal)
	c.JSON(http.StatusOK, response)
}

// UpdateProposal updates an existing medication proposal
// @Summary Update a medication proposal
// @Description Updates an existing medication proposal
// @Tags medication-proposals
// @Accept json
// @Produce json
// @Param id path string true "Proposal ID"
// @Param proposal body UpdateProposalRequest true "Updated proposal data"
// @Success 200 {object} ProposalResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/medication-proposals/{id} [put]
func (h *MedicationProposalHandler) UpdateProposal(c *gin.Context) {
	proposalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid proposal ID format",
			Code:    "BAD_REQUEST",
		})
		return
	}

	var req UpdateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Code:    "BAD_REQUEST",
			Details: err.Error(),
		})
		return
	}

	// Get authenticated user
	authCtx, exists := middleware.GetAuthContext(c)
	if !exists {
		c.JSON(http.StatusUnauthorized, ErrorResponse{
			Error:   "authentication_required",
			Message: "Authentication required",
			Code:    "UNAUTHORIZED",
		})
		return
	}

	// Get existing proposal to build update request
	existingProposal, err := h.service.GetProposal(c.Request.Context(), proposalID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "proposal_not_found",
				Message: "Medication proposal not found",
				Code:    "NOT_FOUND",
			})
			return
		}

		h.logger.Error("Failed to get existing proposal", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "retrieval_failed",
			Message: "Failed to retrieve existing proposal",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Build updated proposal
	updatedProposal := *existingProposal
	if req.Indication != nil {
		updatedProposal.Indication = *req.Indication
	}
	if req.ClinicalContext != nil {
		updatedProposal.ClinicalContext = req.ClinicalContext
	}
	if req.MedicationDetails != nil {
		updatedProposal.MedicationDetails = req.MedicationDetails
	}
	if req.Status != nil {
		updatedProposal.Status = *req.Status
	}

	updateReq := services.UpdateProposalRequest{
		Proposal:  &updatedProposal,
		UpdatedBy: authCtx.UserID,
	}

	proposal, err := h.service.UpdateProposal(c.Request.Context(), proposalID, updateReq)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "proposal_not_found",
				Message: "Medication proposal not found",
				Code:    "NOT_FOUND",
			})
			return
		}

		h.logger.Error("Failed to update proposal", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "update_failed",
			Message: "Failed to update medication proposal",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	response := h.convertToProposalResponse(proposal)
	c.JSON(http.StatusOK, response)
}

// DeleteProposal deletes a medication proposal
// @Summary Delete a medication proposal
// @Description Deletes a medication proposal by ID
// @Tags medication-proposals
// @Param id path string true "Proposal ID"
// @Success 204
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/medication-proposals/{id} [delete]
func (h *MedicationProposalHandler) DeleteProposal(c *gin.Context) {
	proposalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid proposal ID format",
			Code:    "BAD_REQUEST",
		})
		return
	}

	err = h.service.DeleteProposal(c.Request.Context(), proposalID)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "proposal_not_found",
				Message: "Medication proposal not found",
				Code:    "NOT_FOUND",
			})
			return
		}

		h.logger.Error("Failed to delete proposal", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "deletion_failed",
			Message: "Failed to delete medication proposal",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	c.Status(http.StatusNoContent)
}

// ListProposals lists medication proposals with pagination and filtering
// @Summary List medication proposals
// @Description Lists medication proposals with pagination and filtering options
// @Tags medication-proposals
// @Produce json
// @Param patient_id query string false "Filter by patient ID"
// @Param status query string false "Filter by proposal status"
// @Param page query int false "Page number (default: 1)"
// @Param page_size query int false "Page size (default: 20)"
// @Param sort_by query string false "Sort field (default: created_at)"
// @Param sort_order query string false "Sort order: asc or desc (default: desc)"
// @Success 200 {object} ListProposalsResponse
// @Failure 400 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/medication-proposals [get]
func (h *MedicationProposalHandler) ListProposals(c *gin.Context) {
	// Parse query parameters
	var patientID *uuid.UUID
	if patientIDStr := c.Query("patient_id"); patientIDStr != "" {
		id, err := uuid.Parse(patientIDStr)
		if err != nil {
			c.JSON(http.StatusBadRequest, ErrorResponse{
				Error:   "invalid_patient_id",
				Message: "Invalid patient ID format",
				Code:    "BAD_REQUEST",
			})
			return
		}
		patientID = &id
	}

	var status entities.ProposalStatus
	if statusStr := c.Query("status"); statusStr != "" {
		status = entities.ProposalStatus(statusStr)
	}

	page := 1
	if pageStr := c.Query("page"); pageStr != "" {
		if p, err := strconv.Atoi(pageStr); err == nil && p > 0 {
			page = p
		}
	}

	pageSize := 20
	if pageSizeStr := c.Query("page_size"); pageSizeStr != "" {
		if ps, err := strconv.Atoi(pageSizeStr); err == nil && ps > 0 && ps <= 100 {
			pageSize = ps
		}
	}

	sortBy := c.DefaultQuery("sort_by", "created_at")
	sortOrder := c.DefaultQuery("sort_order", "desc")

	// Build list request
	listReq := services.ListProposalsRequest{
		PatientID: patientID,
		Status:    status,
		Page:      page,
		PageSize:  pageSize,
		SortBy:    sortBy,
		SortOrder: sortOrder,
	}

	result, err := h.service.ListProposals(c.Request.Context(), listReq)
	if err != nil {
		h.logger.Error("Failed to list proposals", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "list_failed",
			Message: "Failed to list medication proposals",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Convert to response format
	proposals := make([]ProposalResponse, len(result.Proposals))
	for i, proposal := range result.Proposals {
		proposals[i] = h.convertToProposalResponse(proposal)
	}

	totalPages := (result.TotalCount + pageSize - 1) / pageSize

	response := ListProposalsResponse{
		Proposals:   proposals,
		TotalCount:  result.TotalCount,
		Page:        page,
		PageSize:    pageSize,
		TotalPages:  totalPages,
		HasNext:     page < totalPages,
		HasPrevious: page > 1,
	}

	// Set pagination headers
	c.Header("X-Total-Count", strconv.Itoa(result.TotalCount))
	c.Header("X-Page", strconv.Itoa(page))
	c.Header("X-Page-Size", strconv.Itoa(pageSize))
	c.Header("X-Total-Pages", strconv.Itoa(totalPages))

	c.JSON(http.StatusOK, response)
}

// ValidateProposal validates a medication proposal
// @Summary Validate a medication proposal
// @Description Validates a medication proposal and marks it as validated
// @Tags medication-proposals
// @Accept json
// @Produce json
// @Param id path string true "Proposal ID"
// @Param validation body ValidateProposalRequest true "Validation data"
// @Success 200 {object} ValidateProposalResponse
// @Failure 400 {object} ErrorResponse
// @Failure 404 {object} ErrorResponse
// @Failure 500 {object} ErrorResponse
// @Security BearerAuth
// @Router /api/v1/medication-proposals/{id}/validate [post]
func (h *MedicationProposalHandler) ValidateProposal(c *gin.Context) {
	proposalID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_id",
			Message: "Invalid proposal ID format",
			Code:    "BAD_REQUEST",
		})
		return
	}

	var req ValidateProposalRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, ErrorResponse{
			Error:   "invalid_request",
			Message: "Invalid request body",
			Code:    "BAD_REQUEST",
			Details: err.Error(),
		})
		return
	}

	validateReq := services.ValidateProposalRequest{
		ProposalID:  proposalID,
		ValidatedBy: req.ValidatedBy,
	}

	validation, err := h.service.ValidateProposal(c.Request.Context(), validateReq)
	if err != nil {
		if isNotFoundError(err) {
			c.JSON(http.StatusNotFound, ErrorResponse{
				Error:   "proposal_not_found",
				Message: "Medication proposal not found",
				Code:    "NOT_FOUND",
			})
			return
		}

		h.logger.Error("Failed to validate proposal", zap.Error(err))
		c.JSON(http.StatusInternalServerError, ErrorResponse{
			Error:   "validation_failed",
			Message: "Failed to validate medication proposal",
			Code:    "INTERNAL_ERROR",
		})
		return
	}

	// Convert validation results
	validationResults := make([]ValidationResult, len(validation.ValidationResults))
	for i, result := range validation.ValidationResults {
		validationResults[i] = ValidationResult{
			Field:    result.Field,
			Message:  result.Message,
			Severity: result.Severity,
			Code:     result.Code,
		}
	}

	isValid := len(validationResults) == 0
	for _, result := range validationResults {
		if result.Severity == "error" || result.Severity == "critical" {
			isValid = false
			break
		}
	}

	response := ValidateProposalResponse{
		Proposal:          h.convertToProposalResponse(validation.Proposal),
		ValidationResults: validationResults,
		IsValid:           isValid,
	}

	c.JSON(http.StatusOK, response)
}

// CommitProposal commits a validated medication proposal
func (h *MedicationProposalHandler) CommitProposal(c *gin.Context) {
	// Implementation for committing a proposal
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Error:   "not_implemented",
		Message: "Commit proposal functionality not yet implemented",
		Code:    "NOT_IMPLEMENTED",
	})
}

// GetProposalHistory retrieves the history of changes for a proposal
func (h *MedicationProposalHandler) GetProposalHistory(c *gin.Context) {
	// Implementation for getting proposal history
	c.JSON(http.StatusNotImplemented, ErrorResponse{
		Error:   "not_implemented",
		Message: "Proposal history functionality not yet implemented",
		Code:    "NOT_IMPLEMENTED",
	})
}

// Helper methods

// convertToProposalResponse converts domain entity to response format
func (h *MedicationProposalHandler) convertToProposalResponse(proposal *entities.MedicationProposal) ProposalResponse {
	return ProposalResponse{
		ID:                    proposal.ID.String(),
		PatientID:             proposal.PatientID.String(),
		ProtocolID:            proposal.ProtocolID,
		Indication:            proposal.Indication,
		Status:                proposal.Status,
		ClinicalContext:       proposal.ClinicalContext,
		MedicationDetails:     proposal.MedicationDetails,
		DosageRecommendations: proposal.DosageRecommendations,
		SafetyConstraints:     proposal.SafetyConstraints,
		SnapshotID:            proposal.SnapshotID.String(),
		CreatedAt:             proposal.CreatedAt,
		UpdatedAt:             proposal.UpdatedAt,
		CreatedBy:             proposal.CreatedBy,
		ValidatedBy:           proposal.ValidatedBy,
		ValidationTimestamp:   proposal.ValidationTimestamp,
	}
}

// isNotFoundError checks if the error represents a "not found" condition
func isNotFoundError(err error) bool {
	// Implementation would depend on your error handling strategy
	return err.Error() == "not found" // Placeholder
}