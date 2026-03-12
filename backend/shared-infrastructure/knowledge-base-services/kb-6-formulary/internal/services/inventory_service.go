package services

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"kb-formulary/internal/cache"
	"kb-formulary/internal/database"
)

// InventoryService provides stock availability and inventory management functionality
type InventoryService struct {
	db    *database.Connection
	cache *cache.RedisManager
}

// NewInventoryService creates a new inventory service instance
func NewInventoryService(db *database.Connection, cache *cache.RedisManager) *InventoryService {
	return &InventoryService{
		db:    db,
		cache: cache,
	}
}

// StockCheckRequest represents a stock availability check request
type StockCheckRequest struct {
	TransactionID        string
	DrugRxNorm          string
	LocationID          string
	IncludeLots         bool
	IncludeAlternatives bool
	RequiredQuantity    int
}

// StockResponse represents stock availability information
type StockResponse struct {
	DatasetVersion    string
	LocationID        string
	DrugRxNorm        string
	QuantityOnHand    int
	QuantityAllocated int
	QuantityAvailable int
	InStock           bool
	SufficientStock   bool
	Lots              []LotDetail
	ReorderInfo       *ReorderInfo
	AlternativeStock  []AlternativeStock
	Alerts            []StockAlert
	DemandForecast    *DemandPrediction
	Evidence          *EvidenceEnvelope
}

// LotDetail represents lot-level inventory information
type LotDetail struct {
	LotNumber      string
	Quantity       int
	ExpirationDate time.Time
	Manufacturer   string
	UnitCost       float64
}

// ReorderInfo represents reorder point and quantity information
type ReorderInfo struct {
	ReorderPoint       int
	ReorderQuantity    int
	MaxStockLevel      int
	ReorderRecommended bool
	DaysUntilStockout  int
}

// AlternativeStock represents alternative stock at other locations
type AlternativeStock struct {
	DrugRxNorm        string
	DrugName          string
	AlternativeType   string
	QuantityAvailable int
	LocationID        string
	DistanceKm        float64
}

// StockAlert represents stock alerts and notifications
type StockAlert struct {
	AlertType         string
	Severity          string
	Message           string
	RecommendedAction string
	TriggeredAt       time.Time
}

// DemandPrediction represents demand forecasting information
type DemandPrediction struct {
	PredictedDemand7d  int
	PredictedDemand30d int
	ConfidenceScore    float64
	StockoutRisk       float64
}

