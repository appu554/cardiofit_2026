package kafka

import (
	"context"
	"encoding/json"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// MockAlertRouter is a mock implementation of AlertRouter for testing
type MockAlertRouter struct {
	routedAlerts  []*Alert
	routeError    error
	routeCount    int32
	routeDelay    time.Duration
}

func NewMockAlertRouter() *MockAlertRouter {
	return &MockAlertRouter{
		routedAlerts: make([]*Alert, 0),
	}
}

func (m *MockAlertRouter) RouteAlert(ctx context.Context, alert *Alert) error {
	if m.routeDelay > 0 {
		time.Sleep(m.routeDelay)
	}

	atomic.AddInt32(&m.routeCount, 1)

	if m.routeError != nil {
		return m.routeError
	}

	m.routedAlerts = append(m.routedAlerts, alert)
	return nil
}

func (m *MockAlertRouter) GetRoutedAlerts() []*Alert {
	return m.routedAlerts
}

func (m *MockAlertRouter) GetRouteCount() int {
	return int(atomic.LoadInt32(&m.routeCount))
}

func (m *MockAlertRouter) SetError(err error) {
	m.routeError = err
}

func (m *MockAlertRouter) SetDelay(delay time.Duration) {
	m.routeDelay = delay
}

func TestNewAlertConsumer(t *testing.T) {
	tests := []struct {
		name        string
		config      *ConsumerConfig
		router      AlertRouter
		expectError bool
		errorMsg    string
	}{
		{
			name:        "nil config",
			config:      nil,
			router:      NewMockAlertRouter(),
			expectError: true,
			errorMsg:    "config cannot be nil",
		},
		{
			name: "nil router",
			config: &ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				GroupID: "test-group",
				Topics:  []string{"test-topic"},
			},
			router:      nil,
			expectError: true,
			errorMsg:    "router cannot be nil",
		},
		{
			name: "empty brokers",
			config: &ConsumerConfig{
				Brokers: []string{},
				GroupID: "test-group",
				Topics:  []string{"test-topic"},
			},
			router:      NewMockAlertRouter(),
			expectError: true,
			errorMsg:    "brokers cannot be empty",
		},
		{
			name: "empty topics",
			config: &ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				GroupID: "test-group",
				Topics:  []string{},
			},
			router:      NewMockAlertRouter(),
			expectError: true,
			errorMsg:    "topics cannot be empty",
		},
		{
			name: "empty group ID",
			config: &ConsumerConfig{
				Brokers: []string{"localhost:9092"},
				GroupID: "",
				Topics:  []string{"test-topic"},
			},
			router:      NewMockAlertRouter(),
			expectError: true,
			errorMsg:    "groupID cannot be empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			logger := zaptest.NewLogger(t)
			consumer, err := NewAlertConsumer(tt.config, tt.router, logger)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
				assert.Nil(t, consumer)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, consumer)
			}
		})
	}
}

