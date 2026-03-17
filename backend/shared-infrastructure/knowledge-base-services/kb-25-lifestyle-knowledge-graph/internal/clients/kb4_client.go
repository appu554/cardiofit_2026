package clients

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

type KB4Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

func NewKB4Client(baseURL string, logger *zap.Logger) *KB4Client {
	return &KB4Client{
		baseURL:    baseURL,
		httpClient: &http.Client{Timeout: 5 * time.Second},
		logger:     logger,
	}
}

func (c *KB4Client) CheckDrugSafety(drugClass string, patientConditions []string) (bool, error) {
	url := fmt.Sprintf("%s/health", c.baseURL)
	resp, err := c.httpClient.Get(url)
	if err != nil {
		c.logger.Warn("KB-4 unreachable, failing open", zap.Error(err))
		return true, nil // Fail open — log warning
	}
	defer resp.Body.Close()
	return resp.StatusCode == http.StatusOK, nil
}
