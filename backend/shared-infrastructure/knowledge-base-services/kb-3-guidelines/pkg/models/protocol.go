package models

import "time"

// ProtocolType categorizes protocols
type ProtocolType string

const (
	ProtocolAcute      ProtocolType = "acute"
	ProtocolChronic    ProtocolType = "chronic"
	ProtocolPreventive ProtocolType = "preventive"
)

// Protocol represents a clinical protocol definition
type Protocol struct {
	ProtocolID      string           `json:"protocol_id"`
	Name            string           `json:"name"`
	Type            ProtocolType     `json:"type"`
	GuidelineSource string           `json:"guideline_source"`
	Version         string           `json:"version"`
	Description     string           `json:"description,omitempty"`
	Stages          []Stage          `json:"stages"`
	Constraints     []TimeConstraint `json:"constraints"`
	EntryConditions []Condition      `json:"entry_conditions"`
	ExitConditions  []Condition      `json:"exit_conditions"`
	Active          bool             `json:"active"`
	EffectiveDate   time.Time        `json:"effective_date"`
	CreatedAt       time.Time        `json:"created_at"`
	UpdatedAt       time.Time        `json:"updated_at"`
}

// Stage represents a protocol stage
type Stage struct {
	StageID         string        `json:"stage_id"`
	Name            string        `json:"name"`
	Description     string        `json:"description,omitempty"`
	Order           int           `json:"order"`
	Actions         []Action      `json:"actions"`
	EntryConditions []Condition   `json:"entry_conditions,omitempty"`
	ExitConditions  []Condition   `json:"exit_conditions,omitempty"`
	MaxDuration     time.Duration `json:"max_duration,omitempty"`
}

// Action within a stage
type Action struct {
	ActionID    string         `json:"action_id"`
	Name        string         `json:"name"`
	Type        ActionType     `json:"type"`
	Description string         `json:"description,omitempty"`
	Required    bool           `json:"required"`
	Deadline    time.Duration  `json:"deadline"` // Relative to stage start
	GracePeriod time.Duration  `json:"grace_period"`
	Parameters  map[string]any `json:"parameters,omitempty"`
}

// TimeConstraint for acute protocols
type TimeConstraint struct {
	ConstraintID   string        `json:"constraint_id"`
	Action         string        `json:"action"`
	Description    string        `json:"description,omitempty"`
	Deadline       time.Duration `json:"deadline"`
	GracePeriod    time.Duration `json:"grace_period"`
	AlertThreshold time.Duration `json:"alert_threshold"`
	Severity       Severity      `json:"severity"`
	Reference      string        `json:"reference"` // Guideline reference
}

// Condition for entry/exit
type Condition struct {
	Type        string `json:"type"`     // lab, diagnosis, medication, age, etc.
	Field       string `json:"field"`
	Operator    string `json:"operator"` // =, !=, >, <, >=, <=, contains, exists
	Value       any    `json:"value"`
	Description string `json:"description,omitempty"`
}

// ListProtocolsResponse for API
type ListProtocolsResponse struct {
	Protocols  []Protocol `json:"protocols"`
	Total      int        `json:"total"`
	Page       int        `json:"page,omitempty"`
	PageSize   int        `json:"page_size,omitempty"`
}

// ProtocolSummary for listing
type ProtocolSummary struct {
	ProtocolID      string       `json:"protocol_id"`
	Name            string       `json:"name"`
	Type            ProtocolType `json:"type"`
	GuidelineSource string       `json:"guideline_source"`
	Active          bool         `json:"active"`
	StageCount      int          `json:"stage_count"`
	ConstraintCount int          `json:"constraint_count"`
}
