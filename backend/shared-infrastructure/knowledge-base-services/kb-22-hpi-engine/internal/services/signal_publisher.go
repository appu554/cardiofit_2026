package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
	"gorm.io/gorm"

	"kb-22-hpi-engine/internal/models"
)

// SignalPublisher publishes ClinicalSignalEvent to KB-23 (/api/v1/clinical-signals)
// via HTTP and to a Kafka topic. Both channels are best-effort: failures are logged
// but never propagated to the caller. Unpublished events (published_to_kb23=false)
// can be swept and retried by a background job.
type SignalPublisher struct {
	kb23URL    string
	kb23Client *http.Client
	kafka      KafkaPublisher
	kafkaTopic string
	retryCount int
	retryDelay time.Duration
	db         *gorm.DB
	log        *zap.Logger
}

// NewSignalPublisher constructs a SignalPublisher.
func NewSignalPublisher(
	kb23URL string,
	kafka KafkaPublisher,
	kafkaTopic string,
	retryCount int,
	retryDelay time.Duration,
	db *gorm.DB,
	log *zap.Logger,
) *SignalPublisher {
	return &SignalPublisher{
		kb23URL: kb23URL,
		kb23Client: &http.Client{
			Timeout: 5 * time.Second,
		},
		kafka:      kafka,
		kafkaTopic: kafkaTopic,
		retryCount: retryCount,
		retryDelay: retryDelay,
		db:         db,
		log:        log,
	}
}

// Publish sends a ClinicalSignalEvent to KB-23 (with retry) and to Kafka
// (fire-and-forget). It always returns nil — publishing failures are logged
// so that unpublished events can be retried later via a sweep of
// clinical_signals where published_to_kb23=false.
func (p *SignalPublisher) Publish(ctx context.Context, event *models.ClinicalSignalEvent) error {
	// --- 1. POST to KB-23 with retry ---
	body, err := json.Marshal(event)
	if err != nil {
		p.log.Error("signal_publisher: failed to marshal ClinicalSignalEvent",
			zap.String("event_id", event.EventID),
			zap.Error(err),
		)
		// Marshal failure is a programming error; still return nil (non-fatal contract).
		goto kafkaPublish
	}

	{
		url := fmt.Sprintf("%s/api/v1/clinical-signals", p.kb23URL)
		published := p.postToKB23WithRetry(ctx, url, body, event.EventID)
		if published {
			p.markPublishedToDB(event.EventID)
		}
	}

kafkaPublish:
	// --- 2. Publish to Kafka (fire-and-forget) ---
	if err := p.kafka.Publish(ctx, p.kafkaTopic, event.PatientID, event); err != nil {
		p.log.Warn("signal_publisher: kafka publish failed (non-fatal)",
			zap.String("event_id", event.EventID),
			zap.String("patient_id", event.PatientID),
			zap.String("topic", p.kafkaTopic),
			zap.Error(err),
		)
	}

	return nil
}

// postToKB23WithRetry attempts to POST the body to the KB-23 URL up to
// retryCount times. Returns true if KB-23 returned 201 or 204.
func (p *SignalPublisher) postToKB23WithRetry(ctx context.Context, url string, body []byte, eventID string) bool {
	for attempt := 1; attempt <= p.retryCount; attempt++ {
		if ctx.Err() != nil {
			p.log.Warn("signal_publisher: context cancelled, stopping KB-23 retries",
				zap.String("event_id", eventID),
				zap.Int("attempt", attempt),
			)
			return false
		}

		status, err := p.doPost(ctx, url, body)
		if err != nil {
			p.log.Warn("signal_publisher: KB-23 POST failed",
				zap.String("event_id", eventID),
				zap.String("url", url),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", p.retryCount),
				zap.Error(err),
			)
		} else {
			switch status {
			case http.StatusCreated: // 201 — card created
				p.log.Info("signal_publisher: KB-23 accepted signal (201)",
					zap.String("event_id", eventID),
				)
				return true
			case http.StatusNoContent: // 204 — acknowledged, no card needed
				p.log.Info("signal_publisher: KB-23 acknowledged signal, no card needed (204)",
					zap.String("event_id", eventID),
				)
				return true
			default:
				p.log.Warn("signal_publisher: KB-23 returned unexpected status",
					zap.String("event_id", eventID),
					zap.Int("status", status),
					zap.Int("attempt", attempt),
					zap.Int("max_retries", p.retryCount),
				)
			}
		}

		if attempt < p.retryCount {
			select {
			case <-time.After(p.retryDelay):
			case <-ctx.Done():
				p.log.Warn("signal_publisher: context cancelled during retry wait",
					zap.String("event_id", eventID),
				)
				return false
			}
		}
	}

	p.log.Error("signal_publisher: KB-23 publish failed after all retries — event will remain unpublished",
		zap.String("event_id", eventID),
		zap.String("url", fmt.Sprintf("%s/api/v1/clinical-signals", p.kb23URL)),
		zap.Int("retries", p.retryCount),
	)
	return false
}

// doPost performs a single HTTP POST and returns the response status code.
// Returns (0, err) on network/request errors.
func (p *SignalPublisher) doPost(ctx context.Context, url string, body []byte) (int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.kb23Client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("HTTP POST: %w", err)
	}
	defer func() {
		io.Copy(io.Discard, resp.Body) //nolint:errcheck
		resp.Body.Close()
	}()

	return resp.StatusCode, nil
}

// markPublishedToDB updates published_to_kb23=true in the clinical_signals table.
// A nil db is silently skipped (used in tests and dev mode).
func (p *SignalPublisher) markPublishedToDB(eventID string) {
	if p.db == nil {
		return
	}
	p.db.Model(&struct{ PublishedToKB23 bool }{}).
		Table("clinical_signals").
		Where("event_id = ?", eventID).
		Update("published_to_kb23", true)
}
