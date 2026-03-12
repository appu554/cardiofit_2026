// Package clients provides HTTP clients for KB services.
//
// KB10HTTPClient implements the KB10Client interface for KB-10 Rules Engine Service.
// It provides clinical rule evaluation and alert management.
//
// ARCHITECTURE NOTE (CTO/CMO Directive):
// KB-10 is a RUNTIME KB - called during workflow execution, NOT during snapshot build.
// It executes rules against CQL outputs and generates clinical alerts.
//
// Workflow Pattern:
// 1. CQL evaluates against frozen snapshot → produces facts
// 2. KB-10 evaluates rules against facts → generates alerts
// 3. ICU Intelligence veto MUST be checked BEFORE rule execution
//
// Connects to: http://localhost:8090 (Docker: kb10-rules-engine)
package clients

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"vaidshala/clinical-runtime-platform/contracts"
)

// KB10HTTPClient implements KB10Client by calling the KB-10 Rules Engine Service REST API.
type KB10HTTPClient struct {
	baseURL    string
	httpClient *http.Client
}

// NewKB10HTTPClient creates a new KB-10 HTTP client.
func NewKB10HTTPClient(baseURL string) *KB10HTTPClient {
	return &KB10HTTPClient{
		baseURL: baseURL,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// NewKB10HTTPClientWithHTTP creates a client with custom HTTP client.
func NewKB10HTTPClientWithHTTP(baseURL string, httpClient *http.Client) *KB10HTTPClient {
	return &KB10HTTPClient{
		baseURL:    baseURL,
		httpClient: httpClient,
	}
}

// ============================================================================
// KB10Client Interface Implementation (RUNTIME)
// ============================================================================

// EvaluateRules executes a rule set against provided clinical facts.
// This is a RUNTIME operation - called during workflow execution.
//
// Parameters:
// - ruleSetID: Identifier for the rule set to evaluate (e.g., "sepsis-bundle", "aki-protocol")
// - facts: Map of clinical facts from CQL evaluation (e.g., {"lactate": 4.5, "qsofaScore": 2})
//
// Returns:
// - RuleEvaluationResult containing triggered rules, generated alerts, and recommendations
func (c *KB10HTTPClient) EvaluateRules(
	ctx context.Context,
	ruleSetID string,
	facts map[string]interface{},
) (*contracts.RuleEvaluationResult, error) {

	req := kb10EvaluateRequest{
		RuleSetID: ruleSetID,
		Facts:     facts,
		Timestamp: time.Now().UTC(),
	}

	resp, err := c.callKB10(ctx, "/api/v1/rules/evaluate", req)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate rules: %w", err)
	}

	var result kb10EvaluateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse evaluation response: %w", err)
	}

	// Convert to contracts type
	triggered := make([]contracts.TriggeredRule, 0, len(result.TriggeredRules))
	for _, tr := range result.TriggeredRules {
		triggered = append(triggered, contracts.TriggeredRule{
			RuleID:          tr.RuleID,
			RuleName:        tr.RuleName,
			Severity:        tr.Severity,
			Condition:       tr.Condition,
			MatchedFacts:    tr.MatchedFacts,
			Recommendation:  tr.Recommendation,
			EvidenceLevel:   tr.EvidenceLevel,
			GuidelineSource: tr.GuidelineSource,
		})
	}

	alerts := make([]contracts.ClinicalAlert, 0, len(result.GeneratedAlerts))
	for _, a := range result.GeneratedAlerts {
		alerts = append(alerts, contracts.ClinicalAlert{
			AlertID:     a.AlertID,
			PatientID:   a.PatientID,
			AlertType:   a.AlertType,
			Severity:    a.Severity,
			Title:       a.Title,
			Description: a.Description,
			SourceRule:  a.SourceRule,
			GeneratedAt: a.GeneratedAt,
			ExpiresAt:   a.ExpiresAt,
			Acknowledged: a.Acknowledged,
			ActionItems: a.ActionItems,
		})
	}

	return &contracts.RuleEvaluationResult{
		RuleSetID:       ruleSetID,
		EvaluatedAt:     time.Now().UTC(),
		TriggeredRules:  triggered,
		GeneratedAlerts: alerts,
		TotalRulesRun:   result.TotalRulesRun,
		ExecutionTimeMs: result.ExecutionTimeMs,
	}, nil
}

