package fatigue

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/cardiofit/notification-service/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
)

// MockRedisClient wraps redis client for testing
type MockRedisClient struct {
	client *redis.Client
}

func setupTestRedis(t *testing.T) *redis.Client {
	// Connect to test Redis instance
	client := redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "",
		DB:       1, // Use DB 1 for tests
	})

	ctx := context.Background()
	if err := client.Ping(ctx).Err(); err != nil {
		t.Skipf("Redis not available for testing: %v", err)
	}

	// Clean test database before tests
	client.FlushDB(ctx)

	return client
}

func setupTestTracker(t *testing.T) (*AlertFatigueTracker, *redis.Client) {
	logger := zaptest.NewLogger(t)
	redisClient := setupTestRedis(t)

	tracker := &AlertFatigueTracker{
		redisClient: redisClient,
		db:          nil, // DB tests require PostgreSQL setup
		logger:      logger,
		config: FatigueConfig{
			MaxAlertsPerHour:  20,
			DuplicateWindowMs: int64(5 * time.Minute / time.Millisecond),
			BundleThreshold:   3,
			QuietHoursStart:   time.Date(0, 1, 1, 22, 0, 0, 0, time.UTC),
			QuietHoursEnd:     time.Date(0, 1, 1, 7, 0, 0, 0, time.UTC),
		},
	}

	return tracker, redisClient
}

func createTestAlert(alertType models.AlertType, severity models.AlertSeverity) *models.Alert {
	return &models.Alert{
		AlertID:      fmt.Sprintf("alert-%d", time.Now().UnixNano()),
		PatientID:    "patient-123",
		HospitalID:   "hospital-1",
		DepartmentID: "dept-icu",
		AlertType:    alertType,
		Severity:     severity,
		Confidence:   0.95,
		Message:      "Test alert message",
		Timestamp:    time.Now().UnixMilli(),
	}
}

func createTestUser(userID string) *models.User {
	return &models.User{
		ID:           userID,
		Name:         "Test User",
		Email:        "test@example.com",
		PhoneNumber:  "+1234567890",
		Role:         "ATTENDING",
		DepartmentID: "dept-icu",
		Preferences: &models.UserPreferences{
			UserID:            userID,
			QuietHoursEnabled: false,
			QuietHoursStart:   22,
			QuietHoursEnd:     7,
			MaxAlertsPerHour:  20,
		},
	}
}

// Test CRITICAL bypass
func TestCriticalAlertBypass(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-1")
	alert := createTestAlert(models.AlertTypeSepsis, models.SeverityCritical)

	result, err := tracker.ShouldSuppress(ctx, alert, user)
	require.NoError(t, err)
	assert.False(t, result.ShouldSuppress, "CRITICAL alerts should never be suppressed")
	assert.Equal(t, "CRITICAL_BYPASS", result.Reason)
}

// Test rate limiting
func TestRateLimiting(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-2")

	// Send max alerts (20) with different patients to avoid duplicate detection
	for i := 0; i < 20; i++ {
		alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
		alert.PatientID = fmt.Sprintf("patient-%d", i) // Different patient each time

		result, err := tracker.ShouldSuppress(ctx, alert, user)
		require.NoError(t, err)
		assert.False(t, result.ShouldSuppress, "Should allow up to 20 alerts")

		// Record notification
		err = tracker.RecordNotification(ctx, user.ID, alert)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(1 * time.Millisecond)
	}

	// 21st alert should be rate limited
	alert21 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	alert21.PatientID = "patient-21"
	result, err := tracker.ShouldSuppress(ctx, alert21, user)
	require.NoError(t, err)
	assert.True(t, result.ShouldSuppress, "Should suppress alert after rate limit")
	assert.Equal(t, reasonRateLimit, result.Reason)
	assert.Equal(t, 20, result.AlertCount)
}

