package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/models"
)

// TelemetryWriter sends per-question telemetry to KB-21 asynchronously.
// Writes are non-blocking: each call to WriteAsync launches a goroutine that
// retries up to TelemetryMaxRetries times at TelemetryRetryDelay intervals.
// Failures are logged but never propagated to the caller.
type TelemetryWriter struct {
	config *config.Config
	log    *zap.Logger
	client *http.Client
}

// NewTelemetryWriter creates a new TelemetryWriter for async KB-21 writes.
func NewTelemetryWriter(cfg *config.Config, log *zap.Logger) *TelemetryWriter {
	return &TelemetryWriter{
		config: cfg,
		log:    log,
		client: &http.Client{
			Timeout: cfg.KB21Timeout() + 5*time.Second, // generous timeout for write path
		},
	}
}

// WriteAsync sends question telemetry to KB-21 in a background goroutine.
// The call returns immediately and never blocks the answer processing path.
//
// Endpoint: POST KB-21 /api/v1/patient/{patient_id}/question-telemetry
// Retry policy: up to TelemetryMaxRetries (default 3) attempts with
// TelemetryRetryDelay (default 30s) between retries.
func (w *TelemetryWriter) WriteAsync(telemetry models.QuestionTelemetry) {
	go func() {
		defer func() {
			if r := recover(); r != nil {
				w.log.Error("telemetry writer recovered from panic",
					zap.String("session_id", telemetry.SessionID.String()),
					zap.String("question_id", telemetry.QuestionID),
					zap.Any("panic", r),
				)
			}
		}()

		url := fmt.Sprintf("%s/api/v1/patient/%s/question-telemetry",
			w.config.KB21URL, telemetry.PatientID.String())

		body, err := json.Marshal(telemetry)
		if err != nil {
			w.log.Error("failed to marshal telemetry payload",
				zap.String("session_id", telemetry.SessionID.String()),
				zap.String("question_id", telemetry.QuestionID),
				zap.Error(err),
			)
			return
		}

		maxRetries := w.config.TelemetryMaxRetries
		if maxRetries <= 0 {
			maxRetries = 3
		}

		for attempt := 1; attempt <= maxRetries; attempt++ {
			if err := w.postTelemetry(url, body); err != nil {
				w.log.Warn("telemetry write attempt failed",
					zap.String("session_id", telemetry.SessionID.String()),
					zap.String("question_id", telemetry.QuestionID),
					zap.Int("attempt", attempt),
					zap.Int("max_retries", maxRetries),
					zap.Error(err),
				)

				if attempt < maxRetries {
					time.Sleep(w.config.TelemetryRetryDelay)
				}
				continue
			}

			w.log.Debug("telemetry written to KB-21",
				zap.String("session_id", telemetry.SessionID.String()),
				zap.String("question_id", telemetry.QuestionID),
				zap.Int("attempt", attempt),
			)
			return
		}

		w.log.Error("telemetry write exhausted all retries",
			zap.String("session_id", telemetry.SessionID.String()),
			zap.String("question_id", telemetry.QuestionID),
			zap.Int("max_retries", maxRetries),
		)
	}()
}

// postTelemetry performs a single POST request to KB-21.
func (w *TelemetryWriter) postTelemetry(url string, body []byte) error {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	resp, err := w.client.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP POST: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("KB-21 returned status %d", resp.StatusCode)
	}

	return nil
}
