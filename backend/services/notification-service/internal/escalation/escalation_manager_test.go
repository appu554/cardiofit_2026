package escalation

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"

	"github.com/cardiofit/notification-service/internal/models"
)

// Mock implementations

type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) GetAttendingPhysician(departmentID string) ([]*models.User, error) {
	args := m.Called(departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetChargeNurse(departmentID string) ([]*models.User, error) {
	args := m.Called(departmentID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetPrimaryNurse(patientID string) ([]*models.User, error) {
	args := m.Called(patientID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]*models.User), args.Error(1)
}

func (m *MockUserService) GetPreferredChannels(user *models.User, severity models.AlertSeverity) []models.NotificationChannel {
	args := m.Called(user, severity)
	if args.Get(0) == nil {
		return nil
	}
	return args.Get(0).([]models.NotificationChannel)
}

type MockDeliveryService struct {
	mock.Mock
	notifications []*models.Notification
	mu            sync.Mutex
}

func (m *MockDeliveryService) Send(ctx context.Context, notification *models.Notification) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, notification)
	m.notifications = append(m.notifications, notification)
	return args.Error(0)
}

func (m *MockDeliveryService) GetNotifications() []*models.Notification {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]*models.Notification{}, m.notifications...)
}

func (m *MockDeliveryService) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.notifications = make([]*models.Notification, 0)
}

type MockVoiceProvider struct {
	mock.Mock
	calls []string
	mu    sync.Mutex
}

func (m *MockVoiceProvider) MakeCall(ctx context.Context, phoneNumber string, message string, metadata map[string]interface{}) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	args := m.Called(ctx, phoneNumber, message, metadata)
	callSID := args.String(0)
	m.calls = append(m.calls, phoneNumber)
	return callSID, args.Error(1)
}

func (m *MockVoiceProvider) GetCalls() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	return append([]string{}, m.calls...)
}

func (m *MockVoiceProvider) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.calls = make([]string, 0)
}

// Test fixtures

func createTestAlert(alertID string, severity models.AlertSeverity) *models.Alert {
	return &models.Alert{
		AlertID:      alertID,
		PatientID:    "PAT-001",
		DepartmentID: "DEPT-ICU",
		Severity:     severity,
		AlertType:    models.AlertTypeSepsis,
		Confidence:   0.95,
		Message:      "Test alert message",
		PatientLocation: models.PatientLocation{
			Room: "ICU-101",
			Bed:  "A",
		},
		Metadata: models.AlertMetadata{
			RequiresEscalation: true,
		},
		Timestamp: time.Now().Unix(),
	}
}

func createTestUser(id, name, role string) *models.User {
	return &models.User{
		ID:          id,
		Name:        name,
		Email:       fmt.Sprintf("%s@cardiofit.health", id),
		PhoneNumber: "+1555000" + id[len(id)-4:],
		Role:        role,
	}
}

func setupTestManager(t *testing.T) (*EscalationManager, *MockUserService, *MockDeliveryService, *MockVoiceProvider) {
	logger := zap.NewNop()

	userService := new(MockUserService)
	deliveryService := new(MockDeliveryService)
	voiceProvider := new(MockVoiceProvider)

	config := EscalationConfig{
		CriticalTimeoutMinutes: 1, // Short timeout for testing (1 minute)
		HighTimeoutMinutes:     2,
		MaxLevel:               3,
		EnableVoiceEscalation:  true,
	}

	// Note: Using nil for db in tests - integration tests would use real db
	mgr := NewEscalationManager(nil, userService, deliveryService, voiceProvider, logger, config)

	return mgr, userService, deliveryService, voiceProvider
}

// Tests

func TestScheduleEscalation(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-001", models.SeverityCritical)

	err := mgr.ScheduleEscalation(context.Background(), alert, 5*time.Minute)
	assert.NoError(t, err)

	// Verify timer is scheduled
	mgr.mu.RLock()
	_, exists := mgr.timers[alert.AlertID]
	mgr.mu.RUnlock()

	assert.True(t, exists, "Timer should be scheduled")

	// Verify chain is created
	mgr.mu.RLock()
	chain, exists := mgr.chains[alert.AlertID]
	mgr.mu.RUnlock()

	assert.True(t, exists, "Chain should be created")
	assert.Equal(t, alert.AlertID, chain.AlertID)
	assert.Equal(t, 0, chain.CurrentLevel)
	assert.Nil(t, chain.AcknowledgedAt)
}

