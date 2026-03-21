package fhirclient

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"go.uber.org/zap"
)

func testLogger() *zap.Logger {
	logger, _ := zap.NewDevelopment()
	return logger
}

func TestGoogleFHIRConfig_BaseURL(t *testing.T) {
	cfg := GoogleFHIRConfig{
		ProjectID:   "cardiofit-905a8",
		Location:    "asia-south1",
		DatasetID:   "clinical-synthesis-hub",
		FhirStoreID: "fhir-store",
	}
	want := "https://healthcare.googleapis.com/v1/projects/cardiofit-905a8/locations/asia-south1/datasets/clinical-synthesis-hub/fhirStores/fhir-store/fhir"
	if got := cfg.BaseURL(); got != want {
		t.Errorf("BaseURL() = %q, want %q", got, want)
	}
}

func TestClient_Create(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/Patient" {
			t.Errorf("expected /Patient, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusCreated)
		w.Write([]byte(`{"resourceType":"Patient","id":"123"}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Create("Patient", []byte(`{"resourceType":"Patient"}`))
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}
	if string(data) != `{"resourceType":"Patient","id":"123"}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_Read(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/Patient/123" {
			t.Errorf("expected /Patient/123, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resourceType":"Patient","id":"123"}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Read("Patient", "123")
	if err != nil {
		t.Fatalf("Read failed: %v", err)
	}
	if string(data) != `{"resourceType":"Patient","id":"123"}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_Read_NotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	_, err := client.Read("Patient", "999")
	if err == nil {
		t.Fatal("expected error for 404")
	}
}

func TestClient_RetryOn429(t *testing.T) {
	attempts := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempts++
		if attempts < 3 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resourceType":"Patient","id":"123"}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Read("Patient", "123")
	if err != nil {
		t.Fatalf("expected success after retries, got: %v", err)
	}
	if attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", attempts)
	}
	if string(data) != `{"resourceType":"Patient","id":"123"}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_Search(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("patient") != "123" {
			t.Errorf("expected patient=123 param")
		}
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"resourceType":"Bundle","total":1}`))
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	data, err := client.Search("Observation", map[string]string{"patient": "123"})
	if err != nil {
		t.Fatalf("Search failed: %v", err)
	}
	if string(data) != `{"resourceType":"Bundle","total":1}` {
		t.Errorf("unexpected response: %s", string(data))
	}
}

func TestClient_HealthCheck(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/metadata" {
			t.Errorf("expected /metadata, got %s", r.URL.Path)
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := NewWithHTTPClient(srv.URL, srv.Client(), testLogger())
	if err := client.HealthCheck(); err != nil {
		t.Fatalf("HealthCheck failed: %v", err)
	}
}
