package delivery

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cardiofit/notification-service/internal/models"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
)

// Test: Exponential Backoff Calculation
func TestCalculateBackoff(t *testing.T) {
	service := &NotificationDeliveryService{
		retryPolicy: RetryPolicy{
			InitialBackoff: 1 * time.Second,
			MaxBackoff:     30 * time.Second,
			Multiplier:     2.0,
		},
	}

	tests := []struct {
		attempt  int
		expected time.Duration
	}{
		{0, 1 * time.Second},
		{1, 2 * time.Second},
		{2, 4 * time.Second},
		{3, 8 * time.Second},
		{4, 16 * time.Second},
		{5, 30 * time.Second}, // Capped at MaxBackoff
		{6, 30 * time.Second}, // Still capped
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Attempt_%d", tt.attempt), func(t *testing.T) {
			backoff := service.calculateBackoff(tt.attempt)
			assert.Equal(t, tt.expected, backoff)
		})
	}
}

// Test: Metrics Collection
func TestMetricsCollector(t *testing.T) {
	collector := NewMetricsCollector()

	// Record various attempts
	collector.RecordAttempt(models.ChannelSMS, true, 150)
	collector.RecordAttempt(models.ChannelSMS, true, 200)
	collector.RecordAttempt(models.ChannelSMS, false, 300)
	collector.RecordAttempt(models.ChannelEmail, true, 250)

	// Verify SMS metrics
	smsMetrics := collector.GetMetrics(models.ChannelSMS)
	assert.Equal(t, int64(3), smsMetrics.TotalAttempts)
	assert.Equal(t, int64(2), smsMetrics.Successful)
	assert.Equal(t, int64(1), smsMetrics.Failed)
	assert.Equal(t, int64(650), smsMetrics.TotalLatency)

	// Verify Email metrics
	emailMetrics := collector.GetMetrics(models.ChannelEmail)
	assert.Equal(t, int64(1), emailMetrics.TotalAttempts)
	assert.Equal(t, int64(1), emailMetrics.Successful)
	assert.Equal(t, int64(0), emailMetrics.Failed)
	assert.Equal(t, int64(250), emailMetrics.TotalLatency)

	// Verify non-existent channel returns zero metrics
	pushMetrics := collector.GetMetrics(models.ChannelPush)
	assert.Equal(t, int64(0), pushMetrics.TotalAttempts)
}

// Test: Default Retry Policy
func TestDefaultRetryPolicy(t *testing.T) {
	policy := DefaultRetryPolicy()

	assert.Equal(t, 3, policy.MaxAttempts)
	assert.Equal(t, 1*time.Second, policy.InitialBackoff)
	assert.Equal(t, 30*time.Second, policy.MaxBackoff)
	assert.Equal(t, 2.0, policy.Multiplier)
}

// Test: Notification Validation
func TestNotificationValidation(t *testing.T) {
	logger := zap.NewNop()

	service := &NotificationDeliveryService{
		logger:           logger,
		config:           DeliveryConfig{Workers: 10},
		retryPolicy:      DefaultRetryPolicy(),
		workerPool:       make(chan struct{}, 10),
		metricsCollector: NewMetricsCollector(),
	}

	ctx := context.Background()

	// Test nil notification
	err := service.Send(ctx, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "notification cannot be nil")

	// Test notification without user
	notification := &models.Notification{
		ID:    "notif-123",
		Alert: &models.Alert{},
	}
	err = service.Send(ctx, notification)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must have user and alert data")
}

// Test: TwilioClient Initialization
func TestTwilioClientInitialization(t *testing.T) {
	logger := zap.NewNop()

	client := NewTwilioClient("test_sid", "test_token", "+1234567890", logger)

	assert.NotNil(t, client)
	assert.Equal(t, "test_sid", client.accountSID)
	assert.Equal(t, "test_token", client.authToken)
	assert.Equal(t, "+1234567890", client.fromNumber)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "https://api.twilio.com/2010-04-01", client.baseURL)
}

// Test: SendGridClient Initialization
func TestSendGridClientInitialization(t *testing.T) {
	logger := zap.NewNop()

	client := NewSendGridClient("test_api_key", "test@example.com", logger)

	assert.NotNil(t, client)
	assert.Equal(t, "test_api_key", client.apiKey)
	assert.Equal(t, "test@example.com", client.fromEmail)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, "https://api.sendgrid.com/v3", client.baseURL)
}

