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

	"kb-23-decision-cards/internal/config"
	"kb-23-decision-cards/internal/metrics"
)

// KB26Client is a narrow HTTP client used by the Phase 7 P7-D inertia
// input assembler. It calls KB-26's stateless target-status compute
// endpoint to convert raw patient measurements into per-domain
// DomainTargetStatusResult verdicts.
type KB26Client struct {
	cfg     *config.Config
	metrics *metrics.Collector
	log     *zap.Logger
	client  *http.Client
}

// NewKB26Client constructs a KB26Client with the configured timeout.
func NewKB26Client(cfg *config.Config, m *metrics.Collector, log *zap.Logger) *KB26Client {
	return &KB26Client{
		cfg:     cfg,
		metrics: m,
		log:     log,
		client:  &http.Client{Timeout: cfg.KB26Timeout()},
	}
}

// KB26TargetStatusRequest mirrors the KB-26 handler's POST body.
type KB26TargetStatusRequest struct {
	HbA1c       *float64 `json:"hba1c,omitempty"`
	HbA1cTarget float64  `json:"hba1c_target,omitempty"`
	MeanSBP7d   *float64 `json:"mean_sbp_7d,omitempty"`
	SBPTarget   float64  `json:"sbp_target,omitempty"`
	EGFR        *float64 `json:"egfr,omitempty"`
	EGFRTarget  float64  `json:"egfr_target,omitempty"`
}

// KB26DomainTargetStatus mirrors the KB-26 DomainTargetStatusResult
// response payload. Duplicated as a separate type to avoid an import
// cycle — KB-23 does not depend on KB-26 as a module.
type KB26DomainTargetStatus struct {
	Domain              string     `json:"Domain"`
	AtTarget            bool       `json:"AtTarget"`
	CurrentValue        float64    `json:"CurrentValue"`
	TargetValue         float64    `json:"TargetValue"`
	FirstUncontrolledAt *time.Time `json:"FirstUncontrolledAt,omitempty"`
	DaysUncontrolled    int        `json:"DaysUncontrolled"`
	ConsecutiveReadings int        `json:"ConsecutiveReadings"`
	DataSource          string     `json:"DataSource"`
	Confidence          string     `json:"Confidence"`
}

// KB26TargetStatusResponse wraps the three domain verdicts returned
// by KB-26.
type KB26TargetStatusResponse struct {
	Glycaemic   KB26DomainTargetStatus `json:"glycaemic"`
	Hemodynamic KB26DomainTargetStatus `json:"hemodynamic"`
	Renal       KB26DomainTargetStatus `json:"renal"`
}

// kb26TargetStatusEnvelope matches the standard {"success": bool, "data": ...}
// envelope returned by KB-26's sendSuccess helper.
type kb26TargetStatusEnvelope struct {
	Success bool                     `json:"success"`
	Data    KB26TargetStatusResponse `json:"data"`
}

// FetchTargetStatus POSTs the patient's raw measurements to KB-26 and
// returns the per-domain target-status verdicts. Phase 7 P7-D.
func (c *KB26Client) FetchTargetStatus(ctx context.Context, patientID string, req KB26TargetStatusRequest) (*KB26TargetStatusResponse, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/v1/kb26/target-status/%s", c.cfg.KB26URL, patientID)

	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal target-status request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create KB-26 target-status request: %w", err)
	}
	httpReq.Header.Set("Accept", "application/json")
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.client.Do(httpReq)
	if c.metrics != nil {
		c.metrics.KB20FetchLatency.Observe(float64(time.Since(start).Milliseconds()))
	}
	if err != nil {
		return nil, fmt.Errorf("KB-26 target-status fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		c.log.Warn("KB-26 target-status returned non-200",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(respBody)))
		return nil, fmt.Errorf("KB-26 target-status returned status %d", resp.StatusCode)
	}

	var env kb26TargetStatusEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode KB-26 target-status response: %w", err)
	}
	return &env.Data, nil
}
