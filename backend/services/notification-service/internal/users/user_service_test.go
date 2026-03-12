package users

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/cardiofit/notification-service/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v3"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

// mockRedisClient is a mock implementation of Redis client for testing
type mockRedisClient struct {
	data      map[string]string
	shouldErr bool
}

func newMockRedisClient() *mockRedisClient {
	return &mockRedisClient{
		data: make(map[string]string),
	}
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	cmd := redis.NewStringCmd(ctx, "get", key)
	if m.shouldErr {
		cmd.SetErr(errors.New("redis error"))
		return cmd
	}
	if val, ok := m.data[key]; ok {
		cmd.SetVal(val)
	} else {
		cmd.SetErr(redis.Nil)
	}
	return cmd
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "set", key, value)
	if m.shouldErr {
		cmd.SetErr(errors.New("redis error"))
		return cmd
	}
	// Handle both string and byte slice values
	switch v := value.(type) {
	case string:
		m.data[key] = v
	case []byte:
		m.data[key] = string(v)
	default:
		m.data[key] = fmt.Sprintf("%v", v)
	}
	cmd.SetVal("OK")
	return cmd
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	cmd := redis.NewIntCmd(ctx, "del", keys)
	if m.shouldErr {
		cmd.SetErr(errors.New("redis error"))
		return cmd
	}
	deleted := int64(0)
	for _, key := range keys {
		if _, ok := m.data[key]; ok {
			delete(m.data, key)
			deleted++
		}
	}
	cmd.SetVal(deleted)
	return cmd
}

func (m *mockRedisClient) Ping(ctx context.Context) *redis.StatusCmd {
	cmd := redis.NewStatusCmd(ctx, "ping")
	if m.shouldErr {
		cmd.SetErr(errors.New("redis error"))
	} else {
		cmd.SetVal("PONG")
	}
	return cmd
}

func (m *mockRedisClient) Close() error {
	return nil
}

func setupTestService(t *testing.T) (*UserPreferenceService, pgxmock.PgxPoolIface, *mockRedisClient) {
	mockDB, err := pgxmock.NewPool()
	require.NoError(t, err)

	mockRedis := newMockRedisClient()
	logger := zap.NewNop()

	// Use the constructor which accepts interfaces
	service := NewUserPreferenceService(mockDB, mockRedis, logger)

	return service, mockDB, mockRedis
}