// GetActiveAlerts returns all currently active (non-acknowledged) alerts for a patient.
// Useful for displaying pending alerts in clinical dashboards.
func (c *KB10HTTPClient) GetActiveAlerts(
	ctx context.Context,
	patientID string,
) ([]contracts.ClinicalAlert, error) {

	req := kb10AlertsRequest{
		PatientID:     patientID,
		ActiveOnly:    true,
		IncludeExpired: false,
	}

	resp, err := c.callKB10(ctx, "/api/v1/alerts/active", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get active alerts: %w", err)
	}

	var result kb10AlertsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse alerts response: %w", err)
	}

	alerts := make([]contracts.ClinicalAlert, 0, len(result.Alerts))
	for _, a := range result.Alerts {
		alerts = append(alerts, contracts.ClinicalAlert{
			AlertID:      a.AlertID,
			PatientID:    a.PatientID,
			AlertType:    a.AlertType,
			Severity:     a.Severity,
			Title:        a.Title,
			Description:  a.Description,
			SourceRule:   a.SourceRule,
			GeneratedAt:  a.GeneratedAt,
			ExpiresAt:    a.ExpiresAt,
			Acknowledged: a.Acknowledged,
			ActionItems:  a.ActionItems,
		})
	}

	return alerts, nil
}

// GetAlertHistory returns all alerts for a patient including acknowledged and expired.
// Useful for audit trails and clinical review.
func (c *KB10HTTPClient) GetAlertHistory(
	ctx context.Context,
	patientID string,
	startDate time.Time,
	endDate time.Time,
) ([]contracts.ClinicalAlert, error) {

	req := kb10AlertHistoryRequest{
		PatientID: patientID,
		StartDate: startDate,
		EndDate:   endDate,
	}

	resp, err := c.callKB10(ctx, "/api/v1/alerts/history", req)
	if err != nil {
		return nil, fmt.Errorf("failed to get alert history: %w", err)
	}

	var result kb10AlertsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse alerts response: %w", err)
	}

	alerts := make([]contracts.ClinicalAlert, 0, len(result.Alerts))
	for _, a := range result.Alerts {
		alerts = append(alerts, contracts.ClinicalAlert{
			AlertID:        a.AlertID,
			PatientID:      a.PatientID,
			AlertType:      a.AlertType,
			Severity:       a.Severity,
			Title:          a.Title,
			Description:    a.Description,
			SourceRule:     a.SourceRule,
			GeneratedAt:    a.GeneratedAt,
			ExpiresAt:      a.ExpiresAt,
			Acknowledged:   a.Acknowledged,
			AcknowledgedBy: a.AcknowledgedBy,
			AcknowledgedAt: a.AcknowledgedAt,
			ActionItems:    a.ActionItems,
		})
	}

	return alerts, nil
}

// AcknowledgeAlert marks an alert as acknowledged by a clinical user.
// Creates an audit trail entry for regulatory compliance.
//
// Parameters:
// - alertID: Unique identifier of the alert to acknowledge
// - acknowledgerID: User ID of the person acknowledging (for audit trail)
// - notes: Optional clinical notes about the acknowledgment
func (c *KB10HTTPClient) AcknowledgeAlert(
	ctx context.Context,
	alertID string,
	acknowledgerID string,
	notes string,
) error {

	req := kb10AcknowledgeRequest{
		AlertID:        alertID,
		AcknowledgerID: acknowledgerID,
		Notes:          notes,
		AcknowledgedAt: time.Now().UTC(),
	}

	_, err := c.callKB10(ctx, "/api/v1/alerts/acknowledge", req)
	if err != nil {
		return fmt.Errorf("failed to acknowledge alert: %w", err)
	}

	return nil
}

