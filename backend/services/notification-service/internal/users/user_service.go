package users

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/cardiofit/notification-service/internal/models"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// Cache TTLs
	defaultCacheTTL      = 15 * time.Minute
	preferencesCacheTTL  = 5 * time.Minute

	// Cache key prefixes
	cacheKeyAttendingPrefix    = "users:attending:"
	cacheKeyChargeNursePrefix  = "users:charge_nurse:"
	cacheKeyPrimaryNursePrefix = "users:primary_nurse:"
	cacheKeyResidentPrefix     = "users:resident:"
	cacheKeyInformaticsPrefix  = "users:informatics"
	cacheKeyPreferencesPrefix  = "prefs:"
)

// UserRole represents user roles in the system
type UserRole string

const (
	RoleAttending   UserRole = "ATTENDING"
	RoleResident    UserRole = "RESIDENT"
	RoleChargeNurse UserRole = "CHARGE_NURSE"
	RolePrimaryNurse UserRole = "PRIMARY_NURSE"
	RoleInformatics UserRole = "INFORMATICS"
)

// PgxPool is an interface for PostgreSQL connection pool operations
type PgxPool interface {
	Query(ctx context.Context, sql string, args ...interface{}) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...interface{}) pgx.Row
	Exec(ctx context.Context, sql string, args ...interface{}) (pgconn.CommandTag, error)
}

// RedisClient is an interface for Redis operations
type RedisClient interface {
	Get(ctx context.Context, key string) *redis.StringCmd
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
	Del(ctx context.Context, keys ...string) *redis.IntCmd
}

// UserPreferenceService manages user lookups and preferences with PostgreSQL and Redis caching
type UserPreferenceService struct {
	db          PgxPool
	redisClient RedisClient
	logger      *zap.Logger
	cacheTTL    time.Duration
}

// NewUserPreferenceService creates a new user preference service
func NewUserPreferenceService(db PgxPool, redisClient RedisClient, logger *zap.Logger) *UserPreferenceService {
	return &UserPreferenceService{
		db:          db,
		redisClient: redisClient,
		logger:      logger,
		cacheTTL:    defaultCacheTTL,
	}
}

// GetAttendingPhysician returns attending physicians for a department
func (s *UserPreferenceService) GetAttendingPhysician(ctx context.Context, departmentID string) ([]*models.User, error) {
	// Try cache first
	cacheKey := s.cacheKey(cacheKeyAttendingPrefix, departmentID)
	var users []*models.User
	found, err := s.getCached(ctx, cacheKey, &users)
	if err != nil {
		s.logger.Warn("Failed to get from cache", zap.Error(err), zap.String("key", cacheKey))
	}
	if found {
		s.logger.Debug("Cache hit for attending physicians", zap.String("department_id", departmentID))
		return users, nil
	}

	// Query database
	query := `
		SELECT
			up.user_id,
			up.phone_number,
			up.email,
			up.pager_number,
			up.fcm_token
		FROM notification_service.user_preferences up
		WHERE up.user_id LIKE $1
		ORDER BY up.updated_at DESC
	`

	// Pattern matching for attending physicians
	pattern := "%attending%"
	rows, err := s.db.Query(ctx, query, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query attending physicians: %w", err)
	}
	defer rows.Close()

	users = make([]*models.User, 0)
	for rows.Next() {
		user := &models.User{
			Role:         string(RoleAttending),
			DepartmentID: departmentID,
		}
		err := rows.Scan(&user.ID, &user.PhoneNumber, &user.Email, &user.PagerNumber, &user.FCMToken)
		if err != nil {
			return nil, fmt.Errorf("failed to scan attending physician: %w", err)
		}
		// Load preferences for the user
		prefs, err := s.getPreferencesFromDB(ctx, user.ID)
		if err != nil {
			s.logger.Warn("Failed to load preferences", zap.Error(err), zap.String("user_id", user.ID))
		} else {
			user.Preferences = prefs
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating attending physicians: %w", err)
	}

	// Cache the result
	if err := s.setCached(ctx, cacheKey, users); err != nil {
		s.logger.Warn("Failed to cache attending physicians", zap.Error(err))
	}

	s.logger.Info("Retrieved attending physicians",
		zap.String("department_id", departmentID),
		zap.Int("count", len(users)))

	return users, nil
}

// GetChargeNurse returns the charge nurse for a department
func (s *UserPreferenceService) GetChargeNurse(ctx context.Context, departmentID string) (*models.User, error) {
	// Try cache first
	cacheKey := s.cacheKey(cacheKeyChargeNursePrefix, departmentID)
	var user *models.User
	found, err := s.getCached(ctx, cacheKey, &user)
	if err != nil {
		s.logger.Warn("Failed to get from cache", zap.Error(err), zap.String("key", cacheKey))
	}
	if found && user != nil {
		s.logger.Debug("Cache hit for charge nurse", zap.String("department_id", departmentID))
		return user, nil
	}

	// Query database
	query := `
		SELECT
			up.user_id,
			up.phone_number,
			up.email,
			up.pager_number,
			up.fcm_token
		FROM notification_service.user_preferences up
		WHERE up.user_id LIKE $1
		ORDER BY up.updated_at DESC
		LIMIT 1
	`

	// Pattern matching for charge nurse
	pattern := "%charge_nurse%"
	user = &models.User{
		Role:         string(RoleChargeNurse),
		DepartmentID: departmentID,
	}

	err = s.db.QueryRow(ctx, query, pattern).Scan(
		&user.ID, &user.PhoneNumber, &user.Email, &user.PagerNumber, &user.FCMToken)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no charge nurse found for department %s", departmentID)
		}
		return nil, fmt.Errorf("failed to query charge nurse: %w", err)
	}

	// Load preferences
	prefs, err := s.getPreferencesFromDB(ctx, user.ID)
	if err != nil {
		s.logger.Warn("Failed to load preferences", zap.Error(err), zap.String("user_id", user.ID))
	} else {
		user.Preferences = prefs
	}

	// Cache the result
	if err := s.setCached(ctx, cacheKey, user); err != nil {
		s.logger.Warn("Failed to cache charge nurse", zap.Error(err))
	}

	s.logger.Info("Retrieved charge nurse",
		zap.String("department_id", departmentID),
		zap.String("user_id", user.ID))

	return user, nil
}

