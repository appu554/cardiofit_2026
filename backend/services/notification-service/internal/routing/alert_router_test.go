package routing

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/cardiofit/notification-service/internal/models"
)

// Mock implementations

type MockFatigueTracker struct {
	mock.Mock
}

func (m *MockFatigueTracker) ShouldSuppress(alert *models.Alert, user *models.User) (bool, string) {
	args := m.Called(alert, user)
	return args.Bool(0), args.String(1)
}

func (m *MockFatigueTracker) RecordNotification(userID string, alert *models.Alert) {
	m.Called(userID, alert)
}

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetAttendingPhysician(departmentID string) ([]*models.User, error) {
	args := m.Called(departmentID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetChargeNurse(departmentID string) ([]*models.User, error) {
	args := m.Called(departmentID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetPrimaryNurse(patientID string) ([]*models.User, error) {
	args := m.Called(patientID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetResident(departmentID string) ([]*models.User, error) {
	args := m.Called(departmentID)
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetClinicalInformaticsTeam() ([]*models.User, error) {
	args := m.Called()
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetPreferredChannels(user *models.User, severity models.AlertSeverity) []models.NotificationChannel {
	args := m.Called(user, severity)
	return args.Get(0).([]models.NotificationChannel)
}

type MockDeliveryService struct {
	mock.Mock
}

func (m *MockDeliveryService) Send(ctx context.Context, notification *models.Notification) error {
	args := m.Called(ctx, notification)
	return args.Error(0)
}

type MockEscalationManager struct {
	mock.Mock
}

func (m *MockEscalationManager) ScheduleEscalation(ctx context.Context, alert *models.Alert, timeout time.Duration) error {
	args := m.Called(ctx, alert, timeout)
	return args.Error(0)
}

// Test helpers

func createTestAlert(severity models.AlertSeverity, alertType models.AlertType) *models.Alert {
	return &models.Alert{
		AlertID:      "alert-123",
		PatientID:    "PAT-001",
		HospitalID:   "HOSP-001",
		DepartmentID: "DEPT-ICU",
		AlertType:    alertType,
		Severity:     severity,
		Confidence:   0.92,
		Message:      "Test alert message",
		Recommendations: []string{
			"Immediate physician review",
			"Check vitals",
		},
		PatientLocation: models.PatientLocation{
			Room: "ICU-5",
			Bed:  "A",
		},
		VitalSigns: &models.VitalSigns{
			HeartRate:             125,
			BloodPressureSystolic: 85,
			Temperature:           39.2,
		},
		Timestamp: time.Now().UnixMilli(),
		Metadata: models.AlertMetadata{
			SourceModule:       "MODULE4_CEP",
			RequiresEscalation: true,
		},
	}
}

func createTestUser(id, name, role string) *models.User {
	return &models.User{
		ID:           id,
		Name:         name,
		Email:        name + "@hospital.com",
		PhoneNumber:  "+1234567890",
		PagerNumber:  "1234567",
		FCMToken:     "fcm-token-" + id,
		Role:         role,
		DepartmentID: "DEPT-ICU",
	}
}

// Tests

func TestAlertRouter_RouteAlert_Critical(t *testing.T) {
	// Setup
	fatigueTracker := new(MockFatigueTracker)
	userService := new(MockUserService)
	deliveryService := new(MockDeliveryService)
	escalationMgr := new(MockEscalationManager)
	logger := zap.NewNop()

	router := NewAlertRouter(
		fatigueTracker,
		userService,
		deliveryService,
		escalationMgr,
		logger,
	)

	// Create test alert
	alert := createTestAlert(models.SeverityCritical, models.AlertTypeSepsis)

	// Create test users
	attendingDoc := createTestUser("user-001", "Dr. Smith", "ATTENDING")
	chargeNurse := createTestUser("user-002", "Nurse Johnson", "CHARGE_NURSE")

	// Setup expectations
	userService.On("GetAttendingPhysician", "DEPT-ICU").Return([]*models.User{attendingDoc}, nil)
	userService.On("GetChargeNurse", "DEPT-ICU").Return([]*models.User{chargeNurse}, nil)

	fatigueTracker.On("ShouldSuppress", alert, attendingDoc).Return(false, "")
	fatigueTracker.On("ShouldSuppress", alert, chargeNurse).Return(false, "")

	userService.On("GetPreferredChannels", attendingDoc, models.SeverityCritical).
		Return([]models.NotificationChannel{models.ChannelPager, models.ChannelSMS})
	userService.On("GetPreferredChannels", chargeNurse, models.SeverityCritical).
		Return([]models.NotificationChannel{models.ChannelSMS, models.ChannelPush})

	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	fatigueTracker.On("RecordNotification", "user-001", alert).Return()
	fatigueTracker.On("RecordNotification", "user-002", alert).Return()

	escalationMgr.On("ScheduleEscalation", mock.Anything, alert, 5*time.Minute).Return(nil)

	// Execute
	ctx := context.Background()
	err := router.RouteAlert(ctx, alert)

	// Assert
	assert.NoError(t, err)
	fatigueTracker.AssertExpectations(t)
	userService.AssertExpectations(t)
	escalationMgr.AssertExpectations(t)

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestAlertRouter_RouteAlert_High(t *testing.T) {
	// Setup
	fatigueTracker := new(MockFatigueTracker)
	userService := new(MockUserService)
	deliveryService := new(MockDeliveryService)
	escalationMgr := new(MockEscalationManager)
	logger := zap.NewNop()

	router := NewAlertRouter(
		fatigueTracker,
		userService,
		deliveryService,
		escalationMgr,
		logger,
	)

	// Create test alert
	alert := createTestAlert(models.SeverityHigh, models.AlertTypeDeterioration)

	// Create test users
	primaryNurse := createTestUser("user-003", "Nurse Wilson", "PRIMARY_NURSE")
	resident := createTestUser("user-004", "Dr. Brown", "RESIDENT")

	// Setup expectations
	userService.On("GetPrimaryNurse", "PAT-001").Return([]*models.User{primaryNurse}, nil)
	userService.On("GetResident", "DEPT-ICU").Return([]*models.User{resident}, nil)

	fatigueTracker.On("ShouldSuppress", alert, primaryNurse).Return(false, "")
	fatigueTracker.On("ShouldSuppress", alert, resident).Return(false, "")

	userService.On("GetPreferredChannels", primaryNurse, models.SeverityHigh).
		Return([]models.NotificationChannel{models.ChannelSMS, models.ChannelPush})
	userService.On("GetPreferredChannels", resident, models.SeverityHigh).
		Return([]models.NotificationChannel{models.ChannelSMS})

	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	fatigueTracker.On("RecordNotification", "user-003", alert).Return()
	fatigueTracker.On("RecordNotification", "user-004", alert).Return()

	escalationMgr.On("ScheduleEscalation", mock.Anything, alert, 15*time.Minute).Return(nil)

	// Execute
	ctx := context.Background()
	err := router.RouteAlert(ctx, alert)

	// Assert
	assert.NoError(t, err)
	userService.AssertExpectations(t)

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestAlertRouter_RouteAlert_Moderate(t *testing.T) {
	// Setup
	fatigueTracker := new(MockFatigueTracker)
	userService := new(MockUserService)
	deliveryService := new(MockDeliveryService)
	escalationMgr := new(MockEscalationManager)
	logger := zap.NewNop()

	router := NewAlertRouter(
		fatigueTracker,
		userService,
		deliveryService,
		escalationMgr,
		logger,
	)

	// Create test alert
	alert := createTestAlert(models.SeverityModerate, models.AlertTypeVitalSignAnomaly)
	alert.Metadata.RequiresEscalation = false

	// Create test user
	primaryNurse := createTestUser("user-005", "Nurse Davis", "PRIMARY_NURSE")

	// Setup expectations
	userService.On("GetPrimaryNurse", "PAT-001").Return([]*models.User{primaryNurse}, nil)

	fatigueTracker.On("ShouldSuppress", alert, primaryNurse).Return(false, "")

	userService.On("GetPreferredChannels", primaryNurse, models.SeverityModerate).
		Return([]models.NotificationChannel{models.ChannelPush, models.ChannelInApp})

	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	fatigueTracker.On("RecordNotification", "user-005", alert).Return()

	// Execute
	ctx := context.Background()
	err := router.RouteAlert(ctx, alert)

	// Assert
	assert.NoError(t, err)
	userService.AssertExpectations(t)
	escalationMgr.AssertNotCalled(t, "ScheduleEscalation") // No escalation for MODERATE

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestAlertRouter_RouteAlert_MLAlert(t *testing.T) {
	// Setup
	fatigueTracker := new(MockFatigueTracker)
	userService := new(MockUserService)
	deliveryService := new(MockDeliveryService)
	escalationMgr := new(MockEscalationManager)
	logger := zap.NewNop()

	router := NewAlertRouter(
		fatigueTracker,
		userService,
		deliveryService,
		escalationMgr,
		logger,
	)

	// Create test alert
	alert := createTestAlert(models.SeverityMLAlert, models.AlertTypeMortalityRisk)
	alert.Metadata.SourceModule = "MODULE5_ML_INFERENCE"
	alert.Metadata.ModelVersion = "1.2.3"

	// Create test users
	informaticsTeam := []*models.User{
		createTestUser("user-006", "Data Scientist", "INFORMATICS"),
		createTestUser("user-007", "ML Engineer", "INFORMATICS"),
	}

	// Setup expectations
	userService.On("GetClinicalInformaticsTeam").Return(informaticsTeam, nil)

	for _, user := range informaticsTeam {
		fatigueTracker.On("ShouldSuppress", alert, user).Return(false, "")
		userService.On("GetPreferredChannels", user, models.SeverityMLAlert).
			Return([]models.NotificationChannel{models.ChannelEmail, models.ChannelPush})
		fatigueTracker.On("RecordNotification", user.ID, alert).Return()
	}

	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Execute
	ctx := context.Background()
	err := router.RouteAlert(ctx, alert)

	// Assert
	assert.NoError(t, err)
	userService.AssertExpectations(t)

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestAlertRouter_RouteAlert_WithFatigueSuppression(t *testing.T) {
	// Setup
	fatigueTracker := new(MockFatigueTracker)
	userService := new(MockUserService)
	deliveryService := new(MockDeliveryService)
	escalationMgr := new(MockEscalationManager)
	logger := zap.NewNop()

	router := NewAlertRouter(
		fatigueTracker,
		userService,
		deliveryService,
		escalationMgr,
		logger,
	)

	// Create test alert
	alert := createTestAlert(models.SeverityHigh, models.AlertTypeDeterioration)

	// Create test users
	primaryNurse := createTestUser("user-008", "Nurse Taylor", "PRIMARY_NURSE")
	resident := createTestUser("user-009", "Dr. Martinez", "RESIDENT")

	// Setup expectations
	userService.On("GetPrimaryNurse", "PAT-001").Return([]*models.User{primaryNurse}, nil)
	userService.On("GetResident", "DEPT-ICU").Return([]*models.User{resident}, nil)

	// Primary nurse suppressed due to rate limit
	fatigueTracker.On("ShouldSuppress", alert, primaryNurse).Return(true, "rate_limit_exceeded")
	// Resident not suppressed
	fatigueTracker.On("ShouldSuppress", alert, resident).Return(false, "")

	userService.On("GetPreferredChannels", resident, models.SeverityHigh).
		Return([]models.NotificationChannel{models.ChannelSMS})

	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	fatigueTracker.On("RecordNotification", "user-009", alert).Return()

	escalationMgr.On("ScheduleEscalation", mock.Anything, alert, 15*time.Minute).Return(nil)

	// Execute
	ctx := context.Background()
	err := router.RouteAlert(ctx, alert)

	// Assert
	assert.NoError(t, err)
	fatigueTracker.AssertExpectations(t)
	fatigueTracker.AssertNotCalled(t, "RecordNotification", "user-008", alert) // Suppressed user

	// Give goroutines time to complete
	time.Sleep(100 * time.Millisecond)
}

func TestAlertRouter_GetRoutingDecision(t *testing.T) {
	// Setup
	fatigueTracker := new(MockFatigueTracker)
	userService := new(MockUserService)
	deliveryService := new(MockDeliveryService)
	escalationMgr := new(MockEscalationManager)
	logger := zap.NewNop()

	router := NewAlertRouter(
		fatigueTracker,
		userService,
		deliveryService,
		escalationMgr,
		logger,
	)

	// Create test alert
	alert := createTestAlert(models.SeverityCritical, models.AlertTypeSepsis)

	// Create test users
	attendingDoc := createTestUser("user-010", "Dr. Anderson", "ATTENDING")
	chargeNurse := createTestUser("user-011", "Nurse White", "CHARGE_NURSE")

	// Setup expectations
	userService.On("GetAttendingPhysician", "DEPT-ICU").Return([]*models.User{attendingDoc}, nil)
	userService.On("GetChargeNurse", "DEPT-ICU").Return([]*models.User{chargeNurse}, nil)

	fatigueTracker.On("ShouldSuppress", alert, attendingDoc).Return(false, "")
	fatigueTracker.On("ShouldSuppress", alert, chargeNurse).Return(true, "duplicate")

	userService.On("GetPreferredChannels", attendingDoc, models.SeverityCritical).
		Return([]models.NotificationChannel{models.ChannelPager, models.ChannelSMS})

	// Execute
	ctx := context.Background()
	decision, err := router.GetRoutingDecision(ctx, alert)

	// Assert
	assert.NoError(t, err)
	assert.NotNil(t, decision)
	assert.Equal(t, 2, len(decision.TargetUsers))
	assert.Equal(t, true, decision.RequiresEscalation)
	assert.Equal(t, 5*time.Minute, decision.EscalationTimeout)
	assert.Equal(t, 2, len(decision.UserChannels["user-010"]))
	assert.Equal(t, "duplicate", decision.SuppressedUsers["user-011"])

	userService.AssertExpectations(t)
	fatigueTracker.AssertExpectations(t)
}

func TestAlertRouter_FormatMessageForChannel(t *testing.T) {
	router := &AlertRouter{
		logger: zap.NewNop(),
	}

	alert := createTestAlert(models.SeverityCritical, models.AlertTypeSepsis)

	tests := []struct {
		name     string
		channel  models.NotificationChannel
		validate func(t *testing.T, msg string)
	}{
		{
			name:    "SMS format",
			channel: models.ChannelSMS,
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "CRITICAL")
				assert.Contains(t, msg, "PAT-001")
				assert.Contains(t, msg, "ICU-5")
				assert.LessOrEqual(t, len(msg), 160)
			},
		},
		{
			name:    "Pager format",
			channel: models.ChannelPager,
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "CRIT")
				assert.Contains(t, msg, "PAT-001")
				assert.LessOrEqual(t, len(msg), 100)
			},
		},
		{
			name:    "Push format",
			channel: models.ChannelPush,
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "CRITICAL")
				assert.Contains(t, msg, "Alert")
				assert.Contains(t, msg, "PAT-001")
			},
		},
		{
			name:    "Email format",
			channel: models.ChannelEmail,
			validate: func(t *testing.T, msg string) {
				assert.Equal(t, alert.Message, msg)
			},
		},
		{
			name:    "Voice format",
			channel: models.ChannelVoice,
			validate: func(t *testing.T, msg string) {
				assert.Contains(t, msg, "Critical alert")
				assert.Contains(t, msg, "Patient")
				assert.Contains(t, msg, "percent")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			msg := router.formatMessageForChannel(alert, tt.channel)
			tt.validate(t, msg)
		})
	}
}

func TestAlertRouter_SeverityToPriority(t *testing.T) {
	router := &AlertRouter{}

	tests := []struct {
		severity         models.AlertSeverity
		expectedPriority int
	}{
		{models.SeverityCritical, 1},
		{models.SeverityHigh, 2},
		{models.SeverityModerate, 3},
		{models.SeverityLow, 4},
		{models.SeverityMLAlert, 3},
	}

	for _, tt := range tests {
		t.Run(string(tt.severity), func(t *testing.T) {
			priority := router.severityToPriority(tt.severity)
			assert.Equal(t, tt.expectedPriority, priority)
		})
	}
}
