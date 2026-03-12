// Package kb1client provides a production-grade client for KB-1 Drug Rules Service.
// This integrates KB-0 governance workflow with KB-1's real database via REST API.
//
// Architecture:
//   KB-0 Workflow Engine → KB1Store → KB-1 REST API → KB-1 PostgreSQL
//
// The KB1Store implements workflow.ItemStore interface, allowing KB-0's
// workflow engine to orchestrate governance while KB-1 persists the state.
package kb1client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"kb-0-governance-platform/internal/models"
)

// Client provides KB-1 API integration for governance workflow.
type Client struct {
	baseURL    string
	httpClient *http.Client
}

// NewClient creates a new KB-1 client.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// =============================================================================
// KB-1 API Response Types
// =============================================================================

// DrugRuleResponse from KB-1 admin API
type DrugRuleResponse struct {
	Drug       DrugInfo       `json:"drug"`
	Dosing     interface{}    `json:"dosing"`
	Safety     SafetyInfo     `json:"safety"`
	Governance GovernanceInfo `json:"governance"`
}

type DrugInfo struct {
	RxNormCode  string `json:"rxnorm_code"`
	Name        string `json:"name"`
	GenericName string `json:"generic_name"`
	DrugClass   string `json:"drug_class"`
}

type SafetyInfo struct {
	BlackBoxWarning  bool     `json:"black_box_warning"`
	BlackBoxText     string   `json:"black_box_text"`
	HighAlertDrug    bool     `json:"high_alert_drug"`
	Contraindications []string `json:"contraindications"`
}

type GovernanceInfo struct {
	Authority    string    `json:"authority"`
	Document     string    `json:"document"`
	URL          string    `json:"url"`
	Jurisdiction string    `json:"jurisdiction"`
	Version      string    `json:"version"`
	ApprovedBy   string    `json:"approved_by"`
	ApprovedAt   time.Time `json:"approved_at"`
	SourceSetID  string    `json:"source_set_id"`
}

// PendingItem from KB-1 pending reviews endpoint
type PendingItem struct {
	ID             string    `json:"ID"`
	RxNormCode     string    `json:"RxNormCode"`
	DrugName       string    `json:"DrugName"`
	GenericName    string    `json:"GenericName"`
	DrugClass      string    `json:"DrugClass"`
	Jurisdiction   string    `json:"Jurisdiction"`
	Authority      string    `json:"Authority"`
	ApprovalStatus string    `json:"ApprovalStatus"`
	RiskLevel      string    `json:"RiskLevel"`
	IsHighAlert    bool      `json:"IsHighAlert"`
	HasBlackBox    bool      `json:"HasBlackBox"`
	SourceSetID    string    `json:"SourceSetID"`
	DocumentURL    string    `json:"DocumentURL"`
	IngestedAt     time.Time `json:"IngestedAt"`
}

type PendingResponse struct {
	Count int           `json:"count"`
	Items []PendingItem `json:"items"`
}

// ReviewRequest for KB-1 review API
type ReviewRequest struct {
	ReviewedBy           string `json:"reviewed_by"`
	ReviewNotes          string `json:"review_notes"`
	DosingVerified       bool   `json:"dosing_verified"`
	RenalVerified        bool   `json:"renal_verified"`
	HepaticVerified      bool   `json:"hepatic_verified"`
	InteractionsVerified bool   `json:"interactions_verified"`
	SafetyVerified       bool   `json:"safety_verified"`
}

// ApproveRequest for KB-1 approve API
type ApproveRequest struct {
	ApprovedBy       string `json:"approved_by"`
	ReviewNotes      string `json:"review_notes"`
	SkipVerification bool   `json:"skip_verification"`
}

// =============================================================================
// ItemStore Implementation
// =============================================================================

// KB1Store implements workflow.ItemStore using KB-1 API
type KB1Store struct {
	client *Client
	items  map[string]*models.KnowledgeItem // Local cache
	mu     sync.RWMutex                     // Thread-safe cache access
}

// NewKB1Store creates a store backed by KB-1 API
func NewKB1Store(baseURL string) *KB1Store {
	return &KB1Store{
		client: NewClient(baseURL),
		items:  make(map[string]*models.KnowledgeItem),
	}
}

