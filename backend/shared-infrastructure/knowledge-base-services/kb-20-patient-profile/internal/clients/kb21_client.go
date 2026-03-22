package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-patient-profile/internal/config"
)

// KB21Client communicates with the KB-21 Behavioral Intelligence service (port 8133).
// Used by PatientService to fetch the aggregate adherence score for inclusion
// in PatientProfileResponse, which V-MCU reads for gain factor modulation.
//
// All calls are best-effort — errors are logged but do not prevent the
// profile response from being returned (graceful degradation: AdherenceScore = nil).
type KB21Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB21Client creates a client for the KB-21 Behavioral Intelligence service.
func NewKB21Client(cfg config.KB21Config, logger *zap.Logger) *KB21Client {
	return &KB21Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// AdherenceResponse is the subset of KB-21's LoopTrustResponse that KB-20 needs.
type AdherenceResponse struct {
	PatientID         string  `json:"patient_id"`
	AdherenceScore30d float64 `json:"adherence_score_30d"`
	AdherenceScore7d  float64 `json:"adherence_score_7d"`
	AdherenceSource   string  `json:"adherence_source"`
}

// FetchAdherence calls KB-21's loop-trust endpoint and extracts the 30-day
// aggregate adherence score. Returns (nil, nil) on any failure so callers
// can treat missing adherence as "unavailable" rather than an error.
func (c *KB21Client) FetchAdherence(ctx context.Context, patientID string) (*float64, error) {
	reqURL := fmt.Sprintf("%s/v1/patient/%s/loop-trust", c.baseURL, patientID)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		c.logger.Warn("KB-21 adherence request creation failed",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return nil, nil
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-21 adherence request failed — AdherenceScore unavailable",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return nil, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Warn("KB-21 adherence returned non-200",
			zap.String("patient_id", patientID),
			zap.Int("status", resp.StatusCode))
		return nil, nil
	}

	var result AdherenceResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		c.logger.Warn("KB-21 adherence response decode failed",
			zap.String("patient_id", patientID),
			zap.Error(err))
		return nil, nil
	}

	score := result.AdherenceScore30d
	c.logger.Debug("KB-21 adherence score fetched",
		zap.String("patient_id", patientID),
		zap.Float64("score_30d", score),
		zap.String("source", result.AdherenceSource))

	return &score, nil
}

// HealthCheck verifies KB-21 is reachable.
func (c *KB21Client) HealthCheck() error {
	resp, err := c.httpClient.Get(c.baseURL + "/health")
	if err != nil {
		return fmt.Errorf("KB-21 health check failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-21 health check returned %d", resp.StatusCode)
	}
	return nil
}