// CheckStock checks stock availability for a drug at a specific location
func (is *InventoryService) CheckStock(ctx context.Context, req *StockCheckRequest) (*StockResponse, error) {
	start := time.Now()

	// Set defaults
	if req.RequiredQuantity == 0 {
		req.RequiredQuantity = 1
	}

	// Try cache first
	cacheKey := fmt.Sprintf("stock:%s:%s", req.LocationID, req.DrugRxNorm)
	
	if cachedData, err := is.cache.GetStock(cacheKey); err == nil && cachedData != nil {
		log.Printf("Cache hit for stock check: %s", req.TransactionID)
		var cachedResponse StockResponse
		if err := json.Unmarshal(cachedData, &cachedResponse); err == nil {
			cachedResponse.Evidence.DecisionHash = is.generateStockDecisionHash(req)
			if cachedResponse.Evidence.Provenance != nil {
				cachedResponse.Evidence.Provenance["cache_status"] = "hit"
				cachedResponse.Evidence.Provenance["cache_retrieval_time"] = time.Now().Format(time.RFC3339)
			}
			return &cachedResponse, nil
		}
	}

	// Query database for stock information
	stock, err := is.getStockFromDatabase(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to get stock information: %w", err)
	}

	// Get lot details if requested
	if req.IncludeLots {
		lots, err := is.getLotDetails(ctx, req.DrugRxNorm, req.LocationID)
		if err != nil {
			log.Printf("Warning: failed to get lot details: %v", err)
		} else {
			stock.Lots = lots
		}
	}

	// Get alternative stock if requested and primary stock is insufficient
	if req.IncludeAlternatives && (!stock.SufficientStock || stock.QuantityAvailable < req.RequiredQuantity) {
		alternatives, err := is.findAlternativeStock(ctx, req.DrugRxNorm, req.LocationID, req.RequiredQuantity)
		if err != nil {
			log.Printf("Warning: failed to find alternative stock: %v", err)
		} else {
			stock.AlternativeStock = alternatives
		}
	}

	// Get stock alerts
	alerts, err := is.getStockAlerts(ctx, req.DrugRxNorm, req.LocationID)
	if err != nil {
		log.Printf("Warning: failed to get stock alerts: %v", err)
	} else {
		stock.Alerts = alerts
	}

	// Get demand forecast
	forecast, err := is.getDemandForecast(ctx, req.DrugRxNorm, req.LocationID)
	if err != nil {
		log.Printf("Warning: failed to get demand forecast: %v", err)
	} else {
		stock.DemandForecast = forecast
	}

	// Add evidence envelope
	stock.Evidence = &EvidenceEnvelope{
		DatasetVersion:   "kb6.inventory.2025Q3.v1",
		DatasetTimestamp: time.Now(),
		SourceSystem:     "kb-6-formulary",
		DecisionHash:     is.generateStockDecisionHash(req),
		DataSources:      []string{"inventory_management", "demand_prediction"},
		KB7Version:       "kb7.2025Q3.v1",
		Provenance: map[string]string{
			"query_time":     time.Now().Format(time.RFC3339),
			"transaction_id": req.TransactionID,
			"cache_status":   "miss",
		},
	}

	// Cache the response (shorter TTL for dynamic stock data)
	cacheTTL := 30 * time.Second
	if stock.QuantityAvailable > 0 {
		cacheTTL = 2 * time.Minute // Cache longer if in stock
	}
	
	if data, err := json.Marshal(stock); err == nil {
		if err := is.cache.SetStock(cacheKey, data, cacheTTL); err != nil {
			log.Printf("Warning: failed to cache stock response: %v", err)
		}
	} else {
		log.Printf("Warning: failed to serialize stock response for caching: %v", err)
	}

	// Log performance
	duration := time.Since(start)
	log.Printf("CheckStock completed in %v for transaction %s", duration, req.TransactionID)

	return stock, nil
}

// getStockFromDatabase queries the database for current stock levels
func (is *InventoryService) getStockFromDatabase(ctx context.Context, req *StockCheckRequest) (*StockResponse, error) {
	query := `
		SELECT 
			location_id,
			location_name,
			drug_rxnorm,
			SUM(quantity_on_hand) as total_on_hand,
			SUM(quantity_allocated) as total_allocated,
			SUM(quantity_available) as total_available,
			MIN(reorder_point) as reorder_point,
			MIN(reorder_quantity) as reorder_quantity,
			MAX(max_stock_level) as max_stock_level
		FROM drug_inventory 
		WHERE drug_rxnorm = $1 
			AND location_id = $2
			AND expiration_date > CURRENT_DATE
		GROUP BY location_id, location_name, drug_rxnorm`

	var response StockResponse
	var locationName string
	var reorderPoint, reorderQuantity, maxStockLevel interface{}

	err := is.db.QueryRow(ctx, query, req.DrugRxNorm, req.LocationID).Scan(
		&response.LocationID,
		&locationName,
		&response.DrugRxNorm,
		&response.QuantityOnHand,
		&response.QuantityAllocated,
		&response.QuantityAvailable,
		&reorderPoint,
		&reorderQuantity,
		&maxStockLevel,
	)

	if err != nil {
		// No stock found
		response.LocationID = req.LocationID
		response.DrugRxNorm = req.DrugRxNorm
		response.QuantityOnHand = 0
		response.QuantityAllocated = 0
		response.QuantityAvailable = 0
		response.InStock = false
		response.SufficientStock = false
	} else {
		// Stock found
		response.InStock = response.QuantityAvailable > 0
		response.SufficientStock = response.QuantityAvailable >= req.RequiredQuantity

		// Build reorder info
		response.ReorderInfo = &ReorderInfo{
			ReorderRecommended: response.QuantityAvailable <= 0,
		}

		if reorderPoint != nil {
			if point, ok := reorderPoint.(int); ok {
				response.ReorderInfo.ReorderPoint = point
				response.ReorderInfo.ReorderRecommended = response.QuantityAvailable <= point
			}
		}

		if reorderQuantity != nil {
			if qty, ok := reorderQuantity.(int); ok {
				response.ReorderInfo.ReorderQuantity = qty
			}
		}

		if maxStockLevel != nil {
			if level, ok := maxStockLevel.(int); ok {
				response.ReorderInfo.MaxStockLevel = level
			}
		}

		// Estimate days until stockout based on historical usage
		response.ReorderInfo.DaysUntilStockout = is.estimateDaysUntilStockout(
			response.QuantityAvailable, req.DrugRxNorm, req.LocationID)
	}

	response.DatasetVersion = "kb6.inventory.2025Q3.v1"
	return &response, nil
}

