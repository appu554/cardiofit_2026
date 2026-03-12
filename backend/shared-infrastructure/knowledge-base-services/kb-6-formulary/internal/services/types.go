package services

import "time"

// Additional request/response types for HTTP API endpoints
// These complement the existing gRPC-focused types in the main service files

// HTTPSearchRequest represents a search request from HTTP API
type HTTPSearchRequest struct {
	Query     string `json:"query"`
	PayerID   string `json:"payer_id,omitempty"`
	Limit     int    `json:"limit,omitempty"`
	Offset    int    `json:"offset,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

// HTTPCoverageRequest represents a coverage request from HTTP API
type HTTPCoverageRequest struct {
	DrugID      string `json:"drug_id"`
	PayerID     string `json:"payer_id"`
	MemberID    string `json:"member_id,omitempty"`
	FormularyID string `json:"formulary_id,omitempty"`
	RequestID   string `json:"request_id,omitempty"`
}

// AlternativesRequest represents a request for drug alternatives
type AlternativesRequest struct {
	DrugID           string `json:"drug_id"`
	PayerID          string `json:"payer_id"`
	TherapeuticClass string `json:"therapeutic_class,omitempty"`
	MaxResults       int    `json:"max_results,omitempty"`
	RequestID        string `json:"request_id,omitempty"`
}

// AlternativesResponse represents alternative drug options
type AlternativesResponse struct {
	DrugID         string               `json:"drug_id"`
	PayerID        string               `json:"payer_id"`
	Alternatives   []DrugAlternative    `json:"alternatives"`
	RequestID      string               `json:"request_id"`
	Timestamp      time.Time            `json:"timestamp"`
	DatasetVersion string               `json:"dataset_version"`
}

// DrugAlternative represents a single alternative drug
type DrugAlternative struct {
	DrugID             string  `json:"drug_id"`
	DrugName           string  `json:"drug_name"`
	GenericName        string  `json:"generic_name"`
	Tier               int     `json:"tier"`
	CoverageStatus     string  `json:"coverage_status"`
	EstimatedCostDiff  float64 `json:"estimated_cost_difference"`
	PriorAuthRequired  bool    `json:"prior_auth_required"`
	StepTherapyReq     bool    `json:"step_therapy_required"`
	QuantityLimits     *string `json:"quantity_limits,omitempty"`
}

// StockRequest represents a stock availability request
type StockRequest struct {
	DrugID              string `json:"drug_id"`
	PharmacyID          string `json:"pharmacy_id,omitempty"`
	NetworkID           string `json:"network_id,omitempty"`
	ZipCode             string `json:"zip_code,omitempty"`
	IncludeAlternatives bool   `json:"include_alternatives,omitempty"`
	RequestID           string `json:"request_id,omitempty"`
}

// StockInfoResponse represents stock availability information for HTTP API
type StockInfoResponse struct {
	DrugID         string              `json:"drug_id"`
	StockInfo      []PharmacyStock     `json:"stock_info"`
	Alternatives   []DrugAlternative   `json:"alternatives,omitempty"`
	RequestID      string              `json:"request_id"`
	Timestamp      time.Time           `json:"timestamp"`
}

// PharmacyStock represents stock information for a specific pharmacy
type PharmacyStock struct {
	PharmacyID       string    `json:"pharmacy_id"`
	PharmacyName     string    `json:"pharmacy_name"`
	Address          string    `json:"address"`
	Phone            string    `json:"phone"`
	DistanceMiles    float64   `json:"distance_miles"`
	InStock          bool      `json:"in_stock"`
	QuantityOnHand   int       `json:"quantity_on_hand"`
	LastUpdated      time.Time `json:"last_updated"`
	ExpectedRestock  *string   `json:"expected_restock,omitempty"`
}

// AvailabilityRequest represents a drug availability search request
type AvailabilityRequest struct {
	DrugID      string  `json:"drug_id"`
	Location    string  `json:"location"`          // zip code or city, state
	RadiusMiles float64 `json:"radius_miles"`
	SortBy      string  `json:"sort_by"`           // distance, price, availability
	Limit       int     `json:"limit"`
	RequestID   string  `json:"request_id,omitempty"`
}

// AvailabilityResponse represents drug availability results
type AvailabilityResponse struct {
	DrugID      string           `json:"drug_id"`
	Location    string           `json:"search_location"`
	Pharmacies  []PharmacyStock  `json:"pharmacies"`
	TotalFound  int              `json:"total_found"`
	RequestID   string           `json:"request_id"`
	Timestamp   time.Time        `json:"timestamp"`
}

// PricingRequest represents a drug pricing request
type PricingRequest struct {
	DrugID     string `json:"drug_id"`
	PharmacyID string `json:"pharmacy_id,omitempty"`
	Quantity   int    `json:"quantity"`
	PayerID    string `json:"payer_id,omitempty"`
	MemberID   string `json:"member_id,omitempty"`
	RequestID  string `json:"request_id,omitempty"`
}

// PricingResponse represents drug pricing information
type PricingResponse struct {
	DrugID        string        `json:"drug_id"`
	PharmacyID    string        `json:"pharmacy_id,omitempty"`
	Quantity      int           `json:"quantity"`
	Pricing       DrugPricing   `json:"pricing"`
	RequestID     string        `json:"request_id"`
	Timestamp     time.Time     `json:"timestamp"`
}

// DrugPricing represents detailed pricing information
type DrugPricing struct {
	CashPrice      float64  `json:"cash_price"`
	InsurancePrice *float64 `json:"insurance_price,omitempty"`
	CopayAmount    *float64 `json:"copay_amount,omitempty"`
	DeductibleApplied bool  `json:"deductible_applied"`
	Tier           int      `json:"tier"`
	PriceSource    string   `json:"price_source"`
	Currency       string   `json:"currency"`
	EffectiveDate  time.Time `json:"effective_date"`
}

// ReserveRequest represents a stock reservation request
type ReserveRequest struct {
	DrugID       string    `json:"drug_id"`
	PharmacyID   string    `json:"pharmacy_id"`
	Quantity     int       `json:"quantity"`
	CustomerID   string    `json:"customer_id,omitempty"`
	ExpirationTime *time.Time `json:"expiration_time,omitempty"`
	RequestID    string    `json:"request_id,omitempty"`
}

// ReserveResponse represents a stock reservation response
type ReserveResponse struct {
	ReservationID   string    `json:"reservation_id"`
	DrugID          string    `json:"drug_id"`
	PharmacyID      string    `json:"pharmacy_id"`
	Quantity        int       `json:"quantity"`
	Status          string    `json:"status"`          // reserved, expired, cancelled
	ExpirationTime  time.Time `json:"expiration_time"`
	CreatedAt       time.Time `json:"created_at"`
	RequestID       string    `json:"request_id"`
}

// ReservationStatus represents current status of a reservation
type ReservationStatus struct {
	ReservationID   string    `json:"reservation_id"`
	DrugID          string    `json:"drug_id"`
	PharmacyID      string    `json:"pharmacy_id"`
	Quantity        int       `json:"quantity"`
	Status          string    `json:"status"`
	ExpirationTime  time.Time `json:"expiration_time"`
	CreatedAt       time.Time `json:"created_at"`
	TimeRemaining   string    `json:"time_remaining"`
}

// AlertsRequest represents a low stock alerts request
type AlertsRequest struct {
	PharmacyID    string `json:"pharmacy_id,omitempty"`
	NetworkID     string `json:"network_id,omitempty"`
	SeverityLevel string `json:"severity_level,omitempty"` // critical, warning, info
	Limit         int    `json:"limit"`
	RequestID     string `json:"request_id,omitempty"`
}

// AlertsResponse represents low stock alerts
type AlertsResponse struct {
	Alerts      []InventoryAlert  `json:"alerts"`
	TotalCount  int               `json:"total_count"`
	RequestID   string            `json:"request_id"`
	Timestamp   time.Time         `json:"timestamp"`
}

// InventoryAlert represents a single stock alert for HTTP API
type InventoryAlert struct {
	AlertID          string    `json:"alert_id"`
	DrugID           string    `json:"drug_id"`
	DrugName         string    `json:"drug_name"`
	PharmacyID       string    `json:"pharmacy_id"`
	PharmacyName     string    `json:"pharmacy_name"`
	CurrentStock     int       `json:"current_stock"`
	MinimumThreshold int       `json:"minimum_threshold"`
	SeverityLevel    string    `json:"severity_level"`
	DaysUntilEmpty   int       `json:"days_until_empty"`
	CreatedAt        time.Time `json:"created_at"`
	LastUpdated      time.Time `json:"last_updated"`
}

// HealthStatus represents service health information
type HealthStatus struct {
	Service     string                 `json:"service"`
	Status      string                 `json:"status"`      // healthy, degraded, unhealthy
	Version     string                 `json:"version"`
	Timestamp   time.Time              `json:"timestamp"`
	Checks      map[string]CheckResult `json:"checks,omitempty"`
	Uptime      string                 `json:"uptime"`
}

// CheckResult represents individual health check result
type CheckResult struct {
	Status      string    `json:"status"`
	Message     string    `json:"message,omitempty"`
	LastChecked time.Time `json:"last_checked"`
	Duration    string    `json:"duration"`
}

// FormularyInfo represents formulary metadata
type FormularyInfo struct {
	FormularyID    string    `json:"formulary_id"`
	Name           string    `json:"name"`
	PayerID        string    `json:"payer_id"`
	PayerName      string    `json:"payer_name"`
	EffectiveDate  time.Time `json:"effective_date"`
	ExpirationDate time.Time `json:"expiration_date"`
	DrugCount      int       `json:"drug_count"`
	LastUpdated    time.Time `json:"last_updated"`
	Version        string    `json:"version"`
	Description    string    `json:"description,omitempty"`
}