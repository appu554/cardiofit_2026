package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/config"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

func testConfig() *config.Config {
	cfg := config.Load()
	cfg.FHIR.Enabled = false
	cfg.Kafka.Brokers = []string{""} // Disable Kafka in tests
	return cfg
}

// mockFHIRServer creates a mock Google FHIR Store that accepts creates.
func mockFHIRServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodPost:
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"resourceType":"Observation","id":"fhir-obs-001"}`))
		case http.MethodGet:
			if r.URL.Path == "/metadata" {
				w.WriteHeader(http.StatusOK)
				return
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte(`{"resourceType":"Bundle","total":0}`))
		default:
			w.WriteHeader(http.StatusMethodNotAllowed)
		}
	}))
}

func TestIntegration_PostFHIRObservation(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()

	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"tenant_id":  uuid.New().String(),
		"loinc_code": "1558-6",
		"value":      142.0,
		"unit":       "mg/dL",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
	assert.NotEmpty(t, resp["observation_id"])
	assert.NotEmpty(t, resp["fhir_resource_id"])
	assert.True(t, resp["quality_score"].(float64) > 0)
}

func TestIntegration_PostFHIRObservation_UnitConversion(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"loinc_code": "1558-6",
		"value":      7.0,
		"unit":       "mmol/L",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
}

func TestIntegration_PostDeviceIngest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()

	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"tenant_id":  uuid.New().String(),
		"timestamp":  "2026-03-21T08:00:00Z",
		"device": map[string]interface{}{
			"device_id":    "bp-001",
			"device_type":  "blood_pressure_monitor",
			"manufacturer": "Omron",
			"model":        "HEM-7120",
		},
		"readings": []map[string]interface{}{
			{"analyte": "systolic_bp", "value": 135.0, "unit": "mmHg"},
			{"analyte": "diastolic_bp", "value": 88.0, "unit": "mmHg"},
			{"analyte": "heart_rate", "value": 74.0, "unit": "bpm"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ingest/devices", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
	assert.Equal(t, float64(3), resp["processed"])
}

func TestIntegration_PostAppCheckin(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"tenant_id":  uuid.New().String(),
		"timestamp":  "2026-03-21T08:00:00Z",
		"readings": []map[string]interface{}{
			{"analyte": "fasting_glucose", "value": 142.0, "unit": "mg/dL"},
			{"analyte": "weight", "value": 72.5, "unit": "kg"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/ingest/app-checkin", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "accepted", resp["status"])
	assert.Equal(t, float64(2), resp["processed"])
}

func TestIntegration_PostHPIIngest(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	observations := []map[string]interface{}{
		{
			"id":               uuid.New().String(),
			"patient_id":       uuid.New().String(),
			"tenant_id":        uuid.New().String(),
			"source_type":      "HPI",
			"observation_type": "HPI",
			"loinc_code":       "1558-6",
			"value":            180.0,
			"unit":             "mg/dL",
			"timestamp":        "2026-03-21T10:00:00Z",
		},
	}
	bodyBytes, _ := json.Marshal(observations)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/internal/hpi", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
}

func TestIntegration_DLQListEmpty(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/fhir/OperationOutcome?category=dlq", nil)
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)
	assert.Equal(t, "Bundle", resp["resourceType"])
	assert.Equal(t, float64(0), resp["total"])
}

func TestIntegration_InvalidObservationGoesToDLQ(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	// Missing patient_id -> should fail validation -> DLQ
	body := map[string]interface{}{
		"loinc_code": "1558-6",
		"value":      100.0,
		"unit":       "mg/dL",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	// Should get 400 (bad request due to invalid patient_id parse)
	assert.True(t, w.Code == http.StatusBadRequest || w.Code == http.StatusUnprocessableEntity,
		"expected 400 or 422, got %d", w.Code)
}

func TestIntegration_CriticalValueFlagged(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	body := map[string]interface{}{
		"patient_id": uuid.New().String(),
		"loinc_code": "2823-3", // Potassium
		"value":      6.8,      // K+ >= 6.0 = critical
		"unit":       "mEq/L",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	flags, ok := resp["flags"].([]interface{})
	require.True(t, ok, "flags should be an array")
	flagStrs := make([]string, len(flags))
	for i, f := range flags {
		flagStrs[i] = f.(string)
	}
	assert.Contains(t, flagStrs, "CRITICAL_VALUE")
}

func TestIntegration_HealthEndpoints(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	endpoints := []string{"/healthz", "/startupz"}
	for _, ep := range endpoints {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, ep, nil)
		server.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code, "endpoint %s should return 200", ep)
	}
}

func TestIntegration_StubEndpointsReturn501(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	cfg := testConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	stubs := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/fhir"},
		{http.MethodPost, "/fhir/DiagnosticReport"},
		{http.MethodPost, "/ingest/ehr/hl7v2"},
		{http.MethodPost, "/ingest/ehr/fhir"},
		{http.MethodPost, "/ingest/labs/thyrocare"},
		{http.MethodPost, "/ingest/abdm/data-push"},
	}

	for _, s := range stubs {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(s.method, s.path, bytes.NewReader([]byte("{}")))
		server.Router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusNotImplemented, w.Code,
			"stub %s %s should return 501", s.method, s.path)
	}
}