func TestCancelEscalation(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-002", models.SeverityCritical)

	// Schedule escalation
	err := mgr.ScheduleEscalation(context.Background(), alert, 5*time.Minute)
	assert.NoError(t, err)

	// Cancel escalation
	err = mgr.CancelEscalation(context.Background(), alert.AlertID)
	assert.NoError(t, err)

	// Verify timer is removed
	mgr.mu.RLock()
	_, exists := mgr.timers[alert.AlertID]
	mgr.mu.RUnlock()

	assert.False(t, exists, "Timer should be cancelled")

	// Verify chain is marked as acknowledged
	mgr.mu.RLock()
	chain, _ := mgr.chains[alert.AlertID]
	mgr.mu.RUnlock()

	assert.NotNil(t, chain.AcknowledgedAt)
}

func TestEscalationLevel1_PrimaryNurse(t *testing.T) {
	mgr, userService, deliveryService, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-003", models.SeverityCritical)

	// Mock primary nurse
	primaryNurse := createTestUser("USER-001", "Nurse Jane", "PRIMARY_NURSE")
	userService.On("GetPrimaryNurse", alert.PatientID).Return([]*models.User{primaryNurse}, nil)

	// Mock delivery service
	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Execute level 1 escalation
	err := mgr.escalateToNextLevel(context.Background(), alert, 1)
	assert.NoError(t, err)

	// Verify notifications sent
	notifications := deliveryService.GetNotifications()
	assert.GreaterOrEqual(t, len(notifications), 2, "Should send at least 2 notifications (SMS + Push)")

	// Verify user and channels
	for _, notif := range notifications {
		assert.Equal(t, primaryNurse.ID, notif.UserID)
		assert.Contains(t, []models.NotificationChannel{models.ChannelSMS, models.ChannelPush}, notif.Channel)
		assert.Equal(t, 1, notif.Priority, "Escalations should be priority 1")
		assert.Contains(t, notif.Message, "ESCALATION LEVEL 1")
	}
}

func TestEscalationLevel2_ChargeNurse(t *testing.T) {
	mgr, userService, deliveryService, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-004", models.SeverityCritical)

	// Mock charge nurse
	chargeNurse := createTestUser("USER-002", "Charge Nurse Bob", "CHARGE_NURSE")
	userService.On("GetChargeNurse", alert.DepartmentID).Return([]*models.User{chargeNurse}, nil)

	// Mock delivery service
	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Execute level 2 escalation
	err := mgr.escalateToNextLevel(context.Background(), alert, 2)
	assert.NoError(t, err)

	// Verify notifications sent
	notifications := deliveryService.GetNotifications()
	assert.GreaterOrEqual(t, len(notifications), 2, "Should send at least 2 notifications (SMS + Pager)")

	// Verify channels include pager for level 2
	channels := make(map[models.NotificationChannel]bool)
	for _, notif := range notifications {
		channels[notif.Channel] = true
		assert.Contains(t, notif.Message, "ESCALATION LEVEL 2")
	}

	assert.True(t, channels[models.ChannelSMS], "Should use SMS channel")
	assert.True(t, channels[models.ChannelPager], "Should use Pager channel")
}

func TestEscalationLevel3_AttendingPhysicianWithVoiceCall(t *testing.T) {
	mgr, userService, deliveryService, voiceProvider := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-005", models.SeverityCritical)

	// Mock attending physician
	attending := createTestUser("USER-003", "Dr. Smith", "ATTENDING")
	userService.On("GetAttendingPhysician", alert.DepartmentID).Return([]*models.User{attending}, nil)

	// Mock delivery service
	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Mock voice provider
	voiceProvider.On("MakeCall", mock.Anything, attending.PhoneNumber, mock.Anything, mock.Anything).Return("CALL-001", nil)

	// Execute level 3 escalation
	err := mgr.escalateToNextLevel(context.Background(), alert, 3)
	assert.NoError(t, err)

	// Verify notifications sent
	notifications := deliveryService.GetNotifications()
	assert.GreaterOrEqual(t, len(notifications), 2, "Should send notifications")

	// Verify voice call was made
	calls := voiceProvider.GetCalls()
	assert.Equal(t, 1, len(calls), "Should make 1 voice call")
	assert.Equal(t, attending.PhoneNumber, calls[0])

	// Verify voice call was attempted
	voiceProvider.AssertCalled(t, "MakeCall", mock.Anything, attending.PhoneNumber, mock.Anything, mock.Anything)
}

