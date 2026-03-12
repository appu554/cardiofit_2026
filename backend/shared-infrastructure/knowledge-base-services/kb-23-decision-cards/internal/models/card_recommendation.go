package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// CardRecommendation is a single actionable recommendation within a DecisionCard.
type CardRecommendation struct {
	RecommendationID          uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"recommendation_id"`
	CardID                    uuid.UUID          `gorm:"type:uuid;index;not null" json:"card_id"`
	RecType                   RecommendationType `gorm:"type:varchar(30);not null" json:"rec_type"`
	Urgency                   Urgency            `gorm:"type:varchar(20);not null" json:"urgency"`
	Target                    string             `json:"target"`
	ActionTextEn              string             `gorm:"type:text" json:"action_text_en"`
	ActionTextHi              string             `gorm:"type:text" json:"action_text_hi"`
	RationaleEn               string             `gorm:"type:text" json:"rationale_en"`
	GuidelineRef              string             `json:"guideline_ref"`
	ConditionCriteria         JSONB              `gorm:"type:jsonb" json:"condition_criteria,omitempty"`
	ConditionStatus           *ConditionStatus   `gorm:"type:varchar(20)" json:"condition_status,omitempty"`
	ConfidenceTierRequired    ConfidenceTier     `gorm:"type:varchar(20)" json:"confidence_tier_required"`
	BypassesConfidenceGate    bool               `gorm:"default:false" json:"bypasses_confidence_gate"`
	TriggerConditionEn        *string            `gorm:"type:text" json:"trigger_condition_en,omitempty"`
	TriggerConditionHi        *string            `gorm:"type:text" json:"trigger_condition_hi,omitempty"`
	FromSecondaryDifferential bool               `gorm:"default:false" json:"from_secondary_differential"`
	ConflictFlag              bool               `gorm:"default:false" json:"conflict_flag"`
	SortOrder                 int                `json:"sort_order"`
	CreatedAt                 time.Time          `json:"created_at"`
}

// TableName sets the PostgreSQL table name.
func (CardRecommendation) TableName() string { return "card_recommendations" }

// BeforeCreate generates a UUID primary key if not already set.
func (r *CardRecommendation) BeforeCreate(tx *gorm.DB) error {
	if r.RecommendationID == uuid.Nil {
		r.RecommendationID = uuid.New()
	}
	return nil
}
