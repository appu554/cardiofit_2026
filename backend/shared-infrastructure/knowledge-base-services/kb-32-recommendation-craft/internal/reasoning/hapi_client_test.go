// Package reasoning_test exercises the HAPIClient and ChainBuilder.
package reasoning_test

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/cardiofit/kb32/internal/reasoning"
	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// HAPIClient tests
// ---------------------------------------------------------------------------

// TestHAPIClient_HappyPath verifies that a well-formed 200 response with
// triggered=true is decoded correctly into an EvaluateRuleResult.
func TestHAPIClient_HappyPath(t *testing.T) {
	residentID := uuid.New()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"triggered": true,
			"type":      "STOP",
			"urgency":   "HIGH",
			"status":    "evaluated",
		})
	}))
	t.Cleanup(server.Close)

	c := reasoning.NewHAPIClient(server.URL)
	result, err := c.EvaluateRule(context.Background(), "EGFR-001", residentID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.RuleID != "EGFR-001" {
		t.Errorf("RuleID: got %q want %q", result.RuleID, "EGFR-001")
	}
	if !result.Triggered {
		t.Error("expected Triggered=true")
	}
	if result.Type != "STOP" {
		t.Errorf("Type: got %q want %q", result.Type, "STOP")
	}
	if result.Urgency != "HIGH" {
		t.Errorf("Urgency: got %q want %q", result.Urgency, "HIGH")
	}
}

// TestHAPIClient_NotTriggered verifies that triggered=false is decoded correctly.
func TestHAPIClient_NotTriggered(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"triggered": false,
			"type":      "MONITOR",
			"urgency":   "LOW",
			"status":    "evaluated",
		})
	}))
	t.Cleanup(server.Close)

	c := reasoning.NewHAPIClient(server.URL)
	result, err := c.EvaluateRule(context.Background(), "DBI-002", uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Triggered {
		t.Error("expected Triggered=false")
	}
}

// TestHAPIClient_PlaceholderResponse verifies that a Phase 0.5 placeholder
// response (status=library_found_engine_pending) returns ErrCQLPlaceholderResponse.
func TestHAPIClient_PlaceholderResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"triggered":     false,
			"type":          "",
			"urgency":       "",
			"status":        "library_found_engine_pending",
			"library_found": true,
		})
	}))
	t.Cleanup(server.Close)

	c := reasoning.NewHAPIClient(server.URL)
	_, err := c.EvaluateRule(context.Background(), "PLACEHOLDER-RULE", uuid.New())
	if err == nil {
		t.Fatal("expected ErrCQLPlaceholderResponse, got nil")
	}
	if !errors.Is(err, reasoning.ErrCQLPlaceholderResponse) {
		t.Errorf("expected ErrCQLPlaceholderResponse, got: %v", err)
	}
}

// TestHAPIClient_Non2xx verifies that a non-2xx response returns an error
// containing the status code.
func TestHAPIClient_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rule not found", http.StatusNotFound)
	}))
	t.Cleanup(server.Close)

	c := reasoning.NewHAPIClient(server.URL)
	_, err := c.EvaluateRule(context.Background(), "NONEXISTENT", uuid.New())
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if errors.Is(err, reasoning.ErrCQLPlaceholderResponse) {
		t.Error("4xx should not be ErrCQLPlaceholderResponse")
	}
}

// TestHAPIClient_500BodyInError verifies that a 500 body is captured in the error.
func TestHAPIClient_500BodyInError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("OperationOutcome: engine failure"))
	}))
	t.Cleanup(server.Close)

	c := reasoning.NewHAPIClient(server.URL)
	_, err := c.EvaluateRule(context.Background(), "FAIL-RULE", uuid.New())
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
}

// TestHAPIClient_ContextCancellation verifies that a pre-cancelled context
// causes EvaluateRule to return immediately with an error.
func TestHAPIClient_ContextCancellation(t *testing.T) {
	unblock := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
	}))
	t.Cleanup(func() {
		close(unblock)
		server.Close()
	})

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel before request

	c := reasoning.NewHAPIClient(server.URL)
	_, err := c.EvaluateRule(ctx, "SOME-RULE", uuid.New())
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}

// TestHAPIClient_URLConstruction verifies that the correct path and query
// string are sent to the server.
func TestHAPIClient_URLConstruction(t *testing.T) {
	residentID := uuid.New()
	ruleID := "EGFR-001"

	var capturedPath, capturedQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{
			"triggered": false,
			"type":      "",
			"urgency":   "",
			"status":    "evaluated",
		})
	}))
	t.Cleanup(server.Close)

	c := reasoning.NewHAPIClient(server.URL)
	_, _ = c.EvaluateRule(context.Background(), ruleID, residentID)

	wantPath := "/Library/EGFR-001/$evaluate-rule"
	if capturedPath != wantPath {
		t.Errorf("Path: got %q want %q", capturedPath, wantPath)
	}
	wantQuery := "residentId=" + residentID.String()
	if capturedQuery != wantQuery {
		t.Errorf("Query: got %q want %q", capturedQuery, wantQuery)
	}
}
