package models

import (
	"time"
	"github.com/google/uuid"
)

// FormularyEntry represents a drug's coverage in an insurance formulary
type FormularyEntry struct {
	ID          uuid.UUID  `json:"id" db:"id"`
	PayerID     string     `json:"payer_id" db:"payer_id"`
	PayerName   string     `json:"payer_name" db:"payer_name"`
	PlanID      string     `json:"plan_id" db:"plan_id"`
	PlanName    string     `json:"plan_name" db:"plan_name"`
	PlanYear    int        `json:"plan_year" db:"plan_year"`
	
	// Drug identification
	DrugRxNorm string `json:"drug_rxnorm" db:"drug_rxnorm"`
	DrugName   string `json:"drug_name" db:"drug_name"`
	DrugType   string `json:"drug_type" db:"drug_type"`
	
	// Coverage details
	Tier   string `json:"tier" db:"tier"`
	Status string `json:"status" db:"status"`
	
	// Cost sharing
	CopayAmount         *float64 `json:"copay_amount" db:"copay_amount"`
	CoinsurancePercent  *int     `json:"coinsurance_percent" db:"coinsurance_percent"`
	DeductibleApplies   bool     `json:"deductible_applies" db:"deductible_applies"`
	
	// Restrictions
	PriorAuthorization bool            `json:"prior_authorization" db:"prior_authorization"`
	StepTherapy        bool            `json:"step_therapy" db:"step_therapy"`
	QuantityLimit      *QuantityLimit  `json:"quantity_limit" db:"quantity_limit"`
	
	// Demographics restrictions
	AgeLimits          *AgeLimits `json:"age_limits" db:"age_limits"`
	GenderRestriction  string     `json:"gender_restriction" db:"gender_restriction"`
	
	// Clinical requirements
	RequiredDiagnosisCodes []string                   `json:"required_diagnosis_codes" db:"required_diagnosis_codes"`
	RequiredLabValues      map[string]interface{}     `json:"required_lab_values" db:"required_lab_values"`
	
	// Alternatives
	PreferredAlternatives []DrugAlternative `json:"preferred_alternatives" db:"preferred_alternatives"`
	GenericAvailable      bool              `json:"generic_available" db:"generic_available"`
	GenericRxNorm         string            `json:"generic_rxnorm" db:"generic_rxnorm"`
	
	// Metadata
	EffectiveDate    time.Time  `json:"effective_date" db:"effective_date"`
	TerminationDate  *time.Time `json:"termination_date" db:"termination_date"`
	LastUpdated      time.Time  `json:"last_updated" db:"last_updated"`
}

// QuantityLimit represents quantity restrictions
type QuantityLimit struct {
	MaxQuantity      int `json:"max_quantity"`
	PerDays          int `json:"per_days"`
	MaxFillsPerYear  int `json:"max_fills_per_year"`
}

// AgeLimits represents age-based restrictions
type AgeLimits struct {
	MinAge int `json:"min_age"`
	MaxAge int `json:"max_age"`
}

// DrugInventory represents stock levels for a drug at a location
type DrugInventory struct {
	ID               uuid.UUID  `json:"id" db:"id"`
	LocationID       string     `json:"location_id" db:"location_id"`
	LocationName     string     `json:"location_name" db:"location_name"`
	DrugRxNorm       string     `json:"drug_rxnorm" db:"drug_rxnorm"`
	DrugNDC          string     `json:"drug_ndc" db:"drug_ndc"`
	
	// Stock levels
	QuantityOnHand   int `json:"quantity_on_hand" db:"quantity_on_hand"`
	QuantityAllocated int `json:"quantity_allocated" db:"quantity_allocated"`
	QuantityAvailable int `json:"quantity_available" db:"quantity_available"`
	
	// Reorder parameters
	ReorderPoint    int `json:"reorder_point" db:"reorder_point"`
	ReorderQuantity int `json:"reorder_quantity" db:"reorder_quantity"`
	MaxStockLevel   int `json:"max_stock_level" db:"max_stock_level"`
	
	// Lot tracking
	LotNumber      string     `json:"lot_number" db:"lot_number"`
	ExpirationDate time.Time  `json:"expiration_date" db:"expiration_date"`
	Manufacturer   string     `json:"manufacturer" db:"manufacturer"`
	
	// Cost information
	UnitCost        float64 `json:"unit_cost" db:"unit_cost"`
	AcquisitionCost float64 `json:"acquisition_cost" db:"acquisition_cost"`
	
	// Timestamps
	LastCounted  *time.Time `json:"last_counted" db:"last_counted"`
	LastOrdered  *time.Time `json:"last_ordered" db:"last_ordered"`
	LastReceived *time.Time `json:"last_received" db:"last_received"`
	UpdatedAt    time.Time  `json:"updated_at" db:"updated_at"`
}

