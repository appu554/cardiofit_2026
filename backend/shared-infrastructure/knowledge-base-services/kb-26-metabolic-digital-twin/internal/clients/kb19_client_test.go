package clients

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"
)

func TestKB19Client_PublishPhenotypeChanged_PostsCorrectEnvelope(t *testing.T) {
	var receivedPath string
	var receivedBody KB19Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewKB19Client(server.URL, 1*time.Second, zap.NewNop(), nil)
	err := client.PublishPhenotypeChanged(context.Background(), "p1", "WHITE_COAT_HTN", "SUSTAINED_HTN")
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if receivedPath != "/api/v1/events" {
		t.Errorf("expected path /api/v1/events, got %s", receivedPath)
	}
	if receivedBody.EventType != "BP_PHENOTYPE_CHANGED" {
		t.Errorf("expected BP_PHENOTYPE_CHANGED, got %s", receivedBody.EventType)
	}
	if receivedBody.PatientID != "p1" {
		t.Errorf("expected patient p1, got %s", receivedBody.PatientID)
	}
	if receivedBody.OldPhenotype != "WHITE_COAT_HTN" || receivedBody.NewPhenotype != "SUSTAINED_HTN" {
		t.Errorf("phenotype fields wrong: %+v", receivedBody)
	}
}

func TestKB19Client_PublishMaskedHTNDetected_IncludesUrgency(t *testing.T) {
	var receivedBody KB19Event

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&receivedBody)
		w.WriteHeader(http.StatusAccepted)
	}))
	defer server.Close()

	client := NewKB19Client(server.URL, 1*time.Second, zap.NewNop(), nil)
	err := client.PublishMaskedHTNDetected(context.Background(), "p1", "MASKED_HTN", "IMMEDIATE")
	if err != nil {
		t.Fatalf("publish failed: %v", err)
	}
	if receivedBody.EventType != "MASKED_HTN_DETECTED" {
		t.Errorf("expected MASKED_HTN_DETECTED, got %s", receivedBody.EventType)
	}
	if receivedBody.BPPhenotype != "MASKED_HTN" {
		t.Errorf("expected BPPhenotype MASKED_HTN, got %s", receivedBody.BPPhenotype)
	}
	if receivedBody.Urgency != "IMMEDIATE" {
		t.Errorf("expected Urgency IMMEDIATE, got %s", receivedBody.Urgency)
	}
}

func TestKB19Client_PublishPhenotypeChanged_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewKB19Client(server.URL, 1*time.Second, zap.NewNop(), nil)
	err := client.PublishPhenotypeChanged(context.Background(), "p1", "WHITE_COAT_HTN", "SUSTAINED_HTN")
	if err == nil {
		t.Error("expected error on 500")
	}
}

func TestKB19Client_NetworkError_DoesNotPanic(t *testing.T) {
	// Pointing at an unreachable URL should return an error, not panic.
	client := NewKB19Client("http://127.0.0.1:1", 100*time.Millisecond, zap.NewNop(), nil)
	err := client.PublishPhenotypeChanged(context.Background(), "p1", "WHITE_COAT_HTN", "SUSTAINED_HTN")
	if err == nil {
		t.Error("expected error from unreachable server")
	}
}
