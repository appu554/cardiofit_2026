package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// EngagementSeason represents the patient's current lifecycle season (Spec Section 2).
type EngagementSeason string

const (
	SeasonCorrection    EngagementSeason = "CORRECTION"    // S1: 90 days, active correction
	SeasonConsolidation EngagementSeason = "CONSOLIDATION" // S2: bi-weekly check-ins
	SeasonIndependence  EngagementSeason = "INDEPENDENCE"  // S3: weekly self-directed
	SeasonStability     EngagementSeason = "STABILITY"     // S4: monthly touchpoints
	SeasonPartnership   EngagementSeason = "PARTNERSHIP"   // S5: quarterly check-ins
)

// SeasonConfig defines coaching parameters per engagement season (Spec Section 4).
// Burden DECREASES over time, value INCREASES — the inverted burden-value curve.
type SeasonConfig struct {
	Season              EngagementSeason
	MaxNudgesPerDay     int    // decreasing: S1=5, S2=3, S3=2, S4=1, S5=1
	MaxDataActionsMonth int    // S1=50, S2=30, S3=15, S4=8, S5=4
	CheckinFrequency    string // "daily", "biweekly", "weekly", "monthly", "quarterly"
	EventTriggered      bool   // S1-S2=false (calendar), S3-S5=true (event-only)
}

// DefaultSeasonConfigs returns the inverted burden-value curve configuration.
func DefaultSeasonConfigs() map[EngagementSeason]SeasonConfig {
	return map[EngagementSeason]SeasonConfig{
		SeasonCorrection: {
			Season: SeasonCorrection, MaxNudgesPerDay: 5,
			MaxDataActionsMonth: 50, CheckinFrequency: "daily", EventTriggered: false,
		},
		SeasonConsolidation: {
			Season: SeasonConsolidation, MaxNudgesPerDay: 3,
			MaxDataActionsMonth: 30, CheckinFrequency: "biweekly", EventTriggered: false,
		},
		SeasonIndependence: {
			Season: SeasonIndependence, MaxNudgesPerDay: 2,
			MaxDataActionsMonth: 15, CheckinFrequency: "weekly", EventTriggered: true,
		},
		SeasonStability: {
			Season: SeasonStability, MaxNudgesPerDay: 1,
			MaxDataActionsMonth: 8, CheckinFrequency: "monthly", EventTriggered: true,
		},
		SeasonPartnership: {
			Season: SeasonPartnership, MaxNudgesPerDay: 1,
			MaxDataActionsMonth: 4, CheckinFrequency: "quarterly", EventTriggered: true,
		},
	}
}

// CeremonyRecord tracks delivery of transition celebrations to prevent duplicates.
type CeremonyRecord struct {
	ID           uuid.UUID          `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	PatientID    string             `gorm:"uniqueIndex:idx_ceremony_patient_season;not null" json:"patient_id"`
	FromSeason   EngagementSeason   `gorm:"type:varchar(20);not null" json:"from_season"`
	ToSeason     EngagementSeason   `gorm:"type:varchar(20);uniqueIndex:idx_ceremony_patient_season;not null" json:"to_season"`
	CeremonyType string             `gorm:"type:varchar(50);not null" json:"ceremony_type"` // GRADUATION, MILESTONE, ANNUAL
	DeliveredAt  time.Time          `gorm:"not null" json:"delivered_at"`
	Channel      InteractionChannel `gorm:"type:varchar(20)" json:"channel"`
	Acknowledged bool               `gorm:"default:false" json:"acknowledged"`
	CreatedAt    time.Time          `gorm:"autoCreateTime" json:"created_at"`
}

// BeforeCreate ensures a UUID is set for CeremonyRecord (SQLite compat).
func (c *CeremonyRecord) BeforeCreate(tx *gorm.DB) error {
	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}
	return nil
}