// Test duplicate detection
func TestDuplicateDetection(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-3")

	// Send first alert
	alert1 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	result, err := tracker.ShouldSuppress(ctx, alert1, user)
	require.NoError(t, err)
	assert.False(t, result.ShouldSuppress)

	// Record notification
	err = tracker.RecordNotification(ctx, user.ID, alert1)
	require.NoError(t, err)

	// Send duplicate alert (same type, patient, severity)
	alert2 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	alert2.PatientID = alert1.PatientID // Same patient

	result, err = tracker.ShouldSuppress(ctx, alert2, user)
	require.NoError(t, err)
	assert.True(t, result.ShouldSuppress, "Should suppress duplicate alert")
	assert.Equal(t, reasonDuplicate, result.Reason)
}

// Test duplicate with different patient should NOT suppress
func TestDuplicateDifferentPatient(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-4")

	// Send first alert for patient 1
	alert1 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	alert1.PatientID = "patient-1"
	result, err := tracker.ShouldSuppress(ctx, alert1, user)
	require.NoError(t, err)
	assert.False(t, result.ShouldSuppress)

	err = tracker.RecordNotification(ctx, user.ID, alert1)
	require.NoError(t, err)

	// Send same type alert for different patient
	alert2 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	alert2.PatientID = "patient-2"

	result, err = tracker.ShouldSuppress(ctx, alert2, user)
	require.NoError(t, err)
	assert.False(t, result.ShouldSuppress, "Different patient should not be duplicate")
}

// Test alert bundling
func TestAlertBundling(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-5")

	alertType := models.AlertTypeVitalSignAnomaly

	// Send 3 similar alerts within bundle window (different patients to avoid duplicate detection)
	for i := 0; i < 3; i++ {
		alert := createTestAlert(alertType, models.SeverityModerate)
		alert.PatientID = fmt.Sprintf("patient-bundle-%d", i) // Different patients
		result, err := tracker.ShouldSuppress(ctx, alert, user)
		require.NoError(t, err)

		// First 2 should not bundle yet
		if i < 2 {
			assert.False(t, result.ShouldSuppress)
			assert.NotEqual(t, reasonBundled, result.Reason)
		}

		err = tracker.RecordNotification(ctx, user.ID, alert)
		require.NoError(t, err)

		// Small delay to ensure different timestamps
		time.Sleep(10 * time.Millisecond)
	}

	// 4th alert should trigger bundling indication (different patient)
	alert4 := createTestAlert(alertType, models.SeverityModerate)
	alert4.PatientID = "patient-bundle-4"
	result, err := tracker.ShouldSuppress(ctx, alert4, user)
	require.NoError(t, err)

	// Bundling doesn't suppress, just indicates bundling opportunity
	assert.False(t, result.ShouldSuppress, "Bundling indicates grouping, not suppression")
	assert.Equal(t, reasonBundled, result.Reason)
	assert.GreaterOrEqual(t, len(result.BundledAlerts), 3, "Should have at least 3 bundled alerts")
}

// Test quiet hours
func TestQuietHours(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-6")
	user.Preferences.QuietHoursEnabled = true

	// Get current hour
	currentHour := time.Now().Hour()

	// Set quiet hours to include current hour
	user.Preferences.QuietHoursStart = currentHour
	user.Preferences.QuietHoursEnd = (currentHour + 2) % 24

	// Non-CRITICAL alert should be suppressed
	alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	result, err := tracker.ShouldSuppress(ctx, alert, user)
	require.NoError(t, err)
	assert.True(t, result.ShouldSuppress, "Should suppress during quiet hours")
	assert.Equal(t, reasonQuietHours, result.Reason)

	// CRITICAL alert should bypass quiet hours
	criticalAlert := createTestAlert(models.AlertTypeSepsis, models.SeverityCritical)
	result, err = tracker.ShouldSuppress(ctx, criticalAlert, user)
	require.NoError(t, err)
	assert.False(t, result.ShouldSuppress, "CRITICAL should bypass quiet hours")
}