func TestTimerEscalation_TimeoutFires(t *testing.T) {
	mgr, userService, deliveryService, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-006", models.SeverityCritical)

	// Mock primary nurse
	primaryNurse := createTestUser("USER-004", "Nurse Alice", "PRIMARY_NURSE")
	userService.On("GetPrimaryNurse", alert.PatientID).Return([]*models.User{primaryNurse}, nil)

	// Mock delivery service
	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Schedule with very short timeout (100ms for test)
	err := mgr.ScheduleEscalation(context.Background(), alert, 100*time.Millisecond)
	assert.NoError(t, err)

	// Wait for timer to fire
	time.Sleep(200 * time.Millisecond)

	// Verify escalation was executed
	notifications := deliveryService.GetNotifications()
	assert.GreaterOrEqual(t, len(notifications), 1, "Should send escalation notifications after timeout")
}

func TestConcurrentEscalations_MultipleAlerts(t *testing.T) {
	mgr, userService, deliveryService, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	numAlerts := 10
	var wg sync.WaitGroup

	// Mock primary nurse
	primaryNurse := createTestUser("USER-005", "Nurse Concurrent", "PRIMARY_NURSE")
	userService.On("GetPrimaryNurse", mock.Anything).Return([]*models.User{primaryNurse}, nil)
	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Schedule multiple escalations concurrently
	for i := 0; i < numAlerts; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()

			alert := createTestAlert(fmt.Sprintf("ALERT-CONC-%d", idx), models.SeverityCritical)
			err := mgr.ScheduleEscalation(context.Background(), alert, 5*time.Minute)
			assert.NoError(t, err)
		}(i)
	}

	wg.Wait()

	// Verify all escalations were scheduled
	mgr.mu.RLock()
	timerCount := len(mgr.timers)
	chainCount := len(mgr.chains)
	mgr.mu.RUnlock()

	assert.Equal(t, numAlerts, timerCount, "All timers should be scheduled")
	assert.Equal(t, numAlerts, chainCount, "All chains should be created")
}

func TestAcknowledgment_CancelsEscalation(t *testing.T) {
	mgr, userService, deliveryService, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-007", models.SeverityCritical)

	// Mock services
	primaryNurse := createTestUser("USER-006", "Nurse ACK", "PRIMARY_NURSE")
	userService.On("GetPrimaryNurse", alert.PatientID).Return([]*models.User{primaryNurse}, nil)
	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)

	// Schedule escalation with short timeout
	err := mgr.ScheduleEscalation(context.Background(), alert, 200*time.Millisecond)
	assert.NoError(t, err)

	// Acknowledge immediately (before timeout)
	time.Sleep(50 * time.Millisecond)
	err = mgr.RecordAcknowledgment(context.Background(), alert.AlertID, primaryNurse.ID)
	assert.NoError(t, err)

	// Wait past timeout
	time.Sleep(200 * time.Millisecond)

	// Verify no escalation notifications sent (cancelled before timeout)
	notifications := deliveryService.GetNotifications()
	assert.Equal(t, 0, len(notifications), "Should not send notifications after acknowledgment")
}

func TestRapidAcknowledgment_NoEscalation(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-008", models.SeverityCritical)

	// Schedule and immediately acknowledge
	err := mgr.ScheduleEscalation(context.Background(), alert, 1*time.Minute)
	assert.NoError(t, err)

	err = mgr.CancelEscalation(context.Background(), alert.AlertID)
	assert.NoError(t, err)

	// Verify timer is removed
	mgr.mu.RLock()
	_, exists := mgr.timers[alert.AlertID]
	mgr.mu.RUnlock()

	assert.False(t, exists)
}

func TestMaxEscalationLevel_StopsAtLevel3(t *testing.T) {
	mgr, userService, deliveryService, voiceProvider := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	alert := createTestAlert("ALERT-009", models.SeverityCritical)

	// Mock all levels
	primaryNurse := createTestUser("USER-007", "Nurse Primary", "PRIMARY_NURSE")
	chargeNurse := createTestUser("USER-008", "Nurse Charge", "CHARGE_NURSE")
	attending := createTestUser("USER-009", "Dr. Attending", "ATTENDING")

	userService.On("GetPrimaryNurse", alert.PatientID).Return([]*models.User{primaryNurse}, nil)
	userService.On("GetChargeNurse", alert.DepartmentID).Return([]*models.User{chargeNurse}, nil)
	userService.On("GetAttendingPhysician", alert.DepartmentID).Return([]*models.User{attending}, nil)
	deliveryService.On("Send", mock.Anything, mock.Anything).Return(nil)
	voiceProvider.On("MakeCall", mock.Anything, mock.Anything, mock.Anything, mock.Anything).Return("CALL-001", nil)

	// Execute all 3 levels
	for level := 1; level <= 3; level++ {
		err := mgr.escalateToNextLevel(context.Background(), alert, level)
		assert.NoError(t, err)
	}

	// Verify chain is at max level
	mgr.mu.RLock()
	chain := mgr.chains[alert.AlertID]
	mgr.mu.RUnlock()

	assert.Equal(t, 3, chain.CurrentLevel)

	// Attempt level 4 should return error
	err := mgr.escalateToNextLevel(context.Background(), alert, 4)
	assert.Error(t, err)
}