// SnoozeAlert temporarily defers an alert for a specified duration.
// Useful for alerts that should be revisited later.
func (c *KB10HTTPClient) SnoozeAlert(
	ctx context.Context,
	alertID string,
	snoozedBy string,
	snoozeDuration time.Duration,
	reason string,
) error {

	req := kb10SnoozeRequest{
		AlertID:    alertID,
		SnoozedBy:  snoozedBy,
		SnoozeUntil: time.Now().UTC().Add(snoozeDuration),
		Reason:     reason,
	}

	_, err := c.callKB10(ctx, "/api/v1/alerts/snooze", req)
	if err != nil {
		return fmt.Errorf("failed to snooze alert: %w", err)
	}

	return nil
}

// GetRuleDefinitions returns all available rule set definitions.
// Useful for administrative configuration and rule set discovery.
func (c *KB10HTTPClient) GetRuleDefinitions(
	ctx context.Context,
) ([]contracts.RuleSetDefinition, error) {

	resp, err := c.callKB10Get(ctx, "/api/v1/rules/definitions")
	if err != nil {
		return nil, fmt.Errorf("failed to get rule definitions: %w", err)
	}

	var result kb10RuleDefsResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rule definitions response: %w", err)
	}

	defs := make([]contracts.RuleSetDefinition, 0, len(result.RuleSets))
	for _, rs := range result.RuleSets {
		rules := make([]contracts.RuleDefinition, 0, len(rs.Rules))
		for _, r := range rs.Rules {
			rules = append(rules, contracts.RuleDefinition{
				RuleID:          r.RuleID,
				Name:            r.Name,
				Description:     r.Description,
				Condition:       r.Condition,
				Severity:        r.Severity,
				Enabled:         r.Enabled,
				EvidenceLevel:   r.EvidenceLevel,
				GuidelineSource: r.GuidelineSource,
			})
		}

		defs = append(defs, contracts.RuleSetDefinition{
			RuleSetID:   rs.RuleSetID,
			Name:        rs.Name,
			Description: rs.Description,
			Version:     rs.Version,
			Category:    rs.Category,
			Enabled:     rs.Enabled,
			Rules:       rules,
		})
	}

	return defs, nil
}

// GetRuleSetByID returns a specific rule set definition by ID.
func (c *KB10HTTPClient) GetRuleSetByID(
	ctx context.Context,
	ruleSetID string,
) (*contracts.RuleSetDefinition, error) {

	endpoint := fmt.Sprintf("/api/v1/rules/definitions/%s", ruleSetID)
	resp, err := c.callKB10Get(ctx, endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to get rule set: %w", err)
	}

	var result kb10RuleSetResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse rule set response: %w", err)
	}

	rules := make([]contracts.RuleDefinition, 0, len(result.RuleSet.Rules))
	for _, r := range result.RuleSet.Rules {
		rules = append(rules, contracts.RuleDefinition{
			RuleID:          r.RuleID,
			Name:            r.Name,
			Description:     r.Description,
			Condition:       r.Condition,
			Severity:        r.Severity,
			Enabled:         r.Enabled,
			EvidenceLevel:   r.EvidenceLevel,
			GuidelineSource: r.GuidelineSource,
		})
	}

	return &contracts.RuleSetDefinition{
		RuleSetID:   result.RuleSet.RuleSetID,
		Name:        result.RuleSet.Name,
		Description: result.RuleSet.Description,
		Version:     result.RuleSet.Version,
		Category:    result.RuleSet.Category,
		Enabled:     result.RuleSet.Enabled,
		Rules:       rules,
	}, nil
}

