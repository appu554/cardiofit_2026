package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/metrics"
	"kb-23-decision-cards/internal/models"
)

type KB19Publisher struct {
	cfg     *config.Config
	metrics *metrics.Collector
	log     *zap.Logger
	client  *http.Client
}

func NewKB19Publisher(cfg *config.Config, m *metrics.Collector, log *zap.Logger) *KB19Publisher {
	return &KB19Publisher{
		cfg:     cfg,
		metrics: m,
		log:     log,
		client: &http.Client{
			Timeout: cfg.KB19Timeout(),
		},
	}
}

// PublishGateChanged sends MCU_GATE_CHANGED event to KB-19 (N-02).
func (p *KB19Publisher) PublishGateChanged(card *models.DecisionCard) {
	event := models.KB19Event{
		EventType:       models.EventMCUGateChanged,
		PatientID:       card.PatientID,
		SessionID:       card.SessionID,
		CardID:          card.CardID,
		Gate:            card.MCUGate,
		ReEntryProtocol: card.ReEntryProtocol,
		Timestamp:       time.Now(),
	}

	if card.DoseAdjustmentNotes != nil {
		event.DoseAdjustmentNotes = *card.DoseAdjustmentNotes
	}

	if err := p.publishEvent(event); err != nil {
		p.log.Error("MCU_GATE_CHANGED publish failed",
			zap.String("card_id", card.CardID.String()),
			zap.Error(err),
		)
		p.metrics.KB19PublishErrors.Inc()
	}
}

// PublishSafetyAlert sends SAFETY_ALERT event to KB-19 for IMMEDIATE flags (< 2s SLA).
func (p *KB19Publisher) PublishSafetyAlert(patientID uuid.UUID, sessionID *uuid.UUID, flag models.SafetyFlagEntry) {
	event := models.KB19Event{
		EventType:         models.EventSafetyAlert,
		PatientID:         patientID,
		SessionID:         sessionID,
		FlagID:            flag.FlagID,
		Severity:          flag.Severity,
		RecommendedAction: flag.RecommendedAction,
		Timestamp:         time.Now(),
	}

	if err := p.publishEvent(event); err != nil {
		p.log.Error("SAFETY_ALERT publish failed",
			zap.String("flag_id", flag.FlagID),
			zap.Error(err),
		)
		p.metrics.KB19PublishErrors.Inc()
	}
}

func (p *KB19Publisher) publishEvent(event models.KB19Event) error {
	start := time.Now()

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal event: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/events", p.cfg.KB19URL)
	resp, err := p.client.Post(url, "application/json", bytes.NewReader(body))

	p.metrics.KB19PublishLatency.Observe(float64(time.Since(start).Milliseconds()))

	if err != nil {
		return fmt.Errorf("KB-19 POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return fmt.Errorf("KB-19 returned status %d", resp.StatusCode)
	}

	p.log.Info("event published to KB-19",
		zap.String("event_type", string(event.EventType)),
		zap.String("patient_id", event.PatientID.String()),
	)
	return nil
}
