package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	dtModels "kb-26-metabolic-digital-twin/pkg/trajectory"
)

// KB26TrajectoryClient calls KB-26's domain trajectory endpoint.
type KB26TrajectoryClient struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB26TrajectoryClient constructs a trajectory client.
func NewKB26TrajectoryClient(baseURL string, timeout time.Duration, log *zap.Logger) *KB26TrajectoryClient {
	return &KB26TrajectoryClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// kb26TrajectoryEnvelope mirrors KB-26's standard sendSuccess wrapper for the
// domain trajectory endpoint.
//
//	{"success": true, "data": {...}, "metadata": {...}}
//
// When insufficient data is available KB-26 returns success=true but the data
// object contains {"status":"INSUFFICIENT_DATA",...} rather than a full
// DecomposedTrajectory. We detect this via the presence of the status key.
type kb26TrajectoryEnvelope struct {
	Success bool                            `json:"success"`
	Data    json.RawMessage                 `json:"data"`
}

// insufficientDataMarker is used to peek at the data object before full
// deserialisation.
type insufficientDataMarker struct {
	Status string `json:"status"`
}

// GetTrajectory fetches the decomposed MHRI trajectory for the given patient.
// Returns (nil, nil) on 404 or when KB-26 reports INSUFFICIENT_DATA — the
// caller should treat both as "not enough data yet, skip trajectory cards".
func (c *KB26TrajectoryClient) GetTrajectory(ctx context.Context, patientID string) (*dtModels.DecomposedTrajectory, error) {
	url := fmt.Sprintf("%s/api/v1/kb26/mri/%s/domain-trajectory", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-26 trajectory request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-26 trajectory endpoint unreachable",
			zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-26 trajectory fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-26 trajectory returned status %d: %s", resp.StatusCode, string(body))
	}

	var envelope kb26TrajectoryEnvelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode KB-26 trajectory response: %w", err)
	}

	// Peek at the data object: if it has {"status":"INSUFFICIENT_DATA"} there
	// are fewer than 2 MRI scores and we cannot compute a trajectory.
	var marker insufficientDataMarker
	if err := json.Unmarshal(envelope.Data, &marker); err == nil && marker.Status == "INSUFFICIENT_DATA" {
		return nil, nil
	}

	var traj dtModels.DecomposedTrajectory
	if err := json.Unmarshal(envelope.Data, &traj); err != nil {
		return nil, fmt.Errorf("unmarshal KB-26 DecomposedTrajectory: %w", err)
	}
	return &traj, nil
}
