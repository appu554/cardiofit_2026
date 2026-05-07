package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"kb-scope-rules/internal/dsl"
	"kb-scope-rules/internal/store"
)

func TestServer_Health(t *testing.T) {
	srv := &Server{Store: store.NewMemoryStore()}
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "ok", body["status"])
	assert.Equal(t, "kb-31-scope-rules", body["service"])
}

func TestServer_ListByJurisdiction(t *testing.T) {
	s := store.NewMemoryStore()
	rule := dsl.ScopeRule{
		RuleID: "R-VIC", Jurisdiction: "AU/VIC",
		Category: "medication_administration_scope_restriction",
		Status:   dsl.StatusActive,
		EffectivePeriod: dsl.EffectivePeriod{
			StartDate: time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC),
		},
		AppliesTo:  dsl.AppliesToScope{Role: "rn", ActionClass: dsl.ActionAdminister},
		Evaluation: dsl.EvaluationBlock{Decision: dsl.DecisionDenied},
		Audit:      dsl.AuditBlock{LegislativeReference: "test"},
	}
	_, err := s.Insert(context.Background(), rule, []byte("y"))
	require.NoError(t, err)

	srv := &Server{Store: s}
	req := httptest.NewRequest(http.MethodGet,
		"/v1/scope-rules?jurisdiction=AU/VIC&at=2026-08-01T00:00:00Z", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)
	var body map[string]any
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, float64(1), body["count"])
}

func TestServer_ListMissingJurisdiction(t *testing.T) {
	srv := &Server{Store: store.NewMemoryStore()}
	req := httptest.NewRequest(http.MethodGet, "/v1/scope-rules", nil)
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestServer_PostInsertValidYAML(t *testing.T) {
	srv := &Server{Store: store.NewMemoryStore()}
	yamlBody := []byte(`
scope_rule:
  rule_id: TEST-INSERT-1
  jurisdiction: AU
  category: prescriber_scope
  status: ACTIVE
  effective_period:
    start_date: 2026-07-01T00:00:00Z
  applies_to:
    role: rn
    action_class: prescribe
  evaluation:
    decision: granted
  audit:
    legislative_reference: test legislation
`)
	req := httptest.NewRequest(http.MethodPost, "/v1/scope-rules", bytes.NewReader(yamlBody))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	require.Equal(t, http.StatusCreated, rec.Code, rec.Body.String())
	var body map[string]string
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &body))
	assert.Equal(t, "TEST-INSERT-1", body["rule_id"])
}

func TestServer_PostInsertInvalidYAML(t *testing.T) {
	srv := &Server{Store: store.NewMemoryStore()}
	yamlBody := []byte(`scope_rule:
  rule_id: ""
  jurisdiction: ""
`)
	req := httptest.NewRequest(http.MethodPost, "/v1/scope-rules", bytes.NewReader(yamlBody))
	rec := httptest.NewRecorder()
	srv.Routes().ServeHTTP(rec, req)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}
