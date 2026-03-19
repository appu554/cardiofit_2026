package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// KB22Client fetches monitoring node classifications from KB-22 HPI Engine.
type KB22Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB22Client creates a client for KB-22 signal queries.
func NewKB22Client(baseURL string, timeout time.Duration, logger *zap.Logger) *KB22Client {
	return &KB22Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: timeout},
		logger:     logger,
	}
}

// SignalResult represents a KB-22 monitoring node signal evaluation.
type SignalResult struct {
	NodeID         string  `json:"node_id"`
	Category       string  `json:"category"`
	Severity       string  `json:"severity"`
	Classification string  `json:"classification"`
	Value          float64 `json:"value"`
	EvaluatedAt    string  `json:"evaluated_at"`
}

// GetLatestSignal fetches the most recent signal for a patient and monitoring node.
// Returns nil if KB-22 is unreachable or no signal exists (graceful degradation).
func (c *KB22Client) GetLatestSignal(ctx context.Context, patientID, nodeID string) (*SignalResult, error) {
	url := fmt.Sprintf("%s/api/v1/signals/patients/%s/signals/%s", c.baseURL, patientID, nodeID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-22 signal fetch failed — using local fallback",
			zap.String("node_id", nodeID), zap.Error(err))
		return nil, nil // graceful degradation
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		c.logger.Debug("KB-22 signal not found",
			zap.String("node_id", nodeID), zap.Int("status", resp.StatusCode))
		return nil, nil
	}

	var wrapper struct {
		Data []SignalResult `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil, err
	}
	if len(wrapper.Data) == 0 {
		return nil, nil
	}
	return &wrapper.Data[0], nil
}

// GetBPDipping fetches the PM-04 BP dipping classification for a patient.
// Returns empty string if unavailable (caller should fall back to local computation).
func (c *KB22Client) GetBPDipping(ctx context.Context, patientID string) string {
	signal, err := c.GetLatestSignal(ctx, patientID, "PM-04")
	if err != nil || signal == nil {
		return "" // caller falls back to local ClassifyBPDipping
	}
	return signal.Classification
}

// GetSleepQuality fetches the PM-07 sleep quality classification for a patient.
// Returns -1 if unavailable (caller should fall back to local SleepQuality field).
func (c *KB22Client) GetSleepQuality(ctx context.Context, patientID string) float64 {
	signal, err := c.GetLatestSignal(ctx, patientID, "PM-07")
	if err != nil || signal == nil {
		return -1 // sentinel: caller uses local value
	}
	return signal.Value
}