// getLotDetails retrieves lot-level inventory details
func (is *InventoryService) getLotDetails(ctx context.Context, drugRxNorm, locationID string) ([]LotDetail, error) {
	query := `
		SELECT 
			lot_number,
			quantity_on_hand,
			expiration_date,
			manufacturer,
			unit_cost
		FROM drug_inventory 
		WHERE drug_rxnorm = $1 
			AND location_id = $2
			AND quantity_on_hand > 0
			AND expiration_date > CURRENT_DATE
		ORDER BY expiration_date ASC`

	rows, err := is.db.Query(ctx, query, drugRxNorm, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query lot details: %w", err)
	}
	defer rows.Close()

	var lots []LotDetail
	for rows.Next() {
		var lot LotDetail
		var unitCost interface{}

		err := rows.Scan(
			&lot.LotNumber,
			&lot.Quantity,
			&lot.ExpirationDate,
			&lot.Manufacturer,
			&unitCost,
		)
		if err != nil {
			log.Printf("Warning: failed to scan lot detail: %v", err)
			continue
		}

		if unitCost != nil {
			if cost, ok := unitCost.(float64); ok {
				lot.UnitCost = cost
			}
		}

		lots = append(lots, lot)
	}

	return lots, nil
}

// findAlternativeStock finds alternative stock at other locations
func (is *InventoryService) findAlternativeStock(ctx context.Context, drugRxNorm, primaryLocationID string, requiredQuantity int) ([]AlternativeStock, error) {
	query := `
		SELECT DISTINCT
			di.location_id,
			di.location_name,
			di.drug_rxnorm,
			'same_drug' as alternative_type,
			SUM(di.quantity_available) as total_available
		FROM drug_inventory di
		WHERE di.drug_rxnorm = $1
			AND di.location_id != $2
			AND di.quantity_available > 0
			AND di.expiration_date > CURRENT_DATE
		GROUP BY di.location_id, di.location_name, di.drug_rxnorm
		HAVING SUM(di.quantity_available) >= $3
		ORDER BY SUM(di.quantity_available) DESC
		LIMIT 5`

	rows, err := is.db.Query(ctx, query, drugRxNorm, primaryLocationID, requiredQuantity)
	if err != nil {
		return nil, fmt.Errorf("failed to query alternative stock: %w", err)
	}
	defer rows.Close()

	var alternatives []AlternativeStock
	for rows.Next() {
		var alt AlternativeStock
		var locationName string

		err := rows.Scan(
			&alt.LocationID,
			&locationName,
			&alt.DrugRxNorm,
			&alt.AlternativeType,
			&alt.QuantityAvailable,
		)
		if err != nil {
			log.Printf("Warning: failed to scan alternative stock: %v", err)
			continue
		}

		alt.DrugName = "Unknown" // TODO: Get from drug name service
		alt.DistanceKm = is.calculateDistance(primaryLocationID, alt.LocationID)

		alternatives = append(alternatives, alt)
	}

	return alternatives, nil
}

// getStockAlerts retrieves active stock alerts for a drug at a location
func (is *InventoryService) getStockAlerts(ctx context.Context, drugRxNorm, locationID string) ([]StockAlert, error) {
	query := `
		SELECT 
			alert_type,
			severity,
			message,
			recommended_action,
			triggered_at
		FROM stock_alerts 
		WHERE drug_rxnorm = $1 
			AND location_id = $2
			AND resolved = false
		ORDER BY triggered_at DESC
		LIMIT 10`

	rows, err := is.db.Query(ctx, query, drugRxNorm, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query stock alerts: %w", err)
	}
	defer rows.Close()

	var alerts []StockAlert
	for rows.Next() {
		var alert StockAlert

		err := rows.Scan(
			&alert.AlertType,
			&alert.Severity,
			&alert.Message,
			&alert.RecommendedAction,
			&alert.TriggeredAt,
		)
		if err != nil {
			log.Printf("Warning: failed to scan stock alert: %v", err)
			continue
		}

		alerts = append(alerts, alert)
	}

	return alerts, nil
}

