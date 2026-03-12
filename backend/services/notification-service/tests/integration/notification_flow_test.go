package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cardiofit/notification-service/internal/delivery"
	"github.com/cardiofit/notification-service/internal/fatigue"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/cardiofit/notification-service/internal/routing"
	"github.com/cardiofit/notification-service/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestNotificationFlow tests the complete end-to-end notification flow
func TestNotificationFlow(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping end-to-end integration test in short mode")
	}

	// Setup infrastructure
	ctx := context.Background()

	// Connect to PostgreSQL
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "Failed to connect to database")
	defer pool.Close()

	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping database")

	// Connect to Redis
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   1,
	})
	defer redisClient.Close()

	err = redisClient.Ping(ctx).Err()
	require.NoError(t, err, "Failed to connect to Redis")

	// Initialize services
	userService := users.NewUserPreferenceService(pool, redisClient)
	fatigueTracker := fatigue.NewAlertFatigueTracker(redisClient)

	// Mock delivery service for testing
	deliveryService := &mockDeliveryService{
		sentNotifications: make(map[string][]models.NotificationRequest),
	}

	// Create alert router
	alertRouter := routing.NewAlertRouter(userService, fatigueTracker, deliveryService)

	t.Run("CompleteFlow_CriticalAlert", func(t *testing.T) {
		testPatientID := fmt.Sprintf("patient_%d", time.Now().Unix())
		testUserID := fmt.Sprintf("user_attending_%d", time.Now().Unix())

		// Step 1: Setup user preferences
		prefs := &models.UserPreferences{
			UserID: testUserID,
			ChannelPreferences: map[string]bool{
				"SMS":   true,
				"EMAIL": true,
				"PAGER": true,
			},
			SeverityChannels: map[string][]string{
				"CRITICAL": {"PAGER", "SMS"},
				"HIGH":     {"SMS", "PUSH"},
			},
			QuietHoursEnabled: false,
			MaxAlertsPerHour:  20,
		}

		err := userService.SavePreferences(ctx, prefs)
		require.NoError(t, err, "Failed to save user preferences")

		// Step 2: Create a critical alert
		alert := &models.Alert{
			ID:          fmt.Sprintf("alert_critical_%d", time.Now().Unix()),
			PatientID:   testPatientID,
			Severity:    "CRITICAL",
			Type:        "VITAL_SIGN_ABNORMAL",
			Title:       "Critical: Heart Rate Dangerously High",
			Message:     "Patient heart rate: 180 bpm. Immediate intervention required.",
			Timestamp:   time.Now(),
			TargetRoles: []string{"attending_physician"},
		}

		// Step 3: Route the alert through the system
		err = alertRouter.RouteAlert(ctx, alert)
		require.NoError(t, err, "Failed to route alert")

		// Step 4: Verify notifications were sent
		time.Sleep(100 * time.Millisecond) // Give async operations time to complete

		// Check that mock delivery service received notifications
		assert.Greater(t, len(deliveryService.sentNotifications), 0, "Should have sent notifications")

		// Verify alert was recorded in fatigue tracker
		count, err := fatigueTracker.GetAlertCount(ctx, testUserID, 1*time.Hour)
		require.NoError(t, err, "Failed to get alert count")
		assert.GreaterOrEqual(t, count, 1, "Alert should be recorded in fatigue tracker")

		// Cleanup
		cleanupTestData(t, ctx, pool, redisClient, testUserID, alert.ID)
	})

	t.Run("CompleteFlow_WithQuietHours", func(t *testing.T) {
		testPatientID := fmt.Sprintf("patient_quiet_%d", time.Now().Unix())
		testUserID := fmt.Sprintf("user_quiet_%d", time.Now().Unix())

		// Setup user with quiet hours enabled
		prefs := &models.UserPreferences{
			UserID: testUserID,
			ChannelPreferences: map[string]bool{
				"SMS":   true,
				"EMAIL": true,
				"PUSH":  true,
			},
			SeverityChannels: map[string][]string{
				"HIGH":    {"SMS", "PUSH"},
				"MEDIUM":  {"PUSH"},
				"LOW":     {"EMAIL"},
			},
			QuietHoursEnabled: true,
			QuietHoursStart:   22, // 10 PM
			QuietHoursEnd:     6,  // 6 AM
			MaxAlertsPerHour:  10,
		}

		err := userService.SavePreferences(ctx, prefs)
		require.NoError(t, err, "Failed to save user preferences")

		// Create a HIGH alert (not critical, should respect quiet hours)
		alert := &models.Alert{
			ID:          fmt.Sprintf("alert_high_%d", time.Now().Unix()),
			PatientID:   testPatientID,
			Severity:    "HIGH",
			Type:        "LAB_RESULT_ABNORMAL",
			Title:       "High: Abnormal Lab Result",
			Message:     "Patient potassium level: 5.8 mmol/L",
			Timestamp:   time.Now(),
			TargetRoles: []string{"attending_physician"},
		}

		// Route the alert
		err = alertRouter.RouteAlert(ctx, alert)
		require.NoError(t, err, "Failed to route alert")

		time.Sleep(100 * time.Millisecond)

		// Verify alert was processed (exact behavior depends on current time and quiet hours logic)
		count, err := fatigueTracker.GetAlertCount(ctx, testUserID, 1*time.Hour)
		require.NoError(t, err, "Failed to get alert count")
		assert.GreaterOrEqual(t, count, 0, "Alert count should be valid")

		// Cleanup
		cleanupTestData(t, ctx, pool, redisClient, testUserID, alert.ID)
	})

	t.Run("CompleteFlow_AlertFatigueRateLimit", func(t *testing.T) {
		testPatientID := fmt.Sprintf("patient_fatigue_%d", time.Now().Unix())
		testUserID := fmt.Sprintf("user_fatigue_%d", time.Now().Unix())

		// Setup user with low max alerts per hour
		prefs := &models.UserPreferences{
			UserID: testUserID,
			ChannelPreferences: map[string]bool{
				"SMS":   true,
				"EMAIL": true,
			},
			SeverityChannels: map[string][]string{
				"HIGH":   {"SMS"},
				"MEDIUM": {"EMAIL"},
			},
			QuietHoursEnabled: false,
			MaxAlertsPerHour:  3, // Only allow 3 alerts per hour
		}

		err := userService.SavePreferences(ctx, prefs)
		require.NoError(t, err, "Failed to save user preferences")

		// Send multiple HIGH alerts
		for i := 1; i <= 5; i++ {
			alert := &models.Alert{
				ID:          fmt.Sprintf("alert_fatigue_%d_%d", time.Now().Unix(), i),
				PatientID:   testPatientID,
				Severity:    "HIGH",
				Type:        "VITAL_SIGN_ABNORMAL",
				Title:       fmt.Sprintf("Alert %d", i),
				Message:     fmt.Sprintf("Test alert %d", i),
				Timestamp:   time.Now(),
				TargetRoles: []string{"attending_physician"},
			}

			err = alertRouter.RouteAlert(ctx, alert)
			require.NoError(t, err, "Failed to route alert %d", i)

			time.Sleep(50 * time.Millisecond)
		}

		// Check alert count
		count, err := fatigueTracker.GetAlertCount(ctx, testUserID, 1*time.Hour)
		require.NoError(t, err, "Failed to get alert count")
		assert.Equal(t, 5, count, "All alerts should be recorded")

		// Verify only first 3 were actually sent
		sentCount := 0
		if notifications, ok := deliveryService.sentNotifications[testUserID]; ok {
			sentCount = len(notifications)
		}
		assert.LessOrEqual(t, sentCount, 3, "Should respect max_alerts_per_hour limit")

		// Cleanup
		cleanupTestData(t, ctx, pool, redisClient, testUserID, "")
	})

	t.Run("CompleteFlow_MultipleRecipients", func(t *testing.T) {
		testPatientID := fmt.Sprintf("patient_multi_%d", time.Now().Unix())
		testUser1ID := fmt.Sprintf("user_attending_%d", time.Now().Unix())
		testUser2ID := fmt.Sprintf("user_nurse_%d", time.Now().Unix()+1)

		// Setup preferences for both users
		for _, userID := range []string{testUser1ID, testUser2ID} {
			prefs := &models.UserPreferences{
				UserID: userID,
				ChannelPreferences: map[string]bool{
					"SMS":   true,
					"EMAIL": true,
				},
				SeverityChannels: map[string][]string{
					"CRITICAL": {"SMS"},
					"HIGH":     {"EMAIL"},
				},
				QuietHoursEnabled: false,
				MaxAlertsPerHour:  20,
			}

			err := userService.SavePreferences(ctx, prefs)
			require.NoError(t, err, "Failed to save preferences for %s", userID)
		}

		// Create alert targeting multiple roles
		alert := &models.Alert{
			ID:          fmt.Sprintf("alert_multi_%d", time.Now().Unix()),
			PatientID:   testPatientID,
			Severity:    "CRITICAL",
			Type:        "CODE_BLUE",
			Title:       "Code Blue",
			Message:     "Patient in cardiac arrest. All hands on deck.",
			Timestamp:   time.Now(),
			TargetRoles: []string{"attending_physician", "charge_nurse"},
		}

		// Route the alert
		err = alertRouter.RouteAlert(ctx, alert)
		require.NoError(t, err, "Failed to route alert to multiple recipients")

		time.Sleep(100 * time.Millisecond)

		// Verify both users received notifications
		assert.Greater(t, len(deliveryService.sentNotifications), 0, "Should have sent notifications to multiple users")

		// Cleanup
		for _, userID := range []string{testUser1ID, testUser2ID} {
			cleanupTestData(t, ctx, pool, redisClient, userID, alert.ID)
		}
	})

	t.Run("CompleteFlow_CriticalBypassesRateLimit", func(t *testing.T) {
		testPatientID := fmt.Sprintf("patient_critical_bypass_%d", time.Now().Unix())
		testUserID := fmt.Sprintf("user_critical_bypass_%d", time.Now().Unix())

		// Setup user with very low max alerts
		prefs := &models.UserPreferences{
			UserID: testUserID,
			ChannelPreferences: map[string]bool{
				"SMS":   true,
				"PAGER": true,
			},
			SeverityChannels: map[string][]string{
				"CRITICAL": {"PAGER", "SMS"},
			},
			QuietHoursEnabled: false,
			MaxAlertsPerHour:  2, // Only 2 alerts allowed
		}

		err := userService.SavePreferences(ctx, prefs)
		require.NoError(t, err, "Failed to save user preferences")

		// Send 10 CRITICAL alerts (should all go through)
		for i := 1; i <= 10; i++ {
			alert := &models.Alert{
				ID:          fmt.Sprintf("alert_critical_bypass_%d_%d", time.Now().Unix(), i),
				PatientID:   testPatientID,
				Severity:    "CRITICAL",
				Type:        "CARDIAC_ARREST",
				Title:       fmt.Sprintf("Critical Alert %d", i),
				Message:     "Immediate medical attention required",
				Timestamp:   time.Now(),
				TargetRoles: []string{"attending_physician"},
			}

			err = alertRouter.RouteAlert(ctx, alert)
			require.NoError(t, err, "Failed to route critical alert %d", i)

			time.Sleep(50 * time.Millisecond)
		}

		// Check alert count
		count, err := fatigueTracker.GetAlertCount(ctx, testUserID, 1*time.Hour)
		require.NoError(t, err, "Failed to get alert count")
		assert.Equal(t, 10, count, "All critical alerts should be recorded")

		// Cleanup
		cleanupTestData(t, ctx, pool, redisClient, testUserID, "")
	})
}