// Test quiet hours spanning midnight
func TestQuietHoursSpanningMidnight(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-7")
	user.Preferences.QuietHoursEnabled = true
	user.Preferences.QuietHoursStart = 22 // 10 PM
	user.Preferences.QuietHoursEnd = 7    // 7 AM

	// Test at 1 AM (should be in quiet hours)
	currentHour := time.Now().Hour()

	// If current hour is between 22-23 or 0-6, should be quiet
	if (currentHour >= 22 && currentHour <= 23) || (currentHour >= 0 && currentHour < 7) {
		alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
		result, err := tracker.ShouldSuppress(ctx, alert, user)
		require.NoError(t, err)
		assert.True(t, result.ShouldSuppress, "Should suppress during midnight-spanning quiet hours")
	}
}

// Test quiet hours disabled
func TestQuietHoursDisabled(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-8")
	user.Preferences.QuietHoursEnabled = false
	user.Preferences.QuietHoursStart = time.Now().Hour()
	user.Preferences.QuietHoursEnd = (time.Now().Hour() + 1) % 24

	alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	result, err := tracker.ShouldSuppress(ctx, alert, user)
	require.NoError(t, err)
	assert.False(t, result.ShouldSuppress, "Should not suppress when quiet hours disabled")
}

// Test concurrent alerts
func TestConcurrentAlerts(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-9")

	// Send 10 alerts concurrently
	done := make(chan bool, 10)
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		go func(idx int) {
			alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
			result, err := tracker.ShouldSuppress(ctx, alert, user)
			if err != nil {
				errors <- err
				done <- false
				return
			}

			if !result.ShouldSuppress {
				err = tracker.RecordNotification(ctx, user.ID, alert)
				if err != nil {
					errors <- err
					done <- false
					return
				}
			}

			done <- true
		}(i)
	}

	// Wait for all goroutines
	successCount := 0
	for i := 0; i < 10; i++ {
		select {
		case success := <-done:
			if success {
				successCount++
			}
		case err := <-errors:
			t.Logf("Concurrent error: %v", err)
		case <-time.After(5 * time.Second):
			t.Fatal("Timeout waiting for concurrent operations")
		}
	}

	assert.Equal(t, 10, successCount, "All concurrent operations should succeed")
}

// Test rate limit expiration
func TestRateLimitExpiration(t *testing.T) {
	t.Skip("Skipping time-dependent test - requires time manipulation or real-time delays")

	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	// Set shorter window for testing
	tracker.config.MaxAlertsPerHour = 2

	ctx := context.Background()
	user := createTestUser("user-10")

	// Send 2 alerts with different patients
	for i := 0; i < 2; i++ {
		alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
		alert.PatientID = fmt.Sprintf("patient-expiry-%d", i)
		result, err := tracker.ShouldSuppress(ctx, alert, user)
		require.NoError(t, err)
		assert.False(t, result.ShouldSuppress)

		err = tracker.RecordNotification(ctx, user.ID, alert)
		require.NoError(t, err)
	}

	// 3rd should be rate limited
	alert3 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	alert3.PatientID = "patient-expiry-3"
	result, err := tracker.ShouldSuppress(ctx, alert3, user)
	require.NoError(t, err)
	assert.True(t, result.ShouldSuppress)

	// Wait for window to expire (1 hour in production, shortened for test)
	// Note: In real tests, you'd mock time or use shorter windows
	t.Log("Rate limit expiration test requires time manipulation - marked as informational")
}

// Test GetUserAlertStats
func TestGetUserAlertStats(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-11")

	// Send some alerts
	for i := 0; i < 5; i++ {
		alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
		err := tracker.RecordNotification(ctx, user.ID, alert)
		require.NoError(t, err)
	}

	// Get stats
	stats, err := tracker.GetUserAlertStats(ctx, user.ID)
	require.NoError(t, err)

	assert.Equal(t, int64(5), stats["alerts_in_last_hour"])
	assert.Equal(t, 20, stats["rate_limit_max"])
	assert.Equal(t, 15, stats["rate_limit_remaining"])
}

