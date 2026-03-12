package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cardiofit/notification-service/internal/fatigue"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRedisIntegration tests real Redis cache operations
// This requires a running Redis instance
func TestRedisIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping Redis integration test in short mode")
	}

	// Get Redis URL from environment or use default
	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	// Connect to Redis
	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"), // "" if no password
		DB:       1,                            // Use DB 1 for testing
	})
	defer client.Close()

	// Verify connection
	err := client.Ping(ctx).Err()
	require.NoError(t, err, "Failed to connect to Redis")

	t.Run("BasicCache_SetAndGet", func(t *testing.T) {
		testKey := fmt.Sprintf("test:cache:%d", time.Now().Unix())
		testValue := "test_value_123"

		// Set value
		err := client.Set(ctx, testKey, testValue, 10*time.Second).Err()
		require.NoError(t, err, "Failed to set value in Redis")

		// Get value
		retrieved, err := client.Get(ctx, testKey).Result()
		require.NoError(t, err, "Failed to get value from Redis")
		assert.Equal(t, testValue, retrieved)

		// Cleanup
		err = client.Del(ctx, testKey).Err()
		assert.NoError(t, err, "Failed to delete test key")
	})

	t.Run("BasicCache_Expiration", func(t *testing.T) {
		testKey := fmt.Sprintf("test:expiry:%d", time.Now().Unix())
		testValue := "expires_soon"

		// Set value with 2 second expiration
		err := client.Set(ctx, testKey, testValue, 2*time.Second).Err()
		require.NoError(t, err, "Failed to set value with expiration")

		// Value should exist immediately
		retrieved, err := client.Get(ctx, testKey).Result()
		require.NoError(t, err, "Failed to get value before expiration")
		assert.Equal(t, testValue, retrieved)

		// Wait for expiration
		time.Sleep(3 * time.Second)

		// Value should be gone
		_, err = client.Get(ctx, testKey).Result()
		assert.Error(t, err, "Value should have expired")
		assert.Equal(t, redis.Nil, err)
	})

	t.Run("AlertFatigue_CounterTracking", func(t *testing.T) {
		testUserID := fmt.Sprintf("test_user_%d", time.Now().Unix())
		counterKey := fmt.Sprintf("alert:counter:%s", testUserID)

		// Initialize counter
		err := client.Set(ctx, counterKey, 0, 1*time.Hour).Err()
		require.NoError(t, err, "Failed to initialize counter")

		// Increment counter multiple times
		for i := 1; i <= 5; i++ {
			count, err := client.Incr(ctx, counterKey).Result()
			require.NoError(t, err, "Failed to increment counter")
			assert.Equal(t, int64(i), count)
		}

		// Get final count
		finalCount, err := client.Get(ctx, counterKey).Int64()
		require.NoError(t, err, "Failed to get final count")
		assert.Equal(t, int64(5), finalCount)

		// Cleanup
		err = client.Del(ctx, counterKey).Err()
		assert.NoError(t, err, "Failed to delete counter")
	})

	t.Run("AlertFatigue_RecentAlerts", func(t *testing.T) {
		testUserID := fmt.Sprintf("test_user_alerts_%d", time.Now().Unix())
		listKey := fmt.Sprintf("alert:recent:%s", testUserID)

		// Add alerts to list
		alerts := []string{
			"alert_001",
			"alert_002",
			"alert_003",
			"alert_004",
			"alert_005",
		}

		for _, alertID := range alerts {
			err := client.LPush(ctx, listKey, alertID).Err()
			require.NoError(t, err, "Failed to push alert to list")
		}

		// Trim to keep only last 3
		err := client.LTrim(ctx, listKey, 0, 2).Err()
		require.NoError(t, err, "Failed to trim list")

		// Get all alerts
		retrieved, err := client.LRange(ctx, listKey, 0, -1).Result()
		require.NoError(t, err, "Failed to get alerts from list")
		assert.Len(t, retrieved, 3)
		assert.Equal(t, "alert_005", retrieved[0]) // Most recent
		assert.Equal(t, "alert_003", retrieved[2]) // Oldest kept

		// Cleanup
		err = client.Del(ctx, listKey).Err()
		assert.NoError(t, err, "Failed to delete list")
	})

	t.Run("AlertFatigueTracker_Integration", func(t *testing.T) {
		// Create alert fatigue tracker with real Redis
		tracker := fatigue.NewAlertFatigueTracker(client)

		testUserID := fmt.Sprintf("test_user_tracker_%d", time.Now().Unix())

		// Check initial state (should allow)
		shouldSend, err := tracker.ShouldSendAlert(ctx, testUserID, "CRITICAL")
		require.NoError(t, err, "Failed to check if should send alert")
		assert.True(t, shouldSend, "Should allow first alert")

		// Record multiple alerts
		for i := 1; i <= 3; i++ {
			alertID := fmt.Sprintf("alert_%d", i)
			err := tracker.RecordAlert(ctx, testUserID, "CRITICAL", alertID)
			require.NoError(t, err, "Failed to record alert")
		}

		// Check alert count
		count, err := tracker.GetAlertCount(ctx, testUserID, 1*time.Hour)
		require.NoError(t, err, "Failed to get alert count")
		assert.Equal(t, 3, count)

		// Get recent alerts
		recent, err := tracker.GetRecentAlerts(ctx, testUserID, 10)
		require.NoError(t, err, "Failed to get recent alerts")
		assert.Len(t, recent, 3)

		// Cleanup
		counterKey := fmt.Sprintf("alert:counter:%s:1h", testUserID)
		listKey := fmt.Sprintf("alert:recent:%s", testUserID)
		err = client.Del(ctx, counterKey, listKey).Err()
		assert.NoError(t, err, "Failed to cleanup tracker data")
	})

	t.Run("UserPreferences_Caching", func(t *testing.T) {
		testUserID := fmt.Sprintf("test_user_prefs_%d", time.Now().Unix())
		cacheKey := fmt.Sprintf("user:prefs:%s", testUserID)

		// Create preferences data
		prefsData := `{
			"user_id": "` + testUserID + `",
			"channel_preferences": {"SMS": true, "EMAIL": true},
			"severity_channels": {"CRITICAL": ["SMS", "PAGER"]},
			"quiet_hours_enabled": true,
			"quiet_hours_start": 22,
			"quiet_hours_end": 6,
			"max_alerts_per_hour": 20
		}`

		// Cache preferences
		err := client.Set(ctx, cacheKey, prefsData, 5*time.Minute).Err()
		require.NoError(t, err, "Failed to cache preferences")

		// Retrieve from cache
		cached, err := client.Get(ctx, cacheKey).Result()
		require.NoError(t, err, "Failed to get cached preferences")
		assert.Contains(t, cached, testUserID)
		assert.Contains(t, cached, "quiet_hours_enabled")

		// Verify TTL
		ttl, err := client.TTL(ctx, cacheKey).Result()
		require.NoError(t, err, "Failed to get TTL")
		assert.Greater(t, ttl.Seconds(), float64(0))
		assert.LessOrEqual(t, ttl.Seconds(), float64(300)) // 5 minutes

		// Cleanup
		err = client.Del(ctx, cacheKey).Err()
		assert.NoError(t, err, "Failed to delete cached preferences")
	})

	t.Run("ConcurrentAccess", func(t *testing.T) {
		testKey := fmt.Sprintf("test:concurrent:%d", time.Now().Unix())

		// Initialize counter
		err := client.Set(ctx, testKey, 0, 10*time.Second).Err()
		require.NoError(t, err, "Failed to initialize counter")

		// Simulate concurrent increments
		done := make(chan bool)
		errors := make(chan error, 50)

		for i := 0; i < 50; i++ {
			go func() {
				_, err := client.Incr(ctx, testKey).Result()
				if err != nil {
					errors <- err
				}
				done <- true
			}()
		}

		// Wait for all goroutines
		for i := 0; i < 50; i++ {
			<-done
		}
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent increment failed: %v", err)
		}

		// Verify final count
		finalCount, err := client.Get(ctx, testKey).Int64()
		require.NoError(t, err, "Failed to get final count")
		assert.Equal(t, int64(50), finalCount, "All increments should be counted")

		// Cleanup
		err = client.Del(ctx, testKey).Err()
		assert.NoError(t, err, "Failed to delete test key")
	})

	t.Run("AlertDeduplication", func(t *testing.T) {
		testAlertID := fmt.Sprintf("alert_dedup_%d", time.Now().Unix())
		dedupKey := fmt.Sprintf("alert:dedup:%s", testAlertID)

		// Check if alert was seen (should be false initially)
		exists, err := client.Exists(ctx, dedupKey).Result()
		require.NoError(t, err, "Failed to check existence")
		assert.Equal(t, int64(0), exists)

		// Mark alert as seen
		err = client.Set(ctx, dedupKey, "1", 5*time.Minute).Err()
		require.NoError(t, err, "Failed to mark alert as seen")

		// Check again (should be true now)
		exists, err = client.Exists(ctx, dedupKey).Result()
		require.NoError(t, err, "Failed to check existence after marking")
		assert.Equal(t, int64(1), exists)

		// Cleanup
		err = client.Del(ctx, dedupKey).Err()
		assert.NoError(t, err, "Failed to delete dedup key")
	})

	t.Run("SessionManagement", func(t *testing.T) {
		testSessionID := fmt.Sprintf("session_%d", time.Now().Unix())
		sessionKey := fmt.Sprintf("session:%s", testSessionID)

		// Create session data
		sessionData := map[string]interface{}{
			"user_id":    "user_123",
			"role":       "attending_physician",
			"department": "cardiology",
			"created_at": time.Now().Unix(),
		}

		// Store session
		err := client.HSet(ctx, sessionKey, sessionData).Err()
		require.NoError(t, err, "Failed to store session")

		// Set expiration
		err = client.Expire(ctx, sessionKey, 30*time.Minute).Err()
		require.NoError(t, err, "Failed to set session expiration")

		// Retrieve session
		retrieved, err := client.HGetAll(ctx, sessionKey).Result()
		require.NoError(t, err, "Failed to get session")
		assert.Equal(t, "user_123", retrieved["user_id"])
		assert.Equal(t, "attending_physician", retrieved["role"])
		assert.Equal(t, "cardiology", retrieved["department"])

		// Update specific field
		err = client.HSet(ctx, sessionKey, "last_activity", time.Now().Unix()).Err()
		require.NoError(t, err, "Failed to update session field")

		// Cleanup
		err = client.Del(ctx, sessionKey).Err()
		assert.NoError(t, err, "Failed to delete session")
	})
}