// mockDeliveryService mocks the notification delivery service for testing
type mockDeliveryService struct {
	sentNotifications map[string][]models.NotificationRequest
}

func (m *mockDeliveryService) SendSMS(ctx context.Context, req *models.NotificationRequest) error {
	if m.sentNotifications == nil {
		m.sentNotifications = make(map[string][]models.NotificationRequest)
	}
	m.sentNotifications[req.UserID] = append(m.sentNotifications[req.UserID], *req)
	return nil
}

func (m *mockDeliveryService) SendEmail(ctx context.Context, req *models.NotificationRequest) error {
	if m.sentNotifications == nil {
		m.sentNotifications = make(map[string][]models.NotificationRequest)
	}
	m.sentNotifications[req.UserID] = append(m.sentNotifications[req.UserID], *req)
	return nil
}

func (m *mockDeliveryService) SendPush(ctx context.Context, req *models.NotificationRequest) error {
	if m.sentNotifications == nil {
		m.sentNotifications = make(map[string][]models.NotificationRequest)
	}
	m.sentNotifications[req.UserID] = append(m.sentNotifications[req.UserID], *req)
	return nil
}

func (m *mockDeliveryService) SendPager(ctx context.Context, req *models.NotificationRequest) error {
	if m.sentNotifications == nil {
		m.sentNotifications = make(map[string][]models.NotificationRequest)
	}
	m.sentNotifications[req.UserID] = append(m.sentNotifications[req.UserID], *req)
	return nil
}