// GetPrimaryNurse returns the primary nurse assigned to a patient
func (s *UserPreferenceService) GetPrimaryNurse(ctx context.Context, patientID string) (*models.User, error) {
	// Try cache first
	cacheKey := s.cacheKey(cacheKeyPrimaryNursePrefix, patientID)
	var user *models.User
	found, err := s.getCached(ctx, cacheKey, &user)
	if err != nil {
		s.logger.Warn("Failed to get from cache", zap.Error(err), zap.String("key", cacheKey))
	}
	if found && user != nil {
		s.logger.Debug("Cache hit for primary nurse", zap.String("patient_id", patientID))
		return user, nil
	}

	// Query database - In production, this would join with a patient_assignments table
	// For now, we'll use the pattern matching approach with primary nurse
	query := `
		SELECT
			up.user_id,
			up.phone_number,
			up.email,
			up.pager_number,
			up.fcm_token
		FROM notification_service.user_preferences up
		WHERE up.user_id LIKE $1
		ORDER BY up.updated_at DESC
		LIMIT 1
	`

	// Pattern matching for primary nurse
	pattern := "%primary_nurse%"
	user = &models.User{
		Role: string(RolePrimaryNurse),
	}

	err = s.db.QueryRow(ctx, query, pattern).Scan(
		&user.ID, &user.PhoneNumber, &user.Email, &user.PagerNumber, &user.FCMToken)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no primary nurse found for patient %s", patientID)
		}
		return nil, fmt.Errorf("failed to query primary nurse: %w", err)
	}

	// Load preferences
	prefs, err := s.getPreferencesFromDB(ctx, user.ID)
	if err != nil {
		s.logger.Warn("Failed to load preferences", zap.Error(err), zap.String("user_id", user.ID))
	} else {
		user.Preferences = prefs
	}

	// Cache the result
	if err := s.setCached(ctx, cacheKey, user); err != nil {
		s.logger.Warn("Failed to cache primary nurse", zap.Error(err))
	}

	s.logger.Info("Retrieved primary nurse",
		zap.String("patient_id", patientID),
		zap.String("user_id", user.ID))

	return user, nil
}

