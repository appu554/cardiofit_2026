package models

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
)

// SessionStatus represents the HPI session state machine.
type SessionStatus string

const (
	StatusInitialising      SessionStatus = "INITIALISING"
	StatusActive            SessionStatus = "ACTIVE"
	StatusSuspended         SessionStatus = "SUSPENDED"
	StatusSafetyEscalated   SessionStatus = "SAFETY_ESCALATED"
	StatusCompleted         SessionStatus = "COMPLETED"
	StatusAbandoned         SessionStatus = "ABANDONED"
	StatusStratumDrifted    SessionStatus = "STRATUM_DRIFTED"
	// G16: session terminated early due to excessive consecutive pata-nahi answers.
	StatusPartialAssessment SessionStatus = "PARTIAL_ASSESSMENT"
)

// HPISession is the central session state.
// Stored in PostgreSQL (authoritative) with Redis cache (24h TTL).
type HPISession struct {
	SessionID   uuid.UUID     `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"session_id"`
	PatientID   uuid.UUID     `gorm:"type:uuid;index;not null" json:"patient_id"`
	NodeID      string        `gorm:"type:varchar(64);index;not null" json:"node_id"`
	StratumLabel string       `gorm:"type:varchar(32);not null" json:"stratum_label"`
	CKDSubstage  *string      `gorm:"type:varchar(16)" json:"ckd_substage,omitempty"`
	Status       SessionStatus `gorm:"type:varchar(32);not null;default:'INITIALISING'" json:"status"`

	// F-01: log-odds internal state vector
	LogOddsState     JSONB `gorm:"type:jsonb;default:'{}'" json:"log_odds_state"`
	// CM audit trail
	CMLogDeltasApplied JSONB `gorm:"type:jsonb;default:'{}'" json:"cm_log_deltas_applied"`
	// R-02: cluster dampening tracking
	ClusterAnswered JSONB `gorm:"type:jsonb;default:'{}'" json:"cluster_answered"`
	// R-03: reliability modifier from KB-21
	ReliabilityModifier float64 `gorm:"type:float8;default:1.0" json:"reliability_modifier"`
	// Adherence-tier gain factor derived from KB-21 adherence weights
	AdherenceGainFactor float64 `gorm:"type:float8;default:1.0" json:"adherence_gain_factor"`
	// N-01: KB-3 guideline references
	GuidelinePriorRefs StringArray `gorm:"type:text[]" json:"guideline_prior_refs,omitempty"`

	QuestionsAsked    int  `gorm:"type:int;default:0" json:"questions_asked"`
	QuestionsPataNahi int  `gorm:"column:questions_pata_nahi;type:int;default:0" json:"questions_pata_nahi"`
	// G16: consecutive low-confidence (pata-nahi) answer counter.
	// Reset to 0 on any non-pata-nahi answer. Drives cascade protocol.
	ConsecutiveLowConf int `gorm:"type:int;default:0" json:"consecutive_low_conf"`
	SafetyFlagIDs     JSONB `gorm:"type:jsonb;default:'[]'" json:"safety_flags"`
	CurrentQuestionID *string `gorm:"type:varchar(64)" json:"current_question_id,omitempty"`

	// R-04: stratum drift detection on resume
	SubstageDrifted bool `gorm:"type:bool;default:false" json:"substage_drifted"`

	StartedAt      time.Time  `gorm:"type:timestamptz;not null;autoCreateTime" json:"started_at"`
	LastActivityAt time.Time  `gorm:"type:timestamptz;index;not null;autoUpdateTime" json:"last_activity_at"`
	CompletedAt    *time.Time `gorm:"type:timestamptz" json:"completed_at,omitempty"`
	OutcomePublished bool     `gorm:"type:bool;default:false" json:"outcome_published"`
}

func (HPISession) TableName() string { return "hpi_sessions" }

// LogOddsMap returns the parsed log-odds state.
func (s *HPISession) LogOddsMap() (map[string]float64, error) {
	result := make(map[string]float64)
	if len(s.LogOddsState) == 0 {
		return result, nil
	}
	err := json.Unmarshal(s.LogOddsState, &result)
	return result, err
}

// SetLogOdds serialises the log-odds map back to JSONB.
func (s *HPISession) SetLogOdds(m map[string]float64) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	s.LogOddsState = data
	return nil
}

// ClusterAnsweredMap returns parsed cluster tracking state.
func (s *HPISession) ClusterAnsweredMap() (map[string]int, error) {
	result := make(map[string]int)
	if len(s.ClusterAnswered) == 0 {
		return result, nil
	}
	err := json.Unmarshal(s.ClusterAnswered, &result)
	return result, err
}

func (s *HPISession) SetClusterAnswered(m map[string]int) error {
	data, err := json.Marshal(m)
	if err != nil {
		return err
	}
	s.ClusterAnswered = data
	return nil
}

// --- Custom GORM types ---

// JSONB implements driver.Valuer and sql.Scanner for JSONB columns.
type JSONB []byte

func (j JSONB) Value() (driver.Value, error) {
	if len(j) == 0 {
		return nil, nil
	}
	return []byte(j), nil
}

func (j *JSONB) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		*j = append((*j)[0:0], v...)
	case string:
		*j = []byte(v)
	default:
		return errors.New("unsupported JSONB scan type")
	}
	return nil
}

func (j JSONB) MarshalJSON() ([]byte, error) {
	if len(j) == 0 {
		return []byte("null"), nil
	}
	return j, nil
}

func (j *JSONB) UnmarshalJSON(data []byte) error {
	*j = append((*j)[0:0], data...)
	return nil
}

// StringArray implements driver.Valuer and sql.Scanner for PostgreSQL text[].
type StringArray []string

func (a StringArray) Value() (driver.Value, error) {
	if a == nil {
		return nil, nil
	}
	return json.Marshal(a)
}

func (a *StringArray) Scan(value interface{}) error {
	if value == nil {
		*a = nil
		return nil
	}
	switch v := value.(type) {
	case []byte:
		return json.Unmarshal(v, a)
	case string:
		return json.Unmarshal([]byte(v), a)
	default:
		return errors.New("unsupported StringArray scan type")
	}
}
