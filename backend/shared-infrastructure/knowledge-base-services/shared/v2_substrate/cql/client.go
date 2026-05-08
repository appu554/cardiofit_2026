// Package cql provides a typed Go client for the kb-cql-runtime Java service,
// which exposes CQL-evaluated clinical decision rules via:
//
//	POST /Library/{ruleId}/$evaluate-rule?residentId={uuid}
//
// The RuleResult shape reflects the Phase 2 "eventual contract". While the
// Java service currently returns a placeholder shape (Task 5 transitional
// state), the Go client targets the final schema so that Phase 2 consumers
// need not be updated when the Java engine catches up.
//
// Example:
//
//	c := cql.NewClient("http://kb-cql-runtime:8095")
//	result, err := c.EvaluateRule(ctx, "EGFR-001", residentID)
package cql

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// defaultTimeout is applied to the internal http.Client. A bare
// &http.Client{} has no timeout, which would cause goroutine leaks when
// the upstream Java service is unresponsive.
const defaultTimeout = 5 * time.Second

// Client is a thin HTTP client over the kb-cql-runtime $evaluate-rule
// endpoint. Construct via NewClient; the zero value is not usable.
type Client struct {
	baseURL string
	http    *http.Client
}

// NewClient returns a Client targeting baseURL (e.g. "http://kb-cql-runtime:8095").
// The underlying HTTP client carries a 5-second request timeout.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL: baseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// SetHTTPClient replaces the underlying HTTP client, e.g. to inject a
// timeout override or an instrumented transport for tracing.
func (c *Client) SetHTTPClient(h *http.Client) {
	if h != nil {
		c.http = h
	}
}

// RuleResult is the Phase 2 contract shape returned by $evaluate-rule.
// While kb-cql-runtime currently returns a placeholder response (Task 5
// transitional state), callers should bind to this struct; the Java engine
// will converge to this shape in Phase 2.
type RuleResult struct {
	Triggered       bool           `json:"triggered"`
	Type            string         `json:"type"`
	Urgency         string         `json:"urgency"`
	ClinicalContent map[string]any `json:"clinical_content"`
}

// EvaluateRule invokes POST /Library/{ruleID}/$evaluate-rule for the given
// residentID and decodes the response into a RuleResult.
//
// The ruleID is path-escaped so that identifiers containing slashes or
// special characters (e.g. "BP-MONITOR/2024") are transmitted safely.
// net/url.URL.RawPath is used to preserve the percent-encoding through the
// Go HTTP client, which would otherwise normalise %2F back to /.
//
// Errors are returned for:
//   - context cancellation or deadline exceeded
//   - network failures
//   - non-2xx HTTP status codes (the status code and body are included in the error)
//   - JSON decode failures
func (c *Client) EvaluateRule(ctx context.Context, ruleID string, residentID uuid.UUID) (*RuleResult, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("cql: parse base URL: %w", err)
	}

	escapedRuleID := url.PathEscape(ruleID)
	// Set both Path (decoded) and RawPath (encoded) so that the Go HTTP
	// transport serialises the percent-encoded form on the wire.
	base.Path = "/Library/" + ruleID + "/$evaluate-rule"
	base.RawPath = "/Library/" + escapedRuleID + "/$evaluate-rule"
	base.RawQuery = url.Values{"residentId": {residentID.String()}}.Encode()

	rawURL := base.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("cql: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("cql: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("cql: POST %s: status %d: %s",
			rawURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var out RuleResult
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return nil, fmt.Errorf("cql: decode response: %w", err)
	}
	return &out, nil
}