func TestAlertValidation(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewMockAlertRouter()
	config := &ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
	}

	consumer, err := NewAlertConsumer(config, router, logger)
	require.NoError(t, err)

	tests := []struct {
		name        string
		alert       *Alert
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid alert",
			alert: &Alert{
				AlertID:      "alert-001",
				PatientID:    "patient-001",
				HospitalID:   "hospital-001",
				DepartmentID: "department-001",
				AlertType:    AlertTypeSepsisAlert,
				Severity:     SeverityCritical,
				Message:      "Sepsis risk elevated",
				Timestamp:    time.Now().UnixMilli(),
			},
			expectError: false,
		},
		{
			name: "missing alert_id",
			alert: &Alert{
				PatientID:    "patient-001",
				HospitalID:   "hospital-001",
				DepartmentID: "department-001",
				AlertType:    AlertTypeSepsisAlert,
				Severity:     SeverityCritical,
				Message:      "Sepsis risk elevated",
				Timestamp:    time.Now().UnixMilli(),
			},
			expectError: true,
			errorMsg:    "alert_id is required",
		},
		{
			name: "missing patient_id",
			alert: &Alert{
				AlertID:      "alert-001",
				HospitalID:   "hospital-001",
				DepartmentID: "department-001",
				AlertType:    AlertTypeSepsisAlert,
				Severity:     SeverityCritical,
				Message:      "Sepsis risk elevated",
				Timestamp:    time.Now().UnixMilli(),
			},
			expectError: true,
			errorMsg:    "patient_id is required",
		},
		{
			name: "missing hospital_id",
			alert: &Alert{
				AlertID:      "alert-001",
				PatientID:    "patient-001",
				DepartmentID: "department-001",
				AlertType:    AlertTypeSepsisAlert,
				Severity:     SeverityCritical,
				Message:      "Sepsis risk elevated",
				Timestamp:    time.Now().UnixMilli(),
			},
			expectError: true,
			errorMsg:    "hospital_id is required",
		},
		{
			name: "missing alert_type",
			alert: &Alert{
				AlertID:      "alert-001",
				PatientID:    "patient-001",
				HospitalID:   "hospital-001",
				DepartmentID: "department-001",
				Severity:     SeverityCritical,
				Message:      "Sepsis risk elevated",
				Timestamp:    time.Now().UnixMilli(),
			},
			expectError: true,
			errorMsg:    "alert_type is required",
		},
		{
			name: "missing severity",
			alert: &Alert{
				AlertID:      "alert-001",
				PatientID:    "patient-001",
				HospitalID:   "hospital-001",
				DepartmentID: "department-001",
				AlertType:    AlertTypeSepsisAlert,
				Message:      "Sepsis risk elevated",
				Timestamp:    time.Now().UnixMilli(),
			},
			expectError: true,
			errorMsg:    "severity is required",
		},
		{
			name: "missing message",
			alert: &Alert{
				AlertID:      "alert-001",
				PatientID:    "patient-001",
				HospitalID:   "hospital-001",
				DepartmentID: "department-001",
				AlertType:    AlertTypeSepsisAlert,
				Severity:     SeverityCritical,
				Timestamp:    time.Now().UnixMilli(),
			},
			expectError: true,
			errorMsg:    "message is required",
		},
		{
			name: "missing timestamp",
			alert: &Alert{
				AlertID:      "alert-001",
				PatientID:    "patient-001",
				HospitalID:   "hospital-001",
				DepartmentID: "department-001",
				AlertType:    AlertTypeSepsisAlert,
				Severity:     SeverityCritical,
				Message:      "Sepsis risk elevated",
			},
			expectError: true,
			errorMsg:    "timestamp is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := consumer.validateAlert(tt.alert)
			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestAlertDeserialization(t *testing.T) {
	tests := []struct {
		name        string
		jsonData    string
		expectError bool
	}{
		{
			name: "complete alert",
			jsonData: `{
				"alert_id": "alert-001",
				"patient_id": "PAT-001",
				"hospital_id": "HOSP-001",
				"department_id": "ICU",
				"alert_type": "SEPSIS_ALERT",
				"severity": "CRITICAL",
				"confidence": 0.95,
				"message": "Patient PAT-001 sepsis risk elevated to 92%",
				"recommendations": ["Immediate physician review", "Blood culture", "Antibiotics within 1h"],
				"patient_location": {
					"room": "ICU-5",
					"bed": "A"
				},
				"vital_signs": {
					"heart_rate": 125,
					"blood_pressure_systolic": 85,
					"temperature": 39.2
				},
				"timestamp": 1699564800000,
				"metadata": {
					"source_module": "MODULE5_ML_INFERENCE",
					"model_version": "1.2.3",
					"requires_escalation": true
				}
			}`,
			expectError: false,
		},
		{
			name: "minimal alert",
			jsonData: `{
				"alert_id": "alert-002",
				"patient_id": "PAT-002",
				"hospital_id": "HOSP-001",
				"department_id": "ER",
				"alert_type": "VITAL_SIGN_ANOMALY",
				"severity": "HIGH",
				"confidence": 0.85,
				"message": "Heart rate elevated",
				"recommendations": [],
				"patient_location": {"room": "ER-12", "bed": "B"},
				"vital_signs": {},
				"timestamp": 1699564800000,
				"metadata": {
					"source_module": "MODULE4_CEP",
					"requires_escalation": false
				}
			}`,
			expectError: false,
		},
		{
			name:        "invalid json",
			jsonData:    `{"alert_id": "incomplete"`,
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var alert Alert
			err := json.Unmarshal([]byte(tt.jsonData), &alert)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.NotEmpty(t, alert.AlertID)
				assert.NotEmpty(t, alert.PatientID)
				assert.NotEmpty(t, alert.HospitalID)
			}
		})
	}
}