// DrugPricing represents pricing information for drugs
type DrugPricing struct {
	ID              uuid.UUID  `json:"id" db:"id"`
	DrugRxNorm      string     `json:"drug_rxnorm" db:"drug_rxnorm"`
	DrugNDC         string     `json:"drug_ndc" db:"drug_ndc"`
	PriceType       string     `json:"price_type" db:"price_type"`
	Price           float64    `json:"price" db:"price"`
	Unit            string     `json:"unit" db:"unit"`
	PackageSize     int        `json:"package_size" db:"package_size"`
	EffectiveDate   time.Time  `json:"effective_date" db:"effective_date"`
	TerminationDate *time.Time `json:"termination_date" db:"termination_date"`
	Source          string     `json:"source" db:"source"`
	ContractID      string     `json:"contract_id" db:"contract_id"`
}

// DrugAlternative represents therapeutic alternatives
type DrugAlternative struct {
	ID                    uuid.UUID `json:"id" db:"id"`
	PrimaryDrugRxNorm     string    `json:"primary_drug_rxnorm" db:"primary_drug_rxnorm"`
	AlternativeDrugRxNorm string    `json:"alternative_drug_rxnorm" db:"alternative_drug_rxnorm"`
	AlternativeType       string    `json:"alternative_type" db:"alternative_type"`
	TherapeuticClass      string    `json:"therapeutic_class" db:"therapeutic_class"`
	EquivalenceRating     string    `json:"equivalence_rating" db:"equivalence_rating"`
	CostDifferencePercent float64   `json:"cost_difference_percent" db:"cost_difference_percent"`
	EfficacyRating        float64   `json:"efficacy_rating" db:"efficacy_rating"`
	SafetyProfile         string    `json:"safety_profile" db:"safety_profile"`
	SwitchComplexity      string    `json:"switch_complexity" db:"switch_complexity"`
	ClinicalNotes         string    `json:"clinical_notes" db:"clinical_notes"`
	EvidenceLevel         string    `json:"evidence_level" db:"evidence_level"`
	CreatedAt             time.Time `json:"created_at" db:"created_at"`
}

// StockAlert represents inventory alerts
type StockAlert struct {
	ID                uuid.UUID  `json:"id" db:"id"`
	LocationID        string     `json:"location_id" db:"location_id"`
	DrugRxNorm        string     `json:"drug_rxnorm" db:"drug_rxnorm"`
	AlertType         string     `json:"alert_type" db:"alert_type"`
	Severity          string     `json:"severity" db:"severity"`
	Message           string     `json:"message" db:"message"`
	CurrentQuantity   int        `json:"current_quantity" db:"current_quantity"`
	RecommendedAction string     `json:"recommended_action" db:"recommended_action"`
	
	// Alert management
	TriggeredAt     time.Time  `json:"triggered_at" db:"triggered_at"`
	Acknowledged    bool       `json:"acknowledged" db:"acknowledged"`
	AcknowledgedBy  string     `json:"acknowledged_by" db:"acknowledged_by"`
	AcknowledgedAt  *time.Time `json:"acknowledged_at" db:"acknowledged_at"`
	Resolved        bool       `json:"resolved" db:"resolved"`
	ResolvedAt      *time.Time `json:"resolved_at" db:"resolved_at"`
	
	// Escalation
	Escalated       bool       `json:"escalated" db:"escalated"`
	EscalatedAt     *time.Time `json:"escalated_at" db:"escalated_at"`
	EscalationLevel int        `json:"escalation_level" db:"escalation_level"`
}

// Request/Response Models

// FormularyCoverageRequest represents a coverage check request
type FormularyCoverageRequest struct {
	DrugRxNorm string `json:"drug_rxnorm" binding:"required"`
	PayerID    string `json:"payer_id" binding:"required"`
	PlanID     string `json:"plan_id" binding:"required"`
	Quantity   int    `json:"quantity"`
	PatientAge int    `json:"patient_age,omitempty"`
	PatientSex string `json:"patient_sex,omitempty"`
}

// FormularyCoverageResponse represents coverage information
type FormularyCoverageResponse struct {
	Covered              bool                   `json:"covered"`
	Tier                 string                 `json:"tier"`
	PatientCost          float64                `json:"patient_cost"`
	RequiresPriorAuth    bool                   `json:"requires_prior_auth"`
	RequiresStepTherapy  bool                   `json:"requires_step_therapy"`
	QuantityRestrictions *QuantityLimit         `json:"quantity_restrictions,omitempty"`
	Alternatives         []FormularyAlternative `json:"alternatives,omitempty"`
	CostSavings          []CostSavingsOption    `json:"cost_savings,omitempty"`
}

// FormularyAlternative represents an alternative drug option
type FormularyAlternative struct {
	DrugRxNorm            string  `json:"drug_rxnorm"`
	DrugName              string  `json:"drug_name"`
	AlternativeType       string  `json:"alternative_type"`
	Tier                  string  `json:"tier"`
	PatientCost           float64 `json:"patient_cost"`
	CostDifferencePercent float64 `json:"cost_difference_percent"`
	SwitchComplexity      string  `json:"switch_complexity"`
}

