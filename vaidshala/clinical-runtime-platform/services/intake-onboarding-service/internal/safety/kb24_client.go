package safety

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// IntakeTriggerDef is the rule definition fetched from KB-24.
// Mirrors kb-24's SafetyTriggerDef with the fields the intake service needs.
type IntakeTriggerDef struct {
	ID       string `json:"id"`
	RuleType string `json:"rule_type"` // HARD_STOP or SOFT_FLAG
	Condition string `json:"condition"`
	Severity string `json:"severity"`
	Action   string `json:"action"`
}

// KB24Client fetches safety rule definitions from the KB-24 Safety Constraint Engine.
type KB24Client struct {
	baseURL    string
	httpClient *http.Client
	logger     *zap.Logger
}

// NewKB24Client creates a client pointing at the KB-24 service.
func NewKB24Client(baseURL string, logger *zap.Logger) *KB24Client {
	return &KB24Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 5 * time.Second,
		},
		logger: logger,
	}
}

// intakeTriggersResponse is the JSON envelope from GET /api/v1/intake-triggers.
type intakeTriggersResponse struct {
	Rules []IntakeTriggerDef `json:"rules"`
	Count int                `json:"count"`
}

// FetchIntakeTriggers retrieves all intake safety rules from KB-24.
func (c *KB24Client) FetchIntakeTriggers() ([]IntakeTriggerDef, error) {
	url := c.baseURL + "/api/v1/intake-triggers"
	resp, err := c.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("KB-24 request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-24 returned %d: %s", resp.StatusCode, string(body))
	}

	var result intakeTriggersResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("KB-24 response decode failed: %w", err)
	}

	c.logger.Info("fetched intake triggers from KB-24",
		zap.Int("count", result.Count),
		zap.String("url", url),
	)
	return result.Rules, nil
}