// getDemandForecast gets demand prediction for a drug at a location
func (is *InventoryService) getDemandForecast(ctx context.Context, drugRxNorm, locationID string) (*DemandPrediction, error) {
	// Use the database function we created in the schema
	query := `
		SELECT 
			predicted_demand,
			confidence_interval_low,
			confidence_interval_high,
			reorder_recommended,
			stockout_risk
		FROM predict_demand($1, $2, 7)`

	var prediction DemandPrediction
	var confidenceIntervalLow, confidenceIntervalHigh int
	var reorderRecommended bool

	err := is.db.QueryRow(ctx, query, locationID, drugRxNorm).Scan(
		&prediction.PredictedDemand7d,
		&confidenceIntervalLow,
		&confidenceIntervalHigh,
		&reorderRecommended,
		&prediction.StockoutRisk,
	)

	if err != nil {
		// Return default prediction if function fails
		return &DemandPrediction{
			PredictedDemand7d:  0,
			PredictedDemand30d: 0,
			ConfidenceScore:    0.0,
			StockoutRisk:       0.0,
		}, nil
	}

	// Calculate confidence score based on interval width
	intervalWidth := confidenceIntervalHigh - confidenceIntervalLow
	if prediction.PredictedDemand7d > 0 {
		prediction.ConfidenceScore = 1.0 - (float64(intervalWidth) / float64(prediction.PredictedDemand7d))
		if prediction.ConfidenceScore < 0 {
			prediction.ConfidenceScore = 0
		}
		if prediction.ConfidenceScore > 1 {
			prediction.ConfidenceScore = 1
		}
	}

	// Get 30-day prediction
	var predicted30d int
	err30d := is.db.QueryRow(ctx, `SELECT predicted_demand FROM predict_demand($1, $2, 30)`, locationID, drugRxNorm).Scan(&predicted30d)
	if err30d == nil {
		prediction.PredictedDemand30d = predicted30d
	}

	return &prediction, nil
}

// estimateDaysUntilStockout estimates days until stockout based on usage patterns
func (is *InventoryService) estimateDaysUntilStockout(currentStock int, drugRxNorm, locationID string) int {
	if currentStock <= 0 {
		return 0
	}

	// Simple estimation - in production, use more sophisticated algorithm
	// Average usage of 1-2 units per day for most drugs
	averageDailyUsage := 2.0
	
	// TODO: Query actual historical usage from demand_history table
	
	daysLeft := int(float64(currentStock) / averageDailyUsage)
	if daysLeft < 0 {
		return 0
	}
	
	return daysLeft
}

// calculateDistance calculates distance between two locations (mock implementation)
func (is *InventoryService) calculateDistance(location1, location2 string) float64 {
	// Mock implementation - in production, use actual geolocation data
	return 5.0 // Default 5km distance
}

// generateStockDecisionHash generates a hash for stock decision reproducibility
func (is *InventoryService) generateStockDecisionHash(req *StockCheckRequest) string {
	return fmt.Sprintf("stock_hash_%s_%s_%d_%d", 
		req.DrugRxNorm, req.LocationID, req.RequiredQuantity, time.Now().Unix())
}

// UpdateStock updates stock levels (for real-time inventory updates)
func (is *InventoryService) UpdateStock(ctx context.Context, updates []StockUpdate) error {
	start := time.Now()

	// Begin transaction
	tx, err := is.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Process each update
	for _, update := range updates {
		if err := is.processStockUpdate(ctx, tx, &update); err != nil {
			return fmt.Errorf("failed to process stock update for %s at %s: %w", 
				update.DrugRxNorm, update.LocationID, err)
		}

		// Invalidate cache for this location
		if err := is.cache.InvalidateStock(update.LocationID); err != nil {
			log.Printf("Warning: failed to invalidate cache: %v", err)
		}
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit stock updates: %w", err)
	}

	log.Printf("Updated stock for %d items in %v", len(updates), time.Since(start))
	return nil
}