// EvaluateMultipleRuleSets evaluates multiple rule sets against the same facts.
// Useful for comprehensive patient evaluation across clinical domains.
func (c *KB10HTTPClient) EvaluateMultipleRuleSets(
	ctx context.Context,
	ruleSetIDs []string,
	facts map[string]interface{},
) (*contracts.MultiRuleEvaluationResult, error) {

	req := kb10MultiEvaluateRequest{
		RuleSetIDs: ruleSetIDs,
		Facts:      facts,
		Timestamp:  time.Now().UTC(),
	}

	resp, err := c.callKB10(ctx, "/api/v1/rules/evaluate-multi", req)
	if err != nil {
		return nil, fmt.Errorf("failed to evaluate multiple rule sets: %w", err)
	}

	var result kb10MultiEvaluateResponse
	if err := json.Unmarshal(resp, &result); err != nil {
		return nil, fmt.Errorf("failed to parse multi-evaluation response: %w", err)
	}

	// Aggregate results
	allTriggered := make([]contracts.TriggeredRule, 0)
	allAlerts := make([]contracts.ClinicalAlert, 0)
	var totalRulesRun int
	var totalExecutionMs int64

	for _, evalResult := range result.Results {
		for _, tr := range evalResult.TriggeredRules {
			allTriggered = append(allTriggered, contracts.TriggeredRule{
				RuleID:          tr.RuleID,
				RuleName:        tr.RuleName,
				Severity:        tr.Severity,
				Condition:       tr.Condition,
				MatchedFacts:    tr.MatchedFacts,
				Recommendation:  tr.Recommendation,
				EvidenceLevel:   tr.EvidenceLevel,
				GuidelineSource: tr.GuidelineSource,
			})
		}

		for _, a := range evalResult.GeneratedAlerts {
			allAlerts = append(allAlerts, contracts.ClinicalAlert{
				AlertID:     a.AlertID,
				PatientID:   a.PatientID,
				AlertType:   a.AlertType,
				Severity:    a.Severity,
				Title:       a.Title,
				Description: a.Description,
				SourceRule:  a.SourceRule,
				GeneratedAt: a.GeneratedAt,
				ExpiresAt:   a.ExpiresAt,
				ActionItems: a.ActionItems,
			})
		}

		totalRulesRun += evalResult.TotalRulesRun
		totalExecutionMs += evalResult.ExecutionTimeMs
	}

	return &contracts.MultiRuleEvaluationResult{
		RuleSetIDs:      ruleSetIDs,
		EvaluatedAt:     time.Now().UTC(),
		TriggeredRules:  allTriggered,
		GeneratedAlerts: allAlerts,
		TotalRulesRun:   totalRulesRun,
		ExecutionTimeMs: totalExecutionMs,
	}, nil
}

// ============================================================================
// HTTP Helper Methods
// ============================================================================

func (c *KB10HTTPClient) callKB10(ctx context.Context, endpoint string, body interface{}) ([]byte, error) {
	url := c.baseURL + endpoint

	bodyBytes, err := json.Marshal(body)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-10 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

func (c *KB10HTTPClient) callKB10Get(ctx context.Context, endpoint string) ([]byte, error) {
	url := c.baseURL + endpoint

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("KB-10 returned HTTP %d: %s", resp.StatusCode, string(respBody))
	}

	return respBody, nil
}

// HealthCheck verifies KB-10 service is healthy.
func (c *KB10HTTPClient) HealthCheck(ctx context.Context) error {
	endpoint := fmt.Sprintf("%s/health", c.baseURL)

	req, err := http.NewRequestWithContext(ctx, "GET", endpoint, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("KB-10 unhealthy: HTTP %d", resp.StatusCode)
	}

	return nil
}

// ============================================================================
// KB-10 Request/Response Types (internal)
// ============================================================================

type kb10EvaluateRequest struct {
	RuleSetID string                 `json:"rule_set_id"`
	Facts     map[string]interface{} `json:"facts"`
	Timestamp time.Time              `json:"timestamp"`
}