// GetItem retrieves a drug rule from KB-1 and converts to KnowledgeItem
func (s *KB1Store) GetItem(ctx context.Context, itemID string) (*models.KnowledgeItem, error) {
	// Check local cache first (thread-safe)
	s.mu.RLock()
	if item, ok := s.items[itemID]; ok {
		s.mu.RUnlock()
		return item, nil
	}
	s.mu.RUnlock()

	// itemID format: "kb1:drug:<name>:<rxnorm>" or just the UUID
	// Extract RxNorm code if possible
	rxnormCode := extractRxNormCode(itemID)
	if rxnormCode == "" {
		return nil, fmt.Errorf("invalid item ID format: %s", itemID)
	}

	// Fetch from KB-1 admin API
	url := fmt.Sprintf("%s/v1/admin/rules/%s", s.client.baseURL, rxnormCode)
	resp, err := s.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch from KB-1: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, fmt.Errorf("drug rule not found: %s", rxnormCode)
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-1 API error: %s", string(body))
	}

	var ruleResp DrugRuleResponse
	if err := json.NewDecoder(resp.Body).Decode(&ruleResp); err != nil {
		return nil, fmt.Errorf("failed to decode KB-1 response: %w", err)
	}

	// Convert to KnowledgeItem
	item := s.convertToKnowledgeItem(itemID, &ruleResp)

	s.mu.Lock()
	s.items[itemID] = item
	s.mu.Unlock()

	return item, nil
}

// UpdateItem updates state in KB-1 via admin API
// This is called by the workflow engine after state transitions.
// It syncs the governance state to KB-1's database.
func (s *KB1Store) UpdateItem(ctx context.Context, item *models.KnowledgeItem) error {
	// Update local cache (thread-safe)
	s.mu.Lock()
	s.items[item.ID] = item
	s.mu.Unlock()

	// Extract the KB-1 rule ID (UUID format)
	ruleID := item.ID

	// Sync state to KB-1 based on the new state
	switch item.State {
	case models.StateReviewed:
		// After review, call KB-1's review API
		if len(item.Governance.Reviews) > 0 {
			lastReview := item.Governance.Reviews[len(item.Governance.Reviews)-1]
			checklist := make(map[string]bool)
			if lastReview.Checklist != nil {
				for _, ci := range lastReview.Checklist.Items {
					checklist[ci.ID] = ci.Verified
				}
			}
			return s.SubmitReview(ctx, ruleID, lastReview.ReviewerName, lastReview.Notes, checklist)
		}

	case models.StateApproved, models.StateActive:
		// After approval, call KB-1's approve API
		if item.Governance.Approval != nil {
			approval := item.Governance.Approval
			isHighRisk := item.RequiresDualReview
			return s.SubmitApproval(ctx, ruleID, approval.ApproverName, approval.Notes, isHighRisk)
		}
	}

	return nil
}

// GetItemsByState returns items in specific states
func (s *KB1Store) GetItemsByState(ctx context.Context, kb models.KB, states []models.ItemState) ([]*models.KnowledgeItem, error) {
	if kb != models.KB1 {
		return nil, nil // Only support KB-1
	}

	// Fetch pending items from KB-1
	url := fmt.Sprintf("%s/v1/admin/pending", s.client.baseURL)
	resp, err := s.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending items: %w", err)
	}
	defer resp.Body.Close()

	var pending PendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&pending); err != nil {
		return nil, fmt.Errorf("failed to decode pending response: %w", err)
	}

	var result []*models.KnowledgeItem
	for _, p := range pending.Items {
		item := s.convertPendingToItem(&p)
		result = append(result, item)
	}

	return result, nil
}

// =============================================================================
// KB-1 Specific Operations
// =============================================================================

// SubmitReview calls KB-1's review API
func (s *KB1Store) SubmitReview(ctx context.Context, ruleID string, reviewedBy, notes string, checklist map[string]bool) error {
	req := ReviewRequest{
		ReviewedBy:           reviewedBy,
		ReviewNotes:          notes,
		DosingVerified:       checklist["dosing"],
		RenalVerified:        checklist["renal"],
		HepaticVerified:      checklist["hepatic"],
		InteractionsVerified: checklist["interactions"],
		SafetyVerified:       checklist["safety"],
	}

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/v1/admin/review/%s", s.client.baseURL, ruleID)

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to submit review: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("review failed: %s", string(respBody))
	}

	return nil
}

// SubmitApproval calls KB-1's approve API
func (s *KB1Store) SubmitApproval(ctx context.Context, ruleID string, approvedBy, notes string, isHighRisk bool) error {
	req := ApproveRequest{
		ApprovedBy:       approvedBy,
		ReviewNotes:      notes,
		SkipVerification: isHighRisk, // High-risk drugs require explicit verification flag
	}

	body, _ := json.Marshal(req)
	url := fmt.Sprintf("%s/v1/admin/approve/%s", s.client.baseURL, ruleID)

	httpReq, _ := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	httpReq.Header.Set("Content-Type", "application/json")

	resp, err := s.client.httpClient.Do(httpReq)
	if err != nil {
		return fmt.Errorf("failed to submit approval: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("approval failed: %s", string(respBody))
	}

	return nil
}

// GetPendingDrugs returns drugs pending review from KB-1
func (s *KB1Store) GetPendingDrugs(ctx context.Context) ([]PendingItem, error) {
	url := fmt.Sprintf("%s/v1/admin/pending", s.client.baseURL)
	resp, err := s.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch pending items: %w", err)
	}
	defer resp.Body.Close()

	var pending PendingResponse
	if err := json.NewDecoder(resp.Body).Decode(&pending); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return pending.Items, nil
}

