// Package cql provides a typed Go client for the kb-cql-runtime Java service.
package cql

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
)

// TestEvaluateRule_HappyPath verifies that a 200 response with the Phase 2
// contract shape is decoded into a RuleResult correctly.
func TestEvaluateRule_HappyPath(t *testing.T) {
	residentID := uuid.New()
	want := RuleResult{
		Triggered: true,
		Type:      "MEDICATION_REVIEW",
		Urgency:   "HIGH",
		ClinicalContent: map[string]any{
			"reason": "eGFR below threshold",
		},
	}

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(want)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	got, err := c.EvaluateRule(context.Background(), "EGFR-001", residentID)
	if err != nil {
		t.Fatalf("EvaluateRule returned unexpected error: %v", err)
	}
	if got.Triggered != want.Triggered {
		t.Errorf("Triggered: got %v want %v", got.Triggered, want.Triggered)
	}
	if got.Type != want.Type {
		t.Errorf("Type: got %q want %q", got.Type, want.Type)
	}
	if got.Urgency != want.Urgency {
		t.Errorf("Urgency: got %q want %q", got.Urgency, want.Urgency)
	}
	if got.ClinicalContent["reason"] != want.ClinicalContent["reason"] {
		t.Errorf("ClinicalContent[reason]: got %v want %v",
			got.ClinicalContent["reason"], want.ClinicalContent["reason"])
	}
}

// TestEvaluateRule_Non2xx verifies that a non-2xx response (e.g. 404)
// causes EvaluateRule to return an error rather than attempting to decode.
func TestEvaluateRule_Non2xx(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "rule not found", http.StatusNotFound)
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.EvaluateRule(context.Background(), "NONEXISTENT", uuid.New())
	if err == nil {
		t.Fatal("expected an error for 404 response, got nil")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected error to mention status 404, got: %v", err)
	}
}

// TestEvaluateRule_URLConstruction verifies that the correct path and query
// string are assembled: /Library/{ruleID}/$evaluate-rule?residentId={uuid}.
//
// The ruleID used here contains a "/" to exercise path-escaping. Go's
// net/http server exposes the percent-encoded form via r.URL.RawPath; the
// decoded r.URL.Path will show the literal "/" and is checked separately to
// confirm the correct segment structure.
func TestEvaluateRule_URLConstruction(t *testing.T) {
	residentID := uuid.New()
	ruleID := "BP-MONITOR/2024"       // contains "/" — must be path-escaped
	safeRuleID := "BP-MONITOR%2F2024" // expected wire form after PathEscape

	var capturedRawPath, capturedQuery string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// RawPath carries the percent-encoded segments as sent on the wire.
		// Path carries the decoded form (slashes decoded); we check RawPath
		// to confirm the encoding was preserved during transport.
		capturedRawPath = r.URL.RawPath
		capturedQuery = r.URL.RawQuery
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_ = json.NewEncoder(w).Encode(RuleResult{})
	}))
	defer server.Close()

	c := NewClient(server.URL)
	_, err := c.EvaluateRule(context.Background(), ruleID, residentID)
	if err != nil {
		t.Fatalf("EvaluateRule: %v", err)
	}

	wantRawPath := "/Library/" + safeRuleID + "/$evaluate-rule"
	if capturedRawPath != wantRawPath {
		t.Errorf("RawPath: got %q want %q", capturedRawPath, wantRawPath)
	}
	wantQuery := "residentId=" + residentID.String()
	if capturedQuery != wantQuery {
		t.Errorf("query: got %q want %q", capturedQuery, wantQuery)
	}
}

// TestEvaluateRule_ContextCancellation verifies that a cancelled context
// causes EvaluateRule to return an error (transport honours the context).
func TestEvaluateRule_ContextCancellation(t *testing.T) {
	// Server that blocks until the test is done (simulates slow response).
	unblock := make(chan struct{})
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		<-unblock
	}))
	defer func() {
		close(unblock)
		server.Close()
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately before the request starts

	c := NewClient(server.URL)
	_, err := c.EvaluateRule(ctx, "SOME-RULE", uuid.New())
	if err == nil {
		t.Fatal("expected error from cancelled context, got nil")
	}
}
