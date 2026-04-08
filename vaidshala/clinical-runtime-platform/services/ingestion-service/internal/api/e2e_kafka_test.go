package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	kafkago "github.com/segmentio/kafka-go"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"

	"github.com/cardiofit/ingestion-service/internal/config"
	kafkapkg "github.com/cardiofit/ingestion-service/internal/kafka"
	"vaidshala/clinical-runtime-platform/pkg/fhirclient"
)

// kafkaBroker is the address of the local Kafka broker running in Docker.
const kafkaBroker = "localhost:9092"

// kafkaAvailable checks if Kafka is reachable before running E2E tests.
func kafkaAvailable(t *testing.T) {
	t.Helper()
	conn, err := kafkago.Dial("tcp", kafkaBroker)
	if err != nil {
		t.Skipf("Kafka not available at %s — skipping E2E test: %v", kafkaBroker, err)
	}
	conn.Close()
}

// e2eConfig returns a config with real Kafka brokers and a mock FHIR server.
func e2eConfig() *config.Config {
	cfg := config.Load()
	cfg.Kafka.Brokers = []string{kafkaBroker}
	cfg.FHIR.Enabled = false
	return cfg
}

// kafkaCapture reads from ALL partitions of a topic so that tests work
// regardless of how the Hash balancer distributes partition keys.
type kafkaCapture struct {
	readers []*kafkago.Reader
}

func newKafkaCapture(t *testing.T, topic string) *kafkaCapture {
	t.Helper()

	// Discover all partitions and snapshot their end offsets BEFORE any
	// test message is published.
	conn, err := kafkago.DialLeader(context.Background(), "tcp", kafkaBroker, topic, 0)
	require.NoError(t, err, "failed to connect to partition leader for %s", topic)
	partitions, err := conn.ReadPartitions(topic)
	require.NoError(t, err, "failed to read partitions for %s", topic)
	conn.Close()

	var readers []*kafkago.Reader
	for _, p := range partitions {
		pConn, err := kafkago.DialLeader(context.Background(), "tcp", kafkaBroker, topic, p.ID)
		require.NoError(t, err)
		endOffset, err := pConn.ReadLastOffset()
		require.NoError(t, err)
		pConn.Close()

		reader := kafkago.NewReader(kafkago.ReaderConfig{
			Brokers:   []string{kafkaBroker},
			Topic:     topic,
			Partition: p.ID,
			MaxWait:   200 * time.Millisecond,
		})
		reader.SetOffset(endOffset)
		readers = append(readers, reader)
	}

	return &kafkaCapture{readers: readers}
}

// readOne polls all partition readers until it finds a message whose key
// matches expectedKey, or the timeout expires.
func (kc *kafkaCapture) readOne(t *testing.T, timeout time.Duration, expectedKey ...string) (key []byte, envelope kafkapkg.Envelope) {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	for {
		for _, reader := range kc.readers {
			pollCtx, pollCancel := context.WithTimeout(ctx, 300*time.Millisecond)
			msg, err := reader.ReadMessage(pollCtx)
			pollCancel()

			if err != nil {
				if ctx.Err() != nil {
					require.NoError(t, ctx.Err(), "no matching message on any partition within %v", timeout)
				}
				continue // timeout on this partition, try next
			}

			if len(expectedKey) > 0 && string(msg.Key) != expectedKey[0] {
				t.Logf("  skipping stale message with key=%s (want %s)", string(msg.Key), expectedKey[0])
				continue
			}

			key = msg.Key
			require.NoError(t, json.Unmarshal(msg.Value, &envelope), "failed to unmarshal Kafka envelope")
			return
		}
	}
}

func (kc *kafkaCapture) close() {
	for _, r := range kc.readers {
		r.Close()
	}
}

// =====================================================================
// E2E Test: FHIR Observation → Pipeline → Kafka ingestion.labs
// =====================================================================

func TestE2E_FHIRObservation_ToKafkaLabs(t *testing.T) {
	kafkaAvailable(t)

	logger, _ := zap.NewDevelopment()
	cfg := e2eConfig()

	// Mock FHIR Store
	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	// handleFHIRObservation sets ObservationType=ObsGeneral → routes to ingestion.observations
	capture := newKafkaCapture(t, "ingestion.observations")
	defer capture.close()

	patientID := uuid.New()
	body := map[string]interface{}{
		"patient_id": patientID.String(),
		"tenant_id":  uuid.New().String(),
		"loinc_code": "1558-6", // Fasting glucose
		"value":      142.0,
		"unit":       "mg/dL",
		"timestamp":  "2026-03-23T08:00:00Z",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	// Verify HTTP response
	assert.Equal(t, http.StatusCreated, w.Code)
	var httpResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &httpResp)
	assert.Equal(t, "accepted", httpResp["status"])
	assert.NotEmpty(t, httpResp["observation_id"])
	assert.NotEmpty(t, httpResp["fhir_resource_id"])

	// Verify Kafka message (filter by patientID to avoid cross-test leakage)
	key, envelope := capture.readOne(t, 10*time.Second, patientID.String())

	assert.Equal(t, patientID.String(), string(key), "partition key should be patient_id")
	assert.Equal(t, "OBSERVATION", envelope.EventType)
	assert.Equal(t, "EHR", envelope.SourceType)
	assert.Equal(t, patientID, envelope.PatientID)
	assert.NotEmpty(t, envelope.FHIRResourceType)
	assert.NotEmpty(t, envelope.EventID)

	// Verify payload contents
	assert.Equal(t, "1558-6", envelope.Payload["loinc_code"])
	assert.InDelta(t, 142.0, envelope.Payload["value"], 0.1)
	assert.Equal(t, "mg/dL", envelope.Payload["unit"])

	t.Logf("E2E PASS: FHIR Observation → ingestion.observations | event_id=%s quality=%.2f",
		envelope.EventID, envelope.QualityScore)
}

