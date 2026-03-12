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

// STHandler handles HTTP requests for Step Therapy operations
type STHandler struct {
	stService *services.STService
}

// NewSTHandler creates a new STHandler
func NewSTHandler(stService *services.STService) *STHandler {
	return &STHandler{
		stService: stService,
	}
}

// GetRequirements handles GET /api/v1/steptherapy/requirements requests
// Returns step therapy requirements for a specific drug
func (h *STHandler) GetRequirements(w http.ResponseWriter, r *http.Request) {
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
	req := &models.STRequirementsRequest{
		DrugRxNorm: rxnormCode,
	}
	if payerID != "" {
		req.PayerID = &payerID
	}
	if planID != "" {
		req.PlanID = &planID
	}

	// Get requirements
	response, err := h.stService.GetRequirements(ctx, req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get ST requirements: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// CheckStepTherapy handles POST /api/v1/steptherapy/check requests
// Evaluates step therapy requirements against patient drug history
func (h *STHandler) CheckStepTherapy(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	var req models.STCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.DrugRxNorm == "" || req.PatientID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: drug_rxnorm and patient_id")
		return
	}

	response, err := h.stService.CheckStepTherapy(ctx, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check step therapy: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// RequestOverride handles POST /api/v1/steptherapy/override requests
// Submits a step therapy override request
func (h *STHandler) RequestOverride(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var req models.STOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	// Validate required fields
	if req.DrugRxNorm == "" || req.PatientID == "" || req.ProviderID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: drug_rxnorm, patient_id, and provider_id")
		return
	}

	if req.OverrideReason == "" || req.ClinicalJustification == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: override_reason and clinical_justification")
		return
	}

	response, err := h.stService.RequestOverride(ctx, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create override request: "+err.Error())
		return
	}

	status := http.StatusCreated
	if response.Override.Status == models.STOverrideApproved {
		status = http.StatusOK
	}

	writeJSONResponse(w, status, response)
}

// GetOverrideStatus handles GET /api/v1/steptherapy/override/status requests
// Returns the status of an override request
func (h *STHandler) GetOverrideStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	overrideID := r.URL.Query().Get("id")
	if overrideID == "" {
		overrideID = r.URL.Query().Get("override_id")
	}

	if overrideID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: id")
		return
	}

	response, err := h.stService.GetOverrideStatus(ctx, overrideID)
	if err != nil {
		writeErrorResponse(w, http.StatusNotFound, err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}