// StockUpdate represents a stock level update
type StockUpdate struct {
	DrugRxNorm        string
	LocationID        string
	LotNumber         string
	QuantityChange    int    // Positive for receipts, negative for dispensing
	UpdateType        string // received, dispensed, allocated, adjusted
	TransactionID     string
	UpdatedBy         string
	Notes             string
}

// processStockUpdate processes a single stock update
func (is *InventoryService) processStockUpdate(ctx context.Context, tx database.Tx, update *StockUpdate) error {
	// Update inventory levels
	query := `
		UPDATE drug_inventory 
		SET quantity_on_hand = quantity_on_hand + $1,
			updated_at = NOW()
		WHERE drug_rxnorm = $2 
			AND location_id = $3 
			AND lot_number = $4`

	result, err := tx.Exec(ctx, query, 
		update.QuantityChange, 
		update.DrugRxNorm, 
		update.LocationID, 
		update.LotNumber)
	
	if err != nil {
		return fmt.Errorf("failed to update inventory: %w", err)
	}

	rowsAffected := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no inventory record found for %s at %s lot %s", 
			update.DrugRxNorm, update.LocationID, update.LotNumber)
	}

	// Log the update for audit trail
	auditQuery := `
		INSERT INTO inventory_audit_log (
			drug_rxnorm, location_id, lot_number, 
			quantity_change, update_type, transaction_id,
			updated_by, notes, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())`

	_, err = tx.Exec(ctx, auditQuery,
		update.DrugRxNorm,
		update.LocationID,
		update.LotNumber,
		update.QuantityChange,
		update.UpdateType,
		update.TransactionID,
		update.UpdatedBy,
		update.Notes,
	)

	if err != nil {
		log.Printf("Warning: failed to log inventory audit: %v", err)
		// Don't fail the update for audit logging issues
	}

	return nil
}

// GetLocationInventory gets all inventory for a specific location
func (is *InventoryService) GetLocationInventory(ctx context.Context, locationID string) ([]StockResponse, error) {
	query := `
		SELECT DISTINCT
			drug_rxnorm,
			SUM(quantity_on_hand) as total_on_hand,
			SUM(quantity_allocated) as total_allocated,
			SUM(quantity_available) as total_available
		FROM drug_inventory
		WHERE location_id = $1
			AND expiration_date > CURRENT_DATE
		GROUP BY drug_rxnorm
		ORDER BY drug_rxnorm`

	rows, err := is.db.Query(ctx, query, locationID)
	if err != nil {
		return nil, fmt.Errorf("failed to query location inventory: %w", err)
	}
	defer rows.Close()

	var inventory []StockResponse
	for rows.Next() {
		var stock StockResponse
		
		err := rows.Scan(
			&stock.DrugRxNorm,
			&stock.QuantityOnHand,
			&stock.QuantityAllocated,
			&stock.QuantityAvailable,
		)
		if err != nil {
			log.Printf("Warning: failed to scan inventory row: %v", err)
			continue
		}

		stock.LocationID = locationID
		stock.InStock = stock.QuantityAvailable > 0
		stock.SufficientStock = stock.QuantityAvailable > 0
		stock.DatasetVersion = "kb6.inventory.2025Q3.v1"

		inventory = append(inventory, stock)
	}

	return inventory, nil
}

// GetStock retrieves stock information for a drug
func (is *InventoryService) GetStock(ctx context.Context, req StockRequest) (*StockInfoResponse, error) {
	log.Printf("Getting stock for drug %s", req.DrugID)

	// Check cache first
	cacheKey := fmt.Sprintf("stock:%s", req.DrugID)
	if cachedData, err := is.cache.GetCoverage(cacheKey); err == nil && cachedData != nil {
		var cachedResponse StockInfoResponse
		if err := json.Unmarshal(cachedData, &cachedResponse); err == nil {
			log.Printf("Returning cached stock for drug %s", req.DrugID)
			return &cachedResponse, nil
		}
	}

	// TODO: Query database for actual stock
	// For now, return mock data
	stockInfo := []PharmacyStock{
		{
			PharmacyID:      "ph-001",
			PharmacyName:    "Downtown Pharmacy",
			Address:         "123 Main St, City, ST 12345",
			Phone:           "(555) 123-4567",
			DistanceMiles:   2.3,
			InStock:         true,
			QuantityOnHand:  45,
			LastUpdated:     time.Now().Add(-30 * time.Minute),
		},
		{
			PharmacyID:      "ph-002", 
			PharmacyName:    "Suburban Drugs",
			Address:         "456 Oak Ave, City, ST 12345",
			Phone:           "(555) 987-6543",
			DistanceMiles:   5.1,
			InStock:         true,
			QuantityOnHand:  12,
			LastUpdated:     time.Now().Add(-15 * time.Minute),
		},
	}

	response := &StockInfoResponse{
		DrugID:    req.DrugID,
		StockInfo: stockInfo,
		RequestID: req.RequestID,
		Timestamp: time.Now(),
	}

	// Cache the response
	if responseData, err := json.Marshal(response); err == nil {
		is.cache.SetCoverage(cacheKey, responseData, 30*time.Second)
	}

	return response, nil
}

