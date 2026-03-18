package services

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-22-hpi-engine/internal/models"
)

// ---------------------------------------------------------------------------
// mockKafkaPublisher — records calls to Publish for assertion.
// ---------------------------------------------------------------------------

type mockKafkaPublisher struct {
	calls []mockKafkaCall
}

type mockKafkaCall struct {
	topic string
	key   string
	event interface{}
}

func (m *mockKafkaPublisher) Publish(_ context.Context, topic string, key string, event interface{}) error {
	m.calls = append(m.calls, mockKafkaCall{topic: topic, key: key, event: event})
	return nil
}

func (m *mockKafkaPublisher) Close() error { return nil }

// ---------------------------------------------------------------------------
// helpers
// ---------------------------------------------------------------------------

func newTestSignalPublisher(kb23URL string, kafka KafkaPublisher, retryCount int, retryDelay time.Duration) *SignalPublisher {
	return NewSignalPublisher(
		kb23URL,
		kafka,
		"clinical.signal.events",
		retryCount,
		retryDelay,
		nil, // db = nil: skip DB updates in tests
		zap.NewNop(),
	)
}

func sampleSignalEvent() *models.ClinicalSignalEvent {
	return &models.ClinicalSignalEvent{
		EventID:      "evt-test-001",
		EventType:    "CLINICAL_SIGNAL",
		SignalType:   models.SignalMonitoringClassification,
		PatientID:    "patient-abc",
		NodeID:       "PM_EGFR_TRAJECTORY",
		NodeVersion:  "1.0.0",
		StratumLabel: "CKD_G3a_DM",
		EmittedAt:    time.Now(),
	}
}

// ---------------------------------------------------------------------------
// Test 1: KB-23 returns 201 → no error returned, event is considered published.
// ---------------------------------------------------------------------------

func TestSignalPublisher_PublishToKB23(t *testing.T) {
	var called bool
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		if r.URL.Path != "/api/v1/clinical-signals" {
			t.Errorf("unexpected path: %s, want /api/v1/clinical-signals", r.URL.Path)
		}
		if ct := r.Header.Get("Content-Type"); ct != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", ct)
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	kafka := &mockKafkaPublisher{}
	p := newTestSignalPublisher(ts.URL, kafka, 3, time.Millisecond)
	event := sampleSignalEvent()

	err := p.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish() returned error = %v, want nil", err)
	}
	if !called {
		t.Error("KB-23 server was never called")
	}
}

// ---------------------------------------------------------------------------
// Test 2: Kafka publisher records the call with the correct topic and key.
// ---------------------------------------------------------------------------

func TestSignalPublisher_PublishToKafka(t *testing.T) {
	// Use a simple 200 server so the KB-23 POST succeeds without noise.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	kafka := &mockKafkaPublisher{}
	p := newTestSignalPublisher(ts.URL, kafka, 1, time.Millisecond)
	event := sampleSignalEvent()

	err := p.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish() returned error = %v, want nil", err)
	}

	if len(kafka.calls) != 1 {
		t.Fatalf("kafka.calls count = %d, want 1", len(kafka.calls))
	}
	call := kafka.calls[0]
	if call.topic != "clinical.signal.events" {
		t.Errorf("kafka topic = %q, want %q", call.topic, "clinical.signal.events")
	}
	if call.key != event.PatientID {
		t.Errorf("kafka key = %q, want %q (PatientID)", call.key, event.PatientID)
	}
}

// ---------------------------------------------------------------------------
// Test 3: KB-23 returns 204 → no error, event logged as "no card needed".
// ---------------------------------------------------------------------------

func TestSignalPublisher_KB23Returns204(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	kafka := &mockKafkaPublisher{}
	p := newTestSignalPublisher(ts.URL, kafka, 3, time.Millisecond)
	event := sampleSignalEvent()

	err := p.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish() returned error = %v, want nil (204 should be treated as success)", err)
	}
	// Kafka should still have been called once.
	if len(kafka.calls) != 1 {
		t.Errorf("kafka.calls count = %d, want 1", len(kafka.calls))
	}
}

// ---------------------------------------------------------------------------
// Test 4: KB-23 returns 500 on the first call, 201 on the second → published
// after one retry.
// ---------------------------------------------------------------------------

func TestSignalPublisher_KB23Retry(t *testing.T) {
	var callCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		n := atomic.AddInt32(&callCount, 1)
		if n == 1 {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusCreated)
	}))
	defer ts.Close()

	kafka := &mockKafkaPublisher{}
	// retryCount=3, retryDelay=1ms — fast for test
	p := newTestSignalPublisher(ts.URL, kafka, 3, time.Millisecond)
	event := sampleSignalEvent()

	err := p.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish() returned error = %v, want nil", err)
	}

	if n := atomic.LoadInt32(&callCount); n < 2 {
		t.Errorf("KB-23 call count = %d, want >= 2 (first fail, second succeed)", n)
	}
}

// ---------------------------------------------------------------------------
// Test 5: KB-23 always returns 500 → all retries exhausted → Publish still
// returns nil, no panic.
// ---------------------------------------------------------------------------

func TestSignalPublisher_KB23FailAfterRetries(t *testing.T) {
	var callCount int32

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&callCount, 1)
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	kafka := &mockKafkaPublisher{}
	retryCount := 3
	p := newTestSignalPublisher(ts.URL, kafka, retryCount, time.Millisecond)
	event := sampleSignalEvent()

	err := p.Publish(context.Background(), event)
	if err != nil {
		t.Fatalf("Publish() returned error = %v, want nil (failures must be non-fatal)", err)
	}

	// All retries should have been attempted.
	if n := atomic.LoadInt32(&callCount); int(n) != retryCount {
		t.Errorf("KB-23 call count = %d, want %d (all retries)", n, retryCount)
	}

	// Kafka should still have been called even though KB-23 failed.
	if len(kafka.calls) != 1 {
		t.Errorf("kafka.calls count = %d, want 1 (Kafka is independent of KB-23 success)", len(kafka.calls))
	}
}
