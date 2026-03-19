package clients

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type KB26Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewKB26Client(baseURL string, logger *zap.Logger) *KB26Client {
	return &KB26Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

type MRIDomainScore struct {
	Name   string  `json:"name"`
	Score  float64 `json:"score"`
	Scaled float64 `json:"scaled"`
}

type MRIDecomposition struct {
	Score     float64          `json:"score"`
	Category  string           `json:"category"`
	TopDriver string           `json:"top_driver"`
	Domains   []MRIDomainScore `json:"domains"`
}

func (c *KB26Client) GetDecomposition(ctx context.Context, patientID string) *MRIDecomposition {
	url := fmt.Sprintf("%s/api/v1/kb26/mri/%s/decomposition", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil
	}
	resp, err := c.httpClient.Do(req)
	if err != nil {
		c.logger.Warn("KB-26 MRI decomposition fetch failed", zap.Error(err))
		return nil
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil
	}
	var wrapper struct {
		Data MRIDecomposition `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&wrapper); err != nil {
		return nil
	}
	return &wrapper.Data
}