func TestGetAttendingPhysician_Success(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	departmentID := "dept_icu"

	// Mock database query
	rows := pgxmock.NewRows([]string{"user_id", "phone_number", "email", "pager_number", "fcm_token"}).
		AddRow("user_attending_001", "+1-555-0101", "dr.attending@cardiofit.com", "1234567", "fcm_token_001").
		AddRow("user_attending_002", "+1-555-0102", "dr.attending2@cardiofit.com", "1234568", "fcm_token_002")

	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)").
		WithArgs("%attending%").
		WillReturnRows(rows)

	// Mock preferences queries for each user (QueryRow expects userID argument)
	for i := 1; i <= 2; i++ {
		userID := fmt.Sprintf("user_attending_%03d", i)
		prefRows := pgxmock.NewRows([]string{
			"user_id", "channel_preferences", "severity_channels",
			"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
			"max_alerts_per_hour", "updated_at",
		}).AddRow(
			userID,
			[]byte(`{"SMS": true, "EMAIL": true, "PUSH": true}`),
			[]byte(`{"CRITICAL": ["PAGER", "SMS"], "HIGH": ["SMS", "PUSH"]}`),
			false,
			nil,
			nil,
			20,
			time.Now(),
		)
		mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
			WithArgs(userID).
			WillReturnRows(prefRows)
	}

	// Execute
	users, err := service.GetAttendingPhysician(ctx, departmentID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user_attending_001", users[0].ID)
	assert.Equal(t, string(RoleAttending), users[0].Role)
	assert.NotNil(t, users[0].Preferences)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetAttendingPhysician_CacheHit(t *testing.T) {
	service, mockDB, mockRedis := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	departmentID := "dept_icu"

	// Prepare cached users
	cachedUsers := []*models.User{
		{
			ID:           "user_attending_001",
			Email:        "dr.attending@cardiofit.com",
			Role:         string(RoleAttending),
			DepartmentID: departmentID,
		},
	}
	cachedData, _ := json.Marshal(cachedUsers)
	cacheKey := "users:attending:dept_icu"
	mockRedis.data[cacheKey] = string(cachedData)

	// Execute - should hit cache, no DB query
	users, err := service.GetAttendingPhysician(ctx, departmentID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, users, 1)
	assert.Equal(t, "user_attending_001", users[0].ID)
	assert.NoError(t, mockDB.ExpectationsWereMet()) // No DB queries should have been made
}

func TestGetChargeNurse_Success(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	departmentID := "dept_er"

	// Mock database query
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)LIMIT 1").
		WithArgs("%charge_nurse%").
		WillReturnRows(
			pgxmock.NewRows([]string{"user_id", "phone_number", "email", "pager_number", "fcm_token"}).
				AddRow("user_charge_nurse_001", "+1-555-0103", "charge.nurse@cardiofit.com", "1234569", "fcm_token_003"),
		)

	// Mock preferences query (QueryRow expects userID argument)
	prefRows := pgxmock.NewRows([]string{
		"user_id", "channel_preferences", "severity_channels",
		"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
		"max_alerts_per_hour", "updated_at",
	}).AddRow(
		"user_charge_nurse_001",
		[]byte(`{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true}`),
		[]byte(`{"CRITICAL": ["PAGER", "SMS"], "HIGH": ["SMS", "PUSH"]}`),
		false,
		nil,
		nil,
		25,
		time.Now(),
	)
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
		WithArgs("user_charge_nurse_001").
		WillReturnRows(prefRows)

	// Execute
	user, err := service.GetChargeNurse(ctx, departmentID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user_charge_nurse_001", user.ID)
	assert.Equal(t, string(RoleChargeNurse), user.Role)
	assert.Equal(t, departmentID, user.DepartmentID)
	assert.NotNil(t, user.Preferences)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetChargeNurse_NotFound(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	departmentID := "dept_unknown"

	// Mock database query returning no rows
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)LIMIT 1").
		WithArgs("%charge_nurse%").
		WillReturnError(pgx.ErrNoRows)

	// Execute
	user, err := service.GetChargeNurse(ctx, departmentID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, user)
	assert.Contains(t, err.Error(), "no charge nurse found")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetPrimaryNurse_Success(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	patientID := "PAT-12345"

	// Mock database query
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)LIMIT 1").
		WithArgs("%primary_nurse%").
		WillReturnRows(
			pgxmock.NewRows([]string{"user_id", "phone_number", "email", "pager_number", "fcm_token"}).
				AddRow("user_primary_nurse_001", "+1-555-0104", "primary.nurse@cardiofit.com", "", "fcm_token_004"),
		)

	// Mock preferences query with pointer values for quiet hours (QueryRow expects userID argument)
	qhStart := 22
	qhEnd := 6
	prefRows := pgxmock.NewRows([]string{
		"user_id", "channel_preferences", "severity_channels",
		"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
		"max_alerts_per_hour", "updated_at",
	}).AddRow(
		"user_primary_nurse_001",
		[]byte(`{"SMS": true, "EMAIL": true, "PUSH": true}`),
		[]byte(`{"CRITICAL": ["SMS", "PUSH"], "HIGH": ["SMS", "PUSH"]}`),
		true,
		&qhStart,
		&qhEnd,
		20,
		time.Now(),
	)
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
		WithArgs("user_primary_nurse_001").
		WillReturnRows(prefRows)

	// Execute
	user, err := service.GetPrimaryNurse(ctx, patientID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, user)
	assert.Equal(t, "user_primary_nurse_001", user.ID)
	assert.Equal(t, string(RolePrimaryNurse), user.Role)
	assert.NotNil(t, user.Preferences)
	assert.True(t, user.Preferences.QuietHoursEnabled)
	assert.Equal(t, 22, user.Preferences.QuietHoursStart)
	assert.Equal(t, 6, user.Preferences.QuietHoursEnd)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetResident_Success(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	departmentID := "dept_cardio"

	// Mock database query
	rows := pgxmock.NewRows([]string{"user_id", "phone_number", "email", "pager_number", "fcm_token"}).
		AddRow("user_resident_001", "+1-555-0105", "resident1@cardiofit.com", "1234570", "fcm_token_005").
		AddRow("user_resident_002", "+1-555-0106", "resident2@cardiofit.com", "1234571", "fcm_token_006")

	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)").
		WithArgs("%resident%").
		WillReturnRows(rows)

	// Mock preferences queries for each resident (QueryRow expects userID argument)
	for i := 1; i <= 2; i++ {
		userID := fmt.Sprintf("user_resident_%03d", i)
		prefRows := pgxmock.NewRows([]string{
			"user_id", "channel_preferences", "severity_channels",
			"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
			"max_alerts_per_hour", "updated_at",
		}).AddRow(
			userID,
			[]byte(`{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": true}`),
			[]byte(`{"CRITICAL": ["PAGER", "SMS"], "HIGH": ["SMS", "PUSH"]}`),
			false,
			nil,
			nil,
			20,
			time.Now(),
		)
		mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
			WithArgs(userID).
			WillReturnRows(prefRows)
	}

	// Execute
	users, err := service.GetResident(ctx, departmentID)

	// Assert
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user_resident_001", users[0].ID)
	assert.Equal(t, string(RoleResident), users[0].Role)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetClinicalInformaticsTeam_Success(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()

	// Mock database query
	rows := pgxmock.NewRows([]string{"user_id", "phone_number", "email", "pager_number", "fcm_token"}).
		AddRow("user_clinical_informatics_001", "", "informatics1@cardiofit.com", "", "fcm_token_007").
		AddRow("user_clinical_informatics_002", "", "informatics2@cardiofit.com", "", "fcm_token_008")

	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)").
		WithArgs("%informatics%").
		WillReturnRows(rows)

	// Mock preferences queries (QueryRow expects userID argument)
	for i := 1; i <= 2; i++ {
		userID := fmt.Sprintf("user_clinical_informatics_%03d", i)
		qhStart := 20
		qhEnd := 8
		prefRows := pgxmock.NewRows([]string{
			"user_id", "channel_preferences", "severity_channels",
			"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
			"max_alerts_per_hour", "updated_at",
		}).AddRow(
			userID,
			[]byte(`{"SMS": false, "EMAIL": true, "PUSH": true}`),
			[]byte(`{"CRITICAL": ["EMAIL", "PUSH"], "HIGH": ["EMAIL", "PUSH"]}`),
			true,
			&qhStart,
			&qhEnd,
			50,
			time.Now(),
		)
		mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
			WithArgs(userID).
			WillReturnRows(prefRows)
	}

	// Execute
	users, err := service.GetClinicalInformaticsTeam(ctx)

	// Assert
	require.NoError(t, err)
	assert.Len(t, users, 2)
	assert.Equal(t, "user_clinical_informatics_001", users[0].ID)
	assert.Equal(t, string(RoleInformatics), users[0].Role)
	assert.NotNil(t, users[0].Preferences)
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetPreferredChannels_UserConfigured(t *testing.T) {
	service, _, _ := setupTestService(t)

	ctx := context.Background()
	user := &models.User{
		ID:   "user_test_001",
		Role: string(RoleAttending),
		Preferences: &models.UserPreferences{
			UserID: "user_test_001",
			ChannelPreferences: map[models.NotificationChannel]bool{
				models.ChannelSMS:   true,
				models.ChannelPager: true,
				models.ChannelEmail: false,
			},
			SeverityChannels: map[models.AlertSeverity][]models.NotificationChannel{
				models.SeverityCritical: {models.ChannelPager, models.ChannelSMS, models.ChannelEmail},
			},
		},
	}

	// Execute
	channels, err := service.GetPreferredChannels(ctx, user, models.SeverityCritical)

	// Assert
	require.NoError(t, err)
	assert.Len(t, channels, 2) // Email should be filtered out (disabled)
	assert.Contains(t, channels, models.ChannelPager)
	assert.Contains(t, channels, models.ChannelSMS)
	assert.NotContains(t, channels, models.ChannelEmail)
}

func TestGetPreferredChannels_DefaultFallback(t *testing.T) {
	service, _, _ := setupTestService(t)

	ctx := context.Background()
	user := &models.User{
		ID:   "user_test_002",
		Role: string(RoleResident),
		Preferences: &models.UserPreferences{
			UserID: "user_test_002",
			ChannelPreferences: map[models.NotificationChannel]bool{
				models.ChannelSMS:  true,
				models.ChannelPush: true,
			},
			SeverityChannels: map[models.AlertSeverity][]models.NotificationChannel{
				// No CRITICAL configuration
			},
		},
	}

	// Execute
	channels, err := service.GetPreferredChannels(ctx, user, models.SeverityCritical)

	// Assert
	require.NoError(t, err)
	// Should return default channels for CRITICAL
	expectedDefaults := models.DefaultSeverityChannels[models.SeverityCritical]
	assert.ElementsMatch(t, expectedDefaults, channels)
}

func TestGetPreferredChannels_NilUser(t *testing.T) {
	service, _, _ := setupTestService(t)

	ctx := context.Background()

	// Execute
	channels, err := service.GetPreferredChannels(ctx, nil, models.SeverityCritical)

	// Assert
	require.Error(t, err)
	assert.Nil(t, channels)
	assert.Contains(t, err.Error(), "user cannot be nil")
}

func TestGetPreferences_DatabaseQuery(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	userID := "user_test_001"

	// Mock database query with pointer values for nullable fields
	qhStart := 22
	qhEnd := 7
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
		WithArgs(userID).
		WillReturnRows(
			pgxmock.NewRows([]string{
				"user_id", "channel_preferences", "severity_channels",
				"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
				"max_alerts_per_hour", "updated_at",
			}).AddRow(
				userID,
				[]byte(`{"SMS": true, "EMAIL": true, "PUSH": true, "PAGER": false}`),
				[]byte(`{"CRITICAL": ["SMS", "PUSH"], "HIGH": ["SMS", "PUSH"], "MODERATE": ["PUSH"]}`),
				true,
				&qhStart,
				&qhEnd,
				15,
				time.Now(),
			),
		)

	// Execute
	prefs, err := service.GetPreferences(ctx, userID)

	// Assert
	require.NoError(t, err)
	assert.NotNil(t, prefs)
	assert.Equal(t, userID, prefs.UserID)
	assert.True(t, prefs.QuietHoursEnabled)
	assert.Equal(t, 22, prefs.QuietHoursStart)
	assert.Equal(t, 7, prefs.QuietHoursEnd)
	assert.Equal(t, 15, prefs.MaxAlertsPerHour)
	assert.True(t, prefs.ChannelPreferences[models.ChannelSMS])
	assert.False(t, prefs.ChannelPreferences[models.ChannelPager])
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestGetPreferences_NotFound(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	userID := "user_nonexistent"

	// Mock database query returning no rows
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
		WithArgs(userID).
		WillReturnError(pgx.ErrNoRows)

	// Execute
	prefs, err := service.GetPreferences(ctx, userID)

	// Assert
	require.Error(t, err)
	assert.Nil(t, prefs)
	assert.Contains(t, err.Error(), "no preferences found")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestUpdatePreferences_Success(t *testing.T) {
	service, mockDB, mockRedis := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	userID := "user_test_001"
	prefs := &models.UserPreferences{
		UserID: userID,
		ChannelPreferences: map[models.NotificationChannel]bool{
			models.ChannelSMS:   true,
			models.ChannelEmail: true,
			models.ChannelPush:  false,
		},
		SeverityChannels: map[models.AlertSeverity][]models.NotificationChannel{
			models.SeverityCritical: {models.ChannelSMS, models.ChannelPager},
			models.SeverityHigh:     {models.ChannelSMS},
		},
		QuietHoursEnabled: true,
		QuietHoursStart:   23,
		QuietHoursEnd:     7,
		MaxAlertsPerHour:  10,
	}

	// Mock database update
	mockDB.ExpectExec("UPDATE notification_service.user_preferences(.+)").
		WithArgs(
			userID,
			pgxmock.AnyArg(), // channel_preferences JSON
			pgxmock.AnyArg(), // severity_channels JSON
			true,
			23,
			7,
			10,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	// Pre-populate cache to test invalidation
	cacheKey := "prefs:user_test_001"
	mockRedis.data[cacheKey] = "old_data"

	// Execute
	err := service.UpdatePreferences(ctx, userID, prefs)

	// Assert
	require.NoError(t, err)
	assert.NoError(t, mockDB.ExpectationsWereMet())
	// Cache should be invalidated
	_, exists := mockRedis.data[cacheKey]
	assert.False(t, exists)
}

func TestUpdatePreferences_NotFound(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	userID := "user_nonexistent"
	prefs := &models.UserPreferences{
		UserID:            userID,
		MaxAlertsPerHour:  10,
		QuietHoursEnabled: false,
	}

	// Mock database update returning 0 rows affected
	mockDB.ExpectExec("UPDATE notification_service.user_preferences(.+)").
		WithArgs(
			userID,
			pgxmock.AnyArg(),
			pgxmock.AnyArg(),
			false,
			nil,
			nil,
			10,
		).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	// Execute
	err := service.UpdatePreferences(ctx, userID, prefs)

	// Assert
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no preferences found to update")
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestCacheOperations(t *testing.T) {
	service, _, mockRedis := setupTestService(t)

	ctx := context.Background()

	t.Run("setCached and getCached", func(t *testing.T) {
		key := "test:key"
		value := &models.User{
			ID:    "user_001",
			Email: "test@example.com",
			Role:  "ATTENDING",
		}

		// Set cache
		err := service.setCached(ctx, key, value)
		require.NoError(t, err)

		// Get cache
		var retrieved *models.User
		found, err := service.getCached(ctx, key, &retrieved)
		require.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, value.ID, retrieved.ID)
		assert.Equal(t, value.Email, retrieved.Email)
	})

	t.Run("getCached miss", func(t *testing.T) {
		var retrieved *models.User
		found, err := service.getCached(ctx, "nonexistent:key", &retrieved)
		require.NoError(t, err)
		assert.False(t, found)
	})

	t.Run("cache error handling", func(t *testing.T) {
		mockRedis.shouldErr = true
		defer func() { mockRedis.shouldErr = false }()

		var retrieved *models.User
		found, err := service.getCached(ctx, "test:key", &retrieved)
		require.Error(t, err)
		assert.False(t, found)
	})
}

func TestInvalidateUserCache(t *testing.T) {
	service, _, mockRedis := setupTestService(t)

	ctx := context.Background()
	userID := "user_test_001"

	// Populate cache
	mockRedis.data["prefs:user_test_001"] = "some_data"

	// Execute
	err := service.InvalidateUserCache(ctx, userID)

	// Assert
	require.NoError(t, err)
	_, exists := mockRedis.data["prefs:user_test_001"]
	assert.False(t, exists)
}

func TestInvalidateDepartmentCache(t *testing.T) {
	service, _, mockRedis := setupTestService(t)

	ctx := context.Background()
	departmentID := "dept_icu"

	// Populate cache
	mockRedis.data["users:attending:dept_icu"] = "attending_data"
	mockRedis.data["users:charge_nurse:dept_icu"] = "charge_nurse_data"
	mockRedis.data["users:resident:dept_icu"] = "resident_data"

	// Execute
	err := service.InvalidateDepartmentCache(ctx, departmentID)

	// Assert
	require.NoError(t, err)
	assert.Empty(t, mockRedis.data)
}

func TestConcurrentAccess(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()
	departmentID := "dept_concurrent"

	// Mock first query (cache miss), subsequent will hit cache
	rows := pgxmock.NewRows([]string{"user_id", "phone_number", "email", "pager_number", "fcm_token"}).
		AddRow("user_attending_001", "+1-555-0101", "dr.attending@cardiofit.com", "1234567", "fcm_token_001")

	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)").
		WithArgs("%attending%").
		WillReturnRows(rows)

	// Mock preferences query (QueryRow expects userID argument)
	prefRows := pgxmock.NewRows([]string{
		"user_id", "channel_preferences", "severity_channels",
		"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
		"max_alerts_per_hour", "updated_at",
	}).AddRow(
		"user_attending_001",
		[]byte(`{"SMS": true, "EMAIL": true}`),
		[]byte(`{"CRITICAL": ["SMS"]}`),
		false,
		nil,
		nil,
		20,
		time.Now(),
	)
	mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
		WithArgs("user_attending_001").
		WillReturnRows(prefRows)

	// Execute concurrent requests
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, err := service.GetAttendingPhysician(ctx, departmentID)
			assert.NoError(t, err)
			done <- true
		}()
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// After first request, subsequent should hit cache
	assert.NoError(t, mockDB.ExpectationsWereMet())
}

func TestEdgeCases(t *testing.T) {
	service, mockDB, _ := setupTestService(t)
	defer mockDB.Close()

	ctx := context.Background()

	t.Run("empty department returns empty list", func(t *testing.T) {
		mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)").
			WithArgs("%attending%").
			WillReturnRows(pgxmock.NewRows([]string{"user_id", "phone_number", "email", "pager_number", "fcm_token"}))

		users, err := service.GetAttendingPhysician(ctx, "dept_empty")
		require.NoError(t, err)
		assert.Empty(t, users)
	})

	t.Run("nil preferences in update", func(t *testing.T) {
		err := service.UpdatePreferences(ctx, "user_001", nil)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "preferences cannot be nil")
	})

	t.Run("preferences with no quiet hours", func(t *testing.T) {
		userID := "user_no_quiet_hours"
		mockDB.ExpectQuery("SELECT(.+)FROM notification_service.user_preferences(.+)WHERE user_id(.+)").
			WithArgs(userID).
			WillReturnRows(
				pgxmock.NewRows([]string{
					"user_id", "channel_preferences", "severity_channels",
					"quiet_hours_enabled", "quiet_hours_start", "quiet_hours_end",
					"max_alerts_per_hour", "updated_at",
				}).AddRow(
					userID,
					[]byte(`{"SMS": true}`),
					[]byte(`{"CRITICAL": ["SMS"]}`),
					false,
					nil,
					nil,
					20,
					time.Now(),
				),
			)

		prefs, err := service.GetPreferences(ctx, userID)
		require.NoError(t, err)
		assert.False(t, prefs.QuietHoursEnabled)
		assert.Equal(t, 0, prefs.QuietHoursStart)
		assert.Equal(t, 0, prefs.QuietHoursEnd)
	})
}

func BenchmarkGetAttendingPhysician(b *testing.B) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	mockRedis := newMockRedisClient()
	logger := zap.NewNop()
	service := NewUserPreferenceService(mockDB, mockRedis, logger)

	ctx := context.Background()
	departmentID := "dept_bench"

	// Pre-populate cache
	cachedUsers := []*models.User{
		{ID: "user_bench_001", Role: string(RoleAttending)},
	}
	cachedData, _ := json.Marshal(cachedUsers)
	cacheKey := "users:attending:dept_bench"
	mockRedis.data[cacheKey] = string(cachedData)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetAttendingPhysician(ctx, departmentID)
	}
}

func BenchmarkGetPreferredChannels(b *testing.B) {
	mockDB, err := pgxmock.NewPool()
	if err != nil {
		b.Fatal(err)
	}
	defer mockDB.Close()

	mockRedis := newMockRedisClient()
	logger := zap.NewNop()
	service := NewUserPreferenceService(mockDB, mockRedis, logger)

	ctx := context.Background()
	user := &models.User{
		ID:   "user_bench_001",
		Role: string(RoleAttending),
		Preferences: &models.UserPreferences{
			UserID: "user_bench_001",
			ChannelPreferences: map[models.NotificationChannel]bool{
				models.ChannelSMS:   true,
				models.ChannelPager: true,
			},
			SeverityChannels: map[models.AlertSeverity][]models.NotificationChannel{
				models.SeverityCritical: {models.ChannelPager, models.ChannelSMS},
			},
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetPreferredChannels(ctx, user, models.SeverityCritical)
	}
}
