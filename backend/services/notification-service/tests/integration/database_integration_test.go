package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/cardiofit/notification-service/internal/models"
	"github.com/cardiofit/notification-service/internal/users"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDatabaseIntegration tests real PostgreSQL database operations
// This requires a running PostgreSQL instance with the notification_service schema
func TestDatabaseIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping database integration test in short mode")
	}

	// Get database URL from environment or use default
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://cardiofit_user:cardiofit_pass@localhost:5432/cardiofit_db?sslmode=disable"
	}

	// Connect to database
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, dbURL)
	require.NoError(t, err, "Failed to connect to database")
	defer pool.Close()

	// Verify connection
	err = pool.Ping(ctx)
	require.NoError(t, err, "Failed to ping database")

	t.Run("UserPreferences_CreateAndRetrieve", func(t *testing.T) {
		testUserID := fmt.Sprintf("test_user_%d", time.Now().Unix())

		// Insert test user preferences
		query := `
			INSERT INTO notification_service.user_preferences (
				user_id,
				channel_preferences,
				severity_channels,
				quiet_hours_enabled,
				quiet_hours_start,
				quiet_hours_end,
				max_alerts_per_hour
			) VALUES ($1, $2, $3, $4, $5, $6, $7)
		`

		channelPrefs := `{"SMS": true, "EMAIL": true, "PUSH": true}`
		severityChannels := `{"CRITICAL": ["PAGER", "SMS"], "HIGH": ["SMS", "PUSH"]}`
		quietHoursStart := 22
		quietHoursEnd := 6

		_, err := pool.Exec(ctx, query,
			testUserID,
			channelPrefs,
			severityChannels,
			true,
			quietHoursStart,
			quietHoursEnd,
			20,
		)
		require.NoError(t, err, "Failed to insert user preferences")

		// Retrieve user preferences
		var retrieved models.UserPreferences
		var channelPrefsJSON, severityChannelsJSON []byte
		var qhStart, qhEnd *int

		retrieveQuery := `
			SELECT
				user_id,
				channel_preferences,
				severity_channels,
				quiet_hours_enabled,
				quiet_hours_start,
				quiet_hours_end,
				max_alerts_per_hour,
				updated_at
			FROM notification_service.user_preferences
			WHERE user_id = $1
		`

		err = pool.QueryRow(ctx, retrieveQuery, testUserID).Scan(
			&retrieved.UserID,
			&channelPrefsJSON,
			&severityChannelsJSON,
			&retrieved.QuietHoursEnabled,
			&qhStart,
			&qhEnd,
			&retrieved.MaxAlertsPerHour,
			&retrieved.UpdatedAt,
		)
		require.NoError(t, err, "Failed to retrieve user preferences")

		// Verify data
		assert.Equal(t, testUserID, retrieved.UserID)
		assert.True(t, retrieved.QuietHoursEnabled)
		assert.NotNil(t, qhStart)
		assert.Equal(t, quietHoursStart, *qhStart)
		assert.NotNil(t, qhEnd)
		assert.Equal(t, quietHoursEnd, *qhEnd)
		assert.Equal(t, 20, retrieved.MaxAlertsPerHour)

		// Cleanup
		_, err = pool.Exec(ctx, "DELETE FROM notification_service.user_preferences WHERE user_id = $1", testUserID)
		assert.NoError(t, err, "Failed to cleanup test data")
	})

	t.Run("UserPreferences_UpdateOperation", func(t *testing.T) {
		testUserID := fmt.Sprintf("test_user_update_%d", time.Now().Unix())

		// Insert initial preferences
		insertQuery := `
			INSERT INTO notification_service.user_preferences (
				user_id,
				channel_preferences,
				severity_channels,
				quiet_hours_enabled,
				max_alerts_per_hour
			) VALUES ($1, $2, $3, $4, $5)
		`

		_, err := pool.Exec(ctx, insertQuery,
			testUserID,
			`{"SMS": true}`,
			`{"CRITICAL": ["SMS"]}`,
			false,
			10,
		)
		require.NoError(t, err, "Failed to insert initial preferences")

		// Update preferences
		updateQuery := `
			UPDATE notification_service.user_preferences
			SET max_alerts_per_hour = $1,
			    quiet_hours_enabled = $2,
			    updated_at = NOW()
			WHERE user_id = $3
		`

		_, err = pool.Exec(ctx, updateQuery, 30, true, testUserID)
		require.NoError(t, err, "Failed to update preferences")

		// Verify update
		var maxAlerts int
		var quietEnabled bool
		err = pool.QueryRow(ctx,
			"SELECT max_alerts_per_hour, quiet_hours_enabled FROM notification_service.user_preferences WHERE user_id = $1",
			testUserID,
		).Scan(&maxAlerts, &quietEnabled)
		require.NoError(t, err, "Failed to retrieve updated preferences")

		assert.Equal(t, 30, maxAlerts)
		assert.True(t, quietEnabled)

		// Cleanup
		_, err = pool.Exec(ctx, "DELETE FROM notification_service.user_preferences WHERE user_id = $1", testUserID)
		assert.NoError(t, err, "Failed to cleanup test data")
	})

	t.Run("UserPreferences_ConcurrentWrites", func(t *testing.T) {
		testUserID := fmt.Sprintf("test_user_concurrent_%d", time.Now().Unix())

		// Insert initial user
		insertQuery := `
			INSERT INTO notification_service.user_preferences (
				user_id,
				channel_preferences,
				severity_channels,
				max_alerts_per_hour
			) VALUES ($1, $2, $3, $4)
		`

		_, err := pool.Exec(ctx, insertQuery,
			testUserID,
			`{"SMS": true}`,
			`{"CRITICAL": ["SMS"]}`,
			10,
		)
		require.NoError(t, err, "Failed to insert initial user")

		// Simulate concurrent updates
		done := make(chan bool)
		errors := make(chan error, 10)

		for i := 0; i < 10; i++ {
			go func(iteration int) {
				updateQuery := `
					UPDATE notification_service.user_preferences
					SET max_alerts_per_hour = $1,
					    updated_at = NOW()
					WHERE user_id = $2
				`
				_, err := pool.Exec(ctx, updateQuery, 10+iteration, testUserID)
				if err != nil {
					errors <- err
				}
				done <- true
			}(i)
		}

		// Wait for all goroutines
		for i := 0; i < 10; i++ {
			<-done
		}
		close(errors)

		// Check for errors
		for err := range errors {
			t.Errorf("Concurrent update failed: %v", err)
		}

		// Verify final state (should be one of the values)
		var finalValue int
		err = pool.QueryRow(ctx,
			"SELECT max_alerts_per_hour FROM notification_service.user_preferences WHERE user_id = $1",
			testUserID,
		).Scan(&finalValue)
		require.NoError(t, err, "Failed to retrieve final value")

		assert.GreaterOrEqual(t, finalValue, 10)
		assert.LessOrEqual(t, finalValue, 19)

		// Cleanup
		_, err = pool.Exec(ctx, "DELETE FROM notification_service.user_preferences WHERE user_id = $1", testUserID)
		assert.NoError(t, err, "Failed to cleanup test data")
	})

	t.Run("UserService_DatabaseIntegration", func(t *testing.T) {
		// Create user service with real database
		cache := &mockCache{}
		userService := users.NewUserPreferenceService(pool, cache)

		testUserID := fmt.Sprintf("test_user_service_%d", time.Now().Unix())

		// Create preferences
		prefs := &models.UserPreferences{
			UserID: testUserID,
			ChannelPreferences: map[string]bool{
				"SMS":   true,
				"EMAIL": true,
				"PUSH":  true,
			},
			SeverityChannels: map[string][]string{
				"CRITICAL": {"PAGER", "SMS"},
				"HIGH":     {"SMS", "PUSH"},
			},
			QuietHoursEnabled: true,
			QuietHoursStart:   22,
			QuietHoursEnd:     6,
			MaxAlertsPerHour:  20,
		}

		// Save preferences
		err := userService.SavePreferences(ctx, prefs)
		require.NoError(t, err, "Failed to save preferences through service")

		// Retrieve preferences
		retrieved, err := userService.GetPreferences(ctx, testUserID)
		require.NoError(t, err, "Failed to retrieve preferences through service")
		require.NotNil(t, retrieved, "Retrieved preferences should not be nil")

		// Verify data
		assert.Equal(t, testUserID, retrieved.UserID)
		assert.True(t, retrieved.QuietHoursEnabled)
		assert.Equal(t, 22, retrieved.QuietHoursStart)
		assert.Equal(t, 6, retrieved.QuietHoursEnd)
		assert.Equal(t, 20, retrieved.MaxAlertsPerHour)
		assert.True(t, retrieved.ChannelPreferences["SMS"])
		assert.True(t, retrieved.ChannelPreferences["EMAIL"])

		// Cleanup
		_, err = pool.Exec(ctx, "DELETE FROM notification_service.user_preferences WHERE user_id = $1", testUserID)
		assert.NoError(t, err, "Failed to cleanup test data")
	})
}

// mockCache is a simple in-memory cache for testing
type mockCache struct {
	data map[string]interface{}
}

func (m *mockCache) Get(ctx context.Context, key string) (string, error) {
	if m.data == nil {
		return "", fmt.Errorf("key not found")
	}
	if val, ok := m.data[key]; ok {
		return val.(string), nil
	}
	return "", fmt.Errorf("key not found")
}

func (m *mockCache) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if m.data == nil {
		m.data = make(map[string]interface{})
	}
	m.data[key] = value
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	if m.data != nil {
		delete(m.data, key)
	}
	return nil
}