// GetAvailability finds drug availability within a geographic area
func (is *InventoryService) GetAvailability(ctx context.Context, req AvailabilityRequest) (*AvailabilityResponse, error) {
	log.Printf("Getting availability for drug %s near %s", req.DrugID, req.Location)

	// TODO: Query database based on location and radius
	// For now, return mock data
	pharmacies := []PharmacyStock{
		{
			PharmacyID:      "ph-003",
			PharmacyName:    "Central Pharmacy",
			Address:         "789 Center St, City, ST 12345",
			Phone:           "(555) 555-0123",
			DistanceMiles:   1.2,
			InStock:         true,
			QuantityOnHand:  28,
			LastUpdated:     time.Now().Add(-20 * time.Minute),
		},
		{
			PharmacyID:      "ph-004",
			PharmacyName:    "East Side Drugs", 
			Address:         "321 East Blvd, City, ST 12345",
			Phone:           "(555) 555-0456",
			DistanceMiles:   3.7,
			InStock:         false,
			QuantityOnHand:  0,
			LastUpdated:     time.Now().Add(-45 * time.Minute),
		},
	}

	response := &AvailabilityResponse{
		DrugID:     req.DrugID,
		Location:   req.Location,
		Pharmacies: pharmacies,
		TotalFound: len(pharmacies),
		RequestID:  req.RequestID,
		Timestamp:  time.Now(),
	}

	return response, nil
}

// GetPricing retrieves pricing information for a drug
func (is *InventoryService) GetPricing(ctx context.Context, req PricingRequest) (*PricingResponse, error) {
	log.Printf("Getting pricing for drug %s, quantity %d", req.DrugID, req.Quantity)

	// TODO: Query database for actual pricing
	// For now, return mock pricing data
	pricing := DrugPricing{
		CashPrice:         89.99,
		InsurancePrice:    &[]float64{25.00}[0],
		CopayAmount:       &[]float64{10.00}[0],
		DeductibleApplied: false,
		Tier:              2,
		PriceSource:       "pharmacy_network",
		Currency:          "USD",
		EffectiveDate:     time.Now().AddDate(0, 0, -1),
	}

	response := &PricingResponse{
		DrugID:     req.DrugID,
		PharmacyID: req.PharmacyID,
		Quantity:   req.Quantity,
		Pricing:    pricing,
		RequestID:  req.RequestID,
		Timestamp:  time.Now(),
	}

	return response, nil
}

// ReserveStock creates a temporary stock reservation
func (is *InventoryService) ReserveStock(ctx context.Context, req ReserveRequest) (*ReserveResponse, error) {
	log.Printf("Reserving stock: %d units of drug %s at pharmacy %s", req.Quantity, req.DrugID, req.PharmacyID)

	// TODO: Check actual stock and create reservation in database
	// For now, return mock reservation
	reservationID := fmt.Sprintf("res-%d", time.Now().Unix())
	expirationTime := time.Now().Add(30 * time.Minute) // 30-minute hold
	if req.ExpirationTime != nil {
		expirationTime = *req.ExpirationTime
	}

	response := &ReserveResponse{
		ReservationID:  reservationID,
		DrugID:         req.DrugID,
		PharmacyID:     req.PharmacyID,
		Quantity:       req.Quantity,
		Status:         "reserved",
		ExpirationTime: expirationTime,
		CreatedAt:      time.Now(),
		RequestID:      req.RequestID,
	}

	return response, nil
}

