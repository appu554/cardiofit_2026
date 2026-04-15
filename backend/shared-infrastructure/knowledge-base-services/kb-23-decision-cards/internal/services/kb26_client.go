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

// KB26CGMLatestReport mirrors the KB-26 CGMPeriodReport row returned
// by GET /api/v1/kb26/cgm-latest/:patientId. Phase 7 P7-E Milestone 2:
// used by the inertia assembler to populate the glycaemic domain's
// CGM_TIR branch when a recent CGM period report exists.
type KB26CGMLatestReport struct {
	ID              uint      `json:"id"`
	PatientID       string    `json:"patient_id"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	CoveragePct     float64   `json:"coverage_pct"`
	SufficientData  bool      `json:"sufficient_data"`
	ConfidenceLevel string    `json:"confidence_level"`
	MeanGlucose     float64   `json:"mean_glucose"`
	SDGlucose       float64   `json:"sd_glucose"`
	CVPct           float64   `json:"cv_pct"`
	GlucoseStable   bool      `json:"glucose_stable"`
	TIRPct          float64   `json:"tir_pct"`
	TBRL1Pct        float64   `json:"tbr_l1_pct"`
	TBRL2Pct        float64   `json:"tbr_l2_pct"`
	TARL1Pct        float64   `json:"tar_l1_pct"`
	TARL2Pct        float64   `json:"tar_l2_pct"`
	GMI             float64   `json:"gmi"`
	GRI             float64   `json:"gri"`
	GRIZone         string    `json:"gri_zone"`
	HypoEvents      int       `json:"hypo_events"`
	CreatedAt       time.Time `json:"created_at"`
}

// kb26CGMLatestEnvelope wraps the KB-26 success envelope for the
// cgm-latest endpoint.
type kb26CGMLatestEnvelope struct {
	Success bool                `json:"success"`
	Data    KB26CGMLatestReport `json:"data"`
}

// FetchLatestCGMReport calls KB-26 GET /api/v1/kb26/cgm-latest/:patientId
// and returns the most recent CGMPeriodReport for the patient. Returns
// (nil, nil) on 404 so a patient without CGM data degrades gracefully
// — the inertia assembler treats a nil report as "no CGM branch" and
// falls back to the HbA1c target status path. Phase 7 P7-E Milestone 2.
func (c *KB26Client) FetchLatestCGMReport(ctx context.Context, patientID string) (*KB26CGMLatestReport, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/v1/kb26/cgm-latest/%s", c.cfg.KB26URL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-26 cgm-latest request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if c.metrics != nil {
		c.metrics.KB20FetchLatency.Observe(float64(time.Since(start).Milliseconds()))
	}
	if err != nil {
		return nil, fmt.Errorf("KB-26 cgm-latest fetch: %w", err)
	}
	defer resp.Body.Close()

	// 404 is an expected outcome for patients with no CGM data.
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("KB-26 cgm-latest returned non-200",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("KB-26 cgm-latest returned status %d", resp.StatusCode)
	}

	var env kb26CGMLatestEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode KB-26 cgm-latest response: %w", err)
	}
	return &env.Data, nil
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
