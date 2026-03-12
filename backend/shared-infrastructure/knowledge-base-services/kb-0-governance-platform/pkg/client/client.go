// Package client provides a Go client for the KB-0 Governance Platform API.
// Other KBs import this package to integrate with the shared governance infrastructure.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"kb-0-governance-platform/internal/models"
)

// =============================================================================
// CLIENT
// =============================================================================

// Client provides access to KB-0 governance services.
type Client struct {
	baseURL    string
	httpClient *http.Client
	apiKey     string
	kb         models.KB
}

// Config holds client configuration.
type Config struct {
	BaseURL    string
	APIKey     string
	KB         models.KB
	Timeout    time.Duration
}

// NewClient creates a new KB-0 governance client.
func NewClient(cfg Config) *Client {
	timeout := cfg.Timeout
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	return &Client{
		baseURL: cfg.BaseURL,
		httpClient: &http.Client{
			Timeout: timeout,
		},
		apiKey: cfg.APIKey,
		kb:     cfg.KB,
	}
}

// =============================================================================
// ITEM OPERATIONS
// =============================================================================

// CreateItem creates a new knowledge item in KB-0.
func (c *Client) CreateItem(ctx context.Context, item *models.KnowledgeItem) error {
	return c.post(ctx, "/api/v1/items", item, nil)
}

// GetItem retrieves a knowledge item by ID.
func (c *Client) GetItem(ctx context.Context, itemID string) (*models.KnowledgeItem, error) {
	var item models.KnowledgeItem
	err := c.get(ctx, fmt.Sprintf("/api/v1/items/%s", itemID), &item)
	if err != nil {
		return nil, err
	}
	return &item, nil
}

// UpdateItem updates a knowledge item.
func (c *Client) UpdateItem(ctx context.Context, item *models.KnowledgeItem) error {
	return c.put(ctx, fmt.Sprintf("/api/v1/items/%s", item.ID), item, nil)
}

// =============================================================================
// WORKFLOW OPERATIONS
// =============================================================================