// ReleaseReservation cancels a stock reservation
func (is *InventoryService) ReleaseReservation(ctx context.Context, reservationID string) error {
	log.Printf("Releasing reservation %s", reservationID)

	// TODO: Update database to release reservation
	// For now, just log the operation
	log.Printf("Reservation %s released successfully", reservationID)
	return nil
}

// GetReservationStatus retrieves current status of a reservation
func (is *InventoryService) GetReservationStatus(ctx context.Context, reservationID string) (*ReservationStatus, error) {
	log.Printf("Getting status for reservation %s", reservationID)

	// TODO: Query database for actual reservation
	// For now, return mock status
	status := &ReservationStatus{
		ReservationID:  reservationID,
		DrugID:         "mock-drug-001",
		PharmacyID:     "ph-001",
		Quantity:       30,
		Status:         "active",
		ExpirationTime: time.Now().Add(25 * time.Minute),
		CreatedAt:      time.Now().Add(-5 * time.Minute),
		TimeRemaining:  "25 minutes",
	}

	return status, nil
}

// GetLowStockAlerts retrieves low stock alerts
func (is *InventoryService) GetLowStockAlerts(ctx context.Context, req AlertsRequest) (*AlertsResponse, error) {
	log.Printf("Getting low stock alerts")

	// TODO: Query database for actual alerts
	// For now, return mock alerts
	alerts := []InventoryAlert{
		{
			AlertID:          "alert-001",
			DrugID:           "drug-123",
			DrugName:         "Important Medicine A",
			PharmacyID:       "ph-001",
			PharmacyName:     "Downtown Pharmacy",
			CurrentStock:     5,
			MinimumThreshold: 15,
			SeverityLevel:    "warning",
			DaysUntilEmpty:   3,
			CreatedAt:        time.Now().Add(-2 * time.Hour),
			LastUpdated:      time.Now().Add(-30 * time.Minute),
		},
		{
			AlertID:          "alert-002",
			DrugID:           "drug-456",
			DrugName:         "Critical Medicine B",
			PharmacyID:       "ph-002",
			PharmacyName:     "Suburban Drugs",
			CurrentStock:     0,
			MinimumThreshold: 10,
			SeverityLevel:    "critical",
			DaysUntilEmpty:   0,
			CreatedAt:        time.Now().Add(-4 * time.Hour),
			LastUpdated:      time.Now().Add(-10 * time.Minute),
		},
	}

	response := &AlertsResponse{
		Alerts:     alerts,
		TotalCount: len(alerts),
		RequestID:  req.RequestID,
		Timestamp:  time.Now(),
	}

	return response, nil
}

// HealthCheck performs a health check of the inventory service
func (is *InventoryService) HealthCheck(ctx context.Context) *HealthStatus {
	checks := make(map[string]CheckResult)
	
	// Database health check
	start := time.Now()
	err := is.db.HealthCheck()
	duration := time.Since(start)
	
	if err != nil {
		checks["database"] = CheckResult{
			Status:      "unhealthy",
			Message:     err.Error(),
			LastChecked: time.Now(),
			Duration:    duration.String(),
		}
	} else {
		checks["database"] = CheckResult{
			Status:      "healthy",
			Message:     "Database connection OK",
			LastChecked: time.Now(),
			Duration:    duration.String(),
		}
	}
	
	// Cache health check
	start = time.Now()
	err = is.cache.Ping()
	duration = time.Since(start)
	
	if err != nil {
		checks["cache"] = CheckResult{
			Status:      "unhealthy",
			Message:     err.Error(),
			LastChecked: time.Now(),
			Duration:    duration.String(),
		}
	} else {
		checks["cache"] = CheckResult{
			Status:      "healthy",
			Message:     "Redis connection OK",
			LastChecked: time.Now(),
			Duration:    duration.String(),
		}
	}
	
	// Determine overall status
	status := "healthy"
	for _, check := range checks {
		if check.Status == "unhealthy" {
			status = "unhealthy"
			break
		}
	}
	
	return &HealthStatus{
		Service:   "inventory-service",
		Status:    status,
		Version:   "1.0.0",
		Timestamp: time.Now(),
		Checks:    checks,
		Uptime:    time.Since(startTime).String(),
	}
}