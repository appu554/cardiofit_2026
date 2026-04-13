package services

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

// KB26BPContextClient calls KB-26's BP context classification endpoint.
type KB26BPContextClient struct {
	baseURL string
	client  *http.Client
	log     *zap.Logger
}

// NewKB26BPContextClient constructs a client.
func NewKB26BPContextClient(baseURL string, timeout time.Duration, log *zap.Logger) *KB26BPContextClient {
	return &KB26BPContextClient{
		baseURL: baseURL,
		client:  &http.Client{Timeout: timeout},
		log:     log,
	}
}

// kb26Envelope mirrors KB-26's standard sendSuccess wrapper:
//
//	{"success": true, "data": {...}, "metadata": {...}}
type kb26Envelope struct {
	Success bool                            `json:"success"`
	Data    *models.BPContextClassification `json:"data"`
}

// Classify requests a fresh BP context classification for the patient.
// Returns nil (no error) on 404 — caller decides how to handle missing data.
func (c *KB26BPContextClient) Classify(ctx context.Context, patientID string) (*models.BPContextClassification, error) {
	url := fmt.Sprintf("%s/api/v1/kb26/bp-context/%s", c.baseURL, patientID)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, nil)
	if err != nil {
		return nil, fmt.Errorf("build KB-26 request: %w", err)
	}
	req.Header.Set("Accept", "application/json")

	resp, err := c.client.Do(req)
	if err != nil {
		c.log.Warn("KB-26 unreachable", zap.String("url", url), zap.Error(err))
		return nil, fmt.Errorf("KB-26 fetch: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, nil
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-26 returned status %d: %s", resp.StatusCode, string(body))
	}

	var envelope kb26Envelope
	if err := json.NewDecoder(resp.Body).Decode(&envelope); err != nil {
		return nil, fmt.Errorf("decode KB-26 response: %w", err)
	}
	return envelope.Data, nil
}