// SubmitReview submits a review for a knowledge item.
func (c *Client) SubmitReview(ctx context.Context, req *ReviewRequest) (*TransitionResult, error) {
	var result TransitionResult
	err := c.post(ctx, "/api/v1/workflow/review", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Approve approves a knowledge item.
func (c *Client) Approve(ctx context.Context, req *ApprovalRequest) (*TransitionResult, error) {
	var result TransitionResult
	err := c.post(ctx, "/api/v1/workflow/approve", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Reject rejects a knowledge item.
func (c *Client) Reject(ctx context.Context, req *ApprovalRequest) (*TransitionResult, error) {
	var result TransitionResult
	err := c.post(ctx, "/api/v1/workflow/reject", req, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// Activate activates an approved item.
func (c *Client) Activate(ctx context.Context, itemID string) (*TransitionResult, error) {
	var result TransitionResult
	err := c.post(ctx, fmt.Sprintf("/api/v1/workflow/activate/%s", itemID), nil, &result)
	if err != nil {
		return nil, err
	}
	return &result, nil
}

// =============================================================================
// QUERY OPERATIONS
// =============================================================================

// GetPendingReviews returns items pending review.
func (c *Client) GetPendingReviews(ctx context.Context) ([]*models.KnowledgeItem, error) {
	var items []*models.KnowledgeItem
	err := c.get(ctx, fmt.Sprintf("/api/v1/items/pending-review?kb=%s", c.kb), &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetPendingApprovals returns items pending approval.
func (c *Client) GetPendingApprovals(ctx context.Context) ([]*models.KnowledgeItem, error) {
	var items []*models.KnowledgeItem
	err := c.get(ctx, fmt.Sprintf("/api/v1/items/pending-approval?kb=%s", c.kb), &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetActiveItems returns all active items.
func (c *Client) GetActiveItems(ctx context.Context) ([]*models.KnowledgeItem, error) {
	var items []*models.KnowledgeItem
	err := c.get(ctx, fmt.Sprintf("/api/v1/items/active?kb=%s", c.kb), &items)
	if err != nil {
		return nil, err
	}
	return items, nil
}

// GetMetrics returns governance metrics for this KB.
func (c *Client) GetMetrics(ctx context.Context) (*Metrics, error) {
	var metrics Metrics
	err := c.get(ctx, fmt.Sprintf("/api/v1/metrics/%s", c.kb), &metrics)
	if err != nil {
		return nil, err
	}
	return &metrics, nil
}

// =============================================================================
// AUDIT OPERATIONS
// =============================================================================

// GetAuditTrail returns the audit trail for an item.
func (c *Client) GetAuditTrail(ctx context.Context, itemID string) ([]*models.AuditEntry, error) {
	var entries []*models.AuditEntry
	err := c.get(ctx, fmt.Sprintf("/api/v1/audit/%s", itemID), &entries)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

// =============================================================================
// REQUEST/RESPONSE TYPES
// =============================================================================

// ReviewRequest contains review submission details.
type ReviewRequest struct {
	ItemID       string                  `json:"item_id"`
	ReviewerID   string                  `json:"reviewer_id"`
	ReviewerName string                  `json:"reviewer_name"`
	ReviewerRole string                  `json:"reviewer_role"`
	Credentials  string                  `json:"credentials,omitempty"`
	Notes        string                  `json:"notes"`
	Checklist    *models.ReviewChecklist `json:"checklist,omitempty"`
}

// ApprovalRequest contains approval details.
type ApprovalRequest struct {
	ItemID       string          `json:"item_id"`
	ApproverID   string          `json:"approver_id"`
	ApproverName string          `json:"approver_name"`
	ApproverRole string          `json:"approver_role"`
	Credentials  string          `json:"credentials,omitempty"`
	Notes        string          `json:"notes"`
	Attestations map[string]bool `json:"attestations,omitempty"`
}

// TransitionResult contains the result of a state transition.
type TransitionResult struct {
	Success       bool             `json:"success"`
	PreviousState models.ItemState `json:"previous_state"`
	NewState      models.ItemState `json:"new_state"`
	ItemID        string           `json:"item_id"`
	AuditID       string           `json:"audit_id"`
	Message       string           `json:"message"`
}

// Metrics contains governance metrics.
type Metrics struct {
	KB                  models.KB `json:"kb"`
	ActiveCount         int       `json:"active_count"`
	PendingReviewCount  int       `json:"pending_review_count"`
	PendingApprovalCount int      `json:"pending_approval_count"`
	HoldCount           int       `json:"hold_count"`
	EmergencyCount      int       `json:"emergency_count"`
	RetiredCount        int       `json:"retired_count"`
	RejectedCount       int       `json:"rejected_count"`
	HighRiskActiveCount int       `json:"high_risk_active_count"`
	TotalCount          int       `json:"total_count"`
}

// =============================================================================
// HTTP HELPERS
// =============================================================================

func (c *Client) get(ctx context.Context, path string, result interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	return c.do(req, result)
}

func (c *Client) post(ctx context.Context, path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req, result)
}

func (c *Client) put(ctx context.Context, path string, body, result interface{}) error {
	var bodyReader io.Reader
	if body != nil {
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyReader = bytes.NewReader(bodyBytes)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPut, c.baseURL+path, bodyReader)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	return c.do(req, result)
}

func (c *Client) do(req *http.Request, result interface{}) error {
	if c.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}
	req.Header.Set("X-KB-ID", string(c.kb))

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("API error (status %d): %s", resp.StatusCode, string(body))
	}

	if result != nil {
		if err := json.NewDecoder(resp.Body).Decode(result); err != nil {
			return fmt.Errorf("failed to decode response: %w", err)
		}
	}

	return nil
}
