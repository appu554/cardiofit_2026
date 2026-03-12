package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/metrics"
)

type KB20Client struct {
	cfg     *config.Config
	metrics *metrics.Collector
	log     *zap.Logger
	client  *http.Client
}

func NewKB20Client(cfg *config.Config, m *metrics.Collector, log *zap.Logger) *KB20Client {
	return &KB20Client{
		cfg:     cfg,
		metrics: m,
		log:     log,
		client: &http.Client{
			Timeout: cfg.KB20Timeout(),
		},
	}
}

// FetchSummaryContext calls KB-20 GET /patient/:id/summary-context.
// Returns PatientContext or nil if KB-20 is unavailable (graceful degradation).
func (c *KB20Client) FetchSummaryContext(ctx context.Context, patientID string) (*PatientContext, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/patient/%s/summary-context", c.cfg.KB20URL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-20 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	c.metrics.KB20FetchLatency.Observe(float64(time.Since(start).Milliseconds()))

	if err != nil {
		c.log.Warn("KB-20 unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-20 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("KB-20 returned non-200",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)),
		)
		return nil, fmt.Errorf("KB-20 returned status %d", resp.StatusCode)
	}

	var result PatientContext
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode KB-20 response: %w", err)
	}

	result.PatientID = patientID
	return &result, nil
}
