package fatigue

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/cardiofit/notification-service/internal/config"
	"github.com/cardiofit/notification-service/internal/database"
	"github.com/cardiofit/notification-service/internal/models"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
)

const (
	// Redis key prefixes
	rateLimitKeyPrefix = "fatigue:rate"
	duplicateKeyPrefix = "fatigue:dup"
	bundleKeyPrefix    = "fatigue:bundle"

	// Time windows
	rateLimitWindow    = 1 * time.Hour
	duplicateWindow    = 5 * time.Minute
	bundleWindow       = 15 * time.Minute

	// Thresholds
	defaultMaxAlertsPerHour = 20
	bundleThreshold         = 3

	// Suppression reasons
	reasonRateLimit   = "RATE_LIMIT"
	reasonDuplicate   = "DUPLICATE"
	reasonBundled     = "BUNDLED"
	reasonQuietHours  = "QUIET_HOURS"
)

// FatigueConfig holds configuration for alert fatigue management
type FatigueConfig struct {
	MaxAlertsPerHour  int
	DuplicateWindowMs int64
	BundleThreshold   int
	QuietHoursStart   time.Time
	QuietHoursEnd     time.Time
}

// AlertFatigueTracker manages alert fatigue prevention with Redis and PostgreSQL
type AlertFatigueTracker struct {
	redisClient *redis.Client
	db          *pgxpool.Pool
	logger      *zap.Logger
	config      FatigueConfig
}

// SuppressionResult contains details about suppression decision
type SuppressionResult struct {
	ShouldSuppress bool
	Reason         string
	BundledAlerts  []string
	AlertCount     int
}

