// Package governance provides Tier-7 governance event emission for KB-16
// This package enables clinical accountability by publishing events that require
// human oversight and creates an immutable audit trail for SaMD compliance.
package governance

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// =============================================================================
// EVENT TYPES
// =============================================================================

// EventType represents the category of governance event
type EventType string

const (
	// Critical lab value events
	EventCriticalLabValue     EventType = "CRITICAL_LAB_VALUE"
	EventPanicLabValue        EventType = "PANIC_LAB_VALUE"
	EventSignificantDeltaLab  EventType = "SIGNIFICANT_DELTA_LAB"

	// Pattern detection events
	EventClinicalPattern      EventType = "CLINICAL_PATTERN_DETECTED"
	EventAKIDetected          EventType = "AKI_DETECTED"
	EventSepsisIndicator      EventType = "SEPSIS_INDICATOR"

	// Care gap events
	EventCareGapIdentified    EventType = "CARE_GAP_IDENTIFIED"
	EventOverdueMonitoring    EventType = "OVERDUE_MONITORING"

	// Trending events
	EventWorseningTrend       EventType = "WORSENING_TREND"
	EventVolatileTrend        EventType = "VOLATILE_TREND"
	EventBaselineDeviation    EventType = "BASELINE_DEVIATION"

	// Review workflow events
	EventReviewRequired       EventType = "REVIEW_REQUIRED"
	EventReviewOverdue        EventType = "REVIEW_OVERDUE"
	EventSLABreach            EventType = "SLA_BREACH"

	// Acknowledgment events
	EventCriticalAcknowledged EventType = "CRITICAL_ACKNOWLEDGED"
	EventReviewCompleted      EventType = "REVIEW_COMPLETED"
	EventActionTaken          EventType = "ACTION_TAKEN"
)

// Severity represents the urgency level of the event
type Severity string

const (
	SeverityInfo     Severity = "INFO"
	SeverityLow      Severity = "LOW"
	SeverityMedium   Severity = "MEDIUM"
	SeverityHigh     Severity = "HIGH"
	SeverityCritical Severity = "CRITICAL"
)

// =============================================================================
// GOVERNANCE EVENT
// =============================================================================

// GovernanceEvent represents a clinical event requiring governance oversight
type GovernanceEvent struct {
	// Event identification
	ID        uuid.UUID `json:"id"`
	EventType EventType `json:"event_type"`
	Source    string    `json:"source"` // KB-16
	Version   string    `json:"version"`

	// Timing
	Timestamp   time.Time  `json:"timestamp"`
	DetectedAt  time.Time  `json:"detected_at"`
	ExpiresAt   *time.Time `json:"expires_at,omitempty"`

	// Clinical context
	PatientID   string   `json:"patient_id"`
	EncounterID string   `json:"encounter_id,omitempty"`
	Severity    Severity `json:"severity"`
	Priority    int      `json:"priority"` // 1 = highest

	// Event details
	Title       string                 `json:"title"`
	Description string                 `json:"description"`
	Payload     map[string]interface{} `json:"payload"`

	// Provenance
	Provenance EventProvenance `json:"provenance"`

	// Accountability requirements
	RequiresAcknowledgment bool   `json:"requires_acknowledgment"`
	AcknowledgmentSLAMin   int    `json:"acknowledgment_sla_minutes,omitempty"`
	RequiresReview         bool   `json:"requires_review"`
	ReviewSLAMin           int    `json:"review_sla_minutes,omitempty"`
	EscalationPath         string `json:"escalation_path,omitempty"`

	// Linked resources
	ResultID     string `json:"result_id,omitempty"`
	KB14TaskID   string `json:"kb14_task_id,omitempty"`
	KB9GapID     string `json:"kb9_gap_id,omitempty"`
	RelatedEvents []string `json:"related_events,omitempty"`

	// Status tracking
	Status        EventStatus `json:"status"`
	StatusHistory []StatusChange `json:"status_history,omitempty"`
}

// EventProvenance tracks the origin and calculation sources of the event
type EventProvenance struct {
	KB8Calculations []KB8Calculation `json:"kb8_calculations,omitempty"`
	ReferenceRanges []ReferenceUsed  `json:"reference_ranges,omitempty"`
	RulesApplied    []RuleApplied    `json:"rules_applied,omitempty"`
	InterpretationVersion string     `json:"interpretation_version"`
	Timestamp       time.Time        `json:"timestamp"`
}

// KB8Calculation records a calculation performed by KB-8
type KB8Calculation struct {
	Calculator string                 `json:"calculator"` // egfr, anion_gap, etc.
	Input      map[string]interface{} `json:"input"`
	Output     map[string]interface{} `json:"output"`
	Formula    string                 `json:"formula,omitempty"`
	Version    string                 `json:"version,omitempty"`
	Timestamp  time.Time              `json:"timestamp"`
}