// GetDrugRule fetches a specific drug rule by RxNorm code
func (s *KB1Store) GetDrugRule(ctx context.Context, rxnormCode string) (*DrugRuleResponse, error) {
	url := fmt.Sprintf("%s/v1/admin/rules/%s", s.client.baseURL, rxnormCode)
	resp, err := s.client.httpClient.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch drug rule: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("KB-1 error: %s", string(body))
	}

	var rule DrugRuleResponse
	if err := json.NewDecoder(resp.Body).Decode(&rule); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &rule, nil
}

// =============================================================================
// Helpers
// =============================================================================

func extractRxNormCode(itemID string) string {
	// Handle format "kb1:drug:<name>:<rxnorm>"
	// or just return if it's already an RxNorm code
	if len(itemID) > 4 && itemID[0:4] == "kb1:" {
		// Parse last segment
		parts := splitLast(itemID, ":")
		if len(parts) > 0 {
			return parts[len(parts)-1]
		}
	}
	return itemID
}

func splitLast(s, sep string) []string {
	var result []string
	start := 0
	for i := 0; i < len(s); i++ {
		if i+len(sep) <= len(s) && s[i:i+len(sep)] == sep {
			result = append(result, s[start:i])
			start = i + len(sep)
		}
	}
	result = append(result, s[start:])
	return result
}

func (s *KB1Store) convertToKnowledgeItem(itemID string, rule *DrugRuleResponse) *models.KnowledgeItem {
	// Determine risk level from safety info
	requiresDual := rule.Safety.HighAlertDrug || rule.Safety.BlackBoxWarning

	return &models.KnowledgeItem{
		ID:                 itemID,
		Type:               models.TypeDosingRule,
		KB:                 models.KB1,
		Name:               rule.Drug.Name + " Dosing Rule",
		Description:        fmt.Sprintf("Dosing guidance for %s (%s)", rule.Drug.Name, rule.Drug.DrugClass),
		ContentRef:         fmt.Sprintf("kb1/drugs/%s/%s.yaml", rule.Drug.GenericName, rule.Drug.RxNormCode),
		Version:            rule.Governance.Version,
		State:              models.StateDraft, // Default, will be updated by workflow
		WorkflowTemplate:   models.TemplateClinicalHigh,
		RequiresDualReview: requiresDual,
		CreatedAt:          rule.Governance.ApprovedAt,
		UpdatedAt:          time.Now(),
		Source: models.SourceAttribution{
			Authority:    models.Authority(rule.Governance.Authority),
			Document:     rule.Governance.Document,
			URL:          rule.Governance.URL,
			Jurisdiction: models.Jurisdiction(rule.Governance.Jurisdiction),
		},
		Governance: models.GovernanceTrail{
			CreatedBy: rule.Governance.ApprovedBy,
			Reviews:   []models.Review{},
		},
	}
}

func (s *KB1Store) convertPendingToItem(p *PendingItem) *models.KnowledgeItem {
	requiresDual := p.IsHighAlert || p.HasBlackBox || p.RiskLevel == "CRITICAL" || p.RiskLevel == "HIGH"

	return &models.KnowledgeItem{
		ID:                 p.ID,
		Type:               models.TypeDosingRule,
		KB:                 models.KB1,
		Name:               p.DrugName + " Dosing Rule",
		Description:        fmt.Sprintf("Dosing guidance for %s (%s)", p.DrugName, p.DrugClass),
		ContentRef:         fmt.Sprintf("kb1/drugs/%s/%s.yaml", p.GenericName, p.RxNormCode),
		State:              models.ItemState(p.ApprovalStatus),
		WorkflowTemplate:   models.TemplateClinicalHigh,
		RequiresDualReview: requiresDual,
		CreatedAt:          p.IngestedAt,
		UpdatedAt:          time.Now(),
		Source: models.SourceAttribution{
			Authority:    models.Authority(p.Authority),
			Document:     "DailyMed SPL",
			URL:          p.DocumentURL,
			Jurisdiction: models.Jurisdiction(p.Jurisdiction),
		},
	}
}
