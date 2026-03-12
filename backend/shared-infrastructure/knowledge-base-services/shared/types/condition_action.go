// Package types provides shared type definitions used across multiple packages.
// This package exists to prevent import cycles between packages like rules and extraction.
package types

import (
	"fmt"

	"github.com/google/uuid"
)

// =============================================================================
// FINGERPRINT INTERFACE - Breaks import cycles between rules and fingerprint_registry
// =============================================================================

// FingerprintableRule is the interface for rules that can be registered in the fingerprint registry.
// This interface breaks the import cycle between rules and fingerprint_registry packages.
// Any type that implements these methods can be registered for semantic deduplication.
type FingerprintableRule interface {
	GetRuleID() uuid.UUID
	GetDomain() string
	GetRuleType() string
	GetFingerprintHash() string
	GetFingerprintVersion() int
}

// =============================================================================
// CONDITION - THE "IF" PART
// =============================================================================

// Condition represents the IF part of a clinical rule
// Schema: {variable, operator, value, unit}
type Condition struct {
	Variable    string   `json:"variable"`               // renal_function.crcl, hepatic.child_pugh, patient.age
	Operator    Operator `json:"operator"`               // <, >, <=, >=, BETWEEN, ==, IN
	Value       *float64 `json:"value,omitempty"`        // Single numeric value
	MinValue    *float64 `json:"min_value,omitempty"`    // For BETWEEN operator
	MaxValue    *float64 `json:"max_value,omitempty"`    // For BETWEEN operator
	StringValue *string  `json:"string_value,omitempty"` // For categorical: "A", "B", "C"
	ListValues  []string `json:"list_values,omitempty"`  // For IN operator
	Unit        string   `json:"unit"`                   // ml/min, mg, percent
}

// Operator defines the comparison type
type Operator string

const (
	OpLessThan       Operator = "<"
	OpGreaterThan    Operator = ">"
	OpLessOrEqual    Operator = "<="
	OpGreaterOrEqual Operator = ">="
	OpBetween        Operator = "BETWEEN"
	OpEquals         Operator = "=="
	OpNotEquals      Operator = "!="
	OpIn             Operator = "IN"
)

// Evaluate checks if a given value satisfies the condition
func (c *Condition) Evaluate(numericValue *float64, stringValue *string) bool {
	switch c.Operator {
	case OpLessThan:
		return numericValue != nil && c.Value != nil && *numericValue < *c.Value
	case OpGreaterThan:
		return numericValue != nil && c.Value != nil && *numericValue > *c.Value
	case OpLessOrEqual:
		return numericValue != nil && c.Value != nil && *numericValue <= *c.Value
	case OpGreaterOrEqual:
		return numericValue != nil && c.Value != nil && *numericValue >= *c.Value
	case OpBetween:
		return numericValue != nil && c.MinValue != nil && c.MaxValue != nil &&
			*numericValue >= *c.MinValue && *numericValue < *c.MaxValue
	case OpEquals:
		if numericValue != nil && c.Value != nil {
			return *numericValue == *c.Value
		}
		if stringValue != nil && c.StringValue != nil {
			return *stringValue == *c.StringValue
		}
		return false
	case OpNotEquals:
		if numericValue != nil && c.Value != nil {
			return *numericValue != *c.Value
		}
		if stringValue != nil && c.StringValue != nil {
			return *stringValue != *c.StringValue
		}
		return false
	case OpIn:
		if stringValue != nil {
			for _, v := range c.ListValues {
				if *stringValue == v {
					return true
				}
			}
		}
		return false
	}
	return false
}

// String returns a human-readable representation of the condition
func (c *Condition) String() string {
	switch c.Operator {
	case OpBetween:
		return fmt.Sprintf("%s %s %v-%v %s", c.Variable, c.Operator, *c.MinValue, *c.MaxValue, c.Unit)
	case OpEquals, OpNotEquals:
		if c.StringValue != nil {
			return fmt.Sprintf("%s %s %s", c.Variable, c.Operator, *c.StringValue)
		}
		return fmt.Sprintf("%s %s %v %s", c.Variable, c.Operator, *c.Value, c.Unit)
	case OpIn:
		return fmt.Sprintf("%s %s [%v]", c.Variable, c.Operator, c.ListValues)
	default:
		return fmt.Sprintf("%s %s %v %s", c.Variable, c.Operator, *c.Value, c.Unit)
	}
}

// =============================================================================
// ACTION - THE "THEN" PART
// =============================================================================

// Action represents the THEN part of a clinical rule
// Schema: {effect, adjustment, message}
type Action struct {
	Effect     Effect          `json:"effect"`               // CONTRAINDICATED, DOSE_ADJUST, AVOID, MONITOR
	Adjustment *DoseAdjustment `json:"adjustment,omitempty"` // Specific dose modification
	Message    string          `json:"message,omitempty"`    // Human-readable recommendation
	Severity   Severity        `json:"severity,omitempty"`   // CRITICAL, HIGH, MODERATE, LOW
	AlertCode  string          `json:"alert_code,omitempty"` // For CDS alert systems
}

// Effect defines the clinical action type
type Effect string

const (
	EffectContraindicated Effect = "CONTRAINDICATED"
	EffectDoseAdjust      Effect = "DOSE_ADJUST"
	EffectAvoid           Effect = "AVOID"
	EffectMonitor         Effect = "MONITOR"
	EffectUseWithCaution  Effect = "USE_WITH_CAUTION"
	EffectNoChange        Effect = "NO_CHANGE"
	EffectHold            Effect = "HOLD"
	EffectDiscontinue     Effect = "DISCONTINUE"
)

// Severity indicates the clinical importance of the action
type Severity string

const (
	SeverityCritical Severity = "CRITICAL"
	SeverityHigh     Severity = "HIGH"
	SeverityModerate Severity = "MODERATE"
	SeverityLow      Severity = "LOW"
	SeverityInfo     Severity = "INFO"
)

// DoseAdjustment contains specific dosing modifications
type DoseAdjustment struct {
	Type         AdjustmentType `json:"type"`                    // PERCENTAGE, ABSOLUTE, INTERVAL, MAX_DOSE
	Percentage   *float64       `json:"percentage,omitempty"`    // 50 = 50% of normal dose
	AbsoluteDose *string        `json:"absolute_dose,omitempty"` // "250mg BID"
	MaxDose      *string        `json:"max_dose,omitempty"`      // "500mg daily"
	Interval     *string        `json:"interval,omitempty"`      // "Every 48 hours"
	Frequency    *string        `json:"frequency,omitempty"`     // "BID", "TID", "daily"
	Duration     *string        `json:"duration,omitempty"`      // "5 days", "until resolved"
}

// AdjustmentType defines how the dose is modified
type AdjustmentType string

const (
	AdjustmentPercentage AdjustmentType = "PERCENTAGE"
	AdjustmentAbsolute   AdjustmentType = "ABSOLUTE"
	AdjustmentInterval   AdjustmentType = "INTERVAL"
	AdjustmentMaxDose    AdjustmentType = "MAX_DOSE"
	AdjustmentFrequency  AdjustmentType = "FREQUENCY"
)