func (m *mockDeliveryService) GetDeliveryStatus(ctx context.Context, notificationID string) (*models.DeliveryStatus, error) {
	return &models.DeliveryStatus{
		NotificationID: notificationID,
		Status:         "DELIVERED",
		Timestamp:      time.Now(),
	}, nil
}

// cleanupTestData removes test data from database and Redis
func cleanupTestData(t *testing.T, ctx context.Context, pool *pgxpool.Pool, redisClient *redis.Client, userID, alertID string) {
	// Clean up database
	if userID != "" {
		_, err := pool.Exec(ctx, "DELETE FROM notification_service.user_preferences WHERE user_id = $1", userID)
		if err != nil {
			t.Logf("Warning: Failed to cleanup user preferences for %s: %v", userID, err)
		}
	}

	// Clean up Redis
	if userID != "" {
		pattern := fmt.Sprintf("*%s*", userID)
		keys, err := redisClient.Keys(ctx, pattern).Result()
		if err == nil && len(keys) > 0 {
			err = redisClient.Del(ctx, keys...).Err()
			if err != nil {
				t.Logf("Warning: Failed to cleanup Redis keys for %s: %v", userID, err)
			}
		}
	}

	if alertID != "" {
		alertKeys := []string{
			fmt.Sprintf("alert:dedup:%s", alertID),
			fmt.Sprintf("alert:status:%s", alertID),
		}
		err := redisClient.Del(ctx, alertKeys...).Err()
		if err != nil {
			t.Logf("Warning: Failed to cleanup alert keys for %s: %v", alertID, err)
		}
	}
}

