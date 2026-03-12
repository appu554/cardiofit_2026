package routing

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"go.uber.org/zap"

	"github.com/cardiofit/notification-service/internal/models"
)

// ExampleIntegration demonstrates how to integrate the AlertRouter
// This file shows practical usage patterns and common scenarios

// Example 1: Basic Integration with Kafka Consumer
func ExampleKafkaConsumerIntegration() {
	// Initialize dependencies (these would come from your DI container)
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	// Mock implementations (replace with actual implementations)
	fatigueTracker := &MockFatigueTrackerImpl{}
	userService := &MockUserServiceImpl{}
	deliveryService := &MockDeliveryServiceImpl{}
	escalationMgr := &MockEscalationManagerImpl{}

	// Create router
	router := NewAlertRouter(
		fatigueTracker,
		userService,
		deliveryService,
		escalationMgr,
		logger,
	)

	// Simulate Kafka message
	kafkaMessage := []byte(`{
		"alert_id": "alert-20251110-001",
		"patient_id": "PAT-001",
		"hospital_id": "HOSP-001",
		"department_id": "DEPT-ICU",
		"alert_type": "SEPSIS_ALERT",
		"severity": "CRITICAL",
		"confidence": 0.92,
		"message": "Patient PAT-001 sepsis risk elevated to 92%",
		"recommendations": [
			"Immediate physician review",
			"Blood culture within 30 minutes",
			"Broad-spectrum antibiotics within 1 hour"
		],
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
			"source_module": "MODULE4_CEP",
			"requires_escalation": true
		}
	}`)

	// Unmarshal alert
	var alert models.Alert
	if err := json.Unmarshal(kafkaMessage, &alert); err != nil {
		logger.Error("Failed to unmarshal alert", zap.Error(err))
		return
	}

	// Route alert
	ctx := context.Background()
	if err := router.RouteAlert(ctx, &alert); err != nil {
		logger.Error("Failed to route alert", zap.Error(err))
		return
	}

	logger.Info("Alert routed successfully", zap.String("alert_id", alert.AlertID))
}

// Example 2: Preview Routing Decision Before Sending
func ExampleRoutingDecisionPreview() {
	logger := zap.NewNop()

	router := NewAlertRouter(
		&MockFatigueTrackerImpl{},
		&MockUserServiceImpl{},
		&MockDeliveryServiceImpl{},
		&MockEscalationManagerImpl{},
		logger,
	)

	alert := &models.Alert{
		AlertID:      "alert-preview-001",
		PatientID:    "PAT-002",
		DepartmentID: "DEPT-ER",
		AlertType:    models.AlertTypeDeterioration,
		Severity:     models.SeverityHigh,
		Confidence:   0.88,
		Message:      "Patient showing signs of deterioration",
		Metadata: models.AlertMetadata{
			SourceModule:       "MODULE5_ML_INFERENCE",
			RequiresEscalation: true,
		},
	}

	ctx := context.Background()
	decision, err := router.GetRoutingDecision(ctx, alert)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	// Print routing decision
	fmt.Printf("Routing Decision for Alert %s:\n", alert.AlertID)
	fmt.Printf("  Target Users: %d\n", len(decision.TargetUsers))
	fmt.Printf("  Requires Escalation: %v\n", decision.RequiresEscalation)
	fmt.Printf("  Escalation Timeout: %v\n", decision.EscalationTimeout)
	fmt.Println("\nUser Channels:")
	for userID, channels := range decision.UserChannels {
		fmt.Printf("  - User %s: %v\n", userID, channels)
	}
	if len(decision.SuppressedUsers) > 0 {
		fmt.Println("\nSuppressed Users:")
		for userID, reason := range decision.SuppressedUsers {
			fmt.Printf("  - User %s: %s\n", userID, reason)
		}
	}
}