// BundleInfo tracks bundled alerts
type BundleInfo struct {
	AlertIDs  []string  `json:"alert_ids"`
	Count     int       `json:"count"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
}

// NewAlertFatigueTracker creates a new fatigue tracker
func NewAlertFatigueTracker(
	redisClient *database.RedisClient,
	db *pgxpool.Pool,
	cfg config.FatigueConfig,
	logger *zap.Logger,
) *AlertFatigueTracker {
	// Parse quiet hours
	quietStart, _ := time.Parse("15:04", cfg.QuietHoursStart)
	quietEnd, _ := time.Parse("15:04", cfg.QuietHoursEnd)

	// Convert window duration to milliseconds
	duplicateWindowMs := int64(duplicateWindow / time.Millisecond)

	fatigueConfig := FatigueConfig{
		MaxAlertsPerHour:  cfg.MaxNotifications,
		DuplicateWindowMs: duplicateWindowMs,
		BundleThreshold:   bundleThreshold,
		QuietHoursStart:   quietStart,
		QuietHoursEnd:     quietEnd,
	}

	if fatigueConfig.MaxAlertsPerHour == 0 {
		fatigueConfig.MaxAlertsPerHour = defaultMaxAlertsPerHour
	}

	return &AlertFatigueTracker{
		redisClient: redisClient.Client(),
		db:          db,
		logger:      logger,
		config:      fatigueConfig,
	}
}

// ShouldSuppress determines if an alert should be suppressed for a user
// Returns suppression decision and reason
func (f *AlertFatigueTracker) ShouldSuppress(
	ctx context.Context,
	alert *models.Alert,
	user *models.User,
) (*SuppressionResult, error) {
	// CRITICAL alerts ALWAYS bypass all suppression
	if alert.Severity == models.SeverityCritical {
		f.logger.Debug("CRITICAL alert bypasses suppression",
			zap.String("alert_id", alert.AlertID),
			zap.String("user_id", user.ID),
		)
		return &SuppressionResult{
			ShouldSuppress: false,
			Reason:         "CRITICAL_BYPASS",
		}, nil
	}

	// Check quiet hours (non-CRITICAL alerts only)
	if f.isQuietHours(user) {
		f.logger.Info("Alert suppressed due to quiet hours",
			zap.String("alert_id", alert.AlertID),
			zap.String("user_id", user.ID),
			zap.String("severity", string(alert.Severity)),
		)

		if f.db != nil {
			if err := f.recordSuppressionHistory(ctx, user.ID, alert, reasonQuietHours, ""); err != nil {
				f.logger.Error("Failed to record quiet hours suppression", zap.Error(err))
			}
		}

		return &SuppressionResult{
			ShouldSuppress: true,
			Reason:         reasonQuietHours,
		}, nil
	}

	// Check for duplicate alert
	isDuplicate, originalAlertID, err := f.checkDuplicate(ctx, user.ID, alert)
	if err != nil {
		f.logger.Error("Failed to check duplicate", zap.Error(err))
	} else if isDuplicate {
		f.logger.Info("Alert suppressed as duplicate",
			zap.String("alert_id", alert.AlertID),
			zap.String("original_alert_id", originalAlertID),
			zap.String("user_id", user.ID),
		)

		if f.db != nil {
			if err := f.recordSuppressionHistory(ctx, user.ID, alert, reasonDuplicate, ""); err != nil {
				f.logger.Error("Failed to record duplicate suppression", zap.Error(err))
			}
		}

		return &SuppressionResult{
			ShouldSuppress: true,
			Reason:         reasonDuplicate,
		}, nil
	}

	// Check rate limit
	exceedsLimit, count, err := f.checkRateLimit(ctx, user.ID)
	if err != nil {
		f.logger.Error("Failed to check rate limit", zap.Error(err))
	} else if exceedsLimit {
		f.logger.Warn("Alert suppressed due to rate limit",
			zap.String("alert_id", alert.AlertID),
			zap.String("user_id", user.ID),
			zap.Int("alert_count", count),
			zap.Int("max_per_hour", f.config.MaxAlertsPerHour),
		)

		if f.db != nil {
			if err := f.recordSuppressionHistory(ctx, user.ID, alert, reasonRateLimit, ""); err != nil {
				f.logger.Error("Failed to record rate limit suppression", zap.Error(err))
			}
		}

		return &SuppressionResult{
			ShouldSuppress: true,
			Reason:         reasonRateLimit,
			AlertCount:     count,
		}, nil
	}

	// Check bundling (similar alerts in short window)
	shouldBundle, bundledAlerts, err := f.checkBundling(ctx, user.ID, alert)
	if err != nil {
		f.logger.Error("Failed to check bundling", zap.Error(err))
	} else if shouldBundle {
		f.logger.Info("Alert should be bundled",
			zap.String("alert_id", alert.AlertID),
			zap.String("user_id", user.ID),
			zap.Int("bundle_count", len(bundledAlerts)),
		)

		// Don't suppress, but indicate bundling
		return &SuppressionResult{
			ShouldSuppress: false,
			Reason:         reasonBundled,
			BundledAlerts:  bundledAlerts,
		}, nil
	}

	// No suppression needed
	return &SuppressionResult{
		ShouldSuppress: false,
		Reason:         "ALLOWED",
	}, nil
}

// RecordNotification records that a notification was sent
func (f *AlertFatigueTracker) RecordNotification(
	ctx context.Context,
	userID string,
	alert *models.Alert,
) error {
	now := time.Now()

	// Record in rate limit sorted set
	rateLimitKey := f.getRateLimitKey(userID)
	score := float64(now.UnixMilli())

	if err := f.redisClient.ZAdd(ctx, rateLimitKey, redis.Z{
		Score:  score,
		Member: alert.AlertID,
	}).Err(); err != nil {
		return fmt.Errorf("failed to add to rate limit set: %w", err)
	}

	// Set expiration on rate limit key
	if err := f.redisClient.Expire(ctx, rateLimitKey, rateLimitWindow).Err(); err != nil {
		f.logger.Warn("Failed to set expiration on rate limit key", zap.Error(err))
	}

	// Record as duplicate detection marker
	duplicateKey := f.getDuplicateKey(userID, alert)
	if err := f.redisClient.Set(ctx, duplicateKey, alert.AlertID, duplicateWindow).Err(); err != nil {
		return fmt.Errorf("failed to set duplicate marker: %w", err)
	}

	// Add to bundling list
	bundleKey := f.getBundleKey(userID, alert)
	bundleData, _ := json.Marshal(BundleInfo{
		AlertIDs:  []string{alert.AlertID},
		Count:     1,
		FirstSeen: now,
		LastSeen:  now,
	})

	if err := f.redisClient.LPush(ctx, bundleKey, alert.AlertID).Err(); err != nil {
		f.logger.Warn("Failed to add to bundle list", zap.Error(err))
	}
	if err := f.redisClient.Expire(ctx, bundleKey, bundleWindow).Err(); err != nil {
		f.logger.Warn("Failed to set expiration on bundle key", zap.Error(err))
	}

	// Store bundle metadata
	bundleMetaKey := bundleKey + ":meta"
	if err := f.redisClient.Set(ctx, bundleMetaKey, bundleData, bundleWindow).Err(); err != nil {
		f.logger.Warn("Failed to store bundle metadata", zap.Error(err))
	}

	// Record in PostgreSQL history (skip if DB not configured)
	if f.db != nil {
		if err := f.recordNotificationHistory(ctx, userID, alert); err != nil {
			f.logger.Warn("Failed to record notification history to DB", zap.Error(err))
		}
	}

	return nil
}

// checkRateLimit checks if user has exceeded alert rate limit
func (f *AlertFatigueTracker) checkRateLimit(ctx context.Context, userID string) (bool, int, error) {
	key := f.getRateLimitKey(userID)
	now := time.Now()
	oneHourAgo := now.Add(-rateLimitWindow)

	// Remove old entries
	if err := f.redisClient.ZRemRangeByScore(ctx, key, "0", fmt.Sprintf("%d", oneHourAgo.UnixMilli())).Err(); err != nil {
		f.logger.Warn("Failed to clean old rate limit entries", zap.Error(err))
	}

	// Count alerts in last hour
	count, err := f.redisClient.ZCard(ctx, key).Result()
	if err != nil {
		return false, 0, fmt.Errorf("failed to count rate limit: %w", err)
	}

	maxAlerts := f.config.MaxAlertsPerHour

	// Check user-specific preference if available
	// This could be enhanced to query user preferences from DB

	return int(count) >= maxAlerts, int(count), nil
}

// checkDuplicate checks if alert is a duplicate within the window
func (f *AlertFatigueTracker) checkDuplicate(
	ctx context.Context,
	userID string,
	alert *models.Alert,
) (bool, string, error) {
	key := f.getDuplicateKey(userID, alert)

	originalAlertID, err := f.redisClient.Get(ctx, key).Result()
	if err == redis.Nil {
		// No duplicate found
		return false, "", nil
	} else if err != nil {
		return false, "", fmt.Errorf("failed to check duplicate: %w", err)
	}

	// Found duplicate
	return true, originalAlertID, nil
}

// checkBundling checks if alerts should be bundled
func (f *AlertFatigueTracker) checkBundling(
	ctx context.Context,
	userID string,
	alert *models.Alert,
) (bool, []string, error) {
	key := f.getBundleKey(userID, alert)

	// Get all alerts in bundle window
	alertIDs, err := f.redisClient.LRange(ctx, key, 0, -1).Result()
	if err != nil && err != redis.Nil {
		return false, nil, fmt.Errorf("failed to get bundle list: %w", err)
	}

	// Check if threshold reached
	if len(alertIDs) >= f.config.BundleThreshold {
		return true, alertIDs, nil
	}

	return false, nil, nil
}

// isQuietHours checks if current time is within user's quiet hours
func (f *AlertFatigueTracker) isQuietHours(user *models.User) bool {
	// Check if user has preferences
	if user.Preferences == nil || !user.Preferences.QuietHoursEnabled {
		return false
	}

	now := time.Now()
	currentHour := now.Hour()

	start := user.Preferences.QuietHoursStart
	end := user.Preferences.QuietHoursEnd

	// Handle quiet hours that span midnight
	if start > end {
		return currentHour >= start || currentHour < end
	}

	return currentHour >= start && currentHour < end
}

// Redis key generators
func (f *AlertFatigueTracker) getRateLimitKey(userID string) string {
	return fmt.Sprintf("%s:%s", rateLimitKeyPrefix, userID)
}

func (f *AlertFatigueTracker) getDuplicateKey(userID string, alert *models.Alert) string {
	// Create hash of alert identifying characteristics
	hash := fmt.Sprintf("%s:%s:%s", alert.AlertType, alert.PatientID, alert.Severity)
	hashBytes := sha256.Sum256([]byte(hash))
	hashStr := fmt.Sprintf("%x", hashBytes[:8]) // First 8 bytes

	return fmt.Sprintf("%s:%s:%s", duplicateKeyPrefix, userID, hashStr)
}

func (f *AlertFatigueTracker) getBundleKey(userID string, alert *models.Alert) string {
	return fmt.Sprintf("%s:%s:%s", bundleKeyPrefix, userID, alert.AlertType)
}

// recordNotificationHistory records notification in PostgreSQL
func (f *AlertFatigueTracker) recordNotificationHistory(
	ctx context.Context,
	userID string,
	alert *models.Alert,
) error {
	query := `
		INSERT INTO notification_service.alert_fatigue_history
		(user_id, alert_id, patient_id, alert_type, severity, suppressed, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour) // Keep for 24 hours

	_, err := f.db.Exec(ctx, query,
		userID,
		alert.AlertID,
		alert.PatientID,
		string(alert.AlertType),
		string(alert.Severity),
		false, // Not suppressed since notification was sent
		now,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record notification history: %w", err)
	}

	return nil
}