func TestConsumerMetrics(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewMockAlertRouter()
	config := &ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
	}

	consumer, err := NewAlertConsumer(config, router, logger)
	require.NoError(t, err)

	// Test initial metrics
	metrics := consumer.GetMetrics()
	assert.Equal(t, int64(0), metrics.MessagesConsumed)
	assert.Equal(t, int64(0), metrics.MessagesProcessed)
	assert.Equal(t, int64(0), metrics.MessagesFailed)

	// Update metrics
	consumer.updateMetrics(func(m *ConsumerMetrics) {
		m.MessagesConsumed = 100
		m.MessagesProcessed = 95
		m.MessagesFailed = 5
		m.ProcessingErrors["network_error"] = 3
		m.ProcessingErrors["validation_error"] = 2
		m.TopicMessageCounts["ml-risk-alerts.v1"] = 50
		m.TopicMessageCounts["clinical-patterns.v1"] = 50
	})

	// Verify updated metrics
	metrics = consumer.GetMetrics()
	assert.Equal(t, int64(100), metrics.MessagesConsumed)
	assert.Equal(t, int64(95), metrics.MessagesProcessed)
	assert.Equal(t, int64(5), metrics.MessagesFailed)
	assert.Equal(t, int64(3), metrics.ProcessingErrors["network_error"])
	assert.Equal(t, int64(2), metrics.ProcessingErrors["validation_error"])
	assert.Equal(t, int64(50), metrics.TopicMessageCounts["ml-risk-alerts.v1"])
	assert.Equal(t, int64(50), metrics.TopicMessageCounts["clinical-patterns.v1"])
}

func TestHealthCheck(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewMockAlertRouter()
	config := &ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
	}

	consumer, err := NewAlertConsumer(config, router, logger)
	require.NoError(t, err)

	// Health check should fail when not running
	err = consumer.HealthCheck()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "consumer not running")

	// Simulate running state
	consumer.mu.Lock()
	consumer.isRunning = true
	consumer.mu.Unlock()

	// Health check should pass when running
	err = consumer.HealthCheck()
	assert.NoError(t, err)

	// Simulate recent message
	consumer.updateMetrics(func(m *ConsumerMetrics) {
		m.LastMessageTimestamp = time.Now()
	})

	err = consumer.HealthCheck()
	assert.NoError(t, err)
}

func TestWorkerPoolSize(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewMockAlertRouter()
	router.SetDelay(100 * time.Millisecond) // Slow down processing

	config := &ConsumerConfig{
		Brokers:        []string{"localhost:9092"},
		GroupID:        "test-group",
		Topics:         []string{"test-topic"},
		WorkerPoolSize: 5,
	}

	consumer, err := NewAlertConsumer(config, router, logger)
	require.NoError(t, err)

	// Verify worker pool size
	assert.Equal(t, 5, cap(consumer.workerPool))
}

func TestConsumerConfigDefaults(t *testing.T) {
	logger := zaptest.NewLogger(t)
	router := NewMockAlertRouter()

	config := &ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
		// Leave defaults unset
	}

	consumer, err := NewAlertConsumer(config, router, logger)
	require.NoError(t, err)

	// Verify defaults are set
	assert.Equal(t, 30000, consumer.config.SessionTimeoutMs)
	assert.Equal(t, 100, consumer.config.MaxPollRecords)
	assert.Equal(t, 10, consumer.config.WorkerPoolSize)
}