// Example 3: Handling Different Alert Types
func ExampleHandleDifferentAlertTypes() {
	logger := zap.NewNop()

	router := NewAlertRouter(
		&MockFatigueTrackerImpl{},
		&MockUserServiceImpl{},
		&MockDeliveryServiceImpl{},
		&MockEscalationManagerImpl{},
		logger,
	)

	ctx := context.Background()

	// Critical Sepsis Alert
	sepsisAlert := &models.Alert{
		AlertID:      "alert-sepsis-001",
		PatientID:    "PAT-003",
		DepartmentID: "DEPT-ICU",
		AlertType:    models.AlertTypeSepsis,
		Severity:     models.SeverityCritical,
		Confidence:   0.95,
		Message:      "High risk sepsis detected",
		PatientLocation: models.PatientLocation{
			Room: "ICU-10",
			Bed:  "B",
		},
		Timestamp: time.Now().UnixMilli(),
		Metadata: models.AlertMetadata{
			SourceModule:       "MODULE5_ML_INFERENCE",
			ModelVersion:       "1.2.3",
			RequiresEscalation: true,
		},
	}

	if err := router.RouteAlert(ctx, sepsisAlert); err != nil {
		fmt.Printf("Failed to route sepsis alert: %v\n", err)
	}

	// ML-based Mortality Risk Alert
	mortalityAlert := &models.Alert{
		AlertID:      "alert-mortality-001",
		PatientID:    "PAT-004",
		DepartmentID: "DEPT-CARDIOLOGY",
		AlertType:    models.AlertTypeMortalityRisk,
		Severity:     models.SeverityMLAlert,
		Confidence:   0.78,
		RiskScore:    0.82,
		Message:      "Elevated 30-day mortality risk detected",
		Timestamp:    time.Now().UnixMilli(),
		Metadata: models.AlertMetadata{
			SourceModule:       "MODULE5_ML_INFERENCE",
			ModelVersion:       "2.1.0",
			RequiresEscalation: false,
		},
	}

	if err := router.RouteAlert(ctx, mortalityAlert); err != nil {
		fmt.Printf("Failed to route mortality alert: %v\n", err)
	}

	// Moderate Vital Sign Anomaly
	vitalAlert := &models.Alert{
		AlertID:      "alert-vital-001",
		PatientID:    "PAT-005",
		DepartmentID: "DEPT-MED-SURG",
		AlertType:    models.AlertTypeVitalSignAnomaly,
		Severity:     models.SeverityModerate,
		Confidence:   0.85,
		Message:      "Blood pressure trending upward",
		PatientLocation: models.PatientLocation{
			Room: "MS-205",
			Bed:  "A",
		},
		VitalSigns: &models.VitalSigns{
			HeartRate:             95,
			BloodPressureSystolic: 165,
			Temperature:           37.1,
		},
		Timestamp: time.Now().UnixMilli(),
		Metadata: models.AlertMetadata{
			SourceModule:       "MODULE4_CEP",
			RequiresEscalation: false,
		},
	}

	if err := router.RouteAlert(ctx, vitalAlert); err != nil {
		fmt.Printf("Failed to route vital alert: %v\n", err)
	}
}

// Example 4: Custom Message Formatting
func ExampleCustomMessageFormatting() {
	router := &AlertRouter{
		logger: zap.NewNop(),
	}

	alert := &models.Alert{
		AlertID:      "alert-format-001",
		PatientID:    "PAT-006",
		AlertType:    models.AlertTypeSepsis,
		Severity:     models.SeverityCritical,
		Confidence:   0.94,
		Message:      "Critical sepsis alert with recommendations",
		PatientLocation: models.PatientLocation{
			Room: "ICU-15",
			Bed:  "C",
		},
		Recommendations: []string{
			"Immediate blood culture",
			"Start broad-spectrum antibiotics",
			"Fluid resuscitation",
		},
	}

	// Format for different channels
	channels := []models.NotificationChannel{
		models.ChannelSMS,
		models.ChannelPager,
		models.ChannelPush,
		models.ChannelEmail,
		models.ChannelVoice,
	}

	fmt.Println("Message Formatting Examples:")
	fmt.Println("=" + string(make([]byte, 60)) + "=")
	for _, channel := range channels {
		msg := router.formatMessageForChannel(alert, channel)
		fmt.Printf("\n%s:\n%s\n", channel, msg)
	}
}

// Mock Implementations (replace with actual implementations)

type MockFatigueTrackerImpl struct{}

func (m *MockFatigueTrackerImpl) ShouldSuppress(alert *models.Alert, user *models.User) (bool, string) {
	// Simple mock: don't suppress critical alerts
	if alert.Severity == models.SeverityCritical {
		return false, ""
	}
	// Simulate 10% suppression rate for others
	return false, ""
}