// GetResident returns residents in a department
func (s *UserPreferenceService) GetResident(ctx context.Context, departmentID string) ([]*models.User, error) {
	// Try cache first
	cacheKey := s.cacheKey(cacheKeyResidentPrefix, departmentID)
	var users []*models.User
	found, err := s.getCached(ctx, cacheKey, &users)
	if err != nil {
		s.logger.Warn("Failed to get from cache", zap.Error(err), zap.String("key", cacheKey))
	}
	if found {
		s.logger.Debug("Cache hit for residents", zap.String("department_id", departmentID))
		return users, nil
	}

	// Query database
	query := `
		SELECT
			up.user_id,
			up.phone_number,
			up.email,
			up.pager_number,
			up.fcm_token
		FROM notification_service.user_preferences up
		WHERE up.user_id LIKE $1
		ORDER BY up.updated_at DESC
	`

	// Pattern matching for residents
	pattern := "%resident%"
	rows, err := s.db.Query(ctx, query, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query residents: %w", err)
	}
	defer rows.Close()

	users = make([]*models.User, 0)
	for rows.Next() {
		user := &models.User{
			Role:         string(RoleResident),
			DepartmentID: departmentID,
		}
		err := rows.Scan(&user.ID, &user.PhoneNumber, &user.Email, &user.PagerNumber, &user.FCMToken)
		if err != nil {
			return nil, fmt.Errorf("failed to scan resident: %w", err)
		}
		// Load preferences
		prefs, err := s.getPreferencesFromDB(ctx, user.ID)
		if err != nil {
			s.logger.Warn("Failed to load preferences", zap.Error(err), zap.String("user_id", user.ID))
		} else {
			user.Preferences = prefs
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating residents: %w", err)
	}

	// Cache the result
	if err := s.setCached(ctx, cacheKey, users); err != nil {
		s.logger.Warn("Failed to cache residents", zap.Error(err))
	}

	s.logger.Info("Retrieved residents",
		zap.String("department_id", departmentID),
		zap.Int("count", len(users)))

	return users, nil
}

// GetClinicalInformaticsTeam returns members of the clinical informatics team
func (s *UserPreferenceService) GetClinicalInformaticsTeam(ctx context.Context) ([]*models.User, error) {
	// Try cache first
	cacheKey := cacheKeyInformaticsPrefix
	var users []*models.User
	found, err := s.getCached(ctx, cacheKey, &users)
	if err != nil {
		s.logger.Warn("Failed to get from cache", zap.Error(err), zap.String("key", cacheKey))
	}
	if found {
		s.logger.Debug("Cache hit for clinical informatics team")
		return users, nil
	}

	// Query database
	query := `
		SELECT
			up.user_id,
			up.phone_number,
			up.email,
			up.pager_number,
			up.fcm_token
		FROM notification_service.user_preferences up
		WHERE up.user_id LIKE $1
		ORDER BY up.updated_at DESC
	`

	// Pattern matching for informatics team
	pattern := "%informatics%"
	rows, err := s.db.Query(ctx, query, pattern)
	if err != nil {
		return nil, fmt.Errorf("failed to query informatics team: %w", err)
	}
	defer rows.Close()

	users = make([]*models.User, 0)
	for rows.Next() {
		user := &models.User{
			Role: string(RoleInformatics),
		}
		err := rows.Scan(&user.ID, &user.PhoneNumber, &user.Email, &user.PagerNumber, &user.FCMToken)
		if err != nil {
			return nil, fmt.Errorf("failed to scan informatics team member: %w", err)
		}
		// Load preferences
		prefs, err := s.getPreferencesFromDB(ctx, user.ID)
		if err != nil {
			s.logger.Warn("Failed to load preferences", zap.Error(err), zap.String("user_id", user.ID))
		} else {
			user.Preferences = prefs
		}
		users = append(users, user)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating informatics team: %w", err)
	}

	// Cache the result
	if err := s.setCached(ctx, cacheKey, users); err != nil {
		s.logger.Warn("Failed to cache informatics team", zap.Error(err))
	}

	s.logger.Info("Retrieved clinical informatics team", zap.Int("count", len(users)))

	return users, nil
}

// GetPreferredChannels returns a user's preferred notification channels based on severity
func (s *UserPreferenceService) GetPreferredChannels(ctx context.Context, user *models.User, severity models.AlertSeverity) ([]models.NotificationChannel, error) {
	if user == nil {
		return nil, fmt.Errorf("user cannot be nil")
	}

	// Load preferences if not already loaded
	if user.Preferences == nil {
		prefs, err := s.GetPreferences(ctx, user.ID)
		if err != nil {
			s.logger.Warn("Failed to load user preferences, using defaults",
				zap.Error(err),
				zap.String("user_id", user.ID))
			// Return default channels for severity
			return models.DefaultSeverityChannels[severity], nil
		}
		user.Preferences = prefs
	}

	// Check if user has severity-specific channel preferences
	if channels, ok := user.Preferences.SeverityChannels[severity]; ok && len(channels) > 0 {
		// Filter out disabled channels
		enabledChannels := make([]models.NotificationChannel, 0)
		for _, channel := range channels {
			if enabled, ok := user.Preferences.ChannelPreferences[channel]; ok && enabled {
				enabledChannels = append(enabledChannels, channel)
			}
		}

		if len(enabledChannels) > 0 {
			s.logger.Debug("Using user-configured channels",
				zap.String("user_id", user.ID),
				zap.String("severity", string(severity)),
				zap.Int("channel_count", len(enabledChannels)))
			return enabledChannels, nil
		}
	}

	// Fallback to default channels
	defaultChannels := models.DefaultSeverityChannels[severity]
	s.logger.Debug("Using default channels",
		zap.String("user_id", user.ID),
		zap.String("severity", string(severity)),
		zap.Int("channel_count", len(defaultChannels)))

	return defaultChannels, nil
}

// GetPreferences retrieves user preferences with caching
func (s *UserPreferenceService) GetPreferences(ctx context.Context, userID string) (*models.UserPreferences, error) {
	// Try cache first
	cacheKey := s.cacheKey(cacheKeyPreferencesPrefix, userID)
	var prefs *models.UserPreferences
	found, err := s.getCached(ctx, cacheKey, &prefs)
	if err != nil {
		s.logger.Warn("Failed to get preferences from cache", zap.Error(err), zap.String("user_id", userID))
	}
	if found && prefs != nil {
		s.logger.Debug("Cache hit for user preferences", zap.String("user_id", userID))
		return prefs, nil
	}

	// Query from database
	prefs, err = s.getPreferencesFromDB(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Cache the result with shorter TTL
	if err := s.setCachedWithTTL(ctx, cacheKey, prefs, preferencesCacheTTL); err != nil {
		s.logger.Warn("Failed to cache user preferences", zap.Error(err))
	}

	return prefs, nil
}

// getPreferencesFromDB retrieves preferences from PostgreSQL
func (s *UserPreferenceService) getPreferencesFromDB(ctx context.Context, userID string) (*models.UserPreferences, error) {
	query := `
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

	var channelPrefsJSON, severityChannelsJSON []byte
	var quietHoursStart, quietHoursEnd *int
	prefs := &models.UserPreferences{
		UserID: userID,
	}

	err := s.db.QueryRow(ctx, query, userID).Scan(
		&prefs.UserID,
		&channelPrefsJSON,
		&severityChannelsJSON,
		&prefs.QuietHoursEnabled,
		&quietHoursStart,
		&quietHoursEnd,
		&prefs.MaxAlertsPerHour,
		&prefs.UpdatedAt,
	)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, fmt.Errorf("no preferences found for user %s", userID)
		}
		return nil, fmt.Errorf("failed to query user preferences: %w", err)
	}

	// Parse channel preferences JSON
	prefs.ChannelPreferences = make(map[models.NotificationChannel]bool)
	if err := json.Unmarshal(channelPrefsJSON, &prefs.ChannelPreferences); err != nil {
		return nil, fmt.Errorf("failed to unmarshal channel preferences: %w", err)
	}

	// Parse severity channels JSON
	prefs.SeverityChannels = make(map[models.AlertSeverity][]models.NotificationChannel)
	if err := json.Unmarshal(severityChannelsJSON, &prefs.SeverityChannels); err != nil {
		return nil, fmt.Errorf("failed to unmarshal severity channels: %w", err)
	}

	// Set quiet hours if present
	if quietHoursStart != nil {
		prefs.QuietHoursStart = *quietHoursStart
	}
	if quietHoursEnd != nil {
		prefs.QuietHoursEnd = *quietHoursEnd
	}

	return prefs, nil
}

// UpdatePreferences updates user notification preferences
func (s *UserPreferenceService) UpdatePreferences(ctx context.Context, userID string, prefs *models.UserPreferences) error {
	if prefs == nil {
		return fmt.Errorf("preferences cannot be nil")
	}

	// Marshal JSONB fields
	channelPrefsJSON, err := json.Marshal(prefs.ChannelPreferences)
	if err != nil {
		return fmt.Errorf("failed to marshal channel preferences: %w", err)
	}

	severityChannelsJSON, err := json.Marshal(prefs.SeverityChannels)
	if err != nil {
		return fmt.Errorf("failed to marshal severity channels: %w", err)
	}

	// Update database
	query := `
		UPDATE notification_service.user_preferences
		SET
			channel_preferences = $2,
			severity_channels = $3,
			quiet_hours_enabled = $4,
			quiet_hours_start = $5,
			quiet_hours_end = $6,
			max_alerts_per_hour = $7,
			updated_at = NOW()
		WHERE user_id = $1
	`

	var quietHoursStart, quietHoursEnd interface{}
	if prefs.QuietHoursEnabled {
		quietHoursStart = prefs.QuietHoursStart
		quietHoursEnd = prefs.QuietHoursEnd
	}

	result, err := s.db.Exec(ctx, query,
		userID,
		channelPrefsJSON,
		severityChannelsJSON,
		prefs.QuietHoursEnabled,
		quietHoursStart,
		quietHoursEnd,
		prefs.MaxAlertsPerHour,
	)
	if err != nil {
		return fmt.Errorf("failed to update preferences: %w", err)
	}

	if result.RowsAffected() == 0 {
		return fmt.Errorf("no preferences found to update for user %s", userID)
	}

	// Invalidate cache
	cacheKey := s.cacheKey(cacheKeyPreferencesPrefix, userID)
	if err := s.redisClient.Del(ctx, cacheKey).Err(); err != nil {
		s.logger.Warn("Failed to invalidate preference cache", zap.Error(err), zap.String("user_id", userID))
	}

	s.logger.Info("Updated user preferences", zap.String("user_id", userID))
	return nil
}

// cacheKey generates a cache key with prefix and ID
func (s *UserPreferenceService) cacheKey(prefix, id string) string {
	return prefix + id
}

// getCached retrieves and unmarshals a cached value
func (s *UserPreferenceService) getCached(ctx context.Context, key string, dest interface{}) (bool, error) {
	val, err := s.redisClient.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return false, nil // Cache miss
		}
		return false, fmt.Errorf("redis get error: %w", err)
	}

	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return false, fmt.Errorf("failed to unmarshal cached value: %w", err)
	}

	return true, nil
}

// setCached marshals and caches a value with default TTL
func (s *UserPreferenceService) setCached(ctx context.Context, key string, value interface{}) error {
	return s.setCachedWithTTL(ctx, key, value, s.cacheTTL)
}

// setCachedWithTTL marshals and caches a value with custom TTL
func (s *UserPreferenceService) setCachedWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value for caching: %w", err)
	}

	if err := s.redisClient.Set(ctx, key, data, ttl).Err(); err != nil {
		return fmt.Errorf("redis set error: %w", err)
	}

	return nil
}

// InvalidateUserCache invalidates all cache entries for a user
func (s *UserPreferenceService) InvalidateUserCache(ctx context.Context, userID string) error {
	keys := []string{
		s.cacheKey(cacheKeyPreferencesPrefix, userID),
	}

	for _, key := range keys {
		if err := s.redisClient.Del(ctx, key).Err(); err != nil {
			s.logger.Warn("Failed to invalidate cache", zap.Error(err), zap.String("key", key))
		}
	}

	s.logger.Debug("Invalidated user cache", zap.String("user_id", userID))
	return nil
}

// InvalidateDepartmentCache invalidates cache entries for a department
func (s *UserPreferenceService) InvalidateDepartmentCache(ctx context.Context, departmentID string) error {
	keys := []string{
		s.cacheKey(cacheKeyAttendingPrefix, departmentID),
		s.cacheKey(cacheKeyChargeNursePrefix, departmentID),
		s.cacheKey(cacheKeyResidentPrefix, departmentID),
	}

	for _, key := range keys {
		if err := s.redisClient.Del(ctx, key).Err(); err != nil {
			s.logger.Warn("Failed to invalidate cache", zap.Error(err), zap.String("key", key))
		}
	}

	s.logger.Debug("Invalidated department cache", zap.String("department_id", departmentID))
	return nil
}
