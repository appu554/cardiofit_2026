package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go.uber.org/zap"

	"kb-23-decision-cards/internal/models"
)

func TestKB26BPContextClient_Classify_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/v1/kb26/bp-context/p1" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		resp := map[string]interface{}{
			"success": true,
			"data": models.BPContextClassification{
				PatientID:     "p1",
				Phenotype:     models.PhenotypeMaskedHTN,
				ClinicSBPMean: 128,
				HomeSBPMean:   148,
				Confidence:    "HIGH",
			},
		}
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewKB26BPContextClient(server.URL, 1*time.Second, zap.NewNop())
	result, err := client.Classify(context.Background(), "p1")
	if err != nil {
		t.Fatalf("Classify failed: %v", err)
	}
	if result == nil {
		t.Fatal("expected non-nil result")
	}
	if result.Phenotype != models.PhenotypeMaskedHTN {
		t.Errorf("expected MASKED_HTN, got %s", result.Phenotype)
	}
}

func TestKB26BPContextClient_Classify_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewKB26BPContextClient(server.URL, 1*time.Second, zap.NewNop())
	result, err := client.Classify(context.Background(), "ghost")
	if err != nil {
		t.Errorf("404 should be nil result, no error; got err=%v", err)
	}
	if result != nil {
		t.Errorf("expected nil for 404, got %+v", result)
	}
}

func TestKB26BPContextClient_Classify_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewKB26BPContextClient(server.URL, 1*time.Second, zap.NewNop())
	_, err := client.Classify(context.Background(), "p1")
	if err == nil {
		t.Error("expected error on 500")
	}
}
