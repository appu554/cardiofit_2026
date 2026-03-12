package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"kb-formulary/internal/services"
)

// InventoryHandler handles HTTP requests for inventory operations
type InventoryHandler struct {
	inventoryService *services.InventoryService
}

// NewInventoryHandler creates a new InventoryHandler
func NewInventoryHandler(inventoryService *services.InventoryService) *InventoryHandler {
	return &InventoryHandler{
		inventoryService: inventoryService,
	}
}

// GetStock handles GET /api/v1/inventory/stock requests
func (h *InventoryHandler) GetStock(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	drugID := r.URL.Query().Get("drug_id")
	pharmacyID := r.URL.Query().Get("pharmacy_id")
	networkID := r.URL.Query().Get("network_id")
	zipCode := r.URL.Query().Get("zip_code")
	includeAlternatives := r.URL.Query().Get("include_alternatives") == "true"

	// Validate required parameters
	if drugID == "" {
		http.Error(w, "Missing required parameter: drug_id", http.StatusBadRequest)
		return
	}

	// Create stock request
	request := services.StockRequest{
		DrugID:              drugID,
		PharmacyID:          pharmacyID,
		NetworkID:           networkID,
		ZipCode:             zipCode,
		IncludeAlternatives: includeAlternatives,
		RequestID:           generateRequestID(),
	}

	// Get stock information
	response, err := h.inventoryService.GetStock(ctx, request)
	if err != nil {
		http.Error(w, "Failed to get stock information: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=30") // 30 seconds cache for stock data

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetAvailability handles GET /api/v1/inventory/availability requests
func (h *InventoryHandler) GetAvailability(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	drugID := r.URL.Query().Get("drug_id")
	location := r.URL.Query().Get("location")
	radiusStr := r.URL.Query().Get("radius_miles")
	sortBy := r.URL.Query().Get("sort_by")
	limitStr := r.URL.Query().Get("limit")

	// Validate required parameters
	if drugID == "" || location == "" {
		http.Error(w, "Missing required parameters: drug_id and location", http.StatusBadRequest)
		return
	}

	// Parse optional parameters
	radius := 10.0 // default 10 miles
	if radiusStr != "" {
		if parsed, err := strconv.ParseFloat(radiusStr, 64); err == nil && parsed > 0 {
			radius = parsed
		}
	}

	limit := 20 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if sortBy == "" {
		sortBy = "distance" // default sort
	}

	// Create availability request
	request := services.AvailabilityRequest{
		DrugID:      drugID,
		Location:    location,
		RadiusMiles: radius,
		SortBy:      sortBy,
		Limit:       limit,
		RequestID:   generateRequestID(),
	}

	// Get availability information
	response, err := h.inventoryService.GetAvailability(ctx, request)
	if err != nil {
		http.Error(w, "Failed to get availability: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60") // 1 minute cache for availability

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetPricing handles GET /api/v1/inventory/pricing requests
func (h *InventoryHandler) GetPricing(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	drugID := r.URL.Query().Get("drug_id")
	pharmacyID := r.URL.Query().Get("pharmacy_id")
	quantity := r.URL.Query().Get("quantity")
	payerID := r.URL.Query().Get("payer_id")
	memberID := r.URL.Query().Get("member_id")

	// Validate required parameters
	if drugID == "" {
		http.Error(w, "Missing required parameter: drug_id", http.StatusBadRequest)
		return
	}

	// Parse quantity
	qty := 30 // default 30-day supply
	if quantity != "" {
		if parsed, err := strconv.Atoi(quantity); err == nil && parsed > 0 {
			qty = parsed
		}
	}

	// Create pricing request
	request := services.PricingRequest{
		DrugID:     drugID,
		PharmacyID: pharmacyID,
		Quantity:   qty,
		PayerID:    payerID,
		MemberID:   memberID,
		RequestID:  generateRequestID(),
	}

	// Get pricing information
	response, err := h.inventoryService.GetPricing(ctx, request)
	if err != nil {
		http.Error(w, "Failed to get pricing: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=300") // 5 minutes cache for pricing

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ReserveStock handles POST /api/v1/inventory/reserve requests
func (h *InventoryHandler) ReserveStock(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 15*time.Second)
	defer cancel()

	// Only accept POST requests
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse request body
	var request services.ReserveRequest
	if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
		http.Error(w, "Invalid request body: "+err.Error(), http.StatusBadRequest)
		return
	}

	// Add request ID if not provided
	if request.RequestID == "" {
		request.RequestID = generateRequestID()
	}

	// Validate required fields
	if request.DrugID == "" || request.PharmacyID == "" || request.Quantity <= 0 {
		http.Error(w, "Missing required fields: drug_id, pharmacy_id, quantity", http.StatusBadRequest)
		return
	}

	// Reserve stock
	response, err := h.inventoryService.ReserveStock(ctx, request)
	if err != nil {
		if err.Error() == "insufficient stock" {
			http.Error(w, "Insufficient stock available", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to reserve stock: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// ReleaseReservation handles DELETE /api/v1/inventory/reserve/{reservation_id} requests
func (h *InventoryHandler) ReleaseReservation(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Only accept DELETE requests
	if r.Method != http.MethodDelete {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Parse path parameter (reservation ID)
	reservationID := r.URL.Path[len("/api/v1/inventory/reserve/"):]
	if reservationID == "" {
		http.Error(w, "Missing reservation ID in path", http.StatusBadRequest)
		return
	}

	// Release reservation
	err := h.inventoryService.ReleaseReservation(ctx, reservationID)
	if err != nil {
		if err.Error() == "reservation not found" {
			http.Error(w, "Reservation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to release reservation: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Return success response
	w.WriteHeader(http.StatusNoContent)
}

// GetReservationStatus handles GET /api/v1/inventory/reserve/{reservation_id} requests
func (h *InventoryHandler) GetReservationStatus(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Parse path parameter (reservation ID)
	reservationID := r.URL.Path[len("/api/v1/inventory/reserve/"):]
	if reservationID == "" {
		http.Error(w, "Missing reservation ID in path", http.StatusBadRequest)
		return
	}

	// Get reservation status
	reservation, err := h.inventoryService.GetReservationStatus(ctx, reservationID)
	if err != nil {
		if err.Error() == "reservation not found" {
			http.Error(w, "Reservation not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Failed to get reservation status: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-cache") // Don't cache reservation status

	// Return response
	if err := json.NewEncoder(w).Encode(reservation); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// GetLowStockAlerts handles GET /api/v1/inventory/alerts requests
func (h *InventoryHandler) GetLowStockAlerts(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Parse query parameters
	pharmacyID := r.URL.Query().Get("pharmacy_id")
	networkID := r.URL.Query().Get("network_id")
	severityLevel := r.URL.Query().Get("severity")
	limitStr := r.URL.Query().Get("limit")

	// Parse limit
	limit := 50 // default
	if limitStr != "" {
		if parsed, err := strconv.Atoi(limitStr); err == nil && parsed > 0 && parsed <= 200 {
			limit = parsed
		}
	}

	// Create alerts request
	request := services.AlertsRequest{
		PharmacyID:    pharmacyID,
		NetworkID:     networkID,
		SeverityLevel: severityLevel,
		Limit:         limit,
		RequestID:     generateRequestID(),
	}

	// Get low stock alerts
	response, err := h.inventoryService.GetLowStockAlerts(ctx, request)
	if err != nil {
		http.Error(w, "Failed to get alerts: "+err.Error(), http.StatusInternalServerError)
		return
	}

	// Set response headers
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "max-age=60") // 1 minute cache for alerts

	// Return response
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "Failed to encode response", http.StatusInternalServerError)
		return
	}
}

// HealthCheck handles GET /health requests for inventory service
func (h *InventoryHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Perform health check
	health := h.inventoryService.HealthCheck(ctx)

	// Set response headers
	w.Header().Set("Content-Type", "application/json")

	// Set status code based on health
	if health.Status == "healthy" {
		w.WriteHeader(http.StatusOK)
	} else {
		w.WriteHeader(http.StatusServiceUnavailable)
	}

	// Return health status
	if err := json.NewEncoder(w).Encode(health); err != nil {
		http.Error(w, "Failed to encode health response", http.StatusInternalServerError)
		return
	}
}