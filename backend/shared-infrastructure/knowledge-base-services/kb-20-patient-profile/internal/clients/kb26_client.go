package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB26Client communicates with the KB-26 Metabolic Digital Twin service (port 8137).
// It is used by callers that populate TrajectoryInput to fetch MRI (Metabolic Risk Index)
// data before invoking the TrajectoryEngine. The engine itself is a pure computation
// engine with no network dependencies.
type KB26Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB26Client creates a client for the KB-26 Metabolic Digital Twin service.
func NewKB26Client(baseURL string, logger *zap.Logger) *KB26Client {
	return &KB26Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
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
	resp, err := c.httpClient.Do(req)
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

// GetMRIHistory fetches the most recent `limit` MRI snapshots for the given patient
// in reverse-chronological order. Returns nil if the call fails.
func (c *KB26Client) GetMRIHistory(ctx context.Context, patientID string, limit int) []MRISnapshot {
	url := fmt.Sprintf("%s/api/v1/kb26/mri/%s/history?limit=%d", c.baseURL, patientID, limit)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := c.httpClient.Do(req)
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