// ReferenceUsed records which reference ranges were applied
type ReferenceUsed struct {
	Code        string   `json:"code"`
	Source      string   `json:"source"` // CAP, CLSI, etc.
	Version     string   `json:"version,omitempty"`
	Low         *float64 `json:"low,omitempty"`
	High        *float64 `json:"high,omitempty"`
	CriticalLow *float64 `json:"critical_low,omitempty"`
	CriticalHigh *float64 `json:"critical_high,omitempty"`
	AgeAdjusted bool     `json:"age_adjusted"`
	SexAdjusted bool     `json:"sex_adjusted"`
}

// RuleApplied records which interpretation rules were triggered
type RuleApplied struct {
	RuleID      string `json:"rule_id"`
	RuleName    string `json:"rule_name"`
	Version     string `json:"version,omitempty"`
	Triggered   bool   `json:"triggered"`
	Confidence  float64 `json:"confidence,omitempty"`
}

// EventStatus represents the current state of a governance event
type EventStatus string

const (
	StatusPending      EventStatus = "PENDING"
	StatusAcknowledged EventStatus = "ACKNOWLEDGED"
	StatusInProgress   EventStatus = "IN_PROGRESS"
	StatusResolved     EventStatus = "RESOLVED"
	StatusEscalated    EventStatus = "ESCALATED"
	StatusExpired      EventStatus = "EXPIRED"
)

// StatusChange records a change in event status
type StatusChange struct {
	From      EventStatus `json:"from"`
	To        EventStatus `json:"to"`
	ChangedAt time.Time   `json:"changed_at"`
	ChangedBy string      `json:"changed_by,omitempty"`
	Reason    string      `json:"reason,omitempty"`
}

// =============================================================================
// EVENT BUILDERS
// =============================================================================

// NewGovernanceEvent creates a new governance event with defaults
func NewGovernanceEvent(eventType EventType, patientID string) *GovernanceEvent {
	now := time.Now().UTC()
	return &GovernanceEvent{
		ID:          uuid.New(),
		EventType:   eventType,
		Source:      "KB-16",
		Version:     "1.0.0",
		Timestamp:   now,
		DetectedAt:  now,
		PatientID:   patientID,
		Status:      StatusPending,
		Payload:     make(map[string]interface{}),
		Provenance: EventProvenance{
			InterpretationVersion: "1.0.0",
			Timestamp:            now,
		},
	}
}

// CriticalLabEvent creates an event for critical lab values
func CriticalLabEvent(patientID, resultID, labCode, labName string, value float64, unit string, flag string) *GovernanceEvent {
	event := NewGovernanceEvent(EventCriticalLabValue, patientID)
	event.Severity = SeverityCritical
	event.Priority = 1
	event.ResultID = resultID
	event.Title = "Critical Lab Value: " + labName
	event.Description = formatCriticalDescription(labName, value, unit, flag)
	event.RequiresAcknowledgment = true
	event.AcknowledgmentSLAMin = 30
	event.RequiresReview = true
	event.ReviewSLAMin = 60
	event.EscalationPath = "attending_physician"

	event.Payload = map[string]interface{}{
		"lab_code":   labCode,
		"lab_name":   labName,
		"value":      value,
		"unit":       unit,
		"flag":       flag,
		"result_id":  resultID,
	}

	return event
}

// PanicLabEvent creates an event for panic lab values
func PanicLabEvent(patientID, resultID, labCode, labName string, value float64, unit string, flag string) *GovernanceEvent {
	event := NewGovernanceEvent(EventPanicLabValue, patientID)
	event.Severity = SeverityCritical
	event.Priority = 1
	event.ResultID = resultID
	event.Title = "PANIC: " + labName
	event.Description = formatPanicDescription(labName, value, unit, flag)
	event.RequiresAcknowledgment = true
	event.AcknowledgmentSLAMin = 15 // Tighter SLA for panic
	event.RequiresReview = true
	event.ReviewSLAMin = 30
	event.EscalationPath = "rapid_response"

	event.Payload = map[string]interface{}{
		"lab_code":       labCode,
		"lab_name":       labName,
		"value":          value,
		"unit":           unit,
		"flag":           flag,
		"result_id":      resultID,
		"immediate_action_required": true,
	}

	return event
}

// SignificantDeltaEvent creates an event for significant lab value changes
func SignificantDeltaEvent(patientID, resultID, labCode, labName string, currentValue, previousValue, percentChange float64, unit string) *GovernanceEvent {
	event := NewGovernanceEvent(EventSignificantDeltaLab, patientID)
	event.Severity = SeverityHigh
	event.Priority = 2
	event.ResultID = resultID
	event.Title = "Significant Change: " + labName
	event.Description = formatDeltaDescription(labName, currentValue, previousValue, percentChange, unit)
	event.RequiresAcknowledgment = true
	event.AcknowledgmentSLAMin = 60
	event.RequiresReview = true
	event.ReviewSLAMin = 120

	direction := "increased"
	if currentValue < previousValue {
		direction = "decreased"
	}

	event.Payload = map[string]interface{}{
		"lab_code":       labCode,
		"lab_name":       labName,
		"current_value":  currentValue,
		"previous_value": previousValue,
		"percent_change": percentChange,
		"direction":      direction,
		"unit":           unit,
		"result_id":      resultID,
	}

	return event
}

