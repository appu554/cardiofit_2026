package clients

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"kb-patient-profile/internal/config"
)

func newTestKB25Client(serverURL string) *KB25Client {
	return &KB25Client{
		baseURL: serverURL,
		httpClient: &http.Client{
			Timeout: 2 * time.Second,
		},
		logger: zap.NewNop(),
	}
}

// ---------------------------------------------------------------------------
// CheckSafety
// ---------------------------------------------------------------------------

func TestCheckSafety_Safe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/safety/check", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var req SafetyCheckRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "patient-1", req.PatientID)
		assert.Equal(t, "M3-PRP", req.ProtocolID)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SafetyCheckResponse{Safe: true})
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)

	resp, err := client.CheckSafety(SafetyCheckRequest{
		PatientID:  "patient-1",
		ProtocolID: "M3-PRP",
		Conditions: map[string]float64{"egfr": 45.0},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.True(t, resp.Safe)
	assert.Empty(t, resp.RuleCode)
}

func TestCheckSafety_Unsafe(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(SafetyCheckResponse{
			Safe:     false,
			RuleCode: "LS-01",
			Reason:   "eGFR < 15 — high-intensity protein protocol contraindicated",
		})
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)

	resp, err := client.CheckSafety(SafetyCheckRequest{
		PatientID:  "patient-2",
		ProtocolID: "M3-PRP",
		Conditions: map[string]float64{"egfr": 12.0},
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.False(t, resp.Safe)
	assert.Equal(t, "LS-01", resp.RuleCode)
	assert.NotEmpty(t, resp.Reason)
}

func TestCheckSafety_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)

	resp, err := client.CheckSafety(SafetyCheckRequest{
		PatientID:  "patient-3",
		ProtocolID: "M3-PRP",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "500")
	assert.Nil(t, resp)
}

func TestCheckSafety_NetworkError(t *testing.T) {
	// Point at a port where nothing is listening.
	client := newTestKB25Client("http://127.0.0.1:19999")

	resp, err := client.CheckSafety(SafetyCheckRequest{
		PatientID:  "patient-4",
		ProtocolID: "M3-PRP",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "KB-25 safety check request failed")
	assert.Nil(t, resp)
}

func TestCheckSafety_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("not-json{{{"))
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)

	resp, err := client.CheckSafety(SafetyCheckRequest{
		PatientID:  "patient-5",
		ProtocolID: "M3-PRP",
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode KB-25 safety check response")
	assert.Nil(t, resp)
}

// ---------------------------------------------------------------------------
// ProjectCombined
// ---------------------------------------------------------------------------

func TestProjectCombined_Success(t *testing.T) {
	expected := ProjectionResponse{
		Synergy: 1.15,
		Projections: []ProjectedOutcome{
			{Metric: "hba1c", BaselineDelta: -0.8, Confidence: 0.82},
			{Metric: "weight_kg", BaselineDelta: -3.2, Confidence: 0.75},
		},
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/v1/project-combined", r.URL.Path)

		var req ProjectionRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "patient-1", req.PatientID)
		assert.Equal(t, []string{"M3-PRP"}, req.ProtocolIDs)
		assert.Equal(t, 90, req.HorizonDays)

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(expected)
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)

	resp, err := client.ProjectCombined(ProjectionRequest{
		PatientID:   "patient-1",
		ProtocolIDs: []string{"M3-PRP"},
		HorizonDays: 90,
		Age:         55,
		EGFR:        48.0,
		BMI:         28.5,
		Adherence:   0.75,
	})

	require.NoError(t, err)
	require.NotNil(t, resp)
	assert.Equal(t, 1.15, resp.Synergy)
	assert.Len(t, resp.Projections, 2)
	assert.Equal(t, "hba1c", resp.Projections[0].Metric)
	assert.Equal(t, -0.8, resp.Projections[0].BaselineDelta)
}

func TestProjectCombined_ServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "service unavailable", http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)

	resp, err := client.ProjectCombined(ProjectionRequest{
		PatientID:   "patient-1",
		ProtocolIDs: []string{"M3-PRP"},
		HorizonDays: 90,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "503")
	assert.Nil(t, resp)
}

func TestProjectCombined_NetworkError(t *testing.T) {
	client := newTestKB25Client("http://127.0.0.1:19999")

	resp, err := client.ProjectCombined(ProjectionRequest{
		PatientID:   "patient-1",
		ProtocolIDs: []string{"M3-PRP"},
		HorizonDays: 90,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "KB-25 projection request failed")
	assert.Nil(t, resp)
}

func TestProjectCombined_MalformedResponse(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`{"projections": "not-an-array"}`))
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)

	resp, err := client.ProjectCombined(ProjectionRequest{
		PatientID:   "patient-1",
		ProtocolIDs: []string{"M3-PRP"},
		HorizonDays: 90,
	})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "decode KB-25 projection response")
	assert.Nil(t, resp)
}

// ---------------------------------------------------------------------------
// NewKB25Client (config-based constructor)
// ---------------------------------------------------------------------------

func TestNewKB25Client_UsesConfigBaseURL(t *testing.T) {
	cfg := config.KB25Config{BaseURL: "http://kb-25:8136"}
	c := NewKB25Client(cfg, zap.NewNop())
	assert.Equal(t, "http://kb-25:8136", c.baseURL)
	assert.NotNil(t, c.httpClient)
	assert.Equal(t, 5*time.Second, c.httpClient.Timeout)
}

// ---------------------------------------------------------------------------
// HealthCheck
// ---------------------------------------------------------------------------

func TestHealthCheck_Healthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/health", r.URL.Path)
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)
	assert.NoError(t, client.HealthCheck())
}

func TestHealthCheck_Unhealthy(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer srv.Close()

	client := newTestKB25Client(srv.URL)
	err := client.HealthCheck()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "503")
}
