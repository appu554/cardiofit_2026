package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// ---------------------------------------------------------------------------
// GET /api/v1/config/risk-scoring
// ---------------------------------------------------------------------------

func TestGetRiskScoringConfig_StatusOK(t *testing.T) {
	s := testServer()
	s.Router.GET("/api/v1/config/risk-scoring", s.getRiskScoringConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/risk-scoring", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestGetRiskScoringConfig_ResponseShape(t *testing.T) {
	s := testServer()
	s.Router.GET("/api/v1/config/risk-scoring", s.getRiskScoringConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/risk-scoring", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var body RiskScoringConfig
	if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	// daily_risk_weights
	if body.DailyRiskWeights.VitalStability != 0.40 {
		t.Errorf("vital_stability: want 0.40, got %f", body.DailyRiskWeights.VitalStability)
	}
	if body.DailyRiskWeights.LabAbnormality != 0.35 {
		t.Errorf("lab_abnormality: want 0.35, got %f", body.DailyRiskWeights.LabAbnormality)
	}
	if body.DailyRiskWeights.MedicationComplexity != 0.25 {
		t.Errorf("medication_complexity: want 0.25, got %f", body.DailyRiskWeights.MedicationComplexity)
	}

	// risk_levels
	if len(body.RiskLevels) != 4 {
		t.Fatalf("risk_levels: want 4 entries, got %d", len(body.RiskLevels))
	}
	expectedLevels := []struct {
		name   string
		min    int
		max    int
		action string
	}{
		{"LOW", 0, 24, "routine monitoring"},
		{"MODERATE", 25, 49, "enhanced monitoring"},
		{"HIGH", 50, 74, "frequent assessment"},
		{"CRITICAL", 75, 100, "ICU-level monitoring"},
	}
	for i, exp := range expectedLevels {
		got := body.RiskLevels[i]
		if got.Name != exp.name || got.Min != exp.min || got.Max != exp.max || got.Action != exp.action {
			t.Errorf("risk_levels[%d]: want %+v, got %+v", i, exp, got)
		}
	}

	// alert_severity_scores spot checks
	if score, ok := body.AlertSeverityScores["CARDIAC_ARREST"]; !ok || score != 10 {
		t.Errorf("alert_severity_scores CARDIAC_ARREST: want 10, got %d (present=%v)", score, ok)
	}
	if score, ok := body.AlertSeverityScores["MEDICATION_ALERT"]; !ok || score != 3 {
		t.Errorf("alert_severity_scores MEDICATION_ALERT: want 3, got %d (present=%v)", score, ok)
	}
	if len(body.AlertSeverityScores) != 11 {
		t.Errorf("alert_severity_scores: want 11 entries, got %d", len(body.AlertSeverityScores))
	}

	// time_sensitivity_scores spot checks
	if score, ok := body.TimeSensitivityScores["CARDIAC_ARREST"]; !ok || score != 5 {
		t.Errorf("time_sensitivity_scores CARDIAC_ARREST: want 5, got %d (present=%v)", score, ok)
	}
	if len(body.TimeSensitivityScores) != 9 {
		t.Errorf("time_sensitivity_scores: want 9 entries, got %d", len(body.TimeSensitivityScores))
	}

	// patient_vulnerability
	pv := body.PatientVulnerability
	if pv.Age75Plus != 2 {
		t.Errorf("age_75_plus: want 2, got %d", pv.Age75Plus)
	}
	if pv.Age65Plus != 1 {
		t.Errorf("age_65_plus: want 1, got %d", pv.Age65Plus)
	}
	if len(pv.HighRiskConditions) != 3 {
		t.Errorf("high_risk_conditions: want 3 entries, got %d", len(pv.HighRiskConditions))
	}
	if pv.HighRiskConditionBonus != 1 {
		t.Errorf("high_risk_condition_bonus: want 1, got %d", pv.HighRiskConditionBonus)
	}
	if pv.NEWS2GTE5Baseline != 1 {
		t.Errorf("news2_gte_5_baseline: want 1, got %d", pv.NEWS2GTE5Baseline)
	}

	// version
	if body.Version != "2026-03-23T00:00:00Z" {
		t.Errorf("version: want 2026-03-23T00:00:00Z, got %s", body.Version)
	}
}

func TestGetRiskScoringConfig_ContentTypeJSON(t *testing.T) {
	s := testServer()
	s.Router.GET("/api/v1/config/risk-scoring", s.getRiskScoringConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/risk-scoring", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json; charset=utf-8" {
		t.Errorf("Content-Type: want application/json; charset=utf-8, got %s", ct)
	}
}

func TestGetRiskScoringConfig_JSONFieldNames(t *testing.T) {
	s := testServer()
	s.Router.GET("/api/v1/config/risk-scoring", s.getRiskScoringConfig)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/config/risk-scoring", nil)
	w := httptest.NewRecorder()
	s.Router.ServeHTTP(w, req)

	var raw map[string]interface{}
	if err := json.NewDecoder(w.Body).Decode(&raw); err != nil {
		t.Fatalf("failed to decode response: %v", err)
	}

	requiredKeys := []string{
		"daily_risk_weights",
		"risk_levels",
		"alert_severity_scores",
		"time_sensitivity_scores",
		"patient_vulnerability",
		"version",
	}
	for _, key := range requiredKeys {
		if _, ok := raw[key]; !ok {
			t.Errorf("missing top-level key %q in response", key)
		}
	}
}
