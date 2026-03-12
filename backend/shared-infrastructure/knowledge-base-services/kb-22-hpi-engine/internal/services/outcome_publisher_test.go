package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/config"
	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// T8: OutcomePublisher and event contract tests
// ---------------------------------------------------------------------------

func TestHPICompleteEvent_Serialization(t *testing.T) {
	event := models.HPICompleteEvent{
		EventType:    models.EventHPIComplete,
		PatientID:    uuid.New(),
		SessionID:    uuid.New(),
		NodeID:       "P1_CHEST_PAIN",
		StratumLabel: "CKD_G3a_DM",
		TopDiagnosis: "ACS",
		TopPosterior: 0.72,
		RankedDifferentials: []models.DifferentialEntry{
			{DifferentialID: "ACS", PosteriorProbability: 0.72},
			{DifferentialID: "PE", PosteriorProbability: 0.15},
		},
		SafetyFlags: []models.SafetyFlagSummary{
			{FlagID: "ST_ELEVATION", Severity: "IMMEDIATE", RecommendedAction: "Activate cath lab"},
		},
		CMLogDeltasApplied: map[string]float64{"CM_CKD_HF": 0.35},
		GuidelinePriorRefs: []string{"KDIGO-2024-CKD-DM"},
		ConvergenceReached: true,
		CompletedAt:        time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded models.HPICompleteEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.EventType != models.EventHPIComplete {
		t.Errorf("event_type = %s, want %s", decoded.EventType, models.EventHPIComplete)
	}
	if decoded.TopDiagnosis != "ACS" {
		t.Errorf("top_diagnosis = %s, want ACS", decoded.TopDiagnosis)
	}
	if len(decoded.RankedDifferentials) != 2 {
		t.Errorf("ranked_differentials count = %d, want 2", len(decoded.RankedDifferentials))
	}
	if len(decoded.SafetyFlags) != 1 {
		t.Errorf("safety_flags count = %d, want 1", len(decoded.SafetyFlags))
	}
	if !decoded.ConvergenceReached {
		t.Error("convergence_reached should be true")
	}
}

func TestSafetyAlertEvent_Serialization(t *testing.T) {
	event := models.SafetyAlertEvent{
		EventType:         models.EventSafetyAlert,
		PatientID:         uuid.New(),
		SessionID:         uuid.New(),
		FlagID:            "ST_ELEVATION",
		Severity:          "IMMEDIATE",
		RecommendedAction: "Activate cath lab protocol",
		FiredAt:           time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded models.SafetyAlertEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.EventType != models.EventSafetyAlert {
		t.Errorf("event_type = %s, want %s", decoded.EventType, models.EventSafetyAlert)
	}
	if decoded.FlagID != "ST_ELEVATION" {
		t.Errorf("flag_id = %s, want ST_ELEVATION", decoded.FlagID)
	}
	if decoded.Severity != "IMMEDIATE" {
		t.Errorf("severity = %s, want IMMEDIATE", decoded.Severity)
	}
}

func TestStratumDriftEvent_Serialization(t *testing.T) {
	ckdOld := "G3a"
	ckdNew := "G3b"
	event := models.StratumDriftEvent{
		EventType:      models.EventStratumDrifted,
		PatientID:      uuid.New(),
		SessionID:      uuid.New(),
		OldStratum:     "CKD_G3a_DM",
		NewStratum:     "CKD_G3b_DM",
		OldCKDSubstage: &ckdOld,
		NewCKDSubstage: &ckdNew,
		DetectedAt:     time.Now(),
	}

	data, err := json.Marshal(event)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded models.StratumDriftEvent
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.OldStratum != "CKD_G3a_DM" {
		t.Errorf("old_stratum = %s, want CKD_G3a_DM", decoded.OldStratum)
	}
	if decoded.NewStratum != "CKD_G3b_DM" {
		t.Errorf("new_stratum = %s, want CKD_G3b_DM", decoded.NewStratum)
	}
	if decoded.OldCKDSubstage == nil || *decoded.OldCKDSubstage != "G3a" {
		t.Error("old_ckd_substage mismatch")
	}
}

func TestOutcomePublisher_PublishSafetyAlert_Success(t *testing.T) {
	// Create a test HTTP server that accepts the safety alert
	var receivedBody []byte
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/events" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.Header.Get("Content-Type") != "application/json" {
			t.Errorf("content-type = %s, want application/json", r.Header.Get("Content-Type"))
		}
		body := make([]byte, r.ContentLength)
		r.Body.Read(body)
		receivedBody = body
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	cfg := &config.Config{
		KB19URL:               ts.URL,
		SafetyAlertRetryDelay: 10 * time.Millisecond,
	}

	publisher := NewOutcomePublisher(cfg, zap.NewNop(), testMetrics())

	event := models.SafetyAlertEvent{
		PatientID:         uuid.New(),
		SessionID:         uuid.New(),
		FlagID:            "TROPONIN_HIGH",
		Severity:          "URGENT",
		RecommendedAction: "Order serial troponin",
		FiredAt:           time.Now(),
	}

	err := publisher.PublishSafetyAlert(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(receivedBody) == 0 {
		t.Fatal("server received empty body")
	}

	var decoded models.SafetyAlertEvent
	if err := json.Unmarshal(receivedBody, &decoded); err != nil {
		t.Fatalf("unmarshal received body: %v", err)
	}
	if decoded.EventType != models.EventSafetyAlert {
		t.Errorf("received event_type = %s, want SAFETY_ALERT", decoded.EventType)
	}
}

func TestOutcomePublisher_PublishSafetyAlert_ServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("internal error"))
	}))
	defer ts.Close()

	cfg := &config.Config{
		KB19URL:               ts.URL,
		SafetyAlertRetryDelay: 1 * time.Millisecond, // fast retry for test
	}

	publisher := NewOutcomePublisher(cfg, zap.NewNop(), testMetrics())

	event := models.SafetyAlertEvent{
		PatientID: uuid.New(),
		SessionID: uuid.New(),
		FlagID:    "ST_ELEVATION",
		Severity:  "IMMEDIATE",
		FiredAt:   time.Now(),
	}

	err := publisher.PublishSafetyAlert(context.Background(), event)
	if err == nil {
		t.Fatal("expected error from 500 response, got nil")
	}
}

func TestOutcomePublisher_PublishHPIComplete_DualTarget(t *testing.T) {
	// Track which targets were called
	var kb23Called, kb19Called bool

	kb23Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kb23Called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer kb23Server.Close()

	kb19Server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		kb19Called = true
		w.WriteHeader(http.StatusOK)
	}))
	defer kb19Server.Close()

	cfg := &config.Config{
		KB23URL:           kb23Server.URL,
		KB19URL:           kb19Server.URL,
		OutcomeRetryDelay: 1 * time.Millisecond,
	}

	publisher := NewOutcomePublisher(cfg, zap.NewNop(), testMetrics())

	event := models.HPICompleteEvent{
		PatientID:    uuid.New(),
		SessionID:    uuid.New(),
		NodeID:       "P1_CHEST_PAIN",
		TopDiagnosis: "ACS",
		TopPosterior: 0.72,
		RankedDifferentials: []models.DifferentialEntry{
			{DifferentialID: "ACS", PosteriorProbability: 0.72},
		},
		CompletedAt: time.Now(),
	}

	err := publisher.PublishHPIComplete(context.Background(), event)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !kb23Called {
		t.Error("KB-23 server was not called")
	}
	if !kb19Called {
		t.Error("KB-19 server was not called")
	}
}
