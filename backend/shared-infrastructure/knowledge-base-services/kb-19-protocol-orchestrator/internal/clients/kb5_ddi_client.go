// Package clients provides HTTP clients for KB-19 to communicate with upstream services.
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/sirupsen/logrus"

	"kb-19-protocol-orchestrator/internal/transaction"
)

// KB5DDIClient is the HTTP client for KB-5 Drug-Drug Interaction service.
// KB-5 provides comprehensive DDI checking with severity scoring and clinical recommendations.
type KB5DDIClient struct {
	baseURL    string
	httpClient *http.Client
	log        *logrus.Entry
}

// NewKB5DDIClient creates a new KB5DDIClient.
func NewKB5DDIClient(baseURL string, timeout time.Duration, log *logrus.Entry) *KB5DDIClient {
	return &KB5DDIClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		log: log.WithField("client", "kb5-ddi"),
	}
}

// DDICheckRequest is the request to check drug-drug interactions.
type DDICheckRequest struct {
	DrugCodes []string `json:"drug_codes"` // Drug name codes (e.g., "warfarin", "ketorolac")
}

// DDICheckResponse is the response from KB-5 interaction check.
type DDICheckResponse struct {
	Success bool `json:"success"`
	Data    struct {
		CheckedDrugs      []string        `json:"checked_drugs"`
		InteractionsFound []DDInteraction `json:"interactions_found"`
		Summary           DDISummary      `json:"summary"`
		Recommendations   []string        `json:"recommendations"`
		CheckTimestamp    time.Time       `json:"check_timestamp"`
	} `json:"data"`
	Meta struct {
		HasContraindications bool   `json:"has_contraindications"`
		HighestSeverity      string `json:"highest_severity"`
		TotalInteractions    int    `json:"total_interactions"`
	} `json:"meta"`
}

// DDInteraction represents a single drug-drug interaction.
type DDInteraction struct {
	InteractionID  string    `json:"interaction_id"`
	DrugA          DrugInfo  `json:"drug_a"`
	DrugB          DrugInfo  `json:"drug_b"`
	Severity       string    `json:"severity"` // contraindicated, major, moderate, minor
	InteractionType string   `json:"interaction_type"`
	EvidenceLevel  string    `json:"evidence_level"`
	Mechanism      string    `json:"mechanism"`
	ClinicalEffect string    `json:"clinical_effect"`
	ManagementStrategy string `json:"management_strategy"`
	DoseAdjustmentRequired bool `json:"dose_adjustment_required"`
	TimeToOnset    string    `json:"time_to_onset"`
}

// DrugInfo contains drug identification.
type DrugInfo struct {
	Code string `json:"code"`
	Name string `json:"name"`
}

// DDISummary provides aggregate DDI statistics.
type DDISummary struct {
	TotalInteractions     int            `json:"total_interactions"`
	SeverityCounts        map[string]int `json:"severity_counts"`
	HighestSeverity       string         `json:"highest_severity"`
	ContraindicatedPairs  int            `json:"contraindicated_pairs"`
	RequiredActions       []string       `json:"required_actions"`
	RiskScore             float64        `json:"risk_score"`
}

// CheckInteractions checks for drug-drug interactions via KB-5.
func (c *KB5DDIClient) CheckInteractions(ctx context.Context, drugCodes []string) (*DDICheckResponse, error) {
	c.log.WithField("drug_count", len(drugCodes)).Info("Checking DDI via KB-5")

	req := DDICheckRequest{DrugCodes: drugCodes}
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/api/v1/interactions/check", c.baseURL)
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(httpReq)
	if err != nil {
		c.log.WithError(err).Error("KB-5 request failed")
		return nil, fmt.Errorf("kb5 request: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		c.log.WithFields(logrus.Fields{
			"status": resp.StatusCode,
			"body":   string(respBody),
		}).Error("KB-5 returned error")
		return nil, fmt.Errorf("kb5 error: %s", string(respBody))
	}

	var result DDICheckResponse
	if err := json.Unmarshal(respBody, &result); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	c.log.WithFields(logrus.Fields{
		"interactions": result.Meta.TotalInteractions,
		"severity":     result.Meta.HighestSeverity,
	}).Info("KB-5 DDI check completed")

	return &result, nil
}

// HasSevereInteractions returns true if any contraindicated or major interactions found.
func (r *DDICheckResponse) HasSevereInteractions() bool {
	if r.Meta.HasContraindications {
		return true
	}
	if r.Meta.HighestSeverity == "contraindicated" || r.Meta.HighestSeverity == "major" {
		return true
	}
	return false
}

// GetBlockingInteractions returns interactions that should trigger hard blocks.
func (r *DDICheckResponse) GetBlockingInteractions() []DDInteraction {
	var blocking []DDInteraction
	for _, interaction := range r.Data.InteractionsFound {
		if interaction.Severity == "contraindicated" || interaction.Severity == "major" {
			blocking = append(blocking, interaction)
		}
	}
	return blocking
}

// Health checks if KB-5 is healthy.
func (c *KB5DDIClient) Health(ctx context.Context) error {
	url := fmt.Sprintf("%s/health", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("health check: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("kb5 unhealthy: status %d", resp.StatusCode)
	}
	return nil
}

// =============================================================================
// KB5DDIChecker ADAPTER
// Implements the KB5DDIChecker interface from transaction package
// =============================================================================

// KB5DDICheckerAdapter wraps KB5DDIClient to implement transaction.KB5DDIChecker interface.
type KB5DDICheckerAdapter struct {
	client *KB5DDIClient
}

// NewKB5DDICheckerAdapter creates an adapter that implements KB5DDIChecker.
func NewKB5DDICheckerAdapter(client *KB5DDIClient) *KB5DDICheckerAdapter {
	return &KB5DDICheckerAdapter{client: client}
}

// CheckInteractions implements KB5DDIChecker interface by calling the underlying client
// and converting the response to the expected transaction.KB5DDIResult type.
func (a *KB5DDICheckerAdapter) CheckInteractions(ctx context.Context, drugCodes []string) (*transaction.KB5DDIResult, error) {
	// Call the underlying client
	resp, err := a.client.CheckInteractions(ctx, drugCodes)
	if err != nil {
		return nil, err
	}

	// Convert DDICheckResponse to transaction.KB5DDIResult
	result := &transaction.KB5DDIResult{
		HighestSeverity:   resp.Meta.HighestSeverity,
		TotalInteractions: resp.Meta.TotalInteractions,
		InteractionsFound: make([]transaction.KB5Interaction, 0, len(resp.Data.InteractionsFound)),
	}

	// Convert each interaction
	for _, interaction := range resp.Data.InteractionsFound {
		result.InteractionsFound = append(result.InteractionsFound, transaction.KB5Interaction{
			InteractionID:      interaction.InteractionID,
			DrugACode:          interaction.DrugA.Code,
			DrugAName:          interaction.DrugA.Name,
			DrugBCode:          interaction.DrugB.Code,
			DrugBName:          interaction.DrugB.Name,
			Severity:           interaction.Severity,
			ClinicalEffect:     interaction.ClinicalEffect,
			ManagementStrategy: interaction.ManagementStrategy,
		})
	}

	return result, nil
}
