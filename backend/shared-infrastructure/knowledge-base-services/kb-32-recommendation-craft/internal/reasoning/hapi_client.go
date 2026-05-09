// Package reasoning implements Stage 2 of the six-stage rendering pipeline:
// reasoning chain building via CQL rule evaluation.
//
// VisibilityClass: PDP — clinical reasoning context
//
// # Design Decision: Local HAPIClient vs Shared cql.Client
//
// The shared substrate provides a cql.Client at
// github.com/cardiofit/shared/v2_substrate/cql that covers the same
// $evaluate-rule endpoint. However, importing the shared module would pull
// redis, logrus, and sqlite into kb-32, which is an undesirable dependency
// footprint for a focused recommendation-craft service.
//
// Instead, HAPIClient mirrors the shared client's conventions exactly:
//   - 5-second default timeout on the internal http.Client
//   - Body drain on non-2xx to avoid goroutine leaks
//   - Identical URL construction (path-escape for special characters)
//   - Same 4096-byte limit on error body capture
//
// The key addition over the shared client is placeholder-response detection:
// kb-cql-runtime currently returns status="library_found_engine_pending" while
// the Phase 0.5 CQF-FHIR-CR engine wiring is deferred. ChainBuilder treats
// this as a skip, not an error.
package reasoning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
)

// defaultTimeout is applied to the internal http.Client, mirroring the
// shared cql.Client convention.
const defaultTimeout = 5 * time.Second

// placeholderStatus is the status value returned by kb-cql-runtime while the
// CQF-FHIR-CR engine is deferred (Phase 0.5 transitional state).
const placeholderStatus = "library_found_engine_pending"

// ErrCQLPlaceholderResponse is returned by EvaluateRule when kb-cql-runtime
// responds with status="library_found_engine_pending", indicating that the
// CQL library was found but the evaluation engine is not yet wired.
//
// ChainBuilder treats this sentinel as a skip signal: the candidate rule is
// excluded from the output without propagating an error. This allows
// production deployment while Phase 0.5 evaluation is pending.
var ErrCQLPlaceholderResponse = errors.New("reasoning: cql placeholder response (engine pending)")

// EvaluateRuleResult carries the decoded output of a single $evaluate-rule call,
// augmented with the originating RuleID for traceability.
type EvaluateRuleResult struct {
	// RuleID echoes the rule identifier passed to EvaluateRule.
	RuleID string

	// LibraryFound indicates whether the CQL library was located by the runtime.
	LibraryFound bool

	// Status is the evaluation status string from the runtime.
	Status string

	// Triggered reports whether the rule fired for the given resident.
	Triggered bool

	// Type is the recommendation type (e.g. "STOP", "MONITOR", "DOSE_CHANGE").
	Type string

	// Urgency is the urgency tier (e.g. "HIGH", "ROUTINE").
	Urgency string
}

// evaluateRuleResponse is the wire shape returned by kb-cql-runtime's
// $evaluate-rule endpoint.
type evaluateRuleResponse struct {
	Triggered    bool   `json:"triggered"`
	Type         string `json:"type"`
	Urgency      string `json:"urgency"`
	Status       string `json:"status"`
	LibraryFound bool   `json:"library_found"`
}

// HAPIClient is a thin HTTP client over kb-cql-runtime's $evaluate-rule
// endpoint. It mirrors the conventions of the shared cql.Client (5-second
// timeout, body drain, path-escaping) and adds placeholder-detection.
//
// Construct via NewHAPIClient; the zero value is not usable.
type HAPIClient struct {
	baseURL string
	http    *http.Client
}

// NewHAPIClient returns a HAPIClient targeting baseURL (e.g.
// "http://kb-cql-runtime:8095"). The underlying HTTP client carries a
// 5-second request timeout.
func NewHAPIClient(baseURL string) *HAPIClient {
	return &HAPIClient{
		baseURL: baseURL,
		http:    &http.Client{Timeout: defaultTimeout},
	}
}

// EvaluateRule invokes POST /Library/{ruleID}/$evaluate-rule for the given
// residentID and decodes the response.
//
// Returns ErrCQLPlaceholderResponse when the runtime is in Phase 0.5 pending
// state. ChainBuilder skips (continues) on this sentinel.
//
// Errors are returned for:
//   - context cancellation or deadline exceeded
//   - network failures
//   - non-2xx HTTP status codes (status code and body included)
//   - JSON decode failures
func (c *HAPIClient) EvaluateRule(ctx context.Context, ruleID string, residentID uuid.UUID) (*EvaluateRuleResult, error) {
	base, err := url.Parse(c.baseURL)
	if err != nil {
		return nil, fmt.Errorf("reasoning: parse base URL: %w", err)
	}

	escapedRuleID := url.PathEscape(ruleID)
	base.Path = "/Library/" + ruleID + "/$evaluate-rule"
	base.RawPath = "/Library/" + escapedRuleID + "/$evaluate-rule"
	base.RawQuery = url.Values{"residentId": {residentID.String()}}.Encode()

	rawURL := base.String()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, rawURL, nil)
	if err != nil {
		return nil, fmt.Errorf("reasoning: build request: %w", err)
	}

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("reasoning: do request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("reasoning: POST %s: status %d: %s",
			rawURL, resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var wire evaluateRuleResponse
	if err := json.NewDecoder(resp.Body).Decode(&wire); err != nil {
		return nil, fmt.Errorf("reasoning: decode response: %w", err)
	}

	// Phase 0.5 placeholder detection: the engine is not yet wired, treat as skip.
	if wire.Status == placeholderStatus {
		return nil, ErrCQLPlaceholderResponse
	}

	return &EvaluateRuleResult{
		RuleID:       ruleID,
		LibraryFound: wire.LibraryFound,
		Status:       wire.Status,
		Triggered:    wire.Triggered,
		Type:         wire.Type,
		Urgency:      wire.Urgency,
	}, nil
}
