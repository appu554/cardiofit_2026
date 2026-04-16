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
	"kb-23-decision-cards/pkg/resilience"
)

// KB20RenalStatus mirrors the KB-20 GET /patient/:id/renal-status response
// payload. Phase 7 P7-C consumes this struct in the RenalAnticipatoryOrchestrator
// to feed FindApproachingThresholds + DetectStaleEGFR with a single fetch.
type KB20RenalStatus struct {
	PatientID         string             `json:"patient_id"`
	EGFR              float64            `json:"egfr"`
	EGFRSlope         float64            `json:"egfr_slope"`
	EGFRMeasuredAt    time.Time          `json:"egfr_measured_at"`
	EGFRDataPoints    int                `json:"egfr_data_points"`
	Potassium         *float64           `json:"potassium,omitempty"`
	ACR               *float64           `json:"acr,omitempty"`
	CKDStage          string             `json:"ckd_stage"`
	IsRapidDecliner   bool               `json:"is_rapid_decliner"`
	ActiveMedications []KB20MedSummary   `json:"active_medications"`
}

// KB20MedSummary is the lightweight medication reference returned in
// KB-20's renal status response. Phase 7 P7-C.
type KB20MedSummary struct {
	DrugName  string `json:"drug_name"`
	DrugClass string `json:"drug_class"`
	DoseMg    string `json:"dose_mg"`
	IsActive  bool   `json:"is_active"`
}

// kb20EnvelopeRenalStatus wraps the KB-20 renal-status response under the
// standard {"success": true, "data": ...} envelope.
type kb20EnvelopeRenalStatus struct {
	Success bool            `json:"success"`
	Data    KB20RenalStatus `json:"data"`
}

// kb20EnvelopeRenalActive wraps the KB-20 renal-active list response.
type kb20EnvelopeRenalActive struct {
	Success bool `json:"success"`
	Data    []struct {
		PatientID string `json:"patient_id"`
	} `json:"data"`
}

type KB20Client struct {
	cfg     *config.Config
	metrics *metrics.Collector
	log     *zap.Logger
	client  *http.Client
	breaker *resilience.CircuitBreaker // Phase 10 P10-B
}

func NewKB20Client(cfg *config.Config, m *metrics.Collector, log *zap.Logger) *KB20Client {
	httpClient := &http.Client{Timeout: cfg.KB20Timeout()}
	// Phase 10 P10-B: circuit breaker wraps the HTTP client with
	// three-state management + exponential backoff + retry. KB-20
	// calls are frequent and latency-sensitive (summary-context,
	// renal-status, intervention-timeline), so MaxRetries=2 and
	// ResetTimeout=15s keep the recovery fast.
	cbCfg := resilience.DefaultConfig("kb20")
	cbCfg.MaxRetries = 2
	cbCfg.ResetTimeout = 15 * time.Second
	return &KB20Client{
		cfg:     cfg,
		metrics: m,
		log:     log,
		client:  httpClient,
		breaker: resilience.NewCircuitBreaker(httpClient, cbCfg),
	}
}

