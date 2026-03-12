package models

import (
	"time"

	"github.com/google/uuid"
)

// AnswerValue enumerates valid answer types.
type AnswerValue string

const (
	AnswerYes     AnswerValue = "YES"
	AnswerNo      AnswerValue = "NO"
	AnswerPata    AnswerValue = "PATA_NAHI"
)

// SessionAnswer is the append-only answer log.
// Each answer records the LR applied, information gain, and pata-nahi status.
type SessionAnswer struct {
	AnswerID  uuid.UUID `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"answer_id"`
	SessionID uuid.UUID `gorm:"type:uuid;index;not null" json:"session_id"`

	QuestionID  string      `gorm:"type:varchar(64);not null" json:"question_id"`
	AnswerValue string      `gorm:"type:varchar(32);not null" json:"answer_value"`

	// Log(LR) applied per differential; 0.0 for PATA_NAHI (F-04)
	LRApplied JSONB `gorm:"type:jsonb;default:'{}'" json:"lr_applied"`

	// H_before - H_after; written to KB-21 telemetry
	InformationGainObserved float64 `gorm:"type:float8;default:0" json:"information_gain_observed"`

	// Fast-path filter for pata-nahi rate queries
	WasPataNahi bool `gorm:"column:was_pata_nahi;type:bool;index;default:false" json:"was_pata_nahi"`

	// WhatsApp round-trip diagnostic
	AnswerLatencyMS int `gorm:"type:int;default:0" json:"answer_latency_ms"`

	AnsweredAt time.Time `gorm:"type:timestamptz;index;not null;autoCreateTime" json:"answered_at"`
}

func (SessionAnswer) TableName() string { return "session_answers" }

// IsPataNahi returns true if this answer is a "pata nahi" response.
func (a *SessionAnswer) IsPataNahi() bool {
	return a.AnswerValue == string(AnswerPata)
}