// ClinicalPatternEvent creates an event for detected clinical patterns
func ClinicalPatternEvent(patientID string, patternCode, patternName string, confidence float64, severity Severity) *GovernanceEvent {
	event := NewGovernanceEvent(EventClinicalPattern, patientID)
	event.Severity = severity
	event.Priority = severityToPriority(severity)
	event.Title = "Pattern Detected: " + patternName
	event.Description = formatPatternDescription(patternName, confidence)
	event.RequiresReview = true
	event.ReviewSLAMin = 120

	if severity == SeverityCritical {
		event.RequiresAcknowledgment = true
		event.AcknowledgmentSLAMin = 30
	}

	event.Payload = map[string]interface{}{
		"pattern_code": patternCode,
		"pattern_name": patternName,
		"confidence":   confidence,
	}

	return event
}

// WorseningTrendEvent creates an event for worsening lab trends
func WorseningTrendEvent(patientID, labCode, labName string, trajectory string, rateOfChange float64, unit string) *GovernanceEvent {
	event := NewGovernanceEvent(EventWorseningTrend, patientID)
	event.Severity = SeverityMedium
	event.Priority = 3
	event.Title = "Worsening Trend: " + labName
	event.Description = formatTrendDescription(labName, trajectory, rateOfChange, unit)
	event.RequiresReview = true
	event.ReviewSLAMin = 240

	event.Payload = map[string]interface{}{
		"lab_code":       labCode,
		"lab_name":       labName,
		"trajectory":     trajectory,
		"rate_of_change": rateOfChange,
		"unit":           unit,
	}

	return event
}

// CareGapEvent creates an event for identified care gaps
func CareGapEvent(patientID, gapCode, gapName string, daysOverdue int) *GovernanceEvent {
	event := NewGovernanceEvent(EventCareGapIdentified, patientID)
	event.Severity = SeverityMedium
	if daysOverdue > 90 {
		event.Severity = SeverityHigh
	}
	event.Priority = severityToPriority(event.Severity)
	event.Title = "Care Gap: " + gapName
	event.Description = formatCareGapDescription(gapName, daysOverdue)
	event.RequiresReview = true
	event.ReviewSLAMin = 480 // 8 hours for care gaps

	event.Payload = map[string]interface{}{
		"gap_code":     gapCode,
		"gap_name":     gapName,
		"days_overdue": daysOverdue,
	}

	return event
}

// =============================================================================
// HELPER FUNCTIONS
// =============================================================================

func formatCriticalDescription(name string, value float64, unit, flag string) string {
	return fmt.Sprintf(
		"%s is at CRITICAL level: %.2f %s (%s). Immediate clinical review required.",
		name, value, unit, flag,
	)
}

func formatPanicDescription(name string, value float64, unit, flag string) string {
	return fmt.Sprintf(
		"PANIC VALUE: %s is %.2f %s (%s). IMMEDIATE CLINICAL ACTION REQUIRED. "+
		"This value represents an imminent life-threatening condition.",
		name, value, unit, flag,
	)
}

func formatDeltaDescription(name string, current, previous, percentChange float64, unit string) string {
	direction := "increased"
	if current < previous {
		direction = "decreased"
	}
	return fmt.Sprintf(
		"%s has %s significantly from %.2f to %.2f %s (%.1f%% change). "+
		"Evaluate for acute changes in patient condition.",
		name, direction, previous, current, unit, percentChange,
	)
}

func formatPatternDescription(name string, confidence float64) string {
	return fmt.Sprintf(
		"Clinical pattern '%s' detected with %.0f%% confidence. "+
		"Review panel results and consider appropriate workup.",
		name, confidence*100,
	)
}

func formatTrendDescription(name, trajectory string, rate float64, unit string) string {
	return fmt.Sprintf(
		"%s shows %s trajectory with rate of change %.3f %s/day. "+
		"Monitor closely and evaluate for underlying causes.",
		name, trajectory, rate, unit,
	)
}

func formatCareGapDescription(name string, daysOverdue int) string {
	return fmt.Sprintf(
		"Care gap identified: %s is %d days overdue. "+
		"Schedule appropriate follow-up testing.",
		name, daysOverdue,
	)
}

func severityToPriority(s Severity) int {
	switch s {
	case SeverityCritical:
		return 1
	case SeverityHigh:
		return 2
	case SeverityMedium:
		return 3
	case SeverityLow:
		return 4
	default:
		return 5
	}
}

// ToJSON serializes the event to JSON
func (e *GovernanceEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// FromJSON deserializes an event from JSON
func FromJSON(data []byte) (*GovernanceEvent, error) {
	var event GovernanceEvent
	if err := json.Unmarshal(data, &event); err != nil {
		return nil, err
	}
	return &event, nil
}

