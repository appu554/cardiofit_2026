package services

import (
	"fmt"
	"time"

	"kb-21-behavioral-intelligence/internal/models"

	"go.uber.org/zap"
	"gorm.io/gorm"
)

// CeremonyEngine delivers transition celebrations between engagement seasons.
type CeremonyEngine struct {
	db     *gorm.DB
	logger *zap.Logger
}

// NewCeremonyEngine creates a CeremonyEngine. Pass nil db for message-only usage.
func NewCeremonyEngine(db *gorm.DB, logger *zap.Logger) *CeremonyEngine {
	return &CeremonyEngine{db: db, logger: logger}
}

// ceremonyMessages maps season transitions to clinically-appropriate celebration text.
var ceremonyMessages = map[string]string{
	"CORRECTION→CONSOLIDATION":   "Congratulations! Your 90-day correction is complete. Your metabolic health has improved significantly. Now we enter a phase where you maintain these gains with less effort.",
	"CONSOLIDATION→INDEPENDENCE": "You've proven your metabolic stability. You're now entering Independence — we'll check in less often because your body is holding steady. You've earned this.",
	"INDEPENDENCE→STABILITY":     "Six months of sustained improvement! You've moved from following a program to living a healthier life. We'll be here when you need us.",
	"STABILITY→PARTNERSHIP":      "One year of metabolic transformation. This is no longer a program — it's a partnership. Your Annual Health Narrative tells the full story of how far you've come.",
}

// GetCeremonyMessage returns the celebration text for a season transition.
// Falls back to a generic message for unknown transitions.
func (ce *CeremonyEngine) GetCeremonyMessage(from, to models.EngagementSeason) string {
	key := fmt.Sprintf("%s→%s", from, to)
	if msg, ok := ceremonyMessages[key]; ok {
		return msg
	}
	return fmt.Sprintf("Congratulations on reaching %s! Your health journey continues.", to)
}

// IsCeremonyDelivered checks whether a transition ceremony has already been
// delivered for the given patient and target season (idempotency guard).
func (ce *CeremonyEngine) IsCeremonyDelivered(patientID string, toSeason models.EngagementSeason) (bool, error) {
	if ce.db == nil {
		return false, nil
	}
	var count int64
	err := ce.db.Model(&models.CeremonyRecord{}).
		Where("patient_id = ? AND to_season = ?", patientID, toSeason).
		Count(&count).Error
	return count > 0, err
}

// RecordCeremony persists a ceremony delivery record. The unique index on
// (patient_id, to_season) prevents duplicate deliveries at the DB level.
func (ce *CeremonyEngine) RecordCeremony(patientID string, from, to models.EngagementSeason, ceremonyType string, channel models.InteractionChannel) error {
	if ce.db == nil {
		return nil
	}
	record := models.CeremonyRecord{
		PatientID:    patientID,
		FromSeason:   from,
		ToSeason:     to,
		CeremonyType: ceremonyType,
		Channel:      channel,
		DeliveredAt:  time.Now(),
	}
	return ce.db.Create(&record).Error
}