// =====================================================================
// E2E Test: Device readings → Pipeline → Kafka ingestion.device-data
// =====================================================================

func TestE2E_DeviceIngest_ToKafkaDeviceData(t *testing.T) {
	kafkaAvailable(t)

	logger, _ := zap.NewDevelopment()
	cfg := e2eConfig()

	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	capture := newKafkaCapture(t, "ingestion.device-data")
	defer capture.close()

	patientID := uuid.New()
	body := map[string]interface{}{
		"patient_id": patientID.String(),
		"tenant_id":  uuid.New().String(),
		"timestamp":  "2026-03-23T09:00:00Z",
		"device": map[string]interface{}{
			"device_id":    "bp-monitor-001",
			"device_type":  "blood_pressure_monitor",
			"manufacturer": "Omron",
			"model":        "HEM-7120",
		},
		"readings": []map[string]interface{}{
			{"analyte": "systolic_bp", "value": 148.0, "unit": "mmHg"},
			{"analyte": "diastolic_bp", "value": 92.0, "unit": "mmHg"},
			{"analyte": "heart_rate", "value": 78.0, "unit": "bpm"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/devices", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var httpResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &httpResp)
	assert.Equal(t, "accepted", httpResp["status"])
	assert.Equal(t, float64(3), httpResp["processed"])

	// Read all 3 device readings from Kafka
	for i := 0; i < 3; i++ {
		key, envelope := capture.readOne(t, 10*time.Second, patientID.String())
		assert.Equal(t, patientID.String(), string(key))
		assert.Equal(t, "DEVICE_READING", envelope.EventType)
		assert.Equal(t, "DEVICE", envelope.SourceType)
		assert.Equal(t, patientID, envelope.PatientID)

		t.Logf("  device reading %d: loinc=%s value=%v unit=%s",
			i+1, envelope.Payload["loinc_code"], envelope.Payload["value"], envelope.Payload["unit"])
	}

	t.Logf("E2E PASS: 3 device readings → ingestion.device-data")
}

// =====================================================================
// E2E Test: Critical potassium → Pipeline → Kafka with CRITICAL_VALUE flag
// =====================================================================

func TestE2E_CriticalValue_FlaggedInKafkaEnvelope(t *testing.T) {
	kafkaAvailable(t)

	logger, _ := zap.NewDevelopment()
	cfg := e2eConfig()

	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	capture := newKafkaCapture(t, "ingestion.observations")
	defer capture.close()

	patientID := uuid.New()
	body := map[string]interface{}{
		"patient_id": patientID.String(),
		"loinc_code": "2823-3", // Potassium
		"value":      6.8,      // >= 6.0 is CRITICAL_VALUE
		"unit":       "mEq/L",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	// Verify HTTP response has CRITICAL_VALUE flag
	var httpResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &httpResp)
	flags := httpResp["flags"].([]interface{})
	flagStrs := make([]string, len(flags))
	for i, f := range flags {
		flagStrs[i] = f.(string)
	}
	assert.Contains(t, flagStrs, "CRITICAL_VALUE")

	// Verify Kafka envelope also has the CRITICAL_VALUE flag
	_, envelope := capture.readOne(t, 10*time.Second, patientID.String())
	assert.Equal(t, "OBSERVATION", envelope.EventType)
	assert.Contains(t, envelope.Flags, "CRITICAL_VALUE",
		"Kafka envelope must carry CRITICAL_VALUE flag for K+ 6.8 mEq/L")

	t.Logf("E2E PASS: K+ 6.8 mEq/L → CRITICAL_VALUE in HTTP + Kafka envelope | flags=%v", envelope.Flags)
}

// =====================================================================
// E2E Test: Unit conversion (mmol/L → mg/dL) verified in Kafka payload
// =====================================================================

func TestE2E_UnitConversion_VerifiedInKafka(t *testing.T) {
	kafkaAvailable(t)

	logger, _ := zap.NewDevelopment()
	cfg := e2eConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	capture := newKafkaCapture(t, "ingestion.observations")
	defer capture.close()

	patientID := uuid.New()
	body := map[string]interface{}{
		"patient_id": patientID.String(),
		"loinc_code": "1558-6", // Fasting glucose
		"value":      7.0,      // 7.0 mmol/L = 126.0 mg/dL
		"unit":       "mmol/L",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	_, envelope := capture.readOne(t, 10*time.Second, patientID.String())

	// After normalization: 7.0 mmol/L × 18.0 = 126.0 mg/dL
	value, ok := envelope.Payload["value"].(float64)
	require.True(t, ok, "payload value should be a float64")
	assert.InDelta(t, 126.0, value, 0.5,
		"7.0 mmol/L glucose should be normalized to ~126 mg/dL")
	assert.Equal(t, "mg/dL", envelope.Payload["unit"],
		"unit should be normalized to mg/dL")

	t.Logf("E2E PASS: 7.0 mmol/L → %.1f mg/dL in Kafka envelope", value)
}

// =====================================================================
// E2E Test: App check-in → Pipeline → Kafka ingestion.patient-reported
// =====================================================================

func TestE2E_AppCheckin_ToKafkaPatientReported(t *testing.T) {
	kafkaAvailable(t)

	logger, _ := zap.NewDevelopment()
	cfg := e2eConfig()

	fhirSrv := mockFHIRServer(t)
	defer fhirSrv.Close()
	mockFHIR := fhirclient.NewWithHTTPClient(fhirSrv.URL, fhirSrv.Client(), logger)

	server := NewServer(cfg, nil, nil, mockFHIR, logger)

	// The app-checkin adapter routes by clinical type:
	// - glucose → PATIENT_REPORTED → ingestion.patient-reported
	// - weight  → VITALS          → ingestion.vitals
	capturePatient := newKafkaCapture(t, "ingestion.patient-reported")
	defer capturePatient.close()
	captureVitals := newKafkaCapture(t, "ingestion.vitals")
	defer captureVitals.close()

	patientID := uuid.New()
	body := map[string]interface{}{
		"patient_id": patientID.String(),
		"tenant_id":  uuid.New().String(),
		"timestamp":  "2026-03-23T07:30:00Z",
		"readings": []map[string]interface{}{
			{"analyte": "fasting_glucose", "value": 156.0, "unit": "mg/dL"},
			{"analyte": "weight", "value": 74.2, "unit": "kg"},
		},
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/app-checkin", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	var httpResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &httpResp)
	assert.Equal(t, float64(2), httpResp["processed"])

	// Glucose → ingestion.patient-reported
	key, envelope := capturePatient.readOne(t, 10*time.Second, patientID.String())
	assert.Equal(t, patientID.String(), string(key))
	assert.Equal(t, "PATIENT_REPORT", envelope.EventType)
	assert.Equal(t, "PATIENT_REPORTED", envelope.SourceType)
	t.Logf("  glucose: loinc=%s value=%v", envelope.Payload["loinc_code"], envelope.Payload["value"])

	// Weight → ingestion.vitals (clinical type takes precedence over source)
	key2, envelope2 := captureVitals.readOne(t, 10*time.Second, patientID.String())
	assert.Equal(t, patientID.String(), string(key2))
	assert.Equal(t, "VITAL_SIGN", envelope2.EventType)
	assert.Equal(t, "PATIENT_REPORTED", envelope2.SourceType)
	t.Logf("  weight: loinc=%s value=%v", envelope2.Payload["loinc_code"], envelope2.Payload["value"])

	t.Logf("E2E PASS: app-checkin → glucose to patient-reported, weight to vitals")
}

// =====================================================================
// E2E Test: eGFR=25 (critical low) → Kafka with CRITICAL_VALUE flag
// =====================================================================

func TestE2E_CriticalEGFR_FlaggedInKafka(t *testing.T) {
	kafkaAvailable(t)

	logger, _ := zap.NewDevelopment()
	cfg := e2eConfig()
	server := NewServer(cfg, nil, nil, nil, logger)

	capture := newKafkaCapture(t, "ingestion.observations")
	defer capture.close()

	patientID := uuid.New()
	body := map[string]interface{}{
		"patient_id": patientID.String(),
		"loinc_code": "33914-3", // eGFR
		"value":      12.0,      // <= 15 is CRITICAL_VALUE (renal failure)
		"unit":       "mL/min/1.73m2",
	}
	bodyBytes, _ := json.Marshal(body)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, "/fhir/Observation", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	server.Router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)

	_, envelope := capture.readOne(t, 10*time.Second, patientID.String())
	assert.Equal(t, "OBSERVATION", envelope.EventType)
	assert.Contains(t, envelope.Flags, "CRITICAL_VALUE",
		"eGFR 12 should trigger CRITICAL_VALUE (threshold: <= 15)")

	t.Logf("E2E PASS: eGFR 12 → CRITICAL_VALUE in Kafka | quality=%.2f flags=%v",
		envelope.QualityScore, envelope.Flags)
}
