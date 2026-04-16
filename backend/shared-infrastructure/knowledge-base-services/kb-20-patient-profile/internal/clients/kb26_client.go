package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/pkg/resilience"
)

// KB26Client communicates with the KB-26 Metabolic Digital Twin service (port 8137).
// It is used by callers that populate TrajectoryInput to fetch MRI (Metabolic Risk Index)
// data before invoking the TrajectoryEngine. The engine itself is a pure computation
// engine with no network dependencies.
type KB26Client struct {
	baseURL    string
	httpClient *http.Client
	breaker    *resilience.CircuitBreaker // Phase 10 P10-C
	logger     *zap.Logger
}

// NewKB26Client creates a client for the KB-26 Metabolic Digital Twin service.
// Phase 10 P10-C: wraps the HTTP client with a circuit breaker so
// MRI + CGM status calls to KB-26 have retry + fast-fail + recovery.
func NewKB26Client(baseURL string, logger *zap.Logger) *KB26Client {
	httpClient := &http.Client{Timeout: 5 * time.Second}
	cbCfg := resilience.DefaultConfig("kb26-from-kb20")
	cbCfg.MaxRetries = 2
	cbCfg.ResetTimeout = 20 * time.Second
	return &KB26Client{
		baseURL:    baseURL,
		httpClient: httpClient,
		breaker:    resilience.NewCircuitBreaker(httpClient, cbCfg),
		logger:     logger,
	}
}

// SetOnStateChange sets the circuit breaker's state-change callback
// after construction. Used by main.go to wire Prometheus metrics
// into the breaker without the clients package needing to import
// the metrics package. Phase 10 P10-D.
func (c *KB26Client) SetOnStateChange(fn func(name string, from, to resilience.State)) {
	if c.breaker != nil {
		c.breaker.SetOnStateChange(fn)
	}
}

// MRISnapshot is a single Metabolic Risk Index reading from KB-26.
type MRISnapshot struct {
	Score      float64 `json:"score"`
	Category   string  `json:"category"`
	Trend      string  `json:"trend"`
	TopDriver  string  `json:"top_driver"`
	ComputedAt string  `json:"computed_at"`
}

// GetCurrentMRI fetches the most recent MRI snapshot for the given patient.
// Returns nil if the patient has no MRI data or the call fails (best-effort).
func (c *KB26Client) GetCurrentMRI(ctx context.Context, patientID string) *MRISnapshot {
	url := fmt.Sprintf("%s/api/v1/kb26/mri/%s", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := c.breaker.Do(req)
	if err != nil {
		c.logger.Warn("KB-26 MRI fetch failed", zap.String("patient_id", patientID), zap.Error(err))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var wrapper struct {
		Data MRISnapshot `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil
	}
	return &wrapper.Data
}

// CGMPeriodReportSnapshot mirrors the fields of KB-26's
// models.CGMPeriodReport that the summary-context builder needs to
// populate its CGM status fields. Only the fields downstream KB-23
// card generation reads are carried — TIR, GRI zone, report date,
// and a HasCGM derived flag. Phase 8 P8-3.
type CGMPeriodReportSnapshot struct {
	PatientID       string    `json:"patient_id"`
	PeriodStart     time.Time `json:"period_start"`
	PeriodEnd       time.Time `json:"period_end"`
	TIRPct          float64   `json:"tir_pct"`
	MeanGlucose     float64   `json:"mean_glucose"`
	GRIZone         string    `json:"gri_zone"`
	ConfidenceLevel string    `json:"confidence_level"`
}

// GetLatestCGMStatus fetches the most recent CGM period report for
// a patient from KB-26's P7-E Milestone 2 cgm-latest endpoint.
//
// Returns (nil, nil) on 404 — a patient with no CGM data is not an
// error condition, it's the normal case for patients who aren't on
// a CGM device. The summary-context builder treats nil as "no CGM
// status, HasCGM=false" and falls back to HbA1c-based glycaemic
// evaluation.
//
// Returns (nil, err) on network failures, 5xx responses, and decode
// errors so the caller can log them at debug level without aborting
// the summary-context assembly. The pattern matches GetCurrentMRI's
// best-effort behaviour. Phase 8 P8-3.
func (c *KB26Client) GetLatestCGMStatus(ctx context.Context, patientID string) (*CGMPeriodReportSnapshot, error) {
	url := fmt.Sprintf("%s/api/v1/kb26/cgm-latest/%s", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-26 cgm-latest request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.breaker.Do(req)
	if err != nil {
		c.logger.Debug("KB-26 cgm-latest fetch failed",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return nil, fmt.Errorf("KB-26 cgm-latest fetch: %w", err)
	}
	defer resp.Body.Close()

	// 404 is expected for patients with no CGM data — degrade cleanly.
	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-26 cgm-latest returned status %d", resp.StatusCode)
	}

	var wrapper struct {
		Success bool                    `json:"success"`
		Data    CGMPeriodReportSnapshot `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, fmt.Errorf("decode KB-26 cgm-latest response: %w", err)
	}
	return &wrapper.Data, nil
}

// GetMRIHistory fetches the most recent `limit` MRI snapshots for the given patient
// in reverse-chronological order. Returns nil if the call fails.
func (c *KB26Client) GetMRIHistory(ctx context.Context, patientID string, limit int) []MRISnapshot {
	url := fmt.Sprintf("%s/api/v1/kb26/mri/%s/history?limit=%d", c.baseURL, patientID, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := c.breaker.Do(req)
	if err != nil {
		c.logger.Warn("KB-26 MRI history fetch failed", zap.String("patient_id", patientID), zap.Error(err))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var wrapper struct {
		Data []MRISnapshot `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil
	}
	return wrapper.Data
}
