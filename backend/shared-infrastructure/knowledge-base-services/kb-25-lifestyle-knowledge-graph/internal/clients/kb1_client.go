package clients

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type DrugEffect struct {
	DrugClass  string  `json:"drug_class"`
	TargetVar  string  `json:"target_variable"`
	EffectSize float64 `json:"effect_size"`
	EffectUnit string  `json:"effect_unit"`
	Grade      string  `json:"evidence_grade"`
}

type KB1Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewKB1Client(baseURL string, logger *zap.Logger) *KB1Client {
	return &KB1Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

func (c *KB1Client) GetDrugEffect(drugClass, targetVar string) (*DrugEffect, error) {
	url := fmt.Sprintf("%s/api/v1/drug-rules/%s/effect/%s", c.baseURL, drugClass, targetVar)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("KB-1 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-1 returned status %d", resp.StatusCode)
	}

	var result struct {
		Data DrugEffect `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("KB-1 decode failed: %w", err)
	}
	return &result.Data, nil
}
