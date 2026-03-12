// Package handlers provides HTTP request handlers for KB-6 Formulary Service.
package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"kb-formulary/internal/models"
	"kb-formulary/internal/services"
)

// QLHandler handles HTTP requests for Quantity Limit operations
type QLHandler struct {
	qlService *services.QLService
}

// NewQLHandler creates a new QLHandler
func NewQLHandler(qlService *services.QLService) *QLHandler {
	return &QLHandler{
		qlService: qlService,
	}
}

// CheckQuantityLimits handles GET /api/v1/quantitylimit/check requests
// Validates a prescription against quantity limits
func (h *QLHandler) CheckQuantityLimits(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	rxnormCode := r.URL.Query().Get("rxnorm")
	if rxnormCode == "" {
		rxnormCode = r.URL.Query().Get("drug_rxnorm")
	}
	quantityStr := r.URL.Query().Get("quantity")
	daysSupplyStr := r.URL.Query().Get("days_supply")
	payerID := r.URL.Query().Get("payer_id")
	planID := r.URL.Query().Get("plan_id")
	patientID := r.URL.Query().Get("patient_id")
	fillsStr := r.URL.Query().Get("fills_this_year")

	// Validate required parameters
	if rxnormCode == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: rxnorm")
		return
	}

	quantity, err := strconv.Atoi(quantityStr)
	if err != nil || quantity <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid quantity parameter")
		return
	}

	daysSupply, err := strconv.Atoi(daysSupplyStr)
	if err != nil || daysSupply <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid days_supply parameter")
		return
	}

	// Build request
	req := &models.QLCheckRequest{
		DrugRxNorm: rxnormCode,
		Quantity:   quantity,
		DaysSupply: daysSupply,
	}

	if payerID != "" {
		req.PayerID = &payerID
	}
	if planID != "" {
		req.PlanID = &planID
	}
	if patientID != "" {
		req.PatientID = &patientID
	}
	if fillsStr != "" {
		if fills, err := strconv.Atoi(fillsStr); err == nil {
			req.FillsThisYear = fills
		}
	}

	response, err := h.qlService.CheckQuantityLimits(ctx, req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check quantity limits: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// CheckQuantityLimitsPost handles POST /api/v1/quantitylimit/check requests
// Validates a prescription against quantity limits with full request body
func (h *QLHandler) CheckQuantityLimitsPost(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	var req models.QLCheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.DrugRxNorm == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required field: drug_rxnorm")
		return
	}

	if req.Quantity <= 0 || req.DaysSupply <= 0 {
		writeErrorResponse(w, http.StatusBadRequest, "Quantity and days_supply must be positive")
		return
	}

	response, err := h.qlService.CheckQuantityLimits(ctx, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to check quantity limits: "+err.Error())
		return
	}

	writeJSONResponse(w, http.StatusOK, response)
}

// RequestOverride handles POST /api/v1/quantitylimit/override requests
// Submits a quantity limit override request
func (h *QLHandler) RequestOverride(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	var req models.QLOverrideRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErrorResponse(w, http.StatusBadRequest, "Invalid request body: "+err.Error())
		return
	}

	if req.DrugRxNorm == "" || req.PatientID == "" || req.ProviderID == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required fields: drug_rxnorm, patient_id, and provider_id")
		return
	}

	if req.OverrideReason == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required field: override_reason")
		return
	}

	response, err := h.qlService.RequestOverride(ctx, &req)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to create override request: "+err.Error())
		return
	}

	status := http.StatusCreated
	if response.Approved {
		status = http.StatusOK
	}

	writeJSONResponse(w, status, response)
}

// GetLimits handles GET /api/v1/quantitylimit/limits requests
// Returns quantity limits for a drug without performing a check
func (h *QLHandler) GetLimits(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	rxnormCode := r.URL.Query().Get("rxnorm")
	if rxnormCode == "" {
		rxnormCode = r.URL.Query().Get("drug_rxnorm")
	}
	payerID := r.URL.Query().Get("payer_id")
	planID := r.URL.Query().Get("plan_id")

	if rxnormCode == "" {
		writeErrorResponse(w, http.StatusBadRequest, "Missing required parameter: rxnorm")
		return
	}

	var payerPtr, planPtr *string
	if payerID != "" {
		payerPtr = &payerID
	}
	if planID != "" {
		planPtr = &planID
	}

	limits, drugName, err := h.qlService.GetLimitsForDrug(ctx, rxnormCode, payerPtr, planPtr)
	if err != nil {
		writeErrorResponse(w, http.StatusInternalServerError, "Failed to get quantity limits: "+err.Error())
		return
	}

	response := map[string]interface{}{
		"drug_rxnorm": rxnormCode,
		"drug_name":   drugName,
		"limits":      limits,
		"has_limits":  limits != nil,
	}

	writeJSONResponse(w, http.StatusOK, response)
}