// Benchmark rate limit check
func BenchmarkRateLimitCheck(b *testing.B) {
	logger := zaptest.NewLogger(b)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	tracker := &AlertFatigueTracker{
		redisClient: redisClient,
		logger:      logger,
		config: FatigueConfig{
			MaxAlertsPerHour: 20,
		},
	}

	user := createTestUser("bench-user")

	// Prepare some data
	for i := 0; i < 10; i++ {
		alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
		tracker.RecordNotification(ctx, user.ID, alert)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = tracker.checkRateLimit(ctx, user.ID)
	}
}

// Benchmark duplicate check
func BenchmarkDuplicateCheck(b *testing.B) {
	logger := zaptest.NewLogger(b)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	tracker := &AlertFatigueTracker{
		redisClient: redisClient,
		logger:      logger,
		config: FatigueConfig{
			DuplicateWindowMs: int64(5 * time.Minute / time.Millisecond),
		},
	}

	user := createTestUser("bench-user-2")
	alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)

	// Record one alert
	tracker.RecordNotification(ctx, user.ID, alert)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _, _ = tracker.checkDuplicate(ctx, user.ID, alert)
	}
}

// Benchmark full suppression check
func BenchmarkShouldSuppress(b *testing.B) {
	logger := zaptest.NewLogger(b)
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
		DB:   1,
	})

	ctx := context.Background()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		b.Skipf("Redis not available: %v", err)
	}

	tracker := &AlertFatigueTracker{
		redisClient: redisClient,
		logger:      logger,
		config: FatigueConfig{
			MaxAlertsPerHour:  20,
			DuplicateWindowMs: int64(5 * time.Minute / time.Millisecond),
			BundleThreshold:   3,
		},
	}

	user := createTestUser("bench-user-3")
	alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = tracker.ShouldSuppress(ctx, alert, user)
	}
}

// Test edge case: nil user preferences
func TestNilUserPreferences(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-12")
	user.Preferences = nil // No preferences

	alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	result, err := tracker.ShouldSuppress(ctx, alert, user)
	require.NoError(t, err)
	assert.False(t, result.ShouldSuppress, "Should handle nil preferences gracefully")
}

// Test cleanup expired data
func TestCleanupExpiredData(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	ctx := context.Background()
	user := createTestUser("user-13")

	// Add some data
	for i := 0; i < 5; i++ {
		alert := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
		err := tracker.RecordNotification(ctx, user.ID, alert)
		require.NoError(t, err)
	}

	// Cleanup (without DB, only Redis cleanup will work)
	err := tracker.CleanupExpiredData(ctx)
	require.NoError(t, err)
}

// Test key generation consistency
func TestKeyGenerationConsistency(t *testing.T) {
	tracker, redisClient := setupTestTracker(t)
	defer redisClient.Close()

	user := createTestUser("user-14")
	alert1 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	alert2 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)

	// Same alert characteristics should generate same duplicate key
	key1 := tracker.getDuplicateKey(user.ID, alert1)
	key2 := tracker.getDuplicateKey(user.ID, alert2)

	assert.Equal(t, key1, key2, "Same alert characteristics should produce same key")

	// Different patient should generate different key
	alert3 := createTestAlert(models.AlertTypeSepsis, models.SeverityHigh)
	alert3.PatientID = "different-patient"
	key3 := tracker.getDuplicateKey(user.ID, alert3)

	assert.NotEqual(t, key1, key3, "Different patient should produce different key")
}

// Test time parsing
func TestParseTimeFromHHMM(t *testing.T) {
	tests := []struct {
		input    string
		wantErr  bool
		wantHour int
		wantMin  int
	}{
		{"22:00", false, 22, 0},
		{"07:30", false, 7, 30},
		{"00:00", false, 0, 0},
		{"23:59", false, 23, 59},
		{"24:00", true, 0, 0},  // Invalid hour
		{"12:60", true, 0, 0},  // Invalid minute
		{"12", true, 0, 0},     // Invalid format
		{"12:30:45", true, 0, 0}, // Invalid format
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseTimeFromHHMM(tt.input)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantHour, result.Hour())
				assert.Equal(t, tt.wantMin, result.Minute())
			}
		})
	}
}