// TestAlertFatigueRateLimiting tests alert fatigue rate limiting logic
func TestAlertFatigueRateLimiting(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping rate limiting integration test in short mode")
	}

	redisAddr := os.Getenv("REDIS_ADDR")
	if redisAddr == "" {
		redisAddr = "localhost:6379"
	}

	ctx := context.Background()
	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
		DB:   1,
	})
	defer client.Close()

	err := client.Ping(ctx).Err()
	require.NoError(t, err, "Failed to connect to Redis")

	t.Run("RateLimit_EnforceMaxAlertsPerHour", func(t *testing.T) {
		tracker := fatigue.NewAlertFatigueTracker(client)
		testUserID := fmt.Sprintf("test_rate_limit_%d", time.Now().Unix())

		// Simulate max alerts per hour = 5
		maxAlerts := 5

		// Send alerts up to limit
		for i := 1; i <= maxAlerts; i++ {
			alertID := fmt.Sprintf("alert_%d", i)
			err := tracker.RecordAlert(ctx, testUserID, "HIGH", alertID)
			require.NoError(t, err, "Failed to record alert %d", i)

			shouldSend, err := tracker.ShouldSendAlert(ctx, testUserID, "HIGH")
			require.NoError(t, err, "Failed to check should send")

			if i < maxAlerts {
				assert.True(t, shouldSend, "Should allow alert %d", i)
			}
		}

		// Get count
		count, err := tracker.GetAlertCount(ctx, testUserID, 1*time.Hour)
		require.NoError(t, err, "Failed to get count")
		assert.Equal(t, maxAlerts, count)

		// Cleanup
		counterKey := fmt.Sprintf("alert:counter:%s:1h", testUserID)
		listKey := fmt.Sprintf("alert:recent:%s", testUserID)
		err = client.Del(ctx, counterKey, listKey).Err()
		assert.NoError(t, err, "Failed to cleanup")
	})

	t.Run("RateLimit_CriticalAlertsAlwaysPass", func(t *testing.T) {
		tracker := fatigue.NewAlertFatigueTracker(client)
		testUserID := fmt.Sprintf("test_critical_bypass_%d", time.Now().Unix())

		// Critical alerts should always be allowed
		for i := 1; i <= 100; i++ {
			shouldSend, err := tracker.ShouldSendAlert(ctx, testUserID, "CRITICAL")
			require.NoError(t, err, "Failed to check critical alert")
			assert.True(t, shouldSend, "Critical alert %d should always be allowed", i)

			alertID := fmt.Sprintf("critical_alert_%d", i)
			err = tracker.RecordAlert(ctx, testUserID, "CRITICAL", alertID)
			require.NoError(t, err, "Failed to record critical alert")
		}

		// Cleanup
		counterKey := fmt.Sprintf("alert:counter:%s:1h", testUserID)
		listKey := fmt.Sprintf("alert:recent:%s", testUserID)
		err = client.Del(ctx, counterKey, listKey).Err()
		assert.NoError(t, err, "Failed to cleanup")
	})
}