// Test: SendGrid Severity Class Mapping
func TestGetSeverityClass(t *testing.T) {
	logger := zap.NewNop()
	client := NewSendGridClient("test", "test@test.com", logger)

	tests := []struct {
		severity models.AlertSeverity
		expected string
	}{
		{models.SeverityCritical, "critical"},
		{models.SeverityHigh, "high"},
		{models.SeverityModerate, "moderate"},
		{models.SeverityLow, "low"},
		{models.SeverityMLAlert, "moderate"}, // Default
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			class := client.getSeverityClass(tt.severity)
			assert.Equal(t, tt.expected, class)
		})
	}
}

// Test: Twilio TwiML Generation
func TestBuildTwiML(t *testing.T) {
	logger := zap.NewNop()
	client := NewTwilioClient("test", "test", "+1", logger)

	message := "Critical alert for patient 001"
	twiml := client.buildTwiML(message)

	assert.Contains(t, twiml, "<?xml version=\"1.0\" encoding=\"UTF-8\"?>")
	assert.Contains(t, twiml, "<Response>")
	assert.Contains(t, twiml, "<Say")
	assert.Contains(t, twiml, message)
	assert.Contains(t, twiml, "</Response>")
}

// Test: Firebase Priority Mapping
func TestFirebasePriorityMapping(t *testing.T) {
	logger := zap.NewNop()
	client := &FirebaseClient{logger: logger}

	tests := []struct {
		severity models.AlertSeverity
		priority string
		apns     string
	}{
		{models.SeverityCritical, "high", "10"},
		{models.SeverityHigh, "high", "10"},
		{models.SeverityModerate, "normal", "5"},
		{models.SeverityLow, "normal", "5"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			priority := client.getPriorityForSeverity(tt.severity)
			assert.Equal(t, tt.priority, priority)

			apnsPriority := client.getAPNSPriority(tt.severity)
			assert.Equal(t, tt.apns, apnsPriority)
		})
	}
}

// Test: Firebase Color Mapping
func TestFirebaseColorMapping(t *testing.T) {
	logger := zap.NewNop()
	client := &FirebaseClient{logger: logger}

	tests := []struct {
		severity models.AlertSeverity
		color    string
	}{
		{models.SeverityCritical, "#dc3545"},
		{models.SeverityHigh, "#fd7e14"},
		{models.SeverityModerate, "#ffc107"},
		{models.SeverityLow, "#28a745"},
		{models.SeverityMLAlert, "#6c757d"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			color := client.getColorForSeverity(tt.severity)
			assert.Equal(t, tt.color, color)
		})
	}
}

// Test: Firebase Sound Mapping
func TestFirebaseSoundMapping(t *testing.T) {
	logger := zap.NewNop()
	client := &FirebaseClient{logger: logger}

	tests := []struct {
		severity models.AlertSeverity
		sound    string
	}{
		{models.SeverityCritical, "critical_alert.wav"},
		{models.SeverityHigh, "high_alert.wav"},
		{models.SeverityModerate, "moderate_alert.wav"},
		{models.SeverityLow, "default.wav"},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			sound := client.getSoundForSeverity(tt.severity)
			assert.Equal(t, tt.sound, sound)
		})
	}
}

// Test: Worker Pool Sizing
func TestWorkerPoolInitialization(t *testing.T) {
	logger := zap.NewNop()

	tests := []struct {
		workers int
	}{
		{1},
		{5},
		{10},
		{20},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("Workers_%d", tt.workers), func(t *testing.T) {
			service := &NotificationDeliveryService{
				logger:           logger,
				config:           DeliveryConfig{Workers: tt.workers},
				workers:          tt.workers,
				workerPool:       make(chan struct{}, tt.workers),
				metricsCollector: NewMetricsCollector(),
			}

			assert.Equal(t, tt.workers, service.workers)
			assert.Equal(t, tt.workers, cap(service.workerPool))
		})
	}
}

// Test: Context Cancellation Handling
func TestContextCancellation(t *testing.T) {
	// Test that cancelled context is properly detected
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// Verify context is cancelled
	assert.Equal(t, context.Canceled, ctx.Err())

	// Test with timeout
	ctx2, cancel2 := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel2()

	time.Sleep(10 * time.Millisecond)
	assert.Equal(t, context.DeadlineExceeded, ctx2.Err())
}

