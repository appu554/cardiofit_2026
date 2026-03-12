package models

import "time"

// =============================================================================
// QUANTITY LIMIT TYPES
// =============================================================================

// QLViolationType defines the type of quantity limit violation
type QLViolationType string

const (
	QLViolationMaxQuantity    QLViolationType = "MAX_QUANTITY"
	QLViolationDaysSupply     QLViolationType = "DAYS_SUPPLY"
	QLViolationFillsPerYear   QLViolationType = "FILLS_PER_YEAR"
	QLViolationDailyDose      QLViolationType = "DAILY_DOSE"
)

// =============================================================================
// QUANTITY LIMIT MODELS
// =============================================================================

// ExtendedQuantityLimit extends the base QuantityLimit with additional fields
type ExtendedQuantityLimit struct {
	MaxQuantity      int      `json:"max_quantity"`
	PerDays          int      `json:"per_days"`
	MaxFillsPerYear  int      `json:"max_fills_per_year,omitempty"`
	MaxDailyDoseMg   *float64 `json:"max_daily_dose_mg,omitempty"`
	MinDaysSupply    *int     `json:"min_days_supply,omitempty"`
	MaxDaysSupply    *int     `json:"max_days_supply,omitempty"`
}

// QLViolation represents a single quantity limit violation
type QLViolation struct {
	Type        QLViolationType `json:"type"`
	Limit       interface{}     `json:"limit"`      // The limit value
	Requested   interface{}     `json:"requested"`  // The requested value
	Message     string          `json:"message"`
	Severity    string          `json:"severity"`   // warning, error
}

// QLOverride represents an approved quantity limit override
type QLOverride struct {
	ApprovedQuantity    int        `json:"approved_quantity,omitempty"`
	ApprovedDaysSupply  int        `json:"approved_days_supply,omitempty"`
	ApprovedFillsYear   int        `json:"approved_fills_year,omitempty"`
	OverrideReason      string     `json:"override_reason"`
	ApprovedBy          string     `json:"approved_by"`
	ApprovedAt          time.Time  `json:"approved_at"`
	ExpiresAt           *time.Time `json:"expires_at,omitempty"`
}

// =============================================================================
// REQUEST/RESPONSE MODELS
// =============================================================================

// QLCheckRequest represents a quantity limit check request
type QLCheckRequest struct {
	DrugRxNorm     string  `json:"drug_rxnorm" binding:"required"`
	Quantity       int     `json:"quantity" binding:"required"`
	DaysSupply     int     `json:"days_supply" binding:"required"`
	FillsThisYear  int     `json:"fills_this_year,omitempty"`
	DailyDoseMg    float64 `json:"daily_dose_mg,omitempty"`
	PayerID        *string `json:"payer_id,omitempty"`
	PlanID         *string `json:"plan_id,omitempty"`
	PatientID      *string `json:"patient_id,omitempty"`
}

// QLCheckResponse represents the result of a quantity limit check
type QLCheckResponse struct {
	DrugRxNorm       string                `json:"drug_rxnorm"`
	DrugName         string                `json:"drug_name"`
	RequestedQty     int                   `json:"requested_quantity"`
	RequestedDays    int                   `json:"requested_days_supply"`
	Limits           *ExtendedQuantityLimit `json:"limits"`
	WithinLimits     bool                  `json:"within_limits"`
	Violations       []QLViolation         `json:"violations,omitempty"`
	OverrideAllowed  bool                  `json:"override_allowed"`
	SuggestedQty     *int                  `json:"suggested_quantity,omitempty"`
	SuggestedDays    *int                  `json:"suggested_days_supply,omitempty"`
	ExistingOverride *QLOverride           `json:"existing_override,omitempty"`
	Message          string                `json:"message"`

	// Enhancement #1: Policy Binding (Tier-7 Governance Integration)
	PolicyBinding    *PolicyBinding        `json:"policy_binding,omitempty"`
}

// QLOverrideRequest represents a quantity limit override request
type QLOverrideRequest struct {
	DrugRxNorm         string `json:"drug_rxnorm" binding:"required"`
	PatientID          string `json:"patient_id" binding:"required"`
	ProviderID         string `json:"provider_id" binding:"required"`
	RequestedQuantity  int    `json:"requested_quantity"`
	RequestedDaysSupply int   `json:"requested_days_supply"`
	OverrideReason     string `json:"override_reason" binding:"required"`
	ClinicalNotes      string `json:"clinical_notes,omitempty"`
	PayerID            *string `json:"payer_id,omitempty"`
	PlanID             *string `json:"plan_id,omitempty"`
}

// QLOverrideResponse represents a quantity limit override response
type QLOverrideResponse struct {
	Approved     bool       `json:"approved"`
	Override     *QLOverride `json:"override,omitempty"`
	Message      string     `json:"message"`
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

// ValidateQuantity checks if requested quantity is within limits
func ValidateQuantity(requested int, limit int) bool {
	return requested <= limit
}

// ValidateDaysSupply checks if requested days supply is within limits
func ValidateDaysSupply(requested int, min, max *int) bool {
	if min != nil && requested < *min {
		return false
	}
	if max != nil && requested > *max {
		return false
	}
	return true
}

// ValidateFillsPerYear checks if fills this year is within annual limit
func ValidateFillsPerYear(current int, limit int) bool {
	return current < limit
}

// CalculateSuggestedQuantity returns a compliant quantity
func CalculateSuggestedQuantity(requested int, limits *ExtendedQuantityLimit) int {
	if limits == nil {
		return requested
	}

	if requested <= limits.MaxQuantity {
		return requested
	}

	return limits.MaxQuantity
}

// CheckQuantityLimits validates a request against quantity limits
func CheckQuantityLimits(req QLCheckRequest, limits *ExtendedQuantityLimit) []QLViolation {
	var violations []QLViolation

	if limits == nil {
		return violations
	}

	// Check max quantity per fill
	if limits.MaxQuantity > 0 && req.Quantity > limits.MaxQuantity {
		violations = append(violations, QLViolation{
			Type:      QLViolationMaxQuantity,
			Limit:     limits.MaxQuantity,
			Requested: req.Quantity,
			Message:   "Requested quantity exceeds maximum quantity per fill",
			Severity:  "error",
		})
	}

	// Check days supply
	if limits.MaxDaysSupply != nil && req.DaysSupply > *limits.MaxDaysSupply {
		violations = append(violations, QLViolation{
			Type:      QLViolationDaysSupply,
			Limit:     *limits.MaxDaysSupply,
			Requested: req.DaysSupply,
			Message:   "Requested days supply exceeds maximum days supply",
			Severity:  "error",
		})
	}

	// Check fills per year
	if limits.MaxFillsPerYear > 0 && req.FillsThisYear >= limits.MaxFillsPerYear {
		violations = append(violations, QLViolation{
			Type:      QLViolationFillsPerYear,
			Limit:     limits.MaxFillsPerYear,
			Requested: req.FillsThisYear + 1,
			Message:   "Annual fill limit reached",
			Severity:  "error",
		})
	}

	// Check daily dose if applicable
	if limits.MaxDailyDoseMg != nil && req.DailyDoseMg > *limits.MaxDailyDoseMg {
		violations = append(violations, QLViolation{
			Type:      QLViolationDailyDose,
			Limit:     *limits.MaxDailyDoseMg,
			Requested: req.DailyDoseMg,
			Message:   "Requested daily dose exceeds maximum",
			Severity:  "warning",
		})
	}

	return violations
}