type kb10EvaluateResponse struct {
	TriggeredRules  []kb10TriggeredRule `json:"triggered_rules"`
	GeneratedAlerts []kb10Alert         `json:"generated_alerts"`
	TotalRulesRun   int                 `json:"total_rules_run"`
	ExecutionTimeMs int64               `json:"execution_time_ms"`
}

type kb10TriggeredRule struct {
	RuleID          string                 `json:"rule_id"`
	RuleName        string                 `json:"rule_name"`
	Severity        string                 `json:"severity"`
	Condition       string                 `json:"condition"`
	MatchedFacts    map[string]interface{} `json:"matched_facts"`
	Recommendation  string                 `json:"recommendation"`
	EvidenceLevel   string                 `json:"evidence_level"`
	GuidelineSource string                 `json:"guideline_source"`
}

type kb10Alert struct {
	AlertID        string    `json:"alert_id"`
	PatientID      string    `json:"patient_id"`
	AlertType      string    `json:"alert_type"`
	Severity       string    `json:"severity"`
	Title          string    `json:"title"`
	Description    string    `json:"description"`
	SourceRule     string    `json:"source_rule"`
	GeneratedAt    time.Time `json:"generated_at"`
	ExpiresAt      time.Time `json:"expires_at"`
	Acknowledged   bool      `json:"acknowledged"`
	AcknowledgedBy string    `json:"acknowledged_by,omitempty"`
	AcknowledgedAt time.Time `json:"acknowledged_at,omitempty"`
	ActionItems    []string  `json:"action_items"`
}

type kb10AlertsRequest struct {
	PatientID      string `json:"patient_id"`
	ActiveOnly     bool   `json:"active_only"`
	IncludeExpired bool   `json:"include_expired"`
}

type kb10AlertHistoryRequest struct {
	PatientID string    `json:"patient_id"`
	StartDate time.Time `json:"start_date"`
	EndDate   time.Time `json:"end_date"`
}

type kb10AlertsResponse struct {
	Alerts []kb10Alert `json:"alerts"`
}

type kb10AcknowledgeRequest struct {
	AlertID        string    `json:"alert_id"`
	AcknowledgerID string    `json:"acknowledger_id"`
	Notes          string    `json:"notes"`
	AcknowledgedAt time.Time `json:"acknowledged_at"`
}

type kb10SnoozeRequest struct {
	AlertID     string    `json:"alert_id"`
	SnoozedBy   string    `json:"snoozed_by"`
	SnoozeUntil time.Time `json:"snooze_until"`
	Reason      string    `json:"reason"`
}

type kb10RuleDefsResponse struct {
	RuleSets []kb10RuleSet `json:"rule_sets"`
}

type kb10RuleSetResponse struct {
	RuleSet kb10RuleSet `json:"rule_set"`
}

type kb10RuleSet struct {
	RuleSetID   string      `json:"rule_set_id"`
	Name        string      `json:"name"`
	Description string      `json:"description"`
	Version     string      `json:"version"`
	Category    string      `json:"category"`
	Enabled     bool        `json:"enabled"`
	Rules       []kb10Rule  `json:"rules"`
}

type kb10Rule struct {
	RuleID          string `json:"rule_id"`
	Name            string `json:"name"`
	Description     string `json:"description"`
	Condition       string `json:"condition"`
	Severity        string `json:"severity"`
	Enabled         bool   `json:"enabled"`
	EvidenceLevel   string `json:"evidence_level"`
	GuidelineSource string `json:"guideline_source"`
}

type kb10MultiEvaluateRequest struct {
	RuleSetIDs []string               `json:"rule_set_ids"`
	Facts      map[string]interface{} `json:"facts"`
	Timestamp  time.Time              `json:"timestamp"`
}

type kb10MultiEvaluateResponse struct {
	Results []kb10EvaluateResponse `json:"results"`
}