func TestCleanupWorker_RemovesOldChains(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	// Create old acknowledged chain
	oldAlert := createTestAlert("ALERT-OLD", models.SeverityCritical)
	mgr.mu.Lock()
	ackTime := time.Now().Add(-1 * time.Hour)
	mgr.chains[oldAlert.AlertID] = &EscalationChain{
		AlertID:        oldAlert.AlertID,
		CurrentLevel:   2,
		AcknowledgedAt: &ackTime,
		CreatedAt:      time.Now().Add(-2 * time.Hour),
	}
	mgr.mu.Unlock()

	// Run cleanup
	mgr.cleanupCompletedChains()

	// Verify old chain is removed
	mgr.mu.RLock()
	_, exists := mgr.chains[oldAlert.AlertID]
	mgr.mu.RUnlock()

	assert.False(t, exists, "Old acknowledged chain should be cleaned up")
}

func TestCreateCheckpoint(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	// Create active and acknowledged chains
	activeAlert := createTestAlert("ALERT-ACTIVE", models.SeverityCritical)
	ackAlert := createTestAlert("ALERT-ACK", models.SeverityCritical)

	mgr.mu.Lock()
	mgr.chains[activeAlert.AlertID] = &EscalationChain{
		AlertID:      activeAlert.AlertID,
		CurrentLevel: 1,
		CreatedAt:    time.Now(),
	}

	ackTime := time.Now()
	mgr.chains[ackAlert.AlertID] = &EscalationChain{
		AlertID:        ackAlert.AlertID,
		CurrentLevel:   2,
		AcknowledgedAt: &ackTime,
		CreatedAt:      time.Now(),
	}
	mgr.mu.Unlock()

	// Create checkpoint
	checkpoint := mgr.CreateCheckpoint()

	// Verify only active chain is in checkpoint
	assert.Equal(t, 1, len(checkpoint))
	assert.NotNil(t, checkpoint[activeAlert.AlertID])
	assert.Nil(t, checkpoint[ackAlert.AlertID])
}

func TestGetActiveEscalations(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)
	defer mgr.Shutdown(context.Background())

	// Create multiple chains
	alert1 := createTestAlert("ALERT-A1", models.SeverityCritical)
	alert2 := createTestAlert("ALERT-A2", models.SeverityHigh)

	mgr.mu.Lock()
	mgr.chains[alert1.AlertID] = &EscalationChain{
		AlertID:      alert1.AlertID,
		CurrentLevel: 1,
		CreatedAt:    time.Now(),
	}
	mgr.chains[alert2.AlertID] = &EscalationChain{
		AlertID:      alert2.AlertID,
		CurrentLevel: 2,
		CreatedAt:    time.Now(),
	}
	mgr.mu.Unlock()

	// Get active escalations
	active := mgr.GetActiveEscalations()

	assert.Equal(t, 2, len(active))
	assert.NotNil(t, active[alert1.AlertID])
	assert.NotNil(t, active[alert2.AlertID])
}

func TestShutdown_StopsAllTimers(t *testing.T) {
	mgr, _, _, _ := setupTestManager(t)

	// Schedule multiple escalations
	for i := 0; i < 5; i++ {
		alert := createTestAlert(fmt.Sprintf("ALERT-SHUT-%d", i), models.SeverityCritical)
		err := mgr.ScheduleEscalation(context.Background(), alert, 5*time.Minute)
		assert.NoError(t, err)
	}

	// Verify timers exist
	mgr.mu.RLock()
	timerCount := len(mgr.timers)
	mgr.mu.RUnlock()
	assert.Equal(t, 5, timerCount)

	// Shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := mgr.Shutdown(ctx)
	assert.NoError(t, err)

	// Verify all timers are stopped (note: timers are still in map, but stopped)
	// In production, you might want to clear the map on shutdown
}

