package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type AdherenceData struct {
	PatientID      string  `json:"patient_id"`
	AdherenceScore float64 `json:"adherence_score"`
	Source         string  `json:"source"`
}

type KB21Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewKB21Client(baseURL string, logger *zap.Logger) *KB21Client {
	return &KB21Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

func (c *KB21Client) GetAdherence(patientID string) (*AdherenceData, error) {
	url := fmt.Sprintf("%s/api/v1/patient/%s/adherence", c.baseURL, patientID)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("KB-21 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-21 returned status %d", resp.StatusCode)
	}

	var result struct {
		Data AdherenceData `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("KB-21 decode failed: %w", err)
	}
	return &result.Data, nil
}