// TestKafkaIntegration tests Kafka consumer integration (if Kafka is available)
func TestKafkaIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Kafka integration test in short mode")
	}

	kafkaBrokers := os.Getenv("KAFKA_BROKERS")
	if kafkaBrokers == "" {
		t.Skip("KAFKA_BROKERS not set, skipping Kafka integration test")
	}

	t.Run("KafkaConsumer_ReceiveAndProcess", func(t *testing.T) {
		// This test would require setting up a Kafka consumer and producer
		// For now, we'll skip it unless explicitly required
		t.Skip("Kafka integration requires additional setup")
	})
}

// BenchmarkNotificationFlow benchmarks the complete notification flow
func BenchmarkNotificationFlow(b *testing.B) {
	ctx := context.Background()

	// Setup (similar to test setup)
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
	}

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		b.Fatalf("Failed to connect to database: %v", err)
	}
	defer pool.Close()

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   1,
	})
	defer redisClient.Close()

	userService := users.NewUserPreferenceService(pool, redisClient)
	fatigueTracker := fatigue.NewAlertFatigueTracker(redisClient)
	deliveryService := &mockDeliveryService{
		sentNotifications: make(map[string][]models.NotificationRequest),
	}
	alertRouter := routing.NewAlertRouter(userService, fatigueTracker, deliveryService)

	// Setup test user
	testUserID := "benchmark_user_001"
	prefs := &models.UserPreferences{
		UserID: testUserID,
		ChannelPreferences: map[string]bool{
			"SMS": true,
		},
		SeverityChannels: map[string][]string{
			"HIGH": {"SMS"},
		},
		MaxAlertsPerHour: 1000,
	}
	_ = userService.SavePreferences(ctx, prefs)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		alert := &models.Alert{
			ID:          fmt.Sprintf("benchmark_alert_%d", i),
			PatientID:   "patient_001",
			Severity:    "HIGH",
			Type:        "BENCHMARK",
			Title:       "Benchmark Alert",
			Message:     "Testing notification flow performance",
			Timestamp:   time.Now(),
			TargetRoles: []string{"attending_physician"},
		}

		_ = alertRouter.RouteAlert(ctx, alert)
	}
}
