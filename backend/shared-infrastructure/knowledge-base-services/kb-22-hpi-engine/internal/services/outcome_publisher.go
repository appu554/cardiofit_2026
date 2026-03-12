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

	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/metrics"
	"kb-22-hpi-engine/internal/models"
)

// OutcomePublisher publishes HPI_COMPLETE and SAFETY_ALERT events to
// downstream knowledge bases (KB-23 Decision Cards and KB-19 Protocol
// Orchestrator).
//
// HPI_COMPLETE is published on session completion to both KB-23 and KB-19.
// SAFETY_ALERT is published immediately when an IMMEDIATE-severity safety
// flag fires, with a fast-path retry (5s interval instead of 30s).
type OutcomePublisher struct {
	config  *config.Config
	log     *zap.Logger
	metrics *metrics.Collector
	client  *http.Client
}

// NewOutcomePublisher creates a new OutcomePublisher.
func NewOutcomePublisher(cfg *config.Config, log *zap.Logger, m *metrics.Collector) *OutcomePublisher {
	return &OutcomePublisher{
		config:  cfg,
		log:     log,
		metrics: m,
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// PublishHPIComplete sends the HPI_COMPLETE event to KB-23 (/api/v1/decision-cards)
// and KB-19 (/api/v1/events). Both targets receive the same payload.
//
// Retry policy: up to 3 attempts with OutcomeRetryDelay (default 30s) between
// retries. Each target is retried independently; failure on one does not block
// the other.
func (p *OutcomePublisher) PublishHPIComplete(ctx context.Context, event models.HPICompleteEvent) error {
	event.EventType = models.EventHPIComplete

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal HPI_COMPLETE event: %w", err)
	}

	type publishResult struct {
		target string
		err    error
	}

	results := make(chan publishResult, 2)

	// Publish to KB-23 Decision Cards
	go func() {
		url := fmt.Sprintf("%s/api/v1/decision-cards", p.config.KB23URL)
		err := p.postWithRetry(ctx, url, body, 3, p.config.OutcomeRetryDelay)
		results <- publishResult{target: "KB-23", err: err}
	}()

	// Publish to KB-19 Protocol Orchestrator
	go func() {
		url := fmt.Sprintf("%s/api/v1/events", p.config.KB19URL)
		err := p.postWithRetry(ctx, url, body, 3, p.config.OutcomeRetryDelay)
		results <- publishResult{target: "KB-19", err: err}
	}()

	// Collect results from both targets
	var publishErrors []error
	for i := 0; i < 2; i++ {
		res := <-results
		if res.err != nil {
			p.log.Error("HPI_COMPLETE publish failed",
				zap.String("target", res.target),
				zap.String("session_id", event.SessionID.String()),
				zap.Error(res.err),
			)
			publishErrors = append(publishErrors, fmt.Errorf("%s: %w", res.target, res.err))
		} else {
			p.log.Info("HPI_COMPLETE published",
				zap.String("target", res.target),
				zap.String("session_id", event.SessionID.String()),
				zap.String("top_diagnosis", event.TopDiagnosis),
			)
		}
	}

	if len(publishErrors) > 0 {
		return fmt.Errorf("HPI_COMPLETE publish partial failure: %v", publishErrors)
	}

	return nil
}

// PublishSafetyAlert sends a SAFETY_ALERT event to KB-19 immediately when an
// IMMEDIATE-severity safety flag fires. Uses a fast-path retry with
// SafetyAlertRetryDelay (default 5s) for rapid delivery.
//
// This method does NOT wait for session completion. It is called inline from
// the answer processing loop whenever the SafetyEngine fires an IMMEDIATE flag.
func (p *OutcomePublisher) PublishSafetyAlert(ctx context.Context, event models.SafetyAlertEvent) error {
	event.EventType = models.EventSafetyAlert

	body, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal SAFETY_ALERT event: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/events", p.config.KB19URL)

	if err := p.postWithRetry(ctx, url, body, 3, p.config.SafetyAlertRetryDelay); err != nil {
		p.log.Error("SAFETY_ALERT publish failed after retries",
			zap.String("session_id", event.SessionID.String()),
			zap.String("flag_id", event.FlagID),
			zap.String("severity", event.Severity),
			zap.Error(err),
		)
		return fmt.Errorf("SAFETY_ALERT publish to KB-19 failed: %w", err)
	}

	p.log.Warn("SAFETY_ALERT published to KB-19",
		zap.String("session_id", event.SessionID.String()),
		zap.String("flag_id", event.FlagID),
		zap.String("severity", event.Severity),
		zap.String("action", event.RecommendedAction),
	)

	return nil
}

// postWithRetry performs an HTTP POST with retry logic. It retries up to
// maxRetries times with the specified delay between attempts. Context
// cancellation is respected between retries.
func (p *OutcomePublisher) postWithRetry(
	ctx context.Context,
	url string,
	body []byte,
	maxRetries int,
	retryDelay time.Duration,
) error {
	var lastErr error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("context cancelled before attempt %d: %w", attempt, err)
		}

		if err := p.doPost(ctx, url, body); err != nil {
			lastErr = err
			p.log.Warn("publish attempt failed",
				zap.String("url", url),
				zap.Int("attempt", attempt),
				zap.Int("max_retries", maxRetries),
				zap.Error(err),
			)

			if attempt < maxRetries {
				select {
				case <-time.After(retryDelay):
				case <-ctx.Done():
					return fmt.Errorf("context cancelled during retry wait: %w", ctx.Err())
				}
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("exhausted %d retries: %w", maxRetries, lastErr)
}

// doPost performs a single HTTP POST request.
func (p *OutcomePublisher) doPost(ctx context.Context, url string, body []byte) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := p.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("returned status %d: %s", resp.StatusCode, string(respBody))
	}

	return nil
}