// recordSuppressionHistory records suppression in PostgreSQL
func (f *AlertFatigueTracker) recordSuppressionHistory(
	ctx context.Context,
	userID string,
	alert *models.Alert,
	reason string,
	bundledWith string,
) error {
	query := `
		INSERT INTO notification_service.alert_fatigue_history
		(user_id, alert_id, patient_id, alert_type, severity, suppressed, suppression_reason, bundled_with, created_at, expires_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	now := time.Now()
	expiresAt := now.Add(24 * time.Hour)

	var bundledWithPtr *string
	if bundledWith != "" {
		bundledWithPtr = &bundledWith
	}

	_, err := f.db.Exec(ctx, query,
		userID,
		alert.AlertID,
		alert.PatientID,
		string(alert.AlertType),
		string(alert.Severity),
		true, // Suppressed
		reason,
		bundledWithPtr,
		now,
		expiresAt,
	)

	if err != nil {
		return fmt.Errorf("failed to record suppression history: %w", err)
	}

	return nil
}

// GetUserAlertStats retrieves alert statistics for a user
func (f *AlertFatigueTracker) GetUserAlertStats(ctx context.Context, userID string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	// Get current rate limit count
	rateLimitKey := f.getRateLimitKey(userID)
	count, err := f.redisClient.ZCard(ctx, rateLimitKey).Result()
	if err != nil && err != redis.Nil {
		return nil, fmt.Errorf("failed to get rate limit count: %w", err)
	}
	stats["alerts_in_last_hour"] = count
	stats["rate_limit_max"] = f.config.MaxAlertsPerHour
	stats["rate_limit_remaining"] = max(0, f.config.MaxAlertsPerHour-int(count))

	// Get suppression stats from DB (last 24 hours) - only if DB is configured
	if f.db != nil {
		query := `
			SELECT
				COUNT(*) FILTER (WHERE suppressed = true) as suppressed_count,
				COUNT(*) FILTER (WHERE suppressed = false) as sent_count,
				COUNT(*) FILTER (WHERE suppression_reason = 'RATE_LIMIT') as rate_limit_count,
				COUNT(*) FILTER (WHERE suppression_reason = 'DUPLICATE') as duplicate_count,
				COUNT(*) FILTER (WHERE suppression_reason = 'QUIET_HOURS') as quiet_hours_count,
				COUNT(*) FILTER (WHERE suppression_reason = 'BUNDLED') as bundled_count
			FROM notification_service.alert_fatigue_history
			WHERE user_id = $1 AND created_at > NOW() - INTERVAL '24 hours'
		`

		var suppressedCount, sentCount, rateLimitCount, duplicateCount, quietHoursCount, bundledCount int64
		err = f.db.QueryRow(ctx, query, userID).Scan(
			&suppressedCount, &sentCount, &rateLimitCount,
			&duplicateCount, &quietHoursCount, &bundledCount,
		)
		if err != nil {
			f.logger.Error("Failed to query suppression stats", zap.Error(err))
		} else {
			stats["suppressed_24h"] = suppressedCount
			stats["sent_24h"] = sentCount
			stats["rate_limit_suppressions_24h"] = rateLimitCount
			stats["duplicate_suppressions_24h"] = duplicateCount
			stats["quiet_hours_suppressions_24h"] = quietHoursCount
			stats["bundled_24h"] = bundledCount
		}
	} else {
		// DB not configured - return only Redis stats
		stats["suppressed_24h"] = "DB not configured"
		stats["sent_24h"] = "DB not configured"
	}

	return stats, nil
}

// CleanupExpiredData removes expired entries from Redis (called periodically)
func (f *AlertFatigueTracker) CleanupExpiredData(ctx context.Context) error {
	// Redis handles TTL automatically, but we clean old sorted set entries
	now := time.Now()
	oneHourAgo := now.Add(-rateLimitWindow)

	// Get all rate limit keys (requires pattern matching - use with caution in production)
	// In production, track user IDs actively and clean only those
	pattern := rateLimitKeyPrefix + ":*"
	iter := f.redisClient.Scan(ctx, 0, pattern, 100).Iterator()

	cleanedCount := 0
	for iter.Next(ctx) {
		key := iter.Val()
		if err := f.redisClient.ZRemRangeByScore(ctx, key, "0", strconv.FormatInt(oneHourAgo.UnixMilli(), 10)).Err(); err != nil {
			f.logger.Warn("Failed to clean rate limit key", zap.String("key", key), zap.Error(err))
		} else {
			cleanedCount++
		}
	}

	if err := iter.Err(); err != nil {
		return fmt.Errorf("failed to scan rate limit keys: %w", err)
	}

	f.logger.Info("Cleaned expired rate limit entries", zap.Int("keys_cleaned", cleanedCount))

	// Cleanup old DB records (older than 7 days) - only if DB configured
	if f.db != nil {
		query := `
			DELETE FROM notification_service.alert_fatigue_history
			WHERE expires_at < NOW()
		`

		result, err := f.db.Exec(ctx, query)
		if err != nil {
			return fmt.Errorf("failed to cleanup DB records: %w", err)
		}

		rowsDeleted := result.RowsAffected()
		f.logger.Info("Cleaned expired DB records", zap.Int64("rows_deleted", rowsDeleted))
	}

	return nil
}

// Helper function for max
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

// ParseTimeFromHHMM parses time string in HH:MM format
func ParseTimeFromHHMM(timeStr string) (time.Time, error) {
	parts := strings.Split(timeStr, ":")
	if len(parts) != 2 {
		return time.Time{}, fmt.Errorf("invalid time format: %s", timeStr)
	}

	hour, err := strconv.Atoi(parts[0])
	if err != nil || hour < 0 || hour > 23 {
		return time.Time{}, fmt.Errorf("invalid hour: %s", parts[0])
	}

	minute, err := strconv.Atoi(parts[1])
	if err != nil || minute < 0 || minute > 59 {
		return time.Time{}, fmt.Errorf("invalid minute: %s", parts[1])
	}

	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, now.Location()), nil
}
