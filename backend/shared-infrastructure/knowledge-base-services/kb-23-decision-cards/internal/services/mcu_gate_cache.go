package services

import (
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/cache"
	"kb-23-decision-cards/internal/database"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

type MCUGateCache struct {
	cache   *cache.CacheClient
	db      *database.Database
	metrics *metrics.Collector
	log     *zap.Logger
}

func NewMCUGateCache(c *cache.CacheClient, db *database.Database, m *metrics.Collector, log *zap.Logger) *MCUGateCache {
	return &MCUGateCache{cache: c, db: db, metrics: m, log: log}
}

// WriteGate writes the enriched MCU_GATE response to Redis and records gate history.
func (g *MCUGateCache) WriteGate(card *models.DecisionCard) error {
	patientID := card.PatientID.String()

	// Read current gate for history tracking
	var previousGate *models.MCUGate
	var currentResponse models.EnrichedMCUGateResponse
	if err := g.cache.GetMCUGate(patientID, &currentResponse); err == nil {
		prev := currentResponse.MCUGate
		previousGate = &prev
	}

	// Build enriched response
	adherenceGain := card.AdherenceGainFactor
	if adherenceGain == 0 {
		adherenceGain = 1.0 // Fallback default if not enriched
	}
	response := models.EnrichedMCUGateResponse{
		MCUGate:                card.MCUGate,
		ObservationReliability: card.ObservationReliability,
		ReEntryProtocol:        card.ReEntryProtocol,
		GateCardID:             card.CardID,
		AdherenceGainFactor:    adherenceGain,
	}
	if card.DoseAdjustmentNotes != nil {
		response.DoseAdjustmentNotes = *card.DoseAdjustmentNotes
	}

	// Write to Redis
	if err := g.cache.SetMCUGate(patientID, response); err != nil {
		return fmt.Errorf("redis set MCU gate: %w", err)
	}

	// Track gate transition in DB
	history := models.MCUGateHistory{
		HistoryID:        uuid.New(),
		PatientID:        card.PatientID,
		CardID:           card.CardID,
		GateValue:        card.MCUGate,
		PreviousGate:     previousGate,
		SessionID:        card.SessionID,
		TransitionReason: card.MCUGateRationale,
		ReEntryProtocol:  card.ReEntryProtocol,
		CreatedAt:        time.Now(),
	}

	if err := g.db.DB.Create(&history).Error; err != nil {
		g.log.Error("gate history record failed", zap.Error(err))
	}

	// Track metrics
	fromGate := "NONE"
	if previousGate != nil {
		fromGate = string(*previousGate)
	}
	g.metrics.GateTransitions.WithLabelValues(fromGate, string(card.MCUGate)).Inc()

	g.log.Info("MCU gate written",
		zap.String("patient_id", patientID),
		zap.String("gate", string(card.MCUGate)),
		zap.String("previous", fromGate),
	)

	return nil
}

// ReadGate reads the enriched MCU_GATE response from Redis cache.
func (g *MCUGateCache) ReadGate(patientID string) (*models.EnrichedMCUGateResponse, error) {
	var response models.EnrichedMCUGateResponse
	if err := g.cache.GetMCUGate(patientID, &response); err != nil {
		return nil, err
	}
	return &response, nil
}

// ReadGateFromDB falls back to the latest gate history record from DB.
func (g *MCUGateCache) ReadGateFromDB(patientID string) (*models.EnrichedMCUGateResponse, error) {
	var history models.MCUGateHistory
	result := g.db.DB.Where("patient_id = ?", patientID).
		Order("created_at DESC").
		First(&history)

	if result.Error != nil {
		return nil, result.Error
	}

	// Reconstruct enriched response from history
	response := &models.EnrichedMCUGateResponse{
		MCUGate:                history.GateValue,
		ReEntryProtocol:        history.ReEntryProtocol,
		GateCardID:             history.CardID,
		AdherenceGainFactor:    1.0,
		ObservationReliability: models.ReliabilityHigh,
	}

	// Write back to cache for future reads
	if err := g.cache.SetMCUGate(patientID, response); err != nil {
		g.log.Warn("cache re-populate failed", zap.Error(err))
	}

	return response, nil
}