func (m *MockFatigueTrackerImpl) RecordNotification(userID string, alert *models.Alert) {
	// Mock implementation
}

type MockUserServiceImpl struct{}

func (m *MockUserServiceImpl) GetAttendingPhysician(departmentID string) ([]*models.User, error) {
	return []*models.User{
		{
			ID:           "user-attending-001",
			Name:         "Dr. Sarah Johnson",
			Email:        "s.johnson@hospital.com",
			PhoneNumber:  "+15551234567",
			PagerNumber:  "1001",
			Role:         "ATTENDING",
			DepartmentID: departmentID,
		},
	}, nil
}

func (m *MockUserServiceImpl) GetChargeNurse(departmentID string) ([]*models.User, error) {
	return []*models.User{
		{
			ID:           "user-charge-001",
			Name:         "Nurse Michael Chen",
			Email:        "m.chen@hospital.com",
			PhoneNumber:  "+15551234568",
			PagerNumber:  "2001",
			Role:         "CHARGE_NURSE",
			DepartmentID: departmentID,
		},
	}, nil
}

func (m *MockUserServiceImpl) GetPrimaryNurse(patientID string) ([]*models.User, error) {
	return []*models.User{
		{
			ID:          "user-primary-001",
			Name:        "Nurse Emily Rodriguez",
			Email:       "e.rodriguez@hospital.com",
			PhoneNumber: "+15551234569",
			FCMToken:    "fcm-token-primary-001",
			Role:        "PRIMARY_NURSE",
		},
	}, nil
}

func (m *MockUserServiceImpl) GetResident(departmentID string) ([]*models.User, error) {
	return []*models.User{
		{
			ID:           "user-resident-001",
			Name:         "Dr. James Wilson",
			Email:        "j.wilson@hospital.com",
			PhoneNumber:  "+15551234570",
			Role:         "RESIDENT",
			DepartmentID: departmentID,
		},
	}, nil
}

func (m *MockUserServiceImpl) GetClinicalInformaticsTeam() ([]*models.User, error) {
	return []*models.User{
		{
			ID:    "user-informatics-001",
			Name:  "Dr. Data Scientist",
			Email: "data.scientist@hospital.com",
			Role:  "INFORMATICS",
		},
		{
			ID:    "user-informatics-002",
			Name:  "ML Engineer",
			Email: "ml.engineer@hospital.com",
			Role:  "INFORMATICS",
		},
	}, nil
}

func (m *MockUserServiceImpl) GetPreferredChannels(user *models.User, severity models.AlertSeverity) []models.NotificationChannel {
	// Return default channels based on severity
	if channels, ok := models.DefaultSeverityChannels[severity]; ok {
		return channels
	}
	return []models.NotificationChannel{models.ChannelPush}
}

type MockDeliveryServiceImpl struct{}

func (m *MockDeliveryServiceImpl) Send(ctx context.Context, notification *models.Notification) error {
	// Mock implementation - in real implementation, this would call Twilio, SendGrid, etc.
	fmt.Printf("  [MOCK SEND] Channel: %s, User: %s, Priority: %d\n",
		notification.Channel,
		notification.User.Name,
		notification.Priority,
	)
	return nil
}

type MockEscalationManagerImpl struct{}

func (m *MockEscalationManagerImpl) ScheduleEscalation(ctx context.Context, alert *models.Alert, timeout time.Duration) error {
	// Mock implementation
	fmt.Printf("  [MOCK ESCALATION] Alert: %s, Timeout: %v\n",
		alert.AlertID,
		timeout,
	)
	return nil
}

// RunExamples runs all example scenarios
func RunExamples() {
	fmt.Println("\n========== Example 1: Kafka Consumer Integration ==========")
	ExampleKafkaConsumerIntegration()

	fmt.Println("\n========== Example 2: Routing Decision Preview ==========")
	ExampleRoutingDecisionPreview()

	fmt.Println("\n========== Example 3: Different Alert Types ==========")
	ExampleHandleDifferentAlertTypes()

	fmt.Println("\n========== Example 4: Message Formatting ==========")
	ExampleCustomMessageFormatting()
}
