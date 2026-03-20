package services

import (
	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// SeasonCoach modulates coaching cadence based on engagement season.
// Implements the inverted burden-value curve from Spec Section 4.
type SeasonCoach struct {
	db      *gorm.DB
	logger  *zap.Logger
	configs map[models.EngagementSeason]models.SeasonConfig
}

func NewSeasonCoach(db *gorm.DB, logger *zap.Logger) *SeasonCoach {
	return &SeasonCoach{
		db:      db,
		logger:  logger,
		configs: models.DefaultSeasonConfigs(),
	}
}

func (sc *SeasonCoach) GetMaxNudgesPerDay(season models.EngagementSeason) int {
	if cfg, ok := sc.configs[season]; ok {
		return cfg.MaxNudgesPerDay
	}
	return 5
}

func (sc *SeasonCoach) GetMaxDataActionsPerMonth(season models.EngagementSeason) int {
	if cfg, ok := sc.configs[season]; ok {
		return cfg.MaxDataActionsMonth
	}
	return 50
}

func (sc *SeasonCoach) IsEventTriggered(season models.EngagementSeason) bool {
	if cfg, ok := sc.configs[season]; ok {
		return cfg.EventTriggered
	}
	return false
}

func (sc *SeasonCoach) ShouldContactPatient(season models.EngagementSeason, hasTriggerEvent bool) bool {
	if !sc.IsEventTriggered(season) {
		return true
	}
	return hasTriggerEvent
}

func (sc *SeasonCoach) GetConfig(season models.EngagementSeason) models.SeasonConfig {
	if cfg, ok := sc.configs[season]; ok {
		return cfg
	}
	return sc.configs[models.SeasonCorrection]
}