// FetchSummaryContext calls KB-20 GET /api/v1/patient/:id/summary-context.
// Returns PatientContext or nil if KB-20 is unavailable (graceful degradation).
//
// Phase 8 P8-4 bug fix: the URL template was missing the /api/v1/
// prefix that every other v1 route on KB-20 uses. The P8-1 integration
// test masked this because its mirror handler used the same wrong
// path. The tightened P8-4 integration test pins the real route.
func (c *KB20Client) FetchSummaryContext(ctx context.Context, patientID string) (*PatientContext, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/v1/patient/%s/summary-context", c.cfg.KB20URL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-20 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.breaker.Do(req)
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

	// Phase 8 P8-1: KB-20 wraps responses in the standard
	// {"success": true, "data": ...} envelope, matching every other
	// handler in the service (renal-status, intervention-timeline,
	// modifier registry, etc.). The P7-era FetchSummaryContext client
	// was decoding the raw body into PatientContext directly — a
	// wire mismatch that would have produced an empty struct even
	// if the KB-20 endpoint had existed. Both halves of the bug are
	// now fixed: KB-20 has the handler, and KB-23 unwraps the
	// envelope. The integration test in this package pins the
	// contract so this mismatch cannot regress.
	var envelope struct {
		Success bool           `json:"success"`
		Data    PatientContext `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode KB-20 response: %w", err)
	}

	result := envelope.Data
	result.PatientID = patientID
	return &result, nil
}

// FetchRenalStatus calls KB-20 GET /api/v1/patient/:id/renal-status and
// returns the full renal snapshot (eGFR, slope, measured-at, active
// medications, CKD stage). Used by P7-C's RenalAnticipatoryOrchestrator
// to feed FindApproachingThresholds + DetectStaleEGFR in one round trip.
func (c *KB20Client) FetchRenalStatus(ctx context.Context, patientID string) (*KB20RenalStatus, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/v1/patient/%s/renal-status", c.cfg.KB20URL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-20 renal-status request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.breaker.Do(req)
	if c.metrics != nil {
		c.metrics.KB20FetchLatency.Observe(float64(time.Since(start).Milliseconds()))
	}
	if err != nil {
		return nil, fmt.Errorf("KB-20 renal-status fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("KB-20 renal-status returned non-200",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("KB-20 renal-status returned status %d", resp.StatusCode)
	}

	var env kb20EnvelopeRenalStatus
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode KB-20 renal-status response: %w", err)
	}
	return &env.Data, nil
}

// KB20InterventionTimeline mirrors the KB-20 InterventionTimelineResult
// response payload. Phase 7 P7-D.
type KB20InterventionTimeline struct {
	PatientID                string                          `json:"PatientID"`
	ByDomain                 map[string]KB20LatestDomainAction `json:"ByDomain"`
	AnyChangeInLast12Weeks   bool                            `json:"AnyChangeInLast12Weeks"`
	TotalActiveInterventions int                             `json:"TotalActiveInterventions"`
}

// KB20LatestDomainAction mirrors the per-domain latest action returned
// by KB-20's intervention timeline service.
type KB20LatestDomainAction struct {
	InterventionID   string    `json:"InterventionID"`
	InterventionType string    `json:"InterventionType"`
	DrugClass        string    `json:"DrugClass"`
	DrugName         string    `json:"DrugName"`
	DoseMg           float64   `json:"DoseMg"`
	ActionDate       time.Time `json:"ActionDate"`
	DaysSince        int       `json:"DaysSince"`
}

// kb20EnvelopeInterventionTimeline wraps the KB-20 intervention timeline
// response under the standard success envelope.
type kb20EnvelopeInterventionTimeline struct {
	Success bool                     `json:"success"`
	Data    KB20InterventionTimeline `json:"data"`
}

// FetchInterventionTimeline calls KB-20 GET /api/v1/patient/:id/intervention-timeline
// and returns the latest clinical action per therapeutic-inertia domain.
// Used by P7-D's InertiaInputAssembler to populate LastIntervention on
// each DomainInertiaInput. Phase 7 P7-D.
func (c *KB20Client) FetchInterventionTimeline(ctx context.Context, patientID string) (*KB20InterventionTimeline, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/v1/patient/%s/intervention-timeline", c.cfg.KB20URL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-20 intervention-timeline request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.breaker.Do(req)
	if c.metrics != nil {
		c.metrics.KB20FetchLatency.Observe(float64(time.Since(start).Milliseconds()))
	}
	if err != nil {
		return nil, fmt.Errorf("KB-20 intervention-timeline fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("KB-20 intervention-timeline returned non-200",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("KB-20 intervention-timeline returned status %d", resp.StatusCode)
	}

	var env kb20EnvelopeInterventionTimeline
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode KB-20 intervention-timeline response: %w", err)
	}
	return &env.Data, nil
}

// FetchRenalActivePatientIDs calls KB-20 GET /api/v1/patients/renal-active
// and returns the patient IDs of everyone on at least one renal-sensitive
// medication. Phase 7 P7-C: the population the monthly anticipatory batch
// iterates over.
func (c *KB20Client) FetchRenalActivePatientIDs(ctx context.Context) ([]string, error) {
	start := time.Now()
	url := fmt.Sprintf("%s/api/v1/patients/renal-active", c.cfg.KB20URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("create KB-20 renal-active request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.breaker.Do(req)
	if c.metrics != nil {
		c.metrics.KB20FetchLatency.Observe(float64(time.Since(start).Milliseconds()))
	}
	if err != nil {
		return nil, fmt.Errorf("KB-20 renal-active fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("KB-20 renal-active returned non-200",
			zap.Int("status", resp.StatusCode),
			zap.String("body", string(body)))
		return nil, fmt.Errorf("KB-20 renal-active returned status %d", resp.StatusCode)
	}

	var env kb20EnvelopeRenalActive
	if err := json.NewDecoder(resp.Body).Decode(&env); err != nil {
		return nil, fmt.Errorf("decode KB-20 renal-active response: %w", err)
	}
	ids := make([]string, 0, len(env.Data))
	for _, entry := range env.Data {
		if entry.PatientID != "" {
			ids = append(ids, entry.PatientID)
		}
	}
	return ids, nil
}
