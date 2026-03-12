// Package handlers provides HTTP request handlers for KB-6 Formulary Service.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"kb-formulary/internal/models"
	"kb-formulary/internal/services"
)

// PAHandler handles HTTP requests for Prior Authorization operations
type PAHandler struct {
	paService *services.PAService
}

// NewPAHandler creates a new PAHandler
func NewPAHandler(paService *services.PAService) *PAHandler {
	return &PAHandler{
		paService: paService,
	}
}

// GetRequirements handles GET /api/v1/pa/requirements requests
// Returns PA requirements for a specific drug
func (h *PAHandler) GetRequirements(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	rxnormCode := r.URL.Query().Get("rxnorm")
	if rxnormCode == "" {
		rxnormCode = r.URL.Query().Get("drug_rxnorm")
	}
	payerID := r.URL.Query().Get("payer_id")
	planID := r.URL.Query().Get("plan_id")

	// Validate required parameters
	if rxnormCode == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: rxnorm")
		return
	}

	// Create request
	req := &models.PARequirementsRequest{
		DrugRxNorm: rxnormCode,
	}
	if payerID != "" {
		req.PayerID = &payerID
	}
	if planID != "" {
		req.PlanID = &planID
	}

	// Get requirements
	response, err := h.paService.GetRequirements(ctx, req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get PA requirements: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// CheckPA handles GET /api/v1/pa/check requests
// Checks if PA is required and evaluates criteria
func (h *PAHandler) CheckPA(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Parse query parameters for simple check
	if r.Method == http.MethodGet {
		rxnormCode := r.URL.Query().Get("rxnorm")
		if rxnormCode == "" {
			rxnormCode = r.URL.Query().Get("drug_rxnorm")
		}
		patientID := r.URL.Query().Get("patient_id")
		payerID := r.URL.Query().Get("payer_id")
		planID := r.URL.Query().Get("plan_id")

		if rxnormCode == "" || patientID == "" {
			writeErrorResponse(w, http.StatusBadRequest, "Missing required parameters: rxnorm and patient_id")
			return
		}

		req := &models.PACheckRequest{
			DrugRxNorm: rxnormCode,
			PatientID:  patientID,
		}
		if payerID != "" {
			req.PayerID = &payerID
		}
		if planID != "" {
			req.PlanID = &planID
		}

		response, err := h.paService.CheckPA(ctx, req)
		if err != nil {
			writeErrorResponse(w, http.StatusInternalServerError, "Failed to check PA: "+err.Error())
			return
		}

		writeJSONResponse(w, http.StatusOK, response)
		return
	}

	// For POST, parse the full request body
	var req models.PACheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.DrugRxNorm == "" || req.PatientID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: drug_rxnorm and patient_id")
		return
	}

	response, err := h.paService.CheckPA(ctx, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check PA: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// SubmitPA handles POST /api/v1/pa/submit requests
// Submits a new PA request
func (h *PAHandler) SubmitPA(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var req models.PASubmitRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.DrugRxNorm == "" || req.PatientID == "" || req.ProviderID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: drug_rxnorm, patient_id, and provider_id")
		return
	}

	if req.Quantity <= 0 || req.DaysSupply <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Quantity and days_supply must be positive")
		return
	}

	// Submit PA
	submission, err := h.paService.SubmitPA(ctx, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to submit PA: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusCreated, map[string]interface{}{
		"success":    true,
		"submission": submission,
		"message":    "PA submission created successfully",
	})
}

// GetStatus handles GET /api/v1/pa/status requests
// Returns the status of a PA submission
func (h *PAHandler) GetStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	paID := r.URL.Query().Get("id")
	if paID == "" {
		paID = r.URL.Query().Get("pa_id")
	}

	if paID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: id")
		return
	}

	response, err := h.paService.GetStatus(ctx, paID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// ListPending handles GET /api/v1/pa/pending requests
// Lists pending PA submissions
func (h *PAHandler) ListPending(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	limit := parseIntParam(r.URL.Query().Get("limit"), 50)
	offset := parseIntParam(r.URL.Query().Get("offset"), 0)

	submissions, err := h.paService.ListPending(ctx, limit, offset)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to list pending PA submissions: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, map[string]interface{}{
		"submissions": submissions,
		"count":       len(submissions),
		"limit":       limit,
		"offset":      offset,
	})
}