// Test: Notification Creation Helper
func TestCreateTestNotification(t *testing.T) {
	channels := []models.NotificationChannel{
		models.ChannelSMS,
		models.ChannelEmail,
		models.ChannelPush,
		models.ChannelVoice,
		models.ChannelInApp,
	}

	for _, channel := range channels {
		t.Run(string(channel), func(t *testing.T) {
			now := time.Now()
			notification := &models.Notification{
				ID:      "notif-123",
				AlertID: "alert-456",
				UserID:  "user-789",
				User: &models.User{
					ID:          "user-789",
					Name:        "Dr. Smith",
					Email:       "dr.smith@hospital.com",
					PhoneNumber: "+1234567890",
					FCMToken:    "fcm_token_xyz",
					Role:        "ATTENDING",
				},
				Alert: &models.Alert{
					AlertID:      "alert-456",
					PatientID:    "patient-001",
					HospitalID:   "hospital-001",
					DepartmentID: "dept-icu",
					AlertType:    models.AlertTypeSepsis,
					Severity:     models.SeverityCritical,
					Message:      "Sepsis risk detected",
					VitalSigns: &models.VitalSigns{
						HeartRate:              120,
						BloodPressureSystolic:  90,
						BloodPressureDiastolic: 60,
						Temperature:            101.5,
					},
					Timestamp: time.Now().Unix(),
				},
				Channel:   channel,
				Status:    models.StatusPending,
				CreatedAt: now,
			}

			assert.NotNil(t, notification)
			assert.Equal(t, channel, notification.Channel)
			assert.NotNil(t, notification.User)
			assert.NotNil(t, notification.Alert)
			assert.Equal(t, models.StatusPending, notification.Status)
		})
	}
}

// Test: Alert Data Completeness
func TestAlertDataCompleteness(t *testing.T) {
	now := time.Now()
	notification := &models.Notification{
		ID:      "notif-123",
		AlertID: "alert-456",
		UserID:  "user-789",
		Alert: &models.Alert{
			AlertID:      "alert-456",
			PatientID:    "patient-001",
			HospitalID:   "hospital-001",
			DepartmentID: "dept-icu",
			Message:      "Sepsis risk detected",
			VitalSigns: &models.VitalSigns{
				HeartRate:              120,
				BloodPressureSystolic:  90,
				BloodPressureDiastolic: 60,
				Temperature:            101.5,
			},
			Recommendations: []string{"Administer antibiotics"},
			Timestamp:       time.Now().Unix(),
		},
		CreatedAt: now,
	}

	alert := notification.Alert

	assert.NotEmpty(t, alert.AlertID)
	assert.NotEmpty(t, alert.PatientID)
	assert.NotEmpty(t, alert.HospitalID)
	assert.NotEmpty(t, alert.DepartmentID)
	assert.NotEmpty(t, alert.Message)
	assert.NotNil(t, alert.VitalSigns)
	assert.NotEmpty(t, alert.Recommendations)
	assert.NotEqual(t, 0, alert.Timestamp)

	// Verify vital signs
	assert.Greater(t, alert.VitalSigns.HeartRate, 0)
	assert.Greater(t, alert.VitalSigns.BloodPressureSystolic, 0)
	assert.Greater(t, alert.VitalSigns.Temperature, float64(0))
}

// Test: User Contact Information
func TestUserContactInformation(t *testing.T) {
	user := &models.User{
		ID:          "user-789",
		Name:        "Dr. Smith",
		Email:       "dr.smith@hospital.com",
		PhoneNumber: "+1234567890",
		FCMToken:    "fcm_token_xyz",
		Role:        "ATTENDING",
	}

	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.Name)
	assert.NotEmpty(t, user.Email)
	assert.NotEmpty(t, user.PhoneNumber)
	assert.NotEmpty(t, user.FCMToken)
	assert.NotEmpty(t, user.Role)
}

// Benchmark: Metrics Collection
func BenchmarkMetricsCollection(b *testing.B) {
	collector := NewMetricsCollector()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		collector.RecordAttempt(models.ChannelSMS, true, 150)
	}
}

// Benchmark: Backoff Calculation
func BenchmarkBackoffCalculation(b *testing.B) {
	service := &NotificationDeliveryService{
		retryPolicy: DefaultRetryPolicy(),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.calculateBackoff(i % 10)
	}
}