// CostSavingsOption represents potential cost savings
type CostSavingsOption struct {
	AlternativeRxNorm string  `json:"alternative_rxnorm"`
	AlternativeName   string  `json:"alternative_name"`
	AlternativeType   string  `json:"alternative_type"`
	PrimaryCost       float64 `json:"primary_cost"`
	AlternativeCost   float64 `json:"alternative_cost"`
	CostSavings       float64 `json:"cost_savings"`
	SavingsPercent    float64 `json:"savings_percent"`
}

// StockStatusRequest represents a stock status query
type StockStatusRequest struct {
	LocationID string   `json:"location_id" binding:"required"`
	DrugRxNorm string   `json:"drug_rxnorm,omitempty"`
	AlertTypes []string `json:"alert_types,omitempty"`
}

// StockStatusResponse represents current stock information
type StockStatusResponse struct {
	LocationID     string              `json:"location_id"`
	TotalItems     int                 `json:"total_items"`
	LowStockItems  int                 `json:"low_stock_items"`
	StockoutItems  int                 `json:"stockout_items"`
	ExpiringItems  int                 `json:"expiring_items"`
	Inventory      []DrugInventory     `json:"inventory,omitempty"`
	Alerts         []StockAlert        `json:"alerts,omitempty"`
	Recommendations []string           `json:"recommendations,omitempty"`
}

// DemandPredictionRequest represents a demand forecast request
type DemandPredictionRequest struct {
	LocationID  string `json:"location_id" binding:"required"`
	DrugRxNorm  string `json:"drug_rxnorm" binding:"required"`
	DaysAhead   int    `json:"days_ahead"`
	IncludeFactors bool `json:"include_factors,omitempty"`
}

// DemandPredictionResponse represents demand forecast results
type DemandPredictionResponse struct {
	LocationID            string                 `json:"location_id"`
	DrugRxNorm            string                 `json:"drug_rxnorm"`
	PredictedDemand       int                    `json:"predicted_demand"`
	ConfidenceIntervalLow int                    `json:"confidence_interval_low"`
	ConfidenceIntervalHigh int                   `json:"confidence_interval_high"`
	ReorderRecommended    bool                   `json:"reorder_recommended"`
	StockoutRisk          float64                `json:"stockout_risk"`
	DemandFactors         map[string]interface{} `json:"demand_factors,omitempty"`
	PredictionTimestamp   time.Time              `json:"prediction_timestamp"`
}

// FormularySearchRequest represents a formulary search request
type FormularySearchRequest struct {
	Query       string   `json:"query"`
	PayerID     string   `json:"payer_id,omitempty"`
	PlanID      string   `json:"plan_id,omitempty"`
	Tiers       []string `json:"tiers,omitempty"`
	DrugTypes   []string `json:"drug_types,omitempty"`
	Limit       int      `json:"limit"`
	Offset      int      `json:"offset"`
}

// FormularySearchResponse represents search results
type FormularySearchResponse struct {
	Results     []FormularyEntry `json:"results"`
	TotalCount  int              `json:"total_count"`
	SearchTime  int64            `json:"search_time_ms"`
	Suggestions []string         `json:"suggestions,omitempty"`
}

// CostAnalysisRequest represents a cost analysis request
type CostAnalysisRequest struct {
	DrugRxNorms []string `json:"drug_rxnorms" binding:"required"`
	PayerID     string   `json:"payer_id"`
	PlanID      string   `json:"plan_id"`
	Quantity    int      `json:"quantity"`
	IncludeAlternatives bool `json:"include_alternatives,omitempty"`
}

// CostAnalysisResponse represents cost analysis results
type CostAnalysisResponse struct {
	TotalPrimaryCost     float64                    `json:"total_primary_cost"`
	TotalAlternativeCost float64                    `json:"total_alternative_cost"`
	TotalSavings         float64                    `json:"total_savings"`
	SavingsPercent       float64                    `json:"savings_percent"`
	DrugAnalysis         []DrugCostAnalysis         `json:"drug_analysis"`
	Recommendations      []CostOptimizationOption   `json:"recommendations,omitempty"`
}

// DrugCostAnalysis represents cost analysis for a single drug
type DrugCostAnalysis struct {
	DrugRxNorm       string              `json:"drug_rxnorm"`
	DrugName         string              `json:"drug_name"`
	PrimaryCost      float64             `json:"primary_cost"`
	BestAlternative  *CostSavingsOption  `json:"best_alternative,omitempty"`
	AllAlternatives  []CostSavingsOption `json:"all_alternatives,omitempty"`
}

// CostOptimizationOption represents a cost optimization recommendation
type CostOptimizationOption struct {
	Type            string                 `json:"type"`
	Description     string                 `json:"description"`
	EstimatedSavings float64               `json:"estimated_savings"`
	Implementation  string                 `json:"implementation"`
	Complexity      string                 `json:"complexity"`
	Details         map[string]interface{} `json:"details,omitempty"`
}