func TestAlertTypes(t *testing.T) {
	// Verify all alert types are defined
	alertTypes := []AlertEventType{
		AlertTypeSepsisAlert,
		AlertTypeMortalityRisk,
		AlertTypeVitalSignAnomaly,
		AlertTypeDeteriorationWarning,
		AlertTypeReadmissionRisk,
		AlertTypeThresholdViolation,
		AlertTypeClinicalPattern,
		AlertTypeCriticalRouting,
		AlertTypeManualTrigger,
		AlertTypeEscalation,
	}

	for _, alertType := range alertTypes {
		assert.NotEmpty(t, string(alertType))
	}
}

func TestAlertSeverities(t *testing.T) {
	// Verify all severity levels are defined
	severities := []AlertSeverity{
		SeverityCritical,
		SeverityHigh,
		SeverityModerate,
		SeverityLow,
	}

	for _, severity := range severities {
		assert.NotEmpty(t, string(severity))
	}
}

func TestHelperFunctions(t *testing.T) {
	t.Run("joinStrings", func(t *testing.T) {
		result := joinStrings([]string{"a", "b", "c"}, ",")
		assert.Equal(t, "a,b,c", result)

		result = joinStrings([]string{"single"}, ",")
		assert.Equal(t, "single", result)

		result = joinStrings([]string{}, ",")
		assert.Equal(t, "", result)
	})

	t.Run("copyErrorMap", func(t *testing.T) {
		original := map[string]int64{"error1": 5, "error2": 10}
		copied := copyErrorMap(original)

		assert.Equal(t, original, copied)

		// Modify copy shouldn't affect original
		copied["error3"] = 15
		assert.NotEqual(t, original, copied)
	})

	t.Run("copyInt64Slice", func(t *testing.T) {
		original := []int64{1, 2, 3, 4, 5}
		copied := copyInt64Slice(original)

		assert.Equal(t, original, copied)

		// Modify copy shouldn't affect original
		copied[0] = 100
		assert.NotEqual(t, original[0], copied[0])
	})

	t.Run("copyInt64Map", func(t *testing.T) {
		original := map[string]int64{"topic1": 100, "topic2": 200}
		copied := copyInt64Map(original)

		assert.Equal(t, original, copied)

		// Modify copy shouldn't affect original
		copied["topic3"] = 300
		assert.NotEqual(t, original, copied)
	})
}

func BenchmarkAlertValidation(b *testing.B) {
	logger := zaptest.NewLogger(b)
	router := NewMockAlertRouter()
	config := &ConsumerConfig{
		Brokers: []string{"localhost:9092"},
		GroupID: "test-group",
		Topics:  []string{"test-topic"},
	}

	consumer, _ := NewAlertConsumer(config, router, logger)

	alert := &Alert{
		AlertID:      "alert-001",
		PatientID:    "patient-001",
		HospitalID:   "hospital-001",
		DepartmentID: "department-001",
		AlertType:    AlertTypeSepsisAlert,
		Severity:     SeverityCritical,
		Message:      "Sepsis risk elevated",
		Timestamp:    time.Now().UnixMilli(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = consumer.validateAlert(alert)
	}
}

func BenchmarkAlertDeserialization(b *testing.B) {
	jsonData := []byte(`{
		"alert_id": "alert-001",
		"patient_id": "PAT-001",
		"hospital_id": "HOSP-001",
		"department_id": "ICU",
		"alert_type": "SEPSIS_ALERT",
		"severity": "CRITICAL",
		"confidence": 0.95,
		"message": "Patient PAT-001 sepsis risk elevated to 92%",
		"recommendations": ["Immediate physician review"],
		"patient_location": {"room": "ICU-5", "bed": "A"},
		"vital_signs": {"heart_rate": 125},
		"timestamp": 1699564800000,
		"metadata": {"source_module": "MODULE5_ML_INFERENCE", "requires_escalation": true}
	}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var alert Alert
		_ = json.Unmarshal(jsonData, &alert)
	}
}
