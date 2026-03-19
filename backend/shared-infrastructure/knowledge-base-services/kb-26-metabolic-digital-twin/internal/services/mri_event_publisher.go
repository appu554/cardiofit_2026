package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"kb-26-metabolic-digital-twin/internal/models"

	"github.com/google/uuid"
	"go.uber.org/zap"
)

// MRIEventPublisher publishes MRI deterioration events to KB-22 and KB-23.
type MRIEventPublisher struct {
	kb22URL    string
	kb23URL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewMRIEventPublisher creates a publisher for MRI cross-service events.
func NewMRIEventPublisher(kb22URL, kb23URL string, logger *zap.Logger) *MRIEventPublisher {
	return &MRIEventPublisher{
		kb22URL:    kb22URL,
		kb23URL:    kb23URL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

// MRIClinicalSignal is the payload sent to KB-23 POST /api/v1/clinical-signals.
type MRIClinicalSignal struct {
	PatientID         string  `json:"patient_id"`
	SignalType        string  `json:"signal_type"`
	NodeID            string  `json:"node_id"`
	Category          string  `json:"category"`
	Severity          string  `json:"severity"`
	Score             float64 `json:"score"`
	TopDriver         string  `json:"top_driver"`
	Trend             string  `json:"trend"`
	MCUGateSuggestion string  `json:"mcu_gate_suggestion"`
}

// PublishDeteriorationEvent checks if MRI crossed a category boundary and
// publishes events to KB-22 and KB-23. Fire-and-forget — errors are logged but don't block.
func (p *MRIEventPublisher) PublishDeteriorationEvent(
	ctx context.Context,
	patientID uuid.UUID,
	current models.MRIResult,
	previous *models.MRIScore,
) {
	if previous == nil {
		return
	}

	prevSeverity := categorySeverity(previous.Category)
	currSeverity := categorySeverity(current.Category)
	if currSeverity <= prevSeverity {
		return
	}

	p.logger.Info("MRI category worsened — publishing deterioration events",
		zap.String("patient_id", patientID.String()),
		zap.String("from", previous.Category),
		zap.String("to", current.Category),
		zap.Float64("score", current.Score),
	)

	pid := patientID.String()

	go p.notifyKB22(pid)
	go p.notifyKB23(pid, current)
}

func (p *MRIEventPublisher) notifyKB22(patientID string) {
	payload := map[string]string{
		"patient_id":    patientID,
		"stratum_label": "DM_HTN_base",
	}
	body, _ := json.Marshal(payload)

	url := fmt.Sprintf("%s/api/v1/signals/events/twin-state-update", p.kb22URL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		p.logger.Error("failed to create KB-22 request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Warn("KB-22 twin-state-update notification failed", zap.Error(err))
		return
	}
	resp.Body.Close()
	p.logger.Debug("KB-22 notified of MRI deterioration", zap.Int("status", resp.StatusCode))
}

func (p *MRIEventPublisher) notifyKB23(patientID string, result models.MRIResult) {
	signal := MRIClinicalSignal{
		PatientID:         patientID,
		SignalType:        "MRI_DETERIORATION",
		NodeID:            "MRI-01",
		Category:          result.Category,
		Severity:          mriCategoryToSeverity(result.Category),
		Score:             result.Score,
		TopDriver:         result.TopDriver,
		Trend:             result.Trend,
		MCUGateSuggestion: mriCategoryToGate(result.Category),
	}
	body, _ := json.Marshal(signal)

	url := fmt.Sprintf("%s/api/v1/clinical-signals", p.kb23URL)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		p.logger.Error("failed to create KB-23 request", zap.Error(err))
		return
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		p.logger.Warn("KB-23 clinical-signal notification failed", zap.Error(err))
		return
	}
	resp.Body.Close()
	p.logger.Debug("KB-23 notified of MRI deterioration", zap.Int("status", resp.StatusCode))
}

// categorySeverity maps MRI category to numeric severity (higher = worse).
func categorySeverity(category string) int {
	switch category {
	case models.MRICategoryOptimal:
		return 0
	case models.MRICategoryMildDysregulation:
		return 1
	case models.MRICategoryModerateDeterioration:
		return 2
	case models.MRICategoryHighDeterioration:
		return 3
	default:
		return 0
	}
}

// mriCategoryToSeverity maps MRI category to clinical signal severity.
func mriCategoryToSeverity(category string) string {
	switch category {
	case models.MRICategoryHighDeterioration:
		return "IMMEDIATE"
	case models.MRICategoryModerateDeterioration:
		return "URGENT"
	case models.MRICategoryMildDysregulation:
		return "ROUTINE"
	default:
		return "ROUTINE"
	}
}

// mriCategoryToGate maps MRI category to MCU gate suggestion.
// Spec §7, Table 8: MRI >75 = mandatory medication review.
func mriCategoryToGate(category string) string {
	switch category {
	case models.MRICategoryHighDeterioration:
		return "MODIFY"
	case models.MRICategoryModerateDeterioration:
		return "SAFE"
	default:
		return "SAFE"
	}
}
